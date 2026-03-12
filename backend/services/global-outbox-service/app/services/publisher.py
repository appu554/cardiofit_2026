"""
Background Publisher for Global Outbox Service

Polls the outbox for pending events and publishes them to Kafka.
Implements retry logic, error handling, and dead letter queue processing.
"""

import asyncio
import logging
import time
from typing import List, Dict, Any, Optional
from concurrent.futures import ThreadPoolExecutor

from confluent_kafka import Producer
from confluent_kafka.error import KafkaException

from app.core.config import settings
from app.core.database import db_manager
from app.services.outbox_manager import OutboxManager

logger = logging.getLogger(__name__)

class BackgroundPublisher:
    """
    Background publisher service
    
    Continuously polls the outbox for pending events and publishes them to Kafka.
    Handles retries, error recovery, and dead letter queue processing.
    """
    
    def __init__(self):
        self.outbox_manager = OutboxManager()
        self.kafka_producer = None
        self.is_running = False
        self.executor = ThreadPoolExecutor(max_workers=settings.PUBLISHER_MAX_WORKERS)
        
    async def start(self):
        """Start the background publisher"""
        if self.is_running:
            logger.warning("Background publisher is already running")
            return
        
        logger.info("🚀 Starting background publisher...")
        
        try:
            # Initialize Kafka producer
            await self._initialize_kafka_producer()
            
            self.is_running = True
            
            # Start publisher loop
            await self._publisher_loop()
            
        except Exception as e:
            logger.error(f"❌ Background publisher failed to start: {e}")
            self.is_running = False
            raise
    
    async def stop(self):
        """Stop the background publisher"""
        logger.info("🛑 Stopping background publisher...")
        
        self.is_running = False
        
        if self.kafka_producer:
            # Flush any remaining messages
            self.kafka_producer.flush(timeout=10)
            self.kafka_producer = None
        
        # Shutdown executor
        self.executor.shutdown(wait=True)
        
        logger.info("✅ Background publisher stopped")
    
    async def _initialize_kafka_producer(self):
        """Initialize Kafka producer with configuration"""
        try:
            kafka_config = settings.get_kafka_config()
            
            # Add producer-specific configuration
            kafka_config.update({
                'enable.idempotence': True,
                'max.in.flight.requests.per.connection': 5,
                'compression.type': 'snappy',
                'batch.size': 16384,
                'linger.ms': 10,
            })
            
            self.kafka_producer = Producer(kafka_config)
            
            logger.info("✅ Kafka producer initialized successfully")
            logger.info(f"   Bootstrap servers: {settings.KAFKA_BOOTSTRAP_SERVERS}")
            
        except Exception as e:
            logger.error(f"❌ Failed to initialize Kafka producer: {e}")
            raise
    
    async def _publisher_loop(self):
        """Main publisher loop"""
        logger.info(f"📡 Publisher loop started (poll interval: {settings.PUBLISHER_POLL_INTERVAL}s)")
        
        consecutive_errors = 0
        max_consecutive_errors = 5
        
        while self.is_running:
            try:
                # Check database connectivity
                if not db_manager.is_connected:
                    logger.warning("⚠️  Database not connected, waiting...")
                    await asyncio.sleep(settings.PUBLISHER_POLL_INTERVAL * 2)
                    continue
                
                # Process pending events
                processed_count = await self._process_pending_events()
                
                if processed_count > 0:
                    logger.info(f"📤 Processed {processed_count} events")
                    consecutive_errors = 0  # Reset error counter on success
                
                # Process scheduled events
                scheduled_count = await self._process_scheduled_events()
                
                if scheduled_count > 0:
                    logger.info(f"⏰ Processed {scheduled_count} scheduled events")
                
                # Sleep before next poll
                await asyncio.sleep(settings.PUBLISHER_POLL_INTERVAL)
                
            except Exception as e:
                consecutive_errors += 1
                logger.error(f"❌ Publisher loop error ({consecutive_errors}/{max_consecutive_errors}): {e}")
                
                if consecutive_errors >= max_consecutive_errors:
                    logger.error("❌ Too many consecutive errors, stopping publisher")
                    break
                
                # Exponential backoff on errors
                error_delay = min(settings.PUBLISHER_POLL_INTERVAL * (2 ** consecutive_errors), 60)
                await asyncio.sleep(error_delay)
        
        logger.info("🛑 Publisher loop stopped")
    
    async def _process_pending_events(self) -> int:
        """Process pending events from the outbox"""
        try:
            # Get pending events
            events = await self.outbox_manager.get_pending_events(
                limit=settings.PUBLISHER_BATCH_SIZE
            )
            
            if not events:
                return 0
            
            # Process events in parallel
            tasks = []
            for event in events:
                task = asyncio.create_task(self._publish_event(event))
                tasks.append(task)
            
            # Wait for all tasks to complete
            results = await asyncio.gather(*tasks, return_exceptions=True)
            
            # Count successful publications
            success_count = sum(1 for result in results if result is True)
            
            return success_count
            
        except Exception as e:
            logger.error(f"❌ Failed to process pending events: {e}")
            return 0
    
    async def _process_scheduled_events(self) -> int:
        """Process scheduled events that are ready for delivery"""
        try:
            # Get scheduled events that are ready
            query = """
                SELECT id, origin_service, kafka_topic, kafka_key, event_payload,
                       event_type, correlation_id, causation_id, subject,
                       priority, retry_count, metadata, created_at
                FROM global_event_outbox 
                WHERE status = 'scheduled' AND scheduled_at <= NOW()
                ORDER BY scheduled_at ASC
                LIMIT $1
                FOR UPDATE SKIP LOCKED
            """
            
            async with db_manager.get_transaction() as conn:
                rows = await conn.fetch(query, settings.PUBLISHER_BATCH_SIZE)
                
                if not rows:
                    return 0
                
                # Mark as processing
                event_ids = [row['id'] for row in rows]
                await conn.execute(
                    "UPDATE global_event_outbox SET status = 'processing' WHERE id = ANY($1)",
                    event_ids
                )
                
                events = [dict(row) for row in rows]
            
            # Process scheduled events
            tasks = []
            for event in events:
                task = asyncio.create_task(self._publish_event(event))
                tasks.append(task)
            
            results = await asyncio.gather(*tasks, return_exceptions=True)
            success_count = sum(1 for result in results if result is True)
            
            return success_count
            
        except Exception as e:
            logger.error(f"❌ Failed to process scheduled events: {e}")
            return 0
    
    async def _publish_event(self, event: Dict[str, Any]) -> bool:
        """Publish a single event to Kafka"""
        event_id = event['id']
        
        try:
            # Prepare Kafka message
            kafka_message = {
                'topic': event['kafka_topic'],
                'value': event['event_payload'],
                'key': event.get('kafka_key'),
                'headers': {
                    'origin_service': event['origin_service'],
                    'event_type': event.get('event_type', ''),
                    'correlation_id': event.get('correlation_id', ''),
                    'causation_id': event.get('causation_id', ''),
                    'subject': event.get('subject', ''),
                    'outbox_id': event_id
                }
            }
            
            # Publish to Kafka (async)
            loop = asyncio.get_event_loop()
            success = await loop.run_in_executor(
                self.executor, 
                self._kafka_publish_sync, 
                kafka_message
            )
            
            if success:
                # Mark as published
                await self.outbox_manager.mark_event_published(event_id)
                logger.debug(f"✅ Event published: {event_id}")
                return True
            else:
                # Mark as failed
                await self.outbox_manager.mark_event_failed(
                    event_id, 
                    "Failed to publish to Kafka"
                )
                return False
                
        except Exception as e:
            logger.error(f"❌ Failed to publish event {event_id}: {e}")
            await self.outbox_manager.mark_event_failed(event_id, str(e))
            return False
    
    def _kafka_publish_sync(self, message: Dict[str, Any]) -> bool:
        """Synchronous Kafka publish (runs in thread pool)"""
        try:
            # Convert headers to bytes
            headers = {}
            for key, value in message.get('headers', {}).items():
                if isinstance(value, str):
                    headers[key] = value.encode('utf-8')
                else:
                    headers[key] = str(value).encode('utf-8')
            
            # Publish message
            self.kafka_producer.produce(
                topic=message['topic'],
                value=message['value'],
                key=message.get('key'),
                headers=headers,
                callback=self._delivery_callback
            )
            
            # Poll for delivery reports
            self.kafka_producer.poll(timeout=1.0)
            
            return True
            
        except KafkaException as e:
            logger.error(f"❌ Kafka publish error: {e}")
            return False
        except Exception as e:
            logger.error(f"❌ Unexpected publish error: {e}")
            return False
    
    def _delivery_callback(self, err, msg):
        """Kafka delivery callback"""
        if err:
            logger.error(f"❌ Message delivery failed: {err}")
        else:
            logger.debug(f"✅ Message delivered to {msg.topic()} [{msg.partition()}] @ {msg.offset()}")

# Global publisher instance
background_publisher = BackgroundPublisher()

async def start_background_publisher():
    """Start the background publisher service"""
    await background_publisher.start()

async def stop_background_publisher():
    """Stop the background publisher service"""
    await background_publisher.stop()
