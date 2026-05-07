"""rule_specification.yaml v2 validator.

Validates a YAML rule_specification against
schemas/rule_specification.v2.json and applies four classic-error
checks beyond raw schema validation:

  1. Missing trigger source (every rule needs >=1)         — covered by schema
  2. Dangling fact reference (every state_machine_reference's
     fact_type must be a known fact in the helper-surface spec)
  3. Missing test case class (every rule needs >=3 test cases
     covering positive/negative/edge classes; substrate_state
     class required for substrate-aware rules)
  4. Missing authorisation_gating for Schedule-8 actions
     (BEERS_2023 / BEERS_RENAL / AU_TGA_BLACKBOX rules with
     recommends_action in {cease, taper} that implicate
     a Schedule-8 medication require populated required_roles)

This module is import-safe: errors are returned as ValidationError
lists rather than raised, so callers can aggregate.

Wave 1 Task 2 — see docs/superpowers/plans/2026-05-04-layer3-rule-encoding-plan.md
"""

from __future__ import annotations

import json
import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

import yaml
from jsonschema import Draft202012Validator

# ---------------------------------------------------------------------------
# Module paths
# ---------------------------------------------------------------------------

_HERE = Path(__file__).resolve().parent
SCHEMA_PATH = _HERE / "schemas" / "rule_specification.v2.json"

# ---------------------------------------------------------------------------
# Known fact types — derived from the Wave 0 helper-surface spec. Authors
# adding new state_machine_references must register their fact_type here
# (and a backing helper in the corresponding *.cql file).
# ---------------------------------------------------------------------------

KNOWN_FACT_TYPES: dict[str, set[str]] = {
    "clinical": {
        "medicine_uses",
        "medicine_uses_filtered_by_atc",
        "baseline_state",
        "baseline_state.potassium",
        "baseline_state.eGFR",
        "baseline_state.sodium",
        "baseline_state.bp_systolic",
        "baseline_state.bp_diastolic",
        "baseline_state.CK",
        "baseline_state.BGL",
        "active_concerns",
        "care_intensity",
        "capacity_assessment",
        "latest_observation_potassium",
        "latest_observation_eGFR",
        "latest_observation_sodium",
        "latest_observation_INR",
        "latest_observation_CK",
        "latest_observation_BGL",
        "latest_observation_bp_systolic_standing",
        "latest_observation_bp_systolic_lying",
        "latest_observation_neuro_check",
        "latest_observation_behavioural_chart",
        "latest_observation_oral_intake",
        "latest_observation_fall",
        "latest_observation_contrast_administration",
        "latest_observation_insulin_administration",
        "observations",
    },
    "consent": {
        "active_consents",
        "active_consents_for_class",
        "consent_expiry",
    },
    "authorisation": {
        "authorisation_active",
        "scope_rule",
        "available_prescribers",
        "facility_inventory",
    },
    "monitoring": {
        "monitoring_plan",
        "monitoring_plan_active",
        "observation_overdue",
        "active_plans",
    },
    "evidence_trace": {
        "rule_fire_node",
        "lineage",
        "consequences",
        "rule_fires_history",
        "recommendation_origin",
    },
    "recommendation": {
        "drafted_recommendation",
        "consent_gathering_recommendation",
        "monitoring_proposal",
    },
}

# Schedule-8 implicated criterion ids (regex match on criterion_id) — used
# to enforce authorisation_gating presence for cease/taper actions.
SCHEDULE_8_CRITERION_PATTERNS = [
    re.compile(r"OPIOID", re.IGNORECASE),
    re.compile(r"BENZODIAZEPINE", re.IGNORECASE),
    re.compile(r"PSYCHOSTIMULANT", re.IGNORECASE),
    re.compile(r"S8", re.IGNORECASE),
]

CEASE_TAPER_ACTIONS = {"cease", "taper"}

# ---------------------------------------------------------------------------
# Result types
# ---------------------------------------------------------------------------


