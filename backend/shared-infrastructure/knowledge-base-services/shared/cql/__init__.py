"""
CQL Validation Module - L5 of the V3 Guideline Curation Pipeline

This module provides tools for validating extracted facts against
existing CQL logic in vaidshala/tier-4-guidelines/.

Components:
- CQL Guideline Registry: Explicit mapping between CQL defines, guidelines, and KB fields
- Compatibility Checker: Validate extracted facts match CQL expectations
- Gap Detector: Bidirectional gap detection (forward, backward, coverage)
- Registry Validator: Ensure registry integrity and consistency

Key Principle: CQL already exists. We validate compatibility, not generate CQL.

The registry is your SECOND MOAT after CR-IR:
- Anyone can parse PDFs with LLMs
- Knowing exactly which clinical logic depends on which guideline sentence
  is institutional knowledge that compounds over time

Usage:
    from cql import CQLCompatibilityChecker, CQLGapDetector, RegistryValidator

    # Check compatibility
    checker = CQLCompatibilityChecker(registry_path, vaidshala_path)
    report = checker.check_compatibility(extracted_facts, "T2DMGuidelines.cql")

    # Detect gaps
    detector = CQLGapDetector(registry_path, vaidshala_path)
    forward_gaps = detector.detect_forward_gaps(extracted_facts)
    backward_gaps = detector.detect_backward_gaps()

    # Validate registry integrity
    validator = RegistryValidator(registry_path, vaidshala_path)
    validation_report = validator.validate_all()
"""

from .compatibility_checker import (
    CQLCompatibilityChecker,
    CompatibilityReport,
    CompatibilityMatch,
    CompatibilityIssue,
)
from .gap_detector import (
    CQLGapDetector,
    GapReport,
    ForwardGap,
    BackwardGap,
    CoverageGap,
)
from .registry.registry_validator import (
    RegistryValidator,
    ValidationReport,
    ValidationError,
    ValidationWarning,
    validate_registry,
)

__all__ = [
    # Compatibility Checker
    "CQLCompatibilityChecker",
    "CompatibilityReport",
    "CompatibilityMatch",
    "CompatibilityIssue",
    # Gap Detector
    "CQLGapDetector",
    "GapReport",
    "ForwardGap",
    "BackwardGap",
    "CoverageGap",
    # Registry Validator
    "RegistryValidator",
    "ValidationReport",
    "ValidationError",
    "ValidationWarning",
    "validate_registry",
]

__version__ = "1.0.0"
