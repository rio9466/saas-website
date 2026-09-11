package handler

import (
	"errors"
	"strings"

	"github.com/rio9466/easy-admin/server/internal/domain/content"
	contentsvc "github.com/rio9466/easy-admin/server/internal/service/contentsvc"
)

// errInvalidID reports a malformed decimal ID in a request body.
var errInvalidID = errors.New("invalid id")

func parseOptionalID(raw *string) (*int64, bool, error) {
	if raw == nil {
		return nil, false, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, true, nil
	}
	id, err := parseID(trimmed)
	if err != nil || id <= 0 {
		return nil, true, errInvalidID
	}
	return &id, true, nil
}

// --- request DTOs ----------------------------------------------------------

type adminSiteSettingsRequest struct {
	SiteName             string                                 `json:"site_name"`
	LogoURL              string                                 `json:"logo_url"`
	LogoDarkURL          string                                 `json:"logo_dark_url"`
	FaviconURL           string                                 `json:"favicon_url"`
	ContactEmail         string                                 `json:"contact_email"`
	ContactPhone         string                                 `json:"contact_phone"`
	ContactAddress       string                                 `json:"contact_address"`
	SocialLinks          []content.SocialLink                   `json:"social_links"`
	SEODefaultOGImageURL string                                 `json:"seo_default_og_image_url"`
	DefaultLocale        string                                 `json:"default_locale"`
	Translations         map[string]adminSiteSettingTranslation `json:"translations"`
	Version              int                                    `json:"version"`
}

type adminSiteSettingTranslation struct {
	Tagline               string `json:"tagline"`
	FooterText            string `json:"footer_text"`
	SEODefaultTitle       string `json:"seo_default_title"`
	SEODefaultDescription string `json:"seo_default_description"`
	ICPRecord             string `json:"icp_record"`
}

func (r adminSiteSettingsRequest) toInput() (contentsvc.SiteSettingsInput, error) {
	in := contentsvc.SiteSettingsInput{
		SiteName:             r.SiteName,
		LogoURL:              r.LogoURL,
		LogoDarkURL:          r.LogoDarkURL,
		FaviconURL:           r.FaviconURL,
		ContactEmail:         r.ContactEmail,
		ContactPhone:         r.ContactPhone,
		ContactAddress:       r.ContactAddress,
		SocialLinks:          r.SocialLinks,
		SEODefaultOGImageURL: r.SEODefaultOGImageURL,
		DefaultLocale:        r.DefaultLocale,
		Version:              r.Version,
	}
	if r.Translations != nil {
		in.Translations = make(map[string]content.SiteSettingTranslation, len(r.Translations))
		for locale, t := range r.Translations {
			in.Translations[locale] = content.SiteSettingTranslation{
				Tagline:               t.Tagline,
				FooterText:            t.FooterText,
				SEODefaultTitle:       t.SEODefaultTitle,
				SEODefaultDescription: t.SEODefaultDescription,
				ICPRecord:             t.ICPRecord,
			}
		}
	}
	return in, nil
}

type adminNavigationItemRequest struct {
	Placement    *string                         `json:"placement"`
	ParentID     *string                         `json:"parent_id"`
	URL          *string                         `json:"url"`
	Target       *string                         `json:"target"`
	SortOrder    *int                            `json:"sort_order"`
	Visible      *bool                           `json:"visible"`
	Translations map[string]adminNavigationLabel `json:"translations"`
}

type adminNavigationLabel struct {
	Label string `json:"label"`
}

func (r adminNavigationItemRequest) toInput() (contentsvc.NavigationItemInput, error) {
	in := contentsvc.NavigationItemInput{
		Placement: r.Placement,
		URL:       r.URL,
		Target:    r.Target,
		SortOrder: r.SortOrder,
		Visible:   r.Visible,
	}
	parentID, set, err := parseOptionalID(r.ParentID)
	if err != nil {
		return in, err
	}
	in.ParentID = parentID
	in.ParentSet = set
	if r.Translations != nil {
		in.Translations = make(map[string]string, len(r.Translations))
		for locale, t := range r.Translations {
			in.Translations[locale] = t.Label
		}
	}
	return in, nil
}

