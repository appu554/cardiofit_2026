// Package fhir provides FHIR R4 resource generation for medication decisions.
package fhir

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/medication-advisor-engine/advisor"
)

// MedicationRequest represents a FHIR R4 MedicationRequest resource
type MedicationRequest struct {
	ResourceType       string              `json:"resourceType"`
	ID                 string              `json:"id"`
	Meta               Meta                `json:"meta"`
	Status             string              `json:"status"`
	Intent             string              `json:"intent"`
	Category           []CodeableConcept   `json:"category,omitempty"`
	Priority           string              `json:"priority,omitempty"`
	MedicationCodeableConcept CodeableConcept `json:"medicationCodeableConcept"`
	Subject            Reference           `json:"subject"`
	Encounter          *Reference          `json:"encounter,omitempty"`
	AuthoredOn         string              `json:"authoredOn"`
	Requester          *Reference          `json:"requester,omitempty"`
	Recorder           *Reference          `json:"recorder,omitempty"`
	ReasonCode         []CodeableConcept   `json:"reasonCode,omitempty"`
	ReasonReference    []Reference         `json:"reasonReference,omitempty"`
	DosageInstruction  []Dosage            `json:"dosageInstruction"`
	Note               []Annotation        `json:"note,omitempty"`
	Extension          []Extension         `json:"extension,omitempty"`
}

// Meta contains resource metadata
type Meta struct {
	VersionID   string   `json:"versionId,omitempty"`
	LastUpdated string   `json:"lastUpdated"`
	Profile     []string `json:"profile,omitempty"`
	Security    []Coding `json:"security,omitempty"`
	Tag         []Coding `json:"tag,omitempty"`
}

// CodeableConcept represents a coded concept with display text
type CodeableConcept struct {
	Coding []Coding `json:"coding"`
	Text   string   `json:"text,omitempty"`
}

// Coding represents a coded value
type Coding struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display,omitempty"`
	Version string `json:"version,omitempty"`
}

// Reference represents a FHIR reference
type Reference struct {
	Reference string `json:"reference,omitempty"`
	Type      string `json:"type,omitempty"`
	Display   string `json:"display,omitempty"`
}

// Dosage represents dosage instructions
type Dosage struct {
	Sequence        int                    `json:"sequence,omitempty"`
	Text            string                 `json:"text,omitempty"`
	Timing          *Timing                `json:"timing,omitempty"`
	Route           *CodeableConcept       `json:"route,omitempty"`
	DoseAndRate     []DoseAndRate          `json:"doseAndRate,omitempty"`
	MaxDosePerPeriod *Ratio                `json:"maxDosePerPeriod,omitempty"`
}

// Timing represents timing instructions
type Timing struct {
	Repeat *Repeat `json:"repeat,omitempty"`
	Code   *CodeableConcept `json:"code,omitempty"`
}

// Repeat represents repeat timing
type Repeat struct {
	Frequency  int     `json:"frequency,omitempty"`
	Period     float64 `json:"period,omitempty"`
	PeriodUnit string  `json:"periodUnit,omitempty"`
	When       []string `json:"when,omitempty"`
}

// DoseAndRate represents dose and rate information
type DoseAndRate struct {
	Type       *CodeableConcept `json:"type,omitempty"`
	DoseQuantity *Quantity       `json:"doseQuantity,omitempty"`
}

// Quantity represents a measured amount
type Quantity struct {
	Value  float64 `json:"value"`
	Unit   string  `json:"unit"`
	System string  `json:"system,omitempty"`
	Code   string  `json:"code,omitempty"`
}

// Ratio represents a ratio of quantities
type Ratio struct {
	Numerator   Quantity `json:"numerator"`
	Denominator Quantity `json:"denominator"`
}

// Annotation represents a text annotation
type Annotation struct {
	AuthorString string `json:"authorString,omitempty"`
	Time         string `json:"time,omitempty"`
	Text         string `json:"text"`
}

