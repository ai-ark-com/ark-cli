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

func TestMailFiltersDryRun(t *testing.T) {
	body := dryRun(t, "company", "search", "--mail-platform", "google,microsoft", "--exclude-mail-security", "proofpoint")
	if got := at(t, body, "account.mail.platform.any.include"); !equalStrings(got, "google", "microsoft") {
		t.Errorf("mail platform = %v", got)
	}
	if got := at(t, body, "account.mail.security.any.exclude"); !equalStrings(got, "proofpoint") {
		t.Errorf("mail security = %v", got)
	}

	body = dryRun(t, "people", "search", "--seniority", "vp", "--company-mail-platform", "google")
	if got := at(t, body, "account.mail.platform.any.include"); !equalStrings(got, "google") {
		t.Errorf("people search mail platform = %v", got)
	}
	if _, has := at(t, body, "account.mail").(map[string]any)["security"]; has {
		t.Error("an unset mail security filter must be omitted")
	}
}

func TestJobPostSearchDryRun(t *testing.T) {
	body := dryRun(t, "job-posts", "search",
		"--title", "sales manager,account executive", "--exclude-title", "intern",
		"--text", "salesforce", "--location", "Germany,Austria",
		"--seniority", "Mid-Senior,director", "--employment-type", "full-time",
		"--technology", "Salesforce", "--language", "german",
		"--salary", "80k-150k", "--hiring-model", "direct",
		"--posted", "2026-10-01..", "--exclude-company-domain", "sap.com",
		"--company-id", sampleUUID, "--page", "1", "--size", "50")

	if got := at(t, body, "job.title.any.include.mode"); got != "WORD" {
		t.Errorf("default match mode of job posts = %v, want WORD", got)
	}
	if got := at(t, body, "job.title.any.include.content"); !equalStrings(got, "sales manager", "account executive") {
		t.Errorf("title = %v", got)
	}
	if got := at(t, body, "job.title.any.exclude.content"); !equalStrings(got, "intern") {
		t.Errorf("title exclude = %v", got)
	}
	if got := at(t, body, "job.titleOrDescription.any.include.content"); !equalStrings(got, "salesforce") {
		t.Errorf("text = %v", got)
	}
	if got := at(t, body, "job.location.any.include.content"); !equalStrings(got, "Germany", "Austria") {
		t.Errorf("location = %v", got)
	}
	if got := at(t, body, "job.seniority.any.include"); !equalStrings(got, "mid_senior", "director") {
		t.Errorf("seniority normalised = %v", got)
	}
	if got := at(t, body, "job.employmentType.any.include"); !equalStrings(got, "full_time") {
		t.Errorf("employment type normalised = %v", got)
	}
	if got := at(t, body, "job.technologies.any.include"); !equalStrings(got, "Salesforce") {
		t.Errorf("technologies = %v", got)
	}
	if got := at(t, body, "job.language.name.any.include"); !equalStrings(got, "german") {
		t.Errorf("language = %v", got)
	}
	salary := at(t, body, "job.yearlySalaryUsd.range").(map[string]any)
	if salary["start"] != 80000.0 || salary["end"] != 150000.0 {
		t.Errorf("salary must be a single range object, got %v", salary)
	}
	if got := at(t, body, "job.hiringModel"); got != "DIRECT" {
		t.Errorf("hiring model = %v", got)
	}
	posted := at(t, body, "job.postedAt").(map[string]any)
	if posted["type"] != "RANGE" || posted["range"].(map[string]any)["start"] != "2026-10-01" {
		t.Errorf("postedAt = %v", posted)
	}
	if _, has := posted["range"].(map[string]any)["end"]; has {
		t.Error("an open posted end must be omitted")
	}
	if got := at(t, body, "company.domain.any.exclude"); !equalStrings(got, "sap.com") {
		t.Errorf("company domain exclude = %v", got)
	}
	if got := at(t, body, "company.id.any.include"); !equalStrings(got, sampleUUID) {
		t.Errorf("company id = %v", got)
	}
	if body["page"] != 1.0 || body["size"] != 50.0 {
		t.Errorf("page/size = %v/%v", body["page"], body["size"])
	}
	for _, key := range []string{"account", "contact", "lists"} {
		if _, has := body[key]; has {
			t.Errorf("job post search must not carry %q", key)
		}
	}

	body = dryRun(t, "job-posts", "search", "--title", "sales manager", "--match", "smart", "--salary", "any")
	if got := at(t, body, "job.title.any.include.mode"); got != "SMART" {
		t.Errorf("explicit --match = %v", got)
	}
	if got := at(t, body, "job.yearlySalaryUsd.type"); got != "ALL" {
		t.Errorf("salary any = %v", got)
	}
	if _, has := body["company"]; has {
		t.Error("an empty company section must be omitted")
	}

	stdin := `{"job":{"title":{"any":{"include":{"mode":"WORD","content":["sales manager"]}}}}}`
	stdout, err := run(t, stdin, "job-posts", "search", "--body", "-", "--dry-run")
	if err != nil || !strings.Contains(stdout, "sales manager") {
		t.Errorf("stdin body: err=%v out=%s", err, stdout)
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
		{"people", "search", "--location", "x", "--radius", "5", "--dry-run"},                   // --radius without --near
		{"people", "search", "--location", "x", "--keyword-source", "SUMMARY", "--dry-run"},     // source without --keyword
		{"company", "search", "--lookalike", ",,,,,,", "--dry-run"},                             // no lookalike values
		{"company", "search", "--funding-date", "2026-02-01..2026-01-01", "--dry-run"},          // inverted date band
		{"job-posts", "search", "--dry-run"},                                                    // no filters
		{"job-posts", "search", "--description", "x", "--match", "strict", "--dry-run"},         // strict on post text
		{"job-posts", "search", "--text", "x", "--match", "strict", "--dry-run"},                // strict on title or text
		{"job-posts", "search", "--title", "x", "--seniority", "c_suite", "--dry-run"},          // people seniority value
		{"job-posts", "search", "--title", "x", "--hiring-model", "both", "--dry-run"},          // unknown hiring model
		{"job-posts", "search", "--title", "x", "--hiring-model", "direct,agency", "--dry-run"}, // two hiring models
		{"job-posts", "search", "--title", "x", "--posted", "2026-10-01", "--dry-run"},          // date band without ..
		{"job-posts", "search", "--title", "x", "--salary", "150k-80k", "--dry-run"},            // inverted salary band
		{"job-posts", "search", "--company-id", "nope", "--dry-run"},                            // bad uuid
		{"job-posts", "search", "--title", "x", "--employees", "51-200", "--dry-run"},           // company search flag
		{"job-posts", "search", "--title", "x", "--size", "500", "--dry-run"},                   // size cap
		{"job-posts", "search", "--body", "-", "--dry-run"},                                     // body without job filters
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
	stdout, err = run(t, "", "catalog", "mail-security", "--search", "proof")
	if err != nil || strings.TrimSpace(stdout) != "proofpoint" {
		t.Errorf("mail-security catalog: err=%v out=%q", err, stdout)
	}
	stdout, err = run(t, "", "catalog", "job-seniority", "--search", "mid")
	if err != nil || strings.TrimSpace(stdout) != "mid_senior" {
		t.Errorf("job-seniority catalog: err=%v out=%q", err, stdout)
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
