package negative_evidence_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cardiofit/kb32/internal/generator"
	"github.com/cardiofit/kb32/internal/negative_evidence"
	"github.com/cardiofit/kb32/internal/reasoning"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// DefaultAbsenceQueryForStopRule tests
// ---------------------------------------------------------------------------

func TestDefaultAbsenceQueryForStopRule_PostFall(t *testing.T) {
	residentID := uuid.New()
	q := negative_evidence.DefaultAbsenceQueryForStopRule("PostFall", residentID)

	if q.Pattern != negative_evidence.PatternBoundedWindow {
		t.Errorf("PostFall: expected PatternBoundedWindow, got %s", q.Pattern)
	}
	if q.ObservationKind != "fall" {
		t.Errorf("PostFall: ObservationKind = %q, want %q", q.ObservationKind, "fall")
	}
	if q.WindowDays != 90 {
		t.Errorf("PostFall: WindowDays = %d, want 90", q.WindowDays)
	}
	if q.ResidentID != residentID {
		t.Errorf("PostFall: ResidentID mismatch")
	}
}

func TestDefaultAbsenceQueryForStopRule_PPI(t *testing.T) {
	residentID := uuid.New()
	q := negative_evidence.DefaultAbsenceQueryForStopRule("PPI", residentID)

	if q.Pattern != negative_evidence.PatternIndicationDocumentation {
		t.Errorf("PPI: expected PatternIndicationDocumentation, got %s", q.Pattern)
	}
	if q.ObservationKind != "ppi_indication" {
		t.Errorf("PPI: ObservationKind = %q, want %q", q.ObservationKind, "ppi_indication")
	}
	if q.ResidentID != residentID {
		t.Errorf("PPI: ResidentID mismatch")
	}
}

func TestDefaultAbsenceQueryForStopRule_BenzodiazepineLongTerm(t *testing.T) {
	residentID := uuid.New()
	q := negative_evidence.DefaultAbsenceQueryForStopRule("BenzodiazepineLongTerm", residentID)

	if q.Pattern != negative_evidence.PatternPeriodicReview {
		t.Errorf("BenzodiazepineLongTerm: expected PatternPeriodicReview, got %s", q.Pattern)
	}
	if q.ObservationKind != "benzodiazepine_review" {
		t.Errorf("BenzodiazepineLongTerm: ObservationKind = %q, want %q", q.ObservationKind, "benzodiazepine_review")
	}
	if q.WindowDays != 365 {
		t.Errorf("BenzodiazepineLongTerm: WindowDays = %d, want 365", q.WindowDays)
	}
	if q.ResidentID != residentID {
		t.Errorf("BenzodiazepineLongTerm: ResidentID mismatch")
	}
}

func TestDefaultAbsenceQueryForStopRule_UnknownFallback(t *testing.T) {
	residentID := uuid.New()
	q := negative_evidence.DefaultAbsenceQueryForStopRule("SomeUnknownRuleXYZ", residentID)

	if q.Pattern != negative_evidence.PatternBoundedWindow {
		t.Errorf("fallback: expected PatternBoundedWindow, got %s", q.Pattern)
	}
	if q.ObservationKind != "general_observation" {
		t.Errorf("fallback: ObservationKind = %q, want %q", q.ObservationKind, "general_observation")
	}
	if q.WindowDays != 90 {
		t.Errorf("fallback: WindowDays = %d, want 90", q.WindowDays)
	}
}

// ---------------------------------------------------------------------------
// AttachNegativeEvidence tests
// ---------------------------------------------------------------------------

func makeSTOPPacket(ruleID string) *generator.Packet {
	residentID := uuid.New()
	return &generator.Packet{
		RecommendationID: uuid.New(),
		AuthorID:         uuid.New(),
		Type:             "STOP",
		SnapshotRef:      residentID,
		Sections: map[string]string{
			"issue":            "test issue",
			"clinical_context": "test context",
			"rationale":        "N/A",
			"evidence":         "N/A",
			"proposed_plan":    "N/A",
			"monitoring":       "N/A",
			"urgency":          "high",
		},
		AppliedRule: reasoning.ApplicableRule{
			RuleID:  ruleID,
			Type:    "STOP",
			Urgency: "high",
		},
	}
}

func TestAttachNegativeEvidence_STOPWithAbsenceConfirmed(t *testing.T) {
	pkt := makeSTOPPacket("PostFall")
	querier := negative_evidence.NewInMemoryQuerier(nil) // absence confirmed

	err := negative_evidence.AttachNegativeEvidence(context.Background(), querier, pkt, pkt.SnapshotRef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	evidenceSection := pkt.Sections["evidence"]
	// Should contain absence evidence text (non-empty, not just "N/A").
	if evidenceSection == "N/A" {
		t.Error("evidence section still N/A after AttachNegativeEvidence with confirmed absence")
	}
	if evidenceSection == "" {
		t.Error("evidence section is empty")
	}
}

func TestAttachNegativeEvidence_STOPWithPresenceDetected(t *testing.T) {
	pkt := makeSTOPPacket("PostFall")
	lastSeen := time.Now().UTC().Add(-48 * time.Hour)
	querier := negative_evidence.NewInMemoryQuerier(&lastSeen) // presence detected

	original := pkt.Sections["evidence"]
	err := negative_evidence.AttachNegativeEvidence(context.Background(), querier, pkt, pkt.SnapshotRef)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Presence negates the claim — section MUST NOT contain "No record" absence text.
	evidenceSection := pkt.Sections["evidence"]
	if strings.Contains(evidenceSection, "No record") {
		t.Errorf("evidence section contains absence text despite presence detected: %s", evidenceSection)
	}
	// The section should remain unchanged (original N/A) when presence is detected.
	if evidenceSection != original {
		t.Errorf("evidence section modified despite presence detection: got %q, want %q", evidenceSection, original)
	}
}

func TestAttachNegativeEvidence_NonSTOP_PacketUnchanged(t *testing.T) {
	residentID := uuid.New()
	for _, packetType := range []string{"MONITOR", "DOSE_CHANGE", "ADD"} {
		pkt := &generator.Packet{
			RecommendationID: uuid.New(),
			Type:             packetType,
			SnapshotRef:      residentID,
			Sections: map[string]string{
				"evidence": "N/A",
			},
		}
		querier := negative_evidence.NewInMemoryQuerier(nil)

		err := negative_evidence.AttachNegativeEvidence(context.Background(), querier, pkt, residentID)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", packetType, err)
		}
		if pkt.Sections["evidence"] != "N/A" {
			t.Errorf("%s: evidence section changed for non-STOP packet", packetType)
		}
	}
}

func TestAttachNegativeEvidence_QuerierError_Propagated(t *testing.T) {
	pkt := makeSTOPPacket("PostFall")
	sentinel := errors.New("db down")
	querier := negative_evidence.NewInMemoryQuerierWithError(sentinel)

	err := negative_evidence.AttachNegativeEvidence(context.Background(), querier, pkt, pkt.SnapshotRef)
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error propagated, got %v", err)
	}
}
