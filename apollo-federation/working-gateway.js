const { ApolloServer } = require('@apollo/server');
const { expressMiddleware } = require('@apollo/server/express4');
const { ApolloGateway } = require('@apollo/gateway');
const express = require('express');
const http = require('http');
const cors = require('cors');
const { json } = require('body-parser');
const jwt = require('jsonwebtoken');
// Import fetch with ESM syntax for node-fetch v3
const fetch = (...args) => import('node-fetch').then(({default: fetch}) => fetch(...args));
require('dotenv').config();

// Configure logging
const logger = {
  info: (message) => console.log(`[INFO] ${new Date().toISOString()} - ${message}`),
  error: (message, error) => console.error(`[ERROR] ${new Date().toISOString()} - ${message}`, error),
  warn: (message) => console.warn(`[WARN] ${new Date().toISOString()} - ${message}`),
  debug: (message) => console.debug(`[DEBUG] ${new Date().toISOString()} - ${message}`)
};

// Define the microservice endpoints - only patient service for now
const serviceList = [
  // Use the dedicated federation endpoint for the patient service
  { name: 'patients', url: (process.env.PATIENT_SERVICE_URL || 'http://localhost:8003/api').replace('/graphql', '/federation') }
];

logger.info('Initializing Apollo Federation Gateway with the following services:');
serviceList.forEach(service => {
  logger.info(`- ${service.name}: ${service.url}`);
});

// No mock authentication token - we'll use real authentication from API Gateway

// Import required modules
const { buildSubgraphSchema } = require('@apollo/subgraph');
const { gql } = require('graphql-tag');
const patientResolvers = require('./resolvers/patient-resolvers');
const fs = require('fs');
const path = require('path');

// Check if supergraph.graphql exists, otherwise use static schema
const supergraphPath = path.join(__dirname, 'supergraph.graphql');
let staticSchema;

if (fs.existsSync(supergraphPath)) {
  logger.info(`Using schema from supergraph.graphql`);
  // We'll still use our static schema but update it to match the supergraph schema
}

// Create a simple static schema without Federation directives
staticSchema = `
schema {
  query: Query
  mutation: Mutation
}

type Query {
  patients(page: Int, limit: Int, count: Int, generalPractitioner: String): PatientConnection
  patient(id: String!): Patient
  searchPatients(name: String): [Patient]
}

type Mutation {
  createPatient(input: CreatePatientInput!): CreatePatientResponse
  updatePatient(id: String!, input: UpdatePatientInput!): UpdatePatientResponse
  deletePatient(id: String!): DeleteResponse
}

type CreatePatientResponse {
  patient: Patient
}

type UpdatePatientResponse {
  patient: Patient
}

type DeleteResponse {
  success: Boolean!
  message: String
}

type PatientConnection {
  items: [Patient]
  total: Int
  page: Int
  count: Int
}

type Patient {
  id: String!
  resourceType: String
  text: TextComponent
  identifier: [Identifier]
  active: Boolean
  name: [HumanName]
  telecom: [ContactPoint]
  gender: String
  birthDate: String
  deceasedBoolean: Boolean
  address: [Address]
  maritalStatus: CodeableConcept
  multipleBirthBoolean: Boolean
  contact: [Contact]
  communication: [Communication]
  generalPractitioner: [Reference]
  managingOrganization: Reference
}

type TextComponent {
  status: String
  div: String
}

type Identifier {
  use: String
  type: CodeableConcept
  system: String
  value: String
  period: Period
  assigner: Reference
}

type HumanName {
  use: String
  family: String
  given: [String]
  prefix: [String]
  suffix: [String]
  period: Period
}

type ContactPoint {
  system: String
  value: String
  use: String
  rank: Int
  period: Period
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
  period: Period
}

type CodeableConcept {
  coding: [Coding]
  text: String
}

type Coding {
  system: String
  version: String
  code: String
  display: String
  userSelected: Boolean
}

type Period {
  start: String
  end: String
}

type Contact {
  relationship: [CodeableConcept]
  name: HumanName
  telecom: [ContactPoint]
  address: Address
  gender: String
  organization: Reference
  period: Period
}

type Communication {
  language: CodeableConcept
  preferred: Boolean
}

type Reference {
  reference: String
  type: String
  identifier: Identifier
  display: String
}

input PatientInput {
  resourceType: String
  text: TextComponentInput
  identifier: [IdentifierInput]
  active: Boolean
  name: [HumanNameInput]
  telecom: [ContactPointInput]
  gender: String
  birthDate: String
  deceasedBoolean: Boolean
  address: [AddressInput]
  maritalStatus: CodeableConceptInput
  multipleBirthBoolean: Boolean
  contact: [ContactInput]
  communication: [CommunicationInput]
  generalPractitioner: [ReferenceInput]
  managingOrganization: ReferenceInput
}

# Alias for PatientInput to match expected schema
input CreatePatientInput {
  resourceType: String
  text: TextComponentInput
  identifier: [IdentifierInput]
  active: Boolean
  name: [HumanNameInput]
  telecom: [ContactPointInput]
  gender: String
  birthDate: String
  deceasedBoolean: Boolean
  address: [AddressInput]
  maritalStatus: CodeableConceptInput
  multipleBirthBoolean: Boolean
  contact: [ContactInput]
  communication: [CommunicationInput]
  generalPractitioner: [ReferenceInput]
  managingOrganization: ReferenceInput
}

# Alias for PatientInput to match expected schema
input UpdatePatientInput {
  resourceType: String
  text: TextComponentInput
  identifier: [IdentifierInput]
  active: Boolean
  name: [HumanNameInput]
  telecom: [ContactPointInput]
  gender: String
  birthDate: String
  deceasedBoolean: Boolean
  address: [AddressInput]
  maritalStatus: CodeableConceptInput
  multipleBirthBoolean: Boolean
  contact: [ContactInput]
  communication: [CommunicationInput]
  generalPractitioner: [ReferenceInput]
  managingOrganization: ReferenceInput
}

input TextComponentInput {
  status: String
  div: String
}

input IdentifierInput {
  use: String
  type: CodeableConceptInput
  system: String
  value: String
  period: PeriodInput
  assigner: ReferenceInput
}

input HumanNameInput {
  use: String
  family: String
  given: [String]
  prefix: [String]
  suffix: [String]
  period: PeriodInput
}

input ContactPointInput {
  system: String
  value: String
  use: String
  rank: Int
  period: PeriodInput
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
  period: PeriodInput
}

input CodeableConceptInput {
  coding: [CodingInput]
  text: String
}

input CodingInput {
  system: String
  version: String
  code: String
  display: String
  userSelected: Boolean
}

input PeriodInput {
  start: String
  end: String
}

input ContactInput {
  relationship: [CodeableConceptInput]
  name: HumanNameInput
  telecom: [ContactPointInput]
  address: AddressInput
  gender: String
  organization: ReferenceInput
  period: PeriodInput
}

input CommunicationInput {
  language: CodeableConceptInput
  preferred: Boolean
}

input ReferenceInput {
  reference: String
  type: String
  identifier: IdentifierInput
  display: String
}
`;

