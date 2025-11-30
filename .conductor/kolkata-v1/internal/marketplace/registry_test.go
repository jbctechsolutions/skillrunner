package marketplace

import (
	"context"
	"testing"
)

// mockSource is a test implementation of Source
type mockSource struct {
	name     string
	priority int
	skills   map[string]*Skill
	agents   map[string]*Agent
	commands map[string]*Command
	healthy  bool
}

func newMockSource(name string, priority int) *mockSource {
	return &mockSource{
		name:     name,
		priority: priority,
		skills:   make(map[string]*Skill),
		agents:   make(map[string]*Agent),
		commands: make(map[string]*Command),
		healthy:  true,
	}
}

func (m *mockSource) Name() string     { return m.name }
func (m *mockSource) Type() SourceType { return SourceTypeLocal }
func (m *mockSource) Priority() int    { return m.priority }

func (m *mockSource) ListSkills(ctx context.Context) ([]*Skill, error) {
	skills := make([]*Skill, 0, len(m.skills))
	for _, skill := range m.skills {
		skills = append(skills, skill)
	}
	return skills, nil
}

func (m *mockSource) GetSkill(ctx context.Context, id string) (*Skill, error) {
	skill, ok := m.skills[id]
	if !ok {
		return nil, ErrNotFound(id)
	}
	return skill, nil
}

func (m *mockSource) ListAgents(ctx context.Context) ([]*Agent, error) {
	agents := make([]*Agent, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}
	return agents, nil
}

func (m *mockSource) GetAgent(ctx context.Context, id string) (*Agent, error) {
	agent, ok := m.agents[id]
	if !ok {
		return nil, ErrNotFound(id)
	}
	return agent, nil
}

func (m *mockSource) ListCommands(ctx context.Context) ([]*Command, error) {
	commands := make([]*Command, 0, len(m.commands))
	for _, cmd := range m.commands {
		commands = append(commands, cmd)
	}
	return commands, nil
}

func (m *mockSource) GetCommand(ctx context.Context, id string) (*Command, error) {
	cmd, ok := m.commands[id]
	if !ok {
		return nil, ErrNotFound(id)
	}
	return cmd, nil
}

func (m *mockSource) Search(ctx context.Context, query string, types []ComponentType) ([]*SearchResult, error) {
	return nil, nil
}

func (m *mockSource) Refresh(ctx context.Context) error {
	return nil
}

func (m *mockSource) IsHealthy(ctx context.Context) bool {
	return m.healthy
}

func (m *mockSource) addSkill(skill *Skill) {
	m.skills[skill.ID] = skill
}

func (m *mockSource) addAgent(agent *Agent) {
	m.agents[agent.ID] = agent
}

// ErrNotFound returns a not found error
func ErrNotFound(id string) error {
	return &notFoundError{id: id}
}

type notFoundError struct {
	id string
}

func (e *notFoundError) Error() string {
	return "not found: " + e.id
}

func TestRegistry_AddAndRemoveSource(t *testing.T) {
	registry := NewRegistry()

	// Add a source
	source := newMockSource("test-source", 0)
	if err := registry.AddSource(source); err != nil {
		t.Fatalf("AddSource() error = %v", err)
	}

	// Verify source was added
	sources := registry.ListSources()
	if len(sources) != 1 {
		t.Errorf("ListSources() = %d sources, want 1", len(sources))
	}

	// Try to add duplicate
	if err := registry.AddSource(source); err == nil {
		t.Error("AddSource() expected error for duplicate source")
	}

	// Remove source
	if err := registry.RemoveSource("test-source"); err != nil {
		t.Fatalf("RemoveSource() error = %v", err)
	}

	// Verify source was removed
	sources = registry.ListSources()
	if len(sources) != 0 {
		t.Errorf("ListSources() = %d sources, want 0", len(sources))
	}

	// Try to remove non-existent source
	if err := registry.RemoveSource("non-existent"); err == nil {
		t.Error("RemoveSource() expected error for non-existent source")
	}
}

func TestRegistry_PriorityOrdering(t *testing.T) {
	registry := NewRegistry()

	// Add sources in non-priority order
	source3 := newMockSource("source3", 30)
	source1 := newMockSource("source1", 10)
	source2 := newMockSource("source2", 20)

	registry.AddSource(source3)
	registry.AddSource(source1)
	registry.AddSource(source2)

	sources := registry.ListSources()
	if len(sources) != 3 {
		t.Fatalf("ListSources() = %d sources, want 3", len(sources))
	}

	// Verify priority order
	expectedOrder := []string{"source1", "source2", "source3"}
	for i, source := range sources {
		if source.Name() != expectedOrder[i] {
			t.Errorf("sources[%d].Name() = %s, want %s", i, source.Name(), expectedOrder[i])
		}
	}
}

