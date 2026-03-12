// Package deqm implements Da Vinci Data Exchange for Quality Measures (DEQM) operations.
// This follows the FHIR R4 + Da Vinci DEQM Implementation Guide for the $care-gaps operation.
package deqm

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"kb-9-care-gaps/internal/models"
)

// CareGapsParameters represents the input parameters for the $care-gaps operation.
// See: http://hl7.org/fhir/us/davinci-deqm/OperationDefinition-care-gaps.html
type CareGapsParameters struct {
	// PeriodStart is the start of the measurement period (required)
	PeriodStart time.Time `json:"periodStart"`

	// PeriodEnd is the end of the measurement period (required)
	PeriodEnd time.Time `json:"periodEnd"`

	// Subject is the patient reference (required) - format: "Patient/{id}"
	Subject string `json:"subject"`

	// Status filters gaps by status: open-gap, closed-gap, or not-applicable
	Status []string `json:"status,omitempty"`

	// Measure filters to specific measures (canonical URL or ID)
	Measure []string `json:"measure,omitempty"`

	// Organization is the reporter organization reference
	Organization string `json:"organization,omitempty"`

	// Practitioner is the practitioner reference
	Practitioner string `json:"practitioner,omitempty"`
}

// CareGapsBundle represents the FHIR Bundle response for $care-gaps operation.
type CareGapsBundle struct {
	ResourceType string            `json:"resourceType"`
	ID           string            `json:"id"`
	Type         string            `json:"type"`
	Timestamp    string            `json:"timestamp"`
	Total        int               `json:"total,omitempty"`
	Entry        []BundleEntry     `json:"entry"`
	Meta         *ResourceMeta     `json:"meta,omitempty"`
}

// BundleEntry represents an entry in the FHIR Bundle.
type BundleEntry struct {
	FullURL  string      `json:"fullUrl,omitempty"`
	Resource interface{} `json:"resource"`
}

// ResourceMeta represents FHIR resource metadata.
type ResourceMeta struct {
	Profile     []string `json:"profile,omitempty"`
	LastUpdated string   `json:"lastUpdated,omitempty"`
}

// FHIRReference represents a FHIR reference.
type FHIRReference struct {
	Reference string `json:"reference"`
	Display   string `json:"display,omitempty"`
}

// FHIRPeriod represents a FHIR Period.
type FHIRPeriod struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// FHIRCoding represents a FHIR Coding.
type FHIRCoding struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display,omitempty"`
}

// FHIRCodeableConcept represents a FHIR CodeableConcept.
type FHIRCodeableConcept struct {
	Coding []FHIRCoding `json:"coding"`
	Text   string       `json:"text,omitempty"`
}

// MeasureReportResource represents a FHIR MeasureReport resource.
type MeasureReportResource struct {
	ResourceType       string                  `json:"resourceType"`
	ID                 string                  `json:"id"`
	Meta               *ResourceMeta           `json:"meta,omitempty"`
	Status             string                  `json:"status"`
	Type               string                  `json:"type"`
	Measure            string                  `json:"measure"`
	Subject            *FHIRReference          `json:"subject"`
	Date               string                  `json:"date"`
	Reporter           *FHIRReference          `json:"reporter,omitempty"`
	Period             FHIRPeriod              `json:"period"`
	Group              []MeasureReportGroup    `json:"group,omitempty"`
	EvaluatedResource  []FHIRReference         `json:"evaluatedResource,omitempty"`
	ImprovementNotation *FHIRCodeableConcept   `json:"improvementNotation,omitempty"`
}

// MeasureReportGroup represents a group in a MeasureReport.
type MeasureReportGroup struct {
	Code        *FHIRCodeableConcept            `json:"code,omitempty"`
	Population  []MeasureReportPopulation       `json:"population,omitempty"`
	MeasureScore *MeasureReportMeasureScore     `json:"measureScore,omitempty"`
	Stratifier  []MeasureReportStratifier       `json:"stratifier,omitempty"`
}

// MeasureReportPopulation represents population results.
type MeasureReportPopulation struct {
	Code  *FHIRCodeableConcept `json:"code"`
	Count int                  `json:"count"`
}

