"""
Avro schemas for business events from core services
"""

# Order Management Events Schema
ORDER_EVENT_SCHEMA = {
    "type": "record",
    "name": "OrderEvent",
    "namespace": "com.clinicalsynthesishub.events.order",
    "doc": "Event for order management operations",
    "fields": [
        {
            "name": "event_id",
            "type": "string",
            "doc": "Unique event identifier"
        },
        {
            "name": "event_type",
            "type": {
                "type": "enum",
                "name": "OrderEventType",
                "symbols": [
                    "order.created",
                    "order.updated", 
                    "order.signed",
                    "order.cancelled",
                    "order.completed",
                    "order.suspended"
                ]
            },
            "doc": "Type of order event"
        },
        {
            "name": "order_id",
            "type": "string",
            "doc": "Order identifier"
        },
        {
            "name": "patient_id",
            "type": "string",
            "doc": "Patient identifier"
        },
        {
            "name": "order_type",
            "type": "string",
            "doc": "Type of order (lab, medication, imaging, etc.)"
        },
        {
            "name": "operation",
            "type": "string",
            "doc": "Operation performed (created, updated, signed, etc.)"
        },
        {
            "name": "status",
            "type": ["null", "string"],
            "default": None,
            "doc": "Current order status"
        },
        {
            "name": "order_data",
            "type": "string",
            "doc": "Complete order data as JSON string"
        },
        {
            "name": "timestamp",
            "type": "long",
            "doc": "Event timestamp (Unix timestamp)"
        },
        {
            "name": "service",
            "type": "string",
            "doc": "Source service name"
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
            "doc": "Causation ID for event sourcing"
        },
        {
            "name": "metadata",
            "type": ["null", "string"],
            "default": None,
            "doc": "Additional metadata as JSON string"
        }
    ]
}

# Patient Events Schema
PATIENT_EVENT_SCHEMA = {
    "type": "record",
    "name": "PatientEvent",
    "namespace": "com.clinicalsynthesishub.events.patient",
    "doc": "Event for patient management operations",
    "fields": [
        {
            "name": "event_id",
            "type": "string",
            "doc": "Unique event identifier"
        },
        {
            "name": "event_type",
            "type": {
                "type": "enum",
                "name": "PatientEventType",
                "symbols": [
                    "patient.created",
                    "patient.updated",
                    "patient.deleted",
                    "patient.merged",
                    "patient.activated",
                    "patient.deactivated"
                ]
            },
            "doc": "Type of patient event"
        },
        {
            "name": "patient_id",
            "type": "string",
            "doc": "Patient identifier"
        },
        {
            "name": "operation",
            "type": "string",
            "doc": "Operation performed (created, updated, deleted, etc.)"
        },
        {
            "name": "patient_data",
            "type": "string",
            "doc": "Complete patient data as JSON string"
        },
        {
            "name": "timestamp",
            "type": "long",
            "doc": "Event timestamp (Unix timestamp)"
        },
        {
            "name": "service",
            "type": "string",
            "doc": "Source service name"
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
            "doc": "Causation ID for event sourcing"
        },
        {
            "name": "metadata",
            "type": ["null", "string"],
            "default": None,
            "doc": "Additional metadata as JSON string"
        }
    ]
}

# Encounter Events Schema
ENCOUNTER_EVENT_SCHEMA = {
    "type": "record",
    "name": "EncounterEvent",
    "namespace": "com.clinicalsynthesishub.events.encounter",
    "doc": "Event for encounter management operations",
    "fields": [
        {
            "name": "event_id",
            "type": "string",
            "doc": "Unique event identifier"
        },
        {
            "name": "event_type",
            "type": {
                "type": "enum",
                "name": "EncounterEventType",
                "symbols": [
                    "encounter.created",
                    "encounter.updated",
                    "encounter.started",
                    "encounter.finished",
                    "encounter.cancelled",
                    "encounter.entered_in_error"
                ]
            },
            "doc": "Type of encounter event"
        },
        {
            "name": "encounter_id",
            "type": "string",
            "doc": "Encounter identifier"
        },
        {
            "name": "patient_id",
            "type": "string",
            "doc": "Patient identifier"
        },
        {
            "name": "operation",
            "type": "string",
            "doc": "Operation performed"
        },
        {
            "name": "encounter_data",
            "type": "string",
            "doc": "Complete encounter data as JSON string"
        },
        {
            "name": "timestamp",
            "type": "long",
            "doc": "Event timestamp (Unix timestamp)"
        },
        {
            "name": "service",
            "type": "string",
            "doc": "Source service name"
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
            "doc": "Causation ID for event sourcing"
        },
        {
            "name": "metadata",
            "type": ["null", "string"],
            "default": None,
            "doc": "Additional metadata as JSON string"
        }
    ]
}

