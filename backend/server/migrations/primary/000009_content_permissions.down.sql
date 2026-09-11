DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM permissions WHERE code IN (
        'admin.content.read',
        'admin.content.manage',
        'admin.contact.read',
        'admin.contact.manage'
    )
);

DELETE FROM permissions WHERE code IN (
    'admin.content.read',
    'admin.content.manage',
    'admin.contact.read',
    'admin.contact.manage'
);

DELETE FROM schema_migrations WHERE version = '000009_content_permissions';
