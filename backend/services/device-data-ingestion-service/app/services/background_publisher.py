
# Global Outbox Service Integration Note:
# When using Global Outbox Service, the centralized publisher handles all event publishing.
# This background publisher serves as a fallback for local outbox processing when
# Global Outbox Service is unavailable.
"""
Background Publisher Service for Transactional Outbox Pattern

This service continuously polls outbox tables and publishes pending messages to Kafka,
ensuring guaranteed delivery and fault tolerance.
"""
import asyncio
import logging
from datetime import datetime, timedelta
from typing import Dict, List, Optional
import json

from app.services.outbox_service import VendorAwareOutboxService
from app.kafka_producer import get_kafka_producer
from app.core.monitoring import metrics_collector

logger = logging.getLogger(__name__)


class BackgroundPublisher:
    """
    Background service that continuously processes outbox messages
    and publishes them to Kafka with guaranteed delivery.
    """
    
    def __init__(self):
        self.outbox_service = VendorAwareOutboxService()
        self.kafka_producer = None
        self.metrics_collector = None
        self.is_running = False
        self.poll_interval = 2  # seconds - faster polling for demo
        self.batch_size = 50
        self.max_retries = 3
        
        # Vendor processing status
        self.vendor_status = {}
        
    async def initialize(self):
        """Initialize the background publisher"""
        try:
            # Initialize Kafka producer
            self.kafka_producer = await get_kafka_producer()

            # Initialize metrics collector
            self.metrics_collector = metrics_collector
            
            # Load vendor registry
            await self.outbox_service._load_vendor_registry()
            
            logger.info("Background publisher initialized successfully")
            return True
            
        except Exception as e:
            logger.error(f"Failed to initialize background publisher: {e}")
            return False
    
    async def start(self):
        """Start the background publisher in continuous loop"""
        if not await self.initialize():
            logger.error("Failed to initialize background publisher")
            return

        self.is_running = True
        logger.info(f"Background publisher started - polling every {self.poll_interval} seconds")

        try:
            while self.is_running:
                try:
                    # Process all vendors
                    await self.process_all_vendors()

                    # Wait before next poll
                    await asyncio.sleep(self.poll_interval)

                except Exception as e:
                    logger.error(f"Error in background publisher loop: {e}")
                    # Continue running even if there's an error
                    await asyncio.sleep(self.poll_interval)

        except asyncio.CancelledError:
            logger.info("Background publisher cancelled")
        except Exception as e:
            logger.error(f"Background publisher fatal error: {e}")
        finally:
            self.is_running = False
            logger.info("Background publisher stopped")
    
    async def stop(self):
        """Stop the background publisher"""
        self.is_running = False
        logger.info("Stopping background publisher...")
    
    async def process_all_vendors(self):
        """Process pending messages for all vendors"""
        vendor_registry = self.outbox_service.vendor_registry

        if not vendor_registry:
            logger.warning("No vendor registry loaded")
            return

        # Check if there are any pending messages first
        total_pending = 0
        try:
            queue_depths = await self.outbox_service.get_queue_depths()
            total_pending = sum(depth for depth in queue_depths.values() if depth > 0)

            if total_pending > 0:
                logger.info(f"Processing {total_pending} pending messages across all vendors")
            else:
                logger.debug("No pending messages to process")

        except Exception as e:
            logger.warning(f"Could not check queue depths: {e}")
        
        # Process each vendor concurrently
        tasks = []
        vendor_ids = []
        for vendor_id, vendor_config in vendor_registry.items():
            # Add vendor_id to config for processing
            vendor_config_with_id = vendor_config.copy()
            vendor_config_with_id["vendor_id"] = vendor_id
            vendor_ids.append(vendor_id)

            task = asyncio.create_task(
                self.process_vendor_messages(vendor_config_with_id)
            )
            tasks.append(task)

        # Wait for all vendor processing to complete
        results = await asyncio.gather(*tasks, return_exceptions=True)

        # Log results
        for i, result in enumerate(results):
            vendor_id = vendor_ids[i]
            if isinstance(result, Exception):
                logger.error(f"Error processing {vendor_id}: {result}")
            else:
                logger.debug(f"Processed {vendor_id}: {result} messages")
    
    async def process_vendor_messages(self, vendor_config: Dict) -> int:
        """Process pending messages for a specific vendor"""
        vendor_id = vendor_config["vendor_id"]
        processed_count = 0
        
        try:
            # Get pending messages for this vendor
            pending_messages = await self.outbox_service.get_pending_messages(
                vendor_id=vendor_id,
                limit=self.batch_size
            )
            
            if not pending_messages:
                return 0
            
            logger.info(f"Processing {len(pending_messages)} pending messages for {vendor_id}")
            
            # Process messages in batch
            for message in pending_messages:
                success = await self.process_single_message(vendor_id, message)
                if success:
                    processed_count += 1
            
            # Update vendor status
            self.vendor_status[vendor_id] = {
                "last_processed": datetime.utcnow(),
                "processed_count": processed_count,
                "pending_count": len(pending_messages) - processed_count
            }
            
            return processed_count
            
        except Exception as e:
            logger.error(f"Error processing vendor {vendor_id}: {e}")
            return 0
    
    async def process_single_message(self, vendor_id: str, message: Dict) -> bool:
        """Process a single outbox message"""
        message_id = str(message["id"])  # Convert UUID to string immediately

        try:
            # Mark message as processing
            await self.outbox_service.mark_message_processing(vendor_id, message_id)
            
            # Prepare Kafka message with proper serialization
            kafka_message = {
                "headers": {
                    "vendor_id": vendor_id,
                    "message_id": message_id,  # Already converted to string above
                    "correlation_id": str(message.get("correlation_id")) if message.get("correlation_id") else None,
                    "trace_id": str(message.get("trace_id")) if message.get("trace_id") else None,
                    "created_at": message["created_at"].isoformat() if isinstance(message["created_at"], datetime) else str(message["created_at"]),
                    "processed_at": datetime.utcnow().isoformat()
                },
                "payload": self._serialize_payload(message["event_payload"])
            }
            
            # Publish to Kafka
            success = await self.publish_to_kafka(
                topic=message["kafka_topic"],
                key=message.get("kafka_key"),
                value=kafka_message
            )
            
            if success:
                # Mark message as completed
                await self.outbox_service.mark_message_completed(vendor_id, message_id)
                
                # Emit success metrics
                await self.metrics_collector.emit_message_success(vendor_id)
                
                logger.debug(f"Successfully published message {message_id} for {vendor_id}")
                return True
            else:
                # Mark message as failed
                await self.outbox_service.mark_message_failed(
                    vendor_id, 
                    message_id, 
                    "Failed to publish to Kafka"
                )
                
                # Emit failure metrics
                await self.metrics_collector.emit_message_failure(vendor_id, "kafka_publish_failed")
                
                return False
                
        except Exception as e:
            logger.error(f"Error processing message {message_id} for {vendor_id}: {e}")
            
            # Mark message as failed
            await self.outbox_service.mark_message_failed(
                vendor_id, 
                message_id, 
                str(e)
            )
            
            return False
    
    def _serialize_payload(self, payload: any) -> any:
        """Serialize payload to ensure JSON compatibility"""
        import uuid
        from datetime import datetime, date

        if isinstance(payload, dict):
            return {key: self._serialize_payload(value) for key, value in payload.items()}
        elif isinstance(payload, list):
            return [self._serialize_payload(item) for item in payload]
        elif isinstance(payload, uuid.UUID):
            return str(payload)
        elif isinstance(payload, (datetime, date)):
            return payload.isoformat()
        elif hasattr(payload, '__dict__'):
            # Handle objects with attributes
            return str(payload)
        else:
            return payload

    async def publish_to_kafka(self, topic: str, key: Optional[str], value: Dict) -> bool:
        """Publish message to Kafka"""
        try:
            if not self.kafka_producer:
                logger.error("Kafka producer not initialized")
                return False

            # Use the correct method from DeviceDataKafkaProducer
            # The producer expects device_reading data, so we pass the payload
            result = await self.kafka_producer.publish_device_data(
                device_reading=value,
                key=key or "outbox_message"
            )

            # Check if publish was successful
            if result and not result.startswith("skipped:"):
                return True
            else:
                logger.warning(f"Kafka publish result: {result}")
                return False

        except Exception as e:
            logger.error(f"Failed to publish to Kafka: {e}")
            return False
    
    async def get_publisher_status(self) -> Dict:
        """Get current publisher status"""
        return {
            "is_running": self.is_running,
            "poll_interval": self.poll_interval,
            "batch_size": self.batch_size,
            "vendor_status": self.vendor_status,
            "last_check": datetime.utcnow().isoformat()
        }
    
    async def health_check(self) -> Dict:
        """Perform health check"""
        try:
            # Check outbox service
            outbox_health = await self.outbox_service.health_check()
            
            # Check Kafka producer
            kafka_healthy = self.kafka_producer is not None
            
            # Get queue depths
            queue_depths = await self.outbox_service.get_queue_depths()
            
            return {
                "status": "healthy" if outbox_health["status"] == "healthy" and kafka_healthy else "unhealthy",
                "outbox_service": outbox_health,
                "kafka_producer": {"status": "healthy" if kafka_healthy else "unhealthy"},
                "queue_depths": queue_depths,
                "publisher_status": await self.get_publisher_status()
            }
            
        except Exception as e:
            return {
                "status": "unhealthy",
                "error": str(e)
            }


# Global background publisher instance
background_publisher = BackgroundPublisher()


async def start_background_publisher():
    """Start the background publisher service"""
    await background_publisher.start()


async def stop_background_publisher():
    """Stop the background publisher service"""
    await background_publisher.stop()


async def get_background_publisher() -> BackgroundPublisher:
    """Get the global background publisher instance"""
    return background_publisher
