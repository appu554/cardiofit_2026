# Masked HTN Phase 3 — Daily Batch Trigger Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wire the Phase 1+2 BP context classifier to a daily scheduled batch that classifies all active patients, persists a snapshot per (patient, day), and publishes phenotype-change events to KB-19. The existing HTTP endpoint stays as the on-demand override.

**Architecture:** A new `BatchScheduler` in KB-26 owns a goroutine that wakes once per day at a configured UTC hour, queries the new `ListActivePatients(window)` repository method (returning patients with `twin_state.UpdatedAt` within the last 30 days), and processes each through a worker pool of size 10. The orchestrator gains an `EmitPhenotypeEvents` flag that, when set, fetches the prior snapshot before saving the new one and POSTs `MASKED_HTN_DETECTED` or `BP_PHENOTYPE_CHANGED` to KB-19 via a new `KB19Client` in `internal/clients/`. Top-level shutdown is propagated via a new `context.WithCancel` in `main.go`. The scheduler is generic: a single batch job interface (`BatchJob`) makes it possible for `quarterly_aggregator` (currently dead-wired in main.go) to use the same scheduler later — but Phase 3 only registers the BP context job.

**Tech Stack:** Go 1.25 (Gin, GORM, Zap, Prometheus), PostgreSQL 15

**Pre-requisite:** Phase 2 commits `919c0fb1` through `4932e56f` are present. Both KB-26 and KB-23 build and test green.

---

## Locked Decisions

These are NOT open questions in this plan — they are fixed constraints based on the trigger design discussion and codebase exploration.

### Decision 1: Trigger design = Design B (daily batch + existing HTTP override)
- One classification per active patient per day at 02:00 UTC
- Existing `POST /api/v1/kb26/bp-context/:patientId` stays as the manual refresh path

### Decision 2: Active patient definition = `twin_state.UpdatedAt > NOW() - 30 days`
- Aligns with KB-21's `DORMANT` definition
- Bounds batch runtime predictably
- Patients who resume signaling re-enter the batch automatically

### Decision 3: Event publication owned by KB-26 (not KB-23)
- The orchestrator is the only place that knows both prior and new phenotype simultaneously
- A new lightweight `KB19Client` lives in `kb-26-metabolic-digital-twin/internal/clients/kb19_client.go`
- The KB-23-side `PublishMaskedHTNDetected` and `PublishPhenotypeChanged` methods (Phase 2 commit `4932e56f`) become dead code paths
  - **They are NOT deleted** — destructive removal is out of scope; they stay as future-extension hooks
  - A code comment on each is added pointing to KB-26 as the canonical publisher

### Decision 4: Scheduler is generic, hosts multiple `BatchJob`s
- A single `BatchJob` interface decouples job logic from scheduling
- Phase 3 registers exactly one job: `BPContextDailyBatch`
- The `_ = quarterlyAggregator` dead reference in `main.go` becomes a TODO comment noting it can be wrapped in a `BatchJob` in a future plan
- This is YAGNI-resistant: the interface is dead-simple (3 methods) and the alternative — a one-off scheduler hardcoded to BP context — would have to be rewritten the moment a second batch job arrives

### Decision 5: Index addition for cross-patient query
- A new single-column index `idx_twin_state_updated_at` is added to the `TwinState` GORM model
- AutoMigrate creates it on next service start
- Without this index, `SELECT DISTINCT patient_id WHERE updated_at > ?` would full-scan
- At current data volumes (10K patients) this matters little; at 1M patients it matters a lot

### Decision 6: Top-level cancellable context introduced in main.go
- Replaces the current `<-quit` followed by hard exit
- All long-running goroutines (Kafka consumer, scheduler, HTTP server) receive the same context
- This is a small but real improvement to KB-26's shutdown semantics

### Decision 7: Same-day phenotype change detection works via in-memory comparison
- `BPContextRepository.FetchLatest` returns the prior snapshot (yesterday's) before today's `SaveSnapshot` upserts it
- The orchestrator captures the prior phenotype in a local variable BEFORE saving the new snapshot, compares the two, and emits events if they differ
- After save, the snapshot reflects today's state — the in-memory `oldPhenotype` is the only source of truth for the comparison

---

## File Structure

### KB-26 (changes)
| Action | File | Responsibility |
|--------|------|---------------|
| Create | `internal/clients/kb19_client.go` | HTTP POST events to `{KB19_URL}/api/v1/events` |
| Create | `internal/clients/kb19_client_test.go` | httptest coverage of the publisher |
| Modify | `internal/services/bp_context_repository.go` | Add `ListActivePatientIDs(window time.Duration) ([]string, error)` |
| Modify | `internal/services/bp_context_repository_test.go` | Test the new query against in-memory sqlite |
| Modify | `internal/services/bp_context_orchestrator.go` | Add `EmitPhenotypeEvents` flag, prior-snapshot comparison, event emission |
| Modify | `internal/services/bp_context_orchestrator_test.go` | Tests for event emission paths (new detection, phenotype change, same phenotype no-op) |
| Create | `internal/services/batch_scheduler.go` | Generic `BatchScheduler` + `BatchJob` interface |
| Create | `internal/services/batch_scheduler_test.go` | Scheduler unit tests with stub job and stub clock |
| Create | `internal/services/bp_context_batch_job.go` | `BPContextDailyBatch` implementing `BatchJob` |
| Create | `internal/services/bp_context_batch_job_test.go` | Job tests: process all active patients, error isolation, concurrency |
| Modify | `internal/models/twin_state.go` | Add standalone index on `UpdatedAt` for cross-patient queries |
| Modify | `internal/metrics/collector.go` | Add `BPBatchDuration`, `BPBatchPatientsProcessed`, `BPBatchErrors`, `KB19PublishLatency` |
| Modify | `internal/config/config.go` | Add `KB19URL`, `KB19TimeoutMS`, `BPBatchEnabled`, `BPBatchHourUTC`, `BPBatchConcurrency`, `BPActiveWindowDays` |
| Modify | `main.go` | Top-level context, instantiate KB-19 client, instantiate scheduler, register batch job, start scheduler goroutine |

### KB-23 (changes)
| Action | File | Responsibility |
|--------|------|---------------|
| Modify | `internal/services/kb19_publisher.go` | Add deprecation comments to `PublishMaskedHTNDetected` and `PublishPhenotypeChanged` pointing to KB-26 |

**Total: 6 create, 8 modify = 14 files**

---

## Task 1: KB-19 Client (KB-26)

KB-26 currently has clients for KB-20, KB-21, KB-22 — but no KB-19 client. Phase 3 adds the first one because the orchestrator needs to publish phenotype events.

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/clients/kb19_client.go`
- Create: `kb-26-metabolic-digital-twin/internal/clients/kb19_client_test.go`

- [ ] **Step 1: Write the failing test**

```go
// kb-26-metabolic-digital-twin/internal/clients/kb19_client_test.go
package clients

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestKB19Client_PublishPhenotypeChanged_PostsCorrectEnvelope(t *testing.T) {
	var receivedPath string
	var receivedBody KB19Event

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := NewKB19Client(server.URL, 1*time.Second, zap.NewNop())
	err := client.PublishPhenotypeChanged(context.Background(), "p1", "WHITE_COAT_HTN", "SUSTAINED_HTN")
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if receivedPath != "/api/v1/events" {
		t.Errorf("expected path /api/v1/events, got %s", receivedPath)
	}
	if receivedBody.EventType != "BP_PHENOTYPE_CHANGED" {
		t.Errorf("expected BP_PHENOTYPE_CHANGED, got %s", receivedBody.EventType)
	}
	if receivedBody.PatientID != "p1" {
		t.Errorf("expected patient p1, got %s", receivedBody.PatientID)
	}
	if receivedBody.OldPhenotype != "WHITE_COAT_HTN" || receivedBody.NewPhenotype != "SUSTAINED_HTN" {
		t.Errorf("phenotype fields wrong: %+v", receivedBody)
	}
}

func TestKB19Client_PublishMaskedHTNDetected_IncludesUrgency(t *testing.T) {
	var receivedBody KB19Event

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := NewKB19Client(server.URL, 1*time.Second, zap.NewNop())
	err := client.PublishMaskedHTNDetected(context.Background(), "p1", "MASKED_HTN", "IMMEDIATE")
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if receivedBody.EventType != "MASKED_HTN_DETECTED" {
		t.Errorf("expected MASKED_HTN_DETECTED, got %s", receivedBody.EventType)
	}
	if receivedBody.BPPhenotype != "MASKED_HTN" {
		t.Errorf("expected BPPhenotype MASKED_HTN, got %s", receivedBody.BPPhenotype)
	}
	if receivedBody.Urgency != "IMMEDIATE" {
		t.Errorf("expected Urgency IMMEDIATE, got %s", receivedBody.Urgency)
	}
}

func TestKB19Client_PublishPhenotypeChanged_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewKB19Client(server.URL, 1*time.Second, zap.NewNop())
	err := client.PublishPhenotypeChanged(context.Background(), "p1", "WHITE_COAT_HTN", "SUSTAINED_HTN")
	if err == nil {
		t.Error("expected error on 500")
	}
}

