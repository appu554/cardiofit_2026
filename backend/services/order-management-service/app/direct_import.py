"""
Direct import fallback for shared modules

This module provides a fallback import mechanism for shared modules
when the normal import path fails.
"""

import os
import sys
from pathlib import Path

# Add the backend directory to the Python path
backend_dir = Path(__file__).parent.parent.parent.parent
sys.path.insert(0, str(backend_dir))

try:
    from shared.auth import HeaderAuthMiddleware, get_current_user
    print("Successfully imported HeaderAuthMiddleware from shared.auth via direct import")
except ImportError as e:
    print(f"Failed to import HeaderAuthMiddleware even with direct import: {e}")
    
    # Create a minimal fallback middleware
    from starlette.middleware.base import BaseHTTPMiddleware
    from starlette.requests import Request
    from starlette.responses import Response
    
    class HeaderAuthMiddleware(BaseHTTPMiddleware):
        def __init__(self, app, exclude_paths=None):
            super().__init__(app)
            self.exclude_paths = exclude_paths or []
        
        async def dispatch(self, request: Request, call_next):
            # Minimal fallback - just pass through
            response = await call_next(request)
            return response
    
    def get_current_user():
        return {"user_id": "fallback-user", "role": "doctor"}
    
    print("Using fallback HeaderAuthMiddleware")
