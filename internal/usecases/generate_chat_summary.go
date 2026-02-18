package usecases

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont/depend"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.yaml.in/yaml/v3"
)

const (
	// Maximum number of unsummarized chat messages to inspect per summary execution.
	MAX_CHAT_SUMMARY_MESSAGES_PER_RUN = 25

	// Minimum number of unsummarized messages that triggers summary generation.
	CHAT_SUMMARY_TRIGGER_MESSAGES = 10

	// Minimum persisted tokens from unsummarized messages that triggers summary generation.
	CHAT_SUMMARY_TRIGGER_TOKENS = 2000

	// Maximum output tokens for the summary model response.
	CHAT_SUMMARY_MAX_TOKENS = 1024

	// Keep chat summary generation stable and focused on state updates.
	CHAT_SUMMARY_TEMPERATURE = 0.2
	CHAT_SUMMARY_TOP_P       = 0.7

	// Frequency penalty to reduce repetition in summaries, especially for longer conversations.
	CHAT_SUMMARY_FREQUENCY_PENALTY = 0.7

	// Maximum number of recent tool calls persisted in conversation summary memory.
	MAX_RECENT_TOOL_CALLS_IN_SUMMARY = 10

	// Summary field used to persist rolling tool-call history.
	SUMMARY_RECENT_TOOL_CALLS_FIELD = "recent_tool_calls"
)

// CompletedConversationSummaryChannel is a channel type for sending processed domain.ConversationSummary items.
// It is used in integration tests to verify summary generation.
type CompletedConversationSummaryChannel chan domain.ConversationSummary

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
	chatMessageRepo         domain.ChatMessageRepository
	conversationSummaryRepo domain.ConversationSummaryRepository
	timeProvider            domain.CurrentTimeProvider
	assistant               domain.Assistant
	model                   string
	completedSummaryCh      CompletedConversationSummaryChannel
}

// NewGenerateChatSummaryImpl creates a new instance of GenerateChatSummaryImpl.
func NewGenerateChatSummaryImpl(
	chatMessageRepo domain.ChatMessageRepository,
	conversationSummaryRepo domain.ConversationSummaryRepository,
	timeProvider domain.CurrentTimeProvider,
	assistant domain.Assistant,
	model string,
	q CompletedConversationSummaryChannel,
) GenerateChatSummaryImpl {
	return GenerateChatSummaryImpl{
		chatMessageRepo:         chatMessageRepo,
		conversationSummaryRepo: conversationSummaryRepo,
		timeProvider:            timeProvider,
		assistant:               assistant,
		model:                   model,
		completedSummaryCh:      q,
	}
}

// Execute updates the current conversation summary using the latest chat message event.
func (gcs GenerateChatSummaryImpl) Execute(ctx context.Context, event domain.ChatMessageEvent) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	if event.Type != domain.EventType_CHAT_MESSAGE_SENT {
		return domain.NewValidationErr("invalid event type for chat summary")
	}
	if event.ConversationID == uuid.Nil {
		return domain.NewValidationErr("conversation id cannot be empty")
	}

	currentSummary := domain.DefaultConversationStateSummary
	previous, found, err := gcs.conversationSummaryRepo.GetConversationSummary(spanCtx, event.ConversationID)
	if telemetry.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to get conversation summary: %w", err)
	}

	if found {
		currentSummary = previous.CurrentStateOrDefault()
	}

	messageOptions := []domain.ListChatMessagesOption{}
	if found && previous.LastSummarizedMessageID != nil {
		messageOptions = append(messageOptions, domain.WithChatMessagesAfterMessageID(*previous.LastSummarizedMessageID))
	}

	unsummarizedMessages, hasMore, err := gcs.chatMessageRepo.ListChatMessages(spanCtx, event.ConversationID, 1, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, messageOptions...)
	if telemetry.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to list chat messages: %w", err)
	}
	span.SetAttributes(
		attribute.Int("unsummarized_messages_count", len(unsummarizedMessages)),
	)

	if len(unsummarizedMessages) == 0 {
		return nil
	}

	if !gcs.shouldGenerateSummary(span, unsummarizedMessages, hasMore) {
		return nil
	}

	promptMessages, err := gcs.buildPromptMessages(currentSummary, formatMessagesForSummary(unsummarizedMessages))
	if telemetry.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to build prompt messages: %w", err)
	}

	resp, err := gcs.assistant.RunTurnSync(spanCtx, domain.AssistantTurnRequest{
		Model:            gcs.model,
		Messages:         promptMessages,
		Stream:           false,
		MaxTokens:        common.Ptr(CHAT_SUMMARY_MAX_TOKENS),
		Temperature:      common.Ptr(CHAT_SUMMARY_TEMPERATURE),
		TopP:             common.Ptr(CHAT_SUMMARY_TOP_P),
		FrequencyPenalty: common.Ptr(CHAT_SUMMARY_FREQUENCY_PENALTY),
	})
	if telemetry.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to generate chat summary: %w", err)
	}

	RecordLLMTokensUsed(spanCtx, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
	summaryContent := strings.TrimSpace(resp.Content)
	if summaryContent == "" {
		return nil
	}
	summaryContent = mergeRecentToolCallsIntoSummary(currentSummary, summaryContent, unsummarizedMessages)

	summaryID := uuid.New()
	if found {
		summaryID = previous.ID
	}

	lastMessage := unsummarizedMessages[len(unsummarizedMessages)-1]

	newSummary := domain.ConversationSummary{
		ID:                      summaryID,
		ConversationID:          event.ConversationID,
		CurrentStateSummary:     summaryContent,
		LastSummarizedMessageID: &lastMessage.ID,
		UpdatedAt:               gcs.timeProvider.Now().UTC(),
	}

	err = gcs.conversationSummaryRepo.StoreConversationSummary(spanCtx, newSummary)
	if telemetry.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to store conversation summary: %w", err)
	}

	if gcs.completedSummaryCh != nil {
		gcs.completedSummaryCh <- newSummary
	}

	return nil
}

