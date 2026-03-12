from fastapi import Request, HTTPException, status
from typing import Optional

def get_current_user(request: Request) -> dict:
    """
    Get current user information from request state.
    
    This function extracts user information that was added by the
    authentication middleware.
    
    Args:
        request: FastAPI request object
        
    Returns:
        dict: User information
        
    Raises:
        HTTPException: If user is not authenticated
    """
    if not hasattr(request.state, 'user') or not request.state.user:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Authentication required"
        )
    
    return request.state.user

def get_current_user_id(request: Request) -> str:
    """
    Get current user ID from request state.
    
    Args:
        request: FastAPI request object
        
    Returns:
        str: User ID
        
    Raises:
        HTTPException: If user is not authenticated
    """
    user = get_current_user(request)
    user_id = user.get('id') or user.get('sub')
    
    if not user_id:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="User ID not found in authentication token"
        )
    
    return user_id

def get_user_permissions(request: Request) -> list:
    """
    Get current user permissions from request state.
    
    Args:
        request: FastAPI request object
        
    Returns:
        list: User permissions
    """
    if hasattr(request.state, 'user_permissions'):
        return request.state.user_permissions or []
    return []

def get_user_roles(request: Request) -> list:
    """
    Get current user roles from request state.
    
    Args:
        request: FastAPI request object
        
    Returns:
        list: User roles
    """
    if hasattr(request.state, 'user_roles'):
        return request.state.user_roles or []
    return []

def get_user_role(request: Request) -> Optional[str]:
    """
    Get current user primary role from request state.
    
    Args:
        request: FastAPI request object
        
    Returns:
        str: User primary role
    """
    if hasattr(request.state, 'user_role'):
        return request.state.user_role
    return None
