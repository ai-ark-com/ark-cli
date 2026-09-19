package client

import (
	"bytes"
	"context"
	"encoding/json"
)

// AccountFilter holds company-level filters. It is accepted by company
// search and, to constrain a person's employer, by every people endpoint.
type AccountFilter struct {
	Domain             *PlainFilter    `json:"domain,omitempty"`
	LinkedIn           *PlainFilter    `json:"linkedin,omitempty"`
	URL                *MatchFilter    `json:"url,omitempty"`
	Name               *MatchFilter    `json:"name,omitempty"`
	SocialMediaLink    *PlainFilter    `json:"socialMediaLink,omitempty"`
	PhoneNumber        *PlainFilter    `json:"phoneNumber,omitempty"`
	Industries         *MatchFilter    `json:"industries,omitempty"`
	Location           *PlainFilter    `json:"location,omitempty"`
	ProductAndServices *MatchFilter    `json:"productAndServices,omitempty"`
	SocialMedia        *PlainFilter    `json:"socialMedia,omitempty"`
	Type               *PlainFilter    `json:"type,omitempty"`
	FoundedYear        *YearFilter     `json:"foundedYear,omitempty"`
	EmployeeSize       *RangeFilter    `json:"employeeSize,omitempty"`
	RetailSize         *RangeFilter    `json:"retailSize,omitempty"`
	Revenue            *RangeFilter    `json:"revenue,omitempty"`
	Language           *LanguageFilter `json:"language,omitempty"`
	GeoLocation        *GeoLocation    `json:"geoLocation,omitempty"`
	Keyword            *KeywordFilter  `json:"keyword,omitempty"`
	Funding            *Funding        `json:"funding,omitempty"`
	Metric             *Metrics        `json:"metric,omitempty"`
	Technologies       *MatchFilter    `json:"technologies,omitempty"`
	// Technology is the original plain-value technology filter, kept by the
	// API for backward compatibility; Technologies supports match modes.
	Technology *PlainFilter `json:"technology,omitempty"`
	NAICS      *PlainFilter `json:"naics,omitempty"`
	// Employee finds companies that employ people matching a job role. It is
	// accepted by company search only.
	Employee *EmployeeFilter `json:"employee,omitempty"`
}

// LanguageFilter is the account-level operating-language filter: plain
// values plus an optional band on how many languages a company operates in.
type LanguageFilter struct {
	Any   *Values `json:"any,omitempty"`
	All   *Values `json:"all,omitempty"`
	Range *Range  `json:"range,omitempty"`
}

// GeoLocation is a radius search around a coordinate.
type GeoLocation struct {
	Position Coordinate `json:"position"`
	Radius   float64    `json:"radius"`
	Unit     string     `json:"unit"` // "km" or "mi"
}

// Coordinate is a WGS84 point.
type Coordinate struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Funding filters by funding history.
type Funding struct {
	Type        []string   `json:"type,omitempty"`
	TotalAmount *Range     `json:"totalAmount,omitempty"`
	LastAmount  *Range     `json:"lastAmount,omitempty"`
	Duration    *TimeRange `json:"duration,omitempty"`
}

// TimeRange is a band of epoch milliseconds, sent as strings as the API expects.
type TimeRange struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

// Metrics filters by per-department headcount and headcount growth.
type Metrics struct {
	Employee []DepartmentHeadcount `json:"employee,omitempty"`
	Growth   []DepartmentGrowth    `json:"growth,omitempty"`
}

// DepartmentHeadcount is an absolute headcount band for the given departments.
type DepartmentHeadcount struct {
	Function []string `json:"function"`
	Start    *float64 `json:"start,omitempty"`
	End      *float64 `json:"end,omitempty"`
}

// DepartmentGrowth is a percentage growth band over a look-back window
// (TimeFrame is a number of months: ONE, THREE, SIX, TWELVE, TWENTY_FOUR).
type DepartmentGrowth struct {
	Function  []string `json:"function"`
	Start     *float64 `json:"start,omitempty"`
	End       *float64 `json:"end,omitempty"`
	TimeFrame string   `json:"timeFrame"`
}

// EmployeeFilter targets companies by the roles of the people they employ.
type EmployeeFilter struct {
	Title                 *MatchFilter `json:"title,omitempty"`
	Seniority             *PlainFilter `json:"seniority,omitempty"`
	DepartmentAndFunction *PlainFilter `json:"departmentAndFunction,omitempty"`
}

// ContactFilter holds person-level filters.
type ContactFilter struct {
	FullName              *MatchFilter           `json:"fullName,omitempty"`
	SocialMediaLink       *PlainFilter           `json:"socialMediaLink,omitempty"`
	Company               *ScopedIDs             `json:"company,omitempty"`
	Experience            *Experience            `json:"experience,omitempty"`
	Seniority             *PlainFilter           `json:"seniority,omitempty"`
	Location              *PlainFilter           `json:"location,omitempty"`
	LinkedIn              *PlainFilter           `json:"linkedin,omitempty"`
	DepartmentAndFunction *PlainFilter           `json:"departmentAndFunction,omitempty"`
	Skill                 *MatchFilter           `json:"skill,omitempty"`
	Certification         *MatchFilter           `json:"certification,omitempty"`
	Keyword               *KeywordFilter         `json:"keyword,omitempty"`
	SocialMedia           *PlainFilter           `json:"socialMedia,omitempty"`
	Language              *MatchedLanguageFilter `json:"language,omitempty"`
	Education             *Education             `json:"education,omitempty"`
	ProfileBadge          *PlainFilter           `json:"profileBadge,omitempty"`
	SocialMediaFollower   *SocialMediaFollower   `json:"socialMediaFollower,omitempty"`
}

