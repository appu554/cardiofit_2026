"""Two-gate validator for Layer 3 v2 rules.

Stage 1 — clinical translation lint (rule_specification YAML wellformedness)
Stage 2 — two-gate semantic check on the CQL define body:

  * Snapshot-semantics gate: substrate-aware rules MUST reference
    substrate primitives (BaselineFor, DeltaFromBaseline, IsTrending,
    VelocityFlag, etc.) and MUST NOT use raw LatestObservationValue
    as the sole basis for a clinical decision (the gate flags rules
    that compare LatestObservationValue directly to a threshold
    without a baseline reference).

  * Substrate-semantics gate: every state_machine_references[] entry
    MUST have a matching helper call in the CQL define body. This
    catches drift between the YAML spec and the CQL implementation.

Wave 1 Task 2 — see plan.

This module operates at the TEXT level over the CQL define body —
no real CQL evaluation is performed (HAPI engine integration is V1
work, marked TODO(wave-1-runtime)).
"""

from __future__ import annotations

import re
from dataclasses import dataclass
from pathlib import Path
from typing import Any

from rule_specification_validator import (  # type: ignore[import-not-found]
    ValidationError,
    ValidationResult,
    load_spec,
    validate_rule_specification,
    _is_substrate_aware,
)

# ---------------------------------------------------------------------------
# Helper-name → state-machine fact mapping
# ---------------------------------------------------------------------------

# Maps the helper name authors call from CQL to the state-machine fact
# the helper backs. Used by the substrate-semantics gate to verify
# state_machine_references[].fact_type ↔ helper-call coherence.
HELPER_TO_FACTS: dict[str, list[tuple[str, str]]] = {
    # MedicationHelpers
    "HasActiveAtcPrefix": [("clinical", "medicine_uses_filtered_by_atc")],
    "IsActiveOnDate": [("clinical", "medicine_uses")],
    "DurationDaysActive": [("clinical", "medicine_uses")],
    "MorphineEquivalentMgPerDay": [("clinical", "medicine_uses")],
    "AnticholinergicBurdenScore": [("clinical", "medicine_uses")],
    "ActiveDoseFor": [("clinical", "medicine_uses")],
    "GetIntentClass": [("clinical", "medicine_uses")],
    "GetTargetType": [("clinical", "medicine_uses")],
    "IsHighRiskFromBeers": [("clinical", "medicine_uses")],
    # ClinicalStateHelpers
    "BaselineFor": [("clinical", "baseline_state")],
    "DeltaFromBaseline": [("clinical", "baseline_state")],
    "IsTrending": [("clinical", "baseline_state")],
    "VelocityFlag": [("clinical", "baseline_state")],
    "ActiveConcernCount": [("clinical", "active_concerns")],
    "HasActiveConcernType": [("clinical", "active_concerns")],
    "CurrentCareIntensity": [("clinical", "care_intensity")],
    "IsPalliative": [("clinical", "care_intensity")],
    "CapacityAssessmentFor": [("clinical", "capacity_assessment")],
    "LatestObservationValue": [("clinical", "observations")],
    "LatestObservationAt": [("clinical", "observations")],
    # ConsentStateHelpers
    "HasActiveConsentForClass": [("consent", "active_consents_for_class")],
    "ConsentExpiringWithin": [("consent", "active_consents_for_class")],
    "NeedsConsentRefresh": [("consent", "active_consents_for_class")],
    "ConsentExpiryDate": [("consent", "consent_expiry")],
    "SdmReferenceFor": [("clinical", "active_concerns")],  # SDM via resident state
    # AuthorisationHelpers
    "RoleCanPrescribeFor": [("authorisation", "authorisation_active")],
    "AuthorisationIsActive": [("authorisation", "authorisation_active")],
    "ScopeRulePermits": [("authorisation", "scope_rule")],
    "AvailablePrescriberForClass": [("authorisation", "available_prescribers")],
    "HasAvailablePrescriberForClass": [("authorisation", "available_prescribers")],
    "FacilityHasReverseAgentFor": [("authorisation", "facility_inventory")],
    # MonitoringHelpers
    "MonitoringPlanIsActive": [("monitoring", "monitoring_plan_active")],
    "ActivePlansFor": [("monitoring", "active_plans")],
    "HasOpenPlanOfType": [("monitoring", "monitoring_plan")],
    "ObservationOverdueBy": [("monitoring", "observation_overdue")],
    "ExpectedObservationCount": [("monitoring", "monitoring_plan")],
    "LastThresholdCrossingAt": [("monitoring", "monitoring_plan")],
    # EvidenceTraceHelpers
    "RuleFiredFor": [("evidence_trace", "rule_fires_history")],
    "LineageOf": [("evidence_trace", "lineage")],
    "ConsequencesOf": [("evidence_trace", "consequences")],
    "RecommendationOriginRule": [("evidence_trace", "recommendation_origin")],
    "ReasoningWindowSummary": [("evidence_trace", "rule_fires_history")],
    # SuppressionHelpers (Wave 3 Task 4)
    "WasActionedRecently": [("recommendation_state", "recently_actioned")],
    "WasDeferredRecently": [("recommendation_state", "recently_deferred")],
    "RecentSimilarActionedCount": [
        ("recommendation_state", "recent_similar_actioned_count"),
    ],
    "RecentlyOpenedSimilarConcern": [
        ("recommendation_state", "recently_opened_similar_concern"),
    ],
    # QualityGapHelpers (Wave 4A Tier 3)
    "ActiveMedicineUseCount": [("clinical", "polypharmacy_count")],
    "AcbScoreFor": [("clinical", "acb_score")],
    "HasRecentEventOfType": [("clinical", "events")],
    "EventCountSince": [("clinical", "events")],
    "HasCompletedReconciliationFor": [
        ("clinical", "discharge_reconciliation_status"),
    ],
    "LatestHospitalDischargeOlderThan": [("clinical", "events")],
    "LatestHospitalDischargeId": [("clinical", "events")],
    "AkpsDeltaSince": [("clinical", "baseline_state.akps")],
}

