import strawberry
from typing import List, Optional

@strawberry.input
class IdentifierInput:
    system: str
    value: str
    use: Optional[str] = None

@strawberry.input
class HumanNameInput:
    family: str
    given: List[str]
    use: Optional[str] = None
    prefix: Optional[List[str]] = None
    suffix: Optional[List[str]] = None

@strawberry.input
class ContactPointInput:
    system: str
    value: str
    use: Optional[str] = None
    rank: Optional[int] = None

@strawberry.input
class AddressInput:
    line: List[str]
    city: Optional[str] = None
    state: Optional[str] = None
    postalCode: Optional[str] = None
    country: Optional[str] = None
    use: Optional[str] = None
    type: Optional[str] = None

@strawberry.input
class PatientInput:
    identifier: List[IdentifierInput]
    name: List[HumanNameInput]
    gender: Optional[str] = None
    birthDate: Optional[str] = None
    active: bool = True
    telecom: Optional[List[ContactPointInput]] = None
    address: Optional[List[AddressInput]] = None

@strawberry.input
class CodingInput:
    system: str
    code: str
    display: Optional[str] = None

@strawberry.input
class CodeableConceptInput:
    coding: List[CodingInput]
    text: Optional[str] = None

@strawberry.input
class ReferenceInput:
    reference: str
    display: Optional[str] = None

@strawberry.input
class QuantityInput:
    value: float
    unit: str
    system: Optional[str] = None
    code: Optional[str] = None

@strawberry.input
class AnnotationInput:
    text: str
    authorString: Optional[str] = None
    time: Optional[str] = None

@strawberry.input
class ObservationInput:
    status: str
    category: List[CodeableConceptInput]
    code: CodeableConceptInput
    subject: ReferenceInput
    effectiveDateTime: str
    valueQuantity: Optional[QuantityInput] = None
    valueString: Optional[str] = None
    valueCodeableConcept: Optional[CodeableConceptInput] = None
    interpretation: Optional[List[CodeableConceptInput]] = None
    note: Optional[List[AnnotationInput]] = None

@strawberry.input
class ConditionInput:
    clinicalStatus: Optional[CodeableConceptInput] = None
    verificationStatus: Optional[CodeableConceptInput] = None
    category: Optional[List[CodeableConceptInput]] = None
    code: CodeableConceptInput
    subject: ReferenceInput
    onsetDateTime: Optional[str] = None
    abatementDateTime: Optional[str] = None
    recordedDate: Optional[str] = None
    note: Optional[List[AnnotationInput]] = None

@strawberry.input
class DosageInstructionInput:
    text: Optional[str] = None
    timing: Optional[str] = None
    asNeededBoolean: Optional[bool] = None
    route: Optional[CodeableConceptInput] = None

@strawberry.input
class MedicationRequestInput:
    status: str
    intent: str
    medicationCodeableConcept: CodeableConceptInput
    subject: ReferenceInput
    authoredOn: str
    requester: Optional[ReferenceInput] = None
    dosageInstruction: Optional[List[DosageInstructionInput]] = None
    note: Optional[List[AnnotationInput]] = None

@strawberry.input
class AttachmentInput:
    contentType: Optional[str] = None
    data: Optional[str] = None
    url: Optional[str] = None
    title: Optional[str] = None

@strawberry.input
class DiagnosticReportInput:
    status: str
    category: Optional[List[CodeableConceptInput]] = None
    code: CodeableConceptInput
    subject: ReferenceInput
    effectiveDateTime: str
    issued: Optional[str] = None
    performer: Optional[List[ReferenceInput]] = None
    result: Optional[List[ReferenceInput]] = None
    conclusion: Optional[str] = None
    presentedForm: Optional[List[AttachmentInput]] = None

@strawberry.input
class PeriodInput:
    start: Optional[str] = None
    end: Optional[str] = None

@strawberry.input
class EncounterParticipantInput:
    type: Optional[List[CodeableConceptInput]] = None
    period: Optional[PeriodInput] = None
    individual: Optional[ReferenceInput] = None

@strawberry.input
class EncounterDiagnosisInput:
    condition: ReferenceInput
    use: Optional[CodeableConceptInput] = None
    rank: Optional[int] = None

@strawberry.input
class EncounterLocationInput:
    location: ReferenceInput
    status: Optional[str] = None
    period: Optional[PeriodInput] = None

@strawberry.input
class EncounterInput:
    status: str
    class_: CodeableConceptInput = strawberry.field(name="class")
    type: Optional[List[CodeableConceptInput]] = None
    subject: ReferenceInput
    participant: Optional[List[EncounterParticipantInput]] = None
    period: Optional[PeriodInput] = None
    reasonCode: Optional[List[CodeableConceptInput]] = None
    diagnosis: Optional[List[EncounterDiagnosisInput]] = None
    location: Optional[List[EncounterLocationInput]] = None
    serviceProvider: Optional[ReferenceInput] = None

@strawberry.input
class DocumentContentInput:
    attachment: AttachmentInput
    format: Optional[CodeableConceptInput] = None

@strawberry.input
class DocumentContextInput:
    encounter: Optional[List[ReferenceInput]] = None
    period: Optional[PeriodInput] = None

@strawberry.input
class DocumentReferenceInput:
    status: str
    docStatus: Optional[str] = None
    type: CodeableConceptInput
    subject: ReferenceInput
    date: str
    author: Optional[List[ReferenceInput]] = None
    content: List[DocumentContentInput]
    context: Optional[DocumentContextInput] = None

@strawberry.input
class NoteInput:
    patient_id: str
    title: str
    content: str
    author: Optional[str] = None
    tags: Optional[List[str]] = None

@strawberry.input
class DosageInstructionInput:
    text: Optional[str] = None
    timing: Optional[str] = None
    asNeededBoolean: Optional[bool] = None
    route: Optional[CodeableConceptInput] = None
    doseAndRate: Optional[List[QuantityInput]] = None

@strawberry.input
class MedicationInput:
    status: Optional[str] = None
    code: CodeableConceptInput
    form: Optional[CodeableConceptInput] = None
    amount: Optional[QuantityInput] = None
    ingredient: Optional[List[str]] = None
    batch: Optional[str] = None

@strawberry.input
class MedicationRequestInput:
    status: str
    intent: str
    medicationCodeableConcept: CodeableConceptInput
    subject: ReferenceInput
    authoredOn: str
    requester: Optional[ReferenceInput] = None
    dosageInstruction: Optional[List[DosageInstructionInput]] = None
    note: Optional[List[AnnotationInput]] = None

@strawberry.input
class MedicationAdministrationInput:
    status: str
    medicationCodeableConcept: CodeableConceptInput
    subject: ReferenceInput
    effectiveDateTime: str
    performer: Optional[List[ReferenceInput]] = None
    request: Optional[ReferenceInput] = None
    dosage: Optional[str] = None
    note: Optional[List[AnnotationInput]] = None

@strawberry.input
class MedicationStatementInput:
    status: str
    medicationCodeableConcept: CodeableConceptInput
    subject: ReferenceInput
    effectiveDateTime: Optional[str] = None
    dateAsserted: Optional[str] = None
    informationSource: Optional[ReferenceInput] = None
    dosage: Optional[List[DosageInstructionInput]] = None
    note: Optional[List[AnnotationInput]] = None
