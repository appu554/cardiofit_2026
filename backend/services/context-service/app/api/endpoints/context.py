"""
Context Service REST API Endpoints

This module provides REST API endpoints for the Context Service
that other services can call directly for clinical context data.
"""

from fastapi import APIRouter, HTTPException, Query, Path
from typing import Dict, List, Any, Optional
import logging
from datetime import datetime

# Import the context assembly service
from app.services.context_assembly_service import ContextAssemblyService
from app.models.context_models import DataPoint, DataSourceType, ContextRecipe

logger = logging.getLogger(__name__)

router = APIRouter()

# Global context assembly service instance
context_service = ContextAssemblyService()


@router.get("/patient/{patient_id}/context")
async def get_patient_context(
    patient_id: str = Path(..., description="Patient ID"),
    fields: Optional[str] = Query(default=None, description="Comma-separated list of fields to include"),
    include_demographics: bool = Query(default=True, description="Include patient demographics"),
    include_medications: bool = Query(default=True, description="Include current medications"),
    include_conditions: bool = Query(default=True, description="Include active conditions"),
    include_observations: bool = Query(default=True, description="Include recent observations"),
    include_encounters: bool = Query(default=False, description="Include recent encounters")
):
    """
    Get comprehensive clinical context for a patient.
    
    This endpoint assembles clinical context from multiple data sources
    and returns a unified view of the patient's clinical information.
    
    Args:
        patient_id: The patient ID
        fields: Specific fields to include (optional)
        include_*: Flags to control what data to include
        
    Returns:
        Comprehensive clinical context for the patient
    """
    try:
        logger.info(f"🔍 Getting clinical context for patient: {patient_id}")
        
        # Build data points based on requested information
        data_points = []
        
        if include_demographics:
            data_points.append(DataPoint(
                name="demographics",
                source_type=DataSourceType.PATIENT_SERVICE,
                fields=["id", "name", "gender", "birthDate", "address", "telecom"] if not fields else fields.split(","),
                required=True
            ))
        
        if include_medications:
            data_points.append(DataPoint(
                name="medications",
                source_type=DataSourceType.MEDICATION_SERVICE,
                fields=["medicationCodeableConcept", "status", "dosageInstruction", "effectiveDateTime"],
                required=False
            ))
        
        if include_conditions:
            data_points.append(DataPoint(
                name="conditions",
                source_type=DataSourceType.CONDITION_SERVICE,
                fields=["code", "clinicalStatus", "verificationStatus", "onsetDateTime"],
                required=False
            ))
        
        if include_observations:
            data_points.append(DataPoint(
                name="observations",
                source_type=DataSourceType.LAB_SERVICE,  # Use LAB_SERVICE instead of OBSERVATION_SERVICE
                fields=["code", "valueQuantity", "effectiveDateTime", "status"],
                required=False
            ))
        
        if include_encounters:
            data_points.append(DataPoint(
                name="encounters",
                source_type=DataSourceType.ENCOUNTER_SERVICE,
                fields=["status", "class", "period", "reasonCode"],
                required=False
            ))
        
        # Create a simple recipe from the data points
        recipe = ContextRecipe(
            recipe_id=f"rest-api-{patient_id}",
            recipe_name="REST API Context Request",
            version="1.0",
            clinical_scenario="REST API Request",
            workflow_category="command_initiated",
            execution_pattern="optimistic",
            required_data_points=data_points
        )

        # Assemble the clinical context
        context_result = await context_service.assemble_context(
            patient_id=patient_id,
            recipe=recipe
        )
        
        logger.info(f"✅ Successfully assembled clinical context for patient {patient_id}")
        
        return {
            "patient_id": patient_id,
            "context": context_result.assembled_data,
            "metadata": {
                "sources": [
                    {
                        "data_point": data_point_name,
                        "source_type": metadata.source_type.value,
                        "endpoint": metadata.source_endpoint,
                        "retrieved_at": metadata.retrieved_at.isoformat(),
                        "response_time_ms": metadata.response_time_ms,
                        "completeness": metadata.completeness
                    }
                    for data_point_name, metadata in context_result.source_metadata.items()
                ],
                "assembly_time_ms": context_result.assembly_duration_ms,
                "cache_hit": context_result.cache_hit,
                "assembled_at": datetime.utcnow().isoformat()
            }
        }
        
    except ValueError as e:
        logger.error(f"❌ Validation error for patient {patient_id}: {str(e)}")
        raise HTTPException(status_code=400, detail=str(e))
    except Exception as e:
        logger.error(f"❌ Error getting context for patient {patient_id}: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error assembling clinical context: {str(e)}")


