package primary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rio9466/easy-admin/server/internal/domain/content"
	"gorm.io/gorm"
)

// ContentRepository persists the site-content tables in primary PostgreSQL.
type ContentRepository struct {
	db *gorm.DB
}

// NewContentRepository constructs a ContentRepository backed by db.
func NewContentRepository(db *gorm.DB) *ContentRepository {
	return &ContentRepository{db: db}
}

func (r *ContentRepository) session(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

// --- helpers ---------------------------------------------------------------

func jsonbObjectFromMap(m map[string]any) (jsonbRaw, error) {
	if m == nil {
		m = map[string]any{}
	}
	raw, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return jsonbRaw(raw), nil
}

func mapFromJSONBObject(j jsonbRaw) map[string]any {
	if len(j) == 0 {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(j, &m); err != nil || m == nil {
		return map[string]any{}
	}
	return m
}

func jsonbFromLinks(links []content.SocialLink) (jsonbRaw, error) {
	if links == nil {
		links = []content.SocialLink{}
	}
	raw, err := json.Marshal(links)
	if err != nil {
		return nil, err
	}
	return jsonbRaw(raw), nil
}

func linksFromJSONB(j jsonbRaw) []content.SocialLink {
	if len(j) == 0 {
		return []content.SocialLink{}
	}
	var out []content.SocialLink
	if err := json.Unmarshal(j, &out); err != nil || out == nil {
		return []content.SocialLink{}
	}
	return out
}

func numericFromMoney(s string) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.Scan(s); err != nil {
		return n, err
	}
	return n, nil
}

func moneyFromNumeric(n pgtype.Numeric) string {
	v, err := n.Value()
	if err != nil || v == nil {
		return "0.00"
	}
	s, ok := v.(string)
	if !ok {
		return "0.00"
	}
	out, err := content.NormalizeMoney(s)
	if err != nil {
		return "0.00"
	}
	return out
}

func wrapContentDBError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return fmt.Errorf("%w: %w", ErrConflict, err)
		case "23503":
			return fmt.Errorf("%w: %w", ErrConflict, err)
		}
	}
	return wrapDBError(err)
}

func listOrdered[T any](db *gorm.DB, order string, page, pageSize int) ([]T, int64, error) {
	if pageSize <= 0 {
		var items []T
		if err := db.Order(order).Find(&items).Error; err != nil {
			return nil, 0, wrapContentDBError(err)
		}
		if items == nil {
			items = []T{}
		}
		return items, int64(len(items)), nil
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, wrapContentDBError(err)
	}
	var items []T
	if err := db.Order(order).Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, wrapContentDBError(err)
	}
	if items == nil {
		items = []T{}
	}
	return items, total, nil
}

func deleteByParent[T any](tx *gorm.DB, column string, id int64) error {
	var zero T
	if err := tx.Where(column+" = ?", id).Delete(&zero).Error; err != nil {
		return err
	}
	return nil
}

// --- locales ---------------------------------------------------------------

// ListLocales returns all supported locales ordered by sort_order.
func (r *ContentRepository) ListLocales(ctx context.Context) ([]content.Locale, error) {
	var models []localeModel
	if err := r.session(ctx).Order("sort_order, code").Find(&models).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	out := make([]content.Locale, 0, len(models))
	for _, m := range models {
		out = append(out, content.Locale{
			Code:      m.Code,
			Label:     m.Label,
			Enabled:   m.Enabled,
			SortOrder: m.SortOrder,
			IsDefault: m.IsDefault,
		})
	}
	return out, nil
}

