-- Remove business-user management role assignments first (safe for both built-in
-- and custom roles), then the permission rows.
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE code IN (
        'admin.customer.read',
        'admin.customer.create',
        'admin.customer.update',
        'admin.customer.disable',
        'admin.customer.reset_password',
        'admin.customer.points',
        'admin.customer.level_assign',
        'admin.user_level.read',
        'admin.user_level.manage',
        'admin.system_settings.read',
        'admin.system_settings.manage'
    )
);

DELETE FROM permissions WHERE code IN (
    'admin.customer.read',
    'admin.customer.create',
    'admin.customer.update',
    'admin.customer.disable',
    'admin.customer.reset_password',
    'admin.customer.points',
    'admin.customer.level_assign',
    'admin.user_level.read',
    'admin.user_level.manage',
    'admin.system_settings.read',
    'admin.system_settings.manage'
);

DELETE FROM schema_migrations WHERE version = '000007_user_rbac_permissions';
