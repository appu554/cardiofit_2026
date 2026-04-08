#!/usr/bin/env python3
"""
Flink E2E Multi-Day Simulation: Rajesh Kumar — ALL Modules 7→13
================================================================
Produces 10 days of clinically realistic data in CORRECT formats.

KEY FINDINGS FROM CODE REVIEW:
  - Modules 8/9/10/11 all read CanonicalEvent (not EnrichedEvent!)
  - Module 8: needs drug_name + drug_class fields in payload
  - Module 9: emits ONLY on daily timer (23:59 UTC processing time)
  - Module 10: emits per-meal MealResponseRecord → 10b aggregates weekly
  - Module 10b: emits ONLY on weekly timer (Monday 00:00 UTC)
  - Module 11/11b: similar weekly timer pattern
  - Module 7: BPReading from ingestion.vitals — emits per reading (immediate)

So modules 9, 10b, 11b use PROCESSING TIME timers — they fire at wall-clock
midnight/Monday regardless of event timestamps. This means:
  - Module 9 output:  wait until 23:59 UTC today
  - Module 10b output: wait until next Monday 00:00 UTC
  - Module 11b output: wait until next Monday 00:00 UTC

Module 8 emits immediately per event IF CID rules match.
Module 7 emits immediately per BP reading.
Module 13 uses a 5s coalescing buffer timer.

This script:
  1. Sends 10 days of BP readings for Module 7 (immediate output)
  2. Sends proper CanonicalEvents with drug_name/drug_class for Module 8
  3. Sends SMBG/activity CanonicalEvents for Module 9 (output at next midnight)
  4. Sends meal CanonicalEvents for Module 10 (output per-event from M10)
  5. Sends activity CanonicalEvents for Module 11
  6. Injects 14 days upstream-format data to Module 13's input topics
  7. Captures ALL output from every intermediate topic
"""

import json
import subprocess
import sys
import time
import uuid

KAFKA = "cardiofit-kafka-lite"
BOOTSTRAP = "localhost:29092"
PATIENT = "e2e-rajesh-multiday-002"
RUN_ID = f"e2e-multi-{int(time.time())}"
NOW = int(time.time() * 1000)
DAY_MS = 86400000
HOUR_MS = 3600000


def publish(topic, msg):
    j = json.dumps(msg, separators=(",", ":"))
    r = subprocess.run(
        ["docker", "exec", "-i", KAFKA, "kafka-console-producer",
         "--bootstrap-server", BOOTSTRAP, "--topic", topic],
        input=j, capture_output=True, text=True, timeout=15)
    return r.returncode == 0


def consume_all(topic, timeout=12, max_msgs=500):
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


def consume_patient(topic, patient_id, timeout=12):
    all_msgs = consume_all(topic, timeout=timeout)
    return [m for m in all_msgs
            if (m.get("patient_id") or m.get("patientId") or "") == patient_id]


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


def canonical_event(patient_id, event_type, ts, payload):
    """Build CanonicalEvent matching Java @JsonProperty annotations."""
    return {
        "eventId": str(uuid.uuid4()),
        "patientId": patient_id,
        "eventType": event_type,
        "timestamp": ts,
        "processingTime": NOW,
        "sourceSystem": "flink-e2e-multiday",
        "correlationId": RUN_ID,
        "payload": payload,
    }


def section(title):
    print(f"\n{'='*70}")
    print(f"  {title}")
    print(f"{'='*70}")


