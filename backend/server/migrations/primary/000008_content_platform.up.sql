-- Content platform: site branding/settings, supported locales, navigation,
-- home sections, features, pricing plans, static pages, and documentation.
--
-- Conventions (see docs/tasks/be-01-content-foundation.md):
--   * every entity has a translation table keyed by (entity_id, locale);
--   * money uses NUMERIC; only pure translation tables cascade on delete;
--   * there is exactly one default locale, and it must stay enabled.

-- Supported locales are declared before site_settings so the default locale is
-- referentially valid (a locale may not be deleted while it is the default).
CREATE TABLE supported_locales (
    code TEXT PRIMARY KEY,
    label TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT supported_locales_code_nonempty CHECK (char_length(btrim(code)) > 0),
    CONSTRAINT supported_locales_label_nonempty CHECK (char_length(btrim(label)) > 0),
    -- The default locale must be enabled; at most one row can be the default.
    CONSTRAINT supported_locales_default_enabled CHECK (NOT is_default OR enabled)
);

CREATE UNIQUE INDEX supported_locales_single_default
    ON supported_locales (is_default) WHERE is_default;

INSERT INTO supported_locales (code, label, enabled, sort_order, is_default) VALUES
    ('en', 'English', TRUE, 1, TRUE),
    ('zh-CN', '简体中文', TRUE, 2, FALSE)
ON CONFLICT (code) DO NOTHING;

-- Singleton branding/SEO/contact settings (id = 1) with an optimistic lock.
CREATE TABLE site_settings (
    id BIGINT PRIMARY KEY CONSTRAINT site_settings_singleton CHECK (id = 1),
    site_name TEXT NOT NULL,
    logo_url TEXT NOT NULL DEFAULT '',
    logo_dark_url TEXT NOT NULL DEFAULT '',
    favicon_url TEXT NOT NULL DEFAULT '',
    contact_email TEXT NOT NULL DEFAULT '',
    contact_phone TEXT NOT NULL DEFAULT '',
    contact_address TEXT NOT NULL DEFAULT '',
    social_links JSONB NOT NULL DEFAULT '[]'::jsonb,
    seo_default_og_image_url TEXT NOT NULL DEFAULT '',
    default_locale TEXT NOT NULL DEFAULT 'en' REFERENCES supported_locales(code),
    version INTEGER NOT NULL DEFAULT 1,
    updated_by BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT site_settings_site_name_nonempty CHECK (char_length(btrim(site_name)) > 0),
    CONSTRAINT site_settings_version_positive CHECK (version >= 1),
    CONSTRAINT site_settings_social_links_array CHECK (jsonb_typeof(social_links) = 'array')
);

INSERT INTO site_settings (id, site_name, default_locale, updated_by)
VALUES (1, 'SaaS Website', 'en', 0)
ON CONFLICT (id) DO NOTHING;

CREATE TABLE site_setting_translations (
    site_setting_id BIGINT NOT NULL REFERENCES site_settings(id) ON DELETE CASCADE,
    locale TEXT NOT NULL,
    tagline TEXT NOT NULL DEFAULT '',
    footer_text TEXT NOT NULL DEFAULT '',
    seo_default_title TEXT NOT NULL DEFAULT '',
    seo_default_description TEXT NOT NULL DEFAULT '',
    icp_record TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (site_setting_id, locale)
);

-- Header/footer navigation, optionally nested one or more levels.
CREATE TABLE navigation_items (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    placement TEXT NOT NULL,
    parent_id BIGINT REFERENCES navigation_items(id) ON DELETE SET NULL,
    url TEXT NOT NULL DEFAULT '',
    target TEXT NOT NULL DEFAULT '_self',
    sort_order INTEGER NOT NULL DEFAULT 0,
    visible BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT navigation_items_placement_valid CHECK (placement IN ('header', 'footer')),
    CONSTRAINT navigation_items_target_valid CHECK (target IN ('_self', '_blank')),
    CONSTRAINT navigation_items_no_self_parent CHECK (parent_id IS NULL OR parent_id <> id)
);

CREATE INDEX navigation_items_placement_sort_idx
    ON navigation_items (placement, sort_order, id);

