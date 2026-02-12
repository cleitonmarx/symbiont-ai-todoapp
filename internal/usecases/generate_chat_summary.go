package usecases

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/cleitonmarx/symbiont/depend"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/common"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont/examples/todoapp/internal/telemetry"
	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"
)

const (
	// Maximum number of unsummarized chat messages to inspect per summary execution.
	MAX_CHAT_SUMMARY_MESSAGES_PER_RUN = 25

	// Minimum number of unsummarized messages that triggers summary generation.
	CHAT_SUMMARY_TRIGGER_MESSAGES = 10

	// Minimum persisted tokens from unsummarized messages that triggers summary generation.
	CHAT_SUMMARY_TRIGGER_TOKENS = 2000

	// Keep chat summary generation stable and focused on state updates.
	CHAT_SUMMARY_TEMPERATURE = 0.2
	CHAT_SUMMARY_TOP_P       = 0.7
)

// Default list of tool function names that imply task state changes.
var stateChangingTools = map[string]struct{}{
	"create_todo":          {},
	"update_todo":          {},
	"update_todo_due_date": {},
	"delete_todo":          {},
}

//go:embed prompts/chat-summary.yml
var chatSummaryPrompt embed.FS

// GenerateChatSummary defines the interface for generating conversation summaries from chat events.
type GenerateChatSummary interface {
	// Execute updates conversation summary state based on one chat-message event.
	Execute(ctx context.Context, event domain.ChatMessageEvent) error
}

// GenerateChatSummaryImpl is the implementation of the GenerateChatSummary use case.
type GenerateChatSummaryImpl struct {
	ChatMessageRepo         domain.ChatMessageRepository
	ConversationSummaryRepo domain.ConversationSummaryRepository
	TimeProvider            domain.CurrentTimeProvider
	LLMClient               domain.LLMClient
	Model                   string
}

// NewGenerateChatSummaryImpl creates a new instance of GenerateChatSummaryImpl.
func NewGenerateChatSummaryImpl(
	chatMessageRepo domain.ChatMessageRepository,
	conversationSummaryRepo domain.ConversationSummaryRepository,
	timeProvider domain.CurrentTimeProvider,
	llmClient domain.LLMClient,
	model string,
) GenerateChatSummaryImpl {

	return GenerateChatSummaryImpl{
		ChatMessageRepo:         chatMessageRepo,
		ConversationSummaryRepo: conversationSummaryRepo,
		TimeProvider:            timeProvider,
		LLMClient:               llmClient,
		Model:                   model,
	}
}

// Execute updates the current conversation summary using the latest chat message event.
func (gcs GenerateChatSummaryImpl) Execute(ctx context.Context, event domain.ChatMessageEvent) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	if event.Type != domain.EventType_CHAT_MESSAGE_SENT {
		return domain.NewValidationErr("invalid event type for chat summary")
	}

	if strings.TrimSpace(event.ConversationID) == "" {
		return domain.NewValidationErr("conversation id cannot be empty")
	}

	if gcs.Model == "" {
		return domain.NewValidationErr("model cannot be empty")
	}

	currentSummary := "No current state."
	previous, found, err := gcs.ConversationSummaryRepo.GetConversationSummary(spanCtx, event.ConversationID)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	if found && strings.TrimSpace(previous.CurrentStateSummary) != "" {
		currentSummary = previous.CurrentStateSummary
	}

	messageOptions := []domain.ListChatMessagesOption{
		domain.ListChatMessagesByConversationID(event.ConversationID),
	}
	if found && previous.LastSummarizedMessageID != nil {
		messageOptions = append(messageOptions, domain.ListChatMessagesAfterMessageID(*previous.LastSummarizedMessageID))
	}

	unsummarizedMessages, hasMore, err := gcs.ChatMessageRepo.ListChatMessages(spanCtx, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, messageOptions...)
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	if len(unsummarizedMessages) == 0 {
		return nil
	}

	if !gcs.shouldGenerateSummary(unsummarizedMessages, hasMore) {
		return nil
	}

	promptMessages, err := gcs.buildPromptMessages(currentSummary, formatMessagesForSummary(unsummarizedMessages))
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	resp, err := gcs.LLMClient.Chat(spanCtx, domain.LLMChatRequest{
		Model:       gcs.Model,
		Messages:    promptMessages,
		Stream:      false,
		Temperature: common.Ptr(CHAT_SUMMARY_TEMPERATURE),
		TopP:        common.Ptr(CHAT_SUMMARY_TOP_P),
	})
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	RecordLLMTokensUsed(spanCtx, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	summaryID := uuid.New()
	if found {
		summaryID = previous.ID
	}

	lastMessage := unsummarizedMessages[len(unsummarizedMessages)-1]
	messageID := lastMessage.ID
	err = gcs.ConversationSummaryRepo.StoreConversationSummary(spanCtx, domain.ConversationSummary{
		ID:                      summaryID,
		ConversationID:          event.ConversationID,
		CurrentStateSummary:     strings.TrimSpace(resp.Content),
		LastSummarizedMessageID: &messageID,
		UpdatedAt:               gcs.TimeProvider.Now().UTC(),
	})
	if telemetry.RecordErrorAndStatus(span, err) {
		return err
	}

	return nil
}

