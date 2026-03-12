"""
Event schemas and data structures for Clinical Synthesis Hub
"""

import json
from typing import Any, Dict, Optional, List
from dataclasses import dataclass, asdict
from datetime import datetime
from enum import Enum

class EventVersion(str, Enum):
    """Event schema versions"""
    V1_0 = "1.0"
    V1_1 = "1.1"
    V2_0 = "2.0"

@dataclass
class EventEnvelope:
    """Standard event envelope for all events in the system"""
    
    # CloudEvents specification fields
    id: str                                    # Unique event identifier
    source: str                               # Event source (service name)
    type: str                                 # Event type
    subject: str                              # Event subject (resource identifier)
    time: str                                 # Event timestamp (ISO 8601)
    
    # Event data
    data: Dict[str, Any]                      # Event payload
    
    # Optional fields
    correlation_id: Optional[str] = None      # Correlation ID for tracing
    causation_id: Optional[str] = None        # ID of the event that caused this event
    metadata: Optional[Dict[str, Any]] = None # Additional metadata
    version: str = EventVersion.V1_0          # Schema version
    
    def __post_init__(self):
        """Initialize default values"""
        if self.metadata is None:
            self.metadata = {}
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return asdict(self)
    
    def to_json(self) -> str:
        """Convert to JSON string"""
        return json.dumps(self.to_dict(), default=str)
    
    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> 'EventEnvelope':
        """Create from dictionary"""
        return cls(**data)
    
    @classmethod
    def from_json(cls, json_str: str) -> 'EventEnvelope':
        """Create from JSON string"""
        data = json.loads(json_str)
        return cls.from_dict(data)

@dataclass
class FHIRResourceEvent:
    """Event data for FHIR resource operations"""
    
    resource_type: str                        # FHIR resource type
    resource_id: str                          # Resource identifier
    operation: str                            # CRUD operation (created, updated, deleted)
    resource: Optional[Dict[str, Any]] = None # Full resource data (for created/updated)
    previous_resource: Optional[Dict[str, Any]] = None # Previous version (for updated)
    
    # FHIR-specific metadata
    version_id: Optional[str] = None          # Resource version
    last_updated: Optional[str] = None        # Last updated timestamp
    profile: Optional[List[str]] = None       # FHIR profiles
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return asdict(self)

@dataclass
class DeviceDataEvent:
    """Event data for device data ingestion"""
    
    device_id: str                            # Device identifier
    device_type: str                          # Type of device
    patient_id: Optional[str] = None          # Associated patient
    measurements: Optional[List[Dict[str, Any]]] = None # Measurement data
    raw_data: Optional[Dict[str, Any]] = None # Raw device data
    
    # Device metadata
    manufacturer: Optional[str] = None        # Device manufacturer
    model: Optional[str] = None               # Device model
    firmware_version: Optional[str] = None    # Firmware version
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return asdict(self)

@dataclass
class ClinicalAlertEvent:
    """Event data for clinical alerts"""
    
    alert_type: str                           # Type of alert
    severity: str                             # Alert severity (low, medium, high, critical)
    patient_id: str                           # Patient identifier
    message: str                              # Alert message
    
    # Alert context
    triggered_by: Optional[str] = None        # What triggered the alert
    resource_references: Optional[List[str]] = None # Related FHIR resources
    recommended_actions: Optional[List[str]] = None # Recommended actions
    
    # Alert metadata
    expires_at: Optional[str] = None          # Alert expiration
    acknowledged_by: Optional[str] = None     # Who acknowledged the alert
    acknowledged_at: Optional[str] = None     # When acknowledged
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return asdict(self)

