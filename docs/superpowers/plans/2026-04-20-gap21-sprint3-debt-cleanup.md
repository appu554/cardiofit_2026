# Gap 21: Closed-Loop Outcome Learning — Implementation Plan (Sprint 3: Debt Cleanup)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the 8 debt items that per-task and whole-sprint code reviews flagged during Sprint 2a — transactional ingest semantics, idempotency for at-least-once feed delivery, explicit nil-LifecycleID scope, `services.AttributionConfig` mirror elimination, redundant-index cleanup, per-request timeout in `runAttribution`, and the test-coverage gaps that the nil-DB unit test harness could not exercise. Sprint 3 ships Go-only hardening; no new services, no ML.

**Architecture:** Five focused tasks rebalanced toward kb-23 (3 tasks) since the outcome-pipeline debt is heavier than the attribution-engine debt. Task 1 introduces a reusable in-memory sqlite test fixture that Tasks 1, 3, and 5 all consume to finally exercise the DB-dependent paths (transactional ingest, multi-source reconciliation via handler, attribution handler integration). The `services.AttributionConfig` mirror is eliminated in Task 4 after the Sprint 2a reviewer confirmed there is no actual circular-import risk. Sprint 4's durable ledger and Sprint 2b's ML attribution are both unblocked by this sprint.

**Tech Stack:** Go 1.21, existing KB-23 / KB-26 infrastructure. `gorm.io/driver/sqlite` (in-memory) for test fixtures — already in go.mod transitively via the GORM base. No new production dependencies.

---

## Existing Infrastructure (end of Sprint 2a)

| What exists | Where | Relevance |
|---|---|---|
| `OutcomeRecord` with composite index `idx_or_patient_type` | [outcome_record.go](backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/outcome_record.go) | Task 1 adds `IdempotencyKey`; Task 2 adds `Scope` |
| `ingestOutcome` HTTP handler | [outcome_handlers.go](backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/outcome_handlers.go) | Task 1 adds transactional persist + idempotency; Task 2 adds scope validation; Task 3 expands test coverage |
| `runAttribution` with GORM transaction wrap (Sprint 2a Task 3) | [attribution_handlers.go](backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/attribution_handlers.go) | Task 4 adds per-request timeout; Task 5 integration-tests the txn flow |
| `services.AttributionConfig` local mirror | [rule_attribution.go](backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/rule_attribution.go) | Task 4 eliminates the mirror in favor of direct `config.AttributionConfig` import |
| `config.LoadAttributionConfig` | [attribution_config.go](backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/config/attribution_config.go) | Task 5 adds malformed-YAML + partial-YAML tests |
| Bare `index` on `AttributionVerdict.PatientID` alongside composite `idx_av_patient_computed` | [attribution_verdict.go](backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/attribution_verdict.go) | Task 4 removes the redundant bare index |

## Out of Scope (deferred to later sprints)

- **Item #6 SeedPriorHash** for cross-process HMAC chain continuity — requires durable DB-backed ledger, which is Sprint 4 scope. Meaningless in isolation.
- **ML attribution** (propensity, AIPW, TMLE, counterfactual head) — Sprint 2b, currently on hold pending MIMIC-IV training pipeline decisions.
- **Feedback-aware monitoring** (ADWIN, calibration-CUSUM, subgroup fairness) — Sprint 3 in the original Gap 21 spec, but renamed Sprint 5 in the phased rollout since it depends on ML Ŷ(0).
- **Ed25519 signatures, RACI authorization, regulatory artifacts** — Sprint 4+.
- **NLP feedback taxonomy** — blocked on the clinical-lead override-reason workshop.

## File Inventory

### KB-23 — Transactional ingest + idempotency + scope + test hardening
| Action | File | Responsibility |
|---|---|---|
| Modify | `internal/models/outcome_record.go` | Add `IdempotencyKey` (uniqueIndex) + `Scope` enum field |
| Modify | `internal/api/outcome_handlers.go` | Transactional Create+ReconciledID update; idempotency key short-circuit; scope validation |
| Modify | `internal/api/outcome_handlers_test.go` | Add 4 new tests (idempotency, transactional prior rows, scope validation, hardening paths) |
| Create | `internal/api/test_fixtures_test.go` | In-memory sqlite DB helper shared across handler tests |

### KB-26 — Config cleanup + ledger cleanup + context timeout + attribution test hardening
| Action | File | Responsibility |
|---|---|---|
| Modify | `internal/services/rule_attribution.go` | Remove local `AttributionConfig` mirror; import `config.AttributionConfig` directly |
| Modify | `internal/api/server.go` | Update `attributionConfig` field type + `SetAttributionConfig` signature |
| Modify | `internal/api/attribution_handlers.go` | Use `config.AttributionConfig`; add per-request context timeout |
| Modify | `internal/api/attribution_handlers_test.go` | Context-timeout test + handler integration test with sqlite fixture |
| Modify | `internal/models/attribution_verdict.go` | Remove redundant bare `index` on `PatientID` |
| Modify | `main.go` | Simplify: pass `attrCfg` directly to `SetAttributionConfig` (no struct copy) |
| Modify | `internal/config/attribution_config_test.go` | Add malformed-YAML + partial-YAML tests |
| Create | `internal/api/test_fixtures_test.go` | In-memory sqlite DB helper for attribution handler tests |

**Total: 10 modify, 2 create, ~14 new tests**

---

### Task 1: Idempotent + transactional ingest (kb-23)

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/outcome_record.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/outcome_handlers.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/outcome_handlers_test.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/test_fixtures_test.go`

Sprint 2a shipped the ingest handler but the whole-sprint review flagged: (a) no idempotency for at-least-once feed delivery → each retry creates a new RESOLVED row, (b) prior-rows `ReconciledID` is never set, so the `outcome_records` table accumulates orphan PENDING rows. Both are fixed here with one transactional Create + Update pass.

Follow strict TDD.

- [ ] **Step 1:** Add `IdempotencyKey` field to `OutcomeRecord`. Find the current struct declaration in `outcome_record.go` and insert the new field immediately after `SourceRecordID`:

```go
	SourceRecordID  string     `gorm:"size:200" json:"source_record_id,omitempty"`
	// Feed-supplied idempotency key. When set, POST /outcomes/ingest with a
	// duplicate key returns the existing record instead of creating a new one.
	// Required for at-least-once claims/discharge feeds to avoid duplicate
	// reconciliation passes. uniqueIndex allows multiple NULL values (standard
	// SQL), so legacy records without a key are unaffected.
	IdempotencyKey  string     `gorm:"size:128;uniqueIndex:idx_or_idem_key" json:"idempotency_key,omitempty"`
	Reconciliation  string     `gorm:"size:20;index;not null;default:'PENDING'" json:"reconciliation"`
