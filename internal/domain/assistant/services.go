package assistant

import "strings"

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

// DetermineContextCompactionDecision evaluates whether unsummarized messages warrant
// generating a compacted conversation summary.
func DetermineContextCompactionDecision(
	messages []ChatMessage,
	policy CompactionPolicy,
) CompactionDecision {
	totalTokens := estimateMessagesContextTokens(messages)
	decision := CompactionDecision{
		ShouldCompact: false,
		Reason:        ContextCompactionReasonNone,
		MessageCount:  len(messages),
		TotalTokens:   totalTokens,
	}

	if totalTokens >= policy.TriggerTokenCount {
		decision.ShouldCompact = true
		decision.Reason = ContextCompactionReasonTokenCountThreshold
	}

	return decision
}

// estimateMessagesContextTokens approximates the active context size represented
// by the persisted chat messages, instead of using model usage/billing tokens.
func estimateMessagesContextTokens(messages []ChatMessage) int {
	tokenCount := 0
	for _, message := range messages {
		tokenCount += message.ContextTokensEstimate
	}
	return tokenCount
}
