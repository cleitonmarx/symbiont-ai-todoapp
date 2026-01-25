package usecases

import (
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
	todoRepo     domain.TodoRepository
	timeProvider domain.CurrentTimeProvider
	llmClient    domain.LLMClient
	model        string
	queue        CompletedSummaryQueue
	// Add dependencies here if needed
}

// NewGenerateBoardSummaryImpl creates a new instance of GenerateBoardSummaryImpl.
func NewGenerateBoardSummaryImpl(
	bsr domain.BoardSummaryRepository,
	td domain.TodoRepository,
	tp domain.CurrentTimeProvider,
	c domain.LLMClient,
	m string,
	q CompletedSummaryQueue,

) GenerateBoardSummaryImpl {
	return GenerateBoardSummaryImpl{
		summaryRepo:  bsr,
		todoRepo:     td,
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

	todos, _, err := gs.todoRepo.ListTodos(
		spanCtx,
		1,
		1000,
	)
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	summary, err := gs.generateBoardSummary(spanCtx, todos)
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

func (gs GenerateBoardSummaryImpl) generateBoardSummary(ctx context.Context, todos []domain.Todo) (domain.BoardSummary, error) {
	now := gs.timeProvider.Now()
	promptMessages, err := buildPromptMessages(todos, now)
	if err != nil {
		return domain.BoardSummary{}, fmt.Errorf("failed to build prompt: %w", err)
	}

	req := domain.LLMChatRequest{
		Model:       gs.model,
		Stream:      false,
		Temperature: common.Ptr[float64](0),
		TopP:        common.Ptr(0.1),
		Messages:    promptMessages,
	}

	content, err := gs.llmClient.Chat(ctx, req)
	if err != nil {
		return domain.BoardSummary{}, err
	}

	summary, err := parseResponse(content)
	if err != nil {
		return domain.BoardSummary{}, fmt.Errorf("failed to parse LLM response: %w", err)
	}

	summary.ID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	summary.Model = gs.model
	summary.GeneratedAt = now
	summary.SourceVersion = 1

	return summary, nil
}

// parseResponse extracts the BoardSummary from the LLM response.
func parseResponse(response string) (domain.BoardSummary, error) {
	// Extract JSON from response (in case there's extra text)
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}") + 1

	if jsonStart == -1 || jsonEnd <= jsonStart {
		return domain.BoardSummary{}, fmt.Errorf("no JSON found in response: %s", response)
	}

	jsonStr := response[jsonStart:jsonEnd]

	var content domain.BoardSummaryContent
	if err := json.Unmarshal([]byte(jsonStr), &content); err != nil {
		return domain.BoardSummary{}, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return domain.BoardSummary{
		Content: content,
	}, nil
}

//go:embed prompts/summary.yml
var summaryPrompt embed.FS

// buildPromptMessages constructs the LLM messages for the summary prompt.
func buildPromptMessages(todos []domain.Todo, now time.Time) ([]domain.LLMChatMessage, error) {
	todosJSON := buildTodosJSON(todos)

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
		msg.Content = fmt.Sprintf(msg.Content,
			now.Format(time.DateOnly),
			now.AddDate(0, 0, 7).Format(time.DateOnly),
			todosJSON,
		)
		messages[i] = msg
	}

	return messages, nil
}

// InitGenerateBoardSummary initializes the GenerateBoardSummary use case.
type InitGenerateBoardSummary struct {
	SummaryRepo  domain.BoardSummaryRepository `resolve:""`
	TodoRepo     domain.TodoRepository         `resolve:""`
	TimeProvider domain.CurrentTimeProvider    `resolve:""`
	LLMClient    domain.LLMClient              `resolve:""`
	Model        string                        `config:"LLM_MODEL"`
}

// Initialize registers the GenerateBoardSummary use case implementation.
func (igbs InitGenerateBoardSummary) Initialize(ctx context.Context) (context.Context, error) {
	queue, _ := depend.Resolve[CompletedSummaryQueue]()
	depend.Register[GenerateBoardSummary](NewGenerateBoardSummaryImpl(
		igbs.SummaryRepo, igbs.TodoRepo, igbs.TimeProvider, igbs.LLMClient, igbs.Model, queue,
	))
	return ctx, nil
}
