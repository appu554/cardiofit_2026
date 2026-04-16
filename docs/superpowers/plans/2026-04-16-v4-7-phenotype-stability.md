# V4-7 Phenotype Clustering Temporal Stability Check — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add temporal stability to the phenotype clustering pipeline so cluster assignments are clinically meaningful over time — preventing therapy pathway whiplash, IOR evidence contamination, and false transitions masking real ones.

**Architecture:** The stability engine sits between the Python clustering pipeline (which produces raw assignments) and KB-20 (which stores stable assignments). Only stable assignments are written to KB-20 and consumed by downstream systems. The engine implements a dwell policy (minimum 4 weeks in a cluster), flap detection (oscillation dampening), override event detection (clinical events that bypass dwell), and transition classification (genuine vs flap vs uncertain). Cross-domain context from MHRI domain decomposition, engagement status, and data modality changes informs each decision.

**Tech Stack:** Go 1.21 (Gin, GORM) for the stability engine in KB-20. PostgreSQL 15 for cluster history + transition records. YAML market configs for stability policies. KB-23 for transition card generation.

---

## Scope Check

This spec covers one cohesive subsystem: the phenotype stability engine. It touches three services (KB-20 for the engine, KB-23 for transition cards, and the Python clustering pipeline for confidence extraction) but they form a single data flow. One plan is appropriate.

## Existing Infrastructure

| What exists | Where | Relevance |
|---|---|---|
| `PatientProfile.PhenotypeCluster` + `PhenotypeConfidence` + `PhenotypeClusterOrigin` | KB-20 `internal/models/patient_profile.go:61-63` | Fields already on the profile — stability engine writes to these |
| `pkg/stability` engine (dwell/flap/override) | KB-26 + KB-23 (Phase 7 P7-D copy) | Generic string-state stability — V4-7 needs phenotype-specific logic with membership probability, separability, conservatism ranking |
| Market config structure | `market-configs/shared/` + `india/` + `australia/` | V4-7 adds `phenotype_stability.yaml` here |
| Seasonal calendar | `market-configs/india/seasonal_calendar.yaml` | Cross-referenced for seasonal dwell extension |
| `SafetyEvent` audit trail | KB-20 `internal/models/safety_event.go` | Override events (hospitalisation) can be sourced from here |
| Event types (CKM transition, medication change) | KB-20 `internal/models/events.go` | Override event detection sources |

## File Inventory

### KB-20 (Patient Profile) — Stability Engine
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/models/cluster_history.go` | GORM models: `ClusterAssignmentRecord`, `ClusterTransitionRecord`, `PatientClusterState`, `StabilityDecision`, `OverrideEvent` |
| Create | `internal/services/phenotype_stability.go` | `StabilityEngine.Evaluate(input) → StabilityDecision` — dwell policy, flap detection, override events, noise handling, data modality |
| Create | `internal/services/phenotype_stability_test.go` | 12+ tests from the spec: first assignment, same cluster, dwell hold, dwell pass, override events, flap detection, noise, low confidence, CGM modality, cross-domain |
| Create | `internal/services/cluster_transition_classifier.go` | `ClassifyTransition(current, proposed, overrides, history) → TransitionType` — GENUINE / PROBABLE_FLAP / UNCERTAIN / DWELL_HELD |
| Create | `internal/services/cluster_transition_classifier_test.go` | 6+ tests: directional genuine, oscillation flap, uncertain (no history), engagement-coincident |
| Modify | `internal/api/v4_state_handlers.go` | Intercept PATCH with stability engine before writing to PatientProfile |
| Modify | `internal/database/connection.go` | AutoMigrate `ClusterAssignmentRecord` + `ClusterTransitionRecord` |

### KB-23 (Decision Cards) — Transition Cards
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/services/phenotype_transition_cards.go` | Generate cards on genuine transitions, flap flags, and engagement-coincident transitions |
| Create | `internal/services/phenotype_transition_cards_test.go` | 4+ tests: genuine transition card, flap warning card, engagement-coincident card, stable-good suppresses inertia |
| Create | `templates/phenotype/genuine_transition.yaml` | Card for confirmed phenotype change with therapy context |
| Create | `templates/phenotype/flap_warning.yaml` | Card for unstable phenotype with root domain identified |

### Market Configs
| Action | File | Responsibility |
|---|---|---|
| Create | `market-configs/shared/phenotype_stability.yaml` | Dwell policy, flap detection, confidence thresholds, override events, conservatism ranking |

**Total: 13 files (11 create, 2 modify)**

---

### Task 1: Create cluster history models + stability config YAML

**Files:**
- Create: `kb-20-patient-profile/internal/models/cluster_history.go`
- Create: `market-configs/shared/phenotype_stability.yaml`

