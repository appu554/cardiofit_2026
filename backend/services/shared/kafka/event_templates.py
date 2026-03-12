"""
Event templates and utilities for Clinical Synthesis Hub
"""

import json
import logging
from typing import Dict, Any, Optional, List
from datetime import datetime, timezone
from uuid import uuid4

from .schemas import EventEnvelope, FHIRResourceEvent, DeviceDataEvent, ClinicalAlertEvent, WorkflowEvent
from .config import TopicNames, EventTypes
from .schema_evolution import get_schema_manager

logger = logging.getLogger(__name__)

class EventTemplateBuilder:
    """Builder for creating standardized events"""
    
    def __init__(self, source: str):
        """Initialize with event source"""
        self.source = source
        self.schema_manager = get_schema_manager()
    
    def create_fhir_event(
        self,
        resource_type: str,
        resource_id: str,
        operation: str,
        resource_data: Optional[Dict[str, Any]] = None,
        previous_resource: Optional[Dict[str, Any]] = None,
        correlation_id: Optional[str] = None,
        causation_id: Optional[str] = None,
        patient_id: Optional[str] = None,
        encounter_id: Optional[str] = None
    ) -> EventEnvelope:
        """Create a FHIR resource event"""
        
        # Create FHIR event data
        fhir_event = FHIRResourceEvent(
            resource_type=resource_type,
            resource_id=resource_id,
            operation=operation,
            resource=resource_data,
            previous_resource=previous_resource,
            version_id=resource_data.get('meta', {}).get('versionId') if resource_data else None,
            last_updated=resource_data.get('meta', {}).get('lastUpdated') if resource_data else None,
            profile=resource_data.get('meta', {}).get('profile', []) if resource_data else []
        )
        
        # Add patient/encounter context if available
        if patient_id:
            fhir_event.patient_id = patient_id
        elif resource_data and resource_type == 'Patient':
            fhir_event.patient_id = resource_id
        elif resource_data:
            # Try to extract patient reference from resource
            patient_ref = self._extract_patient_reference(resource_data)
            if patient_ref:
                fhir_event.patient_id = patient_ref
        
        if encounter_id:
            fhir_event.encounter_id = encounter_id
        elif resource_data:
            # Try to extract encounter reference from resource
            encounter_ref = self._extract_encounter_reference(resource_data)
            if encounter_ref:
                fhir_event.encounter_id = encounter_ref
        
        # Create event envelope
        event_type = f"{resource_type.lower()}.{operation}"
        subject = f"{resource_type}/{resource_id}"
        
        envelope = EventEnvelope(
            id=str(uuid4()),
            source=self.source,
            type=event_type,
            subject=subject,
            time=datetime.now(timezone.utc).isoformat(),
            data=fhir_event.to_dict(),
            correlation_id=correlation_id,
            causation_id=causation_id,
            metadata={
                'fhir_version': 'R4',
                'resource_type': resource_type,
                'schema_version': '1.0'
            }
        )
        
        return envelope
    
    def create_device_data_event(
        self,
        device_id: str,
        device_type: str,
        measurements: List[Dict[str, Any]],
        patient_id: Optional[str] = None,
        raw_data: Optional[Dict[str, Any]] = None,
        device_metadata: Optional[Dict[str, Any]] = None,
        correlation_id: Optional[str] = None
    ) -> EventEnvelope:
        """Create a device data event"""
        
        # Create device event data
        device_event = DeviceDataEvent(
            device_id=device_id,
            device_type=device_type,
            patient_id=patient_id,
            measurements=measurements,
            raw_data=raw_data,
            manufacturer=device_metadata.get('manufacturer') if device_metadata else None,
            model=device_metadata.get('model') if device_metadata else None,
            firmware_version=device_metadata.get('firmware_version') if device_metadata else None
        )
        
        # Create event envelope
        envelope = EventEnvelope(
            id=str(uuid4()),
            source=self.source,
            type=EventTypes.OBSERVATION_RECORDED,
            subject=f"Device/{device_id}",
            time=datetime.now(timezone.utc).isoformat(),
            data=device_event.to_dict(),
            correlation_id=correlation_id,
            metadata={
                'device_type': device_type,
                'measurement_count': len(measurements),
                'schema_version': '1.0'
            }
        )
        
        return envelope
    
    def create_clinical_alert_event(
        self,
        alert_type: str,
        severity: str,
        patient_id: str,
        message: str,
        triggered_by: Optional[str] = None,
        resource_references: Optional[List[str]] = None,
        recommended_actions: Optional[List[str]] = None,
        expires_at: Optional[str] = None,
        correlation_id: Optional[str] = None
    ) -> EventEnvelope:
        """Create a clinical alert event"""
        
        # Create alert event data
        alert_event = ClinicalAlertEvent(
            alert_type=alert_type,
            severity=severity,
            patient_id=patient_id,
            message=message,
            triggered_by=triggered_by,
            resource_references=resource_references or [],
            recommended_actions=recommended_actions or [],
            expires_at=expires_at
        )
        
        # Create event envelope
        envelope = EventEnvelope(
            id=str(uuid4()),
            source=self.source,
            type=f"alert.{alert_type}",
            subject=f"Patient/{patient_id}",
            time=datetime.now(timezone.utc).isoformat(),
            data=alert_event.to_dict(),
            correlation_id=correlation_id,
            metadata={
                'alert_severity': severity,
                'alert_type': alert_type,
                'schema_version': '1.0'
            }
        )
        
        return envelope
    
    def create_workflow_event(
        self,
        workflow_id: str,
        workflow_type: str,
        event_type: str,
        patient_id: Optional[str] = None,
        task_id: Optional[str] = None,
        user_id: Optional[str] = None,
        workflow_data: Optional[Dict[str, Any]] = None,
        task_data: Optional[Dict[str, Any]] = None,
        error_message: Optional[str] = None,
        duration_ms: Optional[int] = None,
        correlation_id: Optional[str] = None,
        causation_id: Optional[str] = None
    ) -> EventEnvelope:
        """Create a workflow event"""
        
        # Create workflow event data
        workflow_event = WorkflowEvent(
            workflow_id=workflow_id,
            workflow_type=workflow_type,
            event_type=event_type,
            patient_id=patient_id,
            task_id=task_id,
            user_id=user_id,
            workflow_data=workflow_data,
            task_data=task_data
        )
        
        # Create event envelope
        subject = f"Workflow/{workflow_id}"
        if task_id:
            subject += f"/Task/{task_id}"
        
        envelope = EventEnvelope(
            id=str(uuid4()),
            source=self.source,
            type=f"workflow.{event_type}",
            subject=subject,
            time=datetime.now(timezone.utc).isoformat(),
            data=workflow_event.to_dict(),
            correlation_id=correlation_id,
            causation_id=causation_id,
            metadata={
                'workflow_type': workflow_type,
                'workflow_event_type': event_type,
                'schema_version': '1.0'
            }
        )
        
        return envelope
    
    def _extract_patient_reference(self, resource: Dict[str, Any]) -> Optional[str]:
        """Extract patient reference from FHIR resource"""
        # Common patterns for patient references
        patient_paths = [
            'subject.reference',
            'patient.reference', 
            'subject.id',
            'patient.id'
        ]
        
        for path in patient_paths:
            value = self._get_nested_value(resource, path)
            if value:
                # Extract ID from reference (e.g., "Patient/123" -> "123")
                if isinstance(value, str) and '/' in value:
                    return value.split('/')[-1]
                return str(value)
        
        return None
    
    def _extract_encounter_reference(self, resource: Dict[str, Any]) -> Optional[str]:
        """Extract encounter reference from FHIR resource"""
        encounter_paths = [
            'encounter.reference',
            'context.reference',
            'encounter.id',
            'context.id'
        ]
        
        for path in encounter_paths:
            value = self._get_nested_value(resource, path)
            if value:
                # Extract ID from reference
                if isinstance(value, str) and '/' in value:
                    return value.split('/')[-1]
                return str(value)
        
        return None
    
    def _get_nested_value(self, data: Dict[str, Any], path: str) -> Any:
        """Get nested value from dictionary using dot notation"""
        keys = path.split('.')
        current = data
        
        for key in keys:
            if isinstance(current, dict) and key in current:
                current = current[key]
            else:
                return None
        
        return current

