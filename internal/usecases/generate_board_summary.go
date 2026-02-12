package usecases

import (
	"context"
	"embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/telemetry"
	"github.com/google/uuid"
	"github.com/toon-format/toon-go"
	"go.yaml.in/yaml/v3"
)

// CompletedSummaryChannel is a channel type for sending processed domain.BoardSummary items.
// It is used in integration tests to verify summary generation.
type CompletedSummaryChannel chan domain.BoardSummary

// GenerateBoardSummary is the use case interface for generating a summary of the todo board.
type GenerateBoardSummary interface {
	Execute(ctx context.Context) error
}

// GenerateBoardSummaryImpl is the implementation of the GenerateBoardSummary use case.
type GenerateBoardSummaryImpl struct {
	repo               domain.BoardSummaryRepository
	timeProvider       domain.CurrentTimeProvider
	llmClient          domain.LLMClient
	model              string
	completedSummaryCh CompletedSummaryChannel
}

// NewGenerateBoardSummaryImpl creates a new instance of GenerateBoardSummaryImpl.
func NewGenerateBoardSummaryImpl(
	bsr domain.BoardSummaryRepository,
	tp domain.CurrentTimeProvider,
	c domain.LLMClient,
	m string,
	q CompletedSummaryChannel,

) GenerateBoardSummaryImpl {
	return GenerateBoardSummaryImpl{
		repo:               bsr,
		timeProvider:       tp,
		llmClient:          c,
		model:              m,
		completedSummaryCh: q,
	}
}

// Execute runs the use case to generate the board summary.
func (gs GenerateBoardSummaryImpl) Execute(ctx context.Context) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	summary, hasChanges, err := gs.generateBoardSummary(spanCtx)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	if !hasChanges {
		return nil
	}

	err = gs.repo.StoreSummary(spanCtx, summary)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	if gs.completedSummaryCh != nil {
		gs.completedSummaryCh <- summary
	}

	return nil
}

// generateBoardSummary calculates the new board summary content, compares it with the previous summary,
// and generates a new summary using the LLM if there are significant changes.
func (gs GenerateBoardSummaryImpl) generateBoardSummary(ctx context.Context) (domain.BoardSummary, bool, error) {

	new, err := gs.repo.CalculateSummaryContent(ctx)
	if err != nil {
		return domain.BoardSummary{}, false, fmt.Errorf("failed to calculate summary content: %w", err)
	}

	previous, found, err := gs.repo.GetLatestSummary(ctx)
	if err != nil {
		return domain.BoardSummary{}, false, fmt.Errorf("failed to get latest summary: %w", err)
	}
	if !found {
		previous.Content.Summary = "no previous summary"
	}

	if hasContentChanges := new.DiffersFrom(previous.Content); !hasContentChanges {
		return domain.BoardSummary{}, false, nil
	}

	now := gs.timeProvider.Now()
	promptMessages, err := buildPromptMessages(new, previous.Content)
	if err != nil {
		return domain.BoardSummary{}, false, fmt.Errorf("failed to build prompt: %w", err)
	}

	req := domain.LLMChatRequest{
		Model:       gs.model,
		Stream:      false,
		Temperature: common.Ptr(1.2),
		TopP:        common.Ptr(0.95),
		Messages:    promptMessages,
	}

	resp, err := gs.llmClient.Chat(ctx, req)
	if err != nil {
		return domain.BoardSummary{}, false, err
	}

	new.Summary = strings.TrimSpace(resp.Content)
	new.Summary = applySummarySafetyGuards(new.Summary, new)

	RecordLLMTokensUsed(ctx, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	summary := domain.BoardSummary{
		ID:            uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Content:       new,
		Model:         gs.model,
		GeneratedAt:   now,
		SourceVersion: 1,
	}

	return summary, true, nil
}

//go:embed prompts/summary.yml
var summaryPrompt embed.FS

// buildPromptMessages constructs the LLM messages for the summary prompt.
func buildPromptMessages(new domain.BoardSummaryContent, previous domain.BoardSummaryContent) ([]domain.LLMChatMessage, error) {
	inputTOON, err := marshalSummaryContent(new)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal summary content: %w", err)
	}

	previousTOON, err := marshalSummaryContent(previous)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal previous summary content: %w", err)
	}

	hints := new.BuildComparisonHints(previous)
	completedCandidatesText := "none"
	if len(hints.CompletedCandidates) > 0 {
		completedCandidatesText = strings.Join(hints.CompletedCandidates, "; ")
	}

	file, err := summaryPrompt.Open("prompts/summary.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to open summary prompt: %w", err)
	}
	defer file.Close() //nolint:errcheck

	messages := []domain.LLMChatMessage{}
	err = yaml.NewDecoder(file).Decode(&messages)
	if err != nil {
		return nil, fmt.Errorf("failed to decode summary prompt: %w", err)
	}

	for i, msg := range messages {
		msg.Content = fmt.Sprintf(
			msg.Content,
			inputTOON,
			previousTOON,
			completedCandidatesText,
			hints.DoneDelta,
			hints.OverdueTitles,
			hints.NearDeadlineTitles,
			hints.NextUpOverdue,
			hints.NextUpDueSoon,
			hints.NextUpUpcoming,
			hints.NextUpFuture,
		)
		messages[i] = msg
	}

	return messages, nil
}

