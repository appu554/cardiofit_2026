"""
Reusable Webhook Template for Microservices.

This template can be copied and customized for each microservice to handle
workflow events from the Workflow Engine Service.

Usage:
1. Copy this file to your service's endpoints directory
2. Rename to webhooks.py
3. Update the service name and implement service-specific logic
4. Add to your service's API router
"""

from fastapi import APIRouter, Depends, HTTPException, Body
from typing import Dict, List, Any, Optional
# Update this import to match your service's auth module
# from app.core.auth import get_token_payload
import logging

logger = logging.getLogger(__name__)

router = APIRouter()


@router.post("/workflow-events")
async def handle_workflow_event(
    event: Dict[str, Any] = Body(...),
    # token_payload: Dict[str, Any] = Depends(get_token_payload)  # Uncomment and update
):
    """
    Handle workflow events from the Workflow Engine Service.
    
    Args:
        event: The workflow event data
        token_payload: Authentication token payload
        
    Returns:
        Success response
    """
    try:
        event_type = event.get("event_type")
        event_data = event.get("event_data", {})
        source = event.get("source", "unknown")
        
        # Update service name here
        service_name = "YOUR_SERVICE_NAME"
        logger.info(f"[{service_name}] Received workflow event: {event_type} from {source}")
        
        # Handle different event types
        if event_type == "workflow.started":
            await _handle_workflow_started(event_data)
        elif event_type == "workflow.completed":
            await _handle_workflow_completed(event_data)
        elif event_type == "workflow.failed":
            await _handle_workflow_failed(event_data)
        elif event_type == "workflow.task.created":
            await _handle_task_created(event_data)
        elif event_type == "workflow.task.completed":
            await _handle_task_completed(event_data)
        elif event_type == "workflow.task.assigned":
            await _handle_task_assigned(event_data)
        elif event_type == "fhir.task.completed":
            await _handle_fhir_task_completed(event_data)
        elif event_type == "fhir.task.cancelled":
            await _handle_fhir_task_cancelled(event_data)
        elif event_type == "fhir.task.failed":
            await _handle_fhir_task_failed(event_data)
        elif event_type.startswith("workflow.service.task.executed"):
            await _handle_service_task_executed(event_data)
        elif event_type.startswith("workflow.fhir.resource.created"):
            await _handle_fhir_resource_created(event_data)
        else:
            logger.info(f"[{service_name}] Unhandled event type: {event_type}")
        
        return {
            "status": "success",
            "message": f"Event {event_type} processed successfully",
            "event_id": event.get("id"),
            "processed_by": service_name
        }
        
    except Exception as e:
        logger.error(f"Error processing workflow event: {e}")
        raise HTTPException(
            status_code=500,
            detail=f"Failed to process workflow event: {str(e)}"
        )


# Event Handlers - Customize these for your service
async def _handle_workflow_started(event_data: Dict[str, Any]):
    """Handle workflow started event."""
    workflow_instance_id = event_data.get("workflow_instance_id")
    patient_id = event_data.get("patient_id")
    
    logger.info(f"Workflow {workflow_instance_id} started for patient {patient_id}")
    
    # TODO: Implement service-specific workflow start logic
    pass


async def _handle_workflow_completed(event_data: Dict[str, Any]):
    """Handle workflow completed event."""
    workflow_instance_id = event_data.get("workflow_instance_id")
    patient_id = event_data.get("patient_id")
    final_variables = event_data.get("final_variables", {})
    
    logger.info(f"Workflow {workflow_instance_id} completed for patient {patient_id}")
    
    # TODO: Implement service-specific workflow completion logic
    pass


async def _handle_workflow_failed(event_data: Dict[str, Any]):
    """Handle workflow failed event."""
    workflow_instance_id = event_data.get("workflow_instance_id")
    patient_id = event_data.get("patient_id")
    error_message = event_data.get("error_message")
    
    logger.warning(f"Workflow {workflow_instance_id} failed for patient {patient_id}: {error_message}")
    
    # TODO: Implement service-specific workflow failure logic
    pass


