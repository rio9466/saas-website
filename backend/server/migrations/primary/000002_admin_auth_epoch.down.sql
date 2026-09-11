ALTER TABLE administrators
    DROP COLUMN IF EXISTS auth_epoch;

DELETE FROM schema_migrations WHERE version = '000002_admin_auth_epoch';
