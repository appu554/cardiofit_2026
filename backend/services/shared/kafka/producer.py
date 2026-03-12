"""
Kafka event producer for Clinical Synthesis Hub
"""

import json
import logging
import time
from typing import Any, Dict, Optional, Union, Callable
from datetime import datetime, timezone
from uuid import uuid4
import asyncio
from concurrent.futures import ThreadPoolExecutor

try:
    from confluent_kafka import Producer
    from confluent_kafka.error import KafkaError, KafkaException
except ImportError:
    Producer = None
    KafkaError = None
    KafkaException = None

from .config import kafka_config, EventTypes
from .schemas import EventEnvelope
from .retry_policy import create_producer_resilient_operation, ResilientKafkaOperation
from .observability import get_metrics_collector, KafkaMetricsCollector

logger = logging.getLogger(__name__)

class EventProducer:
    """High-level event producer for Clinical Synthesis Hub"""

    def __init__(
        self,
        config: Optional[Dict[str, Any]] = None,
        service_name: str = "event-producer",
        enable_resilience: bool = True
    ):
        """Initialize the event producer"""
        if Producer is None:
            raise ImportError("confluent-kafka is required. Install with: pip install confluent-kafka")

        self.service_name = service_name
        self.config = config or kafka_config.get_producer_config()
        self.producer = Producer(self.config)
        self.executor = ThreadPoolExecutor(max_workers=4)

        # Resilience features
        self.resilient_operation = create_producer_resilient_operation() if enable_resilience else None

        # Observability
        self.metrics_collector = get_metrics_collector(service_name)

        # Legacy metrics (for backward compatibility)
        self.messages_sent = 0
        self.messages_failed = 0
        self.last_error = None

        # Update connection status
        self.metrics_collector.update_connection_status(True)

        logger.info("EventProducer initialized with config: %s",
                   {k: v for k, v in self.config.items() if 'password' not in k.lower()})
    
    def _delivery_callback(self, err: Optional[KafkaError], msg) -> None:
        """Callback for message delivery confirmation"""
        if err is not None:
            self.messages_failed += 1
            self.last_error = str(err)

            # Record metrics
            if msg:
                self.metrics_collector.record_message_produced(msg.topic(), False)

            logger.error("Message delivery failed: %s", err)
        else:
            self.messages_sent += 1

            # Record metrics
            self.metrics_collector.record_message_produced(msg.topic(), True)

            logger.debug("Message delivered to %s [%d] at offset %d",
                        msg.topic(), msg.partition(), msg.offset())
    
    def create_event_envelope(
        self,
        event_type: str,
        data: Dict[str, Any],
        source: str,
        subject: Optional[str] = None,
        correlation_id: Optional[str] = None,
        causation_id: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None
    ) -> EventEnvelope:
        """Create a standardized event envelope"""
        return EventEnvelope(
            id=str(uuid4()),
            source=source,
            type=event_type,
            subject=subject or f"{source}/{event_type}",
            time=datetime.now(timezone.utc).isoformat(),
            data=data,
            correlation_id=correlation_id,
            causation_id=causation_id,
            metadata=metadata or {},
            version="1.0"
        )
    
    def publish_event(
        self,
        topic: str,
        event_type: str,
        data: Dict[str, Any],
        source: str,
        key: Optional[str] = None,
        subject: Optional[str] = None,
        correlation_id: Optional[str] = None,
        causation_id: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
        headers: Optional[Dict[str, str]] = None,
        callback: Optional[Callable] = None
    ) -> str:
        """Publish an event to a Kafka topic"""

        def _publish_operation():
            # Create event envelope
            envelope = self.create_event_envelope(
                event_type=event_type,
                data=data,
                source=source,
                subject=subject,
                correlation_id=correlation_id,
                causation_id=causation_id,
                metadata=metadata
            )

            # Serialize to JSON
            envelope_dict = envelope.to_dict()
            message_value = json.dumps(envelope_dict, default=str)

            # Prepare headers
            event_headers = {
                'event-type': event_type,
                'event-source': source,
                'event-id': envelope.id,
                'content-type': 'application/json'
            }
            if headers:
                event_headers.update(headers)

            # Convert headers to bytes (required by confluent-kafka)
            kafka_headers = [(k, v.encode('utf-8')) for k, v in event_headers.items()]

            # Publish message
            self.producer.produce(
                topic=topic,
                key=key,
                value=message_value,
                headers=kafka_headers,
                callback=callback or self._delivery_callback
            )

            # Trigger delivery (non-blocking)
            self.producer.poll(0)

            logger.info("Event published: topic=%s, type=%s, id=%s",
                       topic, event_type, envelope.id)

            return envelope.id

        # Execute with resilience if enabled
        if self.resilient_operation:
            with self.metrics_collector.measure_operation(f"publish_event_{topic}"):
                try:
                    return self.resilient_operation.execute(_publish_operation)
                except Exception as e:
                    self.messages_failed += 1
                    self.last_error = str(e)
                    self.metrics_collector.record_message_produced(topic, False)
                    logger.error("Failed to publish event: %s", e)
                    raise
        else:
            with self.metrics_collector.measure_operation(f"publish_event_{topic}"):
                try:
                    return _publish_operation()
                except Exception as e:
                    self.messages_failed += 1
                    self.last_error = str(e)
                    self.metrics_collector.record_message_produced(topic, False)
                    logger.error("Failed to publish event: %s", e)
                    raise
    
    def publish_fhir_event(
        self,
        resource_type: str,
        operation: str,
        resource_id: str,
        resource_data: Dict[str, Any],
        source: str,
        correlation_id: Optional[str] = None,
        causation_id: Optional[str] = None
    ) -> str:
        """Publish a FHIR resource event"""
        
        # Determine topic based on resource type
        topic_map = {
            'Patient': 'fhir-patient-events',
            'Encounter': 'fhir-encounter-events',
            'Observation': 'fhir-observation-events',
            'Medication': 'fhir-medication-events',
            'MedicationRequest': 'fhir-medication-events',
            'ServiceRequest': 'fhir-order-events',
            'Condition': 'fhir-condition-events'
        }
        
        topic = topic_map.get(resource_type, 'fhir-generic-events')
        event_type = f"{resource_type.lower()}.{operation}"
        
        return self.publish_event(
            topic=topic,
            event_type=event_type,
            data={
                'resourceType': resource_type,
                'operation': operation,
                'resourceId': resource_id,
                'resource': resource_data
            },
            source=source,
            key=resource_id,
            subject=f"{resource_type}/{resource_id}",
            correlation_id=correlation_id,
            causation_id=causation_id,
            metadata={
                'fhir_version': 'R4',
                'resource_type': resource_type
            }
        )
    
    def flush(self, timeout: float = 10.0) -> int:
        """Flush all pending messages"""
        return self.producer.flush(timeout)
    
    def close(self):
        """Close the producer and clean up resources"""
        try:
            # Flush any remaining messages
            remaining = self.flush(timeout=10.0)
            if remaining > 0:
                logger.warning("Producer closed with %d messages still pending", remaining)

            # Note: confluent-kafka Producer doesn't have a close() method
            # It's automatically cleaned up when the object is destroyed

            self.executor.shutdown(wait=True)
            logger.info("EventProducer closed. Sent: %d, Failed: %d",
                       self.messages_sent, self.messages_failed)
        except Exception as e:
            logger.error("Error closing producer: %s", e)
    
    async def publish_event_async(
        self,
        topic: str,
        event_type: str,
        data: Dict[str, Any],
        source: str,
        **kwargs
    ) -> str:
        """Async wrapper for publish_event"""
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(
            self.executor,
            self.publish_event,
            topic, event_type, data, source,
            **kwargs
        )
    
    def get_stats(self) -> Dict[str, Any]:
        """Get producer statistics"""
        stats = {
            'messages_sent': self.messages_sent,
            'messages_failed': self.messages_failed,
            'last_error': self.last_error,
            'service_name': self.service_name
        }

        # Try to get producer stats safely
        try:
            metadata = self.producer.list_topics(timeout=1)
            stats['broker_count'] = len(metadata.brokers)
            stats['topic_count'] = len(metadata.topics)
        except Exception as e:
            stats['producer_error'] = str(e)

        return stats

# Global producer instance
_producer_instance = None

def get_event_producer() -> EventProducer:
    """Get global event producer instance"""
    global _producer_instance
    if _producer_instance is None:
        _producer_instance = EventProducer()
    return _producer_instance

def close_event_producer():
    """Close global event producer instance"""
    global _producer_instance
    if _producer_instance is not None:
        _producer_instance.close()
        _producer_instance = None
