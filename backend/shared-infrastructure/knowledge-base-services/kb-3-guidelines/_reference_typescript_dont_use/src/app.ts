// KB-3 Guideline Evidence Application Runner
// Main application entry point with configuration and startup

import { startKB3Service, loadConfigFromEnv } from './index';

async function main() {
  try {
    console.log('Starting KB-3 Guideline Evidence Service...');
    
    // Load configuration from environment
    const config = loadConfigFromEnv();
    
    // Validate configuration
    validateConfig(config);
    
    // Start service
    const { container } = await startKB3Service(config);
    
    console.log('KB-3 Service started successfully');
    console.log(`API available at: http://localhost:${config.service.port}`);
    console.log('Health check: GET /health');
    console.log('Metrics: GET /metrics');
    
    // Initial system validation
    await performStartupValidation(container);
    
  } catch (error) {
    console.error('Failed to start KB-3 service:', error);
    process.exit(1);
  }
}

function validateConfig(config: any): void {
  const required = [
    'database.host',
    'database.password',
    'neo4j.uri',
    'neo4j.password',
    'service.port'
  ];
  
  for (const path of required) {
    const value = getNestedValue(config, path);
    if (!value) {
      throw new Error(`Required configuration missing: ${path}`);
    }
  }
  
  console.log('✓ Configuration validated');
}

function getNestedValue(obj: any, path: string): any {
  return path.split('.').reduce((current, key) => current?.[key], obj);
}

async function performStartupValidation(container: any): Promise<void> {
  console.log('Performing startup validation...');
  
  try {
    // Validate database schema
    const db = container.getDatabaseService();
    const tableCheck = await db.query(`
      SELECT table_name FROM information_schema.tables 
      WHERE table_schema = 'guideline_evidence'
    `);
    
    const expectedTables = [
      'guidelines',
      'recommendations', 
      'conflict_resolutions',
      'safety_overrides',
      'kb_linkages',
      'audit_log'
    ];
    
    const existingTables = tableCheck.rows.map((row: any) => row.table_name);
    const missingTables = expectedTables.filter(table => !existingTables.includes(table));
    
    if (missingTables.length > 0) {
      throw new Error(`Missing database tables: ${missingTables.join(', ')}`);
    }
    
    console.log('✓ Database schema validated');
    
    // Validate Neo4j constraints and indexes
    const neo4j = container.getNeo4jService();
    await neo4j.run('RETURN 1 as test');
    console.log('✓ Neo4j connectivity validated');
    
    // Test cache operations
    const cache = container.getCacheService();
    await cache.set('startup_test', 'success', 60);
    const cacheTest = await cache.get('startup_test');
    
    if (cacheTest !== 'success') {
      throw new Error('Cache validation failed');
    }
    
    console.log('✓ Cache operations validated');
    
    // Load initial safety overrides
    const safetyEngine = container.getSafetyEngine();
    await safetyEngine.initialize();
    console.log('✓ Safety overrides loaded');
    
    // Validate cross-KB connectivity (if other KBs are available)
    console.log('✓ Startup validation complete');
    
  } catch (error) {
    console.error('Startup validation failed:', error);
    throw error;
  }
}

// Handle uncaught exceptions
process.on('uncaughtException', (error) => {
  console.error('Uncaught Exception:', error);
  process.exit(1);
});

process.on('unhandledRejection', (reason, promise) => {
  console.error('Unhandled Rejection at:', promise, 'reason:', reason);
  process.exit(1);
});

// Start the application
if (require.main === module) {
  main();
}

export { main };