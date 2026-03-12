#!/usr/bin/env python3
"""
Comprehensive Test Suite for Transactional Outbox Pattern Implementation

Tests:
1. Database schema and migrations
2. VendorAwareOutboxService CRUD operations
3. SELECT FOR UPDATE SKIP LOCKED behavior
4. OutboxPublisher service functionality
5. Dead letter handling
6. Cloud-native metrics emission
7. End-to-end integration flow
"""
import asyncio
import json
import logging
import sys
import uuid
from datetime import datetime, timedelta
from pathlib import Path

# Add the app directory to Python path
sys.path.append(str(Path(__file__).parent / "app"))

import pytest
from app.db.database import startup_database, shutdown_database, get_async_session, run_migration_script
from app.services.outbox_service import VendorAwareOutboxService
from app.services.outbox_publisher import outbox_publisher
from app.services.dead_letter_manager import dead_letter_manager
from app.core.monitoring import metrics_collector
from app.db.models import is_supported_vendor, SUPPORTED_VENDORS

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


class OutboxTestSuite:
    """Comprehensive test suite for outbox implementation"""
    
    def __init__(self):
        self.outbox_service = VendorAwareOutboxService()
        self.test_data = {
            "device_id": "test-device-001",
            "timestamp": int(datetime.utcnow().timestamp()),
            "reading_type": "heart_rate",
            "value": 75.5,
            "unit": "bpm",
            "patient_id": "test-patient-123",
            "metadata": {"test": True}
        }
        self.test_results = {}
    
    async def run_all_tests(self):
        """Run all tests in sequence"""
        logger.info("🚀 Starting Outbox Implementation Test Suite")
        
        try:
            # Initialize database
            await startup_database()
            
            # Run tests
            await self.test_database_schema()
            await self.test_vendor_registry()
            await self.test_outbox_service_crud()
            await self.test_select_for_update_skip_locked()
            await self.test_dead_letter_handling()
            await self.test_metrics_collection()
            await self.test_publisher_service()
            await self.test_end_to_end_flow()
            
            # Print results
            self.print_test_results()
            
        except Exception as e:
            logger.error(f"💥 Test suite failed: {e}")
            raise
        finally:
            await shutdown_database()
    
    async def test_database_schema(self):
        """Test database schema and migration"""
        logger.info("🔍 Testing database schema...")
        
        try:
            # Test migration
            migration_file = Path(__file__).parent / "migrations" / "001_create_outbox_tables.sql"
            if migration_file.exists():
                success = await run_migration_script(str(migration_file))
                self.test_results["migration"] = success
                logger.info(f"✅ Migration test: {'PASSED' if success else 'FAILED'}")
            else:
                logger.warning("⚠️ Migration file not found, skipping migration test")
                self.test_results["migration"] = None
            
            # Test table existence
            async with get_async_session() as session:
                from sqlalchemy import text
                
                # Check outbox tables
                for vendor_id in SUPPORTED_VENDORS.keys():
                    table_name = f"{vendor_id}_outbox"
                    result = await session.execute(
                        text("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = :table_name"),
                        {"table_name": table_name}
                    )
                    table_exists = result.scalar() > 0
                    self.test_results[f"table_{table_name}"] = table_exists
                    logger.info(f"✅ Table {table_name}: {'EXISTS' if table_exists else 'MISSING'}")
                
                # Check vendor registry
                result = await session.execute(
                    text("SELECT COUNT(*) FROM vendor_outbox_registry WHERE is_active = true")
                )
                registry_count = result.scalar()
                self.test_results["vendor_registry"] = registry_count >= 3
                logger.info(f"✅ Vendor registry: {registry_count} active vendors")
            
        except Exception as e:
            logger.error(f"❌ Database schema test failed: {e}")
            self.test_results["database_schema"] = False
    
    async def test_vendor_registry(self):
        """Test vendor registry functionality"""
        logger.info("🔍 Testing vendor registry...")
        
        try:
            # Test vendor support check
            for vendor_id in ["fitbit", "garmin", "apple_health"]:
                is_supported = is_supported_vendor(vendor_id)
                self.test_results[f"vendor_support_{vendor_id}"] = is_supported
                logger.info(f"✅ Vendor {vendor_id} supported: {is_supported}")
            
            # Test unsupported vendor
            unsupported = is_supported_vendor("unknown_vendor")
            self.test_results["vendor_support_unknown"] = not unsupported
            logger.info(f"✅ Unknown vendor correctly rejected: {not unsupported}")
            
        except Exception as e:
            logger.error(f"❌ Vendor registry test failed: {e}")
            self.test_results["vendor_registry_test"] = False
    
    async def test_outbox_service_crud(self):
        """Test outbox service CRUD operations"""
        logger.info("🔍 Testing outbox service CRUD operations...")
        
        try:
            # Test storing device data
            correlation_id = str(uuid.uuid4())
            outbox_id = await self.outbox_service.store_device_data_transactionally(
                device_data=self.test_data,
                vendor_id="fitbit",
                correlation_id=correlation_id,
                trace_id="test-trace-001"
            )
            
            self.test_results["store_device_data"] = outbox_id is not None
            logger.info(f"✅ Store device data: {'PASSED' if outbox_id else 'FAILED'}")
            
            # Test getting pending messages
            messages = await self.outbox_service.get_pending_messages_with_lock("fitbit", 10)
            self.test_results["get_pending_messages"] = len(messages) > 0
            logger.info(f"✅ Get pending messages: {len(messages)} messages found")
            
            # Test health status
            health_status = await self.outbox_service.get_health_status()
            self.test_results["health_status"] = health_status.get("status") == "healthy"
            logger.info(f"✅ Health status: {health_status.get('status', 'unknown')}")
            
            # Test queue depths
            queue_depths = await self.outbox_service.get_queue_depths()
            self.test_results["queue_depths"] = all(depth >= 0 for depth in queue_depths.values())
            logger.info(f"✅ Queue depths: {queue_depths}")
            
        except Exception as e:
            logger.error(f"❌ Outbox service CRUD test failed: {e}")
            self.test_results["outbox_crud"] = False
    
    async def test_select_for_update_skip_locked(self):
        """Test SELECT FOR UPDATE SKIP LOCKED behavior"""
        logger.info("🔍 Testing SELECT FOR UPDATE SKIP LOCKED...")
        
        try:
            # Store multiple test messages
            correlation_ids = []
            for i in range(5):
                test_data = self.test_data.copy()
                test_data["device_id"] = f"test-device-{i:03d}"
                correlation_id = str(uuid.uuid4())
                correlation_ids.append(correlation_id)
                
                await self.outbox_service.store_device_data_transactionally(
                    device_data=test_data,
                    vendor_id="garmin",
                    correlation_id=correlation_id
                )
            
            # Test concurrent access simulation
            async def get_messages_batch(batch_id):
                messages = await self.outbox_service.get_pending_messages_with_lock("garmin", 3)
                return len(messages), batch_id
            
            # Simulate concurrent access
            tasks = [get_messages_batch(i) for i in range(3)]
            results = await asyncio.gather(*tasks)
            
            total_messages = sum(result[0] for result in results)
            self.test_results["skip_locked_behavior"] = total_messages <= 5  # Should not exceed stored messages
            
            logger.info(f"✅ SELECT FOR UPDATE SKIP LOCKED: {results}")
            logger.info(f"✅ Total messages retrieved: {total_messages} (max 5)")
            
        except Exception as e:
            logger.error(f"❌ SELECT FOR UPDATE SKIP LOCKED test failed: {e}")
            self.test_results["skip_locked"] = False
    
    async def test_dead_letter_handling(self):
        """Test dead letter handling functionality"""
        logger.info("🔍 Testing dead letter handling...")
        
        try:
            # Test dead letter statistics
            stats = await dead_letter_manager.get_dead_letter_statistics()
            self.test_results["dead_letter_stats"] = "total_dead_letters" in stats
            logger.info(f"✅ Dead letter statistics: {stats.get('total_dead_letters', 0)} total")
            
            # Test failure pattern analysis
            analysis = await dead_letter_manager.analyze_failure_patterns()
            self.test_results["failure_analysis"] = "recommendations" in analysis
            logger.info(f"✅ Failure analysis: {len(analysis.get('recommendations', []))} recommendations")
            
        except Exception as e:
            logger.error(f"❌ Dead letter handling test failed: {e}")
            self.test_results["dead_letter"] = False
    
    async def test_metrics_collection(self):
        """Test cloud-native metrics collection"""
        logger.info("🔍 Testing metrics collection...")
        
        try:
            # Test metrics collector health
            metrics_health = metrics_collector.get_health_status()
            self.test_results["metrics_health"] = metrics_health.get("metrics_enabled", False)
            logger.info(f"✅ Metrics enabled: {metrics_health.get('metrics_enabled', False)}")
            
            # Test metric emission (will log locally)
            await metrics_collector.emit_outbox_queue_depth("test_vendor", 10)
            await metrics_collector.emit_processing_latency("test_vendor", 25.5)
            await metrics_collector.emit_message_success("test_vendor")
            
            self.test_results["metrics_emission"] = True
            logger.info("✅ Metrics emission test completed")
            
        except Exception as e:
            logger.error(f"❌ Metrics collection test failed: {e}")
            self.test_results["metrics"] = False
    
    async def test_publisher_service(self):
        """Test publisher service functionality"""
        logger.info("🔍 Testing publisher service...")
        
        try:
            # Test publisher initialization
            success = await outbox_publisher.initialize()
            self.test_results["publisher_init"] = success
            logger.info(f"✅ Publisher initialization: {'PASSED' if success else 'FAILED'}")
            
            # Test publisher health status
            health = outbox_publisher.get_health_status()
            self.test_results["publisher_health"] = "is_running" in health
            logger.info(f"✅ Publisher health check: {health.get('is_running', False)}")
            
            # Test processing stats
            stats = outbox_publisher.get_processing_stats()
            self.test_results["publisher_stats"] = "messages_processed" in stats
            logger.info(f"✅ Publisher stats: {stats}")
            
        except Exception as e:
            logger.error(f"❌ Publisher service test failed: {e}")
            self.test_results["publisher"] = False
    
    async def test_end_to_end_flow(self):
        """Test complete end-to-end flow"""
        logger.info("🔍 Testing end-to-end flow...")
        
        try:
            # Store a message
            correlation_id = str(uuid.uuid4())
            outbox_id = await self.outbox_service.store_device_data_transactionally(
                device_data=self.test_data,
                vendor_id="apple_health",
                correlation_id=correlation_id
            )
            
            # Verify it's in the outbox
            messages = await self.outbox_service.get_pending_messages_with_lock("apple_health", 1)
            message_found = any(msg.correlation_id == correlation_id for msg in messages)
            
            self.test_results["end_to_end_store"] = message_found
            logger.info(f"✅ End-to-end store: {'PASSED' if message_found else 'FAILED'}")
            
            # Test the complete flow would work
            if messages:
                message = messages[0]
                # Simulate successful processing
                success = await self.outbox_service.mark_message_completed(message)
                self.test_results["end_to_end_complete"] = success
                logger.info(f"✅ End-to-end completion: {'PASSED' if success else 'FAILED'}")
            
        except Exception as e:
            logger.error(f"❌ End-to-end flow test failed: {e}")
            self.test_results["end_to_end"] = False
    
    def print_test_results(self):
        """Print comprehensive test results"""
        logger.info("\n" + "="*60)
        logger.info("📊 TRANSACTIONAL OUTBOX PATTERN TEST RESULTS")
        logger.info("="*60)
        
        passed = 0
        failed = 0
        skipped = 0
        
        for test_name, result in self.test_results.items():
            if result is True:
                status = "✅ PASSED"
                passed += 1
            elif result is False:
                status = "❌ FAILED"
                failed += 1
            else:
                status = "⚠️ SKIPPED"
                skipped += 1
            
            logger.info(f"{test_name:30} {status}")
        
        logger.info("-" * 60)
        logger.info(f"Total Tests: {len(self.test_results)}")
        logger.info(f"Passed: {passed}")
        logger.info(f"Failed: {failed}")
        logger.info(f"Skipped: {skipped}")
        
        if failed == 0:
            logger.info("🎉 ALL TESTS PASSED! Outbox implementation is ready for production.")
        else:
            logger.warning(f"⚠️ {failed} tests failed. Please review and fix issues before deployment.")
        
        logger.info("="*60)


async def main():
    """Main test runner"""
    test_suite = OutboxTestSuite()
    await test_suite.run_all_tests()


if __name__ == "__main__":
    asyncio.run(main())
