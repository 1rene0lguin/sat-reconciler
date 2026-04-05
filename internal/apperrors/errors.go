package apperrors

import (
	"fmt"
	"strings"
)

// Param represents a key-value pair for error context.
type Param struct {
	Key   string
	Value any
}

// AppError is a structured application error with context parameters.
type AppError struct {
	Message string
	Params  []Param
	Cause   error
}

// Error returns a human-readable string representation of the error.
// Format: "message [key=value, key2=value2]: cause"
func (e *AppError) Error() string {
	var sb strings.Builder
	sb.WriteString(e.Message)

	if len(e.Params) > 0 {
		sb.WriteString(" [")
		for i, p := range e.Params {
			if i > 0 {
				sb.WriteString(", ")
			}
			fmt.Fprintf(&sb, "%s=%v", p.Key, p.Value)
		}
		sb.WriteString("]")
	}

	if e.Cause != nil {
		sb.WriteString(": ")
		sb.WriteString(e.Cause.Error())
	}

	return sb.String()
}

// Unwrap returns the underlying cause, compatible with errors.Is/As.
func (e *AppError) Unwrap() error {
	return e.Cause
}

// New creates a new AppError with a message constant and optional parameters.
func New(message string, params ...Param) *AppError {
	return &AppError{Message: message, Params: params}
}

// Wrap creates a new AppError wrapping a cause error with optional parameters.
func Wrap(message string, cause error, params ...Param) *AppError {
	return &AppError{Message: message, Params: params, Cause: cause}
}

// P is a shorthand constructor for a Param key-value pair.
func P(key string, value any) Param {
	return Param{Key: key, Value: value}
}
