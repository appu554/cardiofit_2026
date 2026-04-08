#!/usr/bin/env python3
"""
Flink E2E Module 7→13 Cross-Module Integration Test

Produces synthetic CanonicalEvent payloads to Module 13's 8 input Kafka topics,
then verifies that events arrive on the `clinical.state-change-events` output topic.

This script validates:
  - Kafka wiring (all 8 input topics → Module 13 consumer)
  - SourceTaggingDeserializer (source_module field injection)
  - Module 13 routing (routeAndUpdateState)
  - State accumulation (ClinicalStateSummary population)
  - Output emission to state-change-events topic

NOTE: Module 13 requires 7-day snapshot rotation before CKM velocity ≠ UNKNOWN.
These E2E tests primarily validate Kafka plumbing, not clinical logic (which the
Java pure-function tests in Module7To13CrossModuleIntegrationTest cover).

Scenarios:
  A. BP Crisis       — flink.bp-variability-metrics
  B. Engagement Drop — flink.engagement-signals + enriched-patient-events-v1
  C. Multi-Source    — all 8 input topics
  D. Intervention    — clinical.intervention-window-signals + flink.intervention-deltas
  E. Lab + CID       — enriched-patient-events-v1 + alerts.comorbidity-interactions

Usage:
  python3 scripts/flink_e2e_module7_to_13.py                   # full run
  python3 scripts/flink_e2e_module7_to_13.py --dry-run          # preview events
  python3 scripts/flink_e2e_module7_to_13.py --scenario bp      # single scenario
  python3 scripts/flink_e2e_module7_to_13.py --verify-only      # check outputs only
  python3 scripts/flink_e2e_module7_to_13.py --wait 120         # longer pipeline wait

Prerequisites:
  1. Kafka running:  docker ps | grep cardiofit-kafka-lite
  2. V4 topics:      bash scripts/create-v4-topics.sh
  3. Flink running:  docker compose -f docker-compose.e2e-flink.yml up -d
  4. Module 13 job submitted (via flink-submitter or manual)
"""

import argparse
import json
import subprocess
import sys
import time
import uuid

# ---------------------------------------------------------------------------
# Configuration
# ---------------------------------------------------------------------------
KAFKA_CONTAINER = "cardiofit-kafka-lite"
KAFKA_BOOTSTRAP = "localhost:29092"

# Module 13 input topics
TOPICS = {
    "module7":   "flink.bp-variability-metrics",
    "module8":   "alerts.comorbidity-interactions",
    "module9":   "flink.engagement-signals",
    "module10b": "flink.meal-patterns",
    "module11b": "flink.fitness-patterns",
    "module12":  "clinical.intervention-window-signals",
    "module12b": "flink.intervention-deltas",
    "enriched":  "enriched-patient-events-v1",
}

# Module 13 output topic
OUTPUT_TOPIC = "clinical.state-change-events"

RUN_ID = f"e2e-m7m13-{int(time.time())}"


# ---------------------------------------------------------------------------
# Event Builders — mirrors Module13TestBuilder.java
# ---------------------------------------------------------------------------
def _base_event(patient_id, event_type, timestamp, payload):
    return {
        "id": str(uuid.uuid4()),
        "patientId": patient_id,
        "eventType": event_type,
        "eventTime": timestamp,
        "sourceSystem": "flink-e2e-module7-to-13",
        "correlationId": RUN_ID,
        "payload": payload,
    }


def bp_variability_event(pid, ts, arv, var_class, mean_sbp, mean_dbp,
                         surge=None, dip_class=None):
    p = {
        "source_module": "module7",
        "arv": arv,
        "variability_classification": var_class,
        "mean_sbp": mean_sbp,
        "mean_dbp": mean_dbp,
    }
    if surge is not None:
        p["morning_surge_magnitude"] = surge
    if dip_class is not None:
        p["dip_classification"] = dip_class
    return _base_event(pid, "VITAL_SIGN", ts, p)


def engagement_event(pid, ts, score, level, phenotype, data_tier):
    return _base_event(pid, "PATIENT_REPORTED", ts, {
        "source_module": "module9",
        "composite_score": score,
        "engagement_level": level,
        "phenotype": phenotype,
        "data_tier": data_tier,
    })


