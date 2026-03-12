package models

import (
	"encoding/json"
	"time"
)

// ============================================================================
// FHIR R4 Resource Models for CDSS
// ============================================================================
// These models represent the subset of FHIR R4 resources needed for clinical
// decision support evaluation. They support parsing patient data from FHIR
// Bundles and individual resources.

// Common FHIR terminology system URIs
const (
	SystemSNOMED  = "http://snomed.info/sct"
	SystemICD10   = "http://hl7.org/fhir/sid/icd-10"
	SystemICD10CM = "http://hl7.org/fhir/sid/icd-10-cm"
	SystemLOINC   = "http://loinc.org"
	SystemRxNorm  = "http://www.nlm.nih.gov/research/umls/rxnorm"
	SystemCPT     = "http://www.ama-assn.org/go/cpt"
	SystemNDC     = "http://hl7.org/fhir/sid/ndc"
	SystemATC     = "http://www.whocc.no/atc"
	SystemUCUM    = "http://unitsofmeasure.org"
)

// FHIR Bundle types
const (
	BundleTypeDocument      = "document"
	BundleTypeMessage       = "message"
	BundleTypeTransaction   = "transaction"
	BundleTypeSearchset     = "searchset"
	BundleTypeCollection    = "collection"
	BundleTypeBatch         = "batch"
	BundleTypeHistory       = "history"
)

// FHIR resource types
const (
	ResourceTypeBundle            = "Bundle"
	ResourceTypeCondition         = "Condition"
	ResourceTypeObservation       = "Observation"
	ResourceTypeMedicationRequest = "MedicationRequest"
	ResourceTypeProcedure         = "Procedure"
	ResourceTypePatient           = "Patient"
	ResourceTypeEncounter         = "Encounter"
	ResourceTypeDiagnosticReport  = "DiagnosticReport"
	ResourceTypeAllergyIntolerance = "AllergyIntolerance"
)

// ============================================================================
// Core FHIR Data Types
// ============================================================================

// Coding represents a FHIR Coding data type
// A reference to a code defined by a terminology system
type Coding struct {
	System       string `json:"system,omitempty"`
	Version      string `json:"version,omitempty"`
	Code         string `json:"code,omitempty"`
	Display      string `json:"display,omitempty"`
	UserSelected bool   `json:"userSelected,omitempty"`
}

// CodeableConcept represents a FHIR CodeableConcept data type
// A set of codes from terminologies that represent the same concept
type CodeableConcept struct {
	Coding []Coding `json:"coding,omitempty"`
	Text   string   `json:"text,omitempty"`
}

// Reference represents a FHIR Reference data type
type Reference struct {
	Reference string `json:"reference,omitempty"`
	Type      string `json:"type,omitempty"`
	Display   string `json:"display,omitempty"`
}

// Period represents a FHIR Period data type
type Period struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// Quantity represents a FHIR Quantity data type
type Quantity struct {
	Value      float64 `json:"value,omitempty"`
	Comparator string  `json:"comparator,omitempty"` // < | <= | >= | >
	Unit       string  `json:"unit,omitempty"`
	System     string  `json:"system,omitempty"`
	Code       string  `json:"code,omitempty"`
}

// Range represents a FHIR Range data type
type Range struct {
	Low  *Quantity `json:"low,omitempty"`
	High *Quantity `json:"high,omitempty"`
}

// Identifier represents a FHIR Identifier data type
type Identifier struct {
	Use    string   `json:"use,omitempty"` // usual | official | temp | secondary | old
	Type   *CodeableConcept `json:"type,omitempty"`
	System string   `json:"system,omitempty"`
	Value  string   `json:"value,omitempty"`
	Period *Period  `json:"period,omitempty"`
}

// Annotation represents a FHIR Annotation data type
type Annotation struct {
	AuthorReference *Reference `json:"authorReference,omitempty"`
	AuthorString    string     `json:"authorString,omitempty"`
	Time            *time.Time `json:"time,omitempty"`
	Text            string     `json:"text,omitempty"`
}

// ============================================================================
// FHIR Bundle
// ============================================================================

