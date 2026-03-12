"""
Clinical Activity Service for Clinical Workflow Engine.
Manages clinical activities with real data validation and error handling.
"""
import logging
from typing import Dict, Any, Optional, List
from datetime import datetime
import asyncio

from app.models.clinical_activity_models import (
    ClinicalActivity, ClinicalActivityType, ClinicalContext, 
    ClinicalError, ClinicalErrorType, DataSourceType,
    ClinicalDataError, CompensationStrategy
)
from app.validation.real_data_validator import real_data_validator
from app.clinical_error_service import clinical_error_handler

logger = logging.getLogger(__name__)


class ClinicalActivityService:
    """
    Service for executing clinical activities with safety and data validation.
    """
    
    def __init__(self):
        self.active_activities = {}
        self.activity_registry = {}
        self.execution_metrics = {}
    
    def register_activity(self, activity: ClinicalActivity):
        """
        Register a clinical activity definition.
        """
        self.activity_registry[activity.activity_id] = activity
        logger.info(f"Registered clinical activity: {activity.activity_id}")
    
    async def execute_activity(
        self,
        activity_id: str,
        context: ClinicalContext,
        workflow_instance_id: str,
        input_data: Dict[str, Any] = None
    ) -> Dict[str, Any]:
        """
        Execute a clinical activity with full validation and error handling.
        
        Args:
            activity_id: ID of the activity to execute
            context: Clinical context for execution
            workflow_instance_id: ID of the workflow instance
            input_data: Input data for the activity
            
        Returns:
            Dict containing execution results
            
        Raises:
            ClinicalDataError: If data validation fails
        """
        if input_data is None:
            input_data = {}
        
        # Get activity definition
        activity = self.activity_registry.get(activity_id)
        if not activity:
            raise ValueError(f"Activity not registered: {activity_id}")
        
        execution_id = f"{workflow_instance_id}_{activity_id}_{datetime.utcnow().timestamp()}"
        
        try:
            # Start activity execution tracking
            self.active_activities[execution_id] = {
                'activity': activity,
                'context': context,
                'workflow_instance_id': workflow_instance_id,
                'started_at': datetime.utcnow(),
                'status': 'running'
            }
            
            logger.info(f"Starting clinical activity: {activity_id} (execution: {execution_id})")
            
            # Validate input data if required
            if activity.real_data_only:
                await self._validate_activity_data(activity, input_data, context)
            
            # Execute based on activity type
            if activity.activity_type == ClinicalActivityType.SYNCHRONOUS:
                result = await self._execute_sync_activity(activity, context, input_data)
            elif activity.activity_type == ClinicalActivityType.ASYNCHRONOUS:
                result = await self._execute_async_activity(activity, context, input_data)
            elif activity.activity_type == ClinicalActivityType.HUMAN:
                result = await self._execute_human_activity(activity, context, input_data)
            else:
                raise ValueError(f"Unknown activity type: {activity.activity_type}")
            
            # Update execution status
            self.active_activities[execution_id]['status'] = 'completed'
            self.active_activities[execution_id]['completed_at'] = datetime.utcnow()
            self.active_activities[execution_id]['result'] = result
            
            # Record metrics
            await self._record_execution_metrics(activity, execution_id, True)
            
            logger.info(f"Completed clinical activity: {activity_id} (execution: {execution_id})")
            
            return result
            
        except Exception as e:
            # Handle activity execution error
            error = ClinicalError(
                error_id=f"error_{execution_id}",
                error_type=self._classify_error(e),
                error_message=str(e),
                activity_id=activity_id,
                workflow_instance_id=workflow_instance_id,
                clinical_context=context,
                error_data={'execution_id': execution_id, 'input_data': input_data}
            )
            
            # Update execution status
            if execution_id in self.active_activities:
                self.active_activities[execution_id]['status'] = 'failed'
                self.active_activities[execution_id]['error'] = error
            
            # Handle the error
            compensation_strategy = await clinical_error_handler.handle_clinical_error(
                error, context, workflow_instance_id
            )
            
            # Record metrics
            await self._record_execution_metrics(activity, execution_id, False)
            
            # Re-raise if immediate failure required
            if compensation_strategy == CompensationStrategy.IMMEDIATE_FAILURE:
                raise
            
            # Return error result for other strategies
            return {
                'status': 'error',
                'error': error,
                'compensation_strategy': compensation_strategy.value
            }
        
        finally:
            # Clean up active activity tracking
            if execution_id in self.active_activities:
                # Move to completed activities (keep for audit)
                completed_activity = self.active_activities.pop(execution_id)
                # TODO: Store in permanent audit log
    
    async def _validate_activity_data(
        self,
        activity: ClinicalActivity,
        input_data: Dict[str, Any],
        context: ClinicalContext
    ):
        """
        Validate that activity data comes from approved real sources.
        """
        # Validate input data sources
        for key, value in input_data.items():
            if isinstance(value, dict) and 'source_type' in value:
                source_type_str = value['source_type']
                try:
                    source_type = DataSourceType(source_type_str)
                    
                    # Check if source is approved for this activity
                    if (activity.approved_data_sources and 
                        source_type not in activity.approved_data_sources):
                        raise ClinicalDataError(
                            f"Data source {source_type.value} not approved for activity {activity.activity_id}"
                        )
                    
                    # Validate the data
                    await real_data_validator.validate_data_source(
                        source_type,
                        value.get('data'),
                        value.get('metadata', {})
                    )
                    
                except ValueError:
                    raise ClinicalDataError(f"Invalid data source type: {source_type_str}")
        
        # Validate clinical context data sources
        if context.data_sources:
            for source_name, endpoint in context.data_sources.items():
                try:
                    source_type = DataSourceType(source_name)
                    metadata = {
                        'source_endpoint': endpoint,
                        'retrieved_at': context.created_at.isoformat()
                    }
                    
                    # Validate context data
                    context_data = context.clinical_data.get(source_name)
                    if context_data:
                        await real_data_validator.validate_data_source(
                            source_type, context_data, metadata
                        )
                        
                except ValueError:
                    logger.warning(f"Unknown data source in context: {source_name}")
    
    async def _execute_sync_activity(
        self,
        activity: ClinicalActivity,
        context: ClinicalContext,
        input_data: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Execute synchronous activity (< 1 second).
        """
        start_time = datetime.utcnow()
        
        try:
            # Apply timeout
            result = await asyncio.wait_for(
                self._call_activity_handler(activity, context, input_data),
                timeout=activity.timeout_seconds
            )
            
            execution_time = (datetime.utcnow() - start_time).total_seconds()
            
            return {
                'status': 'success',
                'result': result,
                'execution_time_seconds': execution_time,
                'activity_type': 'synchronous'
            }
            
        except asyncio.TimeoutError:
            execution_time = (datetime.utcnow() - start_time).total_seconds()
            raise ClinicalDataError(
                f"Synchronous activity {activity.activity_id} timed out after {execution_time}s"
            )
    
    async def _execute_async_activity(
        self,
        activity: ClinicalActivity,
        context: ClinicalContext,
        input_data: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Execute asynchronous activity (1-30 seconds).
        """
        start_time = datetime.utcnow()
        
        try:
            # Apply timeout
            result = await asyncio.wait_for(
                self._call_activity_handler(activity, context, input_data),
                timeout=activity.timeout_seconds
            )
            
            execution_time = (datetime.utcnow() - start_time).total_seconds()
            
            return {
                'status': 'success',
                'result': result,
                'execution_time_seconds': execution_time,
                'activity_type': 'asynchronous'
            }
            
        except asyncio.TimeoutError:
            execution_time = (datetime.utcnow() - start_time).total_seconds()
            raise ClinicalDataError(
                f"Asynchronous activity {activity.activity_id} timed out after {execution_time}s"
            )
    
    async def _execute_human_activity(
        self,
        activity: ClinicalActivity,
        context: ClinicalContext,
        input_data: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Execute human activity (minutes-hours).
        """
        # Human activities are typically handled by creating tasks
        # and waiting for human completion
        
        task_data = {
            'activity_id': activity.activity_id,
            'context': context,
            'input_data': input_data,
            'timeout_seconds': activity.timeout_seconds,
            'created_at': datetime.utcnow().isoformat()
        }
        
        # TODO: Create human task in workflow engine
        # task_id = await task_service.create_human_task(task_data)
        
        return {
            'status': 'pending',
            'activity_type': 'human',
            'message': f'Human task created for activity {activity.activity_id}',
            'task_data': task_data
        }
    
    async def _call_activity_handler(
        self,
        activity: ClinicalActivity,
        context: ClinicalContext,
        input_data: Dict[str, Any]
    ) -> Any:
        """
        Call the actual activity handler (placeholder for now).
        """
        # TODO: Implement actual activity handlers based on activity type
        # This would call the appropriate service (harmonization, safety gateway, etc.)
        
        logger.info(f"Executing activity handler for {activity.activity_id}")
        
        # Simulate activity execution
        await asyncio.sleep(0.1)  # Simulate processing time
        
        return {
            'activity_id': activity.activity_id,
            'processed_at': datetime.utcnow().isoformat(),
            'input_data': input_data,
            'context_patient_id': context.patient_id
        }
    
    def _classify_error(self, error: Exception) -> ClinicalErrorType:
        """
        Classify an exception into a clinical error type.
        """
        if isinstance(error, ClinicalDataError):
            return error.error_type
        elif isinstance(error, asyncio.TimeoutError):
            return ClinicalErrorType.TECHNICAL_ERROR
        elif isinstance(error, ConnectionError):
            return ClinicalErrorType.TECHNICAL_ERROR
        else:
            return ClinicalErrorType.TECHNICAL_ERROR
    
    async def _record_execution_metrics(
        self,
        activity: ClinicalActivity,
        execution_id: str,
        success: bool
    ):
        """
        Record execution metrics for monitoring.
        """
        if activity.activity_id not in self.execution_metrics:
            self.execution_metrics[activity.activity_id] = {
                'total_executions': 0,
                'successful_executions': 0,
                'failed_executions': 0,
                'average_execution_time': 0.0
            }
        
        metrics = self.execution_metrics[activity.activity_id]
        metrics['total_executions'] += 1
        
        if success:
            metrics['successful_executions'] += 1
        else:
            metrics['failed_executions'] += 1
        
        # TODO: Calculate and update average execution time
        
        logger.debug(f"Updated metrics for activity {activity.activity_id}: {metrics}")
    
    def get_activity_metrics(self, activity_id: Optional[str] = None) -> Dict[str, Any]:
        """
        Get execution metrics for activities.
        """
        if activity_id:
            return self.execution_metrics.get(activity_id, {})
        else:
            return self.execution_metrics.copy()
    
    def get_active_activities(self) -> Dict[str, Any]:
        """
        Get currently active activities.
        """
        return self.active_activities.copy()


# Global clinical activity service instance
clinical_activity_service = ClinicalActivityService()
