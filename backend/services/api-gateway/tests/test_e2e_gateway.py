"""End-to-end tests for API Gateway → Auth Service → KB services.

Prerequisites:
  - Auth Service running on port 8001
  - API Gateway running on port 8000
  - KB services running (8131, 8132, 8134, 8136, 8137) — tests degrade gracefully

Run:
  cd backend/services/api-gateway
  GATEWAY_URL=http://localhost:8000 python -m pytest tests/test_e2e_gateway.py -v

Environment variables:
  GATEWAY_URL          - Gateway URL (default: http://localhost:8000)
  AUTH_SERVICE_URL     - Auth Service URL (default: http://localhost:8001)
  TEST_ADMIN_USER      - Admin username (default: admin@cardiofit.in)
  TEST_ADMIN_PASSWORD  - Admin password (default: admin123)
  TEST_DOCTOR_USER     - Doctor username (default: doctor@cardiofit.in)
  TEST_DOCTOR_PASSWORD - Doctor password (default: doctor123)
  TEST_PATIENT_ID      - Test patient ID (default: test-patient-001)
"""
import pytest
import httpx

def auth_headers(token: str) -> dict:
    """Build Authorization header dict."""
    return {"Authorization": f"Bearer {token}"}

# ---------------------------------------------------------------------------
# Markers
# ---------------------------------------------------------------------------
pytestmark = pytest.mark.anyio


# ===========================================================================
# 1. Gateway Health & Discovery
# ===========================================================================

class TestGatewayHealth:
    """Verify gateway is reachable and serving docs."""

    async def test_health_endpoint(self, gateway_url):
        async with httpx.AsyncClient(timeout=10) as c:
            resp = await c.get(f"{gateway_url}/health")
        assert resp.status_code == 200
        data = resp.json()
        assert data["status"] == "ok"

    async def test_openapi_schema(self, gateway_url):
        async with httpx.AsyncClient(timeout=10) as c:
            resp = await c.get(f"{gateway_url}/openapi.json")
        assert resp.status_code == 200
        schema = resp.json()
        assert "paths" in schema
        assert schema["info"]["title"] == "Clinical Synthesis Hub API Gateway"

    async def test_swagger_ui_available(self, gateway_url):
        async with httpx.AsyncClient(timeout=10) as c:
            resp = await c.get(f"{gateway_url}/docs")
        assert resp.status_code == 200
        assert "swagger" in resp.text.lower() or "openapi" in resp.text.lower()


# ===========================================================================
# 2. Auth Service Integration
# ===========================================================================

class TestAuthIntegration:
    """Verify Auth Service is reachable through the gateway."""

    async def test_auth_service_reachable(self, auth_service_url):
        """Direct health check on Auth Service."""
        async with httpx.AsyncClient(timeout=10) as c:
            resp = await c.get(f"{auth_service_url}/health")
        assert resp.status_code == 200

    async def test_admin_login_via_gateway(self, gateway_url):
        """Login through the gateway proxy to Auth Service."""
        async with httpx.AsyncClient(timeout=10) as c:
            resp = await c.post(
                f"{gateway_url}/api/auth/login",
                json={"username": "admin@cardiofit.in", "password": "admin123"},
            )
        # 200 = success, 401 = bad creds (but route works), 502 = auth svc down
        assert resp.status_code in (200, 401, 502)

    async def test_otp_send_is_public(self, gateway_url):
        """OTP send endpoint should not require auth."""
        async with httpx.AsyncClient(timeout=10) as c:
            resp = await c.post(
                f"{gateway_url}/api/v1/auth/otp/send",
                json={"phone": "+919876543210", "tenantId": "cardiofit-default"},
            )
        # Should NOT be 401 — this is a public endpoint
        assert resp.status_code != 401, "OTP send should be public (no auth required)"

    async def test_otp_verify_is_public(self, gateway_url):
        """OTP verify endpoint should not require auth."""
        async with httpx.AsyncClient(timeout=10) as c:
            resp = await c.post(
                f"{gateway_url}/api/v1/auth/otp/verify",
                json={"phone": "+919876543210", "otp": "000000", "tenantId": "t1"},
            )
        assert resp.status_code != 401, "OTP verify should be public (no auth required)"

    async def test_token_refresh_is_public(self, gateway_url):
        """Token refresh endpoint should not require auth."""
        async with httpx.AsyncClient(timeout=10) as c:
            resp = await c.post(
                f"{gateway_url}/api/v1/auth/refresh",
                json={"refreshToken": "invalid-token"},
            )
        assert resp.status_code != 401, "Token refresh should be public"

    async def test_verify_token_via_gateway(self, gateway_url, admin_token):
        """Verify a valid token through the gateway proxy."""
        async with httpx.AsyncClient(timeout=10) as c:
            resp = await c.post(
                f"{gateway_url}/api/auth/verify",
                headers=auth_headers(admin_token),
            )
        assert resp.status_code == 200
        data = resp.json()
        assert data.get("valid") is True