// FHIRBundle represents a FHIR R4 Bundle resource
type FHIRBundle struct {
	ResourceType string        `json:"resourceType"`
	ID           string        `json:"id,omitempty"`
	Type         string        `json:"type"` // document | message | transaction | searchset | collection | batch | history
	Total        int           `json:"total,omitempty"`
	Timestamp    *time.Time    `json:"timestamp,omitempty"`
	Entry        []BundleEntry `json:"entry,omitempty"`
}

// BundleEntry represents an entry in a FHIR Bundle
type BundleEntry struct {
	FullURL  string          `json:"fullUrl,omitempty"`
	Resource json.RawMessage `json:"resource,omitempty"`
}

// ============================================================================
// FHIR Condition
// ============================================================================

// FHIRCondition represents a FHIR R4 Condition resource
// Used for diagnoses, problems, and health concerns
type FHIRCondition struct {
	ResourceType string `json:"resourceType"`
	ID           string `json:"id,omitempty"`

	// Identifiers
	Identifier []Identifier `json:"identifier,omitempty"`

	// Clinical Status (required for active conditions)
	ClinicalStatus *CodeableConcept `json:"clinicalStatus,omitempty"` // active | recurrence | relapse | inactive | remission | resolved

	// Verification Status
	VerificationStatus *CodeableConcept `json:"verificationStatus,omitempty"` // unconfirmed | provisional | differential | confirmed | refuted | entered-in-error

	// Category (problem-list-item, encounter-diagnosis, health-concern)
	Category []CodeableConcept `json:"category,omitempty"`

	// Severity
	Severity *CodeableConcept `json:"severity,omitempty"` // severe | moderate | mild

	// Code - the condition being identified
	Code *CodeableConcept `json:"code,omitempty"`

	// Body Site
	BodySite []CodeableConcept `json:"bodySite,omitempty"`

	// Subject - the patient
	Subject *Reference `json:"subject,omitempty"`

	// Encounter context
	Encounter *Reference `json:"encounter,omitempty"`

	// Onset (when condition started)
	OnsetDateTime *time.Time `json:"onsetDateTime,omitempty"`
	OnsetAge      *Quantity  `json:"onsetAge,omitempty"`
	OnsetPeriod   *Period    `json:"onsetPeriod,omitempty"`
	OnsetRange    *Range     `json:"onsetRange,omitempty"`
	OnsetString   string     `json:"onsetString,omitempty"`

	// Abatement (when condition ended/resolved)
	AbatementDateTime *time.Time `json:"abatementDateTime,omitempty"`
	AbatementAge      *Quantity  `json:"abatementAge,omitempty"`
	AbatementPeriod   *Period    `json:"abatementPeriod,omitempty"`
	AbatementRange    *Range     `json:"abatementRange,omitempty"`
	AbatementString   string     `json:"abatementString,omitempty"`

	// Recording info
	RecordedDate *time.Time `json:"recordedDate,omitempty"`
	Recorder     *Reference `json:"recorder,omitempty"`
	Asserter     *Reference `json:"asserter,omitempty"`

	// Stage information
	Stage []ConditionStage `json:"stage,omitempty"`

	// Evidence supporting the condition
	Evidence []ConditionEvidence `json:"evidence,omitempty"`

	// Notes
	Note []Annotation `json:"note,omitempty"`
}

// ConditionStage represents staging information for a condition
type ConditionStage struct {
	Summary    *CodeableConcept `json:"summary,omitempty"`
	Assessment []Reference      `json:"assessment,omitempty"`
	Type       *CodeableConcept `json:"type,omitempty"`
}

// ConditionEvidence represents evidence supporting a condition
type ConditionEvidence struct {
	Code   []CodeableConcept `json:"code,omitempty"`
	Detail []Reference       `json:"detail,omitempty"`
}

// IsActive returns true if the condition has an active clinical status
func (c *FHIRCondition) IsActive() bool {
	if c.ClinicalStatus == nil {
		return false
	}
	for _, coding := range c.ClinicalStatus.Coding {
		if coding.Code == "active" || coding.Code == "recurrence" || coding.Code == "relapse" {
			return true
		}
	}
	return false
}

// GetPrimaryCoding returns the first coding from the condition code
func (c *FHIRCondition) GetPrimaryCoding() *Coding {
	if c.Code == nil || len(c.Code.Coding) == 0 {
		return nil
	}
	return &c.Code.Coding[0]
}

