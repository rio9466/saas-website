package contentsvc

import (
	"context"
	"strings"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/content"
)

func validPlacement(v string) bool {
	return v == content.PlacementHeader || v == content.PlacementFooter
}

func validTarget(v string) bool {
	return v == content.TargetSelf || v == content.TargetBlank
}

func boolOrDefault(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

func validateSlug(field, slug string) error {
	if slug == "" {
		return validationError(field + " is required")
	}
	if len(slug) > 200 {
		return validationError(field + " is too long")
	}
	for _, r := range slug {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_', r == '/', r == '.', r == '~':
		default:
			return validationError(field + " contains invalid characters")
		}
	}
	return nil
}

// --- site settings ---------------------------------------------------------

// GetSiteSettings returns the singleton with all translations.
func (s *Service) GetSiteSettings(ctx context.Context, actor Actor) (*content.SiteSettings, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionRead); err != nil {
		return nil, err
	}
	settings, err := s.content.GetSiteSettings(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	return settings, nil
}

// UpdateSiteSettings replaces the singleton with the optimistic-lock version.
func (s *Service) UpdateSiteSettings(ctx context.Context, actor Actor, in SiteSettingsInput) (*content.SiteSettings, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return nil, err
	}
	if in.Version < 1 {
		return nil, validationError("settings version is required")
	}
	name := strings.TrimSpace(in.SiteName)
	if name == "" {
		return nil, validationError("site_name is required")
	}
	for field, value := range map[string]string{
		"logo_url":                 in.LogoURL,
		"logo_dark_url":            in.LogoDarkURL,
		"favicon_url":              in.FaviconURL,
		"seo_default_og_image_url": in.SEODefaultOGImageURL,
	} {
		if err := validateOptionalURL(field, value, false); err != nil {
			return nil, err
		}
	}
	social := make([]content.SocialLink, 0, len(in.SocialLinks))
	for _, link := range in.SocialLinks {
		platform := strings.TrimSpace(link.Platform)
		if platform == "" {
			return nil, validationError("social link platform is required")
		}
		if err := validateOptionalURL("social_links.url", link.URL, false); err != nil {
			return nil, err
		}
		if strings.TrimSpace(link.URL) == "" {
			return nil, validationError("social link url is required")
		}
		social = append(social, content.SocialLink{Platform: platform, URL: strings.TrimSpace(link.URL)})
	}

	locales, err := s.content.ListLocales(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	defaultLocale := strings.TrimSpace(in.DefaultLocale)
	if defaultLocale == "" {
		defaultLocale = content.DefaultLocale(locales)
	}
	if !enabledLocaleSet(locales)[defaultLocale] {
		return nil, validationError("default_locale must be an enabled locale")
	}
	if err := s.validateTranslationLocales(ctx, mapKeys(in.Translations)); err != nil {
		return nil, err
	}

	settings := &content.SiteSettings{
		SiteName:             name,
		LogoURL:              strings.TrimSpace(in.LogoURL),
		LogoDarkURL:          strings.TrimSpace(in.LogoDarkURL),
		FaviconURL:           strings.TrimSpace(in.FaviconURL),
		ContactEmail:         strings.TrimSpace(in.ContactEmail),
		ContactPhone:         strings.TrimSpace(in.ContactPhone),
		ContactAddress:       strings.TrimSpace(in.ContactAddress),
		SocialLinks:          social,
		SEODefaultOGImageURL: strings.TrimSpace(in.SEODefaultOGImageURL),
		DefaultLocale:        defaultLocale,
		UpdatedBy:            actor.AdminID,
	}
	settings.Translations = make(map[string]content.SiteSettingTranslation, len(in.Translations))
	for locale, t := range in.Translations {
		settings.Translations[locale] = content.SiteSettingTranslation{
			Tagline:               t.Tagline,
			FooterText:            t.FooterText,
			SEODefaultTitle:       t.SEODefaultTitle,
			SEODefaultDescription: t.SEODefaultDescription,
			ICPRecord:             t.ICPRecord,
		}
	}

	details := map[string]any{"default_locale": defaultLocale}
	if err := s.withPendingAudit(ctx, actor, ActionSiteSettingsUpdate, ResourceSiteSettings, "1", details, func() error {
		return s.content.UpdateSiteSettings(ctx, settings, in.Version)
	}); err != nil {
		return nil, err
	}
	return s.content.GetSiteSettings(ctx)
}

