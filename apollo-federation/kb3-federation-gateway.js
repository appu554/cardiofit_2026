const { ApolloServer } = require('@apollo/server');
const { expressMiddleware } = require('@apollo/server/express4');
const { ApolloGateway, IntrospectAndCompose } = require('@apollo/gateway');
const express = require('express');
const http = require('http');
const cors = require('cors');
const { json } = require('body-parser');

// Configure logging
const logger = {
  info: (message) => console.log(`[INFO] ${new Date().toISOString()} - ${message}`),
  error: (message, error) => console.error(`[ERROR] ${new Date().toISOString()} - ${message}`, error?.message || error),
  warn: (message) => console.warn(`[WARN] ${new Date().toISOString()} - ${message}`),
  debug: (message) => console.debug(`[DEBUG] ${new Date().toISOString()} - ${message}`)
};

// Configure fetch with proper error handling
const fetch = (...args) => import('node-fetch').then(({default: fetch}) => fetch(...args));

// Function to verify service health
async function checkServiceHealth(service) {
  try {
    const healthUrl = service.url.replace('/api/federation', '/health');
    const response = await fetch(healthUrl, {
      method: 'GET',
      timeout: 5000
    });

    if (response.ok) {
      const health = await response.json();
      logger.info(`✅ Service ${service.name} is healthy: ${health.status}`);
      return true;
    } else {
      logger.warn(`⚠️  Service ${service.name} health check failed: ${response.status}`);
      return false;
    }
  } catch (error) {
    logger.warn(`❌ Service ${service.name} is not available: ${error.message}`);
    return false;
  }
}

// Custom build service function with enhanced logging
function buildService({ name, url }) {
  return {
    process: async (options) => {
      const { request, context } = options;

      logger.info(`Processing request to ${name} service at ${url}`);
      logger.debug('Request details:', {
        query: request.query?.substring(0, 200) + '...',
        variables: request.variables,
        operationName: request.operationName
      });

      const headers = {
        'Content-Type': 'application/json',
        ...(context?.token && { 'Authorization': context.token }),
        ...(context?.userId && { 'X-User-ID': context.userId }),
        ...(context?.userRole && { 'X-User-Role': context.userRole }),
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
          body: JSON.stringify(requestBody),
          timeout: 30000
        });

        if (!response.ok) {
          logger.error(`HTTP ${response.status} from ${name} service: ${response.statusText}`);
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }

        const result = await response.json();
        logger.debug(`✅ Response from ${name} service:`, {
          data: !!result.data,
          errors: result.errors?.length || 0
        });

        return result;

      } catch (error) {
        logger.error(`Error calling ${name} service at ${url}:`, error.message);
        throw error;
      }
    }
  };
}

// Initialize Apollo Gateway
async function initializeGateway() {
  try {
    logger.info('🚀 Initializing Apollo Federation Gateway with KB-3 Guidelines');

    // Define available services
    const availableServices = [
      { name: 'kb3-guidelines', url: 'http://localhost:8085/api/federation' },
      { name: 'patients', url: 'http://localhost:8003/api/federation' },
      { name: 'medications', url: 'http://localhost:8004/api/federation' }, // Updated port
      { name: 'kb2-clinical-context', url: 'http://localhost:8082/api/federation' }
    ];

    // Check which services are actually running
    logger.info('🔍 Checking service availability...');
    const healthChecks = await Promise.all(
      availableServices.map(async (service) => ({
        ...service,
        healthy: await checkServiceHealth(service)
      }))
    );

    // Only include healthy services
    const healthyServices = healthChecks.filter(service => service.healthy);

    if (healthyServices.length === 0) {
      throw new Error('No healthy services found');
    }

    logger.info(`📊 Found ${healthyServices.length} healthy services:`);
    healthyServices.forEach(service => {
      logger.info(`   • ${service.name} - ${service.url}`);
    });

    return new ApolloGateway({
      supergraphSdl: new IntrospectAndCompose({
        subgraphs: healthyServices,
        introspectionHeaders: {
          'Apollo-Require-Preflight': 'true'
        }
      }),
      debug: true,
      buildService,
    });

  } catch (error) {
    logger.error('Failed to initialize Apollo Gateway:', error);
    throw error;
  }
}

