import strawberry
from typing import List, Optional
from strawberry.scalars import JSON

@strawberry.input
class IdentifierInput:
    system: str
    value: str
    use: Optional[str] = None
    type: Optional[JSON] = None
    period: Optional[JSON] = None
    assigner: Optional[JSON] = None

@strawberry.input
class HumanNameInput:
    family: Optional[str] = None
    given: Optional[List[str]] = None
    use: Optional[str] = None
    prefix: Optional[List[str]] = None
    suffix: Optional[List[str]] = None
    text: Optional[str] = None
    period: Optional[JSON] = None

@strawberry.input
class ContactPointInput:
    system: Optional[str] = None
    value: Optional[str] = None
    use: Optional[str] = None
    rank: Optional[int] = None
    period: Optional[JSON] = None

@strawberry.input
class AddressInput:
    line: Optional[List[str]] = None
    city: Optional[str] = None
    state: Optional[str] = None
    postalCode: Optional[str] = None
    country: Optional[str] = None
    use: Optional[str] = None
    type: Optional[str] = None
    text: Optional[str] = None
    period: Optional[JSON] = None

# PatientInput moved after ReferenceInput

@strawberry.input
class CodingInput:
    system: Optional[str] = None
    code: Optional[str] = None
    display: Optional[str] = None
    version: Optional[str] = None
    userSelected: Optional[bool] = None

@strawberry.input
class CodeableConceptInput:
    coding: Optional[List[CodingInput]] = None
    text: Optional[str] = None

@strawberry.input
class ReferenceInput:
    reference: Optional[str] = None
    display: Optional[str] = None
    type: Optional[str] = None
    identifier: Optional[IdentifierInput] = None

@strawberry.input
class PatientInput:
    # Make all fields optional to allow for more flexibility
    identifier: Optional[List[IdentifierInput]] = None
    name: Optional[List[HumanNameInput]] = None
    gender: Optional[str] = None
    birthDate: Optional[str] = None
    active: Optional[bool] = None
    telecom: Optional[List[ContactPointInput]] = None
    address: Optional[List[AddressInput]] = None
    # Allow for additional fields
    resourceType: Optional[str] = None
    text: Optional[JSON] = None
    maritalStatus: Optional[JSON] = None
    deceasedBoolean: Optional[bool] = None
    multipleBirthBoolean: Optional[bool] = None
    contact: Optional[List[JSON]] = None
    communication: Optional[List[JSON]] = None
    generalPractitioner: Optional[List[ReferenceInput]] = None
    managingOrganization: Optional[ReferenceInput] = None
    # Generic field to allow any other fields
    extension: Optional[JSON] = None

@strawberry.input
class QuantityInput:
    value: Optional[float] = None
    unit: Optional[str] = None
    system: Optional[str] = None
    code: Optional[str] = None
    comparator: Optional[str] = None

@strawberry.input
class PeriodInput:
    start: Optional[str] = None
    end: Optional[str] = None

@strawberry.input
class ObservationInput:
    status: Optional[str] = None
    category: Optional[List[CodeableConceptInput]] = None
    code: Optional[CodeableConceptInput] = None
    subject: Optional[ReferenceInput] = None
    effectiveDateTime: Optional[str] = None
    effectivePeriod: Optional[PeriodInput] = None
    issued: Optional[str] = None
    valueQuantity: Optional[QuantityInput] = None
    valueString: Optional[str] = None
    valueBoolean: Optional[bool] = None
    valueCodeableConcept: Optional[CodeableConceptInput] = None
    valueInteger: Optional[int] = None
    valueRange: Optional[JSON] = None
    valueRatio: Optional[JSON] = None
    valueSampledData: Optional[JSON] = None
    valueTime: Optional[str] = None
    valueDateTime: Optional[str] = None
    valuePeriod: Optional[PeriodInput] = None
    interpretation: Optional[List[CodeableConceptInput]] = None
    note: Optional[List[JSON]] = None
    bodySite: Optional[CodeableConceptInput] = None
    method: Optional[CodeableConceptInput] = None
    specimen: Optional[ReferenceInput] = None
    device: Optional[ReferenceInput] = None
    referenceRange: Optional[List[JSON]] = None
    hasMember: Optional[List[ReferenceInput]] = None
    derivedFrom: Optional[List[ReferenceInput]] = None
    component: Optional[List[JSON]] = None

@strawberry.input
class ConditionInput:
    clinicalStatus: Optional[CodeableConceptInput] = None
    verificationStatus: Optional[CodeableConceptInput] = None
    category: Optional[List[CodeableConceptInput]] = None
    severity: Optional[CodeableConceptInput] = None
    code: Optional[CodeableConceptInput] = None
    subject: Optional[ReferenceInput] = None
    onsetDateTime: Optional[str] = None
    onsetPeriod: Optional[PeriodInput] = None
    onsetString: Optional[str] = None
    onsetAge: Optional[JSON] = None
    onsetRange: Optional[JSON] = None
    abatementDateTime: Optional[str] = None
    abatementPeriod: Optional[PeriodInput] = None
    abatementString: Optional[str] = None
    abatementAge: Optional[JSON] = None
    abatementRange: Optional[JSON] = None
    abatementBoolean: Optional[bool] = None
    recordedDate: Optional[str] = None
    recorder: Optional[ReferenceInput] = None
    asserter: Optional[ReferenceInput] = None
    stage: Optional[List[JSON]] = None
    evidence: Optional[List[JSON]] = None
    note: Optional[List[JSON]] = None

