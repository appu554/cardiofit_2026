"""
Slot endpoints for the Scheduling Service.

This module provides REST API endpoints for slot management.
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
class SlotCreate(BaseModel):
    """Model for creating slots"""
    schedule_reference: str  # Reference to the schedule
    status: str = "free"
    start: datetime
    end: datetime
    overbooked: Optional[bool] = False
    comment: Optional[str] = None

class SlotUpdate(BaseModel):
    """Model for updating slots"""
    status: Optional[str] = None
    start: Optional[datetime] = None
    end: Optional[datetime] = None
    overbooked: Optional[bool] = None
    comment: Optional[str] = None

class SlotResponse(BaseModel):
    """Model for slot responses"""
    id: str
    resource_type: str = "Slot"
    schedule: Dict[str, str]  # Reference to schedule
    status: str
    start: datetime
    end: datetime
    overbooked: Optional[bool] = None
    comment: Optional[str] = None

def get_current_user(request: Request):
    """Get current user from request state"""
    return {
        "user_id": getattr(request.state, "user_id", None),
        "user_role": getattr(request.state, "user_role", None),
        "user_roles": getattr(request.state, "user_roles", []),
        "user_permissions": getattr(request.state, "user_permissions", [])
    }

@router.post("/", response_model=SlotResponse)
async def create_slot(
    slot: SlotCreate,
    request: Request,
    current_user: dict = Depends(get_current_user)
):
    """Create a new slot"""
    try:
        fhir_service = get_fhir_service()
        if not fhir_service:
            raise HTTPException(status_code=500, detail="FHIR service not available")
        
        # Convert to FHIR format
        fhir_data = {
            "resourceType": "Slot",
            "schedule": {
                "reference": slot.schedule_reference
            },
            "status": slot.status,
            "start": slot.start.isoformat(),
            "end": slot.end.isoformat(),
            "overbooked": slot.overbooked,
            "comment": slot.comment
        }
        
        # Create the slot
        created_resource = await fhir_service.create_slot(fhir_data)
        
        if not created_resource:
            raise HTTPException(status_code=500, detail="Failed to create slot")
        
        # Convert back to response model
        return SlotResponse(
            id=created_resource["id"],
            schedule=created_resource.get("schedule", {}),
            status=created_resource.get("status", "free"),
            start=datetime.fromisoformat(created_resource["start"].replace("Z", "+00:00")),
            end=datetime.fromisoformat(created_resource["end"].replace("Z", "+00:00")),
            overbooked=created_resource.get("overbooked"),
            comment=created_resource.get("comment")
        )
        
    except Exception as e:
        logger.error(f"Error creating slot: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/{slot_id}", response_model=SlotResponse)
async def get_slot(
    slot_id: str,
    current_user: dict = Depends(get_current_user)
):
    """Get a slot by ID"""
    try:
        fhir_service = get_fhir_service()
        if not fhir_service:
            raise HTTPException(status_code=500, detail="FHIR service not available")
        
        # Get the slot
        slot_data = await fhir_service.get_slot(slot_id)
        
        if not slot_data:
            raise HTTPException(status_code=404, detail="Slot not found")
        
        # Convert to response model
        return SlotResponse(
            id=slot_data["id"],
            schedule=slot_data.get("schedule", {}),
            status=slot_data.get("status", "free"),
            start=datetime.fromisoformat(slot_data["start"].replace("Z", "+00:00")),
            end=datetime.fromisoformat(slot_data["end"].replace("Z", "+00:00")),
            overbooked=slot_data.get("overbooked"),
            comment=slot_data.get("comment")
        )
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error getting slot: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/", response_model=List[SlotResponse])
async def search_slots(
    schedule_id: Optional[str] = Query(None, description="Schedule ID"),
    status: Optional[str] = Query(None, description="Slot status"),
    start: Optional[datetime] = Query(None, description="Start time filter"),
    end: Optional[datetime] = Query(None, description="End time filter"),
    current_user: dict = Depends(get_current_user)
):
    """Search for slots with optional filters"""
    try:
        fhir_service = get_fhir_service()
        if not fhir_service:
            raise HTTPException(status_code=500, detail="FHIR service not available")
        
        # Build search parameters
        search_params = {}
        if schedule_id:
            search_params["schedule"] = f"Schedule/{schedule_id}"
        if status:
            search_params["status"] = status
        if start:
            search_params["start"] = start.isoformat()
        if end:
            search_params["end"] = end.isoformat()
        
        # Search for slots
        slots_data = await fhir_service.search_slots(search_params)
        
        # Convert to response models
        slots = []
        for slot_data in slots_data:
            slot = SlotResponse(
                id=slot_data["id"],
                schedule=slot_data.get("schedule", {}),
                status=slot_data.get("status", "free"),
                start=datetime.fromisoformat(slot_data["start"].replace("Z", "+00:00")),
                end=datetime.fromisoformat(slot_data["end"].replace("Z", "+00:00")),
                overbooked=slot_data.get("overbooked"),
                comment=slot_data.get("comment")
            )
            slots.append(slot)
        
        return slots
        
    except Exception as e:
        logger.error(f"Error searching slots: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.put("/{slot_id}", response_model=SlotResponse)
async def update_slot(
    slot_id: str,
    slot: SlotUpdate,
    current_user: dict = Depends(get_current_user)
):
    """Update an existing slot"""
    try:
        fhir_service = get_fhir_service()
        if not fhir_service:
            raise HTTPException(status_code=500, detail="FHIR service not available")
        
        # Get the current slot
        current_slot = await fhir_service.get_slot(slot_id)
        if not current_slot:
            raise HTTPException(status_code=404, detail="Slot not found")
        
        # Update fields that are provided
        if slot.status is not None:
            current_slot["status"] = slot.status
        if slot.start is not None:
            current_slot["start"] = slot.start.isoformat()
        if slot.end is not None:
            current_slot["end"] = slot.end.isoformat()
        if slot.overbooked is not None:
            current_slot["overbooked"] = slot.overbooked
        if slot.comment is not None:
            current_slot["comment"] = slot.comment
        
        # Update the slot
        updated_resource = await fhir_service.update_slot(slot_id, current_slot)
        
        if not updated_resource:
            raise HTTPException(status_code=500, detail="Failed to update slot")
        
        # Convert back to response model
        return SlotResponse(
            id=updated_resource["id"],
            schedule=updated_resource.get("schedule", {}),
            status=updated_resource.get("status", "free"),
            start=datetime.fromisoformat(updated_resource["start"].replace("Z", "+00:00")),
            end=datetime.fromisoformat(updated_resource["end"].replace("Z", "+00:00")),
            overbooked=updated_resource.get("overbooked"),
            comment=updated_resource.get("comment")
        )
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error updating slot: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.delete("/{slot_id}")
async def delete_slot(
    slot_id: str,
    current_user: dict = Depends(get_current_user)
):
    """Delete a slot"""
    try:
        fhir_service = get_fhir_service()
        if not fhir_service:
            raise HTTPException(status_code=500, detail="FHIR service not available")
        
        # Delete the slot
        success = await fhir_service.delete_slot(slot_id)
        
        if not success:
            raise HTTPException(status_code=404, detail="Slot not found or could not be deleted")
        
        return {"message": "Slot deleted successfully"}
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error deleting slot: {e}")
        raise HTTPException(status_code=500, detail=str(e))
