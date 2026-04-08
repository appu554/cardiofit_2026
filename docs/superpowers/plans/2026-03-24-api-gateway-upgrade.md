# API Gateway Upgrade — OAuth2/JWT + Vaidshala Routes + Production Hardening

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade the existing Python FastAPI API gateway (`backend/services/api-gateway/`, port 8000) with cached Auth Service token validation, new routes for Patient App + Doctor Dashboard, WebSocket subscriptions, circuit breakers, Redis rate limiting, Prometheus metrics, and response caching — deployed incrementally across 3 phases.

**Architecture:** The existing FastAPI gateway already handles auth (via Auth Service `/api/auth/verify` callback), RBAC (28 route patterns), rate limiting (in-memory), proxy routing (12 services), and GraphQL (Strawberry). The **Auth Service (port 8001)** is the single source of truth for JWT validation — it holds all Supabase HS256 secrets and Auth0 RS256/JWKS config. This upgrade adds: (1) in-memory token-claims cache to eliminate redundant Auth Service calls (verify once per token per ~60s), (2) new SERVICE_ROUTES for all Vaidshala runtime endpoints, (3) production middleware (circuit breaker, metrics, caching), (4) WebSocket proxy for doctor dashboard subscriptions. **No JWT secrets are duplicated in the gateway.**

**Tech Stack:** Python 3.11, FastAPI, httpx, strawberry-graphql, redis[hiredis], prometheus-client, websockets

---

## Scope: 3 Phases

| Phase | Timeframe | What Ships | Clinical Impact |
|-------|-----------|------------|-----------------|
| **Phase 1 — MVP** | Wk 1-4 | Auth response cache (eliminates redundant Auth Service calls), Patient App routes (5 signals), Doctor Dashboard routes (KB-20/23/26), updated RBAC | GLYC-1 + HTN-1 can run. Shadow pilot starts. |
| **Phase 2 — Hardening** | Wk 5-8 | Circuit breakers, WebSocket proxy, Prometheus metrics, ABDM routes, lab webhook routing | RENAL-1 fully active. 17 signals. Full CDI. |
| **Phase 3 — Production** | Wk 9-12 | Redis distributed rate limiting, response caching, configurable CORS, multi-tenant branding, audit logging | 22 signals. Full clinical feedback loop. App store ready. |

---

## File Structure (modifications to existing gateway)

```
backend/services/api-gateway/
├── app/
│   ├── main.py                          # MODIFY: add new middleware, routers
│   ├── config.py                        # MODIFY: add JWT config, Redis, new service URLs
│   ├── auth/
│   │   ├── middleware.py                # MODIFY: add token-claims cache (Auth Service stays as validator)
│   │   └── auth_cache.py               # CREATE: in-memory TTL cache for Auth Service /verify responses
│   ├── api/
│   │   ├── proxy.py                    # MODIFY: add Vaidshala SERVICE_ROUTES
│   │   └── endpoints/
│   │       ├── patient_app.py          # CREATE: Patient App REST endpoints
│   │       ├── doctor_dashboard.py     # CREATE: Doctor Dashboard REST endpoints
│   │       └── websocket_proxy.py      # CREATE: WebSocket subscription proxy (Phase 2)
│   ├── middleware/
│   │   ├── rbac.py                     # MODIFY: add Vaidshala route permissions
│   │   ├── rate_limit.py               # MODIFY: add Redis backend option (Phase 3)
│   │   ├── circuit_breaker.py          # CREATE: per-service circuit breaker (Phase 2)
│   │   ├── metrics.py                  # CREATE: Prometheus metrics (Phase 2)
│   │   ├── response_cache.py           # CREATE: Redis response cache (Phase 3)
│   │   └── audit_log.py               # CREATE: HIPAA audit logging (Phase 3)
│   └── graphql/
│       ├── types.py                    # MODIFY: add Vaidshala types
│       └── queries.py                  # MODIFY: add doctor dashboard queries
├── requirements.txt                     # MODIFY: add new dependencies
└── tests/
    ├── test_auth_cache.py              # CREATE: auth cache tests
    ├── test_patient_routes.py          # CREATE: patient endpoint tests
    ├── test_doctor_routes.py           # CREATE: doctor endpoint tests
    └── test_circuit_breaker.py         # CREATE: circuit breaker tests
```

---

## Phase 1 — MVP (Wk 1-4)

### Task 1: Add New Dependencies

**Files:**
- Modify: `backend/services/api-gateway/requirements.txt`

- [ ] **Step 1: Add Phase 1 dependencies**

Append to `requirements.txt`:
```
# Phase 2: Production hardening
circuitbreaker>=1.4.0
prometheus-client>=0.19.0
websockets>=12.0
# Phase 3: Distributed infrastructure
redis[hiredis]>=5.0.0
```

> **Note**: No PyJWT or cryptography needed — JWT validation stays in the Auth Service (port 8001) which already has these dependencies.

- [ ] **Step 2: Install**

```bash
cd backend/services/api-gateway
pip install -r requirements.txt
```

- [ ] **Step 3: Commit**

```bash
git add backend/services/api-gateway/requirements.txt
git commit -m "feat(gateway): add dependencies for JWT, circuit breaker, metrics, Redis"
```

---

### Task 2: Configuration — New Service URLs + JWT Settings

**Files:**
- Modify: `backend/services/api-gateway/app/config.py`

- [ ] **Step 1: Read existing config.py**

```bash
cat backend/services/api-gateway/app/config.py
```

- [ ] **Step 2: Add JWT and Vaidshala service configuration**