type adminHomeSectionRequest struct {
	Type         *string                   `json:"type"`
	SortOrder    *int                      `json:"sort_order"`
	Published    *bool                     `json:"published"`
	Payload      map[string]any            `json:"data"`
	Translations map[string]map[string]any `json:"translations"`
}

func (r adminHomeSectionRequest) toInput() contentsvc.HomeSectionInput {
	return contentsvc.HomeSectionInput{
		Type:         r.Type,
		SortOrder:    r.SortOrder,
		Published:    r.Published,
		Payload:      r.Payload,
		Translations: r.Translations,
	}
}

type adminFeatureRequest struct {
	Icon         *string                            `json:"icon"`
	SortOrder    *int                               `json:"sort_order"`
	Published    *bool                              `json:"published"`
	Translations map[string]adminFeatureTranslation `json:"translations"`
}

type adminFeatureTranslation struct {
	Title    string `json:"title"`
	Summary  string `json:"summary"`
	BodyMD   string `json:"body_md"`
	ImageURL string `json:"image_url"`
}

func (r adminFeatureRequest) toInput() contentsvc.FeatureInput {
	in := contentsvc.FeatureInput{
		Icon:      r.Icon,
		SortOrder: r.SortOrder,
		Published: r.Published,
	}
	if r.Translations != nil {
		in.Translations = make(map[string]content.FeatureTranslation, len(r.Translations))
		for locale, t := range r.Translations {
			in.Translations[locale] = content.FeatureTranslation{
				Title:    t.Title,
				Summary:  t.Summary,
				BodyMD:   t.BodyMD,
				ImageURL: t.ImageURL,
			}
		}
	}
	return in
}

type adminPricingPlanRequest struct {
	Code         *string                                `json:"code"`
	MonthlyPrice *string                                `json:"monthly_price"`
	YearlyPrice  *string                                `json:"yearly_price"`
	Currency     *string                                `json:"currency"`
	Highlighted  *bool                                  `json:"highlighted"`
	SortOrder    *int                                   `json:"sort_order"`
	Visible      *bool                                  `json:"visible"`
	Translations map[string]adminPricingPlanTranslation `json:"translations"`
	Features     map[string][]string                    `json:"features"`
}

type adminPricingPlanTranslation struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CTALabel    string `json:"cta_label"`
	CTAURL      string `json:"cta_url"`
}

func (r adminPricingPlanRequest) toInput() contentsvc.PricingPlanInput {
	in := contentsvc.PricingPlanInput{
		Code:         r.Code,
		MonthlyPrice: r.MonthlyPrice,
		YearlyPrice:  r.YearlyPrice,
		Currency:     r.Currency,
		Highlighted:  r.Highlighted,
		SortOrder:    r.SortOrder,
		Visible:      r.Visible,
		Features:     r.Features,
	}
	if r.Translations != nil {
		in.Translations = make(map[string]content.PricingPlanTranslation, len(r.Translations))
		for locale, t := range r.Translations {
			in.Translations[locale] = content.PricingPlanTranslation{
				Name:        t.Name,
				Description: t.Description,
				CTALabel:    t.CTALabel,
				CTAURL:      t.CTAURL,
			}
		}
	}
	return in
}

type adminPageRequest struct {
	Slug         *string                         `json:"slug"`
	Published    *bool                           `json:"published"`
	SortOrder    *int                            `json:"sort_order"`
	Translations map[string]adminPageTranslation `json:"translations"`
}

type adminPageTranslation struct {
	Title          string `json:"title"`
	BodyMD         string `json:"body_md"`
	SEOTitle       string `json:"seo_title"`
	SEODescription string `json:"seo_description"`
}

