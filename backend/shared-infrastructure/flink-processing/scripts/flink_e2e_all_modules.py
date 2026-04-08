#!/usr/bin/env python3
"""
Flink E2E ALL MODULES: Rajesh Kumar — Individual Module 7→13 Validation
=======================================================================
Tests EACH module independently and captures input/output per module.

Module 7:  ingestion.vitals              → flink.bp-variability-metrics
Module 8:  enriched-patient-events-v1    → alerts.comorbidity-interactions
Module 9:  enriched-patient-events-v1    → flink.engagement-signals
Module 10: enriched-patient-events-v1    → flink.meal-response
Module 10b: flink.meal-response          → flink.meal-patterns
Module 11: enriched-patient-events-v1    → flink.activity-response
Module 11b: flink.activity-response      → flink.fitness-patterns
Module 13: all above outputs             → clinical.state-change-events

For Module 13: injects 14 days of simulated upstream output to trigger
real CKM velocity computation (not just cold-start DATA_ABSENCE_WARNING).
"""

import json
import subprocess
import sys
import time
import uuid

KAFKA = "cardiofit-kafka-lite"
BOOTSTRAP = "localhost:29092"
PATIENT = "e2e-rajesh-allmod-002"
RUN_ID = f"e2e-allmod-{int(time.time())}"
NOW = int(time.time() * 1000)
DAY_MS = 86400000


def publish(topic, msg):
    j = json.dumps(msg, separators=(",", ":"))
    r = subprocess.run(
        ["docker", "exec", "-i", KAFKA, "kafka-console-producer",
         "--bootstrap-server", BOOTSTRAP, "--topic", topic],
        input=j, capture_output=True, text=True, timeout=15)
    return r.returncode == 0


def consume(topic, timeout=12, max_msgs=500, patient_filter=None):
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
            m = json.loads(line.strip())
            if patient_filter:
                pid = m.get("patient_id") or m.get("patientId") or ""
                if pid != patient_filter:
                    continue
            msgs.append(m)
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


def section(title):
    print(f"\n{'='*65}")
    print(f"  {title}")
    print(f"{'='*65}")


# ═══════════════════════════════════════════════════════════════════
#  MODULE 7: BP Variability Engine
# ═══════════════════════════════════════════════════════════════════
def test_module7():
    section("MODULE 7: BP Variability Engine")
    print(f"  Input:  ingestion.vitals (BPReading)")
    print(f"  Output: flink.bp-variability-metrics")

    before = topic_count("flink.bp-variability-metrics")

    readings = []
    # 6 readings over 3 days — morning surge pattern, non-dipper
    bp_data = [
        (172, 102, "MORNING", -2), (168, 100, "EVENING", -2),
        (175, 104, "MORNING", -1), (164, 98,  "EVENING", -1),
        (180, 106, "MORNING",  0), (160, 96,  "EVENING",  0),
    ]
    for sbp, dbp, ctx, day_offset in bp_data:
        bp = {
            "patient_id": PATIENT,
            "systolic": sbp, "diastolic": dbp,
            "heart_rate": 78,
            "timestamp": NOW + day_offset * DAY_MS + (6 * 3600000 if ctx == "MORNING" else 20 * 3600000),
            "time_context": ctx,
            "source": "HOME_CUFF", "position": "SEATED",
            "device_type": "oscillometric_cuff",
            "correlation_id": RUN_ID,
        }
        readings.append(bp)
        ok = publish("ingestion.vitals", bp)
        print(f"    {'✓' if ok else '✗'} BP {sbp}/{dbp} ({ctx}, day {day_offset})")

    print(f"  Waiting 15s for Module 7 processing...")
    time.sleep(15)

    after = topic_count("flink.bp-variability-metrics")
    delta = after - before
    output = consume("flink.bp-variability-metrics", timeout=10, patient_filter=PATIENT)

    print(f"  Output: +{delta} messages on flink.bp-variability-metrics")
    print(f"  Rajesh-filtered: {len(output)} messages")

    if output:
        for i, m in enumerate(output[:3], 1):
            arv = m.get("arv") or m.get("arv_sbp_7d") or "?"
            cls = m.get("variability_classification") or m.get("classification") or "?"
            surge = m.get("morning_surge_magnitude") or "?"
            dip = m.get("dip_classification") or "?"
            mean_sbp = m.get("mean_sbp") or m.get("mean_sbp_7d") or "?"
            print(f"    [{i}] ARV={arv}, class={cls}, surge={surge}, dip={dip}, mean_sbp={mean_sbp}")

    return {"input": readings, "output": output, "delta": delta}


