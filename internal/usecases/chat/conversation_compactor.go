package chat

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/telemetry"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/usecases/metrics"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.yaml.in/yaml/v3"
)

const (
	// CHAT_SUMMARY_MAX_TOKENS is the maximum number of output tokens for the compaction model response.
	CHAT_SUMMARY_MAX_TOKENS = 1024

	// CHAT_SUMMARY_TEMPERATURE controls generation randomness for compacted context generation.
	CHAT_SUMMARY_TEMPERATURE = 0.2
	// CHAT_SUMMARY_TOP_P controls nucleus sampling for compacted context generation.
	CHAT_SUMMARY_TOP_P = 0.7

	// CHAT_SUMMARY_FREQUENCY_PENALTY reduces repetition in generated compacted context.
	CHAT_SUMMARY_FREQUENCY_PENALTY = 0.7

	// MAX_SUMMARY_CONTENT_CHARS is the maximum character count for summarized message content.
	MAX_SUMMARY_CONTENT_CHARS = 320
	// MAX_SUMMARY_TOOL_CONTENT_CHARS is the maximum character count for summarized tool content.
	MAX_SUMMARY_TOOL_CONTENT_CHARS = 180
	// MAX_SUMMARY_ERROR_CONTENT_CHARS is the maximum character count for summarized tool error content.
	MAX_SUMMARY_ERROR_CONTENT_CHARS = 180
	// MAX_SUMMARY_ACTION_CALLS_PER_LINE is the maximum number of action calls serialized per summary line.
	MAX_SUMMARY_ACTION_CALLS_PER_LINE = 5
	// MAX_COMPACTED_CONTEXT_LINES bounds the compacted memory footprint persisted after compaction.
	MAX_COMPACTED_CONTEXT_LINES = 12
	// MAX_COMPACTED_CONTEXT_CHARS bounds the compacted memory size persisted after compaction.
	MAX_COMPACTED_CONTEXT_CHARS = 2400
)

// CompletedConversationSummaryChannel is a channel type for sending processed assistant.ConversationSummary items.
// It is used in integration tests to verify context compaction.
// type CompletedConversationSummaryChannel chan assistant.ConversationSummary

//go:embed prompts/chat-summary.yml
var chatSummaryPrompt embed.FS

// ConversationCompactor defines synchronous conversation compaction operations.
type ConversationCompactor interface {
	// EvaluateConversationCompaction returns whether the conversation should be compacted.
	EvaluateConversationCompaction(
		ctx context.Context,
		conversationID uuid.UUID,
		policy assistant.CompactionPolicy,
	) (assistant.CompactionDecision, error)
	// Compact compacts conversation memory using unsummarized messages.
	Compact(ctx context.Context, conversationID uuid.UUID) error
}

// ConversationCompactorImpl compacts unsummarized conversation messages into persisted compact memory.
type ConversationCompactorImpl struct {
	chatMessageRepo         assistant.ChatMessageRepository
	conversationSummaryRepo assistant.ConversationSummaryRepository
	timeProvider            core.CurrentTimeProvider
	assistant               assistant.Assistant
	model                   string
}

// NewConversationCompactorImpl creates a new instance of ConversationCompactorImpl.
func NewConversationCompactorImpl(
	chatMessageRepo assistant.ChatMessageRepository,
	conversationSummaryRepo assistant.ConversationSummaryRepository,
	timeProvider core.CurrentTimeProvider,
	assistant assistant.Assistant,
	model string,
) ConversationCompactorImpl {
	return ConversationCompactorImpl{
		chatMessageRepo:         chatMessageRepo,
		conversationSummaryRepo: conversationSummaryRepo,
		timeProvider:            timeProvider,
		assistant:               assistant,
		model:                   model,
	}
}

