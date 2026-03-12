const { gql } = require('graphql-tag');

const kb20PatientProfileTypeDefs = gql`
  extend schema @link(url: "https://specs.apollo.dev/federation/v2.0", import: ["@key", "@shareable", "@external", "@requires", "@provides"])

  extend type Query {
    # Patient Stratum
    patientStratum(patientId: ID!): PatientStratum

    # Lab Results
    patientLabResults(patientId: ID!, labType: String, limit: Int = 20): [LabResult!]!

    # eGFR Trajectory
    eGFRTrajectory(patientId: ID!, windowMonths: Int = 12): EGFRTrajectoryAnalysis

    # Plausibility Check
    checkLabPlausibility(patientId: ID!, labType: String!, value: Float!): PlausibilityResult
  }

  # Extended Patient type with profile data
  extend type Patient @key(fields: "id") {
    id: ID! @external
    stratum: PatientStratum
    latestLabs: [LabResult!]!
    eGFRTrajectory: EGFRTrajectoryAnalysis
  }

  type PatientStratum {
    patientId: ID!
    stratumLevel: String!
    ckdStage: String
    diabetesType: String
    heartFailureClass: String
    riskCategory: String!
    egfr: Float
    lastUpdated: DateTime!
  }

  type LabResult {
    id: ID!
    patientId: ID!
    labType: String!
    value: Float!
    unit: String!
    measuredAt: DateTime!
    source: String
    validationStatus: String!
    isDerived: Boolean!
  }

  type EGFRTrajectoryAnalysis {
    patientId: ID!
    slope: Float!
    classification: String!
    rSquared: Float!
    dataPoints: Int!
    windowMonths: Int!
    latestEGFR: Float
    trend: String!
  }

  type PlausibilityResult {
    verdict: String!
    confidence: Float!
    reason: String
    maxExpectedDelta: Float
    actualDelta: Float
  }

  scalar DateTime
`;

module.exports = { kb20PatientProfileTypeDefs };
