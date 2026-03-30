# Flink Modules 1-4 E2E Deterioration Testing Design

**Date:** 2026-03-30
**Status:** Approved
**Scope:** Python E2E script that fetches real FHIR patient data from intake service, overlays 5 deterioration scenarios in-flight, publishes to Kafka, and verifies Flink Modules 1→1b→2→3→4 output

---

## 1. Overview

Build a Python E2E test script (`flink_e2e_deterioration.py`) that:
1. Fetches real patient data from GCP FHIR store (`cardiofit-fhir-r4`) — patients created by intake-onboarding-service
2. Assigns 5 patients to deterioration scenarios, keeps 3 as healthy controls
3. Overlays deteriorating vitals/labs/meds on real baselines (T0→T1→T2) in-flight — FHIR store is NOT mutated
4. Publishes formatted RawEvent JSON to Kafka input topics
5. Verifies output from all 4 intermediate topics (Module 1→2→3→4)
6. Asserts: deteriorating patients trigger correct CEP patterns; healthy controls trigger zero patterns

**Flink pipeline runs with FHIR disabled** — Module 2 degrades gracefully using stream-only aggregation. No Neo4j, no Elasticsearch.

---

## 2. Infrastructure

### What Exists (No Changes)
- **Kafka:** `docker-compose.hpi-lite.yml` — Zookeeper + Kafka on localhost:9092
- **Kafka topics:** Created by `create-kafka-topics.sh` + `create-ingestion-topics.sh`

### What Gets Updated
- **Flink:** `docker-compose.e2e-flink.yml` — Updated to:
  - Submit all 5 modules (1, 1b, 2, 3, 4) instead of current 3
  - Remove FHIR credentials mount from Flink containers
  - Set `GOOGLE_CLOUD_CREDENTIALS_PATH=/dev/null` to force graceful degradation
  - Keep existing memory/parallelism settings (JobManager 1.2GB, TaskManager 2.5GB)

### What's New
- **Script:** `backend/shared-infrastructure/flink-processing/scripts/flink_e2e_deterioration.py`

---

## 3. Data Flow

```
GCP FHIR Store (cardiofit-fhir-r4)
    ↓ Python script fetches Observations, Conditions, MedicationRequests
    ↓
Python E2E Script
    ├── 5 patients: overlay deterioration (T0→T1→T2, 30s apart)
    ├── 3 patients: pass real data unchanged (healthy controls)
    ↓
Kafka Input Topics (localhost:9092)
    ├── vital-signs-events
    ├── lab-result-events
    └── medication-events
    ↓
Flink Pipeline (FHIR disabled, Neo4j disabled)
    ├── Module 1:  Validate + canonicalize → enriched-patient-events-v1
    ├── Module 1b: (ingestion.* topics — not used in this test)
    ├── Module 2:  Stream aggregation + risk indicators → enriched-patient-events-v1
    ├── Module 3:  CDS scoring + protocols → comprehensive-cds-events.v1
    └── Module 4:  CEP pattern detection → clinical-patterns.v1
    ↓
Python Verifier (consumes from 4 output topics)
    ├── Assert correct patterns for 5 deteriorating patients
    ├── Assert zero patterns for 3 healthy controls
    └── Generate JSON report
```

---

## 4. Patient Assignment

| Patient | Scenario | Expected Pattern | Expected Severity |
|---------|----------|-----------------|-------------------|
| P1 (1st from FHIR) | Sepsis progression | SEPSIS | CRITICAL |
| P2 (2nd) | AKI | AKI | HIGH |
| P3 (3rd) | Rapid deterioration | RAPID_DETERIORATION | CRITICAL |
| P4 (4th) | Drug-lab interaction | DRUG_LAB_INTERACTION | HIGH |
| P5 (5th) | Cardiac decompensation | RAPID_DETERIORATION | HIGH |
| P6 (6th) | Healthy control | NONE | — |
| P7 (7th) | Healthy control | NONE | — |
| P8 (8th) | Healthy control | NONE | — |

Patients are assigned by order of discovery from FHIR store. If fewer than 8 patients exist, script exits with error.

---

## 5. Deterioration Scenarios

### 5.1 Sepsis Progression (P1)

| Timepoint | HR | SBP | Temp | RR | SpO2 | Lactate | WBC | Procalcitonin |
|-----------|----|-----|------|----|------|---------|-----|---------------|
| T0 (baseline) | 78 | 128 | 37.0 | 16 | 97% | 1.0 | 8.0K | 0.1 |
| T1 (+30s) | 105 | 95 | 38.5 | 22 | 94% | 2.5 | 14K | 0.8 |
| T2 (+60s) | 130 | 78 | 39.5 | 28 | 86% | 5.2 | 22K | 4.5 |

**LOINC codes:** HR 8867-4, SBP 8480-6, Temp 8310-5, RR 9279-1, SpO2 2708-6, Lactate 2524-7, WBC 6690-2, Procalcitonin 33959-8

**CEP trigger:** lactate ≥ 2.0 + fever ≥ 38.3 + tachycardia (HR > 100) + hypotension (SBP < 90)

