package analytics

import (
	"strings"
	"testing"
	"time"
)

func TestValidPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		path string
		want bool
	}{
		{"root", "/", true},
		{"plain", "/features", true},
		{"with query", "/pricing?plan=pro", true},
		{"empty", "", false},
		{"no leading slash", "features", false},
		{"relative", "./features", false},
		{"absolute url", "https://example.com/features", false},
		{"max length", "/" + strings.Repeat("a", MaxPathLength-1), true},
		{"too long", "/" + strings.Repeat("a", MaxPathLength), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ValidPath(tc.path); got != tc.want {
				t.Fatalf("ValidPath(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestSourceFromReferrer(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		referrer string
		want     string
	}{
		{"empty", "", "direct"},
		{"spaces", "   ", "direct"},
		{"https host with subdomain", "https://www.baidu.com/s?wd=acme", "www.baidu.com"},
		{"http host with port", "http://example.com:8443/path", "example.com"},
		{"uppercase host", "HTTPS://GitHub.COM/acme/repo", "github.com"},
		{"no scheme", "www.baidu.com/s", "direct"},
		{"protocol relative", "//www.baidu.com/s", "direct"},
		{"mailto", "mailto:ops@example.com", "direct"},
		{"javascript", "javascript:alert(1)", "direct"},
		{"http without host", "https://", "direct"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := SourceFromReferrer(tc.referrer); got != tc.want {
				t.Fatalf("SourceFromReferrer(%q) = %q, want %q", tc.referrer, got, tc.want)
			}
		})
	}
}

func TestNormalizeLocale(t *testing.T) {
	t.Parallel()
	if got := NormalizeLocale("  zh-CN  "); got != "zh-CN" {
		t.Fatalf("NormalizeLocale trimmed = %q, want zh-CN", got)
	}
	if got := NormalizeLocale(strings.Repeat("x", MaxLocaleLength)); got == "" {
		t.Fatal("locale at the cap must be kept")
	}
	if got := NormalizeLocale(strings.Repeat("x", MaxLocaleLength+1)); got != "" {
		t.Fatalf("over-long locale = %q, want dropped", got)
	}
}

func TestResolveWindow(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 9, 11, 15, 30, 0, 0, time.UTC)

	today, err := ResolveWindow("", now)
	if err != nil {
		t.Fatalf("default range: %v", err)
	}
	if today.Name != RangeToday {
		t.Fatalf("default name = %q, want %q", today.Name, RangeToday)
	}
	wantToday := time.Date(2026, 9, 11, 0, 0, 0, 0, time.UTC)
	if !today.Start.Equal(wantToday) {
		t.Fatalf("today start = %s, want %s", today.Start, wantToday)
	}

	for name, want := range map[string]time.Time{
		Range7d:  now.Add(-7 * 24 * time.Hour),
		Range30d: now.Add(-30 * 24 * time.Hour),
	} {
		window, err := ResolveWindow(name, now)
		if err != nil {
			t.Fatalf("range %s: %v", name, err)
		}
		if window.Name != name || !window.Start.Equal(want) {
			t.Fatalf("range %s = %+v, want start %s", name, window, want)
		}
	}

	if _, err := ResolveWindow("90d", now); err == nil {
		t.Fatal("unsupported range must return an error")
	}
}
