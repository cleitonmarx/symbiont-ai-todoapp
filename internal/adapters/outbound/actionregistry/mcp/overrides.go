package mcp

import (
	"embed"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"go.yaml.in/yaml/v3"
)

//go:embed tool_overrides.yaml
var toolOverridesFS embed.FS

// toolOverrideConfig represents the structure of the YAML configuration for overriding tool definitions and status messages.
type toolOverrideConfig struct {
	Tools []assistantActionDefinitionOverride `yaml:"tools"`
}

// assistantActionDefinitionOverride allows partial overrides of discovered MCP tool metadata.
type assistantActionDefinitionOverride struct {
	Name          string                        `yaml:"name"`
	Description   string                        `yaml:"description"`
	StatusMessage string                        `yaml:"status_message"`
	Input         assistantActionInputConfig    `yaml:"input"`
	Approval      assistantActionApprovalConfig `yaml:"approval"`
	Approvals     assistantActionApprovalConfig `yaml:"approvals"`
}

// assistantActionInputConfig allows overriding MCP tool input schema with a simplified format.
type assistantActionInputConfig struct {
	Type   string                                  `yaml:"type"`
	Fields map[string]assistantActionFieldOverride `yaml:"fields"`
}

// assistantActionFieldOverride represents the YAML structure for overriding individual input fields of an MCP tool.
type assistantActionFieldOverride struct {
	Type        string                                  `yaml:"type"`
	Description string                                  `yaml:"description"`
	Required    bool                                    `yaml:"required"`
	Format      string                                  `yaml:"format"`
	Enum        []any                                   `yaml:"enum"`
	Fields      map[string]assistantActionFieldOverride `yaml:"fields"`
	Items       *assistantActionFieldOverride           `yaml:"items"`
}

// assistantActionApprovalConfig allows configuring human-in-the-loop approval policies for MCP tools.
type assistantActionApprovalConfig struct {
	Required      bool     `yaml:"required"`
	Title         string   `yaml:"title"`
	Description   string   `yaml:"description"`
	PreviewFields []string `yaml:"preview_fields"`
	Timeout       string   `yaml:"timeout"`
}

// toDomain converts one approval override block into a domain approval policy.
func (c assistantActionApprovalConfig) toDomain() (assistant.ActionApproval, error) {
	timeout, err := parseApprovalTimeout(c.Timeout)
	if err != nil {
		return assistant.ActionApproval{}, err
	}

	return assistant.ActionApproval{
		Required:      c.Required,
		Title:         strings.TrimSpace(c.Title),
		Description:   strings.TrimSpace(c.Description),
		PreviewFields: sanitizePreviewFields(c.PreviewFields),
		Timeout:       timeout,
	}, nil
}

// hasValues returns true when at least one approval override field is set.
func (c assistantActionApprovalConfig) hasValues() bool {
	return c.Required ||
		strings.TrimSpace(c.Title) != "" ||
		strings.TrimSpace(c.Description) != "" ||
		len(sanitizePreviewFields(c.PreviewFields)) > 0 ||
		strings.TrimSpace(c.Timeout) != ""
}

func sanitizePreviewFields(fields []string) []string {
	if len(fields) == 0 {
		return nil
	}

	next := make([]string, 0, len(fields))
	for _, raw := range fields {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		next = append(next, trimmed)
	}
	if len(next) == 0 {
		return nil
	}
	return slices.Compact(next)
}

// parseApprovalTimeout supports duration strings (e.g., "30s").
func parseApprovalTimeout(timeout string) (time.Duration, error) {
	trimmed := strings.TrimSpace(timeout)
	if trimmed != "" {
		parsed, err := time.ParseDuration(trimmed)
		if err != nil {
			return 0, fmt.Errorf("invalid approval timeout %q: %w", trimmed, err)
		}
		return parsed, nil
	}
	return 0, nil
}

// toolOverrides bundles both definition and status message overrides loaded from YAML.
type toolOverrides struct {
	Definitions    map[string]assistant.ActionDefinition
	StatusMessages map[string]string
}

func loadToolOverrides() (toolOverrides, error) {
	embeddedBytes, err := toolOverridesFS.ReadFile("tool_overrides.yaml")
	if err != nil {
		return toolOverrides{}, err
	}
	return parseToolOverrides(embeddedBytes)
}

// parseToolOverrideDefinitions parses the embedded YAML override file into action definitions.
func parseToolOverrideDefinitions(content []byte) (map[string]assistant.ActionDefinition, error) {
	overrides, err := parseToolOverrides(content)
	if err != nil {
		return nil, err
	}
	return overrides.Definitions, nil
}

// parseToolOverrideStatusMessages parses the embedded YAML override file into status messages.
func parseToolOverrideStatusMessages(content []byte) (map[string]string, error) {
	overrides, err := parseToolOverrides(content)
	if err != nil {
		return nil, err
	}
	return overrides.StatusMessages, nil
}

// parseToolOverrides unmarshals the YAML configuration and separates definition and status message overrides.
func parseToolOverrides(content []byte) (toolOverrides, error) {
	var cfg toolOverrideConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return toolOverrides{}, err
	}

	byName := map[string]assistant.ActionDefinition{}
	statusByName := map[string]string{}
	for _, override := range cfg.Tools {
		name := strings.TrimSpace(override.Name)
		if name == "" {
			continue
		}

		if statusMessage := strings.TrimSpace(override.StatusMessage); statusMessage != "" {
			statusByName[name] = statusMessage
		}

		fields := map[string]assistant.ActionField{}
		for fieldName, field := range override.Input.Fields {
			fields[fieldName] = overrideFieldToDomain(field)
		}

		def := assistant.ActionDefinition{
			Name:        name,
			Description: strings.TrimSpace(override.Description),
			Input: assistant.ActionInput{
				Type:   strings.TrimSpace(override.Input.Type),
				Fields: fields,
			},
		}

		approvalCfg := override.Approval
		if override.Approvals.hasValues() {
			approvalCfg = override.Approvals
		}
		if approvalCfg.hasValues() {
			approval, err := approvalCfg.toDomain()
			if err != nil {
				return toolOverrides{}, fmt.Errorf("tool %q approval override: %w", name, err)
			}
			def.Approval = approval
		}

		byName[name] = def
	}
	return toolOverrides{
		Definitions:    byName,
		StatusMessages: statusByName,
	}, nil
}
