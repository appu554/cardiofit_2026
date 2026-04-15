package services

import (
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-patient-profile/internal/models"
)

// setupSafetyEventTestDB creates an in-memory sqlite database with
// just the safety_events table. Uses raw DDL (not GORM AutoMigrate)
// because sqlite does not understand the Postgres-specific
// gen_random_uuid() default on the production model. The column
// set mirrors models.SafetyEvent exactly so GORM reads work —
// any field addition requires updating this DDL.
func setupSafetyEventTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	err = db.Exec(`
		CREATE TABLE safety_events (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			severity TEXT,
			description TEXT,
			lab_type TEXT,
			old_value TEXT,
			new_value TEXT,
			observed_at DATETIME NOT NULL,
			created_at DATETIME
		)
	`).Error
	if err != nil {
		t.Fatalf("create safety_events table: %v", err)
	}
	return db
}

// TestSafetyEventRecorder_Record_WritesRow verifies the basic write
// path: a single Record call persists a row with all fields set.
func TestSafetyEventRecorder_Record_WritesRow(t *testing.T) {
	db := setupSafetyEventTestDB(t)
	r := NewSafetyEventRecorder(db, zap.NewNop())

	observedAt := time.Now().UTC().Add(-2 * time.Hour)
	err := r.Record(
		"p-1",
		models.SafetyEventAcuteIllness,
		"MODERATE",
		"patient presented with acute febrile illness",
		observedAt,
	)
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	var count int64
	db.Model(&models.SafetyEvent{}).Where("patient_id = ?", "p-1").Count(&count)
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}

	var row models.SafetyEvent
	if err := db.Where("patient_id = ?", "p-1").First(&row).Error; err != nil {
		t.Fatalf("fetch row: %v", err)
	}
	if row.EventType != models.SafetyEventAcuteIllness {
		t.Errorf("EventType = %q, want ACUTE_ILLNESS", row.EventType)
	}
	if row.Severity != "MODERATE" {
		t.Errorf("Severity = %q, want MODERATE", row.Severity)
	}
	if !row.ObservedAt.Equal(observedAt) {
		t.Errorf("ObservedAt = %v, want %v", row.ObservedAt, observedAt)
	}
}

// TestSafetyEventRecorder_RecordLabEvent_CarriesLabFields verifies
// that the lab-event variant also populates LabType / OldValue /
// NewValue. Used by the lab_service safety-alert publish paths.
func TestSafetyEventRecorder_RecordLabEvent_CarriesLabFields(t *testing.T) {
	db := setupSafetyEventTestDB(t)
	r := NewSafetyEventRecorder(db, zap.NewNop())

	err := r.RecordLabEvent(
		"p-lab",
		models.SafetyEventEGFRCritical,
		"CRITICAL",
		"eGFR dropped below 15",
		"EGFR",
		"32.5",
		"12.8",
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("RecordLabEvent: %v", err)
	}

	var row models.SafetyEvent
	if err := db.Where("patient_id = ?", "p-lab").First(&row).Error; err != nil {
		t.Fatalf("fetch row: %v", err)
	}
	if row.LabType != "EGFR" {
		t.Errorf("LabType = %q, want EGFR", row.LabType)
	}
	if row.OldValue != "32.5" {
		t.Errorf("OldValue = %q, want 32.5", row.OldValue)
	}
	if row.NewValue != "12.8" {
		t.Errorf("NewValue = %q, want 12.8", row.NewValue)
	}
}

// TestSafetyEventRecorder_ConfounderFlags_AcuteIllness_InsideWindow
// verifies that an ACUTE_ILLNESS event observed 3 days ago trips
// the IsAcuteIll flag (7-day window).
func TestSafetyEventRecorder_ConfounderFlags_AcuteIllness_InsideWindow(t *testing.T) {
	db := setupSafetyEventTestDB(t)
	r := NewSafetyEventRecorder(db, zap.NewNop())

	_ = r.Record("p-acute", models.SafetyEventAcuteIllness, "SEVERE", "sepsis",
		time.Now().UTC().Add(-3*24*time.Hour))

	isAcute, hasTransfusion, hasHypo := r.ConfounderFlags("p-acute", time.Now().UTC())
	if !isAcute {
		t.Error("IsAcuteIll = false, want true (event 3d old, window=7d)")
	}
	if hasTransfusion {
		t.Error("HasRecentTransfusion = true, want false (no transfusion event)")
	}
	if hasHypo {
		t.Error("HasRecentHypoglycaemia = true, want false (no hypo event)")
	}
}

