from fastapi import Depends, HTTPException, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
import httpx
import logging
import os
import json
from typing import Dict, Any

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

security = HTTPBearer()

# Default Auth Service URL
DEFAULT_AUTH_SERVICE_URL = "http://localhost:8001"

async def get_token_payload(credentials: HTTPAuthorizationCredentials = Depends(security)) -> Dict[str, Any]:
    """
    Validate the token with the Auth Service and return the payload.

    Args:
        credentials: The HTTP Authorization credentials

    Returns:
        A dictionary containing the token payload

    Raises:
        HTTPException: If the token is invalid or the Auth Service is unavailable
    """
    # Extract the token from the credentials
    token = credentials.credentials

    # Get the Auth Service URL from environment variables
    auth_service_url = os.getenv("AUTH_SERVICE_URL", DEFAULT_AUTH_SERVICE_URL)

    # Check if we're in strict mode
    strict_mode = os.getenv("ENVIRONMENT", "development").lower() == "production"

    # Construct the verify URL
    # The auth service endpoint is at /api/auth/verify
    verify_url = f"{auth_service_url}/api/auth/verify"

    # Log the constructed URL
    logger.info(f"Constructed verify URL: {verify_url}")

    # Log the token validation attempt
    logger.info(f"Validating token with Auth Service at {verify_url} (strict_mode={strict_mode})")

    # Try to validate the token with the Auth Service
    try:
        # Use a short timeout to prevent hanging
        async with httpx.AsyncClient(timeout=3.0) as client:
            try:
                # Make the request to the Auth Service
                logger.info(f"Making request to Auth Service at: {verify_url}")
                response = await client.post(
                    verify_url,
                    headers={"Authorization": f"Bearer {token}"},
                    timeout=3.0  # Explicit timeout for the request
                )

                # Check if the response is successful
                if response.status_code == 200:
                    # Parse the response
                    result = response.json()

                    # Check if the token is valid
                    if result.get("valid", False):
                        # Get the user info from the response
                        user_info = result.get("user", {})

                        # Create a payload with the token and user info
                        payload = {
                            "token": token,
                            "sub": user_info.get("sub") or user_info.get("id", "unknown"),
                            "email": user_info.get("email", ""),
                            "role": user_info.get("role", "user"),
                            "permissions": user_info.get("permissions", ["read:patients", "write:patients"])
                        }

                        # Log successful validation
                        logger.info(f"Token validated successfully for user: {payload['sub']}")

                        return payload
                    else:
                        # Log the validation error
                        error_message = result.get("error", "Unknown error")
                        logger.error(f"Token validation failed: {error_message}")

                        # Raise an exception
                        raise HTTPException(
                            status_code=status.HTTP_401_UNAUTHORIZED,
                            detail=error_message,
                            headers={"WWW-Authenticate": "Bearer"},
                        )
                elif response.status_code == 401:
                    # Auth Service rejected the token
                    try:
                        error_detail = response.json().get("detail", "Invalid authentication credentials")
                    except Exception:
                        error_detail = "Invalid authentication credentials"

                    logger.error(f"Auth Service rejected the token: {error_detail}")

                    # In strict mode, reject the request
                    if strict_mode:
                        raise HTTPException(
                            status_code=status.HTTP_401_UNAUTHORIZED,
                            detail=error_detail,
                            headers={"WWW-Authenticate": "Bearer"},
                        )
                    else:
                        # In development mode, log a warning and continue
                        logger.warning(f"DEVELOPMENT MODE: Accepting request despite Auth Service rejection: {error_detail}")
                else:
                    # Log the error
                    logger.error(f"Auth Service returned status code {response.status_code}: {response.text}")

                    # Raise an exception
                    raise HTTPException(
                        status_code=status.HTTP_401_UNAUTHORIZED,
                        detail="Invalid authentication credentials",
                        headers={"WWW-Authenticate": "Bearer"},
                    )
            except httpx.HTTPStatusError as e:
                # Handle HTTP errors
                logger.error(f"HTTP error from Auth Service: {e.response.status_code} - {e.response.text}")

                # In strict mode, reject the request
                if strict_mode:
                    raise HTTPException(
                        status_code=status.HTTP_401_UNAUTHORIZED,
                        detail=f"Auth Service error: {e.response.status_code}",
                        headers={"WWW-Authenticate": "Bearer"},
                    )
                else:
                    # In development mode, log a warning and continue
                    logger.warning(f"DEVELOPMENT MODE: Accepting request despite Auth Service error: {e}")
            except Exception as e:
                # Handle other errors
                logger.error(f"Error calling Auth Service: {str(e)}")

                # In strict mode, reject the request
                if strict_mode:
                    raise HTTPException(
                        status_code=status.HTTP_401_UNAUTHORIZED,
                        detail=f"Error validating token: {str(e)}",
                        headers={"WWW-Authenticate": "Bearer"},
                    )
                else:
                    # In development mode, log a warning and continue
                    logger.warning(f"DEVELOPMENT MODE: Accepting request despite error: {str(e)}")

    except httpx.TimeoutException:
        # Log the timeout
        logger.error("Timeout when calling Auth Service")

        # In strict mode, reject the request
        if strict_mode:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Authentication service timeout",
                headers={"WWW-Authenticate": "Bearer"},
            )
        else:
            # In development mode, log a warning and continue
            logger.warning("DEVELOPMENT MODE: Accepting request despite Auth Service timeout")

    except httpx.RequestError as e:
        # Log the error
        logger.error(f"Error calling Auth Service: {str(e)}")

        # In strict mode, reject the request
        if strict_mode:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail=f"Authentication service unavailable: {str(e)}",
                headers={"WWW-Authenticate": "Bearer"},
            )
        else:
            # In development mode, log a warning and continue
            logger.warning(f"DEVELOPMENT MODE: Accepting request despite Auth Service error: {str(e)}")

    except Exception as e:
        # Log any other errors
        logger.error(f"Unexpected error in token validation: {str(e)}")

        # In strict mode, reject the request
        if strict_mode:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail=f"Authentication error: {str(e)}",
                headers={"WWW-Authenticate": "Bearer"},
            )
        else:
            # In development mode, log a warning and continue
            logger.warning(f"DEVELOPMENT MODE: Accepting request despite error: {str(e)}")

    # If we get here, we're in development mode and there was an error
    # Return a default payload for development
    logger.warning("DEVELOPMENT MODE: Returning default payload")
    return {
        "token": token,
        "sub": "test-user",
        "email": "test@example.com",
        "role": "user",
        "permissions": ["read:patients", "write:patients"]
    }
