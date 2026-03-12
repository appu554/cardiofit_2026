#!/usr/bin/env python3
"""
Order Management Service Runner

This script sets up the environment and runs the Order Management Service
following the established microservice pattern.
"""

import os
import sys
import subprocess
import logging
from pathlib import Path

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def setup_environment():
    """Set up environment variables for the service"""
    
    # Get the current directory (order-management-service)
    current_dir = Path(__file__).parent.absolute()
    
    # Get the backend directory (2 levels up: order-management-service -> services -> backend)
    backend_dir = current_dir.parent.parent
    
    # Set up Python path to include backend directory for shared modules
    python_path = str(backend_dir)
    if "PYTHONPATH" in os.environ:
        python_path = f"{python_path}{os.pathsep}{os.environ['PYTHONPATH']}"
    os.environ["PYTHONPATH"] = python_path
    
    # Set up Google Healthcare API environment variables
    os.environ["USE_GOOGLE_HEALTHCARE_API"] = "true"
    os.environ["GOOGLE_CLOUD_PROJECT"] = "cardiofit-905a8"
    os.environ["GOOGLE_CLOUD_LOCATION"] = "asia-south1"
    os.environ["GOOGLE_CLOUD_DATASET"] = "clinical-synthesis-hub"
    os.environ["GOOGLE_CLOUD_FHIR_STORE"] = "fhir-store"
    
    # Set credentials path
    credentials_path = current_dir / "credentials" / "google-credentials.json"
    if credentials_path.exists():
        os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = str(credentials_path)
        logger.info(f"Using Google credentials from: {credentials_path}")
    else:
        # Try service account key as fallback
        fallback_path = current_dir / "credentials" / "service-account-key.json"
        if fallback_path.exists():
            os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = str(fallback_path)
            logger.info(f"Using Google credentials from: {fallback_path}")
        else:
            logger.warning("No Google credentials file found. Service may not work properly.")
    
    # Set service-specific environment variables
    os.environ["ORDER_MANAGEMENT_SERVICE_PORT"] = "8013"
    os.environ["ORDER_MANAGEMENT_SERVICE_HOST"] = "0.0.0.0"
    os.environ["AUTH_SERVICE_URL"] = "http://localhost:8001/api"
    os.environ["FHIR_SERVICE_URL"] = "http://localhost:8004"
    
    return current_dir

def main():
    """Main function to run the service"""
    logger.info("Setting up Order Management Service environment...")
    
    current_dir = setup_environment()
    
    # Print configuration
    logger.info("Starting Order Management Service with the following configuration:")
    logger.info(f"  Service Directory: {current_dir}")
    logger.info(f"  Python Path: {os.environ.get('PYTHONPATH', 'Not set')}")
    logger.info(f"  Google Cloud Project: {os.environ.get('GOOGLE_CLOUD_PROJECT')}")
    logger.info(f"  Google Cloud Location: {os.environ.get('GOOGLE_CLOUD_LOCATION')}")
    logger.info(f"  Google Cloud Dataset: {os.environ.get('GOOGLE_CLOUD_DATASET')}")
    logger.info(f"  Google Cloud FHIR Store: {os.environ.get('GOOGLE_CLOUD_FHIR_STORE')}")
    logger.info(f"  Google Application Credentials: {os.environ.get('GOOGLE_APPLICATION_CREDENTIALS')}")
    logger.info(f"  Auth Service URL: {os.environ.get('AUTH_SERVICE_URL')}")
    logger.info("")
    
    # Start the service
    try:
        port = int(os.environ.get("ORDER_MANAGEMENT_SERVICE_PORT", "8013"))
        host = os.environ.get("ORDER_MANAGEMENT_SERVICE_HOST", "0.0.0.0")
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
