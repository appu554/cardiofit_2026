"""
Noise Gate Classifier: Binary XGBoost model for noise detection.

Predicts whether a span is noise (REJECT) or signal (CONFIRM/EDIT).
Used as a pre-filter before the tier assigner — spans classified as
noise with high confidence skip tier assignment entirely.

Model Training:
    Train with: python data/train_noise_gate.py
    Input: golden_dataset_enriched.parquet + golden_dataset_splits.json
    Output: models/noise_gate.joblib

Runtime Integration:
    Called from RuleBasedTieringClassifier.classify() when model file exists.
    Falls back to rule-based when model absent (zero behavior change).
"""

from __future__ import annotations

import logging
from pathlib import Path
from typing import Optional

logger = logging.getLogger(__name__)

# Feature columns used by the noise gate (subset of enrich_span_features output)
NOISE_GATE_FEATURES = [
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
]


class NoiseGateClassifier:
    """Binary noise classifier using XGBoost.

    Predicts P(noise | features). A span is classified as noise if
    the predicted probability exceeds the configured threshold.
    """

    def __init__(
        self,
        model_path: str | Path,
        threshold: float = 0.85,
    ) -> None:
        """Load a trained noise gate model.

        Args:
            model_path: Path to the joblib-serialized XGBoost model.
            threshold: Probability threshold for noise classification.
                Only predict noise when P(noise) > threshold.

        Raises:
            FileNotFoundError: If model_path does not exist.
        """
        model_path = Path(model_path)
        if not model_path.exists():
            raise FileNotFoundError(f"Noise gate model not found: {model_path}")

        import joblib
        self._model = joblib.load(model_path)
        self._threshold = threshold
        logger.info(f"Loaded noise gate model from {model_path} (threshold={threshold})")

    def predict(self, features: dict) -> tuple[bool, float]:
        """Predict whether a span is noise.

        Args:
            features: Feature dict from enrich_span_features().

        Returns:
            (is_noise, probability) — True if noise with P > threshold.
        """
        import numpy as np

        # Extract feature vector in correct column order
        x = np.array([[features.get(f, 0) for f in NOISE_GATE_FEATURES]])
        proba = self._model.predict_proba(x)[0]

        # proba[1] is P(noise) since label encoding: 0=signal, 1=noise
        noise_prob = float(proba[1]) if len(proba) > 1 else float(proba[0])
        is_noise = noise_prob > self._threshold

        return is_noise, noise_prob

    @staticmethod
    def try_load(model_path: Optional[str | Path]) -> Optional[NoiseGateClassifier]:
        """Attempt to load a noise gate model, returning None if unavailable.

        This is the recommended entry point for pipeline integration —
        returns None instead of raising when the model doesn't exist,
        enabling graceful fallback to rule-based classification.
        """
        if not model_path:
            return None
        try:
            return NoiseGateClassifier(model_path)
        except (FileNotFoundError, ImportError) as e:
            logger.debug(f"Noise gate not available: {e}")
            return None