// --- navigation ------------------------------------------------------------

// ListNavigationItems returns a paginated navigation page.
func (s *Service) ListNavigationItems(ctx context.Context, actor Actor, page, pageSize int, placement string) (adminauth.Page[content.NavigationItem], error) {
	if err := s.requirePermission(ctx, actor, content.PermissionRead); err != nil {
		return adminauth.Page[content.NavigationItem]{}, err
	}
	items, total, err := s.content.ListNavigation(ctx, strings.TrimSpace(placement), page, pageSize)
	if err != nil {
		return adminauth.Page[content.NavigationItem]{}, mapError(err)
	}
	return adminauth.Page[content.NavigationItem]{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// CreateNavigationItem inserts a navigation item.
func (s *Service) CreateNavigationItem(ctx context.Context, actor Actor, in NavigationItemInput) (*content.NavigationItem, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return nil, err
	}
	placement := derefString(in.Placement)
	if !validPlacement(placement) {
		return nil, validationError("placement must be header or footer")
	}
	target := derefString(in.Target)
	if target == "" {
		target = content.TargetSelf
	}
	if !validTarget(target) {
		return nil, validationError("target must be _self or _blank")
	}
	url := derefString(in.URL)
	if err := validateOptionalURL("url", url, true); err != nil {
		return nil, err
	}
	if err := s.validateTranslationLocales(ctx, mapKeys(in.Translations)); err != nil {
		return nil, err
	}
	item := &content.NavigationItem{
		Placement:    placement,
		ParentID:     in.ParentID,
		URL:          url,
		Target:       target,
		SortOrder:    derefInt(in.SortOrder),
		Visible:      boolOrDefault(in.Visible, true),
		Translations: cloneStringMap(in.Translations),
	}
	details := map[string]any{"placement": placement, "visible": item.Visible, "sort_order": item.SortOrder}
	id, err := s.createAudited(ctx, actor, ActionNavigationCreate, ResourceNavigation, details, func() (int64, error) {
		if createErr := s.content.CreateNavigation(ctx, item); createErr != nil {
			return 0, createErr
		}
		return item.ID, nil
	})
	if err != nil {
		return nil, err
	}
	return s.content.GetNavigation(ctx, id)
}

// UpdateNavigationItem applies a partial navigation update.
func (s *Service) UpdateNavigationItem(ctx context.Context, actor Actor, id int64, in NavigationItemInput) (*content.NavigationItem, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return nil, err
	}
	item, err := s.content.GetNavigation(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	if in.Placement != nil {
		v := derefString(in.Placement)
		if !validPlacement(v) {
			return nil, validationError("placement must be header or footer")
		}
		item.Placement = v
	}
	if in.ParentSet {
		item.ParentID = in.ParentID
	}
	if in.URL != nil {
		v := derefString(in.URL)
		if err := validateOptionalURL("url", v, true); err != nil {
			return nil, err
		}
		item.URL = v
	}
	if in.Target != nil {
		v := derefString(in.Target)
		if !validTarget(v) {
			return nil, validationError("target must be _self or _blank")
		}
		item.Target = v
	}
	if in.SortOrder != nil {
		item.SortOrder = *in.SortOrder
	}
	if in.Visible != nil {
		item.Visible = *in.Visible
	}
	if in.Translations != nil {
		if err := s.validateTranslationLocales(ctx, mapKeys(in.Translations)); err != nil {
			return nil, err
		}
		item.Translations = cloneStringMap(in.Translations)
	}
	details := map[string]any{"placement": item.Placement, "visible": item.Visible, "sort_order": item.SortOrder}
	if err := s.withPendingAudit(ctx, actor, ActionNavigationUpdate, ResourceNavigation, idString(id), details, func() error {
		return s.content.UpdateNavigation(ctx, item)
	}); err != nil {
		return nil, err
	}
	return s.content.GetNavigation(ctx, id)
}

// DeleteNavigationItem removes a navigation item.
func (s *Service) DeleteNavigationItem(ctx context.Context, actor Actor, id int64) error {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return err
	}
	return s.withPendingAudit(ctx, actor, ActionNavigationDelete, ResourceNavigation, idString(id), nil, func() error {
		return s.content.DeleteNavigation(ctx, id)
	})
}