// MeasureReportMeasureScore represents measure score.
type MeasureReportMeasureScore struct {
	Value float64 `json:"value,omitempty"`
}

// MeasureReportStratifier represents stratifier results.
type MeasureReportStratifier struct {
	Code    []FHIRCodeableConcept          `json:"code,omitempty"`
	Stratum []MeasureReportStratum         `json:"stratum,omitempty"`
}

// MeasureReportStratum represents stratum results.
type MeasureReportStratum struct {
	Value      *FHIRCodeableConcept       `json:"value,omitempty"`
	Population []MeasureReportPopulation  `json:"population,omitempty"`
}

// DetectedIssueResource represents a FHIR DetectedIssue resource for care gaps.
type DetectedIssueResource struct {
	ResourceType  string               `json:"resourceType"`
	ID            string               `json:"id"`
	Meta          *ResourceMeta        `json:"meta,omitempty"`
	Status        string               `json:"status"`
	Code          *FHIRCodeableConcept `json:"code"`
	Severity      string               `json:"severity,omitempty"`
	Patient       *FHIRReference       `json:"patient"`
	IdentifiedDateTime string          `json:"identifiedDateTime"`
	Evidence      []DetectedIssueEvidence `json:"evidence,omitempty"`
	Detail        string               `json:"detail,omitempty"`
	Implicated    []FHIRReference      `json:"implicated,omitempty"`
}

// DetectedIssueEvidence represents evidence for a detected issue.
type DetectedIssueEvidence struct {
	Code   []FHIRCodeableConcept `json:"code,omitempty"`
	Detail []FHIRReference       `json:"detail,omitempty"`
}

// CompositionResource represents a FHIR Composition resource for the care gaps document.
type CompositionResource struct {
	ResourceType string                `json:"resourceType"`
	ID           string                `json:"id"`
	Meta         *ResourceMeta         `json:"meta,omitempty"`
	Status       string                `json:"status"`
	Type         *FHIRCodeableConcept  `json:"type"`
	Subject      *FHIRReference        `json:"subject"`
	Date         string                `json:"date"`
	Author       []FHIRReference       `json:"author,omitempty"`
	Title        string                `json:"title"`
	Section      []CompositionSection  `json:"section,omitempty"`
}

// CompositionSection represents a section in a Composition.
type CompositionSection struct {
	Title string          `json:"title,omitempty"`
	Code  *FHIRCodeableConcept `json:"code,omitempty"`
	Text  *Narrative      `json:"text,omitempty"`
	Entry []FHIRReference `json:"entry,omitempty"`
}

// Narrative represents FHIR narrative text.
type Narrative struct {
	Status string `json:"status"`
	Div    string `json:"div"`
}

// DEQM profile URLs
const (
	DEQMCareGapsMeasureReportProfile = "http://hl7.org/fhir/us/davinci-deqm/StructureDefinition/gaps-in-care-measure-report"
	DEQMDetectedIssueProfile         = "http://hl7.org/fhir/us/davinci-deqm/StructureDefinition/gaps-in-care-detected-issue"
	DEQMCompositionProfile           = "http://hl7.org/fhir/us/davinci-deqm/StructureDefinition/gaps-in-care-composition"
)

// Gap status codes from DEQM
const (
	GapStatusOpenGap       = "open-gap"
	GapStatusClosedGap     = "closed-gap"
	GapStatusNotApplicable = "not-applicable"
)

// CareGapsConverter converts KB-9 care gaps to FHIR DEQM format.
type CareGapsConverter struct{}

// NewCareGapsConverter creates a new converter.
func NewCareGapsConverter() *CareGapsConverter {
	return &CareGapsConverter{}
}

