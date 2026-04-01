// pkg/wispy/context.go
package wispy

import "wispy/internal/vm"

// Data is the map type passed to Render methods.
type Data map[string]any

// Resolvable is implemented by Go types that want to expose fields to templates.
// Only keys returned by WispyResolve are accessible; all other fields are hidden.
type Resolvable = vm.Resolvable
