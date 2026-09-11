package model

import (
	"sync"

	"github.com/chenlong/keystone/internal/types"
)

// Registry is a concurrent-safe model adapter registration center.
type Registry struct {
	mu       sync.RWMutex
	adapters map[string]ModelAdapter
}

// NewRegistry creates an empty model registry.
func NewRegistry() *Registry {
	return &Registry{
		adapters: make(map[string]ModelAdapter),
	}
}

// Register adds a model adapter.
func (r *Registry) Register(a ModelAdapter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[a.ID()] = a
}

// Get retrieves a model adapter by ID.
func (r *Registry) Get(id string) (ModelAdapter, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	a, ok := r.adapters[id]
	return a, ok
}

// List returns all registered model providers.
func (r *Registry) List() []types.ModelProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]types.ModelProvider, 0, len(r.adapters))
	for _, a := range r.adapters {
		result = append(result, types.ModelProvider{
			ID:            a.ID(),
			ProviderType:  types.ProviderType(a.ProviderType()),
			SupportsTools: a.SupportsTools(),
			Enabled:       true,
		})
	}
	return result
}
