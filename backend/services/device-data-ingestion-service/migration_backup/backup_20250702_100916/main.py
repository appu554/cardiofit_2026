"""
Main FastAPI application for Device Data Ingestion Service
"""
import asyncio
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse

from app.config import settings
from app.api.routes import router as api_router
from app.api.resilience_routes import router as resilience_router
from app.kafka_producer import get_kafka_producer, close_kafka_producer

# Import database components for transactional outbox
from app.db.database import startup_database, shutdown_database, db_manager
from app.services.supabase_service import supabase_service

# Import background publisher for outbox processing
from app.services.background_publisher import start_background_publisher, stop_background_publisher

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """
    Application lifespan manager
    Handles startup and shutdown events
    """
    # Startup
    logger.info("Starting Device Data Ingestion Service...")

    try:
        # Initialize database connection for transactional outbox
        logger.info("Initializing database connection...")
        await startup_database()
        logger.info("Database connection established")

        # Initialize Supabase service (matching other services)
        logger.info("Initializing Supabase service...")
        supabase_initialized = await supabase_service.initialize()
        if supabase_initialized:
            logger.info("Supabase service initialized successfully")
        else:
            logger.warning("Supabase service initialization failed - continuing with database-only mode")

        # Initialize Kafka producer
        producer = await get_kafka_producer()
        logger.info("Kafka producer initialized")

        # Start background publisher for outbox processing
        logger.info("Starting background publisher...")
        asyncio.create_task(start_background_publisher())
        logger.info("Background publisher started")

        # Test Kafka connection
        if producer.health_check():
            logger.info("Kafka connection verified")
        else:
            logger.warning("Kafka connection could not be verified")

        logger.info("Device Data Ingestion Service started successfully")
        
    except Exception as e:
        logger.error(f"Failed to start service: {e}")
        raise
    
    yield
    
    # Shutdown
    logger.info("Shutting down Device Data Ingestion Service...")

    try:
        # Stop background publisher
        logger.info("Stopping background publisher...")
        await stop_background_publisher()
        logger.info("Background publisher stopped")

        # Close Kafka producer
        close_kafka_producer()
        logger.info("Kafka producer closed")

        # Close database connections
        await shutdown_database()
        logger.info("Database connections closed")

        logger.info("Device Data Ingestion Service shutdown complete")
        
    except Exception as e:
        logger.error(f"Error during shutdown: {e}")


# Create FastAPI application
app = FastAPI(
    title=settings.PROJECT_NAME,
    description="""
    ## Device Data Ingestion Service
    
    A high-performance, secure service for ingesting device data from medical devices and wearables.
    
    ### Features

    - **Secure API Key Authentication**: Each device vendor gets unique API keys
    - **Rate Limiting**: Per-vendor and per-device rate limiting to prevent abuse
    - **Transactional Outbox Pattern**: Guaranteed message delivery with fault isolation
    - **Real-time Processing**: Immediate publishing to Kafka for downstream processing
    - **Batch Support**: Efficient batch ingestion for high-volume scenarios
    - **FHIR Compliance**: Data is structured for FHIR Observation resource creation
    - **Cloud-Native Monitoring**: Direct metrics emission to Google Cloud Monitoring
    - **Vendor Isolation**: Per-vendor outbox tables prevent cross-contamination
    
    ### Supported Device Types
    
    - Heart Rate Monitors
    - Blood Pressure Monitors  
    - Blood Glucose Meters
    - Temperature Sensors
    - Pulse Oximeters
    - Smart Scales
    - Activity Trackers
    - Sleep Monitors
    
    ### Authentication
    
    All endpoints require an API key in the `X-API-Key` header:
    
    ```
    X-API-Key: your-vendor-api-key
    ```
    
    ### Rate Limits
    
    - Per vendor: 1000 requests/minute (configurable)
    - Per device: 100 requests/minute
    
    ### Data Flow

    **Standard Flow:**
    ```
    Device → Vendor System → Ingestion Service → Kafka → ETL Pipeline → FHIR Store + Elasticsearch
    ```

    **Transactional Outbox Flow (Recommended):**
    ```
    Device → Vendor System → Ingestion Service → Outbox Table → Publisher Service → Kafka → ETL Pipeline → FHIR Store + Elasticsearch
    ```
    """,
    version=settings.VERSION,
    lifespan=lifespan
)

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # In production, specify allowed origins
    allow_credentials=True,
    allow_methods=["GET", "POST"],
    allow_headers=["*"],
)

# Include API routes
app.include_router(api_router, prefix=settings.API_PREFIX)
app.include_router(resilience_router)


@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": settings.PROJECT_NAME,
        "version": settings.VERSION,
        "status": "running",
        "description": "Device Data Ingestion Service for Clinical Synthesis Hub"
    }


@app.exception_handler(Exception)
async def global_exception_handler(request, exc):
    """Global exception handler"""
    logger.error(f"Unhandled exception: {exc}", exc_info=True)
    
    return JSONResponse(
        status_code=500,
        content={
            "status": "error",
            "message": "Internal server error",
            "service": settings.PROJECT_NAME
        }
    )


if __name__ == "__main__":
    import uvicorn
    
    uvicorn.run(
        "app.main:app",
        host=settings.HOST,
        port=settings.PORT,
        reload=settings.DEBUG,
        log_level="info"
    )
