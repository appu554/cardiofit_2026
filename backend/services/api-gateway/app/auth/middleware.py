from typing import Dict, Any, Callable, Optional
from fastapi import Request, HTTPException, status, Depends
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
import httpx
import logging
import os
from starlette.middleware.base import BaseHTTPMiddleware
from starlette.responses import Response

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Security scheme for token extraction
security = HTTPBearer()

# Default Auth Service URL
DEFAULT_AUTH_SERVICE_URL = "http://localhost:8001/api"

class AuthenticationMiddleware(BaseHTTPMiddleware):
    """
    Middleware for authenticating requests using the Auth Service.

    This middleware extracts the JWT token from the Authorization header,
    validates it with the Auth Service, and adds the user information to the request state.
    """

    def __init__(
        self,
        app,
        auth_service_url: Optional[str] = None,
        exclude_paths: Optional[list] = None
    ):
        super().__init__(app)
        self.auth_service_url = auth_service_url or os.getenv("AUTH_SERVICE_URL", DEFAULT_AUTH_SERVICE_URL)
        self.exclude_paths = exclude_paths or ["/docs", "/openapi.json", "/redoc", "/health"]
        logger.info(f"Initialized AuthenticationMiddleware with Auth Service URL: {self.auth_service_url}")

    async def dispatch(self, request: Request, call_next: Callable) -> Response:
        """
        Process the request, authenticate if needed, and pass it to the next middleware.

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

        # Get the token from the Authorization header
        auth_header = request.headers.get("Authorization")
        if not auth_header:
            logger.warning(f"No Authorization header found for path: {path}")
            return Response(
                status_code=status.HTTP_401_UNAUTHORIZED,
                content="Authorization header missing",
                headers={"WWW-Authenticate": "Bearer"}
            )

        # Extract the token from the Authorization header
        try:
            scheme, token = auth_header.split()
            if scheme.lower() != "bearer":
                logger.warning(f"Invalid authentication scheme: {scheme}")
                return Response(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    content="Invalid authentication scheme",
                    headers={"WWW-Authenticate": "Bearer"}
                )
        except ValueError:
            logger.warning(f"Invalid Authorization header format: {auth_header}")
            return Response(
                status_code=status.HTTP_401_UNAUTHORIZED,
                content="Invalid Authorization header format",
                headers={"WWW-Authenticate": "Bearer"}
            )

        # Validate the token with the Auth Service
        try:
            async with httpx.AsyncClient() as client:
                # Log the URL we're calling
                # The auth service endpoint is at /api/auth/verify
                # If auth_service_url already includes /api, don't add it again
                if "/api" in self.auth_service_url:
                    verify_url = f"{self.auth_service_url}/auth/verify"
                else:
                    verify_url = f"{self.auth_service_url}/api/auth/verify"
                logger.info(f"Calling Auth Service verify endpoint: {verify_url}")

                response = await client.post(
                    verify_url,
                    headers={"Authorization": f"Bearer {token}"}
                )

                if response.status_code != 200:
                    logger.error(f"Token validation failed: {response.status_code} - {response.text}")
                    return Response(
                        status_code=status.HTTP_401_UNAUTHORIZED,
                        content="Invalid authentication credentials",
                        headers={"WWW-Authenticate": "Bearer"}
                    )

                result = response.json()

                if not result.get("valid", False):
                    logger.error(f"Token validation failed: {result.get('error', 'Unknown error')}")
                    return Response(
                        status_code=status.HTTP_401_UNAUTHORIZED,
                        content=result.get("error", "Invalid token"),
                        headers={"WWW-Authenticate": "Bearer"}
                    )

                # Get the user info from the response
                user_info = result.get("user", {})

                # Extract RBAC information
                roles = user_info.get("roles", [])
                role = user_info.get("role", "authenticated")
                permissions = user_info.get("permissions", [])

                # Add the user info to the request state
                request.state.user = user_info

                # Add specific RBAC fields to request state for easier access
                request.state.user_role = role
                request.state.user_roles = roles
                request.state.user_permissions = permissions

                # Log RBAC information
                logger.info(f"User authenticated: {user_info.get('id')} with role: {role}")
                logger.info(f"User roles: {roles}")
                logger.info(f"User permissions: {permissions}")

                # Continue with the request
                return await call_next(request)

        except httpx.RequestError as e:
            logger.error(f"Error connecting to Auth Service: {str(e)}")
            return Response(
                status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
                content="Authentication service unavailable",
                headers={"WWW-Authenticate": "Bearer"}
            )

# Function to get the authenticated user from the request
async def get_current_user(request: Request) -> Dict[str, Any]:
    """
    Get the authenticated user from the request state.

    This function should be used as a dependency in route handlers.

    Args:
        request: The incoming request

    Returns:
        The authenticated user information

    Raises:
        HTTPException: If the user is not authenticated
    """
    user = getattr(request.state, "user", None)
    if not user:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Not authenticated",
            headers={"WWW-Authenticate": "Bearer"}
        )
    return user

# Function to get token payload (for backward compatibility)
async def get_token_payload(credentials: HTTPAuthorizationCredentials = Depends(security)) -> Dict[str, Any]:
    """
    Validate the token with the Auth Service and return the payload.

    This function is provided for backward compatibility with existing code.
    New code should use the AuthenticationMiddleware and get_current_user function.

    Args:
        credentials: The HTTP Authorization credentials

    Returns:
        The token payload

    Raises:
        HTTPException: If the token is invalid
    """
    token = credentials.credentials
    auth_service_url = os.getenv("AUTH_SERVICE_URL", DEFAULT_AUTH_SERVICE_URL)

    # Call the Auth Service to verify the token
    async with httpx.AsyncClient() as client:
        try:
            # Log the URL we're calling
            verify_url = f"{auth_service_url}/auth/verify"
            logger.info(f"get_token_payload: Calling Auth Service verify endpoint: {verify_url}")

            response = await client.post(
                verify_url,
                headers={"Authorization": f"Bearer {token}"}
            )

            if response.status_code != 200:
                logger.error(f"Token validation failed: {response.status_code} - {response.text}")
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail="Invalid authentication credentials",
                    headers={"WWW-Authenticate": "Bearer"},
                )

            result = response.json()

            if not result.get("valid", False):
                logger.error(f"Token validation failed: {result.get('error', 'Unknown error')}")
                raise HTTPException(
                    status_code=status.HTTP_401_UNAUTHORIZED,
                    detail=result.get("error", "Invalid token"),
                    headers={"WWW-Authenticate": "Bearer"},
                )

            # Get the user info from the response
            user_info = result.get("user", {})

            # Extract RBAC information
            roles = user_info.get("roles", [])
            role = user_info.get("role", "authenticated")
            permissions = user_info.get("permissions", [])

            # Create a payload with the token and user info including RBAC details
            payload = {
                "token": token,
                "sub": user_info.get("sub") or user_info.get("id", "unknown"),
                "email": user_info.get("email", ""),
                "role": role,
                "roles": roles,
                "permissions": permissions
            }

            # Log RBAC information
            logger.info(f"Token payload created with role: {role}")
            logger.info(f"Token payload roles: {roles}")
            logger.info(f"Token payload permissions: {permissions}")

            return payload

        except httpx.RequestError as e:
            logger.error(f"Error connecting to Auth Service: {str(e)}")
            raise HTTPException(
                status_code=status.HTTP_503_SERVICE_UNAVAILABLE,
                detail="Authentication service unavailable",
                headers={"WWW-Authenticate": "Bearer"},
            )
