-- Typed singleton system settings. Exactly one row (id = 1). SMTP passwords
-- are stored only as random-nonce AES-256-GCM ciphertext (application
-- boundary encryption); the master key lives in environment/config secrets.
CREATE TABLE system_settings (
    id BIGINT PRIMARY KEY CONSTRAINT system_settings_singleton CHECK (id = 1),
    platform_name TEXT NOT NULL,
    public_frontend_url TEXT NOT NULL,
    public_api_url TEXT NOT NULL,
    registration_enabled BOOLEAN NOT NULL,
    username_login_enabled BOOLEAN NOT NULL,
    email_login_enabled BOOLEAN NOT NULL,
    email_verification_required BOOLEAN NOT NULL,
    default_level_id BIGINT NOT NULL REFERENCES user_levels(id),
    default_avatar_url TEXT NOT NULL DEFAULT '',
    registration_points NUMERIC(20,4) NOT NULL DEFAULT 0.0000,
    smtp_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    smtp_host TEXT NOT NULL DEFAULT '',
    smtp_port INTEGER NOT NULL DEFAULT 0,
    smtp_username TEXT NOT NULL DEFAULT '',
    smtp_password_encrypted TEXT NOT NULL DEFAULT '',
    smtp_from_email TEXT NOT NULL DEFAULT '',
    smtp_from_name TEXT NOT NULL DEFAULT '',
    smtp_tls_mode TEXT NOT NULL DEFAULT 'starttls',
    version INTEGER NOT NULL DEFAULT 1,
    updated_by BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT system_settings_platform_name_nonempty CHECK (char_length(btrim(platform_name)) > 0),
    CONSTRAINT system_settings_frontend_url_nonempty CHECK (char_length(btrim(public_frontend_url)) > 0),
    CONSTRAINT system_settings_api_url_nonempty CHECK (char_length(btrim(public_api_url)) > 0),
    CONSTRAINT system_settings_registration_points_nonnegative CHECK (registration_points >= 0),
    CONSTRAINT system_settings_port_range CHECK (smtp_port >= 0 AND smtp_port <= 65535),
    CONSTRAINT system_settings_tls_mode_valid CHECK (smtp_tls_mode IN ('none', 'starttls', 'ssl'))
);

-- Seed the singleton with safe development defaults. The default level id is
-- resolved from the seeded 'default' level.
INSERT INTO system_settings (
    id, platform_name, public_frontend_url, public_api_url,
    registration_enabled, username_login_enabled, email_login_enabled,
    email_verification_required, default_level_id, default_avatar_url,
    registration_points, updated_by
)
SELECT
    1,
    'easy-admin',
    'http://localhost:3000',
    'http://127.0.0.1:8080',
    TRUE, TRUE, TRUE,
    FALSE,
    id,
    '',
    0.0000,
    0
FROM user_levels
WHERE lower(btrim(code)) = 'default'
ON CONFLICT DO NOTHING;