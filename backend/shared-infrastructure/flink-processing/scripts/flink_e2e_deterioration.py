#!/usr/bin/env python3
"""
Flink E2E Deterioration Test — Clinical Pattern Detection Validation

Fetches real patient data from GCP FHIR store (cardiofit-fhir-r4), assigns 5
patients to deterioration scenarios and 3 as healthy controls, then overlays
deteriorating vitals/labs/meds at T0 → T1 → T2 (30 seconds apart).

Events are published to Kafka via docker exec, then the script verifies that
Flink Modules 1→1b→2→3→4 produce expected clinical patterns on output topics.

Scenarios:
  1. Sepsis — rising HR/temp/lactate/WBC/procalcitonin, falling BP/SpO2
  2. AKI — rising creatinine/BUN/K+, falling eGFR
  3. Rapid Deterioration — multi-system decline (vitals + labs)
  4. Drug-Lab Interaction — Warfarin + rising INR, falling Hgb/platelets
  5. Cardiac Decompensation — BP variability, rising BNP/weight, falling SpO2

Usage:
  python3 scripts/flink_e2e_deterioration.py                      # full run
  python3 scripts/flink_e2e_deterioration.py --dry-run             # preview events
  python3 scripts/flink_e2e_deterioration.py --scenario sepsis     # single scenario
  python3 scripts/flink_e2e_deterioration.py --verify-only         # check outputs only
  python3 scripts/flink_e2e_deterioration.py --wait 120            # longer pipeline wait

Prerequisites:
  1. gcloud auth:  gcloud auth application-default login
     OR service account credentials at CREDENTIALS_PATH
  2. Kafka running:  cd ../kafka && docker compose -f docker-compose.hpi-lite.yml up -d
  3. Flink running:  docker compose -f docker-compose.e2e-flink.yml up -d
  4. pip install google-auth requests
"""

import argparse
import json
import os
import subprocess
import sys
import time
import uuid
from datetime import datetime
from pathlib import Path

try:
    from google.oauth2 import service_account
    from google.auth import default as google_default
    from google.auth.transport.requests import Request
    import requests
except ImportError:
    print("Missing dependencies. Install with:")
    print("  pip install google-auth google-auth-httplib2 requests")
    sys.exit(1)

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
PROJECT_ID = "project-2bbef9ac-174b-4b59-8fe"
LOCATION = "asia-south1"
DATASET_ID = "vaidshala-clinical"
FHIR_STORE_ID = "cardiofit-fhir-r4"
FHIR_BASE_URL = (
    f"https://healthcare.googleapis.com/v1/projects/{PROJECT_ID}"
    f"/locations/{LOCATION}/datasets/{DATASET_ID}/fhirStores/{FHIR_STORE_ID}/fhir"
)

CREDENTIALS_PATH = str(
    Path(__file__).resolve().parent.parent.parent
    / "services" / "patient-service" / "credentials" / "google-credentials.json"
)

KAFKA_CONTAINER = "cardiofit-kafka-lite"
KAFKA_BOOTSTRAP = "kafka-lite:29092"

# Module 1 input topics
TOPIC_VITAL_SIGNS = "vital-signs-events-v1"
TOPIC_LAB_RESULTS = "lab-result-events-v1"
TOPIC_MEDICATIONS = "medication-events-v1"

# Module output topics (verification)
TOPIC_ENRICHED = "enriched-patient-events-v1"
TOPIC_CONTEXT = "patient-context-snapshots-v1"
TOPIC_CDS = "comprehensive-cds-events.v1"
TOPIC_PATTERNS = "clinical-patterns.v1"

# ---------------------------------------------------------------------------
# LOINC Mappings (must match Module4SemanticConverter)
# ---------------------------------------------------------------------------
VITAL_LOINC = {
    "8867-4": "heartRate",
    "8480-6": "systolicBP",
    "8462-4": "diastolicBP",
    "8310-5": "temperature",
    "9279-1": "respiratoryRate",
    "2708-6": "oxygenSaturation",
    "59408-5": "oxygenSaturation",
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
}

# Reverse lookup: lab name → LOINC code
LAB_NAME_TO_LOINC = {v: k for k, v in LAB_LOINC.items()}
VITAL_NAME_TO_LOINC = {v: k for k, v in VITAL_LOINC.items() if k != "59408-5"}

RUN_ID = f"e2e-det-{int(time.time())}"


