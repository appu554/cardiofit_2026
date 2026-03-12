#!/usr/bin/env python3
"""
Debug test to check what's working
"""

print("🔧 Debug Test Starting...")

try:
    import sys
    import os
    print(f"✅ Python version: {sys.version}")
    print(f"✅ Current directory: {os.getcwd()}")
    
    # Check shared directory
    shared_path = os.path.join(os.path.dirname(__file__), '..', 'shared')
    print(f"🔍 Shared path: {shared_path}")
    print(f"🔍 Shared path exists: {os.path.exists(shared_path)}")
    
    if os.path.exists(shared_path):
        files = os.listdir(shared_path)
        print(f"🔍 Files in shared: {files[:5]}...")  # Show first 5 files
    
    # Try to import grpc
    try:
        import grpc
        print("✅ grpc module available")
        
        # Test gRPC connection
        channel = grpc.insecure_channel('localhost:8027')
        state = channel.get_state(try_to_connect=True)
        print(f"✅ gRPC server state: {state}")
        channel.close()
        
    except ImportError as e:
        print(f"❌ grpc import failed: {e}")
    except Exception as e:
        print(f"❌ gRPC connection failed: {e}")
    
    # Try to import CAE client
    sys.path.insert(0, shared_path)
    try:
        from cae_grpc_client import CAEGrpcClient
        print("✅ CAE gRPC client imported successfully")
        
        # Try to create client
        client = CAEGrpcClient()
        print("✅ CAE gRPC client created")
        
    except ImportError as e:
        print(f"❌ CAE client import failed: {e}")
    except Exception as e:
        print(f"❌ CAE client creation failed: {e}")
    
    print("🎯 Debug test completed")
    
except Exception as e:
    print(f"❌ Debug test failed: {e}")
    import traceback
    traceback.print_exc()