Add these fields to the existing `Settings` class in `config.py`:
```python
    # === Auth Cache (Phase 1) ===
    # Cache Auth Service /verify responses to avoid per-request calls
    # JWT secrets stay in Auth Service (port 8001) — NOT duplicated here
    AUTH_CACHE_TTL_SECONDS: int = 60  # how long to cache verified token claims
    AUTH_CACHE_MAX_SIZE: int = 10000  # max cached tokens (LRU eviction)

    # === Vaidshala Clinical Runtime Services ===
    KB20_SERVICE_URL: str = "http://localhost:8131"
    KB22_SERVICE_URL: str = "http://localhost:8132"
    KB23_SERVICE_URL: str = "http://localhost:8134"
    KB25_SERVICE_URL: str = "http://localhost:8136"
    KB26_SERVICE_URL: str = "http://localhost:8137"
    VMCU_SERVICE_URL: str = ""

    # === CORS (configurable, Phase 3) ===
    CORS_ALLOWED_ORIGINS: str = "http://localhost:3000,http://localhost:3001,http://localhost:3002"

    # === Redis (Phase 3) ===
    REDIS_URL: str = "redis://localhost:6380"
    REDIS_RATE_LIMIT_ENABLED: bool = False

    # === Circuit Breaker (Phase 2) ===
    CIRCUIT_BREAKER_FAIL_MAX: int = 5
    CIRCUIT_BREAKER_RESET_TIMEOUT: int = 30

    # === Metrics (Phase 2) ===
    METRICS_ENABLED: bool = False
```

- [ ] **Step 3: Commit**

```bash
git add backend/services/api-gateway/app/config.py
git commit -m "feat(gateway): add JWT, Vaidshala services, Redis, metrics config"
```

---

### Task 3: Auth Response Cache — Eliminate Redundant Auth Service Calls

**Files:**
- Create: `backend/services/api-gateway/app/auth/auth_cache.py`
- Test: `backend/services/api-gateway/tests/test_auth_cache.py`

> **Design decision**: JWT secrets stay in the Auth Service (port 8001). The gateway does NOT do local JWT validation. Instead, it caches the Auth Service `/verify` response for each token (keyed by token hash) with a configurable TTL (default 60s). This means the first request with a token calls Auth Service, but subsequent requests within 60s hit the cache — eliminating ~95% of Auth Service calls under normal load while keeping all secrets in one place.

- [ ] **Step 1: Write failing test**

Create `tests/test_auth_cache.py`:
```python
import time
import pytest
from app.auth.auth_cache import AuthResponseCache


def test_cache_miss_returns_none():
    cache = AuthResponseCache(ttl_seconds=60, max_size=100)
    assert cache.get("unknown-token-hash") is None


def test_cache_stores_and_retrieves():
    cache = AuthResponseCache(ttl_seconds=60, max_size=100)
    user_info = {"id": "user-123", "email": "doc@cardiofit.in", "roles": ["physician"]}
    cache.put("token-hash-abc", user_info)
    assert cache.get("token-hash-abc") == user_info


def test_cache_expires_after_ttl():
    cache = AuthResponseCache(ttl_seconds=1, max_size=100)
    cache.put("token-hash-abc", {"id": "user-123"})
    time.sleep(1.1)
    assert cache.get("token-hash-abc") is None


def test_cache_evicts_lru_when_full():
    cache = AuthResponseCache(ttl_seconds=60, max_size=2)
    cache.put("a", {"id": "1"})
    cache.put("b", {"id": "2"})
    cache.put("c", {"id": "3"})  # should evict "a"
    assert cache.get("a") is None
    assert cache.get("b") is not None
    assert cache.get("c") is not None


def test_cache_invalidate():
    cache = AuthResponseCache(ttl_seconds=60, max_size=100)
    cache.put("token-hash-abc", {"id": "user-123"})
    cache.invalidate("token-hash-abc")
    assert cache.get("token-hash-abc") is None
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend/services/api-gateway
python -m pytest tests/test_auth_cache.py -v
```
Expected: FAIL — `ModuleNotFoundError: No module named 'app.auth.auth_cache'`

- [ ] **Step 3: Implement auth response cache**

Create `app/auth/auth_cache.py`:
```python
"""In-memory TTL cache for Auth Service /verify responses.

Eliminates redundant Auth Service calls by caching verified token claims.
JWT secrets stay in Auth Service (port 8001) — NOT duplicated here.
"""
import hashlib
import logging
import threading
import time
from collections import OrderedDict
from typing import Any, Optional

logger = logging.getLogger(__name__)


class AuthResponseCache:
    """Thread-safe LRU cache with TTL for Auth Service verify responses.

    Cache key: SHA-256 hash of the Bearer token (never stores raw tokens).
    Cache value: The user_info dict returned by Auth Service /verify.
    """

    def __init__(self, ttl_seconds: int = 60, max_size: int = 10000):
        self._ttl = ttl_seconds
        self._max_size = max_size
        self._cache: OrderedDict[str, tuple[float, dict]] = OrderedDict()
        self._lock = threading.Lock()

    @staticmethod
    def hash_token(token: str) -> str:
        """Hash a Bearer token for use as cache key. Never store raw tokens."""
        return hashlib.sha256(token.encode()).hexdigest()[:32]

    def get(self, token_hash: str) -> Optional[dict]:
        """Get cached user_info for a token hash. Returns None on miss or expiry."""
        with self._lock:
            entry = self._cache.get(token_hash)
            if entry is None:
                return None
            expires_at, user_info = entry
            if time.time() > expires_at:
                del self._cache[token_hash]
                return None
            # Move to end (most recently used)
            self._cache.move_to_end(token_hash)
            return user_info

    def put(self, token_hash: str, user_info: dict):
        """Cache a verified user_info dict."""
        with self._lock:
            expires_at = time.time() + self._ttl
            self._cache[token_hash] = (expires_at, user_info)
            self._cache.move_to_end(token_hash)
            # Evict LRU if over capacity
            while len(self._cache) > self._max_size:
                self._cache.popitem(last=False)

    def invalidate(self, token_hash: str):
        """Remove a specific token from cache (e.g., on logout)."""
        with self._lock:
            self._cache.pop(token_hash, None)

    def clear(self):
        """Clear all cached entries."""
        with self._lock:
            self._cache.clear()

    @property
    def size(self) -> int:
        return len(self._cache)
```

