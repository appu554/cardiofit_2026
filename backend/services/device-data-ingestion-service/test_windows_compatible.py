#!/usr/bin/env python3
"""
Windows-Compatible Outbox Pattern Test

This test works on Windows without Unicode issues and tests the outbox
pattern logic without requiring database connectivity.
"""
import asyncio
import json
import logging
import sys
import time
from pathlib import Path
from typing import Dict, Any

# Add the app directory to Python path
sys.path.append(str(Path(__file__).parent / "app"))

# Configure logging for Windows compatibility
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s',
    handlers=[logging.StreamHandler(sys.stdout)]
)
logger = logging.getLogger(__name__)


class WindowsCompatibleOutboxTest:
    """Windows-compatible test for outbox pattern"""
    
    def __init__(self):
        # Test device data samples
        self.test_devices = {
            "fitbit_heart_rate": {
                "device_id": "fitbit_charge5_001",
                "reading_type": "heart_rate",
                "value": 75.5,
                "unit": "bpm",
                "timestamp": int(time.time()),
                "patient_id": "test-patient-123",
                "metadata": {"vendor": "fitbit", "medical_grade": False}
            },
            "omron_blood_pressure": {
                "device_id": "omron_bp7000_001", 
                "reading_type": "blood_pressure",
                "systolic": 120,
                "diastolic": 80,
                "unit": "mmHg",
                "timestamp": int(time.time()),
                "patient_id": "test-patient-123",
                "metadata": {"vendor": "omron", "medical_grade": True}
            },
            "medical_glucose": {
                "device_id": "hospital_glucose_001",
                "reading_type": "blood_glucose",
                "value": 95.0,
                "unit": "mg/dL",
                "timestamp": int(time.time()),
                "patient_id": "test-patient-123",
                "metadata": {"medical_grade": True, "hospital_device": True}
            },
            "apple_ecg": {
                "device_id": "apple_watch_001",
                "reading_type": "ecg",
                "waveform": [0.1, 0.2, 0.8, 0.1, -0.1, 0.0],
                "heart_rate": 72,
                "unit": "mV", 
                "timestamp": int(time.time()),
                "patient_id": "test-patient-123",
                "metadata": {"vendor": "apple", "medical_grade": True}
            },
            "unknown_steps": {
                "device_id": "unknown_tracker_001",
                "reading_type": "steps",
                "value": 8500,
                "unit": "steps",
                "timestamp": int(time.time()),
                "patient_id": "test-patient-123",
                "metadata": {}
            }
        }
        
        self.test_results = {}
    
    async def run_tests(self):
        """Run all Windows-compatible tests"""
        logger.info("Starting Windows-Compatible Outbox Pattern Tests")
        logger.info("=" * 60)
        
        try:
            # Test 1: Import validation
            await self.test_imports()
            
            # Test 2: Configuration validation
            await self.test_configuration()
            
            # Test 3: Vendor detection logic
            await self.test_vendor_detection_logic()
            
            # Test 4: Outbox table mapping
            await self.test_outbox_table_mapping()
            
            # Test 5: Service health (without database)
            await self.test_service_health_offline()
            
            # Print results
            self.print_results()
            
        except Exception as e:
            logger.error(f"Test failed: {e}")
            raise
    
    async def test_imports(self):
        """Test that all modules can be imported"""
        logger.info("Testing module imports...")
        
        try:
            # Test core imports
            from app.config import settings
            logger.info(f"SUCCESS: Config loaded - Database: {settings.DATABASE_URL[:30]}...")
            self.test_results["config_import"] = True
            
            # Test models import
            from app.db.models import SUPPORTED_VENDORS, is_supported_vendor
            vendor_count = len(SUPPORTED_VENDORS)
            logger.info(f"SUCCESS: Models loaded - {vendor_count} supported vendors")
            self.test_results["models_import"] = True
            self.test_results["vendor_count"] = vendor_count
            
            # List supported vendors
            logger.info("Supported vendors:")
            for vendor_id, config in SUPPORTED_VENDORS.items():
                device_types = len(config.get("device_types", []))
                logger.info(f"  - {vendor_id}: {device_types} device types")
            
        except Exception as e:
            logger.error(f"FAILED: Import test failed: {e}")
            self.test_results["imports"] = False
    
    async def test_configuration(self):
        """Test configuration values"""
        logger.info("Testing configuration...")
        
        try:
            from app.config import settings
            
            # Test required settings
            config_tests = {
                "database_url": bool(settings.DATABASE_URL),
                "kafka_servers": bool(settings.KAFKA_BOOTSTRAP_SERVERS),
                "kafka_api_key": bool(settings.KAFKA_API_KEY),
                "outbox_batch_size": settings.OUTBOX_BATCH_SIZE > 0,
                "poll_interval": settings.OUTBOX_POLL_INTERVAL > 0
            }
            
            for test_name, result in config_tests.items():
                status = "SUCCESS" if result else "FAILED"
                logger.info(f"{status}: {test_name}")
                self.test_results[f"config_{test_name}"] = result
            
            logger.info(f"Configuration summary:")
            logger.info(f"  - Service Port: {settings.PORT}")
            logger.info(f"  - Batch Size: {settings.OUTBOX_BATCH_SIZE}")
            logger.info(f"  - Poll Interval: {settings.OUTBOX_POLL_INTERVAL}s")
            logger.info(f"  - Max Concurrent Vendors: {settings.MAX_CONCURRENT_VENDORS}")
            
        except Exception as e:
            logger.error(f"FAILED: Configuration test failed: {e}")
            self.test_results["configuration"] = False
    
    async def test_vendor_detection_logic(self):
        """Test vendor detection without database"""
        logger.info("Testing vendor detection logic...")
        
        try:
            # Import vendor detection service
            from app.services.vendor_detection import VendorDetectionService
            
            # Create service instance
            detection_service = VendorDetectionService()
            
            # Test each device sample
            for device_name, device_data in self.test_devices.items():
                try:
                    # Use the internal detection logic
                    detection_result = detection_service.detect_vendor(device_data)
                    
                    vendor_id = detection_result["vendor_id"]
                    confidence = detection_result["confidence"]
                    method = detection_result["method"]
                    outbox_table = detection_result["outbox_table"]
                    
                    logger.info(f"SUCCESS: {device_name}")
                    logger.info(f"  Vendor: {vendor_id} (confidence: {confidence:.2f})")
                    logger.info(f"  Method: {method}")
                    logger.info(f"  Outbox Table: {outbox_table}")
                    
                    self.test_results[f"detection_{device_name}"] = {
                        "success": True,
                        "vendor_id": vendor_id,
                        "confidence": confidence,
                        "method": method,
                        "outbox_table": outbox_table
                    }
                    
                except Exception as e:
                    logger.error(f"FAILED: Detection failed for {device_name}: {e}")
                    self.test_results[f"detection_{device_name}"] = {
                        "success": False,
                        "error": str(e)
                    }
            
        except Exception as e:
            logger.error(f"FAILED: Vendor detection test failed: {e}")
            self.test_results["vendor_detection"] = False
    
    async def test_outbox_table_mapping(self):
        """Test outbox table mapping logic"""
        logger.info("Testing outbox table mapping...")
        
        try:
            from app.db.models import SUPPORTED_VENDORS
            
            # Test table mapping for each vendor
            table_mappings = {}
            for vendor_id, config in SUPPORTED_VENDORS.items():
                outbox_table = config["outbox_table"]
                dead_letter_table = config["dead_letter_table"]
                device_types = config.get("device_types", [])
                
                table_mappings[vendor_id] = {
                    "outbox_table": outbox_table,
                    "dead_letter_table": dead_letter_table,
                    "device_types": device_types
                }
                
                logger.info(f"SUCCESS: {vendor_id}")
                logger.info(f"  Outbox: {outbox_table}")
                logger.info(f"  Dead Letter: {dead_letter_table}")
                logger.info(f"  Device Types: {len(device_types)}")
            
            self.test_results["table_mapping"] = {
                "success": True,
                "mappings": table_mappings
            }
            
        except Exception as e:
            logger.error(f"FAILED: Table mapping test failed: {e}")
            self.test_results["table_mapping"] = False
    
    async def test_service_health_offline(self):
        """Test service health without database connection"""
        logger.info("Testing service health (offline mode)...")
        
        try:
            # Test that services can be instantiated
            from app.services.outbox_service import VendorAwareOutboxService
            from app.services.vendor_detection import vendor_detection_service
            
            # Create service instances
            outbox_service = VendorAwareOutboxService()
            logger.info("SUCCESS: Outbox service instantiated")
            
            # Test vendor detection service
            capabilities = await vendor_detection_service.list_all_supported_vendors()
            logger.info(f"SUCCESS: Vendor detection service - {len(capabilities)} vendors")
            
            self.test_results["service_health"] = {
                "success": True,
                "outbox_service": True,
                "vendor_detection": True,
                "vendor_capabilities_count": len(capabilities)
            }
            
        except Exception as e:
            logger.error(f"FAILED: Service health test failed: {e}")
            self.test_results["service_health"] = False
    
    def print_results(self):
        """Print test results in Windows-compatible format"""
        logger.info("")
        logger.info("=" * 60)
        logger.info("WINDOWS-COMPATIBLE OUTBOX PATTERN TEST RESULTS")
        logger.info("=" * 60)
        
        # Import tests
        logger.info("")
        logger.info("MODULE IMPORTS:")
        config_status = "PASS" if self.test_results.get("config_import") else "FAIL"
        models_status = "PASS" if self.test_results.get("models_import") else "FAIL"
        logger.info(f"Config Import: {config_status}")
        logger.info(f"Models Import: {models_status}")
        logger.info(f"Vendor Count: {self.test_results.get('vendor_count', 0)}")
        
        # Configuration tests
        logger.info("")
        logger.info("CONFIGURATION:")
        config_keys = [k for k in self.test_results.keys() if k.startswith("config_")]
        for key in config_keys:
            test_name = key.replace("config_", "").replace("_", " ").title()
            status = "PASS" if self.test_results.get(key) else "FAIL"
            logger.info(f"{test_name}: {status}")
        
        # Vendor detection tests
        logger.info("")
        logger.info("VENDOR DETECTION:")
        for device_name in self.test_devices.keys():
            result = self.test_results.get(f"detection_{device_name}", {})
            if result.get("success"):
                vendor = result.get("vendor_id", "unknown")
                confidence = result.get("confidence", 0.0)
                logger.info(f"{device_name}: PASS -> {vendor} ({confidence:.2f})")
            else:
                logger.info(f"{device_name}: FAIL")
        
        # Table mapping
        logger.info("")
        logger.info("TABLE MAPPING:")
        mapping_result = self.test_results.get("table_mapping", {})
        if mapping_result.get("success"):
            mappings = mapping_result.get("mappings", {})
            logger.info(f"Table Mappings: PASS ({len(mappings)} vendors)")
        else:
            logger.info("Table Mappings: FAIL")
        
        # Service health
        logger.info("")
        logger.info("SERVICE HEALTH:")
        health_result = self.test_results.get("service_health", {})
        if health_result.get("success"):
            logger.info("Service Health: PASS")
            logger.info(f"Outbox Service: PASS")
            logger.info(f"Vendor Detection: PASS")
        else:
            logger.info("Service Health: FAIL")
        
        # Summary
        logger.info("")
        logger.info("=" * 60)
        logger.info("SUMMARY:")
        
        # Count successful tests
        total_tests = 0
        passed_tests = 0
        
        # Core tests
        core_tests = ["config_import", "models_import", "vendor_detection", "table_mapping", "service_health"]
        for test in core_tests:
            total_tests += 1
            if self.test_results.get(test, {}).get("success", self.test_results.get(test, False)):
                passed_tests += 1
        
        # Detection tests
        detection_tests = [k for k in self.test_results.keys() if k.startswith("detection_")]
        for test in detection_tests:
            total_tests += 1
            if self.test_results.get(test, {}).get("success", False):
                passed_tests += 1
        
        logger.info(f"Tests Passed: {passed_tests}/{total_tests}")
        logger.info(f"Success Rate: {(passed_tests/total_tests)*100:.1f}%")
        
        if passed_tests == total_tests:
            logger.info("RESULT: ALL TESTS PASSED!")
            logger.info("The outbox pattern is properly integrated and ready for use.")
        elif passed_tests >= total_tests * 0.8:
            logger.info("RESULT: MOSTLY SUCCESSFUL!")
            logger.info("Core functionality is working. Minor issues may need attention.")
        else:
            logger.info("RESULT: ISSUES DETECTED!")
            logger.info("Some core functionality may need fixing before production use.")
        
        logger.info("=" * 60)


async def main():
    """Main test runner"""
    test = WindowsCompatibleOutboxTest()
    await test.run_tests()


if __name__ == "__main__":
    asyncio.run(main())
