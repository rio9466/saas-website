DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS administrator_roles;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS administrators;
DELETE FROM schema_migrations WHERE version = '000001_admin_rbac';
