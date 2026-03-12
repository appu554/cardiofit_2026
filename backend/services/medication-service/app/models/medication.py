from typing import Dict, List, Optional, Any, Union
from pydantic import BaseModel
from enum import Enum
from datetime import datetime

# Import shared FHIR models
from shared.models import (
    Medication, MedicationRequest, MedicationAdministration, MedicationStatement,
    CodeableConcept, Coding, Reference, Quantity, Ratio, Annotation
)

# Status enums for validation
class MedicationStatus(str, Enum):
    """Medication status based on FHIR standard"""
    ACTIVE = "active"
    INACTIVE = "inactive"
    ENTERED_IN_ERROR = "entered-in-error"

class MedicationRequestStatus(str, Enum):
    """Medication request status based on FHIR standard"""
    ACTIVE = "active"
    ON_HOLD = "on-hold"
    CANCELLED = "cancelled"
    COMPLETED = "completed"
    ENTERED_IN_ERROR = "entered-in-error"
    STOPPED = "stopped"
    DRAFT = "draft"
    UNKNOWN = "unknown"

class MedicationRequestIntent(str, Enum):
    """Medication request intent based on FHIR standard"""
    PROPOSAL = "proposal"
    PLAN = "plan"
    ORDER = "order"
    ORIGINAL_ORDER = "original-order"
    REFLEX_ORDER = "reflex-order"
    FILLER_ORDER = "filler-order"
    INSTANCE_ORDER = "instance-order"
    OPTION = "option"

class MedicationAdministrationStatus(str, Enum):
    """Medication administration status based on FHIR standard"""
    IN_PROGRESS = "in-progress"
    NOT_DONE = "not-done"
    ON_HOLD = "on-hold"
    COMPLETED = "completed"
    ENTERED_IN_ERROR = "entered-in-error"
    STOPPED = "stopped"
    UNKNOWN = "unknown"

class MedicationStatementStatus(str, Enum):
    """Medication statement status based on FHIR standard"""
    ACTIVE = "active"
    COMPLETED = "completed"
    ENTERED_IN_ERROR = "entered-in-error"
    INTENDED = "intended"
    STOPPED = "stopped"
    ON_HOLD = "on-hold"
    UNKNOWN = "unknown"
    NOT_TAKEN = "not-taken"

# Define DosageInstruction model
class DosageInstruction(BaseModel):
    """Dosage instructions"""
    sequence: Optional[int] = None
    text: Optional[str] = None
    timing: Optional[Dict[str, Any]] = None
    asNeededBoolean: Optional[bool] = None
    asNeededCodeableConcept: Optional[CodeableConcept] = None
    site: Optional[CodeableConcept] = None
    route: Optional[CodeableConcept] = None
    method: Optional[CodeableConcept] = None
    doseAndRate: Optional[List[Dict[str, Any]]] = None
    maxDosePerPeriod: Optional[Ratio] = None
    maxDosePerAdministration: Optional[Quantity] = None
    maxDosePerLifetime: Optional[Quantity] = None

# Create models for API endpoints
# These models will be converted to shared FHIR models

# Create models
class MedicationCreate(BaseModel):
    """Model for creating a medication"""
    status: Optional[MedicationStatus] = None
    code: CodeableConcept
    manufacturer: Optional[Reference] = None
    form: Optional[CodeableConcept] = None
    amount: Optional[Ratio] = None
    ingredient: Optional[List[Dict[str, Any]]] = None
    batch: Optional[Dict[str, Any]] = None

    def to_fhir_medication(self) -> Medication:
        """Convert to a FHIR Medication."""
        data = self.model_dump(exclude_unset=True)
        return Medication(**data)

class MedicationRequestCreate(BaseModel):
    """Model for creating a medication request"""
    status: MedicationRequestStatus = MedicationRequestStatus.ACTIVE
    intent: MedicationRequestIntent = MedicationRequestIntent.ORDER
    medicationCodeableConcept: Optional[CodeableConcept] = None
    medicationReference: Optional[Reference] = None
    subject: Reference
    encounter: Optional[Reference] = None
    authoredOn: Optional[str] = None
    requester: Optional[Reference] = None
    recorder: Optional[Reference] = None
    reasonCode: Optional[List[CodeableConcept]] = None
    reasonReference: Optional[List[Reference]] = None
    note: Optional[List[Annotation]] = None
    dosageInstruction: Optional[List[DosageInstruction]] = None
    dispenseRequest: Optional[Dict[str, Any]] = None
    substitution: Optional[Dict[str, Any]] = None
    priorPrescription: Optional[Reference] = None

    def to_fhir_medication_request(self) -> MedicationRequest:
        """Convert to a FHIR MedicationRequest."""
        data = self.model_dump(exclude_unset=True)
        # Convert status and intent to strings for the shared model
        if isinstance(data.get('status'), MedicationRequestStatus):
            data['status'] = data['status'].value
        if isinstance(data.get('intent'), MedicationRequestIntent):
            data['intent'] = data['intent'].value
        return MedicationRequest(**data)

class MedicationAdministrationCreate(BaseModel):
    """Model for creating a medication administration"""
    status: MedicationAdministrationStatus
    medicationCodeableConcept: Optional[CodeableConcept] = None
    medicationReference: Optional[Reference] = None
    subject: Reference
    context: Optional[Reference] = None
    effectiveDateTime: Optional[str] = None
    effectivePeriod: Optional[Dict[str, str]] = None
    performer: Optional[List[Dict[str, Any]]] = None
    reasonCode: Optional[List[CodeableConcept]] = None
    reasonReference: Optional[List[Reference]] = None
    request: Optional[Reference] = None
    note: Optional[List[Annotation]] = None
    dosage: Optional[Dict[str, Any]] = None

    def to_fhir_medication_administration(self) -> MedicationAdministration:
        """Convert to a FHIR MedicationAdministration."""
        data = self.model_dump(exclude_unset=True)
        # Convert status to string for the shared model
        if isinstance(data.get('status'), MedicationAdministrationStatus):
            data['status'] = data['status'].value
        return MedicationAdministration(**data)

