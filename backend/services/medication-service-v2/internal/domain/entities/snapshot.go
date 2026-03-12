package entities

import (
	"time"
	"github.com/google/uuid"
)

// ClinicalSnapshot represents an immutable snapshot of clinical context
type ClinicalSnapshot struct {
	ID                uuid.UUID                  `json:"id" db:"id"`
	PatientID         uuid.UUID                  `json:"patient_id" db:"patient_id"`
	RecipeID          uuid.UUID                  `json:"recipe_id" db:"recipe_id"`
	SnapshotType      SnapshotType               `json:"snapshot_type" db:"snapshot_type"`
	Status            SnapshotStatus             `json:"status" db:"status"`
	Version           int                        `json:"version" db:"version"`
	ClinicalData      ClinicalSnapshotData       `json:"clinical_data" db:"clinical_data"`
	FreshnessMetadata FreshnessMetadata          `json:"freshness_metadata" db:"freshness_metadata"`
	ValidationResults ValidationResults          `json:"validation_results" db:"validation_results"`
	CreatedAt         time.Time                  `json:"created_at" db:"created_at"`
	ExpiresAt         time.Time                  `json:"expires_at" db:"expires_at"`
	CreatedBy         string                     `json:"created_by" db:"created_by"`
	Hash              string                     `json:"hash" db:"hash"`
	PreviousSnapshotID *uuid.UUID                `json:"previous_snapshot_id,omitempty" db:"previous_snapshot_id"`
	ChangeReason      string                     `json:"change_reason,omitempty" db:"change_reason"`
	AuditTrail        []SnapshotAuditEntry       `json:"audit_trail" db:"audit_trail"`
}

// SnapshotType represents different types of clinical snapshots
type SnapshotType string

const (
	SnapshotTypeCalculation SnapshotType = "calculation"
	SnapshotTypeValidation  SnapshotType = "validation"
	SnapshotTypeCommit      SnapshotType = "commit"
	SnapshotTypeMonitoring  SnapshotType = "monitoring"
	SnapshotTypeEmergency   SnapshotType = "emergency"
)

// SnapshotStatus represents the status of a clinical snapshot
type SnapshotStatus string

const (
	SnapshotStatusPending   SnapshotStatus = "pending"
	SnapshotStatusActive    SnapshotStatus = "active"
	SnapshotStatusSuperseded SnapshotStatus = "superseded"
	SnapshotStatusExpired   SnapshotStatus = "expired"
	SnapshotStatusInvalid   SnapshotStatus = "invalid"
)

// ClinicalSnapshotData contains the actual clinical data
type ClinicalSnapshotData struct {
	Demographics     PatientDemographics        `json:"demographics"`
	VitalSigns       VitalSigns                 `json:"vital_signs,omitempty"`
	LabResults       []LabResult                `json:"lab_results,omitempty"`
	Medications      []MedicationEntry          `json:"medications,omitempty"`
	Allergies        []AllergyEntry             `json:"allergies,omitempty"`
	Conditions       []ConditionEntry           `json:"conditions,omitempty"`
	Procedures       []ProcedureEntry           `json:"procedures,omitempty"`
	Observations     []ObservationEntry         `json:"observations,omitempty"`
	AssessmentScores []AssessmentScore          `json:"assessment_scores,omitempty"`
	RiskFactors      []RiskFactor               `json:"risk_factors,omitempty"`
}

// PatientDemographics contains basic patient demographic information
type PatientDemographics struct {
	PatientID        uuid.UUID `json:"patient_id"`
	MRN              string    `json:"mrn"`
	DateOfBirth      time.Time `json:"date_of_birth"`
	Gender           string    `json:"gender"`
	WeightKg         *float64  `json:"weight_kg,omitempty"`
	HeightCm         *float64  `json:"height_cm,omitempty"`
	BMI              *float64  `json:"bmi,omitempty"`
	BSAm2            *float64  `json:"bsa_m2,omitempty"`
	Race             string    `json:"race,omitempty"`
	Ethnicity        string    `json:"ethnicity,omitempty"`
	PreferredLanguage string   `json:"preferred_language,omitempty"`
	SnapshotTime     time.Time `json:"snapshot_time"`
}