### 5.2 AKI (P2)

| Timepoint | Creatinine | eGFR | BUN | Potassium | SBP | HR |
|-----------|-----------|------|-----|-----------|-----|-----|
| T0 | 1.0 | 90 | 15 | 4.0 | 130 | 75 |
| T1 (+30s) | 1.8 | 45 | 28 | 4.8 | 125 | 82 |
| T2 (+60s) | 3.2 | 18 | 45 | 6.2 | 110 | 95 |

**LOINC codes:** Creatinine 2160-0, eGFR 33914-3, BUN 3094-0, Potassium 2823-3

**CEP trigger:** Creatinine rising >1.5x baseline + eGFR < 30 (KDIGO Stage 2-3)

### 5.3 Rapid Deterioration (P3)

| Timepoint | HR | SBP | RR | SpO2 | Temp | NEWS2 (calculated) |
|-----------|----|-----|----|------|------|-----|
| T0 | 80 | 125 | 16 | 97% | 37.0 | ~2 |
| T1 (+30s) | 110 | 100 | 24 | 92% | 38.2 | ~6 |
| T2 (+60s) | 135 | 82 | 32 | 85% | 39.0 | ~10 |

**CEP trigger:** NEWS2 ≥ 10 OR multi-vital breach (3+ parameters in critical range)

### 5.4 Drug-Lab Interaction (P4)

| Timepoint | Medication | INR | Hgb | Platelets | SBP | HR |
|-----------|-----------|-----|-----|-----------|-----|-----|
| T0 | Warfarin 5mg | 2.5 | 13.0 | 250K | 128 | 72 |
| T1 (+30s) | Warfarin 5mg | 4.0 | 11.5 | 180K | 122 | 78 |
| T2 (+60s) | Warfarin 5mg | 6.0 | 9.5 | 120K | 115 | 88 |

**LOINC codes:** INR 6301-6, Hgb 718-7, Platelets 777-3

**CEP trigger:** Drug (warfarin) + INR > 5.0 (supratherapeutic anticoagulation)

### 5.5 Cardiac Decompensation (P5)

| Timepoint | SBP | DBP | HR | BNP | Weight | SpO2 | RR |
|-----------|-----|-----|----|-----|--------|------|----|
| T0 | 135 | 85 | 72 | 150 | 78.0 | 97% | 16 |
| T1 (+30s) | 155 | 95 | 90 | 600 | 80.0 | 93% | 22 |
| T2 (+60s) | 100 | 55 | 115 | 1200 | 83.0 | 88% | 28 |

**LOINC codes:** BNP 30934-4, Weight 29463-7

**CEP trigger:** BP variability (SBP swing >50 mmHg) + BNP > 1000 + SpO2 declining

### 5.6 Healthy Controls (P6, P7, P8)

Real FHIR observations passed through unmodified. No value overlays. Expected: events flow through Modules 1-3 normally, Module 4 emits zero pattern events.

---

## 6. Script Architecture

```
flink_e2e_deterioration.py
│
├── FHIRFetcher
│   ├── authenticate(credentials_path) → OAuth2 token
│   ├── list_patients() → List[dict] (FHIR Patient resources)
│   ├── get_observations(patient_id) → List[dict] (all Observations)
│   ├── get_conditions(patient_id) → List[dict]
│   └── get_medication_requests(patient_id) → List[dict]
│
├── DeteriorationEngine
│   ├── assign_scenarios(patients) → Dict[patient_id, scenario|control]
│   ├── SepsisScenario.generate(patient, baseline) → List[RawEvent] (T0,T1,T2)
│   ├── AKIScenario.generate(patient, baseline) → List[RawEvent]
│   ├── RapidDeteriorationScenario.generate(patient, baseline) → List[RawEvent]
│   ├── DrugLabScenario.generate(patient, baseline) → List[RawEvent]
│   ├── CardiacDecompScenario.generate(patient, baseline) → List[RawEvent]
│   └── HealthyControlScenario.generate(patient, observations) → List[RawEvent]
│
├── KafkaPublisher
│   ├── connect(bootstrap_servers)
│   ├── publish_timepoint(events, topic) → send batch
│   └── run_timeline(all_patients) → T0, sleep 30s, T1, sleep 30s, T2
│
├── PipelineVerifier
│   ├── consume_topic(topic, timeout, run_id) → List[dict]
│   ├── verify_module1(events) → per-patient canonicalization check
│   ├── verify_module2(events) → risk_indicators populated for deteriorating
│   ├── verify_module3(events) → CDS protocols for deteriorating
│   ├── verify_module4(events) → pattern_type + severity assertions
│   └── verify_controls(events) → zero patterns for P6/P7/P8
│
└── ReportGenerator
    ├── build_summary(results) → dict
    ├── print_console(summary) → formatted table
    └── save_json(summary, path) → test-data/e2e-deterioration-{ts}.json
```

### CLI Interface

