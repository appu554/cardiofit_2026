# V5 Plan A: KB-14 PAI Integration — Escalation + Worklist + Notifications (Gaps 15+18+23)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Connect the clinical intelligence engine (KB-23 decision cards + KB-26 PAI scores) to the operational engine (KB-14 Care Navigator) so that every decision card generates a triaged, escalated, PAI-sorted worklist item with real notification delivery.

**Architecture:** Three integration bridges: (1) KB-23 Client in KB-14 that creates tasks from decision cards with clinical urgency mapping; (2) KB-26 Client in KB-14 that fetches PAI scores for worklist sorting; (3) Enhanced notification service with WhatsApp Business API + Firebase push. The worklist sort order changes from `due_date ASC` to `PAI DESC` as primary, with `due_date` as tiebreaker. A new SAFETY escalation tier maps KB-23's MCU HALT gate to a 30-minute response window with multi-channel blast.

**Tech Stack:** Go 1.21 (Gin, GORM, logrus) for KB-14 extensions. Existing KB-14 task/escalation/worklist infrastructure. HTTP clients for KB-23 + KB-26 cross-service calls. WhatsApp Business API (Twilio) + Firebase Cloud Messaging for delivery.

---

## Existing Infrastructure (KB-14)

| What exists | Where | Relevance |
|---|---|---|
| Task lifecycle (CREATED→ASSIGNED→COMPLETED) | KB-14 `internal/models/task.go` | Decision cards become tasks |
| 4-level SLA escalation (WARNING→EXECUTIVE) | KB-14 `internal/services/escalation_engine.go` | Extend with clinical SAFETY tier |
| Worklist sorted by due_date | KB-14 `internal/services/worklist_service.go` | Add PAI sort order |
| 5 notification channels (stub) | KB-14 `internal/services/notification_service.go` | Wire real providers |
| Task factory from KB-3/KB-9/KB-12 | KB-14 `internal/services/task_factory.go` | Add KB-23 card source |
| WorklistItem model | KB-14 `internal/models/worklist.go` | Add PAI fields |

## File Inventory

