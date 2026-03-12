#!/usr/bin/env node

/**
 * Test WorkflowEngine Federation Integration
 * This script tests the workflow engine service federation integration
 */

const fs = require('fs');
const path = require('path');

// Logger utility
const logger = {
  info: (msg) => console.log(`ℹ️  ${msg}`),
  success: (msg) => console.log(`✅ ${msg}`),
  error: (msg) => console.error(`❌ ${msg}`),
  warn: (msg) => console.warn(`⚠️  ${msg}`)
};

// Test federation configuration
function testFederationConfig() {
  logger.info('Testing federation configuration files...');
  
  const configFiles = [
    'supergraph.yaml',
    'supergraph-with-workflows.yaml',
    '.env.example'
  ];
  
  let allConfigsValid = true;
  
  for (const configFile of configFiles) {
    const configPath = path.join(__dirname, configFile);
    if (fs.existsSync(configPath)) {
      const content = fs.readFileSync(configPath, 'utf8');
      if (content.includes('workflows') || content.includes('WORKFLOW_ENGINE')) {
        logger.success(`${configFile} includes workflow service configuration`);
      } else {
        logger.error(`${configFile} missing workflow service configuration`);
        allConfigsValid = false;
      }
    } else {
      logger.warn(`${configFile} not found`);
    }
  }
  
  return allConfigsValid;
}

// Test JavaScript configuration files
function testJSConfig() {
  logger.info('Testing JavaScript configuration files...');
  
  const jsFiles = [
    'rover-gateway.js',
    'generate-supergraph.js',
    'index.js'
  ];
  
  let allJSConfigsValid = true;
  
  for (const jsFile of jsFiles) {
    const jsPath = path.join(__dirname, jsFile);
    if (fs.existsSync(jsPath)) {
      const content = fs.readFileSync(jsPath, 'utf8');
      if (content.includes('workflows') && content.includes('8015')) {
        logger.success(`${jsFile} includes workflow service configuration`);
      } else {
        logger.error(`${jsFile} missing workflow service configuration`);
        allJSConfigsValid = false;
      }
    } else {
      logger.error(`${jsFile} not found`);
      allJSConfigsValid = false;
    }
  }
  
  return allJSConfigsValid;
}

// Generate test supergraph config
function generateTestSupergraphConfig() {
  logger.info('Generating test supergraph configuration...');
  
  const testConfig = `federation_version: 2
subgraphs:
  patients:
    routing_url: http://localhost:8003/api/federation
    schema:
      subgraph_url: http://localhost:8003/api/federation
  observations:
    routing_url: http://localhost:8007/api/federation
    schema:
      subgraph_url: http://localhost:8007/api/federation
  medications:
    routing_url: http://localhost:8009/api/federation
    schema:
      subgraph_url: http://localhost:8009/api/federation
  organizations:
    routing_url: http://localhost:8012/api/federation
    schema:
      subgraph_url: http://localhost:8012/api/federation
  orders:
    routing_url: http://localhost:8013/api/federation
    schema:
      subgraph_url: http://localhost:8013/api/federation
  scheduling:
    routing_url: http://localhost:8014/api/federation
    schema:
      subgraph_url: http://localhost:8014/api/federation
  encounters:
    routing_url: http://localhost:8011/api/federation
    schema:
      subgraph_url: http://localhost:8011/api/federation
  workflows:
    routing_url: http://localhost:8015/api/federation
    schema:
      subgraph_url: http://localhost:8015/api/federation
`;

  const configPath = path.join(__dirname, 'supergraph-test.yaml');
  fs.writeFileSync(configPath, testConfig);
  logger.success(`Test supergraph config created at ${configPath}`);
  
  return configPath;
}

// Create federation test queries
function createTestQueries() {
  logger.info('Creating federation test queries...');
  
  const testQueries = {
    "workflow_definitions": {
      "query": `
        query GetWorkflowDefinitions {
          workflowDefinitions {
            id
            name
            version
            status
            category
            description
          }
        }
      `,
      "description": "Get all workflow definitions"
    },
    "patient_with_tasks": {
      "query": `
        query GetPatientWithTasks($patientId: ID!) {
          patient(id: $patientId) {
            id
            name {
              family
              given
            }
            tasks(status: READY) {
              id
              description
              priority
              status
              for {
                reference
                display
              }
            }
            workflowInstances(status: ACTIVE) {
              id
              status
              startTime
            }
          }
        }
      `,
      "variables": {
        "patientId": "patient-123"
      },
      "description": "Get patient with associated tasks and workflow instances"
    },
    "user_assigned_tasks": {
      "query": `
        query GetUserTasks($userId: ID!) {
          user(id: $userId) {
            id
            assignedTasks(status: READY) {
              id
              description
              priority
              status
              for {
                reference
                display
              }
            }
          }
        }
      `,
      "variables": {
        "userId": "user-123"
      },
      "description": "Get tasks assigned to a user"
    },
    "start_workflow": {
      "query": `
        mutation StartWorkflow($patientId: ID!) {
          startWorkflow(
            definitionId: "1"
            patientId: $patientId
            initialVariables: [
              { key: "patientData", value: "{\\"name\\":\\"John Doe\\"}" }
            ]
          ) {
            id
            status
            startTime
          }
        }
      `,
      "variables": {
        "patientId": "patient-123"
      },
      "description": "Start a workflow for a patient"
    }
  };
  
  const queriesPath = path.join(__dirname, 'workflow-federation-test-queries.json');
  fs.writeFileSync(queriesPath, JSON.stringify(testQueries, null, 2));
  logger.success(`Test queries created at ${queriesPath}`);
  
  return queriesPath;
}

// Main test function
function main() {
  logger.info('🧪 Testing WorkflowEngine Federation Integration...');
  
  let allTestsPassed = true;
  
  // Test configuration files
  if (!testFederationConfig()) {
    allTestsPassed = false;
  }
  
  // Test JavaScript files
  if (!testJSConfig()) {
    allTestsPassed = false;
  }
  
  // Generate test configuration
  generateTestSupergraphConfig();
  
  // Create test queries
  createTestQueries();
  
  if (allTestsPassed) {
    logger.success('🎉 All federation configuration tests passed!');
    logger.info('');
    logger.info('Next steps to complete federation integration:');
    logger.info('1. Start WorkflowEngine service: cd ../backend/services/workflow-engine-service && python run_service.py');
    logger.info('2. Start other required services (patients, etc.)');
    logger.info('3. Generate supergraph: node regenerate-supergraph-with-workflows.js');
    logger.info('4. Start Apollo Federation Gateway: npm start');
    logger.info('5. Test federation queries using the generated test queries');
    logger.info('');
    logger.info('Test files created:');
    logger.info('- supergraph-test.yaml: Test supergraph configuration');
    logger.info('- workflow-federation-test-queries.json: Federation test queries');
  } else {
    logger.error('❌ Some federation configuration tests failed!');
    logger.info('Please check the configuration files and try again.');
  }
  
  return allTestsPassed ? 0 : 1;
}

// Run if called directly
if (require.main === module) {
  process.exit(main());
}

module.exports = { main };
