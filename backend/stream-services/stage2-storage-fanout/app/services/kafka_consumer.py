"""
Kafka Consumer Service for Stage 2: Storage Fan-Out

Consumes validated device data from Stage 1 and orchestrates FHIR transformation
and multi-sink writes with comprehensive DLQ handling.
"""

import asyncio
import json
import threading
from typing import Dict, Any, Optional

import structlog
from kafka import KafkaConsumer
from kafka.errors import KafkaError

from app.config import settings, get_kafka_config
from app.services.fhir_transformation import FHIRTransformationService
from app.services.multi_sink_writer import MultiSinkWriterService

logger = structlog.get_logger(__name__)


class KafkaConsumerService:
    """
    Kafka Consumer for Stage 2 Storage Fan-Out
    
    Consumes validated device data from Stage 1, transforms to FHIR,
    and writes to multiple sinks with comprehensive error handling.
    """
    
    def __init__(self, fhir_transformer: FHIRTransformationService,
                 multi_sink_writer: MultiSinkWriterService):
        self.service_name = "stage2-storage-fanout"
        self.consumer = None
        self.fhir_transformer = fhir_transformer
        self.multi_sink_writer = multi_sink_writer
        self.running = False
        
        # Metrics
        self.total_messages = 0
        self.processed_messages = 0
        self.failed_messages = 0
        self.transformation_failures = 0
        self.sink_write_failures = 0
        
        logger.info("Kafka Consumer Service initialized",
                   input_topic=settings.KAFKA_INPUT_TOPIC,
                   consumer_group=settings.KAFKA_CONSUMER_GROUP)
    
    async def start_consuming(self):
        """Start consuming messages from Kafka in a separate thread"""
        try:
            # Initialize Kafka consumer
            kafka_config = get_kafka_config()
            self.consumer = KafkaConsumer(
                settings.KAFKA_INPUT_TOPIC,
                **kafka_config
            )

            self.running = True

            logger.info("Kafka consumer started",
                       topic=settings.KAFKA_INPUT_TOPIC,
                       consumer_group=settings.KAFKA_CONSUMER_GROUP)

            # Start the blocking consumer in a separate thread
            self.consumer_thread = threading.Thread(
                target=self._consume_messages_blocking,
                daemon=True
            )
            self.consumer_thread.start()

            logger.info("Kafka consumer thread started successfully")

        except Exception as e:
            logger.error("Failed to start Kafka consumer", error=str(e))
            raise

    def _consume_messages_blocking(self):
        """Blocking message consumption loop (runs in separate thread)"""
        try:
            # Main consumption loop
            while self.running:
                try:
                    # Poll for messages (blocking operation)
                    message_batch = self.consumer.poll(
                        timeout_ms=1000,
                        max_records=settings.KAFKA_MAX_POLL_RECORDS
                    )

                    if message_batch:
                        # Process messages synchronously in this thread
                        self._process_message_batch_sync(message_batch)

                    # Commit offsets
                    if settings.KAFKA_ENABLE_AUTO_COMMIT:
                        self.consumer.commit()

                except KafkaError as e:
                    logger.error("Kafka error during consumption", error=str(e))
                    import time
                    time.sleep(5)  # Back off on Kafka errors

                except Exception as e:
                    logger.error("Unexpected error during consumption", error=str(e))
                    import time
                    time.sleep(1)

        except Exception as e:
            logger.error("Failed in blocking consumer loop", error=str(e))
        finally:
            self._cleanup_consumer_sync()

    def _process_message_batch_sync(self, message_batch: Dict):
        """Process a batch of messages synchronously (for thread)"""
        for topic_partition, messages in message_batch.items():
            logger.debug("Processing message batch",
                        topic=topic_partition.topic,
                        partition=topic_partition.partition,
                        message_count=len(messages))

            for message in messages:
                try:
                    # Log message received
                    message_key = None
                    if message.key:
                        if isinstance(message.key, bytes):
                            message_key = message.key.decode('utf-8')
                        else:
                            message_key = str(message.key)

                    logger.info("📥 Received message from Kafka",
                               key=message_key,
                               offset=message.offset,
                               partition=message.partition)

                    # Decode message value
                    if isinstance(message.value, bytes):
                        enriched_data = message.value.decode('utf-8')
                    else:
                        enriched_data = str(message.value)

                    # Process the message synchronously
                    self._process_single_message_sync(enriched_data, message_key)

                except Exception as e:
                    logger.error("Error processing individual message",
                               error=str(e), offset=message.offset)

    def _process_single_message_sync(self, enriched_data: str, message_key: str):
        """Process a single message synchronously"""
        try:
            # Parse enriched device data
            enriched_reading = json.loads(enriched_data)

            logger.info("🔄 Processing enriched device data",
                       device_id=enriched_reading.get('device_id'),
                       reading_type=enriched_reading.get('reading_type'))

            # Transform to FHIR (synchronous call)
            fhir_data = self.fhir_transformer.transform_to_fhir_sync(enriched_reading)

            if fhir_data:
                logger.debug("🔄 FHIR transformation successful",
                           device_id=enriched_reading.get('device_id'),
                           fhir_data_length=len(fhir_data),
                           fhir_resource_type=json.loads(fhir_data).get('resourceType') if fhir_data else None)
                # Write to sinks (synchronous call)
                self.multi_sink_writer.write_to_sinks_sync(fhir_data, enriched_reading)
                logger.info("✅ Message processed successfully",
                           device_id=enriched_reading.get('device_id'))
            else:
                logger.warning("⚠️ FHIR transformation failed",
                             device_id=enriched_reading.get('device_id'))

        except json.JSONDecodeError as e:
            logger.error("❌ Invalid JSON in message", error=str(e))
        except Exception as e:
            logger.error("❌ Error processing message", error=str(e))

    async def _process_message_batch(self, message_batch: Dict):
        """Process a batch of messages"""
        for topic_partition, messages in message_batch.items():
            logger.debug("Processing message batch",
                        topic=topic_partition.topic,
                        partition=topic_partition.partition,
                        message_count=len(messages))
            
            for message in messages:
                await self._process_single_message(message)
    
    async def _process_single_message(self, message):
        """Process a single Kafka message"""
        try:
            self.total_messages += 1
            
            # Extract message data
            key = message.key.decode('utf-8') if message.key else None
            value = message.value.decode('utf-8') if message.value else None
            
            if not value:
                logger.warning("Empty message received", key=key)
                return
            
            # Parse validated device data
            try:
                validated_data = json.loads(value)
            except json.JSONDecodeError as e:
                logger.error("Invalid JSON in validated data", key=key, error=str(e))
                await self.multi_sink_writer.dlq_service.send_poison_message(
                    {"raw_message": value}, "Invalid JSON in validated data", 0, key
                )
                self.failed_messages += 1
                return
            
            # Extract device information
            device_id = validated_data.get("device_id", key)
            patient_id = validated_data.get("patient_id")
            is_critical = validated_data.get("is_critical_data", False)
            
            logger.debug("Processing validated device data",
                        device_id=device_id, patient_id=patient_id,
                        is_critical=is_critical)
            
            # Step 1: Transform to FHIR and UI formats
            fhir_data, ui_data = await self._transform_data(validated_data, device_id)
            
            if not fhir_data or not ui_data:
                logger.error("Data transformation failed", device_id=device_id)
                self.transformation_failures += 1
                self.failed_messages += 1
                return
            
            # Step 2: Write to all sinks
            sink_results = await self.multi_sink_writer.write_to_all_sinks(
                fhir_data, ui_data, validated_data
            )
            
            # Step 3: Check results and handle failures
            successful_sinks = sum(1 for success in sink_results.values() if success)
            total_sinks = len(sink_results)
            
            if successful_sinks == total_sinks:
                self.processed_messages += 1
                logger.debug("Message processed successfully",
                            device_id=device_id, sink_results=sink_results)
            elif successful_sinks > 0:
                # Partial success
                self.processed_messages += 1
                logger.warning("Partial sink write success",
                              device_id=device_id, sink_results=sink_results)
            else:
                # Complete failure
                self.failed_messages += 1
                self.sink_write_failures += 1
                logger.error("All sink writes failed",
                            device_id=device_id, sink_results=sink_results)
                
                # Send to DLQ as poison message if all sinks failed
                await self.multi_sink_writer.dlq_service.send_poison_message(
                    validated_data, "All sink writes failed", 1, device_id
                )
            
        except Exception as e:
            self.failed_messages += 1
            logger.error("Error processing message", error=str(e), key=key)
            
            # Send unexpected errors to DLQ
            try:
                await self.multi_sink_writer.dlq_service.send_poison_message(
                    {"raw_message": value if 'value' in locals() else None},
                    f"Unexpected processing error: {str(e)}", 0,
                    key
                )
            except Exception as dlq_error:
                logger.error("Failed to send error to DLQ", error=str(dlq_error))
    
    async def _transform_data(self, validated_data: Dict[str, Any], 
                            device_id: str) -> tuple[Optional[str], Optional[str]]:
        """Transform validated data to FHIR and UI formats"""
        try:
            # Transform to FHIR Observation
            fhir_data = self.fhir_transformer.create_fhir_observation_from_device_data(validated_data)
            
            # Transform to UI document
            ui_data = self.fhir_transformer.create_ui_reading_from_device_data(validated_data)
            
            return fhir_data, ui_data
            
        except Exception as e:
            logger.error("Data transformation failed", device_id=device_id, error=str(e))
            
            # Send transformation failure to DLQ
            await self.multi_sink_writer.dlq_service.send_fhir_transformation_failure(
                validated_data, e, device_id
            )
            
            return None, None
    
    async def stop(self):
        """Stop the Kafka consumer"""
        logger.info("Stopping Kafka consumer")
        self.running = False
        
        # Give some time for current processing to complete
        await asyncio.sleep(2)
        
        await self._cleanup_consumer()
    
    async def _cleanup_consumer(self):
        """Cleanup Kafka consumer resources"""
        if self.consumer:
            try:
                self.consumer.close()
                logger.info("Kafka consumer closed")
            except Exception as e:
                logger.error("Error closing Kafka consumer", error=str(e))

    def _cleanup_consumer_sync(self):
        """Synchronous cleanup for use in thread"""
        if self.consumer:
            try:
                self.consumer.close()
                logger.info("Kafka consumer closed (sync)")
            except Exception as e:
                logger.error("Error closing Kafka consumer (sync)", error=str(e))
            finally:
                self.consumer = None
    
    def get_metrics(self) -> Dict[str, Any]:
        """Get consumer metrics"""
        return {
            "service_name": self.service_name,
            "total_messages": self.total_messages,
            "processed_messages": self.processed_messages,
            "failed_messages": self.failed_messages,
            "transformation_failures": self.transformation_failures,
            "sink_write_failures": self.sink_write_failures,
            "success_rate": self.processed_messages / max(self.total_messages, 1),
            "is_running": self.running,
            "input_topic": settings.KAFKA_INPUT_TOPIC,
            "consumer_group": settings.KAFKA_CONSUMER_GROUP
        }
    
    def is_healthy(self) -> bool:
        """Check if consumer is healthy"""
        return (self.running and 
                self.consumer is not None and
                self.fhir_transformer.is_healthy() and
                self.multi_sink_writer.is_healthy())
    
    def get_consumer_lag(self) -> Dict[str, Any]:
        """Get consumer lag information"""
        try:
            if not self.consumer:
                return {"error": "Consumer not initialized"}
            
            # Get partition assignments
            partitions = self.consumer.assignment()
            
            if not partitions:
                return {"error": "No partitions assigned"}
            
            # Get current positions and high water marks
            lag_info = {}
            for partition in partitions:
                try:
                    current_offset = self.consumer.position(partition)
                    high_water_mark = self.consumer.highwater(partition)
                    lag = high_water_mark - current_offset
                    
                    lag_info[f"partition_{partition.partition}"] = {
                        "current_offset": current_offset,
                        "high_water_mark": high_water_mark,
                        "lag": lag
                    }
                except Exception as e:
                    lag_info[f"partition_{partition.partition}"] = {
                        "error": str(e)
                    }
            
            return lag_info
            
        except Exception as e:
            return {"error": f"Failed to get consumer lag: {str(e)}"}
