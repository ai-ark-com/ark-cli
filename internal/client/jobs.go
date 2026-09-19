package client

import (
	"context"
	"net/url"
	"strconv"
)

// JobKind selects one of the two asynchronous job families. Both expose the
// same statistics / results / submissions / notify sub-resources.
type JobKind string

// Asynchronous job families.
const (
	JobExport      JobKind = "export"       // Export People with Email
	JobEmailFinder JobKind = "email-finder" // Find Emails by Track ID
)

func (k JobKind) base() string { return "/v1/people/" + string(k) }

// EmailFinderRequest starts email finding for the result set of a previous
// people search. The track id is single-use and expires six hours after the
// search that produced it.
type EmailFinderRequest struct {
	TrackID string `json:"trackId"`
	Webhook string `json:"webhook"`
}

// FindEmails submits an email-finding job (1 credit per found email).
func (c *Client) FindEmails(ctx context.Context, trackID, webhook string) (*Result, error) {
	if err := ValidateID("track id", trackID); err != nil {
		return nil, err
	}
	if err := ValidateWebhook(webhook); err != nil {
		return nil, err
	}
	return c.post(ctx, JobEmailFinder.base(), &EmailFinderRequest{TrackID: trackID, Webhook: webhook})
}

// JobStatistics returns the state and counters of a job. Free.
func (c *Client) JobStatistics(ctx context.Context, kind JobKind, trackID string) (*Result, error) {
	if err := ValidateID("track id", trackID); err != nil {
		return nil, err
	}
	return c.get(ctx, kind.base()+"/"+url.PathEscape(trackID)+"/statistics", nil)
}

// JobResults returns one page of a job's results. Free. Export jobs answer
// 409 until the job is done; email-finder jobs can be read while running.
func (c *Client) JobResults(ctx context.Context, kind JobKind, trackID string, page, size int) (*Result, error) {
	if err := ValidateID("track id", trackID); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("size", strconv.Itoa(size))
	return c.get(ctx, kind.base()+"/"+url.PathEscape(trackID)+"/inquiries", q)
}

// SubmissionsQuery filters and pages the caller's submission history.
type SubmissionsQuery struct {
	State         string // "PENDING" or "SETTLED"; empty for all
	FullyRefunded *bool  // nil for all
	Page          int
	Size          int
	Sort          []string // "property,asc|desc", repeatable
}

// JobSubmissions lists the caller's own submissions of the given kind. Free.
func (c *Client) JobSubmissions(ctx context.Context, kind JobKind, sq *SubmissionsQuery) (*Result, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(sq.Page))
	q.Set("size", strconv.Itoa(sq.Size))
	if sq.State != "" {
		q.Set("state", sq.State)
	}
	if sq.FullyRefunded != nil {
		q.Set("fullyRefunded", strconv.FormatBool(*sq.FullyRefunded))
	}
	for _, s := range sq.Sort {
		q.Add("sort", s)
	}
	return c.get(ctx, kind.base()+"/submissions", q)
}

// WebhookRequest names the URL to (re)notify.
type WebhookRequest struct {
	Webhook string `json:"webhook"`
}

// ResendWebhook re-delivers a job's completion webhook to the given URL. Free.
func (c *Client) ResendWebhook(ctx context.Context, kind JobKind, trackID, webhook string) (*Result, error) {
	if err := ValidateID("track id", trackID); err != nil {
		return nil, err
	}
	if err := ValidateWebhook(webhook); err != nil {
		return nil, err
	}
	return c.patch(ctx, kind.base()+"/"+url.PathEscape(trackID)+"/notify", &WebhookRequest{Webhook: webhook})
}
