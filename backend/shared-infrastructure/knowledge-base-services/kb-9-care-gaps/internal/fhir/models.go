// Package fhir provides FHIR client and resource models for KB-9 Care Gaps Service.
package fhir

import (
	"time"
)

// ============================================================================
// FHIR Bundle Types
// ============================================================================

// Bundle represents a FHIR Bundle resource.
type Bundle struct {
	ResourceType string        `json:"resourceType"`
	Type         string        `json:"type"`
	Total        int           `json:"total,omitempty"`
	Link         []BundleLink  `json:"link,omitempty"`
	Entry        []BundleEntry `json:"entry,omitempty"`
}

// BundleLink represents a link in a Bundle.
type BundleLink struct {
	Relation string `json:"relation"`
	URL      string `json:"url"`
}

// BundleEntry represents an entry in a Bundle.
type BundleEntry struct {
	FullURL  string      `json:"fullUrl,omitempty"`
	Resource interface{} `json:"resource,omitempty"`
	Search   *Search     `json:"search,omitempty"`
}

// Search contains search-related information for a bundle entry.
type Search struct {
	Mode  string  `json:"mode,omitempty"`
	Score float64 `json:"score,omitempty"`
}

// ============================================================================
// Core FHIR Resources
// ============================================================================

// Patient represents a FHIR Patient resource.
type Patient struct {
	ResourceType string           `json:"resourceType"`
	ID           string           `json:"id,omitempty"`
	Identifier   []Identifier     `json:"identifier,omitempty"`
	Active       bool             `json:"active,omitempty"`
	Name         []HumanName      `json:"name,omitempty"`
	Telecom      []ContactPoint   `json:"telecom,omitempty"`
	Gender       string           `json:"gender,omitempty"`
	BirthDate    string           `json:"birthDate,omitempty"`
	Address      []Address        `json:"address,omitempty"`
	MaritalStatus *CodeableConcept `json:"maritalStatus,omitempty"`
}

// Condition represents a FHIR Condition resource.
type Condition struct {
	ResourceType       string           `json:"resourceType"`
	ID                 string           `json:"id,omitempty"`
	ClinicalStatus     *CodeableConcept `json:"clinicalStatus,omitempty"`
	VerificationStatus *CodeableConcept `json:"verificationStatus,omitempty"`
	Category           []CodeableConcept `json:"category,omitempty"`
	Severity           *CodeableConcept `json:"severity,omitempty"`
	Code               *CodeableConcept `json:"code,omitempty"`
	Subject            *Reference       `json:"subject,omitempty"`
	OnsetDateTime      string           `json:"onsetDateTime,omitempty"`
	RecordedDate       string           `json:"recordedDate,omitempty"`
}

// Observation represents a FHIR Observation resource.
type Observation struct {
	ResourceType         string                 `json:"resourceType"`
	ID                   string                 `json:"id,omitempty"`
	Status               string                 `json:"status,omitempty"`
	Category             []CodeableConcept      `json:"category,omitempty"`
	Code                 *CodeableConcept       `json:"code,omitempty"`
	Subject              *Reference             `json:"subject,omitempty"`
	EffectiveDateTime    string                 `json:"effectiveDateTime,omitempty"`
	Issued               string                 `json:"issued,omitempty"`
	ValueQuantity        *Quantity              `json:"valueQuantity,omitempty"`
	ValueCodeableConcept *CodeableConcept       `json:"valueCodeableConcept,omitempty"`
	ValueString          string                 `json:"valueString,omitempty"`
	Interpretation       []CodeableConcept      `json:"interpretation,omitempty"`
	ReferenceRange       []ReferenceRange       `json:"referenceRange,omitempty"`
	Component            []ObservationComponent `json:"component,omitempty"` // For composite observations like BP
}

// ObservationComponent represents a component of a composite FHIR Observation.
// Used for observations like blood pressure that have multiple values (systolic/diastolic).
type ObservationComponent struct {
	Code          *CodeableConcept `json:"code,omitempty"`
	ValueQuantity *Quantity        `json:"valueQuantity,omitempty"`
}

