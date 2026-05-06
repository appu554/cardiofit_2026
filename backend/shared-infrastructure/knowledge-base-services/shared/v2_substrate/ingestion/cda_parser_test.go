package ingestion

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCDAPathology_SyntheticFixture(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "synthetic_cda_pathology.xml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	res, err := ParseCDAPathology(raw)
	if err != nil {
		t.Fatalf("ParseCDAPathology: %v", err)
	}

	if got, want := res.DocumentID, "DOC-SYN-0001"; got != want {
		t.Errorf("DocumentID = %q, want %q", got, want)
	}
	if got, want := res.PatientIHI, "8003608000000001"; got != want {
		t.Errorf("PatientIHI = %q, want %q", got, want)
	}
	if res.AuthoredAt.IsZero() {
		t.Errorf("AuthoredAt should be parsed, got zero")
	}

	if got, want := len(res.Observations), 3; got != want {
		t.Fatalf("len(Observations) = %d, want %d", got, want)
	}

	// Observation 1: potassium PQ + LOINC primary + SNOMED translation + H flag
	po := res.Observations[0]
	if po.LOINCCode != "2823-3" {
		t.Errorf("obs[0].LOINCCode = %q, want 2823-3", po.LOINCCode)
	}
	if po.SNOMEDCode != "271001000087101" {
		t.Errorf("obs[0].SNOMEDCode = %q, want 271001000087101", po.SNOMEDCode)
	}
	if po.Value == nil || *po.Value != 5.8 {
		t.Errorf("obs[0].Value = %v, want 5.8", po.Value)
	}
	if po.Unit != "mmol/L" {
		t.Errorf("obs[0].Unit = %q, want mmol/L", po.Unit)
	}
	if po.AbnormalFlag != "high" {
		t.Errorf("obs[0].AbnormalFlag = %q, want high", po.AbnormalFlag)
	}

	// Observation 2: eGFR PQ + L flag, no SNOMED translation
	po = res.Observations[1]
	if po.LOINCCode != "33914-3" {
		t.Errorf("obs[1].LOINCCode = %q, want 33914-3", po.LOINCCode)
	}
	if po.SNOMEDCode != "" {
		t.Errorf("obs[1].SNOMEDCode = %q, want empty", po.SNOMEDCode)
	}
	if po.Value == nil || *po.Value != 42 {
		t.Errorf("obs[1].Value = %v, want 42", po.Value)
	}
	if po.AbnormalFlag != "low" {
		t.Errorf("obs[1].AbnormalFlag = %q, want low", po.AbnormalFlag)
	}

	// Observation 3: ST microscopy comment — text payload
	po = res.Observations[2]
	if po.LOINCCode != "11556-8" {
		t.Errorf("obs[2].LOINCCode = %q, want 11556-8", po.LOINCCode)
	}
	if po.Value != nil {
		t.Errorf("obs[2].Value should be nil for ST value")
	}
	if po.ValueText != "No organisms seen on Gram stain." {
		t.Errorf("obs[2].ValueText = %q", po.ValueText)
	}
	if po.AbnormalFlag != "" {
		t.Errorf("obs[2].AbnormalFlag = %q, want empty", po.AbnormalFlag)
	}
}

func TestParseCDAPathology_MalformedXML(t *testing.T) {
	_, err := ParseCDAPathology([]byte("not xml"))
	if err == nil {
		t.Fatalf("expected error on malformed XML, got nil")
	}
}

func TestStubMHRSOAPClient_ReturnsDeferredError(t *testing.T) {
	c := NewStubMHRSOAPClient()
	if _, err := c.GetPathologyDocumentList(nil, "8003608000000001", parseCDATime("20260101000000")); err == nil {
		t.Fatalf("GetPathologyDocumentList should return ErrMHRWiringDeferred")
	}
	if _, err := c.FetchCDADocument(nil, "DOC-SYN-0001"); err == nil {
		t.Fatalf("FetchCDADocument should return ErrMHRWiringDeferred")
	}
}