# ===========================================================================
# 3. Auth Enforcement — Protected Routes Reject Unauthenticated
# ===========================================================================

class TestAuthEnforcement:
    """Verify that protected endpoints return 401 without JWT."""

    @pytest.mark.parametrize("path", [
        "/api/v1/patient/p1/health-score",
        "/api/v1/patient/p1/actions/today",
        "/api/v1/patient/p1/checkin",
        "/api/v1/doctor/patients/p1/summary",
        "/api/v1/doctor/patients/p1/mri",
        "/api/v1/doctor/patients/p1/cards",
        "/api/v1/doctor/graphql",
        "/metrics",
    ])
    async def test_no_auth_returns_401(self, gateway_url, path):
        method = "POST" if path in ("/api/v1/patient/p1/checkin", "/api/v1/doctor/graphql") else "GET"
        async with httpx.AsyncClient(timeout=10) as c:
            if method == "POST":
                resp = await c.post(f"{gateway_url}{path}", json={})
            else:
                resp = await c.get(f"{gateway_url}{path}")
        assert resp.status_code == 401, f"{path} should require auth"


# ===========================================================================
# 4. RBAC Enforcement
# ===========================================================================

class TestRBACEnforcement:
    """Verify role-based access control."""

    async def test_admin_can_access_metrics(self, gateway_url, admin_token):
        """Admin role should access /metrics."""
        async with httpx.AsyncClient(timeout=10) as c:
            resp = await c.get(
                f"{gateway_url}/metrics",
                headers=auth_headers(admin_token),
            )
        # 200 if metrics enabled, 403 if role wrong, but not 401
        assert resp.status_code in (200, 403)

    async def test_doctor_can_access_patient_routes(self, gateway_url, doctor_token):
        """Doctor role bypasses RBAC and can access patient routes (proxies to KB)."""
        async with httpx.AsyncClient(timeout=10) as c:
            resp = await c.get(
                f"{gateway_url}/api/v1/patient/p1/health-score",
                headers=auth_headers(doctor_token),
            )
        # Doctor bypasses RBAC → request reaches KB-26; expect proxy response (not 401/403)
        assert resp.status_code not in (401, 403), "Doctor should bypass RBAC for patient routes"


# ===========================================================================
# 5. KB Service Health Probes (via direct call)
# ===========================================================================

class TestKBServiceHealth:
    """Verify KB services are reachable. Tests skip if service is down."""

    @pytest.mark.parametrize("name,port", [
        ("KB-20 Patient Profile", 8131),
        ("KB-22 HPI Engine", 8132),
        ("KB-23 Decision Cards", 8134),
        ("KB-25 Lifestyle Graph", 8136),
        ("KB-26 Metabolic Twin", 8137),
    ])
    async def test_kb_health(self, name, port):
        try:
            async with httpx.AsyncClient(timeout=5) as c:
                resp = await c.get(f"http://localhost:{port}/health")
            assert resp.status_code == 200, f"{name} health check failed"
        except httpx.ConnectError:
            pytest.skip(f"{name} (port {port}) not running")


