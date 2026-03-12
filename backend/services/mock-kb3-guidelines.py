#!/usr/bin/env python3
"""
Mock KB3 Guidelines Service
Demonstrates isolated service running on port 8084 with dedicated PostgreSQL
"""

import asyncio
import json
from aiohttp import web
import psycopg2
import logging
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class MockKB3Service:
    def __init__(self):
        self.postgres_conn = None

    async def init_databases(self):
        """Initialize connections to dedicated databases"""
        try:
            # Connect to dedicated PostgreSQL (port 5434)
            self.postgres_conn = psycopg2.connect(
                host="localhost",
                port=5434,
                database="kb3_guidelines",
                user="kb3admin",
                password="kb3_postgres_password"
            )
            logger.info("✅ Connected to dedicated PostgreSQL on port 5434")

        except Exception as e:
            logger.error(f"❌ Database connection failed: {e}")

    async def health_check(self, request):
        """Health check endpoint"""
        status = {
            "service": "KB3 Guidelines",
            "status": "healthy",
            "port": 8084,
            "timestamp": datetime.now().isoformat(),
            "databases": {
                "postgresql": "connected" if self.postgres_conn else "disconnected"
            }
        }
        return web.json_response(status)

    async def guidelines_endpoint(self, request):
        """Mock guidelines analysis endpoint"""
        guidelines_data = {
            "service": "KB3 Guidelines",
            "endpoint": "/guidelines",
            "message": "Clinical guidelines service running on dedicated PostgreSQL",
            "databases": {
                "postgresql_port": 5434
            },
            "timestamp": datetime.now().isoformat()
        }
        return web.json_response(guidelines_data)

async def create_app():
    """Create and configure the web application"""
    service = MockKB3Service()
    await service.init_databases()

    app = web.Application()
    app.router.add_get('/health', service.health_check)
    app.router.add_get('/guidelines', service.guidelines_endpoint)
    app.router.add_post('/guidelines', service.guidelines_endpoint)

    return app

async def main():
    """Main entry point"""
    logger.info("🚀 Starting KB3 Guidelines Service on port 8084")
    logger.info("📊 Using dedicated PostgreSQL (port 5434)")

    app = await create_app()
    runner = web.AppRunner(app)
    await runner.setup()

    site = web.TCPSite(runner, '0.0.0.0', 8084)
    await site.start()

    logger.info("✅ KB3 Guidelines Service is ready!")
    logger.info("🌐 Health Check: http://localhost:8084/health")
    logger.info("🔧 API Endpoint: http://localhost:8084/guidelines")

    # Keep the server running
    try:
        while True:
            await asyncio.sleep(1)
    except KeyboardInterrupt:
        logger.info("🛑 Shutting down KB3 Guidelines Service")
    finally:
        await runner.cleanup()

if __name__ == "__main__":
    asyncio.run(main())