# Gap 18: Clinician Worklist — Implementation Plan (Core Engine)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a PAI-sorted, persona-filtered clinician worklist that shows "one most important thing per patient" with specific actions and resolution tracking — transforming the platform's clinical intelligence into a sorted daily triage list.

**Architecture:** The worklist engine lives in KB-23 (where cards + escalation already exist) and composes data from KB-20 (patient profiles), KB-23 (cards + escalations), and KB-26 (PAI scores). A WorklistAggregator joins these sources per-patient applying "one most important thing" prioritization. A SortTierEngine sorts by escalation tier → PAI → trajectory → attention gap. PersonaFilters scope the list per clinician role. A ResolutionHandler processes one-tap actions and updates upstream state. KB-14's existing worklist infrastructure gets the PAI-enriched items via its existing API.

**Tech Stack:** Go 1.21 (Gin, GORM) for KB-23 extensions. Existing KB-20/KB-23/KB-26 HTTP clients. YAML market configs for persona definitions. REST API for worklist consumption.

---

## Existing Infrastructure

| What exists | Where | Relevance |
|---|---|---|
| KB-14 WorklistItem model + service + handlers | KB-14 `internal/services/worklist_service.go` | Has worklist infrastructure — we extend it with PAI |
| KB-23 DecisionCards + EscalationEvents | KB-23 | Primary data source for worklist items |
| KB-26 PAI scores + AcuteEvents | KB-26 | Urgency scoring + acute detection |
| KB-20 PatientProfile + SummaryContext | KB-20 | Patient demographics + care team |
| KB-23 PAICardPrioritizer | KB-23 `internal/services/pai_card_prioritizer.go` | Already sorts cards by PAI — extend to full worklist |
| KB-20 CareTransition | KB-20 | Post-discharge status for context tags |

## File Inventory