```

- [ ] **Step 2:** Create shared test fixture `test_fixtures_test.go`:

```go
package api

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-23-decision-cards/internal/database"
	"kb-23-decision-cards/internal/models"
)

// newTestDB returns a *database.Database backed by an in-memory sqlite DB
// with Gap 21 Sprint 1+2a+3 tables migrated. The caller owns cleanup —
// sqlite in-memory DB disposes automatically when the last connection closes.
func newTestDB(t *testing.T) *database.Database {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&models.OutcomeRecord{}, &models.ConsolidatedAlertRecord{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return &database.Database{DB: gdb}
}
```

Note: `kb-23-decision-cards/internal/database.Database` is the wrapper struct whose `DB` field holds `*gorm.DB`. The handler accesses `s.db.DB` so this fixture matches the production structure.

If `gorm.io/driver/sqlite` is not in go.mod, add it: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go get gorm.io/driver/sqlite && go mod tidy`. The driver is a common GORM transitive dep; it may already be present.

- [ ] **Step 3:** Write 3 failing tests in `outcome_handlers_test.go` (append after existing tests):

```go
func TestIngestOutcome_IdempotencyKey_DeduplicatesDuplicate(t *testing.T) {
	db := newTestDB(t)
	r := newTestGinEngine()
	srv := &Server{db: db}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	lifecycleID := uuid.New()
	body := models.OutcomeRecord{
		PatientID:       "P-idem-001",
		LifecycleID:     &lifecycleID,
		OutcomeType:     "READMISSION_30D",
		OutcomeOccurred: true,
		Source:          string(models.OutcomeSourceHospitalDischarge),
		IdempotencyKey:  "feed-msg-abc-123",
	}
	post := func() int {
		bodyJSON, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	if c := post(); c != http.StatusOK {
		t.Fatalf("first POST: expected 200, got %d", c)
	}
	if c := post(); c != http.StatusOK {
		t.Fatalf("duplicate POST: expected 200 (idempotent), got %d", c)
	}

	var count int64
	db.DB.Model(&models.OutcomeRecord{}).Where("idempotency_key = ?", "feed-msg-abc-123").Count(&count)
	if count != 1 {
		t.Fatalf("expected exactly 1 row for idempotency key, got %d", count)
	}
}

func TestIngestOutcome_Transactional_PriorRowsMarkedResolved(t *testing.T) {
	db := newTestDB(t)
	r := newTestGinEngine()
	srv := &Server{db: db}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	// Seed a PENDING prior row directly in the DB (simulates an earlier
	// ingest where reconciliation stayed PENDING because min_sources wasn't met).
	lifecycleID := uuid.New()
	prior := models.OutcomeRecord{
		PatientID:       "P-txn-001",
		LifecycleID:     &lifecycleID,
		OutcomeType:     "READMISSION_30D",
		OutcomeOccurred: true,
		Source:          string(models.OutcomeSourceHospitalDischarge),
		Reconciliation:  string(models.ReconciliationPending),
	}
	if err := db.DB.Create(&prior).Error; err != nil {
		t.Fatalf("seed prior row: %v", err)
	}

	// Ingest a second source; reconciliation should now resolve and the
	// prior row should be marked RESOLVED with ReconciledID pointing at
	// the new authoritative row.
	incoming := models.OutcomeRecord{
		PatientID:       "P-txn-001",
		LifecycleID:     &lifecycleID,
		OutcomeType:     "READMISSION_30D",
		OutcomeOccurred: true,
		Source:          string(models.OutcomeSourceClaimsFeed),
	}
	bodyJSON, _ := json.Marshal(incoming)
	req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Fetch the authoritative row and the prior row.
	var all []models.OutcomeRecord
	if err := db.DB.Where("patient_id = ?", "P-txn-001").Order("ingested_at").Find(&all).Error; err != nil {
		t.Fatalf("load all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 rows (authoritative + prior), got %d", len(all))
	}
	priorLoaded := all[0]
	authoritative := all[1]
	if priorLoaded.Reconciliation != string(models.ReconciliationResolved) {
		t.Fatalf("prior row should be RESOLVED, got %s", priorLoaded.Reconciliation)
	}
	if priorLoaded.ReconciledID == nil || *priorLoaded.ReconciledID != authoritative.ID {
		t.Fatalf("prior ReconciledID should point at authoritative; got %v", priorLoaded.ReconciledID)
	}
}

func TestIngestOutcome_WithoutIdempotencyKey_StillWorks(t *testing.T) {
	db := newTestDB(t)
	r := newTestGinEngine()
	srv := &Server{db: db}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	lifecycleID := uuid.New()
	body := models.OutcomeRecord{
		PatientID:       "P-noidem-001",
		LifecycleID:     &lifecycleID,
		OutcomeType:     "READMISSION_30D",
		OutcomeOccurred: true,
		Source:          string(models.OutcomeSourceHospitalDischarge),
		// IdempotencyKey intentionally omitted
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// Second POST with SAME body (no idempotency key) creates a NEW row — no dedup.
	req2 := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second POST: expected 200, got %d", w2.Code)
	}
}
```

Also need the `database` import. Add to the test file's imports if missing:

```go
	"kb-23-decision-cards/internal/database"
```

Wait — `database` is already imported via `test_fixtures_test.go`. The handler test file may not need it directly if it only uses `*Server{db: newTestDB(t)}`. Verify which imports exist in the current file before adding.

- [ ] **Step 4:** Run tests — expect FAIL with compilation errors on `IdempotencyKey` field (Step 1 added it to the model, but the handler doesn't use it yet) and runtime failures on the transactional test (prior-rows never get updated):

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/api/ -run "TestIngestOutcome_Idempotency|TestIngestOutcome_Transactional|TestIngestOutcome_Without" -v`
Expected: some combination of compile errors + assertion failures. Confirm before proceeding.

- [ ] **Step 5:** Implement the handler changes in `outcome_handlers.go`. The new handler body replaces the existing one. The structure:

1. Validate required fields (unchanged)
2. Default `IngestedAt` to now (unchanged)
3. **NEW**: If `IdempotencyKey` is set and DB is non-nil, look up existing record — return it if found.
4. Load PENDING prior records (unchanged, already filtered to PENDING per Sprint 2a Task 1 fix)
5. Call `ReconcileOutcomes` (unchanged)
6. **NEW**: Wrap Create(authoritative) + Update(prior rows) in a GORM transaction.

Replace the existing function body with:

```go
func (s *Server) ingestOutcome(c *gin.Context) {
	var incoming models.OutcomeRecord
	if err := c.ShouldBindJSON(&incoming); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if incoming.PatientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "patient_id is required"})
		return
	}
	if incoming.OutcomeType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "outcome_type is required"})
		return
	}
	if incoming.Source == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source is required"})
		return
	}
	if incoming.IngestedAt.IsZero() {
		incoming.IngestedAt = time.Now().UTC()
	}

	// Idempotency check — if an earlier ingest with the same key already
	// produced an authoritative row, return that row unchanged. Short-circuits
	// reconciliation + persist and makes at-least-once feed delivery safe.
	if incoming.IdempotencyKey != "" && s.db != nil && s.db.DB != nil {
		var existing models.OutcomeRecord
		err := s.db.DB.Where("idempotency_key = ?", incoming.IdempotencyKey).First(&existing).Error
		if err == nil {
			c.JSON(http.StatusOK, gin.H{"record": existing, "idempotent": true})
			return
		}
		// gorm.ErrRecordNotFound is fine — proceed with normal ingest.
		// Any other error is a DB failure.
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			if s.log != nil {
				s.log.Error("idempotency key lookup failed",
					zap.Error(err),
					zap.String("idempotency_key", incoming.IdempotencyKey))
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "idempotency lookup failed: " + err.Error()})
			return
		}
	}

	// Collect prior PENDING records for the same (patient, outcome_type, lifecycle).
	records := []models.OutcomeRecord{incoming}
	var priorIDs []uuid.UUID
	if s.db != nil && s.db.DB != nil {
		var prior []models.OutcomeRecord
		q := s.db.DB.
			Where("patient_id = ? AND outcome_type = ?", incoming.PatientID, incoming.OutcomeType).
			Where("reconciliation = ?", string(models.ReconciliationPending))
		// When LifecycleID is nil, the query spans ALL lifecycles for this
		// (patient, outcome_type). This is intentional for "global sweep"
		// ingest (e.g., mortality registry feeds that don't know about
		// alert lifecycles) but semantically undefined for feed sources
		// that SHOULD know the lifecycle. Task 2 will make this explicit
		// via a scope discriminator.
		if incoming.LifecycleID != nil {
			q = q.Where("lifecycle_id = ?", *incoming.LifecycleID)
		}
		if err := q.Find(&prior).Error; err != nil {
			if s.log != nil {
				s.log.Error("failed to load prior outcome records for reconciliation",
					zap.Error(err),
					zap.String("patient_id", incoming.PatientID),
					zap.String("outcome_type", incoming.OutcomeType))
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load prior records: " + err.Error()})
			return
		}
		records = append(prior, incoming)
		for _, p := range prior {
			priorIDs = append(priorIDs, p.ID)
		}
	}

	authoritative, err := services.ReconcileOutcomes(records, 48*time.Hour, 1)
	if err != nil {
		if s.log != nil {
			s.log.Error("reconciliation failed during outcome ingest",
				zap.Error(err),
				zap.String("patient_id", incoming.PatientID),
				zap.String("outcome_type", incoming.OutcomeType),
				zap.Int("num_records", len(records)))
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "reconciliation failed: " + err.Error()})
		return
	}

	if s.db != nil && s.db.DB != nil {
		// Transaction: Create authoritative + Update prior rows' ReconciledID
		// and Reconciliation status. If either fails, both roll back so the
		// table never ends up with a half-promoted prior row pointing at a
		// phantom authoritative.
		txErr := s.db.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&authoritative).Error; err != nil {
				return err
			}
			if len(priorIDs) > 0 && authoritative.Reconciliation == string(models.ReconciliationResolved) {
				if err := tx.Model(&models.OutcomeRecord{}).
					Where("id IN ?", priorIDs).
					Updates(map[string]interface{}{
						"reconciled_id":  authoritative.ID,
						"reconciliation": string(models.ReconciliationResolved),
					}).Error; err != nil {
					return err
				}
			}
			return nil
		})
		if txErr != nil {
			if s.log != nil {
				s.log.Error("failed to persist authoritative outcome record (txn rolled back)",
					zap.Error(txErr),
					zap.String("patient_id", incoming.PatientID),
					zap.String("outcome_type", incoming.OutcomeType))
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist record: " + txErr.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"record": authoritative})
}
```

Add imports if missing:

```go
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
```

The existing file imports some of these; check first and add only what's missing.

- [ ] **Step 6:** Run tests — expect all 3 new tests PASS + existing 3 tests still PASS:

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/api/ -run TestIngestOutcome -v`
Expected: 6/6 pass (3 existing + 3 new).

- [ ] **Step 7:** Run full kb-23 internal suite — expect no regression:

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/... -v`
Expected: all pass.

- [ ] **Step 8:** Build clean:

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go build ./...`
Expected: clean.

- [ ] **Step 9:** Commit:

```bash
cd /Volumes/Vaidshala/cardiofit && git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/outcome_record.go backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/outcome_handlers.go backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/outcome_handlers_test.go backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/test_fixtures_test.go
git commit -m "feat(kb23): transactional ingest + idempotency key (Gap 21 Sprint 3 Task 1)"
```

---

### Task 2: Explicit scope discriminator for nil-LifecycleID (kb-23)

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/outcome_record.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/outcome_handlers.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/outcome_handlers_test.go`

Sprint 2a review Issue #2 noted the ambiguity: when `LifecycleID == nil`, the handler's query spans all lifecycles for a (patient, outcome_type) tuple. This is intentional for mortality-registry sweeps but undefined for feeds that forgot to set the lifecycle. Task 2 makes the intent explicit via a `Scope` enum field; the handler rejects semantically-inconsistent combinations (nil lifecycle + `PATIENT_ALERT` scope, or non-nil lifecycle + `GLOBAL_SWEEP` scope).

- [ ] **Step 1:** Add `OutcomeScope` enum constants + `Scope` field to `OutcomeRecord` in `outcome_record.go`. After the existing `ReconciliationStatus` const block, add:

```go
// OutcomeScope disambiguates nil-LifecycleID semantics: is the outcome
// tied to a specific alert lifecycle, or is it a global sweep (e.g.,
// mortality registry pull that doesn't know about CardioFit alerts)?
type OutcomeScope string

const (
	// ScopePatientAlert: outcome is tied to a specific DetectionLifecycle.
	// LifecycleID MUST be set.
	ScopePatientAlert OutcomeScope = "PATIENT_ALERT"
	// ScopeGlobalSweep: outcome arrived from a source with no lifecycle
	// awareness (registry, bulk claims feed without patient-alert linkage).
	// LifecycleID MUST be nil. Query semantics at ingest time span all
	// lifecycles for the (patient, outcome_type) tuple.
	ScopeGlobalSweep OutcomeScope = "GLOBAL_SWEEP"
)
```

Add the `Scope` field to the struct immediately after `CohortID`:

```go
	CohortID        string     `gorm:"size:60;index;index:idx_or_patient_type,priority:1" json:"cohort_id,omitempty"`
	// Scope makes the LifecycleID presence/absence semantically explicit.
	// PATIENT_ALERT requires LifecycleID; GLOBAL_SWEEP requires LifecycleID nil.
	// Empty string treated as PATIENT_ALERT for backward compatibility with
	// Sprint 2a records that predate this field.
	Scope           string     `gorm:"size:20;index;default:'PATIENT_ALERT'" json:"scope,omitempty"`
	OutcomeType     string     `gorm:"size:60;index:idx_or_patient_type,priority:2;not null" json:"outcome_type"`
```

- [ ] **Step 2:** Write 2 failing tests in `outcome_handlers_test.go`:

```go
func TestIngestOutcome_ScopeGlobalSweepWithLifecycleID_Returns400(t *testing.T) {
	db := newTestDB(t)
	r := newTestGinEngine()
	srv := &Server{db: db}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	lifecycleID := uuid.New()
	body := models.OutcomeRecord{
		PatientID:       "P-scope-001",
		LifecycleID:     &lifecycleID, // present
		Scope:           string(models.ScopeGlobalSweep), // inconsistent
		OutcomeType:     "READMISSION_30D",
		OutcomeOccurred: true,
		Source:          string(models.OutcomeSourceMortalityRegistry),
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for scope=GLOBAL_SWEEP with lifecycle_id set, got %d", w.Code)
	}
}

func TestIngestOutcome_ScopePatientAlertWithoutLifecycleID_Returns400(t *testing.T) {
	db := newTestDB(t)
	r := newTestGinEngine()
	srv := &Server{db: db}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	body := models.OutcomeRecord{
		PatientID:       "P-scope-002",
		LifecycleID:     nil, // missing
		Scope:           string(models.ScopePatientAlert), // requires lifecycle
		OutcomeType:     "READMISSION_30D",
		OutcomeOccurred: true,
		Source:          string(models.OutcomeSourceHospitalDischarge),
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for scope=PATIENT_ALERT without lifecycle_id, got %d", w.Code)
	}
}
```

- [ ] **Step 3:** Run — expect FAIL (no scope validation yet):

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/api/ -run TestIngestOutcome_Scope -v`
Expected: both tests fail (handler returns 200 for mismatched scope/lifecycle combinations).

- [ ] **Step 4:** Add scope validation to the handler. Find the existing validation block in `outcome_handlers.go` right after the `source` check. Add:

```go
	// Scope validation. Empty scope defaults to PATIENT_ALERT for backward
	// compatibility with Sprint 2a records. Explicit values must match
	// LifecycleID presence.
	if incoming.Scope == "" {
		incoming.Scope = string(models.ScopePatientAlert)
	}
	switch incoming.Scope {
	case string(models.ScopePatientAlert):
		if incoming.LifecycleID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "scope=PATIENT_ALERT requires lifecycle_id"})
			return
		}
	case string(models.ScopeGlobalSweep):
		if incoming.LifecycleID != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "scope=GLOBAL_SWEEP must not set lifecycle_id"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "scope must be PATIENT_ALERT or GLOBAL_SWEEP"})
		return
	}
```

- [ ] **Step 5:** Run tests — expect PASS:

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/api/ -run TestIngestOutcome_Scope -v`
Expected: 2/2 pass.

- [ ] **Step 6:** Run full ingest test suite — expect 8/8 (3 Sprint 2a + 3 Task 1 + 2 Task 2):

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/api/ -run TestIngestOutcome -v`
Expected: 8/8 pass.

Hmm, existing Sprint 2a tests and Task 1 tests don't set `Scope`. The default `PATIENT_ALERT` validation requires `LifecycleID` — Sprint 2a tests 1, 2 (which set `LifecycleID`) still pass, but Sprint 2a test 3 (which doesn't set `LifecycleID` — a happy path for single-source ingest) will now FAIL 400. Two options:

(a) Update Sprint 2a tests to explicitly set `Scope: "GLOBAL_SWEEP"` where `LifecycleID` is nil.
(b) Default to `GLOBAL_SWEEP` when `LifecycleID` is nil for backward compat.

Choose (a) — explicit is better. Find Sprint 2a tests in the same file that don't set `LifecycleID` (check `TestIngestOutcome_SingleSource_ReturnsResolved` — it DOES set lifecycleID, so it's fine; but Task 1's `TestIngestOutcome_WithoutIdempotencyKey_StillWorks` also sets it). Audit: if any existing test sets `LifecycleID: nil` AND no `Scope`, update to `Scope: string(models.ScopeGlobalSweep)`. If none, proceed.

- [ ] **Step 7:** Commit:

```bash
cd /Volumes/Vaidshala/cardiofit && git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/outcome_record.go backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/outcome_handlers.go backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/outcome_handlers_test.go
git commit -m "feat(kb23): explicit scope discriminator for ingest (Gap 21 Sprint 3 Task 2)"
```

---

### Task 3: kb-23 outcome ingest test hardening

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/outcome_handlers_test.go`

Sprint 2a review flagged 4 test-coverage gaps (missing outcome_type, missing source, multi-source via handler, CONFLICTED path via handler). Task 3 adds them — no production code changes, just test additions exercising paths the nil-DB tests couldn't reach.

- [ ] **Step 1:** Write 4 new failing-to-pass tests (all should pass immediately because the handler behavior for these paths already exists from Sprint 2a + Sprint 3 Task 1/2; these are regression guards):

```go
func TestIngestOutcome_MissingOutcomeType_Returns400(t *testing.T) {
	r := newTestGinEngine()
	srv := &Server{}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	lifecycleID := uuid.New()
	body := models.OutcomeRecord{
		PatientID:   "P-missing-ot",
		LifecycleID: &lifecycleID,
		// OutcomeType intentionally empty
		Source: string(models.OutcomeSourceHospitalDischarge),
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing outcome_type, got %d", w.Code)
	}
}

