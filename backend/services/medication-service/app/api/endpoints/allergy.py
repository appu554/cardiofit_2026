"""
Allergy and Intolerance endpoints for the Medication Service.

This module provides REST API endpoints for managing patient allergies and intolerances.
"""

from typing import List, Optional, Dict, Any
from fastapi import APIRouter, HTTPException, Depends, Query
from pydantic import BaseModel
import logging

from app.services.fhir_service_factory import get_fhir_service
from shared.auth import get_current_user_from_token

logger = logging.getLogger(__name__)

router = APIRouter()

# Public router for testing (bypasses authentication)
public_router = APIRouter()

class AllergyIntoleranceCreate(BaseModel):
    """Model for creating an allergy intolerance."""
    patient_id: str
    code: str
    display: Optional[str] = None
    system: Optional[str] = "http://snomed.info/sct"
    clinical_status: str = "active"
    verification_status: str = "confirmed"
    criticality: Optional[str] = None
    category: Optional[List[str]] = None
    reaction: Optional[List[Dict[str, Any]]] = None
    note: Optional[str] = None

class AllergyIntoleranceUpdate(BaseModel):
    """Model for updating an allergy intolerance."""
    clinical_status: Optional[str] = None
    verification_status: Optional[str] = None
    criticality: Optional[str] = None
    category: Optional[List[str]] = None
    reaction: Optional[List[Dict[str, Any]]] = None
    note: Optional[str] = None

@router.get("/patient/{patient_id}")
async def get_patient_allergies(
    patient_id: str,
    current_user: dict = Depends(get_current_user_from_token)
) -> List[Dict[str, Any]]:
    """
    Get all allergies for a specific patient.
    
    Args:
        patient_id: The patient ID
        current_user: Current authenticated user
        
    Returns:
        List of allergy intolerance resources
    """
    try:
        fhir_service = get_fhir_service()
        
        # Search for allergy intolerances for this patient
        search_params = {"patient": f"Patient/{patient_id}"}
        resources = await fhir_service.search_resources("AllergyIntolerance", search_params)
        
        logger.info(f"Retrieved {len(resources)} allergies for patient {patient_id}")
        return resources
    except Exception as e:
        logger.error(f"Error fetching patient allergies: {e}")
        raise HTTPException(status_code=500, detail=f"Error fetching allergies: {str(e)}")

@router.get("/")
async def get_allergies(
    page: int = Query(1, ge=1),
    limit: int = Query(10, ge=1, le=100),
    patient: Optional[str] = None,
    status: Optional[str] = None,
    current_user: dict = Depends(get_current_user_from_token)
) -> Dict[str, Any]:
    """
    Get allergies with pagination and filtering.
    
    Args:
        page: Page number
        limit: Items per page
        patient: Filter by patient ID
        status: Filter by clinical status
        current_user: Current authenticated user
        
    Returns:
        Paginated list of allergy resources
    """
    try:
        fhir_service = get_fhir_service()
        
        # Build search parameters
        search_params = {
            "_count": str(limit),
            "_offset": str((page - 1) * limit)
        }
        
        if patient:
            search_params["patient"] = f"Patient/{patient}"
        if status:
            search_params["clinical-status"] = status
            
        resources = await fhir_service.search_resources("AllergyIntolerance", search_params)
        
        return {
            "items": resources,
            "total": len(resources),
            "page": page,
            "count": limit
        }
    except Exception as e:
        logger.error(f"Error fetching allergies: {e}")
        raise HTTPException(status_code=500, detail=f"Error fetching allergies: {str(e)}")

@router.get("/{allergy_id}")
async def get_allergy(
    allergy_id: str,
    current_user: dict = Depends(get_current_user_from_token)
) -> Dict[str, Any]:
    """
    Get a specific allergy by ID.
    
    Args:
        allergy_id: The allergy ID
        current_user: Current authenticated user
        
    Returns:
        Allergy intolerance resource
    """
    try:
        fhir_service = get_fhir_service()
        resource = await fhir_service.get_resource("AllergyIntolerance", allergy_id)
        
        if not resource:
            raise HTTPException(status_code=404, detail="Allergy not found")
            
        return resource
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error fetching allergy: {e}")
        raise HTTPException(status_code=500, detail=f"Error fetching allergy: {str(e)}")