class EventValidator:
    """Validates events against schemas"""
    
    def __init__(self):
        self.schema_manager = get_schema_manager()
    
    def validate_event(self, envelope: EventEnvelope) -> tuple[bool, Optional[str]]:
        """Validate event envelope and data"""
        
        # Validate envelope structure
        try:
            envelope_dict = envelope.to_dict()
            is_valid, error = self.schema_manager.validate_data('event-envelope', envelope_dict)
            if not is_valid:
                return False, f"Envelope validation failed: {error}"
        except Exception as e:
            return False, f"Envelope validation error: {str(e)}"
        
        # Validate event data based on type
        event_type = envelope.type
        data = envelope.data
        
        # Determine schema based on event type
        schema_name = self._get_schema_name_for_event_type(event_type)
        if schema_name:
            is_valid, error = self.schema_manager.validate_data(schema_name, data)
            if not is_valid:
                return False, f"Data validation failed: {error}"
        
        return True, None
    
    def _get_schema_name_for_event_type(self, event_type: str) -> Optional[str]:
        """Get schema name based on event type"""
        if any(event_type.startswith(prefix) for prefix in ['patient.', 'encounter.', 'observation.', 'medication.', 'condition.']):
            return 'fhir-resource-event'
        elif event_type.startswith('alert.'):
            return 'clinical-alert-event'
        elif event_type.startswith('workflow.'):
            return 'workflow-event'
        elif 'device' in event_type.lower():
            return 'device-data-event'
        
        return None

