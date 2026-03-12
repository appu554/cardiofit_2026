#!/usr/bin/env python3
"""
Test script for Kafka schemas and event templates
"""

import sys
import logging
import json
from pathlib import Path

# Setup logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

def test_avro_schemas():
    """Test Avro schema definitions"""
    logger.info("Testing Avro schemas...")
    
    try:
        from .avro_schemas import get_all_schemas, get_avro_schema
        
        # Test getting all schemas
        schemas = get_all_schemas()
        assert len(schemas) > 0, "No schemas found"
        
        expected_schemas = [
            'event-envelope',
            'fhir-resource-event', 
            'device-data-event',
            'clinical-alert-event',
            'workflow-event'
        ]
        
        for schema_name in expected_schemas:
            schema = get_avro_schema(schema_name)
            assert schema is not None, f"Schema {schema_name} not found"
            assert 'type' in schema, f"Schema {schema_name} missing type"
            assert 'name' in schema, f"Schema {schema_name} missing name"
            assert 'fields' in schema, f"Schema {schema_name} missing fields"
            
            logger.info(f"✓ Schema {schema_name} is valid")
        
        logger.info("✓ All Avro schemas are valid")
        return True
        
    except Exception as e:
        logger.error(f"✗ Avro schema test failed: {e}")
        return False

def test_schema_evolution():
    """Test schema evolution manager"""
    logger.info("Testing schema evolution...")
    
    try:
        from .schema_evolution import get_schema_manager, CompatibilityLevel
        
        manager = get_schema_manager()
        
        # Test listing schemas
        schemas = manager.list_schemas()
        assert len(schemas) > 0, "No schemas in manager"
        
        # Test getting schema
        schema = manager.get_schema('fhir-resource-event')
        assert schema is not None, "FHIR resource event schema not found"
        
        # Test schema versions
        versions = manager.list_versions('fhir-resource-event')
        assert len(versions) > 0, "No versions found for FHIR resource event schema"
        
        # Test registering new version
        test_schema = {
            "type": "record",
            "name": "TestEvent",
            "fields": [
                {"name": "id", "type": "string"},
                {"name": "message", "type": "string"}
            ]
        }
        
        success = manager.register_schema_version(
            schema_name="test-event",
            version="1.0.0",
            schema=test_schema,
            compatibility_level=CompatibilityLevel.BACKWARD,
            description="Test schema"
        )
        assert success, "Failed to register test schema"
        
        # Test validation
        test_data = {"id": "test-123", "message": "Hello World"}
        is_valid, error = manager.validate_data("test-event", test_data)
        assert is_valid, f"Test data validation failed: {error}"
        
        logger.info("✓ Schema evolution manager working correctly")
        return True
        
    except Exception as e:
        logger.error(f"✗ Schema evolution test failed: {e}")
        return False

def test_event_templates():
    """Test event template builder"""
    logger.info("Testing event templates...")
    
    try:
        from .event_templates import EventTemplateBuilder, validate_event
        from .event_templates import create_patient_created_event, create_observation_recorded_event
        
        builder = EventTemplateBuilder("test-service")
        
        # Test FHIR event creation
        patient_data = {
            "resourceType": "Patient",
            "id": "patient-123",
            "name": [{"family": "Doe", "given": ["John"]}],
            "meta": {
                "versionId": "1",
                "lastUpdated": "2024-01-01T00:00:00Z"
            }
        }
        
        fhir_event = builder.create_fhir_event(
            resource_type="Patient",
            resource_id="patient-123",
            operation="created",
            resource_data=patient_data
        )
        
        assert fhir_event.type == "patient.created"
        assert fhir_event.subject == "Patient/patient-123"
        assert fhir_event.source == "test-service"
        assert fhir_event.data['resource_type'] == "Patient"
        
        # Test device data event
        measurements = [
            {"type": "blood_pressure", "systolic": 120, "diastolic": 80},
            {"type": "heart_rate", "value": 72}
        ]
        
        device_event = builder.create_device_data_event(
            device_id="device-456",
            device_type="blood_pressure_monitor",
            measurements=measurements,
            patient_id="patient-123"
        )
        
        assert device_event.subject == "Device/device-456"
        assert device_event.data['device_type'] == "blood_pressure_monitor"
        assert device_event.data['patient_id'] == "patient-123"
        
        # Test clinical alert event
        alert_event = builder.create_clinical_alert_event(
            alert_type="critical_value",
            severity="critical",
            patient_id="patient-123",
            message="Critical blood pressure reading",
            triggered_by="Observation/obs-789"
        )
        
        assert alert_event.type == "alert.critical_value"
        assert alert_event.data['severity'] == "critical"
        assert alert_event.data['patient_id'] == "patient-123"
        
        # Test workflow event
        workflow_event = builder.create_workflow_event(
            workflow_id="workflow-999",
            workflow_type="patient_admission",
            event_type="started",
            patient_id="patient-123",
            user_id="user-456"
        )
        
        assert workflow_event.type == "workflow.started"
        assert workflow_event.data['workflow_type'] == "patient_admission"
        assert workflow_event.data['patient_id'] == "patient-123"
        
        # Test utility functions
        patient_event = create_patient_created_event(
            source="patient-service",
            patient_id="patient-999",
            patient_data=patient_data
        )
        
        assert patient_event.type == "patient.created"
        assert patient_event.source == "patient-service"
        
        observation_data = {
            "resourceType": "Observation",
            "id": "obs-123",
            "status": "final",
            "code": {"coding": [{"code": "8480-6", "display": "Systolic blood pressure"}]},
            "valueQuantity": {"value": 140, "unit": "mmHg"}
        }
        
        obs_event = create_observation_recorded_event(
            source="observation-service",
            observation_id="obs-123",
            observation_data=observation_data,
            patient_id="patient-123"
        )
        
        assert obs_event.type == "observation.created"
        assert obs_event.data['resource_type'] == "Observation"
        
        # Test event validation
        is_valid, error = validate_event(fhir_event)
        if not is_valid:
            logger.warning(f"Event validation failed: {error}")
        
        logger.info("✓ Event templates working correctly")
        return True
        
    except Exception as e:
        logger.error(f"✗ Event template test failed: {e}")
        return False

