-- Define saldo disponivel de todos os jogadores em 1.000.000 fichas.
ALTER TABLE users ALTER COLUMN saldo_fichas SET DEFAULT 1000000;
UPDATE users SET saldo_fichas = 1000000;

-- Mantem valores reservados em mesas e atualiza o saldo disponivel das carteiras.
UPDATE wallets
SET available_cents = 1000000,
    balance_cents = 1000000 + reserved_cents,
    updated_at = CURRENT_TIMESTAMP;
