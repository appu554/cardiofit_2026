"""
Stage 2: Storage Fan-Out Service
Port 8042 - Multi-Sink Writer with FHIR Transformations

This service implements the exact same FHIR transformation logic as the
monolithic PySpark reactor, but in a dedicated, scalable Python service.

Core Responsibility:
- ONLY consume "validated" events and handle complex sink writes
- NO validation, NO enrichment - those are Stage 1's job
- Focus: Reliable, parallel persistence to multiple storage systems
"""

import asyncio
import logging
import signal
import sys
from contextlib import asynccontextmanager

import structlog
import uvicorn
from fastapi import FastAPI
from prometheus_client import start_http_server

from app.config import settings
from app.services.kafka_consumer import KafkaConsumerService
from app.services.fhir_transformation import FHIRTransformationService
from app.services.multi_sink_writer import MultiSinkWriterService
# Temporarily comment out router imports to fix hanging issue
# from app.controllers.health_controller import router as health_router, set_service_references as set_health_refs
# from app.controllers.metrics_controller import router as metrics_router, set_service_references as set_metrics_refs

# Configure structured logging
structlog.configure(
    processors=[
        structlog.stdlib.filter_by_level,
        structlog.stdlib.add_logger_name,
        structlog.stdlib.add_log_level,
        structlog.stdlib.PositionalArgumentsFormatter(),
        structlog.processors.TimeStamper(fmt="iso"),
        structlog.processors.StackInfoRenderer(),
        structlog.processors.format_exc_info,
        structlog.processors.UnicodeDecoder(),
        structlog.processors.JSONRenderer()
    ],
    context_class=dict,
    logger_factory=structlog.stdlib.LoggerFactory(),
    wrapper_class=structlog.stdlib.BoundLogger,
    cache_logger_on_first_use=True,
)

logger = structlog.get_logger(__name__)

# Global services
kafka_consumer: KafkaConsumerService = None
fhir_transformer: FHIRTransformationService = None
multi_sink_writer: MultiSinkWriterService = None


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan management"""
    global kafka_consumer, fhir_transformer, multi_sink_writer
    
    logger.info("Starting Stage 2: Storage Fan-Out Service", port=settings.PORT)
    
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

        # Set service references for health and metrics controllers
        logger.info("🚀 STEP 6: Setting service references...")
        try:
            # Temporarily comment out to fix hanging issue
            # set_health_refs(kafka_consumer, fhir_transformer, multi_sink_writer)
            # set_metrics_refs(kafka_consumer, fhir_transformer, multi_sink_writer)
            logger.info("✅ STEP 6: Service references skipped (routers disabled)")
        except Exception as e:
            logger.warning("⚠️ STEP 6: Failed to set service references", error=str(e))

        # Start Kafka consumer in background (non-blocking)
        logger.info("🚀 STEP 7: Starting Kafka consumer in background...")
        consumer_task = asyncio.create_task(kafka_consumer.start_consuming())

        # Give it a moment to start, but don't wait for full connection
        await asyncio.sleep(1.0)
        logger.info("✅ STEP 7: Kafka consumer started in background")

        logger.info("Stage 2 service started successfully")

        yield
        
    except Exception as e:
        logger.error("Failed to start Stage 2 service", error=str(e))
        raise
    finally:
        # Cleanup
        logger.info("Shutting down Stage 2 service...")
        
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
        
        logger.info("Stage 2 service shutdown complete")


# Create FastAPI application
app = FastAPI(
    title="Stage 2: Storage Fan-Out Service",
    description="Multi-sink writer with FHIR transformations for Clinical Synthesis Hub",
    version="1.0.0",
    lifespan=lifespan
)

# Include routers (temporarily commented out to fix hanging issue)
# app.include_router(health_router, prefix="/api/v1/health", tags=["health"])
# app.include_router(metrics_router, prefix="/api/v1/metrics", tags=["metrics"])


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


@app.get("/api/v1/status")
async def status():
    """Service status endpoint"""
    global kafka_consumer, fhir_transformer, multi_sink_writer
    
    return {
        "service": "stage2-storage-fanout",
        "port": settings.PORT,
        "components": {
            "kafka_consumer": kafka_consumer.is_healthy() if kafka_consumer else False,
            "fhir_transformer": fhir_transformer.is_healthy() if fhir_transformer else False,
            "multi_sink_writer": multi_sink_writer.is_healthy() if multi_sink_writer else False
        },
        "topics": {
            "input": settings.KAFKA_INPUT_TOPIC,
            "dlq": settings.KAFKA_DLQ_TOPIC
        },
        "sinks": {
            "fhir_store": settings.FHIR_STORE_ENABLED,
            "elasticsearch": settings.ELASTICSEARCH_ENABLED,
            "mongodb": settings.MONGODB_ENABLED
        }
    }


def signal_handler(signum, frame):
    """Handle shutdown signals"""
    logger.info("Received shutdown signal", signal=signum)
    sys.exit(0)


if __name__ == "__main__":
    # Register signal handlers
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)
    
    # Configure logging level
    log_level = "DEBUG" if settings.DEBUG else "INFO"
    logging.basicConfig(level=getattr(logging, log_level))
    
    logger.info("Starting Stage 2: Storage Fan-Out Service", 
                port=settings.PORT, 
                debug=settings.DEBUG)
    
    # Run the application
    uvicorn.run(
        "app.main:app",
        host="0.0.0.0",
        port=settings.PORT,
        log_level=log_level.lower(),
        reload=settings.DEBUG,
        workers=1  # Single worker for Kafka consumer coordination
    )
