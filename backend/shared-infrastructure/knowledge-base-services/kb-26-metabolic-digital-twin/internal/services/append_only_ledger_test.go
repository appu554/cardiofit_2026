package services

import (
	"testing"
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
	// Two ledger entries with different field splits that, under the old
	// "|"-separator scheme, could produce identical hash inputs. With
	// length-prefixing, they must produce different hashes.
	ledger := NewInMemoryLedger([]byte("test-key"))

	// Entry A: entryType="A", subjectID="B|C", payload="D"
	e1, err := ledger.AppendEntry("A", "B|C", "D")
	if err != nil {
		t.Fatalf("append A failed: %v", err)
	}

	// Entry B (on a fresh ledger so prior_hash is the same genesis):
	ledger2 := NewInMemoryLedger([]byte("test-key"))
	// entryType="A|B", subjectID="C", payload="D" — same "|"-joined string
	// would collide.
	e2, err := ledger2.AppendEntry("A|B", "C", "D")
	if err != nil {
		t.Fatalf("append B failed: %v", err)
	}

	if e1.EntryHash == e2.EntryHash {
		t.Fatalf("hash collision between different field splits — length-prefixing not applied")
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
