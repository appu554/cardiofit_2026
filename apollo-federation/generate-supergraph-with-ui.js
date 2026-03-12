/**
 * Enhanced Supergraph Generation Script with UI Interaction Support
 * Generates Apollo Federation supergraph including workflow UI interaction schema
 */

const { execSync, spawn } = require('child_process');
const fs = require('fs');
const path = require('path');
const fetch = (...args) => import('node-fetch').then(({default: fetch}) => fetch(...args));

// Configuration
const TIMEOUT = 10000; // 10 seconds timeout for service checks
const OUTPUT_FILE = 'supergraph-with-ui.graphql';

const logger = {
  info: (message) => console.log(`[INFO] ${new Date().toISOString()} - ${message}`),
  error: (message, error) => console.error(`[ERROR] ${new Date().toISOString()} - ${message}`, error?.message || error),
  warn: (message) => console.warn(`[WARN] ${new Date().toISOString()} - ${message}`),
  success: (message) => console.log(`[SUCCESS] ${new Date().toISOString()} - ${message}`)
};

// Enhanced service configuration with UI interaction endpoints
const services = [
  {
    name: 'patients',
    url: 'http://localhost:8003/api/federation',
    healthCheck: 'http://localhost:8003/health',
    description: 'Patient management service with FHIR compliance'
  },
  {
    name: 'medications',
    url: 'http://localhost:8004/api/federation',
    healthCheck: 'http://localhost:8004/health',
    description: 'Medication service with dosing and interaction support'
  },
  {
    name: 'workflows',
    url: 'http://localhost:8015/api/federation',
    healthCheck: 'http://localhost:8015/health',
    description: 'Workflow Engine Go - Strategic Orchestrator with UI interaction'
  },
  {
    name: 'context-gateway',
    url: 'http://localhost:8117/api/federation',
    healthCheck: 'http://localhost:8117/health',
    description: 'Go Context Gateway for clinical data aggregation'
  },
  {
    name: 'clinical-data-hub',
    url: 'http://localhost:8118/api/federation',
    healthCheck: 'http://localhost:8118/health',
    description: 'Rust Clinical Data Hub for high-performance data processing'
  },
  // Knowledge Base Services
  {
    name: 'kb1-drug-rules',
    url: 'http://localhost:8081/api/federation',
    healthCheck: 'http://localhost:8081/health',
    description: 'Drug rules and dosing calculations'
  },
  {
    name: 'kb2-clinical-context',
    url: 'http://localhost:8082/api/federation',
    healthCheck: 'http://localhost:8082/health',
    description: 'Clinical context and patient assessment'
  },
  {
    name: 'kb3-guidelines',
    url: 'http://localhost:8084/graphql',
    healthCheck: 'http://localhost:8084/health',
    description: 'Clinical guidelines and evidence-based protocols'
  },
  {
    name: 'kb4-patient-safety',
    url: 'http://localhost:8085/api/federation',
    healthCheck: 'http://localhost:8085/health',
    description: 'Patient safety rules and contraindication checking'
  },
  {
    name: 'kb5-ddi',
    url: 'http://localhost:8086/api/federation',
    healthCheck: 'http://localhost:8086/health',
    description: 'Drug-drug interaction detection and analysis'
  },
  {
    name: 'kb6-formulary',
    url: 'http://localhost:8087/api/federation',
    healthCheck: 'http://localhost:8087/health',
    description: 'Hospital formulary and medication availability'
  },
  {
    name: 'kb7-terminology',
    url: 'http://localhost:8088/api/federation',
    healthCheck: 'http://localhost:8088/health',
    description: 'Medical terminology and coding standards'
  },
  {
    name: 'evidence-envelope',
    url: 'http://localhost:8089/api/federation',
    healthCheck: 'http://localhost:8089/health',
    description: 'Evidence envelope for audit trails and compliance'
  }
];

// Check if Rover CLI is available
function checkRoverCLI() {
  try {
    execSync('rover --version', { stdio: 'pipe' });
    logger.success('Rover CLI is available');
    return true;
  } catch (error) {
    logger.error('Rover CLI not found. Please install it first:',
      'curl -sSL https://rover.apollo.dev/nix/latest | sh');
    return false;
  }
}

