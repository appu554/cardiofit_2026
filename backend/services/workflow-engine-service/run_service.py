#!/usr/bin/env python3
"""
Run script for Workflow Engine Service.
"""
import os
import sys
import uvicorn

# Add the current directory to Python path
current_dir = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, current_dir)

# Add the backend services directory to Python path for shared imports
backend_dir = os.path.dirname(os.path.dirname(current_dir))
services_dir = os.path.join(backend_dir, "services")
sys.path.insert(0, services_dir)

if __name__ == "__main__":
    # Import settings after path setup
    from app.core.config import settings
    
    print(f"Starting {settings.SERVICE_NAME} on port {settings.SERVICE_PORT}")
    print(f"Debug mode: {settings.DEBUG}")
    print(f"Google Healthcare API: {'Enabled' if settings.USE_GOOGLE_HEALTHCARE_API else 'Disabled'}")
    
    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=settings.SERVICE_PORT,
        reload=settings.DEBUG,
        log_level="info"
    )
