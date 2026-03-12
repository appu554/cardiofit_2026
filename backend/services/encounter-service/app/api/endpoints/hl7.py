from typing import Dict, Any
from fastapi import APIRouter, Depends, HTTPException, Body, status
from app.core.auth import get_token_payload
from app.models.hl7 import HL7MessageRequest
from app.services.hl7_service import hl7_service

router = APIRouter()

@router.post("/process", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def process_hl7_message(
    message_request: HL7MessageRequest = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """
    Process an HL7 message.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', 'dummy_token')}"

        # Process the HL7 message
        return await hl7_service.process_message(message_request, auth_header)
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error processing HL7 message: {str(e)}"
        )

@router.post("/adt", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def process_adt_message(
    message_request: HL7MessageRequest = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """
    Process an HL7 ADT message.
    """
    try:
        # Get the authorization header from the request
        auth_header = f"Bearer {token_payload.get('token', 'dummy_token')}"

        # Process the HL7 message
        result = await hl7_service.process_message(message_request, auth_header)
        
        # Check if it's an ADT message
        if "error" in result.get("status", "") and "Unsupported message type" in result.get("message", ""):
            raise HTTPException(
                status_code=status.HTTP_400_BAD_REQUEST,
                detail="The provided message is not an ADT message"
            )
            
        return result
    except HTTPException:
        raise
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=f"Error processing ADT message: {str(e)}"
        )
