"""
Global Outbox Service Adapter for Device Data Ingestion Service

This adapter replaces the vendor-specific outbox implementation with
Global Outbox Service integration while maintaining all existing functionality
and performance characteristics.

Key Features:
- Maintains vendor-specific routing and metadata
- Preserves all existing API contracts
- Provides fallback to local outbox if Global Outbox Service is unavailable
- Supports all device data types and medical device vendors
- Maintains transactional guarantees
"""

import asyncio
import json
import logging
import uuid
from datetime import datetime
from typing import Dict, Any, Optional, List
import sys
import os

# Add shared directory to path for outbox client
# From device-data-ingestion-service/app/services/ to services/shared/
current_dir = os.path.dirname(__file__)  # app/services/
app_dir = os.path.dirname(current_dir)  # app/
service_dir = os.path.dirname(app_dir)  # device-data-ingestion-service/
services_dir = os.path.dirname(service_dir)  # services/
shared_path = os.path.join(services_dir, 'shared')
if shared_path not in sys.path:
    sys.path.append(shared_path)

try:
    from outbox_client import GlobalOutboxClient, publish_to_global_outbox
    GLOBAL_OUTBOX_CLIENT_AVAILABLE = True
except ImportError as e:
    logger.warning(f"Global Outbox Client not available: {e}")
    logger.warning("Falling back to local outbox service only")
    GlobalOutboxClient = None
    publish_to_global_outbox = None
    GLOBAL_OUTBOX_CLIENT_AVAILABLE = False

# Import existing outbox service for fallback
from app.services.outbox_service import VendorAwareOutboxService

logger = logging.getLogger(__name__)


