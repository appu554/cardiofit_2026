"""
FHIR Service Factory for Patient resources.

This module provides a factory for creating FHIR service instances based on configuration.
It supports both MongoDB and Google Cloud Healthcare API implementations.
"""

import logging
from typing import Any

# Import settings
from app.core.config import settings

# Import FHIR service implementations
from app.services.fhir_service import PatientFHIRService
from app.services.google_fhir_service import GooglePatientFHIRService

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Global instance of the FHIR service
_patient_fhir_service = None

async def initialize_fhir_service():
    """
    Initialize the FHIR service based on configuration.

    This function creates a global instance of the appropriate FHIR service
    implementation based on the configuration settings.

    Returns:
        The initialized FHIR service instance
    """
    global _patient_fhir_service

    if _patient_fhir_service is None:
        # Create the appropriate FHIR service based on configuration
        if settings.USE_GOOGLE_HEALTHCARE_API:
            logger.info("Creating new GooglePatientFHIRService instance...")
            _patient_fhir_service = GooglePatientFHIRService()
        else:
            logger.info("Creating new PatientFHIRService instance (MongoDB)...")
            _patient_fhir_service = PatientFHIRService()

    # Initialize the service
    await _patient_fhir_service.initialize()
    
    logger.info(f"FHIR service initialized: {type(_patient_fhir_service).__name__}")
    return _patient_fhir_service

def get_fhir_service():
    """
    Get the global FHIR service instance.

    Returns:
        The global FHIR service instance
    """
    global _patient_fhir_service

    if _patient_fhir_service is None:
        logger.warning("FHIR service not initialized yet. Creating a new instance.")
        
        # Create the appropriate FHIR service based on configuration
        if settings.USE_GOOGLE_HEALTHCARE_API:
            logger.info("Creating new GooglePatientFHIRService instance...")
            _patient_fhir_service = GooglePatientFHIRService()
        else:
            logger.info("Creating new PatientFHIRService instance (MongoDB)...")
            _patient_fhir_service = PatientFHIRService()

    return _patient_fhir_service
