CREATE TABLE ai_chat_messages (
    id UUID PRIMARY KEY,
    conversation_id TEXT NOT NULL DEFAULT 'global',
    turn_id UUID,
    turn_sequence INTEGER,
    chat_role TEXT NOT NULL,
    content TEXT NOT NULL,
    tool_call_id TEXT,
    tool_calls JSONB,
    model TEXT,
    message_state TEXT NOT NULL DEFAULT 'COMPLETED',
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_ai_chat_messages_state CHECK (message_state IN ('PENDING', 'STREAMING', 'COMPLETED', 'FAILED')),
    CONSTRAINT chk_ai_chat_messages_turn_sequence_non_negative CHECK (turn_sequence IS NULL OR turn_sequence >= 0)
);

CREATE INDEX IF NOT EXISTS idx_ai_chat_messages_convo_created_at_id ON ai_chat_messages(conversation_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_ai_chat_messages_convo_turn_sequence ON ai_chat_messages(conversation_id, turn_id, turn_sequence);
CREATE INDEX IF NOT EXISTS idx_ai_chat_messages_convo_id ON ai_chat_messages(conversation_id, id);
CREATE INDEX IF NOT EXISTS idx_ai_chat_messages_convo_incomplete ON ai_chat_messages(conversation_id, created_at) WHERE message_state <> 'COMPLETED';
