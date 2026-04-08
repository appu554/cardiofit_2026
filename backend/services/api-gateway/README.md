# API Gateway — Clinical Synthesis Hub

Centralized reverse-proxy gateway for the CardioFit platform. Handles authentication (via Auth Service), RBAC, rate limiting, circuit breakers, audit logging, and request routing to 17+ downstream microservices.

**Port**: 8000 | **Framework**: Python 3.11 + FastAPI | **Auth**: Auth Service (port 8001) — no local JWT secrets

## Architecture

```
  Flutter App ──┐                           ┌── Auth Service (8001)
                │   ┌───────────────────┐   ├── KB-20 Patient Profile (8131)
  React Dashboard──►│  API Gateway:8000 │──►├── KB-22 HPI Engine (8132)
                │   │                   │   ├── KB-23 Decision Cards (8134)
  External API ─┘   │  CORS → Rate Limit│   ├── KB-25 Lifestyle Graph (8136)
                    │  → Auth (cached)  │   ├── KB-26 Metabolic Twin (8137)
                    │  → RBAC → Audit   │   ├── V-MCU Engine (embedded)
                    │  → Circuit Breaker│   ├── Apollo Federation (4000)
                    │  → Proxy          │   ├── Patient Service (8003)
                    └───────────────────┘   └── 8 more services...
```

## Quick Start

```bash
# Install
cd backend/services/api-gateway
pip install -r requirements.txt

# Run (requires Auth Service on port 8001)
uvicorn app.main:app --host 0.0.0.0 --port 8000 --reload

# Docker
docker build -t api-gateway .
docker run -p 8000:8000 api-gateway
```

**Swagger UI**: http://localhost:8000/docs
**OpenAPI JSON**: http://localhost:8000/openapi.json
**GraphQL Explorer**: http://localhost:8000/graphql-explorer

## Authentication Flow

```
1. Client sends POST /api/v1/auth/otp/send  →  Auth Service issues OTP
2. Client sends POST /api/v1/auth/otp/verify →  Auth Service returns JWT
3. Client includes "Authorization: Bearer <jwt>" on all subsequent requests
4. Gateway calls Auth Service /api/auth/verify (cached 60s) → sets request.state.user
5. RBAC middleware checks role + permissions against route table
6. Request proxied to downstream service with X-User-* headers
```

No JWT secrets exist in the gateway — Auth Service (port 8001) is the single source of truth.

## Route Table

### Public Routes (No Auth)

| Route | Method | Service | Description |
|-------|--------|---------|-------------|
| `/api/v1/auth/otp/send` | POST | Auth Service | Send OTP to phone |
| `/api/v1/auth/otp/verify` | POST | Auth Service | Verify OTP, get JWT |
| `/api/v1/auth/refresh` | POST | Auth Service | Refresh access token |
| `/api/v1/tenants/{id}/branding` | GET | Patient Service | Multi-tenant branding |
| `/api/v1/family/{token}` | GET | Patient Service | Family view (token-scoped) |
| `/health` | GET | Gateway | Health check |
| `/docs` | GET | Gateway | Swagger UI |
| `/graphql` | GET/POST | Strawberry | GraphQL endpoint |

### Patient App Routes (JWT + `patient` role)

| Route | Method | RBAC | Service | Description |
|-------|--------|------|---------|-------------|
| `/api/v1/patient/{id}/health-score` | GET | patient:read | KB-26 (8137) | Composite health score |
| `/api/v1/patient/{id}/actions/today` | GET | patient:read | KB-23 (8134) | Today's action items |
| `/api/v1/patient/{id}/health-drive` | GET | patient:read | KB-25 (8136) | Lifestyle health drive |
| `/api/v1/patient/{id}/progress` | GET | patient:read | KB-20 (8131) | Protocol progress |
| `/api/v1/patient/{id}/cause-effect` | GET | patient:read | KB-26 (8137) | Cause-effect analysis |
| `/api/v1/patient/{id}/timeline` | GET | patient:read | KB-20 (8131) | Clinical timeline |
| `/api/v1/patient/{id}/insights` | GET | patient:read | KB-26 (8137) | Patient-facing insights |
| `/api/v1/patient/{id}/checkin` | POST | patient:write | KB-22 (8132) | Daily check-in |
| `/api/v1/patient/{id}/abdm/verify` | POST | patient:write | Patient Svc (8003) | ABDM verification |

