"""
Safety Criticality Detector: Binary logistic regression for safety-critical spans.

Detects spans containing contraindications, black-box warnings, dose limits,
and other safety-critical clinical content. Safety-critical spans receive a
tier floor of TIER_2 — they are never classified as NOISE even if the noise
gate would otherwise suppress them.

Lightweight model (sklearn LogisticRegression) using a subset of features
focused on safety indicators.

Model Training:
    Train with: python data/train_safety_criticality.py
    Input: golden_dataset_enriched.parquet with manual safety labels
    Output: models/safety_criticality.joblib
"""

from __future__ import annotations

import logging
import re
from pathlib import Path
from typing import Optional

logger = logging.getLogger(__name__)

# Subset of features relevant to safety criticality
SAFETY_FEATURES = [
    "has_comparison",
    "has_drug_name",
    "has_unit",
    "has_safety_keyword",
    "clinical_term_density",
    "text_length",
    "has_number",
    "n_channels",
    "merged_confidence",
]

# Heuristic safety patterns (used as fallback when no trained model)
_SAFETY_PATTERN_RE = re.compile(
    r'contraindicated|avoid\b|black\s*box|maximum\s+dose|do\s+not\s+use|'
    r'not\s+recommended|discontinue\s+if|withhold\s+if|'
    r'life[\s-]threatening|fatal|anaphylaxis|'
    r'box(?:ed)?\s+warning|renal\s+failure|hepatotoxic',
    re.IGNORECASE,
)


class SafetyCriticalityDetector:
    """Binary safety criticality classifier.

    Can operate in two modes:
    1. Trained model: Uses sklearn LogisticRegression loaded from joblib.
    2. Heuristic fallback: Regex pattern matching for safety keywords.

    The heuristic mode is always available; the trained model improves
    precision by considering feature context rather than keyword matching alone.
    """

    def __init__(self, model_path: Optional[str | Path] = None) -> None:
        """Initialize with optional trained model.

        Args:
            model_path: Path to joblib-serialized LogisticRegression model.
                If None or file not found, uses heuristic fallback.
        """
        self._model = None
        if model_path:
            model_path = Path(model_path)
            if model_path.exists():
                try:
                    import joblib
                    self._model = joblib.load(model_path)
                    logger.info(f"Loaded safety criticality model from {model_path}")
                except (ImportError, Exception) as e:
                    logger.warning(f"Could not load safety model: {e}, using heuristic")

    @property
    def uses_trained_model(self) -> bool:
        """Whether a trained model is loaded (vs heuristic fallback)."""
        return self._model is not None

    def is_safety_critical(
        self,
        text: str,
        features: Optional[dict] = None,
    ) -> tuple[bool, float]:
        """Predict whether a span is safety-critical.

        Args:
            text: The span text.
            features: Optional feature dict from enrich_span_features().
                Required when using trained model; ignored in heuristic mode.

        Returns:
            (is_safety_critical, confidence) — True if the span contains
            safety-critical clinical content.
        """
        # Trained model path
        if self._model is not None and features is not None:
            return self._predict_trained(features)

        # Heuristic fallback
        return self._predict_heuristic(text)

    def _predict_trained(self, features: dict) -> tuple[bool, float]:
        """Use trained logistic regression model."""
        import numpy as np

        x = np.array([[features.get(f, 0) for f in SAFETY_FEATURES]])
        proba = self._model.predict_proba(x)[0]
        safety_prob = float(proba[1]) if len(proba) > 1 else float(proba[0])

        # Conservative threshold: only flag as safety-critical with high confidence
        is_critical = safety_prob > 0.7
        return is_critical, safety_prob

    def _predict_heuristic(self, text: str) -> tuple[bool, float]:
        """Use regex pattern matching as fallback."""
        match = _SAFETY_PATTERN_RE.search(text)
        if match:
            return True, 0.8
        return False, 0.1

    @staticmethod
    def try_load(model_path: Optional[str | Path] = None) -> SafetyCriticalityDetector:
        """Create a SafetyCriticalityDetector, always succeeds (heuristic fallback).

        Unlike the other classifiers, this always returns an instance because
        the heuristic fallback provides reasonable safety detection.
        """
        return SafetyCriticalityDetector(model_path)
