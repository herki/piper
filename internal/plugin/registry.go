package plugin

import (
	"fmt"
	"sync"
)

// Registry holds all registered connectors.
type Registry struct {
	mu         sync.RWMutex
	connectors map[string]Connector
}

// NewRegistry creates a new empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		connectors: make(map[string]Connector),
	}
}

// Register adds a connector to the registry.
func (r *Registry) Register(c Connector) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := c.Name()
	if _, exists := r.connectors[name]; exists {
		return fmt.Errorf("connector %q already registered", name)
	}
	r.connectors[name] = c
	return nil
}

// Get returns a connector by name.
func (r *Registry) Get(name string) (Connector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, ok := r.connectors[name]
	return c, ok
}

// List returns the names of all registered connectors.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.connectors))
	for name := range r.connectors {
		names = append(names, name)
	}
	return names
}

// Has checks whether a connector is registered.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.connectors[name]
	return ok
}
