#!/usr/bin/env python
"""
Setup All FHIR Routers Script

This script helps set up the shared FHIR router in all microservices.
It creates the necessary files and updates the API routers.
"""

import os
import sys
import logging
from setup_fhir_router import setup_fhir_router

# Configure logging
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

# Define the microservices and their resource types
MICROSERVICES = {
    "patient-service": "Patient",
    "condition-service": "Condition",
    "observation-service": "Observation",
    "medication-service": "Medication",
    "encounter-service": "Encounter",
    "lab-service": "DiagnosticReport",
    "notes-service": "DocumentReference"
}

def setup_all_fhir_routers(services_dir: str):
    """
    Set up the shared FHIR router in all microservices.
    
    Args:
        services_dir: The directory containing all microservices
    """
    try:
        # Get the absolute path to the services directory
        services_dir = os.path.abspath(services_dir)
        
        # Check if the services directory exists
        if not os.path.exists(services_dir):
            logger.error(f"Services directory {services_dir} does not exist")
            return False
        
        # Set up the FHIR router in each microservice
        for service_name, resource_type in MICROSERVICES.items():
            service_dir = os.path.join(services_dir, service_name)
            
            # Check if the service directory exists
            if not os.path.exists(service_dir):
                logger.warning(f"Service directory {service_dir} does not exist, skipping")
                continue
            
            logger.info(f"Setting up FHIR router for {resource_type} in {service_dir}")
            setup_fhir_router(service_dir, resource_type)
        
        logger.info("Successfully set up FHIR routers in all microservices")
        return True
    except Exception as e:
        logger.error(f"Error setting up FHIR routers: {str(e)}")
        return False

def main():
    """Main function."""
    # Get the services directory from the command line or use the default
    if len(sys.argv) > 1:
        services_dir = sys.argv[1]
    else:
        # Get the directory of this script
        script_dir = os.path.dirname(os.path.abspath(__file__))
        
        # Get the services directory (parent of the shared directory)
        services_dir = os.path.dirname(os.path.dirname(script_dir))
    
    logger.info(f"Using services directory: {services_dir}")
    setup_all_fhir_routers(services_dir)

if __name__ == "__main__":
    main()
