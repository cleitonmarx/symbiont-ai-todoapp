CREATE TABLE conversation_message_sequence (
    conversation_id TEXT PRIMARY KEY,
    next_sequence BIGINT NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