// Start server function
async function startServer() {
  const app = express();
  const httpServer = http.createServer(app);

  try {
    logger.info('🏗️  Starting Apollo Federation Gateway...');

    // Enable CORS
    app.use(cors({
      origin: ['http://localhost:3000', 'http://localhost:4000'],
      credentials: true
    }));

    // Parse JSON bodies
    app.use(express.json());

    // Health check endpoint
    app.get('/health', (req, res) => {
      res.status(200).json({
        status: 'healthy',
        service: 'Apollo Federation Gateway',
        timestamp: new Date().toISOString()
      });
    });

    // Service discovery endpoint
    app.get('/services', async (req, res) => {
      const services = [
        { name: 'kb3-guidelines', url: 'http://localhost:8085/api/federation', port: 8085 },
        { name: 'patients', url: 'http://localhost:8003/api/federation', port: 8003 },
        { name: 'medications', url: 'http://localhost:8004/api/federation', port: 8004 },
        { name: 'kb2-clinical-context', url: 'http://localhost:8082/api/federation', port: 8082 }
      ];

      const healthChecks = await Promise.all(
        services.map(async (service) => ({
          ...service,
          healthy: await checkServiceHealth(service)
        }))
      );

      res.json({
        services: healthChecks,
        healthy_count: healthChecks.filter(s => s.healthy).length,
        total_count: healthChecks.length
      });
    });

    // Initialize the Apollo Gateway
    logger.info('🔗 Initializing Federation Gateway...');
    const gateway = await initializeGateway();

    // Create Apollo Server with the gateway
    const server = new ApolloServer({
      gateway,
      introspection: true,
      debug: true,
      plugins: [
        {
          requestDidStart() {
            return {
              didResolveOperation(requestContext) {
                logger.info(`GraphQL Operation: ${requestContext.request.operationName || 'Anonymous'}`);
              },
              didEncounterErrors(requestContext) {
                logger.error('GraphQL Errors:', requestContext.errors?.map(e => e.message));
              }
            };
          }
        }
      ],
      formatError: (error) => {
        logger.error('GraphQL Error:', error.message);
        return {
          message: error.message,
          locations: error.locations,
          path: error.path,
          extensions: {
            code: error.extensions?.code,
            timestamp: new Date().toISOString()
          }
        };
      }
    });

    await server.start();

    app.use(
      '/graphql',
      expressMiddleware(server, {
        context: async ({ req }) => ({
          token: req.headers.authorization,
          userId: req.headers['x-user-id'],
          userRole: req.headers['x-user-role']
        })
      })
    );

    const PORT = process.env.PORT || 4000;
    httpServer.listen(PORT, () => {
      logger.info(`🚀 Apollo Federation Gateway is running!`);
      logger.info(`📊 GraphQL Playground: http://localhost:${PORT}/graphql`);
      logger.info(`🏥 Health Check: http://localhost:${PORT}/health`);
      logger.info(`🔍 Services Status: http://localhost:${PORT}/services`);
      logger.info('');
      logger.info('🎯 Available GraphQL Operations:');
      logger.info('   • Query guidelines from KB-3');
      logger.info('   • Search by condition, organization, evidence grade');
      logger.info('   • Get top quality guidelines');
      logger.info('');
    });

  } catch (error) {
    logger.error('❌ Failed to start server:', error);
    process.exit(1);
  }
}

// Handle graceful shutdown
process.on('SIGINT', () => {
  logger.info('🛑 Shutting down Apollo Federation Gateway...');
  process.exit(0);
});

process.on('SIGTERM', () => {
  logger.info('🛑 Shutting down Apollo Federation Gateway...');
  process.exit(0);
});

// Start the server
startServer();