func (r adminPageRequest) toInput() contentsvc.PageInput {
	in := contentsvc.PageInput{
		Slug:      r.Slug,
		Published: r.Published,
		SortOrder: r.SortOrder,
	}
	if r.Translations != nil {
		in.Translations = make(map[string]content.PageTranslation, len(r.Translations))
		for locale, t := range r.Translations {
			in.Translations[locale] = content.PageTranslation{
				Title:          t.Title,
				BodyMD:         t.BodyMD,
				SEOTitle:       t.SEOTitle,
				SEODescription: t.SEODescription,
			}
		}
	}
	return in
}

type adminDocCategoryRequest struct {
	Slug         *string                         `json:"slug"`
	SortOrder    *int                            `json:"sort_order"`
	Translations map[string]adminDocCategoryName `json:"translations"`
}

type adminDocCategoryName struct {
	Name string `json:"name"`
}

func (r adminDocCategoryRequest) toInput() contentsvc.DocCategoryInput {
	in := contentsvc.DocCategoryInput{Slug: r.Slug, SortOrder: r.SortOrder}
	if r.Translations != nil {
		in.Translations = make(map[string]string, len(r.Translations))
		for locale, t := range r.Translations {
			in.Translations[locale] = t.Name
		}
	}
	return in
}

type adminDocArticleRequest struct {
	Slug         *string                               `json:"slug"`
	CategoryID   *string                               `json:"category_id"`
	SortOrder    *int                                  `json:"sort_order"`
	Published    *bool                                 `json:"published"`
	Translations map[string]adminDocArticleTranslation `json:"translations"`
}

type adminDocArticleTranslation struct {
	Title          string `json:"title"`
	BodyMD         string `json:"body_md"`
	SEOTitle       string `json:"seo_title"`
	SEODescription string `json:"seo_description"`
}

func (r adminDocArticleRequest) toInput() (contentsvc.DocArticleInput, error) {
	in := contentsvc.DocArticleInput{
		Slug:      r.Slug,
		SortOrder: r.SortOrder,
		Published: r.Published,
	}
	if r.CategoryID != nil {
		id, err := parseID(strings.TrimSpace(*r.CategoryID))
		if err != nil || id <= 0 {
			return in, errInvalidID
		}
		in.CategoryID = &id
	}
	if r.Translations != nil {
		in.Translations = make(map[string]content.DocArticleTranslation, len(r.Translations))
		for locale, t := range r.Translations {
			in.Translations[locale] = content.DocArticleTranslation{
				Title:          t.Title,
				BodyMD:         t.BodyMD,
				SEOTitle:       t.SEOTitle,
				SEODescription: t.SEODescription,
			}
		}
	}
	return in, nil
}

// --- response DTOs ---------------------------------------------------------

type adminSiteSettingsData struct {
	SiteName             string                                 `json:"site_name"`
	LogoURL              string                                 `json:"logo_url"`
	LogoDarkURL          string                                 `json:"logo_dark_url"`
	FaviconURL           string                                 `json:"favicon_url"`
	ContactEmail         string                                 `json:"contact_email"`
	ContactPhone         string                                 `json:"contact_phone"`
	ContactAddress       string                                 `json:"contact_address"`
	SocialLinks          []content.SocialLink                   `json:"social_links"`
	SEODefaultOGImageURL string                                 `json:"seo_default_og_image_url"`
	DefaultLocale        string                                 `json:"default_locale"`
	Translations         map[string]adminSiteSettingTranslation `json:"translations"`
	Version              int                                    `json:"version"`
	UpdatedBy            string                                 `json:"updated_by"`
	UpdatedAt            string                                 `json:"updated_at"`
}

