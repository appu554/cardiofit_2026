# Gap 15: Escalation Protocol Engine — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build an escalation protocol engine in KB-23 that converts every persisted decision card into a tier-routed, channel-selected, time-tracked notification with acknowledgment countdown and automatic escalation — turning detection into action.

**Architecture:** The engine lives in KB-23 (where cards are generated) and hooks into the existing `notifyFHIR` post-persist path. Seven components form a pipeline: Router (card→tier), ChannelSelector (tier→channels), DeliveryScheduler (channels→dispatch), AcknowledgmentTracker (T0-T3 timestamps), EscalationManager (orchestrator + timeout goroutine), ChannelAdapters (noop/SMS/WhatsApp/push behind interface), and AuditLogger (hash-chained lifecycle events via existing AuditService). YAML configs define tier→channel→timeout mappings with India/Australia overrides.

**Tech Stack:** Go 1.21 (Gin, GORM) for KB-23 extensions. PostgreSQL for escalation state. Existing AuditService for lifecycle audit trail. NotificationChannel interface with pluggable adapters (noop for tests, stubs for Sprint 1, real providers in deployment). YAML market configs.

---

## Existing Infrastructure

| What exists | Where | Relevance |
|---|---|---|
| 9 card persist sites + `notifyFHIR` | KB-23 `internal/services/` | Escalation hooks into post-persist path |
| `FHIRCardNotifier` interface | KB-23 `internal/services/inertia_orchestrator.go` | Pattern for optional post-persist hooks |
| `AuditService` (hash-chained) | KB-23 `internal/services/audit_service.go` | Lifecycle events append to same audit trail |
| `MCUGate` (HALT/MODIFY/SAFE) | KB-23 card models | Maps to SAFETY/IMMEDIATE/ROUTINE escalation tiers |
| PAI `EscalationTier` field | KB-26 `internal/services/pai_engine.go` | PAI already computes SAFETY/IMMEDIATE/URGENT/ROUTINE |
| KB-14 escalation infrastructure | KB-14 `internal/services/escalation_engine.go` | Future consumer of KB-23 escalation events |

## File Inventory