# Utility functions for common event patterns
def create_patient_created_event(
    source: str,
    patient_id: str,
    patient_data: Dict[str, Any],
    correlation_id: Optional[str] = None
) -> EventEnvelope:
    """Create a patient created event"""
    builder = EventTemplateBuilder(source)
    return builder.create_fhir_event(
        resource_type='Patient',
        resource_id=patient_id,
        operation='created',
        resource_data=patient_data,
        correlation_id=correlation_id
    )

def create_observation_recorded_event(
    source: str,
    observation_id: str,
    observation_data: Dict[str, Any],
    patient_id: Optional[str] = None,
    correlation_id: Optional[str] = None
) -> EventEnvelope:
    """Create an observation recorded event"""
    builder = EventTemplateBuilder(source)
    return builder.create_fhir_event(
        resource_type='Observation',
        resource_id=observation_id,
        operation='created',
        resource_data=observation_data,
        patient_id=patient_id,
        correlation_id=correlation_id
    )

def create_critical_value_alert(
    source: str,
    patient_id: str,
    value: str,
    reference_range: str,
    observation_id: str,
    correlation_id: Optional[str] = None
) -> EventEnvelope:
    """Create a critical value alert"""
    builder = EventTemplateBuilder(source)
    return builder.create_clinical_alert_event(
        alert_type='critical_value',
        severity='critical',
        patient_id=patient_id,
        message=f"Critical value detected: {value} (normal: {reference_range})",
        triggered_by=f"Observation/{observation_id}",
        resource_references=[f"Observation/{observation_id}"],
        recommended_actions=['Review immediately', 'Contact physician', 'Verify result'],
        correlation_id=correlation_id
    )

# Global event validator
event_validator = EventValidator()

def validate_event(envelope: EventEnvelope) -> tuple[bool, Optional[str]]:
    """Validate an event envelope"""
    return event_validator.validate_event(envelope)
