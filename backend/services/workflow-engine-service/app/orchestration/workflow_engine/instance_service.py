"""
Workflow Instance Service for managing workflow execution instances.
"""
import logging
import json
import uuid
from datetime import datetime
from typing import Dict, List, Optional, Any
from sqlalchemy.orm import Session
from sqlalchemy import and_, or_

from app.models.workflow_models import WorkflowInstance, WorkflowDefinition, WorkflowEvent
from app.workflow_definition_service import workflow_definition_service
from app.supabase_service import supabase_service
from app.db.database import get_db

logger = logging.getLogger(__name__)


class WorkflowInstanceService:
    """
    Service for managing workflow instances and execution state.
    """
    
    def __init__(self):
        self.definition_service = workflow_definition_service
        self.supabase_service = supabase_service
    
    async def start_workflow_instance(
        self,
        definition_id: int,
        patient_id: str,
        initial_variables: Optional[Dict[str, Any]] = None,
        context: Optional[Dict[str, Any]] = None,
        created_by: Optional[str] = None,
        db: Optional[Session] = None
    ) -> Optional[WorkflowInstance]:
        """
        Start a new workflow instance.
        
        Args:
            definition_id: Workflow definition ID
            patient_id: Patient ID for the workflow
            initial_variables: Initial process variables
            context: Additional context data
            created_by: User ID who started the workflow
            db: Database session
            
        Returns:
            Created WorkflowInstance or None if failed
        """
        if not db:
            db = next(get_db())
        
        try:
            # Get workflow definition
            workflow_def = await self.definition_service.get_workflow_definition(definition_id, db)
            if not workflow_def:
                logger.error(f"Workflow definition {definition_id} not found")
                return None
            
            if workflow_def.status != "active":
                logger.error(f"Workflow definition {definition_id} is not active")
                return None
            
            # Generate external ID for Camunda integration
            external_id = str(uuid.uuid4())
            
            # Create workflow instance
            instance = WorkflowInstance(
                external_id=external_id,
                definition_id=definition_id,
                patient_id=patient_id,
                status="active",
                variables=initial_variables or {},
                context=context or {},
                created_by=created_by
            )
            
            db.add(instance)
            db.commit()
            db.refresh(instance)
            
            # Log workflow start event
            await self._log_workflow_event(
                instance.id,
                "workflow_started",
                {
                    "definition_id": definition_id,
                    "patient_id": patient_id,
                    "external_id": external_id,
                    "variables": initial_variables or {},
                    "context": context or {}
                },
                created_by,
                db
            )
            
            logger.info(f"Started workflow instance: {instance.id} (external: {external_id})")
            return instance
            
        except Exception as e:
            logger.error(f"Error starting workflow instance: {e}")
            db.rollback()
            return None
    
    async def get_workflow_instance(
        self,
        instance_id: int,
        db: Optional[Session] = None
    ) -> Optional[WorkflowInstance]:
        """
        Get workflow instance by ID.
        
        Args:
            instance_id: Workflow instance ID
            db: Database session
            
        Returns:
            WorkflowInstance or None if not found
        """
        if not db:
            db = next(get_db())
        
        try:
            return db.query(WorkflowInstance).filter(
                WorkflowInstance.id == instance_id
            ).first()
        except Exception as e:
            logger.error(f"Error getting workflow instance {instance_id}: {e}")
            return None
    
    async def get_workflow_instances(
        self,
        status: Optional[str] = None,
        patient_id: Optional[str] = None,
        definition_id: Optional[int] = None,
        created_by: Optional[str] = None,
        db: Optional[Session] = None
    ) -> List[WorkflowInstance]:
        """
        Get workflow instances with optional filters.
        
        Args:
            status: Filter by status
            patient_id: Filter by patient ID
            definition_id: Filter by definition ID
            created_by: Filter by creator
            db: Database session
            
        Returns:
            List of WorkflowInstance objects
        """
        if not db:
            db = next(get_db())
        
        try:
            query = db.query(WorkflowInstance)
            
            if status:
                query = query.filter(WorkflowInstance.status == status)
            if patient_id:
                query = query.filter(WorkflowInstance.patient_id == patient_id)
            if definition_id:
                query = query.filter(WorkflowInstance.definition_id == definition_id)
            if created_by:
                query = query.filter(WorkflowInstance.created_by == created_by)
            
            return query.order_by(WorkflowInstance.start_time.desc()).all()
            
        except Exception as e:
            logger.error(f"Error getting workflow instances: {e}")
            return []
    
    async def update_workflow_instance(
        self,
        instance_id: int,
        updates: Dict[str, Any],
        user_id: Optional[str] = None,
        db: Optional[Session] = None
    ) -> Optional[WorkflowInstance]:
        """
        Update workflow instance.
        
        Args:
            instance_id: Workflow instance ID
            updates: Dictionary of fields to update
            user_id: User making the update
            db: Database session
            
        Returns:
            Updated WorkflowInstance or None if failed
        """
        if not db:
            db = next(get_db())
        
        try:
            instance = await self.get_workflow_instance(instance_id, db)
            if not instance:
                return None
            
            old_status = instance.status
            
            # Update fields
            for key, value in updates.items():
                if hasattr(instance, key):
                    setattr(instance, key, value)
            
            # Set end time if workflow is completed or terminated
            if instance.status in ["completed", "terminated", "cancelled"] and not instance.end_time:
                instance.end_time = datetime.utcnow()
            
            db.commit()
            db.refresh(instance)
            
            # Log status change event
            if old_status != instance.status:
                await self._log_workflow_event(
                    instance.id,
                    "workflow_status_changed",
                    {
                        "old_status": old_status,
                        "new_status": instance.status,
                        "updates": updates
                    },
                    user_id,
                    db
                )
            
            logger.info(f"Updated workflow instance: {instance_id}")
            return instance
            
        except Exception as e:
            logger.error(f"Error updating workflow instance {instance_id}: {e}")
            db.rollback()
            return None
    
    async def signal_workflow_instance(
        self,
        instance_id: int,
        signal_name: str,
        variables: Optional[Dict[str, Any]] = None,
        user_id: Optional[str] = None,
        db: Optional[Session] = None
    ) -> bool:
        """
        Send signal to workflow instance.
        
        Args:
            instance_id: Workflow instance ID
            signal_name: Signal name
            variables: Signal variables
            user_id: User sending the signal
            db: Database session
            
        Returns:
            True if signal sent successfully, False otherwise
        """
        if not db:
            db = next(get_db())
        
        try:
            instance = await self.get_workflow_instance(instance_id, db)
            if not instance:
                return False
            
            if instance.status not in ["active", "suspended"]:
                logger.warning(f"Cannot signal workflow instance {instance_id} with status {instance.status}")
                return False
            
            # Update instance variables if provided
            if variables:
                current_variables = instance.variables or {}
                current_variables.update(variables)
                instance.variables = current_variables
                db.commit()
            
            # Log signal event
            await self._log_workflow_event(
                instance.id,
                "workflow_signal_received",
                {
                    "signal_name": signal_name,
                    "variables": variables or {},
                    "external_id": instance.external_id
                },
                user_id,
                db
            )
            
            logger.info(f"Sent signal '{signal_name}' to workflow instance: {instance_id}")
            return True
            
        except Exception as e:
            logger.error(f"Error signaling workflow instance {instance_id}: {e}")
            return False
    
    async def cancel_workflow_instance(
        self,
        instance_id: int,
        reason: Optional[str] = None,
        user_id: Optional[str] = None,
        db: Optional[Session] = None
    ) -> bool:
        """
        Cancel workflow instance.
        
        Args:
            instance_id: Workflow instance ID
            reason: Cancellation reason
            user_id: User cancelling the workflow
            db: Database session
            
        Returns:
            True if cancelled successfully, False otherwise
        """
        return await self.update_workflow_instance(
            instance_id,
            {
                "status": "cancelled",
                "context": {
                    **(await self.get_workflow_instance(instance_id, db)).context,
                    "cancellation_reason": reason or "Manual cancellation",
                    "cancelled_by": user_id,
                    "cancelled_at": datetime.utcnow().isoformat()
                }
            },
            user_id,
            db
        ) is not None
    
    async def suspend_workflow_instance(
        self,
        instance_id: int,
        reason: Optional[str] = None,
        user_id: Optional[str] = None,
        db: Optional[Session] = None
    ) -> bool:
        """
        Suspend workflow instance.
        
        Args:
            instance_id: Workflow instance ID
            reason: Suspension reason
            user_id: User suspending the workflow
            db: Database session
            
        Returns:
            True if suspended successfully, False otherwise
        """
        return await self.update_workflow_instance(
            instance_id,
            {
                "status": "suspended",
                "context": {
                    **(await self.get_workflow_instance(instance_id, db)).context,
                    "suspension_reason": reason or "Manual suspension",
                    "suspended_by": user_id,
                    "suspended_at": datetime.utcnow().isoformat()
                }
            },
            user_id,
            db
        ) is not None
    
    async def resume_workflow_instance(
        self,
        instance_id: int,
        user_id: Optional[str] = None,
        db: Optional[Session] = None
    ) -> bool:
        """
        Resume suspended workflow instance.
        
        Args:
            instance_id: Workflow instance ID
            user_id: User resuming the workflow
            db: Database session
            
        Returns:
            True if resumed successfully, False otherwise
        """
        instance = await self.get_workflow_instance(instance_id, db)
        if not instance or instance.status != "suspended":
            return False
        
        return await self.update_workflow_instance(
            instance_id,
            {
                "status": "active",
                "context": {
                    **instance.context,
                    "resumed_by": user_id,
                    "resumed_at": datetime.utcnow().isoformat()
                }
            },
            user_id,
            db
        ) is not None
    
    async def _log_workflow_event(
        self,
        workflow_instance_id: int,
        event_type: str,
        event_data: Dict[str, Any],
        user_id: Optional[str] = None,
        db: Optional[Session] = None
    ) -> None:
        """
        Log workflow event to database and Supabase.
        
        Args:
            workflow_instance_id: Workflow instance ID
            event_type: Type of event
            event_data: Event data
            user_id: User associated with event
            db: Database session
        """
        if not db:
            db = next(get_db())
        
        try:
            # Log to database
            event = WorkflowEvent(
                workflow_instance_id=workflow_instance_id,
                event_type=event_type,
                event_data=event_data,
                user_id=user_id,
                source="workflow-engine"
            )
            
            db.add(event)
            db.commit()
            
            # Log event to Supabase for real-time updates
            if self.supabase_service and self.supabase_service.initialized:
                await self.supabase_service.log_workflow_event(
                    instance_id=workflow_instance_id,
                    task_id=None, # This is a workflow-level event
                    event_type=event_type,
                    event_data=event_data,
                    user_id=user_id,
                    source="workflow-engine"
                )
            else:
                logger.info("Supabase service not initialized, skipping event logging.")
            
        except Exception as e:
            logger.error(f"Error logging workflow event: {e}")


# Global service instance
workflow_instance_service = WorkflowInstanceService()
