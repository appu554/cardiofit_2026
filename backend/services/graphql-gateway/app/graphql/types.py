import strawberry
from typing import List, Optional, Dict, Any
from datetime import datetime

@strawberry.type
class AuthResponse:
    success: bool
    token: Optional[str] = None
    message: Optional[str] = None

@strawberry.type
class User:
    id: str
    email: str
    full_name: Optional[str] = None
    role: str
    is_active: bool
    created_at: datetime

@strawberry.type
class Identifier:
    system: str
    value: str
    use: Optional[str] = None

@strawberry.type
class HumanName:
    family: str
    given: List[str]
    use: Optional[str] = None
    prefix: Optional[List[str]] = None
    suffix: Optional[List[str]] = None

@strawberry.type
class ContactPoint:
    system: str
    value: str
    use: Optional[str] = None
    rank: Optional[int] = None

@strawberry.type
class Address:
    line: List[str]
    city: Optional[str] = None
    state: Optional[str] = None
    postalCode: Optional[str] = None
    country: Optional[str] = None
    use: Optional[str] = None
    type: Optional[str] = None

@strawberry.type
class Patient:
    id: str
    resourceType: str = "Patient"
    identifier: List[Identifier]
    name: List[HumanName]
    gender: Optional[str] = None
    birthDate: Optional[str] = None
    active: bool = True
    telecom: Optional[List[ContactPoint]] = None
    address: Optional[List[Address]] = None

@strawberry.type
class Coding:
    system: str
    code: str
    display: Optional[str] = None

@strawberry.type
class CodeableConcept:
    coding: List[Coding]
    text: Optional[str] = None

@strawberry.type
class Reference:
    reference: str
    display: Optional[str] = None

@strawberry.type
class Quantity:
    value: float
    unit: str
    system: Optional[str] = None
    code: Optional[str] = None

@strawberry.type
class Annotation:
    text: str
    authorString: Optional[str] = None
    time: Optional[str] = None

@strawberry.type
class ReferenceRange:
    low: Optional[Quantity] = None
    high: Optional[Quantity] = None
    text: Optional[str] = None

@strawberry.type
class ObservationCode:
    system: str
    code: str
    display: Optional[str] = None

@strawberry.type
class ObservationCodeEntry:
    id: str
    code: str
    system: str
    display: Optional[str] = None

@strawberry.type
class ObservationSubject:
    reference: str

@strawberry.type
class ObservationSubjectEntry:
    id: str
    reference: str

@strawberry.type
class ObservationValueQuantity:
    value: float
    unit: str
    system: Optional[str] = None
    code: Optional[str] = None

@strawberry.type
class ObservationValueEntry:
    id: str
    value: float
    unit: str
    system: Optional[str] = None
    code: Optional[str] = None

@strawberry.type
class ObservationInterpretation:
    system: str
    code: str
    display: Optional[str] = None

@strawberry.type
class ObservationInterpretationEntry:
    id: str
    system: str
    code: str
    display: Optional[str] = None

@strawberry.type
class ObservationReferenceRangeQuantity:
    value: float
    unit: str
    system: Optional[str] = None
    code: Optional[str] = None

@strawberry.type
class ObservationReferenceRange:
    low: Optional[ObservationReferenceRangeQuantity] = None
    high: Optional[ObservationReferenceRangeQuantity] = None
    text: Optional[str] = None

@strawberry.type
class ObservationReferenceRangeEntry:
    id: str
    low: Optional[ObservationReferenceRangeQuantity] = None
    high: Optional[ObservationReferenceRangeQuantity] = None
    text: Optional[str] = None

@strawberry.type
class LabResult:
    id: str
    _id: Optional[str] = None
    status: str
    category: str
    code: Optional[ObservationCode] = None
    subject: Optional[ObservationSubject] = None
    effective_datetime: Optional[str] = strawberry.field(name="effectiveDateTime", default=None)
    value_quantity: Optional[ObservationValueQuantity] = strawberry.field(name="valueQuantity", default=None)
    interpretation: Optional[List[ObservationInterpretation]] = None
    reference_range: Optional[List[ObservationReferenceRange]] = strawberry.field(name="referenceRange", default=None)

@strawberry.type
class VitalSign(LabResult):
    """Vital signs observation type"""
    pass

@strawberry.type
class PhysicalMeasurement(LabResult):
    """Physical measurement observation type"""
    pass

@strawberry.type
class CompleteObservation:
    """Complete observation with all fields"""
    id: str
    status: str
    category: str
    type: str
    code: Optional[ObservationCode] = None
    subject: Optional[ObservationSubject] = None
    effective_datetime: Optional[str] = strawberry.field(name="effectiveDateTime", default=None)
    value_quantity: Optional[ObservationValueQuantity] = strawberry.field(name="valueQuantity", default=None)
    interpretation: Optional[List[ObservationInterpretation]] = None
    reference_range: Optional[List[ObservationReferenceRange]] = strawberry.field(name="referenceRange", default=None)

