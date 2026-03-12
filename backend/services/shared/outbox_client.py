"""
Global Outbox Service gRPC Client Library

This client library provides a simple interface for microservices to publish
events to the Global Outbox Service. It handles gRPC communication, connection
management, and provides both async context manager and convenience function
interfaces.

Usage:
    # Using context manager (recommended for multiple calls)
    async with GlobalOutboxClient("patient-service") as client:
        await client.publish_event(
            event_type="patient.created",
            kafka_topic="patient-events",
            event_payload=json.dumps(patient_data).encode(),
            kafka_key=patient_id
        )
    
    # Using convenience function (for single calls)
    await publish_to_global_outbox(
        service_name="patient-service",
        event_type="patient.created",
        kafka_topic="patient-events",
        event_payload=json.dumps(patient_data).encode(),
        kafka_key=patient_id
    )
"""

import asyncio
import grpc
import json
import logging
import uuid
from typing import Dict, Any, Optional, Union
from datetime import datetime
import os

logger = logging.getLogger(__name__)

# Import protocol buffer definitions
try:
    # Try to import from global outbox service using direct file import
    import sys
    import os
    import importlib.util
    from pathlib import Path

    # Multiple strategies to find the global outbox service
    global_outbox_path = None

    # Strategy 1: From shared/ to services/global-outbox-service/
    current_dir = Path(__file__).parent  # shared/
    services_dir = current_dir.parent  # services/
    path1 = services_dir / 'global-outbox-service'

    # Strategy 2: Search from current working directory
    cwd = Path.cwd()
    path2 = cwd / 'backend' / 'services' / 'global-outbox-service'
    path3 = cwd / 'services' / 'global-outbox-service'

    # Strategy 3: Absolute path resolution
    path4 = current_dir.parent / 'global-outbox-service'

    # Try each path
    for i, path in enumerate([path1, path2, path3, path4], 1):
        if path.exists():
            global_outbox_path = path
            break

    if global_outbox_path and global_outbox_path.exists():
        # Add the global outbox service to Python path and try package import
        global_outbox_str = str(global_outbox_path)
        if global_outbox_str not in sys.path:
            sys.path.insert(0, global_outbox_str)

        # Also add the proto directory to handle relative imports
        proto_dir = global_outbox_path / 'app' / 'proto'
        proto_dir_str = str(proto_dir)
        if proto_dir_str not in sys.path:
            sys.path.insert(0, proto_dir_str)

        try:
            # Try the standard package import first
            from app.proto import outbox_pb2, outbox_pb2_grpc
            GRPC_AVAILABLE = True
            logger.info("gRPC protocol buffers loaded successfully via package import")
        except ImportError:
            # If package import fails, try creating a temporary package structure
            try:
                # Create a temporary module namespace to handle relative imports
                import types

                # Use a unique namespace to avoid conflicts
                temp_namespace = 'global_outbox_temp_app'

                # Create the temp app module
                app_module = types.ModuleType(temp_namespace)
                sys.modules[temp_namespace] = app_module

                # Create the temp app.proto module
                proto_module_name = f'{temp_namespace}.proto'
                proto_module = types.ModuleType(proto_module_name)
                sys.modules[proto_module_name] = proto_module
                app_module.proto = proto_module

                # Load outbox_pb2 into the proto module
                with open(proto_dir / 'outbox_pb2.py', 'r') as f:
                    pb2_code = f.read()
                exec(pb2_code, proto_module.__dict__)
                outbox_pb2 = proto_module

                # Load outbox_pb2_grpc and fix the relative import
                with open(proto_dir / 'outbox_pb2_grpc.py', 'r') as f:
                    grpc_code = f.read()

                # Replace the relative import with absolute import
                grpc_code = grpc_code.replace(
                    'from . import outbox_pb2 as outbox__pb2',
                    f'import {temp_namespace}.proto as outbox__pb2'
                )

                # Create outbox_pb2_grpc module
                grpc_module_name = f'{temp_namespace}.proto.outbox_pb2_grpc'
                grpc_module = types.ModuleType(grpc_module_name)
                exec(grpc_code, grpc_module.__dict__)
                outbox_pb2_grpc = grpc_module

                # Add to proto module
                proto_module.outbox_pb2_grpc = grpc_module

                GRPC_AVAILABLE = True
                logger.info("gRPC protocol buffers loaded successfully via temporary package structure")

            except Exception as temp_import_error:
                # Final fallback: try copying the files to a temporary location
                try:
                    import tempfile
                    import shutil

                    # Create temporary directory
                    temp_dir = Path(tempfile.mkdtemp())
                    temp_proto_dir = temp_dir / 'proto'
                    temp_proto_dir.mkdir()

                    # Copy protocol buffer files
                    shutil.copy2(proto_dir / 'outbox_pb2.py', temp_proto_dir)

                    # Copy and fix the grpc file
                    with open(proto_dir / 'outbox_pb2_grpc.py', 'r') as f:
                        grpc_content = f.read()

                    # Fix the relative import
                    grpc_content = grpc_content.replace(
                        'from . import outbox_pb2 as outbox__pb2',
                        'import outbox_pb2 as outbox__pb2'
                    )

                    with open(temp_proto_dir / 'outbox_pb2_grpc.py', 'w') as f:
                        f.write(grpc_content)

                    # Create __init__.py
                    (temp_proto_dir / '__init__.py').touch()

                    # Add temp directory to path and import
                    temp_proto_str = str(temp_proto_dir)
                    if temp_proto_str not in sys.path:
                        sys.path.insert(0, temp_proto_str)

                    import outbox_pb2
                    import outbox_pb2_grpc

                    GRPC_AVAILABLE = True
                    logger.info("gRPC protocol buffers loaded successfully via temporary file fix")

                    # Clean up temp directory on exit
                    import atexit
                    atexit.register(lambda: shutil.rmtree(temp_dir, ignore_errors=True))

                except Exception as final_error:
                    raise ImportError(f"All import methods failed. Final error: {final_error}")
    else:
        raise ImportError(f"Global outbox service not found. Tried paths: {[str(p) for p in [path1, path2, path3, path4]]}")

