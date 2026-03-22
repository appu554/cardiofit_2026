package ehr

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"go.uber.org/zap"
)

func TestFHIRRestAdapter_ParseBundle_Transaction(t *testing.T) {
	bundle := FHIRBundle{
		ResourceType: "Bundle",
		Type:         "transaction",
		Entry: []BundleEntry{
			{
				Resource: mustMarshal(t, map[string]interface{}{
					"resourceType":      "Observation",
					"id":                "obs-sbp-1",
					"status":            "final",
					"effectiveDateTime": "2026-03-20T10:00:00Z",
					"code": map[string]interface{}{
						"coding": []map[string]interface{}{
							{"system": "http://loinc.org", "code": "8480-6", "display": "Systolic blood pressure"},
						},
					},
					"valueQuantity": map[string]interface{}{
						"value": 138.0,
						"unit":  "mmHg",
					},
				}),
			},
			{
				Resource: mustMarshal(t, map[string]interface{}{
					"resourceType":      "Observation",
					"id":                "obs-hba1c-1",
					"status":            "final",
					"effectiveDateTime": "2026-03-20T10:00:00Z",
					"code": map[string]interface{}{
						"coding": []map[string]interface{}{
							{"system": "http://loinc.org", "code": "4548-4", "display": "HbA1c"},
						},
					},
					"valueQuantity": map[string]interface{}{
						"value": 7.2,
						"unit":  "%",
					},
				}),
			},
		},
	}

	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("marshal bundle: %v", err)
	}

	adapter := NewFHIRRestAdapter(zap.NewNop())
	_, obs, err := adapter.ParseBundle(context.Background(), raw)
	if err != nil {
		t.Fatalf("ParseBundle: %v", err)
	}

	if len(obs) != 2 {
		t.Fatalf("expected 2 observations, got %d", len(obs))
	}

	// First observation: SBP → Vitals
	sbp := obs[0]
	if sbp.LOINCCode != "8480-6" {
		t.Errorf("expected LOINC 8480-6, got %s", sbp.LOINCCode)
	}
	if sbp.Value != 138.0 {
		t.Errorf("expected value 138, got %f", sbp.Value)
	}
	if sbp.ObservationType != canonical.ObsVitals {
		t.Errorf("expected ObsVitals, got %s", sbp.ObservationType)
	}
	if sbp.SourceType != canonical.SourceEHR {
		t.Errorf("expected SourceEHR, got %s", sbp.SourceType)
	}

	// Second observation: HbA1c → Labs
	hba1c := obs[1]
	if hba1c.LOINCCode != "4548-4" {
		t.Errorf("expected LOINC 4548-4, got %s", hba1c.LOINCCode)
	}
	if hba1c.Value != 7.2 {
		t.Errorf("expected value 7.2, got %f", hba1c.Value)
	}
	if hba1c.ObservationType != canonical.ObsLabs {
		t.Errorf("expected ObsLabs, got %s", hba1c.ObservationType)
	}
	if hba1c.SourceType != canonical.SourceEHR {
		t.Errorf("expected SourceEHR, got %s", hba1c.SourceType)
	}
}

func TestFHIRRestAdapter_ParseBundle_InvalidResourceType(t *testing.T) {
	raw := mustMarshal(t, map[string]interface{}{
		"resourceType": "Patient",
		"type":         "transaction",
		"entry":        []interface{}{},
	})

	adapter := NewFHIRRestAdapter(zap.NewNop())
	_, _, err := adapter.ParseBundle(context.Background(), raw)
	if err == nil {
		t.Fatal("expected error for non-Bundle resourceType")
	}
}

func TestFHIRRestAdapter_ParseBundle_EmptyBundle(t *testing.T) {
	raw := mustMarshal(t, map[string]interface{}{
		"resourceType": "Bundle",
		"type":         "transaction",
		"entry":        []interface{}{},
	})

	adapter := NewFHIRRestAdapter(zap.NewNop())
	_, _, err := adapter.ParseBundle(context.Background(), raw)
	if err == nil {
		t.Fatal("expected error for empty bundle entries")
	}
}

func TestClassifyByLOINC(t *testing.T) {
	tests := []struct {
		name     string
		loinc    string
		expected canonical.ObservationType
	}{
		{"SBP", "8480-6", canonical.ObsVitals},
		{"DBP", "8462-4", canonical.ObsVitals},
		{"Heart rate", "8867-4", canonical.ObsVitals},
		{"HbA1c", "4548-4", canonical.ObsLabs},
		{"eGFR", "33914-3", canonical.ObsLabs},
		{"empty code", "", canonical.ObsLabs},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyByLOINC(tt.loinc)
			if got != tt.expected {
				t.Errorf("classifyByLOINC(%q) = %s, want %s", tt.loinc, got, tt.expected)
			}
		})
	}
}

// mustMarshal is a test helper that marshals v to JSON or fails the test.
func mustMarshal(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("mustMarshal: %v", err)
	}
	return data
}
