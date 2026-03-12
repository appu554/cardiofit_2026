"""
FHIR Resource Monitor for Workflow Engine Service.

This module monitors FHIR resources for changes and triggers workflow actions
based on resource updates, particularly Task resource status changes.
"""

import asyncio
import json
import logging
from typing import Dict, Any, Optional, List, Set
from datetime import datetime, timedelta
from app.core.config import settings

logger = logging.getLogger(__name__)


class FHIRResourceMonitor:
    """
    Monitors FHIR resources for changes and triggers workflow actions.
    """
    
    def __init__(self):
        self.running = False
        self.polling_interval = 30.0  # seconds
        self.monitored_resources = {
            "Task": self._monitor_task_resources,
            "Patient": self._monitor_patient_resources,
            "Encounter": self._monitor_encounter_resources,
            "ServiceRequest": self._monitor_service_request_resources,
            "Appointment": self._monitor_appointment_resources
        }
        
        # Track last modification times for resources
        self.last_modified_times: Dict[str, Dict[str, datetime]] = {}
        
        # Initialize last modified times
        for resource_type in self.monitored_resources.keys():
            self.last_modified_times[resource_type] = {}
    
    async def start_monitoring(self):
        """Start the FHIR resource monitoring loop."""
        if self.running:
            logger.warning("FHIR resource monitor is already running")
            return
        
        self.running = True
        logger.info("Starting FHIR resource monitor...")
        
        # Start monitoring tasks for each resource type
        tasks = []
        for resource_type, monitor_func in self.monitored_resources.items():
            task = asyncio.create_task(self._monitor_resource_type(resource_type, monitor_func))
            tasks.append(task)
        
        # Wait for all monitoring tasks
        await asyncio.gather(*tasks, return_exceptions=True)
    
    async def stop_monitoring(self):
        """Stop the FHIR resource monitoring loop."""
        self.running = False
        logger.info("Stopping FHIR resource monitor...")
    
    async def _monitor_resource_type(self, resource_type: str, monitor_func):
        """Monitor a specific resource type."""
        logger.info(f"Starting monitor for {resource_type} resources")
        
        while self.running:
            try:
                await monitor_func()
                await asyncio.sleep(self.polling_interval)
                
            except Exception as e:
                logger.error(f"Error monitoring {resource_type} resources: {e}")
                await asyncio.sleep(self.polling_interval * 2)  # Back off on error
    
    async def _monitor_task_resources(self):
        """Monitor Task resources for status changes."""
        try:
            # Import here to avoid circular imports
            from app.google_fhir_service import google_fhir_service

            # Get all Task resources modified since last check
            since_time = datetime.utcnow() - timedelta(minutes=5)

            tasks = await google_fhir_service.search_resources(
                "Task",
                {
                    "_lastUpdated": f"gt{since_time.isoformat()}",
                    "_count": "100"
                }
            )
            
            if not tasks:
                return
            
            logger.info(f"Found {len(tasks)} modified Task resources")
            
            for task in tasks:
                await self._process_task_change(task)
                
        except Exception as e:
            logger.error(f"Error monitoring Task resources: {e}")
    
    async def _process_task_change(self, task: Dict[str, Any]):
        """Process a Task resource change."""
        try:
            task_id = task.get("id")
            if not task_id:
                return
            
            # Check if this is a workflow-managed task
            workflow_instance_id = self._extract_workflow_instance_id(task)
            if not workflow_instance_id:
                return  # Not a workflow task
            
            current_status = task.get("status")
            last_modified = task.get("meta", {}).get("lastUpdated")
            
            # Check if this task has been processed before
            last_known_modified = self.last_modified_times["Task"].get(task_id)
            
            if last_known_modified and last_modified:
                last_modified_dt = datetime.fromisoformat(last_modified.replace('Z', '+00:00'))
                if last_modified_dt <= last_known_modified:
                    return  # No change since last check
            
            # Update last modified time
            if last_modified:
                self.last_modified_times["Task"][task_id] = datetime.fromisoformat(
                    last_modified.replace('Z', '+00:00')
                )
            
            logger.info(f"Processing Task change: {task_id} - Status: {current_status}")
            
            # Handle different task status changes
            if current_status == "completed":
                await self._handle_task_completed(task, workflow_instance_id)
            elif current_status == "cancelled":
                await self._handle_task_cancelled(task, workflow_instance_id)
            elif current_status == "failed":
                await self._handle_task_failed(task, workflow_instance_id)
            elif current_status == "in-progress":
                await self._handle_task_started(task, workflow_instance_id)
            
            # Import here to avoid circular imports
            from app.event_publisher import event_publisher

            # Publish task change event
            await event_publisher.publish_custom_event(
                f"fhir.task.{current_status}",
                {
                    "task_id": task_id,
                    "workflow_instance_id": workflow_instance_id,
                    "status": current_status,
                    "task_data": task
                }
            )
            
        except Exception as e:
            logger.error(f"Error processing Task change: {e}")
    
    def _extract_workflow_instance_id(self, task: Dict[str, Any]) -> Optional[str]:
        """Extract workflow instance ID from Task resource."""
        # Check for workflow instance ID in task extensions or identifiers
        
        # Check extensions
        extensions = task.get("extension", [])
        for ext in extensions:
            if ext.get("url") == "http://clinical-synthesis-hub.com/workflow-instance-id":
                return ext.get("valueString")
        
        # Check identifiers
        identifiers = task.get("identifier", [])
        for identifier in identifiers:
            if identifier.get("system") == "http://clinical-synthesis-hub.com/workflow-instance":
                return identifier.get("value")
        
        # Check requester reference (might contain workflow instance ID)
        requester = task.get("requester", {})
        if requester.get("reference", "").startswith("WorkflowInstance/"):
            return requester["reference"].replace("WorkflowInstance/", "")
        
        return None
    
    async def _handle_task_completed(self, task: Dict[str, Any], workflow_instance_id: str):
        """Handle task completion."""
        task_id = task.get("id")
        
        # Extract output variables from task
        output_variables = self._extract_task_output(task)
        
        # Import here to avoid circular imports
        from app.workflow_engine_service import workflow_engine_service

        # Signal the workflow instance
        await workflow_engine_service.signal_workflow(
            instance_id=workflow_instance_id,
            signal_name="task_completed",
            variables=[
                {"key": "task_id", "value": task_id},
                {"key": "task_status", "value": "completed"},
                {"key": "task_output", "value": json.dumps(output_variables)}
            ],
            user_id="system"
        )
        
        logger.info(f"Signaled workflow {workflow_instance_id} for task completion: {task_id}")
    
    async def _handle_task_cancelled(self, task: Dict[str, Any], workflow_instance_id: str):
        """Handle task cancellation."""
        task_id = task.get("id")

        # Import here to avoid circular imports
        from app.workflow_engine_service import workflow_engine_service

        # Signal the workflow instance
        await workflow_engine_service.signal_workflow(
            instance_id=workflow_instance_id,
            signal_name="task_cancelled",
            variables=[
                {"key": "task_id", "value": task_id},
                {"key": "task_status", "value": "cancelled"}
            ],
            user_id="system"
        )
        
        logger.info(f"Signaled workflow {workflow_instance_id} for task cancellation: {task_id}")
    
    async def _handle_task_failed(self, task: Dict[str, Any], workflow_instance_id: str):
        """Handle task failure."""
        task_id = task.get("id")

        # Extract failure reason
        status_reason = task.get("statusReason", {}).get("text", "Unknown error")

        # Import here to avoid circular imports
        from app.workflow_engine_service import workflow_engine_service

        # Signal the workflow instance
        await workflow_engine_service.signal_workflow(
            instance_id=workflow_instance_id,
            signal_name="task_failed",
            variables=[
                {"key": "task_id", "value": task_id},
                {"key": "task_status", "value": "failed"},
                {"key": "failure_reason", "value": status_reason}
            ],
            user_id="system"
        )
        
        logger.info(f"Signaled workflow {workflow_instance_id} for task failure: {task_id}")
    
    async def _handle_task_started(self, task: Dict[str, Any], workflow_instance_id: str):
        """Handle task start."""
        task_id = task.get("id")

        # Import here to avoid circular imports
        from app.workflow_engine_service import workflow_engine_service

        # Signal the workflow instance
        await workflow_engine_service.signal_workflow(
            instance_id=workflow_instance_id,
            signal_name="task_started",
            variables=[
                {"key": "task_id", "value": task_id},
                {"key": "task_status", "value": "in-progress"}
            ],
            user_id="system"
        )
        
        logger.info(f"Signaled workflow {workflow_instance_id} for task start: {task_id}")
    
    def _extract_task_output(self, task: Dict[str, Any]) -> Dict[str, Any]:
        """Extract output variables from completed task."""
        output = {}
        
        # Check task output extension
        extensions = task.get("extension", [])
        for ext in extensions:
            if ext.get("url") == "http://clinical-synthesis-hub.com/task-output":
                try:
                    output = json.loads(ext.get("valueString", "{}"))
                except:
                    pass
        
        # Check task note for output data
        notes = task.get("note", [])
        for note in notes:
            text = note.get("text", "")
            if text.startswith("OUTPUT:"):
                try:
                    output_json = text.replace("OUTPUT:", "").strip()
                    output.update(json.loads(output_json))
                except:
                    pass
        
        return output
    
    # Placeholder monitor functions for other resource types
    async def _monitor_patient_resources(self):
        """Monitor Patient resources for changes."""
        # Implementation for patient resource monitoring
        pass
    
    async def _monitor_encounter_resources(self):
        """Monitor Encounter resources for changes."""
        # Implementation for encounter resource monitoring
        pass
    
    async def _monitor_service_request_resources(self):
        """Monitor ServiceRequest resources for changes."""
        # Implementation for service request resource monitoring
        pass
    
    async def _monitor_appointment_resources(self):
        """Monitor Appointment resources for changes."""
        # Implementation for appointment resource monitoring
        pass
    
    async def add_resource_to_monitor(self, resource_type: str, resource_id: str):
        """Add a specific resource to monitoring."""
        if resource_type not in self.last_modified_times:
            self.last_modified_times[resource_type] = {}
        
        # Initialize with current time
        self.last_modified_times[resource_type][resource_id] = datetime.utcnow()
        
        logger.info(f"Added {resource_type}/{resource_id} to monitoring")
    
    async def remove_resource_from_monitor(self, resource_type: str, resource_id: str):
        """Remove a specific resource from monitoring."""
        if resource_type in self.last_modified_times:
            self.last_modified_times[resource_type].pop(resource_id, None)
        
        logger.info(f"Removed {resource_type}/{resource_id} from monitoring")


# Global service instance
fhir_resource_monitor = FHIRResourceMonitor()
