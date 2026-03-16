package chat

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/common"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/core"
	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"
)

//go:embed prompts/chat.yml
var chatPrompt embed.FS

// BuildSessionParams contains the inputs required to prepare a streaming turn.
type BuildSessionParams struct {
	UserMessage         string
	Model               string
	MaxActionCycles     int
	Conversation        assistant.Conversation
	ConversationCreated bool
}

// TurnSessionBuilder prepares the conversation, history, skills, and turn request.
type TurnSessionBuilder interface {
	// Build assembles all pre-turn context needed to start streaming.
	Build(ctx context.Context, params BuildSessionParams) (TurnSession, error)
}

// turnSessionBuilder prepares conversation context and request payloads for streaming.
type turnSessionBuilder struct {
	conversationSummaryRepo assistant.ConversationSummaryRepository
	chatMessageRepo         assistant.ChatMessageRepository
	timeProvider            core.CurrentTimeProvider
	skillRegistry           assistant.SkillRegistry
	actionRegistry          assistant.ActionRegistry
}

// newTurnSessionBuilder builds the default session builder for stream chat.
func newTurnSessionBuilder(
	conversationSummaryRepo assistant.ConversationSummaryRepository,
	chatMessageRepo assistant.ChatMessageRepository,
	timeProvider core.CurrentTimeProvider,
	skillRegistry assistant.SkillRegistry,
	actionRegistry assistant.ActionRegistry,
) TurnSessionBuilder {
	return turnSessionBuilder{
		conversationSummaryRepo: conversationSummaryRepo,
		chatMessageRepo:         chatMessageRepo,
		timeProvider:            timeProvider,
		skillRegistry:           skillRegistry,
		actionRegistry:          actionRegistry,
	}
}

// Build assembles the target conversation, prompt history, selected skills, and turn request.
func (b turnSessionBuilder) Build(ctx context.Context, params BuildSessionParams) (TurnSession, error) {
	messagesHistory, summaryContext, err := b.loadMessagesHistory(ctx, params.Conversation.ID)
	if err != nil {
		return nil, err
	}

	messagesHistory = append(messagesHistory, assistant.Message{
		Role:    assistant.ChatRole_User,
		Content: params.UserMessage,
	})

	skills := b.skillRegistry.ListRelevant(ctx, assistant.SkillQueryContext{
		Messages:            messagesHistory,
		ConversationSummary: summaryContext,
	})
	selectedSkills := make([]assistant.SelectedSkill, 0, len(skills))
	relevantActions := make([]assistant.ActionDefinition, 0, len(skills))
	uniqueActionNames := make(map[string]struct{})
	for _, s := range skills {
		selectedSkills = append(selectedSkills, assistant.NewSelectedSkill(s))
		for _, tool := range s.Tools {
			if action, ok := b.actionRegistry.GetDefinition(tool); ok {
				if _, exists := uniqueActionNames[action.Name]; !exists {
					relevantActions = append(relevantActions, action)
					uniqueActionNames[action.Name] = struct{}{}
				}
			}
		}
	}

	if skillsPrompt := buildSkillsPrompt(skills); skillsPrompt != "" {
		messagesHistory = append(messagesHistory, assistant.Message{
			Role:    assistant.ChatRole_System,
			Content: skillsPrompt,
		})
	}

	request := assistant.TurnRequest{
		Model:            params.Model,
		Messages:         messagesHistory,
		Stream:           true,
		Temperature:      common.Ptr(CHAT_TEMPERATURE),
		TopP:             common.Ptr(CHAT_TOP_P),
		AvailableActions: relevantActions,
	}

	return NewTurnSession(
		params.Conversation,
		params.ConversationCreated,
		params.UserMessage,
		selectedSkills,
		request,
		params.MaxActionCycles,
	), nil
}

// loadMessagesHistory combines the current system prompt with recent non-system conversation history.
func (b turnSessionBuilder) loadMessagesHistory(ctx context.Context, conversationID uuid.UUID) ([]assistant.Message, string, error) {
	systemPrompt, summaryContext, lastSummarizedMessageID, err := b.buildSystemPrompt(ctx, conversationID)
	if err != nil {
		return nil, "", err
	}

	historyOptions := make([]assistant.ListChatMessagesOption, 0, 1)
	if lastSummarizedMessageID != nil {
		historyOptions = append(historyOptions, assistant.WithChatMessagesAfterMessageID(*lastSummarizedMessageID))
	}

	history, _, err := b.chatMessageRepo.ListChatMessages(ctx, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES, historyOptions...)
	if err != nil {
		return nil, "", err
	}

	messages := make([]assistant.Message, 0, len(systemPrompt)+len(history)+1)
	messages = append(messages, systemPrompt...)

	if len(history) > 0 && history[0].ChatRole == assistant.ChatRole_Tool {
		history = history[1:]
	}

	for _, msg := range history {
		if msg.ChatRole != assistant.ChatRole_System {
			messages = append(messages, assistant.Message{
				Role:         msg.ChatRole,
				Content:      msg.Content,
				ActionCallID: msg.ActionCallID,
				ActionCalls:  msg.ActionCalls,
				ActionError:  msg.ErrorMessage,
			})
		}
	}

	return messages, summaryContext, nil
}

