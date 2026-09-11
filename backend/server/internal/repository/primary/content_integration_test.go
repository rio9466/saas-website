package primary_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/rio9466/easy-admin/server/internal/config"
	"github.com/rio9466/easy-admin/server/internal/domain/content"
	platformpostgres "github.com/rio9466/easy-admin/server/internal/platform/postgres"
	"github.com/rio9466/easy-admin/server/internal/repository/primary"
)

func openContentRepo(t *testing.T) *primary.ContentRepository {
	t.Helper()
	requireIsolatedDB(t)
	cfgPath := os.Getenv("EASY_ADMIN_CONFIG")
	if cfgPath == "" {
		cfgPath = findRepoFile("configs/config.local.toml")
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	ctx := context.Background()
	db, err := platformpostgres.Open(ctx, cfg.Database.Primary)
	if err != nil {
		t.Skipf("skip: primary postgres unavailable: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return primary.NewContentRepository(db.GORM())
}

func TestContentLocalesSeeded(t *testing.T) {
	repo := openContentRepo(t)
	ctx := context.Background()
	locales, err := repo.ListLocales(ctx)
	if err != nil {
		t.Fatalf("list locales: %v", err)
	}
	byCode := map[string]content.Locale{}
	for _, l := range locales {
		byCode[l.Code] = l
	}
	if !byCode["en"].IsDefault || !byCode["en"].Enabled {
		t.Fatalf("en must be the enabled default: %#v", byCode["en"])
	}
	if !byCode["zh-CN"].Enabled {
		t.Fatalf("zh-CN must be enabled: %#v", byCode["zh-CN"])
	}
}

func TestContentSiteSettingsOptimisticLock(t *testing.T) {
	repo := openContentRepo(t)
	ctx := context.Background()

	current, err := repo.GetSiteSettings(ctx)
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	updated := &content.SiteSettings{
		SiteName:      "Acme " + uniqueTestDBName("site"),
		DefaultLocale: "en",
		UpdatedBy:     0,
		Translations: map[string]content.SiteSettingTranslation{
			"en": {Tagline: "Hello", SEODefaultTitle: "Acme"},
		},
	}
	if err := repo.UpdateSiteSettings(ctx, updated, current.Version); err != nil {
		t.Fatalf("update settings: %v", err)
	}
	reloaded, err := repo.GetSiteSettings(ctx)
	if err != nil {
		t.Fatalf("reload settings: %v", err)
	}
	if reloaded.Version != current.Version+1 {
		t.Fatalf("version = %d, want %d", reloaded.Version, current.Version+1)
	}
	if reloaded.Translations["en"].Tagline != "Hello" {
		t.Fatalf("translation not persisted: %#v", reloaded.Translations)
	}

	// A stale version must be rejected as an optimistic-lock conflict.
	stale := *updated
	if err := repo.UpdateSiteSettings(ctx, &stale, current.Version); !errors.Is(err, primary.ErrSettingsVersionConflict) {
		t.Fatalf("stale update err = %v, want ErrSettingsVersionConflict", err)
	}
}

func TestContentNavigationCRUDAndTranslations(t *testing.T) {
	repo := openContentRepo(t)
	ctx := context.Background()
	placement := "footer"

	parent := &content.NavigationItem{
		Placement:    placement,
		URL:          "/about",
		Target:       content.TargetSelf,
		SortOrder:    1,
		Visible:      true,
		Translations: map[string]string{"en": "About", "zh-CN": "关于"},
	}
	if err := repo.CreateNavigation(ctx, parent); err != nil {
		t.Fatalf("create parent: %v", err)
	}
	if parent.ID == 0 {
		t.Fatal("parent id not assigned")
	}

	child := &content.NavigationItem{
		Placement:    placement,
		ParentID:     &parent.ID,
		URL:          "/about/team",
		Target:       content.TargetSelf,
		SortOrder:    1,
		Visible:      true,
		Translations: map[string]string{"en": "Team"},
	}
	if err := repo.CreateNavigation(ctx, child); err != nil {
		t.Fatalf("create child: %v", err)
	}

	items, total, err := repo.ListNavigation(ctx, placement, 1, 0)
	if err != nil {
		t.Fatalf("list navigation: %v", err)
	}
	if total < 2 {
		t.Fatalf("total = %d, want >= 2", total)
	}
	var found *content.NavigationItem
	for i := range items {
		if items[i].ID == parent.ID {
			found = &items[i]
			break
		}
	}
	if found == nil || found.Translations["zh-CN"] != "关于" {
		t.Fatalf("parent translations not loaded: %#v", found)
	}

	// Update replaces translations.
	found.Translations = map[string]string{"en": "About us"}
	if err := repo.UpdateNavigation(ctx, found); err != nil {
		t.Fatalf("update navigation: %v", err)
	}
	reloaded, err := repo.GetNavigation(ctx, parent.ID)
	if err != nil {
		t.Fatalf("get navigation: %v", err)
	}
	if len(reloaded.Translations) != 1 || reloaded.Translations["en"] != "About us" {
		t.Fatalf("translations after update: %#v", reloaded.Translations)
	}

	if err := repo.DeleteNavigation(ctx, child.ID); err != nil {
		t.Fatalf("delete child: %v", err)
	}
	if err := repo.DeleteNavigation(ctx, parent.ID); err != nil {
		t.Fatalf("delete parent: %v", err)
	}
	if _, err := repo.GetNavigation(ctx, parent.ID); !errors.Is(err, primary.ErrNotFound) {
		t.Fatalf("get deleted = %v, want ErrNotFound", err)
	}
}

func TestContentFeaturePricingPageDocRoundTrip(t *testing.T) {
	repo := openContentRepo(t)
	ctx := context.Background()
	suffix := uniqueTestDBName("c")

	feature := &content.Feature{
		Icon:      "i-lucide-zap",
		SortOrder: 1,
		Published: true,
		Translations: map[string]content.FeatureTranslation{
			"en": {Title: "Fast", Summary: "Very fast", BodyMD: "**bold**"},
		},
	}
	if err := repo.CreateFeature(ctx, feature); err != nil {
		t.Fatalf("create feature: %v", err)
	}
	loadedFeature, err := repo.GetFeature(ctx, feature.ID)
	if err != nil {
		t.Fatalf("get feature: %v", err)
	}
	if loadedFeature.Translations["en"].Title != "Fast" {
		t.Fatalf("feature translation: %#v", loadedFeature.Translations)
	}

	plan := &content.PricingPlan{
		Code:         "pro-" + suffix,
		MonthlyPrice: "29.00",
		YearlyPrice:  "290.00",
		Currency:     "USD",
		Highlighted:  true,
		SortOrder:    1,
		Visible:      true,
		Translations: map[string]content.PricingPlanTranslation{
			"en": {Name: "Pro", CTALabel: "Go", CTAURL: "/register"},
		},
		Features: map[string][]string{"en": {"10 seats", "Priority"}},
	}
	if err := repo.CreatePricingPlan(ctx, plan); err != nil {
		t.Fatalf("create plan: %v", err)
	}
	loadedPlan, err := repo.GetPricingPlan(ctx, plan.ID)
	if err != nil {
		t.Fatalf("get plan: %v", err)
	}
	if loadedPlan.MonthlyPrice != "29.00" || len(loadedPlan.Features["en"]) != 2 {
		t.Fatalf("plan round trip: %#v", loadedPlan)
	}

	page := &content.Page{
		Slug:         "about-" + suffix,
		Published:    true,
		SortOrder:    1,
		Translations: map[string]content.PageTranslation{"en": {Title: "About", BodyMD: "# hi"}},
	}
	if err := repo.CreatePage(ctx, page); err != nil {
		t.Fatalf("create page: %v", err)
	}
	loadedPage, err := repo.GetPageBySlug(ctx, page.Slug)
	if err != nil {
		t.Fatalf("get page by slug: %v", err)
	}
	if loadedPage.Translations["en"].Title != "About" {
		t.Fatalf("page translation: %#v", loadedPage.Translations)
	}

	category := &content.DocCategory{
		Slug:         "docs-" + suffix,
		SortOrder:    1,
		Translations: map[string]string{"en": "Guides"},
	}
	if err := repo.CreateDocCategory(ctx, category); err != nil {
		t.Fatalf("create category: %v", err)
	}
	article := &content.DocArticle{
		Slug:         "install-" + suffix,
		CategoryID:   category.ID,
		SortOrder:    1,
		Published:    true,
		Translations: map[string]content.DocArticleTranslation{"en": {Title: "Install", BodyMD: "run"}},
	}
	if err := repo.CreateDocArticle(ctx, article); err != nil {
		t.Fatalf("create article: %v", err)
	}
	loadedArticle, err := repo.GetDocArticleBySlug(ctx, article.Slug)
	if err != nil {
		t.Fatalf("get article: %v", err)
	}
	if loadedArticle.CategoryID != category.ID || loadedArticle.Translations["en"].Title != "Install" {
		t.Fatalf("article round trip: %#v", loadedArticle)
	}

	// A category still holding an article cannot be deleted.
	if err := repo.DeleteDocCategory(ctx, category.ID); !errors.Is(err, primary.ErrConflict) {
		t.Fatalf("delete non-empty category = %v, want ErrConflict", err)
	}
	if err := repo.DeleteDocArticle(ctx, article.ID); err != nil {
		t.Fatalf("delete article: %v", err)
	}
	if err := repo.DeleteDocCategory(ctx, category.ID); err != nil {
		t.Fatalf("delete empty category: %v", err)
	}
}
