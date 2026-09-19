package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/ai-ark-com/ark-cli/internal/catalog"
	"github.com/ai-ark-com/ark-cli/internal/client"
)

// filterScope selects which filter groups a search command exposes: people
// commands take contact filters plus company-level ("company-*") filters;
// company search takes company-level filters under their short names.
type filterScope int

const (
	peopleScope filterScope = iota
	companyScope
)

// facetKind is the wire shape of a list-valued facet.
type facetKind int

const (
	plainFacet   facetKind = iota // {"any":{"include":[...]}}
	matchedFacet                  // {"any":{"include":{"mode","content"}}}
	keywordFacet                  // {"any":{"include":{"sources","content"}}}
)

// facet describes one list-valued filter of the API. Every facet is exposed
// as four flags: --<name> and --exclude-<name> (any: OR across values), and
// --all-<name> and --all-exclude-<name> (all: AND across values).
type facet struct {
	name   string
	usage  string
	kind   facetKind
	enum   []string // allowed values (normalised); nil for free text
	ids    bool     // values must be AI-Ark UUIDs
	sample string   // a valid example value; tests use it to exercise every path

	// keyword facets: the companion --<name>-source flag, its catalog, the
	// default sources, and the API's cap on how many may be given.
	sources        []string
	sourceCatalog  string
	defaultSources []string
	maxSources     int

	// Exactly one of these stores the built filter on the request.
	plain   func(*client.SearchRequest, *client.PlainFilter)
	matched func(*client.SearchRequest, *client.MatchFilter)
	keyword func(*client.SearchRequest, *client.KeywordFilter)
}

// facetValues holds the four flag lists of one facet.
type facetValues struct {
	include, exclude, allInclude, allExclude []string
}

// searchFlags collects every filter flag of a search command and turns them
// into a client.SearchRequest. Facets are table-driven (see facetsFor);
// the remaining fields are the scalar and range filters.
type searchFlags struct {
	scope filterScope
	names []string // registered filter flag names, to detect clashes with --body

	match  string
	body   string
	dryRun bool

	facets []*facet
	values map[string]*facetValues

	// company-level scalars (both scopes)
	founded              string
	employees            []string
	retailLocations      []string
	revenue              []string
	companyLanguageCount string
	near                 string
	radius               float64
	radiusUnit           string
	fundingType          []string
	fundingTotal         string
	fundingLast          string
	fundingDate          string
	headcount            []string
	headcountGrowth      []string

	// person-level scalars (people scope)
	languageCount string
	educationDate string
	studying      bool
	followers     []string
	connections   []string
	tenureCompany string
	tenureJob     string
	tenureTotal   string

	lookalike   []string // company scope only
	excludeList []string
}

// searchPlan is the outcome of parsing the flags: either a typed request or
// a raw body from --body.
type searchPlan struct {
	req *client.SearchRequest
	raw json.RawMessage
}

