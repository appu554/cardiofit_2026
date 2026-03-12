#!/usr/bin/env python3
"""
Test script for Global Outbox Service gRPC endpoints

Tests the basic functionality of the gRPC service including:
- Health checks
- Event publishing
- Statistics retrieval
"""

import asyncio
import grpc
import logging
import sys
import uuid
import base64
import json
from datetime import datetime

# Add the app directory to the path
sys.path.insert(0, 'app')

from app.proto import outbox_pb2, outbox_pb2_grpc
from app.core.config import settings

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

class GlobalOutboxServiceTester:
    """Test client for Global Outbox Service"""
    
    def __init__(self, service_url: str = None):
        self.service_url = service_url or f"localhost:{settings.GRPC_PORT}"
        self.channel = None
        self.stub = None
    
    async def __aenter__(self):
        """Async context manager entry"""
        self.channel = grpc.aio.insecure_channel(self.service_url)
        self.stub = outbox_pb2_grpc.GlobalOutboxServiceStub(self.channel)
        return self
    
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit"""
        if self.channel:
            await self.channel.close()
    
    async def test_health_check(self):
        """Test the health check endpoint"""
        logger.info("Testing health check...")
        
        try:
            request = outbox_pb2.HealthCheckRequest()
            response = await self.stub.HealthCheck(request)
            
            logger.info(f"Health check response:")
            logger.info(f"  Status: {response.status}")
            logger.info(f"  Message: {response.message}")
            logger.info(f"  Components: {dict(response.components)}")
            
            return response.status == "HEALTHY"
            
        except Exception as e:
            logger.error(f"Health check failed: {e}")
            return False
    
    async def test_publish_event(self):
        """Test event publishing"""
        logger.info("Testing event publishing...")
        
        try:
            # Create test event payload
            test_payload = {
                "event_type": "test_event",
                "data": {
                    "message": "Hello from Global Outbox Service test",
                    "timestamp": datetime.utcnow().isoformat(),
                    "test_id": str(uuid.uuid4())
                }
            }
            
            # Encode payload as base64
            payload_json = json.dumps(test_payload)
            payload_bytes = payload_json.encode('utf-8')
            
            # Create publish request
            request = outbox_pb2.PublishEventRequest(
                idempotency_key=str(uuid.uuid4()),
                origin_service="test-service",
                kafka_topic="test-events",
                kafka_key="test-key",
                event_payload=payload_bytes,
                event_type="test_event",
                correlation_id=str(uuid.uuid4()),
                subject="test-subject",
                priority=1
            )
            
            # Add metadata
            request.metadata.update({
                "test_run": "grpc_service_test",
                "environment": "development"
            })
            
            response = await self.stub.PublishEvent(request)
            
            logger.info(f"Publish event response:")
            logger.info(f"  Outbox Record ID: {response.outbox_record_id}")
            logger.info(f"  Status: {response.status}")
            logger.info(f"  Accepted At: {response.accepted_at}")
            
            return response.status in ["QUEUED", "SCHEDULED"]
            
        except Exception as e:
            logger.error(f"Event publishing failed: {e}")
            return False
    
    async def test_get_outbox_stats(self):
        """Test outbox statistics retrieval"""
        logger.info("Testing outbox statistics...")
        
        try:
            request = outbox_pb2.OutboxStatsRequest()
            response = await self.stub.GetOutboxStats(request)
            
            logger.info(f"Outbox statistics:")
            logger.info(f"  Queue Depths: {dict(response.queue_depths)}")
            logger.info(f"  Total Events Processed: {response.total_events_processed}")
            logger.info(f"  Dead Letter Count: {response.dead_letter_count}")
            logger.info(f"  Timestamp: {response.timestamp}")
            
            return True
            
        except Exception as e:
            logger.error(f"Statistics retrieval failed: {e}")
            return False
    
    async def test_get_events_by_correlation(self):
        """Test getting events by correlation ID"""
        logger.info("Testing events by correlation ID...")
        
        try:
            # Use a test correlation ID
            test_correlation_id = str(uuid.uuid4())
            
            request = outbox_pb2.GetEventsByCorrelationRequest(
                correlation_id=test_correlation_id,
                limit=10
            )
            
            response = await self.stub.GetEventsByCorrelation(request)
            
            logger.info(f"Events by correlation response:")
            logger.info(f"  Total Count: {response.total_count}")
            logger.info(f"  Events Found: {len(response.events)}")
            
            for event in response.events:
                logger.info(f"    Event ID: {event.id}")
                logger.info(f"    Origin Service: {event.origin_service}")
                logger.info(f"    Status: {event.status}")
            
            return True
            
        except Exception as e:
            logger.error(f"Events by correlation failed: {e}")
            return False

async def run_tests():
    """Run all tests"""
    logger.info("=" * 60)
    logger.info("Global Outbox Service gRPC Test Suite")
    logger.info("=" * 60)
    
    test_results = {}
    
    try:
        async with GlobalOutboxServiceTester() as tester:
            logger.info(f"Connecting to Global Outbox Service at {tester.service_url}")
            
            # Test 1: Health Check
            test_results["health_check"] = await tester.test_health_check()
            
            # Test 2: Event Publishing
            test_results["publish_event"] = await tester.test_publish_event()
            
            # Test 3: Outbox Statistics
            test_results["outbox_stats"] = await tester.test_get_outbox_stats()
            
            # Test 4: Events by Correlation
            test_results["events_by_correlation"] = await tester.test_get_events_by_correlation()
            
    except Exception as e:
        logger.error(f"Test suite failed: {e}")
        test_results["connection"] = False
    
    # Print results
    logger.info("=" * 60)
    logger.info("Test Results Summary")
    logger.info("=" * 60)
    
    all_passed = True
    for test_name, result in test_results.items():
        status = "PASS" if result else "FAIL"
        logger.info(f"  {test_name}: {status}")
        if not result:
            all_passed = False
    
    logger.info("=" * 60)
    
    if all_passed:
        logger.info("All tests PASSED! Global Outbox Service is working correctly.")
        return 0
    else:
        logger.error("Some tests FAILED! Check the logs above for details.")
        return 1

async def test_service_startup():
    """Test if the service can start up properly"""
    logger.info("Testing service startup...")
    
    try:
        # Import the main application
        from app.main import app
        from app.core.database import db_manager
        
        # Test database connection
        logger.info("Testing database connection...")
        connected = await db_manager.connect()
        
        if connected:
            logger.info("Database connection successful")
            health = await db_manager.health_check()
            logger.info(f"Database health: {health}")
            await db_manager.disconnect()
            return True
        else:
            logger.error("Database connection failed")
            return False
            
    except Exception as e:
        logger.error(f"Service startup test failed: {e}")
        return False

if __name__ == "__main__":
    import argparse
    
    parser = argparse.ArgumentParser(description="Test Global Outbox Service")
    parser.add_argument("--startup-only", action="store_true", 
                       help="Only test service startup, not gRPC endpoints")
    parser.add_argument("--service-url", default=None,
                       help="gRPC service URL (default: localhost:50051)")
    
    args = parser.parse_args()
    
    if args.startup_only:
        # Test service startup only
        result = asyncio.run(test_service_startup())
        sys.exit(0 if result else 1)
    else:
        # Run full gRPC test suite
        result = asyncio.run(run_tests())
        sys.exit(result)
