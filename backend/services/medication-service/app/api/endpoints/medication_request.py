from typing import Dict, List, Optional, Any
from fastapi import APIRouter, Depends, HTTPException, Body, Query, Path, status
from app.core.auth import get_auth_payload
from app.services.medication_service import get_medication_service
from app.models.medication import (
    MedicationRequestCreate, MedicationRequestUpdate,
    MedicationRequestStatus, MedicationRequestIntent
)
from shared.models import MedicationRequest

router = APIRouter()
medication_service = get_medication_service()

@router.post("/", response_model=Dict[str, Any], status_code=status.HTTP_201_CREATED)
async def create_medication_request(
    medication_request: MedicationRequestCreate = Body(...),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Create a new medication request.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Create the medication request
        return await medication_service.create_medication_request(medication_request, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error creating medication request: {str(e)}"
        )

@router.get("/{medication_request_id}", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def get_medication_request(
    medication_request_id: str = Path(..., description="Medication request ID"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Get a medication request by ID.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Get the medication request
        return await medication_service.get_medication_request(medication_request_id, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Error getting medication request: {str(e)}"
        )

@router.put("/{medication_request_id}", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def update_medication_request(
    medication_request_id: str = Path(..., description="Medication request ID"),
    medication_request: MedicationRequestUpdate = Body(...),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Update a medication request.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Update the medication request
        return await medication_service.update_medication_request(medication_request_id, medication_request, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error updating medication request: {str(e)}"
        )

@router.delete("/{medication_request_id}", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def delete_medication_request(
    medication_request_id: str = Path(..., description="Medication request ID"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Delete a medication request.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Delete the medication request
        return await medication_service.delete_medication_request(medication_request_id, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error deleting medication request: {str(e)}"
        )

@router.get("/", response_model=List[Dict[str, Any]], status_code=status.HTTP_200_OK)
async def search_medication_requests(
    status: Optional[MedicationRequestStatus] = Query(None, description="Medication request status"),
    intent: Optional[MedicationRequestIntent] = Query(None, description="Medication request intent"),
    subject: Optional[str] = Query(None, description="Patient reference (e.g., Patient/123)"),
    authored_on: Optional[str] = Query(None, description="Date when request was created"),
    requester: Optional[str] = Query(None, description="Practitioner reference (e.g., Practitioner/123)"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> List[Dict[str, Any]]:
    """
    Search for medication requests.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Get all query parameters
        params = {k: v for k, v in locals().items() if k not in ["auth_payload", "medication_service", "auth_header"] and v is not None}

        # Search for medication requests
        return await medication_service.search_medication_requests(params, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error searching medication requests: {str(e)}"
        )

@router.get("/patient/{patient_id}", response_model=List[Dict[str, Any]], status_code=status.HTTP_200_OK)
async def get_patient_medication_requests(
    patient_id: str = Path(..., description="Patient ID"),
    status: Optional[MedicationRequestStatus] = Query(None, description="Medication request status"),
    intent: Optional[MedicationRequestIntent] = Query(None, description="Medication request intent"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> List[Dict[str, Any]]:
    """
    Get medication requests for a patient.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Get all query parameters
        params = {k: v for k, v in locals().items() if k not in ["patient_id", "auth_payload", "medication_service", "auth_header"] and v is not None}

        # Get patient medication requests
        return await medication_service.get_patient_medication_requests(patient_id, params, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error getting patient medication requests: {str(e)}"
        )
