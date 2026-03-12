// KB-3 Guideline Evidence Service Entry Point
// Main service initialization and dependency injection

import { KB3GuidelineService } from './api/guideline_service';
import { DatabaseService } from './services/database_service';
import { Neo4jService } from './services/neo4j_service';
import { MultiLayerCache } from './services/cache_service';
import { AuditLogger } from './services/audit_logger';
import { ProductionConflictResolver } from './engines/production_conflict_resolver';
import { SafetyOverrideEngine } from './engines/safety_override_engine';

export interface KB3Config {
  database: {
    host: string;
    port: number;
    database: string;
    username: string;
    password: string;
    ssl?: boolean;
  };
  neo4j: {
    uri: string;
    username: string;
    password: string;
    database?: string;
  };
  cache: {
    redis_url?: string;
    memory_limit?: number;
    ttl_default?: number;
  };
  service: {
    port: number;
    version: string;
    environment: 'development' | 'staging' | 'production';
  };
}

export class KB3ServiceContainer {
  private config: KB3Config;
  private services: {
    database?: DatabaseService;
    neo4j?: Neo4jService;
    cache?: MultiLayerCache;
    auditLogger?: AuditLogger;
    guidelineService?: KB3GuidelineService;
    conflictResolver?: ProductionConflictResolver;
    safetyEngine?: SafetyOverrideEngine;
  } = {};

  constructor(config: KB3Config) {
    this.config = config;
  }

  async initialize(): Promise<void> {
    console.log('Initializing KB-3 Guideline Evidence Service...');
    
    try {
      // Initialize core services
      await this.initializeDatabaseService();
      await this.initializeNeo4jService();
      await this.initializeAuditLogger();
      await this.initializeCacheService();
      
      // Initialize business logic services
      await this.initializeConflictResolver();
      await this.initializeSafetyEngine();
      await this.initializeGuidelineService();
      
      console.log('KB-3 Service initialization complete');
      
      // Health check
      const health = await this.healthCheck();
      if (health.status !== 'healthy') {
        throw new Error(`Service health check failed: ${JSON.stringify(health)}`);
      }
      
    } catch (error) {
      console.error('KB-3 Service initialization failed:', error);
      await this.cleanup();
      throw error;
    }
  }

  private async initializeDatabaseService(): Promise<void> {
    this.services.database = new DatabaseService(this.config.database);
    await this.services.database.initialize();
    console.log('✓ Database service initialized');
  }

  private async initializeNeo4jService(): Promise<void> {
    this.services.neo4j = new Neo4jService(this.config.neo4j);
    await this.services.neo4j.initialize();
    console.log('✓ Neo4j service initialized');
  }

  private async initializeCacheService(): Promise<void> {
    this.services.cache = new MultiLayerCache(
      this.services.auditLogger!,
      this.services.database!
    );
    await this.services.cache.initialize();
    console.log('✓ Cache service initialized');
  }

  private async initializeAuditLogger(): Promise<void> {
    this.services.auditLogger = new AuditLogger(this.services.database!);
    console.log('✓ Audit logger initialized');
  }

  private async initializeConflictResolver(): Promise<void> {
    this.services.conflictResolver = new ProductionConflictResolver(
      this.services.database!,
      this.services.neo4j!,
      this.services.auditLogger!
    );
    console.log('✓ Conflict resolver initialized');
  }

  private async initializeSafetyEngine(): Promise<void> {
    this.services.safetyEngine = new SafetyOverrideEngine(
      this.services.database!,
      this.services.auditLogger!
    );
    await this.services.safetyEngine.initialize();
    console.log('✓ Safety engine initialized');
  }

  private async initializeGuidelineService(): Promise<void> {
    this.services.guidelineService = new KB3GuidelineService(
      this.services.database!,
      this.services.neo4j!,
      this.services.cache!,
      this.services.auditLogger!
    );
    await this.services.guidelineService.initialize();
    console.log('✓ Guideline service initialized');
  }

  // Service access methods
  getGuidelineService(): KB3GuidelineService {
    if (!this.services.guidelineService) {
      throw new Error('Guideline service not initialized');
    }
    return this.services.guidelineService;
  }

  getDatabaseService(): DatabaseService {
    if (!this.services.database) {
      throw new Error('Database service not initialized');
    }
    return this.services.database;
  }

  getNeo4jService(): Neo4jService {
    if (!this.services.neo4j) {
      throw new Error('Neo4j service not initialized');
    }
    return this.services.neo4j;
  }

  getCacheService(): MultiLayerCache {
    if (!this.services.cache) {
      throw new Error('Cache service not initialized');
    }
    return this.services.cache;
  }

  getConflictResolver(): ProductionConflictResolver {
    if (!this.services.conflictResolver) {
      throw new Error('Conflict resolver not initialized');
    }
    return this.services.conflictResolver;
  }

