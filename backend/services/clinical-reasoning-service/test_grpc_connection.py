#!/usr/bin/env python3
"""
Simple test to verify CAE gRPC server is running and responding
"""

import asyncio
import sys
import os

# Add the shared directory to Python path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', 'shared'))

try:
    from cae_grpc_client import CAEGrpcClient
    print("✅ CAE gRPC client imported successfully")
except ImportError as e:
    print(f"❌ Failed to import CAE gRPC client: {e}")
    sys.exit(1)

async def test_cae_connection():
    """Test basic connection to CAE gRPC server"""
    print("🧪 Testing CAE gRPC Server Connection...")
    print("=" * 50)
    
    try:
        # Initialize client
        client = CAEGrpcClient()
        print("✅ CAE gRPC client initialized")
        
        # Test health check
        print("🔍 Testing health check...")
        health_status = await client.health_check()
        print(f"✅ Health check: {health_status}")
        
        # Test simple medication interaction
        print("🔍 Testing medication interaction check...")
        result = await client.check_medication_interactions(
            patient_id="test_patient_001",
            medication_ids=["rxnorm:855332", "rxnorm:1114195"],  # Warfarin + Aspirin
            clinical_context={
                "encounter_type": "outpatient",
                "care_setting": "clinic"
            }
        )
        
        print(f"✅ Interaction check completed:")
        print(f"   - Assertions: {len(result.get('assertions', []))}")
        for assertion in result.get('assertions', []):
            print(f"   - {assertion.get('type', 'unknown')}: {assertion.get('severity', 'unknown')}")
        
        print("\n🎉 CAE gRPC Server is working correctly!")
        return True
        
    except Exception as e:
        print(f"❌ Error testing CAE server: {e}")
        return False
    finally:
        try:
            await client.close()
        except:
            pass

if __name__ == "__main__":
    success = asyncio.run(test_cae_connection())
    sys.exit(0 if success else 1)
