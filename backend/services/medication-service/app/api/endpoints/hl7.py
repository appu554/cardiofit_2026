from typing import Dict, Any
from fastapi import APIRouter, Depends, HTTPException, Body, status
from app.core.auth import get_auth_payload
from app.services.hl7_service import get_hl7_service
from app.models.hl7 import HL7MessageRequest, HL7MessageResponse

router = APIRouter()
hl7_service = get_hl7_service()

@router.post("/process", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def process_hl7_message(
    request: HL7MessageRequest = Body(...),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Process an HL7 message and convert it to FHIR resources.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Process the message
        return await hl7_service.process_message(request.message, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error processing HL7 message: {str(e)}"
        )

@router.post("/rde", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def process_rde_message(
    request: HL7MessageRequest = Body(...),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Process an RDE (Pharmacy/Treatment Encoded Order) message.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Process the message
        return await hl7_service.process_rde_message(request.message, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error processing RDE message: {str(e)}"
        )

@router.post("/ras", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def process_ras_message(
    request: HL7MessageRequest = Body(...),
    auth_payload: Dict[str, Any] = Depends(get_auth_payload)
) -> Dict[str, Any]:
    """
    Process an RAS (Pharmacy/Treatment Administration) message.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {auth_payload.get('token', 'dummy_token')}"

        # Process the message
        return await hl7_service.process_ras_message(request.message, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error processing RAS message: {str(e)}"
        )