// Procedure represents a FHIR Procedure resource.
type Procedure struct {
	ResourceType  string           `json:"resourceType"`
	ID            string           `json:"id,omitempty"`
	Status        string           `json:"status,omitempty"`
	Category      *CodeableConcept `json:"category,omitempty"`
	Code          *CodeableConcept `json:"code,omitempty"`
	Subject       *Reference       `json:"subject,omitempty"`
	PerformedDateTime string       `json:"performedDateTime,omitempty"`
	PerformedPeriod   *Period      `json:"performedPeriod,omitempty"`
}

// MedicationRequest represents a FHIR MedicationRequest resource.
type MedicationRequest struct {
	ResourceType           string           `json:"resourceType"`
	ID                     string           `json:"id,omitempty"`
	Status                 string           `json:"status,omitempty"`
	Intent                 string           `json:"intent,omitempty"`
	MedicationCodeableConcept *CodeableConcept `json:"medicationCodeableConcept,omitempty"`
	Subject                *Reference       `json:"subject,omitempty"`
	AuthoredOn             string           `json:"authoredOn,omitempty"`
	DosageInstruction      []Dosage         `json:"dosageInstruction,omitempty"`
}

// Immunization represents a FHIR Immunization resource.
type Immunization struct {
	ResourceType  string           `json:"resourceType"`
	ID            string           `json:"id,omitempty"`
	Status        string           `json:"status,omitempty"`
	VaccineCode   *CodeableConcept `json:"vaccineCode,omitempty"`
	Patient       *Reference       `json:"patient,omitempty"`
	OccurrenceDateTime string      `json:"occurrenceDateTime,omitempty"`
}

// Encounter represents a FHIR Encounter resource.
type Encounter struct {
	ResourceType string           `json:"resourceType"`
	ID           string           `json:"id,omitempty"`
	Status       string           `json:"status,omitempty"`
	Class        *Coding          `json:"class,omitempty"`
	Type         []CodeableConcept `json:"type,omitempty"`
	Subject      *Reference       `json:"subject,omitempty"`
	Period       *Period          `json:"period,omitempty"`
}

// ============================================================================
// Common FHIR Data Types
// ============================================================================

// Identifier represents a FHIR Identifier.
type Identifier struct {
	Use    string     `json:"use,omitempty"`
	Type   *CodeableConcept `json:"type,omitempty"`
	System string     `json:"system,omitempty"`
	Value  string     `json:"value,omitempty"`
}

// HumanName represents a FHIR HumanName.
type HumanName struct {
	Use    string   `json:"use,omitempty"`
	Family string   `json:"family,omitempty"`
	Given  []string `json:"given,omitempty"`
	Prefix []string `json:"prefix,omitempty"`
	Suffix []string `json:"suffix,omitempty"`
}

// ContactPoint represents a FHIR ContactPoint.
type ContactPoint struct {
	System string `json:"system,omitempty"`
	Value  string `json:"value,omitempty"`
	Use    string `json:"use,omitempty"`
	Rank   int    `json:"rank,omitempty"`
}

// Address represents a FHIR Address.
type Address struct {
	Use        string   `json:"use,omitempty"`
	Type       string   `json:"type,omitempty"`
	Line       []string `json:"line,omitempty"`
	City       string   `json:"city,omitempty"`
	State      string   `json:"state,omitempty"`
	PostalCode string   `json:"postalCode,omitempty"`
	Country    string   `json:"country,omitempty"`
}

// CodeableConcept represents a FHIR CodeableConcept.
type CodeableConcept struct {
	Coding []Coding `json:"coding,omitempty"`
	Text   string   `json:"text,omitempty"`
}

// Coding represents a FHIR Coding.
type Coding struct {
	System  string `json:"system,omitempty"`
	Version string `json:"version,omitempty"`
	Code    string `json:"code,omitempty"`
	Display string `json:"display,omitempty"`
}

// Reference represents a FHIR Reference.
type Reference struct {
	Reference string `json:"reference,omitempty"`
	Type      string `json:"type,omitempty"`
	Display   string `json:"display,omitempty"`
}

// Period represents a FHIR Period.
type Period struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

// Quantity represents a FHIR Quantity.
type Quantity struct {
	Value  float64 `json:"value,omitempty"`
	Unit   string  `json:"unit,omitempty"`
	System string  `json:"system,omitempty"`
	Code   string  `json:"code,omitempty"`
}

