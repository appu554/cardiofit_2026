#!/usr/bin/env python3
"""
Flink E2E Real Data Generator — Fetches from GCP FHIR Store

Queries real patient data (Observations, Conditions, MedicationStatements)
from the Google Cloud Healthcare FHIR store and converts them into RawEvent
JSON for Kafka ingestion through Modules 1 → 2 → 3 → 4.

Each event is sent as single-line JSON (kafka-console-producer uses newline
as record delimiter — multi-line JSON breaks into separate messages).

Usage:
  python3 scripts/flink_e2e_real_data.py                    # all patients, all data
  python3 scripts/flink_e2e_real_data.py --patient <ID>     # specific patient
  python3 scripts/flink_e2e_real_data.py --list              # list available patients
  python3 scripts/flink_e2e_real_data.py --check             # check downstream topics
  python3 scripts/flink_e2e_real_data.py --no-check          # send but skip verification
  python3 scripts/flink_e2e_real_data.py --max-obs 20        # limit observations per patient
  python3 scripts/flink_e2e_real_data.py --dry-run           # show what would be sent

Prerequisites:
  1. gcloud auth:  gcloud auth application-default login
     OR service account credentials at the CREDENTIALS_PATH below
  2. Kafka running:  cd ../kafka && docker compose -f docker-compose.hpi-lite.yml up -d
  3. Flink running:  docker compose -f docker-compose.e2e-flink.yml up -d
  4. pip install google-auth requests
"""

import argparse
import json
import subprocess
import sys
import time
import uuid
from datetime import datetime

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
# FHIR Store Config
# ---------------------------------------------------------------------------
PROJECT_ID = "project-2bbef9ac-174b-4b59-8fe"
LOCATION = "asia-south1"
DATASET_ID = "vaidshala-clinical"
FHIR_STORE_ID = "cardiofit-fhir-r4"
FHIR_BASE_URL = (
    f"https://healthcare.googleapis.com/v1/projects/{PROJECT_ID}"
    f"/locations/{LOCATION}/datasets/{DATASET_ID}/fhirStores/{FHIR_STORE_ID}/fhir"
)

# Service account credentials (fallback if ADC not available)
CREDENTIALS_PATH = "/Users/apoorvabk/Downloads/cardiofit/backend/services/patient-service/credentials/google-credentials.json"

# ---------------------------------------------------------------------------
# Kafka Config
# ---------------------------------------------------------------------------
KAFKA_CONTAINER = "cardiofit-kafka-lite"
KAFKA_BOOTSTRAP = "kafka-lite:29092"

# Module 1 input topics
TOPIC_VITAL_SIGNS = "vital-signs-events-v1"
TOPIC_LAB_RESULTS = "lab-result-events-v1"
TOPIC_OBSERVATIONS = "observation-events-v1"
TOPIC_MEDICATIONS = "medication-events-v1"
TOPIC_PATIENT_EVENTS = "patient-events-v1"

# Downstream output topics
TOPIC_ENRICHED = "enriched-patient-events-v1"
TOPIC_CONTEXT = "patient-context-snapshots-v1"
TOPIC_CDS = "comprehensive-cds-events.v1"
TOPIC_PATTERNS = "clinical-patterns.v1"

# LOINC codes → vital sign names (for classifying Observations)
VITAL_LOINC = {
    "8867-4": "heartRate",
    "8480-6": "systolicBP",
    "8462-4": "diastolicBP",
    "8310-5": "temperature",
    "9279-1": "respiratoryRate",
    "2708-6": "oxygenSaturation",
    "59408-5": "oxygenSaturation",  # SpO2 by pulse oximetry
    "8302-2": "height",
    "29463-7": "weight",
    "39156-5": "bmi",
}

# LOINC codes considered lab results
LAB_LOINC = {
    "2160-0": "creatinine",
    "3094-0": "bun",
    "6690-2": "wbc",
    "718-7": "hemoglobin",
    "777-3": "platelets",
    "2345-7": "glucose",
    "2823-3": "potassium",
    "2951-2": "sodium",
    "4548-4": "hba1c",
    "2524-7": "lactate",
    "33959-8": "procalcitonin",
    "2085-9": "hdlCholesterol",
    "2089-1": "ldlCholesterol",
    "2093-3": "totalCholesterol",
    "2571-8": "triglycerides",
    "48642-3": "egfr",
    "14959-1": "microalbuminCreatinineRatio",
    "6299-2": "urineMicroalbumin",
    "5902-2": "pt",
    "34714-6": "inr",
    "3173-2": "aptt",
    "1742-6": "alt",
    "1920-8": "ast",
    "1975-2": "totalBilirubin",
}

