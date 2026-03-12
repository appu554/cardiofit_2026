# Global Authentication for Clinical Synthesis Hub

This document provides instructions for implementing the global authentication solution for all microservices in the Clinical Synthesis Hub.

## Overview

Instead of implementing authentication in each service individually, we've created a shared authentication module that all services can use. This approach has several advantages:

1. **Consistency**: All services use the same authentication logic
2. **Maintainability**: Changes to authentication logic only need to be made in one place
3. **Security**: Centralized validation of tokens through the Auth Service
4. **Simplicity**: Services don't need to implement their own token validation logic

## Implementation

### 1. Update Docker Compose Configuration

The Docker Compose configuration has been updated to:

- Add Supabase environment variables to the Auth Service
- Add the Auth Service URL to all services
- Mount the shared directory in all services
- Add dependencies on the Auth Service

### 2. Use the Authentication Middleware

Each service should use the authentication middleware in its main application file:

```python
from fastapi import FastAPI
from shared.auth.middleware import AuthenticationMiddleware
import os

app = FastAPI()

# Add authentication middleware
app.add_middleware(
    AuthenticationMiddleware,
    auth_service_url=os.getenv("AUTH_SERVICE_URL", "http://localhost:8001/api"),
    exclude_paths=["/docs", "/openapi.json", "/redoc", "/health"]
)
```

### 3. Use the Authentication Decorators

Use the decorators for permission and role-based access control:

```python
from fastapi import APIRouter, Request
from shared.auth.decorators import require_permissions, require_role

router = APIRouter()

@router.get("/patients")
@require_permissions(["read:patients"])
async def get_patients(request: Request):
    # Access is granted only if the user has the "read:patients" permission
    return {"message": "Access granted"}

@router.post("/admin/settings")
@require_role(["admin"])
async def update_settings(request: Request):
    # Access is granted only if the user has the "admin" role
    return {"message": "Settings updated"}
```

### 4. Get the Authenticated User

Use the `get_current_user` function as a dependency in your route handlers:

```python
from fastapi import Depends
from shared.auth.middleware import get_current_user

@router.get("/me")
async def get_me(user=Depends(get_current_user)):
    return user
```

## Configuration

### Environment Variables

Each service should have the following environment variable:

```
AUTH_SERVICE_URL=http://auth-service:8000/api
```

### Auth Service Configuration

The Auth Service should have the following environment variables:

```
SUPABASE_URL=https://your-supabase-url.supabase.co
SUPABASE_KEY=your-supabase-key
SUPABASE_JWT_SECRET=your-supabase-jwt-secret
```

## Testing

To test the authentication system:

1. Start the services using Docker Compose:
   ```
   docker-compose up
   ```

2. Get a token from Supabase using the frontend

3. Call a protected endpoint with the token:
   ```
   curl -H "Authorization: Bearer YOUR_TOKEN" http://localhost:8003/patients
   ```

## Troubleshooting

If you encounter authentication issues:

1. Check that the Auth Service is running and accessible
2. Verify that the AUTH_SERVICE_URL environment variable is set correctly
3. Check that the token is valid and not expired
4. Look at the logs of the Auth Service for validation errors

## Security Considerations

- The Auth Service should be deployed in a secure environment
- All communication between services should be encrypted (HTTPS)
- The JWT secret should be kept secure and not exposed
- Token validation should always be performed before processing requests

## Example Implementation

See the Patient Service for an example implementation of the global authentication solution:

```python
# backend/services/patient-service/app/main.py
from fastapi import FastAPI, Depends, Request
from fastapi.middleware.cors import CORSMiddleware
from shared.auth.middleware import AuthenticationMiddleware, get_current_user
from shared.auth.decorators import require_permissions, require_role
import logging
import os

# Create FastAPI app
app = FastAPI()

# Add authentication middleware
app.add_middleware(
    AuthenticationMiddleware,
    auth_service_url=os.getenv("AUTH_SERVICE_URL", "http://localhost:8001/api"),
    exclude_paths=["/docs", "/openapi.json", "/redoc", "/health"]
)

# Define routes
@app.get("/patients")
@require_permissions(["read:patients"])
async def get_patients(request: Request):
    # Access is granted only if the user has the "read:patients" permission
    return {"patients": [...]}
```
