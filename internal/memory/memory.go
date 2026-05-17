package memory

import (
	"context"
	"fmt"
	"strings"

	crabcontext "github.com/crabcoder/crabcoder/internal/context"
)

// Embedder provides semantic embeddings for text.
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float64, error)
}

type Memory interface {
	Store(key, value string) error
	Recall(query string, k int) ([]SearchResult, error)
}

type MemoryManager struct {
	shortTerm      *crabcontext.MessageHistory
	longTerm       VectorStore
	knowledgeGraph *KnowledgeGraph
	embedder       Embedder
}

func NewMemoryManager() *MemoryManager {
	return &MemoryManager{
		shortTerm:      crabcontext.NewMessageHistory(),
		longTerm:       NewInMemoryVectorStore(),
		knowledgeGraph: NewKnowledgeGraph(),
	}
}

// NewMemoryManagerWithStore creates a manager with a custom vector store.
func NewMemoryManagerWithStore(vs VectorStore) *MemoryManager {
	return &MemoryManager{
		shortTerm:      crabcontext.NewMessageHistory(),
		longTerm:       vs,
		knowledgeGraph: NewKnowledgeGraph(),
	}
}

// SetEmbedder configures an LLM-based embedder for semantic search.
// When set, embeddings use the LLM API; otherwise falls back to trigram bag-of-words.
func (m *MemoryManager) SetEmbedder(e Embedder) {
	m.embedder = e
}

func (m *MemoryManager) Store(key, value string) error {
	if value == "" {
		return nil
	}
	embedding := m.embed(value)
	return m.longTerm.Store(key, embedding, map[string]any{
		"key":   key,
		"value": value,
	})
}

func (m *MemoryManager) Recall(query string, k int) ([]SearchResult, error) {
	if query == "" || k <= 0 {
		return nil, nil
	}
	embedding := m.embed(query)
	return m.longTerm.Search(embedding, k)
}

func (m *MemoryManager) embed(text string) []float64 {
	if m.embedder != nil {
		vec, err := m.embedder.Embed(context.Background(), text)
		if err == nil && len(vec) > 0 {
			return vec
		}
	}
	return bagOfWordsEmbed(text, 128)
}

// IndexCompressed stores compressed message summaries in the vector store for RAG retrieval.
func (m *MemoryManager) IndexCompressed(compressed []string) {
	for i, text := range compressed {
		if text == "" {
			continue
		}
		key := fmt.Sprintf("compressed-%d-%s", i, hashString(text))
		m.Store(key, text)
	}
}

// RetrieveContext searches for relevant context snippets given the current user message.
func (m *MemoryManager) RetrieveContext(query string, k int) string {
	results, err := m.Recall(query, k)
	if err != nil || len(results) == 0 {
		return ""
	}

	var parts []string
	for _, r := range results {
		if val, ok := r.Metadata["value"].(string); ok && val != "" {
			parts = append(parts, val)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "[Retrieved context]\n" + strings.Join(parts, "\n---\n")
}

func (m *MemoryManager) ShortTerm() *crabcontext.MessageHistory {
	return m.shortTerm
}

// bagOfWordsEmbed creates a simple character n-gram embedding (no external API needed).
func bagOfWordsEmbed(text string, dim int) []float64 {
	vec := make([]float64, dim)
	n := 3 // trigram
	runes := []rune(strings.ToLower(text))
	if len(runes) < n {
		// Pad with zeros if text is too short
		for i := 0; i < len(runes) && i < dim; i++ {
			vec[i%dim] += float64(runes[i]) / 256.0
		}
		return vec
	}
	for i := 0; i <= len(runes)-n; i++ {
		// Hash the trigram to a position in the vector
		h := (int(runes[i])*31+int(runes[i+1]))*31 + int(runes[i+2])
		idx := h % dim
		if idx < 0 {
			idx = -idx
		}
		vec[idx] += 1.0
	}
	// Normalize
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	if norm > 0 {
		for i := range vec {
			vec[i] /= norm
		}
	}
	return vec
}

func hashString(s string) string {
	var h uint64
	for _, r := range s {
		h = h*31 + uint64(r)
	}
	return fmt.Sprintf("%x", h)
}