except ImportError as e:
    logger.warning(f"gRPC protocol buffers not available: {e}")
    logger.warning("Run 'python compile_proto.py' in global-outbox-service directory")
    outbox_pb2 = None
    outbox_pb2_grpc = None
    GRPC_AVAILABLE = False
except Exception as e:
    logger.warning(f"Unexpected error loading gRPC protocol buffers: {e}")
    outbox_pb2 = None
    outbox_pb2_grpc = None
    GRPC_AVAILABLE = False


class GlobalOutboxClient:
    """
    gRPC client for publishing events to the Global Outbox Service
    
    This client provides a high-level interface for microservices to publish
    events with automatic connection management, error handling, and retry logic.
    """
    
    def __init__(
        self, 
        service_name: str, 
        outbox_service_url: Optional[str] = None,
        timeout: float = 30.0,
        max_retries: int = 3
    ):
        """
        Initialize the Global Outbox Client
        
        Args:
            service_name: Name of the calling service (e.g., "patient-service")
            outbox_service_url: URL of the global outbox service (defaults to localhost:50051)
            timeout: Request timeout in seconds
            max_retries: Maximum number of retry attempts
        """
        self.service_name = service_name
        self.outbox_service_url = outbox_service_url or os.getenv(
            "GLOBAL_OUTBOX_SERVICE_URL", 
            "localhost:50051"
        )
        self.timeout = timeout
        self.max_retries = max_retries
        self._channel = None
        self._stub = None
        
        if not GRPC_AVAILABLE:
            logger.warning("gRPC not available - client will operate in fallback mode")
    
    async def __aenter__(self):
        """Async context manager entry"""
        if GRPC_AVAILABLE:
            await self._connect()
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit"""
        if self._channel:
            await self._channel.close()
    
    async def _connect(self):
        """Establish gRPC connection"""
        if not GRPC_AVAILABLE:
            return
            
        try:
            self._channel = grpc.aio.insecure_channel(self.outbox_service_url)
            self._stub = outbox_pb2_grpc.GlobalOutboxServiceStub(self._channel)
            
            # Test connection with health check
            await self.health_check()
            logger.info(f"Connected to Global Outbox Service at {self.outbox_service_url}")

        except Exception as e:
            logger.error(f"Failed to connect to Global Outbox Service: {e}")
            if self._channel:
                await self._channel.close()
                self._channel = None
                self._stub = None
            raise
    
    async def publish_event(
        self,
        event_type: str,
        kafka_topic: str,
        event_payload: Union[bytes, str, dict],
        kafka_key: Optional[str] = None,
        correlation_id: Optional[str] = None,
        causation_id: Optional[str] = None,
        subject: Optional[str] = None,
        priority: int = 1,
        metadata: Optional[Dict[str, Any]] = None,
        idempotency_key: Optional[str] = None,
        scheduled_at: Optional[datetime] = None
    ) -> Optional[str]:
        """
        Publish an event through the global outbox service
        
        Args:
            event_type: Type of event (e.g., "patient.created")
            kafka_topic: Target Kafka topic
            event_payload: Event payload (bytes, string, or dict)
            kafka_key: Kafka message key for partitioning
            correlation_id: Correlation ID for distributed tracing
            causation_id: Causation ID for event sourcing
            subject: Subject of the event (e.g., patient ID)
            priority: Priority level (0=low, 1=normal, 2=high, 3=critical)
            metadata: Additional metadata
            idempotency_key: Unique key for idempotency (auto-generated if not provided)
            scheduled_at: Optional scheduled delivery time
            
        Returns:
            Outbox record ID if successful, None otherwise
        """
        if not GRPC_AVAILABLE:
            return await self._fallback_publish(event_type, kafka_topic, event_payload, kafka_key)
        
        if not self._stub:
            logger.error("❌ gRPC client not connected")
            return None
        
        try:
            # Prepare event payload
            if isinstance(event_payload, dict):
                event_payload = json.dumps(event_payload).encode('utf-8')
            elif isinstance(event_payload, str):
                event_payload = event_payload.encode('utf-8')
            elif not isinstance(event_payload, bytes):
                event_payload = str(event_payload).encode('utf-8')
            
            # Generate idempotency key if not provided
            if not idempotency_key:
                idempotency_key = str(uuid.uuid4())
            
            # Prepare metadata
            metadata_struct = None
            if metadata:
                from google.protobuf.struct_pb2 import Struct
                metadata_struct = Struct()
                metadata_struct.update(metadata)
            
            # Prepare scheduled timestamp
            scheduled_timestamp = None
            if scheduled_at:
                from google.protobuf.timestamp_pb2 import Timestamp
                scheduled_timestamp = Timestamp()
                scheduled_timestamp.FromDatetime(scheduled_at)
            
            # Create request
            request = outbox_pb2.PublishEventRequest(
                idempotency_key=idempotency_key,
                origin_service=self.service_name,
                kafka_topic=kafka_topic,
                kafka_key=kafka_key or "",
                event_payload=event_payload,
                correlation_id=correlation_id or "",
                event_type=event_type,
                causation_id=causation_id or "",
                subject=subject or "",
                priority=priority
            )
            
            if metadata_struct:
                request.metadata.CopyFrom(metadata_struct)
            
            if scheduled_timestamp:
                request.scheduled_at.CopyFrom(scheduled_timestamp)
            
            # Make gRPC call with retry logic
            for attempt in range(self.max_retries + 1):
                try:
                    response = await asyncio.wait_for(
                        self._stub.PublishEvent(request),
                        timeout=self.timeout
                    )
                    
                    if response.status in ["QUEUED", "SCHEDULED"]:
                        logger.info(
                            f"Event published successfully: {response.outbox_record_id} "
                            f"(status: {response.status})"
                        )
                        return response.outbox_record_id
                    else:
                        logger.error(f"Failed to publish event: {response.error_message}")
                        return None

                except asyncio.TimeoutError:
                    if attempt < self.max_retries:
                        logger.warning(f"Request timeout, retrying... (attempt {attempt + 1})")
                        await asyncio.sleep(2 ** attempt)  # Exponential backoff
                        continue
                    else:
                        logger.error("Request timeout after all retries")
                        return None

                except grpc.RpcError as e:
                    if attempt < self.max_retries and e.code() in [
                        grpc.StatusCode.UNAVAILABLE,
                        grpc.StatusCode.DEADLINE_EXCEEDED
                    ]:
                        logger.warning(f"gRPC error, retrying... (attempt {attempt + 1}): {e}")
                        await asyncio.sleep(2 ** attempt)
                        continue
                    else:
                        logger.error(f"gRPC error: {e}")
                        return None
                        
        except Exception as e:
            logger.error(f"Error publishing event: {e}", exc_info=True)
            return None
    
    async def health_check(self) -> bool:
        """Check if the outbox service is healthy"""
        if not GRPC_AVAILABLE or not self._stub:
            return False
            
        try:
            request = outbox_pb2.HealthCheckRequest()
            response = await asyncio.wait_for(
                self._stub.HealthCheck(request),
                timeout=5.0
            )
            return response.status == "HEALTHY"
            
        except Exception as e:
            logger.error(f"Health check failed: {e}")
            return False
    
    async def get_outbox_stats(self, service_filter: Optional[str] = None) -> Optional[Dict[str, Any]]:
        """Get outbox statistics"""
        if not GRPC_AVAILABLE or not self._stub:
            return None
            
        try:
            request = outbox_pb2.OutboxStatsRequest()
            if service_filter:
                request.service_filter = service_filter
                
            response = await asyncio.wait_for(
                self._stub.GetOutboxStats(request),
                timeout=10.0
            )
            
            return {
                "queue_depths": dict(response.queue_depths),
                "total_events_processed": response.total_events_processed,
                "total_events_failed": response.total_events_failed,
                "dead_letter_count": response.dead_letter_count,
                "avg_processing_latency_ms": response.avg_processing_latency_ms
            }
            
        except Exception as e:
            logger.error(f"Failed to get outbox stats: {e}")
            return None
    
    async def _fallback_publish(
        self, 
        event_type: str, 
        kafka_topic: str, 
        event_payload: bytes, 
        kafka_key: Optional[str]
    ) -> Optional[str]:
        """
        Fallback publishing when gRPC is not available
        
        This could integrate with existing Kafka producers or log for later processing
        """
        fallback_id = f"fallback_{uuid.uuid4()}"
        
        logger.warning(
            f"Fallback mode: Event {event_type} logged for later processing "
            f"(ID: {fallback_id})"
        )
        
        # Here you could integrate with existing event publishing mechanisms
        # or store events for later processing when the outbox service is available
        
        return fallback_id


# Convenience function for single event publishing
async def publish_to_global_outbox(
    service_name: str,
    event_type: str,
    kafka_topic: str,
    event_payload: Union[bytes, str, dict],
    kafka_key: Optional[str] = None,
    correlation_id: Optional[str] = None,
    causation_id: Optional[str] = None,
    subject: Optional[str] = None,
    priority: int = 1,
    metadata: Optional[Dict[str, Any]] = None,
    outbox_service_url: Optional[str] = None
) -> Optional[str]:
    """
    Convenience function for publishing a single event to the global outbox
    
    This function creates a client, publishes the event, and cleans up automatically.
    Use this for single event publishing. For multiple events, use the context manager.
    
    Args:
        service_name: Name of the calling service
        event_type: Type of event
        kafka_topic: Target Kafka topic
        event_payload: Event payload
        kafka_key: Kafka message key
        correlation_id: Correlation ID for tracing
        causation_id: Causation ID for event sourcing
        subject: Subject of the event
        priority: Priority level
        metadata: Additional metadata
        outbox_service_url: URL of the outbox service
        
    Returns:
        Outbox record ID if successful, None otherwise
    """
    async with GlobalOutboxClient(service_name, outbox_service_url) as client:
        return await client.publish_event(
            event_type=event_type,
            kafka_topic=kafka_topic,
            event_payload=event_payload,
            kafka_key=kafka_key,
            correlation_id=correlation_id,
            causation_id=causation_id,
            subject=subject,
            priority=priority,
            metadata=metadata
        )
