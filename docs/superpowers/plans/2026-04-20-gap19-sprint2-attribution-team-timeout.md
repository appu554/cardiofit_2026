# Gap 19 Sprint 2: Attribution Engine + Team Comparison + Timeout Events

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Elevate Gap 19 from "loop closed at plumbing" to "loop closed at evidence." Sprint 1 (shipped) proves detections → actions happen; Sprint 2 proves actions → outcomes happen and feeds the signal back to clinicians to drive response-time improvement.

**Strategic importance:** The `GET /api/v1/metrics/pilot?window=90` endpoint is the evidence HCF's purchasing committee needs at month 11 for the $1.35M renewal. Today it can report "612 actions taken"; Sprint 2 makes it report "340 of those actions were followed by measurable clinical improvement attributable to the action." That attribution distinction is what converts activity evidence into outcome evidence.

**Architecture:**
- **Attribution engine**: a KB-23 service that, on T4 arrival, scores the candidate lifecycle against causal-plausibility criteria and records a confidence grade (HIGH / MODERATE / LOW / NONE). MVP is single-candidate ("most recent wins + confidence score"); true fractional multi-causal attribution waits for real pilot data.
- **Team comparison**: a `/metrics/clinician/:id/peer-comparison` endpoint that returns the clinician's median alongside their role-based peer cohort's median. Drives the self-regulating feedback loop documented in Bates 2022.
- **Timeout goroutine**: a background poller that detects lifecycles exceeding their tier's t2/t3 threshold without acknowledgment/action and emits explicit timeout events. Today a stuck detection is silently absent from metrics; explicit events enable Prometheus alerting.

**Tech stack:** Go 1.21, GORM, PostgreSQL (the SQL-backed aggregations will work through pilot scale; TimescaleDB remains deferred to Sprint 3 when data volume forces it). No new services; all three work items fold into KB-23.

---

## Existing infrastructure

| What exists | Where | Relevance |
|---|---|---|
| DetectionLifecycle with T0–T4 timestamps + CohortID | `kb-23-decision-cards/internal/models/detection_lifecycle.go` | Raw material for attribution scoring |
| `FindMostRecentActionedByPatient` + POST /tracking/resolve | `kb-23-decision-cards/internal/services/lifecycle_tracker.go`, `internal/api/tracking_handlers.go` | Sprint 1 crude attribution — replace with engine output |
| KB-26 `HandleNewReading` resolution branch firing T4 bridge | `kb-26-metabolic-digital-twin/internal/services/acute_event_handler.go` | Primary T4 signal source |
| `ResponseTrackingConfig` per-tier thresholds loaded from YAML | `kb-23-decision-cards/internal/services/response_tracking_config.go` | Timeout goroutine reads the same thresholds |
| `ComputeClinicianMetrics` with role field on clinician | `kb-23-decision-cards/internal/services/response_metrics.go` | Peer cohort = same role; needs role lookup or denormalized column |

## File inventory

### Sprint 2a — Attribution engine
| Action | File | Responsibility |
|---|---|---|
| Create | `kb-23-decision-cards/internal/models/attribution.go` | AttributionRecord struct, ConfidenceGrade enum |
| Create | `kb-23-decision-cards/internal/services/attribution_engine.go` | `ScoreCandidate` implementing 7-step causal plausibility |
| Create | `kb-23-decision-cards/internal/services/attribution_engine_test.go` | 8 tests covering each plausibility axis |
| Modify | `kb-23-decision-cards/internal/services/lifecycle_tracker.go` | RecordT4 invokes attribution engine; stores AttributionRecord |
| Modify | `kb-23-decision-cards/internal/api/tracking_handlers.go` | Resolve endpoint returns {lifecycle_id, confidence, rationale} |
| Modify | `kb-23-decision-cards/internal/database/connection.go` | AutoMigrate AttributionRecord |

### Sprint 2b — Peer-comparison endpoint
| Action | File | Responsibility |
|---|---|---|
| Create | `kb-23-decision-cards/internal/services/peer_comparison.go` | Compute {clinician_median, peer_median, delta, percentile} |
| Create | `kb-23-decision-cards/internal/services/peer_comparison_test.go` | 4 tests |
| Modify | `kb-23-decision-cards/internal/models/detection_lifecycle.go` | Add `AssignedClinicianRole` (denormalized column) |
| Modify | `kb-23-decision-cards/internal/services/lifecycle_tracker.go` | RecordT2 populates role from incoming clinicianID (or lookup) |
| Modify | `kb-23-decision-cards/internal/api/tracking_handlers.go` | GET /metrics/clinician/:id/peer-comparison |
| Modify | `kb-23-decision-cards/internal/api/routes.go` | Register new route |

