package client

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

// The gateway describes every request limit of the caller's plan in the
// response headers. A window is named by a unit ("Second", "Minute") or by
// its length in seconds ("14400"); the plans decide which windows exist, so
// none is named here. Every header is optional.
const (
	limitPrefix     = "X-Ratelimit-Limit-"
	remainingPrefix = "X-Ratelimit-Remaining-"
	resetPrefix     = "X-Ratelimit-Reset-"
	// Plans without X-RateLimit-Reset-* still send this trio, which describes
	// the one window that is closest to its limit.
	tightestLimit     = "RateLimit-Limit"
	tightestRemaining = "RateLimit-Remaining"
	tightestReset     = "RateLimit-Reset"
)

// unknown marks a value the gateway did not report.
const unknown = -1

var unitSeconds = map[string]int64{
	"second": 1, "minute": 60, "hour": 3600, "day": 86400, "month": 2592000, "year": 31536000,
}

// Limit is one request limit of the caller's plan: Max requests per Window.
type Limit struct {
	Window    time.Duration
	Max       int64
	Remaining int64         // requests left in the window; unknown (-1) when not reported
	Reset     time.Duration // time until the window ends; unknown (-1) when not reported
}

// UsedUp reports whether the window has no requests left.
func (l Limit) UsedUp() bool { return l.Remaining == 0 }

// String describes the limit for a person, e.g.
// "340 of 4000 requests per 4 hours used (resets in 1h 2m)".
func (l Limit) String() string {
	s := fmt.Sprintf("%d requests per %s", l.Max, l.window())
	if l.Remaining != unknown {
		s = fmt.Sprintf("%d of %d requests per %s used", l.Max-l.Remaining, l.Max, l.window())
	}
	if l.Reset != unknown {
		s += " (resets in " + formatWait(l.Reset) + ")"
	}
	return s
}

func (l Limit) window() string {
	secs := int64(l.Window / time.Second)
	switch {
	case secs%86400 == 0:
		return count(secs/86400, "day")
	case secs%3600 == 0:
		return count(secs/3600, "hour")
	case secs%60 == 0:
		return count(secs/60, "minute")
	}
	return count(secs, "second")
}

func count(n int64, unit string) string {
	if n == 1 {
		return unit
	}
	return fmt.Sprintf("%d %ss", n, unit)
}

// limitsFrom reads the limits out of a response, shortest window first.
func limitsFrom(h http.Header) []Limit {
	var limits []Limit
	for name := range h {
		window, ok := strings.CutPrefix(name, limitPrefix)
		if !ok {
			continue
		}
		seconds, known := windowSeconds(window)
		limit, valid := headerInt(h, name)
		if !known || !valid {
			continue
		}
		l := Limit{Window: time.Duration(seconds) * time.Second, Max: limit, Remaining: unknown, Reset: unknown}
		if remaining, ok := headerInt(h, remainingPrefix+window); ok {
			l.Remaining = remaining
		}
		if reset, ok := resetSeconds(h, window, l); ok {
			l.Reset = time.Duration(reset) * time.Second
		}
		limits = append(limits, l)
	}
	sort.Slice(limits, func(i, j int) bool { return limits[i].Window < limits[j].Window })
	return limits
}

func windowSeconds(window string) (int64, bool) {
	if seconds, ok := unitSeconds[strings.ToLower(window)]; ok {
		return seconds, true
	}
	seconds, err := strconv.ParseInt(window, 10, 64)
	return seconds, err == nil && seconds > 0
}

// resetSeconds takes the window's own reset header, or the shared one when
// the shared trio describes this window.
func resetSeconds(h http.Header, window string, l Limit) (int64, bool) {
	if reset, ok := headerInt(h, resetPrefix+window); ok {
		return reset, true
	}
	limit, hasLimit := headerInt(h, tightestLimit)
	remaining, hasRemaining := headerInt(h, tightestRemaining)
	if !hasLimit || !hasRemaining || limit != l.Max || remaining != l.Remaining {
		return 0, false
	}
	return headerInt(h, tightestReset)
}

// headerInt reads a non-negative whole number; anything else counts as absent.
func headerInt(h http.Header, name string) (int64, bool) {
	n, err := strconv.ParseInt(strings.TrimSpace(h.Get(name)), 10, 64)
	return n, err == nil && n >= 0
}

// formatWait prints a wait in its two largest units: "45s", "1m 29s",
// "3h 17m", "5d 13h".
func formatWait(d time.Duration) string {
	secs := int64(d.Round(time.Second) / time.Second)
	days, hours, minutes, seconds := secs/86400, secs%86400/3600, secs%3600/60, secs%60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, minutes)
	case minutes > 0:
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
