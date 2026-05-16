package model

type TaskResult struct {
	Success bool
	Output  string
	Error   string
	Files   []FileChange
	Metrics map[string]any
}

type FileChange struct {
	Path   string
	Action string // created, modified, deleted
	Lines  int
}
