# Authentication package for Clinical Synthesis Hub
# This package provides authentication utilities for all microservices

import logging
import sys
import os

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Attempt to import middleware modules
try:
    from .middleware import AuthenticationMiddleware, get_current_user, get_token_payload, get_current_user_from_token
    from .header_middleware import HeaderAuthMiddleware
except ImportError as e:
    # If direct import fails, it might be due to Python path issues
    logger.warning(f"Import error: {e}")
    logger.warning("Attempting to fix Python path for shared module...")
    
    # Add backend directory to Python path if needed
    backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
    if backend_dir not in sys.path:
        sys.path.insert(0, backend_dir)
        logger.info(f"Added {backend_dir} to Python path")
    
    # Try import again with corrected path
    try:
        from shared.auth.middleware import AuthenticationMiddleware, get_current_user, get_token_payload, get_current_user_from_token
        from shared.auth.header_middleware import HeaderAuthMiddleware
        logger.info("Successfully imported authentication modules after fixing Python path")
    except ImportError as e2:
        logger.error(f"Failed to import authentication modules even after fixing Python path: {e2}")
        logger.error("Please check your project structure and make sure shared module is available")
        # Re-raise the error to make it clear something is wrong
        raise

__all__ = ["AuthenticationMiddleware", "HeaderAuthMiddleware", "get_current_user", "get_token_payload", "get_current_user_from_token"]