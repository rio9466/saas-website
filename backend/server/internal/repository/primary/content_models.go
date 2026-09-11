package primary

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// jsonbRaw is a raw JSON document persisted in a jsonb column. It encodes as a
// JSON text value and scans PostgreSQL's jsonb text representation.
type jsonbRaw []byte

// Value implements driver.Valuer. A nil document is stored as an empty object.
func (j jsonbRaw) Value() (driver.Value, error) {
	if len(j) == 0 {
		return "{}", nil
	}
	return string(j), nil
}

// Scan implements sql.Scanner for text/binary jsonb output.
func (j *jsonbRaw) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*j = nil
	case []byte:
		*j = append((*j)[:0], v...)
	case string:
		*j = append((*j)[:0], v...)
	default:
		return fmt.Errorf("cannot scan %T into jsonbRaw", src)
	}
	return nil
}

type localeModel struct {
	Code      string    `gorm:"column:code;primaryKey"`
	Label     string    `gorm:"column:label"`
	Enabled   bool      `gorm:"column:enabled"`
	SortOrder int       `gorm:"column:sort_order"`
	IsDefault bool      `gorm:"column:is_default"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (localeModel) TableName() string { return "supported_locales" }

type siteSettingsModel struct {
	ID                   int64     `gorm:"column:id;primaryKey"`
	SiteName             string    `gorm:"column:site_name"`
	LogoURL              string    `gorm:"column:logo_url"`
	LogoDarkURL          string    `gorm:"column:logo_dark_url"`
	FaviconURL           string    `gorm:"column:favicon_url"`
	ContactEmail         string    `gorm:"column:contact_email"`
	ContactPhone         string    `gorm:"column:contact_phone"`
	ContactAddress       string    `gorm:"column:contact_address"`
	SocialLinks          jsonbRaw  `gorm:"column:social_links;type:jsonb"`
	SEODefaultOGImageURL string    `gorm:"column:seo_default_og_image_url"`
	DefaultLocale        string    `gorm:"column:default_locale"`
	Version              int       `gorm:"column:version"`
	UpdatedBy            int64     `gorm:"column:updated_by"`
	UpdatedAt            time.Time `gorm:"column:updated_at"`
}

func (siteSettingsModel) TableName() string { return "site_settings" }

type siteSettingTranslationModel struct {
	SiteSettingID         int64     `gorm:"column:site_setting_id;primaryKey"`
	Locale                string    `gorm:"column:locale;primaryKey"`
	Tagline               string    `gorm:"column:tagline"`
	FooterText            string    `gorm:"column:footer_text"`
	SEODefaultTitle       string    `gorm:"column:seo_default_title"`
	SEODefaultDescription string    `gorm:"column:seo_default_description"`
	ICPRecord             string    `gorm:"column:icp_record"`
	CreatedAt             time.Time `gorm:"column:created_at"`
	UpdatedAt             time.Time `gorm:"column:updated_at"`
}

func (siteSettingTranslationModel) TableName() string { return "site_setting_translations" }

type navigationItemModel struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Placement string    `gorm:"column:placement"`
	ParentID  *int64    `gorm:"column:parent_id"`
	URL       string    `gorm:"column:url"`
	Target    string    `gorm:"column:target"`
	SortOrder int       `gorm:"column:sort_order"`
	Visible   bool      `gorm:"column:visible"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (navigationItemModel) TableName() string { return "navigation_items" }

type navigationItemTranslationModel struct {
	NavigationItemID int64     `gorm:"column:navigation_item_id;primaryKey"`
	Locale           string    `gorm:"column:locale;primaryKey"`
	Label            string    `gorm:"column:label"`
	CreatedAt        time.Time `gorm:"column:created_at"`
	UpdatedAt        time.Time `gorm:"column:updated_at"`
}

func (navigationItemTranslationModel) TableName() string { return "navigation_item_translations" }

type homeSectionModel struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Type      string    `gorm:"column:type"`
	SortOrder int       `gorm:"column:sort_order"`
	Published bool      `gorm:"column:published"`
	Payload   jsonbRaw  `gorm:"column:payload;type:jsonb"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (homeSectionModel) TableName() string { return "home_sections" }

type homeSectionTranslationModel struct {
	HomeSectionID int64     `gorm:"column:home_section_id;primaryKey"`
	Locale        string    `gorm:"column:locale;primaryKey"`
	Payload       jsonbRaw  `gorm:"column:payload;type:jsonb"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (homeSectionTranslationModel) TableName() string { return "home_section_translations" }

type featureModel struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Icon      string    `gorm:"column:icon"`
	SortOrder int       `gorm:"column:sort_order"`
	Published bool      `gorm:"column:published"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (featureModel) TableName() string { return "features" }

type featureTranslationModel struct {
	FeatureID int64     `gorm:"column:feature_id;primaryKey"`
	Locale    string    `gorm:"column:locale;primaryKey"`
	Title     string    `gorm:"column:title"`
	Summary   string    `gorm:"column:summary"`
	BodyMD    string    `gorm:"column:body_md"`
	ImageURL  string    `gorm:"column:image_url"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (featureTranslationModel) TableName() string { return "feature_translations" }

type pricingPlanModel struct {
	ID           int64          `gorm:"column:id;primaryKey"`
	Code         string         `gorm:"column:code"`
	MonthlyPrice pgtype.Numeric `gorm:"column:monthly_price;type:numeric(20,2)"`
	YearlyPrice  pgtype.Numeric `gorm:"column:yearly_price;type:numeric(20,2)"`
	Currency     string         `gorm:"column:currency"`
	Highlighted  bool           `gorm:"column:highlighted"`
	SortOrder    int            `gorm:"column:sort_order"`
	Visible      bool           `gorm:"column:visible"`
	CreatedAt    time.Time      `gorm:"column:created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at"`
}

func (pricingPlanModel) TableName() string { return "pricing_plans" }

type pricingPlanTranslationModel struct {
	PricingPlanID int64     `gorm:"column:pricing_plan_id;primaryKey"`
	Locale        string    `gorm:"column:locale;primaryKey"`
	Name          string    `gorm:"column:name"`
	Description   string    `gorm:"column:description"`
	CTALabel      string    `gorm:"column:cta_label"`
	CTAURL        string    `gorm:"column:cta_url"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (pricingPlanTranslationModel) TableName() string { return "pricing_plan_translations" }

type pricingPlanFeatureModel struct {
	ID            int64     `gorm:"column:id;primaryKey"`
	PricingPlanID int64     `gorm:"column:pricing_plan_id"`
	Locale        string    `gorm:"column:locale"`
	SortOrder     int       `gorm:"column:sort_order"`
	Text          string    `gorm:"column:text"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func (pricingPlanFeatureModel) TableName() string { return "pricing_plan_features" }

type pageModel struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Slug      string    `gorm:"column:slug"`
	Published bool      `gorm:"column:published"`
	SortOrder int       `gorm:"column:sort_order"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (pageModel) TableName() string { return "pages" }

type pageTranslationModel struct {
	PageID         int64     `gorm:"column:page_id;primaryKey"`
	Locale         string    `gorm:"column:locale;primaryKey"`
	Title          string    `gorm:"column:title"`
	BodyMD         string    `gorm:"column:body_md"`
	SEOTitle       string    `gorm:"column:seo_title"`
	SEODescription string    `gorm:"column:seo_description"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (pageTranslationModel) TableName() string { return "page_translations" }

type docCategoryModel struct {
	ID        int64     `gorm:"column:id;primaryKey"`
	Slug      string    `gorm:"column:slug"`
	SortOrder int       `gorm:"column:sort_order"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (docCategoryModel) TableName() string { return "doc_categories" }

type docCategoryTranslationModel struct {
	DocCategoryID int64     `gorm:"column:doc_category_id;primaryKey"`
	Locale        string    `gorm:"column:locale;primaryKey"`
	Name          string    `gorm:"column:name"`
	CreatedAt     time.Time `gorm:"column:created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at"`
}

func (docCategoryTranslationModel) TableName() string { return "doc_category_translations" }

type docArticleModel struct {
	ID         int64     `gorm:"column:id;primaryKey"`
	Slug       string    `gorm:"column:slug"`
	CategoryID int64     `gorm:"column:category_id"`
	SortOrder  int       `gorm:"column:sort_order"`
	Published  bool      `gorm:"column:published"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (docArticleModel) TableName() string { return "doc_articles" }

type docArticleTranslationModel struct {
	DocArticleID   int64     `gorm:"column:doc_article_id;primaryKey"`
	Locale         string    `gorm:"column:locale;primaryKey"`
	Title          string    `gorm:"column:title"`
	BodyMD         string    `gorm:"column:body_md"`
	SEOTitle       string    `gorm:"column:seo_title"`
	SEODescription string    `gorm:"column:seo_description"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (docArticleTranslationModel) TableName() string { return "doc_article_translations" }
