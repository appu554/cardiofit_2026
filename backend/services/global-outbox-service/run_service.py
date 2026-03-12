#!/usr/bin/env python3
"""
Global Outbox Service Runner

Main entry point for the Global Outbox Service.
Handles service startup, configuration validation, and graceful shutdown.
"""

import asyncio
import logging
import os
import sys
import signal
from pathlib import Path

# Add the backend directory to Python path for shared imports
backend_dir = Path(__file__).parent.parent.parent
if str(backend_dir) not in sys.path:
    sys.path.insert(0, str(backend_dir))

# Now we can import our modules
from app.core.config import settings
from app.core.database import db_manager

# Configure logging
logging.basicConfig(
    level=getattr(logging, settings.LOG_LEVEL.upper()),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.FileHandler('global-outbox-service.log')
    ]
)

logger = logging.getLogger(__name__)

def validate_environment():
    """
    Validate environment configuration before starting
    
    Checks for required environment variables and configuration
    """
    logger.info("🔍 Validating environment configuration...")
    
    issues = []
    
    # Check database configuration
    if not settings.DATABASE_URL or settings.DATABASE_URL == "postgresql://postgres.auugxeqzgrnknklgwqrh:PASSWORD@aws-0-ap-south-1.pooler.supabase.com:5432/postgres":
        issues.append("DATABASE_URL not properly configured (contains placeholder PASSWORD)")
    
    # Check Kafka configuration
    if not settings.KAFKA_API_SECRET:
        issues.append("KAFKA_API_SECRET not configured")
    
    # Check ports
    if settings.PORT == settings.GRPC_PORT:
        issues.append(f"HTTP and gRPC ports cannot be the same ({settings.PORT})")
    
    # Check for port conflicts
    import socket
    for port_name, port in [("HTTP", settings.PORT), ("gRPC", settings.GRPC_PORT)]:
        try:
            with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
                s.bind(('localhost', port))
        except OSError:
            issues.append(f"{port_name} port {port} is already in use")
    
    if issues:
        logger.error("❌ Environment validation failed:")
        for issue in issues:
            logger.error(f"   - {issue}")
        return False
    
    logger.info("✅ Environment validation passed")
    return True

def check_dependencies():
    """
    Check if all required dependencies are available
    """
    logger.info("🔍 Checking dependencies...")
    
    missing_deps = []
    
    # Check Protocol Buffers
    try:
        from app.proto import outbox_pb2, outbox_pb2_grpc
    except ImportError:
        missing_deps.append("Protocol buffer files (run 'python compile_proto.py')")
    
    # Check database driver
    try:
        import asyncpg
    except ImportError:
        missing_deps.append("asyncpg (run 'pip install asyncpg')")
    
    # Check gRPC
    try:
        import grpc
    except ImportError:
        missing_deps.append("grpcio (run 'pip install grpcio')")
    
    # Check FastAPI
    try:
        import fastapi
    except ImportError:
        missing_deps.append("fastapi (run 'pip install fastapi')")
    
    if missing_deps:
        logger.error("❌ Missing dependencies:")
        for dep in missing_deps:
            logger.error(f"   - {dep}")
        return False
    
    logger.info("✅ All dependencies available")
    return True

async def test_database_connection():
    """
    Test database connection before starting the service
    """
    logger.info("🔍 Testing database connection...")
    
    try:
        connected = await db_manager.connect()
        if connected:
            health = await db_manager.health_check()
            if health.get("status") == "healthy":
                logger.info("✅ Database connection successful")
                await db_manager.disconnect()
                return True
            else:
                logger.error(f"❌ Database health check failed: {health}")
                return False
        else:
            logger.error("❌ Failed to connect to database")
            return False
    except Exception as e:
        logger.error(f"❌ Database connection test failed: {e}")
        return False

def setup_signal_handlers():
    """Setup signal handlers for graceful shutdown"""
    def signal_handler(signum, frame):
        logger.info(f"Received signal {signum}, initiating graceful shutdown...")
        sys.exit(0)
    
    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

