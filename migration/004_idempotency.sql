CREATE TABLE idempotency_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    idempotency_key VARCHAR(255) NOT NULL,
    response_body JSONB NOT NULL,
    status_code INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE (user_id, idempotency_key)
);

CREATE INDEX idx_idempotency_keys_user_id
    ON idempotency_keys(user_id);