// buildPromptMessages constructs the prompt messages for the LLM based
// on the current conversation summary and new chat messages.
func (gcs GenerateChatSummaryImpl) buildPromptMessages(currentState, newMessages string) ([]domain.AssistantMessage, error) {
	file, err := chatSummaryPrompt.Open("prompts/chat-summary.yml")
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck

	messages := []domain.AssistantMessage{}
	err = yaml.NewDecoder(file).Decode(&messages)
	if err != nil {
		return nil, err
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
		parts = append(parts, fmt.Sprintf("  tool_success: %t", message.IsActionCallSuccess()))
	}

	if message.ErrorMessage != nil && strings.TrimSpace(*message.ErrorMessage) != "" {
		parts = append(parts, fmt.Sprintf("  error: %s", strings.TrimSpace(*message.ErrorMessage)))
	}

	return strings.Join(parts, "\n")
}

// shouldGenerateSummary determines whether a new conversation summary should be generated
// based on the unsummarized messages and defined generation policies.
func (gcs GenerateChatSummaryImpl) shouldGenerateSummary(span trace.Span, messages []domain.ChatMessage, hasMore bool) bool {
	decision := domain.DetermineConversationSummaryGenerationDecision(
		messages,
		hasMore,
		domain.ConversationSummaryGenerationPolicy{
			TriggerMessageCount: CHAT_SUMMARY_TRIGGER_MESSAGES,
			TriggerTokenCount:   CHAT_SUMMARY_TRIGGER_TOKENS,
		},
		stateChangingTools,
	)

	switch decision.Reason {
	case domain.ConversationSummaryGenerationReason_StateChangingToolSuccess:
		span.AddEvent("Triggering summary generation due to successful state-changing tool call")
	case domain.ConversationSummaryGenerationReason_MessageCountThreshold:
		span.AddEvent(fmt.Sprintf("Triggering summary generation due to message count threshold: %d messages", decision.MessageCount))
	case domain.ConversationSummaryGenerationReason_TokenCountThreshold:
		span.AddEvent(fmt.Sprintf("Triggering summary generation due to token count threshold: %d tokens", decision.TotalTokens))
	}

	return decision.ShouldGenerate
}

// mergeRecentToolCallsIntoSummary extracts recent tool calls from the new unsummarized messages,
// merges them with any existing tool call history in the previous summary, and upserts the combined list
// back into the new summary content.
func mergeRecentToolCallsIntoSummary(previousSummary, newSummary string, messages []domain.ChatMessage) string {
	existing := parseRecentToolCallsFromSummary(previousSummary)
	latest := extractRecentToolCalls(messages)
	merged := append(existing, latest...)
	merged = keepLastNToolCalls(merged, MAX_RECENT_TOOL_CALLS_IN_SUMMARY)
	return upsertSummaryField(newSummary, SUMMARY_RECENT_TOOL_CALLS_FIELD, formatRecentToolCalls(merged))
}

// parseRecentToolCallsFromSummary looks for the recent tool calls field in the given summary content
// and parses it into a list of tool function names.
func parseRecentToolCallsFromSummary(summary string) []string {
	value, ok := findSummaryFieldValue(summary, SUMMARY_RECENT_TOOL_CALLS_FIELD)
	if !ok {
		return nil
	}
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "none") {
		return nil
	}

	parts := strings.Split(value, ";")
	toolCalls := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		toolCalls = append(toolCalls, name)
	}
	return keepLastNToolCalls(toolCalls, MAX_RECENT_TOOL_CALLS_IN_SUMMARY)
}

