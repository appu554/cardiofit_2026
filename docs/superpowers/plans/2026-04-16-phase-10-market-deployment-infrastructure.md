# Phase 10: Market Deployment Infrastructure — Gaps 9-13

**Goal:** Transform the platform from "clinically operational" (Phase 7-9 achievement) to "market-deployable" by closing the five infrastructure gaps that block Australia MHR and India ABDM market entry: FHIR outbound write-back, unified explainability chain, clinical audit event sourcing, cross-service circuit breaking, and formulary accessibility filtering.

**Pre-requisite:** Phase 9 complete. Branch at `98d885b6`. All KB services build and test green. The 20-patient clinical scorecard is 18/20.

**Strategic context:** Phase 10 is fundamentally different from Phases 7-9. Phases 7-9 asked "which patients would we get wrong?" Phase 10 asks "which markets can we enter, and what blocks entry?" The success criteria shift from clinical accuracy to compliance, resilience, and market specificity.

**Effort estimate:** 14-22 weeks total across 5 gaps if done sequentially. With 2-3 parallel workstreams, 6-10 weeks calendar time. This plan scopes Session 1 (what ships this week) and documents Sessions 2+ (what ships over the following weeks).

---

## Prioritization — Why This Order

| Priority | Gap | Rationale | Effort |
|---|---|---|---|
| **1** | **Gap 12 — Circuit Breaking** | Lowest effort (2-3 weeks), enables all other gaps. Every Gap 9/10/13 feature adds cross-service HTTP calls — without circuit breaking, each new call is a reliability risk. Foundation first. | 2-3 wk |
| **2** | **Gap 13 — Formulary Filtering** | Market-access blocker. Without it, cards recommend drugs patients can't access in their market (India NLEM, Australia PBS). KB-6 formulary service already exists with US/Medicare data — India + Australia adapters are additive. | 3-4 wk |
| **3** | **Gap 9 — FHIR Outbound** | Compliance blocker. Australia MHR and India ABDM both require clinical data write-back to the national health record. Significant partial implementation exists (FhirWriteRequest model, Module 6 FHIR output tag, KB-20 FHIR publisher). | 4-6 wk |
| **4** | **Gap 11 — Clinical Audit** | Compliance hardening. Required for regulatory audits (HIPAA, GDPR, India DPDP Act). Event outbox pattern exists but no unified audit service, no hash chaining, no immutability enforcement. | 4-5 wk |
| **5** | **Gap 10 — Explainability** | Clinical quality, not market entry. DecisionCard.ReasoningChain JSONB field exists but is not populated. Depends on KB-22 HPI engine improvements. Lower priority for initial market launch. | 3-4 wk |

---

## Session 1 Scope — Circuit Breaking (Gap 12)

Session 1 targets the highest-leverage, lowest-effort gap first: cross-service circuit breaking. After this session, every KB client HTTP call in the platform has retry logic, exponential backoff, circuit-breaker state management, and Prometheus observability.

### Existing Infrastructure (from recon)

| What exists | Where | Notes |
|---|---|---|
| HTTP clients with timeouts | KB-23 `KB20Client`, `KB26Client`, `KB26BPContextClient` | Timeout only, no retry |
| Python circuit breaker | `api-gateway/app/middleware/circuit_breaker.py` | States: closed/open/half-open. Not usable from Go. |
| Graceful degradation | KB-23 clients return (nil, error) on failure | Callers handle nil but don't retry |
| Health check endpoints | Every KB service has `/health` | Not proactively checked before requests |

### Sub-projects for Session 1

#### P10-A: Go Circuit Breaker Wrapper for http.Client

**Files:**
- Create: `kb-23-decision-cards/pkg/resilience/circuit_breaker.go`
- Create: `kb-23-decision-cards/pkg/resilience/circuit_breaker_test.go`

A zero-dependency Go circuit breaker that wraps `http.Client.Do`:
- Three states: **closed** (normal), **open** (failing, reject fast), **half-open** (probe one request)
- Configurable: `MaxFailures` (default 5), `ResetTimeout` (default 30s), `HalfOpenMaxRequests` (default 1)
- Thread-safe via `sync.Mutex`
- Prometheus counters: `circuit_breaker_state_transitions_total{service,from,to}`, `circuit_breaker_requests_total{service,state,outcome}`
- Exponential backoff with jitter on retries (configurable `MaxRetries`, default 3)

