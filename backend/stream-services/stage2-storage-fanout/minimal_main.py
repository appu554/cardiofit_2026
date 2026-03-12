#!/usr/bin/env python3
"""
Minimal main.py to isolate the hanging issue
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
    """Minimal lifespan - just log messages"""
    logger.info("🚀 MINIMAL: Starting lifespan...")
    
    try:
        logger.info("🚀 MINIMAL: Testing imports...")
        
        # Test imports one by one
        logger.info("🔧 Importing config...")
        from app.config import settings
        logger.info(f"✅ Config imported - Port: {settings.PORT}")
        
        logger.info("🔧 Importing FHIR transformer...")
        from app.services.fhir_transformation import FHIRTransformationService
        logger.info("✅ FHIR transformer imported")
        
        logger.info("🔧 Importing multi-sink writer...")
        from app.services.multi_sink_writer import MultiSinkWriterService
        logger.info("✅ Multi-sink writer imported")
        
        logger.info("🔧 Importing Kafka consumer...")
        from app.services.kafka_consumer import KafkaConsumerService
        logger.info("✅ Kafka consumer imported")
        
        logger.info("🔧 Creating FHIR transformer...")
        fhir_transformer = FHIRTransformationService()
        logger.info("✅ FHIR transformer created")
        
        logger.info("🔧 Creating multi-sink writer...")
        multi_sink_writer = MultiSinkWriterService()
        logger.info("✅ Multi-sink writer created")
        
        logger.info("🔧 Creating Kafka consumer...")
        kafka_consumer = KafkaConsumerService(
            fhir_transformer=fhir_transformer,
            multi_sink_writer=multi_sink_writer
        )
        logger.info("✅ Kafka consumer created")
        
        logger.info("✅ MINIMAL: All services created successfully!")
        
        yield
        
    except Exception as e:
        logger.error("❌ MINIMAL: Failed", error=str(e))
        import traceback
        traceback.print_exc()
        raise
    finally:
        logger.info("🛑 MINIMAL: Shutting down...")

# Create minimal FastAPI application
app = FastAPI(
    title="Minimal Stage 2",
    description="Minimal test to isolate hanging issue",
    version="1.0.0",
    lifespan=lifespan
)

@app.get("/")
async def root():
    return {"message": "Minimal Stage 2 is working!"}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8044)