### KB-23 — Worklist Engine
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/models/worklist_item.go` | WorklistItem, WorklistView, WorklistActionRequest, WorklistFeedback |
| Create | `internal/services/worklist_aggregator.go` | AggregateWorklistForClinician — joins PAI + cards + escalation + profile |
| Create | `internal/services/worklist_aggregator_test.go` | 6 tests |
| Create | `internal/services/sort_tier_engine.go` | SortAndTier — escalation tier → PAI → trajectory → attention gap |
| Create | `internal/services/sort_tier_engine_test.go` | 5 tests |
| Create | `internal/services/persona_filter.go` | ApplyPersonaFilter — HCF care manager, aged care nurse, GP, ASHA |
| Create | `internal/services/persona_filter_test.go` | 4 tests |
| Create | `internal/services/worklist_resolution.go` | HandleWorklistAction — ACKNOWLEDGE, CALL, DEFER, DISMISS |
| Create | `internal/services/worklist_resolution_test.go` | 5 tests |
| Create | `internal/api/worklist_handlers.go` | GET /worklist/:clinicianId, POST /worklist/:itemId/action, POST /worklist/:itemId/feedback |
| Modify | `internal/api/routes.go` | Add worklist route group |
| Modify | `internal/api/server.go` | Add worklist aggregator + resolver dependencies |

### Market Configs
| Action | File | Responsibility |
|---|---|---|
| Create | `market-configs/shared/worklist_parameters.yaml` | Sort rules, max items, refresh interval, persona action sets |
| Create | `market-configs/shared/persona_definitions.yaml` | 5 personas with scope, actions, language, channel |

**Total: 14 files (12 create, 2 modify), ~25 tests**

---

### Task 1: Worklist data models + persona config

**Files:**
- Create: `kb-23-decision-cards/internal/models/worklist_item.go`
- Create: `market-configs/shared/worklist_parameters.yaml`
- Create: `market-configs/shared/persona_definitions.yaml`

- [ ] **Step 1:** Create `worklist_item.go` with 4 types:

`WorklistItem` — ID, PatientID, PatientName, PatientAge, UrgencyTier (CRITICAL/HIGH/MODERATE/LOW), PAIScore, PAITrend (RISING/STABLE/FALLING), EscalationTier, TriggeringSource (PAI_CHANGE/CARD_GENERATED/ACUTE_EVENT/ESCALATION/TRANSITION_MILESTONE), PrimaryReason string, SuggestedAction string, SuggestedTimeframe string, ActionButtons []ActionButton, ContextTags []string, DominantDimension string, LastClinicianContactDays int, UnderlyingCardIDs []string, ResolutionState (PENDING/IN_PROGRESS/RESOLVED/DEFERRED), ResolvedBy string, ResolutionAction string, ComputedAt time.Time.

`ActionButton` — ActionCode string, DisplayLabel string, ConfirmationRequired bool.

`WorklistView` — ClinicianID, PersonaType, Items []WorklistItem, TotalCount, CriticalCount, HighCount, ModerateCount, LowCount, LastRefreshed time.Time.

`WorklistActionRequest` — ItemID, ClinicianID, ActionType (ACKNOWLEDGE/CALL_PATIENT/SCHEDULE/DEFER/DISMISS/ESCALATE), Notes string.

`WorklistFeedback` — ItemID, ClinicianID, FeedbackType (USEFUL/NOT_USEFUL/WRONG_PRIORITY), Reason string.

- [ ] **Step 2:** Create `worklist_parameters.yaml` with sort rules, max items per persona, refresh interval (5 min), action code definitions.

- [ ] **Step 3:** Create `persona_definitions.yaml` with 5 personas (HCF_CARE_MANAGER, AGED_CARE_NURSE, AUSTRALIA_GP, INDIA_GP, ASHA_WORKER) — each with scope, max_items, action_buttons, language, channel.

- [ ] **Step 4:** Verify compile + YAML parse. Commit: `feat(kb23): worklist models + persona config (Gap 18 Task 1)`

---

### Task 2: Worklist aggregator — "one most important thing per patient"

**Files:**
- Create: `kb-23-decision-cards/internal/services/worklist_aggregator.go`
- Create: `kb-23-decision-cards/internal/services/worklist_aggregator_test.go`

- [ ] **Step 1:** Write 6 tests:
1. `TestAggregator_SafetyEscalation_TopPriority` — patient with SAFETY escalation → worklist item with UrgencyTier CRITICAL, TriggeringSource ESCALATION
2. `TestAggregator_PAICritical_HighPriority` — patient with PAI CRITICAL, no escalation → worklist item with UrgencyTier HIGH
3. `TestAggregator_MultipleCards_OneItem` — patient with 3 active cards → single worklist item showing most urgent card
4. `TestAggregator_NoUrgentFindings_Excluded` — patient with PAI LOW, no cards, no escalation → no worklist item
5. `TestAggregator_AcuteEvent_Surfaces` — patient with CRITICAL acute event from Gap 16 → worklist item with temporal classification
6. `TestAggregator_TransitionMilestone_Surfaces` — patient in active transition with pending milestone → worklist item

- [ ] **Step 2:** Implement `WorklistAggregator` with:

```go
type PatientWorklistData struct {
    PatientID        string
    PatientName      string
    PatientAge       int
    PAIScore         float64
    PAITier          string
    PAITrend         string
    ActiveCards      []CardSummary
    PendingEscalations []EscalationSummary
    AcuteEvents      []AcuteEventSummary
    TransitionStatus *TransitionSummary
    LastClinicianContactDays int
    CKMStage         string
    IsPostDischarge  bool
}

type CardSummary struct {
    CardID           string
    TemplateID       string
    DifferentialID   string
    MCUGate          string
    ClinicianSummary string
    SafetyTier       string
    CreatedAt        time.Time
}

