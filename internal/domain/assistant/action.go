package assistant

import (
	"context"
	"time"
)

// ActionCall contains one action invocation requested by the assistant.
type ActionCall struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Input string `json:"input"`
	Text  string `json:"text"`
}

// ActionDefinition describes one action that can be used by the assistant.
type ActionDefinition struct {
	Name        string
	Description string
	Input       ActionInput
	Approval    ActionApproval
}

// ActionApproval holds human approval policy metadata for one action.
type ActionApproval struct {
	Required bool
	// Title is a short approval title for UI prompts.
	Title string
	// Description explains what the action will do and why approval is needed.
	Description string
	// PreviewFields are JSON paths (e.g. todos[].title) used by the UI to render a readable approval preview.
	PreviewFields []string
	// Timeout controls how long the system should wait for a decision.
	Timeout time.Duration
}

// RequiresApproval returns true when the action policy requires explicit human approval.
func (d ActionDefinition) RequiresApproval() bool {
	return d.Approval.Required
}

// ActionField represents one action input field.
type ActionField struct {
	Type        string
	Description string
	Required    bool
	Fields      map[string]ActionField
	Items       *ActionField
	Format      string
	Enum        []any
}

// ActionInput describes the action input shape.
type ActionInput struct {
	Type   string
	Fields map[string]ActionField
}

// Action represents one executable assistant action.
type Action interface {
	// Definition returns the action definition for this action.
	Definition() ActionDefinition
	// StatusMessage returns a status message about the action execution, or a default message if not implemented.
	StatusMessage() string
	// Renderer returns an optional deterministic renderer for successful action results.
	Renderer() (ActionResultRenderer, bool)
	// Execute runs the action with the given input and returns the resulting assistant message.
	Execute(context.Context, ActionCall, []Message) Message
}

// ActionRegistry resolves and executes assistant actions.
type ActionRegistry interface {
	// Execute runs the given action call and returns the resulting assistant message.
	Execute(context.Context, ActionCall, []Message) Message
	// GetDefinition returns one action definition by name.
	GetDefinition(actionName string) (ActionDefinition, bool)
	// GetRenderer returns one deterministic action result renderer by action name when available.
	GetRenderer(actionName string) (ActionResultRenderer, bool)
	// StatusMessage returns a status message about the action execution, or a default message if not implemented.
	StatusMessage(actionName string) string
}

// ActionResultRenderer converts a raw action result into a deterministic
// assistant-facing message when the action result format is known.
type ActionResultRenderer interface {
	// Render transforms a successful action result into an assistant message.
	// It returns ok=false when the renderer does not support the action or the
	// result payload cannot be deterministically interpreted.
	Render(actionCall ActionCall, result Message) (rendered Message, ok bool)
}
