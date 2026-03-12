"""
Flow 2 Integration Tests

Comprehensive test suite for Flow 2 medication safety integration,
testing the complete workflow from API request to clinical validation.

Test Coverage:
- Recipe Orchestrator functionality
- Context Service integration
- Clinical Recipe execution with real context
- API endpoint validation
- Error handling and graceful degradation
- Performance requirements
"""

import pytest
import asyncio
import json
from datetime import datetime
from unittest.mock import Mock, patch, AsyncMock
from fastapi.testclient import TestClient

# Import Flow 2 components
from app.domain.services.recipe_orchestrator import (
    RecipeOrchestrator, 
    MedicationSafetyRequest, 
    Flow2Result
)
from app.domain.services.context_data_adapter import ContextDataAdapter
from app.infrastructure.context_service_client import ClinicalContext
from app.api.endpoints.flow2_medication_safety import router
from app.main import app

# Test client
client = TestClient(app)


class TestRecipeOrchestrator:
    """Test the Recipe Orchestrator core functionality"""
    
    @pytest.fixture
    def orchestrator(self):
        """Create Recipe Orchestrator instance for testing"""
        return RecipeOrchestrator()
    
    @pytest.fixture
    def sample_medication_request(self):
        """Sample medication safety request"""
        return MedicationSafetyRequest(
            patient_id="905a60cb-8241-418f-b29b-5b020e851392",
            medication={
                "name": "warfarin",
                "dose": "5mg",
                "frequency": "daily",
                "is_anticoagulant": True
            },
            provider_id="provider_123",
            action_type="prescribe",
            urgency="routine"
        )
    
    def test_context_recipe_determination(self, orchestrator, sample_medication_request):
        """Test context recipe selection logic"""
        # Test anticoagulant medication
        recipe_id = orchestrator._determine_context_recipe(sample_medication_request)
        assert recipe_id == "medication_safety_base_context_v2"
        
        # Test chemotherapy medication
        chemo_request = MedicationSafetyRequest(
            patient_id="test_patient",
            medication={"name": "doxorubicin", "is_chemotherapy": True},
            action_type="prescribe"
        )
        recipe_id = orchestrator._determine_context_recipe(chemo_request)
        assert recipe_id == "medication_safety_base_context_v2"
        
        # Test renal adjustment medication
        renal_request = MedicationSafetyRequest(
            patient_id="test_patient",
            medication={"name": "metformin", "requires_renal_adjustment": True},
            action_type="prescribe"
        )
        recipe_id = orchestrator._determine_context_recipe(renal_request)
        assert recipe_id == "medication_renal_context_v2"
    
    @pytest.mark.asyncio
    async def test_minimal_context_fallback(self, orchestrator, sample_medication_request):
        """Test graceful degradation when Context Service is unavailable"""
        minimal_context = await orchestrator._get_minimal_context_fallback(sample_medication_request)
        
        assert minimal_context.patient_id == sample_medication_request.patient_id
        assert minimal_context.recipe_used == "minimal_fallback"
        assert minimal_context.completeness_score == 0.3
        assert len(minimal_context.safety_flags) > 0
        assert minimal_context.safety_flags[0]["flagType"] == "CONTEXT_UNAVAILABLE"
    
    @pytest.mark.asyncio
    async def test_health_check(self, orchestrator):
        """Test Recipe Orchestrator health check"""
        health_status = await orchestrator.health_check()
        
        assert "recipe_orchestrator" in health_status
        assert "context_service" in health_status
        assert "clinical_recipe_engine" in health_status
        assert "registered_recipes" in health_status
        assert health_status["recipe_orchestrator"] == "healthy"


