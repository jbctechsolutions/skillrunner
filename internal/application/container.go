// Package application provides application-level services and dependency injection.
package application

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jbctechsolutions/skillrunner/internal/adapters/backend"
	"github.com/jbctechsolutions/skillrunner/internal/adapters/cache"
	adapterMCP "github.com/jbctechsolutions/skillrunner/internal/adapters/mcp"
	adapterProvider "github.com/jbctechsolutions/skillrunner/internal/adapters/provider"
	"github.com/jbctechsolutions/skillrunner/internal/adapters/sync/sqlite"
	"github.com/jbctechsolutions/skillrunner/internal/application/observability"
	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	appProvider "github.com/jbctechsolutions/skillrunner/internal/application/provider"
	"github.com/jbctechsolutions/skillrunner/internal/application/session"
	appSkills "github.com/jbctechsolutions/skillrunner/internal/application/skills"
	"github.com/jbctechsolutions/skillrunner/internal/application/workflow"
	"github.com/jbctechsolutions/skillrunner/internal/domain/provider"
	domainSession "github.com/jbctechsolutions/skillrunner/internal/domain/session"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/config"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/logging"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/skills"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/storage"
	"github.com/jbctechsolutions/skillrunner/internal/infrastructure/tracing"
)

// Container holds all application dependencies and provides a central
// point for dependency injection. It manages the lifecycle of services
// and ensures proper initialization order.
type Container struct {
	// Configuration
	config  *config.Config
	verbose bool // Override log level to info when true

	// Database connection
	dbConn *sqlite.Connection
	db     *sql.DB

	// Repositories
	sessionRepo            ports.SessionStateStoragePort
	workspaceRepo          ports.WorkspaceStateStoragePort
	checkpointRepo         ports.CheckpointStateStoragePort
	workflowCheckpointRepo ports.WorkflowCheckpointPort
	contextRepo            ports.ContextItemStoragePort
	rulesRepo              ports.RuleStoragePort

	// Application services
	sessionManager    *session.Manager
	workflowExecutor  workflow.Executor
	streamingExecutor workflow.StreamingExecutor
	skillLoader       *skills.Loader
	skillRegistry     *appSkills.Registry
	skillWatchService *appSkills.WatchService

	// Registries
	providerRegistry    *adapterProvider.Registry
	providerInitializer *appProvider.Initializer
	backendRegistry     *backend.Registry

	// MCP (Model Context Protocol)
	mcpRegistry *adapterMCP.Registry

	// Wave 10: Cache
	memoryCache    *cache.MemoryCache
	sqliteCache    *cache.SQLiteCache
	compositeCache *cache.CompositeCache
	responseCache  *cache.ResponseCache

	// Wave 11: Observability
	logger               *logging.Logger
	tracer               *tracing.Tracer
	metricsRepo          ports.MetricsStoragePort
	costCalculator       *provider.CostCalculator
	observabilityService *observability.Service

	// Machine ID for session tracking
	machineID string
}

// NewContainer creates a new dependency injection container with all services
// initialized based on the provided configuration.
func NewContainer(cfg *config.Config, verbose bool) (*Container, error) {
	if cfg == nil {
		cfg = config.NewDefaultConfig()
	}

	c := &Container{
		config:  cfg,
		verbose: verbose,
	}

	// Generate or retrieve machine ID
	c.machineID = getMachineID()

	// Initialize database connection
	if err := c.initDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	// Initialize repositories
	c.initRepositories()

	// Initialize registries (includes provider encryption setup)
	if err := c.initRegistries(); err != nil {
		_ = c.Close() // Clean up on error
		return nil, fmt.Errorf("failed to initialize registries: %w", err)
	}

	// Initialize MCP (Model Context Protocol)
	if err := c.initMCP(); err != nil {
		_ = c.Close() // Clean up on error
		return nil, fmt.Errorf("failed to initialize MCP: %w", err)
	}

	// Wave 11: Initialize observability
	if err := c.initObservability(); err != nil {
		_ = c.Close() // Clean up on error
		return nil, fmt.Errorf("failed to initialize observability: %w", err)
	}

	// Initialize application services
	if err := c.initServices(); err != nil {
		_ = c.Close() // Clean up on error
		return nil, fmt.Errorf("failed to initialize services: %w", err)
	}

	return c, nil
}