**Design:** The circuit breaker wraps `http.Client` rather than replacing it, so existing client code changes minimally — swap `c.client.Do(req)` for `c.breaker.Do(req)`.

#### P10-B: Wire Circuit Breaker into KB-23 Clients

**Files:**
- Modify: `kb-23-decision-cards/internal/services/kb20_client.go`
- Modify: `kb-23-decision-cards/internal/services/kb26_client.go`
- Modify: `kb-23-decision-cards/internal/services/kb26_bp_context_client.go`

Each client gets a `*resilience.CircuitBreaker` field, constructed in `NewKB20Client` / `NewKB26Client` with service-specific settings:
- KB-20 calls (summary-context, renal-status, intervention-timeline): `MaxRetries=2, ResetTimeout=15s` (fast, frequent, latency-sensitive)
- KB-26 calls (target-status, cgm-latest): `MaxRetries=3, ResetTimeout=30s` (slower, less frequent, higher tolerance)

#### P10-C: Wire Circuit Breaker into KB-20 Clients

**Files:**
- Modify: `kb-20-patient-profile/internal/clients/kb26_client.go`

KB-20's KB-26 client (MRI + CGM latest) gets the same circuit breaker wrapper.

#### P10-D: Prometheus Metrics + Health Check Integration

**Files:**
- Modify: `kb-23-decision-cards/internal/metrics/collector.go`
- Modify: `kb-20-patient-profile/internal/metrics/collector.go`

Circuit breaker state transitions + request outcomes as Prometheus counters so Grafana dashboards can alert on open circuits.

### Session 1 Verification Questions

1. Does a simulated KB-20 outage (5 consecutive 500s) open the circuit breaker? (yes / test)
2. Does the circuit breaker reject requests immediately when open? (yes / test)
3. Does the circuit transition to half-open after ResetTimeout and allow one probe request? (yes / test)
4. Does a successful probe close the circuit? (yes / test)
5. Does exponential backoff with jitter produce increasing delays on retries? (yes / test)
6. Are Prometheus metrics emitted on state transitions? (yes / code review)
7. Do existing KB-23 + KB-20 test suites still pass? (yes / full sweep)

---

## Session 2+ Scope — Gaps 13, 9, 11, 10

### Gap 13 — Formulary Accessibility Filtering (Sessions 2-3)

**Existing infrastructure:** KB-6 formulary service with US/Medicare Part D data. Market-configs directory with Australia + India overrides for renal thresholds. CMS Formulary ETL pipeline.

**Sub-projects:**
- **P10-E:** NLEM India adapter — CSV ingest from India Ministry of Health essential medicines list. Map drug classes to accessibility status (available / restricted / not listed). ~400 LOC.
- **P10-F:** PBS Australia adapter — API or CSV ingest from Services Australia Pharmaceutical Benefits Scheme. Map drug classes to PBS authority rules (general benefit / authority required / not PBS listed). ~400 LOC.
- **P10-G:** Market-code filtering in KB-23 card builder — before recommending a drug class in a card recommendation, check KB-6 for market accessibility. If the recommended drug is not accessible, append a note with the nearest available alternative. ~500 LOC.
- **P10-H:** Formulary-aware decision card templates — extend existing templates with `{{.FormularyNote}}` placeholder that the card builder populates from the market check. ~200 LOC.

**Dependencies:** KB-6 query endpoint must support market_code filtering. If it doesn't, add a `GET /api/v1/formulary/:drug_class?market=india` endpoint to KB-6. ~300 LOC.

### Gap 9 — FHIR Outbound Write-Back (Sessions 3-5)

**Existing infrastructure:** `FhirWriteRequest` Java model (Flink). Module 6 `FHIR_TAG` output to `fhir-writeback` Kafka topic. KB-20 `fhir_publisher.go` handling threshold crossings + stratum changes. `fhir_client.go` with Google Healthcare FHIR API integration.

