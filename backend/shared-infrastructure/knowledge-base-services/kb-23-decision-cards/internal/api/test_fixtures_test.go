package api

import (
	"fmt"
	"sync/atomic"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-23-decision-cards/internal/database"
)

// dbCounter ensures each test gets a fully isolated in-memory SQLite DB.
var dbCounter int64

// newTestDB returns a *database.Database backed by an in-memory sqlite DB
// with Gap 21 Sprint 1+2a+3 tables created. The caller owns cleanup —
// sqlite in-memory DB disposes automatically when the last connection closes.
//
// We create tables with explicit SQLite-compatible DDL instead of AutoMigrate
// because the GORM struct tags use gen_random_uuid() (a Postgres function)
// which SQLite rejects at DDL parse time. The BeforeCreate hooks on the
// models handle UUID generation at insert time so the missing DEFAULT is safe.
//
// Partial unique index matches Postgres semantics: NULL values (nil pointers
// on the Go side, representing "no key supplied") are allowed to repeat.
// Non-null idempotency keys must be unique. The production Postgres schema
// relies on *string nil = SQL NULL and the standard uniqueIndex allowing
// multiple NULLs.
func newTestDB(t *testing.T) *database.Database {
	t.Helper()
	// Each test gets a unique named in-memory DB to avoid cross-test pollution.
	n := atomic.AddInt64(&dbCounter, 1)
	dsn := fmt.Sprintf("file:testdb_%d?mode=memory&cache=shared", n)
	gdb, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}

	ddl := []string{
		`CREATE TABLE IF NOT EXISTS outcome_records (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL,
			lifecycle_id TEXT,
			cohort_id TEXT,
			scope TEXT DEFAULT 'PATIENT_ALERT',
			outcome_type TEXT NOT NULL,
			outcome_occurred INTEGER NOT NULL DEFAULT 0,
			occurred_at DATETIME,
			source TEXT NOT NULL,
			source_record_id TEXT,
			idempotency_key TEXT,
			reconciliation TEXT NOT NULL DEFAULT 'PENDING',
			reconciled_id TEXT,
			ingested_at DATETIME,
			notes TEXT
		)`,
		// Partial unique index: only enforce uniqueness when idempotency_key is
		// supplied. Matches Postgres semantics: a standard unique index allows
		// multiple NULL values (nil *string pointers on the Go side serialize
		// to SQL NULL), so legacy/keyless rows never collide. Only non-null
		// keys must be unique.
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_or_idem_key
		 ON outcome_records(idempotency_key)
		 WHERE idempotency_key IS NOT NULL`,
		`CREATE TABLE IF NOT EXISTS consolidated_alert_records (
			id TEXT PRIMARY KEY,
			lifecycle_id TEXT NOT NULL,
			patient_id TEXT NOT NULL,
			cohort_id TEXT,
			pre_alert_risk_score REAL,
			pre_alert_risk_tier TEXT,
			prediction_model_id TEXT,
			detected_at DATETIME NOT NULL,
			delivered_at DATETIME,
			acknowledged_at DATETIME,
			actioned_at DATETIME,
			resolved_at DATETIME,
			time_zero DATETIME NOT NULL,
			treatment_strategy TEXT NOT NULL,
			action_type TEXT,
			override_reason TEXT,
			outcome_record_id TEXT,
			outcome_occurred INTEGER,
			outcome_type TEXT,
			horizon_days INTEGER,
			built_at DATETIME
		)`,
	}

	for _, stmt := range ddl {
		if err := gdb.Exec(stmt).Error; err != nil {
			t.Fatalf("create table: %v\nSQL: %s", err, stmt)
		}
	}

	return &database.Database{DB: gdb}
}