### KB-23 (Decision Cards) — Escalation Engine
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/models/escalation_event.go` | EscalationEvent GORM model, EscalationTier constants, EscalationState constants, ClinicianPreferences |
| Create | `internal/services/escalation_router.go` | `RouteCardToTier(card) → EscalationTier` — MCU gate mapping + PAI amplification + sustained-elevation gate + deduplication |
| Create | `internal/services/escalation_router_test.go` | 7 tests |
| Create | `internal/services/channel_selector.go` | `SelectChannels(tier, prefs, config) → []Channel` — tier-based + preference + quiet hours |
| Create | `internal/services/channel_selector_test.go` | 6 tests |
| Create | `internal/services/notification_channel.go` | `NotificationChannel` interface + `NoopChannel` + `SMSChannel` + `WhatsAppChannel` + `PushChannel` (stubs) |
| Create | `internal/services/acknowledgment_tracker.go` | T0-T3 timestamps, timeout management, de-escalation |
| Create | `internal/services/acknowledgment_tracker_test.go` | 7 tests |
| Create | `internal/services/escalation_manager.go` | `HandleCardCreated(card)` orchestrator + background timeout goroutine |
| Create | `internal/services/escalation_manager_test.go` | 5 tests |
| Create | `internal/api/escalation_handlers.go` | POST acknowledge, POST action, GET patient escalations, GET metrics, POST preferences |
| Modify | `internal/api/routes.go` | Add escalation route group |
| Modify | `internal/api/server.go` | Add EscalationManager field + setter |
| Modify | `internal/database/connection.go` | AutoMigrate EscalationEvent + ClinicianPreferences |

### Market Configs
| Action | File | Responsibility |
|---|---|---|
| Create | `market-configs/shared/escalation_protocols.yaml` | Tier definitions, card type routing table, PAI routing, sustained-elevation gate, deduplication, quiet hours |
| Create | `market-configs/india/escalation_overrides.yaml` | WhatsApp primary, ASHA chain, rural timeouts, Ramadan quiet hours |
| Create | `market-configs/australia/escalation_overrides.yaml` | Aged care chain, indigenous remote gaps, GPMP alignment |

**Total: 17 files (14 create, 3 modify)**

---

### Task 1: Escalation models + YAML config

**Files:**
- Create: `kb-23-decision-cards/internal/models/escalation_event.go`
- Create: `market-configs/shared/escalation_protocols.yaml`
- Create: `market-configs/india/escalation_overrides.yaml`
- Create: `market-configs/australia/escalation_overrides.yaml`

- [ ] **Step 1:** Create `escalation_event.go` with:
- `EscalationTier` constants: SAFETY (30min), IMMEDIATE (4hr), URGENT (24hr), ROUTINE (7d), INFORMATIONAL (passive)
- `EscalationState` constants: PENDING, DELIVERED, ACKNOWLEDGED, ACTED, ESCALATED, RESOLVED, CANCELLED, EXPIRED
- `EscalationEvent` GORM model: ID, PatientID, CardID, TriggerType (CARD_GENERATED/PAI_CHANGE), EscalationTier, CurrentState, AssignedClinicianID, AssignedClinicianRole, Channels (JSON array), DeliveryAttempts, CreatedAt (T0), DeliveredAt (T1), AcknowledgedAt (T2), ActedAt (T3), ResolvedAt, EscalatedAt, EscalationLevel (1-3), TimeoutAt, PreviousEventID, PAIScoreAtTrigger, PAITierAtTrigger, PrimaryReason, SuggestedAction, SuggestedTimeframe. Indexes on (patient_id, current_state), (timeout_at).
- `ClinicianPreferences` GORM model: ClinicianID (primary), PreferredChannels (JSON array), QuietHoursStart (string "22:00"), QuietHoursEnd (string "06:00"), Timezone, MaxNotificationsPerHour, CreatedAt, UpdatedAt.

- [ ] **Step 2:** Create `escalation_protocols.yaml` with:
```yaml
tiers:
  SAFETY:
    target_response_minutes: 30
    channels: [push, sms, whatsapp]
    simultaneous: true
    timeout_minutes: 30
    max_escalation_levels: 3
    quiet_hours_policy: BYPASS
    auto_resolve_on_improvement: false
  IMMEDIATE:
    target_response_minutes: 240
    channels: [push]
    fallback_channels: [sms]
    fallback_after_minutes: 120
    timeout_minutes: 240
    max_escalation_levels: 2
    quiet_hours_policy: QUEUE
    auto_resolve_on_improvement: true
  URGENT:
    target_response_minutes: 1440
    channels: [push]
    fallback_channels: [sms]
    fallback_after_minutes: 1440
    timeout_minutes: 1440
    max_escalation_levels: 1
    quiet_hours_policy: SUPPRESS
    auto_resolve_on_improvement: true
  ROUTINE:
    target_response_minutes: 10080
    channels: [in_app]
    timeout_minutes: 0
    max_escalation_levels: 0
    quiet_hours_policy: SUPPRESS
    auto_resolve_on_improvement: true
  INFORMATIONAL:
    target_response_minutes: 0
    channels: []
    timeout_minutes: 0
    max_escalation_levels: 0

card_type_routing:
  RENAL_CONTRAINDICATION: SAFETY
  RENAL_DOSE_REDUCE: URGENT
  CKM_4C_MANDATORY_MEDICATION: IMMEDIATE
  THERAPEUTIC_INERTIA: URGENT
  DUAL_DOMAIN_INERTIA: IMMEDIATE
  MASKED_HYPERTENSION: URGENT
  ADHERENCE_GAP: ROUTINE
  DEPRESCRIBING_REVIEW: ROUTINE
  PHENOTYPE_TRANSITION: ROUTINE
  PHENOTYPE_FLAP_WARNING: ROUTINE
  MONITORING_LAPSED: URGENT

