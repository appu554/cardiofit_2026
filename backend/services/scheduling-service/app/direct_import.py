"""
Direct import module for HeaderAuthMiddleware.
This is a fallback when the shared module import fails.
"""

import os
import sys
import logging
from typing import Optional
from fastapi import Request, HTTPException
from starlette.middleware.base import BaseHTTPMiddleware

logger = logging.getLogger(__name__)

class HeaderAuthMiddleware(BaseHTTPMiddleware):
    """
    Middleware for extracting user information from headers set by the API Gateway.
    
    This middleware extracts user information from headers that are set by the API Gateway
    after it validates the JWT token with the Auth Service.
    """

    def __init__(
        self,
        app,
        exclude_paths: Optional[list] = None
    ):
        super().__init__(app)
        self.exclude_paths = exclude_paths or ["/docs", "/openapi.json", "/redoc", "/health", "/api/federation"]
        logger.info(f"Initialized HeaderAuthMiddleware with exclude paths: {self.exclude_paths}")

    async def dispatch(self, request: Request, call_next):
        # Skip authentication for excluded paths
        if any(request.url.path.startswith(path) for path in self.exclude_paths):
            return await call_next(request)

        # Extract user information from headers
        user_id = request.headers.get("X-User-ID")
        user_role = request.headers.get("X-User-Role")
        user_roles = request.headers.get("X-User-Roles")
        user_permissions = request.headers.get("X-User-Permissions")

        # Add user information to request state
        request.state.user_id = user_id
        request.state.user_role = user_role
        request.state.user_roles = user_roles.split(",") if user_roles else []
        request.state.user_permissions = user_permissions.split(",") if user_permissions else []

        # Log the user information for debugging
        logger.debug(f"User ID: {user_id}, Role: {user_role}, Roles: {request.state.user_roles}")

        return await call_next(request)
