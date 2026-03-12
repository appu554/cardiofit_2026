"""
Base Kafka consumer for Module 8 projectors

Provides common functionality:
- Kafka consumer configuration with Confluent Cloud SSL/SASL
- Batch processing logic
- Automatic offset commit handling
- Error handling with DLQ support
- Graceful shutdown
"""

import json
import signal
import time
from abc import ABC, abstractmethod
from typing import Dict, List, Optional, Callable, Any

from kafka import KafkaConsumer, KafkaProducer
from kafka.errors import KafkaError
import structlog

from module8_shared.batch_processor import BatchProcessor
from module8_shared.metrics import ProjectorMetrics
from module8_shared.data_adapter import Module6DataAdapter


logger = structlog.get_logger(__name__)


class KafkaConsumerBase(ABC):
    """
    Abstract base class for all Module 8 storage projectors

    Subclasses must implement:
    - process_batch(): Process a batch of messages and write to storage
    - get_projector_name(): Return unique projector identifier
    """

    def __init__(
        self,
        kafka_config: Dict[str, Any],
        topics: List[str],
        batch_size: int = 100,
        batch_timeout_seconds: float = 5.0,
        dlq_topic: Optional[str] = None,
        message_deserializer: Optional[Callable] = None,
    ):
        """
        Initialize Kafka consumer base

        Args:
            kafka_config: Kafka consumer configuration dict
            topics: List of topics to consume from
            batch_size: Number of messages per batch
            batch_timeout_seconds: Max time to wait before flushing batch
            dlq_topic: Dead letter queue topic for failed messages
            message_deserializer: Custom deserializer function (default: JSON)
        """
        self.kafka_config = kafka_config
        self.topics = topics
        self.batch_size = batch_size
        self.batch_timeout_seconds = batch_timeout_seconds
        self.dlq_topic = dlq_topic

        # Default JSON deserializer with Module 6 adapter
        if message_deserializer is None:
            def default_deserializer(m):
                raw_data = json.loads(m.decode('utf-8'))
                # Apply Module 6 to Module 8 adapter
                adapted_data = Module6DataAdapter.adapt_event(raw_data)
                return adapted_data
            message_deserializer = default_deserializer
        self.message_deserializer = message_deserializer

        # Initialize batch processor
        self.batch_processor = BatchProcessor(
            batch_size=batch_size,
            batch_timeout_seconds=batch_timeout_seconds,
            flush_callback=self._flush_batch_callback,
        )

        # Initialize metrics
        self.metrics = ProjectorMetrics(projector_name=self.get_projector_name())

        # Consumer and producer instances
        self.consumer: Optional[KafkaConsumer] = None
        self.dlq_producer: Optional[KafkaProducer] = None

        # Shutdown flag
        self.shutdown_requested = False

        # Register signal handlers
        signal.signal(signal.SIGINT, self._handle_shutdown)
        signal.signal(signal.SIGTERM, self._handle_shutdown)

        logger.info(
            "Kafka consumer base initialized",
            projector=self.get_projector_name(),
            topics=topics,
            batch_size=batch_size,
            batch_timeout=batch_timeout_seconds,
        )

    @abstractmethod
    def process_batch(self, messages: List[Any]) -> None:
        """
        Process a batch of messages and write to storage

        Args:
            messages: List of deserialized message values

        Raises:
            Exception: If batch processing fails
        """
        pass

    @abstractmethod
    def get_projector_name(self) -> str:
        """Return unique projector identifier (e.g., 'postgresql-projector')"""
        pass

    def _convert_kafka_config(self, config: Dict[str, Any]) -> Dict[str, Any]:
        """Convert Confluent-style config to kafka-python style"""
        key_mapping = {
            'bootstrap.servers': 'bootstrap_servers',
            'group.id': 'group_id',
            'auto.offset.reset': 'auto_offset_reset',
            'enable.auto.commit': 'enable_auto_commit',
            'max.poll.records': 'max_poll_records',
            'max.poll.interval.ms': 'max_poll_interval_ms',
            'session.timeout.ms': 'session_timeout_ms',
            'security.protocol': 'security_protocol',
            'sasl.mechanism': 'sasl_mechanism',
            'sasl.username': 'sasl_plain_username',
            'sasl.password': 'sasl_plain_password',
        }

        converted = {}
        for key, value in config.items():
            new_key = key_mapping.get(key, key)
            # Convert security_protocol to uppercase enum value if needed
            if new_key == 'security_protocol' and value:
                converted[new_key] = value.upper()
            else:
                converted[new_key] = value

        return converted

    def start(self) -> None:
        """Start consuming messages from Kafka"""
        try:
            # Convert config to kafka-python style
            consumer_config = self._convert_kafka_config(self.kafka_config)

            # Initialize Kafka consumer
            self.consumer = KafkaConsumer(
                *self.topics,
                **consumer_config,
                value_deserializer=self.message_deserializer,
            )

            # Initialize DLQ producer if configured
            if self.dlq_topic:
                producer_config = {
                    'bootstrap_servers': consumer_config.get('bootstrap_servers'),
                    'value_serializer': lambda v: json.dumps(v).encode('utf-8'),
                }
                # Add security config if present
                if 'security_protocol' in consumer_config:
                    producer_config['security_protocol'] = consumer_config['security_protocol']
                if 'sasl_mechanism' in consumer_config:
                    producer_config['sasl_mechanism'] = consumer_config['sasl_mechanism']
                if 'sasl_plain_username' in consumer_config:
                    producer_config['sasl_plain_username'] = consumer_config['sasl_plain_username']
                if 'sasl_plain_password' in consumer_config:
                    producer_config['sasl_plain_password'] = consumer_config['sasl_plain_password']

                self.dlq_producer = KafkaProducer(**producer_config)

            logger.info(
                "Kafka consumer started",
                projector=self.get_projector_name(),
                topics=self.topics,
                group_id=consumer_config.get('group_id'),
            )

            # Main consumption loop
            for message in self.consumer:
                if self.shutdown_requested:
                    logger.info("Shutdown requested, stopping consumption")
                    break

                try:
                    # Update metrics
                    self.metrics.messages_consumed.inc()

                    # Add message to batch
                    self.batch_processor.add(message.value)

                    # Track consumer lag (optional monitoring, non-critical)
                    try:
                        from kafka import TopicPartition
                        tp = TopicPartition(message.topic, message.partition)
                        lag = self.consumer.highwater(tp) - message.offset
                        self.metrics.consumer_lag.set(lag)
                    except Exception:
                        # Lag tracking failed, continue processing
                        pass

                except Exception as e:
                    logger.error(
                        "Error processing message",
                        error=str(e),
                        topic=message.topic,
                        partition=message.partition,
                        offset=message.offset,
                    )
                    self.metrics.messages_failed.inc()
                    self.send_to_dlq(message)

        except Exception as e:
            logger.error("Fatal error in consumer loop", error=str(e))
            raise
        finally:
            self.shutdown()

    def _flush_batch_callback(self, batch: List[Any]) -> None:
        """
        Internal callback for batch processor

        Args:
            batch: List of messages to process
        """
        if not batch:
            return

        start_time = time.time()

        try:
            # Call subclass implementation
            self.process_batch(batch)

            # Commit offsets
            if self.consumer:
                self.consumer.commit()

            # Update metrics
            batch_size = len(batch)
            flush_duration = time.time() - start_time

            self.metrics.messages_processed.inc(batch_size)
            self.metrics.batch_size.observe(batch_size)
            self.metrics.batch_flush_duration.observe(flush_duration)

            logger.info(
                "Batch processed successfully",
                projector=self.get_projector_name(),
                batch_size=batch_size,
                flush_duration=flush_duration,
            )

        except Exception as e:
            logger.error(
                "Batch processing failed",
                error=str(e),
                batch_size=len(batch),
            )
            self.metrics.messages_failed.inc(len(batch))

            # Send all messages in batch to DLQ
            for message in batch:
                self.send_to_dlq_value(message)

            raise

    def send_to_dlq(self, message) -> None:
        """
        Send failed message to dead letter queue

        Args:
            message: Kafka message object
        """
        if not self.dlq_topic or not self.dlq_producer:
            logger.warning("DLQ not configured, dropping failed message")
            return

        try:
            dlq_payload = {
                "original_topic": message.topic,
                "original_partition": message.partition,
                "original_offset": message.offset,
                "original_key": message.key.decode('utf-8') if message.key else None,
                "original_value": message.value,
                "error_timestamp": int(time.time() * 1000),
                "projector": self.get_projector_name(),
            }

            self.dlq_producer.send(
                self.dlq_topic,
                key=message.key,
                value=dlq_payload,
            )

            logger.debug(
                "Message sent to DLQ",
                dlq_topic=self.dlq_topic,
                original_topic=message.topic,
            )

        except Exception as e:
            logger.error("Failed to send message to DLQ", error=str(e))

    def send_to_dlq_value(self, value: Any) -> None:
        """
        Send failed message value to DLQ (for batch processing failures)

        Args:
            value: Deserialized message value
        """
        if not self.dlq_topic or not self.dlq_producer:
            return

        try:
            dlq_payload = {
                "original_value": value,
                "error_timestamp": int(time.time() * 1000),
                "projector": self.get_projector_name(),
            }

            self.dlq_producer.send(self.dlq_topic, value=dlq_payload)

        except Exception as e:
            logger.error("Failed to send value to DLQ", error=str(e))

    def _handle_shutdown(self, signum, frame) -> None:
        """Handle shutdown signals"""
        logger.info(
            "Shutdown signal received",
            signal=signum,
            projector=self.get_projector_name(),
        )
        self.shutdown_requested = True

    def shutdown(self) -> None:
        """Gracefully shutdown consumer"""
        logger.info("Shutting down consumer", projector=self.get_projector_name())

        # Flush remaining batch
        if self.batch_processor:
            self.batch_processor.flush()

        # Close consumer
        if self.consumer:
            self.consumer.close()
            logger.info("Kafka consumer closed")

        # Close DLQ producer
        if self.dlq_producer:
            self.dlq_producer.close()
            logger.info("DLQ producer closed")

        logger.info("Shutdown complete")
