// pkg/wispy/errors.go
package grove

import "grove/internal/groverrors"

// ParseError is returned for syntax errors. Template, Line, and Column identify the source location.
type ParseError = groverrors.ParseError

// RuntimeError is returned for errors during template execution.
type RuntimeError = groverrors.RuntimeError
