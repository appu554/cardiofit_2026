import { ApolloServer } from '@apollo/server';
import { startStandaloneServer } from '@apollo/server/standalone';
import { ApolloGateway, RemoteGraphQLDataSource } from '@apollo/gateway';
import { readFileSync } from 'fs';
import { join } from 'path';
import dotenv from 'dotenv';
import { createLogger, format, transports } from 'winston';
import { v4 as uuidv4 } from 'uuid';
import crypto from 'crypto';

import { VersionAwareDataSource } from './datasources/VersionAwareDataSource';
import { EvidenceEnvelopePlugin } from './plugins/EvidenceEnvelopePlugin';
import { VersionManagementPlugin } from './plugins/VersionManagementPlugin';
import { AuditLoggingPlugin } from './plugins/AuditLoggingPlugin';
import { MetricsPlugin } from './plugins/MetricsPlugin';
import { KBVersionManager } from './services/KBVersionManager';
import { DatabaseManager } from './database/DatabaseManager';

// Load environment variables
dotenv.config();

// Configure logger
const logger = createLogger({
  level: process.env.LOG_LEVEL || 'info',
  format: format.combine(
    format.timestamp(),
    format.errors({ stack: true }),
    format.json()
  ),
  transports: [
    new transports.Console({
      format: format.combine(
        format.colorize(),
        format.simple()
      )
    }),
    new transports.File({ filename: 'logs/api-gateway.log' })
  ]
});

interface KBServiceConfig {
  name: string;
  url: string;
  healthCheck?: {
    interval: number;
    timeout: number;
    retries: number;
  };
  circuitBreaker?: {
    threshold: number;
    duration: number;
    bucketSize: number;
  };
}

class KnowledgeBrokerGateway {
  private gateway: ApolloGateway;
  private server: ApolloServer;
  private dbManager: DatabaseManager;
  private versionManager: KBVersionManager;
  private services: KBServiceConfig[] = [
    {
      name: 'kb_1_dosing',
      url: process.env.KB_DOSING_URL || 'http://localhost:8081/graphql',
      healthCheck: { interval: 5000, timeout: 1000, retries: 3 },
      circuitBreaker: { threshold: 0.5, duration: 10000, bucketSize: 10 }
    },
    {
      name: 'kb_2_context',
      url: process.env.KB_CONTEXT_URL || 'http://localhost:8082/graphql',
      healthCheck: { interval: 5000, timeout: 1000, retries: 3 },
      circuitBreaker: { threshold: 0.5, duration: 10000, bucketSize: 10 }
    },
    {
      name: 'kb_3_guidelines',
      url: process.env.KB_GUIDELINES_URL || 'http://localhost:8083/graphql',
      healthCheck: { interval: 5000, timeout: 1000, retries: 3 },
      circuitBreaker: { threshold: 0.5, duration: 10000, bucketSize: 10 }
    },
    {
      name: 'kb_4_safety',
      url: process.env.KB_SAFETY_URL || 'http://localhost:8084/graphql',
      healthCheck: { interval: 5000, timeout: 1000, retries: 3 },
      circuitBreaker: { threshold: 0.5, duration: 10000, bucketSize: 10 }
    },
    {
      name: 'kb_5_ddi',
      url: process.env.KB_DDI_URL || 'http://localhost:8085/graphql',
      healthCheck: { interval: 5000, timeout: 1000, retries: 3 },
      circuitBreaker: { threshold: 0.5, duration: 10000, bucketSize: 10 }
    },
    {
      name: 'kb_6_formulary',
      url: process.env.KB_FORMULARY_URL || 'http://localhost:8086/graphql',
      healthCheck: { interval: 5000, timeout: 1000, retries: 3 },
      circuitBreaker: { threshold: 0.5, duration: 10000, bucketSize: 10 }
    },
    {
      name: 'kb_7_terminology',
      url: process.env.KB_TERMINOLOGY_URL || 'http://localhost:8087/graphql',
      healthCheck: { interval: 5000, timeout: 1000, retries: 3 },
      circuitBreaker: { threshold: 0.5, duration: 10000, bucketSize: 10 }
    }
  ];