# ---------------------------------------------------------------------------
# FHIR Fetcher
# ---------------------------------------------------------------------------
class FHIRFetcher:
    """Fetches patient data from GCP Healthcare FHIR store."""

    def __init__(self, credentials_path=None):
        self._credentials_path = credentials_path or CREDENTIALS_PATH
        self._token = None
        self._authenticate()

    def _authenticate(self):
        """Try ADC first, fall back to service account key file."""
        try:
            credentials, _ = google_default(
                scopes=["https://www.googleapis.com/auth/cloud-healthcare"],
            )
            credentials.refresh(Request())
            self._token = credentials.token
            return
        except Exception:
            pass

        if os.path.exists(self._credentials_path):
            credentials = service_account.Credentials.from_service_account_file(
                self._credentials_path,
                scopes=["https://www.googleapis.com/auth/cloud-healthcare"],
            )
            credentials.refresh(Request())
            self._token = credentials.token
            return

        raise RuntimeError(
            "No valid credentials found. Run: gcloud auth application-default login"
        )

    def _get(self, path, params=None):
        """GET request to FHIR store. Returns parsed JSON or None."""
        url = f"{FHIR_BASE_URL}/{path}" if not path.startswith("http") else path
        headers = {
            "Authorization": f"Bearer {self._token}",
            "Accept": "application/fhir+json",
        }
        resp = requests.get(url, headers=headers, params=params, timeout=15)
        if resp.status_code == 200:
            return resp.json()
        return None

    def _search_all(self, resource_type, params=None, max_pages=10):
        """Paginated FHIR search following 'next' links."""
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
        """Return list of Patient resources."""
        return self._search_all("Patient", max_pages=3)

    def get_observations(self, patient_id):
        """Return Observation resources for a patient, sorted by -date."""
        return self._search_all(
            "Observation",
            params={"patient": patient_id, "_sort": "-date", "_count": "50"},
            max_pages=2,
        )

    def get_conditions(self, patient_id):
        """Return Condition resources for a patient."""
        return self._search_all(
            "Condition",
            params={"patient": patient_id, "_count": "50"},
            max_pages=1,
        )

    def get_medication_requests(self, patient_id):
        """Return MedicationRequest resources for a patient."""
        return self._search_all(
            "MedicationRequest",
            params={"patient": patient_id, "_count": "50"},
            max_pages=1,
        )


# ---------------------------------------------------------------------------
# Event Builders
# ---------------------------------------------------------------------------
def _now_ms():
    return int(time.time() * 1000)


def raw_event(event_type, patient_id, payload, metadata=None,
              encounter_id="", event_time=None):
    """Build a RawEvent dict matching Module 1 Java schema."""
    return {
        "id": f"{RUN_ID}-{event_type[:4]}-{uuid.uuid4().hex[:8]}",
        "source": "e2e-deterioration-test",
        "type": event_type,
        "patient_id": patient_id,
        "encounter_id": encounter_id,
        "event_time": event_time or _now_ms(),
        "received_time": _now_ms(),
        "payload": payload,
        "metadata": metadata or {
            "source": "e2e-deterioration-test",
            "location": "CARDIOLOGY_ICU",
            "device_id": "FHIR",
        },
        "correlation_id": str(uuid.uuid4()),
        "version": "1.0",
    }


def vital_event(patient_id, vitals_dict, encounter_id="", event_time=None):
    """Build a vital-signs RawEvent.

    vitals_dict: {heartRate: 80, systolicBP: 120, ...}
    """
    payload = dict(vitals_dict)
    return raw_event(
        event_type="vital-signs",
        patient_id=patient_id,
        payload=payload,
        metadata={
            "source": "e2e-deterioration-test",
            "location": "CARDIOLOGY_ICU",
            "device_id": "bedside-monitor",
        },
        encounter_id=encounter_id,
        event_time=event_time,
    )


def lab_event(patient_id, loinc_code, value, unit, lab_name,
              encounter_id="", event_time=None):
    """Build a lab-result RawEvent."""
    payload = {
        "testName": lab_name,
        "results": {lab_name: value},
        "units": {lab_name: unit},
        "loinc_code": loinc_code,
    }
    return raw_event(
        event_type="lab-result",
        patient_id=patient_id,
        payload=payload,
        metadata={
            "source": "e2e-deterioration-test",
            "location": "pathology-lab",
            "device_id": "FHIR",
            "loinc_code": loinc_code,
        },
        encounter_id=encounter_id,
        event_time=event_time,
    )


def medication_event(patient_id, med_name, dose_value, dose_unit,
                     encounter_id="", event_time=None):
    """Build a medication-administration RawEvent."""
    payload = {
        "medicationName": med_name,
        "status": "active",
        "dosageText": f"{dose_value} {dose_unit}",
        "dose": dose_value,
        "doseUnit": dose_unit,
    }
    return raw_event(
        event_type="medication-administration",
        patient_id=patient_id,
        payload=payload,
        metadata={
            "source": "e2e-deterioration-test",
            "location": "pharmacy",
            "device_id": "FHIR",
        },
        encounter_id=encounter_id,
        event_time=event_time,
    )