### Sprint 2c — Timeout event goroutine
| Action | File | Responsibility |
|---|---|---|
| Create | `kb-23-decision-cards/internal/services/timeout_checker.go` | Background poller with configurable interval |
| Create | `kb-23-decision-cards/internal/services/timeout_checker_test.go` | 5 tests including scheduler boundaries |
| Modify | `kb-23-decision-cards/internal/services/lifecycle_tracker.go` | `MarkTimedOut(lc, stage, reason)` emits LifecycleTimedOut |
| Modify | `kb-23-decision-cards/internal/api/server.go` | Start timeout_checker goroutine on server boot |
| Modify | `kb-23-decision-cards/internal/config/config.go` | TIMEOUT_CHECK_INTERVAL_SECS env var |

### Market configs
| Action | File | Responsibility |
|---|---|---|
| Create | `backend/shared-infrastructure/market-configs/shared/attribution_parameters.yaml` | Per-detection-type attribution windows + weights |

**Total: ~18 files (10 create, 8 modify), ~17 tests. Effort: 3–5 focused days.**

---

## Sprint 2a: Outcome attribution engine

The MVP scope is deliberately narrow. True multi-causal attribution (fractional credit across overlapping actions) needs months of pilot data to tune; single-candidate attribution with confidence grading is useful on day one and doesn't need tuning.

### Causal plausibility scoring (7 axes)

Each axis scores 0–2. Total 0–14 → grade boundaries: `12+` = HIGH, `8–11` = MODERATE, `4–7` = LOW, `<4` = NONE.

| # | Axis | 0 | 1 | 2 |
|---|---|---|---|---|
| 1 | **Temporal window** | T4–T3 > 2× tier expected t4_outcome_hours | Within 2× | Within 1× expected |
| 2 | **Action↔outcome fit** | Action type unrelated to vital (e.g. appointment → eGFR) | Weakly related | Directly targets the vital (med review → BP) |
| 3 | **Deviation magnitude** | Outcome signal within measurement noise (<5%) | 5–15% improvement | >15% improvement |
| 4 | **No confounders** | Multiple parallel lifecycles for same patient within window | One other lifecycle | This is the only active lifecycle |
| 5 | **Detection specificity** | Detection fired for different vital than the one that resolved | Related vital family | Exact match |
| 6 | **Clinician engagement** | Acknowledged but no action taken | Action taken within t3_action_hours | Action taken within ¼ of t3_action_hours |
| 7 | **Cohort baseline** | Patient's cohort has higher spontaneous-resolution rate than 30% | 15–30% | <15% |

### Tasks

- [ ] **Step 1:** Create `attribution.go` with `AttributionRecord` struct: `{lifecycle_id, confidence_grade, score, axis_scores[7], rationale string, computed_at time}`. GORM with `unique(lifecycle_id)`.

- [ ] **Step 2:** Write 8 tests before implementation:
  1. `TestAttribution_HighConfidence_AllAxesFavorable` — score 14 → HIGH
  2. `TestAttribution_LowConfidence_TemporalTooLong` — axis 1 scores 0 → total ≤ 12 → expected grade
  3. `TestAttribution_NoConfidence_ActionUnrelated` — axis 2 = 0 capped score
  4. `TestAttribution_ParallelLifecycles_PenalizesConfounders` — axis 4 penalty applied
  5. `TestAttribution_MismatchedVital_FailsSpecificity` — axis 5 = 0
  6. `TestAttribution_LateAction_PenalizesEngagement` — axis 6 scoring
  7. `TestAttribution_HighBaselineCohort_PenalizesCredit` — axis 7 scoring
  8. `TestAttribution_Rationale_IsHumanReadable` — rationale string includes axis names + scores

- [ ] **Step 3:** Implement `attribution_engine.go`:
  ```go
  func (e *AttributionEngine) ScoreCandidate(
      lc *models.DetectionLifecycle,
      outcome OutcomeSignal,
      parallelLifecycles []models.DetectionLifecycle,
      cohortBaselineRate float64,
  ) (*models.AttributionRecord, error)
  ```

- [ ] **Step 4:** Modify `RecordT4` to call `ScoreCandidate` before persisting the lifecycle. Store the resulting `AttributionRecord`. Downgrade the lifecycle to LOW confidence when the engine grades `< MODERATE` so pilot metrics can surface "resolved but not attributable."

- [ ] **Step 5:** Extend `ComputePilotMetrics` to expose `outcomes_with_high_confidence`, `outcomes_with_moderate_confidence`, `outcomes_low_or_none`. This is what HCF's review committee actually sees.

- [ ] **Step 6:** Update the resolve endpoint response to include `{confidence_grade, score, rationale}` so callers (KB-26) can log the grading.