  constructor() {
    this.dbManager = new DatabaseManager();
    this.versionManager = new KBVersionManager(this.dbManager);
  }

  async initialize(): Promise<void> {
    try {
      logger.info('Initializing Knowledge Broker Gateway...');
      
      // Initialize database connection
      await this.dbManager.connect();
      logger.info('Database connection established');

      // Load active version set
      const activeVersionSet = await this.versionManager.getActiveVersionSet(
        process.env.ENVIRONMENT || 'development'
      );
      logger.info('Active version set loaded', { versionSet: activeVersionSet });

      // Configure Apollo Gateway
      this.gateway = new ApolloGateway({
        supergraphSdl: await this.loadSupergraphSchema(),
        buildService: ({ name, url }) => {
          const config = this.services.find(s => s.name === name);
          return new VersionAwareDataSource({
            url,
            name,
            versionSet: activeVersionSet,
            ...config
          });
        },
        experimental_pollInterval: 10000,
        debug: process.env.NODE_ENV === 'development'
      });

      // Create Apollo Server
      this.server = new ApolloServer({
        gateway: this.gateway,
        
        plugins: [
          new EvidenceEnvelopePlugin(this.dbManager),
          new VersionManagementPlugin(this.versionManager),
          new AuditLoggingPlugin(this.dbManager, logger),
          new MetricsPlugin()
        ],
        
        formatError: (err) => {
          logger.error('GraphQL Error', { error: err });
          return {
            message: err.message,
            locations: err.locations,
            path: err.path,
            extensions: {
              code: err.extensions?.code,
              timestamp: new Date().toISOString(),
              traceId: err.extensions?.traceId
            }
          };
        },

        // Custom context creation
        context: async ({ req }) => {
          const transactionId = this.generateTransactionId();
          const evidenceEnvelopeId = this.generateEvidenceEnvelopeId();
          
          // Determine version set from request headers or use active
          let versionSet = activeVersionSet;
          const requestedVersionSet = req.headers['x-kb-version-set'];
          if (requestedVersionSet) {
            versionSet = await this.versionManager.getVersionSet(requestedVersionSet as string);
          }

          const context = {
            transactionId,
            evidenceEnvelopeId,
            versionSet,
            userContext: await this.extractUserContext(req),
            startTime: Date.now(),
            requestId: req.headers['x-request-id'] || uuidv4(),
            ipAddress: req.ip || req.connection.remoteAddress,
            userAgent: req.headers['user-agent']
          };

          logger.info('Request context created', {
            transactionId,
            evidenceEnvelopeId,
            versionSetId: versionSet.id
          });

          return context;
        }
      });

      logger.info('Knowledge Broker Gateway initialized successfully');
    } catch (error) {
      logger.error('Failed to initialize Knowledge Broker Gateway', { error });
      throw error;
    }
  }

  async start(): Promise<void> {
    try {
      const port = parseInt(process.env.PORT || '4000', 10);
      
      const { url } = await startStandaloneServer(this.server, {
        listen: { port },
        context: async ({ req }) => {
          // Context is already handled in server configuration
          return {};
        }
      });

      logger.info(`🚀 Knowledge Broker Gateway ready at ${url}`);
      logger.info('Available endpoints:');
      logger.info(`  - GraphQL Playground: ${url}`);
      logger.info(`  - Health Check: ${url}health`);
      logger.info(`  - Metrics: ${url}metrics`);
      logger.info(`  - Version Info: ${url}version`);
      
    } catch (error) {
      logger.error('Failed to start server', { error });
      throw error;
    }
  }

  private async loadSupergraphSchema(): Promise<string> {
    try {
      const schemaPath = join(__dirname, '..', 'schema', 'federation.graphql');
      return readFileSync(schemaPath, 'utf8');
    } catch (error) {
      logger.warn('Could not load supergraph schema from file, using default');
      return this.getDefaultSupergraphSchema();
    }
  }

