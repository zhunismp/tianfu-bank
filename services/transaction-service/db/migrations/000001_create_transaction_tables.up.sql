CREATE TABLE transaction_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id      VARCHAR(36) NOT NULL,
    event_type      VARCHAR(30) NOT NULL,
    amount          NUMERIC(20,2) NOT NULL DEFAULT 0,
    reference_id    VARCHAR(36),
    idempotency_key VARCHAR(100),
    sequence_number BIGINT NOT NULL,
    metadata        JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_txe_account_seq ON transaction_events(account_id, sequence_number);
CREATE UNIQUE INDEX idx_txe_idemp ON transaction_events(idempotency_key) WHERE idempotency_key IS NOT NULL;

CREATE TABLE account_snapshots (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id           VARCHAR(36) NOT NULL,
    balance              NUMERIC(20,2) NOT NULL DEFAULT 0,
    last_sequence_number BIGINT NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_snap_account ON account_snapshots(account_id, last_sequence_number DESC);

CREATE TABLE accounts (
    account_id   VARCHAR(36) PRIMARY KEY,
    user_id      VARCHAR(36) NOT NULL,
    account_type VARCHAR(20) NOT NULL,
    branch_id    VARCHAR(36) NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE idempotency_keys (
    idempotency_key VARCHAR(100) PRIMARY KEY,
    status_code     INT NOT NULL,
    response_body   JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '24 hours'
);