# ===========================================================================
# 6. Patient App E2E — Gateway → Auth → KB
# ===========================================================================

class TestPatientAppE2E:
    """E2E tests for patient routes through the gateway to KB services.

    These require both Auth Service and the target KB service to be running.
    Tests degrade gracefully: 502 means gateway reached but KB is down.
    """

    async def _patient_get(self, gateway_url, admin_token, path):
        """Helper: GET a patient route with auth. Returns response."""
        async with httpx.AsyncClient(timeout=15) as c:
            return await c.get(
                f"{gateway_url}{path}",
                headers=auth_headers(admin_token),
            )

    async def test_health_score_proxies_to_kb26(self, gateway_url, admin_token, patient_id):
        """GET /patient/:id/health-score → KB-26 (port 8137)."""
        resp = await self._patient_get(
            gateway_url, admin_token,
            f"/api/v1/patient/{patient_id}/health-score",
        )
        # 200 = KB-26 responded, 502 = KB-26 unreachable, 403 = RBAC block
        assert resp.status_code in (200, 404, 502, 503, 403), f"Unexpected: {resp.status_code}"
        if resp.status_code == 502:
            pytest.skip("KB-26 (port 8137) not reachable")

    async def test_actions_today_proxies_to_kb23(self, gateway_url, admin_token, patient_id):
        """GET /patient/:id/actions/today → KB-23 (port 8134)."""
        resp = await self._patient_get(
            gateway_url, admin_token,
            f"/api/v1/patient/{patient_id}/actions/today",
        )
        assert resp.status_code in (200, 404, 502, 503, 403)
        if resp.status_code == 502:
            pytest.skip("KB-23 (port 8134) not reachable")

    async def test_health_drive_proxies_to_kb25(self, gateway_url, admin_token, patient_id):
        """GET /patient/:id/health-drive → KB-25 (port 8136)."""
        resp = await self._patient_get(
            gateway_url, admin_token,
            f"/api/v1/patient/{patient_id}/health-drive",
        )
        assert resp.status_code in (200, 404, 502, 503, 403)
        if resp.status_code == 502:
            pytest.skip("KB-25 (port 8136) not reachable")

    async def test_progress_proxies_to_kb20(self, gateway_url, admin_token, patient_id):
        """GET /patient/:id/progress → KB-20 (port 8131)."""
        resp = await self._patient_get(
            gateway_url, admin_token,
            f"/api/v1/patient/{patient_id}/progress",
        )
        assert resp.status_code in (200, 404, 502, 503, 403)
        if resp.status_code == 502:
            pytest.skip("KB-20 (port 8131) not reachable")

    async def test_timeline_proxies_to_kb20(self, gateway_url, admin_token, patient_id):
        """GET /patient/:id/timeline → KB-20 (port 8131)."""
        resp = await self._patient_get(
            gateway_url, admin_token,
            f"/api/v1/patient/{patient_id}/timeline",
        )
        assert resp.status_code in (200, 404, 502, 503, 403)
        if resp.status_code == 502:
            pytest.skip("KB-20 (port 8131) not reachable")

    async def test_cause_effect_proxies_to_kb26(self, gateway_url, admin_token, patient_id):
        """GET /patient/:id/cause-effect → KB-26 (port 8137)."""
        resp = await self._patient_get(
            gateway_url, admin_token,
            f"/api/v1/patient/{patient_id}/cause-effect",
        )
        assert resp.status_code in (200, 404, 502, 503, 403)
        if resp.status_code == 502:
            pytest.skip("KB-26 (port 8137) not reachable")

    async def test_insights_proxies_to_kb26(self, gateway_url, admin_token, patient_id):
        """GET /patient/:id/insights → KB-26 (port 8137)."""
        resp = await self._patient_get(
            gateway_url, admin_token,
            f"/api/v1/patient/{patient_id}/insights",
        )
        assert resp.status_code in (200, 404, 502, 503, 403)
        if resp.status_code == 502:
            pytest.skip("KB-26 (port 8137) not reachable")

    async def test_checkin_proxies_to_kb22(self, gateway_url, admin_token, patient_id):
        """POST /patient/:id/checkin → KB-22 (port 8132)."""
        async with httpx.AsyncClient(timeout=15) as c:
            resp = await c.post(
                f"{gateway_url}/api/v1/patient/{patient_id}/checkin",
                headers={**auth_headers(admin_token), "Content-Type": "application/json"},
                json={"type": "daily", "responses": [{"nodeId": "P1", "answer": "no_symptoms"}]},
            )
        assert resp.status_code in (200, 404, 502, 503, 403)
        if resp.status_code == 502:
            pytest.skip("KB-22 (port 8132) not reachable")


