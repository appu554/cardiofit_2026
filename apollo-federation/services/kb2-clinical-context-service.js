const { ApolloServer } = require('@apollo/server');
const { buildSubgraphSchema } = require('@apollo/subgraph');
const { expressMiddleware } = require('@apollo/server/express4');
const express = require('express');
const http = require('http');
const cors = require('cors');
const { json } = require('body-parser');
const axios = require('axios');

// Import schema and resolvers
const kb2ClinicalContextTypeDefs = require('../schemas/kb2-clinical-context-schema');
const kb2ClinicalContextResolvers = require('../resolvers/kb2-clinical-context-resolvers');

require('dotenv').config();

// Configuration
const PORT = process.env.KB2_SUBGRAPH_PORT || 8082;
const KB2_SERVICE_URL = process.env.KB2_CLINICAL_CONTEXT_URL || 'http://localhost:8082';

// Configure logging
const logger = {
  info: (message, data) => console.log(`[KB2-SUBGRAPH][INFO] ${new Date().toISOString()} - ${message}`, data || ''),
  error: (message, error) => console.error(`[KB2-SUBGRAPH][ERROR] ${new Date().toISOString()} - ${message}`, error?.message || error),
  warn: (message, data) => console.warn(`[KB2-SUBGRAPH][WARN] ${new Date().toISOString()} - ${message}`, data || ''),
  debug: (message, data) => console.debug(`[KB2-SUBGRAPH][DEBUG] ${new Date().toISOString()} - ${message}`, data || '')
};

// Health check function for KB-2 service
async function checkKB2ServiceHealth() {
  try {
    const response = await axios.get(`${KB2_SERVICE_URL}/health`, {
      timeout: 5000
    });
    return response.status === 200;
  } catch (error) {
    logger.warn('KB-2 service health check failed', { error: error.message });
    return false;
  }
}

// Performance monitoring middleware
function performanceMiddleware(req, res, next) {
  const startTime = Date.now();
  const originalSend = res.send;

  res.send = function(data) {
    const endTime = Date.now();
    const duration = endTime - startTime;
    
    // Log slow queries (>500ms)
    if (duration > 500) {
      logger.warn('Slow query detected', {
        duration: `${duration}ms`,
        operation: req.body?.operationName,
        query: req.body?.query?.substring(0, 200)
      });
    }

    // Add performance headers
    res.set('X-Response-Time', `${duration}ms`);
    res.set('X-Service', 'kb2-clinical-context');
    
    originalSend.call(this, data);
  };

  next();
}

// Error tracking middleware
function errorTrackingMiddleware(err, req, res, next) {
  logger.error('GraphQL subgraph error', {
    error: err.message,
    stack: err.stack,
    operation: req.body?.operationName,
    variables: req.body?.variables
  });

  // Don't expose internal errors in production
  if (process.env.NODE_ENV === 'production') {
    res.status(500).json({
      errors: [{
        message: 'Internal server error',
        extensions: {
          code: 'INTERNAL_ERROR'
        }
      }]
    });
  } else {
    next(err);
  }
}

