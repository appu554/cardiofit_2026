# Ingestion Service + Intake-Onboarding Service + API Gateway Design

**Date:** 2026-03-21
**Status:** Draft
**Approach:** Thin Gateway Proxy (Approach A)

---

## 1. Overview

Two new Go microservices under `vaidshala/clinical-runtime-platform/services/` with FHIR-compliant APIs, Google FHIR Store as source of truth, and FastAPI gateway updates for CRUD REST routing.

| Service | Port | Purpose |
|---------|------|---------|
| Ingestion Service | 8140 | Single entry point for all external health data (7 source types) — normalize, validate, FHIR map, route to Kafka |
| Intake-Onboarding Service (M0) | 8141 | Patient enrollment, 50-slot clinical history collection, deterministic safety engine, biweekly check-ins |

**Design principles (from spec docs):**

- Ingestion: NEVER makes clinical decisions — normalizes, validates, maps, routes
- Intake M0: ZERO LLM clinical decisions — safety engine is deterministic compiled Go, <5ms
- Both: Event-sourced, FHIR R4 compliant (ABDM IG v7.0), Google FHIR Store as patient record store

---

## 2. Service Placement & Structure

### 2.1 Ingestion Service

```
vaidshala/clinical-runtime-platform/services/ingestion-service/
├── cmd/ingestion/main.go
├── internal/
│   ├── adapters/
│   │   ├── ehr/                # HL7v2 MLLP listener, FHIR REST, SFTP batch
│   │   │   ├── mllp.go        # TCP listener with MLLP framing (0x0B/0x1C+0x0D)
│   │   │   ├── fhir_rest.go   # POST /ingest/ehr/fhir
│   │   │   └── sftp.go        # 15-min polling, CSV per-hospital templates
│   │   ├── abdm/              # HIU flow (consent → decrypt → parse), HIP flow (outbound)
│   │   │   ├── hiu_handler.go
│   │   │   ├── hip_publisher.go
│   │   │   └── consent.go
│   │   ├── labs/              # Per-lab sub-adapters
│   │   │   ├── thyrocare.go
│   │   │   ├── redcliffe.go
│   │   │   ├── srl_agilus.go
│   │   │   ├── dr_lal.go
│   │   │   ├── metropolis.go
│   │   │   ├── orange_health.go
│   │   │   └── generic_csv.go
│   │   ├── patient_reported/
│   │   │   ├── app_checkin.go  # Flutter app structured JSON
│   │   │   └── whatsapp.go    # Tier-1 NLU parsed intent+entities
│   │   ├── hpi/               # M0 slot data receiver
│   │   ├── devices/           # BLE via app relay, vendor cloud APIs
│   │   └── wearables/
│   │       ├── health_connect.go
│   │       ├── ultrahuman.go  # CGM aggregation: TIR, TAR, TBR, CV, MAG
│   │       └── apple_health.go
│   ├── pipeline/
│   │   ├── receiver.go
│   │   ├── parser.go
│   │   ├── normalizer.go      # Unit conversion, code mapping, temporal alignment
│   │   ├── validator.go       # Clinical range checks, quality scoring (0.0-1.0)
│   │   ├── mapper.go          # CanonicalObservation → FHIR R4 (ABDM IG v7.0)
│   │   └── router.go          # Kafka topic selection by category + urgency
│   ├── canonical/
│   │   ├── observation.go     # CanonicalObservation struct
│   │   ├── context.go         # DeviceContext, ClinicalContext, ABDMContext
│   │   └── flags.go           # CRITICAL_VALUE, IMPLAUSIBLE, LOW_QUALITY, etc.
│   ├── fhir/
│   │   ├── observation_mapper.go
│   │   ├── diagnostic_report_mapper.go
│   │   ├── condition_mapper.go
│   │   ├── medication_mapper.go
│   │   ├── abdm_composition.go  # ABDM artifact wrappers
│   │   └── validator.go         # FHIRPath validation against IG v7.0
│   ├── coding/
│   │   ├── loinc_mapper.go
│   │   ├── snomed_mapper.go
│   │   ├── icd10_mapper.go
│   │   ├── unit_converter.go    # mmol/L→mg/dL, kPa→mmHg, °F→°C
│   │   └── lab_code_registry.go # PostgreSQL-backed per-lab mapping table
│   ├── patient/
│   │   ├── resolver.go          # Resolve patientId from ABHA/phone/MRN via KB-20
│   │   ├── abha_client.go
│   │   ├── phone_index.go
│   │   └── pending_queue.go     # 24hr hold for unresolved patients
│   ├── kafka/
│   │   ├── producer.go
│   │   ├── router.go            # Topic selection + priority partitioning
│   │   └── wal.go               # Write-Ahead Log for Kafka failover (10GB cap)
│   ├── dlq/
│   │   ├── publisher.go
│   │   ├── resolver.go
│   │   └── replay.go
│   ├── crypto/
│   │   ├── x25519.go            # ABDM X25519-XSalsa20-Poly1305
│   │   └── consent_verifier.go
│   ├── metrics/
│   │   └── collectors.go
│   └── config/
│       ├── config.go
│       ├── source_registry.go
│       └── lab_templates.go
├── configs/
│   └── lab_mappings/            # Per-lab LOINC mapping YAML
├── migrations/
├── Dockerfile
└── go.mod
```

