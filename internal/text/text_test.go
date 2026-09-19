package text

import "testing"

func TestSanitize(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"plain":                     "plain",
		"a\nb\tc\r\nd":              "a b c  d",
		"red\x1b[31mtext":           "red[31mtext",
		"c1\u009bcsi":               "c1csi",
		"line sep ":                 "linesep",
		"unicode ok: héllo wörld ✓": "unicode ok: héllo wörld ✓",
	}
	for in, want := range tests {
		if got := Sanitize(in); got != want {
			t.Errorf("Sanitize(%q) = %q, want %q", in, got, want)
		}
	}
}