@strawberry.input
class ObservationCodeInput:
    system: str
    code: str
    display: Optional[str] = None

@strawberry.input
class ObservationSubjectInput:
    reference: str

@strawberry.input
class ObservationValueQuantityInput:
    value: float
    unit: str
    system: Optional[str] = None
    code: Optional[str] = None

@strawberry.input
class ObservationInterpretationInput:
    system: str
    code: str
    display: Optional[str] = None

@strawberry.input
class ObservationReferenceRangeQuantityInput:
    value: float
    unit: str
    system: Optional[str] = None
    code: Optional[str] = None

@strawberry.input
class ObservationReferenceRangeInput:
    low: Optional[ObservationReferenceRangeQuantityInput] = None
    high: Optional[ObservationReferenceRangeQuantityInput] = None
    text: Optional[str] = None

@strawberry.input
class CreateObservationInput:
    status: str
    category: str
    code: ObservationCodeInput
    subject: ObservationSubjectInput
    effective_datetime: Optional[str] = strawberry.field(name="effectiveDatetime", default=None)
    value_quantity: Optional[ObservationValueQuantityInput] = strawberry.field(name="valueQuantity", default=None)
    interpretation: Optional[List[ObservationInterpretationInput]] = None
    reference_range: Optional[List[ObservationReferenceRangeInput]] = strawberry.field(name="referenceRange", default=None)

@strawberry.input
class UpdateObservationInput:
    id: str
    status: Optional[str] = None
    category: Optional[str] = None
    code: Optional[ObservationCodeInput] = None
    subject: Optional[ObservationSubjectInput] = None
    effective_datetime: Optional[str] = strawberry.field(name="effectiveDatetime", default=None)
    value_quantity: Optional[ObservationValueQuantityInput] = strawberry.field(name="valueQuantity", default=None)
    interpretation: Optional[List[ObservationInterpretationInput]] = None
    reference_range: Optional[List[ObservationReferenceRangeInput]] = strawberry.field(name="referenceRange", default=None)

@strawberry.input
class CreateVitalSignInput:
    status: str
    code: ObservationCodeInput
    subject: ObservationSubjectInput
    effective_datetime: Optional[str] = strawberry.field(name="effectiveDatetime", default=None)
    value_quantity: Optional[ObservationValueQuantityInput] = strawberry.field(name="valueQuantity", default=None)
    interpretation: Optional[List[ObservationInterpretationInput]] = None
    reference_range: Optional[List[ObservationReferenceRangeInput]] = strawberry.field(name="referenceRange", default=None)

@strawberry.input
class UpdateVitalSignInput:
    id: str
    status: Optional[str] = None
    code: Optional[ObservationCodeInput] = None
    subject: Optional[ObservationSubjectInput] = None
    effective_datetime: Optional[str] = strawberry.field(name="effectiveDatetime", default=None)
    value_quantity: Optional[ObservationValueQuantityInput] = strawberry.field(name="valueQuantity", default=None)
    interpretation: Optional[List[ObservationInterpretationInput]] = None
    reference_range: Optional[List[ObservationReferenceRangeInput]] = strawberry.field(name="referenceRange", default=None)

@strawberry.input
class CreatePhysicalMeasurementInput:
    status: str
    code: ObservationCodeInput
    subject: ObservationSubjectInput
    effective_datetime: Optional[str] = strawberry.field(name="effectiveDatetime", default=None)
    value_quantity: Optional[ObservationValueQuantityInput] = strawberry.field(name="valueQuantity", default=None)
    interpretation: Optional[List[ObservationInterpretationInput]] = None
    reference_range: Optional[List[ObservationReferenceRangeInput]] = strawberry.field(name="referenceRange", default=None)

@strawberry.input
class UpdatePhysicalMeasurementInput:
    id: str
    status: Optional[str] = None
    code: Optional[ObservationCodeInput] = None
    subject: Optional[ObservationSubjectInput] = None
    effective_datetime: Optional[str] = strawberry.field(name="effectiveDatetime", default=None)
    value_quantity: Optional[ObservationValueQuantityInput] = strawberry.field(name="valueQuantity", default=None)
    interpretation: Optional[List[ObservationInterpretationInput]] = None
    reference_range: Optional[List[ObservationReferenceRangeInput]] = strawberry.field(name="referenceRange", default=None)

@strawberry.type
class Condition:
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

@strawberry.type
class ProblemListItem(Condition):
    """Problem list item condition type"""
    pass

@strawberry.type
class Diagnosis(Condition):
    """Encounter diagnosis condition type"""
    pass

@strawberry.type
class HealthConcern(Condition):
    """Health concern condition type"""
    pass

@strawberry.type
class DosageInstruction:
    text: Optional[str] = None
    timing: Optional[str] = None
    asNeededBoolean: Optional[bool] = None
    route: Optional[CodeableConcept] = None

@strawberry.type
class MedicationRequest:
    id: str
    resourceType: str = "MedicationRequest"
    status: str
    intent: str
    medicationCodeableConcept: CodeableConcept
    subject: Reference
    authoredOn: str
    requester: Optional[Reference] = None
    dosageInstruction: Optional[List[DosageInstruction]] = None
    note: Optional[List[Annotation]] = None