// ============================================================================
// FHIR Observation
// ============================================================================

// FHIRObservation represents a FHIR R4 Observation resource
// Used for lab results, vital signs, and clinical assessments
type FHIRObservation struct {
	ResourceType string `json:"resourceType"`
	ID           string `json:"id,omitempty"`

	// Identifiers
	Identifier []Identifier `json:"identifier,omitempty"`

	// Status (required)
	Status string `json:"status"` // registered | preliminary | final | amended | corrected | cancelled | entered-in-error | unknown

	// Category
	Category []CodeableConcept `json:"category,omitempty"` // vital-signs | laboratory | survey | exam | imaging | etc.

	// Code (required) - what was observed
	Code *CodeableConcept `json:"code,omitempty"`

	// Subject - the patient
	Subject *Reference `json:"subject,omitempty"`

	// Context
	Encounter *Reference `json:"encounter,omitempty"`
	Focus     []Reference `json:"focus,omitempty"`

	// When observed
	EffectiveDateTime *time.Time `json:"effectiveDateTime,omitempty"`
	EffectivePeriod   *Period    `json:"effectivePeriod,omitempty"`
	EffectiveInstant  *time.Time `json:"effectiveInstant,omitempty"`
	Issued            *time.Time `json:"issued,omitempty"`

	// Who/what performed the observation
	Performer []Reference `json:"performer,omitempty"`

	// Observation value (choice of types)
	ValueQuantity        *Quantity        `json:"valueQuantity,omitempty"`
	ValueCodeableConcept *CodeableConcept `json:"valueCodeableConcept,omitempty"`
	ValueString          string           `json:"valueString,omitempty"`
	ValueBoolean         *bool            `json:"valueBoolean,omitempty"`
	ValueInteger         *int             `json:"valueInteger,omitempty"`
	ValueRange           *Range           `json:"valueRange,omitempty"`
	ValueRatio           *Ratio           `json:"valueRatio,omitempty"`
	ValueTime            string           `json:"valueTime,omitempty"`
	ValueDateTime        *time.Time       `json:"valueDateTime,omitempty"`
	ValuePeriod          *Period          `json:"valuePeriod,omitempty"`

	// Data absent reason
	DataAbsentReason *CodeableConcept `json:"dataAbsentReason,omitempty"`

	// Interpretation
	Interpretation []CodeableConcept `json:"interpretation,omitempty"` // H | L | A | HH | LL | etc.

	// Notes
	Note []Annotation `json:"note,omitempty"`

	// Body site
	BodySite *CodeableConcept `json:"bodySite,omitempty"`

	// Method used
	Method *CodeableConcept `json:"method,omitempty"`

	// Specimen
	Specimen *Reference `json:"specimen,omitempty"`

	// Device
	Device *Reference `json:"device,omitempty"`

	// Reference range
	ReferenceRange []ObservationReferenceRange `json:"referenceRange,omitempty"`

	// Component observations (for panels/multi-part obs)
	Component []ObservationComponent `json:"component,omitempty"`
}

// Ratio represents a FHIR Ratio data type
type Ratio struct {
	Numerator   *Quantity `json:"numerator,omitempty"`
	Denominator *Quantity `json:"denominator,omitempty"`
}

// ObservationReferenceRange represents reference range for an observation
type ObservationReferenceRange struct {
	Low       *Quantity        `json:"low,omitempty"`
	High      *Quantity        `json:"high,omitempty"`
	Type      *CodeableConcept `json:"type,omitempty"`
	AppliesTo []CodeableConcept `json:"appliesTo,omitempty"`
	Age       *Range           `json:"age,omitempty"`
	Text      string           `json:"text,omitempty"`
}

