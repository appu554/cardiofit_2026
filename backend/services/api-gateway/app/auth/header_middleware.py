"""
Header-based Authentication Middleware for Microservices.

This middleware extracts user information from request headers set by the API Gateway.
It does NOT call the Auth Service directly, making it more efficient and reducing dependencies.
"""

import logging
from typing import Callable, Optional, List
from fastapi import Request, Response, status
from fastapi.responses import JSONResponse
from starlette.middleware.base import BaseHTTPMiddleware

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class HeaderAuthMiddleware(BaseHTTPMiddleware):
    """
    Middleware for extracting user information from request headers.

    This middleware extracts user information from headers set by the API Gateway,
    and adds the user information to the request state.

    It does NOT call the Auth Service directly, making it more efficient and reducing dependencies.
    """

    def __init__(
        self,
        app,
        exclude_paths: Optional[List[str]] = None
    ):
        super().__init__(app)
        self.exclude_paths = exclude_paths or ["/docs", "/openapi.json", "/redoc", "/health"]
        logger.info(f"Initialized HeaderAuthMiddleware with excluded paths: {self.exclude_paths}")

    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        """
        Process the request, extract user info from headers, and pass it to the next middleware.

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

        # Extract user information from headers
        user_id = request.headers.get("X-User-ID")
        user_role = request.headers.get("X-User-Role", "authenticated")
        user_roles_str = request.headers.get("X-User-Roles", "")
        user_permissions_str = request.headers.get("X-User-Permissions", "")

        # Parse roles and permissions from comma-separated strings
        user_roles = user_roles_str.split(",") if user_roles_str else []
        user_permissions = user_permissions_str.split(",") if user_permissions_str else []

        # If no user ID is provided, the request is not authenticated
        if not user_id:
            logger.warning(f"Authentication failed for {path}: No X-User-ID header")
            return JSONResponse(
                status_code=status.HTTP_401_UNAUTHORIZED,
                content={"detail": "Authentication required"},
                headers={"WWW-Authenticate": "Bearer"}
            )

        # Create user info object
        user_info = {
            "id": user_id,
            "role": user_role,
            "roles": user_roles,
            "permissions": user_permissions,
            "email": request.headers.get("X-User-Email", ""),
            "name": request.headers.get("X-User-Name", "")
        }

        # Add the user info to the request state
        request.state.user = user_info

        # Add specific RBAC fields to request state for easier access
        request.state.user_role = user_role
        request.state.user_roles = user_roles
        request.state.user_permissions = user_permissions

        # Log authentication success with detailed information
        logger.info(f"=== HEADER AUTH MIDDLEWARE ===")
        logger.info(f"Path: {request.url.path}")
        logger.info(f"Method: {request.method}")
        logger.info(f"User authenticated from headers: {user_id}")
        logger.info(f"User email: {request.headers.get('X-User-Email', 'N/A')}")
        logger.info(f"User role: {user_role}")
        logger.info(f"User roles: {user_roles}")
        logger.info(f"User permissions: {user_permissions}")

        # Log all headers for debugging
        logger.debug("All headers received:")
        for header_name, header_value in request.headers.items():
            if header_name.startswith('X-User') or header_name.startswith('Authorization'):
                logger.debug(f"  {header_name}: {header_value}")
        logger.info(f"=== END HEADER AUTH MIDDLEWARE ===")

        # Continue with the request
        return await call_next(request)
