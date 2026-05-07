"""Wave 1 Task 2 — pytest coverage for the two-gate validator.

Positive: the three anchor rules (PPI, hyperkalemia, antipsychotic
consent) each pass both gates against their TierOne/TierTwo CQL
library define bodies.

Negative:
  - Snapshot-style rule (raw LatestObservationValue() > 5.5) rejected
    by snapshot gate.
  - Spec declares a state_machine_reference that the CQL body does not
    back via any helper call → substrate gate fails.
"""

from __future__ import annotations

from pathlib import Path

import yaml
import pytest

from two_gate_validator import run_two_gate, run_two_gate_for_files

EXAMPLES = (
    Path(__file__).resolve().parents[2]
    / "cql-libraries"
    / "examples"
)
RULES = (
    Path(__file__).resolve().parents[2]
    / "cql-libraries"
    / "rules"
)


# ---------------------------------------------------------------------------
# Positive — three anchor rules round-trip
# ---------------------------------------------------------------------------


@pytest.mark.parametrize(
    "spec_filename, library_filename",
    [
        ("ppi-deprescribe.yaml", "TierTwoDeprescribing.cql"),
        ("hyperkalemia-trajectory.yaml", "TierOneImmediateSafety.cql"),
        ("antipsychotic-consent-gating.yaml", "TierOneImmediateSafety.cql"),
    ],
)
def test_anchor_rule_passes_both_gates(spec_filename, library_filename):
    result = run_two_gate_for_files(
        EXAMPLES / spec_filename,
        RULES / library_filename,
    )
    assert result.ok, [str(e) for e in result.errors]
    assert result.snapshot_gate.ok
    assert result.substrate_gate.ok


# ---------------------------------------------------------------------------
# Negative — snapshot-style decisioning rejected
# ---------------------------------------------------------------------------


def test_snapshot_style_rule_rejected_by_snapshot_gate():
    spec = yaml.safe_load(
        (EXAMPLES / "hyperkalemia-trajectory.yaml").read_text()
    )
    snapshot_body = (
        "LatestObservationValue(Patient.id, 'potassium') > 5.5\n"
        "and HasActiveAtcPrefix(Patient.id, 'C09')"
    )
    result = run_two_gate(spec, snapshot_body)
    assert not result.ok
    assert any(
        e.code == "SNAPSHOT_GATE" for e in result.snapshot_gate.errors
    ), [str(e) for e in result.snapshot_gate.errors]


def test_substrate_aware_without_primitive_call_rejected():
    spec = yaml.safe_load(
        (EXAMPLES / "hyperkalemia-trajectory.yaml").read_text()
    )
    # Body has helper calls but no substrate primitive — should fail.
    weak_body = "HasActiveAtcPrefix(Patient.id, 'C09')"
    result = run_two_gate(spec, weak_body)
    assert not result.ok
    assert any(
        "must call at least one substrate primitive" in e.message
        for e in result.snapshot_gate.errors
    )


# ---------------------------------------------------------------------------
# Negative — substrate-semantics gate (state-machine ref drift)
# ---------------------------------------------------------------------------


def test_substrate_gate_flags_unbacked_reference():
    spec = yaml.safe_load(
        (EXAMPLES / "ppi-deprescribe.yaml").read_text()
    )
    # CQL body references one helper but spec declares 3 reads — at
    # least 2 must be unbacked.
    sparse_body = "HasActiveAtcPrefix(Patient.id, 'A02BC')"
    result = run_two_gate(spec, sparse_body)
    assert not result.ok
    assert any(
        e.code == "SUBSTRATE_GATE" for e in result.substrate_gate.errors
    )
