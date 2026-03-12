const { gql } = require('apollo-server-express');

const typeDefs = gql`
  extend schema @link(url: "https://specs.apollo.dev/federation/v2.0", import: ["@key", "@shareable", "@external", "@provides", "@requires"])

  # Root Query Extensions
  extend type Query {
    # Core Order Queries
    order(id: ID!): ClinicalOrder
    orders(filters: OrderSearchFilters, pagination: PaginationInput, sort: OrderSortInput): OrderConnection
    medicationOrder(id: ID!): MedicationOrder
    labOrder(id: ID!): LabOrder
    imagingOrder(id: ID!): ImagingOrder

    # Order Set Queries
    orderSet(id: ID!): OrderSet
    orderSets(category: String, condition: String, status: String): [OrderSet!]!

    # Clinical Decision Support Queries
    checkDrugInteractions(patientId: ID!, medications: [String!]!): [DrugInteractionAlert!]!
    validateOrderAppropriateness(orderData: ClinicalOrderInput!, patientId: ID!): [ClinicalAlert!]!

    # Analytics and Reporting Queries
    orderStatistics(dateRange: JSON, department: String): OrderStatistics
    orderHistory(orderId: ID!): OrderAuditTrail
    searchOrders(query: String!, filters: OrderSearchFilters): [OrderSearchResult!]!
  }

  # Root Mutation Extensions
  extend type Mutation {
    # Order Creation Mutations
    createOrder(orderData: ClinicalOrderInput!): ClinicalOrder!
    createMedicationOrder(orderData: MedicationOrderInput!): MedicationOrder!
    createLabOrder(orderData: LabOrderInput!): LabOrder!
    createImagingOrder(orderData: ImagingOrderInput!): ImagingOrder!
    createOrderBatch(batchData: OrderBatchInput!): [ClinicalOrder!]!

    # Clinical Decision Support Mutations
    createOrderWithCDS(orderData: ClinicalOrderInput!, cdsOptions: CDSOptionsInput!): JSON!

    # Order Lifecycle Management Mutations
    signOrder(id: ID!, signatureData: SignatureInput!): ClinicalOrder!
    holdOrder(id: ID!, holdData: OrderActionInput!): ClinicalOrder!
    cancelOrder(id: ID!, cancelData: OrderActionInput!): ClinicalOrder!
    releaseOrder(id: ID!): ClinicalOrder!
    completeOrder(id: ID!, completionData: OrderActionInput!): ClinicalOrder!
    updateOrder(id: ID!, orderData: ClinicalOrderInput!): ClinicalOrder!
    bulkOrderActions(actionData: BulkOrderActionInput!): [ClinicalOrder!]!

    # Order Set Management Mutations
    createOrderSet(orderSetData: OrderSetInput!): OrderSet!
    applyOrderSet(orderSetId: ID!, patientId: ID!, customizations: JSON): [ClinicalOrder!]!
    updateOrderSet(id: ID!, orderSetData: OrderSetInput!): OrderSet!
    activateOrderSet(id: ID!): OrderSet!
    retireOrderSet(id: ID!): OrderSet!
  }

  # Extend Patient type to add comprehensive order-related fields
  extend type Patient @key(fields: "id") {
    id: ID! @external
    orders(filters: OrderSearchFilters, pagination: PaginationInput, sort: OrderSortInput): OrderConnection
    activeOrders: [ClinicalOrder!]!
    medicationOrders: [MedicationOrder!]!
    labOrders: [LabOrder!]!
    imagingOrders: [ImagingOrder!]!
    orderStatistics: OrderStatistics
  }

  # Extend User type to add practitioner order management
  extend type User @key(fields: "id") {
    id: ID! @external
    ordersRequested(filters: OrderSearchFilters, pagination: PaginationInput): OrderConnection
    ordersSigned: [ClinicalOrder!]!
    orderSetsCreated: [OrderSet!]!
  }

  # Extend Encounter type to add order management
  extend type Encounter @key(fields: "id") {
    id: ID! @external
    orders: [ClinicalOrder!]!
    orderSummary: OrderStatistics
  }

  # Core Order Types
  type ClinicalOrder @key(fields: "id") {
    id: ID!
    resourceType: String!
    status: OrderStatusEnum!
    intent: OrderIntentEnum!
    category: [CodeableConcept!]
    priority: OrderPriorityEnum
    code: CodeableConcept!
    subject: Reference!
    encounter: Reference
    occurrenceDatetime: String
    authoredOn: String
    requester: Reference
    performer: [Reference!]
    reasonCode: [CodeableConcept!]
    reasonReference: [Reference!]
    note: [Annotation!]
    patientInstruction: String
    supportingInfo: [Reference!]
    specimen: [Reference!]
    bodySite: [CodeableConcept!]
    statusHistory: [OrderStatusHistory!]
    signature: OrderSignature
  }

  type MedicationOrder @key(fields: "id") {
    id: ID!
    resourceType: String!
    status: OrderStatusEnum!
    intent: OrderIntentEnum!
    priority: OrderPriorityEnum
    medicationCodeableConcept: CodeableConcept!
    subject: Reference!
    encounter: Reference
    requester: Reference
    dosageInstruction: [DosageInstruction!]
    dispenseRequest: DispenseRequest
    substitution: MedicationSubstitution
    reasonCode: [CodeableConcept!]
    note: [Annotation!]
  }

  type LabOrder @key(fields: "id") {
    id: ID!
    resourceType: String!
    status: OrderStatusEnum!
    intent: OrderIntentEnum!
    code: CodeableConcept!
    subject: Reference!
    specimen: [Reference!]
    specimenRequirements: SpecimenRequirements
    clinicalContext: ClinicalContext
    urgency: String
    priority: OrderPriorityEnum
  }

  type ImagingOrder @key(fields: "id") {
    id: ID!
    resourceType: String!
    status: OrderStatusEnum!
    intent: OrderIntentEnum!
    code: CodeableConcept!
    subject: Reference!
    modality: CodeableConcept
    bodySite: CodeableConcept
    contrast: ContrastRequirements
    radiationSafety: RadiationSafety
    clinicalIndication: CodeableConcept
    urgency: String
    priority: OrderPriorityEnum
  }

  type OrderSet @key(fields: "id") {
    id: ID!
    name: String!
    description: String
    category: CodeableConcept
    status: String!
    orders: [OrderSetItem!]!
    applicableConditions: [CodeableConcept!]
    customizations: [OrderSetCustomization!]
    metadata: OrderSetMetadata
  }

  # Supporting Types
  enum OrderStatusEnum {
    DRAFT
    ACTIVE
    ON_HOLD
    REVOKED
    COMPLETED
    ENTERED_IN_ERROR
    UNKNOWN
  }

  enum OrderIntentEnum {
    PROPOSAL
    PLAN
    DIRECTIVE
    ORDER
    ORIGINAL_ORDER
    REFLEX_ORDER
    FILLER_ORDER
    INSTANCE_ORDER
    OPTION
  }

  enum OrderPriorityEnum {
    ROUTINE
    URGENT
    ASAP
    STAT
  }

  type CodeableConcept {
    coding: [Coding!]
    text: String
  }

  type Coding {
    system: String
    code: String
    display: String
    version: String
  }

  type Reference {
    reference: String!
    display: String
    type: String
  }

  type Quantity {
    value: Float
    unit: String
    system: String
    code: String
  }

  type Annotation {
    text: String!
    authorString: String
    time: String
  }

  type OrderStatusHistory {
    status: OrderStatusEnum!
    timestamp: String!
    reason: String
    user: Reference
  }

  type OrderSignature {
    type: String!
    signedOn: String!
    signer: Reference!
    digitalSignature: DigitalSignature
  }

  type DigitalSignature {
    algorithm: String!
    hash: String!
    timestamp: String!
  }

  type DosageInstruction {
    text: String
    timing: Timing
    route: CodeableConcept
    doseAndRate: [DoseAndRate!]
    maxDosePerPeriod: Ratio
  }

  type Timing {
    repeat: TimingRepeat
  }

  type TimingRepeat {
    frequency: Int
    period: Float
    periodUnit: String
    timeOfDay: [String!]
    when: [String!]
  }

  type DoseAndRate {
    doseQuantity: Quantity
    rateQuantity: Quantity
  }

  type Ratio {
    numerator: Quantity
    denominator: Quantity
  }

  type DispenseRequest {
    quantity: Quantity
    expectedSupplyDuration: Quantity
    numberOfRepeatsAllowed: Int
    performer: Reference
  }

  type MedicationSubstitution {
    allowedBoolean: Boolean
    reason: CodeableConcept
  }

  type SpecimenRequirements {
    type: CodeableConcept
    collection: SpecimenCollection
    container: SpecimenContainer
  }

  type SpecimenCollection {
    method: CodeableConcept
    bodySite: CodeableConcept
    fastingStatus: CodeableConcept
  }

  type SpecimenContainer {
    type: CodeableConcept
    capacity: Quantity
  }

  type ClinicalContext {
    indication: CodeableConcept
    urgency: String
    priority: String
  }

  type ContrastRequirements {
    required: Boolean!
    type: CodeableConcept
    contraindications: String
  }

  type RadiationSafety {
    dose: Quantity
    pregnancy: PregnancyStatus
  }

  type PregnancyStatus {
    status: String
    screening: String
  }

  type OrderSetItem {
    type: String!
    priority: OrderPriorityEnum
    code: CodeableConcept
    dosageInstruction: DosageInstruction
  }

  type OrderSetCustomization {
    field: String!
    options: [String!]!
    defaultValue: String
  }

  type OrderSetMetadata {
    version: String
    author: Reference
    dateCreated: String
    lastModified: String
  }

  # Clinical Decision Support Types
  type DrugInteractionAlert {
    id: ID!
    severity: String!
    interactionType: String!
    description: String!
    recommendation: String
    source: String
    evidenceLevel: String
    affectedMedications: [Reference!]!
    clinicalSignificance: String
  }

  type ClinicalAlert {
    id: ID!
    alertType: String!
    severity: String!
    title: String!
    description: String!
    recommendation: String
    source: String
    triggeredBy: Reference
    patientContext: [String!]
  }

  # Analytics and Search Types
  type OrderStatistics {
    totalOrders: Int!
    draftOrders: Int!
    activeOrders: Int!
    completedOrders: Int!
    cancelledOrders: Int!
    onHoldOrders: Int!
    ordersByPriority: JSON
    ordersByCategory: JSON
    ordersByRequester: JSON
    ordersByDateRange: JSON
    averageCompletionTime: Float
    mostCommonOrders: [JSON!]!
  }

  type OrderConnection {
    edges: [OrderEdge!]!
    pageInfo: PageInfo!
    totalCount: Int!
  }

  type OrderEdge {
    node: ClinicalOrder!
    cursor: String!
  }

  type PageInfo {
    hasNextPage: Boolean!
    hasPreviousPage: Boolean!
    startCursor: String
    endCursor: String
  }

  type OrderSearchResult {
    order: ClinicalOrder!
    score: Float!
    highlights: [String!]!
  }

  type OrderAuditTrail {
    order: ClinicalOrder!
    auditTrail: [AuditEntry!]!
    statusHistory: [OrderStatusHistory!]!
    modifications: [OrderModification!]!
    signatures: [OrderSignature!]!
  }

  type AuditEntry {
    action: String!
    timestamp: String!
    user: Reference!
    details: [AuditDetail!]!
    reason: String
    ipAddress: String
    userAgent: String
  }

  type AuditDetail {
    field: String!
    oldValue: String
    newValue: String
  }

  type OrderModification {
    field: String!
    oldValue: String
    newValue: String
    timestamp: String!
    user: Reference!
    reason: String
  }

  # Input Types for Mutations and Queries
  input ClinicalOrderInput {
    status: OrderStatusEnum!
    intent: OrderIntentEnum!
    category: [CodeableConceptInput!]
    priority: OrderPriorityEnum
    code: CodeableConceptInput!
    subject: ReferenceInput!
    encounter: ReferenceInput
    occurrenceDatetime: String
    requester: ReferenceInput
    performer: [ReferenceInput!]
    reasonCode: [CodeableConceptInput!]
    reasonReference: [ReferenceInput!]
    note: [AnnotationInput!]
    patientInstruction: String
    supportingInfo: [ReferenceInput!]
    specimen: [ReferenceInput!]
    bodySite: [CodeableConceptInput!]
  }

  input MedicationOrderInput {
    status: OrderStatusEnum!
    intent: OrderIntentEnum!
    priority: OrderPriorityEnum
    medicationCodeableConcept: CodeableConceptInput!
    subject: ReferenceInput!
    encounter: ReferenceInput
    requester: ReferenceInput
    dosageInstruction: [DosageInstructionInput!]
    dispenseRequest: DispenseRequestInput
    substitution: MedicationSubstitutionInput
    reasonCode: [CodeableConceptInput!]
    note: [AnnotationInput!]
  }

  input LabOrderInput {
    status: OrderStatusEnum!
    intent: OrderIntentEnum!
    code: CodeableConceptInput!
    subject: ReferenceInput!
    specimen: [ReferenceInput!]
    specimenRequirements: SpecimenRequirementsInput
    clinicalContext: ClinicalContextInput
    urgency: String
    priority: OrderPriorityEnum
  }

  input ImagingOrderInput {
    status: OrderStatusEnum!
    intent: OrderIntentEnum!
    code: CodeableConceptInput!
    subject: ReferenceInput!
    modality: CodeableConceptInput
    bodySite: CodeableConceptInput
    contrast: ContrastRequirementsInput
    radiationSafety: RadiationSafetyInput
    clinicalIndication: CodeableConceptInput
    urgency: String
    priority: OrderPriorityEnum
  }

  input OrderSetInput {
    name: String!
    description: String
    category: CodeableConceptInput
    status: String!
    orders: [OrderSetItemInput!]!
    applicableConditions: [CodeableConceptInput!]
    customizations: [OrderSetCustomizationInput!]
    metadata: OrderSetMetadataInput
  }

  input CodeableConceptInput {
    coding: [CodingInput!]
    text: String
  }

  input CodingInput {
    system: String
    code: String
    display: String
    version: String
  }

  input ReferenceInput {
    reference: String!
    display: String
    type: String
  }

  input QuantityInput {
    value: Float
    unit: String
    system: String
    code: String
  }

  input AnnotationInput {
    text: String!
    authorString: String
    time: String
  }

  input DosageInstructionInput {
    text: String
    timing: TimingInput
    route: CodeableConceptInput
    doseAndRate: [DoseAndRateInput!]
    maxDosePerPeriod: RatioInput
  }

  input TimingInput {
    repeat: TimingRepeatInput
  }

  input TimingRepeatInput {
    frequency: Int
    period: Float
    periodUnit: String
    timeOfDay: [String!]
    when: [String!]
  }

  input DoseAndRateInput {
    doseQuantity: QuantityInput
    rateQuantity: QuantityInput
  }

  input RatioInput {
    numerator: QuantityInput
    denominator: QuantityInput
  }

  input DispenseRequestInput {
    quantity: QuantityInput
    expectedSupplyDuration: QuantityInput
    numberOfRepeatsAllowed: Int
    performer: ReferenceInput
  }

  input MedicationSubstitutionInput {
    allowedBoolean: Boolean
    reason: CodeableConceptInput
  }

  input SpecimenRequirementsInput {
    type: CodeableConceptInput
    collection: SpecimenCollectionInput
    container: SpecimenContainerInput
  }

  input SpecimenCollectionInput {
    method: CodeableConceptInput
    bodySite: CodeableConceptInput
    fastingStatus: CodeableConceptInput
  }

  input SpecimenContainerInput {
    type: CodeableConceptInput
    capacity: QuantityInput
  }

  input ClinicalContextInput {
    indication: CodeableConceptInput
    urgency: String
    priority: String
  }

  input ContrastRequirementsInput {
    required: Boolean!
    type: CodeableConceptInput
    contraindications: String
  }

  input RadiationSafetyInput {
    dose: QuantityInput
    pregnancy: PregnancyStatusInput
  }

  input PregnancyStatusInput {
    status: String
    screening: String
  }

  input OrderSetItemInput {
    type: String!
    priority: OrderPriorityEnum
    code: CodeableConceptInput
    dosageInstruction: DosageInstructionInput
  }

  input OrderSetCustomizationInput {
    field: String!
    options: [String!]!
    defaultValue: String
  }

  input OrderSetMetadataInput {
    version: String
    author: ReferenceInput
  }

  input OrderSearchFilters {
    status: [OrderStatusEnum!]
    patientId: String
    requesterId: String
    encounterId: String
    dateRange: DateRangeInput
    priority: [OrderPriorityEnum!]
    category: [String!]
  }

  input DateRangeInput {
    start: String
    end: String
  }

  input PaginationInput {
    page: Int = 1
    limit: Int = 20
  }

  input OrderSortInput {
    field: String!
    direction: SortDirection!
  }

  enum SortDirection {
    ASC
    DESC
  }

  input CDSOptionsInput {
    checkDrugInteractions: Boolean = true
    checkAllergies: Boolean = true
    checkDuplicateTherapy: Boolean = true
    checkContraindications: Boolean = true
    checkDosing: Boolean = true
    includeRecommendations: Boolean = true
  }

  input OrderActionInput {
    reason: String!
    reasonCode: CodeableConceptInput
    user: ReferenceInput!
    timestamp: String
    note: String
  }

  input SignatureInput {
    type: String!
    signer: ReferenceInput!
    signedOn: String
    digitalSignature: DigitalSignatureInput
    reason: String
  }

  input DigitalSignatureInput {
    algorithm: String!
    hash: String!
    timestamp: String!
  }

  input BulkOrderActionInput {
    orderIds: [ID!]!
    action: String!
    reason: String
    user: ReferenceInput!
  }

  input OrderBatchInput {
    patientId: String!
    encounterId: String
    requesterId: String!
    orders: [OrderBatchItemInput!]!
  }

  input OrderBatchItemInput {
    type: String!
    status: OrderStatusEnum!
    intent: OrderIntentEnum!
    code: CodeableConceptInput
    medicationCodeableConcept: CodeableConceptInput
    dosageInstruction: [DosageInstructionInput!]
  }

  scalar JSON
`;

module.exports = typeDefs;
