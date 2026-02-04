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

const (
	// Maximum number of chat history messages to include in the context
	MAX_CHAT_HISTORY_MESSAGES = 10

	// Maximum number of repeated tool call hits to prevent infinite loops
	MAX_REPEATED_TOOL_CALL_HIT = 5
)

//go:embed prompts/chat.yml
var chatPrompt embed.FS

// StreamChat defines the interface for the StreamChat use case
type StreamChat interface {
	Execute(ctx context.Context, userMessage string, onEvent domain.LLMStreamEventCallback) error
}

// StreamChatImpl is the implementation of the StreamChat use case
type StreamChatImpl struct {
	chatMessageRepo   domain.ChatMessageRepository
	timeProvider      domain.CurrentTimeProvider
	llmClient         domain.LLMClient
	llmToolRegistry   domain.LLMToolRegistry
	llmModel          string
	llmEmbeddingModel string
	maxToolCycles     int
}

// NewStreamChatImpl creates a new instance of StreamChatImpl
func NewStreamChatImpl(
	chatMessageRepo domain.ChatMessageRepository,
	timeProvider domain.CurrentTimeProvider,
	llmClient domain.LLMClient,
	llmToolRegistry domain.LLMToolRegistry,
	llmModel string,
	llmEmbeddingModel string,
	maxToolCycles int,
) StreamChatImpl {
	return StreamChatImpl{
		chatMessageRepo:   chatMessageRepo,
		timeProvider:      timeProvider,
		llmClient:         llmClient,
		llmToolRegistry:   llmToolRegistry,
		llmModel:          llmModel,
		llmEmbeddingModel: llmEmbeddingModel,
		maxToolCycles:     maxToolCycles,
	}
}

