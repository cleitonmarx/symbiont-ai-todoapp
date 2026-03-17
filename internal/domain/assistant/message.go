package assistant

// Message represents a message exchanged during assistant turns.
type Message struct {
	Role         ChatRole
	Content      string
	ActionCallID *string
	ActionCalls  []ActionCall
	ActionError  *string
}

// IsActionCallSuccess returns true when this message is a successful action result.
func (m Message) IsActionCallSuccess() bool {
	return m.Role == ChatRole_Tool &&
		m.ActionCallID != nil &&
		m.ActionError == nil
}
