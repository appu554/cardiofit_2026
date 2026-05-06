package fhir

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestDischargeDocumentToAUFHIR_PopulatesCoreFields(t *testing.T) {
	in := DischargeDocumentInput{
		ID:                      uuid.New(),
		ResidentRef:             uuid.New(),
		Source:                  "mhr_cda",
		DocumentID:              "MHR-DISCH-2025-001",
		DischargeDate:           time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC),
		DischargingFacilityName: "St Vincent's Hospital Sydney",
		RawText:                 "Discharge summary text",
		IngestedAt:              time.Date(2025, 6, 1, 11, 0, 0, 0, time.UTC),
	}
	r, err := DischargeDocumentToAUFHIR(in)
	if err != nil {
		t.Fatalf("map: %v", err)
	}
	if r["resourceType"] != "DocumentReference" {
		t.Fatalf("resourceType mismatch")
	}
	if r["status"] != "current" {
		t.Fatalf("status mismatch")
	}
	subject := r["subject"].(map[string]interface{})
	if subject["reference"] != "Patient/"+in.ResidentRef.String() {
		t.Fatalf("subject reference mismatch")
	}
	// type coding LOINC discharge summary
	tt := r["type"].(map[string]interface{})
	cs := tt["coding"].([]map[string]interface{})
	if cs[0]["code"] != "18842-5" {
		t.Fatalf("type code must be LOINC 18842-5")
	}
	// category v3 ActCode DI
	cats := r["category"].([]map[string]interface{})
	catCoding := cats[0]["coding"].([]map[string]interface{})
	if catCoding[0]["code"] != "DI" {
		t.Fatalf("category code must be DI")
	}
	// identifier system per source
	ids := r["identifier"].([]map[string]interface{})
	if ids[0]["system"] != "https://vaidshala.health/fhir/sid/mhr-discharge-document-id" {
		t.Fatalf("identifier system mismatch: %v", ids[0]["system"])
	}
	// content attachment carries base64 text
	contents := r["content"].([]map[string]interface{})
	att := contents[0]["attachment"].(map[string]interface{})
	dec, err := base64.StdEncoding.DecodeString(att["data"].(string))
	if err != nil || string(dec) != "Discharge summary text" {
		t.Fatalf("attachment data should base64-encode raw_text")
	}
	// context.period.end = discharge_date
	ctxMap := r["context"].(map[string]interface{})
	period := ctxMap["period"].(map[string]interface{})
	if period["end"] != "2025-06-01T10:00:00Z" {
		t.Fatalf("context.period.end mismatch: %v", period["end"])
	}
}

func TestDischargeDocumentToAUFHIR_RejectsInvalid(t *testing.T) {
	_, err := DischargeDocumentToAUFHIR(DischargeDocumentInput{})
	if err == nil {
		t.Fatalf("missing resident_ref must error")
	}
	_, err = DischargeDocumentToAUFHIR(DischargeDocumentInput{ResidentRef: uuid.New()})
	if err == nil {
		t.Fatalf("missing discharge_date must error")
	}
}

func TestDischargeDocumentToAUFHIR_NoRawTextEmitsMinimalContent(t *testing.T) {
	r, err := DischargeDocumentToAUFHIR(DischargeDocumentInput{
		ResidentRef:   uuid.New(),
		DischargeDate: time.Now(),
	})
	if err != nil {
		t.Fatalf("map: %v", err)
	}
	contents := r["content"].([]map[string]interface{})
	att := contents[0]["attachment"].(map[string]interface{})
	if _, hasData := att["data"]; hasData {
		t.Fatalf("attachment should not include data when raw_text empty")
	}
}