### 2.2 Intake-Onboarding Service (M0)

```
vaidshala/clinical-runtime-platform/services/intake-onboarding-service/
├── cmd/intake/main.go
├── internal/
│   ├── enrollment/
│   │   ├── states.go            # 8 states: CREATED → ENROLLED
│   │   ├── transitions.go
│   │   └── channel_variants.go  # Corporate / Insurance / Government
│   ├── slots/
│   │   ├── table.go             # 50-slot definition across 8 domains
│   │   ├── events.go            # Event-sourced slot storage (append-only)
│   │   └── view.go              # current_slots view (latest event per slot)
│   ├── safety/
│   │   ├── engine.go            # Synchronous evaluation, <5ms, zero external deps
│   │   ├── hard_stops.go        # 11 rules: H1-H11
│   │   ├── soft_flags.go        # 8 rules: SF-01 to SF-08
│   │   └── rules_registry.go
│   ├── flow/
│   │   ├── engine.go            # Generic graph traversal engine
│   │   ├── graph.go             # Node + edge data structures
│   │   └── nodes.go             # Question nodes with extraction mode
│   ├── extraction/
│   │   ├── buttons.go           # WhatsApp interactive buttons (<50ms, 100%)
│   │   ├── regex.go             # Numeric extraction (<10ms, 95%+)
│   │   ├── nlu_client.go        # LLM for Hindi free text (200-500ms, 85-92%)
│   │   └── device.go            # Auto-populated from connected device
│   ├── whatsapp/
│   │   ├── webhook.go
│   │   ├── sender.go
│   │   └── templates.go         # Hindi/regional language message templates
│   ├── app/
│   │   ├── handler.go           # Flutter app REST — structured forms
│   │   └── form_validator.go
│   ├── asha/
│   │   ├── handler.go           # ASHA tablet adapter
│   │   ├── sync.go              # Offline → online sync
│   │   └── offline_queue.go     # Local SQLite queue
│   ├── abdm/
│   │   ├── abha_client.go       # ABHA creation/linking
│   │   └── consent_collector.go # DPDPA + ABDM consent flows
│   ├── checkin/
│   │   ├── machine.go           # M0-CI 7-state biweekly (CS1-CS7, 12 slots)
│   │   └── trajectory.go        # STABLE/FRAGILE/FAILURE/DISENGAGE signal
│   ├── review/
│   │   ├── queue.go             # Pharmacist review queue
│   │   └── reviewer.go          # Approve / clarify / escalate
│   ├── fhir/
│   │   └── generator.go         # Slot values → FHIR resources
│   ├── kafka/
│   │   ├── producer.go
│   │   └── topics.go
│   ├── session/
│   │   ├── manager.go
│   │   ├── lock.go              # Redis distributed lock per patient
│   │   ├── timeout.go           # 4hr pause, 24/48/72hr reminders, 7d abandon
│   │   └── dedup.go             # Redis messageId dedup (24hr TTL)
│   ├── metrics/
│   │   └── collectors.go
│   └── config/
│       ├── config.go
│       └── thresholds.go
├── configs/flows/
│   ├── intake_full.yaml         # Full 50-slot intake (~25 nodes)
│   ├── checkin_14day.yaml       # Biweekly check-in (7 states, 12 slots)
│   ├── meal_test.yaml           # Structured meal test (4 nodes, 6-8 slots)
│   └── re_intake_90day.yaml     # 90-day re-intake
├── migrations/
├── Dockerfile
└── go.mod
```

