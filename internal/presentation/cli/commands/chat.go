// Package commands implements the CLI commands for skillrunner.
package commands

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"

	"github.com/jbctechsolutions/skillrunner/internal/application/chat"
	appProvider "github.com/jbctechsolutions/skillrunner/internal/application/provider"
	domainChat "github.com/jbctechsolutions/skillrunner/internal/domain/chat"
	"github.com/jbctechsolutions/skillrunner/internal/domain/session"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// chatFlags holds the flags for the chat command.
type chatFlags struct {
	Model        string
	Profile      string
	SessionName  string
	SystemPrompt string
}

var chatOpts chatFlags

// NewChatCmd creates the chat command for interactive REPL mode.
func NewChatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Interactive chat REPL",
		Long: `Start an interactive chat session with the AI.

The chat command provides a REPL (Read-Eval-Print Loop) interface for
continuous conversation with the AI. Each session maintains context
across multiple exchanges.

Special commands:
  /exit, /quit    - Exit the chat session
  /clear          - Clear conversation history
  /help           - Show help message
  /session        - Show current session info
  /model <name>   - Switch to a different model
  /profile <name> - Switch to a different profile

Examples:
  # Start a chat session with default settings
  sr chat

  # Start with a specific model
  sr chat --model claude-3-opus

  # Start with a named session
  sr chat --session my-project

  # Start with a custom system prompt
  sr chat --system "You are a helpful coding assistant"`,
		Args: cobra.NoArgs,
		RunE: runChat,
	}

	// Define flags
	cmd.Flags().StringVarP(&chatOpts.Model, "model", "m", "",
		"initial model selection (e.g., claude-3-opus, gpt-4)")
	cmd.Flags().StringVarP(&chatOpts.Profile, "profile", "p", skill.ProfileBalanced,
		fmt.Sprintf("routing profile: %s, %s, %s", skill.ProfileCheap, skill.ProfileBalanced, skill.ProfilePremium))
	cmd.Flags().StringVarP(&chatOpts.SessionName, "session", "s", "",
		"session name (auto-generated if not provided)")
	cmd.Flags().StringVar(&chatOpts.SystemPrompt, "system", "",
		"custom system prompt")

	return cmd
}

// runChat executes the interactive chat REPL.
func runChat(cmd *cobra.Command, args []string) error {
	formatter := GetFormatter()
	ctx := context.Background()

	// Initialize chat service
	chatService, err := initChatService()
	if err != nil {
		return fmt.Errorf("could not initialize chat service: %w", err)
	}

	// Generate or use provided session name
	sessionName := chatOpts.SessionName
	if sessionName == "" {
		sessionName = session.GenerateSessionName()
	}

	// Create conversation
	conversation := domainChat.NewConversation()

	// Add system prompt if provided
	if chatOpts.SystemPrompt != "" {
		if err := conversation.AddSystemMessage(chatOpts.SystemPrompt); err != nil {
			return fmt.Errorf("could not add system prompt: %w", err)
		}
	}

	// Print welcome message
	formatter.Header(fmt.Sprintf("Chat Session: %s", sessionName))
	formatter.Item("Profile", chatOpts.Profile)
	if chatOpts.Model != "" {
		formatter.Item("Model", chatOpts.Model)
	}
	formatter.Println("")
	formatter.Info("Type your message and press Enter. Type /help for commands.")
	formatter.Println("")

	// Create readline instance
	rl, err := readline.New("> ")
	if err != nil {
		return fmt.Errorf("could not create readline: %w", err)
	}
	defer rl.Close()

	// REPL loop
	currentProfile := chatOpts.Profile
	currentModel := chatOpts.Model

	for {
		line, err := rl.Readline()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Handle special commands
		if strings.HasPrefix(line, "/") {
			shouldExit, err := handleChatCommand(line, conversation, &currentProfile, &currentModel, formatter, sessionName)
			if err != nil {
				formatter.Error("Command error: %s", err.Error())
				continue
			}
			if shouldExit {
				break
			}
			continue
		}

		// Build ask request
		askReq := &chat.AskRequest{
			Question:      line,
			Profile:       currentProfile,
			ModelOverride: currentModel,
			SystemPrompt:  chatOpts.SystemPrompt,
			MaxTokens:     2048,
			Temperature:   0.7,
		}

		// Get response using conversation context
		response, err := chatService.AskWithConversation(ctx, askReq, conversation)
		if err != nil {
			formatter.Error("Error: %s", err.Error())
			continue
		}

		// Print response
		formatter.Success("\nAssistant (%s):", response.ModelUsed)
		formatter.Println(response.Answer)
		formatter.Println("")
	}

	formatter.Info("Chat session ended. Goodbye!")
	return nil
}

