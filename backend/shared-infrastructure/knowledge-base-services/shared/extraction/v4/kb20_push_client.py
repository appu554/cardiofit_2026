"""
KB-20 Push Client — D01 Gap Fix.

Pushes extracted L3 context modifier facts and ADR profiles from the
guideline extraction pipeline directly to KB-20's batch write endpoints.

This closes the extraction → storage loop:
  Pipeline (L3 output) → KB20PushClient → KB-20 /api/v1/pipeline/modifiers
                                        → KB-20 /api/v1/pipeline/adr-profiles

Three-source architecture:
  1. PIPELINE  — guideline extraction (KDIGO, ADA, RSSDI profiles)
  2. SPL       — SPLGuard FDA structured labeling ETL
  3. MANUAL_CURATED — clinician-authored overrides (highest priority)
"""

from __future__ import annotations

import logging
import os
from dataclasses import dataclass, field
from datetime import date
from typing import Optional

import requests

from .guideline_profile import GuidelineProfile

logger = logging.getLogger(__name__)

KB20_DEFAULT_URL = os.getenv("KB20_URL", "http://localhost:8131")


@dataclass
class PushResult:
    """Outcome of a push operation to KB-20."""

    modifiers_succeeded: int = 0
    modifiers_failed: int = 0
    adr_succeeded: int = 0
    adr_failed: int = 0
    errors: list[str] = field(default_factory=list)

    @property
    def total_succeeded(self) -> int:
        return self.modifiers_succeeded + self.adr_succeeded

    @property
    def total_failed(self) -> int:
        return self.modifiers_failed + self.adr_failed


