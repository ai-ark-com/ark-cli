// Package text holds small string helpers shared by the client and the
// output renderer.
package text

import (
	"strings"
	"unicode"
)

// Sanitize keeps server-supplied text on one line and strips control
// characters (including C1 controls and line separators), so API data
// cannot reflow the terminal or inject escape sequences.
func Sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\t' || r == '\r':
			return ' '
		case unicode.IsControl(r) || r == ' ' || r == ' ':
			return -1
		}
		return r
	}, s)
}
