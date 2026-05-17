package memory

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

type KnowledgeGraph struct {
	nodes map[string]*KGNode
}

type KGNode struct {
	ID    string         `json:"id"`
	Type  string         `json:"type"`
	Props map[string]any `json:"props,omitempty"`
	Edges []*KGEdge      `json:"edges,omitempty"`
}

type KGEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Type string `json:"type"`
}

func NewKnowledgeGraph() *KnowledgeGraph {
	return &KnowledgeGraph{nodes: make(map[string]*KGNode)}
}

func (kg *KnowledgeGraph) AddNode(id, nodeType string) *KGNode {
	n := &KGNode{ID: id, Type: nodeType, Props: make(map[string]any)}
	kg.nodes[id] = n
	return n
}

func (kg *KnowledgeGraph) GetNode(id string) *KGNode {
	return kg.nodes[id]
}

func (kg *KnowledgeGraph) HasNode(id string) bool {
	_, ok := kg.nodes[id]
	return ok
}

func (kg *KnowledgeGraph) NodeCount() int {
	return len(kg.nodes)
}

func (kg *KnowledgeGraph) EdgeCount() int {
	count := 0
	for _, n := range kg.nodes {
		count += len(n.Edges)
	}
	return count
}

func (kg *KnowledgeGraph) AddEdge(from, to, edgeType string) {
	if n, ok := kg.nodes[from]; ok {
		// Deduplicate
		for _, e := range n.Edges {
			if e.To == to && e.Type == edgeType {
				return
			}
		}
		n.Edges = append(n.Edges, &KGEdge{From: from, To: to, Type: edgeType})
	}
}

// Dependencies returns all nodes that `id` directly depends on (outgoing edges).
func (kg *KnowledgeGraph) Dependencies(id string) []string {
	n := kg.nodes[id]
	if n == nil {
		return nil
	}
	var deps []string
	for _, e := range n.Edges {
		deps = append(deps, e.To)
	}
	sort.Strings(deps)
	return deps
}

// Dependents returns all nodes that directly depend on `id` (incoming edges).
func (kg *KnowledgeGraph) Dependents(id string) []string {
	var result []string
	for _, n := range kg.nodes {
		for _, e := range n.Edges {
			if e.To == id {
				result = append(result, n.ID)
				break
			}
		}
	}
	sort.Strings(result)
	return result
}

