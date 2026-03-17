package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var conversationFields = []string{
	"id",
	"title",
	"title_source",
	"last_message_at",
	"created_at",
	"updated_at",
}

// ConversationRepository is a PostgreSQL implementation of assistant.ConversationRepository.
type ConversationRepository struct {
	sb squirrel.StatementBuilderType
}

// NewConversationRepository creates a new instance of ConversationRepository.
func NewConversationRepository(br squirrel.BaseRunner) ConversationRepository {
	return ConversationRepository{
		sb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).RunWith(br),
	}
}

// CreateConversation creates a conversation with an auto-generated ID and timestamps.
func (r ConversationRepository) CreateConversation(
	ctx context.Context,
	title string,
	source assistant.ConversationTitleSource,
) (assistant.Conversation, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	now := time.Now().UTC()
	input := assistant.Conversation{
		ID:            uuid.New(),
		Title:         title,
		TitleSource:   source,
		LastMessageAt: nil,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := input.Validate(); telemetry.IsErrorRecorded(span, err) {
		return assistant.Conversation{}, err
	}

	var created assistant.Conversation
	err := r.sb.
		Insert("conversations").
		Columns(conversationFields...).
		Values(
			input.ID,
			input.Title,
			input.TitleSource,
			input.LastMessageAt,
			input.CreatedAt,
			input.UpdatedAt,
		).
		Suffix("RETURNING id, title, title_source, last_message_at, created_at, updated_at").
		QueryRowContext(spanCtx).
		Scan(
			&created.ID,
			&created.Title,
			&created.TitleSource,
			&created.LastMessageAt,
			&created.CreatedAt,
			&created.UpdatedAt,
		)
	if telemetry.IsErrorRecorded(span, err) {
		return assistant.Conversation{}, err
	}

	return created, nil
}

// GetConversation retrieves a conversation by ID.
func (r ConversationRepository) GetConversation(
	ctx context.Context,
	conversationID uuid.UUID,
) (assistant.Conversation, bool, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	var conversation assistant.Conversation
	err := r.sb.
		Select(conversationFields...).
		From("conversations").
		Where(squirrel.Eq{"id": conversationID}).
		Limit(1).
		QueryRowContext(spanCtx).
		Scan(
			&conversation.ID,
			&conversation.Title,
			&conversation.TitleSource,
			&conversation.LastMessageAt,
			&conversation.CreatedAt,
			&conversation.UpdatedAt,
		)

	if errors.Is(err, sql.ErrNoRows) {
		return assistant.Conversation{}, false, nil
	}

	if telemetry.IsErrorRecorded(span, err) {
		return assistant.Conversation{}, false, err
	}

	return conversation, true, nil
}

// GetConversationContextTokenUsage returns the current unsummarized context token usage for the given conversations.
func (r ConversationRepository) GetConversationContextTokenUsage(
	ctx context.Context,
	conversationIDs []uuid.UUID,
) (map[uuid.UUID]int64, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	usageByConversationID := make(map[uuid.UUID]int64, len(conversationIDs))
	if len(conversationIDs) == 0 {
		return usageByConversationID, nil
	}

	conversationTokenUsageSubquery := r.sb.
		Select(
			"COALESCE(SUM(chat_messages.context_tokens_estimate), 0)::BIGINT AS total_tokens_used",
		).
		From("chat_messages").
		LeftJoin("conversations_summary conversation_summary ON conversation_summary.conversation_id = conversations.id").
		LeftJoin("chat_messages checkpoint ON checkpoint.conversation_id = conversations.id AND checkpoint.id = conversation_summary.last_summarized_message_id").
		Where("chat_messages.conversation_id = conversations.id").
		Where(`(
			checkpoint.id IS NULL
			OR chat_messages.created_at > checkpoint.created_at
			OR (
				chat_messages.created_at = checkpoint.created_at
				AND chat_messages.id > checkpoint.id
			)
		)`).
		Prefix("LEFT JOIN LATERAL (").
		Suffix(") conversation_token_usage ON TRUE")

	query := r.sb.
		Select(
			"conversations.id AS conversation_id",
			"COALESCE(conversation_token_usage.total_tokens_used, 0) AS total_tokens_used",
		).
		From("conversations").
		JoinClause(conversationTokenUsageSubquery).
		//Where(squirrel.Eq{"conversations.id": conversationIDs})
		Where(squirrel.Expr("conversations.id = ANY(?)", pq.Array(conversationIDs)))

	rows, err := query.QueryContext(spanCtx)
	if telemetry.IsErrorRecorded(span, err) {
		return nil, err
	}
	defer rows.Close() //nolint:errcheck

	for rows.Next() {
		var conversationID uuid.UUID
		var totalTokensUsed int64
		if err := rows.Scan(&conversationID, &totalTokensUsed); telemetry.IsErrorRecorded(span, err) {
			return nil, err
		}
		usageByConversationID[conversationID] = totalTokensUsed
	}
	if err := rows.Err(); telemetry.IsErrorRecorded(span, err) {
		return nil, err
	}

	return usageByConversationID, nil
}

// UpdateConversation updates mutable fields for one conversation.
func (r ConversationRepository) UpdateConversation(
	ctx context.Context,
	conversation assistant.Conversation,
) error {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	_, err := r.sb.
		Update("conversations").
		Set("title", conversation.Title).
		Set("title_source", conversation.TitleSource).
		Set("last_message_at", conversation.LastMessageAt).
		Set("updated_at", conversation.UpdatedAt).
		Where(squirrel.Eq{"id": conversation.ID}).
		ExecContext(spanCtx)
	if telemetry.IsErrorRecorded(span, err) {
		return err
	}

	return nil
}

// ListConversations returns paginated conversations ordered by last interaction recency.
func (r ConversationRepository) ListConversations(
	ctx context.Context,
	page int,
	pageSize int,
) ([]assistant.Conversation, bool, error) {
	spanCtx, span := telemetry.StartSpan(ctx, trace.WithAttributes(
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
	))
	defer span.End()

	if page <= 0 {
		err := core.NewValidationErr("page must be greater than 0")
		telemetry.IsErrorRecorded(span, err)
		return nil, false, err
	}
	if pageSize <= 0 {
		err := core.NewValidationErr("page_size must be greater than 0")
		telemetry.IsErrorRecorded(span, err)
		return nil, false, err
	}

	rows, err := r.sb.
		Select(conversationFields...).
		From("conversations").
		OrderBy("last_message_at DESC NULLS LAST", "updated_at DESC", "created_at DESC").
		Limit(uint64(pageSize + 1)).
		Offset(uint64((page - 1) * pageSize)).
		QueryContext(spanCtx)
	if telemetry.IsErrorRecorded(span, err) {
		return nil, false, err
	}
	defer rows.Close() //nolint:errcheck

	conversations := []assistant.Conversation{}
	for rows.Next() {
		var conversation assistant.Conversation
		err := rows.Scan(
			&conversation.ID,
			&conversation.Title,
			&conversation.TitleSource,
			&conversation.LastMessageAt,
			&conversation.CreatedAt,
			&conversation.UpdatedAt,
		)
		if telemetry.IsErrorRecorded(span, err) {
			return nil, false, err
		}

		conversations = append(conversations, conversation)
	}
	if err := rows.Err(); telemetry.IsErrorRecorded(span, err) {
		return nil, false, err
	}

	hasMore := false
	if len(conversations) > pageSize {
		hasMore = true
		conversations = conversations[:pageSize]
	}

	return conversations, hasMore, nil
}

// DeleteConversation deletes a conversation by ID.
func (r ConversationRepository) DeleteConversation(ctx context.Context, conversationID uuid.UUID) error {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	_, err := r.sb.
		Delete("conversations").
		Where(squirrel.Eq{"id": conversationID}).
		ExecContext(spanCtx)
	if telemetry.IsErrorRecorded(span, err) {
		return err
	}

	return nil
}