SUBSTRATE_PRIMITIVES = {
    "BaselineFor",
    "DeltaFromBaseline",
    "IsTrending",
    "VelocityFlag",
    "ActiveConcernCount",
    "HasActiveConcernType",
    "CurrentCareIntensity",
    "IsPalliative",
    "MonitoringPlanIsActive",
    "HasActiveConsentForClass",
    "RuleFiredFor",
}

# Pattern that catches snapshot-style threshold comparisons:
# LatestObservationValue(..., 'kind') > 5.5 (or other comparators).
_SNAPSHOT_DIRECT_COMPARE = re.compile(
    r'LatestObservationValue\s*\([^)]*\)\s*[<>]=?\s*[-0-9.]+',
    re.IGNORECASE,
)

_HELPER_CALL = re.compile(
    r'\b(?:[A-Za-z_][A-Za-z_0-9]*\.)?"?([A-Z][A-Za-z0-9_]+)"?\s*\(',
)


# ---------------------------------------------------------------------------
# Result types
# ---------------------------------------------------------------------------


@dataclass
class TwoGateResult:
    ok: bool
    stage1: ValidationResult
    snapshot_gate: ValidationResult
    substrate_gate: ValidationResult

    @property
    def errors(self) -> list[ValidationError]:
        return (
            list(self.stage1.errors)
            + list(self.snapshot_gate.errors)
            + list(self.substrate_gate.errors)
        )


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def run_two_gate(
    spec: dict[str, Any],
    cql_define_body: str,
) -> TwoGateResult:
    """Run Stage 1 (rule_specification validator) + Stage 2 (snapshot
    + substrate gates) over a single rule."""

    stage1 = validate_rule_specification(spec)
    snapshot = _snapshot_semantics_gate(spec, cql_define_body)
    substrate = _substrate_semantics_gate(spec, cql_define_body)
    ok = stage1.ok and snapshot.ok and substrate.ok
    return TwoGateResult(
        ok=ok,
        stage1=stage1,
        snapshot_gate=snapshot,
        substrate_gate=substrate,
    )


def run_two_gate_for_files(
    spec_path: Path,
    cql_library_path: Path,
) -> TwoGateResult:
    spec = load_spec(spec_path)
    body = _extract_define_body(cql_library_path.read_text(), spec.get("define", ""))
    return run_two_gate(spec, body)


# ---------------------------------------------------------------------------
# Gates
# ---------------------------------------------------------------------------