RUN_ID = f"e2e-{int(time.time())}"


# ---------------------------------------------------------------------------
# FHIR Authentication
# ---------------------------------------------------------------------------

def get_fhir_token():
    """Get OAuth2 token — tries ADC first, then service account file."""
    import os
    # Try Application Default Credentials first (gcloud auth application-default login)
    try:
        credentials, _ = google_default(
            scopes=["https://www.googleapis.com/auth/cloud-healthcare"],
        )
        credentials.refresh(Request())
        return credentials.token
    except Exception:
        pass

    # Fallback to service account key file
    if os.path.exists(CREDENTIALS_PATH):
        credentials = service_account.Credentials.from_service_account_file(
            CREDENTIALS_PATH,
            scopes=["https://www.googleapis.com/auth/cloud-healthcare"],
        )
        credentials.refresh(Request())
        return credentials.token

    raise RuntimeError("No valid credentials found. Run: gcloud auth application-default login")


def fhir_get(path, token, params=None):
    """GET request to FHIR store. Returns parsed JSON or None."""
    url = f"{FHIR_BASE_URL}/{path}" if not path.startswith("http") else path
    headers = {"Authorization": f"Bearer {token}", "Accept": "application/fhir+json"}
    resp = requests.get(url, headers=headers, params=params, timeout=15)
    if resp.status_code == 200:
        return resp.json()
    return None


def fhir_search_all(resource_type, token, params=None, max_pages=10):
    """Search with pagination. Returns list of FHIR resources."""
    resources = []
    params = params or {}
    params.setdefault("_count", "100")
    bundle = fhir_get(resource_type, token, params=params)
    page = 0
    while bundle and page < max_pages:
        for entry in bundle.get("entry", []):
            resources.append(entry.get("resource", {}))
        # Follow next link
        next_link = None
        for link in bundle.get("link", []):
            if link.get("relation") == "next":
                next_link = link.get("url")
                break
        if not next_link:
            break
        bundle = fhir_get(next_link, token)
        page += 1
    return resources


# ---------------------------------------------------------------------------
# FHIR → RawEvent converters
# ---------------------------------------------------------------------------

def now_ms():
    return int(time.time() * 1000)


def fhir_timestamp_to_epoch_ms(ts_str):
    """Convert FHIR datetime string to epoch milliseconds."""
    if not ts_str:
        return now_ms()
    for fmt in ("%Y-%m-%dT%H:%M:%S%z", "%Y-%m-%dT%H:%M:%S.%f%z",
                "%Y-%m-%dT%H:%M:%S", "%Y-%m-%d"):
        try:
            dt = datetime.strptime(ts_str.replace("+00:00", "+0000").replace("Z", "+0000"), fmt)
            return int(dt.timestamp() * 1000)
        except ValueError:
            continue
    return now_ms()


def raw_event(event_id, source, event_type, patient_id, encounter_id,
              payload, metadata=None, correlation_id=None, event_time=None):
    """Build a RawEvent dict matching Java @JsonProperty snake_case annotations."""
    return {
        "id": event_id,
        "source": source,
        "type": event_type,
        "patient_id": patient_id,
        "encounter_id": encounter_id or "",
        "event_time": event_time or now_ms(),
        "received_time": now_ms(),
        "payload": payload,
        "metadata": metadata or {"source": source, "location": "UNKNOWN", "device_id": "UNKNOWN"},
        "correlation_id": correlation_id or str(uuid.uuid4()),
        "version": "1.0",
    }


