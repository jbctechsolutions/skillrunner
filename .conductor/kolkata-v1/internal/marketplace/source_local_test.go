package marketplace

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalSource_NewLocalSource(t *testing.T) {
	// Create temp directory structure
	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, "skills")
	agentsDir := filepath.Join(tempDir, "agents")

	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	// Create a skill
	skillDir := filepath.Join(skillsDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}

	skillContent := `---
name: Test Skill
description: A test skill for unit testing
keywords: test, unit, example
version: 1.0.0
author: Test Author
last_updated: 2024-01-01
---

# Test Skill

This is a test skill.
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("failed to write skill file: %v", err)
	}

	// Create an agent
	categoryDir := filepath.Join(agentsDir, "test-category")
	if err := os.MkdirAll(categoryDir, 0755); err != nil {
		t.Fatalf("failed to create category dir: %v", err)
	}

	agentContent := `---
name: Test Agent
description: A test agent for unit testing
model: gpt-4
primary_skill: test-skill
tools: Read, Write, Bash
---

# Test Agent

This is a test agent.
`
	if err := os.WriteFile(filepath.Join(categoryDir, "test-agent.md"), []byte(agentContent), 0644); err != nil {
		t.Fatalf("failed to write agent file: %v", err)
	}

	// Create local source
	config := SourceConfig{
		Name:     "test-source",
		Type:     SourceTypeLocal,
		Path:     tempDir,
		Priority: 0,
		Enabled:  true,
	}

	source, err := NewLocalSource(config)
	if err != nil {
		t.Fatalf("NewLocalSource() error = %v", err)
	}

	// Verify source properties
	if source.Name() != "test-source" {
		t.Errorf("Name() = %s, want test-source", source.Name())
	}
	if source.Type() != SourceTypeLocal {
		t.Errorf("Type() = %s, want local", source.Type())
	}
	if source.Priority() != 0 {
		t.Errorf("Priority() = %d, want 0", source.Priority())
	}
}

func TestLocalSource_ListSkills(t *testing.T) {
	// Create temp directory structure
	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}

	// Create two skills
	for _, skillID := range []string{"skill1", "skill2"} {
		skillDir := filepath.Join(skillsDir, skillID)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("failed to create skill dir: %v", err)
		}

		content := `---
name: ` + skillID + `
description: Test skill ` + skillID + `
version: 1.0.0
---

# ` + skillID + `
`
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write skill file: %v", err)
		}
	}

	config := SourceConfig{
		Name:     "test-source",
		Type:     SourceTypeLocal,
		Path:     tempDir,
		Priority: 0,
		Enabled:  true,
	}

	source, err := NewLocalSource(config)
	if err != nil {
		t.Fatalf("NewLocalSource() error = %v", err)
	}

	ctx := context.Background()
	skills, err := source.ListSkills(ctx)
	if err != nil {
		t.Fatalf("ListSkills() error = %v", err)
	}

	if len(skills) != 2 {
		t.Errorf("ListSkills() = %d skills, want 2", len(skills))
	}
}

func TestLocalSource_GetSkill(t *testing.T) {
	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, "skills")
	skillDir := filepath.Join(skillsDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}

	content := `---
name: Test Skill
description: A test skill
version: 1.0.0
---

# Test Skill
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write skill file: %v", err)
	}

	config := SourceConfig{
		Name:     "test-source",
		Type:     SourceTypeLocal,
		Path:     tempDir,
		Priority: 0,
		Enabled:  true,
	}

	source, err := NewLocalSource(config)
	if err != nil {
		t.Fatalf("NewLocalSource() error = %v", err)
	}

	ctx := context.Background()

	// Get existing skill
	skill, err := source.GetSkill(ctx, "test-skill")
	if err != nil {
		t.Fatalf("GetSkill() error = %v", err)
	}
	if skill.Name != "Test Skill" {
		t.Errorf("skill.Name = %s, want Test Skill", skill.Name)
	}

	// Get non-existent skill
	_, err = source.GetSkill(ctx, "non-existent")
	if err == nil {
		t.Error("GetSkill() expected error for non-existent skill")
	}
}

func TestLocalSource_ListAgents(t *testing.T) {
	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, "skills")
	agentsDir := filepath.Join(tempDir, "agents")

	// Create skills directory with at least one skill
	skillDir := filepath.Join(skillsDir, "dummy-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	skillContent := `---
name: Dummy Skill
description: A dummy skill
version: 1.0.0
---

# Dummy Skill
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("failed to write skill file: %v", err)
	}

	// Create agents directory with category
	categoryDir := filepath.Join(agentsDir, "test-category")
	if err := os.MkdirAll(categoryDir, 0755); err != nil {
		t.Fatalf("failed to create category dir: %v", err)
	}

	// Create two agents
	for _, agentID := range []string{"agent1", "agent2"} {
		content := `---
name: ` + agentID + `
description: Test agent ` + agentID + `
model: gpt-4
---

# ` + agentID + `
`
		if err := os.WriteFile(filepath.Join(categoryDir, agentID+".md"), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write agent file: %v", err)
		}
	}

	config := SourceConfig{
		Name:     "test-source",
		Type:     SourceTypeLocal,
		Path:     tempDir,
		Priority: 0,
		Enabled:  true,
	}

	source, err := NewLocalSource(config)
	if err != nil {
		t.Fatalf("NewLocalSource() error = %v", err)
	}

	ctx := context.Background()
	agents, err := source.ListAgents(ctx)
	if err != nil {
		t.Fatalf("ListAgents() error = %v", err)
	}

	if len(agents) != 2 {
		t.Errorf("ListAgents() = %d agents, want 2", len(agents))
	}
}

