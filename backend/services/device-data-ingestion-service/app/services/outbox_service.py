"""
Vendor-Aware Transactional Outbox Service for Device Data Ingestion

CRITICAL FEATURES:
- Per-vendor table isolation for true fault tolerance
- SELECT FOR UPDATE SKIP LOCKED for race condition prevention
- Cloud-native metrics emission (no database metrics writes)
"""
import json
import logging
import uuid
from datetime import datetime, timedelta
from typing import Dict, Any, Optional, List

from sqlalchemy import text
from sqlalchemy.exc import SQLAlchemyError

from app.db.database import get_async_session
from app.db.models import (
    OutboxMessage, 
    DeadLetterMessage, 
    VendorOutboxRegistry,
    get_vendor_config,
    is_supported_vendor
)

logger = logging.getLogger(__name__)


class VendorAwareOutboxService:
    """
    Vendor-Aware Transactional Outbox Service for Device Data Ingestion
    
    CRITICAL FEATURES:
    - Per-vendor table isolation for true fault tolerance
    - SELECT FOR UPDATE SKIP LOCKED for race condition prevention
    - Cloud-native metrics emission (no database metrics writes)
    """

    def __init__(self):
        self.vendor_registry = {}
        self._registry_loaded = False

    async def _load_vendor_registry(self):
        """Load vendor configurations from registry table"""
        if self._registry_loaded:
            return
            
        try:
            async with get_async_session() as session:
                result = await session.execute(
                    text("SELECT * FROM vendor_outbox_registry WHERE is_active = true")
                )
                vendors = result.fetchall()
                
                for vendor in vendors:
                    self.vendor_registry[vendor.vendor_id] = {
                        'outbox_table': vendor.outbox_table_name,
                        'dead_letter_table': vendor.dead_letter_table_name,
                        'max_retries': vendor.max_retries,
                        'kafka_topic': vendor.kafka_topic,
                        'retry_backoff_seconds': vendor.retry_backoff_seconds
                    }
                
                self._registry_loaded = True
                logger.info(f"Loaded {len(self.vendor_registry)} vendor configurations")
                
        except Exception as e:
            logger.error(f"Failed to load vendor registry: {e}")
            # Fallback to hardcoded configuration
            self.vendor_registry = {
                'fitbit': {
                    'outbox_table': 'fitbit_outbox',
                    'dead_letter_table': 'fitbit_dead_letter',
                    'max_retries': 3,
                    'kafka_topic': 'raw-device-data.v1',
                    'retry_backoff_seconds': 60
                },
                'garmin': {
                    'outbox_table': 'garmin_outbox',
                    'dead_letter_table': 'garmin_dead_letter',
                    'max_retries': 3,
                    'kafka_topic': 'raw-device-data.v1',
                    'retry_backoff_seconds': 60
                },
                'apple_health': {
                    'outbox_table': 'apple_health_outbox',
                    'dead_letter_table': 'apple_health_dead_letter',
                    'max_retries': 3,
                    'kafka_topic': 'raw-device-data.v1',
                    'retry_backoff_seconds': 60
                }
            }
            self._registry_loaded = True

    async def store_device_data_transactionally(
        self,
        device_data: Dict[str, Any],
        vendor_id: str,
        correlation_id: Optional[str] = None,
        trace_id: Optional[str] = None
    ) -> str:
        """
        Store device data in vendor-specific outbox table within a database transaction

        Args:
            device_data: Device reading data
            vendor_id: Vendor identifier (e.g., 'fitbit', 'garmin')
            correlation_id: Request correlation ID
            trace_id: Distributed tracing ID

        Returns:
            str: Outbox record ID
        """
        await self._load_vendor_registry()
        
        if vendor_id not in self.vendor_registry:
            raise ValueError(f"Unknown vendor: {vendor_id}")

        vendor_config = self.vendor_registry[vendor_id]
        outbox_table = vendor_config['outbox_table']

        record_id = str(uuid.uuid4())

        try:
            async with get_async_session() as session:
                async with session.begin():  # Explicit transaction
                    await session.execute(
                        text(f"""
                            INSERT INTO {outbox_table}
                            (id, device_id, event_payload, kafka_key, correlation_id, trace_id, kafka_topic)
                            VALUES (:id, :device_id, :payload, :kafka_key, :correlation_id, :trace_id, :kafka_topic)
                        """),
                        {
                            'id': record_id,
                            'device_id': device_data.get('device_id'),
                            'payload': json.dumps(device_data),
                            'kafka_key': device_data.get('device_id'),
                            'correlation_id': correlation_id,
                            'trace_id': trace_id,
                            'kafka_topic': vendor_config['kafka_topic']
                        }
                    )

            logger.info(f"Stored device data in {outbox_table}", extra={
                'vendor_id': vendor_id,
                'record_id': record_id,
                'correlation_id': correlation_id
            })

            return record_id

        except SQLAlchemyError as e:
            logger.error(f"Failed to store device data in outbox: {e}", extra={
                'vendor_id': vendor_id,
                'correlation_id': correlation_id
            })
            raise

    async def get_pending_messages_with_lock(
        self,
        vendor_id: str,
        batch_size: int = 100
    ) -> List[OutboxMessage]:
        """
        CRITICAL: Retrieve pending messages with PostgreSQL row-level locking
        Uses SELECT FOR UPDATE SKIP LOCKED to prevent race conditions between
        multiple publisher instances
        
        This is the industry-standard pattern for concurrent queue polling
        """
        await self._load_vendor_registry()
        
        if vendor_id not in self.vendor_registry:
            return []

        vendor_config = self.vendor_registry[vendor_id]
        outbox_table = vendor_config['outbox_table']

        try:
            async with get_async_session() as session:
                async with session.begin():
                    # CRITICAL: SELECT FOR UPDATE SKIP LOCKED prevents race conditions
                    result = await session.execute(
                        text(f"""
                            SELECT id, device_id, event_payload, kafka_key, kafka_topic,
                                   correlation_id, trace_id, retry_count, created_at
                            FROM {outbox_table}
                            WHERE status = 'pending'
                            ORDER BY created_at ASC
                            LIMIT :batch_size
                            FOR UPDATE SKIP LOCKED
                        """),
                        {'batch_size': batch_size}
                    )

                    messages = []
                    rows = result.fetchall()
                    
                    if rows:
                        # Mark messages as processing to prevent re-selection
                        message_ids = [str(row.id) for row in rows]
                        await session.execute(
                            text(f"""
                                UPDATE {outbox_table}
                                SET status = 'processing', processed_at = NOW()
                                WHERE id = ANY(:message_ids)
                            """),
                            {'message_ids': message_ids}
                        )

                        # Convert rows to OutboxMessage objects
                        for row in rows:
                            message = OutboxMessage.from_db_row(
                                row, 
                                vendor_id=vendor_id, 
                                outbox_table=outbox_table
                            )
                            messages.append(message)

                    return messages

        except SQLAlchemyError as e:
            logger.error(f"Failed to get pending messages for {vendor_id}: {e}")
            return []

    async def mark_message_completed(self, message: OutboxMessage) -> bool:
        """Mark message as successfully processed"""
        try:
            async with get_async_session() as session:
                result = await session.execute(
                    text(f"""
                        UPDATE {message.outbox_table}
                        SET status = 'completed', processed_at = NOW()
                        WHERE id = :message_id
                    """),
                    {'message_id': message.id}
                )
                await session.commit()
                
                return result.rowcount > 0

        except SQLAlchemyError as e:
            logger.error(f"Failed to mark message as completed: {e}")
            return False

    async def handle_message_failure(
        self,
        message: OutboxMessage,
        error: str,
        should_retry: bool = True
    ) -> bool:
        """Handle message processing failure with retry logic"""
        try:
            message.increment_retry()
            message.last_error = error

            if should_retry and message.can_retry():
                # Update for retry
                async with get_async_session() as session:
                    await session.execute(
                        text(f"""
                            UPDATE {message.outbox_table}
                            SET status = 'pending', retry_count = :retry_count, 
                                last_error = :error, processed_at = NULL
                            WHERE id = :message_id
                        """),
                        {
                            'message_id': message.id,
                            'retry_count': message.retry_count,
                            'error': error
                        }
                    )
                    await session.commit()
                
                logger.warning(f"Message {message.id} scheduled for retry {message.retry_count}/{message.max_retries}")
                return True
            else:
                # Move to dead letter
                return await self.move_to_dead_letter(message, error)

        except SQLAlchemyError as e:
            logger.error(f"Failed to handle message failure: {e}")
            return False

    async def move_to_dead_letter(self, message: OutboxMessage, final_error: str) -> bool:
        """Move failed message to dead letter table"""
        await self._load_vendor_registry()
        
        vendor_config = self.vendor_registry.get(message.vendor_id)
        if not vendor_config:
            logger.error(f"Unknown vendor for dead letter: {message.vendor_id}")
            return False

        dead_letter_table = vendor_config['dead_letter_table']
        
        try:
            dead_letter_msg = DeadLetterMessage.from_outbox_message(message, final_error)
            
            async with get_async_session() as session:
                async with session.begin():
                    # Insert into dead letter table
                    await session.execute(
                        text(f"""
                            INSERT INTO {dead_letter_table}
                            (id, device_id, event_type, event_payload, kafka_topic, kafka_key,
                             original_created_at, final_error, retry_count, correlation_id, trace_id)
                            VALUES (:id, :device_id, :event_type, :payload, :kafka_topic, :kafka_key,
                                    :original_created_at, :final_error, :retry_count, :correlation_id, :trace_id)
                        """),
                        {
                            'id': dead_letter_msg.id,
                            'device_id': dead_letter_msg.device_id,
                            'event_type': dead_letter_msg.event_type,
                            'payload': json.dumps(dead_letter_msg.event_payload),
                            'kafka_topic': dead_letter_msg.kafka_topic,
                            'kafka_key': dead_letter_msg.kafka_key,
                            'original_created_at': dead_letter_msg.original_created_at,
                            'final_error': dead_letter_msg.final_error,
                            'retry_count': dead_letter_msg.retry_count,
                            'correlation_id': dead_letter_msg.correlation_id,
                            'trace_id': dead_letter_msg.trace_id
                        }
                    )

                    # Remove from outbox table
                    await session.execute(
                        text(f"""
                            DELETE FROM {message.outbox_table}
                            WHERE id = :message_id
                        """),
                        {'message_id': message.id}
                    )

            logger.error(f"Message {message.id} moved to dead letter table", extra={
                'vendor_id': message.vendor_id,
                'correlation_id': message.correlation_id,
                'final_error': final_error
            })
            
            return True

        except SQLAlchemyError as e:
            logger.error(f"Failed to move message to dead letter: {e}")
            return False

    async def get_queue_depths(self) -> Dict[str, int]:
        """Get queue depths for all vendors"""
        await self._load_vendor_registry()
        
        queue_depths = {}
        
        for vendor_id, config in self.vendor_registry.items():
            try:
                async with get_async_session() as session:
                    result = await session.execute(
                        text(f"""
                            SELECT COUNT(*) as pending_count
                            FROM {config['outbox_table']}
                            WHERE status = 'pending'
                        """)
                    )
                    count = result.scalar()
                    queue_depths[vendor_id] = count or 0
                    
            except SQLAlchemyError as e:
                logger.error(f"Failed to get queue depth for {vendor_id}: {e}")
                queue_depths[vendor_id] = -1  # Error indicator
        
        return queue_depths

    async def get_health_status(self) -> Dict[str, Any]:
        """Get comprehensive health status of outbox system"""
        await self._load_vendor_registry()
        
        health_status = {
            "status": "healthy",
            "vendors": {},
            "total_pending": 0,
            "total_processing": 0,
            "registry_loaded": self._registry_loaded
        }
        
        for vendor_id, config in self.vendor_registry.items():
            try:
                async with get_async_session() as session:
                    result = await session.execute(
                        text(f"""
                            SELECT 
                                COUNT(CASE WHEN status = 'pending' THEN 1 END) as pending,
                                COUNT(CASE WHEN status = 'processing' THEN 1 END) as processing,
                                COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed,
                                MIN(CASE WHEN status = 'pending' THEN created_at END) as oldest_pending
                            FROM {config['outbox_table']}
                        """)
                    )
                    stats = result.fetchone()
                    
                    vendor_health = {
                        "pending": stats.pending or 0,
                        "processing": stats.processing or 0,
                        "failed": stats.failed or 0,
                        "oldest_pending": stats.oldest_pending.isoformat() if stats.oldest_pending else None,
                        "outbox_table": config['outbox_table']
                    }
                    
                    health_status["vendors"][vendor_id] = vendor_health
                    health_status["total_pending"] += vendor_health["pending"]
                    health_status["total_processing"] += vendor_health["processing"]
                    
            except SQLAlchemyError as e:
                logger.error(f"Failed to get health status for {vendor_id}: {e}")
                health_status["vendors"][vendor_id] = {"error": str(e)}
                health_status["status"] = "degraded"
        
        return health_status

    async def get_pending_messages(self, vendor_id: str, limit: int = 50) -> List[Dict]:
        """Get pending messages for a specific vendor"""
        await self._load_vendor_registry()

        vendor_config = self.vendor_registry.get(vendor_id)
        if not vendor_config:
            logger.error(f"Vendor {vendor_id} not found in registry")
            return []

        outbox_table = vendor_config['outbox_table']

        try:
            async with get_async_session() as session:
                result = await session.execute(
                    text(f"""
                        SELECT id, device_id, event_type, event_payload, kafka_topic, kafka_key,
                               created_at, retry_count, max_retries, last_error, status,
                               correlation_id, trace_id
                        FROM {outbox_table}
                        WHERE status = 'pending'
                        ORDER BY created_at
                        LIMIT :limit
                        FOR UPDATE SKIP LOCKED
                    """),
                    {"limit": limit}
                )

                messages = []
                for row in result.fetchall():
                    message = {
                        "id": row.id,
                        "device_id": row.device_id,
                        "event_type": row.event_type,
                        "event_payload": row.event_payload,
                        "kafka_topic": row.kafka_topic,
                        "kafka_key": row.kafka_key,
                        "created_at": row.created_at,
                        "retry_count": row.retry_count,
                        "max_retries": row.max_retries,
                        "last_error": row.last_error,
                        "status": row.status,
                        "correlation_id": row.correlation_id,
                        "trace_id": row.trace_id
                    }
                    messages.append(message)

                return messages

        except Exception as e:
            logger.error(f"Failed to get pending messages for {vendor_id}: {e}")
            return []

    async def mark_message_processing(self, vendor_id: str, message_id: str) -> bool:
        """Mark a message as processing"""
        await self._load_vendor_registry()

        vendor_config = self.vendor_registry.get(vendor_id)
        if not vendor_config:
            return False

        outbox_table = vendor_config['outbox_table']

        try:
            async with get_async_session() as session:
                await session.execute(
                    text(f"""
                        UPDATE {outbox_table}
                        SET status = 'processing'
                        WHERE id = :message_id AND status = 'pending'
                    """),
                    {"message_id": message_id}
                )
                await session.commit()
                return True

        except Exception as e:
            logger.error(f"Failed to mark message {message_id} as processing: {e}")
            return False

    async def mark_message_completed(self, vendor_id: str, message_id: str) -> bool:
        """Mark a message as completed"""
        await self._load_vendor_registry()

        vendor_config = self.vendor_registry.get(vendor_id)
        if not vendor_config:
            return False

        outbox_table = vendor_config['outbox_table']

        try:
            async with get_async_session() as session:
                await session.execute(
                    text(f"""
                        UPDATE {outbox_table}
                        SET status = 'completed', processed_at = NOW()
                        WHERE id = :message_id
                    """),
                    {"message_id": message_id}
                )
                await session.commit()
                return True

        except Exception as e:
            logger.error(f"Failed to mark message {message_id} as completed: {e}")
            return False

    async def mark_message_failed(self, vendor_id: str, message_id: str, error: str) -> bool:
        """Mark a message as failed and increment retry count"""
        await self._load_vendor_registry()

        vendor_config = self.vendor_registry.get(vendor_id)
        if not vendor_config:
            return False

        outbox_table = vendor_config['outbox_table']

        try:
            async with get_async_session() as session:
                # Get current retry count
                result = await session.execute(
                    text(f"""
                        SELECT retry_count, max_retries
                        FROM {outbox_table}
                        WHERE id = :message_id
                    """),
                    {"message_id": message_id}
                )

                row = result.fetchone()
                if not row:
                    return False

                current_retry = row.retry_count or 0
                max_retries = row.max_retries or 3
                new_retry_count = current_retry + 1

                # Determine new status
                new_status = "failed" if new_retry_count >= max_retries else "pending"

                # Update message
                await session.execute(
                    text(f"""
                        UPDATE {outbox_table}
                        SET status = :status, retry_count = :retry_count, last_error = :error
                        WHERE id = :message_id
                    """),
                    {
                        "message_id": message_id,
                        "status": new_status,
                        "retry_count": new_retry_count,
                        "error": error
                    }
                )
                await session.commit()
                return True

        except Exception as e:
            logger.error(f"Failed to mark message {message_id} as failed: {e}")
            return False

    async def health_check(self) -> Dict[str, Any]:
        """Perform health check on the outbox service"""
        try:
            await self._load_vendor_registry()

            # Test database connectivity
            async with get_async_session() as session:
                await session.execute(text("SELECT 1"))

            # Get queue depths
            queue_depths = await self.get_queue_depths()

            return {
                "status": "healthy",
                "vendor_count": len(self.vendor_registry),
                "queue_depths": queue_depths,
                "registry_loaded": self._registry_loaded
            }

        except Exception as e:
            return {
                "status": "unhealthy",
                "error": str(e)
            }