### KB-14 (Care Navigator) — Integration
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/clients/kb23_client.go` | Fetch decision cards + card details from KB-23 |
| Create | `internal/clients/kb26_client.go` | Fetch PAI score for a patient from KB-26 |
| Create | `internal/clients/kb23_client_test.go` | 3 integration tests |
| Create | `internal/clients/kb26_client_test.go` | 2 integration tests |
| Modify | `internal/models/worklist.go` | Add PAIScore, PAITier, PAITrend, DominantDimension fields |
| Modify | `internal/models/escalation.go` | Add SAFETY level (level 5) with 30-minute SLA |
| Modify | `internal/models/task.go` | Add KB23_DECISION_CARD source + ClinicalUrgency field |
| Create | `internal/services/kb23_task_factory.go` | Create tasks from KB-23 decision cards with urgency mapping |
| Create | `internal/services/kb23_task_factory_test.go` | 5 tests: HALT→SAFETY, MODIFY→URGENT, SAFE→ROUTINE, card fields mapped, missing card handled |
| Modify | `internal/services/worklist_service.go` | PAI-first sort order, PAI enrichment on worklist build |
| Create | `internal/services/worklist_pai_enricher.go` | Batch-fetch PAI scores for worklist patients |
| Create | `internal/services/worklist_pai_enricher_test.go` | 3 tests: enrichment populates, missing PAI defaults, empty worklist |
| Modify | `internal/services/escalation_engine.go` | Add SAFETY tier logic (30-min SLA, multi-channel blast) |
| Create | `internal/services/notification_whatsapp.go` | WhatsApp Business API integration via Twilio |
| Create | `internal/services/notification_firebase.go` | Firebase Cloud Messaging push notifications |
| Modify | `internal/services/notification_service.go` | Wire real providers, dispatch by channel |
| Create | `internal/api/kb23_sync_handlers.go` | POST /api/v1/sync/kb23 + webhook for real-time card events |
| Modify | `internal/api/server.go` | Add KB-23 client, KB-26 client, enhanced notification deps |
| Modify | `main.go` | Wire new clients and services |
| Create | `market-configs/shared/escalation_protocols.yaml` | Tier→channel→timeout mapping |

**Total: 20 files (10 create, 10 modify)**

---

### Task 1: Add PAI fields to WorklistItem + SAFETY escalation tier

**Files:**
- Modify: `kb-14-care-navigator/internal/models/worklist.go`
- Modify: `kb-14-care-navigator/internal/models/escalation.go`
- Modify: `kb-14-care-navigator/internal/models/task.go`
- Create: `market-configs/shared/escalation_protocols.yaml`

- [ ] **Step 1:** Add PAI fields to WorklistItem: `PAIScore float64`, `PAITier string`, `PAITrend string` (RISING/STABLE/FALLING), `DominantDimension string`, `ClinicalUrgency string` (SAFETY/IMMEDIATE/URGENT/ROUTINE).

- [ ] **Step 2:** Add SAFETY escalation level (level 5) to the escalation model with 30-minute SLA. Add `ClinicalEscalationTier string` field to the Escalation model for the clinical urgency mapping alongside the existing SLA-based levels.

- [ ] **Step 3:** Add `KB23_DECISION_CARD` to the task Source enum. Add `ClinicalUrgency string` and `MCUGate string` fields to the Task model.

- [ ] **Step 4:** Create `escalation_protocols.yaml` defining the tier→channel→timeout mapping:
```yaml
tiers:
  SAFETY:
    sla_minutes: 30
    channels: [push, sms, whatsapp, in_app, email]
    fallback_minutes: 15
    fallback_action: "escalate_to_team_lead"
  IMMEDIATE:
    sla_minutes: 120
    channels: [push, whatsapp, in_app, email]
    fallback_minutes: 60
    fallback_action: "sms_reminder"
  URGENT:
    sla_minutes: 1440
    channels: [push, in_app, email]
    fallback_minutes: 720
    fallback_action: "push_reminder"
  ROUTINE:
    sla_minutes: 10080
    channels: [in_app]
    fallback_minutes: 0
    fallback_action: "none"
