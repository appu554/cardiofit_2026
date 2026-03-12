"""
End-to-End Integration Tests for Calculate > Validate > Commit Workflow.

These tests verify the complete medication workflow from initial request
through calculation, validation, and final commitment.
"""
import pytest
import asyncio
import uuid
import json
import httpx
from datetime import datetime, timezone, timedelta
from typing import Dict, Any, List
from unittest.mock import Mock, patch, AsyncMock

from app.orchestration.strategic_orchestrator import (
    StrategicOrchestrator,
    CalculateRequest,
    OrchestrationResult
)
from app.integration.safety_gateway_http_client import (
    SafetyGatewayHTTPClient,
    SafetyValidationRequest,
    ValidationVerdict
)


class TestCalculateValidateCommitFlow:
    """Integration tests for the complete workflow."""
    
    @pytest.fixture
    def orchestrator(self):
        """Create strategic orchestrator instance."""
        return StrategicOrchestrator()
    
    @pytest.fixture
    def sample_calculate_request(self):
        """Sample calculate request for testing."""
        return CalculateRequest(
            patient_id="patient_12345",
            medication_request={
                "medication_code": "213269",  # Lisinopril
                "medication_name": "Lisinopril 10mg",
                "dosage": "10mg",
                "frequency": "once daily",
                "route": "oral",
                "indication": "Hypertension"
            },
            clinical_intent={
                "primary_indication": "hypertension",
                "target_bp": "< 140/90",
                "treatment_goal": "blood_pressure_control"
            },
            provider_context={
                "provider_id": "provider_789",
                "specialty": "internal_medicine",
                "experience_level": "attending"
            },
            correlation_id=f"test_{uuid.uuid4().hex[:8]}",
            urgency="ROUTINE"
        )
    
    @pytest.fixture
    def mock_services(self):
        """Mock external services for testing."""
        return {
            "flow2_go": Mock(),
            "safety_gateway": Mock(),
            "medication_service": Mock(),
            "context_gateway": Mock()
        }
    
    @pytest.mark.asyncio
    async def test_complete_workflow_success_path(
        self, 
        orchestrator: StrategicOrchestrator,
        sample_calculate_request: CalculateRequest,
        mock_services: Dict[str, Mock]
    ):
        """
        Test the complete workflow success path: Calculate > Validate > Commit
        """
        # ARRANGE
        correlation_id = sample_calculate_request.correlation_id
        
        # Mock Flow2 Go Engine response (CALCULATE phase)
        mock_calculate_response = {
            "proposal_set_id": f"propset_{uuid.uuid4().hex[:12]}",
            "snapshot_id": f"snap_{uuid.uuid4().hex[:12]}",
            "ranked_proposals": [
                {
                    "proposal_id": f"prop_{uuid.uuid4().hex[:12]}",
                    "medication_code": "213269",
                    "medication_name": "Lisinopril 10mg",
                    "dosage": "10mg",
                    "frequency": "once daily",
                    "route": "oral",
                    "confidence_score": 0.95,
                    "clinical_evidence": {
                        "hypertension_efficacy": "high",
                        "side_effect_profile": "low",
                        "drug_interactions": "none_detected"
                    }
                },
                {
                    "proposal_id": f"prop_{uuid.uuid4().hex[:12]}",
                    "medication_code": "213269",
                    "medication_name": "Lisinopril 5mg",
                    "dosage": "5mg",
                    "frequency": "once daily",
                    "route": "oral",
                    "confidence_score": 0.87,
                    "clinical_evidence": {
                        "hypertension_efficacy": "moderate",
                        "side_effect_profile": "very_low"
                    }
                }
            ],
            "clinical_evidence": {
                "patient_factors": {
                    "age": 55,
                    "weight": 75,
                    "renal_function": "normal",
                    "allergies": []
                },
                "medication_factors": {
                    "first_line_therapy": True,
                    "ace_inhibitor_class": True,
                    "evidence_level": "A"
                }
            },
            "monitoring_plan": {
                "initial_followup": "2_weeks",
                "monitoring_parameters": ["blood_pressure", "potassium", "creatinine"],
                "target_response": "systolic_reduction_10_20_mmhg"
            },
            "kb_versions": {
                "drug_rules": "v2.4.1",
                "guideline_evidence": "v1.8.3",
                "interaction_db": "v3.1.0"
            }
        }
        
        # Mock Safety Gateway response (VALIDATE phase)
        mock_validation_response = {
            "validation_id": f"val_{uuid.uuid4().hex[:12]}",
            "verdict": "SAFE",
            "findings": [],
            "override_tokens": None,
            "override_requirements": None,
            "processing_time_ms": 85,
            "engine_results": [
                {
                    "engine_id": "cae_engine",
                    "engine_name": "Clinical Assertion Engine",
                    "status": "SUCCESS",
                    "risk_score": 0.1,
                    "violations": [],
                    "warnings": [],
                    "confidence": 0.98,
                    "duration_ms": 65,
                    "tier": 1
                }
            ],
            "risk_score": 0.1,
            "timestamp": datetime.now(timezone.utc).isoformat()
        }
        
        # Mock Medication Service response (COMMIT phase)
        mock_commit_response = {
            "medication_order_id": f"order_{uuid.uuid4().hex[:12]}",
            "persistence_status": "SUCCESS",
            "event_publication_status": "SUCCESS",
            "audit_trail_id": f"audit_{uuid.uuid4().hex[:16]}",
            "result": "SUCCESS",
            "processing_time_ms": 45,
            "fhir_result": {
                "id": f"MedicationRequest_{uuid.uuid4().hex[:12]}",
                "resourceType": "MedicationRequest",
                "status": "active"
            }
        }
        
        # Mock HTTP clients
        with patch('httpx.AsyncClient') as mock_http_client:
            # Configure Flow2 Go response
            mock_flow2_response = Mock()
            mock_flow2_response.status_code = 200
            mock_flow2_response.json.return_value = mock_calculate_response
            
            # Configure Safety Gateway response
            mock_safety_response = Mock()
            mock_safety_response.status_code = 200
            mock_safety_response.json.return_value = mock_validation_response
            
            # Configure Medication Service response
            mock_medication_response = Mock()
            mock_medication_response.status_code = 200
            mock_medication_response.json.return_value = mock_commit_response
            
            # Configure HTTP client mock to return appropriate responses
            mock_client_instance = Mock()
            mock_http_client.return_value = mock_client_instance
            
            def mock_post_side_effect(url, **kwargs):
                if "flow2-go" in url or ":8080" in url:
                    return asyncio.create_future().set_result(mock_flow2_response)
                elif "safety-gateway" in url or ":8018" in url:
                    return asyncio.create_future().set_result(mock_safety_response)
                elif "medication-service" in url or ":8004" in url:
                    return asyncio.create_future().set_result(mock_medication_response)
                else:
                    raise ValueError(f"Unexpected URL: {url}")
            
            mock_client_instance.post = AsyncMock(side_effect=mock_post_side_effect)
            
            # ACT
            start_time = datetime.now(timezone.utc)
            result = await orchestrator.orchestrate_medication_request(sample_calculate_request)
            end_time = datetime.now(timezone.utc)
            total_time_ms = (end_time - start_time).total_seconds() * 1000
            
            # ASSERT
            assert result["status"] == "SUCCESS"
            assert result["correlation_id"] == correlation_id
            assert "medication_order_id" in result
            assert "calculation" in result
            assert "validation" in result
            assert "commitment" in result
            assert "performance" in result
            
            # Verify calculation phase
            calculation = result["calculation"]
            assert calculation["proposal_set_id"] == mock_calculate_response["proposal_set_id"]
            assert calculation["snapshot_id"] == mock_calculate_response["snapshot_id"]
            assert calculation["execution_time_ms"] > 0
            
            # Verify validation phase
            validation = result["validation"]
            assert validation["validation_id"] == mock_validation_response["validation_id"]
            assert validation["verdict"] == "SAFE"
            
            # Verify commitment phase
            commitment = result["commitment"]
            assert commitment["order_id"] == mock_commit_response["medication_order_id"]
            assert commitment["audit_trail_id"] == mock_commit_response["audit_trail_id"]
            
            # Verify performance
            performance = result["performance"]
            assert performance["total_time_ms"] > 0
            assert performance["total_time_ms"] < 5000  # Should be reasonable
            
            # Verify service calls were made
            assert mock_client_instance.post.call_count == 3  # Calculate + Validate + Commit
    
    @pytest.mark.asyncio
    async def test_workflow_with_validation_warning(
        self,
        orchestrator: StrategicOrchestrator,
        sample_calculate_request: CalculateRequest
    ):
        """Test workflow when validation returns WARNING verdict."""
        # ARRANGE - Create warning scenario
        mock_calculate_response = {
            "proposal_set_id": f"propset_{uuid.uuid4().hex[:12]}",
            "snapshot_id": f"snap_{uuid.uuid4().hex[:12]}",
            "ranked_proposals": [{
                "proposal_id": f"prop_{uuid.uuid4().hex[:12]}",
                "medication_code": "213269",
                "medication_name": "Lisinopril 10mg",
                "confidence_score": 0.82
            }],
            "clinical_evidence": {},
            "monitoring_plan": {},
            "kb_versions": {}
        }
        
        mock_validation_response = {
            "validation_id": f"val_{uuid.uuid4().hex[:12]}",
            "verdict": "WARNING",
            "findings": [
                {
                    "finding_id": "warn_001",
                    "severity": "MEDIUM",
                    "category": "DRUG_INTERACTION",
                    "description": "Potential interaction with ACE inhibitor sensitivity",
                    "clinical_significance": "Monitor potassium levels closely",
                    "recommendation": "Check potassium within 1 week",
                    "confidence_score": 0.75,
                    "engine_source": "cae_engine"
                }
            ],
            "override_tokens": ["override_12345"],
            "override_requirements": {
                "required_level": "ATTENDING",
                "justification_required": True
            },
            "processing_time_ms": 95,
            "engine_results": [{
                "engine_id": "cae_engine",
                "status": "SUCCESS",
                "risk_score": 0.4,
                "warnings": ["Potassium monitoring recommended"],
                "confidence": 0.85
            }],
            "risk_score": 0.4
        }
        
        with patch('httpx.AsyncClient') as mock_http_client:
            mock_client_instance = Mock()
            mock_http_client.return_value = mock_client_instance
            
            # Mock responses
            mock_flow2_response = Mock()
            mock_flow2_response.status_code = 200
            mock_flow2_response.json.return_value = mock_calculate_response
            
            mock_safety_response = Mock()
            mock_safety_response.status_code = 200
            mock_safety_response.json.return_value = mock_validation_response
            
            def mock_post_side_effect(url, **kwargs):
                if "flow2-go" in url or ":8080" in url:
                    return asyncio.create_future().set_result(mock_flow2_response)
                elif "safety-gateway" in url or ":8018" in url:
                    return asyncio.create_future().set_result(mock_safety_response)
                else:
                    raise ValueError(f"Unexpected URL: {url}")
            
            mock_client_instance.post = AsyncMock(side_effect=mock_post_side_effect)
            
            # ACT
            result = await orchestrator.orchestrate_medication_request(sample_calculate_request)
            
            # ASSERT
            assert result["status"] == "REQUIRES_PROVIDER_DECISION"
            assert "validation_findings" in result
            assert "override_tokens" in result
            assert "proposals" in result
            assert "snapshot_id" in result
            
            # Verify warning details
            findings = result["validation_findings"]
            assert len(findings) == 1
            assert findings[0]["severity"] == "MEDIUM"
            assert findings[0]["category"] == "DRUG_INTERACTION"
            
            # Verify override tokens are provided
            assert result["override_tokens"] == ["override_12345"]
            
            # Verify no commit was attempted
            assert mock_client_instance.post.call_count == 2  # Only Calculate + Validate
    
    @pytest.mark.asyncio
    async def test_workflow_with_unsafe_validation(
        self,
        orchestrator: StrategicOrchestrator,
        sample_calculate_request: CalculateRequest
    ):
        """Test workflow when validation returns UNSAFE verdict."""
        # ARRANGE
        mock_calculate_response = {
            "proposal_set_id": f"propset_{uuid.uuid4().hex[:12]}",
            "snapshot_id": f"snap_{uuid.uuid4().hex[:12]}",
            "ranked_proposals": [{
                "proposal_id": f"prop_{uuid.uuid4().hex[:12]}",
                "medication_code": "213269",
                "medication_name": "Lisinopril 10mg"
            }],
            "clinical_evidence": {},
            "monitoring_plan": {},
            "kb_versions": {}
        }
        
        mock_validation_response = {
            "validation_id": f"val_{uuid.uuid4().hex[:12]}",
            "verdict": "UNSAFE",
            "findings": [
                {
                    "finding_id": "unsafe_001",
                    "severity": "HIGH",
                    "category": "CONTRAINDICATION",
                    "description": "Absolute contraindication: ACE inhibitor allergy",
                    "clinical_significance": "Life-threatening allergic reaction possible",
                    "recommendation": "DO NOT PRESCRIBE - Consider ARB alternative",
                    "confidence_score": 0.95,
                    "engine_source": "allergy_engine"
                }
            ],
            "override_tokens": None,
            "processing_time_ms": 120,
            "engine_results": [{
                "engine_id": "allergy_engine",
                "status": "SUCCESS",
                "risk_score": 0.9,
                "violations": ["ACE inhibitor allergy detected"],
                "confidence": 0.95
            }],
            "risk_score": 0.9
        }
        
        with patch('httpx.AsyncClient') as mock_http_client:
            mock_client_instance = Mock()
            mock_http_client.return_value = mock_client_instance
            
            # Mock responses
            mock_flow2_response = Mock()
            mock_flow2_response.status_code = 200
            mock_flow2_response.json.return_value = mock_calculate_response
            
            mock_safety_response = Mock()
            mock_safety_response.status_code = 200
            mock_safety_response.json.return_value = mock_validation_response
            
            # Mock alternative generation
            mock_alternatives_response = Mock()
            mock_alternatives_response.status_code = 200
            mock_alternatives_response.json.return_value = {
                "alternatives": [
                    {
                        "medication_code": "83515",
                        "medication_name": "Losartan 50mg",
                        "rationale": "ARB alternative for ACE inhibitor intolerant patients"
                    }
                ]
            }
            
            def mock_post_side_effect(url, **kwargs):
                if "flow2-go" in url and "execute" in url:
                    return asyncio.create_future().set_result(mock_flow2_response)
                elif "flow2-go" in url and "alternatives" in url:
                    return asyncio.create_future().set_result(mock_alternatives_response)
                elif "safety-gateway" in url:
                    return asyncio.create_future().set_result(mock_safety_response)
                else:
                    raise ValueError(f"Unexpected URL: {url}")
            
            mock_client_instance.post = AsyncMock(side_effect=mock_post_side_effect)
            
            # ACT
            result = await orchestrator.orchestrate_medication_request(sample_calculate_request)
            
            # ASSERT
            assert result["status"] == "BLOCKED_UNSAFE"
            assert "blocking_findings" in result
            assert "alternative_approaches" in result
            
            # Verify blocking details
            blocking_findings = result["blocking_findings"]
            assert len(blocking_findings) == 1
            assert blocking_findings[0]["severity"] == "HIGH"
            assert blocking_findings[0]["category"] == "CONTRAINDICATION"
            
            # Verify alternatives are provided
            alternatives = result["alternative_approaches"]
            assert len(alternatives) == 1
            assert alternatives[0]["medication_name"] == "Losartan 50mg"
    
    @pytest.mark.asyncio 
    async def test_workflow_calculate_phase_failure(
        self,
        orchestrator: StrategicOrchestrator,
        sample_calculate_request: CalculateRequest
    ):
        """Test workflow when Calculate phase fails."""
        with patch('httpx.AsyncClient') as mock_http_client:
            mock_client_instance = Mock()
            mock_http_client.return_value = mock_client_instance
            
            # Mock Flow2 Go service failure
            mock_flow2_response = Mock()
            mock_flow2_response.status_code = 500
            mock_flow2_response.text = "Internal server error"
            
            mock_client_instance.post = AsyncMock(return_value=mock_flow2_response)
            
            # ACT
            result = await orchestrator.orchestrate_medication_request(sample_calculate_request)
            
            # ASSERT
            assert result["status"] == "ERROR"
            assert result["error_code"] == "CALCULATE_FAILED"
            assert "error_message" in result
            assert result["correlation_id"] == sample_calculate_request.correlation_id
    
    @pytest.mark.asyncio
    async def test_safety_gateway_client_comprehensive_validation(self):
        """Test SafetyGatewayHTTPClient comprehensive validation."""
        # ARRANGE
        client = SafetyGatewayHTTPClient(base_url="http://localhost:8018")
        
        validation_request = SafetyValidationRequest(
            proposal_set_id=f"propset_{uuid.uuid4().hex[:12]}",
            snapshot_id=f"snap_{uuid.uuid4().hex[:12]}",
            proposals=[{
                "proposal_id": f"prop_{uuid.uuid4().hex[:12]}",
                "medication_code": "213269",
                "medication_name": "Lisinopril 10mg"
            }],
            patient_context={
                "patient_id": "patient_12345",
                "age": 55,
                "allergies": []
            },
            validation_requirements={
                "cae_engine": True,
                "protocol_engine": True,
                "comprehensive_validation": True
            },
            correlation_id=f"test_{uuid.uuid4().hex[:8]}"
        )
        
        mock_response_data = {
            "validation_id": f"val_{uuid.uuid4().hex[:12]}",
            "verdict": "SAFE",
            "findings": [],
            "processing_time_ms": 95,
            "engine_results": [{
                "engine_id": "cae_engine",
                "engine_name": "Clinical Assertion Engine",
                "status": "SUCCESS",
                "risk_score": 0.1,
                "violations": [],
                "warnings": [],
                "confidence": 0.98,
                "duration_ms": 75,
                "tier": 1
            }],
            "risk_score": 0.1,
            "timestamp": datetime.now(timezone.utc).isoformat()
        }
        
        with patch.object(client.client, 'post') as mock_post:
            mock_response = Mock()
            mock_response.status_code = 200
            mock_response.json.return_value = mock_response_data
            mock_post.return_value = mock_response
            
            # ACT
            result = await client.comprehensive_validation(validation_request)
            
            # ASSERT
            assert result.verdict == ValidationVerdict.SAFE
            assert result.validation_id == mock_response_data["validation_id"]
            assert result.risk_score == 0.1
            assert len(result.engine_results) == 1
            assert result.engine_results[0].engine_id == "cae_engine"
            assert result.processing_time_ms == 95
    
    @pytest.mark.asyncio
    async def test_performance_requirements(
        self,
        orchestrator: StrategicOrchestrator,
        sample_calculate_request: CalculateRequest
    ):
        """Test that workflow meets performance requirements."""
        # ARRANGE - Fast mock responses
        mock_responses = {
            "calculate": {"execution_time": 50},  # Fast calculate
            "validate": {"execution_time": 30},   # Fast validate  
            "commit": {"execution_time": 20}      # Fast commit
        }
        
        with patch('httpx.AsyncClient') as mock_http_client:
            mock_client_instance = Mock()
            mock_http_client.return_value = mock_client_instance
            
            # Configure fast responses
            def create_fast_response(phase):
                mock_response = Mock()
                mock_response.status_code = 200
                if phase == "calculate":
                    mock_response.json.return_value = {
                        "proposal_set_id": "fast_test",
                        "snapshot_id": "snap_fast",
                        "ranked_proposals": [{"proposal_id": "prop_fast"}],
                        "clinical_evidence": {},
                        "monitoring_plan": {},
                        "kb_versions": {}
                    }
                elif phase == "validate":
                    mock_response.json.return_value = {
                        "validation_id": "val_fast",
                        "verdict": "SAFE",
                        "findings": [],
                        "engine_results": [],
                        "risk_score": 0.1,
                        "processing_time_ms": 30
                    }
                elif phase == "commit":
                    mock_response.json.return_value = {
                        "medication_order_id": "order_fast",
                        "persistence_status": "SUCCESS",
                        "event_publication_status": "SUCCESS",
                        "audit_trail_id": "audit_fast",
                        "result": "SUCCESS"
                    }
                return mock_response
            
            def mock_post_side_effect(url, **kwargs):
                if "flow2-go" in url or ":8080" in url:
                    return asyncio.create_future().set_result(create_fast_response("calculate"))
                elif "safety-gateway" in url or ":8018" in url:
                    return asyncio.create_future().set_result(create_fast_response("validate"))
                elif "medication-service" in url or ":8004" in url:
                    return asyncio.create_future().set_result(create_fast_response("commit"))
            
            mock_client_instance.post = AsyncMock(side_effect=mock_post_side_effect)
            
            # ACT
            start_time = datetime.now(timezone.utc)
            result = await orchestrator.orchestrate_medication_request(sample_calculate_request)
            end_time = datetime.now(timezone.utc)
            
            # ASSERT
            total_time_ms = (end_time - start_time).total_seconds() * 1000
            
            # Should meet performance targets (from strategic_orchestrator.py:134-139)
            assert result["status"] == "SUCCESS"
            assert total_time_ms < 1000  # Should be well under 325ms target for mocked services
            assert result["performance"]["total_time_ms"] < 1000
            
            # Individual phase performance (mocked, so very fast)
            performance = result["performance"]
            assert performance["meets_target"] == True


if __name__ == "__main__":
    # Run tests with pytest
    pytest.main([__file__, "-v", "--tb=short"])