func toSiteSettingsData(s *content.SiteSettings) adminSiteSettingsData {
	if s == nil {
		return adminSiteSettingsData{SocialLinks: []content.SocialLink{}, Translations: map[string]adminSiteSettingTranslation{}}
	}
	translations := make(map[string]adminSiteSettingTranslation, len(s.Translations))
	for locale, t := range s.Translations {
		translations[locale] = adminSiteSettingTranslation{
			Tagline:               t.Tagline,
			FooterText:            t.FooterText,
			SEODefaultTitle:       t.SEODefaultTitle,
			SEODefaultDescription: t.SEODefaultDescription,
			ICPRecord:             t.ICPRecord,
		}
	}
	return adminSiteSettingsData{
		SiteName:             s.SiteName,
		LogoURL:              s.LogoURL,
		LogoDarkURL:          s.LogoDarkURL,
		FaviconURL:           s.FaviconURL,
		ContactEmail:         s.ContactEmail,
		ContactPhone:         s.ContactPhone,
		ContactAddress:       s.ContactAddress,
		SocialLinks:          append([]content.SocialLink{}, s.SocialLinks...),
		SEODefaultOGImageURL: s.SEODefaultOGImageURL,
		DefaultLocale:        s.DefaultLocale,
		Translations:         translations,
		Version:              s.Version,
		UpdatedBy:            idString(s.UpdatedBy),
		UpdatedAt:            formatTime(s.UpdatedAt),
	}
}

type adminNavigationItemData struct {
	ID           string                          `json:"id"`
	Placement    string                          `json:"placement"`
	ParentID     *string                         `json:"parent_id"`
	URL          string                          `json:"url"`
	Target       string                          `json:"target"`
	SortOrder    int                             `json:"sort_order"`
	Visible      bool                            `json:"visible"`
	Translations map[string]adminNavigationLabel `json:"translations"`
	CreatedAt    string                          `json:"created_at"`
	UpdatedAt    string                          `json:"updated_at"`
}

func toNavigationItemData(item *content.NavigationItem) adminNavigationItemData {
	if item == nil {
		return adminNavigationItemData{Translations: map[string]adminNavigationLabel{}}
	}
	translations := make(map[string]adminNavigationLabel, len(item.Translations))
	for locale, label := range item.Translations {
		translations[locale] = adminNavigationLabel{Label: label}
	}
	var parentID *string
	if item.ParentID != nil {
		s := idString(*item.ParentID)
		parentID = &s
	}
	return adminNavigationItemData{
		ID:           idString(item.ID),
		Placement:    item.Placement,
		ParentID:     parentID,
		URL:          item.URL,
		Target:       item.Target,
		SortOrder:    item.SortOrder,
		Visible:      item.Visible,
		Translations: translations,
		CreatedAt:    formatTime(item.CreatedAt),
		UpdatedAt:    formatTime(item.UpdatedAt),
	}
}

func toNavigationItemList(items []content.NavigationItem) []adminNavigationItemData {
	out := make([]adminNavigationItemData, 0, len(items))
	for i := range items {
		out = append(out, toNavigationItemData(&items[i]))
	}
	return out
}

type adminHomeSectionData struct {
	ID           string                    `json:"id"`
	Type         string                    `json:"type"`
	SortOrder    int                       `json:"sort_order"`
	Published    bool                      `json:"published"`
	Payload      map[string]any            `json:"data"`
	Translations map[string]map[string]any `json:"translations"`
	CreatedAt    string                    `json:"created_at"`
	UpdatedAt    string                    `json:"updated_at"`
}

func toHomeSectionData(section *content.HomeSection) adminHomeSectionData {
	if section == nil {
		return adminHomeSectionData{Payload: map[string]any{}, Translations: map[string]map[string]any{}}
	}
	payload := section.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	translations := section.Translations
	if translations == nil {
		translations = map[string]map[string]any{}
	}
	return adminHomeSectionData{
		ID:           idString(section.ID),
		Type:         section.Type,
		SortOrder:    section.SortOrder,
		Published:    section.Published,
		Payload:      payload,
		Translations: translations,
		CreatedAt:    formatTime(section.CreatedAt),
		UpdatedAt:    formatTime(section.UpdatedAt),
	}
}

