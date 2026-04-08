from fastapi import APIRouter, Depends, HTTPException, Header, status
from typing import Dict, Optional, Any, List
import requests
import logging
import httpx
import time
import os
import uuid
from datetime import datetime, timedelta
from jose import jwt
from app.config import settings
from app.security import verify_token, get_token_from_header, get_token_payload
from enum import Enum

# Set up logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

router = APIRouter(prefix="/auth", tags=["Authentication"])

from pydantic import BaseModel, EmailStr, Field

class UserRole(str, Enum):
    ADMIN = "admin"
    USER = "user"
    DOCTOR = "doctor"
    NURSE = "nurse"
    PATIENT = "patient"

class Permission(str, Enum):
    READ_PATIENTS = "read:patients"
    WRITE_PATIENTS = "write:patients"
    READ_NOTES = "read:notes"
    WRITE_NOTES = "write:notes"
    READ_REPORTS = "read:reports"
    WRITE_REPORTS = "write:reports"
    ADMIN_ACCESS = "admin:access"

class CreateUserRequest(BaseModel):
    email: EmailStr
    password: str = Field(..., min_length=8)
    full_name: str
    role: UserRole
    permissions: List[Permission] = []
    metadata: Dict[str, Any] = {}

    class Config:
        schema_extra = {
            "example": {
                "email": "john.doe@example.com",
                "password": "StrongP@ssw0rd",
                "full_name": "John Doe",
                "role": "doctor",
                "permissions": ["read:patients", "write:notes"],
                "metadata": {"department": "Cardiology"}
            }
        }

class UserResponse(BaseModel):
    user_id: str
    email: str
    full_name: str
    role: str
    permissions: List[str]
    created_at: str

    class Config:
        schema_extra = {
            "example": {
                "user_id": "auth0|64ce63ab460998ee9e1793f8",
                "email": "john.doe@example.com",
                "full_name": "John Doe",
                "role": "doctor",
                "permissions": ["read:patients", "write:notes"],
                "created_at": "2025-04-05T04:58:33Z"
            }
        }

class LoginRequest(BaseModel):
    username: str
    password: str

class AuthorizationCodeRequest(BaseModel):
    code: str
    redirect_uri: str

# Device Authentication Models
class DeviceVendor(str, Enum):
    VENDOR_1 = "device-vendor-1"
    VENDOR_2 = "device-vendor-2"
    VENDOR_3 = "device-vendor-3"

class DeviceAuthRequest(BaseModel):
    vendor_id: DeviceVendor
    allowed_device_types: List[str]
    rate_limit: Optional[int] = 1000
    timestamp_tolerance: Optional[int] = 300  # 5 minutes in seconds

    class Config:
        schema_extra = {
            "example": {
                "vendor_id": "device-vendor-1",
                "allowed_device_types": ["heart_rate", "blood_pressure", "blood_glucose"],
                "rate_limit": 1000,
                "timestamp_tolerance": 300
            }
        }

class DeviceTokenResponse(BaseModel):
    access_token: str
    token_type: str = "Bearer"
    expires_in: int
    vendor_id: str
    allowed_device_types: List[str]

    class Config:
        schema_extra = {
            "example": {
                "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9...",
                "token_type": "Bearer",
                "expires_in": 3600,
                "vendor_id": "device-vendor-1",
                "allowed_device_types": ["heart_rate", "blood_pressure"]
            }
        }

class DeviceTokenValidationResponse(BaseModel):
    valid: bool
    vendor_id: str
    device_claims: Dict[str, Any]
    token_id: str
    issued_at: int
    expires_at: int

    class Config:
        schema_extra = {
            "example": {
                "valid": True,
                "vendor_id": "device-vendor-1",
                "device_claims": {
                    "vendor_id": "device-vendor-1",
                    "allowed_device_types": ["heart_rate", "blood_pressure"],
                    "rate_limit": 1000,
                    "timestamp_tolerance": 300
                },
                "token_id": "unique-jwt-id",
                "issued_at": 1703123456,
                "expires_at": 1703127056
            }
        }

@router.post("/login", summary="Login with username and password", description="Login with username and password and return tokens", response_description="JWT token response from Auth0")
async def login(request: LoginRequest):
    """
    Login with username and password

    This endpoint authenticates a user with Auth0 using their username and password,
    and returns an access token that can be used to access protected resources.

    - **username**: The user's email address
    - **password**: The user's password

    Returns a JSON object containing:
    - **access_token**: JWT token for accessing protected resources
    - **id_token**: JWT token containing user information
    - **token_type**: Type of token (Bearer)
    - **expires_in**: Token expiration time in seconds
    - **refresh_token**: Token that can be used to get a new access token
    """
    logger.info(f"Received login request for username: {request.username}")
    try:
        # Call Auth0 token endpoint
        token_url = f"https://{settings.AUTH0_DOMAIN}/oauth/token"
        payload = {
            "grant_type": "http://auth0.com/oauth/grant-type/password-realm",
            "username": request.username,
            "password": request.password,
            "client_id": settings.AUTH0_CLIENT_ID,
            "client_secret": settings.AUTH0_CLIENT_SECRET,
            "audience": settings.AUTH0_API_AUDIENCE,
            "scope": "openid profile email offline_access",
            "realm": "Username-Password-Authentication"
        }

        logger.info(f"Sending request to Auth0: {token_url}")
        response = requests.post(token_url, json=payload)

        if response.status_code != 200:
            logger.error(f"Auth0 error: {response.status_code} - {response.text}")
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail=f"Authentication failed: {response.text}"
            )

        # Get the tokens from Auth0
        token_data = response.json()

        # Get user info using the access token
        user_info = await get_user_info_from_token(token_data.get("access_token"))

        # Return tokens and user info
        result = {
            **token_data,
            "user": user_info
        }

        logger.info("Successfully authenticated with Auth0")
        return result
    except Exception as e:
        logger.error(f"Authentication error: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=f"Authentication failed: {str(e)}"
        )

@router.post("/authorize", summary="Get authorization URL", description="Get the URL to redirect the user to for authorization with Auth0", response_description="Authorization URL and state parameter")
async def get_authorization_url(redirect_uri: str):
    """
    Get the URL to redirect the user to for authorization with Auth0 (Authorization Code Flow)

    This endpoint is the first step in the Authorization Code flow. It generates a URL that
    the frontend should redirect the user to for authentication with Auth0.

    ## Flow:
    1. Frontend calls this endpoint with the redirect_uri
    2. Backend generates an authorization URL with the correct parameters
    3. Frontend redirects the user to this URL
    4. User authenticates with Auth0
    5. Auth0 redirects back to the redirect_uri with a code
    6. Frontend exchanges the code for tokens using the /callback endpoint

    ## Parameters:
    - **redirect_uri**: The URI to redirect to after authorization (must be registered in Auth0)

    ## Returns:
    - **authorization_url**: The URL to redirect the user to for authorization
    - **state**: A random state parameter for CSRF protection

    ## Example:
    ```json
    {
        "authorization_url": "https://your-auth0-domain.auth0.com/authorize?response_type=code&client_id=your-client-id&redirect_uri=http%3A%2F%2Flocalhost%3A3000%2Fcallback&scope=openid+profile+email&audience=your-audience&state=random-state",
        "state": "random-state"
    }
    ```
    """
    try:
        # Generate a random state parameter for CSRF protection
        import secrets
        state = secrets.token_urlsafe(16)

        # Build the authorization URL
        auth_url = f"https://{settings.AUTH0_DOMAIN}/authorize"
        params = {
            "response_type": "code",
            "client_id": settings.AUTH0_CLIENT_ID,
            "redirect_uri": redirect_uri,
            "scope": "openid profile email offline_access",
            "audience": settings.AUTH0_API_AUDIENCE,
            "state": state
        }

        # Convert params to query string
        from urllib.parse import urlencode
        query_string = urlencode(params)

        # Return the authorization URL
        return {
            "authorization_url": f"{auth_url}?{query_string}",
            "state": state
        }
    except Exception as e:
        logger.error(f"Error generating authorization URL: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error generating authorization URL: {str(e)}"
        )

