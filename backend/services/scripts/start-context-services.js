#!/usr/bin/env node
/**
 * Startup script for Context Services (Go Context Gateway + Rust Clinical Data Hub)
 * Handles dependency management, proper startup sequence, and monitoring
 */

const { spawn, exec } = require('child_process');
const path = require('path');
const fs = require('fs');
const fetch = require('node-fetch');

// Service configurations
const SERVICES = {
  contextGateway: {
    name: 'Context Gateway (Go)',
    dir: path.join(__dirname, '../context-gateway-go'),
    cmd: 'go',
    args: ['run', 'cmd/main.go'],
    healthUrl: 'http://localhost:8117/health',
    federationUrl: 'http://localhost:8117/api/federation',
    ports: [8017, 8117] // gRPC, HTTP
  },
  clinicalHub: {
    name: 'Clinical Data Hub (Rust)',
    dir: path.join(__dirname, '../clinical-data-hub-rust'),
    cmd: 'cargo',
    args: ['run'],
    healthUrl: 'http://localhost:8118/health',
    federationUrl: 'http://localhost:8118/api/federation',
    ports: [8018, 8118] // gRPC, HTTP
  }
};

// Infrastructure dependencies
const DEPENDENCIES = {
  redis: {
    name: 'Redis',
    checkCmd: 'redis-cli ping',
    startCmd: 'redis-server',
    healthCheck: async () => {
      try {
        const { stdout } = await execAsync('redis-cli ping');
        return stdout.trim() === 'PONG';
      } catch {
        return false;
      }
    }
  },
  postgres: {
    name: 'PostgreSQL',
    checkCmd: 'pg_isready',
    startCmd: null, // Usually managed separately
    healthCheck: async () => {
      try {
        await execAsync('pg_isready');
        return true;
      } catch {
        return false;
      }
    }
  }
};

const activeProcesses = new Map();
let shutdownInProgress = false;

// Utility functions
function execAsync(command) {
  return new Promise((resolve, reject) => {
    exec(command, (error, stdout, stderr) => {
      if (error) {
        reject({ error, stderr });
      } else {
        resolve({ stdout, stderr });
      }
    });
  });
}

async function waitForPort(port, timeout = 30000, service = 'service') {
  const startTime = Date.now();
  console.log(`   ⏳ Waiting for ${service} on port ${port}...`);
  
  while (Date.now() - startTime < timeout) {
    try {
      const response = await fetch(`http://localhost:${port}`, { 
        timeout: 1000,
        method: 'GET'
      });
      // Any response (even error) means port is open
      console.log(`   ✅ ${service} is responding on port ${port}`);
      return true;
    } catch (error) {
      // Only ECONNREFUSED means port is closed, other errors might mean service is starting
      if (error.code !== 'ECONNREFUSED') {
        console.log(`   ✅ ${service} is responding on port ${port}`);
        return true;
      }
    }
    await new Promise(resolve => setTimeout(resolve, 1000));
  }
  
  console.log(`   ❌ Timeout waiting for ${service} on port ${port}`);
  return false;
}

async function waitForHealthCheck(url, timeout = 30000, serviceName = 'service') {
  const startTime = Date.now();
  console.log(`   🔍 Checking health endpoint: ${url}`);
  
  while (Date.now() - startTime < timeout) {
    try {
      const response = await fetch(url, { timeout: 5000 });
      if (response.ok) {
        console.log(`   ✅ ${serviceName} health check passed`);
        return true;
      }
    } catch (error) {
      // Service might still be starting up
    }
    await new Promise(resolve => setTimeout(resolve, 2000));
  }
  
  console.log(`   ⚠️  ${serviceName} health check timeout (service might still be starting)`);
  return false;
}

function isPortInUse(port) {
  return new Promise((resolve) => {
    const { spawn } = require('child_process');
    const process = spawn('netstat', ['-an']);
    
    let output = '';
    process.stdout.on('data', (data) => {
      output += data.toString();
    });
    
    process.on('close', () => {
      const inUse = output.includes(`:${port} `) || output.includes(`:${port}\t`);
      resolve(inUse);
    });
    
    process.on('error', () => resolve(false));
  });
}

