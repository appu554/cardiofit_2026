"""
Supabase-Only Outbox Service

Alternative implementation using only Supabase SDK instead of direct PostgreSQL.
This works when REST API is accessible but direct database connection fails.
"""
import logging
import uuid
from datetime import datetime
from typing import Dict, Any, List, Optional
from dataclasses import dataclass

from supabase import create_client, Client
from app.config import settings

logger = logging.getLogger(__name__)


@dataclass
class OutboxMessage:
    """Outbox message data structure"""
    id: str
    device_id: str
    event_type: str
    event_payload: Dict[str, Any]
    kafka_topic: str
    kafka_key: Optional[str]
    created_at: datetime
    status: str
    correlation_id: Optional[str]
    trace_id: Optional[str]
    retry_count: int = 0
    max_retries: int = 3


class SupabaseOutboxService:
    """
    Outbox service using Supabase SDK instead of direct PostgreSQL.
    Provides the same functionality as the PostgreSQL version.
    """
    
    def __init__(self):
        self.client: Optional[Client] = None
        self.initialized = False
        
        # Vendor configuration
        self.vendor_tables = {
            "fitbit": "fitbit_outbox",
            "garmin": "garmin_outbox", 
            "apple_health": "apple_health_outbox",
            "medical_device": "medical_device_outbox",
            "generic_device": "generic_device_outbox"
        }
    
    async def initialize(self) -> bool:
        """Initialize the Supabase client"""
        try:
            self.client = create_client(settings.SUPABASE_URL, settings.SUPABASE_KEY)
            self.initialized = True
            logger.info("Supabase outbox service initialized")
            return True
        except Exception as e:
            logger.error(f"Failed to initialize Supabase outbox service: {e}")
            return False
    
    async def store_device_data_transactionally(
        self,
        device_data: Dict[str, Any],
        vendor_id: str,
        correlation_id: str,
        trace_id: str
    ) -> Optional[str]:
        """
        Store device data in vendor-specific outbox table.
        
        Args:
            device_data: Device data to store
            vendor_id: Vendor identifier
            correlation_id: Correlation ID for tracing
            trace_id: Trace ID for monitoring
            
        Returns:
            Optional[str]: Outbox message ID if successful, None otherwise
        """
        if not self.initialized or not self.client:
            logger.error("Supabase outbox service not initialized")
            return None
        
        try:
            # Get the appropriate table for this vendor
            table_name = self.vendor_tables.get(vendor_id, "generic_device_outbox")
            
            # Create outbox message
            outbox_message = {
                "id": str(uuid.uuid4()),
                "device_id": device_data.get("device_id", "unknown"),
                "event_type": "device_reading",
                "event_payload": device_data,
                "kafka_topic": "raw-device-data.v1",
                "kafka_key": f"{vendor_id}:{device_data.get('device_id', 'unknown')}",
                "created_at": datetime.utcnow().isoformat(),
                "status": "pending",
                "correlation_id": correlation_id,
                "trace_id": trace_id,
                "retry_count": 0,
                "max_retries": 3
            }
            
            # Insert into Supabase table
            response = self.client.table(table_name).insert(outbox_message).execute()
            
            if response.data and len(response.data) > 0:
                message_id = response.data[0]["id"]
                logger.info(f"Device data stored in {table_name}: {message_id}")
                return message_id
            else:
                logger.error(f"Failed to store device data in {table_name}")
                return None
                
        except Exception as e:
            logger.error(f"Error storing device data in outbox: {e}")
            return None
    
    async def get_pending_messages(self, vendor_id: str, limit: int = 50) -> List[OutboxMessage]:
        """
        Get pending messages for a vendor.
        
        Args:
            vendor_id: Vendor identifier
            limit: Maximum number of messages to retrieve
            
        Returns:
            List[OutboxMessage]: List of pending messages
        """
        if not self.initialized or not self.client:
            return []
        
        try:
            table_name = self.vendor_tables.get(vendor_id, "generic_device_outbox")
            
            response = self.client.table(table_name)\
                .select("*")\
                .eq("status", "pending")\
                .order("created_at")\
                .limit(limit)\
                .execute()
            
            messages = []
            for row in response.data or []:
                message = OutboxMessage(
                    id=row["id"],
                    device_id=row["device_id"],
                    event_type=row["event_type"],
                    event_payload=row["event_payload"],
                    kafka_topic=row["kafka_topic"],
                    kafka_key=row.get("kafka_key"),
                    created_at=datetime.fromisoformat(row["created_at"].replace('Z', '+00:00')),
                    status=row["status"],
                    correlation_id=row.get("correlation_id"),
                    trace_id=row.get("trace_id"),
                    retry_count=row.get("retry_count", 0),
                    max_retries=row.get("max_retries", 3)
                )
                messages.append(message)
            
            return messages
            
        except Exception as e:
            logger.error(f"Error getting pending messages for {vendor_id}: {e}")
            return []
    
    async def mark_message_processing(self, vendor_id: str, message_id: str) -> bool:
        """Mark a message as processing"""
        if not self.initialized or not self.client:
            return False
        
        try:
            table_name = self.vendor_tables.get(vendor_id, "generic_device_outbox")
            
            response = self.client.table(table_name)\
                .update({"status": "processing"})\
                .eq("id", message_id)\
                .execute()
            
            return len(response.data or []) > 0
            
        except Exception as e:
            logger.error(f"Error marking message as processing: {e}")
            return False
    
    async def mark_message_completed(self, vendor_id: str, message_id: str) -> bool:
        """Mark a message as completed"""
        if not self.initialized or not self.client:
            return False
        
        try:
            table_name = self.vendor_tables.get(vendor_id, "generic_device_outbox")
            
            response = self.client.table(table_name)\
                .update({
                    "status": "completed",
                    "processed_at": datetime.utcnow().isoformat()
                })\
                .eq("id", message_id)\
                .execute()
            
            return len(response.data or []) > 0
            
        except Exception as e:
            logger.error(f"Error marking message as completed: {e}")
            return False
    
    async def mark_message_failed(self, vendor_id: str, message_id: str, error: str) -> bool:
        """Mark a message as failed and increment retry count"""
        if not self.initialized or not self.client:
            return False
        
        try:
            table_name = self.vendor_tables.get(vendor_id, "generic_device_outbox")
            
            # Get current retry count
            current_response = self.client.table(table_name)\
                .select("retry_count, max_retries")\
                .eq("id", message_id)\
                .execute()
            
            if not current_response.data:
                return False
            
            current_retry = current_response.data[0].get("retry_count", 0)
            max_retries = current_response.data[0].get("max_retries", 3)
            new_retry_count = current_retry + 1
            
            # Update message
            update_data = {
                "retry_count": new_retry_count,
                "last_error": error,
                "status": "failed" if new_retry_count >= max_retries else "pending"
            }
            
            response = self.client.table(table_name)\
                .update(update_data)\
                .eq("id", message_id)\
                .execute()
            
            return len(response.data or []) > 0
            
        except Exception as e:
            logger.error(f"Error marking message as failed: {e}")
            return False
    
    async def get_queue_depths(self) -> Dict[str, Dict[str, int]]:
        """Get queue depths for all vendors"""
        if not self.initialized or not self.client:
            return {}
        
        queue_depths = {}
        
        for vendor_id, table_name in self.vendor_tables.items():
            try:
                # Get counts by status
                pending_response = self.client.table(table_name)\
                    .select("id", count="exact")\
                    .eq("status", "pending")\
                    .execute()
                
                processing_response = self.client.table(table_name)\
                    .select("id", count="exact")\
                    .eq("status", "processing")\
                    .execute()
                
                failed_response = self.client.table(table_name)\
                    .select("id", count="exact")\
                    .eq("status", "failed")\
                    .execute()
                
                queue_depths[vendor_id] = {
                    "pending": pending_response.count or 0,
                    "processing": processing_response.count or 0,
                    "failed": failed_response.count or 0
                }
                
            except Exception as e:
                logger.warning(f"Error getting queue depth for {vendor_id}: {e}")
                queue_depths[vendor_id] = {"pending": 0, "processing": 0, "failed": 0}
        
        return queue_depths
    
    async def health_check(self) -> Dict[str, Any]:
        """Perform health check"""
        if not self.initialized or not self.client:
            return {"status": "unhealthy", "error": "Not initialized"}
        
        try:
            # Test connection by querying vendor registry
            response = self.client.table("vendor_outbox_registry")\
                .select("vendor_id")\
                .limit(1)\
                .execute()
            
            return {
                "status": "healthy",
                "initialized": True,
                "vendor_tables": len(self.vendor_tables)
            }
            
        except Exception as e:
            return {
                "status": "unhealthy", 
                "error": str(e),
                "initialized": self.initialized
            }


# Global Supabase outbox service instance
supabase_outbox_service = SupabaseOutboxService()