func toHomeSectionList(items []content.HomeSection) []adminHomeSectionData {
	out := make([]adminHomeSectionData, 0, len(items))
	for i := range items {
		out = append(out, toHomeSectionData(&items[i]))
	}
	return out
}

type adminFeatureData struct {
	ID           string                             `json:"id"`
	Icon         string                             `json:"icon"`
	SortOrder    int                                `json:"sort_order"`
	Published    bool                               `json:"published"`
	Translations map[string]adminFeatureTranslation `json:"translations"`
	CreatedAt    string                             `json:"created_at"`
	UpdatedAt    string                             `json:"updated_at"`
}

func toFeatureData(feature *content.Feature) adminFeatureData {
	if feature == nil {
		return adminFeatureData{Translations: map[string]adminFeatureTranslation{}}
	}
	translations := make(map[string]adminFeatureTranslation, len(feature.Translations))
	for locale, t := range feature.Translations {
		translations[locale] = adminFeatureTranslation{Title: t.Title, Summary: t.Summary, BodyMD: t.BodyMD, ImageURL: t.ImageURL}
	}
	return adminFeatureData{
		ID:           idString(feature.ID),
		Icon:         feature.Icon,
		SortOrder:    feature.SortOrder,
		Published:    feature.Published,
		Translations: translations,
		CreatedAt:    formatTime(feature.CreatedAt),
		UpdatedAt:    formatTime(feature.UpdatedAt),
	}
}

func toFeatureList(items []content.Feature) []adminFeatureData {
	out := make([]adminFeatureData, 0, len(items))
	for i := range items {
		out = append(out, toFeatureData(&items[i]))
	}
	return out
}

type adminPricingPlanData struct {
	ID           string                                 `json:"id"`
	Code         string                                 `json:"code"`
	MonthlyPrice string                                 `json:"monthly_price"`
	YearlyPrice  string                                 `json:"yearly_price"`
	Currency     string                                 `json:"currency"`
	Highlighted  bool                                   `json:"highlighted"`
	SortOrder    int                                    `json:"sort_order"`
	Visible      bool                                   `json:"visible"`
	Translations map[string]adminPricingPlanTranslation `json:"translations"`
	Features     map[string][]string                    `json:"features"`
	CreatedAt    string                                 `json:"created_at"`
	UpdatedAt    string                                 `json:"updated_at"`
}

func toPricingPlanData(plan *content.PricingPlan) adminPricingPlanData {
	if plan == nil {
		return adminPricingPlanData{Translations: map[string]adminPricingPlanTranslation{}, Features: map[string][]string{}}
	}
	translations := make(map[string]adminPricingPlanTranslation, len(plan.Translations))
	for locale, t := range plan.Translations {
		translations[locale] = adminPricingPlanTranslation{Name: t.Name, Description: t.Description, CTALabel: t.CTALabel, CTAURL: t.CTAURL}
	}
	features := make(map[string][]string, len(plan.Features))
	for locale, list := range plan.Features {
		features[locale] = append([]string{}, list...)
	}
	return adminPricingPlanData{
		ID:           idString(plan.ID),
		Code:         plan.Code,
		MonthlyPrice: plan.MonthlyPrice,
		YearlyPrice:  plan.YearlyPrice,
		Currency:     plan.Currency,
		Highlighted:  plan.Highlighted,
		SortOrder:    plan.SortOrder,
		Visible:      plan.Visible,
		Translations: translations,
		Features:     features,
		CreatedAt:    formatTime(plan.CreatedAt),
		UpdatedAt:    formatTime(plan.UpdatedAt),
	}
}

func toPricingPlanList(items []content.PricingPlan) []adminPricingPlanData {
	out := make([]adminPricingPlanData, 0, len(items))
	for i := range items {
		out = append(out, toPricingPlanData(&items[i]))
	}
	return out
}

type adminPageData struct {
	ID           string                          `json:"id"`
	Slug         string                          `json:"slug"`
	Published    bool                            `json:"published"`
	SortOrder    int                             `json:"sort_order"`
	Translations map[string]adminPageTranslation `json:"translations"`
	CreatedAt    string                          `json:"created_at"`
	UpdatedAt    string                          `json:"updated_at"`
}

