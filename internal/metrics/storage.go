package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/config"
	"github.com/jbctechsolutions/skillrunner/internal/types"
)

// ExecutionRecord represents a single execution metric
type ExecutionRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	Skill        string    `json:"skill,omitempty"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	Cost         float64   `json:"cost"`
	DurationMs   int64     `json:"duration_ms"`
	Success      bool      `json:"success"`
	Error        string    `json:"error,omitempty"`
}

// Storage manages metrics persistence
type Storage struct {
	filePath string
	mu       sync.RWMutex
	records  []ExecutionRecord
}

// NewStorage creates a new metrics storage
func NewStorage() (*Storage, error) {
	// Check if path is overridden in config
	var skillrunnerDir string
	cfgManager, err := config.NewManager("")
	if err == nil {
		cfg := cfgManager.Get()
		if cfg.Paths != nil && cfg.Paths.SkillrunnerDir != "" {
			skillrunnerDir = cfg.Paths.SkillrunnerDir
		}
	}

	// Default to ~/.skillrunner if not overridden
	if skillrunnerDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home directory: %w", err)
		}
		skillrunnerDir = filepath.Join(homeDir, ".skillrunner")
	}

	metricsDir := filepath.Join(skillrunnerDir, "metrics")
	if err := os.MkdirAll(metricsDir, 0755); err != nil {
		return nil, fmt.Errorf("create metrics directory: %w", err)
	}

	filePath := filepath.Join(metricsDir, "executions.json")
	storage := &Storage{
		filePath: filePath,
		records:  []ExecutionRecord{},
	}

	// Load existing records
	if err := storage.load(); err != nil {
		// If file doesn't exist, that's okay - start fresh
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("load metrics: %w", err)
		}
	}

	return storage, nil
}

// RecordExecution records a new execution metric
func (s *Storage) RecordExecution(record ExecutionRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Set timestamp if not set
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	s.records = append(s.records, record)

	// Persist to disk
	return s.save()
}

// GetMetrics returns aggregated metrics for a time range
func (s *Storage) GetMetrics(since time.Time) *types.RouterMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var localCalls, cloudCalls, totalTokens int
	var totalCost, totalDuration float64
	var startTime time.Time = time.Now()

	for _, record := range s.records {
		if record.Timestamp.Before(since) {
			continue
		}

		if startTime.After(record.Timestamp) {
			startTime = record.Timestamp
		}

		// Determine if local or cloud based on model name
		if isLocalModel(record.Model) {
			localCalls++
		} else {
			cloudCalls++
		}

		totalTokens += record.InputTokens + record.OutputTokens
		totalCost += record.Cost
		totalDuration += float64(record.DurationMs) / 1000.0 // Convert ms to seconds
	}

	return &types.RouterMetrics{
		LocalCalls:    localCalls,
		CloudCalls:    cloudCalls,
		TotalTokens:   totalTokens,
		EstimatedCost: totalCost,
		ElapsedTime:   totalDuration,
		StartTime:     startTime,
	}
}

// GetRecords returns all execution records within a time range
func (s *Storage) GetRecords(since time.Time, skillFilter, modelFilter string) []ExecutionRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []ExecutionRecord
	for _, record := range s.records {
		if record.Timestamp.Before(since) {
			continue
		}
		if skillFilter != "" && record.Skill != skillFilter {
			continue
		}
		if modelFilter != "" && record.Model != modelFilter {
			continue
		}
		filtered = append(filtered, record)
	}

	return filtered
}

// load loads records from disk
func (s *Storage) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil
	}

	return json.Unmarshal(data, &s.records)
}

// save saves records to disk
func (s *Storage) save() error {
	// Keep only last 10000 records to prevent file from growing too large
	maxRecords := 10000
	if len(s.records) > maxRecords {
		s.records = s.records[len(s.records)-maxRecords:]
	}

	data, err := json.MarshalIndent(s.records, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal records: %w", err)
	}

	// Write to temp file first, then rename (atomic write)
	tmpPath := s.filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := os.Rename(tmpPath, s.filePath); err != nil {
		return fmt.Errorf("rename file: %w", err)
	}

	return nil
}

// isLocalModel checks if a model is local (Ollama)
func isLocalModel(model string) bool {
	return len(model) > 7 && model[:7] == "ollama/"
}
