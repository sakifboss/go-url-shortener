ALTER TABLE idempotency_keys
    ADD COLUMN state VARCHAR(16) NOT NULL DEFAULT 'completed';

ALTER TABLE idempotency_keys
    ADD CONSTRAINT idempotency_keys_state_check
    CHECK (state IN ('pending', 'completed'));