# ---------------------------------------------------------------------------
# Deterioration Engine
# ---------------------------------------------------------------------------
class DeteriorationEngine:
    """Generates timed deterioration event sequences for each scenario."""

    SCENARIOS = [
        "sepsis",
        "aki",
        "rapid_deterioration",
        "drug_lab",
        "cardiac_decompensation",
    ]

    # Expected Flink pattern names per scenario
    EXPECTED_PATTERNS = {
        "sepsis": "SEPSIS",
        "aki": "AKI",
        "rapid_deterioration": "RAPID_DETERIORATION",
        "drug_lab": "DRUG_LAB_INTERACTION",
        "cardiac_decompensation": "RAPID_DETERIORATION",
    }

    @staticmethod
    def assign_scenarios(patients):
        """Assign first 5 patients to scenarios, rest as controls.

        Returns dict {patient_id: scenario_name | 'control'}.
        """
        assignments = {}
        scenario_list = DeteriorationEngine.SCENARIOS
        for i, pat in enumerate(patients):
            pid = pat.get("id", pat) if isinstance(pat, dict) else pat
            if i < len(scenario_list):
                assignments[pid] = scenario_list[i]
            else:
                assignments[pid] = "control"
        return assignments

    @classmethod
    def generate_events(cls, patient_id, scenario, encounter_id, base_time_ms):
        """Generate (label, delay_secs, events_list) tuples for a scenario.

        Each events_list contains (topic, event_dict) pairs.
        delay_secs is relative to test start (T0=0, T1=30, T2=60).
        """
        gen = {
            "sepsis": cls._sepsis,
            "aki": cls._aki,
            "rapid_deterioration": cls._rapid_deterioration,
            "drug_lab": cls._drug_lab,
            "cardiac_decompensation": cls._cardiac_decompensation,
        }
        return gen[scenario](patient_id, encounter_id, base_time_ms)

    @classmethod
    def generate_control_events(cls, patient_id, observations, encounter_id,
                                base_time_ms):
        """Convert real FHIR observations into control events (no deterioration).

        Returns list of (label, delay_secs, events_list).
        """
        events = []
        for obs in observations[:10]:
            codings = obs.get("code", {}).get("coding", [])
            loinc_code = None
            for coding in codings:
                if coding.get("system", "").endswith("loinc.org"):
                    loinc_code = coding.get("code")
                    break
            if not loinc_code:
                continue

            value = None
            unit = ""
            if "valueQuantity" in obs:
                value = obs["valueQuantity"].get("value")
                unit = obs["valueQuantity"].get("unit", "")
            if value is None:
                continue

            if loinc_code in VITAL_LOINC:
                vname = VITAL_LOINC[loinc_code]
                evt = vital_event(
                    patient_id, {vname: value},
                    encounter_id=encounter_id, event_time=base_time_ms,
                )
                events.append((TOPIC_VITAL_SIGNS, evt))
            elif loinc_code in LAB_LOINC:
                lname = LAB_LOINC[loinc_code]
                evt = lab_event(
                    patient_id, loinc_code, value, unit, lname,
                    encounter_id=encounter_id, event_time=base_time_ms,
                )
                events.append((TOPIC_LAB_RESULTS, evt))

        return [("control-baseline", 0, events)]

    # ---- Scenario Generators ----

    @staticmethod
    def _sepsis(pid, enc, t0):
        """Sepsis: HR↑ temp↑ lactate↑ WBC↑ procalcitonin↑, BP↓ SpO2↓."""
        t1 = t0 + 30_000
        t2 = t0 + 60_000
        return [
            ("sepsis-T0-baseline", 0, [
                (TOPIC_VITAL_SIGNS, vital_event(pid, {
                    "heartRate": 78, "systolicBP": 128, "diastolicBP": 78,
                    "temperature": 37.0, "respiratoryRate": 16,
                    "oxygenSaturation": 97,
                }, encounter_id=enc, event_time=t0)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "2524-7", 1.0, "mmol/L", "lactate", enc, t0)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "6690-2", 8.0, "10*3/uL", "wbc", enc, t0)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "33959-8", 0.1, "ng/mL", "procalcitonin", enc, t0)),
            ]),
            ("sepsis-T1-early", 30, [
                (TOPIC_VITAL_SIGNS, vital_event(pid, {
                    "heartRate": 105, "systolicBP": 95, "diastolicBP": 60,
                    "temperature": 38.5, "respiratoryRate": 22,
                    "oxygenSaturation": 94,
                }, encounter_id=enc, event_time=t1)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "2524-7", 2.5, "mmol/L", "lactate", enc, t1)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "6690-2", 14.0, "10*3/uL", "wbc", enc, t1)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "33959-8", 0.8, "ng/mL", "procalcitonin", enc, t1)),
            ]),
            ("sepsis-T2-severe", 60, [
                (TOPIC_VITAL_SIGNS, vital_event(pid, {
                    "heartRate": 130, "systolicBP": 78, "diastolicBP": 45,
                    "temperature": 39.5, "respiratoryRate": 28,
                    "oxygenSaturation": 86,
                }, encounter_id=enc, event_time=t2)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "2524-7", 5.2, "mmol/L", "lactate", enc, t2)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "6690-2", 22.0, "10*3/uL", "wbc", enc, t2)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "33959-8", 4.5, "ng/mL", "procalcitonin", enc, t2)),
            ]),
        ]

    @staticmethod
    def _aki(pid, enc, t0):
        """AKI: creatinine↑ BUN↑ K+↑, eGFR↓, vitals mildly worsening."""
        t1 = t0 + 30_000
        t2 = t0 + 60_000
        return [
            ("aki-T0-baseline", 0, [
                (TOPIC_VITAL_SIGNS, vital_event(pid, {
                    "heartRate": 75, "systolicBP": 130, "diastolicBP": 80,
                    "temperature": 37.0, "respiratoryRate": 16,
                    "oxygenSaturation": 97,
                }, encounter_id=enc, event_time=t0)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "2160-0", 1.0, "mg/dL", "creatinine", enc, t0)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "48642-3", 90.0, "mL/min/1.73m2", "egfr", enc, t0)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "3094-0", 15.0, "mg/dL", "bun", enc, t0)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "2823-3", 4.0, "mmol/L", "potassium", enc, t0)),
            ]),
            ("aki-T1-early", 30, [
                (TOPIC_VITAL_SIGNS, vital_event(pid, {
                    "heartRate": 82, "systolicBP": 125, "diastolicBP": 78,
                    "temperature": 37.1, "respiratoryRate": 18,
                    "oxygenSaturation": 96,
                }, encounter_id=enc, event_time=t1)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "2160-0", 1.8, "mg/dL", "creatinine", enc, t1)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "48642-3", 45.0, "mL/min/1.73m2", "egfr", enc, t1)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "3094-0", 28.0, "mg/dL", "bun", enc, t1)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "2823-3", 4.8, "mmol/L", "potassium", enc, t1)),
            ]),
            ("aki-T2-severe", 60, [
                (TOPIC_VITAL_SIGNS, vital_event(pid, {
                    "heartRate": 90, "systolicBP": 118, "diastolicBP": 72,
                    "temperature": 37.3, "respiratoryRate": 20,
                    "oxygenSaturation": 95,
                }, encounter_id=enc, event_time=t2)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "2160-0", 3.2, "mg/dL", "creatinine", enc, t2)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "48642-3", 18.0, "mL/min/1.73m2", "egfr", enc, t2)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "3094-0", 45.0, "mg/dL", "bun", enc, t2)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "2823-3", 6.2, "mmol/L", "potassium", enc, t2)),
            ]),
        ]

    @staticmethod
    def _rapid_deterioration(pid, enc, t0):
        """Rapid deterioration: multi-system decline (vitals + labs)."""
        t1 = t0 + 30_000
        t2 = t0 + 60_000
        return [
            ("rapid-T0-baseline", 0, [
                (TOPIC_VITAL_SIGNS, vital_event(pid, {
                    "heartRate": 80, "systolicBP": 125, "diastolicBP": 78,
                    "temperature": 37.0, "respiratoryRate": 16,
                    "oxygenSaturation": 97,
                }, encounter_id=enc, event_time=t0)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "2524-7", 0.8, "mmol/L", "lactate", enc, t0)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "6690-2", 7.5, "10*3/uL", "wbc", enc, t0)),
            ]),
            ("rapid-T1-early", 30, [
                (TOPIC_VITAL_SIGNS, vital_event(pid, {
                    "heartRate": 110, "systolicBP": 100, "diastolicBP": 62,
                    "temperature": 38.2, "respiratoryRate": 24,
                    "oxygenSaturation": 92,
                }, encounter_id=enc, event_time=t1)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "2524-7", 1.8, "mmol/L", "lactate", enc, t1)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "6690-2", 12.0, "10*3/uL", "wbc", enc, t1)),
            ]),
            ("rapid-T2-severe", 60, [
                (TOPIC_VITAL_SIGNS, vital_event(pid, {
                    "heartRate": 135, "systolicBP": 82, "diastolicBP": 48,
                    "temperature": 39.0, "respiratoryRate": 32,
                    "oxygenSaturation": 85,
                }, encounter_id=enc, event_time=t2)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "2524-7", 3.5, "mmol/L", "lactate", enc, t2)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "6690-2", 18.0, "10*3/uL", "wbc", enc, t2)),
            ]),
        ]

    @staticmethod
    def _drug_lab(pid, enc, t0):
        """Drug-lab interaction: Warfarin + rising INR, falling Hgb/platelets."""
        t1 = t0 + 30_000
        t2 = t0 + 60_000
        return [
            ("drug-lab-T0-baseline", 0, [
                (TOPIC_VITAL_SIGNS, vital_event(pid, {
                    "heartRate": 72, "systolicBP": 122, "diastolicBP": 76,
                    "temperature": 37.0, "respiratoryRate": 15,
                    "oxygenSaturation": 98,
                }, encounter_id=enc, event_time=t0)),
                (TOPIC_MEDICATIONS, medication_event(pid, "Warfarin", 5.0, "mg", enc, t0)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "34714-6", 2.5, "ratio", "inr", enc, t0)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "718-7", 13.0, "g/dL", "hemoglobin", enc, t0)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "777-3", 250.0, "10*3/uL", "platelets", enc, t0)),
            ]),
            ("drug-lab-T1-early", 30, [
                (TOPIC_VITAL_SIGNS, vital_event(pid, {
                    "heartRate": 78, "systolicBP": 118, "diastolicBP": 74,
                    "temperature": 37.0, "respiratoryRate": 16,
                    "oxygenSaturation": 97,
                }, encounter_id=enc, event_time=t1)),
                (TOPIC_MEDICATIONS, medication_event(pid, "Warfarin", 5.0, "mg", enc, t1)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "34714-6", 4.0, "ratio", "inr", enc, t1)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "718-7", 11.5, "g/dL", "hemoglobin", enc, t1)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "777-3", 180.0, "10*3/uL", "platelets", enc, t1)),
            ]),
            ("drug-lab-T2-severe", 60, [
                (TOPIC_VITAL_SIGNS, vital_event(pid, {
                    "heartRate": 85, "systolicBP": 110, "diastolicBP": 68,
                    "temperature": 37.1, "respiratoryRate": 18,
                    "oxygenSaturation": 96,
                }, encounter_id=enc, event_time=t2)),
                (TOPIC_MEDICATIONS, medication_event(pid, "Warfarin", 5.0, "mg", enc, t2)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "34714-6", 6.0, "ratio", "inr", enc, t2)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "718-7", 9.5, "g/dL", "hemoglobin", enc, t2)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "777-3", 120.0, "10*3/uL", "platelets", enc, t2)),
            ]),
        ]

    @staticmethod
    def _cardiac_decompensation(pid, enc, t0):
        """Cardiac decompensation: BP variability, BNP↑ weight↑, SpO2↓ RR↑."""
        t1 = t0 + 30_000
        t2 = t0 + 60_000
        return [
            ("cardiac-T0-baseline", 0, [
                (TOPIC_VITAL_SIGNS, vital_event(pid, {
                    "heartRate": 72, "systolicBP": 135, "diastolicBP": 82,
                    "respiratoryRate": 16, "oxygenSaturation": 97,
                    "weight": 78.0,
                }, encounter_id=enc, event_time=t0)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "30934-4", 150.0, "pg/mL", "bnp", enc, t0)),
            ]),
            ("cardiac-T1-early", 30, [
                (TOPIC_VITAL_SIGNS, vital_event(pid, {
                    "heartRate": 90, "systolicBP": 155, "diastolicBP": 92,
                    "respiratoryRate": 22, "oxygenSaturation": 93,
                    "weight": 80.0,
                }, encounter_id=enc, event_time=t1)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "30934-4", 600.0, "pg/mL", "bnp", enc, t1)),
            ]),
            ("cardiac-T2-severe", 60, [
                (TOPIC_VITAL_SIGNS, vital_event(pid, {
                    "heartRate": 115, "systolicBP": 100, "diastolicBP": 60,
                    "respiratoryRate": 28, "oxygenSaturation": 88,
                    "weight": 83.0,
                }, encounter_id=enc, event_time=t2)),
                (TOPIC_LAB_RESULTS, lab_event(pid, "30934-4", 1200.0, "pg/mL", "bnp", enc, t2)),
            ]),
        ]


