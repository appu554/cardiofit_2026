"""
Event Bus Orchestrator for KB7 Terminology Service
Manages event flow between services, adapter, and cache warming
Coordinates service interactions and triggers downstream actions
"""

from aiokafka import AIOKafkaProducer, AIOKafkaConsumer
import asyncio
from typing import Dict, List, Any, Optional
import json
from datetime import datetime
from loguru import logger
import hashlib


class EventRouter:
    """
    Routes events to appropriate topics and determines triggers
    """

    def __init__(self):
        self.routing_rules = self._initialize_routing_rules()
        self.trigger_rules = self._initialize_trigger_rules()

    def _initialize_routing_rules(self) -> Dict[str, List[str]]:
        """Initialize event routing rules"""
        return {
            'medication.calculation_starting': [
                'cache.warming.requests',
                'snapshot.creation.requests'
            ],
            'safety.validation_required': [
                'cache.warming.requests'
            ],
            'admin.kb_updated': [
                'adapter.sync.requests',
                'cache.invalidation.requests'
            ],
            'patient.medications_changed': [
                'adapter.cdc.events',
                'cache.warming.requests'
            ],
            'terminology.concept_updated': [
                'adapter.cdc.events',
                'cache.invalidation.requests'
            ]
        }

    def _initialize_trigger_rules(self) -> Dict[str, List[str]]:
        """Initialize trigger determination rules"""
        return {
            'medication': {
                'calculation_starting': ['cache_warming', 'snapshot_creation'],
                'scoring_requested': ['cache_warming'],
                'alternatives_requested': ['cache_warming']
            },
            'safety': {
                'validation_required': ['cache_warming'],
                'interaction_check': ['cache_warming'],
                'contraindication_check': ['cache_warming']
            },
            'admin': {
                'kb_updated': ['data_sync', 'cache_invalidation'],
                'configuration_changed': ['cache_invalidation']
            },
            'patient': {
                'medications_changed': ['data_sync', 'cache_warming'],
                'conditions_updated': ['cache_warming']
            }
        }

    def route_event(self, service_id: str, event: Dict[str, Any]) -> List[str]:
        """
        Determine which topics to route event to

        Args:
            service_id: Service that generated the event
            event: Event data

        Returns:
            List of topic names to publish to
        """
        event_key = f"{service_id}.{event.get('type', 'unknown')}"
        return self.routing_rules.get(event_key, [])

    def determine_triggers(self, service_id: str, event: Dict[str, Any]) -> List[str]:
        """
        Determine what downstream actions to trigger

        Args:
            service_id: Service that generated the event
            event: Event data

        Returns:
            List of trigger types
        """
        service_rules = self.trigger_rules.get(service_id, {})
        event_type = event.get('type', 'unknown')
        return service_rules.get(event_type, [])


