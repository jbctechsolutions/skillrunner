package errors

import (
	"errors"
	"testing"
)

func TestSkillrunnerError_Error(t *testing.T) {
	err := New(ErrorCodeSkillNotFound, "skill not found")
	if err.Error() == "" {
		t.Error("Error() should return non-empty string")
	}

	wrapped := Wrap(errors.New("underlying error"), ErrorCodeModelUnavailable, "model unavailable")
	if wrapped.Error() == "" {
		t.Error("Wrapped error should return non-empty string")
	}
}

func TestSkillrunnerError_WithContext(t *testing.T) {
	err := New(ErrorCodeSkillNotFound, "skill not found").
		WithContext("skill", "test-skill")

	if err.Context["skill"] != "test-skill" {
		t.Error("Context should be set")
	}
}

func TestSkillrunnerError_WithError(t *testing.T) {
	underlying := errors.New("underlying error")
	err := New(ErrorCodeSkillNotFound, "skill not found").
		WithError(underlying)

	if err.Unwrap() != underlying {
		t.Error("Unwrap should return underlying error")
	}
}

func TestSkillrunnerError_IsRetryable(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected bool
	}{
		{ErrorCodeModelTimeout, true},
		{ErrorCodeNetworkError, true},
		{ErrorCodeModelHealthCheckFailed, true},
		{ErrorCodeSkillNotFound, false},
		{ErrorCodeExecutionFailed, false},
	}

	for _, tt := range tests {
		err := New(tt.code, "test error")
		if err.IsRetryable() != tt.expected {
			t.Errorf("IsRetryable() for %s = %v, want %v", tt.code, err.IsRetryable(), tt.expected)
		}
	}
}

func TestSkillrunnerError_UserMessage(t *testing.T) {
	err := New(ErrorCodeSkillNotFound, "skill not found").
		WithContext("skill", "test-skill")

	msg := err.UserMessage()
	if msg == "" {
		t.Error("UserMessage should return non-empty string")
	}
}

func TestWrap(t *testing.T) {
	underlying := errors.New("underlying error")
	wrapped := Wrap(underlying, ErrorCodeModelUnavailable, "model unavailable")

	if wrapped.Err != underlying {
		t.Error("Wrap should preserve underlying error")
	}
	if wrapped.Code != ErrorCodeModelUnavailable {
		t.Error("Wrap should set error code")
	}
}
