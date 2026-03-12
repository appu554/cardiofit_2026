#!/usr/bin/env python3
"""
Minimal FastAPI test to isolate startup issues
"""

import os
import sys
import asyncio
from contextlib import asynccontextmanager

# Add the app directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app'))

from fastapi import FastAPI
import structlog

logger = structlog.get_logger(__name__)

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Minimal lifespan management"""
    logger.info("🚀 Starting minimal FastAPI app...")
    
    try:
        # Test our components
        logger.info("🔧 Testing component imports...")
        from app.services.multi_sink_writer import MultiSinkWriterService
        from app.services.fhir_transformation import FHIRTransformationService
        
        logger.info("🔧 Creating services...")
        fhir_transformer = FHIRTransformationService()
        multi_sink_writer = MultiSinkWriterService()
        
        logger.info("🔧 Initializing multi-sink writer...")
        await asyncio.wait_for(multi_sink_writer.initialize(), timeout=30.0)
        
        logger.info("✅ Minimal FastAPI app started successfully!")
        
        yield
        
    except Exception as e:
        logger.error("❌ Failed to start minimal FastAPI app", error=str(e))
        raise
    finally:
        logger.info("🛑 Shutting down minimal FastAPI app...")
        if 'multi_sink_writer' in locals():
            await multi_sink_writer.close()

# Create minimal FastAPI application
app = FastAPI(
    title="Minimal Stage 2 Test",
    description="Test FastAPI startup with Stage 2 components",
    version="1.0.0",
    lifespan=lifespan
)

@app.get("/")
async def root():
    return {"message": "Minimal Stage 2 test is working!"}

@app.get("/health")
async def health():
    return {"status": "healthy", "service": "minimal-stage2-test"}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8043)
