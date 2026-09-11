CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE audit_events (
    id BIGSERIAL PRIMARY KEY,
    actor_id BIGINT,
    actor_username TEXT NOT NULL DEFAULT '',
    actor_display_name TEXT NOT NULL DEFAULT '',
    actor_role_codes TEXT[] NOT NULL DEFAULT '{}',
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL DEFAULT '',
    outcome TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    request_id TEXT NOT NULL DEFAULT '',
    source_ip TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    event_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT audit_events_action_nonempty CHECK (char_length(btrim(action)) > 0),
    CONSTRAINT audit_events_resource_type_nonempty CHECK (char_length(btrim(resource_type)) > 0),
    CONSTRAINT audit_events_outcome_valid CHECK (outcome IN ('pending', 'succeeded', 'failed'))
);

CREATE INDEX idx_audit_events_event_at ON audit_events (event_at DESC);
CREATE INDEX idx_audit_events_actor_id ON audit_events (actor_id);
CREATE INDEX idx_audit_events_action ON audit_events (action);
CREATE INDEX idx_audit_events_resource ON audit_events (resource_type, resource_id);
CREATE INDEX idx_audit_events_outcome ON audit_events (outcome);
CREATE INDEX idx_audit_events_request_id ON audit_events (request_id);
