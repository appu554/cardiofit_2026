from typing import Dict, List, Optional, Any, Union
from pydantic import BaseModel
from datetime import datetime

class HL7Message(BaseModel):
    """Base model for HL7 messages"""
    message_type: str
    message_control_id: str
    message_datetime: datetime
    raw_message: str
    
    class Config:
        extra = "allow"  # Allow extra fields for HL7 messages

class ORUMessage(HL7Message):
    """Model for ORU (Observation Result) messages"""
    message_type: str = "ORU"
    event_type: str  # R01, etc.
    
    # Patient information
    patient_id: str
    patient_id_type: Optional[str] = None
    patient_id_authority: Optional[str] = None
    
    # Observation information
    observation_datetime: datetime
    observation_value: Any
    observation_type: str
    observation_unit: Optional[str] = None
    observation_range: Optional[str] = None
    observation_status: str
    observation_method: Optional[str] = None
    
    # Order information
    order_number: Optional[str] = None
    ordering_provider: Optional[Dict[str, Any]] = None

class HL7MessageRequest(BaseModel):
    """Request model for HL7 message processing"""
    message: str

class HL7MessageResponse(BaseModel):
    """Response model for HL7 message processing"""
    message: str
    resources: Dict[str, Any]
