package content

import "testing"

func testLocales() []Locale {
	return []Locale{
		{Code: "en", Label: "English", Enabled: true, SortOrder: 1, IsDefault: true},
		{Code: "zh-CN", Label: "简体中文", Enabled: true, SortOrder: 2},
		{Code: "fr", Label: "Français", Enabled: false, SortOrder: 3},
	}
}

func TestResolveLocale(t *testing.T) {
	locales := testLocales()
	cases := []struct {
		name      string
		requested string
		want      string
	}{
		{name: "empty uses default", requested: "", want: "en"},
		{name: "unknown falls back", requested: "xx-XX", want: "en"},
		{name: "disabled falls back", requested: "fr", want: "en"},
		{name: "enabled exact", requested: "zh-CN", want: "zh-CN"},
		{name: "case insensitive", requested: "ZH-cn", want: "zh-CN"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveLocale(tt.requested, locales); got != tt.want {
				t.Fatalf("ResolveLocale(%q) = %q, want %q", tt.requested, got, tt.want)
			}
		})
	}
}

func TestDefaultLocaleFallsBackToFirstEnabled(t *testing.T) {
	locales := []Locale{
		{Code: "fr", Enabled: false},
		{Code: "zh-CN", Enabled: true},
	}
	if got := DefaultLocale(locales); got != "zh-CN" {
		t.Fatalf("DefaultLocale = %q, want zh-CN", got)
	}
	if got := DefaultLocale(nil); got != "" {
		t.Fatalf("DefaultLocale(nil) = %q, want empty", got)
	}
}

func TestEnabledLocalesFiltersDisabled(t *testing.T) {
	enabled := EnabledLocales(testLocales())
	if len(enabled) != 2 {
		t.Fatalf("EnabledLocales len = %d, want 2", len(enabled))
	}
}

func TestPickTranslationFallsBackToDefault(t *testing.T) {
	m := map[string]string{"en": "English", "zh-CN": "中文"}
	if got, ok := PickTranslation(m, "zh-CN", "en"); !ok || got != "中文" {
		t.Fatalf("pick zh-CN = %q ok=%v", got, ok)
	}
	if got, ok := PickTranslation(m, "de", "en"); !ok || got != "English" {
		t.Fatalf("pick fallback = %q ok=%v", got, ok)
	}
	if _, ok := PickTranslation(map[string]string{}, "de", "en"); ok {
		t.Fatal("empty map must not resolve")
	}
}

func TestMergePayload(t *testing.T) {
	base := map[string]any{"cta_url": "/register", "title": "base"}
	override := map[string]any{"title": "localized"}
	got := MergePayload(base, override)
	if got["title"] != "localized" || got["cta_url"] != "/register" {
		t.Fatalf("MergePayload = %#v", got)
	}
	if base["title"] != "base" {
		t.Fatal("MergePayload mutated base")
	}
}

func TestSanitizeMarkdownStripsExecutableHTML(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "script block", in: "hello <script>alert(1)</script> world", want: "hello  world"},
		{name: "raw tag", in: "<img src=x onerror=alert(1)>text", want: "text"},
		{name: "javascript link", in: "[click](javascript:alert(1))", want: "[click](#)"},
		{name: "comment", in: "a<!-- secret -->b", want: "ab"},
		{name: "plain markdown kept", in: "# Title\n\n- item", want: "# Title\n\n- item"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := SanitizeMarkdown(tt.in); got != tt.want {
				t.Fatalf("SanitizeMarkdown(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidateContentURL(t *testing.T) {
	valid := []string{"", "/features", "https://example.com", "http://example.com/x"}
	for _, raw := range valid {
		if err := ValidateContentURL(raw, false); err != nil {
			t.Fatalf("ValidateContentURL(%q) = %v, want nil", raw, err)
		}
	}
	if err := ValidateContentURL("mailto:hi@example.com", true); err != nil {
		t.Fatalf("mailto contact = %v, want nil", err)
	}
	invalid := []string{"javascript:alert(1)", "data:text/html,x", "//evil.com", "ftp://example.com", "mailto:hi@example.com"}
	for _, raw := range invalid {
		if err := ValidateContentURL(raw, false); err == nil {
			t.Fatalf("ValidateContentURL(%q) = nil, want error", raw)
		}
	}
}

func TestNormalizeMoney(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "", want: "0.00"},
		{in: "29", want: "29.00"},
		{in: "29.5", want: "29.50"},
		{in: "0", want: "0.00"},
		{in: "007.9", want: "7.90"},
		{in: "1.234", wantErr: true},
		{in: "-1", wantErr: true},
		{in: "abc", wantErr: true},
	}
	for _, tt := range cases {
		got, err := NormalizeMoney(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("NormalizeMoney(%q) = %q, want error", tt.in, got)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Fatalf("NormalizeMoney(%q) = %q, %v; want %q", tt.in, got, err, tt.want)
		}
	}
}
