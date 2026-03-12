// Package fhir provides FHIR R4 mapping for lab results
package fhir

import (
	"fmt"
	"time"

	"kb-16-lab-interpretation/pkg/types"
)

// Observation represents a FHIR R4 Observation resource
type Observation struct {
	ResourceType     string              `json:"resourceType"`
	ID               string              `json:"id"`
	Meta             *Meta               `json:"meta,omitempty"`
	Status           string              `json:"status"`
	Category         []CodeableConcept   `json:"category"`
	Code             CodeableConcept     `json:"code"`
	Subject          Reference           `json:"subject"`
	EffectiveDateTime string             `json:"effectiveDateTime,omitempty"`
	Issued           string              `json:"issued,omitempty"`
	Performer        []Reference         `json:"performer,omitempty"`
	ValueQuantity    *Quantity           `json:"valueQuantity,omitempty"`
	ValueString      string              `json:"valueString,omitempty"`
	Interpretation   []CodeableConcept   `json:"interpretation,omitempty"`
	ReferenceRange   []ObservationRefRange `json:"referenceRange,omitempty"`
	Note             []Annotation        `json:"note,omitempty"`
}

// DiagnosticReport represents a FHIR R4 DiagnosticReport resource
type DiagnosticReport struct {
	ResourceType     string            `json:"resourceType"`
	ID               string            `json:"id"`
	Meta             *Meta             `json:"meta,omitempty"`
	Status           string            `json:"status"`
	Category         []CodeableConcept `json:"category"`
	Code             CodeableConcept   `json:"code"`
	Subject          Reference         `json:"subject"`
	EffectiveDateTime string           `json:"effectiveDateTime,omitempty"`
	Issued           string            `json:"issued,omitempty"`
	Result           []Reference       `json:"result,omitempty"`
	Conclusion       string            `json:"conclusion,omitempty"`
	ConclusionCode   []CodeableConcept `json:"conclusionCode,omitempty"`
}

// Meta represents FHIR resource metadata
type Meta struct {
	Profile     []string `json:"profile,omitempty"`
	LastUpdated string   `json:"lastUpdated,omitempty"`
}

// CodeableConcept represents a FHIR CodeableConcept
type CodeableConcept struct {
	Coding []Coding `json:"coding"`
	Text   string   `json:"text,omitempty"`
}

// Coding represents a FHIR Coding
type Coding struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display,omitempty"`
}

// Reference represents a FHIR Reference
type Reference struct {
	Reference string `json:"reference"`
	Display   string `json:"display,omitempty"`
}

// Quantity represents a FHIR Quantity
type Quantity struct {
	Value  *float64 `json:"value,omitempty"`
	Unit   string   `json:"unit,omitempty"`
	System string   `json:"system,omitempty"`
	Code   string   `json:"code,omitempty"`
}

// ObservationRefRange represents a FHIR Observation reference range
type ObservationRefRange struct {
	Low  *Quantity `json:"low,omitempty"`
	High *Quantity `json:"high,omitempty"`
	Text string    `json:"text,omitempty"`
}

// Annotation represents a FHIR Annotation
type Annotation struct {
	Text string `json:"text"`
	Time string `json:"time,omitempty"`
}

// Bundle represents a FHIR Bundle
type Bundle struct {
	ResourceType string        `json:"resourceType"`
	Type         string        `json:"type"`
	Total        int           `json:"total"`
	Entry        []BundleEntry `json:"entry,omitempty"`
}

// BundleEntry represents a FHIR Bundle entry
type BundleEntry struct {
	FullURL  string      `json:"fullUrl,omitempty"`
	Resource interface{} `json:"resource"`
}

// Mapper handles FHIR resource mapping
type Mapper struct{}

// NewMapper creates a new FHIR mapper
func NewMapper() *Mapper {
	return &Mapper{}
}

// ToObservation converts a lab result to a FHIR Observation
func (m *Mapper) ToObservation(result *types.LabResult, interpretation *types.Interpretation) *Observation {
	obs := &Observation{
		ResourceType: "Observation",
		ID:           result.ID.String(),
		Meta: &Meta{
			Profile: []string{"http://hl7.org/fhir/us/core/StructureDefinition/us-core-observation-lab"},
			LastUpdated: time.Now().Format(time.RFC3339),
		},
		Status: mapStatus(string(result.Status)),
		Category: []CodeableConcept{
			{
				Coding: []Coding{
					{
						System:  "http://terminology.hl7.org/CodeSystem/observation-category",
						Code:    "laboratory",
						Display: "Laboratory",
					},
				},
			},
		},
		Code: CodeableConcept{
			Coding: []Coding{
				{
					System:  "http://loinc.org",
					Code:    result.Code,
					Display: result.Name,
				},
			},
			Text: result.Name,
		},
		Subject: Reference{
			Reference: fmt.Sprintf("Patient/%s", result.PatientID),
		},
		EffectiveDateTime: result.CollectedAt.Format(time.RFC3339),
		Issued:           result.ReportedAt.Format(time.RFC3339),
	}

	// Add performer if present
	if result.Performer != "" {
		obs.Performer = []Reference{
			{Reference: result.Performer},
		}
	}

	// Add value
	if result.ValueNumeric != nil {
		obs.ValueQuantity = &Quantity{
			Value:  result.ValueNumeric,
			Unit:   result.Unit,
			System: "http://unitsofmeasure.org",
			Code:   result.Unit,
		}
	} else if result.ValueString != "" {
		obs.ValueString = result.ValueString
	}

	// Add interpretation if provided
	if interpretation != nil {
		obs.Interpretation = []CodeableConcept{
			{
				Coding: []Coding{
					{
						System:  "http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation",
						Code:    mapInterpretationToFHIR(interpretation.Flag),
						Display: string(interpretation.Flag),
					},
				},
			},
		}

		// Add clinical comment as note
		if interpretation.ClinicalComment != "" {
			obs.Note = []Annotation{
				{
					Text: interpretation.ClinicalComment,
					Time: time.Now().Format(time.RFC3339),
				},
			}
		}
	}

	// Add reference range
	if result.ReferenceRange.Low != nil || result.ReferenceRange.High != nil {
		refRange := ObservationRefRange{}
		if result.ReferenceRange.Low != nil {
			refRange.Low = &Quantity{
				Value: result.ReferenceRange.Low,
				Unit:  result.Unit,
			}
		}
		if result.ReferenceRange.High != nil {
			refRange.High = &Quantity{
				Value: result.ReferenceRange.High,
				Unit:  result.Unit,
			}
		}
		obs.ReferenceRange = []ObservationRefRange{refRange}
	}

	return obs
}

