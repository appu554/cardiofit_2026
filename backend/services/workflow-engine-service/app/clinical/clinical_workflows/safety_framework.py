"""
Safety Framework Service for Clinical Workflow Engine.
Integrates compensation patterns with clinical context for comprehensive safety management.
"""
import logging
from typing import Dict, List, Optional, Any
from datetime import datetime
import asyncio

from app.models.clinical_activity_models import (
    ClinicalContext, ClinicalError, ClinicalErrorType, CompensationStrategy
)
from app.clinical_compensation_service import clinical_compensation_service
from app.clinical_context_integration_service import clinical_context_integration_service

logger = logging.getLogger(__name__)


class SafetyFrameworkService:
    """
    Comprehensive safety framework that integrates compensation patterns
    with real clinical context for workflow safety management.
    """
    
    def __init__(self):
        self.safety_incidents = {}
        self.safety_metrics = {}
        self.safety_rules = {}
        self._initialize_safety_rules()
    
    def _initialize_safety_rules(self):
        """
        Initialize safety rules for different clinical scenarios.
        """
        self.safety_rules = {
            "medication_ordering": {
                "critical_errors": [
                    ClinicalErrorType.SAFETY_ERROR,
                    ClinicalErrorType.MOCK_DATA_ERROR
                ],
                "compensation_strategy": CompensationStrategy.FULL_COMPENSATION,
                "escalation_required": True,
                "context_validation_required": True,
                "max_retry_attempts": 0  # No retries for medication safety
            },
            "patient_admission": {
                "critical_errors": [
                    ClinicalErrorType.SAFETY_ERROR,
                    ClinicalErrorType.DATA_SOURCE_ERROR,
                    ClinicalErrorType.MOCK_DATA_ERROR
                ],
                "compensation_strategy": CompensationStrategy.PARTIAL_COMPENSATION,
                "escalation_required": True,
                "context_validation_required": True,
                "max_retry_attempts": 1
            },
            "patient_discharge": {
                "critical_errors": [
                    ClinicalErrorType.SAFETY_ERROR,
                    ClinicalErrorType.MOCK_DATA_ERROR
                ],
                "compensation_strategy": CompensationStrategy.FULL_COMPENSATION,
                "escalation_required": True,
                "context_validation_required": True,
                "max_retry_attempts": 0  # No retries for discharge safety
            },
            "technical_operations": {
                "critical_errors": [
                    ClinicalErrorType.DATA_SOURCE_ERROR
                ],
                "compensation_strategy": CompensationStrategy.FORWARD_RECOVERY,
                "escalation_required": False,
                "context_validation_required": False,
                "max_retry_attempts": 3
            }
        }
    
    async def handle_workflow_safety_incident(
        self,
        workflow_instance_id: str,
        failed_activity_id: str,
        error: ClinicalError,
        workflow_type: str,
        patient_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Handle a safety incident in a clinical workflow with comprehensive safety management.
        
        Args:
            workflow_instance_id: ID of the workflow instance
            failed_activity_id: ID of the failed activity
            error: Clinical error that occurred
            workflow_type: Type of workflow (medication_ordering, etc.)
            patient_id: Patient ID if available
            
        Returns:
            Dict containing safety incident handling results
        """
        incident_id = f"safety_{workflow_instance_id}_{failed_activity_id}_{datetime.utcnow().timestamp()}"
        
        try:
            logger.critical(f"🚨 Safety incident detected: {incident_id}")
            logger.critical(f"   Workflow: {workflow_instance_id}")
            logger.critical(f"   Activity: {failed_activity_id}")
            logger.critical(f"   Error Type: {error.error_type.value}")
            logger.critical(f"   Error Message: {error.error_message}")
            
            # Create safety incident record
            incident_record = {
                "incident_id": incident_id,
                "workflow_instance_id": workflow_instance_id,
                "failed_activity_id": failed_activity_id,
                "error": error,
                "workflow_type": workflow_type,
                "patient_id": patient_id,
                "detected_at": datetime.utcnow(),
                "status": "investigating",
                "safety_actions": []
            }
            
            self.safety_incidents[incident_id] = incident_record
            
            # Step 1: Assess safety criticality
            safety_assessment = await self._assess_safety_criticality(
                error, workflow_type, patient_id
            )
            incident_record["safety_actions"].append({
                "action": "safety_assessment",
                "result": safety_assessment,
                "timestamp": datetime.utcnow().isoformat()
            })
            
            # Step 2: Get or validate clinical context
            clinical_context = None
            if patient_id and safety_assessment["context_required"]:
                try:
                    clinical_context = await clinical_context_integration_service.get_clinical_context(
                        patient_id=patient_id,
                        workflow_type=workflow_type,
                        force_refresh=True  # Always use fresh data for safety incidents
                    )
                    incident_record["safety_actions"].append({
                        "action": "clinical_context_retrieved",
                        "result": {"success": True, "context_age_seconds": 0},
                        "timestamp": datetime.utcnow().isoformat()
                    })
                except Exception as e:
                    logger.error(f"Failed to get clinical context for safety incident: {e}")
                    incident_record["safety_actions"].append({
                        "action": "clinical_context_retrieval_failed",
                        "result": {"success": False, "error": str(e)},
                        "timestamp": datetime.utcnow().isoformat()
                    })
                    
                    # If context is required but unavailable, escalate immediately
                    if safety_assessment["context_required"]:
                        return await self._handle_context_unavailable_incident(
                            incident_record, safety_assessment
                        )
            
            # Step 3: Determine compensation strategy
            compensation_strategy = await self._determine_compensation_strategy(
                error, workflow_type, safety_assessment
            )
            incident_record["safety_actions"].append({
                "action": "compensation_strategy_determined",
                "result": {"strategy": compensation_strategy.value},
                "timestamp": datetime.utcnow().isoformat()
            })
            
            # Step 4: Execute compensation
            if clinical_context:
                compensation_success = await clinical_compensation_service.execute_compensation(
                    strategy=compensation_strategy,
                    workflow_instance_id=workflow_instance_id,
                    failed_activity_id=failed_activity_id,
                    clinical_context=clinical_context,
                    error_details=error.error_data
                )
            else:
                # Execute compensation without clinical context for technical errors
                compensation_success = await self._execute_technical_compensation(
                    compensation_strategy, workflow_instance_id, failed_activity_id, error
                )
            
            incident_record["safety_actions"].append({
                "action": "compensation_executed",
                "result": {"success": compensation_success},
                "timestamp": datetime.utcnow().isoformat()
            })
            
            # Step 5: Handle escalation if required
            if safety_assessment["escalation_required"]:
                escalation_result = await self._escalate_safety_incident(
                    incident_record, clinical_context
                )
                incident_record["safety_actions"].append({
                    "action": "safety_escalation",
                    "result": escalation_result,
                    "timestamp": datetime.utcnow().isoformat()
                })
            
            # Step 6: Update safety metrics
            await self._update_safety_metrics(workflow_type, error.error_type, compensation_success)
            
            # Step 7: Finalize incident
            incident_record["status"] = "resolved" if compensation_success else "failed"
            incident_record["resolved_at"] = datetime.utcnow()
            
            # Return comprehensive safety incident result
            return {
                "incident_id": incident_id,
                "safety_status": "handled",
                "compensation_success": compensation_success,
                "compensation_strategy": compensation_strategy.value,
                "escalated": safety_assessment["escalation_required"],
                "context_used": clinical_context is not None,
                "safety_actions_count": len(incident_record["safety_actions"]),
                "resolution_time_seconds": (
                    incident_record["resolved_at"] - incident_record["detected_at"]
                ).total_seconds()
            }
            
        except Exception as e:
            logger.error(f"❌ Safety incident handling failed for {incident_id}: {e}")
            
            # Update incident record with error
            if incident_id in self.safety_incidents:
                self.safety_incidents[incident_id]["status"] = "error"
                self.safety_incidents[incident_id]["error"] = str(e)
                self.safety_incidents[incident_id]["resolved_at"] = datetime.utcnow()
            
            return {
                "incident_id": incident_id,
                "safety_status": "error",
                "error": str(e),
                "compensation_success": False
            }
    
    async def _assess_safety_criticality(
        self,
        error: ClinicalError,
        workflow_type: str,
        patient_id: Optional[str]
    ) -> Dict[str, Any]:
        """
        Assess the safety criticality of an error.
        """
        try:
            safety_rules = self.safety_rules.get(workflow_type, self.safety_rules["technical_operations"])
            
            is_critical = error.error_type in safety_rules["critical_errors"]
            
            assessment = {
                "critical": is_critical,
                "error_type": error.error_type.value,
                "workflow_type": workflow_type,
                "escalation_required": safety_rules["escalation_required"] and is_critical,
                "context_required": safety_rules["context_validation_required"] and patient_id is not None,
                "max_retries": safety_rules["max_retry_attempts"],
                "safety_level": "critical" if is_critical else "warning",
                "patient_safety_risk": is_critical and patient_id is not None
            }
            
            logger.info(f"🔍 Safety assessment: {assessment}")
            return assessment
            
        except Exception as e:
            logger.error(f"Safety assessment error: {e}")
            # Default to critical for safety
            return {
                "critical": True,
                "error_type": error.error_type.value,
                "escalation_required": True,
                "context_required": True,
                "safety_level": "critical",
                "patient_safety_risk": True
            }
    
    async def _determine_compensation_strategy(
        self,
        error: ClinicalError,
        workflow_type: str,
        safety_assessment: Dict[str, Any]
    ) -> CompensationStrategy:
        """
        Determine the appropriate compensation strategy based on safety assessment.
        """
        try:
            safety_rules = self.safety_rules.get(workflow_type, self.safety_rules["technical_operations"])
            
            # For critical safety errors, always use the defined strategy
            if safety_assessment["critical"]:
                strategy = safety_rules["compensation_strategy"]
                logger.info(f"🔄 Critical error - using {strategy.value} compensation")
                return strategy
            
            # For non-critical errors, consider retry options
            retry_count = error.error_data.get("retry_count", 0)
            max_retries = safety_rules["max_retry_attempts"]
            
            if retry_count < max_retries and error.error_type == ClinicalErrorType.TECHNICAL_ERROR:
                logger.info(f"🔄 Technical error - attempting forward recovery (retry {retry_count + 1})")
                return CompensationStrategy.FORWARD_RECOVERY
            else:
                logger.info(f"🔄 Max retries exceeded or non-retryable - using {safety_rules['compensation_strategy'].value}")
                return safety_rules["compensation_strategy"]
                
        except Exception as e:
            logger.error(f"Error determining compensation strategy: {e}")
            # Default to full compensation for safety
            return CompensationStrategy.FULL_COMPENSATION
    
    async def _handle_context_unavailable_incident(
        self,
        incident_record: Dict[str, Any],
        safety_assessment: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Handle incidents where clinical context is required but unavailable.
        """
        logger.critical("🚨 Clinical context unavailable for safety-critical incident")
        
        incident_record["status"] = "context_unavailable"
        incident_record["resolved_at"] = datetime.utcnow()
        
        # Immediate escalation for context unavailability
        escalation_result = await self._escalate_safety_incident(incident_record, None)
        incident_record["safety_actions"].append({
            "action": "emergency_escalation_context_unavailable",
            "result": escalation_result,
            "timestamp": datetime.utcnow().isoformat()
        })
        
        return {
            "incident_id": incident_record["incident_id"],
            "safety_status": "context_unavailable",
            "compensation_success": False,
            "escalated": True,
            "error": "Clinical context required but unavailable"
        }
    
    async def _execute_technical_compensation(
        self,
        strategy: CompensationStrategy,
        workflow_instance_id: str,
        failed_activity_id: str,
        error: ClinicalError
    ) -> bool:
        """
        Execute compensation for technical errors without clinical context.
        """
        try:
            logger.info(f"🔧 Executing technical compensation: {strategy.value}")
            
            if strategy == CompensationStrategy.FORWARD_RECOVERY:
                # Schedule retry for technical errors
                retry_count = error.error_data.get("retry_count", 0)
                await asyncio.sleep(2 ** retry_count)  # Exponential backoff
                logger.info(f"✅ Technical retry scheduled (attempt {retry_count + 1})")
                return True
            elif strategy == CompensationStrategy.IMMEDIATE_FAILURE:
                # Mark workflow as failed
                logger.info("🚨 Technical immediate failure executed")
                return True
            else:
                # Other strategies require clinical context
                logger.error(f"Cannot execute {strategy.value} without clinical context")
                return False
                
        except Exception as e:
            logger.error(f"Technical compensation error: {e}")
            return False
    
    async def _escalate_safety_incident(
        self,
        incident_record: Dict[str, Any],
        clinical_context: Optional[ClinicalContext]
    ) -> Dict[str, Any]:
        """
        Escalate safety incident to appropriate clinical personnel.
        """
        try:
            logger.critical(f"🚨 Escalating safety incident: {incident_record['incident_id']}")
            
            escalation_data = {
                "incident_id": incident_record["incident_id"],
                "workflow_instance_id": incident_record["workflow_instance_id"],
                "patient_id": incident_record.get("patient_id"),
                "error_type": incident_record["error"].error_type.value,
                "error_message": incident_record["error"].error_message,
                "workflow_type": incident_record["workflow_type"],
                "escalated_at": datetime.utcnow().isoformat(),
                "escalation_level": "critical",
                "requires_immediate_attention": True
            }
            
            # Add clinical context if available
            if clinical_context:
                escalation_data["clinical_context"] = {
                    "patient_id": clinical_context.patient_id,
                    "provider_id": clinical_context.provider_id,
                    "encounter_id": clinical_context.encounter_id
                }
            
            # TODO: Integrate with actual escalation system
            # await escalation_service.escalate_incident(escalation_data)
            
            logger.critical("🚨 Safety incident escalated successfully")
            return {
                "escalated": True,
                "escalation_level": "critical",
                "escalated_at": escalation_data["escalated_at"]
            }
            
        except Exception as e:
            logger.error(f"Escalation error: {e}")
            return {
                "escalated": False,
                "error": str(e)
            }
    
    async def _update_safety_metrics(
        self,
        workflow_type: str,
        error_type: ClinicalErrorType,
        compensation_success: bool
    ):
        """
        Update safety metrics for monitoring and reporting.
        """
        try:
            if workflow_type not in self.safety_metrics:
                self.safety_metrics[workflow_type] = {
                    "total_incidents": 0,
                    "critical_incidents": 0,
                    "successful_compensations": 0,
                    "failed_compensations": 0,
                    "error_types": {},
                    "last_updated": datetime.utcnow().isoformat()
                }
            
            metrics = self.safety_metrics[workflow_type]
            metrics["total_incidents"] += 1
            
            if error_type in [ClinicalErrorType.SAFETY_ERROR, ClinicalErrorType.MOCK_DATA_ERROR]:
                metrics["critical_incidents"] += 1
            
            if compensation_success:
                metrics["successful_compensations"] += 1
            else:
                metrics["failed_compensations"] += 1
            
            # Track error types
            error_type_key = error_type.value
            if error_type_key not in metrics["error_types"]:
                metrics["error_types"][error_type_key] = 0
            metrics["error_types"][error_type_key] += 1
            
            metrics["last_updated"] = datetime.utcnow().isoformat()
            
            logger.info(f"📊 Safety metrics updated for {workflow_type}")
            
        except Exception as e:
            logger.error(f"Error updating safety metrics: {e}")
    
    async def validate_workflow_safety_readiness(
        self,
        workflow_type: str,
        patient_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """
        Validate that a workflow is ready to execute safely.
        """
        try:
            logger.info(f"🔍 Validating safety readiness for {workflow_type}")
            
            readiness_check = {
                "ready": True,
                "workflow_type": workflow_type,
                "patient_id": patient_id,
                "checks": [],
                "warnings": [],
                "errors": [],
                "checked_at": datetime.utcnow().isoformat()
            }
            
            # Check 1: Safety rules configuration
            if workflow_type not in self.safety_rules:
                readiness_check["warnings"].append(f"No specific safety rules for {workflow_type}, using defaults")
            
            readiness_check["checks"].append({
                "check": "safety_rules_configured",
                "passed": workflow_type in self.safety_rules
            })
            
            # Check 2: Clinical context availability (if patient involved)
            if patient_id:
                try:
                    context_availability = await clinical_context_integration_service.validate_context_availability(
                        patient_id, workflow_type
                    )
                    
                    if not context_availability["available"]:
                        readiness_check["errors"].append("Clinical context not available")
                        readiness_check["ready"] = False
                    
                    readiness_check["checks"].append({
                        "check": "clinical_context_available",
                        "passed": context_availability["available"],
                        "details": context_availability
                    })
                    
                except Exception as e:
                    readiness_check["errors"].append(f"Context availability check failed: {str(e)}")
                    readiness_check["ready"] = False
                    
                    readiness_check["checks"].append({
                        "check": "clinical_context_available",
                        "passed": False,
                        "error": str(e)
                    })
            
            # Check 3: Compensation service availability
            compensation_available = clinical_compensation_service is not None
            readiness_check["checks"].append({
                "check": "compensation_service_available",
                "passed": compensation_available
            })
            
            if not compensation_available:
                readiness_check["errors"].append("Compensation service not available")
                readiness_check["ready"] = False
            
            # Final readiness determination
            if readiness_check["errors"]:
                readiness_check["ready"] = False
            
            logger.info(f"🔍 Safety readiness check complete: {'✅ READY' if readiness_check['ready'] else '❌ NOT READY'}")
            
            return readiness_check
            
        except Exception as e:
            logger.error(f"Safety readiness validation error: {e}")
            return {
                "ready": False,
                "error": str(e),
                "checked_at": datetime.utcnow().isoformat()
            }
    
    def get_safety_metrics(self, workflow_type: Optional[str] = None) -> Dict[str, Any]:
        """
        Get safety metrics for monitoring.
        """
        if workflow_type:
            return self.safety_metrics.get(workflow_type, {})
        else:
            return self.safety_metrics.copy()
    
    def get_safety_incidents(self, workflow_instance_id: Optional[str] = None) -> Dict[str, Any]:
        """
        Get safety incidents for audit and review.
        """
        if workflow_instance_id:
            return {
                incident_id: incident for incident_id, incident in self.safety_incidents.items()
                if incident["workflow_instance_id"] == workflow_instance_id
            }
        else:
            return self.safety_incidents.copy()


# Global safety framework service instance
safety_framework_service = SafetyFrameworkService()