// extractRecentToolCalls inspects the given list of chat messages and extracts the function names of any tool calls,
// especially those that are relevant for state changes, to be included in the conversation summary memory.
func extractRecentToolCalls(messages []domain.ChatMessage) []string {
	toolCalls := make([]string, 0, len(messages))
	for _, message := range messages {
		if len(message.ActionCalls) == 0 {
			continue
		}
		for _, toolCall := range message.ActionCalls {
			functionName := strings.TrimSpace(toolCall.Name)
			if functionName == "" {
				continue
			}
			toolCalls = append(toolCalls, functionName)
		}
	}
	return toolCalls
}

// keepLastNToolCalls ensures that only the most recent N tool calls are kept in the conversation summary memory,
// to prevent unbounded growth while still retaining relevant recent tool usage history for context in future summaries.
func keepLastNToolCalls(toolCalls []string, max int) []string {
	if max <= 0 {
		return nil
	}
	if len(toolCalls) <= max {
		return toolCalls
	}
	return toolCalls[len(toolCalls)-max:]
}

// formatRecentToolCalls takes a list of tool function names and formats them into a single string representation
// suitable for inclusion in the conversation summary content.
func formatRecentToolCalls(toolCalls []string) string {
	if len(toolCalls) == 0 {
		return "none"
	}
	return strings.Join(toolCalls, "; ")
}

// findSummaryFieldValue searches the given summary content for a field with the specified name and returns its value if found.
func findSummaryFieldValue(summary, targetField string) (string, bool) {
	targetField = strings.ToLower(strings.TrimSpace(targetField))
	if targetField == "" {
		return "", false
	}

	for line := range strings.SplitSeq(summary, "\n") {
		key, value, ok := parseSummaryFieldLine(line)
		if !ok {
			continue
		}
		if key == targetField {
			return value, true
		}
	}

	return "", false
}

// upsertSummaryField takes the given summary content and upserts a field with the specified name and value.
func upsertSummaryField(summary, fieldName, fieldValue string) string {
	summary = strings.TrimSpace(summary)
	fieldName = strings.ToLower(strings.TrimSpace(fieldName))
	fieldValue = strings.TrimSpace(fieldValue)
	if summary == "" || fieldName == "" {
		return summary
	}
	if fieldValue == "" {
		fieldValue = "none"
	}

	updatedLines := make([]string, 0)
	replaced := false
	insertedAfterLastAction := false

	for line := range strings.SplitSeq(summary, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		key, _, ok := parseSummaryFieldLine(trimmed)
		if ok && key == fieldName {
			updatedLines = append(updatedLines, fmt.Sprintf("%s: %s", fieldName, fieldValue))
			replaced = true
			continue
		}

		updatedLines = append(updatedLines, trimmed)

		if !replaced && !insertedAfterLastAction && ok && key == "last_action" {
			updatedLines = append(updatedLines, fmt.Sprintf("%s: %s", fieldName, fieldValue))
			insertedAfterLastAction = true
		}
	}

	if !replaced && !insertedAfterLastAction {
		updatedLines = append(updatedLines, fmt.Sprintf("%s: %s", fieldName, fieldValue))
	}

	return strings.Join(updatedLines, "\n")
}

// parseSummaryFieldLine attempts to parse a line of summary content as a key-value pair separated by a colon.
// It returns the key, value, and a boolean indicating whether the parsing was successful.
func parseSummaryFieldLine(line string) (string, string, bool) {
	key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
	if !ok {
		return "", "", false
	}

	key = strings.ToLower(strings.TrimSpace(key))
	value = strings.TrimSpace(value)
	if key == "" {
		return "", "", false
	}

	return key, value, true
}

// InitGenerateChatSummary initializes the GenerateChatSummary use case.
type InitGenerateChatSummary struct {
	ChatMessageRepo         domain.ChatMessageRepository         `resolve:""`
	ConversationSummaryRepo domain.ConversationSummaryRepository `resolve:""`
	TimeProvider            domain.CurrentTimeProvider           `resolve:""`
	Assistant               domain.Assistant                     `resolve:""`
	Model                   string                               `config:"LLM_CHAT_SUMMARY_MODEL"`
}

// Initialize registers the GenerateChatSummary use case implementation.
func (i InitGenerateChatSummary) Initialize(ctx context.Context) (context.Context, error) {
	queue, _ := depend.Resolve[CompletedConversationSummaryChannel]()
	depend.Register[GenerateChatSummary](NewGenerateChatSummaryImpl(
		i.ChatMessageRepo,
		i.ConversationSummaryRepo,
		i.TimeProvider,
		i.Assistant,
		i.Model,
		queue,
	))
	return ctx, nil
}
