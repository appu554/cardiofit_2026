const { gql } = require('apollo-server-express');

const typeDefs = gql`
  extend schema @link(url: "https://specs.apollo.dev/federation/v2.0", import: ["@key", "@shareable"])

  # Organization type with federation support
  type Organization @key(fields: "id") {
    id: ID!
    resourceType: String!
    active: Boolean!
    
    # Organization identification
    identifier: [Identifier]
    name: String
    alias: [String]
    legalName: String
    tradingName: String
    
    # Organization classification
    organizationType: OrganizationType
    status: OrganizationStatus
    
    # Contact information
    telecom: [ContactPoint]
    address: [Address]
    websiteUrl: String
    
    # Business information
    taxId: String
    licenseNumber: String
    
    # Hierarchical relationships
    partOf: String
    
    # Verification information
    verificationStatus: String
    verificationDocuments: [String]
    verifiedBy: String
    verificationTimestamp: String
    
    # Audit information
    createdAt: String
    updatedAt: String
    createdBy: String
    updatedBy: String
  }

  # Supporting types
  type Identifier {
    use: String
    typeCode: String
    typeDisplay: String
    system: String
    value: String
    assigner: String
  }

  type ContactPoint {
    system: String
    value: String
    use: String
    rank: Int
  }

  type Address {
    use: String
    type: String
    text: String
    line: [String]
    city: String
    district: String
    state: String
    postalCode: String
    country: String
  }

  # Enums
  enum OrganizationType {
    HOSPITAL
    CLINIC
    SPECIALTY_PRACTICE
    LABORATORY
    PHARMACY
    DEPARTMENT
    HEALTHCARE_COMPANY
    INSURANCE_COMPANY
    OTHER
  }

  enum OrganizationStatus {
    ACTIVE
    INACTIVE
    SUSPENDED
    PENDING_VERIFICATION
    VERIFIED
  }

  # Input types
  input OrganizationInput {
    name: String!
    legalName: String
    tradingName: String
    organizationType: OrganizationType
    active: Boolean = true
    telecom: [ContactPointInput]
    address: [AddressInput]
    websiteUrl: String
    taxId: String
    licenseNumber: String
    partOf: String
    identifier: [IdentifierInput]
    alias: [String]
  }

  input OrganizationUpdateInput {
    name: String
    legalName: String
    tradingName: String
    organizationType: OrganizationType
    active: Boolean
    telecom: [ContactPointInput]
    address: [AddressInput]
    websiteUrl: String
    taxId: String
    licenseNumber: String
    partOf: String
    identifier: [IdentifierInput]
    alias: [String]
  }

  input IdentifierInput {
    use: String
    typeCode: String
    typeDisplay: String
    system: String
    value: String
    assigner: String
  }

  input ContactPointInput {
    system: String
    value: String
    use: String
    rank: Int
  }

  input AddressInput {
    use: String
    type: String
    text: String
    line: [String]
    city: String
    district: String
    state: String
    postalCode: String
    country: String
  }

  # Search result type
  type OrganizationSearchResult {
    organizations: [Organization!]!
    totalCount: Int!
    hasMore: Boolean!
  }

  # Extend Query type
  extend type Query {
    organization(id: ID!): Organization
    organizations(
      name: String
      organizationType: OrganizationType
      status: OrganizationStatus
      active: Boolean
    ): OrganizationSearchResult
  }

  # Extend Mutation type
  extend type Mutation {
    createOrganization(organizationData: OrganizationInput!): Organization
    updateOrganization(id: ID!, updateData: OrganizationUpdateInput!): Organization
    submitOrganizationForVerification(id: ID!, documents: [String]): Boolean
    approveOrganization(id: ID!, notes: String): Boolean
  }
`;

module.exports = typeDefs;
