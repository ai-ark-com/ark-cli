package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// isStatus reports whether err is an APIError with the given HTTP status.
func isStatus(err error, status int) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == status
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	c := New(srv.URL, "secret-token", "test")
	c.after = func(time.Duration) <-chan time.Time {
		ch := make(chan time.Time, 1)
		ch <- time.Time{}
		return ch
	}
	return c
}

func TestRequestHeadersAndPath(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != apiPrefix+"/v1/payments/credits" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get(tokenHeader); got != "secret-token" {
			t.Errorf("%s = %q", tokenHeader, got)
		}
		if got := r.Header.Get("User-Agent"); got != "ark-cli/test" {
			t.Errorf("User-Agent = %q", got)
		}
		w.Header().Set(creditHeader, "0")
		_, _ = w.Write([]byte(`{"total":42}`))
	})
	res, err := c.Credits(context.Background())
	if err != nil {
		t.Fatalf("Credits: %v", err)
	}
	if string(res.Body) != `{"total":42}` || res.Credit != "0" {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestRedirectIsRefused(t *testing.T) {
	t.Parallel()

	var leaked atomic.Bool
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/elsewhere" {
			leaked.Store(true)
			return
		}
		http.Redirect(w, r, "/elsewhere", http.StatusFound)
	})
	_, err := c.Credits(context.Background())
	if !errors.Is(err, ErrRedirect) {
		t.Fatalf("err = %v, want ErrRedirect", err)
	}
	if leaked.Load() {
		t.Fatal("client followed the redirect and re-sent the token")
	}
}

func TestAPIErrorEnvelope(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"timestamp":"2025-10-14T13:27:32Z","status":404,"error":"data not found","path":""}`))
	})
	_, err := c.ReverseLookup(context.Background(), "nobody@example.com")
	if !isStatus(err, http.StatusNotFound) {
		t.Fatalf("err = %v, want 404 APIError", err)
	}
	if got, want := err.Error(), "API error (HTTP 404): data not found"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestAPIErrorStripsControlCharacters(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad\x1b[31m gateway\r\nline two"))
	})
	_, err := c.Credits(context.Background())
	if err == nil || err.Error() != "API error (HTTP 502): bad[31m gateway  line two" {
		t.Errorf("Error() = %v", err)
	}
}

func TestRateLimitIsRetried(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"total":1}`))
	})
	if _, err := c.Credits(context.Background()); err != nil {
		t.Fatalf("Credits: %v", err)
	}
	if calls.Load() != 3 {
		t.Errorf("calls = %d, want 3", calls.Load())
	}
}

func TestRateLimitGivesUp(t *testing.T) {
	t.Parallel()

	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	_, err := c.Credits(context.Background())
	if !isStatus(err, http.StatusTooManyRequests) {
		t.Fatalf("err = %v, want 429 APIError", err)
	}
}

func TestRetryDelay(t *testing.T) {
	t.Parallel()

	tests := map[string]time.Duration{
		"":    time.Second,
		"abc": time.Second,
		"-3":  time.Second,
		"2":   2 * time.Second,
		"999": 10 * time.Second,
	}
	for in, want := range tests {
		if got := retryDelay(in); got != want {
			t.Errorf("retryDelay(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestJobPathsAndQuery(t *testing.T) {
	t.Parallel()

	const id = "719aba5a-876f-4690-bb57-5157153836b4"
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET " + apiPrefix + "/v1/people/export/" + id + "/inquiries":
			if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("size") != "50" {
				t.Errorf("query = %q", r.URL.RawQuery)
			}
		case "GET " + apiPrefix + "/v1/people/email-finder/submissions":
			q := r.URL.Query()
			if q.Get("state") != "SETTLED" || q.Get("fullyRefunded") != "true" || len(q["sort"]) != 2 {
				t.Errorf("query = %q", r.URL.RawQuery)
			}
		case "PATCH " + apiPrefix + "/v1/people/email-finder/" + id + "/notify":
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{}`))
	})
	ctx := context.Background()
	if _, err := c.JobResults(ctx, JobExport, id, 2, 50); err != nil {
		t.Fatal(err)
	}
	refunded := true
	if _, err := c.JobSubmissions(ctx, JobEmailFinder, &SubmissionsQuery{
		State: "SETTLED", FullyRefunded: &refunded, Size: 25, Sort: []string{"created,desc", "state,asc"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.ResendWebhook(ctx, JobEmailFinder, id, "https://example.com/hook"); err != nil {
		t.Fatal(err)
	}
}

func TestValidation(t *testing.T) {
	t.Parallel()

	c := New("https://unused.invalid", "t", "test")
	ctx := context.Background()
	if _, err := c.JobStatistics(ctx, JobExport, "../../etc/passwd"); err == nil {
		t.Error("malformed track id must be rejected before any request")
	}
	if _, err := c.FindEmails(ctx, "719aba5a-876f-4690-bb57-5157153836b4", "http://insecure.example.com"); err == nil {
		t.Error("plain-http webhook must be rejected")
	}
	if _, err := c.SaveList(ctx, &ListRequest{ID: "not-a-uuid", Values: []string{"x"}}); err == nil {
		t.Error("malformed list id must be rejected")
	}
}