// buildSystemPrompt loads the base prompt template and appends the latest conversation summary context.
func (b turnSessionBuilder) buildSystemPrompt(
	ctx context.Context,
	conversationID uuid.UUID,
) ([]assistant.Message, string, *uuid.UUID, error) {
	file, err := chatPrompt.Open("prompts/chat.yml")
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to open chat prompt: %w", err)
	}
	defer file.Close() //nolint:errcheck

	messages := []assistant.Message{}
	if err := yaml.NewDecoder(file).Decode(&messages); err != nil {
		return nil, "", nil, fmt.Errorf("failed to decode summary prompt: %w", err)
	}

	for i, msg := range messages {
		if msg.Role == assistant.ChatRole_Developer || msg.Role == assistant.ChatRole_System {
			now := b.timeProvider.Now()
			messages[i].Content = fmt.Sprintf(
				msg.Content,
				now.Format(time.DateOnly),
				now.Format(time.DateOnly),
				now.AddDate(0, 0, -1).Format(time.DateOnly),
				now.AddDate(0, 0, 1).Format(time.DateOnly),
			)
		}
	}

	latestSummary, found, err := b.conversationSummaryRepo.GetConversationSummary(ctx, conversationID)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to load conversation summary: %w", err)
	}

	summaryText := "No conversation summary available."
	if found && strings.TrimSpace(latestSummary.CurrentStateSummary) != "" {
		summaryText = strings.TrimSpace(latestSummary.CurrentStateSummary)
	}
	messages = append(messages, assistant.Message{
		Role: assistant.ChatRole_System,
		Content: fmt.Sprintf(
			"Conversation compacted context:\n%s\n\nUse this as compact memory, but prioritize explicit user instructions in this turn.",
			summaryText,
		),
	})

	summaryContext := ""
	if summaryText != "No conversation summary available." {
		summaryContext = summaryText
	}

	var lastSummarizedMessageID *uuid.UUID
	if found && latestSummary.LastSummarizedMessageID != nil {
		lastSummarizedMessageID = latestSummary.LastSummarizedMessageID
	}

	return messages, summaryContext, lastSummarizedMessageID, nil
}

// buildSkillsPrompt serializes the selected skills into a compact runbook prompt for the model.
func buildSkillsPrompt(skills []assistant.SkillDefinition) string {
	if len(skills) == 0 {
		return ""
	}

	unique := func(values []string) []string {
		out := make([]string, 0, len(values))
		seen := map[string]struct{}{}
		for _, raw := range values {
			v := strings.TrimSpace(raw)
			if v == "" {
				continue
			}
			key := strings.ToLower(v)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, v)
		}
		return out
	}

	var builder strings.Builder
	builder.WriteString("Skill runbooks for this turn:\n")
	builder.WriteString("- Follow these workflows when deciding and calling tools.\n")
	builder.WriteString("- Use strict JSON for tool arguments and match tool schemas exactly.\n\n")

	for _, skill := range skills {
		name := strings.TrimSpace(skill.Name)
		if name == "" {
			continue
		}

		builder.WriteString("Skill: ")
		builder.WriteString(name)
		builder.WriteString("\n")

		if useWhen := strings.TrimSpace(skill.UseWhen); useWhen != "" {
			builder.WriteString("Use when: ")
			builder.WriteString(useWhen)
			builder.WriteString("\n")
		}
		if avoidWhen := strings.TrimSpace(skill.AvoidWhen); avoidWhen != "" {
			builder.WriteString("Avoid when: ")
			builder.WriteString(avoidWhen)
			builder.WriteString("\n")
		}

		tools := unique(skill.Tools)
		if len(tools) > 0 {
			builder.WriteString("Tools: ")
			builder.WriteString(strings.Join(tools, ", "))
			builder.WriteString("\n")
		}

		if content := strings.TrimSpace(skill.Content); content != "" {
			builder.WriteString("Workflow:\n")
			builder.WriteString(content)
			builder.WriteString("\n")
		}

		builder.WriteString("\n")
	}

	prompt := strings.TrimSpace(builder.String())
	if prompt == "" {
		return ""
	}

	return truncateToFirstChars(prompt, MAX_SKILLS_PROMPT_CHARS)
}

// compactToLastMessages returns a copy of the last maxMessages messages, or all messages when already within the limit.
func compactToLastMessages(messages []assistant.Message, maxMessages int) []assistant.Message {
	if maxMessages <= 0 || len(messages) == 0 {
		return nil
	}

	if len(messages) <= maxMessages {
		out := make([]assistant.Message, len(messages))
		copy(out, messages)
		return out
	}

	start := len(messages) - maxMessages
	out := make([]assistant.Message, maxMessages)
	copy(out, messages[start:])
	return out
}

// truncateToFirstChars trims the input and returns at most maxChars runes without splitting a rune.
func truncateToFirstChars(input string, maxChars int) string {
	trimmed := strings.TrimSpace(input)
	if maxChars <= 0 {
		return ""
	}

	runes := []rune(trimmed)
	if len(runes) <= maxChars {
		return trimmed
	}

	return string(runes[:maxChars])
}
