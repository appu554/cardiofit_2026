from fastapi import FastAPI, Request, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from contextlib import asynccontextmanager
import logging
import os

from app.config import settings
from app.auth import AuthenticationMiddleware, HeaderAuthMiddleware
from app.middleware import RBACMiddleware, RequestLoggingMiddleware, RateLimitMiddleware
from app.middleware.metrics import MetricsMiddleware, metrics_endpoint
from app.middleware.audit_log import AuditLogMiddleware
from app.api.proxy import router as proxy_router
from app.api.endpoints.patient_app import router as patient_router, auth_router as otp_router
from app.api.endpoints.doctor_dashboard import router as doctor_router
from app.api.endpoints.websocket_proxy import router as ws_router

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)

@asynccontextmanager
async def lifespan(app: FastAPI):
    logger.info("Starting API Gateway (REST proxy mode)")
    logger.info(f"Auth Service URL: {settings.AUTH_SERVICE_URL}")
    yield
    logger.info("Shutting down API Gateway")

# Initialize FastAPI app
app = FastAPI(
    title=settings.PROJECT_NAME,
    description="Clinical Synthesis Hub API Gateway — REST proxy to Auth, KB, and clinical services.",
    version="2.1.0",
    lifespan=lifespan,
    openapi_tags=[
        {"name": "health", "description": "Gateway health and discovery"},
        {"name": "auth", "description": "Public authentication (OTP, token refresh)"},
        {"name": "patient-app", "description": "Patient App routes (Flutter) — JWT + patient role"},
        {"name": "doctor-dashboard", "description": "Doctor Dashboard routes (React) — JWT + physician/doctor role"},
        {"name": "websocket", "description": "WebSocket subscriptions for real-time events"},
        {"name": "admin", "description": "Admin-only endpoints (metrics, management)"},
    ],
)

# Configurable CORS origins — use CORS_ALLOWED_ORIGINS env var
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

# HIPAA audit logging — outermost middleware after CORS
app.add_middleware(AuditLogMiddleware)

# Rate limiting
if settings.RATE_LIMIT_ENABLED:
    app.add_middleware(
        RateLimitMiddleware,
        requests_limit=settings.RATE_LIMIT_REQUESTS,
        window_size=settings.RATE_LIMIT_WINDOW,
        exclude_paths=["/docs", "/openapi.json", "/redoc", "/health"]
    )

# Request logging
if settings.ENABLE_REQUEST_LOGGING:
    app.add_middleware(
        RequestLoggingMiddleware,
        log_request_body=settings.LOG_REQUEST_BODY,
        log_response_body=settings.LOG_RESPONSE_BODY
    )

# Authentication middleware
app.add_middleware(
    AuthenticationMiddleware,
    auth_service_url=settings.AUTH_SERVICE_URL,
    exclude_paths=[
        "/docs", "/openapi.json", "/redoc", "/health", "/favicon.ico",
        "/api/auth/login", "/api/auth/token", "/api/auth/authorize",
        "/api/auth/callback", "/api/auth/verify",
        "/api/v1/auth/otp/send", "/api/v1/auth/otp/verify",
        "/api/v1/auth/refresh", "/api/v1/tenants", "/api/v1/family",
    ]
)

# RBAC middleware
app.add_middleware(
    RBACMiddleware,
    exclude_paths=[
        "/docs", "/openapi.json", "/redoc", "/health", "/favicon.ico",
        "/api/auth/login", "/api/auth/token", "/api/auth/authorize",
        "/api/auth/callback", "/api/auth/verify",
        "/api/v1/auth/otp/send", "/api/v1/auth/otp/verify",
        "/api/v1/auth/refresh", "/api/v1/tenants", "/api/v1/family",
    ]
)

# Prometheus metrics
if settings.METRICS_ENABLED:
    app.add_middleware(MetricsMiddleware)

# ── Endpoints ──────────────────────────────────────────────

@app.get("/health")
async def health_check():
    return {"status": "ok", "mode": "rest-proxy"}

app.add_route("/metrics", metrics_endpoint)

@app.get("/")
async def root():
    return {
        "message": "Clinical Synthesis Hub API Gateway",
        "documentation": "/docs",
        "health": "/health",
    }

@app.get("/favicon.ico", include_in_schema=False)
async def favicon():
    return {"status": "ok"}

# ── REST Routers ───────────────────────────────────────────
# Specific routes MUST come BEFORE the catch-all proxy

app.include_router(otp_router)       # /api/v1/auth/* (public)
app.include_router(patient_router)   # /api/v1/patient/* (JWT required)
app.include_router(doctor_router)    # /api/v1/doctor/* (JWT required)
app.include_router(ws_router)        # /api/v1/doctor/subscriptions (WebSocket)

# Catch-all proxy — last
app.include_router(proxy_router)