// Parse the static schema
const typeDefs = gql(staticSchema);

// Create a standalone Apollo Server with resolvers
const server = {
  schema: buildSubgraphSchema({
    typeDefs,
    resolvers: patientResolvers
  })
};

// Initialize Express variables
let app;
let httpServer;

// Create a context builder function
function buildContext({ req }) {
  // Get the authorization header
  const token = req.headers.authorization;

  // Check for user information headers from API Gateway
  const userId = req.headers['x-user-id'];
  const userEmail = req.headers['x-user-email'];
  const userName = req.headers['x-user-name'];
  const userRole = req.headers['x-user-role'];
  const userRolesStr = req.headers['x-user-roles'];
  const userPermissionsStr = req.headers['x-user-permissions'];

  // Parse roles and permissions from comma-separated strings
  const userRoles = userRolesStr ? userRolesStr.split(',') : [];
  const userPermissions = userPermissionsStr ? userPermissionsStr.split(',') : [];

  // Create a context object with the token and user info
  const context = {
    token,
    userId,
    userEmail,
    userName,
    userRole,
    userRoles,
    userPermissions
  };

  // If we have user information from the API Gateway headers, use that
  if (userId && userRole) {
    logger.info(`Using user information from API Gateway headers: User ${userId} with role ${userRole}`);
    return context;
  }

  // If no API Gateway headers but we have a token, try to extract user info from it
  if (token && token.startsWith('Bearer ')) {
    try {
      // Extract the token
      const jwtToken = token.replace('Bearer ', '');

      // Verify the token if JWT_SECRET is provided
      let decoded;
      if (process.env.JWT_SECRET) {
        decoded = jwt.verify(jwtToken, process.env.JWT_SECRET);
        logger.debug('Token verified successfully');
      } else {
        // Fallback to decode without verification
        decoded = jwt.decode(jwtToken);
        logger.debug('JWT_SECRET not provided, token decoded without verification');
      }

      if (decoded) {
        // Extract standard claims
        context.userId = decoded.sub || context.userId;
        context.userRole = decoded.role || context.userRole;
        context.userEmail = decoded.email || context.userEmail;

        // Extract additional claims if available
        if (decoded.app_metadata && decoded.app_metadata.roles) {
          context.userRoles = decoded.app_metadata.roles;
        }
        if (decoded.app_metadata && decoded.app_metadata.permissions) {
          context.userPermissions = decoded.app_metadata.permissions;
        }

        logger.info(`User context built from token for user ${context.userId} with role ${context.userRole}`);
        return context;
      }
    } catch (error) {
      logger.error('Error processing token:', error);
    }
  }

  // If we get here, we don't have valid user information
  logger.warn('No valid authentication information found, using anonymous context');
  return {
    token: null,
    userId: null,
    userRole: null,
    userRoles: [],
    userPermissions: []
  };
}

