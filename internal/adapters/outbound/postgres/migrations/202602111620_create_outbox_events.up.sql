CREATE TABLE outbox_events (
    id UUID PRIMARY KEY,
    entity_type TEXT NOT NULL,
    entity_id UUID NOT NULL,
    topic TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'PENDING',
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,
    last_error TEXT,
    dedupe_key TEXT,
    available_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox_events(status, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_outbox_pending_available ON outbox_events(status, available_at ASC, created_at ASC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_outbox_dedupe_key_unique ON outbox_events(dedupe_key) WHERE dedupe_key IS NOT NULL;
