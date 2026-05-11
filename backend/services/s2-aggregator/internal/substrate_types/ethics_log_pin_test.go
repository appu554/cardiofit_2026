package substrate_types

import (
	"reflect"
	"testing"
)

// TestEthicsLogEntryFieldPinning pins the field names of EthicsLogEntry
// against the canonical Phase 1c ethics_log.Entry. Drift in either
// direction fails the build so the audit emitter cannot silently produce
// rows the shared store rejects.
//
// SOURCE OF TRUTH: shared/v2_substrate/ethics/ethics_log/logger.go (Entry).
func TestEthicsLogEntryFieldPinning(t *testing.T) {
	want := []string{
		"ID", "DecisionID", "EntryType", "Severity",
		"Description", "Reviewer", "ReviewOutcome",
		"RemediationActions", "Status", "CreatedAt", "UpdatedAt",
	}
	got := fieldNames(EthicsLogEntry{})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("EthicsLogEntry fields drifted: want %v got %v\n"+
			"if canonical Entry changed, update local copy + SOURCE OF TRUTH comment",
			want, got)
	}
}

// TestEthicsLogEntryTypeValues pins the five canonical EntryType strings.
// These string values are the on-disk persisted form via the shared
// Store.Append path; any drift would mean s2-aggregator's audit rows
// would be rejected (or worse, accepted but mis-categorised) downstream.
func TestEthicsLogEntryTypeValues(t *testing.T) {
	want := map[EthicsLogEntryType]string{
		EthicsEntryTypeDecision:        "decision",
		EthicsEntryTypeConcernFlagged:  "concern_flagged",
		EthicsEntryTypeReviewRequested: "review_requested",
		EthicsEntryTypePatternDetected: "pattern_detected",
		EthicsEntryTypeIncident:        "incident",
	}
	for k, v := range want {
		if string(k) != v {
			t.Fatalf("EthicsLogEntryType %q drifted: want %q got %q", v, v, string(k))
		}
	}
}

// TestEthicsSeverityScale pins the 1..5 severity scale and confirms that
// severity=1 is Primary Decision (the value kb-32 Stage 7 emits per
// Phase 2-completion Task 4; the s2-aggregator audit package uses the
// same convention for primary pharmacist actions / cognitive escalation).
func TestEthicsSeverityScale(t *testing.T) {
	if EthicsSeverityPrimaryDecision != 1 {
		t.Errorf("EthicsSeverityPrimaryDecision = %d, want 1", EthicsSeverityPrimaryDecision)
	}
	if EthicsSeverityCritical != 5 {
		t.Errorf("EthicsSeverityCritical = %d, want 5", EthicsSeverityCritical)
	}
	if EthicsSeverityLow != 2 || EthicsSeverityModerate != 3 || EthicsSeverityHigh != 4 {
		t.Errorf("severity scale midpoints drifted: %d %d %d (want 2 3 4)",
			EthicsSeverityLow, EthicsSeverityModerate, EthicsSeverityHigh)
	}
}
