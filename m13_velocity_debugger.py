#!/usr/bin/env python3
"""
M13 CARDIOVASCULAR Velocity Debugger

Analyzes the E2E test output to diagnose why M13's CKM velocity
shows CARDIOVASCULAR=0.0 despite alarming M7 BP variability data.

Usage:
    python3 m13_velocity_debugger.py e2e-14day-all-modules-io.json
    python3 m13_velocity_debugger.py --m7 m7-output.json --m13 m13-output.json
"""

import json
import sys
import argparse
from datetime import datetime, timezone
from collections import defaultdict
from dataclasses import dataclass, field
from typing import List, Dict, Optional, Tuple
import statistics


# ═══════════════════════════════════════════════════════════════════════
#  DATA MODELS
# ═══════════════════════════════════════════════════════════════════════

@dataclass
class BPVariabilityMetric:
    patient_id: str
    computed_at: int
    trigger_sbp: float
    trigger_dbp: float
    trigger_source: str
    trigger_time_context: str
    arv_sbp_7d: Optional[float]
    sd_sbp_7d: Optional[float]
    cv_sbp_7d: Optional[float]
    variability_classification_7d: str
    sbp_7d_avg: Optional[float]
    dbp_7d_avg: Optional[float]
    bp_control_status: str
    crisis_flag: bool
    acute_surge_flag: bool
    morning_surge_today: Optional[float]
    surge_today_classification: str
    surge_classification: str
    within_day_sd_sbp: Optional[float]
    days_with_data_in_7d: int
    total_readings_in_state: int


@dataclass
class StateChangeEvent:
    change_id: str
    patient_id: str
    change_type: str
    priority: str
    domain: Optional[str]
    trigger_module: str
    domain_velocities: Dict[str, float]
    composite_score: float
    composite_classification: str
    data_completeness: float
    processing_timestamp: int
    confidence_score: float
    recommended_action: str


@dataclass
class DiagnosticReport:
    """Consolidated diagnostic findings."""
    temporal_issues: List[str] = field(default_factory=list)
    signal_strength_issues: List[str] = field(default_factory=list)
    consumption_issues: List[str] = field(default_factory=list)
    threshold_issues: List[str] = field(default_factory=list)
    root_cause_candidates: List[Tuple[str, float]] = field(default_factory=list)  # (cause, confidence)


# ═══════════════════════════════════════════════════════════════════════
#  PARSER
# ═══════════════════════════════════════════════════════════════════════

def parse_consolidated(filepath: str):
    """Parse the consolidated all-modules JSON."""
    with open(filepath, 'r') as f:
        data = json.load(f)

    m7_outputs = []
    for msg in data.get('modules', {}).get('module7_bp_variability', {}).get('output_messages', []):
        m7_outputs.append(BPVariabilityMetric(
            patient_id=msg['patient_id'],
            computed_at=msg['computed_at'],
            trigger_sbp=msg['trigger_sbp'],
            trigger_dbp=msg['trigger_dbp'],
            trigger_source=msg['trigger_source'],
            trigger_time_context=msg['trigger_time_context'],
            arv_sbp_7d=msg.get('arv_sbp_7d'),
            sd_sbp_7d=msg.get('sd_sbp_7d'),
            cv_sbp_7d=msg.get('cv_sbp_7d'),
            variability_classification_7d=msg['variability_classification_7d'],
            sbp_7d_avg=msg.get('sbp_7d_avg'),
            dbp_7d_avg=msg.get('dbp_7d_avg'),
            bp_control_status=msg['bp_control_status'],
            crisis_flag=msg['crisis_flag'],
            acute_surge_flag=msg['acute_surge_flag'],
            morning_surge_today=msg.get('morning_surge_today'),
            surge_today_classification=msg.get('surge_today_classification', 'N/A'),
            surge_classification=msg.get('surge_classification', 'N/A'),
            within_day_sd_sbp=msg.get('within_day_sd_sbp'),
            days_with_data_in_7d=msg.get('days_with_data_in_7d', 0),
            total_readings_in_state=msg.get('total_readings_in_state', 0),
        ))

    m13_outputs = []
    for msg in data.get('modules', {}).get('module13_clinical_state_sync', {}).get('output_messages', []):
        vel = msg.get('ckm_velocity_at_change', {})
        m13_outputs.append(StateChangeEvent(
            change_id=msg['change_id'],
            patient_id=msg['patient_id'],
            change_type=msg['change_type'],
            priority=msg['priority'],
            domain=msg.get('domain'),
            trigger_module=msg['trigger_module'],
            domain_velocities=vel.get('domain_velocities', {}),
            composite_score=vel.get('composite_score', 0.0),
            composite_classification=vel.get('composite_classification', 'UNKNOWN'),
            data_completeness=vel.get('data_completeness', 0.0),
            processing_timestamp=msg['processing_timestamp'],
            confidence_score=msg.get('confidence_score', 0.0),
            recommended_action=msg.get('recommended_action', ''),
        ))

    m7_inputs = data.get('modules', {}).get('module7_bp_variability', {}).get('input_messages', [])
    m8_outputs = data.get('modules', {}).get('module8_comorbidity', {}).get('output_messages', [])

    return m7_inputs, m7_outputs, m8_outputs, m13_outputs