// Extension represents a FHIR extension
type Extension struct {
	URL          string      `json:"url"`
	ValueString  string      `json:"valueString,omitempty"`
	ValueBoolean *bool       `json:"valueBoolean,omitempty"`
	ValueCode    string      `json:"valueCode,omitempty"`
	ValueReference *Reference `json:"valueReference,omitempty"`
}

// MedicationRequestBuilder builds FHIR MedicationRequest resources
type MedicationRequestBuilder struct {
	request MedicationRequest
}

// NewMedicationRequestBuilder creates a new builder
func NewMedicationRequestBuilder() *MedicationRequestBuilder {
	now := time.Now().UTC().Format(time.RFC3339)
	return &MedicationRequestBuilder{
		request: MedicationRequest{
			ResourceType: "MedicationRequest",
			ID:           uuid.New().String(),
			Meta: Meta{
				LastUpdated: now,
				Profile: []string{
					"http://hl7.org/fhir/us/core/StructureDefinition/us-core-medicationrequest",
				},
			},
			Status:     "active",
			Intent:     "order",
			AuthoredOn: now,
		},
	}
}

// FromProposal creates a MedicationRequest from a MedicationProposal
func (b *MedicationRequestBuilder) FromProposal(
	proposal advisor.MedicationProposal,
	patientID string,
	providerID string,
	indication string,
) *MedicationRequestBuilder {

	// Set medication
	b.request.MedicationCodeableConcept = CodeableConcept{
		Coding: []Coding{
			{
				System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
				Code:    proposal.Medication.Code,
				Display: proposal.Medication.Display,
			},
		},
		Text: proposal.Medication.Display,
	}

	// Set subject (patient)
	b.request.Subject = Reference{
		Reference: fmt.Sprintf("Patient/%s", patientID),
		Type:      "Patient",
	}

	// Set requester (provider)
	b.request.Requester = &Reference{
		Reference: fmt.Sprintf("Practitioner/%s", providerID),
		Type:      "Practitioner",
	}

	// Set reason code (indication)
	if indication != "" {
		b.request.ReasonCode = []CodeableConcept{
			{
				Text: indication,
			},
		}
	}

	// Set dosage instructions
	b.request.DosageInstruction = []Dosage{
		{
			Sequence: 1,
			Text:     b.buildDosageText(proposal.Dosage),
			Route:    b.buildRoute(proposal.Dosage.Route),
			Timing:   b.buildTiming(proposal.Dosage.Frequency),
			DoseAndRate: []DoseAndRate{
				{
					DoseQuantity: &Quantity{
						Value:  proposal.Dosage.Value,
						Unit:   proposal.Dosage.Unit,
						System: "http://unitsofmeasure.org",
					},
				},
			},
		},
	}

	// Add rationale as note
	if proposal.Rationale != "" {
		b.request.Note = append(b.request.Note, Annotation{
			AuthorString: "Medication Advisor Engine",
			Time:         b.request.AuthoredOn,
			Text:         proposal.Rationale,
		})
	}

	// Add warnings as notes
	for _, warning := range proposal.Warnings {
		b.request.Note = append(b.request.Note, Annotation{
			AuthorString: warning.Source,
			Time:         b.request.AuthoredOn,
			Text:         fmt.Sprintf("[%s] %s", warning.Severity, warning.Message),
		})
	}

	return b
}

// WithEvidenceExtension adds evidence envelope reference extension
func (b *MedicationRequestBuilder) WithEvidenceExtension(envelopeID string) *MedicationRequestBuilder {
	b.request.Extension = append(b.request.Extension, Extension{
		URL:         "http://cardiofit.com/fhir/StructureDefinition/evidence-envelope",
		ValueString: envelopeID,
	})
	return b
}

// WithQualityScoreExtension adds quality score extension
func (b *MedicationRequestBuilder) WithQualityScoreExtension(score float64) *MedicationRequestBuilder {
	b.request.Extension = append(b.request.Extension, Extension{
		URL:         "http://cardiofit.com/fhir/StructureDefinition/quality-score",
		ValueString: fmt.Sprintf("%.2f", score),
	})
	return b
}