def _snapshot_semantics_gate(
    spec: dict[str, Any],
    cql_body: str,
) -> ValidationResult:
    errors: list[ValidationError] = []
    if not _is_substrate_aware(spec):
        return ValidationResult(ok=True)

    # Substrate-aware rules must call >=1 substrate primitive in body.
    helpers_called = _extract_helper_names(cql_body)
    if not (helpers_called & SUBSTRATE_PRIMITIVES):
        errors.append(
            ValidationError(
                code="SNAPSHOT_GATE",
                message=(
                    "substrate-aware rule must call at least one substrate "
                    f"primitive ({sorted(SUBSTRATE_PRIMITIVES)}); "
                    f"found helpers: {sorted(helpers_called) or '<none>'}"
                ),
                path="cql_define_body",
            )
        )

    # Forbid raw LatestObservationValue() <op> <number> patterns —
    # this is the hallmark snapshot-style decision the gate exists
    # to reject (see Layer 3 v2 doc Part 0.5.4).
    if _SNAPSHOT_DIRECT_COMPARE.search(cql_body):
        errors.append(
            ValidationError(
                code="SNAPSHOT_GATE",
                message=(
                    "rule compares LatestObservationValue() directly to a "
                    "numeric threshold — this is snapshot-style decisioning "
                    "and produces alert noise. Use BaselineFor / "
                    "DeltaFromBaseline / VelocityFlag instead."
                ),
                path="cql_define_body",
            )
        )

    return ValidationResult(ok=not errors, errors=errors)


def _substrate_semantics_gate(
    spec: dict[str, Any],
    cql_body: str,
) -> ValidationResult:
    errors: list[ValidationError] = []
    refs = (spec.get("state_machine_references") or {}).get("reads_from", [])
    helpers_called = _extract_helper_names(cql_body)

    # Build set of (machine, fact) actually backed by helpers in body.
    backed: set[tuple[str, str]] = set()
    for h in helpers_called:
        for mf in HELPER_TO_FACTS.get(h, []):
            backed.add(mf)

    for idx, ref in enumerate(refs):
        machine = ref.get("machine")
        fact = ref.get("fact_type", "")
        # fact may have a dotted suffix (e.g. baseline_state.potassium);
        # match on the prefix before '.'
        fact_root = fact.split(".", 1)[0]
        # latest_observation_<kind> facts are backed by the
        # `observations` helper family.
        if fact.startswith("latest_observation_"):
            fact_root = "observations"
        if (machine, fact) in backed or (machine, fact_root) in backed:
            continue
        errors.append(
            ValidationError(
                code="SUBSTRATE_GATE",
                message=(
                    f"state_machine_references[{idx}] declares read of "
                    f"({machine}, {fact}) but no matching helper call "
                    f"appears in the CQL define body. "
                    f"Helpers found: {sorted(helpers_called) or '<none>'}"
                ),
                path=f"state_machine_references/reads_from/{idx}",
            )
        )

    return ValidationResult(ok=not errors, errors=errors)


# ---------------------------------------------------------------------------
# CQL parsing helpers (text-level, intentional — no real CQL eval)
# ---------------------------------------------------------------------------


def _extract_helper_names(cql_body: str) -> set[str]:
    """Return helper function names called in the body. Strips quotes
    and library namespace prefixes."""
    names: set[str] = set()
    for m in _HELPER_CALL.finditer(cql_body):
        name = m.group(1)
        # filter out CQL keywords / type constructors that match the
        # capital-letter pattern.
        if name in {
            "Count", "Exists", "First", "Last", "Sum", "Coalesce",
            "Now", "Today", "DateTime", "Interval", "Tuple", "Case",
            "Patient", "List", "String", "Integer", "Boolean", "Decimal",
            "Quantity", "Reference", "MedicationRequest",
        }:
            continue
        names.add(name)
    return names


def _extract_define_body(cql_text: str, define_name: str) -> str:
    """Extract the body of `define "<name>": ...` up to the next
    top-level `define` or end of file. Quote-tolerant."""
    if not define_name:
        return cql_text
    pattern = re.compile(
        r'define\s+"' + re.escape(define_name) + r'"\s*:(.*?)(?=\n\s*define\s+|\Z)',
        re.DOTALL,
    )
    m = pattern.search(cql_text)
    if not m:
        return ""
    return m.group(1).strip()