@strawberry.input
class DosageInstructionInput:
    text: Optional[str] = None
    timing: Optional[str] = None
    doseQuantity: Optional[QuantityInput] = None
    route: Optional[CodeableConceptInput] = None
    method: Optional[CodeableConceptInput] = None

@strawberry.input
class MedicationRequestInput:
    status: Optional[str] = None
    intent: Optional[str] = None
    medicationCodeableConcept: Optional[CodeableConceptInput] = None
    medicationReference: Optional[ReferenceInput] = None
    subject: Optional[ReferenceInput] = None
    encounter: Optional[ReferenceInput] = None
    supportingInformation: Optional[List[ReferenceInput]] = None
    authoredOn: Optional[str] = None
    requester: Optional[ReferenceInput] = None
    performer: Optional[ReferenceInput] = None
    performerType: Optional[CodeableConceptInput] = None
    recorder: Optional[ReferenceInput] = None
    reasonCode: Optional[List[CodeableConceptInput]] = None
    reasonReference: Optional[List[ReferenceInput]] = None
    basedOn: Optional[List[ReferenceInput]] = None
    groupIdentifier: Optional[IdentifierInput] = None
    courseOfTherapyType: Optional[CodeableConceptInput] = None
    insurance: Optional[List[ReferenceInput]] = None
    note: Optional[List[JSON]] = None
    dosageInstruction: Optional[List[DosageInstructionInput]] = None
    dispenseRequest: Optional[JSON] = None
    substitution: Optional[JSON] = None
    priorPrescription: Optional[ReferenceInput] = None
    detectedIssue: Optional[List[ReferenceInput]] = None
    eventHistory: Optional[List[ReferenceInput]] = None

@strawberry.input
class DiagnosticReportInput:
    status: Optional[str] = None
    category: Optional[List[CodeableConceptInput]] = None
    code: Optional[CodeableConceptInput] = None
    subject: Optional[ReferenceInput] = None
    encounter: Optional[ReferenceInput] = None
    effectiveDateTime: Optional[str] = None
    effectivePeriod: Optional[PeriodInput] = None
    issued: Optional[str] = None
    performer: Optional[List[ReferenceInput]] = None
    resultsInterpreter: Optional[List[ReferenceInput]] = None
    specimen: Optional[List[ReferenceInput]] = None
    result: Optional[List[ReferenceInput]] = None
    imagingStudy: Optional[List[ReferenceInput]] = None
    media: Optional[List[JSON]] = None
    conclusion: Optional[str] = None
    conclusionCode: Optional[List[CodeableConceptInput]] = None
    presentedForm: Optional[List[JSON]] = None

@strawberry.input
class EncounterParticipantInput:
    type: Optional[List[CodeableConceptInput]] = None
    individual: Optional[ReferenceInput] = None
    period: Optional[PeriodInput] = None

@strawberry.input
class EncounterInput:
    status: Optional[str] = None
    class_: Optional[str] = None
    type: Optional[List[CodeableConceptInput]] = None
    serviceType: Optional[CodeableConceptInput] = None
    priority: Optional[CodeableConceptInput] = None
    subject: Optional[ReferenceInput] = None
    episodeOfCare: Optional[List[ReferenceInput]] = None
    basedOn: Optional[List[ReferenceInput]] = None
    participant: Optional[List[EncounterParticipantInput]] = None
    appointment: Optional[List[ReferenceInput]] = None
    period: Optional[PeriodInput] = None
    length: Optional[QuantityInput] = None
    reasonCode: Optional[List[CodeableConceptInput]] = None
    reasonReference: Optional[List[ReferenceInput]] = None
    diagnosis: Optional[List[JSON]] = None
    account: Optional[List[ReferenceInput]] = None
    hospitalization: Optional[JSON] = None
    location: Optional[List[JSON]] = None
    serviceProvider: Optional[ReferenceInput] = None
    partOf: Optional[ReferenceInput] = None

@strawberry.input
class AttachmentInput:
    contentType: Optional[str] = None
    language: Optional[str] = None
    data: Optional[str] = None
    url: Optional[str] = None
    size: Optional[int] = None
    hash: Optional[str] = None
    title: Optional[str] = None
    creation: Optional[str] = None

@strawberry.input
class DocumentReferenceContentInput:
    attachment: Optional[AttachmentInput] = None
    format: Optional[str] = None

@strawberry.input
class DocumentReferenceInput:
    status: Optional[str] = None
    docStatus: Optional[str] = None
    type: Optional[CodeableConceptInput] = None
    category: Optional[List[CodeableConceptInput]] = None
    subject: Optional[ReferenceInput] = None
    date: Optional[str] = None
    author: Optional[List[ReferenceInput]] = None
    authenticator: Optional[ReferenceInput] = None
    custodian: Optional[ReferenceInput] = None
    relatesTo: Optional[List[JSON]] = None
    description: Optional[str] = None
    securityLabel: Optional[List[CodeableConceptInput]] = None
    content: Optional[List[DocumentReferenceContentInput]] = None
    context: Optional[JSON] = None

@strawberry.input
class NoteInput:
    patient_id: str
    title: str
    content: str
    note_type: str