// EvaluateConversationCompaction evaluates whether unsummarized conversation messages should be compacted.
func (gcs ConversationCompactorImpl) EvaluateConversationCompaction(
	ctx context.Context,
	conversationID uuid.UUID,
	policy assistant.CompactionPolicy,
) (assistant.CompactionDecision, error) {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	if conversationID == uuid.Nil {
		err := core.NewValidationErr("conversation id cannot be empty")
		telemetry.IsErrorRecorded(span, err)
		return assistant.CompactionDecision{}, err
	}

	_, _, _, unsummarizedMessages, err := gcs.loadCompactionInput(spanCtx, conversationID)
	if telemetry.IsErrorRecorded(span, err) {
		return assistant.CompactionDecision{}, err
	}
	span.SetAttributes(
		attribute.Int("unsummarized_messages_count", len(unsummarizedMessages)),
	)

	if len(unsummarizedMessages) == 0 {
		return assistant.CompactionDecision{
			ShouldCompact: false,
			Reason:        assistant.ContextCompactionReasonNone,
			MessageCount:  0,
			TotalTokens:   0,
		}, nil
	}

	return gcs.determineCompactionDecision(span, unsummarizedMessages, policy), nil
}

// Compact generates and persists refreshed compacted context for the given conversation.
func (gcs ConversationCompactorImpl) Compact(ctx context.Context, conversationID uuid.UUID) error {
	spanCtx, span := telemetry.StartSpan(ctx)
	defer span.End()

	if conversationID == uuid.Nil {
		return core.NewValidationErr("conversation id cannot be empty")
	}

	currentSummary, previous, found, unsummarizedMessages, err := gcs.loadCompactionInput(spanCtx, conversationID)
	if telemetry.IsErrorRecorded(span, err) {
		return err
	}

	if len(unsummarizedMessages) == 0 {
		return nil
	}

	return gcs.compactConversationFromState(spanCtx, conversationID, currentSummary, previous, found, unsummarizedMessages)
}

// compactConversationFromState runs the compaction prompt against the current unsummarized window and persists the result.
func (gcs ConversationCompactorImpl) compactConversationFromState(
	spanCtx context.Context,
	conversationID uuid.UUID,
	currentSummary string,
	previous assistant.ConversationSummary,
	found bool,
	unsummarizedMessages []assistant.ChatMessage,
) error {
	span := trace.SpanFromContext(spanCtx)

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
		return fmt.Errorf("failed to compact conversation context: %w", err)
	}

	metrics.RecordLLMTokensUsed(spanCtx, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
	summaryContent := strings.TrimSpace(resp.Content)
	if summaryContent == "" {
		return nil
	}
	summaryContent = normalizeConversationSummary(currentSummary, summaryContent)
	if summaryContent == "" {
		summaryContent = normalizeConversationSummary("", currentSummary)
	}

	summaryID := uuid.New()
	if found {
		summaryID = previous.ID
	}

	lastMessage := unsummarizedMessages[len(unsummarizedMessages)-1]

	newSummary := assistant.ConversationSummary{
		ID:                      summaryID,
		ConversationID:          conversationID,
		CurrentStateSummary:     summaryContent,
		LastSummarizedMessageID: &lastMessage.ID,
		UpdatedAt:               gcs.timeProvider.Now(),
	}

	err = gcs.conversationSummaryRepo.StoreConversationSummary(spanCtx, newSummary)
	if telemetry.IsErrorRecorded(span, err) {
		return fmt.Errorf("failed to store compacted conversation context: %w", err)
	}

	return nil
}

// loadCompactionInput loads the latest compacted context and the unsummarized message slice that still needs compaction.
func (gcs ConversationCompactorImpl) loadCompactionInput(
	ctx context.Context,
	conversationID uuid.UUID,
) (
	string,
	assistant.ConversationSummary,
	bool,
	[]assistant.ChatMessage,
	error,
) {
	currentSummary := assistant.DefaultConversationStateSummary
	previous, found, err := gcs.conversationSummaryRepo.GetConversationSummary(ctx, conversationID)
	if err != nil {
		return "", assistant.ConversationSummary{}, false, nil, fmt.Errorf("failed to get conversation summary: %w", err)
	}

	if found {
		currentSummary = previous.CurrentStateOrDefault()
	}

	messageOptions := []assistant.ListChatMessagesOption{}
	if found && previous.LastSummarizedMessageID != nil {
		messageOptions = append(messageOptions, assistant.WithChatMessagesAfterMessageID(*previous.LastSummarizedMessageID))
	}

	unsummarizedMessages, _, err := gcs.chatMessageRepo.ListChatMessages(ctx, conversationID, 1, 0, messageOptions...)
	if err != nil {
		return "", assistant.ConversationSummary{}, false, nil, fmt.Errorf("failed to list chat messages: %w", err)
	}

	return currentSummary, previous, found, unsummarizedMessages, nil
}