### Doctor Dashboard Routes (JWT + `physician`/`doctor`/`nurse`/`admin` role)

| Route | Method | RBAC | Service | Description |
|-------|--------|------|---------|-------------|
| `/api/v1/doctor/graphql` | POST | doctor:read | Apollo (4000) | GraphQL queries |
| `/api/v1/doctor/patients/{id}/summary` | GET | doctor:read | KB-20 (8131) | Patient summary |
| `/api/v1/doctor/patients/{id}/mri` | GET | doctor:read | KB-26 (8137) | Metabolic Risk Index |
| `/api/v1/doctor/patients/{id}/cards` | GET | doctor:read | KB-23 (8134) | Decision cards |
| `/api/v1/doctor/cards/{id}/action` | POST | doctor:write | KB-23 (8134) | Card action |
| `/api/v1/doctor/traces/{id}` | GET | doctor:admin | V-MCU | Safety traces |
| `/api/v1/doctor/patients/{id}/channel-b-inputs` | GET | doctor:read | KB-20 (8131) | Channel B inputs |
| `/api/v1/doctor/patients/{id}/channel-c-inputs` | GET | doctor:read | KB-20 (8131) | Channel C inputs |
| `/api/v1/doctor/subscriptions` | WebSocket | doctor:read | Apollo WS | Real-time events |

### Admin Routes

| Route | Method | RBAC | Service | Description |
|-------|--------|------|---------|-------------|
| `/metrics` | GET | admin | Gateway | Prometheus metrics |

### Legacy Proxy Routes (catch-all)

All existing routes (`/api/auth/*`, `/api/patients/*`, `/api/fhir/*`, `/api/observations/*`, etc.) continue to work via the catch-all proxy. See `app/api/proxy.py` for the full SERVICE_ROUTES list.

## Configuration

### Required

| Variable | Description | Default |
|----------|-------------|---------|
| `AUTH_SERVICE_URL` | Auth Service (JWT validator) | `http://localhost:8001` |

### Vaidshala KB Services

| Variable | Service | Default |
|----------|---------|---------|
| `KB20_SERVICE_URL` | KB-20 Patient Profile | `http://localhost:8131` |
| `KB22_SERVICE_URL` | KB-22 HPI Engine | `http://localhost:8132` |
| `KB23_SERVICE_URL` | KB-23 Decision Cards | `http://localhost:8134` |
| `KB25_SERVICE_URL` | KB-25 Lifestyle Knowledge Graph | `http://localhost:8136` |
| `KB26_SERVICE_URL` | KB-26 Metabolic Digital Twin | `http://localhost:8137` |
| `VMCU_SERVICE_URL` | V-MCU Engine (optional) | _(empty)_ |
| `APOLLO_FEDERATION_URL` | Apollo GraphQL Federation | `http://localhost:4000/graphql` |

### Other Services

| Variable | Default |
|----------|---------|
| `PATIENT_SERVICE_URL` | `http://localhost:8003` |
| `FHIR_SERVICE_URL` | `http://localhost:8014` |
| `OBSERVATION_SERVICE_URL` | `http://localhost:8008` |
| `MEDICATION_SERVICE_URL` | `http://localhost:8009` |
| `CONDITION_SERVICE_URL` | `http://localhost:8010` |
| `ENCOUNTER_SERVICE_URL` | `http://localhost:8020` |
| `TIMELINE_SERVICE_URL` | `http://localhost:8012` |
| `INGESTION_SERVICE_URL` | `http://localhost:8140` |
| `INTAKE_SERVICE_URL` | `http://localhost:8141` |

### Production Settings

