const { gql } = require('graphql-tag');

const kb2ClinicalContextTypeDefs = gql`
  extend schema @link(url: "https://specs.apollo.dev/federation/v2.0", import: ["@key", "@shareable", "@external", "@requires", "@provides"])

  extend type Query {
    # Clinical Context Operations
    evaluatePatientPhenotypes(input: PhenotypeEvaluationInput!): PhenotypeEvaluationResponse!
    explainPhenotypes(input: PhenotypeEvaluationInput!): PhenotypeExplanationResponse!
    assessPatientRisk(input: RiskAssessmentInput!): RiskAssessmentResponse!
    getPatientTreatmentPreferences(input: TreatmentPreferencesInput!): TreatmentPreferencesResponse!
    assemblePatientContext(input: ClinicalContextInput!): ClinicalContextResponse!
    
    # Phenotype Management
    availablePhenotypes(category: String): [PhenotypeDefinition!]!
    patientContextHistory(patientId: ID!, limit: Int = 10): [ClinicalContextHistory!]!
  }

  # Extended Patient type with clinical context
  extend type Patient @key(fields: "id") {
    id: ID! @external
    clinicalContext: ClinicalContext
    phenotypes: [ClinicalPhenotype!]!
    riskAssessments: [RiskAssessment!]!
    treatmentPreferences: [TreatmentPreference!]!
  }

  # Core Clinical Intelligence Types
  type ClinicalContext {
    patientId: ID!
    phenotypes: [ClinicalPhenotype!]!
    riskAssessments: [RiskAssessment!]!
    treatmentPreferences: [TreatmentPreference!]!
    contextMetadata: ContextMetadata!
    assemblyTime: DateTime!
    detailLevel: String!
  }

  type ClinicalPhenotype {
    id: ID!
    name: String!
    category: String!
    domain: String!
    priority: Int!
    matched: Boolean!
    confidence: Float!
    celRule: String!
    implications: [ClinicalImplication!]!
    evaluationDetails: PhenotypeEvaluationDetails
    lastEvaluated: DateTime!
  }

  type ClinicalImplication {
    type: String!
    severity: ImplicationSeverity!
    description: String!
    recommendations: [String!]!
    clinicalEvidence: String
  }

  type PhenotypeEvaluationDetails {
    evaluationPath: [String!]!
    factorsConsidered: [EvaluationFactor!]!
    celExpression: String!
    executionTime: String!
  }

  type EvaluationFactor {
    name: String!
    value: String!
    weight: Float
    contribution: String!
  }

  type RiskAssessment {
    id: ID!
    model: String!
    category: RiskCategory!
    score: Float!
    percentile: Float
    category_result: RiskLevel!
    recommendations: [RiskRecommendation!]!
    riskFactors: [RiskFactor!]!
    calculationMethod: String!
    validUntil: DateTime
    lastCalculated: DateTime!
  }

  type RiskRecommendation {
    priority: Int!
    action: String!
    rationale: String!
    urgency: String!
    clinicalEvidence: String
  }

  type RiskFactor {
    name: String!
    value: String!
    contribution: Float!
    modifiable: Boolean!
    severity: RiskFactorSeverity!
  }

  type TreatmentPreference {
    id: ID!
    condition: String!
    firstLine: [MedicationPreference!]!
    alternatives: [MedicationPreference!]!
    avoid: [MedicationConstraint!]!
    rationale: String!
    guidelineSource: String!
    confidenceLevel: Float!
    lastUpdated: DateTime!
  }

  type MedicationPreference {
    medication: MedicationReference!
    preferenceScore: Float!
    dosageForm: String
    frequency: String
    costTier: Int
    reasons: [String!]!
  }

  type MedicationReference {
    id: ID!
    name: String!
    genericName: String!
    brandNames: [String!]!
    drugClass: String!
    mechanism: String
  }

  type MedicationConstraint {
    medication: MedicationReference!
    constraintType: ConstraintType!
    severity: ConstraintSeverity!
    reason: String!
    alternatives: [MedicationReference!]!
  }

  type PhenotypeDefinition {
    id: ID!
    name: String!
    category: String!
    description: String!
    celRule: String!
    requiredData: [String!]!
    clinicalDomain: String!
    priority: Int!
    active: Boolean!
  }

  type ClinicalContextHistory {
    id: ID!
    patientId: ID!
    contextSnapshot: ClinicalContext!
    changesFromPrevious: [ContextChange!]!
    createdAt: DateTime!
    triggeredBy: String
  }

  type ContextChange {
    field: String!
    previousValue: String
    newValue: String!
    changeType: ChangeType!
    significance: ChangeSeverity!
  }

  type ContextMetadata {
    processingTime: String!
    slaCompliant: Boolean!
    dataCompleteness: Float!
    confidenceScore: Float!
    componentsEvaluated: [String!]!
    cacheHit: Boolean!
  }

  # Response Types
  type PhenotypeEvaluationResponse {
    results: [PatientPhenotypeResult!]!
    processingTime: String!
    batchSize: Int!
    slaCompliant: Boolean!
    metadata: ProcessingMetadata!
  }

  type PatientPhenotypeResult {
    patientId: ID!
    phenotypes: [ClinicalPhenotype!]!
    evaluationSummary: EvaluationSummary!
  }

  type EvaluationSummary {
    totalPhenotypes: Int!
    matchedPhenotypes: Int!
    highConfidenceMatches: Int!
    averageConfidence: Float!
    processingTime: String!
  }

  type PhenotypeExplanationResponse {
    results: [PatientPhenotypeExplanation!]!
    processingTime: String!
    slaCompliant: Boolean!
  }

  type PatientPhenotypeExplanation {
    patientId: ID!
    explanations: [PhenotypeExplanation!]!
  }

  type PhenotypeExplanation {
    phenotype: ClinicalPhenotype!
    reasoningChain: [ReasoningStep!]!
    decisionFactors: [DecisionFactor!]!
    alternativeOutcomes: [AlternativeOutcome!]!
  }

  type ReasoningStep {
    stepNumber: Int!
    description: String!
    celExpression: String!
    result: Boolean!
    dataUsed: [String!]!
  }

  type DecisionFactor {
    factor: String!
    value: String!
    weight: Float!
    influence: String!
  }

  type AlternativeOutcome {
    scenario: String!
    outcome: Boolean!
    probability: Float!
    explanation: String!
  }

  type RiskAssessmentResponse {
    patientId: ID!
    riskAssessments: [RiskAssessment!]!
    overallRiskProfile: RiskProfile!
    processingTime: String!
    slaCompliant: Boolean!
  }

  type RiskProfile {
    overallRisk: RiskLevel!
    primaryConcerns: [String!]!
    riskDistribution: [RiskCategoryScore!]!
    recommendedActions: [String!]!
  }

  type RiskCategoryScore {
    category: RiskCategory!
    score: Float!
    level: RiskLevel!
    trend: String
  }

  type TreatmentPreferencesResponse {
    patientId: ID!
    condition: String!
    preferences: TreatmentPreference!
    alternativeOptions: [TreatmentOption!]!
    conflictResolution: [ConflictResolution!]!
    processingTime: String!
  }

  type TreatmentOption {
    medication: MedicationReference!
    suitabilityScore: Float!
    rationale: String!
    considerations: [String!]!
  }

  type ConflictResolution {
    conflictType: String!
    resolution: String!
    priority: Int!
    reasoning: String!
  }

  type ClinicalContextResponse {
    patientId: ID!
    context: ClinicalContext!
    warnings: [ContextWarning!]!
    recommendations: [ContextRecommendation!]!
    processingTime: String!
    slaCompliant: Boolean!
  }

  type ContextWarning {
    severity: WarningSeverity!
    category: String!
    message: String!
    actionRequired: String
  }

  type ContextRecommendation {
    priority: Int!
    category: String!
    recommendation: String!
    rationale: String!
    timeframe: String
  }

  type ProcessingMetadata {
    cacheHitRate: Float!
    averageProcessingTime: String!
    componentsProcessed: [String!]!
    errorCount: Int!
  }

  # Input Types
  input PhenotypeEvaluationInput {
    patients: [PatientClinicalData!]!
    phenotypeIds: [String!]
    includeExplanation: Boolean = false
    includeImplications: Boolean = true
    confidenceThreshold: Float = 0.7
  }

  input PatientClinicalData {
    id: ID!
    age: Int!
    gender: String!
    conditions: [String!]!
    medications: [String!]
    labs: [LabValue!]
    vitals: [VitalSign!]
    procedures: [String!]
    allergies: [String!]
    familyHistory: [String!]
    socialHistory: SocialHistoryInput
  }

  input LabValue {
    name: String!
    value: Float!
    unit: String!
    referenceRange: String
    testDate: DateTime
  }

  input VitalSign {
    name: String!
    value: Float!
    unit: String!
    measurementDate: DateTime
  }

  input SocialHistoryInput {
    smokingStatus: String
    alcoholUse: String
    exerciseFrequency: String
    dietaryPatterns: String
  }

  input RiskAssessmentInput {
    patientId: ID!
    patientData: PatientClinicalData!
    riskCategories: [RiskCategory!]!
    includeFactors: Boolean = true
    includeRecommendations: Boolean = true
  }

  input TreatmentPreferencesInput {
    patientId: ID!
    condition: String!
    patientData: PatientClinicalData!
    preferenceProfile: PreferenceProfileInput
    includeAlternatives: Boolean = true
  }

  input PreferenceProfileInput {
    onceDailyPreferred: Boolean
    injectableAccepted: Boolean
    costConscious: Boolean
    brandPreference: String
    dosageFormPreferences: [String!]
  }

  input ClinicalContextInput {
    patientId: ID!
    patientData: PatientClinicalData!
    detailLevel: ContextDetailLevel = COMPREHENSIVE
    includePhenotypes: Boolean = true
    includeRisks: Boolean = true
    includeTreatments: Boolean = true
    useCache: Boolean = true
  }

  # Enumerations
  enum RiskCategory {
    CARDIOVASCULAR
    DIABETES
    MEDICATION
    FALL
    BLEEDING
    KIDNEY
    LIVER
    RESPIRATORY
    COGNITIVE
    GENERAL
  }

  enum RiskLevel {
    LOW
    MODERATE
    HIGH
    VERY_HIGH
    EXTREME
  }

  enum RiskFactorSeverity {
    MINOR
    MODERATE
    MAJOR
    CRITICAL
  }

  enum ImplicationSeverity {
    INFORMATIONAL
    LOW
    MODERATE
    HIGH
    CRITICAL
  }

  enum ConstraintType {
    CONTRAINDICATION
    CAUTION
    PREFERENCE
    ALLERGY
    INTERACTION
  }

  enum ConstraintSeverity {
    MINOR
    MODERATE
    MAJOR
    ABSOLUTE
  }

  enum ContextDetailLevel {
    SUMMARY
    STANDARD
    COMPREHENSIVE
    DETAILED
  }

  enum ChangeType {
    ADDED
    MODIFIED
    REMOVED
    STATUS_CHANGE
  }

  enum ChangeSeverity {
    TRIVIAL
    MINOR
    MODERATE
    SIGNIFICANT
    CRITICAL
  }

  enum WarningSeverity {
    INFO
    WARNING
    ERROR
    CRITICAL
  }

  # Custom Scalars
  scalar DateTime
  scalar JSON
`;

module.exports = kb2ClinicalContextTypeDefs;