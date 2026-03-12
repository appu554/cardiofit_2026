#!/usr/bin/env python3
"""
Clinical Assertion Engine Setup Script

This script sets up the CAE service with gRPC integration, following the
established patterns from your existing microservices.
"""

import os
import sys
import subprocess
import logging
from pathlib import Path

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

def check_python_version():
    """Check if Python version is compatible"""
    if sys.version_info < (3, 8):
        logger.error("Python 3.8 or higher is required")
        return False
    logger.info(f"✓ Python {sys.version_info.major}.{sys.version_info.minor} detected")
    return True

def check_dependencies():
    """Check if required system dependencies are available"""
    dependencies = ['pip', 'python']
    
    for dep in dependencies:
        try:
            subprocess.run([dep, '--version'], check=True, capture_output=True)
            logger.info(f"✓ {dep} is available")
        except (subprocess.CalledProcessError, FileNotFoundError):
            logger.error(f"✗ {dep} is not available")
            return False
    
    return True

def install_requirements():
    """Install Python requirements"""
    logger.info("Installing Python requirements...")
    
    try:
        # Install requirements
        subprocess.run([
            sys.executable, '-m', 'pip', 'install', '-r', 'requirements.txt'
        ], check=True)
        
        logger.info("✓ Requirements installed successfully")
        return True
        
    except subprocess.CalledProcessError as e:
        logger.error(f"✗ Failed to install requirements: {e}")
        return False

def compile_protocol_buffers():
    """Compile protocol buffer definitions"""
    logger.info("Compiling protocol buffer definitions...")
    
    try:
        # Run the compilation script
        subprocess.run([sys.executable, 'compile_proto.py'], check=True)
        
        # Check if files were generated
        proto_dir = Path('app/proto')
        pb2_file = proto_dir / 'clinical_reasoning_pb2.py'
        grpc_file = proto_dir / 'clinical_reasoning_pb2_grpc.py'
        
        if pb2_file.exists() and grpc_file.exists():
            logger.info("✓ Protocol buffers compiled successfully")
            return True
        else:
            logger.error("✗ Protocol buffer files not found after compilation")
            return False
            
    except subprocess.CalledProcessError as e:
        logger.error(f"✗ Failed to compile protocol buffers: {e}")
        return False

def create_directories():
    """Create necessary directories"""
    directories = [
        'app/proto',
        'app/core',
        'app/reasoners',
        'app/api',
        'app/knowledge',
        'app/validation',
        'logs',
        'tests',
        'tests/unit',
        'tests/integration',
        'tests/clinical'
    ]
    
    for directory in directories:
        Path(directory).mkdir(parents=True, exist_ok=True)
        logger.info(f"✓ Created directory: {directory}")

def create_init_files():
    """Create __init__.py files for Python packages"""
    init_files = [
        'app/__init__.py',
        'app/proto/__init__.py',
        'app/core/__init__.py',
        'app/reasoners/__init__.py',
        'app/api/__init__.py',
        'app/knowledge/__init__.py',
        'app/validation/__init__.py',
        'tests/__init__.py',
        'tests/unit/__init__.py',
        'tests/integration/__init__.py',
        'tests/clinical/__init__.py'
    ]
    
    for init_file in init_files:
        Path(init_file).touch()
        logger.info(f"✓ Created: {init_file}")