func TestLocalSource_GetAgent(t *testing.T) {
	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, "skills")
	agentsDir := filepath.Join(tempDir, "agents")

	// Create skill
	skillDir := filepath.Join(skillsDir, "dummy-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	skillContent := `---
name: Dummy Skill
description: A dummy skill
version: 1.0.0
---

# Dummy Skill
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0644); err != nil {
		t.Fatalf("failed to write skill file: %v", err)
	}

	// Create agent
	categoryDir := filepath.Join(agentsDir, "test-category")
	if err := os.MkdirAll(categoryDir, 0755); err != nil {
		t.Fatalf("failed to create category dir: %v", err)
	}

	content := `---
name: Test Agent
description: A test agent
model: gpt-4
primary_skill: test-skill
tools: Read, Write
---

# Test Agent
`
	if err := os.WriteFile(filepath.Join(categoryDir, "test-agent.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write agent file: %v", err)
	}

	config := SourceConfig{
		Name:     "test-source",
		Type:     SourceTypeLocal,
		Path:     tempDir,
		Priority: 0,
		Enabled:  true,
	}

	source, err := NewLocalSource(config)
	if err != nil {
		t.Fatalf("NewLocalSource() error = %v", err)
	}

	ctx := context.Background()

	// Get existing agent
	agent, err := source.GetAgent(ctx, "test-category/test-agent")
	if err != nil {
		t.Fatalf("GetAgent() error = %v", err)
	}
	if agent.Name != "Test Agent" {
		t.Errorf("agent.Name = %s, want Test Agent", agent.Name)
	}
	if agent.Model != "gpt-4" {
		t.Errorf("agent.Model = %s, want gpt-4", agent.Model)
	}
	if len(agent.Tools) != 2 {
		t.Errorf("len(agent.Tools) = %d, want 2", len(agent.Tools))
	}

	// Get non-existent agent
	_, err = source.GetAgent(ctx, "non-existent")
	if err == nil {
		t.Error("GetAgent() expected error for non-existent agent")
	}
}

func TestLocalSource_Search(t *testing.T) {
	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, "skills")

	// Create skills with searchable content
	skills := []struct {
		id          string
		name        string
		description string
	}{
		{"golang", "Golang Expert", "Expert in Go programming language"},
		{"python", "Python Expert", "Expert in Python programming language"},
		{"database", "Database Expert", "Expert in SQL and database design"},
	}

	for _, s := range skills {
		skillDir := filepath.Join(skillsDir, s.id)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("failed to create skill dir: %v", err)
		}

		content := `---
name: ` + s.name + `
description: ` + s.description + `
version: 1.0.0
---

# ` + s.name + `
`
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
			t.Fatalf("failed to write skill file: %v", err)
		}
	}

	config := SourceConfig{
		Name:     "test-source",
		Type:     SourceTypeLocal,
		Path:     tempDir,
		Priority: 0,
		Enabled:  true,
	}

	source, err := NewLocalSource(config)
	if err != nil {
		t.Fatalf("NewLocalSource() error = %v", err)
	}

	ctx := context.Background()

	// Search for "programming"
	results, err := source.Search(ctx, "programming", nil)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Search('programming') = %d results, want 2", len(results))
	}

	// Search for "golang"
	results, err = source.Search(ctx, "golang", []ComponentType{ComponentTypeSkill})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Search('golang') = %d results, want 1", len(results))
	}
}

func TestLocalSource_IsHealthy(t *testing.T) {
	tempDir := t.TempDir()
	skillsDir := filepath.Join(tempDir, "skills")
	skillDir := filepath.Join(skillsDir, "test-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}

	content := `---
name: Test Skill
description: A test skill
version: 1.0.0
---

# Test Skill
`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write skill file: %v", err)
	}

	config := SourceConfig{
		Name:     "test-source",
		Type:     SourceTypeLocal,
		Path:     tempDir,
		Priority: 0,
		Enabled:  true,
	}

	source, err := NewLocalSource(config)
	if err != nil {
		t.Fatalf("NewLocalSource() error = %v", err)
	}

	ctx := context.Background()

	if !source.IsHealthy(ctx) {
		t.Error("IsHealthy() = false, want true")
	}

	// Corrupt the path
	source.config.Path = "/non/existent/path"
	if source.IsHealthy(ctx) {
		t.Error("IsHealthy() = true for non-existent path, want false")
	}
}

func TestLocalSource_InvalidConfig(t *testing.T) {
	// Wrong type
	config := SourceConfig{
		Name:     "test-source",
		Type:     SourceTypeGitHub,
		Priority: 0,
		Enabled:  true,
	}

	_, err := NewLocalSource(config)
	if err == nil {
		t.Error("NewLocalSource() expected error for wrong type")
	}
}