// ConvertToBundle converts a KB-9 CareGapReport to a FHIR Bundle.
func (c *CareGapsConverter) ConvertToBundle(report *models.CareGapReport, params *CareGapsParameters) *CareGapsBundle {
	now := time.Now().UTC()
	bundleID := uuid.New().String()

	// Create bundle entries
	var entries []BundleEntry

	// 1. Add Composition resource (document structure)
	composition := c.createComposition(report, params, now)
	entries = append(entries, BundleEntry{
		FullURL:  fmt.Sprintf("urn:uuid:%s", composition.ID),
		Resource: composition,
	})

	// 2. Add MeasureReport for each measure with gaps
	measureReportRefs := make(map[string]string) // measureID -> fullURL
	for _, gap := range report.OpenGaps {
		measureReport := c.createMeasureReport(&gap, report.PatientID, params, now, false)
		measureReportRefs[gap.Measure.CMSID] = fmt.Sprintf("urn:uuid:%s", measureReport.ID)
		entries = append(entries, BundleEntry{
			FullURL:  fmt.Sprintf("urn:uuid:%s", measureReport.ID),
			Resource: measureReport,
		})
	}

	// Add closed gap MeasureReports
	for _, gap := range report.ClosedGaps {
		measureReport := c.createMeasureReport(&gap, report.PatientID, params, now, true)
		entries = append(entries, BundleEntry{
			FullURL:  fmt.Sprintf("urn:uuid:%s", measureReport.ID),
			Resource: measureReport,
		})
	}

	// 3. Add DetectedIssue for each open gap
	for _, gap := range report.OpenGaps {
		detectedIssue := c.createDetectedIssue(&gap, report.PatientID, now)
		entries = append(entries, BundleEntry{
			FullURL:  fmt.Sprintf("urn:uuid:%s", detectedIssue.ID),
			Resource: detectedIssue,
		})
	}

	return &CareGapsBundle{
		ResourceType: "Bundle",
		ID:           bundleID,
		Type:         "document",
		Timestamp:    now.Format(time.RFC3339),
		Total:        len(entries),
		Entry:        entries,
		Meta: &ResourceMeta{
			Profile: []string{"http://hl7.org/fhir/us/davinci-deqm/StructureDefinition/gaps-in-care-bundle"},
		},
	}
}

// createComposition creates the Composition resource for the care gaps document.
func (c *CareGapsConverter) createComposition(report *models.CareGapReport, params *CareGapsParameters, now time.Time) *CompositionResource {
	sections := []CompositionSection{}

	// Open gaps section
	if len(report.OpenGaps) > 0 {
		openGapRefs := make([]FHIRReference, 0, len(report.OpenGaps))
		for _, gap := range report.OpenGaps {
			openGapRefs = append(openGapRefs, FHIRReference{
				Reference: fmt.Sprintf("DetectedIssue/%s", gap.ID),
			})
		}
		sections = append(sections, CompositionSection{
			Title: "Open Gaps",
			Code: &FHIRCodeableConcept{
				Coding: []FHIRCoding{{
					System:  "http://hl7.org/fhir/us/davinci-deqm/CodeSystem/gaps-status",
					Code:    GapStatusOpenGap,
					Display: "Open Gap",
				}},
			},
			Entry: openGapRefs,
		})
	}

	// Closed gaps section
	if len(report.ClosedGaps) > 0 {
		closedGapRefs := make([]FHIRReference, 0, len(report.ClosedGaps))
		for _, gap := range report.ClosedGaps {
			closedGapRefs = append(closedGapRefs, FHIRReference{
				Reference: fmt.Sprintf("MeasureReport/%s", gap.ID),
			})
		}
		sections = append(sections, CompositionSection{
			Title: "Closed Gaps",
			Code: &FHIRCodeableConcept{
				Coding: []FHIRCoding{{
					System:  "http://hl7.org/fhir/us/davinci-deqm/CodeSystem/gaps-status",
					Code:    GapStatusClosedGap,
					Display: "Closed Gap",
				}},
			},
			Entry: closedGapRefs,
		})
	}

	return &CompositionResource{
		ResourceType: "Composition",
		ID:           uuid.New().String(),
		Meta: &ResourceMeta{
			Profile:     []string{DEQMCompositionProfile},
			LastUpdated: now.Format(time.RFC3339),
		},
		Status: "final",
		Type: &FHIRCodeableConcept{
			Coding: []FHIRCoding{{
				System:  "http://loinc.org",
				Code:    "96315-7",
				Display: "Gaps in care report",
			}},
		},
		Subject: &FHIRReference{
			Reference: fmt.Sprintf("Patient/%s", report.PatientID),
		},
		Date:   now.Format(time.RFC3339),
		Title:  "Care Gaps Report",
		Section: sections,
	}
}

