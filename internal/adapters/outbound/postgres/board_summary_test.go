package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBoardSummaryRepository_StoreSummary(t *testing.T) {
	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	generatedAt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	summary := domain.BoardSummary{
		ID:            fixedUUID,
		Model:         "mistral",
		GeneratedAt:   generatedAt,
		SourceVersion: 1,
		Content: domain.BoardSummaryContent{
			Counts: domain.TodoStatusCounts{
				Open: 3,
				Done: 5,
			},
			NextUp: []domain.NextUpTodoItem{
				{
					Title:  "Pay electricity bill",
					Reason: "Overdue by 6 days",
				},
			},
			Overdue: []string{
				"Pay electricity bill",
				"Renew car insurance",
			},
			NearDeadline: []string{
				"Schedule annual medical checkup",
			},
			Summary: "You have 2 overdue tasks and 1 task due tomorrow.",
		},
	}

	contentJSON, _ := json.Marshal(summary.Content)

	tests := map[string]struct {
		setExpectations func(mock sqlmock.Sqlmock)
		summary         domain.BoardSummary
		shouldError     bool
	}{
		"success-insert": {
			summary: summary,
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO board_summary (id,summary,model,generated_at,source_version) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (id) DO UPDATE SET summary = EXCLUDED.summary, model = EXCLUDED.model, generated_at = EXCLUDED.generated_at, source_version = EXCLUDED.source_version`).
					WithArgs(
						summary.ID,
						contentJSON,
						summary.Model,
						summary.GeneratedAt,
						summary.SourceVersion,
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			shouldError: false,
		},
		"success-upsert": {
			summary: summary,
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO board_summary (id,summary,model,generated_at,source_version) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (id) DO UPDATE SET summary = EXCLUDED.summary, model = EXCLUDED.model, generated_at = EXCLUDED.generated_at, source_version = EXCLUDED.source_version`).
					WithArgs(
						summary.ID,
						contentJSON,
						summary.Model,
						summary.GeneratedAt,
						summary.SourceVersion,
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			shouldError: false,
		},
		"database-error": {
			summary: summary,
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`INSERT INTO board_summary (id,summary,model,generated_at,source_version) VALUES ($1,$2,$3,$4,$5) ON CONFLICT (id) DO UPDATE SET summary = EXCLUDED.summary, model = EXCLUDED.model, generated_at = EXCLUDED.generated_at, source_version = EXCLUDED.source_version`).
					WithArgs(
						summary.ID,
						contentJSON,
						summary.Model,
						summary.GeneratedAt,
						summary.SourceVersion,
					).
					WillReturnError(sql.ErrConnDone)
			},
			shouldError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			require.NoError(t, err)
			defer db.Close() // nolint:errcheck

			tt.setExpectations(mock)

			repo := NewBoardSummaryRepository(db)
			gotErr := repo.StoreSummary(context.Background(), tt.summary)

			if tt.shouldError {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestBoardSummaryRepository_GetLatestSummary(t *testing.T) {
	fixedUUID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	generatedAt := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	summary := domain.BoardSummary{
		ID:            fixedUUID,
		Model:         "mistral",
		GeneratedAt:   generatedAt,
		SourceVersion: 1,
		Content: domain.BoardSummaryContent{
			Counts: domain.TodoStatusCounts{
				Open: 3,
				Done: 5,
			},
			NextUp: []domain.NextUpTodoItem{
				{
					Title:  "Pay electricity bill",
					Reason: "Overdue by 6 days",
				},
			},
			Overdue: []string{
				"Pay electricity bill",
				"Renew car insurance",
			},
			NearDeadline: []string{
				"Schedule annual medical checkup",
			},
			Summary: "You have 2 overdue tasks and 1 task due tomorrow.",
		},
	}

	contentJSON, _ := json.Marshal(summary.Content)

	tests := map[string]struct {
		setExpectations func(mock sqlmock.Sqlmock)
		expectedSummary domain.BoardSummary
		expectedFound   bool
		shouldError     bool
	}{
		"success": {
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(boardSummaryFields).
					AddRow(
						summary.ID,
						contentJSON,
						summary.Model,
						summary.GeneratedAt,
						summary.SourceVersion,
					)
				mock.ExpectQuery(`SELECT id, summary, model, generated_at, source_version FROM board_summary ORDER BY generated_at DESC LIMIT 1`).
					WillReturnRows(rows)
			},
			expectedSummary: summary,
			expectedFound:   true,
			shouldError:     false,
		},
		"not-found": {
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, summary, model, generated_at, source_version FROM board_summary ORDER BY generated_at DESC LIMIT 1`).
					WillReturnError(sql.ErrNoRows)
			},
			expectedSummary: domain.BoardSummary{},
			expectedFound:   false,
			shouldError:     false,
		},
		"database-error": {
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT id, summary, model, generated_at, source_version FROM board_summary ORDER BY generated_at DESC LIMIT 1`).
					WillReturnError(sql.ErrConnDone)
			},
			expectedSummary: domain.BoardSummary{},
			expectedFound:   false,
			shouldError:     true,
		},
		"unmarshal-error": {
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(boardSummaryFields).
					AddRow(
						summary.ID,
						[]byte("invalid json"),
						summary.Model,
						summary.GeneratedAt,
						summary.SourceVersion,
					)
				mock.ExpectQuery(`SELECT id, summary, model, generated_at, source_version FROM board_summary ORDER BY generated_at DESC LIMIT 1`).
					WillReturnRows(rows)
			},
			expectedSummary: domain.BoardSummary{},
			expectedFound:   false,
			shouldError:     true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			require.NoError(t, err)
			defer db.Close() // nolint:errcheck

			tt.setExpectations(mock)

			repo := NewBoardSummaryRepository(db)
			got, found, gotErr := repo.GetLatestSummary(context.Background())

			if tt.shouldError {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
				assert.Equal(t, tt.expectedFound, found)
				assert.Equal(t, tt.expectedSummary.ID, got.ID)
				assert.Equal(t, tt.expectedSummary.Model, got.Model)
				assert.Equal(t, tt.expectedSummary.Content.Counts, got.Content.Counts)
				assert.Equal(t, tt.expectedSummary.Content.Summary, got.Content.Summary)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestBoardSummaryRepository_CalculateSummaryContent(t *testing.T) {
	tests := map[string]struct {
		setExpectations func(mock sqlmock.Sqlmock)
		expectedSummary domain.BoardSummaryContent
		expectedFound   bool
		shouldError     bool
	}{
		"success": {
			setExpectations: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"counts", "overdue", "near_deadline", "next_up"}).
					AddRow(
						[]byte(`{"OPEN":4,"DONE":6}`),
						[]byte(`["File annual report","Pay credit card bill"]`),
						[]byte(`["Book flight tickets"]`),
						[]byte(`[{"title":"Submit tax documents","reason":"Due in 2 days"}]`),
					)

				mock.ExpectQuery(boardSummaryCTEQry + ` SELECT stats.counts, near_deadline.overdue, near_deadline.near_deadline, next_tasks.next_up FROM stats, near_deadline, next_tasks`).
					WillReturnRows(rows)
			},
			expectedSummary: domain.BoardSummaryContent{
				Counts: domain.TodoStatusCounts{
					Open: 4,
					Done: 6,
				},
				NextUp: []domain.NextUpTodoItem{
					{
						Title:  "Submit tax documents",
						Reason: "Due in 2 days",
					},
				},
				Overdue: []string{
					"File annual report",
					"Pay credit card bill",
				},
				NearDeadline: []string{
					"Book flight tickets",
				},
			},
			shouldError: false,
		},
		"database-error": {
			setExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(boardSummaryCTEQry + ` SELECT stats.counts, near_deadline.overdue, near_deadline.near_deadline, next_tasks.next_up FROM stats, near_deadline, next_tasks`).
					WillReturnError(sql.ErrConnDone)
			},
			expectedSummary: domain.BoardSummaryContent{},
			shouldError:     true,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
			require.NoError(t, err)
			defer db.Close() // nolint:errcheck

			tt.setExpectations(mock)

			repo := NewBoardSummaryRepository(db)
			got, gotErr := repo.CalculateSummaryContent(context.Background())

			if tt.shouldError {
				assert.Error(t, gotErr)
			} else {
				assert.NoError(t, gotErr)
				assert.Equal(t, tt.expectedSummary, got)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestInitBoardSummaryRepository_Initialize(t *testing.T) {
	i := &InitBoardSummaryRepository{
		DB: &sql.DB{},
	}

	_, err := i.Initialize(context.Background())
	assert.NoError(t, err)

	_, err = depend.Resolve[domain.BoardSummaryRepository]()
	assert.NoError(t, err)
}
