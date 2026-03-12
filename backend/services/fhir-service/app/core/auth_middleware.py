"""
Authentication middleware for the FHIR service.
"""
from fastapi import Request, status
from fastapi.responses import JSONResponse
from starlette.middleware.base import BaseHTTPMiddleware
from typing import Dict, List, Optional, Callable, Any
import logging
import httpx
import os

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Default Auth Service URL
DEFAULT_AUTH_SERVICE_URL = "http://localhost:8001/api"

class AuthenticationMiddleware(BaseHTTPMiddleware):
    """
    Middleware for authenticating requests using the Auth Service.

    This middleware extracts the JWT token from the Authorization header,
    validates it with the Auth Service, and adds the user information to the request state.
    """

    def __init__(
        self,
        app,
        auth_service_url: Optional[str] = None,
        exclude_paths: Optional[list] = None
    ):
        super().__init__(app)
        self.auth_service_url = auth_service_url or os.getenv("AUTH_SERVICE_URL", DEFAULT_AUTH_SERVICE_URL)
        self.exclude_paths = exclude_paths or ["/docs", "/openapi.json", "/redoc", "/health"]
        logger.info(f"Initialized AuthenticationMiddleware with Auth Service URL: {self.auth_service_url}")

    async def dispatch(self, request: Request, call_next: Callable) -> Any:
        """
        Process the request, authenticate if needed, and pass it to the next middleware.

        Args:
            request: The incoming request
            call_next: The next middleware to call

        Returns:
            The response from the next middleware
        """
        # Skip authentication for excluded paths
        path = request.url.path
        if any(path.startswith(excluded) for excluded in self.exclude_paths):
            return await call_next(request)

        # Skip authentication for OPTIONS requests (CORS preflight)
        if request.method == "OPTIONS":
            return await call_next(request)

        # Get the Authorization header
        auth_header = request.headers.get("Authorization")
        if not auth_header:
            logger.warning(f"No Authorization header found for path: {path}")
            return JSONResponse(
                status_code=status.HTTP_401_UNAUTHORIZED,
                content={"detail": "Authorization header missing"},
                headers={"WWW-Authenticate": "Bearer"}
            )

        # Extract the token
        parts = auth_header.split()
        if len(parts) != 2 or parts[0].lower() != "bearer":
            logger.warning(f"Invalid Authorization header format: {auth_header}")
            return JSONResponse(
                status_code=status.HTTP_401_UNAUTHORIZED,
                content={"detail": "Invalid Authorization header format"},
                headers={"WWW-Authenticate": "Bearer"}
            )

        token = parts[1]

        # Validate the token with the Auth Service
        try:
            async with httpx.AsyncClient() as client:
                # Construct the verify URL with the correct path
                verify_url = f"{self.auth_service_url}/api/auth/verify"
                logger.info(f"Calling Auth Service verify endpoint: {verify_url}")

                response = await client.post(
                    verify_url,
                    headers={"Authorization": f"Bearer {token}"}
                )

                if response.status_code != 200:
                    logger.error(f"Token validation failed: {response.status_code} - {response.text}")
                    return JSONResponse(
                        status_code=status.HTTP_401_UNAUTHORIZED,
                        content={"detail": "Invalid authentication credentials"},
                        headers={"WWW-Authenticate": "Bearer"}
                    )

                result = response.json()

                if not result.get("valid", False):
                    logger.error(f"Token validation failed: {result.get('error', 'Unknown error')}")
                    return JSONResponse(
                        status_code=status.HTTP_401_UNAUTHORIZED,
                        content={"detail": result.get("error", "Invalid token")},
                        headers={"WWW-Authenticate": "Bearer"}
                    )

                # Get the user info from the response
                user_info = result.get("user", {})
                raw_payload = result.get("raw_payload", {})

                # Extract RBAC information
                roles = user_info.get("roles", [])
                role = user_info.get("role", "authenticated")
                permissions = user_info.get("permissions", [])

                # Also try to get permissions directly from the raw payload
                if not permissions and raw_payload:
                    # Try to get from app_metadata
                    app_metadata = raw_payload.get("app_metadata", {})
                    if app_metadata:
                        permissions = app_metadata.get("permissions", [])
                        roles = app_metadata.get("roles", roles)

                    # Try to get from user_roles
                    user_roles = raw_payload.get("user_roles", {})
                    if user_roles and not permissions:
                        permissions = user_roles.get("permissions", [])
                        if not roles:
                            roles = user_roles.get("roles", [])

                # Add the user info to the request state
                request.state.user = user_info

                # Add specific RBAC fields to request state for easier access
                request.state.user_role = role
                request.state.user_roles = roles
                request.state.user_permissions = permissions

                # Log the user info and permissions
                logger.info(f"User info: {user_info}")
                logger.info(f"Setting user_permissions in request state: {permissions}")
                logger.info(f"Setting user_roles in request state: {roles}")

                # Add the token to the request state
                request.state.token = token

                # Log RBAC information
                logger.info(f"User authenticated: {user_info.get('id')} with role: {role}")
                logger.info(f"User roles: {roles}")
                logger.info(f"User permissions: {permissions}")

                # Continue with the request
                return await call_next(request)

        except Exception as e:
            logger.error(f"Error validating token: {str(e)}")
            return JSONResponse(
                status_code=status.HTTP_401_UNAUTHORIZED,
                content={"detail": "Error validating token"},
                headers={"WWW-Authenticate": "Bearer"}
            )
