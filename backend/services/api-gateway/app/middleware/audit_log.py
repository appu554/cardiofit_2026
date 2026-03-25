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