// Check infrastructure dependencies
async function checkDependencies() {
  console.log('🔍 Checking Infrastructure Dependencies...\n');
  
  let allGood = true;
  
  for (const [key, dep] of Object.entries(DEPENDENCIES)) {
    try {
      console.log(`   Checking ${dep.name}...`);
      const isHealthy = await dep.healthCheck();
      
      if (isHealthy) {
        console.log(`   ✅ ${dep.name}: Running`);
      } else {
        console.log(`   ⚠️  ${dep.name}: Not responding (may need manual start)`);
        if (key !== 'postgres') { // PostgreSQL usually managed separately
          allGood = false;
        }
      }
    } catch (error) {
      console.log(`   ⚠️  ${dep.name}: ${error.message}`);
      if (key !== 'postgres') {
        allGood = false;
      }
    }
  }
  
  console.log('');
  return allGood;
}

// Check if services are already running
async function checkExistingServices() {
  console.log('🔍 Checking for Running Services...\n');
  
  for (const [key, service] of Object.entries(SERVICES)) {
    for (const port of service.ports) {
      const inUse = await isPortInUse(port);
      if (inUse) {
        console.log(`   ⚠️  Port ${port} is already in use (${service.name})`);
        console.log('      Consider stopping existing services or check for conflicts');
      }
    }
  }
  
  console.log('');
}

// Build services if needed
async function buildServices() {
  console.log('🔨 Building Services...\n');
  
  // Build Go service
  try {
    console.log('   Building Context Gateway (Go)...');
    const goDir = SERVICES.contextGateway.dir;
    
    if (!fs.existsSync(path.join(goDir, 'go.mod'))) {
      console.log('   ⚠️  go.mod not found, running go mod init...');
      await execAsync(`cd "${goDir}" && go mod init context-gateway-go`);
    }
    
    await execAsync(`cd "${goDir}" && go mod tidy`);
    console.log('   ✅ Go dependencies resolved');
    
    // Check if we need to generate proto files
    const protoDir = path.join(goDir, 'proto');
    if (fs.existsSync(protoDir)) {
      console.log('   📦 Generating Go proto files...');
      await execAsync(`cd "${goDir}" && protoc --go_out=. --go-grpc_out=. proto/*.proto`);
      console.log('   ✅ Proto files generated');
    }
    
  } catch (error) {
    console.log(`   ⚠️  Go build warning: ${error.stderr || error.error?.message}`);
  }

  // Build Rust service
  try {
    console.log('   Building Clinical Data Hub (Rust)...');
    const rustDir = SERVICES.clinicalHub.dir;
    
    await execAsync(`cd "${rustDir}" && cargo check`);
    console.log('   ✅ Rust dependencies resolved');
    
  } catch (error) {
    console.log(`   ⚠️  Rust build warning: ${error.stderr || error.error?.message}`);
  }
  
  console.log('');
}

// Start a single service
async function startService(key, config) {
  return new Promise((resolve, reject) => {
    console.log(`🚀 Starting ${config.name}...`);
    
    const process = spawn(config.cmd, config.args, {
      cwd: config.dir,
      stdio: ['ignore', 'pipe', 'pipe'],
      shell: true
    });
    
    activeProcesses.set(key, process);
    
    // Handle process output
    process.stdout.on('data', (data) => {
      const output = data.toString().trim();
      if (output) {
        console.log(`[${config.name}] ${output}`);
      }
    });
    
    process.stderr.on('data', (data) => {
      const output = data.toString().trim();
      if (output) {
        console.log(`[${config.name}] ${output}`);
      }
    });
    
    // Handle process exit
    process.on('close', (code) => {
      activeProcesses.delete(key);
      if (!shutdownInProgress) {
        console.log(`❌ ${config.name} exited with code ${code}`);
        if (code !== 0) {
          reject(new Error(`${config.name} failed to start`));
        }
      }
    });
    
    process.on('error', (error) => {
      activeProcesses.delete(key);
      console.log(`❌ ${config.name} error: ${error.message}`);
      reject(error);
    });
    
    // Give the service time to start, then check health
    setTimeout(async () => {
      try {
        // Wait for ports to be available
        const mainPort = config.ports[1] || config.ports[0];
        await waitForPort(mainPort, 15000, config.name);
        
        // Wait for health check
        await waitForHealthCheck(config.healthUrl, 20000, config.name);
        
        console.log(`✅ ${config.name} is ready\n`);
        resolve();
      } catch (error) {
        console.log(`⚠️  ${config.name} startup validation failed: ${error.message}\n`);
        resolve(); // Don't fail completely if health check times out
      }
    }, 3000);
  });
}

