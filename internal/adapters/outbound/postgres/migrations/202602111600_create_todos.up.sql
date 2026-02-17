CREATE TABLE todos (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    status TEXT NOT NULL,
    due_date DATE NOT NULL,
    embedding VECTOR(768),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_todos_status ON todos(status);
CREATE INDEX IF NOT EXISTS idx_todos_due_date ON todos(due_date);
CREATE INDEX IF NOT EXISTS idx_todos_created ON todos(created_at);
CREATE INDEX IF NOT EXISTS idx_todos_status_due_date ON todos(status, due_date);
CREATE INDEX IF NOT EXISTS idx_todos_embedding ON todos USING hnsw (embedding vector_cosine_ops) WITH (m = 24, ef_construction = 128);
