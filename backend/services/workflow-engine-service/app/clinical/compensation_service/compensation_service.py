"""
Clinical Compensation Service for Clinical Workflow Engine.
Implements clinical-specific compensation patterns for workflow failures with safety mechanisms.
"""
import logging
from typing import Dict, List, Optional, Any
from datetime import datetime, timedelta
import asyncio
import uuid

from app.models.clinical_activity_models import (
    ClinicalContext, ClinicalError, ClinicalErrorType, CompensationStrategy
)

logger = logging.getLogger(__name__)


class ClinicalCompensationService:
    """
    Service for executing clinical compensation patterns when workflows fail.
    Implements safety-first compensation with real data requirements.
    """
    
    def __init__(self):
        self.compensation_history = {}
        self.active_compensations = {}
        self.compensation_handlers = {}
        self._register_compensation_handlers()
    
    def _register_compensation_handlers(self):
        """
        Register compensation handlers for different clinical scenarios.
        """
        self.compensation_handlers = {
            CompensationStrategy.FULL_COMPENSATION: self._execute_full_compensation,
            CompensationStrategy.PARTIAL_COMPENSATION: self._execute_partial_compensation,
            CompensationStrategy.FORWARD_RECOVERY: self._execute_forward_recovery,
            CompensationStrategy.IMMEDIATE_FAILURE: self._execute_immediate_failure
        }
    
    async def execute_compensation(
        self,
        strategy: CompensationStrategy,
        workflow_instance_id: str,
        failed_activity_id: str,
        clinical_context: ClinicalContext,
        error_details: Optional[Dict[str, Any]] = None
    ) -> bool:
        """
        Execute clinical compensation based on the specified strategy.
        
        Args:
            strategy: Compensation strategy to execute
            workflow_instance_id: ID of the failed workflow instance
            failed_activity_id: ID of the activity that failed
            clinical_context: Clinical context for the compensation
            error_details: Additional error information
            
        Returns:
            bool: True if compensation successful, False otherwise
        """
        compensation_id = str(uuid.uuid4())
        
        try:
            logger.info(f"🔄 Starting clinical compensation: {compensation_id}")
            logger.info(f"   Strategy: {strategy.value}")
            logger.info(f"   Workflow: {workflow_instance_id}")
            logger.info(f"   Failed Activity: {failed_activity_id}")
            logger.info(f"   Patient: {clinical_context.patient_id}")
            
            # Record compensation start
            compensation_record = {
                "compensation_id": compensation_id,
                "strategy": strategy.value,
                "workflow_instance_id": workflow_instance_id,
                "failed_activity_id": failed_activity_id,
                "patient_id": clinical_context.patient_id,
                "started_at": datetime.utcnow(),
                "status": "running",
                "error_details": error_details or {},
                "compensation_steps": []
            }
            
            self.active_compensations[compensation_id] = compensation_record
            
            # Validate clinical context before compensation
            if not await self._validate_clinical_context_for_compensation(clinical_context):
                raise ValueError("Clinical context validation failed - cannot proceed with compensation")
            
            # Execute compensation strategy
            handler = self.compensation_handlers.get(strategy)
            if not handler:
                raise ValueError(f"No handler found for compensation strategy: {strategy.value}")
            
            success = await handler(
                compensation_id,
                workflow_instance_id,
                failed_activity_id,
                clinical_context,
                error_details or {}
            )
            
            # Update compensation record
            compensation_record["status"] = "completed" if success else "failed"
            compensation_record["completed_at"] = datetime.utcnow()
            compensation_record["success"] = success
            
            # Move to history
            self.compensation_history[compensation_id] = compensation_record
            del self.active_compensations[compensation_id]
            
            # Log completion
            if success:
                logger.info(f"✅ Clinical compensation completed successfully: {compensation_id}")
            else:
                logger.error(f"❌ Clinical compensation failed: {compensation_id}")
            
            return success
            
        except Exception as e:
            logger.error(f"❌ Clinical compensation error {compensation_id}: {e}")
            
            # Update compensation record with error
            if compensation_id in self.active_compensations:
                self.active_compensations[compensation_id]["status"] = "error"
                self.active_compensations[compensation_id]["error"] = str(e)
                self.active_compensations[compensation_id]["completed_at"] = datetime.utcnow()
                
                # Move to history
                self.compensation_history[compensation_id] = self.active_compensations[compensation_id]
                del self.active_compensations[compensation_id]
            
            return False
    
    async def _validate_clinical_context_for_compensation(
        self,
        clinical_context: ClinicalContext
    ) -> bool:
        """
        Validate that clinical context is sufficient for safe compensation.
        """
        try:
            # Check required fields
            if not clinical_context.patient_id:
                logger.error("Missing patient_id in clinical context")
                return False
            
            # Validate clinical data is present and real
            if not clinical_context.clinical_data:
                logger.error("No clinical data available for compensation")
                return False
            
            # Check data sources are approved
            if not clinical_context.data_sources:
                logger.error("No data sources specified in clinical context")
                return False
            
            # Validate data freshness (no stale data for compensation)
            context_age = datetime.utcnow() - clinical_context.created_at
            if context_age > timedelta(minutes=30):
                logger.error(f"Clinical context too old for compensation: {context_age}")
                return False
            
            logger.info("✅ Clinical context validated for compensation")
            return True
            
        except Exception as e:
            logger.error(f"Clinical context validation error: {e}")
            return False
    
    async def _execute_full_compensation(
        self,
        compensation_id: str,
        workflow_instance_id: str,
        failed_activity_id: str,
        clinical_context: ClinicalContext,
        error_details: Dict[str, Any]
    ) -> bool:
        """
        Execute full compensation - reverse all activities in the workflow.
        Used for safety-critical failures where patient safety is at risk.
        """
        try:
            logger.info(f"🔄 Executing FULL compensation for workflow {workflow_instance_id}")
            
            compensation_steps = []
            
            # Step 1: Stop all active workflow activities
            step_result = await self._stop_all_workflow_activities(workflow_instance_id)
            compensation_steps.append({
                "step": "stop_all_activities",
                "success": step_result,
                "timestamp": datetime.utcnow().isoformat()
            })
            
            if not step_result:
                logger.error("Failed to stop workflow activities")
                return False
            
            # Step 2: Reverse all completed activities in reverse order
            step_result = await self._reverse_completed_activities(
                workflow_instance_id, clinical_context
            )
            compensation_steps.append({
                "step": "reverse_completed_activities",
                "success": step_result,
                "timestamp": datetime.utcnow().isoformat()
            })
            
            if not step_result:
                logger.error("Failed to reverse completed activities")
                return False
            
            # Step 3: Create safety incident report
            step_result = await self._create_safety_incident_report(
                workflow_instance_id, failed_activity_id, clinical_context, error_details
            )
            compensation_steps.append({
                "step": "create_safety_incident_report",
                "success": step_result,
                "timestamp": datetime.utcnow().isoformat()
            })
            
            # Step 4: Notify clinical team
            step_result = await self._notify_clinical_team_of_compensation(
                workflow_instance_id, "FULL_COMPENSATION", clinical_context
            )
            compensation_steps.append({
                "step": "notify_clinical_team",
                "success": step_result,
                "timestamp": datetime.utcnow().isoformat()
            })
            
            # Step 5: Update patient record with compensation details
            step_result = await self._update_patient_record_with_compensation(
                clinical_context.patient_id, compensation_id, "FULL_COMPENSATION"
            )
            compensation_steps.append({
                "step": "update_patient_record",
                "success": step_result,
                "timestamp": datetime.utcnow().isoformat()
            })
            
            # Update compensation record
            self.active_compensations[compensation_id]["compensation_steps"] = compensation_steps
            
            # All steps must succeed for full compensation
            all_successful = all(step["success"] for step in compensation_steps)
            
            if all_successful:
                logger.info("✅ Full compensation completed successfully")
            else:
                logger.error("❌ Some full compensation steps failed")
            
            return all_successful
            
        except Exception as e:
            logger.error(f"Full compensation execution error: {e}")
            return False
    
    async def _execute_partial_compensation(
        self,
        compensation_id: str,
        workflow_instance_id: str,
        failed_activity_id: str,
        clinical_context: ClinicalContext,
        error_details: Dict[str, Any]
    ) -> bool:
        """
        Execute partial compensation - reverse only the failed branch.
        Used for non-critical failures where only specific activities need reversal.
        """
        try:
            logger.info(f"🔄 Executing PARTIAL compensation for activity {failed_activity_id}")
            
            compensation_steps = []
            
            # Step 1: Identify activities in the failed branch
            failed_branch_activities = await self._identify_failed_branch_activities(
                workflow_instance_id, failed_activity_id
            )
            compensation_steps.append({
                "step": "identify_failed_branch",
                "activities_found": len(failed_branch_activities),
                "timestamp": datetime.utcnow().isoformat()
            })
            
            # Step 2: Reverse activities in the failed branch only
            step_result = await self._reverse_branch_activities(
                failed_branch_activities, clinical_context
            )
            compensation_steps.append({
                "step": "reverse_branch_activities",
                "success": step_result,
                "timestamp": datetime.utcnow().isoformat()
            })
            
            # Step 3: Create incident log (not safety-critical)
            step_result = await self._create_incident_log(
                workflow_instance_id, failed_activity_id, clinical_context, error_details
            )
            compensation_steps.append({
                "step": "create_incident_log",
                "success": step_result,
                "timestamp": datetime.utcnow().isoformat()
            })
            
            # Step 4: Notify relevant team members
            step_result = await self._notify_clinical_team_of_compensation(
                workflow_instance_id, "PARTIAL_COMPENSATION", clinical_context
            )
            compensation_steps.append({
                "step": "notify_clinical_team",
                "success": step_result,
                "timestamp": datetime.utcnow().isoformat()
            })
            
            # Update compensation record
            self.active_compensations[compensation_id]["compensation_steps"] = compensation_steps
            
            # Partial compensation succeeds if critical steps succeed
            critical_steps_successful = compensation_steps[1]["success"]  # reverse_branch_activities
            
            if critical_steps_successful:
                logger.info("✅ Partial compensation completed successfully")
            else:
                logger.error("❌ Critical partial compensation steps failed")
            
            return critical_steps_successful
            
        except Exception as e:
            logger.error(f"Partial compensation execution error: {e}")
            return False
    
    async def _execute_forward_recovery(
        self,
        compensation_id: str,
        workflow_instance_id: str,
        failed_activity_id: str,
        clinical_context: ClinicalContext,
        error_details: Dict[str, Any]
    ) -> bool:
        """
        Execute forward recovery - retry with exponential backoff.
        Used for technical failures that may be transient.
        """
        try:
            logger.info(f"🔄 Executing FORWARD RECOVERY for activity {failed_activity_id}")
            
            compensation_steps = []
            
            # Get retry configuration
            retry_count = error_details.get("retry_count", 0)
            max_retries = error_details.get("max_retries", 3)
            base_delay = error_details.get("base_delay_seconds", 2)
            
            if retry_count >= max_retries:
                logger.error(f"Max retries ({max_retries}) exceeded for activity {failed_activity_id}")
                return False
            
            # Calculate exponential backoff delay
            delay_seconds = base_delay * (2 ** retry_count)
            
            # Step 1: Wait for backoff period
            logger.info(f"Waiting {delay_seconds} seconds before retry {retry_count + 1}")
            await asyncio.sleep(delay_seconds)
            
            compensation_steps.append({
                "step": "exponential_backoff",
                "delay_seconds": delay_seconds,
                "retry_attempt": retry_count + 1,
                "timestamp": datetime.utcnow().isoformat()
            })
            
            # Step 2: Validate clinical context is still valid
            context_valid = await self._validate_clinical_context_for_compensation(clinical_context)
            compensation_steps.append({
                "step": "validate_context",
                "success": context_valid,
                "timestamp": datetime.utcnow().isoformat()
            })
            
            if not context_valid:
                logger.error("Clinical context no longer valid for retry")
                return False
            
            # Step 3: Schedule retry of the failed activity
            step_result = await self._schedule_activity_retry(
                workflow_instance_id, failed_activity_id, clinical_context, retry_count + 1
            )
            compensation_steps.append({
                "step": "schedule_retry",
                "success": step_result,
                "timestamp": datetime.utcnow().isoformat()
            })
            
            # Step 4: Log retry attempt
            step_result = await self._log_retry_attempt(
                workflow_instance_id, failed_activity_id, retry_count + 1, clinical_context
            )
            compensation_steps.append({
                "step": "log_retry_attempt",
                "success": step_result,
                "timestamp": datetime.utcnow().isoformat()
            })
            
            # Update compensation record
            self.active_compensations[compensation_id]["compensation_steps"] = compensation_steps
            
            logger.info("✅ Forward recovery scheduled successfully")
            return True
            
        except Exception as e:
            logger.error(f"Forward recovery execution error: {e}")
            return False
    
    async def _execute_immediate_failure(
        self,
        compensation_id: str,
        workflow_instance_id: str,
        failed_activity_id: str,
        clinical_context: ClinicalContext,
        error_details: Dict[str, Any]
    ) -> bool:
        """
        Execute immediate failure - no compensation, fail immediately.
        Used for data integrity issues where compensation is not possible.
        """
        try:
            logger.critical(f"🚨 Executing IMMEDIATE FAILURE for workflow {workflow_instance_id}")
            
            compensation_steps = []
            
            # Step 1: Mark workflow as failed immediately
            step_result = await self._mark_workflow_failed(
                workflow_instance_id, failed_activity_id, error_details
            )
            compensation_steps.append({
                "step": "mark_workflow_failed",
                "success": step_result,
                "timestamp": datetime.utcnow().isoformat()
            })
            
            # Step 2: Create critical incident report
            step_result = await self._create_critical_incident_report(
                workflow_instance_id, failed_activity_id, clinical_context, error_details
            )
            compensation_steps.append({
                "step": "create_critical_incident_report",
                "success": step_result,
                "timestamp": datetime.utcnow().isoformat()
            })
            
            # Step 3: Immediate escalation to clinical supervisor
            step_result = await self._escalate_to_clinical_supervisor(
                workflow_instance_id, clinical_context, error_details
            )
            compensation_steps.append({
                "step": "escalate_to_supervisor",
                "success": step_result,
                "timestamp": datetime.utcnow().isoformat()
            })
            
            # Update compensation record
            self.active_compensations[compensation_id]["compensation_steps"] = compensation_steps
            
            logger.critical("🚨 Immediate failure processing completed")
            return True  # Immediate failure always "succeeds" in failing fast
            
        except Exception as e:
            logger.error(f"Immediate failure execution error: {e}")
            return False
    
    # Helper methods for compensation steps
    async def _stop_all_workflow_activities(self, workflow_instance_id: str) -> bool:
        """Stop all active activities in the workflow."""
        try:
            logger.info(f"Stopping all activities for workflow {workflow_instance_id}")
            # TODO: Integrate with workflow engine to stop activities
            await asyncio.sleep(0.1)  # Simulate processing
            return True
        except Exception as e:
            logger.error(f"Error stopping workflow activities: {e}")
            return False
    
    async def _reverse_completed_activities(
        self, workflow_instance_id: str, clinical_context: ClinicalContext
    ) -> bool:
        """Reverse all completed activities in the workflow."""
        try:
            logger.info(f"Reversing completed activities for workflow {workflow_instance_id}")
            # TODO: Implement activity reversal logic
            await asyncio.sleep(0.1)  # Simulate processing
            return True
        except Exception as e:
            logger.error(f"Error reversing activities: {e}")
            return False
    
    async def _create_safety_incident_report(
        self, workflow_instance_id: str, failed_activity_id: str, 
        clinical_context: ClinicalContext, error_details: Dict[str, Any]
    ) -> bool:
        """Create a safety incident report."""
        try:
            logger.info(f"Creating safety incident report for workflow {workflow_instance_id}")
            # TODO: Integrate with incident reporting system
            await asyncio.sleep(0.1)  # Simulate processing
            return True
        except Exception as e:
            logger.error(f"Error creating safety incident report: {e}")
            return False
    
    async def _notify_clinical_team_of_compensation(
        self, workflow_instance_id: str, compensation_type: str, clinical_context: ClinicalContext
    ) -> bool:
        """Notify clinical team of compensation execution."""
        try:
            logger.info(f"Notifying clinical team of {compensation_type} for workflow {workflow_instance_id}")
            # TODO: Integrate with notification system
            await asyncio.sleep(0.1)  # Simulate processing
            return True
        except Exception as e:
            logger.error(f"Error notifying clinical team: {e}")
            return False
    
    async def _update_patient_record_with_compensation(
        self, patient_id: str, compensation_id: str, compensation_type: str
    ) -> bool:
        """Update patient record with compensation details."""
        try:
            logger.info(f"Updating patient record {patient_id} with compensation {compensation_id}")
            # TODO: Integrate with patient record system
            await asyncio.sleep(0.1)  # Simulate processing
            return True
        except Exception as e:
            logger.error(f"Error updating patient record: {e}")
            return False
    
    async def _identify_failed_branch_activities(
        self, workflow_instance_id: str, failed_activity_id: str
    ) -> List[str]:
        """Identify activities in the failed branch."""
        try:
            logger.info(f"Identifying failed branch activities for {failed_activity_id}")
            # TODO: Implement branch identification logic
            await asyncio.sleep(0.1)  # Simulate processing
            return [failed_activity_id]  # Simplified for now
        except Exception as e:
            logger.error(f"Error identifying failed branch: {e}")
            return []
    
    async def _reverse_branch_activities(
        self, activities: List[str], clinical_context: ClinicalContext
    ) -> bool:
        """Reverse activities in a specific branch."""
        try:
            logger.info(f"Reversing {len(activities)} branch activities")
            # TODO: Implement branch reversal logic
            await asyncio.sleep(0.1)  # Simulate processing
            return True
        except Exception as e:
            logger.error(f"Error reversing branch activities: {e}")
            return False
    
    async def _create_incident_log(
        self, workflow_instance_id: str, failed_activity_id: str,
        clinical_context: ClinicalContext, error_details: Dict[str, Any]
    ) -> bool:
        """Create an incident log entry."""
        try:
            logger.info(f"Creating incident log for activity {failed_activity_id}")
            # TODO: Integrate with logging system
            await asyncio.sleep(0.1)  # Simulate processing
            return True
        except Exception as e:
            logger.error(f"Error creating incident log: {e}")
            return False
    
    async def _schedule_activity_retry(
        self, workflow_instance_id: str, failed_activity_id: str,
        clinical_context: ClinicalContext, retry_count: int
    ) -> bool:
        """Schedule retry of a failed activity."""
        try:
            logger.info(f"Scheduling retry {retry_count} for activity {failed_activity_id}")
            # TODO: Integrate with workflow engine retry mechanism
            await asyncio.sleep(0.1)  # Simulate processing
            return True
        except Exception as e:
            logger.error(f"Error scheduling retry: {e}")
            return False
    
    async def _log_retry_attempt(
        self, workflow_instance_id: str, failed_activity_id: str,
        retry_count: int, clinical_context: ClinicalContext
    ) -> bool:
        """Log retry attempt."""
        try:
            logger.info(f"Logging retry attempt {retry_count} for activity {failed_activity_id}")
            # TODO: Integrate with audit logging
            await asyncio.sleep(0.1)  # Simulate processing
            return True
        except Exception as e:
            logger.error(f"Error logging retry attempt: {e}")
            return False
    
    async def _mark_workflow_failed(
        self, workflow_instance_id: str, failed_activity_id: str, error_details: Dict[str, Any]
    ) -> bool:
        """Mark workflow as failed."""
        try:
            logger.info(f"Marking workflow {workflow_instance_id} as failed")
            # TODO: Integrate with workflow engine
            await asyncio.sleep(0.1)  # Simulate processing
            return True
        except Exception as e:
            logger.error(f"Error marking workflow failed: {e}")
            return False
    
    async def _create_critical_incident_report(
        self, workflow_instance_id: str, failed_activity_id: str,
        clinical_context: ClinicalContext, error_details: Dict[str, Any]
    ) -> bool:
        """Create critical incident report."""
        try:
            logger.info(f"Creating critical incident report for workflow {workflow_instance_id}")
            # TODO: Integrate with critical incident system
            await asyncio.sleep(0.1)  # Simulate processing
            return True
        except Exception as e:
            logger.error(f"Error creating critical incident report: {e}")
            return False
    
    async def _escalate_to_clinical_supervisor(
        self, workflow_instance_id: str, clinical_context: ClinicalContext, error_details: Dict[str, Any]
    ) -> bool:
        """Escalate to clinical supervisor."""
        try:
            logger.info(f"Escalating workflow {workflow_instance_id} to clinical supervisor")
            # TODO: Integrate with escalation system
            await asyncio.sleep(0.1)  # Simulate processing
            return True
        except Exception as e:
            logger.error(f"Error escalating to supervisor: {e}")
            return False
    
    def get_compensation_history(self, workflow_instance_id: Optional[str] = None) -> Dict[str, Any]:
        """Get compensation history for a workflow or all workflows."""
        if workflow_instance_id:
            return {
                comp_id: comp_data for comp_id, comp_data in self.compensation_history.items()
                if comp_data['workflow_instance_id'] == workflow_instance_id
            }
        else:
            return self.compensation_history.copy()
    
    def get_active_compensations(self) -> Dict[str, Any]:
        """Get currently active compensations."""
        return self.active_compensations.copy()


# Global compensation service instance
clinical_compensation_service = ClinicalCompensationService()
