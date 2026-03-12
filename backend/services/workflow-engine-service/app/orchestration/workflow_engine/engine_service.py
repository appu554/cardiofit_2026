"""
Main Workflow Engine Service that orchestrates all workflow components.
"""
import logging
import asyncio
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any
from sqlalchemy.orm import Session

from app.workflow_definition_service import workflow_definition_service
from app.workflow_instance_service import workflow_instance_service
from app.task_service import task_service
from app.camunda_service import camunda_service
from app.camunda_cloud_service import camunda_cloud_service
from app.supabase_service import supabase_service
from app.core.config import settings
from app.models.workflow_models import WorkflowInstance, WorkflowTask
from app.db.database import get_db

logger = logging.getLogger(__name__)


class WorkflowEngineService:
    """
    Main workflow engine service that orchestrates workflow execution.
    """
    
    def __init__(self):
        self.definition_service = workflow_definition_service
        self.instance_service = workflow_instance_service
        self.task_service = task_service
        self.camunda_service = camunda_service
        self.camunda_cloud_service = camunda_cloud_service
        self.supabase_service = supabase_service

        # Phase 4 services - import here to avoid circular imports
        self.service_task_executor = None
        self.event_listener = None
        self.event_publisher = None
        self.fhir_resource_monitor = None

        # Phase 5 advanced services - import here to avoid circular imports
        self.timer_service = None
        self.escalation_service = None
        self.gateway_service = None
        self.error_recovery_service = None

        self.initialized = False
        self.running = False
        self.use_camunda_cloud = settings.USE_CAMUNDA_CLOUD
    
    async def initialize(self) -> bool:
        """
        Initialize the workflow engine.

        Returns:
            True if initialization successful, False otherwise
        """
        try:
            # Initialize appropriate Camunda service
            if self.use_camunda_cloud:
                logger.info("Initializing Camunda Cloud service...")
                camunda_initialized = await self.camunda_cloud_service.initialize()
                if not camunda_initialized:
                    logger.warning("Camunda Cloud service initialization failed, continuing without it")
            else:
                logger.info("Initializing local Camunda service...")
                camunda_initialized = await self.camunda_service.initialize()
                if not camunda_initialized:
                    logger.warning("Local Camunda service initialization failed, continuing without it")

            # Initialize Phase 4 services
            await self._initialize_phase4_services()

            # Initialize Phase 5 advanced services
            await self._initialize_phase5_services()

            self.initialized = True
            logger.info("Workflow engine service initialized successfully")
            return True

        except Exception as e:
            logger.error(f"Failed to initialize workflow engine service: {e}")
            self.initialized = False
            return False

    async def _initialize_phase4_services(self):
        """Initialize Phase 4 service integration components."""
        try:
            # Import Phase 4 services (avoid circular imports)
            from app.service_task_executor import service_task_executor
            from app.event_listener import event_listener
            from app.event_publisher import event_publisher
            from app.fhir_resource_monitor import fhir_resource_monitor

            self.service_task_executor = service_task_executor
            self.event_listener = event_listener
            self.event_publisher = event_publisher
            self.fhir_resource_monitor = fhir_resource_monitor

            logger.info("Phase 4 services initialized successfully")

        except Exception as e:
            logger.error(f"Error initializing Phase 4 services: {e}")
            # Continue without Phase 4 services if they fail to initialize

    async def _initialize_phase5_services(self):
        """Initialize Phase 5 advanced service components."""
        try:
            # Import Phase 5 services (avoid circular imports)
            from app.timer_service import timer_service
            from app.escalation_service import escalation_service
            from app.gateway_service import gateway_service
            from app.error_recovery_service import error_recovery_service

            self.timer_service = timer_service
            self.escalation_service = escalation_service
            self.gateway_service = gateway_service
            self.error_recovery_service = error_recovery_service

            # Initialize each service
            await self.timer_service.initialize()
            await self.escalation_service.initialize()
            await self.gateway_service.initialize()
            await self.error_recovery_service.initialize()

            logger.info("Phase 5 advanced services initialized successfully")

        except Exception as e:
            logger.error(f"Error initializing Phase 5 services: {e}")
            # Continue without Phase 5 services if they fail to initialize
    
    async def start_workflow(
        self,
        definition_id: Optional[int] = None,
        bpmn_process_id: Optional[str] = None,
        patient_id: str = None,
        initial_variables: Optional[Dict[str, Any]] = None,
        context: Optional[Dict[str, Any]] = None,
        created_by: Optional[str] = None
    ) -> Optional[Dict[str, Any]]:
        """
        Start a new workflow instance.
        
        Args:
            definition_id: Workflow definition ID
            patient_id: Patient ID
            initial_variables: Initial process variables
            context: Additional context data
            created_by: User starting the workflow
            
        Returns:
            Workflow instance summary or None if failed
        """
        try:
            db = next(get_db())
            
            # Get workflow definition
            if definition_id:
                workflow_def = await self.definition_service.get_workflow_definition(definition_id, db)
                if not workflow_def:
                    logger.error(f"Workflow definition with ID {definition_id} not found")
                    return None
                bpmn_process_id = workflow_def.name # Assume name is the bpmn_process_id for old workflows
            elif bpmn_process_id:
                workflow_def = await self.definition_service.get_workflow_definition_by_name(bpmn_process_id, db)
                if not workflow_def:
                    logger.error(f"Workflow definition with name {bpmn_process_id} not found")
                    return None
            else:
                logger.error("Either definition_id or bpmn_process_id must be provided.")
                return None
            
            # Create workflow instance
            instance = await self.instance_service.start_workflow_instance(
                definition_id=workflow_def.id, # Use the ID from the fetched definition
                patient_id=patient_id,
                initial_variables=initial_variables,
                context=context,
                created_by=created_by,
                db=db
            )
            
            if not instance:
                logger.error("Failed to create workflow instance")
                return None
            
            # Start Camunda process instance if available
            process_instance_id = None
            if self.use_camunda_cloud:
                if self.camunda_cloud_service and self.camunda_cloud_service.initialized and workflow_def.bpmn_xml:
                    # For Camunda Cloud, we need the bpmn_process_id. We assume it's the definition's name.
                    logger.info(f"Starting Camunda Cloud process '{workflow_def.name}' for instance {instance.id}")
                    process_instance_id = await self.camunda_cloud_service.start_process_instance(
                        bpmn_process_id=bpmn_process_id,
                        version=workflow_def.version,
                        variables={
                            "workflow_instance_id": instance.id,
                            "patient_id": patient_id,
                            **(initial_variables or {})
                        }
                    )
            elif self.camunda_service.initialized and workflow_def.bpmn_xml:
                logger.info(f"Starting local Camunda process for instance {instance.id}")
                process_instance_id = await self.camunda_service.start_process_instance(
                    process_definition_key=f"workflow_{workflow_def.id}",
                    variables={
                        "workflow_instance_id": instance.id,
                        "patient_id": patient_id,
                        **(initial_variables or {})
                    }
                )

            if process_instance_id:
                logger.info(f"Successfully started process in Camunda, process_instance_id: {process_instance_id}")
                await self.instance_service.set_camunda_process_instance_id(
                    instance_id=instance.id,
                    process_instance_id=str(process_instance_id),
                    db=db
                )
            
            # Return workflow instance summary
            return {
                "id": instance.id,
                "external_id": instance.external_id,
                "definition_id": workflow_def.id,
                "definition_name": workflow_def.name,
                "patient_id": patient_id,
                "status": instance.status,
                "start_time": instance.start_time.isoformat(),
                "variables": instance.variables,
                "context": instance.context
            }
            
        except Exception as e:
            logger.error(f"Error starting workflow: {e}")
            return None
    
    async def signal_workflow(
        self,
        instance_id: int,
        signal_name: str,
        variables: Optional[Dict[str, Any]] = None,
        user_id: Optional[str] = None
    ) -> bool:
        """
        Send signal to workflow instance.
        
        Args:
            instance_id: Workflow instance ID
            signal_name: Signal name
            variables: Signal variables
            user_id: User sending the signal
            
        Returns:
            True if signal sent successfully, False otherwise
        """
        try:
            db = next(get_db())
            
            # Get workflow instance
            instance = await self.instance_service.get_workflow_instance(instance_id, db)
            if not instance:
                return False
            
            # Send signal to workflow instance
            success = await self.instance_service.signal_workflow_instance(
                instance_id, signal_name, variables, user_id, db
            )
            
            if not success:
                return False
            
            # Send signal to Camunda if available
            if self.camunda_service.initialized and instance.external_id:
                await self.camunda_service.signal_process_instance(
                    instance.external_id, signal_name, variables
                )
            
            return True
            
        except Exception as e:
            logger.error(f"Error signaling workflow: {e}")
            return False
    
    async def complete_task(
        self,
        task_id: int,
        user_id: str,
        output_variables: Optional[Dict[str, Any]] = None
    ) -> Optional[Dict[str, Any]]:
        """
        Complete a workflow task.
        
        Args:
            task_id: Task ID
            user_id: User completing the task
            output_variables: Task output variables
            
        Returns:
            Updated task summary or None if failed
        """
        try:
            db = next(get_db())
            
            # Complete the task
            task = await self.task_service.complete_task(
                task_id, user_id, output_variables, db
            )
            
            if not task:
                return None
            
            # Complete external task in Camunda if available
            if self.camunda_service.initialized and task.external_task_id:
                await self.camunda_service.complete_external_task(
                    task.external_task_id,
                    "workflow-engine-service",
                    output_variables
                )
            
            # Return task summary
            return {
                "id": task.id,
                "name": task.name,
                "status": task.status,
                "assignee": task.assignee,
                "completed_at": task.completed_at.isoformat() if task.completed_at else None,
                "completed_by": task.completed_by,
                "variables": task.variables
            }
            
        except Exception as e:
            logger.error(f"Error completing task: {e}")
            return None
    
    async def get_user_tasks(
        self,
        user_id: str,
        status: Optional[str] = None,
        patient_id: Optional[str] = None
    ) -> List[Dict[str, Any]]:
        """
        Get tasks assigned to a user.
        
        Args:
            user_id: User ID
            status: Optional status filter
            patient_id: Optional patient filter
            
        Returns:
            List of task summaries
        """
        try:
            db = next(get_db())
            
            # Get user's roles and groups for candidate group filtering
            user_roles = await self.supabase_service.get_user_roles(user_id)
            
            # Get tasks assigned to user
            assigned_tasks = await self.task_service.get_tasks(
                assignee=user_id,
                status=status,
                patient_id=patient_id,
                db=db
            )
            
            # Get tasks for candidate groups
            candidate_tasks = await self.task_service.get_tasks(
                candidate_groups=user_roles,
                status=status,
                patient_id=patient_id,
                db=db
            )
            
            # Combine and deduplicate tasks
            all_tasks = {task.id: task for task in assigned_tasks + candidate_tasks}
            
            # Convert to summaries
            task_summaries = []
            for task in all_tasks.values():
                # Get workflow instance for patient info
                instance = await self.instance_service.get_workflow_instance(
                    task.workflow_instance_id, db
                )
                
                task_summaries.append({
                    "id": task.id,
                    "name": task.name,
                    "description": task.description,
                    "status": task.status,
                    "priority": task.priority,
                    "assignee": task.assignee,
                    "candidate_groups": task.candidate_groups,
                    "due_date": task.due_date.isoformat() if task.due_date else None,
                    "created_at": task.created_at.isoformat(),
                    "patient_id": instance.patient_id if instance else None,
                    "workflow_instance_id": task.workflow_instance_id,
                    "form_key": task.form_key,
                    "variables": task.variables
                })
            
            # Sort by priority and due date
            task_summaries.sort(key=lambda x: (
                -x["priority"],  # Higher priority first
                x["due_date"] or "9999-12-31"  # Earlier due date first
            ))
            
            return task_summaries
            
        except Exception as e:
            logger.error(f"Error getting user tasks: {e}")
            return []
    
    async def get_patient_workflows(
        self,
        patient_id: str,
        status: Optional[str] = None
    ) -> List[Dict[str, Any]]:
        """
        Get workflow instances for a patient.
        
        Args:
            patient_id: Patient ID
            status: Optional status filter
            
        Returns:
            List of workflow instance summaries
        """
        try:
            db = next(get_db())
            
            # Get workflow instances
            instances = await self.instance_service.get_workflow_instances(
                patient_id=patient_id,
                status=status,
                db=db
            )
            
            # Convert to summaries
            summaries = []
            for instance in instances:
                # Get workflow definition
                workflow_def = await self.definition_service.get_workflow_definition(
                    instance.definition_id, db
                )
                
                # Get active tasks count
                active_tasks = await self.task_service.get_tasks(
                    workflow_instance_id=instance.id,
                    status="assigned",
                    db=db
                )
                
                summaries.append({
                    "id": instance.id,
                    "external_id": instance.external_id,
                    "definition_id": instance.definition_id,
                    "definition_name": workflow_def.name if workflow_def else "Unknown",
                    "definition_category": workflow_def.category if workflow_def else None,
                    "patient_id": instance.patient_id,
                    "status": instance.status,
                    "start_time": instance.start_time.isoformat(),
                    "end_time": instance.end_time.isoformat() if instance.end_time else None,
                    "active_tasks_count": len(active_tasks),
                    "variables": instance.variables,
                    "context": instance.context
                })
            
            return summaries
            
        except Exception as e:
            logger.error(f"Error getting patient workflows: {e}")
            return []
    
    async def start_monitoring(self) -> None:
        """
        Start workflow monitoring and maintenance tasks.
        """
        if not self.initialized:
            logger.error("Workflow engine not initialized")
            return

        try:
            self.running = True
            logger.info("Starting workflow engine monitoring...")

            # Start Camunda worker if available
            if self.camunda_service.initialized:
                await self.camunda_service.start_worker()

            # Start monitoring tasks
            asyncio.create_task(self._monitor_overdue_tasks())
            asyncio.create_task(self._monitor_stalled_workflows())

            # Start Phase 4 services
            await self._start_phase4_services()

            # Start Phase 5 advanced services
            await self._start_phase5_services()

        except Exception as e:
            logger.error(f"Error starting workflow monitoring: {e}")

    async def _start_phase4_services(self):
        """Start Phase 4 service integration components."""
        try:
            # Start event listener
            if self.event_listener:
                asyncio.create_task(self.event_listener.start_listening())
                logger.info("Event listener started")

            # Start FHIR resource monitor
            if self.fhir_resource_monitor:
                asyncio.create_task(self.fhir_resource_monitor.start_monitoring())
                logger.info("FHIR resource monitor started")

            logger.info("Phase 4 services started successfully")

        except Exception as e:
            logger.error(f"Error starting Phase 4 services: {e}")

    async def _start_phase5_services(self):
        """Start Phase 5 advanced service components."""
        try:
            # Phase 5 services are mostly event-driven and don't need explicit starting
            # Timer service will load active timers during initialization
            # Escalation service will be triggered by events
            # Gateway service will be triggered by workflow events
            # Error recovery service will be triggered by error events

            logger.info("Phase 5 advanced services started successfully")

        except Exception as e:
            logger.error(f"Error starting Phase 5 services: {e}")
    
    async def stop_monitoring(self) -> None:
        """
        Stop workflow monitoring.
        """
        try:
            self.running = False
            logger.info("Stopping workflow engine monitoring...")
            
            # Stop Camunda worker
            if self.camunda_service.initialized:
                await self.camunda_service.stop_worker()
            
        except Exception as e:
            logger.error(f"Error stopping workflow monitoring: {e}")
    
    async def _monitor_overdue_tasks(self) -> None:
        """
        Monitor and escalate overdue tasks.
        """
        while self.running:
            try:
                db = next(get_db())
                
                # Find overdue tasks
                overdue_tasks = db.query(WorkflowTask).filter(
                    WorkflowTask.due_date < datetime.utcnow(),
                    WorkflowTask.status.in_(["created", "assigned"]),
                    WorkflowTask.escalated == False
                ).all()
                
                for task in overdue_tasks:
                    # Escalate task
                    await self.task_service.escalate_task(
                        task.id,
                        "supervisor",  # Default escalation target
                        "overdue",
                        1,
                        None,  # System escalation
                        db
                    )
                    
                    # Mark as escalated
                    task.escalated = True
                    db.commit()
                
                # Wait before next check
                await asyncio.sleep(300)  # Check every 5 minutes
                
            except Exception as e:
                logger.error(f"Error monitoring overdue tasks: {e}")
                await asyncio.sleep(60)  # Wait before retrying
    
    async def _monitor_stalled_workflows(self) -> None:
        """
        Monitor and handle stalled workflows.
        """
        while self.running:
            try:
                db = next(get_db())
                
                # Find workflows that have been active for too long without progress
                stall_threshold = datetime.utcnow() - timedelta(hours=24)
                
                stalled_instances = db.query(WorkflowInstance).filter(
                    WorkflowInstance.status == "active",
                    WorkflowInstance.start_time < stall_threshold,
                    WorkflowInstance.updated_at < stall_threshold
                ).all()
                
                for instance in stalled_instances:
                    logger.warning(f"Detected stalled workflow instance: {instance.id}")
                    
                    # Could implement automatic recovery or notification here
                    # For now, just log the issue
                
                # Wait before next check
                await asyncio.sleep(3600)  # Check every hour
                
            except Exception as e:
                logger.error(f"Error monitoring stalled workflows: {e}")
                await asyncio.sleep(300)  # Wait before retrying


# Global service instance
workflow_engine_service = WorkflowEngineService()
