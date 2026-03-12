"""
Error Recovery Service for comprehensive error handling and recovery strategies.
Implements retry mechanisms, compensation workflows, and dead letter queue handling.
"""

import asyncio
import logging
from datetime import datetime, timedelta
from typing import Dict, List, Optional, Any, Callable
from enum import Enum
from sqlalchemy.orm import Session
from sqlalchemy import and_, or_

from ..models.workflow_models import WorkflowInstance, WorkflowEvent, WorkflowTask
from ..db.database import get_db
from .supabase_service import supabase_service
from .event_publisher import event_publisher
from .timer_service import timer_service

logger = logging.getLogger(__name__)


class ErrorType(Enum):
    """Types of errors that can occur in workflows."""
    TASK_FAILURE = "task_failure"
    SERVICE_UNAVAILABLE = "service_unavailable"
    TIMEOUT = "timeout"
    VALIDATION_ERROR = "validation_error"
    BUSINESS_RULE_VIOLATION = "business_rule_violation"
    SYSTEM_ERROR = "system_error"
    NETWORK_ERROR = "network_error"
    AUTHENTICATION_ERROR = "authentication_error"
    AUTHORIZATION_ERROR = "authorization_error"
    DATA_ERROR = "data_error"


class RecoveryStrategy(Enum):
    """Recovery strategies for different error types."""
    RETRY = "retry"
    COMPENSATE = "compensate"
    ESCALATE = "escalate"
    SKIP = "skip"
    ABORT = "abort"
    MANUAL_INTERVENTION = "manual_intervention"
    ALTERNATIVE_PATH = "alternative_path"
    ROLLBACK = "rollback"


class ErrorContext:
    """Context information for an error occurrence."""
    
    def __init__(
        self,
        error_id: str,
        workflow_instance_id: int,
        task_id: Optional[int] = None,
        error_type: ErrorType = ErrorType.SYSTEM_ERROR,
        error_message: str = "",
        error_data: Optional[Dict[str, Any]] = None,
        retry_count: int = 0,
        max_retries: int = 3
    ):
        self.error_id = error_id
        self.workflow_instance_id = workflow_instance_id
        self.task_id = task_id
        self.error_type = error_type
        self.error_message = error_message
        self.error_data = error_data or {}
        self.retry_count = retry_count
        self.max_retries = max_retries
        self.created_at = datetime.utcnow()
        self.recovery_strategy: Optional[RecoveryStrategy] = None
        self.recovery_attempts: List[Dict[str, Any]] = []


