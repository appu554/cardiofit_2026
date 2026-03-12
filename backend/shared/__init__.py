"""
Shared module for Clinical Synthesis Hub microservices.

This module provides common functionality used across all microservices
in the Clinical Synthesis Hub, including:

- Authentication middleware
- Common utilities
- Shared models
- FHIR utilities
"""

# Import auto_import to ensure shared module is importable
from .auto_import import ensure_shared_importable

# Import packages
from . import auth
from . import models
from . import fhir

__all__ = ["auth", "models", "fhir", "ensure_shared_importable"]
