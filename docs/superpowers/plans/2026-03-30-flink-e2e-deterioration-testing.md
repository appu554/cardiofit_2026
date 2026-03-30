# Flink E2E Deterioration Testing Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Python E2E script that fetches real FHIR patient data, overlays 5 deterioration scenarios in-flight, publishes to Kafka, and verifies Flink Modules 1→1b→2→3→4 output — with 3 healthy controls asserting zero false-positive patterns.

**Architecture:** Single Python script (`flink_e2e_deterioration.py`) with 4 components: FHIRFetcher (GCP FHIR store), DeteriorationEngine (5 scenario overlays), KafkaPublisher (docker exec kafka-console-producer), PipelineVerifier (consume + assert). Flink runs with FHIR disabled — Module 2 degrades gracefully to stream-only aggregation.

**Tech Stack:** Python 3.11+, google-auth, requests, subprocess (kafka-console-producer/consumer via docker exec), json, argparse

**Spec:** `docs/superpowers/specs/2026-03-30-flink-e2e-deterioration-testing-design.md`

---

## File Structure

| Action | File | Responsibility |
|--------|------|---------------|
| Create | `backend/shared-infrastructure/flink-processing/scripts/flink_e2e_deterioration.py` | Main E2E script with all 4 components |
| Modify | `backend/shared-infrastructure/flink-processing/docker-compose.e2e-flink.yml` | Add Module 3 + 4 submission, disable FHIR credentials |

---

### Task 1: Update docker-compose to submit all 5 modules with FHIR disabled

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/docker-compose.e2e-flink.yml:128-190` (flink-submitter service)

- [ ] **Step 1: Add Module 3 and Module 4 job submissions**

In `docker-compose.e2e-flink.yml`, replace the `flink-submitter` service `command` block. The current command submits 3 modules. Add `comprehensive-cds` and `pattern-detection`:

```yaml
  flink-submitter:
    image: flink:2.1-java17
    container_name: cardiofit-flink-submitter
    depends_on:
      flink-taskmanager:
        condition: service_started
    environment:
      KAFKA_BOOTSTRAP_SERVERS: "kafka-lite:29092"
      DOCKER_CONTAINER: "true"
      USE_GOOGLE_HEALTHCARE_API: "false"
      GOOGLE_APPLICATION_CREDENTIALS: "/dev/null"
    entrypoint: /bin/bash
    command:
      - -c
      - |
        echo "Waiting 20s for TaskManager to register slots..."
        sleep 20

        JAR=/opt/flink/usrlib/flink-ehr-intelligence-1.0.0.jar
        if [ ! -f "$JAR" ]; then
          echo "ERROR: JAR not found at $$JAR"
          echo "Run: mvn clean package -DskipTests -q"
          exit 1
        fi

        submit() {
          local name="$$1"
          local job_arg="$$2"
          echo ""
          echo "━━━ Submitting $$name ━━━"
          /opt/flink/bin/flink run -d -m flink-jobmanager:8081 \
            -c com.cardiofit.flink.FlinkJobOrchestrator \
            "$$JAR" "$$job_arg" development
          local rc=$$?
          if [ $$rc -eq 0 ]; then
            echo "  ✓ $$name submitted"
          else
            echo "  ✗ $$name FAILED (exit code $$rc)"
          fi
          sleep 5
        }

        submit "Module 1:  Ingestion & Gateway"      "ingestion-only"
        submit "Module 1b: Ingestion Canonicalizer"   "module1b-canonicalizer"
        submit "Module 2:  Context Assembly"          "context-assembly"
        submit "Module 3:  Comprehensive CDS"         "comprehensive-cds"
        submit "Module 4:  Pattern Detection"         "pattern-detection"

        echo ""
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo "All 5 modules submitted. Check Flink UI: http://localhost:8181"
        echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

        sleep 3
        /opt/flink/bin/flink list -m flink-jobmanager:8081 || true
    volumes:
      - ./target:/opt/flink/usrlib:ro
    networks:
      - cardiofit-lite
    restart: "no"
```

- [ ] **Step 2: Disable FHIR on jobmanager and taskmanager**

In the `flink-jobmanager` service environment section, change:
```yaml
    environment:
      KAFKA_BOOTSTRAP_SERVERS: "kafka-lite:29092"
      USE_GOOGLE_HEALTHCARE_API: "false"
      GOOGLE_CLOUD_CREDENTIALS_PATH: "/dev/null"
      GOOGLE_APPLICATION_CREDENTIALS: "/dev/null"
      DOCKER_CONTAINER: "true"
```

Apply the same environment changes to `flink-taskmanager`.

Remove `- ./credentials:/opt/flink/credentials:ro` from both `flink-jobmanager` and `flink-taskmanager` volumes.

- [ ] **Step 3: Verify docker-compose is valid**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && docker compose -f docker-compose.e2e-flink.yml config --quiet`
Expected: No errors (exit code 0)

- [ ] **Step 4: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/docker-compose.e2e-flink.yml
git commit -m "feat(flink): submit all 5 modules in E2E compose, disable FHIR"
```

---

### Task 2: Create the FHIR fetcher and configuration module

**Files:**
- Create: `backend/shared-infrastructure/flink-processing/scripts/flink_e2e_deterioration.py`

This task creates the script skeleton with FHIR fetching. We reuse the same patterns from the existing `flink_e2e_real_data.py`.

- [ ] **Step 1: Create the script with imports, config, and FHIR fetcher**

```python
#!/usr/bin/env python3
"""
Flink E2E Deterioration Test — Modules 1→1b→2→3→4

Fetches real patient data from GCP FHIR store (intake-created patients),
overlays 5 deterioration scenarios in-flight, publishes to Kafka,
verifies pattern detection output across all modules.

Usage:
    python3 scripts/flink_e2e_deterioration.py                    # Full run
    python3 scripts/flink_e2e_deterioration.py --dry-run           # Show events, don't send
    python3 scripts/flink_e2e_deterioration.py --scenario sepsis   # Single scenario
    python3 scripts/flink_e2e_deterioration.py --verify-only       # Check output topics only
"""

import argparse
import json
import os
import subprocess
import sys
import time
import uuid
from datetime import datetime, timezone