**Sub-projects:**
- **P10-I:** FHIR Writer Service — new Go service (or extension of KB-20) that consumes `fhir-writeback` Kafka topic, deserialises `FhirWriteRequest`, routes to the appropriate FHIR Store, and confirms write success. Circuit breaker from P10-A applied to the FHIR Store HTTP calls. ~800 LOC.
- **P10-J:** FHIR Bundle Builder — group related writes (e.g., a patient's lab result + derived eGFR + CKD status update) into a single FHIR Transaction Bundle for atomic write. ~400 LOC.
- **P10-K:** ABDM India Gateway Adapter — India Ayushman Bharat Digital Mission gateway integration. ABDM uses a specific consent + health information flow distinct from raw FHIR write. Requires M1/M2/M3 flow implementation per ABDM spec. ~500 LOC + integration testing against ABDM sandbox.
- **P10-L:** MHR Australia Adapter — My Health Record upload via the Healthcare Identifiers Service + National Infrastructure API. Requires HPI-I (individual) and HPI-O (organisation) registration. ~500 LOC + integration testing against MHR test environment.
- **P10-M:** FHIR Outbound Audit Trail — every FHIR write produces an audit event (consumed by Gap 11's audit service). Links the outbound write to the originating DecisionCard for traceability. ~300 LOC.

**Dependencies:** ABDM gateway sandbox credentials. MHR test environment API keys. Both require organisational registration with the respective national health authorities — lead time is typically 2-4 weeks.

### Gap 11 — Clinical Audit Event Sourcing (Sessions 4-6)

**Existing infrastructure:** Event outbox pattern in KB-20. SafetyEvent audit table. Module 6 AuditRecord output. `audit-events.v1` Kafka topic.

**Sub-projects:**
- **P10-N:** Unified Audit Service — new Go service that consumes `audit-events.v1` + `envelope-events.v1` + SafetyEvent publishes, normalises them into a standard audit schema, and persists to an append-only `clinical_audit_log` table. ~600 LOC.
- **P10-O:** Hash Chain Verification — each audit entry carries a SHA-256 hash of the previous entry, creating a tamper-evident chain. A background verifier periodically checks chain integrity. ~300 LOC.
- **P10-P:** Immutability Enforcement — Postgres triggers that reject UPDATE/DELETE on the `clinical_audit_log` table. Superuser bypass only for emergency compliance actions. ~200 LOC.
- **P10-Q:** Audit Query/Export Endpoint — `GET /api/v1/audit/patient/:id?from=&to=` for compliance officers. Returns the full audit trail for a patient within a date range, including hash verification status. ~500 LOC.
- **P10-R:** Audit Schema Standardisation — define a canonical audit event shape (EventID, EventType, PatientID, ServiceSource, Timestamp, Payload, PreviousHash, Hash) that all services emit. Migrate existing SafetyEvent and EventOutbox to emit this shape alongside their existing payloads. ~400 LOC.

### Gap 10 — Unified Explainability Chain (Sessions 5-7)

**Existing infrastructure:** `DecisionCard.ReasoningChain` JSONB field (exists, unpopulated). `MCUGateRationale` string (populated). `SafetyCheckSummary` JSONB (populated).

**Sub-projects:**
- **P10-S:** KB-22 Reasoning Chain Emission — modify KB-22 HPI engine to emit structured Bayesian reasoning (prior → likelihood → posterior per differential) as part of the `HPICompleteEvent`. KB-23's `CardBuilder.Build` already has a slot for `ReasoningChain` (CTL Panel 4). ~300 LOC on the KB-22 side.
- **P10-T:** Evidence Trail Builder — new KB-23 service that assembles a complete explainability narrative from: (1) KB-22 Bayesian reasoning, (2) MCU gate evaluation rationale, (3) safety check summary, (4) confounder flag status, (5) template selection logic. Output is a human-readable + machine-parseable JSON structure. ~600 LOC.
- **P10-U:** Explainability HTTP Endpoint — `GET /api/v1/cards/:cardId/explainability` returning the full evidence trail for a specific card. Enables clinician review of "why did the system recommend this?" ~300 LOC.
- **P10-V:** Clinician Summary Auto-Generation — instead of static template fragments, generate the `ClinicianSummary` field from the evidence trail builder's output. This is the most ambitious sub-project in Gap 10 — it replaces manually-authored template text with reasoning-derived text. ~400 LOC + clinical review.

---

## Cross-Gap Dependency Map

```
Gap 12 (Circuit Breaking)  ←── foundation for all cross-service calls
    ↓
Gap 13 (Formulary)         ←── market-specific drug recommendations
    ↓
Gap 9  (FHIR Outbound)     ←── compliance gate for market entry
    ↓                           uses circuit breaker for FHIR writes
Gap 11 (Audit)             ←── compliance hardening
    ↓                           audits Gap 9 outbound writes
Gap 10 (Explainability)    ←── clinical quality
                                reasoning enters audit trail
```

Each gap enables the next. Circuit breaking is the foundation because every subsequent gap adds HTTP calls that need fault tolerance. Formulary enables market-specific cards. FHIR outbound enables write-back. Audit enables compliance proof. Explainability enables clinical transparency.

---

## Execution Timeline (estimated)

| Week | Session | Sub-projects | Gaps addressed |
|---|---|---|---|
| 1 | Session 1 | P10-A, B, C, D (circuit breaking) | Gap 12 |
| 2-3 | Session 2-3 | P10-E, F, G, H (formulary) | Gap 13 |
| 4-6 | Session 3-5 | P10-I, J, K, L, M (FHIR outbound) | Gap 9 |
| 6-8 | Session 4-6 | P10-N, O, P, Q, R (audit) | Gap 11 |
| 8-10 | Session 5-7 | P10-S, T, U, V (explainability) | Gap 10 |

**Milestones:**
- **Week 1:** Circuit breaking live → all KB clients are fault-tolerant
- **Week 3:** Formulary filtering live → cards respect market drug availability
- **Week 6:** FHIR outbound live → platform writes clinical data to MHR/ABDM
- **Week 8:** Audit service live → platform can demonstrate compliance
- **Week 10:** Explainability live → clinicians can ask "why this card?"

---

## What Phase 10 Does NOT Cover

| Item | Why deferred |
|---|---|
| HL7 v2 outbound | Legacy protocol, not required for MHR/ABDM. Phase 11 if legacy EHR integration is needed. |
| FHIR bulk data export | Not required for initial market entry. Phase 11 for analytics/population health use cases. |
| Multi-language card templates beyond EN/HI | Requires clinical translation review. Phase 11 for additional markets (Tamil, Telugu, Bengali). |
| Patient-facing mobile app integration | Different workstream — UI/UX, not infrastructure. |
| Advanced frailty scoring (Rockwood CFS, STOPP/START) | Phase 9 P9-F shipped Beers Criteria polypharmacy screen. Full geriatric assessment is Phase 11. |

---

## Effort Summary

| Sub-project | Gap | Toolset | Upper bound | Expected actual | LOC |
|---|---|---|---|---|---|
| P10-A Circuit Breaker Library | 12 | Go/pkg | 1 wk | 2-3 days | ~400 |
| P10-B KB-23 Client Wiring | 12 | Go/KB-23 | 3 days | 1-2 days | ~200 |
| P10-C KB-20 Client Wiring | 12 | Go/KB-20 | 2 days | 1 day | ~100 |
| P10-D Metrics + Health Check | 12 | Go/KB-23+KB-20 | 2 days | 1 day | ~200 |
| P10-E NLEM India Adapter | 13 | Go/KB-6 | 1 wk | 3-4 days | ~400 |
| P10-F PBS Australia Adapter | 13 | Go/KB-6 | 1 wk | 3-4 days | ~400 |
| P10-G Market-Code Filtering | 13 | Go/KB-23 | 1 wk | 2-3 days | ~500 |
| P10-H Formulary-Aware Templates | 13 | YAML+Go | 3 days | 1-2 days | ~200 |
| P10-I FHIR Writer Service | 9 | Go/new svc | 2 wk | 1 wk | ~800 |
| P10-J Bundle Builder | 9 | Go | 1 wk | 3-4 days | ~400 |
| P10-K ABDM India Adapter | 9 | Go | 2 wk | 1 wk | ~500 |
| P10-L MHR Australia Adapter | 9 | Go | 2 wk | 1 wk | ~500 |
| P10-M FHIR Outbound Audit | 9 | Go | 3 days | 1-2 days | ~300 |
| P10-N Unified Audit Service | 11 | Go/new svc | 2 wk | 1 wk | ~600 |
| P10-O Hash Chain | 11 | Go | 1 wk | 2-3 days | ~300 |
| P10-P Immutability | 11 | SQL | 3 days | 1 day | ~200 |
| P10-Q Audit Query Endpoint | 11 | Go | 1 wk | 3-4 days | ~500 |
| P10-R Audit Schema | 11 | Go/cross-svc | 1 wk | 3-4 days | ~400 |
| P10-S KB-22 Reasoning | 10 | Go/KB-22 | 1 wk | 3-4 days | ~300 |
| P10-T Evidence Trail Builder | 10 | Go/KB-23 | 2 wk | 1 wk | ~600 |
| P10-U Explainability Endpoint | 10 | Go/KB-23 | 1 wk | 2-3 days | ~300 |
| P10-V Clinician Summary Auto | 10 | Go/KB-23 | 2 wk | 1 wk | ~400 |
