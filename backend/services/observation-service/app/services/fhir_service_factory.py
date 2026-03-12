"""
FHIR Service Factory for Observation resources.

This module provides a factory for creating FHIR service instances based on configuration.
It supports both MongoDB and Google Cloud Healthcare API implementations.
"""

import logging
from typing import Any

# Import settings
from app.core.config import settings

# Import FHIR service implementations
from app.services.fhir_service import ObservationFHIRService # This is our Google Cloud enabled service
# GOOGLE_FHIR_AVAILABLE flag is no longer needed as ObservationFHIRService handles Google Cloud integration directly.

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Global instance of the FHIR service
_observation_fhir_service = None

async def initialize_fhir_service():
    """
    Initialize the FHIR service based on configuration.

    This function creates a global instance of the appropriate FHIR service
    implementation based on the configuration settings.

    Returns:
        The initialized FHIR service instance
    """
    global _observation_fhir_service

    if _observation_fhir_service is None:
        # Create the FHIR service instance
        if settings.USE_GOOGLE_HEALTHCARE_API:
            logger.info("USE_GOOGLE_HEALTHCARE_API is True. Creating ObservationFHIRService (intended for Google Cloud)...")
            _observation_fhir_service = ObservationFHIRService()
        else:
            logger.error(
                "USE_GOOGLE_HEALTHCARE_API is False. "
                "No alternative FHIR service backend is currently configured in the factory. "
                "ObservationFHIRService is designed for Google Cloud integration."
            )
            raise RuntimeError(
                "FHIR Service Configuration Error: USE_GOOGLE_HEALTHCARE_API is set to False, "
                "and no alternative backend (e.g., mock or in-memory) is configured in the factory. "
                "To use Google Cloud, set USE_GOOGLE_HEALTHCARE_API to True."
            )

    # Initialize the service if it was created
    if _observation_fhir_service:
        await _observation_fhir_service.initialize()
    else:
        # This case should ideally not be reached if the above logic correctly instantiates or raises an error.
        logger.error("FHIR service instance was not created prior to initialization step.")
        raise RuntimeError("Failed to create FHIR service instance.")
    
    logger.info(f"FHIR service initialized: {type(_observation_fhir_service).__name__}")
    return _observation_fhir_service


def get_fhir_service():
    """
    Get the FHIR service instance.

    This function returns the global FHIR service instance, initializing it if necessary.

    Returns:
        The FHIR service instance
    """
    global _observation_fhir_service
    
    if _observation_fhir_service is None:
        raise RuntimeError("FHIR service has not been initialized. Call initialize_fhir_service() first.")
        
    return _observation_fhir_service