// --- home sections ---------------------------------------------------------

// ListHomeSections returns a paginated home-section page.
func (s *Service) ListHomeSections(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.HomeSection], error) {
	if err := s.requirePermission(ctx, actor, content.PermissionRead); err != nil {
		return adminauth.Page[content.HomeSection]{}, err
	}
	sections, total, err := s.content.ListHomeSections(ctx, page, pageSize)
	if err != nil {
		return adminauth.Page[content.HomeSection]{}, mapError(err)
	}
	return adminauth.Page[content.HomeSection]{Items: sections, Total: total, Page: page, PageSize: pageSize}, nil
}

// CreateHomeSection inserts a home section.
func (s *Service) CreateHomeSection(ctx context.Context, actor Actor, in HomeSectionInput) (*content.HomeSection, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return nil, err
	}
	sectionType := derefString(in.Type)
	if sectionType == "" {
		return nil, validationError("type is required")
	}
	if err := s.validateTranslationLocales(ctx, mapKeys(in.Translations)); err != nil {
		return nil, err
	}
	section := &content.HomeSection{
		Type:         sectionType,
		SortOrder:    derefInt(in.SortOrder),
		Published:    derefBool(in.Published),
		Payload:      clonePayload(in.Payload),
		Translations: clonePayloadMap(in.Translations),
	}
	details := map[string]any{"section_type": sectionType, "published": section.Published, "sort_order": section.SortOrder}
	id, err := s.createAudited(ctx, actor, ActionHomeSectionCreate, ResourceHomeSection, details, func() (int64, error) {
		if createErr := s.content.CreateHomeSection(ctx, section); createErr != nil {
			return 0, createErr
		}
		return section.ID, nil
	})
	if err != nil {
		return nil, err
	}
	return s.content.GetHomeSection(ctx, id)
}

// UpdateHomeSection applies a partial home-section update.
func (s *Service) UpdateHomeSection(ctx context.Context, actor Actor, id int64, in HomeSectionInput) (*content.HomeSection, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return nil, err
	}
	section, err := s.content.GetHomeSection(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	if in.Type != nil {
		v := derefString(in.Type)
		if v == "" {
			return nil, validationError("type is required")
		}
		section.Type = v
	}
	if in.SortOrder != nil {
		section.SortOrder = *in.SortOrder
	}
	if in.Published != nil {
		section.Published = *in.Published
	}
	if in.Payload != nil {
		section.Payload = clonePayload(in.Payload)
	}
	if in.Translations != nil {
		if err := s.validateTranslationLocales(ctx, mapKeys(in.Translations)); err != nil {
			return nil, err
		}
		section.Translations = clonePayloadMap(in.Translations)
	}
	details := map[string]any{"section_type": section.Type, "published": section.Published, "sort_order": section.SortOrder}
	if err := s.withPendingAudit(ctx, actor, ActionHomeSectionUpdate, ResourceHomeSection, idString(id), details, func() error {
		return s.content.UpdateHomeSection(ctx, section)
	}); err != nil {
		return nil, err
	}
	return s.content.GetHomeSection(ctx, id)
}

