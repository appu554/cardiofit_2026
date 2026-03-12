package entities

import (
	"fmt"
	"strings"
	"time"
)

// FHIRMedication represents a FHIR Medication resource
type FHIRMedication struct {
	ID           string               `json:"id" db:"id"`
	ResourceType string               `json:"resourceType" db:"resource_type"`
	Identifiers  []*FHIRIdentifier    `json:"identifier,omitempty" db:"identifiers"`
	Code         *FHIRCodeableConcept `json:"code,omitempty" db:"code"`
	Status       string               `json:"status" db:"status"`
	Manufacturer *FHIRReference       `json:"manufacturer,omitempty" db:"manufacturer"`
	Form         *FHIRCodeableConcept `json:"form,omitempty" db:"form"`
	Amount       *FHIRRatio           `json:"amount,omitempty" db:"amount"`
	Ingredients  []*FHIRMedicationIngredient `json:"ingredient,omitempty" db:"ingredients"`
	Batch        *FHIRMedicationBatch `json:"batch,omitempty" db:"batch"`
	CreatedAt    time.Time            `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time            `json:"updatedAt" db:"updated_at"`
}

// FHIRMedicationIngredient represents a Medication.ingredient
type FHIRMedicationIngredient struct {
	ItemCodeableConcept *FHIRCodeableConcept `json:"itemCodeableConcept,omitempty"`
	ItemReference       *FHIRReference       `json:"itemReference,omitempty"`
	IsActive            *bool                `json:"isActive,omitempty"`
	Strength            *FHIRRatio           `json:"strength,omitempty"`
}

// FHIRMedicationBatch represents a Medication.batch
type FHIRMedicationBatch struct {
	LotNumber      string     `json:"lotNumber,omitempty"`
	ExpirationDate *time.Time `json:"expirationDate,omitempty"`
}

// FHIRMedicationRequest represents a FHIR MedicationRequest resource
type FHIRMedicationRequest struct {
	ID                        string                      `json:"id" db:"id"`
	ResourceType              string                      `json:"resourceType" db:"resource_type"`
	Identifiers               []*FHIRIdentifier           `json:"identifier,omitempty" db:"identifiers"`
	Status                    string                      `json:"status" db:"status"`
	StatusReason              *FHIRCodeableConcept        `json:"statusReason,omitempty" db:"status_reason"`
	Intent                    string                      `json:"intent" db:"intent"`
	Category                  []*FHIRCodeableConcept      `json:"category,omitempty" db:"category"`
	Priority                  string                      `json:"priority,omitempty" db:"priority"`
	DoNotPerform              *bool                       `json:"doNotPerform,omitempty" db:"do_not_perform"`
	ReportedBoolean           *bool                       `json:"reportedBoolean,omitempty" db:"reported_boolean"`
	ReportedReference         *FHIRReference              `json:"reportedReference,omitempty" db:"reported_reference"`
	MedicationCodeableConcept *FHIRCodeableConcept        `json:"medicationCodeableConcept,omitempty" db:"medication_codeable_concept"`
	MedicationReference       *FHIRReference              `json:"medicationReference,omitempty" db:"medication_reference"`
	Subject                   *FHIRReference              `json:"subject" db:"subject"`
	Encounter                 *FHIRReference              `json:"encounter,omitempty" db:"encounter"`
	SupportingInformation     []*FHIRReference            `json:"supportingInformation,omitempty" db:"supporting_information"`
	AuthoredOn                *time.Time                  `json:"authoredOn,omitempty" db:"authored_on"`
	Requester                 *FHIRReference              `json:"requester,omitempty" db:"requester"`
	Performer                 *FHIRReference              `json:"performer,omitempty" db:"performer"`
	PerformerType             *FHIRCodeableConcept        `json:"performerType,omitempty" db:"performer_type"`
	Recorder                  *FHIRReference              `json:"recorder,omitempty" db:"recorder"`
	ReasonCode                []*FHIRCodeableConcept      `json:"reasonCode,omitempty" db:"reason_code"`
	ReasonReference           []*FHIRReference            `json:"reasonReference,omitempty" db:"reason_reference"`
	InstantiatesCanonical     []string                    `json:"instantiatesCanonical,omitempty" db:"instantiates_canonical"`
	InstantiatesUri           []string                    `json:"instantiatesUri,omitempty" db:"instantiates_uri"`
	BasedOn                   []*FHIRReference            `json:"basedOn,omitempty" db:"based_on"`
	GroupIdentifier           *FHIRIdentifier             `json:"groupIdentifier,omitempty" db:"group_identifier"`
	CourseOfTherapyType       *FHIRCodeableConcept        `json:"courseOfTherapyType,omitempty" db:"course_of_therapy_type"`
	Insurance                 []*FHIRReference            `json:"insurance,omitempty" db:"insurance"`
	Notes                     []*FHIRAnnotation           `json:"note,omitempty" db:"notes"`
	DosageInstructions        []*FHIRDosage               `json:"dosageInstruction,omitempty" db:"dosage_instructions"`
	DispenseRequest           *FHIRMedicationRequestDispenseRequest `json:"dispenseRequest,omitempty" db:"dispense_request"`
	Substitution              *FHIRMedicationRequestSubstitution    `json:"substitution,omitempty" db:"substitution"`
	PriorPrescription         *FHIRReference              `json:"priorPrescription,omitempty" db:"prior_prescription"`
	DetectedIssue             []*FHIRReference            `json:"detectedIssue,omitempty" db:"detected_issue"`
	EventHistory              []*FHIRReference            `json:"eventHistory,omitempty" db:"event_history"`
	CreatedAt                 time.Time                   `json:"createdAt" db:"created_at"`
	UpdatedAt                 time.Time                   `json:"updatedAt" db:"updated_at"`
}

// FHIRMedicationRequestDispenseRequest represents a MedicationRequest.dispenseRequest
type FHIRMedicationRequestDispenseRequest struct {
	InitialFill             *FHIRMedicationRequestInitialFill `json:"initialFill,omitempty"`
	DispenseInterval        *FHIRDuration                     `json:"dispenseInterval,omitempty"`
	ValidityPeriod          *FHIRPeriod                       `json:"validityPeriod,omitempty"`
	NumberOfRepeatsAllowed  *int                              `json:"numberOfRepeatsAllowed,omitempty"`
	Quantity                *FHIRQuantity                     `json:"quantity,omitempty"`
	ExpectedSupplyDuration  *FHIRDuration                     `json:"expectedSupplyDuration,omitempty"`
	Performer               *FHIRReference                    `json:"performer,omitempty"`
}

// FHIRMedicationRequestInitialFill represents a MedicationRequest.dispenseRequest.initialFill
type FHIRMedicationRequestInitialFill struct {
	Quantity *FHIRQuantity `json:"quantity,omitempty"`
	Duration *FHIRDuration `json:"duration,omitempty"`
}

// FHIRMedicationRequestSubstitution represents a MedicationRequest.substitution
type FHIRMedicationRequestSubstitution struct {
	AllowedBoolean          *bool                `json:"allowedBoolean,omitempty"`
	AllowedCodeableConcept  *FHIRCodeableConcept `json:"allowedCodeableConcept,omitempty"`
	Reason                  *FHIRCodeableConcept `json:"reason,omitempty"`
}

// FHIR Data Types

// FHIRCodeableConcept represents a FHIR CodeableConcept
type FHIRCodeableConcept struct {
	Text   string       `json:"text,omitempty"`
	Coding []*FHIRCoding `json:"coding,omitempty"`
}

// FHIRCoding represents a FHIR Coding
type FHIRCoding struct {
	System       string `json:"system,omitempty"`
	Version      string `json:"version,omitempty"`
	Code         string `json:"code,omitempty"`
	Display      string `json:"display,omitempty"`
	UserSelected *bool  `json:"userSelected,omitempty"`
}

// FHIRIdentifier represents a FHIR Identifier
type FHIRIdentifier struct {
	Use      string               `json:"use,omitempty"`
	Type     *FHIRCodeableConcept `json:"type,omitempty"`
	System   string               `json:"system,omitempty"`
	Value    string               `json:"value,omitempty"`
	Period   *FHIRPeriod          `json:"period,omitempty"`
	Assigner *FHIRReference       `json:"assigner,omitempty"`
}

// FHIRReference represents a FHIR Reference
type FHIRReference struct {
	Reference  string          `json:"reference,omitempty"`
	Type       string          `json:"type,omitempty"`
	Identifier *FHIRIdentifier `json:"identifier,omitempty"`
	Display    string          `json:"display,omitempty"`
}

// FHIRPeriod represents a FHIR Period
type FHIRPeriod struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
}

// FHIRQuantity represents a FHIR Quantity
type FHIRQuantity struct {
	Value      *float64 `json:"value,omitempty"`
	Comparator string   `json:"comparator,omitempty"`
	Unit       string   `json:"unit,omitempty"`
	System     string   `json:"system,omitempty"`
	Code       string   `json:"code,omitempty"`
}

// FHIRRatio represents a FHIR Ratio
type FHIRRatio struct {
	Numerator   *FHIRQuantity `json:"numerator,omitempty"`
	Denominator *FHIRQuantity `json:"denominator,omitempty"`
}

// FHIRRange represents a FHIR Range
type FHIRRange struct {
	Low  *FHIRQuantity `json:"low,omitempty"`
	High *FHIRQuantity `json:"high,omitempty"`
}

// FHIRDuration represents a FHIR Duration
type FHIRDuration struct {
	Value  *float64 `json:"value,omitempty"`
	Unit   string   `json:"unit,omitempty"`
	System string   `json:"system,omitempty"`
	Code   string   `json:"code,omitempty"`
}

// FHIRAnnotation represents a FHIR Annotation
type FHIRAnnotation struct {
	AuthorReference *FHIRReference `json:"authorReference,omitempty"`
	AuthorString    string         `json:"authorString,omitempty"`
	Time            *time.Time     `json:"time,omitempty"`
	Text            string         `json:"text"`
}

// FHIRDosage represents a FHIR Dosage instruction
type FHIRDosage struct {
	Sequence                   *int                      `json:"sequence,omitempty"`
	Text                       string                    `json:"text,omitempty"`
	AdditionalInstruction      []*FHIRCodeableConcept    `json:"additionalInstruction,omitempty"`
	PatientInstruction         string                    `json:"patientInstruction,omitempty"`
	Timing                     *FHIRTiming               `json:"timing,omitempty"`
	AsNeededBoolean            *bool                     `json:"asNeededBoolean,omitempty"`
	AsNeededCodeableConcept    *FHIRCodeableConcept      `json:"asNeededCodeableConcept,omitempty"`
	Site                       *FHIRCodeableConcept      `json:"site,omitempty"`
	Route                      *FHIRCodeableConcept      `json:"route,omitempty"`
	Method                     *FHIRCodeableConcept      `json:"method,omitempty"`
	DoseAndRate                []*FHIRDoseAndRate        `json:"doseAndRate,omitempty"`
	MaxDosePerPeriod           *FHIRRatio                `json:"maxDosePerPeriod,omitempty"`
	MaxDosePerAdministration   *FHIRQuantity             `json:"maxDosePerAdministration,omitempty"`
	MaxDosePerLifetime         *FHIRQuantity             `json:"maxDosePerLifetime,omitempty"`
}

// FHIRTiming represents a FHIR Timing
type FHIRTiming struct {
	Event  []*time.Time          `json:"event,omitempty"`
	Repeat *FHIRTimingRepeat     `json:"repeat,omitempty"`
	Code   *FHIRCodeableConcept  `json:"code,omitempty"`
}

// FHIRTimingRepeat represents a FHIR Timing.repeat
type FHIRTimingRepeat struct {
	BoundsRange        *FHIRRange       `json:"boundsRange,omitempty"`
	BoundsPeriod       *FHIRPeriod      `json:"boundsPeriod,omitempty"`
	BoundsQuantity     *FHIRQuantity    `json:"boundsQuantity,omitempty"`
	Count              *int             `json:"count,omitempty"`
	CountMax           *int             `json:"countMax,omitempty"`
	Duration           *float64         `json:"duration,omitempty"`
	DurationMax        *float64         `json:"durationMax,omitempty"`
	DurationUnit       string           `json:"durationUnit,omitempty"`
	Frequency          *int             `json:"frequency,omitempty"`
	FrequencyMax       *int             `json:"frequencyMax,omitempty"`
	Period             *float64         `json:"period,omitempty"`
	PeriodMax          *float64         `json:"periodMax,omitempty"`
	PeriodUnit         string           `json:"periodUnit,omitempty"`
	DayOfWeek          []string         `json:"dayOfWeek,omitempty"`
	TimeOfDay          []string         `json:"timeOfDay,omitempty"`
	When               []string         `json:"when,omitempty"`
	Offset             *int             `json:"offset,omitempty"`
}

// FHIRDoseAndRate represents a FHIR DoseAndRate
type FHIRDoseAndRate struct {
	Type         *FHIRCodeableConcept `json:"type,omitempty"`
	DoseRange    *FHIRRange           `json:"doseRange,omitempty"`
	DoseQuantity *FHIRQuantity        `json:"doseQuantity,omitempty"`
	RateRatio    *FHIRRatio           `json:"rateRatio,omitempty"`
	RateRange    *FHIRRange           `json:"rateRange,omitempty"`
	RateQuantity *FHIRQuantity        `json:"rateQuantity,omitempty"`
}

// Validation methods

// IsValid validates a FHIR Medication entity
func (m *FHIRMedication) IsValid() error {
	if m.ID == "" {
		return fmt.Errorf("medication ID is required")
	}

	if m.Status == "" {
		return fmt.Errorf("medication status is required")
	}

	validStatuses := []string{"active", "inactive", "entered-in-error"}
	if !contains(validStatuses, m.Status) {
		return fmt.Errorf("invalid medication status: %s", m.Status)
	}

	return nil
}

// IsValid validates a FHIR MedicationRequest entity
func (mr *FHIRMedicationRequest) IsValid() error {
	if mr.ID == "" {
		return fmt.Errorf("medication request ID is required")
	}

	if mr.Status == "" {
		return fmt.Errorf("medication request status is required")
	}

	validStatuses := []string{
		"active", "on-hold", "cancelled", "completed",
		"entered-in-error", "stopped", "draft", "unknown",
	}
	if !contains(validStatuses, mr.Status) {
		return fmt.Errorf("invalid medication request status: %s", mr.Status)
	}

	if mr.Intent == "" {
		return fmt.Errorf("medication request intent is required")
	}

	validIntents := []string{
		"proposal", "plan", "order", "original-order",
		"reflex-order", "filler-order", "instance-order", "option",
	}
	if !contains(validIntents, mr.Intent) {
		return fmt.Errorf("invalid medication request intent: %s", mr.Intent)
	}

	if mr.Subject == nil {
		return fmt.Errorf("medication request subject is required")
	}

	if mr.MedicationCodeableConcept == nil && mr.MedicationReference == nil {
		return fmt.Errorf("medication request must have either medicationCodeableConcept or medicationReference")
	}

	return nil
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// String returns a string representation of the medication
func (m *FHIRMedication) String() string {
	return fmt.Sprintf("Medication{ID: %s, Status: %s}", m.ID, m.Status)
}

// String returns a string representation of the medication request
func (mr *FHIRMedicationRequest) String() string {
	return fmt.Sprintf("MedicationRequest{ID: %s, Status: %s, Intent: %s}", mr.ID, mr.Status, mr.Intent)
}

// GetSubjectID extracts the patient ID from the subject reference
func (mr *FHIRMedicationRequest) GetSubjectID() string {
	if mr.Subject == nil || mr.Subject.Reference == "" {
		return ""
	}

	// Extract ID from reference like "Patient/123" -> "123"
	parts := strings.Split(mr.Subject.Reference, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}

	return mr.Subject.Reference
}

// GetRequesterID extracts the requester ID from the requester reference
func (mr *FHIRMedicationRequest) GetRequesterID() string {
	if mr.Requester == nil || mr.Requester.Reference == "" {
		return ""
	}

	// Extract ID from reference like "Practitioner/123" -> "123"
	parts := strings.Split(mr.Requester.Reference, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}

	return mr.Requester.Reference
}

// GetEncounterID extracts the encounter ID from the encounter reference
func (mr *FHIRMedicationRequest) GetEncounterID() string {
	if mr.Encounter == nil || mr.Encounter.Reference == "" {
		return ""
	}

	// Extract ID from reference like "Encounter/123" -> "123"
	parts := strings.Split(mr.Encounter.Reference, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-1]
	}

	return mr.Encounter.Reference
}

// Convert from existing medication proposal to FHIR entities

// ToFHIRMedicationRequest converts a MedicationProposal to a FHIR MedicationRequest
func (mp *MedicationProposal) ToFHIRMedicationRequest() *FHIRMedicationRequest {
	mr := &FHIRMedicationRequest{
		ID:           mp.ID.String(),
		ResourceType: "MedicationRequest",
		Status:       mp.getFHIRStatus(),
		Intent:       "order",
		Subject: &FHIRReference{
			Reference: fmt.Sprintf("Patient/%s", mp.PatientID.String()),
		},
		AuthoredOn: &mp.CreatedAt,
		CreatedAt:  mp.CreatedAt,
		UpdatedAt:  mp.UpdatedAt,
	}

	// Convert medication details to CodeableConcept
	if mp.MedicationDetails != nil {
		mr.MedicationCodeableConcept = &FHIRCodeableConcept{
			Text: mp.MedicationDetails.DrugName,
			Coding: []*FHIRCoding{
				{
					Display: mp.MedicationDetails.DrugName,
					Code:    mp.MedicationDetails.GenericName,
				},
			},
		}
	}

	// Convert dosage recommendations to dosage instructions
	for _, rec := range mp.DosageRecommendations {
		dosage := &FHIRDosage{
			Text: fmt.Sprintf("%g mg %s", rec.DoseMg, rec.Route),
			PatientInstruction: rec.ClinicalNotes,
		}

		if rec.FrequencyPerDay > 0 {
			dosage.Timing = &FHIRTiming{
				Repeat: &FHIRTimingRepeat{
					Frequency: &rec.FrequencyPerDay,
					Period:    toFloat64Ptr(1.0),
					PeriodUnit: "d",
				},
			}
		}

		dosage.DoseAndRate = []*FHIRDoseAndRate{
			{
				DoseQuantity: &FHIRQuantity{
					Value: &rec.DoseMg,
					Unit:  "mg",
					System: "http://unitsofmeasure.org",
					Code:  "mg",
				},
			},
		}

		mr.DosageInstructions = append(mr.DosageInstructions, dosage)
	}

	// Add clinical notes
	if mp.ClinicalContext != nil {
		note := &FHIRAnnotation{
			Text:         fmt.Sprintf("Clinical context: Age %d, Weight %v kg", mp.ClinicalContext.AgeYears, mp.ClinicalContext.WeightKg),
			AuthorString: mp.CreatedBy,
			Time:         &mp.CreatedAt,
		}
		mr.Notes = append(mr.Notes, note)
	}

	return mr
}

// getFHIRStatus converts proposal status to FHIR status
func (mp *MedicationProposal) getFHIRStatus() string {
	switch mp.Status {
	case ProposalStatusDraft:
		return "draft"
	case ProposalStatusProposed:
		return "active"
	case ProposalStatusValidated:
		return "active"
	case ProposalStatusCommitted:
		return "completed"
	case ProposalStatusRejected:
		return "cancelled"
	case ProposalStatusExpired:
		return "entered-in-error"
	default:
		return "unknown"
	}
}

// Helper function
func toFloat64Ptr(f float64) *float64 {
	return &f
}