- [ ] **Step 1:** Create `cluster_history.go` with five types from the spec: `ClusterAssignmentRecord` (raw assignment per batch run with membership probability + separability ratio + noise flag), `ClusterTransitionRecord` (logged when stable cluster changes, with cross-domain context fields), `PatientClusterState` (current stability state with dwell days + flap count + pending raw cluster), `StabilityDecision` (output of stability engine evaluation), `OverrideEvent` (clinical event that bypasses dwell). All types use the exact field names + GORM tags from the spec lines 277-365.

- [ ] **Step 2:** Create `phenotype_stability.yaml` with the complete configuration from the spec lines 372-451: dwell policy (4 weeks min, 8 weeks extended for flapping patients), flap detection (90-day lookback, 2+ oscillations), confidence thresholds (0.7/0.4 membership probability), override events list (8 event types from MHRI crossing through CGM cessation), transition classification criteria, conservatism ranking, and data modality grace periods.

- [ ] **Step 3:** Verify the models compile: `go build ./...` in KB-20.

- [ ] **Step 4:** Commit: `feat(kb20): cluster history models + phenotype stability config (V4-7 Task 1)`

---

### Task 2: Build StabilityEngine with dwell + override + noise + flap logic

**Files:**
- Create: `kb-20-patient-profile/internal/services/phenotype_stability.go`
- Create: `kb-20-patient-profile/internal/services/phenotype_stability_test.go`

- [ ] **Step 1:** Write 12 failing tests from the spec (lines 463-800): `TestStability_FirstAssignment_Accepted`, `TestStability_SameCluster_NoChange`, `TestStability_DifferentCluster_WithinDwell_Held`, `TestStability_DifferentCluster_PastDwell_Accepted`, `TestStability_DifferentCluster_WithinDwell_OverrideEvent_Accepted`, `TestStability_CKMStageTransition_OverridesDwell`, `TestStability_FlapDetected_HeldAtConservative`, `TestStability_FlapDetected_OverrideStillWorks`, `TestStability_NoiseLabel_HoldPrevious`, `TestStability_LowConfidence_Flagged`, `TestStability_CGMStarted_GracePeriod`, `TestStability_TransitionWithDomainDriver`.

- [ ] **Step 2:** Run tests to verify they all fail (functions not defined).

- [ ] **Step 3:** Implement `StabilityEngine.Evaluate(input StabilityInput) StabilityDecision` with the decision cascade from the spec (lines 857-998): first assignment → same cluster → noise → data modality grace → flap check → dwell check → override check → accept. Each case returns a `StabilityDecision` with the appropriate `Decision` (ACCEPT / HOLD_DWELL / HOLD_FLAP / OVERRIDE_EVENT), `StableClusterLabel`, `TransitionType`, and `Reason`.

- [ ] **Step 4:** Run tests — all 12 should pass.

- [ ] **Step 5:** Commit: `feat(kb20): phenotype stability engine (V4-7 Task 2)`

---

### Task 3: Build ClusterTransitionClassifier

**Files:**
- Create: `kb-20-patient-profile/internal/services/cluster_transition_classifier.go`
- Create: `kb-20-patient-profile/internal/services/cluster_transition_classifier_test.go`

- [ ] **Step 1:** Write 6 failing tests: `TestClassify_DirectionalWithOverride_Genuine` (single directional move + clinical event = GENUINE), `TestClassify_OscillationNoEvent_ProbableFlap` (back-and-forth between same pair without clinical event = PROBABLE_FLAP), `TestClassify_InsufficientHistory_Uncertain`, `TestClassify_EngagementCoincident_DataQuality` (transition coincides with engagement collapse = UNCERTAIN with engagement annotation), `TestClassify_SeasonalWindow_Uncertain` (transition during known seasonal window), `TestClassify_MedicationOverride_Genuine` (transition after medication class addition = GENUINE regardless of oscillation history).

- [ ] **Step 2:** Implement `ClassifyTransition(fromCluster, toCluster string, overrideEvents []OverrideEvent, recentHistory []ClusterTransitionRecord, seasonalActive bool, engagementCollapse bool) string` returning GENUINE_TRANSITION / PROBABLE_FLAP / UNCERTAIN / DWELL_HELD.

- [ ] **Step 3:** Run tests — all 6 should pass.

- [ ] **Step 4:** Commit: `feat(kb20): cluster transition classifier (V4-7 Task 3)`

---

### Task 4: Wire stability engine into KB-20's v4-state PATCH handler

**Files:**
- Modify: `kb-20-patient-profile/internal/api/v4_state_handlers.go`
- Modify: `kb-20-patient-profile/internal/database/connection.go`