// Check service health with enhanced reporting
async function checkServiceHealth(service) {
  try {
    logger.info(`Checking health of ${service.name} (${service.description})`);

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), TIMEOUT);

    const response = await fetch(service.healthCheck, {
      method: 'GET',
      signal: controller.signal,
      headers: { 'User-Agent': 'Apollo-Federation-Health-Check' }
    });

    clearTimeout(timeoutId);

    if (response.ok) {
      const healthData = await response.json();
      logger.success(`✅ ${service.name} is healthy - ${service.description}`);
      return {
        healthy: true,
        service: service.name,
        status: healthData.status || 'ok',
        features: healthData.features || []
      };
    } else {
      logger.warn(`⚠️ ${service.name} returned ${response.status}`);
      return { healthy: false, service: service.name, error: `HTTP ${response.status}` };
    }
  } catch (error) {
    if (error.name === 'AbortError') {
      logger.warn(`⏰ ${service.name} health check timed out`);
      return { healthy: false, service: service.name, error: 'timeout' };
    }
    logger.warn(`❌ ${service.name} health check failed: ${error.message}`);
    return { healthy: false, service: service.name, error: error.message };
  }
}

// Validate federation schema for service
async function validateFederationSchema(service) {
  try {
    logger.info(`Validating federation schema for ${service.name}`);

    const response = await fetch(service.url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        query: `
          query GetServiceSDL {
            _service {
              sdl
            }
          }
        `
      })
    });

    if (!response.ok) {
      throw new Error(`HTTP ${response.status}`);
    }

    const result = await response.json();
    if (result.errors) {
      throw new Error(`GraphQL errors: ${JSON.stringify(result.errors)}`);
    }

    if (!result.data?._service?.sdl) {
      throw new Error('No SDL returned from service');
    }

    // Check for UI interaction schema elements
    const sdl = result.data._service.sdl;
    const hasUIFeatures = {
      overrideMutations: sdl.includes('resolveClinicalOverride'),
      uiSubscriptions: sdl.includes('workflowUIUpdates'),
      realTimeFeatures: sdl.includes('overrideRequired'),
      clinicalTypes: sdl.includes('OverrideSession')
    };

    logger.success(`✅ ${service.name} federation schema is valid`);

    if (service.name === 'workflows') {
      const uiFeatureCount = Object.values(hasUIFeatures).filter(Boolean).length;
      if (uiFeatureCount > 0) {
        logger.success(`🎨 ${service.name} has ${uiFeatureCount}/4 UI interaction features`);
      }
    }

    return { valid: true, service: service.name, sdl, uiFeatures: hasUIFeatures };
  } catch (error) {
    logger.error(`❌ ${service.name} federation schema validation failed:`, error.message);
    return { valid: false, service: service.name, error: error.message };
  }
}

// Create supergraph configuration
function createSupergraphConfig(healthyServices) {
  const config = {
    federation_version: "=2.5.1",
    subgraphs: {}
  };

  healthyServices.forEach(service => {
    config.subgraphs[service.name] = {
      routing_url: service.url,
      schema: {
        subgraph_url: service.url
      }
    };
  });

  const configPath = path.join(__dirname, 'supergraph-ui-config.yaml');
  const yamlContent = `federation_version: ${config.federation_version}\nsubgraphs:\n` +
    Object.entries(config.subgraphs)
      .map(([name, config]) =>
        `  ${name}:\n    routing_url: ${config.routing_url}\n    schema:\n      subgraph_url: ${config.schema.subgraph_url}\n`
      ).join('');

  fs.writeFileSync(configPath, yamlContent);
  logger.success(`📝 Supergraph configuration written to ${configPath}`);

  return configPath;
}

// Generate supergraph schema
function generateSupergraphSchema(configPath) {
  try {
    logger.info('🏗️ Generating supergraph schema with Rover...');

    const outputPath = path.join(__dirname, OUTPUT_FILE);
    const command = `rover supergraph compose --config "${configPath}" --output "${outputPath}"`;

    logger.info(`Running: ${command}`);
    const result = execSync(command, {
      encoding: 'utf-8',
      stdio: 'pipe'
    });

    logger.success(`🎉 Supergraph schema generated successfully: ${outputPath}`);
    logger.info('Rover output:', result);

    // Validate the generated schema
    const schemaContent = fs.readFileSync(outputPath, 'utf-8');
    const schemaSize = Math.round(schemaContent.length / 1024);
    logger.info(`📊 Generated schema size: ${schemaSize}KB`);

    // Check for UI interaction features in the supergraph
    const uiFeatures = {
      overrideMutations: schemaContent.includes('resolveClinicalOverride'),
      uiSubscriptions: schemaContent.includes('workflowUIUpdates'),
      realTimeFeatures: schemaContent.includes('overrideRequired'),
      clinicalTypes: schemaContent.includes('OverrideSession')
    };

    const enabledFeatures = Object.entries(uiFeatures)
      .filter(([_, enabled]) => enabled)
      .map(([feature, _]) => feature);

    if (enabledFeatures.length > 0) {
      logger.success(`🎨 UI interaction features enabled: ${enabledFeatures.join(', ')}`);
    } else {
      logger.warn('⚠️ No UI interaction features detected in supergraph');
    }

    return outputPath;
  } catch (error) {
    logger.error('❌ Failed to generate supergraph schema:', error.message);
    throw error;
  }
}