# ---------------------------------------------------------------------------
# Kafka Publisher
# ---------------------------------------------------------------------------
class KafkaPublisher:
    """Publishes events to Kafka via docker exec kafka-console-producer."""

    def __init__(self, container=None, bootstrap=None):
        self.container = container or KAFKA_CONTAINER
        self.bootstrap = bootstrap or KAFKA_BOOTSTRAP
        self.sent_count = 0
        self.error_count = 0

    def publish(self, topic, event_dict):
        """Send a single-line JSON event to Kafka topic."""
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
            print(f"    ERROR producing to {topic}: {result.stderr.strip()}")
            self.error_count += 1
            return False
        self.sent_count += 1
        return True

    def publish_scenario_timeline(self, patient_id, scenario_name, timepoints,
                                  dry_run=False):
        """Publish events for a scenario, sleeping between timepoints.

        timepoints: list of (label, delay_secs, [(topic, event), ...])
        For controls, events are already (topic, event) tuples.
        """
        print(f"\n  [{scenario_name.upper()}] Patient: {patient_id}")

        last_delay = 0
        for label, delay_secs, events in timepoints:
            # Sleep the delta between this timepoint and the previous one
            wait = delay_secs - last_delay
            if wait > 0 and not dry_run:
                print(f"    Waiting {wait}s before {label}...")
                time.sleep(wait)
            last_delay = delay_secs

            print(f"    {label}: sending {len(events)} events")

            for topic, event in events:
                if dry_run:
                    print(f"      [DRY-RUN] {topic} → {event['type']} "
                          f"(patient={event['patient_id'][:12]}...)")
                else:
                    self.publish(topic, event)

        return self.sent_count


