"""
Event Publisher for Workflow Engine Service.

This module publishes workflow-related events to other services in the federation.
"""

import asyncio
import json
import logging
from typing import Dict, Any, Optional, List
from datetime import datetime
import httpx
from app.core.config import settings

logger = logging.getLogger(__name__)


class EventPublisher:
    """
    Publishes workflow events to other services and the event store.
    """
    
    def __init__(self):
        self.event_store_enabled = True
        self.webhook_endpoints = {}
        self.retry_attempts = 3
        self.retry_delay = 1.0
        self.timeout = 10.0

        # Configure integration mode from environment or settings
        from app.core.config import settings
        self.mock_mode = getattr(settings, 'WORKFLOW_MOCK_MODE', False)  # Default to full integration

        # Initialize webhook endpoints for other services
        self._initialize_webhook_endpoints()
    
    def _initialize_webhook_endpoints(self):
        """Initialize webhook endpoints for other services."""
        self.webhook_endpoints = {
            "patient-service": f"http://localhost:8003/api/webhooks/workflow-events",
            "encounter-service": f"http://localhost:8020/api/webhooks/workflow-events",
            "order-service": f"http://localhost:8013/api/webhooks/workflow-events",
            "scheduling-service": f"http://localhost:8014/api/webhooks/workflow-events",
            "organization-service": f"http://localhost:8012/api/webhooks/workflow-events",
            "medication-service": f"http://localhost:8009/api/webhooks/workflow-events"
        }
    
    async def publish_workflow_started(
        self,
        workflow_instance_id: str,
        workflow_definition_id: str,
        patient_id: str,
        variables: Dict[str, Any],
        user_id: Optional[str] = None
    ):
        """Publish workflow started event."""
        event_data = {
            "workflow_instance_id": workflow_instance_id,
            "workflow_definition_id": workflow_definition_id,
            "patient_id": patient_id,
            "variables": variables,
            "user_id": user_id,
            "started_at": datetime.utcnow().isoformat()
        }
        
        await self._publish_event("workflow.started", event_data)
    
    async def publish_workflow_completed(
        self,
        workflow_instance_id: str,
        workflow_definition_id: str,
        patient_id: str,
        final_variables: Dict[str, Any],
        user_id: Optional[str] = None
    ):
        """Publish workflow completed event."""
        event_data = {
            "workflow_instance_id": workflow_instance_id,
            "workflow_definition_id": workflow_definition_id,
            "patient_id": patient_id,
            "final_variables": final_variables,
            "user_id": user_id,
            "completed_at": datetime.utcnow().isoformat()
        }
        
        await self._publish_event("workflow.completed", event_data)
    
    async def publish_workflow_failed(
        self,
        workflow_instance_id: str,
        workflow_definition_id: str,
        patient_id: str,
        error_message: str,
        user_id: Optional[str] = None
    ):
        """Publish workflow failed event."""
        event_data = {
            "workflow_instance_id": workflow_instance_id,
            "workflow_definition_id": workflow_definition_id,
            "patient_id": patient_id,
            "error_message": error_message,
            "user_id": user_id,
            "failed_at": datetime.utcnow().isoformat()
        }
        
        await self._publish_event("workflow.failed", event_data)
    
    async def publish_task_created(
        self,
        task_id: str,
        workflow_instance_id: str,
        patient_id: str,
        assignee_id: Optional[str],
        task_data: Dict[str, Any]
    ):
        """Publish task created event."""
        event_data = {
            "task_id": task_id,
            "workflow_instance_id": workflow_instance_id,
            "patient_id": patient_id,
            "assignee_id": assignee_id,
            "task_data": task_data,
            "created_at": datetime.utcnow().isoformat()
        }
        
        await self._publish_event("workflow.task.created", event_data)
    
    async def publish_task_completed(
        self,
        task_id: str,
        workflow_instance_id: str,
        patient_id: str,
        completed_by: str,
        output_variables: Dict[str, Any]
    ):
        """Publish task completed event."""
        event_data = {
            "task_id": task_id,
            "workflow_instance_id": workflow_instance_id,
            "patient_id": patient_id,
            "completed_by": completed_by,
            "output_variables": output_variables,
            "completed_at": datetime.utcnow().isoformat()
        }
        
        await self._publish_event("workflow.task.completed", event_data)
    
    async def publish_task_assigned(
        self,
        task_id: str,
        workflow_instance_id: str,
        patient_id: str,
        assignee_id: str,
        assigned_by: Optional[str] = None
    ):
        """Publish task assigned event."""
        event_data = {
            "task_id": task_id,
            "workflow_instance_id": workflow_instance_id,
            "patient_id": patient_id,
            "assignee_id": assignee_id,
            "assigned_by": assigned_by,
            "assigned_at": datetime.utcnow().isoformat()
        }
        
        await self._publish_event("workflow.task.assigned", event_data)
    
    async def publish_fhir_resource_created(
        self,
        resource_type: str,
        resource_id: str,
        workflow_instance_id: str,
        patient_id: str,
        resource_data: Dict[str, Any]
    ):
        """Publish FHIR resource created by workflow event."""
        event_data = {
            "resource_type": resource_type,
            "resource_id": resource_id,
            "workflow_instance_id": workflow_instance_id,
            "patient_id": patient_id,
            "resource_data": resource_data,
            "created_at": datetime.utcnow().isoformat()
        }
        
        await self._publish_event("workflow.fhir.resource.created", event_data)
    
    async def publish_service_task_executed(
        self,
        service_name: str,
        operation: str,
        workflow_instance_id: str,
        patient_id: str,
        result: Dict[str, Any]
    ):
        """Publish service task executed event."""
        event_data = {
            "service_name": service_name,
            "operation": operation,
            "workflow_instance_id": workflow_instance_id,
            "patient_id": patient_id,
            "result": result,
            "executed_at": datetime.utcnow().isoformat()
        }
        
        await self._publish_event("workflow.service.task.executed", event_data)
    
    async def _publish_event(self, event_type: str, event_data: Dict[str, Any]):
        """Publish an event to all configured destinations."""
        try:
            # Create event record
            event = {
                "event_type": event_type,
                "event_data": event_data,
                "source": "workflow-engine-service",
                "created_at": datetime.utcnow().isoformat()
            }
            
            # Store event in event store
            if self.event_store_enabled:
                await self._store_event(event)
            
            # Send to webhook endpoints
            await self._send_to_webhooks(event)
            
            logger.info(f"Published event: {event_type}")
            
        except Exception as e:
            logger.error(f"Error publishing event {event_type}: {e}")
    
    async def _store_event(self, event: Dict[str, Any]):
        """Store event in the event store (Supabase)."""
        try:
            # Import here to avoid circular imports
            from app.supabase_service import supabase_service
            await supabase_service.store_event(event)
            
        except Exception as e:
            logger.error(f"Error storing event in event store: {e}")
    
    async def _send_to_webhooks(self, event: Dict[str, Any]):
        """Send event to webhook endpoints."""
        # Send to all webhook endpoints concurrently
        tasks = []
        
        for service_name, endpoint in self.webhook_endpoints.items():
            task = asyncio.create_task(
                self._send_webhook(service_name, endpoint, event)
            )
            tasks.append(task)
        
        # Wait for all webhook calls to complete
        if tasks:
            await asyncio.gather(*tasks, return_exceptions=True)
    
    async def _send_webhook(self, service_name: str, endpoint: str, event: Dict[str, Any]):
        """Send event to a specific webhook endpoint."""
        try:
            if self.mock_mode:
                logger.info(f"MOCK: Would send webhook to {service_name} at {endpoint}")
                logger.debug(f"MOCK: Event data: {event.get('event_type', 'unknown')}")
                return

            await self._send_webhook_with_retry(service_name, endpoint, event)

        except Exception as e:
            logger.error(f"Failed to send webhook to {service_name}: {e}")
    
    async def _send_webhook_with_retry(
        self,
        service_name: str,
        endpoint: str,
        event: Dict[str, Any]
    ):
        """Send webhook with retry logic."""
        last_exception = None
        
        for attempt in range(self.retry_attempts):
            try:
                if attempt > 0:
                    await asyncio.sleep(self.retry_delay * attempt)
                
                async with httpx.AsyncClient(timeout=self.timeout) as client:
                    response = await client.post(
                        endpoint,
                        json=event,
                        headers={"Content-Type": "application/json"}
                    )
                    
                    if response.status_code in [200, 201, 202]:
                        logger.debug(f"Webhook sent successfully to {service_name}")
                        return
                    else:
                        raise httpx.HTTPStatusError(
                            f"HTTP {response.status_code}",
                            request=response.request,
                            response=response
                        )
                
            except Exception as e:
                last_exception = e
                logger.warning(f"Webhook attempt {attempt + 1} to {service_name} failed: {str(e)}")
                
                if attempt == self.retry_attempts - 1:
                    raise last_exception
        
        raise last_exception
    
    async def publish_custom_event(
        self,
        event_type: str,
        event_data: Dict[str, Any],
        target_services: Optional[List[str]] = None
    ):
        """Publish a custom event, optionally to specific services only."""
        try:
            # Create event record
            event = {
                "event_type": event_type,
                "event_data": event_data,
                "source": "workflow-engine-service",
                "created_at": datetime.utcnow().isoformat()
            }
            
            # Store event in event store
            if self.event_store_enabled:
                await self._store_event(event)
            
            # Send to specific services or all services
            if target_services:
                tasks = []
                for service_name in target_services:
                    endpoint = self.webhook_endpoints.get(service_name)
                    if endpoint:
                        task = asyncio.create_task(
                            self._send_webhook(service_name, endpoint, event)
                        )
                        tasks.append(task)
                
                if tasks:
                    await asyncio.gather(*tasks, return_exceptions=True)
            else:
                await self._send_to_webhooks(event)
            
            logger.info(f"Published custom event: {event_type}")
            
        except Exception as e:
            logger.error(f"Error publishing custom event {event_type}: {e}")


# Global service instance
event_publisher = EventPublisher()