class KB20PushClient:
    """
    HTTP client that pushes extracted facts to KB-20 batch endpoints.

    Usage:
        client = KB20PushClient()
        result = client.push_extraction(extraction_result, profile)
    """

    def __init__(
        self,
        base_url: str = KB20_DEFAULT_URL,
        timeout: int = 30,
        api_key: Optional[str] = None,
    ) -> None:
        self.base_url = base_url.rstrip("/")
        self.timeout = timeout
        self.session = requests.Session()
        if api_key:
            self.session.headers["Authorization"] = f"Bearer {api_key}"
        self.session.headers["Content-Type"] = "application/json"

    def push_extraction(
        self,
        extraction_result: dict,
        profile: GuidelineProfile,
        source: str = "PIPELINE",
    ) -> PushResult:
        """
        Push a complete KB20ExtractionResult to KB-20 batch endpoints.

        Args:
            extraction_result: Dict from KB20ExtractionResult.model_dump()
            profile: GuidelineProfile for governance metadata
            source: Source tag (PIPELINE, SPL, MANUAL_CURATED)

        Returns:
            PushResult with success/failure counts
        """
        result = PushResult()

        # Build governance context from profile
        governance = profile.guideline_context()

        # 1. Push ADR profiles
        adr_profiles = extraction_result.get("adr_profiles", [])
        if adr_profiles:
            kb20_adrs = self._transform_adr_profiles(
                adr_profiles, governance, source
            )
            adr_result = self._post_batch(
                "/api/v1/pipeline/adr-profiles", kb20_adrs
            )
            result.adr_succeeded = adr_result.get("succeeded", 0)
            result.adr_failed = adr_result.get("failed", 0)
            if adr_result.get("errors"):
                result.errors.extend(adr_result["errors"])

        # 2. Push standalone modifiers
        standalone = extraction_result.get("standalone_modifiers", [])
        # Also collect modifiers embedded within ADR profiles
        embedded_modifiers = []
        for adr in adr_profiles:
            for cm in adr.get("contextual_modifiers", []):
                cm["_drug_class"] = adr.get("drug_class", "")
                cm["_target_node"] = _infer_target_node(adr)
                embedded_modifiers.append(cm)

        all_modifiers = standalone + embedded_modifiers
        if all_modifiers:
            kb20_mods = self._transform_modifiers(
                all_modifiers, governance, source
            )
            mod_result = self._post_batch(
                "/api/v1/pipeline/modifiers", kb20_mods
            )
            result.modifiers_succeeded = mod_result.get("succeeded", 0)
            result.modifiers_failed = mod_result.get("failed", 0)
            if mod_result.get("errors"):
                result.errors.extend(mod_result["errors"])

        logger.info(
            "KB-20 push complete: %d succeeded, %d failed",
            result.total_succeeded,
            result.total_failed,
        )
        return result

    def _transform_adr_profiles(
        self,
        adr_profiles: list[dict],
        governance: dict,
        source: str,
    ) -> list[dict]:
        """
        Transform L3 ADR profiles to KB-20 Go model format.

        Maps the four-element chain:
          Drug → Symptom (reaction mapped to HPI differential)
          Mechanism → pharmacological pathway
          Onset Window → temporal onset category
          CM Rule → context_modifier_rule JSON
        """
        kb20_records = []
        for adr in adr_profiles:
            # Build context_modifier_rule JSON from embedded modifiers
            cm_rule = self._build_cm_rule(adr.get("contextual_modifiers", []))

            record = {
                "rxnorm_code": adr.get("rxnorm_code", ""),
                "drug_name": adr.get("drug_name", ""),
                "drug_class": adr.get("drug_class", ""),
                "reaction": adr.get("reaction", ""),
                "reaction_snomed": adr.get("reaction_snomed", ""),
                "mechanism": adr.get("mechanism", ""),
                "symptom": adr.get("symptom", ""),
                "onset_window": adr.get("onset_window", ""),
                "onset_category": adr.get("onset_category", ""),
                "frequency": adr.get("frequency", ""),
                "severity": adr.get("severity", ""),
                "risk_factors": adr.get("risk_factors", []),
                "context_modifier_rule": cm_rule,
                "source": source,
                "confidence": adr.get("confidence", 0.50),
                "source_snippet": adr.get("source_snippet", ""),
                "source_authority": governance.get("authority", ""),
                "source_document": governance.get("document", ""),
                "source_section": adr.get("governance", {}).get(
                    "source_section", ""
                ),
                "evidence_level": adr.get("governance", {}).get(
                    "evidence_level", ""
                ),
            }
            kb20_records.append(record)

        return kb20_records

    def _transform_modifiers(
        self,
        modifiers: list[dict],
        governance: dict,
        source: str,
    ) -> list[dict]:
        """Transform L3 modifiers to KB-20 ContextModifier Go model format."""
        kb20_records = []
        for mod in modifiers:
            record = {
                "modifier_type": mod.get("modifier_type", ""),
                "modifier_value": mod.get("modifier_value", ""),
                "target_node_id": mod.get("_target_node", ""),
                "drug_class_trigger": mod.get("_drug_class", ""),
                "effect": _map_effect(mod.get("effect", "")),
                "target_differential": mod.get("target_differential", ""),
                "magnitude": mod.get("effect_magnitude_numeric", 0.0),
                "lab_parameter": mod.get("lab_parameter", ""),
                "lab_operator": mod.get("lab_operator", ""),
                "lab_threshold": mod.get("lab_threshold", 0.0),
                "lab_unit": mod.get("lab_unit", ""),
                "context_modifier_rule": mod.get("context_modifier_rule", ""),
                "source": source,
                "confidence": mod.get("confidence", 0.50),
            }
            kb20_records.append(record)

        return kb20_records

    def _build_cm_rule(self, modifiers: list[dict]) -> str:
        """
        Build context_modifier_rule JSON string from modifier list.

        The CM rule encodes conditions that alter a drug reaction's
        clinical significance. KB-20 stores this as JSONB for flexible
        querying by the HPI engine.
        """
        import json

        if not modifiers:
            return "{}"

        rules = []
        for mod in modifiers:
            rule = {
                "type": mod.get("modifier_type", ""),
                "value": mod.get("modifier_value", ""),
                "effect": mod.get("effect", ""),
            }
            if mod.get("lab_parameter"):
                rule["lab"] = {
                    "parameter": mod["lab_parameter"],
                    "operator": mod.get("lab_operator", ""),
                    "threshold": mod.get("lab_threshold"),
                    "unit": mod.get("lab_unit", ""),
                }
            if mod.get("context_modifier_rule"):
                rule["cm_id"] = mod["context_modifier_rule"]
            rules.append(rule)

        return json.dumps({"conditions": rules}, separators=(",", ":"))

    def _post_batch(self, endpoint: str, records: list[dict]) -> dict:
        """POST a batch of records to KB-20 and return the result."""
        url = f"{self.base_url}{endpoint}"
        try:
            resp = self.session.post(url, json=records, timeout=self.timeout)
            resp.raise_for_status()
            return resp.json().get("data", {})
        except requests.ConnectionError:
            logger.warning("KB-20 not reachable at %s — skipping push", url)
            return {"succeeded": 0, "failed": len(records), "errors": ["connection_refused"]}
        except requests.HTTPError as e:
            logger.error("KB-20 batch write failed: %s", e)
            return {"succeeded": 0, "failed": len(records), "errors": [str(e)]}
        except Exception as e:
            logger.error("Unexpected error pushing to KB-20: %s", e)
            return {"succeeded": 0, "failed": len(records), "errors": [str(e)]}

    def health_check(self) -> bool:
        """Check if KB-20 is reachable."""
        try:
            resp = self.session.get(
                f"{self.base_url}/health", timeout=5
            )
            return resp.status_code == 200
        except Exception:
            return False