# ---------------------------------------------------------------------------
# Pipeline Verifier
# ---------------------------------------------------------------------------
class PipelineVerifier:
    """Consumes Flink output topics and verifies expected patterns."""

    def __init__(self, container=None, bootstrap=None):
        self.container = container or KAFKA_CONTAINER
        self.bootstrap = bootstrap or KAFKA_BOOTSTRAP

    def consume_topic(self, topic, timeout_sec=30, max_messages=500):
        """Consume messages from a topic. Returns list of parsed JSON dicts."""
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
            result = subprocess.run(
                cmd, capture_output=True, text=True,
                timeout=timeout_sec + 15,
            )
        except subprocess.TimeoutExpired:
            return []

        messages = []
        if result.stdout.strip():
            for line in result.stdout.strip().split("\n"):
                line = line.strip()
                if not line:
                    continue
                try:
                    messages.append(json.loads(line))
                except json.JSONDecodeError:
                    continue
        return messages

    @staticmethod
    def filter_by_run(messages, run_id):
        """Filter messages containing the RUN_ID anywhere in the JSON."""
        filtered = []
        for msg in messages:
            if run_id in json.dumps(msg):
                filtered.append(msg)
        return filtered

    def verify_all(self, assignments, wait_sec=90):
        """Wait for pipeline processing, then verify output topics.

        Returns dict with per-patient verification results.
        """
        print(f"\n{'='*60}")
        print(f"  VERIFICATION PHASE — waiting {wait_sec}s for pipeline")
        print(f"{'='*60}")
        time.sleep(wait_sec)

        results = {
            "enriched": {"total": 0, "by_patient": {}},
            "context": {"total": 0, "by_patient": {}},
            "cds": {"total": 0, "by_patient": {}},
            "patterns": {"total": 0, "by_patient": {}},
            "scenario_verification": {},
        }

        # Consume all 4 output topics
        topics = {
            "enriched": TOPIC_ENRICHED,
            "context": TOPIC_CONTEXT,
            "cds": TOPIC_CDS,
            "patterns": TOPIC_PATTERNS,
        }

        for label, topic in topics.items():
            print(f"\n  Consuming {topic}...")
            messages = self.consume_topic(topic, timeout_sec=30, max_messages=500)
            run_messages = self.filter_by_run(messages, RUN_ID)
            results[label]["total"] = len(run_messages)
            print(f"    Total: {len(messages)}, Run-filtered: {len(run_messages)}")

            # Group by patient
            for msg in run_messages:
                pid = (msg.get("patient_id") or msg.get("patientId")
                       or msg.get("payload", {}).get("patient_id") or "unknown")
                results[label]["by_patient"].setdefault(pid, []).append(msg)

        # Verify per-scenario pattern detection
        for pid, scenario in assignments.items():
            verification = {
                "scenario": scenario,
                "expected_pattern": DeteriorationEngine.EXPECTED_PATTERNS.get(scenario),
                "enriched_count": len(results["enriched"]["by_patient"].get(pid, [])),
                "context_count": len(results["context"]["by_patient"].get(pid, [])),
                "cds_count": len(results["cds"]["by_patient"].get(pid, [])),
                "patterns_count": len(results["patterns"]["by_patient"].get(pid, [])),
                "detected_patterns": [],
                "pass": False,
            }

            # Extract detected pattern names from clinical-patterns topic
            for msg in results["patterns"]["by_patient"].get(pid, []):
                pattern_name = (msg.get("patternType") or msg.get("pattern_type")
                                or msg.get("type") or "")
                if pattern_name:
                    verification["detected_patterns"].append(pattern_name)

            # Determine pass/fail
            if scenario == "control":
                # Controls should NOT trigger deterioration patterns
                verification["pass"] = verification["patterns_count"] == 0
            else:
                expected = verification["expected_pattern"]
                verification["pass"] = expected in verification["detected_patterns"]

            results["scenario_verification"][pid] = verification

        return results