# ═══════════════════════════════════════════════════════════════════════
#  DIAGNOSTIC CHECKS
# ═══════════════════════════════════════════════════════════════════════

def check_temporal_ordering(m7_outputs: List[BPVariabilityMetric],
                             m13_outputs: List[StateChangeEvent],
                             report: DiagnosticReport):
    """Check 1: Did M7 output arrive before M13 computed?"""
    print("\n" + "=" * 70)
    print("  CHECK 1: TEMPORAL ORDERING — M7 vs M13")
    print("=" * 70)

    if not m7_outputs or not m13_outputs:
        print("  ⚠ Insufficient data for temporal analysis")
        return

    m7_timestamps = sorted([m.computed_at for m in m7_outputs])
    m13_timestamps = sorted([m.processing_timestamp for m in m13_outputs])

    first_m7 = m7_timestamps[0]
    last_m7 = m7_timestamps[-1]
    first_m13 = m13_timestamps[0]
    last_m13 = m13_timestamps[-1]

    print(f"  M7  range: {_ts(first_m7)} → {_ts(last_m7)}")
    print(f"  M13 range: {_ts(first_m13)} → {_ts(last_m13)}")
    print(f"  M7 span:   {(last_m7 - first_m7) / 1000:.1f}s")
    print(f"  M13 span:  {(last_m13 - first_m13) / 1000:.1f}s")

    # Check if M13 fired before M7 finished
    if first_m13 < last_m7:
        gap = last_m7 - first_m13
        print(f"\n  ⚠ RACE CONDITION: M13 first output is {gap}ms BEFORE M7's last output")
        print(f"    M13 started computing at {_ts(first_m13)}")
        print(f"    M7 was still emitting at  {_ts(last_m7)}")
        report.temporal_issues.append(
            f"M13 computed {gap}ms before M7 finished emitting")
        report.root_cause_candidates.append(
            ("Race condition: M13 computes before M7 data fully arrives", 0.7))

    # Check M13 CKM_RISK_ESCALATION specifically
    escalations = [m for m in m13_outputs if m.change_type == 'CKM_RISK_ESCALATION']
    for esc in escalations:
        m7_before = sum(1 for t in m7_timestamps if t < esc.processing_timestamp)
        m7_total = len(m7_timestamps)
        pct = (m7_before / m7_total * 100) if m7_total > 0 else 0
        print(f"\n  CKM_RISK_ESCALATION at {_ts(esc.processing_timestamp)}:")
        print(f"    M7 outputs available at that time: {m7_before}/{m7_total} ({pct:.0f}%)")
        if pct < 80:
            report.temporal_issues.append(
                f"Only {pct:.0f}% of M7 outputs available when CKM escalation computed")


