-- Content-management and contact-inbox permissions. super_admin receives all
-- new permissions (migration 000007 pattern); other roles are granted through
-- role management.
INSERT INTO permissions (code, name, description)
VALUES
    ('admin.content.read', 'Read site content', 'List and view public site content resources'),
    ('admin.content.manage', 'Manage site content', 'Create, update, and delete public site content resources'),
    ('admin.contact.read', 'Read contact submissions', 'List and view contact form submissions'),
    ('admin.contact.manage', 'Manage contact submissions', 'Update contact submission status')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE lower(r.code) = 'super_admin'
  AND p.code IN (
    'admin.content.read',
    'admin.content.manage',
    'admin.contact.read',
    'admin.contact.manage'
  )
ON CONFLICT DO NOTHING;
