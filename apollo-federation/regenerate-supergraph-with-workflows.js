#!/usr/bin/env node

/**
 * Regenerate Apollo Federation supergraph schema with WorkflowEngineService integration
 */

const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

// Logger utility
const logger = {
  info: (msg) => console.log(`ℹ️  ${msg}`),
  success: (msg) => console.log(`✅ ${msg}`),
  error: (msg) => console.error(`❌ ${msg}`),
  warn: (msg) => console.warn(`⚠️  ${msg}`)
};

// Define all microservices including workflow engine
const services = [
  { name: 'patients', url: 'http://localhost:8003/api/federation', healthUrl: 'http://localhost:8003/health' },
  { name: 'observations', url: 'http://localhost:8007/api/federation', healthUrl: 'http://localhost:8007/health' },
  { name: 'medications', url: 'http://localhost:8009/api/federation', healthUrl: 'http://localhost:8009/health' },
  { name: 'organizations', url: 'http://localhost:8012/api/federation', healthUrl: 'http://localhost:8012/health' },
  { name: 'orders', url: 'http://localhost:8013/api/federation', healthUrl: 'http://localhost:8013/health' },
  { name: 'scheduling', url: 'http://localhost:8014/api/federation', healthUrl: 'http://localhost:8014/health' },
  { name: 'encounters', url: 'http://localhost:8020/api/federation', healthUrl: 'http://localhost:8020/health' },
  { name: 'workflows', url: 'http://localhost:8015/api/federation', healthUrl: 'http://localhost:8015/health' }
];

// Check service health
async function checkServiceHealth(service) {
  try {
    const fetch = (await import('node-fetch')).default;
    logger.info(`Checking health of ${service.name} service...`);
    const response = await fetch(service.healthUrl, { timeout: 5000 });
    
    if (response.ok) {
      logger.success(`${service.name} service is healthy`);
      return true;
    } else {
      logger.error(`${service.name} service health check failed: ${response.status}`);
      return false;
    }
  } catch (error) {
    logger.error(`${service.name} service is not accessible: ${error.message}`);
    return false;
  }
}

// Check federation endpoint
async function checkFederationEndpoint(service) {
  try {
    const fetch = (await import('node-fetch')).default;
    logger.info(`Checking federation endpoint for ${service.name}...`);
    const response = await fetch(service.url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        query: `
          query IntrospectionQuery {
            __schema {
              queryType { name }
              mutationType { name }
              types {
                name
                kind
              }
            }
          }
        `
      }),
      timeout: 10000
    });

    if (response.ok) {
      const data = await response.json();
      if (data.data && data.data.__schema) {
        logger.success(`${service.name} federation endpoint is working`);
        return true;
      } else {
        logger.error(`${service.name} federation endpoint returned invalid schema`);
        return false;
      }
    } else {
      logger.error(`${service.name} federation endpoint failed: ${response.status}`);
      return false;
    }
  } catch (error) {
    logger.error(`${service.name} federation endpoint error: ${error.message}`);
    return false;
  }
}

// Create supergraph configuration
function createSupergraphConfig() {
  const configPath = path.join(__dirname, 'supergraph-with-workflows.yaml');
  
  const configContent = `federation_version: 2
subgraphs:
${services.map(service => `  ${service.name}:
    routing_url: ${service.url}
    schema:
      subgraph_url: ${service.url}`).join('\n')}
`;

  fs.writeFileSync(configPath, configContent);
  logger.success(`Created supergraph config at ${configPath}`);
  return configPath;
}

// Generate supergraph schema using Apollo Rover
async function generateSupergraph(configPath) {
  try {
    logger.info('Generating supergraph schema with Apollo Rover...');
    
    const outputPath = path.join(__dirname, 'supergraph-with-workflows.graphql');
    
    // Run rover supergraph compose with proper quoting for paths with spaces
    const command = `npx @apollo/rover supergraph compose --config "${configPath}" --output "${outputPath}"`;

    logger.info(`Running command: ${command}`);
    execSync(command, { stdio: 'inherit' });
    
    logger.success(`Supergraph schema generated at ${outputPath}`);
    return outputPath;
    
  } catch (error) {
    logger.error('Failed to generate supergraph schema:', error);
    throw error;
  }
}

// Validate generated schema
function validateSchema(schemaPath) {
  try {
    logger.info('Validating generated supergraph schema...');
    
    const schemaContent = fs.readFileSync(schemaPath, 'utf8');
    
    // Check for workflow-specific types (using GraphQL naming conventions)
    const workflowTypes = [
      'WorkflowDefinition',
      'WorkflowInstanceSummary',  // Strawberry converts WorkflowInstance_Summary to WorkflowInstanceSummary
      'Task',
      'startWorkflow',
      'completeTask',
      'claimTask'
    ];
    
    const missingTypes = workflowTypes.filter(type => !schemaContent.includes(type));
    
    if (missingTypes.length > 0) {
      logger.error(`Missing workflow types in schema: ${missingTypes.join(', ')}`);
      return false;
    }
    
    // Check for federation extensions
    const federationExtensions = [
      'extend type Patient',
      'extend type User'
    ];
    
    const missingExtensions = federationExtensions.filter(ext => !schemaContent.includes(ext));
    
    if (missingExtensions.length > 0) {
      logger.warn(`Missing federation extensions: ${missingExtensions.join(', ')}`);
    }
    
    logger.success('Schema validation completed successfully');
    return true;
    
  } catch (error) {
    logger.error('Schema validation failed:', error);
    return false;
  }
}

// Main execution function
async function main() {
  try {
    logger.info('🚀 Starting WorkflowEngine Federation Integration...');
    
    // Check all service health
    logger.info('Checking service health...');
    const healthChecks = await Promise.all(services.map(checkServiceHealth));
    const healthyServices = services.filter((_, index) => healthChecks[index]);
    
    if (healthyServices.length < services.length) {
      const unhealthyServices = services.filter((_, index) => !healthChecks[index]);
      logger.warn(`Some services are not healthy: ${unhealthyServices.map(s => s.name).join(', ')}`);
      logger.warn('Proceeding with available services...');
    }
    
    // Check federation endpoints for healthy services
    logger.info('Checking federation endpoints...');
    const federationChecks = await Promise.all(healthyServices.map(checkFederationEndpoint));
    const workingServices = healthyServices.filter((_, index) => federationChecks[index]);
    
    if (workingServices.length === 0) {
      logger.error('No services have working federation endpoints');
      process.exit(1);
    }
    
    logger.success(`${workingServices.length} services ready for federation`);
    
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
      logger.info('2. Test the workflow queries through the gateway');
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

module.exports = { main };
