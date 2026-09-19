package cmd

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/ai-ark-com/ark-cli/internal/catalog"
	"github.com/ai-ark-com/ark-cli/internal/client"
)

// Parsers for the scalar and range filter flags. Bands are written as
// "a-b", "a+", "-b", or a single value; numbers accept k/m/b suffixes.

// parser accumulates the first error of a sequence of parses, so builders
// can read like assignments instead of error ladders.
type parser struct {
	err error
}

func (p *parser) fail(flag string, err error) {
	if p.err == nil && err != nil {
		p.err = usageErrorf("--%s: %v", flag, err)
	}
}

func (p *parser) bands(flag string, in []string) []client.Range {
	specs := splitCSV(in)
	bands := make([]client.Range, 0, len(specs))
	for _, spec := range specs {
		r, err := parseBand(spec)
		p.fail(flag, err)
		bands = append(bands, r)
	}
	return bands
}

// band parses a single optional band; nil when the flag is unset.
func (p *parser) band(flag, spec string) *client.Range {
	if strings.TrimSpace(spec) == "" {
		return nil
	}
	r, err := parseBand(spec)
	p.fail(flag, err)
	return &r
}

// rangeOrKind handles filters that accept bands or the words any/none.
func (p *parser) rangeOrKind(flag string, in []string) *client.RangeFilter {
	specs := splitCSV(in)
	if len(specs) == 1 {
		switch strings.ToLower(specs[0]) {
		case "any", "all":
			return &client.RangeFilter{Type: client.RangeTypeAll}
		case "none":
			return &client.RangeFilter{Type: client.RangeTypeNone}
		}
	}
	return client.Ranges(p.bands(flag, specs))
}

func (p *parser) year(flag, spec string) *client.YearFilter {
	switch strings.ToLower(strings.TrimSpace(spec)) {
	case "":
		return nil
	case "any", "all":
		return &client.YearFilter{Type: client.RangeTypeAll}
	case "none":
		return &client.YearFilter{Type: client.RangeTypeNone}
	}
	return &client.YearFilter{Type: client.RangeTypeRange, Range: p.band(flag, spec)}
}

func (p *parser) enum(flag string, allowed, in []string) []string {
	values, err := enumValues(flag, allowed, in)
	if p.err == nil {
		p.err = err
	}
	return values
}

// dateBand parses --funding-date, "YYYY-MM-DD..YYYY-MM-DD" (either side
// optional), into the epoch-millisecond strings the API expects. The end
// date is inclusive: it maps to the last millisecond of that day.
func (p *parser) dateBand(spec string) *client.TimeRange {
	const flag = "funding-date"
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil
	}
	lo, hi, ok := strings.Cut(spec, "..")
	if !ok || (lo == "" && hi == "") {
		p.fail(flag, fmt.Errorf("want YYYY-MM-DD..YYYY-MM-DD (either side may be empty), got %q", spec))
		return nil
	}
	millis := func(s string, endOfDay bool) string {
		if s == "" {
			return ""
		}
		t, err := time.Parse("2006-01-02", s)
		if err != nil {
			p.fail(flag, fmt.Errorf("%q is not a YYYY-MM-DD date", s))
			return ""
		}
		if endOfDay {
			t = t.AddDate(0, 0, 1).Add(-time.Millisecond)
		}
		return strconv.FormatInt(t.UnixMilli(), 10)
	}
	return &client.TimeRange{Start: millis(lo, false), End: millis(hi, true)}
}

// geo builds a radius search from --near, --radius, and --radius-unit.
func (p *parser) geo(near string, radius float64, unit string) *client.GeoLocation {
	if near == "" {
		return nil
	}
	latStr, lngStr, ok := strings.Cut(near, ",")
	lat, err1 := strconv.ParseFloat(strings.TrimSpace(latStr), 64)
	lng, err2 := strconv.ParseFloat(strings.TrimSpace(lngStr), 64)
	if !ok || err1 != nil || err2 != nil || !(math.Abs(lat) <= 90 && math.Abs(lng) <= 180) {
		p.fail("near", fmt.Errorf("want lat,lng with lat in [-90,90] and lng in [-180,180], got %q", near))
	}
	if unit = strings.ToLower(unit); unit != "km" && unit != "mi" {
		p.fail("radius-unit", errors.New("must be km or mi"))
	}
	if !(radius > 0) || math.IsInf(radius, 0) {
		p.fail("radius", errors.New("must be a positive number"))
	}
	return &client.GeoLocation{Position: client.Coordinate{Lat: lat, Lng: lng}, Radius: radius, Unit: unit}
}

// headcount parses "dept[,dept]:band" into a department headcount band.
func (p *parser) headcount(spec string) client.DepartmentHeadcount {
	depts, band, ok := strings.Cut(spec, ":")
	if !ok {
		p.fail("headcount", fmt.Errorf("want dept[,dept]:band, e.g. sales:10-100, got %q", spec))
		return client.DepartmentHeadcount{}
	}
	r, err := parseBand(band)
	p.fail("headcount", err)
	return client.DepartmentHeadcount{
		Function: p.enum("headcount", catalog.MetricFunctions, []string{depts}),
		Start:    r.Start,
		End:      r.End,
	}
}

