package todo

import (
	"context"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	domain "github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/metrics"
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
	Status     *domain.Status
	Search     *string
	SearchType *SearchType
	DueAfter   *time.Time
	DueBefore  *time.Time
	SortBy     *string
}

// ListTodoOptions defines a function type for specifying options when listing todos.
type ListTodoOptions func(*ListTodoParams)

// WithStatus creates a ListTodoOptions to filter todos by status.
func WithStatus(status domain.Status) ListTodoOptions {
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

// List defines the interface for the list use case.
type List interface {
	Query(ctx context.Context, page int, pageSize int, opts ...ListTodoOptions) ([]domain.Todo, bool, error)
}

// ListImpl is the implementation of the list use case.
type ListImpl struct {
	todoRepo        domain.Repository
	semanticEncoder semantic.Encoder
	embeddingModel  string
}

// NewListImpl creates a new instance of ListImpl.
func NewListImpl(todoRepo domain.Repository, semanticEncoder semantic.Encoder, embeddingModel string) ListImpl {
	return ListImpl{
		todoRepo:        todoRepo,
		semanticEncoder: semanticEncoder,
		embeddingModel:  embeddingModel,
	}
}

// Query retrieves a list of todo items with pagination support.
func (lti ListImpl) Query(ctx context.Context, page int, pageSize int, opts ...ListTodoOptions) ([]domain.Todo, bool, error) {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	params := ListTodoParams{}
	for _, opt := range opts {
		opt(&params)
	}

	builder := NewTodoSearchBuilder().
		WithStatus(params.Status).
		WithDueDateRange(params.DueAfter, params.DueBefore).
		WithSortBy(params.SortBy).
		WithSearch(params.Search, params.SearchType)

	buildResult, err := builder.Build(spanCtx, lti.semanticEncoder, lti.embeddingModel)
	if telemetry.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}
	if buildResult.EmbeddingTotalTokens > 0 {
		metrics.RecordLLMTokensEmbedding(spanCtx, buildResult.EmbeddingTotalTokens)
	}

	todos, hasMore, err := lti.todoRepo.ListTodos(spanCtx, page, pageSize, buildResult.Options...)
	if telemetry.RecordErrorAndStatus(span, err) {
		return nil, false, err
	}
	return todos, hasMore, nil
}
