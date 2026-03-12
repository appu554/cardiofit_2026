"""
Cache Coordinator for Multi-KB Query Router
Manages L2 (proactively warmed) and L3 (router-cached) cache layers
Coordinates with Cache Prefetcher for optimal performance
"""

import asyncio
import json
import hashlib
from typing import Dict, Optional, Any, List
from datetime import datetime, timedelta
from loguru import logger

from .multi_kb_query_router import MultiKBQueryRequest, MultiKBQueryResponse


class CacheCoordinator:
    """
    Coordinates caching across L2 (proactive) and L3 (reactive) cache layers

    L2 Cache (Redis 6379):
    - Proactively warmed by Cache Prefetcher based on Kafka events
    - Contains specific data items workflows are likely to request
    - 5 minute TTL for frequently accessed data

    L3 Cache (Redis 6380):
    - Contains router-orchestrated cross-KB query results
    - 1 hour TTL for complex analytical queries
    - Reduces latency for repeated complex operations
    """

    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self._l2_client = None  # Proactive cache (6379)
        self._l3_client = None  # Reactive cache (6380)

        # Cache statistics
        self.stats = {
            'l2_hits': 0,
            'l2_misses': 0,
            'l3_hits': 0,
            'l3_misses': 0,
            'total_requests': 0,
            'cache_writes': 0,
            'prefetch_coordination': 0
        }

    async def initialize(self, redis_config: Dict[str, Any]):
        """Initialize Redis cache clients"""
        try:
            # Initialize L2 cache client (proactive - port 6379)
            if 'l2' in redis_config:
                # Replace with your Redis client initialization
                self._l2_client = await self._create_redis_client(
                    redis_config['l2']
                )
                logger.info("L2 cache (proactive) initialized on port 6379")

            # Initialize L3 cache client (reactive - port 6380)
            if 'l3' in redis_config:
                # Replace with your Redis client initialization
                self._l3_client = await self._create_redis_client(
                    redis_config['l3']
                )
                logger.info("L3 cache (reactive) initialized on port 6380")

        except Exception as e:
            logger.error(f"Failed to initialize cache clients: {e}")
            raise

    async def _create_redis_client(self, config: Dict[str, Any]):
        """Create Redis client - replace with your preferred Redis library"""
        # Placeholder for Redis client creation
        # Example with aioredis:
        # import aioredis
        # return await aioredis.from_url(
        #     f"redis://{config['host']}:{config['port']}/{config['db']}"
        # )
        return None  # Placeholder

    async def check_cache(self, request: MultiKBQueryRequest) -> Optional[Dict[str, Any]]:
        """
        Check cache for existing results

        Priority order:
        1. L2 cache (proactively warmed data)
        2. L3 cache (router-cached complex results)
        """
        self.stats['total_requests'] += 1
        cache_key = self._generate_cache_key(request)

        try:
            # First check L2 cache (proactively warmed)
            if self._l2_client:
                l2_result = await self._get_from_l2(cache_key)
                if l2_result:
                    self.stats['l2_hits'] += 1
                    logger.debug(f"L2 cache hit for {cache_key}")
                    return {
                        'data': l2_result,
                        'cache_layer': 'l2',
                        'kb_sources': self._extract_kb_sources(request)
                    }
                else:
                    self.stats['l2_misses'] += 1

            # Then check L3 cache (router-cached results)
            if self._l3_client:
                l3_result = await self._get_from_l3(cache_key)
                if l3_result:
                    self.stats['l3_hits'] += 1
                    logger.debug(f"L3 cache hit for {cache_key}")
                    return {
                        'data': l3_result,
                        'cache_layer': 'l3',
                        'kb_sources': self._extract_kb_sources(request)
                    }
                else:
                    self.stats['l3_misses'] += 1

        except Exception as e:
            logger.warning(f"Cache check failed for {cache_key}: {e}")

        return None

    async def cache_result(self, request: MultiKBQueryRequest, response: MultiKBQueryResponse):
        """
        Cache query result in appropriate layer

        Caching strategy:
        - Single KB queries → L2 cache (likely to be requested again)
        - Cross-KB queries → L3 cache (expensive to recompute)
        - Analytics queries → L3 cache (longer TTL)
        """
        cache_key = self._generate_cache_key(request)

        try:
            if self._should_cache_in_l2(request):
                await self._cache_in_l2(cache_key, response.data, request)

            if self._should_cache_in_l3(request):
                await self._cache_in_l3(cache_key, response.data, request)

            self.stats['cache_writes'] += 1

        except Exception as e:
            logger.warning(f"Failed to cache result for {cache_key}: {e}")

    def _should_cache_in_l2(self, request: MultiKBQueryRequest) -> bool:
        """Determine if result should be cached in L2 (proactive layer)"""
        # Cache single KB lookups that are likely to be accessed again
        l2_patterns = [
            'kb7_terminology_lookup',
            'kb1_patient_lookup',
            'kb5_interaction_check',
            'kb4_safety_rule_check'
        ]
        return request.pattern.value in l2_patterns and not request.cross_kb_scope

    def _should_cache_in_l3(self, request: MultiKBQueryRequest) -> bool:
        """Determine if result should be cached in L3 (reactive layer)"""
        # Cache complex cross-KB queries and analytics
        l3_patterns = [
            'cross_kb_patient_view',
            'cross_kb_drug_analysis',
            'cross_kb_semantic_search',
            'patient_analytics',
            'drug_analytics',
            'clinical_reasoning'
        ]
        return (
            request.pattern.value in l3_patterns or
            request.cross_kb_scope is not None or
            'analytics' in request.pattern.value
        )

    async def _get_from_l2(self, cache_key: str) -> Optional[Any]:
        """Get data from L2 cache (proactively warmed)"""
        if not self._l2_client:
            return None

        try:
            # Replace with your Redis client's get method
            # cached_data = await self._l2_client.get(cache_key)
            # if cached_data:
            #     return json.loads(cached_data)
            return None  # Placeholder

        except Exception as e:
            logger.warning(f"L2 cache get failed for {cache_key}: {e}")
            return None

    async def _get_from_l3(self, cache_key: str) -> Optional[Any]:
        """Get data from L3 cache (router-cached results)"""
        if not self._l3_client:
            return None

        try:
            # Replace with your Redis client's get method
            # cached_data = await self._l3_client.get(cache_key)
            # if cached_data:
            #     return json.loads(cached_data)
            return None  # Placeholder

        except Exception as e:
            logger.warning(f"L3 cache get failed for {cache_key}: {e}")
            return None

    async def _cache_in_l2(self, cache_key: str, data: Any, request: MultiKBQueryRequest):
        """Cache data in L2 (proactive cache) with 5 minute TTL"""
        if not self._l2_client:
            return

        try:
            ttl = 300  # 5 minutes for frequently accessed data
            serialized_data = json.dumps(data, default=str)

            # Replace with your Redis client's setex method
            # await self._l2_client.setex(cache_key, ttl, serialized_data)

            logger.debug(f"Cached in L2: {cache_key} (TTL: {ttl}s)")

        except Exception as e:
            logger.warning(f"L2 cache set failed for {cache_key}: {e}")

    async def _cache_in_l3(self, cache_key: str, data: Any, request: MultiKBQueryRequest):
        """Cache data in L3 (reactive cache) with 1 hour TTL"""
        if not self._l3_client:
            return

        try:
            ttl = 3600  # 1 hour for complex query results
            serialized_data = json.dumps(data, default=str)

            # Replace with your Redis client's setex method
            # await self._l3_client.setex(cache_key, ttl, serialized_data)

            logger.debug(f"Cached in L3: {cache_key} (TTL: {ttl}s)")

        except Exception as e:
            logger.warning(f"L3 cache set failed for {cache_key}: {e}")

    def _generate_cache_key(self, request: MultiKBQueryRequest) -> str:
        """Generate unique cache key for request"""
        # Create deterministic key from request components
        key_components = {
            'kb_id': request.kb_id,
            'pattern': request.pattern.value,
            'params': request.params,
            'cross_kb_scope': request.cross_kb_scope
        }

        key_string = json.dumps(key_components, sort_keys=True)
        key_hash = hashlib.sha256(key_string.encode()).hexdigest()[:16]

        return f"cardiofit:{request.pattern.value}:{key_hash}"

    def _extract_kb_sources(self, request: MultiKBQueryRequest) -> List[str]:
        """Extract KB sources from request"""
        if request.cross_kb_scope:
            return request.cross_kb_scope
        elif request.kb_id:
            return [request.kb_id]
        else:
            return []

    async def coordinate_with_prefetcher(self, event_data: Dict[str, Any]):
        """
        Coordinate with Cache Prefetcher for proactive warming
        Called when Kafka events (e.g., recipe_determined) are received
        """
        try:
            self.stats['prefetch_coordination'] += 1

            # Extract workflow context from event
            workflow_id = event_data.get('workflow_id')
            patient_id = event_data.get('patient_id')
            recipe_data = event_data.get('recipe', {})

            # Predict likely data needs based on recipe
            cache_keys = await self._predict_cache_needs(workflow_id, patient_id, recipe_data)

            # Pre-warm L2 cache with predicted data
            for cache_key, data_query in cache_keys.items():
                await self._proactive_warm(cache_key, data_query)

            logger.info(f"Proactive cache warming completed for workflow {workflow_id}")

        except Exception as e:
            logger.error(f"Cache prefetcher coordination failed: {e}")

    async def _predict_cache_needs(self, workflow_id: str, patient_id: str, recipe_data: Dict[str, Any]) -> Dict[str, Dict[str, Any]]:
        """
        Predict what data will be needed based on workflow recipe
        Returns cache_key -> data_query mapping
        """
        predicted_queries = {}

        # Predict patient data needs
        if patient_id:
            patient_cache_key = f"cardiofit:kb1_patient_lookup:{patient_id}"
            predicted_queries[patient_cache_key] = {
                'kb_id': 'kb1',
                'pattern': 'kb1_patient_lookup',
                'params': {'patient_id': patient_id}
            }

        # Predict terminology needs based on recipe
        if 'conditions' in recipe_data:
            for condition_code in recipe_data['conditions']:
                term_cache_key = f"cardiofit:kb7_terminology_lookup:{condition_code}"
                predicted_queries[term_cache_key] = {
                    'kb_id': 'kb7',
                    'pattern': 'kb7_terminology_lookup',
                    'params': {'code': condition_code, 'system': 'ICD10'}
                }

        # Predict drug interaction needs
        if 'medications' in recipe_data:
            drug_codes = [med.get('rxnorm') for med in recipe_data['medications'] if med.get('rxnorm')]
            if drug_codes:
                interaction_cache_key = f"cardiofit:kb5_interaction_check:{hash(tuple(drug_codes))}"
                predicted_queries[interaction_cache_key] = {
                    'kb_id': 'kb5',
                    'pattern': 'kb5_interaction_check',
                    'params': {'drug_codes': drug_codes}
                }

        return predicted_queries

    async def _proactive_warm(self, cache_key: str, data_query: Dict[str, Any]):
        """Proactively warm cache with predicted data"""
        try:
            # This would typically execute the predicted query and cache the result
            # For now, we'll just log the warming action
            logger.debug(f"Proactively warming cache key: {cache_key}")

            # In a real implementation, you would:
            # 1. Execute the predicted query
            # 2. Cache the result in L2 with appropriate TTL
            # 3. Mark as proactively warmed

        except Exception as e:
            logger.warning(f"Proactive cache warming failed for {cache_key}: {e}")

    async def invalidate_cache(self, pattern: str = None, kb_id: str = None):
        """Invalidate cache entries based on pattern or KB"""
        try:
            if pattern:
                # Invalidate all entries matching pattern
                pattern_key = f"cardiofit:{pattern}:*"
                await self._invalidate_pattern(pattern_key)

            if kb_id:
                # Invalidate all entries for specific KB
                kb_key = f"cardiofit:*{kb_id}*"
                await self._invalidate_pattern(kb_key)

            logger.info(f"Cache invalidation completed for pattern={pattern}, kb_id={kb_id}")

        except Exception as e:
            logger.error(f"Cache invalidation failed: {e}")

    async def _invalidate_pattern(self, pattern: str):
        """Invalidate cache entries matching pattern"""
        # Placeholder for cache invalidation logic
        # Would use Redis SCAN and DEL commands
        pass

    async def get_stats(self) -> Dict[str, Any]:
        """Get cache statistics"""
        total_requests = self.stats['total_requests']
        l2_hit_rate = (self.stats['l2_hits'] / total_requests) if total_requests > 0 else 0
        l3_hit_rate = (self.stats['l3_hits'] / total_requests) if total_requests > 0 else 0
        overall_hit_rate = ((self.stats['l2_hits'] + self.stats['l3_hits']) / total_requests) if total_requests > 0 else 0

        return {
            'l2_cache': {
                'hits': self.stats['l2_hits'],
                'misses': self.stats['l2_misses'],
                'hit_rate': l2_hit_rate
            },
            'l3_cache': {
                'hits': self.stats['l3_hits'],
                'misses': self.stats['l3_misses'],
                'hit_rate': l3_hit_rate
            },
            'overall': {
                'total_requests': total_requests,
                'overall_hit_rate': overall_hit_rate,
                'cache_writes': self.stats['cache_writes'],
                'prefetch_coordination': self.stats['prefetch_coordination']
            }
        }