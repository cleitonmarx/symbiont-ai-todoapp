package usecases

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTodoSearchBuilder_Build(t *testing.T) {
	dueAfter := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	dueBefore := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
	done := domain.TodoStatus_DONE
	dueBeforeEarlier := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	searchGroceries := "groceries"
	searchUrgent := "urgent"
	searchUrgentSpaced := "  urgent  "
	searchBlank := "   "
	searchReport := "report"
	searchMeeting := "meeting"
	sortDueDateAsc := "dueDateAsc"
	sortSimilarityAsc := "similarityAsc"

	type searchInput struct {
		query      *string
		searchType *SearchType
	}
	type testCase struct {
		model      string
		status     *domain.TodoStatus
		dueAfter   *time.Time
		dueBefore  *time.Time
		sortBy     *string
		searches   []searchInput
		setupMocks func(t *testing.T, semanticEncoder *domain.MockSemanticEncoder)
		wantErr    string
		assertRes  func(t *testing.T, semanticEncoder *domain.MockSemanticEncoder, res TodoSearchBuildResult)
	}

	tests := map[string]testCase{
		"builds-options-without-similarity": {
			status:    &done,
			dueAfter:  &dueAfter,
			dueBefore: &dueBefore,
			sortBy:    &sortDueDateAsc,
			searches: []searchInput{
				{query: &searchGroceries, searchType: common.Ptr(SearchType_Title)},
			},
			assertRes: func(t *testing.T, _ *domain.MockSemanticEncoder, res TodoSearchBuildResult) {
				assert.Equal(t, 0, res.EmbeddingTotalTokens)
				params := domain.ListTodosParams{}
				for _, opt := range res.Options {
					opt(&params)
				}
				if assert.NotNil(t, params.Status) {
					assert.Equal(t, done, *params.Status)
				}
				if assert.NotNil(t, params.TitleContains) {
					assert.Equal(t, "groceries", *params.TitleContains)
				}
				if assert.NotNil(t, params.DueAfter) {
					assert.Equal(t, dueAfter, *params.DueAfter)
				}
				if assert.NotNil(t, params.DueBefore) {
					assert.Equal(t, dueBefore, *params.DueBefore)
				}
				if assert.NotNil(t, params.SortBy) {
					assert.Equal(t, "dueDate", params.SortBy.Field)
					assert.Equal(t, "ASC", params.SortBy.Direction)
				}
			},
		},
		"builds-options-with-similarity-embedding": {
			model: "embedding-model",
			searches: []searchInput{
				{query: &searchUrgentSpaced, searchType: common.Ptr(SearchType_Similarity)},
			},
			setupMocks: func(t *testing.T, semanticEncoder *domain.MockSemanticEncoder) {
				semanticEncoder.EXPECT().
					VectorizeQuery(mock.Anything, "embedding-model", "urgent").
					Return(domain.EmbeddingVector{
						Vector:      []float64{0.1, 0.2},
						TotalTokens: 17,
					}, nil).
					Once()
			},
			assertRes: func(t *testing.T, _ *domain.MockSemanticEncoder, res TodoSearchBuildResult) {
				assert.Equal(t, 17, res.EmbeddingTotalTokens)
				params := domain.ListTodosParams{}
				for _, opt := range res.Options {
					opt(&params)
				}
				assert.Equal(t, []float64{0.1, 0.2}, params.Embedding)
			},
		},
		"does-not-embed-with-blank-similarity-query": {
			model: "embedding-model",
			searches: []searchInput{
				{query: &searchBlank, searchType: common.Ptr(SearchType_Similarity)},
			},
			assertRes: func(t *testing.T, semanticEncoder *domain.MockSemanticEncoder, res TodoSearchBuildResult) {
				assert.Equal(t, 0, res.EmbeddingTotalTokens)
				semanticEncoder.AssertNotCalled(t, "VectorizeQuery", mock.Anything, "embedding-model", mock.Anything)
			},
		},
		"fails-on-partial-due-range": {
			dueAfter: &dueAfter,
			wantErr:  "due_after and due_before must be provided together",
		},
		"fails-on-invalid-due-range-order": {
			dueAfter:  &dueAfter,
			dueBefore: &dueBeforeEarlier,
			wantErr:   "due_after must be less than or equal to due_before",
		},
		"fails-on-similarity-sort-without-query": {
			sortBy:  &sortSimilarityAsc,
			wantErr: "search_by_similarity is required when using similarity sorting",
		},
		"fails-when-embedding-model-missing-for-similarity": {
			searches: []searchInput{
				{query: &searchUrgent, searchType: common.Ptr(SearchType_Similarity)},
			},
			wantErr: "embedding model cannot be empty for similarity search",
		},
		"returns-embedding-error": {
			model: "embedding-model",
			searches: []searchInput{
				{query: &searchUrgent, searchType: common.Ptr(SearchType_Similarity)},
			},
			setupMocks: func(t *testing.T, semanticEncoder *domain.MockSemanticEncoder) {
				semanticEncoder.EXPECT().
					VectorizeQuery(mock.Anything, "embedding-model", "urgent").
					Return(domain.EmbeddingVector{}, errors.New("embedding failed")).
					Once()
			},
			wantErr: "embedding failed",
		},
		"fails-with-invalid-search-type": {
			searches: []searchInput{
				{query: &searchReport, searchType: nil},
			},
			wantErr: "invalid search type",
		},
		"fails-when-multiple-search-queries-are-provided": {
			searches: []searchInput{
				{query: &searchMeeting, searchType: common.Ptr(SearchType_Similarity)},
				{query: &searchReport, searchType: common.Ptr(SearchType_Title)},
			},
			wantErr: "only one search query is allowed",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			semanticEncoder := domain.NewMockSemanticEncoder(t)
			if tt.setupMocks != nil {
				tt.setupMocks(t, semanticEncoder)
			}

			builder := NewTodoSearchBuilder(semanticEncoder, tt.model).
				WithStatus(tt.status).
				WithDueDateRange(tt.dueAfter, tt.dueBefore).
				WithSortBy(tt.sortBy)
			for _, search := range tt.searches {
				builder.WithSearch(search.query, search.searchType)
			}

			res, err := builder.Build(context.Background())
			if tt.wantErr != "" {
				if assert.Error(t, err) {
					assert.Equal(t, tt.wantErr, err.Error())
				}
				return
			}

			if assert.NoError(t, err) && tt.assertRes != nil {
				tt.assertRes(t, semanticEncoder, res)
			}
		})
	}
}
