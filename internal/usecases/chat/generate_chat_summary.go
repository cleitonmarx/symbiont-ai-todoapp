package chat

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/outbox"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/metrics"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.yaml.in/yaml/v3"
)

const (
	// MAX_CHAT_SUMMARY_MESSAGES_PER_RUN is the maximum number of unsummarized chat messages
	// inspected in one summary execution.
	MAX_CHAT_SUMMARY_MESSAGES_PER_RUN = 25

	// CHAT_SUMMARY_TRIGGER_MESSAGES is the minimum unsummarized message count required to trigger summary generation.
	CHAT_SUMMARY_TRIGGER_MESSAGES = 6

	// CHAT_SUMMARY_TRIGGER_TOKENS is the minimum unsummarized token count required to trigger summary generation.
	CHAT_SUMMARY_TRIGGER_TOKENS = 900

	// CHAT_SUMMARY_MAX_TOKENS is the maximum number of output tokens for the summary model response.
	CHAT_SUMMARY_MAX_TOKENS = 1024

	// CHAT_SUMMARY_TEMPERATURE controls generation randomness for chat summaries.
	CHAT_SUMMARY_TEMPERATURE = 0.2
	// CHAT_SUMMARY_TOP_P controls nucleus sampling for chat summaries.
	CHAT_SUMMARY_TOP_P = 0.7

	// CHAT_SUMMARY_FREQUENCY_PENALTY reduces repetition in generated summaries.
	CHAT_SUMMARY_FREQUENCY_PENALTY = 0.7

	// MAX_RECENT_ACTION_CALLS_IN_SUMMARY limits stored recent action calls in summary memory.
	MAX_RECENT_ACTION_CALLS_IN_SUMMARY = 5

	// SUMMARY_RECENT_ACTION_CALLS_FIELD is the summary field key for rolling action-call history.
	SUMMARY_RECENT_ACTION_CALLS_FIELD = "recent_action_calls"

	// SUMMARY_OPEN_LOOPS_FIELD is the summary field key for unresolved corrections and follow-ups.
	SUMMARY_OPEN_LOOPS_FIELD = "open_loops"

	// DEFAULT_SUMMARY_FIELD_VALUE is the default fallback value for missing summary fields.
	DEFAULT_SUMMARY_FIELD_VALUE = "none"
	// DEFAULT_SUMMARY_OUTPUT_FORMAT is the default output format for summaries.
	DEFAULT_SUMMARY_OUTPUT_FORMAT = "concise text"
	// MAX_SUMMARY_CONTENT_CHARS is the maximum character count for summarized message content.
	MAX_SUMMARY_CONTENT_CHARS = 320
	// MAX_SUMMARY_TOOL_CONTENT_CHARS is the maximum character count for summarized tool content.
	MAX_SUMMARY_TOOL_CONTENT_CHARS = 180
	// MAX_SUMMARY_ERROR_CONTENT_CHARS is the maximum character count for summarized tool error content.
	MAX_SUMMARY_ERROR_CONTENT_CHARS = 180
	// MAX_SUMMARY_ACTION_CALLS_PER_LINE is the maximum number of action calls serialized per summary line.
	MAX_SUMMARY_ACTION_CALLS_PER_LINE = 5
)

// CompletedConversationSummaryChannel is a channel type for sending processed assistant.ConversationSummary items.
// It is used in integration tests to verify summary generation.
type CompletedConversationSummaryChannel chan assistant.ConversationSummary

// Default list of action function names that imply task state changes.
var stateChangingActions = map[string]struct{}{
	"create_todo":          {},
	"update_todo":          {},
	"update_todo_due_date": {},
	"delete_todo":          {},
}

var summaryOrderedFields = []string{
	"current_intent",
	"active_view",
	"user_nuances",
	"tasks",
	"last_action",
	SUMMARY_RECENT_ACTION_CALLS_FIELD,
	SUMMARY_OPEN_LOOPS_FIELD,
	"output_format",
}

//go:embed prompts/chat-summary.yml
var chatSummaryPrompt embed.FS

// GenerateChatSummary defines the interface for generating conversation summaries from chat events.
type GenerateChatSummary interface {
	// Execute updates conversation summary state based on one chat-message event.
	Execute(ctx context.Context, event outbox.ChatMessageEvent) error
}

// GenerateChatSummaryImpl is the implementation of the GenerateChatSummary use case.
type GenerateChatSummaryImpl struct {
	chatMessageRepo         assistant.ChatMessageRepository
	conversationSummaryRepo assistant.ConversationSummaryRepository
	timeProvider            core.CurrentTimeProvider
	assistant               assistant.Assistant
	model                   string
	completedSummaryCh      CompletedConversationSummaryChannel
}

