-- Business-user management, user-level, and system-settings permissions.
--
-- super_admin receives every new permission. Only the super_admin role is
-- seeded on a fresh database (work order admin-platform-startup-20260907);
-- other roles get permissions through role management. Level management and
-- system-settings mutation are super_admin-only (the manage permission
-- defaults to super_admin).
INSERT INTO permissions (code, name, description)
VALUES
    ('admin.customer.read', 'Read business users', 'List and view business user accounts'),
    ('admin.customer.create', 'Create business users', 'Create business user accounts'),
    ('admin.customer.update', 'Update business users', 'Update business user profiles'),
    ('admin.customer.disable', 'Enable or disable business users', 'Enable or disable business user accounts'),
    ('admin.customer.reset_password', 'Reset business user passwords', 'Reset a business user password'),
    ('admin.customer.points', 'Adjust business user points', 'Adjust available and cumulative consumption points with ledger'),
    ('admin.customer.level_assign', 'Assign business user levels', 'Assign a manual level or switch back to automatic calculation'),
    ('admin.user_level.read', 'Read user levels', 'List and view business user levels'),
    ('admin.user_level.manage', 'Manage user levels', 'Create and update business user levels'),
    ('admin.system_settings.read', 'Read system settings', 'List and view platform system settings'),
    ('admin.system_settings.manage', 'Manage system settings', 'Update platform login, registration, and SMTP settings')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE lower(r.code) = 'super_admin'
  AND p.code IN (
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
ON CONFLICT DO NOTHING;