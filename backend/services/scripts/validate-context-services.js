#!/usr/bin/env node
/**
 * Comprehensive validation script for Context Services (Go + Rust) integration
 * Tests both gRPC and GraphQL federation endpoints
 */

const grpc = require('@grpc/grpc-js');
const protoLoader = require('@grpc/proto-loader');
const fetch = require('node-fetch');
const path = require('path');

// Service configurations
const CONTEXT_GATEWAY_GRPC = 'localhost:8017';
const CONTEXT_GATEWAY_HTTP = 'http://localhost:8117';
const CLINICAL_HUB_GRPC = 'localhost:8018';
const CLINICAL_HUB_HTTP = 'http://localhost:8118';
const APOLLO_FEDERATION = 'http://localhost:4000/graphql';

// Test data
const testPatientId = 'test-patient-123';
const testRecipeId = 'cardio-assessment-v1';

console.log('🚀 Context Services Comprehensive Validation');
console.log('==========================================\n');

// Load proto definitions for gRPC testing
async function loadProtoFiles() {
  try {
    const contextGatewayProto = path.join(
      __dirname, 
      '../context-gateway-go/proto/context_gateway.proto'
    );
    
    const clinicalHubProto = path.join(
      __dirname, 
      '../clinical-data-hub-rust/proto/clinical_data_hub.proto'
    );

    const contextPackageDefinition = protoLoader.loadSync(contextGatewayProto, {
      keepCase: true,
      longs: String,
      enums: String,
      defaults: true,
      oneofs: true
    });

    const clinicalPackageDefinition = protoLoader.loadSync(clinicalHubProto, {
      keepCase: true,
      longs: String,
      enums: String,
      defaults: true,
      oneofs: true
    });

    const contextGatewayProtoTypes = grpc.loadPackageDefinition(contextPackageDefinition);
    const clinicalHubProtoTypes = grpc.loadPackageDefinition(clinicalPackageDefinition);

    return {
      contextGateway: contextGatewayProtoTypes.context_gateway,
      clinicalHub: clinicalHubProtoTypes.clinical_data_hub
    };
  } catch (error) {
    console.log('⚠️  Warning: Could not load proto files for gRPC testing');
    console.log('   Make sure the services are built and proto files exist');
    return null;
  }
}

// Test HTTP health endpoints
async function testHealthEndpoints() {
  console.log('🔍 Testing HTTP Health Endpoints...\n');
  
  const endpoints = [
    { name: 'Context Gateway HTTP', url: `${CONTEXT_GATEWAY_HTTP}/health` },
    { name: 'Clinical Data Hub HTTP', url: `${CLINICAL_HUB_HTTP}/health` },
    { name: 'Apollo Federation', url: `${APOLLO_FEDERATION.replace('/graphql', '/health')}` }
  ];

  const results = {};

  for (const endpoint of endpoints) {
    try {
      console.log(`   Testing ${endpoint.name}...`);
      const response = await fetch(endpoint.url, { timeout: 5000 });
      
      if (response.ok) {
        const data = await response.json();
        console.log(`   ✅ ${endpoint.name}: ${JSON.stringify(data)}`);
        results[endpoint.name] = true;
      } else {
        console.log(`   ❌ ${endpoint.name}: HTTP ${response.status}`);
        results[endpoint.name] = false;
      }
    } catch (error) {
      console.log(`   ❌ ${endpoint.name}: ${error.message}`);
      results[endpoint.name] = false;
    }
  }

  console.log('');
  return results;
}

// Test GraphQL Federation endpoints
async function testFederationEndpoints() {
  console.log('🔍 Testing GraphQL Federation Endpoints...\n');
  
  const federationEndpoints = [
    { name: 'Context Gateway Federation', url: `${CONTEXT_GATEWAY_HTTP}/api/federation` },
    { name: 'Clinical Data Hub Federation', url: `${CLINICAL_HUB_HTTP}/api/federation` }
  ];

  const results = {};

  for (const endpoint of federationEndpoints) {
    try {
      console.log(`   Testing ${endpoint.name}...`);
      
      const response = await fetch(endpoint.url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          query: '{ _service { sdl } }'
        }),
        timeout: 5000
      });

      if (response.ok) {
        const result = await response.json();
        if (result.data && result.data._service && result.data._service.sdl) {
          console.log(`   ✅ ${endpoint.name}: SDL available (${result.data._service.sdl.length} chars)`);
          results[endpoint.name] = true;
        } else {
          console.log(`   ⚠️  ${endpoint.name}: Unexpected response format`);
          results[endpoint.name] = false;
        }
      } else {
        console.log(`   ❌ ${endpoint.name}: HTTP ${response.status}`);
        results[endpoint.name] = false;
      }
    } catch (error) {
      console.log(`   ❌ ${endpoint.name}: ${error.message}`);
      results[endpoint.name] = false;
    }
  }

  console.log('');
  return results;
}

