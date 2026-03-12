from typing import Dict, List, Any, Optional
from fastapi import APIRouter, Depends, HTTPException, Body, Query, status
from pydantic import BaseModel
from app.core.auth import get_token_payload
from app.services.hl7_service import get_hl7_service
from app.services.fhir_service import get_fhir_service

router = APIRouter()
hl7_service = get_hl7_service()
fhir_service = get_fhir_service()

class HL7MessageRequest(BaseModel):
    """Request model for HL7 message processing"""
    message: str

class HL7MessageResponse(BaseModel):
    """Response model for HL7 message processing"""
    message: str
    resources: Dict[str, Any]

class HL7ResourcesResponse(BaseModel):
    """Response model for HL7 resources query"""
    patients: List[Dict[str, Any]] = []
    encounters: List[Dict[str, Any]] = []
    total_count: int = 0

@router.post("/process", response_model=HL7MessageResponse, status_code=status.HTTP_200_OK)
async def process_hl7_message(
    request: HL7MessageRequest = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """
    Process an HL7 message and convert it to FHIR resources.

    This endpoint accepts a raw HL7 message, processes it, and stores the resulting
    FHIR resources in the appropriate services.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Process the message
        result = await hl7_service.process_message(request.message, auth_header)

        return result
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error processing HL7 message: {str(e)}"
        )

@router.post("/adt", response_model=HL7MessageResponse, status_code=status.HTTP_200_OK)
async def process_adt_message(
    request: HL7MessageRequest = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """
    Process an HL7 ADT message and convert it to FHIR resources.

    This endpoint is specifically for ADT (Admission, Discharge, Transfer) messages.
    It processes the message and stores the resulting Patient and Encounter resources.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Process the message
        result = await hl7_service.process_message(request.message, auth_header)

        return result
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error processing ADT message: {str(e)}"
        )

@router.get("/resources", response_model=HL7ResourcesResponse, status_code=status.HTTP_200_OK)
async def get_hl7_resources(
    message_type: Optional[str] = Query(None, description="HL7 message type (e.g., A01, A02, A03)"),
    resource_type: Optional[str] = Query(None, description="Resource type (Patient, Encounter)"),
    limit: int = Query(100, description="Maximum number of resources to return"),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """
    Get resources created from HL7 messages.

    This endpoint allows querying for resources that were created from HL7 messages,
    with optional filtering by message type (e.g., A01 for admissions) and resource type.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', '')}"

        # Prepare the response
        response = {
            "patients": [],
            "encounters": [],
            "total_count": 0
        }

        # Prepare the tag query parameter
        tag_query = None
        if message_type:
            tag_query = f"http://clinicalsynthesishub.com/hl7/message_type|{message_type}"
        else:
            tag_query = "http://clinicalsynthesishub.com/source|hl7v2"

        # Get patients with HL7 tags
        if not resource_type or resource_type.lower() == "patient":
            patient_params = {"_tag": tag_query, "_count": limit}
            patients = await fhir_service.search_resources("Patient", patient_params)
            response["patients"] = patients
            response["total_count"] += len(patients)

        # Get encounters with HL7 tags
        if not resource_type or resource_type.lower() == "encounter":
            encounter_params = {"_tag": tag_query, "_count": limit}
            encounters = await fhir_service.search_resources("Encounter", encounter_params)
            response["encounters"] = encounters
            response["total_count"] += len(encounters)

        return response
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error retrieving HL7 resources: {str(e)}"
        )
