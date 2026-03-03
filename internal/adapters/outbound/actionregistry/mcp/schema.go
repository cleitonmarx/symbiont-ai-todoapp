package mcp

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// toolToActionDefinition converts an MCP tool schema into the domain action format.
func toolToActionDefinition(tool *mcp.Tool) domain.AssistantActionDefinition {
	if tool == nil {
		return domain.AssistantActionDefinition{}
	}

	description := strings.TrimSpace(tool.Description)
	if description == "" {
		description = strings.TrimSpace(tool.Title)
	}

	return domain.AssistantActionDefinition{
		Name:        strings.TrimSpace(tool.Name),
		Description: description,
		Input:       schemaToInput(tool.InputSchema),
	}
}

// schemaToInput extracts a simplified action input definition from JSON Schema-like MCP input.
func schemaToInput(schema any) domain.AssistantActionInput {
	input := domain.AssistantActionInput{
		Type:   "object",
		Fields: map[string]domain.AssistantActionField{},
	}

	schemaMap, ok := anyToMap(schema)
	if !ok || len(schemaMap) == 0 {
		return input
	}

	if schemaType := strings.TrimSpace(asString(schemaMap["type"])); schemaType != "" {
		input.Type = schemaType
	}

	required := requiredSet(schemaMap["required"])
	props, ok := anyToMap(schemaMap["properties"])
	if !ok {
		return input
	}

	for fieldName, fieldSchemaRaw := range props {
		fieldSchema, _ := anyToMap(fieldSchemaRaw)
		input.Fields[fieldName] = schemaFieldToDomain(fieldSchema, required[fieldName])
	}

	return input
}

// overrideFieldToDomain converts one assistantActionFieldOverride block into a domain AssistantActionField recursively.
func overrideFieldToDomain(field assistantActionFieldOverride) domain.AssistantActionField {
	result := domain.AssistantActionField{
		Type:        strings.TrimSpace(field.Type),
		Description: strings.TrimSpace(field.Description),
		Required:    field.Required,
		Format:      strings.TrimSpace(field.Format),
		Enum:        field.Enum,
	}

	if len(field.Fields) > 0 {
		result.Fields = make(map[string]domain.AssistantActionField, len(field.Fields))
		for fieldName, child := range field.Fields {
			result.Fields[fieldName] = overrideFieldToDomain(child)
		}
	}

	if field.Items != nil {
		items := overrideFieldToDomain(*field.Items)
		result.Items = &items
	}

	return result
}

// schemaFieldToDomain converts one JSON Schema field definition into a domain AssistantActionField recursively.
func schemaFieldToDomain(fieldSchema map[string]any, required bool) domain.AssistantActionField {
	result := domain.AssistantActionField{
		Type:        schemaFieldType(fieldSchema),
		Description: strings.TrimSpace(asString(fieldSchema["description"])),
		Required:    required,
		Format:      strings.TrimSpace(asString(fieldSchema["format"])),
	}

	if enumValues, ok := fieldSchema["enum"].([]any); ok && len(enumValues) > 0 {
		result.Enum = enumValues
	}

	if props, ok := anyToMap(fieldSchema["properties"]); ok && len(props) > 0 {
		result.Fields = make(map[string]domain.AssistantActionField, len(props))
		requiredFields := requiredSet(fieldSchema["required"])
		for name, raw := range props {
			childSchema, _ := anyToMap(raw)
			result.Fields[name] = schemaFieldToDomain(childSchema, requiredFields[name])
		}
	}

	if itemsSchema, ok := anyToMap(fieldSchema["items"]); ok && len(itemsSchema) > 0 {
		items := schemaFieldToDomain(itemsSchema, false)
		result.Items = &items
	}

	return result
}

// schemaFieldType resolves field type from direct or composed schema nodes (anyOf/oneOf/allOf).
func schemaFieldType(fieldSchema map[string]any) string {
	if len(fieldSchema) == 0 {
		return ""
	}

	if direct := strings.TrimSpace(parseTypeValue(fieldSchema["type"])); direct != "" {
		return direct
	}

	compoundKeys := []string{"anyOf", "oneOf", "allOf"}
	typeCandidates := make([]string, 0, 2)
	for _, key := range compoundKeys {
		values, ok := fieldSchema[key].([]any)
		if !ok {
			continue
		}
		for _, raw := range values {
			inner, ok := anyToMap(raw)
			if !ok {
				continue
			}
			if typ := strings.TrimSpace(parseTypeValue(inner["type"])); typ != "" {
				typeCandidates = append(typeCandidates, typ)
			}
		}
	}

	if len(typeCandidates) == 0 {
		return ""
	}
	typeCandidates = slices.Compact(typeCandidates)
	sort.Strings(typeCandidates)
	return strings.Join(typeCandidates, "|")
}

// parseTypeValue normalizes type declarations that may be a single value or an array of values.
func parseTypeValue(raw any) string {
	switch value := raw.(type) {
	case string:
		return value
	case []any:
		candidates := make([]string, 0, len(value))
		for _, item := range value {
			itemString := strings.TrimSpace(asString(item))
			if itemString == "" {
				continue
			}
			candidates = append(candidates, itemString)
		}
		if len(candidates) == 0 {
			return ""
		}
		candidates = slices.Compact(candidates)
		sort.Strings(candidates)
		return strings.Join(candidates, "|")
	default:
		return ""
	}
}

// requiredSet converts a JSON schema required array to a lookup set.
func requiredSet(raw any) map[string]bool {
	set := map[string]bool{}
	values, ok := raw.([]any)
	if !ok {
		return set
	}
	for _, value := range values {
		name := strings.TrimSpace(asString(value))
		if name == "" {
			continue
		}
		set[name] = true
	}
	return set
}

// mergeAssistantActionDefinition overlays configured overrides on top of discovered tool metadata.
func mergeAssistantActionDefinition(base, override domain.AssistantActionDefinition) domain.AssistantActionDefinition {
	merged := base

	if name := strings.TrimSpace(override.Name); name != "" {
		merged.Name = name
	}
	if description := strings.TrimSpace(override.Description); description != "" {
		merged.Description = description
	}

	if inputType := strings.TrimSpace(override.Input.Type); inputType != "" {
		merged.Input.Type = inputType
	}
	baseFields := map[string]domain.AssistantActionField{}
	if len(base.Input.Fields) > 0 {
		baseFields = make(map[string]domain.AssistantActionField, len(base.Input.Fields))
		maps.Copy(baseFields, base.Input.Fields)
	}
	merged.Input.Fields = baseFields
	maps.Copy(merged.Input.Fields, override.Input.Fields)

	if hasApprovalOverride(override.Approval) {
		merged.Approval = override.Approval
	}
	return merged
}

func hasApprovalOverride(approval domain.AssistantActionApproval) bool {
	return approval.Required ||
		strings.TrimSpace(approval.Title) != "" ||
		strings.TrimSpace(approval.Description) != "" ||
		len(approval.PreviewFields) > 0 ||
		approval.Timeout > 0
}

// anyToMap best-effort normalizes unknown schema values to map[string]any.
func anyToMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case nil:
		return nil, false
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return nil, false
		}
		var out map[string]any
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, false
		}
		return out, true
	}
}

// asString stringifies unknown values for permissive schema parsing.
func asString(v any) string {
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		return value
	default:
		return fmt.Sprintf("%v", v)
	}
}
