"""
V4 ML Classifiers for span tiering.

Three classifiers that progressively replace rule-based heuristics:
1. NoiseGateClassifier — binary XGBoost: is this span noise?
2. TierAssignerClassifier — multiclass XGBoost: TIER_1 / TIER_2 / NOISE
3. SafetyCriticalityDetector — binary logistic regression: is this safety-critical?

All classifiers are optional at runtime. When model files are absent,
the pipeline falls back to RuleBasedTieringClassifier with zero behavior change.
"""

from .noise_gate import NoiseGateClassifier
from .tier_assigner import TierAssignerClassifier
from .safety_criticality import SafetyCriticalityDetector

__all__ = [
    "NoiseGateClassifier",
    "TierAssignerClassifier",
    "SafetyCriticalityDetector",
]
