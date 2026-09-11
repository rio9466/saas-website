package contentsvc

import (
	"context"
	"sort"
	"strings"

	"github.com/rio9466/easy-admin/server/internal/domain/apperr"
	"github.com/rio9466/easy-admin/server/internal/domain/content"
)

// resolveLocale returns the effective locale and the configured default.
func (s *Service) resolveLocale(ctx context.Context, requested string) (string, string, error) {
	locales, err := s.content.ListLocales(ctx)
	if err != nil {
		return "", "", mapError(err)
	}
	return content.ResolveLocale(requested, locales), content.DefaultLocale(locales), nil
}

// GetPublicSettings returns the content-owned public settings slice.
func (s *Service) GetPublicSettings(ctx context.Context, locale string) (*content.PublicSettingsView, error) {
	effective, fallback, err := s.resolveLocale(ctx, locale)
	if err != nil {
		return nil, err
	}
	locales, err := s.content.ListLocales(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	settings, err := s.content.GetSiteSettings(ctx)
	if err != nil {
		return nil, mapError(err)
	}
	translation, _ := content.PickTranslation(settings.Translations, effective, fallback)

	options := make([]content.LocaleOptionView, 0, len(locales))
	for _, l := range content.EnabledLocales(locales) {
		options = append(options, content.LocaleOptionView{Code: l.Code, Label: l.Label})
	}
	return &content.PublicSettingsView{
		SiteName:              settings.SiteName,
		LogoURL:               settings.LogoURL,
		LogoDarkURL:           settings.LogoDarkURL,
		FaviconURL:            settings.FaviconURL,
		Tagline:               translation.Tagline,
		FooterText:            translation.FooterText,
		ICPRecord:             translation.ICPRecord,
		ContactEmail:          settings.ContactEmail,
		ContactPhone:          settings.ContactPhone,
		ContactAddress:        settings.ContactAddress,
		SocialLinks:           append([]content.SocialLink(nil), settings.SocialLinks...),
		SEODefaultTitle:       translation.SEODefaultTitle,
		SEODefaultDescription: translation.SEODefaultDescription,
		SEODefaultOGImageURL:  settings.SEODefaultOGImageURL,
		DefaultLocale:         content.DefaultLocale(locales),
		Locales:               options,
	}, nil
}

// GetPublicNavigation returns the visible navigation tree for a placement.
func (s *Service) GetPublicNavigation(ctx context.Context, locale, placement string) (*content.PublicNavigationView, error) {
	effective, fallback, err := s.resolveLocale(ctx, locale)
	if err != nil {
		return nil, err
	}
	placement = strings.TrimSpace(placement)
	if placement == "" {
		placement = content.PlacementHeader
	}
	if !validPlacement(placement) {
		return nil, validationError("placement must be header or footer")
	}
	items, _, err := s.content.ListNavigation(ctx, placement, 1, 0)
	if err != nil {
		return nil, mapError(err)
	}
	visible := make(map[int64]bool, len(items))
	nodes := make(map[int64]*content.PublicNavigationItemView, len(items))
	for _, item := range items {
		if !item.Visible {
			continue
		}
		label, _ := content.PickTranslation(item.Translations, effective, fallback)
		nodes[item.ID] = &content.PublicNavigationItemView{
			ID:        idString(item.ID),
			Label:     label,
			URL:       item.URL,
			Target:    item.Target,
			SortOrder: item.SortOrder,
			Children:  []content.PublicNavigationItemView{},
		}
		visible[item.ID] = true
	}
	roots := make([]*content.PublicNavigationItemView, 0, len(nodes))
	for _, item := range items {
		node, ok := nodes[item.ID]
		if !ok {
			continue
		}
		if item.ParentID != nil && visible[*item.ParentID] {
			parent := nodes[*item.ParentID]
			parent.Children = append(parent.Children, *node)
			continue
		}
		roots = append(roots, node)
	}
	out := make([]content.PublicNavigationItemView, 0, len(roots))
	for _, node := range roots {
		out = append(out, *node)
	}
	return &content.PublicNavigationView{Locale: effective, Items: out}, nil
}

// GetPublicHome returns published home sections with merged locale payloads.
func (s *Service) GetPublicHome(ctx context.Context, locale string) (*content.PublicHomeView, error) {
	effective, fallback, err := s.resolveLocale(ctx, locale)
	if err != nil {
		return nil, err
	}
	sections, _, err := s.content.ListHomeSections(ctx, 1, 0)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]content.PublicHomeSectionView, 0, len(sections))
	for _, section := range sections {
		if !section.Published {
			continue
		}
		data := section.Payload
		if translation, ok := content.PickTranslation(section.Translations, effective, fallback); ok {
			data = content.MergePayload(section.Payload, translation)
		}
		out = append(out, content.PublicHomeSectionView{
			ID:        idString(section.ID),
			Type:      section.Type,
			SortOrder: section.SortOrder,
			Data:      data,
		})
	}
	return &content.PublicHomeView{Locale: effective, Sections: out}, nil
}