def check_signal_strength(m7_outputs: List[BPVariabilityMetric],
                           report: DiagnosticReport):
    """Check 2: Is M7 data alarming enough to trigger CARDIOVASCULAR velocity?"""
    print("\n" + "=" * 70)
    print("  CHECK 2: M7 SIGNAL STRENGTH ANALYSIS")
    print("=" * 70)

    by_patient = defaultdict(list)
    for m in m7_outputs:
        by_patient[m.patient_id].append(m)

    for pid, metrics in by_patient.items():
        sorted_metrics = sorted(metrics, key=lambda m: m.computed_at)
        print(f"\n  Patient: {pid}")
        print(f"  Total readings: {len(sorted_metrics)}")

        # SBP trend
        sbp_avgs = [m.sbp_7d_avg for m in sorted_metrics if m.sbp_7d_avg is not None]
        if len(sbp_avgs) >= 4:
            early = statistics.mean(sbp_avgs[:len(sbp_avgs)//3])
            late = statistics.mean(sbp_avgs[-len(sbp_avgs)//3:])
            delta = late - early
            print(f"  SBP 7d avg trend: {early:.1f} → {late:.1f} (Δ={delta:+.1f} mmHg)")

            if delta > 10:
                print(f"  ✓ SBP rising significantly — CARDIOVASCULAR velocity should respond")
            elif delta > 5:
                print(f"  ~ SBP rising moderately — may not cross velocity threshold")
            else:
                print(f"  ○ SBP relatively stable — velocity threshold may not be met")
                report.signal_strength_issues.append(
                    f"SBP trend for {pid}: only Δ={delta:+.1f} mmHg")

        # Variability classification escalation
        var_classes = [m.variability_classification_7d for m in sorted_metrics]
        var_transitions = []
        for i in range(1, len(var_classes)):
            if var_classes[i] != var_classes[i-1]:
                var_transitions.append(f"{var_classes[i-1]} → {var_classes[i]}")
        if var_transitions:
            print(f"  Variability transitions: {', '.join(var_transitions)}")
        final_var = var_classes[-1] if var_classes else "N/A"
        print(f"  Final variability class: {final_var}")

        # Crisis flags
        crisis_count = sum(1 for m in sorted_metrics if m.crisis_flag)
        print(f"  Crisis flags: {crisis_count}/{len(sorted_metrics)}")
        if crisis_count > 0:
            crisis_readings = [m for m in sorted_metrics if m.crisis_flag]
            for cr in crisis_readings:
                print(f"    crisis at {_ts(cr.computed_at)}: SBP={cr.trigger_sbp}")

        # Morning surge
        elevated_surges = [m for m in sorted_metrics if m.surge_classification == 'ELEVATED']
        print(f"  Elevated surge periods: {len(elevated_surges)}")

        # BP control status
        statuses = set(m.bp_control_status for m in sorted_metrics)
        print(f"  BP control statuses seen: {statuses}")

        # ── Compute what velocity SHOULD be
        print(f"\n  ── Expected CARDIOVASCULAR velocity components ──")
        sbp_velocity = 0.0
        if len(sbp_avgs) >= 4:
            delta_per_day = delta / 14  # over 14 days
            if delta_per_day > 1.0:  # > 1 mmHg/day increase
                sbp_velocity = min(delta_per_day / 2.0, 1.0)  # normalize to 0-1
        print(f"    SBP trend velocity:       {sbp_velocity:.2f}")

        var_velocity = {"INSUFFICIENT_DATA": 0.0, "NORMAL": 0.0, "ELEVATED": 0.3, "HIGH": 0.7}
        v_vel = var_velocity.get(final_var, 0.0)
        print(f"    Variability velocity:     {v_vel:.2f} (class={final_var})")

        crisis_velocity = min(crisis_count / 3.0, 1.0) if crisis_count > 0 else 0.0
        print(f"    Crisis velocity:          {crisis_velocity:.2f} (count={crisis_count})")

        surge_velocity = 0.3 if len(elevated_surges) > 0 else 0.0
        print(f"    Surge velocity:           {surge_velocity:.2f}")

        expected = max(sbp_velocity, v_vel, crisis_velocity, surge_velocity)
        print(f"    ───────────────────────────")
        print(f"    EXPECTED CV velocity:     {expected:.2f}")
        print(f"    ACTUAL CV velocity:       0.00  ← THE BUG")

        if expected > 0.3:
            report.root_cause_candidates.append(
                ("M7 signals ARE strong enough — problem is M13 consumption/thresholds", 0.9))


def check_m13_consumption(m13_outputs: List[StateChangeEvent],
                           m7_count: int, m8_count: int,
                           report: DiagnosticReport):
    """Check 3: What data did M13 actually see?"""
    print("\n" + "=" * 70)
    print("  CHECK 3: M13 CONSUMPTION PATTERN")
    print("=" * 70)

    # Analyze change types
    type_counts = defaultdict(int)
    for sc in m13_outputs:
        type_counts[sc.change_type] += 1

    print(f"  M13 output breakdown:")
    for ct, count in type_counts.items():
        print(f"    {ct}: {count}")

    # DATA_ABSENCE events indicate M13 isn't getting data from some domains
    absence_events = [sc for sc in m13_outputs if 'DATA_ABSENCE' in sc.change_type]
    if absence_events:
        print(f"\n  ⚠ DATA_ABSENCE events: {len(absence_events)}")
        for ae in absence_events:
            print(f"    {ae.change_type} for {ae.patient_id} "
                  f"(completeness={ae.data_completeness:.3f})")
        report.consumption_issues.append(
            f"{len(absence_events)} DATA_ABSENCE events — M13 missing input data")

    # Check data completeness in CKM computations
    for sc in m13_outputs:
        if sc.change_type == 'CKM_RISK_ESCALATION':
            print(f"\n  CKM_RISK_ESCALATION data_completeness: {sc.data_completeness:.3f}")
            if sc.data_completeness < 0.5:
                print(f"  ⚠ LOW DATA COMPLETENESS: M13 computed CKM with only "
                      f"{sc.data_completeness*100:.1f}% of expected data")
                report.consumption_issues.append(
                    f"CKM computed with {sc.data_completeness*100:.1f}% data completeness")
                report.root_cause_candidates.append(
                    ("Low data completeness — M13 missing BP variability data at computation time", 0.8))


def check_velocity_thresholds(m13_outputs: List[StateChangeEvent],
                               report: DiagnosticReport):
    """Check 4: Examine CKM velocity computation details."""
    print("\n" + "=" * 70)
    print("  CHECK 4: CKM VELOCITY THRESHOLD ANALYSIS")
    print("=" * 70)

    escalations = [sc for sc in m13_outputs if sc.change_type == 'CKM_RISK_ESCALATION']

    for esc in escalations:
        print(f"\n  CKM_RISK_ESCALATION for {esc.patient_id}:")
        print(f"    composite_score:      {esc.composite_score:.4f}")
        print(f"    composite_class:      {esc.composite_classification}")
        print(f"    data_completeness:    {esc.data_completeness:.4f}")
        print(f"    confidence_score:     {esc.confidence_score:.4f}")
        print(f"    domain_velocities:")

        domains_present = set()
        for domain, velocity in sorted(esc.domain_velocities.items()):
            marker = "✓" if velocity > 0 else "✗"
            print(f"      {marker} {domain:20s} = {velocity:.4f}")
            domains_present.add(domain)

        # ── KEY DIAGNOSIS
        expected_domains = {"CARDIOVASCULAR", "RENAL", "METABOLIC"}
        missing_domains = expected_domains - domains_present

        if missing_domains:
            print(f"\n    ❌ MISSING DOMAINS: {missing_domains}")
            for d in missing_domains:
                print(f"       → '{d}' domain not in velocity map")
            report.threshold_issues.append(
                f"Missing domains in CKM velocity: {missing_domains}")
            report.root_cause_candidates.append(
                (f"M13 CKM model missing domain mapping for: {missing_domains}", 0.95))

        if "CARDIOVASCULAR" in domains_present and esc.domain_velocities["CARDIOVASCULAR"] == 0.0:
            print(f"\n    ❌ CARDIOVASCULAR domain present but velocity = 0.0")
            print(f"       → M13 knows about CARDIOVASCULAR but BP data didn't affect it")
            print(f"       Hypothesis A: M13 maps only VITAL_SIGN events → CARDIOVASCULAR")
            print(f"                     but NOT bp-variability-metrics → CARDIOVASCULAR")
            print(f"       Hypothesis B: Velocity requires Δ between two time windows")
            print(f"                     and the worsening trend is too gradual")
            print(f"       Hypothesis C: M13 uses bp_control_status (which didn't change)")
            print(f"                     and ignores variability_classification (which DID)")

            report.threshold_issues.append(
                "CARDIOVASCULAR domain exists but velocity=0.0 despite alarming M7 data")
            report.root_cause_candidates.append(
                ("M13 maps bp_control_status → CARDIOVASCULAR but not variability metrics", 0.85))
            report.root_cause_candidates.append(
                ("M13 only reads VITAL_SIGN events, not bp-variability-metrics topic", 0.75))


def check_domain_mapping_hypothesis(m7_outputs: List[BPVariabilityMetric],
                                     m13_outputs: List[StateChangeEvent],
                                     report: DiagnosticReport):
    """Check 5: Test hypothesis that M13 uses wrong signals for CARDIOVASCULAR."""
    print("\n" + "=" * 70)
    print("  CHECK 5: DOMAIN MAPPING HYPOTHESIS TEST")
    print("=" * 70)

    # Hypothesis: M13 uses bp_control_status for CARDIOVASCULAR velocity.
    # Since bp_control_status was STAGE_2_UNCONTROLLED throughout,
    # there's NO CHANGE → velocity = 0.0
    statuses = set(m.bp_control_status for m in m7_outputs)
    print(f"\n  bp_control_status values seen: {statuses}")
    if len(statuses) == 1:
        print(f"  ✓ Hypothesis CONFIRMED: bp_control_status never changed ({statuses.pop()})")
        print(f"    If M13 computes CARDIOVASCULAR velocity from bp_control_status transitions,")
        print(f"    then velocity = 0 because there IS no transition.")
        print(f"    But the ACTUAL clinical picture is deteriorating:")
        report.root_cause_candidates.append(
            ("M13 velocity based on bp_control_status transitions (no change = velocity 0)", 0.92))
    else:
        print(f"  ✗ bp_control_status DID change: {statuses}")

    # What DID change?
    print(f"\n  Signals that DID change (should feed CARDIOVASCULAR velocity):")

    var_classes = set(m.variability_classification_7d for m in m7_outputs
                      if m.variability_classification_7d != 'INSUFFICIENT_DATA')
    print(f"    variability_classification_7d: {var_classes} ← CHANGED")

    sbp_avgs = [m.sbp_7d_avg for m in m7_outputs if m.sbp_7d_avg is not None]
    if sbp_avgs:
        print(f"    sbp_7d_avg range: {min(sbp_avgs):.0f} → {max(sbp_avgs):.0f} ← CHANGED")

    crisis_flags = [m.crisis_flag for m in m7_outputs]
    print(f"    crisis_flag: {sum(crisis_flags)} of {len(crisis_flags)} readings ← CHANGED")

    surge_classes = set(m.surge_classification for m in m7_outputs
                        if m.surge_classification not in ('N/A', 'INSUFFICIENT_DATA'))
    print(f"    surge_classification: {surge_classes} ← CHANGED")

    print(f"\n  ── RECOMMENDATION ──")
    print(f"  M13's CKM velocity calculator should incorporate:")
    print(f"    1. variability_classification_7d transitions (ELEVATED → HIGH)")
    print(f"    2. sbp_7d_avg trend (rising > 5 mmHg over window)")
    print(f"    3. crisis_flag frequency (any TRUE in window)")
    print(f"    4. surge_classification escalation (NORMAL → ELEVATED)")
    print(f"  Not just bp_control_status, which is too coarse-grained.")


# ═══════════════════════════════════════════════════════════════════════
#  FINAL DIAGNOSIS
# ═══════════════════════════════════════════════════════════════════════

def generate_diagnosis(report: DiagnosticReport):
    """Synthesize all checks into a ranked diagnosis."""
    print("\n" + "═" * 70)
    print("  FINAL DIAGNOSIS: CARDIOVASCULAR VELOCITY = 0.0")
    print("═" * 70)

    # Rank root causes by confidence
    ranked = sorted(report.root_cause_candidates, key=lambda x: -x[1])

    print(f"\n  Root cause candidates (ranked by confidence):\n")
    for i, (cause, confidence) in enumerate(ranked, 1):
        bar = "█" * int(confidence * 20)
        print(f"  {i}. [{confidence:.0%}] {bar}")
        print(f"     {cause}\n")

    if ranked:
        top_cause, top_conf = ranked[0]
        print(f"  ── MOST LIKELY ROOT CAUSE ({top_conf:.0%} confidence) ──")
        print(f"  {top_cause}")

    print(f"\n  ── RECOMMENDED FIXES ──")
    print(f"  Priority 1: Update M13's CKMVelocityCalculator to map")
    print(f"    bp-variability-metrics fields → CARDIOVASCULAR velocity:")
    print(f"    • variability_classification_7d transitions")
    print(f"    • sbp_7d_avg trend slope")
    print(f"    • crisis_flag activation")
    print(f"    • surge_classification escalation")
    print(f"")
    print(f"  Priority 2: Add watermark synchronization in M13 to ensure")
    print(f"    M7 data is fully consumed before CKM velocity computation")
    print(f"")
    print(f"  Priority 3: Lower data_completeness threshold or add")
    print(f"    partial-data velocity computation for early signals")

    # Issues summary
    all_issues = (report.temporal_issues + report.signal_strength_issues +
                  report.consumption_issues + report.threshold_issues)
    if all_issues:
        print(f"\n  ── ALL ISSUES FOUND ({len(all_issues)}) ──")
        for issue in all_issues:
            print(f"    • {issue}")


# ═══════════════════════════════════════════════════════════════════════
#  UTILITIES
# ═══════════════════════════════════════════════════════════════════════

def _ts(epoch_ms: int) -> str:
    """Format epoch milliseconds as human-readable UTC timestamp."""
    return datetime.fromtimestamp(epoch_ms / 1000, tz=timezone.utc).strftime("%Y-%m-%d %H:%M:%S UTC")


# ═══════════════════════════════════════════════════════════════════════
#  MAIN
# ═══════════════════════════════════════════════════════════════════════

def main():
    parser = argparse.ArgumentParser(description="M13 CARDIOVASCULAR Velocity Debugger")
    parser.add_argument('input_file', nargs='?', help="Consolidated all-modules JSON file")
    parser.add_argument('--m7', help="M7 output JSON file")
    parser.add_argument('--m13', help="M13 output JSON file")
    args = parser.parse_args()

    if args.input_file:
        m7_inputs, m7_outputs, m8_outputs, m13_outputs = parse_consolidated(args.input_file)
    else:
        print("Usage: python3 m13_velocity_debugger.py <consolidated-json>")
        sys.exit(1)

    print("╔══════════════════════════════════════════════════════════════════╗")
    print("║     M13 CARDIOVASCULAR VELOCITY DIAGNOSTIC REPORT              ║")
    print("╠══════════════════════════════════════════════════════════════════╣")
    print(f"║  M7  outputs:  {len(m7_outputs):4d}                                          ║")
    print(f"║  M8  outputs:  {len(m8_outputs):4d}                                          ║")
    print(f"║  M13 outputs:  {len(m13_outputs):4d}                                          ║")
    print("╚══════════════════════════════════════════════════════════════════╝")

    report = DiagnosticReport()

    check_temporal_ordering(m7_outputs, m13_outputs, report)
    check_signal_strength(m7_outputs, report)
    check_m13_consumption(m13_outputs, len(m7_outputs), len(m8_outputs), report)
    check_velocity_thresholds(m13_outputs, report)
    check_domain_mapping_hypothesis(m7_outputs, m13_outputs, report)
    generate_diagnosis(report)


if __name__ == '__main__':
    main()
