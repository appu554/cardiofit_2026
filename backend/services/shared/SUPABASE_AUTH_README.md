# Supabase Authentication Integration

This document provides instructions for integrating Supabase authentication with the backend services.

## Overview

The authentication system has been updated to use Supabase JWT tokens instead of Auth0. The Auth Service now validates Supabase tokens and provides a consistent interface for other services to verify tokens.

## Configuration

### Auth Service Configuration

The Auth Service has been updated to support Supabase JWT tokens. The configuration is in `app/config.py`:

```python
# Supabase Configuration
SUPABASE_URL: str = os.getenv("SUPABASE_URL", "https://auugxeqzgrnknklgwqrh.supabase.co")
SUPABASE_KEY: str = os.getenv("SUPABASE_KEY", "your-supabase-key")
SUPABASE_JWT_SECRET: str = os.getenv("SUPABASE_JWT_SECRET", "")  # This should be set in production
SUPABASE_ALGORITHMS: List[str] = ["HS256"]  # Supabase uses HS256 by default
```

For production, you should set the `SUPABASE_JWT_SECRET` environment variable to the JWT secret from your Supabase project settings.

### Service Configuration

Each service should have the Auth Service URL configured in its `app/core/config.py`:

```python
# Auth Service URL
AUTH_SERVICE_URL: str = os.getenv("AUTH_SERVICE_URL", "http://localhost:8001/api")
```

## Updating Services

To update a service to use Supabase authentication:

1. Copy the `auth_template.py` file to your service's `app/core/auth.py`
2. Customize the permissions in the `get_token_payload` function based on your service's requirements
3. Make sure your service's `app/core/config.py` has the `AUTH_SERVICE_URL` configured

## Token Validation Flow

1. The frontend authenticates with Supabase and gets a JWT token
2. The frontend includes the token in the Authorization header of API requests
3. The backend service extracts the token and calls the Auth Service to verify it
4. The Auth Service validates the token and returns the payload
5. The backend service uses the payload to authorize the request

## Endpoints

The Auth Service provides the following endpoints for token validation:

- `/api/auth/verify`: Verifies any JWT token (Supabase or Auth0)
- `/api/auth/supabase/verify`: Specifically verifies Supabase JWT tokens

## Backward Compatibility

The system maintains backward compatibility with Auth0 tokens for a smooth transition. The Auth Service will first try to validate a token as a Supabase token, and if that fails, it will try to validate it as an Auth0 token.

## Testing

To test the authentication system:

1. Get a token from Supabase using the frontend
2. Call the `/api/auth/verify` endpoint with the token
3. Verify that the response includes `"valid": true` and the user information