@router.post("/")
async def create_allergy(
    allergy_data: AllergyIntoleranceCreate,
    current_user: dict = Depends(get_current_user_from_token)
) -> Dict[str, Any]:
    """
    Create a new allergy intolerance.
    
    Args:
        allergy_data: Allergy data
        current_user: Current authenticated user
        
    Returns:
        Created allergy resource
    """
    try:
        fhir_service = get_fhir_service()
        
        # Create FHIR AllergyIntolerance resource
        resource = {
            "resourceType": "AllergyIntolerance",
            "clinicalStatus": {
                "coding": [{
                    "system": "http://terminology.hl7.org/CodeSystem/allergyintolerance-clinical",
                    "code": allergy_data.clinical_status,
                    "display": allergy_data.clinical_status.title()
                }]
            },
            "verificationStatus": {
                "coding": [{
                    "system": "http://terminology.hl7.org/CodeSystem/allergyintolerance-verification",
                    "code": allergy_data.verification_status,
                    "display": allergy_data.verification_status.title()
                }]
            },
            "code": {
                "coding": [{
                    "system": allergy_data.system,
                    "code": allergy_data.code,
                    "display": allergy_data.display or allergy_data.code
                }],
                "text": allergy_data.display or allergy_data.code
            },
            "patient": {
                "reference": f"Patient/{allergy_data.patient_id}"
            }
        }
        
        # Add optional fields
        if allergy_data.criticality:
            resource["criticality"] = allergy_data.criticality
            
        if allergy_data.category:
            resource["category"] = allergy_data.category
            
        if allergy_data.reaction:
            resource["reaction"] = allergy_data.reaction
            
        if allergy_data.note:
            resource["note"] = [{
                "text": allergy_data.note
            }]
        
        created_resource = await fhir_service.create_resource("AllergyIntolerance", resource)
        logger.info(f"Created allergy with ID: {created_resource.get('id')}")
        
        return created_resource
    except Exception as e:
        logger.error(f"Error creating allergy: {e}")
        raise HTTPException(status_code=500, detail=f"Error creating allergy: {str(e)}")

@router.put("/{allergy_id}")
async def update_allergy(
    allergy_id: str,
    allergy_data: AllergyIntoleranceUpdate,
    current_user: dict = Depends(get_current_user_from_token)
) -> Dict[str, Any]:
    """
    Update an existing allergy intolerance.
    
    Args:
        allergy_id: The allergy ID
        allergy_data: Updated allergy data
        current_user: Current authenticated user
        
    Returns:
        Updated allergy resource
    """
    try:
        fhir_service = get_fhir_service()
        
        # Get existing resource
        existing_resource = await fhir_service.get_resource("AllergyIntolerance", allergy_id)
        if not existing_resource:
            raise HTTPException(status_code=404, detail="Allergy not found")
        
        # Update fields
        if allergy_data.clinical_status:
            existing_resource["clinicalStatus"] = {
                "coding": [{
                    "system": "http://terminology.hl7.org/CodeSystem/allergyintolerance-clinical",
                    "code": allergy_data.clinical_status,
                    "display": allergy_data.clinical_status.title()
                }]
            }
            
        if allergy_data.verification_status:
            existing_resource["verificationStatus"] = {
                "coding": [{
                    "system": "http://terminology.hl7.org/CodeSystem/allergyintolerance-verification",
                    "code": allergy_data.verification_status,
                    "display": allergy_data.verification_status.title()
                }]
            }
            
        if allergy_data.criticality:
            existing_resource["criticality"] = allergy_data.criticality
            
        if allergy_data.category:
            existing_resource["category"] = allergy_data.category
            
        if allergy_data.reaction:
            existing_resource["reaction"] = allergy_data.reaction
            
        if allergy_data.note:
            existing_resource["note"] = [{
                "text": allergy_data.note
            }]
        
        updated_resource = await fhir_service.update_resource("AllergyIntolerance", allergy_id, existing_resource)
        logger.info(f"Updated allergy with ID: {allergy_id}")
        
        return updated_resource
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error updating allergy: {e}")
        raise HTTPException(status_code=500, detail=f"Error updating allergy: {str(e)}")

