"""
Appointment endpoints for the Scheduling Service.

This module provides REST API endpoints for appointment management.
"""

import logging
from typing import List, Optional, Dict, Any
from fastapi import APIRouter, HTTPException, Depends, Query, Request
from pydantic import BaseModel
from datetime import datetime

from app.services.fhir_service_factory import get_fhir_service

logger = logging.getLogger(__name__)

router = APIRouter()

# Pydantic models for request/response
class AppointmentCreate(BaseModel):
    """Model for creating appointments"""
    status: str = "proposed"
    description: Optional[str] = None
    start: Optional[datetime] = None
    end: Optional[datetime] = None
    minutes_duration: Optional[int] = None
    comment: Optional[str] = None
    patient_instruction: Optional[str] = None
    participant: Optional[List[Dict[str, Any]]] = None

class AppointmentUpdate(BaseModel):
    """Model for updating appointments"""
    status: Optional[str] = None
    description: Optional[str] = None
    start: Optional[datetime] = None
    end: Optional[datetime] = None
    minutes_duration: Optional[int] = None
    comment: Optional[str] = None
    patient_instruction: Optional[str] = None
    participant: Optional[List[Dict[str, Any]]] = None

class AppointmentResponse(BaseModel):
    """Model for appointment responses"""
    id: str
    resource_type: str = "Appointment"
    status: str
    description: Optional[str] = None
    start: Optional[datetime] = None
    end: Optional[datetime] = None
    minutes_duration: Optional[int] = None
    comment: Optional[str] = None
    patient_instruction: Optional[str] = None
    participant: Optional[List[Dict[str, Any]]] = None

def get_current_user(request: Request):
    """Get current user from request state"""
    return {
        "user_id": getattr(request.state, "user_id", None),
        "user_role": getattr(request.state, "user_role", None),
        "user_roles": getattr(request.state, "user_roles", []),
        "user_permissions": getattr(request.state, "user_permissions", [])
    }

@router.post("/", response_model=AppointmentResponse)
async def create_appointment(
    appointment: AppointmentCreate,
    request: Request,
    current_user: dict = Depends(get_current_user)
):
    """Create a new appointment"""
    try:
        fhir_service = get_fhir_service()
        if not fhir_service:
            raise HTTPException(status_code=500, detail="FHIR service not available")
        
        # Convert to FHIR format
        fhir_data = {
            "resourceType": "Appointment",
            "status": appointment.status,
            "description": appointment.description,
            "start": appointment.start.isoformat() if appointment.start else None,
            "end": appointment.end.isoformat() if appointment.end else None,
            "minutesDuration": appointment.minutes_duration,
            "comment": appointment.comment,
            "patientInstruction": appointment.patient_instruction,
            "participant": appointment.participant or []
        }
        
        # Create the appointment
        created_resource = await fhir_service.create_appointment(fhir_data)
        
        if not created_resource:
            raise HTTPException(status_code=500, detail="Failed to create appointment")
        
        # Convert back to response model
        return AppointmentResponse(
            id=created_resource["id"],
            status=created_resource.get("status", "proposed"),
            description=created_resource.get("description"),
            start=datetime.fromisoformat(created_resource["start"].replace("Z", "+00:00")) if created_resource.get("start") else None,
            end=datetime.fromisoformat(created_resource["end"].replace("Z", "+00:00")) if created_resource.get("end") else None,
            minutes_duration=created_resource.get("minutesDuration"),
            comment=created_resource.get("comment"),
            patient_instruction=created_resource.get("patientInstruction"),
            participant=created_resource.get("participant", [])
        )
        
    except Exception as e:
        logger.error(f"Error creating appointment: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/{appointment_id}", response_model=AppointmentResponse)
