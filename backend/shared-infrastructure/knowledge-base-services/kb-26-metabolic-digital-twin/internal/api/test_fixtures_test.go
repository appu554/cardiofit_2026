package api

import (
	"fmt"
	"sync/atomic"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"kb-26-metabolic-digital-twin/internal/database"
)

var testDBCounter int64

// newTestDB returns a fresh in-memory sqlite *database.Database with
// AttributionVerdict and LedgerEntry tables created. Per-test isolation
// via atomic counter on the DB name; sqlite in-memory DB auto-disposes
// when the last connection closes.
//
// Uses explicit DDL (not gorm.AutoMigrate) because the GORM default
// `default:gen_random_uuid()` on model primary keys generates Postgres
// SQL that sqlite rejects at parse time.
func newTestDB(t *testing.T) *database.Database {
	t.Helper()
	n := atomic.AddInt64(&testDBCounter, 1)
	dsn := fmt.Sprintf("file:kb26_test_%d?mode=memory&cache=shared", n)
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	// Close the connection pool when the test ends. sqlite in-memory DBs
	// auto-dispose when the last connection closes, but GORM holds the pool
	// open until the process exits or it's explicitly closed.
	t.Cleanup(func() {
		if sqlDB, derr := gdb.DB(); derr == nil {
			_ = sqlDB.Close()
		}
	})

	ddl := []string{
		// attribution_verdicts: computed_at must be NOT NULL to match
		// GORM's autoCreateTime semantic (the model field has that tag,
		// and the column is a sort key in idx_av_patient_computed — a
		// NULL value would silently break ordering).
		// ledger_entry_id is intentionally nullable: the handler sets it
		// to &entry.ID only after the ledger append succeeds.
		`CREATE TABLE IF NOT EXISTS attribution_verdicts (
			id TEXT PRIMARY KEY,
			consolidated_record_id TEXT NOT NULL,
			patient_id TEXT NOT NULL,
			cohort_id TEXT,
			clinician_label TEXT NOT NULL,
			technical_label TEXT,
			risk_difference REAL,
			risk_reduction_pct REAL,
			counterfactual_risk REAL,
			observed_outcome INTEGER DEFAULT 0,
			prediction_window_days INTEGER,
			attribution_method TEXT NOT NULL DEFAULT 'RULE_BASED',
			method_version TEXT,
			rationale TEXT,
			ledger_entry_id TEXT,
			computed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_av_consolidated_record ON attribution_verdicts(consolidated_record_id)`,
		`CREATE INDEX IF NOT EXISTS idx_av_patient_computed ON attribution_verdicts(patient_id, computed_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_av_cohort_label ON attribution_verdicts(cohort_id, clinician_label)`,
		// governance_ledger_entries: created_at must be NOT NULL to match
		// GORM's autoCreateTime tag. The column is indexed and used for
		// ordering in the Snapshot/VerifyChain path.
		`CREATE TABLE IF NOT EXISTS governance_ledger_entries (
			id TEXT PRIMARY KEY,
			sequence INTEGER UNIQUE NOT NULL,
			entry_type TEXT NOT NULL,
			subject_id TEXT,
			payload_json TEXT NOT NULL,
			prior_hash TEXT NOT NULL,
			entry_hash TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ledger_subject ON governance_ledger_entries(subject_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ledger_created ON governance_ledger_entries(created_at)`,
		// Append additional CREATE TABLE statements here as new attribution
		// or governance tables are added in future sprints.
	}
	for _, stmt := range ddl {
		if err := gdb.Exec(stmt).Error; err != nil {
			t.Fatalf("ddl %q: %v", stmt, err)
		}
	}
	return &database.Database{DB: gdb}
}
