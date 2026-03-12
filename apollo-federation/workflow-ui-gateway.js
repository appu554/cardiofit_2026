/**
 * Enhanced Apollo Federation Gateway with Workflow UI Interaction Support
 * Includes real-time subscriptions and clinical override management
 */

const { ApolloServer } = require('@apollo/server');
const { expressMiddleware } = require('@apollo/server/express4');
const { ApolloGateway, IntrospectAndCompose } = require('@apollo/gateway');
const { makeExecutableSchema } = require('@graphql-tools/schema');
const { stitchSchemas } = require('@graphql-tools/stitch');
const express = require('express');
const http = require('http');
const cors = require('cors');
const { json } = require('body-parser');
const fs = require('fs');
const path = require('path');
const { PubSub } = require('graphql-subscriptions');
const { useServer } = require('graphql-ws/lib/use/ws');
const { WebSocketServer } = require('ws');
const { createServer } = require('http');

// Import the UI interaction resolvers
const workflowUIResolvers = require('./resolvers/workflow-ui-interaction-resolver');

require('dotenv').config();

// Configure logging
const logger = {
  info: (message) => console.log(`[INFO] ${new Date().toISOString()} - ${message}`),
  error: (message, error) => console.error(`[ERROR] ${new Date().toISOString()} - ${message}`, error?.message || error),
  warn: (message) => console.warn(`[WARN] ${new Date().toISOString()} - ${message}`),
  debug: (message) => console.debug(`[DEBUG] ${new Date().toISOString()} - ${message}`)
};

// Initialize PubSub for subscriptions
const pubsub = new PubSub();

// Load all schema files
function loadSchemas() {
  const schemaDir = path.join(__dirname, 'schemas');
  const schemas = {
    orchestration: fs.readFileSync(path.join(schemaDir, 'workflow-orchestration-schema.graphql'), 'utf8'),
    uiInteraction: fs.readFileSync(path.join(schemaDir, 'workflow-ui-interaction-schema.graphql'), 'utf8')
  };

  logger.info('Loaded workflow orchestration and UI interaction schemas');
  return schemas;
}

// Create combined schema with resolvers
function createWorkflowSchema() {
  const schemas = loadSchemas();

  // Combine type definitions
  const typeDefs = `
    ${schemas.orchestration}
    ${schemas.uiInteraction}

    # Add authorization directive
    directive @requiresAuth on FIELD_DEFINITION | OBJECT | INTERFACE | UNION | ARGUMENT_DEFINITION | SCALAR | ENUM | ENUM_VALUE | INPUT_OBJECT | INPUT_FIELD_DEFINITION
  `;

  // Create executable schema
  const schema = makeExecutableSchema({
    typeDefs,
    resolvers: workflowUIResolvers
  });

  return schema;
}

// Enhanced service builder with UI interaction support
function buildService({ name, url }) {
  return {
    async process({ request, context }) {
      logger.info(`Processing request to ${name} service at ${url}`);

      const headers = {
        'Content-Type': 'application/json',
        ...(context?.token && { 'Authorization': context.token }),
        ...(context?.userId && { 'X-User-ID': context.userId }),
        ...(context?.userRole && { 'X-User-Role': context.userRole }),
        ...(context?.clinicianId && { 'X-Clinician-ID': context.clinicianId }),
        ...(context?.workflowId && { 'X-Workflow-ID': context.workflowId })
      };

      try {
        const requestBody = {
          query: request.query,
          variables: request.variables || {},
          operationName: request.operationName || null
        };

        const response = await fetch(url, {
          method: 'POST',
          headers,
          body: JSON.stringify(requestBody)
        });

        if (!response.ok) {
          const errorBody = await response.text();
          logger.error(`Error from ${name} service (${response.status}):`, errorBody);
          throw new Error(`HTTP error! status: ${response.status}, body: ${errorBody}`);
        }

        const responseBody = await response.text();
        logger.debug(`Response from ${name} service:`, responseBody);
        return JSON.parse(responseBody);
      } catch (error) {
        logger.error(`Error calling ${name} service at ${url}:`, error);
        throw error;
      }
    }
  };
}

// Initialize gateway with workflow services
async function initializeGateway() {
  try {
    logger.info('Initializing Enhanced Apollo Federation Gateway with UI Interaction');

    // Federation services including workflow engine
    const federationServices = [
      { name: 'patients', url: 'http://localhost:8003/api/federation' },
      { name: 'medications', url: 'http://localhost:8004/api/federation' },
      { name: 'workflows', url: 'http://localhost:8015/api/federation' }, // Workflow Engine Go
      { name: 'context-gateway', url: 'http://localhost:8117/api/federation' },
      { name: 'clinical-data-hub', url: 'http://localhost:8118/api/federation' },
      // Knowledge base services
      { name: 'kb1-drug-rules', url: 'http://localhost:8081/api/federation' },
      { name: 'kb2-clinical-context', url: 'http://localhost:8082/api/federation' },
      { name: 'kb3-guidelines', url: 'http://localhost:8084/graphql' },
      { name: 'kb4-patient-safety', url: 'http://localhost:8085/api/federation' },
      { name: 'kb5-ddi', url: 'http://localhost:8086/api/federation' },
      { name: 'kb6-formulary', url: 'http://localhost:8087/api/federation' },
      { name: 'kb7-terminology', url: 'http://localhost:8088/api/federation' },
      { name: 'evidence-envelope', url: 'http://localhost:8089/api/federation' }
    ];

    // Create federated gateway
    const gateway = new ApolloGateway({
      supergraphSdl: new IntrospectAndCompose({
        subgraphs: federationServices,
      }),
      debug: true,
      buildService
    });

    return gateway;
  } catch (error) {
    logger.error('Failed to initialize Apollo Gateway:', error);
    throw error;
  }
}

