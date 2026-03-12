"""
Fallback Handler for Multi-KB Query Router
Comprehensive error handling with intelligent fallback chains
Implements circuit breaker pattern and graceful degradation
"""

import asyncio
import time
from typing import Dict, Optional, Any, List
from enum import Enum
from datetime import datetime, timedelta
from loguru import logger

from .multi_kb_query_router import (
    MultiKBQueryRequest,
    MultiKBQueryResponse,
    QueryPattern,
    DataSource
)


class FallbackStrategy(Enum):
    """Available fallback strategies"""
    ALTERNATIVE_SOURCE = "alternative_source"      # Try different data source
    DEGRADED_RESPONSE = "degraded_response"       # Return partial/cached data
    CACHED_RESPONSE = "cached_response"           # Return stale cache if available
    ERROR_RESPONSE = "error_response"             # Return structured error
    RETRY_WITH_BACKOFF = "retry_with_backoff"     # Exponential backoff retry


class FallbackHandler:
    """
    Handles query failures with intelligent fallback strategies

    Fallback chain priorities:
    1. Alternative data source (e.g., Neo4j KB7 → PostgreSQL)
    2. Degraded response (partial data, cached results)
    3. Retry with exponential backoff
    4. Structured error response with diagnostic info
    """

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.max_retries = config.get('max_retries', 3)
        self.base_delay = config.get('base_delay_ms', 1000) / 1000  # Convert to seconds
        self.max_delay = config.get('max_delay_ms', 30000) / 1000

        # Fallback routing rules
        self.fallback_chains = self._initialize_fallback_chains()
        self.circuit_breakers = {}

        # Failure tracking
        self.failure_stats = {}

    def _initialize_fallback_chains(self) -> Dict[DataSource, List[Dict[str, Any]]]:
        """Initialize fallback chains for each data source"""
        return {
            # Neo4j KB7 fallback chain
            DataSource.NEO4J_KB7: [
                {
                    'strategy': FallbackStrategy.ALTERNATIVE_SOURCE,
                    'source': DataSource.POSTGRES,
                    'reason': 'Use PostgreSQL for exact terminology lookups'
                },
                {
                    'strategy': FallbackStrategy.CACHED_RESPONSE,
                    'reason': 'Return stale cache if available'
                },
                {
                    'strategy': FallbackStrategy.DEGRADED_RESPONSE,
                    'reason': 'Return basic terminology without relationships'
                }
            ],

            # ClickHouse fallback chain
            DataSource.CLICKHOUSE_KB3: [
                {
                    'strategy': FallbackStrategy.ALTERNATIVE_SOURCE,
                    'source': DataSource.NEO4J_KB3,
                    'reason': 'Use Neo4j for basic drug calculation lookup'
                },
                {
                    'strategy': FallbackStrategy.CACHED_RESPONSE,
                    'reason': 'Return cached calculation results'
                }
            ],

            DataSource.CLICKHOUSE_KB7: [
                {
                    'strategy': FallbackStrategy.ALTERNATIVE_SOURCE,
                    'source': DataSource.NEO4J_KB7,
                    'reason': 'Use Neo4j for terminology analytics'
                },
                {
                    'strategy': FallbackStrategy.CACHED_RESPONSE,
                    'reason': 'Return cached analytics'
                }
            ],

            # Elasticsearch fallback chain
            DataSource.ELASTICSEARCH: [
                {
                    'strategy': FallbackStrategy.ALTERNATIVE_SOURCE,
                    'source': DataSource.POSTGRES,
                    'reason': 'Use PostgreSQL for exact match search'
                },
                {
                    'strategy': FallbackStrategy.DEGRADED_RESPONSE,
                    'reason': 'Return limited search results'
                }
            ],

            # GraphDB fallback chain
            DataSource.GRAPHDB: [
                {
                    'strategy': FallbackStrategy.ALTERNATIVE_SOURCE,
                    'source': DataSource.NEO4J_KB7,
                    'reason': 'Use Neo4j for basic relationship queries'
                },
                {
                    'strategy': FallbackStrategy.DEGRADED_RESPONSE,
                    'reason': 'Return terminology without semantic inference'
                }
            ],

            # PostgreSQL fallback (rare)
            DataSource.POSTGRES: [
                {
                    'strategy': FallbackStrategy.CACHED_RESPONSE,
                    'reason': 'Return cached data if PostgreSQL fails'
                },
                {
                    'strategy': FallbackStrategy.RETRY_WITH_BACKOFF,
                    'reason': 'Retry PostgreSQL connection'
                }
            ]
        }

    async def handle_error(self, request: MultiKBQueryRequest, error: Exception, start_time: float) -> MultiKBQueryResponse:
        """
        Main error handling method with comprehensive fallback strategies
        """
        logger.warning(f"Handling error for {request.request_id}: {error}")

        # Update failure statistics
        await self._record_failure(request, error)

        # Try fallback strategies in order
        for attempt in range(self.max_retries):
            try:
                # Get primary data source that failed
                failed_source = await self._identify_failed_source(request, error)

                # Try fallback strategies
                fallback_response = await self._try_fallback_strategies(
                    request, failed_source, error, attempt
                )

                if fallback_response:
                    fallback_response.latency_ms = (time.time() - start_time) * 1000
                    return fallback_response

            except Exception as fallback_error:
                logger.warning(f"Fallback attempt {attempt + 1} failed: {fallback_error}")

                # Apply exponential backoff before next attempt
                if attempt < self.max_retries - 1:
                    delay = min(self.base_delay * (2 ** attempt), self.max_delay)
                    await asyncio.sleep(delay)

        # All fallback strategies exhausted, return structured error
        return await self._create_error_response(request, error, start_time)

    async def _identify_failed_source(self, request: MultiKBQueryRequest, error: Exception) -> Optional[DataSource]:
        """Identify which data source failed based on request pattern and error"""
        try:
            # Map request patterns to primary data sources
            pattern_source_map = {
                QueryPattern.KB1_PATIENT_LOOKUP: DataSource.NEO4J_KB1,
                QueryPattern.KB2_GUIDELINE_SEARCH: DataSource.ELASTICSEARCH,
                QueryPattern.KB3_DRUG_CALCULATION: DataSource.CLICKHOUSE_KB3,
                QueryPattern.KB5_INTERACTION_CHECK: DataSource.NEO4J_KB5,
                QueryPattern.KB7_TERMINOLOGY_LOOKUP: DataSource.POSTGRES,
                QueryPattern.KB7_TERMINOLOGY_SEARCH: DataSource.ELASTICSEARCH,
                QueryPattern.KB7_SEMANTIC_INFERENCE: DataSource.GRAPHDB,
            }

            return pattern_source_map.get(request.pattern)

        except Exception as e:
            logger.warning(f"Could not identify failed source: {e}")
            return None

    async def _try_fallback_strategies(
        self,
        request: MultiKBQueryRequest,
        failed_source: Optional[DataSource],
        original_error: Exception,
        attempt: int
    ) -> Optional[MultiKBQueryResponse]:
        """Try fallback strategies for the failed data source"""

        if not failed_source or failed_source not in self.fallback_chains:
            return None

        fallback_chain = self.fallback_chains[failed_source]

        for fallback_option in fallback_chain:
            strategy = fallback_option['strategy']
            reason = fallback_option['reason']

            logger.info(f"Trying fallback strategy {strategy.value}: {reason}")

            try:
                if strategy == FallbackStrategy.ALTERNATIVE_SOURCE:
                    response = await self._try_alternative_source(
                        request, fallback_option['source'], reason
                    )
                    if response:
                        return response

                elif strategy == FallbackStrategy.CACHED_RESPONSE:
                    response = await self._try_cached_response(request, reason)
                    if response:
                        return response

                elif strategy == FallbackStrategy.DEGRADED_RESPONSE:
                    response = await self._create_degraded_response(request, reason)
                    if response:
                        return response

                elif strategy == FallbackStrategy.RETRY_WITH_BACKOFF:
                    if attempt < 2:  # Only retry on first attempts
                        delay = min(self.base_delay * (2 ** attempt), self.max_delay)
                        await asyncio.sleep(delay)
                        # Return None to trigger retry of original query
                        return None

            except Exception as e:
                logger.warning(f"Fallback strategy {strategy.value} failed: {e}")
                continue

        return None

    async def _try_alternative_source(
        self,
        request: MultiKBQueryRequest,
        alternative_source: DataSource,
        reason: str
    ) -> Optional[MultiKBQueryResponse]:
        """Try alternative data source for the same query"""

        try:
            # Check if alternative source is healthy
            if not await self._is_source_healthy(alternative_source):
                logger.warning(f"Alternative source {alternative_source.value} is unhealthy")
                return None

            # Create modified request for alternative source
            alt_request = await self._adapt_request_for_source(request, alternative_source)
            if not alt_request:
                return None

            # Execute query on alternative source
            # Note: This would typically call back to the router's data source methods
            result = await self._execute_on_alternative_source(alternative_source, alt_request)

            return MultiKBQueryResponse(
                data=result,
                sources_used=[alternative_source.value],
                kb_sources=[request.kb_id] if request.kb_id else [],
                latency_ms=0.0,  # Will be set by caller
                cache_status="fallback",
                request_id=request.request_id
            )

        except Exception as e:
            logger.warning(f"Alternative source {alternative_source.value} failed: {e}")
            return None

    async def _try_cached_response(self, request: MultiKBQueryRequest, reason: str) -> Optional[MultiKBQueryResponse]:
        """Try to return cached response, even if stale"""

        try:
            # This would interface with the cache coordinator to get stale data
            # For now, return None as placeholder
            logger.info(f"Attempting cached response: {reason}")
            return None

        except Exception as e:
            logger.warning(f"Cached response fallback failed: {e}")
            return None

    async def _create_degraded_response(self, request: MultiKBQueryRequest, reason: str) -> Optional[MultiKBQueryResponse]:
        """Create degraded response with partial functionality"""

        try:
            degraded_data = {}

            # Pattern-specific degraded responses
            if request.pattern == QueryPattern.KB7_TERMINOLOGY_LOOKUP:
                degraded_data = {
                    'code': request.params.get('code'),
                    'system': request.params.get('system'),
                    'display': 'Term lookup temporarily unavailable',
                    'degraded': True,
                    'reason': reason
                }

            elif request.pattern == QueryPattern.KB5_INTERACTION_CHECK:
                degraded_data = {
                    'drug_codes': request.params.get('drug_codes', []),
                    'interactions': [],
                    'degraded': True,
                    'message': 'Interaction checking temporarily limited',
                    'reason': reason
                }

            elif request.pattern == QueryPattern.CROSS_KB_PATIENT_VIEW:
                degraded_data = {
                    'patient_id': request.params.get('patient_id'),
                    'basic_info': 'Available',
                    'detailed_relationships': 'Temporarily unavailable',
                    'degraded': True,
                    'reason': reason
                }

            else:
                degraded_data = {
                    'message': 'Service temporarily degraded',
                    'degraded': True,
                    'reason': reason
                }

            return MultiKBQueryResponse(
                data=degraded_data,
                sources_used=['degraded'],
                kb_sources=[request.kb_id] if request.kb_id else [],
                latency_ms=0.0,
                cache_status="degraded",
                request_id=request.request_id
            )

        except Exception as e:
            logger.error(f"Failed to create degraded response: {e}")
            return None

    async def _execute_on_alternative_source(self, source: DataSource, request: MultiKBQueryRequest) -> Dict[str, Any]:
        """Execute query on alternative data source - placeholder for actual implementation"""

        # Placeholder implementation
        # In reality, this would call the appropriate data source client
        return {
            'source': source.value,
            'data': 'fallback_result',
            'fallback': True
        }

    async def _adapt_request_for_source(self, request: MultiKBQueryRequest, source: DataSource) -> Optional[MultiKBQueryRequest]:
        """Adapt request parameters for alternative data source"""

        try:
            # Create copy of request
            adapted_request = MultiKBQueryRequest(
                service_id=request.service_id,
                pattern=request.pattern,
                params=request.params.copy(),
                kb_id=request.kb_id,
                cross_kb_scope=request.cross_kb_scope,
                require_snapshot=False,  # Disable snapshot for fallback
                priority="high",  # Prioritize fallback queries
                timeout_ms=request.timeout_ms
            )

            # Source-specific adaptations
            if source == DataSource.POSTGRES and request.pattern == QueryPattern.KB7_TERMINOLOGY_SEARCH:
                # Convert fuzzy search to exact lookup for PostgreSQL
                adapted_request.pattern = QueryPattern.KB7_TERMINOLOGY_LOOKUP
                if 'query' in adapted_request.params:
                    adapted_request.params['code'] = adapted_request.params.pop('query')

            return adapted_request

        except Exception as e:
            logger.warning(f"Failed to adapt request for {source.value}: {e}")
            return None

    async def _is_source_healthy(self, source: DataSource) -> bool:
        """Check if data source is healthy using circuit breaker pattern"""

        source_name = source.value
        breaker = self.circuit_breakers.get(source_name, {
            'state': 'closed',
            'failure_count': 0,
            'last_failure': 0,
            'next_retry': 0
        })

        current_time = time.time()

        if breaker['state'] == 'open':
            # Check if enough time has passed for half-open attempt
            if current_time >= breaker['next_retry']:
                self.circuit_breakers[source_name]['state'] = 'half_open'
                return True
            return False

        elif breaker['state'] == 'half_open':
            # Allow one attempt in half-open state
            return True

        # Circuit is closed, source is healthy
        return True

    async def _record_failure(self, request: MultiKBQueryRequest, error: Exception):
        """Record failure for circuit breaker and statistics"""

        # Update failure statistics
        pattern_key = request.pattern.value
        self.failure_stats[pattern_key] = self.failure_stats.get(pattern_key, 0) + 1

        # Update circuit breaker state
        failed_source = await self._identify_failed_source(request, error)
        if failed_source:
            await self._update_circuit_breaker(failed_source, success=False)

    async def _update_circuit_breaker(self, source: DataSource, success: bool):
        """Update circuit breaker state based on success/failure"""

        source_name = source.value
        breaker = self.circuit_breakers.get(source_name, {
            'state': 'closed',
            'failure_count': 0,
            'last_failure': 0,
            'next_retry': 0
        })

        current_time = time.time()

        if success:
            # Reset on success
            breaker.update({
                'state': 'closed',
                'failure_count': 0,
                'last_failure': 0,
                'next_retry': 0
            })
        else:
            # Increment failure count
            breaker['failure_count'] += 1
            breaker['last_failure'] = current_time

            # Open circuit if threshold exceeded
            failure_threshold = self.config.get('circuit_breaker_threshold', 5)
            if breaker['failure_count'] >= failure_threshold:
                breaker['state'] = 'open'
                retry_delay = self.config.get('circuit_breaker_timeout', 60)
                breaker['next_retry'] = current_time + retry_delay

                logger.warning(
                    f"Circuit breaker opened for {source_name} "
                    f"(failures: {breaker['failure_count']})"
                )

        self.circuit_breakers[source_name] = breaker

    async def _create_error_response(
        self,
        request: MultiKBQueryRequest,
        error: Exception,
        start_time: float
    ) -> MultiKBQueryResponse:
        """Create structured error response when all fallbacks fail"""

        error_data = {
            'error': True,
            'error_type': type(error).__name__,
            'error_message': str(error),
            'request_id': request.request_id,
            'pattern': request.pattern.value,
            'kb_id': request.kb_id,
            'timestamp': datetime.utcnow().isoformat(),
            'fallback_attempted': True,
            'help': 'All fallback strategies exhausted. Check system health.'
        }

        return MultiKBQueryResponse(
            data=error_data,
            sources_used=['error_handler'],
            kb_sources=[request.kb_id] if request.kb_id else [],
            latency_ms=(time.time() - start_time) * 1000,
            cache_status="error",
            request_id=request.request_id
        )

    async def get_health_status(self) -> Dict[str, Any]:
        """Get health status of fallback handler"""
        return {
            'circuit_breakers': self.circuit_breakers,
            'failure_stats': self.failure_stats,
            'fallback_chains': {
                source.value: [
                    {
                        'strategy': fb['strategy'].value,
                        'reason': fb['reason']
                    }
                    for fb in chain
                ]
                for source, chain in self.fallback_chains.items()
            }
        }

    async def reset_circuit_breaker(self, source: DataSource):
        """Manually reset circuit breaker for a data source"""
        source_name = source.value
        if source_name in self.circuit_breakers:
            self.circuit_breakers[source_name] = {
                'state': 'closed',
                'failure_count': 0,
                'last_failure': 0,
                'next_retry': 0
            }
            logger.info(f"Circuit breaker reset for {source_name}")

    async def get_fallback_stats(self) -> Dict[str, Any]:
        """Get fallback usage statistics"""
        return {
            'total_failures': sum(self.failure_stats.values()),
            'failure_by_pattern': self.failure_stats,
            'circuit_breaker_states': {
                source: breaker['state']
                for source, breaker in self.circuit_breakers.items()
            }
        }