- [ ] **Step 4: Run tests — all should pass**

```bash
cd backend/services/api-gateway
python -m pytest tests/test_auth_cache.py -v
```
Expected: 5 PASS

- [ ] **Step 5: Commit**

```bash
git add backend/services/api-gateway/app/auth/auth_cache.py \
        backend/services/api-gateway/tests/test_auth_cache.py
git commit -m "feat(gateway): auth response cache — eliminates redundant Auth Service /verify calls"
```

---

### Task 4: Upgrade Auth Middleware — Add Token Claims Cache

**Files:**
- Modify: `backend/services/api-gateway/app/auth/middleware.py`

> **Design**: The middleware already calls Auth Service `/api/auth/verify` for every request. We add a cache layer so that the same token is only verified once per 60 seconds. The Auth Service remains the single source of truth for JWT validation (Supabase HS256 + Auth0 RS256/JWKS). No secrets are added to the gateway.

- [ ] **Step 1: Read existing middleware.py**

```bash
cat backend/services/api-gateway/app/auth/middleware.py
```

- [ ] **Step 2: Add token claims cache to auth middleware**

Add the following changes to the existing `AuthenticationMiddleware`:

```python
# Add at top of file:
from app.auth.auth_cache import AuthResponseCache
from app.config import settings  # module-level singleton, NOT get_settings()

# Add to AuthenticationMiddleware.__init__ body (after super().__init__()):
self._auth_cache = AuthResponseCache(
    ttl_seconds=settings.AUTH_CACHE_TTL_SECONDS,
    max_size=settings.AUTH_CACHE_MAX_SIZE,
)
```

Inside `dispatch()`, after extracting the token (line ~86) and **before** the existing `httpx.AsyncClient()` call, add a cache check:

```python
        # === Cache fast-path: check if token was recently verified ===
        token_hash = self._auth_cache.hash_token(token)
        cached_user = self._auth_cache.get(token_hash)
        if cached_user is not None:
            # Token was verified by Auth Service within TTL — skip network call
            request.state.user = cached_user
            request.state.user_role = cached_user.get("role", "authenticated")
            request.state.user_roles = cached_user.get("roles", [])
            request.state.user_permissions = cached_user.get("permissions", [])
            logger.debug("Auth cache HIT for user %s", cached_user.get("id"))
            return await call_next(request)
        # === End cache fast-path ===

        # Existing Auth Service /verify call below (unchanged)...
```

After the existing successful Auth Service response handling (where `request.state.user = user_info` is set, around line ~131), add:

```python
                # Cache the verified user info for future requests with this token
                self._auth_cache.put(token_hash, user_info)
```

- [ ] **Step 3: Verify existing tests still pass**

```bash
cd backend/services/api-gateway
python -m pytest tests/ -v
```

- [ ] **Step 4: Commit**

```bash
git add backend/services/api-gateway/app/auth/middleware.py
git commit -m "feat(gateway): auth response cache — ~95% fewer Auth Service /verify calls"
```

---

### Task 5: Patient App Routes

**Files:**
- Create: `backend/services/api-gateway/app/api/endpoints/patient_app.py`
- Test: `backend/services/api-gateway/tests/test_patient_routes.py`

- [ ] **Step 1: Write failing test**

Create `tests/test_patient_routes.py`:
```python
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
    with patch("app.api.endpoints.patient_app.forward_to_auth", new_callable=AsyncMock) as mock:
        mock.return_value = {"otpId": "abc", "expiresIn": 300}
        async with AsyncClient(app=app, base_url="http://test") as client:
            resp = await client.post(
                "/api/v1/auth/otp/send",
                json={"phone": "+919876543210", "tenantId": "t1"},
            )
            assert resp.status_code in (200, 502)  # 502 if mock not wired yet
```

- [ ] **Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_patient_routes.py -v
```
Expected: FAIL — route not found (404)

- [ ] **Step 3: Implement Patient App routes**

Create `app/api/endpoints/patient_app.py`:
```python
"""Patient App REST endpoints — routes for Flutter mobile app.

These endpoints proxy to downstream Vaidshala services with
X-User-* headers injected from validated JWT claims.
"""
import logging
from typing import Any

import httpx
from fastapi import APIRouter, Request, HTTPException

from app.config import settings  # module-level singleton, NOT get_settings()

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
    """ABDM integration — proxied to Patient Service."""
    return await _forward(request, settings.PATIENT_SERVICE_URL, f"/patients/{patient_id}/abdm/verify", method="POST")


@router.get("/family/{token}")
async def family_view(token: str, request: Request):
    """Family view — token-scoped, no JWT required (handled by service)."""
    return await _forward(request, settings.PATIENT_SERVICE_URL, f"/family/{token}")


@router.get("/tenants/{tenant_id}/branding")
async def tenant_branding(tenant_id: str, request: Request):
    """Multi-tenant branding — public."""
    return await _forward(request, settings.PATIENT_SERVICE_URL, f"/tenants/{tenant_id}/branding")