def main():
    section(f"MULTI-DAY E2E: Rajesh Kumar — ALL Modules 7→13")
    print(f"  Run ID:  {RUN_ID}")
    print(f"  Patient: {PATIENT}")

    results = {}
    total_sent = 0

    # ═══════════════════════════════════════════════════════════════
    #  MODULE 7: 10 days × 2 BP readings/day = 20 readings
    # ═══════════════════════════════════════════════════════════════
    section("MODULE 7: BP Variability Engine — 10 days of BP readings")
    before7 = topic_count("flink.bp-variability-metrics")

    bp_inputs = []
    # Rajesh: HTN Stage 2, morning surge pattern, non-dipper
    for day in range(10):
        ts_morning = NOW - (10 - day) * DAY_MS + 6 * HOUR_MS
        ts_evening = NOW - (10 - day) * DAY_MS + 20 * HOUR_MS

        # Morning readings: higher (surge)
        sbp_m = 168 + (day % 3) * 4  # 168-176 range
        dbp_m = 100 + (day % 3) * 2

        # Evening readings: lower but still elevated (non-dipper: <10% nocturnal dip)
        sbp_e = 162 + (day % 3) * 3  # minimal drop = non-dipper
        dbp_e = 96 + (day % 3) * 2

        for sbp, dbp, ctx, ts in [(sbp_m, dbp_m, "MORNING", ts_morning),
                                   (sbp_e, dbp_e, "EVENING", ts_evening)]:
            bp = {
                "patient_id": PATIENT,
                "systolic": sbp, "diastolic": dbp,
                "heart_rate": 78 + day % 5,
                "timestamp": ts,
                "time_context": ctx,
                "source": "HOME_CUFF", "position": "SEATED",
                "device_type": "oscillometric_cuff",
                "correlation_id": RUN_ID,
            }
            bp_inputs.append(bp)
            ok = publish("ingestion.vitals", bp)
            total_sent += 1

    print(f"  Sent: {len(bp_inputs)} BP readings (10 days × 2/day)")
    print(f"  Waiting 20s for Module 7...")
    time.sleep(20)

    m7_output = consume_patient("flink.bp-variability-metrics", PATIENT)
    delta7 = topic_count("flink.bp-variability-metrics") - before7
    print(f"  Output: +{delta7} total, {len(m7_output)} for Rajesh")

    if m7_output:
        last = m7_output[-1]
        print(f"\n  Last M7 output (reading #{last.get('total_readings_in_state','?')}):")
        for k in ["arv_sbp_7d", "variability_classification_7d", "sbp_7d_avg", "dbp_7d_avg",
                   "morning_surge_7d_avg", "surge_classification",
                   "dip_ratio", "dip_classification", "bp_control_status",
                   "days_with_data_in_7d", "contributing_dates_7d", "context_depth"]:
            print(f"    {k}: {last.get(k, '—')}")

    results["module7"] = {"inputs": len(bp_inputs), "outputs": len(m7_output),
                          "delta": delta7, "last_output": m7_output[-1] if m7_output else None}

    # ═══════════════════════════════════════════════════════════════
    #  MODULE 8: Metformin + low eGFR → CID rule match
    # ═══════════════════════════════════════════════════════════════
    section("MODULE 8: Comorbidity Engine — drug+lab CID triggers")
    before8 = topic_count("alerts.comorbidity-interactions")

    m8_inputs = []

    # Metformin with drug_class (what Module 8 actually reads)
    ev = canonical_event(PATIENT, "MEDICATION_ORDERED", NOW - 5 * DAY_MS, {
        "drug_name": "metformin", "drug_class": "BIGUANIDE",
        "dose_mg": 1000.0, "route": "oral", "frequency": "BD",
    })
    m8_inputs.append(ev)
    publish("enriched-patient-events-v1", ev)
    print(f"  ✓ Medication: metformin (BIGUANIDE) 1000mg")

    # Amlodipine
    ev = canonical_event(PATIENT, "MEDICATION_ORDERED", NOW - 5 * DAY_MS + 1000, {
        "drug_name": "amlodipine", "drug_class": "CCB",
        "dose_mg": 10.0, "route": "oral",
    })
    m8_inputs.append(ev)
    publish("enriched-patient-events-v1", ev)
    print(f"  ✓ Medication: amlodipine (CCB) 10mg")

    # Telmisartan
    ev = canonical_event(PATIENT, "MEDICATION_ORDERED", NOW - 5 * DAY_MS + 2000, {
        "drug_name": "telmisartan", "drug_class": "ARB",
        "dose_mg": 80.0, "route": "oral",
    })
    m8_inputs.append(ev)
    publish("enriched-patient-events-v1", ev)
    print(f"  ✓ Medication: telmisartan (ARB) 80mg")

    # eGFR declining — should trigger CID-01 (AKI/renal decline)
    ev = canonical_event(PATIENT, "LAB_RESULT", NOW - 4 * DAY_MS, {
        "lab_type": "egfr", "value": 55.0, "unit": "mL/min",
    })
    m8_inputs.append(ev)
    publish("enriched-patient-events-v1", ev)
    print(f"  ✓ Lab: eGFR=55 (baseline)")

    ev = canonical_event(PATIENT, "LAB_RESULT", NOW - 1 * DAY_MS, {
        "lab_type": "egfr", "value": 42.0, "unit": "mL/min",
    })
    m8_inputs.append(ev)
    publish("enriched-patient-events-v1", ev)
    print(f"  ✓ Lab: eGFR=42 (decline >20% from baseline → CID-01)")

    # Potassium high — CID-02 (hyperkalemia with ARB)
    ev = canonical_event(PATIENT, "LAB_RESULT", NOW - 1 * DAY_MS + 1000, {
        "lab_type": "potassium", "value": 5.8, "unit": "mEq/L",
    })
    m8_inputs.append(ev)
    publish("enriched-patient-events-v1", ev)
    print(f"  ✓ Lab: K+=5.8 (hyperkalemia with ARB → CID-02)")

    # FBG high
    ev = canonical_event(PATIENT, "LAB_RESULT", NOW, {
        "lab_type": "fbg", "value": 185.0, "unit": "mg/dL",
    })
    m8_inputs.append(ev)
    publish("enriched-patient-events-v1", ev)
    print(f"  ✓ Lab: FBG=185")

    total_sent += len(m8_inputs)

    print(f"  Waiting 15s for Module 8...")
    time.sleep(15)

    m8_output = consume_patient("alerts.comorbidity-interactions", PATIENT)
    delta8 = topic_count("alerts.comorbidity-interactions") - before8
    print(f"  Output: +{delta8} total, {len(m8_output)} for Rajesh")

    if m8_output:
        for i, m in enumerate(m8_output, 1):
            rid = m.get("ruleId") or m.get("rule_id") or "?"
            sev = m.get("severity") or "?"
            desc = m.get("description") or m.get("triggerSummary") or "?"
            print(f"    [{i}] {rid} ({sev}): {str(desc)[:80]}")

    results["module8"] = {"inputs": len(m8_inputs), "outputs": len(m8_output),
                          "delta": delta8, "output": m8_output}

    # ═══════════════════════════════════════════════════════════════
    #  MODULE 9: Engagement Monitor (output at next 23:59 UTC)
    # ═══════════════════════════════════════════════════════════════
    section("MODULE 9: Engagement Monitor — SMBG + activity events")
    before9 = topic_count("flink.engagement-signals")

    m9_inputs = []
    # 5 SMBG readings + 2 activity events
    smbg_data = [
        ("fasting", 185), ("post_meal", 210), ("bedtime", 178),
        ("fasting", 192), ("post_meal", 225),
    ]
    for i, (timing, glucose) in enumerate(smbg_data):
        ev = canonical_event(PATIENT, "PATIENT_REPORTED", NOW - 2 * DAY_MS + i * HOUR_MS, {
            "type": "smbg", "glucose": glucose, "timing": timing,
        })
        m9_inputs.append(ev)
        publish("enriched-patient-events-v1", ev)

    for i in range(2):
        ev = canonical_event(PATIENT, "DEVICE_READING", NOW - DAY_MS + i * 12 * HOUR_MS, {
            "type": "activity", "steps": 3200 + i * 1000,
            "active_minutes": 22 + i * 10,
        })
        m9_inputs.append(ev)
        publish("enriched-patient-events-v1", ev)

    total_sent += len(m9_inputs)
    print(f"  Sent: {len(m9_inputs)} events (5 SMBG + 2 activity)")
    print(f"  NOTE: Module 9 emits at 23:59 UTC (processing time daily timer)")
    print(f"  Checking for any output...")
    time.sleep(10)

    m9_output = consume_patient("flink.engagement-signals", PATIENT)
    delta9 = topic_count("flink.engagement-signals") - before9
    print(f"  Output: +{delta9} total, {len(m9_output)} for Rajesh")
    if not m9_output:
        print(f"  Expected: output will appear at next 23:59 UTC")

    results["module9"] = {"inputs": len(m9_inputs), "outputs": len(m9_output),
                          "delta": delta9, "output": m9_output,
                          "note": "Emits on daily timer at 23:59 UTC"}

    # ═══════════════════════════════════════════════════════════════
    #  MODULE 10/10b: Meal Response + Pattern Aggregator
    # ═══════════════════════════════════════════════════════════════
    section("MODULE 10/10b: Meal Response Correlator + Pattern Aggregator")
    before10 = topic_count("flink.meal-response")
    before10b = topic_count("flink.meal-patterns")

    m10_inputs = []
    meals = [
        ("breakfast", 45, 145, 195, 800), ("lunch", 65, 155, 225, 2200),
        ("dinner", 55, 140, 200, 1500), ("breakfast", 40, 138, 185, 700),
        ("lunch", 70, 160, 240, 1800),
    ]
    for i, (meal, carbs, pre, post, sodium) in enumerate(meals):
        ev = canonical_event(PATIENT, "PATIENT_REPORTED", NOW - 3 * DAY_MS + i * 6 * HOUR_MS, {
            "type": "meal_log", "meal_type": meal,
            "carb_estimate_g": carbs, "pre_meal_glucose": pre,
            "post_meal_glucose": post, "sodium_mg": sodium,
        })
        m10_inputs.append(ev)
        publish("enriched-patient-events-v1", ev)

    total_sent += len(m10_inputs)
    print(f"  Sent: {len(m10_inputs)} meal events")
    print(f"  NOTE: Module 10b emits on weekly timer (Monday 00:00 UTC)")
    time.sleep(10)

    m10_output = consume_patient("flink.meal-response", PATIENT)
    m10b_output = consume_patient("flink.meal-patterns", PATIENT)
    delta10 = topic_count("flink.meal-response") - before10
    delta10b = topic_count("flink.meal-patterns") - before10b
    print(f"  M10 output:  +{delta10} total, {len(m10_output)} for Rajesh")
    print(f"  M10b output: +{delta10b} total, {len(m10b_output)} for Rajesh")

    results["module10_10b"] = {"inputs": len(m10_inputs),
                               "m10_outputs": len(m10_output), "m10b_outputs": len(m10b_output),
                               "delta10": delta10, "delta10b": delta10b,
                               "note": "M10b emits on weekly timer (Monday 00:00 UTC)"}

    # ═══════════════════════════════════════════════════════════════
    #  MODULE 11/11b: Activity Response + Fitness Pattern Aggregator
    # ═══════════════════════════════════════════════════════════════
    section("MODULE 11/11b: Activity Response + Fitness Pattern Aggregator")
    before11 = topic_count("flink.activity-response")
    before11b = topic_count("flink.fitness-patterns")

    m11_inputs = []
    for i in range(5):
        ev = canonical_event(PATIENT, "DEVICE_READING", NOW - 5 * DAY_MS + i * DAY_MS, {
            "type": "activity", "steps": 3000 + i * 500,
            "active_minutes": 20 + i * 5,
            "calories_burned": 150 + i * 30,
            "heart_rate_avg": 90 + i * 3,
            "heart_rate_max": 115 + i * 5,
        })
        m11_inputs.append(ev)
        publish("enriched-patient-events-v1", ev)

    total_sent += len(m11_inputs)
    print(f"  Sent: {len(m11_inputs)} activity events (5 days)")
    print(f"  NOTE: Module 11b emits on weekly timer")
    time.sleep(10)

    m11_output = consume_patient("flink.activity-response", PATIENT)
    m11b_output = consume_patient("flink.fitness-patterns", PATIENT)
    delta11 = topic_count("flink.activity-response") - before11
    delta11b = topic_count("flink.fitness-patterns") - before11b
    print(f"  M11 output:  +{delta11} total, {len(m11_output)} for Rajesh")
    print(f"  M11b output: +{delta11b} total, {len(m11b_output)} for Rajesh")

    results["module11_11b"] = {"inputs": len(m11_inputs),
                               "m11_outputs": len(m11_output), "m11b_outputs": len(m11b_output),
                               "delta11": delta11, "delta11b": delta11b}

    # ═══════════════════════════════════════════════════════════════
    #  MODULE 13: 14-day simulated upstream data (direct injection)
    # ═══════════════════════════════════════════════════════════════
    section("MODULE 13: Clinical State Synchroniser — 14-day simulation")
    before13 = topic_count("clinical.state-change-events")
    m13_patient = "e2e-rajesh-m13-full-002"
    m13_sent = 0

    for day in range(14):
        ts = NOW - (14 - day) * DAY_MS
        worsening = day >= 7

        # BP variability
        arv = 12.0 + (day * 1.0 if worsening else 0)
        bp = {
            "id": str(uuid.uuid4()), "patientId": m13_patient,
            "eventType": "VITAL_SIGN", "eventTime": ts,
            "sourceSystem": "flink-e2e-sim", "correlationId": RUN_ID,
            "payload": {
                "arv": round(arv, 1), "arv_sbp_7d": round(arv, 1),
                "variability_classification": "HIGH" if arv > 16 else "MODERATE",
                "mean_sbp": round(145 + (day * 2.5 if worsening else 0), 1),
                "mean_sbp_7d": round(145 + (day * 2.5 if worsening else 0), 1),
                "mean_dbp": 92.0, "mean_dbp_7d": 92.0,
                "morning_surge_magnitude": round(22 + (day * 2 if worsening else 0), 0),
                "dip_classification": "NON_DIPPER" if worsening else "DIPPER",
                "arv_contributing_dates": [
                    time.strftime("%Y-%m-%d", time.gmtime((ts - i * DAY_MS) / 1000))
                    for i in range(min(day + 1, 7))
                ],
            },
        }
        publish("flink.bp-variability-metrics", bp)
        m13_sent += 1

        # Engagement
        score = 0.78 - (day * 0.04 if worsening else 0)
        eng = {
            "id": str(uuid.uuid4()), "patientId": m13_patient,
            "eventType": "PATIENT_REPORTED", "eventTime": ts + 1000,
            "sourceSystem": "flink-e2e-sim", "correlationId": RUN_ID,
            "payload": {
                "compositeScore": round(max(score, 0.15), 2),
                "engagementLevel": "RED" if score < 0.4 else ("AMBER" if score < 0.6 else "GREEN"),
                "phenotype": "DECLINING_ENGAGER" if score < 0.5 else "ACTIVE",
                "dataTier": "TIER_2",
            },
        }
        publish("flink.engagement-signals", eng)
        m13_sent += 1

        # Meal patterns every 3 days
        if day % 3 == 0:
            iauc = 30 + (day * 2 if worsening else 0)
            meal = {
                "id": str(uuid.uuid4()), "patientId": m13_patient,
                "eventType": "PATIENT_REPORTED", "eventTime": ts + 2000,
                "sourceSystem": "flink-e2e-sim", "correlationId": RUN_ID,
                "payload": {
                    "meanIAUC": round(iauc, 1),
                    "medianExcursion": round(38 + day * 1.5, 1),
                    "saltSensitivityClass": "HIGH" if worsening else "MODERATE",
                    "saltBeta": round(0.35 + (0.04 * day if worsening else 0), 2),
                },
            }
            publish("flink.meal-patterns", meal)
            m13_sent += 1

        # Fitness patterns every 3 days
        if day % 3 == 1:
            vo2 = 30 - (day * 0.4 if worsening else 0)
            fit = {
                "id": str(uuid.uuid4()), "patientId": m13_patient,
                "eventType": "DEVICE_READING", "eventTime": ts + 3000,
                "sourceSystem": "flink-e2e-sim", "correlationId": RUN_ID,
                "payload": {
                    "estimatedVO2max": round(max(vo2, 18), 1),
                    "vo2maxTrend": round(-0.8 if worsening else 0.2, 1),
                    "totalMetMinutes": round(160 - (day * 6 if worsening else 0), 0),
                    "meanExerciseGlucoseDelta": -7.0,
                },
            }
            publish("flink.fitness-patterns", fit)
            m13_sent += 1

        # Comorbidity alert on day 10
        if day == 10:
            cid = {
                "id": str(uuid.uuid4()), "patientId": m13_patient,
                "eventType": "CLINICAL_DOCUMENT", "eventTime": ts + 4000,
                "sourceSystem": "flink-e2e-sim", "correlationId": RUN_ID,
                "payload": {
                    "ruleId": "CID-01", "severity": "HALT",
                    "interaction_type": "DRUG_DISEASE",
                    "description": "eGFR acute decline >25% — hold metformin",
                },
            }
            publish("alerts.comorbidity-interactions", cid)
            m13_sent += 1

    total_sent += m13_sent
    print(f"  Injected: {m13_sent} events across 14 days (week 1 stable, week 2 deteriorating)")
    print(f"  Waiting 45s for Module 13 coalescing + snapshot rotation...")
    time.sleep(45)

    m13_output = consume_patient("clinical.state-change-events", m13_patient)
    delta13 = topic_count("clinical.state-change-events") - before13
    print(f"  Output: +{delta13} total, {len(m13_output)} for Rajesh-sim")

    if m13_output:
        for i, m in enumerate(m13_output, 1):
            ct = m.get("change_type", "?")
            pri = m.get("priority", "?")
            action = m.get("recommended_action", "?")
            comp = m.get("data_completeness_at_change", 0)
            ckm = m.get("ckm_velocity_at_change", {})
            print(f"\n    ── M13 Output #{i} ──")
            print(f"    change_type:        {ct}")
            print(f"    priority:           {pri}")
            print(f"    recommended_action: {action}")
            print(f"    data_completeness:  {comp:.3f}")
            print(f"    ckm_composite:      {ckm.get('composite_score',0)} ({ckm.get('composite_classification','?')})")
            print(f"    domains_deteriorating: {ckm.get('domains_deteriorating',0)}")
            dv = ckm.get("domain_velocities", {})
            if dv:
                print(f"    domain_velocities:  {json.dumps(dv)}")

    results["module13"] = {"patient_id": m13_patient, "inputs": m13_sent,
                           "outputs": len(m13_output), "delta": delta13,
                           "output": m13_output}

    # ═══════════════════════════════════════════════════════════════
    #  FINAL SUMMARY + JSON SAVE
    # ═══════════════════════════════════════════════════════════════
    section("FINAL SUMMARY")

    summary = [
        ("Module 7",  "BP Variability",        results["module7"]["outputs"],
         "flink.bp-variability-metrics", "Immediate per reading"),
        ("Module 8",  "Comorbidity CID",       results["module8"]["outputs"],
         "alerts.comorbidity-interactions", "Immediate if CID rule matches"),
        ("Module 9",  "Engagement",            results["module9"]["outputs"],
         "flink.engagement-signals", "Daily timer 23:59 UTC"),
        ("Module 10", "Meal Response",         results.get("module10_10b",{}).get("m10_outputs",0),
         "flink.meal-response", "Per meal event"),
        ("Module 10b","Meal Patterns",         results.get("module10_10b",{}).get("m10b_outputs",0),
         "flink.meal-patterns", "Weekly timer (Monday)"),
        ("Module 11", "Activity Response",     results.get("module11_11b",{}).get("m11_outputs",0),
         "flink.activity-response", "Per activity event"),
        ("Module 11b","Fitness Patterns",      results.get("module11_11b",{}).get("m11b_outputs",0),
         "flink.fitness-patterns", "Weekly timer (Monday)"),
        ("Module 13", "Clinical State Sync",   results["module13"]["outputs"],
         "clinical.state-change-events", "5s coalescing buffer"),
    ]

    for name, desc, count, topic, emit_trigger in summary:
        status = "✓" if count > 0 else "—"
        print(f"  {status} {name:10s} {desc:22s}  {count:3d} output  [{emit_trigger}]")

    flowing = sum(1 for _, _, c, _, _ in summary if c > 0)
    print(f"\n  {flowing}/{len(summary)} modules produced output")
    print(f"  Total events sent: {total_sent}")

    # Save JSON
    out_path = "/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/flink-processing/test-data/e2e-rajesh-multiday-all-modules.json"
    with open(out_path, "w") as f:
        json.dump({
            "test_run": RUN_ID,
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "patient_ids": {"modules_7_11": PATIENT, "module_13": "e2e-rajesh-m13-full-002"},
            "total_events_sent": total_sent,
            "results": results,
        }, f, indent=2, default=str)
    print(f"\n  Full I/O JSON → {out_path}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
