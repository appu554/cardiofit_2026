"""
Tiering Classifier: Pluggable span classification interface.

Classifies MergedSpans into TIER_1 (high-confidence clinical signal),
TIER_2 (medium confidence), or NOISE (non-clinical).

Architecture:
    - ``TieringClassifier`` — Abstract base class defining the interface
    - ``RuleBasedTieringClassifier`` — Default implementation using
      multi-channel corroboration, confidence, and text heuristics
    - ``TrainedTieringClassifier`` — Placeholder for future ML model

The classifier is invoked by Signal Merger after span construction.
GuidelineProfile.tiering_classifier selects which implementation to use.

Pipeline Position:
    Signal Merger -> Tiering Classifier (THIS) -> MergedSpan.tier field
"""

from __future__ import annotations

from abc import ABC, abstractmethod
from dataclasses import dataclass, field
from typing import Literal, Optional

from .noise_archetypes import classify_noise_archetype


TierLabel = Literal["TIER_1", "TIER_2", "NOISE"]


@dataclass
class TieringResult:
    """Result of classifying a single span."""
    tier: TierLabel
    confidence: float          # 0.0-1.0 classifier confidence
    reason: str                # human-readable explanation
    noise_archetype: Optional[str] = None  # NoiseArchetype value if classified as noise


class TieringClassifier(ABC):
    """Abstract base class for span tiering classifiers."""

    @abstractmethod
    def classify(
        self,
        text: str,
        merged_confidence: float,
        contributing_channels: list[str],
        channel_confidences: dict[str, float],
        has_disagreement: bool,
        section_id: Optional[str] = None,
        page_number: Optional[int] = None,
    ) -> TieringResult:
        """Classify a single MergedSpan into a tier.

        Args:
            text: The merged span text.
            merged_confidence: Signal Merger confidence (0.0-1.0).
            contributing_channels: List of channel IDs (e.g. ["B", "C"]).
            channel_confidences: Per-channel confidence scores.
            has_disagreement: Whether contributing channels disagree on text.
            section_id: Section ID from the guideline tree.
            page_number: Page number in the source PDF.

        Returns:
            TieringResult with tier label, confidence, and reason.
        """
        ...


