# Ingestion Service — E2E API Test Report

**Date**: 2026-03-23 15:45 IST
**Service**: ingestion-service (port 8140)
**Patient ID**: `77e5ad0f-b93b-47a5-9b9b-b3d405d2a5a7`
**Branch**: `feature/kb25-kb26-implementation`

## Test Environment

| Component | Status | Details |
|-----------|--------|---------|
| PostgreSQL | Connected | `localhost:5433` (Docker: kb-postgres) |
| Redis | Connected | `localhost:6380` (DB 2) |
| Kafka | Connected | `localhost:9092` (Docker: cardiofit-kafka-lite) |
| FHIR Store | Connected | GCP Healthcare API (gcloud CLI auth) |

## API Results Summary

| # | Endpoint | Method | HTTP | Latency | Result |
|---|----------|--------|------|---------|--------|
| 1 | `/healthz` | GET | 200 | 33ms | Liveness OK |
| 2 | `/readyz` | GET | 200 | 748ms | PG + Redis + FHIR all healthy |
| 3 | `/startupz` | GET | 200 | 31ms | Started OK |
| 4 | `/metrics` | GET | 200 | — | 110 Prometheus metric lines |
| 5 | `/fhir/Observation` | POST | 201 | 294ms | Glucose 142 mg/dL, quality 0.9 |
| 6 | `/fhir/Observation` | POST | 201 | 284ms | K+ 6.8 mEq/L → **CRITICAL_VALUE** |
| 7 | `/fhir/Observation` | POST | 201 | 268ms | eGFR 12 → **CRITICAL_VALUE** |
| 8 | `/fhir/Observation` | POST | 201 | 259ms | 7.0 mmol/L → 126 mg/dL (unit conversion) |
| 9 | `/devices` | POST | 201 | 871ms | BP monitor: 3/3 readings processed |
| 10 | `/devices` | POST | 201 | 421ms | Glucometer: 1/1 reading processed |
| 11 | `/wearables/health_connect` | POST | 200 | 30ms | 2 obs (HR + Steps), quality 0.85 |
| 12 | `/wearables/ultrahuman` | POST | 200 | 28ms | 12 readings → 7 CGM metrics computed |
| 13 | `/wearables/apple_health` | POST | 200 | 30ms | 2 obs (HR + SpO2), quality 0.90 |
| 14 | `/app-checkin` | POST | 201 | 905ms | 4/4 readings (split-routed to Kafka) |
| 15 | `/whatsapp` | POST | 201 | 232ms | NLU glucose, confidence 0.92 |
| 16 | `/ehr/fhir` | POST | 202 | 30ms | FHIR Bundle accepted (OperationOutcome) |
| 17 | `/labs/thyrocare` | POST | 401 | — | API key validation (expected rejection) |
| 18 | `/admin/dlq/$count` | GET | 200 | 34ms | Empty counts (no failures) |
| 19 | `/admin/dlq?status=PENDING` | GET | 200 | 36ms | 0 entries |
| 20 | `/fhir/OperationOutcome` | GET | 200 | 31ms | Empty FHIR Bundle (searchset) |
| 21 | `/admin/dlq?error_class=VALIDATION` | GET | 200 | 30ms | 0 entries |

**Result: 20/21 passed** (Thyrocare 401 is expected — no API key configured in dev)

## Kafka Message Verification

**Total messages published**: 13 across 4 topics

### Topic: `ingestion.observations` (4 messages)

| # | Event Type | Source | LOINC | Value | Quality | Flags |
|---|-----------|--------|-------|-------|---------|-------|
| 1 | OBSERVATION | EHR | 1558-6 | 142 mg/dL | 0.9 | — |
| 2 | OBSERVATION | EHR | 2823-3 | 6.8 mEq/L | 0.9 | CRITICAL_VALUE |
| 3 | OBSERVATION | EHR | 33914-3 | 12 mL/min/1.73m2 | 0.9 | CRITICAL_VALUE |
| 4 | OBSERVATION | EHR | 1558-6 | 126 mg/dL | 0.9 | — |

> Message 4 confirms unit conversion: 7.0 mmol/L input → 126 mg/dL in Kafka payload (×18.0182 factor).

### Topic: `ingestion.device-data` (4 messages)

| # | Event Type | Source | LOINC | Value | Quality | Flags |
|---|-----------|--------|-------|-------|---------|-------|
| 1 | DEVICE_READING | DEVICE | 8480-6 | 148 mmHg | 0.9 | — |
| 2 | DEVICE_READING | DEVICE | 8462-4 | 92 mmHg | 0.9 | — |
| 3 | DEVICE_READING | DEVICE | 8867-4 | 78 bpm | 0.9 | — |
| 4 | DEVICE_READING | DEVICE | 1558-6 | 156 mg/dL | 0.9 | — |

> Messages 1-3 from BP monitor (systolic, diastolic, HR). Message 4 from glucometer.

### Topic: `ingestion.vitals` (3 messages)

| # | Event Type | Source | LOINC | Value | Quality | Flags |
|---|-----------|--------|-------|-------|---------|-------|
| 1 | VITAL_SIGN | PATIENT_REPORTED | 29463-7 | 74.2 kg | 0.65 | MANUAL_ENTRY |
| 2 | VITAL_SIGN | PATIENT_REPORTED | 8480-6 | 138 mmHg | 0.65 | MANUAL_ENTRY |
| 3 | VITAL_SIGN | PATIENT_REPORTED | 8462-4 | 88 mmHg | 0.65 | MANUAL_ENTRY |

> From app check-in: weight + BP readings classified as VITALS (clinical type > source type).
> Quality 0.65 (patient-reported baseline) with MANUAL_ENTRY flag.