  private getDefaultSupergraphSchema(): string {
    return `
      type Query {
        # Unified clinical decision query
        clinicalDecision(
          input: ClinicalDecisionInput!
          versionSet: String
        ): ClinicalDecisionResponse!
        
        # Individual KB queries with version tracking
        dosing(drugCode: String!, context: PatientContext!): DosingResponse!
        phenotype(patientId: String!): PhenotypeResponse!
        guideline(condition: String!, locale: String): GuidelineResponse!
        
        # Cross-KB impact analysis
        impactAnalysis(
          changeType: ChangeType!
          kbName: String!
          changeId: String!
        ): ImpactAnalysisResponse!
        
        # Real-time safety monitoring
        safetySignals(
          timeWindow: TimeWindow!
          signalType: SignalType
        ): SafetySignalResponse!
        
        # Version management
        activeVersionSet: KBVersionSet!
        versionHistory(kbName: String): [KBVersionSet!]!
      }
      
      type Mutation {
        # Version management
        deployVersionSet(input: VersionSetDeployInput!): DeployResult!
        rollbackVersionSet(versionSetId: String!): RollbackResult!
        
        # Governance
        submitForApproval(changeId: String!, notes: String): ApprovalRequest!
        approveChange(approvalId: String!, decision: ApprovalDecision!): ApprovalResult!
      }
      
      type Subscription {
        # KB version changes
        kbVersionUpdate(kbName: String): KBVersionUpdate!
        
        # Safety signal alerts
        safetySignalAlert(severity: Severity, kbName: String): SafetySignal!
        
        # Governance notifications
        governanceAlert(type: GovernanceAlertType): GovernanceAlert!
      }
    `;
  }

  private generateTransactionId(): string {
    const timestamp = Date.now().toString(36);
    const random = Math.random().toString(36).substr(2, 5);
    return `txn_${timestamp}_${random}`;
  }

  private generateEvidenceEnvelopeId(): string {
    return `env_${uuidv4().replace(/-/g, '')}`;
  }

  private async extractUserContext(req: any): Promise<any> {
    // Extract user context from JWT token or headers
    const authHeader = req.headers.authorization;
    if (authHeader && authHeader.startsWith('Bearer ')) {
      const token = authHeader.substring(7);
      // TODO: Implement JWT decoding and validation
      return { token, userId: 'extracted_from_jwt' };
    }
    
    return { 
      userId: req.headers['x-user-id'] || 'anonymous',
      role: req.headers['x-user-role'] || 'user'
    };
  }

  async shutdown(): Promise<void> {
    logger.info('Shutting down Knowledge Broker Gateway...');
    
    try {
      await this.server.stop();
      await this.dbManager.disconnect();
      logger.info('Knowledge Broker Gateway shut down gracefully');
    } catch (error) {
      logger.error('Error during shutdown', { error });
    }
  }
}

// Main execution
async function main() {
  const gateway = new KnowledgeBrokerGateway();
  
  // Graceful shutdown handling
  process.on('SIGINT', async () => {
    logger.info('Received SIGINT, initiating graceful shutdown...');
    await gateway.shutdown();
    process.exit(0);
  });

  process.on('SIGTERM', async () => {
    logger.info('Received SIGTERM, initiating graceful shutdown...');
    await gateway.shutdown();
    process.exit(0);
  });

  process.on('unhandledRejection', (reason, promise) => {
    logger.error('Unhandled Rejection at:', { promise, reason });
  });

  process.on('uncaughtException', (error) => {
    logger.error('Uncaught Exception:', { error });
    process.exit(1);
  });

  try {
    await gateway.initialize();
    await gateway.start();
  } catch (error) {
    logger.error('Failed to start Knowledge Broker Gateway', { error });
    process.exit(1);
  }
}

// Start the application
if (require.main === module) {
  main();
}

export { KnowledgeBrokerGateway };