// Start all services
async function startAllServices() {
  console.log('🚀 Starting Context Services...\n');
  
  const startupPromises = [];
  
  // Start services in parallel
  for (const [key, config] of Object.entries(SERVICES)) {
    startupPromises.push(startService(key, config));
  }
  
  try {
    await Promise.all(startupPromises);
    console.log('🎉 All Context Services started successfully!\n');
    return true;
  } catch (error) {
    console.log(`❌ Service startup failed: ${error.message}\n`);
    return false;
  }
}

// Validate running services
async function validateServices() {
  console.log('🔍 Validating Service Integration...\n');
  
  // Test federation endpoints
  for (const [key, service] of Object.entries(SERVICES)) {
    try {
      console.log(`   Testing ${service.name} federation endpoint...`);
      
      const response = await fetch(service.federationUrl, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          query: '{ _service { sdl } }'
        }),
        timeout: 5000
      });
      
      if (response.ok) {
        const result = await response.json();
        if (result.data && result.data._service) {
          console.log(`   ✅ ${service.name} federation endpoint working`);
        } else {
          console.log(`   ⚠️  ${service.name} federation endpoint: unexpected response`);
        }
      } else {
        console.log(`   ⚠️  ${service.name} federation endpoint: HTTP ${response.status}`);
      }
    } catch (error) {
      console.log(`   ⚠️  ${service.name} federation endpoint: ${error.message}`);
    }
  }
  
  console.log('\n✨ Service validation complete!');
  console.log('\n📋 Service URLs:');
  console.log('   Context Gateway:');
  console.log('     • gRPC: localhost:8017');
  console.log('     • HTTP: http://localhost:8117');
  console.log('     • Health: http://localhost:8117/health');
  console.log('     • Federation: http://localhost:8117/api/federation');
  console.log('   Clinical Data Hub:');
  console.log('     • gRPC: localhost:8018');
  console.log('     • HTTP: http://localhost:8118');
  console.log('     • Health: http://localhost:8118/health');
  console.log('     • Federation: http://localhost:8118/api/federation');
  console.log('\n🎯 Next Steps:');
  console.log('   • Start Apollo Federation: cd apollo-federation && npm start');
  console.log('   • Run validation script: node scripts/validate-context-services.js');
  console.log('   • Test integration: node scripts/test-federation-integration.js');
}

// Graceful shutdown
async function shutdown() {
  if (shutdownInProgress) return;
  shutdownInProgress = true;
  
  console.log('\n🛑 Shutting down Context Services...');
  
  const shutdownPromises = [];
  
  for (const [key, process] of activeProcesses.entries()) {
    shutdownPromises.push(new Promise((resolve) => {
      console.log(`   Stopping ${key}...`);
      
      process.on('close', () => {
        console.log(`   ✅ ${key} stopped`);
        resolve();
      });
      
      // Try graceful shutdown first
      process.kill('SIGTERM');
      
      // Force kill after timeout
      setTimeout(() => {
        if (!process.killed) {
          process.kill('SIGKILL');
          resolve();
        }
      }, 5000);
    }));
  }
  
  await Promise.all(shutdownPromises);
  console.log('✅ All services stopped');
  process.exit(0);
}

// Main function
async function main() {
  console.log('🏥 Context Services Startup Manager');
  console.log('==================================\n');
  
  // Check dependencies
  const depsOk = await checkDependencies();
  if (!depsOk) {
    console.log('⚠️  Some dependencies are not available. Services may have limited functionality.\n');
  }
  
  // Check for existing services
  await checkExistingServices();
  
  // Build services
  await buildServices();
  
  // Start services
  const started = await startAllServices();
  if (!started) {
    console.log('❌ Failed to start all services');
    process.exit(1);
  }
  
  // Validate services
  await validateServices();
  
  console.log('\n🔄 Services are running. Press Ctrl+C to stop.\n');
}

// Handle shutdown signals
process.on('SIGINT', shutdown);
process.on('SIGTERM', shutdown);
process.on('uncaughtException', (error) => {
  console.error('\n💥 Uncaught Exception:', error);
  shutdown();
});

// Run the startup manager
main().catch((error) => {
  console.error('\n💥 Startup error:', error);
  shutdown();
});