// createMeasureReport creates a MeasureReport for a care gap.
func (c *CareGapsConverter) createMeasureReport(gap *models.CareGap, patientID string, params *CareGapsParameters, now time.Time, isClosed bool) *MeasureReportResource {
	status := "complete"
	measureURL := fmt.Sprintf("http://ecqi.healthit.gov/ecqms/Measure/%s", gap.Measure.CMSID)

	// Determine population counts
	numeratorCount := 0
	denominatorCount := 1
	if isClosed {
		numeratorCount = 1
	}

	populations := []MeasureReportPopulation{
		{
			Code: &FHIRCodeableConcept{
				Coding: []FHIRCoding{{
					System:  "http://terminology.hl7.org/CodeSystem/measure-population",
					Code:    "initial-population",
					Display: "Initial Population",
				}},
			},
			Count: 1,
		},
		{
			Code: &FHIRCodeableConcept{
				Coding: []FHIRCoding{{
					System:  "http://terminology.hl7.org/CodeSystem/measure-population",
					Code:    "denominator",
					Display: "Denominator",
				}},
			},
			Count: denominatorCount,
		},
		{
			Code: &FHIRCodeableConcept{
				Coding: []FHIRCoding{{
					System:  "http://terminology.hl7.org/CodeSystem/measure-population",
					Code:    "numerator",
					Display: "Numerator",
				}},
			},
			Count: numeratorCount,
		},
	}

	return &MeasureReportResource{
		ResourceType: "MeasureReport",
		ID:           uuid.New().String(),
		Meta: &ResourceMeta{
			Profile:     []string{DEQMCareGapsMeasureReportProfile},
			LastUpdated: now.Format(time.RFC3339),
		},
		Status:  status,
		Type:    "individual",
		Measure: measureURL,
		Subject: &FHIRReference{
			Reference: fmt.Sprintf("Patient/%s", patientID),
		},
		Date: now.Format(time.RFC3339),
		Period: FHIRPeriod{
			Start: params.PeriodStart.Format("2006-01-02"),
			End:   params.PeriodEnd.Format("2006-01-02"),
		},
		Group: []MeasureReportGroup{
			{
				Population: populations,
			},
		},
		ImprovementNotation: &FHIRCodeableConcept{
			Coding: []FHIRCoding{{
				System:  "http://terminology.hl7.org/CodeSystem/measure-improvement-notation",
				Code:    "increase",
				Display: "Increased score indicates improvement",
			}},
		},
	}
}

// createDetectedIssue creates a DetectedIssue for an open care gap.
func (c *CareGapsConverter) createDetectedIssue(gap *models.CareGap, patientID string, now time.Time) *DetectedIssueResource {
	severity := c.mapPriorityToSeverity(gap.Priority)

	return &DetectedIssueResource{
		ResourceType: "DetectedIssue",
		ID:           gap.ID,
		Meta: &ResourceMeta{
			Profile:     []string{DEQMDetectedIssueProfile},
			LastUpdated: now.Format(time.RFC3339),
		},
		Status: "final",
		Code: &FHIRCodeableConcept{
			Coding: []FHIRCoding{
				{
					System:  "http://hl7.org/fhir/us/davinci-deqm/CodeSystem/gaps-status",
					Code:    GapStatusOpenGap,
					Display: "Open Gap",
				},
				{
					System:  "http://ecqi.healthit.gov/ecqms/Measure",
					Code:    gap.Measure.CMSID,
					Display: gap.Measure.Name,
				},
			},
			Text: fmt.Sprintf("Care Gap: %s", gap.Measure.Name),
		},
		Severity: severity,
		Patient: &FHIRReference{
			Reference: fmt.Sprintf("Patient/%s", patientID),
		},
		IdentifiedDateTime: now.Format(time.RFC3339),
		Detail:             gap.Reason,
		Evidence: []DetectedIssueEvidence{
			{
				Code: []FHIRCodeableConcept{
					{
						Text: gap.Recommendation,
					},
				},
			},
		},
	}
}

