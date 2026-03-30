// internal/store/store.go
package store

import (
	"fmt"
	"sync"
)

// Store is the template source backend.
type Store interface {
	// Load returns the template source for the given name, or an error if not found.
	Load(name string) ([]byte, error)
}

// MemoryStore holds templates in memory. Safe for concurrent use.
type MemoryStore struct {
	mu    sync.RWMutex
	tmpls map[string]string
}

// NewMemoryStore creates an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{tmpls: make(map[string]string)}
}

// Set stores a template under the given name.
func (s *MemoryStore) Set(name, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tmpls[name] = content
}

// Load implements Store.
func (s *MemoryStore) Load(name string) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	content, ok := s.tmpls[name]
	if !ok {
		return nil, fmt.Errorf("template %q not found", name)
	}
	return []byte(content), nil
}