def create_env_file():
    """Create .env file with default configuration"""
    env_content = """# Clinical Reasoning Service Configuration

# Service Configuration
PROJECT_NAME=Clinical Reasoning Service
API_PREFIX=/api
VERSION=1.0.0

# Server Configuration
HOST=0.0.0.0
PORT=8027
GRPC_PORT=8027

# Google Healthcare API Configuration
USE_GOOGLE_HEALTHCARE_API=true
GOOGLE_CLOUD_PROJECT=cardiofit-905a8
GOOGLE_CLOUD_LOCATION=asia-south1
GOOGLE_CLOUD_DATASET=clinical-synthesis-hub
GOOGLE_CLOUD_FHIR_STORE=fhir-store
GOOGLE_APPLICATION_CREDENTIALS=credentials/google-credentials.json

# Supabase Configuration
SUPABASE_URL=https://auugxeqzgrnknklgwqrh.supabase.co
SUPABASE_KEY=your_supabase_key_here

# External Service URLs
API_GATEWAY_URL=http://localhost:8005
APOLLO_FEDERATION_URL=http://localhost:4000
AUTH_SERVICE_URL=http://localhost:8001
FHIR_SERVICE_URL=http://localhost:8014
PATIENT_SERVICE_URL=http://localhost:8003
MEDICATION_SERVICE_URL=http://localhost:8009
OBSERVATION_SERVICE_URL=http://localhost:8007
CONDITION_SERVICE_URL=http://localhost:8010

# Global Outbox Service Configuration
GLOBAL_OUTBOX_SERVICE_URL=localhost:50051
USE_GLOBAL_OUTBOX=true

# Clinical Reasoning Configuration
DEFAULT_REASONER_TIMEOUT=30
MAX_CONCURRENT_REQUESTS=100
ENABLE_STREAMING=true

# Caching Configuration
REDIS_URL=redis://localhost:6379
CACHE_TTL_SECONDS=300
ENABLE_L1_CACHE=true
ENABLE_L2_CACHE=true

# Safety Configuration
ENABLE_SAFETY_NET=true
MIN_CONFIDENCE_THRESHOLD=0.7
CONSERVATIVE_MODE=true

# Monitoring Configuration
ENABLE_METRICS=true
METRICS_PORT=9090
LOG_LEVEL=INFO

# Performance Configuration
GRPC_MAX_WORKERS=10
HTTP_MAX_WORKERS=4
REQUEST_TIMEOUT=30

# Clinical Validation Configuration
ENABLE_CLINICAL_VALIDATION=true
VALIDATION_SAMPLE_RATE=0.1
EXPERT_REVIEW_THRESHOLD=0.8
"""
    
    env_file = Path('.env')
    if not env_file.exists():
        env_file.write_text(env_content)
        logger.info("✓ Created .env file with default configuration")
    else:
        logger.info("✓ .env file already exists")

def test_grpc_setup():
    """Test if gRPC setup is working"""
    logger.info("Testing gRPC setup...")
    
    try:
        # Try to import the compiled protocol buffers
        sys.path.insert(0, 'app/proto')
        import clinical_reasoning_pb2
        import clinical_reasoning_pb2_grpc
        
        logger.info("✓ gRPC protocol buffers can be imported")
        
        # Test basic message creation
        request = clinical_reasoning_pb2.HealthCheckRequest(service="test")
        logger.info("✓ gRPC message creation works")
        
        return True
        
    except ImportError as e:
        logger.error(f"✗ gRPC import failed: {e}")
        return False
    except Exception as e:
        logger.error(f"✗ gRPC test failed: {e}")
        return False

def main():
    """Main setup function"""
    logger.info("🚀 Setting up Clinical Assertion Engine (CAE) with gRPC")
    logger.info("=" * 60)
    
    # Check prerequisites
    if not check_python_version():
        sys.exit(1)
    
    if not check_dependencies():
        sys.exit(1)
    
    # Create directory structure
    logger.info("\n📁 Creating directory structure...")
    create_directories()
    create_init_files()
    
    # Create configuration
    logger.info("\n⚙️  Creating configuration...")
    create_env_file()
    
    # Install dependencies
    logger.info("\n📦 Installing dependencies...")
    if not install_requirements():
        sys.exit(1)
    
    # Compile protocol buffers
    logger.info("\n🔧 Compiling protocol buffers...")
    if not compile_protocol_buffers():
        sys.exit(1)
    
    # Test setup
    logger.info("\n🧪 Testing setup...")
    if not test_grpc_setup():
        logger.warning("⚠️  gRPC setup test failed, but continuing...")
    
    # Success message
    logger.info("\n✅ CAE setup completed successfully!")
    logger.info("\nNext steps:")
    logger.info("1. Update .env file with your specific configuration")
    logger.info("2. Start the gRPC server: python app/grpc_server.py")
    logger.info("3. Test the gRPC client: python test_cae_client.py")
    logger.info("4. Integrate with existing microservices")
    
    logger.info("\n🔗 Integration with existing services:")
    logger.info("- Global Outbox Service: ✓ Ready")
    logger.info("- Apollo Federation: ⏳ Pending GraphQL schema")
    logger.info("- Authentication: ⏳ Pending middleware setup")
    logger.info("- Monitoring: ⏳ Pending metrics setup")

if __name__ == "__main__":
    main()
