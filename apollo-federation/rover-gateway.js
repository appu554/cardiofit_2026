/**
 * Apollo Federation Gateway using Rover-generated supergraph schema
 *
 * This implementation uses a static schema approach with resolvers that forward
 * requests to the appropriate microservices.
 */

const { ApolloServer } = require('@apollo/server');
const { expressMiddleware } = require('@apollo/server/express4');
const express = require('express');
const http = require('http');
const cors = require('cors');
const { json } = require('body-parser');
const jwt = require('jsonwebtoken');
const fs = require('fs');
const path = require('path');
const { buildSchema } = require('graphql');
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

// Define the microservice endpoints
const serviceList = [
  // Use the dedicated federation endpoint for the patient service
  { name: 'patients', url: (process.env.PATIENT_SERVICE_URL || 'http://localhost:8003/api').replace('/graphql', '/federation') },
  // Use the dedicated federation endpoint for the observation service
  { name: 'observations', url: (process.env.OBSERVATION_SERVICE_URL || 'http://localhost:8007/api').replace('/graphql', '/federation') },
  // Use the dedicated federation endpoint for the medication service
  { name: 'medications', url: (process.env.MEDICATION_SERVICE_URL || 'http://localhost:8005').replace('/graphql', '/federation') },
  // Use the dedicated federation endpoint for the organization service
  { name: 'organizations', url: (process.env.ORGANIZATION_SERVICE_URL || 'http://localhost:8012/api').replace('/graphql', '/federation') },
  // Use the dedicated federation endpoint for the order management service
  { name: 'orders', url: (process.env.ORDER_MANAGEMENT_SERVICE_URL || 'http://localhost:8013/api/federation') },
  // Use the dedicated federation endpoint for the scheduling service
  { name: 'scheduling', url: (process.env.SCHEDULING_SERVICE_URL || 'http://localhost:8014/api/federation') },
  // Use the dedicated federation endpoint for the encounter management service
  { name: 'encounters', url: (process.env.ENCOUNTER_SERVICE_URL || 'http://localhost:8020/api/federation') },
  // Use the dedicated federation endpoint for the workflow engine service
  { name: 'workflows', url: (process.env.WORKFLOW_ENGINE_SERVICE_URL || 'http://localhost:8015/api/federation') }
];

logger.info('Initializing Apollo Federation Gateway with the following services:');
serviceList.forEach(service => {
  logger.info(`- ${service.name}: ${service.url}`);
});

// Create custom resolvers that forward requests to the patient service
const patientServiceUrl = (process.env.PATIENT_SERVICE_URL || 'http://localhost:8003/api/graphql');

