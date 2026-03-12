"""
FHIR Terminology Client for KB7 Hybrid Query Router Integration

This module provides a client for integrating FHIR terminology operations
with the Go query router service, implementing the hybrid architecture.
"""

import json
import asyncio
import logging
from typing import Dict, List, Any, Optional, Union
from datetime import datetime, timedelta
import aiohttp
import aioredis
from urllib.parse import urlencode

from .models import (
    CodeSystemLookupRequest, CodeSystemLookupResponse,
    ValueSetExpandRequest, ConceptMapTranslateRequest,
    ValidateCodeRequest, ValidateCodeResponse,
    OperationOutcome, Parameters
)

logger = logging.getLogger(__name__)


class QueryRouterTimeoutError(Exception):
    """Raised when query router request times out"""
    pass


class QueryRouterUnavailableError(Exception):
    """Raised when query router is unavailable"""
    pass


class FHIRTerminologyClient:
    """
    Client for FHIR terminology operations using KB7 hybrid query router.

    This client routes different types of terminology queries to the optimal
    backend (PostgreSQL or GraphDB) through the Go query router service.
    """

    def __init__(
        self,
        query_router_url: str = "http://localhost:8087",
        redis_url: str = "redis://localhost:6379",
        timeout: int = 30,
        max_retries: int = 3,
        cache_ttl: int = 3600  # 1 hour default cache TTL
    ):
        """
        Initialize the FHIR terminology client.

        Args:
            query_router_url: URL of the KB7 query router service
            redis_url: Redis connection URL for caching
            timeout: Request timeout in seconds
            max_retries: Maximum number of retry attempts
            cache_ttl: Default cache TTL in seconds
        """
        self.query_router_url = query_router_url.rstrip('/')
        self.redis_url = redis_url
        self.timeout = timeout
        self.max_retries = max_retries
        self.cache_ttl = cache_ttl

        # Initialize async session and cache
        self._session: Optional[aiohttp.ClientSession] = None
        self._redis: Optional[aioredis.Redis] = None

        # Performance metrics
        self.metrics = {
            'cache_hits': 0,
            'cache_misses': 0,
            'postgresql_queries': 0,
            'graphdb_queries': 0,
            'hybrid_queries': 0,
            'errors': 0,
            'total_queries': 0,
            'average_latency': 0.0
        }

    async def __aenter__(self):
        """Async context manager entry"""
        await self._ensure_connections()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit"""
        await self.close()

    async def _ensure_connections(self):
        """Ensure HTTP session and Redis connection are established"""
        if self._session is None:
            connector = aiohttp.TCPConnector(
                limit=100,
                limit_per_host=20,
                keepalive_timeout=30
            )
            timeout = aiohttp.ClientTimeout(total=self.timeout)
            self._session = aiohttp.ClientSession(
                connector=connector,
                timeout=timeout,
                headers={'Content-Type': 'application/json'}
            )

        if self._redis is None:
            self._redis = aioredis.from_url(
                self.redis_url,
                encoding="utf-8",
                decode_responses=True,
                max_connections=20
            )

    async def close(self):
        """Close connections"""
        if self._session:
            await self._session.close()
            self._session = None

        if self._redis:
            await self._redis.close()
            self._redis = None

    async def _get_cache_key(self, operation: str, params: Dict[str, Any]) -> str:
        """Generate cache key for operation and parameters"""
        # Sort parameters for consistent cache keys
        sorted_params = json.dumps(params, sort_keys=True, default=str)
        cache_key = f"fhir:terminology:{operation}:{hash(sorted_params)}"
        return cache_key

    async def _get_cached_result(self, cache_key: str) -> Optional[Dict[str, Any]]:
        """Get cached result if available"""
        try:
            await self._ensure_connections()
            cached = await self._redis.get(cache_key)
            if cached:
                self.metrics['cache_hits'] += 1
                return json.loads(cached)
            else:
                self.metrics['cache_misses'] += 1
                return None
        except Exception as e:
            logger.warning(f"Cache get error: {e}")
            self.metrics['cache_misses'] += 1
            return None

    async def _set_cache_result(self, cache_key: str, result: Dict[str, Any], ttl: int = None):
        """Cache the result"""
        try:
            await self._ensure_connections()
            cache_ttl = ttl or self.cache_ttl
            await self._redis.setex(
                cache_key,
                cache_ttl,
                json.dumps(result, default=str)
            )
        except Exception as e:
            logger.warning(f"Cache set error: {e}")

    async def _make_request(
        self,
        method: str,
        endpoint: str,
        params: Optional[Dict[str, Any]] = None,
        data: Optional[Dict[str, Any]] = None,
        target_store: str = "auto"
    ) -> Dict[str, Any]:
        """
        Make HTTP request to query router with retry logic.

        Args:
            method: HTTP method (GET, POST)
            endpoint: API endpoint
            params: Query parameters
            data: Request body data
            target_store: Target backend store (postgresql/graphdb/auto)
        """
        await self._ensure_connections()

        url = f"{self.query_router_url}/api/v1{endpoint}"
        headers = {}

        # Add routing hint if specified
        if target_store != "auto":
            headers['X-Target-Store'] = target_store

        self.metrics['total_queries'] += 1
        start_time = datetime.utcnow()

        for attempt in range(self.max_retries):
            try:
                if method.upper() == "GET":
                    async with self._session.get(url, params=params, headers=headers) as response:
                        result = await self._handle_response(response)
                        self._update_latency_metrics(start_time)
                        return result

                elif method.upper() == "POST":
                    async with self._session.post(url, json=data, params=params, headers=headers) as response:
                        result = await self._handle_response(response)
                        self._update_latency_metrics(start_time)
                        return result

            except (aiohttp.ClientError, asyncio.TimeoutError) as e:
                self.metrics['errors'] += 1
                logger.warning(f"Request attempt {attempt + 1} failed: {e}")

                if attempt == self.max_retries - 1:
                    if isinstance(e, asyncio.TimeoutError):
                        raise QueryRouterTimeoutError(f"Query router timeout after {self.max_retries} attempts")
                    else:
                        raise QueryRouterUnavailableError(f"Query router unavailable after {self.max_retries} attempts")

                # Exponential backoff
                await asyncio.sleep(2 ** attempt)

        raise QueryRouterUnavailableError("Unexpected error in request retry logic")

    async def _handle_response(self, response: aiohttp.ClientResponse) -> Dict[str, Any]:
        """Handle HTTP response and error cases"""
        if response.status == 200:
            return await response.json()

        elif response.status == 404:
            # Return FHIR OperationOutcome for not found
            return {
                "resourceType": "OperationOutcome",
                "issue": [{
                    "severity": "error",
                    "code": "not-found",
                    "diagnostics": "Resource not found"
                }]
            }

        elif response.status >= 500:
            # Server error
            error_text = await response.text()
            logger.error(f"Query router server error {response.status}: {error_text}")
            return {
                "resourceType": "OperationOutcome",
                "issue": [{
                    "severity": "error",
                    "code": "exception",
                    "diagnostics": f"Server error: {response.status}"
                }]
            }

        else:
            # Client error
            error_text = await response.text()
            logger.warning(f"Query router client error {response.status}: {error_text}")
            return {
                "resourceType": "OperationOutcome",
                "issue": [{
                    "severity": "error",
                    "code": "invalid",
                    "diagnostics": f"Client error: {response.status}"
                }]
            }

    def _update_latency_metrics(self, start_time: datetime):
        """Update latency metrics"""
        latency = (datetime.utcnow() - start_time).total_seconds() * 1000  # ms

        # Simple moving average
        if self.metrics['average_latency'] == 0:
            self.metrics['average_latency'] = latency
        else:
            self.metrics['average_latency'] = (self.metrics['average_latency'] * 0.9) + (latency * 0.1)

    # FHIR Terminology Operations

    async def lookup_code(
        self,
        system: str,
        code: str,
        version: Optional[str] = None,
        properties: Optional[List[str]] = None,
        display_language: Optional[str] = None,
        use_cache: bool = True
    ) -> Dict[str, Any]:
        """
        Perform CodeSystem $lookup operation.
        Routes to PostgreSQL for fast exact lookups.

        Args:
            system: Code system URI
            code: Code to lookup
            version: Specific version of code system
            properties: Requested properties
            display_language: Language for display
            use_cache: Whether to use caching

        Returns:
            FHIR Parameters resource with lookup results
        """
        params = {
            'system': system,
            'code': code
        }
        if version:
            params['version'] = version
        if properties:
            params['property'] = properties
        if display_language:
            params['displayLanguage'] = display_language

        # Check cache first
        cache_key = await self._get_cache_key('lookup', params)
        if use_cache:
            cached = await self._get_cached_result(cache_key)
            if cached:
                return cached

        # Route to PostgreSQL for exact lookup
        try:
            result = await self._make_request(
                "GET",
                f"/concepts/{system}/{code}",
                target_store="postgresql"
            )

            self.metrics['postgresql_queries'] += 1

            # Transform to FHIR Parameters format
            fhir_result = self._transform_concept_to_lookup_response(result)

            # Cache successful results
            if use_cache and fhir_result.get('resourceType') == 'Parameters':
                await self._set_cache_result(cache_key, fhir_result, ttl=3600)

            return fhir_result

        except Exception as e:
            logger.error(f"Code lookup failed: {e}")
            return self._create_error_outcome("exception", f"Lookup failed: {str(e)}")

    async def expand_valueset(
        self,
        url: Optional[str] = None,
        valueset: Optional[Dict[str, Any]] = None,
        filter_text: Optional[str] = None,
        count: Optional[int] = None,
        offset: Optional[int] = None,
        include_definition: bool = False,
        use_cache: bool = True
    ) -> Dict[str, Any]:
        """
        Perform ValueSet $expand operation.
        Routes to GraphDB for semantic expansion.

        Args:
            url: ValueSet canonical URL
            valueset: ValueSet resource
            filter_text: Text filter for expansion
            count: Maximum number of concepts to return
            offset: Starting position for pagination
            include_definition: Include definition in expansion
            use_cache: Whether to use caching

        Returns:
            FHIR ValueSet resource with expansion
        """
        params = {}
        if url:
            params['url'] = url
        if filter_text:
            params['filter'] = filter_text
        if count:
            params['count'] = count
        if offset:
            params['offset'] = offset
        if include_definition:
            params['includeDefinition'] = include_definition

        # Check cache first
        cache_key = await self._get_cache_key('expand', params)
        if use_cache:
            cached = await self._get_cached_result(cache_key)
            if cached:
                return cached

        try:
            # Route to GraphDB for semantic expansion
            # This would typically involve complex subsumption queries
            result = await self._make_request(
                "POST",
                "/valueset/expand",
                data={'valueset': valueset, 'parameters': params},
                target_store="graphdb"
            )

            self.metrics['graphdb_queries'] += 1

            # Cache successful results (shorter TTL for expansions)
            if use_cache and result.get('resourceType') == 'ValueSet':
                await self._set_cache_result(cache_key, result, ttl=1800)  # 30 minutes

            return result

        except Exception as e:
            logger.error(f"ValueSet expansion failed: {e}")
            return self._create_error_outcome("exception", f"Expansion failed: {str(e)}")

    async def translate_concept(
        self,
        source_system: str,
        source_code: str,
        target_system: str,
        concept_map_url: Optional[str] = None,
        reverse: bool = False,
        use_cache: bool = True
    ) -> Dict[str, Any]:
        """
        Perform ConceptMap $translate operation.
        Routes to PostgreSQL for mapping lookups.

        Args:
            source_system: Source terminology system
            source_code: Source code to translate
            target_system: Target terminology system
            concept_map_url: Specific ConceptMap to use
            reverse: Reverse translation
            use_cache: Whether to use caching

        Returns:
            FHIR Parameters resource with translation results
        """
        params = {
            'source_system': source_system,
            'source_code': source_code,
            'target_system': target_system
        }
        if concept_map_url:
            params['concept_map_url'] = concept_map_url
        if reverse:
            params['reverse'] = reverse

        # Check cache first
        cache_key = await self._get_cache_key('translate', params)
        if use_cache:
            cached = await self._get_cached_result(cache_key)
            if cached:
                return cached

        try:
            # Route to PostgreSQL for mapping lookup
            result = await self._make_request(
                "GET",
                f"/mappings/{source_system}/{source_code}/{target_system}",
                target_store="postgresql"
            )

            self.metrics['postgresql_queries'] += 1

            # Transform to FHIR Parameters format
            fhir_result = self._transform_mapping_to_translate_response(result)

            # Cache successful results
            if use_cache and fhir_result.get('resourceType') == 'Parameters':
                await self._set_cache_result(cache_key, fhir_result, ttl=7200)  # 2 hours

            return fhir_result

        except Exception as e:
            logger.error(f"Concept translation failed: {e}")
            return self._create_error_outcome("exception", f"Translation failed: {str(e)}")

    async def validate_code(
        self,
        system: str,
        code: str,
        display: Optional[str] = None,
        valueset_url: Optional[str] = None,
        use_cache: bool = True
    ) -> Dict[str, Any]:
        """
        Perform $validate-code operation.
        Uses hybrid routing based on validation type.

        Args:
            system: Code system URI
            code: Code to validate
            display: Display text to validate
            valueset_url: ValueSet URL for validation context
            use_cache: Whether to use caching

        Returns:
            FHIR Parameters resource with validation results
        """
        params = {
            'system': system,
            'code': code
        }
        if display:
            params['display'] = display
        if valueset_url:
            params['valueset_url'] = valueset_url

        # Check cache first
        cache_key = await self._get_cache_key('validate', params)
        if use_cache:
            cached = await self._get_cached_result(cache_key)
            if cached:
                return cached

        try:
            # Simple validation goes to PostgreSQL, ValueSet validation to GraphDB
            target_store = "graphdb" if valueset_url else "postgresql"

            result = await self._make_request(
                "GET",
                f"/concepts/{system}/{code}",
                params={'validate': True, 'display': display, 'valueset': valueset_url},
                target_store=target_store
            )

            if target_store == "postgresql":
                self.metrics['postgresql_queries'] += 1
            else:
                self.metrics['graphdb_queries'] += 1

            # Transform to FHIR Parameters format
            fhir_result = self._transform_validation_response(result, display)

            # Cache successful results
            if use_cache and fhir_result.get('resourceType') == 'Parameters':
                await self._set_cache_result(cache_key, fhir_result, ttl=1800)  # 30 minutes

            return fhir_result

        except Exception as e:
            logger.error(f"Code validation failed: {e}")
            return self._create_error_outcome("exception", f"Validation failed: {str(e)}")

    async def search_concepts(
        self,
        query: str,
        system: Optional[str] = None,
        limit: int = 20,
        use_cache: bool = True
    ) -> Dict[str, Any]:
        """
        Search concepts using text search.
        Routes to PostgreSQL for full-text search.

        Args:
            query: Search query text
            system: Limit to specific system
            limit: Maximum results
            use_cache: Whether to use caching

        Returns:
            FHIR Bundle with search results
        """
        params = {
            'q': query,
            'limit': limit
        }
        if system:
            params['system'] = system

        # Check cache first
        cache_key = await self._get_cache_key('search', params)
        if use_cache:
            cached = await self._get_cached_result(cache_key)
            if cached:
                return cached

        try:
            # Route to PostgreSQL for text search
            result = await self._make_request(
                "GET",
                "/search",
                params=params,
                target_store="postgresql"
            )

            self.metrics['postgresql_queries'] += 1

            # Transform to FHIR Bundle format
            fhir_result = self._transform_search_to_bundle(result, query)

            # Cache successful results (shorter TTL for searches)
            if use_cache and fhir_result.get('resourceType') == 'Bundle':
                await self._set_cache_result(cache_key, fhir_result, ttl=900)  # 15 minutes

            return fhir_result

        except Exception as e:
            logger.error(f"Concept search failed: {e}")
            return self._create_error_outcome("exception", f"Search failed: {str(e)}")

    # Helper methods for FHIR transformations

    def _transform_concept_to_lookup_response(self, concept_data: Dict[str, Any]) -> Dict[str, Any]:
        """Transform concept data to FHIR $lookup response"""
        if concept_data.get('resourceType') == 'OperationOutcome':
            return concept_data

        parameters = []

        if 'display' in concept_data:
            parameters.append({
                "name": "display",
                "valueString": concept_data['display']
            })

        if 'definition' in concept_data:
            parameters.append({
                "name": "definition",
                "valueString": concept_data['definition']
            })

        if 'system' in concept_data:
            parameters.append({
                "name": "system",
                "valueUri": concept_data['system']
            })

        if 'version' in concept_data:
            parameters.append({
                "name": "version",
                "valueString": concept_data['version']
            })

        # Add properties if available
        if 'properties' in concept_data:
            for prop in concept_data['properties']:
                parameters.append({
                    "name": "property",
                    "part": [
                        {"name": "code", "valueCode": prop.get('code')},
                        {"name": "value", "valueString": prop.get('value')}
                    ]
                })

        return {
            "resourceType": "Parameters",
            "parameter": parameters
        }

    def _transform_mapping_to_translate_response(self, mapping_data: Dict[str, Any]) -> Dict[str, Any]:
        """Transform mapping data to FHIR $translate response"""
        if mapping_data.get('resourceType') == 'OperationOutcome':
            return mapping_data

        parameters = [
            {
                "name": "result",
                "valueBoolean": True
            }
        ]

        if 'target_code' in mapping_data:
            parameters.append({
                "name": "match",
                "part": [
                    {"name": "equivalence", "valueCode": mapping_data.get('equivalence', 'equivalent')},
                    {"name": "concept", "valueCoding": {
                        "system": mapping_data.get('target_system'),
                        "code": mapping_data['target_code'],
                        "display": mapping_data.get('target_display')
                    }}
                ]
            })

        return {
            "resourceType": "Parameters",
            "parameter": parameters
        }

    def _transform_validation_response(self, validation_data: Dict[str, Any], expected_display: Optional[str]) -> Dict[str, Any]:
        """Transform validation data to FHIR $validate-code response"""
        if validation_data.get('resourceType') == 'OperationOutcome':
            # Invalid code
            return {
                "resourceType": "Parameters",
                "parameter": [
                    {"name": "result", "valueBoolean": False},
                    {"name": "message", "valueString": "Code not found"}
                ]
            }

        parameters = [
            {"name": "result", "valueBoolean": True}
        ]

        if 'display' in validation_data:
            actual_display = validation_data['display']
            parameters.append({
                "name": "display",
                "valueString": actual_display
            })

            # Check display match if provided
            if expected_display and expected_display != actual_display:
                parameters.append({
                    "name": "message",
                    "valueString": f"Display text mismatch. Expected: {expected_display}, Actual: {actual_display}"
                })

        return {
            "resourceType": "Parameters",
            "parameter": parameters
        }

    def _transform_search_to_bundle(self, search_data: Dict[str, Any], query: str) -> Dict[str, Any]:
        """Transform search results to FHIR Bundle"""
        if not isinstance(search_data, list):
            search_data = []

        entries = []
        for concept in search_data:
            entries.append({
                "resource": {
                    "resourceType": "CodeSystem",
                    "id": f"{concept.get('system', 'unknown')}-{concept.get('code', 'unknown')}",
                    "url": concept.get('system'),
                    "concept": [{
                        "code": concept.get('code'),
                        "display": concept.get('display'),
                        "definition": concept.get('definition')
                    }]
                },
                "search": {
                    "mode": "match"
                }
            })

        return {
            "resourceType": "Bundle",
            "id": f"search-{hash(query)}",
            "type": "searchset",
            "total": len(entries),
            "entry": entries
        }

    def _create_error_outcome(self, code: str, message: str) -> Dict[str, Any]:
        """Create FHIR OperationOutcome for errors"""
        return {
            "resourceType": "OperationOutcome",
            "issue": [{
                "severity": "error",
                "code": code,
                "diagnostics": message
            }]
        }

    async def get_metrics(self) -> Dict[str, Any]:
        """Get client performance metrics"""
        return {
            **self.metrics,
            'cache_hit_rate': (
                self.metrics['cache_hits'] /
                (self.metrics['cache_hits'] + self.metrics['cache_misses'])
                if (self.metrics['cache_hits'] + self.metrics['cache_misses']) > 0 else 0
            )
        }

    async def health_check(self) -> Dict[str, Any]:
        """Perform health check on query router"""
        try:
            await self._ensure_connections()

            # Check query router health
            async with self._session.get(f"{self.query_router_url}/health") as response:
                if response.status == 200:
                    router_health = await response.json()

                    # Check Redis connection
                    redis_healthy = True
                    try:
                        await self._redis.ping()
                    except Exception:
                        redis_healthy = False

                    return {
                        "healthy": router_health.get('healthy', False) and redis_healthy,
                        "query_router": router_health,
                        "redis": {"healthy": redis_healthy},
                        "metrics": await self.get_metrics()
                    }
                else:
                    return {
                        "healthy": False,
                        "error": f"Query router returned {response.status}"
                    }

        except Exception as e:
            return {
                "healthy": False,
                "error": str(e)
            }