class RuleBasedTieringClassifier(TieringClassifier):
    """Rule-based tiering using multi-channel corroboration and text heuristics.

    Classification Rules:
        TIER_1: Multi-channel (≥2) corroboration with merged_confidence ≥ 0.7,
                OR 2-channel WITH disagreement and confidence ≥ 0.7
                (disagreement = multiple channels found the span independently),
                OR single-channel B/C with confidence ≥ 0.9 and clinical text.
        TIER_2: Single channel with confidence ≥ 0.5, or moderate confidence.
        NOISE:  Single Channel D (bare table cells), very short text (≤ 4 chars),
                bare drug class abbreviations (single-channel), or confidence < 0.3.

    Golden Dataset Statistics (6,326 reviewer-labelled spans, 2026-03-02):
        By channel count:
        - 1-channel: 24.6% precision (75.4% noise)
        - 2-channel: 48.1% precision → TIER_2
        - 3-channel: 58.3% precision → TIER_1
        - 4-channel: 100.0% precision → TIER_1
        By channel type (single-channel):
        - Channel B alone: dominanted by bare class abbreviations (ARB 0/70, MRA 0/66, ACEi 1/79)
        - Channel D alone: 6.6% precision (93.4% noise)
        - Channel F alone: 30.8% precision
        Disagreement signal:
        - has_disagreement=True: 57.4% precision (UPGRADE signal, not downgrade)
        - has_disagreement=False: 24.2% precision
        Confidence is NOT discriminative:
        - CONFIRMED avg=0.921, REJECTED avg=0.930 (overlapping distributions)
        - Low-end gate (< 0.3) and Channel D gate (< 0.6) still valid as safety floors
    """

    # Minimum text length to be considered clinical
    MIN_CLINICAL_LENGTH = 4

    # Channels that produce high-quality primary matches
    PRIMARY_CHANNELS = {"B", "C"}

    # Bare drug class abbreviations that are overwhelmingly noise when
    # single-channel (golden dataset: ARB 0/70, MRA 0/66, ACEi 1/79 confirmed).
    # Multi-channel corroboration can still elevate these to TIER_1/TIER_2.
    _BARE_CLASS_ABBREVS = frozenset({
        "ARB", "MRA", "ACEi", "ACEI", "ACE", "CCB", "BB",
        "NSAID", "NSAIDs", "SGLT2", "DPP4", "GLP1",
    })

    # Noise indicators in text
    _NOISE_PATTERNS = frozenset({
        "|", "---", "───", "***", "...", "—", "→",
    })

    def classify(
        self,
        text: str,
        merged_confidence: float,
        contributing_channels: list[str],
        channel_confidences: dict[str, float],
        has_disagreement: bool,
        section_id: Optional[str] = None,
        page_number: Optional[int] = None,
    ) -> TieringResult:
        """Classify a span using rule-based heuristics."""
        n_channels = len(contributing_channels)
        text_stripped = text.strip()

        # ── NOISE checks ─────────────────────────────────────────────
        # Very short text
        if len(text_stripped) <= self.MIN_CLINICAL_LENGTH:
            return TieringResult(
                tier="NOISE",
                confidence=0.9,
                reason=f"Text too short ({len(text_stripped)} chars)",
            )

        # Noise archetype classification (8-archetype taxonomy)
        # Multi-channel corroboration overrides archetype noise classification,
        # since multiple channels independently finding the same span indicates
        # genuine clinical content even if the text matches an archetype.
        archetype = classify_noise_archetype(text_stripped)
        if archetype is not None and n_channels == 1:
            return TieringResult(
                tier="NOISE",
                confidence=0.86,
                reason=f"Noise archetype: {archetype} (single-channel)",
                noise_archetype=archetype,
            )

        # Bare drug class abbreviations as single-channel spans
        # Golden dataset: ARB 0/70, MRA 0/66, ACEi 1/79 confirmed.
        # Multi-channel corroboration can still elevate these (ACEi confirmed at 4-channel).
        if (n_channels == 1 and text_stripped in self._BARE_CLASS_ABBREVS):
            return TieringResult(
                tier="NOISE",
                confidence=0.88,
                reason=f"Bare class abbreviation '{text_stripped}' (single-channel, golden dataset <1.5% precision)",
                noise_archetype="standalone_drug_name",
            )

        # Table artifact patterns
        if any(p in text_stripped for p in self._NOISE_PATTERNS):
            if n_channels == 1 and "D" in contributing_channels:
                return TieringResult(
                    tier="NOISE",
                    confidence=0.85,
                    reason="Table artifact (single Channel D with noise pattern)",
                )

        # Single Channel D with low confidence
        if (n_channels == 1 and "D" in contributing_channels
                and merged_confidence < 0.6):
            return TieringResult(
                tier="NOISE",
                confidence=0.80,
                reason="Single Channel D, low confidence (93.4% noise rate)",
            )

        # Very low confidence (safety floor — confidence is not discriminative
        # between CONFIRMED/REJECTED at higher values, but < 0.3 is still rare garbage)
        if merged_confidence < 0.3:
            return TieringResult(
                tier="NOISE",
                confidence=0.75,
                reason=f"Very low confidence ({merged_confidence:.2f})",
            )

        # ── TIER_1 checks ────────────────────────────────────────────
        # 3+ channel corroboration (golden dataset: 58.3-100% precision)
        if n_channels >= 3 and merged_confidence >= 0.7:
            return TieringResult(
                tier="TIER_1",
                confidence=0.95,
                reason=f"{n_channels}-channel corroboration, confidence {merged_confidence:.2f}",
            )

        # 2+ channel corroboration with confidence ≥ 0.7
        # NOTE: Disagreement is an UPGRADE signal (57.4% precision vs 24.2% without).
        # Multiple channels independently finding the same span — even with text
        # differences — indicates genuine clinical content worth reviewing.
        if n_channels >= 2 and merged_confidence >= 0.7:
            qualifier = "with disagreement" if has_disagreement else "agreement"
            return TieringResult(
                tier="TIER_1",
                confidence=0.87 if has_disagreement else 0.85,
                reason=f"2-channel {qualifier}, confidence {merged_confidence:.2f}",
            )

        # Single primary channel (B or C) with very high confidence
        if (n_channels == 1
                and set(contributing_channels) & self.PRIMARY_CHANNELS
                and merged_confidence >= 0.9):
            return TieringResult(
                tier="TIER_1",
                confidence=0.75,
                reason=f"Single primary channel ({contributing_channels[0]}), high confidence",
            )

        # ── TIER_2 (default for moderate signals) ────────────────────
        # Multi-channel below confidence threshold
        if n_channels >= 2:
            return TieringResult(
                tier="TIER_2",
                confidence=0.65,
                reason=f"{n_channels}-channel, moderate confidence ({merged_confidence:.2f})",
            )

        if merged_confidence >= 0.5:
            return TieringResult(
                tier="TIER_2",
                confidence=0.60,
                reason=f"Moderate confidence ({merged_confidence:.2f})",
            )

        # Fallback
        return TieringResult(
            tier="TIER_2",
            confidence=0.50,
            reason="Default tier (no strong signal either way)",
        )