// Custom resolvers that forward requests to the patient service
const resolvers = {
  Query: {
    patients: async (_, args, context) => {
      logger.info(`Forwarding patients query to ${patientServiceUrl}`);
      logger.info(`Query arguments: ${JSON.stringify(args)}`);
      logger.info(`Context: ${JSON.stringify({
        userId: context.userId,
        userRole: context.userRole,
        userRoles: context.userRoles,
        userPermissions: context.userPermissions
      })}`);

      try {
        const requestBody = JSON.stringify({
          query: `
            query GetPatients($page: Int, $limit: Int, $count: Int, $generalPractitioner: String) {
              patients(page: $page, limit: $limit, count: $count, generalPractitioner: $generalPractitioner) {
                items {
                  id
                  resourceType
                  name {
                    family
                    given
                  }
                  gender
                  birthDate
                  telecom {
                    system
                    value
                  }
                  address {
                    line
                    city
                    state
                    postalCode
                    country
                  }
                }
                total
                page
                count
              }
            }
          `,
          variables: args
        });

        logger.info(`Request body: ${requestBody}`);

        const response = await fetch(patientServiceUrl, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': context.token,
            'X-User-ID': context.userId,
            'X-User-Role': context.userRole,
            'X-User-Roles': context.userRoles.join(','),
            'X-User-Permissions': context.userPermissions.join(',')
          },
          body: JSON.stringify({
            query: `
              query GetPatients($page: Int, $limit: Int, $count: Int, $generalPractitioner: String) {
                patients(page: $page, limit: $limit, count: $count, generalPractitioner: $generalPractitioner) {
                  items {
                    id
                    resourceType
                    name {
                      family
                      given
                    }
                    gender
                    birthDate
                    telecom {
                      system
                      value
                    }
                    address {
                      line
                      city
                      state
                      postalCode
                      country
                    }
                  }
                  total
                  page
                  count
                }
              }
            `,
            variables: args
          })
        });

        logger.info(`Response status: ${response.status}`);
        logger.info(`Response headers: ${JSON.stringify([...response.headers.entries()])}`);

        const responseText = await response.text();
        logger.info(`Response text: ${responseText}`);

        let data;
        try {
          data = JSON.parse(responseText);
          logger.info(`Patient service response: ${JSON.stringify(data)}`);

          if (data.errors) {
            logger.error(`Error from patient service: ${JSON.stringify(data.errors)}`);
            throw new Error(data.errors[0].message);
          }

          return data.data.patients;
        } catch (parseError) {
          logger.error(`Error parsing response: ${parseError.message}`);
          throw new Error(`Failed to parse response from patient service: ${parseError.message}`);
        }
      } catch (error) {
        logger.error(`Error forwarding patients query: ${error.message}`);
        throw error;
      }
    },

    patient: async (_, { id }, context) => {
      logger.info(`Forwarding patient query for id ${id} to ${patientServiceUrl}`);
      try {
        const response = await fetch(patientServiceUrl, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': context.token,
            'X-User-ID': context.userId,
            'X-User-Role': context.userRole,
            'X-User-Roles': context.userRoles.join(','),
            'X-User-Permissions': context.userPermissions.join(',')
          },
          body: JSON.stringify({
            query: `
              query GetPatient($id: String!) {
                patient(id: $id) {
                  id
                  resourceType
                  name {
                    family
                    given
                  }
                  gender
                  birthDate
                  telecom {
                    system
                    value
                  }
                  address {
                    line
                    city
                    state
                    postalCode
                    country
                  }
                  maritalStatus {
                    text
                  }
                  communication {
                    language {
                      text
                    }
                    preferred
                  }
                  generalPractitioner {
                    reference
                    display
                  }
                  managingOrganization {
                    reference
                    display
                  }
                }
              }
            `,
            variables: { id }
          })
        });

        const data = await response.json();
        logger.debug(`Patient service response: ${JSON.stringify(data)}`);

        if (data.errors) {
          logger.error(`Error from patient service: ${JSON.stringify(data.errors)}`);
          throw new Error(data.errors[0].message);
        }

        return data.data.patient;
      } catch (error) {
        logger.error(`Error forwarding patient query: ${error.message}`);
        throw error;
      }
    },

    searchPatients: async (_, { name }, context) => {
      logger.info(`Forwarding searchPatients query for name ${name} to ${patientServiceUrl}`);
      try {
        const response = await fetch(patientServiceUrl, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': context.token,
            'X-User-ID': context.userId,
            'X-User-Role': context.userRole,
            'X-User-Roles': context.userRoles.join(','),
            'X-User-Permissions': context.userPermissions.join(',')
          },
          body: JSON.stringify({
            query: `
              query SearchPatients($name: String) {
                searchPatients(name: $name) {
                  id
                  resourceType
                  name {
                    family
                    given
                  }
                  gender
                  birthDate
                }
              }
            `,
            variables: { name }
          })
        });

        const data = await response.json();
        logger.debug(`Patient service response: ${JSON.stringify(data)}`);

        if (data.errors) {
          logger.error(`Error from patient service: ${JSON.stringify(data.errors)}`);
          throw new Error(data.errors[0].message);
        }

        return data.data.searchPatients;
      } catch (error) {
        logger.error(`Error forwarding searchPatients query: ${error.message}`);
        throw error;
      }
    }
  },

  Mutation: {
    createPatient: async (_, { patientData }, context) => {
      logger.info(`Forwarding createPatient mutation to ${patientServiceUrl}`);
      try {
        const response = await fetch(patientServiceUrl, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': context.token,
            'X-User-ID': context.userId,
            'X-User-Role': context.userRole,
            'X-User-Roles': context.userRoles.join(','),
            'X-User-Permissions': context.userPermissions.join(',')
          },
          body: JSON.stringify({
            query: `
              mutation CreatePatient($patientData: PatientInput!) {
                createPatient(patientData: $patientData) {
                  id
                  resourceType
                  name {
                    family
                    given
                  }
                  gender
                  birthDate
                }
              }
            `,
            variables: { patientData }
          })
        });

        const data = await response.json();
        logger.debug(`Patient service response: ${JSON.stringify(data)}`);

        if (data.errors) {
          logger.error(`Error from patient service: ${JSON.stringify(data.errors)}`);
          throw new Error(data.errors[0].message);
        }

        return data.data.createPatient;
      } catch (error) {
        logger.error(`Error forwarding createPatient mutation: ${error.message}`);
        throw error;
      }
    },

    updatePatient: async (_, { id, patientData }, context) => {
      logger.info(`Forwarding updatePatient mutation for id ${id} to ${patientServiceUrl}`);
      try {
        const response = await fetch(patientServiceUrl, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': context.token,
            'X-User-ID': context.userId,
            'X-User-Role': context.userRole,
            'X-User-Roles': context.userRoles.join(','),
            'X-User-Permissions': context.userPermissions.join(',')
          },
          body: JSON.stringify({
            query: `
              mutation UpdatePatient($id: String!, $patientData: PatientInput!) {
                updatePatient(id: $id, patientData: $patientData) {
                  id
                  resourceType
                  name {
                    family
                    given
                  }
                  gender
                  birthDate
                }
              }
            `,
            variables: { id, patientData }
          })
        });

        const data = await response.json();
        logger.debug(`Patient service response: ${JSON.stringify(data)}`);

        if (data.errors) {
          logger.error(`Error from patient service: ${JSON.stringify(data.errors)}`);
          throw new Error(data.errors[0].message);
        }

        return data.data.updatePatient;
      } catch (error) {
        logger.error(`Error forwarding updatePatient mutation: ${error.message}`);
        throw error;
      }
    },

    deletePatient: async (_, { id }, context) => {
      logger.info(`Forwarding deletePatient mutation for id ${id} to ${patientServiceUrl}`);
      try {
        const response = await fetch(patientServiceUrl, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Authorization': context.token,
            'X-User-ID': context.userId,
            'X-User-Role': context.userRole,
            'X-User-Roles': context.userRoles.join(','),
            'X-User-Permissions': context.userPermissions.join(',')
          },
          body: JSON.stringify({
            query: `
              mutation DeletePatient($id: String!) {
                deletePatient(id: $id) {
                  success
                  message
                }
              }
            `,
            variables: { id }
          })
        });

        const data = await response.json();
        logger.debug(`Patient service response: ${JSON.stringify(data)}`);

        if (data.errors) {
          logger.error(`Error from patient service: ${JSON.stringify(data.errors)}`);
          throw new Error(data.errors[0].message);
        }

        return data.data.deletePatient;
      } catch (error) {
        logger.error(`Error forwarding deletePatient mutation: ${error.message}`);
        throw error;
      }
    }
  }
};

