#!/usr/bin/env python3
"""
Simple Background Publisher Test

Direct approach to process outbox messages and publish to Kafka
using existing service methods.
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


async def simple_publisher_test():
    """Simple test to process outbox messages and publish to Kafka"""
    logger.info("🚀 Simple Background Publisher Test - Direct Kafka Publishing")
    logger.info("=" * 70)
    
    try:
        # Import services
        from app.services.outbox_service import VendorAwareOutboxService
        from app.kafka_producer import get_kafka_producer
        from app.db.database import get_async_session
        from sqlalchemy import text
        
        # Step 1: Check pending messages
        logger.info("📊 Step 1: Checking pending messages...")
        outbox_service = VendorAwareOutboxService()
        await outbox_service._load_vendor_registry()
        
        queue_depths = await outbox_service.get_queue_depths()
        logger.info(f"Queue depths: {queue_depths}")
        
        total_pending = sum(depth for depth in queue_depths.values() if depth > 0)
        if total_pending == 0:
            logger.warning("⚠️ No pending messages found!")
            logger.info("💡 Send a test request first")
            return
        
        logger.info(f"✅ Found {total_pending} pending messages")
        
        # Step 2: Initialize Kafka producer
        logger.info("🔧 Step 2: Initializing Kafka producer...")
        kafka_producer = await get_kafka_producer()
        logger.info("✅ Kafka producer initialized")
        
        # Step 3: Process each vendor with pending messages
        logger.info("⚙️ Step 3: Processing pending messages...")
        total_processed = 0
        
        for vendor_id, pending_count in queue_depths.items():
            if pending_count == 0:
                continue
                
            logger.info(f"🔄 Processing {vendor_id} ({pending_count} pending)...")
            
            try:
                # Get vendor config
                vendor_config = outbox_service.vendor_registry.get(vendor_id)
                if not vendor_config:
                    logger.error(f"❌ No config found for {vendor_id}")
                    continue
                
                outbox_table = vendor_config['outbox_table']
                
                # Get pending messages directly from database
                async with get_async_session() as session:
                    # Get pending messages
                    result = await session.execute(text(f"""
                        SELECT id, device_id, event_payload, kafka_topic, kafka_key, 
                               correlation_id, trace_id, created_at
                        FROM {outbox_table}
                        WHERE status = 'pending'
                        ORDER BY created_at
                        LIMIT 10
                    """))
                    
                    pending_messages = result.fetchall()
                    
                    for message in pending_messages:
                        message_id = message.id
                        
                        try:
                            # Mark as processing
                            await session.execute(text(f"""
                                UPDATE {outbox_table}
                                SET status = 'processing'
                                WHERE id = :message_id
                            """), {"message_id": message_id})
                            
                            # Prepare Kafka message
                            kafka_message = {
                                "headers": {
                                    "vendor_id": vendor_id,
                                    "message_id": str(message_id),
                                    "correlation_id": message.correlation_id,
                                    "trace_id": message.trace_id,
                                    "created_at": message.created_at.isoformat() if hasattr(message.created_at, 'isoformat') else str(message.created_at),
                                    "processed_at": datetime.utcnow().isoformat()
                                },
                                "payload": message.event_payload
                            }
                            
                            # Publish to Kafka
                            logger.info(f"📡 Publishing message {message_id} to Kafka...")
                            await kafka_producer.send_message(
                                topic=message.kafka_topic or "raw-device-data.v1",
                                key=message.kafka_key or f"{vendor_id}:{message.device_id}",
                                value=json.dumps(kafka_message)
                            )
                            
                            # Mark as completed
                            await session.execute(text(f"""
                                UPDATE {outbox_table}
                                SET status = 'completed', processed_at = NOW()
                                WHERE id = :message_id
                            """), {"message_id": message_id})
                            
                            await session.commit()
                            total_processed += 1
                            
                            logger.info(f"✅ Successfully published message {message_id}")
                            
                        except Exception as e:
                            logger.error(f"❌ Failed to process message {message_id}: {e}")
                            
                            # Mark as failed
                            await session.execute(text(f"""
                                UPDATE {outbox_table}
                                SET status = 'failed', last_error = :error
                                WHERE id = :message_id
                            """), {"message_id": message_id, "error": str(e)})
                            
                            await session.commit()
                
                logger.info(f"✅ {vendor_id}: Processed messages")
                
            except Exception as e:
                logger.error(f"❌ Error processing {vendor_id}: {e}")
        
        # Step 4: Final status
        logger.info("📈 Step 4: Checking final status...")
        final_queue_depths = await outbox_service.get_queue_depths()
        logger.info(f"Final queue depths: {final_queue_depths}")
        
        final_pending = sum(depth for depth in final_queue_depths.values() if depth > 0)
        
        # Summary
        logger.info("")
        logger.info("=" * 70)
        logger.info("📋 SIMPLE PUBLISHER TEST SUMMARY")
        logger.info("=" * 70)
        logger.info(f"Initial pending messages: {total_pending}")
        logger.info(f"Messages processed: {total_processed}")
        logger.info(f"Final pending messages: {final_pending}")
        logger.info(f"Success rate: {(total_processed/total_pending)*100:.1f}%" if total_pending > 0 else "N/A")
        
        if total_processed > 0:
            logger.info("")
            logger.info("🎉 SUCCESS: Messages published to Kafka!")
            logger.info("📡 Check your Confluent Cloud Console:")
            logger.info("   - Topic: raw-device-data.v1")
            logger.info("   - Look for messages with vendor headers")
            logger.info("   - Verify message payload contains device data")
            logger.info("")
            logger.info("🔍 NEXT STEPS:")
            logger.info("1. ✅ Phase 6: Background Publishing - COMPLETE")
            logger.info("2. 🎯 Phase 7: End-to-End Kafka - TEST IN POSTMAN")
            logger.info("3. 📊 Monitor Kafka consumer logs")
            logger.info("4. 🚀 Test ETL pipeline consumption")
        else:
            logger.warning("⚠️ No messages were processed successfully")
            logger.info("🔧 Check Kafka connectivity and configuration")
        
        logger.info("=" * 70)
        
    except Exception as e:
        logger.error(f"❌ Simple publisher test failed: {e}", exc_info=True)


if __name__ == "__main__":
    asyncio.run(simple_publisher_test())
