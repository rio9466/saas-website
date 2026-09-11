package contentsvc

import (
	"context"
	"strings"

	"github.com/rio9466/easy-admin/server/internal/domain/content"
)

func validateOptionalURL(field, raw string, allowContact bool) error {
	if err := content.ValidateContentURL(raw, allowContact); err != nil {
		return validationError(field + " must be a site-relative path or an http/https URL")
	}
	return nil
}

// enabledLocaleSet returns the set of enabled locale codes.
func enabledLocaleSet(locales []content.Locale) map[string]bool {
	out := make(map[string]bool, len(locales))
	for _, l := range locales {
		if l.Enabled {
			out[l.Code] = true
		}
	}
	return out
}

// configuredLocaleSet returns every configured locale code (enabled or not).
func configuredLocaleSet(locales []content.Locale) map[string]bool {
	out := make(map[string]bool, len(locales))
	for _, l := range locales {
		out[l.Code] = true
	}
	return out
}

func (s *Service) validateTranslationLocales(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	locales, err := s.content.ListLocales(ctx)
	if err != nil {
		return mapError(err)
	}
	configured := configuredLocaleSet(locales)
	for _, key := range keys {
		if !configured[key] {
			return validationError("unknown locale: " + key)
		}
	}
	return nil
}

func mapKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func sanitizeMarkdown(raw string) string {
	return content.SanitizeMarkdown(strings.TrimSpace(raw))
}