# ---------------------------------------------------------------------------
# GCP FHIR Store configuration (same store the intake service writes to)
# ---------------------------------------------------------------------------
PROJECT_ID = "project-2bbef9ac-174b-4b59-8fe"
LOCATION = "asia-south1"
DATASET_ID = "vaidshala-clinical"
FHIR_STORE_ID = "cardiofit-fhir-r4"
FHIR_BASE_URL = (
    f"https://healthcare.googleapis.com/v1/projects/{PROJECT_ID}"
    f"/locations/{LOCATION}/datasets/{DATASET_ID}/fhirStores/{FHIR_STORE_ID}/fhir"
)

CREDENTIALS_PATH = os.path.join(
    os.path.dirname(__file__), "..", "..", "..", "..",
    "backend", "services", "patient-service", "credentials", "google-credentials.json",
)

# ---------------------------------------------------------------------------
# Kafka configuration
# ---------------------------------------------------------------------------
KAFKA_CONTAINER = "cardiofit-kafka-lite"
KAFKA_BOOTSTRAP = "kafka-lite:29092"

TOPIC_VITAL_SIGNS = "vital-signs-events-v1"
TOPIC_LAB_RESULTS = "lab-result-events-v1"
TOPIC_MEDICATIONS = "medication-events-v1"

TOPIC_ENRICHED = "enriched-patient-events-v1"
TOPIC_CONTEXT = "patient-context-snapshots-v1"
TOPIC_CDS = "comprehensive-cds-events.v1"
TOPIC_PATTERNS = "clinical-patterns.v1"

# ---------------------------------------------------------------------------
# LOINC mappings (must match Module4SemanticConverter expectations)
# ---------------------------------------------------------------------------
VITAL_LOINC = {
    "8867-4": "heartRate",
    "8480-6": "systolicBP",
    "8462-4": "diastolicBP",
    "8310-5": "temperature",
    "9279-1": "respiratoryRate",
    "2708-6": "oxygenSaturation",
    "29463-7": "weight",
}

LAB_LOINC = {
    "2524-7": "lactate",
    "6690-2": "wbc",
    "33959-8": "procalcitonin",
    "2160-0": "creatinine",
    "3094-0": "bun",
    "2823-3": "potassium",
    "48642-3": "egfr",
    "777-3": "platelets",
    "718-7": "hemoglobin",
    "34714-6": "inr",
    "30934-4": "bnp",
    "6301-6": "inr_alt",  # alternate INR code
}

# Reverse lookup: name -> LOINC code
NAME_TO_LOINC = {}
for code, name in {**VITAL_LOINC, **LAB_LOINC}.items():
    NAME_TO_LOINC[name] = code
# Manual overrides for names not matching dict values
NAME_TO_LOINC["heartrate"] = "8867-4"
NAME_TO_LOINC["systolicbp"] = "8480-6"
NAME_TO_LOINC["diastolicbp"] = "8462-4"
NAME_TO_LOINC["respiratoryrate"] = "9279-1"
NAME_TO_LOINC["oxygensaturation"] = "2708-6"
NAME_TO_LOINC["temperature"] = "8310-5"
NAME_TO_LOINC["weight"] = "29463-7"

# Run ID for tracing events through the pipeline
RUN_ID = f"e2e-det-{int(time.time())}"
MIN_PATIENTS = 8

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def now_ms():
    return int(time.time() * 1000)


def log(msg, level="INFO"):
    ts = datetime.now(timezone.utc).strftime("%H:%M:%S")
    print(f"[{ts}] [{level}] {msg}")


# ---------------------------------------------------------------------------
# FHIR Fetcher
# ---------------------------------------------------------------------------

class FHIRFetcher:
    """Fetch real patient data from GCP FHIR store."""

    def __init__(self, credentials_path=None):
        self.token = self._authenticate(credentials_path or CREDENTIALS_PATH)

    def _authenticate(self, creds_path):
        """Get OAuth2 token — tries ADC first, then service account key."""
        try:
            from google.auth.transport.requests import Request
            from google.auth import default as google_default
            credentials, _ = google_default(
                scopes=["https://www.googleapis.com/auth/cloud-healthcare"],
            )
            credentials.refresh(Request())
            log("Authenticated via Application Default Credentials")
            return credentials.token
        except Exception:
            pass

        try:
            from google.auth.transport.requests import Request
            from google.oauth2 import service_account
            if os.path.exists(creds_path):
                credentials = service_account.Credentials.from_service_account_file(
                    creds_path,
                    scopes=["https://www.googleapis.com/auth/cloud-healthcare"],
                )
                credentials.refresh(Request())
                log(f"Authenticated via service account key: {creds_path}")
                return credentials.token
        except Exception:
            pass

        raise RuntimeError(
            "No valid GCP credentials. Run: gcloud auth application-default login"
        )

    def _get(self, path, params=None):
        import requests
        url = f"{FHIR_BASE_URL}/{path}" if not path.startswith("http") else path
        headers = {"Authorization": f"Bearer {self.token}", "Accept": "application/fhir+json"}
        resp = requests.get(url, headers=headers, params=params, timeout=15)
        if resp.status_code == 200:
            return resp.json()
        log(f"FHIR GET {path} returned {resp.status_code}", "WARN")
        return None

    def _search_all(self, resource_type, params=None, max_pages=10):
        resources = []
        params = params or {}
        params.setdefault("_count", "100")
        bundle = self._get(resource_type, params=params)
        page = 0
        while bundle and page < max_pages:
            for entry in bundle.get("entry", []):
                resources.append(entry.get("resource", {}))
            next_link = None
            for link in bundle.get("link", []):
                if link.get("relation") == "next":
                    next_link = link.get("url")
                    break
            if not next_link:
                break
            bundle = self._get(next_link)
            page += 1
        return resources

    def list_patients(self):
        """Return list of Patient resources from FHIR store."""
        patients = self._search_all("Patient")
        log(f"Found {len(patients)} patients in FHIR store")
        return patients

    def get_observations(self, patient_id):
        """Get all Observations for a patient."""
        return self._search_all("Observation", {"subject": f"Patient/{patient_id}", "_sort": "-date"})

    def get_conditions(self, patient_id):
        """Get all Conditions for a patient."""
        return self._search_all("Condition", {"subject": f"Patient/{patient_id}"})

    def get_medication_requests(self, patient_id):
        """Get MedicationRequests for a patient."""
        return self._search_all("MedicationRequest", {"subject": f"Patient/{patient_id}"})