// TransitiveDeps returns the full transitive closure of dependencies from `id`.
func (kg *KnowledgeGraph) TransitiveDeps(id string) []string {
	visited := make(map[string]bool)
	var walk func(nid string)
	walk = func(nid string) {
		if visited[nid] {
			return
		}
		visited[nid] = true
		for _, dep := range kg.Dependencies(nid) {
			walk(dep)
		}
	}
	walk(id)
	delete(visited, id) // exclude self
	var result []string
	for k := range visited {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

// TransitiveDependents returns all nodes that transitively depend on `id`.
func (kg *KnowledgeGraph) TransitiveDependents(id string) []string {
	visited := make(map[string]bool)
	var walk func(nid string)
	walk = func(nid string) {
		if visited[nid] {
			return
		}
		visited[nid] = true
		for _, dep := range kg.Dependents(nid) {
			walk(dep)
		}
	}
	walk(id)
	delete(visited, id)
	var result []string
	for k := range visited {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

// FindPath returns the shortest path from `from` to `to` using BFS.
func (kg *KnowledgeGraph) FindPath(from, to string) []string {
	if from == to {
		return []string{from}
	}
	visited := map[string]bool{from: true}
	parent := map[string]string{}
	queue := []string{from}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == to {
			// Reconstruct path
			var path []string
			for cur := to; cur != ""; cur = parent[cur] {
				path = append([]string{cur}, path...)
			}
			return path
		}
		for _, dep := range kg.Dependencies(current) {
			if !visited[dep] {
				visited[dep] = true
				parent[dep] = current
				queue = append(queue, dep)
			}
		}
		// Also check reverse: dependents
		for _, dep := range kg.Dependents(current) {
			if !visited[dep] {
				visited[dep] = true
				parent[dep] = current
				queue = append(queue, dep)
			}
		}
	}
	return nil
}

// Subgraph returns the induced subgraph containing `id` plus its neighborhood
// up to `depth` hops away.
func (kg *KnowledgeGraph) Subgraph(id string, depth int) *KnowledgeGraph {
	if depth < 0 {
		depth = 1
	}
	sub := NewKnowledgeGraph()
	visited := make(map[string]bool)
	var walk func(nid string, d int)
	walk = func(nid string, d int) {
		if d < 0 || visited[nid] {
			return
		}
		visited[nid] = true
		orig := kg.nodes[nid]
		if orig == nil {
			return
		}
		sub.AddNode(nid, orig.Type)
		for _, e := range orig.Edges {
			sub.AddEdge(e.From, e.To, e.Type)
			walk(e.To, d-1)
		}
		// Also traverse incoming edges for neighborhood
		for _, other := range kg.nodes {
			for _, e := range other.Edges {
				if e.To == nid {
					if !visited[e.From] {
						sub.AddNode(e.From, other.Type)
						sub.AddEdge(e.From, e.To, e.Type)
					}
				}
			}
		}
	}
	walk(id, depth)
	return sub
}

// TopologicalSort returns nodes in dependency order (deps first).
func (kg *KnowledgeGraph) TopologicalSort() ([]string, error) {
	inDegree := make(map[string]int)
	for id := range kg.nodes {
		if _, ok := inDegree[id]; !ok {
			inDegree[id] = 0
		}
		for _, e := range kg.nodes[id].Edges {
			inDegree[e.To]++
		}
	}

	var queue []string
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	var result []string
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		result = append(result, id)
		for _, e := range kg.nodes[id].Edges {
			inDegree[e.To]--
			if inDegree[e.To] == 0 {
				queue = append(queue, e.To)
			}
		}
	}

	if len(result) != len(kg.nodes) {
		return result, fmt.Errorf("cycle detected: %d nodes unreachable", len(kg.nodes)-len(result))
	}
	return result, nil
}

// DetectCycles returns all simple cycles found via DFS.
func (kg *KnowledgeGraph) DetectCycles() [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	stack := make(map[string]bool)
	var path []string

	var dfs func(id string)
	dfs = func(id string) {
		visited[id] = true
		stack[id] = true
		path = append(path, id)

		for _, e := range kg.nodes[id].Edges {
			if stack[e.To] {
				// Found a cycle — extract the cycle from path
				for i, p := range path {
					if p == e.To {
						cycle := make([]string, len(path)-i)
						copy(cycle, path[i:])
						cycle = append(cycle, e.To)
						cycles = append(cycles, cycle)
						break
					}
				}
			} else if !visited[e.To] {
				dfs(e.To)
			}
		}
		path = path[:len(path)-1]
		stack[id] = false
	}

	for id := range kg.nodes {
		if !visited[id] {
			dfs(id)
		}
	}
	return cycles
}

// Stats returns summary statistics for the graph.
func (kg *KnowledgeGraph) Stats() map[string]any {
	types := make(map[string]int)
	edgeTypes := make(map[string]int)
	for _, n := range kg.nodes {
		types[n.Type]++
		for _, e := range n.Edges {
			edgeTypes[e.Type]++
		}
	}
	return map[string]any{
		"nodes":      kg.NodeCount(),
		"edges":      kg.EdgeCount(),
		"node_types": types,
		"edge_types": edgeTypes,
	}
}

// MarshalJSON serializes the graph as structured JSON.
func (kg *KnowledgeGraph) MarshalJSON() ([]byte, error) {
	nodes := make([]*KGNode, 0, len(kg.nodes))
	for _, n := range kg.nodes {
		nodes = append(nodes, n)
	}
	return json.Marshal(struct {
		Nodes []*KGNode `json:"nodes"`
	}{Nodes: nodes})
}

// BuildFromImports analyzes Go import relationships and builds a dependency graph.
// imports is a map of package -> []importedPackages.
func (kg *KnowledgeGraph) BuildFromImports(imports map[string][]string) {
	for pkg, deps := range imports {
		if !kg.HasNode(pkg) {
			kg.AddNode(pkg, "package")
		}
		for _, dep := range deps {
			depName := strings.Trim(dep, "\"")
			if !kg.HasNode(depName) {
				kg.AddNode(depName, "package")
			}
			kg.AddEdge(pkg, depName, "imports")
		}
	}
}

// QueryResult is the structured output of a knowledge graph query.
type QueryResult struct {
	Query  string `json:"query"`
	Result string `json:"result"`
	Data   any    `json:"data,omitempty"`
}

// Query runs a named query against the graph. Supported queries:
// deps:<id>, dependents:<id>, path:<from>:<to>, cycles, stats, topo
func (kg *KnowledgeGraph) Query(query string) (*QueryResult, error) {
	parts := strings.SplitN(query, ":", 2)
	op := parts[0]

	switch op {
	case "deps":
		if len(parts) < 2 {
			return nil, fmt.Errorf("deps requires a node id")
		}
		id := parts[1]
		result := kg.Dependencies(id)
		transitive := kg.TransitiveDeps(id)
		return &QueryResult{
			Query:  query,
			Result: fmt.Sprintf("%s depends on: %s (transitive: %s)", id, strings.Join(result, ", "), strings.Join(transitive, ", ")),
			Data:   map[string][]string{"direct": result, "transitive": transitive},
		}, nil

	case "dependents":
		if len(parts) < 2 {
			return nil, fmt.Errorf("dependents requires a node id")
		}
		id := parts[1]
		result := kg.Dependents(id)
		transitive := kg.TransitiveDependents(id)
		return &QueryResult{
			Query:  query,
			Result: fmt.Sprintf("Depends on %s: %s (transitive: %s)", id, strings.Join(result, ", "), strings.Join(transitive, ", ")),
			Data:   map[string][]string{"direct": result, "transitive": transitive},
		}, nil

	case "path":
		args := strings.SplitN(parts[1], ":", 2)
		if len(args) < 2 {
			return nil, fmt.Errorf("path requires from:to")
		}
		path := kg.FindPath(args[0], args[1])
		if path == nil {
			return &QueryResult{Query: query, Result: fmt.Sprintf("No path from %s to %s", args[0], args[1])}, nil
		}
		return &QueryResult{
			Query:  query,
			Result: fmt.Sprintf("Path: %s", strings.Join(path, " → ")),
			Data:   path,
		}, nil

	case "cycles":
		cycles := kg.DetectCycles()
		if len(cycles) == 0 {
			return &QueryResult{Query: query, Result: "No cycles detected"}, nil
		}
		var strs []string
		for _, c := range cycles {
			strs = append(strs, strings.Join(c, " → "))
		}
		return &QueryResult{
			Query:  query,
			Result: fmt.Sprintf("Found %d cycles:\n%s", len(cycles), strings.Join(strs, "\n")),
			Data:   cycles,
		}, nil

	case "stats":
		s := kg.Stats()
		return &QueryResult{
			Query:  query,
			Result: fmt.Sprintf("Nodes: %v, Edges: %v", s["nodes"], s["edges"]),
			Data:   s,
		}, nil

	case "topo":
		order, err := kg.TopologicalSort()
		if err != nil {
			return &QueryResult{Query: query, Result: err.Error()}, nil
		}
		return &QueryResult{
			Query:  query,
			Result: fmt.Sprintf("Topological order (%d nodes):\n%s", len(order), strings.Join(order, "\n")),
			Data:   order,
		}, nil

	default:
		return nil, fmt.Errorf("unknown query: %q. Valid: deps, dependents, path, cycles, stats, topo", op)
	}
}
