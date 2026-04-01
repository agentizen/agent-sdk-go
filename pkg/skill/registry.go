package skill

import (
	"fmt"
	"sort"
	"sync"
)

// Registry is a thread-safe container for managing skills by name.
type Registry struct {
	mu     sync.RWMutex
	skills map[string]Skill
}

// NewRegistry creates a new empty skill registry.
func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]Skill),
	}
}

// Register adds a skill to the registry. It returns an error if a skill
// with the same name is already registered.
func (r *Registry) Register(s Skill) error {
	name := s.Header().Name
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.skills[name]; exists {
		return fmt.Errorf("skill: skill %q is already registered", name)
	}
	r.skills[name] = s
	return nil
}

// Get retrieves a skill by name. The second return value indicates whether
// the skill was found.
func (r *Registry) Get(name string) (Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.skills[name]
	return s, ok
}

// All returns all registered skills sorted by name.
func (r *Registry) All() []Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Skill, 0, len(r.skills))
	for _, s := range r.skills {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Header().Name < result[j].Header().Name
	})
	return result
}

// Names returns the sorted names of all registered skills.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.skills))
	for name := range r.skills {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Remove deletes a skill from the registry by name. It is a no-op if the
// skill does not exist.
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.skills, name)
}
