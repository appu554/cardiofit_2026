"""CDS Hooks v2.0 emitter — Layer 3 Wave 1 Task 4.

Converts a rule fire result + Layer 2 substrate context into a
CDS Hooks v2.0 response (https://cds-hooks.org/specification/2.0/).

Supports the order-select and order-sign hook types.

Includes a minimal PlanDefinition $apply scaffold that converts a
rule's PlanDefinition + ActivityDefinition into a RequestOrchestration,
which is then funneled into a CDS Hooks Card.

This is a synchronous, in-memory emitter. Real CDS Hooks service
mounting (HTTP, OAuth2 CDS Hooks Service Discovery) is V1 work.
TODO(wave-1-runtime): mount under HAPI CDS Hooks service.
"""

from __future__ import annotations

import json
import uuid
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

# CDS Hooks v2.0 indicator levels (response card.indicator).
CDS_INDICATORS = {"info", "warning", "critical"}

CDS_HOOK_TYPES = {"order-select", "order-sign"}


# ---------------------------------------------------------------------------
# Rule-fire input shape
# ---------------------------------------------------------------------------


@dataclass
class RuleFire:
    rule_id: str
    summary: str
    indicator: str  # one of CDS_INDICATORS
    detail: str = ""
    recommendation_text: str = ""
    recommendation_resource: dict[str, Any] | None = None
    source_url: str = "https://vaidshala.cardiofit/rules/"
    source_label: str = "Vaidshala Clinical Reasoning"
    links: list[dict[str, str]] = field(default_factory=list)


# ---------------------------------------------------------------------------
# PlanDefinition $apply (minimal)
# ---------------------------------------------------------------------------


def apply_plan_definition(
    bundle: dict[str, Any],
    rule_fire: RuleFire,
) -> dict[str, Any]:
    """Reduce a PlanDefinition Bundle (containing PlanDefinition +
    ActivityDefinition + Library) into a RequestOrchestration FHIR
    resource that names the action to be drafted on the resident
    record.

    This is a deliberately small slice of the Clinical Reasoning Module
    `$apply` operation — enough to round-trip the rule fire into CDS
    Hooks card suggestions.
    """
    plan_def = _find_resource(bundle, "PlanDefinition")
    activity_def = _find_resource(bundle, "ActivityDefinition")
    if plan_def is None or activity_def is None:
        raise ValueError(
            "Bundle must include PlanDefinition + ActivityDefinition"
        )

    return {
        "resourceType": "RequestOrchestration",
        "id": f"req-{rule_fire.rule_id.lower().replace('_', '-')}-{uuid.uuid4().hex[:8]}",
        "status": "draft",
        "intent": "proposal",
        "instantiatesCanonical": [plan_def.get("url", "")],
        "action": [
            {
                "title": activity_def.get("code", {}).get("text", rule_fire.recommendation_text),
                "description": rule_fire.detail or rule_fire.summary,
                "resource": {
                    "reference": f"ActivityDefinition/{activity_def.get('id')}"
                },
            }
        ],
    }


# ---------------------------------------------------------------------------
# CDS Hooks v2.0 response
# ---------------------------------------------------------------------------


def emit_cds_hooks_response(
    rule_fire: RuleFire,
    request_orchestration: dict[str, Any] | None = None,
    hook_type: str = "order-select",
) -> dict[str, Any]:
    """Build a CDS Hooks v2.0 response payload (one Card)."""

    if rule_fire.indicator not in CDS_INDICATORS:
        raise ValueError(
            f"indicator '{rule_fire.indicator}' not in {CDS_INDICATORS}"
        )
    if hook_type not in CDS_HOOK_TYPES:
        raise ValueError(f"hook_type '{hook_type}' not in {CDS_HOOK_TYPES}")

    card_uuid = str(uuid.uuid5(uuid.NAMESPACE_URL, f"vaidshala/{rule_fire.rule_id}"))

    suggestions: list[dict[str, Any]] = []
    if request_orchestration is not None:
        suggestions.append(
            {
                "label": rule_fire.recommendation_text or "Apply suggested action",
                "uuid": str(uuid.uuid4()),
                "actions": [
                    {
                        "type": "create",
                        "description": rule_fire.detail or rule_fire.summary,
                        "resource": request_orchestration,
                    }
                ],
            }
        )

    links = list(rule_fire.links)
    # Add a default link to the rule documentation.
    links.append(
        {
            "label": f"Rule: {rule_fire.rule_id}",
            "url": f"{rule_fire.source_url}{rule_fire.rule_id}",
            "type": "absolute",
        }
    )

    card = {
        "uuid": card_uuid,
        "summary": rule_fire.summary,
        "indicator": rule_fire.indicator,
        "detail": rule_fire.detail or rule_fire.summary,
        "source": {
            "label": rule_fire.source_label,
            "url": rule_fire.source_url,
        },
        "suggestions": suggestions,
        "selectionBehavior": "any" if suggestions else "at-most-one",
        "links": links,
    }

    return {"cards": [card]}


# ---------------------------------------------------------------------------
# Schema validation (in-code, since no live CDS Hooks v2.0 validator)
# ---------------------------------------------------------------------------

# Minimum required keys per CDS Hooks v2.0 Card and response shape.
_REQUIRED_RESPONSE_KEYS = ("cards",)
_REQUIRED_CARD_KEYS = ("uuid", "summary", "indicator", "source")
_REQUIRED_SOURCE_KEYS = ("label",)
_REQUIRED_SUGGESTION_KEYS = ("label", "uuid", "actions")
_REQUIRED_LINK_KEYS = ("label", "url", "type")


def validate_cds_hooks_v2_response(
    response: dict[str, Any],
) -> list[str]:
    """Return list of validation errors (empty == valid)."""
    errors: list[str] = []
    for k in _REQUIRED_RESPONSE_KEYS:
        if k not in response:
            errors.append(f"response missing required key '{k}'")
    if errors:
        return errors
    cards = response["cards"]
    if not isinstance(cards, list):
        return ["response.cards must be a list"]
    for ci, card in enumerate(cards):
        for k in _REQUIRED_CARD_KEYS:
            if k not in card:
                errors.append(f"cards[{ci}] missing required key '{k}'")
        if "indicator" in card and card["indicator"] not in CDS_INDICATORS:
            errors.append(
                f"cards[{ci}].indicator '{card['indicator']}' not in {CDS_INDICATORS}"
            )
        for k in _REQUIRED_SOURCE_KEYS:
            if k not in card.get("source", {}):
                errors.append(f"cards[{ci}].source missing required key '{k}'")
        for si, sug in enumerate(card.get("suggestions", [])):
            for k in _REQUIRED_SUGGESTION_KEYS:
                if k not in sug:
                    errors.append(
                        f"cards[{ci}].suggestions[{si}] missing key '{k}'"
                    )
        for li, link in enumerate(card.get("links", [])):
            for k in _REQUIRED_LINK_KEYS:
                if k not in link:
                    errors.append(
                        f"cards[{ci}].links[{li}] missing key '{k}'"
                    )
    return errors


# ---------------------------------------------------------------------------
# helpers
# ---------------------------------------------------------------------------


def _find_resource(bundle: dict[str, Any], resource_type: str) -> dict[str, Any] | None:
    for entry in bundle.get("entry", []):
        res = entry.get("resource") or {}
        if res.get("resourceType") == resource_type:
            return res
    return None


def load_bundle(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text())
