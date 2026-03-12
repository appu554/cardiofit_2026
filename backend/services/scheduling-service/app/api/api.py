"""
Main API router for the Scheduling Service.

This module aggregates all API endpoints for the scheduling service.
"""

from fastapi import APIRouter
from app.api.endpoints import appointments, schedules, slots, webhooks

api_router = APIRouter()

# Include all endpoint routers
api_router.include_router(appointments.router, prefix="/appointments", tags=["appointments"])
api_router.include_router(schedules.router, prefix="/schedules", tags=["schedules"])
api_router.include_router(slots.router, prefix="/slots", tags=["slots"])
api_router.include_router(webhooks.router, prefix="/webhooks", tags=["Webhooks"])
