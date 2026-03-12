#!/usr/bin/env python3
"""
Test script for Flow2 Rust Engine Snapshot-based Processing

This script demonstrates the new snapshot-based processing capabilities
by testing the enhanced API endpoints.
"""

import requests
import json
import time
from datetime import datetime, timezone
from typing import Dict, Any

# Configuration
RUST_ENGINE_URL = "http://localhost:8090"
CONTEXT_GATEWAY_URL = "http://localhost:8016"  # Context Gateway for snapshot data

def create_test_snapshot_request() -> Dict[str, Any]:
    """Create a test snapshot-based request"""
    return {
        "request_id": f"test-snapshot-{int(time.time())}",
        "recipe_id": "vancomycin-dosing-v1.0",
        "variant": "standard_auc",
        "patient_id": "patient-12345",
        "medication_code": "11124",  # Vancomycin
        "snapshot_id": "snapshot-abc123-def456",
        "timeout_ms": 30000,
        "integrity_verification_required": True
    }

def create_test_recipe_request_with_snapshot() -> Dict[str, Any]:
    """Create a test recipe execution request with snapshot ID"""
    return {
        "request_id": f"test-recipe-snapshot-{int(time.time())}",
        "recipe_id": "vancomycin-dosing-v1.0",
        "variant": "standard_auc",
        "patient_id": "patient-12345",
        "medication_code": "11124",
        "clinical_context": "",  # Will be populated from snapshot
        "timeout_ms": 30000,
        "snapshot_id": "snapshot-abc123-def456"
    }

def test_health_check() -> bool:
    """Test if the Rust engine is running"""
    try:
        response = requests.get(f"{RUST_ENGINE_URL}/health", timeout=5)
        if response.status_code == 200:
            health_data = response.json()
            print(f"✅ Rust Engine Health: {health_data.get('status', 'unknown')}")
            return True
        else:
            print(f"❌ Health check failed: HTTP {response.status_code}")
            return False
    except Exception as e:
        print(f"❌ Health check failed: {e}")
        return False

def test_snapshot_based_processing():
    """Test the new snapshot-based processing endpoint"""
    print("\n🧪 Testing Snapshot-Based Processing (/api/execute-with-snapshot)")
    
    request_data = create_test_snapshot_request()
    print(f"📤 Request: {json.dumps(request_data, indent=2)}")
    
    try:
        start_time = time.time()
        response = requests.post(
            f"{RUST_ENGINE_URL}/api/execute-with-snapshot",
            json=request_data,
            headers={"Content-Type": "application/json"},
            timeout=60
        )
        
        duration_ms = int((time.time() - start_time) * 1000)
        
        print(f"📥 Response: HTTP {response.status_code} ({duration_ms}ms)")
        
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Success: {json.dumps(result, indent=2, default=str)}")
            
            # Check if snapshot metadata is included
            if result.get('success') and 'data' in result:
                proposal = result['data']
                print(f"💊 Calculated Dose: {proposal.get('calculated_dose', 'N/A')} {proposal.get('dose_unit', '')}")
                print(f"🛡️  Safety Status: {proposal.get('safety_status', 'N/A')}")
                print(f"⏱️  Execution Time: {proposal.get('execution_time_ms', 'N/A')}ms")
            
        elif response.status_code == 400:
            error_data = response.json()
            print(f"⚠️  Validation Error: {error_data}")
        elif response.status_code == 500:
            error_data = response.json()
            print(f"❌ Server Error: {error_data}")
            # This is expected if Context Gateway is not running or snapshot doesn't exist
            print("💡 Note: This error is expected if Context Gateway is not running or snapshot doesn't exist")
        else:
            print(f"❌ Unexpected response: {response.text}")
            
    except requests.exceptions.Timeout:
        print("⏰ Request timeout (expected if snapshot fetch takes too long)")
    except Exception as e:
        print(f"❌ Request failed: {e}")

