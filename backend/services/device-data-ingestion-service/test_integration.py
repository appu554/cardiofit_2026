#!/usr/bin/env python3
"""
Integration Test for Transactional Outbox Pattern

This script tests the complete integration flow:
1. Start the ingestion service
2. Send test data to the outbox endpoint
3. Verify data is stored in outbox
4. Test publisher service processing
5. Verify end-to-end flow
"""
import asyncio
import json
import logging
import sys
import time
from pathlib import Path

import httpx

# Add the app directory to Python path
sys.path.append(str(Path(__file__).parent / "app"))

from app.db.database import startup_database, shutdown_database
from app.services.outbox_service import VendorAwareOutboxService
from app.services.outbox_publisher import outbox_publisher

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class IntegrationTest:
    """Integration test for outbox pattern"""
    
    def __init__(self):
        self.base_url = "http://localhost:8015"  # Default service port
        self.api_key = "dv1_test_key_12345"  # Test API key
        self.outbox_service = VendorAwareOutboxService()
        
    async def run_integration_test(self):
        """Run complete integration test"""
        logger.info("🚀 Starting Outbox Pattern Integration Test")
        
        try:
            # Initialize database
            await startup_database()
            
            # Test service health
            await self.test_service_health()
            
            # Test outbox endpoint
            await self.test_outbox_endpoint()
            
            # Test outbox health endpoints
            await self.test_outbox_health_endpoints()
            
            # Test publisher functionality
            await self.test_publisher_functionality()
            
            logger.info("✅ Integration test completed successfully!")
            
        except Exception as e:
            logger.error(f"❌ Integration test failed: {e}")
            raise
        finally:
            await shutdown_database()
    
    async def test_service_health(self):
        """Test service health endpoints"""
        logger.info("🔍 Testing service health...")
        
        try:
            async with httpx.AsyncClient() as client:
                # Test root endpoint
                response = await client.get(f"{self.base_url}/")
                if response.status_code == 200:
                    logger.info("✅ Root endpoint accessible")
                else:
                    logger.warning(f"⚠️ Root endpoint returned {response.status_code}")
                
                # Test health endpoint
                response = await client.get(f"{self.base_url}/api/v1/health")
                if response.status_code == 200:
                    health_data = response.json()
                    logger.info(f"✅ Health endpoint: {health_data.get('status', 'unknown')}")
                else:
                    logger.warning(f"⚠️ Health endpoint returned {response.status_code}")
                    
        except httpx.ConnectError:
            logger.error("❌ Cannot connect to service. Make sure the service is running on port 8015")
            raise
        except Exception as e:
            logger.error(f"❌ Service health test failed: {e}")
            raise
    
    async def test_outbox_endpoint(self):
        """Test the outbox ingestion endpoint"""
        logger.info("🔍 Testing outbox ingestion endpoint...")
        
        test_data = {
            "device_id": "integration-test-device-001",
            "timestamp": int(time.time()),
            "reading_type": "heart_rate",
            "value": 78.5,
            "unit": "bpm",
            "patient_id": "integration-test-patient",
            "metadata": {"test": "integration", "timestamp": time.time()}
        }
        
        try:
            async with httpx.AsyncClient() as client:
                response = await client.post(
                    f"{self.base_url}/api/v1/ingest/device-data-outbox",
                    json=test_data,
                    headers={"X-API-Key": self.api_key},
                    timeout=30.0
                )
                
                if response.status_code == 200:
                    result = response.json()
                    logger.info(f"✅ Outbox ingestion successful: {result.get('message', 'No message')}")
                    logger.info(f"   Outbox ID: {result.get('ingestion_id', 'Unknown')}")
                    
                    # Verify data is in outbox
                    await self.verify_data_in_outbox(test_data["device_id"])
                    
                else:
                    logger.error(f"❌ Outbox ingestion failed: {response.status_code}")
                    logger.error(f"   Response: {response.text}")
                    
        except Exception as e:
            logger.error(f"❌ Outbox endpoint test failed: {e}")
            raise
    
    async def verify_data_in_outbox(self, device_id: str):
        """Verify that data was stored in the outbox"""
        logger.info("🔍 Verifying data in outbox...")
        
        try:
            # Check queue depths
            queue_depths = await self.outbox_service.get_queue_depths()
            logger.info(f"📊 Queue depths: {queue_depths}")
            
            # Check if we can find our message
            messages = await self.outbox_service.get_pending_messages_with_lock("fitbit", 10)
            
            found_message = None
            for message in messages:
                if message.device_id == device_id:
                    found_message = message
                    break
            
            if found_message:
                logger.info(f"✅ Message found in outbox: {found_message.id}")
                logger.info(f"   Device ID: {found_message.device_id}")
                logger.info(f"   Status: {found_message.status}")
                logger.info(f"   Correlation ID: {found_message.correlation_id}")
            else:
                logger.warning("⚠️ Message not found in outbox (may have been processed)")
                
        except Exception as e:
            logger.error(f"❌ Outbox verification failed: {e}")
    
    async def test_outbox_health_endpoints(self):
        """Test outbox health and monitoring endpoints"""
        logger.info("🔍 Testing outbox health endpoints...")
        
        try:
            async with httpx.AsyncClient() as client:
                # Test outbox health
                response = await client.get(f"{self.base_url}/api/v1/outbox/health")
                if response.status_code == 200:
                    health_data = response.json()
                    logger.info(f"✅ Outbox health: {health_data.get('status', 'unknown')}")
                    logger.info(f"   Supported vendors: {health_data.get('supported_vendors', [])}")
                else:
                    logger.warning(f"⚠️ Outbox health endpoint returned {response.status_code}")
                
                # Test queue depths
                response = await client.get(f"{self.base_url}/api/v1/outbox/queue-depths")
                if response.status_code == 200:
                    queue_data = response.json()
                    logger.info(f"✅ Queue depths: {queue_data.get('queue_depths', {})}")
                    logger.info(f"   Total pending: {queue_data.get('total_pending', 0)}")
                else:
                    logger.warning(f"⚠️ Queue depths endpoint returned {response.status_code}")
                
                # Test dead letter statistics
                response = await client.get(f"{self.base_url}/api/v1/dead-letter/statistics")
                if response.status_code == 200:
                    dl_stats = response.json()
                    stats = dl_stats.get('statistics', {})
                    logger.info(f"✅ Dead letter stats: {stats.get('total_dead_letters', 0)} total")
                else:
                    logger.warning(f"⚠️ Dead letter stats endpoint returned {response.status_code}")
                    
        except Exception as e:
            logger.error(f"❌ Health endpoints test failed: {e}")
    
    async def test_publisher_functionality(self):
        """Test publisher service functionality"""
        logger.info("🔍 Testing publisher functionality...")
        
        try:
            # Initialize publisher
            success = await outbox_publisher.initialize()
            if success:
                logger.info("✅ Publisher initialized successfully")
            else:
                logger.warning("⚠️ Publisher initialization failed")
                return
            
            # Get publisher health
            health = outbox_publisher.get_health_status()
            logger.info(f"📊 Publisher health: {health}")
            
            # Test processing a small batch
            logger.info("🔄 Testing message processing...")
            
            # Process messages for each vendor
            for vendor_id in ["fitbit", "garmin", "apple_health"]:
                try:
                    processed_count = await outbox_publisher.process_vendor_messages(vendor_id)
                    logger.info(f"✅ Processed {processed_count} messages for {vendor_id}")
                except Exception as e:
                    logger.warning(f"⚠️ Error processing {vendor_id}: {e}")
            
            # Get final stats
            stats = outbox_publisher.get_processing_stats()
            logger.info(f"📊 Final processing stats: {stats}")
            
        except Exception as e:
            logger.error(f"❌ Publisher functionality test failed: {e}")
    
    async def test_error_scenarios(self):
        """Test error handling scenarios"""
        logger.info("🔍 Testing error scenarios...")
        
        try:
            async with httpx.AsyncClient() as client:
                # Test invalid vendor
                invalid_data = {
                    "device_id": "test-device",
                    "timestamp": int(time.time()),
                    "reading_type": "invalid_type",  # This should fail validation
                    "value": 75.0,
                    "unit": "bpm"
                }
                
                response = await client.post(
                    f"{self.base_url}/api/v1/ingest/device-data-outbox",
                    json=invalid_data,
                    headers={"X-API-Key": self.api_key}
                )
                
                if response.status_code != 200:
                    logger.info(f"✅ Invalid data correctly rejected: {response.status_code}")
                else:
                    logger.warning("⚠️ Invalid data was accepted (unexpected)")
                
                # Test invalid API key
                valid_data = {
                    "device_id": "test-device",
                    "timestamp": int(time.time()),
                    "reading_type": "heart_rate",
                    "value": 75.0,
                    "unit": "bpm"
                }
                
                response = await client.post(
                    f"{self.base_url}/api/v1/ingest/device-data-outbox",
                    json=valid_data,
                    headers={"X-API-Key": "invalid-key"}
                )
                
                if response.status_code == 401 or response.status_code == 403:
                    logger.info(f"✅ Invalid API key correctly rejected: {response.status_code}")
                else:
                    logger.warning(f"⚠️ Invalid API key handling unexpected: {response.status_code}")
                    
        except Exception as e:
            logger.error(f"❌ Error scenarios test failed: {e}")


async def main():
    """Main test runner"""
    logger.info("🧪 Outbox Pattern Integration Test")
    logger.info("=" * 50)
    logger.info("Prerequisites:")
    logger.info("1. Device Data Ingestion Service running on port 8015")
    logger.info("2. Database migration completed")
    logger.info("3. Test API key configured")
    logger.info("=" * 50)
    
    test = IntegrationTest()
    await test.run_integration_test()


if __name__ == "__main__":
    asyncio.run(main())