def observation_to_raw_event(obs, patient_id):
    """Convert a FHIR Observation resource to a RawEvent."""
    codings = obs.get("code", {}).get("coding", [])
    loinc_code = None
    display = obs.get("code", {}).get("text", "Unknown")
    for coding in codings:
        if coding.get("system", "").endswith("loinc.org"):
            loinc_code = coding.get("code")
            display = coding.get("display", display)
            break

    # Extract value
    value = None
    unit = None
    if "valueQuantity" in obs:
        value = obs["valueQuantity"].get("value")
        unit = obs["valueQuantity"].get("unit", "")
    elif "valueCodeableConcept" in obs:
        value = obs["valueCodeableConcept"].get("text", "")
    elif "component" in obs:
        # Multi-component (e.g., blood pressure)
        components = {}
        for comp in obs["component"]:
            comp_code = None
            for c in comp.get("code", {}).get("coding", []):
                comp_code = c.get("code")
                break
            if "valueQuantity" in comp:
                comp_name = VITAL_LOINC.get(comp_code, comp_code or "unknown")
                components[comp_name] = comp["valueQuantity"].get("value")
        if components:
            value = components

    # Determine topic based on LOINC code
    is_vital = loinc_code in VITAL_LOINC
    is_lab = loinc_code in LAB_LOINC

    encounter_id = ""
    if obs.get("encounter", {}).get("reference"):
        encounter_id = obs["encounter"]["reference"].replace("Encounter/", "")

    event_time = fhir_timestamp_to_epoch_ms(
        obs.get("effectiveDateTime") or obs.get("issued")
    )

    if is_vital:
        vital_name = VITAL_LOINC[loinc_code]
        payload = {vital_name: value}
        if unit:
            payload["unit"] = unit
        return TOPIC_VITAL_SIGNS, raw_event(
            event_id=f"{RUN_ID}-obs-{obs.get('id', uuid.uuid4().hex[:8])}",
            source="gcp-fhir-store",
            event_type="vital-signs",
            patient_id=patient_id,
            encounter_id=encounter_id,
            payload=payload,
            metadata={"source": "gcp-fhir-store", "location": "fhir-observation",
                      "device_id": "FHIR", "loinc_code": loinc_code or ""},
            event_time=event_time,
        )
    elif is_lab:
        lab_name = LAB_LOINC[loinc_code]
        payload = {
            "testName": display,
            "results": {lab_name: value},
            "units": {lab_name: unit or ""},
            "loinc_code": loinc_code,
        }
        return TOPIC_LAB_RESULTS, raw_event(
            event_id=f"{RUN_ID}-obs-{obs.get('id', uuid.uuid4().hex[:8])}",
            source="gcp-fhir-store",
            event_type="lab-result",
            patient_id=patient_id,
            encounter_id=encounter_id,
            payload=payload,
            metadata={"source": "gcp-fhir-store", "location": "pathology-lab",
                      "device_id": "FHIR", "loinc_code": loinc_code or ""},
            event_time=event_time,
        )
    else:
        # Generic observation
        payload = {"observationType": display, "value": value, "unit": unit or ""}
        if loinc_code:
            payload["loinc_code"] = loinc_code
        return TOPIC_OBSERVATIONS, raw_event(
            event_id=f"{RUN_ID}-obs-{obs.get('id', uuid.uuid4().hex[:8])}",
            source="gcp-fhir-store",
            event_type="observation",
            patient_id=patient_id,
            encounter_id=encounter_id,
            payload=payload,
            metadata={"source": "gcp-fhir-store", "location": "fhir-observation",
                      "device_id": "FHIR", "loinc_code": loinc_code or ""},
            event_time=event_time,
        )


def medication_to_raw_event(med, patient_id):
    """Convert a FHIR MedicationStatement/MedicationRequest to a RawEvent."""
    med_name = "Unknown"
    med_concept = med.get("medicationCodeableConcept") or med.get("medicationReference", {})
    if isinstance(med_concept, dict):
        med_name = med_concept.get("text", med_concept.get("display", "Unknown"))
        for coding in med_concept.get("coding", []):
            med_name = coding.get("display", med_name)
            break

    dosage_text = ""
    dose_value = None
    dose_unit = ""
    route = ""
    if med.get("dosage"):
        d = med["dosage"][0]
        dosage_text = d.get("text", "")
        if d.get("doseAndRate"):
            dr = d["doseAndRate"][0]
            if dr.get("doseQuantity"):
                dose_value = dr["doseQuantity"].get("value")
                dose_unit = dr["doseQuantity"].get("unit", "")
        if d.get("route", {}).get("text"):
            route = d["route"]["text"]

    encounter_id = ""
    if med.get("encounter", {}).get("reference"):
        encounter_id = med["encounter"]["reference"].replace("Encounter/", "")

    event_time = fhir_timestamp_to_epoch_ms(
        med.get("effectiveDateTime") or med.get("dateAsserted")
        or med.get("authoredOn")
    )

    payload = {
        "medicationName": med_name,
        "status": med.get("status", "unknown"),
        "dosageText": dosage_text,
    }
    if dose_value is not None:
        payload["dose"] = dose_value
        payload["doseUnit"] = dose_unit
    if route:
        payload["route"] = route

    return TOPIC_MEDICATIONS, raw_event(
        event_id=f"{RUN_ID}-med-{med.get('id', uuid.uuid4().hex[:8])}",
        source="gcp-fhir-store",
        event_type="medication-administration",
        patient_id=patient_id,
        encounter_id=encounter_id,
        payload=payload,
        metadata={"source": "gcp-fhir-store", "location": "pharmacy", "device_id": "FHIR"},
        event_time=event_time,
    )


