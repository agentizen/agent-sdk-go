package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/agentizen/agent-sdk-go/pkg/tool"
)

// Registry is a thread-safe registry of MCP server configurations.
type Registry struct {
	mu      sync.RWMutex
	servers map[string]ServerConfig
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		servers: make(map[string]ServerConfig),
	}
}

// Register adds a server configuration to the registry.
// Returns an error if a server with the same handle is already registered.
func (r *Registry) Register(cfg ServerConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.servers[cfg.Handle]; exists {
		return fmt.Errorf("mcp: server %q is already registered", cfg.Handle)
	}
	r.servers[cfg.Handle] = cfg
	return nil
}

// Get returns the server configuration for the given handle.
// The second return value indicates whether the handle was found.
func (r *Registry) Get(handle string) (ServerConfig, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	cfg, ok := r.servers[handle]
	return cfg, ok
}

// All returns a snapshot of every registered server configuration.
func (r *Registry) All() []ServerConfig {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ServerConfig, 0, len(r.servers))
	for _, cfg := range r.servers {
		out = append(out, cfg)
	}
	return out
}

// ToolsFor discovers all tools from the server identified by handle and
// returns them as standard tool.Tool values.
func (r *Registry) ToolsFor(ctx context.Context, handle string) ([]tool.Tool, error) {
	cfg, ok := r.Get(handle)
	if !ok {
		return nil, fmt.Errorf("mcp: server %q not found", handle)
	}
	return ToolsFromServer(ctx, cfg)
}

// AllTools discovers tools from every registered server and returns them
// as a combined slice of tool.Tool values.
func (r *Registry) AllTools(ctx context.Context) ([]tool.Tool, error) {
	servers := r.All()
	var all []tool.Tool
	for _, cfg := range servers {
		tools, err := ToolsFromServer(ctx, cfg)
		if err != nil {
			return nil, fmt.Errorf("mcp: failed to list tools for %q: %w", cfg.Handle, err)
		}
		all = append(all, tools...)
	}
	return all, nil
}

// Remove deletes the server configuration for the given handle.
// It is a no-op if the handle does not exist.
func (r *Registry) Remove(handle string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.servers, handle)
}
