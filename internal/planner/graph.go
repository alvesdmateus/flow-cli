package planner

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// DependencyGraph represents a directed acyclic graph for task dependencies.
type DependencyGraph struct {
	nodes    map[string]*GraphNode
	edges    map[string][]string // from -> to
	reverse  map[string][]string // to -> from (for reverse lookup)
	mu       sync.RWMutex
}

// GraphNode represents a node in the dependency graph.
type GraphNode struct {
	ID       string
	Task     *Task
	InDegree int // Number of incoming edges (dependencies)
	Level    int // Level in the graph (for parallel execution)
}

// NewDependencyGraph creates a new dependency graph.
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes:   make(map[string]*GraphNode),
		edges:   make(map[string][]string),
		reverse: make(map[string][]string),
	}
}

// AddTask adds a task to the graph.
func (g *DependencyGraph) AddTask(task *Task) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.nodes[task.ID]; !exists {
		g.nodes[task.ID] = &GraphNode{
			ID:   task.ID,
			Task: task,
		}
	}

	// Add edges for dependencies
	for _, dep := range task.Dependencies {
		g.addEdgeLocked(dep, task.ID)
	}
}

// AddEdge adds a dependency edge (from must complete before to).
func (g *DependencyGraph) AddEdge(from, to string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.addEdgeLocked(from, to)
}

func (g *DependencyGraph) addEdgeLocked(from, to string) error {
	// Ensure nodes exist
	if _, exists := g.nodes[from]; !exists {
		g.nodes[from] = &GraphNode{ID: from}
	}
	if _, exists := g.nodes[to]; !exists {
		g.nodes[to] = &GraphNode{ID: to}
	}

	// Check for self-loop
	if from == to {
		return errors.New("self-loop detected")
	}

	// Add edge
	g.edges[from] = append(g.edges[from], to)
	g.reverse[to] = append(g.reverse[to], from)
	g.nodes[to].InDegree++

	return nil
}

// RemoveTask removes a task from the graph.
func (g *DependencyGraph) RemoveTask(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Remove outgoing edges
	for _, to := range g.edges[id] {
		g.nodes[to].InDegree--
		// Remove from reverse map
		for i, from := range g.reverse[to] {
			if from == id {
				g.reverse[to] = append(g.reverse[to][:i], g.reverse[to][i+1:]...)
				break
			}
		}
	}
	delete(g.edges, id)

	// Remove incoming edges
	for _, from := range g.reverse[id] {
		for i, to := range g.edges[from] {
			if to == id {
				g.edges[from] = append(g.edges[from][:i], g.edges[from][i+1:]...)
				break
			}
		}
	}
	delete(g.reverse, id)

	// Remove node
	delete(g.nodes, id)
}

// HasCycle checks if the graph has a cycle.
func (g *DependencyGraph) HasCycle() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	for id := range g.nodes {
		if g.hasCycleDFS(id, visited, recStack) {
			return true
		}
	}
	return false
}

func (g *DependencyGraph) hasCycleDFS(id string, visited, recStack map[string]bool) bool {
	if recStack[id] {
		return true
	}
	if visited[id] {
		return false
	}

	visited[id] = true
	recStack[id] = true

	for _, neighbor := range g.edges[id] {
		if g.hasCycleDFS(neighbor, visited, recStack) {
			return true
		}
	}

	recStack[id] = false
	return false
}

