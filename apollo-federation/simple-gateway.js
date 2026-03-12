const { ApolloServer } = require('@apollo/server');
const { expressMiddleware } = require('@apollo/server/express4');
const { ApolloGateway } = require('@apollo/gateway');
const express = require('express');
const http = require('http');
const cors = require('cors');
const { json } = require('body-parser');
const fetch = (...args) => import('node-fetch').then(({default: fetch}) => fetch(...args));

// Configure logging
const logger = {
  info: (message) => console.log(`[INFO] ${new Date().toISOString()} - ${message}`),
  error: (message, error) => console.error(`[ERROR] ${new Date().toISOString()} - ${message}`, error),
  warn: (message) => console.warn(`[WARN] ${new Date().toISOString()} - ${message}`),
  debug: (message) => console.debug(`[DEBUG] ${new Date().toISOString()} - ${message}`)
};

// Define the patient service URL
const PATIENT_SERVICE_URL = process.env.PATIENT_SERVICE_URL || 'http://localhost:8003/api';

// Create a simple proxy server that forwards requests to the patient service
async function startServer() {
  // Initialize Express
  const app = express();
  const httpServer = http.createServer(app);

  // Enable CORS and JSON parsing
  app.use(cors());
  app.use(json({ limit: '2mb' }));

  // Health check endpoint
  app.get('/health', async (req, res) => {
    try {
      // Check if patient service is reachable
      const response = await fetch(`${PATIENT_SERVICE_URL}/health`, {
        method: 'GET',
        timeout: 5000
      }).catch(error => {
        logger.warn(`Patient service health check failed: ${error.message}`);
        return { ok: false };
      });

      if (response.ok) {
        res.json({
          status: 'ok',
          timestamp: new Date().toISOString(),
          message: 'Patient service is healthy',
          services: [
            { service: 'patients', url: PATIENT_SERVICE_URL, isHealthy: true }
          ]
        });
      } else {
        res.json({
          status: 'degraded',
          timestamp: new Date().toISOString(),
          message: 'Patient service is not healthy',
          services: [
            { service: 'patients', url: PATIENT_SERVICE_URL, isHealthy: false }
          ]
        });
      }
    } catch (error) {
      logger.error('Health check error:', error);
      res.status(500).json({
        status: 'error',
        message: error.message
      });
    }
  });

  // GraphQL endpoint - proxy to patient service
  app.post('/graphql', async (req, res) => {
    try {
      // Extract the GraphQL query and variables
      const { query, variables, operationName } = req.body;
      
      // Get the authorization header
      const authHeader = req.headers.authorization;
      
      // Forward the request to the patient service
      const headers = {
        'Content-Type': 'application/json'
      };
      
      // Add authorization header if available
      if (authHeader) {
        headers['Authorization'] = authHeader;
        
        // Extract user ID from token (simplified)
        const userId = '123456'; // This would normally be extracted from the token
        headers['X-User-ID'] = userId;
        headers['X-User-Role'] = 'admin';
      }
      
      // Make the request to the patient service
      const response = await fetch(`${PATIENT_SERVICE_URL}/graphql`, {
        method: 'POST',
        headers,
        body: JSON.stringify({
          query,
          variables,
          operationName
        })
      });
      
      // Get the response
      const data = await response.json();
      
      // Return the response
      res.json(data);
    } catch (error) {
      logger.error('GraphQL proxy error:', error);
      res.status(500).json({
        errors: [
          {
            message: error.message
          }
        ]
      });
    }
  });

  // Add a dedicated route for Apollo Sandbox
  app.get('/sandbox', (req, res) => {
    res.send(`
      <!DOCTYPE html>
      <html>
      <head>
        <title>Simple Federation Gateway - GraphQL Sandbox</title>
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
          <h1>Simple Federation Gateway - GraphQL Sandbox</h1>
          <p>Use the button below to open the Apollo Sandbox to build and test GraphQL queries:</p>
          <a class="button" href="/graphql" target="_blank">Open Apollo Sandbox</a>
        </div>
      </body>
      </html>
    `);
  });

  // Start the HTTP server
  const PORT = process.env.PORT || 4000;
  httpServer.listen(PORT, () => {
    logger.info(`🚀 Simple Federation Gateway ready at http://localhost:${PORT}/graphql`);
    logger.info(`GraphQL Sandbox available at http://localhost:${PORT}/sandbox`);
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

// Start the server
startServer().catch((err) => {
  logger.error('Error starting server:', err);
  process.exit(1);
});
