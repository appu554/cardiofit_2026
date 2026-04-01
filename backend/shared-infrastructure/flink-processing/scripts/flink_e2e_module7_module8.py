#!/usr/bin/env python3
"""
Flink E2E Test — Module 7 (BP Variability) + Module 8 (Comorbidity Interaction)

Produces realistic clinical data directly to Kafka input topics and verifies
Flink output on the respective output topics.

Module 7 flow:
  ingestion.vitals (BPReading JSON)
    → Flink Module7_BPVariabilityEngine
    → flink.bp-variability-metrics  (BPVariabilityMetrics)
    → ingestion.safety-critical     (crisis / acute-surge side-outputs)

Module 8 flow:
  enriched-patient-events-v1 (CanonicalEvent JSON)
    → Flink Module8_ComorbidityEngine
    → alerts.comorbidity-interactions  (CIDAlert)
    → ingestion.safety-critical        (HALT side-output)

Usage:
  python3 scripts/flink_e2e_module7_module8.py                # full run
  python3 scripts/flink_e2e_module7_module8.py --module7-only  # BP variability only
  python3 scripts/flink_e2e_module7_module8.py --module8-only  # comorbidity only
  python3 scripts/flink_e2e_module7_module8.py --dry-run       # show payloads, don't send
  python3 scripts/flink_e2e_module7_module8.py --check         # just check output topics

Prerequisites:
  1. Kafka running:  cd ../kafka && docker compose -f docker-compose.hpi-lite.yml up -d
  2. Build JAR:      mvn clean package -DskipTests -q
  3. Flink running:  docker compose -f docker-compose.e2e-flink.yml up -d
  4. Topics created: python3 scripts/flink_e2e_module7_module8.py --create-topics
"""

import argparse
import json
import subprocess
import sys
import time
import uuid
from datetime import datetime, timezone, timedelta

# ---------------------------------------------------------------------------
# Kafka Config
# ---------------------------------------------------------------------------
KAFKA_CONTAINER = "cardiofit-kafka-lite"
KAFKA_BOOTSTRAP_INTERNAL = "kafka-lite:29092"

# Module 7 topics
TOPIC_M7_INPUT = "ingestion.vitals"
TOPIC_M7_OUTPUT = "flink.bp-variability-metrics"
TOPIC_SAFETY_CRITICAL = "ingestion.safety-critical"

# Module 8 topics
TOPIC_M8_INPUT = "enriched-patient-events-v1"
TOPIC_M8_OUTPUT = "alerts.comorbidity-interactions"

# All topics needed for Module 7-8 e2e
ALL_TOPICS = {
    TOPIC_M7_INPUT: 8,
    TOPIC_M7_OUTPUT: 8,
    TOPIC_M8_INPUT: 4,
    TOPIC_M8_OUTPUT: 4,
    TOPIC_SAFETY_CRITICAL: 4,
}

RUN_ID = f"e2e-m7m8-{int(time.time())}"
# Short suffix used in patient IDs (last 6 chars of timestamp)
RUN_SUFFIX = RUN_ID[-6:]


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
def now_ms():
    return int(time.time() * 1000)


def days_ago_ms(days):
    return now_ms() - days * 86_400_000


def hours_ago_ms(hours):
    return now_ms() - hours * 3_600_000


def morning_time(days_ago=0):
    """Return epoch ms for 07:00 UTC on the given day."""
    d = datetime.now(timezone.utc).replace(hour=7, minute=0, second=0, microsecond=0)
    d -= timedelta(days=days_ago)
    return int(d.timestamp() * 1000)


def evening_time(days_ago=0):
    """Return epoch ms for 20:00 UTC on the given day."""
    d = datetime.now(timezone.utc).replace(hour=20, minute=0, second=0, microsecond=0)
    d -= timedelta(days=days_ago)
    return int(d.timestamp() * 1000)


def night_time(days_ago=0):
    """Return epoch ms for 02:00 UTC on the given day."""
    d = datetime.now(timezone.utc).replace(hour=2, minute=0, second=0, microsecond=0)
    d -= timedelta(days=days_ago)
    return int(d.timestamp() * 1000)


