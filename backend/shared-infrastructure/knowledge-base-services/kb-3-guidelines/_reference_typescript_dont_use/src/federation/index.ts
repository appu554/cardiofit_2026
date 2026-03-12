// Main entry point for KB-3 Apollo Federation Subgraph
// Starts KB-3 as a federation subgraph ready for gateway composition

import { startKB3Federation } from './subgraph-server';
import { loadConfigFromEnv } from '../index';

async function main() {
  try {
    console.log('🚀 Starting KB-3 Guideline Evidence Federation Subgraph...');
    
    // Load configuration
    const config = loadConfigFromEnv();
    
    // Add federation-specific config
    config.federation = {
      port: parseInt(process.env.KB3_FEDERATION_PORT || '8084'),
      host: process.env.KB3_FEDERATION_HOST || '0.0.0.0',
      subgraph_name: 'kb3-guidelines',
      schema_version: '2.3'
    };
    
    console.log('📋 Configuration loaded:', {
      service_port: config.service.port,
      federation_port: config.federation.port,
      environment: config.service.environment,
      version: config.service.version
    });
    
    // Start federation server
    const { server, url } = await startKB3Federation(config);
    
    console.log('✅ KB-3 Federation Subgraph successfully started!');
    console.log('🔗 Ready for Apollo Gateway composition');
    console.log('📊 GraphQL Playground available for testing');
    
    // Log federation capabilities
    console.log('\n🎯 Federation Capabilities:');
    console.log('  • Extended Patient entities with guidelines and clinical pathways');
    console.log('  • Extended Medication entities with guideline relationships');
    console.log('  • Extended Observation entities with safety triggers');
    console.log('  • Native Guideline, SafetyOverride, and ClinicalPathway entities');
    console.log('  • Cross-KB validation and conflict resolution');
    
    return { server, url };
    
  } catch (error) {
    console.error('❌ Failed to start KB-3 Federation subgraph:', error);
    process.exit(1);
  }
}

// Run if called directly
if (require.main === module) {
  main().catch((error) => {
    console.error('Fatal error:', error);
    process.exit(1);
  });
}

export { main as startKB3FederationSubgraph };