// ReferenceRange represents a reference range for an Observation.
type ReferenceRange struct {
	Low  *Quantity `json:"low,omitempty"`
	High *Quantity `json:"high,omitempty"`
	Type *CodeableConcept `json:"type,omitempty"`
	Text string    `json:"text,omitempty"`
}

// Dosage represents a FHIR Dosage.
type Dosage struct {
	Sequence int    `json:"sequence,omitempty"`
	Text     string `json:"text,omitempty"`
	Timing   *Timing `json:"timing,omitempty"`
	DoseAndRate []DoseAndRate `json:"doseAndRate,omitempty"`
}

// Timing represents a FHIR Timing.
type Timing struct {
	Repeat *TimingRepeat `json:"repeat,omitempty"`
	Code   *CodeableConcept `json:"code,omitempty"`
}

// TimingRepeat represents a FHIR Timing.repeat.
type TimingRepeat struct {
	Frequency    int     `json:"frequency,omitempty"`
	Period       float64 `json:"period,omitempty"`
	PeriodUnit   string  `json:"periodUnit,omitempty"`
}

// DoseAndRate represents a FHIR Dosage.doseAndRate.
type DoseAndRate struct {
	Type        *CodeableConcept `json:"type,omitempty"`
	DoseQuantity *Quantity       `json:"doseQuantity,omitempty"`
}

// ============================================================================
// Patient Data Aggregation
// ============================================================================

// PatientData contains all relevant clinical data for a patient.
type PatientData struct {
	Patient            *Patient
	Conditions         []Condition
	Observations       []Observation
	Procedures         []Procedure
	MedicationRequests []MedicationRequest
	Immunizations      []Immunization
	Encounters         []Encounter

	// Metadata
	FetchedAt          time.Time
	MeasurementPeriod  *Period
}

// HasConditionCode checks if patient has a condition with the given code.
func (pd *PatientData) HasConditionCode(system, code string) bool {
	for _, cond := range pd.Conditions {
		if cond.Code != nil {
			for _, coding := range cond.Code.Coding {
				if coding.System == system && coding.Code == code {
					return true
				}
			}
		}
	}
	return false
}

// GetObservationsByCode returns observations matching the given LOINC code.
func (pd *PatientData) GetObservationsByCode(code string) []Observation {
	var results []Observation
	for _, obs := range pd.Observations {
		if obs.Code != nil {
			for _, coding := range obs.Code.Coding {
				if coding.Code == code {
					results = append(results, obs)
					break
				}
			}
		}
	}
	return results
}

// GetMostRecentObservation returns the most recent observation for a code.
func (pd *PatientData) GetMostRecentObservation(code string) *Observation {
	obs := pd.GetObservationsByCode(code)
	if len(obs) == 0 {
		return nil
	}

	// Sort by effective date (most recent first)
	var mostRecent *Observation
	var mostRecentTime time.Time

	for i := range obs {
		if obs[i].EffectiveDateTime != "" {
			t, err := time.Parse(time.RFC3339, obs[i].EffectiveDateTime)
			if err == nil && (mostRecent == nil || t.After(mostRecentTime)) {
				mostRecent = &obs[i]
				mostRecentTime = t
			}
		}
	}

	return mostRecent
}

// HasProcedureCode checks if patient has a procedure with the given code.
func (pd *PatientData) HasProcedureCode(system, code string) bool {
	for _, proc := range pd.Procedures {
		if proc.Code != nil {
			for _, coding := range proc.Code.Coding {
				if coding.System == system && coding.Code == code {
					return true
				}
			}
		}
	}
	return false
}

// GetAge calculates patient age as of the reference date.
func (pd *PatientData) GetAge(referenceDate time.Time) int {
	if pd.Patient == nil || pd.Patient.BirthDate == "" {
		return 0
	}

	birthDate, err := time.Parse("2006-01-02", pd.Patient.BirthDate)
	if err != nil {
		return 0
	}

	age := referenceDate.Year() - birthDate.Year()
	if referenceDate.YearDay() < birthDate.YearDay() {
		age--
	}

	return age
}
