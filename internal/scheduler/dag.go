package scheduler

import (
	"fmt"

	"github.com/crabcoder/crabcoder/pkg/model"
)

type DAG struct {
	tasks    map[string]*model.Task
	adj      map[string][]string // adjacency list: taskID → successors
	indegree map[string]int      // indegree for Kahn's algorithm
}

func NewDAG() *DAG {
	return &DAG{
		tasks:    make(map[string]*model.Task),
		adj:      make(map[string][]string),
		indegree: make(map[string]int),
	}
}

func (d *DAG) AddTask(task *model.Task) error {
	if _, ok := d.tasks[task.ID]; ok {
		return fmt.Errorf("duplicate task ID: %s", task.ID)
	}
	d.tasks[task.ID] = task
	d.adj[task.ID] = []string{}
	if _, ok := d.indegree[task.ID]; !ok {
		d.indegree[task.ID] = 0
	}
	return nil
}

// Build validates dependencies and detects cycles.
func (d *DAG) Build() error {
	// Validate all DependsOn reference real tasks
	for id, task := range d.tasks {
		for _, dep := range task.DependsOn {
			if _, ok := d.tasks[dep]; !ok {
				return fmt.Errorf("task %q depends on unknown task %q", id, dep)
			}
		}
	}

	return d.detectCycle()
}

// detectCycle uses Kahn's algorithm to check for cycles.
func (d *DAG) detectCycle() error {
	// Build indegree and adjacency from DependsOn
	indeg := make(map[string]int)
	adj := make(map[string][]string)

	for id := range d.tasks {
		indeg[id] = 0
		adj[id] = []string{}
	}

	for id, task := range d.tasks {
		indeg[id] = len(task.DependsOn)
		for _, dep := range task.DependsOn {
			adj[dep] = append(adj[dep], id)
		}
	}

	// Store for execution
	d.indegree = indeg
	d.adj = adj

	// Kahn's algorithm
	queue := []string{}
	for id, deg := range d.indegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	visited := 0
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		visited++

		for _, succ := range d.adj[current] {
			d.indegree[succ]--
			if d.indegree[succ] == 0 {
				queue = append(queue, succ)
			}
		}
	}

	if visited != len(d.tasks) {
		return fmt.Errorf("cycle detected: %d tasks visited out of %d", visited, len(d.tasks))
	}

	return nil
}

// ReadyTasks returns tasks whose dependencies are satisfied (indegree == 0
// after re-running Kahn's with current state).
func (d *DAG) ReadyTasks(completed map[string]bool) []string {
	// Recompute indegree based on which tasks haven't been completed
	indeg := make(map[string]int)
	for id := range d.tasks {
		if completed[id] {
			continue
		}
		indeg[id] = 0
		for _, dep := range d.tasks[id].DependsOn {
			if !completed[dep] {
				indeg[id]++
			}
		}
	}

	var ready []string
	for id, deg := range indeg {
		if deg == 0 {
			ready = append(ready, id)
		}
	}
	return ready
}

func (d *DAG) Task(id string) *model.Task {
	return d.tasks[id]
}

func (d *DAG) Tasks() map[string]*model.Task {
	return d.tasks
}
