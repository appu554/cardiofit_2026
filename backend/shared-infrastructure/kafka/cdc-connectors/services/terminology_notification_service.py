"""
Terminology Release Notification Service

Consumes CDC events from kb7.terminology.releases topic and notifies
downstream services when new terminology versions become ACTIVE.

Architecture Flow:
1. KB-7 Knowledge Factory Pipeline → GraphDB load complete
2. Pipeline commits to kb_releases table (Commit-Last Strategy)
3. Debezium captures INSERT/UPDATE → publishes to Kafka
4. This service receives event → notifies downstream consumers

Notification Methods:
- HTTP webhooks to KB services (KB1-KB7)
- Redis pub/sub for real-time cache invalidation
- Kafka topic for async notification propagation

@author CDC Integration Team
@version 1.0
@since 2025-12-03
"""

import json
import logging
import asyncio
import aiohttp
import redis.asyncio as redis
from typing import Dict, List, Optional, Any, Callable
from datetime import datetime
from dataclasses import dataclass, asdict
from kafka import KafkaConsumer, KafkaProducer
from kafka.errors import KafkaError
import threading
import os

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


@dataclass
class TerminologyReleaseEvent:
    """Parsed terminology release event from CDC"""
    operation: str  # c=create, u=update, d=delete, r=read (snapshot)
    version_id: str
    status: str  # PENDING, LOADING, ACTIVE, ARCHIVED, FAILED
    snomed_version: Optional[str] = None
    rxnorm_version: Optional[str] = None
    loinc_version: Optional[str] = None
    triple_count: Optional[int] = None
    graphdb_endpoint: Optional[str] = None
    graphdb_repository: Optional[str] = None
    gcs_uri: Optional[str] = None
    timestamp_ms: Optional[int] = None

    @classmethod
    def from_debezium(cls, cdc_event: Dict[str, Any]) -> Optional['TerminologyReleaseEvent']:
        """Parse Debezium CDC event into TerminologyReleaseEvent"""
        try:
            payload = cdc_event.get('payload', cdc_event)  # Handle wrapped/unwrapped

            operation = payload.get('op', 'u')

            # Get the 'after' state for creates/updates, 'before' for deletes
            data = payload.get('after') or payload.get('before') or payload

            if not data:
                return None

            return cls(
                operation=operation,
                version_id=data.get('version_id'),
                status=data.get('status'),
                snomed_version=data.get('snomed_version'),
                rxnorm_version=data.get('rxnorm_version'),
                loinc_version=data.get('loinc_version'),
                triple_count=data.get('triple_count'),
                graphdb_endpoint=data.get('graphdb_endpoint'),
                graphdb_repository=data.get('graphdb_repository'),
                gcs_uri=data.get('gcs_uri'),
                timestamp_ms=payload.get('ts_ms')
            )
        except Exception as e:
            logger.error(f"Failed to parse CDC event: {e}")
            return None

    def is_active_transition(self) -> bool:
        """Check if this represents a transition to ACTIVE status"""
        return self.status == 'ACTIVE' and self.operation in ('c', 'u', 'r')


