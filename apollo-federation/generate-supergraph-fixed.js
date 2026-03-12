/**
 * Generate Supergraph Schema for Apollo Federation - Fixed Version
 *
 * This script generates a supergraph schema from the subgraph schemas of all microservices.
 * It uses the exact same configuration that worked in the test.
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
  success: (message) => console.log(`[SUCCESS] ${new Date().toISOString()} - ${message}`)
};

// Use the exact same service configuration that worked in the test
const serviceList = [
  {
    name: 'patients',
    url: 'http://localhost:8003/api/federation'
  },
  {
    name: 'observations',
    url: 'http://localhost:8007/api/federation'
  },
  {
    name: 'medications',
    url: 'http://localhost:8009/api/federation'
  },
  {
    name: 'organizations',
    url: 'http://localhost:8012/api/federation'
  },
  {
    name: 'orders',
    url: 'http://localhost:8013/api/federation'
  },
  {
    name: 'scheduling',
    url: 'http://localhost:8014/api/federation'
  },
  {
    name: 'encounters',
    url: 'http://localhost:8020/api/federation'
  }
];

// Test each service before generating supergraph
async function testAllServices() {
  logger.info('Testing all services before supergraph generation...');
  
  for (const service of serviceList) {
    try {
      logger.info(`Testing ${service.name} at ${service.url}...`);
      
      const command = `npx @apollo/rover subgraph introspect ${service.url}`;
      execSync(command, { 
        stdio: 'pipe',
        timeout: 10000
      });
      
      logger.success(`✅ ${service.name}: OK`);
    } catch (error) {
      logger.error(`❌ ${service.name}: FAILED`);
      logger.error(`Error: ${error.message}`);
      return false;
    }
  }
  
  logger.success('All services are responding correctly!');
  return true;
}

// Create a supergraph config file
function createSupergraphConfig() {
  const configPath = path.join(__dirname, 'supergraph-working.yaml');

  // Create the config content with exact formatting
  const configContent = `federation_version: 2
subgraphs:
${serviceList.map(service => `  ${service.name}:
    routing_url: ${service.url}
    schema:
      subgraph_url: ${service.url}`).join('\n')}
`;

  // Write the config file
  fs.writeFileSync(configPath, configContent);
  logger.info(`Created supergraph config at ${configPath}`);

  return configPath;
}

// Generate the supergraph schema
async function generateSupergraph() {
  try {
    // Test all services first
    const allServicesOk = await testAllServices();
    if (!allServicesOk) {
      logger.error('Some services are not responding. Cannot generate supergraph.');
      return false;
    }

    // Create the supergraph config
    const configPath = createSupergraphConfig();

    // Generate the supergraph schema
    const outputPath = path.join(__dirname, 'supergraph-working.graphql');

    // Check if rover is installed
    try {
      execSync('rover --version', { stdio: 'ignore' });
      logger.info('Apollo Rover is installed');
    } catch (error) {
      logger.error('Apollo Rover is not installed. Please install it with: npm install -g @apollo/rover');
      process.exit(1);
    }

    // Run rover supergraph compose with more verbose output
    const command = `rover supergraph compose --config "${configPath}" --output "${outputPath}"`;
    logger.info(`Running command: ${command}`);

    execSync(command, { stdio: 'inherit' });

    logger.success(`Generated supergraph schema at ${outputPath}`);
    
    // Copy to the main supergraph file
    const mainSupergraphPath = path.join(__dirname, 'supergraph.graphql');
    fs.copyFileSync(outputPath, mainSupergraphPath);
    logger.success(`Copied to main supergraph file: ${mainSupergraphPath}`);
    
    return true;
  } catch (error) {
    logger.error('Error generating supergraph schema:', error);
    
    // Try to get more detailed error information
    if (error.stdout) {
      logger.error('Stdout:', error.stdout.toString());
    }
    if (error.stderr) {
      logger.error('Stderr:', error.stderr.toString());
    }
    
    return false;
  }
}

// Main execution
async function main() {
  logger.info('🚀 Starting supergraph generation with working configuration...');
  
  const success = await generateSupergraph();
  
  if (success) {
    logger.success('🎉 Supergraph generation completed successfully!');
    logger.info('You can now start the Apollo Federation Gateway:');
    logger.info('npm start');
  } else {
    logger.error('❌ Supergraph generation failed');
    process.exit(1);
  }
}

// Run if called directly
if (require.main === module) {
  main();
}

module.exports = { generateSupergraph };