### 2.3 Shared FHIR Client Package

```
vaidshala/clinical-runtime-platform/pkg/
└── fhirclient/
    ├── client.go       # Google Healthcare FHIR REST client (OAuth2, retry)
    ├── retry.go        # Exponential backoff on 429/5xx (3 attempts)
    └── config.go       # GoogleFHIRConfig struct
```

Reused from KB-20's proven `fhir_client.go` pattern. Both services import `pkg/fhirclient`.

---

## 3. FHIR-Compliant API Design

### 3.1 Intake-Onboarding Service — FHIR Endpoints

**Standard FHIR CRUD (all write to Google FHIR Store):**

```
POST   /fhir/Patient                              → Create patient resource (ABDM IG v7.0)
GET    /fhir/Patient/{id}                          → Read patient
PUT    /fhir/Patient/{id}                          → Update patient
GET    /fhir/Patient?identifier=phone|{number}     → Search by phone
GET    /fhir/Patient?identifier=abha|{abhaId}      → Search by ABHA

POST   /fhir/Observation                           → Record clinical slot (FBG, BP, HbA1c, etc.)
GET    /fhir/Observation?patient={id}&code={loinc}  → Get observations by type
GET    /fhir/Observation?patient={id}&category=intake → All intake observations

POST   /fhir/Encounter                             → Create intake encounter
PUT    /fhir/Encounter/{id}                         → Update encounter status
GET    /fhir/Encounter/{id}                         → Get session state

POST   /fhir/MedicationStatement                    → Record current medications (slot 1.1)
GET    /fhir/MedicationStatement?patient={id}        → Get patient medications

GET    /fhir/DetectedIssue?patient={id}              → Get HARD_STOPs/SOFT_FLAGs

POST   /fhir/Condition                               → Record symptoms/diagnoses
GET    /fhir/Condition?patient={id}                   → Get patient conditions

POST   /fhir                                         → FHIR Transaction Bundle
```

**Custom FHIR Operations ($operation):**

```
# Enrollment workflow
POST   /fhir/Patient/$enroll                        → Create Patient + Encounter
POST   /fhir/Patient/{id}/$verify-otp               → OTP → IDENTITY_VERIFIED
POST   /fhir/Patient/{id}/$link-abha                → ABHA linking via ABDM

# Safety engine
POST   /fhir/Patient/{id}/$evaluate-safety           → Evaluate all safety rules
POST   /fhir/Encounter/{id}/$fill-slot               → Fill slot + safety engine + next question

# Pharmacist review
POST   /fhir/Encounter/{id}/$submit-review           → Submit for review
POST   /fhir/Encounter/{id}/$approve                  → Pharmacist approves → ENROLLED
POST   /fhir/Encounter/{id}/$request-clarification    → Re-open specific slots
POST   /fhir/Encounter/{id}/$escalate                 → Escalate to physician

# Biweekly check-in (M0-CI)
POST   /fhir/Patient/{id}/$checkin                    → Start biweekly check-in
POST   /fhir/Encounter/{id}/$checkin-slot              → Fill check-in slot (12 subset)

# Co-enrollee
POST   /fhir/Patient/{id}/$register-co-enrollee       → Register meal preparer
```

### 3.2 Ingestion Service — FHIR Endpoints

**FHIR-compliant inbound:**

```
POST   /fhir                                          → FHIR Transaction Bundle
POST   /fhir/Observation                              → Single observation
POST   /fhir/DiagnosticReport                         → Lab report bundle
POST   /fhir/MedicationStatement                      → Medication adherence
```

**Source-specific receivers (accept native format, output FHIR):**

```
POST   /ingest/ehr/hl7v2                              → HL7 v2 MLLP-over-HTTP
POST   /ingest/ehr/fhir                               → FHIR R4 Bundle passthrough
POST   /ingest/labs/{labId}                            → Lab proprietary format
POST   /ingest/devices                                 → Device reading (IEEE 11073 → FHIR)
POST   /ingest/wearables/{provider}                    → Wearable (Open mHealth → FHIR)
POST   /ingest/abdm/data-push                          → ABDM HIU callback (encrypted)
```

**Admin/Dashboard:**

