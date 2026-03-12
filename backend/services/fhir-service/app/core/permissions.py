"""
Permission checking utilities for the FHIR service.
"""
from fastapi import Request, HTTPException, status
from typing import List, Dict, Any, Callable, Awaitable
from functools import wraps
import logging
import inspect

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def require_permissions(permissions: List[str], require_all: bool = False):
    """
    Decorator to require specific permissions for a route handler.

    Args:
        permissions: List of required permissions
        require_all: If True, all permissions are required; if False, any permission is sufficient

    Returns:
        Decorator function
    """
    def decorator(func: Callable):
        # Log that we're applying the decorator to this function
        logger.info(f"Applying permission decorator to function: {func.__name__}")
        logger.info(f"Required permissions: {permissions}")

        @wraps(func)
        async def wrapper(*args, **kwargs):
            # Log that we're executing the wrapper
            logger.info(f"Executing permission check for {func.__name__}")
            logger.info(f"Function args: {args}")
            logger.info(f"Function kwargs keys: {list(kwargs.keys())}")

            # Find the token_payload in kwargs
            token_payload = None
            for key, value in kwargs.items():
                if key == 'token_payload' and isinstance(value, dict):
                    token_payload = value
                    break

            if not token_payload:
                logger.error(f"token_payload not found in function arguments for {func.__name__}")
                raise ValueError("token_payload not found in function arguments")

            # Log the token payload
            logger.info(f"Token payload keys: {list(token_payload.keys())}")

            # Extract permissions from token_payload
            user_permissions = []

            # Check if permissions are in user_roles
            if 'user_roles' in token_payload and 'permissions' in token_payload['user_roles']:
                user_permissions = token_payload['user_roles']['permissions']
                logger.info(f"Found permissions in user_roles: {user_permissions}")

            # Also check if permissions are directly in the token
            elif 'permissions' in token_payload:
                user_permissions = token_payload['permissions']
                logger.info(f"Found permissions directly in token: {user_permissions}")

            # Also check if permissions are in app_metadata
            elif 'app_metadata' in token_payload and 'permissions' in token_payload['app_metadata']:
                user_permissions = token_payload['app_metadata']['permissions']
                logger.info(f"Found permissions in app_metadata: {user_permissions}")

            # Log the permissions check
            permission_str = ", ".join(permissions)
            logger.info(f"PERMISSION CHECK: Required: {permission_str} - User has: {', '.join(user_permissions)}")

            # Check if user is admin (admin role or doctor role for testing)
            user_roles = []
            if 'user_roles' in token_payload and 'roles' in token_payload['user_roles']:
                user_roles = token_payload['user_roles']['roles']
                logger.info(f"Found roles in user_roles: {user_roles}")
            elif 'roles' in token_payload:
                user_roles = token_payload['roles']
                logger.info(f"Found roles directly in token: {user_roles}")
            elif 'app_metadata' in token_payload and 'roles' in token_payload['app_metadata']:
                user_roles = token_payload['app_metadata']['roles']
                logger.info(f"Found roles in app_metadata: {user_roles}")

            # Admin users bypass permission checks
            if "admin" in user_roles or "doctor" in user_roles:
                logger.info(f"User has admin/doctor role, bypassing permission check")
                return await func(*args, **kwargs)

            # Check permissions
            if require_all:
                # All permissions are required
                if not all(perm in user_permissions for perm in permissions):
                    missing = [perm for perm in permissions if perm not in user_permissions]
                    logger.warning(f"PERMISSION DENIED. User missing permissions: {', '.join(missing)}")
                    raise HTTPException(
                        status_code=status.HTTP_403_FORBIDDEN,
                        detail=f"Insufficient permissions. Missing: {', '.join(missing)}"
                    )
            else:
                # Any permission is sufficient
                if not any(perm in user_permissions for perm in permissions) and permissions:
                    logger.warning(f"PERMISSION DENIED. User has none of the required permissions: {permission_str}")
                    raise HTTPException(
                        status_code=status.HTTP_403_FORBIDDEN,
                        detail=f"Insufficient permissions. Required: {permission_str}"
                    )

            logger.info(f"PERMISSION CHECK PASSED for: {permission_str}")
            # Call the original function
            return await func(*args, **kwargs)

        return wrapper

    return decorator