// initDatabase initializes the SQLite database connection.
func (c *Container) initDatabase() error {
	// Use default path: ~/.skillrunner/skillrunner.db
	conn, err := sqlite.NewConnection("")
	if err != nil {
		return fmt.Errorf("failed to create database connection: %w", err)
	}

	if err := conn.Open(); err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	db, err := conn.DB()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to get database handle: %w", err)
	}

	c.dbConn = conn
	c.db = db
	return nil
}

// initRepositories initializes all storage repositories.
func (c *Container) initRepositories() {
	c.sessionRepo = storage.NewSessionRepository(c.db)
	c.workspaceRepo = storage.NewWorkspaceRepository(c.db)
	c.checkpointRepo = storage.NewCheckpointRepository(c.db)
	c.workflowCheckpointRepo = storage.NewWorkflowCheckpointRepository(c.db)
	c.contextRepo = storage.NewContextItemRepository(c.db)
	c.rulesRepo = storage.NewRuleRepository(c.db)
}

// initRegistries initializes the provider and backend registries.
func (c *Container) initRegistries() error {
	c.providerRegistry = adapterProvider.NewRegistry()
	c.backendRegistry = backend.NewRegistry()

	// Initialize provider initializer with encryption support
	var err error
	c.providerInitializer, err = appProvider.NewInitializer(c.providerRegistry)
	if err != nil {
		return fmt.Errorf("failed to create provider initializer: %w", err)
	}

	// Register providers from config
	if err := c.providerInitializer.InitFromConfig(c.config); err != nil {
		// Log warning but don't fail - some providers may have initialized successfully
		// In production, this should be logged properly
		_ = err
	}

	return nil
}

// initMCP initializes the MCP (Model Context Protocol) subsystem.
func (c *Container) initMCP() error {
	manager := adapterMCP.NewServerManager()
	loader := adapterMCP.NewConfigLoader()
	c.mcpRegistry = adapterMCP.NewRegistry(manager, loader)

	// Load configs but don't start servers yet (they start on-demand)
	if err := c.mcpRegistry.LoadConfigs(context.Background()); err != nil {
		// Log warning but don't fail - MCP is optional
		// Configs may not exist if user hasn't configured any MCP servers
		_ = err
	}

	return nil
}

// initServices initializes application services.
func (c *Container) initServices() error {
	// Create session storage adapter that wraps the session repository
	sessionStorage := newSessionStorageAdapter(c.sessionRepo)

	// Create session manager
	c.sessionManager = session.NewManager(sessionStorage, c.backendRegistry, c.machineID)

	// Wave 10: Initialize cache if enabled
	if c.config.Cache.Enabled {
		c.initCache()
	}

	// Create workflow executors with a composite provider
	// For now, we use a placeholder that will be replaced when providers are configured
	executorConfig := workflow.DefaultExecutorConfig()
	c.workflowExecutor = workflow.NewExecutor(nil, executorConfig)
	c.streamingExecutor = workflow.NewStreamingExecutor(nil, executorConfig)

	// Create skill loader
	c.skillLoader = skills.NewLoader()

	// Create skill registry and load skills
	c.skillRegistry = appSkills.NewRegistry(c.skillLoader)

	// Load all skills (built-in and user)
	if err := c.skillRegistry.LoadAll(); err != nil {
		// Log warning but don't fail - skills are optional
		// The registry will still work, just with fewer skills
		_ = err // Ignore loading errors for now
	}

	// Initialize skill hot reload if enabled
	if c.config.Skills.HotReload {
		if err := c.initSkillWatcher(); err != nil {
			// Log warning but don't fail - hot reload is optional
			c.logger.Warn("failed to initialize skill watcher", "error", err)
		}
	}

	return nil
}