// VitalSigns contains patient vital signs at snapshot time
type VitalSigns struct {
	SystolicBP       *int      `json:"systolic_bp,omitempty"`
	DiastolicBP      *int      `json:"diastolic_bp,omitempty"`
	HeartRate        *int      `json:"heart_rate,omitempty"`
	RespiratoryRate  *int      `json:"respiratory_rate,omitempty"`
	Temperature      *float64  `json:"temperature,omitempty"`
	OxygenSaturation *float64  `json:"oxygen_saturation,omitempty"`
	PainScore        *int      `json:"pain_score,omitempty"`
	MeasuredAt       time.Time `json:"measured_at"`
}

// LabResult represents a laboratory test result
type LabResult struct {
	ID              uuid.UUID         `json:"id"`
	TestName        string            `json:"test_name"`
	TestCode        string            `json:"test_code"`
	Value           interface{}       `json:"value"`
	Unit            string            `json:"unit"`
	ReferenceRange  string            `json:"reference_range"`
	AbnormalFlag    string            `json:"abnormal_flag,omitempty"`
	Status          LabResultStatus   `json:"status"`
	CollectedAt     time.Time         `json:"collected_at"`
	ReportedAt      time.Time         `json:"reported_at"`
	PerformingLab   string            `json:"performing_lab"`
	CriticalValue   bool              `json:"critical_value"`
	Comments        string            `json:"comments,omitempty"`
}

// LabResultStatus represents the status of a lab result
type LabResultStatus string

const (
	LabStatusFinal      LabResultStatus = "final"
	LabStatusPreliminary LabResultStatus = "preliminary"
	LabStatusCorrected  LabResultStatus = "corrected"
	LabStatusCancelled  LabResultStatus = "cancelled"
)

// MedicationEntry represents a medication in the snapshot
type MedicationEntry struct {
	ID              uuid.UUID           `json:"id"`
	MedicationName  string              `json:"medication_name"`
	GenericName     string              `json:"generic_name"`
	DoseMg          float64             `json:"dose_mg"`
	Unit            string              `json:"unit"`
	Route           string              `json:"route"`
	Frequency       string              `json:"frequency"`
	StartDate       time.Time           `json:"start_date"`
	EndDate         *time.Time          `json:"end_date,omitempty"`
	Status          MedicationStatus    `json:"status"`
	Indication      string              `json:"indication"`
	PrescribedBy    string              `json:"prescribed_by"`
	Instructions    string              `json:"instructions,omitempty"`
	LastDoseTime    *time.Time          `json:"last_dose_time,omitempty"`
	AdherenceScore  *float64            `json:"adherence_score,omitempty"`
}

// MedicationStatus represents the status of a medication
type MedicationStatus string

const (
	MedStatusActive      MedicationStatus = "active"
	MedStatusCompleted   MedicationStatus = "completed"
	MedStatusDiscontinued MedicationStatus = "discontinued"
	MedStatusHeld        MedicationStatus = "held"
)

// AllergyEntry represents an allergy in the snapshot
type AllergyEntry struct {
	ID           uuid.UUID       `json:"id"`
	Allergen     string          `json:"allergen"`
	AllergenType AllergenType    `json:"allergen_type"`
	Reaction     string          `json:"reaction"`
	Severity     AllergySeverity `json:"severity"`
	OnsetDate    *time.Time      `json:"onset_date,omitempty"`
	Status       AllergyStatus   `json:"status"`
	Notes        string          `json:"notes,omitempty"`
	ReportedBy   string          `json:"reported_by"`
	VerifiedBy   string          `json:"verified_by,omitempty"`
}

// AllergenType represents different types of allergens
type AllergenType string

const (
	AllergenDrug        AllergenType = "drug"
	AllergenFood        AllergenType = "food"
	AllergenEnvironmental AllergenType = "environmental"
	AllergenOther       AllergenType = "other"
)

// AllergySeverity represents allergy severity levels
type AllergySeverity string

