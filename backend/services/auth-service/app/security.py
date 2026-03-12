from fastapi import Depends, HTTPException, status
from fastapi.security import OAuth2AuthorizationCodeBearer, HTTPBearer, HTTPAuthorizationCredentials
from jose import jwt
import requests
import logging
import time
import os
from typing import Dict, List, Optional, Union
from app.config import settings

# Configure logging
logger = logging.getLogger(__name__)

# HTTP Bearer scheme for token authentication
security = HTTPBearer()

# Legacy Auth0 OAuth2 scheme (kept for backward compatibility)
oauth2_scheme = OAuth2AuthorizationCodeBearer(
    authorizationUrl=f"https://{settings.AUTH0_DOMAIN}/authorize",
    tokenUrl=f"https://{settings.AUTH0_DOMAIN}/oauth/token",
)

# Cache for Auth0 JWKS
jwks_cache = None
jwks_cache_timestamp = 0

def get_jwks():
    """Fetch the JSON Web Key Set (JWKS) from Auth0 (legacy support)"""
    global jwks_cache, jwks_cache_timestamp
    import time

    current_time = time.time()
    # Cache JWKS for 24 hours
    if jwks_cache and current_time - jwks_cache_timestamp < 86400:
        return jwks_cache

    try:
        jwks_url = f"https://{settings.AUTH0_DOMAIN}/.well-known/jwks.json"
        response = requests.get(jwks_url)
        response.raise_for_status()
        jwks_cache = response.json()
        jwks_cache_timestamp = current_time
        return jwks_cache
    except Exception as e:
        logger.error(f"Failed to fetch JWKS: {e}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail="Failed to fetch authentication keys"
        )

