// Package models defines all data structures for KB-3 Temporal Service
package models

import "time"

// Severity levels for clinical alerts and conflicts
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityMajor    Severity = "major"
	SeverityMinor    Severity = "minor"
)

// EvidenceGrade represents clinical evidence strength
type EvidenceGrade string

const (
	GradeA             EvidenceGrade = "A"
	GradeB             EvidenceGrade = "B"
	GradeC             EvidenceGrade = "C"
	GradeD             EvidenceGrade = "D"
	GradeExpertOpinion EvidenceGrade = "Expert Opinion"
)

// EvidenceStrength returns numeric strength for comparison (higher = stronger)
func (g EvidenceGrade) Strength() int {
	switch g {
	case GradeA:
		return 4
	case GradeB:
		return 3
	case GradeC:
		return 2
	case GradeD:
		return 1
	default:
		return 0
	}
}

// Guideline represents a clinical guideline
type Guideline struct {
	GuidelineID      string                 `json:"guideline_id"`
	Name             string                 `json:"name"`
	Source           string                 `json:"source"`           // Authoritative source
	Organization     string                 `json:"organization"`
	Version          string                 `json:"version"`
	PublicationDate  time.Time              `json:"publication_date"`
	EffectiveDate    time.Time              `json:"effective_date"`
	ExpiryDate       *time.Time             `json:"expiry_date,omitempty"`
	Region           string                 `json:"region"`
	Domain           string                 `json:"domain"`
	Status           string                 `json:"status"`           // draft, active, deprecated
	Conditions       []string               `json:"conditions"`
	EvidenceGrade    EvidenceGrade          `json:"evidence_grade"`
	QualityScore     int                    `json:"quality_score"`
	Recommendations  []Recommendation       `json:"recommendations"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	Active           bool                   `json:"active"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// Recommendation within a guideline
type Recommendation struct {
	RecommendationID string                 `json:"recommendation_id"`
	GuidelineID      string                 `json:"guideline_id"`
	Text             string                 `json:"text"`
	Action           string                 `json:"action"`           // Recommended action
	Strength         string                 `json:"strength"`         // Strong, Moderate, Weak
	EvidenceQuality  string                 `json:"evidence_quality"` // High, Moderate, Low
	Domain           string                 `json:"domain"`
	EvidenceGrade    EvidenceGrade          `json:"evidence_grade"`
	QualityScore     int                    `json:"quality_score"`
	TargetValue      *float64               `json:"target_value,omitempty"`
	TargetUnit       string                 `json:"target_unit,omitempty"`
	TargetRange      *TargetRange           `json:"target_range,omitempty"`
	Parameters       map[string]interface{} `json:"parameters,omitempty"`
}

// TargetRange for recommendations with ranges
type TargetRange struct {
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Unit string  `json:"unit"`
}

// AuditEntry for tracking changes and actions
type AuditEntry struct {
	EntryID     string                 `json:"entry_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Action      string                 `json:"action"`
	Actor       string                 `json:"actor"`
	UserID      string                 `json:"user_id,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Checksum    string                 `json:"checksum,omitempty"`
}

// HealthStatus for service health checks
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Services  map[string]ServiceHealth `json:"services"`
}

// ServiceHealth for individual service health
type ServiceHealth struct {
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Message string `json:"message,omitempty"`
}
