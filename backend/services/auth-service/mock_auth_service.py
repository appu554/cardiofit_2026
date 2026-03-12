from fastapi import FastAPI, Header, HTTPException, status
from fastapi.middleware.cors import CORSMiddleware
from typing import Dict, Optional, Any
import uvicorn
import logging
import json

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Initialize FastAPI app
app = FastAPI(
    title="Mock Auth Service",
    description="A mock authentication service for testing RBAC",
    version="1.0.0",
)

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Health check endpoint
@app.get("/health")
async def health_check():
    return {"status": "ok"}

# Mock token verification endpoint
@app.post("/api/auth/verify")
async def verify_auth_token(authorization: Optional[str] = Header(None)):
    """
    Mock endpoint to verify a JWT token
    
    This endpoint always returns a successful response with mock user data.
    """
    if not authorization:
        logger.warning("Authorization header missing in verify request")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Authorization header missing"
        )
    
    try:
        # Extract the token from the header
        scheme, token = authorization.split()
        if scheme.lower() != "bearer":
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Invalid authentication scheme"
            )
        
        # Log the token format (first 20 chars only)
        token_prefix = token[:20] + "..." if len(token) > 20 else token
        logger.info(f"Received token for verification: {token_prefix}")
        
        # Return a mock successful response
        return {
            "valid": True,
            "user": {
                "id": "mock-user-id",
                "email": "doctor@example.com",
                "full_name": "Mock Doctor",
                "role": "doctor",
                "roles": ["doctor"],
                "is_active": True,
                "created_at": 1743826196,
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
            }
        }
    except Exception as e:
        logger.error(f"Error in mock verify endpoint: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=f"Error: {str(e)}"
        )

# Mock permission check endpoint
@app.post("/api/auth/check-permissions")
async def check_permissions(request: Dict[str, Any], authorization: Optional[str] = Header(None)):
    """
    Mock endpoint to check permissions
    
    This endpoint always returns a successful response.
    """
    if not authorization:
        logger.warning("Authorization header missing in check-permissions request")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Authorization header missing"
        )
    
    # Log the request
    logger.info(f"Received permission check request: {json.dumps(request)}")
    
    # Return a mock successful response
    return {
        "has_permission": True,
        "detail": "Permission granted by mock auth service"
    }

if __name__ == "__main__":
    logger.info("Starting mock auth service on http://localhost:8001")
    uvicorn.run(app, host="0.0.0.0", port=8001)
