"""
Kafka client utilities for Evidence Envelope Service
Handles audit event publishing and envelope event streaming
"""

import json
import asyncio
from typing import Dict, Any, Optional, List
from datetime import datetime
import structlog
from aiokafka import AIOKafkaProducer
from aiokafka.errors import KafkaError

from .config import settings

logger = structlog.get_logger()


class KafkaProducer:
    """Async Kafka producer for audit and envelope events"""

    def __init__(self):
        self.producer: Optional[AIOKafkaProducer] = None
        self._is_connected: bool = False

    async def start(self) -> None:
        """Initialize Kafka producer"""
        try:
            self.producer = AIOKafkaProducer(
                bootstrap_servers=settings.KAFKA_BOOTSTRAP_SERVERS,
                client_id=settings.KAFKA_CLIENT_ID,
                compression_type=settings.KAFKA_COMPRESSION_TYPE,
                batch_size=settings.KAFKA_BATCH_SIZE,
                linger_ms=settings.KAFKA_LINGER_MS,
                acks=settings.KAFKA_ACKS,
                value_serializer=lambda v: json.dumps(
                    v, default=str, ensure_ascii=False
                ).encode('utf-8'),
                key_serializer=lambda k: str(k).encode('utf-8') if k else None,
                retry_backoff_ms=100,
                request_timeout_ms=30000,
                max_block_ms=60000
            )

            await self.producer.start()
            self._is_connected = True

            logger.info(
                "kafka_producer_started",
                bootstrap_servers=settings.KAFKA_BOOTSTRAP_SERVERS,
                client_id=settings.KAFKA_CLIENT_ID
            )

        except Exception as e:
            logger.error("kafka_producer_start_failed", error=str(e))
            self._is_connected = False
            raise

    async def stop(self) -> None:
        """Stop Kafka producer"""
        if self.producer:
            await self.producer.stop()
            self._is_connected = False
            logger.info("kafka_producer_stopped")

    def is_connected(self) -> bool:
        """Check if producer is connected"""
        return self._is_connected and self.producer is not None

    async def publish_audit_event(
        self,
        event_type: str,
        envelope_id: str,
        user_id: Optional[str],
        details: Dict[str, Any],
        timestamp: Optional[datetime] = None
    ) -> bool:
        """Publish audit event to Kafka"""
        if not self.is_connected():
            logger.error("kafka_producer_not_connected")
            return False

        try:
            audit_event = {
                "event_id": f"audit_{envelope_id}_{int(datetime.utcnow().timestamp() * 1000)}",
                "event_type": event_type,
                "envelope_id": envelope_id,
                "user_id": user_id,
                "timestamp": (timestamp or datetime.utcnow()).isoformat(),
                "service": settings.SERVICE_NAME,
                "version": settings.VERSION,
                "details": details,
                "compliance": {
                    "hipaa_applicable": True,
                    "fda_21cfr11": True,
                    "audit_level": "full"
                }
            }

            await self.producer.send_and_wait(
                topic=settings.KAFKA_TOPIC_AUDIT_EVENTS,
                key=envelope_id,
                value=audit_event
            )

            logger.info(
                "audit_event_published",
                event_type=event_type,
                envelope_id=envelope_id,
                topic=settings.KAFKA_TOPIC_AUDIT_EVENTS
            )

            return True

        except KafkaError as e:
            logger.error(
                "kafka_publish_failed",
                error=str(e),
                event_type=event_type,
                envelope_id=envelope_id
            )
            return False
        except Exception as e:
            logger.error(
                "audit_event_publish_failed",
                error=str(e),
                event_type=event_type,
                envelope_id=envelope_id
            )
            return False

    async def publish_envelope_event(
        self,
        event_type: str,
        envelope_id: str,
        envelope_data: Dict[str, Any],
        metadata: Optional[Dict[str, Any]] = None
    ) -> bool:
        """Publish envelope lifecycle event"""
        if not self.is_connected():
            logger.error("kafka_producer_not_connected")
            return False

        try:
            envelope_event = {
                "event_id": f"envelope_{envelope_id}_{int(datetime.utcnow().timestamp() * 1000)}",
                "event_type": event_type,
                "envelope_id": envelope_id,
                "timestamp": datetime.utcnow().isoformat(),
                "service": settings.SERVICE_NAME,
                "envelope_data": envelope_data,
                "metadata": metadata or {}
            }

            await self.producer.send_and_wait(
                topic=settings.KAFKA_TOPIC_ENVELOPE_EVENTS,
                key=envelope_id,
                value=envelope_event
            )

            logger.info(
                "envelope_event_published",
                event_type=event_type,
                envelope_id=envelope_id,
                topic=settings.KAFKA_TOPIC_ENVELOPE_EVENTS
            )

            return True

        except KafkaError as e:
            logger.error(
                "kafka_publish_failed",
                error=str(e),
                event_type=event_type,
                envelope_id=envelope_id
            )
            return False
        except Exception as e:
            logger.error(
                "envelope_event_publish_failed",
                error=str(e),
                event_type=event_type,
                envelope_id=envelope_id
            )
            return False

    async def publish_batch_events(
        self,
        events: List[Dict[str, Any]],
        topic: str
    ) -> int:
        """Publish multiple events in batch"""
        if not self.is_connected():
            logger.error("kafka_producer_not_connected")
            return 0

        successful_count = 0

        try:
            # Send all events concurrently
            tasks = []
            for event in events:
                key = event.get('envelope_id') or event.get('event_id')
                task = self.producer.send_and_wait(
                    topic=topic,
                    key=key,
                    value=event
                )
                tasks.append(task)

            # Wait for all to complete
            results = await asyncio.gather(*tasks, return_exceptions=True)

            # Count successful sends
            for result in results:
                if not isinstance(result, Exception):
                    successful_count += 1

            logger.info(
                "batch_events_published",
                total_events=len(events),
                successful_count=successful_count,
                topic=topic
            )

        except Exception as e:
            logger.error(
                "batch_publish_failed",
                error=str(e),
                topic=topic,
                event_count=len(events)
            )

        return successful_count


# Global Kafka producer instance
kafka_producer = KafkaProducer()