#!/usr/bin/env python3
"""
Run script for Device Data Ingestion Service
"""
import os
import sys
import uvicorn

# Add the app directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app'))

from app.config import settings

if __name__ == "__main__":
    print(f"Starting {settings.PROJECT_NAME} on {settings.HOST}:{settings.PORT}")
    
    uvicorn.run(
        "app.main:app",
        host=settings.HOST,
        port=settings.PORT,
        reload=settings.DEBUG,
        log_level="info"
    )