class TerminologyNotificationService:
    """
    Service for notifying downstream consumers of terminology version changes.
    Implements multiple notification strategies for reliability.
    """

    # KB Service webhook endpoints
    KB_SERVICE_WEBHOOKS = {
        'KB1': os.getenv('KB1_WEBHOOK_URL', 'http://localhost:8081/webhooks/terminology-update'),
        'KB2': os.getenv('KB2_WEBHOOK_URL', 'http://localhost:8086/webhooks/terminology-update'),
        'KB3': os.getenv('KB3_WEBHOOK_URL', 'http://localhost:8087/webhooks/terminology-update'),
        'KB4': os.getenv('KB4_WEBHOOK_URL', 'http://localhost:8088/webhooks/terminology-update'),
        'KB5': os.getenv('KB5_WEBHOOK_URL', 'http://localhost:8089/webhooks/terminology-update'),
        'KB6': os.getenv('KB6_WEBHOOK_URL', 'http://localhost:8091/webhooks/terminology-update'),
        'KB7': os.getenv('KB7_WEBHOOK_URL', 'http://localhost:8092/webhooks/terminology-update'),
    }

    def __init__(self):
        # Kafka configuration
        self.kafka_config = {
            'bootstrap_servers': os.getenv('KAFKA_BOOTSTRAP_SERVERS', 'localhost:9092').split(','),
            'security_protocol': os.getenv('KAFKA_SECURITY_PROTOCOL', 'PLAINTEXT'),
        }

        # Add SASL config if using Confluent Cloud
        if os.getenv('KAFKA_SASL_USERNAME'):
            self.kafka_config.update({
                'security_protocol': 'SASL_SSL',
                'sasl_mechanism': 'PLAIN',
                'sasl_plain_username': os.getenv('KAFKA_SASL_USERNAME'),
                'sasl_plain_password': os.getenv('KAFKA_SASL_PASSWORD'),
            })

        # Redis configuration
        self.redis_url = os.getenv('REDIS_URL', 'redis://localhost:6379')

        # Topics
        self.source_topic = 'kb7.terminology.releases'
        self.notification_topic = 'terminology.version.notifications'

        # State
        self.consumer = None
        self.producer = None
        self.redis_client = None
        self.consumer_thread = None
        self.running = False

        # Statistics
        self.stats = {
            'events_received': 0,
            'active_transitions': 0,
            'webhook_notifications_sent': 0,
            'webhook_failures': 0,
            'redis_notifications_sent': 0,
            'kafka_notifications_sent': 0,
            'last_event_time': None,
            'current_active_version': None
        }

    async def start(self):
        """Start the notification service"""
        try:
            logger.info("Starting Terminology Notification Service...")

            # Initialize Kafka consumer
            self.consumer = KafkaConsumer(
                self.source_topic,
                **self.kafka_config,
                group_id='terminology-notification-service',
                auto_offset_reset='earliest',
                enable_auto_commit=True,
                value_deserializer=lambda x: json.loads(x.decode('utf-8')) if x else None
            )

            # Initialize Kafka producer for notification topic
            self.producer = KafkaProducer(
                **self.kafka_config,
                value_serializer=lambda x: json.dumps(x).encode('utf-8')
            )

            # Initialize Redis client
            try:
                self.redis_client = await redis.from_url(self.redis_url)
                await self.redis_client.ping()
                logger.info("Redis connection established")
            except Exception as e:
                logger.warning(f"Redis not available, continuing without: {e}")
                self.redis_client = None

            # Start consumer thread
            self.running = True
            self.consumer_thread = threading.Thread(target=self._consume_events, daemon=True)
            self.consumer_thread.start()

            logger.info("Terminology Notification Service started successfully")
            logger.info(f"Consuming from: {self.source_topic}")
            logger.info(f"Publishing to: {self.notification_topic}")

        except Exception as e:
            logger.error(f"Failed to start service: {e}")
            raise

    async def stop(self):
        """Stop the notification service"""
        try:
            logger.info("Stopping Terminology Notification Service...")

            self.running = False

            if self.consumer:
                self.consumer.close()

            if self.producer:
                self.producer.close()

            if self.redis_client:
                await self.redis_client.close()

            if self.consumer_thread and self.consumer_thread.is_alive():
                self.consumer_thread.join(timeout=5)

            logger.info("Terminology Notification Service stopped")

        except Exception as e:
            logger.error(f"Error stopping service: {e}")

    def _consume_events(self):
        """Background thread for consuming CDC events"""
        logger.info(f"Starting CDC event consumption from {self.source_topic}")

        try:
            for message in self.consumer:
                if not self.running:
                    break

                try:
                    asyncio.run(self._process_cdc_event(message))
                except Exception as e:
                    logger.error(f"Error processing CDC event: {e}")

        except Exception as e:
            logger.error(f"Consumer error: {e}")

        logger.info("CDC event consumption stopped")

    async def _process_cdc_event(self, message):
        """Process a single CDC event from Kafka"""
        try:
            # Parse the Debezium event
            event = TerminologyReleaseEvent.from_debezium(message.value)

            if not event:
                logger.warning("Could not parse CDC event, skipping")
                return

            self.stats['events_received'] += 1
            self.stats['last_event_time'] = datetime.utcnow().isoformat()

            logger.info(f"Received terminology event: version={event.version_id}, status={event.status}, op={event.operation}")

            # Check if this is a transition to ACTIVE
            if event.is_active_transition():
                logger.info(f"ACTIVE transition detected for version {event.version_id}")
                self.stats['active_transitions'] += 1
                self.stats['current_active_version'] = event.version_id

                # Notify all downstream consumers
                await self._notify_all(event)

        except Exception as e:
            logger.error(f"Error processing CDC event: {e}")

    async def _notify_all(self, event: TerminologyReleaseEvent):
        """Send notifications via all channels"""

        # Create notification payload
        notification = {
            'event_type': 'terminology.version.activated',
            'version_id': event.version_id,
            'snomed_version': event.snomed_version,
            'rxnorm_version': event.rxnorm_version,
            'loinc_version': event.loinc_version,
            'triple_count': event.triple_count,
            'graphdb_endpoint': event.graphdb_endpoint,
            'graphdb_repository': event.graphdb_repository,
            'timestamp': datetime.utcnow().isoformat(),
            'source': 'terminology-notification-service'
        }

        # Run all notification methods concurrently
        await asyncio.gather(
            self._notify_via_webhooks(notification),
            self._notify_via_redis(notification),
            self._notify_via_kafka(notification),
            return_exceptions=True
        )

    async def _notify_via_webhooks(self, notification: Dict[str, Any]):
        """Send HTTP webhook notifications to KB services"""
        logger.info("Sending webhook notifications to KB services...")

        async with aiohttp.ClientSession() as session:
            tasks = []
            for kb_name, webhook_url in self.KB_SERVICE_WEBHOOKS.items():
                tasks.append(self._send_webhook(session, kb_name, webhook_url, notification))

            results = await asyncio.gather(*tasks, return_exceptions=True)

            for kb_name, result in zip(self.KB_SERVICE_WEBHOOKS.keys(), results):
                if isinstance(result, Exception):
                    logger.warning(f"Webhook to {kb_name} failed: {result}")
                    self.stats['webhook_failures'] += 1
                else:
                    self.stats['webhook_notifications_sent'] += 1

    async def _send_webhook(self, session: aiohttp.ClientSession, kb_name: str,
                           url: str, payload: Dict[str, Any]) -> bool:
        """Send a single webhook notification"""
        try:
            async with session.post(
                url,
                json=payload,
                timeout=aiohttp.ClientTimeout(total=10),
                headers={'Content-Type': 'application/json'}
            ) as response:
                if response.status < 300:
                    logger.debug(f"Webhook to {kb_name} succeeded: {response.status}")
                    return True
                else:
                    logger.warning(f"Webhook to {kb_name} returned {response.status}")
                    return False
        except aiohttp.ClientError as e:
            # Service might not have webhook endpoint yet, log at debug level
            logger.debug(f"Webhook to {kb_name} failed (service may not support webhooks): {e}")
            return False

    async def _notify_via_redis(self, notification: Dict[str, Any]):
        """Send notification via Redis pub/sub for real-time subscribers"""
        if not self.redis_client:
            return

        try:
            channel = 'terminology:version:updated'
            message = json.dumps(notification)

            await self.redis_client.publish(channel, message)

            # Also update the current version key for polling clients
            await self.redis_client.set(
                'terminology:current_version',
                json.dumps({
                    'version_id': notification['version_id'],
                    'snomed_version': notification['snomed_version'],
                    'rxnorm_version': notification['rxnorm_version'],
                    'loinc_version': notification['loinc_version'],
                    'updated_at': notification['timestamp']
                })
            )

            self.stats['redis_notifications_sent'] += 1
            logger.info(f"Redis notification published to {channel}")

        except Exception as e:
            logger.warning(f"Redis notification failed: {e}")

    async def _notify_via_kafka(self, notification: Dict[str, Any]):
        """Publish notification to Kafka topic for async consumers"""
        try:
            self.producer.send(self.notification_topic, notification)
            self.producer.flush()

            self.stats['kafka_notifications_sent'] += 1
            logger.info(f"Kafka notification published to {self.notification_topic}")

        except Exception as e:
            logger.warning(f"Kafka notification failed: {e}")

    async def get_statistics(self) -> Dict[str, Any]:
        """Get service statistics"""
        return {
            'service': 'terminology-notification-service',
            'running': self.running,
            'source_topic': self.source_topic,
            'notification_topic': self.notification_topic,
            'statistics': self.stats
        }

    async def get_current_version(self) -> Optional[Dict[str, Any]]:
        """Get the current active terminology version from Redis"""
        if not self.redis_client:
            return {'version_id': self.stats['current_active_version']}

        try:
            data = await self.redis_client.get('terminology:current_version')
            if data:
                return json.loads(data)
        except Exception as e:
            logger.warning(f"Failed to get current version from Redis: {e}")

        return {'version_id': self.stats['current_active_version']}


async def main():
    """Main entry point for running the service"""
    service = TerminologyNotificationService()

    try:
        await service.start()

        # Keep running until interrupted
        while True:
            await asyncio.sleep(60)
            stats = await service.get_statistics()
            logger.info(f"Service stats: {json.dumps(stats['statistics'], indent=2)}")

    except KeyboardInterrupt:
        logger.info("Received shutdown signal")
    finally:
        await service.stop()


if __name__ == '__main__':
    asyncio.run(main())