# ---------------------------------------------------------------------------
# Report Generator
# ---------------------------------------------------------------------------
class ReportGenerator:
    """Builds and outputs E2E test reports."""

    @staticmethod
    def build_report(results, events_sent, run_id):
        """Build structured report dict."""
        scenarios = results.get("scenario_verification", {})
        total = len(scenarios)
        passed = sum(1 for v in scenarios.values() if v["pass"])
        failed = total - passed

        return {
            "run_id": run_id,
            "timestamp": datetime.utcnow().isoformat() + "Z",
            "events_sent": events_sent,
            "summary": {
                "total_scenarios": total,
                "passed": passed,
                "failed": failed,
                "pass_rate": f"{(passed/total*100):.0f}%" if total > 0 else "N/A",
            },
            "topic_counts": {
                "enriched": results["enriched"]["total"],
                "context": results["context"]["total"],
                "cds": results["cds"]["total"],
                "patterns": results["patterns"]["total"],
            },
            "scenario_results": {
                pid: {
                    "scenario": v["scenario"],
                    "expected": v.get("expected_pattern"),
                    "detected": v["detected_patterns"],
                    "enriched": v["enriched_count"],
                    "context": v["context_count"],
                    "cds": v["cds_count"],
                    "patterns": v["patterns_count"],
                    "pass": v["pass"],
                }
                for pid, v in scenarios.items()
            },
        }

    @staticmethod
    def print_summary(report):
        """Print formatted console summary."""
        print(f"\n{'='*70}")
        print(f"  E2E DETERIORATION TEST REPORT — {report['run_id']}")
        print(f"{'='*70}")
        s = report["summary"]
        print(f"  Events Sent:  {report['events_sent']}")
        print(f"  Pass Rate:    {s['pass_rate']} ({s['passed']}/{s['total_scenarios']})")
        print()

        tc = report["topic_counts"]
        print(f"  Topic Counts (run-filtered):")
        print(f"    enriched:  {tc['enriched']}")
        print(f"    context:   {tc['context']}")
        print(f"    cds:       {tc['cds']}")
        print(f"    patterns:  {tc['patterns']}")
        print()

        # Per-scenario table
        print(f"  {'Patient':<20} {'Scenario':<25} {'Expected':<22} "
              f"{'Detected':<25} {'Result'}")
        print(f"  {'-'*18:<20} {'-'*23:<25} {'-'*20:<22} "
              f"{'-'*23:<25} {'-'*6}")
        for pid, sr in report["scenario_results"].items():
            pid_short = pid[:18] if len(pid) > 18 else pid
            detected = ",".join(sr["detected"]) if sr["detected"] else "(none)"
            status = "PASS" if sr["pass"] else "FAIL"
            expected = sr.get("expected") or "(none)"
            print(f"  {pid_short:<20} {sr['scenario']:<25} {expected:<22} "
                  f"{detected:<25} {status}")

        print(f"\n{'='*70}\n")

    @staticmethod
    def save_json(report, output_dir=None):
        """Save report as JSON file."""
        if output_dir is None:
            output_dir = Path(__file__).resolve().parent.parent / "test-data"
        else:
            output_dir = Path(output_dir)
        output_dir.mkdir(parents=True, exist_ok=True)

        filepath = output_dir / f"e2e-deterioration-{report['run_id']}.json"
        with open(filepath, "w") as f:
            json.dump(report, f, indent=2)
        print(f"  Report saved: {filepath}")
        return str(filepath)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
