// Package config provides configuration structs and utilities for the skillrunner application.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Config represents the root configuration for the skillrunner application.
type Config struct {
	Providers ProviderConfigs `yaml:"providers"`
	Routing   RoutingConfig   `yaml:"routing"`
	Logging   LoggingConfig   `yaml:"logging"`
	Skills    SkillsConfig    `yaml:"skills"`
	Cache     CacheConfig     `yaml:"cache"`
}

// ProviderConfigs holds configuration for all supported LLM providers.
type ProviderConfigs struct {
	Ollama    OllamaConfig `yaml:"ollama"`
	Anthropic CloudConfig  `yaml:"anthropic"`
	OpenAI    CloudConfig  `yaml:"openai"`
	Groq      CloudConfig  `yaml:"groq"`
}

// OllamaConfig holds configuration for the Ollama local LLM provider.
type OllamaConfig struct {
	URL     string        `yaml:"url"`
	Enabled bool          `yaml:"enabled"`
	Timeout time.Duration `yaml:"timeout"`
}

// CloudConfig holds configuration for cloud-based LLM providers.
type CloudConfig struct {
	APIKeyEncrypted string        `yaml:"api_key_encrypted"`
	BaseURL         string        `yaml:"base_url,omitempty"` // Optional custom endpoint (e.g., for proxies)
	Enabled         bool          `yaml:"enabled"`
	Timeout         time.Duration `yaml:"timeout"`
}

// RoutingConfig holds configuration for model routing.
type RoutingConfig struct {
	DefaultProfile string `yaml:"default_profile"`
}

// LoggingConfig holds configuration for application logging.
type LoggingConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Format string `yaml:"format"` // json, text
}

// SkillsConfig holds configuration for skill management.
type SkillsConfig struct {
	Directory string `yaml:"directory"`
}

// CacheConfig holds configuration for response caching.
type CacheConfig struct {
	Enabled       bool          `yaml:"enabled"`
	DefaultTTL    time.Duration `yaml:"default_ttl"`
	MaxMemorySize int64         `yaml:"max_memory_size"` // Maximum in-memory cache size in bytes
	MaxDiskSize   int64         `yaml:"max_disk_size"`   // Maximum SQLite cache size in bytes
	CleanupPeriod time.Duration `yaml:"cleanup_period"`  // How often to run cleanup
}

// BatchConfig holds configuration for batch processing.
type BatchConfig struct {
	Enabled       bool          `yaml:"enabled"`
	MaxBatchSize  int           `yaml:"max_batch_size"`  // Maximum requests per batch
	MaxWaitTime   time.Duration `yaml:"max_wait_time"`   // Maximum time to wait for batch to fill
	PerProviderBatching bool     `yaml:"per_provider_batching"` // Whether to batch per provider
}

// Default configuration values.
const (
	DefaultOllamaURL       = "http://localhost:11434"
	DefaultTimeout         = 30 * time.Second
	DefaultLogLevel        = "info"
	DefaultLogFormat       = "text"
	DefaultSkillsDirectory = "~/.skillrunner/skills"
	DefaultRoutingProfile  = "default"

	// Cache defaults
	DefaultCacheEnabled       = true
	DefaultCacheTTL           = 24 * time.Hour     // 24 hours default TTL
	DefaultCacheMaxMemorySize = 100 * 1024 * 1024  // 100 MB in-memory cache
	DefaultCacheMaxDiskSize   = 1024 * 1024 * 1024 // 1 GB disk cache
	DefaultCacheCleanupPeriod = 1 * time.Hour      // Cleanup every hour

	// Batch defaults
	DefaultBatchEnabled     = true
	DefaultBatchMaxSize     = 10
	DefaultBatchMaxWaitTime = 100 * time.Millisecond
)

// Valid log levels.
var validLogLevels = map[string]bool{
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
}

// Valid log formats.
var validLogFormats = map[string]bool{
	"json": true,
	"text": true,
}

// NewDefaultConfig creates a new Config with sensible default values.
func NewDefaultConfig() *Config {
	return &Config{
		Providers: ProviderConfigs{
			Ollama: OllamaConfig{
				URL:     DefaultOllamaURL,
				Enabled: true,
				Timeout: DefaultTimeout,
			},
			Anthropic: CloudConfig{
				Enabled: false,
				Timeout: DefaultTimeout,
			},
			OpenAI: CloudConfig{
				Enabled: false,
				Timeout: DefaultTimeout,
			},
			Groq: CloudConfig{
				Enabled: false,
				Timeout: DefaultTimeout,
			},
		},
		Routing: RoutingConfig{
			DefaultProfile: DefaultRoutingProfile,
		},
		Logging: LoggingConfig{
			Level:  DefaultLogLevel,
			Format: DefaultLogFormat,
		},
		Skills: SkillsConfig{
			Directory: DefaultSkillsDirectory,
		},
		Cache: CacheConfig{
			Enabled:       DefaultCacheEnabled,
			DefaultTTL:    DefaultCacheTTL,
			MaxMemorySize: DefaultCacheMaxMemorySize,
			MaxDiskSize:   DefaultCacheMaxDiskSize,
			CleanupPeriod: DefaultCacheCleanupPeriod,
		},
	}
}

