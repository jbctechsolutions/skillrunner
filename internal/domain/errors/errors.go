// Package errors provides domain-specific errors for the skillrunner application.
package errors

import (
	"errors"
	"fmt"
)

// Sentinel errors for common domain error conditions.
var (
	ErrSkillNotFound       = errors.New("skill not found")
	ErrSkillIDRequired     = errors.New("skill ID required")
	ErrSkillNameRequired   = errors.New("skill name required")
	ErrNoPhasesDefied      = errors.New("at least one phase required")
	ErrCycleDetected       = errors.New("cycle in phase dependencies")
	ErrModelUnavailable    = errors.New("model unavailable")
	ErrProviderUnreachable = errors.New("provider unreachable")
	ErrContextTooLarge     = errors.New("context exceeds max tokens")
	ErrPhaseNotFound       = errors.New("phase not found")
	ErrDependencyNotFound  = errors.New("dependency phase not found")
)

// ErrorCode categorizes errors for handling and reporting.
type ErrorCode string

const (
	CodeValidation    ErrorCode = "VALIDATION"
	CodeNotFound      ErrorCode = "NOT_FOUND"
	CodeProvider      ErrorCode = "PROVIDER"
	CodeExecution     ErrorCode = "EXECUTION"
	CodeConfiguration ErrorCode = "CONFIG"
)

// SkillrunnerError wraps errors with additional context for debugging and handling.
type SkillrunnerError struct {
	Code    ErrorCode
	Message string
	Cause   error
	Context map[string]interface{}
}

// Error returns a formatted error string including the code, message, and cause if present.
func (e *SkillrunnerError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause error for use with errors.Is and errors.As.
func (e *SkillrunnerError) Unwrap() error {
	return e.Cause
}

// NewError creates a new SkillrunnerError with the given code, message, and optional cause.
func NewError(code ErrorCode, message string, cause error) *SkillrunnerError {
	return &SkillrunnerError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// WithContext adds a key-value pair to the error's context and returns the error.
// This allows for method chaining when adding multiple context values.
func WithContext(err *SkillrunnerError, key string, value interface{}) *SkillrunnerError {
	if err.Context == nil {
		err.Context = make(map[string]interface{})
	}
	err.Context[key] = value
	return err
}

// Is reports whether err matches target using errors.Is semantics.
// This is a convenience wrapper around the standard library's errors.Is.
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As finds the first error in err's chain that matches target and sets target to that error value.
// This is a convenience wrapper around the standard library's errors.As.
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// New creates a new SkillrunnerError with the given domain and message.
// This is a convenience function for creating domain errors.
func New(domain, message string) *SkillrunnerError {
	return &SkillrunnerError{
		Code:    CodeValidation,
		Message: fmt.Sprintf("[%s] %s", domain, message),
		Context: make(map[string]interface{}),
	}
}
