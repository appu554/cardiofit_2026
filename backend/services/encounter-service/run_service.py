#!/usr/bin/env python
"""
Run script for Encounter Management Service.
This script adds the backend directory to the Python path and starts the service.
"""

import sys
import os
import subprocess
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def main():
    # Add the backend directory to the Python path
    # Need to go up two levels: encounter-service -> services -> backend
    backend_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), "../.."))
    sys.path.insert(0, backend_dir)

    # Print the backend directory for debugging
    logger.info(f"Backend directory: {backend_dir}")
    logger.info(f"Checking if shared module exists: {os.path.exists(os.path.join(backend_dir, 'shared'))}")

    if os.path.exists(os.path.join(backend_dir, 'shared')):
        logger.info("Contents of shared directory:")
        for item in os.listdir(os.path.join(backend_dir, 'shared')):
            logger.info(f"  {item}")

    # Set environment variables
    os.environ["AUTH_SERVICE_URL"] = "http://localhost:8001/api"
    os.environ["PYTHONPATH"] = backend_dir

    # Google Healthcare API environment variables
    os.environ["GOOGLE_CLOUD_PROJECT"] = "cardiofit-905a8"
    os.environ["GOOGLE_CLOUD_LOCATION"] = "asia-south1"
    os.environ["GOOGLE_CLOUD_DATASET"] = "clinical-synthesis-hub"
    os.environ["GOOGLE_CLOUD_FHIR_STORE"] = "fhir-store"
    os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = "credentials/google-credentials.json"

    # Verify shared module is importable
    try:
        from shared.auto_import import ensure_shared_importable
        ensure_shared_importable()
        logger.info("[OK] Successfully imported shared module")

        # Verify HeaderAuthMiddleware is importable
        from shared.auth import HeaderAuthMiddleware
        logger.info("✓ Successfully imported HeaderAuthMiddleware")
    except ImportError as e:
        logger.warning(f"Could not import shared module: {e}")
        logger.warning("This might cause problems when importing HeaderAuthMiddleware")
        logger.warning(f"Make sure the shared module exists at: {os.path.join(backend_dir, 'shared')}")

    # Print configuration
    logger.info("Starting Encounter Management Service with the following configuration:")
    logger.info(f"  Python Path: {sys.path[0]}")
    logger.info(f"  PYTHONPATH: {os.environ['PYTHONPATH']}")
    logger.info(f"  AUTH_SERVICE_URL: {os.environ['AUTH_SERVICE_URL']}")
    logger.info(f"  Google Cloud Project: {os.environ['GOOGLE_CLOUD_PROJECT']}")
    logger.info(f"  FHIR Store: {os.environ['GOOGLE_CLOUD_DATASET']}/{os.environ['GOOGLE_CLOUD_FHIR_STORE']}")
    logger.info("")

    # Get current directory for uvicorn
    current_dir = os.path.dirname(os.path.abspath(__file__))

    # Start the service
    try:
        port = int(os.environ.get("ENCOUNTER_SERVICE_PORT", "8020"))
        host = os.environ.get("ENCOUNTER_SERVICE_HOST", "0.0.0.0")
        debug = os.environ.get("DEBUG", "true").lower() == "true"

        cmd = [
            sys.executable, "-m", "uvicorn",
            "app.main:app",
            "--host", host,
            "--port", str(port),
            "--reload" if debug else "--no-reload",
            "--log-level", "info"
        ]

        logger.info(f"Starting server with command: {' '.join(cmd)}")
        subprocess.run(cmd, cwd=current_dir)

    except KeyboardInterrupt:
        logger.info("Service stopped by user")
    except Exception as e:
        logger.error(f"Error starting service: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
