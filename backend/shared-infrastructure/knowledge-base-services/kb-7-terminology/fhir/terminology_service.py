"""
FHIR Terminology Service for KB7 Hybrid Architecture

This module provides the main FHIR terminology service that integrates with
the hybrid query router to provide optimal performance for different types
of terminology operations.
"""

import asyncio
import logging
import time
from typing import Dict, List, Any, Optional, Union
from datetime import datetime, timedelta
from fastapi import HTTPException
import json

from .client import FHIRTerminologyClient, QueryRouterTimeoutError, QueryRouterUnavailableError
from .models import (
    CodeSystemLookupRequest, CodeSystemLookupResponse,
    ValueSetExpandRequest, ConceptMapTranslateRequest,
    ValidateCodeRequest, ValidateCodeResponse,
    OperationOutcome, Parameters, TerminologyCapabilities,
    FHIRBundle
)

logger = logging.getLogger(__name__)


class FHIRTerminologyService:
    """
    Main FHIR terminology service integrating with KB7 hybrid query router.

    This service implements FHIR R4 terminology operations with intelligent
    routing to PostgreSQL (fast lookups/mappings) and GraphDB (semantic reasoning).
    """

    def __init__(
        self,
        query_router_url: str = "http://localhost:8087",
        redis_url: str = "redis://localhost:6379",
        enable_caching: bool = True,
        performance_target_ms: int = 200,
        fallback_enabled: bool = True
    ):
        """
        Initialize the FHIR terminology service.

        Args:
            query_router_url: URL of the KB7 query router
            redis_url: Redis connection URL for caching
            enable_caching: Enable/disable caching
            performance_target_ms: Target response time in milliseconds
            fallback_enabled: Enable fallback mechanisms
        """
        self.query_router_url = query_router_url
        self.redis_url = redis_url
        self.enable_caching = enable_caching
        self.performance_target_ms = performance_target_ms
        self.fallback_enabled = fallback_enabled

        # Initialize client
        self.client = FHIRTerminologyClient(
            query_router_url=query_router_url,
            redis_url=redis_url,
            timeout=30,
            max_retries=3,
            cache_ttl=3600
        )

        # Performance monitoring
        self.performance_metrics = {
            'total_operations': 0,
            'operations_under_target': 0,
            'average_response_time_ms': 0.0,
            'error_rate': 0.0,
            'cache_hit_rate': 0.0,
            'postgresql_operations': 0,
            'graphdb_operations': 0,
            'hybrid_operations': 0,
            'fallback_activations': 0
        }

        # Circuit breaker state
        self.circuit_breaker = {
            'postgresql': {'failures': 0, 'last_failure': None, 'open': False},
            'graphdb': {'failures': 0, 'last_failure': None, 'open': False}
        }

    async def __aenter__(self):
        """Async context manager entry"""
        await self.client.__aenter__()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit"""
        await self.client.__aexit__(exc_type, exc_val, exc_tb)

    def _start_performance_tracking(self) -> float:
        """Start performance tracking for an operation"""
        return time.time()

    def _end_performance_tracking(self, start_time: float, operation_type: str):
        """End performance tracking and update metrics"""
        duration_ms = (time.time() - start_time) * 1000

        self.performance_metrics['total_operations'] += 1

        if duration_ms <= self.performance_target_ms:
            self.performance_metrics['operations_under_target'] += 1

        # Update average response time (simple moving average)
        current_avg = self.performance_metrics['average_response_time_ms']
        total_ops = self.performance_metrics['total_operations']
        self.performance_metrics['average_response_time_ms'] = (
            (current_avg * (total_ops - 1) + duration_ms) / total_ops
        )

        # Track operation type
        if operation_type in ['postgresql', 'graphdb', 'hybrid']:
            self.performance_metrics[f'{operation_type}_operations'] += 1

        logger.info(f"Operation completed in {duration_ms:.2f}ms (target: {self.performance_target_ms}ms)")

    def _check_circuit_breaker(self, store: str) -> bool:
        """Check if circuit breaker is open for a store"""
        cb = self.circuit_breaker.get(store, {})

        if cb.get('open', False):
            # Check if we should try to close it (after 60 seconds)
            if cb.get('last_failure') and (
                datetime.now() - cb['last_failure']
            ).total_seconds() > 60:
                cb['open'] = False
                cb['failures'] = 0
                logger.info(f"Circuit breaker for {store} reset")
                return False
            return True

        return False

    def _record_failure(self, store: str):
        """Record a failure for circuit breaker logic"""
        cb = self.circuit_breaker.setdefault(store, {'failures': 0, 'last_failure': None, 'open': False})
        cb['failures'] += 1
        cb['last_failure'] = datetime.now()

        # Open circuit breaker after 5 failures
        if cb['failures'] >= 5:
            cb['open'] = True
            logger.warning(f"Circuit breaker opened for {store} after {cb['failures']} failures")

    async def _handle_operation_error(self, error: Exception, operation: str, fallback_fn=None):
        """Handle operation errors with fallback logic"""
        self.performance_metrics['fallback_activations'] += 1

        if isinstance(error, QueryRouterTimeoutError):
            logger.warning(f"{operation} timed out")
            if self.fallback_enabled and fallback_fn:
                return await fallback_fn()

        elif isinstance(error, QueryRouterUnavailableError):
            logger.error(f"{operation} failed - query router unavailable")
            if self.fallback_enabled and fallback_fn:
                return await fallback_fn()

        # Create FHIR OperationOutcome for error
        return {
            "resourceType": "OperationOutcome",
            "issue": [{
                "severity": "error",
                "code": "exception",
                "diagnostics": f"{operation} failed: {str(error)}"
            }]
        }

    # FHIR Terminology Operations

    async def lookup_code_system_concept(
        self,
        system: str,
        code: str,
        version: Optional[str] = None,
        properties: Optional[List[str]] = None,
        display_language: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        CodeSystem $lookup operation.

        Fast exact code lookup routed to PostgreSQL for optimal performance.
        Target: <10ms response time for 95% of requests.

        Args:
            system: Code system URI (e.g., "http://snomed.info/sct")
            code: Code to lookup
            version: Specific version of code system
            properties: Requested properties to include
            display_language: Language for display text

        Returns:
            FHIR Parameters resource with concept details
        """
        start_time = self._start_performance_tracking()

        try:
            # Check circuit breaker
            if self._check_circuit_breaker('postgresql'):
                return await self._handle_operation_error(
                    Exception("PostgreSQL circuit breaker open"),
                    "CodeSystem lookup"
                )

            result = await self.client.lookup_code(
                system=system,
                code=code,
                version=version,
                properties=properties,
                display_language=display_language,
                use_cache=self.enable_caching
            )

            self._end_performance_tracking(start_time, 'postgresql')
            return result

        except Exception as e:
            self._record_failure('postgresql')
            return await self._handle_operation_error(e, "CodeSystem lookup")

    async def expand_value_set(
        self,
        url: Optional[str] = None,
        value_set: Optional[Dict[str, Any]] = None,
        value_set_version: Optional[str] = None,
        filter_text: Optional[str] = None,
        count: Optional[int] = None,
        offset: Optional[int] = None,
        include_definition: bool = False,
        include_designation: bool = False,
        active_only: bool = True,
        display_language: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        ValueSet $expand operation.

        Semantic expansion routed to GraphDB for subsumption reasoning.
        Target: <50ms response time for 95% of requests.

        Args:
            url: ValueSet canonical URL
            value_set: ValueSet resource to expand
            value_set_version: Specific version
            filter_text: Text filter for concepts
            count: Maximum concepts to return
            offset: Pagination offset
            include_definition: Include concept definitions
            include_designation: Include designations
            active_only: Only active concepts
            display_language: Language for display

        Returns:
            FHIR ValueSet resource with expansion
        """
        start_time = self._start_performance_tracking()

        try:
            # Check circuit breaker
            if self._check_circuit_breaker('graphdb'):
                # Fallback: try simplified expansion from PostgreSQL
                return await self._expand_valueset_fallback(url, filter_text, count, offset)

            result = await self.client.expand_valueset(
                url=url,
                valueset=value_set,
                filter_text=filter_text,
                count=count,
                offset=offset,
                include_definition=include_definition,
                use_cache=self.enable_caching
            )

            self._end_performance_tracking(start_time, 'graphdb')
            return result

        except Exception as e:
            self._record_failure('graphdb')
            return await self._handle_operation_error(
                e,
                "ValueSet expansion",
                lambda: self._expand_valueset_fallback(url, filter_text, count, offset)
            )

    async def translate_concept_map(
        self,
        source_system: str,
        source_code: str,
        target_system: str,
        concept_map_url: Optional[str] = None,
        concept_map_version: Optional[str] = None,
        reverse: bool = False
    ) -> Dict[str, Any]:
        """
        ConceptMap $translate operation.

        Cross-terminology mapping routed to PostgreSQL for fast lookup.
        Target: <15ms response time for 95% of requests.

        Args:
            source_system: Source terminology system
            source_code: Code to translate
            target_system: Target terminology system
            concept_map_url: Specific ConceptMap URL
            concept_map_version: ConceptMap version
            reverse: Reverse translation direction

        Returns:
            FHIR Parameters resource with translation results
        """
        start_time = self._start_performance_tracking()

        try:
            # Check circuit breaker
            if self._check_circuit_breaker('postgresql'):
                return await self._handle_operation_error(
                    Exception("PostgreSQL circuit breaker open"),
                    "ConceptMap translation"
                )

            result = await self.client.translate_concept(
                source_system=source_system,
                source_code=source_code,
                target_system=target_system,
                concept_map_url=concept_map_url,
                reverse=reverse,
                use_cache=self.enable_caching
            )

            self._end_performance_tracking(start_time, 'postgresql')
            return result

        except Exception as e:
            self._record_failure('postgresql')
            return await self._handle_operation_error(e, "ConceptMap translation")

    async def validate_code(
        self,
        system: Optional[str] = None,
        code: Optional[str] = None,
        display: Optional[str] = None,
        value_set_url: Optional[str] = None,
        value_set: Optional[Dict[str, Any]] = None,
        coding: Optional[Dict[str, Any]] = None,
        codeable_concept: Optional[Dict[str, Any]] = None,
        date: Optional[datetime] = None,
        abstract: Optional[bool] = None
    ) -> Dict[str, Any]:
        """
        Terminology $validate-code operation.

        Hybrid routing: simple validation to PostgreSQL,
        ValueSet validation to GraphDB.
        Target: <25ms response time for 95% of requests.

        Args:
            system: Code system URI
            code: Code to validate
            display: Display text to validate
            value_set_url: ValueSet URL for context
            value_set: ValueSet resource for context
            coding: Coding to validate
            codeable_concept: CodeableConcept to validate
            date: Validation date
            abstract: Allow abstract codes

        Returns:
            FHIR Parameters resource with validation results
        """
        start_time = self._start_performance_tracking()

        try:
            # Extract parameters from complex types if provided
            if coding and not (system and code):
                system = coding.get('system')
                code = coding.get('code')
                display = display or coding.get('display')

            if codeable_concept and not (system and code):
                # Use first coding in CodeableConcept
                codings = codeable_concept.get('coding', [])
                if codings:
                    first_coding = codings[0]
                    system = first_coding.get('system')
                    code = first_coding.get('code')
                    display = display or first_coding.get('display')

            if not (system and code):
                return {
                    "resourceType": "Parameters",
                    "parameter": [
                        {"name": "result", "valueBoolean": False},
                        {"name": "message", "valueString": "System and code are required"}
                    ]
                }

            # Determine routing strategy
            is_valueset_validation = bool(value_set_url or value_set)
            target_store = 'graphdb' if is_valueset_validation else 'postgresql'

            # Check circuit breaker
            if self._check_circuit_breaker(target_store):
                return await self._validate_code_fallback(system, code, display)

            result = await self.client.validate_code(
                system=system,
                code=code,
                display=display,
                valueset_url=value_set_url,
                use_cache=self.enable_caching
            )

            self._end_performance_tracking(start_time, target_store)
            return result

        except Exception as e:
            self._record_failure(target_store if 'target_store' in locals() else 'postgresql')
            return await self._handle_operation_error(
                e,
                "Code validation",
                lambda: self._validate_code_fallback(system, code, display)
            )

    async def search_terminology(
        self,
        query: str,
        system: Optional[str] = None,
        count: int = 20,
        offset: int = 0,
        include_abstract: bool = False,
        active_only: bool = True,
        display_language: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Terminology text search operation.

        Full-text search routed to PostgreSQL for performance.
        Target: <50ms response time for 95% of requests.

        Args:
            query: Search query text
            system: Limit to specific terminology system
            count: Maximum results to return
            offset: Pagination offset
            include_abstract: Include abstract concepts
            active_only: Only active concepts
            display_language: Language for results

        Returns:
            FHIR Bundle with search results
        """
        start_time = self._start_performance_tracking()

        try:
            # Check circuit breaker
            if self._check_circuit_breaker('postgresql'):
                return await self._search_terminology_fallback(query, count)

            result = await self.client.search_concepts(
                query=query,
                system=system,
                limit=count,
                use_cache=self.enable_caching
            )

            self._end_performance_tracking(start_time, 'postgresql')
            return result

        except Exception as e:
            self._record_failure('postgresql')
            return await self._handle_operation_error(
                e,
                "Terminology search",
                lambda: self._search_terminology_fallback(query, count)
            )

    async def get_terminology_capabilities(self) -> Dict[str, Any]:
        """
        Get TerminologyCapabilities resource describing service capabilities.

        Returns:
            FHIR TerminologyCapabilities resource
        """
        return {
            "resourceType": "TerminologyCapabilities",
            "id": "kb7-terminology-service",
            "url": "http://cardiofit.com/fhir/TerminologyCapabilities/kb7-terminology-service",
            "version": "1.0.0",
            "name": "KB7TerminologyService",
            "title": "KB7 Clinical Terminology Service",
            "status": "active",
            "experimental": False,
            "date": datetime.now().isoformat(),
            "publisher": "CardioFit Clinical Synthesis Hub",
            "description": "FHIR terminology service with hybrid PostgreSQL/GraphDB architecture for optimal performance",
            "purpose": "Provide high-performance terminology operations for clinical applications",
            "kind": "instance",
            "software": {
                "name": "KB7 Terminology Service",
                "version": "1.0.0"
            },
            "implementation": {
                "description": "Hybrid architecture with PostgreSQL and GraphDB backends",
                "url": self.query_router_url
            },
            "lockedDate": False,
            "codeSystem": [
                {
                    "uri": "http://snomed.info/sct",
                    "version": [{"code": "http://snomed.info/sct/900000000000207008/version/20250701"}]
                },
                {
                    "uri": "http://loinc.org",
                    "version": [{"code": "2.76"}]
                },
                {
                    "uri": "http://www.nlm.nih.gov/research/umls/rxnorm",
                    "version": [{"code": "20250101"}]
                }
            ],
            "expansion": {
                "hierarchical": True,
                "paging": True,
                "incomplete": True,
                "parameter": [
                    {"name": "count", "documentation": "Maximum number of concepts to return"},
                    {"name": "offset", "documentation": "Starting position for pagination"},
                    {"name": "filter", "documentation": "Text filter for concept selection"},
                    {"name": "includeDefinition", "documentation": "Include concept definitions"},
                    {"name": "displayLanguage", "documentation": "Language for display text"}
                ]
            },
            "codeSearch": "explicit",
            "validateCode": {
                "translations": True
            },
            "translation": {
                "needsMap": False
            }
        }

    # Fallback methods

    async def _expand_valueset_fallback(
        self,
        url: Optional[str],
        filter_text: Optional[str],
        count: Optional[int],
        offset: Optional[int]
    ) -> Dict[str, Any]:
        """Fallback ValueSet expansion using PostgreSQL only"""
        logger.warning("Using fallback ValueSet expansion (PostgreSQL only)")

        # Simple expansion without full semantic reasoning
        # This would query PostgreSQL for concepts matching the filter
        return {
            "resourceType": "ValueSet",
            "id": "fallback-expansion",
            "expansion": {
                "identifier": f"fallback-{int(time.time())}",
                "timestamp": datetime.now().isoformat(),
                "total": 0,
                "contains": []
            },
            "text": {
                "status": "additional",
                "div": "<div>Fallback expansion - limited functionality available</div>"
            }
        }

    async def _validate_code_fallback(
        self,
        system: str,
        code: str,
        display: Optional[str]
    ) -> Dict[str, Any]:
        """Fallback code validation"""
        logger.warning("Using fallback code validation")

        return {
            "resourceType": "Parameters",
            "parameter": [
                {"name": "result", "valueBoolean": False},
                {"name": "message", "valueString": "Validation service temporarily unavailable"}
            ]
        }

    async def _search_terminology_fallback(self, query: str, count: int) -> Dict[str, Any]:
        """Fallback terminology search"""
        logger.warning("Using fallback terminology search")

        return {
            "resourceType": "Bundle",
            "id": "fallback-search",
            "type": "searchset",
            "total": 0,
            "entry": []
        }

    # Monitoring and metrics

    async def get_performance_metrics(self) -> Dict[str, Any]:
        """Get service performance metrics"""
        client_metrics = await self.client.get_metrics()

        # Calculate success rate
        total_ops = self.performance_metrics['total_operations']
        success_rate = 1.0 - self.performance_metrics['error_rate'] if total_ops > 0 else 0.0

        # Calculate target achievement rate
        target_achievement_rate = (
            self.performance_metrics['operations_under_target'] / total_ops
            if total_ops > 0 else 0.0
        )

        return {
            **self.performance_metrics,
            'client_metrics': client_metrics,
            'success_rate': success_rate,
            'target_achievement_rate': target_achievement_rate,
            'circuit_breaker_status': self.circuit_breaker,
            'performance_target_ms': self.performance_target_ms
        }

    async def health_check(self) -> Dict[str, Any]:
        """Comprehensive health check"""
        try:
            # Check client health
            client_health = await self.client.health_check()

            # Check if we're meeting performance targets
            metrics = await self.get_performance_metrics()
            performance_ok = (
                metrics['target_achievement_rate'] >= 0.95 and
                metrics['success_rate'] >= 0.99
            )

            # Check circuit breakers
            circuits_ok = not any(
                cb.get('open', False) for cb in self.circuit_breaker.values()
            )

            overall_healthy = (
                client_health.get('healthy', False) and
                performance_ok and
                circuits_ok
            )

            return {
                "healthy": overall_healthy,
                "timestamp": datetime.now().isoformat(),
                "components": {
                    "query_router": client_health,
                    "performance": {
                        "healthy": performance_ok,
                        "target_achievement_rate": metrics['target_achievement_rate'],
                        "success_rate": metrics['success_rate']
                    },
                    "circuit_breakers": {
                        "healthy": circuits_ok,
                        "status": self.circuit_breaker
                    }
                },
                "metrics": metrics
            }

        except Exception as e:
            return {
                "healthy": False,
                "timestamp": datetime.now().isoformat(),
                "error": str(e)
            }