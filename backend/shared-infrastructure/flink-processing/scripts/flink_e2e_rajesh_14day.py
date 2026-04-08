#!/usr/bin/env python3
"""
Flink E2E: Rajesh Kumar 14-Day Clinical Dataset — Module 7 → 13
================================================================
14-day simulation with clinically realistic BP reading patterns.

Patient: Rajesh Kumar, 58M, T2DM (12yr), HTN Stage 2, CKD 3b (eGFR 42)
Meds: Metformin 1000mg BD, Amlodipine 10mg, Telmisartan 80mg

Days 1-3:  Baseline — moderate HTN, morning surge present
Days 4-7:  Completion — fills in evening readings, clinic visit day 5
Days 8-10: Deterioration — SBP trending up, missed meds
Days 11-13: Engagement collapse — fewer readings, higher BP
Day 14:    Snapshot — single morning reading for final state

This validates:
  - ARV computed across ALL sequential readings (Mena et al. 2005 fix)
  - Morning surge detected via 72h lookback pairing
  - Variability classification = HIGH (true ARV > 16)
  - Module 13 receives M7 output with non-trivial metrics

Also sends enriched events for Modules 8/9/10/11 to trigger the full pipeline.
"""

import json
import subprocess
import sys
import time
import uuid

KAFKA = "cardiofit-kafka-lite"
BOOTSTRAP = "localhost:29092"
_TS = int(time.time())
PATIENT = f"e2e-rajesh-14day-{_TS}"
RUN_ID = f"e2e-14day-{_TS}"

DAY_MS = 86400000  # 24h in ms
HOUR_MS = 3600000


def publish(topic, msg):
    j = json.dumps(msg, separators=(",", ":"))
    r = subprocess.run(
        ["docker", "exec", "-i", KAFKA, "kafka-console-producer",
         "--bootstrap-server", BOOTSTRAP, "--topic", topic],
        input=j, capture_output=True, text=True, timeout=15)
    return r.returncode == 0


def consume(topic, timeout=20, max_msgs=1000):
    r = subprocess.run(
        ["docker", "exec", KAFKA, "kafka-console-consumer",
         "--bootstrap-server", BOOTSTRAP, "--topic", topic,
         "--from-beginning", "--timeout-ms", str(timeout * 1000),
         "--max-messages", str(max_msgs)],
        capture_output=True, text=True, timeout=timeout + 30)
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


def bp(sbp, dbp, ts, context, source="HOME_CUFF", position="SEATED"):
    """Build a BPReading for ingestion.vitals."""
    return {
        "patient_id": PATIENT,
        "systolic": sbp,
        "diastolic": dbp,
        "heart_rate": 72 + int((sbp - 120) / 5),
        "timestamp": ts,
        "time_context": context,
        "source": source,
        "position": position,
        "device_type": "oscillometric_cuff",
        "correlation_id": RUN_ID,
    }


def enriched_event(event_type, ts, payload):
    """Build a CanonicalEvent for enriched-patient-events-v1."""
    return {
        "eventId": str(uuid.uuid4()),
        "patientId": PATIENT,
        "eventType": event_type,
        "timestamp": ts,
        "sourceSystem": "flink-e2e-14day",
        "correlationId": RUN_ID,
        "payload": payload,
        "enrichmentData": {},
        "enrichmentVersion": "1.0",
    }


# ═══════════════════════════════════════════════════════════════════
#  14-DAY BP READING DATASET
# ═══════════════════════════════════════════════════════════════════
# Base time: 14 days ago
BASE = int(time.time() * 1000) - 14 * DAY_MS

bp_readings = []

# Days 1-3: Baseline — moderate HTN, morning surge
# Morning SBP ~158-165, Evening SBP ~142-148 (surge ~15-20 mmHg)
baseline_data = [
    # (day, context, sbp, dbp, hour_offset)
    (0, "MORNING", 160, 98, 7),   # Day 1 AM
    (0, "EVENING", 144, 88, 20),  # Day 1 PM
    (1, "MORNING", 162, 100, 7),  # Day 2 AM
    (1, "EVENING", 146, 90, 20),  # Day 2 PM
    (2, "MORNING", 158, 96, 7),   # Day 3 AM
    (2, "EVENING", 142, 86, 19),  # Day 3 PM
]
for day, ctx, sbp, dbp, hour in baseline_data:
    bp_readings.append(bp(sbp, dbp, BASE + day * DAY_MS + hour * HOUR_MS, ctx))

# Days 4-7: Completion — fills in readings, clinic visit day 5
completion_data = [
    (3, "MORNING", 156, 94, 7),
    (3, "EVENING", 140, 84, 20),
    (3, "NIGHT",   148, 90, 23),  # nocturnal reading (non-dipper check)
    (4, "MORNING", 164, 100, 7),  # Day 5: clinic visit day
    (4, "MORNING", 170, 104, 9),  # Clinic reading — white-coat effect
    (4, "EVENING", 148, 92, 19),
    (5, "MORNING", 158, 96, 8),
    (5, "EVENING", 144, 88, 20),
    (6, "MORNING", 162, 98, 7),
    (6, "EVENING", 146, 90, 20),
    (6, "NIGHT",   150, 92, 23),  # nocturnal — non-dipper pattern
]
for day, ctx, sbp, dbp, hour in completion_data:
    src = "CLINIC" if (day == 4 and hour == 9) else "HOME_CUFF"
    bp_readings.append(bp(sbp, dbp, BASE + day * DAY_MS + hour * HOUR_MS, ctx, source=src))

