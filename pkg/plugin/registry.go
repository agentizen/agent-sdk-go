package plugin

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Registry manages registered plugins in a thread-safe manner.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
}

// NewRegistry creates a new plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
	}
}

// Register adds a plugin to the registry, calling Init before storing it.
// Returns an error if a plugin with the same name already exists or if Init fails.
func (r *Registry) Register(ctx context.Context, p Plugin) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := p.Name()
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %q is already registered", name)
	}

	if err := p.Init(ctx); err != nil {
		return fmt.Errorf("plugin %q init failed: %w", name, err)
	}

	r.plugins[name] = p
	return nil
}

// Get returns the plugin with the given name, or false if not found.
func (r *Registry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[name]
	return p, ok
}

// All returns all registered plugins sorted by name.
func (r *Registry) All() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.plugins))
	for name := range r.plugins {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]Plugin, 0, len(names))
	for _, name := range names {
		result = append(result, r.plugins[name])
	}
	return result
}

// Unregister removes a plugin from the registry.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.plugins, name)
}