// buildPromptMessages constructs the prompt messages for the LLM based
// on the current conversation summary and new chat messages.
func (gcs GenerateChatSummaryImpl) buildPromptMessages(currentState, newMessages string) ([]domain.LLMChatMessage, error) {
	file, err := chatSummaryPrompt.Open("prompts/chat-summary.yml")
	if err != nil {
		return nil, fmt.Errorf("failed to open chat summary prompt: %w", err)
	}
	defer file.Close() //nolint:errcheck

	messages := []domain.LLMChatMessage{}
	err = yaml.NewDecoder(file).Decode(&messages)
	if err != nil {
		return nil, fmt.Errorf("failed to decode chat summary prompt: %w", err)
	}

	for i, msg := range messages {
		if msg.Role == domain.ChatRole_System || msg.Role == domain.ChatRole_Developer {
			messages[i].Content = fmt.Sprintf(msg.Content, currentState, newMessages)
		}
	}

	return messages, nil
}

// formatMessagesForSummary formats a list of chat messages into a string representation
// suitable for LLM summarization.
func formatMessagesForSummary(messages []domain.ChatMessage) string {
	formatted := make([]string, 0, len(messages))
	for _, message := range messages {
		formatted = append(formatted, formatMessageForSummary(message))
	}
	return strings.Join(formatted, "\n")
}

// formatMessageForSummary formats a single chat message into a string representation,
// including relevant details for summary generation.
func formatMessageForSummary(message domain.ChatMessage) string {
	parts := []string{
		fmt.Sprintf("- role: %s", message.ChatRole),
		fmt.Sprintf("  state: %s", message.MessageState),
		fmt.Sprintf("  content: %s", strings.TrimSpace(message.Content)),
	}

	if message.ChatRole == domain.ChatRole_Tool {
		parts = append(parts, fmt.Sprintf("  tool_success: %t", isToolMessageSuccess(message)))
	}

	if message.ErrorMessage != nil && strings.TrimSpace(*message.ErrorMessage) != "" {
		parts = append(parts, fmt.Sprintf("  error: %s", strings.TrimSpace(*message.ErrorMessage)))
	}

	return strings.Join(parts, "\n")
}

// isToolMessageSuccess determines if a tool message indicates a successful tool execution
// based on its message state.
func isToolMessageSuccess(message domain.ChatMessage) bool {
	return message.MessageState != domain.ChatMessageState_Failed
}

func (gcs GenerateChatSummaryImpl) shouldGenerateSummary(messages []domain.ChatMessage, hasMore bool) bool {
	if gcs.hasStateChangingToolSuccess(messages) {
		return true
	}

	if hasMore || len(messages) >= CHAT_SUMMARY_TRIGGER_MESSAGES {
		return true
	}

	return sumMessagesTotalTokens(messages) >= CHAT_SUMMARY_TRIGGER_TOKENS
}

// hasStateChangingToolSuccess checks if any of the chat messages indicate a
// successful execution of a state-changing tool,
func (gcs GenerateChatSummaryImpl) hasStateChangingToolSuccess(messages []domain.ChatMessage) bool {
	toolCallFunctionsByID := map[string]string{}
	for _, message := range messages {
		if message.ChatRole != domain.ChatRole_Assistant {
			continue
		}
		for _, toolCall := range message.ToolCalls {
			toolCallFunctionsByID[toolCall.ID] = strings.ToLower(toolCall.Function)
		}
	}

	for _, message := range messages {
		if message.ChatRole != domain.ChatRole_Tool || !isToolMessageSuccess(message) || message.ToolCallID == nil {
			continue
		}
		toolFunction, found := toolCallFunctionsByID[*message.ToolCallID]
		if !found {
			continue
		}
		if _, stateChanging := stateChangingTools[toolFunction]; stateChanging {
			return true
		}
	}

	return false
}

// sumMessagesTotalTokens calculates the total number of tokens from a list of chat messages.
func sumMessagesTotalTokens(messages []domain.ChatMessage) int {
	tokenCount := 0
	for _, message := range messages {
		tokenCount += message.TotalTokens
	}
	return tokenCount
}

// InitGenerateChatSummary initializes the GenerateChatSummary use case.
type InitGenerateChatSummary struct {
	ChatMessageRepo         domain.ChatMessageRepository         `resolve:""`
	ConversationSummaryRepo domain.ConversationSummaryRepository `resolve:""`
	TimeProvider            domain.CurrentTimeProvider           `resolve:""`
	LLMClient               domain.LLMClient                     `resolve:""`
	Model                   string                               `config:"LLM_CHAT_SUMMARY_MODEL"`
}

// Initialize registers the GenerateChatSummary use case implementation.
func (i InitGenerateChatSummary) Initialize(ctx context.Context) (context.Context, error) {
	depend.Register[GenerateChatSummary](NewGenerateChatSummaryImpl(
		i.ChatMessageRepo,
		i.ConversationSummaryRepo,
		i.TimeProvider,
		i.LLMClient,
		i.Model,
	))
	return ctx, nil
}
