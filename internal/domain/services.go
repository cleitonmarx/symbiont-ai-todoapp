package domain

import (
	"strings"
)

// GenerateAutoConversationTitle generates a conversation title based on the user's initial message.
func GenerateAutoConversationTitle(userMessage string) string {
	// Simple heuristic: use the first 5 words of the user's message as the title, or "New Conversation" if empty.
	words := strings.Fields(userMessage)
	if len(words) == 0 {
		return "New Conversation"
	}
	if len(words) <= 5 {
		return strings.Join(words, " ")
	}
	return strings.Join(words[:5], " ") + "..."
}

// DetermineConversationSummaryGenerationDecision evaluates whether unsummarized messages warrant
// generating a new conversation summary.
func DetermineConversationSummaryGenerationDecision(
	messages []ChatMessage,
	hasMore bool,
	policy ConversationSummaryGenerationPolicy,
	stateChangingTools map[string]struct{},
) ConversationSummaryGenerationDecision {
	totalTokens := sumMessagesTotalTokens(messages)
	decision := ConversationSummaryGenerationDecision{
		ShouldGenerate: false,
		Reason:         ConversationSummaryGenerationReason_None,
		MessageCount:   len(messages),
		TotalTokens:    totalTokens,
	}

	if hasStateChangingToolSuccess(messages, stateChangingTools) {
		decision.ShouldGenerate = true
		decision.Reason = ConversationSummaryGenerationReason_StateChangingToolSuccess
		return decision
	}

	if hasMore || len(messages) >= policy.TriggerMessageCount {
		decision.ShouldGenerate = true
		decision.Reason = ConversationSummaryGenerationReason_MessageCountThreshold
		return decision
	}

	if totalTokens >= policy.TriggerTokenCount {
		decision.ShouldGenerate = true
		decision.Reason = ConversationSummaryGenerationReason_TokenCountThreshold
	}

	return decision
}

// hasStateChangingToolSuccess checks if any of the messages indicate a successful execution of a state-changing tool.
func hasStateChangingToolSuccess(messages []ChatMessage, stateChangingTools map[string]struct{}) bool {
	if len(stateChangingTools) == 0 {
		return false
	}

	toolCallFunctionsByID := map[string]string{}
	for _, message := range messages {
		if message.ChatRole != ChatRole_Assistant {
			continue
		}
		for _, toolCall := range message.ToolCalls {
			toolCallFunctionsByID[toolCall.ID] = strings.ToLower(toolCall.Function)
		}
	}

	for _, message := range messages {
		if !message.IsToolCallSuccess() {
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

// sumMessagesTotalTokens calculates the total number of tokens across a slice of chat messages.
func sumMessagesTotalTokens(messages []ChatMessage) int {
	tokenCount := 0
	for _, message := range messages {
		tokenCount += message.TotalTokens
	}
	return tokenCount
}
