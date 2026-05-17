package memory

import (
	"strings"
	"testing"
)

func TestKnowledgeGraph_Dependencies(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.AddNode("a", "pkg")
	kg.AddNode("b", "pkg")
	kg.AddNode("c", "pkg")
	kg.AddEdge("a", "b", "imports")
	kg.AddEdge("a", "c", "imports")
	kg.AddEdge("b", "c", "imports")

	deps := kg.Dependencies("a")
	if len(deps) != 2 {
		t.Errorf("expected 2 deps, got %d", len(deps))
	}
}

func TestKnowledgeGraph_Dependents(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.AddNode("a", "pkg")
	kg.AddNode("b", "pkg")
	kg.AddNode("c", "pkg")
	kg.AddEdge("a", "c", "imports")
	kg.AddEdge("b", "c", "imports")

	deps := kg.Dependents("c")
	if len(deps) != 2 {
		t.Errorf("expected 2 dependents, got %d", len(deps))
	}
}

func TestKnowledgeGraph_TransitiveDeps(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.AddNode("a", "pkg")
	kg.AddNode("b", "pkg")
	kg.AddNode("c", "pkg")
	kg.AddEdge("a", "b", "imports")
	kg.AddEdge("b", "c", "imports")

	transitive := kg.TransitiveDeps("a")
	if len(transitive) != 2 {
		t.Errorf("expected 2 transitive deps, got %d: %v", len(transitive), transitive)
	}
}

func TestKnowledgeGraph_FindPath(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.AddNode("a", "pkg")
	kg.AddNode("b", "pkg")
	kg.AddNode("c", "pkg")
	kg.AddEdge("a", "b", "imports")
	kg.AddEdge("b", "c", "imports")

	path := kg.FindPath("a", "c")
	if path == nil {
		t.Fatal("expected path, got nil")
	}
	if len(path) != 3 || path[0] != "a" || path[2] != "c" {
		t.Errorf("expected [a b c], got %v", path)
	}
}

func TestKnowledgeGraph_DetectCycles(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.AddNode("a", "pkg")
	kg.AddNode("b", "pkg")
	kg.AddEdge("a", "b", "imports")
	kg.AddEdge("b", "a", "imports")

	cycles := kg.DetectCycles()
	if len(cycles) == 0 {
		t.Error("expected cycles, got none")
	}
}

func TestKnowledgeGraph_TopoSort(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.AddNode("a", "pkg")
	kg.AddNode("b", "pkg")
	kg.AddNode("c", "pkg")
	kg.AddEdge("a", "b", "imports")
	kg.AddEdge("b", "c", "imports")

	order, err := kg.TopologicalSort()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(order) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(order))
	}
	// a before b, b before c
	posA := indexOf(order, "a")
	posB := indexOf(order, "b")
	posC := indexOf(order, "c")
	if posA > posB || posB > posC {
		t.Errorf("wrong order: %v", order)
	}
}

func TestKnowledgeGraph_TopoSortCycle(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.AddNode("a", "pkg")
	kg.AddNode("b", "pkg")
	kg.AddEdge("a", "b", "imports")
	kg.AddEdge("b", "a", "imports")

	_, err := kg.TopologicalSort()
	if err == nil {
		t.Error("expected cycle error")
	}
}

func TestKnowledgeGraph_Subgraph(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.AddNode("a", "pkg")
	kg.AddNode("b", "pkg")
	kg.AddNode("c", "pkg")
	kg.AddEdge("a", "b", "imports")
	kg.AddEdge("b", "c", "imports")

	sub := kg.Subgraph("b", 1)
	if sub.NodeCount() == 0 {
		t.Error("expected subgraph with nodes")
	}
}

func TestKnowledgeGraph_Query(t *testing.T) {
	kg := NewKnowledgeGraph()
	kg.AddNode("a", "pkg")
	kg.AddNode("b", "pkg")
	kg.AddEdge("a", "b", "imports")

	r, err := kg.Query("deps:a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(r.Result, "b") {
		t.Errorf("expected 'b' in result, got %q", r.Result)
	}

	r, err = kg.Query("stats")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestKnowledgeGraph_QueryUnknown(t *testing.T) {
	kg := NewKnowledgeGraph()
	_, err := kg.Query("unknown")
	if err == nil {
		t.Error("expected error for unknown query")
	}
}

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}
