package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jbctechsolutions/skillrunner/internal/config"
	"github.com/jbctechsolutions/skillrunner/internal/envelope"
	"github.com/jbctechsolutions/skillrunner/internal/importer"
	"github.com/jbctechsolutions/skillrunner/internal/models"
	"github.com/jbctechsolutions/skillrunner/internal/orchestration"
	"github.com/jbctechsolutions/skillrunner/internal/types"
)

const Version = "0.1.0"

// DefaultModel is used when no model is specified. Empty means auto-detect.
const DefaultModel = ""

var builtinSkills = map[string]types.SkillConfig{}

var builtinSkillOrder = []string{}

// Skillrunner is the main orchestration engine
type Skillrunner struct {
	workspacePath string
	skillsCache   map[string]*types.SkillConfig
	models        *models.Orchestrator
	modelPolicy   models.ResolvePolicy
	importer      *importer.Importer
	skillsDir     string // optional override for orchestrated skills directory
}

// NewSkillrunner creates a new Skillrunner instance with the provided model policy.
func NewSkillrunner(workspace string, policy models.ResolvePolicy) *Skillrunner {
	if workspace == "" {
		workspace = "."
	}
	if policy == "" {
		policy = models.ResolvePolicyAuto
	}

	orchestrator := models.NewOrchestrator()

	// Try to load config for provider URLs
	var cfg *config.Config
	if cfgManager, err := config.NewManager(""); err == nil {
		cfg = cfgManager.Get()
	}

	registerDefaultProviders(orchestrator, cfg)

	// Initialize importer for marketplace skills
	imp, err := importer.NewImporter()
	if err != nil {
		// Log warning but continue - we can still use builtin skills
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize skill importer: %v\n", err)
	}

	return &Skillrunner{
		workspacePath: workspace,
		skillsCache:   make(map[string]*types.SkillConfig),
		models:        orchestrator,
		modelPolicy:   policy,
		importer:      imp,
	}
}

// GetImporter returns the importer instance (for accessing imported skills)
func (sr *Skillrunner) GetImporter() *importer.Importer {
	return sr.importer
}

// Run executes a skill with the given request
func (sr *Skillrunner) Run(skillName, request, modelOverride string) (*types.Envelope, error) {
	// Load skill configuration
	skillConfig, err := sr.loadSkill(skillName)
	if err != nil {
		return nil, err
	}

	// Create envelope builder
	builder := envelope.NewBuilder(skillName, request)

	// Add metadata
	builder.AddMetadata("workspace", sr.workspacePath)

	// Generate steps
	steps, err := sr.generateSteps(skillConfig, request, modelOverride)
	if err != nil {
		return nil, err
	}

	// Add steps to envelope
	for _, step := range steps {
		builder.AddStep(step)
	}

	// Build and return envelope
	return builder.Build(), nil
}

// ListSkills returns all available skills (builtin + orchestrated + imported)
func (sr *Skillrunner) ListSkills() ([]types.SkillConfig, error) {
	skills := make([]types.SkillConfig, 0, len(builtinSkills)+20)

	// Add builtin skills first
	for _, name := range builtinSkillOrder {
		if skill, ok := builtinSkills[name]; ok {
			skills = append(skills, skill)
		}
	}

	// Add orchestrated skills from ~/.skillrunner/skills/
	skillsDir := sr.getOrchestratedSkillsDir()
	loader := orchestration.NewSkillLoader(skillsDir)
	orchestratedSkillNames, err := loader.ListSkills()
	if err == nil {
		for _, skillName := range orchestratedSkillNames {
			skill, err := loader.LoadSkill(skillName)
			if err == nil {
				skills = append(skills, types.SkillConfig{
					Name:         skill.Name,
					Version:      skill.Version,
					Description:  skill.Description,
					DefaultModel: DefaultModel,
				})
			}
		}
	}

	// Add imported skills from marketplace
	if sr.importer != nil {
		importedSkills := sr.importer.ListSkills()
		for _, imported := range importedSkills {
			skill := types.SkillConfig{
				Name:         imported.Name,
				Version:      imported.Version,
				Description:  imported.Description,
				DefaultModel: DefaultModel, // Auto-detect available model
			}
			skills = append(skills, skill)
		}
	}

	return skills, nil
}

// getOrchestratedSkillsDir returns the directory where orchestrated skills are stored
func (sr *Skillrunner) getOrchestratedSkillsDir() string {
	// Check for programmatic override (used in tests)
	if sr.skillsDir != "" {
		return sr.skillsDir
	}

	// Check if path is overridden in config
	cfgManager, err := config.NewManager("")
	if err == nil {
		cfg := cfgManager.Get()
		if cfg.Paths != nil && cfg.Paths.SkillrunnerDir != "" {
			return filepath.Join(cfg.Paths.SkillrunnerDir, "skills")
		}
	}

	// Default to ~/.skillrunner/skills
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./skills"
	}
	return filepath.Join(homeDir, ".skillrunner", "skills")
}

// GetStatus returns system status information
func (sr *Skillrunner) GetStatus() (*types.SystemStatus, error) {
	skills, _ := sr.ListSkills()

	registeredModels, err := sr.models.ListModels(context.Background())
	if err != nil {
		return nil, fmt.Errorf("list configured models: %w", err)
	}

	configuredModels := make([]string, 0, len(registeredModels))
	for _, model := range registeredModels {
		status := model.Name
		if !model.Available {
			status = fmt.Sprintf("%s (unavailable)", status)
		}
		configuredModels = append(configuredModels, status)
	}

	return &types.SystemStatus{
		Version:          Version,
		SkillCount:       len(skills),
		Workspace:        sr.workspacePath,
		Ready:            true,
		ConfiguredModels: configuredModels,
	}, nil
}

