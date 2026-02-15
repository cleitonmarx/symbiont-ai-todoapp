package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
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

// ConversationRepository is a PostgreSQL implementation of domain.ConversationRepository.
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
	source domain.ConversationTitleSource,
) (domain.Conversation, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	now := time.Now().UTC()
	input := domain.Conversation{
		ID:            uuid.New(),
		Title:         title,
		TitleSource:   source,
		LastMessageAt: nil,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := input.Validate(); telemetry.RecordErrorAndStatus(span, err) {
		return domain.Conversation{}, err
	}

	var created domain.Conversation
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
	if telemetry.RecordErrorAndStatus(span, err) {
		return domain.Conversation{}, err
	}

	return created, nil
}

// GetConversation retrieves a conversation by ID.
func (r ConversationRepository) GetConversation(
	ctx context.Context,
	conversationID uuid.UUID,
) (domain.Conversation, bool, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	var conversation domain.Conversation
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
	if telemetry.RecordErrorAndStatus(span, err) {
		if err == sql.ErrNoRows {
			return domain.Conversation{}, false, nil
		}
		return domain.Conversation{}, false, err
	}

	return conversation, true, nil
}

// UpdateConversation updates mutable fields for one conversation.
func (r ConversationRepository) UpdateConversation(
	ctx context.Context,
	conversation domain.Conversation,
) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	_, err := r.sb.
		Update("conversations").
		Set("title", conversation.Title).
		Set("title_source", conversation.TitleSource).
		Set("last_message_at", conversation.LastMessageAt).
		Set("updated_at", conversation.UpdatedAt).
		Where(squirrel.Eq{"id": conversation.ID}).
		ExecContext(spanCtx)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	return nil
}

// ListConversations returns paginated conversations ordered by last interaction recency.
func (r ConversationRepository) ListConversations(
	ctx context.Context,
	page int,
	pageSize int,
) ([]domain.Conversation, bool, error) {
	spanCtx, span := telemetry.Start(ctx, trace.WithAttributes(
		attribute.Int("page", page),
		attribute.Int("page_size", pageSize),
	))
	defer span.End()

	if page <= 0 {
		err := domain.NewValidationErr("page must be greater than 0")
		telemetry.RecordErrorAndStatus(span, err)
		return nil, false, err
	}
	if pageSize <= 0 {
		err := domain.NewValidationErr("page_size must be greater than 0")
		telemetry.RecordErrorAndStatus(span, err)
		return nil, false, err
	}

	rows, err := r.sb.
		Select(conversationFields...).
		From("conversations").
		OrderBy("last_message_at DESC NULLS LAST", "updated_at DESC", "created_at DESC").
		Limit(uint64(pageSize + 1)).
		Offset(uint64((page - 1) * pageSize)).
		QueryContext(spanCtx)
	if telemetry.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}
	defer rows.Close() //nolint:errcheck

	conversations := []domain.Conversation{}
	for rows.Next() {
		var conversation domain.Conversation
		err := rows.Scan(
			&conversation.ID,
			&conversation.Title,
			&conversation.TitleSource,
			&conversation.LastMessageAt,
			&conversation.CreatedAt,
			&conversation.UpdatedAt,
		)
		if telemetry.RecordErrorAndStatus(span, err) {
			return nil, false, err
		}

		conversations = append(conversations, conversation)
	}
	if err := rows.Err(); telemetry.RecordErrorAndStatus(span, err) {
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
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	_, err := r.sb.
		Delete("conversations").
		Where(squirrel.Eq{"id": conversationID}).
		ExecContext(spanCtx)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	return nil
}

// InitConversationRepository is a Symbiont initializer for ConversationRepository.
type InitConversationRepository struct {
	DB *sql.DB `resolve:""`
}

// Initialize registers the ConversationRepository in the dependency container.
func (i InitConversationRepository) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.ConversationRepository](NewConversationRepository(i.DB))
	return ctx, nil
}