def main():
    parser = argparse.ArgumentParser(
        description="Flink E2E Deterioration Test — Clinical Pattern Detection",
    )
    parser.add_argument("--dry-run", action="store_true",
                        help="Preview events without sending to Kafka")
    parser.add_argument("--scenario", choices=DeteriorationEngine.SCENARIOS,
                        help="Run a single scenario (first patient gets it, rest control)")
    parser.add_argument("--verify-only", action="store_true",
                        help="Only verify output topics (skip event generation)")
    parser.add_argument("--kafka", default=None,
                        help="Override Kafka bootstrap servers")
    parser.add_argument("--container", default=None,
                        help="Override Docker container name")
    parser.add_argument("--credentials", default=None,
                        help="Override GCP credentials path")
    parser.add_argument("--wait", type=int, default=90,
                        help="Seconds to wait for pipeline processing (default: 90)")
    args = parser.parse_args()

    global KAFKA_CONTAINER, KAFKA_BOOTSTRAP
    if args.kafka:
        KAFKA_BOOTSTRAP = args.kafka
    if args.container:
        KAFKA_CONTAINER = args.container

    print(f"\n{'='*60}")
    print(f"  Flink E2E Deterioration Test")
    print(f"  Run ID: {RUN_ID}")
    print(f"{'='*60}")

    # ---- Phase 1: Fetch patients from FHIR ----
    print(f"\n  Phase 1: Fetching patients from FHIR store...")
    try:
        fetcher = FHIRFetcher(credentials_path=args.credentials)
        patients = fetcher.list_patients()
    except Exception as e:
        print(f"  ERROR: Could not fetch patients: {e}")
        sys.exit(1)

    if len(patients) < 8:
        print(f"  WARNING: Found only {len(patients)} patients (need 8 for full test)")
        if len(patients) < 1:
            print("  FATAL: No patients found in FHIR store.")
            sys.exit(1)

    patient_ids = [p["id"] for p in patients[:8]]
    print(f"  Found {len(patients)} patients, using {len(patient_ids)}")

    # ---- Phase 2: Assign scenarios ----
    print(f"\n  Phase 2: Assigning scenarios...")
    engine = DeteriorationEngine()

    if args.scenario:
        # Single scenario mode: first patient gets it, rest are controls
        assignments = {}
        assignments[patient_ids[0]] = args.scenario
        for pid in patient_ids[1:]:
            assignments[pid] = "control"
    else:
        assignments = engine.assign_scenarios(patient_ids)

    for pid, scenario in assignments.items():
        pid_short = pid[:20] if len(pid) > 20 else pid
        print(f"    {pid_short}... → {scenario}")

    if args.verify_only:
        # Skip to verification
        verifier = PipelineVerifier(
            container=args.container or KAFKA_CONTAINER,
            bootstrap=args.kafka or KAFKA_BOOTSTRAP,
        )
        results = verifier.verify_all(assignments, wait_sec=5)
        report = ReportGenerator.build_report(results, 0, RUN_ID)
        ReportGenerator.print_summary(report)
        ReportGenerator.save_json(report)
        has_failures = report["summary"]["failed"] > 0
        sys.exit(1 if has_failures else 0)

    # ---- Phase 3: Generate and publish events ----
    print(f"\n  Phase 3: Generating and publishing events...")
    publisher = KafkaPublisher(
        container=args.container or KAFKA_CONTAINER,
        bootstrap=args.kafka or KAFKA_BOOTSTRAP,
    )
    base_time_ms = _now_ms()

    for pid, scenario in assignments.items():
        encounter_id = f"enc-{RUN_ID}-{uuid.uuid4().hex[:8]}"

        if scenario == "control":
            # Fetch real observations and send as-is
            try:
                observations = fetcher.get_observations(pid)
            except Exception:
                observations = []
            timepoints = engine.generate_control_events(
                pid, observations, encounter_id, base_time_ms,
            )
        else:
            timepoints = engine.generate_events(
                pid, scenario, encounter_id, base_time_ms,
            )

        publisher.publish_scenario_timeline(
            pid, scenario, timepoints, dry_run=args.dry_run,
        )

    total_sent = publisher.sent_count
    total_errors = publisher.error_count
    print(f"\n  Events sent: {total_sent}, errors: {total_errors}")

    if args.dry_run:
        print("\n  DRY-RUN complete. No events were sent.")
        sys.exit(0)

    # ---- Phase 4: Verify pipeline output ----
    verifier = PipelineVerifier(
        container=args.container or KAFKA_CONTAINER,
        bootstrap=args.kafka or KAFKA_BOOTSTRAP,
    )
    results = verifier.verify_all(assignments, wait_sec=args.wait)

    # ---- Phase 5: Generate report ----
    print(f"\n  Phase 5: Generating report...")
    report = ReportGenerator.build_report(results, total_sent, RUN_ID)
    ReportGenerator.print_summary(report)
    ReportGenerator.save_json(report)

    has_failures = report["summary"]["failed"] > 0
    if has_failures:
        print("  RESULT: SOME SCENARIOS FAILED — exit code 1")
    else:
        print("  RESULT: ALL SCENARIOS PASSED")

    sys.exit(1 if has_failures else 0)


if __name__ == "__main__":
    main()
