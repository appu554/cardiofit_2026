from functools import wraps
from typing import List, Callable, Optional
from fastapi import HTTPException, status, Request
import logging

# Configure logging
logger = logging.getLogger(__name__)

def require_permissions(permissions: List[str], require_all: bool = False):
    """
    Decorator to require specific permissions for a route handler.
    
    Args:
        permissions: List of required permissions
        require_all: If True, all permissions are required; if False, any one is sufficient
        
    Returns:
        Decorator function
    """
    def decorator(func: Callable):
        @wraps(func)
        async def wrapper(*args, **kwargs):
            # Find the request object in args or kwargs
            request = None
            for arg in args:
                if isinstance(arg, Request):
                    request = arg
                    break
            
            if not request and 'request' in kwargs:
                request = kwargs['request']
                
            if not request:
                raise ValueError("Request object not found in function arguments")
                
            # Get the permissions from request state (optimized path)
            user_permissions = getattr(request.state, "user_permissions", [])
            
            # If user_permissions not directly available, get from user object
            if not user_permissions:
                user = getattr(request.state, "user", {})
                user_permissions = user.get("permissions", [])
            
            # Log the permissions check
            permission_str = ", ".join(permissions)
            logger.info(f"Checking permissions: {permission_str} - user has: {', '.join(user_permissions)}")
            
            # Admin users bypass permission checks
            if "admin" in getattr(request.state, "user_roles", []) or getattr(request.state, "user_role", "") == "admin":
                logger.info(f"User is admin, bypassing permission check")
                return await func(*args, **kwargs)
            
            # Check permissions
            if require_all:
                # All permissions are required
                if not all(perm in user_permissions for perm in permissions):
                    missing = [perm for perm in permissions if perm not in user_permissions]
                    logger.warning(f"Permission denied. User missing permissions: {', '.join(missing)}")
                    raise HTTPException(
                        status_code=status.HTTP_403_FORBIDDEN,
                        detail=f"Insufficient permissions. Missing: {', '.join(missing)}"
                    )
            else:
                # Any permission is sufficient
                if not any(perm in user_permissions for perm in permissions) and permissions:
                    logger.warning(f"Permission denied. User has none of the required permissions: {permission_str}")
                    raise HTTPException(
                        status_code=status.HTTP_403_FORBIDDEN,
                        detail=f"Insufficient permissions. Required: {permission_str}"
                    )
            
            logger.info(f"Permission check passed for: {permission_str}")
            # Call the original function
            return await func(*args, **kwargs)
        
        return wrapper
    
    return decorator

def require_role(roles: List[str]):
    """
    Decorator to require specific roles for a route handler.
    
    Args:
        roles: List of allowed roles
        
    Returns:
        Decorator function
    """
    def decorator(func: Callable):
        @wraps(func)
        async def wrapper(*args, **kwargs):
            # Find the request object in args or kwargs
            request = None
            for arg in args:
                if isinstance(arg, Request):
                    request = arg
                    break
            
            if not request and 'request' in kwargs:
                request = kwargs['request']
                
            if not request:
                raise ValueError("Request object not found in function arguments")
                
            # Get the primary role from optimized request state
            user_role = getattr(request.state, "user_role", "")
            
            # Get all roles from optimized request state
            user_roles = getattr(request.state, "user_roles", [])
            
            # If no optimized roles, get from user object
            if not user_roles and not user_role:
                user = getattr(request.state, "user", {})
                user_role = user.get("role", "")
                user_roles = user.get("roles", [])
            
            # Log the role check
            roles_str = ", ".join(roles)
            logger.info(f"Checking user roles: {user_role} and {user_roles} against required: {roles_str}")
            
            # First check if primary role meets requirements
            primary_role_authorized = user_role in roles
            
            # Then check if any assigned role meets requirements
            any_role_authorized = any(role in roles for role in user_roles)
            
            # Only grant access if at least one check passes
            if not primary_role_authorized and not any_role_authorized:
                logger.warning(f"Role authorization failed. User has roles: {user_role}, {user_roles}")
                allowed_roles = ", ".join(roles)
                raise HTTPException(
                    status_code=status.HTTP_403_FORBIDDEN,
                    detail=f"Role not authorized. Required roles: {allowed_roles}"
                )
            
            logger.info(f"Role authorization passed for roles: {roles_str}")   
            # Call the original function
            return await func(*args, **kwargs)
        
        return wrapper
    
    return decorator
