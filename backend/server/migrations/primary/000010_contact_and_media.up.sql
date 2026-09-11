-- Contact inbox and media library (BE-02).
--
-- contact_submissions stores public "contact us" form submissions together
-- with the anti-spam metadata (source IP, user agent) an operator needs to
-- triage them. media_assets indexes uploaded media used across the public
-- site; the bytes live in the configured MediaStore (local disk by default).

CREATE TABLE contact_submissions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    company TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL,
    locale TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'new',
    source_ip TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT contact_submissions_name_nonempty CHECK (char_length(btrim(name)) > 0),
    CONSTRAINT contact_submissions_email_nonempty CHECK (char_length(btrim(email)) > 0),
    CONSTRAINT contact_submissions_message_nonempty CHECK (char_length(btrim(message)) > 0),
    CONSTRAINT contact_submissions_status_valid CHECK (status IN ('new', 'read', 'handled'))
);

CREATE INDEX contact_submissions_status_created_idx
    ON contact_submissions (status, created_at DESC, id DESC);

CREATE INDEX contact_submissions_created_idx
    ON contact_submissions (created_at DESC, id DESC);

CREATE TABLE media_assets (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    url TEXT NOT NULL UNIQUE,
    original_name TEXT NOT NULL DEFAULT '',
    mime TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    width INTEGER NOT NULL DEFAULT 0,
    height INTEGER NOT NULL DEFAULT 0,
    created_by BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT media_assets_url_nonempty CHECK (char_length(btrim(url)) > 0),
    CONSTRAINT media_assets_mime_nonempty CHECK (char_length(btrim(mime)) > 0),
    CONSTRAINT media_assets_size_nonnegative CHECK (size_bytes >= 0),
    CONSTRAINT media_assets_dimensions_nonnegative CHECK (width >= 0 AND height >= 0)
);

CREATE INDEX media_assets_created_idx ON media_assets (created_at DESC, id DESC);
