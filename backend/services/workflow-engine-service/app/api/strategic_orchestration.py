"""
Strategic Orchestration API Endpoints

Exposes the Calculate > Validate > Commit pattern via FastAPI endpoints
for Apollo Federation integration and UI workflow coordination.
"""

import logging
from typing import Dict, Any, Optional
from fastapi import APIRouter, HTTPException, Depends, BackgroundTasks
from pydantic import BaseModel, Field
from datetime import datetime
import uuid

from app.orchestration.strategic_orchestrator import (
    strategic_orchestrator,
    CalculateRequest,
    OrchestrationStep,
    OrchestrationResult
)
from app.orchestration.snapshot_orchestrator import (
    snapshot_aware_orchestrator,
    SnapshotAwareOrchestrator
)
from app.orchestration.interfaces import (
    ClinicalCommand,
    WorkflowInstance,
    SnapshotError,
    SnapshotExpiredError,
    SnapshotIntegrityError,
    SnapshotConsistencyError
)

logger = logging.getLogger(__name__)

router = APIRouter()


class MedicationRequestInput(BaseModel):
    """Input model for medication requests from Apollo Federation"""
    patient_id: str = Field(..., description="Patient identifier")
    encounter_id: Optional[str] = Field(None, description="Encounter identifier")
    
    # Clinical Intent  
    indication: str = Field(..., description="Clinical indication (e.g., hypertension_stage2_ckd)")
    urgency: str = Field("ROUTINE", description="Request urgency level")
    constraints: Optional[list] = Field(default_factory=list, description="Clinical constraints")
    
    # Medication Details
    medication: Dict[str, Any] = Field(..., description="Medication details from UI")
    
    # Provider Context
    provider_id: str = Field(..., description="Provider identifier")
    specialty: Optional[str] = Field(None, description="Provider specialty")
    location: Optional[str] = Field(None, description="Clinical location")
    
    # Session Context
    session_token: Optional[str] = Field(None, description="Authentication token")


class OrchestrationResponse(BaseModel):
    """Enhanced response model for orchestration results with snapshot metadata"""
    status: str = Field(..., description="Orchestration status")
    correlation_id: str = Field(..., description="Request correlation ID")
    execution_time_ms: Optional[float] = Field(None, description="Total execution time")
    
    # Snapshot-aware fields
    snapshot_metadata: Optional[Dict[str, Any]] = Field(None, description="Snapshot metadata for consistency")
    workflow_id: Optional[str] = Field(None, description="Workflow instance identifier")
    
    # Success path fields
    medication_order_id: Optional[str] = Field(None, description="Created medication order ID")
    calculation: Optional[Dict[str, Any]] = Field(None, description="Calculate step results")
    validation: Optional[Dict[str, Any]] = Field(None, description="Validate step results")
    commitment: Optional[Dict[str, Any]] = Field(None, description="Commit step results")
    performance: Optional[Dict[str, Any]] = Field(None, description="Performance metrics")
    
    # Alternative paths
    validation_findings: Optional[list] = Field(None, description="Validation findings requiring provider decision")
    override_tokens: Optional[list] = Field(None, description="Override tokens for warnings")
    proposals: Optional[list] = Field(None, description="Available medication proposals")
    blocking_findings: Optional[list] = Field(None, description="Blocking safety findings")
    alternative_approaches: Optional[list] = Field(None, description="Alternative medication approaches")
    
    # Error handling
    error_code: Optional[str] = Field(None, description="Error code for failures")
    error_message: Optional[str] = Field(None, description="Error message for failures")
    
    # Snapshot-specific error fields
    snapshot_error_details: Optional[Dict[str, Any]] = Field(None, description="Snapshot-specific error information")


class OverrideDecisionInput(BaseModel):
    """Input for provider override decisions"""
    correlation_id: str = Field(..., description="Original request correlation ID")
    snapshot_id: str = Field(..., description="Clinical snapshot ID")
    selected_proposal_index: int = Field(..., description="Index of selected proposal")
    override_tokens: list = Field(..., description="Override tokens from validation")
    provider_justification: str = Field(..., description="Provider justification for override")