// GetPublicFeatures returns published features.
func (s *Service) GetPublicFeatures(ctx context.Context, locale string) (*content.PublicFeaturesView, error) {
	effective, fallback, err := s.resolveLocale(ctx, locale)
	if err != nil {
		return nil, err
	}
	features, _, err := s.content.ListFeatures(ctx, 1, 0)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]content.PublicFeatureView, 0, len(features))
	for _, feature := range features {
		if !feature.Published {
			continue
		}
		translation, _ := content.PickTranslation(feature.Translations, effective, fallback)
		out = append(out, content.PublicFeatureView{
			ID:        idString(feature.ID),
			Icon:      feature.Icon,
			Title:     translation.Title,
			Summary:   translation.Summary,
			BodyMD:    content.SanitizeMarkdown(translation.BodyMD),
			ImageURL:  translation.ImageURL,
			SortOrder: feature.SortOrder,
		})
	}
	return &content.PublicFeaturesView{Locale: effective, Items: out}, nil
}

// GetPublicPricing returns visible pricing plans.
func (s *Service) GetPublicPricing(ctx context.Context, locale string) (*content.PublicPricingView, error) {
	effective, fallback, err := s.resolveLocale(ctx, locale)
	if err != nil {
		return nil, err
	}
	plans, _, err := s.content.ListPricingPlans(ctx, 1, 0)
	if err != nil {
		return nil, mapError(err)
	}
	view := &content.PublicPricingView{Locale: effective, Currency: "", Plans: []content.PublicPlanView{}}
	for _, plan := range plans {
		if !plan.Visible {
			continue
		}
		translation, _ := content.PickTranslation(plan.Translations, effective, fallback)
		features, _ := content.PickTranslation(plan.Features, effective, fallback)
		if features == nil {
			features = []string{}
		}
		view.Plans = append(view.Plans, content.PublicPlanView{
			ID:           idString(plan.ID),
			Code:         plan.Code,
			Name:         translation.Name,
			Description:  translation.Description,
			MonthlyPrice: plan.MonthlyPrice,
			YearlyPrice:  plan.YearlyPrice,
			Currency:     plan.Currency,
			Highlighted:  plan.Highlighted,
			CTALabel:     translation.CTALabel,
			CTAURL:       translation.CTAURL,
			Features:     append([]string(nil), features...),
			SortOrder:    plan.SortOrder,
		})
		if view.Currency == "" {
			view.Currency = plan.Currency
		}
	}
	return view, nil
}

// GetPublicPages returns published pages.
func (s *Service) GetPublicPages(ctx context.Context, locale string) (*content.PublicPageListView, error) {
	effective, fallback, err := s.resolveLocale(ctx, locale)
	if err != nil {
		return nil, err
	}
	pages, _, err := s.content.ListPages(ctx, 1, 0)
	if err != nil {
		return nil, mapError(err)
	}
	out := make([]content.PublicPageSummaryView, 0, len(pages))
	for _, page := range pages {
		if !page.Published {
			continue
		}
		translation, _ := content.PickTranslation(page.Translations, effective, fallback)
		out = append(out, content.PublicPageSummaryView{
			Slug:      page.Slug,
			Title:     translation.Title,
			UpdatedAt: content.FormatTime(page.UpdatedAt),
		})
	}
	return &content.PublicPageListView{Locale: effective, Items: out}, nil
}

// GetPublicPage returns one published page by slug.
func (s *Service) GetPublicPage(ctx context.Context, locale, slug string) (*content.PublicPageView, error) {
	effective, fallback, err := s.resolveLocale(ctx, locale)
	if err != nil {
		return nil, err
	}
	page, err := s.content.GetPageBySlug(ctx, strings.TrimSpace(slug))
	if err != nil {
		return nil, mapError(err)
	}
	if !page.Published {
		return nil, apperr.New(404, apperr.CodeNotFound, "not found", apperr.ErrNotFound)
	}
	translation, _ := content.PickTranslation(page.Translations, effective, fallback)
	return &content.PublicPageView{
		Locale:         effective,
		Slug:           page.Slug,
		Title:          translation.Title,
		BodyMD:         content.SanitizeMarkdown(translation.BodyMD),
		SEOTitle:       translation.SEOTitle,
		SEODescription: translation.SEODescription,
		UpdatedAt:      content.FormatTime(page.UpdatedAt),
	}, nil
}

