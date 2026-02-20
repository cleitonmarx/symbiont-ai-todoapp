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
	// Inspect only a small recent window; this runs frequently.
	MAX_CHAT_MESSAGES_FOR_TITLE = 20

	// Keep generation deterministic.
	CHAT_TITLE_MAX_TOKENS  = 32
	CHAT_TITLE_TEMPERATURE = 0.2
	CHAT_TITLE_TOP_P       = 0.7

	// Heuristics for skipping title generation on very short or very long conversations,
	// to avoid low-quality titles and excessive LLM calls.
	MAX_PROMPT_MESSAGE_CHARS = 220
	MAX_PROMPT_SUMMARY_CHARS = 420
	MAX_PROMPT_TASK_TOPICS   = 3
)

//go:embed prompts/conversation-title.yml
var conversationTitlePrompt embed.FS

type CompletedConversationTitleUpdateChannel chan domain.Conversation

// GenerateConversationTitle defines the interface for generating an LLM title for auto-named conversations.
type GenerateConversationTitle interface {
	// Execute tries to generate and persist a better title for the given conversation event.
	Execute(ctx context.Context, event domain.ChatMessageEvent) error
}

// GenerateConversationTitleImpl is the implementation of GenerateConversationTitle.
type GenerateConversationTitleImpl struct {
	conversationRepo        domain.ConversationRepository
	conversationSummaryRepo domain.ConversationSummaryRepository
	chatMessageRepo         domain.ChatMessageRepository
	timeProvider            domain.CurrentTimeProvider
	assistant               domain.Assistant
	model                   string
	completedTitleCh        CompletedConversationTitleUpdateChannel
}

// NewGenerateConversationTitleImpl creates a new instance of GenerateConversationTitleImpl.
func NewGenerateConversationTitleImpl(
	conversationRepo domain.ConversationRepository,
	conversationSummaryRepo domain.ConversationSummaryRepository,
	chatMessageRepo domain.ChatMessageRepository,
	timeProvider domain.CurrentTimeProvider,
	assistant domain.Assistant,
	model string,
	q CompletedConversationTitleUpdateChannel,
) GenerateConversationTitleImpl {
	return GenerateConversationTitleImpl{
		conversationRepo:        conversationRepo,
		conversationSummaryRepo: conversationSummaryRepo,
		chatMessageRepo:         chatMessageRepo,
		timeProvider:            timeProvider,
		assistant:               assistant,
		model:                   model,
		completedTitleCh:        q,
	}
}

// Execute tries to update only auto-named conversations with an LLM-generated title.
func (gct GenerateConversationTitleImpl) Execute(ctx context.Context, event domain.ChatMessageEvent) error {
	spanCtx, span := telemetry.Start(ctx)
	defer span.End()

	if event.Type != domain.EventType_CHAT_MESSAGE_SENT {
		return domain.NewValidationErr("invalid event type for conversation title generation")
	}
	if event.ConversationID == uuid.Nil {
		return domain.NewValidationErr("conversation id cannot be empty")
	}
	if event.ChatRole != domain.ChatRole_Assistant {
		return nil
	}

	conversation, found, err := gct.conversationRepo.GetConversation(spanCtx, event.ConversationID)
	if telemetry.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to get conversation: %w", err)
	}
	if !found {
		// Conversation could have been deleted between event publish and worker processing.
		return nil
	}

	if !conversation.CanBeLLMRetitled() {
		return nil
	}

	messages, _, err := gct.chatMessageRepo.ListChatMessages(spanCtx, event.ConversationID, 1, MAX_CHAT_MESSAGES_FOR_TITLE)
	if telemetry.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to list chat messages: %w", err)
	}

	conversationSummary := "No summary available."
	summary, found, err := gct.conversationSummaryRepo.GetConversationSummary(spanCtx, event.ConversationID)
	if telemetry.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to get conversation summary: %w", err)
	}
	if found && strings.TrimSpace(summary.CurrentStateSummary) != "" {
		conversationSummary = strings.TrimSpace(summary.CurrentStateSummary)
	}

	focusedSummary := focusConversationSummaryForTitle(conversationSummary)

	promptMessages, err := gct.buildPromptMessages(conversation.Title, focusedSummary, messages)
	if telemetry.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to build title prompt: %w", err)
	}

	resp, err := gct.assistant.RunTurnSync(spanCtx, domain.AssistantTurnRequest{
		Model:       gct.model,
		Messages:    promptMessages,
		Stream:      false,
		MaxTokens:   common.Ptr(CHAT_TITLE_MAX_TOKENS),
		Temperature: common.Ptr(CHAT_TITLE_TEMPERATURE),
		TopP:        common.Ptr(CHAT_TITLE_TOP_P),
	})
	if telemetry.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to generate conversation title: %w", err)
	}

	RecordLLMTokensUsed(spanCtx, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	applyStatus := conversation.ApplyLLMGeneratedTitle(resp.Content, focusedSummary)
	if applyStatus != domain.ConversationTitleApplyStatus_Updated {
		rawTitle := strings.TrimSpace(resp.Content)
		switch applyStatus {
		case domain.ConversationTitleApplyStatus_SkippedNotGrounded:
			span.AddEvent("Generated title rejected for low grounding in summary", trace.WithAttributes(
				attribute.String("generated_title_raw", rawTitle),
				attribute.String("focused_summary", focusedSummary),
			))
		case domain.ConversationTitleApplyStatus_SkippedEmpty, domain.ConversationTitleApplyStatus_SkippedUnchanged:
			span.AddEvent("Generated title is empty or unchanged, skipping update", trace.WithAttributes(
				attribute.String("generated_title_raw", rawTitle),
				attribute.String("current_title", conversation.Title),
			))
		default:
			span.AddEvent("Generated title skipped by domain policy")
		}
		return nil
	}
	conversation.UpdatedAt = gct.timeProvider.Now()

	if err := gct.conversationRepo.UpdateConversation(spanCtx, conversation); telemetry.RecordErrorAndStatus(span, err) {
		return fmt.Errorf("failed to update conversation title: %w", err)
	}

	gct.queueTitleUpdate(conversation)

	return nil
}

