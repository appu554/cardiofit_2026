"""
Clinical Validation Framework for Production Clinical Intelligence System

This module provides comprehensive clinical validation capabilities including:
- Evidence-based testing and validation
- Clinical outcome tracking and analysis
- Regulatory compliance validation
- Clinical safety protocols
- Performance benchmarking against clinical standards
"""

from .clinical_validator import (
    ClinicalValidator,
    ValidationResult,
    ValidationSeverity,
    ValidationCategory,
    ClinicalEvidence,
    ValidationMetrics
)

# Additional validation modules would be imported here
# from .evidence_based_testing import (...)
# from .outcome_tracker import (...)
# from .regulatory_compliance import (...)
# from .safety_protocols import (...)

__all__ = [
    # Core Validation
    'ClinicalValidator',
    'ValidationResult',
    'ValidationSeverity',
    'ValidationCategory',
    'ClinicalEvidence',
    'ValidationMetrics'
]