# ---------------------------------------------------------------------------
# Raw Event Builder
# ---------------------------------------------------------------------------

def raw_event(event_type, patient_id, payload, metadata=None,
              encounter_id=None, event_time=None):
    """Build a RawEvent dict matching Module 1 Java @JsonProperty annotations."""
    return {
        "id": f"{RUN_ID}-{event_type}-{uuid.uuid4().hex[:8]}",
        "source": "e2e-deterioration-test",
        "type": event_type,
        "patient_id": patient_id,
        "encounter_id": encounter_id or str(uuid.uuid4()),
        "event_time": event_time or now_ms(),
        "received_time": now_ms(),
        "payload": payload,
        "metadata": metadata or {
            "source": "e2e-deterioration-test",
            "location": "CARDIOLOGY_ICU",
            "device_id": "FHIR",
        },
        "correlation_id": str(uuid.uuid4()),
        "version": "1.0",
    }


def vital_event(patient_id, vitals_dict, encounter_id=None, event_time=None):
    """Create a vital-signs RawEvent from a dict of vital_name -> value."""
    payload = {}
    metadata_loinc = None
    for name, value in vitals_dict.items():
        # Map name to payload key expected by Module 1
        if name in ("heartRate", "heartrate"):
            payload["heartRate"] = value
            metadata_loinc = "8867-4"
        elif name in ("systolicBP", "systolicbp"):
            payload["systolicBP"] = value
            metadata_loinc = "8480-6"
        elif name in ("diastolicBP", "diastolicbp"):
            payload["diastolicBP"] = value
            metadata_loinc = "8462-4"
        elif name == "temperature":
            payload["temperature"] = value
            metadata_loinc = "8310-5"
        elif name in ("respiratoryRate", "respiratoryrate"):
            payload["respiratoryRate"] = value
            metadata_loinc = "9279-1"
        elif name in ("oxygenSaturation", "oxygensaturation", "spo2"):
            payload["oxygenSaturation"] = value
            metadata_loinc = "2708-6"
        elif name == "weight":
            payload["weight"] = value
            metadata_loinc = "29463-7"
        else:
            payload[name] = value

    return raw_event(
        "vital-signs", patient_id, payload,
        metadata={"source": "e2e-deterioration-test", "location": "CARDIOLOGY_ICU",
                   "device_id": "FHIR", "loinc_code": metadata_loinc or ""},
        encounter_id=encounter_id, event_time=event_time,
    )


def lab_event(patient_id, loinc_code, value, unit, lab_name=None,
              encounter_id=None, event_time=None):
    """Create a lab-result RawEvent."""
    payload = {
        "loinc_code": loinc_code,
        "value": value,
        "unit": unit,
        "lab_name": lab_name or LAB_LOINC.get(loinc_code, "unknown"),
        "status": "final",
    }
    return raw_event(
        "lab-result", patient_id, payload,
        metadata={"source": "e2e-deterioration-test", "location": "CARDIOLOGY_ICU",
                   "device_id": "FHIR", "loinc_code": loinc_code},
        encounter_id=encounter_id, event_time=event_time,
    )


def medication_event(patient_id, med_name, dose_value, dose_unit,
                     encounter_id=None, event_time=None):
    """Create a medication RawEvent."""
    payload = {
        "medication_name": med_name,
        "status": "active",
        "dosage": f"{dose_value} {dose_unit}",
        "dose_value": dose_value,
        "dose_unit": dose_unit,
        "route": "oral",
    }
    return raw_event(
        "medication-administration", patient_id, payload,
        encounter_id=encounter_id, event_time=event_time,
    )
```

- [ ] **Step 2: Verify syntax**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && python3 -c "import ast; ast.parse(open('scripts/flink_e2e_deterioration.py').read()); print('OK')"`
Expected: `OK`

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/scripts/flink_e2e_deterioration.py
git commit -m "feat(e2e): add FHIR fetcher and event builders for deterioration test"
```

---

### Task 3: Implement the 5 deterioration scenarios

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/scripts/flink_e2e_deterioration.py` (append after raw event builders)

- [ ] **Step 1: Add the DeteriorationEngine with all 5 scenarios**

Append after the `medication_event` function:

