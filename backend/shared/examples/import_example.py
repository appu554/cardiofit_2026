"""
Example showing how to import the shared module.
"""

import sys
import os
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Method 1: Add the backend directory to Python path
backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), '..', '..'))
if backend_dir not in sys.path:
    sys.path.insert(0, backend_dir)
    logger.info(f"Added {backend_dir} to Python path")

# Now we can import from the shared module
from shared.auth.header_middleware import HeaderAuthMiddleware

def create_app():
    """
    Example function to create a FastAPI app with HeaderAuthMiddleware.
    
    Returns:
        A FastAPI app with HeaderAuthMiddleware
    """
    from fastapi import FastAPI
    
    # Create the app
    app = FastAPI()
    
    # Add the middleware
    app.add_middleware(
        HeaderAuthMiddleware,
        exclude_paths=["/docs", "/openapi.json", "/health", "/metrics"]
    )
    
    # Add some routes
    @app.get("/")
    async def root():
        return {"message": "Hello World"}
    
    @app.get("/health")
    async def health():
        return {"status": "ok"}
    
    return app

if __name__ == "__main__":
    # Test that we can import the middleware
    logger.info(f"Successfully imported HeaderAuthMiddleware: {HeaderAuthMiddleware}")
    
    # Create an app
    app = create_app()
    logger.info(f"Created app with middleware: {app}")