# ═══════════════════════════════════════════════════════════════════
#  MODULE 8: Comorbidity Interaction Detector
# ═══════════════════════════════════════════════════════════════════
def test_module8():
    section("MODULE 8: Comorbidity Interaction Detector")
    print(f"  Input:  enriched-patient-events-v1 (EnrichedEvent)")
    print(f"  Output: alerts.comorbidity-interactions")

    before = topic_count("alerts.comorbidity-interactions")

    events = []
    # Medication + lab combo that should trigger CID rules
    med_event = {
        "eventId": str(uuid.uuid4()), "patientId": PATIENT,
        "eventType": "MEDICATION_ORDERED", "timestamp": NOW,
        "sourceSystem": "flink-e2e", "correlationId": RUN_ID,
        "payload": {"medicationName": "Metformin", "dose": 1000, "doseUnit": "mg",
                    "route": "oral", "status": "active"},
        "enrichmentData": {}, "enrichmentVersion": "1.0",
    }
    events.append(med_event)
    print(f"    {'✓' if publish('enriched-patient-events-v1', med_event) else '✗'} Medication: Metformin 1000mg")

    # eGFR low — should trigger Metformin-CKD interaction
    lab_event = {
        "eventId": str(uuid.uuid4()), "patientId": PATIENT,
        "eventType": "LAB_RESULT", "timestamp": NOW + 1000,
        "sourceSystem": "flink-e2e", "correlationId": RUN_ID,
        "payload": {"lab_type": "EGFR", "value": 42.0, "unit": "mL/min",
                    "testName": "eGFR", "results": {"egfr": 42.0}},
        "enrichmentData": {}, "enrichmentVersion": "1.0",
    }
    events.append(lab_event)
    print(f"    {'✓' if publish('enriched-patient-events-v1', lab_event) else '✗'} Lab: eGFR=42")

    # Another medication for drug-drug check
    med2 = {
        "eventId": str(uuid.uuid4()), "patientId": PATIENT,
        "eventType": "MEDICATION_ORDERED", "timestamp": NOW + 2000,
        "sourceSystem": "flink-e2e", "correlationId": RUN_ID,
        "payload": {"medicationName": "Amlodipine", "dose": 10, "doseUnit": "mg",
                    "route": "oral", "status": "active"},
        "enrichmentData": {}, "enrichmentVersion": "1.0",
    }
    events.append(med2)
    print(f"    {'✓' if publish('enriched-patient-events-v1', med2) else '✗'} Medication: Amlodipine 10mg")

    print(f"  Waiting 15s for Module 8 processing...")
    time.sleep(15)

    after = topic_count("alerts.comorbidity-interactions")
    delta = after - before
    output = consume("alerts.comorbidity-interactions", timeout=10, patient_filter=PATIENT)

    print(f"  Output: +{delta} messages on alerts.comorbidity-interactions")
    print(f"  Rajesh-filtered: {len(output)} messages")
    if output:
        for i, m in enumerate(output[:3], 1):
            rid = m.get("ruleId") or m.get("rule_id") or "?"
            sev = m.get("severity") or "?"
            desc = m.get("description") or m.get("interaction_type") or "?"
            print(f"    [{i}] rule={rid}, severity={sev}, desc={desc[:60]}")

    return {"input": events, "output": output, "delta": delta}


