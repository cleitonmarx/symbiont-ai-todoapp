CREATE TABLE board_summary (
    id UUID PRIMARY KEY,
    summary JSONB NOT NULL,
    model TEXT NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    source_version BIGINT NOT NULL
);
