// Package domain holds entities, value objects, and errors. It imports only
// the standard library.
package domain

import "errors"

// Sentinel errors. Use cases return these (wrapped with context); the HTTP
// adapter maps them to status codes.
var (
	ErrNotFound    = errors.New("not found")
	ErrForbidden   = errors.New("forbidden")
	ErrConflict    = errors.New("conflict")
	ErrUnavailable = errors.New("dependency unavailable") // required dependency (Postgres) down; also returned by llm.Disabled when extraction is off
)

// ValidationError reports per-field problems with an input.
type ValidationError struct {
	Fields map[string]string
}

// NewValidationError builds a ValidationError from field → message.
func NewValidationError(fields map[string]string) *ValidationError {
	return &ValidationError{Fields: fields}
}

func (e *ValidationError) Error() string { return "validation failed" }