@router.get("/patient/{patient_id}/demographics")
async def get_patient_demographics(
    patient_id: str = Path(..., description="Patient ID")
):
    """
    Get patient demographics only.
    
    Args:
        patient_id: The patient ID
        
    Returns:
        Patient demographic information
    """
    try:
        logger.info(f"🔍 Getting demographics for patient: {patient_id}")
        
        # Create data point for demographics only
        data_points = [DataPoint(
            name="demographics",
            source_type=DataSourceType.PATIENT_SERVICE,
            fields=["id", "name", "gender", "birthDate", "address", "telecom"],
            required=True
        )]
        
        # Assemble the context
        context_result = await context_service.assemble_clinical_context(
            patient_id=patient_id,
            data_points=data_points
        )
        
        logger.info(f"✅ Successfully retrieved demographics for patient {patient_id}")
        
        return {
            "patient_id": patient_id,
            "demographics": context_result.assembled_data,
            "metadata": {
                "retrieved_at": datetime.utcnow().isoformat(),
                "source": "patient-service",
                "cache_hit": context_result.cache_hit
            }
        }
        
    except Exception as e:
        logger.error(f"❌ Error getting demographics for patient {patient_id}: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error retrieving demographics: {str(e)}")


@router.get("/patient/{patient_id}/medications")
async def get_patient_medications(
    patient_id: str = Path(..., description="Patient ID"),
    status: Optional[str] = Query(default="active", description="Medication status filter")
):
    """
    Get patient medications only.
    
    Args:
        patient_id: The patient ID
        status: Medication status filter
        
    Returns:
        Patient medication information
    """
    try:
        logger.info(f"🔍 Getting medications for patient: {patient_id}")
        
        # Create data point for medications only
        data_points = [DataPoint(
            name="medications",
            source_type=DataSourceType.MEDICATION_SERVICE,
            fields=["medicationCodeableConcept", "status", "dosageInstruction", "effectiveDateTime"],
            required=True
        )]
        
        # Assemble the context
        context_result = await context_service.assemble_clinical_context(
            patient_id=patient_id,
            data_points=data_points
        )
        
        logger.info(f"✅ Successfully retrieved medications for patient {patient_id}")
        
        return {
            "patient_id": patient_id,
            "medications": context_result.assembled_data,
            "metadata": {
                "retrieved_at": datetime.utcnow().isoformat(),
                "source": "medication-service",
                "cache_hit": context_result.cache_hit,
                "status_filter": status
            }
        }
        
    except Exception as e:
        logger.error(f"❌ Error getting medications for patient {patient_id}: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error retrieving medications: {str(e)}")


@router.get("/patient/{patient_id}/summary")
async def get_patient_summary(
    patient_id: str = Path(..., description="Patient ID")
):
    """
    Get a quick patient summary with essential information.
    
    Args:
        patient_id: The patient ID
        
    Returns:
        Patient summary with key clinical information
    """
    try:
        logger.info(f"🔍 Getting summary for patient: {patient_id}")
        
        # Create data points for summary (demographics + key clinical data)
        data_points = [
            DataPoint(
                name="demographics",
                source_type=DataSourceType.PATIENT_SERVICE,
                fields=["id", "name", "gender", "birthDate"],
                required=True
            ),
            DataPoint(
                name="medications",
                source_type=DataSourceType.MEDICATION_SERVICE,
                fields=["medicationCodeableConcept", "status"],
                required=False
            ),
            DataPoint(
                name="conditions",
                source_type=DataSourceType.CONDITION_SERVICE,
                fields=["code", "clinicalStatus"],
                required=False
            )
        ]
        
        # Assemble the context
        context_result = await context_service.assemble_clinical_context(
            patient_id=patient_id,
            data_points=data_points
        )
        
        logger.info(f"✅ Successfully retrieved summary for patient {patient_id}")
        
        return {
            "patient_id": patient_id,
            "summary": context_result.assembled_data,
            "metadata": {
                "retrieved_at": datetime.utcnow().isoformat(),
                "sources_used": len(context_result.source_metadata),
                "cache_hit": context_result.cache_hit,
                "assembly_time_ms": context_result.assembly_time_ms
            }
        }
        
    except Exception as e:
        logger.error(f"❌ Error getting summary for patient {patient_id}: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error retrieving patient summary: {str(e)}")


