package postgres

import (
	"context"
	"database/sql"

	"github.com/Masterminds/squirrel"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
)

var conversationSummaryFields = []string{
	"id",
	"conversation_id",
	"current_state_summary",
	"last_summarized_message_id",
	"updated_at",
}

// ConversationSummaryRepository is a PostgreSQL implementation of domain.ConversationSummaryRepository.
type ConversationSummaryRepository struct {
	sb squirrel.StatementBuilderType
}

// NewConversationSummaryRepository creates a new instance of ConversationSummaryRepository.
func NewConversationSummaryRepository(br squirrel.BaseRunner) ConversationSummaryRepository {
	return ConversationSummaryRepository{
		sb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).RunWith(br),
	}
}

// GetConversationSummary retrieves a conversation summary by conversation ID.
func (r ConversationSummaryRepository) GetConversationSummary(
	ctx context.Context,
	conversationID string,
) (domain.ConversationSummary, bool, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	var summary domain.ConversationSummary
	err := r.sb.
		Select(conversationSummaryFields...).
		From("conversations_summary").
		Where(squirrel.Eq{"conversation_id": conversationID}).
		Limit(1).
		QueryRowContext(spanCtx).
		Scan(
			&summary.ID,
			&summary.ConversationID,
			&summary.CurrentStateSummary,
			&summary.LastSummarizedMessageID,
			&summary.UpdatedAt,
		)
	if telemetry.RecordErrorAndStatus(span, err) {
		if err == sql.ErrNoRows {
			return domain.ConversationSummary{}, false, nil
		}
		return domain.ConversationSummary{}, false, err
	}

	return summary, true, nil
}

// StoreConversationSummary stores the latest conversation summary.
func (r ConversationSummaryRepository) StoreConversationSummary(ctx context.Context, summary domain.ConversationSummary) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	_, err := r.sb.
		Insert("conversations_summary").
		Columns(conversationSummaryFields...).
		Values(
			summary.ID,
			summary.ConversationID,
			summary.CurrentStateSummary,
			summary.LastSummarizedMessageID,
			summary.UpdatedAt,
		).
		Suffix(`ON CONFLICT (conversation_id) DO UPDATE SET
			current_state_summary = EXCLUDED.current_state_summary,
			last_summarized_message_id = EXCLUDED.last_summarized_message_id,
			updated_at = EXCLUDED.updated_at`).
		ExecContext(spanCtx)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	return nil
}

// DeleteConversationSummary deletes the conversation summary for a conversation (used for testing).
func (r ConversationSummaryRepository) DeleteConversationSummary(ctx context.Context, conversationID string) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	_, err := r.sb.
		Delete("conversations_summary").
		Where(squirrel.Eq{"conversation_id": conversationID}).
		ExecContext(spanCtx)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	return nil
}

// InitConversationSummaryRepository is a Symbiont initializer for ConversationSummaryRepository.
type InitConversationSummaryRepository struct {
	DB *sql.DB `resolve:""`
}

// Initialize registers the ConversationSummaryRepository in the dependency container.
func (i InitConversationSummaryRepository) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.ConversationSummaryRepository](NewConversationSummaryRepository(i.DB))
	return ctx, nil
}
