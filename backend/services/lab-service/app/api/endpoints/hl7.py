from typing import Dict, List, Any, Optional
from fastapi import APIRouter, Depends, HTTPException, Body, Query, status
from app.core.auth import get_token_payload
from app.services.hl7_service import get_hl7_service
from app.models.hl7 import HL7MessageRequest, HL7MessageResponse

router = APIRouter()
hl7_service = get_hl7_service()

@router.post("/process", response_model=HL7MessageResponse, status_code=status.HTTP_200_OK)
async def process_hl7_message(
    request: HL7MessageRequest = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """
    Process an HL7 message and convert it to lab data.

    This endpoint accepts a raw HL7 message, processes it, and stores the resulting
    lab tests and panels in the lab service.
    """
    try:
        # Process the message
        result = await hl7_service.process_message(request.message)

        return result
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error processing HL7 message: {str(e)}"
        )

@router.post("/oru", response_model=HL7MessageResponse, status_code=status.HTTP_200_OK)
async def process_oru_message(
    request: HL7MessageRequest = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """
    Process an HL7 ORU message and convert it to lab data.

    This endpoint is specifically for ORU (Observation Result) messages.
    It processes the message and stores the resulting lab tests and panels.
    """
    try:
        # Process the message
        result = await hl7_service.process_message(request.message)

        return result
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error processing ORU message: {str(e)}"
        )
