package services

import (
	"testing"
	"time"
)

func TestLedger_AppendAndVerifyChain(t *testing.T) {
	ledger := NewInMemoryLedger([]byte("sprint1-test-hmac-key"))
	_, err := ledger.AppendEntry("ATTRIBUTION_RUN", "subject-001", `{"verdict":"prevented"}`)
	if err != nil {
		t.Fatalf("append 1 failed: %v", err)
	}
	_, err = ledger.AppendEntry("ATTRIBUTION_RUN", "subject-002", `{"verdict":"no_effect_detected"}`)
	if err != nil {
		t.Fatalf("append 2 failed: %v", err)
	}
	_, err = ledger.AppendEntry("MODEL_PROMOTION", "gap20-heuristic-v2", `{"from":"v1","to":"v2"}`)
	if err != nil {
		t.Fatalf("append 3 failed: %v", err)
	}

	ok, idx, err := ledger.VerifyChain()
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected chain valid, got first-broken-index=%d", idx)
	}
}

func TestLedger_EmptyChain_IsValid(t *testing.T) {
	ledger := NewInMemoryLedger([]byte("sprint1-test-hmac-key"))
	ok, idx, err := ledger.VerifyChain()
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !ok {
		t.Fatalf("expected empty chain to verify as valid, got first-broken-index=%d", idx)
	}
	if idx != -1 {
		t.Fatalf("expected broken index -1 for empty chain, got %d", idx)
	}

	entries, snapValid, snapIdx := ledger.Snapshot()
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries from snapshot, got %d", len(entries))
	}
	if !snapValid {
		t.Fatalf("expected snapshot of empty chain to be valid")
	}
	if snapIdx != -1 {
		t.Fatalf("expected snapshot broken index -1, got %d", snapIdx)
	}
}

func TestLedger_TamperedEntry_VerifyFails(t *testing.T) {
	ledger := NewInMemoryLedger([]byte("sprint1-test-hmac-key"))
	_, _ = ledger.AppendEntry("ATTRIBUTION_RUN", "s1", `{"a":1}`)
	_, _ = ledger.AppendEntry("ATTRIBUTION_RUN", "s2", `{"a":2}`)
	_, _ = ledger.AppendEntry("ATTRIBUTION_RUN", "s3", `{"a":3}`)

	// Tamper with entry 1's payload in place (simulates post-hoc edit).
	ledger.tamperForTest(1, `{"a":2,"tampered":true}`)

	ok, idx, err := ledger.VerifyChain()
	if err != nil {
		t.Fatalf("verify should not error, got %v", err)
	}
	if ok {
		t.Fatalf("expected tampered chain to be invalid")
	}
	if idx < 1 {
		t.Fatalf("expected break at or after index 1, got %d", idx)
	}
}

func TestLedger_LengthPrefixPreventsFieldCollision(t *testing.T) {
	// Two field-split combinations that under the old "|"-separator scheme
	// would produce identical hash inputs:
	//   split A: entryType="A", subjectID="B|C", payload="D"
	//   split B: entryType="A|B", subjectID="C", payload="D"
	// The pre-fix `|`-joined string is identical: "A|B|C|D".
	// With length-prefixing each field's byte count differs, so the hashes
	// must diverge.
	//
	// This test calls the unexported computeEntryHash directly with fixed
	// inputs (same key, same prior, same seq, same timestamp) because
	// AppendEntry stamps a fresh time.Now() per call and would otherwise
	// mask the collision behind timestamp differences.
	ledger := NewInMemoryLedger([]byte("test-key"))
	fixedTS := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)

	hashA := ledger.computeEntryHash(genesisHash, "A", "B|C", "D", 0, fixedTS)
	hashB := ledger.computeEntryHash(genesisHash, "A|B", "C", "D", 0, fixedTS)

	if hashA == hashB {
		t.Fatalf("hash collision between different field splits under length-prefix scheme: got identical hash %s", hashA)
	}
}

func TestLedger_SeedSequence_Idempotent(t *testing.T) {
	ledger := NewInMemoryLedger([]byte("test-key"))
	ledger.SeedSequence(100)
	// Second seed with a DIFFERENT starting sequence must be a no-op.
	ledger.SeedSequence(500)

	e, err := ledger.AppendEntry("T", "S", "P")
	if err != nil {
		t.Fatalf("append failed: %v", err)
	}
	if e.Sequence != 100 {
		t.Fatalf("expected sequence=100 (first seed preserved), got %d", e.Sequence)
	}
}
