// Package models defines domain models for KB-9 Care Gaps Service.
package models

import (
	"time"
)

// MeasureType represents supported quality measures.
type MeasureType string

const (
	MeasureCMS122DiabetesHbA1c       MeasureType = "CMS122_DIABETES_HBA1C"
	MeasureCMS165BPControl           MeasureType = "CMS165_BP_CONTROL"
	MeasureCMS130ColorectalScreening MeasureType = "CMS130_COLORECTAL_SCREENING"
	MeasureCMS125BreastCancer        MeasureType = "CMS125_BREAST_CANCER_SCREENING"
	MeasureCMS127PneumoniaVaccine    MeasureType = "CMS127_PNEUMONIA_VACCINE"
	MeasureCMS138TobaccoScreening    MeasureType = "CMS138_TOBACCO_SCREENING"
	MeasureCMS139FallsScreening      MeasureType = "CMS139_FALLS_SCREENING"
	MeasureCMS147FluVaccine          MeasureType = "CMS147_FLU_VACCINE"
	MeasureCMS154URITreatment        MeasureType = "CMS154_URI_TREATMENT"
	MeasureCMS156HighRiskMeds        MeasureType = "CMS156_HIGH_RISK_MEDS"
	MeasureCMS69BMIScreening         MeasureType = "CMS69_BMI_SCREENING"
	MeasureCMS2DepressionScreening   MeasureType = "CMS2_DEPRESSION_SCREENING"
	MeasureIndiaDiabetesCare         MeasureType = "INDIA_DIABETES_CARE"
	MeasureIndiaHypertensionCare     MeasureType = "INDIA_HYPERTENSION_CARE"
)

// GapStatus represents the status of a care gap.
type GapStatus string

const (
	GapStatusOpen          GapStatus = "OPEN"
	GapStatusClosed        GapStatus = "CLOSED"
	GapStatusPending       GapStatus = "PENDING"
	GapStatusNotApplicable GapStatus = "NOT_APPLICABLE"
	GapStatusExcluded      GapStatus = "EXCLUDED"
)

// GapPriority represents the priority of a care gap.
type GapPriority string

const (
	GapPriorityCritical GapPriority = "CRITICAL" // Time-sensitive, overdue gaps
	GapPriorityUrgent   GapPriority = "URGENT"
	GapPriorityHigh     GapPriority = "HIGH"
	GapPriorityMedium   GapPriority = "MEDIUM"
	GapPriorityLow      GapPriority = "LOW"
)

// PopulationType represents measure population types.
type PopulationType string

const (
	PopulationInitial             PopulationType = "INITIAL_POPULATION"
	PopulationDenominator         PopulationType = "DENOMINATOR"
	PopulationDenominatorExcl     PopulationType = "DENOMINATOR_EXCLUSION"
	PopulationDenominatorExcept   PopulationType = "DENOMINATOR_EXCEPTION"
	PopulationNumerator           PopulationType = "NUMERATOR"
	PopulationNumeratorExclusion  PopulationType = "NUMERATOR_EXCLUSION"
)

// InterventionType represents types of interventions to close gaps.
type InterventionType string

const (
	InterventionLabOrder         InterventionType = "LAB_ORDER"
	InterventionMedicationOrder  InterventionType = "MEDICATION_ORDER"
	InterventionProcedureOrder   InterventionType = "PROCEDURE_ORDER"
	InterventionReferral         InterventionType = "REFERRAL"
	InterventionPatientEducation InterventionType = "PATIENT_EDUCATION"
	InterventionScreening        InterventionType = "SCREENING"
	InterventionVaccination      InterventionType = "VACCINATION"
	InterventionCounseling       InterventionType = "COUNSELING"
)

// DataCompleteness represents data quality for gap assessment.
type DataCompleteness string

const (
	DataComplete     DataCompleteness = "COMPLETE"
	DataPartial      DataCompleteness = "PARTIAL"
	DataInsufficient DataCompleteness = "INSUFFICIENT"
)

// Period represents a time period.
type Period struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ConstraintStatus represents the temporal status of a care obligation.
// Maps to KB-3's PathwayEngine constraint evaluation.
type ConstraintStatus string