```bash
# Full run (all 8 patients, all scenarios)
python3 scripts/flink_e2e_deterioration.py

# Dry run (show events without sending)
python3 scripts/flink_e2e_deterioration.py --dry-run

# Single scenario
python3 scripts/flink_e2e_deterioration.py --scenario sepsis

# Verify only (check output topics from prior run)
python3 scripts/flink_e2e_deterioration.py --verify-only

# Custom Kafka bootstrap
python3 scripts/flink_e2e_deterioration.py --kafka localhost:9092

# Custom credentials path
python3 scripts/flink_e2e_deterioration.py --credentials path/to/creds.json
```

### Dependencies
- `google-auth` — GCP OAuth2 authentication
- `requests` — FHIR REST API calls
- `confluent-kafka` or `kafka-python` — Kafka producer/consumer
- Standard library: `json`, `uuid`, `time`, `argparse`, `datetime`

---

## 7. RawEvent Format (Kafka Message Schema)

Module 1 expects this JSON structure on input topics:

```json
{
  "id": "e2e-det-{run_id}-{patient}-{type}-{seq}",
  "source": "e2e-deterioration-test",
  "type": "vital-signs|lab-result|medication-administration",
  "patient_id": "<real FHIR patient UUID>",
  "encounter_id": "<real FHIR encounter UUID or generated>",
  "event_time": "<epoch_ms>",
  "received_time": "<epoch_ms>",
  "payload": {
    "heartRate": 130,
    "systolicBP": 78,
    "loinc_code": "8867-4",
    "value": 130,
    "unit": "bpm"
  },
  "metadata": {
    "source": "e2e-deterioration-test",
    "location": "CARDIOLOGY_ICU",
    "device_id": "FHIR",
    "loinc_code": "8867-4"
  },
  "correlation_id": "<uuid>",
  "version": "1.0"
}
```

Kafka key: `patient_id` (ensures per-patient ordering).

---

## 8. Verification Assertions

### Per Deteriorating Patient (P1-P5)

| Module | Topic | Assertion |
|--------|-------|-----------|
| M1 | `enriched-patient-events-v1` | Events present with correct `eventType` (VITAL_SIGN, LAB_RESULT, MEDICATION) |
| M2 | `enriched-patient-events-v1` | `risk_indicators` show escalating flags at T1/T2 (e.g., `tachycardia: true`) |
| M3 | `comprehensive-cds-events.v1` | CDS events with `applicable_protocols` populated |
| M4 | `clinical-patterns.v1` | `PatternEvent` with correct `pattern_type` and `severity` matching scenario |

### Per Healthy Control (P6-P8)

| Module | Topic | Assertion |
|--------|-------|-----------|
| M1-M3 | respective topics | Events flow through normally |
| M4 | `clinical-patterns.v1` | Zero `PatternEvent` records for this `patient_id` |

### Global Assertions
- All 8 patients appear in Module 1 output
- No DLQ messages for valid events
- Pattern detection latency < 30s after T2 publish
- Run tagged with `run_id` for filtering

---

## 9. Docker Compose Changes

### `docker-compose.e2e-flink.yml` Updates

1. **Job submitter** — add Module 3 (CDS) and Module 4 (Pattern Detection) submissions:
   ```yaml
   command: >
     /opt/flink/bin/flink run -d /opt/flink/jobs/flink-processing.jar --module ingestion &&
     /opt/flink/bin/flink run -d /opt/flink/jobs/flink-processing.jar --module ingestion-canonicalizer &&
     /opt/flink/bin/flink run -d /opt/flink/jobs/flink-processing.jar --module context-assembly &&
     /opt/flink/bin/flink run -d /opt/flink/jobs/flink-processing.jar --module comprehensive-cds &&
     /opt/flink/bin/flink run -d /opt/flink/jobs/flink-processing.jar --module pattern-detection
   ```

2. **FHIR disabled** — set env on jobmanager and taskmanager:
   ```yaml
   GOOGLE_CLOUD_CREDENTIALS_PATH: /dev/null
   ```

3. **No other changes** — memory, parallelism, checkpointing stay as-is

---

## 10. Output Report

Saved to `test-data/e2e-deterioration-{timestamp}.json`:

```json
{
  "run_id": "e2e-det-1774850000",
  "timestamp": "2026-03-30T15:00:00Z",
  "patients_total": 8,
  "patients_deteriorating": 5,
  "patients_control": 3,
  "results": [
    {
      "patient_id": "uuid-1",
      "scenario": "sepsis",
      "events_sent": 24,
      "module1_events": 24,
      "module2_risk_indicators": {"tachycardia": true, "hypotension": true, "fever": true},
      "module3_cds_events": 3,
      "module4_patterns": [{"type": "SEPSIS", "severity": "CRITICAL", "confidence": 0.92}],
      "latency_ms": 12500,
      "pass": true
    },
    ...
    {
      "patient_id": "uuid-6",
      "scenario": "healthy_control",
      "events_sent": 6,
      "module4_patterns": [],
      "pass": true
    }
  ],
  "summary": {
    "total_pass": 8,
    "total_fail": 0,
    "deterioration_detected": 5,
    "false_positives": 0,
    "avg_latency_ms": 14200
  }
}
```

Console output: colored table with pass/fail per patient per module.
