CREATE TABLE ai_chat_messages (
    id UUID PRIMARY KEY,
    conversation_id UUID NOT NULL,
    turn_id UUID NOT NULL,
    turn_sequence INTEGER NOT NULL,
    chat_role TEXT NOT NULL,
    content TEXT NOT NULL,
    tool_call_id TEXT,
    tool_calls JSONB,
    model TEXT,
    message_state TEXT NOT NULL,
    error_message TEXT,
    prompt_tokens INTEGER NOT NULL,
    completion_tokens INTEGER NOT NULL,
    total_tokens INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ai_chat_messages_convo_created_at_id ON ai_chat_messages(conversation_id, created_at, id);
CREATE UNIQUE INDEX IF NOT EXISTS uidx_ai_chat_messages_convo_turn_sequence ON ai_chat_messages(conversation_id, turn_id, turn_sequence);
CREATE INDEX IF NOT EXISTS idx_ai_chat_messages_convo_id ON ai_chat_messages(conversation_id, id);
CREATE INDEX IF NOT EXISTS idx_ai_chat_messages_convo_incomplete ON ai_chat_messages(conversation_id, created_at) WHERE message_state <> 'COMPLETED';