// ObservationComponent represents a component of a multi-part observation
type ObservationComponent struct {
	Code                 *CodeableConcept            `json:"code,omitempty"`
	ValueQuantity        *Quantity                   `json:"valueQuantity,omitempty"`
	ValueCodeableConcept *CodeableConcept            `json:"valueCodeableConcept,omitempty"`
	ValueString          string                      `json:"valueString,omitempty"`
	ValueBoolean         *bool                       `json:"valueBoolean,omitempty"`
	ValueInteger         *int                        `json:"valueInteger,omitempty"`
	ValueRange           *Range                      `json:"valueRange,omitempty"`
	ValueRatio           *Ratio                      `json:"valueRatio,omitempty"`
	ValueTime            string                      `json:"valueTime,omitempty"`
	ValueDateTime        *time.Time                  `json:"valueDateTime,omitempty"`
	ValuePeriod          *Period                     `json:"valuePeriod,omitempty"`
	DataAbsentReason     *CodeableConcept            `json:"dataAbsentReason,omitempty"`
	Interpretation       []CodeableConcept           `json:"interpretation,omitempty"`
	ReferenceRange       []ObservationReferenceRange `json:"referenceRange,omitempty"`
}

// IsFinal returns true if the observation has a final status
func (o *FHIRObservation) IsFinal() bool {
	return o.Status == "final" || o.Status == "amended" || o.Status == "corrected"
}

// GetPrimaryCoding returns the first coding from the observation code
func (o *FHIRObservation) GetPrimaryCoding() *Coding {
	if o.Code == nil || len(o.Code.Coding) == 0 {
		return nil
	}
	return &o.Code.Coding[0]
}

// GetNumericValue returns the numeric value if available
func (o *FHIRObservation) GetNumericValue() (float64, bool) {
	if o.ValueQuantity != nil {
		return o.ValueQuantity.Value, true
	}
	if o.ValueInteger != nil {
		return float64(*o.ValueInteger), true
	}
	return 0, false
}

// IsAbnormal returns true if interpretation indicates an abnormal result
func (o *FHIRObservation) IsAbnormal() bool {
	abnormalCodes := map[string]bool{
		"H": true, "HH": true, "L": true, "LL": true,
		"A": true, "AA": true, "HU": true, "LU": true,
	}
	for _, interp := range o.Interpretation {
		for _, coding := range interp.Coding {
			if abnormalCodes[coding.Code] {
				return true
			}
		}
	}
	return false
}

// ============================================================================
// FHIR MedicationRequest
// ============================================================================

// FHIRMedicationRequest represents a FHIR R4 MedicationRequest resource
// Used for medication orders and prescriptions
type FHIRMedicationRequest struct {
	ResourceType string `json:"resourceType"`
	ID           string `json:"id,omitempty"`

	// Identifiers
	Identifier []Identifier `json:"identifier,omitempty"`

	// Status (required)
	Status string `json:"status"` // active | on-hold | cancelled | completed | entered-in-error | stopped | draft | unknown

	// Status Reason
	StatusReason *CodeableConcept `json:"statusReason,omitempty"`

	// Intent (required)
	Intent string `json:"intent"` // proposal | plan | order | original-order | reflex-order | filler-order | instance-order | option

	// Category
	Category []CodeableConcept `json:"category,omitempty"` // inpatient | outpatient | community | discharge

	// Priority
	Priority string `json:"priority,omitempty"` // routine | urgent | asap | stat

	// Medication (choice - CodeableConcept or Reference)
	MedicationCodeableConcept *CodeableConcept `json:"medicationCodeableConcept,omitempty"`
	MedicationReference       *Reference       `json:"medicationReference,omitempty"`

	// Subject - the patient
	Subject *Reference `json:"subject,omitempty"`

	// Context
	Encounter *Reference `json:"encounter,omitempty"`

	// Authorization/support
	SupportingInformation []Reference `json:"supportingInformation,omitempty"`

	// When requested
	AuthoredOn *time.Time `json:"authoredOn,omitempty"`

	// Requester
	Requester *Reference `json:"requester,omitempty"`

	// Performer
	Performer     *Reference `json:"performer,omitempty"`
	PerformerType *CodeableConcept `json:"performerType,omitempty"`

	// Recorder
	Recorder *Reference `json:"recorder,omitempty"`

	// Reasons
	ReasonCode      []CodeableConcept `json:"reasonCode,omitempty"`
	ReasonReference []Reference       `json:"reasonReference,omitempty"`

	// Insurance/coverage
	Insurance []Reference `json:"insurance,omitempty"`

	// Notes
	Note []Annotation `json:"note,omitempty"`

	// Dosage instructions
	DosageInstruction []Dosage `json:"dosageInstruction,omitempty"`

	// Dispense request
	DispenseRequest *DispenseRequest `json:"dispenseRequest,omitempty"`

	// Substitution
	Substitution *MedicationSubstitution `json:"substitution,omitempty"`

	// Prior prescriptions this replaces
	PriorPrescription *Reference `json:"priorPrescription,omitempty"`
}

