package client

import (
	"encoding/json"
	"testing"
)

// The API is strict about the filter envelope. These tests lock the exact
// JSON the CLI sends so a refactor cannot silently change the wire shape.

func plain(include, exclude []string) *PlainFilter {
	return NewPlain(NewValues(include, exclude), nil)
}

func matched(mode string, include, exclude []string) *MatchFilter {
	return NewMatched(NewMatchValues(mode, include, exclude), nil)
}

func TestPeopleRequestJSONShape(t *testing.T) {
	t.Parallel()

	start, end := 51.0, 200.0
	req := &SearchRequest{
		Contact: &ContactFilter{
			Location:  plain([]string{"Germany"}, []string{"Berlin"}),
			Seniority: plain([]string{"c_suite", "vp"}, nil),
			Experience: &Experience{
				Latest: &Position{Title: matched(MatchSmart, []string{"founder"}, nil)},
			},
			Company: &ScopedIDs{Latest: plain([]string{"931fc1a0-1ed5-baf8-0dd2-a068a0fd0430"}, nil)},
		},
		Account: &AccountFilter{
			Industries:   matched(MatchWord, []string{"software development"}, nil),
			EmployeeSize: Ranges([]Range{{Start: &start, End: &end}}),
		},
		Lists: &ListsFilter{PeopleID: &ListExclude{Exclude: []string{"294ce93e-8881-414c-8951-e91b5df7f560"}}},
		Page:  0,
		Size:  25,
	}
	got := mustJSON(t, req)
	want := `{"account":{"industries":{"any":{"include":{"mode":"WORD","content":["software development"]}}},` +
		`"employeeSize":{"type":"RANGE","range":[{"start":51,"end":200}]}},` +
		`"contact":{"company":{"latest":{"any":{"include":["931fc1a0-1ed5-baf8-0dd2-a068a0fd0430"]}}},` +
		`"experience":{"latest":{"title":{"any":{"include":{"mode":"SMART","content":["founder"]}}}}},` +
		`"seniority":{"any":{"include":["c_suite","vp"]}},` +
		`"location":{"any":{"include":["Germany"],"exclude":["Berlin"]}}},` +
		`"lists":{"people_id":{"exclude":["294ce93e-8881-414c-8951-e91b5df7f560"]}},"page":0,"size":25}`
	if got != want {
		t.Errorf("people body\n got: %s\nwant: %s", got, want)
	}
}

func TestCompanyRequestJSONShape(t *testing.T) {
	t.Parallel()

	req := &SearchRequest{
		LookalikeDomains: []string{"raisin.com"},
		Account: &AccountFilter{
			Location:     plain([]string{"Germany"}, nil),
			Technologies: matched(MatchWord, []string{"hubspot"}, []string{"salesforce"}),
			Type:         plain([]string{"PRIVATELY_HELD"}, nil),
			FoundedYear:  &YearFilter{Type: RangeTypeRange, Range: &Range{Start: f(2015)}},
			Keyword:      NewKeywords(NewKeywordValues(MatchSmart, []string{"DESCRIPTION"}, []string{"carbon accounting"}, nil), nil),
		},
		Size: 10,
	}
	got := mustJSON(t, req)
	want := `{"lookalikeDomains":["raisin.com"],"account":{"location":{"any":{"include":["Germany"]}},` +
		`"type":{"any":{"include":["PRIVATELY_HELD"]}},"foundedYear":{"type":"RANGE","range":{"start":2015}},` +
		`"keyword":{"any":{"include":{"sources":[{"mode":"SMART","source":"DESCRIPTION"}],"content":["carbon accounting"]}}},` +
		`"technologies":{"any":{"include":{"mode":"WORD","content":["hubspot"]},"exclude":{"mode":"WORD","content":["salesforce"]}}}},` +
		`"page":0,"size":10}`
	if got != want {
		t.Errorf("company body\n got: %s\nwant: %s", got, want)
	}
}

func TestExportRequestCarriesWebhook(t *testing.T) {
	t.Parallel()

	req := &SearchRequest{
		Account: &AccountFilter{Domain: plain([]string{"stripe.com"}, nil)},
		Size:    1000,
		Webhook: "https://example.com/hook",
	}
	got := mustJSON(t, req)
	want := `{"account":{"domain":{"any":{"include":["stripe.com"]}}},"page":0,"size":1000,"webhook":"https://example.com/hook"}`
	if got != want {
		t.Errorf("export body\n got: %s\nwant: %s", got, want)
	}
}

func TestConstructorsReturnNilWhenEmpty(t *testing.T) {
	t.Parallel()

	if NewValues(nil, nil) != nil || NewPlain(nil, nil) != nil {
		t.Error("empty plain filters should be nil so the field is omitted")
	}
	if NewMatchValues(MatchSmart, nil, nil) != nil || NewMatched(nil, nil) != nil {
		t.Error("empty matched filters should be nil")
	}
	if Ranges(nil) != nil {
		t.Error("Ranges(nil) should be nil")
	}
	if NewKeywordValues(MatchSmart, []string{"HEADLINE"}, nil, nil) != nil || NewKeywords(nil, nil) != nil {
		t.Error("empty keyword filters should be nil")
	}
}

func TestHasFiltersAndCompact(t *testing.T) {
	t.Parallel()

	empty := &SearchRequest{Account: &AccountFilter{}, Contact: &ContactFilter{}, Size: 10}
	if empty.HasFilters() {
		t.Error("request with empty filter objects should report no filters")
	}
	if !(&SearchRequest{Contact: &ContactFilter{Location: plain([]string{"Germany"}, nil)}}).HasFilters() {
		t.Error("request with a contact filter should report filters")
	}
	if !(&SearchRequest{LookalikeDomains: []string{"raisin.com"}}).HasFilters() {
		t.Error("lookalike search should count as filtered")
	}

	req := (&SearchRequest{
		Account: &AccountFilter{Domain: plain([]string{"stripe.com"}, nil)},
		Contact: &ContactFilter{},
		Lists:   &ListsFilter{},
		Size:    10,
	}).Compact()
	if got, want := mustJSON(t, req), `{"account":{"domain":{"any":{"include":["stripe.com"]}}},"page":0,"size":10}`; got != want {
		t.Errorf("compacted body: got %s, want %s", got, want)
	}
}

func TestIsEmptyJSON(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"", "null", "{}", "[]", " { } "} {
		if !IsEmptyJSON(json.RawMessage(raw)) {
			t.Errorf("IsEmptyJSON(%q) should be true", raw)
		}
	}
	if IsEmptyJSON(json.RawMessage(`{"a":1}`)) || IsEmptyJSON(json.RawMessage(`["x"]`)) {
		t.Error("non-empty JSON reported as empty")
	}
}

func f(v float64) *float64 { return &v }

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}
