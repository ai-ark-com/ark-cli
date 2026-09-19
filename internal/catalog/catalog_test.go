package catalog

import (
	"strings"
	"testing"
)

func TestParseCSV(t *testing.T) {
	t.Parallel()

	got, err := parseCSV(strings.NewReader("technology,company_count\nmobile friendly,11672404\n wordpress.org ,5571354\n\n,\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []Entry{{Value: "mobile friendly", Count: 11672404}, {Value: "wordpress.org", Count: 5571354}}
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	single, err := parseCSV(strings.NewReader("industry\nreal estate\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(single) != 1 || single[0].Value != "real estate" || single[0].Count != -1 {
		t.Errorf("single column parsed as %+v", single)
	}

	if _, err := parseCSV(strings.NewReader("")); err == nil {
		t.Error("empty CSV should be an error")
	}
}

func TestFilter(t *testing.T) {
	t.Parallel()

	entries := []Entry{{Value: "real estate"}, {Value: "software development"}, {Value: "Retail"}}
	if got := Filter(entries, "TAIL"); len(got) != 1 || got[0].Value != "Retail" {
		t.Errorf("Filter(TAIL) = %+v", got)
	}
	if got := Filter(entries, "re"); len(got) != 3 {
		t.Errorf("Filter(re) = %+v, want all three", got)
	}
	if got := Filter(entries, " "); len(got) != 3 {
		t.Errorf("blank query should keep everything, got %+v", got)
	}
}

func TestNormalize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		values []string
		in     string
		want   string
		ok     bool
	}{
		{SeniorityLevels, "C-Suite", "c_suite", true},
		{SeniorityLevels, "mid level", "mid-level", true},
		{CompanyTypeValues, "privately held", "PRIVATELY_HELD", true},
		{CompanyTypeValues, "startup", "", false},
	}
	for _, tc := range tests {
		got, ok := Normalize(tc.values, tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("Normalize(%q) = %q,%v want %q,%v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestLookupCoversAll(t *testing.T) {
	t.Parallel()

	for _, c := range All() {
		if _, ok := Lookup(c.Name); !ok {
			t.Errorf("Lookup(%q) failed", c.Name)
		}
		if c.Remote() == (len(c.Values) > 0) {
			t.Errorf("catalog %q must be either remote or embedded", c.Name)
		}
	}
	if _, ok := Lookup("nope"); ok {
		t.Error("Lookup(nope) should fail")
	}
}