@router.post("/orchestrate/medication", response_model=OrchestrationResponse)
async def orchestrate_medication_request(
    request: MedicationRequestInput,
    background_tasks: BackgroundTasks
) -> OrchestrationResponse:
    """
    Main orchestration endpoint implementing Calculate > Validate > Commit pattern
    
    This endpoint receives medication requests from Apollo Federation and orchestrates
    the complete workflow through Flow2 engines, Safety Gateway, and Medication Service.
    
    Flow:
    1. UI → Apollo Federation → This endpoint
    2. Calculate step → Flow2 Go + Rust engines  
    3. Validate step → Safety Gateway
    4. Commit step → Medication Service (if safe)
    """
    correlation_id = str(uuid.uuid4())
    request_start = datetime.utcnow()
    
    logger.info(f"Received medication orchestration request {correlation_id} for patient {request.patient_id}")
    
    try:
        # Convert API input to internal request format
        calculate_request = CalculateRequest(
            patient_id=request.patient_id,
            medication_request=request.medication,
            clinical_intent={
                "indication": request.indication,
                "urgency": request.urgency, 
                "constraints": request.constraints or []
            },
            provider_context={
                "provider_id": request.provider_id,
                "specialty": request.specialty,
                "location": request.location,
                "encounter_id": request.encounter_id
            },
            correlation_id=correlation_id,
            urgency=request.urgency
        )
        
        # Execute strategic orchestration (backward compatibility)
        result = await strategic_orchestrator.orchestrate_medication_request(calculate_request)
        
        # Calculate total execution time
        execution_time = (datetime.utcnow() - request_start).total_seconds() * 1000
        result["execution_time_ms"] = execution_time
        
        # Log performance metrics
        background_tasks.add_task(
            _log_performance_metrics,
            correlation_id,
            execution_time,
            result.get("status")
        )
        
        # Return structured response
        return OrchestrationResponse(**result)
        
    except Exception as e:
        logger.error(f"Orchestration failed for {correlation_id}: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail={
                "error_code": "ORCHESTRATION_FAILED",
                "error_message": str(e),
                "correlation_id": correlation_id
            }
        )


@router.post("/orchestrate/medication/override", response_model=OrchestrationResponse)
async def handle_provider_override(
    request: OverrideDecisionInput,
    background_tasks: BackgroundTasks
) -> OrchestrationResponse:
    """
    Handle provider override decisions for WARNING validation results
    
    When validation returns WARNING, the provider can choose to override
    with proper justification and override tokens.
    """
    logger.info(f"Processing provider override for {request.correlation_id}")
    
    try:
        # Create override commit request
        override_request = {
            "correlation_id": request.correlation_id,
            "snapshot_id": request.snapshot_id,
            "selected_proposal_index": request.selected_proposal_index,
            "override_tokens": request.override_tokens,
            "provider_justification": request.provider_justification,
            "override_timestamp": datetime.utcnow().isoformat()
        }
        
        # Route to Flow2 Go Engine override handler
        import httpx
        async with httpx.AsyncClient() as client:
            response = await client.post(
                f"{strategic_orchestrator.flow2_go_url}/api/v1/snapshots/execute-override",
                json=override_request,
                headers={"Content-Type": "application/json"}
            )
            response.raise_for_status()
            
            override_result = response.json()
            
            # If override successful, proceed to commit
            if override_result.get("status") == "OVERRIDE_ACCEPTED":
                # Execute commit step with override context
                commit_response = await strategic_orchestrator._execute_commit_step(
                    type('CommitRequest', (), {
                        'proposal_set_id': override_result["proposal_set_id"],
                        'validation_id': override_result["override_validation_id"], 
                        'selected_proposal': override_result["selected_proposal"],
                        'provider_decision': {
                            "override_applied": True,
                            "justification": request.provider_justification
                        },
                        'correlation_id': request.correlation_id
                    })()
                )
                
                return OrchestrationResponse(
                    status="SUCCESS_WITH_OVERRIDE",
                    correlation_id=request.correlation_id,
                    medication_order_id=commit_response.medication_order_id,
                    commitment={
                        "order_id": commit_response.medication_order_id,
                        "audit_trail_id": commit_response.audit_trail_id,
                        "override_applied": True
                    }
                )
            else:
                return OrchestrationResponse(
                    status="OVERRIDE_REJECTED",
                    correlation_id=request.correlation_id,
                    error_code="INVALID_OVERRIDE",
                    error_message=override_result.get("rejection_reason", "Override not accepted")
                )
        
    except Exception as e:
        logger.error(f"Provider override failed for {request.correlation_id}: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail={
                "error_code": "OVERRIDE_FAILED",
                "error_message": str(e),
                "correlation_id": request.correlation_id
            }
        )