- [ ] **Step 7:** Commit: `feat(gap19-sprint2): outcome attribution engine — 7-axis plausibility scoring`

### Sprint 2a gotcha — don't retro-attribute

Attribution must be computed **at T4 time using data available then**, not retroactively. Otherwise a later identical lifecycle could retroactively steal credit. The `AttributionRecord.computed_at` is the T4 timestamp; score frozen at that point even if the underlying lifecycle is later amended.

---

## Sprint 2b: Peer-comparison endpoint

Bates 2022 documents that clinicians shown their personal response time vs their peer median reduce their response time 30–40% within 8 weeks. This is the single highest-ROI behavioral intervention we can add to Gap 19.

### Design

- **Peer group** = clinicians with the same `AssignedClinicianRole` and the same `CohortID`. For a cardiologist in HCF Catalyst, the peer group is "other HCF cardiologists."
- **Denormalize role** onto `DetectionLifecycle`: at `RecordT2` time, look up the clinician's role (from a new `ClinicianDirectory` interface — stub for MVP) and stamp it.
- Response payload:
  ```json
  {
    "clinician_id": "dr-jones",
    "role": "CARDIOLOGIST",
    "cohort_id": "hcf_catalyst_chf",
    "clinician_median_ack_ms": 2_700_000,
    "peer_median_ack_ms": 1_800_000,
    "delta_ms": 900_000,
    "percentile": 72,
    "peer_n": 8
  }
  ```

### Tasks

- [ ] **Step 1:** Add `AssignedClinicianRole` column to `DetectionLifecycle` (size:30, index). AutoMigrate.

- [ ] **Step 2:** Define a `ClinicianDirectory` interface in services:
  ```go
  type ClinicianDirectory interface {
      GetRole(clinicianID string) (string, bool)
  }
  ```
  Nil-safe — when unset, role stays empty string and peer-comparison endpoint returns 503 with a clear "directory not configured" message.

- [ ] **Step 3:** In `RecordT2`, when the directory is set, populate `AssignedClinicianRole`. First-write-wins applies here too (role stays whatever was stamped on earliest T2).

- [ ] **Step 4:** Implement `peer_comparison.go` with 4 tests:
  1. `TestPeerComparison_WithinRoleAndCohort` — peer group correctly scoped
  2. `TestPeerComparison_EmptyPeerGroup` — returns `peer_n: 0, peer_median_ack_ms: null`; delta omitted
  3. `TestPeerComparison_Percentile_CorrectForSortedPeers`
  4. `TestPeerComparison_ExcludesSubjectFromPeerSet` — subject's own data doesn't skew their own comparison

- [ ] **Step 5:** Add `GET /api/v1/metrics/clinician/:id/peer-comparison?window=30&cohort=<id|all>`.

- [ ] **Step 6:** Commit: `feat(gap19-sprint2): per-clinician peer-comparison endpoint (Bates self-regulation)`

### Sprint 2b gotcha — privacy

Do **not** expose individual peers' IDs in the response. Return aggregate-only (`peer_median_ack_ms`, `peer_n`). Clinicians must not be able to infer "Dr Smith is slow" from the endpoint.

---

## Sprint 2c: Timeout event goroutine

### Design

- A `TimeoutChecker` goroutine runs on a ticker (default 30s, configurable via `TIMEOUT_CHECK_INTERVAL_SECS`).
- On each tick, query `DetectionLifecycle` where `(state=NOTIFIED AND detected_at < now - tier_t2_threshold)` OR `(state=ACKNOWLEDGED AND acknowledged_at < now - tier_t3_threshold)`.
- For each match, call `tracker.MarkTimedOut(lc, stage, reason)` which sets `CurrentState=TIMED_OUT` and (optionally) publishes a Kafka event so Prometheus/alerting can fire.
- Idempotent: a lifecycle that's already `TIMED_OUT` is skipped.

### Tasks

- [ ] **Step 1:** Implement `MarkTimedOut(lc, stage, reason)`:
  ```go
  func (t *LifecycleTracker) MarkTimedOut(lc *models.DetectionLifecycle, stage, reason string)
  ```
  Sets `CurrentState = TIMED_OUT`, appends `reason` to `OutcomeDescription`, saves. Idempotent via the state check.

- [ ] **Step 2:** Write 5 tests:
  1. `TestTimeoutChecker_T2Expired_MarksTimedOut` — ack threshold elapsed without T2
  2. `TestTimeoutChecker_T3Expired_MarksTimedOut` — action threshold elapsed after T2
  3. `TestTimeoutChecker_AlreadyTimedOut_Skipped` — idempotency
  4. `TestTimeoutChecker_ResolvedNotTouched` — T4-closed lifecycles aren't re-timed-out
  5. `TestTimeoutChecker_TickerBoundary` — lifecycle at exactly threshold → NOT yet timed out (strict >); just past → timed out

