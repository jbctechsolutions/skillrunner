package engine

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/jbctechsolutions/skillrunner/internal/models"
)

type runnerConfig struct {
	ModelPolicy string `json:"model_policy"`
}

// ResolveModelPolicy determines the model policy using the precedence:
// CLI flag > environment variable > config file > default.
func ResolveModelPolicy(flagValue string) (models.ResolvePolicy, error) {
	if flagValue != "" {
		return models.ParseResolvePolicy(flagValue)
	}

	if envValue := os.Getenv("SKILLRUNNER_MODEL_POLICY"); envValue != "" {
		return models.ParseResolvePolicy(envValue)
	}

	if cfgValue, err := loadPolicyFromConfig(); err != nil {
		return models.ResolvePolicyAuto, err
	} else if cfgValue != "" {
		return models.ParseResolvePolicy(cfgValue)
	}

	return models.ResolvePolicyAuto, nil
}

func loadPolicyFromConfig() (string, error) {
	path := os.Getenv("SKILLRUNNER_CONFIG")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", nil
		}
		path = filepath.Join(home, ".skillrunner", "config.json")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	var cfg runnerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", err
	}

	return cfg.ModelPolicy, nil
}