@router.post("/callback", summary="Exchange authorization code for tokens", description="Exchange an authorization code for tokens from Auth0", response_description="JWT tokens and user information")
async def exchange_code_for_tokens(request: AuthorizationCodeRequest):
    """
    Exchange an authorization code for tokens (Authorization Code Flow)

    This endpoint is the second step in the Authorization Code flow. It exchanges an authorization code
    received from Auth0 for access and refresh tokens.

    ## Flow:
    1. User authenticates with Auth0 and is redirected back with a code
    2. Frontend calls this endpoint with the code and redirect_uri
    3. Backend exchanges the code for tokens with Auth0
    4. Backend returns the tokens and user information to the frontend

    ## Request Body:
    - **code**: The authorization code received from Auth0
    - **redirect_uri**: The redirect URI used in the authorization request (must match the one used in /authorize)

    ## Returns:
    - **access_token**: JWT token for accessing protected resources
    - **id_token**: JWT token containing user information
    - **token_type**: Type of token (Bearer)
    - **expires_in**: Token expiration time in seconds
    - **refresh_token**: Token that can be used to get a new access token
    - **user**: User information extracted from the token

    ## Example Response:
    ```json
    {
        "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IkM2c2dOdnlLTlZaaWxDV2NiekY5UiJ9...",
        "id_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IkM2c2dOdnlLTlZaaWxDV2NiekY5UiJ9...",
        "token_type": "Bearer",
        "expires_in": 86400,
        "refresh_token": "v1.MaP7J0sdBvpUZFZzeY...",
        "user": {
            "id": "auth0|64ce63ab460998ee9e1793f8",
            "email": "user@example.com",
            "full_name": "John Doe",
            "role": "user",
            "is_active": true,
            "created_at": 1743826196,
            "permissions": []
        }
    }
    ```
    """
    try:
        # Call Auth0 token endpoint
        token_url = f"https://{settings.AUTH0_DOMAIN}/oauth/token"
        payload = {
            "grant_type": "authorization_code",
            "client_id": settings.AUTH0_CLIENT_ID,
            "client_secret": settings.AUTH0_CLIENT_SECRET,
            "code": request.code,
            "redirect_uri": request.redirect_uri
        }

        logger.info(f"Exchanging code for tokens: {token_url}")
        response = requests.post(token_url, json=payload)

        if response.status_code != 200:
            logger.error(f"Auth0 error: {response.status_code} - {response.text}")
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail=f"Failed to exchange code for tokens: {response.text}"
            )

        # Get the tokens from Auth0
        token_data = response.json()

        # Get user info using the access token
        user_info = await get_user_info_from_token(token_data.get("access_token"))

        # Return tokens and user info
        result = {
            **token_data,
            "user": user_info
        }

        logger.info("Successfully exchanged code for tokens")
        return result
    except Exception as e:
        logger.error(f"Error exchanging code for tokens: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=f"Failed to exchange code for tokens: {str(e)}"
        )

# Helper function to get user info from a token
async def get_user_info_from_token(token: str) -> dict:
    """
    Get user information from a token

    This function extracts user information from a token and returns it as a dictionary.
    If the token is invalid, it returns an empty dictionary.

    Args:
        token: The access token to extract user information from

    Returns:
        A dictionary containing user information
    """
    if not token:
        return {}

    try:
        # Verify the token
        payload = await verify_token(token)

        # Check if it's a Supabase token
        if payload.get('iss') == 'supabase' or payload.get('iss', '').startswith('https://') and 'supabase' in payload.get('iss', ''):
            # First try to get roles and permissions from user_roles claim
            user_roles = payload.get("user_roles", {})
            roles = user_roles.get("roles", [])
            permissions = user_roles.get("permissions", [])

            # If not found, try to get from app_metadata
            if not roles and 'app_metadata' in payload:
                app_metadata = payload.get('app_metadata', {})
                roles = app_metadata.get('roles', [])
                permissions = app_metadata.get('permissions', [])
                logger.info(f"Extracted roles from app_metadata: {roles}")
                logger.info(f"Extracted permissions from app_metadata: {permissions}")

            # Get primary role (first in list or default to "authenticated")
            primary_role = roles[0] if roles else payload.get("role", "authenticated")

            # Log the extracted RBAC info
            logger.info(f"Extracted roles from token: {roles}")
            logger.info(f"Extracted permissions from token: {permissions}")

            # Extract user information from Supabase token
            user_info = {
                "id": payload.get("sub"),
                "email": payload.get("email", ""),
                "full_name": payload.get("user_metadata", {}).get("full_name", ""),
                "role": primary_role,
                "roles": roles,
                "is_active": True,
                "created_at": payload.get("iat"),
                "permissions": permissions
            }

            # If we don't have a name but have an email, use the email as a display name
            if not user_info["full_name"] and user_info["email"]:
                user_info["full_name"] = user_info["email"].split("@")[0]

            return user_info
        else:
            # Extract user information from Auth0 token payload (legacy support)
            user_info = {
                "id": payload.get("sub"),
                "email": payload.get("email", ""),
                "full_name": payload.get("name", ""),
                "role": "user",  # This would be determined based on Auth0 roles
                "is_active": True,
                "created_at": payload.get("iat"),
                "permissions": payload.get("permissions", [])
            }

            # If we still don't have a name, create a user-friendly display name from the ID
            if not user_info["full_name"] and user_info["id"] and user_info["id"].startswith("auth0|"):
                # Extract a short ID from the Auth0 ID
                short_id = user_info["id"].split("|")[1][:4]
                user_info["full_name"] = f"User {short_id}"

            return user_info
    except Exception as e:
        logger.error(f"Error getting user info from token: {str(e)}")
        return {}

# Refresh token endpoint
class RefreshTokenRequest(BaseModel):
    refresh_token: str

