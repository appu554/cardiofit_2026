#!/usr/bin/env node
/**
 * Test script to verify Apollo Federation integration with Context Services
 */
const fetch = require('node-fetch');

// Service endpoints
const APOLLO_FEDERATION = 'http://localhost:4000/graphql';
const CONTEXT_GATEWAY = 'http://localhost:8117/api/federation';
const CLINICAL_HUB = 'http://localhost:8118/api/federation';

async function testServiceHealth(name, url) {
  console.log(`🔍 Testing ${name} at ${url}...`);
  
  try {
    const response = await fetch(url, {
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
        console.log(`✅ ${name} federation endpoint is working`);
        console.log(`   SDL length: ${result.data._service.sdl.length} characters`);
        return true;
      } else {
        console.log(`⚠️  ${name} federation endpoint returned unexpected response`);
        console.log(`   Response:`, JSON.stringify(result, null, 2));
        return false;
      }
    } else {
      console.log(`❌ ${name} federation endpoint failed with status ${response.status}`);
      return false;
    }
  } catch (error) {
    console.log(`❌ ${name} federation endpoint error: ${error.message}`);
    return false;
  }
}

async function testFederationQuery() {
  console.log('\n🔍 Testing Apollo Federation gateway...');
  
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
        console.log(`✅ Apollo Federation is working`);
        console.log(`   Query type: ${result.data.__schema.queryType.name}`);
        console.log(`   Available types: ${result.data.__schema.types.length}`);
        
        // Look for context-specific types
        const contextTypes = result.data.__schema.types.filter(type => 
          type.name.includes('Snapshot') || 
          type.name.includes('Recipe') ||
          type.name.includes('Cache') ||
          type.name.includes('Clinical')
        );
        
        if (contextTypes.length > 0) {
          console.log(`   Context-related types found: ${contextTypes.map(t => t.name).join(', ')}`);
        } else {
          console.log(`   ⚠️  No context-related types found in federation schema`);
        }
        
        return true;
      }
    }
    
    console.log(`❌ Apollo Federation query failed`);
    return false;
  } catch (error) {
    console.log(`❌ Apollo Federation error: ${error.message}`);
    return false;
  }
}

async function testEntityResolution() {
  console.log('\n🔍 Testing entity resolution...');
  
  // Test snapshot entity resolution
  const snapshotQuery = `
    query($_representations: [_Any!]!) {
      _entities(representations: $_representations) {
        ... on ClinicalSnapshot {
          id
          recipeId
          patientId
          createdAt
          metadata {
            version
            checksum
          }
        }
      }
    }
  `;
  
  try {
    const response = await fetch(APOLLO_FEDERATION, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        query: snapshotQuery,
        variables: {
          _representations: [
            {
              __typename: 'ClinicalSnapshot',
              id: 'test-snapshot-123'
            }
          ]
        }
      }),
      timeout: 10000
    });

    if (response.ok) {
      const result = await response.json();
      console.log(`✅ Entity resolution working`);
      console.log(`   Response:`, JSON.stringify(result, null, 2));
      return true;
    } else {
      console.log(`❌ Entity resolution failed with status ${response.status}`);
      const text = await response.text();
      console.log(`   Response:`, text);
      return false;
    }
  } catch (error) {
    console.log(`❌ Entity resolution error: ${error.message}`);
    return false;
  }
}

async function testPatientContextQuery() {
  console.log('\n🔍 Testing patient context query...');
  
  const patientQuery = `
    query GetPatientContext($patientId: ID!) {
      patient(id: $patientId) {
        id
        snapshots {
          id
          recipeId
          createdAt
          metadata {
            version
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
          metadata {
            accessCount
            lastAccessed
          }
        }
      }
    }
  `;
  
  try {
    const response = await fetch(APOLLO_FEDERATION, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        query: patientQuery,
        variables: {
          patientId: 'test-patient-456'
        }
      }),
      timeout: 10000
    });

    if (response.ok) {
      const result = await response.json();
      console.log(`✅ Patient context query working`);
      console.log(`   Response:`, JSON.stringify(result, null, 2));
      return true;
    } else {
      console.log(`❌ Patient context query failed with status ${response.status}`);
      const text = await response.text();
      console.log(`   Response:`, text);
      return false;
    }
  } catch (error) {
    console.log(`❌ Patient context query error: ${error.message}`);
    return false;
  }
}

async function main() {
  console.log('🚀 Testing Apollo Federation Integration with Context Services');
  console.log('=' * 70);
  
  const results = {
    contextGateway: false,
    clinicalHub: false,
    federation: false,
    entityResolution: false,
    patientContext: false
  };
  
  // Test individual service federation endpoints
  results.contextGateway = await testServiceHealth('Context Gateway', CONTEXT_GATEWAY);
  results.clinicalHub = await testServiceHealth('Clinical Data Hub', CLINICAL_HUB);
  
  // Test Apollo Federation gateway
  results.federation = await testFederationQuery();
  
  // Test entity resolution
  if (results.federation) {
    results.entityResolution = await testEntityResolution();
  }
  
  // Test complex patient context query
  if (results.federation) {
    results.patientContext = await testPatientContextQuery();
  }
  
  // Summary
  console.log('\n📊 Test Results Summary:');
  console.log('=' * 50);
  
  const tests = [
    { name: 'Context Gateway Federation', result: results.contextGateway },
    { name: 'Clinical Data Hub Federation', result: results.clinicalHub },
    { name: 'Apollo Federation Gateway', result: results.federation },
    { name: 'Entity Resolution', result: results.entityResolution },
    { name: 'Patient Context Query', result: results.patientContext }
  ];
  
  let passedTests = 0;
  
  tests.forEach(test => {
    const status = test.result ? '✅ PASS' : '❌ FAIL';
    console.log(`${status} - ${test.name}`);
    if (test.result) passedTests++;
  });
  
  console.log(`\n📈 Overall: ${passedTests}/${tests.length} tests passed`);
  
  if (passedTests === tests.length) {
    console.log('🎉 All tests passed! Context Services are fully integrated with Apollo Federation.');
  } else if (passedTests >= 3) {
    console.log('⚠️  Most tests passed. Context Services integration is mostly working.');
  } else {
    console.log('❌ Integration needs work. Check service configurations and endpoints.');
  }
  
  console.log('\n🔧 Next Steps:');
  if (!results.contextGateway) {
    console.log('   - Start Context Gateway Go service on port 8117');
    console.log('   - Ensure /api/federation endpoint is working');
  }
  if (!results.clinicalHub) {
    console.log('   - Start Clinical Data Hub Rust service on port 8118');
    console.log('   - Ensure /api/federation endpoint is working');
  }
  if (!results.federation) {
    console.log('   - Start Apollo Federation on port 4000');
    console.log('   - Check federation service configuration');
  }
  
  // Exit with appropriate code
  process.exit(passedTests === tests.length ? 0 : 1);
}

// Handle script errors
process.on('uncaughtException', (error) => {
  console.error('💥 Uncaught Exception:', error);
  process.exit(1);
});

process.on('unhandledRejection', (reason, promise) => {
  console.error('💥 Unhandled Rejection at:', promise, 'reason:', reason);
  process.exit(1);
});

// Run the test suite
main().catch(error => {
  console.error('💥 Test suite error:', error);
  process.exit(1);
});