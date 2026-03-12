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

class RDEMessage(HL7Message):
    """Model for RDE (Pharmacy/Treatment Encoded Order) messages"""
    message_type: str = "RDE"
    event_type: str  # O11, etc.
    
    # Patient information
    patient_id: str
    patient_id_type: Optional[str] = None
    patient_id_authority: Optional[str] = None
    
    # Order information
    order_number: str
    order_status: str
    order_datetime: datetime
    medication_code: str
    medication_name: str
    dosage: Optional[str] = None
    frequency: Optional[str] = None
    duration: Optional[str] = None
    quantity: Optional[str] = None
    ordering_provider: Optional[Dict[str, Any]] = None

class RASMessage(HL7Message):
    """Model for RAS (Pharmacy/Treatment Administration) messages"""
    message_type: str = "RAS"
    event_type: str  # O17, etc.
    
    # Patient information
    patient_id: str
    patient_id_type: Optional[str] = None
    patient_id_authority: Optional[str] = None
    
    # Administration information
    order_number: str
    administration_datetime: datetime
    medication_code: str
    medication_name: str
    dosage: Optional[str] = None
    administering_provider: Optional[Dict[str, Any]] = None
    administration_status: str

class HL7MessageRequest(BaseModel):
    """Request model for HL7 message processing"""
    message: str

class HL7MessageResponse(BaseModel):
    """Response model for HL7 message processing"""
    message_type: str
    message_control_id: str
    resources_created: List[Dict[str, Any]]
    status: str = "success"
    message: str = "HL7 message processed successfully"