```
GET    /fhir/OperationOutcome?category=dlq             → DLQ entries
POST   /fhir/OperationOutcome/{id}/$replay             → Replay DLQ message
GET    /$source-status                                  → Source freshness & health
```

---

## 4. API Gateway Updates (FastAPI)

Target: `backend/services/api-gateway/`

### 4.1 New SERVICE_ROUTES in proxy.py

```python
"ingestion":          { "url": INGESTION_SERVICE_URL,   "prefix": "/api/v1/ingest" }
"intake_onboarding":  { "url": INTAKE_SERVICE_URL,      "prefix": "/api/v1/intake" }
"intake_fhir":        { "url": INTAKE_SERVICE_URL,      "prefix": "/api/v1/intake/fhir" }
"ingestion_fhir":     { "url": INGESTION_SERVICE_URL,   "prefix": "/api/v1/ingest/fhir" }
```

**Routing examples:**

```
Flutter App  → Gateway /api/v1/intake/fhir/Patient/$enroll     → :8141/fhir/Patient/$enroll
Dashboard    → Gateway /api/v1/intake/fhir/Encounter?status=pending → :8141/fhir/Encounter?status=pending
Lab webhook  → Gateway /api/v1/ingest/labs/thyrocare           → :8140/ingest/labs/thyrocare
Device data  → Gateway /api/v1/ingest/fhir/Observation         → :8140/fhir/Observation
```

### 4.2 New Config in config.py

```python
INGESTION_SERVICE_URL: str = "http://localhost:8140"
INTAKE_SERVICE_URL: str = "http://localhost:8141"
```

### 4.3 RBAC Permissions in rbac.py

```python
ROUTE_PERMISSIONS = {
    # Intake/Onboarding — Patient App
    r"^/api/v1/intake/fhir/Patient/\$enroll":           { "POST": ["intake:enroll"] },
    r"^/api/v1/intake/fhir/Patient/[^/]+/\$verify":     { "POST": ["intake:enroll"] },
    r"^/api/v1/intake/fhir/Patient/[^/]+/\$checkin":     { "POST": ["intake:checkin"] },
    r"^/api/v1/intake/fhir/Encounter/[^/]+/\$fill-slot": { "POST": ["intake:write"] },
    r"^/api/v1/intake/fhir/Observation":                 { "POST": ["intake:write"], "GET": ["intake:read"] },
    r"^/api/v1/intake/fhir/Patient":                     { "GET": ["patient:read"], "POST": ["patient:write"] },

    # Intake/Onboarding — Dashboard (Pharmacist + Physician)
    r"^/api/v1/intake/fhir/Encounter/[^/]+/\$approve":   { "POST": ["intake:review"] },
    r"^/api/v1/intake/fhir/Encounter/[^/]+/\$escalate":  { "POST": ["intake:review"] },
    r"^/api/v1/intake/fhir/DetectedIssue":               { "GET": ["safety:read"] },

    # Ingestion — App (self-reports, devices)
    r"^/api/v1/ingest/fhir/Observation":        { "POST": ["ingest:write"] },
    r"^/api/v1/ingest/devices":                 { "POST": ["ingest:device"] },
    r"^/api/v1/ingest/wearables":               { "POST": ["ingest:device"] },

    # Ingestion — Dashboard (admin, monitoring)
    r"^/api/v1/ingest/fhir/OperationOutcome":   { "GET": ["ingest:admin"] },
    r"^/api/v1/ingest/\$source-status":         { "GET": ["ingest:admin"] },
    r"^/api/v1/ingest/labs":                    { "POST": ["ingest:lab"] },
    r"^/api/v1/ingest/ehr":                     { "POST": ["ingest:ehr"] },
    r"^/api/v1/ingest/abdm":                    { "POST": ["ingest:abdm"] },
}
```

### 4.4 Role-Route Restrictions in rbac.py

```python
ROLE_ROUTE_RESTRICTIONS = {
    # Intake — Pharmacist AND Physician
    r"^/api/v1/intake/fhir/Encounter/[^/]+/\$(approve|escalate|request-clarification)":
        ["pharmacist", "physician"],
    r"^/api/v1/intake/fhir/Patient/\$enroll":
        ["pharmacist", "physician", "asha"],
    r"^/api/v1/intake/fhir/DetectedIssue":
        ["pharmacist", "physician"],
    r"^/api/v1/intake/fhir/Encounter":
        ["pharmacist", "physician"],

    # Ingestion — Physician can view and submit
    r"^/api/v1/ingest/\$source-status":         ["admin", "pharmacist", "physician"],
    r"^/api/v1/ingest/fhir/OperationOutcome":   ["admin", "physician"],
    r"^/api/v1/ingest/labs":                    ["system", "physician"],
    r"^/api/v1/ingest/ehr":                     ["system", "physician"],
    r"^/api/v1/ingest/abdm":                    ["system"],
}
```