const (
	ConstraintPending     ConstraintStatus = "PENDING"     // Not yet due
	ConstraintApproaching ConstraintStatus = "APPROACHING" // Within alert threshold
	ConstraintOverdue     ConstraintStatus = "OVERDUE"     // Past deadline, within grace
	ConstraintMissed      ConstraintStatus = "MISSED"      // Grace period expired
	ConstraintMet         ConstraintStatus = "MET"         // Obligation fulfilled
)

// TemporalInfo contains temporal enrichment from KB-3.
// This is added to care gaps to provide deadline information.
type TemporalInfo struct {
	// DueDate is when the care gap should be addressed
	DueDate *time.Time `json:"dueDate,omitempty"`

	// OverdueDate is when the gap becomes overdue (after grace period)
	OverdueDate *time.Time `json:"overdueDate,omitempty"`

	// GracePeriodDays before the gap is marked overdue
	GracePeriodDays int `json:"gracePeriodDays,omitempty"`

	// Status is the current temporal status
	Status ConstraintStatus `json:"status,omitempty"`

	// LastCompletedDate is when this care item was last fulfilled
	LastCompletedDate *time.Time `json:"lastCompletedDate,omitempty"`

	// NextDueDate is when the next occurrence is due (for recurring items)
	NextDueDate *time.Time `json:"nextDueDate,omitempty"`

	// DaysUntilDue is the number of days until due date (negative if overdue)
	DaysUntilDue int `json:"daysUntilDue,omitempty"`

	// DaysOverdue is the number of days past due (0 if not overdue)
	DaysOverdue int `json:"daysOverdue,omitempty"`

	// IsRecurring indicates if this is a recurring care obligation
	IsRecurring bool `json:"isRecurring,omitempty"`

	// RecurrenceMonths is the interval in months for recurring items
	RecurrenceMonths int `json:"recurrenceMonths,omitempty"`

	// SourcedFromKB3 indicates if this temporal info came from KB-3
	SourcedFromKB3 bool `json:"sourcedFromKB3,omitempty"`
}

// MeasureInfo contains information about a quality measure.
type MeasureInfo struct {
	Type            MeasureType `json:"type"`
	CMSID           string      `json:"cmsId"`
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	Domain          string      `json:"domain"`
	Steward         string      `json:"steward"`
	Version         string      `json:"version"`
	CQLLibrary      string      `json:"cqlLibrary"`
	GuidelineSource string      `json:"guidelineSource,omitempty"` // e.g., "ACC/AHA", "USPSTF"
}

// CareGap represents an individual care gap.
type CareGap struct {
	ID               string           `json:"id"`
	Measure          MeasureInfo      `json:"measure"`
	Status           GapStatus        `json:"status"`
	Priority         GapPriority      `json:"priority"`
	Reason           string           `json:"reason"`
	Recommendation   string           `json:"recommendation"`
	IdentifiedDate   time.Time        `json:"identifiedDate"`
	ClosedDate       *time.Time       `json:"closedDate,omitempty"`
	DueDate          *time.Time       `json:"dueDate,omitempty"`
	DaysUntilDue     *int             `json:"daysUntilDue,omitempty"`
	Evidence         *CQLEvidence     `json:"evidence,omitempty"`
	RelatedResources []FHIRReference  `json:"relatedResources,omitempty"`
	Interventions    []Intervention   `json:"interventions,omitempty"`

	// TemporalContext contains KB-3 temporal enrichment (Tier 7 integration)
	// This provides deadline awareness: when gaps are due, overdue, or approaching
	TemporalContext *TemporalInfo `json:"temporalContext,omitempty"`
}

// CQLEvidence contains evidence from CQL evaluation.
type CQLEvidence struct {
	LibraryID      string                `json:"libraryId"`
	LibraryVersion string                `json:"libraryVersion"`
	Populations    []PopulationMembership `json:"populations"`
	DataElements   []EvaluatedDataElement `json:"dataElements"`
	EvaluatedAt    time.Time             `json:"evaluatedAt"`
}