  getSafetyEngine(): SafetyOverrideEngine {
    if (!this.services.safetyEngine) {
      throw new Error('Safety engine not initialized');
    }
    return this.services.safetyEngine;
  }

  // Health and monitoring
  async healthCheck(): Promise<any> {
    const checks = await Promise.allSettled([
      this.services.database?.healthCheck(),
      this.services.neo4j?.healthCheck(),
      this.services.cache?.getStats(),
      this.services.conflictResolver?.healthCheck()
    ]);

    const health = {
      status: 'healthy',
      version: this.config.service.version,
      environment: this.config.service.environment,
      timestamp: new Date().toISOString(),
      services: {
        database: checks[0].status === 'fulfilled' ? checks[0].value : { status: 'unhealthy', error: (checks[0] as any).reason.message },
        neo4j: checks[1].status === 'fulfilled' ? checks[1].value : { status: 'unhealthy', error: (checks[1] as any).reason.message },
        cache: checks[2].status === 'fulfilled' ? checks[2].value : { status: 'unhealthy', error: (checks[2] as any).reason.message },
        conflict_resolver: checks[3].status === 'fulfilled' ? checks[3].value : { status: 'unhealthy', error: (checks[3] as any).reason.message }
      }
    };

    // Determine overall health - require core services to be healthy
    const coreServicesHealthy =
      health.services.database?.status === 'healthy' &&
      health.services.neo4j?.status === 'healthy' &&
      health.services.conflict_resolver?.status === 'healthy';

    if (!coreServicesHealthy) {
      health.status = 'unhealthy';
    } else {
      health.status = 'healthy';
    }

    return health;
  }

  async getMetrics(): Promise<any> {
    const [
      conflictStats,
      dbStats,
      neo4jStats,
      cacheStats
    ] = await Promise.allSettled([
      this.services.conflictResolver?.getConflictStatistics(),
      this.services.database?.getTableStats(),
      this.services.neo4j?.getPerformanceStats(),
      this.services.cache?.getStats()
    ]);

    return {
      conflict_resolution: conflictStats.status === 'fulfilled' ? conflictStats.value : null,
      database: dbStats.status === 'fulfilled' ? dbStats.value : null,
      neo4j: neo4jStats.status === 'fulfilled' ? neo4jStats.value : null,
      cache: cacheStats.status === 'fulfilled' ? cacheStats.value : null,
      timestamp: new Date().toISOString()
    };
  }

  async cleanup(): Promise<void> {
    console.log('Shutting down KB-3 services...');
    
    const cleanupPromises = [];
    
    if (this.services.auditLogger) {
      cleanupPromises.push(this.services.auditLogger.close());
    }
    
    if (this.services.database) {
      cleanupPromises.push(this.services.database.close());
    }
    
    if (this.services.neo4j) {
      cleanupPromises.push(this.services.neo4j.close());
    }
    
    if (this.services.cache) {
      cleanupPromises.push(this.services.cache.close());
    }

    await Promise.allSettled(cleanupPromises);
    console.log('KB-3 services shutdown complete');
  }
}

// Factory function for easy service creation
export async function createKB3Service(config: KB3Config): Promise<KB3ServiceContainer> {
  const container = new KB3ServiceContainer(config);
  await container.initialize();
  return container;
}

// Configuration loading utilities
export function loadConfigFromEnv(): KB3Config {
  return {
    database: {
      host: process.env.KB3_DB_HOST || 'localhost',
      port: parseInt(process.env.KB3_DB_PORT || '5433'),
      database: process.env.KB3_DB_NAME || 'kb3_guidelines',
      username: process.env.KB3_DB_USER || 'kb3_user',
      password: process.env.KB3_DB_PASSWORD || '',
      ssl: process.env.KB3_DB_SSL === 'true'
    },
    neo4j: {
      uri: process.env.KB3_NEO4J_URI || 'bolt://localhost:7687',
      username: process.env.KB3_NEO4J_USER || 'neo4j',
      password: process.env.KB3_NEO4J_PASSWORD || '',
      database: process.env.KB3_NEO4J_DATABASE || 'kb3'
    },
    cache: {
      redis_url: process.env.KB3_REDIS_URL,
      memory_limit: process.env.KB3_MEMORY_CACHE_SIZE ? parseInt(process.env.KB3_MEMORY_CACHE_SIZE) : 209715200, // 200MB
      ttl_default: process.env.KB3_CACHE_TTL ? parseInt(process.env.KB3_CACHE_TTL) : 1800 // 30 minutes
    },
    service: {
      port: parseInt(process.env.KB3_PORT || '8084'),
      version: process.env.KB3_VERSION || '3.0.0',
      environment: (process.env.NODE_ENV as any) || 'development'
    }
  };
}

