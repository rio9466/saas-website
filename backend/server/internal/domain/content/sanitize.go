package content

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// ErrInvalidURL reports a URL outside the accepted protocol/link whitelist.
var ErrInvalidURL = errors.New("invalid url")

// dangerousTags are raw HTML elements that must never survive storage.
var dangerousTags = []string{"script", "style", "iframe", "object", "embed", "form", "link", "meta", "base", "svg", "math", "template", "applet"}

// dangerousBlockREs matches dangerous elements with content (and their
// self-closing form). Each tag is compiled separately because Go's RE2 engine
// has no backreferences.
var dangerousBlockREs = compileDangerousBlocks()

func compileDangerousBlocks() []*regexp.Regexp {
	out := make([]*regexp.Regexp, 0, len(dangerousTags)*2)
	for _, tag := range dangerousTags {
		out = append(out,
			regexp.MustCompile(`(?is)<\s*`+tag+`\b[^>]*>.*?<\s*/\s*`+tag+`\s*>`),
			regexp.MustCompile(`(?is)<\s*`+tag+`\b[^>]*/?>`),
		)
	}
	return out
}

// anyTagRE matches any remaining raw HTML tag, including comments.
var anyTagRE = regexp.MustCompile(`(?is)<!--.*?-->|<[^>]*>`)

// markdownLinkRE captures the destination of a Markdown link or image,
// allowing one level of nested parentheses in the destination.
var markdownLinkRE = regexp.MustCompile(`(!?\[[^\]]*\]\()\s*((?:[^()\s]|\([^()]*\))*)\s*\)`)

// unsafeSchemeRE matches scripting/data URL schemes anywhere in the text.
var unsafeSchemeRE = regexp.MustCompile(`(?i)\b(javascript|vbscript|data)\s*:`)

// SanitizeMarkdown strips raw HTML and scripting URL schemes from stored
// Markdown. Output stays valid Markdown for the frontend's whitelist renderer;
// it never contains executable HTML.
func SanitizeMarkdown(raw string) string {
	if raw == "" {
		return ""
	}
	out := raw
	for _, re := range dangerousBlockREs {
		out = re.ReplaceAllString(out, "")
	}
	out = anyTagRE.ReplaceAllString(out, "")
	out = markdownLinkRE.ReplaceAllStringFunc(out, func(match string) string {
		groups := markdownLinkRE.FindStringSubmatch(match)
		if len(groups) != 3 {
			return match
		}
		if isUnsafeDestination(groups[2]) {
			return groups[1] + "#)"
		}
		return match
	})
	out = unsafeSchemeRE.ReplaceAllString(out, "")
	return out
}

func isUnsafeDestination(dest string) bool {
	trimmed := strings.TrimSpace(strings.Trim(dest, "<>\"'"))
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	return strings.HasPrefix(lower, "javascript:") ||
		strings.HasPrefix(lower, "vbscript:") ||
		strings.HasPrefix(lower, "data:")
}

// ValidateContentURL accepts an optional URL restricted to the protocol
// whitelist: site-relative paths, http/https, and (when allowContact) mailto
// and tel links. An empty value is allowed (optional fields).
func ValidateContentURL(raw string, allowContact bool) error {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil
	}
	if strings.HasPrefix(s, "/") {
		if strings.HasPrefix(s, "//") {
			return ErrInvalidURL
		}
		return nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return ErrInvalidURL
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		if u.Host == "" {
			return ErrInvalidURL
		}
		return nil
	case "mailto", "tel":
		if allowContact && u.Opaque != "" {
			return nil
		}
		return ErrInvalidURL
	default:
		return ErrInvalidURL
	}
}
