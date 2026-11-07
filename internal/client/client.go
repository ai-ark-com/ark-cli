// Package client is the HTTP layer for the AI-Ark developer-portal API.
//
// It authenticates with the X-TOKEN header, sends and receives JSON, surfaces
// the per-call credit cost from the X-Credit response header, and turns
// non-2xx responses into typed errors. It knows nothing about billing: the
// server meters every call by token.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ai-ark-com/ark-cli/internal/text"
)

const (
	apiPrefix     = "/api/developer-portal"
	tokenHeader   = "X-TOKEN"
	creditHeader  = "X-Credit"
	userAgentBase = "ark-cli"

	requestTimeout = 120 * time.Second
	// maxErrorBody caps how much of an error response is kept, so a stray
	// HTML page from a proxy cannot flood the terminal.
	maxErrorBody = 4 << 10
	// maxResponseBody caps a successful response; the largest legitimate
	// payload (100 full profiles) is far below this.
	maxResponseBody = 64 << 20
	// retryAttempts is how many times a rate-limited (429) call is retried.
	retryAttempts = 3
	// retryCeiling is the longest wait the client sleeps through on its own.
	// A longer one (a full minute, hour or week window) fails at once and
	// the error says when to retry.
	retryCeiling = 10 * time.Second
)

// ErrRedirect is returned when the API answers with a redirect. Redirects
// are never followed: Go's HTTP client would forward the X-TOKEN header to
// the new location, which could leak the token to a third-party host.
var ErrRedirect = errors.New("server answered with a redirect; refusing to follow it with credentials")

// Client talks to one AI-Ark host with one token.
type Client struct {
	http    *http.Client
	baseURL string
	token   string
	ua      string
	after   func(time.Duration) <-chan time.Time // time.After; replaced in tests
}

// New returns a Client. baseURL must not end in a slash.
func New(baseURL, token, version string) *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	return &Client{
		http: &http.Client{
			Timeout:   requestTimeout,
			Transport: transport,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		baseURL: baseURL,
		token:   token,
		ua:      userAgentBase + "/" + version,
		after:   time.After,
	}
}

// BaseURL returns the host the client talks to.
func (c *Client) BaseURL() string { return c.baseURL }

// Result carries the decoded body, the credits the call consumed and where
// the request limits of the plan stand after it.
type Result struct {
	Body   json.RawMessage
	Credit string // value of X-Credit, e.g. "-0.5"; empty if the call was not charged
	Limits []Limit
}

// APIError is a non-2xx response from the API.
type APIError struct {
	Status  int
	Message string // parsed from the {"error"|"message"} envelope, or the raw body

	// Set on a rate-limited (429) response.
	Limits  []Limit
	RetryAt time.Time // when a request is admitted again; zero if the gateway did not say
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("API error: HTTP %d %s", e.Status, http.StatusText(e.Status))
	}
	return fmt.Sprintf("API error (HTTP %d): %s", e.Status, e.Message) + e.rateLimitDetail()
}

// rateLimitDetail names the window that is used up and when to retry, e.g.
// ". All 5 requests per minute are used. Retry in 45s (at 15:21:04 UTC)".
func (e *APIError) rateLimitDetail() string {
	var detail string
	for _, l := range e.Limits {
		if l.UsedUp() {
			detail = fmt.Sprintf(". All %d requests per %s are used", l.Max, l.window())
		}
	}
	if !e.RetryAt.IsZero() {
		detail += fmt.Sprintf(". Retry in %s (at %s)", formatWait(time.Until(e.RetryAt)), e.RetryAt.UTC().Format("15:04:05 MST"))
	}
	return detail
}

func newAPIError(status int, body []byte) *APIError {
	if len(body) > maxErrorBody {
		body = body[:maxErrorBody]
	}
	var env struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	msg := strings.TrimSpace(string(body))
	if json.Unmarshal(body, &env) == nil {
		switch {
		case env.Message != "":
			msg = env.Message
		case env.Error != "":
			msg = env.Error
		}
	}
	return &APIError{Status: status, Message: text.Sanitize(msg)}
}