// NewGenerateChatSummaryImpl creates a new instance of GenerateChatSummaryImpl.
func NewGenerateChatSummaryImpl(
	chatMessageRepo assistant.ChatMessageRepository,
	conversationSummaryRepo assistant.ConversationSummaryRepository,
	timeProvider core.CurrentTimeProvider,
	assistant assistant.Assistant,
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
func (gcs GenerateChatSummaryImpl) Execute(ctx context.Context, event outbox.ChatMessageEvent) error {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	if event.Type != outbox.EventType_CHAT_MESSAGE_SENT {
		return core.NewValidationErr("invalid event type for chat summary")
	}
	if event.ConversationID == uuid.Nil {
		return core.NewValidationErr("conversation id cannot be empty")
	}

	currentSummary := assistant.DefaultConversationStateSummary
	previous, found, err := gcs.conversationSummaryRepo.GetConversationSummary(spanCtx, event.ConversationID)
	if telemetry.IsErrorRecorded(span, err) {
		return fmt.Errorf("failed to get conversation summary: %w", err)
	}

	if found {
		currentSummary = previous.CurrentStateOrDefault()
	}

	messageOptions := []assistant.ListChatMessagesOption{}
	if found && previous.LastSummarizedMessageID != nil {
		messageOptions = append(messageOptions, assistant.WithChatMessagesAfterMessageID(*previous.LastSummarizedMessageID))
	}

	unsummarizedMessages, hasMore, err := gcs.chatMessageRepo.ListChatMessages(spanCtx, event.ConversationID, 1, MAX_CHAT_SUMMARY_MESSAGES_PER_RUN, messageOptions...)
	if telemetry.IsErrorRecorded(span, err) {
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
	if telemetry.IsErrorRecorded(span, err) {
		return fmt.Errorf("failed to build prompt messages: %w", err)
	}

	resp, err := gcs.assistant.RunTurnSync(spanCtx, assistant.TurnRequest{
		Model:            gcs.model,
		Messages:         promptMessages,
		Stream:           false,
		MaxTokens:        common.Ptr(CHAT_SUMMARY_MAX_TOKENS),
		Temperature:      common.Ptr(CHAT_SUMMARY_TEMPERATURE),
		TopP:             common.Ptr(CHAT_SUMMARY_TOP_P),
		FrequencyPenalty: common.Ptr(CHAT_SUMMARY_FREQUENCY_PENALTY),
	})
	if telemetry.IsErrorRecorded(span, err) {
		return fmt.Errorf("failed to generate chat summary: %w", err)
	}

	metrics.RecordLLMTokensUsed(spanCtx, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
	summaryContent := strings.TrimSpace(resp.Content)
	if summaryContent == "" {
		return nil
	}
	summaryContent = normalizeConversationSummary(currentSummary, summaryContent)
	summaryContent = mergeRecentActionCallsIntoSummary(currentSummary, summaryContent, unsummarizedMessages)
	summaryContent = normalizeConversationSummary(currentSummary, summaryContent)

	summaryID := uuid.New()
	if found {
		summaryID = previous.ID
	}

	lastMessage := unsummarizedMessages[len(unsummarizedMessages)-1]

	newSummary := assistant.ConversationSummary{
		ID:                      summaryID,
		ConversationID:          event.ConversationID,
		CurrentStateSummary:     summaryContent,
		LastSummarizedMessageID: &lastMessage.ID,
		UpdatedAt:               gcs.timeProvider.Now(),
	}

	err = gcs.conversationSummaryRepo.StoreConversationSummary(spanCtx, newSummary)
	if telemetry.IsErrorRecorded(span, err) {
		return fmt.Errorf("failed to store conversation summary: %w", err)
	}

	if gcs.completedSummaryCh != nil {
		select {
		case gcs.completedSummaryCh <- newSummary:
		case <-ctx.Done():
		}
	}

	return nil
}

// buildPromptMessages constructs the prompt messages for the LLM based
// on the current conversation summary and new chat messages.
func (gcs GenerateChatSummaryImpl) buildPromptMessages(currentState, newMessages string) ([]assistant.Message, error) {
	file, err := chatSummaryPrompt.Open("prompts/chat-summary.yml")
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck

	messages := []assistant.Message{}
	err = yaml.NewDecoder(file).Decode(&messages)
	if err != nil {
		return nil, err
	}

	for i, msg := range messages {
		if msg.Role == assistant.ChatRole_System || msg.Role == assistant.ChatRole_Developer {
			messages[i].Content = fmt.Sprintf(msg.Content, currentState, newMessages)
		}
	}

	return messages, nil
}

// formatMessagesForSummary formats a list of chat messages into a string representation
// suitable for LLM summarization.
func formatMessagesForSummary(messages []assistant.ChatMessage) string {
	formatted := make([]string, 0, len(messages))
	for _, message := range messages {
		formatted = append(formatted, formatMessageForSummary(message))
	}
	return strings.Join(formatted, "\n")
}

// formatMessageForSummary formats a single chat message into a string representation,
// including relevant details for summary generation.
func formatMessageForSummary(message assistant.ChatMessage) string {
	contentMaxChars := MAX_SUMMARY_CONTENT_CHARS
	if message.ChatRole == assistant.ChatRole_Tool {
		contentMaxChars = MAX_SUMMARY_TOOL_CONTENT_CHARS
	}
	content := compactSummaryText(message.Content, contentMaxChars)

	parts := []string{
		fmt.Sprintf("- role: %s", message.ChatRole),
		fmt.Sprintf("  state: %s", message.MessageState),
		fmt.Sprintf("  content: %s", content),
	}

	if actionCalls := formatMessageActionCallsForSummary(message.ActionCalls); actionCalls != "" {
		parts = append(parts, fmt.Sprintf("  action_calls: %s", actionCalls))
	}

	if message.ChatRole == assistant.ChatRole_Tool {
		parts = append(parts, fmt.Sprintf("  action_success: %t", message.IsActionCallSuccess()))
	}

	if message.ErrorMessage != nil && strings.TrimSpace(*message.ErrorMessage) != "" {
		parts = append(parts, fmt.Sprintf("  error: %s", compactSummaryText(*message.ErrorMessage, MAX_SUMMARY_ERROR_CONTENT_CHARS)))
	}

	return strings.Join(parts, "\n")
}

// normalizeConversationSummary repairs and normalizes summary content into a stable
// compact schema so malformed or partial model outputs do not erase durable memory.
func normalizeConversationSummary(previousSummary, candidateSummary string) string {
	previousFields := parseConversationSummaryFields(previousSummary)
	candidateFields := parseConversationSummaryFields(candidateSummary)

	lines := make([]string, 0, len(summaryOrderedFields))
	for _, field := range summaryOrderedFields {
		value := strings.TrimSpace(candidateFields[field])
		if value == "" {
			value = strings.TrimSpace(previousFields[field])
		}
		if value == "" {
			value = defaultSummaryFieldValue(field)
		}
		lines = append(lines, fmt.Sprintf("%s: %s", field, value))
	}
	return strings.Join(lines, "\n")
}

// parseConversationSummaryFields parses summary field lines into a lower-cased key-value map.
func parseConversationSummaryFields(summary string) map[string]string {
	fields := make(map[string]string)
	for line := range strings.SplitSeq(summary, "\n") {
		key, value, ok := parseSummaryFieldLine(line)
		if !ok {
			continue
		}
		fields[key] = value
	}
	return fields
}

// defaultSummaryFieldValue provides default values for summary fields when they are missing or empty.
func defaultSummaryFieldValue(field string) string {
	switch field {
	case "output_format":
		return DEFAULT_SUMMARY_OUTPUT_FORMAT
	default:
		return DEFAULT_SUMMARY_FIELD_VALUE
	}
}

// shouldGenerateSummary determines whether a new conversation summary should be generated
// based on the unsummarized messages and defined generation policies.
func (gcs GenerateChatSummaryImpl) shouldGenerateSummary(span trace.Span, messages []assistant.ChatMessage, hasMore bool) bool {
	decision := assistant.DetermineConversationSummaryGenerationDecision(
		messages,
		hasMore,
		assistant.ConversationSummaryGenerationPolicy{
			TriggerMessageCount: CHAT_SUMMARY_TRIGGER_MESSAGES,
			TriggerTokenCount:   CHAT_SUMMARY_TRIGGER_TOKENS,
		},
		stateChangingActions,
	)

	switch decision.Reason {
	case assistant.ConversationSummaryGenerationReason_StateChangingActionSuccess:
		span.AddEvent("Triggering summary generation due to successful state-changing action call")
	case assistant.ConversationSummaryGenerationReason_MessageCountThreshold:
		span.AddEvent(fmt.Sprintf("Triggering summary generation due to message count threshold: %d messages", decision.MessageCount))
	case assistant.ConversationSummaryGenerationReason_TokenCountThreshold:
		span.AddEvent(fmt.Sprintf("Triggering summary generation due to token count threshold: %d tokens", decision.TotalTokens))
	}

	return decision.ShouldGenerate
}

// mergeRecentActionCallsIntoSummary extracts recent action calls from the new unsummarized messages,
// merges them with any existing action call history in the previous summary, and upserts the combined list
// back into the new summary content.
func mergeRecentActionCallsIntoSummary(previousSummary, newSummary string, messages []assistant.ChatMessage) string {
	existing := parseRecentActionCallsFromSummary(previousSummary)
	latest := extractRecentActionCalls(messages)
	merged := append(existing, latest...)
	merged = keepLastNActionCalls(merged, MAX_RECENT_ACTION_CALLS_IN_SUMMARY)
	return upsertSummaryField(newSummary, SUMMARY_RECENT_ACTION_CALLS_FIELD, formatRecentActionCalls(merged))
}

// parseRecentActionCallsFromSummary looks for the recent action calls field in the given summary content
// and parses it into a list of action function names.
func parseRecentActionCallsFromSummary(summary string) []string {
	value, ok := findSummaryFieldValue(summary, SUMMARY_RECENT_ACTION_CALLS_FIELD)
	if !ok {
		return nil
	}
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, "none") {
		return nil
	}

	parts := strings.Split(value, ";")
	actionCalls := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		actionCalls = append(actionCalls, name)
	}
	return keepLastNActionCalls(actionCalls, MAX_RECENT_ACTION_CALLS_IN_SUMMARY)
}

// extractRecentActionCalls inspects the given list of chat messages and extracts the function names of any action calls,
// especially those that are relevant for state changes, to be included in the conversation summary memory.
func extractRecentActionCalls(messages []assistant.ChatMessage) []string {
	actionCalls := make([]string, 0, len(messages))
	for _, message := range messages {
		if len(message.ActionCalls) == 0 {
			continue
		}
		for _, actionCall := range message.ActionCalls {
			functionName := strings.TrimSpace(actionCall.Name)
			if functionName == "" {
				continue
			}
			actionCalls = append(actionCalls, functionName)
		}
	}
	return actionCalls
}

// keepLastNActionCalls ensures that only the most recent N action calls are kept in the conversation summary memory,
// to prevent unbounded growth while still retaining relevant recent action usage history for context in future summaries.
func keepLastNActionCalls(actionCalls []string, max int) []string {
	if max <= 0 {
		return nil
	}
	if len(actionCalls) <= max {
		return actionCalls
	}
	return actionCalls[len(actionCalls)-max:]
}

// formatRecentActionCalls takes a list of action function names and formats them into a single string representation
// suitable for inclusion in the conversation summary content.
func formatRecentActionCalls(actionCalls []string) string {
	if len(actionCalls) == 0 {
		return "none"
	}
	return strings.Join(actionCalls, "; ")
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
	fieldLine := fmt.Sprintf("%s: %s", fieldName, fieldValue)

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
			// Keep exactly one instance of the target field to avoid duplicates.
			if !replaced {
				updatedLines = append(updatedLines, fieldLine)
				replaced = true
			}
			continue
		}

		updatedLines = append(updatedLines, trimmed)

		if !replaced && !insertedAfterLastAction && ok && key == "last_action" {
			updatedLines = append(updatedLines, fieldLine)
			insertedAfterLastAction = true
			replaced = true
		}
	}

	if !replaced && !insertedAfterLastAction {
		updatedLines = append(updatedLines, fieldLine)
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

// compactSummaryText trims, collapses whitespace, and truncates content
// so summarization input remains compact and stable.
func compactSummaryText(text string, maxChars int) string {
	normalized := common.NormalizeWhitespace(text)
	if normalized == "" {
		return "none"
	}
	if maxChars <= 0 {
		return normalized
	}
	runes := []rune(normalized)
	if len(runes) <= maxChars {
		return normalized
	}
	return string(runes[:maxChars]) + "..."
}

// formatMessageActionCallsForSummary formats the action calls of a chat message into a compact string
// representation for summary input.
func formatMessageActionCallsForSummary(actionCalls []assistant.ActionCall) string {
	if len(actionCalls) == 0 {
		return ""
	}

	names := make([]string, 0, min(len(actionCalls), MAX_SUMMARY_ACTION_CALLS_PER_LINE))
	for _, actionCall := range actionCalls {
		name := strings.TrimSpace(actionCall.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
		if len(names) >= MAX_SUMMARY_ACTION_CALLS_PER_LINE {
			break
		}
	}

	if len(names) == 0 {
		return ""
	}
	return strings.Join(names, "; ")
}
