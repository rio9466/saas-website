// Package analytics holds the page-view entity, traffic-source resolution, and
// the range windows used by the administrator analytics overview. It has no
// persistence or transport dependencies.
package analytics

import (
	"errors"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

// Contract limits (§4.10).
const (
	// MaxPathLength bounds a reported site path in characters.
	MaxPathLength = 512
	// MaxLocaleLength bounds a reported locale; longer values are dropped.
	MaxLocaleLength = 35
)

// SourceDirect is the traffic source used when the referrer is empty or not
// http(s). No domain classification is performed (§4.10).
const SourceDirect = "direct"

// SourcesLimit caps the overview source list, ordered by count descending
// (§6.3).
const SourcesLimit = 50

// Overview range names (§6.3).
const (
	RangeToday = "today"
	Range7d    = "7d"
	Range30d   = "30d"
)

// ErrInvalidRange reports an unsupported overview range value.
var ErrInvalidRange = errors.New("invalid analytics range")

// PageViewInput is a raw public page-view report as bound from JSON.
type PageViewInput struct {
	Path     string `json:"path"`
	Referrer string `json:"referrer"`
	Locale   string `json:"locale"`
}

// PageView is one persisted public page-view report.
type PageView struct {
	ID        int64
	Path      string
	Source    string
	Locale    string
	CreatedAt time.Time
}

// SourceCount is one aggregated traffic source and its page-view count.
type SourceCount struct {
	Source string
	Count  int64
}

// Overview is the aggregated analytics window returned to administrators.
type Overview struct {
	Range       string
	PV          int64
	SourceCount int64
	Sources     []SourceCount
}

// Window is a resolved overview range: its name and inclusive lower bound.
type Window struct {
	Name  string
	Start time.Time
}

// ValidPath reports whether path is a site path: it starts with "/" and its
// character length is within the contract cap (§4.10).
func ValidPath(path string) bool {
	if !strings.HasPrefix(path, "/") {
		return false
	}
	return utf8.RuneCountInString(path) <= MaxPathLength
}

// SourceFromReferrer resolves the traffic source for a referrer value:
// "direct" when empty or not http(s), otherwise the referrer host with
// subdomains preserved and port/path removed. No domain classification.
func SourceFromReferrer(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return SourceDirect
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return SourceDirect
	}
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return SourceDirect
	}
	host := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if host == "" {
		return SourceDirect
	}
	return host
}

// NormalizeLocale trims a reported locale and drops values beyond the cap so a
// hostile payload cannot bloat storage.
func NormalizeLocale(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if utf8.RuneCountInString(trimmed) > MaxLocaleLength {
		return ""
	}
	return trimmed
}

// ResolveWindow maps a range query value to its window. An empty value defaults
// to "today" (the server's local day); 7d/30d are rolling windows ending now
// that include the current day. Unknown values return ErrInvalidRange.
func ResolveWindow(raw string, now time.Time) (Window, error) {
	switch strings.TrimSpace(raw) {
	case "", RangeToday:
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		return Window{Name: RangeToday, Start: start}, nil
	case Range7d:
		return Window{Name: Range7d, Start: now.Add(-7 * 24 * time.Hour)}, nil
	case Range30d:
		return Window{Name: Range30d, Start: now.Add(-30 * 24 * time.Hour)}, nil
	default:
		return Window{}, ErrInvalidRange
	}
}