async def get_appointment(
    appointment_id: str,
    current_user: dict = Depends(get_current_user)
):
    """Get an appointment by ID"""
    try:
        fhir_service = get_fhir_service()
        if not fhir_service:
            raise HTTPException(status_code=500, detail="FHIR service not available")
        
        # Get the appointment
        appointment_data = await fhir_service.get_appointment(appointment_id)
        
        if not appointment_data:
            raise HTTPException(status_code=404, detail="Appointment not found")
        
        # Convert to response model
        return AppointmentResponse(
            id=appointment_data["id"],
            status=appointment_data.get("status", "proposed"),
            description=appointment_data.get("description"),
            start=datetime.fromisoformat(appointment_data["start"].replace("Z", "+00:00")) if appointment_data.get("start") else None,
            end=datetime.fromisoformat(appointment_data["end"].replace("Z", "+00:00")) if appointment_data.get("end") else None,
            minutes_duration=appointment_data.get("minutesDuration"),
            comment=appointment_data.get("comment"),
            patient_instruction=appointment_data.get("patientInstruction"),
            participant=appointment_data.get("participant", [])
        )
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error getting appointment: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/", response_model=List[AppointmentResponse])
async def search_appointments(
    patient_id: Optional[str] = Query(None, description="Patient ID"),
    practitioner_id: Optional[str] = Query(None, description="Practitioner ID"),
    status: Optional[str] = Query(None, description="Appointment status"),
    date: Optional[str] = Query(None, description="Appointment date (YYYY-MM-DD)"),
    current_user: dict = Depends(get_current_user)
):
    """Search for appointments with optional filters"""
    try:
        fhir_service = get_fhir_service()
        if not fhir_service:
            raise HTTPException(status_code=500, detail="FHIR service not available")
        
        # Build search parameters
        search_params = {}
        if patient_id:
            search_params["patient"] = f"Patient/{patient_id}"
        if practitioner_id:
            search_params["practitioner"] = f"Practitioner/{practitioner_id}"
        if status:
            search_params["status"] = status
        if date:
            search_params["date"] = date
        
        # Search for appointments
        appointments_data = await fhir_service.search_appointments(search_params)
        
        # Convert to response models
        appointments = []
        for apt_data in appointments_data:
            appointment = AppointmentResponse(
                id=apt_data["id"],
                status=apt_data.get("status", "proposed"),
                description=apt_data.get("description"),
                start=datetime.fromisoformat(apt_data["start"].replace("Z", "+00:00")) if apt_data.get("start") else None,
                end=datetime.fromisoformat(apt_data["end"].replace("Z", "+00:00")) if apt_data.get("end") else None,
                minutes_duration=apt_data.get("minutesDuration"),
                comment=apt_data.get("comment"),
                patient_instruction=apt_data.get("patientInstruction"),
                participant=apt_data.get("participant", [])
            )
            appointments.append(appointment)
        
        return appointments
        
    except Exception as e:
        logger.error(f"Error searching appointments: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.put("/{appointment_id}", response_model=AppointmentResponse)
async def update_appointment(
    appointment_id: str,
    appointment: AppointmentUpdate,
    current_user: dict = Depends(get_current_user)
):
    """Update an existing appointment"""
    try:
        fhir_service = get_fhir_service()
        if not fhir_service:
            raise HTTPException(status_code=500, detail="FHIR service not available")
        
        # Get the current appointment
        current_appointment = await fhir_service.get_appointment(appointment_id)
        if not current_appointment:
            raise HTTPException(status_code=404, detail="Appointment not found")
        
        # Update fields that are provided
        if appointment.status is not None:
            current_appointment["status"] = appointment.status
        if appointment.description is not None:
            current_appointment["description"] = appointment.description
        if appointment.start is not None:
            current_appointment["start"] = appointment.start.isoformat()
        if appointment.end is not None:
            current_appointment["end"] = appointment.end.isoformat()
        if appointment.minutes_duration is not None:
            current_appointment["minutesDuration"] = appointment.minutes_duration
        if appointment.comment is not None:
            current_appointment["comment"] = appointment.comment
        if appointment.patient_instruction is not None:
            current_appointment["patientInstruction"] = appointment.patient_instruction
        if appointment.participant is not None:
            current_appointment["participant"] = appointment.participant
        
        # Update the appointment
        updated_resource = await fhir_service.update_appointment(appointment_id, current_appointment)
        
        if not updated_resource:
            raise HTTPException(status_code=500, detail="Failed to update appointment")
        
        # Convert back to response model
        return AppointmentResponse(
            id=updated_resource["id"],
            status=updated_resource.get("status", "proposed"),
            description=updated_resource.get("description"),
            start=datetime.fromisoformat(updated_resource["start"].replace("Z", "+00:00")) if updated_resource.get("start") else None,
            end=datetime.fromisoformat(updated_resource["end"].replace("Z", "+00:00")) if updated_resource.get("end") else None,
            minutes_duration=updated_resource.get("minutesDuration"),
            comment=updated_resource.get("comment"),
            patient_instruction=updated_resource.get("patientInstruction"),
            participant=updated_resource.get("participant", [])
        )
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error updating appointment: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.delete("/{appointment_id}")
async def delete_appointment(
    appointment_id: str,
    current_user: dict = Depends(get_current_user)
):
    """Delete an appointment"""
    try:
        fhir_service = get_fhir_service()
        if not fhir_service:
            raise HTTPException(status_code=500, detail="FHIR service not available")
        
        # Delete the appointment
        success = await fhir_service.delete_appointment(appointment_id)
        
        if not success:
            raise HTTPException(status_code=404, detail="Appointment not found or could not be deleted")
        
        return {"message": "Appointment deleted successfully"}
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error deleting appointment: {e}")
        raise HTTPException(status_code=500, detail=str(e))
