"""
Auto-import module for the shared package.

This module automatically adjusts the Python path to ensure the shared module
can be imported from any service without "No module named 'shared'" errors.

Usage:
    import sys, os; sys.path.insert(0, os.path.abspath(os.path.join(os.path.dirname(__file__), '..')))
    from shared.auto_import import ensure_shared_importable
    ensure_shared_importable()
    
    # Now you can import any shared module
    from shared.auth import HeaderAuthMiddleware
"""

import sys
import os
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def ensure_shared_importable():
    """
    Ensures the shared module is importable by adding the backend directory to Python path.
    
    This function should be called before importing any shared modules to ensure they can be found.
    """
    # Get the absolute path to the backend directory (parent of shared)
    backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), '..'))
    
    # Add the backend directory to Python path if not already present
    if backend_dir not in sys.path:
        sys.path.insert(0, backend_dir)
        logger.info(f"Added {backend_dir} to Python path")
    
    return True

def fix_shared_imports():
    """
    Fix shared imports by adding backend directory to Python path and returning
    the HeaderAuthMiddleware class.
    
    Returns:
        HeaderAuthMiddleware class
    """
    ensure_shared_importable()
    
    # Now we can safely import the HeaderAuthMiddleware
    try:
        from shared.auth import HeaderAuthMiddleware
        return HeaderAuthMiddleware
    except ImportError as e:
        logger.error(f"Failed to import HeaderAuthMiddleware: {e}")
        logger.error("Please make sure the shared module is properly installed")
        raise

# Auto-run the ensure_shared_importable function when the module is imported
ensure_shared_importable()

if __name__ == "__main__":
    # If this script is run directly, print the Python path
    logger.info("Current Python path:")
    for path in sys.path:
        logger.info(f"  {path}")
    
    # Try importing HeaderAuthMiddleware to verify it works
    try:
        from shared.auth import HeaderAuthMiddleware
        logger.info(f"Successfully imported HeaderAuthMiddleware: {HeaderAuthMiddleware}")
    except ImportError as e:
        logger.error(f"Failed to import HeaderAuthMiddleware: {e}")