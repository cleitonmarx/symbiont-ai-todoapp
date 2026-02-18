package usecases

import (
	"context"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
)

// TodoSearchBuildResult is the output of TodoSearchBuilder.Build.
type TodoSearchBuildResult struct {
	Options              []domain.ListTodoOption
	EmbeddingTotalTokens int
}

type todoSearchClause struct {
	query      *string
	searchType *SearchType
}

// TodoSearchBuilder builds todo list options and centralizes validation plus
// optional similarity embedding generation for usecases.
type TodoSearchBuilder struct {
	llmClient         domain.LLMClient
	llmEmbeddingModel string

	status       *domain.TodoStatus
	dueAfter     *time.Time
	dueBefore    *time.Time
	sortBy       *string
	searchClause []todoSearchClause
}

// NewTodoSearchBuilder creates a new TodoSearchBuilder.
func NewTodoSearchBuilder(llmClient domain.LLMClient, llmEmbeddingModel string) *TodoSearchBuilder {
	return &TodoSearchBuilder{
		llmClient:         llmClient,
		llmEmbeddingModel: llmEmbeddingModel,
	}
}

// WithStatus sets an optional status filter.
func (b *TodoSearchBuilder) WithStatus(status *domain.TodoStatus) *TodoSearchBuilder {
	b.status = status
	return b
}

// WithSearch sets an optional search query and search type.
func (b *TodoSearchBuilder) WithSearch(query *string, searchType *SearchType) *TodoSearchBuilder {
	b.searchClause = append(b.searchClause, todoSearchClause{
		query:      query,
		searchType: searchType,
	})
	return b
}

// WithTitleContains sets the search query for a title substring match.
func (b *TodoSearchBuilder) WithTitleContains(query *string) *TodoSearchBuilder {
	b.searchClause = append(b.searchClause, todoSearchClause{
		query:      query,
		searchType: common.Ptr(SearchType_Title),
	})
	return b
}

// WithSimilaritySearch sets the search query for a similarity search.
func (b *TodoSearchBuilder) WithSimilaritySearch(query *string) *TodoSearchBuilder {
	b.searchClause = append(b.searchClause, todoSearchClause{
		query:      query,
		searchType: common.Ptr(SearchType_Similarity),
	})
	return b
}

// WithDueDateRange sets an optional due-date range filter.
func (b *TodoSearchBuilder) WithDueDateRange(dueAfter, dueBefore *time.Time) *TodoSearchBuilder {
	b.dueAfter = dueAfter
	b.dueBefore = dueBefore
	return b
}

// WithSortBy sets an optional sort filter.
func (b *TodoSearchBuilder) WithSortBy(sortBy *string) *TodoSearchBuilder {
	b.sortBy = sortBy
	return b
}

// Build validates configured filters and returns repository options.
func (b *TodoSearchBuilder) Build(ctx context.Context) (TodoSearchBuildResult, error) {
	if (b.dueAfter == nil) != (b.dueBefore == nil) {
		return TodoSearchBuildResult{}, domain.NewValidationErr("due_after and due_before must be provided together")
	}
	if b.dueAfter != nil && b.dueBefore != nil && b.dueAfter.After(*b.dueBefore) {
		return TodoSearchBuildResult{}, domain.NewValidationErr("due_after must be less than or equal to due_before")
	}

	if b.status != nil && *b.status != domain.TodoStatus_OPEN && *b.status != domain.TodoStatus_DONE {
		return TodoSearchBuildResult{}, domain.NewValidationErr("status must be either OPEN or DONE")
	}

	opts := []domain.ListTodoOption{}
	if b.status != nil {
		opts = append(opts, domain.WithStatus(*b.status))
	}
	if b.dueAfter != nil && b.dueBefore != nil {
		opts = append(opts, domain.WithDueDateRange(*b.dueAfter, *b.dueBefore))
	}
	if b.sortBy != nil {
		opts = append(opts, domain.WithSortBy(*b.sortBy))
	}

	var titleSearch *string
	similarityQuery := ""
	resolvedSearchCount := 0
	for _, clause := range b.searchClause {
		if clause.query == nil {
			continue
		}
		query := strings.TrimSpace(*clause.query)
		if query == "" {
			continue
		}
		if clause.searchType == nil {
			return TodoSearchBuildResult{}, domain.NewValidationErr("invalid search type")
		}

		switch *clause.searchType {
		case SearchType_Title:
			titleSearch = &query
		case SearchType_Similarity:
			similarityQuery = query
		default:
			return TodoSearchBuildResult{}, domain.NewValidationErr("invalid search type")
		}
		resolvedSearchCount++
	}

	if resolvedSearchCount > 1 {
		return TodoSearchBuildResult{}, domain.NewValidationErr("only one search query is allowed")
	}

	if titleSearch != nil {
		opts = append(opts, domain.WithTitleContains(*titleSearch))
	}

	if b.sortBy != nil && strings.HasPrefix(strings.ToLower(strings.TrimSpace(*b.sortBy)), "similarity") && similarityQuery == "" {
		return TodoSearchBuildResult{}, domain.NewValidationErr("search_by_similarity is required when using similarity sorting")
	}

	result := TodoSearchBuildResult{Options: opts}

	if similarityQuery == "" {
		return result, nil
	}

	if b.llmClient == nil {
		return TodoSearchBuildResult{}, domain.NewValidationErr("llm client is required for similarity search")
	}
	if strings.TrimSpace(b.llmEmbeddingModel) == "" {
		return TodoSearchBuildResult{}, domain.NewValidationErr("embedding model cannot be empty for similarity search")
	}

	resp, err := b.llmClient.EmbedSearch(ctx, b.llmEmbeddingModel, similarityQuery)
	if err != nil {
		return TodoSearchBuildResult{}, err
	}
	result.Options = append(result.Options, domain.WithEmbedding(resp.Embedding))
	result.EmbeddingTotalTokens = resp.TotalTokens
	return result, nil
}
