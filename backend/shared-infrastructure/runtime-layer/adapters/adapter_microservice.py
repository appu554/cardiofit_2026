"""
Adapter Microservice for KB7 Terminology Service
Central adapter that synchronizes data from authoritative KBs to runtime stores
Publishes CDC events for cache warming and service coordination
"""

from aiokafka import AIOKafkaProducer, AIOKafkaConsumer
import asyncio
from typing import Dict, List, Any, Optional
import json
from datetime import datetime
from loguru import logger
import hashlib


class AdapterMicroservice:
    """
    Central adapter that synchronizes data from authoritative KBs
    to runtime stores and publishes CDC events
    """

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize Adapter Microservice

        Args:
            config: Configuration dictionary with Kafka, Neo4j, ClickHouse settings
        """
        self.config = config

        # Kafka producer for CDC events
        self.kafka_producer = AIOKafkaProducer(
            bootstrap_servers=config.get('kafka_brokers', ['localhost:9092']),
            value_serializer=lambda v: json.dumps(v).encode('utf-8'),
            key_serializer=lambda k: k.encode('utf-8') if k else None,
            compression_type='gzip',
            batch_size=16384,
            linger_ms=10
        )

        # Kafka consumer for KB change events
        self.kafka_consumer = AIOKafkaConsumer(
            'kb.changes',
            'terminology.updates',
            bootstrap_servers=config.get('kafka_brokers', ['localhost:9092']),
            group_id='adapter-microservice',
            value_deserializer=lambda m: json.loads(m.decode('utf-8')),
            auto_offset_reset='latest'
        )

        # Data store managers (initialized lazily)
        self._neo4j_manager = None
        self._clickhouse_manager = None
        self._graphdb_client = None
        self._postgres_client = None

        # CDC topics
        self.cdc_topic = 'adapter.cdc.events'
        self.cache_warming_topic = 'cache.warming.requests'
        self.sync_requests_topic = 'adapter.sync.requests'

        logger.info("Adapter Microservice initialized")

    async def start(self) -> None:
        """Start the adapter microservice"""
        await self._initialize_clients()
        await self.kafka_producer.start()
        await self.kafka_consumer.start()

        # Start processing tasks
        asyncio.create_task(self._process_kb_changes())
        asyncio.create_task(self._periodic_sync())

        logger.info("Adapter Microservice started")

    async def stop(self) -> None:
        """Stop the adapter microservice"""
        await self.kafka_consumer.stop()
        await self.kafka_producer.stop()

        if self._neo4j_manager:
            await self._neo4j_manager.close()
        if self._clickhouse_manager:
            self._clickhouse_manager.close()

        logger.info("Adapter Microservice stopped")

    async def _initialize_clients(self) -> None:
        """Initialize data store clients"""
        if not self._neo4j_manager:
            from ..neo4j_setup.dual_stream_manager import Neo4jDualStreamManager
            self._neo4j_manager = Neo4jDualStreamManager(self.config.get('neo4j', {}))
            await self._neo4j_manager.initialize_databases()

        if not self._clickhouse_manager:
            from ..clickhouse_runtime.manager import ClickHouseRuntimeManager
            self._clickhouse_manager = ClickHouseRuntimeManager(self.config.get('clickhouse', {}))

        if not self._postgres_client:
            from ..internal.database import database
            self._postgres_client = await database.get_connection()

        logger.info("Adapter clients initialized")

    async def _process_kb_changes(self) -> None:
        """Process KB change events from Kafka"""
        try:
            async for message in self.kafka_consumer:
                try:
                    change_event = message.value
                    logger.info(f"Processing KB change: {change_event.get('type', 'unknown')}")

                    await self.sync_kb_changes(change_event)

                except Exception as e:
                    logger.error(f"Error processing KB change: {e}")

        except Exception as e:
            logger.error(f"Error in KB change processing loop: {e}")

    async def sync_kb_changes(self, change_event: Dict[str, Any]) -> None:
        """
        Process KB changes and sync to runtime stores

        Args:
            change_event: KB change event dictionary
        """
        kb_source = change_event.get('source')
        change_type = change_event.get('type')
        entity_data = change_event.get('data', {})

        logger.debug(f"Syncing change from {kb_source}: {change_type}")

        try:
            # Route to appropriate runtime stores based on KB source
            if kb_source in ['KB-4', 'KB-5', 'KB-7']:  # Semantic KBs
                await self._sync_to_neo4j_semantic(change_event)

            if kb_source in ['KB-3', 'KB-6']:  # Scoring and Guidelines KBs
                await self._sync_to_clickhouse(change_event)

            if kb_source in ['KB-1', 'KB-2']:  # Core Terminology
                await self._sync_to_postgres(change_event)

            # Publish CDC event for downstream processing
            await self._publish_cdc_event(change_event)

        except Exception as e:
            logger.error(f"Error syncing KB change: {e}")
            await self._publish_error_event(change_event, str(e))

    async def _sync_to_neo4j_semantic(self, change_event: Dict[str, Any]) -> None:
        """Sync KB changes to Neo4j semantic mesh"""
        change_type = change_event.get('type')
        data = change_event.get('data', {})
        kb_source = change_event.get('source')

        async with self._neo4j_manager.driver.session(database="semantic_mesh") as session:

            if change_type == 'drug_interaction_added':
                await session.run("""
                    MERGE (d1:Drug {rxnorm: $drug1})
                    MERGE (d2:Drug {rxnorm: $drug2})
                    MERGE (d1)-[i:INTERACTS_WITH]-(d2)
                    SET i.severity = $severity,
                        i.mechanism = $mechanism,
                        i.source = $kb_source,
                        i.updated = datetime(),
                        i.evidence_level = $evidence_level
                """, **data, kb_source=kb_source)

            elif change_type == 'contraindication_added':
                await session.run("""
                    MERGE (d:Drug {rxnorm: $drug_rxnorm})
                    MERGE (c:Condition {code: $condition_code})
                    MERGE (d)-[r:CONTRAINDICATED_IN]->(c)
                    SET r.severity = $severity,
                        r.rationale = $rationale,
                        r.source = $kb_source,
                        r.updated = datetime()
                """, **data, kb_source=kb_source)

            elif change_type == 'drug_class_updated':
                await session.run("""
                    MERGE (d:Drug {rxnorm: $drug_rxnorm})
                    MERGE (dc:DrugClass {code: $class_code})
                    SET dc.name = $class_name,
                        dc.updated = datetime()
                    MERGE (d)-[r:BELONGS_TO]->(dc)
                    SET r.updated = datetime()
                """, **data)

        logger.debug(f"Synced {change_type} to Neo4j semantic mesh")

    async def _sync_to_clickhouse(self, change_event: Dict[str, Any]) -> None:
        """Sync KB changes to ClickHouse for analytics"""
        change_type = change_event.get('type')
        data = change_event.get('data', {})
        kb_source = change_event.get('source')

        if change_type == 'medication_score_updated':
            self._clickhouse_manager.client.execute("""
                INSERT INTO kb7_analytics.medication_scores
                (drug_rxnorm, indication_code, guideline_score, safety_score,
                 efficacy_score, kb_version, calculated_at)
                VALUES
            """, [{
                'drug_rxnorm': data.get('drug_rxnorm'),
                'indication_code': data.get('indication_code'),
                'guideline_score': data.get('guideline_score', 0),
                'safety_score': data.get('safety_score', 0),
                'efficacy_score': data.get('efficacy_score', 0),
                'kb_version': data.get('version', 'unknown'),
                'calculated_at': datetime.utcnow()
            }])

        elif change_type == 'guideline_updated':
            self._clickhouse_manager.client.execute("""
                INSERT INTO kb7_analytics.guideline_compliance
                (guideline_id, guideline_name, indication_code, drug_rxnorm,
                 compliance_score, recommendation_strength, evidence_level, updated_at)
                VALUES
            """, [{
                'guideline_id': data.get('guideline_id'),
                'guideline_name': data.get('guideline_name'),
                'indication_code': data.get('indication_code'),
                'drug_rxnorm': data.get('drug_rxnorm'),
                'compliance_score': data.get('compliance_score', 0),
                'recommendation_strength': data.get('strength', 'moderate'),
                'evidence_level': data.get('evidence_level', 'C'),
                'updated_at': datetime.utcnow()
            }])

        logger.debug(f"Synced {change_type} to ClickHouse")

    async def _sync_to_postgres(self, change_event: Dict[str, Any]) -> None:
        """Sync core terminology changes to PostgreSQL"""
        change_type = change_event.get('type')
        data = change_event.get('data', {})

        if change_type == 'concept_added':
            await self._postgres_client.execute("""
                INSERT INTO concepts
                (concept_uuid, code, preferred_term, system, active, updated_at)
                VALUES ($1, $2, $3, $4, $5, NOW())
                ON CONFLICT (code, system)
                DO UPDATE SET
                    preferred_term = EXCLUDED.preferred_term,
                    active = EXCLUDED.active,
                    updated_at = NOW()
            """,
            data.get('concept_uuid'),
            data.get('code'),
            data.get('preferred_term'),
            data.get('system'),
            data.get('active', True))

        elif change_type == 'concept_updated':
            await self._postgres_client.execute("""
                UPDATE concepts
                SET preferred_term = $1, active = $2, updated_at = NOW()
                WHERE code = $3 AND system = $4
            """,
            data.get('preferred_term'),
            data.get('active', True),
            data.get('code'),
            data.get('system'))

        logger.debug(f"Synced {change_type} to PostgreSQL")

    async def _publish_cdc_event(self, change_event: Dict[str, Any]) -> None:
        """Publish CDC event for cache warming and service coordination"""
        cdc_event = {
            'event_type': 'kb_synchronized',
            'kb_source': change_event.get('source'),
            'change_type': change_event.get('type'),
            'affected_entities': change_event.get('entities', []),
            'timestamp': datetime.utcnow().isoformat(),
            'sync_id': self._generate_sync_id(change_event)
        }

        # Publish to CDC topic
        await self.kafka_producer.send(
            self.cdc_topic,
            key=cdc_event['kb_source'],
            value=cdc_event
        )

        # Publish to cache warming topic if relevant
        if self._should_trigger_cache_warming(change_event):
            cache_warming_event = {
                'event_type': 'cache_warming_required',
                'patterns': self._determine_cache_patterns(change_event),
                'priority': self._determine_priority(change_event),
                'source_event': cdc_event
            }

            await self.kafka_producer.send(
                self.cache_warming_topic,
                key=cdc_event['kb_source'],
                value=cache_warming_event
            )

        logger.debug(f"Published CDC event for {change_event.get('type')}")

    async def _publish_error_event(self, change_event: Dict[str, Any], error: str) -> None:
        """Publish error event for monitoring"""
        error_event = {
            'event_type': 'sync_error',
            'source_event': change_event,
            'error_message': error,
            'timestamp': datetime.utcnow().isoformat()
        }

        await self.kafka_producer.send(
            'adapter.errors',
            value=error_event
        )

    def _should_trigger_cache_warming(self, change_event: Dict[str, Any]) -> bool:
        """Determine if change should trigger cache warming"""
        warming_triggers = [
            'drug_interaction_added',
            'contraindication_added',
            'medication_score_updated',
            'concept_updated'
        ]
        return change_event.get('type') in warming_triggers

    def _determine_cache_patterns(self, change_event: Dict[str, Any]) -> List[str]:
        """Determine which cache patterns to warm based on change type"""
        change_type = change_event.get('type')

        pattern_map = {
            'drug_interaction_added': ['drug_interactions', 'safety_analytics'],
            'contraindication_added': ['contraindications', 'safety_analytics'],
            'medication_score_updated': ['medication_scoring'],
            'concept_updated': ['terminology_lookup'],
            'guideline_updated': ['guideline_compliance']
        }

        return pattern_map.get(change_type, [])

    def _determine_priority(self, change_event: Dict[str, Any]) -> str:
        """Determine cache warming priority"""
        high_priority_types = [
            'drug_interaction_added',
            'contraindication_added'
        ]

        if change_event.get('type') in high_priority_types:
            return 'high'
        elif change_event.get('severity') == 'critical':
            return 'high'
        else:
            return 'normal'

    def _generate_sync_id(self, change_event: Dict[str, Any]) -> str:
        """Generate unique sync ID for tracking"""
        data = f"{change_event.get('source')}:{change_event.get('type')}:{datetime.utcnow().isoformat()}"
        return hashlib.md5(data.encode()).hexdigest()[:16]

    async def _periodic_sync(self) -> None:
        """Periodic synchronization task"""
        while True:
            try:
                await asyncio.sleep(300)  # 5 minutes
                await self._perform_health_sync()
            except Exception as e:
                logger.error(f"Error in periodic sync: {e}")

    async def _perform_health_sync(self) -> None:
        """Perform health check and sync statistics"""
        health_data = {
            'adapter_id': 'kb7-adapter',
            'timestamp': datetime.utcnow().isoformat(),
            'sync_stats': await self._get_sync_statistics(),
            'store_health': await self._check_store_health()
        }

        await self.kafka_producer.send(
            'adapter.health',
            value=health_data
        )

    async def _get_sync_statistics(self) -> Dict[str, Any]:
        """Get synchronization statistics"""
        # Would implement actual statistics gathering
        return {
            'total_syncs': 0,
            'successful_syncs': 0,
            'failed_syncs': 0,
            'last_sync': datetime.utcnow().isoformat()
        }

    async def _check_store_health(self) -> Dict[str, str]:
        """Check health of all connected stores"""
        health = {}

        try:
            neo4j_health = await self._neo4j_manager.health_check()
            health['neo4j'] = neo4j_health['status']
        except:
            health['neo4j'] = 'unhealthy'

        try:
            ch_health = self._clickhouse_manager.health_check()
            health['clickhouse'] = ch_health['status']
        except:
            health['clickhouse'] = 'unhealthy'

        try:
            await self._postgres_client.fetchval("SELECT 1")
            health['postgres'] = 'healthy'
        except:
            health['postgres'] = 'unhealthy'

        return health

    async def manual_sync(self, kb_source: str, sync_type: str = 'full') -> Dict[str, Any]:
        """
        Manually trigger synchronization for a specific KB

        Args:
            kb_source: Knowledge base source identifier
            sync_type: Type of sync ('full', 'incremental')

        Returns:
            Sync result dictionary
        """
        logger.info(f"Manual sync triggered for {kb_source} ({sync_type})")

        sync_event = {
            'source': kb_source,
            'type': 'manual_sync',
            'sync_type': sync_type,
            'triggered_at': datetime.utcnow().isoformat(),
            'triggered_by': 'manual'
        }

        try:
            await self.sync_kb_changes(sync_event)

            return {
                'status': 'success',
                'kb_source': kb_source,
                'sync_type': sync_type,
                'timestamp': datetime.utcnow().isoformat()
            }

        except Exception as e:
            logger.error(f"Manual sync failed: {e}")
            return {
                'status': 'error',
                'error': str(e),
                'kb_source': kb_source,
                'sync_type': sync_type,
                'timestamp': datetime.utcnow().isoformat()
            }