// Update package.json with new script
function updatePackageJson() {
  const packagePath = path.join(__dirname, 'package.json');
  const packageJson = JSON.parse(fs.readFileSync(packagePath, 'utf-8'));

  // Add new scripts for UI interaction
  packageJson.scripts = {
    ...packageJson.scripts,
    'start:ui': 'node workflow-ui-gateway.js',
    'dev:ui': 'nodemon workflow-ui-gateway.js',
    'generate-supergraph:ui': 'node generate-supergraph-with-ui.js',
    'test:ui-integration': 'node test-ui-integration.js'
  };

  // Add new dependencies if needed
  const newDependencies = {
    'graphql-subscriptions': '^2.0.0',
    'graphql-ws': '^5.14.0',
    'ws': '^8.14.2',
    'ioredis': '^5.3.2'
  };

  packageJson.dependencies = {
    ...packageJson.dependencies,
    ...newDependencies
  };

  fs.writeFileSync(packagePath, JSON.stringify(packageJson, null, 2));
  logger.success('📦 Updated package.json with UI interaction scripts and dependencies');
}

// Main execution function
async function main() {
  try {
    logger.info('🚀 Starting enhanced supergraph generation with UI interaction support');

    // Check prerequisites
    if (!checkRoverCLI()) {
      process.exit(1);
    }

    // Check service health
    logger.info('🏥 Checking health of all services...');
    const healthResults = await Promise.all(
      services.map(service => checkServiceHealth(service))
    );

    const healthyServices = services.filter((service, index) =>
      healthResults[index].healthy
    );

    const unhealthyServices = services.filter((service, index) =>
      !healthResults[index].healthy
    );

    logger.info(`📊 Health check results: ${healthyServices.length}/${services.length} services healthy`);

    if (unhealthyServices.length > 0) {
      logger.warn('⚠️ Unhealthy services:');
      unhealthyServices.forEach((service, index) => {
        const result = healthResults[services.indexOf(service)];
        logger.warn(`  - ${service.name}: ${result.error}`);
      });
    }

    if (healthyServices.length === 0) {
      logger.error('❌ No healthy services found. Cannot generate supergraph.');
      process.exit(1);
    }

    // Validate federation schemas
    logger.info('🔍 Validating federation schemas...');
    const schemaResults = await Promise.all(
      healthyServices.map(service => validateFederationSchema(service))
    );

    const validServices = healthyServices.filter((service, index) =>
      schemaResults[index].valid
    );

    logger.info(`📋 Schema validation results: ${validServices.length}/${healthyServices.length} services valid`);

    if (validServices.length === 0) {
      logger.error('❌ No services with valid federation schemas. Cannot generate supergraph.');
      process.exit(1);
    }

    // Create supergraph configuration
    const configPath = createSupergraphConfig(validServices);

    // Generate supergraph schema
    const outputPath = generateSupergraphSchema(configPath);

    // Update package.json
    updatePackageJson();

    // Final summary
    logger.success('🎉 Enhanced supergraph generation completed successfully!');
    logger.info(`📁 Generated files:`);
    logger.info(`   - Supergraph schema: ${outputPath}`);
    logger.info(`   - Configuration: ${configPath}`);
    logger.info(`   - Updated package.json with UI scripts`);

    logger.info('🚀 Next steps:');
    logger.info('   1. Run: npm install (to install new dependencies)');
    logger.info('   2. Run: npm run start:ui (to start enhanced gateway)');
    logger.info('   3. Test: npm run test:ui-integration (to validate UI features)');

    // Check for critical workflow service
    const workflowService = validServices.find(s => s.name === 'workflows');
    if (workflowService) {
      logger.success('✅ Workflow Engine with UI interaction is included in supergraph');
    } else {
      logger.warn('⚠️ Workflow Engine service not available - UI interactions will be limited');
    }

  } catch (error) {
    logger.error('💥 Supergraph generation failed:', error);
    process.exit(1);
  }
}

// Run if this is the main module
if (require.main === module) {
  main();
}

module.exports = {
  main,
  checkServiceHealth,
  validateFederationSchema,
  generateSupergraphSchema
};