def test_schema_registry():
    """Test schema registry"""
    logger.info("Testing schema registry...")
    
    try:
        from .schemas import schema_registry
        
        # Test listing schemas
        schemas = schema_registry.list_schemas()
        assert len(schemas) > 0, "No schemas in registry"
        
        # Test getting schema
        fhir_schema = schema_registry.get_schema('fhir-resource-event')
        assert fhir_schema is not None, "FHIR schema not found"
        
        # Test event validation
        test_data = {
            'resource_type': 'Patient',
            'resource_id': 'patient-123',
            'operation': 'created'
        }
        
        is_valid = schema_registry.validate_event('fhir-resource-event', test_data)
        assert is_valid, "Event validation failed"
        
        logger.info("✓ Schema registry working correctly")
        return True
        
    except Exception as e:
        logger.error(f"✗ Schema registry test failed: {e}")
        return False

def test_integration():
    """Test integration between components"""
    logger.info("Testing component integration...")
    
    try:
        from .event_templates import EventTemplateBuilder
        from .producer import EventProducer
        from .config import TopicNames
        
        # Create event
        builder = EventTemplateBuilder("integration-test")
        
        patient_data = {
            "resourceType": "Patient",
            "id": "integration-patient-123",
            "name": [{"family": "Integration", "given": ["Test"]}]
        }
        
        event = builder.create_fhir_event(
            resource_type="Patient",
            resource_id="integration-patient-123",
            operation="created",
            resource_data=patient_data
        )
        
        # Test that event can be serialized
        event_json = event.to_json()
        assert len(event_json) > 0, "Event serialization failed"
        
        # Test that event can be deserialized
        from .schemas import EventEnvelope
        restored_event = EventEnvelope.from_json(event_json)
        assert restored_event.id == event.id, "Event deserialization failed"
        
        logger.info("✓ Component integration working correctly")
        return True
        
    except Exception as e:
        logger.error(f"✗ Integration test failed: {e}")
        return False

def main():
    """Run all schema tests"""
    logger.info("🧪 Testing Kafka schemas and templates...")
    logger.info("=" * 60)
    
    tests = [
        ("Avro Schemas", test_avro_schemas),
        ("Schema Evolution", test_schema_evolution),
        ("Event Templates", test_event_templates),
        ("Schema Registry", test_schema_registry),
        ("Integration", test_integration),
    ]
    
    passed = 0
    total = len(tests)
    
    for test_name, test_func in tests:
        logger.info(f"\n📋 Running {test_name}...")
        try:
            if test_func():
                passed += 1
                logger.info(f"✅ {test_name} PASSED")
            else:
                logger.error(f"❌ {test_name} FAILED")
        except Exception as e:
            logger.error(f"❌ {test_name} FAILED with exception: {e}")
    
    logger.info(f"\n📊 Test Results: {passed}/{total} tests passed")
    
    if passed == total:
        logger.info("🎉 All schema tests passed!")
        return True
    else:
        logger.error("❌ Some schema tests failed.")
        return False

if __name__ == "__main__":
    success = main()
    sys.exit(0 if success else 1)
