package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// run executes the CLI with args and returns stdout and the error.
func run(t *testing.T, stdin string, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd(BuildInfo{Version: "test"})
	var stdout, stderr bytes.Buffer
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetIn(strings.NewReader(stdin))
	root.SetArgs(args)
	err := root.Execute()
	return stdout.String(), err
}

// dryRun executes a search command with --dry-run and decodes the printed body.
func dryRun(t *testing.T, args ...string) map[string]any {
	t.Helper()
	stdout, err := run(t, "", append(args, "--dry-run")...)
	if err != nil {
		t.Fatalf("%v: %v", args, err)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(stdout), &body); err != nil {
		t.Fatalf("dry-run output is not JSON: %v\n%s", err, stdout)
	}
	return body
}

// at walks a decoded JSON document by dotted path.
func at(t *testing.T, doc any, path string) any {
	t.Helper()
	cur := doc
	for _, key := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			t.Fatalf("path %q: %T is not an object", path, cur)
		}
		if cur, ok = m[key]; !ok {
			t.Fatalf("path %q: key %q missing in %v", path, key, m)
		}
	}
	return cur
}

func TestPeopleSearchDryRunBuildsSpecShapedBody(t *testing.T) {
	body := dryRun(t, "people", "search",
		"--location", "Germany,Austria", "--exclude-location", "Berlin",
		"--seniority", "C-Suite,vp", "--title", "founder", "--match", "word",
		"--company-domain", "stripe.com", "--industry", "software development",
		"--employees", "51-200", "--employees", "1k+", "--revenue", "1m-10m",
		"--founded", "2015+", "--company-type", "privately held",
		"--keyword", "climate", "--exclude-list", "294ce93e-8881-414c-8951-e91b5df7f560",
		"--page", "2", "--size", "50")

	if got := at(t, body, "contact.location.any.include"); !equalStrings(got, "Germany", "Austria") {
		t.Errorf("location include = %v", got)
	}
	if got := at(t, body, "contact.location.any.exclude"); !equalStrings(got, "Berlin") {
		t.Errorf("location exclude = %v", got)
	}
	if got := at(t, body, "contact.seniority.any.include"); !equalStrings(got, "c_suite", "vp") {
		t.Errorf("seniority normalised = %v", got)
	}
	if got := at(t, body, "contact.experience.latest.title.any.include.mode"); got != "WORD" {
		t.Errorf("title mode = %v", got)
	}
	if got := at(t, body, "contact.keyword.any.include.sources"); len(got.([]any)) != 2 {
		t.Errorf("default keyword sources = %v", got)
	}
	if got := at(t, body, "account.domain.any.include"); !equalStrings(got, "stripe.com") {
		t.Errorf("domain = %v", got)
	}
	if got := at(t, body, "account.industries.any.include.content"); !equalStrings(got, "software development") {
		t.Errorf("industries = %v", got)
	}
	if got := at(t, body, "account.employeeSize.range"); len(got.([]any)) != 2 {
		t.Errorf("employee bands = %v", got)
	}
	revenue := at(t, body, "account.revenue.range").([]any)[0].(map[string]any)
	if revenue["start"] != 1e6 || revenue["end"] != 1e7 {
		t.Errorf("revenue band = %v", revenue)
	}
	if got := at(t, body, "account.foundedYear.range.start"); got != 2015.0 {
		t.Errorf("founded = %v", got)
	}
	if _, isArray := at(t, body, "account.foundedYear.range").([]any); isArray {
		t.Error("foundedYear.range must be an object, not an array")
	}
	if got := at(t, body, "account.type.any.include"); !equalStrings(got, "PRIVATELY_HELD") {
		t.Errorf("company type normalised = %v", got)
	}
	if got := at(t, body, "lists.people_id.exclude"); !equalStrings(got, "294ce93e-8881-414c-8951-e91b5df7f560") {
		t.Errorf("lists = %v", got)
	}
	if body["page"] != 2.0 || body["size"] != 50.0 {
		t.Errorf("page/size = %v/%v", body["page"], body["size"])
	}
	if _, has := body["webhook"]; has {
		t.Error("search body must not carry a webhook")
	}

	body = dryRun(t, "people", "search", "--company-domain", "stripe.com")
	if _, has := body["contact"]; has {
		t.Error("an empty contact section must be omitted")
	}
}