class TestContextDataAdapter:
    """Test the Context Data Adapter functionality"""
    
    @pytest.fixture
    def adapter(self):
        """Create Context Data Adapter instance"""
        return ContextDataAdapter()
    
    @pytest.fixture
    def sample_context_data(self):
        """Sample context data from Context Service"""
        return ClinicalContext(
            context_id="test_context_123",
            patient_id="905a60cb-8241-418f-b29b-5b020e851392",
            recipe_used="medication_safety_base_context_v2",
            assembled_data={
                "patient": {
                    "age": 65,
                    "weight": 75,
                    "gender": "male",
                    "conditions": ["hypertension", "diabetes"],
                    "allergies": [{"allergen": "penicillin", "reaction": "rash"}]
                },
                "labs": {
                    "creatinine": 1.2,
                    "alt": 30,
                    "inr": 2.1
                },
                "medications": {
                    "current": [
                        {
                            "name": "lisinopril",
                            "dose": "10mg",
                            "therapeutic_class": "ACE_INHIBITOR"
                        }
                    ]
                },
                "vitals": {
                    "heart_rate": 72,
                    "blood_pressure": {"systolic": 140, "diastolic": 90}
                }
            },
            completeness_score=0.85,
            data_freshness={},
            source_metadata={},
            safety_flags=[],
            governance_tags=[],
            status="COMPLETE",
            assembled_at=datetime.now(),
            assembly_duration_ms=85.0,
            connection_errors=[]
        )
    
    def test_patient_data_transformation(self, adapter, sample_context_data):
        """Test patient data transformation"""
        patient_data = adapter._transform_patient_data(
            sample_context_data.assembled_data["patient"]
        )
        
        assert patient_data["age"] == 65
        assert patient_data["weight_kg"] == 75
        assert patient_data["gender"] == "male"
        assert "hypertension" in patient_data["conditions"]
        assert "diabetes" in patient_data["conditions"]
        assert len(patient_data["allergies"]) == 1
        assert patient_data["allergies"][0]["allergen"] == "penicillin"
    
    def test_lab_data_transformation(self, adapter, sample_context_data):
        """Test laboratory data transformation"""
        lab_data = adapter._transform_lab_data(
            sample_context_data.assembled_data["labs"]
        )
        
        assert lab_data["creatinine"] == 1.2
        assert lab_data["alt"] == 30
        assert lab_data["inr"] == 2.1
        # Should have default values for missing labs
        assert lab_data["ast"] == 25  # Default value
    
    def test_medication_list_transformation(self, adapter):
        """Test medication list transformation"""
        medications = [
            {
                "name": "lisinopril",
                "dose": "10mg",
                "therapeutic_class": "ACE_INHIBITOR"
            }
        ]
        
        transformed = adapter._transform_medication_list(medications)
        
        assert len(transformed) == 1
        assert transformed[0]["name"] == "lisinopril"
        assert transformed[0]["dose"] == "10mg"
        assert transformed[0]["therapeutic_class"] == "ACE_INHIBITOR"
        assert transformed[0]["nephrotoxic_risk"] == "NONE"  # Default
    
    def test_complete_context_transformation(self, adapter, sample_context_data):
        """Test complete context transformation"""
        medication_data = {"name": "warfarin", "is_anticoagulant": True}
        
        recipe_context = adapter.transform_context_for_recipes(
            context_data=sample_context_data,
            medication_data=medication_data,
            action_type="prescribe"
        )
        
        assert recipe_context.patient_id == sample_context_data.patient_id
        assert recipe_context.medication_data == medication_data
        assert recipe_context.action_type == "prescribe"
        assert recipe_context.patient_data["age"] == 65
        assert "labs" in recipe_context.clinical_data
        assert "current_medications" in recipe_context.clinical_data
        assert "context_metadata" in recipe_context.clinical_data
    
    def test_data_validation(self, adapter, sample_context_data):
        """Test transformed data validation"""
        medication_data = {"name": "warfarin"}
        
        recipe_context = adapter.transform_context_for_recipes(
            context_data=sample_context_data,
            medication_data=medication_data
        )
        
        validation_results = adapter.validate_transformed_data(recipe_context)
        
        assert "data_completeness" in validation_results
        assert "data_quality_score" in validation_results
        assert validation_results["data_quality_score"] > 0.5  # Should be reasonable quality