type EscalationSummary struct {
    ID        string
    Tier      string
    State     string
    Reason    string
    Action    string
    Timeframe string
}
```

`ComposeWorklistItem(data PatientWorklistData) *WorklistItem` — the core function. Applies 9-level priority cascade:
1. SAFETY escalation → CRITICAL item
2. IMMEDIATE escalation → CRITICAL item
3. PAI CRITICAL sustained → HIGH item
4. URGENT escalation → HIGH item
5. Acute event (72h, unresolved) → HIGH item
6. Transition milestone pending → MODERATE item
7. PAI HIGH → MODERATE item
8. Unacknowledged card (non-routine) → MODERATE item
9. Attention gap (>30 days no contact) → LOW item

Returns nil if no priority matches (patient in routine maintenance).

- [ ] **Step 3:** Run tests — all 6 pass. Commit: `feat(kb23): worklist aggregator — one-most-important-thing per patient (Gap 18 Task 2)`

---

### Task 3: Sort + tier engine

**Files:**
- Create: `kb-23-decision-cards/internal/services/sort_tier_engine.go`
- Create: `kb-23-decision-cards/internal/services/sort_tier_engine_test.go`

- [ ] **Step 1:** Write 5 tests:
1. `TestSort_SafetyFirst` — 3 items (SAFETY, URGENT, ROUTINE) → SAFETY first regardless of PAI
2. `TestSort_PAIDescending_SameTier` — 3 HIGH items with PAI 85/72/60 → sorted 85, 72, 60
3. `TestSort_RisingBeatsStable` — 2 items same tier, PAI within 5 points, one RISING one STABLE → RISING first
4. `TestSort_TransitionBoost` — post-discharge patient sorts before non-transition at equal criteria
5. `TestSort_TierCounts_Correct` — 10 items across tiers → CriticalCount/HighCount/etc correct

- [ ] **Step 2:** Implement `SortAndTier(items []WorklistItem, maxItems int) WorklistView` — multi-key stable sort: escalation tier desc → PAI desc → trajectory (RISING > STABLE > FALLING) → attention gap desc → transition boost. Tier grouping into CRITICAL/HIGH/MODERATE/LOW. Max truncation preserving all CRITICAL items.

- [ ] **Step 3:** Run tests. Commit: `feat(kb23): sort + tier engine — PAI-first multi-key sorting (Gap 18 Task 3)`

---

### Task 4: Persona filter

**Files:**
- Create: `kb-23-decision-cards/internal/services/persona_filter.go`
- Create: `kb-23-decision-cards/internal/services/persona_filter_test.go`

- [ ] **Step 1:** Write 4 tests:
1. `TestPersona_HCFCareManager_PanelScope` — HCF persona filters to assigned panel only, max 15 items
2. `TestPersona_AgedCareNurse_FacilityScope` — aged care persona shows all facility residents
3. `TestPersona_IndiaGP_ActionButtons` — India GP gets CALL_PATIENT, TELECONSULT, ASHA_OUTREACH buttons
4. `TestPersona_ASHA_SimplifiedLanguage` — ASHA persona translates "Cardiorenal syndrome" → "Heart and kidney strain — visit patient today"

- [ ] **Step 2:** Implement `ApplyPersonaFilter(items []WorklistItem, persona PersonaConfig, assignedPatientIDs []string) []WorklistItem` — filters by scope, applies max items, sets persona-specific action buttons, translates language for ASHA.

PersonaConfig loaded from YAML: Scope, MaxItems, ActionButtons []ActionButton, Language, SimplifyLanguage bool.

For ASHA language simplification: a `clinicalToLayperson` map translating clinical terms to plain language.

- [ ] **Step 3:** Run tests. Commit: `feat(kb23): persona filter — scope + actions + ASHA translation (Gap 18 Task 4)`

---

### Task 5: Resolution handler — one-tap actions

**Files:**
- Create: `kb-23-decision-cards/internal/services/worklist_resolution.go`
- Create: `kb-23-decision-cards/internal/services/worklist_resolution_test.go`

- [ ] **Step 1:** Write 5 tests:
1. `TestResolution_Acknowledge_UpdatesEscalation` — ACKNOWLEDGE action → escalation state ACKNOWLEDGED, worklist item RESOLVED
2. `TestResolution_Defer_MovesItem` — DEFER action → item ResolutionState DEFERRED
3. `TestResolution_Dismiss_CreatesFeedback` — DISMISS_NOT_USEFUL → WorklistFeedback record created
4. `TestResolution_CallPatient_LogsIntent` — CALL_PATIENT → action logged, item IN_PROGRESS
5. `TestResolution_Escalate_UpdatesState` — ESCALATE → item state ESCALATED

- [ ] **Step 2:** Implement `WorklistResolutionHandler` with `HandleAction(req WorklistActionRequest) (*ResolutionResult, error)`:
- ACKNOWLEDGE: calls escalation tracker RecordAcknowledgment, marks item RESOLVED
- CALL_PATIENT: logs call intent, marks item IN_PROGRESS
- DEFER: marks item DEFERRED with expiry (4h/24h/7d based on payload)
- DISMISS: creates WorklistFeedback, marks item RESOLVED
- ESCALATE: creates specialist referral record, marks item ESCALATED

`ResolutionResult` struct: Success bool, UpdatedState string, Message string.

- [ ] **Step 3:** Run tests. Commit: `feat(kb23): worklist resolution — one-tap action handling (Gap 18 Task 5)`

---

### Task 6: API handlers + server wiring

**Files:**
- Create: `kb-23-decision-cards/internal/api/worklist_handlers.go`
- Modify: `kb-23-decision-cards/internal/api/routes.go`
- Modify: `kb-23-decision-cards/internal/api/server.go`

- [ ] **Step 1:** Create 3 handlers:
- `GET /api/v1/worklist/:clinicianId` — aggregates worklist for clinician. Query params: persona (required), assigned_patients (comma-separated IDs), facility_id (for aged care). Returns WorklistView JSON.
- `POST /api/v1/worklist/:itemId/action` — body: WorklistActionRequest. Calls resolution handler. Returns ResolutionResult.
- `POST /api/v1/worklist/:itemId/feedback` — body: WorklistFeedback. Persists for trust calibration.

- [ ] **Step 2:** The GET handler flow:
1. Parse clinician ID + persona from request
2. Get assigned patient IDs (from query params or future KB-20 care team lookup)
3. For each patient: build PatientWorklistData from KB-23 card/escalation queries + PAI from KB-26 client
4. Call ComposeWorklistItem for each patient (filter nils)
5. Apply PersonaFilter
6. Call SortAndTier
7. Return WorklistView

For Sprint 1, the handler queries KB-23's local database for cards and escalations (same DB). PAI scores are fetched from KB-26 via the existing KB-26 client (or cached if available).

- [ ] **Step 3:** Add routes + server wiring with setter injection.

- [ ] **Step 4:** Build + full test sweep. Commit: `feat(kb23): worklist API — GET + action + feedback endpoints (Gap 18 Task 6)`

---

### Task 7: Integration test + YAML verification + final commit

- [ ] **Step 1:** Full test sweep KB-23 + KB-20 + KB-26.
- [ ] **Step 2:** Verify all YAML files parse.
- [ ] **Step 3:** Commit: `feat: complete Gap 18 clinician worklist (core engine)`
- [ ] **Step 4:** Push to origin.

---

## What This Plan Delivers vs Defers

| Delivered (this plan) | Deferred (Plan B) |
|----------------------|-------------------|
| WorklistItem model with all fields | Redis worklist cache |
| Aggregator with "one most important thing" | WebSocket for SAFETY pushes |
| Sort/tier engine (5-key sort) | Kafka event consumer for cache invalidation |
| 5 persona filters with ASHA translation | WhatsApp summary generator |
| Resolution handler (6 action types) | Shift handover generator |
| REST API (GET worklist, POST action, POST feedback) | Facility aggregation service |
| Persona + worklist YAML configs | PMS integration (Best Practice/Medical Director) |
| | Hindi/regional language YAML files |

## Verification Questions

1. Does a SAFETY escalation surface as the top worklist item? (yes / test)
2. Does a patient with 3 cards produce only 1 worklist item? (yes / test)
3. Does PAI descending sort work within same tier? (yes / test)
4. Does RISING trajectory beat STABLE at equal PAI? (yes / test)
5. Does HCF persona filter to assigned panel + max 15? (yes / test)
6. Does ASHA persona simplify clinical language? (yes / test)
7. Does ACKNOWLEDGE action update escalation state? (yes / test)
8. Does DISMISS create feedback for trust calibration? (yes / test)

## Effort Estimate

| Task | Scope | Expected |
|---|---|---|
| Task 1: Models + YAML | 3 files | 1-2 hours |
| Task 2: Aggregator (6 tests) | 2 files | 3-4 hours |
| Task 3: Sort/tier (5 tests) | 2 files | 1-2 hours |
| Task 4: Persona filter (4 tests) | 2 files | 2-3 hours |
| Task 5: Resolution (5 tests) | 2 files | 2-3 hours |
| Task 6: API + wiring | 3 files | 2-3 hours |
| Task 7: Integration + commit | sweep | 30 min |
| **Total** | **~14 files, ~25 tests** | **~12-18 hours** |
