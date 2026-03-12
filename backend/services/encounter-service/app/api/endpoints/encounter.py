from typing import Dict, List, Optional, Any
from fastapi import APIRouter, Depends, HTTPException, Query, Path, Body, status
from app.core.auth import get_token_payload
from app.models.encounter import EncounterCreate, EncounterUpdate
from app.services.encounter_service import encounter_service
from shared.models import Encounter

router = APIRouter()

@router.post("/", response_model=Dict[str, Any], status_code=status.HTTP_201_CREATED)
async def create_encounter(
    encounter: EncounterCreate = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """
    Create a new encounter.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', 'dummy_token')}"

        # Create the encounter
        return await encounter_service.create_encounter(encounter, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error creating encounter: {str(e)}"
        )

@router.get("/{id}", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def get_encounter(
    id: str = Path(..., description="Encounter ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """
    Get an encounter by ID.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', 'dummy_token')}"

        # Get the encounter
        encounter = await encounter_service.get_encounter(id, auth_header)
        if not encounter:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Encounter with ID {id} not found"
            )
        return encounter
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting encounter: {str(e)}"
        )

@router.put("/{id}", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def update_encounter(
    id: str = Path(..., description="Encounter ID"),
    encounter: EncounterUpdate = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """
    Update an encounter.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', 'dummy_token')}"

        # Update the encounter
        updated_encounter = await encounter_service.update_encounter(id, encounter, auth_header)
        if not updated_encounter:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Encounter with ID {id} not found"
            )
        return updated_encounter
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error updating encounter: {str(e)}"
        )

@router.delete("/{id}", status_code=status.HTTP_204_NO_CONTENT)
async def delete_encounter(
    id: str = Path(..., description="Encounter ID"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> None:
    """
    Delete an encounter.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', 'dummy_token')}"

        # Delete the encounter
        deleted = await encounter_service.delete_encounter(id, auth_header)
        if not deleted:
            raise HTTPException(
                status_code=status.HTTP_404_NOT_FOUND,
                detail=f"Encounter with ID {id} not found"
            )
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error deleting encounter: {str(e)}"
        )

@router.get("/", response_model=List[Dict[str, Any]], status_code=status.HTTP_200_OK)
async def search_encounters(
    status: Optional[str] = Query(None, description="Encounter status"),
    subject: Optional[str] = Query(None, description="Patient reference (e.g., Patient/123)"),
    date: Optional[str] = Query(None, description="Encounter date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> List[Dict[str, Any]]:
    """
    Search for encounters.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', 'dummy_token')}"

        # Build search parameters
        params = {}
        if status:
            params["status"] = status
        if subject:
            params["subject"] = subject
        if date:
            params["date"] = date
        params["_count"] = _count
        params["_page"] = _page

        # Search for encounters
        return await encounter_service.search_encounters(params, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error searching encounters: {str(e)}"
        )

@router.get("/patient/{patient_id}", response_model=List[Dict[str, Any]], status_code=status.HTTP_200_OK)
async def get_patient_encounters(
    patient_id: str = Path(..., description="Patient ID"),
    status: Optional[str] = Query(None, description="Encounter status"),
    date: Optional[str] = Query(None, description="Encounter date"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> List[Dict[str, Any]]:
    """
    Get encounters for a patient.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', 'dummy_token')}"

        # Build search parameters
        params = {}
        if status:
            params["status"] = status
        if date:
            params["date"] = date
        params["_count"] = _count
        params["_page"] = _page

        # Get patient encounters
        return await encounter_service.get_patient_encounters(patient_id, params, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error getting patient encounters: {str(e)}"
        )
