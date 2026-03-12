"""
Outbox Publisher Service for Transactional Outbox Pattern

Background service that:
- Polls vendor-specific outbox tables using SELECT FOR UPDATE SKIP LOCKED
- Publishes messages to Kafka with exponential backoff retry
- Handles dead letter processing and poison pill isolation
- Emits cloud-native metrics without database writes
- Supports concurrent processing per vendor for maximum throughput
"""
import asyncio
import logging
import time
from datetime import datetime, timedelta
from typing import List, Dict, Any, Optional
from concurrent.futures import ThreadPoolExecutor
import math

from app.services.outbox_service import VendorAwareOutboxService
from app.kafka_producer import get_kafka_producer
from app.core.monitoring import metrics_collector
from app.config import settings
from app.db.models import OutboxMessage, SUPPORTED_VENDORS

logger = logging.getLogger(__name__)


class OutboxPublisher:
    """
    Background service for publishing outbox messages to Kafka
    
    CRITICAL FEATURES:
    - Concurrent vendor processing with ThreadPoolExecutor
    - SELECT FOR UPDATE SKIP LOCKED for race condition prevention
    - Exponential backoff retry strategy
    - Cloud-native metrics emission
    - Poison pill isolation and dead letter handling
    """

    def __init__(self):
        self.outbox_service = VendorAwareOutboxService()
        self.kafka_producer = None
        self.batch_size = settings.OUTBOX_BATCH_SIZE or 50
        self.poll_interval = settings.OUTBOX_POLL_INTERVAL or 5  # seconds
        self.max_concurrent_vendors = settings.MAX_CONCURRENT_VENDORS or 10
        self.retry_backoff_base = settings.OUTBOX_RETRY_BACKOFF_SECONDS or 60
        
        # Publisher state
        self.is_running = False
        self.vendor_processors = {}
        self.health_status = {}
        
        # Performance tracking
        self.processing_stats = {
            "messages_processed": 0,
            "messages_succeeded": 0,
            "messages_failed": 0,
            "last_processing_time": None,
            "average_latency_ms": 0.0
        }

    async def initialize(self):
        """Initialize the publisher service"""
        try:
            # Initialize Kafka producer
            self.kafka_producer = await get_kafka_producer()
            logger.info("✅ Outbox publisher Kafka producer initialized")
            
            # Initialize outbox service
            await self.outbox_service._load_vendor_registry()
            logger.info("✅ Outbox publisher vendor registry loaded")
            
            # Initialize health status for all vendors
            for vendor_id in SUPPORTED_VENDORS.keys():
                self.health_status[vendor_id] = {
                    "is_healthy": True,
                    "last_successful_processing": datetime.utcnow(),
                    "consecutive_failures": 0,
                    "last_error": None
                }
            
            return True
            
        except Exception as e:
            logger.error(f"❌ Failed to initialize outbox publisher: {e}")
            return False

    async def start_publishing_loop(self):
        """Main publishing loop with concurrent vendor processing"""
        if self.is_running:
            logger.warning("Publisher service is already running")
            return
            
        self.is_running = True
        logger.info("🚀 Starting outbox publisher service")
        
        try:
            # Initialize the service
            if not await self.initialize():
                raise RuntimeError("Failed to initialize publisher service")
            
            # Main processing loop
            while self.is_running:
                try:
                    await self.process_all_vendors_concurrently()
                    await asyncio.sleep(self.poll_interval)
                    
                except Exception as e:
                    logger.error(f"Error in publishing loop: {e}")
                    await asyncio.sleep(self.poll_interval * 2)  # Back off on error
                    
        except Exception as e:
            logger.error(f"💥 Fatal error in publishing loop: {e}")
        finally:
            self.is_running = False
            logger.info("🛑 Outbox publisher service stopped")

    async def process_all_vendors_concurrently(self):
        """Process all vendors concurrently using ThreadPoolExecutor pattern"""
        start_time = time.time()
        
        # Get all supported vendors
        vendors = list(SUPPORTED_VENDORS.keys())
        
        # Create tasks for concurrent processing
        tasks = []
        for vendor_id in vendors:
            task = asyncio.create_task(self.process_vendor_messages(vendor_id))
            tasks.append(task)
        
        # Wait for all vendor processing to complete
        results = await asyncio.gather(*tasks, return_exceptions=True)
        
        # Process results and update health status
        total_processed = 0
        for i, result in enumerate(results):
            vendor_id = vendors[i]
            
            if isinstance(result, Exception):
                logger.error(f"Error processing vendor {vendor_id}: {result}")
                await self.update_vendor_health(vendor_id, False, str(result))
            else:
                processed_count = result or 0
                total_processed += processed_count
                await self.update_vendor_health(vendor_id, True, None)
        
        # Update processing stats
        end_time = time.time()
        processing_time_ms = (end_time - start_time) * 1000
        
        self.processing_stats["last_processing_time"] = datetime.utcnow()
        if total_processed > 0:
            self.processing_stats["messages_processed"] += total_processed
            self.processing_stats["average_latency_ms"] = processing_time_ms / total_processed
        
        # Emit batch processing metrics
        if total_processed > 0:
            await metrics_collector.emit_batch_metrics([
                {
                    "metric_type": "custom.googleapis.com/outbox/batch_processing_time_ms",
                    "value": processing_time_ms,
                    "labels": {"service": "outbox-publisher"}
                },
                {
                    "metric_type": "custom.googleapis.com/outbox/batch_messages_processed",
                    "value": total_processed,
                    "labels": {"service": "outbox-publisher"}
                }
            ])

    async def process_vendor_messages(self, vendor_id: str) -> int:
        """Process messages for a specific vendor"""
        try:
            # Get pending messages with lock
            async with metrics_collector.measure_processing_time(vendor_id):
                messages = await self.outbox_service.get_pending_messages_with_lock(
                    vendor_id, 
                    self.batch_size
                )
            
            if not messages:
                return 0
            
            logger.info(f"Processing {len(messages)} messages for vendor {vendor_id}")
            
            # Process each message
            processed_count = 0
            for message in messages:
                success = await self.process_single_message(message)
                if success:
                    processed_count += 1
            
            # Emit queue depth metrics
            queue_depths = await self.outbox_service.get_queue_depths()
            if vendor_id in queue_depths:
                await metrics_collector.emit_outbox_queue_depth(
                    vendor_id, 
                    queue_depths[vendor_id]
                )
            
            return processed_count
            
        except Exception as e:
            logger.error(f"Error processing messages for vendor {vendor_id}: {e}")
            await metrics_collector.emit_message_failure(vendor_id, "processing_error")
            raise

    async def process_single_message(self, message: OutboxMessage) -> bool:
        """Process a single outbox message"""
        try:
            # Publish to Kafka
            await self.kafka_producer.publish_device_data(
                device_reading=message.event_payload,
                key=message.kafka_key
            )
            
            # Mark as completed
            success = await self.outbox_service.mark_message_completed(message)
            
            if success:
                await metrics_collector.emit_message_success(message.vendor_id)
                self.processing_stats["messages_succeeded"] += 1
                
                logger.debug(f"✅ Message {message.id} published successfully", extra={
                    "vendor_id": message.vendor_id,
                    "correlation_id": message.correlation_id,
                    "device_id": message.device_id
                })
                
                return True
            else:
                logger.error(f"Failed to mark message {message.id} as completed")
                return False
                
        except Exception as e:
            # Handle failure with retry logic
            logger.error(f"Failed to process message {message.id}: {e}", extra={
                "vendor_id": message.vendor_id,
                "correlation_id": message.correlation_id,
                "retry_count": message.retry_count
            })
            
            # Calculate exponential backoff delay
            should_retry = await self.should_retry_message(message, str(e))
            
            success = await self.outbox_service.handle_message_failure(
                message, 
                str(e), 
                should_retry
            )
            
            if not success:
                logger.error(f"Failed to handle message failure for {message.id}")
            
            await metrics_collector.emit_message_failure(
                message.vendor_id, 
                "kafka_publish_error"
            )
            
            self.processing_stats["messages_failed"] += 1
            return False

    async def should_retry_message(self, message: OutboxMessage, error: str) -> bool:
        """Determine if a message should be retried based on error type and retry count"""
        # Don't retry if max retries exceeded
        if not message.can_retry():
            return False
        
        # Don't retry certain types of errors (validation, auth, etc.)
        non_retryable_errors = [
            "validation",
            "authentication", 
            "authorization",
            "malformed",
            "invalid"
        ]
        
        error_lower = error.lower()
        for non_retryable in non_retryable_errors:
            if non_retryable in error_lower:
                logger.warning(f"Non-retryable error for message {message.id}: {error}")
                return False
        
        # Calculate exponential backoff delay
        delay_seconds = self.retry_backoff_base * (2 ** message.retry_count)
        max_delay = 3600  # 1 hour max
        delay_seconds = min(delay_seconds, max_delay)
        
        # Check if enough time has passed since last attempt
        if message.processed_at:
            time_since_last_attempt = datetime.utcnow() - message.processed_at
            if time_since_last_attempt.total_seconds() < delay_seconds:
                logger.debug(f"Message {message.id} not ready for retry yet")
                return False
        
        return True

    async def update_vendor_health(self, vendor_id: str, is_healthy: bool, error: Optional[str]):
        """Update health status for a vendor"""
        if vendor_id not in self.health_status:
            self.health_status[vendor_id] = {
                "is_healthy": True,
                "last_successful_processing": datetime.utcnow(),
                "consecutive_failures": 0,
                "last_error": None
            }
        
        vendor_health = self.health_status[vendor_id]
        
        if is_healthy:
            vendor_health["is_healthy"] = True
            vendor_health["last_successful_processing"] = datetime.utcnow()
            vendor_health["consecutive_failures"] = 0
            vendor_health["last_error"] = None
        else:
            vendor_health["is_healthy"] = False
            vendor_health["consecutive_failures"] += 1
            vendor_health["last_error"] = error
        
        # Emit health metrics
        await metrics_collector.emit_publisher_health(vendor_id, is_healthy)

    async def stop(self):
        """Stop the publishing service gracefully"""
        logger.info("🛑 Stopping outbox publisher service...")
        self.is_running = False

    def get_health_status(self) -> Dict[str, Any]:
        """Get comprehensive health status of the publisher service"""
        return {
            "is_running": self.is_running,
            "vendor_health": self.health_status,
            "processing_stats": self.processing_stats,
            "configuration": {
                "batch_size": self.batch_size,
                "poll_interval": self.poll_interval,
                "max_concurrent_vendors": self.max_concurrent_vendors,
                "retry_backoff_base": self.retry_backoff_base
            }
        }

    def get_processing_stats(self) -> Dict[str, Any]:
        """Get processing statistics"""
        return self.processing_stats.copy()


# Global publisher instance
outbox_publisher = OutboxPublisher()
