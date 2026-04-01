// pkg/wispy/errors.go
package wispy

import "wispy/internal/wispyrrors"

// ParseError is returned for syntax errors. Template, Line, and Column identify the source location.
type ParseError = wispyrrors.ParseError

// RuntimeError is returned for errors during template execution.
type RuntimeError = wispyrrors.RuntimeError