// Create Apollo subgraph server
async function createSubgraphServer() {
  try {
    logger.info('Creating KB-2 Clinical Context subgraph server...');

    // Check if KB-2 service is available
    const isServiceHealthy = await checkKB2ServiceHealth();
    if (!isServiceHealthy) {
      logger.warn('KB-2 service is not available, but starting subgraph anyway');
    } else {
      logger.info('KB-2 service health check passed');
    }

    // Build the federated schema
    const schema = buildSubgraphSchema({
      typeDefs: kb2ClinicalContextTypeDefs,
      resolvers: kb2ClinicalContextResolvers
    });

    // Create Apollo Server
    const server = new ApolloServer({
      schema,
      introspection: true,
      debug: process.env.NODE_ENV !== 'production',
      plugins: [
        // Custom plugin for KB-2 specific metrics
        {
          requestDidStart() {
            return {
              willSendResponse(requestContext) {
                // Add KB-2 specific headers
                requestContext.response.http.headers.set('X-KB2-Service', KB2_SERVICE_URL);
                requestContext.response.http.headers.set('X-Subgraph', 'kb2-clinical-context');
                
                // Log GraphQL operations
                logger.debug('GraphQL operation completed', {
                  operation: requestContext.request.operationName,
                  variables: requestContext.request.variables
                });
              },
              didEncounterErrors(requestContext) {
                // Log GraphQL errors with context
                requestContext.errors?.forEach(error => {
                  logger.error('GraphQL execution error', {
                    error: error.message,
                    path: error.path,
                    operation: requestContext.request.operationName,
                    extensions: error.extensions
                  });
                });
              }
            };
          }
        }
      ],
      formatError: (err) => {
        // Enhanced error formatting for KB-2 subgraph
        logger.error('GraphQL formatted error', {
          message: err.message,
          path: err.path,
          extensions: err.extensions
        });

        return {
          message: err.message,
          path: err.path,
          extensions: {
            ...err.extensions,
            service: 'kb2-clinical-context',
            timestamp: new Date().toISOString()
          }
        };
      }
    });

    await server.start();
    logger.info('KB-2 Clinical Context subgraph server started successfully');
    
    return server;

  } catch (error) {
    logger.error('Failed to create KB-2 subgraph server', error);
    throw error;
  }
}

