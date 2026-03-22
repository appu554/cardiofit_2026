package flow

// Graph represents a directed flow graph for intake questioning.
type Graph struct {
	ID        string           `yaml:"id" json:"id"`
	Name      string           `yaml:"name" json:"name"`
	Version   string           `yaml:"version" json:"version"`
	StartNode string           `yaml:"start_node" json:"start_node"`
	Nodes     map[string]*Node `yaml:"nodes" json:"nodes"`
}

// ExtractionMode specifies how a node collects data from the patient.
type ExtractionMode string

const (
	ExtractionForm     ExtractionMode = "form"      // Structured form UI
	ExtractionNLU      ExtractionMode = "nlu"       // Natural language extraction (WhatsApp, voice)
	ExtractionDevice   ExtractionMode = "device"    // Auto-populated from device/wearable data
	ExtractionEHR      ExtractionMode = "ehr"       // Pre-filled from EHR/FHIR import
	ExtractionUpload   ExtractionMode = "upload"    // Document/image upload (lab reports)
	ExtractionHybrid   ExtractionMode = "hybrid"    // Multi-mode: form + NLU fallback
)

// Node represents a single step in the flow graph.
type Node struct {
	ID             string         `yaml:"id" json:"id"`
	Type           NodeType       `yaml:"type" json:"type"`
	Label          string         `yaml:"label" json:"label"`
	Slots          []string       `yaml:"slots" json:"slots"`                                            // Slots to fill at this node
	Edges          []Edge         `yaml:"edges" json:"edges"`                                            // Outgoing edges
	SkipIf         string         `yaml:"skip_if,omitempty" json:"skip_if,omitempty"`                    // Slot condition to skip
	ExtractionMode ExtractionMode `yaml:"extraction_mode,omitempty" json:"extraction_mode,omitempty"`    // How this node collects data
	NLUHints       []string       `yaml:"nlu_hints,omitempty" json:"nlu_hints,omitempty"`                // Prompt hints for NLU extraction
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
