# Global Authentication for Clinical Synthesis Hub

This package provides a global authentication solution for all microservices in the Clinical Synthesis Hub.

## Overview

Instead of implementing authentication in each service individually, this package provides:

1. An authentication middleware that validates tokens with the Auth Service
2. Utility functions for getting the authenticated user
3. Decorators for permission and role-based access control

## Installation

To use this package in a service, copy the `shared` directory to your service's root directory or add it to your Python path.

## Usage

### Adding the Authentication Middleware

Add the middleware to your FastAPI application:

```python
from fastapi import FastAPI
from shared.auth.middleware import AuthenticationMiddleware

app = FastAPI()

# Add the authentication middleware
app.add_middleware(
    AuthenticationMiddleware,
    auth_service_url="http://auth-service:8001/api",  # Optional, defaults to environment variable
    exclude_paths=["/docs", "/openapi.json", "/health"]  # Optional, paths to exclude from authentication
)
```

### Getting the Authenticated User

Use the `get_current_user` function as a dependency in your route handlers:

```python
from fastapi import Depends, APIRouter
from shared.auth.middleware import get_current_user

router = APIRouter()

@router.get("/me")
async def get_me(user=Depends(get_current_user)):
    return user
```

### Using Permission Decorators

Use the decorators for permission-based access control:

```python
from fastapi import APIRouter
from shared.auth.decorators import require_permissions

router = APIRouter()

@router.get("/patients")
@require_permissions(["read:patients"])
async def get_patients(request: Request):
    # Access is granted only if the user has the "read:patients" permission
    return {"message": "Access granted"}
```

### Using Role Decorators

Use the decorators for role-based access control:

```python
from fastapi import APIRouter
from shared.auth.decorators import require_role

router = APIRouter()

@router.post("/admin/settings")
@require_role(["admin"])
async def update_settings(request: Request):
    # Access is granted only if the user has the "admin" role
    return {"message": "Settings updated"}
```

## Configuration

The authentication middleware uses the following configuration:

- `AUTH_SERVICE_URL`: The URL of the Auth Service (default: `http://localhost:8001/api`)
- `exclude_paths`: Paths to exclude from authentication (default: `["/docs", "/openapi.json", "/redoc", "/health"]`)

## Backward Compatibility

For backward compatibility with existing code, you can use the `get_token_payload` function:

```python
from fastapi import Depends
from shared.auth.middleware import get_token_payload, security

@router.get("/legacy")
async def legacy_route(payload=Depends(get_token_payload)):
    return {"message": "Legacy route", "user": payload}
```

## Docker Compose Configuration

Make sure your Docker Compose configuration includes the Auth Service:

```yaml
services:
  auth-service:
    build: ./services/auth-service
    ports:
      - "8001:8000"
    environment:
      - SUPABASE_URL=https://your-supabase-url.supabase.co
      - SUPABASE_KEY=your-supabase-key
      - SUPABASE_JWT_SECRET=your-supabase-jwt-secret
```

## Environment Variables

Set the following environment variables in your service:

```
AUTH_SERVICE_URL=http://auth-service:8001/api
```

## Security Considerations

- The Auth Service should be deployed in a secure environment
- All communication between services should be encrypted (HTTPS)
- The JWT secret should be kept secure and not exposed
- Token validation should always be performed before processing requests
