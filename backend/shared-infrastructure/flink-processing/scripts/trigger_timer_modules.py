#!/usr/bin/env python3
"""
Trigger Timer-Dependent Modules (M10 Meal Response + M11 Activity Response)
===========================================================================
Sends backdated events so that Flink's processing-time timers fire immediately.

Key insight:
  - M10 fires at mealTimestamp + 3h05m (processing time).
    If we send a meal event timestamped 4h ago, the timer target is already
    in the past -> Flink fires immediately on the next watermark advance.
  - M11 fires at activityEnd + 2h05m (processing time).
    If we send an activity event timestamped 3h ago (30min walk ended 2.5h ago),
    the timer target is already passed -> immediate fire.

Usage:
    python trigger_timer_modules.py [--m10-only | --m11-only]
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
RUN_ID = f"e2e-timer-trigger-{_TS}"

HOUR_MS = 3_600_000
MIN_MS = 60_000


def publish(topic: str, msg: dict) -> bool:
    """Publish a single JSON message to a Kafka topic via docker exec."""
    j = json.dumps(msg, separators=(",", ":"))
    r = subprocess.run(
        ["docker", "exec", "-i", KAFKA, "kafka-console-producer",
         "--bootstrap-server", BOOTSTRAP, "--topic", topic],
        input=j, capture_output=True, text=True, timeout=15,
    )
    return r.returncode == 0


def consume(topic: str, timeout: int = 20, max_msgs: int = 500) -> list:
    """Consume messages from a Kafka topic (from beginning)."""
    r = subprocess.run(
        ["docker", "exec", KAFKA, "kafka-console-consumer",
         "--bootstrap-server", BOOTSTRAP, "--topic", topic,
         "--from-beginning", "--timeout-ms", str(timeout * 1000),
         "--max-messages", str(max_msgs)],
        capture_output=True, text=True, timeout=timeout + 30,
    )
    msgs = []
    for line in r.stdout.strip().split("\n"):
        if not line.strip():
            continue
        try:
            msgs.append(json.loads(line.strip()))
        except json.JSONDecodeError:
            pass
    return msgs


def topic_count(topic: str) -> int:
    """Get current offset (message count) for a topic."""
    r = subprocess.run(
        ["docker", "exec", KAFKA, "kafka-run-class",
         "kafka.tools.GetOffsetShell", "--broker-list", "localhost:9092",
         "--topic", topic, "--time", "-1"],
        capture_output=True, text=True, timeout=10,
    )
    total = 0
    for line in r.stdout.strip().split("\n"):
        parts = line.split(":")
        if len(parts) >= 3:
            total += int(parts[2])
    return total


def enriched_event(event_type: str, ts: int, payload: dict) -> dict:
    """Build a CanonicalEvent for enriched-patient-events-v1."""
    return {
        "eventId": str(uuid.uuid4()),
        "patientId": PATIENT,
        "eventType": event_type,
        "timestamp": ts,
        "sourceSystem": "flink-e2e-timer-trigger",
        "correlationId": RUN_ID,
        "payload": payload,
        "enrichmentData": {},
        "enrichmentVersion": "1.0",
    }


# =====================================================================
#  M10 MEAL RESPONSE EVENTS (backdated 4 hours)
# =====================================================================

def build_m10_events() -> list:
    """
    Build a meal session with CGM glucose + BP, all backdated 4 hours.

    Timeline (relative to now):
      -4h30m  Pre-meal CGM glucose (baseline)
      -4h05m  Pre-meal BP
      -4h00m  MEAL_LOG (lunch, high-carb South Indian)
      -3h30m  Post-meal CGM +30m (glucose rising)
      -3h00m  Post-meal CGM +60m (peak)
      -3h00m  Post-meal BP  +60m
      -2h30m  Post-meal CGM +90m (descending)
      -2h00m  Post-meal CGM +120m (return toward baseline)

    M10 timer target = meal_time + 3h05m = now - 4h + 3h05m = now - 55m
    Since now - 55m is already in the past, Flink fires immediately.
    """
    now_ms = int(time.time() * 1000)
    meal_time = now_ms - 4 * HOUR_MS  # 4 hours ago
    events = []

    # Pre-meal CGM glucose (30 min before meal)
    events.append(enriched_event("DEVICE_READING", meal_time - 30 * MIN_MS, {
        "glucose_value": 138,
        "source": "CGM",
        "data_tier": "TIER_1_CGM",
    }))

    # Pre-meal BP (5 min before meal)
    events.append(enriched_event("VITAL_SIGN", meal_time - 5 * MIN_MS, {
        "systolic_bp": 168,
        "diastolic_bp": 100,
    }))

    # MEAL_LOG -- opens the M10 session
    events.append(enriched_event("PATIENT_REPORTED", meal_time, {
        "report_type": "MEAL_LOG",
        "meal_type": "lunch",
        "carb_grams": 72,
        "protein_grams": 22,
        "fat_grams": 16,
        "sodium_mg": 1800,
        "protein_flag": True,
        "data_tier": "TIER_1_CGM",
    }))

    # Post-meal CGM readings at +30m, +60m, +90m, +120m
    cgm_curve = [
        (30, 192),   # rising
        (60, 218),   # peak
        (90, 198),   # descending
        (120, 162),  # returning toward baseline
    ]
    for offset_min, glucose in cgm_curve:
        events.append(enriched_event("DEVICE_READING",
            meal_time + offset_min * MIN_MS, {
                "glucose_value": glucose,
                "source": "CGM",
                "data_tier": "TIER_1_CGM",
            }))

    # Post-meal BP (1h after meal)
    events.append(enriched_event("VITAL_SIGN", meal_time + 60 * MIN_MS, {
        "systolic_bp": 176,
        "diastolic_bp": 106,
    }))

    return events


# =====================================================================
#  M11 ACTIVITY RESPONSE EVENTS (backdated 3 hours)
# =====================================================================

def build_m11_events() -> list:
    """
    Build an activity session with HR/glucose/BP, all backdated 3 hours.

    Timeline (relative to now):
      -3h10m  Resting HR baseline
      -3h05m  Pre-exercise BP
      -3h03m  Pre-exercise CGM glucose
      -3h00m  ACTIVITY_LOG (30min brisk walk)
      -2h55m  Exercise HR at +5m
      -2h45m  Exercise HR at +15m + peak BP
      -2h35m  Exercise HR at +25m
      -2h30m  Activity ends
      -2h29m  Post-exercise HR recovery +1m
      -2h28m  Post-exercise HR recovery +2m
      -2h25m  Post-exercise HR recovery +5m
      -2h00m  Post-exercise glucose +30m
      -2h00m  Post-exercise BP +30m

    M11 timer target = activity_end + 2h05m = (now - 2.5h) + 2h05m = now - 25m
    Since now - 25m is already in the past, Flink fires immediately.
    """
    now_ms = int(time.time() * 1000)
    activity_start = now_ms - 3 * HOUR_MS  # 3 hours ago
    activity_end = activity_start + 30 * MIN_MS  # 30 min walk
    events = []

    # Resting HR baseline (10 min before)
    events.append(enriched_event("DEVICE_READING", activity_start - 10 * MIN_MS, {
        "heart_rate": 76,
        "source": "WEARABLE",
        "activity_state": "RESTING",
        "data_tier": "TIER_1_CGM",
    }))

    # Pre-exercise BP
    events.append(enriched_event("VITAL_SIGN", activity_start - 5 * MIN_MS, {
        "systolic_bp": 164,
        "diastolic_bp": 96,
        "resting_heart_rate": 76,
    }))

    # Pre-exercise CGM glucose
    events.append(enriched_event("DEVICE_READING", activity_start - 3 * MIN_MS, {
        "glucose_value": 155,
        "source": "CGM",
        "data_tier": "TIER_1_CGM",
    }))

    # ACTIVITY_LOG -- opens the M11 session
    events.append(enriched_event("PATIENT_REPORTED", activity_start, {
        "report_type": "ACTIVITY_LOG",
        "exercise_type": "BRISK_WALKING",
        "duration_minutes": 30,
        "patient_age": 58,
        "patient_sex": "M",
        "data_tier": "TIER_1_CGM",
    }))

    # Exercise HR readings during 30min walk
    exercise_hr = [(5, 108), (15, 120), (25, 114)]
    for offset_min, hr in exercise_hr:
        events.append(enriched_event("DEVICE_READING",
            activity_start + offset_min * MIN_MS, {
                "heart_rate": hr,
                "source": "WEARABLE",
                "data_tier": "TIER_1_CGM",
            }))

    # Peak exercise BP (at 15 min)
    events.append(enriched_event("VITAL_SIGN",
        activity_start + 15 * MIN_MS, {
            "systolic_bp": 190,
            "diastolic_bp": 106,
        }))

    # Post-exercise HR recovery: +1m, +2m, +5m after activity end
    recovery_hr = [(1, 104), (2, 92), (5, 84)]
    for offset_min, hr in recovery_hr:
        events.append(enriched_event("DEVICE_READING",
            activity_end + offset_min * MIN_MS, {
                "heart_rate": hr,
                "source": "WEARABLE",
                "data_tier": "TIER_1_CGM",
            }))

    # Post-exercise glucose (30 min after end)
    events.append(enriched_event("DEVICE_READING",
        activity_end + 30 * MIN_MS, {
            "glucose_value": 128,
            "source": "CGM",
            "data_tier": "TIER_1_CGM",
        }))

    # Post-exercise BP (30 min after end)
    events.append(enriched_event("VITAL_SIGN",
        activity_end + 30 * MIN_MS, {
            "systolic_bp": 156,
            "diastolic_bp": 90,
        }))

    return events


# =====================================================================
#  MAIN EXECUTION
# =====================================================================

def main() -> int:
    run_m10 = True
    run_m11 = True

    if "--m10-only" in sys.argv:
        run_m11 = False
    elif "--m11-only" in sys.argv:
        run_m10 = False

    now_ms = int(time.time() * 1000)
    print(f"{'=' * 70}")
    print(f"  TIMER-DEPENDENT MODULE TRIGGER")
    print(f"  Run:     {RUN_ID}")
    print(f"  Patient: {PATIENT}")
    print(f"  Mode:    {'M10 + M11' if (run_m10 and run_m11) else 'M10 only' if run_m10 else 'M11 only'}")
    print(f"{'=' * 70}")

    # Pre-counts
    m10_topic = "flink.meal-response"
    m11_topic = "flink.activity-response"
    before_m10 = topic_count(m10_topic) if run_m10 else 0
    before_m11 = topic_count(m11_topic) if run_m11 else 0

    if run_m10:
        print(f"\n  Pre-count {m10_topic}: {before_m10}")
    if run_m11:
        print(f"\n  Pre-count {m11_topic}: {before_m11}")

    # ── M10: Meal Response Events ──
    if run_m10:
        m10_events = build_m10_events()
        meal_ts = now_ms - 4 * HOUR_MS
        timer_target = meal_ts + 3 * HOUR_MS + 5 * MIN_MS
        timer_delta_min = (now_ms - timer_target) / MIN_MS

        print(f"\n  M10 MEAL RESPONSE ({len(m10_events)} events)")
        print(f"    Meal timestamp:  {4 * 60} min ago")
        print(f"    Timer target:    meal + 3h05m = {timer_delta_min:.0f} min ago (already passed)")
        print(f"    Expected:        Flink fires immediately")

        ok = 0
        for ev in m10_events:
            if publish("enriched-patient-events-v1", ev):
                ok += 1
                etype = ev["eventType"]
                p = ev["payload"]
                age_min = (now_ms - ev["timestamp"]) / MIN_MS
                if etype == "DEVICE_READING" and "glucose_value" in p:
                    print(f"      [{ok:2d}] CGM glucose={p['glucose_value']}  ({age_min:.0f}m ago)")
                elif etype == "VITAL_SIGN":
                    print(f"      [{ok:2d}] BP {p.get('systolic_bp')}/{p.get('diastolic_bp')}  ({age_min:.0f}m ago)")
                elif etype == "PATIENT_REPORTED":
                    print(f"      [{ok:2d}] MEAL_LOG {p.get('meal_type')} carb={p.get('carb_grams')}g  ({age_min:.0f}m ago)")
            else:
                print(f"      FAILED event #{ok + 1}")
            time.sleep(0.15)
        print(f"    Sent: {ok}/{len(m10_events)}")

    # ── M11: Activity Response Events ──
    if run_m11:
        m11_events = build_m11_events()
        activity_start_ts = now_ms - 3 * HOUR_MS
        activity_end_ts = activity_start_ts + 30 * MIN_MS
        timer_target = activity_end_ts + 2 * HOUR_MS + 5 * MIN_MS
        timer_delta_min = (now_ms - timer_target) / MIN_MS

        print(f"\n  M11 ACTIVITY RESPONSE ({len(m11_events)} events)")
        print(f"    Activity start:  {3 * 60} min ago")
        print(f"    Activity end:    {3 * 60 - 30} min ago (30min walk)")
        print(f"    Timer target:    end + 2h05m = {timer_delta_min:.0f} min ago (already passed)")
        print(f"    Expected:        Flink fires immediately")

        ok = 0
        for ev in m11_events:
            if publish("enriched-patient-events-v1", ev):
                ok += 1
                etype = ev["eventType"]
                p = ev["payload"]
                age_min = (now_ms - ev["timestamp"]) / MIN_MS
                if etype == "DEVICE_READING" and "heart_rate" in p:
                    state = p.get("activity_state", "")
                    print(f"      [{ok:2d}] HR={p['heart_rate']} {state}  ({age_min:.0f}m ago)")
                elif etype == "DEVICE_READING" and "glucose_value" in p:
                    print(f"      [{ok:2d}] CGM glucose={p['glucose_value']}  ({age_min:.0f}m ago)")
                elif etype == "VITAL_SIGN":
                    print(f"      [{ok:2d}] BP {p.get('systolic_bp')}/{p.get('diastolic_bp')}  ({age_min:.0f}m ago)")
                elif etype == "PATIENT_REPORTED":
                    print(f"      [{ok:2d}] ACTIVITY_LOG {p.get('exercise_type')} {p.get('duration_minutes')}min  ({age_min:.0f}m ago)")
            else:
                print(f"      FAILED event #{ok + 1}")
            time.sleep(0.15)
        print(f"    Sent: {ok}/{len(m11_events)}")

    # ── Wait for timers to fire ──
    print(f"\n{'=' * 70}")
    print(f"  Waiting 15s for Flink processing-time timers to fire...")
    print(f"{'=' * 70}")
    time.sleep(15)

    # ── Check output topics ──
    def is_ours(msg):
        return msg.get("patient_id") == PATIENT or msg.get("patientId") == PATIENT

    if run_m10:
        print(f"\n  ── M10 Meal Response Output ──")
        after_m10 = topic_count(m10_topic)
        delta = after_m10 - before_m10
        print(f"    Topic delta: +{delta}")

        m10_msgs = consume(m10_topic, timeout=10)
        ours = [m for m in m10_msgs if is_ours(m)]
        print(f"    Total messages: {len(m10_msgs)}, Ours: {len(ours)}")

        if ours:
            for i, m in enumerate(ours, 1):
                print(f"    Record #{i}:")
                print(f"      iAUC:              {m.get('iAUC', m.get('iauc', '?'))}")
                print(f"      glucoseExcursion:  {m.get('glucoseExcursion', m.get('glucose_excursion', '?'))}")
                print(f"      peakGlucose:       {m.get('peakGlucose', m.get('peak_glucose', '?'))}")
                print(f"      dataTier:          {m.get('dataTier', m.get('data_tier', '?'))}")
                print(f"      mealType:          {m.get('mealType', m.get('meal_type', '?'))}")
                print(f"      bpDelta:           {m.get('bpDelta', m.get('bp_delta', '?'))}")
        else:
            print(f"    No output yet -- timer may not have fired.")
            print(f"    Check: is M10 operator deployed? Is the enriched topic wired?")

    if run_m11:
        print(f"\n  ── M11 Activity Response Output ──")
        after_m11 = topic_count(m11_topic)
        delta = after_m11 - before_m11
        print(f"    Topic delta: +{delta}")

        m11_msgs = consume(m11_topic, timeout=10)
        ours = [m for m in m11_msgs if is_ours(m)]
        print(f"    Total messages: {len(m11_msgs)}, Ours: {len(ours)}")

        if ours:
            for i, m in enumerate(ours, 1):
                print(f"    Record #{i}:")
                print(f"      peakHR:         {m.get('peakHR', m.get('peak_hr', '?'))}")
                print(f"      HRR1:           {m.get('hrr1', m.get('HRR1', '?'))}")
                print(f"      metMinutes:     {m.get('metMinutes', m.get('met_minutes', '?'))}")
                print(f"      glucoseDelta:   {m.get('glucoseDelta', m.get('glucose_delta', '?'))}")
                print(f"      bpRecovery:     {m.get('bpRecovery', m.get('bp_recovery', '?'))}")
                print(f"      exerciseType:   {m.get('exerciseType', m.get('exercise_type', '?'))}")
        else:
            print(f"    No output yet -- timer may not have fired.")
            print(f"    Check: is M11 operator deployed? Is the enriched topic wired?")

    # ── Summary ──
    print(f"\n{'=' * 70}")
    print(f"  SUMMARY")
    print(f"{'=' * 70}")
    if run_m10:
        m10_out = len([m for m in consume(m10_topic, timeout=5) if is_ours(m)])
        marker = "PASS" if m10_out > 0 else "PENDING"
        print(f"    M10 Meal Response:     {marker} ({m10_out} outputs)")
    if run_m11:
        m11_out = len([m for m in consume(m11_topic, timeout=5) if is_ours(m)])
        marker = "PASS" if m11_out > 0 else "PENDING"
        print(f"    M11 Activity Response: {marker} ({m11_out} outputs)")
    print(f"{'=' * 70}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