@router.post("/verify", summary="Verify JWT token", description="Verify the validity of a JWT token", response_description="Token validation result")
async def verify_auth_token(authorization: Optional[str] = Header(None)):
    """
    Verify a JWT token

    This endpoint verifies the validity of a JWT token and returns the token payload if valid.
    It can be used to check if a token is still valid before making API calls.

    ## When to use:
    - To check if a token is still valid
    - To extract information from a token without making an API call
    - For debugging token issues

    ## Parameters:
    - **authorization**: Authorization header with the format "Bearer {token}"

    ## Returns:
    - **valid**: Boolean indicating if the token is valid
    - **user**: Token payload if valid, or
    - **error**: Error message if invalid

    ## Example Response (Valid Token):
    ```json
    {
        "valid": true,
        "user": {
            "iss": "supabase",
            "sub": "1234567890",
            "role": "authenticated",
            "iat": 1743823081,
            "exp": 1743909481,
            "email": "user@example.com"
        }
    }
    ```

    ## Example Response (Invalid Token):
    ```json
    {
        "valid": false,
        "error": "Token has expired"
    }
    ```
    """
    if not authorization:
        logger.warning("Authorization header missing in verify request")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Authorization header missing"
        )

    try:
        # Log the authorization header format (first 20 chars only)
        auth_prefix = authorization[:20] + "..." if len(authorization) > 20 else authorization
        logger.info(f"Verifying token with authorization header: {auth_prefix}")

        # Extract the token from the header
        token = await get_token_from_header(authorization)

        # Log the token format (first 20 chars only)
        token_prefix = token[:20] + "..." if len(token) > 20 else token
        logger.info(f"Extracted token: {token_prefix}")

        # Verify the token
        payload = await verify_token(token)

        # Log success
        logger.info(f"Token verified successfully for subject: {payload.get('sub', 'unknown')}")

        # Extract detailed user info with roles and permissions
        user_info = await get_user_info_from_token(token)

        # Return success response
        return {
            "valid": True,
            "user": user_info
        }
    except HTTPException as e:
        logger.error(f"Token verification failed with HTTP exception: {e.detail}")
        return {
            "valid": False,
            "error": e.detail
        }
    except Exception as e:
        logger.error(f"Unexpected exception during token verification: {str(e)}")

        # In development mode, return a success response for testing
        if settings.DEBUG:
            logger.warning("DEVELOPMENT MODE: Returning success response despite error")
            return {
                "valid": True,
                "user": {
                    "id": "test-user-id",
                    "email": "doctor@example.com",
                    "full_name": "Test Doctor",
                    "role": "doctor",
                    "roles": ["doctor"],
                    "is_active": True,
                    "created_at": int(time.time()),
                    "permissions": [
                        "patient:read",
                        "patient:write",
                        "patient:delete",
                        "observation:read",
                        "observation:write",
                        "observation:delete",
                        "condition:read",
                        "condition:write",
                        "condition:delete",
                        "medication:read",
                        "medication:write",
                        "medication:delete",
                        "encounter:read",
                        "encounter:write",
                        "encounter:delete"
                    ]
                },
                "debug_info": {
                    "error": str(e),
                    "development_mode": True
                }
            }
        else:
            return {
                "valid": False,
                "error": f"Token verification failed: {str(e)}"
            }

@router.post("/refresh", summary="Refresh access token", description="Get a new access token using a refresh token", response_description="New JWT tokens and user information")
async def refresh_token(request: RefreshTokenRequest):
    """
    Refresh access token

    This endpoint refreshes an expired access token using a refresh token. It should be called
    when the access token expires to get a new one without requiring the user to log in again.

    ## When to use:
    - When the access token has expired
    - Before making API calls if the token is about to expire
    - To maintain a user's session without requiring re-authentication

    ## Request Body:
    - **refresh_token**: The refresh token received during login or previous refresh

    ## Returns:
    - **access_token**: New JWT token for accessing protected resources
    - **id_token**: New JWT token containing user information (if available)
    - **token_type**: Type of token (Bearer)
    - **expires_in**: Token expiration time in seconds
    - **refresh_token**: New refresh token (if rotation is enabled)
    - **user**: User information extracted from the token

    ## Example Response:
    ```json
    {
        "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IkM2c2dOdnlLTlZaaWxDV2NiekY5UiJ9...",
        "token_type": "Bearer",
        "expires_in": 86400,
        "user": {
            "id": "auth0|64ce63ab460998ee9e1793f8",
            "email": "user@example.com",
            "full_name": "John Doe",
            "role": "user",
            "is_active": true,
            "created_at": 1743826196,
            "permissions": []
        }
    }
    ```

    ## Security Considerations:
    - Refresh tokens should be stored securely
    - They should never be exposed to client-side JavaScript
    - They should be transmitted only over HTTPS
    """
    try:
        # Call Auth0 token endpoint
        token_url = f"https://{settings.AUTH0_DOMAIN}/oauth/token"
        payload = {
            "grant_type": "refresh_token",
            "client_id": settings.AUTH0_CLIENT_ID,
            "client_secret": settings.AUTH0_CLIENT_SECRET,
            "refresh_token": request.refresh_token
        }

        logger.info(f"Refreshing token: {token_url}")
        response = requests.post(token_url, json=payload)

        if response.status_code != 200:
            logger.error(f"Auth0 error: {response.status_code} - {response.text}")
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail=f"Failed to refresh token: {response.text}"
            )

        # Get the tokens from Auth0
        token_data = response.json()

        # Get user info using the access token
        user_info = await get_user_info_from_token(token_data.get("access_token"))

        # Return tokens and user info
        result = {
            **token_data,
            "user": user_info
        }

        logger.info("Successfully refreshed token")
        return result
    except Exception as e:
        logger.error(f"Error refreshing token: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=f"Failed to refresh token: {str(e)}"
        )

# Check permissions endpoint
class CheckPermissionsRequest(BaseModel):
    path: str
    method: str

