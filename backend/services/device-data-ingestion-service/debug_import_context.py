#!/usr/bin/env python3
"""
Debug Import Context

This script tests the import context when running from the device-data-ingestion-service
to understand why the protocol buffer import fails during service startup.
"""

import os
import sys
from pathlib import Path

print("=== Import Context Debug ===")
print(f"Current working directory: {os.getcwd()}")
print(f"Script location: {__file__}")
print(f"Python path: {sys.path}")

# Simulate the same import context as the running service
print("\n=== Simulating Service Import Context ===")

# Add the app directory to Python path (like the real service does)
app_dir = Path(__file__).parent / "app"
if str(app_dir) not in sys.path:
    sys.path.insert(0, str(app_dir))

print(f"Added app directory to path: {app_dir}")

# Now try to import the global_outbox_adapter (which imports outbox_client)
try:
    print("\nTesting global_outbox_adapter import...")
    from app.services.global_outbox_adapter import global_outbox_adapter, GLOBAL_OUTBOX_CLIENT_AVAILABLE
    
    print(f"SUCCESS: global_outbox_adapter imported")
    print(f"GLOBAL_OUTBOX_CLIENT_AVAILABLE: {GLOBAL_OUTBOX_CLIENT_AVAILABLE}")
    
    # Test the adapter
    print("\nTesting adapter functionality...")
    import asyncio
    
    async def test_adapter():
        health = await global_outbox_adapter.health_check()
        print(f"Health check result: {health}")
        
        stats = await global_outbox_adapter.get_outbox_statistics()
        print(f"Statistics: {stats}")
    
    asyncio.run(test_adapter())
    
except Exception as e:
    print(f"FAILED: {e}")
    import traceback
    traceback.print_exc()

print("\n=== Direct outbox_client Import Test ===")

# Test direct import of outbox_client
try:
    # Add shared directory to path
    current_dir = Path(__file__).parent  # device-data-ingestion-service/
    services_dir = current_dir.parent  # services/
    shared_path = services_dir / 'shared'
    
    if str(shared_path) not in sys.path:
        sys.path.insert(0, str(shared_path))
    
    print(f"Added shared path: {shared_path}")
    
    from outbox_client import GlobalOutboxClient, GRPC_AVAILABLE
    print(f"SUCCESS: outbox_client imported directly")
    print(f"GRPC_AVAILABLE: {GRPC_AVAILABLE}")
    
except Exception as e:
    print(f"FAILED: {e}")
    import traceback
    traceback.print_exc()

print("\n=== Protocol Buffer Direct Import Test ===")

# Test direct protocol buffer import
try:
    global_outbox_path = services_dir / 'global-outbox-service'
    if str(global_outbox_path) not in sys.path:
        sys.path.insert(0, str(global_outbox_path))
    
    print(f"Added global outbox path: {global_outbox_path}")
    
    from app.proto import outbox_pb2, outbox_pb2_grpc
    print("SUCCESS: Protocol buffers imported directly")
    
    # Test creating a request
    request = outbox_pb2.HealthCheckRequest()
    print(f"SUCCESS: HealthCheckRequest created: {type(request)}")
    
except Exception as e:
    print(f"FAILED: {e}")
    import traceback
    traceback.print_exc()

print("\n=== End Debug ===")
