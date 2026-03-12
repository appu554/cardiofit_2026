from typing import Dict, List, Optional, Any
from pydantic import BaseModel

# Import shared FHIR models
from shared.models import (
    Condition, CodeableConcept, Reference, Annotation
)

class ConditionCreate(BaseModel):
    """Model for creating a condition"""
    clinicalStatus: Optional[CodeableConcept] = None
    verificationStatus: Optional[CodeableConcept] = None
    category: List[CodeableConcept]
    code: CodeableConcept
    subject: Reference
    onsetDateTime: Optional[str] = None
    abatementDateTime: Optional[str] = None
    recordedDate: Optional[str] = None
    note: Optional[List[Annotation]] = None

    def to_fhir_condition(self) -> Condition:
        """Convert to a FHIR Condition."""
        data = self.model_dump(exclude_unset=True)
        return Condition(**data)

class ConditionUpdate(BaseModel):
    """Model for updating a condition"""
    clinicalStatus: Optional[CodeableConcept] = None
    verificationStatus: Optional[CodeableConcept] = None
    category: Optional[List[CodeableConcept]] = None
    code: Optional[CodeableConcept] = None
    subject: Optional[Reference] = None
    onsetDateTime: Optional[str] = None
    abatementDateTime: Optional[str] = None
    recordedDate: Optional[str] = None
    note: Optional[List[Annotation]] = None

    def to_fhir_condition_update(self) -> Dict[str, Any]:
        """Convert to a FHIR Condition update."""
        return self.model_dump(exclude_unset=True)

class ConditionInDB(BaseModel):
    """Model for a condition in the database"""
    id: str
    resourceType: str = "Condition"
    clinicalStatus: Optional[CodeableConcept] = None
    verificationStatus: Optional[CodeableConcept] = None
    category: List[CodeableConcept]
    code: CodeableConcept
    subject: Reference
    onsetDateTime: Optional[str] = None
    abatementDateTime: Optional[str] = None
    recordedDate: Optional[str] = None
    note: Optional[List[Annotation]] = None

    @classmethod
    def from_fhir_condition(cls, condition: Condition) -> 'ConditionInDB':
        """Create from a FHIR Condition."""
        data = condition.model_dump(exclude_unset=True)
        return cls(**data)
