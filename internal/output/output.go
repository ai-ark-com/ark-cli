// Package output renders API responses as JSON, a compact table, or CSV.
//
// Search endpoints return a Spring "page" object: {content:[...],
// totalElements, ...}. Table and CSV rendering flatten each content item
// into dotted paths (profile.full_name, company.summary.name) and show a
// preset of columns for people and company rows; callers can pick any
// paths explicitly. Responses that are not pages are rendered as JSON.
package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode/utf8"

	"github.com/ai-ark-com/ark-cli/internal/text"
)

// Format is a supported render format.
type Format string

// Supported formats.
const (
	JSON  Format = "json"
	Table Format = "table"
	CSV   Format = "csv"
)

// ParseFormat validates a --output value.
func ParseFormat(s string) (Format, error) {
	switch f := Format(strings.ToLower(strings.TrimSpace(s))); f {
	case JSON, Table, CSV:
		return f, nil
	default:
		return "", fmt.Errorf("unknown output format %q (want json, table, or csv)", s)
	}
}

// Options controls rendering.
type Options struct {
	Format Format
	// Columns are dotted paths to show in table/CSV output. Empty selects a
	// preset based on the row shape.
	Columns []string
}

// Column presets for the row shapes the API returns.
var (
	peopleColumns = []string{
		"id", "profile.first_name", "profile.last_name", "profile.title",
		"company.summary.name", "location.short", "link.linkedin",
	}
	companyColumns = []string{
		"id", "summary.name", "summary.industry", "summary.type", "summary.staff.total",
		"location.headquarter.country", "link.domain",
	}
)

const (
	// maxAutoColumns bounds the generic fallback so a wide object stays readable.
	maxAutoColumns = 12
	// maxCellWidth truncates long table cells; CSV is never truncated.
	maxCellWidth = 48
	// maxExactInt is the largest float64 that is still an exact integer.
	maxExactInt = 1 << 53
)

// page is the subset of the Spring page envelope that is rendered.
type page struct {
	Content       []map[string]any `json:"content"`
	TotalElements *int64           `json:"totalElements"`
}

// Render writes body to w in the requested format. It returns the total
// number of matches reported by the server, or -1 when the response is not
// a page (or is rendered as JSON), so the caller can print a summary line.
func Render(w io.Writer, body json.RawMessage, opts Options) (int64, error) {
	if opts.Format == JSON {
		return -1, renderJSON(w, body)
	}
	var p page
	if err := json.Unmarshal(body, &p); err != nil || p.Content == nil {
		// Not a page (a scalar, an object, ...): JSON is the only faithful view.
		return -1, renderJSON(w, body)
	}

	rows := make([]map[string]string, 0, len(p.Content))
	for _, item := range p.Content {
		flat := map[string]string{}
		flatten("", item, flat)
		rows = append(rows, flat)
	}
	cols := opts.Columns
	if len(cols) == 0 {
		cols = pickColumns(rows)
	}

	var err error
	switch {
	case len(cols) == 0: // empty page: nothing to draw
	case opts.Format == Table:
		err = renderTable(w, rows, cols)
	case opts.Format == CSV:
		err = renderCSV(w, rows, cols)
	default:
		err = fmt.Errorf("unsupported format %q", opts.Format)
	}
	if err != nil {
		return -1, err
	}
	if p.TotalElements != nil {
		return *p.TotalElements, nil
	}
	return int64(len(p.Content)), nil
}

// renderJSON pretty-prints the body without re-encoding it, so key order
// and large integers survive. Invalid JSON is passed through untouched.
func renderJSON(w io.Writer, body json.RawMessage) error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, body, "", "  "); err != nil {
		_, err = w.Write(body)
		return err
	}
	buf.WriteByte('\n')
	_, err := w.Write(buf.Bytes())
	return err
}

// flatten writes every scalar reachable from v into out under its dotted
// path. Arrays of scalars are joined with "; "; arrays of objects are
// omitted, since a table cell cannot show them meaningfully.
func flatten(prefix string, v any, out map[string]string) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			flatten(key, child, out)
		}
	case []any:
		parts := make([]string, 0, len(t))
		for _, item := range t {
			if _, isObj := item.(map[string]any); isObj {
				return
			}
			if s := cell(item); s != "" {
				parts = append(parts, s)
			}
		}
		if prefix != "" {
			out[prefix] = strings.Join(parts, "; ")
		}
	default:
		if prefix != "" {
			out[prefix] = cell(v)
		}
	}
}

// pickColumns chooses a preset when the rows look like people or companies,
// otherwise the first few flattened keys in sorted order.
func pickColumns(rows []map[string]string) []string {
	if len(rows) == 0 {
		return nil
	}
	has := func(prefix string) bool {
		for k := range rows[0] {
			if strings.HasPrefix(k, prefix) {
				return true
			}
		}
		return false
	}
	switch {
	case has("profile."):
		return peopleColumns
	case has("summary."):
		return companyColumns
	}
	seen := map[string]struct{}{}
	for _, row := range rows {
		for k := range row {
			seen[k] = struct{}{}
		}
	}
	cols := make([]string, 0, len(seen))
	for k := range seen {
		cols = append(cols, k)
	}
	sort.Strings(cols)
	return cols[:min(len(cols), maxAutoColumns)]
}

func cell(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		if math.Trunc(t) == t && math.Abs(t) < maxExactInt {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	default:
		return fmt.Sprint(t)
	}
}

func renderTable(w io.Writer, rows []map[string]string, cols []string) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(cols, "\t"))
	line := make([]string, len(cols))
	for _, row := range rows {
		for i, c := range cols {
			line[i] = truncate(text.Sanitize(row[c]), maxCellWidth)
		}
		fmt.Fprintln(tw, strings.Join(line, "\t"))
	}
	return tw.Flush()
}

func renderCSV(w io.Writer, rows []map[string]string, cols []string) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(cols); err != nil {
		return err
	}
	line := make([]string, len(cols))
	for _, row := range rows {
		for i, c := range cols {
			line[i] = csvSafe(row[c])
		}
		if err := cw.Write(line); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// csvSafe neutralises spreadsheet formula injection: a cell starting with
// a formula trigger character is prefixed with a quote so Excel and
// LibreOffice show it as text instead of evaluating it.
func csvSafe(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r", rune(s[0])) {
		return "'" + s
	}
	return s
}

func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n-3]) + "..."
}