// initSkillWatcher initializes the skill watch service for hot reload.
func (c *Container) initSkillWatcher() error {
	// Resolve user skills directory
	userDir := c.config.Skills.Directory
	if userDir == "~/.skillrunner/skills" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		userDir = filepath.Join(homeDir, ".skillrunner", "skills")
	}

	// Get current working directory for project skills
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	projectDir := filepath.Join(cwd, ".skillrunner", "skills")

	cfg := appSkills.WatchServiceConfig{
		UserDir:          userDir,
		ProjectDir:       projectDir,
		DebounceDuration: c.config.Skills.DebounceDuration,
		OnReload: func(event appSkills.SkillReloadEvent) {
			if event.Error != nil {
				c.logger.Warn("skill reload failed",
					"path", event.FilePath,
					"error", event.Error,
				)
			} else {
				c.logger.Info("skill reloaded",
					"skill_id", event.SkillID,
					"event", event.EventType,
					"path", event.FilePath,
				)
			}
		},
	}

	watchService, err := appSkills.NewWatchService(cfg, c.skillRegistry, c.skillLoader, c.logger)
	if err != nil {
		return fmt.Errorf("failed to create watch service: %w", err)
	}

	c.skillWatchService = watchService
	return nil
}

// initCache initializes the caching subsystem.
func (c *Container) initCache() {
	// Create memory cache (L1 - fast, limited size)
	c.memoryCache = cache.NewMemoryCache(c.config.Cache.MaxMemorySize, c.config.Cache.CleanupPeriod)

	// Create SQLite cache (L2 - persistent, larger capacity)
	c.sqliteCache = cache.NewSQLiteCache(c.db, c.config.Cache.MaxDiskSize)

	// Create composite cache (combines L1 and L2)
	c.compositeCache = cache.NewCompositeCache(c.memoryCache, c.sqliteCache)

	// Create response cache (LLM-specific caching layer)
	c.responseCache = cache.NewResponseCache(c.compositeCache, c.config.Cache.DefaultTTL)
}

// initObservability initializes the observability subsystem (logging, tracing, metrics).
func (c *Container) initObservability() error {
	ctx := context.Background()

	// Initialize logger
	logLevel := logging.LevelInfo // default

	// Check verbose flag first - overrides config
	if c.verbose {
		logLevel = logging.LevelInfo
	} else {
		// Use config level if not verbose
		switch c.config.Logging.Level {
		case "debug":
			logLevel = logging.LevelDebug
		case "info":
			logLevel = logging.LevelInfo
		case "warn":
			logLevel = logging.LevelWarn
		case "error":
			logLevel = logging.LevelError
		}
	}

	logFormat := logging.FormatText
	if c.config.Logging.Format == "json" {
		logFormat = logging.FormatJSON
	}

	logCfg := logging.Config{
		Level:  logLevel,
		Format: logFormat,
	}
	c.logger = logging.New(logCfg)

	// Initialize tracer if enabled
	if c.config.Observability.Tracing.Enabled {
		tracingCfg := tracing.Config{
			Enabled:      true,
			ExporterType: tracing.ExporterType(c.config.Observability.Tracing.ExporterType),
			OTLPEndpoint: c.config.Observability.Tracing.OTLPEndpoint,
			ServiceName:  c.config.Observability.Tracing.ServiceName,
			Environment:  "production",
			SampleRate:   c.config.Observability.Tracing.SampleRate,
		}
		tracer, err := tracing.New(ctx, tracingCfg)
		if err != nil {
			return fmt.Errorf("failed to create tracer: %w", err)
		}
		c.tracer = tracer
	} else {
		// Create no-op tracer
		c.tracer = tracing.Default()
	}

	// Initialize metrics repository if enabled
	if c.config.Observability.Metrics.Enabled {
		c.metricsRepo = storage.NewMetricsRepository(c.db)
	}

	// Initialize cost calculator with default model pricing
	c.costCalculator = provider.NewCostCalculator()
	provider.PopulateCostCalculator(c.costCalculator)

	// Initialize observability service
	c.observabilityService = observability.NewService(observability.ServiceConfig{
		Logger:         c.logger,
		Tracer:         c.tracer,
		MetricsStorage: c.metricsRepo,
		CostCalculator: c.costCalculator,
	})

	return nil
}

