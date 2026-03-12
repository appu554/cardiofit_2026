#!/usr/bin/env python3
"""
Quick start script for Workflow Engine Service.
"""
import asyncio
import uvicorn
import sys
from pathlib import Path

# Add the app directory to Python path
sys.path.insert(0, str(Path(__file__).parent / "app"))

def main():
    """Start the Workflow Engine Service."""
    print("🚀 Starting Workflow Engine Service...")
    print("📍 Service will be available at: http://localhost:8015")
    print("🏥 Health check: http://localhost:8015/health")
    print("🔗 Federation endpoint: http://localhost:8015/api/federation")
    print("📚 API docs: http://localhost:8015/docs")
    print()
    print("Press Ctrl+C to stop the service")
    print("=" * 60)
    
    try:
        uvicorn.run(
            "app.main:app",
            host="0.0.0.0",
            port=8015,
            reload=True,
            log_level="info"
        )
    except KeyboardInterrupt:
        print("\n🛑 Service stopped by user")
    except Exception as e:
        print(f"\n❌ Error starting service: {e}")
        sys.exit(1)

if __name__ == "__main__":
    main()
