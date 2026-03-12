#!/usr/bin/env python3
"""
Vaidshala Phase 5: Knowledge Base Router

Routes atomised recommendations to appropriate Knowledge Base services.
Each KB has specific schemas and validation requirements.

KB Architecture:
- KB-3: Guidelines Brain (temporal constraints, protocol sequences)
- KB-15: Evidence Engine (COR/LOE, guideline source, citations)
- KB-1: Drug Rules (medication-specific logic)
- KB-4: Patient Safety (contraindications, alerts)

Usage:
    from atomiser import KBRouter

    router = KBRouter()
    routed = router.route(atomised_recommendation)

    # Returns: {"kb_target": "KB-15", "payload": {...}, "validation": {...}}
"""

import json
import re
from dataclasses import dataclass, field
from datetime import datetime
from enum import Enum
from typing import Dict, List, Optional, Any
from pathlib import Path

# Import sibling modules
try:
    from .constrained_atomiser import AtomisedRecommendation
except ImportError:
    # For standalone testing
    AtomisedRecommendation = None


class KnowledgeBase(Enum):
    """Target Knowledge Base services."""
    KB_1_DRUG_RULES = "KB-1"      # Drug calculations, dosing rules
    KB_3_GUIDELINES = "KB-3"      # Temporal constraints, protocols
    KB_4_SAFETY = "KB-4"          # Patient safety, contraindications
    KB_15_EVIDENCE = "KB-15"      # Evidence envelopes, COR/LOE


@dataclass
class RoutingResult:
    """Result of KB routing decision."""
    kb_target: KnowledgeBase
    payload: Dict[str, Any]
    validation_result: Dict[str, Any]
    routing_confidence: float
    routing_rationale: str
    secondary_targets: List[KnowledgeBase] = field(default_factory=list)
    warnings: List[str] = field(default_factory=list)


