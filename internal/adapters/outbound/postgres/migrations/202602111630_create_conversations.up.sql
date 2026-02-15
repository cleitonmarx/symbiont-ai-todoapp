CREATE TABLE conversations (
    id UUID PRIMARY KEY,
    title TEXT NOT NULL,
    title_source TEXT NOT NULL,
    last_message_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_conversations_last_message_at ON conversations(last_message_at);