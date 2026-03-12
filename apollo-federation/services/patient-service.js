const { ApolloServer } = require('@apollo/server');
const { expressMiddleware } = require('@apollo/server/express4');
const { buildSubgraphSchema } = require('@apollo/subgraph');
const express = require('express');
const http = require('http');
const cors = require('cors');
const bodyParser = require('body-parser');
require('dotenv').config();

// Import schema and resolvers
const typeDefs = require('../schemas/patient-schema');
const resolvers = require('../resolvers/patient-resolvers');

// Initialize Express
const app = express();
const httpServer = http.createServer(app);

// Start the server
async function startServer() {
  // Create Apollo Server
  const server = new ApolloServer({
    schema: buildSubgraphSchema({ typeDefs, resolvers }),
  });

  // Start Apollo Server
  await server.start();

  // Apply middleware
  app.use(
    '/api/graphql',
    cors(),
    bodyParser.json(),
    expressMiddleware(server, {
      context: async ({ req }) => {
        // Get the authorization header
        const token = req.headers.authorization || '';
        
        // Get user info from headers
        const userId = req.headers['x-user-id'];
        const userRole = req.headers['x-user-role'];
        
        // Return context with auth info
        return {
          token,
          userId,
          userRole
        };
      },
    }),
  );

  // Health check endpoint
  app.get('/health', (req, res) => {
    res.json({ status: 'ok' });
  });

  // Start the HTTP server
  const PORT = process.env.PATIENT_SERVICE_PORT || 8003;
  httpServer.listen(PORT, () => {
    console.log(`🚀 Patient Service ready at http://localhost:${PORT}/api/graphql`);
  });
}

startServer().catch((err) => {
  console.error('Error starting server:', err);
});
