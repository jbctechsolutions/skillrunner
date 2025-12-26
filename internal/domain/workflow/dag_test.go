package workflow

import (
	"slices"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/domain/errors"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// Helper function to create a phase for testing
func testPhase(id string, deps ...string) skill.Phase {
	return skill.Phase{
		ID:             id,
		Name:           id + " phase",
		PromptTemplate: "test prompt",
		RoutingProfile: skill.RoutingProfileBalanced,
		DependsOn:      deps,
		MaxTokens:      4096,
		Temperature:    0.7,
	}
}

func TestNewDAG_EmptyPhases(t *testing.T) {
	_, err := NewDAG(nil)
	if err == nil {
		t.Fatal("expected error for nil phases")
	}
	if !errors.Is(err, errors.ErrNoPhasesDefied) {
		t.Errorf("expected ErrNoPhasesDefied, got %v", err)
	}

	_, err = NewDAG([]skill.Phase{})
	if err == nil {
		t.Fatal("expected error for empty phases")
	}
	if !errors.Is(err, errors.ErrNoPhasesDefied) {
		t.Errorf("expected ErrNoPhasesDefied, got %v", err)
	}
}

func TestNewDAG_SinglePhase(t *testing.T) {
	phases := []skill.Phase{testPhase("a")}
	dag, err := NewDAG(phases)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dag.Size() != 1 {
		t.Errorf("expected size 1, got %d", dag.Size())
	}
}

func TestNewDAG_LinearDependencies(t *testing.T) {
	// a -> b -> c (linear chain)
	phases := []skill.Phase{
		testPhase("a"),
		testPhase("b", "a"),
		testPhase("c", "b"),
	}
	dag, err := NewDAG(phases)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dag.Size() != 3 {
		t.Errorf("expected size 3, got %d", dag.Size())
	}

	// Check dependencies
	if deps := dag.GetDependencies("a"); deps != nil {
		t.Errorf("expected no dependencies for a, got %v", deps)
	}
	if deps := dag.GetDependencies("b"); len(deps) != 1 || deps[0] != "a" {
		t.Errorf("expected b to depend on a, got %v", deps)
	}
	if deps := dag.GetDependencies("c"); len(deps) != 1 || deps[0] != "b" {
		t.Errorf("expected c to depend on b, got %v", deps)
	}

	// Check dependents
	if deps := dag.GetDependents("a"); len(deps) != 1 || deps[0] != "b" {
		t.Errorf("expected a's dependent to be b, got %v", deps)
	}
	if deps := dag.GetDependents("b"); len(deps) != 1 || deps[0] != "c" {
		t.Errorf("expected b's dependent to be c, got %v", deps)
	}
	if deps := dag.GetDependents("c"); deps != nil {
		t.Errorf("expected no dependents for c, got %v", deps)
	}
}

func TestNewDAG_DiamondDependencies(t *testing.T) {
	// Diamond pattern:
	//      a
	//     / \
	//    b   c
	//     \ /
	//      d
	phases := []skill.Phase{
		testPhase("a"),
		testPhase("b", "a"),
		testPhase("c", "a"),
		testPhase("d", "b", "c"),
	}
	dag, err := NewDAG(phases)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dag.Size() != 4 {
		t.Errorf("expected size 4, got %d", dag.Size())
	}

	// Check d's dependencies
	deps := dag.GetDependencies("d")
	if len(deps) != 2 {
		t.Errorf("expected d to have 2 dependencies, got %d", len(deps))
	}
	if !slices.Contains(deps, "b") || !slices.Contains(deps, "c") {
		t.Errorf("expected d to depend on b and c, got %v", deps)
	}

	// Check a's dependents
	aDeps := dag.GetDependents("a")
	if len(aDeps) != 2 {
		t.Errorf("expected a to have 2 dependents, got %d", len(aDeps))
	}
	if !slices.Contains(aDeps, "b") || !slices.Contains(aDeps, "c") {
		t.Errorf("expected a's dependents to be b and c, got %v", aDeps)
	}
}

func TestNewDAG_MissingDependency(t *testing.T) {
	phases := []skill.Phase{
		testPhase("a"),
		testPhase("b", "nonexistent"),
	}
	_, err := NewDAG(phases)
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
	if !errors.Is(err, errors.ErrDependencyNotFound) {
		t.Errorf("expected ErrDependencyNotFound, got %v", err)
	}
}

func TestNewDAG_SimpleCycle(t *testing.T) {
	// a -> b -> a (simple cycle)
	phases := []skill.Phase{
		testPhase("a", "b"),
		testPhase("b", "a"),
	}
	_, err := NewDAG(phases)
	if err == nil {
		t.Fatal("expected error for cycle")
	}
	if !errors.Is(err, errors.ErrCycleDetected) {
		t.Errorf("expected ErrCycleDetected, got %v", err)
	}
}

func TestNewDAG_SelfCycle(t *testing.T) {
	// a -> a (self-cycle)
	phases := []skill.Phase{
		testPhase("a", "a"),
	}
	_, err := NewDAG(phases)
	if err == nil {
		t.Fatal("expected error for self-cycle")
	}
	if !errors.Is(err, errors.ErrCycleDetected) {
		t.Errorf("expected ErrCycleDetected, got %v", err)
	}
}

func TestNewDAG_LongCycle(t *testing.T) {
	// a -> b -> c -> d -> a (long cycle)
	phases := []skill.Phase{
		testPhase("a", "d"),
		testPhase("b", "a"),
		testPhase("c", "b"),
		testPhase("d", "c"),
	}
	_, err := NewDAG(phases)
	if err == nil {
		t.Fatal("expected error for long cycle")
	}
	if !errors.Is(err, errors.ErrCycleDetected) {
		t.Errorf("expected ErrCycleDetected, got %v", err)
	}
}

func TestNewDAG_PartialCycle(t *testing.T) {
	// a -> b -> c
	//      ^---d
	// b -> d -> b (cycle in subgraph)
	phases := []skill.Phase{
		testPhase("a"),
		testPhase("b", "a", "d"),
		testPhase("c", "b"),
		testPhase("d", "b"),
	}
	_, err := NewDAG(phases)
	if err == nil {
		t.Fatal("expected error for partial cycle")
	}
	if !errors.Is(err, errors.ErrCycleDetected) {
		t.Errorf("expected ErrCycleDetected, got %v", err)
	}
}

func TestTopologicalSort_Linear(t *testing.T) {
	phases := []skill.Phase{
		testPhase("a"),
		testPhase("b", "a"),
		testPhase("c", "b"),
	}
	dag, err := NewDAG(phases)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sorted, err := dag.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"a", "b", "c"}
	if !slices.Equal(sorted, expected) {
		t.Errorf("expected %v, got %v", expected, sorted)
	}
}