def condition_to_raw_event(cond, patient_id):
    """Convert a FHIR Condition resource to a RawEvent."""
    display = cond.get("code", {}).get("text", "Unknown condition")
    codings = cond.get("code", {}).get("coding", [])
    icd_code = ""
    for coding in codings:
        display = coding.get("display", display)
        if "icd" in coding.get("system", "").lower() or "snomed" in coding.get("system", "").lower():
            icd_code = coding.get("code", "")

    clinical_status = "unknown"
    if cond.get("clinicalStatus", {}).get("coding"):
        clinical_status = cond["clinicalStatus"]["coding"][0].get("code", "unknown")

    encounter_id = ""
    if cond.get("encounter", {}).get("reference"):
        encounter_id = cond["encounter"]["reference"].replace("Encounter/", "")

    event_time = fhir_timestamp_to_epoch_ms(
        cond.get("onsetDateTime") or cond.get("recordedDate")
    )

    payload = {
        "conditionName": display,
        "clinicalStatus": clinical_status,
        "code": icd_code,
    }

    return TOPIC_OBSERVATIONS, raw_event(
        event_id=f"{RUN_ID}-cond-{cond.get('id', uuid.uuid4().hex[:8])}",
        source="gcp-fhir-store",
        event_type="clinical-assessment",
        patient_id=patient_id,
        encounter_id=encounter_id,
        payload=payload,
        metadata={"source": "gcp-fhir-store", "location": "clinical-record", "device_id": "FHIR"},
        event_time=event_time,
    )


# ---------------------------------------------------------------------------
# Kafka Producer
# ---------------------------------------------------------------------------

def produce(topic, event_dict):
    """Send a single-line JSON to Kafka via docker exec."""
    json_line = json.dumps(event_dict, separators=(",", ":"))
    cmd = [
        "docker", "exec", "-i", KAFKA_CONTAINER,
        "kafka-console-producer",
        "--bootstrap-server", KAFKA_BOOTSTRAP,
        "--topic", topic,
    ]
    result = subprocess.run(cmd, input=json_line, capture_output=True, text=True, timeout=15)
    if result.returncode != 0:
        print(f"  ERROR producing to {topic}: {result.stderr.strip()}")
        return False
    return True


def consume_check(topic, pattern, timeout=25, max_messages=200):
    """Consume from topic with unique group. Returns matching lines."""
    group = f"e2e-py-{uuid.uuid4().hex[:8]}"
    cmd = [
        "docker", "exec", KAFKA_CONTAINER,
        "kafka-console-consumer",
        "--bootstrap-server", KAFKA_BOOTSTRAP,
        "--topic", topic,
        "--from-beginning",
        "--group", group,
        "--max-messages", str(max_messages),
        "--timeout-ms", str(timeout * 1000),
    ]
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout + 15)
    except subprocess.TimeoutExpired:
        return []
    lines = result.stdout.strip().split("\n") if result.stdout.strip() else []
    return [l for l in lines if pattern in l]


# ---------------------------------------------------------------------------
# Core: Fetch from FHIR → produce to Kafka
# ---------------------------------------------------------------------------

