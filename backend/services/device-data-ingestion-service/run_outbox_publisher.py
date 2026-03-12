#!/usr/bin/env python3
"""
Standalone Outbox Publisher Service

This script runs the outbox publisher as a standalone background service.
It can be deployed separately from the main ingestion service for better
scalability and fault isolation.

Usage:
    python run_outbox_publisher.py

Environment Variables:
    - DATABASE_URL: PostgreSQL connection string
    - KAFKA_BOOTSTRAP_SERVERS: Kafka bootstrap servers
    - KAFKA_API_KEY: Kafka API key
    - KAFKA_API_SECRET: Kafka API secret
    - OUTBOX_BATCH_SIZE: Batch size for processing (default: 50)
    - OUTBOX_POLL_INTERVAL: Poll interval in seconds (default: 5)
    - MAX_CONCURRENT_VENDORS: Max concurrent vendor processing (default: 10)
"""
import asyncio
import logging
import signal
import sys
from pathlib import Path

# Add the app directory to Python path
sys.path.append(str(Path(__file__).parent / "app"))

from app.services.outbox_publisher import outbox_publisher
from app.db.database import startup_database, shutdown_database
from app.config import settings

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.FileHandler('outbox_publisher.log')
    ]
)
logger = logging.getLogger(__name__)


class OutboxPublisherService:
    """Standalone outbox publisher service with graceful shutdown"""
    
    def __init__(self):
        self.shutdown_event = asyncio.Event()
        self.publisher_task = None
        
    async def start(self):
        """Start the outbox publisher service"""
        logger.info("🚀 Starting Outbox Publisher Service")
        logger.info(f"📊 Configuration:")
        logger.info(f"   - Batch Size: {settings.OUTBOX_BATCH_SIZE}")
        logger.info(f"   - Poll Interval: {settings.OUTBOX_POLL_INTERVAL}s")
        logger.info(f"   - Max Concurrent Vendors: {settings.MAX_CONCURRENT_VENDORS}")
        logger.info(f"   - Database: {settings.DATABASE_URL[:50]}...")
        logger.info(f"   - Kafka: {settings.KAFKA_BOOTSTRAP_SERVERS}")
        
        try:
            # Initialize database
            logger.info("📡 Initializing database connection...")
            await startup_database()
            
            # Start the publisher
            logger.info("🔄 Starting outbox publisher...")
            self.publisher_task = asyncio.create_task(
                outbox_publisher.start_publishing_loop()
            )
            
            # Wait for shutdown signal
            await self.shutdown_event.wait()
            
        except Exception as e:
            logger.error(f"💥 Fatal error in publisher service: {e}")
            raise
        finally:
            await self.stop()
    
    async def stop(self):
        """Stop the outbox publisher service gracefully"""
        logger.info("🛑 Stopping Outbox Publisher Service...")
        
        try:
            # Stop the publisher
            if self.publisher_task and not self.publisher_task.done():
                await outbox_publisher.stop()
                
                # Wait for publisher to stop gracefully
                try:
                    await asyncio.wait_for(self.publisher_task, timeout=30.0)
                except asyncio.TimeoutError:
                    logger.warning("Publisher did not stop gracefully, cancelling...")
                    self.publisher_task.cancel()
                    try:
                        await self.publisher_task
                    except asyncio.CancelledError:
                        pass
            
            # Close database connections
            await shutdown_database()
            logger.info("✅ Outbox Publisher Service stopped gracefully")
            
        except Exception as e:
            logger.error(f"❌ Error during shutdown: {e}")
    
    def signal_handler(self, signum, frame):
        """Handle shutdown signals"""
        logger.info(f"📡 Received signal {signum}, initiating graceful shutdown...")
        self.shutdown_event.set()


async def health_check():
    """Perform a health check of the outbox system"""
    try:
        logger.info("🔍 Performing health check...")
        
        # Initialize database
        await startup_database()
        
        # Check outbox service health
        from app.services.outbox_service import VendorAwareOutboxService
        outbox_service = VendorAwareOutboxService()
        
        health_status = await outbox_service.get_health_status()
        queue_depths = await outbox_service.get_queue_depths()
        
        logger.info("📊 Health Check Results:")
        logger.info(f"   - Overall Status: {health_status.get('status', 'unknown')}")
        logger.info(f"   - Registry Loaded: {health_status.get('registry_loaded', False)}")
        logger.info(f"   - Total Pending: {health_status.get('total_pending', 0)}")
        logger.info(f"   - Queue Depths: {queue_depths}")
        
        # Check publisher health
        publisher_health = outbox_publisher.get_health_status()
        logger.info(f"   - Publisher Running: {publisher_health.get('is_running', False)}")
        
        await shutdown_database()
        
        if health_status.get('status') == 'healthy':
            logger.info("✅ Health check passed")
            return True
        else:
            logger.warning("⚠️ Health check found issues")
            return False
            
    except Exception as e:
        logger.error(f"❌ Health check failed: {e}")
        return False


async def main():
    """Main entry point"""
    import argparse
    
    parser = argparse.ArgumentParser(description="Outbox Publisher Service")
    parser.add_argument(
        "--health-check", 
        action="store_true", 
        help="Perform health check and exit"
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Initialize and test configuration without starting publisher"
    )
    
    args = parser.parse_args()
    
    if args.health_check:
        success = await health_check()
        sys.exit(0 if success else 1)
    
    if args.dry_run:
        logger.info("🧪 Dry run mode - testing configuration...")
        try:
            await startup_database()
            success = await outbox_publisher.initialize()
            await shutdown_database()
            
            if success:
                logger.info("✅ Dry run successful - configuration is valid")
                sys.exit(0)
            else:
                logger.error("❌ Dry run failed - configuration issues detected")
                sys.exit(1)
        except Exception as e:
            logger.error(f"❌ Dry run failed: {e}")
            sys.exit(1)
    
    # Normal operation - start the service
    service = OutboxPublisherService()
    
    # Set up signal handlers for graceful shutdown
    for sig in [signal.SIGTERM, signal.SIGINT]:
        signal.signal(sig, service.signal_handler)
    
    try:
        await service.start()
    except KeyboardInterrupt:
        logger.info("📡 Received keyboard interrupt")
    except Exception as e:
        logger.error(f"💥 Service failed: {e}")
        sys.exit(1)


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("👋 Goodbye!")
    except Exception as e:
        logger.error(f"💥 Fatal error: {e}")
        sys.exit(1)