// register adds the filter flags to fs.
func (s *searchFlags) register(fs *pflag.FlagSet, scope filterScope) {
	s.scope = scope
	s.values = map[string]*facetValues{}
	s.facets = facetsFor(scope)

	track := func(names ...string) { s.names = append(s.names, names...) }
	slice := func(p *[]string, name, usage string) {
		track(name)
		fs.StringSliceVar(p, name, nil, usage)
	}
	str := func(p *string, name, usage string) {
		track(name)
		fs.StringVar(p, name, "", usage)
	}

	fs.StringVar(&s.match, "match", "smart", "text match mode for --title, --skill, --industry, ...: smart, word, or strict")
	fs.StringVar(&s.body, "body", "", "send this JSON request body instead of building one from flags (\"-\" for stdin)")
	fs.BoolVar(&s.dryRun, "dry-run", false, "print the request body and exit without calling the API")
	track("match")

	for _, f := range s.facets {
		v := &facetValues{}
		s.values[f.name] = v
		slice(&v.include, f.name, f.usage)
		slice(&v.exclude, "exclude-"+f.name, "exclude "+f.usage)
		slice(&v.allInclude, "all-"+f.name, "require every value of "+f.usage)
		slice(&v.allExclude, "all-exclude-"+f.name, "exclude only when every value matches; "+f.usage)
		// The all-* forms are described once in filterHelp; hiding them
		// keeps the flag list readable.
		_ = fs.MarkHidden("all-" + f.name)
		_ = fs.MarkHidden("all-exclude-" + f.name)
		if f.kind == keywordFacet {
			slice(&f.sources, f.name+"-source", "sections searched by --"+f.name+" (default "+
				joinValues(f.defaultSources)+"), see 'ark catalog "+f.sourceCatalog+"'")
		}
	}

	if scope == peopleScope {
		str(&s.languageCount, "language-count", "number of languages spoken, as a band like 3+ or 2-4")
		str(&s.educationDate, "education-date", "study period as a year band like 2015-2025, 2018+ or -2010")
		fs.BoolVar(&s.studying, "studying", false, "only people currently studying (no education end date)")
		track("studying")
		slice(&s.followers, "followers", "LinkedIn follower band(s) like 1k-5k or 30k+ (repeat to widen)")
		slice(&s.connections, "connections", "LinkedIn connection band(s) like 500+ (repeat to widen)")
		str(&s.tenureCompany, "tenure-company", "time at the current company, as a band like 2y-5y, 1y6m+ or -3y")
		str(&s.tenureJob, "tenure-job", "time in the current job, same format as --tenure-company")
		str(&s.tenureTotal, "tenure-total", "total career length, same format as --tenure-company")
	}

	acct := accountFlagName(scope)
	str(&s.founded, "founded", "founded-year band like 2015-2022, 2020+ or -1990; or any / none")
	slice(&s.employees, "employees", "headcount band(s) like 51-200, 1000+ or -50 (repeat to widen)")
	slice(&s.retailLocations, "retail-locations", "number of physical locations, band(s) like 10-100")
	slice(&s.revenue, "revenue", "annual revenue band(s) in USD like 1m-10m, 100m+; or any / none")
	str(&s.companyLanguageCount, acct("language-count"), "number of languages the company operates in, as a band like 2+")
	str(&s.near, "near", "centre of a radius search as lat,lng (use with --radius)")
	fs.Float64Var(&s.radius, "radius", 50, "radius for --near")
	fs.StringVar(&s.radiusUnit, "radius-unit", "km", "unit for --radius: km or mi")
	track("radius", "radius-unit")
	slice(&s.fundingType, "funding-type", "funding round type(s) raised, see 'ark catalog funding-types'")
	str(&s.fundingTotal, "funding-total", "total funding raised in USD, as a band like 1m-5m or 10m+")
	str(&s.fundingLast, "funding-last", "last funding round amount in USD, as a band like 500k-2m")
	str(&s.fundingDate, "funding-date", "last funding date band as YYYY-MM-DD..YYYY-MM-DD (either side may be empty)")
	// StringArray: a spec's department list is itself comma-separated, so
	// these flags are repeated rather than split.
	fs.StringArrayVar(&s.headcount, "headcount", nil, "people per department, dept[,dept]:band (e.g. sales:10-100); repeatable")
	fs.StringArrayVar(&s.headcountGrowth, "headcount-growth", nil, "growth % per department, dept[,dept]:from..to:window (e.g. sales:-20..-5:TWELVE); repeatable")
	track("headcount", "headcount-growth")

	if scope == companyScope {
		slice(&s.lookalike, "lookalike", "up to 5 domains or LinkedIn company URLs; returns similar companies")
	}
	slice(&s.excludeList, "exclude-list", "suppression list id(s) created with 'ark lists create'")
}

// accountFlagName returns the account flag name for a scope: short names
// for company search, "company-" prefixed for people commands where the
// short names mean the person.
func accountFlagName(scope filterScope) func(string) string {
	return func(base string) string {
		if scope == peopleScope {
			return "company-" + base
		}
		return base
	}
}

