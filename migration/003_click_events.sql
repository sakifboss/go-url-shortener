-- Store every redirect click asynchronously.
CREATE TABLE click_events (
    id BIGSERIAL PRIMARY KEY,

    -- Stable event key makes worker retries idempotent.
    event_key VARCHAR(64) NOT NULL UNIQUE,

    url_id BIGINT NOT NULL
        REFERENCES urls(id)
        ON DELETE CASCADE,

    clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    user_agent TEXT,
    referrer TEXT,
    device_category VARCHAR(32)
);

CREATE INDEX idx_click_events_url_id
    ON click_events(url_id);

CREATE INDEX idx_click_events_clicked_at
    ON click_events(clicked_at);

CREATE INDEX idx_click_events_url_time
    ON click_events(url_id, clicked_at DESC);