class KBRouter:
    """
    Routes atomised recommendations to appropriate Knowledge Bases.

    Routing Rules:
    1. COR/LOE evidence → KB-15 (primary), KB-3 (if temporal)
    2. Drug dosing/interactions → KB-1 (primary), KB-4 (if safety)
    3. Temporal sequences → KB-3 (primary)
    4. Safety alerts/contraindications → KB-4 (primary)

    CONSTRAINT: All LLM-atomised content routes with DRAFT status.
    """

    # Classification patterns for routing decisions
    DRUG_PATTERNS = [
        r'\b(?:dose|dosing|mg|mcg|ml|units?)\b',
        r'\b(?:medication|drug|pharmaceutical)\b',
        r'\b(?:titrat|escalat|adjust)\b',
        r'\b(?:administer|infuse|inject)\b',
        r'\b(?:ARNi|ACEi|ARB|SGLT2i|beta.?blocker|MRA|statin|opioid)\b',
    ]

    TEMPORAL_PATTERNS = [
        r'\bwithin\s+\d+\s*(?:hour|minute|day|week)\b',
        r'\bevery\s+\d+\s*(?:hour|minute|day|week)\b',
        r'\b(?:before|after|prior\s+to)\b',
        r'\b(?:immediately|stat|urgent|emergent)\b',
        r'\b(?:sequence|order|first|then|next)\b',
    ]

    SAFETY_PATTERNS = [
        r'\b(?:contraindicate?d?|avoid|do\s+not|never)\b',
        r'\b(?:warning|caution|alert|risk)\b',
        r'\b(?:harm|adverse|side\s+effect|toxicity)\b',
        r'\b(?:monitor|watch|check|assess)\b',
        r'\bClass\s*III\b',  # Class III recommendations (harm/no benefit)
    ]

    EVIDENCE_PATTERNS = [
        r'\bClass\s*(?:I|II[ab]?|III)\b',
        r'\bLOE\s*(?:A|B|C)(?:-[RNE][RO]?)?\b',
        r'\b(?:strong|weak|conditional)\s+recommendation\b',
        r'\b(?:high|moderate|low|very\s+low)[\-\s]*(?:quality|certainty)\b',
        r'\b(?:evidence|guideline|recommendation)\b',
    ]

    def __init__(self, kb_endpoints: Dict[str, str] = None):
        """
        Initialize router with optional KB endpoints.

        Args:
            kb_endpoints: Map of KB name to endpoint URL (for future integration)
        """
        self.kb_endpoints = kb_endpoints or {
            "KB-1": "http://localhost:8081",
            "KB-3": "http://localhost:8087",
            "KB-4": "http://localhost:8088",
            "KB-15": "http://localhost:8094",
        }

        # Compile patterns for efficiency
        self._drug_regex = re.compile('|'.join(self.DRUG_PATTERNS), re.IGNORECASE)
        self._temporal_regex = re.compile('|'.join(self.TEMPORAL_PATTERNS), re.IGNORECASE)
        self._safety_regex = re.compile('|'.join(self.SAFETY_PATTERNS), re.IGNORECASE)
        self._evidence_regex = re.compile('|'.join(self.EVIDENCE_PATTERNS), re.IGNORECASE)

    def route(self, recommendation: 'AtomisedRecommendation') -> RoutingResult:
        """
        Route an atomised recommendation to the appropriate KB.

        Args:
            recommendation: AtomisedRecommendation from ConstrainedAtomiser

        Returns:
            RoutingResult with target KB, payload, and validation
        """
        # Analyze content for routing signals
        text = recommendation.original_text
        extraction = recommendation.extraction

        scores = {
            KnowledgeBase.KB_1_DRUG_RULES: self._score_drug(text, extraction),
            KnowledgeBase.KB_3_GUIDELINES: self._score_temporal(text, extraction),
            KnowledgeBase.KB_4_SAFETY: self._score_safety(text, extraction),
            KnowledgeBase.KB_15_EVIDENCE: self._score_evidence(text, extraction),
        }

        # Primary target is highest score
        primary = max(scores, key=scores.get)
        primary_score = scores[primary]

        # Secondary targets above threshold
        secondaries = [
            kb for kb, score in scores.items()
            if kb != primary and score > 0.3
        ]

        # Build KB-specific payload
        payload = self._build_payload(recommendation, primary)

        # Validate payload against KB schema
        validation = self._validate_payload(payload, primary)

        # Determine routing rationale
        rationale = self._build_rationale(scores, primary)

        # Check for warnings
        warnings = self._check_warnings(recommendation, primary)

        return RoutingResult(
            kb_target=primary,
            payload=payload,
            validation_result=validation,
            routing_confidence=primary_score,
            routing_rationale=rationale,
            secondary_targets=secondaries,
            warnings=warnings
        )

    def route_batch(self, recommendations: List['AtomisedRecommendation']) -> List[RoutingResult]:
        """Route multiple recommendations."""
        return [self.route(rec) for rec in recommendations]

    def _score_drug(self, text: str, extraction: Dict) -> float:
        """Score relevance to KB-1 Drug Rules."""
        score = 0.0

        # Check text patterns
        matches = len(self._drug_regex.findall(text))
        score += min(0.4, matches * 0.1)

        # Check extraction type
        if extraction.get("recommendation_type") in ["dosing", "drug_therapy", "titration"]:
            score += 0.3

        # Check for medication fields
        if extraction.get("medications"):
            score += 0.2

        # Check for dosing information
        if extraction.get("dose") or extraction.get("dosing_parameters"):
            score += 0.1

        return min(1.0, score)

    def _score_temporal(self, text: str, extraction: Dict) -> float:
        """Score relevance to KB-3 Guidelines (temporal focus)."""
        score = 0.0

        # Check text patterns
        matches = len(self._temporal_regex.findall(text))
        score += min(0.4, matches * 0.1)

        # Check for temporal constraints in extraction
        temporal = extraction.get("temporal_constraints", [])
        if temporal:
            score += min(0.3, len(temporal) * 0.1)

        # Check extraction type
        if extraction.get("recommendation_type") in ["protocol", "sequence", "timing"]:
            score += 0.2

        # Deadline urgency boost
        if any(t.get("constraint_type") == "URGENT" for t in temporal):
            score += 0.1

        return min(1.0, score)

    def _score_safety(self, text: str, extraction: Dict) -> float:
        """Score relevance to KB-4 Patient Safety."""
        score = 0.0

        # Check text patterns
        matches = len(self._safety_regex.findall(text))
        score += min(0.4, matches * 0.1)

        # Check for Class III (harm/no benefit)
        cor = extraction.get("class_of_recommendation", "")
        if "III" in str(cor):
            score += 0.4

        # Check extraction type
        if extraction.get("recommendation_type") in ["safety", "contraindication", "warning"]:
            score += 0.3

        # Check for contraindications
        if extraction.get("contraindications"):
            score += 0.2

        return min(1.0, score)

    def _score_evidence(self, text: str, extraction: Dict) -> float:
        """Score relevance to KB-15 Evidence Engine."""
        score = 0.0

        # Check text patterns
        matches = len(self._evidence_regex.findall(text))
        score += min(0.3, matches * 0.08)

        # Evidence envelope is present
        if extraction.get("class_of_recommendation") and extraction.get("level_of_evidence"):
            score += 0.4

        # Has guideline source
        if extraction.get("source_guideline"):
            score += 0.2

        # Has citations
        if extraction.get("citations"):
            score += 0.1

        return min(1.0, score)

    def _build_payload(self, rec: 'AtomisedRecommendation', target: KnowledgeBase) -> Dict:
        """Build KB-specific payload from atomised recommendation."""
        base = {
            "recommendation_id": rec.recommendation_id,
            "source": rec.guideline_source,
            "original_text": rec.original_text,
            "confidence": rec.confidence,
            "governance_status": rec.governance_status,
            "requires_sme_review": rec.requires_sme_review,
            "extracted_at": rec.extracted_at,
        }

        extraction = rec.extraction

        if target == KnowledgeBase.KB_15_EVIDENCE:
            # Evidence envelope format for KB-15
            return {
                **base,
                "evidence_envelope": {
                    "class_of_recommendation": extraction.get("class_of_recommendation"),
                    "level_of_evidence": extraction.get("level_of_evidence"),
                    "cor_display": self._cor_display(extraction.get("class_of_recommendation")),
                    "loe_display": self._loe_display(extraction.get("level_of_evidence")),
                    "source_guideline": extraction.get("source_guideline"),
                    "citations": extraction.get("citations", []),
                },
                "recommendation": {
                    "text": extraction.get("recommendation_text", rec.original_text),
                    "clinical_context": extraction.get("clinical_context"),
                    "population": extraction.get("population"),
                },
            }

        elif target == KnowledgeBase.KB_3_GUIDELINES:
            # Temporal constraint format for KB-3
            return {
                **base,
                "temporal_constraints": [
                    self._format_temporal(t) for t in extraction.get("temporal_constraints", [])
                ],
                "protocol_sequence": extraction.get("protocol_sequence"),
                "trigger_conditions": extraction.get("trigger_conditions", []),
                "clinical_context": extraction.get("clinical_context"),
            }

        elif target == KnowledgeBase.KB_1_DRUG_RULES:
            # Drug rule format for KB-1
            return {
                **base,
                "drug_rule": {
                    "medications": extraction.get("medications", []),
                    "dosing": extraction.get("dosing_parameters"),
                    "titration": extraction.get("titration_rules"),
                    "route": extraction.get("route"),
                    "frequency": extraction.get("frequency"),
                },
                "clinical_context": extraction.get("clinical_context"),
                "contraindications": extraction.get("contraindications", []),
            }

        elif target == KnowledgeBase.KB_4_SAFETY:
            # Safety alert format for KB-4
            return {
                **base,
                "safety_alert": {
                    "alert_type": extraction.get("recommendation_type", "warning"),
                    "severity": self._infer_severity(extraction),
                    "contraindications": extraction.get("contraindications", []),
                    "populations_at_risk": extraction.get("populations_at_risk", []),
                    "required_monitoring": extraction.get("monitoring_requirements", []),
                },
                "clinical_context": extraction.get("clinical_context"),
                "evidence_level": extraction.get("level_of_evidence"),
            }

        return base

    def _format_temporal(self, constraint: Dict) -> Dict:
        """Format temporal constraint for KB-3."""
        return {
            "constraint_type": constraint.get("constraint_type"),
            "value": constraint.get("value"),
            "unit": constraint.get("unit"),
            "iso8601_duration": self._to_iso8601(
                constraint.get("value"),
                constraint.get("unit")
            ),
            "relative_to": constraint.get("relative_to"),
        }

    def _to_iso8601(self, value: str, unit: str) -> Optional[str]:
        """Convert value/unit to ISO 8601 duration."""
        if not value or not unit:
            return None

        try:
            num = int(value.split('-')[0])  # Handle ranges like "2-4"
        except (ValueError, AttributeError):
            return None

        unit_lower = unit.lower()
        if 'minute' in unit_lower:
            return f"PT{num}M"
        elif 'hour' in unit_lower:
            return f"PT{num}H"
        elif 'day' in unit_lower:
            return f"P{num}D"
        elif 'week' in unit_lower:
            return f"P{num}W"
        elif 'month' in unit_lower:
            return f"P{num}M"
        elif 'year' in unit_lower:
            return f"P{num}Y"

        return None

    def _cor_display(self, cor: str) -> str:
        """Generate human-readable COR display."""
        displays = {
            "I": "Class I (Strong - Benefit >>> Risk)",
            "IIa": "Class IIa (Moderate - Benefit >> Risk)",
            "IIb": "Class IIb (Weak - Benefit ≥ Risk)",
            "III-Harm": "Class III (Harm - Risk >>> Benefit)",
            "III-NoBenefit": "Class III (No Benefit - Risk ≈ Benefit)",
        }
        return displays.get(cor, cor or "Unknown")

    def _loe_display(self, loe: str) -> str:
        """Generate human-readable LOE display."""
        displays = {
            "A": "Level A (Multiple RCTs or meta-analyses)",
            "B-R": "Level B-R (Moderate from randomized trials)",
            "B-NR": "Level B-NR (Moderate from non-randomized studies)",
            "C-LD": "Level C-LD (Limited data)",
            "C-EO": "Level C-EO (Expert opinion)",
        }
        return displays.get(loe, loe or "Unknown")

    def _infer_severity(self, extraction: Dict) -> str:
        """Infer safety severity from extraction."""
        cor = extraction.get("class_of_recommendation", "")

        if "III" in str(cor) and "Harm" in str(cor):
            return "CRITICAL"
        elif "III" in str(cor):
            return "HIGH"
        elif extraction.get("contraindications"):
            return "HIGH"
        elif extraction.get("monitoring_requirements"):
            return "MEDIUM"
        else:
            return "LOW"

    def _validate_payload(self, payload: Dict, target: KnowledgeBase) -> Dict:
        """Validate payload against KB schema requirements."""
        errors = []
        warnings = []

        # Common validations
        if not payload.get("recommendation_id"):
            errors.append("Missing recommendation_id")

        if not payload.get("governance_status"):
            errors.append("Missing governance_status")

        if payload.get("confidence", 1.0) > 0.85:
            warnings.append("Confidence exceeds 0.85 governance limit")

        # KB-specific validations
        if target == KnowledgeBase.KB_15_EVIDENCE:
            env = payload.get("evidence_envelope", {})
            if not env.get("class_of_recommendation"):
                warnings.append("Missing class_of_recommendation")
            if not env.get("level_of_evidence"):
                warnings.append("Missing level_of_evidence")

        elif target == KnowledgeBase.KB_3_GUIDELINES:
            temporal = payload.get("temporal_constraints", [])
            if not temporal:
                warnings.append("No temporal constraints for KB-3")

        elif target == KnowledgeBase.KB_1_DRUG_RULES:
            rule = payload.get("drug_rule", {})
            if not rule.get("medications"):
                warnings.append("No medications specified for drug rule")

        elif target == KnowledgeBase.KB_4_SAFETY:
            alert = payload.get("safety_alert", {})
            if not alert.get("alert_type"):
                warnings.append("No alert_type for safety alert")

        return {
            "valid": len(errors) == 0,
            "errors": errors,
            "warnings": warnings,
        }

    def _build_rationale(self, scores: Dict[KnowledgeBase, float], primary: KnowledgeBase) -> str:
        """Build human-readable routing rationale."""
        score_str = ", ".join(f"{kb.value}: {score:.2f}" for kb, score in sorted(scores.items(), key=lambda x: -x[1]))
        return f"Selected {primary.value} (highest score). Scores: {score_str}"

    def _check_warnings(self, rec: 'AtomisedRecommendation', target: KnowledgeBase) -> List[str]:
        """Check for routing warnings."""
        warnings = []

        if rec.confidence < 0.5:
            warnings.append("Low extraction confidence - may need manual review")

        if rec.governance_status == "DRAFT":
            warnings.append("DRAFT status - requires SME approval before production use")

        if target == KnowledgeBase.KB_4_SAFETY and rec.confidence < 0.8:
            warnings.append("Safety-critical routing with low confidence - prioritize review")

        return warnings

    def to_json(self, result: RoutingResult) -> str:
        """Serialize routing result to JSON."""
        return json.dumps({
            "kb_target": result.kb_target.value,
            "payload": result.payload,
            "validation": result.validation_result,
            "routing_confidence": result.routing_confidence,
            "routing_rationale": result.routing_rationale,
            "secondary_targets": [kb.value for kb in result.secondary_targets],
            "warnings": result.warnings,
        }, indent=2, default=str)