async def verify_supabase_token(token: str) -> Dict:
    """
    Verify and decode a Supabase JWT token with RBAC information

    Args:
        token: JWT token from Supabase

    Returns:
        Decoded token payload with RBAC information

    Raises:
        HTTPException: If token validation fails
    """
    try:
        # For Supabase tokens, we use the JWT secret to verify
        # In production, you should set SUPABASE_JWT_SECRET in your environment
        # For development, we can use a more permissive approach

        # First, try to decode without verification (for development)
        if not settings.SUPABASE_JWT_SECRET and settings.DEBUG:
            logger.warning("SUPABASE_JWT_SECRET not set, decoding token without verification (DEVELOPMENT ONLY)")
            try:
                # Get the unverified payload
                payload = jwt.get_unverified_claims(token)

                # Check if it's a Supabase token by looking for the 'iss' claim
                # Supabase tokens can have iss=supabase or iss=https://your-project.supabase.co/auth/v1
                if not payload:
                    logger.error("Empty payload")
                    raise jwt.JWTError("Empty payload")

                issuer = payload.get('iss', '')
                if not issuer.startswith('https://') and issuer != 'supabase':
                    logger.error(f"Not a Supabase token. Issuer: {issuer}")
                    raise jwt.JWTError("Not a Supabase token")

                logger.info(f"Successfully decoded token without verification: {payload.get('sub')}")

                # Check if RBAC data is present
                if 'user_roles' in payload:
                    logger.info(f"Found user_roles in token: {payload.get('user_roles')}")

                return payload
            except Exception as e:
                logger.error(f"Error decoding token claims: {str(e)}")
                # In development mode, return a fake payload for testing
                if settings.DEBUG:
                    logger.warning("DEVELOPMENT MODE: Returning fake payload for testing")
                    return {
                        "iss": "supabase",
                        "sub": "test-user-id",
                        "email": "test@example.com",
                        "role": "authenticated",
                        "iat": int(time.time()),
                        "exp": int(time.time()) + 3600,
                        "user_roles": {
                            "roles": ["doctor"],
                            "permissions": ["read:patients", "write:notes", "read:observations"]
                        }
                    }
                raise

        # For production, properly verify the token
        try:
            # Log the token format for debugging (first 10 chars only)
            token_prefix = token[:10] + "..." if len(token) > 10 else token
            logger.info(f"Verifying token: {token_prefix}")

            # Log the secret format for debugging (first 5 chars only)
            secret_prefix = settings.SUPABASE_JWT_SECRET[:5] + "..." if len(settings.SUPABASE_JWT_SECRET) > 5 else settings.SUPABASE_JWT_SECRET
            logger.info(f"Using JWT secret: {secret_prefix}")

            # Get JWT validation options from environment variables
            verify_audience = os.getenv("SUPABASE_JWT_VERIFY_AUDIENCE", "true").lower() == "true"
            verify_issuer = os.getenv("SUPABASE_JWT_VERIFY_ISSUER", "true").lower() == "true"

            # Set the correct audience for Supabase tokens
            audience = os.getenv("SUPABASE_JWT_AUDIENCE", "authenticated")

            # Set the correct issuer for Supabase tokens
            # Supabase tokens can have iss=supabase or iss=https://your-project.supabase.co/auth/v1
            issuer = os.getenv("SUPABASE_JWT_ISSUER", None)

            # If no issuer is specified, don't verify it
            if not issuer:
                verify_issuer = False

            # Log JWT validation options
            logger.info(f"JWT validation options:")
            logger.info(f"  verify_audience: {verify_audience}")
            logger.info(f"  verify_issuer: {verify_issuer}")
            logger.info(f"  audience: {audience}")
            logger.info(f"  issuer: {issuer}")

            # Set up options for token validation
            options = {
                "verify_signature": True,
                "verify_aud": verify_audience,
                "verify_iss": verify_issuer,
                "require_exp": True,
            }

            # For development/debugging, log the token header
            try:
                header = jwt.get_unverified_header(token)
                logger.info(f"Token header: {header}")
                logger.info(f"Token algorithm: {header.get('alg', 'unknown')}")
            except Exception as e:
                logger.error(f"Error getting token header: {str(e)}")

            # Try to decode the token with the configured secret
            try:
                payload = jwt.decode(
                    token,
                    settings.SUPABASE_JWT_SECRET,
                    algorithms=settings.SUPABASE_ALGORITHMS,
                    audience=audience if verify_audience else None,
                    issuer=issuer if verify_issuer else None,
                    options=options
                )
            except Exception as e:
                logger.error(f"Error decoding token with configured secret: {str(e)}")

                # If we're in development mode and the token is from Supabase, try with a more permissive approach
                if settings.DEBUG:
                    logger.warning("DEVELOPMENT MODE: Attempting to decode token without verification")
                    try:
                        # Get the unverified claims to check if it's a Supabase token
                        unverified_claims = jwt.get_unverified_claims(token)
                        if unverified_claims.get('iss') == 'https://auugxeqzgrnknklgwqrh.supabase.co/auth/v1':
                            logger.info("Token is from Supabase, returning unverified claims for development")
                            return unverified_claims
                    except Exception as inner_e:
                        logger.error(f"Error getting unverified claims: {str(inner_e)}")

                # Re-raise the original exception
                raise

            # Log success and RBAC information
            logger.info(f"Successfully verified token: {payload.get('sub')}")

            # Extract RBAC information from the token
            # First check for user_roles (custom claim)
            if 'user_roles' in payload:
                roles = payload.get('user_roles', {}).get('roles', [])
                permissions = payload.get('user_roles', {}).get('permissions', [])
                logger.info(f"RBAC roles from user_roles claim: {roles}")
                logger.info(f"RBAC permissions from user_roles claim: {permissions}")
            # Then check for app_metadata (Supabase standard)
            elif 'app_metadata' in payload:
                app_metadata = payload.get('app_metadata', {})
                roles = app_metadata.get('roles', [])
                permissions = app_metadata.get('permissions', [])
                logger.info(f"RBAC roles from app_metadata: {roles}")
                logger.info(f"RBAC permissions from app_metadata: {permissions}")

                # Add the roles and permissions to the payload in the expected format
                payload['user_roles'] = {
                    'roles': roles,
                    'permissions': permissions
                }
            else:
                logger.warning(f"No RBAC information found in token for user: {payload.get('sub')}")

                # For development, add default roles and permissions
                if settings.DEBUG:
                    logger.warning("DEVELOPMENT MODE: Adding default roles and permissions")
                    payload['user_roles'] = {
                        'roles': ['doctor'],
                        'permissions': ['patient:read', 'patient:write', 'observation:read']
                    }

            return payload
        except Exception as e:
            logger.error(f"Error verifying token: {str(e)}")
            raise

    except jwt.ExpiredSignatureError:
        logger.error("Token has expired")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Token has expired"
        )
    except jwt.JWTClaimsError as e:
        logger.error(f"Invalid claims in token: {str(e)}")

        # In development mode, if the error is about audience, try to decode without audience validation
        if settings.DEBUG and "audience" in str(e).lower():
            logger.warning("DEVELOPMENT MODE: Attempting to decode token without audience validation")
            try:
                # Set up options without audience validation
                options = {
                    "verify_signature": True,
                    "verify_aud": False,
                    "verify_iss": True,
                    "require_exp": True,
                }

                # Decode the token without audience validation
                payload = jwt.decode(
                    token,
                    settings.SUPABASE_JWT_SECRET,
                    algorithms=settings.SUPABASE_ALGORITHMS,
                    issuer="supabase",
                    options=options
                )

                logger.info(f"Successfully decoded token without audience validation: {payload.get('sub')}")
                return payload
            except Exception as inner_e:
                logger.error(f"Failed to decode token without audience validation: {str(inner_e)}")
                # Continue with the original error

        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=f"Invalid claims in token: {str(e)}"
        )
    except jwt.JWTError as e:
        logger.error(f"JWT error: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=f"Invalid token: {str(e)}"
        )
    except Exception as e:
        logger.error(f"Token validation error: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=f"Token validation failed: {str(e)}"
        )

