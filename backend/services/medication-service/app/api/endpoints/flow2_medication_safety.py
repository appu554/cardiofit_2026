"""
Flow 2 Medication Safety API Endpoints

This module provides REST API endpoints for Flow 2 medication safety validation,
integrating Context Service with Clinical Recipe Engine for comprehensive
medication safety assessment.

Endpoints:
- POST /flow2/medication-safety/validate - Main Flow 2 validation endpoint
- GET /flow2/medication-safety/health - Health check for Flow 2 components
- GET /flow2/medication-safety/metrics - Performance metrics
- POST /flow2/medication-safety/batch - Batch validation for multiple medications
"""

import logging
from typing import Dict, List, Any, Optional
from fastapi import APIRouter, HTTPException, Depends, BackgroundTasks
from pydantic import BaseModel, Field
from datetime import datetime

from app.domain.services.recipe_orchestrator import (
    RecipeOrchestrator, 
    MedicationSafetyRequest, 
    Flow2Result
)

logger = logging.getLogger(__name__)

router = APIRouter()

# Pydantic models for API requests/responses

class MedicationData(BaseModel):
    """Medication information for safety validation"""
    name: str = Field(..., description="Medication name")
    generic_name: Optional[str] = Field(None, description="Generic medication name")
    dose: Optional[str] = Field(None, description="Medication dose")
    frequency: Optional[str] = Field(None, description="Dosing frequency")
    route: Optional[str] = Field(None, description="Route of administration")
    therapeutic_class: Optional[str] = Field(None, description="Therapeutic class")
    is_anticoagulant: bool = Field(False, description="Is this an anticoagulant medication")
    is_chemotherapy: bool = Field(False, description="Is this a chemotherapy medication")
    requires_renal_adjustment: bool = Field(False, description="Requires renal dose adjustment")
    is_controlled_substance: bool = Field(False, description="Is this a controlled substance")
    is_high_risk: bool = Field(False, description="Is this a high-risk medication")


class Flow2ValidationRequest(BaseModel):
    """Request for Flow 2 medication safety validation"""
    patient_id: str = Field(..., description="Patient identifier")
    medication: MedicationData = Field(..., description="Medication to validate")
    provider_id: Optional[str] = Field(None, description="Provider identifier")
    encounter_id: Optional[str] = Field(None, description="Encounter identifier")
    action_type: str = Field("prescribe", description="Action type: prescribe, modify, discontinue")
    urgency: str = Field("routine", description="Urgency level: routine, urgent, emergency")
    workflow_id: Optional[str] = Field(None, description="Workflow identifier")


class Flow2ValidationResponse(BaseModel):
    """Response from Flow 2 medication safety validation"""
    request_id: str
    patient_id: str
    overall_safety_status: str
    context_recipe_used: str
    clinical_recipes_executed: List[str]
    context_completeness_score: float
    execution_time_ms: float
    safety_summary: Dict[str, Any]
    performance_metrics: Dict[str, Any]
    errors: Optional[List[str]] = None


class BatchValidationRequest(BaseModel):
    """Request for batch medication safety validation"""
    patient_id: str = Field(..., description="Patient identifier")
    medications: List[MedicationData] = Field(..., description="List of medications to validate")
    provider_id: Optional[str] = Field(None, description="Provider identifier")
    encounter_id: Optional[str] = Field(None, description="Encounter identifier")
    urgency: str = Field("routine", description="Urgency level")


class HealthCheckResponse(BaseModel):
    """Health check response for Flow 2 components"""
    status: str
    timestamp: datetime
    components: Dict[str, Any]
    performance_metrics: Dict[str, Any]


# Dependency to get Recipe Orchestrator
def get_recipe_orchestrator() -> RecipeOrchestrator:
    """
    Get Recipe Orchestrator instance focused on pharmaceutical intelligence

    Safety Gateway Platform integration is disabled by default -
    Workflow Engine should handle safety orchestration
    """
    return RecipeOrchestrator(
        context_service_url="http://localhost:8016",
        enable_safety_gateway=False,  # Workflow Engine handles safety orchestration
        safety_gateway_url="localhost:8030"
    )


