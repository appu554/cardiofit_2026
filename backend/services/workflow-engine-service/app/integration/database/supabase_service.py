"""
Supabase service for additional database operations.
"""
import logging
from datetime import datetime
from typing import Dict, List, Optional, Any
from supabase import create_client, Client
from app.core.config import settings

logger = logging.getLogger(__name__)


class SupabaseService:
    """
    Service for interacting with Supabase for additional operations.
    This complements the SQLAlchemy workflow state management.
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
    
    async def get_user_info(self, user_id: str) -> Optional[Dict[str, Any]]:
        """
        Get user information from Supabase auth.users table.
        
        Args:
            user_id: User ID
            
        Returns:
            User information or None if not found
        """
        if not self.initialized or not self.client:
            logger.error("Supabase service not initialized")
            return None
        
        try:
            # Query the auth.users table for user information
            response = self.client.table("auth.users").select("*").eq("id", user_id).execute()
            
            if response.data and len(response.data) > 0:
                return response.data[0]
            else:
                logger.warning(f"User {user_id} not found in Supabase")
                return None
                
        except Exception as e:
            logger.error(f"Error getting user info for {user_id}: {e}")
            return None
    
    async def get_user_roles(self, user_id: str) -> List[str]:
        """
        Get user roles from Supabase.
        
        Args:
            user_id: User ID
            
        Returns:
            List of user roles
        """
        if not self.initialized or not self.client:
            logger.error("Supabase service not initialized")
            return []
        
        try:
            # Get user info which includes app_metadata with roles
            user_info = await self.get_user_info(user_id)
            
            if user_info and "app_metadata" in user_info:
                app_metadata = user_info["app_metadata"]
                if isinstance(app_metadata, dict) and "roles" in app_metadata:
                    return app_metadata["roles"]
            
            return []
            
        except Exception as e:
            logger.error(f"Error getting user roles for {user_id}: {e}")
            return []
    
    async def get_user_permissions(self, user_id: str) -> List[str]:
        """
        Get user permissions from Supabase.
        
        Args:
            user_id: User ID
            
        Returns:
            List of user permissions
        """
        if not self.initialized or not self.client:
            logger.error("Supabase service not initialized")
            return []
        
        try:
            # Get user info which includes app_metadata with permissions
            user_info = await self.get_user_info(user_id)
            
            if user_info and "app_metadata" in user_info:
                app_metadata = user_info["app_metadata"]
                if isinstance(app_metadata, dict) and "permissions" in app_metadata:
                    return app_metadata["permissions"]
            
            return []
            
        except Exception as e:
            logger.error(f"Error getting user permissions for {user_id}: {e}")
            return []
    
    async def log_workflow_event(self, event_data: Dict[str, Any]) -> bool:
        """
        Log workflow events to a Supabase table for analytics.
        
        Args:
            event_data: Event data to log
            
        Returns:
            True if logged successfully, False otherwise
        """
        if not self.initialized or not self.client:
            logger.error("Supabase service not initialized")
            return False
        
        try:
            # Insert event into workflow_events_log table
            response = self.client.table("workflow_events_log").insert(event_data).execute()
            
            if response.data:
                logger.debug(f"Workflow event logged: {event_data.get('event_type', 'unknown')}")
                return True
            else:
                logger.error(f"Failed to log workflow event: {response}")
                return False
                
        except Exception as e:
            logger.error(f"Error logging workflow event: {e}")
            return False
    
    async def get_workflow_analytics(self, filters: Dict[str, Any] = None) -> List[Dict[str, Any]]:
        """
        Get workflow analytics data from Supabase.
        
        Args:
            filters: Optional filters for the query
            
        Returns:
            List of analytics data
        """
        if not self.initialized or not self.client:
            logger.error("Supabase service not initialized")
            return []
        
        try:
            query = self.client.table("workflow_events_log").select("*")
            
            # Apply filters if provided
            if filters:
                for key, value in filters.items():
                    query = query.eq(key, value)
            
            response = query.execute()
            return response.data or []
            
        except Exception as e:
            logger.error(f"Error getting workflow analytics: {e}")
            return []

    async def log_service_task_execution(self, log_entry: Dict[str, Any]) -> bool:
        """
        Log service task execution to Supabase.

        Args:
            log_entry: Service task execution log entry

        Returns:
            True if successful, False otherwise
        """
        try:
            if not self.initialized:
                return False

            response = self.client.table("service_task_logs").insert(log_entry).execute()
            return len(response.data) > 0

        except Exception as e:
            logger.error(f"Error logging service task execution: {e}")
            return False

    async def store_event(self, event: Dict[str, Any]) -> bool:
        """
        Store an event in the event store.

        Args:
            event: Event to store

        Returns:
            True if successful, False otherwise
        """
        try:
            if not self.initialized:
                return False

            response = self.client.table("event_store").insert(event).execute()
            return len(response.data) > 0

        except Exception as e:
            logger.error(f"Error storing event: {e}")
            return False

    async def get_events_since(self, timestamp: datetime) -> List[Dict[str, Any]]:
        """
        Get events from the event store since a specific timestamp.

        Args:
            timestamp: Timestamp to get events since

        Returns:
            List of events
        """
        try:
            if not self.initialized:
                return []

            response = self.client.table("event_store").select("*").gte(
                "created_at", timestamp.isoformat()
            ).order("created_at", desc=False).execute()

            return response.data

        except Exception as e:
            logger.error(f"Error getting events since timestamp: {e}")
            return []

    async def log_event_processing(self, log_entry: Dict[str, Any]) -> bool:
        """
        Log event processing to Supabase.

        Args:
            log_entry: Event processing log entry

        Returns:
            True if successful, False otherwise
        """
        try:
            if not self.initialized:
                return False

            response = self.client.table("event_processing_logs").insert(log_entry).execute()
            return len(response.data) > 0

        except Exception as e:
            logger.error(f"Error logging event processing: {e}")
            return False


# Global service instance
supabase_service = SupabaseService()