```python
# ---------------------------------------------------------------------------
# Deterioration Scenarios
# ---------------------------------------------------------------------------

class DeteriorationEngine:
    """Overlay deterioration trajectories on real patient baselines."""

    SCENARIOS = ["sepsis", "aki", "rapid_deterioration", "drug_lab", "cardiac_decompensation"]

    def assign_scenarios(self, patients):
        """Assign first 5 patients to scenarios, rest as controls.
        Returns dict: {patient_id: scenario_name_or_'control'}
        """
        assignment = {}
        for i, patient in enumerate(patients):
            pid = patient.get("id", "")
            if i < len(self.SCENARIOS):
                assignment[pid] = self.SCENARIOS[i]
            else:
                assignment[pid] = "control"
        return assignment

    def generate_events(self, patient_id, scenario, encounter_id, base_time_ms):
        """Generate T0, T1, T2 event lists for a given scenario.
        Returns: list of (timepoint_label, delay_seconds, list_of_events)
        """
        generators = {
            "sepsis": self._sepsis,
            "aki": self._aki,
            "rapid_deterioration": self._rapid_deterioration,
            "drug_lab": self._drug_lab,
            "cardiac_decompensation": self._cardiac_decompensation,
        }
        gen = generators.get(scenario)
        if gen is None:
            raise ValueError(f"Unknown scenario: {scenario}")
        return gen(patient_id, encounter_id, base_time_ms)

    def _sepsis(self, pid, eid, t0):
        """Sepsis progression: HR↑, SBP↓, lactate↑, temp↑, WBC↑."""
        t1 = t0 + 30_000
        t2 = t0 + 60_000

        return [
            ("T0", 0, [
                vital_event(pid, {"heartRate": 78, "systolicBP": 128, "diastolicBP": 82,
                                  "temperature": 37.0, "respiratoryRate": 16,
                                  "oxygenSaturation": 97}, eid, t0),
                lab_event(pid, "2524-7", 1.0, "mmol/L", "lactate", eid, t0),
                lab_event(pid, "6690-2", 8000, "/uL", "wbc", eid, t0),
                lab_event(pid, "33959-8", 0.1, "ng/mL", "procalcitonin", eid, t0),
            ]),
            ("T1", 30, [
                vital_event(pid, {"heartRate": 105, "systolicBP": 95, "diastolicBP": 60,
                                  "temperature": 38.5, "respiratoryRate": 22,
                                  "oxygenSaturation": 94}, eid, t1),
                lab_event(pid, "2524-7", 2.5, "mmol/L", "lactate", eid, t1),
                lab_event(pid, "6690-2", 14000, "/uL", "wbc", eid, t1),
                lab_event(pid, "33959-8", 0.8, "ng/mL", "procalcitonin", eid, t1),
            ]),
            ("T2", 60, [
                vital_event(pid, {"heartRate": 130, "systolicBP": 78, "diastolicBP": 45,
                                  "temperature": 39.5, "respiratoryRate": 28,
                                  "oxygenSaturation": 86}, eid, t2),
                lab_event(pid, "2524-7", 5.2, "mmol/L", "lactate", eid, t2),
                lab_event(pid, "6690-2", 22000, "/uL", "wbc", eid, t2),
                lab_event(pid, "33959-8", 4.5, "ng/mL", "procalcitonin", eid, t2),
            ]),
        ]

    def _aki(self, pid, eid, t0):
        """AKI: creatinine↑, eGFR↓, BUN↑, potassium↑."""
        t1 = t0 + 30_000
        t2 = t0 + 60_000

        return [
            ("T0", 0, [
                vital_event(pid, {"heartRate": 75, "systolicBP": 130, "diastolicBP": 80,
                                  "respiratoryRate": 16, "oxygenSaturation": 97,
                                  "temperature": 37.0}, eid, t0),
                lab_event(pid, "2160-0", 1.0, "mg/dL", "creatinine", eid, t0),
                lab_event(pid, "48642-3", 90, "mL/min/1.73m2", "egfr", eid, t0),
                lab_event(pid, "3094-0", 15, "mg/dL", "bun", eid, t0),
                lab_event(pid, "2823-3", 4.0, "mEq/L", "potassium", eid, t0),
            ]),
            ("T1", 30, [
                vital_event(pid, {"heartRate": 82, "systolicBP": 125, "diastolicBP": 78,
                                  "respiratoryRate": 18, "oxygenSaturation": 96,
                                  "temperature": 37.1}, eid, t1),
                lab_event(pid, "2160-0", 1.8, "mg/dL", "creatinine", eid, t1),
                lab_event(pid, "48642-3", 45, "mL/min/1.73m2", "egfr", eid, t1),
                lab_event(pid, "3094-0", 28, "mg/dL", "bun", eid, t1),
                lab_event(pid, "2823-3", 4.8, "mEq/L", "potassium", eid, t1),
            ]),
            ("T2", 60, [
                vital_event(pid, {"heartRate": 95, "systolicBP": 110, "diastolicBP": 70,
                                  "respiratoryRate": 22, "oxygenSaturation": 95,
                                  "temperature": 37.2}, eid, t2),
                lab_event(pid, "2160-0", 3.2, "mg/dL", "creatinine", eid, t2),
                lab_event(pid, "48642-3", 18, "mL/min/1.73m2", "egfr", eid, t2),
                lab_event(pid, "3094-0", 45, "mg/dL", "bun", eid, t2),
                lab_event(pid, "2823-3", 6.2, "mEq/L", "potassium", eid, t2),
            ]),
        ]

    def _rapid_deterioration(self, pid, eid, t0):
        """Rapid deterioration: multi-vital breach, NEWS2 escalation."""
        t1 = t0 + 30_000
        t2 = t0 + 60_000

        return [
            ("T0", 0, [
                vital_event(pid, {"heartRate": 80, "systolicBP": 125, "diastolicBP": 80,
                                  "temperature": 37.0, "respiratoryRate": 16,
                                  "oxygenSaturation": 97}, eid, t0),
                lab_event(pid, "2524-7", 0.8, "mmol/L", "lactate", eid, t0),
                lab_event(pid, "6690-2", 7500, "/uL", "wbc", eid, t0),
            ]),
            ("T1", 30, [
                vital_event(pid, {"heartRate": 110, "systolicBP": 100, "diastolicBP": 65,
                                  "temperature": 38.2, "respiratoryRate": 24,
                                  "oxygenSaturation": 92}, eid, t1),
                lab_event(pid, "2524-7", 1.8, "mmol/L", "lactate", eid, t1),
                lab_event(pid, "6690-2", 12000, "/uL", "wbc", eid, t1),
            ]),
            ("T2", 60, [
                vital_event(pid, {"heartRate": 135, "systolicBP": 82, "diastolicBP": 48,
                                  "temperature": 39.0, "respiratoryRate": 32,
                                  "oxygenSaturation": 85}, eid, t2),
                lab_event(pid, "2524-7", 3.5, "mmol/L", "lactate", eid, t2),
                lab_event(pid, "6690-2", 18000, "/uL", "wbc", eid, t2),
            ]),
        ]

    def _drug_lab(self, pid, eid, t0):
        """Drug-lab interaction: warfarin + rising INR + falling Hgb/platelets."""
        t1 = t0 + 30_000
        t2 = t0 + 60_000

        return [
            ("T0", 0, [
                medication_event(pid, "Warfarin", 5.0, "mg", eid, t0),
                vital_event(pid, {"heartRate": 72, "systolicBP": 128, "diastolicBP": 78,
                                  "respiratoryRate": 16, "oxygenSaturation": 98,
                                  "temperature": 36.8}, eid, t0),
                lab_event(pid, "34714-6", 2.5, "INR", "inr", eid, t0),
                lab_event(pid, "718-7", 13.0, "g/dL", "hemoglobin", eid, t0),
                lab_event(pid, "777-3", 250000, "/uL", "platelets", eid, t0),
            ]),
            ("T1", 30, [
                medication_event(pid, "Warfarin", 5.0, "mg", eid, t1),
                vital_event(pid, {"heartRate": 78, "systolicBP": 122, "diastolicBP": 75,
                                  "respiratoryRate": 16, "oxygenSaturation": 97,
                                  "temperature": 36.9}, eid, t1),
                lab_event(pid, "34714-6", 4.0, "INR", "inr", eid, t1),
                lab_event(pid, "718-7", 11.5, "g/dL", "hemoglobin", eid, t1),
                lab_event(pid, "777-3", 180000, "/uL", "platelets", eid, t1),
            ]),
            ("T2", 60, [
                medication_event(pid, "Warfarin", 5.0, "mg", eid, t2),
                vital_event(pid, {"heartRate": 88, "systolicBP": 115, "diastolicBP": 72,
                                  "respiratoryRate": 18, "oxygenSaturation": 97,
                                  "temperature": 37.0}, eid, t2),
                lab_event(pid, "34714-6", 6.0, "INR", "inr", eid, t2),
                lab_event(pid, "718-7", 9.5, "g/dL", "hemoglobin", eid, t2),
                lab_event(pid, "777-3", 120000, "/uL", "platelets", eid, t2),
            ]),
        ]

    def _cardiac_decompensation(self, pid, eid, t0):
        """Cardiac decompensation: BP variability, BNP spike, weight gain, SpO2 drop."""
        t1 = t0 + 30_000
        t2 = t0 + 60_000

        return [
            ("T0", 0, [
                vital_event(pid, {"heartRate": 72, "systolicBP": 135, "diastolicBP": 85,
                                  "respiratoryRate": 16, "oxygenSaturation": 97,
                                  "temperature": 36.8, "weight": 78.0}, eid, t0),
                lab_event(pid, "30934-4", 150, "pg/mL", "bnp", eid, t0),
            ]),
            ("T1", 30, [
                vital_event(pid, {"heartRate": 90, "systolicBP": 155, "diastolicBP": 95,
                                  "respiratoryRate": 22, "oxygenSaturation": 93,
                                  "temperature": 36.9, "weight": 80.0}, eid, t1),
                lab_event(pid, "30934-4", 600, "pg/mL", "bnp", eid, t1),
            ]),
            ("T2", 60, [
                vital_event(pid, {"heartRate": 115, "systolicBP": 100, "diastolicBP": 55,
                                  "respiratoryRate": 28, "oxygenSaturation": 88,
                                  "temperature": 37.0, "weight": 83.0}, eid, t2),
                lab_event(pid, "30934-4", 1200, "pg/mL", "bnp", eid, t2),
            ]),
        ]

    def generate_control_events(self, patient_id, observations, encounter_id, base_time_ms):
        """Pass through real FHIR observations as-is (healthy control)."""
        events = []
        for obs in observations[:10]:  # Limit to 10 most recent
            code = ""
            if "code" in obs and "coding" in obs["code"]:
                for coding in obs["code"]["coding"]:
                    if coding.get("system", "").endswith("loinc.org"):
                        code = coding.get("code", "")
                        break

            value = None
            unit = ""
            if "valueQuantity" in obs:
                value = obs["valueQuantity"].get("value")
                unit = obs["valueQuantity"].get("unit", "")

            if value is None:
                continue

            # Determine event time from FHIR resource
            evt_time = base_time_ms
            for dt_field in ("effectiveDateTime", "issued"):
                if dt_field in obs:
                    try:
                        from datetime import datetime as dt_cls
                        parsed = dt_cls.fromisoformat(obs[dt_field].replace("Z", "+00:00"))
                        evt_time = int(parsed.timestamp() * 1000)
                    except Exception:
                        pass
                    break

            if code in VITAL_LOINC:
                name = VITAL_LOINC[code]
                events.append((TOPIC_VITAL_SIGNS,
                               vital_event(patient_id, {name: value}, encounter_id, evt_time)))
            elif code in LAB_LOINC:
                events.append((TOPIC_LAB_RESULTS,
                               lab_event(patient_id, code, value, unit,
                                         LAB_LOINC[code], encounter_id, evt_time)))
        return [("T0-control", 0, events)]
```

