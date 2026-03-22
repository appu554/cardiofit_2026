# Vaidshala Clinical Runtime — API Guide

Complete API reference for the **Ingestion Service** (`:8140`) and **Intake-Onboarding Service** (`:8141`), accessed through the **API Gateway** (`:8000`).

## Authentication

All requests through the API Gateway require a Bearer token:

```
Authorization: Bearer <jwt_token>
```

The gateway validates the token with the Auth Service, then injects these headers before forwarding:

| Header | Description |
|--------|-------------|
| `X-User-ID` | Authenticated user UUID |
| `X-User-Email` | User email |
| `X-User-Role` | Primary role (`admin`, `doctor`, `pharmacist`, `patient`) |
| `X-User-Roles` | Comma-separated role list |
| `X-User-Permissions` | Comma-separated permission list |

Admin and doctor roles bypass per-endpoint permission checks.

## Gateway Routing

| Gateway Path | Target Service | Port |
|-------------|----------------|------|
| `/api/v1/ingest/*` | Ingestion Service | 8140 |
| `/api/v1/intake/*` | Intake-Onboarding Service | 8141 |

The gateway strips the prefix before forwarding. For example:
- `POST /api/v1/ingest/fhir/Observation` → `POST /fhir/Observation` on `:8140`
- `POST /api/v1/intake/fhir/Patient/$enroll` → `POST /fhir/Patient/$enroll` on `:8141`

---

## Ingestion Service API

Base URL: `http://localhost:8000/api/v1/ingest` (gateway) or `http://localhost:8140` (direct)

### POST /fhir/Observation

Ingest a clinical observation through the pipeline (normalize → validate → FHIR map → Kafka).

**Permission**: `ingest:write`

```bash
curl -X POST http://localhost:8000/api/v1/ingest/fhir/Observation \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "550e8400-e29b-41d4-a716-446655440000",
    "tenant_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "loinc_code": "8480-6",
    "value": 128,
    "unit": "mmHg",
    "timestamp": "2026-03-22T10:30:00Z"
  }'
```

**Response** `201 Created`:
```json
{
  "status": "accepted",
  "observation_id": "7f3dc9a2-...",
  "fhir_resource_id": "Observation/abc123",
  "quality_score": 0.95,
  "flags": []
}
```

**Response** `422 Unprocessable Entity` (rejected by pipeline):
```json
{
  "error": "observation rejected -- check DLQ for details",
  "dlq_url": "/fhir/OperationOutcome?category=dlq"
}
```

---

### POST /devices

Ingest IoT or medical device data.

**Permission**: `ingest:device`

```bash
curl -X POST http://localhost:8000/api/v1/ingest/devices \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "BP-MONITOR-001",
    "patient_id": "550e8400-e29b-41d4-a716-446655440000",
    "tenant_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "readings": [
      {"type": "systolic_bp", "value": 135, "unit": "mmHg", "timestamp": "2026-03-22T10:00:00Z"},
      {"type": "diastolic_bp", "value": 85, "unit": "mmHg", "timestamp": "2026-03-22T10:00:00Z"},
      {"type": "heart_rate", "value": 72, "unit": "bpm", "timestamp": "2026-03-22T10:00:00Z"}
    ]
  }'
```

**Response** `201 Created`:
```json
{
  "status": "accepted",
  "processed": 3,
  "total": 3,
  "rejected": 0
}
```

---

### POST /wearables/:provider

Ingest wearable device data. Supported providers: `health_connect`, `ultrahuman`, `apple_health`.

**Permission**: `ingest:device`

#### Health Connect

```bash
curl -X POST http://localhost:8000/api/v1/ingest/wearables/health_connect \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "550e8400-e29b-41d4-a716-446655440000",
    "tenant_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "records": [
      {"type": "HeartRate", "value": 72, "unit": "bpm", "time": "2026-03-22T10:00:00Z"},
      {"type": "BloodPressure", "systolic": 128, "diastolic": 82, "unit": "mmHg", "time": "2026-03-22T10:00:00Z"},
      {"type": "Steps", "value": 8500, "unit": "count", "time": "2026-03-22T10:00:00Z"}
    ]
  }'
```

#### Ultrahuman CGM

```bash
curl -X POST http://localhost:8000/api/v1/ingest/wearables/ultrahuman \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "550e8400-e29b-41d4-a716-446655440000",
    "tenant_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "readings": [
      {"glucose_mg_dl": 110, "timestamp": "2026-03-22T08:00:00Z"},
      {"glucose_mg_dl": 145, "timestamp": "2026-03-22T08:15:00Z"},
      {"glucose_mg_dl": 130, "timestamp": "2026-03-22T08:30:00Z"}
    ]
  }'
```

