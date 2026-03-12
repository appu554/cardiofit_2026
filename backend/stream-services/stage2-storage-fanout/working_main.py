#!/usr/bin/env python3
"""
Working main.py based on successful test apps
"""

import os
import sys
import asyncio
from contextlib import asynccontextmanager

# Add the app directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), 'app'))

from fastapi import FastAPI
import structlog
from prometheus_client import start_http_server

from app.config import settings
from app.services.kafka_consumer import KafkaConsumerService
from app.services.fhir_transformation import FHIRTransformationService
from app.services.multi_sink_writer import MultiSinkWriterService

# Use simple structlog configuration (complex config was causing hanging)
# structlog.configure(...)  # Commented out - was causing startup hang

logger = structlog.get_logger(__name__)

# Global service references
kafka_consumer = None
fhir_transformer = None
multi_sink_writer = None

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan management"""
    global kafka_consumer, fhir_transformer, multi_sink_writer
    
    logger.info("🚀 Starting Stage 2: Storage Fan-Out Service", port=settings.PORT)
    
    try:
        # Initialize services with detailed logging
        logger.info("🚀 STEP 1: Initializing FHIR transformation service...")
        fhir_transformer = FHIRTransformationService()
        logger.info("✅ STEP 1: FHIR transformation service created")
        
        logger.info("🚀 STEP 2: Initializing multi-sink writer service...")
        multi_sink_writer = MultiSinkWriterService()
        logger.info("✅ STEP 2: Multi-sink writer service created")
        
        # Initialize with timeout to prevent hanging
        logger.info("🚀 STEP 3: Initializing sinks (with 30s timeout)...")
        try:
            await asyncio.wait_for(multi_sink_writer.initialize(), timeout=30.0)
            logger.info("✅ STEP 3: Multi-sink writer initialized successfully")
        except asyncio.TimeoutError:
            logger.error("❌ STEP 3: Multi-sink writer initialization timed out after 30 seconds")
            raise
        except Exception as e:
            logger.error("❌ STEP 3: Multi-sink writer initialization failed", error=str(e))
            raise
        
        logger.info("🚀 STEP 4: Creating Kafka consumer service...")
        kafka_consumer = KafkaConsumerService(
            fhir_transformer=fhir_transformer,
            multi_sink_writer=multi_sink_writer
        )
        logger.info("✅ STEP 4: Kafka consumer service created")
        
        # Start Prometheus metrics server
        logger.info("🚀 STEP 5: Starting Prometheus metrics server...")
        if settings.PROMETHEUS_ENABLED:
            start_http_server(settings.PROMETHEUS_PORT)
            logger.info("✅ STEP 5: Prometheus metrics server started", port=settings.PROMETHEUS_PORT)
        else:
            logger.info("⏭️ STEP 5: Prometheus metrics disabled")

        # Start Kafka consumer in background (non-blocking)
        logger.info("🚀 STEP 6: Starting Kafka consumer in background...")
        consumer_task = asyncio.create_task(kafka_consumer.start_consuming())
        
        # Give it a moment to start, but don't wait for full connection
        await asyncio.sleep(1.0)
        logger.info("✅ STEP 6: Kafka consumer started in background")

        logger.info("✅ Stage 2 service started successfully")

        yield
        
    except Exception as e:
        logger.error("❌ Failed to start Stage 2 service", error=str(e))
        raise
    finally:
        # Cleanup
        logger.info("🛑 Shutting down Stage 2 service...")
        
        if kafka_consumer:
            await kafka_consumer.stop()
        
        if multi_sink_writer:
            await multi_sink_writer.close()
        
        # Cancel background tasks
        if 'consumer_task' in locals():
            consumer_task.cancel()
            try:
                await consumer_task
            except asyncio.CancelledError:
                pass
        
        logger.info("🛑 Stage 2 service shutdown complete")


# Create FastAPI application
app = FastAPI(
    title="Stage 2: Storage Fan-Out Service",
    description="Multi-sink writer with FHIR transformations for Clinical Synthesis Hub",
    version="1.0.0",
    lifespan=lifespan
)

@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": "stage2-storage-fanout",
        "version": "1.0.0",
        "description": "Multi-sink writer with FHIR transformations",
        "port": settings.PORT,
        "status": "running"
    }

@app.get("/health")
async def health():
    """Simple health check"""
    return {"status": "healthy", "service": "stage2-storage-fanout"}

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8042)
