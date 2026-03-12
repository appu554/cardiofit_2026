package models

import "time"

// ChangeType categorizes version changes
type ChangeType string

const (
	ChangeMajor     ChangeType = "major"
	ChangeMinor     ChangeType = "minor"
	ChangePatch     ChangeType = "patch"
	ChangeEmergency ChangeType = "emergency"
	ChangeSecurity  ChangeType = "security"
)

// ImpactLevel for clinical impact assessment
type ImpactLevel string

const (
	ImpactCritical ImpactLevel = "critical"
	ImpactMajor    ImpactLevel = "major"
	ImpactMinor    ImpactLevel = "minor"
	ImpactCosmetic ImpactLevel = "cosmetic"
)

// VersionStatus tracks version lifecycle
type VersionStatus string

const (
	VersionDraft     VersionStatus = "draft"
	VersionPending   VersionStatus = "pending_approval"
	VersionApproved  VersionStatus = "approved"
	VersionActive    VersionStatus = "active"
	VersionWithdrawn VersionStatus = "withdrawn"
	VersionSuperseded VersionStatus = "superseded"
)

// ApproverRole for approval chain
type ApproverRole string

const (
	ApproverTechnicalLead  ApproverRole = "technical_lead"
	ApproverClinicalLead   ApproverRole = "clinical_lead"
	ApproverMedicalDirector ApproverRole = "medical_director"
	ApproverSafetyCommittee ApproverRole = "safety_committee"
	ApproverLegalReview    ApproverRole = "legal_review"
)

// ApprovalStatus for approval chain
type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
	ApprovalDeferred ApprovalStatus = "deferred"
)

// RiskLevel for change assessment
type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskModerate RiskLevel = "moderate"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

// GuidelineVersion represents a version of a guideline
type GuidelineVersion struct {
	VersionID      string           `json:"version_id"`
	GuidelineID    string           `json:"guideline_id"`
	Version        string           `json:"version"` // Semantic version X.Y.Z
	ChangeType     ChangeType       `json:"change_type"`
	Changes        []ChangeRecord   `json:"changes"`
	ClinicalImpact ClinicalImpact   `json:"clinical_impact"`
	ApprovalChain  []ApprovalRecord `json:"approval_chain"`
	TransitionPlan *TransitionPlan  `json:"transition_plan,omitempty"`
	Status         VersionStatus    `json:"status"`
	CreatedBy      string           `json:"created_by"`
	CreatedAt      time.Time        `json:"created_at"`
	EffectiveDate  *time.Time       `json:"effective_date,omitempty"`
}

// ChangeRecord documents a specific change
type ChangeRecord struct {
	ChangeID             string    `json:"change_id"`
	Field                string    `json:"field"`
	OldValue             any       `json:"old_value"`
	NewValue             any       `json:"new_value"`
	ChangeReason         string    `json:"change_reason"`
	EvidenceReference    string    `json:"evidence_reference,omitempty"`
	ClinicalJustification string   `json:"clinical_justification"`
	RiskAssessment       RiskLevel `json:"risk_assessment"`
}

// ClinicalImpact assessment of changes
type ClinicalImpact struct {
	Score                int         `json:"score"` // 0-100
	Level                ImpactLevel `json:"level"`
	AffectedDomains      []string    `json:"affected_domains"`
	AffectedPopulations  []string    `json:"affected_populations"`
	SafetyImplications   []string    `json:"safety_implications"`
	MonitoringChanges    []string    `json:"monitoring_changes"`
	TrainingRequired     bool        `json:"training_required"`
	NotificationRequired bool        `json:"notification_required"`
}

// ApprovalRecord tracks approval chain progress
type ApprovalRecord struct {
	ApproverRole     ApproverRole   `json:"approver_role"`
	ApproverID       string         `json:"approver_id"`
	ApprovalStatus   ApprovalStatus `json:"approval_status"`
	ApprovalDate     *time.Time     `json:"approval_date,omitempty"`
	Comments         string         `json:"comments,omitempty"`
	Conditions       []string       `json:"conditions,omitempty"`
	DigitalSignature string         `json:"digital_signature,omitempty"`
}

// TransitionPlan for version transitions
type TransitionPlan struct {
	TransitionID         string                 `json:"transition_id"`
	OldVersion           string                 `json:"old_version"`
	NewVersion           string                 `json:"new_version"`
	TransitionPeriodDays int                    `json:"transition_period_days"`
	ParallelRun          bool                   `json:"parallel_run"`
	AutoMigration        bool                   `json:"auto_migration"`
	NotificationStrategy NotificationStrategy   `json:"notification_strategy"`
	RollbackPlan         RollbackPlan           `json:"rollback_plan"`
	ValidationCheckpoints []ValidationCheckpoint `json:"validation_checkpoints"`
}

// NotificationStrategy for version changes
type NotificationStrategy struct {
	TargetAudiences  []string `json:"target_audiences"`
	Channels         []string `json:"channels"`
	AdvanceNoticeDays int     `json:"advance_notice_days"`
	ReminderSchedule []string `json:"reminder_schedule"`
}

// RollbackPlan for version rollback
type RollbackPlan struct {
	Triggers           []string `json:"triggers"`
	Procedure          []string `json:"procedure"`
	DataPreservation   bool     `json:"data_preservation"`
	ApprovalRequired   bool     `json:"approval_required"`
}

// ValidationCheckpoint for transition validation
type ValidationCheckpoint struct {
	CheckpointID   string    `json:"checkpoint_id"`
	Name           string    `json:"name"`
	DaysFromStart  int       `json:"days_from_start"`
	Criteria       []string  `json:"criteria"`
	Status         string    `json:"status"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
}

// CreateVersionRequest for API
type CreateVersionRequest struct {
	GuidelineID string         `json:"guideline_id" binding:"required"`
	ChangeType  ChangeType     `json:"change_type" binding:"required"`
	Changes     []ChangeRecord `json:"changes" binding:"required"`
	RequestorID string         `json:"requestor_id" binding:"required"`
}

// ProcessApprovalRequest for API
type ProcessApprovalRequest struct {
	ApproverRole ApproverRole   `json:"approver_role" binding:"required"`
	ApproverID   string         `json:"approver_id" binding:"required"`
	Status       ApprovalStatus `json:"status" binding:"required"`
	Comments     string         `json:"comments,omitempty"`
	Conditions   []string       `json:"conditions,omitempty"`
}

// GetImpactLevel returns impact level based on score
// Critical ≥30, Major ≥15, Minor ≥5, Cosmetic <5
func GetImpactLevel(score int) ImpactLevel {
	switch {
	case score >= 30:
		return ImpactCritical
	case score >= 15:
		return ImpactMajor
	case score >= 5:
		return ImpactMinor
	default:
		return ImpactCosmetic
	}
}

// GetTransitionPeriod returns transition period based on impact level
func GetTransitionPeriod(level ImpactLevel) int {
	switch level {
	case ImpactCritical:
		return 90
	case ImpactMajor:
		return 60
	case ImpactMinor:
		return 30
	default:
		return 14
	}
}