@router.get("/status")
async def get_context_service_status():
    """
    Get Context Service status and health information.
    
    Returns:
        Service status and configuration information
    """
    try:
        return {
            "service": "context-service",
            "status": "healthy",
            "version": "1.0.0",
            "endpoints": [
                "/api/context/patient/{patient_id}/context",
                "/api/context/patient/{patient_id}/demographics", 
                "/api/context/patient/{patient_id}/medications",
                "/api/context/patient/{patient_id}/summary",
                "/api/context/status"
            ],
            "data_sources": [
                "patient-service",
                "medication-service", 
                "condition-service",
                "observation-service",
                "encounter-service"
            ],
            "features": [
                "clinical-context-assembly",
                "multi-source-data-aggregation",
                "intelligent-caching",
                "rest-api"
            ],
            "timestamp": datetime.utcnow().isoformat()
        }
        
    except Exception as e:
        logger.error(f"❌ Error getting service status: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error getting service status: {str(e)}")

@router.get("/recipes")
async def get_available_recipes():
    """
    Get all available clinical context recipes.

    Returns:
        List of available recipes with their metadata
    """
    try:
        from app.services.recipe_management_service import RecipeManagementService

        recipe_service = RecipeManagementService()

        recipes_info = []
        for recipe_id, recipe in recipe_service.loaded_recipes.items():
            recipes_info.append({
                "recipe_id": recipe.recipe_id,
                "recipe_name": recipe.recipe_name,
                "version": recipe.version,
                "clinical_scenario": recipe.clinical_scenario,
                "workflow_category": recipe.workflow_category,
                "execution_pattern": recipe.execution_pattern,
                "sla_ms": recipe.sla_ms,
                "data_points_count": len(recipe.required_data_points),
                "governance_approved": getattr(recipe, 'governance_approved', True),
                "qos_tier": getattr(recipe, 'qos_tier', 'standard')
            })

        return {
            "status": "success",
            "total_recipes": len(recipes_info),
            "recipes": recipes_info,
            "timestamp": datetime.utcnow().isoformat()
        }

    except Exception as e:
        logger.error(f"❌ Error getting available recipes: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error retrieving recipes: {str(e)}")


@router.get("/recipes/{recipe_id}")
async def get_recipe_details(recipe_id: str):
    """
    Get detailed information about a specific recipe.

    Args:
        recipe_id: The recipe ID to retrieve

    Returns:
        Detailed recipe information
    """
    try:
        from app.services.recipe_management_service import RecipeManagementService

        recipe_service = RecipeManagementService()
        recipe = await recipe_service.load_recipe(recipe_id)

        if not recipe:
            raise HTTPException(status_code=404, detail=f"Recipe {recipe_id} not found")

        return {
            "status": "success",
            "recipe": {
                "recipe_id": recipe.recipe_id,
                "recipe_name": recipe.recipe_name,
                "version": recipe.version,
                "clinical_scenario": recipe.clinical_scenario,
                "workflow_category": recipe.workflow_category,
                "execution_pattern": recipe.execution_pattern,
                "sla_ms": recipe.sla_ms,
                "required_data_points": [
                    {
                        "name": dp.name,
                        "source_type": dp.source_type.value if hasattr(dp.source_type, 'value') else str(dp.source_type),
                        "fields": dp.fields,
                        "required": dp.required,
                        "timeout_ms": getattr(dp, 'timeout_ms', None),
                        "max_age_hours": getattr(dp, 'max_age_hours', None)
                    }
                    for dp in recipe.required_data_points
                ],
                "optional_data_points": [
                    {
                        "name": dp.name,
                        "source_type": dp.source_type.value if hasattr(dp.source_type, 'value') else str(dp.source_type),
                        "fields": dp.fields,
                        "required": dp.required,
                        "timeout_ms": getattr(dp, 'timeout_ms', None),
                        "max_age_hours": getattr(dp, 'max_age_hours', None)
                    }
                    for dp in getattr(recipe, 'optional_data_points', [])
                ],
                "governance_approved": getattr(recipe, 'governance_approved', True),
                "qos_tier": getattr(recipe, 'qos_tier', 'standard'),
                "cache_duration_seconds": getattr(recipe, 'cache_duration_seconds', 300)
            },
            "timestamp": datetime.utcnow().isoformat()
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"❌ Error getting recipe details for {recipe_id}: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error retrieving recipe details: {str(e)}")