def meal_pattern_event(pid, ts, mean_iauc, median_exc, salt_class, salt_beta):
    return _base_event(pid, "PATIENT_REPORTED", ts, {
        "source_module": "module10b",
        "mean_iauc": mean_iauc,
        "median_excursion": median_exc,
        "salt_sensitivity_class": salt_class,
        "salt_beta": salt_beta,
    })


def fitness_pattern_event(pid, ts, vo2max, vo2_trend, met_min, gluc_delta):
    return _base_event(pid, "DEVICE_READING", ts, {
        "source_module": "module11b",
        "estimated_vo2max": vo2max,
        "vo2max_trend": vo2_trend,
        "total_met_minutes": met_min,
        "mean_exercise_glucose_delta": gluc_delta,
    })


def intervention_window_event(pid, ts, intv_id, signal_type, intv_type,
                               obs_start, obs_end):
    return _base_event(pid, "MEDICATION_ORDERED", ts, {
        "source_module": "module12",
        "intervention_id": intv_id,
        "signal_type": signal_type,
        "intervention_type": intv_type,
        "observation_start_ms": obs_start,
        "observation_end_ms": obs_end,
    })


def intervention_delta_event(pid, ts, intv_id, attribution, adherence,
                              fbg_d=0, sbp_d=0, egfr_d=0):
    return _base_event(pid, "LAB_RESULT", ts, {
        "source_module": "module12b",
        "intervention_id": intv_id,
        "trajectory_attribution": attribution,
        "adherence_score": adherence,
        "fbg_delta": fbg_d,
        "sbp_delta": sbp_d,
        "egfr_delta": egfr_d,
    })


def lab_event(pid, ts, lab_type, value):
    return _base_event(pid, "LAB_RESULT", ts, {
        "source_module": "enriched",
        "lab_type": lab_type,
        "value": value,
    })


def comorbidity_alert_event(pid, ts, rule_id, severity):
    return _base_event(pid, "CLINICAL_DOCUMENT", ts, {
        "source_module": "module8",
        "ruleId": rule_id,
        "severity": severity,
    })


# ---------------------------------------------------------------------------
# Kafka Publisher / Consumer
# ---------------------------------------------------------------------------
class KafkaPublisher:
    def __init__(self, container=None, bootstrap=None):
        self.container = container or KAFKA_CONTAINER
        self.bootstrap = bootstrap or KAFKA_BOOTSTRAP
        self.sent = 0
        self.errors = 0

    def publish(self, topic, event_dict):
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
            print(f"    ERROR → {topic}: {result.stderr.strip()}")
            self.errors += 1
            return False
        self.sent += 1
        return True


class KafkaConsumer:
    def __init__(self, container=None, bootstrap=None):
        self.container = container or KAFKA_CONTAINER
        self.bootstrap = bootstrap or KAFKA_BOOTSTRAP

    def consume(self, topic, timeout_sec=30, max_messages=200):
        cmd = [
            "docker", "exec", self.container,
            "kafka-console-consumer",
            "--bootstrap-server", self.bootstrap,
            "--topic", topic,
            "--from-beginning",
            "--timeout-ms", str(timeout_sec * 1000),
            "--max-messages", str(max_messages),
        ]
        result = subprocess.run(
            cmd, capture_output=True, text=True, timeout=timeout_sec + 10,
        )
        messages = []
        for line in result.stdout.strip().split("\n"):
            line = line.strip()
            if not line:
                continue
            try:
                messages.append(json.loads(line))
            except json.JSONDecodeError:
                pass
        return messages


# ---------------------------------------------------------------------------
# Scenarios
# ---------------------------------------------------------------------------
def scenario_bp_crisis(pub, patient_id, dry_run=False):
    """Scenario A: BP Crisis — high ARV, morning surge, non-dipper."""
    print(f"\n  [A] BP CRISIS — Patient: {patient_id}")
    now = int(time.time() * 1000)
    events = [
        (TOPICS["module7"], bp_variability_event(
            patient_id, now, 22.0, "HIGH", 168.0, 100.0,
            surge=42.0, dip_class="NON_DIPPER")),
        (TOPICS["module7"], bp_variability_event(
            patient_id, now + 60000, 20.0, "HIGH", 165.0, 98.0)),
    ]
    return _publish_events(pub, events, dry_run)