def fetch_and_produce_patient(patient_id, token, max_obs=50, dry_run=False):
    """Fetch all data for a patient from FHIR store and send to Kafka."""
    print(f"\n{'='*60}")
    print(f"  Patient: {patient_id}")
    print(f"{'='*60}")

    events = []

    # 1. Fetch Observations — query by known LOINC codes to skip intake-form data
    all_known_loincs = list(VITAL_LOINC.keys()) + list(LAB_LOINC.keys())

    vitals_count = 0
    labs_count = 0
    other_count = 0

    print(f"  Fetching Observations (vitals + labs by LOINC code)...")
    for loinc in all_known_loincs:
        observations = fhir_search_all(
            "Observation", token,
            params={
                "patient": patient_id,
                "code": f"http://loinc.org|{loinc}",
                "_sort": "-date",
                "_count": "10",
            },
            max_pages=1,
        )
        for obs in observations[:max_obs]:
            topic, event = observation_to_raw_event(obs, patient_id)
            events.append((topic, event))
            if topic == TOPIC_VITAL_SIGNS:
                vitals_count += 1
            elif topic == TOPIC_LAB_RESULTS:
                labs_count += 1
            else:
                other_count += 1

    print(f"    Found: {vitals_count} vitals, {labs_count} labs, {other_count} other")

    # 2. Fetch MedicationStatements
    print(f"  Fetching MedicationStatements...")
    medications = fhir_search_all(
        "MedicationStatement", token,
        params={"patient": patient_id},
    )
    print(f"    Found {len(medications)} medication statements")
    for med in medications:
        topic, event = medication_to_raw_event(med, patient_id)
        events.append((topic, event))

    # 3. Fetch MedicationRequests (some stores use this instead)
    med_requests = fhir_search_all(
        "MedicationRequest", token,
        params={"patient": patient_id},
    )
    if med_requests:
        print(f"    Found {len(med_requests)} medication requests")
        for med in med_requests:
            topic, event = medication_to_raw_event(med, patient_id)
            events.append((topic, event))

    # 4. Fetch Conditions
    print(f"  Fetching Conditions...")
    conditions = fhir_search_all(
        "Condition", token,
        params={"patient": patient_id},
    )
    print(f"    Found {len(conditions)} conditions")
    for cond in conditions:
        topic, event = condition_to_raw_event(cond, patient_id)
        events.append((topic, event))

    # Sort by event_time for realistic temporal ordering
    events.sort(key=lambda x: x[1]["event_time"])

    print(f"\n  Total events to send: {len(events)}")

    if dry_run:
        print(f"\n  [DRY RUN] Would send {len(events)} events. First 3:")
        for topic, event in events[:3]:
            print(f"    {topic}: {json.dumps(event, indent=2)[:300]}...")
        return events

    # Produce to Kafka
    ok_count = 0
    fail_count = 0
    topic_counts = {}

    for i, (topic, event) in enumerate(events, 1):
        ok = produce(topic, event)
        topic_counts[topic] = topic_counts.get(topic, 0) + 1
        if ok:
            ok_count += 1
        else:
            fail_count += 1

        # Progress every 10 events
        if i % 10 == 0 or i == len(events):
            print(f"    Sent {i}/{len(events)} events ({ok_count} ok, {fail_count} fail)")

        # Small delay every 5 events to avoid overwhelming Kafka
        if i % 5 == 0 and i < len(events):
            time.sleep(0.3)

    print(f"\n  Per-topic breakdown:")
    for topic, count in sorted(topic_counts.items()):
        print(f"    {topic}: {count}")

    return events


def list_patients(token, count=10):
    """List patients in the FHIR store."""
    print(f"\n{'='*60}")
    print(f"  Patients in FHIR Store")
    print(f"{'='*60}")

    bundle = fhir_get("Patient", token, params={"_count": str(count)})
    if not bundle:
        print("  ERROR: Could not query FHIR store")
        return []

    total = bundle.get("total", 0)
    entries = bundle.get("entry", [])
    print(f"  Total: {total}, showing first {len(entries)}\n")

    patient_ids = []
    for entry in entries:
        res = entry.get("resource", {})
        pid = res.get("id", "?")
        patient_ids.append(pid)
        name = "Unknown"
        if res.get("name"):
            n = res["name"][0]
            given = " ".join(n.get("given", []))
            family = n.get("family", "")
            name = f"{given} {family}".strip()
        gender = res.get("gender", "?")
        dob = res.get("birthDate", "?")
        print(f"  {pid}")
        print(f"    Name: {name}  |  Gender: {gender}  |  DOB: {dob}")

    return patient_ids