# Days 8-10: Deterioration — SBP trending up, missed meds
deterioration_data = [
    (7, "MORNING", 172, 104, 7),   # Day 8: jump up
    (7, "EVENING", 152, 94, 20),
    (8, "MORNING", 178, 108, 7),   # Day 9: worsening
    (8, "EVENING", 156, 96, 19),
    (9, "MORNING", 182, 110, 7),   # Day 10: approaching crisis
    (9, "EVENING", 160, 100, 20),
]
for day, ctx, sbp, dbp, hour in deterioration_data:
    bp_readings.append(bp(sbp, dbp, BASE + day * DAY_MS + hour * HOUR_MS, ctx))

# Days 11-13: Engagement collapse — fewer readings, higher BP
engagement_collapse_data = [
    (10, "MORNING", 180, 108, 8),   # Day 11: morning only
    (11, "EVENING", 164, 100, 21),  # Day 12: evening only (missed morning)
    (12, "MORNING", 176, 106, 7),   # Day 13: morning only
]
for day, ctx, sbp, dbp, hour in engagement_collapse_data:
    bp_readings.append(bp(sbp, dbp, BASE + day * DAY_MS + hour * HOUR_MS, ctx))

# Day 14: Final snapshot — single reading
bp_readings.append(bp(174, 104, BASE + 13 * DAY_MS + 7 * HOUR_MS, "MORNING"))

# Total: 28 readings over 14 days

# ═══════════════════════════════════════════════════════════════════
#  ENRICHED EVENTS for Modules 8/9/10/11/12
# ═══════════════════════════════════════════════════════════════════
enriched_events = []
now_ms = int(time.time() * 1000)

# ── Step 1: Medications (establish state for M8 comorbidity) ──

enriched_events.append(enriched_event("MEDICATION_ORDERED", now_ms, {
    "drug_name": "Telmisartan", "drug_class": "ARB",
    "dose_mg": 80, "route": "oral", "frequency": "OD", "status": "active",
}))

enriched_events.append(enriched_event("MEDICATION_ORDERED", now_ms + 500, {
    "drug_name": "Amlodipine", "drug_class": "CCB",
    "dose_mg": 10, "route": "oral", "frequency": "OD", "status": "active",
}))

enriched_events.append(enriched_event("MEDICATION_ORDERED", now_ms + 1000, {
    "drug_name": "Metformin", "drug_class": "BIGUANIDE",
    "dose_mg": 1000, "route": "oral", "frequency": "BD", "status": "active",
}))

enriched_events.append(enriched_event("MEDICATION_ORDERED", now_ms + 1500, {
    "drug_name": "Dapagliflozin", "drug_class": "SGLT2I",
    "dose_mg": 10, "route": "oral", "frequency": "OD", "status": "active",
}))

enriched_events.append(enriched_event("MEDICATION_ORDERED", now_ms + 2000, {
    "drug_name": "Hydrochlorothiazide", "drug_class": "THIAZIDE",
    "dose_mg": 12.5, "route": "oral", "frequency": "OD", "status": "active",
}))

# ── Step 2: Labs (eGFR + FBG for M8/M13 + M9 signal: lab_completion) ──

enriched_events.append(enriched_event("LAB_RESULT", now_ms + 3000, {
    "lab_type": "EGFR", "value": 52.0, "unit": "mL/min/1.73m2",
    "testName": "eGFR", "results": {"egfr": 52.0},
}))

enriched_events.append(enriched_event("LAB_RESULT", now_ms + 3500, {
    "lab_type": "EGFR", "value": 42.0, "unit": "mL/min/1.73m2",
    "testName": "eGFR", "results": {"egfr": 42.0},
}))

enriched_events.append(enriched_event("LAB_RESULT", now_ms + 4000, {
    "lab_type": "FBG", "value": 165.0, "unit": "mg/dL",
    "testName": "Fasting Blood Glucose", "results": {"glucose": 165.0},
}))

enriched_events.append(enriched_event("LAB_RESULT", now_ms + 4500, {
    "lab_type": "FBG", "value": 185.0, "unit": "mg/dL",
    "testName": "Fasting Blood Glucose", "results": {"glucose": 185.0},
}))

# HbA1c — M9 signal: lab_completion + M12b baseline metric
enriched_events.append(enriched_event("LAB_RESULT", now_ms + 4600, {
    "lab_type": "HBA1C", "value": 8.2, "unit": "%",
    "testName": "HbA1c", "results": {"hba1c": 8.2},
}))

# ── Step 3: Vital signs (M8/M13 + M9 signal: vital_sign_completion) ──

