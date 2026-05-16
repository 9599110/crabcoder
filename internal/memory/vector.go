package memory

import (
	"math"
	"sort"
	"sync"
)

// VectorStore persists embeddings and supports similarity search.
type VectorStore interface {
	Store(key string, embedding []float64, metadata map[string]any) error
	Search(embedding []float64, k int) ([]SearchResult, error)
	Delete(key string) error
	Len() int
}

// SearchResult represents a single vector search result.
type SearchResult struct {
	Key      string
	Score    float64
	Metadata map[string]any
}

// InMemoryVectorStore is a simple in-memory vector store using cosine similarity.
type InMemoryVectorStore struct {
	mu     sync.RWMutex
	items  map[string]*vectorEntry
}

type vectorEntry struct {
	key       string
	embedding []float64
	metadata  map[string]any
}

// NewInMemoryVectorStore creates a new in-memory vector store.
func NewInMemoryVectorStore() *InMemoryVectorStore {
	return &InMemoryVectorStore{
		items: make(map[string]*vectorEntry),
	}
}

// Store adds or updates a vector entry.
func (s *InMemoryVectorStore) Store(key string, embedding []float64, metadata map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[key] = &vectorEntry{
		key:       key,
		embedding: embedding,
		metadata:  metadata,
	}
	return nil
}

// Search finds the k nearest neighbors by cosine similarity.
func (s *InMemoryVectorStore) Search(embedding []float64, k int) ([]SearchResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.items) == 0 {
		return nil, nil
	}

	type scored struct {
		entry *vectorEntry
		score float64
	}

	var results []scored
	for _, entry := range s.items {
		score := cosineSimilarity(embedding, entry.embedding)
		results = append(results, scored{entry: entry, score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	if k > len(results) {
		k = len(results)
	}

	out := make([]SearchResult, k)
	for i := 0; i < k; i++ {
		out[i] = SearchResult{
			Key:      results[i].entry.key,
			Score:    results[i].score,
			Metadata: results[i].entry.metadata,
		}
	}
	return out, nil
}

// Delete removes a vector entry by key.
func (s *InMemoryVectorStore) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.items, key)
	return nil
}

// Len returns the number of stored vectors.
func (s *InMemoryVectorStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// cosineSimilarity computes the cosine similarity between two vectors.
func cosineSimilarity(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}

	minLen := len(a)
	if len(b) < minLen {
		minLen = len(b)
	}

	var dot, normA, normB float64
	for i := 0; i < minLen; i++ {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