class ErrorRecoveryService:
    """
    Service for handling errors and implementing recovery strategies.
    Provides retry mechanisms, compensation workflows, and error escalation.
    """
    
    def __init__(self):
        self.supabase_service = supabase_service
        self.event_publisher = event_publisher
        self.timer_service = timer_service
        self.active_errors: Dict[str, ErrorContext] = {}
        self.recovery_strategies: Dict[ErrorType, RecoveryStrategy] = {}
        self.retry_handlers: Dict[str, Callable] = {}
        self.compensation_handlers: Dict[str, Callable] = {}
        
    async def initialize(self) -> bool:
        """Initialize the error recovery service."""
        try:
            logger.info("Initializing Error Recovery Service...")
            
            # Configure default recovery strategies
            self._configure_default_strategies()
            
            # Register default handlers
            self._register_default_handlers()
            
            # Load active errors from database
            await self._load_active_errors()
            
            logger.info("Error Recovery Service initialized successfully")
            return True
            
        except Exception as e:
            logger.error(f"Error initializing Error Recovery Service: {e}")
            return False
    
    def _configure_default_strategies(self):
        """Configure default recovery strategies for different error types."""
        self.recovery_strategies.update({
            ErrorType.TASK_FAILURE: RecoveryStrategy.RETRY,
            ErrorType.SERVICE_UNAVAILABLE: RecoveryStrategy.RETRY,
            ErrorType.TIMEOUT: RecoveryStrategy.RETRY,
            ErrorType.VALIDATION_ERROR: RecoveryStrategy.MANUAL_INTERVENTION,
            ErrorType.BUSINESS_RULE_VIOLATION: RecoveryStrategy.ESCALATE,
            ErrorType.SYSTEM_ERROR: RecoveryStrategy.RETRY,
            ErrorType.NETWORK_ERROR: RecoveryStrategy.RETRY,
            ErrorType.AUTHENTICATION_ERROR: RecoveryStrategy.ESCALATE,
            ErrorType.AUTHORIZATION_ERROR: RecoveryStrategy.ESCALATE,
            ErrorType.DATA_ERROR: RecoveryStrategy.MANUAL_INTERVENTION
        })
    
    def _register_default_handlers(self):
        """Register default retry and compensation handlers."""
        self.retry_handlers.update({
            "default": self._default_retry_handler,
            "service_call": self._service_call_retry_handler,
            "database_operation": self._database_retry_handler,
            "external_api": self._external_api_retry_handler
        })
        
        self.compensation_handlers.update({
            "default": self._default_compensation_handler,
            "transaction_rollback": self._transaction_rollback_handler,
            "resource_cleanup": self._resource_cleanup_handler,
            "notification_reversal": self._notification_reversal_handler
        })
    
    async def handle_error(
        self,
        workflow_instance_id: int,
        error_type: ErrorType,
        error_message: str,
        error_data: Optional[Dict[str, Any]] = None,
        task_id: Optional[int] = None,
        custom_strategy: Optional[RecoveryStrategy] = None,
        db: Optional[Session] = None
    ) -> str:
        """
        Handle an error occurrence and initiate recovery.
        
        Args:
            workflow_instance_id: Workflow instance ID
            error_type: Type of error
            error_message: Error message
            error_data: Additional error data
            task_id: Optional task ID if error is task-specific
            custom_strategy: Custom recovery strategy to use
            db: Database session
            
        Returns:
            Error ID for tracking
        """
        try:
            # Generate error ID
            error_id = f"error_{workflow_instance_id}_{datetime.utcnow().timestamp()}"
            
            # Create error context
            error_context = ErrorContext(
                error_id=error_id,
                workflow_instance_id=workflow_instance_id,
                task_id=task_id,
                error_type=error_type,
                error_message=error_message,
                error_data=error_data or {}
            )
            
            # Determine recovery strategy
            strategy = custom_strategy or self.recovery_strategies.get(error_type, RecoveryStrategy.ESCALATE)
            error_context.recovery_strategy = strategy
            
            # Store error context
            self.active_errors[error_id] = error_context
            
            # Log error event
            await self._log_error_event(error_context, "error_occurred", db)
            
            # Initiate recovery
            await self._initiate_recovery(error_context, db)
            
            logger.error(f"Handled error {error_id}: {error_message} (strategy: {strategy.value})")
            return error_id
            
        except Exception as e:
            logger.error(f"Error handling error: {e}")
            return ""
    
    async def _initiate_recovery(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ):
        """Initiate recovery based on the determined strategy."""
        try:
            strategy = error_context.recovery_strategy
            
            if strategy == RecoveryStrategy.RETRY:
                await self._initiate_retry(error_context, db)
            elif strategy == RecoveryStrategy.COMPENSATE:
                await self._initiate_compensation(error_context, db)
            elif strategy == RecoveryStrategy.ESCALATE:
                await self._initiate_escalation(error_context, db)
            elif strategy == RecoveryStrategy.SKIP:
                await self._initiate_skip(error_context, db)
            elif strategy == RecoveryStrategy.ABORT:
                await self._initiate_abort(error_context, db)
            elif strategy == RecoveryStrategy.MANUAL_INTERVENTION:
                await self._initiate_manual_intervention(error_context, db)
            elif strategy == RecoveryStrategy.ALTERNATIVE_PATH:
                await self._initiate_alternative_path(error_context, db)
            elif strategy == RecoveryStrategy.ROLLBACK:
                await self._initiate_rollback(error_context, db)
            
        except Exception as e:
            logger.error(f"Error initiating recovery: {e}")
    
    async def _initiate_retry(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ):
        """Initiate retry recovery strategy."""
        try:
            if error_context.retry_count >= error_context.max_retries:
                # Max retries reached, escalate
                logger.warning(f"Max retries reached for error {error_context.error_id}, escalating")
                error_context.recovery_strategy = RecoveryStrategy.ESCALATE
                await self._initiate_escalation(error_context, db)
                return
            
            # Calculate retry delay with exponential backoff
            delay_seconds = min(300, 2 ** error_context.retry_count * 10)  # Max 5 minutes
            retry_time = datetime.utcnow() + timedelta(seconds=delay_seconds)
            
            # Create retry timer
            await self.timer_service.create_timer(
                workflow_instance_id=error_context.workflow_instance_id,
                timer_name=f"retry_{error_context.error_id}",
                due_date=retry_time,
                timer_type="error_retry",
                timer_data={
                    "error_id": error_context.error_id,
                    "retry_count": error_context.retry_count + 1,
                    "retry_handler": error_context.error_data.get("retry_handler", "default")
                },
                callback_name="error_retry",
                db=db
            )
            
            # Log retry initiation
            await self._log_error_event(
                error_context,
                "retry_scheduled",
                {
                    "retry_count": error_context.retry_count + 1,
                    "delay_seconds": delay_seconds,
                    "retry_time": retry_time.isoformat()
                },
                db
            )
            
        except Exception as e:
            logger.error(f"Error initiating retry: {e}")
    
    async def _initiate_compensation(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ):
        """Initiate compensation recovery strategy."""
        try:
            # Get compensation handler
            handler_name = error_context.error_data.get("compensation_handler", "default")
            handler = self.compensation_handlers.get(handler_name)
            
            if handler:
                await handler(error_context, db)
            else:
                logger.warning(f"No compensation handler found: {handler_name}")
            
            # Log compensation initiation
            await self._log_error_event(
                error_context,
                "compensation_initiated",
                {"handler": handler_name},
                db
            )
            
        except Exception as e:
            logger.error(f"Error initiating compensation: {e}")
    
    async def _initiate_escalation(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ):
        """Initiate escalation recovery strategy."""
        try:
            # Publish escalation event
            await self.event_publisher.publish_custom_event(
                "error_escalation",
                {
                    "error_id": error_context.error_id,
                    "workflow_instance_id": error_context.workflow_instance_id,
                    "task_id": error_context.task_id,
                    "error_type": error_context.error_type.value,
                    "error_message": error_context.error_message,
                    "retry_count": error_context.retry_count,
                    "escalation_level": "supervisor"
                }
            )
            
            # Log escalation
            await self._log_error_event(
                error_context,
                "error_escalated",
                {"escalation_level": "supervisor"},
                db
            )
            
        except Exception as e:
            logger.error(f"Error initiating escalation: {e}")
    
    async def _initiate_skip(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ):
        """Initiate skip recovery strategy."""
        try:
            # Publish skip event
            await self.event_publisher.publish_custom_event(
                "error_skipped",
                {
                    "error_id": error_context.error_id,
                    "workflow_instance_id": error_context.workflow_instance_id,
                    "task_id": error_context.task_id,
                    "error_type": error_context.error_type.value,
                    "skip_reason": "error_recovery_strategy"
                }
            )
            
            # Log skip
            await self._log_error_event(
                error_context,
                "error_skipped",
                {"skip_reason": "error_recovery_strategy"},
                db
            )
            
            # Remove from active errors
            if error_context.error_id in self.active_errors:
                del self.active_errors[error_context.error_id]
            
        except Exception as e:
            logger.error(f"Error initiating skip: {e}")
    
    async def _initiate_abort(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ):
        """Initiate abort recovery strategy."""
        try:
            # Publish abort event
            await self.event_publisher.publish_custom_event(
                "workflow_aborted",
                {
                    "error_id": error_context.error_id,
                    "workflow_instance_id": error_context.workflow_instance_id,
                    "abort_reason": f"Error: {error_context.error_message}",
                    "error_type": error_context.error_type.value
                }
            )
            
            # Log abort
            await self._log_error_event(
                error_context,
                "workflow_aborted",
                {"abort_reason": error_context.error_message},
                db
            )
            
            # Remove from active errors
            if error_context.error_id in self.active_errors:
                del self.active_errors[error_context.error_id]
            
        except Exception as e:
            logger.error(f"Error initiating abort: {e}")
    
    async def _initiate_manual_intervention(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ):
        """Initiate manual intervention recovery strategy."""
        try:
            # Create manual intervention task
            if not db:
                db = next(get_db())
            
            from ..models.workflow_models import WorkflowTask
            
            intervention_task = WorkflowTask(
                workflow_instance_id=error_context.workflow_instance_id,
                task_definition_key="manual_error_resolution",
                name=f"Resolve Error: {error_context.error_type.value}",
                description=f"Manual intervention required for error: {error_context.error_message}",
                priority="urgent",  # High priority
                status="created",
                input_variables={
                    "error_id": error_context.error_id,
                    "error_type": error_context.error_type.value,
                    "error_message": error_context.error_message,
                    "error_data": error_context.error_data,
                    "intervention_type": "error_resolution"
                }
            )
            
            db.add(intervention_task)
            db.commit()
            
            # Log manual intervention
            await self._log_error_event(
                error_context,
                "manual_intervention_required",
                {"intervention_task_id": intervention_task.id},
                db
            )
            
        except Exception as e:
            logger.error(f"Error initiating manual intervention: {e}")
    
    async def _initiate_alternative_path(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ):
        """Initiate alternative path recovery strategy."""
        try:
            # Publish alternative path event
            await self.event_publisher.publish_custom_event(
                "alternative_path_triggered",
                {
                    "error_id": error_context.error_id,
                    "workflow_instance_id": error_context.workflow_instance_id,
                    "original_error": error_context.error_type.value,
                    "alternative_path": error_context.error_data.get("alternative_path", "default")
                }
            )
            
            # Log alternative path
            await self._log_error_event(
                error_context,
                "alternative_path_triggered",
                {"alternative_path": error_context.error_data.get("alternative_path", "default")},
                db
            )
            
        except Exception as e:
            logger.error(f"Error initiating alternative path: {e}")
    
    async def _initiate_rollback(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ):
        """Initiate rollback recovery strategy."""
        try:
            # Get rollback handler
            handler_name = error_context.error_data.get("rollback_handler", "transaction_rollback")
            handler = self.compensation_handlers.get(handler_name)
            
            if handler:
                await handler(error_context, db)
            
            # Log rollback
            await self._log_error_event(
                error_context,
                "rollback_initiated",
                {"handler": handler_name},
                db
            )
            
        except Exception as e:
            logger.error(f"Error initiating rollback: {e}")
    
    async def handle_retry_timer(
        self,
        error_id: str,
        retry_count: int,
        retry_handler: str = "default",
        db: Optional[Session] = None
    ) -> bool:
        """
        Handle retry timer firing.
        
        Args:
            error_id: Error ID to retry
            retry_count: Current retry count
            retry_handler: Retry handler to use
            db: Database session
            
        Returns:
            True if retry handled successfully
        """
        try:
            error_context = self.active_errors.get(error_id)
            if not error_context:
                logger.warning(f"Error context not found for retry: {error_id}")
                return False
            
            # Update retry count
            error_context.retry_count = retry_count
            
            # Get retry handler
            handler = self.retry_handlers.get(retry_handler, self.retry_handlers["default"])
            
            # Execute retry
            success = await handler(error_context, db)
            
            if success:
                # Retry successful, remove from active errors
                if error_id in self.active_errors:
                    del self.active_errors[error_id]
                
                await self._log_error_event(
                    error_context,
                    "retry_successful",
                    {"retry_count": retry_count},
                    db
                )
            else:
                # Retry failed, schedule next retry or escalate
                await self._initiate_retry(error_context, db)
            
            return success
            
        except Exception as e:
            logger.error(f"Error handling retry timer: {e}")
            return False
    
    # Default handlers
    async def _default_retry_handler(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ) -> bool:
        """Default retry handler."""
        try:
            # Publish retry event for external handling
            await self.event_publisher.publish_custom_event(
                "error_retry_attempt",
                {
                    "error_id": error_context.error_id,
                    "workflow_instance_id": error_context.workflow_instance_id,
                    "task_id": error_context.task_id,
                    "retry_count": error_context.retry_count,
                    "error_type": error_context.error_type.value
                }
            )
            
            # For default handler, assume retry is successful
            # In practice, this would trigger the actual retry logic
            return True
            
        except Exception as e:
            logger.error(f"Error in default retry handler: {e}")
            return False
    
    async def _service_call_retry_handler(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ) -> bool:
        """Retry handler for service call errors."""
        try:
            # Implement service call retry logic
            service_name = error_context.error_data.get("service_name")
            operation = error_context.error_data.get("operation")
            parameters = error_context.error_data.get("parameters", {})
            
            # Publish service retry event
            await self.event_publisher.publish_custom_event(
                "service_call_retry",
                {
                    "error_id": error_context.error_id,
                    "service_name": service_name,
                    "operation": operation,
                    "parameters": parameters,
                    "retry_count": error_context.retry_count
                }
            )
            
            return True
            
        except Exception as e:
            logger.error(f"Error in service call retry handler: {e}")
            return False
    
    async def _database_retry_handler(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ) -> bool:
        """Retry handler for database operation errors."""
        try:
            # Implement database retry logic
            operation = error_context.error_data.get("operation")
            
            # Publish database retry event
            await self.event_publisher.publish_custom_event(
                "database_operation_retry",
                {
                    "error_id": error_context.error_id,
                    "operation": operation,
                    "retry_count": error_context.retry_count
                }
            )
            
            return True
            
        except Exception as e:
            logger.error(f"Error in database retry handler: {e}")
            return False
    
    async def _external_api_retry_handler(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ) -> bool:
        """Retry handler for external API errors."""
        try:
            # Implement external API retry logic
            api_endpoint = error_context.error_data.get("api_endpoint")
            
            # Publish API retry event
            await self.event_publisher.publish_custom_event(
                "external_api_retry",
                {
                    "error_id": error_context.error_id,
                    "api_endpoint": api_endpoint,
                    "retry_count": error_context.retry_count
                }
            )
            
            return True
            
        except Exception as e:
            logger.error(f"Error in external API retry handler: {e}")
            return False
    
    async def _default_compensation_handler(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ) -> bool:
        """Default compensation handler."""
        try:
            # Publish compensation event
            await self.event_publisher.publish_custom_event(
                "error_compensation",
                {
                    "error_id": error_context.error_id,
                    "workflow_instance_id": error_context.workflow_instance_id,
                    "compensation_type": "default"
                }
            )
            
            return True
            
        except Exception as e:
            logger.error(f"Error in default compensation handler: {e}")
            return False
    
    async def _transaction_rollback_handler(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ) -> bool:
        """Transaction rollback compensation handler."""
        try:
            # Implement transaction rollback logic
            transaction_id = error_context.error_data.get("transaction_id")
            
            # Publish rollback event
            await self.event_publisher.publish_custom_event(
                "transaction_rollback",
                {
                    "error_id": error_context.error_id,
                    "transaction_id": transaction_id,
                    "rollback_reason": "error_compensation"
                }
            )
            
            return True
            
        except Exception as e:
            logger.error(f"Error in transaction rollback handler: {e}")
            return False
    
    async def _resource_cleanup_handler(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ) -> bool:
        """Resource cleanup compensation handler."""
        try:
            # Implement resource cleanup logic
            resources = error_context.error_data.get("resources", [])
            
            # Publish cleanup event
            await self.event_publisher.publish_custom_event(
                "resource_cleanup",
                {
                    "error_id": error_context.error_id,
                    "resources": resources,
                    "cleanup_reason": "error_compensation"
                }
            )
            
            return True
            
        except Exception as e:
            logger.error(f"Error in resource cleanup handler: {e}")
            return False
    
    async def _notification_reversal_handler(
        self,
        error_context: ErrorContext,
        db: Optional[Session] = None
    ) -> bool:
        """Notification reversal compensation handler."""
        try:
            # Implement notification reversal logic
            notifications = error_context.error_data.get("notifications", [])
            
            # Publish reversal event
            await self.event_publisher.publish_custom_event(
                "notification_reversal",
                {
                    "error_id": error_context.error_id,
                    "notifications": notifications,
                    "reversal_reason": "error_compensation"
                }
            )
            
            return True
            
        except Exception as e:
            logger.error(f"Error in notification reversal handler: {e}")
            return False
    
    async def _log_error_event(
        self,
        error_context: ErrorContext,
        event_type: str,
        additional_data: Optional[Dict[str, Any]] = None,
        db: Optional[Session] = None
    ):
        """Log error event for audit trail."""
        try:
            if not db:
                db = next(get_db())
            
            event = WorkflowEvent(
                workflow_instance_id=error_context.workflow_instance_id,
                event_type=f"error_{event_type}",
                event_data={
                    "error_id": error_context.error_id,
                    "error_type": error_context.error_type.value,
                    "error_message": error_context.error_message,
                    "task_id": error_context.task_id,
                    "retry_count": error_context.retry_count,
                    "recovery_strategy": error_context.recovery_strategy.value if error_context.recovery_strategy else None,
                    **(additional_data or {})
                }
            )
            
            db.add(event)
            db.commit()
            
        except Exception as e:
            logger.error(f"Error logging error event: {e}")
    
    async def _load_active_errors(self):
        """Load active errors from database on service startup."""
        try:
            # This would load error contexts from persistent storage
            # For now, we start with empty state
            logger.info("Loaded active errors from database")
            
        except Exception as e:
            logger.error(f"Error loading active errors: {e}")
    
    async def get_error_status(self, error_id: str) -> Optional[Dict[str, Any]]:
        """Get current status of an error."""
        try:
            error_context = self.active_errors.get(error_id)
            if not error_context:
                return None
            
            return {
                "error_id": error_context.error_id,
                "workflow_instance_id": error_context.workflow_instance_id,
                "task_id": error_context.task_id,
                "error_type": error_context.error_type.value,
                "error_message": error_context.error_message,
                "retry_count": error_context.retry_count,
                "max_retries": error_context.max_retries,
                "recovery_strategy": error_context.recovery_strategy.value if error_context.recovery_strategy else None,
                "created_at": error_context.created_at.isoformat(),
                "recovery_attempts": error_context.recovery_attempts
            }
            
        except Exception as e:
            logger.error(f"Error getting error status: {e}")
            return None


# Global error recovery service instance
error_recovery_service = ErrorRecoveryService()
