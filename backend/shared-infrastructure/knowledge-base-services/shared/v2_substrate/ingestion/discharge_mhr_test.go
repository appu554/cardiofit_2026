package ingestion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
)

func TestParseMHRDischargeCDA_SyntheticFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "synthetic_cda_discharge.xml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	doc, err := ParseMHRDischargeCDA(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if doc.Source != DischargeSourceMHRCDA {
		t.Errorf("source mismatch: %s", doc.Source)
	}
	if doc.DocumentID != "MHR-DISCH-2025-001" {
		t.Errorf("document_id mismatch: %s", doc.DocumentID)
	}
	if doc.DischargeDate.IsZero() {
		t.Errorf("discharge_date not parsed")
	}
	if doc.DischargingFacilityName != "St Vincent's Hospital Sydney" {
		t.Errorf("custodian name mismatch: %q", doc.DischargingFacilityName)
	}
	if got, ok := doc.StructuredPayload["patient_ihi"]; !ok || got != "8003600000000099" {
		t.Errorf("patient_ihi not surfaced via structured_payload: %v", doc.StructuredPayload)
	}
	if len(doc.MedicationLines) != 3 {
		t.Fatalf("expected 3 medication lines, got %d", len(doc.MedicationLines))
	}
	met := doc.MedicationLines[0]
	if met.AMTCode != "61428011000036109" {
		t.Errorf("metformin AMT code mismatch: %q", met.AMTCode)
	}
	if met.DoseRaw != "500 mg" {
		t.Errorf("metformin dose mismatch: %q", met.DoseRaw)
	}
	if met.FrequencyRaw != "every 12h" {
		t.Errorf("metformin frequency mismatch: %q", met.FrequencyRaw)
	}
	if met.RouteRaw != "oral" {
		t.Errorf("metformin route mismatch: %q", met.RouteRaw)
	}
	if met.IndicationText != "type 2 diabetes mellitus" {
		t.Errorf("metformin indication mismatch: %q", met.IndicationText)
	}
	// amoxicillin line should carry the post-op note for the classifier.
	amox := doc.MedicationLines[1]
	if amox.IndicationText != "post-op infection" {
		t.Errorf("amox indication mismatch: %q", amox.IndicationText)
	}
	if amox.Notes == "" {
		t.Errorf("amox notes should carry substanceAdministration text")
	}
}

func TestParseMHRDischargeCDA_RejectsMalformedXML(t *testing.T) {
	_, err := ParseMHRDischargeCDA([]byte("<not valid xml"))
	if err == nil {
		t.Fatalf("expected error for malformed XML")
	}
}

func TestAssignResident(t *testing.T) {
	doc := &ParsedDischargeDocument{}
	rid := uuid.New()
	got := AssignResident(doc, rid)
	if got.ResidentRef != rid {
		t.Fatalf("AssignResident did not set ref")
	}
	if AssignResident(nil, rid) != nil {
		t.Fatalf("AssignResident(nil) must return nil")
	}
}