### 4.5 Role Access Matrix

| Role | Intake Access | Ingestion Access |
|------|--------------|-----------------|
| patient | Enroll self, fill slots, checkin, view own data | Submit self-reports, device readings |
| asha | Enroll patients (govt channel), fill slots | Submit ASHA-measured vitals |
| pharmacist | Review queue, approve/reject, view safety alerts, enroll | View source status, view DLQ |
| physician | Full — enroll, review, approve, escalate, safety alerts | Full — labs, EHR, source status, DLQ |
| admin | Full access | Full access including DLQ replay |
| system | N/A | EHR webhooks, ABDM callbacks, lab webhooks |

---

## 5. Google FHIR Store Integration

### 5.1 Store Configuration

```
Project:     cardiofit-905a8
Location:    asia-south1
Dataset:     clinical-synthesis-hub
Store:       fhir-store
Credentials: credentials/google-credentials.json
Auth:        OAuth2 service account (golang.org/x/oauth2/google)
```

### 5.2 Data Distribution

| Data | Stored In | Rationale |
|------|-----------|-----------|
| Patient, Observation, Encounter, MedicationStatement, Condition, DetectedIssue, DiagnosticReport | Google FHIR Store | Source of truth, FHIR searchable, ABDM shareable |
| Enrollment state machine, flow graph position | PostgreSQL | Operational state — transitions, locks, timeouts |
| Session locks, message dedup | Redis | Transient concurrency control (24hr TTL) |
| DLQ messages, lab code mappings, audit trail | PostgreSQL | Service-specific operational data |
| WAL (Kafka failover buffer) | Local disk | Durability — 10GB cap, 30s retry |

### 5.3 Write Flow — Intake (slot fill)

```
Patient fills slot (FBG = 178)
  ├→ Safety Engine evaluates (<5ms, deterministic)
  │    └→ If HARD_STOP: create DetectedIssue in FHIR Store
  ├→ Create FHIR Observation in Google FHIR Store
  │    (LOINC 1558-6, 178 mg/dL, subject=Patient/{id})
  ├→ Update Encounter status in FHIR Store
  ├→ Publish SLOT_FILLED to Kafka → KB-20/22
  └→ Update flow graph position in PostgreSQL
```

### 5.4 Write Flow — Ingestion (lab result)

```
Lab result (Thyrocare, eGFR = 42)
  ├→ Parse proprietary → CanonicalObservation
  ├→ Normalize: proprietary code → LOINC 33914-3
  ├→ Validate: range 0-200 ✓, patient exists ✓
  ├→ Map to FHIR Observation (ABDM DiagnosticReportLab)
  ├→ Create FHIR resources in Google FHIR Store
  ├→ Publish to Kafka ingestion.labs → KB-20/22/26
  └→ If critical (eGFR < 30): high-priority partition + alert
```

---

## 6. Kafka Topics & Event Flow

### 6.1 Ingestion Service Topics

| Topic | Consumers | Content |
|-------|-----------|---------|
| `ingestion.observations` | KB-20 | General observations |
| `ingestion.labs` | KB-20, KB-22 | Lab results (LOINC coded) |
| `ingestion.vitals` | KB-20, KB-26 | BP, HR — twin recompute trigger |
| `ingestion.patient-reported` | KB-20, KB-21 | App checkins, WhatsApp self-reports |
| `ingestion.medications` | KB-20 | Medication adherence |
| `ingestion.hpi` | KB-20, KB-23 | HPI slot data from M0 |
| `ingestion.device-data` | KB-20, KB-26 | Device readings |
| `ingestion.abdm-records` | KB-20 | External health records via ABDM |
| `ingestion.dlq` | DLQ Dashboard | Failed messages |

### 6.2 Intake-Onboarding Service Topics

