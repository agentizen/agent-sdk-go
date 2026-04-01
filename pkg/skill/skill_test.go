package skill_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/agentizen/agent-sdk-go/pkg/skill"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const validSkillContent = `---
name: greet
description: A greeting skill
version: "1.0"
---
Hello, world!
This is the body.`

func TestLoadFromString_ValidFrontmatter(t *testing.T) {
	s, err := skill.LoadFromString(validSkillContent)
	require.NoError(t, err)

	h := s.Header()
	assert.Equal(t, "greet", h.Name)
	assert.Equal(t, "A greeting skill", h.Description)
	assert.Equal(t, "1.0", h.Version)
}

func TestLoadFromString_BodyContent(t *testing.T) {
	s, err := skill.LoadFromString(validSkillContent)
	require.NoError(t, err)

	body, err := s.Load(context.Background())
	require.NoError(t, err)
	assert.Contains(t, body, "Hello, world!")
	assert.Contains(t, body, "This is the body.")
}

func TestLoadFromString_MissingOpeningDelimiter(t *testing.T) {
	content := `name: greet
---
body content`

	_, err := skill.LoadFromString(content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "frontmatter delimiter")
}

func TestLoadFromString_MissingClosingDelimiter(t *testing.T) {
	content := `---
name: greet
description: missing close`

	_, err := skill.LoadFromString(content)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closing frontmatter delimiter")
}

func TestLoadFromString_EmptyName(t *testing.T) {
	content := `---
name: ""
description: empty name skill
version: "1.0"
---
Body text.`

	s, err := skill.LoadFromString(content)
	require.NoError(t, err)
	// The loader does not validate name; an empty name is accepted.
	assert.Equal(t, "", s.Header().Name)
}

func TestLoadFromString_HeaderReturnsCorrectValues(t *testing.T) {
	content := `---
name: analytics
description: Business analytics helper
version: "2.5.1"
---
Some body.`

	s, err := skill.LoadFromString(content)
	require.NoError(t, err)

	h := s.Header()
	assert.Equal(t, "analytics", h.Name)
	assert.Equal(t, "Business analytics helper", h.Description)
	assert.Equal(t, "2.5.1", h.Version)
}

func TestLoadFromString_LoadReturnsBody(t *testing.T) {
	content := `---
name: test
description: test skill
version: "0.1"
---
Line one.
Line two.
Line three.`

	s, err := skill.LoadFromString(content)
	require.NoError(t, err)

	body, err := s.Load(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "Line one.\nLine two.\nLine three.", body)
}

func TestLoadFromFile_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.md")
	err := os.WriteFile(path, []byte(validSkillContent), 0644)
	require.NoError(t, err)

	s, err := skill.LoadFromFile(path)
	require.NoError(t, err)

	assert.Equal(t, "greet", s.Header().Name)

	body, err := s.Load(context.Background())
	require.NoError(t, err)
	assert.Contains(t, body, "Hello, world!")
}

func TestLoadFromFile_NonexistentFile(t *testing.T) {
	_, err := skill.LoadFromFile("/nonexistent/path/skill.md")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read file")
}

func TestLoadFromString_WhitespaceBeforeDelimiter(t *testing.T) {
	content := `
---
name: trimmed
description: leading whitespace
version: "1.0"
---
Body.`

	s, err := skill.LoadFromString(content)
	require.NoError(t, err)
	assert.Equal(t, "trimmed", s.Header().Name)
}
