package services

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

func newTestSignalCardBuilder() *SignalCardBuilder {
	log, _ := zap.NewDevelopment()
	return NewSignalCardBuilder(log)
}

func strPtr(s string) *string { return &s }

func TestSignalCardBuilder_TemplateResolution(t *testing.T) {
	builder := newTestSignalCardBuilder()
	event := &models.ClinicalSignalEvent{
		EventID:   "evt-001",
		PatientID: "00000000-0000-0000-0000-000000000001",
		NodeID:    "PM-03",
		SignalType: "MONITORING_CLASSIFICATION",
		Classification: &models.ClassificationResult{
			Category:        "REVERSE_DIPPER",
			DataSufficiency: "SUFFICIENT",
		},
	}

	card, err := builder.Build(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card == nil {
		t.Fatal("expected card, got nil")
	}
	if card.TemplateID != "dc-pm03-reverse-dipper-v1" {
		t.Errorf("expected template dc-pm03-reverse-dipper-v1, got %s", card.TemplateID)
	}
}

func TestSignalCardBuilder_PM09Template(t *testing.T) {
	builder := newTestSignalCardBuilder()
	event := &models.ClinicalSignalEvent{
		EventID:   "evt-002",
		PatientID: "00000000-0000-0000-0000-000000000002",
		NodeID:    "PM-09",
		SignalType: "MONITORING_CLASSIFICATION",
		Classification: &models.ClassificationResult{
			Category:        "SEVERELY_DISRUPTED",
			DataSufficiency: "SUFFICIENT",
		},
	}

	card, err := builder.Build(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card == nil {
		t.Fatal("expected card, got nil")
	}
	if card.TemplateID != "dc-pm09-severely-disrupted-v1" {
		t.Errorf("expected template dc-pm09-severely-disrupted-v1, got %s", card.TemplateID)
	}
}

func TestSignalCardBuilder_NoTemplate(t *testing.T) {
	builder := newTestSignalCardBuilder()
	event := &models.ClinicalSignalEvent{
		EventID:   "evt-003",
		PatientID: "00000000-0000-0000-0000-000000000003",
		NodeID:    "PM-01",
		SignalType: "MONITORING_CLASSIFICATION",
		Classification: &models.ClassificationResult{
			Category:        "AT_TARGET",
			DataSufficiency: "SUFFICIENT",
		},
	}

	card, err := builder.Build(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card != nil {
		t.Errorf("expected nil card for AT_TARGET, got card with template %s", card.TemplateID)
	}
}

func TestSignalCardBuilder_ConfidenceTier(t *testing.T) {
	builder := newTestSignalCardBuilder()

	tests := []struct {
		name           string
		sufficiency    string
		severity       string
		gateSuggestion *string
		expectedTier   models.ConfidenceTier
	}{
		{
			name:         "SUFFICIENT + CRITICAL = FIRM",
			sufficiency:  "SUFFICIENT",
			severity:     "CRITICAL",
			expectedTier: models.TierFirm,
		},
		{
			name:           "PARTIAL + MODERATE (via gate) = POSSIBLE",
			sufficiency:    "PARTIAL",
			severity:       "",
			gateSuggestion: strPtr("MODIFY"),
			expectedTier:   models.TierPossible,
		},
		{
			name:         "SUFFICIENT + no severity = PROBABLE",
			sufficiency:  "SUFFICIENT",
			severity:     "",
			expectedTier: models.TierProbable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &models.ClinicalSignalEvent{
				EventID:           "evt-tier",
				PatientID:         "00000000-0000-0000-0000-000000000004",
				NodeID:            "MD-01",
				SignalType:        "DETERIORATION_SIGNAL",
				MCUGateSuggestion: tt.gateSuggestion,
				DeteriorationSignal: &models.DeteriorationResult{
					Signal:   "TEST_SIGNAL",
					Severity: tt.severity,
				},
				Classification: &models.ClassificationResult{
					DataSufficiency: tt.sufficiency,
				},
			}

			card, err := builder.Build(context.Background(), event)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if card == nil {
				t.Fatal("expected card, got nil")
			}
			if card.DiagnosticConfidenceTier != tt.expectedTier {
				t.Errorf("expected tier %s, got %s", tt.expectedTier, card.DiagnosticConfidenceTier)
			}
		})
	}
}

func TestSignalCardBuilder_HALTGate(t *testing.T) {
	builder := newTestSignalCardBuilder()
	haltGate := "HALT"
	event := &models.ClinicalSignalEvent{
		EventID:           "evt-halt",
		PatientID:         "00000000-0000-0000-0000-000000000005",
		NodeID:            "MD-06",
		SignalType:        "DETERIORATION_SIGNAL",
		MCUGateSuggestion: &haltGate,
		DeteriorationSignal: &models.DeteriorationResult{
			Signal:   "CRITICAL_DECLINE",
			Severity: "CRITICAL",
		},
	}

	card, err := builder.Build(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card == nil {
		t.Fatal("expected card, got nil")
	}
	if card.MCUGate != models.GateHalt {
		t.Errorf("expected gate HALT, got %s", card.MCUGate)
	}
	if !card.PendingReaffirmation {
		t.Error("expected PendingReaffirmation=true for HALT gate")
	}
}

func TestSignalCardBuilder_MDTemplate(t *testing.T) {
	builder := newTestSignalCardBuilder()
	event := &models.ClinicalSignalEvent{
		EventID:   "evt-md01",
		PatientID: "00000000-0000-0000-0000-000000000006",
		NodeID:    "MD-01",
		SignalType: "DETERIORATION_SIGNAL",
		DeteriorationSignal: &models.DeteriorationResult{
			Signal:   "IS_CRITICAL_DECLINE",
			Severity: "CRITICAL",
		},
	}

	card, err := builder.Build(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card == nil {
		t.Fatal("expected card, got nil")
	}
	if card.TemplateID != "dc-md01-is-critical-decline-v1" {
		t.Errorf("expected template dc-md01-is-critical-decline-v1, got %s", card.TemplateID)
	}
}

func TestSignalCardBuilder_CardSource(t *testing.T) {
	builder := newTestSignalCardBuilder()
	event := &models.ClinicalSignalEvent{
		EventID:   "evt-src",
		PatientID: "00000000-0000-0000-0000-000000000007",
		NodeID:    "PM-03",
		SignalType: "MONITORING_CLASSIFICATION",
		Classification: &models.ClassificationResult{
			Category:        "REVERSE_DIPPER",
			DataSufficiency: "SUFFICIENT",
		},
	}

	card, err := builder.Build(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card == nil {
		t.Fatal("expected card, got nil")
	}
	if card.CardSource != models.SourceClinicalSignal {
		t.Errorf("expected source CLINICAL_SIGNAL, got %s", card.CardSource)
	}
}
