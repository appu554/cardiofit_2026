"""
Context Service endpoints for Patient Service.

This module provides endpoints specifically for the Context Service
to access patient data without authentication (internal service-to-service communication).
"""

from fastapi import APIRouter, HTTPException, Query
from typing import Dict, List, Any, Optional
import logging

# Import the patient service
from app.services.patient_service import get_patient_service

logger = logging.getLogger(__name__)

router = APIRouter()


@router.get("/patients")
async def get_patients_for_context(
    limit: int = Query(default=10, ge=1, le=100),
    offset: int = Query(default=0, ge=0)
):
    """
    Get patients for Context Service (no authentication required).
    
    This endpoint is designed for internal service-to-service communication
    between the Context Service and Patient Service.
    
    Args:
        limit: Maximum number of patients to return (1-100)
        offset: Number of patients to skip
        
    Returns:
        List of patient resources
    """
    try:
        logger.info(f"Context Service requesting {limit} patients (offset: {offset})")
        
        # Get the patient service
        patient_service = await get_patient_service()
        
        # Search for patients
        search_params = {
            "_count": str(limit),
            "_page": str((offset // limit) + 1)
        }
        
        patients_result = await patient_service.search_patients(search_params)
        
        # Extract patients from the result
        if hasattr(patients_result, 'patients'):
            patients = patients_result.patients
        else:
            patients = patients_result if isinstance(patients_result, list) else []
        
        logger.info(f"Returning {len(patients)} patients to Context Service")
        
        return {
            "patients": patients,
            "count": len(patients),
            "limit": limit,
            "offset": offset,
            "source": "patient-service",
            "for": "context-service"
        }
        
    except Exception as e:
        logger.error(f"Error getting patients for Context Service: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error retrieving patients: {str(e)}")


@router.get("/patients/{patient_id}")
async def get_patient_for_context(patient_id: str):
    """
    Get a specific patient for Context Service (no authentication required).
    
    Args:
        patient_id: The patient ID
        
    Returns:
        Patient resource or 404 if not found
    """
    try:
        logger.info(f"Context Service requesting patient: {patient_id}")
        
        # Get the patient service
        patient_service = await get_patient_service()
        
        # Get the patient
        patient = await patient_service.get_patient(patient_id)
        
        if patient is None:
            logger.warning(f"Patient {patient_id} not found for Context Service")
            raise HTTPException(status_code=404, detail=f"Patient {patient_id} not found")
        
        logger.info(f"Returning patient {patient_id} to Context Service")
        
        return {
            "patient": patient,
            "source": "patient-service",
            "for": "context-service"
        }
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error getting patient {patient_id} for Context Service: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error retrieving patient: {str(e)}")


@router.get("/patients/{patient_id}/fhir")
async def get_patient_fhir_for_context(patient_id: str):
    """
    Get a specific patient in FHIR format for Context Service (no authentication required).
    
    Args:
        patient_id: The patient ID
        
    Returns:
        FHIR Patient resource or 404 if not found
    """
    try:
        logger.info(f"Context Service requesting FHIR patient: {patient_id}")
        
        # Get the patient service
        patient_service = await get_patient_service()
        
        # Get the patient (this returns FHIR format when using Google Healthcare API)
        patient = await patient_service.get_patient(patient_id)
        
        if patient is None:
            logger.warning(f"FHIR patient {patient_id} not found for Context Service")
            raise HTTPException(status_code=404, detail=f"Patient {patient_id} not found")
        
        # Ensure it's in FHIR format
        if not patient.get("resourceType"):
            patient["resourceType"] = "Patient"
        
        logger.info(f"Returning FHIR patient {patient_id} to Context Service")
        
        return patient  # Return the FHIR resource directly
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error getting FHIR patient {patient_id} for Context Service: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error retrieving FHIR patient: {str(e)}")


@router.get("/patients/search")
async def search_patients_for_context(
    name: Optional[str] = Query(default=None),
    identifier: Optional[str] = Query(default=None),
    active: Optional[bool] = Query(default=None),
    limit: int = Query(default=10, ge=1, le=100)
):
    """
    Search patients for Context Service (no authentication required).
    
    Args:
        name: Patient name to search for
        identifier: Patient identifier to search for
        active: Filter by active status
        limit: Maximum number of results
        
    Returns:
        List of matching patient resources
    """
    try:
        logger.info(f"Context Service searching patients with filters: name={name}, identifier={identifier}, active={active}")
        
        # Get the patient service
        patient_service = await get_patient_service()
        
        # Build search parameters
        search_params = {"_count": str(limit)}
        
        if name:
            search_params["name"] = name
        if identifier:
            search_params["identifier"] = identifier
        if active is not None:
            search_params["active"] = str(active).lower()
        
        # Search for patients
        patients_result = await patient_service.search_patients(search_params)
        
        # Extract patients from the result
        if hasattr(patients_result, 'patients'):
            patients = patients_result.patients
        else:
            patients = patients_result if isinstance(patients_result, list) else []
        
        logger.info(f"Found {len(patients)} patients for Context Service")
        
        return {
            "patients": patients,
            "count": len(patients),
            "filters": {
                "name": name,
                "identifier": identifier,
                "active": active
            },
            "limit": limit,
            "source": "patient-service",
            "for": "context-service"
        }
        
    except Exception as e:
        logger.error(f"Error searching patients for Context Service: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error searching patients: {str(e)}")


@router.get("/status")
async def get_context_service_status():
    """
    Get status information for Context Service integration.
    
    Returns:
        Status information about the Patient Service
    """
    try:
        # Get the patient service
        patient_service = await get_patient_service()
        
        # Check if FHIR service is available
        fhir_service_type = type(patient_service.fhir_service).__name__ if patient_service.fhir_service else "None"
        
        return {
            "status": "healthy",
            "service": "patient-service",
            "for": "context-service",
            "fhir_service": fhir_service_type,
            "endpoints": [
                "/api/context/patients",
                "/api/context/patients/{patient_id}",
                "/api/context/patients/{patient_id}/fhir",
                "/api/context/patients/search",
                "/api/context/status"
            ]
        }
        
    except Exception as e:
        logger.error(f"Error getting status for Context Service: {str(e)}")
        raise HTTPException(status_code=500, detail=f"Error getting status: {str(e)}")