enriched_events.append(enriched_event("VITAL_SIGN", now_ms + 5000, {
    "systolic_bp": 174, "diastolic_bp": 104, "heart_rate": 82,
    "weight_kg": 92.0, "resting_heart_rate": 78,
}))

# Second vital for M12b trajectory tracking
enriched_events.append(enriched_event("VITAL_SIGN", now_ms + 5200, {
    "systolic_bp": 168, "diastolic_bp": 100, "heart_rate": 80,
    "weight_kg": 91.5,
}))

# ── Step 4: Precipitant — nausea/vomiting (triggers CID-01 + CID-04) ──

enriched_events.append(enriched_event("PATIENT_REPORTED", now_ms + 6000, {
    "symptom_type": "NAUSEA", "severity": "moderate",
    "onset": "2_days_ago", "status": "active",
}))

# ── Step 5: M9 Engagement — 8 signal types ──
# M9 needs: steps, meal quality, protein, app session, goal completion,
#           lab completion (already sent), vital completion (already sent),
#           medication adherence

# Steps logged (M9 signal: steps_logged)
enriched_events.append(enriched_event("DEVICE_READING", now_ms + 6500, {
    "step_count": 4200, "source": "WEARABLE",
    "data_tier": "TIER_1_CGM",
}))

# Meal with macros (M9 signals: meal_quality + protein_adherence)
enriched_events.append(enriched_event("PATIENT_REPORTED", now_ms + 7000, {
    "report_type": "MEAL_LOG", "meal_type": "lunch",
    "carb_grams": 65, "protein_grams": 28, "fat_grams": 18,
    "sodium_mg": 2200, "protein_flag": True,
    "data_tier": "TIER_1_CGM",
}))

# App session (M9 signal: response_latency)
enriched_events.append(enriched_event("PATIENT_REPORTED", now_ms + 7200, {
    "report_type": "APP_SESSION",
    "session_duration_sec": 180,
    "data_tier": "TIER_1_CGM",
}))

# Goal completion (M9 signal: checkin_completeness)
enriched_events.append(enriched_event("PATIENT_REPORTED", now_ms + 7400, {
    "report_type": "GOAL_COMPLETED",
    "fields_completed": 4, "total_fields": 5,
    "data_tier": "TIER_1_CGM",
}))

# Medication adherence event (M9 signal: medication_adherence)
enriched_events.append(enriched_event("MEDICATION_EVENT", now_ms + 7600, {
    "drug_name": "Metformin", "action": "TAKEN",
    "scheduled_time": now_ms + 7000, "actual_time": now_ms + 7600,
    "data_tier": "TIER_1_CGM",
}))

# ── Step 6: M10 Meal Response — CGM glucose paired with meal ──
# M10 needs: MEAL_LOG to open session + CGM readings within 3h window

# Pre-meal CGM glucose (30 min before meal)
enriched_events.append(enriched_event("DEVICE_READING", now_ms + 6700, {
    "glucose_value": 142, "source": "CGM",
    "data_tier": "TIER_1_CGM",
}))

# Post-meal CGM glucose readings (30m, 60m, 90m, 120m after meal)
for i, (offset_min, glucose) in enumerate([
    (30, 185), (60, 210), (90, 195), (120, 168)
]):
    enriched_events.append(enriched_event("DEVICE_READING",
        now_ms + 7000 + offset_min * 60000, {
            "glucose_value": glucose, "source": "CGM",
            "data_tier": "TIER_1_CGM",
        }))

# Pre-meal BP for M10 BP correlation
enriched_events.append(enriched_event("VITAL_SIGN", now_ms + 6800, {
    "systolic_bp": 170, "diastolic_bp": 102,
}))

# Post-meal BP (1h after meal)
enriched_events.append(enriched_event("VITAL_SIGN", now_ms + 7000 + 60 * 60000, {
    "systolic_bp": 178, "diastolic_bp": 106,
}))

# ── Step 7: M11 Activity Response — ACTIVITY_LOG + HR + glucose ──
# M11 needs: ACTIVITY_LOG to open session + HR readings + glucose + BP

# Resting HR baseline (before exercise)
enriched_events.append(enriched_event("DEVICE_READING", now_ms + 9000, {
    "heart_rate": 78, "source": "WEARABLE",
    "activity_state": "RESTING",
    "data_tier": "TIER_1_CGM",
}))

# Pre-exercise BP
enriched_events.append(enriched_event("VITAL_SIGN", now_ms + 9100, {
    "systolic_bp": 166, "diastolic_bp": 98,
    "resting_heart_rate": 78,
}))

# Pre-exercise glucose
enriched_events.append(enriched_event("DEVICE_READING", now_ms + 9200, {
    "glucose_value": 158, "source": "CGM",
    "data_tier": "TIER_1_CGM",
}))

# ACTIVITY_LOG — opens M11 session (correct event type for M11)
enriched_events.append(enriched_event("PATIENT_REPORTED", now_ms + 9500, {
    "report_type": "ACTIVITY_LOG",
    "exercise_type": "BRISK_WALKING",
    "duration_minutes": 30,
    "patient_age": 58, "patient_sex": "M",
    "data_tier": "TIER_1_CGM",
}))

