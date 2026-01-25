package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var (
	boardSummaryFields = []string{
		"id",
		"summary",
		"model",
		"generated_at",
		"source_version",
	}
)

// BoardSummaryRepository is a PostgreSQL implementation of domain.BoardSummaryRepository.
type BoardSummaryRepository struct {
	db    *sql.DB
	pqsql squirrel.StatementBuilderType
}

// NewBoardSummaryRepository creates a new instance of BoardSummaryRepository.
func NewBoardSummaryRepository(db *sql.DB) BoardSummaryRepository {
	return BoardSummaryRepository{
		db:    db,
		pqsql: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar).RunWith(db),
	}
}

// StoreSummary stores a board summary in the database, updating if it already exists.
func (bsr BoardSummaryRepository) StoreSummary(ctx context.Context, summary domain.BoardSummary) error {
	spanCtx, span := tracing.Start(ctx, trace.WithAttributes(
		attribute.String("summary_id", summary.ID.String()),
		attribute.String("model", summary.Model),
	))
	defer span.End()

	// Marshal the content to JSON
	contentJSON, err := json.Marshal(summary.Content)
	if tracing.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to marshal summary content: %w", err)
	}

	query := bsr.pqsql.
		Insert("board_summary").
		Columns(
			boardSummaryFields...,
		).
		Values(
			summary.ID,
			contentJSON,
			summary.Model,
			summary.GeneratedAt,
			summary.SourceVersion,
		).
		Suffix(`ON CONFLICT (id) DO UPDATE SET
            summary = EXCLUDED.summary,
            model = EXCLUDED.model,
            generated_at = EXCLUDED.generated_at,
            source_version = EXCLUDED.source_version`)

	_, err = query.ExecContext(spanCtx)

	if tracing.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to store summary: %w", err)
	}

	return nil
}

// GetLatestSummary retrieves the most recently generated board summary.
func (bsr BoardSummaryRepository) GetLatestSummary(ctx context.Context) (domain.BoardSummary, bool, error) {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	var summary domain.BoardSummary
	var contentJSON []byte

	err := bsr.pqsql.
		Select(
			boardSummaryFields...,
		).
		From("board_summary").
		OrderBy("generated_at DESC").
		Limit(1).
		QueryRowContext(spanCtx).
		Scan(
			&summary.ID,
			&contentJSON,
			&summary.Model,
			&summary.GeneratedAt,
			&summary.SourceVersion,
		)

	if tracing.RecordErrorAndStatus(span, err) {
		if err == sql.ErrNoRows {
			return domain.BoardSummary{}, false, nil
		}
		return domain.BoardSummary{}, false, err
	}

	// Unmarshal the JSON content
	err = json.Unmarshal(contentJSON, &summary.Content)
	if tracing.RecordErrorAndStatus(span, err) {
		return domain.BoardSummary{}, false, fmt.Errorf("failed to unmarshal summary content: %w", err)
	}

	return summary, true, nil
}

// InitBoardSummaryRepository is a Symbiont initializer for BoardSummaryRepository.
type InitBoardSummaryRepository struct {
	DB *sql.DB `resolve:""`
}

// Initialize registers the BoardSummaryRepository in the dependency container.
func (ibsr InitBoardSummaryRepository) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[domain.BoardSummaryRepository](NewBoardSummaryRepository(ibsr.DB))
	return ctx, nil
}
