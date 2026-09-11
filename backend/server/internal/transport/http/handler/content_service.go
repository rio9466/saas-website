package handler

import (
	"context"

	"github.com/rio9466/easy-admin/server/internal/domain/adminauth"
	"github.com/rio9466/easy-admin/server/internal/domain/content"
	contentsvc "github.com/rio9466/easy-admin/server/internal/service/contentsvc"
)

// ContentService is the transport-facing content surface: public reads plus
// administrator CRUD. Input types are shared with the service package.
type ContentService interface {
	// Public reads (no authentication).
	GetPublicSettings(ctx context.Context, locale string) (*content.PublicSettingsView, error)
	GetPublicNavigation(ctx context.Context, locale, placement string) (*content.PublicNavigationView, error)
	GetPublicHome(ctx context.Context, locale string) (*content.PublicHomeView, error)
	GetPublicFeatures(ctx context.Context, locale string) (*content.PublicFeaturesView, error)
	GetPublicPricing(ctx context.Context, locale string) (*content.PublicPricingView, error)
	GetPublicPages(ctx context.Context, locale string) (*content.PublicPageListView, error)
	GetPublicPage(ctx context.Context, locale, slug string) (*content.PublicPageView, error)
	GetPublicDocs(ctx context.Context, locale string) (*content.PublicDocListView, error)
	GetPublicDoc(ctx context.Context, locale, slug string) (*content.PublicDocArticleView, error)

	// Administrator content management.
	GetSiteSettings(ctx context.Context, actor Actor) (*content.SiteSettings, error)
	UpdateSiteSettings(ctx context.Context, actor Actor, in contentsvc.SiteSettingsInput) (*content.SiteSettings, error)

	ListNavigationItems(ctx context.Context, actor Actor, page, pageSize int, placement string) (adminauth.Page[content.NavigationItem], error)
	CreateNavigationItem(ctx context.Context, actor Actor, in contentsvc.NavigationItemInput) (*content.NavigationItem, error)
	UpdateNavigationItem(ctx context.Context, actor Actor, id int64, in contentsvc.NavigationItemInput) (*content.NavigationItem, error)
	DeleteNavigationItem(ctx context.Context, actor Actor, id int64) error

	ListHomeSections(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.HomeSection], error)
	CreateHomeSection(ctx context.Context, actor Actor, in contentsvc.HomeSectionInput) (*content.HomeSection, error)
	UpdateHomeSection(ctx context.Context, actor Actor, id int64, in contentsvc.HomeSectionInput) (*content.HomeSection, error)
	DeleteHomeSection(ctx context.Context, actor Actor, id int64) error

	ListFeatures(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.Feature], error)
	CreateFeature(ctx context.Context, actor Actor, in contentsvc.FeatureInput) (*content.Feature, error)
	UpdateFeature(ctx context.Context, actor Actor, id int64, in contentsvc.FeatureInput) (*content.Feature, error)
	DeleteFeature(ctx context.Context, actor Actor, id int64) error

	ListPricingPlans(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.PricingPlan], error)
	CreatePricingPlan(ctx context.Context, actor Actor, in contentsvc.PricingPlanInput) (*content.PricingPlan, error)
	UpdatePricingPlan(ctx context.Context, actor Actor, id int64, in contentsvc.PricingPlanInput) (*content.PricingPlan, error)
	DeletePricingPlan(ctx context.Context, actor Actor, id int64) error

	ListPages(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.Page], error)
	GetPage(ctx context.Context, actor Actor, id int64) (*content.Page, error)
	CreatePage(ctx context.Context, actor Actor, in contentsvc.PageInput) (*content.Page, error)
	UpdatePage(ctx context.Context, actor Actor, id int64, in contentsvc.PageInput) (*content.Page, error)
	DeletePage(ctx context.Context, actor Actor, id int64) error

	ListDocCategories(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.DocCategory], error)
	CreateDocCategory(ctx context.Context, actor Actor, in contentsvc.DocCategoryInput) (*content.DocCategory, error)
	UpdateDocCategory(ctx context.Context, actor Actor, id int64, in contentsvc.DocCategoryInput) (*content.DocCategory, error)
	DeleteDocCategory(ctx context.Context, actor Actor, id int64) error

	ListDocArticles(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.DocArticle], error)
	GetDocArticle(ctx context.Context, actor Actor, id int64) (*content.DocArticle, error)
	CreateDocArticle(ctx context.Context, actor Actor, in contentsvc.DocArticleInput) (*content.DocArticle, error)
	UpdateDocArticle(ctx context.Context, actor Actor, id int64, in contentsvc.DocArticleInput) (*content.DocArticle, error)
	DeleteDocArticle(ctx context.Context, actor Actor, id int64) error
}

// ContentServiceAdapter adapts *contentsvc.Service to ContentService.
type ContentServiceAdapter struct {
	Svc *contentsvc.Service
}

