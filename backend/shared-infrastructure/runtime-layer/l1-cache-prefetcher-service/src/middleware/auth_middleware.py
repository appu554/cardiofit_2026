"""
Authentication middleware for L1 Cache and Prefetcher Service
Provides JWT token validation and user context injection
"""

import jwt
from typing import Optional, Dict, Any
from datetime import datetime, timedelta
from fastapi import Request, HTTPException, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
import structlog

from ..utils.config import settings

logger = structlog.get_logger()

# Security scheme for FastAPI documentation
security = HTTPBearer()


class AuthMiddleware:
    """JWT Authentication middleware"""

    def __init__(self, app):
        self.app = app

    async def __call__(self, scope, receive, send):
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return

        request = Request(scope, receive)

        # Skip authentication for health checks and metrics
        path = request.url.path
        if path in ["/health", "/health/ready", "/metrics"]:
            await self.app(scope, receive, send)
            return

        # Skip authentication for OPTIONS requests
        if request.method == "OPTIONS":
            await self.app(scope, receive, send)
            return

        try:
            # Extract and validate JWT token
            user_context = await self._authenticate_request(request)

            # Add user context to request state
            scope["state"] = getattr(scope, "state", {})
            scope["state"]["user"] = user_context

            # Log successful authentication
            logger.debug(
                "request_authenticated",
                user_id=user_context.get("user_id"),
                path=path,
                method=request.method
            )

        except HTTPException as e:
            # Log authentication failure
            logger.warning(
                "authentication_failed",
                path=path,
                method=request.method,
                error=e.detail,
                status_code=e.status_code
            )

            # Return authentication error
            response = {
                "status_code": e.status_code,
                "headers": [(b"content-type", b"application/json")],
                "body": f'{{"error": "{e.detail}"}}'.encode()
            }

            await send({
                "type": "http.response.start",
                **response
            })
            await send({
                "type": "http.response.body",
                "body": response["body"]
            })
            return

        await self.app(scope, receive, send)

    async def _authenticate_request(self, request: Request) -> Dict[str, Any]:
        """Extract and validate JWT token from request"""

        # Get Authorization header
        authorization = request.headers.get("Authorization")
        if not authorization:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Missing Authorization header"
            )

        # Validate Bearer format
        if not authorization.startswith("Bearer "):
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Invalid authorization header format"
            )

        # Extract token
        token = authorization.replace("Bearer ", "")
        if not token:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Missing JWT token"
            )

        try:
            # Decode and validate JWT
            payload = jwt.decode(
                token,
                settings.JWT_SECRET_KEY,
                algorithms=[settings.JWT_ALGORITHM]
            )

            # Validate required claims
            required_claims = ["user_id", "exp"]
            for claim in required_claims:
                if claim not in payload:
                    raise HTTPException(
                        status_code=status.HTTP_401_UNAUTHORIZED,
                        detail=f"Missing required claim: {claim}"
                    )

            # Check token expiration
            exp_timestamp = payload["exp"]
            if datetime.utcnow().timestamp() > exp_timestamp:
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Token has expired"
                )

            # Extract user context
            user_context = {
                "user_id": payload["user_id"],
                "email": payload.get("email"),
                "role": payload.get("role", "user"),
                "permissions": payload.get("permissions", []),
                "organization_id": payload.get("organization_id"),
                "session_id": payload.get("session_id"),
                "token_issued_at": payload.get("iat"),
                "token_expires_at": exp_timestamp
            }

            return user_context

        except jwt.ExpiredSignatureError:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Token has expired"
            )
        except jwt.InvalidTokenError as e:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail=f"Invalid token: {str(e)}"
            )
        except Exception as e:
            logger.error("token_validation_error", error=str(e))
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Token validation failed"
            )


def get_current_user(request: Request) -> Dict[str, Any]:
    """Extract current user context from request state"""
    user_context = getattr(request.state, "user", None)
    if not user_context:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="User context not found"
        )
    return user_context