@dataclass
class ValidationError:
    code: str
    message: str
    path: str = ""

    def __str__(self) -> str:  # pragma: no cover — formatting
        prefix = f"[{self.code}]"
        if self.path:
            prefix += f" {self.path}:"
        return f"{prefix} {self.message}"


@dataclass
class ValidationResult:
    ok: bool
    errors: list[ValidationError] = field(default_factory=list)

    def __bool__(self) -> bool:  # pragma: no cover — sugar
        return self.ok


# ---------------------------------------------------------------------------
# Loaders
# ---------------------------------------------------------------------------


def load_schema(path: Path = SCHEMA_PATH) -> dict[str, Any]:
    return json.loads(path.read_text())


def load_spec(spec_path: Path) -> dict[str, Any]:
    return yaml.safe_load(spec_path.read_text())


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def validate_rule_specification(
    spec: dict[str, Any],
    *,
    schema: dict[str, Any] | None = None,
) -> ValidationResult:
    """Apply schema validation + the four classic-error checks."""

    schema = schema or load_schema()
    errors: list[ValidationError] = []

    # 1) JSON-Schema pass
    schema_validator = Draft202012Validator(schema)
    for err in sorted(schema_validator.iter_errors(spec), key=lambda e: e.path):
        path = "/".join(str(p) for p in err.absolute_path) or "<root>"
        errors.append(
            ValidationError(
                code="SCHEMA",
                message=err.message,
                path=path,
            )
        )

    # If schema fails grossly, skip semantic checks (they assume shape).
    if errors and any(e.path in ("<root>",) for e in errors):
        return ValidationResult(ok=False, errors=errors)

    # 2) Dangling fact reference
    errors.extend(_check_fact_types(spec))

    # 3) Test-case class coverage
    errors.extend(_check_test_case_classes(spec))

    # 4) Authorisation gating for Schedule-8 cease/taper
    errors.extend(_check_authorisation_for_s8(spec))

    return ValidationResult(ok=not errors, errors=errors)


def validate_file(path: Path) -> ValidationResult:
    return validate_rule_specification(load_spec(path))


# ---------------------------------------------------------------------------
# Classic-error checks
# ---------------------------------------------------------------------------


def _check_fact_types(spec: dict[str, Any]) -> list[ValidationError]:
    errors: list[ValidationError] = []
    refs = spec.get("state_machine_references", {})
    for direction in ("reads_from", "writes_to"):
        for idx, ref in enumerate(refs.get(direction, [])):
            machine = ref.get("machine")
            fact = ref.get("fact_type")
            if machine not in KNOWN_FACT_TYPES:
                errors.append(
                    ValidationError(
                        code="DANGLING_FACT",
                        message=f"unknown machine '{machine}'",
                        path=f"state_machine_references/{direction}/{idx}",
                    )
                )
                continue
            if fact not in KNOWN_FACT_TYPES[machine]:
                errors.append(
                    ValidationError(
                        code="DANGLING_FACT",
                        message=(
                            f"fact_type '{fact}' is not registered for "
                            f"machine '{machine}'. Add it to "
                            "rule_specification_validator.KNOWN_FACT_TYPES "
                            "after registering a backing helper."
                        ),
                        path=f"state_machine_references/{direction}/{idx}",
                    )
                )
    return errors