@router.delete("/{allergy_id}")
async def delete_allergy(
    allergy_id: str,
    current_user: dict = Depends(get_current_user_from_token)
) -> Dict[str, str]:
    """
    Delete an allergy intolerance.
    
    Args:
        allergy_id: The allergy ID
        current_user: Current authenticated user
        
    Returns:
        Success message
    """
    try:
        fhir_service = get_fhir_service()
        
        # Check if resource exists
        existing_resource = await fhir_service.get_resource("AllergyIntolerance", allergy_id)
        if not existing_resource:
            raise HTTPException(status_code=404, detail="Allergy not found")
        
        await fhir_service.delete_resource("AllergyIntolerance", allergy_id)
        logger.info(f"Deleted allergy with ID: {allergy_id}")
        
        return {"message": "Allergy deleted successfully"}
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error deleting allergy: {e}")
        raise HTTPException(status_code=500, detail=f"Error deleting allergy: {str(e)}")

# ========================================
# PUBLIC ENDPOINTS (NO AUTHENTICATION)
# ========================================

@public_router.get("/patient/{patient_id}")
async def get_patient_allergies_public(patient_id: str) -> Dict[str, Any]:
    """
    PUBLIC ENDPOINT: Get allergies for a patient (bypasses authentication).

    This endpoint is for testing purposes and bypasses authentication.
    Use the regular /api/allergies/patient/{patient_id} endpoint for production.

    Args:
        patient_id: The patient ID

    Returns:
        List of allergy intolerance resources
    """
    try:
        logger.info(f"PUBLIC: Fetching allergies for patient {patient_id}")
        fhir_service = get_fhir_service()

        if not fhir_service:
            logger.error("PUBLIC: FHIR service not available")
            return {
                "patient_id": patient_id,
                "allergies": [],
                "count": 0,
                "source": "medication_service_public",
                "resource_type": "AllergyIntolerance",
                "error": "FHIR service not available",
                "note": "No known allergies - FHIR service unavailable"
            }

        # Search for allergy intolerances for this patient
        search_params = {"patient": f"Patient/{patient_id}"}
        logger.info(f"PUBLIC: Searching with params: {search_params}")

        resources = await fhir_service.search_resources("AllergyIntolerance", search_params)

        if resources is None:
            resources = []

        logger.info(f"PUBLIC: Retrieved {len(resources)} allergies for patient {patient_id}")

        # Add note about public endpoint usage
        result = {
            "patient_id": patient_id,
            "allergies": resources,
            "count": len(resources),
            "source": "medication_service_public",
            "resource_type": "AllergyIntolerance",
            "note": "Retrieved via public endpoint - bypasses authentication"
        }

        if len(resources) == 0:
            result["note"] = "No known allergies found via public endpoint"

        logger.info(f"PUBLIC: Returning result: count={len(resources)}")
        return result
    except Exception as e:
        logger.error(f"PUBLIC: Error fetching patient allergies: {e}", exc_info=True)
        # Return structured error response instead of raising exception
        return {
            "patient_id": patient_id,
            "allergies": [],
            "count": 0,
            "source": "medication_service_public",
            "resource_type": "AllergyIntolerance",
            "error": str(e),
            "note": "Error fetching allergies via public endpoint"
        }

@public_router.get("")
async def get_all_allergies_public(
    page: int = Query(1, ge=1, description="Page number"),
    limit: int = Query(10, ge=1, le=100, description="Number of items per page"),
    patient: Optional[str] = Query(None, description="Filter by patient ID"),
    status: Optional[str] = Query(None, description="Filter by clinical status")
) -> Dict[str, Any]:
    """
    PUBLIC ENDPOINT: Get all allergies with pagination (bypasses authentication).

    This endpoint is for testing purposes and bypasses authentication.
    Use the regular /api/allergies endpoint for production.

    Args:
        page: Page number (1-based)
        limit: Number of items per page
        patient: Optional patient ID filter
        status: Optional clinical status filter

    Returns:
        Paginated list of allergy resources
    """
    try:
        fhir_service = get_fhir_service()

        # Build search parameters
        search_params = {
            "_count": str(limit),
            "_offset": str((page - 1) * limit)
        }

        if patient:
            search_params["patient"] = f"Patient/{patient}"
        if status:
            search_params["clinical-status"] = status

        resources = await fhir_service.search_resources("AllergyIntolerance", search_params)

        return {
            "items": resources,
            "total": len(resources),
            "page": page,
            "count": limit,
            "note": "Retrieved via public endpoint - bypasses authentication"
        }
    except Exception as e:
        logger.error(f"PUBLIC: Error fetching allergies: {e}")
        return {
            "items": [],
            "total": 0,
            "page": page,
            "count": limit,
            "error": str(e),
            "note": "Error fetching allergies via public endpoint"
        }