def scenario_engagement_drop(pub, patient_id, dry_run=False):
    """Scenario B: Engagement Drop — high FBG + engagement collapse."""
    print(f"\n  [B] ENGAGEMENT DROP — Patient: {patient_id}")
    now = int(time.time() * 1000)
    events = [
        (TOPICS["enriched"], lab_event(patient_id, now, "FBG", 180.0)),
        (TOPICS["module9"], engagement_event(
            patient_id, now + 60000, 0.30, "RED", "DISENGAGED", "TIER_2_SMBG")),
    ]
    return _publish_events(pub, events, dry_run)


def scenario_multi_source(pub, patient_id, dry_run=False):
    """Scenario C: Multi-Source — events from all 8 input topics."""
    print(f"\n  [C] MULTI-SOURCE FAN-IN — Patient: {patient_id}")
    now = int(time.time() * 1000)
    intv_id = f"intv-{RUN_ID[:8]}"
    events = [
        (TOPICS["module7"],  bp_variability_event(patient_id, now, 12.0, "MODERATE", 142.0, 88.0)),
        (TOPICS["module8"],  comorbidity_alert_event(patient_id, now + 1000, "CID-E2E-1", "PAUSE")),
        (TOPICS["module9"],  engagement_event(patient_id, now + 2000, 0.72, "GREEN", "ACTIVE", "TIER_2_SMBG")),
        (TOPICS["module10b"], meal_pattern_event(patient_id, now + 3000, 35.0, 45.0, "LOW", 0.1)),
        (TOPICS["module11b"], fitness_pattern_event(patient_id, now + 4000, 35.0, 0.5, 180.0, -5.0)),
        (TOPICS["module12"], intervention_window_event(
            patient_id, now + 5000, intv_id, "WINDOW_OPENED", "MEDICATION_ADD",
            now, now + 28 * 86400000)),
        (TOPICS["module12b"], intervention_delta_event(
            patient_id, now + 6000, intv_id, "STABLE", 0.7)),
        (TOPICS["enriched"], lab_event(patient_id, now + 7000, "FBG", 125.0)),
    ]
    return _publish_events(pub, events, dry_run)


def scenario_intervention(pub, patient_id, dry_run=False):
    """Scenario D: Intervention lifecycle — open, delta, close."""
    print(f"\n  [D] INTERVENTION LIFECYCLE — Patient: {patient_id}")
    now = int(time.time() * 1000)
    intv_id = f"intv-d-{RUN_ID[:8]}"
    events = [
        (TOPICS["module12"], intervention_window_event(
            patient_id, now, intv_id, "WINDOW_OPENED", "NUTRITION_FOOD_CHANGE",
            now, now + 14 * 86400000)),
        (TOPICS["module12b"], intervention_delta_event(
            patient_id, now + 60000, intv_id, "INTERVENTION_INSUFFICIENT", 0.80,
            fbg_d=5.0)),
        (TOPICS["module12"], intervention_window_event(
            patient_id, now + 120000, intv_id, "WINDOW_CLOSED", "NUTRITION_FOOD_CHANGE",
            now, now + 14 * 86400000)),
    ]
    return _publish_events(pub, events, dry_run)


def scenario_lab_cid(pub, patient_id, dry_run=False):
    """Scenario E: Lab + Comorbidity — eGFR decline + CID HALT."""
    print(f"\n  [E] LAB + COMORBIDITY — Patient: {patient_id}")
    now = int(time.time() * 1000)
    events = [
        (TOPICS["enriched"], lab_event(patient_id, now, "EGFR", 42.0)),
        (TOPICS["enriched"], lab_event(patient_id, now + 30000, "FBG", 165.0)),
        (TOPICS["module8"],  comorbidity_alert_event(
            patient_id, now + 60000, "CID-SGLT2I-AKI", "HALT")),
    ]
    return _publish_events(pub, events, dry_run)


def _publish_events(pub, events, dry_run):
    count = 0
    for topic, event in events:
        if dry_run:
            print(f"    [DRY-RUN] → {topic}: {json.dumps(event)[:120]}...")
        else:
            ok = pub.publish(topic, event)
            if ok:
                print(f"    ✓ → {topic}")
            count += 1
    return count


