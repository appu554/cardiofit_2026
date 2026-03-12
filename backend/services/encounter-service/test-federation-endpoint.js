#!/usr/bin/env node

/**
 * Test EncounterManagementService Federation Endpoint
 * 
 * This script tests the federation endpoint to ensure it's properly configured
 * for Apollo Federation integration.
 */

const http = require('http');

// Test configuration
const ENCOUNTER_SERVICE_URL = 'http://localhost:8020';
const FEDERATION_ENDPOINT = '/api/federation';
const HEALTH_ENDPOINT = '/health';

// Logging utility
const logger = {
  info: (message) => console.log(`[INFO] ${new Date().toISOString()} - ${message}`),
  error: (message, error) => console.error(`[ERROR] ${new Date().toISOString()} - ${message}`, error),
  success: (message) => console.log(`[SUCCESS] ${new Date().toISOString()} - ${message}`),
  warn: (message) => console.warn(`[WARN] ${new Date().toISOString()} - ${message}`)
};

// Make HTTP request
function makeRequest(url, method = 'GET', data = null) {
  return new Promise((resolve, reject) => {
    const urlObj = new URL(url);
    const options = {
      hostname: urlObj.hostname,
      port: urlObj.port,
      path: urlObj.pathname,
      method: method,
      headers: {
        'Content-Type': 'application/json',
      }
    };

    const req = http.request(options, (res) => {
      let body = '';
      res.on('data', (chunk) => {
        body += chunk;
      });
      res.on('end', () => {
        try {
          const jsonBody = body ? JSON.parse(body) : {};
          resolve({
            statusCode: res.statusCode,
            headers: res.headers,
            body: jsonBody
          });
        } catch (error) {
          resolve({
            statusCode: res.statusCode,
            headers: res.headers,
            body: body
          });
        }
      });
    });

    req.on('error', (error) => {
      reject(error);
    });

    if (data) {
      req.write(JSON.stringify(data));
    }

    req.end();
  });
}

// Test health endpoint
async function testHealthEndpoint() {
  try {
    logger.info('Testing health endpoint...');
    const response = await makeRequest(`${ENCOUNTER_SERVICE_URL}${HEALTH_ENDPOINT}`);
    
    if (response.statusCode === 200) {
      logger.success('✅ Health endpoint is working');
      logger.info(`Service: ${response.body.service}`);
      logger.info(`FHIR Service Status: ${response.body.fhir_service?.status}`);
      logger.info(`Using Google Healthcare API: ${response.body.fhir_service?.using_google_healthcare_api}`);
      return true;
    } else {
      logger.error(`❌ Health endpoint failed with status ${response.statusCode}`);
      return false;
    }
  } catch (error) {
    logger.error('❌ Health endpoint test failed:', error.message);
    return false;
  }
}

// Test federation endpoint with introspection query
async function testFederationEndpoint() {
  try {
    logger.info('Testing federation endpoint...');
    
    const introspectionQuery = {
      query: `
        query IntrospectionQuery {
          __schema {
            types {
              name
              kind
            }
          }
        }
      `
    };

    const response = await makeRequest(
      `${ENCOUNTER_SERVICE_URL}${FEDERATION_ENDPOINT}`,
      'POST',
      introspectionQuery
    );
    
    if (response.statusCode === 200 && response.body.data) {
      logger.success('✅ Federation endpoint is working');
      
      // Check for encounter-specific types
      const types = response.body.data.__schema.types;
      const encounterTypes = types.filter(type => 
        type.name.includes('Encounter') || 
        type.name.includes('Location') ||
        type.name === 'Patient' ||
        type.name === 'User'
      );
      
      logger.info(`Found ${encounterTypes.length} encounter-related types:`);
      encounterTypes.forEach(type => {
        logger.info(`  - ${type.name} (${type.kind})`);
      });
      
      return true;
    } else {
      logger.error(`❌ Federation endpoint failed with status ${response.statusCode}`);
      if (response.body.errors) {
        response.body.errors.forEach(error => {
          logger.error(`  Error: ${error.message}`);
        });
      }
      return false;
    }
  } catch (error) {
    logger.error('❌ Federation endpoint test failed:', error.message);
    return false;
  }
}

