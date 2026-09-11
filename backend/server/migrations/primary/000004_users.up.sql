-- Business users. Separate from administrators: no shared table, JWT audience,
-- refresh cookie, Redis session namespace, role, or permission.
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT NOT NULL,
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    avatar_url TEXT NOT NULL DEFAULT '',
    nickname TEXT NOT NULL,
    registration_ip INET NOT NULL,
    status TEXT NOT NULL,
    email_verified_at TIMESTAMPTZ NULL,
    last_login_at TIMESTAMPTZ NULL,
    points_balance NUMERIC(20,4) NOT NULL DEFAULT 0.0000,
    consumption_points NUMERIC(20,4) NOT NULL DEFAULT 0.0000,
    level_id BIGINT NOT NULL REFERENCES user_levels(id),
    level_mode TEXT NOT NULL,
    remark TEXT NOT NULL DEFAULT '',
    auth_epoch BIGINT NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_username_nonempty CHECK (char_length(btrim(username)) > 0),
    CONSTRAINT users_username_no_at CHECK (position('@' in username) = 0),
    CONSTRAINT users_email_valid CHECK (email ~ '^[^@[:space:]]+@[^@[:space:]]+$'),
    CONSTRAINT users_password_hash_nonempty CHECK (char_length(password_hash) > 0),
    CONSTRAINT users_nickname_nonempty CHECK (char_length(btrim(nickname)) > 0),
    CONSTRAINT users_status_valid CHECK (status IN ('pending_verification', 'active', 'disabled')),
    CONSTRAINT users_level_mode_valid CHECK (level_mode IN ('auto', 'manual')),
    CONSTRAINT users_points_balance_nonnegative CHECK (points_balance >= 0),
    CONSTRAINT users_consumption_nonnegative CHECK (consumption_points >= 0)
);

CREATE UNIQUE INDEX users_username_normalized ON users ((lower(btrim(username))));
CREATE UNIQUE INDEX users_email_normalized ON users ((lower(btrim(email))));
CREATE INDEX idx_users_status ON users (status);
CREATE INDEX idx_users_level_id ON users (level_id);
CREATE INDEX idx_users_created_at ON users (created_at);