// plan validates the flags and builds the request body. maxSize is the
// largest size the target endpoint accepts.
func (s *searchFlags) plan(cmd *cobra.Command, page, size, maxSize int) (*searchPlan, error) {
	if s.body != "" {
		return s.rawPlan(cmd, page, size, maxSize)
	}
	if err := s.checkCompanions(cmd); err != nil {
		return nil, err
	}
	mode, err := parseMatch(s.match)
	if err != nil {
		return nil, err
	}
	req := &client.SearchRequest{Page: page, Size: size}
	if err := s.buildFacets(req, mode); err != nil {
		return nil, err
	}
	if err := s.buildAccountScalars(req); err != nil {
		return nil, err
	}
	if s.scope == peopleScope {
		if err := s.buildContactScalars(req); err != nil {
			return nil, err
		}
	}
	if req.Lists, err = s.lists(); err != nil {
		return nil, err
	}
	if req.LookalikeDomains = splitCSV(s.lookalike); len(req.LookalikeDomains) > 5 {
		return nil, usageErrorf("--lookalike takes at most 5 values")
	}
	if !req.Compact().HasFilters() {
		return nil, errNoFilters
	}
	return &searchPlan{req: req}, nil
}

// checkCompanions rejects flags that only modify another flag which was not
// given, since they would otherwise be silently ignored.
func (s *searchFlags) checkCompanions(cmd *cobra.Command) error {
	changed := cmd.Flags().Changed
	if s.near == "" && (changed("radius") || changed("radius-unit")) {
		return usageErrorf("--radius and --radius-unit need --near")
	}
	for _, f := range s.facets {
		if f.kind != keywordFacet || len(f.sources) == 0 {
			continue
		}
		v := s.values[f.name]
		if len(v.include)+len(v.exclude)+len(v.allInclude)+len(v.allExclude) == 0 {
			return usageErrorf("--%s-source needs --%s", f.name, f.name)
		}
	}
	return nil
}

// buildFacets turns the four flag lists of every facet into its filter.
func (s *searchFlags) buildFacets(req *client.SearchRequest, mode string) error {
	for _, f := range s.facets {
		v := s.values[f.name]
		lists := [4][]string{v.include, v.exclude, v.allInclude, v.allExclude}
		flagNames := [4]string{f.name, "exclude-" + f.name, "all-" + f.name, "all-exclude-" + f.name}
		var clean [4][]string
		empty := true
		for i, list := range lists {
			values, err := f.parse(flagNames[i], list)
			if err != nil {
				return err
			}
			clean[i] = values
			empty = empty && len(values) == 0
		}
		if empty {
			continue
		}
		inc, exc, allInc, allExc := clean[0], clean[1], clean[2], clean[3]
		switch f.kind {
		case plainFacet:
			f.plain(req, client.NewPlain(client.NewValues(inc, exc), client.NewValues(allInc, allExc)))
		case matchedFacet:
			f.matched(req, client.NewMatched(
				client.NewMatchValues(mode, inc, exc), client.NewMatchValues(mode, allInc, allExc)))
		case keywordFacet:
			sources, err := f.keywordSources()
			if err != nil {
				return err
			}
			f.keyword(req, client.NewKeywords(
				client.NewKeywordValues(mode, sources, inc, exc), client.NewKeywordValues(mode, sources, allInc, allExc)))
		}
	}
	return nil
}

// parse splits, validates, and normalises one flag's values.
func (f *facet) parse(flagName string, raw []string) ([]string, error) {
	values := splitCSV(raw)
	if f.enum != nil {
		return enumValues(flagName, f.enum, values)
	}
	if f.ids {
		for _, id := range values {
			if err := client.ValidateID("--"+flagName, id); err != nil {
				return nil, usageError(err)
			}
		}
	}
	return values, nil
}

