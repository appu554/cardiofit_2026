# Masked HTN Phase 4 — Quality & Correctness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Address quality and correctness gaps in the Phase 1+2+3 masked HTN feature, in priority order: graceful HTTP shutdown (data integrity in K8s deploys), real per-reading BP storage wiring (the cross-domain insight that makes cards actionable), phenotype stability engine (eliminates flapping), confidence tier mapping (consistency with rest of KB-23), card text fragmentation (clinical author agency), and operational readiness review (production deployment prerequisites).

**Architecture:** Phase 4 is NOT one coherent feature. It is a collection of independently-scopable improvements, each closing a specific gap from the Phase 3 deferred list and the Phase 1+2+3 code reviews. Tasks are grouped into 6 sub-projects (P1, P3, P2, P8, P7, P5) that can be executed in any order. A planner choosing to ship only some of them must read each sub-project's "depends on" line to confirm prerequisite work is in place.

**Pre-requisite:** Phase 1+2+3 commits through `7bd6280c` plus the Phase 4 hotfix commits for P0 (selection bias dampening) and P4 (per-market batch hours) are present. KB-26 and KB-23 build and test green.

**Tech Stack:** Go 1.25 (Gin, GORM, Zap, Prometheus), PostgreSQL 15

---

## Locked Decisions

These are NOT open questions in this plan. They are fixed constraints derived from the Phase 4 exploration and the reviewer's findings.

### Decision 1: KB-20 LabEntry is the BP reading source — no new table

The Phase 3 "no per-reading store anywhere" assumption was wrong. KB-20 has `LabEntry` rows with `LabType="SBP"`/`"DBP"`, `Value`, `MeasuredAt`, `Source`, and `FHIRObservationID`. P3 wires KB-26 to query these rather than synthesize from aggregates. **No new `bp_readings` table is created.** This drops P3 from "multi-week design + ingestion + retention" to "add a KB-20 endpoint + KB-26 client method."

### Decision 2: The phenotype stability engine is built from scratch, in a shared package

The reviewer's framing implied an existing engine. There isn't one. KB-23's `HysteresisEngine` is for MCU gate transitions (different domain). P2 builds a generic `pkg/stability/` package consumed by the BP context orchestrator first. The package is **designed for reuse** so future systems (phenotype clustering, engagement classification) can adopt it, but Phase 4 only wires it to the BP context flow — no speculative integration.

### Decision 3: Graceful shutdown adopts the KB-23 pattern verbatim

KB-23's `main.go:157-200` is the reference. Phase 4 P1 replaces KB-26's `router.Run()` + `time.Sleep(500ms)` with `http.Server{Handler: server.Router}` + `httpServer.Shutdown(shutdownCtx)` with a 15-second timeout. **No new abstraction.** Match the existing pattern character-for-character because consistency across KB-* services matters more than novelty.

### Decision 4: BatchScheduler exposes a Drain method for shutdown coordination

The Phase 3 batch scheduler doesn't expose any way for `main.go` to wait for an in-flight batch run during shutdown. P1 adds a `Drain(ctx)` method that blocks until any currently-executing `RunOnce` returns, with the shutdown context as the deadline. The `BPContextDailyBatch.Run`'s internal `wg.Wait()` already drains in-flight goroutines on context cancel, so the scheduler's `Drain` is just "wait for the current `RunOnce` to return." Tested with a stub that takes a measurable time to run.

### Decision 5: Confidence tier integration is opt-in per template, not global

HTN safety templates explicitly set `bypasses_confidence_gate: true` because BP threshold violations are guideline-concordant facts, not Bayesian inferences. P8 maps BP context confidence ("HIGH"/"MODERATE"/"LOW") to ConfidenceTier (FIRM/PROBABLE/POSSIBLE) but **only for the masked HTN cards**, and **does not modify the existing HTN safety templates** (`bp_above_target`, `bp_severe`, etc.). The mapping is additive: the masked HTN card builder calls a new helper `confidenceStringToTier()` and sets `card.ConfidenceTier` accordingly. Existing cards are unchanged.

### Decision 6: Card YAML fragment migration covers only the 8 new BP context cards

P7 migrates the 8 hardcoded card rationales in `masked_htn_cards.go` to YAML templates. **It does NOT touch the existing HTN safety templates.** The migration is complete only for the BP context family; other card families remain hardcoded (where they were hardcoded) or template-driven (where they were template-driven). This avoids scope creep and keeps the migration testable as a self-contained unit.

### Decision 7: Composite card aggregation uses the existing service, not a new one

The exploration confirmed `CompositeCardService.Synthesize(ctx, patientID)` already exists. P9 wires it to fire after the BP context daily batch completes for each patient — adding one call to the orchestrator's `Classify` post-emit hook. **No new composite logic.** The existing 72-hour window and most-restrictive-gate behavior are accepted as-is.

### Decision 8: P5 (Operational Readiness Review) is a documentation deliverable, not code

P5 ships a runbook document (`docs/runbooks/masked-htn-operations.md`) covering: alerting thresholds for the 8 Prometheus metrics, on-call escalation paths for batch failures, canary deployment plan (start with 1% of patients via `BP_BATCH_PERCENT`), feature flag wrapping for the scheduler, rollback procedure. **No new code.** The deliverable is reviewable Markdown.

---

## File Structure

### KB-26 changes
| Action | File | Sub-project |
|--------|------|-------------|
| Modify | `main.go` | P1 — switch to `http.Server` + `Shutdown` |
| Modify | `internal/services/batch_scheduler.go` | P1 — add `Drain` method |
| Modify | `internal/services/batch_scheduler_test.go` | P1 — drain test |
| Create | `pkg/stability/engine.go` | P2 — generic stability engine |
| Create | `pkg/stability/engine_test.go` | P2 |
| Create | `pkg/stability/policies.go` | P2 — dwell/flap/override types |
| Modify | `internal/services/bp_context_orchestrator.go` | P2, P3, P8 |
| Modify | `internal/services/bp_context_orchestrator_test.go` | P2, P3, P8 |
| Modify | `internal/clients/kb20_client.go` | P3 — add `FetchBPReadings` method |
| Modify | `internal/clients/kb20_client_test.go` (new) | P3 |

