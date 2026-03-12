#!/usr/bin/env python3
"""
Database Migration Runner for Transactional Outbox Pattern
Runs the outbox tables migration script against Supabase PostgreSQL
"""
import asyncio
import logging
import sys
from pathlib import Path

# Add the app directory to Python path
sys.path.append(str(Path(__file__).parent / "app"))

from app.db.database import run_migration_script, startup_database, shutdown_database

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


async def main():
    """Run the outbox tables migration"""
    logger.info("🚀 Starting Transactional Outbox Pattern Migration")
    
    try:
        # Initialize database connection
        logger.info("📡 Initializing database connection...")
        await startup_database()
        
        # Run migration
        migration_file = Path(__file__).parent / "migrations" / "001_create_outbox_tables.sql"
        logger.info(f"📄 Running migration: {migration_file}")
        
        success = await run_migration_script(str(migration_file))
        
        if success:
            logger.info("✅ Migration completed successfully!")
            logger.info("🎯 Outbox tables created:")
            logger.info("   - fitbit_outbox & fitbit_dead_letter")
            logger.info("   - garmin_outbox & garmin_dead_letter") 
            logger.info("   - apple_health_outbox & apple_health_dead_letter")
            logger.info("   - vendor_outbox_registry")
            logger.info("   - Monitoring views and utility functions")
            
            # Test the setup
            logger.info("🔍 Testing outbox setup...")
            await test_outbox_setup()
            
        else:
            logger.error("❌ Migration failed!")
            sys.exit(1)
            
    except Exception as e:
        logger.error(f"💥 Migration error: {e}")
        sys.exit(1)
    finally:
        await shutdown_database()


async def test_outbox_setup():
    """Test that the outbox setup is working correctly"""
    try:
        from app.services.outbox_service import VendorAwareOutboxService
        
        # Initialize outbox service
        outbox_service = VendorAwareOutboxService()
        
        # Test health status
        health_status = await outbox_service.get_health_status()
        logger.info(f"📊 Outbox Health Status: {health_status}")
        
        # Test queue depths
        queue_depths = await outbox_service.get_queue_depths()
        logger.info(f"📈 Queue Depths: {queue_depths}")
        
        if health_status.get("registry_loaded"):
            logger.info("✅ Vendor registry loaded successfully")
        else:
            logger.warning("⚠️ Vendor registry not loaded")
            
        if all(depth >= 0 for depth in queue_depths.values()):
            logger.info("✅ All outbox tables accessible")
        else:
            logger.warning("⚠️ Some outbox tables may have issues")
            
    except Exception as e:
        logger.error(f"❌ Outbox setup test failed: {e}")


if __name__ == "__main__":
    asyncio.run(main())