// buildPromptMessages constructs the prompt messages for the LLM based
// on the current compacted context and new chat messages.
func (gcs ConversationCompactorImpl) buildPromptMessages(currentState, newMessages string) ([]assistant.Message, error) {
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

// formatMessagesForSummary formats a list of chat messages into a compact transcript
// representation suitable for LLM-driven context compaction.
func formatMessagesForSummary(messages []assistant.ChatMessage) string {
	formatted := make([]string, 0, len(messages))
	for _, message := range messages {
		formatted = append(formatted, formatMessageForSummary(message))
	}
	return strings.Join(formatted, "\n")
}

// formatMessageForSummary formats a single chat message into a compact transcript line.
func formatMessageForSummary(message assistant.ChatMessage) string {
	contentMaxChars := MAX_SUMMARY_CONTENT_CHARS
	if message.ChatRole == assistant.ChatRole_Tool {
		contentMaxChars = MAX_SUMMARY_TOOL_CONTENT_CHARS
	}
	content := compactSummaryText(message.Content, contentMaxChars)

	roleLabel := string(message.ChatRole)
	if message.ChatRole == assistant.ChatRole_Tool && message.ActionCallID != nil {
		roleLabel = fmt.Sprintf("tool[%s]", *message.ActionCallID)
	}

	parts := []string{fmt.Sprintf("%s: %s", roleLabel, content)}

	if message.MessageState != "" && message.MessageState != assistant.ChatMessageState_Completed {
		parts = append(parts, fmt.Sprintf("state=%s", message.MessageState))
	}

	if actionCalls := formatMessageActionCallsForSummary(message.ActionCalls); actionCalls != "" {
		parts = append(parts, fmt.Sprintf("calls=%s", actionCalls))
	}

	if message.ChatRole == assistant.ChatRole_Tool {
		parts = append(parts, fmt.Sprintf("success=%t", message.IsActionCallSuccess()))
	}

	if message.ErrorMessage != nil && strings.TrimSpace(*message.ErrorMessage) != "" {
		parts = append(parts, fmt.Sprintf("error=%s", compactSummaryText(*message.ErrorMessage, MAX_SUMMARY_ERROR_CONTENT_CHARS)))
	}

	return strings.Join(parts, " | ")
}

// normalizeConversationSummary normalizes the compacted memory returned by the model
// into a small plain-text transcript block.
func normalizeConversationSummary(previousSummary, candidateSummary string) string {
	_ = previousSummary

	candidateSummary = strings.TrimSpace(candidateSummary)
	if candidateSummary == "" {
		return ""
	}

	candidateSummary = strings.TrimPrefix(candidateSummary, "```")
	candidateSummary = strings.TrimSuffix(candidateSummary, "```")

	lines := make([]string, 0, MAX_COMPACTED_CONTEXT_LINES)
	for rawLine := range strings.SplitSeq(candidateSummary, "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		line = strings.TrimLeft(line, "-*0123456789. \t")
		line = strings.Join(strings.Fields(line), " ")
		if line == "" {
			continue
		}

		lines = append(lines, clampRunes(line, MAX_SUMMARY_CONTENT_CHARS))
		if len(lines) == MAX_COMPACTED_CONTEXT_LINES {
			break
		}
	}

	return clampRunes(strings.Join(lines, "\n"), MAX_COMPACTED_CONTEXT_CHARS)
}

// determineCompactionDecision determines whether conversation compaction should run
// based on unsummarized message/token thresholds.
func (gcs ConversationCompactorImpl) determineCompactionDecision(
	span trace.Span,
	messages []assistant.ChatMessage,
	policy assistant.CompactionPolicy,
) assistant.CompactionDecision {
	decision := assistant.DetermineContextCompactionDecision(messages, policy)

	switch decision.Reason {
	case assistant.ContextCompactionReasonTokenCountThreshold:
		span.AddEvent(fmt.Sprintf("Triggering context compaction due to token count threshold: %d tokens", decision.TotalTokens))
	}

	return decision
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
