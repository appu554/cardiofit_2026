"""
FHIR Service Factory for Scheduling Service.

This module provides a factory function to initialize the appropriate FHIR service
based on configuration settings.
"""

import logging
from typing import Optional
from app.services.google_fhir_service import SchedulingFHIRService
from app.core.config import settings

logger = logging.getLogger(__name__)

# Global FHIR service instance
_fhir_service: Optional[SchedulingFHIRService] = None

async def initialize_fhir_service() -> SchedulingFHIRService:
    """
    Initialize and return the FHIR service instance.
    
    Returns:
        SchedulingFHIRService: The initialized FHIR service
    """
    global _fhir_service
    
    if _fhir_service is None:
        logger.info("Initializing Google Healthcare API FHIR service for scheduling")
        _fhir_service = SchedulingFHIRService()
        
        # Initialize the service
        success = await _fhir_service.initialize()
        if not success:
            logger.error("Failed to initialize FHIR service")
            raise RuntimeError("Failed to initialize FHIR service")
        
        logger.info("FHIR service initialized successfully")
    
    return _fhir_service

def get_fhir_service() -> Optional[SchedulingFHIRService]:
    """
    Get the current FHIR service instance.
    
    Returns:
        SchedulingFHIRService or None: The FHIR service instance if initialized
    """
    return _fhir_service
