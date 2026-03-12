"""
Feature Drift Monitor: PSI-based distribution shift detection.

Compares feature distributions between training data and recent predictions
to detect when the model's operating environment has changed enough to
warrant retraining.

PSI (Population Stability Index) interpretation:
    < 0.1  — No significant change
    0.1-0.2 — Moderate change, monitor
    > 0.2  — Significant shift, retrain recommended

Integrated into the weekly retraining pipeline (weekly_retrain.py).
"""

from __future__ import annotations

import logging
from dataclasses import dataclass, field

import numpy as np

logger = logging.getLogger(__name__)


@dataclass
class FeatureDriftResult:
    """PSI result for a single feature."""
    feature_name: str
    psi_value: float
    status: str  # "OK", "MONITOR", "RETRAIN"


@dataclass
class DriftReport:
    """Complete drift analysis report."""
    feature_results: list[FeatureDriftResult] = field(default_factory=list)
    overall_max_psi: float = 0.0
    drifted_features: list[str] = field(default_factory=list)
    recommendation: str = "OK"  # "OK", "MONITOR", "RETRAIN"

    def to_dict(self) -> dict:
        """Serialize for inclusion in retrain_report.json."""
        return {
            "overall_max_psi": self.overall_max_psi,
            "drifted_features": self.drifted_features,
            "recommendation": self.recommendation,
            "features": [
                {
                    "name": r.feature_name,
                    "psi": round(r.psi_value, 4),
                    "status": r.status,
                }
                for r in self.feature_results
            ],
        }


class DriftMonitor:
    """PSI-based feature drift detector.

    Usage:
        monitor = DriftMonitor()
        report = monitor.analyze(training_df, recent_df, feature_columns)
    """

    PSI_MONITOR_THRESHOLD = 0.1
    PSI_RETRAIN_THRESHOLD = 0.2

    @staticmethod
    def compute_psi(
        expected: np.ndarray,
        actual: np.ndarray,
        n_bins: int = 10,
    ) -> float:
        """Compute Population Stability Index between two distributions.

        Args:
            expected: Training data feature values.
            actual: Recent prediction feature values.
            n_bins: Number of bins for discretization.

        Returns:
            PSI value (0 = identical distributions).
        """
        # Handle edge cases
        if len(expected) == 0 or len(actual) == 0:
            return 0.0

        # Create bins from expected distribution
        breakpoints = np.linspace(
            min(expected.min(), actual.min()),
            max(expected.max(), actual.max()),
            n_bins + 1,
        )

        # Compute bin proportions
        expected_counts = np.histogram(expected, bins=breakpoints)[0]
        actual_counts = np.histogram(actual, bins=breakpoints)[0]

        # Add small epsilon to avoid division by zero and log(0)
        eps = 1e-4
        expected_pct = (expected_counts + eps) / (len(expected) + eps * n_bins)
        actual_pct = (actual_counts + eps) / (len(actual) + eps * n_bins)

        # PSI formula: sum((actual% - expected%) * ln(actual% / expected%))
        psi = np.sum((actual_pct - expected_pct) * np.log(actual_pct / expected_pct))
        return float(psi)

    def analyze(
        self,
        training_features: dict[str, np.ndarray],
        recent_features: dict[str, np.ndarray],
    ) -> DriftReport:
        """Analyze feature drift between training and recent data.

        Args:
            training_features: Dict of feature_name -> array of training values.
            recent_features: Dict of feature_name -> array of recent values.

        Returns:
            DriftReport with per-feature PSI and overall recommendation.
        """
        results = []
        max_psi = 0.0
        drifted = []

        for feature_name in training_features:
            if feature_name not in recent_features:
                continue

            expected = np.asarray(training_features[feature_name], dtype=float)
            actual = np.asarray(recent_features[feature_name], dtype=float)

            psi = self.compute_psi(expected, actual)
            max_psi = max(max_psi, psi)

            if psi > self.PSI_RETRAIN_THRESHOLD:
                status = "RETRAIN"
                drifted.append(feature_name)
            elif psi > self.PSI_MONITOR_THRESHOLD:
                status = "MONITOR"
            else:
                status = "OK"

            results.append(FeatureDriftResult(
                feature_name=feature_name,
                psi_value=psi,
                status=status,
            ))

        # Overall recommendation
        if drifted:
            recommendation = "RETRAIN"
        elif max_psi > self.PSI_MONITOR_THRESHOLD:
            recommendation = "MONITOR"
        else:
            recommendation = "OK"

        return DriftReport(
            feature_results=results,
            overall_max_psi=max_psi,
            drifted_features=drifted,
            recommendation=recommendation,
        )