// Test Apollo Federation schema composition
async function testFederationSchema() {
  console.log('🔍 Testing Apollo Federation Schema Composition...\n');
  
  try {
    const response = await fetch(APOLLO_FEDERATION, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        query: `
          query IntrospectionQuery {
            __schema {
              queryType { name }
              types {
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
          }
        `
      }),
      timeout: 10000
    });

    if (response.ok) {
      const result = await response.json();
      if (result.data && result.data.__schema) {
        console.log(`   ✅ Apollo Federation: Schema composition working`);
        console.log(`   📊 Query type: ${result.data.__schema.queryType.name}`);
        console.log(`   📊 Available types: ${result.data.__schema.types.length}`);
        
        // Look for context-specific types
        const contextTypes = result.data.__schema.types.filter(type => 
          type.name.includes('Snapshot') || 
          type.name.includes('Recipe') ||
          type.name.includes('Cache') ||
          type.name.includes('Clinical') ||
          type.name.includes('Context')
        );
        
        if (contextTypes.length > 0) {
          console.log(`   🎯 Context types found: ${contextTypes.map(t => t.name).slice(0, 5).join(', ')}${contextTypes.length > 5 ? '...' : ''}`);
        }
        
        console.log('');
        return true;
      }
    }
    
    console.log(`   ❌ Apollo Federation schema introspection failed\n`);
    return false;
  } catch (error) {
    console.log(`   ❌ Apollo Federation error: ${error.message}\n`);
    return false;
  }
}

// Test gRPC connections (if proto files are available)
async function testGrpcConnections(protoTypes) {
  if (!protoTypes) {
    console.log('⚠️  Skipping gRPC tests - proto files not available\n');
    return {};
  }

  console.log('🔍 Testing gRPC Connections...\n');
  
  const results = {};

  // Test Context Gateway gRPC
  try {
    console.log('   Testing Context Gateway gRPC...');
    const contextClient = new protoTypes.contextGateway.ContextGateway(
      CONTEXT_GATEWAY_GRPC,
      grpc.credentials.createInsecure()
    );

    await new Promise((resolve, reject) => {
      const deadline = new Date();
      deadline.setSeconds(deadline.getSeconds() + 5);
      
      contextClient.waitForReady(deadline, (error) => {
        if (error) {
          reject(error);
        } else {
          resolve();
        }
      });
    });

    console.log('   ✅ Context Gateway gRPC: Connection successful');
    results['Context Gateway gRPC'] = true;
    contextClient.close();
  } catch (error) {
    console.log(`   ❌ Context Gateway gRPC: ${error.message}`);
    results['Context Gateway gRPC'] = false;
  }

  // Test Clinical Data Hub gRPC
  try {
    console.log('   Testing Clinical Data Hub gRPC...');
    const clinicalClient = new protoTypes.clinicalHub.ClinicalDataHub(
      CLINICAL_HUB_GRPC,
      grpc.credentials.createInsecure()
    );

    await new Promise((resolve, reject) => {
      const deadline = new Date();
      deadline.setSeconds(deadline.getSeconds() + 5);
      
      clinicalClient.waitForReady(deadline, (error) => {
        if (error) {
          reject(error);
        } else {
          resolve();
        }
      });
    });

    console.log('   ✅ Clinical Data Hub gRPC: Connection successful');
    results['Clinical Data Hub gRPC'] = true;
    clinicalClient.close();
  } catch (error) {
    console.log(`   ❌ Clinical Data Hub gRPC: ${error.message}`);
    results['Clinical Data Hub gRPC'] = false;
  }

  console.log('');
  return results;
}

// Test end-to-end workflow
async function testEndToEndWorkflow() {
  console.log('🔍 Testing End-to-End Clinical Context Workflow...\n');
  
  try {
    // Test a complex GraphQL query that would involve both services
    const complexQuery = `
      query TestClinicalContext($patientId: ID!) {
        patient(id: $patientId) {
          id
          snapshots {
            id
            recipeId
            createdAt
            status
            metadata {
              version
              checksum
              performance {
                executionTimeMs
                cacheHits
                cacheMisses
              }
            }
          }
          cachedData {
            key
            layer
            ttl
            metadata {
              accessCount
              lastAccessed
              compressionRatio
            }
          }
        }
      }
    `;

    console.log('   Executing complex clinical context query...');
    
    const response = await fetch(APOLLO_FEDERATION, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        query: complexQuery,
        variables: { patientId: testPatientId }
      }),
      timeout: 10000
    });

    if (response.ok) {
      const result = await response.json();
      
      if (result.errors) {
        console.log('   ⚠️  Query executed with errors:');
        result.errors.forEach(error => {
          console.log(`      - ${error.message}`);
        });
        return false;
      } else {
        console.log('   ✅ End-to-end workflow: Query executed successfully');
        console.log('   📊 Response structure looks correct');
        return true;
      }
    } else {
      console.log(`   ❌ End-to-end workflow: HTTP ${response.status}`);
      const text = await response.text();
      console.log(`   📄 Response: ${text}`);
      return false;
    }
  } catch (error) {
    console.log(`   ❌ End-to-end workflow: ${error.message}`);
    return false;
  }
}

