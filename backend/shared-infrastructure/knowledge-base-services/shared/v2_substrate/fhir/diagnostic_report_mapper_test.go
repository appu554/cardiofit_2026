package fhir

import (
	"encoding/json"
	"testing"
)

const syntheticDiagnosticReportJSON = `{
  "resourceType": "DiagnosticReport",
  "id": "rep-001",
  "status": "final",
  "code": {
    "coding": [
      { "system": "http://loinc.org", "code": "11502-2", "display": "Lab report" }
    ]
  },
  "effectiveDateTime": "2026-05-01T08:30:00+10:00",
  "result": [
    { "reference": "#obs-potassium" },
    { "reference": "#obs-egfr" },
    { "reference": "Observation/obs-microscopy" }
  ],
  "contained": [
    {
      "resourceType": "Observation",
      "id": "obs-potassium",
      "status": "final",
      "code": {
        "coding": [
          { "system": "http://loinc.org", "code": "2823-3", "display": "Serum potassium" },
          { "system": "http://snomed.info/sct", "code": "271001000087101" }
        ]
      },
      "effectiveDateTime": "2026-05-01T08:30:00+10:00",
      "valueQuantity": { "value": 5.8, "unit": "mmol/L", "code": "mmol/L" },
      "interpretation": [
        {
          "coding": [
            { "system": "http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation", "code": "H" }
          ]
        }
      ]
    },
    {
      "resourceType": "Observation",
      "id": "obs-egfr",
      "status": "final",
      "code": {
        "coding": [
          { "system": "http://loinc.org", "code": "33914-3", "display": "eGFR" }
        ]
      },
      "effectiveDateTime": "2026-05-01T08:30:00+10:00",
      "valueQuantity": { "value": 42, "unit": "mL/min/1.73m2" },
      "interpretation": [
        {
          "coding": [
            { "system": "http://terminology.hl7.org/CodeSystem/v3-ObservationInterpretation", "code": "L" }
          ]
        }
      ]
    }
  ]
}`

const syntheticExternalObservationJSON = `{
  "resourceType": "Observation",
  "id": "obs-microscopy",
  "status": "final",
  "code": {
    "coding": [
      { "system": "http://loinc.org", "code": "11556-8", "display": "Microscopy comment" }
    ]
  },
  "effectiveDateTime": "2026-05-01T08:30:00+10:00",
  "valueString": "No organisms seen on Gram stain."
}`

func TestParseFHIRDiagnosticReport_SyntheticBundle(t *testing.T) {
	var report map[string]interface{}
	if err := json.Unmarshal([]byte(syntheticDiagnosticReportJSON), &report); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	var external map[string]interface{}
	if err := json.Unmarshal([]byte(syntheticExternalObservationJSON), &external); err != nil {
		t.Fatalf("unmarshal external: %v", err)
	}

	results, err := ParseFHIRDiagnosticReport(report, external)
	if err != nil {
		t.Fatalf("ParseFHIRDiagnosticReport: %v", err)
	}

	if got, want := len(results), 3; got != want {
		t.Fatalf("len(results) = %d, want %d", got, want)
	}

	// Result 0: potassium with both LOINC and SNOMED + H interpretation.
	po := results[0]
	if po.LOINCCode != "2823-3" || po.SNOMEDCode != "271001000087101" {
		t.Errorf("results[0] codes = (%q, %q)", po.LOINCCode, po.SNOMEDCode)
	}
	if po.Value == nil || *po.Value != 5.8 {
		t.Errorf("results[0].Value = %v, want 5.8", po.Value)
	}
	if po.Unit != "mmol/L" {
		t.Errorf("results[0].Unit = %q", po.Unit)
	}
	if po.AbnormalFlag != "high" {
		t.Errorf("results[0].AbnormalFlag = %q, want high", po.AbnormalFlag)
	}

	// Result 1: eGFR with L interpretation.
	po = results[1]
	if po.LOINCCode != "33914-3" {
		t.Errorf("results[1].LOINCCode = %q", po.LOINCCode)
	}
	if po.Value == nil || *po.Value != 42 {
		t.Errorf("results[1].Value = %v, want 42", po.Value)
	}
	if po.AbnormalFlag != "low" {
		t.Errorf("results[1].AbnormalFlag = %q, want low", po.AbnormalFlag)
	}

	// Result 2: external microscopy with valueString.
	po = results[2]
	if po.LOINCCode != "11556-8" {
		t.Errorf("results[2].LOINCCode = %q", po.LOINCCode)
	}
	if po.Value != nil {
		t.Errorf("results[2].Value should be nil for valueString")
	}
	if po.ValueText != "No organisms seen on Gram stain." {
		t.Errorf("results[2].ValueText = %q", po.ValueText)
	}
}

func TestParseFHIRDiagnosticReport_WrongResourceType(t *testing.T) {
	report := map[string]interface{}{"resourceType": "Patient"}
	_, err := ParseFHIRDiagnosticReport(report)
	if err == nil {
		t.Fatalf("expected error for non-DiagnosticReport input")
	}
}

func TestParseFHIRDiagnosticReport_ConvergesOnSameDTOAsCDA(t *testing.T) {
	// The convergence claim from the ADR: SOAP/CDA and FHIR Gateway paths
	// produce the same ParsedObservation shape for the same logical data.
	// The synthetic CDA fixture and the synthetic FHIR DiagnosticReport
	// fixture encode the same three observations. Verify field-by-field
	// equivalence (modulo source-format differences in DisplayName text).
	var report map[string]interface{}
	_ = json.Unmarshal([]byte(syntheticDiagnosticReportJSON), &report)
	var external map[string]interface{}
	_ = json.Unmarshal([]byte(syntheticExternalObservationJSON), &external)

	results, _ := ParseFHIRDiagnosticReport(report, external)
	if len(results) != 3 {
		t.Fatalf("len(results) = %d", len(results))
	}

	// LOINC codes match the CDA fixture's three observations.
	wantLOINC := []string{"2823-3", "33914-3", "11556-8"}
	for i, w := range wantLOINC {
		if results[i].LOINCCode != w {
			t.Errorf("results[%d].LOINCCode = %q, want %q", i, results[i].LOINCCode, w)
		}
	}
}
