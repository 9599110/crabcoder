package scheduler

import (
	"testing"

	"github.com/crabcoder/crabcoder/pkg/model"
)

func TestDAGNoCycle(t *testing.T) {
	d := NewDAG()

	// Task1 → Task2 → Task3
	// Task1 → Task3
	d.AddTask(&model.Task{ID: "1", Description: "Task 1", DependsOn: []string{}})
	d.AddTask(&model.Task{ID: "2", Description: "Task 2", DependsOn: []string{"1"}})
	d.AddTask(&model.Task{ID: "3", Description: "Task 3", DependsOn: []string{"1", "2"}})

	if err := d.Build(); err != nil {
		t.Fatalf("expected no cycle: %v", err)
	}
}

func TestDAGCycleDetection(t *testing.T) {
	d := NewDAG()

	// 1 → 2 → 3 → 1 (cycle)
	d.AddTask(&model.Task{ID: "1", Description: "Task 1", DependsOn: []string{"3"}})
	d.AddTask(&model.Task{ID: "2", Description: "Task 2", DependsOn: []string{"1"}})
	d.AddTask(&model.Task{ID: "3", Description: "Task 3", DependsOn: []string{"2"}})

	err := d.Build()
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
}

func TestDAGUnknownDependency(t *testing.T) {
	d := NewDAG()

	d.AddTask(&model.Task{ID: "1", Description: "Task 1", DependsOn: []string{"nonexistent"}})

	err := d.Build()
	if err == nil {
		t.Fatal("expected unknown dependency error")
	}
}

func TestDAGReadyTasks(t *testing.T) {
	d := NewDAG()

	// 1 (no deps) → 2, 3 → 4
	d.AddTask(&model.Task{ID: "1", Description: "Task 1", DependsOn: []string{}})
	d.AddTask(&model.Task{ID: "2", Description: "Task 2", DependsOn: []string{"1"}})
	d.AddTask(&model.Task{ID: "3", Description: "Task 3", DependsOn: []string{"1"}})
	d.AddTask(&model.Task{ID: "4", Description: "Task 4", DependsOn: []string{"2", "3"}})

	if err := d.Build(); err != nil {
		t.Fatalf("build: %v", err)
	}

	// Initially, only task 1 is ready
	ready := d.ReadyTasks(map[string]bool{})
	if len(ready) != 1 || ready[0] != "1" {
		t.Fatalf("expected [1], got %v", ready)
	}

	// After task 1 completes, tasks 2 and 3 are ready
	ready = d.ReadyTasks(map[string]bool{"1": true})
	if len(ready) != 2 {
		t.Fatalf("expected [2,3], got %v", ready)
	}

	// After tasks 1,2,3 complete, task 4 is ready
	ready = d.ReadyTasks(map[string]bool{"1": true, "2": true, "3": true})
	if len(ready) != 1 || ready[0] != "4" {
		t.Fatalf("expected [4], got %v", ready)
	}

	// All done → no ready tasks
	ready = d.ReadyTasks(map[string]bool{"1": true, "2": true, "3": true, "4": true})
	if len(ready) != 0 {
		t.Fatalf("expected [], got %v", ready)
	}
}

func TestDAGMultipleRoots(t *testing.T) {
	d := NewDAG()

	// Two independent chains: 1→3 and 2→4
	d.AddTask(&model.Task{ID: "1", Description: "Task 1", DependsOn: []string{}})
	d.AddTask(&model.Task{ID: "2", Description: "Task 2", DependsOn: []string{}})
	d.AddTask(&model.Task{ID: "3", Description: "Task 3", DependsOn: []string{"1"}})
	d.AddTask(&model.Task{ID: "4", Description: "Task 4", DependsOn: []string{"2"}})

	if err := d.Build(); err != nil {
		t.Fatalf("expected no cycle: %v", err)
	}

	ready := d.ReadyTasks(map[string]bool{})
	if len(ready) != 2 {
		t.Fatalf("expected [1,2], got %v", ready)
	}
}
