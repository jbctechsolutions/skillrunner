// Package chat provides domain entities for chat-based interactions.
package chat

import (
	"fmt"
	"time"
)

// MessageRole represents the role of a message sender.
type MessageRole string

const (
	// RoleSystem represents a system message
	RoleSystem MessageRole = "system"
	// RoleUser represents a user message
	RoleUser MessageRole = "user"
	// RoleAssistant represents an assistant message
	RoleAssistant MessageRole = "assistant"
)

// Message represents a single chat message.
type Message struct {
	Role      MessageRole
	Content   string
	Timestamp time.Time
}

// NewMessage creates a new message with the current timestamp.
func NewMessage(role MessageRole, content string) *Message {
	return &Message{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// NewSystemMessage creates a new system message.
func NewSystemMessage(content string) *Message {
	return NewMessage(RoleSystem, content)
}

// NewUserMessage creates a new user message.
func NewUserMessage(content string) *Message {
	return NewMessage(RoleUser, content)
}

// NewAssistantMessage creates a new assistant message.
func NewAssistantMessage(content string) *Message {
	return NewMessage(RoleAssistant, content)
}

// Validate validates the message.
func (m *Message) Validate() error {
	if m.Content == "" {
		return fmt.Errorf("message content cannot be empty")
	}

	if !m.Role.IsValid() {
		return fmt.Errorf("invalid message role: %s", m.Role)
	}

	return nil
}

// IsValid checks if the message role is valid.
func (r MessageRole) IsValid() bool {
	switch r {
	case RoleSystem, RoleUser, RoleAssistant:
		return true
	default:
		return false
	}
}

// String returns the string representation of the role.
func (r MessageRole) String() string {
	return string(r)
}
