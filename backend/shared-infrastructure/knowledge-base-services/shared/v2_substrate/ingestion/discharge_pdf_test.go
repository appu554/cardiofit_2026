package ingestion

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIngestDischargePDF_ValidatesRequiredFields(t *testing.T) {
	_, err := IngestDischargePDF(PDFDischargeInput{ExtractedText: "x"})
	if err == nil {
		t.Fatalf("missing resident_ref must error")
	}
	_, err = IngestDischargePDF(PDFDischargeInput{ResidentRef: uuid.New(), ExtractedText: "x"})
	if err == nil {
		t.Fatalf("missing discharge_date must error")
	}
	_, err = IngestDischargePDF(PDFDischargeInput{ResidentRef: uuid.New(), DischargeDate: time.Now()})
	if err == nil {
		t.Fatalf("missing both extracted_text and medication_lines must error")
	}
}

func TestIngestDischargePDF_PopulatesAndNormalises(t *testing.T) {
	in := PDFDischargeInput{
		DocumentID:              "DISCH-PDF-1",
		ResidentRef:             uuid.New(),
		DischargeDate:           time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC),
		DischargingFacilityName: "St Vincent's Hospital",
		ExtractedText:           "Discharge summary…",
		StructuredMetadata:      map[string]interface{}{"page_count": 3},
		MedicationLines: []ParsedDischargeMedicationLine{
			{MedicationNameRaw: "metformin", DoseRaw: "500mg"},
			{LineNumber: 5, MedicationNameRaw: "ramipril", DoseRaw: "5mg"},
		},
	}
	doc, err := IngestDischargePDF(in)
	if err != nil {
		t.Fatalf("ingest: %v", err)
	}
	if doc.Source != DischargeSourcePDF {
		t.Fatalf("source mismatch: %s", doc.Source)
	}
	if doc.DocumentID != "DISCH-PDF-1" {
		t.Fatalf("document_id mismatch")
	}
	if doc.MedicationLines[0].LineNumber != 1 {
		t.Errorf("first line should auto-number to 1, got %d", doc.MedicationLines[0].LineNumber)
	}
	if doc.MedicationLines[1].LineNumber != 5 {
		t.Errorf("explicit line number should be preserved, got %d", doc.MedicationLines[1].LineNumber)
	}
	if doc.RawText != "Discharge summary…" {
		t.Errorf("raw_text not propagated")
	}
}

func TestIngestDischargePDF_StructuredOnlyAllowed(t *testing.T) {
	// MedicationLines populated, ExtractedText empty — valid for a
	// structured PDF form where OCR is not needed.
	doc, err := IngestDischargePDF(PDFDischargeInput{
		ResidentRef:   uuid.New(),
		DischargeDate: time.Now(),
		MedicationLines: []ParsedDischargeMedicationLine{
			{MedicationNameRaw: "warfarin"},
		},
	})
	if err != nil {
		t.Fatalf("structured-only should be accepted: %v", err)
	}
	if doc.RawText != "" {
		t.Errorf("raw_text should remain empty")
	}
}