# ---------------------------------------------------------------------------
# Module 7: BPReading builders
# ---------------------------------------------------------------------------
def bp_reading(patient_id, systolic, diastolic, timestamp,
               time_context=None, source="HOME_CUFF", heart_rate=None):
    """Build BPReading JSON matching @JsonProperty annotations."""
    r = {
        "patient_id": patient_id,
        "systolic": systolic,
        "diastolic": diastolic,
        "timestamp": timestamp,
        "source": source,
        "correlation_id": f"{RUN_ID}-{uuid.uuid4().hex[:8]}",
    }
    if time_context:
        r["time_context"] = time_context
    if heart_rate:
        r["heart_rate"] = heart_rate
    return r


# ---------------------------------------------------------------------------
# Module 8: CanonicalEvent builders
# ---------------------------------------------------------------------------
def canonical_event(patient_id, event_type, payload, event_time=None):
    """Build CanonicalEvent JSON matching @JsonProperty annotations."""
    return {
        "eventId": f"{RUN_ID}-{uuid.uuid4().hex[:8]}",
        "patientId": patient_id,
        "eventType": event_type,
        "timestamp": event_time or now_ms(),
        "processingTime": now_ms(),
        "sourceSystem": "e2e-test",
        "payload": payload,
    }


def med_event(patient_id, drug_name, drug_class, dose_mg=10.0, event_time=None):
    return canonical_event(patient_id, "MEDICATION_ORDERED", {
        "drug_name": drug_name,
        "drug_class": drug_class,
        "dose_mg": dose_mg,
    }, event_time)


def vital_event(patient_id, sbp=None, dbp=None, weight=None, event_time=None):
    payload = {}
    if sbp is not None:
        payload["systolic_bp"] = sbp
    if dbp is not None:
        payload["diastolic_bp"] = dbp
    if weight is not None:
        payload["weight"] = weight
    return canonical_event(patient_id, "VITAL_SIGN", payload, event_time)


def lab_event(patient_id, lab_type, value, event_time=None):
    return canonical_event(patient_id, "LAB_RESULT", {
        "lab_type": lab_type,
        "value": value,
    }, event_time)


def symptom_event(patient_id, symptom_type, status=None, event_time=None):
    payload = {"symptom_type": symptom_type}
    if status:
        payload["status"] = status
    return canonical_event(patient_id, "PATIENT_REPORTED", payload, event_time)


# ---------------------------------------------------------------------------
# Clinical Scenarios — Module 7
# ---------------------------------------------------------------------------
def build_module7_scenarios():
    """
    Build realistic BP reading sequences for 4 clinical profiles.

    Scenario 1: Uncontrolled hypertensive with morning surge
    Scenario 2: Well-controlled patient (normal readings)
    Scenario 3: Crisis reading (SBP >= 180)
    Scenario 4: White-coat HTN pattern (high clinic, normal home)
    """
    events = []

    # --- Scenario 1: Morning surge patient (P-M7-SURGE) ---
    # 7 days of morning (high) + evening (lower) readings → surge ≈ 35 mmHg
    pid = f"P-M7-SURGE-{RUN_ID[-6:]}"
    for d in range(7, 0, -1):
        events.append((TOPIC_M7_INPUT, bp_reading(
            pid, 152 + (d % 3), 92, morning_time(d), "MORNING")))
        events.append((TOPIC_M7_INPUT, bp_reading(
            pid, 117 - (d % 2), 74, evening_time(d), "EVENING")))
    # Today's morning reading — triggers surge computation
    events.append((TOPIC_M7_INPUT, bp_reading(
        pid, 155, 94, morning_time(0), "MORNING")))

    # --- Scenario 2: Controlled patient (P-M7-CTRL) ---
    pid2 = f"P-M7-CTRL-{RUN_ID[-6:]}"
    for d in range(5, 0, -1):
        events.append((TOPIC_M7_INPUT, bp_reading(
            pid2, 122 + (d % 3), 78 + (d % 2), morning_time(d), "MORNING")))
        events.append((TOPIC_M7_INPUT, bp_reading(
            pid2, 118 + (d % 2), 75, evening_time(d), "EVENING")))
    events.append((TOPIC_M7_INPUT, bp_reading(
        pid2, 120, 76, morning_time(0), "MORNING")))

    # --- Scenario 3: Crisis reading (P-M7-CRISIS) ---
    # SBP >= 180 should trigger crisis side-output
    pid3 = f"P-M7-CRISIS-{RUN_ID[-6:]}"
    events.append((TOPIC_M7_INPUT, bp_reading(
        pid3, 135, 85, hours_ago_ms(2))))
    events.append((TOPIC_M7_INPUT, bp_reading(
        pid3, 192, 112, now_ms(), heart_rate=105)))

    # --- Scenario 4: White-coat HTN (P-M7-WC) ---
    # High clinic readings, normal home readings → white-coat suspected
    pid4 = f"P-M7-WC-{RUN_ID[-6:]}"
    for d in range(5, 0, -1):
        events.append((TOPIC_M7_INPUT, bp_reading(
            pid4, 155 + (d % 3), 92, morning_time(d), source="CLINIC")))
        events.append((TOPIC_M7_INPUT, bp_reading(
            pid4, 122 + (d % 2), 78, evening_time(d), source="HOME_CUFF")))
    events.append((TOPIC_M7_INPUT, bp_reading(
        pid4, 158, 95, morning_time(0), source="CLINIC")))

    return events


