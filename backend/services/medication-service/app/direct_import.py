"""
Direct import module for the HeaderAuthMiddleware.
This module provides a direct way to import the HeaderAuthMiddleware
without relying on the Python path.
"""

import os
import sys
import importlib.util
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def get_header_auth_middleware():
    """
    Get the HeaderAuthMiddleware class using a direct import approach.
    
    Returns:
        The HeaderAuthMiddleware class
    """
    # Get the absolute path to the services directory
    # Need to go up two levels: app -> medication-service -> services
    services_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
    logger.info(f"Services directory: {services_dir}")
    
    # Get the absolute path to the services/shared/auth directory
    services_shared_auth_dir = os.path.join(services_dir, "shared", "auth")
    logger.info(f"Services shared auth directory: {services_shared_auth_dir}")
    
    # Get the absolute path to the header_middleware.py file in services/shared/auth
    header_middleware_path = os.path.join(services_shared_auth_dir, "header_middleware.py")
    logger.info(f"Looking for header_middleware.py at: {header_middleware_path}")
    
    # Check if the file exists
    if not os.path.exists(header_middleware_path):
        logger.error(f"Could not find header_middleware.py at {header_middleware_path}")
        
        # Check if the services/shared directory exists
        services_shared_dir = os.path.join(services_dir, "shared")
        if os.path.exists(services_shared_dir):
            logger.info(f"Contents of {services_shared_dir}:")
            for item in os.listdir(services_shared_dir):
                logger.info(f"  {item}")
            
            # Check if services/shared/auth directory exists
            if os.path.exists(services_shared_auth_dir):
                logger.info(f"Contents of {services_shared_auth_dir}:")
                for item in os.listdir(services_shared_auth_dir):
                    logger.info(f"  {item}")
            else:
                logger.error(f"Auth directory does not exist: {services_shared_auth_dir}")
        else:
            logger.error(f"Shared directory does not exist: {services_shared_dir}")
        
        # Try to import using the normal Python import mechanism as a fallback
        try:
            # Add services directory to Python path if not already there
            if services_dir not in sys.path:
                sys.path.insert(0, services_dir)
                logger.info(f"Added {services_dir} to Python path")
            
            # Try to import from services/shared/auth
            from shared.auth import HeaderAuthMiddleware
            logger.info("Successfully imported HeaderAuthMiddleware using normal import")
            return HeaderAuthMiddleware
        except ImportError as e:
            logger.error(f"Normal import failed: {e}")
            
            # Try to import from backend/shared/auth as a last resort
            try:
                # Get the absolute path to the backend directory
                # Need to go up three levels: app -> medication-service -> services -> backend
                backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../.."))
                logger.info(f"Backend directory: {backend_dir}")
                
                # Add backend directory to Python path if not already there
                if backend_dir not in sys.path:
                    sys.path.insert(0, backend_dir)
                    logger.info(f"Added {backend_dir} to Python path")
                
                # Try to import from backend/shared/auth
                from shared.auth import HeaderAuthMiddleware
                logger.info("Successfully imported HeaderAuthMiddleware from backend/shared/auth")
                return HeaderAuthMiddleware
            except ImportError as e2:
                logger.error(f"Backend import failed: {e2}")
                raise ImportError(f"Could not find header_middleware.py at {header_middleware_path}")
    
    logger.info(f"Found header_middleware.py at {header_middleware_path}")
    
    # Import the module using importlib
    try:
        spec = importlib.util.spec_from_file_location("header_middleware", header_middleware_path)
        header_middleware = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(header_middleware)
        
        # Return the HeaderAuthMiddleware class
        logger.info("Successfully imported HeaderAuthMiddleware using direct file import")
        return header_middleware.HeaderAuthMiddleware
    except Exception as e:
        logger.error(f"Error importing module using importlib: {e}")
        raise

# Try to get the HeaderAuthMiddleware class
try:
    logger.info("Attempting to get HeaderAuthMiddleware...")
    HeaderAuthMiddleware = get_header_auth_middleware()
    logger.info(f"Successfully got HeaderAuthMiddleware: {HeaderAuthMiddleware}")
except Exception as e:
    logger.error(f"Error importing HeaderAuthMiddleware: {e}")
    
    # We need to raise the exception to ensure proper authentication
    logger.error("Failed to import HeaderAuthMiddleware. This is a critical error.")
    logger.error("Please make sure the shared module is properly installed.")
    logger.error("You can run setup_shared_links.py in the backend directory to set up the shared module.")
    raise
