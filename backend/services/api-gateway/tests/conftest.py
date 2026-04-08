"""Shared fixtures for API Gateway tests."""
from __future__ import annotations

import os
from typing import Optional

import pytest
import httpx

# E2E test configuration — override with env vars
GATEWAY_URL = os.getenv("GATEWAY_URL", "http://localhost:8000")
AUTH_SERVICE_URL = os.getenv("AUTH_SERVICE_URL", "http://localhost:8001")

# Test credentials — set via env or use defaults for local dev
TEST_ADMIN_USER = os.getenv("TEST_ADMIN_USER", "doctor@vaidshala.com")
TEST_ADMIN_PASSWORD = os.getenv("TEST_ADMIN_PASSWORD", "test@554")
TEST_DOCTOR_USER = os.getenv("TEST_DOCTOR_USER", "doctor@vaidshala.com")
TEST_DOCTOR_PASSWORD = os.getenv("TEST_DOCTOR_PASSWORD", "test@554")
TEST_PATIENT_PHONE = os.getenv("TEST_PATIENT_PHONE", "+919876543210")
TEST_PATIENT_ID = os.getenv("TEST_PATIENT_ID", "test-patient-001")


@pytest.fixture(scope="session")
def gateway_url():
    return GATEWAY_URL


@pytest.fixture(scope="session")
def auth_service_url():
    return AUTH_SERVICE_URL


@pytest.fixture(scope="session")
def patient_id():
    return TEST_PATIENT_ID


async def _get_token_via_login(auth_url: str, username: str, password: str) -> Optional[str]:
    """Get JWT via Auth Service /api/auth/login."""
    try:
        async with httpx.AsyncClient(timeout=10) as client:
            resp = await client.post(
                f"{auth_url}/api/auth/login",
                json={"username": username, "password": password},
            )
            if resp.status_code == 200:
                data = resp.json()
                return data.get("access_token") or data.get("accessToken") or data.get("token")
    except Exception:
        pass
    return None


@pytest.fixture(scope="session")
def anyio_backend():
    return "asyncio"


@pytest.fixture(scope="session")
async def admin_token(auth_service_url):
    """Get an admin JWT for E2E tests."""
    token = await _get_token_via_login(auth_service_url, TEST_ADMIN_USER, TEST_ADMIN_PASSWORD)
    if not token:
        pytest.skip("Could not obtain admin token — is Auth Service running?")
    return token


@pytest.fixture(scope="session")
async def doctor_token(auth_service_url):
    """Get a doctor JWT for E2E tests."""
    token = await _get_token_via_login(auth_service_url, TEST_DOCTOR_USER, TEST_DOCTOR_PASSWORD)
    if not token:
        pytest.skip("Could not obtain doctor token — is Auth Service running?")
    return token


def auth_headers(token: str) -> dict:
    """Build Authorization header dict."""
    return {"Authorization": f"Bearer {token}"}