@router.post("/validate", response_model=Flow2ValidationResponse)
async def validate_medication_safety(
    request: Flow2ValidationRequest,
    orchestrator: RecipeOrchestrator = Depends(get_recipe_orchestrator)
):
    """
    Main Flow 2 medication safety validation endpoint
    
    This endpoint implements the complete Flow 2 workflow:
    1. Determines appropriate context recipe
    2. Retrieves optimized clinical context
    3. Executes applicable clinical recipes
    4. Returns comprehensive safety assessment
    """
    try:
        logger.info(f"🚀 Flow 2 validation request received")
        logger.info(f"   Patient: {request.patient_id}")
        logger.info(f"   Medication: {request.medication.name}")
        logger.info(f"   Urgency: {request.urgency}")
        
        # Convert API request to internal format
        safety_request = MedicationSafetyRequest(
            patient_id=request.patient_id,
            medication=request.medication.dict(),
            provider_id=request.provider_id,
            encounter_id=request.encounter_id,
            action_type=request.action_type,
            urgency=request.urgency,
            workflow_id=request.workflow_id
        )
        
        # Execute Flow 2 validation
        result = await orchestrator.execute_medication_safety(safety_request)
        
        # Convert to API response format
        response = Flow2ValidationResponse(
            request_id=result.request_id,
            patient_id=result.patient_id,
            overall_safety_status=result.overall_safety_status,
            context_recipe_used=result.context_recipe_used,
            clinical_recipes_executed=result.clinical_recipes_executed,
            context_completeness_score=result.context_completeness_score,
            execution_time_ms=result.execution_time_ms,
            safety_summary=result.safety_summary or {},
            performance_metrics=result.performance_metrics or {},
            errors=result.errors
        )
        
        logger.info(f"✅ Flow 2 validation completed")
        logger.info(f"   Status: {result.overall_safety_status}")
        logger.info(f"   Execution time: {result.execution_time_ms:.1f}ms")
        
        return response
        
    except Exception as e:
        logger.error(f"❌ Flow 2 validation failed: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=f"Flow 2 validation failed: {str(e)}"
        )


@router.post("/batch", response_model=List[Flow2ValidationResponse])
async def batch_validate_medications(
    request: BatchValidationRequest,
    background_tasks: BackgroundTasks,
    orchestrator: RecipeOrchestrator = Depends(get_recipe_orchestrator)
):
    """
    Batch validation for multiple medications
    
    Validates multiple medications for the same patient efficiently,
    with shared context optimization.
    """
    try:
        logger.info(f"🚀 Batch Flow 2 validation request received")
        logger.info(f"   Patient: {request.patient_id}")
        logger.info(f"   Medications: {len(request.medications)}")
        
        results = []
        
        # Process each medication
        for medication in request.medications:
            safety_request = MedicationSafetyRequest(
                patient_id=request.patient_id,
                medication=medication.dict(),
                provider_id=request.provider_id,
                encounter_id=request.encounter_id,
                action_type="prescribe",
                urgency=request.urgency
            )
            
            # Execute Flow 2 validation
            result = await orchestrator.execute_medication_safety(safety_request)
            
            # Convert to API response format
            response = Flow2ValidationResponse(
                request_id=result.request_id,
                patient_id=result.patient_id,
                overall_safety_status=result.overall_safety_status,
                context_recipe_used=result.context_recipe_used,
                clinical_recipes_executed=result.clinical_recipes_executed,
                context_completeness_score=result.context_completeness_score,
                execution_time_ms=result.execution_time_ms,
                safety_summary=result.safety_summary or {},
                performance_metrics=result.performance_metrics or {},
                errors=result.errors
            )
            
            results.append(response)
        
        logger.info(f"✅ Batch Flow 2 validation completed")
        logger.info(f"   Processed: {len(results)} medications")
        
        return results
        
    except Exception as e:
        logger.error(f"❌ Batch Flow 2 validation failed: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=f"Batch validation failed: {str(e)}"
        )


@router.get("/health", response_model=HealthCheckResponse)
async def health_check(
    orchestrator: RecipeOrchestrator = Depends(get_recipe_orchestrator)
):
    """
    Health check for Flow 2 components
    
    Checks the health of all Flow 2 dependencies:
    - Recipe Orchestrator
    - Context Service
    - Clinical Recipe Engine
    """
    try:
        logger.info("🔍 Flow 2 health check requested")
        
        # Get health status from orchestrator
        health_status = await orchestrator.health_check()
        
        # Get performance metrics
        performance_metrics = orchestrator.context_service_client.get_flow2_performance_metrics()
        
        response = HealthCheckResponse(
            status="healthy" if all(
                status != "error" for status in health_status.values() 
                if isinstance(status, str)
            ) else "degraded",
            timestamp=datetime.now(),
            components=health_status,
            performance_metrics=performance_metrics
        )
        
        logger.info(f"✅ Flow 2 health check completed: {response.status}")
        
        return response
        
    except Exception as e:
        logger.error(f"❌ Flow 2 health check failed: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=f"Health check failed: {str(e)}"
        )


