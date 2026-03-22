package flow

// Graph represents a directed flow graph for intake questioning.
type Graph struct {
	ID        string           `yaml:"id" json:"id"`
	Name      string           `yaml:"name" json:"name"`
	Version   string           `yaml:"version" json:"version"`
	StartNode string           `yaml:"start_node" json:"start_node"`
	Nodes     map[string]*Node `yaml:"nodes" json:"nodes"`
}

// Node represents a single step in the flow graph.
type Node struct {
	ID     string   `yaml:"id" json:"id"`
	Type   NodeType `yaml:"type" json:"type"`
	Label  string   `yaml:"label" json:"label"`
	Slots  []string `yaml:"slots" json:"slots"`                        // Slots to fill at this node
	Edges  []Edge   `yaml:"edges" json:"edges"`                        // Outgoing edges
	SkipIf string   `yaml:"skip_if,omitempty" json:"skip_if,omitempty"` // Slot condition to skip
}

// NodeType identifies the kind of flow node.
type NodeType string

const (
	NodeTypeQuestion NodeType = "question" // Ask patient to fill slots
	NodeTypeGate     NodeType = "gate"     // Conditional branching
	NodeTypeComplete NodeType = "complete" // Terminal -- intake finished
	NodeTypeReview   NodeType = "review"   // Send to pharmacist review
)

// Edge represents a directed connection between nodes.
type Edge struct {
	Target    string `yaml:"target" json:"target"`
	Condition string `yaml:"condition,omitempty" json:"condition,omitempty"` // Simple condition expression
	Label     string `yaml:"label,omitempty" json:"label,omitempty"`
}