@strawberry.type
class Attachment:
    contentType: Optional[str] = None
    data: Optional[str] = None
    url: Optional[str] = None
    title: Optional[str] = None

@strawberry.type
class DiagnosticReport:
    id: str
    resourceType: str = "DiagnosticReport"
    status: str
    category: Optional[List[CodeableConcept]] = None
    code: CodeableConcept
    subject: Reference
    effectiveDateTime: str
    issued: Optional[str] = None
    performer: Optional[List[Reference]] = None
    result: Optional[List[Reference]] = None
    conclusion: Optional[str] = None
    presentedForm: Optional[List[Attachment]] = None

@strawberry.type
class Period:
    start: Optional[str] = None
    end: Optional[str] = None

@strawberry.type
class EncounterParticipant:
    type: Optional[List[CodeableConcept]] = None
    period: Optional[Period] = None
    individual: Optional[Reference] = None

@strawberry.type
class EncounterDiagnosis:
    condition: Reference
    use: Optional[CodeableConcept] = None
    rank: Optional[int] = None

@strawberry.type
class EncounterLocation:
    location: Reference
    status: Optional[str] = None
    period: Optional[Period] = None

@strawberry.type
class Meta:
    versionId: Optional[str] = None
    lastUpdated: Optional[str] = None
    source: Optional[str] = None
    profile: Optional[List[str]] = None
    security: Optional[List[CodeableConcept]] = None
    tag: Optional[List[CodeableConcept]] = None

@strawberry.type
class Encounter:
    id: str
    resourceType: str = "Encounter"
    status: str
    class_: CodeableConcept = strawberry.field(name="class")
    type: Optional[List[CodeableConcept]] = None
    subject: Reference
    participant: Optional[List[EncounterParticipant]] = None
    period: Optional[Period] = None
    reasonCode: Optional[List[CodeableConcept]] = None
    diagnosis: Optional[List[EncounterDiagnosis]] = None
    location: Optional[List[EncounterLocation]] = None
    serviceProvider: Optional[Reference] = None
    meta: Optional[Meta] = None

@strawberry.type
class DocumentContent:
    attachment: Attachment
    format: Optional[CodeableConcept] = None

@strawberry.type
class DocumentContext:
    encounter: Optional[List[Reference]] = None
    period: Optional[Period] = None

@strawberry.type
class DocumentReference:
    id: str
    resourceType: str = "DocumentReference"
    status: str
    docStatus: Optional[str] = None
    type: CodeableConcept
    subject: Reference
    date: str
    author: Optional[List[Reference]] = None
    content: List[DocumentContent]
    context: Optional[DocumentContext] = None

@strawberry.type
class Note:
    id: str
    patient_id: str
    title: str
    content: str
    created_at: str
    updated_at: Optional[str] = None
    author: Optional[str] = None
    tags: Optional[List[str]] = None

@strawberry.type
class EventDetails:
    code: Optional[str] = None
    value: Optional[str] = None
    unit: Optional[str] = None
    display: Optional[str] = None

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
    details: Optional[EventDetails] = None

@strawberry.type
class PatientTimeline:
    patient_id: str
    events: List[TimelineEvent]

@strawberry.type
class Medication:
    id: str
    resourceType: str = "Medication"
    status: Optional[str] = None
    code: CodeableConcept
    form: Optional[CodeableConcept] = None
    amount: Optional[Quantity] = None
    ingredient: Optional[List[str]] = None
    batch: Optional[str] = None

@strawberry.type
class DosageInstruction:
    text: Optional[str] = None
    timing: Optional[str] = None
    asNeededBoolean: Optional[bool] = None
    route: Optional[CodeableConcept] = None
    doseAndRate: Optional[List[Quantity]] = None

@strawberry.type
class MedicationRequest:
    id: str
    resourceType: str = "MedicationRequest"
    status: str
    intent: str
    medicationCodeableConcept: CodeableConcept
    subject: Reference
    authoredOn: str
    requester: Optional[Reference] = None
    dosageInstruction: Optional[List[DosageInstruction]] = None
    note: Optional[List[Annotation]] = None

@strawberry.type
class MedicationAdministration:
    id: str
    resourceType: str = "MedicationAdministration"
    status: str
    medicationCodeableConcept: CodeableConcept
    subject: Reference
    effectiveDateTime: str
    performer: Optional[List[Reference]] = None
    request: Optional[Reference] = None
    dosage: Optional[str] = None
    note: Optional[List[Annotation]] = None

@strawberry.type
class MedicationStatement:
    id: str
    resourceType: str = "MedicationStatement"
    status: str
    medicationCodeableConcept: CodeableConcept
    subject: Reference
    effectiveDateTime: Optional[str] = None
    dateAsserted: Optional[str] = None
    informationSource: Optional[Reference] = None
    dosage: Optional[List[DosageInstruction]] = None
    note: Optional[List[Annotation]] = None
