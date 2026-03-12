#!/usr/bin/env python3
"""
Context Service Snapshot Integration Tests

Comprehensive test suite for the snapshot functionality in the Context Service,
validating snapshot creation, integrity verification, and API functionality.
"""

import asyncio
import json
import hashlib
import pytest
from datetime import datetime, timedelta
from fastapi.testclient import TestClient
from unittest.mock import Mock, patch

# Import the FastAPI app and services
from app.main import app
from app.services.snapshot_service import SnapshotService, CryptographicService
from app.models.snapshot_models import (
    SnapshotRequest,
    ClinicalSnapshot,
    SnapshotValidationResult,
    SnapshotStatus,
    SignatureMethod
)

# Create test client
client = TestClient(app)

# Test data
TEST_PATIENT_ID = "patient-snapshot-test-001"
TEST_RECIPE_ID = "diabetes-standard"
TEST_PROVIDER_ID = "provider-test-001"


class TestSnapshotService:
    """Test suite for snapshot service functionality"""
    
    @pytest.fixture
    def snapshot_service(self):
        """Create a snapshot service instance for testing"""
        return SnapshotService()
    
    @pytest.fixture
    def sample_clinical_data(self):
        """Sample clinical data for testing"""
        return {
            "demographics": {
                "age": 45,
                "gender": "M",
                "weight_kg": 80.5
            },
            "medications": [
                {
                    "code": "metformin",
                    "name": "Metformin",
                    "dose_mg": 500,
                    "frequency": "BID"
                }
            ],
            "conditions": [
                {
                    "code": "E11.9",
                    "name": "Type 2 diabetes mellitus"
                }
            ],
            "lab_results": {
                "HbA1c": 7.2,
                "eGFR": 85,
                "creatinine": 1.1
            }
        }
    
    def test_cryptographic_service_checksum(self, sample_clinical_data):
        """Test checksum calculation and verification"""
        crypto_service = CryptographicService()
        
        # Calculate checksum
        checksum = crypto_service.calculate_checksum(sample_clinical_data)
        assert isinstance(checksum, str)
        assert len(checksum) == 64  # SHA-256 produces 64-character hex string
        
        # Verify checksum
        is_valid = crypto_service.verify_checksum(sample_clinical_data, checksum)
        assert is_valid
        
        # Test with modified data
        modified_data = sample_clinical_data.copy()
        modified_data["demographics"]["age"] = 46
        is_valid_modified = crypto_service.verify_checksum(modified_data, checksum)
        assert not is_valid_modified
    
    def test_cryptographic_service_signature(self, sample_clinical_data):
        """Test digital signature creation and verification"""
        crypto_service = CryptographicService()
        
        # Create signature
        signature = crypto_service.create_signature(sample_clinical_data, SignatureMethod.MOCK)
        assert isinstance(signature, str)
        assert signature.startswith("MOCK_SIGNATURE_")
        
        # Verify signature
        is_valid = crypto_service.verify_signature(sample_clinical_data, signature, SignatureMethod.MOCK)
        assert is_valid
        
        # Test with modified data
        modified_data = sample_clinical_data.copy()
        modified_data["demographics"]["weight_kg"] = 85.0
        is_valid_modified = crypto_service.verify_signature(modified_data, signature, SignatureMethod.MOCK)
        assert not is_valid_modified


class TestSnapshotAPI:
    """Test suite for snapshot REST API endpoints"""
    
    def test_snapshot_creation_api(self):
        """Test snapshot creation via REST API"""
        snapshot_request = {
            "patient_id": TEST_PATIENT_ID,
            "recipe_id": TEST_RECIPE_ID,
            "provider_id": TEST_PROVIDER_ID,
            "ttl_hours": 2,
            "force_refresh": True,
            "signature_method": "mock"
        }
        
        # Mock the context assembly to avoid external dependencies
        with patch('app.services.context_assembly_service.ContextAssemblyService') as mock_context_service:
            # Mock context assembly result
            mock_result = Mock()
            mock_result.context_id = "test-context-001"
            mock_result.assembled_data = {
                "demographics": {"age": 45, "gender": "M"},
                "medications": []
            }
            mock_result.completeness_score = 0.95
            mock_result.assembly_duration_ms = 150
            mock_result.cache_hit = False
            mock_result.source_metadata = {}
            mock_result.safety_flags = []
            
            mock_context_service.return_value.assemble_context.return_value = mock_result
            
            response = client.post("/api/snapshots", json=snapshot_request)
            
            assert response.status_code == 200
            snapshot_data = response.json()
            
            # Validate response structure
            assert "id" in snapshot_data
            assert snapshot_data["patient_id"] == TEST_PATIENT_ID
            assert snapshot_data["recipe_id"] == TEST_RECIPE_ID
            assert "checksum" in snapshot_data
            assert "signature" in snapshot_data
            assert "created_at" in snapshot_data
            assert "expires_at" in snapshot_data
    
    def test_snapshot_list_api(self):
        """Test snapshot listing API"""
        response = client.get("/api/snapshots")
        assert response.status_code == 200
        
        snapshots = response.json()
        assert isinstance(snapshots, list)
    
    def test_snapshot_metrics_api(self):
        """Test snapshot metrics API"""
        response = client.get("/api/snapshots/metrics")
        assert response.status_code == 200
        
        metrics = response.json()
        assert "total_snapshots" in metrics
        assert "active_snapshots" in metrics
        assert "average_completeness" in metrics
    
    def test_snapshot_service_status(self):
        """Test snapshot service status endpoint"""
        response = client.get("/api/snapshots/status")
        assert response.status_code == 200
        
        status = response.json()
        assert status["service"] == "clinical-snapshot-service"
        assert "features" in status
        assert "endpoints" in status