// WithSnapshotExtension adds snapshot reference extension
func (b *MedicationRequestBuilder) WithSnapshotExtension(snapshotID string) *MedicationRequestBuilder {
	b.request.Extension = append(b.request.Extension, Extension{
		URL:         "http://cardiofit.com/fhir/StructureDefinition/clinical-snapshot",
		ValueString: snapshotID,
	})
	return b
}

// WithPriority sets the priority
func (b *MedicationRequestBuilder) WithPriority(priority string) *MedicationRequestBuilder {
	b.request.Priority = priority
	return b
}

// WithEncounter sets the encounter reference
func (b *MedicationRequestBuilder) WithEncounter(encounterID string) *MedicationRequestBuilder {
	b.request.Encounter = &Reference{
		Reference: fmt.Sprintf("Encounter/%s", encounterID),
		Type:      "Encounter",
	}
	return b
}

// Build returns the completed MedicationRequest
func (b *MedicationRequestBuilder) Build() *MedicationRequest {
	return &b.request
}

// Helper methods

func (b *MedicationRequestBuilder) buildDosageText(dosage advisor.Dosage) string {
	text := fmt.Sprintf("%.1f %s", dosage.Value, dosage.Unit)
	if dosage.Route != "" {
		text += fmt.Sprintf(" %s", dosage.Route)
	}
	if dosage.Frequency != "" {
		text += fmt.Sprintf(" %s", dosage.Frequency)
	}
	return text
}

func (b *MedicationRequestBuilder) buildRoute(route string) *CodeableConcept {
	if route == "" {
		return nil
	}

	routeCodes := map[string]Coding{
		"oral": {
			System:  "http://snomed.info/sct",
			Code:    "26643006",
			Display: "Oral route",
		},
		"iv": {
			System:  "http://snomed.info/sct",
			Code:    "47625008",
			Display: "Intravenous route",
		},
		"im": {
			System:  "http://snomed.info/sct",
			Code:    "78421000",
			Display: "Intramuscular route",
		},
		"sc": {
			System:  "http://snomed.info/sct",
			Code:    "34206005",
			Display: "Subcutaneous route",
		},
	}

	if coding, ok := routeCodes[route]; ok {
		return &CodeableConcept{
			Coding: []Coding{coding},
			Text:   coding.Display,
		}
	}

	return &CodeableConcept{
		Text: route,
	}
}

func (b *MedicationRequestBuilder) buildTiming(frequency string) *Timing {
	if frequency == "" {
		return nil
	}

	// Parse common frequencies
	timingCodes := map[string]Repeat{
		"once daily": {Frequency: 1, Period: 1, PeriodUnit: "d"},
		"twice daily": {Frequency: 2, Period: 1, PeriodUnit: "d"},
		"three times daily": {Frequency: 3, Period: 1, PeriodUnit: "d"},
		"four times daily": {Frequency: 4, Period: 1, PeriodUnit: "d"},
		"every 12 hours": {Frequency: 1, Period: 12, PeriodUnit: "h"},
		"every 8 hours": {Frequency: 1, Period: 8, PeriodUnit: "h"},
		"every 6 hours": {Frequency: 1, Period: 6, PeriodUnit: "h"},
		"weekly": {Frequency: 1, Period: 1, PeriodUnit: "wk"},
	}

	if repeat, ok := timingCodes[frequency]; ok {
		return &Timing{
			Repeat: &repeat,
		}
	}

	return &Timing{
		Code: &CodeableConcept{
			Text: frequency,
		},
	}
}

// CreateMedicationRequestFromCommit creates a FHIR MedicationRequest from commit data
func CreateMedicationRequestFromCommit(
	proposal advisor.MedicationProposal,
	patientID string,
	providerID string,
	snapshotID string,
	envelopeID string,
	indication string,
) *MedicationRequest {
	builder := NewMedicationRequestBuilder()

	return builder.
		FromProposal(proposal, patientID, providerID, indication).
		WithSnapshotExtension(snapshotID).
		WithEvidenceExtension(envelopeID).
		WithQualityScoreExtension(proposal.QualityScore).
		Build()
}
