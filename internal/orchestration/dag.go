package orchestration

import (
	"fmt"

	"github.com/jbctechsolutions/skillrunner/internal/types"
)

// DAG represents a Directed Acyclic Graph of phases
type DAG struct {
	Nodes map[string]*DAGNode // phase_id -> node
	Edges map[string][]string // phase_id -> dependent phase_ids
}

// DAGNode represents a phase in the dependency graph
type DAGNode struct {
	PhaseID      string
	Phase        *types.Phase
	Dependencies []string // phase_ids this phase depends on
}

// NewDAG creates a new DAG from skill phases
func NewDAG(phases []types.Phase) (*DAG, error) {
	dag := &DAG{
		Nodes: make(map[string]*DAGNode),
		Edges: make(map[string][]string),
	}

	// Create nodes for each phase
	for i := range phases {
		phase := &phases[i]
		node := &DAGNode{
			PhaseID:      phase.ID,
			Phase:        phase,
			Dependencies: phase.DependsOn,
		}
		dag.Nodes[phase.ID] = node

		// Initialize edges
		if _, exists := dag.Edges[phase.ID]; !exists {
			dag.Edges[phase.ID] = []string{}
		}
	}

	// Build edges based on dependencies
	for phaseID, node := range dag.Nodes {
		for _, depID := range node.Dependencies {
			// Verify dependency exists
			if _, exists := dag.Nodes[depID]; !exists {
				return nil, fmt.Errorf("phase %s depends on non-existent phase %s", phaseID, depID)
			}

			// Add edge from dependency to dependent
			dag.Edges[depID] = append(dag.Edges[depID], phaseID)
		}
	}

	// Validate DAG (check for cycles)
	if err := dag.ValidateAcyclic(); err != nil {
		return nil, err
	}

	return dag, nil
}

// ValidateAcyclic checks if the graph contains cycles
func (dag *DAG) ValidateAcyclic() error {
	// Use depth-first search to detect cycles
	visited := make(map[string]bool)
	recursionStack := make(map[string]bool)

	for phaseID := range dag.Nodes {
		if !visited[phaseID] {
			if dag.hasCycle(phaseID, visited, recursionStack) {
				return fmt.Errorf("cycle detected in phase dependencies involving phase %s", phaseID)
			}
		}
	}

	return nil
}

// hasCycle performs DFS to detect cycles
func (dag *DAG) hasCycle(phaseID string, visited, recursionStack map[string]bool) bool {
	visited[phaseID] = true
	recursionStack[phaseID] = true

	// Visit all dependent phases
	for _, dependentID := range dag.Edges[phaseID] {
		if !visited[dependentID] {
			if dag.hasCycle(dependentID, visited, recursionStack) {
				return true
			}
		} else if recursionStack[dependentID] {
			// Found a back edge (cycle)
			return true
		}
	}

	recursionStack[phaseID] = false
	return false
}

// TopologicalSort returns phases in execution order
func (dag *DAG) TopologicalSort() ([]string, error) {
	// Calculate in-degree for each node
	inDegree := make(map[string]int)
	for phaseID := range dag.Nodes {
		inDegree[phaseID] = 0
	}

	for _, node := range dag.Nodes {
		for range node.Dependencies {
			inDegree[node.PhaseID]++
		}
	}

	// Queue of nodes with no dependencies
	queue := []string{}
	for phaseID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, phaseID)
		}
	}

	// Process nodes in topological order
	sorted := []string{}
	for len(queue) > 0 {
		// Dequeue
		phaseID := queue[0]
		queue = queue[1:]
		sorted = append(sorted, phaseID)

		// Process dependent phases
		for _, dependentID := range dag.Edges[phaseID] {
			inDegree[dependentID]--
			if inDegree[dependentID] == 0 {
				queue = append(queue, dependentID)
			}
		}
	}

	// Check if all nodes were processed
	if len(sorted) != len(dag.Nodes) {
		return nil, fmt.Errorf("cycle detected: only %d of %d phases could be ordered", len(sorted), len(dag.Nodes))
	}

	return sorted, nil
}

// GetBatches returns phases grouped into parallel execution batches
func (dag *DAG) GetBatches() ([][]string, error) {
	sorted, err := dag.TopologicalSort()
	if err != nil {
		return nil, err
	}

	// Track which phases have been scheduled
	scheduled := make(map[string]bool)
	batches := [][]string{}

	for len(scheduled) < len(dag.Nodes) {
		batch := []string{}

		// Find all phases whose dependencies are satisfied
		for _, phaseID := range sorted {
			if scheduled[phaseID] {
				continue
			}

			// Check if all dependencies are satisfied
			canSchedule := true
			node := dag.Nodes[phaseID]
			for _, depID := range node.Dependencies {
				if !scheduled[depID] {
					canSchedule = false
					break
				}
			}

			if canSchedule {
				batch = append(batch, phaseID)
			}
		}

		if len(batch) == 0 {
			// This shouldn't happen if DAG is valid
			return nil, fmt.Errorf("unable to schedule remaining phases")
		}

		// Mark batch as scheduled
		for _, phaseID := range batch {
			scheduled[phaseID] = true
		}

		batches = append(batches, batch)
	}

	return batches, nil
}

// GetRootPhases returns phases with no dependencies
func (dag *DAG) GetRootPhases() []string {
	roots := []string{}
	for phaseID, node := range dag.Nodes {
		if len(node.Dependencies) == 0 {
			roots = append(roots, phaseID)
		}
	}
	return roots
}

// GetLeafPhases returns phases that nothing depends on
func (dag *DAG) GetLeafPhases() []string {
	leaves := []string{}
	for phaseID := range dag.Nodes {
		if len(dag.Edges[phaseID]) == 0 {
			leaves = append(leaves, phaseID)
		}
	}
	return leaves
}