// Express.js integration helper
export function createExpressApp(container: KB3ServiceContainer) {
  const express = require('express');
  const app = express();
  
  app.use(express.json());
  
  // Health check endpoint
  app.get('/health', async (_req: any, res: any) => {
    try {
      const health = await container.healthCheck();
      res.status(health.status === 'healthy' ? 200 : 503).json(health);
    } catch (error) {
      res.status(503).json({ status: 'unhealthy', error: (error as Error).message });
    }
  });
  
  // Metrics endpoint
  app.get('/metrics', async (_req: any, res: any) => {
    try {
      const metrics = await container.getMetrics();
      res.json(metrics);
    } catch (error) {
      res.status(500).json({ error: (error as Error).message });
    }
  });
  
  // Guidelines endpoint
  app.post('/api/guidelines', async (req: any, res: any) => {
    try {
      const guidelineService = container.getGuidelineService();
      const result = await guidelineService.getGuidelines(req.body);
      res.json(result);
    } catch (error) {
      res.status(500).json({ error: (error as Error).message });
    }
  });
  
  // Clinical pathway endpoint
  app.post('/api/clinical-pathway', async (req: any, res: any) => {
    try {
      const guidelineService = container.getGuidelineService();
      const { conditions, contraindications, region, patientFactors } = req.body;
      
      const result = await guidelineService.getClinicalPathway(
        conditions,
        contraindications,
        region,
        patientFactors
      );
      
      res.json(result);
    } catch (error) {
      res.status(500).json({ error: (error as Error).message });
    }
  });
  
  // Guideline comparison endpoint
  app.post('/api/guidelines/compare', async (req: any, res: any) => {
    try {
      const guidelineService = container.getGuidelineService();
      const { guideline_ids, domain } = req.body;
      
      const result = await guidelineService.compareGuidelines(guideline_ids, domain);
      res.json(result);
    } catch (error) {
      res.status(500).json({ error: (error as Error).message });
    }
  });
  
  // Cross-KB validation endpoint
  app.get('/api/validate/cross-kb', async (_req: any, res: any) => {
    try {
      const guidelineService = container.getGuidelineService();
      const result = await guidelineService.validateCrossKBLinks();
      res.json(result);
    } catch (error) {
      res.status(500).json({ error: (error as Error).message });
    }
  });
  
  // Conflict resolution statistics
  app.get('/api/conflicts/stats', async (req: any, res: any) => {
    try {
      const conflictResolver = container.getConflictResolver();
      const timeRange = req.query.days ? {
        start: new Date(Date.now() - parseInt(req.query.days) * 24 * 60 * 60 * 1000),
        end: new Date()
      } : undefined;
      
      const stats = await conflictResolver.getConflictStatistics(timeRange);
      res.json(stats);
    } catch (error) {
      res.status(500).json({ error: (error as Error).message });
    }
  });
  
  // Error handling middleware
  app.use((error: any, _req: any, res: any, _next: any) => {
    console.error('API Error:', error);
    res.status(500).json({
      error: 'Internal server error',
      message: process.env.NODE_ENV === 'development' ? (error as Error).message : 'Service temporarily unavailable'
    });
  });

  return app;
}

// Main service runner
export async function startKB3Service(config?: KB3Config): Promise<{
  container: KB3ServiceContainer;
  app: any;
  server: any;
}> {
  const serviceConfig = config || loadConfigFromEnv();

  // Initialize service container
  const container = await createKB3Service(serviceConfig);

  // Create Express app
  const app = createExpressApp(container);

  // Start server
  const server = app.listen(serviceConfig.service.port, () => {
    console.log(`KB-3 Guideline Evidence Service running on port ${serviceConfig.service.port}`);
    console.log(`Environment: ${serviceConfig.service.environment}`);
    console.log(`Version: ${serviceConfig.service.version}`);
  });

  // Graceful shutdown handling
  process.on('SIGTERM', async () => {
    console.log('Received SIGTERM, initiating graceful shutdown...');

    server.close(async () => {
      await container.cleanup();
      process.exit(0);
    });
  });

  process.on('SIGINT', async () => {
    console.log('Received SIGINT, initiating graceful shutdown...');

    server.close(async () => {
      await container.cleanup();
      process.exit(0);
    });
  });

  return { container, app, server };
}

// Export main service components
export {
  KB3GuidelineService,
  DatabaseService,
  Neo4jService,
  MultiLayerCache,
  AuditLogger,
  ProductionConflictResolver,
  SafetyOverrideEngine
};

// Export types
export type { PatientContext, SafetyOverride } from './api/guideline_service';
export * from './engines/conflict_resolver';

// Export federation components
export * from './federation/subgraph-server';
export * from './federation/federation-types';
export * from './federation/federation-resolvers';