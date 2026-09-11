CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE administrators (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT administrators_username_nonempty CHECK (char_length(btrim(username)) > 0)
);

CREATE UNIQUE INDEX administrators_username_normalized ON administrators ((lower(btrim(username))));

CREATE TABLE roles (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    built_in BOOLEAN NOT NULL DEFAULT FALSE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT roles_code_nonempty CHECK (char_length(btrim(code)) > 0)
);

CREATE UNIQUE INDEX roles_code_normalized ON roles ((lower(btrim(code))));

CREATE TABLE permissions (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT permissions_code_nonempty CHECK (char_length(btrim(code)) > 0)
);

CREATE UNIQUE INDEX permissions_code_normalized ON permissions ((lower(btrim(code))));

CREATE TABLE administrator_roles (
    administrator_id BIGINT NOT NULL REFERENCES administrators(id) ON DELETE CASCADE,
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (administrator_id, role_id)
);

CREATE TABLE role_permissions (
    role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX idx_administrators_enabled ON administrators (enabled);
CREATE INDEX idx_roles_built_in ON roles (built_in);
CREATE INDEX idx_administrator_roles_role_id ON administrator_roles (role_id);
CREATE INDEX idx_role_permissions_permission_id ON role_permissions (permission_id);

-- Built-in role seeds: only super_admin is initialized on a fresh database
-- (work order admin-platform-startup-20260907). Other roles — including the
-- reserved "admin" and "finance" codes — are created through role management
-- when needed. Databases that already applied the previous version of this
-- migration keep their existing roles; nothing is ever deleted here.
INSERT INTO roles (code, name, description, built_in, enabled)
VALUES
    ('super_admin', 'Super Administrator', 'Unrestricted administrative access', TRUE, TRUE)
ON CONFLICT DO NOTHING;

INSERT INTO permissions (code, name, description)
VALUES
    ('dashboard.view', 'View dashboard', 'View the operations dashboard'),
    ('admin.user.read', 'Read administrators', 'List and view administrator accounts'),
    ('admin.user.create', 'Create administrators', 'Create administrator accounts'),
    ('admin.user.update', 'Update administrators', 'Update administrator profiles'),
    ('admin.user.disable', 'Enable or disable administrators', 'Enable or disable administrator accounts'),
    ('admin.user.reset_password', 'Reset administrator passwords', 'Reset another administrator password'),
    ('admin.user.assign_role', 'Assign administrator roles', 'Assign non-super roles to administrators'),
    ('admin.role.read', 'Read roles', 'List and view roles and permissions'),
    ('admin.role.manage', 'Manage roles', 'Create and update custom roles and permission assignments'),
    ('audit.log.read', 'Read audit logs', 'List and view durable audit and security events')
ON CONFLICT DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE lower(r.code) = 'super_admin'
ON CONFLICT DO NOTHING;