var _ ContentService = (*ContentServiceAdapter)(nil)

func toContentActor(a Actor) contentsvc.Actor {
	return contentsvc.Actor{
		AdminID:          a.ID,
		AdminUsername:    a.Username,
		AdminDisplayName: a.DisplayName,
		AdminRoleCodes:   a.RoleCodes,
		RequestID:        a.RequestID,
		SourceIP:         a.SourceIP,
		UserAgent:        a.UserAgent,
	}
}

func (a *ContentServiceAdapter) GetPublicSettings(ctx context.Context, locale string) (*content.PublicSettingsView, error) {
	return a.Svc.GetPublicSettings(ctx, locale)
}

func (a *ContentServiceAdapter) GetPublicNavigation(ctx context.Context, locale, placement string) (*content.PublicNavigationView, error) {
	return a.Svc.GetPublicNavigation(ctx, locale, placement)
}

func (a *ContentServiceAdapter) GetPublicHome(ctx context.Context, locale string) (*content.PublicHomeView, error) {
	return a.Svc.GetPublicHome(ctx, locale)
}

func (a *ContentServiceAdapter) GetPublicFeatures(ctx context.Context, locale string) (*content.PublicFeaturesView, error) {
	return a.Svc.GetPublicFeatures(ctx, locale)
}

func (a *ContentServiceAdapter) GetPublicPricing(ctx context.Context, locale string) (*content.PublicPricingView, error) {
	return a.Svc.GetPublicPricing(ctx, locale)
}

func (a *ContentServiceAdapter) GetPublicPages(ctx context.Context, locale string) (*content.PublicPageListView, error) {
	return a.Svc.GetPublicPages(ctx, locale)
}

func (a *ContentServiceAdapter) GetPublicPage(ctx context.Context, locale, slug string) (*content.PublicPageView, error) {
	return a.Svc.GetPublicPage(ctx, locale, slug)
}

func (a *ContentServiceAdapter) GetPublicDocs(ctx context.Context, locale string) (*content.PublicDocListView, error) {
	return a.Svc.GetPublicDocs(ctx, locale)
}

func (a *ContentServiceAdapter) GetPublicDoc(ctx context.Context, locale, slug string) (*content.PublicDocArticleView, error) {
	return a.Svc.GetPublicDoc(ctx, locale, slug)
}

func (a *ContentServiceAdapter) GetSiteSettings(ctx context.Context, actor Actor) (*content.SiteSettings, error) {
	return a.Svc.GetSiteSettings(ctx, toContentActor(actor))
}

func (a *ContentServiceAdapter) UpdateSiteSettings(ctx context.Context, actor Actor, in contentsvc.SiteSettingsInput) (*content.SiteSettings, error) {
	return a.Svc.UpdateSiteSettings(ctx, toContentActor(actor), in)
}

func (a *ContentServiceAdapter) ListNavigationItems(ctx context.Context, actor Actor, page, pageSize int, placement string) (adminauth.Page[content.NavigationItem], error) {
	return a.Svc.ListNavigationItems(ctx, toContentActor(actor), page, pageSize, placement)
}

func (a *ContentServiceAdapter) CreateNavigationItem(ctx context.Context, actor Actor, in contentsvc.NavigationItemInput) (*content.NavigationItem, error) {
	return a.Svc.CreateNavigationItem(ctx, toContentActor(actor), in)
}

func (a *ContentServiceAdapter) UpdateNavigationItem(ctx context.Context, actor Actor, id int64, in contentsvc.NavigationItemInput) (*content.NavigationItem, error) {
	return a.Svc.UpdateNavigationItem(ctx, toContentActor(actor), id, in)
}

func (a *ContentServiceAdapter) DeleteNavigationItem(ctx context.Context, actor Actor, id int64) error {
	return a.Svc.DeleteNavigationItem(ctx, toContentActor(actor), id)
}

func (a *ContentServiceAdapter) ListHomeSections(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.HomeSection], error) {
	return a.Svc.ListHomeSections(ctx, toContentActor(actor), page, pageSize)
}

func (a *ContentServiceAdapter) CreateHomeSection(ctx context.Context, actor Actor, in contentsvc.HomeSectionInput) (*content.HomeSection, error) {
	return a.Svc.CreateHomeSection(ctx, toContentActor(actor), in)
}

func (a *ContentServiceAdapter) UpdateHomeSection(ctx context.Context, actor Actor, id int64, in contentsvc.HomeSectionInput) (*content.HomeSection, error) {
	return a.Svc.UpdateHomeSection(ctx, toContentActor(actor), id, in)
}

func (a *ContentServiceAdapter) DeleteHomeSection(ctx context.Context, actor Actor, id int64) error {
	return a.Svc.DeleteHomeSection(ctx, toContentActor(actor), id)
}

func (a *ContentServiceAdapter) ListFeatures(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.Feature], error) {
	return a.Svc.ListFeatures(ctx, toContentActor(actor), page, pageSize)
}

