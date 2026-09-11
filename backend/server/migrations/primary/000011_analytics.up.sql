-- Page-view analytics (BE-ANALYTICS).
--
-- page_views stores one row per public page-view report (contract §4.10).
-- source is the resolved traffic source: the fixed string 'direct' or the
-- referrer host (subdomain preserved, no domain classification). locale is
-- optional. The indexes support the range scan and the source aggregation of
-- the administrator overview endpoint (contract §6.3).

CREATE TABLE page_views (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    path TEXT NOT NULL,
    source TEXT NOT NULL,
    locale TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT page_views_path_prefix CHECK (path LIKE '/%'),
    CONSTRAINT page_views_path_length CHECK (char_length(path) <= 512),
    CONSTRAINT page_views_source_nonempty CHECK (char_length(btrim(source)) > 0)
);

CREATE INDEX page_views_created_idx ON page_views (created_at);
CREATE INDEX page_views_source_created_idx ON page_views (source, created_at);

-- Analytics read permission, granted to the built-in super_admin role
-- (migration 000009 pattern); other roles are granted through role management.
INSERT INTO permissions (code, name, description)
VALUES ('admin.analytics.read', 'Read site analytics', 'View aggregated page views and traffic sources')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE lower(r.code) = 'super_admin'
  AND p.code = 'admin.analytics.read'
ON CONFLICT DO NOTHING;