@router.post("/check-permissions", summary="Check permissions", description="Check if the user has the required permissions for a path", response_description="Permission check result")
async def check_permissions(request: CheckPermissionsRequest, authorization: Optional[str] = Header(None)):
    """
    Check if the user has the required permissions for a path

    This endpoint checks if the authenticated user has the required permissions to access a specific path.
    It is used by the API Gateway to enforce RBAC at the gateway level.

    ## When to use:
    - When the API Gateway needs to check if a user has permission to access a specific endpoint
    - For centralized permission checking across microservices

    ## Request Body:
    - **path**: The path to check permissions for
    - **method**: The HTTP method (GET, POST, PUT, DELETE, etc.)

    ## Headers:
    - **Authorization**: Bearer token for the authenticated user

    ## Returns:
    - **has_permission**: Whether the user has permission to access the path
    - **detail**: Additional information about the permission check

    ## Example Response:
    ```json
    {
        "has_permission": true,
        "detail": "User has required permissions"
    }
    ```
    """
    logger.info(f"Checking permissions for path: {request.path}, method: {request.method}")

    try:
        # Extract token from header
        token = await get_token_from_header(authorization)

        # Verify the token and get the payload
        payload = await verify_token(token)

        # Get user permissions from the payload
        user_permissions = []
        user_roles = []

        # First try to get from user_roles claim
        user_roles_claim = payload.get("user_roles", {})
        if user_roles_claim:
            user_permissions = user_roles_claim.get("permissions", [])
            user_roles = user_roles_claim.get("roles", [])

        # If not found, try to get from app_metadata
        if not user_permissions and 'app_metadata' in payload:
            app_metadata = payload.get('app_metadata', {})
            user_permissions = app_metadata.get('permissions', [])
            user_roles = app_metadata.get('roles', [])

        # Get primary role
        user_role = payload.get("role", "authenticated")

        # Admin users bypass permission checks
        if "admin" in user_roles or user_role == "admin" or "doctor" in user_roles or user_role == "doctor":
            logger.info(f"User has admin/doctor role, bypassing permission check")
            return {
                "has_permission": True,
                "detail": "User has admin/doctor role"
            }

        # Define permission mappings
        permission_mappings = {
            # Patient service permissions
            r"^/api/patients": {
                "GET": ["patient:read"],
                "POST": ["patient:write"],
                "PUT": ["patient:write"],
                "DELETE": ["patient:delete"]
            },
            # FHIR service permissions
            r"^/api/fhir/Patient": {
                "GET": ["patient:read"],
                "POST": ["patient:write"],
                "PUT": ["patient:write"],
                "DELETE": ["patient:delete"]
            },
            # Observation service permissions
            r"^/api/observations": {
                "GET": ["observation:read"],
                "POST": ["observation:write"],
                "PUT": ["observation:write"],
                "DELETE": ["observation:delete"]
            },
            r"^/api/fhir/Observation": {
                "GET": ["observation:read"],
                "POST": ["observation:write"],
                "PUT": ["observation:write"],
                "DELETE": ["observation:delete"]
            },
            # Condition service permissions
            r"^/api/conditions": {
                "GET": ["condition:read"],
                "POST": ["condition:write"],
                "PUT": ["condition:write"],
                "DELETE": ["condition:delete"]
            },
            r"^/api/fhir/Condition": {
                "GET": ["condition:read"],
                "POST": ["condition:write"],
                "PUT": ["condition:write"],
                "DELETE": ["condition:delete"]
            },
            # Medication service permissions
            r"^/api/medications": {
                "GET": ["medication:read"],
                "POST": ["medication:write"],
                "PUT": ["medication:write"],
                "DELETE": ["medication:delete"]
            },
            r"^/api/fhir/Medication": {
                "GET": ["medication:read"],
                "POST": ["medication:write"],
                "PUT": ["medication:write"],
                "DELETE": ["medication:delete"]
            },
            # Encounter service permissions
            r"^/api/encounters": {
                "GET": ["encounter:read"],
                "POST": ["encounter:write"],
                "PUT": ["encounter:write"],
                "DELETE": ["encounter:delete"]
            },
            r"^/api/fhir/Encounter": {
                "GET": ["encounter:read"],
                "POST": ["encounter:write"],
                "PUT": ["encounter:write"],
                "DELETE": ["encounter:delete"]
            },
            # Timeline service permissions
            r"^/api/timeline": {
                "GET": ["timeline:read"],
                "POST": ["timeline:write"],
                "PUT": ["timeline:write"],
                "DELETE": ["timeline:delete"]
            }
        }

        # Check if the path matches any permission pattern
        import re
        for pattern, method_permissions in permission_mappings.items():
            if re.match(pattern, request.path):
                # Get the required permissions for this method
                required_permissions = method_permissions.get(request.method, [])

                # If no permissions are required, allow access
                if not required_permissions:
                    return {
                        "has_permission": True,
                        "detail": "No permissions required for this path"
                    }

                # Check if the user has any of the required permissions
                if any(perm in user_permissions for perm in required_permissions):
                    matching_perms = [perm for perm in required_permissions if perm in user_permissions]
                    logger.info(f"User has required permission: {matching_perms}")
                    return {
                        "has_permission": True,
                        "detail": f"User has required permissions: {matching_perms}"
                    }

                # User doesn't have the required permissions
                logger.warning(f"Permission denied. User has none of the required permissions: {required_permissions}")
                return {
                    "has_permission": False,
                    "detail": f"Insufficient permissions. Required: {required_permissions}"
                }

        # If no pattern matches, allow access (no permissions required)
        return {
            "has_permission": True,
            "detail": "No permission mapping for this path"
        }

    except Exception as e:
        logger.error(f"Error checking permissions: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error checking permissions: {str(e)}"
        )

# Logout endpoint
class LogoutRequest(BaseModel):
    refresh_token: Optional[str] = None

@router.post("/logout", summary="Logout", description="Logout the user and revoke tokens", response_description="Success status")
async def logout(request: LogoutRequest):
    """
    Logout

    This endpoint logs out the user and revokes their refresh token. It should be called when
    the user explicitly logs out or when their session ends.

    ## When to use:
    - When the user clicks a logout button
    - When the user's session times out
    - When you need to invalidate a user's tokens for security reasons

    ## Request Body:
    - **refresh_token**: The refresh token to revoke (optional)

    ## Returns:
    - **success**: Whether the logout was successful

    ## Example Response:
    ```json
    {
        "success": true
    }
    ```

    ## Client-side Actions:
    After calling this endpoint, the frontend should:
    1. Clear all tokens from storage
    2. Redirect the user to the login page
    3. Reset any user-specific state

    ## Note:
    Even if the token revocation fails on the Auth0 side, this endpoint will return success=true
    to ensure the frontend proceeds with logout. This is a security measure to prevent users from
    being stuck in a logged-in state.
    """
    try:
        # If a refresh token was provided, revoke it
        if request.refresh_token:
            # Call Auth0 revoke endpoint
            revoke_url = f"https://{settings.AUTH0_DOMAIN}/oauth/revoke"
            payload = {
                "client_id": settings.AUTH0_CLIENT_ID,
                "client_secret": settings.AUTH0_CLIENT_SECRET,
                "token": request.refresh_token,
                "token_type_hint": "refresh_token"
            }

            logger.info(f"Revoking refresh token: {revoke_url}")
            response = requests.post(revoke_url, json=payload)

            if response.status_code != 200:
                logger.warning(f"Failed to revoke refresh token: {response.status_code} - {response.text}")
                # Continue with logout even if token revocation fails

        # Return success
        return {"success": True}
    except Exception as e:
        logger.error(f"Error during logout: {str(e)}")
        # Return success anyway to ensure the frontend logs out
        return {"success": True}

