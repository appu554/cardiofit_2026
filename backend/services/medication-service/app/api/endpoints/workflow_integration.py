"""
Workflow Engine Integration API

This module provides API endpoints specifically designed for Workflow Engine integration,
including Safety Gateway Platform orchestration and comprehensive medication safety validation.

The Workflow Engine is responsible for:
1. Safety orchestration via Safety Gateway Platform
2. Clinical workflow management
3. Multi-service coordination

The Medication Service provides:
1. Pharmaceutical intelligence
2. Clinical recipe execution
3. Context-aware medication assessment
"""

import logging
from typing import Dict, List, Any, Optional
from fastapi import APIRouter, HTTPException, Depends
from pydantic import BaseModel, Field
from datetime import datetime

from app.domain.services.recipe_orchestrator import (
    RecipeOrchestrator, 
    MedicationSafetyRequest, 
    Flow2Result
)

logger = logging.getLogger(__name__)

router = APIRouter()


class WorkflowMedicationRequest(BaseModel):
    """Medication request from Workflow Engine with safety orchestration context"""
    patient_id: str = Field(..., description="Patient identifier")
    medication: Dict[str, Any] = Field(..., description="Medication details")
    provider_id: Optional[str] = Field(None, description="Provider identifier")
    encounter_id: Optional[str] = Field(None, description="Encounter identifier")
    workflow_id: str = Field(..., description="Workflow instance identifier")
    action_type: str = Field("prescribe", description="Clinical action type")
    urgency: str = Field("routine", description="Request urgency level")
    
    # Workflow Engine context
    safety_orchestration_enabled: bool = Field(True, description="Enable Safety Gateway Platform integration")
    workflow_step: str = Field("medication_validation", description="Current workflow step")
    previous_validations: List[Dict[str, Any]] = Field(default_factory=list, description="Previous validation results")


class WorkflowMedicationResponse(BaseModel):
    """Response for Workflow Engine with pharmaceutical intelligence and optional safety validation"""
    request_id: str
    workflow_id: str
    patient_id: str
    overall_status: str  # SAFE, WARNING, UNSAFE, ERROR
    pharmaceutical_assessment: Dict[str, Any]
    safety_validation: Optional[Dict[str, Any]] = None
    context_completeness: float
    execution_time_ms: float
    recommendations: List[str]
    next_workflow_actions: List[str]
    timestamp: datetime


# Dependency for Workflow Engine integration
def get_workflow_orchestrator() -> RecipeOrchestrator:
    """
    Get Recipe Orchestrator configured for Workflow Engine integration
    
    This enables Safety Gateway Platform integration when requested by Workflow Engine
    """
    return RecipeOrchestrator(
        context_service_url="http://localhost:8016",
        enable_safety_gateway=True,  # Enable for Workflow Engine
        safety_gateway_url="localhost:8030"
    )


