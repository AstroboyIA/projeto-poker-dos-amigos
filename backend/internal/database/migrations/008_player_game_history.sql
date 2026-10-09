CREATE TABLE IF NOT EXISTS cash_game_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    table_id UUID NOT NULL,
    table_name VARCHAR(150) NOT NULL,
    payout_cents BIGINT NOT NULL DEFAULT 0 CHECK (payout_cents >= 0),
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    finished_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX IF NOT EXISTS cash_game_sessions_active_user_table_idx
    ON cash_game_sessions(user_id, table_id) WHERE finished_at IS NULL;

CREATE INDEX IF NOT EXISTS cash_game_sessions_user_started_idx
    ON cash_game_sessions(user_id, started_at DESC);

CREATE TABLE IF NOT EXISTS cash_game_session_buyins (
    idempotency_key VARCHAR(255) PRIMARY KEY,
    session_id UUID NOT NULL REFERENCES cash_game_sessions(id) ON DELETE CASCADE,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