func TestKB19Client_NetworkError_DoesNotPanic(t *testing.T) {
	// Pointing at an unreachable URL should return an error, not panic.
	client := NewKB19Client("http://127.0.0.1:1", 100*time.Millisecond, zap.NewNop())
	err := client.PublishPhenotypeChanged(context.Background(), "p1", "WHITE_COAT_HTN", "SUSTAINED_HTN")
	if err == nil {
		t.Error("expected error from unreachable server")
	}
}
```

- [ ] **Step 2: Verify test fails (compilation error)**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/clients/ -run "TestKB19Client" -v 2>&1 | head -10`
Expected: compilation error — `KB19Client`, `KB19Event`, `NewKB19Client` undefined.

- [ ] **Step 3: Create the client**

```go
// kb-26-metabolic-digital-twin/internal/clients/kb19_client.go
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// KB19Event is the envelope KB-19 expects on POST /api/v1/events.
// Fields use omitempty so each event type only sends the fields it cares
// about. Mirrors KB-23's KB19Event struct so KB-19 sees one consistent shape.
type KB19Event struct {
	EventType    string    `json:"event_type"`
	PatientID    string    `json:"patient_id"`
	Timestamp    time.Time `json:"timestamp"`
	BPPhenotype  string    `json:"bp_phenotype,omitempty"`
	Urgency      string    `json:"urgency,omitempty"`
	OldPhenotype string    `json:"old_phenotype,omitempty"`
	NewPhenotype string    `json:"new_phenotype,omitempty"`
}

// KB19Client publishes events to KB-19 via HTTP POST.
type KB19Client struct {
	baseURL string
	client  *http.Client
	log     *zap.Logger
}

// NewKB19Client constructs a client.
func NewKB19Client(baseURL string, timeout time.Duration, log *zap.Logger) *KB19Client {
	return &KB19Client{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
		log:     log,
	}
}

// PublishMaskedHTNDetected announces a newly classified masked HTN patient.
func (c *KB19Client) PublishMaskedHTNDetected(ctx context.Context, patientID, phenotype, urgency string) error {
	return c.post(ctx, KB19Event{
		EventType:   "MASKED_HTN_DETECTED",
		PatientID:   patientID,
		Timestamp:   time.Now().UTC(),
		BPPhenotype: phenotype,
		Urgency:     urgency,
	})
}

// PublishPhenotypeChanged announces a transition from one phenotype to another.
func (c *KB19Client) PublishPhenotypeChanged(ctx context.Context, patientID, oldPhenotype, newPhenotype string) error {
	return c.post(ctx, KB19Event{
		EventType:    "BP_PHENOTYPE_CHANGED",
		PatientID:    patientID,
		Timestamp:    time.Now().UTC(),
		OldPhenotype: oldPhenotype,
		NewPhenotype: newPhenotype,
	})
}

func (c *KB19Client) post(ctx context.Context, event KB19Event) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal KB-19 event: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/events", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build KB-19 request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Warn("KB-19 publish failed", zap.String("url", url), zap.Error(err))
		return fmt.Errorf("KB-19 POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("KB-19 returned status %d: %s", resp.StatusCode, string(respBody))
	}

	c.log.Debug("KB-19 event published",
		zap.String("event_type", event.EventType),
		zap.String("patient_id", event.PatientID),
	)
	return nil
}
```

- [ ] **Step 4: Run client tests**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/clients/ -run "TestKB19Client" -v`
Expected: 4/4 PASS.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/clients/kb19_client.go
git add kb-26-metabolic-digital-twin/internal/clients/kb19_client_test.go
git commit -m "feat(kb26): KB-19 event publisher client

First KB-19 publisher in KB-26. POSTs MASKED_HTN_DETECTED and
BP_PHENOTYPE_CHANGED events to {KB19_URL}/api/v1/events. Mirrors KB-23's
KB19Event envelope schema (typed fields with omitempty). HTTP-based, no
Kafka — matches the established cross-service event pattern."
```

---

## Task 2: TwinState Index for Cross-Patient Queries

The composite index `(patient_id, updated_at DESC)` doesn't help a `WHERE updated_at > ?` query without a `patient_id` predicate. Add a standalone index.

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/models/twin_state.go`

- [ ] **Step 1: Add the GORM tag**

In `twin_state.go`, locate the `UpdatedAt` field. Currently:

```go
UpdatedAt time.Time `gorm:"not null;default:now();index:idx_twin_patient,priority:2,sort:desc" json:"updated_at"`
```

Change to add a second standalone index:

```go
UpdatedAt time.Time `gorm:"not null;default:now();index:idx_twin_patient,priority:2,sort:desc;index:idx_twin_state_updated_at,sort:desc" json:"updated_at"`
```

This adds `idx_twin_state_updated_at` as a separate single-column descending index, supporting the `WHERE updated_at > ?` cross-patient query that the batch scheduler will introduce.

- [ ] **Step 2: Build to verify**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go build ./...`
Expected: clean build.

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/models/twin_state.go
git commit -m "perf(kb26): add standalone index on twin_state.updated_at

The existing composite index (patient_id, updated_at DESC) cannot serve
WHERE updated_at > ? queries that omit a patient_id predicate, because
the leading column is unconstrained. Adds a standalone descending index
on updated_at to support the Phase 3 daily batch scheduler's
ListActivePatients query, which selects all patients with twin_state
activity in the last 30 days."
```

---

## Task 3: ListActivePatientIDs Query

Add a method to `BPContextRepository` that returns active patient IDs for the batch.

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_repository.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_repository_test.go`

- [ ] **Step 1: Write the failing test**

Append to `bp_context_repository_test.go`:

```go
func TestBPContextRepository_ListActivePatientIDs(t *testing.T) {
	db := setupBPContextTestDB(t)

	// Set up TwinState test rows. The setup helper only migrates BPContextHistory,
	// so we explicitly migrate TwinState here too.
	if err := db.AutoMigrate(&models.TwinState{}); err != nil {
		t.Fatalf("migrate twin_state: %v", err)
	}

	now := time.Now().UTC()
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()

	// Active: updated 5 days ago
	if err := db.Create(&models.TwinState{
		ID:           uuid.New(),
		PatientID:    id1,
		StateVersion: 1,
		UpdateSource: "TEST",
		UpdatedAt:    now.AddDate(0, 0, -5),
	}).Error; err != nil {
		t.Fatalf("seed active patient: %v", err)
	}

	// Active: updated yesterday
	if err := db.Create(&models.TwinState{
		ID:           uuid.New(),
		PatientID:    id2,
		StateVersion: 1,
		UpdateSource: "TEST",
		UpdatedAt:    now.AddDate(0, 0, -1),
	}).Error; err != nil {
		t.Fatalf("seed active patient: %v", err)
	}

	// Inactive: updated 60 days ago
	if err := db.Create(&models.TwinState{
		ID:           uuid.New(),
		PatientID:    id3,
		StateVersion: 1,
		UpdateSource: "TEST",
		UpdatedAt:    now.AddDate(0, 0, -60),
	}).Error; err != nil {
		t.Fatalf("seed inactive patient: %v", err)
	}

	repo := NewBPContextRepository(db)
	active, err := repo.ListActivePatientIDs(30 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("ListActivePatientIDs: %v", err)
	}

	if len(active) != 2 {
		t.Errorf("expected 2 active patients, got %d", len(active))
	}

	got := map[string]bool{}
	for _, p := range active {
		got[p] = true
	}
	if !got[id1.String()] || !got[id2.String()] {
		t.Errorf("expected active set to contain %s and %s, got %v", id1, id2, got)
	}
	if got[id3.String()] {
		t.Errorf("inactive patient %s should not be in result", id3)
	}
}

func TestBPContextRepository_ListActivePatientIDs_DeduplicatesMultipleSnapshots(t *testing.T) {
	db := setupBPContextTestDB(t)
	if err := db.AutoMigrate(&models.TwinState{}); err != nil {
		t.Fatalf("migrate twin_state: %v", err)
	}

	now := time.Now().UTC()
	patientID := uuid.New()

	// Three snapshots for the same patient — query must return one row.
	for i := 0; i < 3; i++ {
		if err := db.Create(&models.TwinState{
			ID:           uuid.New(),
			PatientID:    patientID,
			StateVersion: i + 1,
			UpdateSource: "TEST",
			UpdatedAt:    now.AddDate(0, 0, -i),
		}).Error; err != nil {
			t.Fatalf("seed snapshot %d: %v", i, err)
		}
	}

	repo := NewBPContextRepository(db)
	active, err := repo.ListActivePatientIDs(30 * 24 * time.Hour)
	if err != nil {
		t.Fatalf("ListActivePatientIDs: %v", err)
	}
	if len(active) != 1 {
		t.Errorf("expected 1 deduplicated patient, got %d", len(active))
	}
}
```

