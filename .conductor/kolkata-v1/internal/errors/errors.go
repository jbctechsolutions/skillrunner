package errors

import "fmt"

// ErrorCode represents a specific error type
type ErrorCode string

const (
	// Skill errors
	ErrorCodeSkillNotFound   ErrorCode = "SKILL_NOT_FOUND"
	ErrorCodeSkillInvalid    ErrorCode = "SKILL_INVALID"
	ErrorCodeSkillLoadFailed ErrorCode = "SKILL_LOAD_FAILED"

	// Model errors
	ErrorCodeModelUnavailable       ErrorCode = "MODEL_UNAVAILABLE"
	ErrorCodeModelHealthCheckFailed ErrorCode = "MODEL_HEALTH_CHECK_FAILED"
	ErrorCodeModelTimeout           ErrorCode = "MODEL_TIMEOUT"

	// Execution errors
	ErrorCodeExecutionFailed ErrorCode = "EXECUTION_FAILED"
	ErrorCodePhaseFailed     ErrorCode = "PHASE_FAILED"
	ErrorCodeContextTooLarge ErrorCode = "CONTEXT_TOO_LARGE"

	// Configuration errors
	ErrorCodeConfigInvalid  ErrorCode = "CONFIG_INVALID"
	ErrorCodeConfigNotFound ErrorCode = "CONFIG_NOT_FOUND"

	// Network errors
	ErrorCodeNetworkError ErrorCode = "NETWORK_ERROR"
	ErrorCodeAPIError     ErrorCode = "API_ERROR"

	// Validation errors
	ErrorCodeValidationFailed ErrorCode = "VALIDATION_FAILED"
	ErrorCodeInvalidInput     ErrorCode = "INVALID_INPUT"
)

// SkillrunnerError represents a structured error with code and context
type SkillrunnerError struct {
	Code    ErrorCode
	Message string
	Context map[string]interface{}
	Err     error
}

// Error implements the error interface
func (e *SkillrunnerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying error
func (e *SkillrunnerError) Unwrap() error {
	return e.Err
}

// New creates a new SkillrunnerError
func New(code ErrorCode, message string) *SkillrunnerError {
	return &SkillrunnerError{
		Code:    code,
		Message: message,
		Context: make(map[string]interface{}),
	}
}

// WithContext adds context to the error
func (e *SkillrunnerError) WithContext(key string, value interface{}) *SkillrunnerError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithError wraps an underlying error
func (e *SkillrunnerError) WithError(err error) *SkillrunnerError {
	e.Err = err
	return e
}

// IsRetryable returns whether the error is retryable
func (e *SkillrunnerError) IsRetryable() bool {
	switch e.Code {
	case ErrorCodeModelTimeout, ErrorCodeNetworkError, ErrorCodeModelHealthCheckFailed:
		return true
	default:
		return false
	}
}

// UserMessage returns a user-friendly error message
func (e *SkillrunnerError) UserMessage() string {
	switch e.Code {
	case ErrorCodeSkillNotFound:
		return fmt.Sprintf("Skill '%s' not found. Use 'skill list' to see available skills.", e.Context["skill"])
	case ErrorCodeModelUnavailable:
		return fmt.Sprintf("Model '%s' is not available. Check if Ollama is running or API keys are configured.", e.Context["model"])
	case ErrorCodeContextTooLarge:
		return fmt.Sprintf("Context is too large (%d tokens). The system will automatically optimize it.", e.Context["tokens"])
	case ErrorCodeExecutionFailed:
		return fmt.Sprintf("Execution failed: %s. Check the logs for more details.", e.Message)
	case ErrorCodeConfigInvalid:
		return fmt.Sprintf("Configuration is invalid: %s. Run 'skill init' to set up configuration.", e.Message)
	default:
		return e.Message
	}
}

// Wrap wraps an error with a SkillrunnerError
func Wrap(err error, code ErrorCode, message string) *SkillrunnerError {
	return &SkillrunnerError{
		Code:    code,
		Message: message,
		Err:     err,
		Context: make(map[string]interface{}),
	}
}
