package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-26-metabolic-digital-twin/internal/models"
)

// setupCalibrationTestDB creates a sqlite in-memory DB with the two tables the
// monitor joins: cate_estimates (Task 2) and attribution_verdicts (Gap 21).
// Uses raw DDL because production GORM defaults (gen_random_uuid, text[])
// are Postgres-specific — same pattern as setupBPContextTestDB and
// setupInterventionTestDB.
// KEEP IN SYNC: if CATEEstimate (models/cate_estimate.go) or AttributionVerdict
// (models/attribution_verdict.go) gains a new NOT NULL column in production,
// update the DDL below to match. Raw DDL is used because GORM auto-migrate
// cannot handle the Postgres-specific defaults used in production.
func setupCalibrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, _ := db.DB()
	stmts := []string{
		`CREATE TABLE cate_estimates (
			id                         TEXT PRIMARY KEY,
			consolidated_record_id     TEXT NOT NULL,
			patient_id                 TEXT NOT NULL,
			cohort_id                  TEXT NOT NULL,
			intervention_id            TEXT NOT NULL,
			learner_type               TEXT NOT NULL,
			point_estimate             REAL,
			ci_lower                   REAL,
			ci_upper                   REAL,
			horizon_days               INTEGER,
			propensity                 REAL,
			overlap_status             TEXT NOT NULL,
			training_n                 INTEGER,
			cohort_treated_n           INTEGER,
			cohort_control_n           INTEGER,
			feature_contributions_json TEXT,
			feature_contribution_keys  TEXT,
			model_version              TEXT,
			ledger_entry_id            TEXT,
			computed_at                DATETIME
		)`,
		`CREATE INDEX idx_cate_consolidated ON cate_estimates(consolidated_record_id)`,
		`CREATE TABLE attribution_verdicts (
			id                         TEXT PRIMARY KEY,
			consolidated_record_id     TEXT NOT NULL,
			patient_id                 TEXT NOT NULL,
			cohort_id                  TEXT,
			clinician_label            TEXT NOT NULL,
			technical_label            TEXT,
			risk_difference            REAL,
			risk_reduction_pct         REAL,
			counterfactual_risk        REAL,
			observed_outcome           INTEGER,
			prediction_window_days     INTEGER,
			attribution_method         TEXT NOT NULL DEFAULT 'RULE_BASED',
			method_version             TEXT,
			rationale                  TEXT,
			ledger_entry_id            TEXT,
			computed_at                DATETIME
		)`,
		`CREATE INDEX idx_av_consolidated ON attribution_verdicts(consolidated_record_id)`,
	}
	for _, s := range stmts {
		if _, err := sqlDB.Exec(s); err != nil {
			t.Fatalf("DDL exec: %v", err)
		}
	}
	return db
}