// loadSkill loads a skill configuration (builtin or imported)
func (sr *Skillrunner) loadSkill(skillName string) (*types.SkillConfig, error) {
	// Check cache
	if cached, ok := sr.skillsCache[skillName]; ok {
		return cached, nil
	}

	// Try orchestrated skills from ~/.skillrunner/skills/
	skillsDir := sr.getOrchestratedSkillsDir()
	loader := orchestration.NewSkillLoader(skillsDir)
	orchestratedSkill, err := loader.LoadSkill(skillName)
	if err == nil {
		skill := &types.SkillConfig{
			Name:         orchestratedSkill.Name,
			Version:      orchestratedSkill.Version,
			Description:  orchestratedSkill.Description,
			DefaultModel: DefaultModel,
		}
		sr.skillsCache[skillName] = skill
		return skill, nil
	}

	// Try imported skills from marketplace
	if sr.importer != nil {
		imported, err := sr.importer.GetSkill(skillName)
		if err == nil {
			skill := &types.SkillConfig{
				Name:         imported.Name,
				Version:      imported.Version,
				Description:  imported.Description,
				DefaultModel: DefaultModel, // Auto-detect available model
			}
			sr.skillsCache[skillName] = skill
			return skill, nil
		}
	}

	// Skill not found
	allSkills, _ := sr.ListSkills()
	skillNames := make([]string, len(allSkills))
	for i, s := range allSkills {
		skillNames[i] = s.Name
	}
	return nil, fmt.Errorf("skill '%s' not found. Available skills: %v", skillName, skillNames)
}

// generateSteps generates workflow steps for a skill
func (sr *Skillrunner) generateSteps(skillConfig *types.SkillConfig, request, modelOverride string) ([]types.Step, error) {
	requestedModel := modelOverride
	if requestedModel == "" {
		requestedModel = skillConfig.DefaultModel
	}

	// If still empty, try to auto-detect first available model
	if requestedModel == "" {
		availableModels, err := sr.models.ListModels(context.Background())
		if err == nil && len(availableModels) > 0 {
			// Prefer local models first
			for _, m := range availableModels {
				if m.Available && m.Provider.Type == models.ProviderTypeLocal {
					requestedModel = m.Name
					break
				}
			}
			// Fall back to any available model
			if requestedModel == "" {
				for _, m := range availableModels {
					if m.Available {
						requestedModel = m.Name
						break
					}
				}
			}
		}
		if requestedModel == "" {
			return nil, fmt.Errorf("no model specified and no available models found")
		}
	}

	preferred := models.ProviderType("")
	if sr.modelPolicy == models.ResolvePolicyAuto || sr.modelPolicy == models.ResolvePolicyLocalFirst {
		preferred = models.ProviderTypeLocal
	}

	// Build fallback list - only include non-empty models
	var fallbackModels []string
	if skillConfig.DefaultModel != "" && skillConfig.DefaultModel != requestedModel {
		fallbackModels = append(fallbackModels, skillConfig.DefaultModel)
	}

	resolvedModel, err := sr.models.ResolveModel(context.Background(), models.ResolveRequest{
		Model:             requestedModel,
		FallbackModels:    fallbackModels,
		PreferredProvider: preferred,
		Policy:            sr.modelPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("resolve model %q: %w", requestedModel, err)
	}

	modelName := resolvedModel.Name

	// Default single-step workflow
	return []types.Step{
		{
			Intent:  types.IntentPlan,
			Model:   modelName,
			Prompt:  fmt.Sprintf("Process request: %s", request),
			Context: []types.ContextItem{},
			FileOps: []types.FileOperation{},
			Metadata: map[string]interface{}{
				"model_provider":           resolvedModel.Provider.Name,
				"model_provider_tier":      tierToString(resolvedModel.Tier),
				"model_cost_per_1k_tokens": resolvedModel.CostPer1KTokens,
			},
		},
	}, nil
}

func registerDefaultProviders(orchestrator *models.Orchestrator, cfg *config.Config) {
	// Use the new unified provider factory which supports both new and legacy config formats
	if cfg == nil {
		cfg = &config.Config{}
	}

	providers, err := models.NewProvidersFromConfig(cfg)
	if err != nil {
		// Log error but don't fail - fall back to defaults
		// In production, you might want to use a proper logger here
		return
	}

	// Register all providers from config
	for _, provider := range providers {
		orchestrator.RegisterProvider(provider)
	}
}

func tierToString(tier models.AgentTier) string {
	switch tier {
	case models.AgentTierFast:
		return "fast"
	case models.AgentTierBalanced:
		return "balanced"
	case models.AgentTierPowerful:
		return "powerful"
	default:
		return "unknown"
	}
}

// RegisterTestProvider registers a provider for testing purposes.
// This method is exposed for testing and should not be used in production code.
func (e *Skillrunner) RegisterTestProvider(provider models.ModelProvider) {
	e.models.RegisterProvider(provider)
}

// NewSkillrunnerForTesting creates a Skillrunner instance without registering default providers.
// This is useful for tests that need to control which providers are available.
// The skillsDir parameter allows overriding the skills directory (use "" for default behavior).
func NewSkillrunnerForTesting(workspace string, policy models.ResolvePolicy, skillsDir string) *Skillrunner {
	if workspace == "" {
		workspace = "."
	}
	if policy == "" {
		policy = models.ResolvePolicyAuto
	}

	orchestrator := models.NewOrchestrator()
	// Don't register default providers - tests will add their own

	return &Skillrunner{
		workspacePath: workspace,
		skillsCache:   make(map[string]*types.SkillConfig),
		models:        orchestrator,
		modelPolicy:   policy,
		importer:      nil, // No importer for tests
		skillsDir:     skillsDir,
	}
}