### Topic: `ingestion.patient-reported` (2 messages)

| # | Event Type | Source | LOINC | Value | Quality | Flags |
|---|-----------|--------|-------|-------|---------|-------|
| 1 | PATIENT_REPORT | PATIENT_REPORTED | 1558-6 | 156 mg/dL | 0.65 | MANUAL_ENTRY |
| 2 | PATIENT_REPORT | PATIENT_REPORTED | — | 165 mg/dL | 0.50 | MANUAL_ENTRY, UNMAPPED_CODE |

> Message 1: app check-in fasting glucose. Message 2: WhatsApp NLU glucose (lower quality 0.50, UNMAPPED_CODE because WhatsApp entities don't carry LOINC).

## Kafka Envelope Structure

```json
{
  "eventId": "UUID",
  "eventType": "OBSERVATION | DEVICE_READING | VITAL_SIGN | PATIENT_REPORT",
  "sourceType": "EHR | DEVICE | PATIENT_REPORTED | WEARABLE",
  "patientId": "UUID",
  "tenantId": "UUID",
  "timestamp": "ISO8601",
  "fhirResourceType": "Observation",
  "fhirResourceId": "UUID",
  "payload": {
    "loinc_code": "string",
    "observation_type": "string",
    "value": "number",
    "unit": "string"
  },
  "qualityScore": 0.0-1.0,
  "flags": ["CRITICAL_VALUE", "MANUAL_ENTRY", "UNMAPPED_CODE"]
}
```

## Topic Routing Rules

| Source Endpoint | Observation Type | Kafka Topic | Event Type |
|----------------|-----------------|-------------|------------|
| `/fhir/Observation` | GENERAL | `ingestion.observations` | OBSERVATION |
| `/devices` | DEVICE_DATA | `ingestion.device-data` | DEVICE_READING |
| `/app-checkin` (glucose) | PATIENT_REPORTED | `ingestion.patient-reported` | PATIENT_REPORT |
| `/app-checkin` (weight/BP) | VITALS | `ingestion.vitals` | VITAL_SIGN |
| `/whatsapp` | PATIENT_REPORTED | `ingestion.patient-reported` | PATIENT_REPORT |

> Clinical type takes precedence over source type: weight and BP from app check-in route to `ingestion.vitals`, not `ingestion.patient-reported`.

## Pipeline Stages Verified

1. **Adapter** — Source-specific parsing to `CanonicalObservation` (8 adapters tested)
2. **Normalizer** — Unit conversion (mmol/L → mg/dL confirmed in message #4 on observations topic)
3. **Validator** — Critical value detection (K+ > 6.0, eGFR < 15 flagged correctly)
4. **FHIR Mapper** — LOINC codes assigned, FHIR R4 resources created and written to FHIR Store
5. **Topic Router** — Correct topic selection based on observation type (13/13 messages on correct topics)
6. **Kafka Publisher** — All 13 messages delivered with correct partition key (patient UUID)

## Wearable Adapters (HTTP-only, no Kafka publish)

The wearable endpoints (Health Connect, Ultrahuman, Apple Health) return parsed observations in the HTTP response but do not currently publish to Kafka. They compute derived metrics (Ultrahuman: 7 CGM aggregations from 12 raw readings) and return them synchronously.

| Provider | Records In | Observations Out | Quality | Computed Metrics |
|----------|-----------|-----------------|---------|------------------|
| Health Connect | 2 | 2 | 0.85 | HeartRate → LOINC 8867-4, Steps → LOINC 55423-8 |
| Ultrahuman CGM | 12 | 7 | 0.60 | TIR=100%, TAR=0%, TBR=0%, CV=7.89%, MAG=61.31, MeanGlucose=99.58, GMI=5.69% |
| Apple HealthKit | 2 | 2 | 0.90 | HeartRate → LOINC 8867-4, SpO2 → LOINC 2708-6 |

## Quality Score Distribution

| Source | Quality Score | Rationale |
|--------|-------------|-----------|
| FHIR / EHR | 0.90 | Structured clinical data, high reliability |
| Device (medical) | 0.90 | Calibrated medical devices |
| Apple HealthKit | 0.90 | Consumer-grade but well-calibrated sensors |
| Health Connect | 0.85 | Android sensor aggregation, slightly lower confidence |
| Patient Reported | 0.65 | Self-reported values, manual entry bias |
| Ultrahuman CGM | 0.60 | CGM data has known calibration variance |
| WhatsApp NLU | 0.50 | NLU-parsed free text, highest uncertainty |

## Issues Found

1. **WhatsApp → FHIR Store**: Returns HTTP 400 (`missing required field "code"`) from Google Healthcare API. The WhatsApp FHIR mapper does not populate the `code` field correctly. Kafka publish still succeeds.
2. **Wearable Kafka gap**: Health Connect, Ultrahuman, and Apple Health adapters do not publish to Kafka. Only the HTTP response contains the observations.
3. **Lab webhook auth**: Cannot test full lab webhook flow without setting `LAB_API_KEY_THYROCARE` env var.

## Prerequisites for Full E2E

```bash
# 1. Start infrastructure
cd backend/shared-infrastructure/knowledge-base-services && make run-kb-docker

# 2. Run database migrations (required for DLQ)
cat migrations/001_init.sql | docker exec -i kb-postgres psql -U ingestion_user -d ingestion_service
cat migrations/002_lab_adapters.sql | docker exec -i kb-postgres psql -U ingestion_user -d ingestion_service

# 3. Build and start
cd vaidshala/clinical-runtime-platform/services/ingestion-service
go build -o ingestion-service ./cmd/ingestion
./ingestion-service
```