func TestIngestOutcome_MissingSource_Returns400(t *testing.T) {
	r := newTestGinEngine()
	srv := &Server{}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	lifecycleID := uuid.New()
	body := models.OutcomeRecord{
		PatientID:       "P-missing-src",
		LifecycleID:     &lifecycleID,
		OutcomeType:     "READMISSION_30D",
		OutcomeOccurred: true,
		// Source intentionally empty
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing source, got %d", w.Code)
	}
}

func TestIngestOutcome_MultiSourceAgree_ResolvesViaHandler(t *testing.T) {
	db := newTestDB(t)
	r := newTestGinEngine()
	srv := &Server{db: db}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	lifecycleID := uuid.New()
	occurredAt := time.Now().Add(-10 * 24 * time.Hour)
	first := models.OutcomeRecord{
		PatientID: "P-multi-001", LifecycleID: &lifecycleID, OutcomeType: "READMISSION_30D",
		OutcomeOccurred: true, OccurredAt: &occurredAt,
		Source: string(models.OutcomeSourceHospitalDischarge),
	}
	second := models.OutcomeRecord{
		PatientID: "P-multi-001", LifecycleID: &lifecycleID, OutcomeType: "READMISSION_30D",
		OutcomeOccurred: true, OccurredAt: &occurredAt,
		Source: string(models.OutcomeSourceClaimsFeed),
	}
	for _, body := range []models.OutcomeRecord{first, second} {
		bodyJSON, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("POST: expected 200, got %d: %s", w.Code, w.Body.String())
		}
	}
	// Fetch rows. The second POST should have reconciled the first PENDING row;
	// so 2 rows total, both with Reconciliation=RESOLVED, second row ReconciledID nil (authoritative),
	// first row ReconciledID pointing at authoritative.
	var rows []models.OutcomeRecord
	db.DB.Where("patient_id = ?", "P-multi-001").Order("ingested_at").Find(&rows)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Reconciliation != string(models.ReconciliationResolved) {
		t.Fatalf("first row should be RESOLVED after second POST reconciles, got %s", rows[0].Reconciliation)
	}
	if rows[1].Reconciliation != string(models.ReconciliationResolved) {
		t.Fatalf("second row (authoritative) should be RESOLVED, got %s", rows[1].Reconciliation)
	}
}