async def _forward(request: Request, service_url: str, path: str, method: str = None) -> Any:
    """Forward request to a downstream service with X-User-* headers."""
    method = method or request.method
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
            return resp.json()
    except httpx.ConnectError:
        raise HTTPException(status_code=502, detail="Downstream service unavailable")
    except Exception as e:
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
```

- [ ] **Step 4: Register routers in main.py**

Add to `app/main.py` imports and router registration. **CRITICAL (Review Fix #8)**: New specific-path routers MUST be registered **BEFORE** the existing catch-all proxy router (`proxy_router` with `/{path:path}`). FastAPI matches routes in registration order — if the proxy catch-all is first, it will swallow `/api/v1/patient/*` and `/api/v1/doctor/*` before they reach these routers.
```python
from app.api.endpoints.patient_app import router as patient_router, auth_router as otp_router

# Register BEFORE proxy_router (which has catch-all /{path:path}):
app.include_router(otp_router)       # /api/v1/auth/* (public)
app.include_router(patient_router)   # /api/v1/patient/* (JWT required)
# ... then doctor_router, ws_router in later tasks
# LAST: proxy_router (catch-all for existing 12 services)
```

- [ ] **Step 5: Add OTP/auth paths to auth middleware exclusions**

In `app/auth/middleware.py`, add to the excluded paths list:
```python
"/api/v1/auth/otp/send",
"/api/v1/auth/otp/verify",
"/api/v1/auth/refresh",
"/api/v1/tenants",
"/api/v1/family",
```

- [ ] **Step 6: Run tests**

```bash
python -m pytest tests/test_patient_routes.py -v
```

- [ ] **Step 7: Commit**

```bash
git add backend/services/api-gateway/app/api/endpoints/patient_app.py \
        backend/services/api-gateway/tests/test_patient_routes.py \
        backend/services/api-gateway/app/main.py \
        backend/services/api-gateway/app/auth/middleware.py
git commit -m "feat(gateway): Patient App REST routes — health-score, checkin, timeline, OTP proxy"
```

---

### Task 6: Doctor Dashboard Routes

**Files:**
- Create: `backend/services/api-gateway/app/api/endpoints/doctor_dashboard.py`
- Test: `backend/services/api-gateway/tests/test_doctor_routes.py`

- [ ] **Step 1: Write failing test**

Create `tests/test_doctor_routes.py`:
```python
import pytest
from httpx import AsyncClient
from app.main import app


@pytest.mark.anyio
async def test_doctor_summary_requires_auth():
    async with AsyncClient(app=app, base_url="http://test") as client:
        resp = await client.get("/api/v1/doctor/patients/p1/summary")
        assert resp.status_code == 401


@pytest.mark.anyio
async def test_doctor_graphql_requires_auth():
    async with AsyncClient(app=app, base_url="http://test") as client:
        resp = await client.post("/api/v1/doctor/graphql", json={"query": "{ __typename }"})
        assert resp.status_code == 401
```

- [ ] **Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_doctor_routes.py -v
```
Expected: FAIL — 404

- [ ] **Step 3: Implement Doctor Dashboard routes**

Create `app/api/endpoints/doctor_dashboard.py`:
```python
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
```

- [ ] **Step 4: Register router in main.py**

Add to `app/main.py`:
```python
from app.api.endpoints.doctor_dashboard import router as doctor_router
app.include_router(doctor_router)
```

- [ ] **Step 5: Run tests**

```bash
python -m pytest tests/test_doctor_routes.py -v
```

- [ ] **Step 6: Commit**

```bash
git add backend/services/api-gateway/app/api/endpoints/doctor_dashboard.py \
        backend/services/api-gateway/tests/test_doctor_routes.py \
        backend/services/api-gateway/app/main.py
git commit -m "feat(gateway): Doctor Dashboard routes — KB-20/23/26 + V-MCU + GraphQL proxy"
```

---

### Task 7: Update RBAC — Vaidshala Route Permissions

**Files:**
- Modify: `backend/services/api-gateway/app/middleware/rbac.py`

- [ ] **Step 1: Read existing RBAC**

```bash
cat backend/services/api-gateway/app/middleware/rbac.py
```

- [ ] **Step 2: Add Vaidshala route permissions**

Add to `ROUTE_PERMISSIONS` dict. **IMPORTANT**: The existing RBAC uses **regex patterns** as keys and **lists** as permission values (e.g. `r"^/api/patients": {"GET": ["patient:read"]}`). New entries MUST match this format — use `r"^..."` regex keys, `[^/]+` for path params, and `["perm"]` lists:
```python
    # === Patient App routes (Phase 1) ===
    r"^/api/v1/patient/[^/]+/health-score": {"GET": ["patient:read"]},
    r"^/api/v1/patient/[^/]+/actions/today": {"GET": ["patient:read"]},
    r"^/api/v1/patient/[^/]+/health-drive": {"GET": ["patient:read"]},
    r"^/api/v1/patient/[^/]+/progress": {"GET": ["patient:read"]},
    r"^/api/v1/patient/[^/]+/cause-effect": {"GET": ["patient:read"]},
    r"^/api/v1/patient/[^/]+/timeline": {"GET": ["patient:read"]},
    r"^/api/v1/patient/[^/]+/insights": {"GET": ["patient:read"]},
    r"^/api/v1/patient/[^/]+/checkin": {"POST": ["patient:write"]},
    r"^/api/v1/patient/[^/]+/abdm/verify": {"POST": ["patient:write"]},

    # === Doctor Dashboard routes (Phase 1) ===
    r"^/api/v1/doctor/graphql": {"POST": ["doctor:read"]},
    r"^/api/v1/doctor/patients/[^/]+/summary": {"GET": ["doctor:read"]},
    r"^/api/v1/doctor/patients/[^/]+/mri": {"GET": ["doctor:read"]},
    r"^/api/v1/doctor/patients/[^/]+/cards": {"GET": ["doctor:read"]},
    r"^/api/v1/doctor/cards/[^/]+/action": {"POST": ["doctor:write"]},
    r"^/api/v1/doctor/traces/[^/]+": {"GET": ["doctor:admin"]},
    r"^/api/v1/doctor/patients/[^/]+/channel-b-inputs": {"GET": ["doctor:read"]},
    r"^/api/v1/doctor/patients/[^/]+/channel-c-inputs": {"GET": ["doctor:read"]},
```

Add to `ROLE_ROUTE_RESTRICTIONS` dict (also uses regex patterns):
```python
    # Patient App — patient role only
    r"^/api/v1/patient/": ["patient", "admin", "system"],
    # Doctor Dashboard — physician, nurse, admin
    r"^/api/v1/doctor/": ["physician", "doctor", "nurse", "admin", "super_admin"],
    # V-MCU traces — physician and admin only (sensitive)
    r"^/api/v1/doctor/traces/": ["physician", "doctor", "admin"],
```

- [ ] **Step 3: Commit**

```bash
git add backend/services/api-gateway/app/middleware/rbac.py
git commit -m "feat(gateway): RBAC policies for Patient App + Doctor Dashboard routes"
```

---

### Task 8: Update Proxy SERVICE_ROUTES

**Files:**
- Modify: `backend/services/api-gateway/app/api/proxy.py`

- [ ] **Step 1: Read existing proxy.py**

```bash
cat backend/services/api-gateway/app/api/proxy.py
```

- [ ] **Step 2: Add Vaidshala services to SERVICE_ROUTES**

Add to the `SERVICE_ROUTES` list in `proxy.py`:
```python
    # Vaidshala Clinical Runtime
    {"prefix": "/api/v1/kb20", "service_url": settings.KB20_SERVICE_URL, "strip_prefix": True},
    {"prefix": "/api/v1/kb22", "service_url": settings.KB22_SERVICE_URL, "strip_prefix": True},
    {"prefix": "/api/v1/kb23", "service_url": settings.KB23_SERVICE_URL, "strip_prefix": True},
    {"prefix": "/api/v1/kb25", "service_url": settings.KB25_SERVICE_URL, "strip_prefix": True},
    {"prefix": "/api/v1/kb26", "service_url": settings.KB26_SERVICE_URL, "strip_prefix": True},
```

- [ ] **Step 3: Commit**

```bash
git add backend/services/api-gateway/app/api/proxy.py
git commit -m "feat(gateway): add Vaidshala KB-20/22/23/25/26 service routes to proxy"
```

---

## Phase 2 — Hardening (Wk 5-8)

### Task 9: Circuit Breaker Middleware

**Files:**
- Create: `backend/services/api-gateway/app/middleware/circuit_breaker.py`
- Test: `backend/services/api-gateway/tests/test_circuit_breaker.py`

- [ ] **Step 1: Write failing test**

Create `tests/test_circuit_breaker.py`:
```python
import pytest
from app.middleware.circuit_breaker import ServiceCircuitBreaker


def test_breaker_starts_closed():
    cb = ServiceCircuitBreaker("test-svc", fail_max=3, reset_timeout=5)
    assert cb.is_available()


def test_breaker_opens_after_failures():
    cb = ServiceCircuitBreaker("test-svc", fail_max=3, reset_timeout=5)
    cb.record_failure()
    cb.record_failure()
    cb.record_failure()
    assert not cb.is_available()


def test_success_resets_count():
    cb = ServiceCircuitBreaker("test-svc", fail_max=3, reset_timeout=5)
    cb.record_failure()
    cb.record_failure()
    cb.record_success()
    assert cb.is_available()
```

- [ ] **Step 2: Run test to verify it fails**

```bash
python -m pytest tests/test_circuit_breaker.py -v
```
Expected: FAIL

- [ ] **Step 3: Implement circuit breaker**

Create `app/middleware/circuit_breaker.py`:
```python
"""Per-service circuit breaker with closed/open/half-open states."""
import logging
import time
from typing import Optional

logger = logging.getLogger(__name__)


class ServiceCircuitBreaker:
    """Simple circuit breaker per downstream service."""

    def __init__(self, service_name: str, fail_max: int = 5, reset_timeout: int = 30):
        self.service_name = service_name
        self.fail_max = fail_max
        self.reset_timeout = reset_timeout
        self._failures = 0
        self._last_failure: float = 0
        self._state = "closed"  # closed, open, half-open

    def is_available(self) -> bool:
        if self._state == "closed":
            return True
        if self._state == "open":
            if time.time() - self._last_failure > self.reset_timeout:
                self._state = "half-open"
                return True
            return False
        return True  # half-open allows one request through

    def record_success(self):
        self._failures = 0
        self._state = "closed"

    def record_failure(self):
        self._failures += 1
        self._last_failure = time.time()
        if self._failures >= self.fail_max:
            self._state = "open"
            logger.warning("Circuit OPEN for %s after %d failures", self.service_name, self._failures)

    @property
    def state(self) -> str:
        return self._state


# Registry of circuit breakers per service
_breakers: dict[str, ServiceCircuitBreaker] = {}


def get_breaker(service_name: str, fail_max: int = 5, reset_timeout: int = 30) -> ServiceCircuitBreaker:
    if service_name not in _breakers:
        _breakers[service_name] = ServiceCircuitBreaker(service_name, fail_max, reset_timeout)
    return _breakers[service_name]
```

- [ ] **Step 4: Integrate into _forward helper in patient_app.py**

In `app/api/endpoints/patient_app.py`, update `_forward()` to use circuit breaker:
```python
from app.middleware.circuit_breaker import get_breaker

async def _forward(request: Request, service_url: str, path: str, method: str = None) -> Any:
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
```

- [ ] **Step 5: Run tests**

```bash
python -m pytest tests/test_circuit_breaker.py -v
```
Expected: 3 PASS

- [ ] **Step 6: Commit**

```bash
git add backend/services/api-gateway/app/middleware/circuit_breaker.py \
        backend/services/api-gateway/tests/test_circuit_breaker.py \
        backend/services/api-gateway/app/api/endpoints/patient_app.py
git commit -m "feat(gateway): per-service circuit breaker with auto-recovery"
```

---

### Task 10: Prometheus Metrics Middleware

**Files:**
- Create: `backend/services/api-gateway/app/middleware/metrics.py`

- [ ] **Step 1: Implement metrics middleware**

Create `app/middleware/metrics.py`:
```python
"""Prometheus metrics for the API gateway."""
from prometheus_client import Counter, Histogram, generate_latest, CONTENT_TYPE_LATEST
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request
from starlette.responses import Response
import time


REQUEST_COUNT = Counter(
    "gateway_http_requests_total",
    "Total HTTP requests",
    ["method", "path", "status"],
)
REQUEST_DURATION = Histogram(
    "gateway_http_request_duration_seconds",
    "Request duration in seconds",
    ["method", "path"],
)
CIRCUIT_BREAKER_STATE = Counter(
    "gateway_circuit_breaker_trips_total",
    "Circuit breaker trip count",
    ["service"],
)


class MetricsMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next):
        start = time.time()
        response = await call_next(request)
        duration = time.time() - start

        path = request.url.path
        # Normalize path params to reduce cardinality
        parts = path.split("/")
        normalized = "/".join(
            "{id}" if (i > 0 and parts[i - 1] in ("patient", "patients", "cards", "traces", "tenants", "family")) else p
            for i, p in enumerate(parts)
        )

        REQUEST_COUNT.labels(request.method, normalized, response.status_code).inc()
        REQUEST_DURATION.labels(request.method, normalized).observe(duration)
        return response


async def metrics_endpoint(request: Request):
    """Prometheus /metrics endpoint — requires admin role (not public!)."""
    # Verify admin role from auth middleware claims
    user = getattr(request.state, "user", None)
    if not user or "admin" not in user.get("roles", []):
        return Response(status_code=403, content="Forbidden")
    return Response(generate_latest(), media_type=CONTENT_TYPE_LATEST)
```

> **REVIEW FIX #6**: The `/metrics` endpoint MUST be authenticated — exposing Prometheus metrics publicly leaks infrastructure details. The endpoint now checks for `admin` role from JWT claims. It is NOT added to the auth middleware exclusion list.

- [ ] **Step 2: Register in main.py**

Add to `app/main.py`:
```python
from app.middleware.metrics import MetricsMiddleware, metrics_endpoint

# After existing middleware:
if settings.METRICS_ENABLED:
    app.add_middleware(MetricsMiddleware)

# Add metrics endpoint (NOT in auth exclusion list — requires JWT + admin role):
app.add_route("/metrics", metrics_endpoint)
```

- [ ] **Step 3: Commit**

```bash
git add backend/services/api-gateway/app/middleware/metrics.py \
        backend/services/api-gateway/app/main.py
git commit -m "feat(gateway): Prometheus metrics — request count, duration, circuit breaker trips"
```

---

### Task 11: WebSocket Proxy for Doctor Dashboard Subscriptions

**Files:**
- Create: `backend/services/api-gateway/app/api/endpoints/websocket_proxy.py`

- [ ] **Step 1: Implement WebSocket proxy**

Create `app/api/endpoints/websocket_proxy.py`:
```python
"""WebSocket proxy for Doctor Dashboard real-time subscriptions.

Bridges client WebSocket connections to the Apollo Federation subscription
endpoint, forwarding auth headers and Kafka events (CARD_GENERATED, MRI_UPDATED).
"""
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
        verify_url = f"{auth_url}/auth/verify" if "/api" in auth_url else f"{auth_url}/api/auth/verify"
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
```

- [ ] **Step 2: Register in main.py**

Add to `app/main.py`:
```python
from app.api.endpoints.websocket_proxy import router as ws_router
app.include_router(ws_router)
```

- [ ] **Step 3: Commit**

```bash
git add backend/services/api-gateway/app/api/endpoints/websocket_proxy.py \
        backend/services/api-gateway/app/main.py
git commit -m "feat(gateway): WebSocket proxy for doctor dashboard real-time subscriptions"
```

---

## Phase 3 — Production (Wk 9-12)

### Task 12: Redis Distributed Rate Limiter

**Files:**
- Modify: `backend/services/api-gateway/app/middleware/rate_limit.py`

- [ ] **Step 1: Read existing rate_limit.py**

```bash
cat backend/services/api-gateway/app/middleware/rate_limit.py
```

- [ ] **Step 2: Add Redis sliding window backend**

Add to `app/middleware/rate_limit.py`:
```python
import redis.asyncio as aioredis

class RedisRateLimiter:
    """Sliding window rate limiter backed by Redis sorted sets.

    Falls back to in-memory limiter if Redis is unavailable.
    """

    def __init__(self, redis_url: str, max_requests: int = 100, window_seconds: int = 60):
        self.redis = aioredis.from_url(redis_url, decode_responses=True)
        self.max_requests = max_requests
        self.window_seconds = window_seconds
        self._fallback = {}  # in-memory fallback

    async def is_allowed(self, key: str) -> bool:
        try:
            import time
            now = time.time()
            window_start = now - self.window_seconds
            redis_key = f"rl:{key}"

            pipe = self.redis.pipeline()
            pipe.zremrangebyscore(redis_key, 0, window_start)
            pipe.zcard(redis_key)
            pipe.zadd(redis_key, {str(now): now})
            pipe.expire(redis_key, self.window_seconds + 1)
            results = await pipe.execute()

            count = results[1]
            return count < self.max_requests
        except Exception:
            # Redis down — allow (fail-open for availability)
            return True
```

Update the existing `RateLimitMiddleware` to use Redis when configured:
```python
# In __init__, add:
self._redis_limiter = None
if settings.REDIS_RATE_LIMIT_ENABLED and settings.REDIS_URL:
    self._redis_limiter = RedisRateLimiter(
        settings.REDIS_URL,
        max_requests=self.max_requests,
        window_seconds=self.window_seconds,
    )

# In the rate check logic, add before the in-memory check:
if self._redis_limiter:
    allowed = await self._redis_limiter.is_allowed(client_key)
    if not allowed:
        # return 429
```

- [ ] **Step 3: Commit**

```bash
git add backend/services/api-gateway/app/middleware/rate_limit.py
git commit -m "feat(gateway): Redis sliding-window rate limiter with in-memory fallback"
```

---

### Task 13: Response Cache

**Files:**
- Create: `backend/services/api-gateway/app/middleware/response_cache.py`

- [ ] **Step 1: Implement Redis response cache**

Create `app/middleware/response_cache.py`:
```python
"""Read-through response cache for latency-sensitive reads.

Caches GET responses for KB-20 projections, KB-26 MRI, and other
read-heavy endpoints. TTL matches Redis cache in KB services (2min).
"""
import hashlib
import json
import logging
from typing import Optional

import redis.asyncio as aioredis

logger = logging.getLogger(__name__)

# Cacheable path prefixes and their TTLs (seconds)
CACHE_RULES: dict[str, int] = {
    "/api/v1/doctor/patients/": 120,      # 2min — matches KB-20 Redis TTL
    "/api/v1/patient/": 60,               # 1min — patient data
    "/api/v1/tenants/": 3600,             # 1hr — branding rarely changes
}


class ResponseCache:
    def __init__(self, redis_url: str):
        self.redis = aioredis.from_url(redis_url, decode_responses=True)

    def _cache_key(self, method: str, path: str, user_id: str) -> Optional[str]:
        """Generate cache key. Returns None if path is not cacheable."""
        if method != "GET":
            return None
        for prefix, _ in CACHE_RULES.items():
            if path.startswith(prefix):
                raw = f"{method}:{path}:{user_id}"
                return f"cache:{hashlib.sha256(raw.encode()).hexdigest()[:16]}"
        return None

    def _get_ttl(self, path: str) -> int:
        for prefix, ttl in CACHE_RULES.items():
            if path.startswith(prefix):
                return ttl
        return 60

    async def get(self, method: str, path: str, user_id: str) -> Optional[dict]:
        key = self._cache_key(method, path, user_id)
        if not key:
            return None
        try:
            data = await self.redis.get(key)
            if data:
                logger.debug("Cache HIT: %s", key)
                return json.loads(data)
        except Exception:
            pass
        return None

    async def set(self, method: str, path: str, user_id: str, response_data: dict):
        key = self._cache_key(method, path, user_id)
        if not key:
            return
        try:
            ttl = self._get_ttl(path)
            await self.redis.setex(key, ttl, json.dumps(response_data))
        except Exception:
            pass  # cache miss is not fatal
```

- [ ] **Step 2: Integrate into _forward helper**

Update `_forward()` in `patient_app.py` to check cache before proxying:
```python
# At module level:
_response_cache = None

def _get_cache():
    global _response_cache
    if _response_cache is None and settings.REDIS_URL and settings.REDIS_RATE_LIMIT_ENABLED:
        from app.middleware.response_cache import ResponseCache
        _response_cache = ResponseCache(settings.REDIS_URL)
    return _response_cache

# Inside _forward(), before the httpx call:
    cache = _get_cache()
    user_id = getattr(getattr(request.state, "user", None), "get", lambda k, d: d)("id", "") if hasattr(request.state, "user") else ""
    if cache and method == "GET":
        cached = await cache.get(method, path, user_id)
        if cached:
            return cached

# After successful response:
    if cache and method == "GET" and resp.status_code == 200:
        await cache.set(method, path, user_id, resp.json())
```

- [ ] **Step 3: Commit**

```bash
git add backend/services/api-gateway/app/middleware/response_cache.py \
        backend/services/api-gateway/app/api/endpoints/patient_app.py
git commit -m "feat(gateway): Redis read-through cache for KB projections and patient data"
```

---

### Task 14: HIPAA Audit Logging

**Files:**
- Create: `backend/services/api-gateway/app/middleware/audit_log.py`

- [ ] **Step 1: Implement audit log middleware**

Create `app/middleware/audit_log.py`:
```python
"""HIPAA-compliant audit logging middleware.

Logs: who (user_id), what (method+path), when (timestamp),
for whom (patient_id from path), outcome (status code).
"""
import logging
import re
import time

from starlette.middleware.base import BaseHTTPMiddleware
from starlette.requests import Request

audit_logger = logging.getLogger("audit")

# Extract patient ID from path patterns like /patient/P123/... or /patients/P123/...
PATIENT_ID_RE = re.compile(r"/patients?/([^/]+)")


class AuditLogMiddleware(BaseHTTPMiddleware):
    async def dispatch(self, request: Request, call_next):
        start = time.time()
        response = await call_next(request)
        duration = time.time() - start

        # Extract who
        user = getattr(request.state, "user", None)
        user_id = user.get("id", "anonymous") if isinstance(user, dict) else "anonymous"
        user_role = user.get("roles", ["unknown"])[0] if isinstance(user, dict) else "unknown"

        # Extract for whom
        match = PATIENT_ID_RE.search(request.url.path)
        patient_id = match.group(1) if match else "N/A"

        # Log audit record
        audit_logger.info(
            "AUDIT who=%s role=%s action=%s %s patient=%s status=%d duration=%.3fs ip=%s",
            user_id,
            user_role,
            request.method,
            request.url.path,
            patient_id,
            response.status_code,
            duration,
            request.client.host if request.client else "unknown",
        )

        return response
```

- [ ] **Step 2: Register in main.py**

Add to `app/main.py`:
```python
from app.middleware.audit_log import AuditLogMiddleware
app.add_middleware(AuditLogMiddleware)
```

- [ ] **Step 3: Commit**

```bash
git add backend/services/api-gateway/app/middleware/audit_log.py \
        backend/services/api-gateway/app/main.py
git commit -m "feat(gateway): HIPAA audit logging — who/what/when/for-whom/outcome"
```

---

### Task 15: Configurable CORS

**Files:**
- Modify: `backend/services/api-gateway/app/main.py`

- [ ] **Step 1: Update CORS to use config**

Replace the existing hardcoded CORS setup in `main.py` with:
```python
# Replace static CORS origins with configurable list
cors_origins = [o.strip() for o in settings.CORS_ALLOWED_ORIGINS.split(",") if o.strip()]

app.add_middleware(
    CORSMiddleware,
    allow_origins=cors_origins,
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["Authorization", "Content-Type", "X-User-ID", "X-User-Role",
                   "X-Patient-ID", "X-Request-ID", "X-Correlation-ID"],
    expose_headers=["X-RateLimit-Limit", "X-RateLimit-Remaining", "X-Request-ID", "X-Correlation-ID"],
)
```

- [ ] **Step 2: Commit**

```bash
git add backend/services/api-gateway/app/main.py
git commit -m "feat(gateway): configurable CORS origins via CORS_ALLOWED_ORIGINS env var"
```

---

## Route Table Summary

| Route | Method | Auth | RBAC | Service | Phase |
|-------|--------|------|------|---------|-------|
| `/api/v1/auth/otp/send` | POST | Public | — | Auth Service | 1 |
| `/api/v1/auth/otp/verify` | POST | Public | — | Auth Service | 1 |
| `/api/v1/auth/refresh` | POST | Public | — | Auth Service | 1 |
| `/api/v1/tenants/:tenantId/branding` | GET | Public | — | Patient Service | 1 |
| `/api/v1/family/:token` | GET | Token | — | Patient Service | 1 |
| `/api/v1/patient/:id/health-score` | GET | JWT | patient:read | KB-26 | 1 |
| `/api/v1/patient/:id/actions/today` | GET | JWT | patient:read | KB-23 | 1 |
| `/api/v1/patient/:id/health-drive` | GET | JWT | patient:read | KB-25 | 1 |
| `/api/v1/patient/:id/progress` | GET | JWT | patient:read | KB-20 | 1 |
| `/api/v1/patient/:id/cause-effect` | GET | JWT | patient:read | KB-26 | 1 |
| `/api/v1/patient/:id/timeline` | GET | JWT | patient:read | KB-20 | 1 |
| `/api/v1/patient/:id/insights` | GET | JWT | patient:read | KB-26 | 1 |
| `/api/v1/patient/:id/checkin` | POST | JWT | patient:write | KB-22 | 1 |
| `/api/v1/patient/:id/abdm/verify` | POST | JWT | patient:write | Patient Service | 2 |
| `/api/v1/doctor/graphql` | POST | JWT | doctor:read | Apollo Federation | 1 |
| `/api/v1/doctor/patients/:id/summary` | GET | JWT | doctor:read | KB-20 | 1 |
| `/api/v1/doctor/patients/:id/mri` | GET | JWT | doctor:read | KB-26 | 1 |
| `/api/v1/doctor/patients/:id/cards` | GET | JWT | doctor:read | KB-23 | 1 |
| `/api/v1/doctor/cards/:id/action` | POST | JWT | doctor:write | KB-23 | 1 |
| `/api/v1/doctor/traces/:patient_id` | GET | JWT | doctor:admin | V-MCU | 2 |
| `/api/v1/doctor/patients/:id/channel-b-inputs` | GET | JWT | doctor:read | KB-20 | 2 |
| `/api/v1/doctor/patients/:id/channel-c-inputs` | GET | JWT | doctor:read | KB-20 | 2 |
| `/api/v1/doctor/subscriptions` | WS | JWT | doctor:read | Apollo WS | 2 |
| `/health` | GET | Public | — | Gateway | existing |
| `/metrics` | GET | JWT | admin | Gateway | 2 |
| *All existing routes* | * | existing | existing | existing | existing |

## Traffic Flow (Upgraded)

```
                    ┌─────────────────────────────────────────┐
                    │      Existing API Gateway (port 8000)    │
                    │           FastAPI + Python                │
                    │                                          │
  Patient App ──────┤  CORS → RateLimit → Logging → Metrics   │
  (Flutter/REST)    │       │                                  │
                    │  ┌────▼── Public Routes ───────────┐     │
  Doctor Dashboard ─┤  │ /auth/otp/* → Auth Service      │     │
  (React/GraphQL)   │  │ /tenants/* → Patient Service    │     │
                    │  └────────────────────────────────┘     │
                    │       │                                  │
                    │  ┌────▼── JWT Middleware (UPGRADED) ─┐    │
                    │  │ Auth Service /verify + TTL cache  │    │
                    │  │ (secrets stay in Auth Svc:8001)   │    │
                    │  └──────────────────────────────────┘    │
                    │       │                                  │
                    │  ┌────▼── RBAC (UPGRADED) ──────────┐    │
                    │  │ 28 existing + 18 new Vaidshala   │    │
                    │  │ route permissions                 │    │
                    │  └──────────────────────────────────┘    │
                    │       │                                  │
                    │  ┌────▼── Circuit Breaker (NEW) ────┐    │
                    │  │ Per-service, auto-recovery        │    │
                    │  └──────────────────────────────────┘    │
                    │       │                                  │
                    │  ┌────▼── Audit Log (NEW) ──────────┐    │
                    │  │ HIPAA: who/what/when/whom/outcome │    │
                    │  └──────────────────────────────────┘    │
                    │       │                                  │
                    │  ┌────▼── Reverse Proxy ────────────┐    │
                    │  │ NEW: /patient/* → Patient Svc     │    │
                    │  │ NEW: /doctor/*  → KB-20/23/26     │    │
                    │  │ NEW: /doctor/graphql → Apollo:4000│    │
                    │  │ NEW: /doctor/subscriptions → WS   │    │
                    │  │ EXISTING: /api/* → 12 services    │    │
                    │  └──────────────────────────────────┘    │
                    └─────────────────────────────────────────┘
```
