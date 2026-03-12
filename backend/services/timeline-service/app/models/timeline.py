from typing import Dict, List, Optional, Any, Union
from pydantic import BaseModel, Field
from datetime import datetime
from enum import Enum

# Import shared FHIR models
from shared.models import (
    Patient, Observation, Condition, Encounter,
    Medication, MedicationRequest, MedicationAdministration, MedicationStatement,
    DiagnosticReport, CodeableConcept, Reference
)

class EventType(str, Enum):
    """Types of events in the timeline"""
    OBSERVATION = "observation"
    CONDITION = "condition"
    MEDICATION = "medication"
    ENCOUNTER = "encounter"
    DOCUMENT = "document"

class ResourceType(str, Enum):
    """FHIR resource types"""
    OBSERVATION = "Observation"
    CONDITION = "Condition"
    MEDICATION_REQUEST = "MedicationRequest"
    MEDICATION_ADMINISTRATION = "MedicationAdministration"
    MEDICATION_STATEMENT = "MedicationStatement"
    ENCOUNTER = "Encounter"
    DOCUMENT_REFERENCE = "DocumentReference"

class EventDetails(BaseModel):
    """Details for a timeline event"""
    code: Optional[str] = None
    value: Optional[str] = None
    unit: Optional[str] = None
    display: Optional[str] = None

    @classmethod
    def from_codeable_concept(cls, concept: CodeableConcept) -> 'EventDetails':
        """Create event details from a CodeableConcept."""
        if not concept or not concept.coding or len(concept.coding) == 0:
            return cls()

        coding = concept.coding[0]
        return cls(
            code=coding.code,
            display=coding.display
        )

    @classmethod
    def from_quantity(cls, quantity: Optional[Dict[str, Any]]) -> 'EventDetails':
        """Create event details from a Quantity."""
        if not quantity:
            return cls()

        return cls(
            value=str(quantity.get('value', '')),
            unit=quantity.get('unit', '')
        )

