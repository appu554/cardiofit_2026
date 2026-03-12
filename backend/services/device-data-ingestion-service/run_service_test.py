#!/usr/bin/env python3
"""
Test run script for Device Data Ingestion Service on different port
"""
import os
import sys
import uvicorn

# Add the app directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app'))

from app.config import settings

if __name__ == "__main__":
    # Use port 8016 to avoid conflict
    test_port = 8016
    print(f"Starting {settings.PROJECT_NAME} on {settings.HOST}:{test_port} (TEST MODE)")
    print("Note: Using port 8016 to avoid conflict with existing service on 8015")
    
    uvicorn.run(
        "app.main:app",
        host=settings.HOST,
        port=test_port,
        reload=settings.DEBUG,
        log_level="info"
    )
