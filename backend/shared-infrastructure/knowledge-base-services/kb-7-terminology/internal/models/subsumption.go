package models

import (
	"time"
)

// ============================================================================
// Subsumption Testing Models
// Used for OWL reasoning and is-a relationship testing between concepts
// ============================================================================

// SubsumptionRequest represents a request to test concept subsumption
type SubsumptionRequest struct {
	// CodeA is the potential subtype (more specific concept)
	CodeA string `json:"code_a" binding:"required"`
	// CodeB is the potential supertype (more general concept)
	CodeB string `json:"code_b" binding:"required"`
	// System is the terminology system URI (e.g., SNOMED CT, ICD-10)
	System string `json:"system" binding:"required"`
	// Version is the optional terminology version
	Version string `json:"version,omitempty"`
}

// SubsumptionResult represents the result of a subsumption test
type SubsumptionResult struct {
	// Subsumes indicates if CodeA is subsumed by CodeB (A is-a B)
	Subsumes bool `json:"subsumes"`
	// Relationship describes the specific relationship found
	Relationship SubsumptionRelationship `json:"relationship"`
	// Codes involved in the test
	CodeA       string `json:"code_a"`
	CodeB       string `json:"code_b"`
	DisplayA    string `json:"display_a,omitempty"`
	DisplayB    string `json:"display_b,omitempty"`
	System      string `json:"system"`
	// PathLength is the distance in the hierarchy (0 = same, 1 = direct parent, etc.)
	PathLength int `json:"path_length,omitempty"`
	// Path contains the concept codes forming the subsumption chain
	Path []string `json:"path,omitempty"`
	// Metadata
	ReasoningType string    `json:"reasoning_type"` // rdfs, owl, transitive
	ExecutionTime float64   `json:"execution_time_ms"`
	CachedResult  bool      `json:"cached_result"`
	TestedAt      time.Time `json:"tested_at"`
}

// SubsumptionRelationship describes the type of relationship between two concepts
type SubsumptionRelationship string

const (
	// RelationshipEquivalent indicates the concepts are equivalent
	RelationshipEquivalent SubsumptionRelationship = "equivalent"
	// RelationshipSubsumedBy indicates CodeA is subsumed by CodeB (A is-a B)
	RelationshipSubsumedBy SubsumptionRelationship = "subsumed_by"
	// RelationshipSubsumes indicates CodeA subsumes CodeB (B is-a A)
	RelationshipSubsumes SubsumptionRelationship = "subsumes"
	// RelationshipNotSubsumed indicates no subsumption relationship exists
	RelationshipNotSubsumed SubsumptionRelationship = "not_subsumed"
	// RelationshipUnknown indicates subsumption could not be determined
	RelationshipUnknown SubsumptionRelationship = "unknown"
)

// BatchSubsumptionRequest represents a batch subsumption test request
type BatchSubsumptionRequest struct {
	Tests []SubsumptionRequest `json:"tests" binding:"required"`
}

// BatchSubsumptionResult represents batch subsumption test results
type BatchSubsumptionResult struct {
	Results      []SubsumptionResult `json:"results"`
	TotalCount   int                 `json:"total_count"`
	SuccessCount int                 `json:"success_count"`
	ErrorCount   int                 `json:"error_count"`
	Errors       []SubsumptionError  `json:"errors,omitempty"`
	ExecutionTime float64            `json:"execution_time_ms"`
}