# ---------------------------------------------------------------------------
# Clinical Scenarios — Module 8
# ---------------------------------------------------------------------------
def build_module8_scenarios():
    """
    Build realistic event sequences for 4 CID rule triggers.

    Scenario 1: CID-01 Triple Whammy (ACEI + SGLT2I + Thiazide + weight drop)
    Scenario 2: CID-04 Euglycemic DKA (SGLT2I + keto diet + nausea)
    Scenario 3: CID-06 Thiazide glucose rise (Thiazide + FBG delta >15 mg/dL)
    Scenario 4: CID-15 SGLT2I + NSAID (soft flag)
    """
    events = []

    # --- Scenario 1: Triple Whammy → HALT (P-M8-TW) ---
    pid = f"P-M8-TW-{RUN_ID[-6:]}"
    events.append((TOPIC_M8_INPUT, med_event(pid, "ramipril", "ACEI", 10.0, days_ago_ms(14))))
    events.append((TOPIC_M8_INPUT, med_event(pid, "empagliflozin", "SGLT2I", 10.0, days_ago_ms(14))))
    events.append((TOPIC_M8_INPUT, med_event(pid, "chlorthalidone", "THIAZIDE", 12.5, days_ago_ms(14))))
    # Weight baseline 7 days ago
    events.append((TOPIC_M8_INPUT, vital_event(pid, sbp=130, weight=75.0, event_time=days_ago_ms(7))))
    # Weight drop today (>2kg in 7 days = precipitant)
    events.append((TOPIC_M8_INPUT, vital_event(pid, sbp=128, weight=72.0, event_time=now_ms())))

    # --- Scenario 2: Euglycemic DKA → HALT (P-M8-DKA) ---
    pid2 = f"P-M8-DKA-{RUN_ID[-6:]}"
    events.append((TOPIC_M8_INPUT, med_event(pid2, "dapagliflozin", "SGLT2I", 10.0, days_ago_ms(7))))
    # Nausea reported (Module 8 accepts "NAUSEA" or "VOMITING", not "NAUSEA_VOMITING")
    events.append((TOPIC_M8_INPUT, symptom_event(pid2, "NAUSEA", event_time=hours_ago_ms(6))))
    # Glucose normal-ish (euglycemic DKA = normal glucose + ketosis)
    events.append((TOPIC_M8_INPUT, lab_event(pid2, "glucose", 140.0, now_ms())))

    # --- Scenario 3: Thiazide FBG rise → PAUSE (P-M8-FBG) ---
    pid3 = f"P-M8-FBG-{RUN_ID[-6:]}"
    events.append((TOPIC_M8_INPUT, med_event(pid3, "hydrochlorothiazide", "THIAZIDE", 25.0, days_ago_ms(21))))
    events.append((TOPIC_M8_INPUT, lab_event(pid3, "fbg", 110.0, days_ago_ms(14))))
    # FBG now 130 → delta +20 mg/dL (exceeds 15 threshold)
    events.append((TOPIC_M8_INPUT, lab_event(pid3, "fbg", 130.0, now_ms())))

    # --- Scenario 4: SGLT2I + NSAID → SOFT_FLAG (P-M8-NSAID) ---
    pid4 = f"P-M8-NSAID-{RUN_ID[-6:]}"
    events.append((TOPIC_M8_INPUT, med_event(pid4, "empagliflozin", "SGLT2I", 10.0, days_ago_ms(7))))
    events.append((TOPIC_M8_INPUT, med_event(pid4, "ibuprofen", "NSAID", 400.0, now_ms())))

    return events