const (
	AllergySeverityMild     AllergySeverity = "mild"
	AllergySeverityModerate AllergySeverity = "moderate"
	AllergySeveritySevere   AllergySeverity = "severe"
	AllergySeverityLifeThreatening AllergySeverity = "life_threatening"
)

// AllergyStatus represents allergy status
type AllergyStatus string

const (
	AllergyStatusActive   AllergyStatus = "active"
	AllergyStatusInactive AllergyStatus = "inactive"
	AllergyStatusResolved AllergyStatus = "resolved"
)

// ConditionEntry represents a medical condition in the snapshot
type ConditionEntry struct {
	ID             uuid.UUID        `json:"id"`
	ConditionName  string           `json:"condition_name"`
	ICD10Code      string           `json:"icd10_code,omitempty"`
	SNOMEDCT       string           `json:"snomed_ct,omitempty"`
	Status         ConditionStatus  `json:"status"`
	Severity       ConditionSeverity `json:"severity,omitempty"`
	OnsetDate      *time.Time       `json:"onset_date,omitempty"`
	DiagnosedDate  *time.Time       `json:"diagnosed_date,omitempty"`
	ResolvedDate   *time.Time       `json:"resolved_date,omitempty"`
	DiagnosedBy    string           `json:"diagnosed_by"`
	Notes          string           `json:"notes,omitempty"`
	Stage          string           `json:"stage,omitempty"`
	Grade          string           `json:"grade,omitempty"`
}

// ConditionStatus represents medical condition status
type ConditionStatus string

const (
	ConditionStatusActive    ConditionStatus = "active"
	ConditionStatusRecurrence ConditionStatus = "recurrence"
	ConditionStatusRelapse   ConditionStatus = "relapse"
	ConditionStatusInactive  ConditionStatus = "inactive"
	ConditionStatusRemission ConditionStatus = "remission"
	ConditionStatusResolved  ConditionStatus = "resolved"
)

// ConditionSeverity represents medical condition severity
type ConditionSeverity string

const (
	ConditionSeverityMild     ConditionSeverity = "mild"
	ConditionSeverityModerate ConditionSeverity = "moderate"
	ConditionSeveritySevere   ConditionSeverity = "severe"
)

// ProcedureEntry represents a medical procedure in the snapshot
type ProcedureEntry struct {
	ID            uuid.UUID       `json:"id"`
	ProcedureName string          `json:"procedure_name"`
	CPTCode       string          `json:"cpt_code,omitempty"`
	SNOMEDCT      string          `json:"snomed_ct,omitempty"`
	Status        ProcedureStatus `json:"status"`
	PerformedDate time.Time       `json:"performed_date"`
	PerformedBy   string          `json:"performed_by"`
	Location      string          `json:"location,omitempty"`
	Indication    string          `json:"indication"`
	Outcome       string          `json:"outcome,omitempty"`
	Complications string          `json:"complications,omitempty"`
	Notes         string          `json:"notes,omitempty"`
}

// ProcedureStatus represents procedure status
type ProcedureStatus string

const (
	ProcedureStatusCompleted    ProcedureStatus = "completed"
	ProcedureStatusInProgress   ProcedureStatus = "in_progress"
	ProcedureStatusAborted      ProcedureStatus = "aborted"
	ProcedureStatusUnknown      ProcedureStatus = "unknown"
)

// ObservationEntry represents a clinical observation
type ObservationEntry struct {
	ID           uuid.UUID         `json:"id"`
	Category     ObservationCategory `json:"category"`
	Code         string            `json:"code"`
	Display      string            `json:"display"`
	Value        interface{}       `json:"value"`
	Unit         string            `json:"unit,omitempty"`
	Status       ObservationStatus `json:"status"`
	EffectiveDate time.Time        `json:"effective_date"`
	Observer     string            `json:"observer"`
	Method       string            `json:"method,omitempty"`
	BodySite     string            `json:"body_site,omitempty"`
	Interpretation string          `json:"interpretation,omitempty"`
	Notes        string            `json:"notes,omitempty"`
}

// ObservationCategory represents different categories of observations
type ObservationCategory string

