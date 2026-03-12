from fastapi import APIRouter
from app.api.endpoints import timeline, fhir
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

api_router = APIRouter()

# Include the Timeline router
api_router.include_router(timeline.router, prefix="/timeline", tags=["Timeline"])

# Include the FHIR router
api_router.include_router(fhir.router, prefix="/fhir", tags=["FHIR"])

# Log the routers
logger.info("=== TIMELINE SERVICE API ROUTERS ===")
logger.info("Timeline router: /api/timeline")
logger.info("FHIR router: /api/fhir")
logger.info("=== END TIMELINE SERVICE API ROUTERS ===")