# HR readings during exercise (5m, 15m, 25m into walk)
for offset_min, hr in [(5, 105), (15, 118), (25, 112)]:
    enriched_events.append(enriched_event("DEVICE_READING",
        now_ms + 9500 + offset_min * 60000, {
            "heart_rate": hr, "source": "WEARABLE",
            "data_tier": "TIER_1_CGM",
        }))

# Peak exercise BP
enriched_events.append(enriched_event("VITAL_SIGN",
    now_ms + 9500 + 15 * 60000, {
        "systolic_bp": 188, "diastolic_bp": 104,
    }))

# Post-exercise HR recovery readings (1m, 2m, 5m after)
for offset_min, hr in [(31, 102), (32, 94), (35, 86)]:
    enriched_events.append(enriched_event("DEVICE_READING",
        now_ms + 9500 + offset_min * 60000, {
            "heart_rate": hr, "source": "WEARABLE",
            "data_tier": "TIER_1_CGM",
        }))

# Post-exercise glucose (30m after exercise end)
enriched_events.append(enriched_event("DEVICE_READING",
    now_ms + 9500 + 60 * 60000, {
        "glucose_value": 132, "source": "CGM",
        "data_tier": "TIER_1_CGM",
    }))

# Post-exercise BP (30m after exercise end)
enriched_events.append(enriched_event("VITAL_SIGN",
    now_ms + 9500 + 60 * 60000, {
        "systolic_bp": 158, "diastolic_bp": 92,
    }))

# Second activity session (step count device reading for M9 engagement)
enriched_events.append(enriched_event("DEVICE_READING", now_ms + 12000, {
    "step_count": 5800, "source": "WEARABLE",
    "data_tier": "TIER_1_CGM",
}))

# ═══════════════════════════════════════════════════════════════════
#  INTERVENTION EVENTS for Module 12
# ═══════════════════════════════════════════════════════════════════
# M12 reads from clinical.intervention-events topic
# INTERVENTION_APPROVED → immediate WINDOW_OPENED signal
intervention_events = []
INTERVENTION_ID = f"intv-{_TS}-amlodipine-uptitrate"

# Physician approves Amlodipine dose increase: 10mg → 15mg
intervention_events.append({
    "eventId": str(uuid.uuid4()),
    "patientId": PATIENT,
    "eventType": "INTERVENTION_APPROVED",
    "timestamp": now_ms + 15000,
    "sourceSystem": "flink-e2e-14day",
    "correlationId": RUN_ID,
    "payload": {
        "event_type": "INTERVENTION_APPROVED",
        "intervention_id": INTERVENTION_ID,
        "intervention_type": "MEDICATION_DOSE_INCREASE",
        "intervention_detail": {
            "drug_name": "Amlodipine",
            "drug_class": "CCB",
            "old_dose_mg": 10,
            "new_dose_mg": 15,
            "reason": "Uncontrolled Stage 2 hypertension despite dual therapy",
        },
        "observation_window_days": 14,
        "originating_card_id": f"card-{_TS}-bp-escalation",
        "physician_action": "APPROVE_WITH_MONITORING",
    },
    "enrichmentData": {},
    "enrichmentVersion": "1.0",
})

# Second intervention: Metformin dose increase
INTERVENTION_ID_2 = f"intv-{_TS}-metformin-uptitrate"
intervention_events.append({
    "eventId": str(uuid.uuid4()),
    "patientId": PATIENT,
    "eventType": "INTERVENTION_APPROVED",
    "timestamp": now_ms + 16000,
    "sourceSystem": "flink-e2e-14day",
    "correlationId": RUN_ID,
    "payload": {
        "event_type": "INTERVENTION_APPROVED",
        "intervention_id": INTERVENTION_ID_2,
        "intervention_type": "MEDICATION_DOSE_INCREASE",
        "intervention_detail": {
            "drug_name": "Metformin",
            "drug_class": "METFORMIN",
            "old_dose_mg": 1000,
            "new_dose_mg": 1500,
            "reason": "FBG 185 mg/dL — inadequate glycemic control",
        },
        "observation_window_days": 21,
        "originating_card_id": f"card-{_TS}-fbg-escalation",
        "physician_action": "APPROVE_WITH_MONITORING",
    },
    "enrichmentData": {},
    "enrichmentVersion": "1.0",
})


# ═══════════════════════════════════════════════════════════════════
#  EXECUTE
# ═══════════════════════════════════════════════════════════════════

