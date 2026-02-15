CREATE TABLE conversations_summary (
    id UUID PRIMARY KEY,
    conversation_id UUID NOT NULL,
    current_state_summary TEXT,
    last_summarized_message_id UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_summary_conversation_id_unique ON conversations_summary(conversation_id);
