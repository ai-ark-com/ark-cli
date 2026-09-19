package client

// The types below mirror the filter envelopes the search endpoints accept.
// Every list-valued facet is wrapped in any/all (OR/AND across values) and
// include/exclude. Two flavours exist:
//
//	plain:   {"any": {"include": ["Germany"], "exclude": ["Berlin"]}}
//	matched: {"any": {"include": {"mode": "SMART", "content": ["founder"]}}}
//
// Constructors return nil for empty input so omitempty drops the field.

// Match mode for text filters.
const (
	MatchSmart  = "SMART"  // AI-assisted, also finds related concepts (API default)
	MatchWord   = "WORD"   // the words must appear, in context
	MatchStrict = "STRICT" // exact match
)

// Values are the include/exclude lists of a plain filter.
type Values struct {
	Include []string `json:"include,omitempty"`
	Exclude []string `json:"exclude,omitempty"`
}

// NewValues builds Values, or nil when both lists are empty.
func NewValues(include, exclude []string) *Values {
	if len(include) == 0 && len(exclude) == 0 {
		return nil
	}
	return &Values{Include: include, Exclude: exclude}
}

// PlainFilter is a facet whose values are exact strings or enum members.
type PlainFilter struct {
	Any *Values `json:"any,omitempty"`
	All *Values `json:"all,omitempty"`
}

// NewPlain builds a PlainFilter, or nil when both scopes are nil.
func NewPlain(anyValues, allValues *Values) *PlainFilter {
	if anyValues == nil && allValues == nil {
		return nil
	}
	return &PlainFilter{Any: anyValues, All: allValues}
}

// Terms is a list of search terms with the mode used to match them.
type Terms struct {
	Mode    string   `json:"mode,omitempty"`
	Content []string `json:"content"`
}

// MatchValues are the include/exclude term sets of a matched filter.
type MatchValues struct {
	Include *Terms `json:"include,omitempty"`
	Exclude *Terms `json:"exclude,omitempty"`
}

// NewMatchValues builds MatchValues, or nil when both lists are empty.
func NewMatchValues(mode string, include, exclude []string) *MatchValues {
	if len(include) == 0 && len(exclude) == 0 {
		return nil
	}
	mv := &MatchValues{}
	if len(include) > 0 {
		mv.Include = &Terms{Mode: mode, Content: include}
	}
	if len(exclude) > 0 {
		mv.Exclude = &Terms{Mode: mode, Content: exclude}
	}
	return mv
}

// MatchFilter is a free-text facet that supports match modes.
type MatchFilter struct {
	Any *MatchValues `json:"any,omitempty"`
	All *MatchValues `json:"all,omitempty"`
}

// NewMatched builds a MatchFilter, or nil when both scopes are nil.
func NewMatched(anyValues, allValues *MatchValues) *MatchFilter {
	if anyValues == nil && allValues == nil {
		return nil
	}
	return &MatchFilter{Any: anyValues, All: allValues}
}

// Range is an inclusive numeric band; a nil bound is open-ended.
type Range struct {
	Start *float64 `json:"start,omitempty"`
	End   *float64 `json:"end,omitempty"`
}

// Range filter types.
const (
	RangeTypeRange = "RANGE" // within the given band(s)
	RangeTypeAll   = "ALL"   // any record that has a value
	RangeTypeNone  = "NONE"  // only records without a value
)

// RangeFilter matches one or more numeric bands (OR'd together): employee
// size, retail size, revenue, follower counts.
type RangeFilter struct {
	Type  string  `json:"type"`
	Range []Range `json:"range,omitempty"`
}

// Ranges builds a RANGE filter, or nil when no band is given.
func Ranges(bands []Range) *RangeFilter {
	if len(bands) == 0 {
		return nil
	}
	return &RangeFilter{Type: RangeTypeRange, Range: bands}
}

// YearFilter is the founded-year filter; unlike RangeFilter its range is a
// single object rather than an array.
type YearFilter struct {
	Type  string `json:"type"`
	Range *Range `json:"range,omitempty"`
}

// KeywordSource is one profile or company section searched by a keyword
// filter, with its own match mode.
type KeywordSource struct {
	Mode   string `json:"mode,omitempty"`
	Source string `json:"source"`
}

// KeywordTerms are search terms plus the sections to search them in.
type KeywordTerms struct {
	Sources []KeywordSource `json:"sources"`
	Content []string        `json:"content"`
}

// KeywordValues are the include/exclude sets of a keyword filter.
type KeywordValues struct {
	Include *KeywordTerms `json:"include,omitempty"`
	Exclude *KeywordTerms `json:"exclude,omitempty"`
}

// NewKeywordValues builds KeywordValues, or nil when both lists are empty.
func NewKeywordValues(mode string, sources, include, exclude []string) *KeywordValues {
	if len(include) == 0 && len(exclude) == 0 {
		return nil
	}
	srcs := make([]KeywordSource, 0, len(sources))
	for _, s := range sources {
		srcs = append(srcs, KeywordSource{Mode: mode, Source: s})
	}
	kv := &KeywordValues{}
	if len(include) > 0 {
		kv.Include = &KeywordTerms{Sources: srcs, Content: include}
	}
	if len(exclude) > 0 {
		kv.Exclude = &KeywordTerms{Sources: srcs, Content: exclude}
	}
	return kv
}

// KeywordFilter is a free-text search across selected sections.
type KeywordFilter struct {
	Any *KeywordValues `json:"any,omitempty"`
	All *KeywordValues `json:"all,omitempty"`
}

// NewKeywords builds a KeywordFilter, or nil when both scopes are nil.
func NewKeywords(anyValues, allValues *KeywordValues) *KeywordFilter {
	if anyValues == nil && allValues == nil {
		return nil
	}
	return &KeywordFilter{Any: anyValues, All: allValues}
}

// ListExclude references suppression lists (see SaveList) whose members are
// removed from the results.
type ListExclude struct {
	Exclude []string `json:"exclude"`
}

// ListsFilter is the top-level "lists" object of a search request.
type ListsFilter struct {
	PeopleID  *ListExclude `json:"people_id,omitempty"`
	CompanyID *ListExclude `json:"company_id,omitempty"`
}