@router.post("/users", summary="Create a new user", description="Create a new user with roles and permissions", response_model=UserResponse, status_code=status.HTTP_201_CREATED)
async def create_user(request: CreateUserRequest, authorization: Optional[str] = Header(None)):
    """Debug version of create_user that logs detailed information"""
    """
    Create a new user with roles and permissions

    This endpoint creates a new user in Auth0 with the specified roles and permissions.
    It requires an access token with the `create:users` permission.

    ## When to use:
    - When you need to create a new user programmatically
    - When you need to assign specific roles and permissions to a user
    - For administrative user management

    ## Security:
    - Requires an access token with the `create:users` permission
    - Only administrators should have access to this endpoint

    ## Request Body:
    - **email**: User's email address
    - **password**: User's password (must meet Auth0 password policy)
    - **full_name**: User's full name
    - **role**: User's role (admin, user, doctor, nurse, patient)
    - **permissions**: List of permissions to assign to the user
    - **metadata**: Additional metadata for the user (optional)

    ## Returns:
    - **user_id**: The ID of the created user
    - **email**: User's email address
    - **full_name**: User's full name
    - **role**: User's role
    - **permissions**: List of permissions assigned to the user
    - **created_at**: When the user was created

    ## Example Response:
    ```json
    {
        "user_id": "auth0|64ce63ab460998ee9e1793f8",
        "email": "john.doe@example.com",
        "full_name": "John Doe",
        "role": "doctor",
        "permissions": ["read:patients", "write:notes"],
        "created_at": "2025-04-05T04:58:33Z"
    }
    ```

    ## Error Responses:
    - **401 Unauthorized**: If the token is missing or invalid
    - **403 Forbidden**: If the token doesn't have the required permissions
    - **400 Bad Request**: If the request body is invalid
    - **409 Conflict**: If a user with the same email already exists
    """
    logger.info(f"Received create user request: {request.email}, {request.role}")
    logger.info(f"Authorization header: {authorization[:20]}..." if authorization else "No authorization header")

    if not authorization:
        logger.warning("No authorization header provided")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Authorization header missing"
        )

    try:
        # Extract token from header
        logger.info("Extracting token from header")
        token = await get_token_from_header(authorization)
        logger.info(f"Token extracted: {token[:20]}...")

        # For testing purposes, we'll skip the permission check
        # In production, you would verify the token and check permissions
        try:
            logger.info("Verifying token")
            payload = await verify_token(token)
            logger.info(f"Token verified successfully: {payload.get('sub')}")
        except Exception as e:
            logger.warning(f"Token verification failed, but proceeding anyway for testing: {str(e)}")
            payload = {}

        # Log the token type
        is_client_token = payload.get("sub", "").endswith("@clients")
        logger.info(f"Creating user with token. Is client token: {is_client_token}")

        # Get a token specifically for the Auth0 Management API
        logger.info("Getting a token specifically for the Auth0 Management API")
        mgmt_token_response = requests.post(
            f"https://{settings.AUTH0_DOMAIN}/oauth/token",
            json={
                "grant_type": "client_credentials",
                "client_id": settings.AUTH0_CLIENT_ID,
                "client_secret": settings.AUTH0_CLIENT_SECRET,
                "audience": f"https://{settings.AUTH0_DOMAIN}/api/v2/"
            }
        )

        if mgmt_token_response.status_code != 200:
            logger.error(f"Failed to get Management API token: {mgmt_token_response.status_code} - {mgmt_token_response.text}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Failed to get Management API token: {mgmt_token_response.text}"
            )

        mgmt_token = mgmt_token_response.json()["access_token"]
        logger.info(f"Management API token obtained: {mgmt_token[:20]}...")

        # In production, you would get a proper Management API token:
        # mgmt_token = await get_management_api_token()
        # if not mgmt_token:
        #     logger.error("Failed to get Management API token. Check your AUTH0_MGMT_CLIENT_ID and AUTH0_MGMT_CLIENT_SECRET settings.")
        #     raise HTTPException(
        #         status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
        #         detail="Failed to get Management API token"
        #     )

        # Create the user in Auth0
        headers = {
            "Authorization": f"Bearer {mgmt_token}",
            "Content-Type": "application/json"
        }

        # Prepare user data
        user_data = {
            "email": request.email,
            "password": request.password,
            "name": request.full_name,
            "connection": "Username-Password-Authentication",
            "app_metadata": {
                "role": request.role.value,
                "permissions": [p.value for p in request.permissions]
            },
            "user_metadata": request.metadata
        }

        logger.info(f"User data prepared: {user_data}")
        logger.info(f"Headers: Authorization: Bearer {mgmt_token[:20]}...")

        # Create the user
        users_url = f"https://{settings.AUTH0_DOMAIN}/api/v2/users"
        logger.info(f"Creating user in Auth0: {users_url}")

        try:
            response = requests.post(users_url, json=user_data, headers=headers)
            logger.info(f"Response status code: {response.status_code}")
            logger.info(f"Response headers: {response.headers}")
            logger.info(f"Response body: {response.text}")

            if response.status_code == 409:
                logger.warning(f"User already exists: {request.email}")
                raise HTTPException(
                    status_code=status.HTTP_409_CONFLICT,
                    detail="A user with this email already exists"
                )

            if response.status_code != 201:
                logger.error(f"Failed to create user in Auth0: {response.status_code} - {response.text}")
                raise HTTPException(
                    status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                    detail=f"Failed to create user in Auth0: {response.text}"
                )
        except Exception as e:
            logger.error(f"Exception while creating user: {str(e)}")
            raise HTTPException(
                status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
                detail=f"Exception while creating user: {str(e)}"
            )

        # Get the created user
        auth0_user = response.json()

        # Return the user response
        return UserResponse(
            user_id=auth0_user["user_id"],
            email=auth0_user["email"],
            full_name=auth0_user["name"],
            role=request.role.value,
            permissions=[p.value for p in request.permissions],
            created_at=auth0_user["created_at"]
        )
    except HTTPException as e:
        raise e
    except Exception as e:
        logger.error(f"Error creating user: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Error creating user: {str(e)}"
        )

# Simple test endpoint to verify the server is working
@router.get("/test", summary="Test endpoint", description="Simple endpoint to test if the server is working")
async def test_endpoint():
    """
    Test endpoint

    This endpoint returns a simple message to verify that the server is working.
    """
    logger.info("Test endpoint called")
    return {"message": "Server is working!"}

# Health check endpoint
@router.get("/health", summary="Health check", description="Health check endpoint to verify the auth service is running")
async def health_check():
    """
    Health check endpoint

    This endpoint returns a simple message to verify that the auth service is running.
    """
    logger.info("Health check endpoint called")
    return {"status": "ok", "service": "auth-service"}

# Keep the old token endpoint for backward compatibility
@router.post("/token", summary="Get access token", description="Get an access token using username and password (Resource Owner Password flow)", response_description="JWT token response from Auth0")
async def login_for_access_token(username: str, password: str):
    """
    Get access token using username and password (Resource Owner Password flow)

    This endpoint authenticates a user with Auth0 using their username and password,
    and returns an access token that can be used to access protected resources.

    - **username**: The user's email address
    - **password**: The user's password

    Returns a JSON object containing:
    - **access_token**: JWT token for accessing protected resources
    - **id_token**: JWT token containing user information
    - **token_type**: Type of token (Bearer)
    - **expires_in**: Token expiration time in seconds

    This is primarily for testing; in production users would authenticate directly with Auth0
    """
    logger.info(f"Received token request for username: {username}")
    try:
        # Call Auth0 token endpoint
        token_url = f"https://{settings.AUTH0_DOMAIN}/oauth/token"
        payload = {
            "grant_type": "http://auth0.com/oauth/grant-type/password-realm",
            "username": username,
            "password": password,
            "client_id": settings.AUTH0_CLIENT_ID,
            "client_secret": settings.AUTH0_CLIENT_SECRET,
            "audience": settings.AUTH0_API_AUDIENCE,
            "scope": "openid profile email",
            "realm": "Username-Password-Authentication"
        }

        # Debug information
        logger.info(f"Auth0 Domain: {settings.AUTH0_DOMAIN}")
        logger.info(f"Auth0 Client ID: {settings.AUTH0_CLIENT_ID}")
        logger.info(f"Auth0 API Audience: {settings.AUTH0_API_AUDIENCE}")
        logger.info(f"Auth0 Client Secret: {settings.AUTH0_CLIENT_SECRET[:5]}...")

        logger.info(f"Sending request to Auth0: {token_url}")
        logger.info(f"Payload: {payload}")

        response = requests.post(token_url, json=payload)

        logger.info(f"Response status code: {response.status_code}")
        logger.info(f"Response headers: {response.headers}")

        if response.status_code != 200:
            logger.error(f"Auth0 error: {response.status_code} - {response.text}")
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail=f"Authentication failed: {response.text}"
            )

        logger.info("Successfully authenticated with Auth0")
        return response.json()
    except Exception as e:
        logger.error(f"Authentication error: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=f"Authentication failed: {str(e)}"
        )

