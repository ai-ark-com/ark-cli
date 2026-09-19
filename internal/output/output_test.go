package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

var peoplePage = json.RawMessage(`{
  "content": [
    {"id": "1", "profile": {"first_name": "Ada", "last_name": "Lovelace", "title": "Analyst"},
     "company": {"summary": {"name": "Analytical Engines"}}, "location": {"short": "London"},
     "link": {"linkedin": "https://linkedin.com/in/ada"}, "skills": [{"name": "math"}]},
    {"id": "2", "profile": {"first_name": "Alan", "last_name": "Tu\tring", "title": "Fellow"},
     "company": {"summary": {"name": "NPL"}}, "location": {"short": "Teddington"},
     "link": {"linkedin": null}}
  ],
  "totalElements": 4218,
  "totalPages": 169,
  "number": 0,
  "size": 2
}`)

func TestTableUsesPeoplePresetAndReportsTotal(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	total, err := Render(&buf, peoplePage, Options{Format: Table})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if total != 4218 {
		t.Errorf("total = %d, want 4218", total)
	}
	out := buf.String()
	for _, want := range []string{"profile.first_name", "Ada", "Lovelace", "Analytical Engines", "Tu ring", "Teddington"} {
		if !strings.Contains(out, want) {
			t.Errorf("table missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "math") {
		t.Errorf("array of objects leaked into table:\n%s", out)
	}
}

func TestCSVWithExplicitColumns(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	_, err := Render(&buf, peoplePage, Options{Format: CSV, Columns: []string{"id", "profile.title", "missing"}})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d CSV lines, want 3:\n%s", len(lines), buf.String())
	}
	if lines[0] != "id,profile.title,missing" || lines[1] != "1,Analyst," {
		t.Errorf("unexpected CSV:\n%s", buf.String())
	}
}

func TestCompanyPresetAndScalarArrays(t *testing.T) {
	t.Parallel()

	body := json.RawMessage(`{"content":[{"id":"c1","summary":{"name":"Jet","industry":"retail","staff":{"total":1537}},
	  "link":{"domain":"jet.com"},"tags":["a","b"]}],"totalElements":1}`)
	var buf bytes.Buffer
	if _, err := Render(&buf, body, Options{Format: Table}); err != nil {
		t.Fatal(err)
	}
	if out := buf.String(); !strings.Contains(out, "summary.staff.total") || !strings.Contains(out, "1537") {
		t.Errorf("company preset not applied:\n%s", out)
	}

	buf.Reset()
	if _, err := Render(&buf, body, Options{Format: CSV, Columns: []string{"tags"}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "a; b") {
		t.Errorf("scalar array not joined:\n%s", buf.String())
	}
}

func TestCSVNeutralisesFormulas(t *testing.T) {
	t.Parallel()

	body := json.RawMessage(`{"content":[{"a":"=HYPERLINK(\"x\")","b":"+1 555","c":"plain"}],"totalElements":1}`)
	var buf bytes.Buffer
	if _, err := Render(&buf, body, Options{Format: CSV, Columns: []string{"a", "b", "c"}}); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 2 || lines[1] != `"'=HYPERLINK(""x"")",'+1 555,plain` {
		t.Errorf("csv = %q", buf.String())
	}
}

func TestNonPageFallsBackToJSON(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	total, err := Render(&buf, json.RawMessage(`{"total": 42}`), Options{Format: Table})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if total != -1 {
		t.Errorf("total = %d, want -1 for non-page", total)
	}
	if !strings.Contains(buf.String(), `"total": 42`) {
		t.Errorf("fallback JSON missing content: %s", buf.String())
	}
}

func TestJSONKeepsKeyOrderAndBigIntegers(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if _, err := Render(&buf, json.RawMessage(`{"z":1,"a":9007199254740993}`), Options{Format: JSON}); err != nil {
		t.Fatal(err)
	}
	if got, want := buf.String(), "{\n  \"z\": 1,\n  \"a\": 9007199254740993\n}\n"; got != want {
		t.Errorf("json = %q, want %q", got, want)
	}

	buf.Reset()
	if _, err := Render(&buf, json.RawMessage(`not json`), Options{Format: JSON}); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "not json" {
		t.Errorf("invalid body should pass through, got %q", buf.String())
	}
}

func TestEmptyPageRendersNothing(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	total, err := Render(&buf, json.RawMessage(`{"content":[],"totalElements":0}`), Options{Format: Table})
	if err != nil || total != 0 || buf.Len() != 0 {
		t.Errorf("empty page: total=%d err=%v out=%q", total, err, buf.String())
	}
}

func TestTruncateAndCell(t *testing.T) {
	t.Parallel()

	if got := truncate("héllo wörld", 8); got != "héllo..." {
		t.Errorf("truncate = %q", got)
	}
	if got := cell(1789821781767.0); got != "1789821781767" {
		t.Errorf("cell(epoch ms) = %q", got)
	}
	if got := cell(1e300); strings.Contains(got, "e+") {
		t.Errorf("cell(1e300) = %q, want a plain decimal", got)
	}
	if got := cell(2.5); got != "2.5" {
		t.Errorf("cell(2.5) = %q", got)
	}
}

func TestParseFormat(t *testing.T) {
	t.Parallel()

	for _, s := range []string{"json", "JSON", " table ", "csv"} {
		if _, err := ParseFormat(s); err != nil {
			t.Errorf("ParseFormat(%q) error: %v", s, err)
		}
	}
	if _, err := ParseFormat("xml"); err == nil {
		t.Error("ParseFormat(xml) should error")
	}
}
