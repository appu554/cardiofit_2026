#!/usr/bin/env python3
"""
Clinical Assertion Engine (CAE) gRPC Server Startup Script
Python script to start the CAE gRPC server with proper error handling
"""

import os
import sys
import subprocess
import asyncio
import logging
from pathlib import Path

def setup_logging():
    """Setup logging configuration"""
    logging.basicConfig(
        level=logging.INFO,
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )
    return logging.getLogger(__name__)

def check_dependencies():
    """Check if required dependencies are installed"""
    logger = logging.getLogger(__name__)
    
    required_packages = ['grpc', 'asyncio', 'google.protobuf']
    missing_packages = []
    
    for package in required_packages:
        try:
            __import__(package)
            logger.info(f"✓ {package} is available")
        except ImportError:
            missing_packages.append(package)
            logger.error(f"❌ {package} is missing")
    
    if missing_packages:
        logger.error(f"Missing packages: {missing_packages}")
        logger.info("Install with: pip install grpcio grpcio-tools protobuf")
        return False
    
    return True

def start_grpc_server():
    """Start the CAE gRPC server"""
    logger = setup_logging()
    
    logger.info("🚀 Starting Clinical Assertion Engine (CAE) gRPC Server...")
    
    # Check if we're in the right directory
    current_dir = Path.cwd()
    app_dir = current_dir / "app"
    
    if not app_dir.exists():
        logger.error(f"❌ App directory not found: {app_dir}")
        logger.info("💡 Make sure you're running this from the clinical-reasoning-service directory")
        return False
    
    # Check dependencies
    if not check_dependencies():
        return False
    
    # Add the app directory to the Python path
    sys.path.insert(0, str(app_dir))
    logger.info(f"📁 Added to sys.path: {app_dir}")

    # Check if grpc_server.py exists
    grpc_server_path = app_dir / "grpc_server.py"
    if not grpc_server_path.exists():
        logger.error(f"❌ gRPC server file not found: {grpc_server_path}")
        return False

    try:
        # Import and run the gRPC server
        logger.info("🔧 Importing gRPC server module...")

        # Now we can import grpc_server directly
        import grpc_server
        
        logger.info("✓ gRPC server module imported successfully")
        logger.info("🚀 Starting gRPC server on port 8027...")
        
        # Run the server
        asyncio.run(grpc_server.serve_grpc())
        
    except KeyboardInterrupt:
        logger.info("🛑 Server stopped by user")
        return True
    except Exception as e:
        logger.error(f"❌ Error starting gRPC server: {e}")
        logger.exception("Full error details:")
        return False

if __name__ == "__main__":
    success = start_grpc_server()
    sys.exit(0 if success else 1)
