"""
Authentication Middleware for SLA Monitoring Service
Handles JWT token validation and user authentication
"""

import os
import jwt
from datetime import datetime, timedelta
from typing import Optional, Dict, Any
import structlog
from fastapi import HTTPException, Request, Depends
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
import httpx
from functools import wraps
import asyncio

logger = structlog.get_logger()

security = HTTPBearer()


class AuthMiddleware:
    """
    JWT-based authentication middleware for SLA monitoring service
    Supports local JWT validation and remote auth service verification
    """

    def __init__(self):
        self.jwt_secret = os.getenv("JWT_SECRET", "cardiofit-runtime-sla-secret")
        self.jwt_algorithm = os.getenv("JWT_ALGORITHM", "HS256")
        self.auth_service_url = os.getenv("AUTH_SERVICE_URL", "http://localhost:8001")
        self.token_cache = {}  # Simple in-memory cache for validated tokens
        self.cache_ttl = 300  # 5 minutes cache TTL

        # Admin users who have full access to SLA management
        self.admin_users = set(os.getenv("SLA_ADMIN_USERS", "admin,sre-team").split(","))

        # Service accounts that can perform monitoring operations
        self.service_accounts = set(os.getenv("SLA_SERVICE_ACCOUNTS", "sla-monitor,prometheus").split(","))

        logger.info(
            "auth_middleware_initialized",
            auth_service_url=self.auth_service_url,
            admin_users_count=len(self.admin_users),
            service_accounts_count=len(self.service_accounts)
        )

    async def get_current_user(self, credentials: HTTPAuthorizationCredentials = Depends(security)) -> Dict[str, Any]:
        """
        Extract and validate user from JWT token
        Returns user information for authorized requests
        """
        try:
            token = credentials.credentials

            # Check cache first
            cached_user = self._get_cached_user(token)
            if cached_user:
                return cached_user

            # Validate token
            user = await self._validate_token(token)

            # Cache valid token
            self._cache_user(token, user)

            return user

        except Exception as e:
            logger.warning("authentication_failed", error=str(e))
            raise HTTPException(
                status_code=401,
                detail="Invalid or expired authentication token",
                headers={"WWW-Authenticate": "Bearer"}
            )

    async def get_admin_user(self, user: Dict[str, Any] = Depends(lambda self=None: self.get_current_user() if self else None)) -> Dict[str, Any]:
        """
        Validate that user has admin privileges for SLA management
        """
        if not self._is_admin_user(user):
            logger.warning("admin_access_denied", user_id=user.get("sub"))
            raise HTTPException(
                status_code=403,
                detail="Administrative privileges required for this operation"
            )

        return user

    async def get_service_account(self, credentials: HTTPAuthorizationCredentials = Depends(security)) -> Dict[str, Any]:
        """
        Validate service account authentication for automated operations
        """
        user = await self.get_current_user(credentials)

        if not self._is_service_account(user):
            logger.warning("service_account_access_denied", user_id=user.get("sub"))
            raise HTTPException(
                status_code=403,
                detail="Service account privileges required for this operation"
            )

        return user

    async def _validate_token(self, token: str) -> Dict[str, Any]:
        """
        Validate JWT token using local secret or remote auth service
        """
        try:
            # Try local JWT validation first
            payload = jwt.decode(
                token,
                self.jwt_secret,
                algorithms=[self.jwt_algorithm]
            )

            # Check token expiration
            exp = payload.get("exp")
            if exp and datetime.utcnow().timestamp() > exp:
                raise jwt.ExpiredSignatureError("Token has expired")

            # Ensure required claims
            required_claims = ["sub", "iat"]
            missing_claims = [claim for claim in required_claims if claim not in payload]
            if missing_claims:
                raise jwt.InvalidTokenError(f"Missing required claims: {missing_claims}")

            logger.debug("jwt_token_validated", user_id=payload.get("sub"))

            return payload

        except jwt.InvalidTokenError:
            # Fallback to remote auth service validation
            return await self._validate_with_auth_service(token)

    async def _validate_with_auth_service(self, token: str) -> Dict[str, Any]:
        """
        Validate token with remote auth service
        """
        try:
            async with httpx.AsyncClient(timeout=5.0) as client:
                response = await client.post(
                    f"{self.auth_service_url}/validate",
                    headers={"Authorization": f"Bearer {token}"},
                    json={"token": token}
                )

                if response.status_code == 200:
                    user_data = response.json()
                    logger.debug("auth_service_validation_success", user_id=user_data.get("sub"))
                    return user_data
                else:
                    logger.warning(
                        "auth_service_validation_failed",
                        status_code=response.status_code,
                        response=response.text
                    )
                    raise HTTPException(status_code=401, detail="Token validation failed")

        except httpx.RequestError as e:
            logger.error("auth_service_unavailable", error=str(e))
            # In production, you might want to fail closed here
            # For now, we'll allow the request if local validation worked
            raise HTTPException(
                status_code=503,
                detail="Authentication service temporarily unavailable"
            )

    def _get_cached_user(self, token: str) -> Optional[Dict[str, Any]]:
        """Get user from cache if token is still valid"""
        if token not in self.token_cache:
            return None

        cached_data = self.token_cache[token]
        if datetime.utcnow().timestamp() > cached_data["expires_at"]:
            # Cache expired, remove entry
            del self.token_cache[token]
            return None

        return cached_data["user"]

    def _cache_user(self, token: str, user: Dict[str, Any]):
        """Cache validated user data"""
        expires_at = datetime.utcnow().timestamp() + self.cache_ttl

        self.token_cache[token] = {
            "user": user,
            "expires_at": expires_at
        }

        # Clean up old cache entries periodically
        if len(self.token_cache) > 1000:
            self._cleanup_cache()

    def _cleanup_cache(self):
        """Remove expired entries from cache"""
        current_time = datetime.utcnow().timestamp()
        expired_tokens = [
            token for token, data in self.token_cache.items()
            if current_time > data["expires_at"]
        ]

        for token in expired_tokens:
            del self.token_cache[token]

        logger.debug("auth_cache_cleanup", removed_entries=len(expired_tokens))

    def _is_admin_user(self, user: Dict[str, Any]) -> bool:
        """Check if user has admin privileges"""
        user_id = user.get("sub", "")
        username = user.get("username", user.get("preferred_username", ""))

        # Check user ID or username against admin list
        return (
            user_id in self.admin_users or
            username in self.admin_users or
            user.get("role") == "admin" or
            "sla_admin" in user.get("roles", [])
        )

    def _is_service_account(self, user: Dict[str, Any]) -> bool:
        """Check if user is a valid service account"""
        user_id = user.get("sub", "")
        username = user.get("username", user.get("preferred_username", ""))

        return (
            user_id in self.service_accounts or
            username in self.service_accounts or
            user.get("account_type") == "service" or
            "service_account" in user.get("roles", [])
        )

    def create_service_token(self, service_name: str, expires_hours: int = 24) -> str:
        """
        Create a service account token for automated operations
        Used for internal service-to-service authentication
        """
        payload = {
            "sub": service_name,
            "username": service_name,
            "account_type": "service",
            "roles": ["service_account"],
            "iat": datetime.utcnow(),
            "exp": datetime.utcnow() + timedelta(hours=expires_hours),
            "iss": "sla-monitoring-service"
        }

        token = jwt.encode(payload, self.jwt_secret, algorithm=self.jwt_algorithm)

        logger.info(
            "service_token_created",
            service_name=service_name,
            expires_hours=expires_hours
        )

        return token

    def create_admin_token(self, admin_username: str, expires_hours: int = 8) -> str:
        """
        Create an admin token for management operations
        Should be used sparingly and with proper authorization
        """
        payload = {
            "sub": admin_username,
            "username": admin_username,
            "role": "admin",
            "roles": ["admin", "sla_admin"],
            "iat": datetime.utcnow(),
            "exp": datetime.utcnow() + timedelta(hours=expires_hours),
            "iss": "sla-monitoring-service"
        }

        token = jwt.encode(payload, self.jwt_secret, algorithm=self.jwt_algorithm)

        logger.info(
            "admin_token_created",
            username=admin_username,
            expires_hours=expires_hours
        )

        return token

    async def verify_service_health_access(self, user: Dict[str, Any]) -> bool:
        """
        Verify user has access to service health information
        Less restrictive than admin access
        """
        # Allow admin users, service accounts, and users with monitoring role
        return (
            self._is_admin_user(user) or
            self._is_service_account(user) or
            "monitor" in user.get("roles", []) or
            "sla_viewer" in user.get("roles", [])
        )

    def require_health_access(self, user: Dict[str, Any] = Depends(lambda self=None: self.get_current_user() if self else None)) -> Dict[str, Any]:
        """
        Dependency that requires health monitoring access
        """
        if not asyncio.run(self.verify_service_health_access(user)):
            raise HTTPException(
                status_code=403,
                detail="Insufficient privileges for health monitoring access"
            )

        return user


# Utility functions for token operations
def extract_token_from_header(authorization_header: Optional[str]) -> Optional[str]:
    """Extract JWT token from Authorization header"""
    if not authorization_header:
        return None

    try:
        scheme, token = authorization_header.split()
        if scheme.lower() != "bearer":
            return None
        return token
    except ValueError:
        return None


def create_test_token(username: str = "test-user", is_admin: bool = False) -> str:
    """
    Create a test token for development and testing
    Should only be used in non-production environments
    """
    if os.getenv("ENVIRONMENT") == "production":
        raise RuntimeError("Test tokens cannot be created in production environment")

    auth_middleware = AuthMiddleware()

    if is_admin:
        return auth_middleware.create_admin_token(username, expires_hours=1)
    else:
        return auth_middleware.create_service_token(username, expires_hours=1)


# Decorator for functions that require authentication
def require_auth(admin_required: bool = False):
    """
    Decorator for functions that require authentication
    Usage: @require_auth() or @require_auth(admin_required=True)
    """
    def decorator(func):
        @wraps(func)
        async def wrapper(*args, **kwargs):
            # This would be used for non-FastAPI functions that need auth
            # Implementation depends on how you want to handle auth outside of FastAPI dependencies
            pass
        return wrapper
    return decorator