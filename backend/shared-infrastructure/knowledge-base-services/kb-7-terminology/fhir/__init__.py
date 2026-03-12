"""
FHIR Terminology Module for KB7 Hybrid Architecture

This module provides FHIR R4 terminology services integrated with the
KB7 hybrid query router for optimal performance.

Key Features:
- Hybrid PostgreSQL/GraphDB routing for different query types
- FHIR R4 compliance with JSON and XML support
- Performance optimization with <200ms target response times
- Intelligent caching and fallback mechanisms
- Circuit breaker protection
- Comprehensive monitoring and metrics

Components:
- models: FHIR data models and validation
- client: Query router integration client
- terminology_service: Main service implementation
- endpoints: FastAPI endpoints for FHIR operations
"""

from .models import (
    CodeSystemLookupRequest,
    CodeSystemLookupResponse,
    ValueSetExpandRequest,
    ConceptMapTranslateRequest,
    ValidateCodeRequest,
    ValidateCodeResponse,
    OperationOutcome,
    Parameters,
    TerminologyCapabilities
)

from .client import (
    FHIRTerminologyClient,
    QueryRouterTimeoutError,
    QueryRouterUnavailableError
)

from .terminology_service import FHIRTerminologyService

from .endpoints import router as fhir_router

__all__ = [
    # Models
    'CodeSystemLookupRequest',
    'CodeSystemLookupResponse',
    'ValueSetExpandRequest',
    'ConceptMapTranslateRequest',
    'ValidateCodeRequest',
    'ValidateCodeResponse',
    'OperationOutcome',
    'Parameters',
    'TerminologyCapabilities',

    # Client
    'FHIRTerminologyClient',
    'QueryRouterTimeoutError',
    'QueryRouterUnavailableError',

    # Service
    'FHIRTerminologyService',

    # Router
    'fhir_router'
]

__version__ = "1.0.0"
__description__ = "FHIR Terminology Service with KB7 Hybrid Architecture"