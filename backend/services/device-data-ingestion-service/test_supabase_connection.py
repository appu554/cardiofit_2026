#!/usr/bin/env python3
"""
Test Supabase Connection and Outbox Pattern for All Medical Devices

This script:
1. Tests Supabase PostgreSQL connection
2. Verifies outbox tables exist for all medical device types
3. Tests the complete outbox flow for each device type
4. Validates universal device handler integration
"""
import asyncio
import json
import logging
import sys
import uuid
from datetime import datetime
from pathlib import Path

# Add the app directory to Python path
sys.path.append(str(Path(__file__).parent / "app"))

from app.db.database import startup_database, shutdown_database, get_async_session
from app.services.outbox_service import VendorAwareOutboxService
from app.universal_handler.device_processor import DeviceType
from app.universal_handler.universal_handler import get_universal_handler
from sqlalchemy import text

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class SupabaseOutboxTester:
    """Comprehensive tester for Supabase connection and outbox pattern"""
    
    def __init__(self):
        self.outbox_service = VendorAwareOutboxService()
        self.test_results = {}
        
        # All medical device types from universal handler
        self.medical_device_types = [
            DeviceType.HEART_RATE,
            DeviceType.BLOOD_PRESSURE,
            DeviceType.BLOOD_GLUCOSE,
            DeviceType.WEIGHT,
            DeviceType.STEPS,
            DeviceType.SLEEP_DURATION,
            DeviceType.ECG,
            DeviceType.TEMPERATURE,
            DeviceType.OXYGEN_SATURATION
        ]
        
        # Sample device data for each type
        self.device_data_samples = {
            DeviceType.HEART_RATE: {
                "device_id": "hr-device-001",
                "reading_type": "heart_rate",
                "value": 75.5,
                "unit": "bpm",
                "timestamp": int(datetime.utcnow().timestamp()),
                "patient_id": "patient-123",
                "metadata": {"device_type": "heart_rate", "medical_grade": True}
            },
            DeviceType.BLOOD_PRESSURE: {
                "device_id": "bp-device-001",
                "reading_type": "blood_pressure",
                "systolic": 120,
                "diastolic": 80,
                "unit": "mmHg",
                "timestamp": int(datetime.utcnow().timestamp()),
                "patient_id": "patient-123",
                "metadata": {"device_type": "blood_pressure", "medical_grade": True}
            },
            DeviceType.BLOOD_GLUCOSE: {
                "device_id": "bg-device-001",
                "reading_type": "blood_glucose",
                "value": 95.0,
                "unit": "mg/dL",
                "timestamp": int(datetime.utcnow().timestamp()),
                "patient_id": "patient-123",
                "metadata": {"device_type": "blood_glucose", "medical_grade": True}
            },
            DeviceType.WEIGHT: {
                "device_id": "weight-device-001",
                "reading_type": "weight",
                "value": 70.5,
                "unit": "kg",
                "timestamp": int(datetime.utcnow().timestamp()),
                "patient_id": "patient-123",
                "metadata": {"device_type": "weight", "medical_grade": False}
            },
            DeviceType.STEPS: {
                "device_id": "steps-device-001",
                "reading_type": "steps",
                "value": 8500,
                "unit": "steps",
                "timestamp": int(datetime.utcnow().timestamp()),
                "patient_id": "patient-123",
                "metadata": {"device_type": "steps", "medical_grade": False}
            },
            DeviceType.SLEEP_DURATION: {
                "device_id": "sleep-device-001",
                "reading_type": "sleep_duration",
                "value": 7.5,
                "unit": "hours",
                "timestamp": int(datetime.utcnow().timestamp()),
                "patient_id": "patient-123",
                "metadata": {"device_type": "sleep_duration", "medical_grade": False}
            },
            DeviceType.ECG: {
                "device_id": "ecg-device-001",
                "reading_type": "ecg",
                "waveform": [0.1, 0.2, 0.8, 0.1, -0.1, 0.0],
                "heart_rate": 72,
                "unit": "mV",
                "timestamp": int(datetime.utcnow().timestamp()),
                "patient_id": "patient-123",
                "metadata": {"device_type": "ecg", "medical_grade": True}
            },
            DeviceType.TEMPERATURE: {
                "device_id": "temp-device-001",
                "reading_type": "temperature",
                "value": 36.8,
                "unit": "celsius",
                "timestamp": int(datetime.utcnow().timestamp()),
                "patient_id": "patient-123",
                "metadata": {"device_type": "temperature", "medical_grade": True}
            },
            DeviceType.OXYGEN_SATURATION: {
                "device_id": "spo2-device-001",
                "reading_type": "oxygen_saturation",
                "value": 98.5,
                "unit": "percent",
                "timestamp": int(datetime.utcnow().timestamp()),
                "patient_id": "patient-123",
                "metadata": {"device_type": "oxygen_saturation", "medical_grade": True}
            }
        }
    
    async def run_comprehensive_test(self):
        """Run all tests"""
        logger.info("🚀 Starting Comprehensive Supabase Outbox Test")
        
        try:
            # Initialize database
            await startup_database()
            
            # Test Supabase connection
            await self.test_supabase_connection()
            
            # Test outbox tables for all vendors
            await self.test_outbox_tables()
            
            # Test universal device handler integration
            await self.test_universal_device_handler()
            
            # Test outbox flow for all device types
            await self.test_outbox_flow_all_devices()
            
            # Test concurrent processing
            await self.test_concurrent_processing()
            
            # Print comprehensive results
            self.print_test_results()
            
        except Exception as e:
            logger.error(f"💥 Test suite failed: {e}")
            raise
        finally:
            await shutdown_database()
    
    async def test_supabase_connection(self):
        """Test direct Supabase PostgreSQL connection"""
        logger.info("🔍 Testing Supabase connection...")
        
        try:
            async with get_async_session() as session:
                # Test basic connectivity
                result = await session.execute(text("SELECT NOW() as current_time, version() as pg_version"))
                row = result.fetchone()
                
                current_time = row.current_time
                pg_version = row.pg_version
                
                logger.info(f"✅ Supabase connected successfully")
                logger.info(f"   Database time: {current_time}")
                logger.info(f"   PostgreSQL version: {pg_version[:50]}...")
                
                # Test database name and connection details
                result = await session.execute(text("SELECT current_database(), current_user"))
                db_info = result.fetchone()
                
                logger.info(f"   Database: {db_info.current_database}")
                logger.info(f"   User: {db_info.current_user}")
                
                self.test_results["supabase_connection"] = True
                
        except Exception as e:
            logger.error(f"❌ Supabase connection failed: {e}")
            self.test_results["supabase_connection"] = False
            raise
    
    async def test_outbox_tables(self):
        """Test that outbox tables exist for all vendors"""
        logger.info("🔍 Testing outbox tables...")
        
        vendors = ["fitbit", "garmin", "apple_health"]
        
        try:
            async with get_async_session() as session:
                for vendor in vendors:
                    # Test outbox table
                    outbox_table = f"{vendor}_outbox"
                    result = await session.execute(
                        text("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = :table_name"),
                        {"table_name": outbox_table}
                    )
                    table_exists = result.scalar() > 0
                    
                    # Test dead letter table
                    dead_letter_table = f"{vendor}_dead_letter"
                    result = await session.execute(
                        text("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = :table_name"),
                        {"table_name": dead_letter_table}
                    )
                    dl_table_exists = result.scalar() > 0
                    
                    logger.info(f"✅ {vendor}: outbox={table_exists}, dead_letter={dl_table_exists}")
                    
                    self.test_results[f"table_{vendor}_outbox"] = table_exists
                    self.test_results[f"table_{vendor}_dead_letter"] = dl_table_exists
                
                # Test vendor registry
                result = await session.execute(
                    text("SELECT COUNT(*) FROM vendor_outbox_registry WHERE is_active = true")
                )
                registry_count = result.scalar()
                
                logger.info(f"✅ Vendor registry: {registry_count} active vendors")
                self.test_results["vendor_registry"] = registry_count >= 3
                
        except Exception as e:
            logger.error(f"❌ Outbox tables test failed: {e}")
            self.test_results["outbox_tables"] = False
    
    async def test_universal_device_handler(self):
        """Test universal device handler integration"""
        logger.info("🔍 Testing universal device handler...")
        
        try:
            universal_handler = await get_universal_handler()
            
            # Test device type detection for each sample
            for device_type, sample_data in self.device_data_samples.items():
                detection_result = await universal_handler.detect_device_type(sample_data)
                
                detected_type = detection_result.get("device_type")
                confidence = detection_result.get("confidence", 0.0)
                
                logger.info(f"✅ {device_type.value}: detected={detected_type}, confidence={confidence}")
                
                self.test_results[f"detection_{device_type.value}"] = {
                    "detected_type": detected_type,
                    "confidence": confidence,
                    "success": detected_type == device_type.value or confidence > 0.5
                }
            
            # Test supported device types
            supported_types = await universal_handler.get_supported_device_types()
            logger.info(f"✅ Supported device types: {len(supported_types)}")
            
            self.test_results["universal_handler"] = True
            
        except Exception as e:
            logger.error(f"❌ Universal device handler test failed: {e}")
            self.test_results["universal_handler"] = False
    
    async def test_outbox_flow_all_devices(self):
        """Test outbox flow for all medical device types"""
        logger.info("🔍 Testing outbox flow for all device types...")
        
        vendors = ["fitbit", "garmin", "apple_health"]
        
        for device_type, sample_data in self.device_data_samples.items():
            for vendor in vendors:
                try:
                    # Store device data in outbox
                    correlation_id = str(uuid.uuid4())
                    
                    outbox_id = await self.outbox_service.store_device_data_transactionally(
                        device_data=sample_data,
                        vendor_id=vendor,
                        correlation_id=correlation_id,
                        trace_id=f"test-{device_type.value}-{vendor}"
                    )
                    
                    # Verify it's stored
                    messages = await self.outbox_service.get_pending_messages_with_lock(vendor, 10)
                    found_message = any(msg.correlation_id == correlation_id for msg in messages)
                    
                    test_key = f"outbox_flow_{device_type.value}_{vendor}"
                    self.test_results[test_key] = {
                        "stored": outbox_id is not None,
                        "retrieved": found_message,
                        "outbox_id": outbox_id
                    }
                    
                    logger.info(f"✅ {device_type.value} + {vendor}: stored={outbox_id is not None}, retrieved={found_message}")
                    
                except Exception as e:
                    logger.error(f"❌ {device_type.value} + {vendor} failed: {e}")
                    self.test_results[f"outbox_flow_{device_type.value}_{vendor}"] = {"error": str(e)}
    
    async def test_concurrent_processing(self):
        """Test concurrent processing with SELECT FOR UPDATE SKIP LOCKED"""
        logger.info("🔍 Testing concurrent processing...")
        
        try:
            # Create multiple concurrent tasks
            async def get_messages_batch(vendor, batch_id):
                messages = await self.outbox_service.get_pending_messages_with_lock(vendor, 5)
                return len(messages), batch_id
            
            # Test concurrent access for each vendor
            for vendor in ["fitbit", "garmin", "apple_health"]:
                tasks = [get_messages_batch(vendor, i) for i in range(3)]
                results = await asyncio.gather(*tasks)
                
                total_messages = sum(result[0] for result in results)
                logger.info(f"✅ {vendor} concurrent access: {results} (total: {total_messages})")
                
                self.test_results[f"concurrent_{vendor}"] = {
                    "results": results,
                    "total_messages": total_messages
                }
            
        except Exception as e:
            logger.error(f"❌ Concurrent processing test failed: {e}")
            self.test_results["concurrent_processing"] = False
    
    def print_test_results(self):
        """Print comprehensive test results"""
        logger.info("\n" + "="*80)
        logger.info("📊 COMPREHENSIVE SUPABASE OUTBOX TEST RESULTS")
        logger.info("="*80)
        
        # Connection tests
        logger.info("\n🔗 CONNECTION TESTS:")
        logger.info(f"Supabase Connection: {'✅ PASS' if self.test_results.get('supabase_connection') else '❌ FAIL'}")
        logger.info(f"Vendor Registry: {'✅ PASS' if self.test_results.get('vendor_registry') else '❌ FAIL'}")
        logger.info(f"Universal Handler: {'✅ PASS' if self.test_results.get('universal_handler') else '❌ FAIL'}")
        
        # Table tests
        logger.info("\n📊 TABLE TESTS:")
        for vendor in ["fitbit", "garmin", "apple_health"]:
            outbox_ok = self.test_results.get(f"table_{vendor}_outbox", False)
            dl_ok = self.test_results.get(f"table_{vendor}_dead_letter", False)
            logger.info(f"{vendor}: outbox={'✅' if outbox_ok else '❌'} dead_letter={'✅' if dl_ok else '❌'}")
        
        # Device type tests
        logger.info("\n🏥 DEVICE TYPE TESTS:")
        for device_type in self.medical_device_types:
            detection = self.test_results.get(f"detection_{device_type.value}", {})
            success = detection.get("success", False)
            confidence = detection.get("confidence", 0.0)
            logger.info(f"{device_type.value}: {'✅' if success else '❌'} (confidence: {confidence:.2f})")
        
        # Outbox flow tests
        logger.info("\n🔄 OUTBOX FLOW TESTS:")
        for device_type in self.medical_device_types:
            for vendor in ["fitbit", "garmin", "apple_health"]:
                flow_result = self.test_results.get(f"outbox_flow_{device_type.value}_{vendor}", {})
                if "error" in flow_result:
                    logger.info(f"{device_type.value} + {vendor}: ❌ ERROR")
                else:
                    stored = flow_result.get("stored", False)
                    retrieved = flow_result.get("retrieved", False)
                    status = "✅" if stored and retrieved else "❌"
                    logger.info(f"{device_type.value} + {vendor}: {status}")
        
        # Summary
        total_tests = len(self.test_results)
        passed_tests = sum(1 for result in self.test_results.values() 
                          if (isinstance(result, bool) and result) or 
                             (isinstance(result, dict) and not result.get("error") and 
                              result.get("stored") and result.get("retrieved")))
        
        logger.info("\n" + "="*80)
        logger.info(f"📈 SUMMARY: {passed_tests}/{total_tests} tests passed")
        
        if passed_tests == total_tests:
            logger.info("🎉 ALL TESTS PASSED! Outbox pattern ready for all medical devices!")
        else:
            logger.warning(f"⚠️ {total_tests - passed_tests} tests failed. Review issues before production.")
        
        logger.info("="*80)


async def main():
    """Main test runner"""
    tester = SupabaseOutboxTester()
    await tester.run_comprehensive_test()


if __name__ == "__main__":
    asyncio.run(main())
