-- Schema Migration: 000001_init_schema.up.sql

-- URLs Table
CREATE TABLE IF NOT EXISTS urls (
    id BIGINT GENERATED ALWAYS AS IDENTITY (START WITH 1000000) PRIMARY KEY,
    short_code VARCHAR(16) NOT NULL,
    original_url TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NULL,
    click_count BIGINT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    CONSTRAINT uq_urls_short_code UNIQUE (short_code)
);

-- Index for fast lookup by short_code (enforces uniqueness & B-tree lookup)
CREATE UNIQUE INDEX IF NOT EXISTS idx_urls_short_code ON urls(short_code);

-- Index for ordering and range queries by creation date
CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at DESC);

-- Composite index to optimize expiration and active status filtering
CREATE INDEX IF NOT EXISTS idx_urls_active_expires ON urls(is_active, expires_at);

-- Click Logs Table (Analytics)
CREATE TABLE IF NOT EXISTS click_logs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    url_id BIGINT NOT NULL REFERENCES urls(id) ON DELETE CASCADE,
    short_code VARCHAR(16) NOT NULL,
    clicked_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ip_address VARCHAR(45) NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    referrer TEXT NOT NULL DEFAULT ''
);

-- Index for fast retrieval of click history per url ordered by time
CREATE INDEX IF NOT EXISTS idx_click_logs_url_id_clicked ON click_logs(url_id, clicked_at DESC);

-- Index for analytics lookup by short_code
CREATE INDEX IF NOT EXISTS idx_click_logs_short_code_clicked ON click_logs(short_code, clicked_at DESC);

-- Index for general time-series analytics and retention pruning
CREATE INDEX IF NOT EXISTS idx_click_logs_clicked_at ON click_logs(clicked_at DESC);
