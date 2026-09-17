-- Authentication fields for users.
ALTER TABLE users
    ADD COLUMN email VARCHAR(320) NOT NULL UNIQUE,
    ADD COLUMN password_hash TEXT NOT NULL,
    ADD COLUMN role VARCHAR(32) NOT NULL DEFAULT 'user';

-- Refresh tokens are stored hashed so the raw token is never
-- persisted in PostgreSQL.
CREATE TABLE refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash CHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_id
    ON refresh_tokens(user_id);

CREATE INDEX idx_refresh_tokens_active
    ON refresh_tokens(token_hash, expires_at)
    WHERE revoked_at IS NULL;