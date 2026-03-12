"""
Outbox Manager for Global Outbox Service

Handles transactional storage and management of events in the outbox.
Provides core business logic for event persistence and retrieval.
"""

import asyncio
import json
import logging
import uuid
from datetime import datetime
from typing import Optional, Dict, Any, List

from app.core.database import db_manager
from app.core.config import settings
from app.services.medical_circuit_breaker import medical_circuit_breaker, MedicalPriority

logger = logging.getLogger(__name__)

class OutboxManager:
    """
    Core outbox management service
    
    Handles:
    - Transactional event storage
    - Idempotency management
    - Event retrieval and status updates
    - Dead letter queue management
    """
    
    def __init__(self):
        self.service_name = "OutboxManager"
        logger.debug(f"Initialized {self.__class__.__name__}")

    def _map_priority_to_medical(self, numeric_priority: int) -> MedicalPriority:
        """Map numeric priority to medical priority"""
        priority_mapping = {
            3: MedicalPriority.EMERGENCY,  # Critical priority
            2: MedicalPriority.CRITICAL,   # High priority
            1: MedicalPriority.NORMAL,     # Normal priority
            0: MedicalPriority.LOW         # Low priority
        }
        return priority_mapping.get(numeric_priority, MedicalPriority.NORMAL)
    
    async def store_event(
        self,
        idempotency_key: str,
        origin_service: str,
        kafka_topic: str,
        kafka_key: Optional[str] = None,
        event_payload: bytes = None,
        event_type: Optional[str] = None,
        correlation_id: Optional[str] = None,
        causation_id: Optional[str] = None,
        subject: Optional[str] = None,
        priority: int = 1,
        metadata: Optional[Dict[str, Any]] = None,
        scheduled_at: Optional[datetime] = None
    ) -> Optional[str]:
        """
        Store event in outbox with transactional guarantees
        
        Args:
            idempotency_key: Unique key for idempotency
            origin_service: Service originating the event
            kafka_topic: Target Kafka topic
            kafka_key: Optional Kafka message key
            event_payload: Event payload as bytes
            event_type: Optional event type
            correlation_id: Optional correlation ID
            causation_id: Optional causation ID
            subject: Optional event subject
            priority: Event priority (0-3)
            metadata: Optional metadata dictionary
            scheduled_at: Optional scheduled delivery time
            
        Returns:
            Outbox record ID if successful, None otherwise
        """
        try:
            # Medical Circuit Breaker Check
            if settings.MEDICAL_CIRCUIT_BREAKER_ENABLED:
                event_data = {
                    "event_type": event_type or "",
                    "origin_service": origin_service,
                    "metadata": metadata or {},
                    "priority": priority
                }

                should_process = await medical_circuit_breaker.should_process_event(event_data)
                if not should_process:
                    logger.warning(f"🚫 Event dropped by medical circuit breaker: {event_type}")
                    return None

            # Generate unique ID for the outbox record
            outbox_id = str(uuid.uuid4())

            # Convert metadata to JSON if present
            metadata_json = None
            if metadata:
                metadata_json = json.dumps(metadata)

            # Determine status based on scheduling
            status = "scheduled" if scheduled_at else "pending"
            
            # Insert into outbox table with conflict resolution
            query = """
                INSERT INTO global_event_outbox (
                    id, origin_service, idempotency_key, kafka_topic, kafka_key,
                    event_payload, event_type, correlation_id, causation_id, subject,
                    priority, status, scheduled_at, metadata
                ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
                ON CONFLICT (origin_service, idempotency_key) 
                DO UPDATE SET 
                    kafka_topic = EXCLUDED.kafka_topic,
                    kafka_key = EXCLUDED.kafka_key,
                    event_payload = EXCLUDED.event_payload,
                    event_type = EXCLUDED.event_type,
                    correlation_id = EXCLUDED.correlation_id,
                    causation_id = EXCLUDED.causation_id,
                    subject = EXCLUDED.subject,
                    priority = EXCLUDED.priority,
                    scheduled_at = EXCLUDED.scheduled_at,
                    metadata = EXCLUDED.metadata
                RETURNING id
            """
            
            async with db_manager.get_transaction() as conn:
                result = await conn.fetchval(
                    query,
                    outbox_id,
                    origin_service,
                    idempotency_key,
                    kafka_topic,
                    kafka_key,
                    event_payload,
                    event_type,
                    correlation_id,
                    causation_id,
                    subject,
                    priority,
                    status,
                    scheduled_at,
                    metadata_json
                )
                
                logger.info(f"✅ Event stored in outbox: {result}")
                return str(result) if result else None
                
        except Exception as e:
            logger.error(f"❌ Failed to store event in outbox: {e}", exc_info=True)
            return None
    
    async def get_pending_events(self, limit: int = 100, service_filter: Optional[str] = None) -> List[Dict[str, Any]]:
        """
        Get pending events for processing
        
        Uses SELECT FOR UPDATE SKIP LOCKED for concurrent processing
        """
        try:
            base_query = """
                SELECT id, origin_service, kafka_topic, kafka_key, event_payload,
                       event_type, correlation_id, causation_id, subject,
                       priority, retry_count, metadata, created_at
                FROM global_event_outbox 
                WHERE status = 'pending'
            """
            
            if service_filter:
                base_query += " AND origin_service = $2"
                query = base_query + " ORDER BY priority DESC, created_at ASC LIMIT $1 FOR UPDATE SKIP LOCKED"
                params = [limit, service_filter]
            else:
                query = base_query + " ORDER BY priority DESC, created_at ASC LIMIT $1 FOR UPDATE SKIP LOCKED"
                params = [limit]
            
            async with db_manager.get_transaction() as conn:
                rows = await conn.fetch(query, *params)
                
                # Mark events as processing
                if rows:
                    event_ids = [row['id'] for row in rows]
                    await conn.execute(
                        "UPDATE global_event_outbox SET status = 'processing' WHERE id = ANY($1)",
                        event_ids
                    )
                
                return [dict(row) for row in rows]
                
        except Exception as e:
            logger.error(f"❌ Failed to get pending events: {e}")
            return []
    
    async def mark_event_published(self, event_id: str) -> bool:
        """Mark an event as successfully published"""
        try:
            query = """
                UPDATE global_event_outbox 
                SET status = 'published', processed_at = NOW()
                WHERE id = $1
            """
            
            async with db_manager.get_connection() as conn:
                result = await conn.execute(query, event_id)
                return result == "UPDATE 1"
                
        except Exception as e:
            logger.error(f"❌ Failed to mark event as published: {e}")
            return False
    
    async def mark_event_failed(self, event_id: str, error_message: str) -> bool:
        """Mark an event as failed and increment retry count"""
        try:
            query = """
                UPDATE global_event_outbox 
                SET status = 'failed', 
                    retry_count = retry_count + 1,
                    last_error = $2
                WHERE id = $1
                RETURNING retry_count
            """
            
            async with db_manager.get_connection() as conn:
                retry_count = await conn.fetchval(query, event_id, error_message)
                
                # Move to dead letter queue if max retries exceeded
                if retry_count and retry_count >= settings.MAX_RETRY_ATTEMPTS:
                    await self._move_to_dead_letter_queue(event_id)
                
                return True
                
        except Exception as e:
            logger.error(f"❌ Failed to mark event as failed: {e}")
            return False
    
    async def _move_to_dead_letter_queue(self, event_id: str) -> bool:
        """Move an event to the dead letter queue"""
        try:
            # Get event details
            event_query = """
                SELECT * FROM global_event_outbox WHERE id = $1
            """
            
            # Insert into dead letter queue
            dlq_query = """
                INSERT INTO global_dead_letter_queue (
                    original_outbox_id, origin_service, event_type, event_payload,
                    kafka_topic, kafka_key, correlation_id, causation_id, subject,
                    final_error, retry_count, original_created_at, metadata
                )
                SELECT id, origin_service, event_type, event_payload,
                       kafka_topic, kafka_key, correlation_id, causation_id, subject,
                       last_error, retry_count, created_at, metadata
                FROM global_event_outbox WHERE id = $1
            """
            
            # Delete from outbox
            delete_query = "DELETE FROM global_event_outbox WHERE id = $1"
            
            async with db_manager.get_transaction() as conn:
                # Move to DLQ
                await conn.execute(dlq_query, event_id)
                # Remove from outbox
                await conn.execute(delete_query, event_id)
                
                logger.warning(f"⚠️  Event moved to dead letter queue: {event_id}")
                return True
                
        except Exception as e:
            logger.error(f"❌ Failed to move event to DLQ: {e}")
            return False
    
    async def get_events_by_correlation(self, correlation_id: str, limit: int = 50) -> List[Dict[str, Any]]:
        """Get events by correlation ID for debugging"""
        try:
            query = """
                SELECT id, origin_service, event_type, kafka_topic, kafka_key,
                       status, correlation_id, causation_id, subject, retry_count,
                       last_error, created_at, processed_at
                FROM global_event_outbox 
                WHERE correlation_id = $1
                ORDER BY created_at DESC
                LIMIT $2
            """
            
            return await db_manager.fetch_all(query, correlation_id, limit)
            
        except Exception as e:
            logger.error(f"❌ Failed to get events by correlation: {e}")
            return []
    
    async def health_check(self) -> bool:
        """Check if the outbox manager is healthy"""
        try:
            # Simple health check - verify database connectivity
            return db_manager.is_connected and db_manager.is_healthy
        except Exception as e:
            logger.error(f"❌ Outbox manager health check failed: {e}")
            return False
