package errors

import (
	"errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"ErrSkillNotFound", ErrSkillNotFound, "skill not found"},
		{"ErrSkillIDRequired", ErrSkillIDRequired, "skill ID required"},
		{"ErrSkillNameRequired", ErrSkillNameRequired, "skill name required"},
		{"ErrNoPhasesDefied", ErrNoPhasesDefied, "at least one phase required"},
		{"ErrCycleDetected", ErrCycleDetected, "cycle in phase dependencies"},
		{"ErrModelUnavailable", ErrModelUnavailable, "model unavailable"},
		{"ErrProviderUnreachable", ErrProviderUnreachable, "provider unreachable"},
		{"ErrContextTooLarge", ErrContextTooLarge, "context exceeds max tokens"},
		{"ErrPhaseNotFound", ErrPhaseNotFound, "phase not found"},
		{"ErrDependencyNotFound", ErrDependencyNotFound, "dependency phase not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSkillrunnerError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *SkillrunnerError
		want string
	}{
		{
			name: "with cause",
			err:  NewError(CodeValidation, "invalid skill", ErrSkillNameRequired),
			want: "[VALIDATION] invalid skill: skill name required",
		},
		{
			name: "without cause",
			err:  NewError(CodeNotFound, "resource not found", nil),
			want: "[NOT_FOUND] resource not found",
		},
		{
			name: "provider error",
			err:  NewError(CodeProvider, "API call failed", ErrProviderUnreachable),
			want: "[PROVIDER] API call failed: provider unreachable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSkillrunnerError_Unwrap(t *testing.T) {
	cause := ErrSkillNotFound
	err := NewError(CodeNotFound, "skill lookup failed", cause)

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestSkillrunnerError_Unwrap_Nil(t *testing.T) {
	err := NewError(CodeValidation, "validation failed", nil)

	unwrapped := err.Unwrap()
	if unwrapped != nil {
		t.Errorf("Unwrap() = %v, want nil", unwrapped)
	}
}

func TestNewError(t *testing.T) {
	err := NewError(CodeExecution, "execution failed", ErrPhaseNotFound)

	if err.Code != CodeExecution {
		t.Errorf("Code = %v, want %v", err.Code, CodeExecution)
	}
	if err.Message != "execution failed" {
		t.Errorf("Message = %v, want %v", err.Message, "execution failed")
	}
	if err.Cause != ErrPhaseNotFound {
		t.Errorf("Cause = %v, want %v", err.Cause, ErrPhaseNotFound)
	}
	if err.Context == nil {
		t.Error("Context should be initialized, got nil")
	}
}

func TestWithContext(t *testing.T) {
	err := NewError(CodeValidation, "validation failed", nil)
	err = WithContext(err, "field", "name")
	err = WithContext(err, "value", "")

	if err.Context["field"] != "name" {
		t.Errorf("Context[field] = %v, want %v", err.Context["field"], "name")
	}
	if err.Context["value"] != "" {
		t.Errorf("Context[value] = %v, want empty string", err.Context["value"])
	}
}

func TestWithContext_NilContext(t *testing.T) {
	// Create error with nil context to test initialization
	err := &SkillrunnerError{
		Code:    CodeValidation,
		Message: "test",
		Context: nil,
	}

	err = WithContext(err, "key", "value")

	if err.Context == nil {
		t.Error("Context should be initialized after WithContext")
	}
	if err.Context["key"] != "value" {
		t.Errorf("Context[key] = %v, want %v", err.Context["key"], "value")
	}
}

func TestErrorsIs(t *testing.T) {
	wrapped := NewError(CodeNotFound, "skill not found", ErrSkillNotFound)

	if !errors.Is(wrapped, ErrSkillNotFound) {
		t.Error("errors.Is should return true for wrapped sentinel error")
	}

	if errors.Is(wrapped, ErrPhaseNotFound) {
		t.Error("errors.Is should return false for different sentinel error")
	}
}

func TestErrorsAs(t *testing.T) {
	wrapped := NewError(CodeProvider, "API error", ErrProviderUnreachable)

	var skillErr *SkillrunnerError
	if !errors.As(wrapped, &skillErr) {
		t.Error("errors.As should return true for SkillrunnerError")
	}

	if skillErr.Code != CodeProvider {
		t.Errorf("Code = %v, want %v", skillErr.Code, CodeProvider)
	}
}

func TestIs_Wrapper(t *testing.T) {
	err := NewError(CodeNotFound, "not found", ErrSkillNotFound)

	if !Is(err, ErrSkillNotFound) {
		t.Error("Is should return true for wrapped error")
	}
	if Is(err, ErrPhaseNotFound) {
		t.Error("Is should return false for non-matching error")
	}
}

func TestAs_Wrapper(t *testing.T) {
	err := NewError(CodeExecution, "failed", nil)

	var target *SkillrunnerError
	if !As(err, &target) {
		t.Error("As should return true and set target")
	}
	if target.Code != CodeExecution {
		t.Errorf("target.Code = %v, want %v", target.Code, CodeExecution)
	}
}

func TestErrorCodes(t *testing.T) {
	tests := []struct {
		code ErrorCode
		want string
	}{
		{CodeValidation, "VALIDATION"},
		{CodeNotFound, "NOT_FOUND"},
		{CodeProvider, "PROVIDER"},
		{CodeExecution, "EXECUTION"},
		{CodeConfiguration, "CONFIG"},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			if string(tt.code) != tt.want {
				t.Errorf("got %q, want %q", string(tt.code), tt.want)
			}
		})
	}
}

func TestChainedContext(t *testing.T) {
	err := NewError(CodeValidation, "validation failed", ErrSkillNameRequired)
	err = WithContext(err, "field", "name")
	err = WithContext(err, "provided_value", "")
	err = WithContext(err, "skill_id", "abc-123")

	if len(err.Context) != 3 {
		t.Errorf("Context length = %d, want 3", len(err.Context))
	}
	if err.Context["field"] != "name" {
		t.Errorf("Context[field] = %v, want name", err.Context["field"])
	}
	if err.Context["provided_value"] != "" {
		t.Errorf("Context[provided_value] = %v, want empty string", err.Context["provided_value"])
	}
	if err.Context["skill_id"] != "abc-123" {
		t.Errorf("Context[skill_id] = %v, want abc-123", err.Context["skill_id"])
	}
}
