package chat

import (
	"testing"
)

func TestNewConversation(t *testing.T) {
	conv := NewConversation()

	if conv == nil {
		t.Fatal("NewConversation returned nil")
	}

	if conv.ID == "" {
		t.Error("conversation ID should not be empty")
	}

	if conv.Messages == nil {
		t.Error("messages should not be nil")
	}

	if len(conv.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(conv.Messages))
	}

	if conv.CreatedAt.IsZero() {
		t.Error("created at should not be zero")
	}

	if conv.UpdatedAt.IsZero() {
		t.Error("updated at should not be zero")
	}
}

func TestConversationAddMessage(t *testing.T) {
	conv := NewConversation()
	msg := NewUserMessage("test")

	err := conv.AddMessage(msg)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(conv.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(conv.Messages))
	}

	// Test invalid message
	invalidMsg := &Message{
		Role:    "invalid",
		Content: "test",
	}
	err = conv.AddMessage(invalidMsg)
	if err == nil {
		t.Error("expected error for invalid message")
	}
}

func TestConversationAddSystemMessage(t *testing.T) {
	conv := NewConversation()

	err := conv.AddSystemMessage("system prompt")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(conv.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(conv.Messages))
	}

	if conv.Messages[0].Role != RoleSystem {
		t.Errorf("expected role %s, got %s", RoleSystem, conv.Messages[0].Role)
	}
}

func TestConversationAddUserMessage(t *testing.T) {
	conv := NewConversation()

	err := conv.AddUserMessage("user question")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(conv.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(conv.Messages))
	}

	if conv.Messages[0].Role != RoleUser {
		t.Errorf("expected role %s, got %s", RoleUser, conv.Messages[0].Role)
	}
}

func TestConversationAddAssistantMessage(t *testing.T) {
	conv := NewConversation()

	err := conv.AddAssistantMessage("assistant response")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(conv.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(conv.Messages))
	}

	if conv.Messages[0].Role != RoleAssistant {
		t.Errorf("expected role %s, got %s", RoleAssistant, conv.Messages[0].Role)
	}
}

func TestConversationGetMessages(t *testing.T) {
	conv := NewConversation()
	conv.AddUserMessage("test1")
	conv.AddAssistantMessage("test2")

	messages := conv.GetMessages()
	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}

	// Verify it's a copy by modifying the returned slice
	_ = append(messages, NewUserMessage("test3"))
	if len(conv.Messages) != 2 {
		t.Error("original messages should not be modified")
	}
}

func TestConversationGetLastMessage(t *testing.T) {
	conv := NewConversation()

	// Test empty conversation
	last := conv.GetLastMessage()
	if last != nil {
		t.Error("expected nil for empty conversation")
	}

	// Add messages
	conv.AddUserMessage("first")
	conv.AddAssistantMessage("second")

	last = conv.GetLastMessage()
	if last == nil {
		t.Fatal("expected message, got nil")
	}

	if last.Content != "second" {
		t.Errorf("expected 'second', got %q", last.Content)
	}
}

func TestConversationGetMessagesByRole(t *testing.T) {
	conv := NewConversation()
	conv.AddSystemMessage("system")
	conv.AddUserMessage("user1")
	conv.AddAssistantMessage("assistant1")
	conv.AddUserMessage("user2")
	conv.AddAssistantMessage("assistant2")

	userMessages := conv.GetMessagesByRole(RoleUser)
	if len(userMessages) != 2 {
		t.Errorf("expected 2 user messages, got %d", len(userMessages))
	}

	assistantMessages := conv.GetMessagesByRole(RoleAssistant)
	if len(assistantMessages) != 2 {
		t.Errorf("expected 2 assistant messages, got %d", len(assistantMessages))
	}

	systemMessages := conv.GetMessagesByRole(RoleSystem)
	if len(systemMessages) != 1 {
		t.Errorf("expected 1 system message, got %d", len(systemMessages))
	}
}

func TestConversationMessageCount(t *testing.T) {
	conv := NewConversation()

	if conv.MessageCount() != 0 {
		t.Errorf("expected 0, got %d", conv.MessageCount())
	}

	conv.AddUserMessage("test1")
	conv.AddAssistantMessage("test2")

	if conv.MessageCount() != 2 {
		t.Errorf("expected 2, got %d", conv.MessageCount())
	}
}

func TestConversationClear(t *testing.T) {
	conv := NewConversation()
	conv.AddUserMessage("test1")
	conv.AddAssistantMessage("test2")

	conv.Clear()

	if len(conv.Messages) != 0 {
		t.Errorf("expected 0 messages after clear, got %d", len(conv.Messages))
	}
}

func TestConversationValidate(t *testing.T) {
	conv := NewConversation()
	conv.AddUserMessage("test")

	err := conv.Validate()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test empty ID
	conv.ID = ""
	err = conv.Validate()
	if err == nil {
		t.Error("expected error for empty ID")
	}

	// Test invalid message
	conv = NewConversation()
	conv.Messages = append(conv.Messages, &Message{
		Role:    "invalid",
		Content: "test",
	})
	err = conv.Validate()
	if err == nil {
		t.Error("expected error for invalid message")
	}
}