class TimelineEvent(BaseModel):
    """Model for a timeline event"""
    id: str
    patient_id: str
    event_type: str
    resource_type: str
    resource_id: str
    title: str
    description: Optional[str] = None
    date: str
    details: Optional[EventDetails] = None

    class Config:
        schema_extra = {
            "example": {
                "id": "obs-123",
                "patient_id": "patient-456",
                "event_type": "observation",
                "resource_type": "Observation",
                "resource_id": "123",
                "title": "Blood Pressure",
                "description": "120/80 mmHg",
                "date": "2023-04-15T10:30:00Z",
                "details": {
                    "code": "8480-6",
                    "value": "120/80",
                    "unit": "mmHg",
                    "display": "Systolic/Diastolic Blood Pressure"
                }
            }
        }

    @classmethod
    def from_observation(cls, obs: Dict[str, Any], patient_id: str) -> 'TimelineEvent':
        """Create a timeline event from an Observation resource."""
        obs_id = obs.get('id', '')

        # Get the observation date
        date = obs.get('effectiveDateTime', obs.get('issued', ''))

        # Get the observation title/display
        title = "Observation"
        if 'code' in obs and 'coding' in obs['code'] and len(obs['code']['coding']) > 0:
            coding = obs['code']['coding'][0]
            title = coding.get('display', title)

        # Get the observation value
        description = None
        if 'valueQuantity' in obs:
            value_quantity = obs['valueQuantity']
            value = value_quantity.get('value', '')
            unit = value_quantity.get('unit', '')
            description = f"{value} {unit}"

        # Create event details
        details = None
        if 'code' in obs and 'coding' in obs['code'] and len(obs['code']['coding']) > 0:
            code_concept = CodeableConcept.model_validate(obs['code'])
            details = EventDetails.from_codeable_concept(code_concept)

            if 'valueQuantity' in obs:
                value_details = EventDetails.from_quantity(obs['valueQuantity'])
                details.value = value_details.value
                details.unit = value_details.unit

        return cls(
            id=f"obs-{obs_id}",
            patient_id=patient_id,
            event_type="observation",
            resource_type="Observation",
            resource_id=obs_id,
            title=title,
            description=description,
            date=date,
            details=details
        )

    @classmethod
    def from_condition(cls, cond: Dict[str, Any], patient_id: str) -> 'TimelineEvent':
        """Create a timeline event from a Condition resource."""
        cond_id = cond.get('id', '')

        # Get the condition date (onset or recorded date)
        date = cond.get('onsetDateTime', cond.get('recordedDate', ''))

        # Get the condition title/display
        title = "Condition"
        if 'code' in cond and 'coding' in cond['code'] and len(cond['code']['coding']) > 0:
            coding = cond['code']['coding'][0]
            title = coding.get('display', title)

        # Get the condition status
        description = None
        if 'clinicalStatus' in cond and 'coding' in cond['clinicalStatus'] and len(cond['clinicalStatus']['coding']) > 0:
            status = cond['clinicalStatus']['coding'][0].get('display', '')
            description = f"Status: {status}"

        # Create event details
        details = None
        if 'code' in cond:
            code_concept = CodeableConcept.model_validate(cond['code'])
            details = EventDetails.from_codeable_concept(code_concept)

        return cls(
            id=f"cond-{cond_id}",
            patient_id=patient_id,
            event_type="condition",
            resource_type="Condition",
            resource_id=cond_id,
            title=title,
            description=description,
            date=date,
            details=details
        )

    @classmethod
    def from_medication_request(cls, med: Dict[str, Any], patient_id: str) -> 'TimelineEvent':
        """Create a timeline event from a MedicationRequest resource."""
        med_id = med.get('id', '')

        # Get the medication date
        date = med.get('authoredOn', '')

        # Get the medication title/display
        title = "Medication"
        if 'medicationCodeableConcept' in med and 'coding' in med['medicationCodeableConcept'] and len(med['medicationCodeableConcept']['coding']) > 0:
            coding = med['medicationCodeableConcept']['coding'][0]
            title = coding.get('display', title)

        # Get the medication dosage
        description = None
        if 'dosageInstruction' in med and len(med['dosageInstruction']) > 0:
            dosage = med['dosageInstruction'][0]
            if 'text' in dosage:
                description = dosage['text']

        # Create event details
        details = None
        if 'medicationCodeableConcept' in med:
            code_concept = CodeableConcept.model_validate(med['medicationCodeableConcept'])
            details = EventDetails.from_codeable_concept(code_concept)

        return cls(
            id=f"med-{med_id}",
            patient_id=patient_id,
            event_type="medication",
            resource_type="MedicationRequest",
            resource_id=med_id,
            title=title,
            description=description,
            date=date,
            details=details
        )

    @classmethod
    def from_encounter(cls, enc: Dict[str, Any], patient_id: str) -> 'TimelineEvent':
        """Create a timeline event from an Encounter resource."""
        enc_id = enc.get('id', '')

        # Get the encounter date
        date = ''
        if 'period' in enc and 'start' in enc['period']:
            date = enc['period']['start']

        # Get the encounter title/display
        title = "Encounter"
        if 'type' in enc and len(enc['type']) > 0 and 'coding' in enc['type'][0] and len(enc['type'][0]['coding']) > 0:
            coding = enc['type'][0]['coding'][0]
            title = coding.get('display', title)

        # Get the encounter class
        description = None
        if 'class' in enc and 'display' in enc['class']:
            class_display = enc['class']['display']
            description = f"Class: {class_display}"

        # Create event details
        details = None
        if 'type' in enc and len(enc['type']) > 0:
            code_concept = CodeableConcept.model_validate(enc['type'][0])
            details = EventDetails.from_codeable_concept(code_concept)

        return cls(
            id=f"enc-{enc_id}",
            patient_id=patient_id,
            event_type="encounter",
            resource_type="Encounter",
            resource_id=enc_id,
            title=title,
            description=description,
            date=date,
            details=details
        )

    @classmethod
    def from_document(cls, doc: Dict[str, Any], patient_id: str) -> 'TimelineEvent':
        """Create a timeline event from a DocumentReference resource."""
        doc_id = doc.get('id', '')

        # Get the document date
        date = doc.get('date', '')

        # Get the document title/display
        title = "Document"
        if 'type' in doc and 'coding' in doc['type'] and len(doc['type']['coding']) > 0:
            coding = doc['type']['coding'][0]
            title = coding.get('display', title)

        # Get the document description
        description = None
        if 'description' in doc:
            description = doc['description']

        # Create event details
        details = None
        if 'type' in doc:
            code_concept = CodeableConcept.model_validate(doc['type'])
            details = EventDetails.from_codeable_concept(code_concept)

        return cls(
            id=f"doc-{doc_id}",
            patient_id=patient_id,
            event_type="document",
            resource_type="DocumentReference",
            resource_id=doc_id,
            title=title,
            description=description,
            date=date,
            details=details
        )

class PatientTimeline(BaseModel):
    """Model for a patient's timeline"""
    patient_id: str
    events: List[TimelineEvent]

    class Config:
        schema_extra = {
            "example": {
                "patient_id": "patient-456",
                "events": [
                    {
                        "id": "obs-123",
                        "patient_id": "patient-456",
                        "event_type": "observation",
                        "resource_type": "Observation",
                        "resource_id": "123",
                        "title": "Blood Pressure",
                        "description": "120/80 mmHg",
                        "date": "2023-04-15T10:30:00Z",
                        "details": {
                            "code": "8480-6",
                            "value": "120/80",
                            "unit": "mmHg",
                            "display": "Systolic/Diastolic Blood Pressure"
                        }
                    }
                ]
            }
        }

class TimelineFilter(BaseModel):
    """Model for filtering timeline events"""
    start_date: Optional[str] = None
    end_date: Optional[str] = None
    event_types: Optional[List[str]] = None
    resource_types: Optional[List[str]] = None

    class Config:
        schema_extra = {
            "example": {
                "start_date": "2023-01-01T00:00:00Z",
                "end_date": "2023-12-31T23:59:59Z",
                "event_types": ["observation", "medication"],
                "resource_types": ["Observation", "MedicationRequest"]
            }
        }