class MedicationStatementCreate(BaseModel):
    """Model for creating a medication statement"""
    status: MedicationStatementStatus
    medicationCodeableConcept: Optional[CodeableConcept] = None
    medicationReference: Optional[Reference] = None
    subject: Reference
    context: Optional[Reference] = None
    effectiveDateTime: Optional[str] = None
    effectivePeriod: Optional[Dict[str, str]] = None
    dateAsserted: Optional[str] = None
    informationSource: Optional[Reference] = None
    derivedFrom: Optional[List[Reference]] = None
    reasonCode: Optional[List[CodeableConcept]] = None
    reasonReference: Optional[List[Reference]] = None
    note: Optional[List[Annotation]] = None
    dosage: Optional[List[DosageInstruction]] = None

    def to_fhir_medication_statement(self) -> MedicationStatement:
        """Convert to a FHIR MedicationStatement."""
        data = self.model_dump(exclude_unset=True)
        # Convert status to string for the shared model
        if isinstance(data.get('status'), MedicationStatementStatus):
            data['status'] = data['status'].value
        return MedicationStatement(**data)

# Update models
class MedicationUpdate(BaseModel):
    """Model for updating a medication"""
    status: Optional[MedicationStatus] = None
    code: Optional[CodeableConcept] = None
    manufacturer: Optional[Reference] = None
    form: Optional[CodeableConcept] = None
    amount: Optional[Ratio] = None
    ingredient: Optional[List[Dict[str, Any]]] = None
    batch: Optional[Dict[str, Any]] = None

    def to_fhir_medication_update(self) -> Dict[str, Any]:
        """Convert to a FHIR Medication update."""
        data = self.model_dump(exclude_unset=True)
        # Convert status to string if it's an enum
        if isinstance(data.get('status'), MedicationStatus):
            data['status'] = data['status'].value
        return data

class MedicationRequestUpdate(BaseModel):
    """Model for updating a medication request"""
    status: Optional[MedicationRequestStatus] = None
    intent: Optional[MedicationRequestIntent] = None
    medicationCodeableConcept: Optional[CodeableConcept] = None
    medicationReference: Optional[Reference] = None
    subject: Optional[Reference] = None
    encounter: Optional[Reference] = None
    authoredOn: Optional[str] = None
    requester: Optional[Reference] = None
    recorder: Optional[Reference] = None
    reasonCode: Optional[List[CodeableConcept]] = None
    reasonReference: Optional[List[Reference]] = None
    note: Optional[List[Annotation]] = None
    dosageInstruction: Optional[List[DosageInstruction]] = None
    dispenseRequest: Optional[Dict[str, Any]] = None
    substitution: Optional[Dict[str, Any]] = None
    priorPrescription: Optional[Reference] = None

    def to_fhir_medication_request_update(self) -> Dict[str, Any]:
        """Convert to a FHIR MedicationRequest update."""
        data = self.model_dump(exclude_unset=True)
        # Convert status and intent to strings if they're enums
        if isinstance(data.get('status'), MedicationRequestStatus):
            data['status'] = data['status'].value
        if isinstance(data.get('intent'), MedicationRequestIntent):
            data['intent'] = data['intent'].value
        return data

class MedicationAdministrationUpdate(BaseModel):
    """Model for updating a medication administration"""
    status: Optional[MedicationAdministrationStatus] = None
    medicationCodeableConcept: Optional[CodeableConcept] = None
    medicationReference: Optional[Reference] = None
    subject: Optional[Reference] = None
    context: Optional[Reference] = None
    effectiveDateTime: Optional[str] = None
    effectivePeriod: Optional[Dict[str, str]] = None
    performer: Optional[List[Dict[str, Any]]] = None
    reasonCode: Optional[List[CodeableConcept]] = None
    reasonReference: Optional[List[Reference]] = None
    request: Optional[Reference] = None
    note: Optional[List[Annotation]] = None
    dosage: Optional[Dict[str, Any]] = None

    def to_fhir_medication_administration_update(self) -> Dict[str, Any]:
        """Convert to a FHIR MedicationAdministration update."""
        data = self.model_dump(exclude_unset=True)
        # Convert status to string if it's an enum
        if isinstance(data.get('status'), MedicationAdministrationStatus):
            data['status'] = data['status'].value
        return data

class MedicationStatementUpdate(BaseModel):
    """Model for updating a medication statement"""
    status: Optional[MedicationStatementStatus] = None
    medicationCodeableConcept: Optional[CodeableConcept] = None
    medicationReference: Optional[Reference] = None
    subject: Optional[Reference] = None
    context: Optional[Reference] = None
    effectiveDateTime: Optional[str] = None
    effectivePeriod: Optional[Dict[str, str]] = None
    dateAsserted: Optional[str] = None
    informationSource: Optional[Reference] = None
    derivedFrom: Optional[List[Reference]] = None
    reasonCode: Optional[List[CodeableConcept]] = None
    reasonReference: Optional[List[Reference]] = None
    note: Optional[List[Annotation]] = None
    dosage: Optional[List[DosageInstruction]] = None

    def to_fhir_medication_statement_update(self) -> Dict[str, Any]:
        """Convert to a FHIR MedicationStatement update."""
        data = self.model_dump(exclude_unset=True)
        # Convert status to string if it's an enum
        if isinstance(data.get('status'), MedicationStatementStatus):
            data['status'] = data['status'].value
        return data
