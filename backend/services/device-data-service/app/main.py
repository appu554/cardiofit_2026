"""
Device Data Service - FastAPI Application

Provides GraphQL API for querying processed device data from Elasticsearch and FHIR Store.
This is the read-side service for the event-driven architecture.
"""
import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from strawberry.fastapi import GraphQLRouter

from .config import settings
from .graphql.federation_schema import schema
from .services.device_data_service import get_device_data_service

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """Application lifespan manager"""
    # Startup
    logger.info("Starting Device Data Service...")
    
    try:
        # Initialize device data service
        service = get_device_data_service()
        logger.info("Device data service initialized")
        
        logger.info("Device Data Service started successfully")
        
    except Exception as e:
        logger.error(f"Failed to start service: {e}")
        raise
    
    yield
    
    # Shutdown
    logger.info("Shutting down Device Data Service...")
    logger.info("Device Data Service shutdown complete")


# Create FastAPI application
app = FastAPI(
    title=settings.PROJECT_NAME,
    description="""
    ## Device Data Service
    
    GraphQL API for querying processed device data from medical devices and wearables.
    
    ### Features
    
    - **Real-time Device Data**: Query the latest device readings
    - **Patient-centric Views**: Get all device data for a specific patient
    - **Device-centric Views**: Get all readings from a specific device
    - **Advanced Filtering**: Filter by reading type, alert level, date ranges
    - **Statistics & Analytics**: Get aggregated statistics and trends
    - **Apollo Federation**: Extends Patient and Device types across the graph
    
    ### Data Sources
    
    - **Elasticsearch**: Fast queries on processed device data
    - **Google Healthcare API**: FHIR-compliant device observations
    
    ### GraphQL Endpoints
    
    - `/api/federation` - Apollo Federation endpoint for gateway integration
    - `/api/graphql` - Direct GraphQL endpoint for development
    
    ### Example Queries
    
    ```graphql
    # Get patient's device readings
    query GetPatientReadings($patientId: ID!) {
      patient(id: $patientId) {
        deviceReadings(limit: 10) {
          items {
            readingType
            readingValue
            readingUnit
            alertLevel
            readingDatetime
          }
        }
      }
    }
    
    # Get critical readings
    query GetCriticalReadings {
      criticalReadings(hours: 24) {
        deviceId
        patientId
        readingType
        readingValue
        alertLevel
        readingDatetime
      }
    }
    
    # Get reading statistics
    query GetReadingStats($patientId: ID!) {
      readingStats(patientId: $patientId) {
        totalReadings
        readingsByType {
          readingType
          count
        }
        readingsByAlertLevel {
          alertLevel
          count
        }
      }
    }
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

# Create GraphQL routers
graphql_router = GraphQLRouter(schema)

# Include GraphQL routes
app.include_router(graphql_router, prefix="/api/graphql")
app.include_router(graphql_router, prefix="/api/federation")


@app.get("/")
async def root():
    """Root endpoint"""
    return {
        "service": settings.PROJECT_NAME,
        "version": settings.VERSION,
        "status": "running",
        "description": "Device Data Service for Clinical Synthesis Hub",
        "endpoints": {
            "graphql": "/api/graphql",
            "federation": "/api/federation",
            "health": "/health"
        }
    }


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    try:
        # Check service dependencies
        service = get_device_data_service()
        
        health_status = {
            "status": "healthy",
            "service": settings.PROJECT_NAME,
            "version": settings.VERSION,
            "dependencies": {
                "elasticsearch": "unknown",  # Would check ES connection
                "fhir_store": "unknown"      # Would check FHIR connection
            }
        }
        
        return health_status
        
    except Exception as e:
        logger.error(f"Health check failed: {e}")
        return {
            "status": "unhealthy",
            "service": settings.PROJECT_NAME,
            "error": str(e)
        }


if __name__ == "__main__":
    import uvicorn
    
    uvicorn.run(
        "app.main:app",
        host=settings.HOST,
        port=settings.PORT,
        reload=settings.DEBUG,
        log_level="info"
    )