| Topic | Consumers | Content |
|-------|-----------|---------|
| `intake.patient-lifecycle` | KB-20 | PATIENT_CREATED, PATIENT_ENROLLED |
| `intake.slot-events` | KB-20, KB-22 | Slot fills with safety result |
| `intake.safety-alerts` | KB-23, Notifications | HARD_STOP — urgent physician card |
| `intake.safety-flags` | Review Queue | SOFT_FLAG — pharmacist awareness |
| `intake.completions` | Review Queue | Ready for pharmacist review |
| `intake.checkin-events` | M4, KB-20, KB-21 | Biweekly check-in + trajectory signal |
| `intake.session-lifecycle` | Admin Dashboard | ABANDONED, PAUSED |
| `intake.lab-orders` | Lab Integration | Missing baseline labs |

### 6.3 Message Envelope

```json
{
  "eventId": "uuid",
  "eventType": "SLOT_FILLED | PATIENT_ENROLLED | LAB_RESULT | ...",
  "sourceType": "INTAKE | EHR | LAB | DEVICE | ...",
  "patientId": "uuid",
  "tenantId": "uuid",
  "timestamp": "ISO 8601 UTC",
  "fhirResourceType": "Observation | Patient | ...",
  "fhirResourceId": "FHIR Store resource ID",
  "payload": {},
  "qualityScore": 0.85,
  "flags": [],
  "traceId": "OpenTelemetry trace ID"
}
```

### 6.4 Cross-Service Flow

```
                    ┌──────────────┐
  EHR/Labs/ABDM ──→│  Ingestion   │──→ Kafka ingestion.* ──→ KB-20/22/26
  Devices/Wearables│  :8140       │──→ Google FHIR Store
                    └──────────────┘

                    ┌──────────────┐
  WhatsApp/App ───→│  Intake M0   │──→ Kafka intake.*    ──→ KB-20/21/23
  ASHA tablet      │  :8141       │──→ Google FHIR Store
                    └──────────────┘
                           │
                           │ HPI slots forwarded for FHIR mapping
                           ▼
                    ┌──────────────┐
                    │  Ingestion   │
                    │  :8140       │
                    └──────────────┘
```

---

## 7. Error Handling & Observability

### 7.1 Ingestion Error Classes

| Error Class | Action | DLQ? |
|-------------|--------|------|
| Transport (MLLP drop, SFTP unreachable) | Retry 3x with backoff | No |
| Parse (malformed HL7v2, invalid JSON) | Reject to DLQ with raw bytes | Yes |
| Normalization (unknown patient, unmapped code) | Pending queue 24hr / flag UNMAPPED | Conditional |
| Validation (out of range, missing field) | Flag IMPLAUSIBLE + publish / reject | Conditional |
| FHIR Mapping (StructureDefinition violation) | Auto-correct → fallback to canonical | No |
| Kafka Publish (broker unavailable) | Retry 3x → WAL (10GB, 30s retry) | WAL |

### 7.2 Intake Error Handling

| Scenario | Action |
|----------|--------|
| Safety engine error | NEVER swallow — fail slot fill, alert SRE |
| FHIR Store write failure | Retry 3x → hold in PostgreSQL → background sync |
| WhatsApp delivery failure | Retry → SMS fallback → pharmacist outreach 24hr |
| NLU confidence < 0.70 | Re-ask simplified → second fail → pharmacist |
| Session lock contention | Wait on Redis lock. Dedup rejects duplicate messageIds |
| Kafka publish failure | WAL pattern. Slot acknowledged to patient regardless |

### 7.3 Prometheus Metrics — Ingestion (10)

```
ingestion_messages_received_total{source_type, source_id, tenant_id}
ingestion_messages_processed_total{source_type, stage, status}
ingestion_pipeline_duration_seconds{source_type, stage}
ingestion_critical_values_total{observation_type, tenant_id}
ingestion_dlq_messages_total{error_class, source_type}
ingestion_wal_messages_pending
ingestion_patient_resolution_pending{tenant_id}
ingestion_abdm_consent_operations_total{operation, status}
ingestion_fhir_validation_failures_total{profile, violation_type}
ingestion_source_freshness_seconds{source_type, source_id}
```

### 7.4 Prometheus Metrics — Intake (10)