- [ ] **Step 2: Verify syntax**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && python3 -c "import ast; ast.parse(open('scripts/flink_e2e_deterioration.py').read()); print('OK')"`
Expected: `OK`

- [ ] **Step 3: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/scripts/flink_e2e_deterioration.py
git commit -m "feat(e2e): add 5 deterioration scenarios + healthy control generator"
```

---

### Task 4: Implement KafkaPublisher and PipelineVerifier

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/scripts/flink_e2e_deterioration.py` (append)

- [ ] **Step 1: Add KafkaPublisher class**

Append after the `DeteriorationEngine` class:

```python
# ---------------------------------------------------------------------------
# Kafka Publisher
# ---------------------------------------------------------------------------

class KafkaPublisher:
    """Publish RawEvent JSON to Kafka via docker exec kafka-console-producer."""

    def __init__(self, container=KAFKA_CONTAINER, bootstrap=KAFKA_BOOTSTRAP):
        self.container = container
        self.bootstrap = bootstrap
        self.sent_count = 0

    def publish(self, topic, event_dict):
        """Send a single JSON event to a Kafka topic."""
        json_line = json.dumps(event_dict, separators=(",", ":"))
        cmd = [
            "docker", "exec", "-i", self.container,
            "kafka-console-producer",
            "--bootstrap-server", self.bootstrap,
            "--topic", topic,
        ]
        result = subprocess.run(
            cmd, input=json_line, capture_output=True, text=True, timeout=15,
        )
        if result.returncode != 0:
            log(f"ERROR producing to {topic}: {result.stderr.strip()}", "ERROR")
            return False
        self.sent_count += 1
        return True

    def publish_batch(self, topic_event_pairs):
        """Publish a list of (topic, event_dict) pairs."""
        success = 0
        for topic, event in topic_event_pairs:
            if self.publish(topic, event):
                success += 1
        return success

    def publish_scenario_timeline(self, patient_id, scenario_name, timepoints, dry_run=False):
        """Publish T0→T1→T2 with delays between timepoints.
        timepoints: list of (label, delay_seconds, events_or_topic_event_pairs)
        For deterioration scenarios, events are RawEvent dicts (need topic routing).
        For controls, events are (topic, event) tuples.
        """
        log(f"  Patient {patient_id[:8]}... scenario={scenario_name}")

        for label, delay_secs, events in timepoints:
            if delay_secs > 0 and not dry_run:
                log(f"    Waiting {delay_secs}s before {label}...")
                time.sleep(delay_secs)

            event_count = 0
            for item in events:
                # Controls return (topic, event) tuples; scenarios return raw events
                if isinstance(item, tuple) and len(item) == 2:
                    topic, event = item
                else:
                    event = item
                    # Route by event type
                    etype = event.get("type", "")
                    if etype == "vital-signs":
                        topic = TOPIC_VITAL_SIGNS
                    elif etype == "lab-result":
                        topic = TOPIC_LAB_RESULTS
                    elif etype == "medication-administration":
                        topic = TOPIC_MEDICATIONS
                    else:
                        topic = TOPIC_VITAL_SIGNS  # fallback

                if dry_run:
                    log(f"    [DRY-RUN] {label} → {topic}: {json.dumps(event.get('payload', {}), separators=(',', ':'))[:100]}")
                else:
                    self.publish(topic, event)
                event_count += 1

            log(f"    {label}: {event_count} events published")

        return True
