"""
Google Healthcare API FHIR service for Workflow Engine Service.
Handles FHIR PlanDefinition and Task resources.
"""
import os
import sys
import logging
from typing import Dict, List, Optional, Any
from app.core.config import settings

logger = logging.getLogger(__name__)


class GoogleFHIRService:
    """
    Service for interacting with Google Healthcare API FHIR store.
    Handles PlanDefinition and Task resources for workflow management.
    """
    
    def __init__(self):
        self.client = None
        self.initialized = False
        self.mock_mode = False
    
    async def initialize(self) -> bool:
        """
        Initialize the Google Healthcare API client.

        Returns:
            bool: True if initialization successful, False otherwise
        """
        try:
            # Add the project root to Python path
            project_root = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../../.."))
            if project_root not in sys.path:
                sys.path.insert(0, project_root)
                
            # Add the services directory to Python path
            services_dir = os.path.join(project_root, "services")
            if services_dir not in sys.path:
                sys.path.insert(0, services_dir)
                
            # Now import the Google Healthcare client
            from shared.google_healthcare.client import GoogleHealthcareClient

            # Initialize the client with settings (same pattern as other services)
            self.client = GoogleHealthcareClient(
                project_id=settings.GOOGLE_CLOUD_PROJECT,
                location=settings.GOOGLE_CLOUD_LOCATION,
                dataset_id=settings.GOOGLE_CLOUD_DATASET,
                fhir_store_id=settings.GOOGLE_CLOUD_FHIR_STORE,
                credentials_path=settings.GOOGLE_APPLICATION_CREDENTIALS
            )

            # Initialize the client (synchronous call like other services)
            success = self.client.initialize()
            if success:
                self.initialized = True
                self.mock_mode = False
                logger.info("Google Healthcare API client initialized successfully")
                return True
            else:
                logger.error("Failed to initialize Google Healthcare API client")
                self.initialized = False
                self.mock_mode = True
                return False

        except Exception as e:
            logger.error(f"Failed to initialize Google Healthcare API client: {e}")
            logger.info("Falling back to mock mode")
            self.initialized = False
            self.mock_mode = True
            return False
    
    async def create_plan_definition(self, plan_definition: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Create a FHIR PlanDefinition resource.

        Args:
            plan_definition: FHIR PlanDefinition resource data

        Returns:
            Created PlanDefinition resource or None if failed
        """
        if not self.initialized:
            logger.error("Google FHIR service not initialized")
            return None

        if self.mock_mode:
            logger.info("Mock mode: PlanDefinition creation simulated")
            return {"id": "mock-plan-def-001", "resourceType": "PlanDefinition", **plan_definition}

        try:
            return await self.client.create_resource("PlanDefinition", plan_definition)
        except Exception as e:
            logger.error(f"Error creating PlanDefinition: {e}")
            return None
    
    async def get_plan_definition(self, plan_definition_id: str) -> Optional[Dict[str, Any]]:
        """
        Get a FHIR PlanDefinition resource by ID.

        Args:
            plan_definition_id: FHIR PlanDefinition ID

        Returns:
            PlanDefinition resource or None if not found
        """
        if not self.initialized:
            logger.error("Google FHIR service not initialized")
            return None

        if self.mock_mode:
            logger.info(f"Mock mode: PlanDefinition {plan_definition_id} retrieval simulated")
            return {"id": plan_definition_id, "resourceType": "PlanDefinition", "status": "active"}

        try:
            return await self.client.get_resource("PlanDefinition", plan_definition_id)
        except Exception as e:
            logger.error(f"Error getting PlanDefinition {plan_definition_id}: {e}")
            return None
    
    async def update_plan_definition(self, plan_definition_id: str, plan_definition: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Update a FHIR PlanDefinition resource.
        
        Args:
            plan_definition_id: FHIR PlanDefinition ID
            plan_definition: Updated PlanDefinition resource data
            
        Returns:
            Updated PlanDefinition resource or None if failed
        """
        if not self.initialized:
            logger.error("Google FHIR service not initialized")
            return None
        
        try:
            return await self.client.update_resource("PlanDefinition", plan_definition_id, plan_definition)
        except Exception as e:
            logger.error(f"Error updating PlanDefinition {plan_definition_id}: {e}")
            return None
    
    async def search_plan_definitions(self, params: Dict[str, str] = None) -> List[Dict[str, Any]]:
        """
        Search for FHIR PlanDefinition resources.
        
        Args:
            params: Search parameters
            
        Returns:
            List of PlanDefinition resources
        """
        if not self.initialized:
            logger.error("Google FHIR service not initialized")
            return []
        
        try:
            return await self.client.search_resources("PlanDefinition", params or {})
        except Exception as e:
            logger.error(f"Error searching PlanDefinitions: {e}")
            return []
    
    async def create_task(self, task: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Create a FHIR Task resource.

        Args:
            task: FHIR Task resource data

        Returns:
            Created Task resource or None if failed
        """
        if not self.initialized:
            logger.error("Google FHIR service not initialized")
            return None

        if self.mock_mode:
            logger.info("Mock mode: Task creation simulated")
            return {"id": "mock-task-001", "resourceType": "Task", **task}

        try:
            return await self.client.create_resource("Task", task)
        except Exception as e:
            logger.error(f"Error creating Task: {e}")
            return None
    
    async def get_task(self, task_id: str) -> Optional[Dict[str, Any]]:
        """
        Get a FHIR Task resource by ID.
        
        Args:
            task_id: FHIR Task ID
            
        Returns:
            Task resource or None if not found
        """
        if not self.initialized:
            logger.error("Google FHIR service not initialized")
            return None
        
        try:
            return await self.client.get_resource("Task", task_id)
        except Exception as e:
            logger.error(f"Error getting Task {task_id}: {e}")
            return None
    
    async def update_task(self, task_id: str, task: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """
        Update a FHIR Task resource.
        
        Args:
            task_id: FHIR Task ID
            task: Updated Task resource data
            
        Returns:
            Updated Task resource or None if failed
        """
        if not self.initialized:
            logger.error("Google FHIR service not initialized")
            return None
        
        try:
            return await self.client.update_resource("Task", task_id, task)
        except Exception as e:
            logger.error(f"Error updating Task {task_id}: {e}")
            return None
    
    async def search_tasks(self, params: Dict[str, str] = None) -> List[Dict[str, Any]]:
        """
        Search for FHIR Task resources.

        Args:
            params: Search parameters (e.g., {"patient": "Patient/123", "status": "ready"})

        Returns:
            List of Task resources
        """
        if not self.initialized:
            logger.error("Google FHIR service not initialized")
            return []

        try:
            return await self.client.search_resources("Task", params or {})
        except Exception as e:
            logger.error(f"Error searching Tasks: {e}")
            return []

    async def search_resources(self, resource_type: str, params: Dict[str, Any] = None) -> List[Dict[str, Any]]:
        """
        Generic method to search for FHIR resources.

        Args:
            resource_type: FHIR resource type (e.g., 'Task', 'PlanDefinition')
            params: Search parameters

        Returns:
            List of matching resources
        """
        if not self.initialized:
            logger.error("Google FHIR service not initialized")
            return []

        if self.mock_mode:
            logger.info(f"Mock mode: {resource_type} search simulated")
            return []

        try:
            return await self.client.search_resources(resource_type, params or {})
        except Exception as e:
            logger.error(f"Error searching {resource_type} resources: {e}")
            return []


# Global service instance
google_fhir_service = GoogleFHIRService()
