package content

import "time"

// FormatTime renders a timestamp as UTC RFC3339, or "" for the zero time.
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// PublicSettingsView is the content-owned part of GET /api/v1/public/settings.
// The platform/login fields remain owned by the user service.
type PublicSettingsView struct {
	SiteName              string             `json:"site_name"`
	LogoURL               string             `json:"logo_url"`
	LogoDarkURL           string             `json:"logo_dark_url"`
	FaviconURL            string             `json:"favicon_url"`
	Tagline               string             `json:"tagline"`
	FooterText            string             `json:"footer_text"`
	ICPRecord             string             `json:"icp_record"`
	ContactEmail          string             `json:"contact_email"`
	ContactPhone          string             `json:"contact_phone"`
	ContactAddress        string             `json:"contact_address"`
	SocialLinks           []SocialLink       `json:"social_links"`
	SEODefaultTitle       string             `json:"seo_default_title"`
	SEODefaultDescription string             `json:"seo_default_description"`
	SEODefaultOGImageURL  string             `json:"seo_default_og_image_url"`
	DefaultLocale         string             `json:"default_locale"`
	Locales               []LocaleOptionView `json:"locales"`
}

// LocaleOptionView is one enabled language option.
type LocaleOptionView struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// PublicNavigationView is GET /api/v1/public/navigation.
type PublicNavigationView struct {
	Locale string                     `json:"locale"`
	Items  []PublicNavigationItemView `json:"items"`
}

// PublicNavigationItemView is one navigation entry with nested children.
type PublicNavigationItemView struct {
	ID        string                     `json:"id"`
	Label     string                     `json:"label"`
	URL       string                     `json:"url"`
	Target    string                     `json:"target"`
	SortOrder int                        `json:"sort_order"`
	Children  []PublicNavigationItemView `json:"children"`
}

// PublicHomeView is GET /api/v1/public/home.
type PublicHomeView struct {
	Locale   string                  `json:"locale"`
	Sections []PublicHomeSectionView `json:"sections"`
}

// PublicHomeSectionView is one home section; Data is type-specific.
type PublicHomeSectionView struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	SortOrder int            `json:"sort_order"`
	Data      map[string]any `json:"data"`
}

// PublicFeaturesView is GET /api/v1/public/features.
type PublicFeaturesView struct {
	Locale string              `json:"locale"`
	Items  []PublicFeatureView `json:"items"`
}

// PublicFeatureView is one published feature card.
type PublicFeatureView struct {
	ID        string `json:"id"`
	Icon      string `json:"icon"`
	Title     string `json:"title"`
	Summary   string `json:"summary"`
	BodyMD    string `json:"body_md"`
	ImageURL  string `json:"image_url"`
	SortOrder int    `json:"sort_order"`
}

// PublicPricingView is GET /api/v1/public/pricing.
type PublicPricingView struct {
	Locale   string           `json:"locale"`
	Currency string           `json:"currency"`
	Plans    []PublicPlanView `json:"plans"`
}

// PublicPlanView is one visible pricing plan.
type PublicPlanView struct {
	ID           string   `json:"id"`
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	MonthlyPrice string   `json:"monthly_price"`
	YearlyPrice  string   `json:"yearly_price"`
	Currency     string   `json:"currency"`
	Highlighted  bool     `json:"highlighted"`
	CTALabel     string   `json:"cta_label"`
	CTAURL       string   `json:"cta_url"`
	Features     []string `json:"features"`
	SortOrder    int      `json:"sort_order"`
}

// PublicPageListView is GET /api/v1/public/pages.
type PublicPageListView struct {
	Locale string                  `json:"locale"`
	Items  []PublicPageSummaryView `json:"items"`
}

// PublicPageSummaryView is one published page in the list.
type PublicPageSummaryView struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	UpdatedAt string `json:"updated_at"`
}

// PublicPageView is GET /api/v1/public/pages/{slug}.
type PublicPageView struct {
	Locale         string `json:"locale"`
	Slug           string `json:"slug"`
	Title          string `json:"title"`
	BodyMD         string `json:"body_md"`
	SEOTitle       string `json:"seo_title"`
	SEODescription string `json:"seo_description"`
	UpdatedAt      string `json:"updated_at"`
}

// PublicDocListView is GET /api/v1/public/docs.
type PublicDocListView struct {
	Locale     string                  `json:"locale"`
	Categories []PublicDocCategoryView `json:"categories"`
}

// PublicDocCategoryView is one documentation category with its articles.
type PublicDocCategoryView struct {
	ID        string                        `json:"id"`
	Slug      string                        `json:"slug"`
	Name      string                        `json:"name"`
	SortOrder int                           `json:"sort_order"`
	Articles  []PublicDocArticleSummaryView `json:"articles"`
}

// PublicDocArticleSummaryView is one article in a category listing.
type PublicDocArticleSummaryView struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	SortOrder int    `json:"sort_order"`
	UpdatedAt string `json:"updated_at"`
}

// PublicDocArticleView is GET /api/v1/public/docs/{slug}.
type PublicDocArticleView struct {
	Locale         string               `json:"locale"`
	ID             string               `json:"id"`
	Slug           string               `json:"slug"`
	Title          string               `json:"title"`
	BodyMD         string               `json:"body_md"`
	Category       PublicDocCategoryRef `json:"category"`
	Prev           *PublicDocRef        `json:"prev"`
	Next           *PublicDocRef        `json:"next"`
	SEOTitle       string               `json:"seo_title"`
	SEODescription string               `json:"seo_description"`
	UpdatedAt      string               `json:"updated_at"`
}

// PublicDocCategoryRef is the category embedded in an article.
type PublicDocCategoryRef struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// PublicDocRef is a prev/next article reference.
type PublicDocRef struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}
