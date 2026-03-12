"""
Clinical Error Handling Service for Clinical Workflow Engine.
Handles clinical errors with strict real data requirements.
"""
import logging
from typing import Dict, Any, Optional
from datetime import datetime
import uuid

from app.models.clinical_activity_models import (
    ClinicalError, ClinicalErrorType, ClinicalContext, 
    CompensationStrategy, ClinicalDataError
)

logger = logging.getLogger(__name__)


class ClinicalErrorHandler:
    """
    Handles clinical errors with strict real data requirements.
    Mock data errors and unapproved sources cause immediate failure.
    """
    
    def __init__(self):
        self.error_history = {}
        self.compensation_handlers = {}
        self._register_compensation_handlers()
    
    def _register_compensation_handlers(self):
        """
        Register compensation handlers for different error types.
        """
        self.compensation_handlers = {
            ClinicalErrorType.SAFETY_ERROR: self._handle_safety_error,
            ClinicalErrorType.WARNING_ERROR: self._handle_warning_error,
            ClinicalErrorType.TECHNICAL_ERROR: self._handle_technical_error,
            ClinicalErrorType.DATA_SOURCE_ERROR: self._handle_data_source_error,
            ClinicalErrorType.MOCK_DATA_ERROR: self._handle_mock_data_error
        }
    
    async def handle_clinical_error(
        self,
        error: ClinicalError,
        context: ClinicalContext,
        workflow_instance_id: str
    ) -> CompensationStrategy:
        """
        Handle clinical errors with strict real data requirements.
        Mock data errors and unapproved sources cause immediate failure.
        
        Args:
            error: The clinical error to handle
            context: Clinical context for the error
            workflow_instance_id: ID of the workflow instance
            
        Returns:
            CompensationStrategy: Strategy to use for error recovery
        """
        try:
            # Log the error
            await self._log_clinical_error(error, context, workflow_instance_id)
            
            # Store error in history
            self.error_history[error.error_id] = {
                'error': error,
                'context': context,
                'workflow_instance_id': workflow_instance_id,
                'handled_at': datetime.utcnow()
            }
            
            # Handle based on error type
            handler = self.compensation_handlers.get(error.error_type)
            if handler:
                strategy = await handler(error, context, workflow_instance_id)
            else:
                logger.error(f"No handler found for error type: {error.error_type}")
                strategy = CompensationStrategy.IMMEDIATE_FAILURE
            
            # Log the compensation strategy
            logger.info(f"Compensation strategy for error {error.error_id}: {strategy.value}")
            
            return strategy
            
        except Exception as e:
            logger.error(f"Error handling clinical error {error.error_id}: {e}")
            # Default to immediate failure for safety
            return CompensationStrategy.IMMEDIATE_FAILURE
    
    async def _handle_safety_error(
        self,
        error: ClinicalError,
        context: ClinicalContext,
        workflow_instance_id: str
    ) -> CompensationStrategy:
        """
        Handle safety-critical errors (e.g., drug interactions).
        Always requires full compensation to ensure patient safety.
        """
        logger.critical(f"Safety error detected: {error.error_message}")
        
        # Safety errors require immediate workflow termination
        await self._fail_workflow_immediately(workflow_instance_id, error)
        
        # Full compensation to reverse all activities
        return CompensationStrategy.FULL_COMPENSATION
    
    async def _handle_warning_error(
        self,
        error: ClinicalError,
        context: ClinicalContext,
        workflow_instance_id: str
    ) -> CompensationStrategy:
        """
        Handle warning errors that can be overridden by clinicians.
        """
        logger.warning(f"Warning error detected: {error.error_message}")
        
        # Warning errors can continue with clinical override
        # This would typically trigger a human task for review
        await self._create_override_task(workflow_instance_id, error, context)
        
        # Partial compensation - only reverse the failed activity
        return CompensationStrategy.PARTIAL_COMPENSATION
    
    async def _handle_technical_error(
        self,
        error: ClinicalError,
        context: ClinicalContext,
        workflow_instance_id: str
    ) -> CompensationStrategy:
        """
        Handle technical errors (network timeouts, service unavailable).
        """
        logger.error(f"Technical error detected: {error.error_message}")
        
        # Technical errors can be retried with exponential backoff
        retry_count = error.error_data.get('retry_count', 0)
        max_retries = error.error_data.get('max_retries', 3)
        
        if retry_count < max_retries:
            # Schedule retry with exponential backoff
            await self._schedule_retry(workflow_instance_id, error, retry_count + 1)
            return CompensationStrategy.FORWARD_RECOVERY
        else:
            # Max retries exceeded, fail the workflow
            await self._fail_workflow_immediately(workflow_instance_id, error)
            return CompensationStrategy.FULL_COMPENSATION
    
    async def _handle_data_source_error(
        self,
        error: ClinicalError,
        context: ClinicalContext,
        workflow_instance_id: str
    ) -> CompensationStrategy:
        """
        Handle data source errors (real data unavailable).
        Immediate workflow failure for data integrity issues.
        """
        logger.critical(f"Data source error detected: {error.error_message}")
        
        # Data source errors cause immediate failure
        await self._fail_workflow_immediately(workflow_instance_id, error)
        
        # No compensation possible - data integrity compromised
        return CompensationStrategy.IMMEDIATE_FAILURE
    
    async def _handle_mock_data_error(
        self,
        error: ClinicalError,
        context: ClinicalContext,
        workflow_instance_id: str
    ) -> CompensationStrategy:
        """
        Handle mock data detection errors.
        Immediate workflow failure - no mock data allowed in clinical workflows.
        """
        logger.critical(f"Mock data detected: {error.error_message}")
        
        # Mock data detection causes immediate failure
        await self._fail_workflow_immediately(workflow_instance_id, error)
        
        # No compensation possible - mock data is not acceptable
        return CompensationStrategy.IMMEDIATE_FAILURE
    
    async def _log_clinical_error(
        self,
        error: ClinicalError,
        context: ClinicalContext,
        workflow_instance_id: str
    ):
        """
        Log clinical error for audit trail and monitoring.
        """
        log_entry = {
            'error_id': error.error_id,
            'error_type': error.error_type.value,
            'error_message': error.error_message,
            'activity_id': error.activity_id,
            'workflow_instance_id': workflow_instance_id,
            'patient_id': context.patient_id if context else None,
            'provider_id': context.provider_id if context else None,
            'timestamp': error.created_at.isoformat(),
            'error_data': error.error_data
        }
        
        # Log to structured logging system
        logger.error(f"Clinical error logged: {log_entry}")
        
        # TODO: Send to audit service for permanent storage
        # await audit_service.log_clinical_error(log_entry)
    
    async def _fail_workflow_immediately(
        self,
        workflow_instance_id: str,
        error: ClinicalError
    ):
        """
        Immediately fail the workflow due to critical error.
        """
        logger.critical(f"Failing workflow {workflow_instance_id} due to error: {error.error_message}")
        
        # TODO: Integrate with workflow engine to terminate workflow
        # await workflow_engine_service.terminate_workflow(
        #     workflow_instance_id, 
        #     reason=f"Critical error: {error.error_message}"
        # )
    
    async def _create_override_task(
        self,
        workflow_instance_id: str,
        error: ClinicalError,
        context: ClinicalContext
    ):
        """
        Create a human task for clinical override review.
        """
        logger.info(f"Creating override task for workflow {workflow_instance_id}")
        
        override_task = {
            'workflow_instance_id': workflow_instance_id,
            'error_id': error.error_id,
            'task_type': 'clinical_override',
            'assignee': context.provider_id if context else None,
            'error_details': {
                'error_type': error.error_type.value,
                'error_message': error.error_message,
                'activity_id': error.activity_id
            },
            'created_at': datetime.utcnow().isoformat()
        }
        
        # TODO: Create actual task in workflow engine
        # await task_service.create_override_task(override_task)
    
    async def _schedule_retry(
        self,
        workflow_instance_id: str,
        error: ClinicalError,
        retry_count: int
    ):
        """
        Schedule retry for technical errors with exponential backoff.
        """
        # Calculate backoff delay (exponential: 2^retry_count seconds)
        delay_seconds = 2 ** retry_count
        
        logger.info(f"Scheduling retry {retry_count} for workflow {workflow_instance_id} in {delay_seconds}s")
        
        retry_data = {
            'workflow_instance_id': workflow_instance_id,
            'error_id': error.error_id,
            'retry_count': retry_count,
            'delay_seconds': delay_seconds,
            'scheduled_at': datetime.utcnow().isoformat()
        }
        
        # TODO: Schedule retry with timer service
        # await timer_service.schedule_retry(retry_data)
    
    async def get_error_history(
        self,
        workflow_instance_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Get error history for a workflow or all workflows.
        """
        if workflow_instance_id:
            return {
                error_id: data for error_id, data in self.error_history.items()
                if data['workflow_instance_id'] == workflow_instance_id
            }
        else:
            return self.error_history.copy()
    
    async def clear_error_history(self, workflow_instance_id: str):
        """
        Clear error history for a completed workflow.
        """
        to_remove = [
            error_id for error_id, data in self.error_history.items()
            if data['workflow_instance_id'] == workflow_instance_id
        ]
        
        for error_id in to_remove:
            del self.error_history[error_id]
        
        logger.info(f"Cleared {len(to_remove)} errors for workflow {workflow_instance_id}")


# Global error handler instance
clinical_error_handler = ClinicalErrorHandler()