Add `"github.com/google/uuid"` to the test imports if not already present.

- [ ] **Step 2: Verify tests fail**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestBPContextRepository_ListActivePatientIDs" -v 2>&1 | head -10`
Expected: compilation error — `ListActivePatientIDs` undefined.

- [ ] **Step 3: Implement the method**

In `bp_context_repository.go`, append:

```go
// ListActivePatientIDs returns distinct patient IDs from twin_state whose
// most recent update is within the given activity window. The query
// dedups via SELECT DISTINCT — three snapshots for the same patient
// return one ID. IDs are returned as strings to match the BP context
// orchestrator's signature.
func (r *BPContextRepository) ListActivePatientIDs(window time.Duration) ([]string, error) {
	cutoff := time.Now().UTC().Add(-window)
	var ids []string
	err := r.db.Model(&models.TwinState{}).
		Distinct("patient_id").
		Where("updated_at > ?", cutoff).
		Pluck("patient_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}
```

Add `"time"` to the imports if not already present (it should be — `models.BPContextHistory` already uses it).

- [ ] **Step 4: Run repository tests**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestBPContextRepository" -v`
Expected: all 6 repository tests PASS (4 from Phase 2 + 2 new).

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/services/bp_context_repository.go
git add kb-26-metabolic-digital-twin/internal/services/bp_context_repository_test.go
git commit -m "feat(kb26): ListActivePatientIDs query for batch scheduling

Returns distinct patient IDs from twin_state whose most recent update
is within the given activity window. Dedup via SELECT DISTINCT — multiple
snapshots per patient collapse to one row. First cross-patient query in
KB-26; backed by the new idx_twin_state_updated_at index."
```

---

## Task 4: Orchestrator — Phenotype Change Detection + Event Emission

Extend the orchestrator to capture the prior phenotype before saving and emit events to KB-19 when the phenotype is new (no prior snapshot) or has changed.

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator_test.go`

- [ ] **Step 1: Read the current orchestrator**

Before editing, read the file end-to-end. Note:
- Current `BPContextOrchestrator` struct fields
- Current `NewBPContextOrchestrator` signature (added metrics in Phase 2 Task 10)
- The structure of `Classify` — where the `SaveSnapshot` call is and where the metrics are recorded

- [ ] **Step 2: Define the publisher interface and add fields**

Add the interface near the top of the file (after the existing `KB20Fetcher` and `KB21Fetcher` interfaces):

```go
// KB19EventPublisher is the narrow interface the orchestrator needs from
// the KB-19 client. Defined here (not in the clients package) so tests
// can stub it without importing the real client.
type KB19EventPublisher interface {
	PublishMaskedHTNDetected(ctx context.Context, patientID, phenotype, urgency string) error
	PublishPhenotypeChanged(ctx context.Context, patientID, oldPhenotype, newPhenotype string) error
}
```

Add a `kb19` field to `BPContextOrchestrator`:

```go
kb19 KB19EventPublisher
```

Update `NewBPContextOrchestrator` to accept the publisher as a new parameter (placed after `metrics`):

```go
func NewBPContextOrchestrator(
	kb20 KB20Fetcher,
	kb21 KB21Fetcher,
	repo *BPContextRepository,
	thresholds *config.BPContextThresholds,
	log *zap.Logger,
	metricsCollector *metrics.Collector,
	kb19 KB19EventPublisher,
) *BPContextOrchestrator {
	return &BPContextOrchestrator{
		kb20:       kb20,
		kb21:       kb21,
		repo:       repo,
		thresholds: thresholds,
		log:        log,
		metrics:    metricsCollector,
		kb19:       kb19,
	}
}
```

- [ ] **Step 3: Add prior-snapshot fetch + post-save event emission**

Inside `Classify`, locate the section where `SaveSnapshot` is called. Modify the flow so:

1. **Before** calling `SaveSnapshot`, fetch the prior snapshot and capture its phenotype:

```go
// Capture the prior phenotype BEFORE saving — SaveSnapshot upserts on
// (patient_id, snapshot_date), so a same-day reclassification overwrites
// yesterday's row and we'd lose the comparison if we fetched after.
var oldPhenotype models.BPContextPhenotype
prior, fetchErr := o.repo.FetchLatest(patientID)
if fetchErr != nil {
    o.log.Warn("prior snapshot fetch failed; treating as first detection",
        zap.String("patient_id", patientID), zap.Error(fetchErr))
} else if prior != nil {
    oldPhenotype = prior.Phenotype
}
```

2. **After** the existing `SaveSnapshot` call (and after the metrics increment for phenotype counter), call a new helper:

```go
o.emitPhenotypeEvents(ctx, patientID, oldPhenotype, result.Phenotype)
```

3. Add the helper method:

```go
// emitPhenotypeEvents publishes events to KB-19 when the classification
// represents a new detection or a phenotype transition.
//
//   oldPhenotype empty + new is masked variant -> MASKED_HTN_DETECTED
//   oldPhenotype != newPhenotype -> BP_PHENOTYPE_CHANGED
//   oldPhenotype == newPhenotype -> no event
//
// Failures are logged but do not affect the caller — events are
// best-effort, the snapshot is the source of truth.
func (o *BPContextOrchestrator) emitPhenotypeEvents(
	ctx context.Context,
	patientID string,
	oldPhenotype models.BPContextPhenotype,
	newPhenotype models.BPContextPhenotype,
) {
	if o.kb19 == nil {
		return
	}

	isNewDetection := oldPhenotype == "" && (newPhenotype == models.PhenotypeMaskedHTN || newPhenotype == models.PhenotypeMaskedUncontrolled)
	isTransition := oldPhenotype != "" && oldPhenotype != newPhenotype

	if isNewDetection {
		urgency := "URGENT"
		if newPhenotype == models.PhenotypeMaskedUncontrolled {
			urgency = "URGENT"
		}
		// Check the saved classification for amplification flags via the
		// latest snapshot — but for new detection we only have the basic
		// phenotype. Amplification-aware urgency stays in the card layer.
		if err := o.kb19.PublishMaskedHTNDetected(ctx, patientID, string(newPhenotype), urgency); err != nil {
			o.log.Warn("KB-19 MASKED_HTN_DETECTED publish failed",
				zap.String("patient_id", patientID), zap.Error(err))
		}
		return
	}

	if isTransition {
		if err := o.kb19.PublishPhenotypeChanged(ctx, patientID, string(oldPhenotype), string(newPhenotype)); err != nil {
			o.log.Warn("KB-19 BP_PHENOTYPE_CHANGED publish failed",
				zap.String("patient_id", patientID), zap.Error(err))
		}
	}
}
```

- [ ] **Step 4: Add tests for event emission**

Append to `bp_context_orchestrator_test.go`:

```go
// stubKB19Publisher implements KB19EventPublisher for tests.
type stubKB19Publisher struct {
	maskedHTNCalls       []string // patientIDs for MASKED_HTN_DETECTED
	phenotypeChangeCalls [][3]string // {patientID, old, new}
}

func (s *stubKB19Publisher) PublishMaskedHTNDetected(ctx context.Context, patientID, phenotype, urgency string) error {
	s.maskedHTNCalls = append(s.maskedHTNCalls, patientID)
	return nil
}

func (s *stubKB19Publisher) PublishPhenotypeChanged(ctx context.Context, patientID, oldPhenotype, newPhenotype string) error {
	s.phenotypeChangeCalls = append(s.phenotypeChangeCalls, [3]string{patientID, oldPhenotype, newPhenotype})
	return nil
}

// Update newOrchestrator to take an optional publisher.
func newOrchestratorWithPublisher(t *testing.T, kb20 KB20Fetcher, kb21 KB21Fetcher, kb19 KB19EventPublisher) *BPContextOrchestrator {
	t.Helper()
	db := setupBPContextTestDB(t)
	repo := NewBPContextRepository(db)
	thresholds := defaultBPContextThresholds()
	return NewBPContextOrchestrator(kb20, kb21, repo, thresholds, zap.NewNop(), nil, kb19)
}

func TestBPContextOrchestrator_NewMaskedHTN_PublishesDetectedEvent(t *testing.T) {
	kb20 := &stubKB20Client{
		profile: &clients.KB20PatientProfile{
			PatientID:        "p1",
			SBP14dMean:       ptrFloat(148),
			DBP14dMean:       ptrFloat(92),
			ClinicSBPMean:    ptrFloat(128),
			ClinicDBPMean:    ptrFloat(78),
			ClinicReadings:   2,
			HomeReadings:     14,
			HomeDaysWithData: 7,
		},
	}
	kb21 := &stubKB21Client{}
	kb19 := &stubKB19Publisher{}
	orch := newOrchestratorWithPublisher(t, kb20, kb21, kb19)

	if _, err := orch.Classify(context.Background(), "p1"); err != nil {
		t.Fatalf("classify: %v", err)
	}

	if len(kb19.maskedHTNCalls) != 1 {
		t.Errorf("expected 1 MASKED_HTN_DETECTED call, got %d", len(kb19.maskedHTNCalls))
	}
	if len(kb19.phenotypeChangeCalls) != 0 {
		t.Errorf("first detection should not emit BP_PHENOTYPE_CHANGED, got %d", len(kb19.phenotypeChangeCalls))
	}
}

func TestBPContextOrchestrator_PhenotypeUnchanged_NoEvent(t *testing.T) {
	kb20 := &stubKB20Client{
		profile: &clients.KB20PatientProfile{
			PatientID:        "p1",
			SBP14dMean:       ptrFloat(148),
			DBP14dMean:       ptrFloat(92),
			ClinicSBPMean:    ptrFloat(128),
			ClinicDBPMean:    ptrFloat(78),
			ClinicReadings:   2,
			HomeReadings:     14,
			HomeDaysWithData: 7,
		},
	}
	kb21 := &stubKB21Client{}
	kb19 := &stubKB19Publisher{}
	orch := newOrchestratorWithPublisher(t, kb20, kb21, kb19)

	// First classification — emits MASKED_HTN_DETECTED
	if _, err := orch.Classify(context.Background(), "p1"); err != nil {
		t.Fatalf("first classify: %v", err)
	}
	// Second classification of identical state — should not emit anything
	// (same-day upsert overwrites the snapshot but in-memory comparison
	// of NEW classification phenotype against the prior phenotype now
	// stored finds no change).
	kb19.maskedHTNCalls = nil
	kb19.phenotypeChangeCalls = nil
	if _, err := orch.Classify(context.Background(), "p1"); err != nil {
		t.Fatalf("second classify: %v", err)
	}
	if len(kb19.maskedHTNCalls) != 0 {
		t.Errorf("expected 0 MASKED_HTN_DETECTED on re-classify, got %d", len(kb19.maskedHTNCalls))
	}
	if len(kb19.phenotypeChangeCalls) != 0 {
		t.Errorf("expected 0 BP_PHENOTYPE_CHANGED on re-classify, got %d", len(kb19.phenotypeChangeCalls))
	}
}

func TestBPContextOrchestrator_PhenotypeChanged_PublishesTransition(t *testing.T) {
	// First call: home reading high (148) -> MASKED_HTN
	kb20 := &stubKB20Client{
		profile: &clients.KB20PatientProfile{
			PatientID:        "p1",
			SBP14dMean:       ptrFloat(148),
			DBP14dMean:       ptrFloat(92),
			ClinicSBPMean:    ptrFloat(128),
			ClinicDBPMean:    ptrFloat(78),
			ClinicReadings:   2,
			HomeReadings:     14,
			HomeDaysWithData: 7,
		},
	}
	kb21 := &stubKB21Client{}
	kb19 := &stubKB19Publisher{}
	orch := newOrchestratorWithPublisher(t, kb20, kb21, kb19)

	if _, err := orch.Classify(context.Background(), "p1"); err != nil {
		t.Fatalf("first classify: %v", err)
	}

	// Second call: home reading dropped to normal (120) -> SUSTAINED_NORMOTENSION
	// (clinic is also normal). Same patient, different day required for
	// the upsert NOT to overwrite, BUT in this test the same-day upsert
	// IS what we want — the in-memory `oldPhenotype` captured from the
	// pre-save fetch is the prior MH classification.
	kb20.profile.SBP14dMean = ptrFloat(120)
	kb20.profile.DBP14dMean = ptrFloat(75)
	kb19.maskedHTNCalls = nil
	kb19.phenotypeChangeCalls = nil

	if _, err := orch.Classify(context.Background(), "p1"); err != nil {
		t.Fatalf("second classify: %v", err)
	}

	if len(kb19.phenotypeChangeCalls) != 1 {
		t.Fatalf("expected 1 BP_PHENOTYPE_CHANGED, got %d", len(kb19.phenotypeChangeCalls))
	}
	got := kb19.phenotypeChangeCalls[0]
	if got[1] != "MASKED_HTN" || got[2] != "SUSTAINED_NORMOTENSION" {
		t.Errorf("expected MH->SN transition, got %v", got)
	}
}

func TestBPContextOrchestrator_NilPublisher_NoEventsNoErrors(t *testing.T) {
	kb20 := &stubKB20Client{
		profile: &clients.KB20PatientProfile{
			PatientID:        "p1",
			SBP14dMean:       ptrFloat(148),
			DBP14dMean:       ptrFloat(92),
			ClinicSBPMean:    ptrFloat(128),
			ClinicDBPMean:    ptrFloat(78),
			ClinicReadings:   2,
			HomeReadings:     14,
			HomeDaysWithData: 7,
		},
	}
	kb21 := &stubKB21Client{}
	orch := newOrchestratorWithPublisher(t, kb20, kb21, nil) // explicit nil publisher

	if _, err := orch.Classify(context.Background(), "p1"); err != nil {
		t.Fatalf("classify with nil publisher should not error: %v", err)
	}
}
```

The existing `newOrchestrator` helper from Phase 2 must also be updated to pass `nil` for the publisher in its callers. Find every existing test that calls `newOrchestrator(...)` and either:
- Update them to use `newOrchestratorWithPublisher(t, kb20, kb21, nil)`, OR
- Modify the `newOrchestrator` helper itself to call `newOrchestratorWithPublisher(t, kb20, kb21, nil)` internally

The second option is less invasive. Apply it.

- [ ] **Step 5: Run tests**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestBPContextOrchestrator" -v`
Expected: all 8 orchestrator tests PASS (4 from Phase 2 + 4 new).

- [ ] **Step 6: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator.go
git add kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator_test.go
git commit -m "feat(kb26): orchestrator publishes phenotype change events to KB-19

Captures the prior phenotype via repo.FetchLatest BEFORE SaveSnapshot
(which upserts on same-day key, overwriting the comparison source).
Emits MASKED_HTN_DETECTED on first detection of a masked variant and
BP_PHENOTYPE_CHANGED on any transition. Nil publisher is safe — events
become a no-op rather than a panic. Failures log but do not affect
classification result (best-effort delivery)."
```

---

## Task 5: BatchScheduler + BatchJob Interface

A generic scheduler that runs registered jobs daily at a configured UTC hour. Phase 3 will register exactly one job (BP context); the scheduler is intentionally generic so that `quarterly_aggregator` (currently dead-wired in main.go) can later be wrapped in a `BatchJob` without rewriting the scheduler.

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/batch_scheduler.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/batch_scheduler_test.go`

- [ ] **Step 1: Write the failing test**