// PopulationMembership indicates patient membership in measure populations.
type PopulationMembership struct {
	Population PopulationType `json:"population"`
	IsMember   bool           `json:"isMember"`
	Reason     string         `json:"reason,omitempty"`
}

// EvaluatedDataElement represents a data element evaluated by CQL.
type EvaluatedDataElement struct {
	Name             string         `json:"name"`
	Value            *string        `json:"value,omitempty"`
	ValueDate        *time.Time     `json:"valueDate,omitempty"`
	ContributedToGap bool           `json:"contributedToGap"`
	SourceResource   *FHIRReference `json:"sourceResource,omitempty"`
}

// FHIRReference is a reference to a FHIR resource.
type FHIRReference struct {
	ResourceType string `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	Display      string `json:"display,omitempty"`
}

// Intervention suggests an action to close a gap.
type Intervention struct {
	Type        InterventionType `json:"type"`
	Description string           `json:"description"`
	Code        string           `json:"code,omitempty"`
	CodeSystem  string           `json:"codeSystem,omitempty"`
	Priority    GapPriority      `json:"priority"`
}

// CareGapSummary provides summary statistics for care gaps.
type CareGapSummary struct {
	TotalOpenGaps    int              `json:"totalOpenGaps"`
	UrgentGaps       int              `json:"urgentGaps"`
	HighPriorityGaps int              `json:"highPriorityGaps"`
	GapsByDomain     []DomainGapCount `json:"gapsByDomain"`
	QualityScore     *float64         `json:"qualityScore,omitempty"`
}

// DomainGapCount represents gap count by clinical domain.
type DomainGapCount struct {
	Domain string `json:"domain"`
	Count  int    `json:"count"`
}

// CareGapReport is the complete care gap report for a patient.
type CareGapReport struct {
	PatientID         string           `json:"patientId"`
	ReportDate        time.Time        `json:"reportDate"`
	MeasurementPeriod Period           `json:"measurementPeriod"`
	OpenGaps          []CareGap        `json:"openGaps"`
	ClosedGaps        []CareGap        `json:"closedGaps,omitempty"`
	UpcomingDue       []CareGap        `json:"upcomingDue,omitempty"`
	Summary           CareGapSummary   `json:"summary"`
	DataCompleteness  DataCompleteness `json:"dataCompleteness"`
	Warnings          []string         `json:"warnings,omitempty"`
}

// MeasureReport represents a FHIR MeasureReport for a single patient.
type MeasureReport struct {
	ID                 string             `json:"id"`
	Measure            MeasureInfo        `json:"measure"`
	PatientID          string             `json:"patientId"`
	Period             Period             `json:"period"`
	Status             string             `json:"status"`
	Type               string             `json:"type"`
	Populations        []PopulationResult `json:"populations"`
	EvaluatedResources []FHIRReference    `json:"evaluatedResources,omitempty"`
	GeneratedAt        time.Time          `json:"generatedAt"`
}

// PopulationResult represents population result in measure report.
type PopulationResult struct {
	Population     PopulationType `json:"population"`
	Count          int            `json:"count"`
	SubjectResults []string       `json:"subjectResults,omitempty"`
}

// PopulationMeasureReport is a population-level measure report.
type PopulationMeasureReport struct {
	ID               string              `json:"id"`
	Measure          MeasureInfo         `json:"measure"`
	Period           Period              `json:"period"`
	TotalPatients    int                 `json:"totalPatients"`
	Populations      []PopulationResult  `json:"populations"`
	PerformanceRate  *float64            `json:"performanceRate,omitempty"`
	PatientsWithGaps []PatientGapSummary `json:"patientsWithGaps,omitempty"`
	GeneratedAt      time.Time           `json:"generatedAt"`
	ProcessingTimeMs int                 `json:"processingTimeMs"`
}

// PatientGapSummary is a summary of gaps for a single patient.
type PatientGapSummary struct {
	PatientID      string    `json:"patientId"`
	Status         GapStatus `json:"status"`
	Recommendation string    `json:"recommendation,omitempty"`
}