// GetPublicDocs returns the documentation tree of categories and published articles.
func (s *Service) GetPublicDocs(ctx context.Context, locale string) (*content.PublicDocListView, error) {
	effective, fallback, err := s.resolveLocale(ctx, locale)
	if err != nil {
		return nil, err
	}
	categories, _, err := s.content.ListDocCategories(ctx, 1, 0)
	if err != nil {
		return nil, mapError(err)
	}
	articles, _, err := s.content.ListDocArticles(ctx, 1, 0)
	if err != nil {
		return nil, mapError(err)
	}
	byCategory := map[int64][]content.DocArticle{}
	for _, article := range articles {
		if article.Published {
			byCategory[article.CategoryID] = append(byCategory[article.CategoryID], article)
		}
	}
	out := make([]content.PublicDocCategoryView, 0, len(categories))
	for _, category := range categories {
		name, _ := content.PickTranslation(category.Translations, effective, fallback)
		list := make([]content.PublicDocArticleSummaryView, 0)
		for _, article := range byCategory[category.ID] {
			tr, _ := content.PickTranslation(article.Translations, effective, fallback)
			list = append(list, content.PublicDocArticleSummaryView{
				ID:        idString(article.ID),
				Slug:      article.Slug,
				Title:     tr.Title,
				SortOrder: article.SortOrder,
				UpdatedAt: content.FormatTime(article.UpdatedAt),
			})
		}
		out = append(out, content.PublicDocCategoryView{
			ID:        idString(category.ID),
			Slug:      category.Slug,
			Name:      name,
			SortOrder: category.SortOrder,
			Articles:  list,
		})
	}
	return &content.PublicDocListView{Locale: effective, Categories: out}, nil
}

// GetPublicDoc returns one published article with category and prev/next links.
func (s *Service) GetPublicDoc(ctx context.Context, locale, slug string) (*content.PublicDocArticleView, error) {
	effective, fallback, err := s.resolveLocale(ctx, locale)
	if err != nil {
		return nil, err
	}
	article, err := s.content.GetDocArticleBySlug(ctx, strings.TrimSpace(slug))
	if err != nil {
		return nil, mapError(err)
	}
	if !article.Published {
		return nil, apperr.New(404, apperr.CodeNotFound, "not found", apperr.ErrNotFound)
	}
	translation, _ := content.PickTranslation(article.Translations, effective, fallback)

	categoryRef := content.PublicDocCategoryRef{ID: idString(article.CategoryID)}
	if category, catErr := s.content.GetDocCategory(ctx, article.CategoryID); catErr == nil {
		categoryRef.Slug = category.Slug
		categoryRef.Name, _ = content.PickTranslation(category.Translations, effective, fallback)
	}

	prev, next := s.articleNeighbours(ctx, article, effective, fallback)
	return &content.PublicDocArticleView{
		Locale:         effective,
		ID:             idString(article.ID),
		Slug:           article.Slug,
		Title:          translation.Title,
		BodyMD:         content.SanitizeMarkdown(translation.BodyMD),
		Category:       categoryRef,
		Prev:           prev,
		Next:           next,
		SEOTitle:       translation.SEOTitle,
		SEODescription: translation.SEODescription,
		UpdatedAt:      content.FormatTime(article.UpdatedAt),
	}, nil
}

func (s *Service) articleNeighbours(ctx context.Context, article *content.DocArticle, effective, fallback string) (*content.PublicDocRef, *content.PublicDocRef) {
	articles, _, err := s.content.ListDocArticles(ctx, 1, 0)
	if err != nil {
		return nil, nil
	}
	siblings := make([]content.DocArticle, 0, len(articles))
	for _, candidate := range articles {
		if candidate.Published && candidate.CategoryID == article.CategoryID {
			siblings = append(siblings, candidate)
		}
	}
	sort.SliceStable(siblings, func(i, j int) bool {
		if siblings[i].SortOrder != siblings[j].SortOrder {
			return siblings[i].SortOrder < siblings[j].SortOrder
		}
		return siblings[i].ID < siblings[j].ID
	})
	index := -1
	for i := range siblings {
		if siblings[i].ID == article.ID {
			index = i
			break
		}
	}
	if index < 0 {
		return nil, nil
	}
	makeRef := func(a content.DocArticle) *content.PublicDocRef {
		tr, _ := content.PickTranslation(a.Translations, effective, fallback)
		return &content.PublicDocRef{Slug: a.Slug, Title: tr.Title}
	}
	var prev, next *content.PublicDocRef
	if index > 0 {
		prev = makeRef(siblings[index-1])
	}
	if index < len(siblings)-1 {
		next = makeRef(siblings[index+1])
	}
	return prev, next
}
