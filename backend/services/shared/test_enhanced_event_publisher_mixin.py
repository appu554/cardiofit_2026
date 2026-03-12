#!/usr/bin/env python3
"""
Test Enhanced Event Publisher Mixin

This script tests the enhanced event publisher mixin to verify that it correctly
integrates with the Global Outbox Service and provides proper fallback functionality.
"""

import asyncio
import logging
import sys
from pathlib import Path

# Add the shared directory to Python path
shared_dir = Path(__file__).parent
if str(shared_dir) not in sys.path:
    sys.path.insert(0, str(shared_dir))

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class TestService:
    """Test service that uses the EventPublisherMixin"""
    
    def __init__(self, service_name: str):
        # Import here to avoid path issues
        from event_publishing.event_publisher_mixin import EventPublisherMixin
        
        # Create a test class that inherits from the mixin
        class TestServiceWithMixin(EventPublisherMixin):
            def __init__(self, name):
                super().__init__()
                self.name = name
        
        self.service = TestServiceWithMixin(service_name)
        self.service_name = service_name
    
    async def initialize(self):
        """Initialize the test service"""
        logger.info(f"Initializing test service: {self.service_name}")
        
        # Initialize event publisher with Global Outbox preference
        self.service.initialize_event_publisher(
            service_name=self.service_name,
            enabled=True,
            use_global_outbox=True
        )
        
        logger.info("Test service initialized")
    
    async def test_health_check(self):
        """Test the health check functionality"""
        logger.info("Testing event publisher health check...")
        
        health = await self.service.get_event_publisher_health()
        logger.info(f"Health status: {health}")
        
        return health
    
    async def test_statistics(self):
        """Test the statistics functionality"""
        logger.info("Testing event publisher statistics...")
        
        try:
            stats = await self.service.get_event_publisher_statistics()
            logger.info(f"Statistics: {stats}")
            return stats
        except Exception as e:
            logger.error(f"Statistics test failed: {e}")
            return None
    
    async def test_business_event(self):
        """Test publishing a business event"""
        logger.info("Testing business event publishing...")
        
        try:
            event_id = await self.service.publish_business_event(
                event_type="test.patient.created",
                resource_type="Patient",
                resource_id="test-patient-123",
                resource_data={
                    "resourceType": "Patient",
                    "id": "test-patient-123",
                    "name": [{"family": "Test", "given": ["Patient"]}],
                    "gender": "unknown"
                },
                operation="created",
                correlation_id="test-correlation-456",
                metadata={
                    "test": True,
                    "source": "enhanced_mixin_test"
                }
            )
            
            if event_id:
                logger.info(f"Business event published successfully: {event_id}")
                return event_id
            else:
                logger.warning("Business event publishing returned no ID")
                return None
                
        except Exception as e:
            logger.error(f"Business event test failed: {e}")
            return None
    
    async def test_custom_event(self):
        """Test publishing a custom event"""
        logger.info("Testing custom event publishing...")
        
        try:
            event_id = await self.service.publish_custom_event(
                topic="test-events",
                event_type="test.custom.event",
                data={
                    "message": "This is a test custom event",
                    "test_data": {
                        "number": 42,
                        "boolean": True,
                        "array": [1, 2, 3]
                    }
                },
                key="test-key-789",
                correlation_id="test-correlation-789",
                subject="test-subject",
                priority=2,
                metadata={
                    "test": True,
                    "event_source": "enhanced_mixin_test"
                }
            )
            
            if event_id:
                logger.info(f"Custom event published successfully: {event_id}")
                return event_id
            else:
                logger.warning("Custom event publishing returned no ID")
                return None
                
        except Exception as e:
            logger.error(f"Custom event test failed: {e}")
            return None
    
    async def test_patient_event(self):
        """Test the convenience patient event method"""
        logger.info("Testing patient event convenience method...")
        
        try:
            event_id = await self.service.publish_patient_event(
                patient_id="test-patient-456",
                operation="updated",
                patient_data={
                    "resourceType": "Patient",
                    "id": "test-patient-456",
                    "name": [{"family": "Updated", "given": ["Patient"]}],
                    "gender": "female"
                },
                correlation_id="test-correlation-patient"
            )
            
            if event_id:
                logger.info(f"Patient event published successfully: {event_id}")
                return event_id
            else:
                logger.warning("Patient event publishing returned no ID")
                return None
                
        except Exception as e:
            logger.error(f"Patient event test failed: {e}")
            return None
    
    def configure_test_settings(self):
        """Configure test-specific settings"""
        logger.info("Configuring test settings...")
        
        # Configure retry policy
        self.service.configure_retry_policy(retry_attempts=2, retry_delay=0.5)
        
        # Configure health check interval
        self.service.configure_health_check_interval(interval_seconds=10)
        
        logger.info("Test settings configured")
    
    def cleanup(self):
        """Cleanup the test service"""
        logger.info("Cleaning up test service...")
        self.service.close_event_publisher()
        logger.info("Test service cleaned up")


async def main():
    """Main test function"""
    logger.info("=" * 60)
    logger.info("Enhanced Event Publisher Mixin Test")
    logger.info("=" * 60)
    
    # Create test service
    test_service = TestService("enhanced-mixin-test-service")
    
    try:
        # Initialize the service
        await test_service.initialize()
        
        # Configure test settings
        test_service.configure_test_settings()
        
        # Test health check
        logger.info("\n--- Test 1: Health Check ---")
        health = await test_service.test_health_check()
        
        # Test statistics
        logger.info("\n--- Test 2: Statistics ---")
        stats = await test_service.test_statistics()
        
        # Test business event
        logger.info("\n--- Test 3: Business Event ---")
        business_event_id = await test_service.test_business_event()
        
        # Test custom event
        logger.info("\n--- Test 4: Custom Event ---")
        custom_event_id = await test_service.test_custom_event()
        
        # Test patient event convenience method
        logger.info("\n--- Test 5: Patient Event ---")
        patient_event_id = await test_service.test_patient_event()
        
        # Summary
        logger.info("\n" + "=" * 60)
        logger.info("Test Summary")
        logger.info("=" * 60)
        logger.info(f"Health Check: {'PASS' if health else 'FAIL'}")
        logger.info(f"Statistics: {'PASS' if stats else 'FAIL'}")
        logger.info(f"Business Event: {'PASS' if business_event_id else 'FAIL'}")
        logger.info(f"Custom Event: {'PASS' if custom_event_id else 'FAIL'}")
        logger.info(f"Patient Event: {'PASS' if patient_event_id else 'FAIL'}")
        
        if health and health.get("global_outbox_available"):
            logger.info("SUCCESS: Global Outbox Service integration working!")
        elif health and health.get("direct_kafka_available"):
            logger.info("SUCCESS: Direct Kafka fallback working!")
        else:
            logger.warning("WARNING: No event publishing method available")
        
    except Exception as e:
        logger.error(f"Test failed with error: {e}", exc_info=True)
    
    finally:
        # Cleanup
        test_service.cleanup()
    
    logger.info("=" * 60)
    logger.info("Enhanced Event Publisher Mixin Test Complete")
    logger.info("=" * 60)


if __name__ == "__main__":
    asyncio.run(main())