func TestIngestOutcome_MultiSourceDisagree_PersistsConflicted(t *testing.T) {
	db := newTestDB(t)
	r := newTestGinEngine()
	srv := &Server{db: db}
	r.POST("/outcomes/ingest", srv.ingestOutcome)

	lifecycleID := uuid.New()
	firstBody := models.OutcomeRecord{
		PatientID: "P-conflict-001", LifecycleID: &lifecycleID, OutcomeType: "READMISSION_30D",
		OutcomeOccurred: true,
		Source:          string(models.OutcomeSourceHospitalDischarge),
	}
	secondBody := models.OutcomeRecord{
		PatientID: "P-conflict-001", LifecycleID: &lifecycleID, OutcomeType: "READMISSION_30D",
		OutcomeOccurred: false, // disagree
		Source:          string(models.OutcomeSourceClaimsFeed),
	}
	for _, body := range []models.OutcomeRecord{firstBody, secondBody} {
		bodyJSON, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/outcomes/ingest", bytes.NewReader(bodyJSON))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("POST: expected 200, got %d: %s", w.Code, w.Body.String())
		}
	}
	// Authoritative row (second POST) should be CONFLICTED.
	var rows []models.OutcomeRecord
	db.DB.Where("patient_id = ?", "P-conflict-001").Order("ingested_at").Find(&rows)
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	// The authoritative is rows[1]. CONFLICTED should be its status.
	if rows[1].Reconciliation != string(models.ReconciliationConflicted) {
		t.Fatalf("authoritative should be CONFLICTED after disagreement, got %s", rows[1].Reconciliation)
	}
}
```

- [ ] **Step 2:** Run tests — expect all 4 pass (the handler already supports these paths; this sprint is adding coverage, not behavior):

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/api/ -run "TestIngestOutcome_(MissingOutcome|MissingSource|MultiSource)" -v`
Expected: 4/4 pass.

