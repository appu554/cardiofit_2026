package fhir

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AU FHIR DocumentReference category codes used by discharge documents.
const (
	// LOINC discharge summary code; carried in DocumentReference.type.
	loincDischargeSummaryCode = "18842-5"
	// FHIR DocumentReference.category preferred binding (HL7 v3 ActCode).
	categoryClinicalNoteSystem = "http://terminology.hl7.org/CodeSystem/v3-ActCode"
	categoryDischargeCode      = "DI"
	categoryDischargeDisplay   = "Discharge summary"
)

// DischargeDocumentInput is the boundary-layer DTO converted onto an
// AU FHIR DocumentReference. The reconciliation engine produces a
// ParsedDischargeDocument; this struct is the read-side view that the
// FHIR egress code consumes (kept distinct from the ingestion DTO so
// the FHIR mapper has no compile-time dep on the ingestion package).
type DischargeDocumentInput struct {
	ID                      uuid.UUID
	ResidentRef             uuid.UUID
	Source                  string // "pdf" | "mhr_cda" | "manual"
	DocumentID              string
	DischargeDate           time.Time
	DischargingFacilityName string
	RawText                 string // optional inlined attachment payload
	IngestedAt              time.Time
}

// DischargeDocumentToAUFHIR maps a discharge-document row onto an AU
// FHIR DocumentReference (R4) as a generic map[string]interface{}
// suitable for json.Marshal to wire format.
//
// Output highlights:
//
//   - resourceType: DocumentReference
//   - status: current
//   - type.coding: LOINC 18842-5 "Discharge summary"
//   - category[0].coding: HL7 v3 ActCode "DI" (Discharge summary)
//   - subject.reference: Patient/<resident_ref>
//   - context.period.end: discharge_date
//   - description: discharging facility name
//   - content[0].attachment: text/plain inline base64 of raw_text
//     (omitted when raw_text is empty)
//   - identifier[0]: canonical {system, value} pair from {source,
//     document_id} so the substrate row can be looked up by the
//     external doc id later.
//
// Egress validates input fields: ResidentRef must be non-nil and
// DischargeDate must be set. ErrInvalid is returned otherwise.
func DischargeDocumentToAUFHIR(in DischargeDocumentInput) (map[string]interface{}, error) {
	if in.ResidentRef == uuid.Nil {
		return nil, fmt.Errorf("fhir/document_reference_mapper: resident_ref required")
	}
	if in.DischargeDate.IsZero() {
		return nil, fmt.Errorf("fhir/document_reference_mapper: discharge_date required")
	}

	r := map[string]interface{}{
		"resourceType": "DocumentReference",
		"status":       "current",
		"type": map[string]interface{}{
			"coding": []map[string]interface{}{{
				"system":  "http://loinc.org",
				"code":    loincDischargeSummaryCode,
				"display": "Discharge summary",
			}},
		},
		"category": []map[string]interface{}{{
			"coding": []map[string]interface{}{{
				"system":  categoryClinicalNoteSystem,
				"code":    categoryDischargeCode,
				"display": categoryDischargeDisplay,
			}},
		}},
		"subject": map[string]interface{}{
			"reference": "Patient/" + in.ResidentRef.String(),
		},
		"context": map[string]interface{}{
			"period": map[string]interface{}{
				"end": in.DischargeDate.UTC().Format(time.RFC3339),
			},
		},
	}

	// id mirrors the substrate row id when present.
	if in.ID != uuid.Nil {
		r["id"] = in.ID.String()
	}

	// description carries the discharging facility name when present.
	if in.DischargingFacilityName != "" {
		r["description"] = in.DischargingFacilityName
	}

	// identifier carries the external doc id under a per-source system URI.
	if in.DocumentID != "" {
		r["identifier"] = []map[string]interface{}{{
			"system": dischargeIdentifierSystem(in.Source),
			"value":  in.DocumentID,
		}}
	}

	// date = ingestion timestamp when available; omitted otherwise.
	if !in.IngestedAt.IsZero() {
		r["date"] = in.IngestedAt.UTC().Format(time.RFC3339)
	}

	// content[0].attachment — inline the raw text only when present.
	if in.RawText != "" {
		r["content"] = []map[string]interface{}{{
			"attachment": map[string]interface{}{
				"contentType": "text/plain",
				"language":    "en-AU",
				"data":        base64.StdEncoding.EncodeToString([]byte(in.RawText)),
			},
		}}
	} else {
		// FHIR requires content[*] to exist; emit a minimal entry pointing
		// nowhere so structural validators stay happy.
		r["content"] = []map[string]interface{}{{
			"attachment": map[string]interface{}{
				"contentType": "text/plain",
				"language":    "en-AU",
			},
		}}
	}

	return r, nil
}

// dischargeIdentifierSystem returns the FHIR identifier.system URI to
// use for an external discharge-document id, namespaced by source so
// the same external id from two different sources never collides.
func dischargeIdentifierSystem(source string) string {
	switch source {
	case "mhr_cda":
		return "https://vaidshala.health/fhir/sid/mhr-discharge-document-id"
	case "pdf":
		return "https://vaidshala.health/fhir/sid/pdf-discharge-document-id"
	case "manual":
		return "https://vaidshala.health/fhir/sid/manual-discharge-document-id"
	}
	return "https://vaidshala.health/fhir/sid/discharge-document-id"
}