class TestEndToEndWorkflow:
    """Test suite for end-to-end snapshot workflows"""
    
    @pytest.mark.asyncio
    async def test_complete_snapshot_workflow(self):
        """Test complete snapshot workflow from creation to processing"""
        # This would test the complete flow:
        # 1. Create snapshot via Context Gateway
        # 2. Use snapshot in Rust engine
        # 3. Coordinate via Flow2 orchestrator
        
        # Mock external services for testing
        with patch('app.services.context_assembly_service.ContextAssemblyService') as mock_context:
            mock_result = Mock()
            mock_result.context_id = "workflow-test-001"
            mock_result.assembled_data = {
                "patient_id": TEST_PATIENT_ID,
                "demographics": {"age": 55, "gender": "F", "weight": 70},
                "medications": [],
                "conditions": [{"code": "E11.9", "name": "Type 2 diabetes"}],
                "lab_results": {"HbA1c": 8.1, "eGFR": 75}
            }
            mock_result.completeness_score = 0.92
            mock_result.assembly_duration_ms = 180
            mock_result.cache_hit = False
            mock_result.source_metadata = {}
            mock_result.safety_flags = []
            
            mock_context.return_value.assemble_context.return_value = mock_result
            
            # Test snapshot creation
            snapshot_request = {
                "patient_id": TEST_PATIENT_ID,
                "recipe_id": "diabetes-comprehensive",
                "ttl_hours": 1,
                "signature_method": "mock"
            }
            
            create_response = client.post("/api/snapshots", json=snapshot_request)
            assert create_response.status_code == 200
            
            snapshot = create_response.json()
            snapshot_id = snapshot["id"]
            
            # Test snapshot retrieval
            get_response = client.get(f"/api/snapshots/{snapshot_id}")
            assert get_response.status_code == 200
            
            # Test snapshot validation
            validate_response = client.post(f"/api/snapshots/{snapshot_id}/validate")
            assert validate_response.status_code == 200
            
            validation = validate_response.json()
            assert validation["valid"]
            assert validation["checksum_valid"]
            assert validation["signature_valid"]


def run_integration_tests():
    """Run all integration tests"""
    print("🧪 Starting Context Service Snapshot Integration Tests")
    print("=" * 55)
    
    # Run tests using pytest
    import subprocess
    import sys
    
    try:
        result = subprocess.run([
            sys.executable, "-m", "pytest", __file__, "-v", "--tb=short"
        ], capture_output=True, text=True, cwd=".")
        
        print("📊 Test Results:")
        print(result.stdout)
        
        if result.stderr:
            print("⚠️ Warnings/Errors:")
            print(result.stderr)
        
        return result.returncode == 0
        
    except Exception as e:
        print(f"❌ Test execution failed: {e}")
        return False


def main():
    """Main function for running tests"""
    print("🔬 Context Service Snapshot Integration Test Suite")
    
    # Check if services are running
    try:
        health_response = client.get("/health")
        if health_response.status_code == 200:
            print("✅ Context Service is running")
        else:
            print("⚠️ Context Service health check failed")
    except Exception as e:
        print(f"❌ Context Service not accessible: {e}")
        return False
    
    # Run integration tests
    success = run_integration_tests()
    
    if success:
        print("\n✅ All Context Service snapshot integration tests passed")
        print("🚀 Snapshot functionality is ready for use")
    else:
        print("\n❌ Some tests failed - review output above")
        print("🔧 Fix issues before using snapshot functionality")
    
    return success


if __name__ == "__main__":
    main()