// queueTitleUpdate sends the updated conversation to the channel for any post-processing after title generation.
func (gct GenerateConversationTitleImpl) queueTitleUpdate(conversation domain.Conversation) {
	if gct.completedTitleCh != nil {
		gct.completedTitleCh <- conversation
	}
}

// buildPromptMessages loads the prompt template and injects the current title and recent messages.
func (gct GenerateConversationTitleImpl) buildPromptMessages(
	currentTitle string,
	conversationSummary string,
	messages []domain.ChatMessage,
) ([]domain.AssistantMessage, error) {
	file, err := conversationTitlePrompt.Open("prompts/conversation-title.yml")
	if err != nil {
		return nil, err
	}
	defer file.Close() //nolint:errcheck

	prompt := []domain.AssistantMessage{}
	if err := yaml.NewDecoder(file).Decode(&prompt); err != nil {
		return nil, err
	}

	formattedMessages := formatMessagesForConversationTitle(messages)
	for i, msg := range prompt {
		if strings.Contains(msg.Content, "%[") {
			prompt[i].Content = fmt.Sprintf(msg.Content, currentTitle, conversationSummary, formattedMessages)
		}
	}

	return prompt, nil
}

// formatMessagesForConversationTitle prepares a concise summary of recent messages for the LLM prompt.
func formatMessagesForConversationTitle(messages []domain.ChatMessage) string {
	lines := make([]string, 0, len(messages))
	for _, message := range messages {
		if message.ChatRole != domain.ChatRole_User && message.ChatRole != domain.ChatRole_Assistant {
			continue
		}
		content := summarizeMessageForTitlePrompt(message.ChatRole, message.Content)
		if content == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", message.ChatRole, content))
	}

	if len(lines) == 0 {
		return "No messages."
	}

	return strings.Join(lines, "\n")
}

// focusConversationSummaryForTitle compresses summary memory into high-signal fields for title generation.
func focusConversationSummaryForTitle(summary string) string {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return "none"
	}

	fields := parseSummaryFields(summary)
	focused := make([]string, 0, 5)

	appendFieldIfNotNone := func(key string) {
		value := strings.TrimSpace(fields[key])
		if value == "" || strings.EqualFold(value, "none") {
			return
		}
		focused = append(focused, fmt.Sprintf("%s: %s", key, clampRunes(value, MAX_PROMPT_SUMMARY_CHARS)))
	}

	appendFieldIfNotNone("current_intent")
	appendFieldIfNotNone("user_nuances")
	appendFieldIfNotNone("active_view")

	taskTopics := extractTaskTopics(fields["tasks"], MAX_PROMPT_TASK_TOPICS)
	if len(taskTopics) > 0 {
		focused = append(focused, fmt.Sprintf("task_topics: %s", strings.Join(taskTopics, "; ")))
	}

	appendFieldIfNotNone("last_action")

	if len(focused) == 0 {
		return clampRunes(strings.Join(strings.Fields(summary), " "), MAX_PROMPT_SUMMARY_CHARS)
	}

	return strings.Join(focused, "\n")
}