// Close releases all resources held by the container.
func (c *Container) Close() error {
	ctx := context.Background()

	// Stop skill watcher
	if c.skillWatchService != nil {
		_ = c.skillWatchService.Stop()
	}

	// Shutdown MCP servers
	if c.mcpRegistry != nil {
		_ = c.mcpRegistry.Close(ctx)
	}

	// Wave 11: Shutdown tracer
	if c.tracer != nil {
		_ = c.tracer.Shutdown(ctx)
	}

	// Wave 10: Stop memory cache cleanup goroutine
	if c.memoryCache != nil {
		_ = c.memoryCache.Close()
	}

	if c.dbConn != nil {
		return c.dbConn.Close()
	}
	return nil
}

// StartSkillWatching starts the skill hot reload watcher.
// This should be called after the container is fully initialized.
func (c *Container) StartSkillWatching(ctx context.Context) error {
	if c.skillWatchService == nil {
		return nil // Hot reload not enabled
	}
	return c.skillWatchService.Start(ctx)
}

// StopSkillWatching stops the skill hot reload watcher.
func (c *Container) StopSkillWatching() error {
	if c.skillWatchService == nil {
		return nil
	}
	return c.skillWatchService.Stop()
}

// SkillWatchService returns the skill watch service.
// Returns nil if hot reload is not enabled.
func (c *Container) SkillWatchService() *appSkills.WatchService {
	return c.skillWatchService
}

// Config returns the application configuration.
func (c *Container) Config() *config.Config {
	return c.config
}

// DB returns the database connection.
func (c *Container) DB() *sql.DB {
	return c.db
}

// SessionRepository returns the session repository.
func (c *Container) SessionRepository() ports.SessionStateStoragePort {
	return c.sessionRepo
}

// WorkspaceRepository returns the workspace repository.
func (c *Container) WorkspaceRepository() ports.WorkspaceStateStoragePort {
	return c.workspaceRepo
}

// CheckpointRepository returns the checkpoint repository.
func (c *Container) CheckpointRepository() ports.CheckpointStateStoragePort {
	return c.checkpointRepo
}

// WorkflowCheckpointRepository returns the workflow checkpoint repository for crash recovery.
func (c *Container) WorkflowCheckpointRepository() ports.WorkflowCheckpointPort {
	return c.workflowCheckpointRepo
}

// ContextItemRepository returns the context item repository.
func (c *Container) ContextItemRepository() ports.ContextItemStoragePort {
	return c.contextRepo
}

// RulesRepository returns the rules repository.
func (c *Container) RulesRepository() ports.RuleStoragePort {
	return c.rulesRepo
}

// SessionManager returns the session manager.
func (c *Container) SessionManager() *session.Manager {
	return c.sessionManager
}

// WorkflowExecutor returns the workflow executor.
func (c *Container) WorkflowExecutor() workflow.Executor {
	return c.workflowExecutor
}

// StreamingExecutor returns the streaming workflow executor.
func (c *Container) StreamingExecutor() workflow.StreamingExecutor {
	return c.streamingExecutor
}

// NewWorkflowExecutor creates a new workflow executor with the specified provider.
func (c *Container) NewWorkflowExecutor(provider ports.ProviderPort) workflow.Executor {
	return workflow.NewExecutor(provider, workflow.DefaultExecutorConfig())
}

// NewStreamingExecutor creates a new streaming executor with the specified provider.
func (c *Container) NewStreamingExecutor(provider ports.ProviderPort) workflow.StreamingExecutor {
	return workflow.NewStreamingExecutor(provider, workflow.DefaultExecutorConfig())
}

// SkillLoader returns the skill loader.
func (c *Container) SkillLoader() *skills.Loader {
	return c.skillLoader
}

// SkillRegistry returns the skill registry.
func (c *Container) SkillRegistry() *appSkills.Registry {
	return c.skillRegistry
}

// ProviderRegistry returns the provider registry.
func (c *Container) ProviderRegistry() *adapterProvider.Registry {
	return c.providerRegistry
}