@router.post("/medication-validation", response_model=WorkflowMedicationResponse)
async def validate_medication_for_workflow(
    request: WorkflowMedicationRequest,
    orchestrator: RecipeOrchestrator = Depends(get_workflow_orchestrator)
):
    """
    Medication validation endpoint for Workflow Engine integration
    
    This endpoint provides:
    1. Pharmaceutical intelligence (always)
    2. Safety Gateway Platform integration (when enabled)
    3. Workflow-specific recommendations
    4. Next workflow action suggestions
    """
    try:
        logger.info(f"🔄 Workflow Engine medication validation request")
        logger.info(f"   Workflow ID: {request.workflow_id}")
        logger.info(f"   Patient: {request.patient_id}")
        logger.info(f"   Medication: {request.medication.get('name', 'Unknown')}")
        logger.info(f"   Safety Orchestration: {request.safety_orchestration_enabled}")
        
        # Configure orchestrator based on workflow requirements
        orchestrator.enable_safety_gateway = request.safety_orchestration_enabled
        
        # Convert to internal request format
        safety_request = MedicationSafetyRequest(
            patient_id=request.patient_id,
            medication=request.medication,
            provider_id=request.provider_id,
            encounter_id=request.encounter_id,
            action_type=request.action_type,
            urgency=request.urgency,
            workflow_id=request.workflow_id
        )
        
        # Execute Flow 2 with workflow configuration
        flow2_result = await orchestrator.execute_medication_safety(safety_request)
        
        # Extract pharmaceutical assessment
        pharmaceutical_assessment = {
            "context_recipe_used": flow2_result.context_recipe_used,
            "clinical_recipes_executed": flow2_result.clinical_recipes_executed,
            "clinical_results": [
                {
                    "recipe_id": result.recipe_id,
                    "recipe_name": result.recipe_name,
                    "status": result.overall_status,
                    "execution_time_ms": result.execution_time_ms,
                    "validations_count": len(result.validations),
                    "clinical_decision_support": result.clinical_decision_support
                }
                for result in flow2_result.clinical_results
                if "safety_gateway_platform" not in result.recipe_id
            ]
        }
        
        # Extract safety validation (if performed)
        safety_validation = None
        if request.safety_orchestration_enabled:
            safety_results = [
                result for result in flow2_result.clinical_results
                if "safety_gateway_platform" in result.recipe_id
            ]
            if safety_results:
                safety_result = safety_results[0]
                safety_validation = {
                    "status": safety_result.overall_status,
                    "risk_score": safety_result.performance_metrics.get('risk_score', 0.0),
                    "confidence": safety_result.performance_metrics.get('confidence', 0.0),
                    "violations": [v['message'] for v in safety_result.validations if not v['passed'] and v['severity'] == 'HIGH'],
                    "warnings": [v['message'] for v in safety_result.validations if not v['passed'] and v['severity'] == 'MEDIUM'],
                    "engines_executed": safety_result.performance_metrics.get('engines_executed', [])
                }
        
        # Generate workflow-specific recommendations
        recommendations = _generate_workflow_recommendations(flow2_result, request)
        
        # Determine next workflow actions
        next_actions = _determine_next_workflow_actions(flow2_result, request)
        
        response = WorkflowMedicationResponse(
            request_id=flow2_result.request_id,
            workflow_id=request.workflow_id,
            patient_id=request.patient_id,
            overall_status=flow2_result.overall_safety_status,
            pharmaceutical_assessment=pharmaceutical_assessment,
            safety_validation=safety_validation,
            context_completeness=flow2_result.context_completeness_score,
            execution_time_ms=flow2_result.execution_time_ms,
            recommendations=recommendations,
            next_workflow_actions=next_actions,
            timestamp=datetime.now()
        )
        
        logger.info(f"✅ Workflow Engine validation completed")
        logger.info(f"   Overall Status: {response.overall_status}")
        logger.info(f"   Safety Validation: {'Enabled' if safety_validation else 'Disabled'}")
        logger.info(f"   Next Actions: {len(next_actions)}")
        
        return response
        
    except Exception as e:
        logger.error(f"❌ Workflow Engine medication validation failed: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=f"Medication validation failed: {str(e)}"
        )


@router.get("/workflow-health")
async def workflow_integration_health():
    """Health check for Workflow Engine integration"""
    try:
        orchestrator = get_workflow_orchestrator()
        health_status = await orchestrator.health_check()
        
        return {
            "status": "healthy",
            "timestamp": datetime.now(),
            "components": health_status,
            "workflow_integration": {
                "safety_gateway_enabled": orchestrator.enable_safety_gateway,
                "safety_gateway_client": orchestrator.safety_gateway_client is not None
            }
        }
        
    except Exception as e:
        logger.error(f"❌ Workflow integration health check failed: {e}")
        return {
            "status": "unhealthy",
            "timestamp": datetime.now(),
            "error": str(e)
        }


def _generate_workflow_recommendations(flow2_result: Flow2Result, request: WorkflowMedicationRequest) -> List[str]:
    """Generate workflow-specific recommendations"""
    recommendations = []
    
    if flow2_result.overall_safety_status == "UNSAFE":
        recommendations.extend([
            "Halt medication administration workflow",
            "Require physician review and override",
            "Consider alternative medications",
            "Document safety concerns in patient record"
        ])
    elif flow2_result.overall_safety_status == "WARNING":
        recommendations.extend([
            "Proceed with enhanced monitoring",
            "Alert prescribing physician",
            "Schedule follow-up assessment",
            "Document monitoring plan"
        ])
    else:
        recommendations.extend([
            "Proceed with standard medication workflow",
            "Continue routine monitoring",
            "Document successful validation"
        ])
    
    # Add context-specific recommendations
    if flow2_result.context_completeness_score < 0.7:
        recommendations.append("Consider gathering additional clinical context")
    
    return recommendations


def _determine_next_workflow_actions(flow2_result: Flow2Result, request: WorkflowMedicationRequest) -> List[str]:
    """Determine next workflow actions based on validation results"""
    next_actions = []
    
    if flow2_result.overall_safety_status == "UNSAFE":
        next_actions.extend([
            "physician_review_required",
            "safety_override_evaluation",
            "alternative_medication_search"
        ])
    elif flow2_result.overall_safety_status == "WARNING":
        next_actions.extend([
            "enhanced_monitoring_setup",
            "physician_notification",
            "patient_counseling_required"
        ])
    else:
        next_actions.extend([
            "proceed_to_administration",
            "standard_monitoring_setup",
            "patient_education"
        ])
    
    # Add workflow-specific actions
    if request.workflow_step == "medication_validation":
        next_actions.append("move_to_administration_planning")
    
    return next_actions