// parseSummaryFields extracts key-value pairs from the summary text for easier access to important fields.
func parseSummaryFields(summary string) map[string]string {
	fields := map[string]string{}
	for line := range strings.SplitSeq(summary, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		fields[key] = value
	}
	return fields
}

// extractTaskTopics parses the tasks field to extract concise topics for the title generation prompt.
func extractTaskTopics(tasksField string, max int) []string {
	tasksField = strings.TrimSpace(tasksField)
	if tasksField == "" || strings.EqualFold(tasksField, "none") || max <= 0 {
		return nil
	}

	topics := make([]string, 0, max)
	seen := map[string]struct{}{}
	for _, rawTask := range strings.Split(tasksField, ";") {
		task := strings.TrimSpace(rawTask)
		if task == "" {
			continue
		}

		parts := strings.Split(task, "|")
		title := strings.TrimSpace(parts[0])
		if strings.HasPrefix(title, "#") && len(parts) > 1 {
			title = strings.TrimSpace(parts[1])
		}
		title = strings.Join(strings.Fields(title), " ")
		if title == "" {
			continue
		}

		key := strings.ToLower(title)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		topics = append(topics, clampRunes(title, MAX_PROMPT_MESSAGE_CHARS))
		if len(topics) >= max {
			break
		}
	}

	return topics
}

// clampRunes safely truncates a string to a maximum number of runes,
// ensuring we don't cut off in the middle of a multi-byte character.
func clampRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= max {
		return string(runes)
	}
	return strings.TrimSpace(string(runes[:max]))
}

// summarizeMessageForTitlePrompt cleans and condenses a message for use in the title generation prompt,
// applying different heuristics for user vs assistant messages.
func summarizeMessageForTitlePrompt(role domain.ChatRole, content string) string {
	text := strings.ReplaceAll(content, "\r\n", "\n")
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	lines := strings.Split(text, "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.ReplaceAll(line, "**", "")
		line = strings.ReplaceAll(line, "__", "")
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		cleaned = append(cleaned, line)
	}
	if len(cleaned) == 0 {
		return ""
	}

	if role == domain.ChatRole_Assistant {
		// Keep only the assistant's lead sentence, drop long bullet/task dumps.
		for _, line := range cleaned {
			if strings.HasPrefix(line, "-") ||
				strings.HasPrefix(line, "*") ||
				strings.HasPrefix(line, "•") ||
				strings.HasPrefix(line, "1.") ||
				strings.HasPrefix(line, "2.") {
				continue
			}
			candidate := strings.TrimLeft(line, "-*•0123456789. ")
			candidate = strings.TrimSpace(candidate)
			if candidate == "" {
				continue
			}
			if len([]rune(candidate)) > MAX_PROMPT_MESSAGE_CHARS {
				candidate = clampRunes(candidate, MAX_PROMPT_MESSAGE_CHARS)
			}
			return strings.Join(strings.Fields(candidate), " ")
		}
		return ""
	}

	candidate := cleaned[0]
	if len([]rune(candidate)) > MAX_PROMPT_MESSAGE_CHARS {
		candidate = clampRunes(candidate, MAX_PROMPT_MESSAGE_CHARS)
	}
	return strings.Join(strings.Fields(candidate), " ")
}

// InitGenerateConversationTitle initializes GenerateConversationTitle.
type InitGenerateConversationTitle struct {
	ConversationRepo        domain.ConversationRepository        `resolve:""`
	ConversationSummaryRepo domain.ConversationSummaryRepository `resolve:""`
	ChatMessageRepo         domain.ChatMessageRepository         `resolve:""`
	TimeProvider            domain.CurrentTimeProvider           `resolve:""`
	Assistant               domain.Assistant                     `resolve:""`
	Model                   string                               `config:"LLM_CHAT_TITLE_MODEL"`
}

// Initialize registers GenerateConversationTitle in the dependency container.
func (i InitGenerateConversationTitle) Initialize(ctx context.Context) (context.Context, error) {
	queue, _ := depend.Resolve[CompletedConversationTitleUpdateChannel]()
	depend.Register[GenerateConversationTitle](NewGenerateConversationTitleImpl(
		i.ConversationRepo,
		i.ConversationSummaryRepo,
		i.ChatMessageRepo,
		i.TimeProvider,
		i.Assistant,
		i.Model,
		queue,
	))
	return ctx, nil
}
