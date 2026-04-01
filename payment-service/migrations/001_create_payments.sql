CREATE TABLE IF NOT EXISTS payments
(
    id             UUID        PRIMARY KEY,
    order_id       UUID        NOT NULL,
    transaction_id TEXT UNIQUE NOT NULL,
    amount         BIGINT      NOT NULL CHECK (amount > 0),
    status         TEXT        NOT NULL CHECK (status IN ('Authorized', 'Declined')),
    created_at     TIMESTAMP   NOT NULL DEFAULT NOW()
);
