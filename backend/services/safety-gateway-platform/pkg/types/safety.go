package types

import (
	"context"
	"time"
)

// SafetyStatus represents the overall safety assessment result
type SafetyStatus string

const (
	SafetyStatusSafe         SafetyStatus = "SAFE"
	SafetyStatusUnsafe       SafetyStatus = "UNSAFE"
	SafetyStatusWarning      SafetyStatus = "WARNING"
	SafetyStatusManualReview SafetyStatus = "MANUAL_REVIEW"
	SafetyStatusError        SafetyStatus = "ERROR"
)

// CriticalityTier defines the criticality level of safety engines
type CriticalityTier int

const (
	TierVetoCritical CriticalityTier = 1 // Failure = UNSAFE (fail closed)
	TierAdvisory     CriticalityTier = 2 // Failure = WARNING (degraded)
)

// EngineStatus represents the health status of an engine
type EngineStatus string

const (
	EngineStatusHealthy   EngineStatus = "healthy"
	EngineStatusDegraded  EngineStatus = "degraded"
	EngineStatusUnhealthy EngineStatus = "unhealthy"
)

// SafetyRequest represents an incoming safety validation request
type SafetyRequest struct {
	RequestID     string            `json:"request_id"`
	PatientID     string            `json:"patient_id"`
	ClinicianID   string            `json:"clinician_id"`
	ActionType    string            `json:"action_type"`
	Priority      string            `json:"priority"`
	MedicationIDs []string          `json:"medication_ids,omitempty"`
	ConditionIDs  []string          `json:"condition_ids,omitempty"`
	AllergyIDs    []string          `json:"allergy_ids,omitempty"`
	Context       map[string]string `json:"context,omitempty"`
	Timestamp     time.Time         `json:"timestamp"`
	Source        string            `json:"source"`
}

// SafetyResponse represents the response from safety validation
type SafetyResponse struct {
	RequestID          string                 `json:"request_id"`
	Status             SafetyStatus           `json:"status"`
	RiskScore          float64                `json:"risk_score"`
	CriticalViolations []string               `json:"critical_violations,omitempty"`
	Warnings           []string               `json:"warnings,omitempty"`
	EngineResults      []EngineResult         `json:"engine_results"`
	EnginesFailed      []string               `json:"engines_failed,omitempty"`
	Explanation        *Explanation           `json:"explanation,omitempty"`
	OverrideToken      *OverrideToken         `json:"override_token,omitempty"`
	ProcessingTime     time.Duration          `json:"processing_time_ms"`
	ContextVersion     string                 `json:"context_version"`
	Timestamp          time.Time              `json:"timestamp"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// EngineResult represents the result from a single safety engine
type EngineResult struct {
	EngineID    string          `json:"engine_id"`
	EngineName  string          `json:"engine_name"`
	Status      SafetyStatus    `json:"status"`
	RiskScore   float64         `json:"risk_score"`
	Violations  []string        `json:"violations,omitempty"`
	Warnings    []string        `json:"warnings,omitempty"`
	Confidence  float64         `json:"confidence"`
	Duration    time.Duration   `json:"duration_ms"`
	Tier        CriticalityTier `json:"tier"`
	Error       string          `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ClinicalContext represents the assembled clinical context for a patient
type ClinicalContext struct {
	PatientID         string                 `json:"patient_id"`
	Demographics      *PatientDemographics   `json:"demographics"`
	ActiveMedications []Medication           `json:"active_medications"`
	Allergies         []Allergy              `json:"allergies"`
	Conditions        []Condition            `json:"conditions"`
	RecentVitals      []VitalSign            `json:"recent_vitals"`
	LabResults        []LabResult            `json:"lab_results"`
	RecentEncounters  []Encounter            `json:"recent_encounters"`
	ContextVersion    string                 `json:"context_version"`
	AssemblyTime      time.Time              `json:"assembly_time"`
	DataSources       []string               `json:"data_sources"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// Demographics represents detailed patient demographic information from FHIR
type Demographics struct {
	PatientID     string    `json:"patient_id"`
	FirstName     string    `json:"first_name"`
	LastName      string    `json:"last_name"`
	Age           int       `json:"age"`
	Gender        string    `json:"gender"`
	DateOfBirth   time.Time `json:"date_of_birth"`
	Weight        float64   `json:"weight_kg,omitempty"`
	Height        float64   `json:"height_cm,omitempty"`
	BMI           float64   `json:"bmi,omitempty"`
	PregnancyStatus string  `json:"pregnancy_status,omitempty"`
}

// PatientDemographics represents basic patient demographic information
type PatientDemographics struct {
	Age           int     `json:"age"`
	Gender        string  `json:"gender"`
	Weight        float64 `json:"weight_kg,omitempty"`
	Height        float64 `json:"height_cm,omitempty"`
	BMI           float64 `json:"bmi,omitempty"`
	PregnancyStatus string `json:"pregnancy_status,omitempty"`
}

// Medication represents a medication in the clinical context
type Medication struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	GenericName  string    `json:"generic_name,omitempty"`
	Dosage       string    `json:"dosage"`
	Route        string    `json:"route"`
	Frequency    string    `json:"frequency"`
	StartDate    time.Time `json:"start_date"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	Status       string    `json:"status"`
	Prescriber   string    `json:"prescriber,omitempty"`
}

// Allergy represents an allergy in the clinical context
type Allergy struct {
	ID          string    `json:"id"`
	Allergen    string    `json:"allergen"`
	Reaction    string    `json:"reaction"`
	Severity    string    `json:"severity"`
	OnsetDate   time.Time `json:"onset_date,omitempty"`
	Status      string    `json:"status"`
	VerifiedBy  string    `json:"verified_by,omitempty"`
}

// Condition represents a medical condition in the clinical context
type Condition struct {
	ID           string    `json:"id"`
	Code         string    `json:"code"`
	Display      string    `json:"display"`
	Severity     string    `json:"severity,omitempty"`
	OnsetDate    time.Time `json:"onset_date,omitempty"`
	Status       string    `json:"status"`
	DiagnosedBy  string    `json:"diagnosed_by,omitempty"`
}

// VitalSign represents a vital sign measurement
type VitalSign struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
}

