"""
Avro schemas for Clinical Synthesis Hub events
"""

import json
from typing import Dict, Any

# Base event envelope schema (CloudEvents compatible)
EVENT_ENVELOPE_SCHEMA = {
    "type": "record",
    "name": "EventEnvelope",
    "namespace": "com.clinicalsynthesishub.events",
    "doc": "Standard event envelope for all Clinical Synthesis Hub events",
    "fields": [
        {
            "name": "id",
            "type": "string",
            "doc": "Unique event identifier (UUID)"
        },
        {
            "name": "source",
            "type": "string", 
            "doc": "Event source (service name)"
        },
        {
            "name": "type",
            "type": "string",
            "doc": "Event type (e.g., patient.created, observation.updated)"
        },
        {
            "name": "subject",
            "type": "string",
            "doc": "Event subject (resource identifier)"
        },
        {
            "name": "time",
            "type": "string",
            "doc": "Event timestamp (ISO 8601)"
        },
        {
            "name": "data",
            "type": "string",
            "doc": "Event payload (JSON string)"
        },
        {
            "name": "correlation_id",
            "type": ["null", "string"],
            "default": None,
            "doc": "Correlation ID for tracing"
        },
        {
            "name": "causation_id", 
            "type": ["null", "string"],
            "default": None,
            "doc": "ID of the event that caused this event"
        },
        {
            "name": "metadata",
            "type": "string",
            "default": "{}",
            "doc": "Additional metadata (JSON string)"
        },
        {
            "name": "version",
            "type": "string",
            "default": "1.0",
            "doc": "Schema version"
        }
    ]
}

# FHIR Resource Event Schema
FHIR_RESOURCE_EVENT_SCHEMA = {
    "type": "record",
    "name": "FHIRResourceEvent",
    "namespace": "com.clinicalsynthesishub.events.fhir",
    "doc": "Event for FHIR resource operations",
    "fields": [
        {
            "name": "resource_type",
            "type": {
                "type": "enum",
                "name": "FHIRResourceType",
                "symbols": [
                    "Patient", "Encounter", "Observation", "Medication", 
                    "MedicationRequest", "ServiceRequest", "Condition",
                    "Organization", "Practitioner", "Device", "DiagnosticReport",
                    "ImagingStudy", "DocumentReference", "Appointment"
                ]
            },
            "doc": "FHIR resource type"
        },
        {
            "name": "resource_id",
            "type": "string",
            "doc": "Resource identifier"
        },
        {
            "name": "operation",
            "type": {
                "type": "enum",
                "name": "CRUDOperation",
                "symbols": ["created", "updated", "deleted", "read"]
            },
            "doc": "CRUD operation performed"
        },
        {
            "name": "resource",
            "type": ["null", "string"],
            "default": None,
            "doc": "Full resource data (JSON string, for created/updated)"
        },
        {
            "name": "previous_resource",
            "type": ["null", "string"],
            "default": None,
            "doc": "Previous version (JSON string, for updated)"
        },
        {
            "name": "version_id",
            "type": ["null", "string"],
            "default": None,
            "doc": "Resource version identifier"
        },
        {
            "name": "last_updated",
            "type": ["null", "string"],
            "default": None,
            "doc": "Last updated timestamp (ISO 8601)"
        },
        {
            "name": "profile",
            "type": {
                "type": "array",
                "items": "string"
            },
            "default": [],
            "doc": "FHIR profiles applied to this resource"
        },
        {
            "name": "patient_id",
            "type": ["null", "string"],
            "default": None,
            "doc": "Associated patient ID (for patient-related resources)"
        },
        {
            "name": "encounter_id",
            "type": ["null", "string"],
            "default": None,
            "doc": "Associated encounter ID (for encounter-related resources)"
        }
    ]
}

# Device Data Event Schema
DEVICE_DATA_EVENT_SCHEMA = {
    "type": "record",
    "name": "DeviceDataEvent",
    "namespace": "com.clinicalsynthesishub.events.device",
    "doc": "Event for device data ingestion",
    "fields": [
        {
            "name": "device_id",
            "type": "string",
            "doc": "Device identifier"
        },
        {
            "name": "device_type",
            "type": {
                "type": "enum",
                "name": "DeviceType",
                "symbols": [
                    "blood_pressure_monitor", "glucose_meter", "pulse_oximeter",
                    "ecg_monitor", "thermometer", "scale", "activity_tracker",
                    "continuous_glucose_monitor", "insulin_pump", "ventilator"
                ]
            },
            "doc": "Type of device"
        },
        {
            "name": "patient_id",
            "type": ["null", "string"],
            "default": None,
            "doc": "Associated patient identifier"
        },
        {
            "name": "measurements",
            "type": "string",
            "default": "[]",
            "doc": "Measurement data (JSON array)"
        },
        {
            "name": "raw_data",
            "type": ["null", "string"],
            "default": None,
            "doc": "Raw device data (JSON string)"
        },
        {
            "name": "manufacturer",
            "type": ["null", "string"],
            "default": None,
            "doc": "Device manufacturer"
        },
        {
            "name": "model",
            "type": ["null", "string"],
            "default": None,
            "doc": "Device model"
        },
        {
            "name": "firmware_version",
            "type": ["null", "string"],
            "default": None,
            "doc": "Device firmware version"
        },
        {
            "name": "timestamp",
            "type": "string",
            "doc": "Measurement timestamp (ISO 8601)"
        },
        {
            "name": "location",
            "type": ["null", "string"],
            "default": None,
            "doc": "Device location (JSON string)"
        }
    ]
}

