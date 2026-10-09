CREATE TABLE IF NOT EXISTS wallet_table_reservations (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    table_id TEXT NOT NULL,
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, table_id)
);

ALTER TABLE wallet_ledger
    DROP CONSTRAINT IF EXISTS wallet_ledger_amount_positive;

ALTER TABLE wallet_ledger
    DROP CONSTRAINT IF EXISTS wallet_ledger_amount_non_negative;

ALTER TABLE wallet_ledger
    ADD CONSTRAINT wallet_ledger_amount_non_negative CHECK (amount_cents >= 0);
