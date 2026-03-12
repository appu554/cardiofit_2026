from typing import Dict, List, Optional, Any
from fastapi import APIRouter, Depends, HTTPException, Body, Query, Path, status
from app.core.auth import get_auth_payload
from app.services.medication_service import get_medication_service
from app.models.medication import (
    MedicationAdministrationCreate, MedicationAdministrationUpdate,
    MedicationAdministrationStatus
)
from shared.models import MedicationAdministration

router = APIRouter()
medication_service = get_medication_service()

@router.post("/", response_model=Dict[str, Any], status_code=status.HTTP_201_CREATED)
async def create_medication_administration(
    medication_administration: MedicationAdministrationCreate = Body(...),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Create a new medication administration.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Create the medication administration
        return await medication_service.create_medication_administration(medication_administration, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error creating medication administration: {str(e)}"
        )

@router.get("/{medication_administration_id}", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def get_medication_administration(
    medication_administration_id: str = Path(..., description="Medication administration ID"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Get a medication administration by ID.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Get the medication administration
        return await medication_service.get_medication_administration(medication_administration_id, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Error getting medication administration: {str(e)}"
        )

@router.put("/{medication_administration_id}", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def update_medication_administration(
    medication_administration_id: str = Path(..., description="Medication administration ID"),
    medication_administration: MedicationAdministrationUpdate = Body(...),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Update a medication administration.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Update the medication administration
        return await medication_service.update_medication_administration(medication_administration_id, medication_administration, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error updating medication administration: {str(e)}"
        )

@router.delete("/{medication_administration_id}", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def delete_medication_administration(
    medication_administration_id: str = Path(..., description="Medication administration ID"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Delete a medication administration.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Delete the medication administration
        return await medication_service.delete_medication_administration(medication_administration_id, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error deleting medication administration: {str(e)}"
        )

@router.get("/", response_model=List[Dict[str, Any]], status_code=status.HTTP_200_OK)
async def search_medication_administrations(
    status: Optional[MedicationAdministrationStatus] = Query(None, description="Medication administration status"),
    subject: Optional[str] = Query(None, description="Patient reference (e.g., Patient/123)"),
    effective_date: Optional[str] = Query(None, description="Date of administration"),
    request: Optional[str] = Query(None, description="MedicationRequest reference (e.g., MedicationRequest/123)"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> List[Dict[str, Any]]:
    """
    Search for medication administrations.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Get all query parameters
        params = {k: v for k, v in locals().items() if k not in ["auth_payload", "medication_service", "auth_header"] and v is not None}

        # Search for medication administrations
        return await medication_service.search_medication_administrations(params, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error searching medication administrations: {str(e)}"
        )

@router.get("/patient/{patient_id}", response_model=List[Dict[str, Any]], status_code=status.HTTP_200_OK)
async def get_patient_medication_administrations(
    patient_id: str = Path(..., description="Patient ID"),
    status: Optional[MedicationAdministrationStatus] = Query(None, description="Medication administration status"),
    effective_date: Optional[str] = Query(None, description="Date of administration"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> List[Dict[str, Any]]:
    """
    Get medication administrations for a patient.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Get all query parameters
        params = {k: v for k, v in locals().items() if k not in ["patient_id", "auth_payload", "medication_service", "auth_header"] and v is not None}

        # Get patient medication administrations
        return await medication_service.get_patient_medication_administrations(patient_id, params, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error getting patient medication administrations: {str(e)}"
        )