// seedMatchedPairs creates n matched CATEEstimate+AttributionVerdict pairs sharing
// the same ConsolidatedRecordID, all in the given cohort × intervention.
func seedMatchedPairs(t *testing.T, db *gorm.DB, cohort, intervention string, horizon int, predCATE, attribEffect float64, n int) {
	t.Helper()
	now := time.Now().UTC()
	for i := 0; i < n; i++ {
		rid := uuid.New()
		if result := db.Exec(`INSERT INTO cate_estimates
			(id, consolidated_record_id, patient_id, cohort_id, intervention_id, learner_type,
			 point_estimate, ci_lower, ci_upper, horizon_days, overlap_status, computed_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
			uuid.New().String(), rid.String(), fmt.Sprintf("P%d", i), cohort, intervention, string(models.LearnerBaselineDiffMeans),
			predCATE, predCATE-0.03, predCATE+0.03, horizon, string(models.OverlapPass), now); result.Error != nil {
			t.Fatalf("seed cate_estimates row %d: %v", i, result.Error)
		}
		if result := db.Exec(`INSERT INTO attribution_verdicts
			(id, consolidated_record_id, patient_id, cohort_id, clinician_label, risk_difference,
			 prediction_window_days, attribution_method, computed_at)
			VALUES (?,?,?,?,?,?,?,?,?)`,
			uuid.New().String(), rid.String(), fmt.Sprintf("P%d", i), cohort, string(models.LabelPrevented),
			attribEffect, horizon, "RULE_BASED", now); result.Error != nil {
			t.Fatalf("seed attribution_verdicts row %d: %v", i, result.Error)
		}
	}
}

func TestCalibrationMonitor_CalibratedSignalNoAlarm(t *testing.T) {
	db := setupCalibrationTestDB(t)
	seedMatchedPairs(t, db, "hcf_catalyst_chf", "nurse_phone_48h", 30, 0.15, 0.15, 30)
	mon := NewCATECalibrationMonitor(db, CalibrationConfig{AbsDiffAlarm: 0.05, RollingWindowDays: 90, MinMatchedPairs: 20})
	sum, err := mon.ComputeCalibrationSummary("hcf_catalyst_chf", "nurse_phone_48h", 30)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if sum.AlarmTriggered {
		t.Fatalf("calibrated signal should not alarm, meanAbsDiff=%.3f", sum.MeanAbsDiff)
	}
	if sum.Status != CalibrationOK {
		t.Fatalf("want OK, got %s", sum.Status)
	}
}

func TestCalibrationMonitor_MiscalibratedSignalAlarms(t *testing.T) {
	db := setupCalibrationTestDB(t)
	// Predicted 0.20 but attributed only 0.02 on every pair → miscalibration.
	seedMatchedPairs(t, db, "hcf_catalyst_chf", "gp_visit_7d", 30, 0.20, 0.02, 30)
	mon := NewCATECalibrationMonitor(db, CalibrationConfig{AbsDiffAlarm: 0.05, RollingWindowDays: 90, MinMatchedPairs: 20})
	sum, _ := mon.ComputeCalibrationSummary("hcf_catalyst_chf", "gp_visit_7d", 30)
	if !sum.AlarmTriggered {
		t.Fatalf("miscalibrated signal should alarm, meanAbsDiff=%.3f", sum.MeanAbsDiff)
	}
	if sum.Status != CalibrationAlarm {
		t.Fatalf("want ALARM, got %s", sum.Status)
	}
}

func TestCalibrationMonitor_InsufficientPairsNoAlarm(t *testing.T) {
	db := setupCalibrationTestDB(t)
	seedMatchedPairs(t, db, "hcf_catalyst_chf", "cardiology_referral", 30, 0.20, 0.02, 3)
	mon := NewCATECalibrationMonitor(db, CalibrationConfig{AbsDiffAlarm: 0.05, RollingWindowDays: 90, MinMatchedPairs: 20})
	sum, _ := mon.ComputeCalibrationSummary("hcf_catalyst_chf", "cardiology_referral", 30)
	if sum.AlarmTriggered {
		t.Fatal("insufficient-signal should not alarm")
	}
	if sum.Status != CalibrationInsufficientSignal {
		t.Fatalf("want INSUFFICIENT_SIGNAL, got %s", sum.Status)
	}
}

func TestCalibrationMonitor_EvaluateAppendsLedgerOnAlarm(t *testing.T) {
	db := setupCalibrationTestDB(t)
	// Miscalibrated signal → alarm → ledger entry must be appended.
	seedMatchedPairs(t, db, "hcf_catalyst_chf", "gp_visit_7d", 30, 0.20, 0.02, 25)
	mon := NewCATECalibrationMonitor(db, CalibrationConfig{AbsDiffAlarm: 0.05, RollingWindowDays: 90, MinMatchedPairs: 20})
	ledger := NewInMemoryLedger([]byte("test-hmac-key"))
	if err := mon.EvaluateAndAlarm("hcf_catalyst_chf", ledger); err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	entries := ledger.Entries()
	var found int
	for _, e := range entries {
		if e.EntryType == "CATE_MISCALIBRATION" {
			found++
		}
	}
	if found < 1 {
		t.Fatalf("expected at least one CATE_MISCALIBRATION ledger entry, got %d (entries=%d total)", found, len(entries))
	}
	if ok, _, _ := ledger.VerifyChain(); !ok {
		t.Fatal("ledger chain must verify after alarm append")
	}
}

func TestCalibrationMonitor_OutOfWindowPairsExcluded(t *testing.T) {
	db := setupCalibrationTestDB(t)
	// Seed pairs with computed_at 200 days in the past — outside the 90-day rolling window.
	oldTime := time.Now().UTC().AddDate(0, 0, -200)
	for i := 0; i < 25; i++ {
		rid := uuid.New()
		res := db.Exec(`INSERT INTO cate_estimates
			(id, consolidated_record_id, patient_id, cohort_id, intervention_id, learner_type,
			 point_estimate, ci_lower, ci_upper, horizon_days, overlap_status, computed_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
			uuid.New().String(), rid.String(), fmt.Sprintf("P%d", i), "hcf_catalyst_chf", "nurse_phone_48h",
			string(models.LearnerBaselineDiffMeans), 0.15, 0.12, 0.18, 30, string(models.OverlapPass), oldTime)
		if res.Error != nil {
			t.Fatalf("seed old cate row: %v", res.Error)
		}
		res = db.Exec(`INSERT INTO attribution_verdicts
			(id, consolidated_record_id, patient_id, cohort_id, clinician_label, risk_difference,
			 prediction_window_days, attribution_method, computed_at)
			VALUES (?,?,?,?,?,?,?,?,?)`,
			uuid.New().String(), rid.String(), fmt.Sprintf("P%d", i), "hcf_catalyst_chf",
			string(models.LabelPrevented), 0.15, 30, "RULE_BASED", oldTime)
		if res.Error != nil {
			t.Fatalf("seed old attribution row: %v", res.Error)
		}
	}
	mon := NewCATECalibrationMonitor(db, CalibrationConfig{AbsDiffAlarm: 0.05, RollingWindowDays: 90, MinMatchedPairs: 20})
	sum, err := mon.ComputeCalibrationSummary("hcf_catalyst_chf", "nurse_phone_48h", 30)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if sum.MatchedPairs != 0 {
		t.Fatalf("expected 0 matched pairs (all outside window), got %d", sum.MatchedPairs)
	}
	if sum.Status != CalibrationInsufficientSignal {
		t.Fatalf("expected INSUFFICIENT_SIGNAL, got %s", sum.Status)
	}
}

func TestCalibrationMonitor_OverlapFailPairsExcluded(t *testing.T) {
	db := setupCalibrationTestDB(t)
	// Seed 25 pairs but with CATE rows carrying OverlapBelowFloor — must be excluded from the join.
	now := time.Now().UTC()
	for i := 0; i < 25; i++ {
		rid := uuid.New()
		res := db.Exec(`INSERT INTO cate_estimates
			(id, consolidated_record_id, patient_id, cohort_id, intervention_id, learner_type,
			 point_estimate, ci_lower, ci_upper, horizon_days, overlap_status, computed_at)
			VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
			uuid.New().String(), rid.String(), fmt.Sprintf("P%d", i), "hcf_catalyst_chf", "nurse_phone_48h",
			string(models.LearnerBaselineDiffMeans), 0.0, 0.0, 0.0, 30, string(models.OverlapBelowFloor), now)
		if res.Error != nil {
			t.Fatalf("seed overlap-fail cate row: %v", res.Error)
		}
		res = db.Exec(`INSERT INTO attribution_verdicts
			(id, consolidated_record_id, patient_id, cohort_id, clinician_label, risk_difference,
			 prediction_window_days, attribution_method, computed_at)
			VALUES (?,?,?,?,?,?,?,?,?)`,
			uuid.New().String(), rid.String(), fmt.Sprintf("P%d", i), "hcf_catalyst_chf",
			string(models.LabelPrevented), 0.15, 30, "RULE_BASED", now)
		if res.Error != nil {
			t.Fatalf("seed attribution row: %v", res.Error)
		}
	}
	mon := NewCATECalibrationMonitor(db, CalibrationConfig{AbsDiffAlarm: 0.05, RollingWindowDays: 90, MinMatchedPairs: 20})
	sum, _ := mon.ComputeCalibrationSummary("hcf_catalyst_chf", "nurse_phone_48h", 30)
	if sum.MatchedPairs != 0 {
		t.Fatalf("expected 0 matched pairs (all CATE rows are overlap-fail), got %d", sum.MatchedPairs)
	}
	if sum.Status != CalibrationInsufficientSignal {
		t.Fatalf("expected INSUFFICIENT_SIGNAL, got %s", sum.Status)
	}
}