class TrainedTieringClassifier(TieringClassifier):
    """Trained ML tiering classifier using TierAssignerClassifier.

    Loads an XGBoost multiclass model from the path specified in
    GuidelineProfile.tiering_golden_dataset.  Select via
    ``tiering_classifier = "trained"`` in the profile.
    """

    def __init__(self, model_path: str) -> None:
        from .classifiers.tier_assigner import TierAssignerClassifier
        self._delegate = TierAssignerClassifier(model_path)

    def classify(
        self,
        text: str,
        merged_confidence: float,
        contributing_channels: list[str],
        channel_confidences: dict[str, float],
        has_disagreement: bool,
        section_id: Optional[str] = None,
        page_number: Optional[int] = None,
    ) -> TieringResult:
        return self._delegate.classify(
            text=text,
            merged_confidence=merged_confidence,
            contributing_channels=contributing_channels,
            channel_confidences=channel_confidences,
            has_disagreement=has_disagreement,
            section_id=section_id,
            page_number=page_number,
        )


class ShadowTieringClassifier(TieringClassifier):
    """Shadow mode: runs BOTH rule-based and trained classifiers.

    Uses rule-based output for the actual tier assignment (no behavior change)
    but logs trained classifier predictions to l2_classifier_shadow_log for
    comparison.  When agreement rate > 95% for 2 consecutive weeks, it is
    safe to switch to ``tiering_classifier = "trained"``.

    Select via ``tiering_classifier = "shadow"`` in the profile.
    """

    def __init__(self, model_path: str) -> None:
        self._rule_based = RuleBasedTieringClassifier()
        self._trained = None
        try:
            from .classifiers.tier_assigner import TierAssignerClassifier
            self._trained = TierAssignerClassifier(model_path)
        except (FileNotFoundError, ImportError) as e:
            import warnings
            warnings.warn(
                f"ShadowTieringClassifier: trained model unavailable ({e}). "
                f"Running rule-based only — shadow log will show 'model_unavailable'.",
                stacklevel=2,
            )
        self._shadow_log: list[dict] = []  # buffered for batch insert

    def classify(
        self,
        text: str,
        merged_confidence: float,
        contributing_channels: list[str],
        channel_confidences: dict[str, float],
        has_disagreement: bool,
        section_id: Optional[str] = None,
        page_number: Optional[int] = None,
    ) -> TieringResult:
        """Classify using rule-based, log trained prediction for comparison."""
        # Rule-based is the authoritative result
        rule_result = self._rule_based.classify(
            text=text,
            merged_confidence=merged_confidence,
            contributing_channels=contributing_channels,
            channel_confidences=channel_confidences,
            has_disagreement=has_disagreement,
            section_id=section_id,
            page_number=page_number,
        )

        # Trained is for shadow comparison only
        if self._trained is not None:
            try:
                trained_result = self._trained.classify(
                    text=text,
                    merged_confidence=merged_confidence,
                    contributing_channels=contributing_channels,
                    channel_confidences=channel_confidences,
                    has_disagreement=has_disagreement,
                    section_id=section_id,
                    page_number=page_number,
                )
            except Exception:
                trained_result = TieringResult(
                    tier="TIER_2", confidence=0.0, reason="trained classifier error"
                )
        else:
            trained_result = TieringResult(
                tier="UNKNOWN", confidence=0.0, reason="model_unavailable"
            )

        # Buffer shadow log entry
        self._shadow_log.append({
            "rule_tier": rule_result.tier,
            "rule_confidence": rule_result.confidence,
            "rule_reason": rule_result.reason,
            "trained_tier": trained_result.tier,
            "trained_confidence": trained_result.confidence,
            "trained_reason": trained_result.reason,
            "tiers_agree": rule_result.tier == trained_result.tier,
        })

        return rule_result

    @property
    def shadow_log(self) -> list[dict]:
        """Access buffered shadow log entries for batch DB insert."""
        return self._shadow_log

    @property
    def agreement_rate(self) -> float:
        """Compute current agreement rate between rule-based and trained."""
        if not self._shadow_log:
            return 0.0
        agree = sum(1 for entry in self._shadow_log if entry["tiers_agree"])
        return agree / len(self._shadow_log)
