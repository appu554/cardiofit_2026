package labs

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"go.uber.org/zap"
)

type mockCodeRegistry struct{}

func (m *mockCodeRegistry) LookupLOINC(labID, labCode string) (string, string, string, error) {
	mappings := map[string]struct{ loinc, name, unit string }{
		"TSH":   {"11580-8", "TSH", "mIU/L"},
		"FT3":   {"3051-0", "Free T3", "pg/mL"},
		"FT4":   {"3024-7", "Free T4", "ng/dL"},
		"HBA1C": {"4548-4", "HbA1c", "%"},
		"FBG":   {"1558-6", "Fasting Blood Glucose", "mg/dL"},
		"CREAT": {"2160-0", "Creatinine", "mg/dL"},
		"EGFR":  {"33914-3", "eGFR", "mL/min/1.73m2"},
	}

	if m, ok := mappings[labCode]; ok {
		return m.loinc, m.name, m.unit, nil
	}
	return "", "", "", nil
}

func TestThyrocareAdapter_Parse(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	adapter := NewThyrocareAdapter("test-key", &mockCodeRegistry{}, logger)

	payload := thyrocarePayload{
		OrderNo:    "TC-2026-001",
		BenName:    "Test Patient",
		BenMobile:  "9876543210",
		SampleDate: "15-03-2026",
		ReportDate: "16-03-2026 10:30",
		Tests: []thyrocareTestResult{
			{TestCode: "TSH", TestName: "TSH", Result: "2.5", Unit: "mIU/L", IsAbnormal: "N"},
			{TestCode: "HBA1C", TestName: "HbA1c", Result: "7.2", Unit: "%", IsAbnormal: "Y"},
			{TestCode: "FBG", TestName: "Fasting Glucose", Result: "145", Unit: "mg/dL", IsAbnormal: "Y"},
		},
	}

	raw, _ := json.Marshal(payload)
	observations, err := adapter.Parse(context.Background(), raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(observations) != 3 {
		t.Fatalf("expected 3 observations, got %d", len(observations))
	}

	tsh := observations[0]
	if tsh.LOINCCode != "11580-8" {
		t.Errorf("TSH LOINC: expected 11580-8, got %s", tsh.LOINCCode)
	}
	if tsh.Value != 2.5 {
		t.Errorf("TSH value: expected 2.5, got %f", tsh.Value)
	}
	if tsh.SourceType != canonical.SourceLab {
		t.Errorf("expected LAB source, got %s", tsh.SourceType)
	}

	hba1c := observations[1]
	if hba1c.LOINCCode != "4548-4" {
		t.Errorf("HbA1c LOINC: expected 4548-4, got %s", hba1c.LOINCCode)
	}
	if hba1c.Value != 7.2 {
		t.Errorf("HbA1c value: expected 7.2, got %f", hba1c.Value)
	}
}

func TestThyrocareAdapter_ParseEmpty(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	adapter := NewThyrocareAdapter("key", &mockCodeRegistry{}, logger)

	payload := thyrocarePayload{Tests: []thyrocareTestResult{}}
	raw, _ := json.Marshal(payload)

	_, err := adapter.Parse(context.Background(), raw)
	if err == nil {
		t.Error("expected error for empty tests")
	}
}

func TestThyrocareAdapter_UnmappedCode(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	adapter := NewThyrocareAdapter("key", &mockCodeRegistry{}, logger)

	payload := thyrocarePayload{
		SampleDate: "15-03-2026",
		Tests: []thyrocareTestResult{
			{TestCode: "UNKNOWN_TEST", TestName: "Unknown", Result: "5.0", Unit: "mg/dL"},
		},
	}

	raw, _ := json.Marshal(payload)
	observations, err := adapter.Parse(context.Background(), raw)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(observations) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(observations))
	}

	obs := observations[0]
	if obs.QualityScore >= 0.9 {
		t.Errorf("unmapped code should have lower quality score, got %f", obs.QualityScore)
	}

	hasFlag := false
	for _, f := range obs.Flags {
		if f == canonical.FlagUnmappedCode {
			hasFlag = true
		}
	}
	if !hasFlag {
		t.Error("expected UNMAPPED_CODE flag for unknown test code")
	}
}

func TestThyrocareAdapter_ValidateAuth(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	adapter := NewThyrocareAdapter("correct-key", &mockCodeRegistry{}, logger)

	if !adapter.ValidateWebhookAuth("correct-key") {
		t.Error("should accept correct API key")
	}
	if adapter.ValidateWebhookAuth("wrong-key") {
		t.Error("should reject wrong API key")
	}
}