# ═══════════════════════════════════════════════════════════════════
#  MODULE 9: Engagement Monitor
# ═══════════════════════════════════════════════════════════════════
def test_module9():
    section("MODULE 9: Engagement Monitor")
    print(f"  Input:  enriched-patient-events-v1 (EnrichedEvent)")
    print(f"  Output: flink.engagement-signals")

    before = topic_count("flink.engagement-signals")

    events = []
    # Multiple patient-reported events to build engagement history
    for i, (etype, payload) in enumerate([
        ("PATIENT_REPORTED", {"type": "smbg", "glucose": 185, "timing": "fasting"}),
        ("PATIENT_REPORTED", {"type": "smbg", "glucose": 210, "timing": "post_meal"}),
        ("DEVICE_READING", {"type": "activity", "steps": 3200, "active_minutes": 22}),
        ("PATIENT_REPORTED", {"type": "meal_log", "carb_estimate_g": 65}),
        ("PATIENT_REPORTED", {"type": "smbg", "glucose": 178, "timing": "bedtime"}),
    ]):
        ev = {
            "eventId": str(uuid.uuid4()), "patientId": PATIENT,
            "eventType": etype, "timestamp": NOW + i * 3600000,
            "sourceSystem": "flink-e2e", "correlationId": RUN_ID,
            "payload": payload,
            "enrichmentData": {}, "enrichmentVersion": "1.0",
        }
        events.append(ev)
        ok = publish("enriched-patient-events-v1", ev)
        desc = payload.get("type") or etype
        print(f"    {'✓' if ok else '✗'} {etype}: {desc}")

    print(f"  Waiting 15s for Module 9 processing...")
    time.sleep(15)

    after = topic_count("flink.engagement-signals")
    delta = after - before
    output = consume("flink.engagement-signals", timeout=10, patient_filter=PATIENT)

    print(f"  Output: +{delta} messages on flink.engagement-signals")
    print(f"  Rajesh-filtered: {len(output)} messages")
    if output:
        for i, m in enumerate(output[:3], 1):
            score = m.get("compositeScore") or m.get("composite_score") or "?"
            level = m.get("engagementLevel") or m.get("engagement_level") or "?"
            pheno = m.get("phenotype") or "?"
            print(f"    [{i}] score={score}, level={level}, phenotype={pheno}")

    return {"input": events, "output": output, "delta": delta}


# ═══════════════════════════════════════════════════════════════════
#  MODULE 10/10b: Meal Response Correlator / Pattern Aggregator
# ═══════════════════════════════════════════════════════════════════
def test_module10():
    section("MODULE 10/10b: Meal Response Correlator + Pattern Aggregator")
    print(f"  Input:  enriched-patient-events-v1 → flink.meal-response → flink.meal-patterns")

    before_resp = topic_count("flink.meal-response")
    before_patt = topic_count("flink.meal-patterns")

    events = []
    meals = [
        {"meal_type": "breakfast", "carb_estimate_g": 45, "pre_meal_glucose": 145,
         "post_meal_glucose": 195, "sodium_mg": 800},
        {"meal_type": "lunch", "carb_estimate_g": 65, "pre_meal_glucose": 155,
         "post_meal_glucose": 225, "sodium_mg": 2200},
        {"meal_type": "dinner", "carb_estimate_g": 55, "pre_meal_glucose": 140,
         "post_meal_glucose": 200, "sodium_mg": 1500},
    ]
    for i, meal in enumerate(meals):
        ev = {
            "eventId": str(uuid.uuid4()), "patientId": PATIENT,
            "eventType": "PATIENT_REPORTED", "timestamp": NOW + i * 21600000,
            "sourceSystem": "flink-e2e", "correlationId": RUN_ID,
            "payload": {"type": "meal_log", **meal},
            "enrichmentData": {}, "enrichmentVersion": "1.0",
        }
        events.append(ev)
        ok = publish("enriched-patient-events-v1", ev)
        print(f"    {'✓' if ok else '✗'} {meal['meal_type']}: carbs={meal['carb_estimate_g']}g, glucose {meal['pre_meal_glucose']}→{meal['post_meal_glucose']}")

    print(f"  Waiting 15s for Module 10/10b processing...")
    time.sleep(15)

    delta_resp = topic_count("flink.meal-response") - before_resp
    delta_patt = topic_count("flink.meal-patterns") - before_patt
    output_resp = consume("flink.meal-response", timeout=8, patient_filter=PATIENT)
    output_patt = consume("flink.meal-patterns", timeout=8, patient_filter=PATIENT)

    print(f"  M10 output:  +{delta_resp} on flink.meal-response, filtered: {len(output_resp)}")
    print(f"  M10b output: +{delta_patt} on flink.meal-patterns, filtered: {len(output_patt)}")
    if output_resp:
        for i, m in enumerate(output_resp[:2], 1):
            print(f"    [M10-{i}] {json.dumps(m)[:120]}...")
    if output_patt:
        for i, m in enumerate(output_patt[:2], 1):
            iauc = m.get("meanIAUC") or m.get("mean_iauc") or "?"
            salt = m.get("saltSensitivityClass") or m.get("salt_sensitivity_class") or "?"
            print(f"    [M10b-{i}] meanIAUC={iauc}, salt={salt}")

    return {"input": events, "output_meal_response": output_resp,
            "output_meal_patterns": output_patt,
            "delta_response": delta_resp, "delta_patterns": delta_patt}


