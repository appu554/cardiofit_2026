"""
FHIR Service Factory for Order Management Service

This module provides factory functions to initialize FHIR services
for Google Healthcare API integration.
"""

import logging
import os
import sys
from typing import Optional

# Add backend directory to path for shared imports
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)

logger = logging.getLogger(__name__)

# Global FHIR service instance
_fhir_service = None

async def initialize_fhir_service():
    """
    Initialize the FHIR service for order management.

    Returns:
        FHIR service instance for Google Healthcare API
    """
    global _fhir_service

    if _fhir_service is not None:
        return _fhir_service

    try:
        from .google_fhir_service import OrderManagementFHIRService

        _fhir_service = OrderManagementFHIRService()
        success = await _fhir_service.initialize()

        if success:
            logger.info("Order Management FHIR service initialized successfully")
            return _fhir_service
        else:
            logger.error("Failed to initialize Order Management FHIR service")
            raise Exception("FHIR service initialization failed")

    except ImportError as e:
        logger.error(f"Failed to import FHIR service: {e}")
        # Return a placeholder service for development
        logger.warning("Using placeholder FHIR service for development")
        return "PlaceholderFHIRService"
    except Exception as e:
        logger.error(f"Failed to initialize FHIR service: {e}")
        raise

def get_fhir_service():
    """
    Get the current FHIR service instance.

    Returns:
        Current FHIR service instance or None if not initialized
    """
    return _fhir_service