CREATE TABLE navigation_item_translations (
    navigation_item_id BIGINT NOT NULL REFERENCES navigation_items(id) ON DELETE CASCADE,
    locale TEXT NOT NULL,
    label TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (navigation_item_id, locale)
);

-- Home sections: `payload` holds non-translatable/base data; the translation
-- table holds per-locale overrides merged over the base for public reads.
CREATE TABLE home_sections (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    type TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    published BOOLEAN NOT NULL DEFAULT FALSE,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT home_sections_type_nonempty CHECK (char_length(btrim(type)) > 0),
    CONSTRAINT home_sections_payload_object CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX home_sections_sort_idx ON home_sections (sort_order, id);

CREATE TABLE home_section_translations (
    home_section_id BIGINT NOT NULL REFERENCES home_sections(id) ON DELETE CASCADE,
    locale TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (home_section_id, locale),
    CONSTRAINT home_section_translations_payload_object CHECK (jsonb_typeof(payload) = 'object')
);

CREATE TABLE features (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    icon TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    published BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX features_sort_idx ON features (sort_order, id);

CREATE TABLE feature_translations (
    feature_id BIGINT NOT NULL REFERENCES features(id) ON DELETE CASCADE,
    locale TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    body_md TEXT NOT NULL DEFAULT '',
    image_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (feature_id, locale)
);

CREATE TABLE pricing_plans (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    monthly_price NUMERIC(20,2) NOT NULL DEFAULT 0,
    yearly_price NUMERIC(20,2) NOT NULL DEFAULT 0,
    currency TEXT NOT NULL DEFAULT 'USD',
    highlighted BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    visible BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT pricing_plans_code_nonempty CHECK (char_length(btrim(code)) > 0),
    CONSTRAINT pricing_plans_prices_nonnegative CHECK (monthly_price >= 0 AND yearly_price >= 0)
);

CREATE INDEX pricing_plans_sort_idx ON pricing_plans (sort_order, id);

CREATE TABLE pricing_plan_translations (
    pricing_plan_id BIGINT NOT NULL REFERENCES pricing_plans(id) ON DELETE CASCADE,
    locale TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    cta_label TEXT NOT NULL DEFAULT '',
    cta_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (pricing_plan_id, locale)
);

-- Per-locale bullet list for a plan.
CREATE TABLE pricing_plan_features (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    pricing_plan_id BIGINT NOT NULL REFERENCES pricing_plans(id) ON DELETE CASCADE,
    locale TEXT NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    text TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX pricing_plan_features_plan_locale_idx
    ON pricing_plan_features (pricing_plan_id, locale, sort_order, id);

CREATE TABLE pages (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    published BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT pages_slug_nonempty CHECK (char_length(btrim(slug)) > 0)
);

CREATE INDEX pages_sort_idx ON pages (sort_order, id);

CREATE TABLE page_translations (
    page_id BIGINT NOT NULL REFERENCES pages(id) ON DELETE CASCADE,
    locale TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    body_md TEXT NOT NULL DEFAULT '',
    seo_title TEXT NOT NULL DEFAULT '',
    seo_description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (page_id, locale)
);

CREATE TABLE doc_categories (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT doc_categories_slug_nonempty CHECK (char_length(btrim(slug)) > 0)
);

CREATE INDEX doc_categories_sort_idx ON doc_categories (sort_order, id);

CREATE TABLE doc_category_translations (
    doc_category_id BIGINT NOT NULL REFERENCES doc_categories(id) ON DELETE CASCADE,
    locale TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (doc_category_id, locale)
);

CREATE TABLE doc_articles (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    category_id BIGINT NOT NULL REFERENCES doc_categories(id) ON DELETE RESTRICT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    published BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT doc_articles_slug_nonempty CHECK (char_length(btrim(slug)) > 0)
);

CREATE INDEX doc_articles_category_sort_idx ON doc_articles (category_id, sort_order, id);

CREATE TABLE doc_article_translations (
    doc_article_id BIGINT NOT NULL REFERENCES doc_articles(id) ON DELETE CASCADE,
    locale TEXT NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    body_md TEXT NOT NULL DEFAULT '',
    seo_title TEXT NOT NULL DEFAULT '',
    seo_description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (doc_article_id, locale)
);