def _check_test_case_classes(spec: dict[str, Any]) -> list[ValidationError]:
    errors: list[ValidationError] = []
    test_cases = spec.get("test_cases") or []
    if len(test_cases) < 3:
        errors.append(
            ValidationError(
                code="MISSING_TEST_COVERAGE",
                message=(
                    f"rule has {len(test_cases)} test cases; "
                    "minimum 3 required (positive, negative, edge)"
                ),
                path="test_cases",
            )
        )
    classes = {tc.get("class") for tc in test_cases}
    has_positive = any(c in {"positive", "baseline_aware_fire", "consent_gating"} for c in classes)
    has_negative = any(
        c in {"suppression", "care_intensity", "missing_data", "boundary"}
        for c in classes
    )
    if not has_positive:
        errors.append(
            ValidationError(
                code="MISSING_TEST_COVERAGE",
                message="no test case in a positive class (positive / baseline_aware_fire / consent_gating)",
                path="test_cases",
            )
        )
    if not has_negative:
        errors.append(
            ValidationError(
                code="MISSING_TEST_COVERAGE",
                message="no test case in a negative class (suppression / care_intensity / missing_data / boundary)",
                path="test_cases",
            )
        )

    # Substrate-state class required when the rule is substrate-aware
    substrate_aware = _is_substrate_aware(spec)
    if substrate_aware and "substrate_state" not in classes:
        # Allow alternative substrate-aware classes
        substrate_classes = {"substrate_state", "baseline_aware_fire", "evidence_trace"}
        if not (classes & substrate_classes):
            errors.append(
                ValidationError(
                    code="MISSING_TEST_COVERAGE",
                    message=(
                        "substrate-aware rule must include at least one "
                        "test case in {substrate_state, baseline_aware_fire, "
                        "evidence_trace}"
                    ),
                    path="test_cases",
                )
            )
    return errors


def _check_authorisation_for_s8(spec: dict[str, Any]) -> list[ValidationError]:
    errors: list[ValidationError] = []
    criterion_set = spec.get("criterion_set", "")
    if criterion_set not in {"BEERS_2023", "BEERS_RENAL", "AU_TGA_BLACKBOX"}:
        return errors
    # MVP: infer recommended action from `summary` + criterion_id.
    summary = (spec.get("summary") or "").lower()
    criterion_id = spec.get("criterion_id", "")
    is_cease_or_taper = any(act in summary for act in CEASE_TAPER_ACTIONS)
    is_s8 = any(p.search(criterion_id) for p in SCHEDULE_8_CRITERION_PATTERNS)
    if not (is_cease_or_taper and is_s8):
        return errors
    auth = spec.get("authorisation_gating") or {}
    roles = auth.get("required_roles") or []
    if not roles:
        errors.append(
            ValidationError(
                code="MISSING_S8_AUTHORISATION",
                message=(
                    f"Schedule-8 cease/taper rule '{spec.get('rule_id')}' "
                    "(criterion_set in {BEERS_2023, BEERS_RENAL, "
                    "AU_TGA_BLACKBOX}) must populate "
                    "authorisation_gating.required_roles[]"
                ),
                path="authorisation_gating/required_roles",
            )
        )
    return errors


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def _is_substrate_aware(spec: dict[str, Any]) -> bool:
    """A rule is substrate-aware if any read references baseline_state,
    active_concerns, care_intensity, capacity_assessment, or trajectory
    fact types — i.e. anything beyond raw observations."""
    refs = (spec.get("state_machine_references") or {}).get("reads_from", [])
    aware_facts = {
        "baseline_state",
        "active_concerns",
        "care_intensity",
        "capacity_assessment",
    }
    for ref in refs:
        ft = ref.get("fact_type", "")
        for af in aware_facts:
            if ft.startswith(af):
                return True
    return False


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------


def _main(argv: list[str]) -> int:  # pragma: no cover — CLI shim
    if len(argv) < 2:
        print("usage: rule_specification_validator.py <spec.yaml> [<spec.yaml> ...]")
        return 2
    rc = 0
    for arg in argv[1:]:
        result = validate_file(Path(arg))
        if result.ok:
            print(f"{arg}: OK")
        else:
            rc = 1
            print(f"{arg}: FAIL ({len(result.errors)} errors)")
            for err in result.errors:
                print(f"  {err}")
    return rc


if __name__ == "__main__":  # pragma: no cover
    import sys

    raise SystemExit(_main(sys.argv))