// SubsumptionError represents an error in subsumption testing
type SubsumptionError struct {
	CodeA   string `json:"code_a"`
	CodeB   string `json:"code_b"`
	System  string `json:"system"`
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// AncestorsRequest represents a request to get all ancestors of a concept
type AncestorsRequest struct {
	Code    string `json:"code" binding:"required"`
	System  string `json:"system" binding:"required"`
	Version string `json:"version,omitempty"`
	// MaxDepth limits the ancestor traversal depth (0 = unlimited)
	MaxDepth int `json:"max_depth,omitempty"`
	// IncludeTransitive includes transitive ancestors (default: true)
	IncludeTransitive bool `json:"include_transitive,omitempty"`
}

// AncestorsResult represents the ancestors of a concept
type AncestorsResult struct {
	Code      string            `json:"code"`
	Display   string            `json:"display,omitempty"`
	System    string            `json:"system"`
	Ancestors []ConceptAncestor `json:"ancestors"`
	Total     int               `json:"total"`
	MaxDepth  int               `json:"max_depth_reached"`
}

// ConceptAncestor represents an ancestor concept
type ConceptAncestor struct {
	Code    string `json:"code"`
	Display string `json:"display,omitempty"`
	Depth   int    `json:"depth"` // Distance from the query concept
	Direct  bool   `json:"direct"` // True if immediate parent
}

// DescendantsRequest represents a request to get all descendants of a concept
type DescendantsRequest struct {
	Code    string `json:"code" binding:"required"`
	System  string `json:"system" binding:"required"`
	Version string `json:"version,omitempty"`
	// MaxDepth limits the descendant traversal depth (0 = unlimited)
	MaxDepth int `json:"max_depth,omitempty"`
	// Limit restricts the number of descendants returned
	Limit int `json:"limit,omitempty"`
}

// DescendantsResult represents the descendants of a concept
type DescendantsResult struct {
	Code        string              `json:"code"`
	Display     string              `json:"display,omitempty"`
	System      string              `json:"system"`
	Descendants []ConceptDescendant `json:"descendants"`
	Total       int                 `json:"total"`
	MaxDepth    int                 `json:"max_depth_reached"`
	Truncated   bool                `json:"truncated"` // True if limited by Limit param
}

// ConceptDescendant represents a descendant concept
type ConceptDescendant struct {
	Code    string `json:"code"`
	Display string `json:"display,omitempty"`
	Depth   int    `json:"depth"` // Distance from the query concept
	Direct  bool   `json:"direct"` // True if immediate child
}

// ClosureTableEntry represents an entry in a transitive closure table
// Used for efficient subsumption testing with pre-computed relationships
type ClosureTableEntry struct {
	AncestorCode   string `json:"ancestor_code" db:"ancestor_code"`
	DescendantCode string `json:"descendant_code" db:"descendant_code"`
	System         string `json:"system" db:"system"`
	PathLength     int    `json:"path_length" db:"path_length"`
	Direct         bool   `json:"direct" db:"direct"`
}

// CommonAncestorRequest represents a request to find common ancestors
type CommonAncestorRequest struct {
	Codes   []string `json:"codes" binding:"required,min=2"`
	System  string   `json:"system" binding:"required"`
	Version string   `json:"version,omitempty"`
}

// CommonAncestorResult represents common ancestors of multiple concepts
type CommonAncestorResult struct {
	Codes           []string          `json:"codes"`
	System          string            `json:"system"`
	CommonAncestors []ConceptAncestor `json:"common_ancestors"`
	// LowestCommonAncestor is the most specific shared ancestor
	LowestCommonAncestor *ConceptAncestor `json:"lowest_common_ancestor,omitempty"`
	Total                int              `json:"total"`
}

// SemanticDistanceRequest represents a request to calculate semantic distance
type SemanticDistanceRequest struct {
	CodeA   string `json:"code_a" binding:"required"`
	CodeB   string `json:"code_b" binding:"required"`
	System  string `json:"system" binding:"required"`
	Version string `json:"version,omitempty"`
}

// SemanticDistanceResult represents the semantic distance between concepts
type SemanticDistanceResult struct {
	CodeA              string   `json:"code_a"`
	CodeB              string   `json:"code_b"`
	System             string   `json:"system"`
	Distance           int      `json:"distance"` // Path length between concepts
	PathThroughAncestor string  `json:"path_through_ancestor,omitempty"`
	RelationshipType   string   `json:"relationship_type"` // ancestor, descendant, sibling, unrelated
	Path               []string `json:"path,omitempty"`
}

// OWLReasoningConfig configures OWL reasoning behavior
type OWLReasoningConfig struct {
	// EnableTransitivity enables transitive closure reasoning
	EnableTransitivity bool `json:"enable_transitivity"`
	// EnableEquivalence enables owl:equivalentClass reasoning
	EnableEquivalence bool `json:"enable_equivalence"`
	// UsePrecomputedClosure uses pre-computed closure tables when available
	UsePrecomputedClosure bool `json:"use_precomputed_closure"`
	// MaxReasoningDepth limits reasoning depth
	MaxReasoningDepth int `json:"max_reasoning_depth"`
	// TimeoutSeconds limits reasoning time
	TimeoutSeconds int `json:"timeout_seconds"`
}

// DefaultOWLReasoningConfig returns default OWL reasoning configuration
func DefaultOWLReasoningConfig() OWLReasoningConfig {
	return OWLReasoningConfig{
		EnableTransitivity:    true,
		EnableEquivalence:     true,
		UsePrecomputedClosure: true,
		MaxReasoningDepth:     20,
		TimeoutSeconds:        30,
	}
}
