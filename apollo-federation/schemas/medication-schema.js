const { gql } = require('apollo-server-express');

const typeDefs = gql`
  extend schema @link(url: "https://specs.apollo.dev/federation/v2.0", import: ["@key", "@shareable"])

  # Extend Patient type to add medication-related fields
  extend type Patient @key(fields: "id") {
    id: ID! @external
    medications: [MedicationRequest]
    medicationStatements: [MedicationStatement]
    medicationAdministrations: [MedicationAdministration]
    allergies: [AllergyIntolerance]
  }

  type Query {
    medications(page: Int, limit: Int, status: String): MedicationConnection
    medication(id: String!): Medication
    medicationRequests(page: Int, limit: Int, status: String, patient: String): MedicationRequestConnection
    medicationRequest(id: String!): MedicationRequest
    medicationStatements(page: Int, limit: Int, patient: String): MedicationStatementConnection
    medicationStatement(id: String!): MedicationStatement
    medicationAdministrations(page: Int, limit: Int, patient: String): MedicationAdministrationConnection
    medicationAdministration(id: String!): MedicationAdministration
    allergies(page: Int, limit: Int, patient: String): AllergyIntoleranceConnection
    allergy(id: String!): AllergyIntolerance
  }

  type Mutation {
    createMedication(medicationData: MedicationInput!): Medication
    updateMedication(id: String!, medicationData: MedicationInput!): Medication
    deleteMedication(id: String!): DeleteResponse
    
    createMedicationRequest(medicationRequestData: MedicationRequestInput!): MedicationRequest
    updateMedicationRequest(id: String!, medicationRequestData: MedicationRequestInput!): MedicationRequest
    deleteMedicationRequest(id: String!): DeleteResponse
    
    createMedicationStatement(medicationStatementData: MedicationStatementInput!): MedicationStatement
    updateMedicationStatement(id: String!, medicationStatementData: MedicationStatementInput!): MedicationStatement
    deleteMedicationStatement(id: String!): DeleteResponse
    
    createMedicationAdministration(medicationAdministrationData: MedicationAdministrationInput!): MedicationAdministration
    updateMedicationAdministration(id: String!, medicationAdministrationData: MedicationAdministrationInput!): MedicationAdministration
    deleteMedicationAdministration(id: String!): DeleteResponse
    
    createAllergy(allergyData: AllergyIntoleranceInput!): AllergyIntolerance
    updateAllergy(id: String!, allergyData: AllergyIntoleranceInput!): AllergyIntolerance
    deleteAllergy(id: String!): DeleteResponse
  }

  # Medication-specific types
  type Medication @key(fields: "id") {
    id: ID!
    resourceType: String!
    status: String
    code: CodeableConcept
    manufacturer: Reference
    form: CodeableConcept
    amount: Ratio
    ingredient: [MedicationIngredient]
    batch: MedicationBatch
  }

  type MedicationIngredient {
    itemCodeableConcept: CodeableConcept
    itemReference: Reference
    isActive: Boolean
    strength: Ratio
  }

  type MedicationBatch {
    lotNumber: String
    expirationDate: String
  }

  type MedicationRequest @key(fields: "id") {
    id: ID!
    resourceType: String!
    identifier: [Identifier]
    status: String!
    intent: String!
    category: [CodeableConcept]
    priority: String
    doNotPerform: Boolean
    reportedBoolean: Boolean
    reportedReference: Reference
    medicationCodeableConcept: CodeableConcept
    medicationReference: Reference
    subject: Reference!
    encounter: Reference
    supportingInformation: [Reference]
    authoredOn: String
    requester: Reference
    performer: Reference
    performerType: CodeableConcept
    recorder: Reference
    reasonCode: [CodeableConcept]
    reasonReference: [Reference]
    instantiatesCanonical: [String]
    instantiatesUri: [String]
    basedOn: [Reference]
    groupIdentifier: Identifier
    courseOfTherapyType: CodeableConcept
    insurance: [Reference]
    note: [Annotation]
    dosageInstruction: [DosageInstruction]
    dispenseRequest: MedicationRequestDispenseRequest
    substitution: MedicationRequestSubstitution
    priorPrescription: Reference
    detectedIssue: [Reference]
    eventHistory: [Reference]
  }

  type DosageInstruction {
    sequence: Int
    text: String
    additionalInstruction: [CodeableConcept]
    patientInstruction: String
    timing: Timing
    asNeededBoolean: Boolean
    asNeededCodeableConcept: CodeableConcept
    site: CodeableConcept
    route: CodeableConcept
    method: CodeableConcept
    doseAndRate: [DosageInstructionDoseAndRate]
    maxDosePerPeriod: Ratio
    maxDosePerAdministration: Quantity
    maxDosePerLifetime: Quantity
  }

  type Timing {
    event: [String]
    repeat: TimingRepeat
    code: CodeableConcept
  }

  type TimingRepeat {
    boundsDuration: Duration
    boundsRange: Range
    boundsPeriod: Period
    count: Int
    countMax: Int
    duration: Float
    durationMax: Float
    durationUnit: String
    frequency: Int
    frequencyMax: Int
    period: Float
    periodMax: Float
    periodUnit: String
    dayOfWeek: [String]
    timeOfDay: [String]
    when: [String]
    offset: Int
  }

  type Duration {
    value: Float
    comparator: String
    unit: String
    system: String
    code: String
  }

  type Range {
    low: Quantity
    high: Quantity
  }

  type DosageInstructionDoseAndRate {
    type: CodeableConcept
    doseRange: Range
    doseQuantity: Quantity
    rateRatio: Ratio
    rateRange: Range
    rateQuantity: Quantity
  }

  type MedicationRequestDispenseRequest {
    initialFill: MedicationRequestDispenseRequestInitialFill
    dispenseInterval: Duration
    validityPeriod: Period
    numberOfRepeatsAllowed: Int
    quantity: Quantity
    expectedSupplyDuration: Duration
    performer: Reference
  }

  type MedicationRequestDispenseRequestInitialFill {
    quantity: Quantity
    duration: Duration
  }

  type MedicationRequestSubstitution {
    allowedBoolean: Boolean
    allowedCodeableConcept: CodeableConcept
    reason: CodeableConcept
  }

  type MedicationStatement @key(fields: "id") {
    id: ID!
    resourceType: String!
    identifier: [Identifier]
    basedOn: [Reference]
    partOf: [Reference]
    status: String!
    statusReason: [CodeableConcept]
    category: CodeableConcept
    medicationCodeableConcept: CodeableConcept
    medicationReference: Reference
    subject: Reference!
    context: Reference
    effectiveDateTime: String
    effectivePeriod: Period
    dateAsserted: String
    informationSource: Reference
    derivedFrom: [Reference]
    reasonCode: [CodeableConcept]
    reasonReference: [Reference]
    note: [Annotation]
    dosage: [DosageInstruction]
  }

  type MedicationAdministration @key(fields: "id") {
    id: ID!
    resourceType: String!
    identifier: [Identifier]
    instantiates: [String]
    partOf: [Reference]
    status: String!
    statusReason: [CodeableConcept]
    category: CodeableConcept
    medicationCodeableConcept: CodeableConcept
    medicationReference: Reference
    subject: Reference!
    context: Reference
    supportingInformation: [Reference]
    effectiveDateTime: String
    effectivePeriod: Period
    performer: [MedicationAdministrationPerformer]
    reasonCode: [CodeableConcept]
    reasonReference: [Reference]
    request: Reference
    device: [Reference]
    note: [Annotation]
    dosage: MedicationAdministrationDosage
    eventHistory: [Reference]
  }

  type MedicationAdministrationPerformer {
    function: CodeableConcept
    actor: Reference!
  }

  type MedicationAdministrationDosage {
    text: String
    site: CodeableConcept
    route: CodeableConcept
    method: CodeableConcept
    dose: Quantity
    rateRatio: Ratio
    rateQuantity: Quantity
  }

  type AllergyIntolerance @key(fields: "id") {
    id: ID!
    resourceType: String!
    identifier: [Identifier]
    clinicalStatus: CodeableConcept
    verificationStatus: CodeableConcept
    type: String
    category: [String]
    criticality: String
    code: CodeableConcept
    patient: Reference!
    encounter: Reference
    onsetDateTime: String
    onsetAge: Quantity
    onsetPeriod: Period
    onsetRange: Range
    onsetString: String
    recordedDate: String
    recorder: Reference
    asserter: Reference
    lastOccurrence: String
    note: [Annotation]
    reaction: [AllergyIntoleranceReaction]
  }

  type AllergyIntoleranceReaction {
    substance: CodeableConcept
    manifestation: [CodeableConcept]!
    description: String
    onset: String
    severity: String
    exposureRoute: CodeableConcept
    note: [Annotation]
  }

  # Connection types for pagination
  type MedicationConnection {
    items: [Medication]
    total: Int
    page: Int
    count: Int
  }

  type MedicationRequestConnection {
    items: [MedicationRequest]
    total: Int
    page: Int
    count: Int
  }

  type MedicationStatementConnection {
    items: [MedicationStatement]
    total: Int
    page: Int
    count: Int
  }

  type MedicationAdministrationConnection {
    items: [MedicationAdministration]
    total: Int
    page: Int
    count: Int
  }

  type AllergyIntoleranceConnection {
    items: [AllergyIntolerance]
    total: Int
    page: Int
    count: Int
  }

  type DeleteResponse {
    success: Boolean!
    message: String
  }

  # Input types for mutations
  input MedicationInput {
    status: String
    code: CodeableConceptInput!
    manufacturer: ReferenceInput
    form: CodeableConceptInput
    amount: RatioInput
    ingredient: [MedicationIngredientInput]
    batch: MedicationBatchInput
  }

  input MedicationIngredientInput {
    itemCodeableConcept: CodeableConceptInput
    itemReference: ReferenceInput
    isActive: Boolean
    strength: RatioInput
  }

  input MedicationBatchInput {
    lotNumber: String
    expirationDate: String
  }

  input MedicationRequestInput {
    identifier: [IdentifierInput]
    status: String!
    intent: String!
    category: [CodeableConceptInput]
    priority: String
    doNotPerform: Boolean
    reportedBoolean: Boolean
    reportedReference: ReferenceInput
    medicationCodeableConcept: CodeableConceptInput
    medicationReference: ReferenceInput
    subject: ReferenceInput!
    encounter: ReferenceInput
    supportingInformation: [ReferenceInput]
    authoredOn: String
    requester: ReferenceInput
    performer: ReferenceInput
    performerType: CodeableConceptInput
    recorder: ReferenceInput
    reasonCode: [CodeableConceptInput]
    reasonReference: [ReferenceInput]
    instantiatesCanonical: [String]
    instantiatesUri: [String]
    basedOn: [ReferenceInput]
    groupIdentifier: IdentifierInput
    courseOfTherapyType: CodeableConceptInput
    insurance: [ReferenceInput]
    note: [AnnotationInput]
    dosageInstruction: [DosageInstructionInput]
    dispenseRequest: MedicationRequestDispenseRequestInput
    substitution: MedicationRequestSubstitutionInput
    priorPrescription: ReferenceInput
    detectedIssue: [ReferenceInput]
    eventHistory: [ReferenceInput]
  }

  input MedicationStatementInput {
    identifier: [IdentifierInput]
    basedOn: [ReferenceInput]
    partOf: [ReferenceInput]
    status: String!
    statusReason: [CodeableConceptInput]
    category: CodeableConceptInput
    medicationCodeableConcept: CodeableConceptInput
    medicationReference: ReferenceInput
    subject: ReferenceInput!
    context: ReferenceInput
    effectiveDateTime: String
    effectivePeriod: PeriodInput
    dateAsserted: String
    informationSource: ReferenceInput
    derivedFrom: [ReferenceInput]
    reasonCode: [CodeableConceptInput]
    reasonReference: [ReferenceInput]
    note: [AnnotationInput]
    dosage: [DosageInstructionInput]
  }

  input MedicationAdministrationInput {
    identifier: [IdentifierInput]
    instantiates: [String]
    partOf: [ReferenceInput]
    status: String!
    statusReason: [CodeableConceptInput]
    category: CodeableConceptInput
    medicationCodeableConcept: CodeableConceptInput
    medicationReference: ReferenceInput
    subject: ReferenceInput!
    context: ReferenceInput
    supportingInformation: [ReferenceInput]
    effectiveDateTime: String
    effectivePeriod: PeriodInput
    performer: [MedicationAdministrationPerformerInput]
    reasonCode: [CodeableConceptInput]
    reasonReference: [ReferenceInput]
    request: ReferenceInput
    device: [ReferenceInput]
    note: [AnnotationInput]
    dosage: MedicationAdministrationDosageInput
    eventHistory: [ReferenceInput]
  }

  input AllergyIntoleranceInput {
    identifier: [IdentifierInput]
    clinicalStatus: CodeableConceptInput
    verificationStatus: CodeableConceptInput
    type: String
    category: [String]
    criticality: String
    code: CodeableConceptInput
    patient: ReferenceInput!
    encounter: ReferenceInput
    onsetDateTime: String
    onsetAge: QuantityInput
    onsetPeriod: PeriodInput
    onsetRange: RangeInput
    onsetString: String
    recordedDate: String
    recorder: ReferenceInput
    asserter: ReferenceInput
    lastOccurrence: String
    note: [AnnotationInput]
    reaction: [AllergyIntoleranceReactionInput]
  }

  input AllergyIntoleranceReactionInput {
    substance: CodeableConceptInput
    manifestation: [CodeableConceptInput]!
    description: String
    onset: String
    severity: String
    exposureRoute: CodeableConceptInput
    note: [AnnotationInput]
  }

  input DosageInstructionInput {
    sequence: Int
    text: String
    additionalInstruction: [CodeableConceptInput]
    patientInstruction: String
    timing: TimingInput
    asNeededBoolean: Boolean
    asNeededCodeableConcept: CodeableConceptInput
    site: CodeableConceptInput
    route: CodeableConceptInput
    method: CodeableConceptInput
    doseAndRate: [DosageInstructionDoseAndRateInput]
    maxDosePerPeriod: RatioInput
    maxDosePerAdministration: QuantityInput
    maxDosePerLifetime: QuantityInput
  }

  input TimingInput {
    event: [String]
    repeat: TimingRepeatInput
    code: CodeableConceptInput
  }

  input TimingRepeatInput {
    boundsDuration: DurationInput
    boundsRange: RangeInput
    boundsPeriod: PeriodInput
    count: Int
    countMax: Int
    duration: Float
    durationMax: Float
    durationUnit: String
    frequency: Int
    frequencyMax: Int
    period: Float
    periodMax: Float
    periodUnit: String
    dayOfWeek: [String]
    timeOfDay: [String]
    when: [String]
    offset: Int
  }

  input DurationInput {
    value: Float
    comparator: String
    unit: String
    system: String
    code: String
  }

  input RangeInput {
    low: QuantityInput
    high: QuantityInput
  }

  input DosageInstructionDoseAndRateInput {
    type: CodeableConceptInput
    doseRange: RangeInput
    doseQuantity: QuantityInput
    rateRatio: RatioInput
    rateRange: RangeInput
    rateQuantity: QuantityInput
  }

  input MedicationRequestDispenseRequestInput {
    initialFill: MedicationRequestDispenseRequestInitialFillInput
    dispenseInterval: DurationInput
    validityPeriod: PeriodInput
    numberOfRepeatsAllowed: Int
    quantity: QuantityInput
    expectedSupplyDuration: DurationInput
    performer: ReferenceInput
  }

  input MedicationRequestDispenseRequestInitialFillInput {
    quantity: QuantityInput
    duration: DurationInput
  }

  input MedicationRequestSubstitutionInput {
    allowedBoolean: Boolean
    allowedCodeableConcept: CodeableConceptInput
    reason: CodeableConceptInput
  }

  input MedicationAdministrationPerformerInput {
    function: CodeableConceptInput
    actor: ReferenceInput!
  }

  input MedicationAdministrationDosageInput {
    text: String
    site: CodeableConceptInput
    route: CodeableConceptInput
    method: CodeableConceptInput
    dose: QuantityInput
    rateRatio: RatioInput
    rateQuantity: QuantityInput
  }
`;

module.exports = typeDefs;