@router.get("/patient/{patient_id}/recipe/{recipe_id}")
async def get_context_by_recipe(
    patient_id: str = Path(..., description="Patient ID"),
    recipe_id: str = Path(..., description="Recipe ID"),
    provider_id: str = Query(None, description="Provider ID"),
    encounter_id: str = Query(None, description="Encounter ID"),
    force_refresh: bool = Query(False, description="Force refresh from sources")
):
    """
    Get clinical context for a patient using a specific recipe.

    Args:
        patient_id: The patient ID
        recipe_id: The recipe ID to use for context assembly
        provider_id: Optional provider ID
        encounter_id: Optional encounter ID
        force_refresh: Force refresh from data sources

    Returns:
        Assembled clinical context based on the recipe
    """
    try:
        from app.services.recipe_management_service import RecipeManagementService

        logger.info(f"🔍 Getting context for patient {patient_id} using recipe {recipe_id}")

        # Load the recipe
        recipe_service = RecipeManagementService()
        recipe = await recipe_service.load_recipe(recipe_id)

        if not recipe:
            raise HTTPException(status_code=404, detail=f"Recipe {recipe_id} not found")

        # Assemble context using the recipe
        context_result = await context_service.assemble_context(
            patient_id=patient_id,
            recipe=recipe,
            provider_id=provider_id,
            encounter_id=encounter_id,
            force_refresh=force_refresh
        )

        logger.info(f"✅ Successfully assembled context for patient {patient_id} using recipe {recipe_id}")

        return {
            "patient_id": patient_id,
            "recipe_id": recipe_id,
            "context_id": context_result.context_id,
            "context": context_result.assembled_data,
            "completeness_score": context_result.completeness_score,
            "safety_flags": [
                {
                    "flag_type": flag.flag_type.value if hasattr(flag.flag_type, 'value') else str(flag.flag_type),
                    "severity": flag.severity.value if hasattr(flag.severity, 'value') else str(flag.severity),
                    "message": flag.message,
                    "data_point": flag.data_point,
                    "timestamp": flag.timestamp.isoformat() if flag.timestamp else None
                }
                for flag in context_result.safety_flags
            ],
            "metadata": {
                "recipe_used": recipe_id,
                "recipe_version": recipe.version,
                "assembled_at": context_result.assembled_at.isoformat() if context_result.assembled_at else datetime.utcnow().isoformat(),
                "assembly_duration_ms": context_result.assembly_duration_ms,
                "cache_hit": context_result.cache_hit,
                "sources_used": len(context_result.source_metadata),
                "provider_id": provider_id,
                "encounter_id": encounter_id,
                "force_refresh": force_refresh
            }
        }

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"❌ Error assembling context for patient {patient_id} with recipe {recipe_id}: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error assembling context: {str(e)}")


@router.get("/test-apollo-federation/{patient_id}")
async def test_apollo_federation(patient_id: str):
    """Test Apollo Federation integration."""
    try:
        from app.clients.apollo_federation_client import get_apollo_federation_client

        apollo_client = get_apollo_federation_client()

        async with apollo_client as client:
            # Test patient data fetch
            patient_data = await client.get_patient_data(patient_id)

            return {
                "status": "success",
                "message": "Apollo Federation test completed",
                "patient_id": patient_id,
                "patient_data_found": bool(patient_data),
                "apollo_federation_url": "http://localhost:4000/graphql",
                "patient_data_sample": patient_data if patient_data else None
            }

    except Exception as e:
        logger.error(f"❌ Apollo Federation test failed: {e}")
        return {
            "status": "error",
            "message": f"Apollo Federation test failed: {str(e)}",
            "patient_id": patient_id
        }
