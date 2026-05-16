package model

import "time"

type TaskStatus int

const (
	TaskPending    TaskStatus = iota
	TaskRunning
	TaskCompleted
	TaskFailed
	TaskCancelled
)

func (s TaskStatus) String() string {
	switch s {
	case TaskPending:
		return "pending"
	case TaskRunning:
		return "running"
	case TaskCompleted:
		return "completed"
	case TaskFailed:
		return "failed"
	case TaskCancelled:
		return "cancelled"
	default:
		return "unknown"
	}
}

type Task struct {
	ID          string
	Description string
	DependsOn   []string
	Status      TaskStatus
	Tool        string
	ToolArgs    map[string]any
	Result      *TaskResult
	Error       string
	CreatedAt   time.Time
	StartedAt   time.Time
	CompletedAt time.Time
}

type TaskResult struct {
	Success bool
	Output  string
	Error   string
}

type FileChange struct {
	Path   string
	Action string // created, modified, deleted
	Lines  int
}