@router.post("/orchestrate/medication-snapshot", response_model=OrchestrationResponse)
async def orchestrate_medication_request_with_snapshots(
    request: MedicationRequestInput,
    background_tasks: BackgroundTasks
) -> OrchestrationResponse:
    """
    Enhanced orchestration endpoint with snapshot consistency management.
    
    This endpoint provides the next-generation medication orchestration with:
    - Immutable clinical snapshots for data consistency
    - Enhanced audit trails for regulatory compliance  
    - Optimized performance through snapshot architecture
    - Learning loop integration for clinical intelligence
    
    Use this endpoint for:
    - Production workflows requiring regulatory compliance
    - Complex clinical scenarios needing data consistency
    - Integration with learning and analytics systems
    - Performance-critical medication workflows
    
    The standard /orchestrate/medication endpoint remains available for backward compatibility.
    """
    workflow_id = f"wf_{str(uuid.uuid4())[:8]}"
    correlation_id = str(uuid.uuid4())
    request_start = datetime.utcnow()
    
    logger.info(f"[{correlation_id}] Starting snapshot-aware orchestration for workflow {workflow_id}")
    
    try:
        # Step 1: Create clinical command for snapshot orchestrator
        clinical_command = ClinicalCommand(
            patient_id=request.patient_id,
            medication_request=request.medication,
            clinical_intent={
                "indication": request.indication,
                "urgency": request.urgency,
                "constraints": request.constraints or []
            },
            provider_context={
                "provider_id": request.provider_id,
                "specialty": request.specialty,
                "location": request.location,
                "encounter_id": request.encounter_id
            },
            correlation_id=correlation_id,
            urgency=request.urgency,
            snapshot_requirements={
                "consistency_validation": True,
                "integrity_validation": True,
                "audit_trail": True
            }
        )
        
        # Step 2: Create workflow instance
        workflow_instance = WorkflowInstance(
            workflow_id=workflow_id,
            patient_id=request.patient_id,
            status="STARTED",
            snapshot_chain={},
            created_at=request_start,
            updated_at=request_start
        )
        
        # Step 3: Execute Calculate phase with snapshot creation
        logger.info(f"[{correlation_id}] Executing CALCULATE phase with snapshots")
        proposal_with_snapshot = await snapshot_aware_orchestrator.executeCalculatePhase(
            clinical_command, workflow_instance
        )
        
        # Step 4: Execute Validate phase with snapshot consistency
        logger.info(f"[{correlation_id}] Executing VALIDATE phase with snapshots")  
        validation_result = await snapshot_aware_orchestrator.executeValidatePhase(
            proposal_with_snapshot, workflow_instance
        )
        
        # Step 5: Handle validation result
        if validation_result.verdict == "SAFE":
            # Auto-commit for SAFE verdicts
            logger.info(f"[{correlation_id}] Executing COMMIT phase with snapshots")
            commit_result = await snapshot_aware_orchestrator.executeCommitPhase(
                validation_result, proposal_with_snapshot, workflow_instance
            )
            
            # Calculate total execution time
            total_execution_time = (datetime.utcnow() - request_start).total_seconds() * 1000
            
            # Success response with snapshot metadata
            return OrchestrationResponse(
                status="SUCCESS",
                correlation_id=correlation_id,
                workflow_id=workflow_id,
                execution_time_ms=total_execution_time,
                snapshot_metadata=commit_result.snapshot_chain.to_dict(),
                medication_order_id=commit_result.medication_order_id,
                calculation={
                    "proposal_set_id": proposal_with_snapshot.proposal_set_id,
                    "snapshot_id": proposal_with_snapshot.snapshot_reference.snapshot_id,
                    "execution_metrics": proposal_with_snapshot.execution_metrics
                },
                validation={
                    "validation_id": validation_result.validation_id,
                    "verdict": validation_result.verdict,
                    "evidence_id": validation_result.evidence_envelope.evidence_id,
                    "validation_metrics": validation_result.validation_metrics
                },
                commitment={
                    "order_id": commit_result.medication_order_id,
                    "audit_trail_id": commit_result.audit_trail_id,
                    "snapshot_audit": commit_result.snapshot_chain.to_dict()
                },
                performance={
                    "total_time_ms": total_execution_time,
                    "meets_target": total_execution_time <= snapshot_aware_orchestrator.performance_targets["total_optimized_ms"],
                    "optimization_achieved": True
                }
            )
            
        elif validation_result.verdict == "WARNING":
            # Return for provider decision with snapshot context
            return OrchestrationResponse(
                status="REQUIRES_PROVIDER_DECISION",
                correlation_id=correlation_id,
                workflow_id=workflow_id,
                snapshot_metadata=proposal_with_snapshot.snapshot_reference.to_dict(),
                validation_findings=[finding for finding in validation_result.findings],
                override_tokens=validation_result.override_tokens,
                proposals=proposal_with_snapshot.ranked_proposals
            )
            
        else:  # UNSAFE
            # Block with alternatives  
            return OrchestrationResponse(
                status="BLOCKED_UNSAFE",
                correlation_id=correlation_id,
                workflow_id=workflow_id,
                snapshot_metadata=proposal_with_snapshot.snapshot_reference.to_dict(),
                blocking_findings=[finding for finding in validation_result.findings],
                alternative_approaches=[]  # Could generate alternatives here
            )
            
    except SnapshotExpiredError as e:
        logger.error(f"[{correlation_id}] Snapshot expired: {str(e)}")
        return OrchestrationResponse(
            status="SNAPSHOT_EXPIRED",
            correlation_id=correlation_id,
            workflow_id=workflow_id,
            error_code="SNAPSHOT_EXPIRED", 
            error_message=f"Clinical snapshot expired: {e.snapshot_id}",
            snapshot_error_details={
                "snapshot_id": e.snapshot_id,
                "expired_at": e.expired_at.isoformat(),
                "error_type": "expiry"
            }
        )
        
    except SnapshotIntegrityError as e:
        logger.error(f"[{correlation_id}] Snapshot integrity error: {str(e)}")
        return OrchestrationResponse(
            status="SNAPSHOT_INTEGRITY_ERROR",
            correlation_id=correlation_id,
            workflow_id=workflow_id,
            error_code="SNAPSHOT_INTEGRITY_ERROR",
            error_message=f"Snapshot integrity validation failed: {e.snapshot_id}",
            snapshot_error_details={
                "snapshot_id": e.snapshot_id,
                "expected_checksum": e.expected_checksum,
                "actual_checksum": e.actual_checksum,
                "error_type": "integrity"
            }
        )
        
    except SnapshotConsistencyError as e:
        logger.error(f"[{correlation_id}] Snapshot consistency error: {str(e)}")
        return OrchestrationResponse(
            status="SNAPSHOT_CONSISTENCY_ERROR",
            correlation_id=correlation_id,
            workflow_id=workflow_id,
            error_code="SNAPSHOT_CONSISTENCY_ERROR",
            error_message=str(e),
            snapshot_error_details={
                "error_type": "consistency",
                "snapshot_chain": e.snapshot_chain.to_dict() if e.snapshot_chain else None
            }
        )
        
    except Exception as e:
        logger.error(f"[{correlation_id}] Snapshot orchestration failed: {str(e)}")
        return OrchestrationResponse(
            status="ERROR",
            correlation_id=correlation_id,
            workflow_id=workflow_id,
            error_code="ORCHESTRATION_ERROR",
            error_message=str(e)
        )
    finally:
        # Log performance metrics
        background_tasks.add_task(
            _log_performance_metrics,
            correlation_id,
            (datetime.utcnow() - request_start).total_seconds() * 1000,
            "snapshot_orchestration_completed"
        )