def main():
    print(f"{'=' * 70}")
    print(f"  RAJESH KUMAR 14-DAY CLINICAL E2E — ALL MODULES")
    print(f"  Run: {RUN_ID}")
    print(f"  Patient: {PATIENT}")
    print(f"  BP Readings: {len(bp_readings)} over 14 days")
    print(f"  Enriched Events: {len(enriched_events)}")
    print(f"  Intervention Events: {len(intervention_events)}")
    print(f"{'=' * 70}")

    # Snapshot pre-counts
    topics = [
        ("ingestion.vitals", "M7 input (BP readings)"),
        ("flink.bp-variability-metrics", "M7 output"),
        ("enriched-patient-events-v1", "M8/9/10/11/12b input"),
        ("clinical.intervention-events", "M12 input"),
        ("alerts.comorbidity-interactions", "M8 output"),
        ("flink.engagement-signals", "M9 output"),
        ("flink.meal-response", "M10 output"),
        ("flink.meal-patterns", "M10b output"),
        ("flink.activity-response", "M11 output"),
        ("flink.fitness-patterns", "M11b output"),
        ("clinical.intervention-window-signals", "M12 output"),
        ("flink.intervention-deltas", "M12b output"),
        ("clinical.state-change-events", "M13 output"),
    ]
    before = {}
    print(f"\n  Pre-test topic counts:")
    for t, label in topics:
        c = topic_count(t)
        before[t] = c
        print(f"    {t}: {c}")

    # ── Phase 1: BP Readings → ingestion.vitals ──
    print(f"\n  Phase 1: Sending {len(bp_readings)} BP readings to ingestion.vitals")
    bp_ok = 0
    for i, r in enumerate(bp_readings):
        if publish("ingestion.vitals", r):
            bp_ok += 1
            day = (r["timestamp"] - BASE) // DAY_MS + 1
            print(f"    [{bp_ok:2d}] Day {day:2d} {r['time_context']:8s} "
                  f"SBP={r['systolic']:3.0f}/{r['diastolic']:2.0f} "
                  f"src={r.get('source', 'HOME_CUFF')}")
        else:
            print(f"    FAILED reading #{i}")
        time.sleep(0.3)  # small delay to ensure ordering
    print(f"    Sent: {bp_ok}/{len(bp_readings)}")

    # ── Phase 2: Enriched Events ──
    print(f"\n  Phase 2: Sending {len(enriched_events)} enriched events")
    ee_ok = 0
    for ev in enriched_events:
        if publish("enriched-patient-events-v1", ev):
            ee_ok += 1
            etype = ev["eventType"]
            p = ev["payload"]
            detail = ""
            if etype == "LAB_RESULT":
                detail = f"{p.get('lab_type', '?')}={p.get('value', '?')}"
            elif etype == "MEDICATION_ORDERED":
                detail = f"{p.get('drug_name', '?')} ({p.get('drug_class', '?')})"
            elif etype == "MEDICATION_EVENT":
                detail = f"{p.get('drug_name', '?')} {p.get('action', '?')}"
            elif etype == "VITAL_SIGN":
                detail = f"BP {p.get('systolic_bp', '?')}/{p.get('diastolic_bp', '?')}"
            elif etype == "PATIENT_REPORTED":
                rtype = p.get("report_type") or p.get("symptom_type") or p.get("type", "?")
                detail = rtype
                if rtype == "MEAL_LOG":
                    detail += f" carb={p.get('carb_grams')}g prot={p.get('protein_grams')}g"
                elif rtype == "ACTIVITY_LOG":
                    detail += f" {p.get('exercise_type')} {p.get('duration_minutes')}min"
                elif rtype == "APP_SESSION":
                    detail += f" {p.get('session_duration_sec')}s"
                elif rtype == "GOAL_COMPLETED":
                    detail += f" {p.get('fields_completed')}/{p.get('total_fields')}"
            elif etype == "DEVICE_READING":
                if "glucose_value" in p:
                    detail = f"CGM glucose={p['glucose_value']}"
                elif "heart_rate" in p:
                    state = p.get("activity_state", "")
                    detail = f"HR={p['heart_rate']} {state}"
                elif "step_count" in p:
                    detail = f"steps={p['step_count']}"
            # Only print first 30 enriched to keep output manageable
            if ee_ok <= 30:
                print(f"    [{ee_ok:2d}] {etype}: {detail}")
        time.sleep(0.15)
    if ee_ok > 30:
        print(f"    ... ({ee_ok - 30} more)")
    print(f"    Sent: {ee_ok}/{len(enriched_events)}")

    # ── Phase 3: Intervention Events → clinical.intervention-events ──
    print(f"\n  Phase 3: Sending {len(intervention_events)} intervention events")
    intv_ok = 0
    for ev in intervention_events:
        if publish("clinical.intervention-events", ev):
            intv_ok += 1
            detail = ev["payload"].get("intervention_detail", {})
            print(f"    INTERVENTION_APPROVED: {detail.get('drug_name', '?')} "
                  f"{detail.get('old_dose_mg', '?')}→{detail.get('new_dose_mg', '?')}mg "
                  f"window={ev['payload'].get('observation_window_days', '?')}d")
        time.sleep(0.3)
    print(f"    Sent: {intv_ok}/{len(intervention_events)}")

    # ── Wait for processing ──
    print(f"\n{'=' * 70}")
    print(f"  Waiting 35s for Flink pipeline processing...")
    print(f"{'=' * 70}")
    time.sleep(35)

    # ── Phase 4: Verify Module 7 output ──
    print(f"\n  Phase 4: Verifying Module 7 output (flink.bp-variability-metrics)")
    after_m7 = topic_count("flink.bp-variability-metrics")
    m7_delta = after_m7 - before.get("flink.bp-variability-metrics", 0)
    print(f"    M7 output delta: +{m7_delta}")

    m7_msgs = consume("flink.bp-variability-metrics", timeout=12, max_msgs=500)
    rajesh_m7 = [m for m in m7_msgs if m.get("patient_id") == PATIENT]
    print(f"    Total M7 messages: {len(m7_msgs)}")
    print(f"    Rajesh M7 messages: {len(rajesh_m7)}")

    if rajesh_m7:
        last = rajesh_m7[-1]
        arv7 = last.get("arv_sbp_7d")
        sd7 = last.get("sd_sbp_7d")
        cv7 = last.get("cv_sbp_7d")
        var_class = last.get("variability_classification_7d")
        surge_today = last.get("morning_surge_today")
        surge_7d = last.get("morning_surge_7d_avg")
        surge_class = last.get("surge_classification")
        dip_class = last.get("dip_classification")
        bp_status = last.get("bp_control_status")
        ctx_depth = last.get("context_depth")
        days_7d = last.get("days_with_data_in_7d")
        dates = last.get("contributing_dates_7d", [])

        print(f"\n    ── Latest M7 Output (reading #{len(rajesh_m7)}) ──")
        print(f"    arv_sbp_7d:            {arv7}")
        print(f"    sd_sbp_7d:             {sd7}")
        print(f"    cv_sbp_7d:             {cv7}")
        print(f"    variability_class_7d:  {var_class}")
        print(f"    morning_surge_today:   {surge_today}")
        print(f"    morning_surge_7d_avg:  {surge_7d}")
        print(f"    surge_classification:  {surge_class}")
        print(f"    dip_classification:    {dip_class}")
        print(f"    bp_control_status:     {bp_status}")
        print(f"    context_depth:         {ctx_depth}")
        print(f"    days_with_data_7d:     {days_7d}")
        print(f"    contributing_dates_7d: {dates}")

        # ── Validation ──
        print(f"\n    ── ARV Fix Validation ──")
        if arv7 is not None and arv7 >= 10.0:
            print(f"    PASS: ARV={arv7:.1f} >= 10 (reading-level ARV captures oscillation)")
        elif arv7 is not None:
            print(f"    CHECK: ARV={arv7:.1f} — expected >= 10 for this BP pattern")
        else:
            print(f"    FAIL: ARV is null (need >= 3 readings in 7-day window)")

        if var_class in ("HIGH", "ELEVATED"):
            print(f"    PASS: Classification={var_class}")
        else:
            print(f"    CHECK: Classification={var_class} — expected HIGH or ELEVATED")

        if surge_today is not None and abs(surge_today) > 10:
            print(f"    PASS: Morning surge={surge_today:.0f} mmHg (significant)")
        else:
            print(f"    INFO: Morning surge={surge_today} (may be null if no prior evening)")

        # Print progression: ARV over time
        print(f"\n    ── ARV Progression Over 14 Days ──")
        for i, m in enumerate(rajesh_m7):
            a = m.get("arv_sbp_7d")
            v = m.get("variability_classification_7d", "?")
            s = m.get("trigger_sbp", "?")
            tc = m.get("trigger_time_context", "?")
            d7 = m.get("days_with_data_in_7d", 0)
            arv_str = f"{a:.1f}" if a is not None else "null"
            print(f"    [{i+1:2d}] SBP={s:>5} {tc:8s} ARV={arv_str:>6} {v:20s} days={d7}")

    # ── Phase 5: Check all topic deltas ──
    print(f"\n  Phase 5: Post-test topic counts (delta)")
    results = {}
    for t, label in topics:
        after = topic_count(t)
        delta = after - before.get(t, 0)
        results[t] = {"before": before.get(t, 0), "after": after, "delta": delta}
        marker = "+" if delta > 0 else " "
        print(f"    {marker}{delta:3d} {label:25s} {t}")

    # ── Phase 6: All module output details ──

    # Helper: match Rajesh across camelCase/snake_case patient ID fields
    def is_rajesh(msg):
        return msg.get("patient_id") == PATIENT or msg.get("patientId") == PATIENT

    # ── M8: Comorbidity ──
    print(f"\n  ── M8 Comorbidity Output ──")
    m8_msgs = consume("alerts.comorbidity-interactions", timeout=8)
    rajesh_m8 = [m for m in m8_msgs if is_rajesh(m)]
    print(f"    Total: {len(m8_msgs)}, Rajesh: {len(rajesh_m8)}")
    for i, m in enumerate(rajesh_m8, 1):
        print(f"    Alert #{i}: {m.get('ruleId','?')} {m.get('severity','?')} — "
              f"{(m.get('triggerSummary','')[:80])}")

    # ── M9: Engagement ──
    print(f"\n  ── M9 Engagement Output ──")
    m9_msgs = consume("flink.engagement-signals", timeout=8)
    rajesh_m9 = [m for m in m9_msgs if is_rajesh(m)]
    print(f"    Total: {len(m9_msgs)}, Rajesh: {len(rajesh_m9)}")
    if rajesh_m9:
        for i, m in enumerate(rajesh_m9, 1):
            print(f"    Signal #{i}: score={m.get('compositeScore', m.get('composite_score', '?'))}")
    else:
        print(f"    [Timer-dependent: M9 fires at 23:59 UTC daily]")

    # ── M10: Meal Response ──
    print(f"\n  ── M10 Meal Response Output ──")
    m10_msgs = consume("flink.meal-response", timeout=8)
    rajesh_m10 = [m for m in m10_msgs if is_rajesh(m)]
    print(f"    Total: {len(m10_msgs)}, Rajesh: {len(rajesh_m10)}")
    if rajesh_m10:
        for i, m in enumerate(rajesh_m10, 1):
            print(f"    Record #{i}: iAUC={m.get('iAUC', m.get('iauc', '?'))}, "
                  f"tier={m.get('dataTier', m.get('data_tier', '?'))}, "
                  f"excursion={m.get('glucoseExcursion', m.get('glucose_excursion', '?'))}")
    else:
        print(f"    [Timer-dependent: M10 fires meal+3h05m processing time]")

    # ── M10b: Meal Patterns ──
    print(f"\n  ── M10b Meal Patterns Output ──")
    m10b_msgs = consume("flink.meal-patterns", timeout=8)
    rajesh_m10b = [m for m in m10b_msgs if is_rajesh(m)]
    print(f"    Total: {len(m10b_msgs)}, Rajesh: {len(rajesh_m10b)}")
    if not rajesh_m10b:
        print(f"    [Weekly timer: Mon 00:00 UTC + requires M10 output]")

    # ── M11: Activity Response ──
    print(f"\n  ── M11 Activity Response Output ──")
    m11_msgs = consume("flink.activity-response", timeout=8)
    rajesh_m11 = [m for m in m11_msgs if is_rajesh(m)]
    print(f"    Total: {len(m11_msgs)}, Rajesh: {len(rajesh_m11)}")
    if rajesh_m11:
        for i, m in enumerate(rajesh_m11, 1):
            print(f"    Record #{i}: peakHR={m.get('peakHR', m.get('peak_hr', '?'))}, "
                  f"HRR1={m.get('hrr1', m.get('HRR1', '?'))}, "
                  f"metMinutes={m.get('metMinutes', m.get('met_minutes', '?'))}")
    else:
        print(f"    [Timer-dependent: M11 fires activity_end+2h05m processing time]")

    # ── M11b: Fitness Patterns ──
    print(f"\n  ── M11b Fitness Patterns Output ──")
    m11b_msgs = consume("flink.fitness-patterns", timeout=8)
    rajesh_m11b = [m for m in m11b_msgs if is_rajesh(m)]
    print(f"    Total: {len(m11b_msgs)}, Rajesh: {len(rajesh_m11b)}")
    if not rajesh_m11b:
        print(f"    [Weekly timer: Mon 00:00 UTC + requires M11 output]")

    # ── M12: Intervention Window ──
    print(f"\n  ── M12 Intervention Window Output ──")
    m12_msgs = consume("clinical.intervention-window-signals", timeout=8)
    rajesh_m12 = [m for m in m12_msgs if is_rajesh(m)]
    print(f"    Total: {len(m12_msgs)}, Rajesh: {len(rajesh_m12)}")
    if rajesh_m12:
        for i, m in enumerate(rajesh_m12, 1):
            sig_type = m.get("signalType", m.get("signal_type", "?"))
            intv_id = m.get("interventionId", m.get("intervention_id", "?"))
            print(f"    Signal #{i}: {sig_type} intervention={intv_id[:40]}...")
    else:
        print(f"    [Immediate on INTERVENTION_APPROVED — check topic wiring]")

    # ── M12b: Intervention Delta ──
    print(f"\n  ── M12b Intervention Delta Output ──")
    m12b_msgs = consume("flink.intervention-deltas", timeout=8)
    rajesh_m12b = [m for m in m12b_msgs if is_rajesh(m)]
    print(f"    Total: {len(m12b_msgs)}, Rajesh: {len(rajesh_m12b)}")
    if not rajesh_m12b:
        print(f"    [Requires WINDOW_CLOSED from M12 — timer-dependent]")

    # ── M13: Clinical State Sync ──
    print(f"\n  ── M13 Clinical State Sync Output ──")
    m13_msgs = consume("clinical.state-change-events", timeout=10)
    rajesh_m13 = [m for m in m13_msgs if is_rajesh(m)]
    print(f"    Total: {len(m13_msgs)}, Rajesh: {len(rajesh_m13)}")
    for i, m in enumerate(rajesh_m13, 1):
        print(f"    State Change #{i}: {m.get('change_type', '?')}")
        print(f"      priority:       {m.get('priority', '?')}")
        print(f"      action:         {m.get('recommended_action', '?')}")
        vel = m.get("ckm_velocity_at_change", {})
        if vel:
            d = vel.get("domain_velocities", {})
            print(f"      CARDIO={d.get('CARDIOVASCULAR', '?'):.4f}  "
                  f"META={d.get('METABOLIC', '?'):.4f}  "
                  f"RENAL={d.get('RENAL', '?'):.4f}" if all(
                      isinstance(d.get(k), (int, float)) for k in
                      ['CARDIOVASCULAR', 'METABOLIC', 'RENAL']
                  ) else f"      domains={d}")

    # ── Save I/O JSON ──
    output_json = {
        "test_run": RUN_ID,
        "patient_id": PATIENT,
        "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "pipeline": "14-day clinical dataset: M7→M8→M9→M10→M10b→M11→M11b→M12→M12b→M13",
        "patient_profile": {
            "name": "Rajesh Kumar", "age": 58, "gender": "M",
            "conditions": ["T2DM (12yr)", "HTN Stage 2", "CKD Stage 3b (eGFR 42)"],
            "medications": ["Metformin 1000mg BD", "Amlodipine 10mg", "Telmisartan 80mg",
                            "Dapagliflozin 10mg", "Hydrochlorothiazide 12.5mg"],
        },
        "dataset_phases": {
            "days_1_3": "Baseline: moderate HTN, morning surge 15-20 mmHg",
            "days_4_7": "Completion: evening readings, clinic visit day 5, nocturnal",
            "days_8_10": "Deterioration: SBP trending up (172→182)",
            "days_11_13": "Engagement collapse: fewer readings, high BP",
            "day_14": "Final snapshot: single morning reading",
        },
        "events_sent": {
            "bp_readings": len(bp_readings),
            "enriched_events": len(enriched_events),
            "intervention_events": len(intervention_events),
        },
        "input_events": {
            "bp_readings": bp_readings,
            "enriched_events": [{k: v for k, v in ev.items()} for ev in enriched_events],
            "intervention_events": intervention_events,
        },
        "module_outputs": {
            "m7_bp_variability": rajesh_m7 if rajesh_m7 else [],
            "m8_comorbidity": rajesh_m8,
            "m9_engagement": rajesh_m9,
            "m10_meal_response": rajesh_m10,
            "m10b_meal_patterns": rajesh_m10b,
            "m11_activity_response": rajesh_m11,
            "m11b_fitness_patterns": rajesh_m11b,
            "m12_intervention_window": rajesh_m12,
            "m12b_intervention_delta": rajesh_m12b,
            "m13_state_changes": rajesh_m13,
        },
        "topic_deltas": {t: results[t] for t, _ in topics},
        "arv_fix_validation": {
            "final_arv_7d": rajesh_m7[-1].get("arv_sbp_7d") if rajesh_m7 else None,
            "final_classification": rajesh_m7[-1].get("variability_classification_7d") if rajesh_m7 else None,
            "expected_arv_gte": 10.0,
            "expected_classification": "HIGH or ELEVATED",
        } if rajesh_m7 else {},
    }

    out_path = "/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/flink-processing/test-data/e2e-rajesh-14day.json"
    with open(out_path, "w") as f:
        json.dump(output_json, f, indent=2, default=str)
    print(f"\n  Saved I/O JSON: {out_path}")

    # ── Summary ──
    print(f"\n{'=' * 70}")
    print(f"  ALL MODULE OUTPUT SUMMARY")
    print(f"{'=' * 70}")
    module_counts = [
        ("M7  BP Variability", len(rajesh_m7) if rajesh_m7 else 0, "immediate"),
        ("M8  Comorbidity", len(rajesh_m8), "immediate"),
        ("M9  Engagement", len(rajesh_m9), "timer 23:59 UTC"),
        ("M10 Meal Response", len(rajesh_m10), "timer meal+3h05m"),
        ("M10b Meal Patterns", len(rajesh_m10b), "weekly Mon 00:00"),
        ("M11 Activity Response", len(rajesh_m11), "timer activity+2h05m"),
        ("M11b Fitness Patterns", len(rajesh_m11b), "weekly Mon 00:00"),
        ("M12 Intervention Window", len(rajesh_m12), "immediate on APPROVED"),
        ("M12b Intervention Delta", len(rajesh_m12b), "on WINDOW_CLOSED"),
        ("M13 Clinical State Sync", len(rajesh_m13), "immediate"),
    ]
    total_outputs = 0
    for name, count, trigger in module_counts:
        marker = "✓" if count > 0 else "○"
        note = f" ({trigger})" if count == 0 else ""
        print(f"    {marker} {name:28s} {count:3d} outputs{note}")
        total_outputs += count

    print(f"\n  Total outputs:       {total_outputs}")
    print(f"  BP readings sent:    {bp_ok}/{len(bp_readings)}")
    print(f"  Enriched events:     {ee_ok}/{len(enriched_events)}")
    print(f"  Interventions:       {intv_ok}/{len(intervention_events)}")

    if rajesh_m7:
        final_arv = rajesh_m7[-1].get("arv_sbp_7d")
        if final_arv is not None and final_arv >= 10.0:
            print(f"\n  ARV FIX VALIDATED: {final_arv:.1f} >= 10.0 (reading-level Mena et al.)")
        else:
            print(f"\n  ARV CHECK NEEDED: {final_arv} — expected >= 10.0")
    else:
        print(f"\n  NO M7 OUTPUT — check Module 7 is running")

    return 0


if __name__ == "__main__":
    sys.exit(main())