// DeleteHomeSection removes a home section.
func (s *Service) DeleteHomeSection(ctx context.Context, actor Actor, id int64) error {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return err
	}
	return s.withPendingAudit(ctx, actor, ActionHomeSectionDelete, ResourceHomeSection, idString(id), nil, func() error {
		return s.content.DeleteHomeSection(ctx, id)
	})
}

// --- features --------------------------------------------------------------

// ListFeatures returns a paginated feature page.
func (s *Service) ListFeatures(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.Feature], error) {
	if err := s.requirePermission(ctx, actor, content.PermissionRead); err != nil {
		return adminauth.Page[content.Feature]{}, err
	}
	items, total, err := s.content.ListFeatures(ctx, page, pageSize)
	if err != nil {
		return adminauth.Page[content.Feature]{}, mapError(err)
	}
	return adminauth.Page[content.Feature]{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// CreateFeature inserts a feature.
func (s *Service) CreateFeature(ctx context.Context, actor Actor, in FeatureInput) (*content.Feature, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return nil, err
	}
	translations, err := sanitizeFeatureTranslations(in.Translations)
	if err != nil {
		return nil, err
	}
	if err := s.validateTranslationLocales(ctx, mapKeys(in.Translations)); err != nil {
		return nil, err
	}
	feature := &content.Feature{
		Icon:         derefString(in.Icon),
		SortOrder:    derefInt(in.SortOrder),
		Published:    derefBool(in.Published),
		Translations: translations,
	}
	details := map[string]any{"published": feature.Published, "sort_order": feature.SortOrder}
	id, err := s.createAudited(ctx, actor, ActionFeatureCreate, ResourceFeature, details, func() (int64, error) {
		if createErr := s.content.CreateFeature(ctx, feature); createErr != nil {
			return 0, createErr
		}
		return feature.ID, nil
	})
	if err != nil {
		return nil, err
	}
	return s.content.GetFeature(ctx, id)
}

// UpdateFeature applies a partial feature update.
func (s *Service) UpdateFeature(ctx context.Context, actor Actor, id int64, in FeatureInput) (*content.Feature, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return nil, err
	}
	feature, err := s.content.GetFeature(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	if in.Icon != nil {
		feature.Icon = derefString(in.Icon)
	}
	if in.SortOrder != nil {
		feature.SortOrder = *in.SortOrder
	}
	if in.Published != nil {
		feature.Published = *in.Published
	}
	if in.Translations != nil {
		translations, err := sanitizeFeatureTranslations(in.Translations)
		if err != nil {
			return nil, err
		}
		if err := s.validateTranslationLocales(ctx, mapKeys(in.Translations)); err != nil {
			return nil, err
		}
		feature.Translations = translations
	}
	details := map[string]any{"published": feature.Published, "sort_order": feature.SortOrder}
	if err := s.withPendingAudit(ctx, actor, ActionFeatureUpdate, ResourceFeature, idString(id), details, func() error {
		return s.content.UpdateFeature(ctx, feature)
	}); err != nil {
		return nil, err
	}
	return s.content.GetFeature(ctx, id)
}

// DeleteFeature removes a feature.
func (s *Service) DeleteFeature(ctx context.Context, actor Actor, id int64) error {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return err
	}
	return s.withPendingAudit(ctx, actor, ActionFeatureDelete, ResourceFeature, idString(id), nil, func() error {
		return s.content.DeleteFeature(ctx, id)
	})
}

// --- pricing plans ---------------------------------------------------------