# ═══════════════════════════════════════════════════════════════════
#  MODULE 11/11b: Activity Response / Fitness Pattern Aggregator
# ═══════════════════════════════════════════════════════════════════
def test_module11():
    section("MODULE 11/11b: Activity Response + Fitness Pattern Aggregator")
    print(f"  Input:  enriched-patient-events-v1 → flink.activity-response → flink.fitness-patterns")

    before_resp = topic_count("flink.activity-response")
    before_patt = topic_count("flink.fitness-patterns")

    events = []
    activities = [
        {"steps": 3200, "active_minutes": 22, "calories": 180, "hr_avg": 95, "hr_max": 118},
        {"steps": 4500, "active_minutes": 35, "calories": 250, "hr_avg": 102, "hr_max": 128},
        {"steps": 2800, "active_minutes": 18, "calories": 150, "hr_avg": 88, "hr_max": 105},
    ]
    for i, act in enumerate(activities):
        ev = {
            "eventId": str(uuid.uuid4()), "patientId": PATIENT,
            "eventType": "DEVICE_READING", "timestamp": NOW + i * DAY_MS,
            "sourceSystem": "flink-e2e", "correlationId": RUN_ID,
            "payload": {"type": "activity", "steps": act["steps"],
                        "active_minutes": act["active_minutes"],
                        "calories_burned": act["calories"],
                        "heart_rate_avg": act["hr_avg"],
                        "heart_rate_max": act["hr_max"]},
            "enrichmentData": {}, "enrichmentVersion": "1.0",
        }
        events.append(ev)
        ok = publish("enriched-patient-events-v1", ev)
        print(f"    {'✓' if ok else '✗'} Day {i}: steps={act['steps']}, active_min={act['active_minutes']}")

    print(f"  Waiting 15s for Module 11/11b processing...")
    time.sleep(15)

    delta_resp = topic_count("flink.activity-response") - before_resp
    delta_patt = topic_count("flink.fitness-patterns") - before_patt
    output_resp = consume("flink.activity-response", timeout=8, patient_filter=PATIENT)
    output_patt = consume("flink.fitness-patterns", timeout=8, patient_filter=PATIENT)

    print(f"  M11 output:  +{delta_resp} on flink.activity-response, filtered: {len(output_resp)}")
    print(f"  M11b output: +{delta_patt} on flink.fitness-patterns, filtered: {len(output_patt)}")
    if output_resp:
        for i, m in enumerate(output_resp[:2], 1):
            print(f"    [M11-{i}] {json.dumps(m)[:120]}...")
    if output_patt:
        for i, m in enumerate(output_patt[:2], 1):
            vo2 = m.get("estimatedVO2max") or m.get("estimated_vo2max") or "?"
            met = m.get("totalMetMinutes") or m.get("total_met_minutes") or "?"
            print(f"    [M11b-{i}] VO2max={vo2}, MET-min={met}")

    return {"input": events, "output_activity_response": output_resp,
            "output_fitness_patterns": output_patt,
            "delta_response": delta_resp, "delta_patterns": delta_patt}


