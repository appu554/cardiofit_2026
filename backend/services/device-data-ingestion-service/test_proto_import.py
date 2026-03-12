#!/usr/bin/env python3
"""
Test Protocol Buffer Import

This script tests different import paths to find the correct way to import
the Global Outbox Service protocol buffers.
"""

import os
import sys
from pathlib import Path

print("Current working directory:", os.getcwd())
print("Script location:", __file__)
print("Python path:", sys.path)

# Test different path configurations
test_paths = []

# Path 1: From device-data-ingestion-service to global-outbox-service
current_dir = Path(__file__).parent  # device-data-ingestion-service/
services_dir = current_dir.parent  # services/
global_outbox_path1 = services_dir / 'global-outbox-service'
test_paths.append(("Direct to global-outbox-service", str(global_outbox_path1)))

# Path 2: Via shared directory (current approach)
shared_dir = services_dir / 'shared'
global_outbox_path2 = shared_dir.parent / 'global-outbox-service'
test_paths.append(("Via shared directory", str(global_outbox_path2)))

# Path 3: Absolute path
global_outbox_path3 = Path(__file__).parent.parent / 'global-outbox-service'
test_paths.append(("Absolute path", str(global_outbox_path3)))

print("\nTesting different paths:")
for name, path in test_paths:
    print(f"\n{name}: {path}")
    print(f"  Exists: {os.path.exists(path)}")
    
    if os.path.exists(path):
        app_proto_path = os.path.join(path, 'app', 'proto')
        print(f"  app/proto exists: {os.path.exists(app_proto_path)}")
        
        if os.path.exists(app_proto_path):
            proto_files = os.listdir(app_proto_path)
            print(f"  Proto files: {proto_files}")
            
            # Test import
            if path not in sys.path:
                sys.path.insert(0, path)
            
            try:
                from app.proto import outbox_pb2, outbox_pb2_grpc
                print(f"  Import SUCCESS!")
                
                # Test basic functionality
                request = outbox_pb2.HealthCheckRequest()
                print(f"  HealthCheckRequest created: {type(request)}")
                break
                
            except ImportError as e:
                print(f"  Import FAILED: {e}")
            finally:
                if path in sys.path:
                    sys.path.remove(path)

print("\nDone testing paths.")
