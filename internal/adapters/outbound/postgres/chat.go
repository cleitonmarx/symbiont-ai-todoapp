package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"

	sq "github.com/Masterminds/squirrel"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var chatFields = []string{
	"id",
	"conversation_id",
	"turn_id",
	"turn_sequence",
	"chat_role",
	"content",
	"tool_call_id",
	"tool_calls",
	"model",
	"message_state",
	"error_message",
	"prompt_tokens",
	"completion_tokens",
	"total_tokens",
	"created_at",
	"updated_at",
}

// ChatMessageRepository persists chat messages in Postgres.
type ChatMessageRepository struct {
	sb sq.StatementBuilderType
}

// NewChatMessageRepository creates a new ChatMessageRepository.
func NewChatMessageRepository(br sq.BaseRunner) ChatMessageRepository {
	return ChatMessageRepository{
		sb: sq.StatementBuilder.PlaceholderFormat(sq.Dollar).RunWith(br),
	}
}

// CreateChatMessages persists chat messages for the global conversation.
func (r ChatMessageRepository) CreateChatMessages(ctx context.Context, messages []domain.ChatMessage) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	insertQry := r.sb.
		Insert("ai_chat_messages").
		Columns(chatFields...)

	for _, message := range messages {
		toolCallsJSON, err := json.Marshal(message.ToolCalls)
		if telemetry.RecordErrorAndStatus(span, err) {
			return err
		}

		insertQry = insertQry.Values(
			message.ID,
			message.ConversationID,
			message.TurnID,
			message.TurnSequence,
			message.ChatRole,
			message.Content,
			message.ToolCallID,
			toolCallsJSON,
			message.Model,
			message.MessageState,
			message.ErrorMessage,
			message.PromptTokens,
			message.CompletionTokens,
			message.TotalTokens,
			message.CreatedAt,
			message.UpdatedAt,
		)
	}

	_, err := insertQry.ExecContext(spanCtx)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}
	return nil
}

// ListChatMessages retrieves messages ordered by creation time using optional filters.
// If limit > 0, returns up to N messages and indicates whether more messages exist.
func (r ChatMessageRepository) ListChatMessages(
	ctx context.Context,
	limit int,
	options ...domain.ListChatMessagesOption,
) ([]domain.ChatMessage, bool, error) {
	spanCtx, span := telemetry.Start(ctx, trace.WithAttributes(
		attribute.Int("limit", limit),
	))
	defer span.End()

	queryOptions := domain.ListChatMessagesOptions{
		ConversationID: domain.GlobalConversationID,
	}
	for _, option := range options {
		if option != nil {
			option(&queryOptions)
		}
	}
	span.SetAttributes(
		attribute.String("conversation_id", queryOptions.ConversationID),
	)

	qry := r.sb.
		Select(chatFields...).
		From("ai_chat_messages").
		Where(sq.Eq{"conversation_id": queryOptions.ConversationID})

	if queryOptions.AfterMessageID != nil {
		span.SetAttributes(
			attribute.String("after_message_id", queryOptions.AfterMessageID.String()),
		)

		qry = qry.JoinClause(
			r.sb.
				Select(
					"created_at AS checkpoint_created_at",
					"id AS checkpoint_id",
				).
				From("ai_chat_messages").
				Where(sq.Eq{
					"conversation_id": queryOptions.ConversationID,
					"id":              *queryOptions.AfterMessageID,
				}).
				Limit(1).
				Prefix("LEFT JOIN (").
				Suffix(") checkpoint ON TRUE"),
		).Where(
			sq.Or{
				sq.Eq{"checkpoint.checkpoint_id": nil},
				sq.Expr("ai_chat_messages.created_at > checkpoint.checkpoint_created_at"),
				sq.And{
					sq.Expr("ai_chat_messages.created_at = checkpoint.checkpoint_created_at"),
					sq.Expr("ai_chat_messages.id > checkpoint.checkpoint_id"),
				},
			},
		)

		qry = qry.OrderBy("created_at ASC", "id ASC")
	} else {
		qry = qry.OrderBy("created_at DESC", "id DESC")
	}

	if limit > 0 {
		qry = qry.Limit(uint64(limit + 1)) // fetch one extra to detect more
	}

	rows, err := qry.QueryContext(spanCtx)
	if telemetry.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}
	defer rows.Close() //nolint:errcheck

	var msgs []domain.ChatMessage
	for rows.Next() {
		var (
			m      domain.ChatMessage
			tcJSON []byte
		)

		if err := rows.Scan(
			&m.ID,
			&m.ConversationID,
			&m.TurnID,
			&m.TurnSequence,
			&m.ChatRole,
			&m.Content,
			&m.ToolCallID,
			&tcJSON,
			&m.Model,
			&m.MessageState,
			&m.ErrorMessage,
			&m.PromptTokens,
			&m.CompletionTokens,
			&m.TotalTokens,
			&m.CreatedAt,
			&m.UpdatedAt,
		); telemetry.RecordErrorAndStatus(span, err) {
			return nil, false, err
		}

		if len(tcJSON) > 0 {
			if err := json.Unmarshal(tcJSON, &m.ToolCalls); telemetry.RecordErrorAndStatus(span, err) {
				return nil, false, err
			}
		}
		msgs = append(msgs, m)
	}
	if err := rows.Err(); telemetry.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}

	hasMore := false
	if limit > 0 && len(msgs) > limit {
		hasMore = true
		msgs = msgs[:limit]
	}

	// Keep chronological order for callers.
	if queryOptions.AfterMessageID == nil {
		// Query path without checkpoint orders DESC for efficient latest reads.
		sort.SliceStable(msgs, func(i, j int) bool {
			return msgs[i].CreatedAt.Before(msgs[j].CreatedAt)
		})
	}

	return msgs, hasMore, nil
}

// DeleteConversation removes all messages for the global conversation.
func (r ChatMessageRepository) DeleteConversation(ctx context.Context) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	_, err := r.sb.
		Delete("ai_chat_messages").
		Where(sq.Eq{"conversation_id": domain.GlobalConversationID}).
		ExecContext(spanCtx)

	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}
	return nil
}

// InitChatMessageRepository is a Symbiont initializer for ChatMessageRepository.
type InitChatMessageRepository struct {
	DB *sql.DB `resolve:""`
}

// Initialize registers the ChatMessageRepository in the dependency container.
func (r InitChatMessageRepository) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.ChatMessageRepository](NewChatMessageRepository(r.DB))
	return ctx, nil
}