### KB-20 changes
| Action | File | Sub-project |
|--------|------|-------------|
| Create | `internal/api/bp_reading_handlers.go` | P3 — `GET /api/v1/patient/:id/bp-readings` |
| Modify | `internal/api/routes.go` | P3 — register route |
| Create | `internal/services/bp_reading_query.go` | P3 — query LabEntry by SBP/DBP types |
| Create | `internal/services/bp_reading_query_test.go` | P3 |

### KB-23 changes
| Action | File | Sub-project |
|--------|------|-------------|
| Modify | `internal/services/masked_htn_cards.go` | P8, P7 |
| Modify | `internal/services/masked_htn_cards_test.go` | P8, P7 |
| Create | `templates/bp_context/masked_hypertension.yaml` | P7 |
| Create | `templates/bp_context/masked_uncontrolled.yaml` | P7 |
| Create | `templates/bp_context/white_coat_hypertension.yaml` | P7 |
| Create | `templates/bp_context/white_coat_uncontrolled.yaml` | P7 |
| Create | `templates/bp_context/masked_htn_morning_surge_compound.yaml` | P7 |
| Create | `templates/bp_context/sustained_htn_morning_surge.yaml` | P7 |
| Create | `templates/bp_context/selection_bias_warning.yaml` | P7 |
| Create | `templates/bp_context/medication_timing.yaml` | P7 |

### Documentation
| Action | File | Sub-project |
|--------|------|-------------|
| Create | `docs/runbooks/masked-htn-operations.md` | P5 |

**Total: 18 create, 8 modify across 4 services + 1 doc = 27 files**

---

# Sub-project P1: Graceful HTTP Shutdown + Batch Drain

**Priority:** Highest. Data integrity risk during K8s rolling deploys.
**Effort:** ~3 tasks, ~4 hours.
**Depends on:** Phase 3 (BatchScheduler).

## Task P1.1: Switch KB-26 main.go to explicit http.Server

**Files:**
- Modify: `kb-26-metabolic-digital-twin/main.go`

- [ ] **Step 1: Read KB-23's reference shutdown pattern**

```bash
grep -n "http.Server\|Shutdown" backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/main.go
```

Confirm the structure: KB-23 has an explicit `httpServer := &http.Server{Addr: ..., Handler: server.Router, ...}`, calls `httpServer.ListenAndServe()` in a goroutine, and uses `httpServer.Shutdown(shutdownCtx)` with a 15s `context.WithTimeout`.

- [ ] **Step 2: Replace KB-26's `router.Run` with explicit http.Server**

In KB-26's `main.go`, locate the existing HTTP server start block (around line 130):

```go
go func() {
    addr := ":" + cfg.Server.Port
    logger.Info("HTTP server starting", zap.String("address", addr))
    if err := server.Router.Run(addr); err != nil {
        logger.Fatal("HTTP server failed", zap.Error(err))
    }
}()
```

Replace with:

```go
httpServer := &http.Server{
    Addr:         ":" + cfg.Server.Port,
    Handler:      server.Router,
    ReadTimeout:  15 * time.Second,
    WriteTimeout: 30 * time.Second,
    IdleTimeout:  60 * time.Second,
}

go func() {
    logger.Info("HTTP server starting", zap.String("addr", httpServer.Addr))
    if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        logger.Fatal("HTTP server failed", zap.Error(err))
    }
}()
```

Add `"net/http"` to imports if not already present.

- [ ] **Step 3: Replace the time.Sleep hack with proper Shutdown**

Locate the existing shutdown block:

```go
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
<-quit

logger.Info("Shutting down KB-26 Metabolic Digital Twin Service")
cancel() // propagate shutdown to scheduler, consumer, all goroutines
// Give goroutines a moment to exit cleanly before the defer chain runs.
time.Sleep(500 * time.Millisecond)
```

Replace with:

```go
quit := make(chan os.Signal, 1)
signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
sig := <-quit

logger.Info("Shutdown signal received", zap.String("signal", sig.String()))
logger.Info("Shutting down KB-26 Metabolic Digital Twin Service")

// Cancel top-level context — propagates to scheduler, Kafka consumer, etc.
cancel()

// Drain in-flight batch work (waits for any currently-executing RunOnce)
batchScheduler.Drain()

// Graceful HTTP shutdown — waits for in-flight requests with a 15s deadline
shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
defer shutdownCancel()
if err := httpServer.Shutdown(shutdownCtx); err != nil {
    logger.Error("HTTP server shutdown error", zap.Error(err))
} else {
    logger.Info("HTTP server shutdown completed")
}
```

The `batchScheduler.Drain()` call depends on Task P1.2 below.

- [ ] **Step 4: Build to verify**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go build ./...
```
Expected: clean build (after Task P1.2 completes — these two tasks must land together as one commit).

## Task P1.2: Add Drain method to BatchScheduler

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/services/batch_scheduler.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/batch_scheduler_test.go`

- [ ] **Step 1: Write the failing test**

Append to `batch_scheduler_test.go`:

```go
func TestBatchScheduler_Drain_WaitsForInFlightRun(t *testing.T) {
    job := &stubBatchJob{name: "slow", delay: 200 * time.Millisecond}
    sched := NewBatchScheduler(zap.NewNop())
    sched.Register(job)

    // Start a long-running RunOnce in a goroutine.
    runDone := make(chan struct{})
    go func() {
        _ = sched.RunOnce(context.Background())
        close(runDone)
    }()

    // Give the run a chance to start.
    time.Sleep(20 * time.Millisecond)

    // Drain should block until the in-flight run finishes.
    drainStart := time.Now()
    sched.Drain()
    drainDuration := time.Since(drainStart)

    if drainDuration < 100*time.Millisecond {
        t.Errorf("Drain returned too early: %v (expected at least 100ms)", drainDuration)
    }

    // RunOnce should have completed by now.
    select {
    case <-runDone:
        // Good
    case <-time.After(50 * time.Millisecond):
        t.Fatal("RunOnce did not complete after Drain returned")
    }
}

func TestBatchScheduler_Drain_NoOpWhenIdle(t *testing.T) {
    sched := NewBatchScheduler(zap.NewNop())
    sched.Register(&stubBatchJob{name: "idle"})

    // Drain when nothing is running should return immediately.
    drainStart := time.Now()
    sched.Drain()
    if time.Since(drainStart) > 10*time.Millisecond {
        t.Errorf("idle Drain blocked unexpectedly: %v", time.Since(drainStart))
    }
}
```