```
intake_enrollments_total{tenant_id, channel_type, status}
intake_slot_fills_total{slot_name, extraction_mode, confidence_tier}
intake_safety_triggers_total{rule_id, rule_type, tenant_id}
intake_session_duration_seconds{channel_type, flow_type}
intake_nlu_latency_seconds{extraction_mode, confidence_tier}
intake_pharmacist_review_queue_depth{tenant_id, risk_stratum}
intake_whatsapp_delivery_rate{message_type}
intake_offline_queue_depth
intake_session_lock_contention
intake_checkin_trajectory_total{trajectory, tenant_id}
```

### 7.5 Health Endpoints (both services)

```
GET /healthz     → Liveness
GET /readyz      → Readiness (Kafka + PostgreSQL + Redis + FHIR Store)
GET /startupz    → Startup (goroutine pools initialized)
GET /metrics     → Prometheus scrape
```

### 7.6 Deployment

```yaml
# Both services
image: Alpine-based, single Go binary (~18-20MB)
replicas: min 2, max 10 (HPA, CPU target 60%)
resources:
  requests: { cpu: 500m, memory: 256Mi }
  limits:   { cpu: 1, memory: 512Mi }

# Database isolation
ingestion_service: postgres://ingestion_user:***@postgres:5433/ingestion_service
intake_service:    postgres://intake_user:***@postgres:5433/intake_service
redis:             shared instance, DB 2 (ingestion), DB 3 (intake)
```

---

## 8. Shared Go Dependencies

| Library | Purpose |
|---------|---------|
| `github.com/samply/golang-fhir-models` | FHIR R4 Go structs |
| `github.com/segmentio/kafka-go` | Kafka producer |
| `github.com/jackc/pgx/v5` | PostgreSQL (pgxpool) |
| `github.com/redis/go-redis/v9` | Session locks, dedup, NLU cache |
| `github.com/go-playground/validator/v10` | Struct validation, clinical ranges |
| `github.com/prometheus/client_golang` | Prometheus metrics |
| `go.opentelemetry.io/otel` | Distributed tracing |
| `golang.org/x/crypto/nacl/box` | ABDM X25519 encryption |
| `golang.org/x/oauth2/google` | Google FHIR Store auth |
| `gopkg.in/yaml.v3` | Flow graph YAML loading |
| `go.uber.org/zap` | Structured logging |

---

## 9. Implementation Phases

### Phase 1: Foundation
- Shared `pkg/fhirclient` package
- Ingestion service scaffolding (cmd, config, pipeline interfaces)
- Intake service scaffolding (cmd, config, enrollment state machine)
- API gateway route + RBAC additions
- PostgreSQL migrations for both services
- Docker Compose additions

### Phase 2: Ingestion Core
- Patient self-report adapter (app checkin + WhatsApp)
- Device adapter (BLE via app relay)
- Pipeline stages: normalizer, validator, mapper, router
- Google FHIR Store writes
- Kafka publishing (ingestion.* topics)

### Phase 3: Intake Core
- Slot table (50 slots, 8 domains) + event store
- Safety engine (11 HARD_STOPs, 8 SOFT_FLAGs)
- Flow graph engine (YAML-driven)
- Flutter app handler (structured form slot filling)
- Google FHIR Store writes (Patient, Observation, Encounter, DetectedIssue)
- Kafka publishing (intake.* topics)

### Phase 4: Channels & Integration
- WhatsApp Business API adapter (intake)
- ASHA tablet adapter + offline sync
- ABDM adapter (ingestion HIU + intake ABHA linking)
- Lab adapters (Thyrocare, Redcliffe, SRL, Dr. Lal, Metropolis, Orange Health)
- EHR adapter (HL7v2 MLLP, FHIR REST, SFTP)

### Phase 5: Advanced
- Biweekly check-in (M0-CI) state machine
- Pharmacist review queue
- Wearable adapters (Health Connect, Ultrahuman CGM, HealthKit)
- DLQ management + replay
- Full observability (20 Prometheus metrics, OpenTelemetry traces, Grafana dashboards)

---

## 10. Port Summary (updated)

| Service | Port |
|---------|------|
| Ingestion Service | 8140 |
| Intake-Onboarding Service (M0) | 8141 |
| KB-20 Patient Profile | 8131 |
| KB-21 Behavioral Intelligence | 8133 |
| KB-22 HPI Engine | 8132 |
| KB-23 Decision Cards | 8134 |
| KB-25 Lifestyle Knowledge Graph | 8136 |
| KB-26 Metabolic Digital Twin | 8137 |
| API Gateway (FastAPI) | 8000 |
| Apollo Federation | 4000 |