// mapPriorityToSeverity maps KB-9 priority to FHIR DetectedIssue severity.
func (c *CareGapsConverter) mapPriorityToSeverity(priority models.GapPriority) string {
	switch priority {
	case models.GapPriorityCritical, models.GapPriorityUrgent:
		return "high"
	case models.GapPriorityHigh:
		return "moderate"
	case models.GapPriorityMedium:
		return "low"
	default:
		return "low"
	}
}

// FilterGapsByStatus filters gaps based on DEQM status parameter.
func FilterGapsByStatus(report *models.CareGapReport, statusFilters []string) *models.CareGapReport {
	if len(statusFilters) == 0 {
		return report // No filter, return all
	}

	// Parse which statuses to include
	includeOpen := false
	includeClosed := false
	for _, s := range statusFilters {
		switch s {
		case GapStatusOpenGap:
			includeOpen = true
		case GapStatusClosedGap:
			includeClosed = true
		}
	}

	filtered := &models.CareGapReport{
		PatientID:         report.PatientID,
		ReportDate:        report.ReportDate,
		MeasurementPeriod: report.MeasurementPeriod,
		Summary:           report.Summary,
		DataCompleteness:  report.DataCompleteness,
	}

	if includeOpen {
		filtered.OpenGaps = report.OpenGaps
	}
	if includeClosed {
		filtered.ClosedGaps = report.ClosedGaps
	}

	return filtered
}

// ParseParametersResource parses a FHIR Parameters resource into CareGapsParameters.
func ParseParametersResource(params map[string]interface{}) (*CareGapsParameters, error) {
	result := &CareGapsParameters{}

	// Extract parameter array
	paramArray, ok := params["parameter"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid Parameters resource: missing parameter array")
	}

	for _, p := range paramArray {
		param, ok := p.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := param["name"].(string)

		switch name {
		case "periodStart":
			if v, ok := param["valueDate"].(string); ok {
				if t, err := time.Parse("2006-01-02", v); err == nil {
					result.PeriodStart = t
				}
			}
		case "periodEnd":
			if v, ok := param["valueDate"].(string); ok {
				if t, err := time.Parse("2006-01-02", v); err == nil {
					result.PeriodEnd = t
				}
			}
		case "subject":
			if v, ok := param["valueString"].(string); ok {
				result.Subject = v
			}
		case "status":
			if v, ok := param["valueCode"].(string); ok {
				result.Status = append(result.Status, v)
			}
		case "measure":
			if v, ok := param["valueCanonical"].(string); ok {
				result.Measure = append(result.Measure, v)
			}
		case "organization":
			if v, ok := param["valueString"].(string); ok {
				result.Organization = v
			}
		case "practitioner":
			if v, ok := param["valueString"].(string); ok {
				result.Practitioner = v
			}
		}
	}

	// Validate required parameters
	if result.Subject == "" {
		return nil, fmt.Errorf("missing required parameter: subject")
	}
	if result.PeriodStart.IsZero() {
		return nil, fmt.Errorf("missing required parameter: periodStart")
	}
	if result.PeriodEnd.IsZero() {
		return nil, fmt.Errorf("missing required parameter: periodEnd")
	}

	return result, nil
}

// ExtractPatientID extracts the patient ID from a FHIR reference.
// Handles formats: "Patient/123", "123", "urn:uuid:123"
func ExtractPatientID(reference string) string {
	// Handle "Patient/123" format
	if len(reference) > 8 && reference[:8] == "Patient/" {
		return reference[8:]
	}
	// Handle "urn:uuid:" format
	if len(reference) > 9 && reference[:9] == "urn:uuid:" {
		return reference[9:]
	}
	// Assume it's already just the ID
	return reference
}
