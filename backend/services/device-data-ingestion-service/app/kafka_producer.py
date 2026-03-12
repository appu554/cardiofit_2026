"""
Kafka producer for device data ingestion
"""
import json
import logging
from typing import Dict, Any, Optional, List
from datetime import datetime, timezone
import asyncio
from concurrent.futures import ThreadPoolExecutor

# Try to import confluent_kafka, fallback if not available
try:
    from confluent_kafka import Producer
    from confluent_kafka.error import KafkaError, KafkaException
    KAFKA_AVAILABLE = True
except ImportError:
    Producer = None
    KafkaError = None
    KafkaException = Exception  # Fallback to base Exception
    KAFKA_AVAILABLE = False

from app.config import settings
from app.resilience.circuit_breaker_manager import (
    circuit_breaker_manager,
    ServiceType,
    CircuitBreakerOpenError
)
from app.performance.adaptive_batching import AdaptiveBatchManager, BatchConfig
from app.envelope.envelope_factory import EnhancedEnvelopeFactory, AuthContext, RequestContext

logger = logging.getLogger(__name__)


class DeviceDataKafkaProducer:
    """Enhanced Kafka producer for device data with circuit breaker protection and adaptive batching"""

    def __init__(self):
        self.producer = None
        self.executor = ThreadPoolExecutor(max_workers=4)
        self.batch_manager: Optional[AdaptiveBatchManager] = None
        self.batching_enabled = True
        self.batching_initialized = False
        self.envelope_factory = None  # Will be initialized async
        self._initialize_producer()
        # Don't initialize batching in constructor - do it lazily
    
    def _initialize_producer(self):
        """Initialize Kafka producer with configuration and fallback support"""
        if not KAFKA_AVAILABLE:
            logger.warning("Confluent Kafka module not installed, running in fallback mode")
            logger.info("Install confluent-kafka with: pip install confluent-kafka")
            logger.info("Kafka operations will be skipped - messages will not be published")
            self.producer = None
            self.batching_enabled = False
            return

        try:
            config = {
                'bootstrap.servers': settings.KAFKA_BOOTSTRAP_SERVERS,
                'security.protocol': 'SASL_SSL',
                'sasl.mechanism': 'PLAIN',
                'sasl.username': settings.KAFKA_API_KEY,
                'sasl.password': settings.KAFKA_API_SECRET,
                'client.id': 'device-data-ingestion-service',
                'acks': 'all',  # Wait for all replicas to acknowledge
                'retries': 3,
                'retry.backoff.ms': 1000,
                'request.timeout.ms': 30000,
                'delivery.timeout.ms': 60000,
                'batch.size': 16384,
                'linger.ms': 10,  # Small delay to batch messages
                'compression.type': 'snappy'
            }
            
            self.producer = Producer(config)
            logger.info("Kafka producer initialized successfully")
            
        except Exception as e:
            logger.error(f"Failed to initialize Kafka producer: {e}")
            raise

    async def _initialize_batching(self):
        """Initialize adaptive batching system"""
        try:
            batch_config = BatchConfig(
                min_batch_size=1,
                max_batch_size=100,
                max_wait_time_ms=5000,  # 5 seconds
                enable_adaptive_sizing=True,
                target_latency_ms=100,
                throughput_threshold_msgs_per_sec=50
            )

            self.batch_manager = AdaptiveBatchManager(batch_config)
            await self.batch_manager.initialize(self._process_message_batch)

            # Initialize enhanced envelope factory with cache manager
            try:
                from app.cache.device_cache_manager import get_device_cache_manager
                cache_manager = await get_device_cache_manager()
                self.envelope_factory = EnhancedEnvelopeFactory("device-data-ingestion-service", cache_manager)
                await self.envelope_factory.initialize()
                logger.info("Enhanced envelope factory initialized with performance optimizations")
            except Exception as e:
                logger.warning(f"Failed to initialize envelope factory with cache: {e}")
                self.envelope_factory = EnhancedEnvelopeFactory("device-data-ingestion-service")
                await self.envelope_factory.initialize()
                logger.info("Enhanced envelope factory initialized without cache")

            logger.info("Adaptive batching system initialized")

        except Exception as e:
            logger.error(f"Failed to initialize batching system: {e}")
            self.batching_enabled = False
        finally:
            self.batching_initialized = True

    async def ensure_batching_initialized(self):
        """Ensure batching system is initialized"""
        if not self.batching_initialized:
            await self._initialize_batching()

    async def _process_message_batch(self, messages: List[Dict[str, Any]]):
        """Process a batch of messages"""
        try:
            # Process messages in parallel using thread pool
            tasks = []
            for message in messages:
                device_id = message.get('device_id', 'unknown')
                task = asyncio.get_event_loop().run_in_executor(
                    self.executor,
                    self._produce_message,
                    settings.KAFKA_TOPIC_DEVICE_DATA,
                    device_id,
                    json.dumps(message)
                )
                tasks.append(task)

            # Wait for all messages to be produced
            results = await asyncio.gather(*tasks, return_exceptions=True)

            # Count successes and failures
            successes = sum(1 for r in results if not isinstance(r, Exception))
            failures = len(results) - successes

            if failures > 0:
                logger.warning(f"Batch processing completed with {failures} failures out of {len(messages)} messages")
            else:
                logger.debug(f"Successfully processed batch of {len(messages)} messages")

        except Exception as e:
            logger.error(f"Batch processing failed: {e}")
            raise
    
    async def publish_device_data(
        self,
        device_reading: Dict[str, Any],
        key: Optional[str] = None,
        auth_context: Optional[Dict[str, Any]] = None,
        request_context: Optional[Dict[str, Any]] = None,
        use_enhanced_envelope: bool = True
    ) -> str:
        """
        Publish device reading data to Kafka topic
        
        Args:
            device_reading: Device reading data dictionary
            key: Optional message key (defaults to device_id)
            
        Returns:
            Message ID or correlation ID
        """
        if not KAFKA_AVAILABLE:
            logger.warning("Kafka not available, simulating message publish")
            return f"fallback:{key}:{datetime.utcnow().timestamp()}"

        if not self.producer:
            logger.error("Kafka producer not initialized")
            return f"error:{key}:{datetime.utcnow().timestamp()}"
        
        # Use device_id as key if not provided
        if key is None:
            key = device_reading.get('device_id', 'unknown')
        
        # Create message with enhanced envelope if requested
        if use_enhanced_envelope and auth_context and request_context:
            try:
                # Create enhanced envelope
                auth_ctx = AuthContext(auth_context)
                req_ctx = RequestContext(
                    timestamp=request_context.get('timestamp'),
                    source_ip=request_context.get('source_ip'),
                    user_agent=request_context.get('user_agent'),
                    request_id=request_context.get('request_id')
                )

                envelope = await self.envelope_factory.create_device_data_envelope(
                    device_reading, auth_ctx, req_ctx
                )

                message = envelope.to_dict()
                logger.debug(f"Created enhanced envelope for device {key}")

            except Exception as e:
                logger.warning(f"Failed to create enhanced envelope, using basic format: {e}")
                # Fallback to basic message format
                message = {
                    'data': device_reading,
                    'metadata': {
                        'ingestion_timestamp': datetime.now(timezone.utc).isoformat(),
                        'service': 'device-data-ingestion-service',
                        'version': '1.0.0'
                    }
                }
        else:
            # Basic message format
            message = {
                'data': device_reading,
                'metadata': {
                    'ingestion_timestamp': datetime.now(timezone.utc).isoformat(),
                    'service': 'device-data-ingestion-service',
                    'version': '1.0.0'
                }
            }

        try:
            # Ensure batching is initialized
            await self.ensure_batching_initialized()

            # Use adaptive batching if enabled and available
            if self.batching_enabled and self.batch_manager:
                success = await self.batch_manager.add_message(message)
                if success:
                    logger.debug(f"Added device data to batch for device {key}")
                    return f"batched:{key}:{datetime.utcnow().timestamp()}"
                else:
                    logger.warning("Batching failed, falling back to direct publishing")

            # Fallback to direct publishing (or if batching disabled)
            async def kafka_publish_operation():
                loop = asyncio.get_event_loop()
                future = loop.run_in_executor(
                    self.executor,
                    self._produce_message,
                    settings.KAFKA_TOPIC_DEVICE_DATA,
                    key,
                    json.dumps(message)
                )
                return await future

            # Execute with circuit breaker protection
            result = await circuit_breaker_manager.call_with_circuit_breaker(
                service_name="kafka_producer",
                service_type=ServiceType.KAFKA_PRODUCER,
                func=kafka_publish_operation
            )

            logger.info(f"Published device data for device {key} to topic {settings.KAFKA_TOPIC_DEVICE_DATA}")
            return result

        except CircuitBreakerOpenError as e:
            logger.warning(f"Kafka circuit breaker is open, skipping message publication: {e}")
            # Return a placeholder result when circuit breaker is open
            return f"skipped:{key}:{datetime.now().timestamp()}"
        except Exception as e:
            logger.error(f"Failed to publish device data: {e}")
            raise
    
    def _produce_message(self, topic: str, key: str, value: str) -> str:
        """Synchronous message production (runs in thread pool)"""
        try:
            # Produce message
            self.producer.produce(
                topic=topic,
                key=key,
                value=value,
                callback=self._delivery_callback
            )
            
            # Trigger delivery (non-blocking)
            self.producer.poll(0)
            
            return f"{topic}:{key}:{datetime.utcnow().timestamp()}"
            
        except KafkaException as e:
            logger.error(f"Kafka exception during message production: {e}")
            raise
        except Exception as e:
            logger.error(f"Unexpected error during message production: {e}")
            raise
    
    def _delivery_callback(self, err, msg):
        """Callback for message delivery confirmation"""
        if err:
            logger.error(f"Message delivery failed: {err}")
        else:
            logger.debug(f"Message delivered to {msg.topic()} [{msg.partition()}] at offset {msg.offset()}")
    
    def flush(self, timeout: float = 10.0) -> int:
        """Flush all pending messages"""
        if self.producer:
            return self.producer.flush(timeout)
        return 0
    
    def close(self):
        """Close the producer and cleanup resources"""
        if self.producer:
            self.producer.flush(10.0)  # Wait up to 10 seconds for pending messages
            self.producer = None
        
        if self.executor:
            self.executor.shutdown(wait=True)
        
        logger.info("Kafka producer closed")
    
    def health_check(self) -> bool:
        """Check if Kafka producer is healthy"""
        try:
            if not self.producer:
                return False
            
            # Try to get metadata (this will fail if connection is broken)
            metadata = self.producer.list_topics(timeout=5.0)
            return True
            
        except Exception as e:
            logger.warning(f"Kafka health check failed: {e}")
            return False


# Global producer instance
_producer_instance: Optional[DeviceDataKafkaProducer] = None


async def get_kafka_producer() -> DeviceDataKafkaProducer:
    """Get or create global Kafka producer instance"""
    global _producer_instance

    if _producer_instance is None:
        _producer_instance = DeviceDataKafkaProducer()

    # Ensure batching is initialized
    await _producer_instance.ensure_batching_initialized()

    return _producer_instance


def close_kafka_producer():
    """Close global Kafka producer instance"""
    global _producer_instance
    
    if _producer_instance:
        _producer_instance.close()
        _producer_instance = None