// Start the subgraph server
async function startServer() {
  try {
    const app = express();
    const httpServer = http.createServer(app);

    // Middleware
    app.use(cors({
      origin: process.env.CORS_ORIGIN || '*',
      credentials: true
    }));
    
    app.use(express.json({ limit: '10mb' }));
    app.use(performanceMiddleware);

    // Health check endpoint
    app.get('/health', async (req, res) => {
      try {
        const kb2ServiceHealthy = await checkKB2ServiceHealth();
        
        const healthStatus = {
          status: kb2ServiceHealthy ? 'healthy' : 'degraded',
          service: 'kb2-clinical-context-subgraph',
          version: '1.0.0',
          timestamp: new Date().toISOString(),
          dependencies: {
            'kb2-service': {
              status: kb2ServiceHealthy ? 'up' : 'down',
              url: KB2_SERVICE_URL
            }
          },
          environment: process.env.NODE_ENV || 'development'
        };

        res.status(kb2ServiceHealthy ? 200 : 503).json(healthStatus);
      } catch (error) {
        logger.error('Health check failed', error);
        res.status(503).json({
          status: 'unhealthy',
          error: error.message,
          timestamp: new Date().toISOString()
        });
      }
    });

    // Readiness probe
    app.get('/ready', async (req, res) => {
      try {
        const kb2ServiceHealthy = await checkKB2ServiceHealth();
        
        if (kb2ServiceHealthy) {
          res.status(200).json({
            status: 'ready',
            service: 'kb2-clinical-context-subgraph',
            timestamp: new Date().toISOString()
          });
        } else {
          res.status(503).json({
            status: 'not-ready',
            reason: 'KB-2 service unavailable',
            timestamp: new Date().toISOString()
          });
        }
      } catch (error) {
        res.status(503).json({
          status: 'not-ready',
          error: error.message,
          timestamp: new Date().toISOString()
        });
      }
    });

    // Metrics endpoint
    app.get('/metrics', (req, res) => {
      // Basic metrics - in production, use Prometheus
      res.set('Content-Type', 'text/plain');
      res.send(`
# HELP kb2_subgraph_requests_total Total number of GraphQL requests
# TYPE kb2_subgraph_requests_total counter
kb2_subgraph_requests_total{service="kb2-clinical-context"} 0

# HELP kb2_subgraph_response_duration_seconds Response duration in seconds
# TYPE kb2_subgraph_response_duration_seconds histogram
kb2_subgraph_response_duration_seconds_bucket{service="kb2-clinical-context",le="0.1"} 0
kb2_subgraph_response_duration_seconds_bucket{service="kb2-clinical-context",le="0.5"} 0
kb2_subgraph_response_duration_seconds_bucket{service="kb2-clinical-context",le="1.0"} 0
kb2_subgraph_response_duration_seconds_bucket{service="kb2-clinical-context",le="+Inf"} 0

# HELP kb2_service_dependency_up KB-2 service dependency status
# TYPE kb2_service_dependency_up gauge
kb2_service_dependency_up{service="kb2-clinical-context",dependency="kb2-service"} 1
      `.trim());
    });

    // Service info endpoint
    app.get('/info', (req, res) => {
      res.json({
        service: 'kb2-clinical-context-subgraph',
        version: '1.0.0',
        description: 'Apollo Federation subgraph for KB-2 Clinical Context Service',
        capabilities: [
          'phenotype-evaluation',
          'risk-assessment',
          'treatment-preferences',
          'clinical-context-assembly'
        ],
        endpoints: {
          graphql: '/api/federation',
          health: '/health',
          ready: '/ready',
          metrics: '/metrics'
        },
        kb2_service: KB2_SERVICE_URL,
        environment: process.env.NODE_ENV || 'development',
        timestamp: new Date().toISOString()
      });
    });

    // Create and configure the GraphQL server
    const server = await createSubgraphServer();
    
    // Setup GraphQL endpoint with context
    const graphqlMiddleware = expressMiddleware(server, {
      context: async ({ req }) => {
        const context = {
          // Authentication context
          token: req.headers.authorization,
          userId: req.headers['x-user-id'],
          userEmail: req.headers['x-user-email'],
          userRole: req.headers['x-user-role'],
          userRoles: req.headers['x-user-roles']?.split(',').map(r => r.trim()) || [],
          userPermissions: req.headers['x-user-permissions']?.split(',').map(p => p.trim()) || [],
          
          // Request context
          requestId: req.headers['x-request-id'] || `kb2-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
          
          // Service context
          service: 'kb2-clinical-context',
          timestamp: new Date().toISOString(),
          
          // Raw headers for debugging
          _headers: req.headers
        };

        logger.debug('GraphQL context created', {
          userId: context.userId,
          userRole: context.userRole,
          requestId: context.requestId,
          hasToken: !!context.token
        });

        return context;
      }
    });

    // Setup GraphQL endpoints
    app.use('/api/federation', cors(), json(), graphqlMiddleware);
    app.use('/graphql', cors(), json(), graphqlMiddleware); // Alternative endpoint

    // Error handling middleware
    app.use(errorTrackingMiddleware);

    // 404 handler
    app.use('*', (req, res) => {
      res.status(404).json({
        error: 'Not Found',
        message: 'The requested endpoint does not exist',
        availableEndpoints: ['/api/federation', '/health', '/ready', '/metrics', '/info'],
        timestamp: new Date().toISOString()
      });
    });

    // Start the HTTP server
    await new Promise((resolve) => httpServer.listen({ port: PORT }, resolve));
    
    logger.info(`🚀 KB-2 Clinical Context subgraph server ready at http://localhost:${PORT}/api/federation`);
    logger.info(`📊 Health check available at http://localhost:${PORT}/health`);
    logger.info(`📈 Metrics available at http://localhost:${PORT}/metrics`);
    logger.info(`ℹ️ Service info available at http://localhost:${PORT}/info`);

    // Graceful shutdown handling
    const shutdown = async () => {
      logger.info('Shutting down KB-2 subgraph server...');
      
      try {
        await server.stop();
        httpServer.close(() => {
          logger.info('KB-2 subgraph server has been shut down gracefully');
          process.exit(0);
        });
      } catch (error) {
        logger.error('Error during shutdown', error);
        process.exit(1);
      }
    };

    process.on('SIGTERM', shutdown);
    process.on('SIGINT', shutdown);

    // Handle uncaught exceptions
    process.on('uncaughtException', (error) => {
      logger.error('Uncaught exception', error);
      process.exit(1);
    });

    process.on('unhandledRejection', (reason, promise) => {
      logger.error('Unhandled promise rejection', { reason, promise });
      process.exit(1);
    });

  } catch (error) {
    logger.error('Failed to start KB-2 subgraph server', error);
    process.exit(1);
  }
}

// Start the server if this file is run directly
if (require.main === module) {
  startServer().catch((error) => {
    logger.error('Failed to start server', error);
    process.exit(1);
  });
}

module.exports = {
  createSubgraphServer,
  startServer
};