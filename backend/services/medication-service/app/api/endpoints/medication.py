from typing import Dict, List, Optional, Any
from fastapi import APIRouter, Depends, HTTPException, Body, Query, Path, status
from app.core.auth import get_auth_payload
from app.services.medication_service import get_medication_service
from app.models.medication import MedicationCreate, MedicationUpdate
from shared.models import Medication

router = APIRouter()
medication_service = get_medication_service()

@router.post("/", response_model=Dict[str, Any], status_code=status.HTTP_201_CREATED)
async def create_medication(
    medication: MedicationCreate = Body(...),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Create a new medication.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Create the medication
        return await medication_service.create_medication(medication, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error creating medication: {str(e)}"
        )

@router.get("/{medication_id}", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def get_medication(
    medication_id: str = Path(..., description="Medication ID"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Get a medication by ID.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Get the medication
        return await medication_service.get_medication(medication_id, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Error getting medication: {str(e)}"
        )

@router.put("/{medication_id}", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def update_medication(
    medication_id: str = Path(..., description="Medication ID"),
    medication: MedicationUpdate = Body(...),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Update a medication.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Update the medication
        return await medication_service.update_medication(medication_id, medication, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error updating medication: {str(e)}"
        )

@router.delete("/{medication_id}", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def delete_medication(
    medication_id: str = Path(..., description="Medication ID"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Delete a medication.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Delete the medication
        return await medication_service.delete_medication(medication_id, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error deleting medication: {str(e)}"
        )

@router.get("/", response_model=List[Dict[str, Any]], status_code=status.HTTP_200_OK)
async def search_medications(
    code: Optional[str] = Query(None, description="Medication code"),
    name: Optional[str] = Query(None, description="Medication name"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> List[Dict[str, Any]]:
    """
    Search for medications.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Get all query parameters
        params = {}
        if code is not None:
            params["code"] = code
        if name is not None:
            params["name"] = name
        if _count is not None:
            params["_count"] = _count
        if _page is not None:
            params["_page"] = _page

        # Search for medications
        return await medication_service.search_medications(params, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error searching medications: {str(e)}"
        )
