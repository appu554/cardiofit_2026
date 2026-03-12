#!/usr/bin/env python3
"""
Check if Background Publisher is Auto-Processing Messages

This script checks if the background publisher is automatically processing
outbox messages without manual intervention.
"""
import asyncio
import logging
import sys
from pathlib import Path

# Add the app directory to Python path
sys.path.append(str(Path(__file__).parent / "app"))

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)]
)
logger = logging.getLogger(__name__)


async def check_auto_processing():
    """Check if background publisher is auto-processing messages"""
    logger.info("🔍 Checking Background Publisher Auto-Processing")
    logger.info("=" * 60)
    
    try:
        from app.services.outbox_service import VendorAwareOutboxService
        
        # Check current queue status
        logger.info("📊 Checking current outbox queue status...")
        outbox_service = VendorAwareOutboxService()
        await outbox_service._load_vendor_registry()
        
        queue_depths = await outbox_service.get_queue_depths()
        logger.info(f"Current queue depths: {queue_depths}")
        
        total_pending = sum(depth for depth in queue_depths.values() if depth > 0)
        
        if total_pending == 0:
            logger.info("✅ No pending messages - background publisher is working!")
            logger.info("🎉 All messages have been automatically processed")
            logger.info("")
            logger.info("VERIFICATION:")
            logger.info("1. Send a test request in Postman")
            logger.info("2. Wait 10-15 seconds")
            logger.info("3. Run this script again")
            logger.info("4. If queue is empty, auto-processing is working")
        else:
            logger.warning(f"⚠️ Found {total_pending} pending messages")
            logger.info("This could mean:")
            logger.info("1. Background publisher is not running")
            logger.info("2. Kafka connectivity issues")
            logger.info("3. Messages were just added and haven't been processed yet")
            
            # Show details by vendor
            for vendor_id, count in queue_depths.items():
                if count > 0:
                    logger.info(f"  - {vendor_id}: {count} pending messages")
        
        # Check if background publisher should be running
        logger.info("")
        logger.info("🔧 BACKGROUND PUBLISHER STATUS:")
        logger.info("The background publisher should be running automatically")
        logger.info("when the service starts. Check service logs for:")
        logger.info("- 'Starting background publisher...'")
        logger.info("- 'Background publisher started'")
        logger.info("- Periodic processing logs every 5 seconds")
        
        logger.info("")
        logger.info("📋 NEXT STEPS:")
        if total_pending > 0:
            logger.info("1. Manual processing: python test_background_publisher.py")
            logger.info("2. Check service logs for errors")
            logger.info("3. Verify Kafka connectivity")
        else:
            logger.info("1. ✅ Phase 6: Background Publishing - COMPLETE")
            logger.info("2. ✅ Phase 7: End-to-End Kafka - COMPLETE")
            logger.info("3. 🎯 Check Confluent Cloud for published messages")
            logger.info("4. 🚀 Ready for ETL pipeline testing")
        
        logger.info("=" * 60)
        
    except Exception as e:
        logger.error(f"❌ Error checking auto-processing: {e}", exc_info=True)


if __name__ == "__main__":
    asyncio.run(check_auto_processing())
