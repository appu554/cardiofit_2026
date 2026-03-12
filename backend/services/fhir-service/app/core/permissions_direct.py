"""
Direct permission checking utilities for the FHIR service.
"""
from fastapi import HTTPException, status, Request
from typing import List, Dict, Any, Optional
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def check_permissions(token_payload: Dict[str, Any], required_permissions: List[str]) -> bool:
    """
    Check if the user has the required permissions.
    
    Args:
        token_payload: The token payload from the request
        required_permissions: List of required permissions
        
    Returns:
        True if the user has the required permissions, False otherwise
    """
    # Extract permissions from token_payload
    permissions = []
    roles = []
    
    # Log the token payload
    logger.info(f"Token payload keys: {list(token_payload.keys())}")
    
    # Try to get permissions from app_metadata
    if 'app_metadata' in token_payload:
        app_metadata = token_payload.get('app_metadata', {})
        permissions = app_metadata.get('permissions', [])
        roles = app_metadata.get('roles', [])
        logger.info(f"Found permissions in app_metadata: {permissions}")
        logger.info(f"Found roles in app_metadata: {roles}")
    
    # Try to get permissions from user_roles
    elif 'user_roles' in token_payload:
        user_roles = token_payload.get('user_roles', {})
        permissions = user_roles.get('permissions', [])
        roles = user_roles.get('roles', [])
        logger.info(f"Found permissions in user_roles: {permissions}")
        logger.info(f"Found roles in user_roles: {roles}")
    
    # Try to get permissions directly from token
    elif 'permissions' in token_payload:
        permissions = token_payload.get('permissions', [])
        logger.info(f"Found permissions directly in token: {permissions}")
    
    # Try to get roles directly from token
    if 'roles' in token_payload and not roles:
        roles = token_payload.get('roles', [])
        logger.info(f"Found roles directly in token: {roles}")
    
    # Check if user is admin or doctor
    if 'doctor' in roles or 'admin' in roles:
        logger.info(f"User has admin/doctor role, bypassing permission check")
        return True
    
    # Check if user has the required permissions
    for perm in required_permissions:
        if perm in permissions:
            logger.info(f"User has required permission: {perm}")
            return True
    
    # User doesn't have the required permissions
    logger.warning(f"Permission denied. User has none of the required permissions: {required_permissions}")
    return False

def require_permissions(token_payload: Dict[str, Any], required_permissions: List[str]):
    """
    Require specific permissions for a route handler.
    
    Args:
        token_payload: The token payload from the request
        required_permissions: List of required permissions
        
    Raises:
        HTTPException: If the user doesn't have the required permissions
    """
    if not check_permissions(token_payload, required_permissions):
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail=f"Insufficient permissions. Required: {', '.join(required_permissions)}"
        )