const (
	ObsCategoryVitalSigns   ObservationCategory = "vital-signs"
	ObsCategoryLaboratory   ObservationCategory = "laboratory"
	ObsCategorySocialHistory ObservationCategory = "social-history"
	ObsCategoryExam         ObservationCategory = "exam"
	ObsCategoryImaging      ObservationCategory = "imaging"
	ObsCategoryProcedure    ObservationCategory = "procedure"
	ObsCategorySurvey       ObservationCategory = "survey"
)

// ObservationStatus represents observation status
type ObservationStatus string

const (
	ObsStatusFinal      ObservationStatus = "final"
	ObsStatusAmended    ObservationStatus = "amended"
	ObsStatusCorrected  ObservationStatus = "corrected"
	ObsStatusCancelled  ObservationStatus = "cancelled"
	ObsStatusPreliminary ObservationStatus = "preliminary"
)

// AssessmentScore represents clinical assessment scores
type AssessmentScore struct {
	ID          uuid.UUID       `json:"id"`
	ScoreName   string          `json:"score_name"`
	ScoreType   AssessmentType  `json:"score_type"`
	Value       interface{}     `json:"value"`
	MaxValue    *float64        `json:"max_value,omitempty"`
	Unit        string          `json:"unit,omitempty"`
	Status      AssessmentStatus `json:"status"`
	AssessedAt  time.Time       `json:"assessed_at"`
	AssessedBy  string          `json:"assessed_by"`
	Interpretation string       `json:"interpretation,omitempty"`
	Notes       string          `json:"notes,omitempty"`
}

// AssessmentType represents different types of assessments
type AssessmentType string

const (
	AssessmentRisk      AssessmentType = "risk"
	AssessmentFunctional AssessmentType = "functional"
	AssessmentCognitive AssessmentType = "cognitive"
	AssessmentPain      AssessmentType = "pain"
	AssessmentQuality   AssessmentType = "quality_of_life"
)

// AssessmentStatus represents assessment status
type AssessmentStatus string

const (
	AssessStatusComplete   AssessmentStatus = "complete"
	AssessStatusIncomplete AssessmentStatus = "incomplete"
	AssessStatusCorrected  AssessmentStatus = "corrected"
)

// RiskFactor represents clinical risk factors
type RiskFactor struct {
	ID          uuid.UUID    `json:"id"`
	Factor      string       `json:"factor"`
	Category    RiskCategory `json:"category"`
	Severity    RiskSeverity `json:"severity"`
	Present     bool         `json:"present"`
	Notes       string       `json:"notes,omitempty"`
	AssessedAt  time.Time    `json:"assessed_at"`
	AssessedBy  string       `json:"assessed_by"`
}

// RiskCategory represents different categories of risk factors
type RiskCategory string

const (
	RiskCategoryCardiovascular RiskCategory = "cardiovascular"
	RiskCategoryMetabolic      RiskCategory = "metabolic"
	RiskCategoryInfectious     RiskCategory = "infectious"
	RiskCategoryBehavioral     RiskCategory = "behavioral"
	RiskCategoryEnvironmental  RiskCategory = "environmental"
	RiskCategoryGenetic        RiskCategory = "genetic"
)

// RiskSeverity represents risk factor severity
type RiskSeverity string

const (
	RiskSeverityLow      RiskSeverity = "low"
	RiskSeverityModerate RiskSeverity = "moderate"
	RiskSeverityHigh     RiskSeverity = "high"
	RiskSeverityCritical RiskSeverity = "critical"
)

// FreshnessMetadata contains information about data freshness
type FreshnessMetadata struct {
	DataSources      map[string]DataSourceInfo `json:"data_sources"`
	FreshnessChecks  []FreshnessCheck          `json:"freshness_checks"`
	OverallFreshness FreshnessStatus           `json:"overall_freshness"`
	LastRefreshAt    time.Time                 `json:"last_refresh_at"`
	NextRefreshAt    time.Time                 `json:"next_refresh_at"`
}

// DataSourceInfo contains information about data sources
type DataSourceInfo struct {
	Source         string    `json:"source"`
	LastUpdated    time.Time `json:"last_updated"`
	RecordCount    int       `json:"record_count"`
	QualityScore   float64   `json:"quality_score"`
	Reliability    string    `json:"reliability"`
}

