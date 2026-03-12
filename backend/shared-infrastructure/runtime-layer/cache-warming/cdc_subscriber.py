"""
CDC Cache Warmer for KB7 Terminology Service
Subscribes to CDC events from Adapter and warms caches proactively
Implements intelligent cache warming based on usage patterns and data changes
"""

from aiokafka import AIOKafkaConsumer
import asyncio
import json
from typing import Dict, List, Any, Optional
from datetime import datetime, timedelta
import redis.asyncio as redis
from loguru import logger
import hashlib


class CachePrefetcher:
    """
    Intelligent cache prefetcher that predicts and loads data
    based on usage patterns and change events
    """

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize Cache Prefetcher

        Args:
            config: Configuration dictionary
        """
        self.config = config
        self.usage_patterns = {}
        self.prefetch_scores = {}

        # Redis clients for different cache layers
        self.redis_l2 = redis.Redis.from_url(
            config.get('redis_l2_url', 'redis://localhost:6379/0'),
            decode_responses=True
        )
        self.redis_l3 = redis.Redis.from_url(
            config.get('redis_l3_url', 'redis://localhost:6379/1'),
            decode_responses=True
        )

        # Data store clients for prefetching
        self._neo4j_manager = None
        self._clickhouse_manager = None
        self._postgres_client = None

        logger.info("Cache Prefetcher initialized")

    async def prefetch_interactions(self, drug_codes: List[str]) -> None:
        """Prefetch drug interaction data"""
        if not self._neo4j_manager:
            from ..neo4j_setup.dual_stream_manager import Neo4jDualStreamManager
            self._neo4j_manager = Neo4jDualStreamManager(self.config.get('neo4j', {}))

        interactions = await self._neo4j_manager.query_drug_interactions(drug_codes)

        for interaction in interactions:
            cache_key = f"ddi:{interaction['drug1']}:{interaction['drug2']}"
            await self.redis_l2.setex(
                cache_key,
                3600,  # 1 hour TTL
                json.dumps(interaction)
            )

        logger.debug(f"Prefetched {len(interactions)} drug interactions")

    async def prefetch_medication_scores(self, indication: str,
                                        drug_codes: List[str]) -> None:
        """Prefetch medication scoring data"""
        if not self._clickhouse_manager:
            from ..clickhouse_runtime.manager import ClickHouseRuntimeManager
            self._clickhouse_manager = ClickHouseRuntimeManager(self.config.get('clickhouse', {}))

        scores = await self._clickhouse_manager.calculate_medication_scores(
            drugs=drug_codes,
            indication=indication
        )

        cache_key = f"scores:{indication}:{':'.join(sorted(drug_codes))}"
        await self.redis_l2.setex(
            cache_key,
            1800,  # 30 minutes TTL
            scores.to_json()
        )

        logger.debug(f"Prefetched medication scores for {len(drug_codes)} drugs")

    async def prefetch_terminology(self, codes: List[str], system: str) -> None:
        """Prefetch terminology lookup data"""
        if not self._postgres_client:
            from ..internal.database import database
            self._postgres_client = await database.get_connection()

        for code in codes:
            result = await self._postgres_client.fetchrow("""
                SELECT concept_uuid, code, preferred_term, system, active
                FROM concepts
                WHERE code = $1 AND system = $2
            """, code, system)

            if result:
                cache_key = f"term:{system}:{code}"
                await self.redis_l2.setex(
                    cache_key,
                    7200,  # 2 hours TTL
                    json.dumps(dict(result))
                )

        logger.debug(f"Prefetched {len(codes)} terminology concepts")

    def record_usage_pattern(self, pattern: str, context: Dict[str, Any]) -> None:
        """Record usage pattern for intelligent prefetching"""
        timestamp = datetime.utcnow()
        pattern_key = f"{pattern}:{hash(str(context))}"

        if pattern_key not in self.usage_patterns:
            self.usage_patterns[pattern_key] = []

        self.usage_patterns[pattern_key].append({
            'timestamp': timestamp,
            'context': context
        })

        # Keep only recent patterns (last 24 hours)
        cutoff = timestamp - timedelta(hours=24)
        self.usage_patterns[pattern_key] = [
            p for p in self.usage_patterns[pattern_key]
            if p['timestamp'] > cutoff
        ]


class CDCCacheWarmer:
    """
    Subscribes to CDC events from Adapter and warms caches
    Implements event-driven cache warming for optimal performance
    """

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize CDC Cache Warmer

        Args:
            config: Configuration dictionary with Kafka and Redis settings
        """
        self.config = config

        # Kafka consumer for CDC events
        self.kafka_consumer = AIOKafkaConsumer(
            'adapter.cdc.events',
            'cache.warming.requests',
            bootstrap_servers=config.get('kafka_brokers', ['localhost:9092']),
            group_id='cache-warmer-cdc',
            value_deserializer=lambda m: json.loads(m.decode('utf-8')),
            auto_offset_reset='latest'
        )

        # Cache prefetcher
        self.cache_prefetcher = CachePrefetcher(config)

        # Redis clients
        self.redis_l2 = redis.Redis.from_url(
            config.get('redis_l2_url', 'redis://localhost:6379/0'),
            decode_responses=True
        )
        self.redis_l3 = redis.Redis.from_url(
            config.get('redis_l3_url', 'redis://localhost:6379/1'),
            decode_responses=True
        )

        # Warming statistics
        self.warming_stats = {
            'events_processed': 0,
            'cache_entries_warmed': 0,
            'errors': 0,
            'start_time': datetime.utcnow()
        }

        logger.info("CDC Cache Warmer initialized")

    async def start_warming_from_cdc(self) -> None:
        """Start listening to CDC events and warm relevant caches"""
        await self.kafka_consumer.start()

        try:
            logger.info("Started CDC cache warming")

            async for msg in self.kafka_consumer:
                try:
                    cdc_event = msg.value
                    await self._process_cdc_event(cdc_event)
                    self.warming_stats['events_processed'] += 1

                except Exception as e:
                    logger.error(f"Error processing CDC event: {e}")
                    self.warming_stats['errors'] += 1

        except Exception as e:
            logger.error(f"Error in CDC warming loop: {e}")
        finally:
            await self.kafka_consumer.stop()

    async def _process_cdc_event(self, event: Dict[str, Any]) -> None:
        """
        Process CDC event and trigger appropriate cache warming

        Args:
            event: CDC event dictionary
        """
        event_type = event.get('event_type')
        logger.debug(f"Processing CDC event: {event_type}")

        if event_type == 'kb_synchronized':
            await self._warm_from_kb_change(event)
        elif event_type == 'cache_warming_required':
            await self._warm_from_explicit_request(event)
        elif event_type == 'entity_updated':
            await self._warm_from_entity_change(event)

    async def _warm_from_kb_change(self, event: Dict[str, Any]) -> None:
        """Warm caches based on KB changes"""
        kb_source = event.get('kb_source')
        change_type = event.get('change_type')
        affected_entities = event.get('affected_entities', [])

        logger.debug(f"Warming cache for KB change: {kb_source}:{change_type}")

        if kb_source == 'KB-5' and change_type == 'drug_interaction_added':
            # Warm drug interaction caches
            drug_codes = [entity.get('drug_code') for entity in affected_entities
                         if entity.get('drug_code')]

            if drug_codes:
                await self.cache_prefetcher.prefetch_interactions(drug_codes)
                self.warming_stats['cache_entries_warmed'] += len(drug_codes)

        elif kb_source == 'KB-3' and change_type == 'medication_score_updated':
            # Warm medication scoring caches
            for entity in affected_entities:
                drug_codes = entity.get('drug_codes', [])
                indication = entity.get('indication')

                if drug_codes and indication:
                    await self.cache_prefetcher.prefetch_medication_scores(
                        indication, drug_codes
                    )
                    self.warming_stats['cache_entries_warmed'] += 1

        elif change_type in ['concept_added', 'concept_updated']:
            # Warm terminology caches
            concepts = [entity for entity in affected_entities
                       if 'code' in entity and 'system' in entity]

            for concept in concepts:
                await self.cache_prefetcher.prefetch_terminology(
                    [concept['code']], concept['system']
                )
                self.warming_stats['cache_entries_warmed'] += 1

    async def _warm_from_explicit_request(self, event: Dict[str, Any]) -> None:
        """Warm caches from explicit cache warming request"""
        patterns = event.get('patterns', [])
        priority = event.get('priority', 'normal')
        source_event = event.get('source_event', {})

        logger.debug(f"Explicit cache warming: {patterns} (priority: {priority})")

        for pattern in patterns:
            await self._warm_pattern(pattern, source_event, priority)

    async def _warm_pattern(self, pattern: str, source_event: Dict[str, Any],
                           priority: str) -> None:
        """Warm cache for specific pattern"""
        if pattern == 'drug_interactions':
            # Extract drug codes from source event
            affected_entities = source_event.get('affected_entities', [])
            drug_codes = [entity.get('drug_code') for entity in affected_entities
                         if entity.get('drug_code')]

            if drug_codes:
                await self.cache_prefetcher.prefetch_interactions(drug_codes)

        elif pattern == 'medication_scoring':
            # Warm popular indication-drug combinations
            await self._warm_popular_medication_scores()

        elif pattern == 'terminology_lookup':
            # Warm frequently accessed terminology
            await self._warm_popular_terminology()

    async def _warm_popular_medication_scores(self) -> None:
        """Warm caches for popular medication scoring queries"""
        # Get popular indication-drug combinations from usage patterns
        popular_combinations = await self._get_popular_combinations()

        for combination in popular_combinations:
            indication = combination.get('indication')
            drug_codes = combination.get('drug_codes', [])

            if indication and drug_codes:
                await self.cache_prefetcher.prefetch_medication_scores(
                    indication, drug_codes
                )

    async def _warm_popular_terminology(self) -> None:
        """Warm caches for frequently accessed terminology"""
        # Get popular terminology lookups
        popular_terms = await self._get_popular_terminology()

        for term in popular_terms:
            code = term.get('code')
            system = term.get('system')

            if code and system:
                await self.cache_prefetcher.prefetch_terminology([code], system)

    async def _get_popular_combinations(self) -> List[Dict[str, Any]]:
        """Get popular indication-drug combinations from usage analytics"""
        # Simplified - would query ClickHouse for actual usage data
        return [
            {'indication': 'I25.10', 'drug_codes': ['1234567', '2345678']},
            {'indication': 'E11.9', 'drug_codes': ['3456789', '4567890']}
        ]

    async def _get_popular_terminology(self) -> List[Dict[str, Any]]:
        """Get popular terminology lookups from usage analytics"""
        # Simplified - would query usage statistics
        return [
            {'code': '387517004', 'system': 'SNOMED'},
            {'code': '1234567', 'system': 'RxNorm'}
        ]

    async def _warm_from_entity_change(self, event: Dict[str, Any]) -> None:
        """Warm caches based on entity updates"""
        entity_type = event.get('entity_type')
        entity_data = event.get('entity_data', {})

        if entity_type == 'drug':
            drug_code = entity_data.get('rxnorm')
            if drug_code:
                # Warm interaction data for this drug
                await self.cache_prefetcher.prefetch_interactions([drug_code])

        elif entity_type == 'patient':
            patient_id = entity_data.get('patient_id')
            if patient_id:
                # Warm patient-specific data
                await self._warm_patient_context(patient_id)

    async def _warm_patient_context(self, patient_id: str) -> None:
        """Warm cache for patient-specific context"""
        # Get patient medications and conditions for context-aware warming
        # Would implement actual patient data retrieval and warming
        logger.debug(f"Warming patient context for {patient_id}")

    async def invalidate_cache(self, pattern: str, keys: List[str]) -> None:
        """
        Invalidate specific cache entries

        Args:
            pattern: Cache pattern to invalidate
            keys: Specific keys to invalidate
        """
        invalidated = 0

        for key in keys:
            cache_key = f"{pattern}:{key}"

            # Remove from L2 cache
            if await self.redis_l2.delete(cache_key):
                invalidated += 1

            # Remove from L3 cache
            await self.redis_l3.delete(cache_key)

        logger.info(f"Invalidated {invalidated} cache entries for pattern {pattern}")

    async def get_cache_statistics(self) -> Dict[str, Any]:
        """Get cache warming statistics"""
        uptime = datetime.utcnow() - self.warming_stats['start_time']

        l2_info = await self.redis_l2.info('memory')
        l3_info = await self.redis_l3.info('memory')

        return {
            'warming_stats': self.warming_stats,
            'uptime_seconds': uptime.total_seconds(),
            'cache_memory': {
                'l2_used_mb': l2_info.get('used_memory', 0) / 1024 / 1024,
                'l3_used_mb': l3_info.get('used_memory', 0) / 1024 / 1024
            },
            'timestamp': datetime.utcnow().isoformat()
        }

    async def close(self) -> None:
        """Close CDC cache warmer"""
        await self.kafka_consumer.stop()
        await self.redis_l2.close()
        await self.redis_l3.close()

        logger.info("CDC Cache Warmer closed")