// ListPricingPlans returns a paginated pricing-plan page.
func (s *Service) ListPricingPlans(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.PricingPlan], error) {
	if err := s.requirePermission(ctx, actor, content.PermissionRead); err != nil {
		return adminauth.Page[content.PricingPlan]{}, err
	}
	items, total, err := s.content.ListPricingPlans(ctx, page, pageSize)
	if err != nil {
		return adminauth.Page[content.PricingPlan]{}, mapError(err)
	}
	return adminauth.Page[content.PricingPlan]{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// CreatePricingPlan inserts a plan.
func (s *Service) CreatePricingPlan(ctx context.Context, actor Actor, in PricingPlanInput) (*content.PricingPlan, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return nil, err
	}
	plan, err := s.buildPricingPlan(ctx, nil, in, true)
	if err != nil {
		return nil, err
	}
	details := map[string]any{"code": plan.Code, "visible": plan.Visible, "highlighted": plan.Highlighted, "sort_order": plan.SortOrder}
	id, err := s.createAudited(ctx, actor, ActionPricingPlanCreate, ResourcePricingPlan, details, func() (int64, error) {
		if createErr := s.content.CreatePricingPlan(ctx, plan); createErr != nil {
			return 0, createErr
		}
		return plan.ID, nil
	})
	if err != nil {
		return nil, err
	}
	return s.content.GetPricingPlan(ctx, id)
}

// UpdatePricingPlan applies a partial plan update.
func (s *Service) UpdatePricingPlan(ctx context.Context, actor Actor, id int64, in PricingPlanInput) (*content.PricingPlan, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return nil, err
	}
	existing, err := s.content.GetPricingPlan(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	plan, err := s.buildPricingPlan(ctx, existing, in, false)
	if err != nil {
		return nil, err
	}
	details := map[string]any{"code": plan.Code, "visible": plan.Visible, "highlighted": plan.Highlighted, "sort_order": plan.SortOrder}
	if err := s.withPendingAudit(ctx, actor, ActionPricingPlanUpdate, ResourcePricingPlan, idString(id), details, func() error {
		return s.content.UpdatePricingPlan(ctx, plan)
	}); err != nil {
		return nil, err
	}
	return s.content.GetPricingPlan(ctx, id)
}

// DeletePricingPlan removes a plan.
func (s *Service) DeletePricingPlan(ctx context.Context, actor Actor, id int64) error {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return err
	}
	return s.withPendingAudit(ctx, actor, ActionPricingPlanDelete, ResourcePricingPlan, idString(id), nil, func() error {
		return s.content.DeletePricingPlan(ctx, id)
	})
}

func (s *Service) buildPricingPlan(ctx context.Context, existing *content.PricingPlan, in PricingPlanInput, create bool) (*content.PricingPlan, error) {
	plan := &content.PricingPlan{}
	if existing != nil {
		*plan = *existing
	}
	code := plan.Code
	if in.Code != nil {
		code = derefString(in.Code)
	}
	if create && code == "" {
		return nil, validationError("code is required")
	}
	if code != "" {
		plan.Code = code
	}
	if create && plan.Currency == "" {
		plan.Currency = "USD"
	}
	if in.Currency != nil {
		v := strings.TrimSpace(*in.Currency)
		if v == "" {
			return nil, validationError("currency is required")
		}
		plan.Currency = v
	}
	if in.MonthlyPrice != nil {
		money, err := content.NormalizeMoney(*in.MonthlyPrice)
		if err != nil {
			return nil, validationError("monthly_price must be a non-negative decimal")
		}
		plan.MonthlyPrice = money
	} else if create {
		plan.MonthlyPrice = "0.00"
	}
	if in.YearlyPrice != nil {
		money, err := content.NormalizeMoney(*in.YearlyPrice)
		if err != nil {
			return nil, validationError("yearly_price must be a non-negative decimal")
		}
		plan.YearlyPrice = money
	} else if create {
		plan.YearlyPrice = "0.00"
	}
	if in.Highlighted != nil {
		plan.Highlighted = *in.Highlighted
	}
	if in.SortOrder != nil {
		plan.SortOrder = *in.SortOrder
	}
	if in.Visible != nil {
		plan.Visible = *in.Visible
	} else if create {
		plan.Visible = true
	}
	if in.Translations != nil {
		plan.Translations = clonePlanTranslations(in.Translations)
	} else if create {
		plan.Translations = map[string]content.PricingPlanTranslation{}
	}
	if in.Features != nil {
		plan.Features = cloneFeatureLists(in.Features)
	} else if create {
		plan.Features = map[string][]string{}
	}
	if in.Translations != nil {
		for _, t := range in.Translations {
			if err := validateOptionalURL("cta_url", t.CTAURL, true); err != nil {
				return nil, err
			}
		}
	}
	if err := s.validateTranslationLocales(ctx, append(mapKeys(in.Translations), mapKeys(in.Features)...)); err != nil {
		return nil, err
	}
	return plan, nil
}