// Authentication middleware
function createAuthContext(req) {
  return {
    token: req.headers.authorization || '',
    userId: req.headers['x-user-id'],
    userEmail: req.headers['x-user-email'],
    userName: req.headers['x-user-name'],
    userRole: req.headers['x-user-role'],
    clinicianId: req.headers['x-clinician-id'],
    department: req.headers['x-department'],
    workflowId: req.headers['x-workflow-id'],
    // User authorization data
    user: req.headers['x-user-id'] ? {
      id: req.headers['x-user-id'],
      email: req.headers['x-user-email'],
      name: req.headers['x-user-name'],
      role: req.headers['x-user-role'],
      department: req.headers['x-department'],
      authorityLevel: req.headers['x-authority-level'] || 'ATTENDING'
    } : null,
    _headers: req.headers
  };
}

// Main server startup
async function startServer() {
  try {
    // Create HTTP server
    const app = express();
    const httpServer = createServer(app);

    // Set up CORS
    app.use(cors({
      origin: process.env.CORS_ORIGIN || 'http://localhost:3000',
      credentials: true
    }));

    // Parse JSON bodies
    app.use(express.json({ limit: '50mb' }));

    // Health check endpoint
    app.get('/health', (req, res) => {
      res.status(200).json({
        status: 'ok',
        service: 'apollo-federation-ui',
        timestamp: new Date().toISOString(),
        features: ['workflow-orchestration', 'ui-interaction', 'real-time-subscriptions']
      });
    });

    // Initialize the gateway
    const gateway = await initializeGateway();

    // Create workflow UI schema for local resolvers
    const workflowUISchema = createWorkflowSchema();

    // Create Apollo Server with both gateway and local schema
    const server = new ApolloServer({
      // Use gateway for federation
      gateway,
      // Enable introspection and playground
      introspection: true,
      playground: true,
      // Enable subscriptions
      subscriptions: {
        path: '/subscriptions',
        keepAlive: 30000,
      },
      // Error handling
      formatError: (err) => {
        logger.error('GraphQL Error:', {
          message: err.message,
          locations: err.locations,
          path: err.path,
          extensions: err.extensions
        });
        return err;
      },
      // Context for federated queries
      context: ({ req, connection }) => {
        if (connection) {
          // WebSocket connection context
          return {
            ...connection.context,
            pubsub
          };
        }
        // HTTP request context
        return {
          ...createAuthContext(req),
          pubsub
        };
      }
    });

    await server.start();

    // GraphQL HTTP middleware
    app.use('/graphql', cors(), json(), expressMiddleware(server, {
      context: async ({ req }) => ({
        ...createAuthContext(req),
        pubsub
      })
    }));

    // API Gateway compatible endpoint
    app.use('/api/graphql', cors(), json(), expressMiddleware(server, {
      context: async ({ req }) => ({
        ...createAuthContext(req),
        pubsub
      })
    }));

    // WebSocket server for subscriptions
    const wsServer = new WebSocketServer({
      server: httpServer,
      path: '/subscriptions'
    });

    // Use the schema with subscriptions
    const serverCleanup = useServer({
      schema: workflowUISchema,
      context: (ctx, msg, args) => {
        return {
          ...createAuthContext(ctx.connectionParams || {}),
          pubsub
        };
      },
      onConnect: (ctx) => {
        logger.info('WebSocket client connected');
      },
      onDisconnect: (ctx) => {
        logger.info('WebSocket client disconnected');
      }
    }, wsServer);

    // Start the server
    const PORT = process.env.PORT || 4000;
    await new Promise((resolve) => httpServer.listen({ port: PORT }, resolve));

    logger.info(`🚀 Enhanced Federation Server ready at http://localhost:${PORT}/graphql`);
    logger.info(`🔌 WebSocket subscriptions ready at ws://localhost:${PORT}/subscriptions`);
    logger.info(`🏥 Clinical UI interactions enabled`);
    logger.info(`📊 Real-time override notifications active`);

    // Graceful shutdown
    const shutdown = async () => {
      logger.info('Shutting down enhanced server...');
      serverCleanup.dispose();
      await server.stop();
      httpServer.close(() => {
        logger.info('Server has been shut down');
        process.exit(0);
      });
    };

    process.on('SIGTERM', shutdown);
    process.on('SIGINT', shutdown);

  } catch (error) {
    logger.error('Failed to start enhanced server:', error);
    process.exit(1);
  }
}

// Export for testing
module.exports = {
  startServer,
  createWorkflowSchema,
  buildService,
  logger
};

// Start server if this is the main module
if (require.main === module) {
  startServer().catch((err) => {
    logger.error('Error starting server:', err);
    process.exit(1);
  });
}