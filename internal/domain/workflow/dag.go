// Package workflow provides workflow orchestration types for skill execution.
package workflow

import (
	"github.com/jbctechsolutions/skillrunner/internal/domain/errors"
	"github.com/jbctechsolutions/skillrunner/internal/domain/skill"
)

// Node represents a single node in the DAG, wrapping a phase with graph metadata.
type Node struct {
	Phase    *skill.Phase
	InDegree int      // Number of phases this phase depends on
	OutEdges []string // IDs of phases that depend on this phase
}

// DAG represents a directed acyclic graph of phases for workflow execution.
// It provides methods for topological sorting and parallel execution planning.
type DAG struct {
	nodes map[string]*Node    // phase ID -> node
	edges map[string][]string // phase ID -> dependent phase IDs
}

// NewDAG builds a DAG from the given phases.
// It validates that all dependencies exist and detects cycles.
// Returns an error if validation fails or a cycle is detected.
func NewDAG(phases []skill.Phase) (*DAG, error) {
	if len(phases) == 0 {
		return nil, errors.ErrNoPhasesDefied
	}

	dag := &DAG{
		nodes: make(map[string]*Node),
		edges: make(map[string][]string),
	}

	// Build nodes from phases
	for i := range phases {
		phase := &phases[i]
		dag.nodes[phase.ID] = &Node{
			Phase:    phase,
			InDegree: len(phase.DependsOn),
			OutEdges: make([]string, 0),
		}
		dag.edges[phase.ID] = make([]string, 0)
	}

	// Build edges from DependsOn and validate dependencies exist
	for _, phase := range phases {
		for _, depID := range phase.DependsOn {
			if _, exists := dag.nodes[depID]; !exists {
				return nil, errors.WithContext(
					errors.NewError(errors.CodeValidation, "dependency not found", errors.ErrDependencyNotFound),
					"phase_id", phase.ID,
				)
			}
			// Add edge from dependency to this phase
			dag.edges[depID] = append(dag.edges[depID], phase.ID)
			dag.nodes[depID].OutEdges = append(dag.nodes[depID].OutEdges, phase.ID)
		}
	}

	// Detect cycles using DFS
	if dag.hasCycle() {
		return nil, errors.NewError(errors.CodeValidation, "cycle detected in phase dependencies", errors.ErrCycleDetected)
	}

	return dag, nil
}

// TopologicalSort returns phase IDs in execution order using Kahn's algorithm.
// Phases with no unresolved dependencies come first.
func (d *DAG) TopologicalSort() ([]string, error) {
	if len(d.nodes) == 0 {
		return nil, nil
	}

	// Copy in-degrees so we don't mutate the original
	inDegree := make(map[string]int)
	for id, node := range d.nodes {
		inDegree[id] = node.InDegree
	}

	// Find all nodes with no dependencies (in-degree 0)
	queue := make([]string, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	result := make([]string, 0, len(d.nodes))

	for len(queue) > 0 {
		// Dequeue
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Reduce in-degree of dependents
		for _, dependent := range d.edges[current] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// If we didn't process all nodes, there's a cycle
	if len(result) != len(d.nodes) {
		return nil, errors.NewError(errors.CodeValidation, "cycle detected during topological sort", errors.ErrCycleDetected)
	}

	return result, nil
}

// GetParallelBatches returns phases grouped by parallel execution opportunity.
// Each batch contains phases that can be executed concurrently because all their
// dependencies have been satisfied by previous batches.
func (d *DAG) GetParallelBatches() ([][]string, error) {
	if len(d.nodes) == 0 {
		return nil, nil
	}

	// Copy in-degrees so we don't mutate the original
	inDegree := make(map[string]int)
	for id, node := range d.nodes {
		inDegree[id] = node.InDegree
	}

	batches := make([][]string, 0)
	remaining := len(d.nodes)

	for remaining > 0 {
		// Find all nodes with in-degree 0 (ready to execute)
		batch := make([]string, 0)
		for id, deg := range inDegree {
			if deg == 0 {
				batch = append(batch, id)
			}
		}

		if len(batch) == 0 {
			// No nodes with in-degree 0 but still remaining nodes means a cycle
			return nil, errors.NewError(errors.CodeValidation, "cycle detected during batch generation", errors.ErrCycleDetected)
		}

		// Mark batch nodes as processed and reduce in-degrees
		for _, id := range batch {
			inDegree[id] = -1 // Mark as processed
			remaining--
			for _, dependent := range d.edges[id] {
				if inDegree[dependent] > 0 {
					inDegree[dependent]--
				}
			}
		}

		batches = append(batches, batch)
	}

	return batches, nil
}

// GetDependencies returns the phase IDs that the given phase depends on.
// Returns nil if the phase doesn't exist or has no dependencies.
func (d *DAG) GetDependencies(phaseID string) []string {
	node, exists := d.nodes[phaseID]
	if !exists || node.Phase == nil {
		return nil
	}
	if len(node.Phase.DependsOn) == 0 {
		return nil
	}
	// Return a copy to prevent external mutation
	deps := make([]string, len(node.Phase.DependsOn))
	copy(deps, node.Phase.DependsOn)
	return deps
}

// GetDependents returns the phase IDs that depend on the given phase.
// Returns nil if the phase doesn't exist or has no dependents.
func (d *DAG) GetDependents(phaseID string) []string {
	node, exists := d.nodes[phaseID]
	if !exists {
		return nil
	}
	if len(node.OutEdges) == 0 {
		return nil
	}
	// Return a copy to prevent external mutation
	deps := make([]string, len(node.OutEdges))
	copy(deps, node.OutEdges)
	return deps
}

// GetNode returns the node for a given phase ID.
// Returns nil if the phase doesn't exist.
func (d *DAG) GetNode(phaseID string) *Node {
	return d.nodes[phaseID]
}

// GetPhase returns the phase for a given phase ID.
// Returns nil if the phase doesn't exist.
func (d *DAG) GetPhase(phaseID string) *skill.Phase {
	node := d.nodes[phaseID]
	if node == nil {
		return nil
	}
	return node.Phase
}

// Size returns the number of phases in the DAG.
func (d *DAG) Size() int {
	return len(d.nodes)
}

// hasCycle uses DFS to detect cycles in the graph.
func (d *DAG) hasCycle() bool {
	// Track visited state: 0=unvisited, 1=in progress, 2=completed
	visited := make(map[string]int)

	var dfs func(id string) bool
	dfs = func(id string) bool {
		if visited[id] == 1 {
			// Currently visiting this node - cycle detected
			return true
		}
		if visited[id] == 2 {
			// Already completed - no cycle through this path
			return false
		}

		visited[id] = 1 // Mark as in progress

		// Check all dependents (outgoing edges)
		for _, dependent := range d.edges[id] {
			if dfs(dependent) {
				return true
			}
		}

		visited[id] = 2 // Mark as completed
		return false
	}

	// Run DFS from each unvisited node
	for id := range d.nodes {
		if visited[id] == 0 {
			if dfs(id) {
				return true
			}
		}
	}

	return false
}
