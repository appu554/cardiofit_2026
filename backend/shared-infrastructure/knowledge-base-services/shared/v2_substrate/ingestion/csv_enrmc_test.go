package ingestion

import (
	"strings"
	"testing"
)

const goodHeader = "patient_id,facility_id,ihi,medicare,family_name,given_name,dob," +
	"medication_name,strength,form,route,frequency,start_date,end_date," +
	"prescriber_name,prescriber_ahpra,indication_text"

func TestParseCSV_HappyPath(t *testing.T) {
	body := goodHeader + "\n" +
		"P1,F1,8003608000000001,1234567890,Smith,Jane,1942-04-12," +
		"paracetamol,500mg,tablet,ORAL,QID,2026-01-01,,Dr Adams,MED0001234567,osteoarthritis pain\n"

	rows, parseErrs, err := ParseCSV(strings.NewReader(body))
	if err != nil {
		t.Fatalf("unexpected fatal error: %v", err)
	}
	if len(parseErrs) != 0 {
		t.Fatalf("unexpected parse errors: %v", parseErrs)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	r := rows[0]
	if r.PatientID != "P1" || r.IHI != "8003608000000001" {
		t.Errorf("identifiers wrong: %+v", r)
	}
	if r.MedicationName != "paracetamol" || r.Frequency != "QID" {
		t.Errorf("clinical fields wrong: %+v", r)
	}
	if r.EndDate != "" {
		t.Errorf("end_date should be empty, got %q", r.EndDate)
	}
	if r.IndicationText != "osteoarthritis pain" {
		t.Errorf("indication_text wrong: %q", r.IndicationText)
	}
	if r.LineNumber != 2 {
		t.Errorf("line number = %d, want 2", r.LineNumber)
	}
}

func TestParseCSV_BOMHandled(t *testing.T) {
	body := "\ufeff" + goodHeader + "\n" +
		"P1,F1,8003608000000001,,Smith,Jane,1942-04-12," +
		"amlodipine,5mg,tablet,ORAL,OD,2026-01-01,,Dr Adams,,hypertension\n"
	rows, _, err := ParseCSV(strings.NewReader(body))
	if err != nil {
		t.Fatalf("BOM should be tolerated: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
}

func TestParseCSV_EmptyRowsSkipped(t *testing.T) {
	body := goodHeader + "\n" +
		"\n" +
		"P1,F1,,,Smith,Jane,1942-04-12,paracetamol,500mg,tablet,ORAL,QID,2026-01-01,,Dr A,,\n" +
		",,,,,,,,,,,,,,,,\n"

	rows, parseErrs, err := ParseCSV(strings.NewReader(body))
	if err != nil {
		t.Fatalf("unexpected fatal: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 non-empty row, got %d", len(rows))
	}
	// Indication empty is allowed (no parse error).
	for _, pe := range parseErrs {
		if pe.Field == "indication_text" {
			t.Errorf("indication_text empty should NOT be a parse error: %v", pe)
		}
	}
}

func TestParseCSV_MissingRequiredFieldEmitsParseError(t *testing.T) {
	body := goodHeader + "\n" +
		"P1,F1,,,Smith,Jane,1942-04-12,,500mg,tablet,ORAL,QID,2026-01-01,,Dr A,,pain\n"

	rows, parseErrs, err := ParseCSV(strings.NewReader(body))
	if err != nil {
		t.Fatalf("unexpected fatal: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row even when required field empty, got %d", len(rows))
	}
	found := false
	for _, pe := range parseErrs {
		if pe.Field == "medication_name" && pe.LineNumber == 2 {
			found = true
		}
	}
	if !found {
		t.Errorf("expected parse error for empty medication_name; got %+v", parseErrs)
	}
}

func TestParseCSV_HeaderValidation(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"missing column", "patient_id,facility_id\nP1,F1\n"},
		{"unexpected column", goodHeader + ",surprise\n"},
		{"empty input", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := ParseCSV(strings.NewReader(tc.body))
			if err == nil {
				t.Errorf("expected fatal error for %q", tc.name)
			}
		})
	}
}

func TestParseCSV_BothNamesEmpty(t *testing.T) {
	body := goodHeader + "\n" +
		"P1,F1,,1234,,,1942-04-12,paracetamol,500mg,tab,ORAL,QID,2026-01-01,,Dr A,,\n"
	_, parseErrs, err := ParseCSV(strings.NewReader(body))
	if err != nil {
		t.Fatalf("unexpected fatal: %v", err)
	}
	found := false
	for _, pe := range parseErrs {
		if strings.Contains(pe.Field, "family_name") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected parse error when both names empty: %+v", parseErrs)
	}
}
