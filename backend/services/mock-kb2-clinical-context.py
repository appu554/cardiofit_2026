#!/usr/bin/env python3
"""
Mock KB2 Clinical Context Service
Demonstrates isolated service running on port 8082 with dedicated MongoDB and Redis
"""

import asyncio
import json
from aiohttp import web
import pymongo
import redis
import logging
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class MockKB2Service:
    def __init__(self):
        self.mongodb_client = None
        self.redis_client = None

    async def init_databases(self):
        """Initialize connections to dedicated databases"""
        try:
            # Connect to dedicated MongoDB (port 27018)
            self.mongodb_client = pymongo.MongoClient(
                "mongodb://kb2admin:kb2_mongodb_password@localhost:27018/kb2_clinical_context"
            )
            # Test connection
            self.mongodb_client.admin.command('ismaster')
            logger.info("✅ Connected to dedicated MongoDB on port 27018")

            # Connect to dedicated Redis (port 6381)
            self.redis_client = redis.Redis(host='localhost', port=6381, decode_responses=True)
            # Test connection
            self.redis_client.ping()
            logger.info("✅ Connected to dedicated Redis on port 6381")

        except Exception as e:
            logger.error(f"❌ Database connection failed: {e}")

    async def health_check(self, request):
        """Health check endpoint"""
        status = {
            "service": "KB2 Clinical Context",
            "status": "healthy",
            "port": 8082,
            "timestamp": datetime.now().isoformat(),
            "databases": {
                "mongodb": "connected" if self.mongodb_client else "disconnected",
                "redis": "connected" if self.redis_client else "disconnected"
            }
        }
        return web.json_response(status)

    async def clinical_context_endpoint(self, request):
        """Mock clinical context analysis endpoint"""
        context_data = {
            "service": "KB2 Clinical Context",
            "endpoint": "/clinical-context",
            "message": "Clinical context analysis service running on dedicated databases",
            "databases": {
                "mongodb_port": 27018,
                "redis_port": 6381
            },
            "timestamp": datetime.now().isoformat()
        }
        return web.json_response(context_data)

async def create_app():
    """Create and configure the web application"""
    service = MockKB2Service()
    await service.init_databases()

    app = web.Application()
    app.router.add_get('/health', service.health_check)
    app.router.add_get('/clinical-context', service.clinical_context_endpoint)
    app.router.add_post('/clinical-context', service.clinical_context_endpoint)

    return app

async def main():
    """Main entry point"""
    logger.info("🚀 Starting KB2 Clinical Context Service on port 8082")
    logger.info("📊 Using dedicated MongoDB (port 27018) and Redis (port 6381)")

    app = await create_app()
    runner = web.AppRunner(app)
    await runner.setup()

    site = web.TCPSite(runner, '0.0.0.0', 8082)
    await site.start()

    logger.info("✅ KB2 Clinical Context Service is ready!")
    logger.info("🌐 Health Check: http://localhost:8082/health")
    logger.info("🔧 API Endpoint: http://localhost:8082/clinical-context")

    # Keep the server running
    try:
        while True:
            await asyncio.sleep(1)
    except KeyboardInterrupt:
        logger.info("🛑 Shutting down KB2 Clinical Context Service")
    finally:
        await runner.cleanup()

if __name__ == "__main__":
    asyncio.run(main())