async def verify_auth0_token(token: str) -> Dict:
    """
    Verify and decode an Auth0 JWT token (legacy support)

    Args:
        token: JWT token from Auth0

    Returns:
        Decoded token payload

    Raises:
        HTTPException: If token validation fails
    """
    try:
        # Fetch the JWKS
        jwks = get_jwks()

        # Extract the token header to get the key ID (kid)
        header = jwt.get_unverified_header(token)

        if "kid" not in header:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Invalid token header"
            )

        # Find the signing key in the JWKS
        rsa_key = None
        for key in jwks.get("keys", []):
            if key.get("kid") == header["kid"]:
                rsa_key = {
                    "kty": key.get("kty"),
                    "kid": key.get("kid"),
                    "use": key.get("use"),
                    "n": key.get("n"),
                    "e": key.get("e")
                }
                break

        if not rsa_key:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Unable to find appropriate key"
            )

        # Verify and decode the token
        payload = jwt.decode(
            token,
            rsa_key,
            algorithms=settings.AUTH0_ALGORITHMS,
            audience=settings.AUTH0_API_AUDIENCE,
            issuer=settings.AUTH0_ISSUER
        )

        return payload
    except jwt.ExpiredSignatureError:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Token has expired"
        )
    except jwt.JWTClaimsError:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid claims: please check the audience and issuer"
        )
    except jwt.JWTError:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid token"
        )
    except Exception as e:
        logger.error(f"Token validation error: {e}")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Token validation failed"
        )

async def verify_token(token: str) -> Dict:
    """
    Verify and decode a JWT token (supports both Supabase and Auth0)

    This function tries to verify the token as a Supabase token first,
    and falls back to Auth0 if that fails.

    Args:
        token: JWT token

    Returns:
        Decoded token payload

    Raises:
        HTTPException: If token validation fails with both methods
    """
    # Try to verify as Supabase token first
    try:
        return await verify_supabase_token(token)
    except HTTPException as e:
        # If it's not a Supabase token, try Auth0 (for backward compatibility)
        if settings.AUTH0_DOMAIN:  # Only try Auth0 if configured
            try:
                return await verify_auth0_token(token)
            except HTTPException:
                # If both fail, raise the original Supabase error
                raise e
        else:
            # If Auth0 is not configured, just raise the Supabase error
            raise e

async def get_current_user(token: str = Depends(oauth2_scheme)) -> Dict:
    """
    Get the current authenticated user from the token

    Args:
        token: JWT token

    Returns:
        User payload from the token
    """
    return await verify_token(token)

async def get_token_payload(credentials: HTTPAuthorizationCredentials = Depends(security)) -> Dict:
    """
    Get the token payload from the authorization credentials

    Args:
        credentials: HTTP Authorization credentials

    Returns:
        Token payload
    """
    token = credentials.credentials
    return await verify_token(token)

async def get_token_from_header(authorization: str) -> str:
    """
    Extract token from Authorization header

    Args:
        authorization: Authorization header value

    Returns:
        JWT token

    Raises:
        HTTPException: If header format is invalid
    """
    if not authorization:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Authorization header missing"
        )

    parts = authorization.split()

    if len(parts) != 2 or parts[0].lower() != "bearer":
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Invalid authorization header format"
        )

    return parts[1]