// FreshnessCheck represents a freshness validation check
type FreshnessCheck struct {
	DataType      string          `json:"data_type"`
	Required      bool            `json:"required"`
	MaxAge        time.Duration   `json:"max_age"`
	ActualAge     time.Duration   `json:"actual_age"`
	Status        FreshnessStatus `json:"status"`
	Message       string          `json:"message,omitempty"`
}

// FreshnessStatus represents data freshness status
type FreshnessStatus string

const (
	FreshnessStatusFresh   FreshnessStatus = "fresh"
	FreshnessStatusStale   FreshnessStatus = "stale"
	FreshnessStatusExpired FreshnessStatus = "expired"
	FreshnessStatusMissing FreshnessStatus = "missing"
)

// ValidationResults contains the results of snapshot validation
type ValidationResults struct {
	IsValid           bool                     `json:"is_valid"`
	ValidationScore   float64                  `json:"validation_score"`
	ComplianceChecks  []ComplianceCheck        `json:"compliance_checks"`
	DataQuality       DataQualityAssessment    `json:"data_quality"`
	SecurityChecks    []SecurityCheck          `json:"security_checks"`
	ValidationErrors  []ValidationError        `json:"validation_errors,omitempty"`
	ValidationWarnings []ValidationWarning     `json:"validation_warnings,omitempty"`
	ValidatedAt       time.Time                `json:"validated_at"`
	ValidatedBy       string                   `json:"validated_by"`
}

// ComplianceCheck represents regulatory compliance validation
type ComplianceCheck struct {
	Regulation string          `json:"regulation"`  // HIPAA, FDA, etc.
	Requirement string         `json:"requirement"`
	Status      ComplianceStatus `json:"status"`
	Details     string          `json:"details,omitempty"`
	Evidence    string          `json:"evidence,omitempty"`
}

// ComplianceStatus represents compliance check status
type ComplianceStatus string

const (
	ComplianceStatusPassed ComplianceStatus = "passed"
	ComplianceStatusFailed ComplianceStatus = "failed"
	ComplianceStatusPartial ComplianceStatus = "partial"
	ComplianceStatusNotApplicable ComplianceStatus = "not_applicable"
)

// DataQualityAssessment represents data quality metrics
type DataQualityAssessment struct {
	CompletenessScore float64                    `json:"completeness_score"`
	AccuracyScore     float64                    `json:"accuracy_score"`
	ConsistencyScore  float64                    `json:"consistency_score"`
	TimelinessScore   float64                    `json:"timeliness_score"`
	QualityIssues     []DataQualityIssue         `json:"quality_issues,omitempty"`
	OverallScore      float64                    `json:"overall_score"`
}

// DataQualityIssue represents a data quality issue
type DataQualityIssue struct {
	Category    QualityCategory `json:"category"`
	Severity    QualitySeverity `json:"severity"`
	Field       string          `json:"field"`
	Description string          `json:"description"`
	Impact      string          `json:"impact"`
	Recommendation string       `json:"recommendation,omitempty"`
}

// QualityCategory represents data quality issue categories
type QualityCategory string

const (
	QualityCategoryCompleteness QualityCategory = "completeness"
	QualityCategoryAccuracy     QualityCategory = "accuracy"
	QualityCategoryConsistency  QualityCategory = "consistency"
	QualityCategoryTimeliness   QualityCategory = "timeliness"
	QualityCategoryFormat       QualityCategory = "format"
)

// QualitySeverity represents data quality issue severity
type QualitySeverity string

const (
	QualitySeverityLow      QualitySeverity = "low"
	QualitySeverityMedium   QualitySeverity = "medium"
	QualitySeverityHigh     QualitySeverity = "high"
	QualitySeverityCritical QualitySeverity = "critical"
)

// SecurityCheck represents security validation checks
type SecurityCheck struct {
	CheckType   SecurityCheckType `json:"check_type"`
	Status      SecurityStatus    `json:"status"`
	Description string            `json:"description"`
	Details     string            `json:"details,omitempty"`
	Risk        SecurityRisk      `json:"risk"`
}

