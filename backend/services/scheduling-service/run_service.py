#!/usr/bin/env python3
"""
Run script for Scheduling Service.

This script sets up the environment and starts the Scheduling Service
with proper configuration for Google Healthcare API integration.
"""

import os
import sys
import subprocess
import logging
from pathlib import Path

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

def setup_environment():
    """Set up environment variables for the service."""
    
    # Service configuration
    os.environ.setdefault("SCHEDULING_SERVICE_PORT", "8014")
    os.environ.setdefault("SCHEDULING_SERVICE_HOST", "0.0.0.0")
    os.environ.setdefault("DEBUG", "true")
    
    # Google Healthcare API configuration (matching other services)
    os.environ.setdefault("GOOGLE_CLOUD_PROJECT", "cardiofit-905a8")
    os.environ.setdefault("GOOGLE_CLOUD_LOCATION", "asia-south1")
    os.environ.setdefault("GOOGLE_CLOUD_DATASET", "clinical-synthesis-hub")
    os.environ.setdefault("GOOGLE_CLOUD_FHIR_STORE", "fhir-store")
    os.environ.setdefault("GOOGLE_APPLICATION_CREDENTIALS", "credentials/google-credentials.json")
    os.environ.setdefault("USE_GOOGLE_HEALTHCARE_API", "true")
    
    # Port configuration
    os.environ.setdefault("PORT", "8014")
    
    # Supabase configuration
    os.environ.setdefault("SUPABASE_URL", "https://auugxeqzgrnknklgwqrh.supabase.co")
    os.environ.setdefault("SUPABASE_KEY", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImF1dWd4ZXF6Z3Jua25rbGd3cXJoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NDU2NTE4NzgsImV4cCI6MjA2MTIyNzg3OH0.yAM1TGNh5aRIvTBal938vM_Ze_9gNH3qLoZ5bdmF-B8")
    os.environ.setdefault("SUPABASE_JWT_SECRET", "nXwqv86rPXO5HqJ1R1xeQnhy9JbeLeLypwUZmMoJ1prMGG6io5lU88nD6lG8MmvpN7Z2pZJvfuF33Z1x2PwCoA==")
    
    # Auth service configuration
    os.environ.setdefault("AUTH_SERVICE_URL", "http://localhost:8001")
    
    # CORS configuration
    os.environ.setdefault("BACKEND_CORS_ORIGINS", "http://localhost:3000,http://localhost:8000,http://localhost:8005")

def setup_python_path():
    """Set up Python path to include backend directory."""
    current_dir = Path(__file__).parent.absolute()
    backend_dir = current_dir.parent.parent
    
    # Add backend directory to Python path
    if str(backend_dir) not in sys.path:
        sys.path.insert(0, str(backend_dir))
    
    # Set PYTHONPATH environment variable
    current_pythonpath = os.environ.get("PYTHONPATH", "")
    if str(backend_dir) not in current_pythonpath:
        if current_pythonpath:
            os.environ["PYTHONPATH"] = f"{backend_dir}{os.pathsep}{current_pythonpath}"
        else:
            os.environ["PYTHONPATH"] = str(backend_dir)
    
    return current_dir, backend_dir

def check_credentials():
    """Check if Google Cloud credentials file exists."""
    credentials_path = "credentials/google-credentials.json"
    
    if not os.path.exists(credentials_path):
        logger.warning(f"Google Cloud credentials file not found at: {credentials_path}")
        logger.warning("Please ensure the credentials file is in place for Google Healthcare API access")
        return False
    
    logger.info(f"Google Cloud credentials found at: {credentials_path}")
    return True

def main():
    """Main function to start the service."""
    logger.info("Starting Scheduling Service...")
    
    # Set up environment
    setup_environment()
    current_dir, backend_dir = setup_python_path()
    
    # Check credentials
    check_credentials()
    
    # Log configuration
    logger.info("Service Configuration:")
    logger.info(f"  Service Directory: {current_dir}")
    logger.info(f"  Backend Directory: {backend_dir}")
    logger.info(f"  Port: {os.environ.get('SCHEDULING_SERVICE_PORT', '8014')}")
    logger.info(f"  Google Cloud Project: {os.environ.get('GOOGLE_CLOUD_PROJECT')}")
    logger.info(f"  FHIR Store: {os.environ.get('GOOGLE_CLOUD_FHIR_STORE')}")
    logger.info(f"  Auth Service URL: {os.environ.get('AUTH_SERVICE_URL')}")
    logger.info(f"  PYTHONPATH: {os.environ.get('PYTHONPATH')}")
    
    # Start the service
    try:
        port = int(os.environ.get("SCHEDULING_SERVICE_PORT", "8014"))
        host = os.environ.get("SCHEDULING_SERVICE_HOST", "0.0.0.0")
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
