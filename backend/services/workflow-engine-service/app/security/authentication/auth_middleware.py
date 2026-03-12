"""
Authentication middleware for Workflow Engine Service.
"""
import logging
from typing import List, Optional
from fastapi import Request, HTTPException
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.responses import Response

logger = logging.getLogger(__name__)


class AuthenticationMiddleware(BaseHTTPMiddleware):
    """
    Simple authentication middleware for the Workflow Engine Service.
    """
    
    def __init__(self, app, auth_service_url: str = None, exclude_paths: List[str] = None):
        super().__init__(app)
        self.auth_service_url = auth_service_url
        self.exclude_paths = exclude_paths or [
            "/docs", "/openapi.json", "/redoc", "/health", "/api/federation", "/"
        ]
        logger.info(f"Authentication middleware initialized with exclude paths: {self.exclude_paths}")
    
    async def dispatch(self, request: Request, call_next):
        """
        Process the request and add authentication context.
        """
        # Skip authentication for excluded paths
        if any(request.url.path.startswith(path) for path in self.exclude_paths):
            return await call_next(request)
        
        # For now, we'll add basic user context without strict authentication
        # This can be enhanced later with proper JWT validation
        try:
            # Extract user information from headers (if provided by API Gateway)
            user_id = request.headers.get("X-User-ID")
            user_role = request.headers.get("X-User-Role")
            user_roles = request.headers.get("X-User-Roles", "").split(",") if request.headers.get("X-User-Roles") else []
            user_permissions = request.headers.get("X-User-Permissions", "").split(",") if request.headers.get("X-User-Permissions") else []
            
            # Add user context to request state
            request.state.user_id = user_id
            request.state.user_role = user_role
            request.state.user_roles = [role.strip() for role in user_roles if role.strip()]
            request.state.user_permissions = [perm.strip() for perm in user_permissions if perm.strip()]
            
            # Log authentication context (for debugging)
            if user_id:
                logger.debug(f"Request authenticated for user: {user_id} with role: {user_role}")
            else:
                logger.debug("Request without user authentication headers")
            
            response = await call_next(request)
            return response
            
        except Exception as e:
            logger.error(f"Authentication middleware error: {e}")
            # Don't fail the request due to auth middleware errors
            return await call_next(request)


def get_current_user(request: Request) -> Optional[dict]:
    """
    Get current user information from request state.
    
    Args:
        request: FastAPI request object
        
    Returns:
        User information dictionary or None
    """
    try:
        user_id = getattr(request.state, 'user_id', None)
        if not user_id:
            return None
        
        return {
            "id": user_id,
            "role": getattr(request.state, 'user_role', None),
            "roles": getattr(request.state, 'user_roles', []),
            "permissions": getattr(request.state, 'user_permissions', [])
        }
    except Exception as e:
        logger.error(f"Error getting current user: {e}")
        return None


def require_permission(permission: str):
    """
    Decorator to require specific permission for an endpoint.
    
    Args:
        permission: Required permission string
    """
    def decorator(func):
        async def wrapper(request: Request, *args, **kwargs):
            user = get_current_user(request)
            if not user:
                raise HTTPException(status_code=401, detail="Authentication required")
            
            if permission not in user.get("permissions", []):
                raise HTTPException(status_code=403, detail=f"Permission '{permission}' required")
            
            return await func(request, *args, **kwargs)
        return wrapper
    return decorator


def require_role(role: str):
    """
    Decorator to require specific role for an endpoint.
    
    Args:
        role: Required role string
    """
    def decorator(func):
        async def wrapper(request: Request, *args, **kwargs):
            user = get_current_user(request)
            if not user:
                raise HTTPException(status_code=401, detail="Authentication required")
            
            if role not in user.get("roles", []) and user.get("role") != role:
                raise HTTPException(status_code=403, detail=f"Role '{role}' required")
            
            return await func(request, *args, **kwargs)
        return wrapper
    return decorator