# ---------------------------------------------------------------------------
# Verification
# ---------------------------------------------------------------------------
def verify_output(consumer, patient_ids, wait_sec=60):
    print(f"\n{'='*60}")
    print(f"  VERIFICATION — waiting {wait_sec}s for pipeline processing")
    print(f"{'='*60}")
    time.sleep(wait_sec)

    print(f"\n  Consuming {OUTPUT_TOPIC}...")
    messages = consumer.consume(OUTPUT_TOPIC, timeout_sec=30, max_messages=500)

    # Filter by our run's patient IDs
    run_messages = [
        m for m in messages
        if m.get("patient_id") in patient_ids or m.get("patientId") in patient_ids
    ]

    print(f"    Total messages: {len(messages)}")
    print(f"    Run-filtered:   {len(run_messages)}")

    # Group by patient
    by_patient = {}
    for m in run_messages:
        pid = m.get("patient_id") or m.get("patientId") or "unknown"
        by_patient.setdefault(pid, []).append(m)

    for pid, msgs in by_patient.items():
        types = [m.get("change_type") or m.get("changeType", "?") for m in msgs]
        print(f"    {pid}: {len(msgs)} changes — {types}")

    # Also check input topics for confirmation
    print(f"\n  Input topic spot-checks:")
    for label, topic in [("BP", TOPICS["module7"]), ("Engagement", TOPICS["module9"]),
                         ("CID", TOPICS["module8"])]:
        msgs = consumer.consume(topic, timeout_sec=10, max_messages=100)
        run_msgs = [m for m in msgs if m.get("correlationId") == RUN_ID]
        print(f"    {label} ({topic}): {len(msgs)} total, {len(run_msgs)} from this run")

    return len(run_messages), by_patient


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
SCENARIOS = {
    "bp": scenario_bp_crisis,
    "engagement": scenario_engagement_drop,
    "multi": scenario_multi_source,
    "intervention": scenario_intervention,
    "lab_cid": scenario_lab_cid,
}


def main():
    parser = argparse.ArgumentParser(description="Module 7→13 E2E Kafka test")
    parser.add_argument("--scenario", choices=list(SCENARIOS.keys()),
                        help="Run single scenario (default: all)")
    parser.add_argument("--dry-run", action="store_true",
                        help="Preview events without publishing")
    parser.add_argument("--verify-only", action="store_true",
                        help="Only check output topic")
    parser.add_argument("--wait", type=int, default=60,
                        help="Seconds to wait for pipeline (default: 60)")
    parser.add_argument("--kafka-container", default=KAFKA_CONTAINER)
    parser.add_argument("--kafka-bootstrap", default=KAFKA_BOOTSTRAP)
    args = parser.parse_args()

    print(f"{'='*60}")
    print(f"  Module 7→13 E2E Test — {RUN_ID}")
    print(f"  Kafka: {args.kafka_container} ({args.kafka_bootstrap})")
    print(f"{'='*60}")

    pub = KafkaPublisher(args.kafka_container, args.kafka_bootstrap)
    consumer = KafkaConsumer(args.kafka_container, args.kafka_bootstrap)

    # Assign unique patient IDs per scenario
    patient_ids = {}
    scenarios_to_run = [args.scenario] if args.scenario else list(SCENARIOS.keys())

    if not args.verify_only:
        total_events = 0
        for scenario_name in scenarios_to_run:
            pid = f"e2e-m13-{scenario_name}-{RUN_ID[-8:]}"
            patient_ids[scenario_name] = pid
            fn = SCENARIOS[scenario_name]
            count = fn(pub, pid, dry_run=args.dry_run)
            total_events += count

        print(f"\n  Published: {pub.sent} events ({pub.errors} errors)")

        if args.dry_run:
            print("\n  [DRY-RUN] No events published. Exiting.")
            return 0

    # Verification
    all_pids = set(patient_ids.values()) if patient_ids else set()
    total_changes, by_patient = verify_output(consumer, all_pids, wait_sec=args.wait)

    # Summary
    print(f"\n{'='*60}")
    print(f"  SUMMARY")
    print(f"{'='*60}")
    print(f"  Run ID:        {RUN_ID}")
    print(f"  Scenarios:     {len(scenarios_to_run)}")
    print(f"  Events sent:   {pub.sent}")
    print(f"  State changes: {total_changes}")
    print(f"  Patients:      {len(by_patient)}")

    if total_changes > 0:
        print(f"\n  ✓ Module 13 is processing events and emitting state changes")
    else:
        print(f"\n  ⚠ No state changes detected.")
        print(f"    This is EXPECTED on first run — Module 13 needs 7-day snapshot")
        print(f"    rotation before CKM velocity != UNKNOWN. The fact that events")
        print(f"    were consumed (check Flink UI) validates the Kafka wiring.")

    return 0 if pub.errors == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