- [ ] **Step 1:** Add `ClusterAssignmentRecord` + `ClusterTransitionRecord` to the AutoMigrate list in `connection.go`.

- [ ] **Step 2:** In the v4-state PATCH handler, intercept incoming `phenotype_cluster` updates: instead of writing the raw assignment directly to `PatientProfile.PhenotypeCluster`, route it through `StabilityEngine.Evaluate`. Write the `StableClusterLabel` to the profile. Persist the `ClusterAssignmentRecord` (raw assignment) and, if the stable cluster changed, a `ClusterTransitionRecord`.

- [ ] **Step 3:** Write 2 handler-level tests: PATCH with stability hold (raw differs from stable, within dwell → profile keeps old cluster), PATCH with stability accept (past dwell → profile gets new cluster).

- [ ] **Step 4:** Build + full KB-20 test sweep.

- [ ] **Step 5:** Commit: `feat(kb20): stability-aware v4-state PATCH handler (V4-7 Task 4)`

---

### Task 5: Build phenotype transition cards in KB-23

**Files:**
- Create: `kb-23-decision-cards/internal/services/phenotype_transition_cards.go`
- Create: `kb-23-decision-cards/internal/services/phenotype_transition_cards_test.go`
- Create: `kb-23-decision-cards/templates/phenotype/genuine_transition.yaml`
- Create: `kb-23-decision-cards/templates/phenotype/flap_warning.yaml`

- [ ] **Step 1:** Create `genuine_transition.yaml` template: `dc-phenotype-transition-v1`, `CROSS_NODE`, `PHENOTYPE_TRANSITION`. CLINICIAN fragment: "Patient phenotype changed from {{.PreviousCluster}} to {{.NewCluster}} — {{.Interpretation}}. Dominant domain driver: {{.DomainDriver}}." MCU gate: SAFE (advisory).

- [ ] **Step 2:** Create `flap_warning.yaml` template: `dc-phenotype-flap-warning-v1`, `CROSS_NODE`, `PHENOTYPE_FLAP_WARNING`. CLINICIAN fragment: "Patient's phenotype classification is unstable — oscillating between {{.FlapPair}}. Root cause: {{.DomainDriver}} domain instability. Hold current pathway pending stabilization."

- [ ] **Step 3:** Write 4 tests: genuine transition produces card, flap warning produces card, stable-in-good-cluster suppresses inertia annotation, template loads from disk.

- [ ] **Step 4:** Implement `EvaluatePhenotypeTransition(decision StabilityDecision, patientMeds []string) []PhenotypeTransitionCard` that generates the appropriate card(s) based on the stability decision.

- [ ] **Step 5:** Build + full KB-23 test sweep.

- [ ] **Step 6:** Commit: `feat(kb23): phenotype transition cards (V4-7 Task 5)`

---

### Task 6: Full integration test + commit

- [ ] **Step 1:** Full test sweep across KB-20, KB-23, KB-26.

- [ ] **Step 2:** Verify all market-config YAML files parse correctly (existing tests + new phenotype_stability.yaml).

- [ ] **Step 3:** Final commit: `feat: complete V4-7 phenotype clustering temporal stability check`

- [ ] **Step 4:** Push to origin.

---

## Verification Questions

1. Does a first-assignment patient get ACCEPT with INITIAL transition type? (yes / test)
2. Does a patient within the 4-week dwell window get HOLD_DWELL? (yes / test)
3. Does a CKM_STAGE_TRANSITION override the dwell policy? (yes / test)
4. Does a flapping patient get held at the more conservative cluster? (yes / test)
5. Does a noise label (-1) get held at the previous cluster? (yes / test)
6. Does CGM initiation trigger a data modality grace period? (yes / test)
7. Does the v4-state PATCH handler write the stable cluster, not the raw cluster? (yes / test)
8. Does a genuine transition produce a PHENOTYPE_TRANSITION card in KB-23? (yes / test)
9. Does a flap produce a PHENOTYPE_FLAP_WARNING card? (yes / test)
10. Are all KB-20 + KB-23 test suites green? (yes / sweep)

---

## Effort Estimate

| Task | Scope | Expected |
|---|---|---|
| Task 1: Models + config | 2 files, ~300 LOC | 1-2 hours |
| Task 2: Stability engine | 2 files, ~400 LOC + 12 tests | 2-3 hours |
| Task 3: Transition classifier | 2 files, ~200 LOC + 6 tests | 1-2 hours |
| Task 4: v4-state handler wiring | 2 files modified, ~150 LOC | 1 hour |
| Task 5: KB-23 transition cards | 4 files, ~300 LOC + 4 tests | 1-2 hours |
| Task 6: Integration test | 0 new files | 30 min |
| **Total** | **~13 files, ~1350 LOC, ~24 tests** | **~7-10 hours** |