# ═══════════════════════════════════════════════════════════════════
#  MODULE 13: Clinical State Synchroniser (multi-day simulation)
# ═══════════════════════════════════════════════════════════════════
def test_module13():
    section("MODULE 13: Clinical State Synchroniser (14-day simulation)")
    print(f"  Input:  all upstream output topics (direct injection)")
    print(f"  Output: clinical.state-change-events")
    print(f"  Strategy: inject 14 days of data to trigger snapshot rotation + CKM velocity")

    before = topic_count("clinical.state-change-events")
    m13_patient = "e2e-rajesh-m13-sim-002"

    events_sent = 0

    # Simulate 14 days: each day has BP, engagement, meal, fitness data
    # Week 1: stable but elevated  |  Week 2: deteriorating
    for day in range(14):
        ts = NOW - (14 - day) * DAY_MS
        deteriorating = day >= 7  # week 2 gets worse

        # BP Variability (Module 7 output format)
        arv = 14.0 + (day * 0.8 if deteriorating else 0)  # rising ARV in week 2
        mean_sbp = 148.0 + (day * 2.0 if deteriorating else 0)
        bp_ev = {
            "id": str(uuid.uuid4()), "patientId": m13_patient,
            "eventType": "VITAL_SIGN", "eventTime": ts,
            "sourceSystem": "flink-e2e-sim", "correlationId": RUN_ID,
            "payload": {
                "arv": round(arv, 1), "arv_sbp_7d": round(arv, 1),
                "variability_classification": "HIGH" if arv > 18 else "MODERATE",
                "mean_sbp": round(mean_sbp, 1), "mean_sbp_7d": round(mean_sbp, 1),
                "mean_dbp": 92.0, "mean_dbp_7d": 92.0,
                "morning_surge_magnitude": 25.0 + (day * 2 if deteriorating else 0),
                "dip_classification": "NON_DIPPER" if deteriorating else "DIPPER",
                "arv_contributing_dates": [
                    time.strftime("%Y-%m-%d", time.gmtime((ts - i * DAY_MS) / 1000))
                    for i in range(min(day + 1, 4))
                ],
            },
        }
        publish("flink.bp-variability-metrics", bp_ev)
        events_sent += 1

        # Engagement (Module 9 output format)
        score = 0.75 - (day * 0.03 if deteriorating else 0)
        eng_ev = {
            "id": str(uuid.uuid4()), "patientId": m13_patient,
            "eventType": "PATIENT_REPORTED", "eventTime": ts + 1000,
            "sourceSystem": "flink-e2e-sim", "correlationId": RUN_ID,
            "payload": {
                "compositeScore": round(max(score, 0.2), 2),
                "engagementLevel": "RED" if score < 0.4 else ("AMBER" if score < 0.6 else "GREEN"),
                "phenotype": "DECLINING_ENGAGER" if score < 0.5 else "ACTIVE",
                "dataTier": "TIER_2",
            },
        }
        publish("flink.engagement-signals", eng_ev)
        events_sent += 1

        # Meal patterns (Module 10b output format) - every 3 days
        if day % 3 == 0:
            iauc = 32.0 + (day * 1.5 if deteriorating else 0)
            meal_ev = {
                "id": str(uuid.uuid4()), "patientId": m13_patient,
                "eventType": "PATIENT_REPORTED", "eventTime": ts + 2000,
                "sourceSystem": "flink-e2e-sim", "correlationId": RUN_ID,
                "payload": {
                    "meanIAUC": round(iauc, 1),
                    "medianExcursion": round(40.0 + day * 1.2, 1),
                    "saltSensitivityClass": "HIGH" if deteriorating else "MODERATE",
                    "saltBeta": round(0.4 + (0.03 * day if deteriorating else 0), 2),
                },
            }
            publish("flink.meal-patterns", meal_ev)
            events_sent += 1

        # Fitness patterns (Module 11b output format) - every 3 days
        if day % 3 == 1:
            vo2 = 28.0 - (day * 0.3 if deteriorating else 0)
            fit_ev = {
                "id": str(uuid.uuid4()), "patientId": m13_patient,
                "eventType": "DEVICE_READING", "eventTime": ts + 3000,
                "sourceSystem": "flink-e2e-sim", "correlationId": RUN_ID,
                "payload": {
                    "estimatedVO2max": round(max(vo2, 20), 1),
                    "vo2maxTrend": round(-0.5 if deteriorating else 0.1, 1),
                    "totalMetMinutes": round(150 - (day * 5 if deteriorating else 0), 0),
                    "meanExerciseGlucoseDelta": -8.0,
                },
            }
            publish("flink.fitness-patterns", fit_ev)
            events_sent += 1

        # Comorbidity alert (Module 8) - day 10 only
        if day == 10:
            cid_ev = {
                "id": str(uuid.uuid4()), "patientId": m13_patient,
                "eventType": "CLINICAL_DOCUMENT", "eventTime": ts + 4000,
                "sourceSystem": "flink-e2e-sim", "correlationId": RUN_ID,
                "payload": {
                    "ruleId": "CID-03", "severity": "PAUSE",
                    "interaction_type": "DRUG_DISEASE",
                    "description": "Metformin dose review — eGFR declining below 45",
                },
            }
            publish("alerts.comorbidity-interactions", cid_ev)
            events_sent += 1

    print(f"  Injected: {events_sent} events across 14 simulated days")
    print(f"  Week 1 (days 0-6): stable baseline")
    print(f"  Week 2 (days 7-13): deteriorating BP, engagement, glycemic control")

    print(f"  Waiting 45s for Module 13 coalescing + snapshot rotation...")
    time.sleep(45)

    after = topic_count("clinical.state-change-events")
    delta = after - before
    output = consume("clinical.state-change-events", timeout=12, patient_filter=m13_patient)

    print(f"\n  Output: +{delta} messages on clinical.state-change-events")
    print(f"  Rajesh-sim-filtered: {len(output)} messages")

    if output:
        for i, m in enumerate(output, 1):
            ct = m.get("change_type", "?")
            pri = m.get("priority", "?")
            action = m.get("recommended_action", "?")
            comp = m.get("data_completeness_at_change", 0)
            ckm = m.get("ckm_velocity_at_change", {})
            ckm_score = ckm.get("composite_score", 0)
            ckm_class = ckm.get("composite_classification", "?")
            domains = ckm.get("domains_deteriorating", 0)
            dom_vel = ckm.get("domain_velocities", {})
            print(f"\n    ── State Change #{i} ──")
            print(f"    change_type:           {ct}")
            print(f"    priority:              {pri}")
            print(f"    recommended_action:     {action}")
            print(f"    data_completeness:      {comp:.3f}")
            print(f"    ckm_composite_score:    {ckm_score}")
            print(f"    ckm_classification:     {ckm_class}")
            print(f"    domains_deteriorating:  {domains}")
            if dom_vel:
                print(f"    domain_velocities:      {json.dumps(dom_vel)}")

    return {"patient_id": m13_patient, "events_injected": events_sent,
            "output": output, "delta": delta}


