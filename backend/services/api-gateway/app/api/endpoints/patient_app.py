"""Patient App REST endpoints -- routes for Flutter mobile app.

These endpoints proxy to downstream Vaidshala services with
X-User-* headers injected from validated JWT claims.
"""
import logging
from typing import Any

import httpx
from fastapi import APIRouter, Request, HTTPException

from app.config import settings  # module-level singleton, NOT get_settings()
from app.middleware.circuit_breaker import get_breaker

logger = logging.getLogger(__name__)
router = APIRouter(prefix="/api/v1", tags=["patient-app"])

# --- Public routes (no auth) ---

auth_router = APIRouter(prefix="/api/v1/auth", tags=["auth"])


@auth_router.post("/otp/send")
async def otp_send(request: Request):
    """Proxy OTP send to Auth Service."""
    return await _forward(request, settings.AUTH_SERVICE_URL, "/api/auth/otp/send")


@auth_router.post("/otp/verify")
async def otp_verify(request: Request):
    """Proxy OTP verify to Auth Service."""
    return await _forward(request, settings.AUTH_SERVICE_URL, "/api/auth/otp/verify")


@auth_router.post("/refresh")
async def token_refresh(request: Request):
    """Proxy token refresh to Auth Service."""
    return await _forward(request, settings.AUTH_SERVICE_URL, "/api/auth/refresh")


# --- Protected routes (JWT required, enforced by middleware) ---

@router.get("/patient/{patient_id}/health-score")
async def patient_health_score(patient_id: str, request: Request):
    """KB-26: Composite health score from Metabolic Digital Twin."""
    return await _forward(request, settings.KB26_SERVICE_URL, f"/patients/{patient_id}/health-score")


@router.get("/patient/{patient_id}/actions/today")
async def patient_actions_today(patient_id: str, request: Request):
    """KB-23: Today's action items from Decision Cards."""
    return await _forward(request, settings.KB23_SERVICE_URL, f"/patients/{patient_id}/actions/today")


@router.get("/patient/{patient_id}/health-drive")
async def patient_health_drive(patient_id: str, request: Request):
    """KB-25: Lifestyle Knowledge Graph health-drive data."""
    return await _forward(request, settings.KB25_SERVICE_URL, f"/patients/{patient_id}/health-drive")


@router.get("/patient/{patient_id}/progress")
async def patient_progress(patient_id: str, request: Request):
    """KB-20: Protocol progress from Patient Profile."""
    return await _forward(request, settings.KB20_SERVICE_URL, f"/patients/{patient_id}/progress")


@router.get("/patient/{patient_id}/cause-effect")
async def patient_cause_effect(patient_id: str, request: Request):
    """KB-26: Cause-effect analysis from Metabolic Digital Twin."""
    return await _forward(request, settings.KB26_SERVICE_URL, f"/patients/{patient_id}/cause-effect")


@router.get("/patient/{patient_id}/timeline")
async def patient_timeline(patient_id: str, request: Request):
    """KB-20: Patient clinical timeline."""
    return await _forward(request, settings.KB20_SERVICE_URL, f"/patients/{patient_id}/timeline")


@router.get("/patient/{patient_id}/insights")
async def patient_insights(patient_id: str, request: Request):
    """KB-26: Patient-facing insights from Metabolic Digital Twin."""
    return await _forward(request, settings.KB26_SERVICE_URL, f"/patients/{patient_id}/insights")


@router.post("/patient/{patient_id}/checkin")
async def patient_checkin(patient_id: str, request: Request):
    """KB-22: Daily check-in via HPI Engine."""
    return await _forward(request, settings.KB22_SERVICE_URL, f"/patients/{patient_id}/checkin", method="POST")


@router.post("/patient/{patient_id}/abdm/verify")
async def patient_abdm_verify(patient_id: str, request: Request):
    """ABDM integration -- proxied to Patient Service."""
    return await _forward(request, settings.PATIENT_SERVICE_URL, f"/patients/{patient_id}/abdm/verify", method="POST")


@router.get("/family/{token}")
async def family_view(token: str, request: Request):
    """Family view -- token-scoped, no JWT required (handled by service)."""
    return await _forward(request, settings.PATIENT_SERVICE_URL, f"/family/{token}")


@router.get("/tenants/{tenant_id}/branding")
async def tenant_branding(tenant_id: str, request: Request):
    """Multi-tenant branding -- public."""
    return await _forward(request, settings.PATIENT_SERVICE_URL, f"/tenants/{tenant_id}/branding")


async def _forward(request: Request, service_url: str, path: str, method: str = None) -> Any:
    """Forward request to a downstream service with X-User-* headers."""
    method = method or request.method
    breaker = get_breaker(service_url)

    if not breaker.is_available():
        raise HTTPException(status_code=503, detail="Service temporarily unavailable")

    headers = _build_headers(request)
    body = await request.body() if method in ("POST", "PUT", "PATCH") else None

    try:
        async with httpx.AsyncClient(timeout=30) as client:
            resp = await client.request(
                method=method,
                url=f"{service_url}{path}",
                headers=headers,
                content=body,
                params=dict(request.query_params),
            )
            if resp.status_code >= 500:
                breaker.record_failure()
            else:
                breaker.record_success()
            return resp.json()
    except httpx.ConnectError:
        breaker.record_failure()
        raise HTTPException(status_code=502, detail="Downstream service unavailable")
    except Exception as e:
        breaker.record_failure()
        logger.error("Proxy error: %s %s → %s", method, path, e)
        raise HTTPException(status_code=502, detail="Downstream service error")


def _build_headers(request: Request) -> dict:
    """Build headers with validated user context for downstream services."""
    headers = {"Content-Type": "application/json"}

    # Forward auth token
    auth = request.headers.get("authorization")
    if auth:
        headers["Authorization"] = auth

    # Inject validated user context from JWT claims (set by auth middleware)
    user = getattr(request.state, "user", None)
    if user and isinstance(user, dict):
        headers["X-User-ID"] = user.get("id", "")
        headers["X-User-Email"] = user.get("email", "")
        roles = user.get("roles", [])
        if roles:
            headers["X-User-Roles"] = ",".join(roles)
            headers["X-User-Role"] = roles[0]
        perms = user.get("permissions", [])
        if perms:
            headers["X-User-Permissions"] = ",".join(perms)

    # Forward request ID / correlation ID
    if rid := request.headers.get("x-request-id"):
        headers["X-Request-ID"] = rid
    if cid := request.headers.get("x-correlation-id"):
        headers["X-Correlation-ID"] = cid

    return headers