```

- [ ] **Step 2: Add PipelineVerifier class**

Append after `KafkaPublisher`:

```python
# ---------------------------------------------------------------------------
# Pipeline Verifier
# ---------------------------------------------------------------------------

class PipelineVerifier:
    """Consume from output topics and verify pattern detection correctness."""

    def __init__(self, container=KAFKA_CONTAINER, bootstrap=KAFKA_BOOTSTRAP):
        self.container = container
        self.bootstrap = bootstrap

    def consume_topic(self, topic, timeout_sec=30, max_messages=500):
        """Consume all messages from a topic. Returns list of parsed JSON dicts."""
        group = f"e2e-det-verify-{uuid.uuid4().hex[:8]}"
        cmd = [
            "docker", "exec", self.container,
            "kafka-console-consumer",
            "--bootstrap-server", self.bootstrap,
            "--topic", topic,
            "--from-beginning",
            "--group", group,
            "--max-messages", str(max_messages),
            "--timeout-ms", str(timeout_sec * 1000),
        ]
        try:
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout_sec + 10)
        except subprocess.TimeoutExpired:
            log(f"Timeout consuming from {topic}", "WARN")
            return []

        messages = []
        for line in result.stdout.strip().split("\n"):
            line = line.strip()
            if not line:
                continue
            try:
                messages.append(json.loads(line))
            except json.JSONDecodeError:
                continue
        return messages

    def filter_by_run(self, messages, run_id=RUN_ID):
        """Filter messages that belong to this test run."""
        matched = []
        for msg in messages:
            msg_str = json.dumps(msg)
            if run_id in msg_str:
                matched.append(msg)
        return matched

    def verify_all(self, assignments, wait_sec=90):
        """Run full verification across all 4 output topics.
        assignments: dict {patient_id: scenario_name_or_'control'}
        Returns: dict with per-patient results
        """
        log(f"Waiting {wait_sec}s for pipeline to process all events...")
        time.sleep(wait_sec)

        log("Consuming from output topics...")
        enriched_msgs = self.consume_topic(TOPIC_ENRICHED, timeout_sec=30, max_messages=1000)
        cds_msgs = self.consume_topic(TOPIC_CDS, timeout_sec=20, max_messages=500)
        pattern_msgs = self.consume_topic(TOPIC_PATTERNS, timeout_sec=20, max_messages=500)

        # Filter by run_id
        enriched_run = self.filter_by_run(enriched_msgs)
        cds_run = self.filter_by_run(cds_msgs)
        pattern_run = self.filter_by_run(pattern_msgs)

        log(f"Messages found — enriched: {len(enriched_run)}, CDS: {len(cds_run)}, patterns: {len(pattern_run)}")

        results = {}
        for patient_id, scenario in assignments.items():
            pid_short = patient_id[:8]

            # Count events per patient per topic
            p_enriched = [m for m in enriched_run if patient_id in json.dumps(m)]
            p_cds = [m for m in cds_run if patient_id in json.dumps(m)]
            p_patterns = [m for m in pattern_run if patient_id in json.dumps(m)]

            # Extract pattern types
            detected_patterns = []
            for p in p_patterns:
                ptype = p.get("pattern_type", p.get("patternType", "UNKNOWN"))
                severity = p.get("severity", "UNKNOWN")
                detected_patterns.append({"type": ptype, "severity": severity})

            # Determine expected outcome
            expected_patterns = {
                "sepsis": ["SEPSIS"],
                "aki": ["AKI"],
                "rapid_deterioration": ["RAPID_DETERIORATION"],
                "drug_lab": ["DRUG_LAB_INTERACTION"],
                "cardiac_decompensation": ["RAPID_DETERIORATION"],
                "control": [],
            }

            expected = expected_patterns.get(scenario, [])

            if scenario == "control":
                passed = len(detected_patterns) == 0
            else:
                # Check that at least one expected pattern type was detected
                detected_types = {p["type"] for p in detected_patterns}
                passed = any(exp in detected_types for exp in expected)

            results[patient_id] = {
                "patient_id": patient_id,
                "scenario": scenario,
                "enriched_count": len(p_enriched),
                "cds_count": len(p_cds),
                "pattern_count": len(p_patterns),
                "detected_patterns": detected_patterns,
                "expected_patterns": expected,
                "pass": passed,
            }

            status = "✓ PASS" if passed else "✗ FAIL"
            if scenario == "control":
                log(f"  {pid_short}... [{scenario:25s}] {status} (patterns: {len(detected_patterns)})")
            else:
                types_str = ", ".join(p["type"] for p in detected_patterns) or "NONE"
                log(f"  {pid_short}... [{scenario:25s}] {status} (detected: {types_str})")

        return results
```

- [ ] **Step 3: Verify syntax**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && python3 -c "import ast; ast.parse(open('scripts/flink_e2e_deterioration.py').read()); print('OK')"`
Expected: `OK`

