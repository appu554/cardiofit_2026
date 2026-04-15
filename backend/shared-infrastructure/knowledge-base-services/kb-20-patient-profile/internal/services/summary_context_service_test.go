package services

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-patient-profile/internal/models"
)

// setupSummaryContextTestDB creates an in-memory sqlite DB with the
// minimum schema needed by BuildContext: patient_profiles +
// medication_states + lab_entries. Uses raw DDL because the production
// schema carries PostgreSQL-specific defaults that sqlite cannot parse.
func setupSummaryContextTestDB(t *testing.T) *gorm.DB {
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
			cv_risk_category TEXT,
			htn_status TEXT,
			hba1c TEXT,
			egfr TEXT,
			uacr TEXT,
			potassium TEXT,
			ckm_stage_v2 TEXT,
			ckm_substage_metadata TEXT,
			engagement_composite TEXT,
			engagement_status TEXT,
			active INTEGER DEFAULT 1,
			created_at DATETIME,
			updated_at DATETIME
		)
	`).Error
	if err != nil {
		t.Fatalf("create patient_profiles: %v", err)
	}

	err = db.Exec(`
		CREATE TABLE medication_states (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL,
			drug_name TEXT NOT NULL,
			drug_class TEXT NOT NULL,
			dose_mg TEXT,
			fhir_medication_request_id TEXT,
			atc_code TEXT,
			is_active INTEGER DEFAULT 1,
			start_date DATETIME,
			route TEXT,
			updated_at DATETIME,
			created_at DATETIME
		)
	`).Error
	if err != nil {
		t.Fatalf("create medication_states: %v", err)
	}

	// Phase 8 P8-5: safety_events table for confounder flag tests.
	// Raw DDL (not GORM AutoMigrate) because sqlite does not
	// understand the Postgres-specific gen_random_uuid() default
	// that the production model uses. Column set mirrors the
	// models.SafetyEvent struct fields exactly so GORM reads work.
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
		t.Fatalf("create lab_entries: %v", err)
	}
	return db
}

func seedSummaryPatient(t *testing.T, db *gorm.DB, patientID string, egfr *float64, weightKg float64, stratum string) {
	t.Helper()
	profile := map[string]interface{}{
		"id":               uuid.New().String(),
		"patient_id":       patientID,
		"age":              55,
		"sex":              "M",
		"bmi":              decimal.NewFromFloat(27.8).String(),
		"dm_type":          "T2DM",
		"cv_risk_category": stratum,
		"ckm_stage_v2":     "3",
		"engagement_status": "ENGAGED",
		"active":           1,
		"created_at":       time.Now().UTC(),
		"updated_at":       time.Now().UTC(),
	}
	if weightKg > 0 {
		profile["weight_kg"] = decimal.NewFromFloat(weightKg).String()
	}
	if egfr != nil {
		profile["egfr"] = decimal.NewFromFloat(*egfr).String()
	}
	if err := db.Table("patient_profiles").Create(profile).Error; err != nil {
		t.Fatalf("seed patient_profile: %v", err)
	}
}

func seedSummaryMed(t *testing.T, db *gorm.DB, patientID, drugClass, drugName string, isActive bool) {
	t.Helper()
	row := map[string]interface{}{
		"id":         uuid.New().String(),
		"patient_id": patientID,
		"drug_name":  drugName,
		"drug_class": drugClass,
		"dose_mg":    "500",
		"is_active":  isActive,
		"start_date": time.Now().UTC(),
		"updated_at": time.Now().UTC(),
		"created_at": time.Now().UTC(),
		"route":      "ORAL",
	}
	if err := db.Table("medication_states").Create(row).Error; err != nil {
		t.Fatalf("seed medication_state: %v", err)
	}
}

func seedSummaryLab(t *testing.T, db *gorm.DB, patientID, labType string, value float64, measuredAt time.Time) {
	t.Helper()
	row := map[string]interface{}{
		"id":                uuid.New().String(),
		"patient_id":        patientID,
		"lab_type":          labType,
		"value":             decimal.NewFromFloat(value).String(),
		"unit":              "",
		"measured_at":       measuredAt,
		"source":            "TEST",
		"validation_status": "ACCEPTED",
	}
	if err := db.Table("lab_entries").Create(row).Error; err != nil {
		t.Fatalf("seed lab_entry: %v", err)
	}
}

// TestSummaryContext_HappyPath verifies the core contract: a fully-
// populated patient produces a SummaryContext with every field set
// correctly. This is the test that would have caught the missing
// endpoint the first time a card generation path tried to use it —
// it exercises the full assembly from the patient profile, active
// medications, and latest labs end-to-end against a real gorm.DB.
func TestSummaryContext_HappyPath(t *testing.T) {
	db := setupSummaryContextTestDB(t)
	egfr := 65.0
	seedSummaryPatient(t, db, "p-happy", &egfr, 82.5, "HIGH")
	seedSummaryMed(t, db, "p-happy", "METFORMIN", "metformin", true)
	seedSummaryMed(t, db, "p-happy", "ACEi", "lisinopril", true)
	// An inactive medication that should NOT appear in the context.
	seedSummaryMed(t, db, "p-happy", "STATIN", "atorvastatin", false)

	now := time.Now().UTC()
	// Two HbA1c readings — the newer one must win.
	seedSummaryLab(t, db, "p-happy", "HBA1C", 8.5, now.AddDate(0, 0, -90))
	seedSummaryLab(t, db, "p-happy", "HBA1C", 7.9, now.AddDate(0, 0, -10))
	seedSummaryLab(t, db, "p-happy", "FBG", 145.0, now.AddDate(0, 0, -5))
	// P8-2: seed a potassium lab to verify LatestPotassium populates
	// from the lab_entries path (same pattern as HbA1c / FBG).
	seedSummaryLab(t, db, "p-happy", "POTASSIUM", 4.2, now.AddDate(0, 0, -3))

	svc := NewSummaryContextService(db, nil, nil, zap.NewNop())
	ctx, err := svc.BuildContext(context.Background(),"p-happy")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	if ctx.PatientID != "p-happy" {
		t.Errorf("PatientID = %q, want p-happy", ctx.PatientID)
	}
	if ctx.Stratum != "HIGH" {
		t.Errorf("Stratum = %q, want HIGH", ctx.Stratum)
	}

	// Phase 8 P8-2 assertions: demographics, CKM stage, engagement
	if ctx.Age != 55 {
		t.Errorf("Age = %d, want 55", ctx.Age)
	}
	if ctx.Sex != "M" {
		t.Errorf("Sex = %q, want M", ctx.Sex)
	}
	if ctx.BMI != 27.8 {
		t.Errorf("BMI = %f, want 27.8", ctx.BMI)
	}
	if ctx.CKMStageV2 != "3" {
		t.Errorf("CKMStageV2 = %q, want 3", ctx.CKMStageV2)
	}
	if ctx.EngagementStatus != "ENGAGED" {
		t.Errorf("EngagementStatus = %q, want ENGAGED", ctx.EngagementStatus)
	}
	if ctx.EGFRValue != 65.0 {
		t.Errorf("EGFRValue = %f, want 65.0", ctx.EGFRValue)
	}
	if ctx.WeightKg != 82.5 {
		t.Errorf("WeightKg = %f, want 82.5", ctx.WeightKg)
	}
	if ctx.LatestHbA1c != 7.9 {
		t.Errorf("LatestHbA1c = %f, want 7.9 (newer reading)", ctx.LatestHbA1c)
	}
	if ctx.LatestFBG != 145.0 {
		t.Errorf("LatestFBG = %f, want 145.0", ctx.LatestFBG)
	}
	if ctx.LatestPotassium != 4.2 {
		t.Errorf("LatestPotassium = %f, want 4.2", ctx.LatestPotassium)
	}

	// Active medications only — STATIN (inactive) must be excluded.
	if len(ctx.Medications) != 2 {
		t.Errorf("len(Medications) = %d, want 2 (active only)", len(ctx.Medications))
	}
	foundMetformin := false
	foundACEi := false
	for _, m := range ctx.Medications {
		if m == "METFORMIN" {
			foundMetformin = true
		}
		if m == "ACEi" {
			foundACEi = true
		}
		if m == "STATIN" {
			t.Error("STATIN should not appear (inactive)")
		}
	}
	if !foundMetformin {
		t.Error("expected METFORMIN in active medications")
	}
	if !foundACEi {
		t.Error("expected ACEi in active medications")
	}
}

// TestSummaryContext_MissingPatientReturnsError asserts the service
// surfaces gorm.ErrRecordNotFound when the patient does not exist.
// The handler layer maps this to HTTP 404.
func TestSummaryContext_MissingPatientReturnsError(t *testing.T) {
	db := setupSummaryContextTestDB(t)
	svc := NewSummaryContextService(db, nil, nil, zap.NewNop())

	ctx, err := svc.BuildContext(context.Background(),"p-nonexistent")
	if err == nil {
		t.Error("expected error for missing patient, got nil")
	}
	if ctx != nil {
		t.Error("expected nil context for missing patient")
	}
	if err != gorm.ErrRecordNotFound {
		t.Errorf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

// TestSummaryContext_PartialData_NoLabs verifies that a patient with
// no lab entries gets a context with LatestHbA1c=0 and LatestFBG=0.
// The KB-23 consumers treat 0 as "no signal" per the MCU gate
// manager's `ctx.LatestHbA1c > 0` guards, so this is the correct
// zero-value semantics.
func TestSummaryContext_PartialData_NoLabs(t *testing.T) {
	db := setupSummaryContextTestDB(t)
	egfr := 70.0
	seedSummaryPatient(t, db, "p-no-labs", &egfr, 75.0, "MODERATE")
	seedSummaryMed(t, db, "p-no-labs", "METFORMIN", "metformin", true)

	svc := NewSummaryContextService(db, nil, nil, zap.NewNop())
	ctx, err := svc.BuildContext(context.Background(),"p-no-labs")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if ctx.LatestHbA1c != 0 {
		t.Errorf("LatestHbA1c = %f, want 0 (no labs)", ctx.LatestHbA1c)
	}
	if ctx.LatestFBG != 0 {
		t.Errorf("LatestFBG = %f, want 0 (no labs)", ctx.LatestFBG)
	}
	if ctx.EGFRValue != 70.0 {
		t.Errorf("EGFRValue = %f, want 70.0 (from profile, not lab)", ctx.EGFRValue)
	}
	if len(ctx.Medications) != 1 {
		t.Errorf("len(Medications) = %d, want 1", len(ctx.Medications))
	}
}

// TestSummaryContext_PartialData_NoMedications verifies a patient with
// no active medications gets an empty slice (not nil, not panicking).
// KB-23's consumer code uses len(ctx.Medications) so both shapes are
// semantically equivalent, but empty-slice is safer for JSON
// round-tripping through the HTTP layer.
func TestSummaryContext_PartialData_NoMedications(t *testing.T) {
	db := setupSummaryContextTestDB(t)
	egfr := 90.0
	seedSummaryPatient(t, db, "p-unmedicated", &egfr, 70.0, "LOW")

	svc := NewSummaryContextService(db, nil, nil, zap.NewNop())
	ctx, err := svc.BuildContext(context.Background(),"p-unmedicated")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if ctx.Medications == nil {
		t.Error("expected non-nil empty slice, got nil")
	}
	if len(ctx.Medications) != 0 {
		t.Errorf("len(Medications) = %d, want 0", len(ctx.Medications))
	}
}

// TestSummaryContext_NilEGFRProfile verifies a patient whose profile
// has a nil EGFR pointer (common on freshly-synced patients before
// any creatinine lab arrives) gets EGFRValue=0 without panicking.
// This was a real failure mode I hit during P7-B test-writing — nil
// pointer dereferences on fresh patients would crash the handler.
func TestSummaryContext_NilEGFRProfile(t *testing.T) {
	db := setupSummaryContextTestDB(t)
	seedSummaryPatient(t, db, "p-fresh", nil, 80.0, "LOW")

	svc := NewSummaryContextService(db, nil, nil, zap.NewNop())
	ctx, err := svc.BuildContext(context.Background(),"p-fresh")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if ctx.EGFRValue != 0 {
		t.Errorf("EGFRValue = %f, want 0 (nil profile EGFR)", ctx.EGFRValue)
	}
}

// TestSummaryContext_DistinctDrugClasses verifies that multiple
// medications in the same drug class (e.g., metformin IR + metformin
// XR) collapse to a single METFORMIN entry. This matters because the
// mandatory med checker iterates Medications and double-counting a
// drug class produces false "missing" verdicts.
func TestSummaryContext_DistinctDrugClasses(t *testing.T) {
	db := setupSummaryContextTestDB(t)
	egfr := 60.0
	seedSummaryPatient(t, db, "p-dupe", &egfr, 85.0, "MODERATE")
	seedSummaryMed(t, db, "p-dupe", "METFORMIN", "metformin IR", true)
	seedSummaryMed(t, db, "p-dupe", "METFORMIN", "metformin XR", true)

	svc := NewSummaryContextService(db, nil, nil, zap.NewNop())
	ctx, err := svc.BuildContext(context.Background(),"p-dupe")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if len(ctx.Medications) != 1 {
		t.Errorf("len(Medications) = %d, want 1 (distinct drug class)", len(ctx.Medications))
	}
}

// TestSummaryContext_EmptyPatientID asserts the service rejects an
// empty patient ID rather than hitting the database with a blank
// WHERE clause.
func TestSummaryContext_EmptyPatientID(t *testing.T) {
	db := setupSummaryContextTestDB(t)
	svc := NewSummaryContextService(db, nil, nil, zap.NewNop())

	_, err := svc.BuildContext(context.Background(),"")
	if err == nil {
		t.Error("expected error for empty patient id, got nil")
	}
}

// TestSummaryContext_JSONTagsMatchKB23Contract pins the wire contract
// by round-tripping a SummaryContext through JSON and asserting every
// field present in the struct shows up under its snake_case tag. If
// this test breaks, either the struct changed or the JSON tags no
// longer match the KB-23 PatientContext struct — both are breaking
// changes and must be caught at CI time, not at runtime.
//
// Phase 8 P8-1: this is the cross-service field-contract test that
// closes the gap the Phase 7 review identified.
func TestSummaryContext_JSONTagsMatchKB23Contract(t *testing.T) {
	// Build a SummaryContext with every field populated distinctly
	// so we can verify each one round-trips under the expected key.
	// Phase 8 P8-2: includes all the new demographics / CKM / labs /
	// engagement / CGM fields so the wire contract test is complete.
	engagement := 0.82
	tir := 78.5
	reportAt := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	cac := 250.0
	lvef := 35.0
	ctx := &SummaryContext{
		// P8-1 core
		PatientID:              "p-wire",
		Stratum:                "HIGH",
		Medications:            []string{"METFORMIN", "ACEi"},
		EGFRValue:              55.0,
		LatestHbA1c:            8.2,
		LatestFBG:              150.0,
		IsAcuteIll:             true,
		HasRecentTransfusion:   true,
		HasRecentHypoglycaemia: false,
		WeightKg:               80.0,

		// P8-2 new fields
		Age:                 58,
		Sex:                 "F",
		BMI:                 29.4,
		CKMStageV2:          "4c",
		LatestPotassium:     4.1,
		EngagementComposite: &engagement,
		EngagementStatus:    "ENGAGED",
		HasCGM:              true,
		LatestCGMTIR:        &tir,
		LatestCGMGRIZone:    "B",
		CGMReportAt:         &reportAt,
		CKMSubstageMetadata: &CKMSubstageWire{
			HFClassification: "HFrEF",
			LVEFPercent:      &lvef,
			NYHAClass:        "II",
			CACScore:         &cac,
		},
	}

	// Marshal → raw JSON → re-parse as a map[string]interface{}
	// so we can assert on field keys directly without depending on
	// the receiving Go struct.
	raw := mustMarshalForTest(t, ctx)

	expectedKeys := []string{
		// P8-1 core
		"patient_id", "stratum", "medications", "egfr_value",
		"latest_hba1c", "latest_fbg", "is_acute_illness",
		"has_recent_transfusion", "has_recent_hypoglycaemia", "weight_kg",
		// P8-2 new
		"age", "sex", "bmi",
		"ckm_stage_v2", "ckm_substage_metadata",
		"latest_potassium",
		"engagement_composite", "engagement_status",
		"has_cgm", "latest_cgm_tir", "latest_cgm_gri_zone", "cgm_report_at",
	}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected JSON key %q in wire format, got %+v", key, raw)
		}
	}

	// Type spot-checks on a few fields to catch accidental type
	// drift (e.g., someone changes EGFRValue from float64 to string).
	if _, ok := raw["egfr_value"].(float64); !ok {
		t.Errorf("egfr_value should serialize as float64, got %T", raw["egfr_value"])
	}
	if meds, ok := raw["medications"].([]interface{}); !ok {
		t.Errorf("medications should serialize as []interface{}, got %T", raw["medications"])
	} else if len(meds) != 2 {
		t.Errorf("medications len = %d, want 2", len(meds))
	}
}

// mustMarshalForTest serializes v to JSON and re-parses it as a
// plain map so the test can assert on field keys without depending
// on the receiving Go struct's shape. This is the core assertion
// mechanism for the wire-contract test — any JSON tag drift breaks
// it deterministically at CI time.
func mustMarshalForTest(t *testing.T, v interface{}) map[string]interface{} {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json marshal: %v", err)
	}
	out := map[string]interface{}{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	return out
}

// TestSummaryContext_ConfounderFlags_PopulatedFromSafetyEvents is
// the Phase 8 P8-5 end-to-end check: seed a patient with an
// ACUTE_ILLNESS + HYPO_EVENT in the safety_events table, call
// BuildContext with a real SafetyEventRecorder wired in, and assert
// that IsAcuteIll + HasRecentHypoglycaemia end up populated on the
// returned SummaryContext. This closes the loop between the
// recorder write path and the summary-context read path.
func TestSummaryContext_ConfounderFlags_PopulatedFromSafetyEvents(t *testing.T) {
	db := setupSummaryContextTestDB(t)
	egfr := 65.0
	seedSummaryPatient(t, db, "p-confounders", &egfr, 80.0, "HIGH")

	// Seed safety events directly via the recorder — matches the
	// production flow where lab_service + other callers invoke
	// Record alongside their eventBus.Publish calls.
	recorder := NewSafetyEventRecorder(db, zap.NewNop())
	_ = recorder.Record("p-confounders", models.SafetyEventAcuteIllness, "SEVERE",
		"hospitalisation for CAP", time.Now().UTC().Add(-2*24*time.Hour))
	_ = recorder.Record("p-confounders", models.SafetyEventHypoEvent, "MODERATE",
		"nocturnal hypo, resolved with glucose tabs", time.Now().UTC().Add(-10*24*time.Hour))
	// No transfusion event — HasRecentTransfusion should stay false.

	svc := NewSummaryContextService(db, nil, recorder, zap.NewNop())
	ctx, err := svc.BuildContext(context.Background(), "p-confounders")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if ctx == nil {
		t.Fatal("expected non-nil context")
	}

	if !ctx.IsAcuteIll {
		t.Error("IsAcuteIll = false, want true (ACUTE_ILLNESS event 2d old)")
	}
	if ctx.HasRecentTransfusion {
		t.Error("HasRecentTransfusion = true, want false (no transfusion event)")
	}
	if !ctx.HasRecentHypoglycaemia {
		t.Error("HasRecentHypoglycaemia = false, want true (HYPO_EVENT 10d old)")
	}
}

// TestSummaryContext_ConfounderFlags_NilRecorderDegradesCleanly
// verifies the existing fallback path still works: when the
// recorder is nil, the confounder flags default to false and
// BuildContext still returns a usable SummaryContext. Pins the
// P8-2 nil-safe fallback as a regression guard after P8-5 added
// the real flag wiring.
func TestSummaryContext_ConfounderFlags_NilRecorderDegradesCleanly(t *testing.T) {
	db := setupSummaryContextTestDB(t)
	egfr := 70.0
	seedSummaryPatient(t, db, "p-no-recorder", &egfr, 75.0, "MODERATE")

	svc := NewSummaryContextService(db, nil, nil, zap.NewNop())
	ctx, err := svc.BuildContext(context.Background(), "p-no-recorder")
	if err != nil {
		t.Fatalf("BuildContext: %v", err)
	}
	if ctx.IsAcuteIll {
		t.Error("IsAcuteIll should default to false when recorder nil")
	}
	if ctx.HasRecentTransfusion {
		t.Error("HasRecentTransfusion should default to false when recorder nil")
	}
	if ctx.HasRecentHypoglycaemia {
		t.Error("HasRecentHypoglycaemia should default to false when recorder nil")
	}
}