- [ ] **Step 3:** Implement `timeout_checker.go` with a `Start(ctx)` + `Stop()` API so server shutdown is clean.

- [ ] **Step 4:** Start the checker from `server.InitServices()`. Goroutine runs for the server's lifetime. Log one INFO line per tick with "N lifecycles marked timed out" to make this visible in logs.

- [ ] **Step 5:** (Optional — defer if Kafka wiring is heavier than the rest of Sprint 2) Emit a `LifecycleTimedOut` Kafka event so ops alerting can page on spikes.

- [ ] **Step 6:** Commit: `feat(gap19-sprint2): timeout checker goroutine — explicit timeout events`

### Sprint 2c gotcha — interval × threshold

If the ticker interval is 30s and the SAFETY t2 threshold is 30 min, the latest a SAFETY timeout can be detected is 30 min + 30 s. That's fine. But for a 2-min t1_delivery_minutes threshold, a 30s ticker adds 25% variance. If we ever add per-stage thresholds < 5 minutes, the ticker interval needs to drop proportionally. Document this trade-off in the config comment.

---

## Sprint 2 integration test

- [ ] **Step 1:** Add `tests/integration/gap19_sprint2_full_lifecycle_test.go`:
  - Inject a synthetic SAFETY escalation → RecordT0–T3 → simulate no response for 90 min → expect `TIMED_OUT` state
  - Second scenario: RecordT0–T3 within thresholds → feed a resolving reading into the acute_event_handler bridge → expect T4 + AttributionRecord with confidence grade
  - Third scenario: peer-comparison endpoint with three synthetic clinicians in same role → expected percentile

- [ ] **Step 2:** Verify `/metrics/pilot` response now includes `outcomes_with_high_confidence`.

- [ ] **Step 3:** Final commit: `feat: complete Gap 19 Sprint 2 — attribution + peer comparison + timeout events`

- [ ] **Step 4:** Push.

---

## Verification questions

1. Does a resolved lifecycle get an AttributionRecord with a confidence grade? (yes / Sprint 2a)
2. Does the pilot metrics endpoint expose high/moderate/low/none counts? (yes / Sprint 2a Step 5)
3. Does retroactive attribution re-score older lifecycles? (no — frozen at T4 time / Sprint 2a gotcha)
4. Does peer-comparison exclude the subject clinician from their own peer set? (yes / Step 4 test 4)
5. Does the endpoint ever reveal individual peer IDs? (no / Sprint 2b gotcha)
6. Does the timeout goroutine skip already-timed-out lifecycles? (yes / idempotency test)
7. Does the integration test prove the full T0→T4 loop with attribution? (yes / integration test step 1)

## Deferred to Sprint 3

| Component | Reason |
|---|---|
| Fractional multi-causal attribution | Needs 3–6 months of real pilot data to tune weights |
| TimescaleDB migration | Current PostgreSQL aggregations work through pilot scale (~50K lifecycles) |
| Grafana dashboards | API contract is stable; analytics team can build dashboards off JSON |
| Per-patient cohort lookup (replacing DEFAULT_COHORT stamp) | Requires KB-20 patient profile to expose cohort membership first |
| Kafka event emission for timeout events | Only needed once ops/alerting moves from log-based to Prometheus-based |
| Real-channel delivery-time measurement (T1 accuracy) | Blocked on WhatsApp/FCM integration; today's T1 is "we tried to send", not "channel confirmed delivery" |

## Effort estimate

| Sprint 2 item | Scope | Expected |
|---|---|---|
| Sprint 2a: attribution engine (8 tests) | 3 create, 3 modify | 1.5–2 days |
| Sprint 2b: peer-comparison endpoint (4 tests) | 2 create, 3 modify | 0.5–1 day |
| Sprint 2c: timeout goroutine (5 tests) | 2 create, 3 modify | 1 day |
| Integration test + final commit | 1 file | 0.5 day |
| **Total** | **~18 files, ~17 tests** | **3–5 days** |

---

## What Sprint 2 does NOT change

- **T0–T4 semantic definitions**: T3 remains "action initiated," T4 remains "outcome observed" (both load-bearing per the note in `lifecycle_tracker.go`).
- **Sprint 1 endpoints**: `/metrics/pilot`, `/metrics/system`, `/metrics/clinician/:id` all keep their existing contracts; Sprint 2 adds fields, doesn't remove.
- **Cohort model**: Sprint 2 still uses the DEFAULT_COHORT stamp from Sprint 1; per-patient lookup is Sprint 3 work.
- **Worklist / escalation logic**: untouched — Sprint 2 is purely a measurement-layer upgrade.
