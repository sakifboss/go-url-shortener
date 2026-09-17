-- Create the users table first because URLs reference users
-- through a foreign key. Authentication details will be added
-- later in Milestone 4.

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Store every shortened URL permanently in PostgreSQL.
-- The unique constraint prevents two URLs from sharing
-- the same short code.

CREATE TABLE urls (
    id BIGSERIAL PRIMARY KEY,
    short_code VARCHAR(32) NOT NULL UNIQUE,
    original_url TEXT NOT NULL,
    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE
);

-- Redirects search by short_code, so the unique constraint
-- also provides an index for this lookup.
CREATE INDEX idx_urls_user_id ON urls(user_id);