// Package testutil provides test fixtures and helpers for testing.
package testutil

import (
	"github.com/jbctechsolutions/skillrunner/internal/application/ports"
	"github.com/jbctechsolutions/skillrunner/internal/domain/provider"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
	"github.com/jbctechsolutions/skillrunner/internal/domain/workflow"
)

// NewTestSkill creates a test skill with two phases for testing.
func NewTestSkill() *skill.Skill {
	analyzePhase, _ := skill.NewPhase("analyze", "Analyze", "Analyze the input: {{.Input}}")
	generatePhase, _ := skill.NewPhase("generate", "Generate", "Generate output based on: {{.PreviousOutput}}")
	generatePhase = generatePhase.WithDependencies([]string{"analyze"})

	phases := []skill.Phase{*analyzePhase, *generatePhase}
	s, _ := skill.NewSkill("test-skill", "Test Skill", "1.0.0", phases)
	return s
}

// NewSimplePhase creates a simple phase for testing.
func NewSimplePhase(id, name string) *skill.Phase {
	p, _ := skill.NewPhase(id, name, "Test prompt for "+name)
	return p
}

// NewTestModel creates a test model with the given tier.
func NewTestModel(id string, tier provider.AgentTier) *provider.Model {
	return provider.NewModel(id, "Test "+id, "test").
		WithTier(tier).
		WithCosts(0.001, 0.002).
		WithContextWindow(4096)
}

// NewUserMessage creates a user message for testing.
func NewUserMessage(content string) ports.Message {
	return ports.Message{Role: "user", Content: content}
}

// NewAssistantMessage creates an assistant message for testing.
func NewAssistantMessage(content string) ports.Message {
	return ports.Message{Role: "assistant", Content: content}
}

// NewSystemMessage creates a system message for testing.
func NewSystemMessage(content string) ports.Message {
	return ports.Message{Role: "system", Content: content}
}

// NewLinearDAG creates a DAG with linear dependencies: A -> B -> C
func NewLinearDAG() (*workflow.DAG, error) {
	phaseA, _ := skill.NewPhase("a", "Phase A", "Prompt A")
	phaseB, _ := skill.NewPhase("b", "Phase B", "Prompt B")
	phaseB = phaseB.WithDependencies([]string{"a"})
	phaseC, _ := skill.NewPhase("c", "Phase C", "Prompt C")
	phaseC = phaseC.WithDependencies([]string{"b"})

	return workflow.NewDAG([]skill.Phase{*phaseA, *phaseB, *phaseC})
}

// NewDiamondDAG creates a diamond dependency DAG: A -> B, A -> C, B -> D, C -> D
func NewDiamondDAG() (*workflow.DAG, error) {
	phaseA, _ := skill.NewPhase("a", "Phase A", "Prompt A")
	phaseB, _ := skill.NewPhase("b", "Phase B", "Prompt B")
	phaseB = phaseB.WithDependencies([]string{"a"})
	phaseC, _ := skill.NewPhase("c", "Phase C", "Prompt C")
	phaseC = phaseC.WithDependencies([]string{"a"})
	phaseD, _ := skill.NewPhase("d", "Phase D", "Prompt D")
	phaseD = phaseD.WithDependencies([]string{"b", "c"})

	return workflow.NewDAG([]skill.Phase{*phaseA, *phaseB, *phaseC, *phaseD})
}

// NewParallelDAG creates a DAG with parallel phases: A, B, C (no dependencies)
func NewParallelDAG() (*workflow.DAG, error) {
	phaseA, _ := skill.NewPhase("a", "Phase A", "Prompt A")
	phaseB, _ := skill.NewPhase("b", "Phase B", "Prompt B")
	phaseC, _ := skill.NewPhase("c", "Phase C", "Prompt C")

	return workflow.NewDAG([]skill.Phase{*phaseA, *phaseB, *phaseC})
}
