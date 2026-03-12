"""
FHIR validators for Clinical Synthesis Hub.

This module provides validators for FHIR resources used across
all microservices in the Clinical Synthesis Hub.
"""

from .fhir import validate_fhir_resource

__all__ = ["validate_fhir_resource"]