def check_pipeline():
    """Check all 4 downstream topics for our test events."""
    print(f"\n{'='*60}")
    print(f"  PIPELINE VERIFICATION")
    print(f"  Run ID: {RUN_ID}")
    print(f"{'='*60}")

    print("\n  Waiting 15s for pipeline processing...")
    time.sleep(15)

    topics = [
        (TOPIC_ENRICHED, "Module 1 output"),
        (TOPIC_CONTEXT, "Module 2 output"),
        (TOPIC_CDS, "Module 3 output"),
        (TOPIC_PATTERNS, "Module 4 output"),
    ]

    results = {}
    for topic, label in topics:
        matches = consume_check(topic, RUN_ID, timeout=25)
        results[topic] = matches
        count = len(matches)
        if count > 0:
            print(f"\n  {label} ({topic}): {count} event(s)")
            for m in matches[:2]:
                display = m[:200] + "..." if len(m) > 200 else m
                print(f"    > {display}")
        else:
            print(f"\n  {label} ({topic}): 0 events")

    print(f"\n{'='*60}")
    print(f"  SUMMARY")
    print(f"{'='*60}")
    for topic, label in topics:
        count = len(results[topic])
        marker = "PASS" if count > 0 else "----"
        print(f"  [{marker}] {label}: {count} events")

    total = sum(len(v) for v in results.values())
    if total > 0:
        print(f"\n  Pipeline flowing: {total} events found across topics.")
    else:
        print(f"\n  No events found. Troubleshooting:")
        print(f"    1. curl http://localhost:8181/jobs/overview")
        print(f"    2. docker logs cardiofit-flink-taskmanager 2>&1 | grep 'Failed to deserialize' | tail -5")
        print(f"    3. docker exec {KAFKA_CONTAINER} kafka-topics --list --bootstrap-server {KAFKA_BOOTSTRAP}")

    return results


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    parser = argparse.ArgumentParser(
        description="Flink E2E Real Data Generator — fetches from GCP FHIR Store",
    )
    parser.add_argument("--patient", type=str, default=None,
                        help="Specific patient ID (default: all patients)")
    parser.add_argument("--list", action="store_true",
                        help="List available patients in FHIR store")
    parser.add_argument("--check", action="store_true",
                        help="Only check downstream topics")
    parser.add_argument("--no-check", action="store_true",
                        help="Send events but skip pipeline verification")
    parser.add_argument("--max-obs", type=int, default=50,
                        help="Max observations per patient (default: 50)")
    parser.add_argument("--max-patients", type=int, default=10,
                        help="Max patients to process (default: 10)")
    parser.add_argument("--dry-run", action="store_true",
                        help="Show what would be sent without producing to Kafka")
    args = parser.parse_args()

    print(f"\n{'#'*60}")
    print(f"  Flink E2E Real Data Generator (FHIR Store)")
    print(f"  Run ID:     {RUN_ID}")
    print(f"  FHIR Store: {PROJECT_ID}/{DATASET_ID}/{FHIR_STORE_ID}")
    print(f"  Kafka:      {KAFKA_CONTAINER} @ {KAFKA_BOOTSTRAP}")
    print(f"{'#'*60}")

    if args.check:
        check_pipeline()
        return

    # Authenticate with FHIR store
    print("\n  Authenticating with GCP...")
    try:
        token = get_fhir_token()
        print("  OK — token obtained")
    except Exception as e:
        print(f"  FAILED: {e}")
        print("  Run: gcloud auth application-default login")
        sys.exit(1)

    if args.list:
        list_patients(token)
        return

    # Verify Kafka container (unless dry-run)
    if not args.dry_run:
        result = subprocess.run(
            ["docker", "inspect", "--format", "{{.State.Running}}", KAFKA_CONTAINER],
            capture_output=True, text=True,
        )
        if result.stdout.strip() != "true":
            print(f"\n  ERROR: {KAFKA_CONTAINER} is not running!")
            print(f"  Start: cd ../kafka && docker compose -f docker-compose.hpi-lite.yml up -d")
            sys.exit(1)

    # Determine which patients to process
    if args.patient:
        patient_ids = [args.patient]
    else:
        print("\n  Discovering patients from FHIR store...")
        patient_ids = list_patients(token, count=args.max_patients)
        if not patient_ids:
            print("  No patients found in FHIR store.")
            sys.exit(1)

    # Fetch and produce for each patient
    total_events = 0
    for pid in patient_ids:
        events = fetch_and_produce_patient(
            pid, token,
            max_obs=args.max_obs,
            dry_run=args.dry_run,
        )
        total_events += len(events)

    print(f"\n{'#'*60}")
    print(f"  Total: {total_events} events from {len(patient_ids)} patient(s)")
    print(f"{'#'*60}")

    if not args.dry_run and not args.no_check:
        check_pipeline()


if __name__ == "__main__":
    main()