**Response** `201 Created`:
```json
{
  "status": "accepted",
  "observations": [
    {"id": "...", "loinc": "2339-0", "type": "DEVICE_DATA", "quality": 0.65}
  ]
}
```

#### Apple HealthKit

```bash
curl -X POST http://localhost:8000/api/v1/ingest/wearables/apple_health \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "550e8400-e29b-41d4-a716-446655440000",
    "tenant_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "samples": [
      {"type": "HKQuantityTypeIdentifierHeartRate", "value": 68, "unit": "count/min", "date": "2026-03-22T10:00:00Z"},
      {"type": "HKQuantityTypeIdentifierBodyMass", "value": 165, "unit": "lb", "date": "2026-03-22T09:00:00Z"}
    ]
  }'
```

Unit conversions applied automatically: lbs→kg, °F→°C, mmol/L→mg/dL, kPa→mmHg.

---

### POST /labs/:labId

Receive lab webhook results from partner labs.

**Permission**: `ingest:lab`

Supported lab IDs: `thyrocare`, `redcliffe`, `srl_agilus`, `dr_lal`, `metropolis`, `orange_health`, `generic_csv`

```bash
curl -X POST http://localhost:8000/api/v1/ingest/labs/thyrocare \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "THY-2026-001",
    "patient_id": "550e8400-e29b-41d4-a716-446655440000",
    "results": [
      {"test_code": "HBA1C", "value": 7.2, "unit": "%", "reference_range": "4.0-5.6"},
      {"test_code": "CREATININE", "value": 1.1, "unit": "mg/dL", "reference_range": "0.7-1.3"}
    ]
  }'
```

---

### POST /ehr/fhir

FHIR passthrough from EHR systems.

**Permission**: `ingest:ehr`

```bash
curl -X POST http://localhost:8000/api/v1/ingest/ehr/fhir \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "resourceType": "Bundle",
    "type": "transaction",
    "entry": [
      {
        "resource": {
          "resourceType": "Observation",
          "code": {"coding": [{"system": "http://loinc.org", "code": "8480-6"}]},
          "valueQuantity": {"value": 130, "unit": "mmHg"}
        }
      }
    ]
  }'
```

---

### POST /app-checkin

Patient self-reported data from the Flutter mobile app.

**Permission**: `ingest:write`

```bash
curl -X POST http://localhost:8000/api/v1/ingest/app-checkin \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "550e8400-e29b-41d4-a716-446655440000",
    "tenant_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "entries": [
      {"slot": "weight_kg", "value": 78.5},
      {"slot": "symptom_score", "value": 2}
    ]
  }'
```

---

### DLQ Admin Endpoints

#### GET /admin/dlq — List DLQ Entries

**Permission**: `ingest:admin`

```bash
# List all pending entries
curl http://localhost:8000/api/v1/ingest/admin/dlq \
  -H "Authorization: Bearer $TOKEN"

# Filter by status and error class
curl "http://localhost:8000/api/v1/ingest/admin/dlq?status=PENDING&error_class=VALIDATION&source_type=DEVICE" \
  -H "Authorization: Bearer $TOKEN"
```

**Response** `200 OK`:
```json
{
  "entries": [
    {
      "id": "d4e5f6a7-...",
      "status": "PENDING",
      "error_class": "VALIDATION",
      "source_type": "DEVICE",
      "error_message": "value out of plausible range",
      "created_at": "2026-03-22T10:00:00Z"
    }
  ],
  "count": 1
}
```

#### GET /admin/dlq/:id — Get Single Entry

```bash
curl http://localhost:8000/api/v1/ingest/admin/dlq/d4e5f6a7-... \
  -H "Authorization: Bearer $TOKEN"
```

#### POST /admin/dlq/:id/$discard — Discard Entry

```bash
curl -X POST http://localhost:8000/api/v1/ingest/admin/dlq/d4e5f6a7-.../\$discard \
  -H "Authorization: Bearer $TOKEN"
```

**Response** `200 OK`:
```json
{"status": "discarded"}
```

#### GET /admin/dlq/$count — Count by Status

```bash
curl http://localhost:8000/api/v1/ingest/admin/dlq/\$count \
  -H "Authorization: Bearer $TOKEN"
```

**Response** `200 OK`:
```json
{
  "counts": {
    "PENDING": 5,
    "REPLAYED": 12,
    "DISCARDED": 3
  }
}
```

#### POST /fhir/OperationOutcome/:id/$replay — Replay DLQ Entry

```bash
curl -X POST http://localhost:8000/api/v1/ingest/fhir/OperationOutcome/d4e5f6a7-.../\$replay \
  -H "Authorization: Bearer $TOKEN"
```

---

