package config

import (
	"testing"
	"time"
)

func TestNewDefaultConfig(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg == nil {
		t.Fatal("NewDefaultConfig returned nil")
	}

	// Check Ollama defaults
	if cfg.Providers.Ollama.URL != DefaultOllamaURL {
		t.Errorf("expected Ollama URL %q, got %q", DefaultOllamaURL, cfg.Providers.Ollama.URL)
	}
	if !cfg.Providers.Ollama.Enabled {
		t.Error("expected Ollama to be enabled by default")
	}
	if cfg.Providers.Ollama.Timeout != DefaultTimeout {
		t.Errorf("expected Ollama timeout %v, got %v", DefaultTimeout, cfg.Providers.Ollama.Timeout)
	}

	// Check cloud providers are disabled by default
	if cfg.Providers.Anthropic.Enabled {
		t.Error("expected Anthropic to be disabled by default")
	}
	if cfg.Providers.OpenAI.Enabled {
		t.Error("expected OpenAI to be disabled by default")
	}
	if cfg.Providers.Groq.Enabled {
		t.Error("expected Groq to be disabled by default")
	}

	// Check logging defaults
	if cfg.Logging.Level != DefaultLogLevel {
		t.Errorf("expected log level %q, got %q", DefaultLogLevel, cfg.Logging.Level)
	}
	if cfg.Logging.Format != DefaultLogFormat {
		t.Errorf("expected log format %q, got %q", DefaultLogFormat, cfg.Logging.Format)
	}

	// Check routing defaults
	if cfg.Routing.DefaultProfile != DefaultRoutingProfile {
		t.Errorf("expected default profile %q, got %q", DefaultRoutingProfile, cfg.Routing.DefaultProfile)
	}

	// Check skills defaults
	if cfg.Skills.Directory != DefaultSkillsDirectory {
		t.Errorf("expected skills directory %q, got %q", DefaultSkillsDirectory, cfg.Skills.Directory)
	}
}

func TestConfig_Validate_DefaultIsValid(t *testing.T) {
	cfg := NewDefaultConfig()
	if err := cfg.Validate(); err != nil {
		t.Errorf("default config should be valid, got error: %v", err)
	}
}

func TestLoggingConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  LoggingConfig
		wantErr bool
	}{
		{
			name:    "valid debug level",
			config:  LoggingConfig{Level: "debug", Format: "json"},
			wantErr: false,
		},
		{
			name:    "valid info level",
			config:  LoggingConfig{Level: "info", Format: "text"},
			wantErr: false,
		},
		{
			name:    "valid warn level",
			config:  LoggingConfig{Level: "warn", Format: "json"},
			wantErr: false,
		},
		{
			name:    "valid error level",
			config:  LoggingConfig{Level: "error", Format: "text"},
			wantErr: false,
		},
		{
			name:    "invalid log level",
			config:  LoggingConfig{Level: "invalid", Format: "json"},
			wantErr: true,
		},
		{
			name:    "invalid log format",
			config:  LoggingConfig{Level: "info", Format: "invalid"},
			wantErr: true,
		},
		{
			name:    "empty values are valid",
			config:  LoggingConfig{Level: "", Format: ""},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOllamaConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  OllamaConfig
		wantErr bool
	}{
		{
			name:    "valid enabled config",
			config:  OllamaConfig{URL: "http://localhost:11434", Enabled: true, Timeout: 30 * time.Second},
			wantErr: false,
		},
		{
			name:    "disabled without URL is valid",
			config:  OllamaConfig{URL: "", Enabled: false, Timeout: 30 * time.Second},
			wantErr: false,
		},
		{
			name:    "enabled without URL is invalid",
			config:  OllamaConfig{URL: "", Enabled: true, Timeout: 30 * time.Second},
			wantErr: true,
		},
		{
			name:    "negative timeout is invalid",
			config:  OllamaConfig{URL: "http://localhost:11434", Enabled: true, Timeout: -1 * time.Second},
			wantErr: true,
		},
		{
			name:    "zero timeout is valid",
			config:  OllamaConfig{URL: "http://localhost:11434", Enabled: true, Timeout: 0},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCloudConfig_Validate(t *testing.T) {
	tests := []struct {
		name         string
		config       CloudConfig
		providerName string
		wantErr      bool
	}{
		{
			name:         "valid enabled config",
			config:       CloudConfig{APIKeyEncrypted: "encrypted_key", Enabled: true, Timeout: 30 * time.Second},
			providerName: "anthropic",
			wantErr:      false,
		},
		{
			name:         "disabled without API key is valid",
			config:       CloudConfig{APIKeyEncrypted: "", Enabled: false, Timeout: 30 * time.Second},
			providerName: "openai",
			wantErr:      false,
		},
		{
			name:         "enabled without API key is invalid",
			config:       CloudConfig{APIKeyEncrypted: "", Enabled: true, Timeout: 30 * time.Second},
			providerName: "groq",
			wantErr:      true,
		},
		{
			name:         "negative timeout is invalid",
			config:       CloudConfig{APIKeyEncrypted: "key", Enabled: true, Timeout: -1 * time.Second},
			providerName: "anthropic",
			wantErr:      true,
		},
		// BaseURL validation tests
		{
			name:         "valid https base_url",
			config:       CloudConfig{APIKeyEncrypted: "key", BaseURL: "https://api.example.com/v1", Enabled: true, Timeout: 30 * time.Second},
			providerName: "anthropic",
			wantErr:      false,
		},
		{
			name:         "valid http base_url for local proxy",
			config:       CloudConfig{APIKeyEncrypted: "key", BaseURL: "http://localhost:8080", Enabled: true, Timeout: 30 * time.Second},
			providerName: "anthropic",
			wantErr:      false,
		},
		{
			name:         "empty base_url is valid (uses default)",
			config:       CloudConfig{APIKeyEncrypted: "key", BaseURL: "", Enabled: true, Timeout: 30 * time.Second},
			providerName: "anthropic",
			wantErr:      false,
		},
		{
			name:         "invalid base_url scheme",
			config:       CloudConfig{APIKeyEncrypted: "key", BaseURL: "ftp://example.com", Enabled: true, Timeout: 30 * time.Second},
			providerName: "anthropic",
			wantErr:      true,
		},
		{
			name:         "base_url without scheme is invalid",
			config:       CloudConfig{APIKeyEncrypted: "key", BaseURL: "example.com/api", Enabled: true, Timeout: 30 * time.Second},
			providerName: "anthropic",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate(tt.providerName)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRoutingConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  RoutingConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  RoutingConfig{DefaultProfile: "default"},
			wantErr: false,
		},
		{
			name:    "empty default profile is invalid",
			config:  RoutingConfig{DefaultProfile: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSkillsConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  SkillsConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  SkillsConfig{Directory: "/path/to/skills"},
			wantErr: false,
		},
		{
			name:    "empty directory is invalid",
			config:  SkillsConfig{Directory: ""},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_Validate_MultipleErrors(t *testing.T) {
	cfg := &Config{
		Providers: ProviderConfigs{
			Ollama: OllamaConfig{
				URL:     "", // Invalid: empty URL when enabled
				Enabled: true,
				Timeout: -1 * time.Second, // Invalid: negative timeout
			},
			Anthropic: CloudConfig{
				APIKeyEncrypted: "", // Invalid: empty key when enabled
				Enabled:         true,
				Timeout:         30 * time.Second,
			},
		},
		Routing: RoutingConfig{
			DefaultProfile: "", // Invalid: empty profile
		},
		Logging: LoggingConfig{
			Level:  "invalid", // Invalid: not a valid level
			Format: "text",
		},
		Skills: SkillsConfig{
			Directory: "", // Invalid: empty directory
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("expected validation error, got nil")
	}
}
