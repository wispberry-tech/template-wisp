// pkg/grove/store.go
package grove

import "grove/internal/store"

// MemoryStore holds templates in memory. Use NewMemoryStore() to create one.
// Pass to an Engine via grove.WithStore(s).
type MemoryStore = store.MemoryStore

// NewMemoryStore creates an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return store.NewMemoryStore()
}
