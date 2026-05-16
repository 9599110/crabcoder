package memory

type KnowledgeGraph struct {
	nodes map[string]*KGNode
}

type KGNode struct {
	ID    string
	Type  string
	Props map[string]any
	Edges []*KGEdge
}

type KGEdge struct {
	From string
	To   string
	Type string
}

func NewKnowledgeGraph() *KnowledgeGraph {
	return &KnowledgeGraph{nodes: make(map[string]*KGNode)}
}

func (kg *KnowledgeGraph) AddNode(id, nodeType string) *KGNode {
	n := &KGNode{ID: id, Type: nodeType, Props: make(map[string]any)}
	kg.nodes[id] = n
	return n
}

func (kg *KnowledgeGraph) AddEdge(from, to, edgeType string) {
	if n, ok := kg.nodes[from]; ok {
		n.Edges = append(n.Edges, &KGEdge{From: from, To: to, Type: edgeType})
	}
}