pai_routing:
  CRITICAL: IMMEDIATE
  HIGH: URGENT
  MODERATE: ROUTINE
  LOW: INFORMATIONAL
  MINIMAL: INFORMATIONAL

amplification_rules:
  - condition: "mcu_gate == HALT"
    escalate_to: SAFETY
  - condition: "pai_tier == CRITICAL AND egfr < 30"
    escalate_to: SAFETY

sustained_elevation:
  enabled: true
  min_consecutive: 2
  exempt_tiers: [SAFETY]

deduplication:
  window_hours: 24

default_quiet_hours:
  start: "22:00"
  end: "06:00"
```

- [ ] **Step 3:** Create India + Australia override YAMLs with channel priorities, ASHA chain, aged care chain, rural connectivity adjustments.

- [ ] **Step 4:** Verify models compile + YAML parses.

- [ ] **Step 5:** Commit: `feat(kb23): escalation models + protocol YAML config (Gap 15 Task 1)`

---

### Task 2: Escalation router — card type + PAI routing + gates

**Files:**
- Create: `kb-23-decision-cards/internal/services/escalation_router.go`
- Create: `kb-23-decision-cards/internal/services/escalation_router_test.go`

- [ ] **Step 1:** Write 7 failing tests:
1. `TestRouter_RenalContraindication_Safety` — RENAL_CONTRAINDICATION card → SAFETY tier
2. `TestRouter_TherapeuticInertia_Urgent` — THERAPEUTIC_INERTIA card → URGENT tier
3. `TestRouter_HaltGate_AmplifiedToSafety` — any card with MCU gate HALT → SAFETY regardless of card type routing
4. `TestRouter_PAICritical_EGFRLow_Safety` — PAI CRITICAL + eGFR <30 → amplified to SAFETY
5. `TestRouter_SustainedElevation_Suppressed` — PAI HIGH but not sustained (first computation) → INFORMATIONAL (suppressed)
6. `TestRouter_SustainedElevation_Bypassed_Safety` — SAFETY tier bypasses sustained gate
7. `TestRouter_Deduplication_24h` — same card type for same patient within 24h → nil (deduplicated)

- [ ] **Step 2:** Implement `EscalationRouter` with:
- `RouteCard(card DecisionCard, paiTier string, paiScore float64, eGFR *float64) (*RoutingResult, error)`
- `RoutingResult` struct: Tier, Reason, Suppressed bool, SuppressionReason
- 5-stage pipeline: card type lookup → PAI routing → amplification → sustained gate → deduplication
- `EscalationProtocolConfig` loaded from YAML with `LoadEscalationConfig(path)`
- In-memory sustained-elevation tracker: `map[string]int` (patientID → consecutive above-threshold count)
- Deduplication via DB query: check for existing PENDING/DELIVERED/ACKNOWLEDGED escalation for same patient+cardType within 24h

- [ ] **Step 3:** Run tests — all 7 pass.

- [ ] **Step 4:** Commit: `feat(kb23): escalation router — 5-stage routing pipeline (Gap 15 Task 2)`

---

### Task 3: Channel selector — tier to channels with quiet hours

**Files:**
- Create: `kb-23-decision-cards/internal/services/channel_selector.go`
- Create: `kb-23-decision-cards/internal/services/channel_selector_test.go`

- [ ] **Step 1:** Write 6 failing tests:
1. `TestSelector_Safety_AllChannels` — SAFETY → [push, sms, whatsapp] simultaneously
2. `TestSelector_Immediate_PrimaryWithFallback` — IMMEDIATE → [push] now, [sms] at fallback time
3. `TestSelector_Routine_InAppOnly` — ROUTINE → [in_app] only
4. `TestSelector_QuietHours_SuppressUrgent` — URGENT during quiet hours (23:00) → suppressed, queued for 06:00
5. `TestSelector_QuietHours_BypassSafety` — SAFETY during quiet hours → delivered immediately
6. `TestSelector_ClinicianPreference_Override` — clinician prefers WhatsApp for IMMEDIATE → [whatsapp] instead of [push]

- [ ] **Step 2:** Implement `ChannelSelector` with:
- `SelectChannels(tier, clinicianPrefs *ClinicianPreferences, now time.Time, config) → ChannelSelection`
- `ChannelSelection` struct: PrimaryChannels, FallbackChannels, FallbackAfter time.Duration, Suppressed bool, SuppressedUntil *time.Time, Simultaneous bool
- Quiet hours check: parse clinicianPrefs.QuietHoursStart/End in their timezone, compare to `now`
- SAFETY always bypasses quiet hours and ignores clinician opt-outs

- [ ] **Step 3:** Run tests — all 6 pass.

- [ ] **Step 4:** Commit: `feat(kb23): channel selector — tier routing + quiet hours + preferences (Gap 15 Task 3)`

---

### Task 4: Notification channel interface + adapters

**Files:**
- Create: `kb-23-decision-cards/internal/services/notification_channel.go`

- [ ] **Step 1:** Create `NotificationChannel` interface:
```go
type NotificationChannel interface {
    Send(notification EscalationNotification) (DeliveryResult, error)
    Name() string
}