# Observation Events Schema
OBSERVATION_EVENT_SCHEMA = {
    "type": "record",
    "name": "ObservationEvent",
    "namespace": "com.clinicalsynthesishub.events.observation",
    "doc": "Event for observation/measurement operations",
    "fields": [
        {
            "name": "event_id",
            "type": "string",
            "doc": "Unique event identifier"
        },
        {
            "name": "event_type",
            "type": {
                "type": "enum",
                "name": "ObservationEventType",
                "symbols": [
                    "observation.created",
                    "observation.updated",
                    "observation.amended",
                    "observation.cancelled",
                    "observation.entered_in_error"
                ]
            },
            "doc": "Type of observation event"
        },
        {
            "name": "observation_id",
            "type": "string",
            "doc": "Observation identifier"
        },
        {
            "name": "patient_id",
            "type": "string",
            "doc": "Patient identifier"
        },
        {
            "name": "observation_type",
            "type": "string",
            "doc": "Type of observation (vital signs, lab result, etc.)"
        },
        {
            "name": "operation",
            "type": "string",
            "doc": "Operation performed"
        },
        {
            "name": "observation_data",
            "type": "string",
            "doc": "Complete observation data as JSON string"
        },
        {
            "name": "timestamp",
            "type": "long",
            "doc": "Event timestamp (Unix timestamp)"
        },
        {
            "name": "service",
            "type": "string",
            "doc": "Source service name"
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
            "doc": "Causation ID for event sourcing"
        },
        {
            "name": "metadata",
            "type": ["null", "string"],
            "default": None,
            "doc": "Additional metadata as JSON string"
        }
    ]
}

# Workflow Events Schema
WORKFLOW_EVENT_SCHEMA = {
    "type": "record",
    "name": "WorkflowEvent",
    "namespace": "com.clinicalsynthesishub.events.workflow",
    "doc": "Event for workflow engine operations",
    "fields": [
        {
            "name": "event_id",
            "type": "string",
            "doc": "Unique event identifier"
        },
        {
            "name": "event_type",
            "type": {
                "type": "enum",
                "name": "WorkflowEventType",
                "symbols": [
                    "workflow.started",
                    "workflow.completed",
                    "workflow.failed",
                    "workflow.cancelled",
                    "task.created",
                    "task.completed",
                    "task.failed",
                    "task.assigned"
                ]
            },
            "doc": "Type of workflow event"
        },
        {
            "name": "workflow_instance_id",
            "type": "string",
            "doc": "Workflow instance identifier"
        },
        {
            "name": "workflow_definition_id",
            "type": "string",
            "doc": "Workflow definition identifier"
        },
        {
            "name": "task_id",
            "type": ["null", "string"],
            "default": None,
            "doc": "Task identifier (for task events)"
        },
        {
            "name": "patient_id",
            "type": ["null", "string"],
            "default": None,
            "doc": "Associated patient identifier"
        },
        {
            "name": "operation",
            "type": "string",
            "doc": "Operation performed"
        },
        {
            "name": "workflow_data",
            "type": "string",
            "doc": "Workflow/task data as JSON string"
        },
        {
            "name": "timestamp",
            "type": "long",
            "doc": "Event timestamp (Unix timestamp)"
        },
        {
            "name": "service",
            "type": "string",
            "doc": "Source service name"
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
            "doc": "Causation ID for event sourcing"
        },
        {
            "name": "metadata",
            "type": ["null", "string"],
            "default": None,
            "doc": "Additional metadata as JSON string"
        }
    ]
}

# Schema registry for easy access
BUSINESS_EVENT_SCHEMAS = {
    "order-events": ORDER_EVENT_SCHEMA,
    "patient-events": PATIENT_EVENT_SCHEMA,
    "encounter-events": ENCOUNTER_EVENT_SCHEMA,
    "observation-events": OBSERVATION_EVENT_SCHEMA,
    "workflow-events": WORKFLOW_EVENT_SCHEMA
}
