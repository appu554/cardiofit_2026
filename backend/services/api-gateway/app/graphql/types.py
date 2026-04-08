import strawberry
from datetime import datetime
from typing import List, Optional, Dict, Union
from strawberry.scalars import JSON

@strawberry.type
class User:
    id: str
    email: str
    full_name: Optional[str] = None
    role: str
    is_active: bool
    created_at: datetime

@strawberry.type
class EventDetails:
    key: str
    value: str

@strawberry.type
class TimelineEvent:
    id: str
    patient_id: str
    event_type: str
    resource_type: str
    resource_id: str
    title: str
    description: Optional[str] = None
    date: str
    details: Optional[List[EventDetails]] = None

@strawberry.type
class PatientTimeline:
    patient_id: str
    events: List[TimelineEvent]

@strawberry.type
class Patient:
    id: str
    resourceType: str = "Patient"
    identifier: Optional[List["Identifier"]] = None
    name: Optional[List["HumanName"]] = None
    gender: Optional[str] = None
    birthDate: Optional[str] = None
    active: bool = True
    telecom: Optional[List["ContactPoint"]] = None
    address: Optional[List["Address"]] = None
    maritalStatus: Optional[strawberry.scalars.JSON] = None
    communication: Optional[List[strawberry.scalars.JSON]] = None
    generalPractitioner: Optional[List["Reference"]] = None
    managingOrganization: Optional["Reference"] = None
    # Generic field to hold any additional FHIR fields
    extension: Optional[strawberry.scalars.JSON] = None

@strawberry.type
class Identifier:
    system: Optional[str] = None
    value: Optional[str] = None
    use: Optional[str] = None

@strawberry.type
class HumanName:
    family: Optional[str] = None
    given: Optional[List[str]] = None
    use: Optional[str] = None
    prefix: Optional[List[str]] = None
    suffix: Optional[List[str]] = None
    text: Optional[str] = None

@strawberry.type
class ContactPoint:
    system: Optional[str] = None
    value: Optional[str] = None
    use: Optional[str] = None
    rank: Optional[int] = None

@strawberry.type
class Address:
    line: Optional[List[str]] = None
    city: Optional[str] = None
    state: Optional[str] = None
    postalCode: Optional[str] = None
    country: Optional[str] = None
    use: Optional[str] = None

@strawberry.type
class Note:
    id: str
    patient_id: str
    title: str
    content: str
    note_type: str
    author_id: str
    created_at: datetime
    updated_at: datetime

@strawberry.type
class LabResult:
    id: str
    resource_type: str = "Observation"
    status: str
    category: List["CodeableConcept"]
    code: "CodeableConcept"
    subject: "Reference"
    effective_date_time: Optional[str] = None
    value_quantity: Optional["Quantity"] = None
    value_string: Optional[str] = None
    value_boolean: Optional[bool] = None
    value_codeable_concept: Optional["CodeableConcept"] = None
    interpretation: Optional[List["CodeableConcept"]] = None

@strawberry.type
class CodeableConcept:
    coding: List["Coding"]
    text: Optional[str] = None

@strawberry.type
class Coding:
    system: str
    code: str
    display: Optional[str] = None

@strawberry.type
class Reference:
    reference: str
    display: Optional[str] = None

@strawberry.type
class Quantity:
    value: float
    unit: Optional[str] = None
    system: Optional[str] = None
    code: Optional[str] = None

@strawberry.type
class Period:
    start: Optional[str] = None
    end: Optional[str] = None

@strawberry.type
class Condition:
    id: str
    resource_type: str = "Condition"
    clinical_status: Optional["CodeableConcept"] = None
    verification_status: Optional["CodeableConcept"] = None
    category: Optional[List["CodeableConcept"]] = None
    severity: Optional["CodeableConcept"] = None
    code: "CodeableConcept"
    subject: "Reference"
    onset_date_time: Optional[str] = None
    onset_period: Optional["Period"] = None
    abatement_date_time: Optional[str] = None
    abatement_period: Optional["Period"] = None

@strawberry.type
class DosageInstruction:
    text: Optional[str] = None
    timing: Optional[str] = None
    dose_quantity: Optional[Quantity] = None
    route: Optional[CodeableConcept] = None
    method: Optional[CodeableConcept] = None

@strawberry.type
class MedicationRequest:
    id: str
    resource_type: str = "MedicationRequest"
    status: str
    intent: str
    medication_codeable_concept: Optional["CodeableConcept"] = None
    medication_reference: Optional["Reference"] = None
    subject: "Reference"
    authored_on: Optional[str] = None
    requester: Optional["Reference"] = None
    dosage_instruction: Optional[List[DosageInstruction]] = None

@strawberry.type
class DiagnosticReport:
    id: str
    resource_type: str = "DiagnosticReport"
    status: str
    category: Optional[List["CodeableConcept"]] = None
    code: "CodeableConcept"
    subject: "Reference"
    effective_date_time: Optional[str] = None
    issued: Optional[str] = None
    performer: Optional[List["Reference"]] = None
    result: Optional[List["Reference"]] = None
    conclusion: Optional[str] = None

@strawberry.type
class EncounterParticipant:
    type: Optional[List[CodeableConcept]] = None
    individual: Optional[Reference] = None
    period: Optional[Period] = None

@strawberry.type
class Encounter:
    id: str
    resource_type: str = "Encounter"
    status: str
    class_field: str = strawberry.field(name="class")
    type: Optional[List["CodeableConcept"]] = None
    subject: "Reference"
    participant: Optional[List[EncounterParticipant]] = None
    period: Optional["Period"] = None
    reason_code: Optional[List["CodeableConcept"]] = None

@strawberry.type
class Attachment:
    content_type: Optional[str] = None
    language: Optional[str] = None
    data: Optional[str] = None
    url: Optional[str] = None
    size: Optional[int] = None
    hash: Optional[str] = None
    title: Optional[str] = None
    creation: Optional[str] = None

@strawberry.type
class DocumentReferenceContent:
    attachment: Attachment
    format: Optional[str] = None

@strawberry.type
class DocumentReference:
    id: str
    resource_type: str = "DocumentReference"
    status: str
    type: Optional["CodeableConcept"] = None
    category: Optional[List["CodeableConcept"]] = None
    subject: "Reference"
    date: Optional[str] = None
    author: Optional[List["Reference"]] = None
    content: List[DocumentReferenceContent]

@strawberry.type
class AuthResponse:
    success: bool
    token: Optional[str] = None
    message: Optional[str] = None