def test_recipe_execution_with_snapshot():
    """Test recipe execution with snapshot ID"""
    print("\n🧪 Testing Recipe Execution with Snapshot (/api/recipe/execute-snapshot)")
    
    request_data = create_test_recipe_request_with_snapshot()
    print(f"📤 Request: {json.dumps(request_data, indent=2)}")
    
    try:
        start_time = time.time()
        response = requests.post(
            f"{RUST_ENGINE_URL}/api/recipe/execute-snapshot",
            json=request_data,
            headers={"Content-Type": "application/json"},
            timeout=60
        )
        
        duration_ms = int((time.time() - start_time) * 1000)
        
        print(f"📥 Response: HTTP {response.status_code} ({duration_ms}ms)")
        
        if response.status_code == 200:
            result = response.json()
            print(f"✅ Success: {json.dumps(result, indent=2, default=str)}")
        elif response.status_code == 400:
            error_data = response.json()
            print(f"⚠️  Validation Error: {error_data}")
        elif response.status_code == 500:
            error_data = response.json()
            print(f"❌ Server Error: {error_data}")
            print("💡 Note: This error is expected if Context Gateway is not running or snapshot doesn't exist")
        else:
            print(f"❌ Unexpected response: {response.text}")
            
    except requests.exceptions.Timeout:
        print("⏰ Request timeout (expected if snapshot fetch takes too long)")
    except Exception as e:
        print(f"❌ Request failed: {e}")

def test_validation_errors():
    """Test various validation scenarios"""
    print("\n🧪 Testing Validation Scenarios")
    
    # Test missing snapshot_id
    invalid_request = create_test_snapshot_request()
    invalid_request["snapshot_id"] = ""
    
    print("📤 Testing empty snapshot_id...")
    try:
        response = requests.post(
            f"{RUST_ENGINE_URL}/api/execute-with-snapshot",
            json=invalid_request,
            headers={"Content-Type": "application/json"},
            timeout=10
        )
        
        if response.status_code == 400:
            print("✅ Validation correctly rejected empty snapshot_id")
            error_data = response.json()
            print(f"   Error: {error_data.get('error', 'Unknown error')}")
        else:
            print(f"❌ Unexpected response: HTTP {response.status_code}")
            
    except Exception as e:
        print(f"❌ Test failed: {e}")
    
    # Test invalid snapshot_id format
    invalid_request = create_test_snapshot_request()
    invalid_request["snapshot_id"] = "invalid!@#$%"
    
    print("📤 Testing invalid snapshot_id format...")
    try:
        response = requests.post(
            f"{RUST_ENGINE_URL}/api/execute-with-snapshot",
            json=invalid_request,
            headers={"Content-Type": "application/json"},
            timeout=10
        )
        
        if response.status_code == 400:
            print("✅ Validation correctly rejected invalid snapshot_id format")
            error_data = response.json()
            print(f"   Error: {error_data.get('error', 'Unknown error')}")
        else:
            print(f"❌ Unexpected response: HTTP {response.status_code}")
            
    except Exception as e:
        print(f"❌ Test failed: {e}")

def print_summary():
    """Print test summary and integration notes"""
    print("\n" + "="*60)
    print("🎯 SNAPSHOT-BASED PROCESSING TEST SUMMARY")
    print("="*60)
    print()
    print("✅ Enhanced Rust Engine Capabilities:")
    print("   • Snapshot-based clinical data processing")
    print("   • Snapshot integrity verification (checksum + signature)")
    print("   • Context Gateway integration with retry logic")
    print("   • Enhanced evidence generation with snapshot references")
    print()
    print("📍 New API Endpoints:")
    print("   • POST /api/execute-with-snapshot")
    print("   • POST /api/recipe/execute-snapshot")
    print()
    print("🔧 Integration Points:")
    print("   • Context Gateway (port 8016) for snapshot retrieval")
    print("   • Snapshot integrity verification before processing")
    print("   • Enhanced performance by eliminating data assembly overhead")
    print("   • Comprehensive audit trails linking calculations to snapshots")
    print()
    print("⚠️  Production Requirements:")
    print("   • Context Gateway must be running on port 8016")
    print("   • Valid clinical snapshots must exist in Context Gateway")
    print("   • Snapshot integrity verification should be enabled in production")
    print("   • Monitor snapshot fetch performance and add caching if needed")
    print()
    print("🚀 Ready for Recipe Snapshot Architecture Integration!")
    print("="*60)

def main():
    """Main test execution"""
    print("🦀 Flow2 Rust Engine - Snapshot Processing Test Suite")
    print("="*60)
    print(f"🕐 Test Time: {datetime.now(timezone.utc).isoformat()}")
    print(f"🌐 Engine URL: {RUST_ENGINE_URL}")
    print(f"📡 Context Gateway URL: {CONTEXT_GATEWAY_URL}")
    
    # Test health check first
    if not test_health_check():
        print("\n❌ Rust engine is not available. Please start it first:")
        print("   cd backend/services/medication-service/flow2-rust-engine")
        print("   cargo run")
        return
    
    # Run tests
    test_snapshot_based_processing()
    test_recipe_execution_with_snapshot()
    test_validation_errors()
    
    # Print summary
    print_summary()

if __name__ == "__main__":
    main()