package skill_test

import (
	"context"
	"testing"

	"github.com/agentizen/agent-sdk-go/pkg/skill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSkills(t *testing.T) []skill.Skill {
	t.Helper()
	contents := []string{
		`---
name: alpha
description: Alpha skill
version: "1.0"
---
Alpha body content.`,
		`---
name: beta
description: Beta skill
version: "2.0"
---
Beta body content.`,
	}
	skills := make([]skill.Skill, 0, len(contents))
	for _, c := range contents {
		s, err := skill.LoadFromString(c)
		require.NoError(t, err)
		skills = append(skills, s)
	}
	return skills
}

func TestNewLoadSkillTool_Name(t *testing.T) {
	tl := skill.NewLoadSkillTool(newTestSkills(t))
	assert.Equal(t, "load_skill", tl.GetName())
}

func TestNewLoadSkillTool_DescriptionListsSkillNames(t *testing.T) {
	tl := skill.NewLoadSkillTool(newTestSkills(t))
	desc := tl.GetDescription()
	assert.Contains(t, desc, "alpha")
	assert.Contains(t, desc, "beta")
}

func TestNewLoadSkillTool_ParametersSchema(t *testing.T) {
	tl := skill.NewLoadSkillTool(newTestSkills(t))
	schema := tl.GetParametersSchema()

	assert.Equal(t, "object", schema["type"])

	props, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)

	nameProp, ok := props["name"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "string", nameProp["type"])

	enumVals, ok := nameProp["enum"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, enumVals, "alpha")
	assert.Contains(t, enumVals, "beta")

	required, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, required, "name")
}

func TestNewLoadSkillTool_ExecuteValidName(t *testing.T) {
	tl := skill.NewLoadSkillTool(newTestSkills(t))
	result, err := tl.Execute(context.Background(), map[string]interface{}{
		"name": "alpha",
	})
	require.NoError(t, err)
	assert.Equal(t, "Alpha body content.", result)
}

func TestNewLoadSkillTool_ExecuteUnknownName(t *testing.T) {
	tl := skill.NewLoadSkillTool(newTestSkills(t))
	_, err := tl.Execute(context.Background(), map[string]interface{}{
		"name": "nonexistent",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown skill")
}

func TestNewLoadSkillTool_ExecuteMissingNameParam(t *testing.T) {
	tl := skill.NewLoadSkillTool(newTestSkills(t))
	_, err := tl.Execute(context.Background(), map[string]interface{}{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing required parameter")
}

func TestNewLoadSkillTool_ExecuteNameNotString(t *testing.T) {
	tl := skill.NewLoadSkillTool(newTestSkills(t))
	_, err := tl.Execute(context.Background(), map[string]interface{}{
		"name": 42,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be a string")
}