// SecurityCheckType represents different types of security checks
type SecurityCheckType string

const (
	SecurityCheckEncryption   SecurityCheckType = "encryption"
	SecurityCheckAccess       SecurityCheckType = "access_control"
	SecurityCheckAudit        SecurityCheckType = "audit_trail"
	SecurityCheckIntegrity    SecurityCheckType = "data_integrity"
	SecurityCheckAnonymization SecurityCheckType = "anonymization"
)

// SecurityStatus represents security check status
type SecurityStatus string

const (
	SecurityStatusSecure   SecurityStatus = "secure"
	SecurityStatusVulnerable SecurityStatus = "vulnerable"
	SecurityStatusUnknown  SecurityStatus = "unknown"
)

// SecurityRisk represents security risk levels
type SecurityRisk string

const (
	SecurityRiskLow      SecurityRisk = "low"
	SecurityRiskMedium   SecurityRisk = "medium"
	SecurityRiskHigh     SecurityRisk = "high"
	SecurityRiskCritical SecurityRisk = "critical"
)

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Code        string    `json:"code"`
	Message     string    `json:"message"`
	Field       string    `json:"field,omitempty"`
	Severity    string    `json:"severity"`
	Recommendation string `json:"recommendation,omitempty"`
}

// SnapshotAuditEntry represents an audit trail entry for the snapshot
type SnapshotAuditEntry struct {
	ID        uuid.UUID   `json:"id"`
	Action    AuditAction `json:"action"`
	Timestamp time.Time   `json:"timestamp"`
	UserID    string      `json:"user_id"`
	UserRole  string      `json:"user_role"`
	IPAddress string      `json:"ip_address,omitempty"`
	Details   string      `json:"details"`
	Changes   interface{} `json:"changes,omitempty"`
}

// AuditAction represents different audit actions
type AuditAction string

const (
	AuditActionCreated   AuditAction = "created"
	AuditActionViewed    AuditAction = "viewed"
	AuditActionValidated AuditAction = "validated"
	AuditActionSuperseded AuditAction = "superseded"
	AuditActionExpired   AuditAction = "expired"
	AuditActionDeleted   AuditAction = "deleted"
)

// Validate validates the clinical snapshot
func (cs *ClinicalSnapshot) Validate() error {
	if cs.PatientID == uuid.Nil {
		return NewValidationError("patient_id is required")
	}

	if cs.RecipeID == uuid.Nil {
		return NewValidationError("recipe_id is required")
	}

	if cs.ExpiresAt.Before(time.Now()) {
		return NewValidationError("snapshot has expired")
	}

	return nil
}

// IsValid checks if the snapshot is valid and not expired
func (cs *ClinicalSnapshot) IsValid() bool {
	return cs.Status == SnapshotStatusActive && 
		   cs.ExpiresAt.After(time.Now()) &&
		   cs.ValidationResults.IsValid
}

// IsExpired checks if the snapshot has expired
func (cs *ClinicalSnapshot) IsExpired() bool {
	return time.Now().After(cs.ExpiresAt)
}

// GetDataAge returns the age of different data types in the snapshot
func (cs *ClinicalSnapshot) GetDataAge() map[string]time.Duration {
	dataAge := make(map[string]time.Duration)
	now := time.Now()
	
	for dataType, sourceInfo := range cs.FreshnessMetadata.DataSources {
		dataAge[dataType] = now.Sub(sourceInfo.LastUpdated)
	}
	
	return dataAge
}

// HasCriticalSecurityIssues checks if the snapshot has critical security issues
func (cs *ClinicalSnapshot) HasCriticalSecurityIssues() bool {
	for _, check := range cs.ValidationResults.SecurityChecks {
		if check.Risk == SecurityRiskCritical || check.Risk == SecurityRiskHigh {
			return true
		}
	}
	return false
}

// GetQualityScore returns the overall data quality score
func (cs *ClinicalSnapshot) GetQualityScore() float64 {
	return cs.ValidationResults.DataQuality.OverallScore
}