```go
// batch_scheduler_test.go
package services

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
)

// stubBatchJob is a BatchJob that records when it ran and lets tests
// drive how long it takes / whether it errors.
type stubBatchJob struct {
	name      string
	runs      atomic.Int32
	delay     time.Duration
	returnErr error
	mu        sync.Mutex
	runTimes  []time.Time
}

func (s *stubBatchJob) Name() string { return s.name }

func (s *stubBatchJob) Run(ctx context.Context) error {
	s.mu.Lock()
	s.runTimes = append(s.runTimes, time.Now())
	s.mu.Unlock()
	s.runs.Add(1)
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return s.returnErr
}

func TestBatchScheduler_RunOnceImmediately(t *testing.T) {
	job := &stubBatchJob{name: "test"}
	sched := NewBatchScheduler(zap.NewNop())
	sched.Register(job)

	// RunOnce is the synchronous-execute-now method, used by tests and
	// by manual triggers (e.g. an admin endpoint).
	if err := sched.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	if job.runs.Load() != 1 {
		t.Errorf("expected 1 run, got %d", job.runs.Load())
	}
}

func TestBatchScheduler_RunOnce_MultipleJobs(t *testing.T) {
	jobA := &stubBatchJob{name: "a"}
	jobB := &stubBatchJob{name: "b"}
	sched := NewBatchScheduler(zap.NewNop())
	sched.Register(jobA)
	sched.Register(jobB)

	if err := sched.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}

	if jobA.runs.Load() != 1 || jobB.runs.Load() != 1 {
		t.Errorf("expected both jobs to run once, got a=%d b=%d", jobA.runs.Load(), jobB.runs.Load())
	}
}

func TestBatchScheduler_OneJobErrors_OthersStillRun(t *testing.T) {
	jobA := &stubBatchJob{name: "a", returnErr: errSimulated()}
	jobB := &stubBatchJob{name: "b"}
	sched := NewBatchScheduler(zap.NewNop())
	sched.Register(jobA)
	sched.Register(jobB)

	// RunOnce should not bail on the first error — each job is isolated.
	_ = sched.RunOnce(context.Background())

	if jobA.runs.Load() != 1 {
		t.Errorf("job A should still have run, got %d", jobA.runs.Load())
	}
	if jobB.runs.Load() != 1 {
		t.Errorf("job B should run despite job A error, got %d", jobB.runs.Load())
	}
}

func TestBatchScheduler_StartLoop_RespectsContextCancel(t *testing.T) {
	job := &stubBatchJob{name: "test"}
	sched := NewBatchScheduler(zap.NewNop())
	sched.Register(job)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		// Use a 50ms loop interval for the test (the production hour
		// schedule is computed differently; see StartLoop docs).
		sched.StartLoop(ctx, 50*time.Millisecond)
		close(done)
	}()

	// Wait for at least one run, then cancel.
	time.Sleep(150 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Good: scheduler exited on context cancel.
	case <-time.After(500 * time.Millisecond):
		t.Fatal("scheduler did not exit within 500ms of context cancel")
	}

	if job.runs.Load() < 1 {
		t.Errorf("expected at least 1 run, got %d", job.runs.Load())
	}
}
```

The `errSimulated()` and `simulatedErr` helpers from `bp_context_orchestrator_test.go` are reusable here — they already exist in the same package.

- [ ] **Step 2: Verify tests fail**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestBatchScheduler" -v 2>&1 | head -10`
Expected: compilation error — `BatchScheduler`, `NewBatchScheduler`, `Register`, `RunOnce`, `StartLoop` undefined.

- [ ] **Step 3: Implement the scheduler**

```go
// batch_scheduler.go
package services

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// BatchJob is the contract for any job runnable by BatchScheduler.
// Implementations must be safe to call from a long-lived goroutine and
// must respect context cancellation for graceful shutdown.
type BatchJob interface {
	// Name returns a human-readable identifier for the job, used in logs
	// and metrics labels.
	Name() string

	// Run executes one full pass of the job. Implementations must process
	// all relevant entities and return nil only on full success. Errors
	// are logged by the scheduler but do not block other registered jobs.
	Run(ctx context.Context) error
}

// BatchScheduler runs registered BatchJobs on a daily cadence.
// Phase 3 registers exactly one job (BPContextDailyBatch); the interface
// is intentionally generic so future jobs (e.g. quarterly_aggregator)
// can be added without rewriting the scheduler.
type BatchScheduler struct {
	jobs []BatchJob
	mu   sync.RWMutex
	log  *zap.Logger
}

// NewBatchScheduler constructs an empty scheduler.
func NewBatchScheduler(log *zap.Logger) *BatchScheduler {
	return &BatchScheduler{log: log}
}

// Register adds a job to the scheduler. Call before StartLoop or RunOnce.
// Registration is goroutine-safe but not expected to happen after start.
func (s *BatchScheduler) Register(job BatchJob) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append(s.jobs, job)
}

// RunOnce executes every registered job sequentially. One job's error
// does NOT prevent subsequent jobs from running — each is isolated.
// Returns the FIRST error encountered, or nil if all succeeded.
func (s *BatchScheduler) RunOnce(ctx context.Context) error {
	s.mu.RLock()
	jobs := make([]BatchJob, len(s.jobs))
	copy(jobs, s.jobs)
	s.mu.RUnlock()

	var firstErr error
	for _, job := range jobs {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		s.log.Info("batch job starting", zap.String("job", job.Name()))
		start := time.Now()
		err := job.Run(ctx)
		duration := time.Since(start)
		if err != nil {
			s.log.Error("batch job failed",
				zap.String("job", job.Name()),
				zap.Duration("duration", duration),
				zap.Error(err))
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		s.log.Info("batch job complete",
			zap.String("job", job.Name()),
			zap.Duration("duration", duration))
	}
	return firstErr
}

// StartLoop runs RunOnce on a fixed interval until ctx is cancelled.
// The interval is the wake cadence — production wires this to a daily
// hour-aligned interval, but the function takes a generic Duration so
// tests can drive it faster. The first run happens after one interval
// has elapsed (not immediately at start).
func (s *BatchScheduler) StartLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.log.Info("batch scheduler started", zap.Duration("interval", interval))
	for {
		select {
		case <-ctx.Done():
			s.log.Info("batch scheduler stopped")
			return
		case <-ticker.C:
			if err := s.RunOnce(ctx); err != nil {
				s.log.Warn("scheduled batch run had errors",
					zap.Error(err))
			}
		}
	}
}
```

- [ ] **Step 4: Run scheduler tests**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestBatchScheduler" -v`
Expected: 4/4 PASS.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/services/batch_scheduler.go
git add kb-26-metabolic-digital-twin/internal/services/batch_scheduler_test.go
git commit -m "feat(kb26): generic batch scheduler with BatchJob interface

First scheduler in KB-26. Holds a list of BatchJobs and runs them all
sequentially when StartLoop ticks or RunOnce is called manually. Per-job
error isolation: one job failing does not stop subsequent jobs. Context
cancellation propagates immediately for graceful shutdown.

Phase 3 will register BPContextDailyBatch as the first job. The
quarterly_aggregator currently sitting unwired in main.go can become a
second job via the same interface in a future plan."
```

---

## Task 6: BPContextDailyBatch Job

The first concrete `BatchJob`: iterate active patients, classify each, with bounded concurrency.

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/bp_context_batch_job.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/bp_context_batch_job_test.go`

- [ ] **Step 1: Write the failing test**

```go
// bp_context_batch_job_test.go
package services

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/clients"
	"kb-26-metabolic-digital-twin/internal/models"
)

// trackingOrchestrator wraps a real orchestrator but counts how many
// patients are processed.
type trackingOrchestrator struct {
	inner    *BPContextOrchestrator
	count    atomic.Int32
	errOn    map[string]bool
	errValue error
}

func (t *trackingOrchestrator) Classify(ctx context.Context, patientID string) (*models.BPContextClassification, error) {
	t.count.Add(1)
	if t.errOn[patientID] {
		return nil, t.errValue
	}
	return t.inner.Classify(ctx, patientID)
}

func setupBatchJobTest(t *testing.T, kb20Profile *clients.KB20PatientProfile) (*BPContextDailyBatch, *BPContextRepository, *trackingOrchestrator) {
	t.Helper()

	db := setupBPContextTestDB(t)
	if err := db.AutoMigrate(&models.TwinState{}); err != nil {
		t.Fatalf("migrate twin_state: %v", err)
	}

	repo := NewBPContextRepository(db)
	kb20 := &stubKB20Client{profile: kb20Profile}
	kb21 := &stubKB21Client{}
	thresholds := defaultBPContextThresholds()
	inner := NewBPContextOrchestrator(kb20, kb21, repo, thresholds, zap.NewNop(), nil, nil)

	tracker := &trackingOrchestrator{inner: inner}
	job := NewBPContextDailyBatch(repo, tracker, 30*24*time.Hour, 4, zap.NewNop())
	return job, repo, tracker
}

func TestBPContextDailyBatch_ProcessesAllActivePatients(t *testing.T) {
	job, repo, tracker := setupBatchJobTest(t, &clients.KB20PatientProfile{
		PatientID:        "shared",
		SBP14dMean:       ptrFloat(120),
		DBP14dMean:       ptrFloat(75),
		ClinicSBPMean:    ptrFloat(118),
		ClinicDBPMean:    ptrFloat(74),
		ClinicReadings:   2,
		HomeReadings:     14,
		HomeDaysWithData: 7,
	})

	// Seed 5 active patients
	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		if err := repo.db.Create(&models.TwinState{
			ID:           uuid.New(),
			PatientID:    uuid.New(),
			StateVersion: 1,
			UpdateSource: "TEST",
			UpdatedAt:    now.AddDate(0, 0, -i),
		}).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := tracker.count.Load(); got != 5 {
		t.Errorf("expected 5 patients processed, got %d", got)
	}
}

func TestBPContextDailyBatch_OneClassificationErrors_OthersStillRun(t *testing.T) {
	job, repo, tracker := setupBatchJobTest(t, &clients.KB20PatientProfile{
		PatientID:        "shared",
		SBP14dMean:       ptrFloat(120),
		DBP14dMean:       ptrFloat(75),
		ClinicSBPMean:    ptrFloat(118),
		ClinicDBPMean:    ptrFloat(74),
		ClinicReadings:   2,
		HomeReadings:     14,
		HomeDaysWithData: 7,
	})

	// Seed 3 active patients
	now := time.Now().UTC()
	patientIDs := make([]uuid.UUID, 3)
	for i := 0; i < 3; i++ {
		patientIDs[i] = uuid.New()
		if err := repo.db.Create(&models.TwinState{
			ID:           uuid.New(),
			PatientID:    patientIDs[i],
			StateVersion: 1,
			UpdateSource: "TEST",
			UpdatedAt:    now.AddDate(0, 0, -i),
		}).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	// Make one patient's classification fail
	tracker.errOn = map[string]bool{patientIDs[1].String(): true}
	tracker.errValue = errSimulated()

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run should not fail when individual patients error: %v", err)
	}
	if got := tracker.count.Load(); got != 3 {
		t.Errorf("all 3 patients should be attempted, got %d", got)
	}
}

func TestBPContextDailyBatch_RespectsContextCancel(t *testing.T) {
	job, repo, tracker := setupBatchJobTest(t, &clients.KB20PatientProfile{
		PatientID:        "shared",
		SBP14dMean:       ptrFloat(120),
		DBP14dMean:       ptrFloat(75),
		ClinicSBPMean:    ptrFloat(118),
		ClinicDBPMean:    ptrFloat(74),
		ClinicReadings:   2,
		HomeReadings:     14,
		HomeDaysWithData: 7,
	})

	// Seed many patients so the batch takes measurable time
	now := time.Now().UTC()
	for i := 0; i < 100; i++ {
		if err := repo.db.Create(&models.TwinState{
			ID:           uuid.New(),
			PatientID:    uuid.New(),
			StateVersion: 1,
			UpdateSource: "TEST",
			UpdatedAt:    now.AddDate(0, 0, -i%30),
		}).Error; err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before run starts

	err := job.Run(ctx)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	// At most a few patients may have been picked up before cancel
	// propagated; the strict expectation is "much less than 100".
	if tracker.count.Load() >= 100 {
		t.Errorf("batch should have aborted on cancel, processed %d/100", tracker.count.Load())
	}
}
```

