-- Daylog 핵심 스키마 (요약본)

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'active',
    tier TEXT NOT NULL DEFAULT 'free',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_settings (
    user_id UUID PRIMARY KEY REFERENCES users(id),
    data_visibility_level TEXT NOT NULL DEFAULT 'private',
    timezone TEXT NOT NULL DEFAULT 'Asia/Seoul'
);

CREATE TABLE IF NOT EXISTS user_labels (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    label_key TEXT NOT NULL,
    label_value TEXT NOT NULL,
    is_verified BOOLEAN NOT NULL DEFAULT FALSE,
    verified_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS activity_events (
    event_id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    source TEXT NOT NULL,
    timestamp_start TIMESTAMPTZ NOT NULL,
    timestamp_end TIMESTAMPTZ NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB
);

CREATE TABLE IF NOT EXISTS timeline_entries (
    timeline_id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    category TEXT NOT NULL,
    confidence NUMERIC NOT NULL DEFAULT 0.0,
    geo_context JSONB NOT NULL DEFAULT '{}'::JSONB,
    source_event_ids UUID[] NOT NULL DEFAULT ARRAY[]::UUID[]
);

CREATE TABLE IF NOT EXISTS activity_feedback (
    feedback_id UUID PRIMARY KEY,
    timeline_id UUID NOT NULL REFERENCES timeline_entries(timeline_id),
    user_id UUID NOT NULL REFERENCES users(id),
    old_category TEXT NOT NULL,
    new_category TEXT NOT NULL,
    note TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 추가 테이블은 docs/architecture.md 참조
