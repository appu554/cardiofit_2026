#!/usr/bin/env node

/**
 * Simple test script to check all federation endpoints
 */

const { execSync } = require('child_process');

// Service configuration
const services = [
  { name: 'patients', port: 8003 },
  { name: 'observations', port: 8007 },
  { name: 'medications', port: 8009 },
  { name: 'organizations', port: 8012 },
  { name: 'orders', port: 8013 },
  { name: 'scheduling', port: 8014 },
  { name: 'encounters', port: 8020 }
];

// Test each service with Apollo Rover
async function testAllServices() {
  console.log('🚀 Testing all federation endpoints with Apollo Rover...\n');
  
  const results = {};
  
  for (const service of services) {
    const url = `http://localhost:${service.port}/api/federation`;
    
    try {
      console.log(`Testing ${service.name} (${url})...`);
      
      const command = `npx @apollo/rover subgraph introspect ${url}`;
      const output = execSync(command, { 
        encoding: 'utf8',
        timeout: 15000,
        stdio: 'pipe'
      });
      
      console.log(`✅ ${service.name}: SUCCESS`);
      results[service.name] = { success: true, error: null };
      
    } catch (error) {
      console.log(`❌ ${service.name}: FAILED`);
      console.log(`   Error: ${error.message}`);
      if (error.stderr) {
        console.log(`   Stderr: ${error.stderr.toString().substring(0, 200)}...`);
      }
      results[service.name] = { success: false, error: error.message };
    }
    
    console.log('');
  }
  
  // Summary
  console.log('📊 Summary:');
  const successful = Object.values(results).filter(r => r.success).length;
  const total = services.length;
  
  console.log(`Successful: ${successful}/${total}`);
  
  if (successful === total) {
    console.log('\n🎉 All services are working! Try supergraph generation:');
    console.log('node generate-supergraph.js');
  } else {
    console.log('\n❌ Some services failed. Fix the failing services first.');
    
    // Show which services failed
    Object.entries(results).forEach(([name, result]) => {
      if (!result.success) {
        console.log(`  - ${name}: ${result.error}`);
      }
    });
  }
  
  return successful === total;
}

// Test supergraph generation if all services work
async function testSupergraphGeneration() {
  try {
    console.log('\n🔧 Testing supergraph generation...');
    
    const command = 'node generate-supergraph.js';
    const output = execSync(command, { 
      encoding: 'utf8',
      timeout: 60000,
      stdio: 'pipe'
    });
    
    console.log('✅ Supergraph generation successful!');
    console.log(output);
    return true;
    
  } catch (error) {
    console.log('❌ Supergraph generation failed:');
    console.log(error.message);
    if (error.stdout) {
      console.log('Stdout:', error.stdout.toString());
    }
    if (error.stderr) {
      console.log('Stderr:', error.stderr.toString());
    }
    return false;
  }
}

// Main function
async function main() {
  const allServicesWork = await testAllServices();
  
  if (allServicesWork) {
    await testSupergraphGeneration();
  }
}

// Run if called directly
if (require.main === module) {
  main().catch(error => {
    console.error('Test execution failed:', error);
    process.exit(1);
  });
}

module.exports = { testAllServices, testSupergraphGeneration };