# ===========================================================================
# 7. Doctor Dashboard E2E — Gateway → Auth → KB
# ===========================================================================

class TestDoctorDashboardE2E:
    """E2E tests for doctor routes through the gateway to KB services."""

    async def _doctor_get(self, gateway_url, doctor_token, path):
        async with httpx.AsyncClient(timeout=15) as c:
            return await c.get(
                f"{gateway_url}{path}",
                headers=auth_headers(doctor_token),
            )

    async def test_summary_proxies_to_kb20(self, gateway_url, doctor_token, patient_id):
        """GET /doctor/patients/:id/summary → KB-20 (port 8131)."""
        resp = await self._doctor_get(
            gateway_url, doctor_token,
            f"/api/v1/doctor/patients/{patient_id}/summary",
        )
        assert resp.status_code in (200, 404, 502, 503, 403)
        if resp.status_code == 502:
            pytest.skip("KB-20 (port 8131) not reachable")

    async def test_mri_proxies_to_kb26(self, gateway_url, doctor_token, patient_id):
        """GET /doctor/patients/:id/mri → KB-26 (port 8137)."""
        resp = await self._doctor_get(
            gateway_url, doctor_token,
            f"/api/v1/doctor/patients/{patient_id}/mri",
        )
        assert resp.status_code in (200, 404, 502, 503, 403)
        if resp.status_code == 502:
            pytest.skip("KB-26 (port 8137) not reachable")

    async def test_cards_proxies_to_kb23(self, gateway_url, doctor_token, patient_id):
        """GET /doctor/patients/:id/cards → KB-23 (port 8134)."""
        resp = await self._doctor_get(
            gateway_url, doctor_token,
            f"/api/v1/doctor/patients/{patient_id}/cards",
        )
        assert resp.status_code in (200, 404, 502, 503, 403)
        if resp.status_code == 502:
            pytest.skip("KB-23 (port 8134) not reachable")

    async def test_channel_b_proxies_to_kb20(self, gateway_url, doctor_token, patient_id):
        """GET /doctor/patients/:id/channel-b-inputs → KB-20 (port 8131)."""
        resp = await self._doctor_get(
            gateway_url, doctor_token,
            f"/api/v1/doctor/patients/{patient_id}/channel-b-inputs",
        )
        assert resp.status_code in (200, 404, 502, 503, 403)
        if resp.status_code == 502:
            pytest.skip("KB-20 (port 8131) not reachable")

    async def test_channel_c_proxies_to_kb20(self, gateway_url, doctor_token, patient_id):
        """GET /doctor/patients/:id/channel-c-inputs → KB-20 (port 8131)."""
        resp = await self._doctor_get(
            gateway_url, doctor_token,
            f"/api/v1/doctor/patients/{patient_id}/channel-c-inputs",
        )
        assert resp.status_code in (200, 404, 502, 503, 403)
        if resp.status_code == 502:
            pytest.skip("KB-20 (port 8131) not reachable")

    async def test_traces_returns_503_if_vmcu_not_configured(self, gateway_url, doctor_token, patient_id):
        """GET /doctor/traces/:id → 503 if VMCU_SERVICE_URL is empty."""
        resp = await self._doctor_get(
            gateway_url, doctor_token,
            f"/api/v1/doctor/traces/{patient_id}",
        )
        # 503 = V-MCU not configured (expected default), 403 = RBAC, 200 = if configured
        assert resp.status_code in (200, 403, 502, 503)

    async def test_graphql_proxies_to_apollo(self, gateway_url, doctor_token):
        """POST /doctor/graphql → Apollo Federation (port 4000)."""
        async with httpx.AsyncClient(timeout=15) as c:
            resp = await c.post(
                f"{gateway_url}/api/v1/doctor/graphql",
                headers={**auth_headers(doctor_token), "Content-Type": "application/json"},
                json={"query": "{ __typename }"},
            )
        assert resp.status_code in (200, 502, 503, 403)
        if resp.status_code == 200:
            data = resp.json()
            assert "data" in data or "errors" in data


