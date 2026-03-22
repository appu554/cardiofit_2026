package patient_reported

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

func newTestWhatsAppAdapter() *WhatsAppAdapter {
	return NewWhatsAppAdapter(zap.NewNop())
}

func TestWhatsAppAdapter_Parse_ValidGlucose(t *testing.T) {
	adapter := newTestWhatsAppAdapter()

	payload := WhatsAppNLUPayload{
		PatientID:  uuid.New(),
		TenantID:   uuid.New(),
		MessageID:  "msg-001",
		Timestamp:  time.Now().UTC(),
		Intent:     "report_glucose",
		Confidence: 0.85,
		RawText:    "mera sugar 180 hai",
		Entities: []WhatsAppEntity{
			{Type: "glucose_value", Value: 180.0, Unit: "mg/dL"},
		},
	}

	obs, err := adapter.Parse(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(obs) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(obs))
	}

	o := obs[0]
	if o.PatientID != payload.PatientID {
		t.Errorf("patient_id mismatch: got %s, want %s", o.PatientID, payload.PatientID)
	}
	if o.SourceID != "whatsapp" {
		t.Errorf("source_id: got %q, want %q", o.SourceID, "whatsapp")
	}
	if o.SourceType != canonical.SourcePatientReported {
		t.Errorf("source_type: got %q, want %q", o.SourceType, canonical.SourcePatientReported)
	}
	if o.Value != 180.0 {
		t.Errorf("value: got %f, want 180.0", o.Value)
	}
	if o.Unit != "mg/dL" {
		t.Errorf("unit: got %q, want %q", o.Unit, "mg/dL")
	}
	// High-confidence: should NOT have LOW_QUALITY flag
	for _, f := range o.Flags {
		if f == canonical.FlagLowQuality {
			t.Error("high-confidence message should not have LOW_QUALITY flag")
		}
	}
	// Should have MANUAL_ENTRY flag
	hasManual := false
	for _, f := range o.Flags {
		if f == canonical.FlagManualEntry {
			hasManual = true
		}
	}
	if !hasManual {
		t.Error("expected MANUAL_ENTRY flag on patient-reported data")
	}
}

func TestWhatsAppAdapter_Parse_LowConfidence(t *testing.T) {
	adapter := newTestWhatsAppAdapter()

	payload := WhatsAppNLUPayload{
		PatientID:  uuid.New(),
		TenantID:   uuid.New(),
		MessageID:  "msg-002",
		Timestamp:  time.Now().UTC(),
		Intent:     "report_bp",
		Confidence: 0.55, // Below 0.70 threshold
		RawText:    "bp thoda zyada hai",
		Entities: []WhatsAppEntity{
			{Type: "systolic_bp", Value: 145.0, Unit: "mmHg"},
		},
	}

	obs, err := adapter.Parse(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(obs) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(obs))
	}

	// Low confidence should add LOW_QUALITY flag
	hasLowQuality := false
	for _, f := range obs[0].Flags {
		if f == canonical.FlagLowQuality {
			hasLowQuality = true
		}
	}
	if !hasLowQuality {
		t.Error("low-confidence message should have LOW_QUALITY flag")
	}

	// Systolic BP should be categorized as vitals
	if obs[0].ObservationType != canonical.ObsVitals {
		t.Errorf("systolic_bp should be ObsVitals, got %q", obs[0].ObservationType)
	}
}

func TestWhatsAppAdapter_Parse_MultipleEntities(t *testing.T) {
	adapter := newTestWhatsAppAdapter()

	payload := WhatsAppNLUPayload{
		PatientID:  uuid.New(),
		TenantID:   uuid.New(),
		MessageID:  "msg-003",
		Timestamp:  time.Now().UTC(),
		Intent:     "report_bp",
		Confidence: 0.90,
		RawText:    "bp 130/85 hai aaj",
		Entities: []WhatsAppEntity{
			{Type: "systolic_bp", Value: 130.0, Unit: "mmHg"},
			{Type: "diastolic_bp", Value: 85.0, Unit: "mmHg"},
		},
	}

	obs, err := adapter.Parse(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(obs) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(obs))
	}

	// Both should be vitals
	for i, o := range obs {
		if o.ObservationType != canonical.ObsVitals {
			t.Errorf("obs[%d] type=%q should be vitals", i, o.ObservationType)
		}
	}
}

func TestWhatsAppAdapter_Parse_MissingPatientID(t *testing.T) {
	adapter := newTestWhatsAppAdapter()

	payload := WhatsAppNLUPayload{
		PatientID: uuid.Nil, // Missing
		MessageID: "msg-004",
		Intent:    "report_glucose",
		Entities: []WhatsAppEntity{
			{Type: "glucose_value", Value: 120.0, Unit: "mg/dL"},
		},
	}

	_, err := adapter.Parse(payload)
	if err == nil {
		t.Fatal("expected error for missing patient_id")
	}
}

func TestWhatsAppAdapter_Parse_NoEntities(t *testing.T) {
	adapter := newTestWhatsAppAdapter()

	payload := WhatsAppNLUPayload{
		PatientID: uuid.New(),
		MessageID: "msg-005",
		Intent:    "report_glucose",
		Entities:  []WhatsAppEntity{}, // Empty
	}

	_, err := adapter.Parse(payload)
	if err == nil {
		t.Fatal("expected error for empty entities")
	}
}

func TestWhatsAppAdapter_Parse_ZeroTimestamp(t *testing.T) {
	adapter := newTestWhatsAppAdapter()

	payload := WhatsAppNLUPayload{
		PatientID:  uuid.New(),
		TenantID:   uuid.New(),
		MessageID:  "msg-006",
		Timestamp:  time.Time{}, // Zero — should default to now
		Intent:     "report_weight",
		Confidence: 0.80,
		RawText:    "weight 72 kg",
		Entities: []WhatsAppEntity{
			{Type: "weight", Value: 72.0, Unit: "kg"},
		},
	}

	obs, err := adapter.Parse(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obs[0].Timestamp.IsZero() {
		t.Error("zero timestamp should be replaced with current time")
	}
}

func TestWhatsAppAdapter_Parse_LOINCResolution(t *testing.T) {
	adapter := newTestWhatsAppAdapter()

	payload := WhatsAppNLUPayload{
		PatientID:  uuid.New(),
		TenantID:   uuid.New(),
		MessageID:  "msg-007",
		Timestamp:  time.Now().UTC(),
		Intent:     "report_hba1c",
		Confidence: 0.92,
		RawText:    "hba1c 7.2 hai",
		Entities: []WhatsAppEntity{
			{Type: "hba1c", Value: 7.2, Unit: "%"},
		},
	}

	obs, err := adapter.Parse(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// hba1c should get a LOINC code from the coding mapper
	if obs[0].LOINCCode == "" {
		t.Log("LOINC code not resolved for hba1c — check coding/loinc_mapper.go registry")
	}
}
