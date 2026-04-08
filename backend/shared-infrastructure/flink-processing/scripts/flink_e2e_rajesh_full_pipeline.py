#!/usr/bin/env python3
"""
Flink FULL PIPELINE E2E: Rajesh Kumar — Module 7 → 8 → 9 → 10 → 11 → 13
=========================================================================
Produces to the REAL upstream input topics and verifies output flows
through ALL intermediate modules to Module 13's output.

Pipeline:
  ingestion.vitals          → Module 7  → flink.bp-variability-metrics       ─┐
  enriched-patient-events-v1 → Module 8  → alerts.comorbidity-interactions    ─┤
  enriched-patient-events-v1 → Module 9  → flink.engagement-signals           ─┤→ Module 13
  enriched-patient-events-v1 → Module 10 → flink.meal-response → M10b         ─┤   → clinical.state-change-events
  enriched-patient-events-v1 → Module 11 → flink.activity-response → M11b     ─┘

Patient: Rajesh Kumar, 58M, T2DM, HTN Stage 2, CKD 3b
"""

import json
import subprocess
import sys
import time
import uuid

KAFKA = "cardiofit-kafka-lite"
BOOTSTRAP = "localhost:29092"
PATIENT = "e2e-rajesh-kumar-full-002"
RUN_ID = f"e2e-full-{int(time.time())}"
TS = int(time.time() * 1000)


def publish(topic, msg):
    j = json.dumps(msg, separators=(",", ":"))
    r = subprocess.run(
        ["docker", "exec", "-i", KAFKA, "kafka-console-producer",
         "--bootstrap-server", BOOTSTRAP, "--topic", topic],
        input=j, capture_output=True, text=True, timeout=15)
    return r.returncode == 0


def consume(topic, timeout=15, max_msgs=200):
    r = subprocess.run(
        ["docker", "exec", KAFKA, "kafka-console-consumer",
         "--bootstrap-server", BOOTSTRAP, "--topic", topic,
         "--from-beginning", "--timeout-ms", str(timeout * 1000),
         "--max-messages", str(max_msgs)],
        capture_output=True, text=True, timeout=timeout + 10)
    msgs = []
    for line in r.stdout.strip().split("\n"):
        if not line.strip():
            continue
        try:
            msgs.append(json.loads(line.strip()))
        except json.JSONDecodeError:
            pass
    return msgs


def topic_count(topic):
    r = subprocess.run(
        ["docker", "exec", KAFKA, "kafka-run-class",
         "kafka.tools.GetOffsetShell", "--broker-list", "localhost:9092",
         "--topic", topic, "--time", "-1"],
        capture_output=True, text=True, timeout=10)
    total = 0
    for line in r.stdout.strip().split("\n"):
        parts = line.split(":")
        if len(parts) >= 3:
            total += int(parts[2])
    return total


# ═══════════════════════════════════════════════════════════════════
#  INPUT EVENTS
# ═══════════════════════════════════════════════════════════════════

# 1. BPReading for Module 7 (ingestion.vitals)
bp_readings = []
for i, (sbp, dbp, ctx) in enumerate([
    (172, 102, "MORNING"), (168, 100, "EVENING"),
    (175, 104, "MORNING"), (164, 98, "EVENING"),
    (180, 106, "MORNING"), (160, 96, "EVENING"),  # morning surge pattern
]):
    bp_readings.append({
        "patient_id": PATIENT,
        "systolic": sbp,
        "diastolic": dbp,
        "heart_rate": 78 + i,
        "timestamp": TS - (6 - i) * 43200000,  # every 12h over 3 days
        "time_context": ctx,
        "source": "HOME_CUFF",
        "position": "SEATED",
        "device_type": "oscillometric_cuff",
        "correlation_id": RUN_ID,
    })

# 2. EnrichedEvent for Modules 8/9/10/11 (enriched-patient-events-v1)
enriched_events = []

# Lab: high FBG
enriched_events.append({
    "eventId": str(uuid.uuid4()),
    "patientId": PATIENT,
    "eventType": "LAB_RESULT",
    "timestamp": TS,
    "sourceSystem": "flink-e2e-full-pipeline",
    "correlationId": RUN_ID,
    "payload": {
        "lab_type": "FBG",
        "value": 185.0,
        "unit": "mg/dL",
        "testName": "Fasting Blood Glucose",
        "results": {"glucose": 185.0},
    },
    "enrichmentData": {},
    "enrichmentVersion": "1.0",
})

# Vital sign: elevated BP (for Module 8 comorbidity context)
enriched_events.append({
    "eventId": str(uuid.uuid4()),
    "patientId": PATIENT,
    "eventType": "VITAL_SIGN",
    "timestamp": TS + 1000,
    "sourceSystem": "flink-e2e-full-pipeline",
    "correlationId": RUN_ID,
    "payload": {
        "systolicBP": 172,
        "diastolicBP": 102,
        "heartRate": 82,
    },
    "enrichmentData": {},
    "enrichmentVersion": "1.0",
})

