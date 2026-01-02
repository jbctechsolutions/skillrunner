package chat

import (
	"testing"
	"time"
)

func TestNewMessage(t *testing.T) {
	content := "test message"
	msg := NewMessage(RoleUser, content)

	if msg == nil {
		t.Fatal("NewMessage returned nil")
	}

	if msg.Role != RoleUser {
		t.Errorf("expected role %s, got %s", RoleUser, msg.Role)
	}

	if msg.Content != content {
		t.Errorf("expected content %q, got %q", content, msg.Content)
	}

	if msg.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
}

func TestNewSystemMessage(t *testing.T) {
	content := "system prompt"
	msg := NewSystemMessage(content)

	if msg.Role != RoleSystem {
		t.Errorf("expected role %s, got %s", RoleSystem, msg.Role)
	}

	if msg.Content != content {
		t.Errorf("expected content %q, got %q", content, msg.Content)
	}
}

func TestNewUserMessage(t *testing.T) {
	content := "user question"
	msg := NewUserMessage(content)

	if msg.Role != RoleUser {
		t.Errorf("expected role %s, got %s", RoleUser, msg.Role)
	}

	if msg.Content != content {
		t.Errorf("expected content %q, got %q", content, msg.Content)
	}
}

func TestNewAssistantMessage(t *testing.T) {
	content := "assistant response"
	msg := NewAssistantMessage(content)

	if msg.Role != RoleAssistant {
		t.Errorf("expected role %s, got %s", RoleAssistant, msg.Role)
	}

	if msg.Content != content {
		t.Errorf("expected content %q, got %q", content, msg.Content)
	}
}

func TestMessageValidate(t *testing.T) {
	tests := []struct {
		name      string
		message   *Message
		wantError bool
	}{
		{
			name:      "valid user message",
			message:   NewUserMessage("test"),
			wantError: false,
		},
		{
			name:      "valid system message",
			message:   NewSystemMessage("test"),
			wantError: false,
		},
		{
			name:      "valid assistant message",
			message:   NewAssistantMessage("test"),
			wantError: false,
		},
		{
			name: "empty content",
			message: &Message{
				Role:      RoleUser,
				Content:   "",
				Timestamp: time.Now(),
			},
			wantError: true,
		},
		{
			name: "invalid role",
			message: &Message{
				Role:      "invalid",
				Content:   "test",
				Timestamp: time.Now(),
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.message.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestMessageRoleIsValid(t *testing.T) {
	tests := []struct {
		role  MessageRole
		valid bool
	}{
		{RoleSystem, true},
		{RoleUser, true},
		{RoleAssistant, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.valid {
				t.Errorf("IsValid() = %v, want %v", got, tt.valid)
			}
		})
	}
}

func TestMessageRoleString(t *testing.T) {
	role := RoleUser
	if role.String() != "user" {
		t.Errorf("expected 'user', got %q", role.String())
	}
}