def main():
    """Demo the KB Router."""
    print("=" * 60)
    print("Phase 5: KB Router Demo")
    print("=" * 60)

    # Create mock AtomisedRecommendation for testing
    from dataclasses import dataclass, field
    from typing import Dict, Any

    @dataclass
    class MockRecommendation:
        recommendation_id: str
        original_text: str
        extraction: Dict[str, Any]
        confidence: float = 0.75
        guideline_source: str = "SSC-2021"
        governance_status: str = "DRAFT"
        requires_sme_review: bool = True
        extracted_at: str = field(default_factory=lambda: datetime.utcnow().isoformat())

    router = KBRouter()

    # Test cases
    test_cases = [
        MockRecommendation(
            recommendation_id="test-001",
            original_text="Class I, LOE A: Administer broad-spectrum antibiotics within 1 hour of sepsis recognition.",
            extraction={
                "class_of_recommendation": "I",
                "level_of_evidence": "A",
                "temporal_constraints": [{"constraint_type": "DEADLINE", "value": "1", "unit": "hour"}],
                "recommendation_type": "timing",
            }
        ),
        MockRecommendation(
            recommendation_id="test-002",
            original_text="Class III (Harm): Do not use dopamine as first-line vasopressor.",
            extraction={
                "class_of_recommendation": "III-Harm",
                "level_of_evidence": "B-R",
                "recommendation_type": "contraindication",
                "medications": ["dopamine"],
                "contraindications": ["first-line vasopressor use"],
            }
        ),
        MockRecommendation(
            recommendation_id="test-003",
            original_text="Titrate norepinephrine from 0.01 to 0.1 mcg/kg/min to maintain MAP >= 65 mmHg.",
            extraction={
                "recommendation_type": "dosing",
                "medications": ["norepinephrine"],
                "dosing_parameters": {"start": "0.01 mcg/kg/min", "max": "0.1 mcg/kg/min"},
                "clinical_context": "maintain MAP >= 65 mmHg",
            }
        ),
    ]

    print("\n--- Routing Tests ---")
    for i, rec in enumerate(test_cases, 1):
        print(f"\n[Test {i}]: {rec.original_text[:60]}...")
        result = router.route(rec)

        print(f"  Primary Target: {result.kb_target.value}")
        print(f"  Confidence: {result.routing_confidence:.2f}")
        print(f"  Secondary: {[kb.value for kb in result.secondary_targets]}")
        print(f"  Validation: {'✓ Valid' if result.validation_result['valid'] else '✗ Invalid'}")
        if result.warnings:
            print(f"  Warnings: {result.warnings}")
        print(f"  Rationale: {result.routing_rationale}")


if __name__ == "__main__":
    main()
