#!/usr/bin/env python
"""
Setup script for the shared module.
This script copies the shared module from the backend directory to the services directory.
"""

import os
import sys
import shutil
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def setup_shared_module():
    """
    Copy the shared module from the backend directory to the services directory.
    """
    # Get the absolute path to the current directory (fhir-service)
    current_dir = os.path.abspath(os.path.dirname(__file__))
    
    # Get the absolute path to the services directory
    services_dir = os.path.abspath(os.path.join(current_dir, ".."))
    
    # Get the absolute path to the backend directory
    backend_dir = os.path.abspath(os.path.join(services_dir, ".."))
    
    # Get the absolute path to the backend shared directory
    backend_shared_dir = os.path.join(backend_dir, "shared")
    
    # Get the absolute path to the services shared directory
    services_shared_dir = os.path.join(services_dir, "shared")
    
    # Print paths for debugging
    logger.info(f"Current directory: {current_dir}")
    logger.info(f"Services directory: {services_dir}")
    logger.info(f"Backend directory: {backend_dir}")
    logger.info(f"Backend shared directory: {backend_shared_dir}")
    logger.info(f"Services shared directory: {services_shared_dir}")
    
    # Check if the backend shared directory exists
    if not os.path.exists(backend_shared_dir):
        logger.error(f"Backend shared directory does not exist: {backend_shared_dir}")
        return False
    
    # Check if the services shared directory exists
    if os.path.exists(services_shared_dir):
        logger.info(f"Services shared directory already exists: {services_shared_dir}")
        logger.info(f"Removing existing services shared directory...")
        shutil.rmtree(services_shared_dir)
    
    # Copy the shared directory from backend to services
    logger.info(f"Copying shared directory from {backend_shared_dir} to {services_shared_dir}...")
    shutil.copytree(backend_shared_dir, services_shared_dir)
    
    # Create __init__.py file in the services shared directory if it doesn't exist
    init_file = os.path.join(services_shared_dir, "__init__.py")
    if not os.path.exists(init_file):
        logger.info(f"Creating __init__.py file in {services_shared_dir}...")
        with open(init_file, "w") as f:
            f.write("# Shared module for Clinical Synthesis Hub microservices\n")
    
    # Create __init__.py file in the services shared auth directory if it doesn't exist
    auth_dir = os.path.join(services_shared_dir, "auth")
    if os.path.exists(auth_dir):
        init_file = os.path.join(auth_dir, "__init__.py")
        if not os.path.exists(init_file):
            logger.info(f"Creating __init__.py file in {auth_dir}...")
            with open(init_file, "w") as f:
                f.write("# Authentication module for Clinical Synthesis Hub microservices\n")
    
    logger.info(f"Shared module setup complete!")
    return True

if __name__ == "__main__":
    setup_shared_module()