func (r *ContentRepository) setDefaultLocale(tx *gorm.DB, code string) error {
	now := time.Now().UTC()
	if err := tx.Model(&localeModel{}).Where("is_default = ?", true).
		Updates(map[string]any{"is_default": false, "updated_at": now}).Error; err != nil {
		return err
	}
	res := tx.Model(&localeModel{}).Where("code = ? AND enabled = ?", code, true).
		Updates(map[string]any{"is_default": true, "updated_at": now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// --- site settings ---------------------------------------------------------

// GetSiteSettings loads the singleton row with all translations.
func (r *ContentRepository) GetSiteSettings(ctx context.Context) (*content.SiteSettings, error) {
	var model siteSettingsModel
	if err := r.session(ctx).Where("id = 1").First(&model).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	settings := &content.SiteSettings{
		SiteName:             model.SiteName,
		LogoURL:              model.LogoURL,
		LogoDarkURL:          model.LogoDarkURL,
		FaviconURL:           model.FaviconURL,
		ContactEmail:         model.ContactEmail,
		ContactPhone:         model.ContactPhone,
		ContactAddress:       model.ContactAddress,
		SocialLinks:          linksFromJSONB(model.SocialLinks),
		SEODefaultOGImageURL: model.SEODefaultOGImageURL,
		DefaultLocale:        model.DefaultLocale,
		Version:              model.Version,
		UpdatedBy:            model.UpdatedBy,
		UpdatedAt:            model.UpdatedAt,
		Translations:         map[string]content.SiteSettingTranslation{},
	}

	var translations []siteSettingTranslationModel
	if err := r.session(ctx).Where("site_setting_id = 1").Find(&translations).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	for _, t := range translations {
		settings.Translations[t.Locale] = content.SiteSettingTranslation{
			Tagline:               t.Tagline,
			FooterText:            t.FooterText,
			SEODefaultTitle:       t.SEODefaultTitle,
			SEODefaultDescription: t.SEODefaultDescription,
			ICPRecord:             t.ICPRecord,
		}
	}
	return settings, nil
}

// UpdateSiteSettings replaces the singleton and its translations under the
// optimistic-lock version. The default locale is also promoted in
// supported_locales inside the same transaction.
func (r *ContentRepository) UpdateSiteSettings(ctx context.Context, in *content.SiteSettings, expectedVersion int) error {
	if in == nil {
		return fmt.Errorf("site settings are nil")
	}
	social, err := jsonbFromLinks(in.SocialLinks)
	if err != nil {
		return err
	}
	return r.session(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		res := tx.Model(&siteSettingsModel{}).
			Where("id = 1 AND version = ?", expectedVersion).
			Updates(map[string]any{
				"site_name":                in.SiteName,
				"logo_url":                 in.LogoURL,
				"logo_dark_url":            in.LogoDarkURL,
				"favicon_url":              in.FaviconURL,
				"contact_email":            in.ContactEmail,
				"contact_phone":            in.ContactPhone,
				"contact_address":          in.ContactAddress,
				"social_links":             social,
				"seo_default_og_image_url": in.SEODefaultOGImageURL,
				"default_locale":           in.DefaultLocale,
				"version":                  gorm.Expr("version + 1"),
				"updated_by":               in.UpdatedBy,
				"updated_at":               now,
			})
		if res.Error != nil {
			return wrapContentDBError(res.Error)
		}
		if res.RowsAffected == 0 {
			var count int64
			if err := tx.Model(&siteSettingsModel{}).Where("id = 1").Count(&count).Error; err != nil {
				return wrapContentDBError(err)
			}
			if count == 0 {
				return ErrNotFound
			}
			return ErrSettingsVersionConflict
		}
		if err := deleteByParent[siteSettingTranslationModel](tx, "site_setting_id", 1); err != nil {
			return wrapContentDBError(err)
		}
		if len(in.Translations) > 0 {
			rows := make([]siteSettingTranslationModel, 0, len(in.Translations))
			for locale, t := range in.Translations {
				rows = append(rows, siteSettingTranslationModel{
					SiteSettingID:         1,
					Locale:                locale,
					Tagline:               t.Tagline,
					FooterText:            t.FooterText,
					SEODefaultTitle:       t.SEODefaultTitle,
					SEODefaultDescription: t.SEODefaultDescription,
					ICPRecord:             t.ICPRecord,
					CreatedAt:             now,
					UpdatedAt:             now,
				})
			}
			if err := tx.Create(&rows).Error; err != nil {
				return wrapContentDBError(err)
			}
		}
		if err := r.setDefaultLocale(tx, in.DefaultLocale); err != nil {
			return wrapContentDBError(err)
		}
		return nil
	})
}

// --- navigation ------------------------------------------------------------

func navigationFromModel(m navigationItemModel) content.NavigationItem {
	return content.NavigationItem{
		ID:           m.ID,
		Placement:    m.Placement,
		ParentID:     m.ParentID,
		URL:          m.URL,
		Target:       m.Target,
		SortOrder:    m.SortOrder,
		Visible:      m.Visible,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		Translations: map[string]string{},
	}
}

// ListNavigation returns navigation items, optionally filtered by placement,
// ordered by sort_order then id. pageSize <= 0 returns every matching row.
func (r *ContentRepository) ListNavigation(ctx context.Context, placement string, page, pageSize int) ([]content.NavigationItem, int64, error) {
	db := r.session(ctx).Model(&navigationItemModel{})
	if p := strings.TrimSpace(placement); p != "" {
		db = db.Where("placement = ?", p)
	}
	models, total, err := listOrdered[navigationItemModel](db, "sort_order, id", page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]content.NavigationItem, 0, len(models))
	ids := make([]int64, 0, len(models))
	for _, m := range models {
		out = append(out, navigationFromModel(m))
		ids = append(ids, m.ID)
	}
	tr, err := r.loadNavigationTranslations(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range out {
		if labels, ok := tr[out[i].ID]; ok {
			out[i].Translations = labels
		}
	}
	return out, total, nil
}

func (r *ContentRepository) loadNavigationTranslations(ctx context.Context, ids []int64) (map[int64]map[string]string, error) {
	out := map[int64]map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []navigationItemTranslationModel
	if err := r.session(ctx).Where("navigation_item_id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	for _, row := range rows {
		if out[row.NavigationItemID] == nil {
			out[row.NavigationItemID] = map[string]string{}
		}
		out[row.NavigationItemID][row.Locale] = row.Label
	}
	return out, nil
}

// GetNavigation loads one navigation item with translations.
func (r *ContentRepository) GetNavigation(ctx context.Context, id int64) (*content.NavigationItem, error) {
	var model navigationItemModel
	if err := r.session(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	item := navigationFromModel(model)
	tr, err := r.loadNavigationTranslations(ctx, []int64{id})
	if err != nil {
		return nil, err
	}
	if labels, ok := tr[id]; ok {
		item.Translations = labels
	}
	return &item, nil
}

// CreateNavigation inserts a navigation item and its translations.
func (r *ContentRepository) CreateNavigation(ctx context.Context, item *content.NavigationItem) error {
	if item == nil {
		return fmt.Errorf("navigation item is nil")
	}
	return r.session(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		model := navigationItemModel{
			Placement: item.Placement,
			ParentID:  item.ParentID,
			URL:       item.URL,
			Target:    item.Target,
			SortOrder: item.SortOrder,
			Visible:   item.Visible,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := tx.Create(&model).Error; err != nil {
			return wrapContentDBError(err)
		}
		item.ID = model.ID
		item.CreatedAt = model.CreatedAt
		item.UpdatedAt = model.UpdatedAt
		return replaceNavigationTranslations(tx, model.ID, item.Translations, now)
	})
}

// UpdateNavigation replaces a navigation item and its translations.
func (r *ContentRepository) UpdateNavigation(ctx context.Context, item *content.NavigationItem) error {
	if item == nil {
		return fmt.Errorf("navigation item is nil")
	}
	return r.session(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		res := tx.Model(&navigationItemModel{}).Where("id = ?", item.ID).
			Updates(map[string]any{
				"placement":  item.Placement,
				"parent_id":  item.ParentID,
				"url":        item.URL,
				"target":     item.Target,
				"sort_order": item.SortOrder,
				"visible":    item.Visible,
				"updated_at": now,
			})
		if res.Error != nil {
			return wrapContentDBError(res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		item.UpdatedAt = now
		return replaceNavigationTranslations(tx, item.ID, item.Translations, now)
	})
}

// DeleteNavigation removes a navigation item (translations cascade; children
// keep their row with parent_id cleared).
func (r *ContentRepository) DeleteNavigation(ctx context.Context, id int64) error {
	res := r.session(ctx).Where("id = ?", id).Delete(&navigationItemModel{})
	if res.Error != nil {
		return wrapContentDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func replaceNavigationTranslations(tx *gorm.DB, id int64, labels map[string]string, now time.Time) error {
	if err := deleteByParent[navigationItemTranslationModel](tx, "navigation_item_id", id); err != nil {
		return wrapContentDBError(err)
	}
	if len(labels) == 0 {
		return nil
	}
	rows := make([]navigationItemTranslationModel, 0, len(labels))
	for locale, label := range labels {
		rows = append(rows, navigationItemTranslationModel{
			NavigationItemID: id,
			Locale:           locale,
			Label:            label,
			CreatedAt:        now,
			UpdatedAt:        now,
		})
	}
	if err := tx.Create(&rows).Error; err != nil {
		return wrapContentDBError(err)
	}
	return nil
}

// --- home sections ---------------------------------------------------------

func homeSectionFromModel(m homeSectionModel) content.HomeSection {
	return content.HomeSection{
		ID:           m.ID,
		Type:         m.Type,
		SortOrder:    m.SortOrder,
		Published:    m.Published,
		Payload:      mapFromJSONBObject(m.Payload),
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		Translations: map[string]map[string]any{},
	}
}

// ListHomeSections returns home sections ordered by sort_order then id.
func (r *ContentRepository) ListHomeSections(ctx context.Context, page, pageSize int) ([]content.HomeSection, int64, error) {
	models, total, err := listOrdered[homeSectionModel](r.session(ctx).Model(&homeSectionModel{}), "sort_order, id", page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]content.HomeSection, 0, len(models))
	ids := make([]int64, 0, len(models))
	for _, m := range models {
		out = append(out, homeSectionFromModel(m))
		ids = append(ids, m.ID)
	}
	tr, err := r.loadHomeTranslations(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range out {
		if payloads, ok := tr[out[i].ID]; ok {
			out[i].Translations = payloads
		}
	}
	return out, total, nil
}

func (r *ContentRepository) loadHomeTranslations(ctx context.Context, ids []int64) (map[int64]map[string]map[string]any, error) {
	out := map[int64]map[string]map[string]any{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []homeSectionTranslationModel
	if err := r.session(ctx).Where("home_section_id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	for _, row := range rows {
		if out[row.HomeSectionID] == nil {
			out[row.HomeSectionID] = map[string]map[string]any{}
		}
		out[row.HomeSectionID][row.Locale] = mapFromJSONBObject(row.Payload)
	}
	return out, nil
}

// GetHomeSection loads one home section with translations.
func (r *ContentRepository) GetHomeSection(ctx context.Context, id int64) (*content.HomeSection, error) {
	var model homeSectionModel
	if err := r.session(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	section := homeSectionFromModel(model)
	tr, err := r.loadHomeTranslations(ctx, []int64{id})
	if err != nil {
		return nil, err
	}
	if payloads, ok := tr[id]; ok {
		section.Translations = payloads
	}
	return &section, nil
}

// CreateHomeSection inserts a home section and its per-locale payloads.
func (r *ContentRepository) CreateHomeSection(ctx context.Context, section *content.HomeSection) error {
	if section == nil {
		return fmt.Errorf("home section is nil")
	}
	payload, err := jsonbObjectFromMap(section.Payload)
	if err != nil {
		return err
	}
	return r.session(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		model := homeSectionModel{
			Type:      section.Type,
			SortOrder: section.SortOrder,
			Published: section.Published,
			Payload:   payload,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := tx.Create(&model).Error; err != nil {
			return wrapContentDBError(err)
		}
		section.ID = model.ID
		section.CreatedAt = now
		section.UpdatedAt = now
		return replaceHomeTranslations(tx, model.ID, section.Translations, now)
	})
}

// UpdateHomeSection replaces a home section and its per-locale payloads.
func (r *ContentRepository) UpdateHomeSection(ctx context.Context, section *content.HomeSection) error {
	if section == nil {
		return fmt.Errorf("home section is nil")
	}
	payload, err := jsonbObjectFromMap(section.Payload)
	if err != nil {
		return err
	}
	return r.session(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		res := tx.Model(&homeSectionModel{}).Where("id = ?", section.ID).
			Updates(map[string]any{
				"type":       section.Type,
				"sort_order": section.SortOrder,
				"published":  section.Published,
				"payload":    payload,
				"updated_at": now,
			})
		if res.Error != nil {
			return wrapContentDBError(res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		section.UpdatedAt = now
		return replaceHomeTranslations(tx, section.ID, section.Translations, now)
	})
}

// DeleteHomeSection removes a home section (translations cascade).
func (r *ContentRepository) DeleteHomeSection(ctx context.Context, id int64) error {
	res := r.session(ctx).Where("id = ?", id).Delete(&homeSectionModel{})
	if res.Error != nil {
		return wrapContentDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func replaceHomeTranslations(tx *gorm.DB, id int64, payloads map[string]map[string]any, now time.Time) error {
	if err := deleteByParent[homeSectionTranslationModel](tx, "home_section_id", id); err != nil {
		return wrapContentDBError(err)
	}
	if len(payloads) == 0 {
		return nil
	}
	rows := make([]homeSectionTranslationModel, 0, len(payloads))
	for locale, payload := range payloads {
		encoded, err := jsonbObjectFromMap(payload)
		if err != nil {
			return err
		}
		rows = append(rows, homeSectionTranslationModel{
			HomeSectionID: id,
			Locale:        locale,
			Payload:       encoded,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}
	if err := tx.Create(&rows).Error; err != nil {
		return wrapContentDBError(err)
	}
	return nil
}

// --- features --------------------------------------------------------------

func featureFromModel(m featureModel) content.Feature {
	return content.Feature{
		ID:           m.ID,
		Icon:         m.Icon,
		SortOrder:    m.SortOrder,
		Published:    m.Published,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		Translations: map[string]content.FeatureTranslation{},
	}
}

// ListFeatures returns features ordered by sort_order then id.
func (r *ContentRepository) ListFeatures(ctx context.Context, page, pageSize int) ([]content.Feature, int64, error) {
	models, total, err := listOrdered[featureModel](r.session(ctx).Model(&featureModel{}), "sort_order, id", page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]content.Feature, 0, len(models))
	ids := make([]int64, 0, len(models))
	for _, m := range models {
		out = append(out, featureFromModel(m))
		ids = append(ids, m.ID)
	}
	tr, err := r.loadFeatureTranslations(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range out {
		if translations, ok := tr[out[i].ID]; ok {
			out[i].Translations = translations
		}
	}
	return out, total, nil
}

func (r *ContentRepository) loadFeatureTranslations(ctx context.Context, ids []int64) (map[int64]map[string]content.FeatureTranslation, error) {
	out := map[int64]map[string]content.FeatureTranslation{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []featureTranslationModel
	if err := r.session(ctx).Where("feature_id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	for _, row := range rows {
		if out[row.FeatureID] == nil {
			out[row.FeatureID] = map[string]content.FeatureTranslation{}
		}
		out[row.FeatureID][row.Locale] = content.FeatureTranslation{
			Title:    row.Title,
			Summary:  row.Summary,
			BodyMD:   row.BodyMD,
			ImageURL: row.ImageURL,
		}
	}
	return out, nil
}

// GetFeature loads one feature with translations.
func (r *ContentRepository) GetFeature(ctx context.Context, id int64) (*content.Feature, error) {
	var model featureModel
	if err := r.session(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	feature := featureFromModel(model)
	tr, err := r.loadFeatureTranslations(ctx, []int64{id})
	if err != nil {
		return nil, err
	}
	if translations, ok := tr[id]; ok {
		feature.Translations = translations
	}
	return &feature, nil
}

// CreateFeature inserts a feature and its translations.
func (r *ContentRepository) CreateFeature(ctx context.Context, feature *content.Feature) error {
	if feature == nil {
		return fmt.Errorf("feature is nil")
	}
	return r.session(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		model := featureModel{
			Icon:      feature.Icon,
			SortOrder: feature.SortOrder,
			Published: feature.Published,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := tx.Create(&model).Error; err != nil {
			return wrapContentDBError(err)
		}
		feature.ID = model.ID
		feature.CreatedAt = now
		feature.UpdatedAt = now
		return replaceFeatureTranslations(tx, model.ID, feature.Translations, now)
	})
}

// UpdateFeature replaces a feature and its translations.
func (r *ContentRepository) UpdateFeature(ctx context.Context, feature *content.Feature) error {
	if feature == nil {
		return fmt.Errorf("feature is nil")
	}
	return r.session(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		res := tx.Model(&featureModel{}).Where("id = ?", feature.ID).
			Updates(map[string]any{
				"icon":       feature.Icon,
				"sort_order": feature.SortOrder,
				"published":  feature.Published,
				"updated_at": now,
			})
		if res.Error != nil {
			return wrapContentDBError(res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		feature.UpdatedAt = now
		return replaceFeatureTranslations(tx, feature.ID, feature.Translations, now)
	})
}

// DeleteFeature removes a feature (translations cascade).
func (r *ContentRepository) DeleteFeature(ctx context.Context, id int64) error {
	res := r.session(ctx).Where("id = ?", id).Delete(&featureModel{})
	if res.Error != nil {
		return wrapContentDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func replaceFeatureTranslations(tx *gorm.DB, id int64, translations map[string]content.FeatureTranslation, now time.Time) error {
	if err := deleteByParent[featureTranslationModel](tx, "feature_id", id); err != nil {
		return wrapContentDBError(err)
	}
	if len(translations) == 0 {
		return nil
	}
	rows := make([]featureTranslationModel, 0, len(translations))
	for locale, t := range translations {
		rows = append(rows, featureTranslationModel{
			FeatureID: id,
			Locale:    locale,
			Title:     t.Title,
			Summary:   t.Summary,
			BodyMD:    t.BodyMD,
			ImageURL:  t.ImageURL,
			CreatedAt: now,
			UpdatedAt: now,
		})
	}
	if err := tx.Create(&rows).Error; err != nil {
		return wrapContentDBError(err)
	}
	return nil
}

// --- pricing plans ---------------------------------------------------------

func pricingPlanFromModel(m pricingPlanModel) content.PricingPlan {
	return content.PricingPlan{
		ID:           m.ID,
		Code:         m.Code,
		MonthlyPrice: moneyFromNumeric(m.MonthlyPrice),
		YearlyPrice:  moneyFromNumeric(m.YearlyPrice),
		Currency:     m.Currency,
		Highlighted:  m.Highlighted,
		SortOrder:    m.SortOrder,
		Visible:      m.Visible,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		Translations: map[string]content.PricingPlanTranslation{},
		Features:     map[string][]string{},
	}
}

// ListPricingPlans returns pricing plans ordered by sort_order then id.
func (r *ContentRepository) ListPricingPlans(ctx context.Context, page, pageSize int) ([]content.PricingPlan, int64, error) {
	models, total, err := listOrdered[pricingPlanModel](r.session(ctx).Model(&pricingPlanModel{}), "sort_order, id", page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]content.PricingPlan, 0, len(models))
	ids := make([]int64, 0, len(models))
	for _, m := range models {
		out = append(out, pricingPlanFromModel(m))
		ids = append(ids, m.ID)
	}
	tr, err := r.loadPlanTranslations(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	feats, err := r.loadPlanFeatures(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range out {
		if translations, ok := tr[out[i].ID]; ok {
			out[i].Translations = translations
		}
		if features, ok := feats[out[i].ID]; ok {
			out[i].Features = features
		}
	}
	return out, total, nil
}

func (r *ContentRepository) loadPlanTranslations(ctx context.Context, ids []int64) (map[int64]map[string]content.PricingPlanTranslation, error) {
	out := map[int64]map[string]content.PricingPlanTranslation{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []pricingPlanTranslationModel
	if err := r.session(ctx).Where("pricing_plan_id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	for _, row := range rows {
		if out[row.PricingPlanID] == nil {
			out[row.PricingPlanID] = map[string]content.PricingPlanTranslation{}
		}
		out[row.PricingPlanID][row.Locale] = content.PricingPlanTranslation{
			Name:        row.Name,
			Description: row.Description,
			CTALabel:    row.CTALabel,
			CTAURL:      row.CTAURL,
		}
	}
	return out, nil
}

func (r *ContentRepository) loadPlanFeatures(ctx context.Context, ids []int64) (map[int64]map[string][]string, error) {
	out := map[int64]map[string][]string{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []pricingPlanFeatureModel
	if err := r.session(ctx).Where("pricing_plan_id IN ?", ids).Order("sort_order, id").Find(&rows).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	for _, row := range rows {
		if out[row.PricingPlanID] == nil {
			out[row.PricingPlanID] = map[string][]string{}
		}
		out[row.PricingPlanID][row.Locale] = append(out[row.PricingPlanID][row.Locale], row.Text)
	}
	return out, nil
}

// GetPricingPlan loads one plan with translations and features.
func (r *ContentRepository) GetPricingPlan(ctx context.Context, id int64) (*content.PricingPlan, error) {
	var model pricingPlanModel
	if err := r.session(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	plan := pricingPlanFromModel(model)
	tr, err := r.loadPlanTranslations(ctx, []int64{id})
	if err != nil {
		return nil, err
	}
	feats, err := r.loadPlanFeatures(ctx, []int64{id})
	if err != nil {
		return nil, err
	}
	if translations, ok := tr[id]; ok {
		plan.Translations = translations
	}
	if features, ok := feats[id]; ok {
		plan.Features = features
	}
	return &plan, nil
}

// CreatePricingPlan inserts a plan with its translations and feature list.
func (r *ContentRepository) CreatePricingPlan(ctx context.Context, plan *content.PricingPlan) error {
	if plan == nil {
		return fmt.Errorf("pricing plan is nil")
	}
	monthly, err := numericFromMoney(plan.MonthlyPrice)
	if err != nil {
		return err
	}
	yearly, err := numericFromMoney(plan.YearlyPrice)
	if err != nil {
		return err
	}
	return r.session(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		model := pricingPlanModel{
			Code:         plan.Code,
			MonthlyPrice: monthly,
			YearlyPrice:  yearly,
			Currency:     plan.Currency,
			Highlighted:  plan.Highlighted,
			SortOrder:    plan.SortOrder,
			Visible:      plan.Visible,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		if err := tx.Create(&model).Error; err != nil {
			return wrapContentDBError(err)
		}
		plan.ID = model.ID
		plan.CreatedAt = now
		plan.UpdatedAt = now
		if err := replacePlanTranslations(tx, model.ID, plan.Translations, now); err != nil {
			return err
		}
		return replacePlanFeatures(tx, model.ID, plan.Features, now)
	})
}

// UpdatePricingPlan replaces a plan with its translations and feature list.
func (r *ContentRepository) UpdatePricingPlan(ctx context.Context, plan *content.PricingPlan) error {
	if plan == nil {
		return fmt.Errorf("pricing plan is nil")
	}
	monthly, err := numericFromMoney(plan.MonthlyPrice)
	if err != nil {
		return err
	}
	yearly, err := numericFromMoney(plan.YearlyPrice)
	if err != nil {
		return err
	}
	return r.session(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		res := tx.Model(&pricingPlanModel{}).Where("id = ?", plan.ID).
			Updates(map[string]any{
				"code":          plan.Code,
				"monthly_price": monthly,
				"yearly_price":  yearly,
				"currency":      plan.Currency,
				"highlighted":   plan.Highlighted,
				"sort_order":    plan.SortOrder,
				"visible":       plan.Visible,
				"updated_at":    now,
			})
		if res.Error != nil {
			return wrapContentDBError(res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		plan.UpdatedAt = now
		if err := replacePlanTranslations(tx, plan.ID, plan.Translations, now); err != nil {
			return err
		}
		return replacePlanFeatures(tx, plan.ID, plan.Features, now)
	})
}

// DeletePricingPlan removes a plan (translations and features cascade).
func (r *ContentRepository) DeletePricingPlan(ctx context.Context, id int64) error {
	res := r.session(ctx).Where("id = ?", id).Delete(&pricingPlanModel{})
	if res.Error != nil {
		return wrapContentDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func replacePlanTranslations(tx *gorm.DB, id int64, translations map[string]content.PricingPlanTranslation, now time.Time) error {
	if err := deleteByParent[pricingPlanTranslationModel](tx, "pricing_plan_id", id); err != nil {
		return wrapContentDBError(err)
	}
	if len(translations) == 0 {
		return nil
	}
	rows := make([]pricingPlanTranslationModel, 0, len(translations))
	for locale, t := range translations {
		rows = append(rows, pricingPlanTranslationModel{
			PricingPlanID: id,
			Locale:        locale,
			Name:          t.Name,
			Description:   t.Description,
			CTALabel:      t.CTALabel,
			CTAURL:        t.CTAURL,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}
	if err := tx.Create(&rows).Error; err != nil {
		return wrapContentDBError(err)
	}
	return nil
}

func replacePlanFeatures(tx *gorm.DB, id int64, features map[string][]string, now time.Time) error {
	if err := deleteByParent[pricingPlanFeatureModel](tx, "pricing_plan_id", id); err != nil {
		return wrapContentDBError(err)
	}
	total := 0
	for _, list := range features {
		total += len(list)
	}
	if total == 0 {
		return nil
	}
	rows := make([]pricingPlanFeatureModel, 0, total)
	for locale, list := range features {
		for i, text := range list {
			rows = append(rows, pricingPlanFeatureModel{
				PricingPlanID: id,
				Locale:        locale,
				SortOrder:     i + 1,
				Text:          text,
				CreatedAt:     now,
			})
		}
	}
	if err := tx.Create(&rows).Error; err != nil {
		return wrapContentDBError(err)
	}
	return nil
}

// --- pages -----------------------------------------------------------------

func pageFromModel(m pageModel) content.Page {
	return content.Page{
		ID:           m.ID,
		Slug:         m.Slug,
		Published:    m.Published,
		SortOrder:    m.SortOrder,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		Translations: map[string]content.PageTranslation{},
	}
}

// ListPages returns pages ordered by sort_order then id.
func (r *ContentRepository) ListPages(ctx context.Context, page, pageSize int) ([]content.Page, int64, error) {
	models, total, err := listOrdered[pageModel](r.session(ctx).Model(&pageModel{}), "sort_order, id", page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]content.Page, 0, len(models))
	ids := make([]int64, 0, len(models))
	for _, m := range models {
		out = append(out, pageFromModel(m))
		ids = append(ids, m.ID)
	}
	tr, err := r.loadPageTranslations(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range out {
		if translations, ok := tr[out[i].ID]; ok {
			out[i].Translations = translations
		}
	}
	return out, total, nil
}

func (r *ContentRepository) loadPageTranslations(ctx context.Context, ids []int64) (map[int64]map[string]content.PageTranslation, error) {
	out := map[int64]map[string]content.PageTranslation{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []pageTranslationModel
	if err := r.session(ctx).Where("page_id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	for _, row := range rows {
		if out[row.PageID] == nil {
			out[row.PageID] = map[string]content.PageTranslation{}
		}
		out[row.PageID][row.Locale] = content.PageTranslation{
			Title:          row.Title,
			BodyMD:         row.BodyMD,
			SEOTitle:       row.SEOTitle,
			SEODescription: row.SEODescription,
		}
	}
	return out, nil
}

// GetPage loads one page by id with translations.
func (r *ContentRepository) GetPage(ctx context.Context, id int64) (*content.Page, error) {
	var model pageModel
	if err := r.session(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	return r.pageWithTranslations(ctx, model)
}

// GetPageBySlug loads one page by slug with translations.
func (r *ContentRepository) GetPageBySlug(ctx context.Context, slug string) (*content.Page, error) {
	var model pageModel
	if err := r.session(ctx).Where("slug = ?", slug).First(&model).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	return r.pageWithTranslations(ctx, model)
}

func (r *ContentRepository) pageWithTranslations(ctx context.Context, model pageModel) (*content.Page, error) {
	page := pageFromModel(model)
	tr, err := r.loadPageTranslations(ctx, []int64{model.ID})
	if err != nil {
		return nil, err
	}
	if translations, ok := tr[model.ID]; ok {
		page.Translations = translations
	}
	return &page, nil
}

// CreatePage inserts a page and its translations.
func (r *ContentRepository) CreatePage(ctx context.Context, page *content.Page) error {
	if page == nil {
		return fmt.Errorf("page is nil")
	}
	return r.session(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		model := pageModel{
			Slug:      page.Slug,
			Published: page.Published,
			SortOrder: page.SortOrder,
			CreatedAt: now,
			UpdatedAt: now,
		}
		if err := tx.Create(&model).Error; err != nil {
			return wrapContentDBError(err)
		}
		page.ID = model.ID
		page.CreatedAt = now
		page.UpdatedAt = now
		return replacePageTranslations(tx, model.ID, page.Translations, now)
	})
}

// UpdatePage replaces a page and its translations.
func (r *ContentRepository) UpdatePage(ctx context.Context, page *content.Page) error {
	if page == nil {
		return fmt.Errorf("page is nil")
	}
	return r.session(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		res := tx.Model(&pageModel{}).Where("id = ?", page.ID).
			Updates(map[string]any{
				"slug":       page.Slug,
				"published":  page.Published,
				"sort_order": page.SortOrder,
				"updated_at": now,
			})
		if res.Error != nil {
			return wrapContentDBError(res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		page.UpdatedAt = now
		return replacePageTranslations(tx, page.ID, page.Translations, now)
	})
}

// DeletePage removes a page (translations cascade).
func (r *ContentRepository) DeletePage(ctx context.Context, id int64) error {
	res := r.session(ctx).Where("id = ?", id).Delete(&pageModel{})
	if res.Error != nil {
		return wrapContentDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func replacePageTranslations(tx *gorm.DB, id int64, translations map[string]content.PageTranslation, now time.Time) error {
	if err := deleteByParent[pageTranslationModel](tx, "page_id", id); err != nil {
		return wrapContentDBError(err)
	}
	if len(translations) == 0 {
		return nil
	}
	rows := make([]pageTranslationModel, 0, len(translations))
	for locale, t := range translations {
		rows = append(rows, pageTranslationModel{
			PageID:         id,
			Locale:         locale,
			Title:          t.Title,
			BodyMD:         t.BodyMD,
			SEOTitle:       t.SEOTitle,
			SEODescription: t.SEODescription,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}
	if err := tx.Create(&rows).Error; err != nil {
		return wrapContentDBError(err)
	}
	return nil
}

// --- documentation ---------------------------------------------------------

func docCategoryFromModel(m docCategoryModel) content.DocCategory {
	return content.DocCategory{
		ID:           m.ID,
		Slug:         m.Slug,
		SortOrder:    m.SortOrder,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		Translations: map[string]string{},
	}
}

// ListDocCategories returns documentation categories ordered by sort_order.
func (r *ContentRepository) ListDocCategories(ctx context.Context, page, pageSize int) ([]content.DocCategory, int64, error) {
	models, total, err := listOrdered[docCategoryModel](r.session(ctx).Model(&docCategoryModel{}), "sort_order, id", page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]content.DocCategory, 0, len(models))
	ids := make([]int64, 0, len(models))
	for _, m := range models {
		out = append(out, docCategoryFromModel(m))
		ids = append(ids, m.ID)
	}
	tr, err := r.loadCategoryNames(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range out {
		if names, ok := tr[out[i].ID]; ok {
			out[i].Translations = names
		}
	}
	return out, total, nil
}

func (r *ContentRepository) loadCategoryNames(ctx context.Context, ids []int64) (map[int64]map[string]string, error) {
	out := map[int64]map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []docCategoryTranslationModel
	if err := r.session(ctx).Where("doc_category_id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	for _, row := range rows {
		if out[row.DocCategoryID] == nil {
			out[row.DocCategoryID] = map[string]string{}
		}
		out[row.DocCategoryID][row.Locale] = row.Name
	}
	return out, nil
}

// GetDocCategory loads one category by id with translations.
func (r *ContentRepository) GetDocCategory(ctx context.Context, id int64) (*content.DocCategory, error) {
	var model docCategoryModel
	if err := r.session(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	category := docCategoryFromModel(model)
	tr, err := r.loadCategoryNames(ctx, []int64{id})
	if err != nil {
		return nil, err
	}
	if names, ok := tr[id]; ok {
		category.Translations = names
	}
	return &category, nil
}

// CreateDocCategory inserts a category and its translations.
func (r *ContentRepository) CreateDocCategory(ctx context.Context, category *content.DocCategory) error {
	if category == nil {
		return fmt.Errorf("doc category is nil")
	}
	return r.session(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		model := docCategoryModel{Slug: category.Slug, SortOrder: category.SortOrder, CreatedAt: now, UpdatedAt: now}
		if err := tx.Create(&model).Error; err != nil {
			return wrapContentDBError(err)
		}
		category.ID = model.ID
		category.CreatedAt = now
		category.UpdatedAt = now
		return replaceCategoryTranslations(tx, model.ID, category.Translations, now)
	})
}

// UpdateDocCategory replaces a category and its translations.
func (r *ContentRepository) UpdateDocCategory(ctx context.Context, category *content.DocCategory) error {
	if category == nil {
		return fmt.Errorf("doc category is nil")
	}
	return r.session(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		res := tx.Model(&docCategoryModel{}).Where("id = ?", category.ID).
			Updates(map[string]any{"slug": category.Slug, "sort_order": category.SortOrder, "updated_at": now})
		if res.Error != nil {
			return wrapContentDBError(res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		category.UpdatedAt = now
		return replaceCategoryTranslations(tx, category.ID, category.Translations, now)
	})
}

// DeleteDocCategory removes a category; a category still referenced by
// articles is rejected as a conflict (doc_articles.category_id RESTRICT).
func (r *ContentRepository) DeleteDocCategory(ctx context.Context, id int64) error {
	res := r.session(ctx).Where("id = ?", id).Delete(&docCategoryModel{})
	if res.Error != nil {
		return wrapContentDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func replaceCategoryTranslations(tx *gorm.DB, id int64, names map[string]string, now time.Time) error {
	if err := deleteByParent[docCategoryTranslationModel](tx, "doc_category_id", id); err != nil {
		return wrapContentDBError(err)
	}
	if len(names) == 0 {
		return nil
	}
	rows := make([]docCategoryTranslationModel, 0, len(names))
	for locale, name := range names {
		rows = append(rows, docCategoryTranslationModel{
			DocCategoryID: id,
			Locale:        locale,
			Name:          name,
			CreatedAt:     now,
			UpdatedAt:     now,
		})
	}
	if err := tx.Create(&rows).Error; err != nil {
		return wrapContentDBError(err)
	}
	return nil
}

func docArticleFromModel(m docArticleModel) content.DocArticle {
	return content.DocArticle{
		ID:           m.ID,
		Slug:         m.Slug,
		CategoryID:   m.CategoryID,
		SortOrder:    m.SortOrder,
		Published:    m.Published,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
		Translations: map[string]content.DocArticleTranslation{},
	}
}

// ListDocArticles returns articles ordered by category, sort_order, then id.
func (r *ContentRepository) ListDocArticles(ctx context.Context, page, pageSize int) ([]content.DocArticle, int64, error) {
	models, total, err := listOrdered[docArticleModel](r.session(ctx).Model(&docArticleModel{}), "category_id, sort_order, id", page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]content.DocArticle, 0, len(models))
	ids := make([]int64, 0, len(models))
	for _, m := range models {
		out = append(out, docArticleFromModel(m))
		ids = append(ids, m.ID)
	}
	tr, err := r.loadArticleTranslations(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	for i := range out {
		if translations, ok := tr[out[i].ID]; ok {
			out[i].Translations = translations
		}
	}
	return out, total, nil
}

func (r *ContentRepository) loadArticleTranslations(ctx context.Context, ids []int64) (map[int64]map[string]content.DocArticleTranslation, error) {
	out := map[int64]map[string]content.DocArticleTranslation{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []docArticleTranslationModel
	if err := r.session(ctx).Where("doc_article_id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	for _, row := range rows {
		if out[row.DocArticleID] == nil {
			out[row.DocArticleID] = map[string]content.DocArticleTranslation{}
		}
		out[row.DocArticleID][row.Locale] = content.DocArticleTranslation{
			Title:          row.Title,
			BodyMD:         row.BodyMD,
			SEOTitle:       row.SEOTitle,
			SEODescription: row.SEODescription,
		}
	}
	return out, nil
}

// GetDocArticle loads one article by id with translations.
func (r *ContentRepository) GetDocArticle(ctx context.Context, id int64) (*content.DocArticle, error) {
	var model docArticleModel
	if err := r.session(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	return r.articleWithTranslations(ctx, model)
}

// GetDocArticleBySlug loads one article by slug with translations.
func (r *ContentRepository) GetDocArticleBySlug(ctx context.Context, slug string) (*content.DocArticle, error) {
	var model docArticleModel
	if err := r.session(ctx).Where("slug = ?", slug).First(&model).Error; err != nil {
		return nil, wrapContentDBError(err)
	}
	return r.articleWithTranslations(ctx, model)
}

func (r *ContentRepository) articleWithTranslations(ctx context.Context, model docArticleModel) (*content.DocArticle, error) {
	article := docArticleFromModel(model)
	tr, err := r.loadArticleTranslations(ctx, []int64{model.ID})
	if err != nil {
		return nil, err
	}
	if translations, ok := tr[model.ID]; ok {
		article.Translations = translations
	}
	return &article, nil
}

// CreateDocArticle inserts an article and its translations.
func (r *ContentRepository) CreateDocArticle(ctx context.Context, article *content.DocArticle) error {
	if article == nil {
		return fmt.Errorf("doc article is nil")
	}
	return r.session(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		model := docArticleModel{
			Slug:       article.Slug,
			CategoryID: article.CategoryID,
			SortOrder:  article.SortOrder,
			Published:  article.Published,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		if err := tx.Create(&model).Error; err != nil {
			return wrapContentDBError(err)
		}
		article.ID = model.ID
		article.CreatedAt = now
		article.UpdatedAt = now
		return replaceArticleTranslations(tx, model.ID, article.Translations, now)
	})
}

// UpdateDocArticle replaces an article and its translations.
func (r *ContentRepository) UpdateDocArticle(ctx context.Context, article *content.DocArticle) error {
	if article == nil {
		return fmt.Errorf("doc article is nil")
	}
	return r.session(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		res := tx.Model(&docArticleModel{}).Where("id = ?", article.ID).
			Updates(map[string]any{
				"slug":        article.Slug,
				"category_id": article.CategoryID,
				"sort_order":  article.SortOrder,
				"published":   article.Published,
				"updated_at":  now,
			})
		if res.Error != nil {
			return wrapContentDBError(res.Error)
		}
		if res.RowsAffected == 0 {
			return ErrNotFound
		}
		article.UpdatedAt = now
		return replaceArticleTranslations(tx, article.ID, article.Translations, now)
	})
}

// DeleteDocArticle removes an article (translations cascade).
func (r *ContentRepository) DeleteDocArticle(ctx context.Context, id int64) error {
	res := r.session(ctx).Where("id = ?", id).Delete(&docArticleModel{})
	if res.Error != nil {
		return wrapContentDBError(res.Error)
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func replaceArticleTranslations(tx *gorm.DB, id int64, translations map[string]content.DocArticleTranslation, now time.Time) error {
	if err := deleteByParent[docArticleTranslationModel](tx, "doc_article_id", id); err != nil {
		return wrapContentDBError(err)
	}
	if len(translations) == 0 {
		return nil
	}
	rows := make([]docArticleTranslationModel, 0, len(translations))
	for locale, t := range translations {
		rows = append(rows, docArticleTranslationModel{
			DocArticleID:   id,
			Locale:         locale,
			Title:          t.Title,
			BodyMD:         t.BodyMD,
			SEOTitle:       t.SEOTitle,
			SEODescription: t.SEODescription,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}
	if err := tx.Create(&rows).Error; err != nil {
		return wrapContentDBError(err)
	}
	return nil
}
