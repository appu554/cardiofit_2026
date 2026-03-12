"""
Supabase Service for Device Data Ingestion Service

This service provides Supabase client functionality similar to other services
in the project, following the same patterns as auth-service and workflow-engine-service.
"""
import logging
from typing import Optional, Dict, Any, List
from supabase import create_client, Client

from app.config import settings

logger = logging.getLogger(__name__)


class SupabaseService:
    """
    Service for interacting with Supabase using the Supabase SDK.
    This complements the SQLAlchemy outbox operations.
    """
    
    def __init__(self):
        self.client: Optional[Client] = None
        self.initialized = False
    
    async def initialize(self) -> bool:
        """
        Initialize the Supabase client.
        
        Returns:
            bool: True if initialization successful, False otherwise
        """
        try:
            self.client = create_client(settings.SUPABASE_URL, settings.SUPABASE_KEY)
            self.initialized = True
            logger.info("Supabase client initialized successfully")
            return True
        except Exception as e:
            logger.error(f"Failed to initialize Supabase client: {e}")
            self.initialized = False
            return False
    
    def get_client(self) -> Optional[Client]:
        """
        Get the Supabase client instance.
        
        Returns:
            Optional[Client]: Supabase client if initialized, None otherwise
        """
        if not self.initialized:
            logger.warning("Supabase client not initialized")
            return None
        return self.client
    
    async def health_check(self) -> bool:
        """
        Perform a health check on the Supabase connection.
        
        Returns:
            bool: True if connection is healthy, False otherwise
        """
        try:
            if not self.client:
                return False
            
            # Try a simple query to test connection
            response = self.client.table('vendor_outbox_registry').select('vendor_id').limit(1).execute()
            return response.data is not None
            
        except Exception as e:
            logger.error(f"Supabase health check failed: {e}")
            return False
    
    async def get_vendor_registry(self) -> List[Dict[str, Any]]:
        """
        Get the vendor registry from Supabase.
        
        Returns:
            List[Dict[str, Any]]: List of vendor configurations
        """
        try:
            if not self.client:
                raise RuntimeError("Supabase client not initialized")
            
            response = self.client.table('vendor_outbox_registry').select('*').eq('is_active', True).execute()
            return response.data or []
            
        except Exception as e:
            logger.error(f"Failed to get vendor registry: {e}")
            return []
    
    async def get_outbox_queue_depths(self) -> Dict[str, Dict[str, int]]:
        """
        Get queue depths for all vendor outbox tables.
        
        Returns:
            Dict[str, Dict[str, int]]: Queue depths by vendor and status
        """
        try:
            if not self.client:
                raise RuntimeError("Supabase client not initialized")
            
            vendors = await self.get_vendor_registry()
            queue_depths = {}
            
            for vendor in vendors:
                vendor_id = vendor['vendor_id']
                outbox_table = vendor['outbox_table_name']
                
                try:
                    # Get counts by status
                    pending_response = self.client.table(outbox_table).select('id', count='exact').eq('status', 'pending').execute()
                    processing_response = self.client.table(outbox_table).select('id', count='exact').eq('status', 'processing').execute()
                    failed_response = self.client.table(outbox_table).select('id', count='exact').eq('status', 'failed').execute()
                    
                    queue_depths[vendor_id] = {
                        'pending': pending_response.count or 0,
                        'processing': processing_response.count or 0,
                        'failed': failed_response.count or 0
                    }
                    
                except Exception as e:
                    logger.warning(f"Failed to get queue depth for {vendor_id}: {e}")
                    queue_depths[vendor_id] = {
                        'pending': 0,
                        'processing': 0,
                        'failed': 0,
                        'error': str(e)
                    }
            
            return queue_depths
            
        except Exception as e:
            logger.error(f"Failed to get outbox queue depths: {e}")
            return {}
    
    async def get_dead_letter_counts(self) -> Dict[str, int]:
        """
        Get dead letter counts for all vendors.
        
        Returns:
            Dict[str, int]: Dead letter counts by vendor
        """
        try:
            if not self.client:
                raise RuntimeError("Supabase client not initialized")
            
            vendors = await self.get_vendor_registry()
            dead_letter_counts = {}
            
            for vendor in vendors:
                vendor_id = vendor['vendor_id']
                dead_letter_table = vendor['dead_letter_table_name']
                
                try:
                    response = self.client.table(dead_letter_table).select('id', count='exact').execute()
                    dead_letter_counts[vendor_id] = response.count or 0
                    
                except Exception as e:
                    logger.warning(f"Failed to get dead letter count for {vendor_id}: {e}")
                    dead_letter_counts[vendor_id] = 0
            
            return dead_letter_counts
            
        except Exception as e:
            logger.error(f"Failed to get dead letter counts: {e}")
            return {}
    
    async def create_vendor_registry_entry(self, vendor_config: Dict[str, Any]) -> bool:
        """
        Create a new vendor registry entry.
        
        Args:
            vendor_config: Vendor configuration dictionary
            
        Returns:
            bool: True if successful, False otherwise
        """
        try:
            if not self.client:
                raise RuntimeError("Supabase client not initialized")
            
            response = self.client.table('vendor_outbox_registry').insert(vendor_config).execute()
            return len(response.data) > 0
            
        except Exception as e:
            logger.error(f"Failed to create vendor registry entry: {e}")
            return False
    
    async def update_vendor_registry_entry(self, vendor_id: str, updates: Dict[str, Any]) -> bool:
        """
        Update a vendor registry entry.
        
        Args:
            vendor_id: Vendor ID to update
            updates: Dictionary of updates to apply
            
        Returns:
            bool: True if successful, False otherwise
        """
        try:
            if not self.client:
                raise RuntimeError("Supabase client not initialized")
            
            response = self.client.table('vendor_outbox_registry').update(updates).eq('vendor_id', vendor_id).execute()
            return len(response.data) > 0
            
        except Exception as e:
            logger.error(f"Failed to update vendor registry entry: {e}")
            return False
    
    async def get_service_metrics(self) -> Dict[str, Any]:
        """
        Get service metrics from Supabase.
        
        Returns:
            Dict[str, Any]: Service metrics
        """
        try:
            queue_depths = await self.get_outbox_queue_depths()
            dead_letter_counts = await self.get_dead_letter_counts()
            vendor_registry = await self.get_vendor_registry()
            
            total_pending = sum(depths.get('pending', 0) for depths in queue_depths.values())
            total_processing = sum(depths.get('processing', 0) for depths in queue_depths.values())
            total_failed = sum(depths.get('failed', 0) for depths in queue_depths.values())
            total_dead_letters = sum(dead_letter_counts.values())
            
            return {
                'vendor_count': len(vendor_registry),
                'total_pending_messages': total_pending,
                'total_processing_messages': total_processing,
                'total_failed_messages': total_failed,
                'total_dead_letter_messages': total_dead_letters,
                'queue_depths': queue_depths,
                'dead_letter_counts': dead_letter_counts,
                'health_status': 'healthy' if total_failed == 0 else 'degraded'
            }
            
        except Exception as e:
            logger.error(f"Failed to get service metrics: {e}")
            return {
                'error': str(e),
                'health_status': 'unhealthy'
            }


# Global Supabase service instance
supabase_service = SupabaseService()
