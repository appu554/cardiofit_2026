"""
FHIR Service Factory for Encounter Management Service.

This module provides a factory function to initialize the appropriate FHIR service
based on configuration settings.
"""

import logging
from typing import Optional
from app.core.config import settings
from .google_fhir_service import EncounterFHIRService

# Configure logging
logger = logging.getLogger(__name__)

# Global FHIR service instance
_fhir_service: Optional[EncounterFHIRService] = None

async def initialize_fhir_service() -> EncounterFHIRService:
    """
    Initialize and return the FHIR service instance.
    
    This function creates a singleton instance of the FHIR service
    based on the configuration settings.
    
    Returns:
        EncounterFHIRService: The initialized FHIR service instance
    """
    global _fhir_service
    
    if _fhir_service is None:
        logger.info("Initializing FHIR service for Encounter Management...")
        
        # Always use Google Healthcare API for this service
        _fhir_service = EncounterFHIRService()
        
        # Initialize the service
        success = await _fhir_service.initialize()
        
        if success:
            logger.info("FHIR service initialized successfully with Google Healthcare API")
        else:
            logger.error("Failed to initialize FHIR service")
            raise RuntimeError("Failed to initialize FHIR service")
    
    return _fhir_service

async def get_fhir_service() -> Optional[EncounterFHIRService]:
    """
    Get the current FHIR service instance.
    
    Returns:
        Optional[EncounterFHIRService]: The FHIR service instance or None if not initialized
    """
    global _fhir_service
    
    if _fhir_service is None:
        logger.warning("FHIR service not initialized. Call initialize_fhir_service() first.")
        return None
    
    return _fhir_service

def reset_fhir_service():
    """
    Reset the FHIR service instance.
    
    This is primarily used for testing purposes.
    """
    global _fhir_service
    _fhir_service = None
    logger.info("FHIR service instance reset")
