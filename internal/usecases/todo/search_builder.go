package todo

import (
	"context"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/semantic"
	domain "github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/todo"
)

// SearchBuildResult is the output of SearchBuilder.Build.
type SearchBuildResult struct {
	Options              []domain.ListOption
	EmbeddingTotalTokens int
}

// searchClause represents a single search query and its type (e.g., title substring or similarity).
type searchClause struct {
	query      *string
	searchType *SearchType
}

// SearchBuilder builds todo list options and centralizes validation plus
// optional similarity embedding generation for usecases.
type SearchBuilder struct {
	status       *domain.Status
	dueAfter     *time.Time
	dueBefore    *time.Time
	sortBy       *string
	searchClause []searchClause
}

// NewSearchBuilder creates a new SearchBuilder.
func NewSearchBuilder() *SearchBuilder {
	return &SearchBuilder{}
}

// WithStatus sets an optional status filter.
func (b *SearchBuilder) WithStatus(status *domain.Status) *SearchBuilder {
	b.status = status
	return b
}

// WithSearch sets an optional search query and search type.
func (b *SearchBuilder) WithSearch(query *string, searchType *SearchType) *SearchBuilder {
	b.searchClause = append(b.searchClause, searchClause{
		query:      query,
		searchType: searchType,
	})
	return b
}

// WithTitleContains sets the search query for a title substring match.
func (b *SearchBuilder) WithTitleContains(query *string) *SearchBuilder {
	b.searchClause = append(b.searchClause, searchClause{
		query:      query,
		searchType: common.Ptr(SearchType_Title),
	})
	return b
}

// WithSimilaritySearch sets the search query for a similarity search.
func (b *SearchBuilder) WithSimilaritySearch(query *string) *SearchBuilder {
	b.searchClause = append(b.searchClause, searchClause{
		query:      query,
		searchType: common.Ptr(SearchType_Similarity),
	})
	return b
}

// WithDueDateRange sets an optional due-date range filter.
func (b *SearchBuilder) WithDueDateRange(dueAfter, dueBefore *time.Time) *SearchBuilder {
	b.dueAfter = dueAfter
	b.dueBefore = dueBefore
	return b
}

// WithSortBy sets an optional sort filter.
func (b *SearchBuilder) WithSortBy(sortBy *string) *SearchBuilder {
	b.sortBy = sortBy
	return b
}

// Validate checks that all configured filters and search options are consistent.
func (b *SearchBuilder) Validate() error {
	if (b.dueAfter == nil) != (b.dueBefore == nil) {
		return core.NewValidationErr("due_after and due_before must be provided together")
	}
	if b.dueAfter != nil && b.dueBefore != nil && b.dueAfter.After(*b.dueBefore) {
		return core.NewValidationErr("due_after must be less than or equal to due_before")
	}

	if b.status != nil && *b.status != domain.Status_OPEN && *b.status != domain.Status_DONE {
		return core.NewValidationErr("status must be either OPEN or DONE")
	}

	resolvedSearchCount := 0
	similarityQuery := ""
	for _, clause := range b.searchClause {
		if clause.query != nil && strings.TrimSpace(*clause.query) != "" {
			resolvedSearchCount++
			if clause.searchType == nil {
				return core.NewValidationErr("invalid search type")
			}
			switch *clause.searchType {
			case SearchType_Similarity:
				similarityQuery = strings.TrimSpace(*clause.query)
			case SearchType_Title:
			default:
				return core.NewValidationErr("invalid search type")
			}
		}
	}

	if resolvedSearchCount > 1 {
		return core.NewValidationErr("only one search query is allowed")
	}

	if b.sortBy != nil {
		sortBy := strings.TrimSpace(*b.sortBy)
		switch sortBy {
		case "dueDateAsc", "dueDateDesc", "createdAtAsc", "createdAtDesc", "similarityAsc", "similarityDesc":
			b.sortBy = &sortBy
		default:
			return core.NewValidationErr("sort_by is invalid")
		}
	}

	if b.sortBy != nil && strings.HasPrefix(strings.ToLower(strings.TrimSpace(*b.sortBy)), "similarity") && similarityQuery == "" {
		return core.NewValidationErr("search_by_similarity is required when using similarity sorting")
	}

	return nil
}

// Build validates configured filters and returns repository options.
func (b *SearchBuilder) Build(ctx context.Context, semanticEncoder semantic.Encoder, embeddingModel string) (SearchBuildResult, error) {
	if err := b.Validate(); err != nil {
		return SearchBuildResult{}, err
	}

	opts := []domain.ListOption{}
	if b.status != nil {
		opts = append(opts, domain.WithStatus(*b.status))
	}
	if b.dueAfter != nil && b.dueBefore != nil {
		opts = append(opts, domain.WithDueDateRange(*b.dueAfter, *b.dueBefore))
	}
	if b.sortBy != nil {
		opts = append(opts, domain.WithSortBy(*b.sortBy))
	}

	var (
		titleSearch     *string
		similarityQuery string
	)
	for _, clause := range b.searchClause {
		if clause.query == nil {
			continue
		}
		query := strings.TrimSpace(*clause.query)
		if query == "" {
			continue
		}

		switch *clause.searchType {
		case SearchType_Title:
			titleSearch = &query
		case SearchType_Similarity:
			similarityQuery = query
		}
	}

	if titleSearch != nil {
		opts = append(opts, domain.WithTitleContains(*titleSearch))
	}

	result := SearchBuildResult{Options: opts}

	if similarityQuery == "" {
		return result, nil
	}

	if semanticEncoder == nil {
		return SearchBuildResult{}, core.NewValidationErr("semantic encoder is required for similarity search")
	}
	if strings.TrimSpace(embeddingModel) == "" {
		return SearchBuildResult{}, core.NewValidationErr("embedding model cannot be empty for similarity search")
	}

	resp, err := semanticEncoder.VectorizeQuery(ctx, embeddingModel, similarityQuery)
	if err != nil {
		return SearchBuildResult{}, err
	}
	result.Options = append(result.Options, domain.WithEmbedding(resp.Vector))
	result.EmbeddingTotalTokens = resp.TotalTokens
	return result, nil
}