func (c *Client) post(ctx context.Context, path string, body any) (*Result, error) {
	return c.do(ctx, http.MethodPost, path, nil, body)
}

func (c *Client) patch(ctx context.Context, path string, body any) (*Result, error) {
	return c.do(ctx, http.MethodPatch, path, nil, body)
}

func (c *Client) get(ctx context.Context, path string, query url.Values) (*Result, error) {
	return c.do(ctx, http.MethodGet, path, query, nil)
}

func (c *Client) do(ctx context.Context, method, path string, query url.Values, body any) (*Result, error) {
	var payload []byte
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			return nil, fmt.Errorf("encoding request: %w", err)
		}
	}
	target := c.baseURL + apiPrefix + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	for attempt := 0; ; attempt++ {
		res, retryAfter, err := c.once(ctx, method, target, payload)
		if err == nil || attempt >= retryAttempts || retryAfter < 0 {
			return res, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-c.after(retryAfter):
		}
	}
}

// once performs a single HTTP exchange. On a 429 it returns the delay to
// wait before retrying; any other outcome returns retryAfter < 0.
func (c *Client) once(ctx context.Context, method, target string, payload []byte) (*Result, time.Duration, error) {
	var reader io.Reader = http.NoBody
	if payload != nil {
		reader = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return nil, -1, err
	}
	req.Header.Set(tokenHeader, c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.ua)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, -1, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	if err != nil {
		return nil, -1, fmt.Errorf("reading response: %w", err)
	}

	switch {
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		return nil, -1, ErrRedirect
	case resp.StatusCode == http.StatusTooManyRequests:
		wait := retryAfter(resp.Header.Get("Retry-After"))
		apiErr := newAPIError(resp.StatusCode, raw)
		apiErr.Limits = limitsFrom(resp.Header)
		if wait > 0 {
			apiErr.RetryAt = time.Now().Add(wait)
		}
		return nil, retryDelay(wait), apiErr
	case resp.StatusCode < 200 || resp.StatusCode >= 300:
		return nil, -1, newAPIError(resp.StatusCode, raw)
	}
	return &Result{Body: raw, Credit: resp.Header.Get(creditHeader), Limits: limitsFrom(resp.Header)}, -1, nil
}

// retryAfter reads a Retry-After header (seconds); zero when the gateway
// sent none or sent something else.
func retryAfter(header string) time.Duration {
	secs, err := strconv.Atoi(strings.TrimSpace(header))
	if err != nil || secs <= 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
}

// retryDelay decides how long to sleep before retrying a 429: the wait the
// gateway asked for, one second when it asked for none, and no retry at all
// (a negative delay) when the wait is past retryCeiling, since a retry before
// then is rejected again.
func retryDelay(wait time.Duration) time.Duration {
	switch {
	case wait <= 0:
		return time.Second
	case wait > retryCeiling:
		return -1
	}
	return wait
}

// Credits reads the current credit balance for the token.
func (c *Client) Credits(ctx context.Context) (*Result, error) {
	return c.get(ctx, "/v1/payments/credits", nil)
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ValidateID checks that id looks like the UUIDs the API hands out. It runs
// before an id is placed in a URL path, so a malformed value fails fast
// locally instead of producing a confusing 404. what names the id in the
// error, e.g. "track id" or "--company-id".
func ValidateID(what, id string) error {
	if !uuidPattern.MatchString(id) {
		return fmt.Errorf("%s: %q is not a valid UUID", what, id)
	}
	return nil
}

// ValidateWebhook enforces the API's requirement that webhooks are public
// HTTPS URLs.
func ValidateWebhook(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("webhook must be an https:// URL, got %q", raw)
	}
	return nil
}
