package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"

	"github.com/Masterminds/squirrel"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var chatFields = []string{
	"id",
	"conversation_id",
	"chat_role",
	"content",
	"tool_call_id",
	"tool_calls",
	"model",
	"created_at",
}

// ChatMessageRepository persists chat messages in Postgres.
type ChatMessageRepository struct {
	sb squirrel.StatementBuilderType
}

// NewChatMessageRepository creates a new ChatMessageRepository.
func NewChatMessageRepository(br squirrel.BaseRunner) ChatMessageRepository {
	return ChatMessageRepository{
		sb: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).RunWith(br),
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
			message.ChatRole,
			message.Content,
			message.ToolCallID,
			toolCallsJSON,
			message.Model,
			message.CreatedAt,
		)
	}

	_, err := insertQry.ExecContext(spanCtx)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}
	return nil
}

// ListChatMessages retrieves messages for the global conversation ordered by creation time.
// If limit > 0, returns up to the latest N messages; hasMore indicates if there are older messages.
func (r ChatMessageRepository) ListChatMessages(ctx context.Context, limit int) ([]domain.ChatMessage, bool, error) {
	spanCtx, span := telemetry.Start(ctx, trace.WithAttributes(
		attribute.Int("limit", limit),
	))
	defer span.End()

	qry := r.sb.
		Select(chatFields...).
		From("ai_chat_messages").
		Where(squirrel.Eq{"conversation_id": domain.GlobalConversationID}).
		OrderBy("created_at DESC")

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
			&m.ChatRole,
			&m.Content,
			&m.ToolCallID,
			&tcJSON,
			&m.Model,
			&m.CreatedAt,
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

	// Currently ordered DESC; reverse to ASC for chronological order
	sort.SliceStable(msgs, func(i, j int) bool {
		return msgs[i].CreatedAt.Before(msgs[j].CreatedAt)
	})

	return msgs, hasMore, nil
}

// DeleteConversation removes all messages for the global conversation.
func (r ChatMessageRepository) DeleteConversation(ctx context.Context) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	_, err := r.sb.
		Delete("ai_chat_messages").
		Where(squirrel.Eq{"conversation_id": domain.GlobalConversationID}).
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