func (f *facet) keywordSources() ([]string, error) {
	given := splitCSV(f.sources)
	if len(given) == 0 {
		return f.defaultSources, nil
	}
	cat, _ := catalog.Lookup(f.sourceCatalog)
	sources, err := enumValues(f.name+"-source", cat.Values, given)
	if err != nil {
		return nil, err
	}
	if f.maxSources > 0 && len(sources) > f.maxSources {
		return nil, usageErrorf("--%s-source takes at most %d sections", f.name, f.maxSources)
	}
	return sources, nil
}

// buildAccountScalars sets the numeric, range, and structured company-level
// filters that are not simple value lists.
func (s *searchFlags) buildAccountScalars(req *client.SearchRequest) error {
	acct := accountFlagName(s.scope)
	var p parser
	a := account(req)
	a.EmployeeSize = client.Ranges(p.bands("employees", s.employees))
	a.RetailSize = client.Ranges(p.bands("retail-locations", s.retailLocations))
	a.Revenue = p.rangeOrKind("revenue", s.revenue)
	a.FoundedYear = p.year("founded", s.founded)
	a.GeoLocation = p.geo(s.near, s.radius, s.radiusUnit)
	a.Funding = s.funding(&p)
	a.Metric = s.metrics(&p)
	if langCount := p.band(acct("language-count"), s.companyLanguageCount); langCount != nil {
		if a.Language == nil {
			a.Language = &client.LanguageFilter{}
		}
		a.Language.Range = langCount
	}
	return p.err
}

func (s *searchFlags) funding(p *parser) *client.Funding {
	f := &client.Funding{
		Type:        p.enum("funding-type", catalog.FundingTypeValues, s.fundingType),
		TotalAmount: p.band("funding-total", s.fundingTotal),
		LastAmount:  p.band("funding-last", s.fundingLast),
		Duration:    p.dateBand(s.fundingDate),
	}
	if len(f.Type) == 0 && f.TotalAmount == nil && f.LastAmount == nil && f.Duration == nil {
		return nil
	}
	return f
}

func (s *searchFlags) metrics(p *parser) *client.Metrics {
	var m client.Metrics
	for _, spec := range nonEmpty(s.headcount) {
		m.Employee = append(m.Employee, p.headcount(spec))
	}
	for _, spec := range nonEmpty(s.headcountGrowth) {
		m.Growth = append(m.Growth, p.headcountGrowth(spec))
	}
	if len(m.Employee) == 0 && len(m.Growth) == 0 {
		return nil
	}
	return &m
}

// buildContactScalars sets the numeric and structured person-level filters.
func (s *searchFlags) buildContactScalars(req *client.SearchRequest) error {
	var p parser
	c := contact(req)
	if langCount := p.band("language-count", s.languageCount); langCount != nil {
		if c.Language == nil {
			c.Language = &client.MatchedLanguageFilter{}
		}
		c.Language.Range = langCount
	}
	if eduDate := p.band("education-date", s.educationDate); eduDate != nil || s.studying {
		date := &client.EducationDate{}
		if eduDate != nil {
			date.Start, date.End = eduDate.Start, eduDate.End
		}
		if s.studying {
			present := true
			date.Present = &present
		}
		education(req).Date = date
	}
	followers := p.bands("followers", s.followers)
	connections := p.bands("connections", s.connections)
	if len(followers) > 0 || len(connections) > 0 {
		c.SocialMediaFollower = &client.SocialMediaFollower{LinkedIn: &client.FollowerCounts{
			Followers:   client.Ranges(followers),
			Connections: client.Ranges(connections),
		}}
	}
	duration := &client.Duration{
		CurrentCompany: p.tenure("tenure-company", s.tenureCompany),
		CurrentJob:     p.tenure("tenure-job", s.tenureJob),
		Total:          p.tenure("tenure-total", s.tenureTotal),
	}
	if duration.CurrentCompany != nil || duration.CurrentJob != nil || duration.Total != nil {
		currentPosition(req).Duration = duration
	}
	return p.err
}

