"""
Kafka event consumer for Clinical Synthesis Hub
"""

import json
import logging
import signal
import sys
import time
from typing import Any, Dict, List, Optional, Callable, Set
from datetime import datetime
import asyncio
from concurrent.futures import ThreadPoolExecutor

try:
    from confluent_kafka import Consumer, TopicPartition
    from confluent_kafka.error import KafkaError, KafkaException
except ImportError:
    Consumer = None
    TopicPartition = None
    KafkaError = None
    KafkaException = None

from .config import kafka_config
from .schemas import EventEnvelope
from .retry_policy import create_consumer_resilient_operation, ResilientKafkaOperation
from .observability import get_metrics_collector, KafkaMetricsCollector

logger = logging.getLogger(__name__)

class EventConsumer:
    """High-level event consumer for Clinical Synthesis Hub"""
    
    def __init__(
        self,
        group_id: str,
        topics: List[str],
        config: Optional[Dict[str, Any]] = None,
        auto_commit: bool = False,
        service_name: Optional[str] = None,
        enable_resilience: bool = True
    ):
        """Initialize the event consumer"""
        if Consumer is None:
            raise ImportError("confluent-kafka is required. Install with: pip install confluent-kafka")

        self.group_id = group_id
        self.topics = topics
        self.auto_commit = auto_commit
        self.running = False
        self.handlers: Dict[str, Callable] = {}
        self.executor = ThreadPoolExecutor(max_workers=8)

        # Service identification
        self.service_name = service_name or f"consumer-{group_id}"

        # Get consumer configuration
        consumer_config = config or kafka_config.get_consumer_config(
            group_id=group_id,
            **{'enable.auto.commit': auto_commit}
        )

        self.consumer = Consumer(consumer_config)

        # Resilience features
        self.resilient_operation = create_consumer_resilient_operation() if enable_resilience else None

        # Observability
        self.metrics_collector = get_metrics_collector(self.service_name)

        # Legacy metrics (for backward compatibility)
        self.messages_processed = 0
        self.messages_failed = 0
        self.last_error = None

        # Setup signal handlers for graceful shutdown
        signal.signal(signal.SIGINT, self._signal_handler)
        signal.signal(signal.SIGTERM, self._signal_handler)

        # Update connection status
        self.metrics_collector.update_connection_status(True)

        logger.info("EventConsumer initialized: group=%s, topics=%s",
                   group_id, topics)
    
    def _signal_handler(self, signum, frame):
        """Handle shutdown signals"""
        logger.info("Received signal %d, shutting down gracefully...", signum)
        self.stop()
    
    def register_handler(self, event_type: str, handler: Callable[[EventEnvelope], None]):
        """Register an event handler for a specific event type"""
        self.handlers[event_type] = handler
        logger.info("Registered handler for event type: %s", event_type)
    
    def register_pattern_handler(self, pattern: str, handler: Callable[[EventEnvelope], None]):
        """Register an event handler for events matching a pattern"""
        # For now, we'll use simple string matching
        # In the future, we could use regex or more sophisticated matching
        self.handlers[f"pattern:{pattern}"] = handler
        logger.info("Registered pattern handler: %s", pattern)
    
    def _parse_event(self, message) -> Optional[EventEnvelope]:
        """Parse a Kafka message into an EventEnvelope"""
        try:
            # Parse JSON message
            data = json.loads(message.value().decode('utf-8'))
            
            # Create EventEnvelope
            envelope = EventEnvelope.from_dict(data)
            
            # Add message metadata
            envelope.metadata.update({
                'kafka_topic': message.topic(),
                'kafka_partition': message.partition(),
                'kafka_offset': message.offset(),
                'kafka_timestamp': message.timestamp()[1] if message.timestamp()[0] == 1 else None,
                'kafka_headers': {k: v.decode('utf-8') for k, v in (message.headers() or [])}
            })
            
            return envelope
            
        except Exception as e:
            logger.error("Failed to parse message: %s", e)
            self.messages_failed += 1
            self.last_error = str(e)
            return None
    
    def _find_handler(self, event_type: str) -> Optional[Callable]:
        """Find a handler for the given event type"""
        # Direct match
        if event_type in self.handlers:
            return self.handlers[event_type]
        
        # Pattern matching
        for pattern_key, handler in self.handlers.items():
            if pattern_key.startswith("pattern:"):
                pattern = pattern_key[8:]  # Remove "pattern:" prefix
                if pattern in event_type:
                    return handler
        
        return None
    
    def _process_message(self, message) -> bool:
        """Process a single message"""
        topic = message.topic()

        def _process_operation():
            # Parse event
            envelope = self._parse_event(message)
            if envelope is None:
                return False

            # Find handler
            handler = self._find_handler(envelope.type)
            if handler is None:
                logger.warning("No handler found for event type: %s", envelope.type)
                return True  # Not an error, just no handler

            # Execute handler
            handler(envelope)
            return True

        # Execute with resilience if enabled
        if self.resilient_operation:
            with self.metrics_collector.measure_operation(f"process_message_{topic}"):
                try:
                    success = self.resilient_operation.execute(_process_operation)
                    if success:
                        self.messages_processed += 1
                        self.metrics_collector.record_message_consumed(topic, True)
                    else:
                        self.messages_failed += 1
                        self.metrics_collector.record_message_consumed(topic, False)
                    return success
                except Exception as e:
                    self.messages_failed += 1
                    self.last_error = str(e)
                    self.metrics_collector.record_message_consumed(topic, False)
                    logger.error("Failed to process message: %s", e)
                    return False
        else:
            with self.metrics_collector.measure_operation(f"process_message_{topic}"):
                try:
                    success = _process_operation()
                    if success:
                        self.messages_processed += 1
                        self.metrics_collector.record_message_consumed(topic, True)
                    else:
                        self.messages_failed += 1
                        self.metrics_collector.record_message_consumed(topic, False)
                    return success
                except Exception as e:
                    self.messages_failed += 1
                    self.last_error = str(e)
                    self.metrics_collector.record_message_consumed(topic, False)
                    logger.error("Failed to process message: %s", e)
                    return False
    
    def start(self):
        """Start consuming messages"""
        self.running = True
        
        try:
            # Subscribe to topics
            self.consumer.subscribe(self.topics)
            logger.info("Started consuming from topics: %s", self.topics)
            
            while self.running:
                try:
                    # Poll for messages
                    message = self.consumer.poll(timeout=1.0)
                    
                    if message is None:
                        continue
                    
                    if message.error():
                        if message.error().code() == KafkaError._PARTITION_EOF:
                            logger.debug("Reached end of partition")
                            continue
                        else:
                            logger.error("Consumer error: %s", message.error())
                            continue
                    
                    # Process message
                    success = self._process_message(message)
                    
                    # Commit offset if not auto-committing and processing succeeded
                    if not self.auto_commit and success:
                        self.consumer.commit(message)
                        
                except KafkaException as e:
                    logger.error("Kafka exception: %s", e)
                    time.sleep(1)  # Brief pause before retrying
                    
                except Exception as e:
                    logger.error("Unexpected error: %s", e)
                    time.sleep(1)
                    
        except Exception as e:
            logger.error("Fatal error in consumer: %s", e)
            raise
        finally:
            self.close()
    
    async def start_async(self):
        """Start consuming messages asynchronously"""
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(self.executor, self.start)
    
    def stop(self):
        """Stop consuming messages"""
        self.running = False
        logger.info("Consumer stop requested")
    
    def close(self):
        """Close the consumer and clean up resources"""
        try:
            self.consumer.close()
            self.executor.shutdown(wait=True)
            logger.info("EventConsumer closed. Processed: %d, Failed: %d", 
                       self.messages_processed, self.messages_failed)
        except Exception as e:
            logger.error("Error closing consumer: %s", e)
    
    def get_stats(self) -> Dict[str, Any]:
        """Get consumer statistics"""
        return {
            'group_id': self.group_id,
            'topics': self.topics,
            'messages_processed': self.messages_processed,
            'messages_failed': self.messages_failed,
            'last_error': self.last_error,
            'running': self.running,
            'handlers': list(self.handlers.keys())
        }

    def register_fhir_handler(self, resource_type: str, handler: Callable[[EventEnvelope], None]):
        """Register handler for FHIR resource events"""
        patterns = [
            f"{resource_type.lower()}.created",
            f"{resource_type.lower()}.updated",
            f"{resource_type.lower()}.deleted"
        ]

        for pattern in patterns:
            self.register_handler(pattern, handler)

        logger.info("Registered FHIR handler for resource type: %s", resource_type)

class WorkerConsumer(EventConsumer):
    """Specialized consumer for worker processes"""

    def __init__(
        self,
        worker_name: str,
        topics: List[str],
        config: Optional[Dict[str, Any]] = None
    ):
        """Initialize worker consumer"""
        group_id = f"clinical-synthesis-hub-{worker_name}"
        super().__init__(group_id, topics, config, auto_commit=False)
        self.worker_name = worker_name

def create_consumer(
    group_id: str,
    topics: List[str],
    handlers: Dict[str, Callable],
    config: Optional[Dict[str, Any]] = None
) -> EventConsumer:
    """Factory function to create a configured consumer"""
    consumer = EventConsumer(group_id, topics, config)
    
    for event_type, handler in handlers.items():
        consumer.register_handler(event_type, handler)
    
    return consumer
