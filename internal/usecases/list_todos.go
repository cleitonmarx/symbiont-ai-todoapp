package usecases

import (
	"context"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
)

// SearchType defines the type of search to perform when listing todos.
type SearchType string

const (
	// SearchType_Title performs a case-insensitive substring match on todo titles.
	SearchType_Title SearchType = "title"
	// SearchType_Similarity uses vector similarity search based on the todo embeddings.
	SearchType_Similarity SearchType = "similarity"
)

// ListTodoParams holds the parameters for listing todos.
type ListTodoParams struct {
	Status     *domain.TodoStatus
	Search     *string
	SearchType *SearchType
	DueAfter   *time.Time
	DueBefore  *time.Time
	SortBy     *string
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
		params.Search = &query
	}
}

// WithSearchType creates a ListTodoOptions to specify the type of search (e.g., title, similarity).
func WithSearchType(searchType SearchType) ListTodoOptions {
	return func(params *ListTodoParams) {
		switch strings.ToLower(string(searchType)) {
		case string(SearchType_Title):
			searchType = SearchType_Title
		case string(SearchType_Similarity):
			searchType = SearchType_Similarity
		}
		params.SearchType = &searchType
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
	todoRepo        domain.TodoRepository
	semanticEncoder domain.SemanticEncoder
	embeddingModel  string
}

// NewListTodosImpl creates a new instance of ListTodosImpl.
func NewListTodosImpl(todoRepo domain.TodoRepository, semanticEncoder domain.SemanticEncoder, embeddingModel string) ListTodosImpl {
	return ListTodosImpl{
		todoRepo:        todoRepo,
		semanticEncoder: semanticEncoder,
		embeddingModel:  embeddingModel,
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

	builder := NewTodoSearchBuilder(lti.semanticEncoder, lti.embeddingModel).
		WithStatus(params.Status).
		WithDueDateRange(params.DueAfter, params.DueBefore).
		WithSortBy(params.SortBy).
		WithSearch(params.Search, params.SearchType)

	buildResult, err := builder.Build(spanCtx)
	if telemetry.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}
	if buildResult.EmbeddingTotalTokens > 0 {
		RecordLLMTokensEmbedding(spanCtx, buildResult.EmbeddingTotalTokens)
	}

	todos, hasMore, err := lti.todoRepo.ListTodos(spanCtx, page, pageSize, buildResult.Options...)
	if telemetry.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}
	return todos, hasMore, nil
}

// InitListTodos initializes the ListTodos use case and registers it in the dependency container.
type InitListTodos struct {
	TodoRepo        domain.TodoRepository  `resolve:""`
	SemanticEncoder domain.SemanticEncoder `resolve:""`
	EmbeddingModel  string                 `config:"LLM_EMBEDDING_MODEL"`
}

// Initialize initializes the ListTodosImpl use case and registers it in the dependency container.
func (ilt InitListTodos) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[ListTodos](NewListTodosImpl(ilt.TodoRepo, ilt.SemanticEncoder, ilt.EmbeddingModel))
	return ctx, nil
}