// ToDiagnosticReport converts a panel to a FHIR DiagnosticReport
func (m *Mapper) ToDiagnosticReport(panel *types.AssembledPanel) *DiagnosticReport {
	report := &DiagnosticReport{
		ResourceType: "DiagnosticReport",
		ID:           fmt.Sprintf("%s-%s-%d", panel.PatientID, panel.Type, panel.AssembledAt.Unix()),
		Meta: &Meta{
			Profile: []string{"http://hl7.org/fhir/us/core/StructureDefinition/us-core-diagnosticreport-lab"},
			LastUpdated: time.Now().Format(time.RFC3339),
		},
		Status: "final",
		Category: []CodeableConcept{
			{
				Coding: []Coding{
					{
						System:  "http://terminology.hl7.org/CodeSystem/v2-0074",
						Code:    "LAB",
						Display: "Laboratory",
					},
				},
			},
		},
		Code: CodeableConcept{
			Coding: []Coding{
				{
					System:  "http://loinc.org",
					Code:    getPanelLOINC(panel.Type),
					Display: panel.Name,
				},
			},
			Text: panel.Name,
		},
		Subject: Reference{
			Reference: fmt.Sprintf("Patient/%s", panel.PatientID),
		},
		EffectiveDateTime: panel.AssembledAt.Format(time.RFC3339),
		Issued:           panel.AssembledAt.Format(time.RFC3339),
	}

	// Add result references
	results := make([]Reference, 0)
	for _, component := range panel.Components {
		if component.Available && component.Result != nil {
			results = append(results, Reference{
				Reference: fmt.Sprintf("Observation/%s", component.Result.ID.String()),
				Display:   component.Name,
			})
		}
	}
	report.Result = results

	// Add conclusion based on detected patterns
	if len(panel.DetectedPatterns) > 0 {
		conclusions := make([]string, 0)
		conclusionCodes := make([]CodeableConcept, 0)

		for _, pattern := range panel.DetectedPatterns {
			conclusions = append(conclusions, pattern.Description)
			conclusionCodes = append(conclusionCodes, CodeableConcept{
				Coding: []Coding{
					{
						System:  "http://snomed.info/sct",
						Code:    pattern.Code,
						Display: pattern.Name,
					},
				},
			})
		}

		report.Conclusion = fmt.Sprintf("Detected patterns: %v", conclusions)
		report.ConclusionCode = conclusionCodes
	}

	return report
}

// ToBundle creates a FHIR Bundle from multiple observations
func (m *Mapper) ToBundle(observations []*Observation, bundleType string) *Bundle {
	entries := make([]BundleEntry, len(observations))
	for i, obs := range observations {
		entries[i] = BundleEntry{
			FullURL:  fmt.Sprintf("urn:uuid:%s", obs.ID),
			Resource: obs,
		}
	}

	return &Bundle{
		ResourceType: "Bundle",
		Type:         bundleType,
		Total:        len(observations),
		Entry:        entries,
	}
}

// SearchObservations searches observations by criteria
func (m *Mapper) SearchObservations(results []types.LabResult, interpretations map[string]*types.Interpretation) *Bundle {
	observations := make([]*Observation, len(results))
	for i, result := range results {
		var interp *types.Interpretation
		if interpretations != nil {
			interp = interpretations[result.ID.String()]
		}
		observations[i] = m.ToObservation(&result, interp)
	}

	return m.ToBundle(observations, "searchset")
}

// mapStatus maps internal status to FHIR observation status
func mapStatus(status string) string {
	switch status {
	case "final":
		return "final"
	case "preliminary":
		return "preliminary"
	case "corrected":
		return "corrected"
	case "cancelled":
		return "cancelled"
	default:
		return "final"
	}
}

// mapInterpretationToFHIR maps interpretation flag to FHIR code
func mapInterpretationToFHIR(flag types.InterpretationFlag) string {
	switch flag {
	case types.FlagNormal:
		return "N"
	case types.FlagLow:
		return "L"
	case types.FlagHigh:
		return "H"
	case types.FlagCriticalLow:
		return "LL"
	case types.FlagCriticalHigh:
		return "HH"
	case types.FlagPanicLow:
		return "LL"
	case types.FlagPanicHigh:
		return "HH"
	default:
		return "N"
	}
}

// getPanelLOINC returns the LOINC code for a panel type
func getPanelLOINC(panelType types.PanelType) string {
	codes := map[types.PanelType]string{
		types.PanelBMP:   "51990-0",
		types.PanelCMP:   "24323-8",
		types.PanelCBC:   "58410-2",
		types.PanelLFT:   "24325-3",
		types.PanelLipid: "24331-1",
		types.PanelRenal: "24362-6",
	}
	if code, exists := codes[panelType]; exists {
		return code
	}
	return "unknown"
}