// --- pages -----------------------------------------------------------------

// ListPages returns a paginated page listing.
func (s *Service) ListPages(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.Page], error) {
	if err := s.requirePermission(ctx, actor, content.PermissionRead); err != nil {
		return adminauth.Page[content.Page]{}, err
	}
	items, total, err := s.content.ListPages(ctx, page, pageSize)
	if err != nil {
		return adminauth.Page[content.Page]{}, mapError(err)
	}
	return adminauth.Page[content.Page]{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// GetPage returns one page with all translations.
func (s *Service) GetPage(ctx context.Context, actor Actor, id int64) (*content.Page, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionRead); err != nil {
		return nil, err
	}
	page, err := s.content.GetPage(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return page, nil
}

// CreatePage inserts a page.
func (s *Service) CreatePage(ctx context.Context, actor Actor, in PageInput) (*content.Page, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return nil, err
	}
	slug := derefString(in.Slug)
	if err := validateSlug("slug", slug); err != nil {
		return nil, err
	}
	translations, err := sanitizePageTranslations(in.Translations)
	if err != nil {
		return nil, err
	}
	if err := s.validateTranslationLocales(ctx, mapKeys(in.Translations)); err != nil {
		return nil, err
	}
	page := &content.Page{
		Slug:         slug,
		Published:    derefBool(in.Published),
		SortOrder:    derefInt(in.SortOrder),
		Translations: translations,
	}
	details := map[string]any{"slug": slug, "published": page.Published, "sort_order": page.SortOrder}
	id, err := s.createAudited(ctx, actor, ActionPageCreate, ResourcePage, details, func() (int64, error) {
		if createErr := s.content.CreatePage(ctx, page); createErr != nil {
			return 0, createErr
		}
		return page.ID, nil
	})
	if err != nil {
		return nil, err
	}
	return s.content.GetPage(ctx, id)
}

// UpdatePage applies a partial page update.
func (s *Service) UpdatePage(ctx context.Context, actor Actor, id int64, in PageInput) (*content.Page, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return nil, err
	}
	page, err := s.content.GetPage(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	if in.Slug != nil {
		v := derefString(in.Slug)
		if err := validateSlug("slug", v); err != nil {
			return nil, err
		}
		page.Slug = v
	}
	if in.Published != nil {
		page.Published = *in.Published
	}
	if in.SortOrder != nil {
		page.SortOrder = *in.SortOrder
	}
	if in.Translations != nil {
		translations, err := sanitizePageTranslations(in.Translations)
		if err != nil {
			return nil, err
		}
		if err := s.validateTranslationLocales(ctx, mapKeys(in.Translations)); err != nil {
			return nil, err
		}
		page.Translations = translations
	}
	details := map[string]any{"slug": page.Slug, "published": page.Published, "sort_order": page.SortOrder}
	if err := s.withPendingAudit(ctx, actor, ActionPageUpdate, ResourcePage, idString(id), details, func() error {
		return s.content.UpdatePage(ctx, page)
	}); err != nil {
		return nil, err
	}
	return s.content.GetPage(ctx, id)
}

// DeletePage removes a page.
func (s *Service) DeletePage(ctx context.Context, actor Actor, id int64) error {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return err
	}
	return s.withPendingAudit(ctx, actor, ActionPageDelete, ResourcePage, idString(id), nil, func() error {
		return s.content.DeletePage(ctx, id)
	})
}

