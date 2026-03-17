package graph

type GraphNode struct {
	Labels     []string               `json:"labels"`
	Properties map[string]interface{} `json:"properties"`
}

type GraphEdge struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

type TraversalPath struct {
	SourceCode  string    `json:"source_code"`
	SourceType  string    `json:"source_type"`
	EdgeTypes   []string  `json:"edge_types"`
	EffectSizes []float64 `json:"effect_sizes"`
	Grades      []string  `json:"grades"`
	Length      int       `json:"length"`
}