## Intake-Onboarding Service API

Base URL: `http://localhost:8000/api/v1/intake` (gateway) or `http://localhost:8141` (direct)

### POST /fhir/Patient/$enroll

Start patient enrollment. Creates an encounter and begins the intake flow.

**Permission**: `intake:enroll`

```bash
curl -X POST http://localhost:8000/api/v1/intake/fhir/Patient/\$enroll \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "patient_id": "550e8400-e29b-41d4-a716-446655440000",
    "tenant_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "program": "CARDIAC_REHAB",
    "channel": "APP"
  }'
```

**Response** `201 Created`:
```json
{
  "encounter_id": "b2c3d4e5-...",
  "patient_id": "550e8400-...",
  "state": "INTAKE_IN_PROGRESS",
  "slots": ["demographics", "vitals", "medications", "conditions", "lifestyle"]
}
```

---

### POST /fhir/Patient/:id/$evaluate-safety

Run the 19-rule safety engine against patient data.

**Permission**: `intake:enroll`

```bash
curl -X POST http://localhost:8000/api/v1/intake/fhir/Patient/550e8400-...../\$evaluate-safety \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "encounter_id": "b2c3d4e5-...",
    "egfr": 28,
    "age": 78,
    "medications": ["metformin", "lisinopril", "amlodipine", "atorvastatin", "aspirin"],
    "conditions": ["CKD_G3b", "HTN", "T2DM"]
  }'
```

**Response** `200 OK`:
```json
{
  "hard_stops": [
    {"rule": "EGFR_BELOW_30", "message": "eGFR < 30 requires specialist review"}
  ],
  "soft_flags": [
    {"rule": "POLYPHARMACY", "message": "5 or more concurrent medications"},
    {"rule": "ELDERLY", "message": "Age > 75 years"}
  ],
  "verdict": "BLOCKED",
  "risk_stratum": "HIGH"
}
```

---

### POST /fhir/Encounter/:id/$fill-slot

Fill an intake slot with collected data.

**Permission**: `intake:write`

```bash
curl -X POST http://localhost:8000/api/v1/intake/fhir/Encounter/b2c3d4e5-.../\$fill-slot \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "slot_name": "vitals",
    "data": {
      "systolic_bp": 138,
      "diastolic_bp": 88,
      "heart_rate": 76,
      "weight_kg": 82.5
    },
    "extraction_mode": "PATIENT_REPORTED"
  }'
```

**Response** `200 OK`:
```json
{
  "encounter_id": "b2c3d4e5-...",
  "slot_name": "vitals",
  "state": "INTAKE_IN_PROGRESS",
  "slots_filled": 3,
  "slots_total": 5
}
```

---

### POST /fhir/Encounter/:encounterID/$submit-review

Submit a completed intake for pharmacist review. Requires enrollment in `INTAKE_COMPLETED` state.

**Permission**: `intake:write`

```bash
curl -X POST http://localhost:8000/api/v1/intake/fhir/Encounter/b2c3d4e5-.../\$submit-review \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "hard_stop_count": 0,
    "soft_flag_count": 2,
    "age": 68,
    "med_count": 4,
    "egfr_value": 55.0
  }'
```

**Response** `201 Created`:
```json
{
  "id": "c3d4e5f6-...",
  "encounter_id": "b2c3d4e5-...",
  "risk_stratum": "MEDIUM",
  "status": "PENDING",
  "created_at": "2026-03-22T10:00:00Z"
}
```

---

### POST /fhir/ReviewEntry/:entryID/$approve

Approve a review entry. Transitions enrollment to `ENROLLED`.

**Permission**: `intake:review`

```bash
curl -X POST http://localhost:8000/api/v1/intake/fhir/ReviewEntry/c3d4e5f6-.../\$approve \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-User-ID: pharmacist-uuid-here"
```

**Response** `200 OK`:
```json
{"status": "approved"}
```

---

### POST /fhir/ReviewEntry/:entryID/$request-clarification

Request clarification on a review entry. Reverts enrollment to `INTAKE_IN_PROGRESS`.

**Permission**: `intake:review`

```bash
curl -X POST http://localhost:8000/api/v1/intake/fhir/ReviewEntry/c3d4e5f6-.../\$request-clarification \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-User-ID: pharmacist-uuid-here" \
  -H "Content-Type: application/json" \
  -d '{
    "slot_names": ["medications", "conditions"],
    "notes": "Please verify current medication list — possible interaction with new prescription"
  }'
```

**Response** `200 OK`:
```json
{"status": "clarification_requested"}
```

---

### POST /fhir/ReviewEntry/:entryID/$escalate

Escalate a review entry to a senior pharmacist.

**Permission**: `intake:review`

