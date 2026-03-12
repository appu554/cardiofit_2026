"""
Task Service for managing workflow tasks and FHIR Task resources.
"""
import logging
import json
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any
from sqlalchemy.orm import Session
from sqlalchemy import and_, or_

from app.models.workflow_models import WorkflowTask, WorkflowInstance, WorkflowEvent
from app.models.task_models import TaskAssignment, TaskComment, TaskEscalation
from app.google_fhir_service import google_fhir_service
from app.supabase_service import supabase_service
from app.db.database import get_db

logger = logging.getLogger(__name__)


class TaskService:
    """
    Service for managing workflow tasks with FHIR Task integration.
    """
    
    def __init__(self):
        self.fhir_service = google_fhir_service
        self.supabase_service = supabase_service
    
    async def create_task(
        self,
        workflow_instance_id: int,
        task_definition_key: str,
        name: str,
        description: Optional[str] = None,
        assignee: Optional[str] = None,
        candidate_groups: Optional[List[str]] = None,
        due_date: Optional[datetime] = None,
        priority: int = 50,
        form_key: Optional[str] = None,
        variables: Optional[Dict[str, Any]] = None,
        db: Optional[Session] = None
    ) -> Optional[WorkflowTask]:
        """
        Create a new workflow task with FHIR Task resource.
        
        Args:
            workflow_instance_id: Workflow instance ID
            task_definition_key: Task definition key from BPMN
            name: Task name
            description: Task description
            assignee: Assigned user ID
            candidate_groups: List of candidate group IDs
            due_date: Task due date
            priority: Task priority (0-100)
            form_key: Form key for task UI
            variables: Task variables
            db: Database session
            
        Returns:
            Created WorkflowTask or None if failed
        """
        if not db:
            db = next(get_db())
        
        try:
            # Get workflow instance
            workflow_instance = db.query(WorkflowInstance).filter(
                WorkflowInstance.id == workflow_instance_id
            ).first()
            
            if not workflow_instance:
                logger.error(f"Workflow instance {workflow_instance_id} not found")
                return None
            
            # Create FHIR Task resource
            fhir_task = await self._create_fhir_task(
                workflow_instance, task_definition_key, name, description,
                assignee, due_date, priority, variables
            )
            
            if not fhir_task:
                logger.error("Failed to create FHIR Task")
                return None
            
            # Create database task
            task = WorkflowTask(
                workflow_instance_id=workflow_instance_id,
                fhir_task_id=fhir_task.get("id"),
                task_definition_key=task_definition_key,
                name=name,
                description=description,
                assignee=assignee,
                candidate_groups=candidate_groups or [],
                status="created",
                priority=priority,
                due_date=due_date,
                form_key=form_key,
                variables=variables or {}
            )
            
            db.add(task)
            db.commit()
            db.refresh(task)
            
            # Create task assignment if assignee specified
            if assignee:
                await self._create_task_assignment(task.id, assignee, "direct", None, db)
            
            # Log task creation event
            await self._log_task_event(
                task.id,
                "task_created",
                {
                    "task_definition_key": task_definition_key,
                    "assignee": assignee,
                    "priority": priority,
                    "due_date": due_date.isoformat() if due_date else None
                },
                None,
                db
            )
            
            logger.info(f"Created task: {task.id} (FHIR: {task.fhir_task_id})")
            return task
            
        except Exception as e:
            logger.error(f"Error creating task: {e}")
            db.rollback()
            return None
    
    async def get_task(
        self,
        task_id: int,
        db: Optional[Session] = None
    ) -> Optional[WorkflowTask]:
        """
        Get task by ID.
        
        Args:
            task_id: Task ID
            db: Database session
            
        Returns:
            WorkflowTask or None if not found
        """
        if not db:
            db = next(get_db())
        
        try:
            return db.query(WorkflowTask).filter(WorkflowTask.id == task_id).first()
        except Exception as e:
            logger.error(f"Error getting task {task_id}: {e}")
            return None
    
    async def get_tasks(
        self,
        assignee: Optional[str] = None,
        patient_id: Optional[str] = None,
        status: Optional[str] = None,
        workflow_instance_id: Optional[int] = None,
        candidate_groups: Optional[List[str]] = None,
        db: Optional[Session] = None
    ) -> List[WorkflowTask]:
        """
        Get tasks with optional filters.
        
        Args:
            assignee: Filter by assignee
            patient_id: Filter by patient ID
            status: Filter by status
            workflow_instance_id: Filter by workflow instance
            candidate_groups: Filter by candidate groups
            db: Database session
            
        Returns:
            List of WorkflowTask objects
        """
        if not db:
            db = next(get_db())
        
        try:
            query = db.query(WorkflowTask)
            
            if assignee:
                query = query.filter(WorkflowTask.assignee == assignee)
            
            if status:
                query = query.filter(WorkflowTask.status == status)
            
            if workflow_instance_id:
                query = query.filter(WorkflowTask.workflow_instance_id == workflow_instance_id)
            
            if patient_id:
                # Join with workflow instance to filter by patient
                query = query.join(WorkflowInstance).filter(
                    WorkflowInstance.patient_id == patient_id
                )
            
            if candidate_groups:
                # Filter by candidate groups (JSON array contains any of the groups)
                for group in candidate_groups:
                    query = query.filter(WorkflowTask.candidate_groups.contains([group]))
            
            return query.order_by(WorkflowTask.created_at.desc()).all()
            
        except Exception as e:
            logger.error(f"Error getting tasks: {e}")
            return []
    
    async def claim_task(
        self,
        task_id: int,
        user_id: str,
        db: Optional[Session] = None
    ) -> Optional[WorkflowTask]:
        """
        Claim a task for a user.
        
        Args:
            task_id: Task ID
            user_id: User ID claiming the task
            db: Database session
            
        Returns:
            Updated WorkflowTask or None if failed
        """
        if not db:
            db = next(get_db())
        
        try:
            task = await self.get_task(task_id, db)
            if not task:
                return None
            
            if task.status != "created":
                logger.warning(f"Cannot claim task {task_id} with status {task.status}")
                return None
            
            if task.assignee and task.assignee != user_id:
                logger.warning(f"Task {task_id} is already assigned to {task.assignee}")
                return None
            
            # Update task
            task.assignee = user_id
            task.status = "assigned"
            task.updated_at = datetime.utcnow()
            db.commit()
            
            # Create task assignment
            await self._create_task_assignment(task_id, user_id, "claimed", user_id, db)
            
            # Update FHIR Task
            await self._update_fhir_task_status(task, "in-progress")
            
            # Log task claim event
            await self._log_task_event(
                task_id,
                "task_claimed",
                {"claimed_by": user_id},
                user_id,
                db
            )
            
            logger.info(f"Task {task_id} claimed by user {user_id}")
            return task
            
        except Exception as e:
            logger.error(f"Error claiming task {task_id}: {e}")
            db.rollback()
            return None
    
    async def complete_task(
        self,
        task_id: int,
        user_id: str,
        output_variables: Optional[Dict[str, Any]] = None,
        db: Optional[Session] = None
    ) -> Optional[WorkflowTask]:
        """
        Complete a task.
        
        Args:
            task_id: Task ID
            user_id: User completing the task
            output_variables: Task output variables
            db: Database session
            
        Returns:
            Updated WorkflowTask or None if failed
        """
        if not db:
            db = next(get_db())
        
        try:
            task = await self.get_task(task_id, db)
            if not task:
                return None
            
            if task.status not in ["assigned", "in-progress"]:
                logger.warning(f"Cannot complete task {task_id} with status {task.status}")
                return None
            
            if task.assignee != user_id:
                logger.warning(f"Task {task_id} is not assigned to user {user_id}")
                return None
            
            # Update task
            task.status = "completed"
            task.completed_at = datetime.utcnow()
            task.completed_by = user_id
            task.updated_at = datetime.utcnow()
            
            # Merge output variables
            if output_variables:
                current_variables = task.variables or {}
                current_variables.update(output_variables)
                task.variables = current_variables
            
            db.commit()
            
            # Update FHIR Task
            await self._update_fhir_task_status(task, "completed")
            
            # Log task completion event
            await self._log_task_event(
                task_id,
                "task_completed",
                {
                    "completed_by": user_id,
                    "output_variables": output_variables or {}
                },
                user_id,
                db
            )
            
            logger.info(f"Task {task_id} completed by user {user_id}")
            return task
            
        except Exception as e:
            logger.error(f"Error completing task {task_id}: {e}")
            db.rollback()
            return None
    
    async def delegate_task(
        self,
        task_id: int,
        from_user_id: str,
        to_user_id: str,
        notes: Optional[str] = None,
        db: Optional[Session] = None
    ) -> Optional[WorkflowTask]:
        """
        Delegate a task to another user.
        
        Args:
            task_id: Task ID
            from_user_id: User delegating the task
            to_user_id: User receiving the task
            notes: Delegation notes
            db: Database session
            
        Returns:
            Updated WorkflowTask or None if failed
        """
        if not db:
            db = next(get_db())
        
        try:
            task = await self.get_task(task_id, db)
            if not task:
                return None
            
            if task.assignee != from_user_id:
                logger.warning(f"Task {task_id} is not assigned to user {from_user_id}")
                return None
            
            if task.status not in ["assigned", "in-progress"]:
                logger.warning(f"Cannot delegate task {task_id} with status {task.status}")
                return None
            
            # Update task assignee
            task.assignee = to_user_id
            task.updated_at = datetime.utcnow()
            db.commit()
            
            # Create new task assignment
            await self._create_task_assignment(task_id, to_user_id, "delegated", from_user_id, db)
            
            # Add delegation comment
            if notes:
                await self.add_task_comment(task_id, from_user_id, f"Delegated to user {to_user_id}: {notes}", db)
            
            # Update FHIR Task
            await self._update_fhir_task_assignee(task, to_user_id)
            
            # Log task delegation event
            await self._log_task_event(
                task_id,
                "task_delegated",
                {
                    "from_user": from_user_id,
                    "to_user": to_user_id,
                    "notes": notes
                },
                from_user_id,
                db
            )
            
            logger.info(f"Task {task_id} delegated from {from_user_id} to {to_user_id}")
            return task
            
        except Exception as e:
            logger.error(f"Error delegating task {task_id}: {e}")
            db.rollback()
            return None
    
    async def add_task_comment(
        self,
        task_id: int,
        author_id: str,
        content: str,
        is_internal: bool = False,
        db: Optional[Session] = None
    ) -> Optional[TaskComment]:
        """
        Add comment to a task.
        
        Args:
            task_id: Task ID
            author_id: Comment author user ID
            content: Comment content
            is_internal: Whether comment is internal
            db: Database session
            
        Returns:
            Created TaskComment or None if failed
        """
        if not db:
            db = next(get_db())
        
        try:
            comment = TaskComment(
                task_id=task_id,
                author_id=author_id,
                content=content,
                is_internal=is_internal
            )
            
            db.add(comment)
            db.commit()
            db.refresh(comment)
            
            # Log comment event
            await self._log_task_event(
                task_id,
                "task_comment_added",
                {
                    "author_id": author_id,
                    "is_internal": is_internal,
                    "content_length": len(content)
                },
                author_id,
                db
            )
            
            logger.info(f"Added comment to task {task_id} by user {author_id}")
            return comment
            
        except Exception as e:
            logger.error(f"Error adding comment to task {task_id}: {e}")
            db.rollback()
            return None
    
    async def escalate_task(
        self,
        task_id: int,
        escalated_to: str,
        escalation_reason: str = "overdue",
        escalation_level: int = 1,
        escalated_by: Optional[str] = None,
        db: Optional[Session] = None
    ) -> Optional[TaskEscalation]:
        """
        Escalate a task.
        
        Args:
            task_id: Task ID
            escalated_to: User or group to escalate to
            escalation_reason: Reason for escalation
            escalation_level: Escalation level
            escalated_by: User escalating (None for system)
            db: Database session
            
        Returns:
            Created TaskEscalation or None if failed
        """
        if not db:
            db = next(get_db())
        
        try:
            escalation = TaskEscalation(
                task_id=task_id,
                escalation_level=escalation_level,
                escalated_to=escalated_to,
                escalated_by=escalated_by,
                escalation_reason=escalation_reason
            )
            
            db.add(escalation)
            db.commit()
            db.refresh(escalation)
            
            # Log escalation event
            await self._log_task_event(
                task_id,
                "task_escalated",
                {
                    "escalated_to": escalated_to,
                    "escalation_reason": escalation_reason,
                    "escalation_level": escalation_level,
                    "escalated_by": escalated_by
                },
                escalated_by,
                db
            )
            
            logger.info(f"Escalated task {task_id} to {escalated_to} (level {escalation_level})")
            return escalation

        except Exception as e:
            logger.error(f"Error escalating task {task_id}: {e}")
            db.rollback()
            return None

    async def _create_task_assignment(
        self,
        task_id: int,
        assignee_id: str,
        assignment_type: str,
        assigned_by: Optional[str] = None,
        db: Optional[Session] = None
    ) -> Optional[TaskAssignment]:
        """
        Create task assignment record.

        Args:
            task_id: Task ID
            assignee_id: User ID being assigned
            assignment_type: Type of assignment (direct, claimed, delegated)
            assigned_by: User making the assignment
            db: Database session

        Returns:
            Created TaskAssignment or None if failed
        """
        if not db:
            db = next(get_db())

        try:
            # Deactivate previous assignments
            db.query(TaskAssignment).filter(
                TaskAssignment.task_id == task_id,
                TaskAssignment.is_active == True
            ).update({"is_active": False, "revoked_at": datetime.utcnow()})

            # Create new assignment
            assignment = TaskAssignment(
                task_id=task_id,
                assignee_id=assignee_id,
                assigned_by=assigned_by,
                assignment_type=assignment_type
            )

            db.add(assignment)
            db.commit()
            db.refresh(assignment)

            return assignment

        except Exception as e:
            logger.error(f"Error creating task assignment: {e}")
            db.rollback()
            return None

    async def _create_fhir_task(
        self,
        workflow_instance: WorkflowInstance,
        task_definition_key: str,
        name: str,
        description: Optional[str] = None,
        assignee: Optional[str] = None,
        due_date: Optional[datetime] = None,
        priority: int = 50,
        variables: Optional[Dict[str, Any]] = None
    ) -> Optional[Dict[str, Any]]:
        """
        Create FHIR Task resource.

        Args:
            workflow_instance: WorkflowInstance object
            task_definition_key: Task definition key
            name: Task name
            description: Task description
            assignee: Assigned user ID
            due_date: Task due date
            priority: Task priority
            variables: Task variables

        Returns:
            Created Task resource or None if failed
        """
        try:
            # Map priority to FHIR priority
            fhir_priority = "routine"
            if priority >= 80:
                fhir_priority = "urgent"
            elif priority >= 60:
                fhir_priority = "asap"
            elif priority <= 20:
                fhir_priority = "stat"

            task_resource = {
                "resourceType": "Task",
                "status": "requested",
                "intent": "order",
                "priority": fhir_priority,
                "code": {
                    "coding": [{
                        "system": "http://clinical-synthesis-hub.com/fhir/CodeSystem/workflow-task-type",
                        "code": task_definition_key,
                        "display": name
                    }]
                },
                "description": description or name,
                "focus": {
                    "reference": f"Patient/{workflow_instance.patient_id}",
                    "display": f"Patient {workflow_instance.patient_id}"
                },
                "for": {
                    "reference": f"Patient/{workflow_instance.patient_id}",
                    "display": f"Patient {workflow_instance.patient_id}"
                },
                "requester": {
                    "reference": f"WorkflowInstance/{workflow_instance.id}",
                    "display": f"Workflow Instance {workflow_instance.id}"
                },
                "authoredOn": datetime.utcnow().isoformat() + "Z",
                "businessStatus": {
                    "coding": [{
                        "system": "http://clinical-synthesis-hub.com/fhir/CodeSystem/task-business-status",
                        "code": "created",
                        "display": "Created"
                    }]
                }
            }

            if assignee:
                task_resource["owner"] = {
                    "reference": f"User/{assignee}",
                    "display": f"User {assignee}"
                }

            if due_date:
                task_resource["restriction"] = {
                    "period": {
                        "end": due_date.isoformat() + "Z"
                    }
                }

            if variables:
                task_resource["input"] = []
                for key, value in variables.items():
                    task_resource["input"].append({
                        "type": {
                            "coding": [{
                                "system": "http://clinical-synthesis-hub.com/fhir/CodeSystem/task-input-type",
                                "code": key,
                                "display": key.replace("_", " ").title()
                            }]
                        },
                        "valueString": str(value)
                    })

            # Create the resource in Google Healthcare API
            return await self.fhir_service.create_resource("Task", task_resource)

        except Exception as e:
            logger.error(f"Error creating FHIR Task: {e}")
            return None

    async def _update_fhir_task_status(
        self,
        task: WorkflowTask,
        status: str
    ) -> bool:
        """
        Update FHIR Task status.

        Args:
            task: WorkflowTask object
            status: New FHIR status

        Returns:
            True if updated successfully, False otherwise
        """
        try:
            if not task.fhir_task_id:
                return False

            # Get existing Task
            fhir_task = await self.fhir_service.get_resource("Task", task.fhir_task_id)
            if not fhir_task:
                return False

            # Update status
            fhir_task["status"] = status

            # Update business status
            business_status_map = {
                "requested": "created",
                "in-progress": "in-progress",
                "completed": "completed",
                "cancelled": "cancelled",
                "failed": "failed"
            }

            if status in business_status_map:
                fhir_task["businessStatus"] = {
                    "coding": [{
                        "system": "http://clinical-synthesis-hub.com/fhir/CodeSystem/task-business-status",
                        "code": business_status_map[status],
                        "display": business_status_map[status].replace("-", " ").title()
                    }]
                }

            # Update the resource
            updated = await self.fhir_service.update_resource("Task", task.fhir_task_id, fhir_task)
            return updated is not None

        except Exception as e:
            logger.error(f"Error updating FHIR Task status {task.fhir_task_id}: {e}")
            return False

    async def _update_fhir_task_assignee(
        self,
        task: WorkflowTask,
        assignee: str
    ) -> bool:
        """
        Update FHIR Task assignee.

        Args:
            task: WorkflowTask object
            assignee: New assignee user ID

        Returns:
            True if updated successfully, False otherwise
        """
        try:
            if not task.fhir_task_id:
                return False

            # Get existing Task
            fhir_task = await self.fhir_service.get_resource("Task", task.fhir_task_id)
            if not fhir_task:
                return False

            # Update owner
            fhir_task["owner"] = {
                "reference": f"User/{assignee}",
                "display": f"User {assignee}"
            }

            # Update the resource
            updated = await self.fhir_service.update_resource("Task", task.fhir_task_id, fhir_task)
            return updated is not None

        except Exception as e:
            logger.error(f"Error updating FHIR Task assignee {task.fhir_task_id}: {e}")
            return False

    async def _log_task_event(
        self,
        task_id: int,
        event_type: str,
        event_data: Dict[str, Any],
        user_id: Optional[str] = None,
        db: Optional[Session] = None
    ) -> None:
        """
        Log task event to database and Supabase.

        Args:
            task_id: Task ID
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
                task_id=task_id,
                event_type=event_type,
                event_data=event_data,
                user_id=user_id,
                source="task-service"
            )

            db.add(event)
            db.commit()

            # Log to Supabase for analytics
            await self.supabase_service.log_workflow_event({
                "task_id": task_id,
                "event_type": event_type,
                "event_data": event_data,
                "user_id": user_id,
                "source": "task-service",
                "timestamp": datetime.utcnow().isoformat()
            })

        except Exception as e:
            logger.error(f"Error logging task event: {e}")


# Global service instance
task_service = TaskService()
