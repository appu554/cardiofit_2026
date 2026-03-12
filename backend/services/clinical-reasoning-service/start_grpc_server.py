"""
Start CAE Engine gRPC Server

Simple startup script for the Clinical Reasoning Service gRPC server
with Neo4j integration.
"""

import asyncio
import logging
import sys
from pathlib import Path

# Add app directory to path
app_dir = Path(__file__).parent / "app"
sys.path.insert(0, str(app_dir))

# Load environment variables
try:
    from dotenv import load_dotenv
    load_dotenv()
    print("✅ Loaded environment variables from .env file")
except ImportError:
    print("⚠️  python-dotenv not installed")

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

logger = logging.getLogger(__name__)

async def main():
    """Start the gRPC server"""
    print("🚀 Starting CAE Engine gRPC Server")
    print("=" * 50)
    
    try:
        # Import and start the gRPC server
        from app.grpc_server import serve_grpc
        
        print("🔧 Initializing CAE Engine with Neo4j...")
        print("🌐 Starting gRPC server on port 50051...")
        print("📡 Ready to receive clinical reasoning requests")
        print("=" * 50)
        print("Press Ctrl+C to stop the server")
        
        # Start the server
        await serve_grpc()
        
    except KeyboardInterrupt:
        print("\n👋 Shutting down gRPC server...")
    except Exception as e:
        print(f"❌ Failed to start gRPC server: {e}")
        logger.error(f"gRPC server startup failed: {e}", exc_info=True)
        return 1
    
    return 0

if __name__ == "__main__":
    exit_code = asyncio.run(main())
    sys.exit(exit_code)
