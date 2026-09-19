package cmd

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
)

// TestFlagsReachEverySpecPath sets every filter flag of every search command
// and checks that the union of the generated bodies contains every leaf path
// of the request schema published in the OpenAPI spec (testdata/search_paths.json,
// extracted from https://docs.ai-ark.com/reference). A path the flags cannot
// produce means a filter the CLI does not expose.
func TestFlagsReachEverySpecPath(t *testing.T) {
	raw, err := os.ReadFile("testdata/search_paths.json")
	if err != nil {
		t.Fatal(err)
	}
	var specPaths []string
	if err := json.Unmarshal(raw, &specPaths); err != nil {
		t.Fatal(err)
	}

	reached := map[string]bool{}
	collect := func(body map[string]any) {
		for _, p := range leafPaths("", body) {
			reached[p] = true
		}
	}
	collect(dryRun(t, append([]string{"people", "search"}, everyFlag(peopleScope)...)...))
	collect(dryRun(t, append([]string{"company", "search"}, everyFlag(companyScope)...)...))
	collect(dryRun(t, append([]string{"people", "export", "--webhook", "https://example.com/hook"}, everyFlag(peopleScope)...)...))

	var missing []string
	for _, p := range specPaths {
		if !reached[p] {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%d spec paths are not reachable from flags:\n  %s", len(missing), strings.Join(missing, "\n  "))
	}
}

// everyFlag returns arguments that set every filter flag of a scope, using
// each facet's sample value and a valid value for every scalar flag.
func everyFlag(scope filterScope) []string {
	var args []string
	for _, f := range facetsFor(scope) {
		for _, prefix := range []string{"", "exclude-", "all-", "all-exclude-"} {
			args = append(args, "--"+prefix+f.name, f.sample)
		}
		if f.kind == keywordFacet {
			args = append(args, "--"+f.name+"-source", f.defaultSources[0])
		}
	}
	acct := accountFlagName(scope)
	args = append(args,
		"--founded", "2015-2022",
		"--employees", "51-200", "--employees", "1000+",
		"--retail-locations", "10-100",
		"--revenue", "1m-10m",
		"--"+acct("language-count"), "2-5",
		"--near", "52.52,13.405", "--radius", "25", "--radius-unit", "mi",
		"--funding-type", "SEED,SERIES_A",
		"--funding-total", "1m-5m", "--funding-last", "500k-2m",
		"--funding-date", "2021-05-01..2025-05-01",
		"--headcount", "sales,marketing:10-100",
		"--headcount-growth", "marketing:10..15:SIX",
		"--exclude-list", sampleUUID,
	)
	if scope == peopleScope {
		args = append(args,
			"--language-count", "2-4",
			"--education-date", "2015-2025", "--studying",
			"--followers", "1k-5k", "--followers", "30k+",
			"--connections", "500-3000",
			"--tenure-company", "2y-5y", "--tenure-job", "1y6m-4y", "--tenure-total", "5y-10y",
		)
	} else {
		args = append(args, "--lookalike", "raisin.com")
	}
	return args
}

// leafPaths lists the dotted paths of every scalar in a decoded JSON
// document, using "[]" for array elements, matching the spec extraction.
func leafPaths(prefix string, v any) []string {
	switch t := v.(type) {
	case map[string]any:
		var out []string
		for k, child := range t {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			out = append(out, leafPaths(key, child)...)
		}
		return out
	case []any:
		if len(t) == 0 {
			return []string{prefix + "[]"}
		}
		seen := map[string]bool{}
		for _, item := range t {
			for _, p := range leafPaths(prefix+"[]", item) {
				seen[p] = true
			}
		}
		out := make([]string, 0, len(seen))
		for p := range seen {
			out = append(out, p)
		}
		sort.Strings(out)
		return out
	default:
		return []string{prefix}
	}
}

func TestTenure(t *testing.T) {
	var p parser
	ym := p.tenure("tenure-company", "1y6m-3y")
	if p.err != nil || ym.Min.Year != 1 || ym.Min.Month != 6 || ym.Max.Year != 3 || ym.Max.Month != 0 {
		t.Errorf("tenure(1y6m-3y) = %+v, %v", ym, p.err)
	}
	ym = p.tenure("tenure-job", "2+")
	if p.err != nil || ym.Min.Year != 2 || ym.Max != nil {
		t.Errorf("tenure(2+) = %+v, %v", ym, p.err)
	}
	ym = p.tenure("tenure-total", "-18m")
	if p.err != nil || ym.Min != nil || ym.Max.Year != 1 || ym.Max.Month != 6 {
		t.Errorf("tenure(-18m) = %+v, %v (months should roll into years)", ym, p.err)
	}
	for _, bad := range []string{"x", "+", "-", "1y2"} {
		var p parser
		p.tenure("tenure-job", bad)
		if p.err == nil {
			t.Errorf("tenure(%q) should fail", bad)
		}
	}
}

func TestDateBandHeadcountAndGrowth(t *testing.T) {
	var p parser
	d := p.dateBand("2021-05-01..2021-05-01")
	if p.err != nil || d.Start != "1619827200000" || d.End != "1619913599999" {
		t.Errorf("dateBand = %+v, %v (end must be the last millisecond of the day)", d, p.err)
	}
	if d := p.dateBand("..2021-05-01"); p.err != nil || d.Start != "" {
		t.Errorf("open start = %+v, %v", d, p.err)
	}
	p = parser{}
	p.dateBand("2021-05-01")
	if p.err == nil {
		t.Error("date band without .. should fail")
	}

	p = parser{}
	h := p.headcount("sales, Marketing:10-100")
	if p.err != nil || len(h.Function) != 2 || h.Function[1] != "marketing" || *h.End != 100 {
		t.Errorf("headcount = %+v, %v", h, p.err)
	}
	g := p.headcountGrowth("engineering:-20..-5:twelve")
	if p.err != nil || g.TimeFrame != "TWELVE" || *g.Start != -20 || *g.End != -5 {
		t.Errorf("headcountGrowth = %+v, %v", g, p.err)
	}
	for _, bad := range []string{"engineering:10-15:SIX", "engineering:10..15", "engineering:15..10:SIX", "nope:1..2:SIX"} {
		var p parser
		p.headcountGrowth(bad)
		if p.err == nil {
			t.Errorf("headcountGrowth(%q) should fail", bad)
		}
	}
}

func TestParseBand(t *testing.T) {
	tests := map[string][2]*float64{
		"51-200": {f(51), f(200)},
		"1000+":  {f(1000), nil},
		"-50":    {nil, f(50)},
		"1k-10m": {f(1e3), f(1e7)},
		"2015":   {f(2015), f(2015)},
		"2.5b+":  {f(2.5e9), nil},
	}
	for in, want := range tests {
		got, err := parseBand(in)
		if err != nil {
			t.Errorf("parseBand(%q): %v", in, err)
			continue
		}
		if !sameBound(got.Start, want[0]) || !sameBound(got.End, want[1]) {
			t.Errorf("parseBand(%q) = %v-%v", in, deref(got.Start), deref(got.End))
		}
	}
	for _, bad := range []string{"", "+", "a-b", "-", "10-5", "-1-5", "inf+", "nan-1", "1e400+"} {
		if _, err := parseBand(bad); err == nil {
			t.Errorf("parseBand(%q) should fail", bad)
		}
	}
}

func f(v float64) *float64 { return &v }

func deref(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

func sameBound(a, b *float64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