async def _handle_task_created(event_data: Dict[str, Any]):
    """Handle task created event."""
    task_id = event_data.get("task_id")
    patient_id = event_data.get("patient_id")
    assignee_id = event_data.get("assignee_id")
    
    logger.info(f"Task {task_id} created for patient {patient_id}, assigned to {assignee_id}")
    
    # TODO: Implement service-specific task creation logic
    pass


async def _handle_task_completed(event_data: Dict[str, Any]):
    """Handle task completed event."""
    task_id = event_data.get("task_id")
    patient_id = event_data.get("patient_id")
    completed_by = event_data.get("completed_by")
    output_variables = event_data.get("output_variables", {})
    
    logger.info(f"Task {task_id} completed for patient {patient_id} by {completed_by}")
    
    # TODO: Implement service-specific task completion logic
    pass


async def _handle_task_assigned(event_data: Dict[str, Any]):
    """Handle task assigned event."""
    task_id = event_data.get("task_id")
    patient_id = event_data.get("patient_id")
    assignee_id = event_data.get("assignee_id")
    assigned_by = event_data.get("assigned_by")
    
    logger.info(f"Task {task_id} assigned to {assignee_id} for patient {patient_id}")
    
    # TODO: Implement service-specific task assignment logic
    pass


async def _handle_fhir_task_completed(event_data: Dict[str, Any]):
    """Handle FHIR task completed event."""
    task_id = event_data.get("task_id")
    workflow_instance_id = event_data.get("workflow_instance_id")
    status = event_data.get("status")
    
    logger.info(f"FHIR Task {task_id} completed with status {status}")
    
    # TODO: Implement FHIR task completion logic
    pass


async def _handle_fhir_task_cancelled(event_data: Dict[str, Any]):
    """Handle FHIR task cancelled event."""
    task_id = event_data.get("task_id")
    workflow_instance_id = event_data.get("workflow_instance_id")
    
    logger.info(f"FHIR Task {task_id} cancelled")
    
    # TODO: Implement FHIR task cancellation logic
    pass


async def _handle_fhir_task_failed(event_data: Dict[str, Any]):
    """Handle FHIR task failed event."""
    task_id = event_data.get("task_id")
    workflow_instance_id = event_data.get("workflow_instance_id")
    
    logger.warning(f"FHIR Task {task_id} failed")
    
    # TODO: Implement FHIR task failure logic
    pass


async def _handle_service_task_executed(event_data: Dict[str, Any]):
    """Handle service task executed event."""
    service_name = event_data.get("service_name")
    operation = event_data.get("operation")
    result = event_data.get("result", {})
    
    logger.info(f"Service task executed: {service_name}.{operation}")
    
    # TODO: Implement service task execution response logic
    pass


async def _handle_fhir_resource_created(event_data: Dict[str, Any]):
    """Handle FHIR resource created event."""
    resource_type = event_data.get("resource_type")
    resource_id = event_data.get("resource_id")
    workflow_instance_id = event_data.get("workflow_instance_id")
    
    logger.info(f"FHIR {resource_type} resource {resource_id} created by workflow {workflow_instance_id}")
    
    # TODO: Implement FHIR resource creation response logic
    pass


@router.get("/workflow-events/health")
async def webhook_health_check():
    """Health check for webhook endpoints."""
    return {
        "status": "healthy",
        "service": "YOUR_SERVICE_NAME",  # Update this
        "webhook_version": "1.0.0",
        "supported_events": [
            "workflow.started",
            "workflow.completed", 
            "workflow.failed",
            "workflow.task.created",
            "workflow.task.completed",
            "workflow.task.assigned",
            "fhir.task.completed",
            "fhir.task.cancelled",
            "fhir.task.failed",
            "workflow.service.task.executed",
            "workflow.fhir.resource.created"
        ]
    }