func toPageData(page *content.Page) adminPageData {
	if page == nil {
		return adminPageData{Translations: map[string]adminPageTranslation{}}
	}
	translations := make(map[string]adminPageTranslation, len(page.Translations))
	for locale, t := range page.Translations {
		translations[locale] = adminPageTranslation{Title: t.Title, BodyMD: t.BodyMD, SEOTitle: t.SEOTitle, SEODescription: t.SEODescription}
	}
	return adminPageData{
		ID:           idString(page.ID),
		Slug:         page.Slug,
		Published:    page.Published,
		SortOrder:    page.SortOrder,
		Translations: translations,
		CreatedAt:    formatTime(page.CreatedAt),
		UpdatedAt:    formatTime(page.UpdatedAt),
	}
}

func toPageList(items []content.Page) []adminPageData {
	out := make([]adminPageData, 0, len(items))
	for i := range items {
		out = append(out, toPageData(&items[i]))
	}
	return out
}

type adminDocCategoryData struct {
	ID           string                          `json:"id"`
	Slug         string                          `json:"slug"`
	SortOrder    int                             `json:"sort_order"`
	Translations map[string]adminDocCategoryName `json:"translations"`
	CreatedAt    string                          `json:"created_at"`
	UpdatedAt    string                          `json:"updated_at"`
}

func toDocCategoryData(category *content.DocCategory) adminDocCategoryData {
	if category == nil {
		return adminDocCategoryData{Translations: map[string]adminDocCategoryName{}}
	}
	translations := make(map[string]adminDocCategoryName, len(category.Translations))
	for locale, name := range category.Translations {
		translations[locale] = adminDocCategoryName{Name: name}
	}
	return adminDocCategoryData{
		ID:           idString(category.ID),
		Slug:         category.Slug,
		SortOrder:    category.SortOrder,
		Translations: translations,
		CreatedAt:    formatTime(category.CreatedAt),
		UpdatedAt:    formatTime(category.UpdatedAt),
	}
}

func toDocCategoryList(items []content.DocCategory) []adminDocCategoryData {
	out := make([]adminDocCategoryData, 0, len(items))
	for i := range items {
		out = append(out, toDocCategoryData(&items[i]))
	}
	return out
}

type adminDocArticleData struct {
	ID           string                                `json:"id"`
	Slug         string                                `json:"slug"`
	CategoryID   string                                `json:"category_id"`
	SortOrder    int                                   `json:"sort_order"`
	Published    bool                                  `json:"published"`
	Translations map[string]adminDocArticleTranslation `json:"translations"`
	CreatedAt    string                                `json:"created_at"`
	UpdatedAt    string                                `json:"updated_at"`
}

func toDocArticleData(article *content.DocArticle) adminDocArticleData {
	if article == nil {
		return adminDocArticleData{Translations: map[string]adminDocArticleTranslation{}}
	}
	translations := make(map[string]adminDocArticleTranslation, len(article.Translations))
	for locale, t := range article.Translations {
		translations[locale] = adminDocArticleTranslation{Title: t.Title, BodyMD: t.BodyMD, SEOTitle: t.SEOTitle, SEODescription: t.SEODescription}
	}
	return adminDocArticleData{
		ID:           idString(article.ID),
		Slug:         article.Slug,
		CategoryID:   idString(article.CategoryID),
		SortOrder:    article.SortOrder,
		Published:    article.Published,
		Translations: translations,
		CreatedAt:    formatTime(article.CreatedAt),
		UpdatedAt:    formatTime(article.UpdatedAt),
	}
}

func toDocArticleList(items []content.DocArticle) []adminDocArticleData {
	out := make([]adminDocArticleData, 0, len(items))
	for i := range items {
		out = append(out, toDocArticleData(&items[i]))
	}
	return out
}
