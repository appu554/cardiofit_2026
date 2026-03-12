/**
 * Generate Supergraph Schema for Apollo Federation
 *
 * This script generates a supergraph schema from the subgraph schemas of all microservices.
 * It uses Apollo Rover to compose the schemas.
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
require('dotenv').config();

// Configure logging
const logger = {
  info: (message) => console.log(`[INFO] ${new Date().toISOString()} - ${message}`),
  error: (message, error) => console.error(`[ERROR] ${new Date().toISOString()} - ${message}`, error),
  warn: (message) => console.warn(`[WARN] ${new Date().toISOString()} - ${message}`),
  debug: (message) => console.debug(`[DEBUG] ${new Date().toISOString()} - ${message}`)
};

// Define the microservice endpoints for federation
// Only include services that are working and have proper federation endpoints
const serviceList = [
  // Use the dedicated federation endpoint for the patient service
  {
    name: 'patients',
    url: 'http://localhost:8003/api/federation'
  },
  // Use the dedicated federation endpoint for the medication service
  {
    name: 'medications',
    url: 'http://localhost:8009/api/federation'
  }
  // Removed other services that are not working or don't have proper federation endpoints
];

// Create a supergraph config file
function createSupergraphConfig() {
  const configPath = path.join(__dirname, 'supergraph.yaml');

  // Create the config content
  const configContent = `
federation_version: 2
subgraphs:
${serviceList.map(service => `  ${service.name}:
    routing_url: ${service.url}
    schema:
      subgraph_url: ${service.url}
`).join('')}
  `;

  // Write the config file
  fs.writeFileSync(configPath, configContent);
  logger.info(`Created supergraph config at ${configPath}`);

  return configPath;
}

// Generate the supergraph schema
async function generateSupergraph() {
  try {
    // Create the supergraph config
    const configPath = createSupergraphConfig();

    // Generate the supergraph schema
    const outputPath = path.join(__dirname, 'supergraph.graphql');

    // Check if rover is installed
    try {
      execSync('rover --version', { stdio: 'ignore' });
      logger.info('Apollo Rover is installed');
    } catch (error) {
      logger.error('Apollo Rover is not installed. Please install it with: npm install -g @apollo/rover');
      process.exit(1);
    }

    // Run rover supergraph compose
    const command = `rover supergraph compose --config "${configPath}" --output "${outputPath}"`;
    logger.info(`Running command: ${command}`);

    execSync(command, { stdio: 'inherit' });

    logger.info(`Generated supergraph schema at ${outputPath}`);
    return true;
  } catch (error) {
    logger.error('Error generating supergraph schema:', error);
    return false;
  }
}

generateSupergraph();
