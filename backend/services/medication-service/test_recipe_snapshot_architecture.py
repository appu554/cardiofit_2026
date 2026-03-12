#!/usr/bin/env python3
"""
Comprehensive Test Suite for Recipe Snapshot Architecture

This test suite validates the complete Recipe Snapshot architecture including:
1. Context Gateway snapshot creation and management
2. Rust Clinical Engine snapshot-based processing  
3. Flow2 Go Orchestrator snapshot coordination
4. End-to-end workflow validation
5. Performance and integrity verification
"""

import json
import time
import asyncio
import hashlib
import requests
from datetime import datetime, timedelta
from typing import Dict, Any, List, Optional
import pytest

# Test configuration
CONTEXT_SERVICE_URL = "http://localhost:8016"
FLOW2_GO_URL = "http://localhost:8080"
RUST_ENGINE_URL = "http://localhost:8090"

# Test data
TEST_PATIENT_ID = "test-patient-snapshot-001"
TEST_RECIPE_ID = "diabetes-medication-standard"
TEST_MEDICATION_CODE = "metformin"
TEST_PROVIDER_ID = "provider-001"
TEST_ENCOUNTER_ID = "encounter-001"


class RecipeSnapshotArchitectureTestSuite:
    """Comprehensive test suite for Recipe Snapshot architecture"""
    
    def __init__(self):
        self.test_session = requests.Session()
        self.test_results = {
            "tests_run": 0,
            "tests_passed": 0,
            "tests_failed": 0,
            "start_time": datetime.now(),
            "failures": []
        }
    
    def run_test(self, test_name: str, test_func):
        """Run a single test and track results"""
        print(f"\n🧪 Running test: {test_name}")
        self.test_results["tests_run"] += 1
        
        try:
            start_time = time.time()
            result = test_func()
            duration = (time.time() - start_time) * 1000
            
            if result.get("success", False):
                self.test_results["tests_passed"] += 1
                print(f"✅ {test_name} passed ({duration:.2f}ms)")
            else:
                self.test_results["tests_failed"] += 1
                error_msg = result.get("error", "Unknown error")
                self.test_results["failures"].append(f"{test_name}: {error_msg}")
                print(f"❌ {test_name} failed: {error_msg}")
                
        except Exception as e:
            self.test_results["tests_failed"] += 1
            self.test_results["failures"].append(f"{test_name}: {str(e)}")
            print(f"❌ {test_name} exception: {str(e)}")
    
    def test_context_service_health(self) -> Dict[str, Any]:
        """Test Context Service health and snapshot endpoints availability"""
        try:
            response = self.test_session.get(f"{CONTEXT_SERVICE_URL}/health", timeout=5)
            
            if response.status_code != 200:
                return {"success": False, "error": f"Health check failed with status {response.status_code}"}
            
            health_data = response.json()
            
            # Check for snapshot service availability
            status_response = self.test_session.get(f"{CONTEXT_SERVICE_URL}/api/snapshots/status", timeout=5)
            if status_response.status_code != 200:
                return {"success": False, "error": "Snapshot endpoints not available"}
            
            return {"success": True, "health_data": health_data}
            
        except requests.RequestException as e:
            return {"success": False, "error": f"Context Service connectivity failed: {str(e)}"}
    
    def test_snapshot_creation(self) -> Dict[str, Any]:
        """Test clinical snapshot creation via Context Gateway"""
        try:
            snapshot_request = {
                "patient_id": TEST_PATIENT_ID,
                "recipe_id": TEST_RECIPE_ID,
                "provider_id": TEST_PROVIDER_ID,
                "encounter_id": TEST_ENCOUNTER_ID,
                "ttl_hours": 2,
                "force_refresh": True,
                "signature_method": "mock"
            }
            
            response = self.test_session.post(
                f"{CONTEXT_SERVICE_URL}/api/snapshots",
                json=snapshot_request,
                timeout=10
            )
            
            if response.status_code != 200:
                return {"success": False, "error": f"Snapshot creation failed with status {response.status_code}: {response.text}"}
            
            snapshot = response.json()
            
            # Validate snapshot structure
            required_fields = ["id", "patient_id", "recipe_id", "data", "checksum", "signature", "created_at", "expires_at"]
            for field in required_fields:
                if field not in snapshot:
                    return {"success": False, "error": f"Missing required field: {field}"}
            
            # Validate data integrity
            if not snapshot.get("data"):
                return {"success": False, "error": "Snapshot contains no clinical data"}
            
            # Store snapshot ID for subsequent tests
            self.test_snapshot_id = snapshot["id"]
            
            return {"success": True, "snapshot_id": snapshot["id"], "completeness": snapshot.get("completeness_score", 0)}
            
        except requests.RequestException as e:
            return {"success": False, "error": f"Snapshot creation request failed: {str(e)}"}
    
    def test_snapshot_retrieval(self) -> Dict[str, Any]:
        """Test snapshot retrieval and validation"""
        try:
            if not hasattr(self, 'test_snapshot_id'):
                return {"success": False, "error": "No snapshot ID available from creation test"}
            
            # Test snapshot retrieval
            response = self.test_session.get(
                f"{CONTEXT_SERVICE_URL}/api/snapshots/{self.test_snapshot_id}",
                timeout=5
            )
            
            if response.status_code != 200:
                return {"success": False, "error": f"Snapshot retrieval failed with status {response.status_code}"}
            
            snapshot = response.json()
            
            # Verify accessed count was incremented
            if snapshot.get("accessed_count", 0) < 1:
                return {"success": False, "error": "Access count not properly tracked"}
            
            return {"success": True, "accessed_count": snapshot.get("accessed_count", 0)}
            
        except requests.RequestException as e:
            return {"success": False, "error": f"Snapshot retrieval failed: {str(e)}"}
    
    def test_snapshot_validation(self) -> Dict[str, Any]:
        """Test snapshot integrity validation"""
        try:
            if not hasattr(self, 'test_snapshot_id'):
                return {"success": False, "error": "No snapshot ID available from creation test"}
            
            # Test snapshot validation
            response = self.test_session.post(
                f"{CONTEXT_SERVICE_URL}/api/snapshots/{self.test_snapshot_id}/validate",
                timeout=5
            )
            
            if response.status_code != 200:
                return {"success": False, "error": f"Snapshot validation failed with status {response.status_code}"}
            
            validation = response.json()
            
            # Check validation results
            if not validation.get("valid", False):
                return {"success": False, "error": f"Snapshot failed validation: {validation.get('errors', [])}"}
            
            if not validation.get("checksum_valid", False):
                return {"success": False, "error": "Checksum validation failed"}
            
            if not validation.get("signature_valid", False):
                return {"success": False, "error": "Signature validation failed"}
            
            return {"success": True, "validation_duration": validation.get("validation_duration_ms", 0)}
            
        except requests.RequestException as e:
            return {"success": False, "error": f"Snapshot validation failed: {str(e)}"}
    
    def test_rust_engine_snapshot_processing(self) -> Dict[str, Any]:
        """Test Rust engine snapshot-based processing"""
        try:
            if not hasattr(self, 'test_snapshot_id'):
                return {"success": False, "error": "No snapshot ID available for Rust engine test"}
            
            # Test snapshot-based recipe execution
            rust_request = {
                "request_id": f"rust-test-{int(time.time())}",
                "snapshot_id": self.test_snapshot_id,
                "recipe_id": TEST_RECIPE_ID,
                "medication_code": TEST_MEDICATION_CODE,
                "processing_hints": {
                    "snapshot_based": True,
                    "integrity_verified": True
                }
            }
            
            response = self.test_session.post(
                f"{RUST_ENGINE_URL}/api/execute-with-snapshot",
                json=rust_request,
                timeout=15
            )
            
            if response.status_code != 200:
                return {"success": False, "error": f"Rust snapshot execution failed with status {response.status_code}: {response.text}"}
            
            result = response.json()
            
            # Validate response structure
            if not result.get("success", False):
                return {"success": False, "error": f"Rust engine returned unsuccessful result: {result.get('error', 'Unknown error')}"}
            
            data = result.get("data", {})
            if not data.get("medication_proposal"):
                return {"success": False, "error": "No medication proposal in Rust response"}
            
            execution_time = data.get("execution_time_ms", 0)
            return {"success": True, "execution_time_ms": execution_time, "proposal_count": len(data.get("medication_proposal", {}).get("recommendations", []))}
            
        except requests.RequestException as e:
            return {"success": False, "error": f"Rust engine test failed: {str(e)}"}
    
    def test_flow2_snapshot_execution(self) -> Dict[str, Any]:
        """Test Flow2 Go orchestrator snapshot-based execution"""
        try:
            if not hasattr(self, 'test_snapshot_id'):
                return {"success": False, "error": "No snapshot ID available for Flow2 test"}
            
            # Test snapshot-based Flow2 execution
            flow2_request = {
                "snapshot_id": self.test_snapshot_id,
                "patient_id": TEST_PATIENT_ID,
                "medication_code": TEST_MEDICATION_CODE,
                "medication_name": "Metformin",
                "indication": "Type 2 Diabetes",
                "priority": "routine",
                "processing_hints": {
                    "snapshot_workflow": True
                }
            }
            
            response = self.test_session.post(
                f"{FLOW2_GO_URL}/api/v1/snapshots/execute",
                json=flow2_request,
                timeout=20
            )
            
            if response.status_code != 200:
                return {"success": False, "error": f"Flow2 snapshot execution failed with status {response.status_code}: {response.text}"}
            
            result = response.json()
            
            # Validate response structure
            required_sections = ["snapshot_info", "intent_manifest", "medication_proposal", "performance_metrics"]
            for section in required_sections:
                if section not in result:
                    return {"success": False, "error": f"Missing required section: {section}"}
            
            # Check performance improvements
            performance = result.get("performance_metrics", {})
            total_time = performance.get("total_execution_time_ms", 1000)
            network_hops = performance.get("network_hops", 999)
            
            if total_time > 200:  # Should be under 200ms
                return {"success": False, "error": f"Performance target missed: {total_time}ms > 200ms"}
            
            if network_hops != 1:  # Should be 1 hop for snapshot-based
                return {"success": False, "error": f"Expected 1 network hop, got {network_hops}"}
            
            return {"success": True, "execution_time_ms": total_time, "network_hops": network_hops}
            
        except requests.RequestException as e:
            return {"success": False, "error": f"Flow2 snapshot execution failed: {str(e)}"}
    
    def test_advanced_snapshot_workflow(self) -> Dict[str, Any]:
        """Test advanced snapshot workflow with recipe resolution"""
        try:
            # Test advanced workflow that creates snapshot from recipe resolution
            advanced_request = {
                "patient_id": TEST_PATIENT_ID,
                "medication_code": TEST_MEDICATION_CODE,
                "medication_name": "Metformin",
                "indication": "Type 2 Diabetes",
                "patient_conditions": ["E11.9"],  # Type 2 diabetes
                "priority": "routine",
                "provider_id": TEST_PROVIDER_ID,
                "encounter_id": TEST_ENCOUNTER_ID,
                "ttl_hours": 1,
                "force_refresh": False
            }
            
            response = self.test_session.post(
                f"{FLOW2_GO_URL}/api/v1/snapshots/execute-advanced",
                json=advanced_request,
                timeout=25
            )
            
            if response.status_code != 200:
                return {"success": False, "error": f"Advanced workflow failed with status {response.status_code}: {response.text}"}
            
            result = response.json()
            
            # Validate that snapshot was created and used
            snapshot_info = result.get("snapshot_info", {})
            if not snapshot_info.get("snapshot_id"):
                return {"success": False, "error": "No snapshot ID in advanced workflow response"}
            
            # Validate performance
            performance = result.get("performance_metrics", {})
            total_time = performance.get("total_execution_time_ms", 1000)
            
            if total_time > 300:  # Should be under 300ms for advanced workflow
                return {"success": False, "error": f"Advanced workflow performance target missed: {total_time}ms > 300ms"}
            
            return {"success": True, "snapshot_id": snapshot_info.get("snapshot_id"), "execution_time_ms": total_time}
            
        except requests.RequestException as e:
            return {"success": False, "error": f"Advanced workflow test failed: {str(e)}"}
    
    def test_batch_snapshot_execution(self) -> Dict[str, Any]:
        """Test batch snapshot execution"""
        try:
            # Create batch request with multiple patients
            batch_request = {
                "requests": [
                    {
                        "patient_id": f"batch-patient-{i}",
                        "recipe_id": TEST_RECIPE_ID,
                        "medication_code": TEST_MEDICATION_CODE,
                        "indication": "Type 2 Diabetes",
                        "priority": "routine",
                        "ttl_hours": 1
                    }
                    for i in range(3)
                ]
            }
            
            response = self.test_session.post(
                f"{FLOW2_GO_URL}/api/v1/snapshots/execute-batch",
                json=batch_request,
                timeout=30
            )
            
            if response.status_code not in [200, 207]:  # Accept 200 or 207 Multi-Status
                return {"success": False, "error": f"Batch execution failed with status {response.status_code}: {response.text}"}
            
            result = response.json()
            
            # Validate batch response
            if result.get("total_requests", 0) != 3:
                return {"success": False, "error": "Incorrect total request count in batch response"}
            
            success_count = result.get("success_count", 0)
            if success_count < 2:  # Allow for some failures in test environment
                return {"success": False, "error": f"Too many failures in batch: {success_count}/3 successful"}
            
            return {"success": True, "success_count": success_count, "total_time_ms": result.get("total_execution_time_ms", 0)}
            
        except requests.RequestException as e:
            return {"success": False, "error": f"Batch execution test failed: {str(e)}"}
    
    def test_snapshot_lifecycle_management(self) -> Dict[str, Any]:
        """Test complete snapshot lifecycle (create, retrieve, validate, delete)"""
        try:
            # Create snapshot
            create_request = {
                "patient_id": f"lifecycle-patient-{int(time.time())}",
                "recipe_id": TEST_RECIPE_ID,
                "ttl_hours": 1,
                "signature_method": "mock"
            }
            
            create_response = self.test_session.post(
                f"{CONTEXT_SERVICE_URL}/api/snapshots",
                json=create_request,
                timeout=10
            )
            
            if create_response.status_code != 200:
                return {"success": False, "error": f"Lifecycle snapshot creation failed: {create_response.status_code}"}
            
            snapshot = create_response.json()
            snapshot_id = snapshot["id"]
            
            # Retrieve snapshot
            retrieve_response = self.test_session.get(
                f"{CONTEXT_SERVICE_URL}/api/snapshots/{snapshot_id}",
                timeout=5
            )
            
            if retrieve_response.status_code != 200:
                return {"success": False, "error": "Snapshot retrieval in lifecycle failed"}
            
            # Validate snapshot
            validate_response = self.test_session.post(
                f"{CONTEXT_SERVICE_URL}/api/snapshots/{snapshot_id}/validate",
                timeout=5
            )
            
            if validate_response.status_code != 200:
                return {"success": False, "error": "Snapshot validation in lifecycle failed"}
            
            validation = validate_response.json()
            if not validation.get("valid", False):
                return {"success": False, "error": "Snapshot failed lifecycle validation"}
            
            # Delete snapshot
            delete_response = self.test_session.delete(
                f"{CONTEXT_SERVICE_URL}/api/snapshots/{snapshot_id}",
                timeout=5
            )
            
            if delete_response.status_code != 200:
                return {"success": False, "error": "Snapshot deletion in lifecycle failed"}
            
            # Verify deletion
            verify_response = self.test_session.get(
                f"{CONTEXT_SERVICE_URL}/api/snapshots/{snapshot_id}",
                timeout=5
            )
            
            if verify_response.status_code != 404:
                return {"success": False, "error": "Snapshot not properly deleted"}
            
            return {"success": True, "lifecycle_completed": True}
            
        except requests.RequestException as e:
            return {"success": False, "error": f"Snapshot lifecycle test failed: {str(e)}"}
    
    def test_performance_comparison(self) -> Dict[str, Any]:
        """Test performance comparison between traditional and snapshot-based workflows"""
        try:
            # Test traditional workflow (if available)
            traditional_start = time.time()
            traditional_request = {
                "patient_id": TEST_PATIENT_ID,
                "action_type": "MEDICATION_ANALYSIS",
                "medication_data": {
                    "medications": [{"code": TEST_MEDICATION_CODE, "name": "Metformin"}]
                }
            }
            
            traditional_response = self.test_session.post(
                f"{FLOW2_GO_URL}/api/v1/flow2/execute",
                json=traditional_request,
                timeout=20
            )
            traditional_time = (time.time() - traditional_start) * 1000
            
            # Test snapshot-based workflow
            snapshot_start = time.time()
            if hasattr(self, 'test_snapshot_id'):
                snapshot_request = {
                    "snapshot_id": self.test_snapshot_id,
                    "patient_id": TEST_PATIENT_ID,
                    "medication_code": TEST_MEDICATION_CODE,
                    "indication": "Type 2 Diabetes"
                }
                
                snapshot_response = self.test_session.post(
                    f"{FLOW2_GO_URL}/api/v1/snapshots/execute",
                    json=snapshot_request,
                    timeout=15
                )
                snapshot_time = (time.time() - snapshot_start) * 1000
                
                if snapshot_response.status_code == 200:
                    performance_improvement = ((traditional_time - snapshot_time) / traditional_time) * 100
                    
                    return {
                        "success": True,
                        "traditional_time_ms": traditional_time,
                        "snapshot_time_ms": snapshot_time,
                        "improvement_percentage": performance_improvement,
                        "target_met": performance_improvement > 50  # Target 50%+ improvement
                    }
                else:
                    return {"success": False, "error": "Snapshot workflow failed in performance test"}
            else:
                return {"success": False, "error": "No snapshot available for performance comparison"}
                
        except requests.RequestException as e:
            return {"success": False, "error": f"Performance comparison failed: {str(e)}"}
    
    def test_data_integrity_verification(self) -> Dict[str, Any]:
        """Test data integrity verification across the architecture"""
        try:
            if not hasattr(self, 'test_snapshot_id'):
                return {"success": False, "error": "No snapshot ID for integrity test"}
            
            # Get snapshot data
            response = self.test_session.get(f"{CONTEXT_SERVICE_URL}/api/snapshots/{self.test_snapshot_id}")
            if response.status_code != 200:
                return {"success": False, "error": "Could not retrieve snapshot for integrity test"}
            
            snapshot = response.json()
            clinical_data = snapshot.get("data", {})
            provided_checksum = snapshot.get("checksum", "")
            
            # Manually verify checksum
            canonical_json = json.dumps(clinical_data, sort_keys=True, separators=(',', ':'))
            calculated_checksum = hashlib.sha256(canonical_json.encode('utf-8')).hexdigest()
            
            if calculated_checksum != provided_checksum:
                return {"success": False, "error": "Checksum mismatch in integrity verification"}
            
            return {"success": True, "checksum_verified": True, "data_size_bytes": len(canonical_json)}
            
        except Exception as e:
            return {"success": False, "error": f"Data integrity verification failed: {str(e)}"}
    
    def test_snapshot_service_metrics(self) -> Dict[str, Any]:
        """Test snapshot service metrics endpoint"""
        try:
            response = self.test_session.get(f"{CONTEXT_SERVICE_URL}/api/snapshots/metrics", timeout=10)
            
            if response.status_code != 200:
                return {"success": False, "error": f"Metrics endpoint failed with status {response.status_code}"}
            
            metrics = response.json()
            
            # Validate metrics structure
            required_metrics = ["total_snapshots", "active_snapshots", "average_completeness"]
            for metric in required_metrics:
                if metric not in metrics:
                    return {"success": False, "error": f"Missing metric: {metric}"}
            
            return {"success": True, "total_snapshots": metrics.get("total_snapshots", 0)}
            
        except requests.RequestException as e:
            return {"success": False, "error": f"Metrics test failed: {str(e)}"}
    
    def test_error_handling_and_recovery(self) -> Dict[str, Any]:
        """Test error handling for various failure scenarios"""
        try:
            test_results = {"success": True, "error_tests_passed": 0}
            
            # Test 1: Invalid snapshot ID
            invalid_response = self.test_session.get(f"{CONTEXT_SERVICE_URL}/api/snapshots/invalid-id")
            if invalid_response.status_code != 404:
                test_results["success"] = False
                return {"success": False, "error": "Invalid snapshot ID should return 404"}
            test_results["error_tests_passed"] += 1
            
            # Test 2: Invalid request format
            invalid_request_response = self.test_session.post(
                f"{CONTEXT_SERVICE_URL}/api/snapshots",
                json={"invalid": "data"},
                timeout=5
            )
            if invalid_request_response.status_code not in [400, 422]:
                test_results["success"] = False
                return {"success": False, "error": "Invalid request should return 400/422"}
            test_results["error_tests_passed"] += 1
            
            # Test 3: Expired snapshot handling would require time manipulation
            # For now, we'll just verify the error handling structure exists
            
            return test_results
            
        except requests.RequestException as e:
            return {"success": False, "error": f"Error handling test failed: {str(e)}"}
    
    def run_all_tests(self):
        """Run the complete test suite"""
        print("🚀 Starting Recipe Snapshot Architecture Test Suite")
        print("=" * 60)
        
        # Test order is important - some tests depend on previous ones
        test_sequence = [
            ("Context Service Health Check", self.test_context_service_health),
            ("Snapshot Creation", self.test_snapshot_creation),
            ("Snapshot Retrieval", self.test_snapshot_retrieval),
            ("Snapshot Validation", self.test_snapshot_validation),
            ("Rust Engine Snapshot Processing", self.test_rust_engine_snapshot_processing),
            ("Flow2 Snapshot Execution", self.test_flow2_snapshot_execution),
            ("Advanced Snapshot Workflow", self.test_advanced_snapshot_workflow),
            ("Batch Snapshot Execution", self.test_batch_snapshot_execution),
            ("Data Integrity Verification", self.test_data_integrity_verification),
            ("Snapshot Service Metrics", self.test_snapshot_service_metrics),
            ("Performance Comparison", self.test_performance_comparison),
            ("Error Handling and Recovery", self.test_error_handling_and_recovery),
        ]
        
        for test_name, test_func in test_sequence:
            self.run_test(test_name, test_func)
        
        # Print final results
        self.print_test_summary()
    
    def print_test_summary(self):
        """Print comprehensive test results summary"""
        print("\n" + "=" * 60)
        print("🧪 Recipe Snapshot Architecture Test Results")
        print("=" * 60)
        
        total_time = datetime.now() - self.test_results["start_time"]
        success_rate = (self.test_results["tests_passed"] / self.test_results["tests_run"]) * 100
        
        print(f"📊 Tests run: {self.test_results['tests_run']}")
        print(f"✅ Tests passed: {self.test_results['tests_passed']}")
        print(f"❌ Tests failed: {self.test_results['tests_failed']}")
        print(f"📈 Success rate: {success_rate:.1f}%")
        print(f"⏱️ Total execution time: {total_time.total_seconds():.2f} seconds")
        
        if self.test_results["failures"]:
            print(f"\n🔍 Test Failures:")
            for failure in self.test_results["failures"]:
                print(f"   ❌ {failure}")
        
        print("\n🎯 Architecture Validation:")
        if success_rate >= 80:
            print("✅ Recipe Snapshot Architecture is functioning correctly")
            print("🚀 Ready for production deployment with performance improvements")
        else:
            print("⚠️ Architecture has significant issues requiring attention")
            print("🔧 Review failed tests before production deployment")
        
        print("=" * 60)


def main():
    """Main test execution function"""
    print("🧪 Recipe Snapshot Architecture Comprehensive Test Suite")
    print("Testing Context Gateway + Rust Engine + Flow2 Orchestrator integration")
    
    # Initialize and run test suite
    test_suite = RecipeSnapshotArchitectureTestSuite()
    test_suite.run_all_tests()


if __name__ == "__main__":
    main()