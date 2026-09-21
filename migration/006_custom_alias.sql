ALTER TABLE urls
    ADD COLUMN custom_alias VARCHAR(32);

CREATE UNIQUE INDEX idx_urls_custom_alias
    ON urls(custom_alias)
    WHERE custom_alias IS NOT NULL;
