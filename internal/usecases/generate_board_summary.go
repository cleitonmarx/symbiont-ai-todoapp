package usecases

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/tracing"
	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"
)

// CompletedSummaryQueue is a channel type for sending processed domain.BoardSummary items.
// It is used in integration tests to verify summary generation.
type CompletedSummaryQueue chan domain.BoardSummary

// GenerateBoardSummary is the use case interface for generating a summary of the todo board.
type GenerateBoardSummary interface {
	Execute(ctx context.Context) error
}

// GenerateBoardSummaryImpl is the implementation of the GenerateBoardSummary use case.
type GenerateBoardSummaryImpl struct {
	summaryRepo  domain.BoardSummaryRepository
	timeProvider domain.CurrentTimeProvider
	llmClient    domain.LLMClient
	model        string
	queue        CompletedSummaryQueue
	// Add dependencies here if needed
}

// NewGenerateBoardSummaryImpl creates a new instance of GenerateBoardSummaryImpl.
func NewGenerateBoardSummaryImpl(
	bsr domain.BoardSummaryRepository,
	tp domain.CurrentTimeProvider,
	c domain.LLMClient,
	m string,
	q CompletedSummaryQueue,

) GenerateBoardSummaryImpl {
	return GenerateBoardSummaryImpl{
		summaryRepo:  bsr,
		timeProvider: tp,
		llmClient:    c,
		model:        m,
		queue:        q,
	}
}

// Execute runs the use case to generate the board summary.
func (gs GenerateBoardSummaryImpl) Execute(ctx context.Context) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	summary, err := gs.generateBoardSummary(spanCtx)
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	err = gs.summaryRepo.StoreSummary(spanCtx, summary)
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	if gs.queue != nil {
		gs.queue <- summary
	}

	return nil
}

func (gs GenerateBoardSummaryImpl) generateBoardSummary(ctx context.Context) (domain.BoardSummary, error) {

	new, err := gs.summaryRepo.CalculateSummaryContent(ctx)
	if err != nil {
		return domain.BoardSummary{}, fmt.Errorf("failed to calculate summary content: %w", err)
	}

	previous, found, err := gs.summaryRepo.GetLatestSummary(ctx)
	if err != nil {
		return domain.BoardSummary{}, fmt.Errorf("failed to get latest summary: %w", err)
	}
	if !found {
		previous.Content.Summary = "no previous summary"
	}

	now := gs.timeProvider.Now()
	promptMessages, err := buildPromptMessages(new, previous.Content, now)
	if err != nil {
		return domain.BoardSummary{}, fmt.Errorf("failed to build prompt: %w", err)
	}

	req := domain.LLMChatRequest{
		Model:       gs.model,
		Stream:      false,
		Temperature: common.Ptr[float64](0),
		TopP:        common.Ptr(0.7),
		Messages:    promptMessages,
	}

	content, err := gs.llmClient.Chat(ctx, req)
	if err != nil {
		return domain.BoardSummary{}, err
	}

	new.Summary = strings.TrimSpace(content)

	summary := domain.BoardSummary{
		ID:            uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Content:       new,
		Model:         gs.model,
		GeneratedAt:   now,
		SourceVersion: 1,
	}

	return summary, nil
}

//go:embed prompts/summary.yml
var summaryPrompt embed.FS

// buildPromptMessages constructs the LLM messages for the summary prompt.
func buildPromptMessages(new domain.BoardSummaryContent, previous domain.BoardSummaryContent, now time.Time) ([]domain.LLMChatMessage, error) {
	inputJSON, err := marshalSummaryContent(new)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal summary content: %w", err)
	}

	previousJSON, err := marshalSummaryContent(previous)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal previous summary content: %w", err)
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
		if msg.Role != "user" {
			continue
		}
		msg.Content = fmt.Sprintf(
			msg.Content,
			inputJSON,
			previousJSON,
		)
		messages[i] = msg
	}

	return messages, nil
}

func marshalSummaryContent(sc domain.BoardSummaryContent) (string, error) {
	summaryContentJSON, err := json.MarshalIndent(sc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal summary content: %w", err)
	}

	var buf bytes.Buffer
	err = json.Compact(&buf, summaryContentJSON)
	if err != nil {
		return "", fmt.Errorf("failed to compact summary content JSON: %w", err)
	}

	return buf.String(), nil
}

// InitGenerateBoardSummary initializes the GenerateBoardSummary use case.
type InitGenerateBoardSummary struct {
	SummaryRepo  domain.BoardSummaryRepository `resolve:""`
	TimeProvider domain.CurrentTimeProvider    `resolve:""`
	LLMClient    domain.LLMClient              `resolve:""`
	Model        string                        `config:"LLM_MODEL"`
}

// Initialize registers the GenerateBoardSummary use case implementation.
func (igbs InitGenerateBoardSummary) Initialize(ctx context.Context) (context.Context, error) {
	queue, _ := depend.Resolve[CompletedSummaryQueue]()
	depend.Register[GenerateBoardSummary](NewGenerateBoardSummaryImpl(
		igbs.SummaryRepo, igbs.TimeProvider, igbs.LLMClient, igbs.Model, queue,
	))
	return ctx, nil
}
