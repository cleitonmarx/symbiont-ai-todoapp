CREATE TABLE conversations_summary (
    id UUID PRIMARY KEY,
    conversation_id TEXT NOT NULL,
    current_state_summary TEXT,
    last_summarized_message_id UUID,
    last_summarized_turn_id UUID,
    last_summarized_turn_sequence INTEGER,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_conversations_summary_last_sequence_non_negative CHECK (last_summarized_turn_sequence IS NULL OR last_summarized_turn_sequence >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_summary_conversation_id_unique ON conversations_summary(conversation_id);
