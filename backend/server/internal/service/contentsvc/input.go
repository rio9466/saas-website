package contentsvc

import "github.com/rio9466/easy-admin/server/internal/domain/content"

// SiteSettingsInput is the full-replace site-settings write.
type SiteSettingsInput struct {
	SiteName             string
	LogoURL              string
	LogoDarkURL          string
	FaviconURL           string
	ContactEmail         string
	ContactPhone         string
	ContactAddress       string
	SocialLinks          []content.SocialLink
	SEODefaultOGImageURL string
	DefaultLocale        string
	Translations         map[string]content.SiteSettingTranslation
	Version              int
}

// NavigationItemInput is a create/partial-update navigation write.
type NavigationItemInput struct {
	Placement    *string
	ParentID     *int64
	ParentSet    bool
	URL          *string
	Target       *string
	SortOrder    *int
	Visible      *bool
	Translations map[string]string
}

// HomeSectionInput is a create/partial-update home-section write.
type HomeSectionInput struct {
	Type         *string
	SortOrder    *int
	Published    *bool
	Payload      map[string]any
	Translations map[string]map[string]any
}

// FeatureInput is a create/partial-update feature write.
type FeatureInput struct {
	Icon         *string
	SortOrder    *int
	Published    *bool
	Translations map[string]content.FeatureTranslation
}

// PricingPlanInput is a create/partial-update pricing-plan write.
type PricingPlanInput struct {
	Code         *string
	MonthlyPrice *string
	YearlyPrice  *string
	Currency     *string
	Highlighted  *bool
	SortOrder    *int
	Visible      *bool
	Translations map[string]content.PricingPlanTranslation
	Features     map[string][]string
}

// PageInput is a create/partial-update page write.
type PageInput struct {
	Slug         *string
	Published    *bool
	SortOrder    *int
	Translations map[string]content.PageTranslation
}

// DocCategoryInput is a create/partial-update documentation-category write.
type DocCategoryInput struct {
	Slug         *string
	SortOrder    *int
	Translations map[string]string
}

// DocArticleInput is a create/partial-update documentation-article write.
type DocArticleInput struct {
	Slug         *string
	CategoryID   *int64
	SortOrder    *int
	Published    *bool
	Translations map[string]content.DocArticleTranslation
}