# Clinical Alert Event Schema
CLINICAL_ALERT_EVENT_SCHEMA = {
    "type": "record",
    "name": "ClinicalAlertEvent",
    "namespace": "com.clinicalsynthesishub.events.clinical",
    "doc": "Event for clinical alerts and notifications",
    "fields": [
        {
            "name": "alert_id",
            "type": "string",
            "doc": "Unique alert identifier"
        },
        {
            "name": "alert_type",
            "type": {
                "type": "enum",
                "name": "AlertType",
                "symbols": [
                    "critical_value", "drug_interaction", "allergy_alert",
                    "duplicate_therapy", "contraindication", "lab_panic_value",
                    "vital_sign_alert", "medication_reminder", "appointment_reminder"
                ]
            },
            "doc": "Type of clinical alert"
        },
        {
            "name": "severity",
            "type": {
                "type": "enum",
                "name": "AlertSeverity",
                "symbols": ["low", "medium", "high", "critical"]
            },
            "doc": "Alert severity level"
        },
        {
            "name": "patient_id",
            "type": "string",
            "doc": "Patient identifier"
        },
        {
            "name": "message",
            "type": "string",
            "doc": "Alert message"
        },
        {
            "name": "triggered_by",
            "type": ["null", "string"],
            "default": None,
            "doc": "What triggered the alert (resource ID or event)"
        },
        {
            "name": "resource_references",
            "type": {
                "type": "array",
                "items": "string"
            },
            "default": [],
            "doc": "Related FHIR resource references"
        },
        {
            "name": "recommended_actions",
            "type": {
                "type": "array",
                "items": "string"
            },
            "default": [],
            "doc": "Recommended actions to take"
        },
        {
            "name": "expires_at",
            "type": ["null", "string"],
            "default": None,
            "doc": "Alert expiration timestamp (ISO 8601)"
        },
        {
            "name": "acknowledged_by",
            "type": ["null", "string"],
            "default": None,
            "doc": "User who acknowledged the alert"
        },
        {
            "name": "acknowledged_at",
            "type": ["null", "string"],
            "default": None,
            "doc": "Acknowledgment timestamp (ISO 8601)"
        }
    ]
}

# Workflow Event Schema
WORKFLOW_EVENT_SCHEMA = {
    "type": "record",
    "name": "WorkflowEvent",
    "namespace": "com.clinicalsynthesishub.events.workflow",
    "doc": "Event for workflow operations",
    "fields": [
        {
            "name": "workflow_id",
            "type": "string",
            "doc": "Workflow instance identifier"
        },
        {
            "name": "workflow_type",
            "type": {
                "type": "enum",
                "name": "WorkflowType",
                "symbols": [
                    "patient_admission", "patient_discharge", "medication_reconciliation",
                    "lab_order_workflow", "imaging_workflow", "clinical_decision_support",
                    "care_plan_execution", "quality_measure_reporting"
                ]
            },
            "doc": "Type of workflow"
        },
        {
            "name": "event_type",
            "type": {
                "type": "enum",
                "name": "WorkflowEventType",
                "symbols": [
                    "started", "completed", "failed", "cancelled", "paused", "resumed",
                    "task_assigned", "task_completed", "task_failed", "escalated"
                ]
            },
            "doc": "Workflow event type"
        },
        {
            "name": "patient_id",
            "type": ["null", "string"],
            "default": None,
            "doc": "Associated patient identifier"
        },
        {
            "name": "task_id",
            "type": ["null", "string"],
            "default": None,
            "doc": "Associated task identifier"
        },
        {
            "name": "user_id",
            "type": ["null", "string"],
            "default": None,
            "doc": "User performing the action"
        },
        {
            "name": "workflow_data",
            "type": ["null", "string"],
            "default": None,
            "doc": "Workflow variables (JSON string)"
        },
        {
            "name": "task_data",
            "type": ["null", "string"],
            "default": None,
            "doc": "Task-specific data (JSON string)"
        },
        {
            "name": "error_message",
            "type": ["null", "string"],
            "default": None,
            "doc": "Error message (for failed events)"
        },
        {
            "name": "duration_ms",
            "type": ["null", "long"],
            "default": None,
            "doc": "Event duration in milliseconds"
        }
    ]
}

# Schema registry mapping
AVRO_SCHEMAS = {
    "event-envelope": EVENT_ENVELOPE_SCHEMA,
    "fhir-resource-event": FHIR_RESOURCE_EVENT_SCHEMA,
    "device-data-event": DEVICE_DATA_EVENT_SCHEMA,
    "clinical-alert-event": CLINICAL_ALERT_EVENT_SCHEMA,
    "workflow-event": WORKFLOW_EVENT_SCHEMA
}

def get_avro_schema(schema_name: str) -> Dict[str, Any]:
    """Get Avro schema by name"""
    return AVRO_SCHEMAS.get(schema_name)

def get_all_schemas() -> Dict[str, Dict[str, Any]]:
    """Get all Avro schemas"""
    return AVRO_SCHEMAS.copy()

def validate_schema_compatibility(old_schema: Dict[str, Any], new_schema: Dict[str, Any]) -> bool:
    """
    Basic schema compatibility check
    In a full implementation, this would use Avro's schema resolution rules
    """
    # For now, just check that required fields haven't been removed
    old_fields = {field["name"] for field in old_schema.get("fields", [])}
    new_fields = {field["name"] for field in new_schema.get("fields", [])}
    
    # Check if any required fields were removed
    old_required = {
        field["name"] for field in old_schema.get("fields", [])
        if "default" not in field and field.get("type") != ["null", "string"]
    }
    
    new_required = {
        field["name"] for field in new_schema.get("fields", [])
        if "default" not in field and field.get("type") != ["null", "string"]
    }
    
    # Backward compatibility: old required fields should still be present
    removed_required = old_required - new_required
    if removed_required:
        return False
    
    return True