```bash
curl -X POST http://localhost:8000/api/v1/intake/fhir/ReviewEntry/c3d4e5f6-.../\$escalate \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-User-ID: pharmacist-uuid-here" \
  -H "Content-Type: application/json" \
  -d '{
    "reason": "Complex polypharmacy with CKD",
    "notes": "Patient on 7 medications with declining eGFR trend — needs senior review"
  }'
```

**Response** `200 OK`:
```json
{"status": "escalated"}
```

---

### POST /fhir/Patient/:id/$checkin

Start a new biweekly check-in session.

**Permission**: `intake:checkin`

```bash
curl -X POST http://localhost:8000/api/v1/intake/fhir/Patient/550e8400-.../\$checkin \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "cycle_number": 1
  }'
```

**Response** `201 Created`:
```json
{
  "session_id": "e5f6a7b8-...",
  "state": "CS3_COLLECTING",
  "slots_total": 12,
  "slots_filled": 0,
  "slots": [
    {"name": "weight_kg", "domain": "vitals", "required": true},
    {"name": "systolic_bp", "domain": "vitals", "required": true},
    {"name": "diastolic_bp", "domain": "vitals", "required": true},
    {"name": "heart_rate", "domain": "vitals", "required": true},
    {"name": "fasting_glucose", "domain": "labs", "required": true},
    {"name": "hba1c", "domain": "labs", "required": false},
    {"name": "creatinine", "domain": "labs", "required": false},
    {"name": "potassium", "domain": "labs", "required": false},
    {"name": "symptom_score", "domain": "symptoms", "required": true},
    {"name": "med_adherence_pct", "domain": "adherence", "required": true},
    {"name": "exercise_minutes", "domain": "lifestyle", "required": true},
    {"name": "sleep_hours", "domain": "lifestyle", "required": true}
  ]
}
```

---

### POST /fhir/CheckinSession/:id/$checkin-slot

Fill a single check-in slot value.

**Permission**: `intake:checkin`

```bash
curl -X POST http://localhost:8000/api/v1/intake/fhir/CheckinSession/e5f6a7b8-.../\$checkin-slot \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "slot_name": "weight_kg",
    "value": 78.2,
    "extraction_mode": "PATIENT_REPORTED"
  }'
```

**Response** `200 OK`:
```json
{
  "session_id": "e5f6a7b8-...",
  "slot_name": "weight_kg",
  "slots_filled": 1
}
```

---

### WhatsApp Webhook

#### GET /webhook/whatsapp — Verification

```bash
curl "http://localhost:8000/api/v1/intake/webhook/whatsapp?hub.mode=subscribe&hub.verify_token=cardiofit-intake-verify&hub.challenge=CHALLENGE_STRING"
```

**Response**: Returns the `hub.challenge` string (plain text).

#### POST /webhook/whatsapp — Incoming Messages

Receives WhatsApp Business API webhook events. Signature verification via `X-Hub-Signature-256` header using `WHATSAPP_APP_SECRET`.

---

### ASHA Tablet Channel

#### POST /channel/asha/submit — Batch Submit

```bash
curl -X POST http://localhost:8000/api/v1/intake/channel/asha/submit \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "ASHA-TAB-042",
    "entries": [
      {"patient_id": "...", "slot_name": "vitals", "data": {"systolic_bp": 140}},
      {"patient_id": "...", "slot_name": "vitals", "data": {"weight_kg": 72}}
    ]
  }'
```

#### GET /channel/asha/sync/:deviceId — Sync Status

```bash
curl http://localhost:8000/api/v1/intake/channel/asha/sync/ASHA-TAB-042 \
  -H "Authorization: Bearer $TOKEN"
```

---

## Health Checks

Both services expose identical infrastructure probes:

```bash
# Ingestion Service
curl http://localhost:8140/healthz    # → 200 {"status": "ok"}
curl http://localhost:8140/readyz     # → 200 {"status": "ready"}

# Intake Service
curl http://localhost:8141/healthz    # → 200 {"status": "ok"}
curl http://localhost:8141/readyz     # → 200 {"status": "ready"}
```

## Error Responses

All endpoints follow a consistent error format:

```json
{
  "error": "human-readable error message"
}
```

| Status | Meaning |
|--------|---------|
| `400` | Bad request (malformed JSON, invalid UUID, missing required fields) |
| `401` | Authentication required (gateway only) |
| `403` | Insufficient permissions (gateway only) |
| `404` | Resource not found |
| `409` | State conflict (e.g., enrollment not in expected state) |
| `422` | Observation rejected by pipeline |
| `500` | Internal server error |
| `501` | Endpoint not yet implemented (stub) |
| `502` | Gateway cannot reach target service |
