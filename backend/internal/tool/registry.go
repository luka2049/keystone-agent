package tool

import (
	"sync"

	"github.com/chenlong/keystone/internal/types"
)

// Registry is a concurrent-safe tool registration center.
// Tools can be registered at startup (built-in) or dynamically via API (ToolSpec).
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
	specs map[string]types.ToolSpec // keep specs for API responses
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
		specs: make(map[string]types.ToolSpec),
	}
}

// Register adds a Tool (and its spec if available) to the registry.
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.ID()] = t
}

// RegisterSpec registers a tool from a ToolSpec (dynamic registration via API).
func (r *Registry) RegisterSpec(spec types.ToolSpec) error {
	t, err := FromSpec(spec)
	if err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[spec.ID] = t
	r.specs[spec.ID] = spec
	return nil
}

// Get retrieves a tool by ID.
func (r *Registry) Get(id string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[id]
	return t, ok
}

// GetSpec retrieves a ToolSpec by ID.
func (r *Registry) GetSpec(id string) (types.ToolSpec, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.specs[id]
	return s, ok
}

// List returns all registered tool specs.
func (r *Registry) ListSpecs() []types.ToolSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]types.ToolSpec, 0, len(r.specs))
	for _, s := range r.specs {
		result = append(result, s)
	}
	return result
}

// List returns all registered tools.
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	return result
}

// Delete removes a tool by ID (soft delete = remove from registry).
func (r *Registry) Delete(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, existed := r.tools[id]
	delete(r.tools, id)
	delete(r.specs, id)
	return existed
}
