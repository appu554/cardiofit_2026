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