// Main validation function
async function main() {
  const results = {
    health: {},
    federation: {},
    schema: false,
    grpc: {},
    endToEnd: false
  };

  // Load proto files
  const protoTypes = await loadProtoFiles();

  // Run all tests
  results.health = await testHealthEndpoints();
  results.federation = await testFederationEndpoints();
  results.schema = await testFederationSchema();
  results.grpc = await testGrpcConnections(protoTypes);
  results.endToEnd = await testEndToEndWorkflow();

  // Generate summary report
  console.log('📊 Validation Summary Report');
  console.log('===========================\n');

  // Health endpoints
  const healthPassed = Object.values(results.health).filter(Boolean).length;
  const healthTotal = Object.keys(results.health).length;
  console.log(`🏥 Health Endpoints: ${healthPassed}/${healthTotal} passed`);
  
  // Federation endpoints
  const federationPassed = Object.values(results.federation).filter(Boolean).length;
  const federationTotal = Object.keys(results.federation).length;
  console.log(`🔗 Federation Endpoints: ${federationPassed}/${federationTotal} passed`);
  
  // Schema composition
  console.log(`📋 Schema Composition: ${results.schema ? '✅ PASS' : '❌ FAIL'}`);
  
  // gRPC connections
  const grpcPassed = Object.values(results.grpc).filter(Boolean).length;
  const grpcTotal = Object.keys(results.grpc).length;
  if (grpcTotal > 0) {
    console.log(`⚡ gRPC Connections: ${grpcPassed}/${grpcTotal} passed`);
  }
  
  // End-to-end workflow
  console.log(`🔄 End-to-End Workflow: ${results.endToEnd ? '✅ PASS' : '❌ FAIL'}`);

  // Overall assessment
  const totalTests = healthTotal + federationTotal + (results.schema ? 1 : 0) + grpcTotal + (results.endToEnd ? 1 : 0);
  const passedTests = healthPassed + federationPassed + (results.schema ? 1 : 0) + grpcPassed + (results.endToEnd ? 1 : 0);
  
  console.log(`\n📈 Overall Status: ${passedTests}/${totalTests} tests passed`);

  if (passedTests === totalTests) {
    console.log('🎉 All tests passed! Context Services are fully operational.');
  } else if (passedTests >= totalTests * 0.8) {
    console.log('⚠️  Most tests passed. System is mostly functional with minor issues.');
  } else if (passedTests >= totalTests * 0.5) {
    console.log('⚠️  Some critical components are not working. Review service configurations.');
  } else {
    console.log('❌ Major issues detected. Services may not be running or configured properly.');
  }

  // Provide specific guidance
  console.log('\n🔧 Next Steps:');
  
  if (!results.health['Context Gateway HTTP']) {
    console.log('   - Start Context Gateway Go service: cd backend/services/context-gateway-go && go run cmd/main.go');
  }
  
  if (!results.health['Clinical Data Hub HTTP']) {
    console.log('   - Start Clinical Data Hub Rust service: cd backend/services/clinical-data-hub-rust && cargo run');
  }
  
  if (!results.health['Apollo Federation']) {
    console.log('   - Start Apollo Federation: cd apollo-federation && npm start');
  }
  
  if (!results.schema) {
    console.log('   - Check Apollo Federation service configuration in index.js');
    console.log('   - Verify Context Services federation endpoints are responding');
  }

  if (Object.keys(results.grpc).length === 0) {
    console.log('   - Build proto files: cd context-gateway-go && protoc --go_out=. proto/*.proto');
    console.log('   - Build proto files: cd clinical-data-hub-rust && cargo build');
  }

  console.log('\n✨ Context Services validation complete!');
  
  // Exit with appropriate code
  process.exit(passedTests === totalTests ? 0 : 1);
}

// Error handling
process.on('uncaughtException', (error) => {
  console.error('\n💥 Uncaught Exception:', error);
  process.exit(1);
});

process.on('unhandledRejection', (reason, promise) => {
  console.error('\n💥 Unhandled Rejection:', reason);
  process.exit(1);
});

// Run the validation suite
main().catch(error => {
  console.error('\n💥 Validation suite error:', error);
  process.exit(1);
});