- [ ] **Step 2: Verify tests fail**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-26-metabolic-digital-twin
go test ./internal/services/ -run "TestBatchScheduler_Drain" -v 2>&1 | head -10
```
Expected: compilation error — `Drain` undefined on `*BatchScheduler`.

- [ ] **Step 3: Implement Drain**

In `batch_scheduler.go`, modify the `BatchScheduler` struct to track in-flight `RunOnce` calls:

```go
type BatchScheduler struct {
    jobs    []BatchJob
    mu      sync.RWMutex
    log     *zap.Logger
    runWg   sync.WaitGroup // tracks active RunOnce calls
}
```

Modify `RunOnce` to wrap its body in `runWg.Add(1)` / `defer runWg.Done()`:

```go
func (s *BatchScheduler) RunOnce(ctx context.Context) error {
    s.runWg.Add(1)
    defer s.runWg.Done()

    s.mu.RLock()
    // ... existing body unchanged ...
}
```

Add the `Drain` method:

```go
// Drain blocks until any currently-executing RunOnce calls return. Idle
// schedulers return immediately. Used by main.go during graceful shutdown
// to ensure in-flight batch work completes (or is interrupted via context
// cancellation) before the process exits.
func (s *BatchScheduler) Drain() {
    s.runWg.Wait()
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/services/ -run "TestBatchScheduler" -v
```
Expected: all 6 tests PASS (4 from Phase 3 + 2 new).

- [ ] **Step 5: Commit P1 (combined)**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
git add kb-26-metabolic-digital-twin/main.go
git add kb-26-metabolic-digital-twin/internal/services/batch_scheduler.go
git add kb-26-metabolic-digital-twin/internal/services/batch_scheduler_test.go
git commit -m "fix(kb26): graceful HTTP shutdown with proper Server.Shutdown + batch Drain

Phase 3 used router.Run() + time.Sleep(500ms), which risked data
integrity in K8s rolling deploys: pod receives SIGTERM, in-flight batch
classifications race against the 500ms grace, and SaveSnapshot may
partially complete or never run.

Now: explicit http.Server with 15s ReadTimeout / 30s WriteTimeout,
proper httpServer.Shutdown(ctx) with 15s timeout for in-flight HTTP
requests, and BatchScheduler.Drain() waits for any currently-executing
RunOnce to finish before the process exits. The internal wg.Wait() in
BPContextDailyBatch.Run already drained per-patient goroutines on
context cancel; Drain is the missing scheduler-level coordination.

Pattern matches kb-23-decision-cards/main.go:157-200 for consistency
across KB services."
```

---

# Sub-project P3: Per-Reading BP Storage Wiring

**Priority:** High. Restores the cross-domain insight that makes cards actionable (medication timing hypothesis).
**Effort:** ~5 tasks, ~1 day.
**Depends on:** Phase 2 (orchestrator), Phase 3 (KB-20 client).

This sub-project wires KB-26 to query KB-20's existing `LabEntry` rows for individual SBP/DBP readings, replacing the synthetic-readings hack from Phase 2 Task 7.

## Task P3.1: KB-20 BP Reading Query Service

**Files:**
- Create: `kb-20-patient-profile/internal/services/bp_reading_query.go`
- Create: `kb-20-patient-profile/internal/services/bp_reading_query_test.go`

- [ ] **Step 1: Read existing LabEntry model and queries**

Read `kb-20-patient-profile/internal/models/lab_tracker.go` end-to-end. Note the `LabEntry` struct fields, the `LabType` constants (`LabTypeSBP = "SBP"`, `LabTypeDBP = "DBP"`), and how existing code queries lab entries.

- [ ] **Step 2: Write the failing test**

Create `bp_reading_query_test.go` with table-driven tests covering:
- Returns SBP and DBP readings paired by `MeasuredAt` (within ±5 minutes)
- Filters by patient ID
- Filters by `MeasuredAt > cutoff`
- Returns empty slice (not error) for unknown patient
- Distinguishes by `Source` field (CLINIC vs HOME_CUFF)

(Full test code omitted for brevity — write 5 test cases following the existing `lab_tracker_test.go` pattern.)

- [ ] **Step 3: Implement the query service**

```go
// bp_reading_query.go
package services

import (
    "math"
    "time"

    "gorm.io/gorm"

    "kb-20-patient-profile/internal/models"
)

// BPReading is a paired SBP+DBP measurement from one observation event.
type BPReading struct {
    PatientID  string    `json:"patient_id"`
    SBP        float64   `json:"sbp"`
    DBP        float64   `json:"dbp"`
    Source     string    `json:"source"`        // CLINIC | HOME_CUFF | etc.
    MeasuredAt time.Time `json:"measured_at"`
}

// BPReadingQuery fetches paired SBP+DBP readings from LabEntry rows.
// SBP and DBP are stored as separate LabEntry rows; this service pairs
// them by MeasuredAt (within a 5-minute window) and Source.
type BPReadingQuery struct {
    db *gorm.DB
}

func NewBPReadingQuery(db *gorm.DB) *BPReadingQuery {
    return &BPReadingQuery{db: db}
}

// FetchSince returns all paired BP readings for a patient since the given time.
// Unpaired readings (SBP without matching DBP within 5 minutes) are dropped.
func (q *BPReadingQuery) FetchSince(patientID string, since time.Time) ([]BPReading, error) {
    var entries []models.LabEntry
    err := q.db.Where(
        "patient_id = ? AND lab_type IN (?, ?) AND measured_at > ?",
        patientID, models.LabTypeSBP, models.LabTypeDBP, since,
    ).Order("measured_at ASC").Find(&entries).Error
    if err != nil {
        return nil, err
    }

    return pairEntries(entries, patientID), nil
}

// pairEntries groups SBP and DBP entries by (Source, MeasuredAt±5min).
func pairEntries(entries []models.LabEntry, patientID string) []BPReading {
    var paired []BPReading
    used := make(map[int]bool)

    for i, e := range entries {
        if used[i] || e.LabType != models.LabTypeSBP {
            continue
        }
        // Find the matching DBP within 5 minutes from the same source.
        for j, c := range entries {
            if used[j] || j == i || c.LabType != models.LabTypeDBP {
                continue
            }
            if c.Source != e.Source {
                continue
            }
            delta := math.Abs(c.MeasuredAt.Sub(e.MeasuredAt).Seconds())
            if delta > 300 {
                continue
            }
            paired = append(paired, BPReading{
                PatientID:  patientID,
                SBP:        e.Value.InexactFloat64(),
                DBP:        c.Value.InexactFloat64(),
                Source:     e.Source,
                MeasuredAt: e.MeasuredAt,
            })
            used[i] = true
            used[j] = true
            break
        }
    }
    return paired
}
```

- [ ] **Step 4: Run tests; commit.**

## Task P3.2: KB-20 HTTP Endpoint for BP Readings

**Files:**
- Create: `kb-20-patient-profile/internal/api/bp_reading_handlers.go`
- Modify: `kb-20-patient-profile/internal/api/routes.go`

Expose the query via `GET /api/v1/patient/:patientId/bp-readings?since=2026-04-01T00:00:00Z`.

- [ ] **Step 1: Create the handler**

```go
// bp_reading_handlers.go
package api

import (
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
)

// listBPReadings handles GET /api/v1/patient/:patientId/bp-readings?since=ISO8601
func (s *Server) listBPReadings(c *gin.Context) {
    patientID := c.Param("patientId")
    if patientID == "" {
        sendError(c, http.StatusBadRequest, "patientId is required", "MISSING_PATIENT_ID", nil)
        return
    }

    sinceStr := c.Query("since")
    if sinceStr == "" {
        // Default to last 30 days
        sinceStr = time.Now().AddDate(0, 0, -30).Format(time.RFC3339)
    }
    since, err := time.Parse(time.RFC3339, sinceStr)
    if err != nil {
        sendError(c, http.StatusBadRequest, "invalid since parameter (expected RFC3339)", "INVALID_SINCE", nil)
        return
    }

    readings, err := s.bpReadingQuery.FetchSince(patientID, since)
    if err != nil {
        sendError(c, http.StatusInternalServerError, "BP reading fetch failed", "BP_READING_FETCH_FAILED", nil)
        return
    }

    sendSuccess(c, readings, map[string]interface{}{
        "patient_id": patientID,
        "since":      since.Format(time.RFC3339),
        "count":      len(readings),
    })
}
```

- [ ] **Step 2: Register the route + wire bpReadingQuery into Server**

In `routes.go`, add: `v1.GET("/patient/:patientId/bp-readings", s.listBPReadings)`.

In `server.go`, add a `bpReadingQuery *services.BPReadingQuery` field and instantiate it in `InitServices()`.

- [ ] **Step 3: Build, test, commit.**

## Task P3.3: KB-26 Client Method to Fetch BP Readings

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/clients/kb20_client.go`
- Create: `kb-26-metabolic-digital-twin/internal/clients/kb20_client_test.go`

- [ ] **Step 1: Add `FetchBPReadings` method**

```go
type KB20BPReading struct {
    PatientID  string    `json:"patient_id"`
    SBP        float64   `json:"sbp"`
    DBP        float64   `json:"dbp"`
    Source     string    `json:"source"`
    MeasuredAt time.Time `json:"measured_at"`
}

func (c *KB20Client) FetchBPReadings(ctx context.Context, patientID string, since time.Time) ([]KB20BPReading, error) {
    url := fmt.Sprintf("%s/api/v1/patient/%s/bp-readings?since=%s",
        c.baseURL, patientID, since.Format(time.RFC3339))
    // ... standard GET pattern matching FetchProfile ...
}
```

- [ ] **Step 2: Test with httptest stub server, commit.**

## Task P3.4: Replace Synthetic Readings with Real Ones in Orchestrator

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator_test.go`

- [ ] **Step 1: Modify `Classify` to fetch real readings**

Replace the `buildBPContextInputFromProfile` synthetic-readings logic with a real fetch:

```go
// Fetch real BP readings (Phase 4 P3 — replaces Phase 2's synthetic readings).
since := time.Now().AddDate(0, 0, -30) // last 30 days
realReadings, err := o.kb20.FetchBPReadings(ctx, patientID, since)
if err != nil {
    o.log.Warn("real BP reading fetch failed; falling back to synthetic",
        zap.String("patient_id", patientID), zap.Error(err))
    // Fall through to the synthetic path below
}

input := buildBPContextInputFromReadings(profile, realReadings, engagementPhenotype)
if len(input.HomeReadings) == 0 && len(input.ClinicReadings) == 0 {
    // Real fetch empty or failed — fall back to synthetic for this run
    input = buildBPContextInputFromProfile(profile, engagementPhenotype)
}
```

The fallback path is critical: production deploys may roll out KB-20 P3.2 endpoint before/after KB-26 P3.4. The classifier must work in both states.

- [ ] **Step 2: Add `buildBPContextInputFromReadings` function**

```go
func buildBPContextInputFromReadings(
    profile *clients.KB20PatientProfile,
    readings []clients.KB20BPReading,
    engagementPhenotype string,
) BPContextInput {
    input := BPContextInput{
        PatientID:           profile.PatientID,
        IsDiabetic:          profile.IsDiabetic,
        HasCKD:              profile.HasCKD,
        OnAntihypertensives: profile.OnHTNMeds,
        EngagementPhenotype: engagementPhenotype,
    }
    if profile.MorningSurge7dAvg != nil {
        input.MorningSurge7dAvg = *profile.MorningSurge7dAvg
    }

    for _, r := range readings {
        bp := BPReading{
            SBP:       r.SBP,
            DBP:       r.DBP,
            Source:    r.Source,
            Timestamp: r.MeasuredAt,
        }
        // Tag time context for medication timing hypothesis
        hour := r.MeasuredAt.Hour()
        switch {
        case hour >= 5 && hour < 11:
            bp.TimeContext = "MORNING"
        case hour >= 17 && hour < 23:
            bp.TimeContext = "EVENING"
        }
        if r.Source == "CLINIC" || r.Source == "OFFICE" || r.Source == "HOSPITAL" {
            input.ClinicReadings = append(input.ClinicReadings, bp)
        } else {
            input.HomeReadings = append(input.HomeReadings, bp)
        }
    }
    return input
}
```

- [ ] **Step 3: Add tests for the new path**

Write tests that:
- Stub KB-20 client returns real readings → orchestrator builds correct `BPContextInput`
- Stub KB-20 client errors → fallback to synthetic path
- Stub KB-20 client returns empty → fallback to synthetic path
- Real readings produce non-empty `MedicationTimingHypothesis` (because TimeContext is now populated)

- [ ] **Step 4: Run tests; commit.**

## Task P3.5: Smoke Test Cross-Service Integration

- [ ] **Step 1: Run KB-20 + KB-26 + KB-23 full test suites**

```bash
cd /Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/knowledge-base-services
for svc in kb-20-patient-profile kb-26-metabolic-digital-twin kb-23-decision-cards; do
    echo "=== $svc ==="
    (cd $svc && go test ./... -count=1 2>&1 | tail -10)
done
```
Expected: all green.

---

# Sub-project P2: Phenotype Stability Engine

**Priority:** High. Eliminates phenotype flapping for actively-monitored patients.
**Effort:** ~6 tasks, ~1.5 days.
**Depends on:** Phase 3 (orchestrator with prior-snapshot fetch).

The exploration confirmed there is NO existing stability engine in the codebase. KB-23's `HysteresisEngine` operates on MCU gates, not phenotypes. Phase 4 builds a generic stability package from scratch in `kb-26-metabolic-digital-twin/pkg/stability/` (designed for reuse but consumed only by the BP context flow in Phase 4).

## Task P2.1: Generic Stability Package

**Files:**
- Create: `kb-26-metabolic-digital-twin/pkg/stability/policies.go`
- Create: `kb-26-metabolic-digital-twin/pkg/stability/engine.go`
- Create: `kb-26-metabolic-digital-twin/pkg/stability/engine_test.go`

- [ ] **Step 1: Define types**

```go
// policies.go
package stability

import "time"

// Decision is the stability engine's verdict on a proposed transition.
type Decision string

const (
    DecisionAccept   Decision = "ACCEPT"   // proposed becomes the new state
    DecisionDamp     Decision = "DAMP"     // proposed is held; current state continues
    DecisionOverride Decision = "OVERRIDE" // override event bypasses dwell
)

// Result is the engine's full output for a transition decision.
type Result struct {
    Decision Decision
    Reason   string
}

// Policy controls stability behavior for one consumer (e.g. BP context).
type Policy struct {
    // MinDwell is the minimum time a state must be held before another
    // transition is accepted. Transitions proposed before this elapses
    // are damped unless an override event applies.
    MinDwell time.Duration

    // FlapWindow is the lookback window for flap detection. If the
    // proposed transition would re-enter a state that was active within
    // this window, it is considered a flap and damped.
    FlapWindow time.Duration

    // MaxFlapsBeforeLock — after N flaps within FlapWindow, the engine
    // refuses all transitions until an override fires. Set to 0 to disable.
    MaxFlapsBeforeLock int
}

// History is the sequence of (state, timestamp) pairs the engine consults.
// Consumers pass a slice of recent transitions; the engine reads it but
// does not persist anything. Persistence is the consumer's responsibility.
type History struct {
    Entries []Entry
}

type Entry struct {
    State     string
    EnteredAt time.Time
}

// LatestState returns the most recent state, or "" if history is empty.
func (h *History) LatestState() string {
    if len(h.Entries) == 0 {
        return ""
    }
    return h.Entries[len(h.Entries)-1].State
}

// LatestEnteredAt returns when the latest state was entered.
func (h *History) LatestEnteredAt() time.Time {
    if len(h.Entries) == 0 {
        return time.Time{}
    }
    return h.Entries[len(h.Entries)-1].EnteredAt
}

// CountFlapsInWindow returns how many distinct state transitions occurred
// within the given window before now.
func (h *History) CountFlapsInWindow(now time.Time, window time.Duration) int {
    cutoff := now.Add(-window)
    var flaps int
    for i := 1; i < len(h.Entries); i++ {
        if h.Entries[i].EnteredAt.Before(cutoff) {
            continue
        }
        if h.Entries[i].State != h.Entries[i-1].State {
            flaps++
        }
    }
    return flaps
}
```

- [ ] **Step 2: Implement the engine**

```go
// engine.go
package stability

import (
    "fmt"
    "time"
)

// Engine evaluates whether a proposed state transition should be accepted,
// damped, or overridden. It is purely functional — no internal state, no
// persistence. Consumers pass a History on every call.
type Engine struct {
    policy Policy
}

// NewEngine constructs an engine with the given policy.
func NewEngine(policy Policy) *Engine {
    return &Engine{policy: policy}
}

// Evaluate returns a Decision for the proposed transition. `override` is
// true when the consumer has detected a clinical event that should bypass
// dwell (e.g. medication change, hospitalization, lab step change).
func (e *Engine) Evaluate(
    history History,
    proposedState string,
    now time.Time,
    override bool,
) Result {
    current := history.LatestState()

    // No history -> accept (first classification).
    if current == "" {
        return Result{Decision: DecisionAccept, Reason: "no prior state"}
    }

    // No state change -> trivially accept (idempotent).
    if current == proposedState {
        return Result{Decision: DecisionAccept, Reason: "no transition"}
    }

    // Override events bypass dwell and flap checks.
    if override {
        return Result{Decision: DecisionOverride, Reason: "override event bypasses dwell"}
    }

    // Dwell check.
    enteredAt := history.LatestEnteredAt()
    elapsed := now.Sub(enteredAt)
    if elapsed < e.policy.MinDwell {
        return Result{
            Decision: DecisionDamp,
            Reason: fmt.Sprintf("dwell not met: %v elapsed of %v required",
                elapsed.Round(time.Hour), e.policy.MinDwell),
        }
    }

    // Flap-lock check.
    if e.policy.MaxFlapsBeforeLock > 0 {
        flaps := history.CountFlapsInWindow(now, e.policy.FlapWindow)
        if flaps >= e.policy.MaxFlapsBeforeLock {
            return Result{
                Decision: DecisionDamp,
                Reason: fmt.Sprintf("flap-locked: %d flaps in last %v (max %d)",
                    flaps, e.policy.FlapWindow, e.policy.MaxFlapsBeforeLock),
            }
        }
    }

    return Result{Decision: DecisionAccept, Reason: "transition accepted"}
}
```

- [ ] **Step 3: Write 8-10 unit tests covering**

- First classification (no history) → ACCEPT
- Same state proposed → ACCEPT
- Different state within dwell window → DAMP
- Different state past dwell window → ACCEPT
- Override bypasses dwell → OVERRIDE
- Override bypasses flap-lock → OVERRIDE
- Flap-lock after MaxFlapsBeforeLock → DAMP
- Flap detection respects FlapWindow boundary

- [ ] **Step 4: Run tests; commit P2.1.**

## Task P2.2: BP Context Stability Integration

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_repository.go` (add `FetchHistory` time-bounded variant)
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator_test.go`

- [ ] **Step 1: Add `FetchHistorySince` to repository**

The orchestrator needs to build a `stability.History` from snapshot rows. The existing `FetchHistory(patientID, limit)` works but doesn't filter by time. Add:

```go
func (r *BPContextRepository) FetchHistorySince(patientID string, since time.Time) ([]models.BPContextHistory, error) {
    var snapshots []models.BPContextHistory
    err := r.db.
        Where("patient_id = ? AND snapshot_date >= ?", patientID, since).
        Order("snapshot_date ASC").
        Find(&snapshots).Error
    return snapshots, err
}
```

- [ ] **Step 2: Wire stability engine into the orchestrator**

In `bp_context_orchestrator.go`:

1. Add a `stability *stability.Engine` field to `BPContextOrchestrator`.
2. Add it to `NewBPContextOrchestrator` as the 8th parameter.
3. In `Classify`, after the classifier produces `result.Phenotype` but BEFORE `SaveSnapshot`, build the history and evaluate stability:

```go
// Stability check (Phase 4 P2): fetch recent history, build the
// stability.History view, and evaluate the proposed transition.
historyRows, _ := o.repo.FetchHistorySince(patientID, time.Now().AddDate(0, 0, -60))
history := buildStabilityHistory(historyRows)
override := detectOverrideEvent(profile) // medication change, hospitalization, etc.
decision := o.stability.Evaluate(history, string(result.Phenotype), time.Now().UTC(), override)

if decision.Decision == stability.DecisionDamp {
    o.log.Info("BP phenotype transition damped",
        zap.String("patient_id", patientID),
        zap.String("proposed", string(result.Phenotype)),
        zap.String("current", history.LatestState()),
        zap.String("reason", decision.Reason))
    // Use the current state instead of the proposed one for this snapshot.
    result.Phenotype = models.BPContextPhenotype(history.LatestState())
    result.Confidence = "DAMPED" // signals that the result is held, not new
}
```

The two helpers:

```go
func buildStabilityHistory(rows []models.BPContextHistory) stability.History {
    entries := make([]stability.Entry, len(rows))
    for i, r := range rows {
        entries[i] = stability.Entry{
            State:     string(r.Phenotype),
            EnteredAt: r.SnapshotDate,
        }
    }
    return stability.History{Entries: entries}
}

func detectOverrideEvent(profile *clients.KB20PatientProfile) bool {
    // Phase 4 placeholder: medication change detection requires a new
    // KB-20 field that doesn't exist yet. For now, return false — all
    // transitions go through dwell. Phase 5 wires this to a real signal.
    return false
}
```

- [ ] **Step 3: Wire policy in main.go**

In `main.go`, instantiate the engine with sensible defaults:

```go
bpStabilityPolicy := stability.Policy{
    MinDwell:           14 * 24 * time.Hour, // 2 weeks
    FlapWindow:         30 * 24 * time.Hour, // 1 month
    MaxFlapsBeforeLock: 3,
}
bpStabilityEngine := stability.NewEngine(bpStabilityPolicy)
```

Pass it as the 8th argument to `NewBPContextOrchestrator`.

- [ ] **Step 4: Add 4 integration tests**

- New patient (no history) → first classification accepted, snapshot saved
- Stable patient with phenotype unchanged → accepted, no event
- Within-dwell flapping (day 1: SH, day 2: WCH) → day 2 damped, phenotype stays SH, no `BP_PHENOTYPE_CHANGED` event
- Past-dwell genuine transition (day 1: SH, day 30: SN) → accepted, event fires

- [ ] **Step 5: Run tests; commit P2.**

---

# Sub-project P8: Confidence Tier Mapping

**Priority:** Medium. Cross-system consistency with KB-23's confidence tier UX.
**Effort:** ~3 tasks, ~5 hours.
**Depends on:** Phase 1 (BP context cards).

## Task P8.1: Add `confidenceStringToTier` helper

**Files:**
- Modify: `kb-23-decision-cards/internal/services/masked_htn_cards.go`
- Modify: `kb-23-decision-cards/internal/services/masked_htn_cards_test.go`

- [ ] **Step 1: Add the helper**

```go
// confidenceStringToTier maps the BP context classifier's string-based
// confidence ("HIGH"/"MODERATE"/"LOW"/"DAMPED") to KB-23's ConfidenceTier
// enum. Used only by masked HTN cards — existing HTN safety templates
// retain bypasses_confidence_gate=true and are unchanged.
func confidenceStringToTier(confidence string) models.ConfidenceTier {
    switch confidence {
    case "HIGH":
        return models.TierFirm
    case "MODERATE":
        return models.TierProbable
    case "LOW":
        return models.TierPossible
    case "DAMPED":
        return models.TierUncertain
    default:
        return models.TierUncertain
    }
}
```

- [ ] **Step 2: Apply tier to cards**

Modify each `MaskedHTNCard` builder to include a `ConfidenceTier` field set via the helper. Update the `MaskedHTNCard` struct to have a `ConfidenceTier models.ConfidenceTier` field.

- [ ] **Step 3: Add 4 mapping tests**

- HIGH → TierFirm
- MODERATE → TierProbable
- LOW → TierPossible
- DAMPED → TierUncertain
- Unknown → TierUncertain

- [ ] **Step 4: Commit P8.**

---

# Sub-project P7: Card YAML Fragment Templates

**Priority:** Medium. Clinical author agency without code deploys.
**Effort:** ~5 tasks, ~1 day.
**Depends on:** Phase 1+2+3 (existing cards), KB-23 fragment loader.

P7 migrates the 8 hardcoded card rationales in `masked_htn_cards.go` to YAML templates following the existing `templates/htn_safety/` pattern. Existing HTN safety templates are NOT modified.

## Task P7.1: Create 8 YAML templates

**Files:**
- Create: `kb-23-decision-cards/templates/bp_context/{8 files}.yaml`

For each of the 8 card types, create a YAML file mirroring the structure of `templates/htn_safety/bp_above_target.yaml`:

```yaml
# templates/bp_context/masked_hypertension.yaml
template_id: "dc-masked-htn-v1"
node_id: "KB26_BP_CONTEXT"
differential_id: "MASKED_HYPERTENSION"
version: "1.0.0"
clinical_reviewer: "dr.bp-context.lead"
mcu_gate_default: "MODIFY"
card_source: "KB26_BP_CONTEXT"
card_type: "MASKED_HYPERTENSION"
trigger_event: "BP_CONTEXT_CLASSIFIED"
trigger_condition: "phenotype == MASKED_HTN"
sla_hours: 24
priority: "HIGH"
confidence_tier: "GUIDELINE_CONCORDANT"
confidence_thresholds:
  firm_posterior: 0.0
  firm_medication_change: 0.0
  probable_posterior: 0.0
  possible_posterior: 0.0
recommendations:
  - rec_type: "MEDICATION_REVIEW"
    urgency: "URGENT"
    target: "clinician"
    action_text_en: "Do not rely on clinic BP alone — initiate or intensify antihypertensive therapy."
    action_text_hi: "केवल क्लिनिक रक्तचाप पर भरोसा न करें — एंटीहाइपरटेन्सिव थेरेपी शुरू करें या तीव्र करें।"
    rationale_en: "Masked hypertension carries higher CV risk than sustained hypertension because treatment is deferred."
    guideline_ref: "ESH 2023; AHA/ACC 2023"
    bypasses_confidence_gate: true
    sort_order: 1
fragments:
  - fragment_type: "CLINICIAN"
    text_en: "Clinic BP {{clinic_sbp}}/{{clinic_dbp}} mmHg (normal) but home mean {{home_sbp}} mmHg (elevated). Home BP exceeds clinic by {{gap_mmhg}} mmHg."
    text_hi: "क्लिनिक BP {{clinic_sbp}}/{{clinic_dbp}} mmHg (सामान्य) लेकिन घर का औसत {{home_sbp}} mmHg (बढ़ा हुआ)।"
  - fragment_type: "PATIENT"
    text_en: "Your in-clinic blood pressure looks normal, but the readings you've taken at home are higher. This pattern means treatment may be needed."
    text_hi: "क्लिनिक में आपका रक्तचाप सामान्य दिखता है, लेकिन घर पर लिए गए रीडिंग अधिक हैं।"
```

Repeat for the other 7 card types: `masked_uncontrolled.yaml`, `white_coat_hypertension.yaml`, `white_coat_uncontrolled.yaml`, `masked_htn_morning_surge_compound.yaml`, `sustained_htn_morning_surge.yaml`, `selection_bias_warning.yaml`, `medication_timing.yaml`.

## Task P7.2: Modify masked_htn_cards.go to load from templates

**Files:**
- Modify: `kb-23-decision-cards/internal/services/masked_htn_cards.go`

Replace the hardcoded rationale strings with calls to `templateLoader.GetByTemplate(templateID)` and `fragmentLoader.GetPatientText(fragmentID)`. The card builder needs access to both loaders, so add them as struct fields on `MaskedHTNCardBuilder` (which may need to be created if `EvaluateMaskedHTNCards` is currently a free function).

This is a non-trivial refactor — the existing `EvaluateMaskedHTNCards` is a pure function. P7.2 introduces a `MaskedHTNCardBuilder` struct that wraps it.

## Task P7.3-7.5: Tests, integration, commit.

(Detailed test specs omitted for brevity — write tests verifying the template-driven cards produce the same rationale text as the hardcoded version, then remove the hardcoded version.)

---

# Sub-project P9: Composite Card Aggregation Wiring

**Priority:** Low. UX polish — multiple BP cards compose into one.
**Effort:** ~2 tasks, ~3 hours.
**Depends on:** Phase 1 (BP context cards), KB-23 `CompositeCardService`.

## Task P9.1: Trigger Synthesize after each batch classification

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/services/bp_context_orchestrator.go` (add KB-23 composite trigger client call)
- Modify: `kb-23-decision-cards/internal/api/server.go` (expose composite trigger endpoint)

The simplest approach: KB-26 emits a `BP_CONTEXT_CLASSIFIED` event after each successful classification, and KB-23 triggers `CompositeCardService.Synthesize` for that patient as a side effect. Or directly: add a "trigger composite synthesis" endpoint to KB-23 and have KB-26 call it from the orchestrator's `emitPhenotypeEvents` helper.

## Task P9.2: Verify composite cards aggregate correctly

Test that a patient with masked HTN + medication timing + selection bias firing all 3 cards in a batch run produces one composite card with the most-restrictive gate.

---

# Sub-project P5: Operational Readiness Review (Documentation)

**Priority:** Medium. Prerequisite for actual production deployment.
**Effort:** 1 task, ~half day.
**Depends on:** Phase 1+2+3 + Phase 4 P1.

## Task P5.1: Write Masked HTN Operations Runbook

**Files:**
- Create: `docs/runbooks/masked-htn-operations.md`

The runbook is a living document covering:

### Section 1: Service overview
- What the masked HTN feature does in production
- Which services are involved (KB-26, KB-23, KB-20, KB-21, KB-19)
- Data flow diagram

### Section 2: Alerting thresholds
For each of the 8 Prometheus metrics from Phase 2+3, document:
- Healthy range
- Warning threshold + on-call escalation
- Critical threshold + paging escalation

Example:
```
kb26_bp_batch_duration_seconds (histogram)
  Healthy: p95 < 60s
  Warning: p95 60-300s — investigate KB-20/KB-21 latency
  Critical: p95 > 300s OR no batch run in last 26h — page on-call
```

### Section 3: Canary deployment plan
- Phase 1: Deploy with `BP_BATCH_ENABLED=false`, validate HTTP endpoint via manual curl
- Phase 2: Enable scheduler with `BP_BATCH_PERCENT=1` env var (new — added in this task)
- Phase 3: Ramp to 10%, 25%, 100% over 2 weeks
- Rollback: `BP_BATCH_ENABLED=false` immediately disables scheduler, HTTP endpoint stays available

### Section 4: Common failures and runbook entries
- "Batch hasn't run in 26 hours" → check scheduler goroutine, check KB-20 reachability
- "BP_PHENOTYPE_CHANGED events not reaching downstream" → check KB-19 publisher errors metric, check KB-19 health
- "Patient classification stuck on stale phenotype" → check stability engine logs for damping, manually trigger via HTTP endpoint
- "Synthetic readings fallback firing for all patients" → check KB-20 BP reading endpoint, may indicate P3.4 fallback path is hot

### Section 5: Manual operations
- How to manually trigger a classification: `curl -XPOST http://kb26:8137/api/v1/kb26/bp-context/${PATIENT_ID}`
- How to query a patient's BP context history: SQL query against `bp_context_history`
- How to disable the scheduler in an emergency

### Section 6: Glossary
- Phenotype names (SH, WCH, MH, MUCH, etc.)
- Selection bias terminology
- Cross-domain amplification rules

- [ ] **Step 1: Write the runbook**
- [ ] **Step 2: Commit P5.**

---

# Tasks Deferred from Phase 4 to Phase 5+

The following items from the original Phase 4 priority list are deferred to a future phase for the reasons documented:

| # | Item | Reason for deferral |
|---|---|---|
| **P6** | Quarterly aggregator scheduling | Belongs to a different feature (PREVENT/MRI quarterly rollup), not masked HTN. The `BatchScheduler` infrastructure built in Phase 3 makes this a 2-task addition for the quarterly aggregator owner — but it's not masked-HTN-scoped work. |
| **P10** | golang-migrate adoption (KB-26 platform) | Platform-level investment touching all KB-26 schema. Should be a dedicated migration project with its own brainstorm + plan + canary, not bundled with masked HTN. |
| **Stability engine reuse** | The Phase 4 stability engine in `pkg/stability/` is built in a shared package, but Phase 4 only wires it to BP context. Future systems (engagement classification, phenotype clustering when it exists) can adopt it independently. |

---

## Plan Summary

| Sub-project | Tasks | New Tests | Outcome |
|------|------|-----------|---------|
| **P1** Graceful shutdown + Drain | 2 | 2 | Data integrity in K8s deploys |
| **P3** Per-reading BP storage wiring | 5 | 12+ | Real readings replace synthetic — restores medication timing hypothesis |
| **P2** Stability engine + integration | 6 | 12+ | Phenotype flapping eliminated for actively-monitored patients |
| **P8** Confidence tier mapping | 3 | 4 | Cross-system UX consistency |
| **P7** Card YAML fragments | 5 | ~6 | Clinical author agency, no code deploys for text changes |
| **P9** Composite card wiring | 2 | 2 | Multiple BP cards compose into one |
| **P5** Operations runbook | 1 | 0 (doc) | Production readiness checklist |
| **Total** | **24** | **38+** | All P0-P9 closed |

## What Phase 4 Delivers (as a sum of all sub-projects)

After every sub-project ships:
- KB-26 honors SIGTERM with proper `http.Server.Shutdown` and batch drain (no more 500ms hack)
- The classifier reads real per-reading BP data from KB-20 LabEntry rows, not synthetic aggregates
- The medication timing hypothesis fires correctly (it's been dead since Phase 1)
- Phenotype flapping is eliminated by the stability engine (2-week dwell, 3-flap lock)
- Override events bypass dwell (placeholder hook for Phase 5 medication change detection)
- BP context cards expose `ConfidenceTier` consistent with the rest of KB-23
- Card text is editable in YAML by clinical authors without a code deploy
- Multiple BP cards aggregate into a single composite card via the existing `CompositeCardService`
- Operations runbook exists for on-call engineers

## What Phase 4 Does NOT Deliver

Documented and deferred:
- **Override event detection** — the stability engine has the hook (`override bool`) but no real signal source. P2 stubs `detectOverrideEvent` to always return `false`. Phase 5 wires it to a medication change signal from KB-20.
- **Quarterly aggregator scheduling** (P6) — not masked HTN scope
- **golang-migrate adoption** (P10) — not masked HTN scope
- **Stability engine reuse by other consumers** — engine exists in shared package, but Phase 4 only wires the BP context consumer

## Execution Order Recommendation

The sub-projects can be implemented in any order, but if a planner can only ship some of them, **ship in this order:**

1. **P1** (graceful shutdown) — production safety
2. **P3** (per-reading wiring) — biggest clinical correctness win
3. **P2** (stability engine) — eliminates phenotype flapping
4. **P5** (runbook) — required before any actual production deploy
5. **P8** (confidence tier) — cross-system polish
6. **P7** (YAML fragments) — clinical author agency
7. **P9** (composite cards) — UX polish

P1, P3, P2, and P5 together represent "production-ready masked HTN." Everything else is improvement.