# ---------------------------------------------------------------------------
# Kafka Operations
# ---------------------------------------------------------------------------
def produce(topic, event_dict):
    """Send a single-line JSON to Kafka via docker exec."""
    json_line = json.dumps(event_dict, separators=(",", ":"))
    cmd = [
        "docker", "exec", "-i", KAFKA_CONTAINER,
        "kafka-console-producer",
        "--bootstrap-server", KAFKA_BOOTSTRAP_INTERNAL,
        "--topic", topic,
    ]
    result = subprocess.run(cmd, input=json_line, capture_output=True, text=True, timeout=45)
    if result.returncode != 0:
        print(f"  ERROR producing to {topic}: {result.stderr.strip()}")
        return False
    return True


def consume_topic(topic, timeout_sec=20, max_messages=200, pattern=None):
    """Consume from topic. Returns list of parsed JSON dicts (or raw lines)."""
    group = f"e2e-m7m8-{uuid.uuid4().hex[:8]}"
    cmd = [
        "docker", "exec", KAFKA_CONTAINER,
        "kafka-console-consumer",
        "--bootstrap-server", KAFKA_BOOTSTRAP_INTERNAL,
        "--topic", topic,
        "--from-beginning",
        "--group", group,
        "--max-messages", str(max_messages),
        "--timeout-ms", str(timeout_sec * 1000),
    ]
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout_sec + 30)
    except subprocess.TimeoutExpired:
        return []

    lines = result.stdout.strip().split("\n") if result.stdout.strip() else []

    # Filter by RUN_ID if pattern given
    if pattern:
        lines = [l for l in lines if pattern in l]

    parsed = []
    for line in lines:
        try:
            parsed.append(json.loads(line))
        except json.JSONDecodeError:
            pass  # skip non-JSON lines (e.g., consumer info)
    return parsed


def create_topics():
    """Create all topics needed for Module 7-8 e2e."""
    print(f"\n  Creating topics for Module 7-8 e2e...")
    for topic, partitions in ALL_TOPICS.items():
        cmd = [
            "docker", "exec", KAFKA_CONTAINER,
            "kafka-topics",
            "--bootstrap-server", KAFKA_BOOTSTRAP_INTERNAL,
            "--create",
            "--topic", topic,
            "--partitions", str(partitions),
            "--replication-factor", "1",
            "--if-not-exists",
        ]
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=45)
        if result.returncode == 0:
            print(f"    {topic} ({partitions}p) — OK")
        else:
            err = result.stderr.strip()
            if "already exists" in err:
                print(f"    {topic} — already exists")
            else:
                print(f"    {topic} — ERROR: {err}")


def list_topics():
    """List all Kafka topics."""
    cmd = [
        "docker", "exec", KAFKA_CONTAINER,
        "kafka-topics", "--list",
        "--bootstrap-server", KAFKA_BOOTSTRAP_INTERNAL,
    ]
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=45)
    return result.stdout.strip().split("\n") if result.stdout.strip() else []


