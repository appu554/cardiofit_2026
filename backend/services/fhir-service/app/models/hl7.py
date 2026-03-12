from typing import Dict, List, Optional, Any, Union
from pydantic import BaseModel, Field
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
    assigned_location: Optional[str] = None
    attending_doctor: Optional[Dict[str, Any]] = None
    referring_doctor: Optional[Dict[str, Any]] = None
    hospital_service: Optional[str] = None
    admit_datetime: Optional[datetime] = None
    discharge_datetime: Optional[datetime] = None

    # Additional fields for specific event types
    previous_location: Optional[str] = None  # For A02 (transfer)
    reason: Optional[str] = None  # Reason for admission/transfer/etc.

# ORUMessage has been moved to the Lab Microservice