// Validate checks if the configuration is valid and returns an error if not.
func (c *Config) Validate() error {
	var errs []error

	// Validate logging config
	if err := c.Logging.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("logging: %w", err))
	}

	// Validate providers config
	if err := c.Providers.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("providers: %w", err))
	}

	// Validate routing config
	if err := c.Routing.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("routing: %w", err))
	}

	// Validate skills config
	if err := c.Skills.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("skills: %w", err))
	}

	// Validate cache config
	if err := c.Cache.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("cache: %w", err))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Validate checks if the LoggingConfig is valid.
func (l *LoggingConfig) Validate() error {
	var errs []error

	if l.Level != "" && !validLogLevels[l.Level] {
		errs = append(errs, fmt.Errorf("invalid log level %q: must be one of debug, info, warn, error", l.Level))
	}

	if l.Format != "" && !validLogFormats[l.Format] {
		errs = append(errs, fmt.Errorf("invalid log format %q: must be one of json, text", l.Format))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Validate checks if the ProviderConfigs is valid.
func (p *ProviderConfigs) Validate() error {
	var errs []error

	if err := p.Ollama.Validate(); err != nil {
		errs = append(errs, fmt.Errorf("ollama: %w", err))
	}

	if err := p.Anthropic.Validate("anthropic"); err != nil {
		errs = append(errs, err)
	}

	if err := p.OpenAI.Validate("openai"); err != nil {
		errs = append(errs, err)
	}

	if err := p.Groq.Validate("groq"); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Validate checks if the OllamaConfig is valid.
func (o *OllamaConfig) Validate() error {
	var errs []error

	if o.Enabled && o.URL == "" {
		errs = append(errs, errors.New("url is required when enabled"))
	}

	if o.Timeout < 0 {
		errs = append(errs, errors.New("timeout must be non-negative"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Validate checks if the CloudConfig is valid.
func (c *CloudConfig) Validate(providerName string) error {
	var errs []error

	if c.Enabled && c.APIKeyEncrypted == "" {
		errs = append(errs, fmt.Errorf("%s: api_key_encrypted is required when enabled", providerName))
	}

	if c.Timeout < 0 {
		errs = append(errs, fmt.Errorf("%s: timeout must be non-negative", providerName))
	}

	// Validate base_url if provided
	if c.BaseURL != "" {
		parsedURL, err := url.Parse(c.BaseURL)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: invalid base_url: %w", providerName, err))
		} else if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			errs = append(errs, fmt.Errorf("%s: base_url must use http or https scheme", providerName))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Validate checks if the RoutingConfig is valid.
func (r *RoutingConfig) Validate() error {
	if r.DefaultProfile == "" {
		return errors.New("default_profile is required")
	}
	return nil
}

// Validate checks if the SkillsConfig is valid.
func (s *SkillsConfig) Validate() error {
	if s.Directory == "" {
		return errors.New("directory is required")
	}
	return nil
}

// Validate checks if the CacheConfig is valid.
func (c *CacheConfig) Validate() error {
	var errs []error

	if c.Enabled {
		if c.DefaultTTL <= 0 {
			errs = append(errs, errors.New("default_ttl must be positive when cache is enabled"))
		}
		if c.MaxMemorySize < 0 {
			errs = append(errs, errors.New("max_memory_size must be non-negative"))
		}
		if c.MaxDiskSize < 0 {
			errs = append(errs, errors.New("max_disk_size must be non-negative"))
		}
		if c.CleanupPeriod <= 0 {
			errs = append(errs, errors.New("cleanup_period must be positive when cache is enabled"))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// Validate checks if the BatchConfig is valid.
func (b *BatchConfig) Validate() error {
	var errs []error

	if b.Enabled {
		if b.MaxBatchSize <= 0 {
			errs = append(errs, errors.New("max_batch_size must be positive when batching is enabled"))
		}
		if b.MaxWaitTime <= 0 {
			errs = append(errs, errors.New("max_wait_time must be positive when batching is enabled"))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}
