"""CompatibilityChecker — Layer 3 v2 Wave 1 Task 3.

Tracks rule compatibility status across four event classes:

  Event A — rule update:        re-validate the changed rule against
                                current schemas and gates.
  Event B — helper update:      re-validate every dependent rule.
  Event C — substrate change:   when Layer 2 publishes a substrate
                                schema change manifest, mark every
                                rule whose state_machine_references[]
                                touches the changed fact as STALE.
  Event D — ScopeRule change:   when ScopeRules engine deploys a new
                                rule, mark every Layer 3 rule whose
                                authorisation_gating.scope_rule_refs[]
                                includes the changed rule as STALE.

This is the canonical CompatibilityChecker for Layer 3 v2 (no v1.0
implementation exists on disk per Wave 1 Task 3 backstop).

The checker is in-memory; persistence is V1 work. Wired into the
governance promoter (Wave 1 Task 5) as a precondition gate.
"""

from __future__ import annotations

from dataclasses import dataclass, field
from enum import Enum
from pathlib import Path
from typing import Any, Iterable

import yaml

from rule_specification_validator import (  # type: ignore[import-not-found]
    load_spec,
    validate_rule_specification,
)
from two_gate_validator import (  # type: ignore[import-not-found]
    HELPER_TO_FACTS,
    run_two_gate,
    _extract_define_body,
)


# ---------------------------------------------------------------------------
# Status enum
# ---------------------------------------------------------------------------


class CompatStatus(str, Enum):
    ACTIVE = "ACTIVE"
    STALE = "STALE"
    INVALID = "INVALID"


# ---------------------------------------------------------------------------
# Substrate change manifest shape
# ---------------------------------------------------------------------------


@dataclass
class SubstrateChange:
    """Layer 2 substrate change manifest entry.

    `machine` and `fact_type` correspond to the values in
    state_machine_references; `change_type` is informational
    (one of `breaking`, `additive`, `deprecated`).
    """

    machine: str
    fact_type: str
    change_type: str = "breaking"


@dataclass
class ScopeRuleChange:
    scope_rule_ref: str
    change_type: str = "breaking"


# ---------------------------------------------------------------------------
# Rule registration
# ---------------------------------------------------------------------------


@dataclass
class RegisteredRule:
    rule_id: str
    spec: dict[str, Any]
    cql_body: str
    status: CompatStatus = CompatStatus.ACTIVE
    last_reason: str = ""


@dataclass
class CompatibilityChecker:
    rules: dict[str, RegisteredRule] = field(default_factory=dict)

    # ------------------------------------------------------------------
    # Registration
    # ------------------------------------------------------------------

    def register(self, spec: dict[str, Any], cql_body: str) -> RegisteredRule:
        rule = RegisteredRule(
            rule_id=spec["rule_id"],
            spec=spec,
            cql_body=cql_body,
        )
        self.rules[rule.rule_id] = rule
        return rule

    def register_from_files(
        self, spec_path: Path, library_path: Path
    ) -> RegisteredRule:
        spec = load_spec(spec_path)
        body = _extract_define_body(library_path.read_text(), spec["define"])
        return self.register(spec, body)

    # ------------------------------------------------------------------
    # Event A — rule update
    # ------------------------------------------------------------------

    def OnRuleUpdate(self, spec: dict[str, Any], cql_body: str) -> list[str]:
        """Re-register and re-validate. Returns [rule_id] if rule moved
        out of ACTIVE."""
        rule = self.register(spec, cql_body)
        result = run_two_gate(spec, cql_body)
        if not result.ok:
            rule.status = CompatStatus.INVALID
            rule.last_reason = "; ".join(str(e) for e in result.errors)
            return [rule.rule_id]
        rule.status = CompatStatus.ACTIVE
        rule.last_reason = ""
        return []

    # ------------------------------------------------------------------
    # Event B — helper update
    # ------------------------------------------------------------------

    def OnHelperUpdate(self, helper_name: str) -> list[str]:
        """When a helper is updated, mark every rule that calls that
        helper as STALE so it is re-validated before promotion."""
        affected = []
        for rule in self.rules.values():
            if helper_name in rule.cql_body:
                rule.status = CompatStatus.STALE
                rule.last_reason = (
                    f"helper '{helper_name}' updated; rule must be re-validated"
                )
                affected.append(rule.rule_id)
        return affected

    # ------------------------------------------------------------------
    # Event C — substrate schema change
    # ------------------------------------------------------------------

    def OnSubstrateChange(
        self, manifest: Iterable[SubstrateChange]
    ) -> list[str]:
        """Mark every rule whose reads_from includes a changed
        (machine, fact_type) pair as STALE."""
        changes = list(manifest)
        affected: list[str] = []
        for rule in self.rules.values():
            refs = (rule.spec.get("state_machine_references") or {}).get(
                "reads_from", []
            )
            for ref in refs:
                m = ref.get("machine")
                ft = ref.get("fact_type", "")
                ft_root = ft.split(".", 1)[0]
                for change in changes:
                    if change.machine != m:
                        continue
                    if change.fact_type in (ft, ft_root):
                        rule.status = CompatStatus.STALE
                        rule.last_reason = (
                            f"substrate change ({change.change_type}): "
                            f"({m}, {change.fact_type})"
                        )
                        affected.append(rule.rule_id)
                        break
                else:
                    continue
                break
        return affected

    # ------------------------------------------------------------------
    # Event D — ScopeRule change
    # ------------------------------------------------------------------

    def OnScopeRuleChange(
        self, manifest: Iterable[ScopeRuleChange]
    ) -> list[str]:
        """Event D consumer (Layer 3 v2 doc Part 4.2). When kb-31-scope-rules
        publishes a new ScopeRule version (or retires one), every Layer 3
        rule whose authorisation_gating.scope_rule_refs[] includes the
        changed rule is marked STALE so it can be re-validated against
        the new authorisation surface before it is re-promoted.

        Wave 5 Task 7 acceptance: a synthetic ScopeRule change marks
        >=3 expected defines STALE without false positives. See
        tests/test_compatibility_checker.py for the selectivity test.
        """
        changes = list(manifest)
        affected: list[str] = []
        for rule in self.rules.values():
            refs = (rule.spec.get("authorisation_gating") or {}).get(
                "scope_rule_refs", []
            )
            for change in changes:
                if change.scope_rule_ref in refs:
                    rule.status = CompatStatus.STALE
                    rule.last_reason = (
                        f"scope_rule change ({change.change_type}): "
                        f"{change.scope_rule_ref}"
                    )
                    affected.append(rule.rule_id)
                    break
        return affected

    # ------------------------------------------------------------------
    # Status query
    # ------------------------------------------------------------------

    def status_of(self, rule_id: str) -> CompatStatus:
        rule = self.rules.get(rule_id)
        if rule is None:
            raise KeyError(f"rule '{rule_id}' not registered")
        return rule.status

    def is_compatible(self, rule_id: str) -> bool:
        return self.status_of(rule_id) == CompatStatus.ACTIVE