// ScopedIDs filters by AI-Ark company IDs across employment scopes.
type ScopedIDs struct {
	Latest   *PlainFilter `json:"latest,omitempty"`   // company of the primary active role
	Current  *PlainFilter `json:"current,omitempty"`  // any active role
	Previous *PlainFilter `json:"previous,omitempty"` // past employers
}

// Experience filters by job title (and, for current roles, tenure).
type Experience struct {
	Latest   *Position        `json:"latest,omitempty"`
	Current  *CurrentPosition `json:"current,omitempty"`
	Previous *Position        `json:"previous,omitempty"`
}

// Position is a job-title filter for one employment scope.
type Position struct {
	Title *MatchFilter `json:"title,omitempty"`
}

// CurrentPosition adds tenure bands to the title filter.
type CurrentPosition struct {
	Title    *MatchFilter `json:"title,omitempty"`
	Duration *Duration    `json:"duration,omitempty"`
}

// Duration holds min/max tenure bands.
type Duration struct {
	CurrentCompany *YearMonthBand `json:"currentCompany,omitempty"`
	CurrentJob     *YearMonthBand `json:"currentJob,omitempty"`
	Total          *YearMonthBand `json:"total,omitempty"`
}

// YearMonthBand is a tenure band expressed in years and months.
type YearMonthBand struct {
	Min *YearMonth `json:"min,omitempty"`
	Max *YearMonth `json:"max,omitempty"`
}

// YearMonth is a duration in whole years and months.
type YearMonth struct {
	Year  int `json:"year"`
	Month int `json:"month"`
}

// MatchedLanguageFilter is the contact-level language filter: matched terms
// plus an optional band on how many languages the person speaks.
type MatchedLanguageFilter struct {
	Any   *MatchValues `json:"any,omitempty"`
	All   *MatchValues `json:"all,omitempty"`
	Range *Range       `json:"range,omitempty"`
}

// Education filters by education history.
type Education struct {
	School       *PlainFilter   `json:"school,omitempty"` // AI-Ark company IDs of institutions
	Degree       *MatchFilter   `json:"degree,omitempty"`
	FieldOfStudy *MatchFilter   `json:"fieldOfStudy,omitempty"`
	Date         *EducationDate `json:"date,omitempty"`
}

// EducationDate is the study period; Present selects people still studying.
type EducationDate struct {
	Start   *float64 `json:"start,omitempty"`
	End     *float64 `json:"end,omitempty"`
	Present *bool    `json:"present,omitempty"`
}

// SocialMediaFollower filters by follower and connection counts. Only
// LinkedIn is supported by the API.
type SocialMediaFollower struct {
	LinkedIn *FollowerCounts `json:"linkedin,omitempty"`
}

// FollowerCounts holds independent follower and connection bands.
type FollowerCounts struct {
	Followers   *RangeFilter `json:"followers,omitempty"`
	Connections *RangeFilter `json:"connections,omitempty"`
}

// SearchRequest is the body shared by people search, preview, export, and
// company search. Fields that only one endpoint accepts (LookalikeDomains,
// Webhook) are omitted when empty.
type SearchRequest struct {
	LookalikeDomains []string       `json:"lookalikeDomains,omitempty"`
	Account          *AccountFilter `json:"account,omitempty"`
	Contact          *ContactFilter `json:"contact,omitempty"`
	Lists            *ListsFilter   `json:"lists,omitempty"`
	Page             int            `json:"page"`
	Size             int            `json:"size"`
	Webhook          string         `json:"webhook,omitempty"`
}

// HasFilters reports whether the request narrows the result set at all. An
// unfiltered search would match everything and bill the caller for nothing
// useful, so commands refuse to send one.
func (r *SearchRequest) HasFilters() bool {
	if len(r.LookalikeDomains) > 0 {
		return true
	}
	return !IsEmptyJSON(r.Account) || !IsEmptyJSON(r.Contact)
}

// Compact drops filter objects that carry no filters, so the wire body only
// contains the sections the caller actually set.
func (r *SearchRequest) Compact() *SearchRequest {
	if IsEmptyJSON(r.Account) {
		r.Account = nil
	}
	if IsEmptyJSON(r.Contact) {
		r.Contact = nil
	}
	if IsEmptyJSON(r.Lists) {
		r.Lists = nil
	}
	return r
}

// IsEmptyJSON reports whether v serialises to null, "{}", or "[]". Checking
// the wire form means a new filter field can never be forgotten in the check.
// A json.RawMessage is compared as-is (after whitespace removal).
func IsEmptyJSON(v any) bool {
	if v == nil {
		return true
	}
	var b []byte
	if raw, ok := v.(json.RawMessage); ok {
		var buf bytes.Buffer
		if json.Compact(&buf, raw) != nil {
			return true
		}
		b = buf.Bytes()
	} else {
		var err error
		if b, err = json.Marshal(v); err != nil {
			return true
		}
	}
	switch string(b) {
	case "", "null", "{}", "[]":
		return true
	}
	return false
}

// Search endpoints. Every search body goes through Search with one of these
// paths; the CLI builds the JSON itself (typed, or verbatim from --body).
const (
	PathPeopleSearch  = "/v1/people"         // 0.5 credits per result
	PathPeoplePreview = "/v1/people/preview" // 1 credit per page
	PathPeopleExport  = "/v1/people/export"  // async; 0.5 per person + 0.5 per found email
	PathCompanySearch = "/v1/companies"      // 0.1 credits per result
)

// Search posts a search body unchanged to one of the Path* endpoints.
func (c *Client) Search(ctx context.Context, path string, body json.RawMessage) (*Result, error) {
	return c.post(ctx, path, body)
}