| Variable | Description | Default |
|----------|-------------|---------|
| `AUTH_CACHE_TTL_SECONDS` | Auth response cache TTL | `60` |
| `AUTH_CACHE_MAX_SIZE` | Max cached tokens (LRU) | `10000` |
| `CORS_ALLOWED_ORIGINS` | Comma-separated origins | `http://localhost:3000,...` |
| `REDIS_URL` | Redis for rate limit + cache | `redis://localhost:6380` |
| `REDIS_RATE_LIMIT_ENABLED` | Enable Redis rate limiter | `False` |
| `RATE_LIMIT_ENABLED` | Enable in-memory rate limiter | `False` |
| `RATE_LIMIT_REQUESTS` | Requests per window | `100` |
| `RATE_LIMIT_WINDOW` | Window size (seconds) | `60` |
| `CIRCUIT_BREAKER_FAIL_MAX` | Failures before trip | `5` |
| `CIRCUIT_BREAKER_RESET_TIMEOUT` | Reset timeout (seconds) | `30` |
| `METRICS_ENABLED` | Enable Prometheus metrics | `False` |
| `ENABLE_REQUEST_LOGGING` | Log requests | `True` |

## Middleware Stack

Middleware executes outermost-first (order matters):

```
1. CORS                    — Configurable origins, credentials, headers
2. Audit Log               — HIPAA: who/what/when/for-whom/outcome/IP
3. Rate Limit (optional)   — Redis sliding window or in-memory fallback
4. Request Logging         — Method, path, status, duration
5. Authentication          — Auth Service /verify + 60s LRU cache
6. RBAC                    — 46+ route permission patterns
7. GraphQL RBAC            — Field-level GraphQL permissions
8. Metrics (optional)      — Prometheus counters + histograms
```

## Testing

```bash
# Unit tests
cd backend/services/api-gateway
python -m pytest tests/ -v

# E2E tests (requires Auth Service + KB services running)
python -m pytest tests/test_e2e_gateway.py -v

# Individual test files
python -m pytest tests/test_auth_cache.py -v        # Auth cache (5 tests)
python -m pytest tests/test_circuit_breaker.py -v    # Circuit breaker (3 tests)
python -m pytest tests/test_patient_routes.py -v     # Patient routes (2 tests)
python -m pytest tests/test_doctor_routes.py -v      # Doctor routes (2 tests)
```

## Development

### Adding a New Service

1. Add service URL to `app/config.py`
2. Add SERVICE_ROUTE entry in `app/api/proxy.py`
3. Add RBAC permissions in `app/middleware/rbac.py` (both `ROUTE_PERMISSIONS` and `ROLE_ROUTE_RESTRICTIONS`)
4. Add auth exclusion path in `app/main.py` if public

### Adding a New Endpoint (Patient/Doctor)

1. Add route handler in `app/api/endpoints/patient_app.py` or `doctor_dashboard.py`
2. Use `_forward(request, settings.SERVICE_URL, "/downstream/path")` for proxying
3. Add RBAC entry in `app/middleware/rbac.py`
4. Register router in `app/main.py` BEFORE `proxy_router`

### Project Structure

```
app/
├── main.py                        # App init, middleware stack, router registration
├── config.py                      # Settings (env vars)
├── auth/
│   ├── auth_cache.py              # LRU+TTL cache for Auth Service /verify
│   ├── middleware.py              # JWT validation via Auth Service
│   ├── header_middleware.py       # Header-based auth passthrough
│   └── decorators.py             # Auth decorators
├── api/
│   ├── proxy.py                   # Catch-all reverse proxy (17 services)
│   └── endpoints/
│       ├── patient_app.py         # Patient App REST routes
│       ├── doctor_dashboard.py    # Doctor Dashboard REST routes
│       ├── websocket_proxy.py     # WebSocket subscription proxy
│       ├── raw_fhir.py            # Direct FHIR operations
│       ├── raw_graphql.py         # Direct GraphQL proxy
│       └── direct_fhir.py         # FHIR without validation
├── middleware/
│   ├── rbac.py                    # RBAC with 46+ route patterns
│   ├── rate_limit.py              # Redis + in-memory rate limiting
│   ├── circuit_breaker.py         # Per-service circuit breaker
│   ├── metrics.py                 # Prometheus metrics
│   ├── response_cache.py          # Redis read-through cache
│   ├── audit_log.py               # HIPAA audit logging
│   ├── logging.py                 # Request/response logging
│   └── graphql_rbac.py            # GraphQL-specific RBAC
├── graphql/                       # Strawberry GraphQL schema
└── shared/transformers/           # FHIR transformers
tests/
├── test_auth_cache.py
├── test_circuit_breaker.py
├── test_patient_routes.py
├── test_doctor_routes.py
└── test_e2e_gateway.py            # E2E integration tests
```