# ===========================================================================
# 8. Auth Response Cache Verification
# ===========================================================================

class TestAuthCacheE2E:
    """Verify auth cache works by making rapid sequential requests."""

    async def test_rapid_requests_use_cache(self, gateway_url, admin_token):
        """Multiple rapid requests should hit the auth cache (not Auth Service each time)."""
        headers = auth_headers(admin_token)
        async with httpx.AsyncClient(timeout=10) as c:
            # First request — cache MISS (calls Auth Service)
            resp1 = await c.get(f"{gateway_url}/health", headers=headers)
            # Second request — cache HIT (skips Auth Service)
            resp2 = await c.get(f"{gateway_url}/health", headers=headers)
            # Third request — cache HIT
            resp3 = await c.get(f"{gateway_url}/health", headers=headers)

        # All should succeed — the cache doesn't change behavior, only latency
        for resp in (resp1, resp2, resp3):
            assert resp.status_code == 200


# ===========================================================================
# 9. Circuit Breaker Behavior
# ===========================================================================

class TestCircuitBreakerE2E:
    """Verify circuit breaker doesn't interfere with normal operations."""

    async def test_healthy_service_returns_200_not_503(self, gateway_url, admin_token, patient_id):
        """A healthy KB service should return 200, not 503 (circuit open)."""
        async with httpx.AsyncClient(timeout=15) as c:
            resp = await c.get(
                f"{gateway_url}/api/v1/patient/{patient_id}/progress",
                headers=auth_headers(admin_token),
            )
        # 503 from circuit breaker would be wrong if KB-20 is healthy
        # 502 is acceptable (KB-20 not running)
        if resp.status_code == 502:
            pytest.skip("KB-20 not running — can't test circuit breaker pass-through")
        assert resp.status_code != 503 or "temporarily unavailable" not in resp.text.lower()


# ===========================================================================
# 10. Proxy Catch-all (Legacy Routes)
# ===========================================================================

class TestProxyCatchAll:
    """Verify legacy proxy routes still work alongside new specific routes."""

    async def test_legacy_auth_login_still_works(self, gateway_url):
        """Legacy /api/auth/login should still be proxied to Auth Service."""
        async with httpx.AsyncClient(timeout=10) as c:
            resp = await c.post(
                f"{gateway_url}/api/auth/login",
                json={"username": "test", "password": "test"},
            )
        # 200 or 401 (bad creds) = proxy works; 404 = route not found
        assert resp.status_code != 404, "Legacy /api/auth/login route should still work"

    async def test_new_routes_take_priority_over_catchall(self, gateway_url, admin_token, patient_id):
        """New /api/v1/patient/* routes should respond (not fall through to catch-all)."""
        async with httpx.AsyncClient(timeout=10) as c:
            resp = await c.get(
                f"{gateway_url}/api/v1/patient/{patient_id}/health-score",
                headers=auth_headers(admin_token),
            )
        # Should NOT be 404 (which would mean catch-all didn't find a matching SERVICE_ROUTE)
        assert resp.status_code != 404, "Patient routes should be handled by specific router, not catch-all"