// Load the supergraph schema
const supergraphPath = path.join(__dirname, 'supergraph.graphql');
let schema;

if (fs.existsSync(supergraphPath)) {
  logger.info(`Loading supergraph schema from ${supergraphPath}`);
  const supergraphSdl = fs.readFileSync(supergraphPath, 'utf8');

  try {
    // Build a GraphQL schema from the supergraph SDL
    schema = buildSchema(supergraphSdl);
    logger.info('Successfully built schema from supergraph SDL');
  } catch (error) {
    logger.error('Error building schema from supergraph SDL:', error);
    process.exit(1);
  }
} else {
  logger.error(`Supergraph schema not found at ${supergraphPath}`);
  logger.error('Please run "npm run rover-supergraph" to generate the supergraph schema');
  process.exit(1);
}

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
    logger.debug(`User roles: ${JSON.stringify(userRoles)}`);
    logger.debug(`User permissions: ${JSON.stringify(userPermissions)}`);
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
  const app = express();
  const httpServer = http.createServer(app);

  // Create Apollo Server with our schema and resolvers
  const apolloServer = new ApolloServer({
    schema,
    resolvers,
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
  const graphqlMiddleware = [
    cors(),
    json({ limit: '2mb' }),
    expressMiddleware(apolloServer, {
      context: buildContext
    })
  ];

  // Handle direct /graphql path
  app.use('/graphql', ...graphqlMiddleware);

  // Also handle /api/graphql path for API Gateway compatibility
  app.use('/api/graphql', ...graphqlMiddleware);

  // Health check handler function
  const healthCheckHandler = async (req, res) => {
    try {
      res.json({
        status: 'ok',
        timestamp: new Date().toISOString(),
        message: 'Using Rover-generated supergraph schema'
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

  // Start the HTTP server
  const PORT = process.env.PORT || 4000;
  httpServer.listen(PORT, () => {
    logger.info(`🚀 Apollo Federation Server ready at http://localhost:${PORT}/graphql`);
    logger.info(`API Gateway compatible endpoint at http://localhost:${PORT}/api/graphql`);
    logger.info(`Health check available at http://localhost:${PORT}/health`);
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
