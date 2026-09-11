-- Business-user levels. Exact four-decimal thresholds; automatic level
-- calculation selects the highest enabled threshold <= cumulative consumption.
CREATE TABLE user_levels (
    id BIGSERIAL PRIMARY KEY,
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    icon_url TEXT NOT NULL DEFAULT '',
    threshold_points NUMERIC(20,4) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_levels_code_nonempty CHECK (char_length(btrim(code)) > 0),
    CONSTRAINT user_levels_name_nonempty CHECK (char_length(btrim(name)) > 0),
    CONSTRAINT user_levels_threshold_nonnegative CHECK (threshold_points >= 0)
);

CREATE UNIQUE INDEX user_levels_code_normalized ON user_levels ((lower(btrim(code))));

-- Enabled thresholds must be unambiguous: two enabled levels cannot share the
-- same threshold. Disabled levels may re-use thresholds freely.
CREATE UNIQUE INDEX user_levels_enabled_threshold
    ON user_levels (threshold_points) WHERE enabled = TRUE;

-- Seed exactly one enabled default level at threshold 0.0000. Any registration
-- or re-calculation with zero cumulative consumption lands on this level.
INSERT INTO user_levels (code, name, icon_url, threshold_points, sort_order, enabled)
VALUES ('default', 'Default', '', 0.0000, 0, TRUE)
ON CONFLICT DO NOTHING;