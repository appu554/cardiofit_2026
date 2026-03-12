"""
Escalation Service for advanced escalation mechanisms.
Handles multi-level escalations, automatic reassignment, and escalation notifications.
"""

import logging
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any
from sqlalchemy.orm import Session
from sqlalchemy import and_, or_

from ..models.workflow_models import WorkflowTask, WorkflowInstance, WorkflowEvent
from ..db.database import get_db
from .supabase_service import supabase_service
from .event_publisher import event_publisher
from .timer_service import timer_service

logger = logging.getLogger(__name__)


class EscalationRule:
    """Represents an escalation rule configuration."""
    
    def __init__(
        self,
        level: int,
        delay_minutes: int,
        target_type: str,  # user, role, supervisor, manager
        target_value: str,
        action: str = "reassign",  # reassign, notify, escalate
        conditions: Optional[Dict[str, Any]] = None
    ):
        self.level = level
        self.delay_minutes = delay_minutes
        self.target_type = target_type
        self.target_value = target_value
        self.action = action
        self.conditions = conditions or {}


class EscalationService:
    """
    Advanced escalation service for workflow tasks and processes.
    Manages multi-level escalation chains and automatic escalation handling.
    """
    
    def __init__(self):
        self.supabase_service = supabase_service
        self.event_publisher = event_publisher
        self.timer_service = timer_service
        self.escalation_rules: Dict[str, List[EscalationRule]] = {}
        
    async def initialize(self) -> bool:
        """Initialize the escalation service."""
        try:
            logger.info("Initializing Escalation Service...")
            
            # Load default escalation rules
            self._load_default_escalation_rules()
            
            logger.info("Escalation Service initialized successfully")
            return True
            
        except Exception as e:
            logger.error(f"Error initializing Escalation Service: {e}")
            return False
    
    def _load_default_escalation_rules(self):
        """Load default escalation rules for different task types."""
        
        # Default escalation rules for human tasks
        self.escalation_rules["human_task"] = [
            EscalationRule(1, 60, "supervisor", "direct_supervisor", "notify"),
            EscalationRule(2, 120, "supervisor", "direct_supervisor", "reassign"),
            EscalationRule(3, 240, "role", "task_manager", "reassign"),
            EscalationRule(4, 480, "role", "department_head", "escalate")
        ]
        
        # Escalation rules for critical tasks
        self.escalation_rules["critical_task"] = [
            EscalationRule(1, 30, "supervisor", "direct_supervisor", "notify"),
            EscalationRule(2, 60, "supervisor", "direct_supervisor", "reassign"),
            EscalationRule(3, 120, "role", "emergency_team", "reassign"),
            EscalationRule(4, 180, "role", "department_head", "escalate")
        ]
        
        # Escalation rules for approval tasks
        self.escalation_rules["approval_task"] = [
            EscalationRule(1, 120, "supervisor", "direct_supervisor", "notify"),
            EscalationRule(2, 240, "role", "approver_backup", "reassign"),
            EscalationRule(3, 480, "role", "senior_approver", "reassign")
        ]
        
        # Escalation rules for workflow timeouts
        self.escalation_rules["workflow_timeout"] = [
            EscalationRule(1, 60, "role", "workflow_admin", "notify"),
            EscalationRule(2, 180, "role", "process_owner", "escalate"),
            EscalationRule(3, 360, "role", "system_admin", "escalate")
        ]
    
    async def create_escalation_chain(
        self,
        task_id: int,
        escalation_type: str = "human_task",
        custom_rules: Optional[List[EscalationRule]] = None,
        db: Optional[Session] = None
    ) -> bool:
        """
        Create an escalation chain for a task.
        
        Args:
            task_id: Task ID to create escalation for
            escalation_type: Type of escalation (human_task, critical_task, etc.)
            custom_rules: Custom escalation rules to use
            db: Database session
            
        Returns:
            True if escalation chain created successfully
        """
        if not db:
            db = next(get_db())
        
        try:
            # Get task
            task = db.query(WorkflowTask).filter(WorkflowTask.id == task_id).first()
            if not task:
                logger.error(f"Task {task_id} not found")
                return False
            
            # Get escalation rules
            rules = custom_rules or self.escalation_rules.get(escalation_type, [])
            if not rules:
                logger.warning(f"No escalation rules found for type: {escalation_type}")
                return False
            
            # Create escalation timers for each level
            for rule in rules:
                escalation_time = datetime.utcnow() + timedelta(minutes=rule.delay_minutes)
                
                await self.timer_service.create_timer(
                    workflow_instance_id=task.workflow_instance_id,
                    timer_name=f"escalation_level_{rule.level}_task_{task_id}",
                    due_date=escalation_time,
                    timer_type="escalation",
                    timer_data={
                        "task_id": task_id,
                        "escalation_level": rule.level,
                        "escalation_rule": {
                            "target_type": rule.target_type,
                            "target_value": rule.target_value,
                            "action": rule.action,
                            "conditions": rule.conditions
                        }
                    },
                    callback_name="task_escalation",
                    db=db
                )
            
            # Update task with escalation info
            task.escalation_data = {
                "escalation_type": escalation_type,
                "escalation_created": datetime.utcnow().isoformat(),
                "escalation_levels": len(rules)
            }
            db.commit()
            
            logger.info(f"Created escalation chain for task {task_id} with {len(rules)} levels")
            return True
            
        except Exception as e:
            logger.error(f"Error creating escalation chain for task {task_id}: {e}")
            return False
    
    async def handle_task_escalation(
        self,
        task_id: int,
        escalation_level: int,
        escalation_rule: Dict[str, Any],
        db: Optional[Session] = None
    ) -> bool:
        """
        Handle task escalation when timer fires.
        
        Args:
            task_id: Task ID to escalate
            escalation_level: Current escalation level
            escalation_rule: Escalation rule configuration
            db: Database session
            
        Returns:
            True if escalation handled successfully
        """
        if not db:
            db = next(get_db())
        
        try:
            # Get task
            task = db.query(WorkflowTask).filter(WorkflowTask.id == task_id).first()
            if not task:
                logger.error(f"Task {task_id} not found for escalation")
                return False
            
            # Check if task is still eligible for escalation
            if task.status not in ["created", "assigned"]:
                logger.info(f"Task {task_id} no longer eligible for escalation (status: {task.status})")
                return True
            
            # Check escalation conditions
            if not self._check_escalation_conditions(task, escalation_rule.get("conditions", {})):
                logger.info(f"Escalation conditions not met for task {task_id}")
                return True
            
            # Perform escalation action
            action = escalation_rule.get("action", "notify")
            
            if action == "notify":
                await self._send_escalation_notification(task, escalation_level, escalation_rule, db)
            elif action == "reassign":
                await self._reassign_task(task, escalation_level, escalation_rule, db)
            elif action == "escalate":
                await self._escalate_to_higher_level(task, escalation_level, escalation_rule, db)
            
            # Log escalation event
            await self._log_escalation_event(task, escalation_level, escalation_rule, action, db)
            
            logger.info(f"Handled escalation level {escalation_level} for task {task_id}: {action}")
            return True
            
        except Exception as e:
            logger.error(f"Error handling task escalation: {e}")
            return False
    
    def _check_escalation_conditions(
        self,
        task: WorkflowTask,
        conditions: Dict[str, Any]
    ) -> bool:
        """Check if escalation conditions are met."""
        try:
            # Check priority condition
            if "min_priority" in conditions:
                if task.priority < conditions["min_priority"]:
                    return False
            
            # Check task age condition
            if "min_age_hours" in conditions:
                age_hours = (datetime.utcnow() - task.created_at).total_seconds() / 3600
                if age_hours < conditions["min_age_hours"]:
                    return False
            
            # Check assignee condition
            if "require_assignee" in conditions:
                if conditions["require_assignee"] and not task.assignee:
                    return False
            
            # Check custom conditions
            if "custom" in conditions:
                # Implement custom condition logic here
                pass
            
            return True
            
        except Exception as e:
            logger.error(f"Error checking escalation conditions: {e}")
            return False
    
    async def _send_escalation_notification(
        self,
        task: WorkflowTask,
        escalation_level: int,
        escalation_rule: Dict[str, Any],
        db: Session
    ):
        """Send escalation notification."""
        try:
            # Determine notification target
            target = await self._resolve_escalation_target(
                escalation_rule["target_type"],
                escalation_rule["target_value"],
                task,
                db
            )
            
            # Publish notification event
            await self.event_publisher.publish_event(
                "escalation_notification",
                {
                    "task_id": task.id,
                    "escalation_level": escalation_level,
                    "target": target,
                    "task_name": task.name,
                    "task_priority": task.priority,
                    "workflow_instance_id": task.workflow_instance_id,
                    "notification_type": "escalation"
                }
            )
            
        except Exception as e:
            logger.error(f"Error sending escalation notification: {e}")
    
    async def _reassign_task(
        self,
        task: WorkflowTask,
        escalation_level: int,
        escalation_rule: Dict[str, Any],
        db: Session
    ):
        """Reassign task to escalation target."""
        try:
            # Determine reassignment target
            target = await self._resolve_escalation_target(
                escalation_rule["target_type"],
                escalation_rule["target_value"],
                task,
                db
            )
            
            # Store original assignee
            original_assignee = task.assignee
            
            # Reassign task
            task.assignee = target
            task.escalation_level = escalation_level
            task.escalated = True
            task.escalated_at = datetime.utcnow()
            
            # Update task data
            task.task_data = {
                **task.task_data,
                "escalation_history": task.task_data.get("escalation_history", []) + [{
                    "level": escalation_level,
                    "from_assignee": original_assignee,
                    "to_assignee": target,
                    "escalated_at": datetime.utcnow().isoformat(),
                    "reason": "automatic_escalation"
                }]
            }
            
            db.commit()
            
            # Publish reassignment event
            await self.event_publisher.publish_event(
                "task_reassigned",
                {
                    "task_id": task.id,
                    "escalation_level": escalation_level,
                    "original_assignee": original_assignee,
                    "new_assignee": target,
                    "reason": "escalation",
                    "workflow_instance_id": task.workflow_instance_id
                }
            )
            
        except Exception as e:
            logger.error(f"Error reassigning task: {e}")
    
    async def _escalate_to_higher_level(
        self,
        task: WorkflowTask,
        escalation_level: int,
        escalation_rule: Dict[str, Any],
        db: Session
    ):
        """Escalate to higher organizational level."""
        try:
            # Determine escalation target
            target = await self._resolve_escalation_target(
                escalation_rule["target_type"],
                escalation_rule["target_value"],
                task,
                db
            )
            
            # Create escalation record
            task.escalation_level = escalation_level
            task.escalated = True
            task.escalated_at = datetime.utcnow()
            
            # Update task data
            task.task_data = {
                **task.task_data,
                "escalation_target": target,
                "escalation_reason": "higher_level_escalation",
                "escalated_at": datetime.utcnow().isoformat()
            }
            
            db.commit()
            
            # Publish escalation event
            await self.event_publisher.publish_event(
                "task_escalated",
                {
                    "task_id": task.id,
                    "escalation_level": escalation_level,
                    "escalation_target": target,
                    "task_name": task.name,
                    "workflow_instance_id": task.workflow_instance_id,
                    "escalation_type": "higher_level"
                }
            )
            
        except Exception as e:
            logger.error(f"Error escalating to higher level: {e}")
    
    async def _resolve_escalation_target(
        self,
        target_type: str,
        target_value: str,
        task: WorkflowTask,
        db: Session
    ) -> str:
        """Resolve escalation target based on type and value."""
        try:
            if target_type == "user":
                return target_value
            
            elif target_type == "role":
                # Find users with the specified role
                # This would integrate with your user/role management system
                return f"role:{target_value}"
            
            elif target_type == "supervisor":
                # Find supervisor of current assignee
                if task.assignee:
                    # This would integrate with your organizational hierarchy
                    return f"supervisor_of:{task.assignee}"
                return "supervisor:default"
            
            elif target_type == "manager":
                # Find manager in organizational hierarchy
                return f"manager:{target_value}"
            
            else:
                logger.warning(f"Unknown escalation target type: {target_type}")
                return target_value
                
        except Exception as e:
            logger.error(f"Error resolving escalation target: {e}")
            return target_value
    
    async def _log_escalation_event(
        self,
        task: WorkflowTask,
        escalation_level: int,
        escalation_rule: Dict[str, Any],
        action: str,
        db: Session
    ):
        """Log escalation event for audit trail."""
        try:
            event = WorkflowEvent(
                workflow_instance_id=task.workflow_instance_id,
                event_type="task_escalation",
                event_data={
                    "task_id": task.id,
                    "escalation_level": escalation_level,
                    "escalation_rule": escalation_rule,
                    "action": action,
                    "task_name": task.name,
                    "task_assignee": task.assignee,
                    "escalated_at": datetime.utcnow().isoformat()
                }
            )
            
            db.add(event)
            db.commit()
            
        except Exception as e:
            logger.error(f"Error logging escalation event: {e}")
    
    async def cancel_escalation_chain(
        self,
        task_id: int,
        reason: str = "task_completed",
        db: Optional[Session] = None
    ) -> bool:
        """
        Cancel escalation chain for a task.
        
        Args:
            task_id: Task ID to cancel escalation for
            reason: Cancellation reason
            db: Database session
            
        Returns:
            True if escalation chain cancelled successfully
        """
        if not db:
            db = next(get_db())
        
        try:
            # Find and cancel escalation timers for this task
            from ..models.workflow_models import WorkflowTimer
            
            escalation_timers = db.query(WorkflowTimer).filter(
                WorkflowTimer.timer_name.like(f"%task_{task_id}"),
                WorkflowTimer.status == "active"
            ).all()
            
            for timer in escalation_timers:
                await self.timer_service.cancel_timer(timer.id, reason, db)
            
            logger.info(f"Cancelled escalation chain for task {task_id}: {reason}")
            return True
            
        except Exception as e:
            logger.error(f"Error cancelling escalation chain for task {task_id}: {e}")
            return False


# Global escalation service instance
escalation_service = EscalationService()
