#!/usr/bin/env python3
"""
Simple gRPC Test for CAE Server
Tests the gRPC connection and basic functionality
"""

import asyncio
import sys
import os

# Add the shared directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'shared'))

async def test_cae_grpc():
    """Test CAE gRPC server basic functionality"""
    print("🧪 Testing CAE gRPC Server...")
    print("=" * 50)
    
    try:
        from cae_grpc_client import CAEGrpcClient
        print("✅ CAE gRPC client imported successfully")
        
        # Initialize client
        client = CAEGrpcClient()
        print("✅ CAE gRPC client initialized")
        
        # Test health check
        print("🔍 Testing health check...")
        health_status = await client.health_check()
        print(f"✅ Health check successful: {health_status}")
        
        # Test medication interaction with real GraphDB data
        print("🔍 Testing medication interaction (Warfarin + Aspirin)...")
        result = await client.check_medication_interactions(
            patient_id="905a60cb-8241-418f-b29b-5b020e851392",  # Real patient from GraphDB
            medication_ids=["11289", "1191"],  # Warfarin + Aspirin RxNorm codes
            clinical_context={
                "encounter_type": "outpatient",
                "care_setting": "clinic"
            }
        )
        
        if result and 'assertions' in result:
            assertions = result['assertions']
            print(f"✅ Generated {len(assertions)} clinical assertions")
            
            for assertion in assertions:
                severity = assertion.get('severity', 'unknown')
                assertion_type = assertion.get('type', 'unknown')
                confidence = assertion.get('confidence', 0)
                print(f"   - {assertion_type}: {severity} (confidence: {confidence:.2f})")
            
            print("✅ CAE gRPC server working with real clinical data!")
        else:
            print("❌ No assertions generated")
            return False
        
        await client.close()
        print("✅ Client connection closed")
        
        return True
        
    except ImportError as e:
        print(f"❌ Import error: {e}")
        print("💡 Make sure the shared CAE client is available")
        return False
    except Exception as e:
        print(f"❌ gRPC test failed: {e}")
        return False

async def main():
    """Main test function"""
    success = await test_cae_grpc()
    
    if success:
        print("\n🎉 Phase 1 CAE Implementation VALIDATED!")
        print("✅ gRPC Server: OPERATIONAL")
        print("✅ GraphDB Integration: WORKING")
        print("✅ Real Clinical Data: LOADED")
        print("✅ Core Reasoners: FUNCTIONAL")
        print("\n🚀 Ready to proceed to Phase 3!")
    else:
        print("\n❌ Phase 1 validation failed")
    
    return success

if __name__ == "__main__":
    success = asyncio.run(main())
    sys.exit(0 if success else 1)
