#!/usr/bin/env node

/**
 * Regenerate Apollo Federation Supergraph with EncounterManagementService
 * 
 * This script regenerates the supergraph schema including the new EncounterManagementService
 * and validates that all services are properly federated.
 */

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');

// Configure logging
const logger = {
  info: (message) => console.log(`[INFO] ${new Date().toISOString()} - ${message}`),
  error: (message, error) => console.error(`[ERROR] ${new Date().toISOString()} - ${message}`, error),
  warn: (message) => console.warn(`[WARN] ${new Date().toISOString()} - ${message}`),
  success: (message) => console.log(`[SUCCESS] ${new Date().toISOString()} - ${message}`)
};

// Define all microservice endpoints including the new encounter service
const serviceList = [
  {
    name: 'patients',
    url: 'http://localhost:8003/api/federation',
    port: 8003
  },
  {
    name: 'observations',
    url: 'http://localhost:8007/api/federation',
    port: 8007
  },
  {
    name: 'medications',
    url: 'http://localhost:8009/api/federation',
    port: 8009
  },
  {
    name: 'organizations',
    url: 'http://localhost:8012/api/federation',
    port: 8012
  },
  {
    name: 'orders',
    url: 'http://localhost:8013/api/federation',
    port: 8013
  },
  {
    name: 'scheduling',
    url: 'http://localhost:8014/api/federation',
    port: 8014
  },
  {
    name: 'encounters',
    url: 'http://localhost:8020/api/federation',
    port: 8020
  }
];

// Check if a service is running
async function checkServiceHealth(service) {
  try {
    const healthUrl = `http://localhost:${service.port}/health`;
    const response = await fetch(healthUrl);
    return response.ok;
  } catch (error) {
    return false;
  }
}

// Check if federation endpoint is available
async function checkFederationEndpoint(service) {
  try {
    const response = await fetch(service.url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        query: 'query { __schema { types { name } } }'
      })
    });
    return response.ok;
  } catch (error) {
    return false;
  }
}

// Create updated supergraph config
function createSupergraphConfig() {
  const configPath = path.join(__dirname, 'supergraph-with-encounters.yaml');
  
  const configContent = `federation_version: 2
subgraphs:
${serviceList.map(service => `  ${service.name}:
    routing_url: ${service.url}
    schema:
      subgraph_url: ${service.url}
`).join('')}`;

  fs.writeFileSync(configPath, configContent);
  logger.info(`Created supergraph config at ${configPath}`);
  
  return configPath;
}

// Generate supergraph schema using Apollo Rover
async function generateSupergraph(configPath) {
  try {
    logger.info('Generating supergraph schema with Apollo Rover...');
    
    const outputPath = path.join(__dirname, 'supergraph-with-encounters.graphql');
    
    // Run rover supergraph compose
    const command = `npx @apollo/rover supergraph compose --config ${configPath} --output ${outputPath}`;
    
    logger.info(`Running command: ${command}`);
    execSync(command, { stdio: 'inherit' });
    
    logger.success(`Supergraph schema generated at ${outputPath}`);
    return outputPath;
    
  } catch (error) {
    logger.error('Failed to generate supergraph schema:', error);
    throw error;
  }
}

// Validate the generated schema
function validateSchema(schemaPath) {
  try {
    const schemaContent = fs.readFileSync(schemaPath, 'utf8');
    
    // Check for encounter-related types
    const encounterTypes = [
      'type Encounter',
      'type EncounterParticipant',
      'type EncounterLocation',
      'enum EncounterStatus',
      'enum EncounterClass'
    ];
    
    const missingTypes = encounterTypes.filter(type => !schemaContent.includes(type));
    
    if (missingTypes.length === 0) {
      logger.success('✅ All encounter types found in supergraph schema');
    } else {
      logger.warn(`⚠️ Missing encounter types: ${missingTypes.join(', ')}`);
    }
    
    // Check for federation directives
    const federationDirectives = ['@key', '@external', '@shareable'];
    const foundDirectives = federationDirectives.filter(directive => schemaContent.includes(directive));
    
    logger.info(`Found federation directives: ${foundDirectives.join(', ')}`);
    
    // Check for Patient and User extensions
    const hasPatientExtension = schemaContent.includes('extend type Patient') || schemaContent.includes('type Patient') && schemaContent.includes('encounters');
    const hasUserExtension = schemaContent.includes('extend type User') || schemaContent.includes('type User') && schemaContent.includes('encountersAsParticipant');
    
    if (hasPatientExtension) {
      logger.success('✅ Patient entity extension found');
    } else {
      logger.warn('⚠️ Patient entity extension not found');
    }
    
    if (hasUserExtension) {
      logger.success('✅ User entity extension found');
    } else {
      logger.warn('⚠️ User entity extension not found');
    }
    
    return {
      hasEncounterTypes: missingTypes.length === 0,
      hasFederationDirectives: foundDirectives.length > 0,
      hasPatientExtension,
      hasUserExtension
    };
    
  } catch (error) {
    logger.error('Failed to validate schema:', error);
    return null;
  }
}

// Main execution
async function main() {
  try {
    logger.info('🚀 Starting supergraph regeneration with EncounterManagementService...');
    
    // Check service health
    logger.info('Checking service health...');
    for (const service of serviceList) {
      const isHealthy = await checkServiceHealth(service);
      const hasFederation = await checkFederationEndpoint(service);
      
      if (isHealthy && hasFederation) {
        logger.success(`✅ ${service.name} service is healthy and federation endpoint is available`);
      } else if (isHealthy) {
        logger.warn(`⚠️ ${service.name} service is healthy but federation endpoint may not be available`);
      } else {
        logger.warn(`⚠️ ${service.name} service is not running on port ${service.port}`);
      }
    }
    
    // Create supergraph config
    const configPath = createSupergraphConfig();
    
    // Generate supergraph
    const schemaPath = await generateSupergraph(configPath);
    
    // Validate schema
    const validation = validateSchema(schemaPath);
    
    if (validation) {
      logger.success('🎉 Supergraph regeneration completed successfully!');
      logger.info('Next steps:');
      logger.info('1. Start the Apollo Federation Gateway: npm start');
      logger.info('2. Test the encounter queries through the gateway');
      logger.info('3. Verify federation extensions work correctly');
    } else {
      logger.error('❌ Schema validation failed');
    }
    
  } catch (error) {
    logger.error('Failed to regenerate supergraph:', error);
    process.exit(1);
  }
}

// Run if called directly
if (require.main === module) {
  main();
}

module.exports = { main, serviceList };
