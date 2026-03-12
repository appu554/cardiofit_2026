"""
Example FastAPI application using the global authentication middleware.
This file demonstrates how to use the authentication middleware in a FastAPI application.
"""

from fastapi import FastAPI, Depends, Request, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from shared.auth.middleware import AuthenticationMiddleware, get_current_user
from shared.auth.decorators import require_permissions, require_role

# Create FastAPI app
app = FastAPI(
    title="Example Service",
    description="Example service using global authentication",
    version="1.0.0"
)

# Add CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # Replace with specific origins in production
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Add authentication middleware
app.add_middleware(
    AuthenticationMiddleware,
    auth_service_url="http://auth-service:8001/api",  # Replace with your Auth Service URL
    exclude_paths=["/docs", "/openapi.json", "/redoc", "/health"]  # Paths to exclude from authentication
)

# Define routes
@app.get("/")
async def root():
    """
    Public endpoint that doesn't require authentication.
    This endpoint is excluded from authentication by the middleware.
    """
    return {"message": "Welcome to the Example Service"}

@app.get("/health")
async def health():
    """
    Health check endpoint that doesn't require authentication.
    This endpoint is excluded from authentication by the middleware.
    """
    return {"status": "healthy"}

@app.get("/me")
async def get_me(user=Depends(get_current_user)):
    """
    Get the authenticated user's information.
    This endpoint requires authentication.
    """
    return user

@app.get("/protected")
async def protected_route(request: Request):
    """
    Protected endpoint that requires authentication.
    This endpoint requires authentication, but doesn't check permissions.
    """
    # The user is already authenticated by the middleware
    # and is available in request.state.user
    user = request.state.user
    return {"message": "This is a protected endpoint", "user": user}

@app.get("/admin")
@require_role(["admin"])
async def admin_route(request: Request):
    """
    Admin endpoint that requires the admin role.
    This endpoint requires authentication and the admin role.
    """
    return {"message": "This is an admin endpoint"}

@app.get("/patients")
@require_permissions(["read:patients"])
async def get_patients(request: Request):
    """
    Patients endpoint that requires the read:patients permission.
    This endpoint requires authentication and the read:patients permission.
    """
    return {"message": "This is a patients endpoint"}

# Run the app
if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8000)
