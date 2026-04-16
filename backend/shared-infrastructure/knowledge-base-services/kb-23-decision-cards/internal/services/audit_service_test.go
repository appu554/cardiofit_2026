package services

import (
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	err = db.Exec(`
		CREATE TABLE clinical_audit_log (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			service_source TEXT NOT NULL,
			payload TEXT NOT NULL,
			previous_hash TEXT NOT NULL,
			hash TEXT NOT NULL,
			occurred_at DATETIME NOT NULL,
			created_at DATETIME
		)
	`).Error
	if err != nil {
		t.Fatalf("create clinical_audit_log: %v", err)
	}
	return db
}

func TestAuditService_Append_CreatesHashChain(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db, zap.NewNop())

	now := time.Now().UTC()
	_ = svc.Append("p1", "CARD_GENERATED", "kb23", map[string]string{"card_id": "c1"}, now)
	_ = svc.Append("p1", "CARD_GENERATED", "kb23", map[string]string{"card_id": "c2"}, now.Add(time.Hour))

	entries, err := svc.FetchPatientTrail("p1", now.Add(-time.Hour), now.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("FetchPatientTrail: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// First entry should have GENESIS as previous hash
	if entries[0].PreviousHash != "GENESIS" {
		t.Errorf("first entry PreviousHash = %q, want GENESIS", entries[0].PreviousHash)
	}
	// Second entry should chain to first
	if entries[1].PreviousHash != entries[0].Hash {
		t.Error("second entry PreviousHash should equal first entry Hash (chain broken)")
	}
	// Both hashes should be non-empty 64-char hex
	if len(entries[0].Hash) != 64 {
		t.Errorf("hash length = %d, want 64 (SHA-256 hex)", len(entries[0].Hash))
	}
}

func TestAuditService_VerifyChain_IntactChain(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db, zap.NewNop())

	now := time.Now().UTC()
	_ = svc.Append("p-intact", "EVENT_A", "kb23", "data-a", now)
	_ = svc.Append("p-intact", "EVENT_B", "kb23", "data-b", now.Add(time.Minute))
	_ = svc.Append("p-intact", "EVENT_C", "kb23", "data-c", now.Add(2*time.Minute))

	idx, err := svc.VerifyChain("p-intact")
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if idx != -1 {
		t.Errorf("expected intact chain (-1), got broken at index %d", idx)
	}
}

func TestAuditService_VerifyChain_DetectsTampering(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db, zap.NewNop())

	now := time.Now().UTC()
	_ = svc.Append("p-tamper", "EVENT_A", "kb23", "original", now)
	_ = svc.Append("p-tamper", "EVENT_B", "kb23", "data-b", now.Add(time.Minute))

	// Tamper with the first entry's payload
	db.Exec("UPDATE clinical_audit_log SET payload = 'TAMPERED' WHERE patient_id = 'p-tamper' AND event_type = 'EVENT_A'")

	idx, err := svc.VerifyChain("p-tamper")
	if err != nil {
		t.Fatalf("VerifyChain: %v", err)
	}
	if idx == -1 {
		t.Error("expected broken chain after tampering, got intact (-1)")
	}
	if idx != 0 {
		t.Errorf("expected break at index 0 (tampered entry), got %d", idx)
	}
}

func TestAuditService_PatientIsolation(t *testing.T) {
	db := setupAuditTestDB(t)
	svc := NewAuditService(db, zap.NewNop())

	now := time.Now().UTC()
	_ = svc.Append("patient-a", "EVENT", "kb23", "a-data", now)
	_ = svc.Append("patient-b", "EVENT", "kb23", "b-data", now)

	// Both patients should have independent GENESIS chains
	entriesA, _ := svc.FetchPatientTrail("patient-a", now.Add(-time.Hour), now.Add(time.Hour))
	entriesB, _ := svc.FetchPatientTrail("patient-b", now.Add(-time.Hour), now.Add(time.Hour))

	if len(entriesA) != 1 || len(entriesB) != 1 {
		t.Fatalf("expected 1 entry each, got A=%d B=%d", len(entriesA), len(entriesB))
	}
	if entriesA[0].PreviousHash != "GENESIS" {
		t.Error("patient-a should have its own GENESIS chain")
	}
	if entriesB[0].PreviousHash != "GENESIS" {
		t.Error("patient-b should have its own GENESIS chain")
	}
}

func TestAuditService_NilService_Degrades(t *testing.T) {
	var svc *AuditService
	if err := svc.Append("p", "E", "s", "d", time.Now()); err != nil {
		t.Errorf("nil service Append should return nil, got %v", err)
	}
	idx, err := svc.VerifyChain("p")
	if err != nil || idx != -1 {
		t.Errorf("nil service VerifyChain should return (-1, nil)")
	}
}