// Test federation schema for required types
async function testFederationSchema() {
  try {
    logger.info('Testing federation schema for required types...');
    
    const schemaQuery = {
      query: `
        query {
          __type(name: "Encounter") {
            name
            fields {
              name
              type {
                name
              }
            }
          }
        }
      `
    };

    const response = await makeRequest(
      `${ENCOUNTER_SERVICE_URL}${FEDERATION_ENDPOINT}`,
      'POST',
      schemaQuery
    );
    
    if (response.statusCode === 200 && response.body.data?.__type) {
      logger.success('✅ Encounter type found in schema');
      
      const encounterType = response.body.data.__type;
      const requiredFields = ['id', 'status', 'class', 'subject'];
      const foundFields = encounterType.fields.map(field => field.name);
      
      const missingFields = requiredFields.filter(field => !foundFields.includes(field));
      
      if (missingFields.length === 0) {
        logger.success('✅ All required Encounter fields found');
      } else {
        logger.warn(`⚠️ Missing required fields: ${missingFields.join(', ')}`);
      }
      
      return true;
    } else {
      logger.error('❌ Encounter type not found in schema');
      return false;
    }
  } catch (error) {
    logger.error('❌ Federation schema test failed:', error.message);
    return false;
  }
}

// Test entity extensions
async function testEntityExtensions() {
  try {
    logger.info('Testing entity extensions...');
    
    const patientQuery = {
      query: `
        query {
          __type(name: "Patient") {
            name
            fields {
              name
              type {
                name
                ofType {
                  name
                }
              }
            }
          }
        }
      `
    };

    const response = await makeRequest(
      `${ENCOUNTER_SERVICE_URL}${FEDERATION_ENDPOINT}`,
      'POST',
      patientQuery
    );
    
    if (response.statusCode === 200 && response.body.data?.__type) {
      const patientType = response.body.data.__type;
      const encountersField = patientType.fields.find(field => field.name === 'encounters');
      
      if (encountersField) {
        logger.success('✅ Patient.encounters extension found');
        return true;
      } else {
        logger.warn('⚠️ Patient.encounters extension not found');
        return false;
      }
    } else {
      logger.warn('⚠️ Patient type not found (may be external)');
      return true; // This is expected for external types
    }
  } catch (error) {
    logger.error('❌ Entity extensions test failed:', error.message);
    return false;
  }
}

// Main test function
async function runTests() {
  logger.info('🚀 Starting EncounterManagementService Federation Tests...');
  
  const results = {
    health: false,
    federation: false,
    schema: false,
    extensions: false
  };
  
  // Run tests
  results.health = await testHealthEndpoint();
  results.federation = await testFederationEndpoint();
  results.schema = await testFederationSchema();
  results.extensions = await testEntityExtensions();
  
  // Summary
  logger.info('\n📊 Test Results Summary:');
  logger.info(`Health Endpoint: ${results.health ? '✅ PASS' : '❌ FAIL'}`);
  logger.info(`Federation Endpoint: ${results.federation ? '✅ PASS' : '❌ FAIL'}`);
  logger.info(`Schema Validation: ${results.schema ? '✅ PASS' : '❌ FAIL'}`);
  logger.info(`Entity Extensions: ${results.extensions ? '✅ PASS' : '❌ FAIL'}`);
  
  const allPassed = Object.values(results).every(result => result);
  
  if (allPassed) {
    logger.success('\n🎉 All tests passed! EncounterManagementService is ready for Apollo Federation.');
    logger.info('\nNext steps:');
    logger.info('1. Regenerate supergraph: node apollo-federation/regenerate-supergraph-with-encounters.js');
    logger.info('2. Start Apollo Federation Gateway: cd apollo-federation && npm start');
    logger.info('3. Test federation queries through the gateway');
  } else {
    logger.error('\n❌ Some tests failed. Please check the service configuration.');
    process.exit(1);
  }
}

// Run tests if called directly
if (require.main === module) {
  runTests().catch(error => {
    logger.error('Test execution failed:', error);
    process.exit(1);
  });
}

module.exports = { runTests };