type EscalationNotification struct {
    EscalationID   string
    PatientID      string
    PatientName    string
    ClinicianID    string
    ClinicianPhone string
    Tier           string
    PrimaryReason  string
    SuggestedAction string
    Timeframe      string
    CardID         string
}

type DeliveryResult struct {
    Status    string // SENT, FAILED, PENDING
    MessageID string
    Channel   string
    SentAt    time.Time
}
```

- [ ] **Step 2:** Implement 4 adapters:
- `NoopChannel` — logs and returns SENT (testing/dev)
- `SMSChannelStub` — logs SMS content, returns SENT (real Twilio wiring in deployment)
- `WhatsAppChannelStub` — logs WhatsApp content with template params, returns SENT
- `PushChannelStub` — logs push payload, returns SENT

Each stub writes to a `zap.Logger` with structured fields so delivery attempts are visible in logs. Real provider wiring (Twilio SID, Firebase credentials) happens via environment variables in deployment — stubs check for env vars and fall back to logging.

- [ ] **Step 3:** Verify compile. Commit: `feat(kb23): notification channel interface + stub adapters (Gap 15 Task 4)`

---

### Task 5: Acknowledgment tracker — T0-T3 + timeout + de-escalation

**Files:**
- Create: `kb-23-decision-cards/internal/services/acknowledgment_tracker.go`
- Create: `kb-23-decision-cards/internal/services/acknowledgment_tracker_test.go`

- [ ] **Step 1:** Write 7 failing tests:
1. `TestTracker_RecordDelivery_SetsT1` — record delivery → T1 set, state DELIVERED
2. `TestTracker_RecordAcknowledgment_SetsT2` — acknowledge → T2 set, state ACKNOWLEDGED
3. `TestTracker_RecordAction_SetsT3` — record action → T3 set, state ACTED
4. `TestTracker_Timeout_Level1_EscalatesToLevel2` — pending escalation past timeout → new level-2 event created
5. `TestTracker_Timeout_MaxLevel_Expires` — level at max → state EXPIRED
6. `TestTracker_DeEscalation_CancelsUrgent` — PAI drops below threshold → URGENT escalation cancelled, state RESOLVED reason CONDITION_IMPROVED
7. `TestTracker_DeEscalation_SafetyNotCancelled` — PAI drops → SAFETY escalation NOT cancelled (requires explicit ack)

- [ ] **Step 2:** Implement `AcknowledgmentTracker` with:
- `RecordDelivery(escalationID, channel, messageID)` — sets DeliveredAt, state→DELIVERED
- `RecordAcknowledgment(escalationID, clinicianID)` — sets AcknowledgedAt, AcknowledgedBy, state→ACKNOWLEDGED
- `RecordAction(escalationID, actionType, actionDetail)` — sets ActedAt, state→ACTED
- `CheckTimeouts(now time.Time)` — queries `WHERE current_state IN ('PENDING','DELIVERED') AND timeout_at < now`, for each: if level < max → create new escalation event at level+1, link via PreviousEventID; if level == max → set state EXPIRED
- `HandlePAIImprovement(patientID, newTier)` — if non-SAFETY pending escalation exists and newTier is below the escalation's tier threshold → set state RESOLVED, reason "CONDITION_IMPROVED"
- Persistence via `gorm.DB` (same database as decision cards)

- [ ] **Step 3:** Run tests — all 7 pass.

- [ ] **Step 4:** Commit: `feat(kb23): acknowledgment tracker — T0-T3 lifecycle + timeout + de-escalation (Gap 15 Task 5)`

---

### Task 6: Escalation manager — orchestrator + background timeout

**Files:**
- Create: `kb-23-decision-cards/internal/services/escalation_manager.go`
- Create: `kb-23-decision-cards/internal/services/escalation_manager_test.go`

- [ ] **Step 1:** Write 5 failing tests:
1. `TestManager_HandleCard_SafetyTier_CreatesEscalation` — RENAL_CONTRAINDICATION card → EscalationEvent created with tier SAFETY, state PENDING, timeout 30min
2. `TestManager_HandleCard_RoutineTier_NoEscalation` — ROUTINE card → no EscalationEvent (ROUTINE uses in_app only, no active escalation tracking)
3. `TestManager_HandleCard_Informational_Suppressed` — INFORMATIONAL → nothing created
4. `TestManager_HandleCard_Deduplicated` — second card of same type within 24h → deduplicated
5. `TestManager_TimeoutTick_EscalatesExpired` — manager's background tick finds expired escalation → escalates

- [ ] **Step 2:** Implement `EscalationManager` with:
- `HandleCardCreated(card *models.DecisionCard, paiTier string, paiScore float64)` — the entry point called from every card persist site:
  1. Call router.RouteCard → get tier
  2. If INFORMATIONAL or suppressed → log and return
  3. If ROUTINE → create event with state PENDING but no active channel dispatch
  4. If SAFETY/IMMEDIATE/URGENT → call channelSelector, create EscalationEvent, dispatch via adapters, call auditService.Append
  5. Set TimeoutAt = now + tier's timeout_minutes
- `StartTimeoutChecker(ctx context.Context, interval time.Duration)` — background goroutine that calls tracker.CheckTimeouts every interval (1 min). Cancellable via context.
- `HandlePAIUpdate(patientID, newTier, newScore)` — delegates to tracker.HandlePAIImprovement for de-escalation
- Dependencies: Router, ChannelSelector, AcknowledgmentTracker, NotificationChannels map, AuditService, DB, Logger

- [ ] **Step 3:** Run tests — all 5 pass.

- [ ] **Step 4:** Commit: `feat(kb23): escalation manager — orchestrator + timeout background (Gap 15 Task 6)`

---

### Task 7: API handlers + server wiring

**Files:**
- Create: `kb-23-decision-cards/internal/api/escalation_handlers.go`
- Modify: `kb-23-decision-cards/internal/api/routes.go`
- Modify: `kb-23-decision-cards/internal/api/server.go`
- Modify: `kb-23-decision-cards/internal/database/connection.go`

- [ ] **Step 1:** Create handlers:
- `POST /api/v1/escalation/:id/acknowledge` — calls tracker.RecordAcknowledgment
- `POST /api/v1/escalation/:id/action` — calls tracker.RecordAction (body: actionType, actionDetail)
- `GET /api/v1/escalation/patient/:patientId` — returns all escalations for patient, sorted by tier desc
- `GET /api/v1/escalation/metrics` — returns aggregate metrics: avg T0→T2 per tier, timeout rate, escalation level distribution
- `POST /api/v1/clinician/:clinicianId/preferences` — upserts ClinicianPreferences (validates SAFETY cannot be opted out)

- [ ] **Step 2:** Add escalation route group to routes.go. Add EscalationManager + AutoMigrate for EscalationEvent + ClinicianPreferences.

- [ ] **Step 3:** Add `SetEscalationManager` setter to Server (same pattern as SetFHIRNotifier).

- [ ] **Step 4:** Build + test.

- [ ] **Step 5:** Commit: `feat(kb23): escalation API handlers + server wiring (Gap 15 Task 7)`

---

### Task 8: Wire escalation into card persist path + integration test

**Files:**
- Modify: `kb-23-decision-cards/internal/services/inertia_orchestrator.go` (exemplar — add escalation call after notifyFHIR)
- Full test sweep

- [ ] **Step 1:** In the InertiaOrchestrator's `persistInertiaCard` (the exemplar), add after `notifyFHIR(o.fhirNotifier, card)`:
```go
if o.escalationManager != nil {
    go o.escalationManager.HandleCardCreated(card, "", 0) // PAI context added by caller when available
}
```
Add `escalationManager *EscalationManager` field + `SetEscalationManager` setter to InertiaOrchestrator.

- [ ] **Step 2:** Document in a comment that the same pattern should be applied to the other 8 persist sites (PrioritySignalHandler, RenalAnticipatoryBatch, MonitoringEngagementBatch) — but defer wiring them all until the escalation manager is validated in production with the inertia path.

- [ ] **Step 3:** Full test sweep: KB-23 `go test ./... -count=1`

- [ ] **Step 4:** Commit: `feat: complete Gap 15 escalation protocol engine`

- [ ] **Step 5:** Push to origin.

---

## Verification Questions

1. Does a RENAL_CONTRAINDICATION card route to SAFETY tier? (yes / test)
2. Does MCU gate HALT amplify any card to SAFETY? (yes / test)
3. Does PAI CRITICAL + eGFR <30 amplify to SAFETY? (yes / test)
4. Does a non-sustained PAI spike get suppressed? (yes / test)
5. Does SAFETY bypass the sustained-elevation gate? (yes / test)
6. Does quiet hours suppress URGENT but not SAFETY? (yes / test)
7. Does a timeout at level 1 escalate to level 2? (yes / test)
8. Does PAI improvement cancel URGENT but not SAFETY? (yes / test)
9. Does deduplication prevent double-notification within 24h? (yes / test)
10. Does the clinician preferences endpoint reject SAFETY opt-out? (yes / handler)
11. Are all KB-23 test suites green? (yes / sweep)

---

## Effort Estimate

| Task | Scope | Expected |
|---|---|---|
| Task 1: Models + YAML | 4 files, ~300 LOC models + ~150 LOC YAML | 1-2 hours |
| Task 2: Router (7 tests) | 2 files, ~250 LOC | 2-3 hours |
| Task 3: Channel selector (6 tests) | 2 files, ~180 LOC | 1-2 hours |
| Task 4: Channel adapters | 1 file, ~150 LOC | 1 hour |
| Task 5: Acknowledgment tracker (7 tests) | 2 files, ~250 LOC | 2-3 hours |
| Task 6: Escalation manager (5 tests) | 2 files, ~200 LOC | 2-3 hours |
| Task 7: API + wiring | 4 files, ~200 LOC | 1-2 hours |
| Task 8: Persist path + integration | 2 files modified + sweep | 1 hour |
| **Total** | **~17 files, ~1680 LOC, ~37 tests** | **~12-18 hours** |