# ═══════════════════════════════════════════════════════════════════
#  MAIN
# ═══════════════════════════════════════════════════════════════════
def main():
    section(f"FULL MODULE 7→13 E2E TEST — {RUN_ID}")
    print(f"  Patient: {PATIENT}")
    print(f"  Testing each module individually with proper input format")

    results = {}

    results["module7"] = test_module7()
    results["module8"] = test_module8()
    results["module9"] = test_module9()
    results["module10_10b"] = test_module10()
    results["module11_11b"] = test_module11()
    results["module13"] = test_module13()

    # ── Final Summary ──
    section("FINAL SUMMARY")

    modules = [
        ("Module 7",  "flink.bp-variability-metrics",     results["module7"].get("delta", 0)),
        ("Module 8",  "alerts.comorbidity-interactions",   results["module8"].get("delta", 0)),
        ("Module 9",  "flink.engagement-signals",          results["module9"].get("delta", 0)),
        ("Module 10", "flink.meal-response",               results["module10_10b"].get("delta_response", 0)),
        ("Module 10b","flink.meal-patterns",               results["module10_10b"].get("delta_patterns", 0)),
        ("Module 11", "flink.activity-response",           results["module11_11b"].get("delta_response", 0)),
        ("Module 11b","flink.fitness-patterns",            results["module11_11b"].get("delta_patterns", 0)),
        ("Module 13", "clinical.state-change-events",      results["module13"].get("delta", 0)),
    ]

    for name, topic, delta in modules:
        status = "✓ PASS" if delta > 0 else "— NO OUTPUT"
        print(f"  {status:12s}  {name:12s}  +{delta:3d} on {topic}")

    flowing = sum(1 for _, _, d in modules if d > 0)
    print(f"\n  {flowing}/{len(modules)} modules produced output")

    # Save JSON
    out_path = "/Volumes/Vaidshala/cardiofit/backend/shared-infrastructure/flink-processing/test-data/e2e-rajesh-all-modules.json"
    with open(out_path, "w") as f:
        json.dump({
            "test_run": RUN_ID,
            "timestamp": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "patient_id": PATIENT,
            "results": {k: {kk: vv for kk, vv in v.items() if kk != "input"}
                        for k, v in results.items()},
        }, f, indent=2, default=str)
    print(f"\n  Full I/O JSON saved → {out_path}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
