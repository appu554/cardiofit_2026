from typing import Dict, Any
from fastapi import APIRouter, Depends, HTTPException, Body, status
from app.core.auth import get_token_payload
from app.services.hl7_service import get_hl7_service
from app.models.hl7 import HL7MessageRequest, HL7MessageResponse

router = APIRouter()
hl7_service = get_hl7_service()

@router.post("/process", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def process_hl7_message(
    request: HL7MessageRequest = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """
    Process an HL7 message and convert it to FHIR resources.
    
    This endpoint accepts any HL7 message and processes it based on the message type.
    Currently supported message types:
    - ORU (Observation Result)
    """
    try:
        # Process the message
        result = await hl7_service.process_message(request.message)
        
        return result
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error processing HL7 message: {str(e)}"
        )

@router.post("/oru", response_model=Dict[str, Any], status_code=status.HTTP_200_OK)
async def process_oru_message(
    request: HL7MessageRequest = Body(...),
    token_payload: Dict[str, Any] = Depends(get_token_payload)
) -> Dict[str, Any]:
    """
    Process an HL7 ORU message and convert it to observations.
    
    This endpoint is specifically for ORU (Observation Result) messages.
    It processes the message and stores the resulting observations.
    """
    try:
        # Parse the message
        parsed_message = hl7.parse(request.message)
        
        # Check if it's an ORU message
        message_type = str(parsed_message.segment('MSH')[9][0])
        if message_type != 'ORU':
            raise ValueError("Not an ORU message")
        
        # Process the message
        result = await hl7_service.process_oru_message(parsed_message, request.message)
        
        return result
    except ValueError as e:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail=str(e)
        )
    except Exception as e:
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error processing ORU message: {str(e)}"
        )