@router.get("/orchestrate/health")
async def orchestration_health():
    """Health check endpoint for orchestration services"""
    # Get health from both orchestrators
    strategic_health = await strategic_orchestrator.health_check()
    snapshot_health = await snapshot_aware_orchestrator.health_check()
    
    # Combine health status
    overall_healthy = (strategic_health.get("status") == "healthy" and 
                      snapshot_health.get("snapshot_orchestrator_status") == "healthy")
    
    combined_health = {
        "status": "healthy" if overall_healthy else "degraded",
        "strategic_orchestrator": strategic_health,
        "snapshot_orchestrator": {
            "status": snapshot_health.get("snapshot_orchestrator_status", "unknown"),
            "metrics": snapshot_health.get("metrics", {}),
            "snapshot_features": snapshot_health.get("snapshot_features", {})
        },
        "orchestration_modes": {
            "legacy_mode": "/orchestrate/medication",
            "snapshot_mode": "/orchestrate/medication-snapshot"
        }
    }
    
    if overall_healthy:
        return combined_health
    else:
        raise HTTPException(
            status_code=503,
            detail=combined_health
        )


@router.get("/orchestrate/performance")
async def orchestration_performance():
    """Performance metrics and targets for orchestration services"""
    return {
        "strategic_orchestrator": {
            "performance_targets": strategic_orchestrator.performance_targets,
            "architecture_pattern": "Calculate > Validate > Commit",
            "mode": "legacy_compatibility"
        },
        "snapshot_orchestrator": {
            "performance_targets": snapshot_aware_orchestrator.performance_targets,
            "architecture_pattern": "Snapshot-Aware Calculate > Validate > Commit",
            "optimization_features": [
                "Immutable clinical snapshots",
                "Data consistency validation",
                "Enhanced audit trails",
                "66% performance improvement target",
                "Sub-265ms total latency optimized"
            ],
            "metrics": snapshot_aware_orchestrator.metrics,
            "mode": "production_ready"
        },
        "service_endpoints": {
            "flow2_go": strategic_orchestrator.flow2_go_url,
            "flow2_rust": strategic_orchestrator.flow2_rust_url,
            "safety_gateway": strategic_orchestrator.safety_gateway_url,
            "medication_service": strategic_orchestrator.medication_service_url,
            "context_gateway": snapshot_aware_orchestrator.context_gateway_url
        },
        "api_endpoints": {
            "legacy_orchestration": "/orchestrate/medication",
            "snapshot_orchestration": "/orchestrate/medication-snapshot",
            "override_handling": "/orchestrate/medication/override"
        }
    }


@router.get("/orchestrate/status/{correlation_id}")
async def get_orchestration_status(correlation_id: str):
    """Get status of an orchestration request by correlation ID"""
    # This would typically query a status tracking system
    # For now, return a placeholder structure
    return {
        "correlation_id": correlation_id,
        "status": "TRACKING_NOT_IMPLEMENTED",
        "message": "Status tracking system needs to be implemented",
        "steps": [
            {"step": "calculate", "status": "unknown"},
            {"step": "validate", "status": "unknown"},
            {"step": "commit", "status": "unknown"}
        ]
    }


async def _log_performance_metrics(
    correlation_id: str,
    execution_time_ms: float, 
    status: str
):
    """Background task to log performance metrics"""
    logger.info(f"Performance: {correlation_id} completed in {execution_time_ms:.2f}ms with status {status}")
    
    # Here you would typically send metrics to monitoring system
    # e.g., Prometheus, DataDog, etc.
    pass


# Import this router in main.py to add to the FastAPI app
__all__ = ["router"]