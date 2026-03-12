#!/usr/bin/env python
"""
Setup FHIR Router Script

This script helps set up the shared FHIR router in a microservice.
It creates the necessary files and updates the API router.
"""

import os
import sys
import shutil
import argparse
import logging

# Configure logging
logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

def setup_fhir_router(service_dir: str, resource_type: str):
    """
    Set up the shared FHIR router in a microservice.
    
    Args:
        service_dir: The directory of the microservice
        resource_type: The FHIR resource type (e.g., "Patient", "Condition")
    """
    try:
        # Get the absolute path to the service directory
        service_dir = os.path.abspath(service_dir)
        
        # Check if the service directory exists
        if not os.path.exists(service_dir):
            logger.error(f"Service directory {service_dir} does not exist")
            return False
        
        # Get the absolute path to the shared FHIR module
        shared_dir = os.path.dirname(os.path.abspath(__file__))
        
        # Create the FHIR service file
        fhir_service_template = os.path.join(shared_dir, "service_template.py")
        fhir_service_dir = os.path.join(service_dir, "app", "services")
        fhir_service_file = os.path.join(fhir_service_dir, "fhir_service.py")
        
        # Create the services directory if it doesn't exist
        os.makedirs(fhir_service_dir, exist_ok=True)
        
        # Copy the FHIR service template
        logger.info(f"Creating FHIR service file at {fhir_service_file}")
        shutil.copy(fhir_service_template, fhir_service_file)
        
        # Replace the placeholder resource type in the FHIR service file
        with open(fhir_service_file, "r") as f:
            content = f.read()
        
        content = content.replace("YourResource", resource_type)
        content = content.replace("yourresource", resource_type.lower())
        content = content.replace("YOURRESOURCE", resource_type.upper())
        
        with open(fhir_service_file, "w") as f:
            f.write(content)
        
        # Create the FHIR router file
        fhir_router_template = os.path.join(shared_dir, "template.py")
        fhir_router_dir = os.path.join(service_dir, "app", "api", "endpoints")
        fhir_router_file = os.path.join(fhir_router_dir, "fhir.py")
        
        # Create the endpoints directory if it doesn't exist
        os.makedirs(fhir_router_dir, exist_ok=True)
        
        # Copy the FHIR router template
        logger.info(f"Creating FHIR router file at {fhir_router_file}")
        shutil.copy(fhir_router_template, fhir_router_file)
        
        # Replace the placeholder resource type in the FHIR router file
        with open(fhir_router_file, "r") as f:
            content = f.read()
        
        content = content.replace("YourResource", resource_type)
        content = content.replace("# from app.services.fhir_service import YourResourceFHIRService", f"from app.services.fhir_service import {resource_type}FHIRService")
        content = content.replace("service_class=MockFHIRService", f"service_class={resource_type}FHIRService")
        content = content.replace("# get_token_payload=get_token_payload", "get_token_payload=get_token_payload")
        content = content.replace("# from app.core.auth import get_token_payload", "from app.core.auth import get_token_payload")
        
        with open(fhir_router_file, "w") as f:
            f.write(content)
        
        # Check if the API router file exists
        api_router_file = os.path.join(service_dir, "app", "api", "api.py")
        if not os.path.exists(api_router_file):
            logger.warning(f"API router file {api_router_file} does not exist")
            logger.warning("You will need to manually include the FHIR router in your API router")
        else:
            # Check if the FHIR router is already included in the API router
            with open(api_router_file, "r") as f:
                content = f.read()
            
            if "api_router.include_router(fhir.router, prefix=\"/fhir\"" not in content:
                logger.warning("FHIR router is not included in the API router")
                logger.warning("You may need to manually include the FHIR router in your API router")
                logger.warning("Add the following line to your API router file:")
                logger.warning("api_router.include_router(fhir.router, prefix=\"/fhir\", tags=[\"FHIR\"])")
            else:
                logger.info("FHIR router is already included in the API router")
        
        logger.info(f"Successfully set up FHIR router for {resource_type} in {service_dir}")
        return True
    except Exception as e:
        logger.error(f"Error setting up FHIR router: {str(e)}")
        return False

def main():
    """Main function."""
    parser = argparse.ArgumentParser(description="Set up the shared FHIR router in a microservice")
    parser.add_argument("service_dir", help="The directory of the microservice")
    parser.add_argument("resource_type", help="The FHIR resource type (e.g., 'Patient', 'Condition')")
    
    args = parser.parse_args()
    
    setup_fhir_router(args.service_dir, args.resource_type)

if __name__ == "__main__":
    main()
