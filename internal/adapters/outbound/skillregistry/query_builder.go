package skillregistry

import (
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
)

// buildSelectionInputs derives the primary current-turn text and auxiliary
// recent-user context used for skill retrieval.
func buildSelectionInputs(messages []assistant.Message, maxChars int, recentLimit int) (string, string) {
	if len(messages) == 0 {
		return "", ""
	}

	currentIndex := -1
	currentInput := ""
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role != assistant.ChatRole_User {
			continue
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		currentIndex = i
		currentInput = content
		if shouldAugmentCurrentInput(content) {
			contextParts := make([]string, 0, 3)
			if assistantContext := previousAssistantMessage(messages, currentIndex, maxChars); assistantContext != "" {
				if previousUser := previousUserMessage(messages, currentIndex, maxChars); previousUser != "" {
					contextParts = append(contextParts, previousUser)
				}
				contextParts = append(contextParts, assistantContext)
			}
			if len(contextParts) > 0 {
				contextParts = append(contextParts, content)
				currentInput = strings.Join(contextParts, "\n")
			}
		}
		currentInput = truncateToLastChars(currentInput, maxChars)
		break
	}
	if currentIndex == -1 || currentInput == "" {
		return "", ""
	}

	if recentLimit <= 0 {
		return currentInput, ""
	}

	recent := make([]string, 0, recentLimit)
	for i := currentIndex - 1; i >= 0 && len(recent) < recentLimit; i-- {
		msg := messages[i]
		if msg.Role != assistant.ChatRole_User {
			continue
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		recent = append(recent, truncateToLastChars(content, maxChars))
	}

	if len(recent) == 0 {
		return currentInput, ""
	}

	for i, j := 0, len(recent)-1; i < j; i, j = i+1, j-1 {
		recent[i], recent[j] = recent[j], recent[i]
	}

	return currentInput, strings.Join(recent, "\n")
}

// shouldAugmentCurrentInput decides whether a short user reply should be
// expanded with nearby context before embedding.
func shouldAugmentCurrentInput(input string) bool {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return false
	}
	if len([]rune(trimmed)) <= 32 {
		return true
	}
	return len(strings.Fields(trimmed)) <= 4
}

// previousAssistantMessage returns the nearest preceding assistant message.
func previousAssistantMessage(messages []assistant.Message, currentIndex int, maxChars int) string {
	for i := currentIndex - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role != assistant.ChatRole_Assistant {
			continue
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		return truncateToLastChars(content, maxChars)
	}
	return ""
}

// previousUserMessage returns the nearest preceding user message.
func previousUserMessage(messages []assistant.Message, currentIndex int, maxChars int) string {
	for i := currentIndex - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role != assistant.ChatRole_User {
			continue
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		return truncateToLastChars(content, maxChars)
	}
	return ""
}

// latestUserInput returns the most recent user message content.
func latestUserInput(messages []assistant.Message, maxChars int) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role != assistant.ChatRole_User {
			continue
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		return truncateToLastChars(content, maxChars)
	}
	return ""
}

// latestUserMessage returns the most recent user message content without truncation.
func latestUserMessage(messages []assistant.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role != assistant.ChatRole_User {
			continue
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		return content
	}
	return ""
}

// parseSelectedSkillDirectives parses leading slash commands (for example
// "/web-research /todo-update ...") and returns normalized skill names.
func parseSelectedSkillDirectives(input string) []string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil
	}

	tokens := strings.Fields(trimmed)
	if len(tokens) == 0 || !strings.HasPrefix(tokens[0], "/") {
		return nil
	}

	directives := make([]string, 0, len(tokens))
	seen := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		if !strings.HasPrefix(token, "/") {
			break
		}

		name := strings.TrimPrefix(token, "/")
		name = strings.TrimRight(name, ".,;:!?")
		if !isValidSkillDirectiveName(name) {
			continue
		}

		normalized := strings.ToLower(name)
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		directives = append(directives, normalized)
	}

	return directives
}

// isValidSkillDirectiveName checks if the directive name contains only allowed characters.
func isValidSkillDirectiveName(name string) bool {
	if strings.TrimSpace(name) == "" {
		return false
	}

	for _, r := range name {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if isAlphaNum || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

// truncateToLastChars keeps only the trailing maxChars runes from the input.
func truncateToLastChars(input string, maxChars int) string {
	trimmed := strings.TrimSpace(input)
	if maxChars <= 0 {
		return ""
	}

	runes := []rune(trimmed)
	if len(runes) <= maxChars {
		return trimmed
	}

	return string(runes[len(runes)-maxChars:])
}