# ---------------------------------------------------------------------------
# Verification
# ---------------------------------------------------------------------------
def verify_module7(wait_sec=15):
    """Check Module 7 output topics for our test events."""
    print(f"\n{'='*60}")
    print(f"  MODULE 7 VERIFICATION: BP Variability Metrics")
    print(f"{'='*60}")

    # Main output: flink.bp-variability-metrics
    metrics = consume_topic(TOPIC_M7_OUTPUT, timeout_sec=30, max_messages=2000, pattern=RUN_SUFFIX)
    print(f"\n  {TOPIC_M7_OUTPUT}: {len(metrics)} event(s)")

    patients_seen = set()
    crisis_count = 0
    surge_detected = False

    for m in metrics:
        pid = m.get("patient_id", "?")
        patients_seen.add(pid)
        if m.get("crisis_flag"):
            crisis_count += 1
        if m.get("surge_classification") and m["surge_classification"] != "INSUFFICIENT_DATA":
            surge_detected = True
        if len(metrics) <= 20:  # Show detail for small result sets
            print(f"    patient={pid}  sbp={m.get('trigger_sbp')}  "
                  f"crisis={m.get('crisis_flag')}  "
                  f"surge={m.get('surge_classification')}  "
                  f"control={m.get('bp_control_status')}  "
                  f"depth={m.get('context_depth')}")

    # Safety-critical side output
    safety = consume_topic(TOPIC_SAFETY_CRITICAL, timeout_sec=15, max_messages=500, pattern=RUN_SUFFIX)
    safety_bp = [s for s in safety if "systolic" in json.dumps(s)]
    print(f"\n  {TOPIC_SAFETY_CRITICAL} (BP crisis/surge): {len(safety_bp)} event(s)")
    for s in safety_bp[:5]:
        print(f"    patient={s.get('patient_id')}  sbp={s.get('systolic')}  "
              f"dbp={s.get('diastolic')}")

    # Assertions
    print(f"\n  --- Module 7 Results ---")
    results = {}

    results["metrics_emitted"] = len(metrics) > 0
    print(f"  [{'PASS' if results['metrics_emitted'] else 'FAIL'}] "
          f"Metrics emitted: {len(metrics)}")

    results["multiple_patients"] = len(patients_seen) >= 2
    print(f"  [{'PASS' if results['multiple_patients'] else 'FAIL'}] "
          f"Multiple patients: {patients_seen}")

    results["crisis_detected"] = crisis_count > 0 or len(safety_bp) > 0
    print(f"  [{'PASS' if results['crisis_detected'] else '----'}] "
          f"Crisis detection: {crisis_count} flagged, {len(safety_bp)} safety-critical")

    return results


