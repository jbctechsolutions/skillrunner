// Package chat provides application services for chat-based interactions.
package chat

import (
	"context"
	"fmt"

	adapterProvider "github.com/jbctechsolutions/skillrunner/internal/adapters/provider"
	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	appProvider "github.com/jbctechsolutions/skillrunner/internal/application/provider"
	"github.com/jbctechsolutions/skillrunner/internal/domain/chat"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// Service provides chat-based interaction services.
type Service struct {
	router   *appProvider.Router
	registry *adapterProvider.Registry
}

// NewService creates a new chat service.
func NewService(router *appProvider.Router, registry *adapterProvider.Registry) (*Service, error) {
	if router == nil {
		return nil, fmt.Errorf("router cannot be nil")
	}
	if registry == nil {
		return nil, fmt.Errorf("registry cannot be nil")
	}

	return &Service{
		router:   router,
		registry: registry,
	}, nil
}

// AskRequest represents a request to ask a one-shot question.
type AskRequest struct {
	Question      string
	Profile       string
	ModelOverride string
	SystemPrompt  string
	MaxTokens     int
	Temperature   float32
}

// AskResponse represents the response from a one-shot question.
type AskResponse struct {
	Answer       string
	ModelUsed    string
	Provider     string
	InputTokens  int
	OutputTokens int
	IsFallback   bool
}

// Ask executes a one-shot question using the configured router and providers.
func (s *Service) Ask(ctx context.Context, req *AskRequest) (*AskResponse, error) {
	if err := s.validateAskRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Determine which model to use
	var modelID, providerName string
	var isFallback bool

	if req.ModelOverride != "" {
		// User specified a model override
		modelID = req.ModelOverride
		provider, err := s.registry.FindByModel(ctx, modelID)
		if err != nil {
			return nil, fmt.Errorf("model override not found: %s", modelID)
		}
		providerName = provider.Info().Name
		isFallback = false
	} else {
		// Use router to select model based on profile
		selection, err := s.router.SelectModel(ctx, req.Profile)
		if err != nil {
			return nil, fmt.Errorf("could not select model: %w", err)
		}
		modelID = selection.ModelID
		providerName = selection.ProviderName
		isFallback = selection.IsFallback
	}

	// Get the provider
	provider := s.registry.Get(providerName)
	if provider == nil {
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}

	// Build the completion request
	completionReq := s.buildCompletionRequest(req, modelID)

	// Execute the completion
	response, err := provider.Complete(ctx, completionReq)
	if err != nil {
		return nil, fmt.Errorf("completion failed: %w", err)
	}

	return &AskResponse{
		Answer:       response.Content,
		ModelUsed:    response.ModelUsed,
		Provider:     providerName,
		InputTokens:  response.InputTokens,
		OutputTokens: response.OutputTokens,
		IsFallback:   isFallback,
	}, nil
}

// AskWithConversation executes a question within the context of a conversation.
func (s *Service) AskWithConversation(ctx context.Context, req *AskRequest, conversation *chat.Conversation) (*AskResponse, error) {
	if conversation == nil {
		return nil, fmt.Errorf("conversation cannot be nil")
	}

	// Add user message to conversation
	if err := conversation.AddUserMessage(req.Question); err != nil {
		return nil, fmt.Errorf("could not add user message: %w", err)
	}

	// Get response
	response, err := s.Ask(ctx, req)
	if err != nil {
		return nil, err
	}

	// Add assistant response to conversation
	if err := conversation.AddAssistantMessage(response.Answer); err != nil {
		return nil, fmt.Errorf("could not add assistant message: %w", err)
	}

	return response, nil
}

// StreamCallback is a function called with each chunk of streamed text.
type StreamCallback func(chunk string) error

// StreamRequest represents a request to stream a response.
type StreamRequest struct {
	AskRequest
	Callback StreamCallback
}

// Stream executes a question and streams the response using the callback.
func (s *Service) Stream(ctx context.Context, req *StreamRequest) (*AskResponse, error) {
	if req.Callback == nil {
		return nil, fmt.Errorf("callback cannot be nil")
	}

	if err := s.validateAskRequest(&req.AskRequest); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Determine which model to use (same as Ask)
	var modelID, providerName string
	var isFallback bool

	if req.ModelOverride != "" {
		modelID = req.ModelOverride
		provider, err := s.registry.FindByModel(ctx, modelID)
		if err != nil {
			return nil, fmt.Errorf("model override not found: %s", modelID)
		}
		providerName = provider.Info().Name
		isFallback = false
	} else {
		selection, err := s.router.SelectModel(ctx, req.Profile)
		if err != nil {
			return nil, fmt.Errorf("could not select model: %w", err)
		}
		modelID = selection.ModelID
		providerName = selection.ProviderName
		isFallback = selection.IsFallback
	}

	// Get the provider
	provider := s.registry.Get(providerName)
	if provider == nil {
		return nil, fmt.Errorf("provider not found: %s", providerName)
	}

	// Build the completion request
	completionReq := s.buildCompletionRequest(&req.AskRequest, modelID)

	// Execute streaming
	response, err := provider.Stream(ctx, completionReq, func(chunk string) error {
		return req.Callback(chunk)
	})
	if err != nil {
		return nil, fmt.Errorf("streaming failed: %w", err)
	}

	return &AskResponse{
		Answer:       response.Content,
		ModelUsed:    response.ModelUsed,
		Provider:     providerName,
		InputTokens:  response.InputTokens,
		OutputTokens: response.OutputTokens,
		IsFallback:   isFallback,
	}, nil
}

// validateAskRequest validates an ask request.
func (s *Service) validateAskRequest(req *AskRequest) error {
	if req.Question == "" {
		return fmt.Errorf("question cannot be empty")
	}

	if req.Profile == "" {
		req.Profile = skill.ProfileBalanced // Default to balanced
	}

	// Validate profile
	validProfiles := []string{skill.ProfileCheap, skill.ProfileBalanced, skill.ProfilePremium}
	isValid := false
	for _, valid := range validProfiles {
		if req.Profile == valid {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid profile: %s", req.Profile)
	}

	// Set defaults
	if req.MaxTokens == 0 {
		req.MaxTokens = 2048
	}

	if req.Temperature == 0 {
		req.Temperature = 0.7
	}

	return nil
}

// buildCompletionRequest builds a provider completion request from an ask request.
func (s *Service) buildCompletionRequest(req *AskRequest, modelID string) ports.CompletionRequest {
	messages := []ports.Message{
		{
			Role:    "user",
			Content: req.Question,
		},
	}

	return ports.CompletionRequest{
		ModelID:      modelID,
		Messages:     messages,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		SystemPrompt: req.SystemPrompt,
	}
}