# Medication: Metformin (triggers CID rules in Module 8)
enriched_events.append({
    "eventId": str(uuid.uuid4()),
    "patientId": PATIENT,
    "eventType": "MEDICATION_ORDERED",
    "timestamp": TS + 2000,
    "sourceSystem": "flink-e2e-full-pipeline",
    "correlationId": RUN_ID,
    "payload": {
        "medicationName": "Metformin",
        "dose": 1000,
        "doseUnit": "mg",
        "route": "oral",
        "frequency": "BD",
        "status": "active",
    },
    "enrichmentData": {},
    "enrichmentVersion": "1.0",
})

# Patient-reported: meal + activity data (for Modules 10, 11)
enriched_events.append({
    "eventId": str(uuid.uuid4()),
    "patientId": PATIENT,
    "eventType": "PATIENT_REPORTED",
    "timestamp": TS + 3000,
    "sourceSystem": "flink-e2e-full-pipeline",
    "correlationId": RUN_ID,
    "payload": {
        "type": "meal_log",
        "meal_type": "lunch",
        "carb_estimate_g": 65,
        "pre_meal_glucose": 145,
        "post_meal_glucose": 210,
        "sodium_mg": 2200,
    },
    "enrichmentData": {},
    "enrichmentVersion": "1.0",
})

enriched_events.append({
    "eventId": str(uuid.uuid4()),
    "patientId": PATIENT,
    "eventType": "DEVICE_READING",
    "timestamp": TS + 4000,
    "sourceSystem": "flink-e2e-full-pipeline",
    "correlationId": RUN_ID,
    "payload": {
        "type": "activity",
        "steps": 3200,
        "active_minutes": 22,
        "calories_burned": 180,
        "heart_rate_avg": 95,
        "heart_rate_max": 118,
    },
    "enrichmentData": {},
    "enrichmentVersion": "1.0",
})

# Lab: eGFR (kidney function — CKD marker)
enriched_events.append({
    "eventId": str(uuid.uuid4()),
    "patientId": PATIENT,
    "eventType": "LAB_RESULT",
    "timestamp": TS + 5000,
    "sourceSystem": "flink-e2e-full-pipeline",
    "correlationId": RUN_ID,
    "payload": {
        "lab_type": "EGFR",
        "value": 42.0,
        "unit": "mL/min/1.73m2",
        "testName": "eGFR",
        "results": {"egfr": 42.0},
    },
    "enrichmentData": {},
    "enrichmentVersion": "1.0",
})

all_input = {
    "bp_readings_to_ingestion_vitals": bp_readings,
    "enriched_events_to_enriched_patient_events": enriched_events,
}


# ═══════════════════════════════════════════════════════════════════
#  EXECUTE
# ═══════════════════════════════════════════════════════════════════