@dataclass
class WorkflowEvent:
    """Event data for workflow operations"""
    
    workflow_id: str                          # Workflow instance ID
    workflow_type: str                        # Type of workflow
    event_type: str                           # Workflow event type
    
    # Workflow context
    patient_id: Optional[str] = None          # Associated patient
    task_id: Optional[str] = None             # Associated task
    user_id: Optional[str] = None             # User performing action
    
    # Workflow data
    workflow_data: Optional[Dict[str, Any]] = None # Workflow variables
    task_data: Optional[Dict[str, Any]] = None     # Task-specific data
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary"""
        return asdict(self)

class SchemaRegistry:
    """Schema registry for event validation and evolution"""

    def __init__(self):
        """Initialize schema registry"""
        self.schemas = {}
        self._register_default_schemas()

    def _register_default_schemas(self):
        """Register default schemas"""
        # Import Avro schemas
        try:
            from .avro_schemas import get_all_schemas
            avro_schemas = get_all_schemas()

            # Convert Avro schemas to JSON Schema format for compatibility
            for schema_name, avro_schema in avro_schemas.items():
                json_schema = self._convert_avro_to_json_schema(avro_schema)
                self.schemas[schema_name] = json_schema

        except ImportError:
            # Fallback to basic schemas if Avro schemas not available
            self._register_fallback_schemas()

    def _register_fallback_schemas(self):
        """Register fallback schemas when Avro schemas not available"""
        # FHIR resource event schema
        self.schemas['fhir-resource-event'] = {
            'version': '1.0',
            'type': 'object',
            'properties': {
                'resource_type': {'type': 'string'},
                'resource_id': {'type': 'string'},
                'operation': {'type': 'string', 'enum': ['created', 'updated', 'deleted']},
                'resource': {'type': 'object'},
                'previous_resource': {'type': 'object'},
                'version_id': {'type': 'string'},
                'last_updated': {'type': 'string'},
                'profile': {'type': 'array', 'items': {'type': 'string'}}
            },
            'required': ['resource_type', 'resource_id', 'operation']
        }

        # Device data event schema
        self.schemas['device-data-event'] = {
            'version': '1.0',
            'type': 'object',
            'properties': {
                'device_id': {'type': 'string'},
                'device_type': {'type': 'string'},
                'patient_id': {'type': 'string'},
                'measurements': {'type': 'array'},
                'raw_data': {'type': 'object'},
                'manufacturer': {'type': 'string'},
                'model': {'type': 'string'},
                'firmware_version': {'type': 'string'}
            },
            'required': ['device_id', 'device_type']
        }

        # Clinical alert event schema
        self.schemas['clinical-alert-event'] = {
            'version': '1.0',
            'type': 'object',
            'properties': {
                'alert_type': {'type': 'string'},
                'severity': {'type': 'string', 'enum': ['low', 'medium', 'high', 'critical']},
                'patient_id': {'type': 'string'},
                'message': {'type': 'string'},
                'triggered_by': {'type': 'string'},
                'resource_references': {'type': 'array', 'items': {'type': 'string'}},
                'recommended_actions': {'type': 'array', 'items': {'type': 'string'}},
                'expires_at': {'type': 'string'},
                'acknowledged_by': {'type': 'string'},
                'acknowledged_at': {'type': 'string'}
            },
            'required': ['alert_type', 'severity', 'patient_id', 'message']
        }

        # Workflow event schema
        self.schemas['workflow-event'] = {
            'version': '1.0',
            'type': 'object',
            'properties': {
                'workflow_id': {'type': 'string'},
                'workflow_type': {'type': 'string'},
                'event_type': {'type': 'string'},
                'patient_id': {'type': 'string'},
                'task_id': {'type': 'string'},
                'user_id': {'type': 'string'},
                'workflow_data': {'type': 'object'},
                'task_data': {'type': 'object'}
            },
            'required': ['workflow_id', 'workflow_type', 'event_type']
        }

    def _convert_avro_to_json_schema(self, avro_schema: Dict[str, Any]) -> Dict[str, Any]:
        """Convert Avro schema to JSON Schema format"""
        # Basic conversion - in a full implementation this would be more comprehensive
        json_schema = {
            'version': '1.0',
            'type': 'object',
            'properties': {},
            'required': []
        }

        for field in avro_schema.get('fields', []):
            field_name = field['name']
            field_type = field['type']

            # Convert Avro type to JSON Schema type
            if isinstance(field_type, str):
                json_type = self._avro_type_to_json_type(field_type)
            elif isinstance(field_type, list):
                # Union type - check if nullable
                if 'null' in field_type:
                    non_null_types = [t for t in field_type if t != 'null']
                    if len(non_null_types) == 1:
                        json_type = self._avro_type_to_json_type(non_null_types[0])
                    else:
                        json_type = 'string'  # Fallback
                else:
                    json_type = 'string'  # Fallback
            elif isinstance(field_type, dict):
                if field_type.get('type') == 'enum':
                    json_type = {'type': 'string', 'enum': field_type.get('symbols', [])}
                elif field_type.get('type') == 'array':
                    json_type = {'type': 'array', 'items': {'type': 'string'}}
                else:
                    json_type = 'string'  # Fallback
            else:
                json_type = 'string'  # Fallback

            json_schema['properties'][field_name] = json_type if isinstance(json_type, dict) else {'type': json_type}

            # Check if field is required (no default and not nullable)
            if 'default' not in field and not self._is_nullable_avro_type(field_type):
                json_schema['required'].append(field_name)

        return json_schema

    def _avro_type_to_json_type(self, avro_type: str) -> str:
        """Convert Avro primitive type to JSON Schema type"""
        type_mapping = {
            'string': 'string',
            'int': 'integer',
            'long': 'integer',
            'float': 'number',
            'double': 'number',
            'boolean': 'boolean',
            'bytes': 'string',
            'null': 'null'
        }
        return type_mapping.get(avro_type, 'string')

    def _is_nullable_avro_type(self, field_type) -> bool:
        """Check if Avro field type is nullable"""
        if isinstance(field_type, list):
            return 'null' in field_type
        return False

    def register_schema(self, name: str, schema: Dict[str, Any]):
        """Register a new schema"""
        self.schemas[name] = schema

    def get_schema(self, name: str) -> Optional[Dict[str, Any]]:
        """Get a schema by name"""
        return self.schemas.get(name)

    def validate_event(self, event_type: str, data: Dict[str, Any]) -> bool:
        """Validate event data against schema"""
        # For now, we'll do basic validation
        # In a full implementation, we'd use jsonschema or similar
        schema = self.get_schema(event_type)
        if not schema:
            return True  # No schema, assume valid

        required_fields = schema.get('required', [])
        for field in required_fields:
            if field not in data:
                return False

        return True

    def list_schemas(self) -> List[str]:
        """List all registered schemas"""
        return list(self.schemas.keys())

# Global schema registry instance
schema_registry = SchemaRegistry()
