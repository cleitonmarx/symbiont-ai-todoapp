package chat

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/google/uuid"
)

// compactConversationContext evaluates and runs pre-turn context compaction while emitting SSE lifecycle events.
func (sc StreamChatImpl) compactConversationContext(
	ctx context.Context,
	conversationID uuid.UUID,
	onEvent assistant.EventCallback,
) error {
	if sc.conversationCompactor == nil {
		return nil
	}

	evalCtx, cancelEval := context.WithTimeout(ctx, sc.compactionTimeout)
	defer cancelEval()

	decision, err := sc.conversationCompactor.EvaluateConversationCompaction(evalCtx, conversationID, sc.compactionPolicy)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("context compaction evaluation timed out after %s", sc.compactionTimeout)
		}
		sc.logger.Printf("StreamChat: context compaction evaluation failed for conversation %s: %v", conversationID, err)
		return onEvent(ctx, assistant.EventType_ContextCompactionFailed, assistant.ContextCompactionFailed{
			ConversationID:           conversationID,
			UnsummarizedMessageCount: 0,
			UnsummarizedTotalTokens:  0,
			Reason:                   assistant.ContextCompactionReasonNone,
			Error:                    err.Error(),
		})
	}

	if !decision.ShouldGenerate {
		return nil
	}

	if err := onEvent(ctx, assistant.EventType_ContextCompactionStarted, assistant.ContextCompactionStarted{
		ConversationID:           conversationID,
		UnsummarizedMessageCount: decision.MessageCount,
		UnsummarizedTotalTokens:  decision.TotalTokens,
		Reason:                   decision.Reason,
	}); err != nil {
		return err
	}

	compactCtx, cancelCompact := context.WithTimeout(ctx, sc.compactionTimeout)
	defer cancelCompact()

	if err := sc.conversationCompactor.CompactConversation(compactCtx, conversationID); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("context compaction timed out after %s", sc.compactionTimeout)
		}
		sc.logger.Printf("StreamChat: context compaction failed for conversation %s: %v", conversationID, err)
		return onEvent(ctx, assistant.EventType_ContextCompactionFailed, assistant.ContextCompactionFailed{
			ConversationID:           conversationID,
			UnsummarizedMessageCount: decision.MessageCount,
			UnsummarizedTotalTokens:  decision.TotalTokens,
			Reason:                   decision.Reason,
			Error:                    err.Error(),
		})
	}

	return onEvent(ctx, assistant.EventType_ContextCompactionCompleted, assistant.ContextCompactionCompleted{
		ConversationID:           conversationID,
		UnsummarizedMessageCount: decision.MessageCount,
		UnsummarizedTotalTokens:  decision.TotalTokens,
		Reason:                   decision.Reason,
		CompactedAt:              sc.timeProvider.Now().Format(time.RFC3339),
	})
}
