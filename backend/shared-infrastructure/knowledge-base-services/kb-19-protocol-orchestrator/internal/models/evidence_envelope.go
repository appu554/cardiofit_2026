// Package models provides domain models for KB-19 Protocol Orchestrator.
package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EvidenceEnvelope provides the legal and clinical evidence chain for a recommendation.
// This is critical for regulatory compliance (FDA 21 CFR Part 11, HIPAA) and
// clinical audit trails. Every ArbitratedDecision must have an EvidenceEnvelope.
type EvidenceEnvelope struct {
	// Unique identifier for this evidence envelope
	ID uuid.UUID `json:"id"`

	// ACC/AHA recommendation class
	RecommendationClass RecommendationClass `json:"recommendation_class"`

	// Evidence level supporting this recommendation
	EvidenceLevel EvidenceLevel `json:"evidence_level"`

	// Source guideline information
	GuidelineSource  string `json:"guideline_source"`  // ACC/AHA, SSC, KDIGO, etc.
	GuidelineVersion string `json:"guideline_version"` // e.g., "2021"
	GuidelineYear    int    `json:"guideline_year"`

	// Citation anchor (DOI or reference)
	CitationAnchor string `json:"citation_anchor"`

	// Full citation text
	CitationText string `json:"citation_text,omitempty"`

	// Inference chain showing how we arrived at this recommendation
	InferenceChain []InferenceStep `json:"inference_chain"`

	// Versions of KB services used in this evaluation
	KBVersions map[string]string `json:"kb_versions"`

	// Version of Vaidshala CQL Engine used
	CQLEngineVersion string `json:"cql_engine_version"`

	// Patient context snapshot ID (for reproducibility)
	PatientContextID uuid.UUID `json:"patient_context_id"`

	// Timestamp when this envelope was created
	Timestamp time.Time `json:"timestamp"`

	// SHA256 checksum for integrity verification
	Checksum string `json:"checksum"`

	// Whether this envelope has been finalized (immutable after finalization)
	Finalized bool `json:"finalized"`

	// Digital signature (optional, for high-security environments)
	DigitalSignature string `json:"digital_signature,omitempty"`

	// Reviewed by (clinician ID who reviewed, if applicable)
	ReviewedBy string `json:"reviewed_by,omitempty"`

	// Review timestamp
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`

	// Override information (if clinician overrode the recommendation)
	Override *OverrideInfo `json:"override,omitempty"`
}

// RecommendationClass represents ACC/AHA recommendation strength.
type RecommendationClass string

const (
	// ClassI - Procedure/treatment SHOULD be performed/administered
	// Benefit >>> Risk
	ClassI RecommendationClass = "I"

	// ClassIIa - It is REASONABLE to perform procedure/administer treatment
	// Benefit >> Risk
	ClassIIa RecommendationClass = "IIa"

	// ClassIIb - Procedure/treatment MAY BE CONSIDERED
	// Benefit >= Risk
	ClassIIb RecommendationClass = "IIb"

	// ClassIII - Procedure/treatment should NOT be performed (no benefit or harmful)
	// Risk >= Benefit
	ClassIII RecommendationClass = "III"
)

// EvidenceLevel represents the level of evidence supporting a recommendation.
type EvidenceLevel string

const (
	// EvidenceA - Multiple populations evaluated; RCTs or meta-analyses
	EvidenceA EvidenceLevel = "A"

	// EvidenceB - Limited populations evaluated; single RCT or nonrandomized studies
	EvidenceB EvidenceLevel = "B"

	// EvidenceC - Very limited populations evaluated; consensus or expert opinion
	EvidenceC EvidenceLevel = "C"

	// EvidenceExpert - Expert consensus only (no studies)
	EvidenceExpert EvidenceLevel = "EXPERT"
)

// InferenceStep represents a single step in the reasoning chain.
type InferenceStep struct {
	// Step number in sequence
	StepNumber int `json:"step_number"`

	// Type of reasoning step
	StepType InferenceStepType `json:"step_type"`

	// Source of this step (KB service, CQL library, etc.)
	Source string `json:"source"`

	// Input facts or values used in this step
	Inputs map[string]interface{} `json:"inputs"`

	// Logic or rule applied
	LogicApplied string `json:"logic_applied"`

	// Output/conclusion from this step
	Output string `json:"output"`

	// Confidence in this step (0.0 - 1.0)
	Confidence float64 `json:"confidence"`

	// Timestamp of this step
	Timestamp time.Time `json:"timestamp"`
}

// InferenceStepType categorizes types of inference steps.
type InferenceStepType string

const (
	// StepCQLEvaluation - CQL fact evaluation
	StepCQLEvaluation InferenceStepType = "CQL_EVALUATION"

	// StepCalculation - Clinical calculator computation
	StepCalculation InferenceStepType = "CALCULATION"

	// StepProtocolMatch - Protocol applicability determination
	StepProtocolMatch InferenceStepType = "PROTOCOL_MATCH"

	// StepConflictResolution - Conflict between protocols resolved
	StepConflictResolution InferenceStepType = "CONFLICT_RESOLUTION"

	// StepSafetyCheck - Safety gatekeeper evaluation
	StepSafetyCheck InferenceStepType = "SAFETY_CHECK"

	// StepGrading - Recommendation class assignment
	StepGrading InferenceStepType = "GRADING"
)

// OverrideInfo captures information when a clinician overrides a recommendation.
type OverrideInfo struct {
	OverriddenAt time.Time `json:"overridden_at"`
	OverriddenBy string    `json:"overridden_by"` // Clinician ID
	Reason       string    `json:"reason"`
	NewDecision  string    `json:"new_decision"` // What was done instead
}

// NewEvidenceEnvelope creates a new EvidenceEnvelope with initialized values.
// Sets default EvidenceLevel to EvidenceExpert (expert consensus) and
// RecommendationClass to ClassIIb (reasonable to consider) as baseline.
// These should be upgraded by the protocol evaluation when stronger evidence exists.
func NewEvidenceEnvelope() *EvidenceEnvelope {
	return &EvidenceEnvelope{
		ID:                  uuid.New(),
		InferenceChain:      make([]InferenceStep, 0),
		KBVersions:          make(map[string]string),
		Timestamp:           time.Now(),
		Finalized:           false,
		EvidenceLevel:       EvidenceExpert, // Default: expert consensus
		RecommendationClass: ClassIIb,       // Default: reasonable to consider
		GuidelineSource:     "KB-19 Protocol Engine", // Default attribution
	}
}

// AddInferenceStep adds a step to the inference chain.
func (e *EvidenceEnvelope) AddInferenceStep(stepType InferenceStepType, source, logic, output string, inputs map[string]interface{}, confidence float64) {
	step := InferenceStep{
		StepNumber:   len(e.InferenceChain) + 1,
		StepType:     stepType,
		Source:       source,
		Inputs:       inputs,
		LogicApplied: logic,
		Output:       output,
		Confidence:   confidence,
		Timestamp:    time.Now(),
	}
	e.InferenceChain = append(e.InferenceChain, step)
}

// SetGuideline sets the guideline source information.
func (e *EvidenceEnvelope) SetGuideline(source, version string, year int, citation string) {
	e.GuidelineSource = source
	e.GuidelineVersion = version
	e.GuidelineYear = year
	e.CitationAnchor = citation
}

// SetRecommendationStrength sets the recommendation class and evidence level.
func (e *EvidenceEnvelope) SetRecommendationStrength(class RecommendationClass, level EvidenceLevel) {
	e.RecommendationClass = class
	e.EvidenceLevel = level
}

// RecordKBVersion records the version of a KB service used.
func (e *EvidenceEnvelope) RecordKBVersion(kbName, version string) {
	e.KBVersions[kbName] = version
}

// Finalize computes the checksum and marks the envelope as finalized.
// After finalization, the envelope should not be modified.
func (e *EvidenceEnvelope) Finalize() error {
	e.Finalized = true
	e.Timestamp = time.Now()

	// Compute checksum over key fields
	checksum, err := e.computeChecksum()
	if err != nil {
		return err
	}
	e.Checksum = checksum

	return nil
}

// computeChecksum computes SHA256 over the envelope contents.
func (e *EvidenceEnvelope) computeChecksum() (string, error) {
	// Create a struct with the fields to hash
	hashInput := struct {
		ID                  uuid.UUID
		RecommendationClass RecommendationClass
		EvidenceLevel       EvidenceLevel
		GuidelineSource     string
		InferenceChain      []InferenceStep
		KBVersions          map[string]string
		Timestamp           time.Time
	}{
		ID:                  e.ID,
		RecommendationClass: e.RecommendationClass,
		EvidenceLevel:       e.EvidenceLevel,
		GuidelineSource:     e.GuidelineSource,
		InferenceChain:      e.InferenceChain,
		KBVersions:          e.KBVersions,
		Timestamp:           e.Timestamp,
	}

	data, err := json.Marshal(hashInput)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// VerifyChecksum verifies the envelope's integrity.
func (e *EvidenceEnvelope) VerifyChecksum() (bool, error) {
	if !e.Finalized {
		return false, nil
	}

	computed, err := e.computeChecksum()
	if err != nil {
		return false, err
	}

	return computed == e.Checksum, nil
}

// String returns the string representation of RecommendationClass.
func (rc RecommendationClass) String() string {
	switch rc {
	case ClassI:
		return "Class I (Strong - Should do)"
	case ClassIIa:
		return "Class IIa (Moderate - Reasonable to do)"
	case ClassIIb:
		return "Class IIb (Weak - May consider)"
	case ClassIII:
		return "Class III (Harmful - Should NOT do)"
	default:
		return "Unknown"
	}
}

// IsRecommended returns true if the class indicates the action is recommended.
func (rc RecommendationClass) IsRecommended() bool {
	return rc == ClassI || rc == ClassIIa
}

// IsHarmful returns true if the class indicates the action is harmful.
func (rc RecommendationClass) IsHarmful() bool {
	return rc == ClassIII
}