class FourElementChainAssembler:
    """
    Assembles the four-element chain from disparate extraction sources.

    Four-Element Chain:
      1. Drug → Drug class/name (from Channel B drug dict)
      2. Symptom → Mapped to HPI differential (from Channel E/F NER)
      3. Mechanism → Pharmacological pathway (from SPL or Channel C)
      4. Onset Window → Temporal onset (from SPL, PK-derived D06, or guideline)
      5. CM Rule → Context modifier conditions (from guideline + pipeline)

    Three sources feed this chain:
      - SPLGuard: Drug→Mechanism, Drug→Onset (FDA structured labeling)
      - Pipeline: Drug→Symptom, Drug→CM Rule (guideline extraction)
      - Manual: Any element (clinician override, highest priority)
    """

    def __init__(self) -> None:
        self._chains: dict[str, dict] = {}

    def add_spl_element(
        self,
        drug_class: str,
        reaction: str,
        mechanism: str = "",
        onset_window: str = "",
        onset_category: str = "",
    ) -> None:
        """Add SPL-sourced elements (mechanism, onset) to the chain."""
        key = f"{drug_class}:{reaction}"
        chain = self._chains.setdefault(key, self._empty_chain(drug_class, reaction))
        if mechanism:
            chain["mechanism"] = mechanism
            chain["mechanism_source"] = "SPL"
        if onset_window:
            chain["onset_window"] = onset_window
            chain["onset_category"] = onset_category
            chain["onset_source"] = "SPL"

    def add_pipeline_element(
        self,
        drug_class: str,
        reaction: str,
        symptom: str = "",
        cm_rule: str = "",
        confidence: float = 0.50,
    ) -> None:
        """Add pipeline-sourced elements (symptom mapping, CM rule)."""
        key = f"{drug_class}:{reaction}"
        chain = self._chains.setdefault(key, self._empty_chain(drug_class, reaction))
        if symptom:
            chain["symptom"] = symptom
            chain["symptom_source"] = "PIPELINE"
        if cm_rule:
            chain["cm_rule"] = cm_rule
            chain["cm_rule_source"] = "PIPELINE"
        chain["confidence"] = max(chain.get("confidence", 0.0), confidence)

    def add_manual_element(
        self,
        drug_class: str,
        reaction: str,
        **kwargs,
    ) -> None:
        """Add manual-curated elements (highest priority, overwrites all)."""
        key = f"{drug_class}:{reaction}"
        chain = self._chains.setdefault(key, self._empty_chain(drug_class, reaction))
        for field_name in ("mechanism", "onset_window", "onset_category", "symptom", "cm_rule"):
            if field_name in kwargs and kwargs[field_name]:
                chain[field_name] = kwargs[field_name]
                chain[f"{field_name}_source"] = "MANUAL_CURATED"

    def get_chains(self) -> list[dict]:
        """Return all assembled four-element chains with completeness grades."""
        results = []
        for chain in self._chains.values():
            chain["completeness_grade"] = self._grade_chain(chain)
            results.append(chain)
        return results

    def get_chain(self, drug_class: str, reaction: str) -> Optional[dict]:
        """Get a specific chain by drug_class and reaction."""
        key = f"{drug_class}:{reaction}"
        chain = self._chains.get(key)
        if chain:
            chain["completeness_grade"] = self._grade_chain(chain)
        return chain

    @staticmethod
    def _empty_chain(drug_class: str, reaction: str) -> dict:
        return {
            "drug_class": drug_class,
            "reaction": reaction,
            "symptom": "",
            "mechanism": "",
            "onset_window": "",
            "onset_category": "",
            "cm_rule": "",
            "confidence": 0.0,
            "symptom_source": "",
            "mechanism_source": "",
            "onset_source": "",
            "cm_rule_source": "",
        }

    @staticmethod
    def _grade_chain(chain: dict) -> str:
        """
        Grade the four-element chain completeness.

        FULL: All 4 elements present (drug+symptom+mechanism+onset+cm_rule)
        PARTIAL: Drug + at least 2 of (symptom, mechanism, onset)
        STUB: Drug + reaction only
        """
        has_symptom = bool(chain.get("symptom"))
        has_mechanism = bool(chain.get("mechanism"))
        has_onset = bool(chain.get("onset_window"))
        has_cm_rule = bool(chain.get("cm_rule"))

        element_count = sum([has_symptom, has_mechanism, has_onset, has_cm_rule])

        if element_count >= 4:
            return "FULL"
        if element_count >= 2:
            return "PARTIAL"
        return "STUB"


def _infer_target_node(adr: dict) -> str:
    """
    Infer the HPI target node from an ADR's symptom field.

    Maps common ADR symptoms to HPI node IDs for the modifier registry.
    """
    symptom = (adr.get("symptom") or "").lower()
    mapping = {
        "dizziness": "P00",
        "chest pain": "P01",
        "dyspnea": "P02",
        "breathlessness": "P02",
        "cough": "P03",
        "palpitation": "P04",
        "irregular": "P04",
        "syncope": "P11",
        "oedema": "P12",
        "edema": "P12",
        "fatigue": "P13",
        "nocturia": "P14",
        "weight gain": "P15",
        "polyuria": "P16",
        "muscle": "P21",
        "cramp": "P21",
    }
    for keyword, node in mapping.items():
        if keyword in symptom:
            return node
    return ""


def _map_effect(effect_str: str) -> str:
    """Map free-text effect description to KB-20 enum."""
    lower = effect_str.lower()
    if any(w in lower for w in ("increase", "higher", "elevat", "worsen")):
        return "INCREASE_PRIOR"
    if any(w in lower for w in ("decrease", "lower", "reduc", "protect")):
        return "DECREASE_PRIOR"
    return "INCREASE_PRIOR"  # conservative default
