package chat

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Conversation represents a chat conversation aggregate.
type Conversation struct {
	ID        string
	Messages  []*Message
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewConversation creates a new conversation.
func NewConversation() *Conversation {
	now := time.Now()
	return &Conversation{
		ID:        uuid.New().String(),
		Messages:  make([]*Message, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddMessage adds a message to the conversation.
func (c *Conversation) AddMessage(message *Message) error {
	if err := message.Validate(); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}

	c.Messages = append(c.Messages, message)
	c.UpdatedAt = time.Now()
	return nil
}

// AddSystemMessage adds a system message to the conversation.
func (c *Conversation) AddSystemMessage(content string) error {
	return c.AddMessage(NewSystemMessage(content))
}

// AddUserMessage adds a user message to the conversation.
func (c *Conversation) AddUserMessage(content string) error {
	return c.AddMessage(NewUserMessage(content))
}

// AddAssistantMessage adds an assistant message to the conversation.
func (c *Conversation) AddAssistantMessage(content string) error {
	return c.AddMessage(NewAssistantMessage(content))
}

// GetMessages returns all messages in the conversation.
func (c *Conversation) GetMessages() []*Message {
	// Return a copy to prevent external modifications
	messages := make([]*Message, len(c.Messages))
	copy(messages, c.Messages)
	return messages
}

// GetLastMessage returns the last message in the conversation, or nil if empty.
func (c *Conversation) GetLastMessage() *Message {
	if len(c.Messages) == 0 {
		return nil
	}
	return c.Messages[len(c.Messages)-1]
}

// GetMessagesByRole returns all messages with the specified role.
func (c *Conversation) GetMessagesByRole(role MessageRole) []*Message {
	result := make([]*Message, 0)
	for _, msg := range c.Messages {
		if msg.Role == role {
			result = append(result, msg)
		}
	}
	return result
}

// MessageCount returns the number of messages in the conversation.
func (c *Conversation) MessageCount() int {
	return len(c.Messages)
}

// Clear removes all messages from the conversation.
func (c *Conversation) Clear() {
	c.Messages = make([]*Message, 0)
	c.UpdatedAt = time.Now()
}

// Validate validates the conversation.
func (c *Conversation) Validate() error {
	if c.ID == "" {
		return fmt.Errorf("conversation ID cannot be empty")
	}

	for i, msg := range c.Messages {
		if err := msg.Validate(); err != nil {
			return fmt.Errorf("invalid message at index %d: %w", i, err)
		}
	}

	return nil
}