async def main():
    """Main service entry point"""
    
    print("=" * 60)
    print(f"🌐 {settings.PROJECT_NAME} v{settings.VERSION}")
    print("=" * 60)
    
    # Pre-flight checks
    logger.info("🚀 Starting Global Outbox Service...")
    logger.info(f"   Environment: {settings.ENVIRONMENT}")
    logger.info(f"   Debug Mode: {settings.DEBUG}")
    logger.info(f"   HTTP Port: {settings.PORT}")
    logger.info(f"   gRPC Port: {settings.GRPC_PORT}")
    
    # Validate environment
    if not validate_environment():
        logger.error("❌ Environment validation failed, exiting...")
        sys.exit(1)
    
    # Check dependencies
    if not check_dependencies():
        logger.error("❌ Dependency check failed, exiting...")
        sys.exit(1)
    
    # Test database connection
    if not await test_database_connection():
        logger.error("❌ Database connection test failed, exiting...")
        sys.exit(1)
    
    # Setup signal handlers
    setup_signal_handlers()
    
    # Start the service
    try:
        logger.info("🚀 All pre-flight checks passed, starting service...")
        
        # Import and run the FastAPI application
        import uvicorn
        from app.main import app
        
        # Configure uvicorn
        config = uvicorn.Config(
            app=app,
            host=settings.HOST,
            port=settings.PORT,
            reload=settings.DEBUG,
            log_level=settings.LOG_LEVEL.lower(),
            access_log=True,
            loop="asyncio"
        )
        
        server = uvicorn.Server(config)
        
        logger.info(f"✅ {settings.PROJECT_NAME} starting on {settings.HOST}:{settings.PORT}")
        logger.info(f"   gRPC endpoint: {settings.HOST}:{settings.GRPC_PORT}")
        logger.info(f"   Health check: http://{settings.HOST}:{settings.PORT}/health")
        logger.info(f"   Statistics: http://{settings.HOST}:{settings.PORT}/stats")
        
        if settings.DEBUG:
            logger.info(f"   API docs: http://{settings.HOST}:{settings.PORT}/docs")
        
        await server.serve()
        
    except KeyboardInterrupt:
        logger.info("🛑 Service interrupted by user")
    except Exception as e:
        logger.error(f"❌ Service failed to start: {e}", exc_info=True)
        sys.exit(1)
    finally:
        logger.info("🛑 Global Outbox Service shutdown complete")

def run_migration():
    """Run database migration"""
    async def _run_migration():
        logger.info("🔧 Running database migration...")
        
        try:
            connected = await db_manager.connect()
            if not connected:
                logger.error("❌ Failed to connect to database for migration")
                return False
            
            success = await db_manager.execute_migration()
            await db_manager.disconnect()
            
            if success:
                logger.info("✅ Database migration completed successfully")
                return True
            else:
                logger.error("❌ Database migration failed")
                return False
                
        except Exception as e:
            logger.error(f"❌ Migration error: {e}")
            return False
    
    return asyncio.run(_run_migration())

def compile_proto():
    """Compile Protocol Buffer definitions"""
    logger.info("🔨 Compiling Protocol Buffer definitions...")
    
    try:
        import subprocess
        result = subprocess.run([sys.executable, "compile_proto.py"], 
                              capture_output=True, text=True, cwd=Path(__file__).parent)
        
        if result.returncode == 0:
            logger.info("✅ Protocol Buffer compilation successful")
            return True
        else:
            logger.error(f"❌ Protocol Buffer compilation failed: {result.stderr}")
            return False
            
    except Exception as e:
        logger.error(f"❌ Protocol Buffer compilation error: {e}")
        return False

if __name__ == "__main__":
    # Handle command line arguments
    if len(sys.argv) > 1:
        command = sys.argv[1].lower()
        
        if command == "migrate":
            success = run_migration()
            sys.exit(0 if success else 1)
        elif command == "compile":
            success = compile_proto()
            sys.exit(0 if success else 1)
        elif command == "check":
            # Run all checks without starting the service
            valid_env = validate_environment()
            valid_deps = check_dependencies()
            valid_db = asyncio.run(test_database_connection())
            
            if valid_env and valid_deps and valid_db:
                print("✅ All checks passed - service is ready to start")
                sys.exit(0)
            else:
                print("❌ Some checks failed - see logs above")
                sys.exit(1)
        else:
            print(f"Unknown command: {command}")
            print("Available commands:")
            print("  migrate  - Run database migration")
            print("  compile  - Compile Protocol Buffer definitions")
            print("  check    - Run pre-flight checks")
            sys.exit(1)
    
    # Default: start the service
    asyncio.run(main())