// --- documentation ---------------------------------------------------------

// ListDocCategories returns a paginated category listing.
func (s *Service) ListDocCategories(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.DocCategory], error) {
	if err := s.requirePermission(ctx, actor, content.PermissionRead); err != nil {
		return adminauth.Page[content.DocCategory]{}, err
	}
	items, total, err := s.content.ListDocCategories(ctx, page, pageSize)
	if err != nil {
		return adminauth.Page[content.DocCategory]{}, mapError(err)
	}
	return adminauth.Page[content.DocCategory]{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// CreateDocCategory inserts a documentation category.
func (s *Service) CreateDocCategory(ctx context.Context, actor Actor, in DocCategoryInput) (*content.DocCategory, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return nil, err
	}
	slug := derefString(in.Slug)
	if err := validateSlug("slug", slug); err != nil {
		return nil, err
	}
	if err := s.validateTranslationLocales(ctx, mapKeys(in.Translations)); err != nil {
		return nil, err
	}
	category := &content.DocCategory{
		Slug:         slug,
		SortOrder:    derefInt(in.SortOrder),
		Translations: cloneStringMap(in.Translations),
	}
	details := map[string]any{"slug": slug, "sort_order": category.SortOrder}
	id, err := s.createAudited(ctx, actor, ActionDocCategoryCreate, ResourceDocCategory, details, func() (int64, error) {
		if createErr := s.content.CreateDocCategory(ctx, category); createErr != nil {
			return 0, createErr
		}
		return category.ID, nil
	})
	if err != nil {
		return nil, err
	}
	return s.content.GetDocCategory(ctx, id)
}

// UpdateDocCategory applies a partial category update.
func (s *Service) UpdateDocCategory(ctx context.Context, actor Actor, id int64, in DocCategoryInput) (*content.DocCategory, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return nil, err
	}
	category, err := s.content.GetDocCategory(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	if in.Slug != nil {
		v := derefString(in.Slug)
		if err := validateSlug("slug", v); err != nil {
			return nil, err
		}
		category.Slug = v
	}
	if in.SortOrder != nil {
		category.SortOrder = *in.SortOrder
	}
	if in.Translations != nil {
		if err := s.validateTranslationLocales(ctx, mapKeys(in.Translations)); err != nil {
			return nil, err
		}
		category.Translations = cloneStringMap(in.Translations)
	}
	details := map[string]any{"slug": category.Slug, "sort_order": category.SortOrder}
	if err := s.withPendingAudit(ctx, actor, ActionDocCategoryUpdate, ResourceDocCategory, idString(id), details, func() error {
		return s.content.UpdateDocCategory(ctx, category)
	}); err != nil {
		return nil, err
	}
	return s.content.GetDocCategory(ctx, id)
}

// DeleteDocCategory removes a category; a category still holding articles is
// rejected as a conflict.
func (s *Service) DeleteDocCategory(ctx context.Context, actor Actor, id int64) error {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return err
	}
	return s.withPendingAudit(ctx, actor, ActionDocCategoryDelete, ResourceDocCategory, idString(id), nil, func() error {
		return s.content.DeleteDocCategory(ctx, id)
	})
}