// headcountGrowth parses "dept[,dept]:from..to:window" into a growth band;
// percentages may be negative, hence the ".." separator.
func (p *parser) headcountGrowth(spec string) client.DepartmentGrowth {
	parts := strings.Split(spec, ":")
	if len(parts) != 3 {
		p.fail("headcount-growth", fmt.Errorf("want dept[,dept]:from..to:window, e.g. marketing:10..15:SIX, got %q", spec))
		return client.DepartmentGrowth{}
	}
	lo, hi, ok := strings.Cut(parts[1], "..")
	start, err1 := parseSigned(lo)
	end, err2 := parseSigned(hi)
	if !ok || err1 != nil || err2 != nil || (start == nil && end == nil) || (start != nil && end != nil && *start > *end) {
		p.fail("headcount-growth", fmt.Errorf("want from..to percentages, e.g. -20..-5 or 10..15, got %q", parts[1]))
	}
	window, ok := catalog.Normalize(catalog.TimeFrameValues, parts[2])
	if !ok {
		p.fail("headcount-growth", fmt.Errorf("window must be one of %s", joinValues(catalog.TimeFrameValues)))
	}
	return client.DepartmentGrowth{
		Function:  p.enum("headcount-growth", catalog.MetricFunctions, []string{parts[0]}),
		Start:     start,
		End:       end,
		TimeFrame: window,
	}
}

// tenure parses a band like "2y-5y", "1y6m+", "-3y", or "2-5" (years) into
// the API's {year, month} min/max form.
func (p *parser) tenure(flag, spec string) *client.YearMonthBand {
	spec = strings.ReplaceAll(strings.TrimSpace(spec), " ", "")
	if spec == "" {
		return nil
	}
	lo, hi := splitBand(spec)
	minYM, err1 := parseYearMonth(lo)
	maxYM, err2 := parseYearMonth(hi)
	if err1 != nil || err2 != nil || (minYM == nil && maxYM == nil) {
		p.fail(flag, fmt.Errorf("want a band like 2y-5y, 1y6m+ or -3y, got %q", spec))
		return nil
	}
	return &client.YearMonthBand{Min: minYM, Max: maxYM}
}

// splitBand splits "a-b", "a+", "-b", or "a" (exact) into its bounds.
func splitBand(spec string) (lo, hi string) {
	switch {
	case strings.HasSuffix(spec, "+"):
		return strings.TrimSuffix(spec, "+"), ""
	case strings.Contains(spec, "-"):
		lo, hi, _ = strings.Cut(spec, "-")
		return lo, hi
	default:
		return spec, spec
	}
}

// parseBand parses "a-b", "a+", "-b" or a single number "a" (exact).
func parseBand(spec string) (client.Range, error) {
	spec = strings.ReplaceAll(strings.TrimSpace(spec), " ", "")
	lo, hi := splitBand(spec)
	if lo == "" && hi == "" {
		return client.Range{}, fmt.Errorf("empty band %q", spec)
	}
	start, err1 := parseBound(lo)
	end, err2 := parseBound(hi)
	if err1 != nil || err2 != nil {
		return client.Range{}, fmt.Errorf("%q is not a band like 51-200, 1000+ or -50", spec)
	}
	if start != nil && end != nil && *start > *end {
		return client.Range{}, fmt.Errorf("band %q has start greater than end", spec)
	}
	return client.Range{Start: start, End: end}, nil
}

// parseBound parses a non-negative number with an optional k / m / b
// suffix; "" is open.
func parseBound(s string) (*float64, error) {
	if s == "" {
		return nil, nil
	}
	mult := 1.0
	switch strings.ToLower(s[len(s)-1:]) {
	case "k":
		mult, s = 1e3, s[:len(s)-1]
	case "m":
		mult, s = 1e6, s[:len(s)-1]
	case "b":
		mult, s = 1e9, s[:len(s)-1]
	}
	v, err := parseSigned(s)
	if err != nil || v == nil || *v < 0 {
		return nil, fmt.Errorf("%q is not a non-negative number", s)
	}
	*v *= mult
	return v, nil
}

// parseSigned parses a finite number; "" is open (nil).
func parseSigned(s string) (*float64, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsInf(v, 0) || math.IsNaN(v) {
		return nil, fmt.Errorf("%q is not a number", s)
	}
	return &v, nil
}

// parseYearMonth parses "3", "3y", "18m", or "3y6m"; "" is open. Months
// beyond 11 roll into years.
func parseYearMonth(s string) (*client.YearMonth, error) {
	if s == "" {
		return nil, nil
	}
	years, months := 0, 0
	rest := strings.ToLower(s)
	if y, tail, ok := strings.Cut(rest, "y"); ok {
		n, err := strconv.Atoi(y)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("%q is not a duration like 2y6m", s)
		}
		years, rest = n, tail
	}
	if rest != "" {
		m, isMonths := strings.CutSuffix(rest, "m")
		n, err := strconv.Atoi(m)
		if err != nil || n < 0 {
			return nil, fmt.Errorf("%q is not a duration like 2y6m", s)
		}
		switch {
		case isMonths:
			months = n
		case years == 0:
			years = n // bare number: years
		default:
			return nil, fmt.Errorf("%q is not a duration like 2y6m", s)
		}
	}
	return &client.YearMonth{Year: years + months/12, Month: months % 12}, nil
}

func parseMatch(s string) (string, error) {
	mode, ok := catalog.Normalize(catalog.MatchModeValues, s)
	if !ok {
		return "", usageErrorf("--match must be smart, word, or strict")
	}
	return mode, nil
}

// enumValues validates (already split) user input against an embedded
// catalog and returns the canonical spellings.
func enumValues(flag string, allowed, in []string) ([]string, error) {
	out := make([]string, 0, len(in))
	for _, v := range splitCSV(in) {
		canon, ok := catalog.Normalize(allowed, v)
		if !ok {
			return nil, usageErrorf("--%s: unknown value %q (allowed: %s)", flag, v, joinValues(allowed))
		}
		out = append(out, canon)
	}
	return out, nil
}

// nonEmpty trims repeated flag values and drops blanks, without splitting
// on commas.
func nonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func joinValues(v []string) string { return strings.Join(v, ", ") }