// TestSafetyEventRecorder_ConfounderFlags_AcuteIllness_OutsideWindow
// verifies that an ACUTE_ILLNESS event observed 10 days ago does
// NOT trip the IsAcuteIll flag (outside the 7-day window). This is
// the clinical recovery semantics: after 7 days the patient is
// considered convalescent and the glycaemic intensification pause
// lifts.
func TestSafetyEventRecorder_ConfounderFlags_AcuteIllness_OutsideWindow(t *testing.T) {
	db := setupSafetyEventTestDB(t)
	r := NewSafetyEventRecorder(db, zap.NewNop())

	_ = r.Record("p-recovered", models.SafetyEventAcuteIllness, "MODERATE", "past illness",
		time.Now().UTC().Add(-10*24*time.Hour))

	isAcute, _, _ := r.ConfounderFlags("p-recovered", time.Now().UTC())
	if isAcute {
		t.Error("IsAcuteIll = true, want false (event 10d old, window=7d)")
	}
}

// TestSafetyEventRecorder_ConfounderFlags_Transfusion90DayWindow
// verifies the 90-day transfusion window. A transfusion 60 days ago
// should still trip the flag (HbA1c has not yet renormalised), but
// one 120 days ago should not.
func TestSafetyEventRecorder_ConfounderFlags_Transfusion90DayWindow(t *testing.T) {
	db := setupSafetyEventTestDB(t)
	r := NewSafetyEventRecorder(db, zap.NewNop())

	_ = r.Record("p-recent-txn", models.SafetyEventBloodTransfusion, "ROUTINE", "2U PRBCs",
		time.Now().UTC().Add(-60*24*time.Hour))
	_ = r.Record("p-old-txn", models.SafetyEventBloodTransfusion, "ROUTINE", "4U PRBCs",
		time.Now().UTC().Add(-120*24*time.Hour))

	_, recentTxn, _ := r.ConfounderFlags("p-recent-txn", time.Now().UTC())
	if !recentTxn {
		t.Error("p-recent-txn HasRecentTransfusion = false, want true (60d old, window=90d)")
	}

	_, oldTxn, _ := r.ConfounderFlags("p-old-txn", time.Now().UTC())
	if oldTxn {
		t.Error("p-old-txn HasRecentTransfusion = true, want false (120d old, window=90d)")
	}
}

// TestSafetyEventRecorder_ConfounderFlags_Hypoglycaemia30DayWindow
// verifies the 30-day hypoglycaemia window. A hypo 10 days ago
// trips the flag; one 60 days ago does not.
func TestSafetyEventRecorder_ConfounderFlags_Hypoglycaemia30DayWindow(t *testing.T) {
	db := setupSafetyEventTestDB(t)
	r := NewSafetyEventRecorder(db, zap.NewNop())

	_ = r.Record("p-recent-hypo", models.SafetyEventHypoEvent, "SEVERE", "ER visit for hypo",
		time.Now().UTC().Add(-10*24*time.Hour))
	_ = r.Record("p-old-hypo", models.SafetyEventHypoEvent, "MODERATE", "nocturnal hypo",
		time.Now().UTC().Add(-60*24*time.Hour))

	_, _, recentHypo := r.ConfounderFlags("p-recent-hypo", time.Now().UTC())
	if !recentHypo {
		t.Error("p-recent-hypo HasRecentHypoglycaemia = false, want true (10d old, window=30d)")
	}

	_, _, oldHypo := r.ConfounderFlags("p-old-hypo", time.Now().UTC())
	if oldHypo {
		t.Error("p-old-hypo HasRecentHypoglycaemia = true, want false (60d old, window=30d)")
	}
}