func TestRegistry_ListSkills(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	// Add two sources with skills
	source1 := newMockSource("source1", 0)
	source1.addSkill(&Skill{ID: "skill1", Source: "source1", Name: "Skill 1"})
	source1.addSkill(&Skill{ID: "skill2", Source: "source1", Name: "Skill 2"})

	source2 := newMockSource("source2", 10)
	source2.addSkill(&Skill{ID: "skill1", Source: "source2", Name: "Skill 1 Alt"}) // Duplicate ID
	source2.addSkill(&Skill{ID: "skill3", Source: "source2", Name: "Skill 3"})

	registry.AddSource(source1)
	registry.AddSource(source2)

	skills, err := registry.ListSkills(ctx)
	if err != nil {
		t.Fatalf("ListSkills() error = %v", err)
	}

	// Should have 3 unique skills (skill1 from source1 wins due to priority)
	if len(skills) != 3 {
		t.Errorf("ListSkills() = %d skills, want 3", len(skills))
	}

	// Verify skill1 comes from source1 (higher priority)
	for _, skill := range skills {
		if skill.ID == "skill1" && skill.Source != "source1" {
			t.Errorf("skill1.Source = %s, want source1", skill.Source)
		}
	}
}

func TestRegistry_GetSkill(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	// Add source with skill
	source := newMockSource("source1", 0)
	source.addSkill(&Skill{ID: "test-skill", Source: "source1", Name: "Test Skill"})
	registry.AddSource(source)

	// Get existing skill
	skill, err := registry.GetSkill(ctx, "test-skill")
	if err != nil {
		t.Fatalf("GetSkill() error = %v", err)
	}
	if skill.ID != "test-skill" {
		t.Errorf("skill.ID = %s, want test-skill", skill.ID)
	}

	// Get non-existent skill
	_, err = registry.GetSkill(ctx, "non-existent")
	if err == nil {
		t.Error("GetSkill() expected error for non-existent skill")
	}
}

func TestRegistry_ListAgents(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	source := newMockSource("source1", 0)
	source.addAgent(&Agent{ID: "agent1", Source: "source1", Name: "Agent 1"})
	source.addAgent(&Agent{ID: "agent2", Source: "source1", Name: "Agent 2"})
	registry.AddSource(source)

	agents, err := registry.ListAgents(ctx)
	if err != nil {
		t.Fatalf("ListAgents() error = %v", err)
	}

	if len(agents) != 2 {
		t.Errorf("ListAgents() = %d agents, want 2", len(agents))
	}
}

func TestRegistry_GetAgent(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	source := newMockSource("source1", 0)
	source.addAgent(&Agent{ID: "test-agent", Source: "source1", Name: "Test Agent"})
	registry.AddSource(source)

	// Get existing agent
	agent, err := registry.GetAgent(ctx, "test-agent")
	if err != nil {
		t.Fatalf("GetAgent() error = %v", err)
	}
	if agent.ID != "test-agent" {
		t.Errorf("agent.ID = %s, want test-agent", agent.ID)
	}

	// Get non-existent agent
	_, err = registry.GetAgent(ctx, "non-existent")
	if err == nil {
		t.Error("GetAgent() expected error for non-existent agent")
	}
}

func TestRegistry_HealthCheck(t *testing.T) {
	registry := NewRegistry()
	ctx := context.Background()

	source1 := newMockSource("healthy-source", 0)
	source1.healthy = true

	source2 := newMockSource("unhealthy-source", 10)
	source2.healthy = false

	registry.AddSource(source1)
	registry.AddSource(source2)

	status := registry.HealthCheck(ctx)

	if !status["healthy-source"] {
		t.Error("healthy-source should be healthy")
	}
	if status["unhealthy-source"] {
		t.Error("unhealthy-source should not be healthy")
	}
}

func TestRegistry_GetSource(t *testing.T) {
	registry := NewRegistry()

	source := newMockSource("test-source", 0)
	registry.AddSource(source)

	// Get existing source
	got, err := registry.GetSource("test-source")
	if err != nil {
		t.Fatalf("GetSource() error = %v", err)
	}
	if got.Name() != "test-source" {
		t.Errorf("source.Name() = %s, want test-source", got.Name())
	}

	// Get non-existent source
	_, err = registry.GetSource("non-existent")
	if err == nil {
		t.Error("GetSource() expected error for non-existent source")
	}
}

func TestRegistry_DefaultConfigs(t *testing.T) {
	registry := NewRegistry()

	defaults := registry.DefaultConfigs()
	if len(defaults) == 0 {
		t.Error("DefaultConfigs() returned empty slice")
	}

	// Verify jbctech-marketplace is in defaults
	foundLocal := false
	for _, config := range defaults {
		if config.Name == "jbctech-marketplace" && config.Type == SourceTypeLocal {
			foundLocal = true
			break
		}
	}
	if !foundLocal {
		t.Error("jbctech-marketplace not found in default configs")
	}
}