class EventBusOrchestrator:
    """
    Manages event flow between services, adapter, and cache warming
    Coordinates service interactions and automates downstream actions
    """

    def __init__(self, config: Dict[str, Any]):
        """
        Initialize Event Bus Orchestrator

        Args:
            config: Configuration dictionary with Kafka settings
        """
        self.config = config

        # Kafka producer for publishing orchestrated events
        self.kafka_producer = AIOKafkaProducer(
            bootstrap_servers=config.get('kafka_brokers', ['localhost:9092']),
            value_serializer=lambda v: json.dumps(v).encode('utf-8'),
            key_serializer=lambda k: k.encode('utf-8') if k else None,
            compression_type='gzip',
            batch_size=16384
        )

        # Kafka consumer for service events
        self.kafka_consumer = AIOKafkaConsumer(
            'service.events',
            'medication.events',
            'safety.events',
            'admin.events',
            'patient.events',
            bootstrap_servers=config.get('kafka_brokers', ['localhost:9092']),
            group_id='event-bus-orchestrator',
            value_deserializer=lambda m: json.loads(m.decode('utf-8')),
            auto_offset_reset='latest'
        )

        # Event router for determining actions
        self.event_router = EventRouter()

        # Event processing statistics
        self.processing_stats = {
            'events_processed': 0,
            'events_routed': 0,
            'triggers_activated': 0,
            'errors': 0,
            'start_time': datetime.utcnow()
        }

        logger.info("Event Bus Orchestrator initialized")

    async def start(self) -> None:
        """Start the event bus orchestrator"""
        await self.kafka_producer.start()
        await self.kafka_consumer.start()

        # Start event processing
        asyncio.create_task(self._process_service_events())

        logger.info("Event Bus Orchestrator started")

    async def stop(self) -> None:
        """Stop the event bus orchestrator"""
        await self.kafka_consumer.stop()
        await self.kafka_producer.stop()

        logger.info("Event Bus Orchestrator stopped")

    async def _process_service_events(self) -> None:
        """Process events from services and orchestrate responses"""
        try:
            async for message in self.kafka_consumer:
                try:
                    event_data = message.value
                    topic = message.topic

                    # Extract service ID from topic
                    service_id = topic.split('.')[0]

                    await self._orchestrate_event(service_id, event_data)
                    self.processing_stats['events_processed'] += 1

                except Exception as e:
                    logger.error(f"Error processing service event: {e}")
                    self.processing_stats['errors'] += 1

        except Exception as e:
            logger.error(f"Error in service event processing loop: {e}")

    async def _orchestrate_event(self, service_id: str, event: Dict[str, Any]) -> None:
        """
        Orchestrate event processing and trigger downstream actions

        Args:
            service_id: Service that generated the event
            event: Event data
        """
        logger.debug(f"Orchestrating event from {service_id}: {event.get('type')}")

        # Enrich event with orchestration metadata
        enriched_event = await self._enrich_event(service_id, event)

        # Determine routing targets
        targets = self.event_router.route_event(service_id, event)

        # Determine triggers
        triggers = self.event_router.determine_triggers(service_id, event)

        # Route to appropriate topics
        for target in targets:
            await self._route_to_topic(target, enriched_event)
            self.processing_stats['events_routed'] += 1

        # Trigger downstream actions
        for trigger in triggers:
            await self._activate_trigger(trigger, enriched_event)
            self.processing_stats['triggers_activated'] += 1

    async def _enrich_event(self, service_id: str, event: Dict[str, Any]) -> Dict[str, Any]:
        """
        Enrich event with orchestration metadata

        Args:
            service_id: Source service ID
            event: Original event

        Returns:
            Enriched event dictionary
        """
        return {
            'source_service': service_id,
            'original_event': event,
            'orchestration_id': self._generate_orchestration_id(service_id, event),
            'timestamp': datetime.utcnow().isoformat(),
            'triggers': self.event_router.determine_triggers(service_id, event),
            'routing_targets': self.event_router.route_event(service_id, event)
        }

    async def _route_to_topic(self, topic: str, event: Dict[str, Any]) -> None:
        """
        Route event to specific topic

        Args:
            topic: Target topic name
            event: Event to route
        """
        await self.kafka_producer.send(
            topic,
            key=event.get('source_service'),
            value=event
        )

        logger.debug(f"Routed event to topic: {topic}")

    async def _activate_trigger(self, trigger: str, event: Dict[str, Any]) -> None:
        """
        Activate specific trigger type

        Args:
            trigger: Trigger type to activate
            event: Event that triggered the action
        """
        if trigger == 'cache_warming':
            await self._trigger_cache_warming(event)
        elif trigger == 'snapshot_creation':
            await self._trigger_snapshot_creation(event)
        elif trigger == 'data_sync':
            await self._trigger_data_sync(event)
        elif trigger == 'cache_invalidation':
            await self._trigger_cache_invalidation(event)

    async def _trigger_cache_warming(self, event: Dict[str, Any]) -> None:
        """Trigger cache warming process"""
        warming_request = {
            'event_type': 'cache_warming_requested',
            'source_event': event,
            'patterns': self._determine_warming_patterns(event),
            'priority': self._determine_warming_priority(event),
            'timestamp': datetime.utcnow().isoformat()
        }

        await self.kafka_producer.send(
            'cache.warming.requests',
            value=warming_request
        )

        logger.debug("Triggered cache warming")

    async def _trigger_snapshot_creation(self, event: Dict[str, Any]) -> None:
        """Trigger snapshot creation for consistency"""
        snapshot_request = {
            'event_type': 'snapshot_creation_requested',
            'service_id': event.get('source_service'),
            'context': event.get('original_event', {}),
            'timestamp': datetime.utcnow().isoformat()
        }

        await self.kafka_producer.send(
            'snapshot.creation.requests',
            value=snapshot_request
        )

        logger.debug("Triggered snapshot creation")

    async def _trigger_data_sync(self, event: Dict[str, Any]) -> None:
        """Trigger data synchronization"""
        sync_request = {
            'event_type': 'data_sync_requested',
            'source_event': event,
            'sync_targets': self._determine_sync_targets(event),
            'timestamp': datetime.utcnow().isoformat()
        }

        await self.kafka_producer.send(
            'adapter.sync.requests',
            value=sync_request
        )

        logger.debug("Triggered data sync")

    async def _trigger_cache_invalidation(self, event: Dict[str, Any]) -> None:
        """Trigger cache invalidation"""
        invalidation_request = {
            'event_type': 'cache_invalidation_requested',
            'source_event': event,
            'patterns': self._determine_invalidation_patterns(event),
            'timestamp': datetime.utcnow().isoformat()
        }

        await self.kafka_producer.send(
            'cache.invalidation.requests',
            value=invalidation_request
        )

        logger.debug("Triggered cache invalidation")

    def _determine_warming_patterns(self, event: Dict[str, Any]) -> List[str]:
        """Determine which cache patterns to warm"""
        original_event = event.get('original_event', {})
        event_type = original_event.get('type', '')

        pattern_map = {
            'calculation_starting': ['medication_scoring', 'drug_interactions'],
            'validation_required': ['safety_analytics', 'contraindications'],
            'interaction_check': ['drug_interactions'],
            'alternatives_requested': ['drug_alternatives']
        }

        return pattern_map.get(event_type, [])

    def _determine_warming_priority(self, event: Dict[str, Any]) -> str:
        """Determine cache warming priority"""
        original_event = event.get('original_event', {})

        # High priority for safety-related events
        safety_events = ['validation_required', 'interaction_check', 'contraindication_check']
        if original_event.get('type') in safety_events:
            return 'high'

        # High priority for real-time calculation events
        if original_event.get('type') == 'calculation_starting':
            return 'high'

        return 'normal'

    def _determine_sync_targets(self, event: Dict[str, Any]) -> List[str]:
        """Determine which data stores need synchronization"""
        original_event = event.get('original_event', {})
        event_type = original_event.get('type', '')

        if event_type == 'kb_updated':
            kb_source = original_event.get('kb_source', '')

            # Map KB sources to data stores
            if kb_source in ['KB-4', 'KB-5', 'KB-7']:
                return ['neo4j_semantic']
            elif kb_source in ['KB-3', 'KB-6']:
                return ['clickhouse']
            elif kb_source in ['KB-1', 'KB-2']:
                return ['postgres', 'elasticsearch']

        elif event_type == 'medications_changed':
            return ['neo4j_patient']

        return []

    def _determine_invalidation_patterns(self, event: Dict[str, Any]) -> List[str]:
        """Determine which cache patterns to invalidate"""
        original_event = event.get('original_event', {})
        event_type = original_event.get('type', '')

        if event_type == 'kb_updated':
            return ['terminology_lookup', 'drug_interactions', 'medication_scoring']
        elif event_type == 'concept_updated':
            return ['terminology_lookup']
        elif event_type == 'configuration_changed':
            return ['all']  # Invalidate all caches

        return []

    def _generate_orchestration_id(self, service_id: str, event: Dict[str, Any]) -> str:
        """Generate unique orchestration ID"""
        data = f"{service_id}:{event.get('type')}:{datetime.utcnow().isoformat()}"
        return hashlib.md5(data.encode()).hexdigest()[:16]

    async def publish_service_event(self, service_id: str, event: Dict[str, Any]) -> str:
        """
        Services publish events that trigger downstream actions

        Args:
            service_id: ID of the service publishing the event
            event: Event data

        Returns:
            Orchestration ID for tracking
        """
        # Enrich event with routing info
        enriched_event = await self._enrich_event(service_id, event)

        # Publish to service-specific topic
        topic = f"{service_id}.events"
        await self.kafka_producer.send(
            topic,
            key=service_id,
            value=enriched_event
        )

        orchestration_id = enriched_event['orchestration_id']
        logger.info(f"Published service event {orchestration_id} from {service_id}")

        return orchestration_id

    async def get_orchestration_statistics(self) -> Dict[str, Any]:
        """Get event orchestration statistics"""
        uptime = datetime.utcnow() - self.processing_stats['start_time']

        return {
            'processing_stats': self.processing_stats,
            'uptime_seconds': uptime.total_seconds(),
            'events_per_second': (
                self.processing_stats['events_processed'] / max(uptime.total_seconds(), 1)
            ),
            'error_rate': (
                self.processing_stats['errors'] /
                max(self.processing_stats['events_processed'], 1)
            ),
            'timestamp': datetime.utcnow().isoformat()
        }

    async def health_check(self) -> Dict[str, Any]:
        """Check health of event bus orchestrator"""
        try:
            # Check Kafka connectivity
            metadata = await self.kafka_producer.client.cluster.metadata()

            return {
                'status': 'healthy',
                'kafka_brokers': len(metadata.brokers),
                'active_topics': len(metadata.topics),
                'processing_stats': self.processing_stats,
                'timestamp': datetime.utcnow().isoformat()
            }
        except Exception as e:
            return {
                'status': 'unhealthy',
                'error': str(e),
                'timestamp': datetime.utcnow().isoformat()
            }