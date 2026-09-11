-- Immutable points ledger. Every points change for a business user is one row
-- here; the balance columns record both resulting balances so history can be
-- replayed exactly. Available balance can never go negative and cumulative
-- consumption points can never be reduced.
CREATE TABLE user_point_transactions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    points_delta NUMERIC(20,4) NOT NULL,
    consumption_delta NUMERIC(20,4) NOT NULL,
    balance_after NUMERIC(20,4) NOT NULL,
    consumption_after NUMERIC(20,4) NOT NULL,
    reason TEXT NOT NULL,
    actor_id BIGINT NULL,
    idempotency_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_point_transactions_reason_nonempty CHECK (char_length(btrim(reason)) > 0),
    CONSTRAINT user_point_transactions_idempotency_nonempty CHECK (char_length(btrim(idempotency_key)) > 0),
    CONSTRAINT user_point_transactions_not_noop CHECK (points_delta <> 0 OR consumption_delta <> 0),
    CONSTRAINT user_point_transactions_delta_nonnegative CHECK (consumption_delta >= 0),
    CONSTRAINT user_point_transactions_balance_nonnegative CHECK (balance_after >= 0),
    CONSTRAINT user_point_transactions_consumption_after_nonnegative CHECK (consumption_after >= 0)
);

CREATE UNIQUE INDEX user_point_transactions_idempotency ON user_point_transactions (idempotency_key);
CREATE INDEX idx_user_point_transactions_user_id ON user_point_transactions (user_id);
CREATE INDEX idx_user_point_transactions_created_at ON user_point_transactions (created_at DESC);