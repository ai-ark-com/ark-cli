// Package catalog exposes the vocabularies the search filters accept.
//
// Small enumerations (seniority levels, company types, ...) are fixed in the
// API and embedded here. The three large vocabularies (industries,
// technologies, departments and functions) are published by AI-Ark as CSV
// files and are downloaded on demand; they carry no authentication because
// they are public.
package catalog

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Entry is one allowed value, with the number of companies using it when the
// source publishes that (technologies only).
type Entry struct {
	Value string
	Count int64 // -1 when unknown
}

// Catalog is a named vocabulary.
type Catalog struct {
	Name        string
	Description string
	// Values holds embedded vocabularies; nil when the catalog is remote.
	Values []string
	// URL is the public CSV for remote vocabularies; empty when embedded.
	URL string
}

// Remote reports whether the values must be downloaded.
func (c Catalog) Remote() bool { return c.URL != "" }

// Catalog names.
const (
	Industries     = "industries"
	Technologies   = "technologies"
	Departments    = "departments"
	Seniority      = "seniority"
	CompanyTypes   = "company-types"
	Languages      = "languages"
	Badges         = "badges"
	SocialMedia    = "social-media"
	FundingTypes   = "funding-types"
	PeopleSources  = "keyword-sources"
	CompanySources = "company-keyword-sources"
	MetricFunction = "metric-functions"
	TimeFrames     = "time-frames"
	MatchModes     = "match-modes"
)

// All lists every catalog in display order.
func All() []Catalog {
	return []Catalog{
		{Name: Industries, Description: "company industries (919 values), --industry", URL: "https://ai-ark.com/static/industries.csv?v=1"},
		{Name: Technologies, Description: "technologies in use (16,000+ values), --technology", URL: "https://ai-ark.com/static/technologies.csv?v=1"},
		{Name: Departments, Description: "departments and job functions (592 values), --department", URL: "https://ai-ark.com/static/departments-and-functions.csv?v=2"},
		{Name: Seniority, Description: "seniority levels, --seniority", Values: SeniorityLevels},
		{Name: CompanyTypes, Description: "company types, --company-type", Values: CompanyTypeValues},
		{Name: Languages, Description: "languages, --language and --company-language", Values: LanguageValues},
		{Name: Badges, Description: "LinkedIn profile badges, --badge", Values: BadgeValues},
		{Name: SocialMedia, Description: "social networks, --social-media", Values: SocialMediaValues},
		{Name: FundingTypes, Description: "funding round types, --funding-type", Values: FundingTypeValues},
		{Name: PeopleSources, Description: "profile sections for --keyword-source (people)", Values: PeopleKeywordSources},
		{Name: CompanySources, Description: "company fields for --keyword-source (companies)", Values: CompanyKeywordSources},
		{Name: MetricFunction, Description: "departments for --headcount and --headcount-growth", Values: MetricFunctions},
		{Name: TimeFrames, Description: "look-back windows for --headcount-growth", Values: TimeFrameValues},
		{Name: MatchModes, Description: "text match modes, --match", Values: MatchModeValues},
	}
}

// Lookup finds a catalog by name.
func Lookup(name string) (Catalog, bool) {
	for _, c := range All() {
		if c.Name == name {
			return c, true
		}
	}
	return Catalog{}, false
}

// Fetcher downloads remote catalogs with a plain, unauthenticated client:
// the CSVs are public and must never see the API token.
type Fetcher struct {
	http *http.Client
	ua   string
}

// NewFetcher returns a Fetcher identifying itself as ark-cli/version.
func NewFetcher(version string) *Fetcher {
	return &Fetcher{http: &http.Client{Timeout: 60 * time.Second}, ua: "ark-cli/" + version}
}

// maxCSV bounds a catalog download; the largest file is well under 1 MB.
const maxCSV = 16 << 20

// Entries returns the values of a catalog, downloading it when needed.
func (f *Fetcher) Entries(ctx context.Context, c Catalog) ([]Entry, error) {
	if !c.Remote() {
		out := make([]Entry, 0, len(c.Values))
		for _, v := range c.Values {
			out = append(out, Entry{Value: v, Count: -1})
		}
		return out, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", f.ua)
	resp, err := f.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading %s catalog: %w", c.Name, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s catalog: HTTP %d", c.Name, resp.StatusCode)
	}
	return parseCSV(io.LimitReader(resp.Body, maxCSV))
}

// parseCSV reads a one- or two-column CSV (value[,count]) with a header row.
func parseCSV(r io.Reader) ([]Entry, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true
	rows, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing catalog CSV: %w", err)
	}
	if len(rows) == 0 {
		return nil, errors.New("catalog CSV is empty")
	}
	out := make([]Entry, 0, len(rows)-1)
	for _, row := range rows[1:] { // skip header
		if len(row) == 0 || strings.TrimSpace(row[0]) == "" {
			continue
		}
		e := Entry{Value: strings.TrimSpace(row[0]), Count: -1}
		if len(row) > 1 {
			if n, err := strconv.ParseInt(strings.TrimSpace(row[1]), 10, 64); err == nil {
				e.Count = n
			}
		}
		out = append(out, e)
	}
	return out, nil
}

// Filter keeps the entries containing q (case-insensitive), preserving order.
// An empty q keeps everything.
func Filter(entries []Entry, q string) []Entry {
	q = strings.ToLower(strings.TrimSpace(q))
	if q == "" {
		return entries
	}
	out := entries[:0:0]
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Value), q) {
			out = append(out, e)
		}
	}
	return out
}

// Normalize maps a user-typed value onto the catalog's canonical spelling,
// ignoring case and treating spaces and dashes as underscores. It returns
// false when the value is not in the catalog.
func Normalize(values []string, in string) (string, bool) {
	key := canonical(in)
	for _, v := range values {
		if canonical(v) == key {
			return v, true
		}
	}
	return "", false
}

func canonical(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	return strings.NewReplacer(" ", "_", "-", "_").Replace(s)
}
