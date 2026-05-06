package ingestion

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DischargeDocumentSource enumerates the provenance buckets for the
// kb-20 discharge_documents table.
type DischargeDocumentSource string

const (
	DischargeSourcePDF    DischargeDocumentSource = "pdf"
	DischargeSourceMHRCDA DischargeDocumentSource = "mhr_cda"
	DischargeSourceManual DischargeDocumentSource = "manual"
)

// IsValidDischargeDocumentSource reports whether s is a recognised source.
func IsValidDischargeDocumentSource(s string) bool {
	switch DischargeDocumentSource(s) {
	case DischargeSourcePDF, DischargeSourceMHRCDA, DischargeSourceManual:
		return true
	}
	return false
}

// ParsedDischargeDocument is the storage-layer DTO produced by the
// ingestion adapters. Wave 4 carries it onto the discharge_documents +
// discharge_medication_lines tables; the FHIR mapper turns it into a
// DocumentReference at egress.
type ParsedDischargeDocument struct {
	Source                  DischargeDocumentSource
	DocumentID              string                  // optional external doc id
	ResidentRef             uuid.UUID               // assigned by the caller (identity match)
	DischargeDate           time.Time
	DischargingFacilityName string
	RawText                 string                  // OCR/parse output; empty for structured-only sources
	StructuredPayload       map[string]interface{}  // serialised as JSONB by the storage layer
	MedicationLines         []ParsedDischargeMedicationLine
}

// ParsedDischargeMedicationLine is one row in the discharge_medication_lines
// table. Fields mirror the schema; LineNumber is 1-based.
type ParsedDischargeMedicationLine struct {
	LineNumber        int
	MedicationNameRaw string
	AMTCode           string
	DoseRaw           string
	FrequencyRaw      string
	RouteRaw          string
	IndicationText    string
	Notes             string
}

// PDFDischargeInput is the input shape for IngestDischargePDF. The OCR
// step is OUT of scope for Wave 4 — callers pass the already-extracted
// text plus structured metadata. Real OCR wiring deferred to V1; see
// the OCRBackend interface below for the seam.
type PDFDischargeInput struct {
	DocumentID              string
	ResidentRef             uuid.UUID
	DischargeDate           time.Time
	DischargingFacilityName string
	// ExtractedText is the already-OCR'd full-document text. Wave 4
	// stores it verbatim into discharge_documents.raw_text.
	ExtractedText string
	// StructuredMetadata carries optional PDF-form metadata (author,
	// creation timestamp, page count, etc.) that the storage layer
	// persists as JSONB.
	StructuredMetadata map[string]interface{}
	// MedicationLines is the parsed med list when the caller has already
	// extracted line-level structure. When empty, the storage layer
	// keeps RawText only — V1 will plug a structured-extraction pipeline
	// behind a separate adapter.
	MedicationLines []ParsedDischargeMedicationLine
}

// IngestDischargePDF validates and packages a PDF discharge document
// for storage. Pure function; no IO. The OCR step is the caller's
// responsibility — Wave 4 accepts pre-extracted text and structure.
//
// Real OCR wiring is deferred to V1: the OCRBackend interface below is
// the seam where a Tesseract / textract / cloud-OCR client will plug
// in. Today the function simply trusts ExtractedText.
//
// Returns an error for missing required fields (resident ref,
// discharge date). Empty ExtractedText is allowed when MedicationLines
// is non-empty (e.g. a structured PDF form parsed without OCR).
func IngestDischargePDF(in PDFDischargeInput) (*ParsedDischargeDocument, error) {
	if in.ResidentRef == uuid.Nil {
		return nil, errors.New("ingestion/discharge_pdf: resident_ref required")
	}
	if in.DischargeDate.IsZero() {
		return nil, errors.New("ingestion/discharge_pdf: discharge_date required")
	}
	if strings.TrimSpace(in.ExtractedText) == "" && len(in.MedicationLines) == 0 {
		return nil, errors.New("ingestion/discharge_pdf: either extracted_text or medication_lines required")
	}
	return &ParsedDischargeDocument{
		Source:                  DischargeSourcePDF,
		DocumentID:              in.DocumentID,
		ResidentRef:             in.ResidentRef,
		DischargeDate:           in.DischargeDate.UTC(),
		DischargingFacilityName: in.DischargingFacilityName,
		RawText:                 in.ExtractedText,
		StructuredPayload:       in.StructuredMetadata,
		MedicationLines:         normaliseLineNumbers(in.MedicationLines),
	}, nil
}

// OCRBackend is the deferred interface for real PDF OCR wiring. V1 will
// implement this against Tesseract / textract / a cloud-OCR provider
// and the ingestion path will become:
//
//	text, err := backend.ExtractText(ctx, pdfBytes)
//	doc, err := IngestDischargePDF(PDFDischargeInput{ExtractedText: text, ...})
//
// TODO(V1): wire a concrete implementation. Today this interface exists
// only to document the seam; no production caller depends on it.
type OCRBackend interface {
	// ExtractText takes a raw PDF byte stream and returns the extracted
	// text plus an optional structured-metadata bag. Implementations
	// must be deterministic-enough for the substrate's content-hash
	// audit (V1 retention work) — non-deterministic OCR output should
	// be paired with a hash-on-bytes rather than hash-on-text.
	ExtractText(pdfBytes []byte) (text string, metadata map[string]interface{}, err error)
}

// normaliseLineNumbers ensures every line carries a 1-based LineNumber.
// Caller-supplied non-zero values are preserved; zeros are filled with
// the slice index + 1.
func normaliseLineNumbers(lines []ParsedDischargeMedicationLine) []ParsedDischargeMedicationLine {
	out := make([]ParsedDischargeMedicationLine, len(lines))
	for i, ln := range lines {
		out[i] = ln
		if out[i].LineNumber == 0 {
			out[i].LineNumber = i + 1
		}
	}
	return out
}