def main():
    print(f"{'='*65}")
    print(f"  FULL PIPELINE E2E: Rajesh Kumar — Module 7 → 13")
    print(f"  Run ID:  {RUN_ID}")
    print(f"  Patient: {PATIENT}")
    print(f"{'='*65}")

    # Snapshot topic counts before
    print(f"\n  ── Pre-test topic counts ──")
    topics_to_check = [
        ("ingestion.vitals", "Module 7 input"),
        ("flink.bp-variability-metrics", "Module 7 → M13"),
        ("enriched-patient-events-v1", "Module 8/9/10/11 input"),
        ("alerts.comorbidity-interactions", "Module 8 → M13"),
        ("flink.engagement-signals", "Module 9 → M13"),
        ("flink.meal-response", "Module 10 → M10b"),
        ("flink.meal-patterns", "Module 10b → M13"),
        ("flink.activity-response", "Module 11 → M11b"),
        ("flink.fitness-patterns", "Module 11b → M13"),
        ("clinical.state-change-events", "Module 13 output"),
    ]
    before = {}
    for t, label in topics_to_check:
        c = topic_count(t)
        before[t] = c
        print(f"    {t}: {c}")

    # ── Produce BP readings to ingestion.vitals ──
    print(f"\n  ── Phase 1: Producing {len(bp_readings)} BP readings → ingestion.vitals ──")
    bp_ok = 0
    for i, bp in enumerate(bp_readings):
        if publish("ingestion.vitals", bp):
            bp_ok += 1
            print(f"    ✓ BP {bp['systolic']}/{bp['diastolic']} ({bp['time_context']})")
        else:
            print(f"    ✗ FAILED BP reading #{i}")
    print(f"    Sent: {bp_ok}/{len(bp_readings)}")

    # ── Produce enriched events ──
    print(f"\n  ── Phase 2: Producing {len(enriched_events)} enriched events ──")
    ee_ok = 0
    for ev in enriched_events:
        etype = ev["eventType"]
        detail = ""
        if etype == "LAB_RESULT":
            detail = f"{ev['payload'].get('lab_type','?')}={ev['payload'].get('value','?')}"
        elif etype == "VITAL_SIGN":
            detail = f"BP {ev['payload'].get('systolicBP','?')}/{ev['payload'].get('diastolicBP','?')}"
        elif etype == "MEDICATION_ORDERED":
            detail = ev['payload'].get('medicationName', '?')
        elif etype == "PATIENT_REPORTED":
            detail = ev['payload'].get('type', '?')
        elif etype == "DEVICE_READING":
            detail = f"steps={ev['payload'].get('steps','?')}"

        if publish("enriched-patient-events-v1", ev):
            ee_ok += 1
            print(f"    ✓ {etype}: {detail}")
        else:
            print(f"    ✗ FAILED {etype}")
    print(f"    Sent: {ee_ok}/{len(enriched_events)}")

    total_sent = bp_ok + ee_ok

    # ── Wait for full pipeline processing ──
    print(f"\n{'='*65}")
    print(f"  Waiting 45s for full pipeline: M7→M8→M9→M10→M11→M13")
    print(f"{'='*65}")
    time.sleep(45)

    # ── Check all intermediate + output topics ──
    print(f"\n  ── Post-test topic counts (delta) ──")
    results = {}
    for t, label in topics_to_check:
        after = topic_count(t)
        delta = after - before.get(t, 0)
        results[t] = {"before": before.get(t, 0), "after": after, "delta": delta}
        marker = "✓" if delta > 0 else "—"
        print(f"    {marker} {label:35s} {t:45s} +{delta}")

    # ── Consume Module 13 output ──
    print(f"\n  ── Module 13 Output: clinical.state-change-events ──")
    all_msgs = consume("clinical.state-change-events", timeout=10)
    rajesh = [m for m in all_msgs
              if m.get("patient_id") == PATIENT or m.get("patientId") == PATIENT]
    print(f"    Total messages: {len(all_msgs)}")
    print(f"    Rajesh Kumar:   {len(rajesh)}")

    for i, m in enumerate(rajesh, 1):
        print(f"\n    ── Output #{i} ──")
        print(f"    change_type:        {m.get('change_type','?')}")
        print(f"    priority:           {m.get('priority','?')}")
        print(f"    recommended_action: {m.get('recommended_action','?')}")
        print(f"    data_completeness:  {m.get('data_completeness_at_change',0):.3f}")
        print(f"    trigger_module:     {m.get('trigger_module','?')}")

    # ── Save full I/O JSON ──
    output_json = {
        "test_run": RUN_ID,
        "patient_id": PATIENT,
        "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "pipeline": "Module 7 → 8 → 9 → 10/10b → 11/11b → 13 (FULL)",
        "input_events": {
            "ingestion_vitals": bp_readings,
            "enriched_patient_events": [
                {k: v for k, v in ev.items()} for ev in enriched_events
            ],
        },
        "intermediate_topic_deltas": {
            t: results[t] for t, _ in topics_to_check
        },
        "output_events": rajesh,
    }

    out_path = "/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/flink-processing/test-data/e2e-rajesh-kumar-full-pipeline.json"
    with open(out_path, "w") as f:
        json.dump(output_json, f, indent=2, default=str)
    print(f"\n  Saved I/O JSON → {out_path}")

    # ── Summary ──
    print(f"\n{'='*65}")
    print(f"  SUMMARY")
    print(f"{'='*65}")
    print(f"  Events produced:  {total_sent} ({bp_ok} BP + {ee_ok} enriched)")
    print(f"  M7 output delta:  +{results.get('flink.bp-variability-metrics',{}).get('delta',0)}")
    print(f"  M8 output delta:  +{results.get('alerts.comorbidity-interactions',{}).get('delta',0)}")
    print(f"  M9 output delta:  +{results.get('flink.engagement-signals',{}).get('delta',0)}")
    print(f"  M10b output delta:+{results.get('flink.meal-patterns',{}).get('delta',0)}")
    print(f"  M11b output delta:+{results.get('flink.fitness-patterns',{}).get('delta',0)}")
    print(f"  M13 output delta: +{results.get('clinical.state-change-events',{}).get('delta',0)}")
    print(f"  Rajesh state changes: {len(rajesh)}")

    if results.get("clinical.state-change-events", {}).get("delta", 0) > 0:
        print(f"\n  ✓ FULL PIPELINE FLOWING: Module 7 → 13")
    elif any(results.get(t, {}).get("delta", 0) > 0 for t in
             ["flink.bp-variability-metrics", "alerts.comorbidity-interactions",
              "flink.engagement-signals"]):
        print(f"\n  ◐ PARTIAL: Some intermediate modules produced output")
        print(f"    Module 13 may need more time or events for state change emission")
    else:
        print(f"\n  ⚠ No intermediate output detected")
        print(f"    Check module logs: docker logs cardiofit-flink-taskmanager 2>&1 | grep ERROR")

    return 0


if __name__ == "__main__":
    sys.exit(main())
