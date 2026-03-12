const { gql } = require('graphql-tag');

const kb23DecisionCardsTypeDefs = gql`
  extend schema @link(url: "https://specs.apollo.dev/federation/v2.0", import: ["@key", "@shareable", "@external", "@requires", "@provides"])

  extend type Query {
    # Decision Card Queries
    activeDecisionCards(patientId: ID!): [DecisionCard!]!
    decisionCard(cardId: ID!): DecisionCard
    decisionCardHistory(patientId: ID!, limit: Int = 20, status: CardStatus): [DecisionCard!]!

    # SLA Monitoring
    slaBreachedCards(limit: Int = 50): [DecisionCard!]!
    pendingReaffirmationCards(limit: Int = 50): [DecisionCard!]!
  }

  # Extended Patient type with decision cards
  extend type Patient @key(fields: "id") {
    id: ID! @external
    activeDecisionCards: [DecisionCard!]!
    pendingCards: [DecisionCard!]!
  }

  type DecisionCard {
    cardId: ID!
    patientId: ID!
    sessionId: ID
    templateId: String!
    nodeId: String!

    # Diagnostic Assessment
    primaryDifferentialId: String!
    primaryPosterior: Float!
    diagnosticConfidenceTier: ConfidenceTier!
    confidenceTierDecayed: Boolean!
    confidenceTierDecayReason: String
    secondaryDifferentials: JSON

    # MCU Gate
    mcuGate: MCUGate!
    mcuGateRationale: String
    doseAdjustmentNotes: String

    # Observation Quality
    observationReliability: ObservationReliability!

    # CTL Panels
    patientStateSnapshot: JSON
    guidelineConditionStatus: ConditionStatus
    safetyCheckSummary: JSON
    reasoningChain: JSON

    # Summaries
    clinicianSummary: String!
    patientSummaryEn: String!
    patientSummaryHi: String!
    patientSummaryLocal: String
    patientSafetyInstructions: JSON

    # Safety & Lifecycle
    safetyTier: SafetyTier!
    status: CardStatus!
    cardSource: CardSource!
    recurrenceCount: Int!
    pendingReaffirmation: Boolean!
    reEntryProtocol: Boolean!

    # SLA
    slaDeadline: DateTime
    slaBreached: Boolean!
    slaBreachedAt: DateTime
    escalatedTo: String

    # Timestamps
    createdAt: DateTime!
    updatedAt: DateTime!
    supersededAt: DateTime
    supersededBy: ID

    # Recommendations
    recommendations: [CardRecommendation!]!
  }

  type CardRecommendation {
    recommendationId: ID!
    cardId: ID!
    type: RecommendationType!
    urgency: Urgency!
    title: String!
    detail: String!
    guidelineRef: String
    evidenceLevel: String
    sortOrder: Int!
  }

  enum ConfidenceTier { FIRM PROBABLE POSSIBLE UNCERTAIN }
  enum MCUGate { SAFE MODIFY PAUSE HALT }
  enum ObservationReliability { HIGH MODERATE LOW }
  enum SafetyTier { IMMEDIATE URGENT ROUTINE }
  enum CardStatus { ACTIVE SUPERSEDED PENDING_REAFFIRMATION ARCHIVED }
  enum CardSource { KB22_SESSION HYPOGLYCAEMIA_FAST_PATH PERTURBATION_DECAY BEHAVIORAL_GAP }
  enum ConditionStatus { CRITERIA_MET CRITERIA_PARTIAL CRITERIA_NOT_MET }
  enum RecommendationType {
    INVESTIGATION REFERRAL MONITORING
    MEDICATION_HOLD MEDICATION_MODIFY MEDICATION_CONTINUE
    LIFESTYLE SAFETY_INSTRUCTION MEDICATION_REVIEW
  }
  enum Urgency { IMMEDIATE URGENT ROUTINE SCHEDULED }

  scalar DateTime
  scalar JSON
`;

module.exports = { kb23DecisionCardsTypeDefs };
