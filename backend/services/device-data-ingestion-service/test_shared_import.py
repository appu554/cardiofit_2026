#!/usr/bin/env python3
"""
Test Shared Outbox Client Import

This script tests the import of the shared outbox client to see what's happening
with the protocol buffer import.
"""

import os
import sys
from pathlib import Path

print("Testing shared outbox client import...")
print("Current working directory:", os.getcwd())
print("Script location:", __file__)

# Add the app directory to Python path (like the real service does)
app_dir = Path(__file__).parent / "app"
if str(app_dir) not in sys.path:
    sys.path.insert(0, str(app_dir))

print("App directory added to path:", str(app_dir))

# Now try to import the shared outbox client
try:
    # Add shared directory to path
    # From device-data-ingestion-service to services/shared
    current_dir = os.path.dirname(__file__)  # device-data-ingestion-service/
    services_dir = os.path.dirname(current_dir)  # services/
    shared_path = os.path.join(services_dir, 'shared')
    print("Shared path:", shared_path)
    print("Shared path exists:", os.path.exists(shared_path))
    
    if shared_path not in sys.path:
        sys.path.insert(0, shared_path)
    
    print("About to import outbox_client...")
    from outbox_client import GlobalOutboxClient, publish_to_global_outbox, GRPC_AVAILABLE
    
    print("SUCCESS: outbox_client imported!")
    print("GRPC_AVAILABLE:", GRPC_AVAILABLE)
    
    if GRPC_AVAILABLE:
        print("Testing GlobalOutboxClient creation...")
        client = GlobalOutboxClient("test-service")
        print("SUCCESS: GlobalOutboxClient created")
    else:
        print("gRPC not available, but import succeeded")
        
except Exception as e:
    print("FAILED: outbox_client import failed:", e)
    import traceback
    traceback.print_exc()

print("Done testing shared import.")