// Execute streams a chat response and persists the conversation
func (sc StreamChatImpl) Execute(ctx context.Context, userMessage string, onEvent domain.LLMStreamEventCallback) error {
	spanCtx, span := tracing.Start(ctx)
	defer span.End()

	// Fetch chat history and append user message
	messages, err := sc.fetchChatHistory(spanCtx)
	if tracing.RecordErrorAndStatus(span, err) {
		return err
	}
	messages = append(messages, domain.LLMChatMessage{
		Role:    domain.ChatRole_User,
		Content: userMessage,
	})

	req := domain.LLMChatRequest{
		Model:       sc.llmModel,
		Messages:    messages,
		Stream:      true,
		Temperature: common.Ptr(1.1),
		TopP:        common.Ptr(0.9),
		Tools:       sc.llmToolRegistry.List(),
	}

	var (
		assistantMsgContent strings.Builder
		chatMessages        []*domain.ChatMessage
		assistantMsgID      uuid.UUID
		tracker             = newToolCycleTracker(
			sc.maxToolCycles,
			MAX_REPEATED_TOOL_CALL_HIT,
		)
	)

	// Append user message first
	userMsg := &domain.ChatMessage{
		ConversationID: domain.GlobalConversationID,
		ChatRole:       domain.ChatRole_User,
		Content:        userMessage,
		Model:          req.Model,
		CreatedAt:      sc.timeProvider.Now().UTC(),
	}
	chatMessages = append(chatMessages, userMsg)

	for continueChatStreaming := true; continueChatStreaming; {
		continueChatStreaming = false

		err = sc.llmClient.ChatStream(spanCtx, req, func(eventType domain.LLMStreamEventType, data any) error {
			switch eventType {
			case domain.LLMStreamEventType_Meta:
				// Capture message IDs from meta event
				if assistantMsgID == uuid.Nil {
					meta := data.(domain.LLMStreamEventMeta)
					assistantMsgID = meta.AssistantMessageID
					userMsg.ID = meta.UserMessageID
					if err := onEvent(eventType, data); err != nil {
						return err
					}
				}

			case domain.LLMStreamEventType_FunctionCall:
				continueChatStreaming = true

				fc := data.(domain.LLMStreamEventFunctionCall)
				if tracker.hasExceededMaxCycles() || tracker.hasExceededMaxToolCalls(fc.Function, fc.Arguments) {
					continueChatStreaming = false
					return nil
				}

				// Append assistant message for function call
				chatMessages = append(chatMessages, &domain.ChatMessage{
					ID:             uuid.New(),
					ConversationID: domain.GlobalConversationID,
					ChatRole:       domain.ChatRole_Assistant,
					ToolCalls:      []domain.LLMStreamEventFunctionCall{fc},
					Model:          req.Model,
					CreatedAt:      sc.timeProvider.Now().UTC(),
				})

				// Process and append tool message
				if err := onEvent(
					domain.LLMStreamEventType_Delta,
					domain.LLMStreamEventDelta{
						Text: sc.llmToolRegistry.StatusMessage(fc.Function),
					},
				); err != nil {
					return err
				}

				toolMessage := sc.llmToolRegistry.Call(spanCtx, fc, req.Messages)

				chatMessages = append(chatMessages, &domain.ChatMessage{
					ID:             uuid.New(),
					ConversationID: domain.GlobalConversationID,
					ChatRole:       domain.ChatRole_Tool,
					ToolCallID:     &fc.ID,
					Content:        toolMessage.Content,
					Model:          req.Model,
					// Increment CreatedAt to ensure ordering
					CreatedAt: sc.timeProvider.Now().UTC().Add(3 * time.Millisecond),
				})

				req.Messages = append(req.Messages,
					domain.LLMChatMessage{
						Role:      domain.ChatRole_Assistant,
						ToolCalls: []domain.LLMStreamEventFunctionCall{fc},
					},
					toolMessage,
				)
			case domain.LLMStreamEventType_Delta:
				delta := data.(domain.LLMStreamEventDelta)
				assistantMsgContent.WriteString(delta.Text)
				if err := onEvent(eventType, data); err != nil {
					return err
				}
			}
			return nil
		})
		if tracing.RecordErrorAndStatus(span, err) {
			return err
		}
	}

	assistantMsg := &domain.ChatMessage{
		ID:             assistantMsgID,
		ConversationID: domain.GlobalConversationID,
		ChatRole:       domain.ChatRole_Assistant,
		Content:        assistantMsgContent.String(),
		Model:          req.Model,
		CreatedAt:      sc.timeProvider.Now().UTC(),
	}
	chatMessages = append(chatMessages, assistantMsg)
	// Append the final assistant message with the full content only if there is content
	if assistantMsg.Content == "" {
		assistantMsg.Content = "Sorry, I could not process your request. Please try again."
		if err := onEvent(domain.LLMStreamEventType_Delta,
			domain.LLMStreamEventDelta{
				Text: assistantMsg.Content + "\n",
			},
		); err != nil {
			return err
		}
	}

	// Persist all messages in order
	msgs := make([]domain.ChatMessage, len(chatMessages))
	for i, m := range chatMessages {
		msgs[i] = *m
	}
	if err := sc.chatMessageRepo.CreateChatMessages(spanCtx, msgs); tracing.RecordErrorAndStatus(span, err) {
		return err
	}

	// Send done event
	if err := onEvent(domain.LLMStreamEventType_Done, domain.LLMStreamEventDone{
		AssistantMessageID: assistantMsgID.String(),
		CompletedAt:        sc.timeProvider.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		return err
	}
	return nil
}

// buildSystemPrompt creates a system prompt with current todos context
func (sc StreamChatImpl) buildSystemPrompt() ([]domain.LLMChatMessage, error) {
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
		if msg.Role == domain.ChatRole_Developer || msg.Role == domain.ChatRole_System {
			messages[i].Content = fmt.Sprintf(
				msg.Content,
				sc.timeProvider.Now().Format(time.DateOnly),
				sc.timeProvider.Now().Unix(),
			)
		}
	}
	// Fetch current todos for context

	return messages, nil
}

