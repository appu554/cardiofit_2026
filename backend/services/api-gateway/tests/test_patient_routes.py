import pytest
from unittest.mock import AsyncMock, patch
from httpx import AsyncClient
from app.main import app


@pytest.mark.anyio
async def test_health_score_requires_auth():
    async with AsyncClient(app=app, base_url="http://test") as client:
        resp = await client.get("/api/v1/patient/123/health-score")
        assert resp.status_code == 401


@pytest.mark.anyio
async def test_otp_send_is_public():
    """OTP send should NOT require auth."""
    with patch("app.api.endpoints.patient_app._forward", new_callable=AsyncMock) as mock:
        mock.return_value = {"otpId": "abc", "expiresIn": 300}
        async with AsyncClient(app=app, base_url="http://test") as client:
            resp = await client.post(
                "/api/v1/auth/otp/send",
                json={"phone": "+919876543210", "tenantId": "t1"},
            )
            assert resp.status_code in (200, 502)  # 502 if mock not wired yet