The test references `repo.db` directly — to enable this, the `BPContextRepository` struct must export the `db` field OR the test must use a getter. The simplest fix is to add a `DB()` helper on the repository. Add this to `bp_context_repository.go`:

```go
// DB returns the underlying GORM handle. Intended for tests and admin
// utilities that need raw query access; production code should call
// repository methods.
func (r *BPContextRepository) DB() *gorm.DB { return r.db }
```

And update the test to use `repo.DB().Create(...)` instead of `repo.db.Create(...)`.

- [ ] **Step 2: Verify tests fail**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestBPContextDailyBatch" -v 2>&1 | head -10`
Expected: compilation error — `BPContextDailyBatch`, `NewBPContextDailyBatch` undefined.

- [ ] **Step 3: Define the orchestrator interface and implement the job**

The job needs to depend on the orchestrator via an interface (so tests can stub it via `trackingOrchestrator`). Add the interface to `bp_context_batch_job.go`:

```go
// bp_context_batch_job.go
package services

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/models"
)

// BPContextClassifier is the narrow interface the batch job needs from
// the orchestrator. Defined here so tests can stub it.
type BPContextClassifier interface {
	Classify(ctx context.Context, patientID string) (*models.BPContextClassification, error)
}

// BPContextDailyBatch classifies every active patient once per run.
// Active = twin_state.UpdatedAt within the configured window.
type BPContextDailyBatch struct {
	repo         *BPContextRepository
	classifier   BPContextClassifier
	activeWindow time.Duration
	concurrency  int
	log          *zap.Logger
}

// NewBPContextDailyBatch constructs the job.
func NewBPContextDailyBatch(
	repo *BPContextRepository,
	classifier BPContextClassifier,
	activeWindow time.Duration,
	concurrency int,
	log *zap.Logger,
) *BPContextDailyBatch {
	if concurrency < 1 {
		concurrency = 1
	}
	return &BPContextDailyBatch{
		repo:         repo,
		classifier:   classifier,
		activeWindow: activeWindow,
		concurrency:  concurrency,
		log:          log,
	}
}

// Name implements BatchJob.
func (j *BPContextDailyBatch) Name() string { return "bp_context_daily" }

// Run implements BatchJob. Fetches active patient IDs, classifies each
// with bounded concurrency, and tolerates per-patient errors (logged but
// not propagated). Returns context.Canceled if the context is cancelled.
func (j *BPContextDailyBatch) Run(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	patientIDs, err := j.repo.ListActivePatientIDs(j.activeWindow)
	if err != nil {
		return err
	}

	j.log.Info("BP context batch starting",
		zap.Int("patients", len(patientIDs)),
		zap.Int("concurrency", j.concurrency))

	if len(patientIDs) == 0 {
		return nil
	}

	// Bounded concurrency via a semaphore channel.
	sem := make(chan struct{}, j.concurrency)
	var wg sync.WaitGroup
	var processed, errored int32
	var processedMu sync.Mutex

	for _, pid := range patientIDs {
		if ctx.Err() != nil {
			break
		}
		select {
		case <-ctx.Done():
			break
		case sem <- struct{}{}:
		}
		if ctx.Err() != nil {
			<-sem
			break
		}

		wg.Add(1)
		go func(patientID string) {
			defer wg.Done()
			defer func() { <-sem }()

			if _, err := j.classifier.Classify(ctx, patientID); err != nil {
				j.log.Warn("BP context classification failed in batch",
					zap.String("patient_id", patientID),
					zap.Error(err))
				processedMu.Lock()
				errored++
				processedMu.Unlock()
				return
			}
			processedMu.Lock()
			processed++
			processedMu.Unlock()
		}(pid)
	}

	wg.Wait()

	if ctx.Err() != nil {
		j.log.Warn("BP context batch cancelled",
			zap.Int32("processed", processed),
			zap.Int32("errored", errored),
			zap.Int("total", len(patientIDs)))
		return ctx.Err()
	}

	j.log.Info("BP context batch complete",
		zap.Int32("processed", processed),
		zap.Int32("errored", errored),
		zap.Int("total", len(patientIDs)))
	return nil
}
```

- [ ] **Step 4: Run job tests**

Run: `cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin && go test ./internal/services/ -run "TestBPContextDailyBatch" -v`
Expected: 3/3 PASS.

Then run the full services suite to confirm no regression:
```bash
go test ./internal/services/ -count=1 2>&1 | tail -10
```
Expected: all pass.

- [ ] **Step 5: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/services/bp_context_batch_job.go
git add kb-26-metabolic-digital-twin/internal/services/bp_context_batch_job_test.go
git add kb-26-metabolic-digital-twin/internal/services/bp_context_repository.go
git commit -m "feat(kb26): BPContextDailyBatch job for scheduler

Implements BatchJob: lists active patients (twin_state updated in last
30 days), classifies each via the orchestrator with bounded concurrency
(default 10 in flight), tolerates per-patient errors, and respects
context cancellation. Adds DB() accessor to BPContextRepository for
test seeding. Wires through BPContextClassifier interface so tests can
stub the orchestrator without instantiating real KB-20/KB-21 clients."
```

---

## Task 7: Batch Metrics

Add Prometheus metrics for the batch job and the new KB-19 publisher.

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/metrics/collector.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_batch_job.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_batch_job_test.go`
- Modify: `kb-26-metabolic-digital-twin/internal/clients/kb19_client.go`

- [ ] **Step 1: Add metric fields to Collector**

In `collector.go`, add four new fields following the Phase 2 BPClassify pattern (exported fields, registered via promauto inline in NewCollector):

```go
// BP context batch metrics
BPBatchDuration         prometheus.Histogram
BPBatchPatientsTotal    *prometheus.CounterVec  // labels: outcome (success|error)
BPBatchErrors           prometheus.Counter
KB19PublishLatency      prometheus.Histogram
KB19PublishErrors       prometheus.Counter
```

In `NewCollector`, register them via `promauto`:

```go
BPBatchDuration: promauto.NewHistogram(prometheus.HistogramOpts{
    Name:    "kb26_bp_batch_duration_seconds",
    Help:    "End-to-end duration of one BP context daily batch run",
    Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s -> 1024s
}),
BPBatchPatientsTotal: promauto.NewCounterVec(
    prometheus.CounterOpts{
        Name: "kb26_bp_batch_patients_total",
        Help: "Patients processed by the BP context batch, by outcome",
    },
    []string{"outcome"},
),
BPBatchErrors: promauto.NewCounter(prometheus.CounterOpts{
    Name: "kb26_bp_batch_errors_total",
    Help: "Number of fatal BP batch failures (does not include per-patient errors)",
}),
KB19PublishLatency: promauto.NewHistogram(prometheus.HistogramOpts{
    Name:    "kb26_kb19_publish_latency_seconds",
    Help:    "Latency of POST to KB-19 /api/v1/events from KB-26",
    Buckets: prometheus.DefBuckets,
}),
KB19PublishErrors: promauto.NewCounter(prometheus.CounterOpts{
    Name: "kb26_kb19_publish_errors_total",
    Help: "Number of failed KB-19 event publishes",
}),
```

- [ ] **Step 2: Instrument the batch job**

Add a `metrics *metrics.Collector` field to `BPContextDailyBatch` and pass it through `NewBPContextDailyBatch` (placed last):

```go
func NewBPContextDailyBatch(
    repo *BPContextRepository,
    classifier BPContextClassifier,
    activeWindow time.Duration,
    concurrency int,
    log *zap.Logger,
    metricsCollector *metrics.Collector,
) *BPContextDailyBatch {
```

In `Run`, wrap the body:

```go
func (j *BPContextDailyBatch) Run(ctx context.Context) error {
    start := time.Now()
    defer func() {
        if j.metrics != nil {
            j.metrics.BPBatchDuration.Observe(time.Since(start).Seconds())
        }
    }()

    // ... existing body up to where errors are counted ...

    // After each successful Classify call:
    if j.metrics != nil {
        j.metrics.BPBatchPatientsTotal.WithLabelValues("success").Inc()
    }

    // After each failing Classify call:
    if j.metrics != nil {
        j.metrics.BPBatchPatientsTotal.WithLabelValues("error").Inc()
    }

    // If ListActivePatientIDs fails:
    if err != nil {
        if j.metrics != nil {
            j.metrics.BPBatchErrors.Inc()
        }
        return err
    }
```

Add `"kb-26-metabolic-digital-twin/internal/metrics"` import.

- [ ] **Step 3: Update batch job tests to pass nil metrics**

Update `setupBatchJobTest` and any direct calls to pass `nil` for the metrics collector (the orchestrator already does this).

- [ ] **Step 4: Instrument the KB-19 client**

In `kb19_client.go`, add an optional metrics collector. Update `NewKB19Client` signature:

```go
func NewKB19Client(baseURL string, timeout time.Duration, log *zap.Logger, metricsCollector *metrics.Collector) *KB19Client {
```

And add a `metrics` field. In `post()`, wrap with latency observation:

```go
func (c *KB19Client) post(ctx context.Context, event KB19Event) error {
    start := time.Now()
    defer func() {
        if c.metrics != nil {
            c.metrics.KB19PublishLatency.Observe(time.Since(start).Seconds())
        }
    }()

    // ... existing body ...

    // On any error path:
    if err != nil {
        if c.metrics != nil {
            c.metrics.KB19PublishErrors.Inc()
        }
        return ...
    }
```

Update the existing kb19_client_test.go to pass `nil` for the collector.

Add `"kb-26-metabolic-digital-twin/internal/metrics"` import.

- [ ] **Step 5: Run all affected tests**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go test ./internal/clients/ -count=1 -v 2>&1 | tail -10
go test ./internal/services/ -count=1 2>&1 | tail -10
```
Expected: all pass.

- [ ] **Step 6: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/metrics/collector.go
git add kb-26-metabolic-digital-twin/internal/services/bp_context_batch_job.go
git add kb-26-metabolic-digital-twin/internal/services/bp_context_batch_job_test.go
git add kb-26-metabolic-digital-twin/internal/clients/kb19_client.go
git add kb-26-metabolic-digital-twin/internal/clients/kb19_client_test.go
git commit -m "feat(kb26): metrics for BP batch and KB-19 publisher

kb26_bp_batch_duration_seconds — histogram (1s..1024s exponential)
kb26_bp_batch_patients_total{outcome=success|error} — counter
kb26_bp_batch_errors_total — fatal batch failures (not per-patient)
kb26_kb19_publish_latency_seconds — KB-19 POST latency histogram
kb26_kb19_publish_errors_total — KB-19 failures

Nil-safe throughout: tests pass nil and instrumentation is no-op."
```

---

## Task 8: Config Env Vars

Add the new environment variables for the KB-19 client and the batch scheduler.

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/config/config.go`

- [ ] **Step 1: Read the current config**

Read the file. Note the existing pattern (`getEnv`, `getEnvAsInt`, `time.Duration` conversion).

- [ ] **Step 2: Add fields**

```go
// KB-19
KB19URL       string
KB19TimeoutMS int

// Batch scheduler
BPBatchEnabled       bool
BPBatchHourUTC       int
BPBatchConcurrency   int
BPActiveWindowDays   int
```

In `Load()`:

```go
cfg.KB19URL = getEnv("KB19_URL", "http://localhost:8103")
cfg.KB19TimeoutMS = getEnvAsInt("KB19_TIMEOUT_MS", 3000)
cfg.BPBatchEnabled = getEnv("BP_BATCH_ENABLED", "true") == "true"
cfg.BPBatchHourUTC = getEnvAsInt("BP_BATCH_HOUR_UTC", 2)
cfg.BPBatchConcurrency = getEnvAsInt("BP_BATCH_CONCURRENCY", 10)
cfg.BPActiveWindowDays = getEnvAsInt("BP_ACTIVE_WINDOW_DAYS", 30)
```

Add a `KB19Timeout()` accessor following the existing `KB22SignalTimeoutMS` accessor pattern (if one exists; otherwise just expose `KB19TimeoutMS` directly):

```go
func (c *Config) KB19Timeout() time.Duration {
    return time.Duration(c.KB19TimeoutMS) * time.Millisecond
}
```

- [ ] **Step 3: Build to verify**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go build ./...
```
Expected: clean build.

- [ ] **Step 4: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/internal/config/config.go
git commit -m "feat(kb26): config env vars for KB-19 client and BP batch

KB19_URL (default localhost:8103, KB-19 default port)
KB19_TIMEOUT_MS (default 3000)
BP_BATCH_ENABLED (default true)
BP_BATCH_HOUR_UTC (default 2)
BP_BATCH_CONCURRENCY (default 10)
BP_ACTIVE_WINDOW_DAYS (default 30)"
```

---

## Task 9: main.go Wiring + Top-Level Context + Scheduler Goroutine

The biggest single edit. Introduces top-level context, instantiates KB-19 client, instantiates scheduler, registers the batch job, starts the scheduler goroutine.

**Files:**
- Modify: `kb-26-metabolic-digital-twin/main.go`

- [ ] **Step 1: Read current main.go end-to-end**

Note specifically:
- The exact line where `signalConsumer` is started (lines 130-144 per exploration)
- The exact line of the `<-quit` shutdown block (lines 158-163 per exploration)
- The exact line where the orchestrator is instantiated and passed to NewServer
- Whether `quarterlyAggregator` is still present as `_ = quarterlyAggregator` (line 96 per exploration)

- [ ] **Step 2: Introduce top-level cancellable context**

Near the start of main (after logger init), add:

```go
// Top-level context for graceful shutdown — propagated to scheduler,
// Kafka consumer, and HTTP server.
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
```

Replace the existing local `consumerCtx, consumerCancel := context.WithCancel(...)` with `ctx`. Remove the local cancel since the top-level one handles it.

- [ ] **Step 3: Instantiate KB-19 client**

After the existing KB-20 and KB-21 client instantiations:

```go
kb19Client := clients.NewKB19Client(cfg.KB19URL, cfg.KB19Timeout(), logger, metricsCollector)
```

- [ ] **Step 4: Pass kb19Client to NewBPContextOrchestrator**

The Phase 2 call currently looks like:
```go
bpContextOrch := services.NewBPContextOrchestrator(kb20Client, kb21Client, bpContextRepo, bpThresholds, logger, metricsCollector)
```

Add the publisher as the new last argument:
```go
bpContextOrch := services.NewBPContextOrchestrator(kb20Client, kb21Client, bpContextRepo, bpThresholds, logger, metricsCollector, kb19Client)
```

- [ ] **Step 5: Instantiate scheduler and register the BP context job**

```go
// BP context daily batch (Phase 3)
bpBatchJob := services.NewBPContextDailyBatch(
    bpContextRepo,
    bpContextOrch,
    time.Duration(cfg.BPActiveWindowDays)*24*time.Hour,
    cfg.BPBatchConcurrency,
    logger,
    metricsCollector,
)
batchScheduler := services.NewBatchScheduler(logger)
batchScheduler.Register(bpBatchJob)
```

- [ ] **Step 6: Start the scheduler goroutine if enabled**

After the HTTP server goroutine, add:

```go
if cfg.BPBatchEnabled {
    go func() {
        // Compute the interval to next run at the configured UTC hour.
        // For simplicity we use a fixed 24h interval — production sites
        // can tune the start time by deploying at the desired hour.
        scheduleInterval := computeNextScheduleInterval(time.Now().UTC(), cfg.BPBatchHourUTC)
        time.Sleep(scheduleInterval) // align first run to BP_BATCH_HOUR_UTC
        batchScheduler.StartLoop(ctx, 24*time.Hour)
    }()
    logger.Info("BP context batch scheduler enabled",
        zap.Int("hour_utc", cfg.BPBatchHourUTC),
        zap.Int("concurrency", cfg.BPBatchConcurrency),
        zap.Int("active_window_days", cfg.BPActiveWindowDays))
} else {
    logger.Info("BP context batch scheduler disabled (BP_BATCH_ENABLED=false)")
}
```

Add the helper function at the bottom of main.go (or in a small new file `internal/services/schedule_time.go` if you prefer not to put utility code in main):

```go
// computeNextScheduleInterval returns the duration until the next
// occurrence of the given UTC hour (0-23). If the current time is
// already past that hour today, it returns the duration until tomorrow's
// occurrence.
func computeNextScheduleInterval(now time.Time, hourUTC int) time.Duration {
    next := time.Date(now.Year(), now.Month(), now.Day(), hourUTC, 0, 0, 0, time.UTC)
    if !next.After(now) {
        next = next.Add(24 * time.Hour)
    }
    return next.Sub(now)
}
```

- [ ] **Step 7: Update the shutdown block to cancel the top-level context**

Modify the existing `<-quit` block:

```go
<-quit
logger.Info("Shutting down KB-26 Metabolic Digital Twin Service")
cancel() // propagates to scheduler, consumer, all goroutines
// give goroutines a moment to exit cleanly before defer chain runs
time.Sleep(500 * time.Millisecond)
```

- [ ] **Step 8: Build and run full test suite**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go build ./...
go test ./... -count=1 2>&1 | tail -20
```
Expected: clean build, all tests pass.

