package chat

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"github.com/google/uuid"
	"go.yaml.in/yaml/v3"
)

//go:embed prompts/chat.yml
var chatPrompt embed.FS

// buildSystemPrompt loads the base prompt template and appends the latest conversation summary context.
func (sc StreamChatImpl) buildSystemPrompt(ctx context.Context, conversationID uuid.UUID) ([]assistant.Message, string, error) {
	file, err := chatPrompt.Open("prompts/chat.yml")
	if err != nil {
		return nil, "", fmt.Errorf("failed to open chat prompt: %w", err)
	}
	defer file.Close() //nolint:errcheck

	messages := []assistant.Message{}
	err = yaml.NewDecoder(file).Decode(&messages)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode summary prompt: %w", err)
	}
	for i, msg := range messages {
		if msg.Role == assistant.ChatRole_Developer || msg.Role == assistant.ChatRole_System {
			now := sc.timeProvider.Now()
			messages[i].Content = fmt.Sprintf(
				msg.Content,
				now.Format(time.DateOnly),
				now.Format(time.DateOnly),
				now.AddDate(0, 0, -1).Format(time.DateOnly),
				now.AddDate(0, 0, 1).Format(time.DateOnly),
			)
		}
	}

	latestSummary, found, err := sc.conversationSummaryRepo.GetConversationSummary(ctx, conversationID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load conversation summary: %w", err)
	}

	summaryText := "No conversation summary available."
	if found && strings.TrimSpace(latestSummary.CurrentStateSummary) != "" {
		summaryText = strings.TrimSpace(latestSummary.CurrentStateSummary)
	}
	messages = append(messages, assistant.Message{
		Role: assistant.ChatRole_System,
		Content: fmt.Sprintf(
			"Conversation summary context:\n%s\n\nUse this as compact memory, but prioritize explicit user instructions in this turn.",
			summaryText,
		),
	})

	summaryContext := ""
	if summaryText != "No conversation summary available." {
		summaryContext = summaryText
	}

	return messages, summaryContext, nil
}

// fetchChatHistory combines the current system prompt with recent non-system conversation history.
func (sc StreamChatImpl) fetchChatHistory(ctx context.Context, conversationID uuid.UUID) ([]assistant.Message, string, error) {
	systemPrompt, summaryContext, err := sc.buildSystemPrompt(ctx, conversationID)
	if err != nil {
		return nil, "", err
	}

	history, _, err := sc.chatMessageRepo.ListChatMessages(ctx, conversationID, 1, MAX_CHAT_HISTORY_MESSAGES)
	if err != nil {
		return nil, "", err
	}

	messages := make([]assistant.Message, 0, len(systemPrompt)+len(history)+1)
	messages = append(messages, systemPrompt...)

	if len(history) > 0 {
		if history[0].ChatRole == assistant.ChatRole_Tool {
			history = history[1:]
		}
	}

	for _, msg := range history {
		if msg.ChatRole != assistant.ChatRole_System {
			messages = append(messages, assistant.Message{
				Role:         msg.ChatRole,
				Content:      msg.Content,
				ActionCallID: msg.ActionCallID,
				ActionCalls:  msg.ActionCalls,
			})
		}
	}
	return messages, summaryContext, nil
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

	var b strings.Builder
	b.WriteString("Skill runbooks for this turn:\n")
	b.WriteString("- Follow these workflows when deciding and calling tools.\n")
	b.WriteString("- Use strict JSON for tool arguments and match tool schemas exactly.\n\n")

	for _, skill := range skills {
		name := strings.TrimSpace(skill.Name)
		if name == "" {
			continue
		}

		b.WriteString("Skill: ")
		b.WriteString(name)
		b.WriteString("\n")

		if useWhen := strings.TrimSpace(skill.UseWhen); useWhen != "" {
			b.WriteString("Use when: ")
			b.WriteString(useWhen)
			b.WriteString("\n")
		}
		if avoidWhen := strings.TrimSpace(skill.AvoidWhen); avoidWhen != "" {
			b.WriteString("Avoid when: ")
			b.WriteString(avoidWhen)
			b.WriteString("\n")
		}

		tools := unique(skill.Tools)
		if len(tools) > 0 {
			b.WriteString("Tools: ")
			b.WriteString(strings.Join(tools, ", "))
			b.WriteString("\n")
		}

		if content := strings.TrimSpace(skill.Content); content != "" {
			b.WriteString("Workflow:\n")
			b.WriteString(content)
			b.WriteString("\n")
		}

		b.WriteString("\n")
	}

	prompt := strings.TrimSpace(b.String())
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
