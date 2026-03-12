#!/usr/bin/env python3
"""
Test Complete End-to-End Flow

This script tests the complete outbox pattern flow:
1. Ingest device data
2. Process outbox messages
3. Publish to Kafka
"""
import asyncio
import logging
import sys
import json
from pathlib import Path
from datetime import datetime

# Add the app directory to Python path
sys.path.append(str(Path(__file__).parent / "app"))

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)]
)
logger = logging.getLogger(__name__)


async def test_complete_flow():
    """Test the complete end-to-end flow"""
    logger.info("🚀 Testing Complete End-to-End Outbox Pattern Flow")
    logger.info("=" * 70)
    
    try:
        # Import services
        from app.services.outbox_service import VendorAwareOutboxService
        from app.services.background_publisher import BackgroundPublisher
        from app.services.vendor_detection import vendor_detection_service
        
        # Step 1: Create test device data
        logger.info("📱 Step 1: Creating test device data...")
        test_device_data = {
            "device_id": "test_complete_flow_001",
            "timestamp": int(datetime.utcnow().timestamp()),
            "reading_type": "heart_rate",
            "value": 88,
            "unit": "bpm",
            "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
            "metadata": {
                "battery_level": 90,
                "signal_quality": "excellent",
                "device_model": "Test Device Pro",
                "test_flow": True
            }
        }
        
        logger.info(f"Test data: {test_device_data['device_id']} - {test_device_data['reading_type']}: {test_device_data['value']} {test_device_data['unit']}")
        
        # Step 2: Detect vendor and route
        logger.info("🔍 Step 2: Detecting vendor and routing...")
        detection_result = await vendor_detection_service.detect_vendor_and_route(test_device_data)
        
        logger.info(f"Detection result: {detection_result.vendor_id} ({detection_result.confidence:.2f}) via {detection_result.detection_method}")
        
        # Step 3: Store in outbox
        logger.info("💾 Step 3: Storing in outbox...")
        outbox_service = VendorAwareOutboxService()
        
        correlation_id = f"test_flow_{int(datetime.utcnow().timestamp())}"
        trace_id = f"trace_{correlation_id}"
        
        outbox_id = await outbox_service.store_device_data_transactionally(
            device_data=test_device_data,
            vendor_id=detection_result.vendor_id,
            correlation_id=correlation_id,
            trace_id=trace_id
        )
        
        if outbox_id:
            logger.info(f"✅ Stored in outbox: {outbox_id}")
        else:
            logger.error("❌ Failed to store in outbox")
            return
        
        # Step 4: Check queue status
        logger.info("📊 Step 4: Checking queue status...")
        queue_depths = await outbox_service.get_queue_depths()
        logger.info(f"Queue depths: {queue_depths}")
        
        vendor_pending = queue_depths.get(detection_result.vendor_id, 0)
        if vendor_pending == 0:
            logger.warning(f"⚠️ No pending messages found for {detection_result.vendor_id}")
            return
        
        logger.info(f"✅ Found {vendor_pending} pending messages for {detection_result.vendor_id}")
        
        # Step 5: Process with background publisher
        logger.info("⚙️ Step 5: Processing with background publisher...")
        publisher = BackgroundPublisher()
        
        initialized = await publisher.initialize()
        if not initialized:
            logger.error("❌ Failed to initialize background publisher")
            return
        
        logger.info("✅ Background publisher initialized")
        
        # Get vendor config for processing
        vendor_config = outbox_service.vendor_registry.get(detection_result.vendor_id)
        if not vendor_config:
            logger.error(f"❌ No config found for {detection_result.vendor_id}")
            return
        
        # Add vendor_id to config
        vendor_config_with_id = vendor_config.copy()
        vendor_config_with_id["vendor_id"] = detection_result.vendor_id
        
        # Process messages
        processed_count = await publisher.process_vendor_messages(vendor_config_with_id)
        
        logger.info(f"📡 Processed {processed_count} messages")
        
        # Step 6: Verify final status
        logger.info("🔍 Step 6: Verifying final status...")
        final_queue_depths = await outbox_service.get_queue_depths()
        logger.info(f"Final queue depths: {final_queue_depths}")
        
        final_vendor_pending = final_queue_depths.get(detection_result.vendor_id, 0)
        
        # Summary
        logger.info("")
        logger.info("=" * 70)
        logger.info("📋 COMPLETE FLOW TEST SUMMARY")
        logger.info("=" * 70)
        logger.info(f"Device ID: {test_device_data['device_id']}")
        logger.info(f"Detected Vendor: {detection_result.vendor_id} ({detection_result.confidence:.2f})")
        logger.info(f"Outbox ID: {outbox_id}")
        logger.info(f"Correlation ID: {correlation_id}")
        logger.info(f"Initial pending: {vendor_pending}")
        logger.info(f"Processed: {processed_count}")
        logger.info(f"Final pending: {final_vendor_pending}")
        
        if processed_count > 0:
            logger.info("")
            logger.info("🎉 SUCCESS: Complete end-to-end flow working!")
            logger.info("✅ Phase 6: Background Publishing - COMPLETE")
            logger.info("✅ Phase 7: End-to-End Kafka - COMPLETE")
            logger.info("")
            logger.info("🔍 VERIFICATION STEPS:")
            logger.info("1. Check Confluent Cloud Console:")
            logger.info("   - Topic: raw-device-data.v1")
            logger.info("   - Look for message with your correlation_id")
            logger.info("2. Message should contain:")
            logger.info(f"   - headers.vendor_id: {detection_result.vendor_id}")
            logger.info(f"   - headers.correlation_id: {correlation_id}")
            logger.info(f"   - payload.device_id: {test_device_data['device_id']}")
            logger.info("3. Verify message format for ETL consumption")
        else:
            logger.warning("⚠️ No messages were processed")
            logger.info("🔧 Check Kafka connectivity")
        
        logger.info("=" * 70)
        
    except Exception as e:
        logger.error(f"❌ Complete flow test failed: {e}", exc_info=True)


if __name__ == "__main__":
    asyncio.run(test_complete_flow())
