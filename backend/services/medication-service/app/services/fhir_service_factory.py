"""
FHIR Service Factory for Medication resources.

This module provides a factory for creating FHIR service instances based on configuration.
It supports Google Cloud Healthcare API implementation for medication resources.
"""

import logging
from typing import Any

# Import settings
from app.core.config import settings

# Import FHIR service implementations
from app.services.google_fhir_service import GoogleMedicationFHIRService

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Global instance of the FHIR service
_medication_fhir_service = None

async def initialize_fhir_service():
    """
    Initialize the FHIR service based on configuration.

    This function creates a global instance of the appropriate FHIR service
    implementation based on the configuration settings.

    Returns:
        The initialized FHIR service instance
    """
    global _medication_fhir_service

    if _medication_fhir_service is None:
        # Create the Google Healthcare API FHIR service
        logger.info("Creating new GoogleMedicationFHIRService instance...")
        _medication_fhir_service = GoogleMedicationFHIRService()

    # Initialize the service
    await _medication_fhir_service.initialize()
    
    logger.info(f"FHIR service initialized: {type(_medication_fhir_service).__name__}")
    return _medication_fhir_service

def get_fhir_service():
    """
    Get the global FHIR service instance.

    Returns:
        The global FHIR service instance
    """
    global _medication_fhir_service

    if _medication_fhir_service is None:
        logger.warning("FHIR service not initialized yet. Creating a new instance.")
        
        # Create the Google Healthcare API FHIR service
        logger.info("Creating new GoogleMedicationFHIRService instance...")
        _medication_fhir_service = GoogleMedicationFHIRService()

    return _medication_fhir_service