- [ ] **Step 4: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/scripts/flink_e2e_deterioration.py
git commit -m "feat(e2e): add KafkaPublisher and PipelineVerifier for deterioration test"
```

---

### Task 5: Implement ReportGenerator and main() entry point

**Files:**
- Modify: `backend/shared-infrastructure/flink-processing/scripts/flink_e2e_deterioration.py` (append)

- [ ] **Step 1: Add ReportGenerator and main function**

Append after `PipelineVerifier`:

```python
# ---------------------------------------------------------------------------
# Report Generator
# ---------------------------------------------------------------------------

class ReportGenerator:
    """Generate JSON report and console summary."""

    @staticmethod
    def build_report(results, events_sent, run_id=RUN_ID):
        total_pass = sum(1 for r in results.values() if r["pass"])
        total_fail = sum(1 for r in results.values() if not r["pass"])
        deteriorating = sum(1 for r in results.values() if r["scenario"] != "control")
        controls = sum(1 for r in results.values() if r["scenario"] == "control")
        false_positives = sum(
            1 for r in results.values()
            if r["scenario"] == "control" and r["pattern_count"] > 0
        )
        detected_count = sum(
            1 for r in results.values()
            if r["scenario"] != "control" and r["pass"]
        )

        return {
            "run_id": run_id,
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "patients_total": len(results),
            "patients_deteriorating": deteriorating,
            "patients_control": controls,
            "events_sent": events_sent,
            "results": list(results.values()),
            "summary": {
                "total_pass": total_pass,
                "total_fail": total_fail,
                "deterioration_detected": detected_count,
                "false_positives": false_positives,
            },
        }

    @staticmethod
    def print_summary(report):
        print("\n" + "=" * 70)
        print(f"  FLINK E2E DETERIORATION TEST REPORT — {report['run_id']}")
        print("=" * 70)
        print(f"  Patients: {report['patients_total']} "
              f"({report['patients_deteriorating']} deteriorating, "
              f"{report['patients_control']} controls)")
        print(f"  Events sent: {report['events_sent']}")
        print("-" * 70)
        print(f"  {'Patient':10s} {'Scenario':28s} {'Enriched':>8s} {'CDS':>5s} "
              f"{'Patterns':>8s} {'Result':>8s}")
        print("-" * 70)

        for r in report["results"]:
            pid = r["patient_id"][:8] + "..."
            status = "✓ PASS" if r["pass"] else "✗ FAIL"
            print(f"  {pid:10s} {r['scenario']:28s} {r['enriched_count']:>8d} "
                  f"{r['cds_count']:>5d} {r['pattern_count']:>8d} {status:>8s}")

        s = report["summary"]
        print("-" * 70)
        print(f"  TOTAL: {s['total_pass']} passed, {s['total_fail']} failed, "
              f"{s['deterioration_detected']}/{report['patients_deteriorating']} detected, "
              f"{s['false_positives']} false positives")
        print("=" * 70)

        if s["total_fail"] > 0:
            print("\n  ⚠ FAILURES:")
            for r in report["results"]:
                if not r["pass"]:
                    types = ", ".join(p["type"] for p in r["detected_patterns"]) or "NONE"
                    expected = ", ".join(r["expected_patterns"]) or "NONE"
                    print(f"    {r['patient_id'][:8]}... scenario={r['scenario']}: "
                          f"expected={expected}, got={types}")

    @staticmethod
    def save_json(report, output_dir):
        os.makedirs(output_dir, exist_ok=True)
        filename = f"e2e-deterioration-{report['run_id']}.json"
        path = os.path.join(output_dir, filename)
        with open(path, "w") as f:
            json.dump(report, f, indent=2)
        log(f"Report saved to {path}")
        return path


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(description="Flink E2E Deterioration Test")
    parser.add_argument("--dry-run", action="store_true",
                        help="Show events without sending to Kafka")
    parser.add_argument("--scenario", choices=DeteriorationEngine.SCENARIOS,
                        help="Run only a single scenario (plus controls)")
    parser.add_argument("--verify-only", action="store_true",
                        help="Only verify output topics (skip publishing)")
    parser.add_argument("--kafka", default=KAFKA_BOOTSTRAP,
                        help=f"Kafka bootstrap servers (default: {KAFKA_BOOTSTRAP})")
    parser.add_argument("--container", default=KAFKA_CONTAINER,
                        help=f"Kafka Docker container name (default: {KAFKA_CONTAINER})")
    parser.add_argument("--credentials", default=CREDENTIALS_PATH,
                        help="Path to GCP service account JSON")
    parser.add_argument("--wait", type=int, default=90,
                        help="Seconds to wait for pipeline processing (default: 90)")
    args = parser.parse_args()

    print(f"\n{'=' * 70}")
    print(f"  FLINK E2E DETERIORATION TEST")
    print(f"  Run ID: {RUN_ID}")
    print(f"  Time:   {datetime.now(timezone.utc).isoformat()}")
    print(f"{'=' * 70}\n")

    # --- Phase 1: Fetch patients from FHIR store ---
    if not args.verify_only:
        log("Phase 1: Fetching patients from FHIR store...")
        fetcher = FHIRFetcher(args.credentials)
        patients = fetcher.list_patients()

        if len(patients) < MIN_PATIENTS:
            log(f"Need at least {MIN_PATIENTS} patients, found {len(patients)}. Aborting.", "ERROR")
            sys.exit(1)

        patients = patients[:MIN_PATIENTS]
        log(f"Using {len(patients)} patients")

        # --- Phase 2: Assign scenarios ---
        log("Phase 2: Assigning deterioration scenarios...")
        engine = DeteriorationEngine()

        if args.scenario:
            # Single scenario mode: first patient gets the scenario, rest are controls
            assignments = {}
            for i, p in enumerate(patients):
                pid = p.get("id", "")
                assignments[pid] = args.scenario if i == 0 else "control"
        else:
            assignments = engine.assign_scenarios(patients)

        for pid, scenario in assignments.items():
            log(f"  {pid[:8]}... → {scenario}")

        # --- Phase 3: Generate and publish events ---
        log("Phase 3: Generating and publishing events...")
        publisher = KafkaPublisher(args.container, args.kafka)
        base_time = now_ms()

        for patient in patients:
            pid = patient.get("id", "")
            scenario = assignments[pid]
            encounter_id = str(uuid.uuid4())

            if scenario == "control":
                observations = fetcher.get_observations(pid)
                timepoints = engine.generate_control_events(
                    pid, observations, encounter_id, base_time,
                )
            else:
                timepoints = engine.generate_events(
                    pid, scenario, encounter_id, base_time,
                )

            publisher.publish_scenario_timeline(
                pid, scenario, timepoints, dry_run=args.dry_run,
            )

        log(f"Total events sent: {publisher.sent_count}")

        if args.dry_run:
            log("Dry run complete — no events were sent to Kafka.")
            return
    else:
        # Verify-only mode: need assignments from prior knowledge
        log("Verify-only mode: fetching patients to reconstruct assignments...")
        fetcher = FHIRFetcher(args.credentials)
        patients = fetcher.list_patients()[:MIN_PATIENTS]
        engine = DeteriorationEngine()
        assignments = engine.assign_scenarios(patients)

    # --- Phase 4: Verify pipeline output ---
    log("Phase 4: Verifying pipeline output...")
    verifier = PipelineVerifier(args.container, args.kafka)
    results = verifier.verify_all(assignments, wait_sec=args.wait)

    # --- Phase 5: Generate report ---
    log("Phase 5: Generating report...")
    events_sent = publisher.sent_count if not args.verify_only else 0
    report = ReportGenerator.build_report(results, events_sent)
    ReportGenerator.print_summary(report)

    test_data_dir = os.path.join(os.path.dirname(__file__), "..", "test-data")
    ReportGenerator.save_json(report, test_data_dir)

    # Exit code based on results
    if report["summary"]["total_fail"] > 0:
        sys.exit(1)


