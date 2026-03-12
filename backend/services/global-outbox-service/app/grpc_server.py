"""
gRPC Server Implementation for Global Outbox Service

Implements the GlobalOutboxService gRPC interface for event publishing.
Handles PublishEvent, HealthCheck, and GetOutboxStats methods.
"""

import asyncio
import grpc
import logging
import uuid
from concurrent import futures
from typing import Optional, Dict, Any
from datetime import datetime
from google.protobuf.timestamp_pb2 import Timestamp

from app.core.config import settings
from app.core.database import db_manager

logger = logging.getLogger(__name__)

# Import will be available after proto compilation
try:
    from app.proto import outbox_pb2, outbox_pb2_grpc
except ImportError:
    logger.warning("⚠️  Protocol buffer files not found. Run 'python compile_proto.py' first.")
    outbox_pb2 = None
    outbox_pb2_grpc = None

class GlobalOutboxServicer(outbox_pb2_grpc.GlobalOutboxServiceServicer):
    """
    gRPC service implementation for Global Outbox Service
    
    Handles:
    - Event publishing with transactional guarantees
    - Health checks for service monitoring
    - Statistics for operational visibility
    """
    
    def __init__(self):
        self.service_name = settings.PROJECT_NAME
        logger.info(f"Initialized {self.__class__.__name__}")
    
    async def PublishEvent(self, request, context):
        """
        Publish an event to the outbox for guaranteed delivery
        
        This is the core method that accepts events from microservices
        and stores them transactionally in the outbox for eventual
        delivery to Kafka.
        """
        try:
            # Validate required fields
            if not request.idempotency_key:
                context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
                context.set_details("idempotency_key is required")
                return outbox_pb2.PublishEventResponse(
                    status="ERROR",
                    error_message="idempotency_key is required"
                )
            
            if not request.origin_service:
                context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
                context.set_details("origin_service is required")
                return outbox_pb2.PublishEventResponse(
                    status="ERROR",
                    error_message="origin_service is required"
                )
            
            if not request.kafka_topic:
                context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
                context.set_details("kafka_topic is required")
                return outbox_pb2.PublishEventResponse(
                    status="ERROR",
                    error_message="kafka_topic is required"
                )
            
            if not request.event_payload:
                context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
                context.set_details("event_payload is required")
                return outbox_pb2.PublishEventResponse(
                    status="ERROR",
                    error_message="event_payload is required"
                )
            
            # Check if database is available
            if not db_manager.is_connected:
                context.set_code(grpc.StatusCode.UNAVAILABLE)
                context.set_details("Database not available")
                return outbox_pb2.PublishEventResponse(
                    status="ERROR",
                    error_message="Database not available"
                )
            
            # Store event in outbox
            outbox_record_id = await self._store_event_in_outbox(request)
            
            if outbox_record_id:
                logger.info(f"✅ Event stored in outbox: {outbox_record_id}")
                
                # Determine status based on scheduling
                status = "SCHEDULED" if request.scheduled_at.seconds > 0 else "QUEUED"
                
                # Create timestamp for accepted_at
                accepted_timestamp = Timestamp()
                accepted_timestamp.FromDatetime(datetime.utcnow())

                return outbox_pb2.PublishEventResponse(
                    outbox_record_id=outbox_record_id,
                    status=status,
                    accepted_at=accepted_timestamp
                )
            else:
                context.set_code(grpc.StatusCode.INTERNAL)
                context.set_details("Failed to store event in outbox")
                return outbox_pb2.PublishEventResponse(
                    status="ERROR",
                    error_message="Failed to store event in outbox"
                )
                
        except Exception as e:
            logger.error(f"❌ Error in PublishEvent: {e}", exc_info=True)
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return outbox_pb2.PublishEventResponse(
                status="ERROR",
                error_message=str(e)
            )
    
    async def HealthCheck(self, request, context):
        """
        Health check endpoint for service monitoring
        
        Returns comprehensive health status including database
        connectivity and component status.
        """
        try:
            # Get database health
            db_health = await db_manager.health_check()
            
            # Determine overall health
            overall_healthy = (
                db_health.get("status") == "healthy" and
                db_manager.is_connected
            )
            
            status = "HEALTHY" if overall_healthy else "UNHEALTHY"
            message = "All systems operational" if overall_healthy else "Some components unhealthy"
            
            # Component details
            components = {
                "database": db_health.get("status", "unknown"),
                "service": "healthy"
            }
            
            # Create timestamp for health check
            health_timestamp = Timestamp()
            health_timestamp.FromDatetime(datetime.utcnow())

            return outbox_pb2.HealthCheckResponse(
                status=status,
                message=message,
                components=components,
                timestamp=health_timestamp
            )
            
        except Exception as e:
            logger.error(f"❌ Error in HealthCheck: {e}")
            return outbox_pb2.HealthCheckResponse(
                status="UNHEALTHY",
                message=f"Health check failed: {str(e)}"
            )
    
    async def GetOutboxStats(self, request, context):
        """
        Get outbox statistics for monitoring and debugging
        
        Returns queue depths, success rates, and other operational metrics.
        """
        try:
            if not db_manager.is_connected:
                context.set_code(grpc.StatusCode.UNAVAILABLE)
                context.set_details("Database not available")
                return outbox_pb2.OutboxStatsResponse()
            
            # Get statistics from database
            stats = await db_manager.get_outbox_stats()
            
            # Create timestamp for stats
            stats_timestamp = Timestamp()
            stats_timestamp.FromDatetime(datetime.utcnow())

            # Build response
            response = outbox_pb2.OutboxStatsResponse(
                queue_depths=stats.get("queue_depths", {}),
                total_events_processed=stats.get("total_processed_24h", 0),
                dead_letter_count=stats.get("dead_letter_count", 0),
                timestamp=stats_timestamp
            )
            
            return response
            
        except Exception as e:
            logger.error(f"❌ Error in GetOutboxStats: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return outbox_pb2.OutboxStatsResponse()
    
    async def GetEventsByCorrelation(self, request, context):
        """
        Get events by correlation ID for debugging
        
        Useful for tracing event flows across services.
        """
        try:
            if not request.correlation_id:
                context.set_code(grpc.StatusCode.INVALID_ARGUMENT)
                context.set_details("correlation_id is required")
                return outbox_pb2.GetEventsByCorrelationResponse()
            
            if not db_manager.is_connected:
                context.set_code(grpc.StatusCode.UNAVAILABLE)
                context.set_details("Database not available")
                return outbox_pb2.GetEventsByCorrelationResponse()
            
            # Query events by correlation ID
            events = await self._get_events_by_correlation(request.correlation_id, request.limit or 50)
            
            # Convert to protobuf format
            pb_events = []
            for event in events:
                pb_event = outbox_pb2.OutboxEvent(
                    id=event["id"],
                    origin_service=event["origin_service"],
                    event_type=event.get("event_type", ""),
                    kafka_topic=event["kafka_topic"],
                    kafka_key=event.get("kafka_key", ""),
                    status=event["status"],
                    correlation_id=event.get("correlation_id", ""),
                    causation_id=event.get("causation_id", ""),
                    subject=event.get("subject", ""),
                    retry_count=event.get("retry_count", 0),
                    last_error=event.get("last_error", "")
                )
                pb_events.append(pb_event)
            
            return outbox_pb2.GetEventsByCorrelationResponse(
                events=pb_events,
                total_count=len(pb_events)
            )
            
        except Exception as e:
            logger.error(f"❌ Error in GetEventsByCorrelation: {e}")
            context.set_code(grpc.StatusCode.INTERNAL)
            context.set_details(str(e))
            return outbox_pb2.GetEventsByCorrelationResponse()
    
    async def _store_event_in_outbox(self, request) -> Optional[str]:
        """
        Store event in outbox table with transactional guarantees
        
        This is the core transactional logic that ensures events
        are reliably stored for eventual delivery.
        """
        try:
            # Generate unique ID for the outbox record
            outbox_id = str(uuid.uuid4())
            
            # Convert metadata to JSON if present
            metadata_json = None
            if request.metadata:
                import json
                from google.protobuf.json_format import MessageToDict
                metadata_json = json.dumps(MessageToDict(request.metadata))
            
            # Determine status based on scheduling
            status = "scheduled" if request.scheduled_at.seconds > 0 else "pending"
            
            # Convert scheduled timestamp
            scheduled_at = None
            if request.scheduled_at.seconds > 0:
                scheduled_at = datetime.fromtimestamp(request.scheduled_at.seconds)
            
            # Insert into outbox table
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
                    request.origin_service,
                    request.idempotency_key,
                    request.kafka_topic,
                    request.kafka_key or None,
                    request.event_payload,
                    request.event_type or None,
                    request.correlation_id or None,
                    request.causation_id or None,
                    request.subject or None,
                    request.priority or 1,
                    status,
                    scheduled_at,
                    metadata_json
                )
                
                return str(result) if result else None
                
        except Exception as e:
            logger.error(f"❌ Failed to store event in outbox: {e}")
            return None
    
    async def _get_events_by_correlation(self, correlation_id: str, limit: int = 50) -> list:
        """Get events by correlation ID"""
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

async def serve_grpc():
    """
    Start the gRPC server
    
    Configures and starts the gRPC server with the GlobalOutboxServicer.
    """
    if not outbox_pb2 or not outbox_pb2_grpc:
        raise RuntimeError("Protocol buffer files not available. Run 'python compile_proto.py' first.")
    
    # Create gRPC server
    server = grpc.aio.server(
        futures.ThreadPoolExecutor(max_workers=settings.PUBLISHER_MAX_WORKERS)
    )
    
    # Add servicer
    outbox_pb2_grpc.add_GlobalOutboxServiceServicer_to_server(
        GlobalOutboxServicer(), server
    )
    
    # Configure server address
    listen_addr = f'[::]:{settings.GRPC_PORT}'
    server.add_insecure_port(listen_addr)
    
    logger.info(f"🚀 Starting gRPC server on {listen_addr}")
    
    # Start server
    await server.start()
    
    try:
        await server.wait_for_termination()
    except KeyboardInterrupt:
        logger.info("gRPC server interrupted")
    finally:
        await server.stop(grace=5.0)
        logger.info("gRPC server stopped")

if __name__ == "__main__":
    # For testing the gRPC server standalone
    asyncio.run(serve_grpc())
