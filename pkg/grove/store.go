// pkg/wispy/store.go
package grove

import "grove/internal/store"

// MemoryStore holds templates in memory. Use NewMemoryStore() to create one.
// Pass to an Engine via grove.WithStore(s).
type MemoryStore = store.MemoryStore

// NewMemoryStore creates an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return store.NewMemoryStore()
}

// FileSystemStore loads templates from a root directory on disk.
// Template names that escape the root via ".." or that are absolute paths are rejected.
type FileSystemStore = store.FileSystemStore

// NewFileSystemStore creates a FileSystemStore rooted at root.
func NewFileSystemStore(root string) *FileSystemStore {
	return store.NewFileSystemStore(root)
}
