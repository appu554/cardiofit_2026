const { gql } = require('apollo-server-express');

const kb7TerminologyTypeDefs = gql`
  extend type Query {
    # Terminology System Operations
    terminologySystems(status: TerminologyStatus): [TerminologySystem!]!
    terminologySystem(identifier: String!): TerminologySystem
    
    # Concept Operations
    searchConcepts(input: ConceptSearchInput!): ConceptSearchResult!
    lookupConcept(system: String!, code: String!): ConceptLookupResult
    validateCode(input: CodeValidationInput!): CodeValidationResult!
    
    # Value Set Operations
    valueSets(domain: String, status: ValueSetStatus): [ValueSet!]!
    valueSet(url: String!, version: String): ValueSet
    expandValueSet(url: String!, version: String, filter: String): ValueSetExpansion
    
    # Concept Mapping Operations
    conceptMappings(
      sourceSystem: String, 
      targetSystem: String, 
      equivalence: MappingEquivalence
    ): [ConceptMapping!]!
    translateConcept(input: ConceptTranslationInput!): ConceptTranslationResult!
    
    # Batch Operations
    batchLookupConcepts(requests: [ConceptLookupRequest!]!): [ConceptLookupResult!]!
    batchValidateCodes(requests: [CodeValidationRequest!]!): [CodeValidationResult!]!
  }

  extend type Mutation {
    # Administrative Operations (if enabled)
    refreshTerminologySystem(systemUri: String!): OperationResult!
    rebuildValueSetExpansion(valueSetUrl: String!): OperationResult!
  }

  # Core Types
  type TerminologySystem {
    id: ID!
    systemUri: String!
    systemName: String!
    version: String!
    description: String
    publisher: String
    status: TerminologyStatus!
    metadata: JSON
    supportedRegions: [String!]!
    conceptCount: Int
    hierarchyMeaning: String
    compositional: Boolean
    versionNeeded: Boolean
    content: ContentType
    createdAt: DateTime!
    updatedAt: DateTime!
  }

  type TerminologyConcept {
    id: ID!
    systemId: String!
    code: String!
    display: String!
    definition: String
    status: ConceptStatus!
    parentCodes: [String!]!
    childCodes: [String!]!
    properties: JSON
    designations: [ConceptDesignation!]!
    clinicalDomain: String
    specialty: String
    createdAt: DateTime!
    updatedAt: DateTime!
  }

  type ConceptDesignation {
    language: String!
    use: JSON
    value: String!
  }

  type ValueSet {
    id: ID!
    url: String!
    identifier: [Identifier!]
    version: String!
    name: String!
    title: String
    description: String
    status: ValueSetStatus!
    experimental: Boolean
    date: DateTime
    publisher: String
    contact: [ContactPoint!]
    useContext: [UsageContext!]
    jurisdiction: [CodeableConcept!]
    purpose: String
    copyright: String
    clinicalDomain: String
    compose: ValueSetCompose
    expansion: ValueSetExpansion
    supportedRegions: [String!]!
    createdAt: DateTime!
    updatedAt: DateTime!
    expiredAt: DateTime
  }

  type ValueSetCompose {
    lockedDate: String
    inactive: Boolean
    include: [ValueSetInclude!]!
    exclude: [ValueSetExclude!]!
  }

  type ValueSetInclude {
    system: String
    version: String
    concept: [ValueSetConcept!]
    filter: [ValueSetFilter!]
    valueSet: [String!]
  }

  type ValueSetExclude {
    system: String
    version: String
    concept: [ValueSetConcept!]
    filter: [ValueSetFilter!]
    valueSet: [String!]
  }

  type ValueSetConcept {
    code: String!
    display: String
    designation: [ConceptDesignation!]
  }

  type ValueSetFilter {
    property: String!
    op: FilterOperator!
    value: String!
  }

  type ValueSetExpansion {
    identifier: String
    timestamp: DateTime!
    total: Int
    offset: Int
    parameter: [ExpansionParameter!]
    contains: [ValueSetExpansionConcept!]!
  }

  type ValueSetExpansionConcept {
    system: String
    abstract: Boolean
    inactive: Boolean
    version: String
    code: String
    display: String
    designation: [ConceptDesignation!]
    contains: [ValueSetExpansionConcept!]
  }

  type ExpansionParameter {
    name: String!
    valueString: String
    valueBoolean: Boolean
    valueInteger: Int
    valueDecimal: Float
    valueUri: String
    valueCode: String
  }

  type ConceptMapping {
    id: ID!
    sourceSystemId: String!
    sourceCode: String!
    targetSystemId: String!
    targetCode: String!
    equivalence: MappingEquivalence!
    mappingType: String!
    confidence: Float!
    comment: String
    mappedBy: String
    evidence: JSON
    verified: Boolean!
    verifiedBy: String
    verifiedAt: DateTime
    usageCount: Int!
    lastUsedAt: DateTime
    createdAt: DateTime!
    updatedAt: DateTime!
  }

  # Search and Lookup Results
  type ConceptSearchResult {
    total: Int!
    concepts: [TerminologyConcept!]!
    facets: [SearchFacet!]
  }

  type ConceptLookupResult {
    concept: TerminologyConcept!
    properties: JSON
    designations: [ConceptDesignation!]
    parents: [TerminologyConcept!]
    children: [TerminologyConcept!]
    system: TerminologySystem!
  }

  type CodeValidationResult {
    valid: Boolean!
    code: String!
    system: String!
    display: String
    message: String
    severity: ValidationSeverity!
    issues: [ValidationIssue!]
  }

  type ValidationIssue {
    severity: ValidationSeverity!
    code: String!
    details: String!
    location: String
  }

  type ConceptTranslationResult {
    result: Boolean!
    message: String
    matches: [ConceptTranslationMatch!]!
  }

  type ConceptTranslationMatch {
    equivalence: MappingEquivalence
    concept: ConceptTranslationConcept!
    source: String
    comment: String
    dependsOn: [ConceptTranslationDependency!]
    product: [ConceptTranslationProduct!]
  }

  type ConceptTranslationConcept {
    system: String
    version: String
    code: String
    display: String
  }

  type ConceptTranslationDependency {
    property: String!
    system: String
    value: String!
    display: String
  }

  type ConceptTranslationProduct {
    property: String!
    system: String
    value: String!
    display: String
  }

  type SearchFacet {
    name: String!
    values: [SearchFacetValue!]!
  }

  type SearchFacetValue {
    value: String!
    count: Int!
  }

  # Input Types
  input ConceptSearchInput {
    query: String
    systemUri: String
    count: Int = 20
    offset: Int = 0
    filter: JSON
    includeDesignations: Boolean = false
    includeFacets: Boolean = false
  }

  input CodeValidationInput {
    code: String!
    system: String!
    version: String
    display: String
    abstract: Boolean
  }

  input ConceptTranslationInput {
    code: String!
    system: String!
    version: String
    targetSystem: String!
    conceptMapUrl: String
    reverse: Boolean = false
  }

  input ConceptLookupRequest {
    system: String!
    code: String!
  }

  input CodeValidationRequest {
    code: String!
    system: String!
    version: String
    display: String
  }

  # Supporting Types
  type Identifier {
    use: IdentifierUse
    type: CodeableConcept
    system: String
    value: String!
    period: Period
    assigner: Reference
  }

  type ContactPoint {
    system: ContactPointSystem
    value: String
    use: ContactPointUse
    rank: Int
    period: Period
  }

  type UsageContext {
    code: Coding!
    valueCodeableConcept: CodeableConcept
    valueQuantity: Quantity
    valueRange: Range
    valueReference: Reference
  }

  type CodeableConcept {
    coding: [Coding!]
    text: String
  }

  type Coding {
    system: String
    version: String
    code: String
    display: String
    userSelected: Boolean
  }

  type Quantity {
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

  type Reference {
    reference: String
    type: String
    identifier: Identifier
    display: String
  }

  type Period {
    start: DateTime
    end: DateTime
  }

  type OperationResult {
    success: Boolean!
    message: String
    details: JSON
  }

  # Enumerations
  enum TerminologyStatus {
    DRAFT
    ACTIVE
    RETIRED
    UNKNOWN
  }

  enum ConceptStatus {
    ACTIVE
    INACTIVE
    ENTERED_IN_ERROR
  }

  enum ValueSetStatus {
    DRAFT
    ACTIVE
    RETIRED
    UNKNOWN
  }

  enum ContentType {
    NOT_PRESENT
    EXAMPLE
    FRAGMENT
    COMPLETE
    SUPPLEMENT
  }

  enum MappingEquivalence {
    RELATEDTO
    EQUIVALENT
    EQUAL
    WIDER
    SUBSUMES
    NARROWER
    SPECIALIZES
    INEXACT
    UNMATCHED
    DISJOINT
  }

  enum FilterOperator {
    EQUALS
    IS_A
    DESCENDENT_OF
    IS_NOT_A
    REGEX
    IN
    NOT_IN
    GENERALIZES
    EXISTS
  }

  enum ValidationSeverity {
    FATAL
    ERROR
    WARNING
    INFORMATION
  }

  enum IdentifierUse {
    USUAL
    OFFICIAL
    TEMP
    SECONDARY
    OLD
  }

  enum ContactPointSystem {
    PHONE
    FAX
    EMAIL
    PAGER
    URL
    SMS
    OTHER
  }

  enum ContactPointUse {
    HOME
    WORK
    TEMP
    OLD
    MOBILE
  }

  # Custom Scalars
  scalar DateTime
  scalar JSON
`;

module.exports = kb7TerminologyTypeDefs;