// Start the server
async function startServer() {
  // Initialize Express
  app = express();
  httpServer = http.createServer(app);

  // Create Apollo Server with our schema and resolvers
  const apolloServer = new ApolloServer({
    schema: server.schema,
    formatError: (formattedError, error) => {
      // Log the error
      logger.error('GraphQL Error:', error);

      // Return a sanitized error in production
      if (process.env.NODE_ENV === 'production') {
        return {
          message: formattedError.message,
          extensions: {
            code: formattedError.extensions?.code || 'INTERNAL_SERVER_ERROR'
          }
        };
      }

      // Return the full error in development
      return formattedError;
    },
    // Apollo Server v4 options
    introspection: true,
    // Enable Apollo Sandbox for query building
    plugins: [
      {
        async serverWillStart() {
          logger.info('Apollo Sandbox enabled at http://localhost:4000/graphql');
          return {};
        }
      }
    ]
  });

  // Start Apollo Server
  await apolloServer.start();
  logger.info('Apollo Server started successfully');

  // Apply middleware for both /graphql and /api/graphql paths
  const graphqlMiddleware = expressMiddleware(apolloServer, {
    context: buildContext
  });

  // Handle direct /graphql path
  app.use('/graphql', cors(), json({ limit: '2mb' }), graphqlMiddleware);

  // Also handle /api/graphql path for API Gateway compatibility
  app.use('/api/graphql', cors(), json({ limit: '2mb' }), graphqlMiddleware);

  // Health check handler function
  const healthCheckHandler = async (req, res) => {
    try {
      res.json({
        status: 'ok',
        timestamp: new Date().toISOString(),
        message: 'Using static schema, no service health checks needed'
      });
    } catch (error) {
      logger.error('Health check error:', error);
      res.status(500).json({
        status: 'error',
        message: error.message
      });
    }
  };

  // Health check endpoints - both /health and /api/health for API Gateway compatibility
  app.get('/health', healthCheckHandler);
  app.get('/api/health', healthCheckHandler);

  // Add a dedicated route for Apollo Sandbox
  app.get('/sandbox', (req, res) => {
    res.send(`
      <!DOCTYPE html>
      <html>
      <head>
        <title>Apollo Federation Gateway - GraphQL Sandbox</title>
        <style>
          body { font-family: Arial, sans-serif; margin: 0; padding: 20px; }
          h1 { color: #333; }
          .container { max-width: 800px; margin: 0 auto; }
          .button {
            display: inline-block;
            background-color: #3f51b5;
            color: white;
            padding: 10px 20px;
            text-decoration: none;
            border-radius: 4px;
            margin-top: 20px;
          }
        </style>
      </head>
      <body>
        <div class="container">
          <h1>Apollo Federation Gateway - GraphQL Sandbox</h1>
          <p>Use the button below to open the Apollo Sandbox to build and test GraphQL queries:</p>
          <a class="button" href="/graphql" target="_blank">Open Apollo Sandbox</a>
        </div>
      </body>
      </html>
    `);
  });

  // Add monitoring endpoint
  app.get('/metrics', (req, res) => {
    res.json({
      uptime: process.uptime(),
      memory: process.memoryUsage(),
      timestamp: new Date().toISOString()
    });
  });

  // Start the HTTP server
  const PORT = process.env.PORT || 4000;
  httpServer.listen(PORT, () => {
    logger.info(`🚀 Apollo Federation Server ready at http://localhost:${PORT}/graphql`);
    logger.info(`API Gateway compatible endpoint at http://localhost:${PORT}/api/graphql`);
    logger.info(`GraphQL Sandbox available at http://localhost:${PORT}/sandbox`);
    logger.info(`Health check available at http://localhost:${PORT}/health`);
    logger.info(`Metrics available at http://localhost:${PORT}/metrics`);
  });

  // Handle graceful shutdown
  process.on('SIGTERM', () => {
    logger.info('SIGTERM received, shutting down gracefully');
    httpServer.close(() => {
      logger.info('HTTP server closed');
      process.exit(0);
    });
  });
}

startServer().catch((err) => {
  logger.error('Error starting server:', err);
  process.exit(1);
});
