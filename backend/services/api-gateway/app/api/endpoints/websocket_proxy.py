"""WebSocket proxy for Doctor Dashboard real-time subscriptions.

Bridges client WebSocket connections to the Apollo Federation subscription
endpoint, forwarding auth headers and Kafka events (CARD_GENERATED, MRI_UPDATED).
"""
from __future__ import annotations

import asyncio
import logging

import httpx
import websockets
from fastapi import APIRouter, WebSocket, WebSocketDisconnect

from app.config import settings  # module-level singleton, NOT get_settings()
from app.auth.auth_cache import AuthResponseCache

logger = logging.getLogger(__name__)
router = APIRouter(tags=["websocket"])

# Allowed origins for WebSocket upgrade (defense against cross-origin WS attacks)
ALLOWED_ORIGINS = set(settings.CORS_ALLOWED_ORIGINS.split(","))

# Auth cache shared with HTTP middleware — avoids re-verifying tokens
_ws_auth_cache = AuthResponseCache(
    ttl_seconds=settings.AUTH_CACHE_TTL_SECONDS,
    max_size=settings.AUTH_CACHE_MAX_SIZE,
)


async def _verify_ws_token(token: str) -> dict | None:
    """Verify a Bearer token via Auth Service (with cache). Returns user_info or None."""
    token_hash = _ws_auth_cache.hash_token(token)
    cached = _ws_auth_cache.get(token_hash)
    if cached is not None:
        return cached
    # Call Auth Service /verify
    try:
        auth_url = settings.AUTH_SERVICE_URL or "http://localhost:8001/api"
        verify_url = f"{auth_url}/api/auth/verify" if "/api" not in auth_url else f"{auth_url}/api/auth/verify"
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.post(verify_url, headers={"Authorization": f"Bearer {token}"})
            if resp.status_code != 200:
                return None
            result = resp.json()
            if not result.get("valid", False):
                return None
            user_info = result.get("user", {})
            _ws_auth_cache.put(token_hash, user_info)
            return user_info
    except Exception as e:
        logger.error("WebSocket auth verify failed: %s", e)
        return None


@router.websocket("/api/v1/doctor/subscriptions")
async def doctor_subscriptions(websocket: WebSocket):
    """Proxy WebSocket connections to Apollo Federation subscription endpoint."""
    # Validate origin
    origin = websocket.headers.get("origin", "")
    if ALLOWED_ORIGINS and origin not in ALLOWED_ORIGINS:
        await websocket.close(code=4003, reason="Origin not allowed")
        return

    # === Validate JWT via Auth Service BEFORE accepting WebSocket connection ===
    # HTTP middleware doesn't cover WebSocket upgrades, so we must validate here.
    auth_header = websocket.headers.get("authorization", "")
    if not auth_header.startswith("Bearer "):
        await websocket.close(code=4001, reason="Authentication required")
        return
    token = auth_header.split(" ", 1)[1]
    user_info = await _verify_ws_token(token)
    if user_info is None:
        await websocket.close(code=4001, reason="Invalid or expired token")
        return
    # Check doctor role
    roles = user_info.get("roles", [])
    if not any(r in roles for r in ("physician", "doctor", "nurse", "admin")):
        await websocket.close(code=4003, reason="Insufficient permissions")
        return
    # === End JWT validation ===

    await websocket.accept()

    # Build backend WS URL
    backend_url = settings.APOLLO_FEDERATION_URL.replace("http://", "ws://").replace("https://", "wss://")
    backend_url += "/subscriptions"

    # Forward validated user context headers
    extra_headers = {
        "Authorization": auth_header,
        "X-User-ID": user_info.get("id", ""),
        "X-User-Roles": ",".join(roles),
    }

    try:
        async with websockets.connect(backend_url, extra_headers=extra_headers) as backend_ws:
            # Bidirectional proxy
            async def client_to_backend():
                try:
                    while True:
                        data = await websocket.receive_text()
                        await backend_ws.send(data)
                except WebSocketDisconnect:
                    pass

            async def backend_to_client():
                try:
                    async for msg in backend_ws:
                        await websocket.send_text(msg)
                except websockets.ConnectionClosed:
                    pass

            await asyncio.gather(client_to_backend(), backend_to_client())

    except Exception as e:
        logger.error("WebSocket proxy error: %s", e)
        await websocket.close(code=1011, reason="Backend connection failed")