// RoutingConfiguration returns a RoutingConfiguration built from the user's config.
// User-defined profiles are merged over defaults, ensuring user settings take precedence.
func (c *Container) RoutingConfiguration() *config.RoutingConfiguration {
	return config.NewRoutingConfigurationFromConfig(c.config)
}

// ProviderInitializer returns the provider initializer for health checks and status.
func (c *Container) ProviderInitializer() *appProvider.Initializer {
	return c.providerInitializer
}

// BackendRegistry returns the backend registry.
func (c *Container) BackendRegistry() *backend.Registry {
	return c.backendRegistry
}

// MCPRegistry returns the MCP server registry for tool access.
// Returns nil if MCP is not initialized.
func (c *Container) MCPRegistry() *adapterMCP.Registry {
	return c.mcpRegistry
}

// MachineID returns the machine identifier.
func (c *Container) MachineID() string {
	return c.machineID
}

// ResponseCache returns the response cache for LLM caching.
// Returns nil if caching is not enabled.
func (c *Container) ResponseCache() *cache.ResponseCache {
	return c.responseCache
}

// MemoryCache returns the in-memory cache (L1 cache).
// Returns nil if caching is not enabled.
func (c *Container) MemoryCache() *cache.MemoryCache {
	return c.memoryCache
}

// CompositeCache returns the composite cache (L1 + L2).
// Returns nil if caching is not enabled.
func (c *Container) CompositeCache() *cache.CompositeCache {
	return c.compositeCache
}

// Logger returns the structured logger.
func (c *Container) Logger() *logging.Logger {
	return c.logger
}

// Tracer returns the OpenTelemetry tracer.
func (c *Container) Tracer() *tracing.Tracer {
	return c.tracer
}

// MetricsRepository returns the metrics storage repository.
// Returns nil if metrics are not enabled.
func (c *Container) MetricsRepository() ports.MetricsStoragePort {
	return c.metricsRepo
}

// CostCalculator returns the cost calculator for provider pricing.
func (c *Container) CostCalculator() *provider.CostCalculator {
	return c.costCalculator
}

// ObservabilityService returns the observability service for workflow execution.
func (c *Container) ObservabilityService() *observability.Service {
	return c.observabilityService
}

// getMachineID generates or retrieves a unique machine identifier.
func getMachineID() string {
	// Try to get hostname as a simple machine ID
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// sessionStorageAdapter adapts SessionStateStoragePort to SessionStoragePort
// required by the session.Manager.
type sessionStorageAdapter struct {
	repo ports.SessionStateStoragePort
}

// newSessionStorageAdapter creates a new session storage adapter.
func newSessionStorageAdapter(repo ports.SessionStateStoragePort) ports.SessionStoragePort {
	return &sessionStorageAdapter{repo: repo}
}

// SaveSession persists a session to storage.
func (a *sessionStorageAdapter) SaveSession(ctx context.Context, sess *domainSession.Session) error {
	return a.repo.Create(ctx, sess)
}

// GetSession retrieves a session by ID.
func (a *sessionStorageAdapter) GetSession(ctx context.Context, id string) (*domainSession.Session, error) {
	return a.repo.Get(ctx, id)
}

// GetActiveByWorkspace retrieves the active session for a workspace.
func (a *sessionStorageAdapter) GetActiveByWorkspace(ctx context.Context, workspaceID string) (*domainSession.Session, error) {
	return a.repo.GetActiveByWorkspace(ctx, workspaceID)
}

// ListSessions returns sessions matching the filter.
func (a *sessionStorageAdapter) ListSessions(ctx context.Context, filter domainSession.Filter) ([]*domainSession.Session, error) {
	return a.repo.List(ctx, filter)
}

// UpdateSession updates an existing session.
func (a *sessionStorageAdapter) UpdateSession(ctx context.Context, sess *domainSession.Session) error {
	return a.repo.Update(ctx, sess)
}

// DeleteSession removes a session from storage.
func (a *sessionStorageAdapter) DeleteSession(ctx context.Context, id string) error {
	return a.repo.Delete(ctx, id)
}