func TestCompanySearchDryRun(t *testing.T) {
	body := dryRun(t, "company", "search",
		"--lookalike", "raisin.com", "--location", "Germany", "--near", "52.52,13.405", "--radius", "25",
		"--employee-seniority", "head", "--exclude-list", "294ce93e-8881-414c-8951-e91b5df7f560")
	if got := at(t, body, "lookalikeDomains"); !equalStrings(got, "raisin.com") {
		t.Errorf("lookalike = %v", got)
	}
	if got := at(t, body, "account.location.any.include"); !equalStrings(got, "Germany") {
		t.Errorf("location = %v", got)
	}
	if got := at(t, body, "account.geoLocation.radius"); got != 25.0 {
		t.Errorf("radius = %v", got)
	}
	if got := at(t, body, "account.employee.seniority.any.include"); !equalStrings(got, "head") {
		t.Errorf("employee seniority = %v", got)
	}
	if got := at(t, body, "lists.company_id.exclude"); len(got.([]any)) != 1 {
		t.Errorf("lists = %v", got)
	}
	if _, has := body["contact"]; has {
		t.Error("company search must not carry contact filters")
	}
}

func TestExportDryRunCarriesWebhookAndSize(t *testing.T) {
	body := dryRun(t, "people", "export", "--industry", "retail", "--size", "5000", "--webhook", "https://example.com/hook")
	if body["webhook"] != "https://example.com/hook" || body["size"] != 5000.0 {
		t.Errorf("body = %v", body)
	}
}

func TestBodyFlagOverlaysPageAndSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "req.json")
	if err := os.WriteFile(path, []byte(`{"contact":{"seniority":{"any":{"include":["vp"]}}},"size":7}`), 0o600); err != nil {
		t.Fatal(err)
	}
	body := dryRun(t, "people", "preview", "--body", path)
	if body["size"] != 7.0 || body["page"] != 0.0 {
		t.Errorf("body should keep its size and gain a page: %v", body)
	}
	body = dryRun(t, "people", "preview", "--body", path, "--size", "9")
	if body["size"] != 9.0 {
		t.Errorf("explicit --size must win: %v", body)
	}
	body = dryRun(t, "people", "export", "--body", path, "--webhook", "https://example.com/h")
	if body["webhook"] != "https://example.com/h" {
		t.Errorf("export --body must gain the webhook: %v", body)
	}

	stdin := `{"account":{"domain":{"any":{"include":["stripe.com"]}}}}`
	stdout, err := run(t, stdin, "company", "search", "--body", "-", "--dry-run")
	if err != nil || !strings.Contains(stdout, "stripe.com") {
		t.Errorf("stdin body: err=%v out=%s", err, stdout)
	}
}

func TestBodyFlagIsHeldToTheSameGuards(t *testing.T) {
	cases := map[string][]string{
		`{"size":10000}`:                            {"people", "export", "--webhook", "https://x"}, // unfiltered export
		`{"account":{},"contact":{}}`:               {"people", "search"},                           // empty filter objects
		`{"lookalikeDomains":[]}`:                   {"company", "search"},                          // empty lookalike list
		`{"contact":{"location":{}},"size":500}`:    {"people", "search"},                           // size over cap
		`{"contact":{"location":{}},"size":20000}`:  {"people", "export", "--webhook", "https://x"}, // export cap
		`{"contact":{"location":{}},"page":-1}`:     {"people", "search"},                           // negative page
		`{"contact":{"location":{}},"size":"lots"}`: {"people", "search"},                           // non-integer size
		`{"contact":{"location":{}},"size":5}`:      {"people", "search", "--match", "strict"},      // ignored flag
	}
	for body, args := range cases {
		_, err := run(t, body, append(args, "--body", "-", "--dry-run")...)
		if err == nil {
			t.Errorf("%s %v: expected an error", body, args)
			continue
		}
		if !errors.Is(err, errUsage) && !errors.Is(err, errNoFilters) {
			t.Errorf("%s %v: got %v", body, args, err)
		}
	}
}