// handleChatCommand handles special chat commands.
// Returns (shouldExit, error).
func handleChatCommand(cmd string, conversation *domainChat.Conversation, currentProfile, currentModel *string, formatter interface{}, sessionName string) (bool, error) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false, nil
	}

	command := strings.ToLower(parts[0])

	// Type assertion to get formatter methods
	f, ok := formatter.(interface {
		Info(format string, args ...interface{})
		Success(format string, args ...interface{})
		Item(key, value string)
		Println(args ...interface{})
		Header(title string)
	})
	if !ok {
		return false, fmt.Errorf("invalid formatter type")
	}

	switch command {
	case "/exit", "/quit":
		return true, nil

	case "/clear":
		conversation.Clear()
		f.Success("Conversation history cleared")
		return false, nil

	case "/help":
		f.Header("Chat Commands")
		f.Item("/exit, /quit", "Exit the chat session")
		f.Item("/clear", "Clear conversation history")
		f.Item("/help", "Show this help message")
		f.Item("/session", "Show current session info")
		f.Item("/model <name>", "Switch to a different model")
		f.Item("/profile <name>", "Switch to a different profile")
		f.Println("")
		return false, nil

	case "/session":
		f.Header("Session Info")
		f.Item("Name", sessionName)
		f.Item("Profile", *currentProfile)
		if *currentModel != "" {
			f.Item("Model Override", *currentModel)
		}
		f.Item("Messages", fmt.Sprintf("%d", conversation.MessageCount()))
		f.Println("")
		return false, nil

	case "/model":
		if len(parts) < 2 {
			return false, fmt.Errorf("usage: /model <model-name>")
		}
		*currentModel = parts[1]
		f.Success("Switched to model: %s", *currentModel)
		return false, nil

	case "/profile":
		if len(parts) < 2 {
			return false, fmt.Errorf("usage: /profile <profile-name>")
		}
		newProfile := parts[1]
		validProfiles := []string{skill.ProfileCheap, skill.ProfileBalanced, skill.ProfilePremium}
		isValid := false
		for _, valid := range validProfiles {
			if newProfile == valid {
				isValid = true
				break
			}
		}
		if !isValid {
			return false, fmt.Errorf("invalid profile: %s (must be one of: %s)", newProfile, strings.Join(validProfiles, ", "))
		}
		*currentProfile = newProfile
		f.Success("Switched to profile: %s", *currentProfile)
		return false, nil

	default:
		return false, fmt.Errorf("unknown command: %s (type /help for help)", command)
	}
}

// initChatService initializes the chat service with provider registry and router.
func initChatService() (*chat.Service, error) {
	appCtx := GetAppContext()
	if appCtx == nil {
		return nil, fmt.Errorf("app context not initialized")
	}

	// Get the container which has the already-initialized provider registry
	container := GetContainer()
	if container == nil {
		return nil, fmt.Errorf("application container not initialized")
	}

	// Get the provider registry from the container
	// This registry is already populated with providers based on configuration
	registry := container.ProviderRegistry()
	if registry == nil {
		return nil, fmt.Errorf("provider registry not available")
	}

	// Check if any providers are registered
	if registry.Count() == 0 {
		return nil, fmt.Errorf("no providers configured - please configure providers in ~/.skillrunner/config.yaml")
	}

	// Create routing configuration from user's app config
	routingCfg := container.RoutingConfiguration()

	// Create router with the populated registry
	router, err := appProvider.NewRouter(routingCfg, registry)
	if err != nil {
		return nil, fmt.Errorf("could not create router: %w", err)
	}

	// Create chat service with the properly initialized registry
	chatService, err := chat.NewService(router, registry)
	if err != nil {
		return nil, fmt.Errorf("could not create chat service: %w", err)
	}

	return chatService, nil
}
