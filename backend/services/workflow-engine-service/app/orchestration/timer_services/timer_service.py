"""
Timer Service for comprehensive workflow timer management.
Handles scheduling, execution, and management of time-based workflow events.
"""

import asyncio
import logging
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Callable
from sqlalchemy.orm import Session
from sqlalchemy import and_, or_

from ..models.workflow_models import WorkflowTimer, WorkflowInstance, WorkflowEvent
from ..db.database import get_db
from .supabase_service import supabase_service
from .event_publisher import event_publisher

logger = logging.getLogger(__name__)


class TimerService:
    """
    Comprehensive timer service for workflow engine.
    Manages timer scheduling, execution, and lifecycle.
    """
    
    def __init__(self):
        self.supabase_service = supabase_service
        self.event_publisher = event_publisher
        self.running = False
        self.timer_tasks: Dict[int, asyncio.Task] = {}
        self.timer_callbacks: Dict[str, Callable] = {}
        
    async def initialize(self) -> bool:
        """Initialize the timer service."""
        try:
            logger.info("Initializing Timer Service...")
            
            # Register default timer callbacks
            self._register_default_callbacks()
            
            # Load active timers from database
            await self._load_active_timers()
            
            logger.info("Timer Service initialized successfully")
            return True
            
        except Exception as e:
            logger.error(f"Error initializing Timer Service: {e}")
            return False
    
    def _register_default_callbacks(self):
        """Register default timer callback handlers."""
        self.timer_callbacks.update({
            "escalation": self._handle_escalation_timer,
            "deadline": self._handle_deadline_timer,
            "reminder": self._handle_reminder_timer,
            "timeout": self._handle_timeout_timer,
            "recurring": self._handle_recurring_timer,
            "workflow_timeout": self._handle_workflow_timeout,
            "task_timeout": self._handle_task_timeout,
            "notification": self._handle_notification_timer
        })
    
    async def create_timer(
        self,
        workflow_instance_id: int,
        timer_name: str,
        due_date: datetime,
        timer_type: str = "deadline",
        repeat_interval: Optional[str] = None,
        timer_data: Optional[Dict[str, Any]] = None,
        callback_name: Optional[str] = None,
        db: Optional[Session] = None
    ) -> Optional[WorkflowTimer]:
        """
        Create a new workflow timer.
        
        Args:
            workflow_instance_id: Workflow instance ID
            timer_name: Timer name/identifier
            due_date: When the timer should fire
            timer_type: Type of timer (deadline, escalation, reminder, etc.)
            repeat_interval: ISO 8601 duration for recurring timers
            timer_data: Additional timer data
            callback_name: Callback function name
            db: Database session
            
        Returns:
            Created timer or None if failed
        """
        if not db:
            db = next(get_db())
        
        try:
            # Create timer record
            timer = WorkflowTimer(
                workflow_instance_id=workflow_instance_id,
                timer_name=timer_name,
                due_date=due_date,
                repeat_interval=repeat_interval,
                status="active",
                timer_data={
                    "type": timer_type,
                    "callback": callback_name or timer_type,
                    **(timer_data or {})
                }
            )
            
            db.add(timer)
            db.commit()
            db.refresh(timer)
            
            # Schedule timer execution
            await self._schedule_timer(timer)
            
            logger.info(f"Created timer {timer.id}: {timer_name} for workflow {workflow_instance_id}")
            return timer
            
        except Exception as e:
            logger.error(f"Error creating timer: {e}")
            db.rollback()
            return None
    
    async def cancel_timer(
        self,
        timer_id: int,
        reason: str = "cancelled",
        db: Optional[Session] = None
    ) -> bool:
        """
        Cancel an active timer.
        
        Args:
            timer_id: Timer ID to cancel
            reason: Cancellation reason
            db: Database session
            
        Returns:
            True if cancelled successfully
        """
        if not db:
            db = next(get_db())
        
        try:
            timer = db.query(WorkflowTimer).filter(WorkflowTimer.id == timer_id).first()
            if not timer:
                return False
            
            # Cancel scheduled task
            if timer_id in self.timer_tasks:
                self.timer_tasks[timer_id].cancel()
                del self.timer_tasks[timer_id]
            
            # Update timer status
            timer.status = "cancelled"
            timer.timer_data = {
                **timer.timer_data,
                "cancellation_reason": reason,
                "cancelled_at": datetime.utcnow().isoformat()
            }
            
            db.commit()
            
            logger.info(f"Cancelled timer {timer_id}: {reason}")
            return True
            
        except Exception as e:
            logger.error(f"Error cancelling timer {timer_id}: {e}")
            return False
    
    async def _schedule_timer(self, timer: WorkflowTimer):
        """Schedule a timer for execution."""
        try:
            # Calculate delay until timer fires
            now = datetime.utcnow()
            delay = (timer.due_date - now).total_seconds()
            
            if delay <= 0:
                # Timer is already due, fire immediately
                await self._fire_timer(timer)
            else:
                # Schedule timer task
                task = asyncio.create_task(self._timer_task(timer, delay))
                self.timer_tasks[timer.id] = task
                
        except Exception as e:
            logger.error(f"Error scheduling timer {timer.id}: {e}")
    
    async def _timer_task(self, timer: WorkflowTimer, delay: float):
        """Timer task that waits and then fires the timer."""
        try:
            await asyncio.sleep(delay)
            await self._fire_timer(timer)
            
        except asyncio.CancelledError:
            logger.info(f"Timer {timer.id} was cancelled")
        except Exception as e:
            logger.error(f"Error in timer task {timer.id}: {e}")
        finally:
            # Clean up task reference
            if timer.id in self.timer_tasks:
                del self.timer_tasks[timer.id]
    
    async def _fire_timer(self, timer: WorkflowTimer):
        """Fire a timer and execute its callback."""
        try:
            db = next(get_db())
            
            # Update timer status
            timer.fired_at = datetime.utcnow()
            timer.status = "fired"
            db.commit()
            
            # Get callback function
            callback_name = timer.timer_data.get("callback", "deadline")
            callback = self.timer_callbacks.get(callback_name)
            
            if callback:
                await callback(timer, db)
            else:
                logger.warning(f"No callback found for timer type: {callback_name}")
            
            # Handle recurring timers
            if timer.repeat_interval:
                await self._schedule_recurring_timer(timer, db)
            
            logger.info(f"Fired timer {timer.id}: {timer.timer_name}")
            
        except Exception as e:
            logger.error(f"Error firing timer {timer.id}: {e}")
    
    async def _schedule_recurring_timer(self, timer: WorkflowTimer, db: Session):
        """Schedule the next occurrence of a recurring timer."""
        try:
            # Parse repeat interval (simplified ISO 8601 duration)
            interval = self._parse_duration(timer.repeat_interval)
            if not interval:
                return
            
            # Create new timer for next occurrence
            next_due_date = timer.due_date + interval
            
            new_timer = WorkflowTimer(
                workflow_instance_id=timer.workflow_instance_id,
                timer_name=timer.timer_name,
                due_date=next_due_date,
                repeat_interval=timer.repeat_interval,
                status="active",
                timer_data=timer.timer_data
            )
            
            db.add(new_timer)
            db.commit()
            db.refresh(new_timer)
            
            # Schedule the new timer
            await self._schedule_timer(new_timer)
            
        except Exception as e:
            logger.error(f"Error scheduling recurring timer: {e}")
    
    def _parse_duration(self, duration_str: str) -> Optional[timedelta]:
        """Parse ISO 8601 duration string to timedelta."""
        try:
            # Simplified parser for common durations
            if duration_str.startswith("PT"):
                # Time duration
                duration_str = duration_str[2:]  # Remove PT prefix
                
                hours = 0
                minutes = 0
                seconds = 0
                
                if "H" in duration_str:
                    hours = int(duration_str.split("H")[0])
                    duration_str = duration_str.split("H")[1]
                
                if "M" in duration_str:
                    minutes = int(duration_str.split("M")[0])
                    duration_str = duration_str.split("M")[1]
                
                if "S" in duration_str:
                    seconds = int(duration_str.split("S")[0])
                
                return timedelta(hours=hours, minutes=minutes, seconds=seconds)
            
            elif duration_str.startswith("P"):
                # Date duration
                duration_str = duration_str[1:]  # Remove P prefix
                
                days = 0
                if "D" in duration_str:
                    days = int(duration_str.split("D")[0])
                
                return timedelta(days=days)
            
            return None
            
        except Exception as e:
            logger.error(f"Error parsing duration {duration_str}: {e}")
            return None
    
    async def _load_active_timers(self):
        """Load and schedule active timers from database."""
        try:
            db = next(get_db())
            
            # Get all active timers
            active_timers = db.query(WorkflowTimer).filter(
                WorkflowTimer.status == "active",
                WorkflowTimer.due_date > datetime.utcnow()
            ).all()
            
            for timer in active_timers:
                await self._schedule_timer(timer)
            
            logger.info(f"Loaded {len(active_timers)} active timers")
            
        except Exception as e:
            logger.error(f"Error loading active timers: {e}")
    
    # Timer callback handlers
    async def _handle_escalation_timer(self, timer: WorkflowTimer, db: Session):
        """Handle escalation timer."""
        try:
            # Publish escalation event
            await self.event_publisher.publish_event(
                "timer_escalation",
                {
                    "timer_id": timer.id,
                    "workflow_instance_id": timer.workflow_instance_id,
                    "timer_name": timer.timer_name,
                    "timer_data": timer.timer_data
                }
            )
            
        except Exception as e:
            logger.error(f"Error handling escalation timer: {e}")
    
    async def _handle_deadline_timer(self, timer: WorkflowTimer, db: Session):
        """Handle deadline timer."""
        try:
            # Publish deadline event
            await self.event_publisher.publish_event(
                "timer_deadline",
                {
                    "timer_id": timer.id,
                    "workflow_instance_id": timer.workflow_instance_id,
                    "timer_name": timer.timer_name,
                    "timer_data": timer.timer_data
                }
            )
            
        except Exception as e:
            logger.error(f"Error handling deadline timer: {e}")
    
    async def _handle_reminder_timer(self, timer: WorkflowTimer, db: Session):
        """Handle reminder timer."""
        try:
            # Publish reminder event
            await self.event_publisher.publish_event(
                "timer_reminder",
                {
                    "timer_id": timer.id,
                    "workflow_instance_id": timer.workflow_instance_id,
                    "timer_name": timer.timer_name,
                    "timer_data": timer.timer_data
                }
            )
            
        except Exception as e:
            logger.error(f"Error handling reminder timer: {e}")
    
    async def _handle_timeout_timer(self, timer: WorkflowTimer, db: Session):
        """Handle timeout timer."""
        try:
            # Publish timeout event
            await self.event_publisher.publish_event(
                "timer_timeout",
                {
                    "timer_id": timer.id,
                    "workflow_instance_id": timer.workflow_instance_id,
                    "timer_name": timer.timer_name,
                    "timer_data": timer.timer_data
                }
            )
            
        except Exception as e:
            logger.error(f"Error handling timeout timer: {e}")
    
    async def _handle_recurring_timer(self, timer: WorkflowTimer, db: Session):
        """Handle recurring timer."""
        try:
            # Publish recurring event
            await self.event_publisher.publish_event(
                "timer_recurring",
                {
                    "timer_id": timer.id,
                    "workflow_instance_id": timer.workflow_instance_id,
                    "timer_name": timer.timer_name,
                    "timer_data": timer.timer_data
                }
            )
            
        except Exception as e:
            logger.error(f"Error handling recurring timer: {e}")
    
    async def _handle_workflow_timeout(self, timer: WorkflowTimer, db: Session):
        """Handle workflow timeout timer."""
        try:
            # Publish workflow timeout event
            await self.event_publisher.publish_event(
                "workflow_timeout",
                {
                    "timer_id": timer.id,
                    "workflow_instance_id": timer.workflow_instance_id,
                    "timer_name": timer.timer_name,
                    "timer_data": timer.timer_data
                }
            )
            
        except Exception as e:
            logger.error(f"Error handling workflow timeout timer: {e}")
    
    async def _handle_task_timeout(self, timer: WorkflowTimer, db: Session):
        """Handle task timeout timer."""
        try:
            # Publish task timeout event
            await self.event_publisher.publish_event(
                "task_timeout",
                {
                    "timer_id": timer.id,
                    "workflow_instance_id": timer.workflow_instance_id,
                    "timer_name": timer.timer_name,
                    "timer_data": timer.timer_data
                }
            )
            
        except Exception as e:
            logger.error(f"Error handling task timeout timer: {e}")
    
    async def _handle_notification_timer(self, timer: WorkflowTimer, db: Session):
        """Handle notification timer."""
        try:
            # Publish notification event
            await self.event_publisher.publish_event(
                "timer_notification",
                {
                    "timer_id": timer.id,
                    "workflow_instance_id": timer.workflow_instance_id,
                    "timer_name": timer.timer_name,
                    "timer_data": timer.timer_data
                }
            )
            
        except Exception as e:
            logger.error(f"Error handling notification timer: {e}")


# Global timer service instance
timer_service = TimerService()
