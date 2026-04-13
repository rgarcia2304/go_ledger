-- +goose Up
-- +goose StatementBegin

CREATE TABLE accounts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name         TEXT NOT NULL,
    currency     TEXT NOT NULL,
    account_type TEXT NOT NULL CHECK (account_type IN (
                     'asset', 'liability', 'equity', 'revenue', 'expense'
                 )),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE transactions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    description     TEXT NOT NULL,
    idempotency_key TEXT UNIQUE NOT NULL,
    occurred_at     TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE entries (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID NOT NULL REFERENCES transactions(id),
    account_id     UUID NOT NULL REFERENCES accounts(id),
    amount_cents   BIGINT NOT NULL CHECK (amount_cents > 0),
    currency       TEXT NOT NULL,
    direction      TEXT NOT NULL CHECK (direction IN ('debit', 'credit')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_entries_account_id_created_at
    ON entries(account_id, created_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX IF EXISTS idx_entries_account_id_created_at;
DROP TABLE IF EXISTS entries;
DROP TABLE IF EXISTS transactions;
DROP TABLE IF EXISTS accounts;

-- +goose StatementEnd
