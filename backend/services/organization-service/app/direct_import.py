"""
Direct import fallback for shared modules.
This is used when the shared module import fails.
"""

from fastapi import Request
from starlette.middleware.base import BaseHTTPMiddleware
import logging

logger = logging.getLogger(__name__)

class HeaderAuthMiddleware(BaseHTTPMiddleware):
    """
    Fallback authentication middleware that extracts user information from headers.
    This is a simplified version for when the shared module is not available.
    """
    
    def __init__(self, app, exclude_paths=None):
        super().__init__(app)
        self.exclude_paths = exclude_paths or []
    
    async def dispatch(self, request: Request, call_next):
        # Skip authentication for excluded paths
        if any(request.url.path.startswith(path) for path in self.exclude_paths):
            return await call_next(request)
        
        # Extract user information from headers (set by API Gateway)
        user_id = request.headers.get("X-User-ID")
        user_role = request.headers.get("X-User-Role")
        user_permissions = request.headers.get("X-User-Permissions", "").split(",")
        
        # Add user information to request state
        request.state.user_id = user_id
        request.state.user_role = user_role
        request.state.user_permissions = [p.strip() for p in user_permissions if p.strip()]
        
        return await call_next(request)
