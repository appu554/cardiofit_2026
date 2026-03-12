#!/usr/bin/env python3
"""
Quick test script for CAE gRPC service
"""

import asyncio
import sys
from pathlib import Path

# Add the shared directory to the path
sys.path.insert(0, str(Path(__file__).parent.parent / 'shared'))

from cae_grpc_client import CAEgRPCClient

async def quick_test():
    """Quick test of the CAE gRPC service"""
    print("🧪 Quick CAE gRPC Test")
    print("=" * 40)
    
    try:
        async with CAEgRPCClient(service_name="quick-test") as client:
            print("✅ Connected to CAE service")
            
            # Test health check
            is_healthy = await client.health_check()
            print(f"Health check: {'✅ Healthy' if is_healthy else '❌ Unhealthy'}")
            
            # Test medication interactions
            print("\n💊 Testing medication interactions...")
            result = await client.check_medication_interactions(
                patient_id="test-patient-123",
                medication_ids=["warfarin", "aspirin"],
                new_medication_id="ibuprofen"
            )
            
            print(f"Found {len(result['interactions'])} interactions")
            for interaction in result['interactions']:
                print(f"  - {interaction['medication_a']} + {interaction['medication_b']}: {interaction['severity']}")
            
            print(f"Processing time: {result['metadata']['processing_time_ms']}ms")
            
            print("\n🎉 Quick test completed successfully!")
            
    except Exception as e:
        print(f"❌ Test failed: {e}")

if __name__ == "__main__":
    asyncio.run(quick_test())
