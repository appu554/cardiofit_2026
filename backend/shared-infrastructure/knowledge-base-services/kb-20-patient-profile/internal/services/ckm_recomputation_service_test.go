package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-patient-profile/internal/models"
)

// setupCKMRecomputationTestDB creates an in-memory sqlite database with
// the minimal schema needed to exercise CKMRecomputationService:
// patient_profiles (for stage reads/writes) and lab_entries (for latest
// LVEF / NT-proBNP / CAC lookups). We use raw DDL because the production
// schema carries PostgreSQL-specific defaults that sqlite cannot parse.
func setupCKMRecomputationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE patient_profiles (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL UNIQUE,
			age INTEGER NOT NULL,
			sex TEXT NOT NULL,
			weight_kg TEXT,
			height_cm TEXT,
			bmi TEXT,
			smoking_status TEXT,
			dm_type TEXT,
			dm_duration_years TEXT,
			comorbidities TEXT,
			htn_status TEXT,
			ckm_stage INTEGER DEFAULT 0,
			ckm_stage_v2 TEXT,
			hba1c TEXT,
			egfr TEXT,
			uacr TEXT,
			last_medication_change_at DATETIME,
			last_medication_change_class TEXT,
			active INTEGER DEFAULT 1,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error
	if err != nil {
		t.Fatalf("create patient_profiles table: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE lab_entries (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL,
			lab_type TEXT NOT NULL,
			value TEXT NOT NULL,
			unit TEXT NOT NULL,
			measured_at DATETIME NOT NULL,
			source TEXT,
			is_derived INTEGER DEFAULT 0,
			validation_status TEXT NOT NULL DEFAULT 'ACCEPTED',
			flag_reason TEXT,
			loinc_code TEXT,
			fhir_observation_id TEXT,
			created_at DATETIME
		)
	`).Error
	if err != nil {
		t.Fatalf("create lab_entries table: %v", err)
	}
	return db
}

// seedCKMPatient inserts a PatientProfile row with the given identifiers
// and starting CKM stage. Other fields default to values that produce a
// Stage 0 classifier result unless a test explicitly seeds labs/comorbidities.
// EGFR defaults to 95 mL/min/1.73m² (healthy) so tests that care only about
// LVEF/CAC don't accidentally trip CKD-driven Stage 3 classification.
func seedCKMPatient(t *testing.T, db *gorm.DB, patientID, startStage string, comorbidities []string) {
	t.Helper()
	profile := map[string]interface{}{
		"id":            uuid.New().String(),
		"patient_id":    patientID,
		"age":           60,
		"sex":           "M",
		"dm_type":       "NONE",
		"htn_status":    "NONE",
		"ckm_stage_v2":  startStage,
		"egfr":          decimal.NewFromFloat(95.0).String(),
		"comorbidities": pq.StringArray(comorbidities),
		"active":        1,
		"created_at":    time.Now().UTC(),
		"updated_at":    time.Now().UTC(),
	}
	if err := db.Table("patient_profiles").Create(profile).Error; err != nil {
		t.Fatalf("seed patient_profile %s: %v", patientID, err)
	}
}

// seedCKMLab inserts a LabEntry with the given type + value for a patient.
// measured_at is set to "now" so the recomputation service sees this as
// the most recent reading.
func seedCKMLab(t *testing.T, db *gorm.DB, patientID, labType string, value float64) {
	t.Helper()
	entry := map[string]interface{}{
		"id":                uuid.New().String(),
		"patient_id":        patientID,
		"lab_type":          labType,
		"value":             decimal.NewFromFloat(value).String(),
		"unit":              "",
		"measured_at":       time.Now().UTC(),
		"source":            "TEST",
		"validation_status": "ACCEPTED",
	}
	if err := db.Table("lab_entries").Create(entry).Error; err != nil {
		t.Fatalf("seed lab_entry %s: %v", labType, err)
	}
}

// TestCKMRecomputation_LowLVEFTriggers4cTransition asserts that a patient
// with no HF comorbidity and a starting stage of "3" transitions to "4c"
// once a new LVEF reading of 30% lands in the database. This is the
// canonical P7-B trigger-gap resolution: the plan's verification question
// #1 is "Does a new LVEF observation on a patient with EF=30% trigger
// CKMRecomputationService.RecomputeAndPublish?" — this test says yes,
// and it writes the stage change through to profile.CKMStageV2.
func TestCKMRecomputation_LowLVEFTriggers4cTransition(t *testing.T) {
	db := setupCKMRecomputationTestDB(t)
	seedCKMPatient(t, db, "p-hf-new", "3", nil)
	seedCKMLab(t, db, "p-hf-new", models.LabTypeLVEF, 30.0)

	publisher := NewCKMTransitionPublisher(db, nil, zap.NewNop())
	svc := NewCKMRecomputationService(db, publisher, zap.NewNop())

	transitioned, err := svc.RecomputeAndPublish("p-hf-new", "obs-lvef-001")
	if err != nil {
		t.Fatalf("RecomputeAndPublish: %v", err)
	}
	if !transitioned {
		t.Fatal("expected transition, got none")
	}

	// Verify the profile was updated.
	var row struct {
		CKMStageV2 string
	}
	if err := db.Table("patient_profiles").
		Select("ckm_stage_v2").
		Where("patient_id = ?", "p-hf-new").
		First(&row).Error; err != nil {
		t.Fatalf("read profile: %v", err)
	}
	if row.CKMStageV2 != "4c" {
		t.Errorf("profile.CKMStageV2 = %q, want 4c", row.CKMStageV2)
	}
}

// TestCKMRecomputation_UnchangedStageIsNoop asserts that a patient whose
// computed stage matches the persisted stage produces no transition
// event. This prevents false-positive CKM_STAGE_TRANSITION spam when
// routine observations arrive that don't cross any staging boundary.
func TestCKMRecomputation_UnchangedStageIsNoop(t *testing.T) {
	db := setupCKMRecomputationTestDB(t)
	// Stage 0 patient with no labs, no HF, no comorbidities.
	seedCKMPatient(t, db, "p-stable", "0", nil)

	publisher := NewCKMTransitionPublisher(db, nil, zap.NewNop())
	svc := NewCKMRecomputationService(db, publisher, zap.NewNop())

	transitioned, err := svc.RecomputeAndPublish("p-stable", "obs-noop")
	if err != nil {
		t.Fatalf("RecomputeAndPublish: %v", err)
	}
	if transitioned {
		t.Error("expected no transition for steady-state patient, got one")
	}
}

// TestCKMRecomputation_ExistingStage4cPreservedWhenLVEFMissing asserts
// that a patient already at 4c is NOT regressed to a lower stage just
// because the latest recomputation ran without an LVEF reading in the
// database. This prevents a race condition where a CKM-relevant
// observation lands before the LVEF lab entry has been synced, which
// would otherwise cause a spurious 4c→0 transition.
func TestCKMRecomputation_ExistingStage4cPreservedWhenLVEFMissing(t *testing.T) {
	db := setupCKMRecomputationTestDB(t)
	seedCKMPatient(t, db, "p-stable-4c", "4c", nil)
	// No labs seeded — LVEF lookup will return nil.

	publisher := NewCKMTransitionPublisher(db, nil, zap.NewNop())
	svc := NewCKMRecomputationService(db, publisher, zap.NewNop())

	transitioned, err := svc.RecomputeAndPublish("p-stable-4c", "obs-no-lvef")
	if err != nil {
		t.Fatalf("RecomputeAndPublish: %v", err)
	}
	if transitioned {
		t.Error("expected existing 4c patient to be preserved, got transition")
	}
}

// TestCKMRecomputation_HFComorbidityHonoured asserts that a patient with
// HEART_FAILURE in their comorbidity list + any LVEF reading ends up at
// Stage 4c. Covers the path where HF is diagnosed upstream (by a
// clinician or Condition sync) before any LVEF lab arrives — the next
// observation of any CKM-relevant type should not regress the stage.
func TestCKMRecomputation_HFComorbidityHonoured(t *testing.T) {
	db := setupCKMRecomputationTestDB(t)
	seedCKMPatient(t, db, "p-hf-coded", "3", []string{"HEART_FAILURE"})
	seedCKMLab(t, db, "p-hf-coded", models.LabTypeLVEF, 55.0) // preserved EF

	publisher := NewCKMTransitionPublisher(db, nil, zap.NewNop())
	svc := NewCKMRecomputationService(db, publisher, zap.NewNop())

	transitioned, err := svc.RecomputeAndPublish("p-hf-coded", "obs-hf")
	if err != nil {
		t.Fatalf("RecomputeAndPublish: %v", err)
	}
	if !transitioned {
		t.Fatal("expected 3→4c transition for HF-coded patient, got none")
	}
}

// TestCKMRecomputation_NilDeps_IsDefensiveNoop asserts that calling the
// service with a nil publisher (e.g. bootstrap/test harness) logs a
// warning and returns cleanly rather than panicking.
func TestCKMRecomputation_NilDeps_IsDefensiveNoop(t *testing.T) {
	svc := NewCKMRecomputationService(nil, nil, zap.NewNop())
	transitioned, err := svc.RecomputeAndPublish("p1", "obs-x")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
	if transitioned {
		t.Error("expected false transition, got true")
	}
}

// TestCKMRecomputation_EmptyPatientID_IsNoop covers the defensive guard
// at the top of RecomputeAndPublish — an empty patient ID must not hit
// the database (which would return an error).
func TestCKMRecomputation_EmptyPatientID_IsNoop(t *testing.T) {
	db := setupCKMRecomputationTestDB(t)
	publisher := NewCKMTransitionPublisher(db, nil, zap.NewNop())
	svc := NewCKMRecomputationService(db, publisher, zap.NewNop())

	transitioned, err := svc.RecomputeAndPublish("", "obs-anon")
	if err != nil {
		t.Errorf("expected nil error for empty patient_id, got %v", err)
	}
	if transitioned {
		t.Error("expected no transition for empty patient_id")
	}
}
