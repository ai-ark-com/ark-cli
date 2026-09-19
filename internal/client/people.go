package client

import "context"

// ExportSingleRequest identifies one person to enrich, by AI-Ark id or by
// LinkedIn profile URL.
type ExportSingleRequest struct {
	ID  string `json:"id,omitempty"`
	URL string `json:"url,omitempty"`
}

// ExportSingle returns the full profile plus a verified email for one person
// (1 credit when an email is found, 0 otherwise). With clay set, the v2
// endpoint is used: it answers HTTP 200 with data: null instead of 404 when
// nothing is found, which row-by-row enrichment tools expect.
func (c *Client) ExportSingle(ctx context.Context, req *ExportSingleRequest, clay bool) (*Result, error) {
	path := "/v1/people/export/single"
	if clay {
		path = "/v2/people/export/single"
	}
	return c.post(ctx, path, req)
}

// ReverseLookupRequest identifies a person from an email address or phone
// number. Kind is the lookup discriminator ("CONTACT" is the only value).
type ReverseLookupRequest struct {
	Kind   string `json:"kind"`
	Search string `json:"search"`
}

// ReverseLookup returns the profile behind an email address or phone
// number (0.5 credits).
func (c *Client) ReverseLookup(ctx context.Context, contact string) (*Result, error) {
	return c.post(ctx, "/v1/people/reverse-lookup", &ReverseLookupRequest{Kind: "CONTACT", Search: contact})
}

// PhoneFinderRequest identifies a person either by LinkedIn URL or by
// company domain together with full name. Type is the number-type
// discriminator ("MOBILE" is the only value the API defines).
type PhoneFinderRequest struct {
	Type     string `json:"type"`
	LinkedIn string `json:"linkedin,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Name     string `json:"name,omitempty"`
}

// PhoneFinder finds a mobile number (5 credits when found). With clay set,
// the v2 endpoint answers 200 with data: null instead of 404 on a miss.
func (c *Client) PhoneFinder(ctx context.Context, req *PhoneFinderRequest, clay bool) (*Result, error) {
	path := "/v1/people/mobile-phone-finder"
	if clay {
		path = "/v2/people/mobile-phone-finder"
	}
	req.Type = "MOBILE"
	return c.post(ctx, path, req)
}

// PersonalityRequest names the LinkedIn profile to analyse.
type PersonalityRequest struct {
	URL string `json:"url"`
}

// Personality returns a DISC/OCEAN profile with outreach guidance (4 credits).
func (c *Client) Personality(ctx context.Context, linkedinURL string) (*Result, error) {
	return c.post(ctx, "/v1/people/analysis", &PersonalityRequest{URL: linkedinURL})
}