```

- [ ] **Step 5:** Verify models compile, commit: `feat(kb14): PAI worklist fields + SAFETY escalation tier (V5 Plan A Task 1)`

---

### Task 2: Build KB-23 + KB-26 HTTP clients

**Files:**
- Create: `kb-14-care-navigator/internal/clients/kb23_client.go`
- Create: `kb-14-care-navigator/internal/clients/kb26_client.go`
- Create: `kb-14-care-navigator/internal/clients/kb23_client_test.go`
- Create: `kb-14-care-navigator/internal/clients/kb26_client_test.go`

- [ ] **Step 1:** Build KB23Client with: `FetchActiveCards(patientID) → []DecisionCardSummary`, `FetchCardDetail(cardID) → *DecisionCardDetail`, `AcknowledgeCard(cardID, clinicianID, action) error`. DecisionCardSummary mirrors KB-23's card output: CardID, PatientID, TemplateID, ClinicianSummary, SafetyTier, MCUGate, CreatedAt. Circuit-breaker protected (reuse `pkg/resilience` pattern from KB-23).

- [ ] **Step 2:** Build KB26Client with: `FetchPAIScore(patientID) → *PAIScoreResponse`, `FetchPAIBatch(patientIDs []string) → map[string]*PAIScoreResponse`. PAIScoreResponse: Score, Tier, Trend, DominantDimension, SuggestedAction, SuggestedTimeframe, ComputedAt.

- [ ] **Step 3:** Write 3 KB-23 integration tests (httptest.Server pattern): fetch cards returns list, fetch detail returns card, acknowledge sends POST.

- [ ] **Step 4:** Write 2 KB-26 integration tests: fetch PAI returns score, batch fetch returns map.

- [ ] **Step 5:** Commit: `feat(kb14): KB-23 + KB-26 HTTP clients (V5 Plan A Task 2)`

---

### Task 3: Build KB-23 task factory — decision cards become worklist items

**Files:**
- Create: `kb-14-care-navigator/internal/services/kb23_task_factory.go`
- Create: `kb-14-care-navigator/internal/services/kb23_task_factory_test.go`

- [ ] **Step 1:** Write 5 failing tests:
1. `TestKB23Factory_HaltGate_SafetyTask` — MCU gate HALT → task with Priority CRITICAL, ClinicalUrgency SAFETY, SLA 30 minutes
2. `TestKB23Factory_ModifyGate_UrgentTask` — MCU gate MODIFY → Priority HIGH, ClinicalUrgency URGENT, SLA 24 hours
3. `TestKB23Factory_SafeGate_RoutineTask` — MCU gate SAFE → Priority MEDIUM, ClinicalUrgency ROUTINE, SLA 7 days
4. `TestKB23Factory_CardFieldsMapped` — ClinicianSummary → Task.Description, TemplateID → Task.SourceID, PatientID preserved
5. `TestKB23Factory_MissingCard_Error` — nil card input → error returned

- [ ] **Step 2:** Implement `KB23TaskFactory.CreateFromDecisionCard(card DecisionCardSummary) (*Task, error)` with MCU gate → clinical urgency mapping: HALT→SAFETY (30min SLA), MODIFY→IMMEDIATE (2hr SLA), SAFE→ROUTINE (7d SLA). Set Source = KB23_DECISION_CARD.

- [ ] **Step 3:** Run tests — all 5 pass.

- [ ] **Step 4:** Commit: `feat(kb14): KB-23 task factory — decision cards to worklist items (V5 Plan A Task 3)`

---

### Task 4: PAI-enriched worklist with smart sort order

**Files:**
- Create: `kb-14-care-navigator/internal/services/worklist_pai_enricher.go`
- Create: `kb-14-care-navigator/internal/services/worklist_pai_enricher_test.go`
- Modify: `kb-14-care-navigator/internal/services/worklist_service.go`

- [ ] **Step 1:** Write 3 tests for PAI enricher:
1. `TestEnricher_PopulatesPAI` — 3 worklist items, KB-26 returns PAI scores → all items have PAIScore/PAITier/PAITrend populated
2. `TestEnricher_MissingPAI_DefaultsZero` — KB-26 returns no score for one patient → that item gets PAIScore=0, PAITier="UNKNOWN"
3. `TestEnricher_EmptyWorklist_NoFetch` — empty worklist → no KB-26 calls made

- [ ] **Step 2:** Implement `WorklistPAIEnricher.Enrich(items []WorklistItem) []WorklistItem` — extracts unique patient IDs, batch-fetches PAI scores from KB-26, populates PAI fields on each worklist item.

- [ ] **Step 3:** Modify worklist_service.go: after building the worklist, call `enricher.Enrich(items)`, then re-sort: primary sort by `PAIScore DESC` (highest acuity first), secondary by `DueDate ASC` (soonest due as tiebreaker). Add `sort_by=pai` query parameter option.

- [ ] **Step 4:** Run tests — all pass.

- [ ] **Step 5:** Commit: `feat(kb14): PAI-enriched worklist with acuity-first sort (V5 Plan A Task 4)`

---

### Task 5: SAFETY escalation tier + enhanced notification dispatch

**Files:**
- Modify: `kb-14-care-navigator/internal/services/escalation_engine.go`
- Create: `kb-14-care-navigator/internal/services/notification_whatsapp.go`
- Create: `kb-14-care-navigator/internal/services/notification_firebase.go`
- Modify: `kb-14-care-navigator/internal/services/notification_service.go`

- [ ] **Step 1:** Extend EscalationEngine to handle ClinicalUrgency: when a task has ClinicalUrgency=SAFETY, use the SAFETY SLA (30 min) and escalation thresholds (25%/50%/75%/100% = 7.5/15/22.5/30 min). On SAFETY escalation level ≥ URGENT, send multi-channel blast (all 5 channels simultaneously).

- [ ] **Step 2:** Implement `WhatsAppNotifier` with Twilio WhatsApp Business API integration: `Send(to, templateName, params) error`. Uses environment variables `TWILIO_ACCOUNT_SID`, `TWILIO_AUTH_TOKEN`, `TWILIO_WHATSAPP_FROM`. Message format uses WhatsApp template with patient name, urgency, and suggested action.

- [ ] **Step 3:** Implement `FirebasePushNotifier` with Firebase Cloud Messaging: `Send(deviceToken, title, body, data) error`. Uses environment variable `FIREBASE_CREDENTIALS_PATH`. Notification payload includes PAI score, tier, and deep link to patient worklist item.

- [ ] **Step 4:** Update NotificationService from stub to real dispatch: route by channel config (from escalation_protocols.yaml). Each channel has a real provider (WhatsApp→Twilio, push→Firebase, SMS→Twilio SMS, email→SendGrid, in_app→database insert, pager→PagerDuty). Providers that aren't configured fall back to logging (graceful degradation).

- [ ] **Step 5:** Build + test. Commit: `feat(kb14): SAFETY escalation + WhatsApp/push notification delivery (V5 Plan A Task 5)`

---

### Task 6: KB-23 sync handler + webhook + server wiring

**Files:**
- Create: `kb-14-care-navigator/internal/api/kb23_sync_handlers.go`
- Modify: `kb-14-care-navigator/internal/api/server.go`
- Modify: `kb-14-care-navigator/main.go`

- [ ] **Step 1:** Create sync handler: `POST /api/v1/sync/kb23` — fetches all active cards from KB-23 for all patients, creates tasks via KB23TaskFactory for any cards without existing tasks. `POST /api/v1/webhooks/kb23-card-created` — real-time webhook called by KB-23 when a new card is persisted (same FHIRCardNotifier pattern), creates task immediately.

- [ ] **Step 2:** Add KB-23 client, KB-26 client, WorklistPAIEnricher, KB23TaskFactory to Server struct. Wire in main.go with environment variables `KB23_URL` and `KB26_URL`.

- [ ] **Step 3:** Add webhook route and sync route to router.

- [ ] **Step 4:** Build + full KB-14 test sweep.

- [ ] **Step 5:** Commit: `feat(kb14): KB-23 sync + real-time webhook + full server wiring (V5 Plan A Task 6)`

---

## Verification Questions

1. Does a HALT-gate card produce a SAFETY task with 30-minute SLA? (yes / test)
2. Does the worklist sort by PAI descending when `sort_by=pai`? (yes / test)
3. Does missing PAI default to score 0 and sort last? (yes / test)
4. Does the KB-23 webhook create a task in real-time? (yes / handler)
5. Does SAFETY escalation send all 5 channels simultaneously? (yes / logic)

## Effort Estimate

| Task | Scope | Expected |
|---|---|---|
| Task 1: Models + YAML | 4 files modified/created | 1-2 hours |
| Task 2: KB-23 + KB-26 clients | 4 files, 5 tests | 2-3 hours |
| Task 3: KB-23 task factory | 2 files, 5 tests | 1-2 hours |
| Task 4: PAI worklist enricher | 3 files, 3 tests | 2-3 hours |
| Task 5: SAFETY escalation + notifications | 4 files | 3-4 hours |
| Task 6: Sync + webhook + wiring | 3 files | 2-3 hours |
| **Total** | **~20 files, ~16 tests** | **~12-18 hours** |