// TopologicalSort returns tasks in execution order.
func (g *DependencyGraph) TopologicalSort() ([]*Task, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.HasCycle() {
		return nil, errors.New("graph has a cycle, cannot perform topological sort")
	}

	// Kahn's algorithm
	inDegree := make(map[string]int)
	for id, node := range g.nodes {
		inDegree[id] = node.InDegree
	}

	// Find all nodes with no incoming edges
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	var result []*Task
	for len(queue) > 0 {
		// Pop front
		id := queue[0]
		queue = queue[1:]

		if node := g.nodes[id]; node != nil && node.Task != nil {
			result = append(result, node.Task)
		}

		// Reduce in-degree of neighbors
		for _, neighbor := range g.edges[id] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	return result, nil
}

// GetExecutionLevels groups tasks by execution level (for parallel execution).
func (g *DependencyGraph) GetExecutionLevels() ([][]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.HasCycle() {
		return nil, errors.New("graph has a cycle")
	}

	// Calculate levels
	levels := make(map[string]int)
	for id := range g.nodes {
		g.calculateLevel(id, levels)
	}

	// Group by level
	maxLevel := 0
	for _, level := range levels {
		if level > maxLevel {
			maxLevel = level
		}
	}

	result := make([][]string, maxLevel+1)
	for id, level := range levels {
		result[level] = append(result[level], id)
	}

	// Sort each level for deterministic order
	for i := range result {
		sort.Strings(result[i])
	}

	return result, nil
}

func (g *DependencyGraph) calculateLevel(id string, levels map[string]int) int {
	if level, ok := levels[id]; ok {
		return level
	}

	// Base case: no dependencies
	if len(g.reverse[id]) == 0 {
		levels[id] = 0
		return 0
	}

	// Calculate as max of dependencies + 1
	maxDepLevel := 0
	for _, dep := range g.reverse[id] {
		depLevel := g.calculateLevel(dep, levels)
		if depLevel >= maxDepLevel {
			maxDepLevel = depLevel + 1
		}
	}

	levels[id] = maxDepLevel
	return maxDepLevel
}

// GetDependencies returns direct dependencies of a task.
func (g *DependencyGraph) GetDependencies(id string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.reverse[id]
}

// GetDependents returns tasks that depend on this task.
func (g *DependencyGraph) GetDependents(id string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.edges[id]
}

// GetAllDependencies returns all transitive dependencies.
func (g *DependencyGraph) GetAllDependencies(id string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	var result []string

	g.collectDependencies(id, visited, &result)
	return result
}

func (g *DependencyGraph) collectDependencies(id string, visited map[string]bool, result *[]string) {
	for _, dep := range g.reverse[id] {
		if !visited[dep] {
			visited[dep] = true
			*result = append(*result, dep)
			g.collectDependencies(dep, visited, result)
		}
	}
}

// GetCriticalPath returns the longest path through the graph.
func (g *DependencyGraph) GetCriticalPath() ([]string, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if g.HasCycle() {
		return nil, errors.New("graph has a cycle")
	}

	// Find all leaf nodes (no outgoing edges)
	var leafNodes []string
	for id := range g.nodes {
		if len(g.edges[id]) == 0 {
			leafNodes = append(leafNodes, id)
		}
	}

	if len(leafNodes) == 0 {
		return nil, nil
	}

	// Find longest path to each leaf
	var longestPath []string
	for _, leaf := range leafNodes {
		path := g.findLongestPathTo(leaf)
		if len(path) > len(longestPath) {
			longestPath = path
		}
	}

	return longestPath, nil
}

func (g *DependencyGraph) findLongestPathTo(target string) []string {
	memo := make(map[string][]string)
	return g.longestPathDFS(target, memo)
}

func (g *DependencyGraph) longestPathDFS(id string, memo map[string][]string) []string {
	if path, ok := memo[id]; ok {
		return path
	}

	deps := g.reverse[id]
	if len(deps) == 0 {
		memo[id] = []string{id}
		return memo[id]
	}

	var longestPrefix []string
	for _, dep := range deps {
		prefix := g.longestPathDFS(dep, memo)
		if len(prefix) > len(longestPrefix) {
			longestPrefix = prefix
		}
	}

	result := append([]string{}, longestPrefix...)
	result = append(result, id)
	memo[id] = result
	return result
}

// Validate checks if the graph is valid.
func (g *DependencyGraph) Validate() error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Check for cycles
	if g.HasCycle() {
		return errors.New("graph contains a cycle")
	}

	// Check for missing dependencies
	for id := range g.nodes {
		for _, dep := range g.reverse[id] {
			if _, exists := g.nodes[dep]; !exists {
				return fmt.Errorf("task %s depends on missing task %s", id, dep)
			}
		}
	}

	return nil
}

// Size returns the number of nodes in the graph.
func (g *DependencyGraph) Size() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// GetRoots returns nodes with no dependencies.
func (g *DependencyGraph) GetRoots() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var roots []string
	for id, node := range g.nodes {
		if node.InDegree == 0 {
			roots = append(roots, id)
		}
	}
	sort.Strings(roots)
	return roots
}

// GetLeaves returns nodes with no dependents.
func (g *DependencyGraph) GetLeaves() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var leaves []string
	for id := range g.nodes {
		if len(g.edges[id]) == 0 {
			leaves = append(leaves, id)
		}
	}
	sort.Strings(leaves)
	return leaves
}

// Clone creates a deep copy of the graph.
func (g *DependencyGraph) Clone() *DependencyGraph {
	g.mu.RLock()
	defer g.mu.RUnlock()

	clone := NewDependencyGraph()

	for id, node := range g.nodes {
		clone.nodes[id] = &GraphNode{
			ID:       node.ID,
			Task:     node.Task,
			InDegree: node.InDegree,
			Level:    node.Level,
		}
	}

	for from, tos := range g.edges {
		clone.edges[from] = append([]string{}, tos...)
	}

	for to, froms := range g.reverse {
		clone.reverse[to] = append([]string{}, froms...)
	}

	return clone
}

// String returns a string representation of the graph.
func (g *DependencyGraph) String() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var result string
	result += fmt.Sprintf("DependencyGraph (%d nodes):\n", len(g.nodes))

	for id := range g.nodes {
		deps := g.reverse[id]
		if len(deps) == 0 {
			result += fmt.Sprintf("  %s (root)\n", id)
		} else {
			result += fmt.Sprintf("  %s <- %v\n", id, deps)
		}
	}

	return result
}
