package skill

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// fileSkill is an unexported implementation of the Skill interface
// that stores eagerly parsed header metadata and the body content.
type fileSkill struct {
	header Header
	body   string
}

// Header returns the skill metadata.
func (s *fileSkill) Header() Header {
	return s.header
}

// Load returns the full skill body content.
func (s *fileSkill) Load(_ context.Context) (string, error) {
	return s.body, nil
}

// LoadFromFile reads a markdown file with YAML frontmatter from the given path
// and returns a Skill with the parsed header and body.
func LoadFromFile(path string) (Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("skill: failed to read file %s: %w", path, err)
	}
	return LoadFromString(string(data))
}

// LoadFromReader parses a skill from the given reader. The content must be a
// markdown file with YAML frontmatter delimited by --- lines.
func LoadFromReader(r io.Reader) (Skill, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("skill: failed to read from reader: %w", err)
	}
	return LoadFromString(string(data))
}

// LoadFromString parses a skill from a raw string. The content must be a
// markdown file with YAML frontmatter delimited by --- lines.
func LoadFromString(content string) (Skill, error) {
	header, body, err := parseFrontmatter(content)
	if err != nil {
		return nil, err
	}
	return &fileSkill{
		header: header,
		body:   body,
	}, nil
}

// parseFrontmatter splits the content on --- delimiters, parses the YAML
// header block, and returns the header and the remaining body.
func parseFrontmatter(content string) (Header, string, error) {
	trimmed := strings.TrimSpace(content)
	if !strings.HasPrefix(trimmed, "---") {
		return Header{}, "", fmt.Errorf("skill: content does not start with frontmatter delimiter (---)")
	}

	// Remove the leading ---
	rest := trimmed[3:]

	// Find the closing ---
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return Header{}, "", fmt.Errorf("skill: missing closing frontmatter delimiter (---)")
	}

	yamlBlock := rest[:idx]
	// Body starts after the closing --- and the newline
	body := strings.TrimLeft(rest[idx+4:], "\n")

	var header Header
	if err := yaml.Unmarshal([]byte(yamlBlock), &header); err != nil {
		return Header{}, "", fmt.Errorf("skill: failed to parse frontmatter YAML: %w", err)
	}

	return header, body, nil
}
