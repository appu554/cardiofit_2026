from typing import Dict, List, Optional, Any
from pydantic import BaseModel, Field
from datetime import datetime
from enum import Enum

# Import shared FHIR models
from shared.models import (
    Encounter, CodeableConcept, Reference, Period,
    EncounterParticipant, EncounterDiagnosis, EncounterLocation
)

class EncounterStatus(str, Enum):
    """Encounter status based on FHIR standard"""
    PLANNED = "planned"
    ARRIVED = "arrived"
    TRIAGED = "triaged"
    IN_PROGRESS = "in-progress"
    ONLEAVE = "onleave"
    FINISHED = "finished"
    CANCELLED = "cancelled"
    ENTERED_IN_ERROR = "entered-in-error"
    UNKNOWN = "unknown"

class EncounterClass(str, Enum):
    """Encounter class based on FHIR standard"""
    AMBULATORY = "AMB"
    EMERGENCY = "EMER"
    INPATIENT = "IMP"
    OUTPATIENT = "OUTPATIENT"
    VIRTUAL = "VR"
    HOME_HEALTH = "HH"

class EncounterCreate(BaseModel):
    """Model for creating an encounter"""
    status: EncounterStatus
    class_value: Dict[str, str] = Field(..., alias="class")
    type: Optional[List[CodeableConcept]] = None
    subject: Reference
    participant: Optional[List[EncounterParticipant]] = None
    period: Optional[Period] = None
    reasonCode: Optional[List[CodeableConcept]] = None
    diagnosis: Optional[List[EncounterDiagnosis]] = None
    location: Optional[List[EncounterLocation]] = None
    serviceProvider: Optional[Reference] = None

    class Config:
        schema_extra = {
            "example": {
                "status": "in-progress",
                "class": {
                    "system": "http://terminology.hl7.org/CodeSystem/v3-ActCode",
                    "code": "AMB",
                    "display": "ambulatory"
                },
                "type": [
                    {
                        "coding": [
                            {
                                "system": "http://snomed.info/sct",
                                "code": "308335008",
                                "display": "Patient encounter procedure"
                            }
                        ],
                        "text": "Outpatient visit"
                    }
                ],
                "subject": {
                    "reference": "Patient/123",
                    "display": "John Smith"
                },
                "period": {
                    "start": "2023-06-15T08:00:00Z"
                }
            }
        }

    def to_fhir_encounter(self) -> Encounter:
        """Convert to a FHIR Encounter."""
        data = self.model_dump(exclude_unset=True)

        # Convert status enum to string
        if isinstance(data.get('status'), EncounterStatus):
            data['status'] = data['status'].value

        # Handle class field
        if 'class_value' in data:
            data['class'] = data.pop('class_value')

        return Encounter(**data)

class EncounterUpdate(BaseModel):
    """Model for updating an encounter"""
    status: Optional[EncounterStatus] = None
    class_value: Optional[Dict[str, str]] = Field(None, alias="class")
    type: Optional[List[CodeableConcept]] = None
    subject: Optional[Reference] = None
    participant: Optional[List[EncounterParticipant]] = None
    period: Optional[Period] = None
    reasonCode: Optional[List[CodeableConcept]] = None
    diagnosis: Optional[List[EncounterDiagnosis]] = None
    location: Optional[List[EncounterLocation]] = None
    serviceProvider: Optional[Reference] = None

    def to_fhir_encounter_update(self) -> Dict[str, Any]:
        """Convert to a FHIR Encounter update."""
        data = self.model_dump(exclude_unset=True)

        # Convert status enum to string
        if isinstance(data.get('status'), EncounterStatus):
            data['status'] = data['status'].value

        # Handle class field
        if 'class_value' in data:
            data['class'] = data.pop('class_value')

        return data

class EncounterInDB(BaseModel):
    """Model for an encounter in the database"""
    id: str
    resourceType: str = "Encounter"
    status: EncounterStatus
    class_value: Dict[str, str] = Field(..., alias="class")
    type: Optional[List[CodeableConcept]] = None
    subject: Reference
    participant: Optional[List[EncounterParticipant]] = None
    period: Optional[Period] = None
    reasonCode: Optional[List[CodeableConcept]] = None
    diagnosis: Optional[List[EncounterDiagnosis]] = None
    location: Optional[List[EncounterLocation]] = None
    serviceProvider: Optional[Reference] = None

    @classmethod
    def from_fhir_encounter(cls, encounter: Encounter) -> 'EncounterInDB':
        """Create from a FHIR Encounter."""
        data = encounter.model_dump(exclude_unset=True)

        # Handle class field
        if 'class' in data:
            data['class_value'] = data.pop('class')

        return cls(**data)