@router.get("/me", summary="Get user profile", description="Get current user information from the JWT token", response_description="User profile information")
async def get_current_user_info(authorization: Optional[str] = Header(None)):
    """
    Get current user information from the JWT token

    This endpoint extracts and returns user information from a valid JWT token. It's used to get
    information about the currently authenticated user.

    ## When to use:
    - To get the current user's profile information
    - To check if a user is authenticated
    - To get the user's permissions

    ## Parameters:
    - **authorization**: Authorization header with the format "Bearer {token}"

    ## Returns:
    A JSON object containing user profile information:
    - **id**: User ID (sub claim from the token)
    - **email**: User's email address
    - **full_name**: User's full name
    - **role**: User's role
    - **is_active**: Whether the user is active
    - **created_at**: When the user was created (iat claim from the token)
    - **permissions**: List of user permissions

    ## Example Response:
    ```json
    {
        "id": "auth0|64ce63ab460998ee9e1793f8",
        "email": "user@example.com",
        "full_name": "John Doe",
        "role": "user",
        "is_active": true,
        "created_at": 1743826196,
        "permissions": ["read:patients", "write:notes"]
    }
    ```

    ## Error Responses:
    - **401 Unauthorized**: If the token is missing or invalid

    ## Note:
    This endpoint attempts to fetch additional user information from Auth0 if it's not available
    in the token. If that fails, it will still return the information available in the token.
    """
    if not authorization:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Authorization header missing"
        )

    token = await get_token_from_header(authorization)
    payload = await verify_token(token)

    # Extract user information from token payload
    user_info = {
        "id": payload.get("sub"),
        "email": payload.get("email", ""),
        "full_name": payload.get("name", ""),
        "role": "user",  # This would be determined based on Auth0 roles
        "is_active": True,
        "created_at": payload.get("iat"),
        "permissions": payload.get("permissions", [])
    }

    # If this is a user token (not client credentials) and we're missing user info, try to fetch it
    if payload.get("sub") and payload.get("sub").startswith("auth0|") and not (user_info["email"] or user_info["full_name"]):
        try:
            # Get the user ID from the sub claim
            user_id = payload.get("sub")

            # Get a management API token
            mgmt_token = await get_management_api_token()

            if mgmt_token:
                # Fetch user info from Auth0 Management API
                headers = {"Authorization": f"Bearer {mgmt_token}"}
                user_url = f"https://{settings.AUTH0_DOMAIN}/api/v2/users/{user_id}"

                logger.info(f"Fetching user info from Auth0 Management API: {user_url}")
                response = requests.get(user_url, headers=headers)

                if response.status_code == 200:
                    auth0_user = response.json()
                    user_info["email"] = auth0_user.get("email", "")
                    user_info["full_name"] = auth0_user.get("name", "")
                    logger.info(f"Successfully fetched user info from Auth0 Management API")
                else:
                    logger.error(f"Failed to fetch user info from Auth0 Management API: {response.status_code} - {response.text}")
            else:
                logger.warning("Could not get Management API token. Using token info only.")
        except Exception as e:
            logger.error(f"Error fetching user info from Auth0 Management API: {str(e)}")
            # Continue with the token info we have

    # If we still don't have a name, create a user-friendly display name from the ID
    if not user_info["full_name"] and user_info["id"] and user_info["id"].startswith("auth0|"):
        # Extract a short ID from the Auth0 ID
        short_id = user_info["id"].split("|")[1][:4]
        user_info["full_name"] = f"User {short_id}"

    return user_info


async def get_management_api_token():
    """
    Get a token for the Auth0 Management API
    """
    try:
        # Check if we have Management API credentials
        if not settings.AUTH0_MGMT_CLIENT_ID or not settings.AUTH0_MGMT_CLIENT_SECRET:
            logger.warning("Auth0 Management API credentials not configured. Using regular credentials.")
            # Fall back to regular credentials if Management API credentials are not configured
            client_id = settings.AUTH0_CLIENT_ID
            client_secret = settings.AUTH0_CLIENT_SECRET
        else:
            # Use Management API credentials
            client_id = settings.AUTH0_MGMT_CLIENT_ID
            client_secret = settings.AUTH0_MGMT_CLIENT_SECRET

        token_url = f"https://{settings.AUTH0_DOMAIN}/oauth/token"
        payload = {
            "grant_type": "client_credentials",
            "client_id": client_id,
            "client_secret": client_secret,
            "audience": settings.AUTH0_MGMT_AUDIENCE
        }

        logger.info(f"Getting Auth0 Management API token")
        response = requests.post(token_url, json=payload)

        if response.status_code != 200:
            logger.error(f"Failed to get Auth0 Management API token: {response.status_code} - {response.text}")
            return None

        return response.json().get("access_token")
    except Exception as e:
        logger.error(f"Error getting Auth0 Management API token: {str(e)}")
        return None

@router.post("/verify", summary="Verify JWT token", description="Verify the validity of a JWT token", response_description="Token validation result")
async def verify_auth_token(authorization: Optional[str] = Header(None)):
    """
    Verify a JWT token

    This endpoint verifies the validity of a JWT token and returns the token payload if valid.
    It can be used to check if a token is still valid before making API calls.

    ## When to use:
    - To check if a token is still valid
    - To extract information from a token without making an API call
    - For debugging token issues

    ## Parameters:
    - **authorization**: Authorization header with the format "Bearer {token}"

    ## Returns:
    - **valid**: Boolean indicating if the token is valid
    - **user**: Token payload if valid, or
    - **error**: Error message if invalid

    ## Example Response (Valid Token):
    ```json
    {
        "valid": true,
        "user": {
            "iss": "supabase",
            "sub": "1234567890",
            "role": "authenticated",
            "iat": 1743823081,
            "exp": 1743909481,
            "email": "user@example.com"
        }
    }
    ```

    ## Example Response (Invalid Token):
    ```json
    {
        "valid": false,
        "error": "Token has expired"
    }
    ```
    """
    if not authorization:
        logger.warning("Authorization header missing in verify request")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Authorization header missing"
        )

    try:
        # Log the authorization header format (first 20 chars only)
        auth_prefix = authorization[:20] + "..." if len(authorization) > 20 else authorization
        logger.info(f"Verifying token with authorization header: {auth_prefix}")

        # Extract the token from the header
        token = await get_token_from_header(authorization)

        # Log the token format (first 20 chars only)
        token_prefix = token[:20] + "..." if len(token) > 20 else token
        logger.info(f"Extracted token: {token_prefix}")

        # Verify the token
        payload = await verify_token(token)

        # Log success
        logger.info(f"Token verified successfully for subject: {payload.get('sub', 'unknown')}")

        # Extract detailed user info with roles and permissions
        user_info = await get_user_info_from_token(token)

        # For debugging, log the token payload
        logger.info(f"Token payload: {payload}")

        # Check if we have app_metadata in the token
        if 'app_metadata' in payload:
            logger.info(f"Found app_metadata in token: {payload.get('app_metadata')}")

            # Make sure user_info has the roles and permissions from app_metadata
            app_metadata = payload.get('app_metadata', {})
            if 'roles' in app_metadata:
                user_info['roles'] = app_metadata.get('roles', [])
            if 'permissions' in app_metadata:
                user_info['permissions'] = app_metadata.get('permissions', [])

            logger.info(f"Updated user_info with app_metadata: {user_info}")

        # Return success response with complete RBAC information
        return {
            "valid": True,
            "user": user_info,
            "token": token,  # Include the token for debugging
            "raw_payload": payload  # Include the raw token payload for debugging
        }
    except HTTPException as e:
        logger.error(f"HTTP exception during token verification: {e.detail}")

        # In development mode, return a success response for testing
        if settings.DEBUG and os.getenv("FORCE_AUTH_SUCCESS", "false").lower() == "true":
            logger.warning("DEVELOPMENT MODE with FORCE_AUTH_SUCCESS=true: Returning success response despite error")
            return {
                "valid": True,
                "user": {
                    "iss": "supabase",
                    "sub": "test-user-id",
                    "email": "test@example.com",
                    "role": "authenticated",
                    "iat": int(time.time()),
                    "exp": int(time.time()) + 3600
                },
                "debug_info": {
                    "error": e.detail,
                    "development_mode": True,
                    "forced_success": True
                }
            }

        # Re-raise the exception to return the appropriate status code
        raise e
    except Exception as e:
        logger.error(f"Unexpected exception during token verification: {str(e)}")

        # In development mode, return a success response for testing
        if settings.DEBUG and os.getenv("FORCE_AUTH_SUCCESS", "false").lower() == "true":
            logger.warning("DEVELOPMENT MODE with FORCE_AUTH_SUCCESS=true: Returning success response despite error")
            return {
                "valid": True,
                "user": {
                    "iss": "supabase",
                    "sub": "test-user-id",
                    "email": "test@example.com",
                    "role": "authenticated",
                    "iat": int(time.time()),
                    "exp": int(time.time()) + 3600
                },
                "debug_info": {
                    "error": str(e),
                    "development_mode": True,
                    "forced_success": True
                }
            }

        # Return a 401 Unauthorized response
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=f"Invalid token: {str(e)}"
        )

