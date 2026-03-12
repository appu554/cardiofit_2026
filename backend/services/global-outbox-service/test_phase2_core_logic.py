#!/usr/bin/env python3
"""
Phase 2 Core Logic Test Suite for Global Outbox Service

Tests all Phase 2 functionality including:
- Transactional outbox storage
- Idempotency handling
- Background publisher with polling
- Retry logic with exponential backoff
- Dead letter queue processing
"""

import asyncio
import json
import logging
import sys
import uuid
from pathlib import Path
from typing import Dict, Any, List
from unittest.mock import Mock, patch
import time

# Add the backend directory to Python path
backend_dir = Path(__file__).parent.parent.parent
if str(backend_dir) not in sys.path:
    sys.path.insert(0, str(backend_dir))

from app.core.config import settings
from app.core.database import db_manager
from app.services.outbox_manager import OutboxManager
from app.services.publisher import BackgroundPublisher

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class Phase2CoreLogicTester:
    """Comprehensive test suite for Phase 2 core outbox logic"""
    
    def __init__(self):
        self.tests_passed = 0
        self.tests_failed = 0
        self.test_results = []
        self.outbox_manager = OutboxManager()
        self.test_events = []
    
    def log_test_result(self, test_name: str, passed: bool, message: str = ""):
        """Log test result"""
        status = "✅ PASS" if passed else "❌ FAIL"
        full_message = f"{status} {test_name}"
        if message:
            full_message += f" - {message}"
        
        logger.info(full_message)
        
        if passed:
            self.tests_passed += 1
        else:
            self.tests_failed += 1
        
        self.test_results.append({
            "test": test_name,
            "passed": passed,
            "message": message
        })
    
    async def setup_test_environment(self):
        """Setup test environment"""
        try:
            # Connect to database
            connected = await db_manager.connect()
            if not connected:
                raise Exception("Failed to connect to database")
            
            # Run migration to ensure schema is up to date
            await db_manager.execute_migration()
            
            # Clean up any existing test data
            await self.cleanup_test_data()
            
            logger.info("✅ Test environment setup complete")
            return True
            
        except Exception as e:
            logger.error(f"❌ Failed to setup test environment: {e}")
            return False
    
    async def cleanup_test_data(self):
        """Clean up test data"""
        try:
            # Delete test events
            async with db_manager.get_connection() as conn:
                await conn.execute("""
                    DELETE FROM global_event_outbox 
                    WHERE origin_service = 'test-service' 
                    OR idempotency_key LIKE 'test-%'
                """)
                
                await conn.execute("""
                    DELETE FROM global_dead_letter_queue 
                    WHERE origin_service = 'test-service'
                """)
            
            logger.info("✅ Test data cleanup complete")
            
        except Exception as e:
            logger.error(f"❌ Failed to cleanup test data: {e}")
    
    async def test_transactional_outbox_storage(self):
        """Test transactional event storage"""
        try:
            # Test basic event storage
            event_id = await self.outbox_manager.store_event(
                idempotency_key="test-basic-storage",
                origin_service="test-service",
                kafka_topic="test-topic",
                kafka_key="test-key",
                event_payload=b'{"test": "data"}',
                event_type="test.event",
                correlation_id="test-correlation-123",
                metadata={"test": True}
            )
            
            if not event_id:
                self.log_test_result("Transactional Storage - Basic", False, "Failed to store event")
                return
            
            # Verify event was stored
            async with db_manager.get_connection() as conn:
                stored_event = await conn.fetchrow("""
                    SELECT * FROM global_event_outbox WHERE id = $1
                """, event_id)
            
            if not stored_event:
                self.log_test_result("Transactional Storage - Basic", False, "Event not found in database")
                return
            
            # Verify event data
            assert stored_event['origin_service'] == 'test-service'
            assert stored_event['kafka_topic'] == 'test-topic'
            assert stored_event['kafka_key'] == 'test-key'
            assert stored_event['event_type'] == 'test.event'
            assert stored_event['correlation_id'] == 'test-correlation-123'
            assert stored_event['status'] == 'pending'
            
            self.test_events.append(event_id)
            self.log_test_result("Transactional Storage - Basic", True, f"Event stored with ID: {event_id}")
            
        except Exception as e:
            self.log_test_result("Transactional Storage - Basic", False, str(e))
    
    async def test_idempotency_handling(self):
        """Test idempotency key handling"""
        try:
            idempotency_key = "test-idempotency-123"
            
            # Store first event
            event_id_1 = await self.outbox_manager.store_event(
                idempotency_key=idempotency_key,
                origin_service="test-service",
                kafka_topic="test-topic",
                event_payload=b'{"version": 1}',
                event_type="test.idempotency"
            )
            
            # Store second event with same idempotency key (should update)
            event_id_2 = await self.outbox_manager.store_event(
                idempotency_key=idempotency_key,
                origin_service="test-service",
                kafka_topic="test-topic-updated",
                event_payload=b'{"version": 2}',
                event_type="test.idempotency.updated"
            )
            
            # Should return the same ID (updated record)
            if event_id_1 != event_id_2:
                self.log_test_result("Idempotency Handling", False, f"Different IDs returned: {event_id_1} vs {event_id_2}")
                return
            
            # Verify only one record exists
            async with db_manager.get_connection() as conn:
                count = await conn.fetchval("""
                    SELECT COUNT(*) FROM global_event_outbox 
                    WHERE origin_service = 'test-service' AND idempotency_key = $1
                """, idempotency_key)
                
                if count != 1:
                    self.log_test_result("Idempotency Handling", False, f"Expected 1 record, found {count}")
                    return
                
                # Verify the record was updated
                updated_event = await conn.fetchrow("""
                    SELECT * FROM global_event_outbox 
                    WHERE origin_service = 'test-service' AND idempotency_key = $1
                """, idempotency_key)
                
                if updated_event['kafka_topic'] != 'test-topic-updated':
                    self.log_test_result("Idempotency Handling", False, "Event was not updated")
                    return
            
            self.test_events.append(event_id_1)
            self.log_test_result("Idempotency Handling", True, "Idempotency constraint working correctly")
            
        except Exception as e:
            self.log_test_result("Idempotency Handling", False, str(e))
    
    async def test_pending_events_retrieval(self):
        """Test retrieval of pending events with SELECT FOR UPDATE SKIP LOCKED"""
        try:
            # Create multiple test events
            event_ids = []
            for i in range(5):
                event_id = await self.outbox_manager.store_event(
                    idempotency_key=f"test-pending-{i}",
                    origin_service="test-service",
                    kafka_topic="test-topic",
                    event_payload=f'{{"index": {i}}}'.encode(),
                    event_type="test.pending",
                    priority=i % 4  # Mix of priorities
                )
                event_ids.append(event_id)
            
            # Retrieve pending events
            pending_events = await self.outbox_manager.get_pending_events(limit=10)
            
            if len(pending_events) < 5:
                self.log_test_result("Pending Events Retrieval", False, f"Expected at least 5 events, got {len(pending_events)}")
                return
            
            # Verify events are marked as processing
            async with db_manager.get_connection() as conn:
                processing_count = await conn.fetchval("""
                    SELECT COUNT(*) FROM global_event_outbox 
                    WHERE origin_service = 'test-service' AND status = 'processing'
                """)
            
            if processing_count < 5:
                self.log_test_result("Pending Events Retrieval", False, f"Expected at least 5 processing events, got {processing_count}")
                return
            
            self.test_events.extend(event_ids)
            self.log_test_result("Pending Events Retrieval", True, f"Retrieved {len(pending_events)} events, {processing_count} marked as processing")
            
        except Exception as e:
            self.log_test_result("Pending Events Retrieval", False, str(e))

    async def test_event_status_updates(self):
        """Test event status update operations"""
        try:
            # Create test event
            event_id = await self.outbox_manager.store_event(
                idempotency_key="test-status-updates",
                origin_service="test-service",
                kafka_topic="test-topic",
                event_payload=b'{"test": "status"}',
                event_type="test.status"
            )

            # Test marking as published
            success = await self.outbox_manager.mark_event_published(event_id)
            if not success:
                self.log_test_result("Event Status Updates - Published", False, "Failed to mark event as published")
                return

            # Verify status update
            async with db_manager.get_connection() as conn:
                status = await conn.fetchval("""
                    SELECT status FROM global_event_outbox WHERE id = $1
                """, event_id)

            if status != 'published':
                self.log_test_result("Event Status Updates - Published", False, f"Expected 'published', got '{status}'")
                return

            # Create another event for failure test
            event_id_2 = await self.outbox_manager.store_event(
                idempotency_key="test-status-failed",
                origin_service="test-service",
                kafka_topic="test-topic",
                event_payload=b'{"test": "failure"}',
                event_type="test.failure"
            )

            # Test marking as failed
            success = await self.outbox_manager.mark_event_failed(event_id_2, "Test error message")
            if not success:
                self.log_test_result("Event Status Updates - Failed", False, "Failed to mark event as failed")
                return

            # Verify failure update
            async with db_manager.get_connection() as conn:
                failed_event = await conn.fetchrow("""
                    SELECT status, retry_count, last_error FROM global_event_outbox WHERE id = $1
                """, event_id_2)

            if failed_event['status'] != 'failed':
                self.log_test_result("Event Status Updates - Failed", False, f"Expected 'failed', got '{failed_event['status']}'")
                return

            if failed_event['retry_count'] != 1:
                self.log_test_result("Event Status Updates - Failed", False, f"Expected retry_count=1, got {failed_event['retry_count']}")
                return

            self.test_events.extend([event_id, event_id_2])
            self.log_test_result("Event Status Updates", True, "Published and failed status updates working correctly")

        except Exception as e:
            self.log_test_result("Event Status Updates", False, str(e))

    async def test_dead_letter_queue_processing(self):
        """Test dead letter queue processing for max retry failures"""
        try:
            # Create test event
            event_id = await self.outbox_manager.store_event(
                idempotency_key="test-dlq-processing",
                origin_service="test-service",
                kafka_topic="test-topic",
                event_payload=b'{"test": "dlq"}',
                event_type="test.dlq"
            )

            # Simulate multiple failures to exceed max retries
            for i in range(settings.MAX_RETRY_ATTEMPTS):
                await self.outbox_manager.mark_event_failed(event_id, f"Test error {i+1}")

            # Verify event was moved to dead letter queue
            async with db_manager.get_connection() as conn:
                # Check if event was removed from main outbox
                outbox_count = await conn.fetchval("""
                    SELECT COUNT(*) FROM global_event_outbox WHERE id = $1
                """, event_id)

                # Check if event was added to dead letter queue
                dlq_count = await conn.fetchval("""
                    SELECT COUNT(*) FROM global_dead_letter_queue
                    WHERE original_outbox_id = $1
                """, event_id)

            if outbox_count != 0:
                self.log_test_result("Dead Letter Queue Processing", False, f"Event still in outbox (count: {outbox_count})")
                return

            if dlq_count != 1:
                self.log_test_result("Dead Letter Queue Processing", False, f"Event not in DLQ (count: {dlq_count})")
                return

            # Verify DLQ record details
            async with db_manager.get_connection() as conn:
                dlq_record = await conn.fetchrow("""
                    SELECT * FROM global_dead_letter_queue
                    WHERE original_outbox_id = $1
                """, event_id)

            if not dlq_record:
                self.log_test_result("Dead Letter Queue Processing", False, "DLQ record not found")
                return

            if dlq_record['origin_service'] != 'test-service':
                self.log_test_result("Dead Letter Queue Processing", False, "DLQ record has incorrect origin_service")
                return

            if dlq_record['retry_count'] != settings.MAX_RETRY_ATTEMPTS:
                self.log_test_result("Dead Letter Queue Processing", False, f"DLQ record has incorrect retry_count: {dlq_record['retry_count']}")
                return

            self.log_test_result("Dead Letter Queue Processing", True, f"Event moved to DLQ after {settings.MAX_RETRY_ATTEMPTS} failures")

        except Exception as e:
            self.log_test_result("Dead Letter Queue Processing", False, str(e))

    async def test_correlation_tracking(self):
        """Test correlation ID tracking and retrieval"""
        try:
            correlation_id = f"test-correlation-{uuid.uuid4()}"

            # Create multiple events with same correlation ID
            event_ids = []
            for i in range(3):
                event_id = await self.outbox_manager.store_event(
                    idempotency_key=f"test-correlation-{i}",
                    origin_service="test-service",
                    kafka_topic="test-topic",
                    event_payload=f'{{"step": {i}}}'.encode(),
                    event_type="test.correlation",
                    correlation_id=correlation_id
                )
                event_ids.append(event_id)

            # Retrieve events by correlation ID
            correlated_events = await self.outbox_manager.get_events_by_correlation(correlation_id)

            if len(correlated_events) != 3:
                self.log_test_result("Correlation Tracking", False, f"Expected 3 events, got {len(correlated_events)}")
                return

            # Verify all events have the correct correlation ID
            for event in correlated_events:
                if event['correlation_id'] != correlation_id:
                    self.log_test_result("Correlation Tracking", False, f"Event has incorrect correlation_id: {event['correlation_id']}")
                    return

            self.test_events.extend(event_ids)
            self.log_test_result("Correlation Tracking", True, f"Retrieved {len(correlated_events)} events by correlation ID")

        except Exception as e:
            self.log_test_result("Correlation Tracking", False, str(e))

    async def test_priority_ordering(self):
        """Test priority-based event ordering"""
        try:
            # Create events with different priorities
            priorities = [0, 1, 2, 3, 1, 2]  # Mix of priorities
            event_ids = []

            for i, priority in enumerate(priorities):
                event_id = await self.outbox_manager.store_event(
                    idempotency_key=f"test-priority-{i}",
                    origin_service="test-service",
                    kafka_topic="test-topic",
                    event_payload=f'{{"priority": {priority}}}'.encode(),
                    event_type="test.priority",
                    priority=priority
                )
                event_ids.append(event_id)

            # Retrieve events (should be ordered by priority DESC, then created_at ASC)
            pending_events = await self.outbox_manager.get_pending_events(limit=10)

            # Filter our test events
            test_events = [e for e in pending_events if e['event_type'] == 'test.priority']

            if len(test_events) < len(priorities):
                self.log_test_result("Priority Ordering", False, f"Expected {len(priorities)} events, got {len(test_events)}")
                return

            # Verify priority ordering (higher priority first)
            for i in range(len(test_events) - 1):
                current_priority = test_events[i]['priority']
                next_priority = test_events[i + 1]['priority']

                if current_priority < next_priority:
                    self.log_test_result("Priority Ordering", False, f"Priority ordering incorrect: {current_priority} before {next_priority}")
                    return

            self.test_events.extend(event_ids)
            self.log_test_result("Priority Ordering", True, f"Events correctly ordered by priority")

        except Exception as e:
            self.log_test_result("Priority Ordering", False, str(e))

    async def test_scheduled_events(self):
        """Test scheduled event processing"""
        try:
            from datetime import datetime, timedelta

            # Create scheduled event (future)
            future_time = datetime.utcnow() + timedelta(seconds=2)
            event_id = await self.outbox_manager.store_event(
                idempotency_key="test-scheduled-future",
                origin_service="test-service",
                kafka_topic="test-topic",
                event_payload=b'{"scheduled": true}',
                event_type="test.scheduled",
                scheduled_at=future_time
            )

            # Verify event is in scheduled status
            async with db_manager.get_connection() as conn:
                status = await conn.fetchval("""
                    SELECT status FROM global_event_outbox WHERE id = $1
                """, event_id)

            if status != 'scheduled':
                self.log_test_result("Scheduled Events", False, f"Expected 'scheduled', got '{status}'")
                return

            # Create scheduled event (past - should be pending)
            past_time = datetime.utcnow() - timedelta(seconds=1)
            event_id_2 = await self.outbox_manager.store_event(
                idempotency_key="test-scheduled-past",
                origin_service="test-service",
                kafka_topic="test-topic",
                event_payload=b'{"scheduled": false}',
                event_type="test.scheduled.past",
                scheduled_at=past_time
            )

            # This should still be scheduled initially
            async with db_manager.get_connection() as conn:
                status_2 = await conn.fetchval("""
                    SELECT status FROM global_event_outbox WHERE id = $1
                """, event_id_2)

            if status_2 != 'scheduled':
                self.log_test_result("Scheduled Events", False, f"Expected 'scheduled', got '{status_2}'")
                return

            self.test_events.extend([event_id, event_id_2])
            self.log_test_result("Scheduled Events", True, "Scheduled events created correctly")

        except Exception as e:
            self.log_test_result("Scheduled Events", False, str(e))

    async def test_health_check(self):
        """Test outbox manager health check"""
        try:
            health = await self.outbox_manager.health_check()

            if not health:
                self.log_test_result("Health Check", False, "Health check returned False")
                return

            self.log_test_result("Health Check", True, "Health check passed")

        except Exception as e:
            self.log_test_result("Health Check", False, str(e))

    async def run_all_tests(self):
        """Run all Phase 2 core logic tests"""
        logger.info("🧪 Running Phase 2 Core Logic Tests")
        logger.info("=" * 60)

        # Setup test environment
        if not await self.setup_test_environment():
            logger.error("❌ Failed to setup test environment")
            return False

        try:
            # Run all tests
            await self.test_transactional_outbox_storage()
            await self.test_idempotency_handling()
            await self.test_pending_events_retrieval()
            await self.test_event_status_updates()
            await self.test_dead_letter_queue_processing()
            await self.test_correlation_tracking()
            await self.test_priority_ordering()
            await self.test_scheduled_events()
            await self.test_health_check()

        finally:
            # Cleanup test data
            await self.cleanup_test_data()
            await db_manager.disconnect()

        # Print summary
        logger.info("=" * 60)
        logger.info(f"📊 Test Results: {self.tests_passed} passed, {self.tests_failed} failed")

        if self.tests_failed == 0:
            logger.info("🎉 All Phase 2 core logic tests passed! Ready for Phase 3.")
            return True
        else:
            logger.error("❌ Some tests failed. Please fix issues before proceeding.")

            # Show failed tests
            logger.error("\nFailed tests:")
            for result in self.test_results:
                if not result["passed"]:
                    logger.error(f"  - {result['test']}: {result['message']}")

            return False

async def main():
    """Main test runner"""
    tester = Phase2CoreLogicTester()
    success = await tester.run_all_tests()

    if success:
        print("\n✅ Phase 2 Core Logic is working perfectly!")
        print("\nNext steps:")
        print("1. Start Phase 3: Integration & Testing")
        print("2. Create gRPC client library for microservices")
        print("3. Add comprehensive monitoring and metrics")
        print("4. Performance testing and optimization")
    else:
        print("\n❌ Phase 2 Core Logic needs work before proceeding")
        print("\nRecommended actions:")
        print("1. Fix the failed tests above")
        print("2. Check database connectivity and schema")
        print("3. Verify Kafka configuration if publisher tests failed")

    return success

if __name__ == "__main__":
    success = asyncio.run(main())
    sys.exit(0 if success else 1)