// ListDocArticles returns a paginated article listing.
func (s *Service) ListDocArticles(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.DocArticle], error) {
	if err := s.requirePermission(ctx, actor, content.PermissionRead); err != nil {
		return adminauth.Page[content.DocArticle]{}, err
	}
	items, total, err := s.content.ListDocArticles(ctx, page, pageSize)
	if err != nil {
		return adminauth.Page[content.DocArticle]{}, mapError(err)
	}
	return adminauth.Page[content.DocArticle]{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// GetDocArticle returns one article with all translations.
func (s *Service) GetDocArticle(ctx context.Context, actor Actor, id int64) (*content.DocArticle, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionRead); err != nil {
		return nil, err
	}
	article, err := s.content.GetDocArticle(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	return article, nil
}

// CreateDocArticle inserts a documentation article.
func (s *Service) CreateDocArticle(ctx context.Context, actor Actor, in DocArticleInput) (*content.DocArticle, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return nil, err
	}
	slug := derefString(in.Slug)
	if err := validateSlug("slug", slug); err != nil {
		return nil, err
	}
	if in.CategoryID == nil || *in.CategoryID <= 0 {
		return nil, validationError("category_id is required")
	}
	translations, err := sanitizeArticleTranslations(in.Translations)
	if err != nil {
		return nil, err
	}
	if err := s.validateTranslationLocales(ctx, mapKeys(in.Translations)); err != nil {
		return nil, err
	}
	article := &content.DocArticle{
		Slug:         slug,
		CategoryID:   *in.CategoryID,
		SortOrder:    derefInt(in.SortOrder),
		Published:    derefBool(in.Published),
		Translations: translations,
	}
	details := map[string]any{"slug": slug, "published": article.Published, "sort_order": article.SortOrder, "category_id": idString(article.CategoryID)}
	id, err := s.createAudited(ctx, actor, ActionDocArticleCreate, ResourceDocArticle, details, func() (int64, error) {
		if createErr := s.content.CreateDocArticle(ctx, article); createErr != nil {
			return 0, createErr
		}
		return article.ID, nil
	})
	if err != nil {
		return nil, err
	}
	return s.content.GetDocArticle(ctx, id)
}

// UpdateDocArticle applies a partial article update.
func (s *Service) UpdateDocArticle(ctx context.Context, actor Actor, id int64, in DocArticleInput) (*content.DocArticle, error) {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return nil, err
	}
	article, err := s.content.GetDocArticle(ctx, id)
	if err != nil {
		return nil, mapError(err)
	}
	if in.Slug != nil {
		v := derefString(in.Slug)
		if err := validateSlug("slug", v); err != nil {
			return nil, err
		}
		article.Slug = v
	}
	if in.CategoryID != nil {
		if *in.CategoryID <= 0 {
			return nil, validationError("category_id is required")
		}
		article.CategoryID = *in.CategoryID
	}
	if in.SortOrder != nil {
		article.SortOrder = *in.SortOrder
	}
	if in.Published != nil {
		article.Published = *in.Published
	}
	if in.Translations != nil {
		translations, err := sanitizeArticleTranslations(in.Translations)
		if err != nil {
			return nil, err
		}
		if err := s.validateTranslationLocales(ctx, mapKeys(in.Translations)); err != nil {
			return nil, err
		}
		article.Translations = translations
	}
	details := map[string]any{"slug": article.Slug, "published": article.Published, "sort_order": article.SortOrder, "category_id": idString(article.CategoryID)}
	if err := s.withPendingAudit(ctx, actor, ActionDocArticleUpdate, ResourceDocArticle, idString(id), details, func() error {
		return s.content.UpdateDocArticle(ctx, article)
	}); err != nil {
		return nil, err
	}
	return s.content.GetDocArticle(ctx, id)
}

// DeleteDocArticle removes an article.
func (s *Service) DeleteDocArticle(ctx context.Context, actor Actor, id int64) error {
	if err := s.requirePermission(ctx, actor, content.PermissionManage); err != nil {
		return err
	}
	return s.withPendingAudit(ctx, actor, ActionDocArticleDelete, ResourceDocArticle, idString(id), nil, func() error {
		return s.content.DeleteDocArticle(ctx, id)
	})
}
