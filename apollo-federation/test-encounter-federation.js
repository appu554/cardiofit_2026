#!/usr/bin/env node

/**
 * Test EncounterManagementService Federation Compatibility
 * 
 * This script tests the encounter service federation endpoint specifically
 * to identify any issues with Apollo Federation compatibility.
 */

const { execSync } = require('child_process');
// Use built-in fetch (Node.js 18+) or fallback to a simple HTTP request
const fetch = globalThis.fetch || require('https').request;

// Test configuration
const ENCOUNTER_SERVICE_URL = 'http://localhost:8020/api/federation';

// Logging utility
const logger = {
  info: (message) => console.log(`[INFO] ${new Date().toISOString()} - ${message}`),
  error: (message, error) => console.error(`[ERROR] ${new Date().toISOString()} - ${message}`, error),
  success: (message) => console.log(`[SUCCESS] ${new Date().toISOString()} - ${message}`),
  warn: (message) => console.warn(`[WARN] ${new Date().toISOString()} - ${message}`)
};

// Test basic GraphQL introspection
async function testBasicIntrospection() {
  try {
    logger.info('Testing basic GraphQL introspection...');
    
    const query = {
      query: `
        query IntrospectionQuery {
          __schema {
            queryType { name }
            mutationType { name }
            types {
              name
              kind
              description
            }
          }
        }
      `
    };

    const response = await fetch(ENCOUNTER_SERVICE_URL, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(query)
    });

    if (!response.ok) {
      logger.error(`HTTP Error: ${response.status} ${response.statusText}`);
      return false;
    }

    const data = await response.json();
    
    if (data.errors) {
      logger.error('GraphQL Errors:', data.errors);
      return false;
    }

    if (data.data && data.data.__schema) {
      logger.success('✅ Basic introspection working');
      logger.info(`Query Type: ${data.data.__schema.queryType?.name}`);
      logger.info(`Mutation Type: ${data.data.__schema.mutationType?.name}`);
      logger.info(`Total Types: ${data.data.__schema.types.length}`);
      return true;
    } else {
      logger.error('Invalid introspection response');
      return false;
    }
  } catch (error) {
    logger.error('Basic introspection failed:', error.message);
    return false;
  }
}

// Test federation-specific introspection
async function testFederationIntrospection() {
  try {
    logger.info('Testing federation-specific introspection...');
    
    const query = {
      query: `
        query FederationIntrospection {
          _service {
            sdl
          }
        }
      `
    };

    const response = await fetch(ENCOUNTER_SERVICE_URL, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(query)
    });

    if (!response.ok) {
      logger.error(`HTTP Error: ${response.status} ${response.statusText}`);
      return false;
    }

    const data = await response.json();
    
    if (data.errors) {
      logger.error('Federation introspection errors:', data.errors);
      return false;
    }

    if (data.data && data.data._service && data.data._service.sdl) {
      logger.success('✅ Federation introspection working');
      logger.info('SDL length:', data.data._service.sdl.length);
      
      // Check for federation directives
      const sdl = data.data._service.sdl;
      const hasKey = sdl.includes('@key');
      const hasExternal = sdl.includes('@external');
      const hasShareable = sdl.includes('@shareable');
      
      logger.info(`Federation directives found:`);
      logger.info(`  @key: ${hasKey ? '✅' : '❌'}`);
      logger.info(`  @external: ${hasExternal ? '✅' : '❌'}`);
      logger.info(`  @shareable: ${hasShareable ? '✅' : '❌'}`);
      
      return true;
    } else {
      logger.error('No federation SDL found');
      return false;
    }
  } catch (error) {
    logger.error('Federation introspection failed:', error.message);
    return false;
  }
}

// Test Apollo Rover introspection
async function testRoverIntrospection() {
  try {
    logger.info('Testing Apollo Rover introspection...');
    
    const command = `npx @apollo/rover subgraph introspect ${ENCOUNTER_SERVICE_URL}`;
    logger.info(`Running: ${command}`);
    
    const output = execSync(command, { 
      encoding: 'utf8',
      timeout: 30000
    });
    
    logger.success('✅ Apollo Rover introspection successful');
    logger.info('Schema preview (first 500 chars):');
    logger.info(output.substring(0, 500) + '...');
    
    return true;
  } catch (error) {
    logger.error('Apollo Rover introspection failed:');
    logger.error('Error message:', error.message);
    if (error.stdout) {
      logger.error('Stdout:', error.stdout.toString());
    }
    if (error.stderr) {
      logger.error('Stderr:', error.stderr.toString());
    }
    return false;
  }
}

// Test specific encounter types
async function testEncounterTypes() {
  try {
    logger.info('Testing encounter-specific types...');
    
    const query = {
      query: `
        query EncounterTypes {
          __type(name: "Encounter") {
            name
            kind
            fields {
              name
              type {
                name
                kind
              }
            }
          }
        }
      `
    };

    const response = await fetch(ENCOUNTER_SERVICE_URL, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(query)
    });

    const data = await response.json();
    
    if (data.errors) {
      logger.error('Encounter type query errors:', data.errors);
      return false;
    }

    if (data.data && data.data.__type) {
      const encounterType = data.data.__type;
      logger.success(`✅ Encounter type found with ${encounterType.fields.length} fields`);
      
      // Check for required fields
      const requiredFields = ['id', 'status', 'encounterClass'];
      const foundFields = encounterType.fields.map(f => f.name);
      
      requiredFields.forEach(field => {
        if (foundFields.includes(field)) {
          logger.info(`  ✅ ${field}`);
        } else {
          logger.warn(`  ❌ ${field} (missing)`);
        }
      });
      
      return true;
    } else {
      logger.error('Encounter type not found');
      return false;
    }
  } catch (error) {
    logger.error('Encounter types test failed:', error.message);
    return false;
  }
}

// Main test function
async function runTests() {
  logger.info('🚀 Testing EncounterManagementService Federation Compatibility\n');
  
  const results = {
    basicIntrospection: false,
    federationIntrospection: false,
    roverIntrospection: false,
    encounterTypes: false
  };
  
  // Run tests
  results.basicIntrospection = await testBasicIntrospection();
  console.log('');
  
  results.federationIntrospection = await testFederationIntrospection();
  console.log('');
  
  results.encounterTypes = await testEncounterTypes();
  console.log('');
  
  results.roverIntrospection = await testRoverIntrospection();
  console.log('');
  
  // Summary
  logger.info('📊 Test Results Summary:');
  logger.info(`Basic Introspection: ${results.basicIntrospection ? '✅ PASS' : '❌ FAIL'}`);
  logger.info(`Federation Introspection: ${results.federationIntrospection ? '✅ PASS' : '❌ FAIL'}`);
  logger.info(`Encounter Types: ${results.encounterTypes ? '✅ PASS' : '❌ FAIL'}`);
  logger.info(`Apollo Rover: ${results.roverIntrospection ? '✅ PASS' : '❌ FAIL'}`);
  
  const allPassed = Object.values(results).every(result => result);
  
  if (allPassed) {
    logger.success('\n🎉 All tests passed! The service should work with Apollo Federation.');
  } else {
    logger.error('\n❌ Some tests failed. This explains why supergraph generation is failing.');
    
    if (!results.roverIntrospection) {
      logger.info('\n💡 The Apollo Rover introspection failure is likely the root cause.');
      logger.info('Check the error details above to identify the specific issue.');
    }
  }
  
  return allPassed;
}

// Run tests if called directly
if (require.main === module) {
  runTests().catch(error => {
    logger.error('Test execution failed:', error);
    process.exit(1);
  });
}

module.exports = { runTests };
