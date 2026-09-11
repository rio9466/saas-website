package content

import "strings"

// DefaultLocale returns the enabled default locale code, falling back to the
// first enabled locale, or "" when no locale is enabled.
func DefaultLocale(locales []Locale) string {
	for _, l := range locales {
		if l.Enabled && l.IsDefault {
			return l.Code
		}
	}
	for _, l := range locales {
		if l.Enabled {
			return l.Code
		}
	}
	return ""
}

// ResolveLocale returns the effective locale for a request. An empty, unknown,
// or disabled requested locale falls back to the default locale; code matching
// is case-insensitive.
func ResolveLocale(requested string, locales []Locale) string {
	r := strings.TrimSpace(requested)
	def := DefaultLocale(locales)
	if r == "" {
		return def
	}
	for _, l := range locales {
		if strings.EqualFold(l.Code, r) {
			if l.Enabled {
				return l.Code
			}
			return def
		}
	}
	return def
}

// EnabledLocales returns enabled locales ordered by sort_order then code.
func EnabledLocales(locales []Locale) []Locale {
	out := make([]Locale, 0, len(locales))
	for _, l := range locales {
		if l.Enabled {
			out = append(out, l)
		}
	}
	return out
}

// PickTranslation returns the translation for effective, falling back to the
// default locale when it is present, then to the effective locale's zero value.
func PickTranslation[T any](m map[string]T, effective, fallback string) (T, bool) {
	if m == nil {
		var zero T
		return zero, false
	}
	if v, ok := m[effective]; ok {
		return v, true
	}
	if fallback != "" && fallback != effective {
		if v, ok := m[fallback]; ok {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// MergePayload returns base with override's keys applied on top. Neither input
// is mutated.
func MergePayload(base, override map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(override))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}