func (a *ContentServiceAdapter) CreateFeature(ctx context.Context, actor Actor, in contentsvc.FeatureInput) (*content.Feature, error) {
	return a.Svc.CreateFeature(ctx, toContentActor(actor), in)
}

func (a *ContentServiceAdapter) UpdateFeature(ctx context.Context, actor Actor, id int64, in contentsvc.FeatureInput) (*content.Feature, error) {
	return a.Svc.UpdateFeature(ctx, toContentActor(actor), id, in)
}

func (a *ContentServiceAdapter) DeleteFeature(ctx context.Context, actor Actor, id int64) error {
	return a.Svc.DeleteFeature(ctx, toContentActor(actor), id)
}

func (a *ContentServiceAdapter) ListPricingPlans(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.PricingPlan], error) {
	return a.Svc.ListPricingPlans(ctx, toContentActor(actor), page, pageSize)
}

func (a *ContentServiceAdapter) CreatePricingPlan(ctx context.Context, actor Actor, in contentsvc.PricingPlanInput) (*content.PricingPlan, error) {
	return a.Svc.CreatePricingPlan(ctx, toContentActor(actor), in)
}

func (a *ContentServiceAdapter) UpdatePricingPlan(ctx context.Context, actor Actor, id int64, in contentsvc.PricingPlanInput) (*content.PricingPlan, error) {
	return a.Svc.UpdatePricingPlan(ctx, toContentActor(actor), id, in)
}

func (a *ContentServiceAdapter) DeletePricingPlan(ctx context.Context, actor Actor, id int64) error {
	return a.Svc.DeletePricingPlan(ctx, toContentActor(actor), id)
}

func (a *ContentServiceAdapter) ListPages(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.Page], error) {
	return a.Svc.ListPages(ctx, toContentActor(actor), page, pageSize)
}

func (a *ContentServiceAdapter) GetPage(ctx context.Context, actor Actor, id int64) (*content.Page, error) {
	return a.Svc.GetPage(ctx, toContentActor(actor), id)
}

func (a *ContentServiceAdapter) CreatePage(ctx context.Context, actor Actor, in contentsvc.PageInput) (*content.Page, error) {
	return a.Svc.CreatePage(ctx, toContentActor(actor), in)
}

func (a *ContentServiceAdapter) UpdatePage(ctx context.Context, actor Actor, id int64, in contentsvc.PageInput) (*content.Page, error) {
	return a.Svc.UpdatePage(ctx, toContentActor(actor), id, in)
}

func (a *ContentServiceAdapter) DeletePage(ctx context.Context, actor Actor, id int64) error {
	return a.Svc.DeletePage(ctx, toContentActor(actor), id)
}

func (a *ContentServiceAdapter) ListDocCategories(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.DocCategory], error) {
	return a.Svc.ListDocCategories(ctx, toContentActor(actor), page, pageSize)
}

func (a *ContentServiceAdapter) CreateDocCategory(ctx context.Context, actor Actor, in contentsvc.DocCategoryInput) (*content.DocCategory, error) {
	return a.Svc.CreateDocCategory(ctx, toContentActor(actor), in)
}

func (a *ContentServiceAdapter) UpdateDocCategory(ctx context.Context, actor Actor, id int64, in contentsvc.DocCategoryInput) (*content.DocCategory, error) {
	return a.Svc.UpdateDocCategory(ctx, toContentActor(actor), id, in)
}

func (a *ContentServiceAdapter) DeleteDocCategory(ctx context.Context, actor Actor, id int64) error {
	return a.Svc.DeleteDocCategory(ctx, toContentActor(actor), id)
}

func (a *ContentServiceAdapter) ListDocArticles(ctx context.Context, actor Actor, page, pageSize int) (adminauth.Page[content.DocArticle], error) {
	return a.Svc.ListDocArticles(ctx, toContentActor(actor), page, pageSize)
}

func (a *ContentServiceAdapter) GetDocArticle(ctx context.Context, actor Actor, id int64) (*content.DocArticle, error) {
	return a.Svc.GetDocArticle(ctx, toContentActor(actor), id)
}

func (a *ContentServiceAdapter) CreateDocArticle(ctx context.Context, actor Actor, in contentsvc.DocArticleInput) (*content.DocArticle, error) {
	return a.Svc.CreateDocArticle(ctx, toContentActor(actor), in)
}

func (a *ContentServiceAdapter) UpdateDocArticle(ctx context.Context, actor Actor, id int64, in contentsvc.DocArticleInput) (*content.DocArticle, error) {
	return a.Svc.UpdateDocArticle(ctx, toContentActor(actor), id, in)
}

func (a *ContentServiceAdapter) DeleteDocArticle(ctx context.Context, actor Actor, id int64) error {
	return a.Svc.DeleteDocArticle(ctx, toContentActor(actor), id)
}
