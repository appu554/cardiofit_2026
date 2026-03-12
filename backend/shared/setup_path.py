"""
Script to set up Python path to include the shared module.

Usage:
    import sys
    import os

    # Get the absolute path to the backend directory
    backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), '..'))
    
    # Add the backend directory to Python path
    if backend_dir not in sys.path:
        sys.path.insert(0, backend_dir)
"""

import sys
import os
import logging

logger = logging.getLogger(__name__)

def setup_shared_path():
    """
    Add the backend directory to the Python path.
    This allows importing the shared module from any service.
    """
    # Get the absolute path to the backend directory
    backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), '..'))
    
    # Add the backend directory to Python path
    if backend_dir not in sys.path:
        sys.path.insert(0, backend_dir)
        logger.info(f"Added {backend_dir} to Python path")
    else:
        logger.info(f"{backend_dir} already in Python path")
    
    return backend_dir

if __name__ == "__main__":
    # Configure logging
    logging.basicConfig(level=logging.INFO)
    
    # Setup shared path
    backend_dir = setup_shared_path()
    
    # Print current Python path
    logger.info("Current Python path:")
    for path in sys.path:
        logger.info(f"  {path}")