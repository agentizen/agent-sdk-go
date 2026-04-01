package skill

import "context"

// Skill represents a loadable skill with a header and body.
type Skill interface {
	// Header returns the skill metadata (loaded eagerly).
	Header() Header
	// Load returns the full skill content (loaded lazily on demand).
	Load(ctx context.Context) (string, error)
}

// Header contains metadata parsed from the skill's YAML frontmatter.
type Header struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
}
