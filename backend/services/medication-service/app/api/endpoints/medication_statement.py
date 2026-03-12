from typing import Dict, List, Optional, Any
from fastapi import APIRouter, Depends, HTTPException, Body, Query, Path, status
from app.core.auth import get_auth_payload
from app.services.medication_service import get_medication_service
from app.models.medication import (
    MedicationStatementCreate, MedicationStatementUpdate,
    MedicationStatementStatus
)
from shared.models import MedicationStatement

router = APIRouter()
medication_service = get_medication_service()

@router.post("/", response_model=Dict[str, Any], status_code=status.HTTP_201_CREATED)
async def create_medication_statement(
    medication_statement: MedicationStatementCreate = Body(...),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Create a new medication statement.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Create the medication statement
        return await medication_service.create_medication_statement(medication_statement, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error creating medication statement: {str(e)}"
        )

@router.get("/{medication_statement_id}", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def get_medication_statement(
    medication_statement_id: str = Path(..., description="Medication statement ID"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Get a medication statement by ID.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Get the medication statement
        return await medication_service.get_medication_statement(medication_statement_id, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail=f"Error getting medication statement: {str(e)}"
        )

@router.put("/{medication_statement_id}", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def update_medication_statement(
    medication_statement_id: str = Path(..., description="Medication statement ID"),
    medication_statement: MedicationStatementUpdate = Body(...),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Update a medication statement.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Update the medication statement
        return await medication_service.update_medication_statement(medication_statement_id, medication_statement, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error updating medication statement: {str(e)}"
        )

@router.delete("/{medication_statement_id}", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def delete_medication_statement(
    medication_statement_id: str = Path(..., description="Medication statement ID"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Delete a medication statement.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Delete the medication statement
        return await medication_service.delete_medication_statement(medication_statement_id, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error deleting medication statement: {str(e)}"
        )

@router.get("/", response_model=List[Dict[str, Any]], status_code=status.HTTP_200_OK)
async def search_medication_statements(
    status: Optional[MedicationStatementStatus] = Query(None, description="Medication statement status"),
    subject: Optional[str] = Query(None, description="Patient reference (e.g., Patient/123)"),
    effective_date: Optional[str] = Query(None, description="Date when medication was taken"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> List[Dict[str, Any]]:
    """
    Search for medication statements.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Get all query parameters
        params = {k: v for k, v in locals().items() if k not in ["auth_payload", "medication_service", "auth_header"] and v is not None}

        # Search for medication statements
        return await medication_service.search_medication_statements(params, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error searching medication statements: {str(e)}"
        )

@router.get("/patient/{patient_id}", response_model=List[Dict[str, Any]], status_code=status.HTTP_200_OK)
async def get_patient_medication_statements(
    patient_id: str = Path(..., description="Patient ID"),
    status: Optional[MedicationStatementStatus] = Query(None, description="Medication statement status"),
    effective_date: Optional[str] = Query(None, description="Date when medication was taken"),
    _count: Optional[int] = Query(100, description="Number of results per page"),
    _page: Optional[int] = Query(1, description="Page number"),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> List[Dict[str, Any]]:
    """
    Get medication statements for a patient.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Get all query parameters
        params = {k: v for k, v in locals().items() if k not in ["patient_id", "auth_payload", "medication_service", "auth_header"] and v is not None}

        # Get patient medication statements
        return await medication_service.get_patient_medication_statements(patient_id, params, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error getting patient medication statements: {str(e)}"
        )