class GlobalOutboxAdapter:
    """
    Adapter that integrates Device Data Ingestion Service with Global Outbox Service
    
    This adapter provides a seamless migration path from vendor-specific outbox tables
    to the centralized Global Outbox Service while maintaining all existing functionality.
    """
    
    def __init__(self, 
                 global_outbox_url: Optional[str] = None,
                 fallback_enabled: bool = True,
                 service_name: str = "device-data-ingestion-service"):
        """
        Initialize the Global Outbox Adapter
        
        Args:
            global_outbox_url: URL of the Global Outbox Service (defaults to localhost:50051)
            fallback_enabled: Whether to fallback to local outbox if Global Outbox is unavailable
            service_name: Name of this service for Global Outbox routing
        """
        self.global_outbox_url = global_outbox_url or os.getenv(
            "GLOBAL_OUTBOX_SERVICE_URL", 
            "localhost:50051"
        )
        self.fallback_enabled = fallback_enabled
        self.service_name = service_name
        
        # Initialize fallback service if enabled
        self.fallback_service = VendorAwareOutboxService() if fallback_enabled else None
        
        # Track Global Outbox Service availability
        self._global_outbox_available = None
        self._last_health_check = None
        self._health_check_interval = 30  # seconds
        
        logger.info(f"Global Outbox Adapter initialized for {service_name}")
        logger.info(f"Global Outbox URL: {self.global_outbox_url}")
        logger.info(f"Fallback enabled: {fallback_enabled}")
    
    async def _check_global_outbox_health(self) -> bool:
        """
        Check if Global Outbox Service is available and healthy

        Uses caching to avoid excessive health checks
        """
        if not GLOBAL_OUTBOX_CLIENT_AVAILABLE:
            return False

        now = datetime.now()

        # Use cached result if recent
        if (self._last_health_check and
            (now - self._last_health_check).total_seconds() < self._health_check_interval):
            return self._global_outbox_available

        try:
            async with GlobalOutboxClient(self.service_name, self.global_outbox_url) as client:
                is_healthy = await client.health_check()
                self._global_outbox_available = is_healthy
                self._last_health_check = now

                if is_healthy:
                    logger.debug("Global Outbox Service is healthy")
                else:
                    logger.warning("Global Outbox Service health check failed")

                return is_healthy

        except Exception as e:
            logger.warning(f"Global Outbox Service health check error: {e}")
            self._global_outbox_available = False
            self._last_health_check = now
            return False
    
    async def store_device_data_transactionally(
        self,
        device_data: Dict[str, Any],
        vendor_id: str,
        correlation_id: Optional[str] = None,
        trace_id: Optional[str] = None
    ) -> str:
        """
        Store device data using Global Outbox Service with fallback to local outbox
        
        This method maintains the exact same API as the original VendorAwareOutboxService
        while routing events through the Global Outbox Service.
        
        Args:
            device_data: Device reading data
            vendor_id: Vendor identifier (e.g., 'fitbit', 'garmin', 'apple_health')
            correlation_id: Request correlation ID
            trace_id: Distributed tracing ID
            
        Returns:
            str: Outbox record ID
        """
        # Check Global Outbox Service availability
        use_global_outbox = await self._check_global_outbox_health()
        
        if use_global_outbox:
            try:
                return await self._store_via_global_outbox(
                    device_data, vendor_id, correlation_id, trace_id
                )
            except Exception as e:
                logger.error(f"❌ Global Outbox Service failed: {e}")
                
                # Fall back to local outbox if enabled
                if self.fallback_enabled:
                    logger.info("Falling back to local outbox service")
                    return await self._store_via_fallback(
                        device_data, vendor_id, correlation_id, trace_id
                    )
                else:
                    raise
        else:
            # Use fallback service if Global Outbox is unavailable
            if self.fallback_enabled:
                logger.info("Using local outbox service (Global Outbox unavailable)")
                return await self._store_via_fallback(
                    device_data, vendor_id, correlation_id, trace_id
                )
            else:
                raise RuntimeError("Global Outbox Service unavailable and fallback disabled")
    
    async def _store_via_global_outbox(
        self,
        device_data: Dict[str, Any],
        vendor_id: str,
        correlation_id: Optional[str] = None,
        trace_id: Optional[str] = None
    ) -> str:
        """Store device data via Global Outbox Service"""

        if not GLOBAL_OUTBOX_CLIENT_AVAILABLE:
            raise RuntimeError("Global Outbox Client not available")

        # Prepare event metadata with vendor-specific information
        metadata = {
            "vendor_id": vendor_id,
            "device_id": device_data.get("device_id"),
            "reading_type": device_data.get("reading_type"),
            "timestamp": device_data.get("timestamp"),
            "patient_id": device_data.get("patient_id"),
            "service_origin": self.service_name
        }

        # Add trace information if available
        if trace_id:
            metadata["trace_id"] = trace_id

        # Determine event type based on device data
        event_type = f"device.data.{vendor_id}.{device_data.get('reading_type', 'unknown')}"

        # Determine Kafka topic (vendor-specific or unified)
        kafka_topic = self._get_kafka_topic_for_vendor(vendor_id)

        # Use device_id as Kafka key for partitioning
        kafka_key = device_data.get("device_id", device_data.get("patient_id"))

        # Determine priority based on medical context
        priority = self._determine_medical_priority(device_data)

        try:
            # Publish to Global Outbox Service
            outbox_record_id = await publish_to_global_outbox(
                service_name=self.service_name,
                event_type=event_type,
                kafka_topic=kafka_topic,
                event_payload=json.dumps(device_data).encode('utf-8'),
                kafka_key=kafka_key,
                correlation_id=correlation_id,
                causation_id=trace_id,
                subject=device_data.get("patient_id"),
                priority=priority,
                metadata=metadata,
                outbox_service_url=self.global_outbox_url
            )
            
            if outbox_record_id:
                logger.info(f"Device data stored via Global Outbox: {outbox_record_id}", extra={
                    'vendor_id': vendor_id,
                    'device_id': device_data.get('device_id'),
                    'correlation_id': correlation_id,
                    'outbox_record_id': outbox_record_id
                })
                return outbox_record_id
            else:
                raise RuntimeError("Global Outbox Service returned no record ID")

        except Exception as e:
            logger.error(f"Failed to store via Global Outbox: {e}")
            raise
    
    async def _store_via_fallback(
        self,
        device_data: Dict[str, Any],
        vendor_id: str,
        correlation_id: Optional[str] = None,
        trace_id: Optional[str] = None
    ) -> str:
        """Store device data via fallback local outbox service"""
        
        if not self.fallback_service:
            raise RuntimeError("Fallback service not available")
        
        return await self.fallback_service.store_device_data_transactionally(
            device_data=device_data,
            vendor_id=vendor_id,
            correlation_id=correlation_id,
            trace_id=trace_id
        )
    
    def _get_kafka_topic_for_vendor(self, vendor_id: str) -> str:
        """Get the appropriate Kafka topic for a vendor"""
        # Vendor-specific topic mapping
        vendor_topics = {
            'fitbit': 'raw-device-data.v1',
            'garmin': 'raw-device-data.v1', 
            'apple_health': 'raw-device-data.v1',
            'samsung_health': 'raw-device-data.v1',
            'google_fit': 'raw-device-data.v1'
        }
        
        return vendor_topics.get(vendor_id, 'raw-device-data.v1')
    
    def _determine_medical_priority(self, device_data: Dict[str, Any]) -> int:
        """
        Determine medical priority based on device data content
        
        Priority levels:
        0 = Low (routine data)
        1 = Normal (standard readings)
        2 = High (abnormal readings)
        3 = Critical (emergency readings)
        """
        reading_type = device_data.get('reading_type', '').lower()
        value = device_data.get('value')
        
        # Critical readings that require immediate attention
        critical_readings = ['heart_rate', 'blood_pressure', 'oxygen_saturation', 'glucose']
        
        if reading_type in critical_readings and value:
            # Apply medical thresholds for critical priority
            if reading_type == 'heart_rate' and (float(value) > 120 or float(value) < 50):
                return 3  # Critical
            elif reading_type == 'oxygen_saturation' and float(value) < 90:
                return 3  # Critical
            elif reading_type in critical_readings:
                return 2  # High priority for critical reading types
        
        # High priority for any abnormal readings
        if 'abnormal' in str(device_data.get('metadata', {})).lower():
            return 2
        
        # Normal priority for standard readings
        return 1

    async def get_pending_messages_for_vendor(
        self,
        vendor_id: str,
        limit: int = 100
    ) -> List[Dict[str, Any]]:
        """
        Get pending messages for a vendor

        This method provides compatibility with the existing background publisher
        by delegating to the Global Outbox Service or fallback service.
        """
        # For Global Outbox Service, we don't need vendor-specific polling
        # The Global Outbox Service handles all pending messages centrally

        if self.fallback_enabled and self.fallback_service:
            # If using fallback, delegate to the original service
            use_global_outbox = await self._check_global_outbox_health()

            if not use_global_outbox:
                return await self.fallback_service.get_pending_messages_for_vendor(
                    vendor_id=vendor_id,
                    limit=limit
                )

        # For Global Outbox Service, return empty list since it handles publishing
        return []

    async def mark_message_as_published(
        self,
        message_id: str,
        vendor_id: str
    ) -> bool:
        """
        Mark a message as published

        For Global Outbox Service, this is handled automatically.
        For fallback service, delegate to the original implementation.
        """
        use_global_outbox = await self._check_global_outbox_health()

        if not use_global_outbox and self.fallback_enabled and self.fallback_service:
            return await self.fallback_service.mark_message_as_published(
                message_id=message_id,
                vendor_id=vendor_id
            )

        # For Global Outbox Service, messages are marked as published automatically
        return True

    async def mark_message_as_failed(
        self,
        message_id: str,
        vendor_id: str,
        error_message: str
    ) -> bool:
        """
        Mark a message as failed

        For Global Outbox Service, this is handled automatically.
        For fallback service, delegate to the original implementation.
        """
        use_global_outbox = await self._check_global_outbox_health()

        if not use_global_outbox and self.fallback_enabled and self.fallback_service:
            return await self.fallback_service.mark_message_as_failed(
                message_id=message_id,
                vendor_id=vendor_id,
                error_message=error_message
            )

        # For Global Outbox Service, failed messages are handled automatically
        return True

    async def get_outbox_statistics(self) -> Dict[str, Any]:
        """
        Get outbox statistics

        Returns statistics from Global Outbox Service or fallback service
        """
        use_global_outbox = await self._check_global_outbox_health()

        if use_global_outbox and GLOBAL_OUTBOX_CLIENT_AVAILABLE:
            try:
                async with GlobalOutboxClient(self.service_name, self.global_outbox_url) as client:
                    stats = await client.get_outbox_stats(service_filter=self.service_name)

                    if stats:
                        return {
                            "service": self.service_name,
                            "global_outbox_enabled": True,
                            "queue_depth": stats.get("queue_depths", {}).get(self.service_name, 0),
                            "total_events_processed": stats.get("total_events_processed", 0),
                            "total_events_failed": stats.get("total_events_failed", 0),
                            "dead_letter_count": stats.get("dead_letter_count", 0),
                            "avg_processing_latency_ms": stats.get("avg_processing_latency_ms", 0)
                        }
            except Exception as e:
                logger.error(f"Failed to get Global Outbox stats: {e}")

        # Fallback to local statistics
        if self.fallback_enabled and self.fallback_service:
            try:
                # Use get_health_status instead of get_outbox_statistics
                fallback_health = await self.fallback_service.get_health_status()
                queue_depths = await self.fallback_service.get_queue_depths()

                fallback_stats = {
                    "service": self.service_name,
                    "global_outbox_enabled": False,
                    "fallback_mode": True,
                    "queue_depth": sum(queue_depths.values()) if queue_depths else 0,
                    "vendor_queue_depths": queue_depths,
                    "fallback_health": fallback_health
                }
                return fallback_stats
            except Exception as e:
                logger.error(f"Failed to get fallback statistics: {e}")
                return {
                    "service": self.service_name,
                    "global_outbox_enabled": False,
                    "fallback_mode": True,
                    "error": f"Fallback statistics failed: {e}"
                }

        return {
            "service": self.service_name,
            "global_outbox_enabled": False,
            "error": "No statistics available"
        }

    async def health_check(self) -> Dict[str, Any]:
        """
        Perform health check for the adapter

        Returns health status of both Global Outbox Service and fallback service
        """
        health_status = {
            "service": self.service_name,
            "adapter_healthy": True,
            "global_outbox_available": False,
            "fallback_available": False,
            "active_mode": "unknown"
        }

        # Check Global Outbox Service
        try:
            global_outbox_healthy = await self._check_global_outbox_health()
            health_status["global_outbox_available"] = global_outbox_healthy

            if global_outbox_healthy:
                health_status["active_mode"] = "global_outbox"
        except Exception as e:
            logger.error(f"Global Outbox health check failed: {e}")
            health_status["global_outbox_error"] = str(e)

        # Check fallback service
        if self.fallback_enabled and self.fallback_service:
            try:
                fallback_health = await self.fallback_service.get_health_status()
                health_status["fallback_available"] = fallback_health.get("status") == "healthy"

                if not health_status["global_outbox_available"] and health_status["fallback_available"]:
                    health_status["active_mode"] = "fallback"
            except Exception as e:
                logger.error(f"Fallback service health check failed: {e}")
                health_status["fallback_error"] = str(e)

        # Determine overall health
        health_status["adapter_healthy"] = (
            health_status["global_outbox_available"] or
            health_status["fallback_available"]
        )

        return health_status


# Global instance for easy import
global_outbox_adapter = GlobalOutboxAdapter()
