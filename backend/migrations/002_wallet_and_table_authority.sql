-- Migration incremental: preserva saldo_fichas e dados já existentes.
ALTER TABLE users ADD COLUMN IF NOT EXISTS wallet_id UUID;
ALTER TABLE poker_tables ADD COLUMN IF NOT EXISTS creator_user_id UUID REFERENCES users(id);

CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    balance_cents BIGINT NOT NULL DEFAULT 0,
    available_cents BIGINT NOT NULL DEFAULT 0,
    reserved_cents BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT wallets_balance_consistent CHECK (balance_cents = available_cents + reserved_cents),
    CONSTRAINT wallets_non_negative CHECK (balance_cents >= 0 AND available_cents >= 0 AND reserved_cents >= 0)
);

CREATE TABLE IF NOT EXISTS wallet_ledger (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    type VARCHAR(32) NOT NULL,
    amount_cents BIGINT NOT NULL,
    balance_before BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    reference_type VARCHAR(64),
    reference_id VARCHAR(255),
    idempotency_key VARCHAR(255) UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT wallet_ledger_amount_positive CHECK (amount_cents > 0)
);

INSERT INTO wallets (user_id, balance_cents, available_cents, reserved_cents)
SELECT id, saldo_fichas, saldo_fichas, 0
FROM users
WHERE NOT EXISTS (SELECT 1 FROM wallets w WHERE w.user_id = users.id);

UPDATE users u
SET wallet_id = w.id
FROM wallets w
WHERE w.user_id = u.id AND u.wallet_id IS NULL;

CREATE TABLE IF NOT EXISTS table_players (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    table_id UUID NOT NULL REFERENCES poker_tables(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    seat_number INT NOT NULL,
    stack_chips BIGINT NOT NULL DEFAULT 0,
    buy_in_cents BIGINT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (table_id, seat_number),
    UNIQUE (table_id, user_id)
);

CREATE INDEX IF NOT EXISTS wallet_ledger_user_created_idx ON wallet_ledger(user_id, created_at DESC);
