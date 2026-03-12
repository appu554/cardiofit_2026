#!/usr/bin/env python3
"""
Debug main.py to isolate the exact hanging issue
"""

import os
import sys
import asyncio
from contextlib import asynccontextmanager

# Add the app directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app'))

print("🔧 DEBUG: Starting debug main...")

# Test imports one by one
print("🔧 DEBUG: Importing FastAPI...")
from fastapi import FastAPI

print("🔧 DEBUG: Importing structlog...")
import structlog

print("🔧 DEBUG: Importing config...")
from app.config import settings

print("🔧 DEBUG: Basic imports successful")

# Simple logger without complex configuration
logger = structlog.get_logger(__name__)

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Debug lifespan - minimal logging"""
    print("🚀 DEBUG: Lifespan starting...")
    logger.info("🚀 DEBUG: Lifespan starting with structlog...")
    
    try:
        print("🔧 DEBUG: In try block...")
        logger.info("🔧 DEBUG: In try block with structlog...")
        
        print("🔧 DEBUG: About to yield...")
        logger.info("🔧 DEBUG: About to yield...")
        
        yield
        
        print("🔧 DEBUG: After yield...")
        logger.info("🔧 DEBUG: After yield...")
        
    except Exception as e:
        print(f"❌ DEBUG: Exception in lifespan: {e}")
        logger.error("❌ DEBUG: Exception in lifespan", error=str(e))
        raise
    finally:
        print("🛑 DEBUG: Finally block...")
        logger.info("🛑 DEBUG: Finally block...")

print("🔧 DEBUG: Creating FastAPI app...")

# Create minimal FastAPI application
app = FastAPI(
    title="Debug Stage 2",
    description="Debug version to isolate hanging issue",
    version="1.0.0",
    lifespan=lifespan
)

print("🔧 DEBUG: FastAPI app created")

@app.get("/")
async def root():
    return {"message": "Debug Stage 2 is working!"}

print("🔧 DEBUG: Routes defined")

if __name__ == "__main__":
    print("🔧 DEBUG: Starting uvicorn...")
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8046)