// fetchChatHistory retrieves the chat history excluding old system messages
func (sc StreamChatImpl) fetchChatHistory(ctx context.Context) ([]domain.LLMChatMessage, error) {
	// Build system prompt with todo context
	systemPrompt, err := sc.buildSystemPrompt()
	if err != nil {
		return nil, err
	}

	// Load prior conversation to preserve context
	history, _, err := sc.chatMessageRepo.ListChatMessages(ctx, MAX_CHAT_HISTORY_MESSAGES)
	if err != nil {
		return nil, err
	}

	// Build chat request: system + history (excluding old system messages) + current user turn
	messages := make([]domain.LLMChatMessage, 0, len(systemPrompt)+len(history)+1)
	messages = append(messages, systemPrompt...)

	//Remove orfaned tool messages from history
	// If the first message in history is a tool message, remove it
	if len(history) > 0 {
		if history[0].ChatRole == domain.ChatRole_Tool {
			history = history[1:]
		}
	}

	// Append prior conversation history, skipping previous system messages
	for _, msg := range history {
		if msg.ChatRole != domain.ChatRole_System {
			messages = append(messages, domain.LLMChatMessage{
				Role:       msg.ChatRole,
				Content:    msg.Content,
				ToolCallID: msg.ToolCallID,
				ToolCalls:  msg.ToolCalls,
			})
		}
	}
	return messages, nil
}

// toolCycleTracker helps track repeated tool calls to prevent infinite loops
type toolCycleTracker struct {
	maxToolCycles          int
	maxRepeatedToolCallHit int
	toolCycles             int
	lastToolCallSignature  string
	repeatToolCallCount    int
}

// newToolCycleTracker creates a new toolCycleTracker
func newToolCycleTracker(maxToolCycles, maxRepeatedToolCallHit int) *toolCycleTracker {
	return &toolCycleTracker{
		maxToolCycles:          maxToolCycles,
		maxRepeatedToolCallHit: maxRepeatedToolCallHit,
	}
}

// hasExceededMaxCycles checks if the maximum number of tool cycles has been exceeded
func (t *toolCycleTracker) hasExceededMaxCycles() bool {
	t.toolCycles++
	return t.toolCycles > t.maxToolCycles
}

// hasExceededMaxToolCalls checks if the same tool call has been repeated too many times
func (t *toolCycleTracker) hasExceededMaxToolCalls(functionName, arguments string) bool {
	signature := functionName + ":" + arguments
	if signature == t.lastToolCallSignature {
		t.repeatToolCallCount++
		return t.repeatToolCallCount >= t.maxRepeatedToolCallHit
	}
	t.lastToolCallSignature = signature
	t.repeatToolCallCount = 0
	return false
}

// InitStreamChat is the initializer for the StreamChat use case
type InitStreamChat struct {
	ChatMessageRepo domain.ChatMessageRepository `resolve:""`
	TimeProvider    domain.CurrentTimeProvider   `resolve:""`
	LLMToolRegistry domain.LLMToolRegistry       `resolve:""`
	LLMClient       domain.LLMClient             `resolve:""`
	LLMModel        string                       `config:"LLM_MODEL"`
	EmbeddingModel  string                       `config:"LLM_EMBEDDING_MODEL"`
	// Maximum number of tool cycles to prevent infinite loops
	// It restricts how many times the LLM can invoke tools in a single chat session
	MaxToolCycles int `config:"LLM_MAX_TOOL_CYCLES" default:"50"`
}

// Initialize registers the StreamChat use case in the dependency container
func (i InitStreamChat) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[StreamChat](NewStreamChatImpl(
		i.ChatMessageRepo,
		i.TimeProvider,
		i.LLMClient,
		i.LLMToolRegistry,
		i.LLMModel,
		i.EmbeddingModel,
		i.MaxToolCycles,
	))
	return ctx, nil
}