// Dosage represents dosage instructions for a medication
type Dosage struct {
	Sequence                 int               `json:"sequence,omitempty"`
	Text                     string            `json:"text,omitempty"`
	AdditionalInstruction    []CodeableConcept `json:"additionalInstruction,omitempty"`
	PatientInstruction       string            `json:"patientInstruction,omitempty"`
	Timing                   *Timing           `json:"timing,omitempty"`
	AsNeededBoolean          *bool             `json:"asNeededBoolean,omitempty"`
	AsNeededCodeableConcept  *CodeableConcept  `json:"asNeededCodeableConcept,omitempty"`
	Site                     *CodeableConcept  `json:"site,omitempty"`
	Route                    *CodeableConcept  `json:"route,omitempty"`
	Method                   *CodeableConcept  `json:"method,omitempty"`
	DoseAndRate              []DoseAndRate     `json:"doseAndRate,omitempty"`
	MaxDosePerPeriod         *Ratio            `json:"maxDosePerPeriod,omitempty"`
	MaxDosePerAdministration *Quantity         `json:"maxDosePerAdministration,omitempty"`
	MaxDosePerLifetime       *Quantity         `json:"maxDosePerLifetime,omitempty"`
}

// DoseAndRate represents dose and rate information
type DoseAndRate struct {
	Type        *CodeableConcept `json:"type,omitempty"`
	DoseRange   *Range           `json:"doseRange,omitempty"`
	DoseQuantity *Quantity       `json:"doseQuantity,omitempty"`
	RateRatio   *Ratio           `json:"rateRatio,omitempty"`
	RateRange   *Range           `json:"rateRange,omitempty"`
	RateQuantity *Quantity       `json:"rateQuantity,omitempty"`
}

// Timing represents timing information
type Timing struct {
	Event  []time.Time   `json:"event,omitempty"`
	Repeat *TimingRepeat `json:"repeat,omitempty"`
	Code   *CodeableConcept `json:"code,omitempty"`
}

// TimingRepeat represents repeat information for timing
type TimingRepeat struct {
	BoundsDuration *Duration `json:"boundsDuration,omitempty"`
	BoundsRange    *Range    `json:"boundsRange,omitempty"`
	BoundsPeriod   *Period   `json:"boundsPeriod,omitempty"`
	Count          int       `json:"count,omitempty"`
	CountMax       int       `json:"countMax,omitempty"`
	Duration       float64   `json:"duration,omitempty"`
	DurationMax    float64   `json:"durationMax,omitempty"`
	DurationUnit   string    `json:"durationUnit,omitempty"` // s | min | h | d | wk | mo | a
	Frequency      int       `json:"frequency,omitempty"`
	FrequencyMax   int       `json:"frequencyMax,omitempty"`
	Period         float64   `json:"period,omitempty"`
	PeriodMax      float64   `json:"periodMax,omitempty"`
	PeriodUnit     string    `json:"periodUnit,omitempty"` // s | min | h | d | wk | mo | a
	DayOfWeek      []string  `json:"dayOfWeek,omitempty"` // mon | tue | wed | thu | fri | sat | sun
	TimeOfDay      []string  `json:"timeOfDay,omitempty"`
	When           []string  `json:"when,omitempty"` // MORN | AFT | EVE | NIGHT | etc.
	Offset         int       `json:"offset,omitempty"`
}

// Duration represents a FHIR Duration data type
type Duration struct {
	Value      float64 `json:"value,omitempty"`
	Comparator string  `json:"comparator,omitempty"`
	Unit       string  `json:"unit,omitempty"`
	System     string  `json:"system,omitempty"`
	Code       string  `json:"code,omitempty"`
}

// DispenseRequest represents dispense request details
type DispenseRequest struct {
	InitialFill               *InitialFill `json:"initialFill,omitempty"`
	DispenseInterval          *Duration    `json:"dispenseInterval,omitempty"`
	ValidityPeriod            *Period      `json:"validityPeriod,omitempty"`
	NumberOfRepeatsAllowed    int          `json:"numberOfRepeatsAllowed,omitempty"`
	Quantity                  *Quantity    `json:"quantity,omitempty"`
	ExpectedSupplyDuration    *Duration    `json:"expectedSupplyDuration,omitempty"`
	Performer                 *Reference   `json:"performer,omitempty"`
}

