"""
Permission checking middleware for the FHIR service.
"""
from fastapi import Request, HTTPException, status
from fastapi.responses import JSONResponse
from starlette.middleware.base import BaseHTTPMiddleware
from typing import Dict, List, Optional, Callable
import logging
import re

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class PermissionMiddleware(BaseHTTPMiddleware):
    """
    Middleware for checking permissions for specific endpoints.

    This middleware intercepts requests to specific endpoints and checks if the user
    has the required permissions before allowing access.
    """

    def __init__(
        self,
        app,
        endpoint_permissions: Dict[str, List[str]] = None,
        debug: bool = False
    ):
        super().__init__(app)
        self.endpoint_permissions = endpoint_permissions or {}
        self.debug = debug
        logger.info(f"Initialized PermissionMiddleware with endpoint permissions: {self.endpoint_permissions}")

    async def dispatch(self, request: Request, call_next: Callable):
        """
        Process the request, check permissions if needed, and pass it to the next middleware.

        Args:
            request: The incoming request
            call_next: The next middleware to call

        Returns:
            The response from the next middleware
        """
        # Get the path
        path = request.url.path

        # Check if this path requires permission checking
        required_permissions = None
        for pattern, permissions in self.endpoint_permissions.items():
            if re.match(pattern, path):
                required_permissions = permissions
                logger.info(f"Found matching pattern {pattern} for path {path}")
                break

        # If no permissions are required, continue with the request
        if not required_permissions:
            logger.info(f"No permissions required for path: {path}")
            return await call_next(request)

        # Log that we're checking permissions for this path
        logger.info(f"Checking permissions for path: {path}")
        logger.info(f"Required permissions: {required_permissions}")

        # Get the user permissions from the request state
        user_permissions = getattr(request.state, "user_permissions", [])
        user_roles = getattr(request.state, "user_roles", [])

        # If user_permissions is empty, try to get them from user_info
        if not user_permissions and hasattr(request.state, "user"):
            user_info = request.state.user
            if isinstance(user_info, dict):
                user_permissions = user_info.get("permissions", [])
                if not user_roles:
                    user_roles = user_info.get("roles", [])

        # Log the user permissions
        logger.info(f"User permissions: {user_permissions}")
        logger.info(f"User roles: {user_roles}")

        # Admin users bypass permission checks
        if "admin" in user_roles or "doctor" in user_roles:
            logger.info(f"User has admin/doctor role, bypassing permission check")
            return await call_next(request)

        # Also check if the user has the doctor role in their role field
        if hasattr(request.state, "user_role") and request.state.user_role in ["admin", "doctor"]:
            logger.info(f"User has admin/doctor primary role, bypassing permission check")
            return await call_next(request)

        # Special case for patient:read permission
        if 'patient:read' in required_permissions:
            # Check if the user has any permission that implies patient:read
            implied_permissions = ['patient:read', 'patient:write', 'admin:all']
            if any(perm in user_permissions for perm in implied_permissions):
                logger.info(f"User has permission that implies patient:read: {[p for p in implied_permissions if p in user_permissions]}")
                return await call_next(request)

        # Check if the user has any of the required permissions
        if not any(perm in user_permissions for perm in required_permissions):
            logger.warning(f"Permission denied. User has none of the required permissions: {required_permissions}")

            # Always enforce permissions, even in debug mode
            logger.warning("Permission check failed, returning 403 Forbidden")
            return JSONResponse(
                status_code=status.HTTP_403_FORBIDDEN,
                content={"detail": f"Insufficient permissions. Required: {', '.join(required_permissions)}"}
            )

        # User has the required permissions, continue with the request
        logger.info(f"Permission check passed for: {path}")
        return await call_next(request)