func TestTopologicalSort_Diamond(t *testing.T) {
	phases := []skill.Phase{
		testPhase("a"),
		testPhase("b", "a"),
		testPhase("c", "a"),
		testPhase("d", "b", "c"),
	}
	dag, err := NewDAG(phases)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sorted, err := dag.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify length
	if len(sorted) != 4 {
		t.Errorf("expected 4 items, got %d", len(sorted))
	}

	// a must come before b and c
	aIdx := slices.Index(sorted, "a")
	bIdx := slices.Index(sorted, "b")
	cIdx := slices.Index(sorted, "c")
	dIdx := slices.Index(sorted, "d")

	if aIdx > bIdx || aIdx > cIdx {
		t.Errorf("a must come before b and c: %v", sorted)
	}
	if bIdx > dIdx || cIdx > dIdx {
		t.Errorf("b and c must come before d: %v", sorted)
	}
}

func TestTopologicalSort_Parallel(t *testing.T) {
	// a, b, c are independent
	phases := []skill.Phase{
		testPhase("a"),
		testPhase("b"),
		testPhase("c"),
	}
	dag, err := NewDAG(phases)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sorted, err := dag.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All phases should be present
	if len(sorted) != 3 {
		t.Errorf("expected 3 items, got %d", len(sorted))
	}
	if !slices.Contains(sorted, "a") || !slices.Contains(sorted, "b") || !slices.Contains(sorted, "c") {
		t.Errorf("expected all phases present, got %v", sorted)
	}
}

func TestGetParallelBatches_Linear(t *testing.T) {
	phases := []skill.Phase{
		testPhase("a"),
		testPhase("b", "a"),
		testPhase("c", "b"),
	}
	dag, err := NewDAG(phases)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	batches, err := dag.GetParallelBatches()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 3 batches, each with 1 phase
	if len(batches) != 3 {
		t.Errorf("expected 3 batches, got %d", len(batches))
	}
	for i, batch := range batches {
		if len(batch) != 1 {
			t.Errorf("batch %d: expected 1 phase, got %d", i, len(batch))
		}
	}
}

func TestGetParallelBatches_AllParallel(t *testing.T) {
	phases := []skill.Phase{
		testPhase("a"),
		testPhase("b"),
		testPhase("c"),
	}
	dag, err := NewDAG(phases)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	batches, err := dag.GetParallelBatches()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 1 batch with all 3 phases
	if len(batches) != 1 {
		t.Errorf("expected 1 batch, got %d", len(batches))
	}
	if len(batches[0]) != 3 {
		t.Errorf("expected 3 phases in batch, got %d", len(batches[0]))
	}
}

func TestGetParallelBatches_Diamond(t *testing.T) {
	phases := []skill.Phase{
		testPhase("a"),
		testPhase("b", "a"),
		testPhase("c", "a"),
		testPhase("d", "b", "c"),
	}
	dag, err := NewDAG(phases)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	batches, err := dag.GetParallelBatches()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 3 batches: [a], [b, c], [d]
	if len(batches) != 3 {
		t.Errorf("expected 3 batches, got %d: %v", len(batches), batches)
	}

	// First batch should be [a]
	if len(batches[0]) != 1 || batches[0][0] != "a" {
		t.Errorf("expected first batch to be [a], got %v", batches[0])
	}

	// Second batch should be [b, c] (order may vary)
	if len(batches[1]) != 2 {
		t.Errorf("expected second batch to have 2 phases, got %d", len(batches[1]))
	}
	if !slices.Contains(batches[1], "b") || !slices.Contains(batches[1], "c") {
		t.Errorf("expected second batch to contain b and c, got %v", batches[1])
	}

	// Third batch should be [d]
	if len(batches[2]) != 1 || batches[2][0] != "d" {
		t.Errorf("expected third batch to be [d], got %v", batches[2])
	}
}