// InitialFill represents initial fill information
type InitialFill struct {
	Quantity *Quantity `json:"quantity,omitempty"`
	Duration *Duration `json:"duration,omitempty"`
}

// MedicationSubstitution represents substitution rules
type MedicationSubstitution struct {
	AllowedBoolean         *bool            `json:"allowedBoolean,omitempty"`
	AllowedCodeableConcept *CodeableConcept `json:"allowedCodeableConcept,omitempty"`
	Reason                 *CodeableConcept `json:"reason,omitempty"`
}

// IsActive returns true if the medication request has an active status
func (m *FHIRMedicationRequest) IsActive() bool {
	return m.Status == "active"
}

// GetMedicationCoding returns the primary coding for the medication
func (m *FHIRMedicationRequest) GetMedicationCoding() *Coding {
	if m.MedicationCodeableConcept != nil && len(m.MedicationCodeableConcept.Coding) > 0 {
		return &m.MedicationCodeableConcept.Coding[0]
	}
	return nil
}

// ============================================================================
// FHIR Procedure
// ============================================================================

// FHIRProcedure represents a FHIR R4 Procedure resource
// Used for surgical procedures, diagnostic procedures, and interventions
type FHIRProcedure struct {
	ResourceType string `json:"resourceType"`
	ID           string `json:"id,omitempty"`

	// Identifiers
	Identifier []Identifier `json:"identifier,omitempty"`

	// Instantiates
	InstantiatesCanonical []string `json:"instantiatesCanonical,omitempty"`
	InstantiatesUri       []string `json:"instantiatesUri,omitempty"`

	// Based on
	BasedOn []Reference `json:"basedOn,omitempty"`
	PartOf  []Reference `json:"partOf,omitempty"`

	// Status (required)
	Status string `json:"status"` // preparation | in-progress | not-done | on-hold | stopped | completed | entered-in-error | unknown

	// Status reason
	StatusReason *CodeableConcept `json:"statusReason,omitempty"`

	// Category
	Category *CodeableConcept `json:"category,omitempty"`

	// Code (required) - what procedure was performed
	Code *CodeableConcept `json:"code,omitempty"`

	// Subject - the patient
	Subject *Reference `json:"subject,omitempty"`

	// Context
	Encounter *Reference `json:"encounter,omitempty"`

	// When performed
	PerformedDateTime *time.Time `json:"performedDateTime,omitempty"`
	PerformedPeriod   *Period    `json:"performedPeriod,omitempty"`
	PerformedString   string     `json:"performedString,omitempty"`
	PerformedAge      *Quantity  `json:"performedAge,omitempty"`
	PerformedRange    *Range     `json:"performedRange,omitempty"`

	// Recorder
	Recorder *Reference `json:"recorder,omitempty"`

	// Asserter
	Asserter *Reference `json:"asserter,omitempty"`

	// Performers
	Performer []ProcedurePerformer `json:"performer,omitempty"`

	// Location
	Location *Reference `json:"location,omitempty"`

	// Reasons
	ReasonCode      []CodeableConcept `json:"reasonCode,omitempty"`
	ReasonReference []Reference       `json:"reasonReference,omitempty"`

	// Body site
	BodySite []CodeableConcept `json:"bodySite,omitempty"`

	// Outcome
	Outcome *CodeableConcept `json:"outcome,omitempty"`

	// Report
	Report []Reference `json:"report,omitempty"`

	// Complications
	Complication       []CodeableConcept `json:"complication,omitempty"`
	ComplicationDetail []Reference       `json:"complicationDetail,omitempty"`

	// Follow-up
	FollowUp []CodeableConcept `json:"followUp,omitempty"`

	// Notes
	Note []Annotation `json:"note,omitempty"`

	// Focal devices
	FocalDevice []ProcedureFocalDevice `json:"focalDevice,omitempty"`

	// Used references and codes
	UsedReference []Reference       `json:"usedReference,omitempty"`
	UsedCode      []CodeableConcept `json:"usedCode,omitempty"`
}