func TestParseContact(t *testing.T) {
	for in, want := range map[string]string{
		"ada@example.com": "ada@example.com", "+1 (415) 555-0100": "+14155550100", "004915112345678": "004915112345678",
	} {
		if got, ok := parseContact(in); !ok || got != want {
			t.Errorf("parseContact(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, bad := range []string{"", "12345", "Ada <ada@example.com>", "+1-abc", "https://x"} {
		if _, ok := parseContact(bad); ok {
			t.Errorf("parseContact(%q) should be rejected", bad)
		}
	}
}

func TestStrayArgumentsAreNeverEchoed(t *testing.T) {
	const pasted = "sk-live-do-not-print-me"
	for _, args := range [][]string{{"auth", "login", pasted}, {pasted}} {
		_, err := run(t, "", args...)
		if err == nil || strings.Contains(err.Error(), pasted) {
			t.Errorf("%v: error must not echo the argument: %v", args, err)
		}
	}
}

func TestUsageErrors(t *testing.T) {
	cases := [][]string{
		{"people", "search", "--dry-run"},                                                    // no filters
		{"people", "search", "--seniority", "boss", "--dry-run"},                             // unknown enum
		{"people", "search", "--location", "x", "--body", "-", "--dry-run"},                  // body + flags
		{"people", "search", "--location", "x", "--size", "500", "--dry-run"},                // size cap
		{"people", "search", "--location", "x", "--match", "fuzzy", "--dry-run"},             // bad match
		{"people", "search", "--employees", "200-51", "--dry-run"},                           // inverted band
		{"people", "search", "--company-id", "nope", "--dry-run"},                            // bad uuid
		{"people", "search", "--near", "1,2,3", "--dry-run"},                                 // bad coordinate
		{"people", "search", "--location", "x", "-o", "xml", "--dry-run"},                    // bad format
		{"people", "search", "--no-such-flag"},                                               // unknown flag
		{"company", "search", "--lookalike", "a,b,c,d,e,f", "--dry-run"},                     // too many lookalikes
		{"people", "export", "--industry", "x", "--webhook", "http://plain"},                 // insecure webhook
		{"people", "export", "--industry", "x", "--size", "20000", "--webhook", "https://x"}, // export cap
		{"people", "emails", "find", "not-a-uuid", "--webhook", "https://x"},                 // bad track id
		{"people", "enrich", "not-an-id-or-url"},                                             // bad identifier
		{"people", "reverse-lookup", "not-an-email-or-phone"},                                // bad email
		{"people", "phone", "--domain", "x.com"},                                             // missing name
		{"people", "personality", "john-doe"},                                                // not a URL
		{"lists", "create", "--type", "people", "--values", "bad"},                           // bad id
		{"catalog", "nope"}, // unknown catalog
		{"people", "search", "--location", "x", "--radius", "5", "--dry-run"},               // --radius without --near
		{"people", "search", "--location", "x", "--keyword-source", "SUMMARY", "--dry-run"}, // source without --keyword
		{"company", "search", "--lookalike", ",,,,,,", "--dry-run"},                         // no lookalike values
	}
	for _, args := range cases {
		_, err := run(t, "", args...)
		if err == nil {
			t.Errorf("%v: expected an error", args)
			continue
		}
		if !errors.Is(err, errUsage) && !errors.Is(err, errNoFilters) {
			t.Errorf("%v: error should be a usage error, got %v", args, err)
		}
	}
}

func TestExitCodes(t *testing.T) {
	if got := exitCode(errNoFilters); got != exitUsage {
		t.Errorf("no-filters error → %d, want usage exit", got)
	}
	if got := exitCode(errors.New("boom")); got != exitError {
		t.Errorf("generic error → %d", got)
	}
	if got := exitCode(usageErrorf("x")); got != exitUsage {
		t.Errorf("usage error → %d", got)
	}
}

func TestCatalogListsEmbeddedValues(t *testing.T) {
	stdout, err := run(t, "", "catalog", "seniority", "--search", "found")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(stdout) != "founder" {
		t.Errorf("catalog output = %q", stdout)
	}
	stdout, err = run(t, "", "catalog")
	if err != nil || !strings.Contains(stdout, "industries") {
		t.Errorf("catalog index: err=%v out=%s", err, stdout)
	}
}

func equalStrings(v any, want ...string) bool {
	arr, ok := v.([]any)
	if !ok || len(arr) != len(want) {
		return false
	}
	for i, w := range want {
		if arr[i] != w {
			return false
		}
	}
	return true
}
