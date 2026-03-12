#!/usr/bin/env python3
"""
Test Global Outbox Service Connection

This script tests the connection to the Global Outbox Service to verify
that the gRPC endpoint is accessible and working correctly.
"""

import asyncio
import logging
import sys
from pathlib import Path

# Add the app directory to Python path
app_dir = Path(__file__).parent / "app"
if str(app_dir) not in sys.path:
    sys.path.insert(0, str(app_dir))

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


async def test_global_outbox_connection():
    """Test connection to Global Outbox Service"""
    
    logger.info("Testing Global Outbox Service connection...")
    
    try:
        from app.services.global_outbox_adapter import GlobalOutboxAdapter
        
        # Create adapter instance
        adapter = GlobalOutboxAdapter(
            global_outbox_url="localhost:50051",
            fallback_enabled=True
        )
        
        logger.info("Created Global Outbox Adapter")
        
        # Test health check
        logger.info("Testing health check...")
        health_status = await adapter.health_check()
        
        logger.info(f"Health status: {health_status}")
        
        if health_status.get("global_outbox_available", False):
            logger.info("SUCCESS: Global Outbox Service is available!")
            
            # Test statistics
            logger.info("Testing statistics retrieval...")
            stats = await adapter.get_outbox_statistics()
            logger.info(f"Statistics: {stats}")
            
            # Test a simple event storage
            logger.info("Testing event storage...")
            test_device_data = {
                "device_id": "test-device-123",
                "patient_id": "test-patient-456",
                "reading_type": "heart_rate",
                "value": 75,
                "unit": "bpm",
                "timestamp": "2025-07-02T10:00:00Z",
                "metadata": {
                    "test": True,
                    "migration_test": True
                }
            }
            
            record_id = await adapter.store_device_data_transactionally(
                device_data=test_device_data,
                vendor_id="test_vendor",
                correlation_id="test-correlation-123",
                trace_id="test-trace-456"
            )
            
            if record_id:
                logger.info(f"SUCCESS: Test event stored with ID: {record_id}")
            else:
                logger.error("FAILED: Test event storage returned no ID")
                
        elif health_status.get("fallback_available", False):
            logger.info("Global Outbox Service not available, but fallback is working")
            
        else:
            logger.error("FAILED: Neither Global Outbox Service nor fallback is available")
            
    except Exception as e:
        logger.error(f"ERROR: {e}", exc_info=True)


async def test_direct_grpc_connection():
    """Test direct gRPC connection to Global Outbox Service"""
    
    logger.info("Testing direct gRPC connection...")
    
    try:
        import grpc
        
        # Test basic gRPC connection
        channel = grpc.aio.insecure_channel("localhost:50051")
        
        # Try to connect
        try:
            await channel.channel_ready()
            logger.info("SUCCESS: gRPC channel is ready")
            
            # Try to import protocol buffers
            try:
                import sys
                import os
                
                # Add global outbox service to path
                global_outbox_path = os.path.join(
                    os.path.dirname(os.path.dirname(__file__)), 
                    'global-outbox-service'
                )
                if global_outbox_path not in sys.path:
                    sys.path.append(global_outbox_path)
                
                from app.proto import outbox_pb2, outbox_pb2_grpc
                logger.info("SUCCESS: Protocol buffers imported successfully")
                
                # Create stub and test health check
                stub = outbox_pb2_grpc.GlobalOutboxServiceStub(channel)
                
                request = outbox_pb2.HealthCheckRequest()
                response = await stub.HealthCheck(request)
                
                logger.info(f"SUCCESS: Health check response: {response.status}")
                
            except ImportError as e:
                logger.error(f"FAILED: Protocol buffer import failed: {e}")
                
        except Exception as e:
            logger.error(f"FAILED: gRPC channel connection failed: {e}")
            
        finally:
            await channel.close()
            
    except ImportError:
        logger.error("FAILED: gRPC not available")
    except Exception as e:
        logger.error(f"ERROR: {e}", exc_info=True)


async def main():
    """Main test function"""
    
    logger.info("=" * 60)
    logger.info("Global Outbox Service Connection Test")
    logger.info("=" * 60)
    
    # Test 1: Direct gRPC connection
    logger.info("\n--- Test 1: Direct gRPC Connection ---")
    await test_direct_grpc_connection()
    
    # Test 2: Global Outbox Adapter
    logger.info("\n--- Test 2: Global Outbox Adapter ---")
    await test_global_outbox_connection()
    
    logger.info("\n" + "=" * 60)
    logger.info("Connection test completed")
    logger.info("=" * 60)


if __name__ == "__main__":
    asyncio.run(main())