// TestSafetyEventRecorder_ConfounderFlags_MultipleFlagsOnSamePatient
// verifies that a patient with all three event types in their
// relevant windows lights up all three flags. This is the
// worst-case confounder scenario — a recently-hospitalised patient
// who received a transfusion and had a hypo event. Every MCU gate
// rule that reads these flags should fire for this patient.
func TestSafetyEventRecorder_ConfounderFlags_MultipleFlagsOnSamePatient(t *testing.T) {
	db := setupSafetyEventTestDB(t)
	r := NewSafetyEventRecorder(db, zap.NewNop())

	_ = r.Record("p-complex", models.SafetyEventAcuteIllness, "SEVERE", "hospitalisation",
		time.Now().UTC().Add(-2*24*time.Hour))
	_ = r.Record("p-complex", models.SafetyEventBloodTransfusion, "ROUTINE", "intraop 2U",
		time.Now().UTC().Add(-1*24*time.Hour))
	_ = r.Record("p-complex", models.SafetyEventHypoEvent, "SEVERE", "post-op hypo",
		time.Now().UTC().Add(-5*24*time.Hour))

	isAcute, hasTxn, hasHypo := r.ConfounderFlags("p-complex", time.Now().UTC())
	if !isAcute {
		t.Error("IsAcuteIll = false, want true")
	}
	if !hasTxn {
		t.Error("HasRecentTransfusion = false, want true")
	}
	if !hasHypo {
		t.Error("HasRecentHypoglycaemia = false, want true")
	}
}

// TestSafetyEventRecorder_ConfounderFlags_EmptyPatient verifies a
// patient with no events gets all three flags as false.
func TestSafetyEventRecorder_ConfounderFlags_EmptyPatient(t *testing.T) {
	db := setupSafetyEventTestDB(t)
	r := NewSafetyEventRecorder(db, zap.NewNop())

	isAcute, hasTxn, hasHypo := r.ConfounderFlags("p-clean", time.Now().UTC())
	if isAcute || hasTxn || hasHypo {
		t.Errorf("expected all flags false for clean patient, got (%v, %v, %v)",
			isAcute, hasTxn, hasHypo)
	}
}

// TestSafetyEventRecorder_NilRecorder_DegradesGracefully asserts
// that a nil recorder (or nil db) does not panic and returns
// all-false from ConfounderFlags. This is the production
// fallback path when the recorder is not wired.
func TestSafetyEventRecorder_NilRecorder_DegradesGracefully(t *testing.T) {
	var r *SafetyEventRecorder
	if err := r.Record("p", "ACUTE_ILLNESS", "", "", time.Now()); err != nil {
		t.Errorf("nil recorder Record should return nil, got %v", err)
	}
	isAcute, hasTxn, hasHypo := r.ConfounderFlags("p", time.Now())
	if isAcute || hasTxn || hasHypo {
		t.Error("nil recorder should return all-false flags")
	}
}

// TestSafetyEventRecorder_PatientIsolation verifies that events
// for one patient do not influence another patient's flags.
// Fundamental correctness check — a hypo on Patient A must not
// trip HasRecentHypoglycaemia on Patient B.
func TestSafetyEventRecorder_PatientIsolation(t *testing.T) {
	db := setupSafetyEventTestDB(t)
	r := NewSafetyEventRecorder(db, zap.NewNop())

	_ = r.Record("patient-a", models.SafetyEventHypoEvent, "SEVERE", "",
		time.Now().UTC().Add(-5*24*time.Hour))

	_, _, hypoA := r.ConfounderFlags("patient-a", time.Now().UTC())
	_, _, hypoB := r.ConfounderFlags("patient-b", time.Now().UTC())

	if !hypoA {
		t.Error("patient-a should have HasRecentHypoglycaemia = true")
	}
	if hypoB {
		t.Error("patient-b should have HasRecentHypoglycaemia = false (no events)")
	}
}
