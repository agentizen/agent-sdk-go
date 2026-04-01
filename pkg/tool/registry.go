package tool

import (
	"fmt"
	"sort"
	"sync"
)

// Registry is a thread-safe container for managing tools and tool groups.
// It allows registering, retrieving, and removing tools by name, as well as
// organizing tools into named groups.
type Registry struct {
	mu     sync.RWMutex
	tools  map[string]Tool
	groups map[string][]string // group name -> tool names
}

// NewRegistry creates a new empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		tools:  make(map[string]Tool),
		groups: make(map[string][]string),
	}
}

// Register adds a tool to the registry. It returns an error if a tool
// with the same name is already registered.
func (r *Registry) Register(t Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := t.GetName()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool %q is already registered", name)
	}

	r.tools[name] = t
	return nil
}

// RegisterGroup registers multiple tools under a named group. All tools
// are individually registered in the registry, and their names are associated
// with the group. It returns an error if any tool has a duplicate name.
func (r *Registry) RegisterGroup(name string, tools ...Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	toolNames := make([]string, 0, len(tools))
	seen := make(map[string]bool, len(tools))
	for _, t := range tools {
		toolName := t.GetName()
		if _, exists := r.tools[toolName]; exists {
			return fmt.Errorf("tool %q is already registered", toolName)
		}
		if seen[toolName] {
			return fmt.Errorf("tool %q appears more than once in the group", toolName)
		}
		seen[toolName] = true
		toolNames = append(toolNames, toolName)
	}

	// Register all tools after validation passes
	for _, t := range tools {
		r.tools[t.GetName()] = t
	}
	r.groups[name] = toolNames

	return nil
}

// Get retrieves a tool by name. The second return value indicates whether
// the tool was found.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.tools[name]
	return t, ok
}

// GetGroup returns all tools belonging to the named group. If the group
// does not exist or contains no valid tools, an empty slice is returned.
func (r *Registry) GetGroup(name string) []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	toolNames, ok := r.groups[name]
	if !ok {
		return nil
	}

	tools := make([]Tool, 0, len(toolNames))
	for _, tn := range toolNames {
		if t, exists := r.tools[tn]; exists {
			tools = append(tools, t)
		}
	}
	return tools
}

// All returns all registered tools. The order is not guaranteed.
func (r *Registry) All() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}

// Names returns the sorted names of all registered tools.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Remove unregisters a tool by name. If the tool does not exist, this is a no-op.
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.tools, name)
}
