CREATE TABLE todos (
    id                 UUID PRIMARY KEY,
    title              TEXT NOT NULL,
    status             TEXT NOT NULL,
    due_date           DATE NOT NULL,
    embedding          VECTOR(768),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_todos_status ON todos(status);
CREATE INDEX IF NOT EXISTS idx_todos_due_date ON todos(due_date);
CREATE INDEX IF NOT EXISTS idx_todos_created ON todos(created_at);
CREATE INDEX IF NOT EXISTS idx_todos_status_due_date ON todos(status, due_date);
CREATE INDEX IF NOT EXISTS idx_todos_embedding  ON todos USING hnsw (embedding vector_cosine_ops) WITH (m = 16, ef_construction = 64);



CREATE TABLE board_summary (
    id UUID PRIMARY KEY,
    summary JSONB NOT NULL,
    model TEXT NOT NULL,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    source_version BIGINT NOT NULL
);


CREATE TABLE outbox_events (
    id                 UUID PRIMARY KEY,
    entity_type      TEXT NOT NULL,
    entity_id        UUID NOT NULL,
    topic            TEXT NOT NULL,
    event_type       TEXT NOT NULL,
    payload            JSONB NOT NULL,
    status             TEXT NOT NULL DEFAULT 'PENDING',
    retry_count        INTEGER NOT NULL DEFAULT 0,
    max_retries        INTEGER NOT NULL DEFAULT 3,
    last_error         TEXT,
    created_at         TIMESTAMPTZ NOT NULL
);

-- Index for unprocessed events (ordered by creation time for FIFO processing)
CREATE INDEX IF NOT EXISTS idx_outbox_pending ON outbox_events(status, created_at ASC);


CREATE TABLE ai_chat_messages (
  id UUID PRIMARY KEY,
  conversation_id text not null DEFAULT 'global',
  chat_role TEXT NOT NULL,
  content TEXT NOT NULL,
  tool_call_id TEXT,
  tool_calls JSONB,
  model TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ai_chat_messages_convo_created_at
  ON ai_chat_messages (conversation_id, created_at);

CREATE INDEX IF NOT EXISTS idx_ai_chat_messages_convo_id
  ON ai_chat_messages (conversation_id, id);