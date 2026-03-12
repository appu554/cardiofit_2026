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

class ADTMessage(HL7Message):
    """Model for ADT (Admission, Discharge, Transfer) messages"""
    message_type: str = "ADT"
    event_type: str  # A01, A02, A03, etc.
    
    # Patient information
    patient_id: str
    patient_id_type: Optional[str] = None
    patient_id_authority: Optional[str] = None
    patient_name_family: str
    patient_name_given: List[str]
    patient_dob: Optional[datetime] = None
    patient_gender: Optional[str] = None
    patient_address: Optional[List[Dict[str, Any]]] = None
    patient_phone: Optional[List[Dict[str, Any]]] = None
    
    # Visit information
    visit_number: Optional[str] = None
    visit_class: Optional[str] = None
    visit_type: Optional[str] = None
    visit_reason: Optional[str] = None
    admit_datetime: Optional[datetime] = None
    discharge_datetime: Optional[datetime] = None
    location: Optional[Dict[str, Any]] = None
    attending_doctor: Optional[Dict[str, Any]] = None
    referring_doctor: Optional[Dict[str, Any]] = None
    consulting_doctor: Optional[List[Dict[str, Any]]] = None
    hospital_service: Optional[str] = None
    
    # Additional information
    diagnoses: Optional[List[Dict[str, Any]]] = None
    procedures: Optional[List[Dict[str, Any]]] = None
    insurance: Optional[List[Dict[str, Any]]] = None

class HL7MessageRequest(BaseModel):
    """Request model for HL7 message processing"""
    message: str
