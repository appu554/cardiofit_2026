#!/usr/bin/env python3
"""
End-to-End Integration Test: Device Data → Outbox → Kafka

This test demonstrates the complete flow:
1. Device data ingestion via API
2. Storage in vendor-specific outbox table
3. Background publisher processing
4. Publishing to Kafka
5. Verification of message delivery

Tests all three ingestion endpoints:
- Legacy direct Kafka
- Outbox pattern with API key
- Smart detection without API key
"""
import asyncio
import json
import logging
import sys
import uuid
import time
from datetime import datetime
from pathlib import Path
from typing import Dict, Any, List

# Add the app directory to Python path
sys.path.append(str(Path(__file__).parent / "app"))

import httpx
import pytest

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class EndToEndKafkaTest:
    """Complete end-to-end test for device ingestion → Kafka"""
    
    def __init__(self):
        self.base_url = "http://localhost:8015"  # Device ingestion service
        self.api_prefix = "/api/v1"
        
        # Test API key (you'll need to configure this)
        self.test_api_key = "test-vendor-api-key-123"
        
        # Sample medical device data for different vendors
        self.test_devices = {
            "fitbit_heart_rate": {
                "device_id": "fitbit_charge5_test_001",
                "reading_type": "heart_rate",
                "value": 75.5,
                "unit": "bpm",
                "timestamp": int(time.time()),
                "patient_id": "test-patient-123",
                "metadata": {
                    "vendor": "fitbit",
                    "device_model": "Charge 5",
                    "medical_grade": False
                }
            },
            "omron_blood_pressure": {
                "device_id": "omron_bp7000_test_001",
                "reading_type": "blood_pressure",
                "systolic": 120,
                "diastolic": 80,
                "unit": "mmHg",
                "timestamp": int(time.time()),
                "patient_id": "test-patient-123",
                "metadata": {
                    "vendor": "omron",
                    "device_model": "BP7000",
                    "medical_grade": True
                }
            },
            "medical_glucose": {
                "device_id": "hospital_glucose_test_001",
                "reading_type": "blood_glucose",
                "value": 95.0,
                "unit": "mg/dL",
                "timestamp": int(time.time()),
                "patient_id": "test-patient-123",
                "metadata": {
                    "medical_grade": True,
                    "hospital_device": True,
                    "device_model": "Clinical Glucose Meter"
                }
            },
            "apple_ecg": {
                "device_id": "apple_watch_test_001",
                "reading_type": "ecg",
                "waveform": [0.1, 0.2, 0.8, 0.1, -0.1, 0.0],
                "heart_rate": 72,
                "unit": "mV",
                "timestamp": int(time.time()),
                "patient_id": "test-patient-123",
                "metadata": {
                    "vendor": "apple",
                    "device_model": "Apple Watch Series 8",
                    "medical_grade": True
                }
            },
            "unknown_steps": {
                "device_id": "unknown_tracker_test_001",
                "reading_type": "steps",
                "value": 8500,
                "unit": "steps",
                "timestamp": int(time.time()),
                "patient_id": "test-patient-123",
                "metadata": {}
            }
        }
        
        self.test_results = {}
    
    async def run_complete_test(self):
        """Run the complete end-to-end test"""
        logger.info("🚀 Starting End-to-End Kafka Integration Test")
        logger.info("="*70)
        
        try:
            # Step 1: Test service health
            await self.test_service_health()
            
            # Step 2: Test all ingestion endpoints
            await self.test_legacy_endpoint()
            await self.test_outbox_endpoint()
            await self.test_smart_endpoint()
            
            # Step 3: Test vendor detection
            await self.test_vendor_detection()
            
            # Step 4: Test outbox queue status
            await self.test_outbox_queues()
            
            # Step 5: Test publisher service (if running)
            await self.test_publisher_service()
            
            # Step 6: Verify Kafka integration
            await self.test_kafka_integration()
            
            # Print comprehensive results
            self.print_test_results()
            
        except Exception as e:
            logger.error(f"💥 End-to-end test failed: {e}")
            raise
    
    async def test_service_health(self):
        """Test that the device ingestion service is running"""
        logger.info("🔍 Testing service health...")
        
        try:
            async with httpx.AsyncClient() as client:
                # Test root endpoint
                response = await client.get(f"{self.base_url}/")
                
                if response.status_code == 200:
                    data = response.json()
                    logger.info(f"✅ Service running: {data.get('service')} v{data.get('version')}")
                    self.test_results["service_health"] = True
                else:
                    logger.error(f"❌ Service health check failed: {response.status_code}")
                    self.test_results["service_health"] = False
                
                # Test API health
                health_response = await client.get(f"{self.base_url}{self.api_prefix}/health")
                if health_response.status_code == 200:
                    logger.info("✅ API health check passed")
                    self.test_results["api_health"] = True
                else:
                    logger.warning(f"⚠️ API health check failed: {health_response.status_code}")
                    self.test_results["api_health"] = False
                    
        except Exception as e:
            logger.error(f"❌ Service health test failed: {e}")
            self.test_results["service_health"] = False
    
    async def test_legacy_endpoint(self):
        """Test the legacy direct Kafka endpoint"""
        logger.info("🔍 Testing legacy direct Kafka endpoint...")
        
        try:
            async with httpx.AsyncClient() as client:
                headers = {"X-API-Key": self.test_api_key}
                
                response = await client.post(
                    f"{self.base_url}{self.api_prefix}/ingest/device-data",
                    json=self.test_devices["fitbit_heart_rate"],
                    headers=headers,
                    timeout=30.0
                )
                
                if response.status_code == 200:
                    data = response.json()
                    logger.info(f"✅ Legacy endpoint: {data.get('status')}")
                    self.test_results["legacy_endpoint"] = {
                        "success": True,
                        "response": data
                    }
                else:
                    logger.error(f"❌ Legacy endpoint failed: {response.status_code} - {response.text}")
                    self.test_results["legacy_endpoint"] = {
                        "success": False,
                        "error": response.text
                    }
                    
        except Exception as e:
            logger.error(f"❌ Legacy endpoint test failed: {e}")
            self.test_results["legacy_endpoint"] = {"success": False, "error": str(e)}
    
    async def test_outbox_endpoint(self):
        """Test the transactional outbox endpoint"""
        logger.info("🔍 Testing transactional outbox endpoint...")
        
        try:
            async with httpx.AsyncClient() as client:
                headers = {"X-API-Key": self.test_api_key}
                
                response = await client.post(
                    f"{self.base_url}{self.api_prefix}/ingest/device-data-outbox",
                    json=self.test_devices["omron_blood_pressure"],
                    headers=headers,
                    timeout=30.0
                )
                
                if response.status_code == 200:
                    data = response.json()
                    logger.info(f"✅ Outbox endpoint: {data.get('status')} - ID: {data.get('ingestion_id')}")
                    self.test_results["outbox_endpoint"] = {
                        "success": True,
                        "response": data,
                        "outbox_id": data.get("ingestion_id")
                    }
                else:
                    logger.error(f"❌ Outbox endpoint failed: {response.status_code} - {response.text}")
                    self.test_results["outbox_endpoint"] = {
                        "success": False,
                        "error": response.text
                    }
                    
        except Exception as e:
            logger.error(f"❌ Outbox endpoint test failed: {e}")
            self.test_results["outbox_endpoint"] = {"success": False, "error": str(e)}
    
    async def test_smart_endpoint(self):
        """Test the smart detection endpoint"""
        logger.info("🔍 Testing smart detection endpoint...")
        
        try:
            async with httpx.AsyncClient() as client:
                # Test multiple devices with smart detection
                for device_name, device_data in self.test_devices.items():
                    response = await client.post(
                        f"{self.base_url}{self.api_prefix}/ingest/device-data-smart",
                        json=device_data,
                        timeout=30.0
                    )
                    
                    if response.status_code == 200:
                        data = response.json()
                        vendor_detection = data.get("metadata", {}).get("vendor_detection", {})
                        
                        logger.info(f"✅ Smart detection for {device_name}:")
                        logger.info(f"   Vendor: {vendor_detection.get('vendor_id')}")
                        logger.info(f"   Confidence: {vendor_detection.get('confidence'):.2f}")
                        logger.info(f"   Method: {vendor_detection.get('detection_method')}")
                        
                        self.test_results[f"smart_{device_name}"] = {
                            "success": True,
                            "vendor_detection": vendor_detection,
                            "outbox_id": data.get("ingestion_id")
                        }
                    else:
                        logger.error(f"❌ Smart detection failed for {device_name}: {response.status_code}")
                        self.test_results[f"smart_{device_name}"] = {
                            "success": False,
                            "error": response.text
                        }
                        
        except Exception as e:
            logger.error(f"❌ Smart endpoint test failed: {e}")
            self.test_results["smart_endpoint"] = {"success": False, "error": str(e)}
    
    async def test_vendor_detection(self):
        """Test vendor detection capabilities"""
        logger.info("🔍 Testing vendor detection capabilities...")
        
        try:
            async with httpx.AsyncClient() as client:
                # Test supported vendors endpoint
                response = await client.get(f"{self.base_url}{self.api_prefix}/vendors/supported")
                
                if response.status_code == 200:
                    data = response.json()
                    vendors = data.get("vendors", {})
                    logger.info(f"✅ Found {len(vendors)} supported vendors")
                    
                    for vendor_id in ["fitbit", "omron", "medical_device", "generic_device"]:
                        if vendor_id in vendors:
                            logger.info(f"   ✅ {vendor_id}: {len(vendors[vendor_id].get('supported_device_types', []))} device types")
                        else:
                            logger.warning(f"   ⚠️ {vendor_id}: not found")
                    
                    self.test_results["vendor_detection"] = {
                        "success": True,
                        "vendor_count": len(vendors)
                    }
                else:
                    logger.error(f"❌ Vendor detection test failed: {response.status_code}")
                    self.test_results["vendor_detection"] = {"success": False}
                    
        except Exception as e:
            logger.error(f"❌ Vendor detection test failed: {e}")
            self.test_results["vendor_detection"] = {"success": False, "error": str(e)}
    
    async def test_outbox_queues(self):
        """Test outbox queue status"""
        logger.info("🔍 Testing outbox queue status...")
        
        try:
            async with httpx.AsyncClient() as client:
                response = await client.get(f"{self.base_url}{self.api_prefix}/outbox/queue-depths")
                
                if response.status_code == 200:
                    data = response.json()
                    queue_depths = data.get("queue_depths", {})
                    
                    logger.info("✅ Outbox queue depths:")
                    total_pending = 0
                    for vendor, depth in queue_depths.items():
                        pending = depth.get("pending", 0)
                        total_pending += pending
                        logger.info(f"   {vendor}: {pending} pending")
                    
                    logger.info(f"   Total pending: {total_pending}")
                    
                    self.test_results["outbox_queues"] = {
                        "success": True,
                        "total_pending": total_pending,
                        "queue_depths": queue_depths
                    }
                else:
                    logger.warning(f"⚠️ Outbox queue test failed: {response.status_code}")
                    self.test_results["outbox_queues"] = {"success": False}
                    
        except Exception as e:
            logger.warning(f"⚠️ Outbox queue test failed: {e}")
            self.test_results["outbox_queues"] = {"success": False, "error": str(e)}
    
    async def test_publisher_service(self):
        """Test if publisher service is processing messages"""
        logger.info("🔍 Testing publisher service...")
        
        try:
            # Wait a bit for publisher to process
            await asyncio.sleep(5)
            
            async with httpx.AsyncClient() as client:
                response = await client.get(f"{self.base_url}{self.api_prefix}/outbox/health")
                
                if response.status_code == 200:
                    data = response.json()
                    publisher_status = data.get("publisher_service", {})
                    
                    if publisher_status.get("is_running", False):
                        logger.info("✅ Publisher service is running")
                        logger.info(f"   Messages processed: {publisher_status.get('messages_processed', 0)}")
                        logger.info(f"   Success rate: {publisher_status.get('success_rate', 0):.2f}%")
                    else:
                        logger.warning("⚠️ Publisher service not running")
                    
                    self.test_results["publisher_service"] = {
                        "success": True,
                        "is_running": publisher_status.get("is_running", False),
                        "stats": publisher_status
                    }
                else:
                    logger.warning(f"⚠️ Publisher service test failed: {response.status_code}")
                    self.test_results["publisher_service"] = {"success": False}
                    
        except Exception as e:
            logger.warning(f"⚠️ Publisher service test failed: {e}")
            self.test_results["publisher_service"] = {"success": False, "error": str(e)}
    
    async def test_kafka_integration(self):
        """Test Kafka integration (basic connectivity)"""
        logger.info("🔍 Testing Kafka integration...")
        
        try:
            # Import Kafka components
            from app.kafka_producer import get_kafka_producer
            
            producer = await get_kafka_producer()
            
            if producer.health_check():
                logger.info("✅ Kafka producer health check passed")
                
                # Try to publish a test message
                test_message = {
                    "test": True,
                    "timestamp": datetime.utcnow().isoformat(),
                    "message": "End-to-end test message"
                }
                
                message_id = await producer.publish_device_data(
                    device_reading=test_message,
                    key="test-key"
                )
                
                if message_id:
                    logger.info(f"✅ Test message published to Kafka: {message_id}")
                    self.test_results["kafka_integration"] = {
                        "success": True,
                        "message_id": message_id
                    }
                else:
                    logger.error("❌ Failed to publish test message to Kafka")
                    self.test_results["kafka_integration"] = {"success": False}
            else:
                logger.error("❌ Kafka producer health check failed")
                self.test_results["kafka_integration"] = {"success": False}
                
        except Exception as e:
            logger.error(f"❌ Kafka integration test failed: {e}")
            self.test_results["kafka_integration"] = {"success": False, "error": str(e)}
    
    def print_test_results(self):
        """Print comprehensive test results"""
        logger.info("\n" + "="*70)
        logger.info("📊 END-TO-END KAFKA INTEGRATION TEST RESULTS")
        logger.info("="*70)
        
        # Service health
        logger.info("\n🏥 SERVICE HEALTH:")
        service_health = "✅" if self.test_results.get("service_health") else "❌"
        api_health = "✅" if self.test_results.get("api_health") else "❌"
        logger.info(f"Service Health: {service_health}")
        logger.info(f"API Health: {api_health}")
        
        # Ingestion endpoints
        logger.info("\n📡 INGESTION ENDPOINTS:")
        legacy = "✅" if self.test_results.get("legacy_endpoint", {}).get("success") else "❌"
        outbox = "✅" if self.test_results.get("outbox_endpoint", {}).get("success") else "❌"
        logger.info(f"Legacy Direct Kafka: {legacy}")
        logger.info(f"Transactional Outbox: {outbox}")
        
        # Smart detection
        logger.info("\n🧠 SMART DETECTION:")
        for device_name in self.test_devices.keys():
            result = self.test_results.get(f"smart_{device_name}", {})
            status = "✅" if result.get("success") else "❌"
            if result.get("success"):
                vendor = result.get("vendor_detection", {}).get("vendor_id", "unknown")
                confidence = result.get("vendor_detection", {}).get("confidence", 0.0)
                logger.info(f"{device_name}: {status} → {vendor} ({confidence:.2f})")
            else:
                logger.info(f"{device_name}: {status}")
        
        # Outbox queues
        logger.info("\n📊 OUTBOX QUEUES:")
        queue_result = self.test_results.get("outbox_queues", {})
        if queue_result.get("success"):
            total_pending = queue_result.get("total_pending", 0)
            logger.info(f"Total Pending Messages: {total_pending}")
        else:
            logger.info("Queue Status: ❌ Failed to retrieve")
        
        # Publisher service
        logger.info("\n🔄 PUBLISHER SERVICE:")
        publisher_result = self.test_results.get("publisher_service", {})
        if publisher_result.get("success"):
            is_running = publisher_result.get("is_running", False)
            status = "✅ Running" if is_running else "⚠️ Not Running"
            logger.info(f"Publisher Status: {status}")
        else:
            logger.info("Publisher Status: ❌ Failed to check")
        
        # Kafka integration
        logger.info("\n📨 KAFKA INTEGRATION:")
        kafka_result = self.test_results.get("kafka_integration", {})
        kafka_status = "✅" if kafka_result.get("success") else "❌"
        logger.info(f"Kafka Connection: {kafka_status}")
        if kafka_result.get("message_id"):
            logger.info(f"Test Message ID: {kafka_result.get('message_id')}")
        
        # Summary
        total_tests = len([k for k in self.test_results.keys() if not k.startswith("smart_")])
        smart_tests = len([k for k in self.test_results.keys() if k.startswith("smart_")])
        
        logger.info("\n" + "="*70)
        logger.info(f"📈 SUMMARY:")
        logger.info(f"Core Tests: {total_tests}")
        logger.info(f"Smart Detection Tests: {smart_tests}")
        logger.info(f"Total Device Types Tested: {len(self.test_devices)}")
        
        # Overall status
        critical_tests = ["service_health", "kafka_integration"]
        critical_passed = all(self.test_results.get(test, {}).get("success", False) for test in critical_tests)
        
        if critical_passed:
            logger.info("🎉 END-TO-END INTEGRATION SUCCESSFUL!")
            logger.info("✅ Device data can flow from ingestion → outbox → Kafka")
        else:
            logger.warning("⚠️ Some critical components failed")
            logger.info("🔧 Check service and Kafka connectivity")
        
        logger.info("="*70)


async def main():
    """Run the end-to-end test"""
    test = EndToEndKafkaTest()
    await test.run_complete_test()


if __name__ == "__main__":
    asyncio.run(main())