func TestGetParallelBatches_Complex(t *testing.T) {
	// Complex graph:
	//   a   b
	//   |\ /|
	//   | X |
	//   |/ \|
	//   c   d
	//    \ /
	//     e
	phases := []skill.Phase{
		testPhase("a"),
		testPhase("b"),
		testPhase("c", "a", "b"),
		testPhase("d", "a", "b"),
		testPhase("e", "c", "d"),
	}
	dag, err := NewDAG(phases)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	batches, err := dag.GetParallelBatches()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 3 batches: [a, b], [c, d], [e]
	if len(batches) != 3 {
		t.Errorf("expected 3 batches, got %d: %v", len(batches), batches)
	}

	// First batch should contain a and b
	if len(batches[0]) != 2 {
		t.Errorf("expected first batch to have 2 phases, got %d", len(batches[0]))
	}

	// Second batch should contain c and d
	if len(batches[1]) != 2 {
		t.Errorf("expected second batch to have 2 phases, got %d", len(batches[1]))
	}

	// Third batch should contain e
	if len(batches[2]) != 1 {
		t.Errorf("expected third batch to have 1 phase, got %d", len(batches[2]))
	}
}

func TestGetDependencies_NonExistent(t *testing.T) {
	phases := []skill.Phase{testPhase("a")}
	dag, _ := NewDAG(phases)

	deps := dag.GetDependencies("nonexistent")
	if deps != nil {
		t.Errorf("expected nil for nonexistent phase, got %v", deps)
	}
}

func TestGetDependents_NonExistent(t *testing.T) {
	phases := []skill.Phase{testPhase("a")}
	dag, _ := NewDAG(phases)

	deps := dag.GetDependents("nonexistent")
	if deps != nil {
		t.Errorf("expected nil for nonexistent phase, got %v", deps)
	}
}

func TestGetNode(t *testing.T) {
	phases := []skill.Phase{
		testPhase("a"),
		testPhase("b", "a"),
	}
	dag, _ := NewDAG(phases)

	node := dag.GetNode("a")
	if node == nil {
		t.Fatal("expected node for a")
	}
	if node.Phase.ID != "a" {
		t.Errorf("expected phase ID a, got %s", node.Phase.ID)
	}
	if node.InDegree != 0 {
		t.Errorf("expected in-degree 0 for a, got %d", node.InDegree)
	}

	nodeB := dag.GetNode("b")
	if nodeB == nil {
		t.Fatal("expected node for b")
	}
	if nodeB.InDegree != 1 {
		t.Errorf("expected in-degree 1 for b, got %d", nodeB.InDegree)
	}

	nodeNone := dag.GetNode("nonexistent")
	if nodeNone != nil {
		t.Errorf("expected nil for nonexistent, got %v", nodeNone)
	}
}

func TestGetPhase(t *testing.T) {
	phases := []skill.Phase{testPhase("a")}
	dag, _ := NewDAG(phases)

	phase := dag.GetPhase("a")
	if phase == nil {
		t.Fatal("expected phase for a")
	}
	if phase.ID != "a" {
		t.Errorf("expected phase ID a, got %s", phase.ID)
	}

	phaseNone := dag.GetPhase("nonexistent")
	if phaseNone != nil {
		t.Errorf("expected nil for nonexistent, got %v", phaseNone)
	}
}

func TestDAG_Size(t *testing.T) {
	tests := []struct {
		name     string
		phases   []skill.Phase
		expected int
	}{
		{"single", []skill.Phase{testPhase("a")}, 1},
		{"two", []skill.Phase{testPhase("a"), testPhase("b")}, 2},
		{"five", []skill.Phase{testPhase("a"), testPhase("b"), testPhase("c"), testPhase("d"), testPhase("e")}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dag, err := NewDAG(tt.phases)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if dag.Size() != tt.expected {
				t.Errorf("expected size %d, got %d", tt.expected, dag.Size())
			}
		})
	}
}

func TestDAG_ImmutableReturns(t *testing.T) {
	phases := []skill.Phase{
		testPhase("a"),
		testPhase("b", "a"),
	}
	dag, _ := NewDAG(phases)

	// Modify returned dependencies - should not affect original
	deps := dag.GetDependencies("b")
	deps[0] = "modified"

	originalDeps := dag.GetDependencies("b")
	if originalDeps[0] != "a" {
		t.Errorf("modifying returned slice affected original: got %v", originalDeps)
	}

	// Modify returned dependents - should not affect original
	dependents := dag.GetDependents("a")
	dependents[0] = "modified"

	originalDependents := dag.GetDependents("a")
	if originalDependents[0] != "b" {
		t.Errorf("modifying returned slice affected original: got %v", originalDependents)
	}
}