- [ ] **Step 9: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/main.go
git commit -m "feat(kb26): wire BP batch scheduler at startup with top-level context

- Introduces top-level cancellable context propagated to scheduler,
  Kafka consumer, and HTTP server (was: hard exit, defer-only)
- Instantiates KB-19 publisher client, passes to BP context orchestrator
- Instantiates BatchScheduler, registers BPContextDailyBatch
- Starts scheduler goroutine that aligns first run to BP_BATCH_HOUR_UTC
  then loops every 24 hours
- Cancellation propagates on SIGTERM with 500ms grace period

BP_BATCH_ENABLED=false disables the scheduler entirely (manual HTTP
endpoint still works as the on-demand override)."
```

---

## Task 10: Deprecate KB-23 Publisher Methods

The KB-23-side `PublishMaskedHTNDetected` and `PublishPhenotypeChanged` from Phase 2 commit `4932e56f` are no longer the publication path. Mark them deprecated but DO NOT delete them — they may be useful as future hooks.

**Files:**
- Modify: `kb-23-decision-cards/internal/services/kb19_publisher.go`

- [ ] **Step 1: Add deprecation comments**

Find the two methods added in Phase 2. Above each, add:

```go
// Deprecated: BP context phenotype events are published by KB-26's
// BPContextOrchestrator directly via its own KB-19 client (Phase 3).
// This method is retained for future use cases where KB-23 might need
// to publish unrelated events with the same envelope shape, but is not
// called by any production code path as of Phase 3.
func (p *KB19Publisher) PublishMaskedHTNDetected(...) error { ... }
```

(Same for `PublishPhenotypeChanged`.)

- [ ] **Step 2: Build and test**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards
go build ./...
go test ./... -count=1 2>&1 | tail -10
```
Expected: clean build, all tests pass (deprecation comments don't change behavior, just produce `staticcheck` SA1019 warnings if linting is enabled).

- [ ] **Step 3: Commit**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-23-decision-cards/internal/services/kb19_publisher.go
git commit -m "chore(kb23): deprecate masked HTN publisher methods

KB-26's BPContextOrchestrator now owns phenotype event publication
directly (Phase 3 Decision 3). The KB-23 methods from Phase 2 commit
4932e56f are not called by any production path. Marked as deprecated
but not deleted — they remain available as hooks for future use cases."
```

---

## Task 11: End-to-End Smoke Verification

Final verification that everything compiles, tests pass, and the wiring is complete.

**Files:** none — this is a verification task.

- [ ] **Step 1: KB-26 full build**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go build ./...
```
Expected: clean.

- [ ] **Step 2: KB-26 full test suite**

```bash
go test ./... -count=1 2>&1 | tail -20
```
Expected: all packages pass.

- [ ] **Step 3: KB-23 full test suite (regression check)**

```bash
cd ../kb-23-decision-cards
go test ./... -count=1 2>&1 | tail -10
```
Expected: all tests pass (deprecations don't break anything).

- [ ] **Step 4: Verify wiring grep**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
grep -n "BatchScheduler\|BPContextDailyBatch\|kb19Client" main.go
```
Expected: shows scheduler instantiation, job registration, kb19Client creation, and the goroutine launch.

- [ ] **Step 5: Final commit (only if any verification surfaced fixes)**

If all checks pass, no commit is needed.

---

## Plan Summary

| Task | Files | New Tests | Outcome |
|------|-------|-----------|---------|
| 1 | 2 create | 4 | KB-19 client publishes events from KB-26 |
| 2 | 1 modify | 0 | Standalone index on twin_state.updated_at |
| 3 | 2 modify | 2 | ListActivePatientIDs query with dedup |
| 4 | 2 modify | 4 | Orchestrator emits phenotype change events |
| 5 | 2 create | 4 | Generic batch scheduler with BatchJob interface |
| 6 | 2 create + 1 modify | 3 | BP context daily batch job with bounded concurrency |
| 7 | 4 modify | 0 (build only) | Prometheus metrics for batch + KB-19 |
| 8 | 1 modify | 0 (build only) | Env vars for KB-19 and batch |
| 9 | 1 modify | 0 (build only) | main.go wires scheduler + top-level context |
| 10 | 1 modify | 0 (build only) | Deprecate KB-23 publisher methods |
| 11 | 0 | 0 | End-to-end verification |

**Total:** 6 create, 9 modify = 15 files. ~17 new test cases. 10 commits expected.

## What Phase 3 Delivers

After all 11 tasks pass, the production system will:

1. **Run the BP context daily batch at 02:00 UTC every day** for every patient with `twin_state` activity in the last 30 days
2. **Publish `MASKED_HTN_DETECTED` events to KB-19** when a patient is first classified as masked HTN or MUCH
3. **Publish `BP_PHENOTYPE_CHANGED` events to KB-19** when any phenotype transition occurs (including WCH → SH progression that the spec called out)
4. **Persist daily snapshots** in `bp_context_history`, enabling future progression analytics
5. **Expose Prometheus metrics** for batch duration, per-patient outcomes, and KB-19 publication latency
6. **Degrade gracefully** when KB-19, KB-20, or KB-21 are unreachable (warnings logged, batch continues, no crashes)
7. **Honor SIGTERM gracefully** via top-level context cancellation propagated to scheduler, Kafka consumer, and HTTP server

The existing `POST /api/v1/kb26/bp-context/:patientId` endpoint continues to work as the on-demand override for clinicians who need a fresh classification mid-visit.

## What Phase 3 Does NOT Deliver

Per Phase 2's deferred list, these remain out of scope:

1. **Per-reading BP storage** — classifier still synthesizes readings from KB-20 aggregates
2. **Card YAML fragment templates** — card text remains hardcoded
3. **Hysteresis engine integration** — daily-batch cadence damps but does not eliminate phenotype flapping
4. **Composite card aggregation** — KB-23 still emits separate cards per phenotype
5. **WCH progression scheduled job** — Phase 3 publishes BP_PHENOTYPE_CHANGED events, but no consumer of those events specifically watches for "WCH for 6+ months → consider intensification"
6. **Confidence tier integration** — `Confidence string` not yet mapped to KB-23's `ConfidenceTier` enum
7. **Quarterly aggregator scheduling** — the dead reference in main.go could be wrapped in a `BatchJob` and registered with the same scheduler, but Phase 3 doesn't do it

These are independently scopable as Phase 4+ projects.