def verify_module8(wait_sec=15):
    """Check Module 8 output topics for our test events."""
    print(f"\n{'='*60}")
    print(f"  MODULE 8 VERIFICATION: Comorbidity Interaction Alerts")
    print(f"{'='*60}")

    # Main output: alerts.comorbidity-interactions
    alerts = consume_topic(TOPIC_M8_OUTPUT, timeout_sec=20, pattern=RUN_SUFFIX)
    print(f"\n  {TOPIC_M8_OUTPUT}: {len(alerts)} alert(s)")

    rules_fired = set()
    halt_count = 0
    pause_count = 0
    soft_count = 0

    for a in alerts:
        rule = a.get("ruleId", "?")
        sev = a.get("severity", "?")
        pid = a.get("patientId", "?")
        rules_fired.add(rule)
        if sev == "HALT":
            halt_count += 1
        elif sev == "PAUSE":
            pause_count += 1
        elif sev == "SOFT_FLAG":
            soft_count += 1
        print(f"    {rule} [{sev}] patient={pid}  "
              f"trigger={a.get('triggerSummary', '')[:60]}")

    # HALT side output
    safety = consume_topic(TOPIC_SAFETY_CRITICAL, timeout_sec=10, pattern=RUN_SUFFIX)
    halt_safety = [s for s in safety if "ruleId" in json.dumps(s)]
    print(f"\n  {TOPIC_SAFETY_CRITICAL} (HALT alerts): {len(halt_safety)} event(s)")
    for s in halt_safety[:5]:
        print(f"    {s.get('ruleId')} [{s.get('severity')}] patient={s.get('patientId')}")

    # Assertions
    print(f"\n  --- Module 8 Results ---")
    results = {}

    results["alerts_emitted"] = len(alerts) > 0
    print(f"  [{'PASS' if results['alerts_emitted'] else 'FAIL'}] "
          f"Alerts emitted: {len(alerts)}")

    results["cid01_fired"] = "CID_01" in rules_fired
    print(f"  [{'PASS' if results['cid01_fired'] else 'FAIL'}] "
          f"CID-01 Triple Whammy: {'fired' if results['cid01_fired'] else 'NOT fired'}")

    results["cid04_fired"] = "CID_04" in rules_fired
    print(f"  [{'PASS' if results['cid04_fired'] else '----'}] "
          f"CID-04 Euglycemic DKA: {'fired' if results['cid04_fired'] else 'NOT fired'}")

    results["cid06_fired"] = "CID_06" in rules_fired
    print(f"  [{'PASS' if results['cid06_fired'] else '----'}] "
          f"CID-06 Thiazide FBG: {'fired' if results['cid06_fired'] else 'NOT fired'}")

    results["cid15_fired"] = "CID_15" in rules_fired
    print(f"  [{'PASS' if results['cid15_fired'] else '----'}] "
          f"CID-15 SGLT2I+NSAID: {'fired' if results['cid15_fired'] else 'NOT fired'}")

    results["halt_via_safety"] = len(halt_safety) > 0 or halt_count > 0
    print(f"  [{'PASS' if results['halt_via_safety'] else '----'}] "
          f"HALT → safety-critical: {halt_count} HALT alerts, {len(halt_safety)} via side-output")

    results["severity_mix"] = halt_count > 0 or (pause_count > 0 and soft_count > 0)
    print(f"  [{'PASS' if results['severity_mix'] else '----'}] "
          f"Severity spread: {halt_count} HALT, {pause_count} PAUSE, {soft_count} SOFT_FLAG")

    print(f"\n  Rules fired: {sorted(rules_fired)}")

    return results


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
def main():
    parser = argparse.ArgumentParser(
        description="Flink E2E Test — Module 7 (BP Variability) + Module 8 (Comorbidity)",
    )
    parser.add_argument("--module7-only", action="store_true", help="Run Module 7 only")
    parser.add_argument("--module8-only", action="store_true", help="Run Module 8 only")
    parser.add_argument("--dry-run", action="store_true", help="Show payloads without sending")
    parser.add_argument("--check", action="store_true", help="Only check output topics")
    parser.add_argument("--create-topics", action="store_true", help="Create Kafka topics and exit")
    parser.add_argument("--wait", type=int, default=20, help="Seconds to wait for Flink processing (default: 20)")
    args = parser.parse_args()

    run_m7 = not args.module8_only
    run_m8 = not args.module7_only

    print(f"\n{'#'*60}")
    print(f"  Flink E2E: Module 7 + Module 8")
    print(f"  Run ID:  {RUN_ID}")
    print(f"  Kafka:   {KAFKA_CONTAINER} @ {KAFKA_BOOTSTRAP_INTERNAL}")
    print(f"  Modules: {'M7' if run_m7 else ''} {'M8' if run_m8 else ''}")
    print(f"{'#'*60}")

    if args.create_topics:
        create_topics()
        return

    if args.check:
        if run_m7:
            verify_module7()
        if run_m8:
            verify_module8()
        return

    # Verify Kafka container is running
    result = subprocess.run(
        ["docker", "inspect", "--format", "{{.State.Running}}", KAFKA_CONTAINER],
        capture_output=True, text=True, timeout=30,
    )
    if result.stdout.strip() != "true":
        print(f"\n  ERROR: {KAFKA_CONTAINER} is not running!")
        print(f"  Start: cd ../kafka && docker compose -f docker-compose.hpi-lite.yml up -d")
        sys.exit(1)
    print(f"\n  Kafka container: running")

    # Create topics if missing
    existing = list_topics()
    missing = [t for t in ALL_TOPICS if t not in existing]
    if missing:
        print(f"  Missing topics: {missing}")
        create_topics()
    else:
        print(f"  All topics exist")

    # -----------------------------------------------------------------------
    # MODULE 7: Produce BP readings
    # -----------------------------------------------------------------------
    if run_m7:
        m7_events = build_module7_scenarios()
        print(f"\n{'='*60}")
        print(f"  MODULE 7: Producing {len(m7_events)} BP readings")
        print(f"{'='*60}")

        if args.dry_run:
            for i, (topic, evt) in enumerate(m7_events[:5]):
                print(f"\n  [{i+1}] {topic}:")
                print(f"  {json.dumps(evt, indent=2)[:300]}")
            print(f"\n  ... ({len(m7_events)} total, dry-run)")
        else:
            ok = fail = 0
            for i, (topic, evt) in enumerate(m7_events, 1):
                if produce(topic, evt):
                    ok += 1
                else:
                    fail += 1
                if i % 10 == 0 or i == len(m7_events):
                    print(f"    Sent {i}/{len(m7_events)} ({ok} ok, {fail} fail)")
                if i % 5 == 0:
                    time.sleep(0.2)
            print(f"  Module 7 input: {ok} sent, {fail} failed")

    # -----------------------------------------------------------------------
    # MODULE 8: Produce CanonicalEvents
    # -----------------------------------------------------------------------
    if run_m8:
        m8_events = build_module8_scenarios()
        print(f"\n{'='*60}")
        print(f"  MODULE 8: Producing {len(m8_events)} canonical events")
        print(f"{'='*60}")

        if args.dry_run:
            for i, (topic, evt) in enumerate(m8_events[:5]):
                print(f"\n  [{i+1}] {topic}:")
                print(f"  {json.dumps(evt, indent=2)[:300]}")
            print(f"\n  ... ({len(m8_events)} total, dry-run)")
        else:
            ok = fail = 0
            for i, (topic, evt) in enumerate(m8_events, 1):
                if produce(topic, evt):
                    ok += 1
                else:
                    fail += 1
                # Small delay between events for temporal ordering
                time.sleep(0.3)
            print(f"  Module 8 input: {ok} sent, {fail} failed")

    if args.dry_run:
        print(f"\n  [DRY RUN] No events sent to Kafka.")
        return

    # -----------------------------------------------------------------------
    # Wait for Flink processing
    # -----------------------------------------------------------------------
    print(f"\n  Waiting {args.wait}s for Flink to process events...")
    time.sleep(args.wait)

    # -----------------------------------------------------------------------
    # Verify output
    # -----------------------------------------------------------------------
    all_results = {}
    if run_m7:
        all_results["module7"] = verify_module7()
    if run_m8:
        all_results["module8"] = verify_module8()

    # -----------------------------------------------------------------------
    # Summary
    # -----------------------------------------------------------------------
    print(f"\n{'#'*60}")
    print(f"  E2E SUMMARY — Run ID: {RUN_ID}")
    print(f"{'#'*60}")

    total_pass = 0
    total_checks = 0
    for module, results in all_results.items():
        for check, passed in results.items():
            total_checks += 1
            if passed:
                total_pass += 1
            marker = "PASS" if passed else "----"
            print(f"  [{marker}] {module}.{check}")

    print(f"\n  Score: {total_pass}/{total_checks} checks passed")

    if total_pass == 0:
        print(f"\n  No output detected. Troubleshooting:")
        print(f"    1. Check Flink UI:  curl http://localhost:8181/jobs/overview")
        print(f"    2. Check logs:      docker logs cardiofit-flink-taskmanager 2>&1 | tail -30")
        print(f"    3. List topics:     docker exec {KAFKA_CONTAINER} kafka-topics --list --bootstrap-server {KAFKA_BOOTSTRAP_INTERNAL}")
        print(f"    4. Verify jobs submitted for module7 / module8")

    # Save results
    output_file = f"test-data/e2e-module7-module8-{RUN_ID}.json"
    try:
        with open(output_file, "w") as f:
            json.dump({
                "run_id": RUN_ID,
                "timestamp": datetime.now(timezone.utc).isoformat(),
                "results": {k: {ck: cv for ck, cv in v.items()} for k, v in all_results.items()},
                "total_pass": total_pass,
                "total_checks": total_checks,
            }, f, indent=2)
        print(f"\n  Results saved to: {output_file}")
    except Exception:
        pass  # Non-critical — don't fail on file write


if __name__ == "__main__":
    main()
