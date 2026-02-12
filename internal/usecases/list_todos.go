package usecases

import (
	"context"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/telemetry"
)

// ListTodoParams holds the parameters for listing todos.
type ListTodoParams struct {
	Status    *domain.TodoStatus
	Query     *string
	DueAfter  *time.Time
	DueBefore *time.Time
	SortBy    *string
}

// ListTodoOptions defines a function type for specifying options when listing todos.
type ListTodoOptions func(*ListTodoParams)

// WithStatus creates a ListTodoOptions to filter todos by status.
func WithStatus(status domain.TodoStatus) ListTodoOptions {
	return func(params *ListTodoParams) {
		params.Status = &status
	}
}

// WithSearchQuery creates a ListTodoOptions to filter todos by a search query.
func WithSearchQuery(query string) ListTodoOptions {
	return func(params *ListTodoParams) {
		params.Query = &query
	}
}

// WithDueDateRange creates a ListTodoOptions to filter todos by due date range.
func WithDueDateRange(dueAfter, dueBefore time.Time) ListTodoOptions {
	return func(params *ListTodoParams) {
		params.DueAfter = &dueAfter
		params.DueBefore = &dueBefore
	}
}

// WithSortBy creates a ListTodoOptions to specify sorting criteria.
func WithSortBy(sortBy string) ListTodoOptions {
	return func(params *ListTodoParams) {
		params.SortBy = &sortBy
	}
}

// ListTodos defines the interface for the ListTodos use case.
type ListTodos interface {
	Query(ctx context.Context, page int, pageSize int, opts ...ListTodoOptions) ([]domain.Todo, bool, error)
}

// ListTodosImpl is the implementation of the ListTodos use case.
type ListTodosImpl struct {
	todoRepo          domain.TodoRepository
	llmClient         domain.LLMClient
	llmEmbeddingModel string
}

// NewListTodosImpl creates a new instance of ListTodosImpl.
func NewListTodosImpl(todoRepo domain.TodoRepository, llmClient domain.LLMClient, llmEmbeddingModel string) ListTodosImpl {
	return ListTodosImpl{
		todoRepo:          todoRepo,
		llmClient:         llmClient,
		llmEmbeddingModel: llmEmbeddingModel,
	}
}

// Query retrieves a list of todo items with pagination support.
func (lti ListTodosImpl) Query(ctx context.Context, page int, pageSize int, opts ...ListTodoOptions) ([]domain.Todo, bool, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	params := ListTodoParams{}
	for _, opt := range opts {
		opt(&params)
	}

	var queryOpts []domain.ListTodoOptions
	if params.Status != nil {
		queryOpts = append(queryOpts, domain.WithStatus(*params.Status))
	}
	if params.Query != nil {
		resp, err := lti.llmClient.Embed(spanCtx, lti.llmEmbeddingModel, *params.Query)
		if telemetry.RecordErrorAndStatus(span, err) {
			return nil, false, err
		}

		RecordLLMTokensEmbedding(spanCtx, resp.TotalTokens)

		queryOpts = append(queryOpts, domain.WithEmbedding(resp.Embedding))
	}
	if params.DueAfter != nil && params.DueBefore != nil {
		queryOpts = append(queryOpts, domain.WithDueDateRange(*params.DueAfter, *params.DueBefore))
	}
	if params.SortBy != nil {
		queryOpts = append(queryOpts, domain.WithSortBy(*params.SortBy))
	}

	todos, hasMore, err := lti.todoRepo.ListTodos(spanCtx, page, pageSize, queryOpts...)
	if telemetry.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}
	return todos, hasMore, nil
}

// InitListTodos initializes the ListTodos use case and registers it in the dependency container.
type InitListTodos struct {
	TodoRepo          domain.TodoRepository `resolve:""`
	LLMClient         domain.LLMClient      `resolve:""`
	LLMEmbeddingModel string                `config:"LLM_EMBEDDING_MODEL"`
}

// Initialize initializes the ListTodosImpl use case and registers it in the dependency container.
func (ilt InitListTodos) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[ListTodos](NewListTodosImpl(ilt.TodoRepo, ilt.LLMClient, ilt.LLMEmbeddingModel))
	return ctx, nil
}
