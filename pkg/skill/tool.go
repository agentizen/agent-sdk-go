package skill

import (
	"context"
	"fmt"
	"strings"

	"github.com/agentizen/agent-sdk-go/pkg/tool"
)

// loadSkillTool implements tool.Tool for loading skill content by name.
type loadSkillTool struct {
	skills map[string]Skill
	names  []string
}

// NewLoadSkillTool creates a tool.Tool that allows an agent to load
// the full content of a skill by name. The tool description lists all
// available skill names, and the parameters schema constrains the name
// to the known set of skills.
func NewLoadSkillTool(skills []Skill) tool.Tool {
	skillMap := make(map[string]Skill, len(skills))
	names := make([]string, 0, len(skills))
	for _, s := range skills {
		name := s.Header().Name
		skillMap[name] = s
		names = append(names, name)
	}
	return &loadSkillTool{
		skills: skillMap,
		names:  names,
	}
}

// GetName returns the tool name.
func (t *loadSkillTool) GetName() string {
	return "load_skill"
}

// GetDescription returns the tool description including available skill names.
func (t *loadSkillTool) GetDescription() string {
	return fmt.Sprintf(
		"Load the full content of a skill by name. Available skills: %s",
		strings.Join(t.names, ", "),
	)
}

// GetParametersSchema returns the JSON schema for the tool parameters.
func (t *loadSkillTool) GetParametersSchema() map[string]interface{} {
	enumValues := make([]interface{}, len(t.names))
	for i, name := range t.names {
		enumValues[i] = name
	}
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{
				"type":        "string",
				"description": "Name of the skill to load",
				"enum":        enumValues,
			},
		},
		"required": []string{"name"},
	}
}

// Execute finds the skill by name and returns its full content.
func (t *loadSkillTool) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	nameRaw, ok := params["name"]
	if !ok {
		return nil, fmt.Errorf("skill: missing required parameter \"name\"")
	}
	name, ok := nameRaw.(string)
	if !ok {
		return nil, fmt.Errorf("skill: parameter \"name\" must be a string")
	}

	s, ok := t.skills[name]
	if !ok {
		return nil, fmt.Errorf("skill: unknown skill %q", name)
	}

	content, err := s.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("skill: failed to load skill %q: %w", name, err)
	}
	return content, nil
}
