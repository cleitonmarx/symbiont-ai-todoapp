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
	stateChangingActions map[string]struct{},
) ConversationSummaryGenerationDecision {
	totalTokens := sumMessagesTotalTokens(messages)
	decision := ConversationSummaryGenerationDecision{
		ShouldGenerate: false,
		Reason:         ConversationSummaryGenerationReason_None,
		MessageCount:   len(messages),
		TotalTokens:    totalTokens,
	}

	if hasStateChangingActionSuccess(messages, stateChangingActions) {
		decision.ShouldGenerate = true
		decision.Reason = ConversationSummaryGenerationReason_StateChangingActionSuccess
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

// hasStateChangingActionSuccess checks if any of the messages indicate a successful execution of a state-changing action.
func hasStateChangingActionSuccess(messages []ChatMessage, stateChangingActions map[string]struct{}) bool {
	if len(stateChangingActions) == 0 {
		return false
	}

	actionCallFunctionsByID := map[string]string{}
	for _, message := range messages {
		if message.ChatRole != ChatRole_Assistant {
			continue
		}
		for _, actionCall := range message.ActionCalls {
			actionCallFunctionsByID[actionCall.ID] = strings.ToLower(actionCall.Name)
		}
	}

	for _, message := range messages {
		if !message.IsActionCallSuccess() {
			continue
		}
		actionFunction, found := actionCallFunctionsByID[*message.ActionCallID]
		if !found {
			continue
		}
		if _, stateChanging := stateChangingActions[actionFunction]; stateChanging {
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
