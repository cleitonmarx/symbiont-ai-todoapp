package skillregistry

import (
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/cleitonmarx/symbiont-ai-todoapp/internal/domain/assistant"
	"go.yaml.in/yaml/v3"
)

// skillFrontMatter mirrors the YAML metadata supported by markdown skill files.
type skillFrontMatter struct {
	Name                  string   `yaml:"name"`
	UseWhen               string   `yaml:"use_when"`
	AvoidWhen             string   `yaml:"avoid_when"`
	Priority              int      `yaml:"priority"`
	Tags                  []string `yaml:"tags"`
	Tools                 []string `yaml:"tools"`
	EmbedFirstContentLine bool     `yaml:"embed_first_content_line"`
}

// LoadSkillsFromFS reads markdown skills from a filesystem tree and returns
// them sorted by priority.
func LoadSkillsFromFS(skillsFS fs.FS) ([]assistant.SkillDefinition, error) {
	if skillsFS == nil {
		return nil, errors.New("skills fs is nil")
	}

	skills := make([]assistant.SkillDefinition, 0)
	err := fs.WalkDir(skillsFS, ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(path), ".md") {
			return nil
		}

		content, err := fs.ReadFile(skillsFS, path)
		if err != nil {
			return fmt.Errorf("failed to read skill file %q: %w", path, err)
		}

		skill, err := parseSkillMarkdown(path, content)
		if err != nil {
			return fmt.Errorf("failed to parse skill file %q: %w", path, err)
		}

		skills = append(skills, skill)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Priority == skills[j].Priority {
			return skills[i].Name < skills[j].Name
		}
		return skills[i].Priority > skills[j].Priority
	})

	return skills, nil
}

// parseSkillMarkdown parses one markdown skill file into a domain definition.
func parseSkillMarkdown(path string, content []byte) (assistant.SkillDefinition, error) {
	raw := strings.ReplaceAll(string(content), "\r\n", "\n")
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return assistant.SkillDefinition{}, errors.New("missing YAML frontmatter opening delimiter")
	}

	endIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			endIdx = i
			break
		}
	}
	if endIdx == -1 {
		return assistant.SkillDefinition{}, errors.New("missing YAML frontmatter closing delimiter")
	}

	metaRaw := strings.Join(lines[1:endIdx], "\n")
	var meta skillFrontMatter
	if err := yaml.Unmarshal([]byte(metaRaw), &meta); err != nil {
		return assistant.SkillDefinition{}, fmt.Errorf("invalid YAML frontmatter: %w", err)
	}

	name := strings.TrimSpace(meta.Name)
	if name == "" {
		return assistant.SkillDefinition{}, errors.New("skill name is required")
	}

	body := strings.TrimSpace(strings.Join(lines[endIdx+1:], "\n"))
	if body == "" {
		return assistant.SkillDefinition{}, errors.New("skill content is required")
	}

	return assistant.SkillDefinition{
		Name:                  name,
		UseWhen:               strings.TrimSpace(meta.UseWhen),
		AvoidWhen:             strings.TrimSpace(meta.AvoidWhen),
		Priority:              max(0, meta.Priority),
		Tags:                  sanitizeStringList(meta.Tags),
		Tools:                 sanitizeStringList(meta.Tools),
		EmbedFirstContentLine: meta.EmbedFirstContentLine,
		Content:               body,
		Source:                path,
	}, nil
}

// sanitizeStringList trims, de-duplicates, and removes empty string values.
func sanitizeStringList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	next := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, raw := range values {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		next = append(next, trimmed)
	}
	if len(next) == 0 {
		return nil
	}
	return next
}

// copySkillDefinitions deep-copies slice fields so registry state is isolated
// from caller-owned skill definitions.
func copySkillDefinitions(skills []assistant.SkillDefinition) []assistant.SkillDefinition {
	if len(skills) == 0 {
		return nil
	}

	copied := make([]assistant.SkillDefinition, 0, len(skills))
	for _, skill := range skills {
		tags := make([]string, len(skill.Tags))
		copy(tags, skill.Tags)
		skill.Tags = tags
		tools := make([]string, len(skill.Tools))
		copy(tools, skill.Tools)
		skill.Tools = tools
		copied = append(copied, skill)
	}
	return copied
}