var (
	reNoOverdueTasks    = regexp.MustCompile(`(?i)\bno overdue tasks?\b`)
	reNoTasksAreOverdue = regexp.MustCompile(`(?i)\bno tasks are overdue\b`)
	reNothingIsOverdue  = regexp.MustCompile(`(?i)\bnothing is overdue\b`)
	reOverdueQualifier  = regexp.MustCompile(`(?i)\boverdue\s+`)
	reLateQualifier     = regexp.MustCompile(`(?i)\blate\s+`)
	rePastDueQualifier  = regexp.MustCompile(`(?i)\bpast[- ]due\s+`)
	reExtraSpaces       = regexp.MustCompile(`\s{2,}`)
	reSpaceBeforePunct  = regexp.MustCompile(`\s+([,.;:!?])`)
)

// applySummarySafetyGuards cleans the generated summary text to prevent certain phrases
// from appearing if they are not supported by the current board facts.
func applySummarySafetyGuards(summary string, content domain.BoardSummaryContent) string {
	cleaned := strings.TrimSpace(summary)
	if cleaned == "" {
		return cleaned
	}

	// Basic guardrail to prevent markdown formatting from leaking into the summary,
	// which can cause issues for some LLMs and is not needed for our use case.
	cleaned = strings.ReplaceAll(cleaned, "**", "")
	// Guardrail for weaker models: if there are no overdue tasks in current facts,
	// do not allow overdue/late phrasing to leak into the final summary text.
	if len(content.Overdue) == 0 {

		// Use placeholders to prevent regexes from interfering with each other
		cleaned = reNoOverdueTasks.ReplaceAllString(cleaned, "__NO_OVERDUE_TASKS__")
		cleaned = reNoTasksAreOverdue.ReplaceAllString(cleaned, "__NO_TASKS_ARE_OVERDUE__")
		cleaned = reNothingIsOverdue.ReplaceAllString(cleaned, "__NOTHING_IS_OVERDUE__")

		// Remove any remaining overdue/late qualifiers that are not supported by current facts
		cleaned = reOverdueQualifier.ReplaceAllString(cleaned, "")
		cleaned = reLateQualifier.ReplaceAllString(cleaned, "")
		cleaned = rePastDueQualifier.ReplaceAllString(cleaned, "")
		cleaned = reExtraSpaces.ReplaceAllString(cleaned, " ")
		cleaned = reSpaceBeforePunct.ReplaceAllString(cleaned, "$1")
		cleaned = strings.TrimSpace(cleaned)

		// Restore placeholders back to user-friendly text
		cleaned = strings.ReplaceAll(cleaned, "__NO_OVERDUE_TASKS__", "no overdue tasks")
		cleaned = strings.ReplaceAll(cleaned, "__NO_TASKS_ARE_OVERDUE__", "no tasks are overdue")
		cleaned = strings.ReplaceAll(cleaned, "__NOTHING_IS_OVERDUE__", "nothing is overdue")
	}

	return cleaned
}

// marshalSummaryContent converts the BoardSummaryContent struct into a TOON string for LLM input.
func marshalSummaryContent(sc domain.BoardSummaryContent) (string, error) {
	summaryContentTOON, err := toon.MarshalString(sc, toon.WithLengthMarkers(true))
	if err != nil {
		return "", fmt.Errorf("failed to marshal summary content: %w", err)
	}

	return summaryContentTOON, nil
}

// InitGenerateBoardSummary initializes the GenerateBoardSummary use case.
type InitGenerateBoardSummary struct {
	SummaryRepo  domain.BoardSummaryRepository `resolve:""`
	TimeProvider domain.CurrentTimeProvider    `resolve:""`
	LLMClient    domain.LLMClient              `resolve:""`
	Model        string                        `config:"LLM_SUMMARY_MODEL"`
}

// Initialize registers the GenerateBoardSummary use case implementation.
func (igbs InitGenerateBoardSummary) Initialize(ctx context.Context) (context.Context, error) {
	queue, _ := depend.Resolve[CompletedSummaryChannel]()
	depend.Register[GenerateBoardSummary](NewGenerateBoardSummaryImpl(
		igbs.SummaryRepo, igbs.TimeProvider, igbs.LLMClient, igbs.Model, queue,
	))
	return ctx, nil
}
