// internal/wispyrrors/errors.go
package wispyrrors

import "fmt"

type ParseError struct {
	Template string
	Message  string
	Line     int
	Column   int
}

func (e *ParseError) Error() string {
	if e.Template != "" {
		return fmt.Sprintf("%s:%d:%d: %s", e.Template, e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("line %d:%d: %s", e.Line, e.Column, e.Message)
}

type RuntimeError struct {
	Template string
	Message  string
	Line     int
}

func (e *RuntimeError) Error() string {
	if e.Template != "" {
		return fmt.Sprintf("%s:%d: %s", e.Template, e.Line, e.Message)
	}
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}