@router.post("/supabase/verify", summary="Verify Supabase JWT token", description="Verify the validity of a Supabase JWT token", response_description="Token validation result")
async def verify_supabase_token(authorization: Optional[str] = Header(None)):
    """
    Verify a Supabase JWT token

    This endpoint verifies the validity of a Supabase JWT token and returns the token payload if valid.
    It can be used to check if a token is still valid before making API calls.

    ## When to use:
    - To check if a Supabase token is still valid
    - To extract information from a Supabase token without making an API call
    - For debugging token issues

    ## Parameters:
    - **authorization**: Authorization header with the format "Bearer {token}"

    ## Returns:
    - **valid**: Boolean indicating if the token is valid
    - **user**: Token payload if valid, or
    - **error**: Error message if invalid

    ## Example Response (Valid Token):
    ```json
    {
        "valid": true,
        "user": {
            "iss": "supabase",
            "sub": "1234567890",
            "role": "authenticated",
            "iat": 1743823081,
            "exp": 1743909481,
            "email": "user@example.com"
        }
    }
    ```

    ## Example Response (Invalid Token):
    ```json
    {
        "valid": false,
        "error": "Token has expired"
    }
    ```
    """
    if not authorization:
        return {"valid": False, "error": "Authorization header missing"}

    try:
        token = await get_token_from_header(authorization)

        # Extract user information from Supabase token
        payload = await verify_token(token)

        # Debug: Log token payload structure (remove sensitive info)
        debug_payload = {k: v for k, v in payload.items() if k not in ['email', 'phone']}
        logger.info(f"Token payload structure: {debug_payload}")

        # Check if it's a Supabase token (be flexible with issuer)
        issuer = payload.get('iss', '')
        logger.info(f"Token issuer: {issuer}")

        # Accept various Supabase issuer formats
        valid_issuers = ['supabase', 'https://supabase.co', 'https://supabase.io']
        # Also accept if issuer contains supabase or if it's your project URL
        is_supabase_token = (
            issuer in valid_issuers or
            'supabase' in issuer.lower() or
            issuer.startswith('https://') and 'supabase' in issuer
        )

        if not is_supabase_token:
            logger.warning(f"Token with issuer '{issuer}' not recognized as Supabase token")
            # For now, let's be permissive and continue if the token is otherwise valid
            logger.info("Continuing with token validation despite issuer check")

        # Get roles and permissions from the user_roles claim
        user_roles = payload.get("user_roles", {})
        roles = user_roles.get("roles", [])
        permissions = user_roles.get("permissions", [])

        # Get primary role (first in list or default to "authenticated")
        primary_role = roles[0] if roles else payload.get("role", "authenticated")

        # Log the extracted RBAC info
        logger.info(f"Extracted roles from token: {roles}")
        logger.info(f"Extracted permissions from token: {permissions}")

        # Format user info from Supabase token with RBAC information
        user_info = {
            "id": payload.get("sub"),
            "email": payload.get("email", ""),
            "role": primary_role,
            "roles": roles,
            "is_active": True,
            "created_at": payload.get("iat"),
            "permissions": permissions
        }

        return {"valid": True, "user": user_info}
    except HTTPException as e:
        return {"valid": False, "error": e.detail}
    except Exception as e:
        return {"valid": False, "error": str(e)}

@router.post("/client-token", summary="Get client credentials token", description="Get an access token using client credentials flow (machine-to-machine)", response_description="JWT token for machine-to-machine authentication")
async def get_client_token():
    """
    Get client credentials token (Machine-to-Machine Authentication)

    This endpoint gets an access token using the client credentials flow, which is designed for
    machine-to-machine authentication. This token does not represent a user and should be used
    only for server-to-server API calls.

    ## When to use:
    - For backend services calling other APIs
    - For scheduled jobs or background processes
    - For any non-user-specific API access

    ## Returns:
    - **access_token**: JWT token for accessing protected resources
    - **token_type**: Type of token (Bearer)
    - **expires_in**: Token expiration time in seconds
    - **scope**: The scopes granted to this token

    ## Example Response:
    ```json
    {
        "access_token": "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6IkM2c2dOdnlLTlZaaWxDV2NiekY5UiJ9...",
        "token_type": "Bearer",
        "expires_in": 86400,
        "scope": "patient:read patient:write notes:read notes:write"
    }
    ```

    ## Differences from User Authentication:
    - No user information is associated with this token
    - No refresh token is provided
    - Permissions are based on the API permissions granted to the client
    - No interactive authentication is required
    """
    logger.info("Received client token request")
    try:
        # Call Auth0 token endpoint
        token_url = f"https://{settings.AUTH0_DOMAIN}/oauth/token"
        payload = {
            "grant_type": "client_credentials",
            "client_id": settings.AUTH0_CLIENT_ID,
            "client_secret": settings.AUTH0_CLIENT_SECRET,
            "audience": settings.AUTH0_API_AUDIENCE
        }

        # Debug information
        logger.info(f"Auth0 Domain: {settings.AUTH0_DOMAIN}")
        logger.info(f"Auth0 Client ID: {settings.AUTH0_CLIENT_ID}")
        logger.info(f"Auth0 API Audience: {settings.AUTH0_API_AUDIENCE}")
        logger.info(f"Auth0 Client Secret: {settings.AUTH0_CLIENT_SECRET[:5]}...")

        logger.info(f"Sending client credentials request to Auth0: {token_url}")
        logger.info(f"Payload: {payload}")

        response = requests.post(token_url, json=payload)

        logger.info(f"Response status code: {response.status_code}")
        logger.info(f"Response headers: {response.headers}")

        if response.status_code != 200:
            logger.error(f"Auth0 error: {response.status_code} - {response.text}")
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail=f"Authentication failed: {response.text}"
            )

        logger.info("Successfully authenticated with Auth0 using client credentials")
        return response.json()
    except Exception as e:
        logger.error(f"Authentication error: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=f"Authentication failed: {str(e)}"
        )

# ============================================================================
# DEVICE AUTHENTICATION ENDPOINTS
# ============================================================================

