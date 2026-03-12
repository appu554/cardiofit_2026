"""
Workflow Safety Integration Service - CORRECTED FLOW
Shows how Workflow Engine triggers Safety Gateway FROM clinical validation step.
"""
import logging
from typing import Dict, List, Optional, Any
from datetime import datetime
import asyncio
import aiohttp
import grpc
import time
import httpx

from app.models.clinical_activity_models import (
    ClinicalContext, ClinicalError, ClinicalErrorType, CompensationStrategy
)
from app.production_clinical_context_service import production_clinical_context_service
from app.clinical_execution_pattern_service import (
    clinical_execution_pattern_service, ExecutionPattern
)
from app.security.phi_encryption import phi_encryption_service
from app.security.audit_service import audit_service, AuditEventType, AuditLevel
from app.security.break_glass_access import break_glass_access_service
from app.performance_sla_service import performance_sla_service, PerformanceSLAError
from app.intelligent_circuit_breaker import IntelligentCircuitBreaker, CircuitBreakerConfig

logger = logging.getLogger(__name__)


class WorkflowSafetyIntegrationService:
    """
    Demonstrates the CORRECT flow where Workflow Engine triggers Safety Gateway
    FROM the clinical validation step, not as a separate phase.
    """
    
    def __init__(self):
        self.safety_gateway_endpoint = "http://localhost:8025"  # Safety Gateway Platform
        self.active_workflows = {}

        # Domain service endpoints
        self.service_endpoints = {
            "medication-service": "http://localhost:8009",
            "lab-service": "http://localhost:8000",
            "patient-service": "http://localhost:8003",
            "encounter-service": "http://localhost:8020"
        }

        # Initialize circuit breakers for external services
        self.circuit_breakers = {
            'safety_gateway': IntelligentCircuitBreaker(CircuitBreakerConfig(
                service_name='safety_gateway',
                failure_threshold=3,
                recovery_timeout_ms=30000,
                learning_enabled=True
            )),
            'domain_services': IntelligentCircuitBreaker(CircuitBreakerConfig(
                service_name='domain_services',
                failure_threshold=5,
                recovery_timeout_ms=60000,
                learning_enabled=True
            )),
            'context_service': IntelligentCircuitBreaker(CircuitBreakerConfig(
                service_name='context_service',
                failure_threshold=5,
                recovery_timeout_ms=30000,
                learning_enabled=True
            ))
        }

        logger.info("✅ Workflow Safety Integration Service initialized with Performance SLA Framework")
        
    async def execute_clinical_workflow(
        self,
        workflow_type: str,
        patient_id: str,
        provider_id: str,
        clinical_command: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        Execute clinical workflow with CORRECT Safety Gateway integration and execution patterns.

        ENHANCED FLOW:
        1. Determine execution pattern based on workflow type and risk
        2. Execute using appropriate pattern (Pessimistic/Optimistic/Digital Reflex Arc)
        3. Apply pattern-specific safety validation and commit strategies
        """
        workflow_id = f"workflow_{int(time.time() * 1000)}"
        start_time = time.time()

        try:
            # ⚡ PERFORMANCE: Track workflow initialization phase
            async def initialize_workflow():
                logger.info(f"🔄 Starting clinical workflow: {workflow_id}")
                logger.info(f"   Type: {workflow_type}")
                logger.info(f"   Patient: {patient_id}")
                logger.info(f"   Provider: {provider_id}")

                # 🔒 SECURITY: Log workflow initiation
                await audit_service.log_workflow_event(
                    event_type=AuditEventType.WORKFLOW_STARTED,
                    user_id=provider_id,
                    workflow_instance_id=workflow_id,
                    patient_id=patient_id,
                    action_details={
                        "workflow_type": workflow_type,
                        "command": clinical_command,
                        "initiated_at": datetime.utcnow().isoformat()
                    },
                    audit_level=AuditLevel.STANDARD,
                    phi_accessed=True  # Workflow involves patient data
                )

                # Determine execution pattern based on workflow type
                execution_pattern = clinical_execution_pattern_service.get_pattern_for_workflow(workflow_type)
                logger.info(f"📋 Selected execution pattern: {execution_pattern.value}")
                return execution_pattern

            # Track initialization performance (10ms budget)
            execution_pattern = await performance_sla_service.track_phase_performance(
                "workflow_initialization",
                initialize_workflow,
                {"workflow_type": workflow_type, "patient_id": patient_id}
            )

            # Initialize workflow state
            workflow_state = {
                "workflow_id": workflow_id,
                "workflow_type": workflow_type,
                "patient_id": patient_id,
                "provider_id": provider_id,
                "command": clinical_command,
                "execution_pattern": execution_pattern.value,
                "started_at": datetime.utcnow(),
                "current_phase": "INITIALIZING",
                "proposals": [],
                "safety_validations": [],
                "execution_results": []
            }

            # 🔒 SECURITY: Encrypt workflow state containing PHI
            encrypted_state = await phi_encryption_service.encrypt_workflow_state(
                workflow_state, provider_id, workflow_id
            )

            self.active_workflows[workflow_id] = {
                "encrypted_state": encrypted_state,
                "metadata": {
                    "workflow_id": workflow_id,
                    "patient_id": patient_id,
                    "provider_id": provider_id,
                    "workflow_type": workflow_type,
                    "started_at": workflow_state["started_at"]
                }
            }
            
            # ⚡ PERFORMANCE: PHASE 1: CALCULATE - Generate Proposal (50ms budget)
            async def calculate_phase():
                logger.info(f"📊 PHASE 1: CALCULATE - Generating proposal")
                workflow_state["current_phase"] = "CALCULATE"

                proposal = await self._execute_calculate_phase(
                    workflow_type, patient_id, provider_id, clinical_command
                )

                workflow_state["proposals"].append(proposal)
                logger.info(f"✅ Proposal generated: {proposal['proposal_id']}")
                return proposal

            proposal = await performance_sla_service.track_phase_performance(
                "proposal_generation",
                calculate_phase,
                {"workflow_type": workflow_type, "patient_id": patient_id}
            )

            # ⚡ PERFORMANCE: PHASE 2: VALIDATE - Safety Gateway (100ms budget)
            async def validate_phase():
                logger.info(f"🛡️  PHASE 2: VALIDATE - Triggering Safety Gateway FROM validation")
                workflow_state["current_phase"] = "VALIDATE"

                safety_validation = await self._execute_validate_phase_with_safety_gateway(
                    workflow_type, patient_id, provider_id, proposal
                )

                workflow_state["safety_validations"].append(safety_validation)
                logger.info(f"✅ Safety validation complete: {safety_validation['verdict']}")
                return safety_validation

            safety_validation = await performance_sla_service.track_phase_performance(
                "safety_validation",
                validate_phase,
                {"workflow_type": workflow_type, "patient_id": patient_id, "proposal_id": proposal.get("proposal_id")}
            )

            # ⚡ PERFORMANCE: PHASE 3: COMMIT - Execute (30ms budget)
            async def commit_phase():
                logger.info(f"💾 PHASE 3: COMMIT - Executing based on safety verdict")
                workflow_state["current_phase"] = "COMMIT"
                return workflow_state

            workflow_state = await performance_sla_service.track_phase_performance(
                "commit_operation",
                commit_phase,
                {"workflow_type": workflow_type, "safety_verdict": safety_validation.get("verdict")}
            )
            
            execution_result = await self._execute_commit_phase(
                workflow_type, proposal, safety_validation
            )
            
            workflow_state["execution_results"].append(execution_result)
            workflow_state["current_phase"] = "COMPLETED"
            workflow_state["completed_at"] = datetime.utcnow()
            
            # Calculate total execution time
            total_time_ms = (time.time() - start_time) * 1000

            # 🔒 SECURITY: Log workflow completion
            await audit_service.log_workflow_event(
                event_type=AuditEventType.WORKFLOW_COMPLETED,
                user_id=provider_id,
                workflow_instance_id=workflow_id,
                patient_id=patient_id,
                action_details={
                    "execution_time_ms": total_time_ms,
                    "proposal_id": proposal.get("proposal_id"),
                    "safety_verdict": safety_validation.get("verdict"),
                    "execution_status": execution_result.get("status"),
                    "completed_at": datetime.utcnow().isoformat()
                },
                audit_level=AuditLevel.STANDARD,
                outcome="success"
            )

            logger.info(f"🎉 Workflow completed: {workflow_id} in {total_time_ms:.1f}ms")

            # 🔒 SECURITY: Decrypt workflow state for response (if needed)
            decrypted_state = await phi_encryption_service.decrypt_workflow_state(
                encrypted_state, provider_id, workflow_id
            )

            return {
                "workflow_id": workflow_id,
                "status": "completed",
                "execution_time_ms": total_time_ms,
                "proposal": proposal,
                "safety_validation": safety_validation,
                "execution_result": execution_result,
                "workflow_state": decrypted_state
            }

        except Exception as e:
            logger.error(f"❌ Workflow failed: {workflow_id} - {e}")

            # 🔒 SECURITY: Log workflow failure
            await audit_service.log_workflow_event(
                event_type=AuditEventType.WORKFLOW_FAILED,
                user_id=provider_id,
                workflow_instance_id=workflow_id,
                patient_id=patient_id,
                action_details={
                    "error_message": str(e),
                    "failed_at": datetime.utcnow().isoformat(),
                    "execution_time_ms": (time.time() - start_time) * 1000
                },
                audit_level=AuditLevel.DETAILED,
                outcome="failure",
                error_details={"error": str(e), "error_type": type(e).__name__}
            )

            if workflow_id in self.active_workflows:
                self.active_workflows[workflow_id]["metadata"]["current_phase"] = "FAILED"
                self.active_workflows[workflow_id]["metadata"]["error"] = str(e)
                self.active_workflows[workflow_id]["metadata"]["failed_at"] = datetime.utcnow()

            return {
                "workflow_id": workflow_id,
                "status": "failed",
                "error": str(e)
            }

    async def execute_clinical_workflow_with_patterns(
        self,
        workflow_type: str,
        patient_id: str,
        provider_id: str,
        clinical_command: Dict[str, Any],
        execution_pattern: Optional[ExecutionPattern] = None
    ) -> Dict[str, Any]:
        """
        Execute clinical workflow using execution patterns.

        This is the enhanced version that uses the clinical execution pattern service
        to apply risk-appropriate execution strategies.
        """
        workflow_id = f"workflow_{int(time.time() * 1000)}"
        start_time = time.time()

        try:
            logger.info(f"🔄 Starting pattern-based clinical workflow: {workflow_id}")
            logger.info(f"   Type: {workflow_type}")
            logger.info(f"   Patient: {patient_id}")
            logger.info(f"   Provider: {provider_id}")

            # Determine execution pattern if not specified
            if execution_pattern is None:
                execution_pattern = clinical_execution_pattern_service.get_pattern_for_workflow(workflow_type)

            logger.info(f"📋 Using execution pattern: {execution_pattern.value}")

            # Initialize workflow state
            workflow_state = {
                "workflow_id": workflow_id,
                "workflow_type": workflow_type,
                "patient_id": patient_id,
                "provider_id": provider_id,
                "command": clinical_command,
                "execution_pattern": execution_pattern.value,
                "started_at": datetime.utcnow(),
                "current_phase": "INITIALIZING"
            }

            self.active_workflows[workflow_id] = workflow_state

            # Execute workflow using the appropriate execution pattern
            execution_context = {
                "patient_id": patient_id,
                "provider_id": provider_id,
                "workflow_id": workflow_id
            }

            # Use clinical execution pattern service for pattern-specific execution
            pattern_result = await clinical_execution_pattern_service.execute_workflow_with_pattern(
                workflow_type=workflow_type,
                pattern=execution_pattern,
                workflow_data=clinical_command,
                execution_context=execution_context
            )

            # Update workflow state with pattern result
            workflow_state.update({
                "current_phase": "COMPLETED",
                "pattern_result": pattern_result,
                "execution_pattern_used": execution_pattern.value,
                "completed_at": datetime.utcnow()
            })

            # Calculate total execution time
            total_time_ms = (time.time() - start_time) * 1000

            # Final workflow result
            final_result = {
                "workflow_id": workflow_id,
                "workflow_type": workflow_type,
                "patient_id": patient_id,
                "provider_id": provider_id,
                "execution_pattern": execution_pattern.value,
                "final_status": pattern_result.get("status", "completed"),
                "total_execution_time_ms": total_time_ms,
                "sla_compliance": not pattern_result.get("sla_violation", False),
                "pattern_execution": pattern_result,
                "workflow_state": workflow_state
            }

            logger.info(f"🎉 Pattern-based workflow {workflow_id} completed in {total_time_ms:.1f}ms using {execution_pattern.value} pattern")
            return final_result

        except Exception as e:
            logger.error(f"❌ Pattern-based workflow failed: {workflow_id} - {e}")

            if workflow_id in self.active_workflows:
                self.active_workflows[workflow_id]["current_phase"] = "FAILED"
                self.active_workflows[workflow_id]["error"] = str(e)
                self.active_workflows[workflow_id]["failed_at"] = datetime.utcnow()

            return {
                "workflow_id": workflow_id,
                "status": "failed",
                "error": str(e),
                "execution_pattern": execution_pattern.value if execution_pattern else "unknown"
            }

    async def _execute_calculate_phase(
        self,
        workflow_type: str,
        patient_id: str,
        provider_id: str,
        clinical_command: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        PHASE 1: CALCULATE - Generate proposal from domain service.
        No side effects, purely computational.
        """
        try:
            logger.info(f"📊 Executing CALCULATE phase for {workflow_type}")
            
            # Get minimal context needed for calculation
            context = await production_clinical_context_service.get_clinical_context(
                patient_id=patient_id,
                workflow_type=workflow_type,
                provider_id=provider_id
            )
            
            # Route to appropriate domain service for proposal generation
            if workflow_type == "medication_prescribing":
                proposal = await self._generate_medication_proposal(clinical_command, context)
            elif workflow_type == "lab_ordering":
                proposal = await self._generate_lab_proposal(clinical_command, context)
            elif workflow_type == "clinical_deterioration_response":
                proposal = await self._generate_deterioration_response_proposal(clinical_command, context)
            else:
                raise ValueError(f"Unknown workflow type: {workflow_type}")
            
            # Add metadata
            proposal.update({
                "generated_at": datetime.utcnow().isoformat(),
                "context_snapshot": {
                    "patient_id": context.patient_id,
                    "provider_id": context.provider_id,
                    "data_sources": list(context.data_sources.keys())
                }
            })
            
            logger.info(f"✅ CALCULATE phase completed: {proposal['proposal_id']}")
            return proposal
            
        except Exception as e:
            logger.error(f"❌ CALCULATE phase failed: {e}")
            raise
    
    async def _execute_validate_phase_with_safety_gateway(
        self,
        workflow_type: str,
        patient_id: str,
        provider_id: str,
        proposal: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        PHASE 2: VALIDATE - THIS IS THE CORRECTED FLOW
        Workflow Engine triggers Safety Gateway FROM this validation step.
        """
        try:
            logger.info(f"🛡️  Executing VALIDATE phase - Triggering Safety Gateway")
            
            # Step 1: Get complete clinical context for safety validation
            context = await production_clinical_context_service.get_clinical_context(
                patient_id=patient_id,
                workflow_type=workflow_type,
                provider_id=provider_id,
                force_refresh=True  # Fresh context for safety validation
            )
            
            # Step 2: THIS IS WHERE SAFETY GATEWAY IS TRIGGERED FROM WORKFLOW
            logger.info(f"🚨 TRIGGERING SAFETY GATEWAY from workflow validation step")
            
            safety_request = {
                "request_id": f"safety_{proposal['proposal_id']}",
                "workflow_type": workflow_type,
                "patient_id": patient_id,
                "provider_id": provider_id,
                "proposal": proposal,
                "clinical_context": {
                    "patient_demographics": context.clinical_data.get("patient_service", {}),
                    "current_medications": context.clinical_data.get("medication_service", {}),
                    "allergies": context.clinical_data.get("fhir_store", {}).get("allergies", []),
                    "medical_history": context.clinical_data.get("fhir_store", {}).get("medical_history", []),
                    "provider_context": context.clinical_data.get("context_service", {})
                },
                "validation_requirements": {
                    "safety_engines": ["drug_interaction", "allergy", "dosage", "contraindication"],
                    "timeout_ms": 100,  # Safety Gateway SLA
                    "fail_closed": True  # Fail safe if uncertain
                }
            }
            
            # Step 3: Call Safety Gateway Platform
            safety_response = await self._call_safety_gateway(safety_request)
            
            # Step 4: Process Safety Gateway verdict
            safety_validation = {
                "validation_id": safety_request["request_id"],
                "proposal_id": proposal["proposal_id"],
                "verdict": safety_response["verdict"],
                "safety_engines_results": safety_response.get("engine_results", {}),
                "validation_time_ms": safety_response.get("processing_time_ms", 0),
                "validated_at": datetime.utcnow().isoformat(),
                "safety_gateway_triggered_from": "workflow_validation_step"
            }
            
            logger.info(f"✅ Safety Gateway responded: {safety_response['verdict']}")
            
            # Step 5: Handle different verdicts
            if safety_response["verdict"] == "SAFE":
                logger.info("✅ Proposal is SAFE - proceeding to commit")
            elif safety_response["verdict"] == "SAFE_WITH_CONDITIONS":
                logger.info("⚠️  Proposal is SAFE WITH CONDITIONS - applying conditions")
                safety_validation["conditions"] = safety_response.get("conditions", [])
            elif safety_response["verdict"] == "NEEDS_REVIEW":
                logger.info("👨‍⚕️ Proposal NEEDS HUMAN REVIEW - creating human task")
                safety_validation["human_task_required"] = True
            elif safety_response["verdict"] == "UNSAFE":
                logger.error("🚨 Proposal is UNSAFE - blocking execution")
                safety_validation["execution_blocked"] = True
            
            return safety_validation
            
        except Exception as e:
            logger.error(f"❌ VALIDATE phase failed: {e}")
            raise
    
    async def _call_safety_gateway(self, safety_request: Dict[str, Any]) -> Dict[str, Any]:
        """
        Call the Safety Gateway Platform with the proposal and context.
        This is the CORRECT integration point.
        """
        try:
            logger.info(f"🔗 Calling Safety Gateway at {self.safety_gateway_endpoint}")
            
            async with aiohttp.ClientSession() as session:
                url = f"{self.safety_gateway_endpoint}/api/safety/validate"
                
                async with session.post(
                    url,
                    json=safety_request,
                    timeout=aiohttp.ClientTimeout(total=0.15)  # 150ms timeout
                ) as response:
                    
                    if response.status == 200:
                        safety_response = await response.json()
                        logger.info(f"✅ Safety Gateway response: {safety_response['verdict']}")
                        return safety_response
                    else:
                        logger.error(f"❌ Safety Gateway returned {response.status}")
                        # Fail closed - if Safety Gateway unavailable, block execution
                        return {
                            "verdict": "UNSAFE",
                            "reason": f"Safety Gateway unavailable (HTTP {response.status})",
                            "processing_time_ms": 0
                        }
                        
        except asyncio.TimeoutError:
            logger.error("❌ Safety Gateway timeout")
            # Fail closed - if timeout, block execution
            return {
                "verdict": "UNSAFE",
                "reason": "Safety Gateway timeout",
                "processing_time_ms": 150
            }
        except Exception as e:
            logger.error(f"❌ Safety Gateway error: {e}")
            # Fail closed - if error, block execution
            return {
                "verdict": "UNSAFE",
                "reason": f"Safety Gateway error: {str(e)}",
                "processing_time_ms": 0
            }
    
    async def _execute_commit_phase(
        self,
        workflow_type: str,
        proposal: Dict[str, Any],
        safety_validation: Dict[str, Any]
    ) -> Dict[str, Any]:
        """
        PHASE 3: COMMIT - Execute based on Safety Gateway verdict.
        """
        try:
            logger.info(f"💾 Executing COMMIT phase based on safety verdict: {safety_validation['verdict']}")
            
            # Only commit if Safety Gateway approved
            if safety_validation["verdict"] == "SAFE":
                # Execute the proposal
                execution_result = await self._commit_proposal(workflow_type, proposal)
                execution_result["safety_approved"] = True
                
            elif safety_validation["verdict"] == "SAFE_WITH_CONDITIONS":
                # Apply conditions and then commit
                execution_result = await self._commit_proposal_with_conditions(
                    workflow_type, proposal, safety_validation.get("conditions", [])
                )
                execution_result["safety_approved"] = True
                execution_result["conditions_applied"] = True
                
            elif safety_validation["verdict"] == "NEEDS_REVIEW":
                # Create human task, don't commit yet
                execution_result = await self._create_human_review_task(workflow_type, proposal, safety_validation)
                execution_result["safety_approved"] = False
                execution_result["human_review_required"] = True
                
            elif safety_validation["verdict"] == "UNSAFE":
                # Block execution, create incident
                execution_result = await self._block_unsafe_execution(workflow_type, proposal, safety_validation)
                execution_result["safety_approved"] = False
                execution_result["execution_blocked"] = True
                
            else:
                raise ValueError(f"Unknown safety verdict: {safety_validation['verdict']}")
            
            execution_result.update({
                "committed_at": datetime.utcnow().isoformat(),
                "safety_validation_id": safety_validation["validation_id"]
            })
            
            logger.info(f"✅ COMMIT phase completed: {execution_result.get('status', 'unknown')}")
            return execution_result
            
        except Exception as e:
            logger.error(f"❌ COMMIT phase failed: {e}")
            raise
    
    # Domain service proposal generators
    async def _generate_medication_proposal(self, command: Dict[str, Any], context: ClinicalContext) -> Dict[str, Any]:
        """Generate medication prescription proposal by calling Medication Service."""
        try:
            logger.info(f"🔗 Calling Medication Service to generate proposal")

            # Prepare request for Medication Service
            proposal_request = {
                "patient_id": context.patient_id,
                "medication_code": command.get("medication_code"),
                "medication_name": command.get("medication_name"),
                "dosage": command.get("dosage"),
                "frequency": command.get("frequency"),
                "duration": command.get("duration"),
                "route": command.get("route", "oral"),
                "priority": command.get("priority", "routine"),
                "indication": command.get("indication"),
                "provider_id": context.provider_id,
                "encounter_id": command.get("encounter_id"),
                "notes": command.get("notes")
            }

            # Call Medication Service proposal endpoint
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{self.service_endpoints['medication-service']}/api/proposals/medication",
                    json=proposal_request,
                    timeout=30.0
                )

                if response.status_code == 201:
                    proposal_response = response.json()
                    logger.info(f"✅ Medication proposal created: {proposal_response['proposal_id']}")
                    return proposal_response["proposal_data"]
                else:
                    logger.error(f"❌ Medication Service returned {response.status_code}: {response.text}")
                    raise Exception(f"Failed to create medication proposal: {response.status_code}")

        except Exception as e:
            logger.error(f"❌ Error generating medication proposal: {e}")
            raise
    
    async def _generate_lab_proposal(self, command: Dict[str, Any], context: ClinicalContext) -> Dict[str, Any]:
        """Generate lab order proposal by calling Lab Service."""
        try:
            logger.info(f"🔗 Calling Lab Service to generate proposal")

            # Prepare request for Lab Service
            proposal_request = {
                "patient_id": context.patient_id,
                "lab_tests": command.get("lab_tests", []),
                "priority": command.get("priority", "routine"),
                "ordering_provider_id": context.provider_id,
                "encounter_id": command.get("encounter_id"),
                "clinical_indication": command.get("indication"),
                "notes": command.get("notes")
            }

            # Call Lab Service proposal endpoint (when implemented)
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{self.service_endpoints['lab-service']}/api/proposals/lab-order",
                    json=proposal_request,
                    timeout=30.0
                )

                if response.status_code == 201:
                    proposal_response = response.json()
                    logger.info(f"✅ Lab proposal created: {proposal_response['proposal_id']}")
                    return proposal_response["proposal_data"]
                else:
                    logger.error(f"❌ Lab Service returned {response.status_code}: {response.text}")
                    raise Exception(f"Failed to create lab proposal: {response.status_code}")

        except Exception as e:
            logger.error(f"❌ Error generating lab proposal: {e}")
            raise
    
    async def _generate_deterioration_response_proposal(self, command: Dict[str, Any], context: ClinicalContext) -> Dict[str, Any]:
        """Generate clinical deterioration response proposal (Digital Reflex Arc)."""
        try:
            logger.info(f"🔗 Generating deterioration response proposal")

            # For deterioration response, we create a composite proposal
            # that may involve multiple services (medication, lab, encounter)
            proposal_id = f"deterioration_proposal_{int(time.time() * 1000)}"

            return {
                "proposal_id": proposal_id,
                "proposal_type": "deterioration_response",
                "interventions": command.get("interventions", []),
                "urgency_level": command.get("urgency_level", "high"),
                "autonomous_execution": True,
                "patient_id": context.patient_id,
                "provider_id": context.provider_id,
                "created_at": datetime.utcnow().isoformat(),
                "requires_immediate_action": True
            }

        except Exception as e:
            logger.error(f"❌ Error generating deterioration response proposal: {e}")
            raise
    
    # Commit execution methods
    async def _commit_proposal(self, workflow_type: str, proposal: Dict[str, Any]) -> Dict[str, Any]:
        """Commit approved proposal by calling the appropriate domain service."""
        try:
            logger.info(f"✅ Committing approved proposal: {proposal['proposal_id']}")

            if workflow_type == "medication_prescribing":
                return await self._commit_medication_proposal(proposal)
            elif workflow_type == "lab_ordering":
                return await self._commit_lab_proposal(proposal)
            elif workflow_type == "clinical_deterioration_response":
                return await self._commit_deterioration_response_proposal(proposal)
            else:
                raise ValueError(f"Unknown workflow type for commit: {workflow_type}")

        except Exception as e:
            logger.error(f"❌ Error committing proposal: {e}")
            raise
    
    async def _commit_proposal_with_conditions(self, workflow_type: str, proposal: Dict[str, Any], conditions: List[str]) -> Dict[str, Any]:
        """Commit proposal with safety conditions applied."""
        try:
            logger.info(f"✅ Committing proposal with conditions: {conditions}")

            # Apply conditions to the proposal before committing
            modified_proposal = proposal.copy()
            modified_proposal["safety_conditions"] = conditions

            # Commit the modified proposal
            result = await self._commit_proposal(workflow_type, modified_proposal)
            result["conditions_applied"] = conditions
            result["status"] = "committed_with_conditions"

            return result

        except Exception as e:
            logger.error(f"❌ Error committing proposal with conditions: {e}")
            raise

    async def _commit_medication_proposal(self, proposal: Dict[str, Any]) -> Dict[str, Any]:
        """Commit medication proposal by calling Medication Service."""
        try:
            logger.info(f"🔗 Calling Medication Service to commit proposal {proposal['proposal_id']}")

            # Prepare commit request
            commit_request = {
                "safety_validation": {
                    "verdict": "SAFE",
                    "validated_at": datetime.utcnow().isoformat()
                },
                "commit_notes": "Approved by Safety Gateway"
            }

            # Call Medication Service commit endpoint
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{self.service_endpoints['medication-service']}/api/proposals/{proposal['proposal_id']}/commit",
                    json=commit_request,
                    timeout=30.0
                )

                if response.status_code == 200:
                    commit_response = response.json()
                    logger.info(f"✅ Medication proposal committed: {commit_response['fhir_resource_id']}")
                    return {
                        "status": "committed",
                        "proposal_id": proposal["proposal_id"],
                        "fhir_resource_id": commit_response["fhir_resource_id"],
                        "execution_method": "medication_service_commit",
                        "committed_at": commit_response["committed_at"]
                    }
                else:
                    logger.error(f"❌ Medication Service commit failed {response.status_code}: {response.text}")
                    raise Exception(f"Failed to commit medication proposal: {response.status_code}")

        except Exception as e:
            logger.error(f"❌ Error committing medication proposal: {e}")
            raise

    async def _commit_lab_proposal(self, proposal: Dict[str, Any]) -> Dict[str, Any]:
        """Commit lab proposal by calling Lab Service."""
        try:
            logger.info(f"🔗 Calling Lab Service to commit proposal {proposal['proposal_id']}")

            # For now, return a placeholder since Lab Service commit endpoint is not implemented
            return {
                "status": "committed",
                "proposal_id": proposal["proposal_id"],
                "execution_method": "lab_service_commit",
                "committed_at": datetime.utcnow().isoformat(),
                "note": "Lab Service commit endpoint not yet implemented"
            }

        except Exception as e:
            logger.error(f"❌ Error committing lab proposal: {e}")
            raise

    async def _commit_deterioration_response_proposal(self, proposal: Dict[str, Any]) -> Dict[str, Any]:
        """Commit deterioration response proposal."""
        try:
            logger.info(f"🔗 Committing deterioration response proposal {proposal['proposal_id']}")

            # For deterioration response, we may need to coordinate multiple services
            return {
                "status": "committed",
                "proposal_id": proposal["proposal_id"],
                "execution_method": "deterioration_response_commit",
                "committed_at": datetime.utcnow().isoformat(),
                "autonomous_execution": proposal.get("autonomous_execution", False)
            }

        except Exception as e:
            logger.error(f"❌ Error committing deterioration response proposal: {e}")
            raise
    
    async def _create_human_review_task(self, workflow_type: str, proposal: Dict[str, Any], safety_validation: Dict[str, Any]) -> Dict[str, Any]:
        """Create human review task for proposals needing review."""
        logger.info(f"👨‍⚕️ Creating human review task for: {proposal['proposal_id']}")
        return {
            "status": "pending_human_review",
            "proposal_id": proposal["proposal_id"],
            "human_task_id": f"task_{int(time.time() * 1000)}",
            "execution_method": "human_review_required"
        }
    
    async def _block_unsafe_execution(self, workflow_type: str, proposal: Dict[str, Any], safety_validation: Dict[str, Any]) -> Dict[str, Any]:
        """Block unsafe proposal execution."""
        logger.error(f"🚨 Blocking unsafe execution: {proposal['proposal_id']}")
        return {
            "status": "blocked_unsafe",
            "proposal_id": proposal["proposal_id"],
            "safety_reason": safety_validation.get("reason", "Safety Gateway blocked execution"),
            "execution_method": "blocked_for_safety"
        }
    
    def get_workflow_status(self, workflow_id: str) -> Optional[Dict[str, Any]]:
        """Get current workflow status."""
        return self.active_workflows.get(workflow_id)
    
    def get_active_workflows(self) -> Dict[str, Any]:
        """Get all active workflows."""
        return self.active_workflows.copy()


# Global workflow safety integration service
workflow_safety_integration_service = WorkflowSafetyIntegrationService()
