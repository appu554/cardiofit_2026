"""
Tier Assigner Classifier: Multiclass XGBoost for TIER_1/TIER_2/NOISE.

Replaces the TrainedTieringClassifier placeholder with a real implementation.
Implements the TieringClassifier ABC so it can be used as a drop-in replacement
for RuleBasedTieringClassifier in the signal merger pipeline.

Model Training:
    Train with: python data/train_tier_assigner.py
    Input: golden_dataset_enriched.parquet + golden_dataset_splits.json
    Output: models/tier_assigner.joblib

Runtime Integration:
    Selected via GuidelineProfile.tiering_classifier = "trained"
    Instantiated in run_pipeline_targeted.py with model_path from
    GuidelineProfile.tiering_golden_dataset.
"""

from __future__ import annotations

import logging
from pathlib import Path
from typing import Optional

from ..tiering_classifier import TieringClassifier, TieringResult

logger = logging.getLogger(__name__)

# Full feature set for tier assignment
TIER_ASSIGNER_FEATURES = [
    "text_length",
    "word_count",
    "has_number",
    "has_unit",
    "has_comparison",
    "has_safety_keyword",
    "n_channels",
    "has_channel_B",
    "has_channel_C",
    "has_channel_D",
    "has_channel_E",
    "has_channel_F",
    "has_channel_G",
    "has_channel_H",
    "has_disagreement",
    "merged_confidence",
    "clinical_verb_count",
    "clinical_term_density",
    "section_depth",
    "has_drug_name",
    "has_drug_class",
    "is_bare_abbreviation",
    "noise_archetype_code",
    "page_number",
]

# Label encoding for tier assignment
_TIER_LABELS = ["NOISE", "TIER_1", "TIER_2"]


class TierAssignerClassifier(TieringClassifier):
    """Multiclass XGBoost tier classifier implementing TieringClassifier ABC.

    Replaces the former TrainedTieringClassifier placeholder.
    """

    def __init__(self, model_path: str | Path) -> None:
        """Load a trained tier assigner model.

        Args:
            model_path: Path to the joblib-serialized XGBoost model.

        Raises:
            FileNotFoundError: If model_path does not exist.
        """
        model_path = Path(model_path)
        if not model_path.exists():
            raise FileNotFoundError(f"Tier assigner model not found: {model_path}")

        import joblib
        self._model = joblib.load(model_path)
        logger.info(f"Loaded tier assigner model from {model_path}")

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
        """Classify a span using the trained XGBoost model.

        Computes features inline (lightweight — no DB access needed)
        and runs the model for prediction.
        """
        from ..feature_enrichment import enrich_span_features

        # Compute features (full_text unavailable at classify time — use span text)
        features = enrich_span_features(
            text=text,
            start=0,
            end=len(text),
            contributing_channels=contributing_channels,
            channel_confidences=channel_confidences,
            merged_confidence=merged_confidence,
            has_disagreement=has_disagreement,
            full_text=text,  # context features will be approximate
            section_id=section_id,
            page_number=page_number,
        )

        import numpy as np

        # Extract feature vector
        x = np.array([[features.get(f, 0) for f in TIER_ASSIGNER_FEATURES]])

        # Predict class and probabilities
        pred_class = int(self._model.predict(x)[0])
        proba = self._model.predict_proba(x)[0]

        tier = _TIER_LABELS[pred_class] if pred_class < len(_TIER_LABELS) else "TIER_2"
        confidence = float(max(proba))

        return TieringResult(
            tier=tier,
            confidence=confidence,
            reason=f"Trained classifier: {tier} (p={confidence:.3f})",
            noise_archetype=features.get("noise_archetype"),
        )

    @staticmethod
    def try_load(model_path: Optional[str | Path]) -> Optional[TierAssignerClassifier]:
        """Attempt to load, returning None if unavailable."""
        if not model_path:
            return None
        try:
            return TierAssignerClassifier(model_path)
        except (FileNotFoundError, ImportError) as e:
            logger.debug(f"Tier assigner not available: {e}")
            return None