// LabResult represents a laboratory test result
type LabResult struct {
	ID           string    `json:"id"`
	TestName     string    `json:"test_name"`
	TestCode     string    `json:"test_code"`
	Value        string    `json:"value"`
	Unit         string    `json:"unit,omitempty"`
	ReferenceRange string  `json:"reference_range,omitempty"`
	Status       string    `json:"status"`
	Timestamp    time.Time `json:"timestamp"`
	Abnormal     bool      `json:"abnormal"`
}

// Encounter represents a clinical encounter
type Encounter struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	StartTime   time.Time `json:"start_time"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Provider    string    `json:"provider,omitempty"`
	Location    string    `json:"location,omitempty"`
	ReasonCode  string    `json:"reason_code,omitempty"`
	ReasonText  string    `json:"reason_text,omitempty"`
}

// GraphContext represents clinical context from GraphDB
type GraphContext struct {
	PatientID     string              `json:"patient_id"`
	Entities      []GraphEntity       `json:"entities"`
	Relationships []GraphRelationship `json:"relationships"`
	Timestamp     time.Time           `json:"timestamp"`
}

// GraphEntity represents an entity in the knowledge graph
type GraphEntity struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

// GraphRelationship represents a relationship in the knowledge graph
type GraphRelationship struct {
	ID         string                 `json:"id"`
	SourceID   string                 `json:"source_id"`
	TargetID   string                 `json:"target_id"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

// SafetyEngine defines the interface that all safety engines must implement
type SafetyEngine interface {
	// Core engine methods
	ID() string
	Name() string
	Capabilities() []string

	// Safety evaluation (legacy mode)
	Evaluate(ctx context.Context, req *SafetyRequest, clinicalContext *ClinicalContext) (*EngineResult, error)

	// Health and lifecycle
	HealthCheck() error
	Initialize(config EngineConfig) error
	Shutdown() error
}

// SnapshotAwareEngine extends SafetyEngine with snapshot-based evaluation support
type SnapshotAwareEngine interface {
	SafetyEngine

	// Snapshot-based evaluation
	EvaluateWithSnapshot(ctx context.Context, req *SafetyRequest, snapshot *ClinicalSnapshot) (*EngineResult, error)
	
	// Snapshot compatibility checks
	IsSnapshotCompatible() bool
	GetSnapshotRequirements() []string
}

// EngineConfig represents configuration for a safety engine
type EngineConfig struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Enabled      bool                   `json:"enabled"`
	Timeout      time.Duration          `json:"timeout"`
	Priority     int                    `json:"priority"`
	Tier         CriticalityTier        `json:"tier"`
	Capabilities []string               `json:"capabilities"`
	Settings     map[string]interface{} `json:"settings,omitempty"`
}
