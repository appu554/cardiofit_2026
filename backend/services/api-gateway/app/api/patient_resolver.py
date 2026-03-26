"""FHIR UUID → ABHA patient ID resolution via KB-20.

The gateway receives FHIR UUIDs from client apps (Flutter, React). Only KB-20
has the DB mapping; other KB services store data by ABHA ID. This module
resolves UUIDs before forwarding to non-KB-20 services.
"""
import logging
import re
from typing import Optional

import httpx
from fastapi import HTTPException

from app.config import settings

logger = logging.getLogger(__name__)

UUID_PATTERN = re.compile(
    r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$",
    re.IGNORECASE,
)

_CACHE_TTL = 3600  # 1 hour
_CACHE_PREFIX = "patient_resolve:"


async def resolve_patient_id(patient_id: str) -> str:
    """Resolve a FHIR UUID to its ABHA patient ID.

    If patient_id is not a UUID (already ABHA format), returns it unchanged.
    Results are cached in Redis with 1-hour TTL.
    Raises HTTPException(502) if KB-20 is unreachable.
    """
    if not UUID_PATTERN.match(patient_id):
        return patient_id

    # Try cache first
    cached = await _cache_get(patient_id)
    if cached is not None:
        return cached

    # Call KB-20
    abha_id = await _call_kb20_resolve(patient_id)

    # Cache the result
    await _cache_set(patient_id, abha_id)
    return abha_id


async def _call_kb20_resolve(fhir_uuid: str) -> str:
    """Call KB-20 profile endpoint to resolve FHIR UUID → ABHA ID."""
    url = f"{settings.KB20_SERVICE_URL}/api/v1/patient/{fhir_uuid}/profile"
    try:
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.get(url)
            if resp.status_code == 404:
                raise HTTPException(status_code=404, detail=f"Patient not found for FHIR ID {fhir_uuid}")
            resp.raise_for_status()
            data = resp.json()
            # KB-20 returns profile with patient_id field (ABHA format)
            abha_id = data.get("patient_id") or data.get("data", {}).get("patient_id")
            if not abha_id:
                raise HTTPException(status_code=502, detail="KB-20 response missing patient_id")
            return abha_id
    except httpx.ConnectError:
        logger.error("KB-20 unreachable for patient ID resolution: %s", fhir_uuid)
        raise HTTPException(status_code=502, detail="KB-20 service unavailable for patient ID resolution")
    except HTTPException:
        raise
    except Exception as e:
        logger.error("Patient ID resolution failed: %s", e)
        raise HTTPException(status_code=502, detail="Patient ID resolution error")


async def _cache_get(fhir_uuid: str) -> Optional[str]:
    """Look up cached UUID → ABHA mapping. Returns None on miss or Redis unavailable."""
    try:
        redis = _get_redis()
        if redis is None:
            return None
        val = await redis.get(f"{_CACHE_PREFIX}{fhir_uuid}")
        if val:
            return val  # decode_responses=True ensures string return
    except Exception:
        logger.debug("Redis cache get failed (non-fatal), skipping cache")
    return None


async def _cache_set(fhir_uuid: str, abha_id: str) -> None:
    """Store UUID → ABHA mapping in Redis. Fails silently if Redis unavailable."""
    try:
        redis = _get_redis()
        if redis is None:
            return
        await redis.set(f"{_CACHE_PREFIX}{fhir_uuid}", abha_id, ex=_CACHE_TTL)
    except Exception:
        logger.debug("Redis cache set failed (non-fatal), skipping cache")


_redis_client = None


def _get_redis():
    """Get async Redis client singleton, or None if Redis is not configured/available.

    Follows gateway's existing optional-Redis pattern (see patient_app.py:21-26).
    Uses a module-level singleton to avoid creating a new connection per call.
    """
    global _redis_client
    if _redis_client is not None:
        return _redis_client
    if not settings.REDIS_URL:
        return None
    try:
        import redis.asyncio as aioredis
        _redis_client = aioredis.from_url(settings.REDIS_URL, decode_responses=True)
        return _redis_client
    except Exception:
        return None