class TestFlow2APIEndpoints:
    """Test Flow 2 API endpoints"""
    
    def test_validate_medication_safety_endpoint(self):
        """Test the main Flow 2 validation endpoint"""
        request_data = {
            "patient_id": "905a60cb-8241-418f-b29b-5b020e851392",
            "medication": {
                "name": "warfarin",
                "dose": "5mg",
                "frequency": "daily",
                "is_anticoagulant": True
            },
            "provider_id": "provider_123",
            "action_type": "prescribe",
            "urgency": "routine"
        }
        
        # Mock the orchestrator to avoid external dependencies
        with patch('app.api.endpoints.flow2_medication_safety.get_recipe_orchestrator') as mock_orchestrator:
            mock_result = Flow2Result(
                request_id="test_request_123",
                patient_id=request_data["patient_id"],
                overall_safety_status="SAFE",
                context_recipe_used="medication_safety_base_context_v2",
                clinical_recipes_executed=["medication-safety-anticoagulation-v3.0"],
                context_completeness_score=0.85,
                execution_time_ms=150.0,
                clinical_results=[],
                safety_summary={"total_validations": 5, "critical_issues": 0},
                performance_metrics={"context_assembly_time_ms": 85.0}
            )
            
            mock_orchestrator.return_value.execute_medication_safety = AsyncMock(return_value=mock_result)
            
            response = client.post("/api/flow2/medication-safety/validate", json=request_data)
            
            assert response.status_code == 200
            data = response.json()
            assert data["overall_safety_status"] == "SAFE"
            assert data["patient_id"] == request_data["patient_id"]
            assert data["context_completeness_score"] == 0.85
            assert data["execution_time_ms"] == 150.0
    
    def test_health_check_endpoint(self):
        """Test the health check endpoint"""
        with patch('app.api.endpoints.flow2_medication_safety.get_recipe_orchestrator') as mock_orchestrator:
            mock_health = {
                "recipe_orchestrator": "healthy",
                "context_service": "healthy",
                "clinical_recipe_engine": "healthy",
                "registered_recipes": 29
            }
            mock_metrics = {
                "total_requests": 100,
                "success_rate": 0.95,
                "average_response_time_ms": 120.0
            }
            
            mock_orchestrator.return_value.health_check = AsyncMock(return_value=mock_health)
            mock_orchestrator.return_value.context_service_client.get_flow2_performance_metrics = Mock(return_value=mock_metrics)
            
            response = client.get("/api/flow2/medication-safety/health")
            
            assert response.status_code == 200
            data = response.json()
            assert data["status"] == "healthy"
            assert "components" in data
            assert "performance_metrics" in data
    
    def test_metrics_endpoint(self):
        """Test the metrics endpoint"""
        with patch('app.api.endpoints.flow2_medication_safety.get_recipe_orchestrator') as mock_orchestrator:
            mock_metrics = {
                "total_requests": 100,
                "success_rate": 0.95,
                "average_response_time_ms": 120.0
            }
            mock_catalog = {
                "medication-safety-anticoagulation-v3.0": {
                    "name": "Anticoagulation Safety",
                    "priority": 98
                }
            }
            
            mock_orchestrator.return_value.context_service_client.get_flow2_performance_metrics = Mock(return_value=mock_metrics)
            mock_orchestrator.return_value.clinical_recipe_engine.get_recipe_catalog = Mock(return_value=mock_catalog)
            
            response = client.get("/api/flow2/medication-safety/metrics")
            
            assert response.status_code == 200
            data = response.json()
            assert "flow2_metrics" in data
            assert "clinical_recipes" in data["flow2_metrics"]
            assert data["flow2_metrics"]["total_requests"] == 100


class TestFlow2Performance:
    """Test Flow 2 performance requirements"""
    
    @pytest.mark.asyncio
    async def test_response_time_requirements(self):
        """Test that Flow 2 meets response time requirements"""
        orchestrator = RecipeOrchestrator()
        
        request = MedicationSafetyRequest(
            patient_id="test_patient",
            medication={"name": "aspirin"},
            action_type="prescribe"
        )
        
        # Mock context service to return quickly
        with patch.object(orchestrator.context_service_client, 'get_medication_safety_context') as mock_context:
            mock_context.return_value = AsyncMock()
            mock_context.return_value.context_id = "test_context"
            mock_context.return_value.completeness_score = 0.8
            mock_context.return_value.assembled_data = {"patient": {}, "labs": {}}
            mock_context.return_value.safety_flags = []
            mock_context.return_value.assembly_duration_ms = 50.0
            
            start_time = datetime.now()
            result = await orchestrator.execute_medication_safety(request)
            end_time = datetime.now()
            
            execution_time = (end_time - start_time).total_seconds() * 1000
            
            # Flow 2 should complete in under 200ms (target from plan)
            assert execution_time < 200, f"Flow 2 took {execution_time:.1f}ms, exceeds 200ms target"
            assert result.overall_safety_status in ["SAFE", "WARNING", "UNSAFE", "ERROR"]


if __name__ == "__main__":
    # Run the tests
    pytest.main([__file__, "-v"])
