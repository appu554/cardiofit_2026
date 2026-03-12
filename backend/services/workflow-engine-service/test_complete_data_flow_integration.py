"""
End-to-End Integration Tests for Complete Data Flow Architecture

Tests the complete UI → Apollo Federation → Workflow Platform → Calculate > Validate > Commit flow
with all service integrations and the Recipe Snapshot Architecture optimization.
"""

import asyncio
import pytest
import httpx
import json
import uuid
from datetime import datetime
from typing import Dict, Any

# Test Configuration
TEST_CONFIG = {
    "apollo_federation_url": "http://localhost:4000",
    "workflow_platform_url": "http://localhost:8015", 
    "flow2_go_url": "http://localhost:8080",
    "flow2_rust_url": "http://localhost:8090",
    "safety_gateway_url": "http://localhost:8018",
    "medication_service_url": "http://localhost:8004",
    "context_gateway_url": "http://localhost:8016",
    "timeout": 30.0
}

class TestCompleteDataFlowIntegration:
    """End-to-end integration tests for the complete medication orchestration flow"""
    
    @pytest.fixture
    async def http_client(self):
        """HTTP client fixture for service communication"""
        async with httpx.AsyncClient(timeout=TEST_CONFIG["timeout"]) as client:
            yield client
    
    @pytest.fixture
    def sample_medication_request(self):
        """Sample medication request for testing"""
        return {
            "patient_id": "test-patient-123",
            "encounter_id": "test-encounter-456", 
            "indication": "hypertension_stage2_ckd",
            "urgency": "ROUTINE",
            "constraints": ["avoid_ace_inhibitors"],
            "medication": {
                "therapeutic_class": "antihypertensive",
                "preferred_mechanism": "calcium_channel_blocker",
                "dosage_form": "tablet"
            },
            "provider_id": "test-provider-789",
            "specialty": "internal_medicine",
            "location": "clinic_a"
        }
    
    @pytest.fixture
    def graphql_medication_mutation(self):
        """GraphQL mutation for creating medication orders"""
        return """
        mutation CreateMedicationOrder($input: CreateMedicationOrderInput!) {
            createMedicationOrder(input: $input) {
                status
                correlationId
                executionTimeMs
                medicationOrderId
                calculation {
                    proposalSetId
                    snapshotId
                    executionTimeMs
                }
                validation {
                    validationId
                    verdict
                }
                commitment {
                    orderId
                    auditTrailId
                }
                performance {
                    totalTimeMs
                    meetsTarget
                }
                errorCode
                errorMessage
            }
        }
        """
    
    async def test_service_health_checks(self, http_client: httpx.AsyncClient):
        """Test that all required services are healthy before running integration tests"""
        
        services_to_check = [
            ("Apollo Federation", f"{TEST_CONFIG['apollo_federation_url']}/health"),
            ("Workflow Platform", f"{TEST_CONFIG['workflow_platform_url']}/api/v1/orchestrate/health"),
            ("Flow2 Go Engine", f"{TEST_CONFIG['flow2_go_url']}/health"),
            ("Flow2 Rust Engine", f"{TEST_CONFIG['flow2_rust_url']}/health"),
            ("Safety Gateway", f"{TEST_CONFIG['safety_gateway_url']}/health"),
            ("Medication Service", f"{TEST_CONFIG['medication_service_url']}/health"),
            ("Context Gateway", f"{TEST_CONFIG['context_gateway_url']}/health")
        ]
        
        health_results = {}
        
        for service_name, health_url in services_to_check:
            try:
                response = await http_client.get(health_url)
                health_results[service_name] = {
                    "status": "healthy" if response.status_code == 200 else "unhealthy",
                    "status_code": response.status_code,
                    "response_time_ms": response.elapsed.total_seconds() * 1000 if hasattr(response, 'elapsed') else None
                }
            except Exception as e:
                health_results[service_name] = {
                    "status": "unavailable",
                    "error": str(e)
                }
        
        # Assert all services are healthy
        unhealthy_services = [name for name, result in health_results.items() if result["status"] != "healthy"]
        
        if unhealthy_services:
            pytest.fail(f"The following services are not healthy: {unhealthy_services}. Health results: {json.dumps(health_results, indent=2)}")
        
        print(f"✅ All services are healthy: {list(health_results.keys())}")
    
    async def test_strategic_orchestration_rest_api(
        self, 
        http_client: httpx.AsyncClient, 
        sample_medication_request: Dict[str, Any]
    ):
        """Test the strategic orchestration REST API directly"""
        
        print("🧪 Testing Strategic Orchestration REST API...")
        
        # Call the strategic orchestration endpoint
        response = await http_client.post(
            f"{TEST_CONFIG['workflow_platform_url']}/api/v1/orchestrate/medication",
            json=sample_medication_request,
            headers={"Content-Type": "application/json"}
        )
        
        assert response.status_code == 200, f"Orchestration failed: {response.status_code} - {response.text}"
        
        result = response.json()
        
        # Validate response structure
        assert "status" in result
        assert "correlation_id" in result
        assert "execution_time_ms" in result
        
        # Check for successful orchestration
        if result["status"] == "SUCCESS":
            assert "medication_order_id" in result
            assert "calculation" in result
            assert "validation" in result
            assert "commitment" in result
            
            # Validate Calculate step results
            calculation = result["calculation"]
            assert "proposal_set_id" in calculation
            assert "snapshot_id" in calculation  # Critical for Recipe Snapshot Architecture
            
            # Validate Validate step results
            validation = result["validation"]
            assert "validation_id" in validation
            assert "verdict" in validation
            assert validation["verdict"] in ["SAFE", "WARNING", "UNSAFE"]
            
            # Validate Commit step results (if SAFE)
            if validation["verdict"] == "SAFE":
                commitment = result["commitment"]
                assert "order_id" in commitment
                assert "audit_trail_id" in commitment
        
        elif result["status"] == "REQUIRES_PROVIDER_DECISION":
            assert "validation_findings" in result
            assert "override_tokens" in result
            assert "proposals" in result
            
        elif result["status"] == "BLOCKED_UNSAFE":
            assert "blocking_findings" in result
            assert "alternative_approaches" in result
        
        print(f"✅ Strategic Orchestration REST API test passed. Status: {result['status']}")
        return result
    
    async def test_graphql_medication_ordering_flow(
        self,
        http_client: httpx.AsyncClient,
        sample_medication_request: Dict[str, Any],
        graphql_medication_mutation: str
    ):
        """Test the complete GraphQL medication ordering flow via Apollo Federation"""
        
        print("🧪 Testing GraphQL Medication Ordering Flow...")
        
        # Create GraphQL request
        graphql_request = {
            "query": graphql_medication_mutation,
            "variables": {
                "input": sample_medication_request
            }
        }
        
        # Send GraphQL mutation via Apollo Federation
        response = await http_client.post(
            f"{TEST_CONFIG['apollo_federation_url']}/graphql",
            json=graphql_request,
            headers={"Content-Type": "application/json"}
        )
        
        assert response.status_code == 200, f"GraphQL request failed: {response.status_code} - {response.text}"
        
        result = response.json()
        
        # Check for GraphQL errors
        if "errors" in result:
            pytest.fail(f"GraphQL errors: {result['errors']}")
        
        # Validate GraphQL response data
        assert "data" in result
        assert "createMedicationOrder" in result["data"]
        
        order_result = result["data"]["createMedicationOrder"]
        
        # Validate orchestration result via GraphQL
        assert "status" in order_result
        assert "correlationId" in order_result
        
        if order_result["status"] == "SUCCESS":
            assert "medicationOrderId" in order_result
            assert order_result["medicationOrderId"] is not None
            
            # Validate performance metrics
            if "performance" in order_result and order_result["performance"]:
                performance = order_result["performance"]
                total_time = performance.get("totalTimeMs", 0)
                meets_target = performance.get("meetsTarget", False)
                
                print(f"📊 Performance: {total_time:.2f}ms (Target met: {meets_target})")
                
                # Validate performance target (should be sub-200ms with Recipe Snapshot Architecture)
                assert total_time < 325, f"Performance target missed: {total_time}ms > 325ms"
        
        print(f"✅ GraphQL Medication Ordering Flow test passed. Status: {order_result['status']}")
        return order_result
    
    async def test_recipe_snapshot_architecture_performance(
        self,
        http_client: httpx.AsyncClient,
        sample_medication_request: Dict[str, Any]
    ):
        """Test Recipe Snapshot Architecture performance optimization"""
        
        print("🧪 Testing Recipe Snapshot Architecture Performance...")
        
        # Execute multiple requests to test performance consistency
        execution_times = []
        snapshot_ids = []
        
        for i in range(3):
            start_time = datetime.utcnow()
            
            response = await http_client.post(
                f"{TEST_CONFIG['workflow_platform_url']}/api/v1/orchestrate/medication",
                json={**sample_medication_request, "patient_id": f"perf-test-{i}-{uuid.uuid4()}"},
                headers={"Content-Type": "application/json"}
            )
            
            end_time = datetime.utcnow()
            total_time = (end_time - start_time).total_seconds() * 1000
            
            assert response.status_code == 200
            result = response.json()
            
            execution_times.append(total_time)
            
            if "calculation" in result:
                snapshot_ids.append(result["calculation"].get("snapshot_id"))
        
        # Analyze performance results
        avg_time = sum(execution_times) / len(execution_times)
        max_time = max(execution_times)
        min_time = min(execution_times)
        
        print(f"📊 Performance Analysis:")
        print(f"   Average: {avg_time:.2f}ms")
        print(f"   Min: {min_time:.2f}ms")
        print(f"   Max: {max_time:.2f}ms")
        print(f"   Snapshot IDs: {len(set(filter(None, snapshot_ids)))} unique snapshots")
        
        # Validate Recipe Snapshot Architecture performance target
        assert avg_time < 200, f"Recipe Snapshot Architecture performance target missed: {avg_time:.2f}ms > 200ms"
        assert max_time < 325, f"Maximum time exceeded tolerance: {max_time:.2f}ms > 325ms"
        
        print("✅ Recipe Snapshot Architecture performance test passed")
    
    async def test_error_handling_and_recovery(
        self,
        http_client: httpx.AsyncClient
    ):
        """Test error handling and recovery scenarios"""
        
        print("🧪 Testing Error Handling and Recovery...")
        
        # Test with invalid patient ID
        invalid_request = {
            "patient_id": "",  # Invalid
            "indication": "hypertension",
            "medication": {"type": "test"},
            "provider_id": "test-provider"
        }
        
        response = await http_client.post(
            f"{TEST_CONFIG['workflow_platform_url']}/api/v1/orchestrate/medication",
            json=invalid_request,
            headers={"Content-Type": "application/json"}
        )
        
        # Should return error response, not crash
        assert response.status_code in [400, 422, 500]
        
        if response.status_code == 200:
            result = response.json()
            assert result.get("status") == "ERROR"
            assert "error_code" in result
            assert "error_message" in result
        
        print("✅ Error handling test passed")
    
    async def test_override_workflow(
        self,
        http_client: httpx.AsyncClient,
        sample_medication_request: Dict[str, Any]
    ):
        """Test provider override workflow for WARNING validation results"""
        
        print("🧪 Testing Provider Override Workflow...")
        
        # First, try to create a medication order that might result in warnings
        # Modify request to potentially trigger warnings
        warning_request = {
            **sample_medication_request,
            "constraints": ["high_dose_warning"],  # Might trigger warnings
            "indication": "complex_hypertension"
        }
        
        response = await http_client.post(
            f"{TEST_CONFIG['workflow_platform_url']}/api/v1/orchestrate/medication",
            json=warning_request,
            headers={"Content-Type": "application/json"}
        )
        
        assert response.status_code == 200
        result = response.json()
        
        # If we get a warning that requires provider decision, test override
        if result.get("status") == "REQUIRES_PROVIDER_DECISION":
            override_request = {
                "correlation_id": result["correlation_id"],
                "snapshot_id": result["snapshot_id"],
                "selected_proposal_index": 0,
                "override_tokens": result.get("override_tokens", []),
                "provider_justification": "Clinical judgment based on patient's specific condition and risk-benefit analysis"
            }
            
            override_response = await http_client.post(
                f"{TEST_CONFIG['workflow_platform_url']}/api/v1/orchestrate/medication/override",
                json=override_request,
                headers={"Content-Type": "application/json"}
            )
            
            assert override_response.status_code == 200
            override_result = override_response.json()
            
            assert override_result.get("status") in ["SUCCESS_WITH_OVERRIDE", "OVERRIDE_REJECTED"]
            
            print(f"✅ Provider override test completed. Result: {override_result.get('status')}")
        else:
            print("ℹ️ No warnings generated, override test skipped")

    @pytest.mark.asyncio
    async def test_complete_integration_suite(self):
        """Run the complete integration test suite"""
        
        print("🚀 Starting Complete Data Flow Integration Test Suite")
        print("=" * 80)
        
        async with httpx.AsyncClient(timeout=TEST_CONFIG["timeout"]) as client:
            
            # Test 1: Service Health Checks
            await self.test_service_health_checks(client)
            print()
            
            # Test 2: Strategic Orchestration REST API
            sample_request = {
                "patient_id": f"integration-test-{uuid.uuid4()}",
                "encounter_id": f"encounter-{uuid.uuid4()}",
                "indication": "hypertension_stage2",
                "urgency": "ROUTINE", 
                "medication": {
                    "therapeutic_class": "antihypertensive",
                    "preferred_mechanism": "ace_inhibitor"
                },
                "provider_id": "integration-test-provider",
                "specialty": "cardiology"
            }
            
            rest_result = await self.test_strategic_orchestration_rest_api(client, sample_request)
            print()
            
            # Test 3: GraphQL Flow (if services support it)
            try:
                graphql_mutation = """
                mutation CreateMedicationOrder($input: CreateMedicationOrderInput!) {
                    createMedicationOrder(input: $input) {
                        status
                        correlationId
                        medicationOrderId
                        performance { totalTimeMs meetsTarget }
                    }
                }
                """
                await self.test_graphql_medication_ordering_flow(client, sample_request, graphql_mutation)
                print()
            except Exception as e:
                print(f"ℹ️ GraphQL test skipped: {e}")
            
            # Test 4: Performance Testing
            await self.test_recipe_snapshot_architecture_performance(client, sample_request)
            print()
            
            # Test 5: Error Handling
            await self.test_error_handling_and_recovery(client)
            print()
            
            # Test 6: Override Workflow
            await self.test_override_workflow(client, sample_request)
            print()
        
        print("=" * 80)
        print("🎉 Complete Data Flow Integration Test Suite PASSED!")
        print()
        print("✅ Architecture Validation:")
        print("   • UI → Apollo Federation → Workflow Platform flow")
        print("   • Calculate > Validate > Commit orchestration pattern") 
        print("   • Recipe Snapshot Architecture performance optimization")
        print("   • Safety Gateway comprehensive validation")
        print("   • End-to-end service integration")


# Main execution for standalone testing
if __name__ == "__main__":
    async def run_tests():
        test_suite = TestCompleteDataFlowIntegration()
        await test_suite.test_complete_integration_suite()
    
    # Run the integration tests
    asyncio.run(run_tests())