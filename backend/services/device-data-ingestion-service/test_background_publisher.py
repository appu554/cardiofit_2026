#!/usr/bin/env python3
"""
Test Background Publisher

This script manually runs the background publisher to process pending outbox messages
and publish them to Kafka for end-to-end testing.
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


async def test_background_publisher():
    """Test the background publisher end-to-end"""
    logger.info("🚀 Testing Background Publisher - End-to-End Kafka Flow")
    logger.info("=" * 70)
    
    try:
        # Import services
        from app.services.background_publisher import BackgroundPublisher
        from app.services.outbox_service import VendorAwareOutboxService
        
        # Step 1: Check current outbox status
        logger.info("📊 Step 1: Checking current outbox status...")
        outbox_service = VendorAwareOutboxService()
        await outbox_service._load_vendor_registry()
        
        queue_depths = await outbox_service.get_queue_depths()
        logger.info(f"Current queue depths: {queue_depths}")
        
        total_pending = sum(depth for depth in queue_depths.values() if depth > 0)
        if total_pending == 0:
            logger.warning("⚠️ No pending messages found in outbox!")
            logger.info("💡 Send a test request to http://localhost:8030/api/v1/ingest/device-data-smart first")
            return
        
        logger.info(f"✅ Found {total_pending} pending messages to process")
        
        # Step 2: Initialize background publisher
        logger.info("🔧 Step 2: Initializing background publisher...")
        publisher = BackgroundPublisher()
        
        initialized = await publisher.initialize()
        if not initialized:
            logger.error("❌ Failed to initialize background publisher")
            return
        
        logger.info("✅ Background publisher initialized successfully")
        
        # Step 3: Check publisher health
        logger.info("🏥 Step 3: Checking publisher health...")
        health = await publisher.health_check()
        logger.info(f"Publisher health: {health['status']}")
        
        if health['status'] != 'healthy':
            logger.warning(f"⚠️ Publisher health issues: {health}")
        
        # Step 4: Process all vendors manually
        logger.info("⚙️ Step 4: Processing all vendor outbox messages...")
        
        # Get vendor registry
        vendor_registry = outbox_service.vendor_registry
        total_processed = 0

        for vendor_id, vendor_config in vendor_registry.items():
            # Add vendor_id to config for processing
            vendor_config_with_id = vendor_config.copy()
            vendor_config_with_id["vendor_id"] = vendor_id
            
            # Check if this vendor has pending messages
            vendor_pending = queue_depths.get(vendor_id, 0)
            if vendor_pending == 0:
                logger.info(f"⏭️ Skipping {vendor_id} - no pending messages")
                continue
            
            logger.info(f"🔄 Processing {vendor_id} ({vendor_pending} pending messages)...")
            
            try:
                processed_count = await publisher.process_vendor_messages(vendor_config_with_id)
                total_processed += processed_count
                
                if processed_count > 0:
                    logger.info(f"✅ {vendor_id}: Successfully processed {processed_count} messages")
                else:
                    logger.warning(f"⚠️ {vendor_id}: No messages processed")
                    
            except Exception as e:
                logger.error(f"❌ {vendor_id}: Error processing messages: {e}")
        
        # Step 5: Check final status
        logger.info("📈 Step 5: Checking final outbox status...")
        final_queue_depths = await outbox_service.get_queue_depths()
        logger.info(f"Final queue depths: {final_queue_depths}")
        
        final_pending = sum(depth for depth in final_queue_depths.values() if depth > 0)
        
        # Step 6: Summary
        logger.info("")
        logger.info("=" * 70)
        logger.info("📋 BACKGROUND PUBLISHER TEST SUMMARY")
        logger.info("=" * 70)
        logger.info(f"Initial pending messages: {total_pending}")
        logger.info(f"Messages processed: {total_processed}")
        logger.info(f"Final pending messages: {final_pending}")
        logger.info(f"Success rate: {(total_processed/total_pending)*100:.1f}%" if total_pending > 0 else "N/A")
        
        if total_processed > 0:
            logger.info("")
            logger.info("🎉 SUCCESS: Messages published to Kafka!")
            logger.info("📡 Check your Kafka consumer or Confluent Cloud to see the messages")
            logger.info("")
            logger.info("🔍 NEXT STEPS:")
            logger.info("1. Check Confluent Cloud Console for new messages")
            logger.info("2. Verify message format and content")
            logger.info("3. Test ETL pipeline consumption")
            logger.info("4. Monitor for any processing errors")
        else:
            logger.warning("⚠️ No messages were processed successfully")
            logger.info("🔧 Check Kafka connectivity and configuration")
        
        logger.info("=" * 70)
        
    except Exception as e:
        logger.error(f"❌ Background publisher test failed: {e}", exc_info=True)


async def test_single_vendor_processing(vendor_id: str = "fitbit"):
    """Test processing for a single vendor"""
    logger.info(f"🎯 Testing single vendor processing: {vendor_id}")
    
    try:
        from app.services.background_publisher import BackgroundPublisher
        from app.services.outbox_service import VendorAwareOutboxService
        
        # Initialize services
        publisher = BackgroundPublisher()
        await publisher.initialize()
        
        outbox_service = VendorAwareOutboxService()
        await outbox_service._load_vendor_registry()
        
        # Find vendor config
        vendor_config = None
        for config in outbox_service.vendor_registry:
            if config["vendor_id"] == vendor_id:
                vendor_config = config
                break
        
        if not vendor_config:
            logger.error(f"❌ Vendor {vendor_id} not found in registry")
            return
        
        # Check pending messages
        pending_count = await outbox_service.get_queue_depth(vendor_id)
        logger.info(f"📊 {vendor_id} has {pending_count} pending messages")
        
        if pending_count == 0:
            logger.warning(f"⚠️ No pending messages for {vendor_id}")
            return
        
        # Process vendor messages
        processed_count = await publisher.process_vendor_messages(vendor_config)
        
        logger.info(f"✅ Processed {processed_count}/{pending_count} messages for {vendor_id}")
        
    except Exception as e:
        logger.error(f"❌ Single vendor test failed: {e}", exc_info=True)


async def main():
    """Main test runner"""
    import sys
    
    if len(sys.argv) > 1:
        vendor_id = sys.argv[1]
        await test_single_vendor_processing(vendor_id)
    else:
        await test_background_publisher()


if __name__ == "__main__":
    asyncio.run(main())
