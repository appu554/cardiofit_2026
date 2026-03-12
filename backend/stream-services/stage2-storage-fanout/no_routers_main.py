#!/usr/bin/env python3
"""
Test main.py without router imports to isolate hanging issue
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

# Global service references
kafka_consumer = None
fhir_transformer = None
multi_sink_writer = None

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Test lifespan without router imports"""
    global kafka_consumer, fhir_transformer, multi_sink_writer
    
    logger.info("🚀 NO-ROUTERS: Starting lifespan...")
    
    try:
        # Test the exact same imports as main.py but without routers
        logger.info("🔧 Testing config import...")
        from app.config import settings
        logger.info("✅ Config imported")
        
        logger.info("🔧 Testing service imports...")
        from app.services.kafka_consumer import KafkaConsumerService
        from app.services.fhir_transformation import FHIRTransformationService
        from app.services.multi_sink_writer import MultiSinkWriterService
        logger.info("✅ Service imports OK")
        
        # DON'T import routers - test if that's the issue
        logger.info("🔧 Skipping router imports...")
        
        logger.info("🔧 Creating services...")
        fhir_transformer = FHIRTransformationService()
        multi_sink_writer = MultiSinkWriterService()
        kafka_consumer = KafkaConsumerService(
            fhir_transformer=fhir_transformer,
            multi_sink_writer=multi_sink_writer
        )
        logger.info("✅ Services created")
        
        logger.info("✅ NO-ROUTERS: App started successfully!")
        
        yield
        
    except Exception as e:
        logger.error("❌ NO-ROUTERS: Failed", error=str(e))
        import traceback
        traceback.print_exc()
        raise
    finally:
        logger.info("🛑 NO-ROUTERS: Shutting down...")

# Create FastAPI application without routers
app = FastAPI(
    title="No-Routers Stage 2",
    description="Test without router imports",
    version="1.0.0",
    lifespan=lifespan
)

@app.get("/")
async def root():
    return {"message": "No-routers Stage 2 is working!"}

@app.get("/health")
async def health():
    return {"status": "healthy", "service": "no-routers-stage2"}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8045)