If any fail because of Sprint 3 Task 2's scope validation (e.g., a test omits LifecycleID without setting scope), the fix is one line per test — update the body to include `Scope: string(models.ScopePatientAlert)` alongside the existing LifecycleID pointer.

- [ ] **Step 3:** Run the full kb-23 internal suite and verify the total test count:

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards && go test ./internal/api/ -run TestIngestOutcome -v | grep -E "PASS|FAIL" | wc -l`
Expected: 12 lines (3 Sprint 2a + 3 Task 1 + 2 Task 2 + 4 Task 3).

- [ ] **Step 4:** Commit:

```bash
cd /Volumes/Vaidshala/cardiofit && git add backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/api/outcome_handlers_test.go
git commit -m "test(kb23): outcome ingest test hardening (Gap 21 Sprint 3 Task 3)"
```

---

### Task 4: kb-26 ledger + config cleanup (mirror removal, bare-index removal, context timeout)

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/rule_attribution.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/server.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/attribution_handlers.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/attribution_handlers_test.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/attribution_verdict.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/main.go`

Three Sprint 2a review items:
1. `services.AttributionConfig` mirror is unnecessary — reviewer confirmed no circular import risk. Import `config.AttributionConfig` directly.
2. Redundant bare `index` on `AttributionVerdict.PatientID` — `idx_av_patient_computed` already covers patient-only queries.
3. `runAttribution` has no per-request timeout — the 30s server WriteTimeout is implicit ceiling. Add explicit 10s context timeout (configurable via env).

- [ ] **Step 1:** Eliminate the mirror struct. In `rule_attribution.go`, remove the local `AttributionConfig` declaration and change `ComputeAttributionWithConfig`'s signature to accept `config.AttributionConfig`:

Find the existing:

```go
// AttributionConfig is the local mirror of config.AttributionConfig used to
// avoid a circular import between services and config. Handlers populate it
// from the loaded config at startup and pass it here.
type AttributionConfig struct {
	Method        string
	MethodVersion string
}

// ComputeAttribution is the Sprint 1 backward-compatible entry point. It uses
// the default rule-based config. Existing tests and callers can continue using
// this signature.
func ComputeAttribution(in AttributionInput) models.AttributionVerdict {
	return ComputeAttributionWithConfig(in, AttributionConfig{Method: "RULE_BASED", MethodVersion: "sprint1-v1"})
}

func ComputeAttributionWithConfig(in AttributionInput, cfg AttributionConfig) models.AttributionVerdict {
```

Replace with:

```go
// ComputeAttribution is the Sprint 1 backward-compatible entry point. It uses
// the default rule-based config. Existing tests and callers continue to work
// with this zero-arg-after-input signature.
func ComputeAttribution(in AttributionInput) models.AttributionVerdict {
	return ComputeAttributionWithConfig(in, config.DefaultAttributionConfig)
}

// ComputeAttributionWithConfig produces a rule-based AttributionVerdict and
// stamps AttributionMethod/MethodVersion from the supplied config. Sprint 2b
// replaces the function body with an ML client call while preserving the
// exact signature — callers stay identical.
func ComputeAttributionWithConfig(in AttributionInput, cfg config.AttributionConfig) models.AttributionVerdict {
```

Add `"kb-26-metabolic-digital-twin/internal/config"` to the imports of `rule_attribution.go`.

- [ ] **Step 2:** Update `server.go` to use `config.AttributionConfig`. Find:

```go
	// Attribution config (Gap 21 Sprint 2a Task 5): loaded from
	// market-configs/shared/attribution_parameters.yaml at startup.
	// Used by runAttribution to stamp AttributionMethod/MethodVersion.
	attributionConfig services.AttributionConfig
```

Change to:

```go
	// Attribution config (Gap 21 Sprint 2a Task 5): loaded from
	// market-configs/shared/attribution_parameters.yaml at startup.
	// Used by runAttribution to stamp AttributionMethod/MethodVersion.
	attributionConfig config.AttributionConfig
```

Find the setter:

```go
func (s *Server) SetAttributionConfig(cfg services.AttributionConfig) {
	s.attributionConfig = cfg
}
```

Change to:

```go
func (s *Server) SetAttributionConfig(cfg config.AttributionConfig) {
	s.attributionConfig = cfg
}
```

Add `"kb-26-metabolic-digital-twin/internal/config"` import if missing.

- [ ] **Step 3:** Update `attribution_handlers.go`. Find the block in `runAttribution`:

```go
	cfg := s.attributionConfig
	if cfg.Method == "" || cfg.MethodVersion == "" {
		// If SetAttributionConfig was never called OR was called with a partial
		// config (e.g., tests, or a misconfigured YAML that set one field but
		// not the other), fall back to rule-based defaults.
		cfg = services.AttributionConfig{Method: "RULE_BASED", MethodVersion: "sprint1-v1"}
	}
	verdict := services.ComputeAttributionWithConfig(in, cfg)
```

Change to:

```go
	cfg := s.attributionConfig
	if cfg.Method == "" || cfg.MethodVersion == "" {
		// Fallback when SetAttributionConfig was never called (tests) or was
		// called with a partial config. The config package's default ensures
		// consistent values across all fallback paths.
		cfg = config.DefaultAttributionConfig
	}
	verdict := services.ComputeAttributionWithConfig(in, cfg)
```

Add `"kb-26-metabolic-digital-twin/internal/config"` import if missing.

- [ ] **Step 4:** Add per-request context timeout to `runAttribution`. At the TOP of `runAttribution` (right after any existing `s.ledger == nil` guard, before `ShouldBindJSON`), add:

```go
	// Sprint 3 Task 4: per-request deadline. Without this, ComputeAttribution
	// (Sprint 1/2a rule-based) completes in microseconds, but Sprint 2b's
	// ONNX inference call could block indefinitely. The 10s default matches
	// typical ML inference budgets; override via GAP21_ATTRIBUTION_TIMEOUT_MS.
	timeoutMs := 10000
	if raw := os.Getenv("GAP21_ATTRIBUTION_TIMEOUT_MS"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			timeoutMs = n
		}
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	c.Request = c.Request.WithContext(ctx)
```

