"""Doctor Dashboard 360 endpoints — routes for React physician dashboard.

Proxies to KB-20 (patient profile), KB-23 (decision cards), KB-26 (MRI),
V-MCU (safety traces), and Apollo Federation (GraphQL).
"""
import logging
from typing import Any

import httpx
from fastapi import APIRouter, Request, HTTPException

from app.config import settings  # module-level singleton, NOT get_settings()
from app.api.endpoints.patient_app import _forward, _build_headers

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/api/v1/doctor", tags=["doctor-dashboard"])


# --- GraphQL proxy to Apollo Federation ---

@router.post("/graphql")
async def doctor_graphql(request: Request):
    """Proxy GraphQL queries to Apollo Federation (port 4000)."""
    return await _forward(request, settings.APOLLO_FEDERATION_URL, "/graphql", method="POST")


# --- Direct REST for latency-sensitive reads ---

@router.get("/patients/{patient_id}/summary")
async def patient_summary(patient_id: str, request: Request):
    """KB-20: Patient profile summary."""
    return await _forward(request, settings.KB20_SERVICE_URL, f"/patients/{patient_id}/summary")


@router.get("/patients/{patient_id}/mri")
async def patient_mri(patient_id: str, request: Request):
    """KB-26: Metabolic Risk Index."""
    return await _forward(request, settings.KB26_SERVICE_URL, f"/patients/{patient_id}/mri")


@router.get("/patients/{patient_id}/cards")
async def patient_cards(patient_id: str, request: Request):
    """KB-23: Decision cards."""
    return await _forward(request, settings.KB23_SERVICE_URL, f"/patients/{patient_id}/cards")


@router.post("/cards/{card_id}/action")
async def card_action(card_id: str, request: Request):
    """KB-23: Physician action on card (approve/modify/escalate)."""
    return await _forward(request, settings.KB23_SERVICE_URL, f"/cards/{card_id}/action", method="POST")


# --- V-MCU safety traces ---

@router.get("/traces/{patient_id}")
async def safety_traces(patient_id: str, request: Request):
    """V-MCU: Safety trace audit log."""
    if not settings.VMCU_SERVICE_URL:
        raise HTTPException(status_code=503, detail="V-MCU service not configured")
    return await _forward(request, settings.VMCU_SERVICE_URL, f"/traces/{patient_id}")


# --- KB-20 projections (Channel B/C) ---

@router.get("/patients/{patient_id}/channel-b-inputs")
async def channel_b_inputs(patient_id: str, request: Request):
    """KB-20: Channel B projection data."""
    return await _forward(request, settings.KB20_SERVICE_URL, f"/patients/{patient_id}/channel-b-inputs")


@router.get("/patients/{patient_id}/channel-c-inputs")
async def channel_c_inputs(patient_id: str, request: Request):
    """KB-20: Channel C projection data."""
    return await _forward(request, settings.KB20_SERVICE_URL, f"/patients/{patient_id}/channel-c-inputs")
