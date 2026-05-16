package memory

import "github.com/crabcoder/crabcoder/internal/context"

type Memory interface {
	Store(key, value string) error
	Recall(query string) (string, error)
}

type MemoryManager struct {
	shortTerm      *context.MessageHistory
	longTerm       VectorStore
	knowledgeGraph *KnowledgeGraph
}

func NewMemoryManager() *MemoryManager {
	return &MemoryManager{
		shortTerm:      context.NewMessageHistory(),
		longTerm:       nil,
		knowledgeGraph: NewKnowledgeGraph(),
	}
}

func (m *MemoryManager) Store(key, value string) error {
	return nil
}

func (m *MemoryManager) Recall(query string) (string, error) {
	return "", nil
}

func (m *MemoryManager) BuildIndex() error {
	return nil
}