@router.get("/metrics")
async def get_performance_metrics(
    orchestrator: RecipeOrchestrator = Depends(get_recipe_orchestrator)
):
    """
    Get Flow 2 performance metrics

    Returns detailed performance metrics for monitoring and optimization.
    """
    try:
        logger.info("📊 Flow 2 metrics requested")

        # Get performance metrics from context service client
        metrics = orchestrator.context_service_client.get_flow2_performance_metrics()

        # Add recipe engine metrics
        recipe_catalog = orchestrator.clinical_recipe_engine.get_recipe_catalog()
        metrics['clinical_recipes'] = {
            'total_registered': len(recipe_catalog),
            'recipe_catalog': recipe_catalog
        }

        logger.info(f"✅ Flow 2 metrics retrieved")

        return {
            "timestamp": datetime.now().isoformat(),
            "flow2_metrics": metrics
        }

    except Exception as e:
        logger.error(f"❌ Flow 2 metrics retrieval failed: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=f"Metrics retrieval failed: {str(e)}"
        )


@router.get("/clinical-recipes")
async def get_clinical_recipes(
    orchestrator: RecipeOrchestrator = Depends(get_recipe_orchestrator)
):
    """
    Get available clinical recipes for Context Service integration

    This endpoint is called by the Context Service to get the list of
    available clinical recipes for context optimization.
    """
    try:
        logger.info("📋 Clinical recipes requested by Context Service")

        # Get recipe catalog from clinical recipe engine
        recipe_catalog = orchestrator.clinical_recipe_engine.get_recipe_catalog()

        # Transform to format expected by Context Service
        recipes = []
        for recipe_id, recipe_info in recipe_catalog.items():
            recipes.append({
                "recipe_id": recipe_id,
                "recipe_name": recipe_info.get("name", "Unknown"),
                "description": recipe_info.get("description", ""),
                "priority": recipe_info.get("priority", 90),
                "qos_tier": recipe_info.get("qos_tier", "silver"),
                "clinical_rationale": recipe_info.get("clinical_rationale", ""),
                "service": "medication-service",
                "version": "3.0"
            })

        logger.info(f"✅ Returning {len(recipes)} clinical recipes to Context Service")

        return {
            "timestamp": datetime.now().isoformat(),
            "service": "medication-service",
            "total_recipes": len(recipes),
            "recipes": recipes
        }

    except Exception as e:
        logger.error(f"❌ Clinical recipes retrieval failed: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=f"Clinical recipes retrieval failed: {str(e)}"
        )


@router.post("/execute-clinical-recipes")
async def execute_clinical_recipes_for_context(
    request: dict,
    orchestrator: RecipeOrchestrator = Depends(get_recipe_orchestrator)
):
    """
    Execute clinical recipes for Context Service integration

    This endpoint is called by the Context Service during context assembly
    to execute specific clinical recipes and get their requirements.
    """
    try:
        logger.info("⚡ Clinical recipe execution requested by Context Service")

        patient_id = request.get("patient_id")
        medication_data = request.get("medication", {})
        recipe_ids = request.get("recipe_ids", [])

        logger.info(f"   Patient: {patient_id}")
        logger.info(f"   Medication: {medication_data.get('name', 'Unknown')}")
        logger.info(f"   Requested recipes: {len(recipe_ids)}")

        # Create a basic recipe context for analysis
        from app.domain.services.clinical_recipe_engine import RecipeContext

        context = RecipeContext(
            patient_id=patient_id,
            action_type="analyze",  # Analysis mode for Context Service
            medication_data=medication_data,
            patient_data=request.get("patient_data", {}),
            provider_data=request.get("provider_data", {}),
            encounter_data=request.get("encounter_data", {}),
            clinical_data=request.get("clinical_data", {}),
            timestamp=datetime.now()
        )

        # Get data requirements for the requested recipes
        recipe_requirements = []

        for recipe_id in recipe_ids:
            if recipe_id in orchestrator.clinical_recipe_engine.recipes:
                recipe = orchestrator.clinical_recipe_engine.recipes[recipe_id]

                # Check if recipe should trigger
                should_trigger = recipe.should_trigger(context)

                # Get recipe requirements
                requirements = {
                    "recipe_id": recipe_id,
                    "recipe_name": recipe.name,
                    "should_trigger": should_trigger,
                    "priority": recipe.priority,
                    "data_requirements": {
                        "patient_data": ["age", "weight_kg", "conditions", "allergies"],
                        "clinical_data": ["labs", "vitals", "current_medications"],
                        "provider_data": ["id", "specialty"],
                        "encounter_data": ["id", "type", "location"]
                    }
                }

                recipe_requirements.append(requirements)

        logger.info(f"✅ Analyzed {len(recipe_requirements)} clinical recipes")

        return {
            "timestamp": datetime.now().isoformat(),
            "patient_id": patient_id,
            "medication": medication_data,
            "recipe_requirements": recipe_requirements,
            "total_analyzed": len(recipe_requirements)
        }

    except Exception as e:
        logger.error(f"❌ Clinical recipe execution failed: {str(e)}")
        raise HTTPException(
            status_code=500,
            detail=f"Clinical recipe execution failed: {str(e)}"
        )