func (s *searchFlags) lists() (*client.ListsFilter, error) {
	ids := splitCSV(s.excludeList)
	if len(ids) == 0 {
		return nil, nil
	}
	if len(ids) > 10 {
		return nil, usageErrorf("--exclude-list takes at most 10 lists")
	}
	for _, id := range ids {
		if err := client.ValidateID("--exclude-list", id); err != nil {
			return nil, usageError(err)
		}
	}
	exclude := &client.ListExclude{Exclude: ids}
	if s.scope == companyScope {
		return &client.ListsFilter{CompanyID: exclude}, nil
	}
	return &client.ListsFilter{PeopleID: exclude}, nil
}

// rawPlan loads --body and overlays page and size when those flags were set
// explicitly (or when the body lacks them, since the API requires both). The
// body is held to the same rules as flag-built requests: it must carry at
// least one filter and its page and size must be within the API's limits.
func (s *searchFlags) rawPlan(cmd *cobra.Command, page, size, maxSize int) (*searchPlan, error) {
	for _, name := range s.names {
		if cmd.Flags().Changed(name) {
			return nil, usageErrorf("--body cannot be combined with filter flags (--%s)", name)
		}
	}
	raw, err := readBody(cmd, s.body)
	if err != nil {
		return nil, err
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, usageError(err)
	}
	if err := setJSONField(body, "page", page, cmd.Flags().Changed("page")); err != nil {
		return nil, err
	}
	if err := setJSONField(body, "size", size, cmd.Flags().Changed("size")); err != nil {
		return nil, err
	}
	if err := validateRawBody(body, maxSize); err != nil {
		return nil, err
	}
	merged, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return &searchPlan{raw: merged}, nil
}

// validateRawBody applies the billing guards to a --body request.
func validateRawBody(body map[string]json.RawMessage, maxSize int) error {
	filtered := false
	for _, key := range []string{"account", "contact", "lookalikeDomains"} {
		if raw, ok := body[key]; ok && !client.IsEmptyJSON(raw) {
			filtered = true
		}
	}
	if !filtered {
		return errNoFilters
	}
	var page, size int
	if json.Unmarshal(body["page"], &page) != nil || page < 0 {
		return usageErrorf("--body: page must be an integer, 0 or greater")
	}
	if json.Unmarshal(body["size"], &size) != nil || size < 1 || size > maxSize {
		return usageErrorf("--body: size must be an integer between 1 and %d", maxSize)
	}
	return nil
}

// Lazy accessors for the nested request objects, so facet setters and
// scalar builders can share parents without ordering constraints.

func account(r *client.SearchRequest) *client.AccountFilter {
	if r.Account == nil {
		r.Account = &client.AccountFilter{}
	}
	return r.Account
}

func contact(r *client.SearchRequest) *client.ContactFilter {
	if r.Contact == nil {
		r.Contact = &client.ContactFilter{}
	}
	return r.Contact
}

func employee(r *client.SearchRequest) *client.EmployeeFilter {
	a := account(r)
	if a.Employee == nil {
		a.Employee = &client.EmployeeFilter{}
	}
	return a.Employee
}

func experience(r *client.SearchRequest) *client.Experience {
	c := contact(r)
	if c.Experience == nil {
		c.Experience = &client.Experience{}
	}
	return c.Experience
}

func currentPosition(r *client.SearchRequest) *client.CurrentPosition {
	e := experience(r)
	if e.Current == nil {
		e.Current = &client.CurrentPosition{}
	}
	return e.Current
}

func companyIDs(r *client.SearchRequest) *client.ScopedIDs {
	c := contact(r)
	if c.Company == nil {
		c.Company = &client.ScopedIDs{}
	}
	return c.Company
}

func education(r *client.SearchRequest) *client.Education {
	c := contact(r)
	if c.Education == nil {
		c.Education = &client.Education{}
	}
	return c.Education
}