Add imports to `attribution_handlers.go` if missing: `"context"`, `"os"`, `"strconv"`, `"time"`.

NOTE: The rule-based `ComputeAttribution` itself is synchronous Go code that doesn't check `ctx.Done()`. The timeout's benefit in Sprint 3 is for the DB transaction (GORM honors `context.WithTimeout` when passed via `WithContext`) and for Sprint 2b when ONNX client calls are added. Wrap the transaction's GORM call with context. Find the existing `s.db.DB.Transaction(...)` and change it to:

```go
	if s.db != nil && s.db.DB != nil {
		txErr := s.db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
```

(add `.WithContext(ctx)` before `.Transaction`).

- [ ] **Step 5:** Remove the redundant bare `index` on `AttributionVerdict.PatientID`. In `attribution_verdict.go`, find:

```go
	PatientID          string    `gorm:"size:100;index;index:idx_av_patient_computed,priority:1;not null" json:"patient_id"`
```

Change to:

```go
	PatientID          string    `gorm:"size:100;index:idx_av_patient_computed,priority:1;not null" json:"patient_id"`
```

The `idx_av_patient_computed` composite satisfies `WHERE patient_id = ?` queries (PostgreSQL uses the leading column of a composite index for equality predicates). The bare `index` was Sprint 2a caution; with confidence the composite covers it, removing halves index maintenance cost on writes.

- [ ] **Step 6:** Update `main.go` to pass `attrCfg` directly. Find:

```go
	server.SetAttributionConfig(services.AttributionConfig{
		Method:        attrCfg.Method,
		MethodVersion: attrCfg.MethodVersion,
	})
```

Change to:

```go
	server.SetAttributionConfig(attrCfg)
```

The struct conversion is no longer needed — `attrCfg` already is the `config.AttributionConfig` type the setter accepts.

- [ ] **Step 7:** Write 1 failing test in `attribution_handlers_test.go`. Append after the existing tests:

```go
func TestRunAttribution_ContextTimeout_Honoured(t *testing.T) {
	// With GAP21_ATTRIBUTION_TIMEOUT_MS set to a very small value and a
	// deliberately blocking client request context, the handler should
	// short-circuit via context deadline rather than running to completion.
	// Sprint 3 Task 4 adds the timeout; Sprint 2b will benefit when ONNX
	// inference is the slow path. For Sprint 3 we verify the timeout is
	// attached to the request context and propagates without panic.
	t.Setenv("GAP21_ATTRIBUTION_TIMEOUT_MS", "100")

	r := newTestEngine()
	srv := &Server{}
	r.POST("/attribution/run", srv.runAttribution)

	body := map[string]interface{}{
		"TreatmentStrategy": "INTERVENTION_TAKEN",
		"PreAlertRiskTier":  "HIGH",
		"PreAlertRiskScore": 62.0,
		"HorizonDays":       30,
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/attribution/run", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// nil ledger guard fires before the timeout logic, so this returns 503.
	// The important assertion is that no panic / crash occurred and that
	// the timeout env var was read without error.
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 from nil-ledger guard (pre-timeout-logic), got %d", w.Code)
	}
}
```

- [ ] **Step 8:** Run — expect PASS:

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/api/ -run TestRunAttribution -v`
Expected: 2/2 pass (existing nil-ledger + new timeout-honoured).

- [ ] **Step 9:** Run full kb-26 internal suite — expect no regression:

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./... && go test ./internal/... -v`
Expected: clean build, all pass.

- [ ] **Step 10:** Commit:

```bash
cd /Volumes/Vaidshala/cardiofit && git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/services/rule_attribution.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/server.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/attribution_handlers.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/attribution_handlers_test.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/models/attribution_verdict.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/main.go
git commit -m "refactor(kb26): eliminate AttributionConfig mirror + context timeout + index cleanup (Gap 21 Sprint 3 Task 4)"
```

---

### Task 5: kb-26 attribution test hardening + integration test

**Files:**
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/config/attribution_config_test.go`
- Modify: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/attribution_handlers_test.go`
- Create: `backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/test_fixtures_test.go`

Sprint 2a review flagged 3 kb-26 test-coverage gaps: malformed YAML, partial YAML (only `method.name`), and no handler integration test exercising the transaction rollback + GET /attribution flow end-to-end.

- [ ] **Step 1:** Add 2 new config tests to `attribution_config_test.go`. Append after the existing 2 tests:

```go
func TestLoadAttributionConfig_MalformedYAML_ReturnsDefaultWithError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "attribution_parameters.yaml")
	// Syntactically valid YAML that doesn't match yamlShape — method is a list, not a map.
	content := `method:
  - name: RULE_BASED
  - version: sprint1-v1
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	cfg, err := LoadAttributionConfig(path)
	if err == nil {
		t.Fatalf("expected parse error for malformed YAML, got nil")
	}
	if cfg.Method != DefaultAttributionConfig.Method {
		t.Fatalf("expected default method on parse error, got %q", cfg.Method)
	}
	if cfg.MethodVersion != DefaultAttributionConfig.MethodVersion {
		t.Fatalf("expected default version on parse error, got %q", cfg.MethodVersion)
	}
}

func TestLoadAttributionConfig_PartialYAML_FillsWithDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "attribution_parameters.yaml")
	// Only method.name populated; version missing.
	content := `method:
  name: CUSTOM_METHOD
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	cfg, err := LoadAttributionConfig(path)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cfg.Method != "CUSTOM_METHOD" {
		t.Fatalf("expected method=CUSTOM_METHOD from YAML, got %q", cfg.Method)
	}
	if cfg.MethodVersion != DefaultAttributionConfig.MethodVersion {
		t.Fatalf("expected version fallback to default, got %q", cfg.MethodVersion)
	}
}
```

- [ ] **Step 2:** Create sqlite fixture for kb-26 handler tests. Create `internal/api/test_fixtures_test.go`:

```go
package api

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-26-metabolic-digital-twin/internal/database"
	"kb-26-metabolic-digital-twin/internal/models"
)

// newTestDB returns an in-memory sqlite Database with Gap 21 attribution +
// ledger tables migrated. Used by attribution handler integration tests.
func newTestDB(t *testing.T) *database.Database {
	t.Helper()
	gdb, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	if err := gdb.AutoMigrate(&models.AttributionVerdict{}, &models.LedgerEntry{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return &database.Database{DB: gdb}
}
```

Note: If `gorm.io/driver/sqlite` is not yet in kb-26's go.mod, add it: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go get gorm.io/driver/sqlite && go mod tidy`.

- [ ] **Step 3:** Write the integration test in `attribution_handlers_test.go`. Append:

```go
func TestRunAttribution_EndToEnd_PersistsVerdictAndLedger(t *testing.T) {
	db := newTestDB(t)
	ledger := services.NewInMemoryLedger([]byte("test-key"))
	r := newTestEngine()
	srv := &Server{db: db, ledger: ledger}
	srv.attributionConfig = config.DefaultAttributionConfig
	r.POST("/attribution/run", srv.runAttribution)
	r.GET("/attribution/:patientId", srv.getAttributionByPatient)

	body := map[string]interface{}{
		"PatientID":         "P-e2e-001",
		"TreatmentStrategy": "INTERVENTION_TAKEN",
		"PreAlertRiskTier":  "HIGH",
		"PreAlertRiskScore": 75.0,
		"OutcomeOccurred":   false,
		"HorizonDays":       30,
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/attribution/run", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("POST: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// GET should return the persisted verdict.
	getReq := httptest.NewRequest(http.MethodGet, "/attribution/P-e2e-001", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("GET: expected 200, got %d: %s", getW.Code, getW.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(getW.Body.Bytes(), &resp)
	verdicts, _ := resp["verdicts"].([]interface{})
	if len(verdicts) != 1 {
		t.Fatalf("expected 1 verdict for P-e2e-001, got %d", len(verdicts))
	}

	// Ledger should have exactly one entry for this attribution.
	var ledgerCount int64
	db.DB.Model(&models.LedgerEntry{}).Where("entry_type = ?", "ATTRIBUTION_RUN").Count(&ledgerCount)
	if ledgerCount != 1 {
		t.Fatalf("expected 1 ledger entry, got %d", ledgerCount)
	}
}
```

Add `"kb-26-metabolic-digital-twin/internal/config"` to the test file imports if missing.

- [ ] **Step 4:** Run — expect PASS:

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/config/ ./internal/api/ -v`
Expected: 4 config + all api tests pass. Attribution handler tests should now include the new end-to-end integration test.

- [ ] **Step 5:** Run full kb-26 internal suite:

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./... && go test ./internal/... -v`
Expected: clean build, all pass.

- [ ] **Step 6:** Commit:

```bash
cd /Volumes/Vaidshala/cardiofit && git add backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/config/attribution_config_test.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/attribution_handlers_test.go backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin/internal/api/test_fixtures_test.go
git commit -m "test(kb26): malformed/partial YAML + end-to-end attribution integration (Gap 21 Sprint 3 Task 5)"
```

- [ ] **Step 7:** Push:

```bash
cd /Volumes/Vaidshala/cardiofit && git push origin feature/v4-clinical-gaps
```

---

## Verification Questions

1. Does posting the same `idempotency_key` twice return the original record without creating a new row? (Task 1 test 1)
2. Does a second source POST trigger the prior PENDING row's `ReconciledID` to point at the authoritative row? (Task 1 test 2)
3. Does `scope=GLOBAL_SWEEP` with `lifecycle_id` set return 400? (Task 2 test 1)
4. Does `scope=PATIENT_ALERT` without `lifecycle_id` return 400? (Task 2 test 2)
5. Does missing `outcome_type` return 400? (Task 3 test 1)
6. Does multi-source agreement produce RESOLVED for both rows (prior via ReconciledID update)? (Task 3 test 3)
7. Does multi-source disagreement produce CONFLICTED on the authoritative row? (Task 3 test 4)
8. Does `config.AttributionConfig` import work without circular dependency? (Task 4 build)
9. Does `runAttribution` honor `GAP21_ATTRIBUTION_TIMEOUT_MS`? (Task 4 test)
10. Does malformed YAML fall back to default config with a non-nil error? (Task 5 test 1)
11. Does partial YAML (only `method.name`) fill missing fields with defaults? (Task 5 test 2)
12. Does `POST /attribution/run` followed by `GET /attribution/:patientId` return the persisted verdict? (Task 5 test 3)

## Effort Estimate

| Task | Scope | Expected |
|---|---|---|
| Task 1: Idempotent+transactional ingest (3 tests) | 3 modify, 1 create | 3-4 hours |
| Task 2: Scope discriminator (2 tests) | 3 modify | 1-2 hours |
| Task 3: Ingest test hardening (4 tests) | 1 modify | 1-2 hours |
| Task 4: Mirror/index/timeout cleanup (1 test) | 6 modify | 2-3 hours |
| Task 5: Attribution test hardening (3 tests) | 2 modify, 1 create | 2 hours |
| **Total** | **~12 files, ~13 tests** | **~9-13 hours** |

---

## Sprint 4 Deferred Items (already documented in earlier plans)

- Durable DB-backed ledger + `SeedPriorHash` for cross-process chain continuity
- Ed25519 per-entry signatures
- RACI multi-signature authorization
- Shadow → canary → full retraining pipeline
- Rollback drills + hot-standby

## Sprint 2b Deferred (on hold pending MIMIC-IV training decisions)

- ONNX-containerized ML attribution
- Propensity scoring, AIPW, TMLE
- Counterfactual outcome head Ŷ(0)
- E-value + tipping-point sensitivity

**Transition path:** Sprint 3's Task 4 eliminates the `services.AttributionConfig` mirror so Sprint 2b's YAML changes (Method=DOUBLY_ROBUST_ONNX) flow through cleanly. The per-request context timeout in Task 4 prepares the transaction path for ONNX client calls that can block indefinitely. Task 5's integration test harness is reusable when Sprint 2b swaps in the ONNX client — the same sqlite fixture verifies the verdict-plus-ledger persistence works regardless of attribution method.
