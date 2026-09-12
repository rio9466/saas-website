-- Reverse of 000011_analytics.up.sql. Idempotent so it can be re-run.
DROP TABLE IF EXISTS page_views;

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE code = 'admin.analytics.read'
);

DELETE FROM permissions WHERE code = 'admin.analytics.read';

DELETE FROM schema_migrations WHERE version = '000011_analytics';