// ProcedurePerformer represents a performer of a procedure
type ProcedurePerformer struct {
	Function   *CodeableConcept `json:"function,omitempty"`
	Actor      *Reference       `json:"actor,omitempty"`
	OnBehalfOf *Reference       `json:"onBehalfOf,omitempty"`
}

// ProcedureFocalDevice represents a device manipulated during the procedure
type ProcedureFocalDevice struct {
	Action      *CodeableConcept `json:"action,omitempty"`
	Manipulated *Reference       `json:"manipulated,omitempty"`
}

// IsCompleted returns true if the procedure has a completed status
func (p *FHIRProcedure) IsCompleted() bool {
	return p.Status == "completed"
}

// GetPrimaryCoding returns the first coding from the procedure code
func (p *FHIRProcedure) GetPrimaryCoding() *Coding {
	if p.Code == nil || len(p.Code.Coding) == 0 {
		return nil
	}
	return &p.Code.Coding[0]
}

// ============================================================================
// FHIR AllergyIntolerance
// ============================================================================

// FHIRAllergyIntolerance represents a FHIR R4 AllergyIntolerance resource
type FHIRAllergyIntolerance struct {
	ResourceType string `json:"resourceType"`
	ID           string `json:"id,omitempty"`

	// Identifiers
	Identifier []Identifier `json:"identifier,omitempty"`

	// Clinical status
	ClinicalStatus *CodeableConcept `json:"clinicalStatus,omitempty"` // active | inactive | resolved

	// Verification status
	VerificationStatus *CodeableConcept `json:"verificationStatus,omitempty"` // unconfirmed | confirmed | refuted | entered-in-error

	// Type
	Type string `json:"type,omitempty"` // allergy | intolerance

	// Category
	Category []string `json:"category,omitempty"` // food | medication | environment | biologic

	// Criticality
	Criticality string `json:"criticality,omitempty"` // low | high | unable-to-assess

	// Code - what the patient is allergic to
	Code *CodeableConcept `json:"code,omitempty"`

	// Patient
	Patient *Reference `json:"patient,omitempty"`

	// Encounter
	Encounter *Reference `json:"encounter,omitempty"`

	// Onset
	OnsetDateTime *time.Time `json:"onsetDateTime,omitempty"`
	OnsetAge      *Quantity  `json:"onsetAge,omitempty"`
	OnsetPeriod   *Period    `json:"onsetPeriod,omitempty"`
	OnsetRange    *Range     `json:"onsetRange,omitempty"`
	OnsetString   string     `json:"onsetString,omitempty"`

	// Recorded date
	RecordedDate *time.Time `json:"recordedDate,omitempty"`

	// Recorder/asserter
	Recorder *Reference `json:"recorder,omitempty"`
	Asserter *Reference `json:"asserter,omitempty"`

	// Last occurrence
	LastOccurrence *time.Time `json:"lastOccurrence,omitempty"`

	// Notes
	Note []Annotation `json:"note,omitempty"`

	// Reactions
	Reaction []AllergyReaction `json:"reaction,omitempty"`
}

// AllergyReaction represents a reaction to an allergen
type AllergyReaction struct {
	Substance     *CodeableConcept  `json:"substance,omitempty"`
	Manifestation []CodeableConcept `json:"manifestation,omitempty"`
	Description   string            `json:"description,omitempty"`
	Onset         *time.Time        `json:"onset,omitempty"`
	Severity      string            `json:"severity,omitempty"` // mild | moderate | severe
	ExposureRoute *CodeableConcept  `json:"exposureRoute,omitempty"`
	Note          []Annotation      `json:"note,omitempty"`
}

// IsActive returns true if the allergy is active
func (a *FHIRAllergyIntolerance) IsActive() bool {
	if a.ClinicalStatus == nil {
		return true // Assume active if not specified
	}
	for _, coding := range a.ClinicalStatus.Coding {
		if coding.Code == "active" {
			return true
		}
	}
	return false
}

// GetPrimaryCoding returns the first coding from the allergy code
func (a *FHIRAllergyIntolerance) GetPrimaryCoding() *Coding {
	if a.Code == nil || len(a.Code.Coding) == 0 {
		return nil
	}
	return &a.Code.Coding[0]
}
