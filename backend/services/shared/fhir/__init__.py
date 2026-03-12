"""
FHIR module for Clinical Synthesis Hub microservices.

This module provides standardized FHIR routing and handling for all microservices.
"""

from .router import create_fhir_router, FHIRRouterConfig
from .service import FHIRServiceBase, MockFHIRService

__all__ = ["create_fhir_router", "FHIRRouterConfig", "FHIRServiceBase", "MockFHIRService"]
