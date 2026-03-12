#!/usr/bin/env python3
"""
Organization Service Runner

This script starts the Organization Management Service with proper configuration
and environment setup.
"""

import os
import sys
import subprocess
import logging
from pathlib import Path

# Add the current directory to Python path
current_dir = Path(__file__).parent
sys.path.insert(0, str(current_dir))

# Add shared modules to path
shared_dir = current_dir.parent / "shared"
sys.path.insert(0, str(shared_dir))

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

def setup_environment():
    """Set up environment variables for the service."""
    
    # Service configuration
    os.environ.setdefault("ORGANIZATION_SERVICE_PORT", "8012")
    os.environ.setdefault("ORGANIZATION_SERVICE_HOST", "0.0.0.0")
    os.environ.setdefault("DEBUG", "true")
    
    # Google Healthcare API configuration (corrected to match your setup)
    os.environ.setdefault("GOOGLE_CLOUD_PROJECT_ID", "cardiofit-905a8")
    os.environ.setdefault("GOOGLE_CLOUD_LOCATION", "asia-south1")
    os.environ.setdefault("GOOGLE_CLOUD_DATASET_ID", "clinical-synthesis-hub")
    os.environ.setdefault("GOOGLE_CLOUD_FHIR_STORE_ID", "fhir-store")
    os.environ.setdefault("GOOGLE_CLOUD_CREDENTIALS_PATH", "credentials/service-account-key.json")
    os.environ.setdefault("USE_GOOGLE_HEALTHCARE_API", "true")
    
    # Set credentials path if not already set
    if not os.environ.get("GOOGLE_APPLICATION_CREDENTIALS"):
        credentials_path = current_dir / "credentials" / "service-account-key.json"
        if credentials_path.exists():
            os.environ["GOOGLE_APPLICATION_CREDENTIALS"] = str(credentials_path)
            logger.info(f"Using Google credentials from: {credentials_path}")
        else:
            logger.warning("Google credentials not found. Please set GOOGLE_APPLICATION_CREDENTIALS")
    
    # Supabase configuration
    os.environ.setdefault("SUPABASE_URL", "https://auugxeqzgrnknklgwqrh.supabase.co")
    os.environ.setdefault("SUPABASE_KEY", "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImF1dWd4ZXF6Z3Jua25rbGd3cXJoIiwicm9sZSI6ImFub24iLCJpYXQiOjE3NDU2NTE4NzgsImV4cCI6MjA2MTIyNzg3OH0.yAM1TGNh5aRIvTBal938vM_Ze_9gNH3qLoZ5bdmF-B8")
    os.environ.setdefault("SUPABASE_JWT_SECRET", "nXwqv86rPXO5HqJ1R1xeQnhy9JbeLeLypwUZmMoJ1prMGG6io5lU88nD6lG8MmvpN7Z2pZJvfuF33Z1x2PwCoA==")
    
    # Auth service configuration
    os.environ.setdefault("AUTH_SERVICE_URL", "http://localhost:8001")
    
    # CORS configuration
    os.environ.setdefault("BACKEND_CORS_ORIGINS", "http://localhost:3000,http://localhost:8000,http://localhost:8005")

def check_dependencies():
    """Check if required dependencies are available."""
    try:
        import fastapi
        import uvicorn
        import strawberry
        import google.auth
        import googleapiclient.discovery
        logger.info("All required dependencies are available")
        return True
    except ImportError as e:
        logger.error(f"Missing dependency: {e}")
        logger.error("Please install dependencies with: pip install -r requirements.txt")
        return False

def main():
    """Main function to start the service."""
    logger.info("Starting Organization Management Service...")
    
    # Setup environment
    setup_environment()
    
    # Check dependencies
    if not check_dependencies():
        sys.exit(1)
    
    # Print configuration
    logger.info("Service Configuration:")
    logger.info(f"  Port: {os.environ.get('ORGANIZATION_SERVICE_PORT')}")
    logger.info(f"  Host: {os.environ.get('ORGANIZATION_SERVICE_HOST')}")
    logger.info(f"  Debug: {os.environ.get('DEBUG')}")
    logger.info(f"  Google Cloud Project: {os.environ.get('GOOGLE_CLOUD_PROJECT_ID')}")
    logger.info(f"  Google Cloud Location: {os.environ.get('GOOGLE_CLOUD_LOCATION')}")
    logger.info(f"  FHIR Store: {os.environ.get('GOOGLE_CLOUD_FHIR_STORE_ID')}")
    logger.info(f"  Auth Service: {os.environ.get('AUTH_SERVICE_URL')}")
    
    # Start the service
    try:
        port = int(os.environ.get("ORGANIZATION_SERVICE_PORT", "8012"))
        host = os.environ.get("ORGANIZATION_SERVICE_HOST", "0.0.0.0")
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