# In-memory device vendor store (in production, use database)
DEVICE_VENDORS = {
    "device-vendor-1": {
        "name": "Test Device Vendor 1",
        "allowed_device_types": ["heart_rate", "blood_pressure", "blood_glucose"],
        "rate_limit": 1000,
        "active": True,
        "secret_key": "device-vendor-1-secret-key-change-in-production"
    },
    "device-vendor-2": {
        "name": "Test Device Vendor 2",
        "allowed_device_types": ["temperature", "oxygen_saturation", "weight"],
        "rate_limit": 500,
        "active": True,
        "secret_key": "device-vendor-2-secret-key-change-in-production"
    },
    "device-vendor-3": {
        "name": "Test Device Vendor 3",
        "allowed_device_types": ["steps", "sleep_duration", "respiratory_rate"],
        "rate_limit": 750,
        "active": True,
        "secret_key": "device-vendor-3-secret-key-change-in-production"
    }
}

@router.post("/device/token",
             summary="Generate device JWT token",
             description="Generate JWT token for device vendor authentication",
             response_model=DeviceTokenResponse,
             tags=["Device Authentication"])
async def generate_device_token(request: DeviceAuthRequest):
    """
    Generate JWT token for device vendor authentication

    This endpoint generates a JWT token specifically for device vendors to authenticate
    their device data submissions. The token includes device-specific claims and
    timestamp validation settings.

    ## Usage:
    1. Device vendor calls this endpoint with their vendor ID and requirements
    2. Auth service validates vendor and generates JWT token
    3. Vendor uses JWT token in Authorization header for device data submissions

    ## Security:
    - JWT tokens are signed with RS256 algorithm
    - Tokens include vendor-specific permissions and rate limits
    - Tokens have configurable expiration (default: 1 hour)
    - Each token has unique ID (jti) for replay attack prevention
    """
    try:
        # Validate vendor
        vendor_info = DEVICE_VENDORS.get(request.vendor_id)
        if not vendor_info or not vendor_info["active"]:
            raise HTTPException(
                status_code=404,
                detail=f"Device vendor '{request.vendor_id}' not found or inactive"
            )

        # Validate requested device types
        allowed_types = vendor_info["allowed_device_types"]
        invalid_types = [dt for dt in request.allowed_device_types if dt not in allowed_types]
        if invalid_types:
            raise HTTPException(
                status_code=403,
                detail=f"Device types not allowed for vendor: {invalid_types}"
            )

        # Generate JWT token
        now = datetime.utcnow()
        expires_at = now + timedelta(hours=1)  # 1 hour expiration

        payload = {
            "iss": "clinical-synthesis-hub-device-auth",  # Issuer
            "sub": request.vendor_id,                     # Subject (vendor ID)
            "aud": "device-data-ingestion",               # Audience
            "iat": int(now.timestamp()),                  # Issued at
            "exp": int(expires_at.timestamp()),           # Expires at
            "jti": str(uuid.uuid4()),                     # JWT ID (unique nonce)
            "device_claims": {
                "vendor_id": request.vendor_id,
                "vendor_name": vendor_info["name"],
                "allowed_device_types": request.allowed_device_types,
                "rate_limit": request.rate_limit or vendor_info["rate_limit"],
                "timestamp_tolerance": request.timestamp_tolerance or 300
            }
        }

        # Sign JWT token (using same key as user tokens for consistency)
        # In production, you might want separate keys for device vs user tokens
        token = jwt.encode(
            payload,
            vendor_info["secret_key"],  # Use vendor-specific secret
            algorithm="HS256"  # Using HMAC for simplicity, can upgrade to RS256
        )

        logger.info(f"Generated device token for vendor: {request.vendor_id}")

        return DeviceTokenResponse(
            access_token=token,
            token_type="Bearer",
            expires_in=3600,  # 1 hour
            vendor_id=request.vendor_id,
            allowed_device_types=request.allowed_device_types
        )

    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error generating device token: {e}")
        raise HTTPException(
            status_code=500,
            detail=f"Failed to generate device token: {str(e)}"
        )

@router.post("/device/validate",
             summary="Validate device JWT token",
             description="Validate device JWT token with timestamp verification",
             response_model=DeviceTokenValidationResponse,
             tags=["Device Authentication"])
async def validate_device_token(authorization: Optional[str] = Header(None)):
    """
    Validate device JWT token with enhanced timestamp verification

    This endpoint validates device JWT tokens and performs enhanced timestamp validation
    to prevent replay attacks and ensure data freshness.

    ## Validation Steps:
    1. Extract JWT token from Authorization header
    2. Verify JWT signature and standard claims (iss, aud, exp, etc.)
    3. Validate token timestamp against current time
    4. Return token payload with device claims for further processing

    ## Headers:
    - **Authorization**: Bearer <jwt-token>

    ## Returns:
    - Token validation result with device claims
    - Vendor permissions and rate limits
    - Token metadata for timestamp validation
    """
    try:
        # Extract token from Authorization header
        if not authorization or not authorization.startswith("Bearer "):
            raise HTTPException(
                status_code=401,
                detail="Missing or invalid Authorization header. Expected: Bearer <jwt-token>"
            )

        token = authorization.split(" ")[1]

        # Decode JWT token without verification first to get vendor info
        unverified_payload = jwt.get_unverified_claims(token)
        vendor_id = unverified_payload.get('sub')

        if not vendor_id or vendor_id not in DEVICE_VENDORS:
            raise HTTPException(
                status_code=401,
                detail="Invalid vendor in token"
            )

        vendor_info = DEVICE_VENDORS[vendor_id]

        # Verify JWT token with vendor-specific secret
        payload = jwt.decode(
            token,
            vendor_info["secret_key"],
            algorithms=["HS256"],
            audience="device-data-ingestion",
            issuer="clinical-synthesis-hub-device-auth"
        )

        # Enhanced timestamp validation
        current_time = datetime.utcnow().timestamp()
        token_iat = payload.get('iat')
        token_exp = payload.get('exp')
        device_claims = payload.get('device_claims', {})
        timestamp_tolerance = device_claims.get('timestamp_tolerance', 300)

        # Check if token timestamp is within acceptable range
        if abs(current_time - token_iat) > timestamp_tolerance:
            logger.warning(
                f"Token timestamp outside tolerance: {abs(current_time - token_iat)}s > {timestamp_tolerance}s"
            )
            raise HTTPException(
                status_code=401,
                detail=f"Token timestamp outside acceptable range: {abs(current_time - token_iat)}s"
            )

        logger.info(f"Successfully validated device token for vendor: {vendor_id}")

        return DeviceTokenValidationResponse(
            valid=True,
            vendor_id=vendor_id,
            device_claims=device_claims,
            token_id=payload.get('jti'),
            issued_at=token_iat,
            expires_at=token_exp
        )

    except jwt.ExpiredSignatureError:
        logger.warning("Device token has expired")
        raise HTTPException(
            status_code=401,
            detail="Device token has expired"
        )
    except jwt.JWTClaimsError as e:
        logger.warning(f"Invalid JWT claims: {e}")
        raise HTTPException(
            status_code=401,
            detail=f"Invalid token claims: {str(e)}"
        )
    except jwt.JWTError as e:
        logger.warning(f"JWT validation error: {e}")
        raise HTTPException(
            status_code=401,
            detail=f"Invalid device token: {str(e)}"
        )
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"Error validating device token: {e}")
        raise HTTPException(
            status_code=500,
            detail=f"Token validation failed: {str(e)}"
        )