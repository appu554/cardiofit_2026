package exports

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"
)

// testPack returns a full RPLPack with one item per APC competency dimension.
func testPack() RPLPack {
	return RPLPack{
		ID:           uuid.New().String(),
		PharmacistID: uuid.New().String(),
		GeneratedAt:  time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
		Items: []EvidenceItem{
			{Title: "Case A", Dimension: "clinical_assessment", Anonymised: true, Annotation: "x"},
			{Title: "Case B", Dimension: "medication_review", Anonymised: true, Annotation: "y"},
			{Title: "Case C", Dimension: "communication", Anonymised: true, Annotation: "z"},
			{Title: "Case D", Dimension: "quality_use_of_medicines", Anonymised: true, Annotation: "w"},
			{Title: "Case E", Dimension: "professional_practice", Anonymised: true, Annotation: "v"},
		},
	}
}

// TestPortfolioPDF_RendersFiveDimensions is the verbatim plan test (Task 6 spec).
// It verifies that a full pack with all five APC dimensions produces a real PDF
// of at least 1000 bytes starting with the PDF magic header.
func TestPortfolioPDF_RendersFiveDimensions(t *testing.T) {
	pack := testPack()
	var buf bytes.Buffer
	if err := RenderPortfolioPDF(pack, "Pharmacist Name", &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if buf.Len() < 1000 {
		t.Errorf("rendered PDF too small (%d bytes); not real PDF output", buf.Len())
	}
	// PDF magic bytes
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Errorf("output does not start with %%PDF- magic")
	}
}

// TestPortfolioPDF_StartsWithPDFMagic confirms the %PDF- magic bytes at offset 0.
// Covered by TestPortfolioPDF_RendersFiveDimensions but expressed as a standalone
// test to make the contract explicit.
func TestPortfolioPDF_StartsWithPDFMagic(t *testing.T) {
	var buf bytes.Buffer
	if err := RenderPortfolioPDF(testPack(), "Jane Doe", &buf); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Errorf("output does not start with %%PDF- magic bytes; got %q", buf.Bytes()[:min(10, buf.Len())])
	}
}

// TestPortfolioPDF_EmptyItemsStillProducesValidPDF verifies that a pack with no
// Items (all dimensions have zero evidence) still produces a valid, non-empty PDF
// rather than returning an error or an empty buffer.
func TestPortfolioPDF_EmptyItemsStillProducesValidPDF(t *testing.T) {
	pack := RPLPack{
		ID:           uuid.New().String(),
		PharmacistID: uuid.New().String(),
		GeneratedAt:  time.Now().UTC(),
		Items:        []EvidenceItem{},
	}
	var buf bytes.Buffer
	if err := RenderPortfolioPDF(pack, "Empty Pharmacist", &buf); err != nil {
		t.Fatalf("render with empty items: %v", err)
	}
	if buf.Len() == 0 {
		t.Error("expected non-empty PDF output for empty pack, got 0 bytes")
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Errorf("empty-items PDF does not start with %%PDF- magic")
	}
}

// TestPortfolioPDF_DimensionOrderingDeterministic renders the same pack twice and
// asserts byte-identical output. This verifies that:
//  1. The five APC dimensions are rendered in a fixed order (not map-iteration order).
//  2. The creation-date metadata is pinned to a deterministic value via
//     pdf.SetCreationDate(time.Time{}) so two renders produce the same bytes.
func TestPortfolioPDF_DimensionOrderingDeterministic(t *testing.T) {
	pack := RPLPack{
		ID:           "fixed-id-for-determinism",
		PharmacistID: "fixed-pharm-id",
		GeneratedAt:  time.Date(2026, 5, 9, 0, 0, 0, 0, time.UTC),
		Items: []EvidenceItem{
			{Title: "Item 1", Dimension: "clinical_assessment", Anonymised: true, Annotation: "note1"},
			{Title: "Item 2", Dimension: "professional_practice", Anonymised: true, Annotation: "note2"},
			{Title: "Item 3", Dimension: "communication", Anonymised: true, Annotation: "note3"},
		},
	}

	var buf1, buf2 bytes.Buffer
	if err := RenderPortfolioPDF(pack, "Deterministic Pharmacist", &buf1); err != nil {
		t.Fatalf("first render: %v", err)
	}
	if err := RenderPortfolioPDF(pack, "Deterministic Pharmacist", &buf2); err != nil {
		t.Fatalf("second render: %v", err)
	}
	if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
		t.Errorf("two renders of identical input produced different output: "+
			"first=%d bytes, second=%d bytes", buf1.Len(), buf2.Len())
	}
}

// min returns the smaller of a and b. Used to avoid panicking when slicing a
// short buffer for error message formatting.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
