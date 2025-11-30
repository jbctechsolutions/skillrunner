package envelope

import (
	"encoding/json"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/types"
)

// Builder provides a fluent interface for creating envelopes
type Builder struct {
	envelope *types.Envelope
}

// NewBuilder creates a new envelope builder
func NewBuilder(skill, request string) *Builder {
	return &Builder{
		envelope: &types.Envelope{
			Version: "1.0",
			Skill:   skill,
			Request: request,
			Steps:   []types.Step{},
			Metadata: map[string]interface{}{
				"created_at": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
}

// AddStep adds a step to the envelope
func (b *Builder) AddStep(step types.Step) *Builder {
	// Ensure non-nil slices
	if step.Context == nil {
		step.Context = []types.ContextItem{}
	}
	if step.FileOps == nil {
		step.FileOps = []types.FileOperation{}
	}
	if step.Metadata == nil {
		step.Metadata = map[string]interface{}{}
	}

	b.envelope.Steps = append(b.envelope.Steps, step)
	return b
}

// AddMetadata adds metadata to the envelope
func (b *Builder) AddMetadata(key string, value interface{}) *Builder {
	b.envelope.Metadata[key] = value
	return b
}

// Build returns the constructed envelope
func (b *Builder) Build() *types.Envelope {
	return b.envelope
}

// ToJSON converts the envelope to JSON
func (b *Builder) ToJSON(compact bool) (string, error) {
	var data []byte
	var err error

	if compact {
		data, err = json.Marshal(b.envelope)
	} else {
		data, err = json.MarshalIndent(b.envelope, "", "  ")
	}

	if err != nil {
		return "", err
	}

	return string(data), nil
}

// ToDict converts the envelope to a map
func (b *Builder) ToDict() map[string]interface{} {
	// Convert to JSON and back to get the map representation
	data, err := json.Marshal(b.envelope)
	if err != nil {
		// If marshaling fails, return empty map
		return map[string]interface{}{}
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		// If unmarshaling fails, return empty map
		return map[string]interface{}{}
	}
	return result
}
