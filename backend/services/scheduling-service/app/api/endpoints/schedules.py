"""
Schedule endpoints for the Scheduling Service.

This module provides REST API endpoints for schedule management.
"""

import logging
from typing import List, Optional, Dict, Any
from fastapi import APIRouter, HTTPException, Depends, Query, Request
from pydantic import BaseModel

from app.services.fhir_service_factory import get_fhir_service

logger = logging.getLogger(__name__)

router = APIRouter()

# Pydantic models for request/response
class ScheduleCreate(BaseModel):
    """Model for creating schedules"""
    active: Optional[bool] = True
    actor: List[Dict[str, str]]  # List of references to practitioners/resources
    comment: Optional[str] = None

class ScheduleUpdate(BaseModel):
    """Model for updating schedules"""
    active: Optional[bool] = None
    actor: Optional[List[Dict[str, str]]] = None
    comment: Optional[str] = None

class ScheduleResponse(BaseModel):
    """Model for schedule responses"""
    id: str
    resource_type: str = "Schedule"
    active: Optional[bool] = None
    actor: List[Dict[str, str]]
    comment: Optional[str] = None

def get_current_user(request: Request):
    """Get current user from request state"""
    return {
        "user_id": getattr(request.state, "user_id", None),
        "user_role": getattr(request.state, "user_role", None),
        "user_roles": getattr(request.state, "user_roles", []),
        "user_permissions": getattr(request.state, "user_permissions", [])
    }

@router.post("/", response_model=ScheduleResponse)
async def create_schedule(
    schedule: ScheduleCreate,
    request: Request,
    current_user: dict = Depends(get_current_user)
):
    """Create a new schedule"""
    try:
        fhir_service = get_fhir_service()
        if not fhir_service:
            raise HTTPException(status_code=500, detail="FHIR service not available")
        
        # Convert to FHIR format
        fhir_data = {
            "resourceType": "Schedule",
            "active": schedule.active,
            "actor": schedule.actor,
            "comment": schedule.comment
        }
        
        # Create the schedule
        created_resource = await fhir_service.create_schedule(fhir_data)
        
        if not created_resource:
            raise HTTPException(status_code=500, detail="Failed to create schedule")
        
        # Convert back to response model
        return ScheduleResponse(
            id=created_resource["id"],
            active=created_resource.get("active", True),
            actor=created_resource.get("actor", []),
            comment=created_resource.get("comment")
        )
        
    except Exception as e:
        logger.error(f"Error creating schedule: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/{schedule_id}", response_model=ScheduleResponse)
async def get_schedule(
    schedule_id: str,
    current_user: dict = Depends(get_current_user)
):
    """Get a schedule by ID"""
    try:
        fhir_service = get_fhir_service()
        if not fhir_service:
            raise HTTPException(status_code=500, detail="FHIR service not available")
        
        # Get the schedule
        schedule_data = await fhir_service.get_schedule(schedule_id)
        
        if not schedule_data:
            raise HTTPException(status_code=404, detail="Schedule not found")
        
        # Convert to response model
        return ScheduleResponse(
            id=schedule_data["id"],
            active=schedule_data.get("active", True),
            actor=schedule_data.get("actor", []),
            comment=schedule_data.get("comment")
        )
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error getting schedule: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.get("/", response_model=List[ScheduleResponse])
async def search_schedules(
    actor_id: Optional[str] = Query(None, description="Actor (Practitioner) ID"),
    active: Optional[bool] = Query(None, description="Schedule active status"),
    current_user: dict = Depends(get_current_user)
):
    """Search for schedules with optional filters"""
    try:
        fhir_service = get_fhir_service()
        if not fhir_service:
            raise HTTPException(status_code=500, detail="FHIR service not available")
        
        # Build search parameters
        search_params = {}
        if actor_id:
            search_params["actor"] = f"Practitioner/{actor_id}"
        if active is not None:
            search_params["active"] = str(active).lower()
        
        # Search for schedules
        schedules_data = await fhir_service.search_schedules(search_params)
        
        # Convert to response models
        schedules = []
        for schedule_data in schedules_data:
            schedule = ScheduleResponse(
                id=schedule_data["id"],
                active=schedule_data.get("active", True),
                actor=schedule_data.get("actor", []),
                comment=schedule_data.get("comment")
            )
            schedules.append(schedule)
        
        return schedules
        
    except Exception as e:
        logger.error(f"Error searching schedules: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.put("/{schedule_id}", response_model=ScheduleResponse)
async def update_schedule(
    schedule_id: str,
    schedule: ScheduleUpdate,
    current_user: dict = Depends(get_current_user)
):
    """Update an existing schedule"""
    try:
        fhir_service = get_fhir_service()
        if not fhir_service:
            raise HTTPException(status_code=500, detail="FHIR service not available")
        
        # Get the current schedule
        current_schedule = await fhir_service.get_schedule(schedule_id)
        if not current_schedule:
            raise HTTPException(status_code=404, detail="Schedule not found")
        
        # Update fields that are provided
        if schedule.active is not None:
            current_schedule["active"] = schedule.active
        if schedule.actor is not None:
            current_schedule["actor"] = schedule.actor
        if schedule.comment is not None:
            current_schedule["comment"] = schedule.comment
        
        # Update the schedule
        updated_resource = await fhir_service.update_schedule(schedule_id, current_schedule)
        
        if not updated_resource:
            raise HTTPException(status_code=500, detail="Failed to update schedule")
        
        # Convert back to response model
        return ScheduleResponse(
            id=updated_resource["id"],
            active=updated_resource.get("active", True),
            actor=updated_resource.get("actor", []),
            comment=updated_resource.get("comment")
        )
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error updating schedule: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@router.delete("/{schedule_id}")
async def delete_schedule(
    schedule_id: str,
    current_user: dict = Depends(get_current_user)
):
    """Delete a schedule"""
    try:
        fhir_service = get_fhir_service()
        if not fhir_service:
            raise HTTPException(status_code=500, detail="FHIR service not available")
        
        # Delete the schedule
        success = await fhir_service.delete_schedule(schedule_id)
        
        if not success:
            raise HTTPException(status_code=404, detail="Schedule not found or could not be deleted")
        
        return {"message": "Schedule deleted successfully"}
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error deleting schedule: {e}")
        raise HTTPException(status_code=500, detail=str(e))
