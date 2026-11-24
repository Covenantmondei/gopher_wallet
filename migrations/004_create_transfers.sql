CREATE TABLE IF NOT EXISTS transfers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_wallet_id UUID NOT NULL REFERENCES wallets(id),
    target_wallet_id UUID NOT NULL REFERENCES wallets(id),
    debit_transaction_id UUID NOT NULL REFERENCES transactions(id),
    credit_transaction_id UUID NOT NULL REFERENCES transactions(id),
    amount BIGINT NOT NULL CHECK (amount > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transfers_source_wallet ON transfers(source_wallet_id);
CREATE INDEX IF NOT EXISTS idx_transfers_target_wallet ON transfers(target_wallet_id);
