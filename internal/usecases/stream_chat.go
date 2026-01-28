package usecases

import (
	"context"
	"embed"
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

// StreamChat defines the interface for the StreamChat use case
type StreamChat interface {
	Execute(ctx context.Context, userMessage string, onEvent domain.LLMStreamEventCallback) error
}

// StreamChatImpl is the implementation of the StreamChat use case
type StreamChatImpl struct {
	chatMessageRepo    domain.ChatMessageRepository
	todoRepo           domain.TodoRepository
	llmClient          domain.LLMClient
	llmModel           string
	llmEmbreddingModel string
}

// NewStreamChatImpl creates a new instance of StreamChatImpl
func NewStreamChatImpl(chatMessageRepo domain.ChatMessageRepository, todoRepo domain.TodoRepository, llmClient domain.LLMClient, llmModel string, llmEmbeddingModel string) StreamChatImpl {
	return StreamChatImpl{
		chatMessageRepo:    chatMessageRepo,
		todoRepo:           todoRepo,
		llmClient:          llmClient,
		llmModel:           llmModel,
		llmEmbreddingModel: llmEmbeddingModel,
	}
}

// buildTodosInput creates the
func buildTodosInput(todos []domain.Todo) string {
	b := strings.Builder{}
	for _, t := range todos {
		b.WriteString(t.ToLLMInput() + "\n")
	}
	return b.String()
}

//go:embed prompts/chat.yml
var chatPrompt embed.FS

// buildSystemPrompt creates a system prompt with current todos context
func (sc StreamChatImpl) buildSystemPrompt(ctx context.Context, userMsg string) ([]domain.LLMChatMessage, error) {
	embed, err := sc.llmClient.Embed(ctx, sc.llmEmbreddingModel, userMsg)
	if err != nil {
		return nil, err
	}

	// Fetch all todos related to the embedding
	todos, _, err := sc.todoRepo.ListTodos(ctx, 1, 30, domain.WithEmbedding(embed))
	if err != nil {
		return nil, err
	}

	// Build todos input
	todosInput := buildTodosInput(todos)

	file, err := chatPrompt.Open("prompts/chat.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to open chat prompt: %w", err)
	}
	defer file.Close() //nolint:errcheck

	messages := []domain.LLMChatMessage{}
	err = yaml.NewDecoder(file).Decode(&messages)
	if err != nil {
		return nil, fmt.Errorf("failed to decode summary prompt: %w", err)
	}

	for i, msg := range messages {
		if msg.Role == domain.ChatRole_System {
			msg.Content = fmt.Sprintf(msg.Content,
				todosInput,
			)
			messages[i] = msg
		}
	}

	return messages, nil
}

// Execute streams a chat response and persists the conversation
func (sc StreamChatImpl) Execute(ctx context.Context, userMessage string, onEvent domain.LLMStreamEventCallback) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	// Build system prompt with todo context
	systemPrompt, err := sc.buildSystemPrompt(spanCtx, userMessage)
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	// Load prior conversation to preserve context
	history, _, err := sc.chatMessageRepo.ListChatMessages(spanCtx, 15)
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	// Build chat request: system + history (excluding old system messages) + current user turn
	messages := make([]domain.LLMChatMessage, 0, len(systemPrompt)+len(history)+1)
	messages = append(messages, systemPrompt...)

	for _, msg := range history {
		// Skip old system messages to avoid stale todo data
		if msg.ChatRole != domain.ChatRole_System {
			messages = append(messages, domain.LLMChatMessage{
				Role:    msg.ChatRole,
				Content: msg.Content,
			})
		}
	}

	messages = append(messages, domain.LLMChatMessage{
		Role:    domain.ChatRole_User,
		Content: userMessage,
	})

	req := domain.LLMChatRequest{
		Model:       sc.llmModel,
		Messages:    messages,
		Stream:      true,
		Temperature: common.Ptr(0.7),  // Controls randomness (0.0 = deterministic, 1.0 = creative)
		MaxTokens:   common.Ptr(2048), // Maximum number of tokens to generate in response
		TopP:        common.Ptr(0.9),  // Nucleus sampling (keeps top 90% probability tokens)
	}

	// Track metadata and accumulate content
	var (
		assistantMessageID uuid.UUID
		userMessageID      uuid.UUID
		finalUsage         *domain.LLMUsage
		fullContent        strings.Builder
		chatTries          = 0
		gotContent         = false
	)

	for chatTries < 3 && !gotContent {
		chatTries++
		fullContent.Reset() // Reset content on retry

		// Stream from LLM client
		err = sc.llmClient.ChatStream(spanCtx, req, func(eventType domain.LLMStreamEventType, data any) error {
			// Forward all events to the caller
			if err := onEvent(eventType, data); err != nil {
				return err
			}

			// Capture metadata from meta event
			if eventType == domain.LLMStreamEventType_Meta {
				meta := data.(domain.LLMStreamEventMeta)
				assistantMessageID = meta.AssistantMessageID
				userMessageID = meta.UserMessageID
			}

			// Accumulate content from delta events
			if eventType == domain.LLMStreamEventType_Delta {
				delta := data.(domain.LLMStreamEventDelta)
				fullContent.WriteString(delta.Text)
			}

			// Capture usage from done event
			if eventType == domain.LLMStreamEventType_Done {
				done := data.(domain.LLMStreamEventDone)
				finalUsage = done.Usage
			}

			return nil
		})

		if tracing.RecordErrorAndStatus(span, err) {
			return err
		}
		if fullContent.Len() > 0 {
			gotContent = true
		}
	}

	// If still no content after retries, return error
	if !gotContent {
		return fmt.Errorf("LLM returned empty response after %d retries", chatTries)
	}

	// Create and persist the user message
	userMsg := domain.ChatMessage{
		ID:             userMessageID,
		ConversationID: domain.GlobalConversationID,
		ChatRole:       domain.ChatRole_User,
		Content:        userMessage,
		Model:          req.Model,
		CreatedAt:      time.Now().UTC(),
	}

	if err := sc.chatMessageRepo.CreateChatMessage(spanCtx, userMsg); tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	// Create and persist the assistant message
	assistantMsg := domain.ChatMessage{
		ID:             assistantMessageID,
		ConversationID: domain.GlobalConversationID,
		ChatRole:       domain.ChatRole_Assistant,
		Content:        fullContent.String(),
		Model:          req.Model,
		CreatedAt:      time.Now().UTC(),
	}

	if finalUsage != nil {
		assistantMsg.PromptTokens = finalUsage.PromptTokens
		assistantMsg.CompletionTokens = finalUsage.CompletionTokens
	}

	if err := sc.chatMessageRepo.CreateChatMessage(spanCtx, assistantMsg); tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	return nil
}

// InitStreamChat is the initializer for the StreamChat use case
type InitStreamChat struct {
	ChatMessageRepo domain.ChatMessageRepository `resolve:""`
	TodoRepo        domain.TodoRepository        `resolve:""`
	LLMClient       domain.LLMClient             `resolve:""`
	LLMModel        string                       `config:"LLM_MODEL"`
	EmbeddingModel  string                       `config:"LLM_EMBEDDING_MODEL"`
}

// Initialize registers the StreamChat use case in the dependency container
func (i InitStreamChat) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[StreamChat](NewStreamChatImpl(i.ChatMessageRepo, i.TodoRepo, i.LLMClient, i.LLMModel, i.EmbeddingModel))
	return ctx, nil
}