if __name__ == "__main__":
    main()
```

- [ ] **Step 2: Verify syntax**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && python3 -c "import ast; ast.parse(open('scripts/flink_e2e_deterioration.py').read()); print('OK')"`
Expected: `OK`

- [ ] **Step 3: Make script executable**

Run: `chmod +x /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing/scripts/flink_e2e_deterioration.py`

- [ ] **Step 4: Commit**

```bash
git add backend/shared-infrastructure/flink-processing/scripts/flink_e2e_deterioration.py
git commit -m "feat(e2e): add report generator and main entry point for deterioration test"
```

---

### Task 6: Run the E2E test end-to-end

**Prerequisites:** Kafka and Flink must be running.

- [ ] **Step 1: Ensure Kafka is running**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/kafka && docker compose -f docker-compose.hpi-lite.yml up -d`
Expected: Zookeeper and Kafka containers start. Verify: `docker ps | grep kafka-lite`

- [ ] **Step 2: Create the docker network (if needed)**

Run: `docker network create cardiofit-lite 2>/dev/null || true`

- [ ] **Step 3: Build the Flink JAR**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && mvn clean package -DskipTests -q`
Expected: `BUILD SUCCESS` with JAR at `target/flink-ehr-intelligence-1.0.0.jar`

- [ ] **Step 4: Start Flink with all 5 modules**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && docker compose -f docker-compose.e2e-flink.yml up -d`
Expected: `flink-jobmanager`, `flink-taskmanager`, `flink-submitter` containers start.

Wait ~30s for jobs to submit. Verify at http://localhost:8181 — should show 5 running jobs.

- [ ] **Step 5: Dry run first**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && python3 scripts/flink_e2e_deterioration.py --dry-run`
Expected: Script fetches 8 patients, assigns scenarios, prints event payloads without sending to Kafka.

- [ ] **Step 6: Full E2E run**

Run: `cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing && python3 scripts/flink_e2e_deterioration.py --wait 120`
Expected:
- Phase 1: 8 patients fetched
- Phase 2: 5 deterioration + 3 control assignments
- Phase 3: Events published with 30s delays between timepoints
- Phase 4: Verification after 120s wait
- Phase 5: Report generated with pass/fail per patient

Target: All 8 patients PASS (5 patterns detected, 3 controls clean).

- [ ] **Step 7: Review results and save test data**

Check the report file at `test-data/e2e-deterioration-{run_id}.json`.

If any scenarios FAIL, check Flink UI (http://localhost:8181) for:
- Job exceptions
- Checkpoint failures
- TaskManager logs

- [ ] **Step 8: Commit test results**

```bash
git add backend/shared-infrastructure/flink-processing/test-data/e2e-deterioration-*.json
git commit -m "test(e2e): add deterioration E2E test results for Modules 1-4"
```

---

### Task 7: Fix any failing scenarios and iterate

This task handles the likely case that some CEP patterns don't fire on first run.

- [ ] **Step 1: If patterns not detected — check Module 4 CEP thresholds**

Read the CEP select functions in Module4_PatternDetection.java to understand what conditions trigger each pattern. Common issues:
- Vital sign keys not matching (`heartRate` vs `heartrate` vs `heart_rate`)
- Lab values not reaching threshold (e.g., lactate must be ≥ 2.0 AND fever ≥ 38.3 for sepsis)
- Window timing — events may need to arrive within the CEP window (6-hour sliding)

- [ ] **Step 2: If Module 2 not enriching — check PatientContextAggregator**

Consume from `enriched-patient-events-v1` and inspect the `risk_indicators` and `clinical_scores` fields. If empty, the aggregator may not be building state from the incoming events.

Run: `docker exec cardiofit-kafka-lite kafka-console-consumer --bootstrap-server kafka-lite:29092 --topic enriched-patient-events-v1 --from-beginning --max-messages 5 --timeout-ms 10000 | python3 -m json.tool | head -100`

- [ ] **Step 3: If Module 3 CDS not producing — check CDS topic**

Run: `docker exec cardiofit-kafka-lite kafka-console-consumer --bootstrap-server kafka-lite:29092 --topic comprehensive-cds-events.v1 --from-beginning --max-messages 5 --timeout-ms 10000 | python3 -m json.tool | head -100`

- [ ] **Step 4: Adjust scenario values if needed and re-run**

Update the deterioration values in the script to match what the CEP patterns expect, then re-run:
`python3 scripts/flink_e2e_deterioration.py --wait 120`

- [ ] **Step 5: Final commit with working test**

```bash
git add -A backend/shared-infrastructure/flink-processing/scripts/flink_e2e_deterioration.py
git add backend/shared-infrastructure/flink-processing/test-data/e2e-deterioration-*.json
git commit -m "test(e2e): working deterioration E2E — all 5 scenarios + 3 controls passing"
```
