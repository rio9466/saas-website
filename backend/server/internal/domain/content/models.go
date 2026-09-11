// Package content holds the site-content entities, their invariants, locale
// resolution/fallback, and the sanitization rules applied to public output.
package content

import "time"

// Placement values for navigation items.
const (
	PlacementHeader = "header"
	PlacementFooter = "footer"
)

// Link target values for navigation items.
const (
	TargetSelf  = "_self"
	TargetBlank = "_blank"
)

// Known home-section types. Unknown types are stored and returned as-is; the
// frontend must ignore unknown types (forward compatibility).
const (
	SectionHero       = "hero"
	SectionFeatures   = "features"
	SectionScreenshot = "screenshot"
	SectionStats      = "stats"
	SectionCTA        = "cta"
)

// Locale is a configured site language.
type Locale struct {
	Code      string
	Label     string
	Enabled   bool
	SortOrder int
	IsDefault bool
}

// SocialLink is one social profile entry in site settings.
type SocialLink struct {
	Platform string `json:"platform"`
	URL      string `json:"url"`
}

// SiteSettings is the singleton branding/SEO/contact configuration.
type SiteSettings struct {
	SiteName             string
	LogoURL              string
	LogoDarkURL          string
	FaviconURL           string
	ContactEmail         string
	ContactPhone         string
	ContactAddress       string
	SocialLinks          []SocialLink
	SEODefaultOGImageURL string
	DefaultLocale        string
	Version              int
	UpdatedBy            int64
	UpdatedAt            time.Time
	Translations         map[string]SiteSettingTranslation
}

// SiteSettingTranslation is the per-locale site-settings overlay.
type SiteSettingTranslation struct {
	Tagline               string
	FooterText            string
	SEODefaultTitle       string
	SEODefaultDescription string
	ICPRecord             string
}

// NavigationItem is a header/footer menu entry.
type NavigationItem struct {
	ID           int64
	Placement    string
	ParentID     *int64
	URL          string
	Target       string
	SortOrder    int
	Visible      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Translations map[string]string // locale -> label
}

// HomeSection is one configurable home-page block. Payload is the base data;
// Translations holds per-locale overrides merged over it for public reads.
type HomeSection struct {
	ID           int64
	Type         string
	SortOrder    int
	Published    bool
	Payload      map[string]any
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Translations map[string]map[string]any
}

// Feature is one feature/marketing card.
type Feature struct {
	ID           int64
	Icon         string
	SortOrder    int
	Published    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Translations map[string]FeatureTranslation
}

// FeatureTranslation is the per-locale feature copy.
type FeatureTranslation struct {
	Title    string
	Summary  string
	BodyMD   string
	ImageURL string
}

// PricingPlan is a pricing tier. Prices are canonical two-decimal strings.
type PricingPlan struct {
	ID           int64
	Code         string
	MonthlyPrice string
	YearlyPrice  string
	Currency     string
	Highlighted  bool
	SortOrder    int
	Visible      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Translations map[string]PricingPlanTranslation
	Features     map[string][]string // locale -> bullet list, ordered
}

// PricingPlanTranslation is the per-locale plan copy.
type PricingPlanTranslation struct {
	Name        string
	Description string
	CTALabel    string
	CTAURL      string
}

// Page is a static content page (about, privacy, ...).
type Page struct {
	ID           int64
	Slug         string
	Published    bool
	SortOrder    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Translations map[string]PageTranslation
}

// PageTranslation is the per-locale page copy.
type PageTranslation struct {
	Title          string
	BodyMD         string
	SEOTitle       string
	SEODescription string
}

// DocCategory groups documentation articles.
type DocCategory struct {
	ID           int64
	Slug         string
	SortOrder    int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Translations map[string]string // locale -> name
}

// DocArticle is one documentation article.
type DocArticle struct {
	ID           int64
	Slug         string
	CategoryID   int64
	SortOrder    int
	Published    bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Translations map[string]DocArticleTranslation
}

// DocArticleTranslation is the per-locale article copy.
type DocArticleTranslation struct {
	Title          string
	BodyMD         string
	SEOTitle       string
	SEODescription string
}
