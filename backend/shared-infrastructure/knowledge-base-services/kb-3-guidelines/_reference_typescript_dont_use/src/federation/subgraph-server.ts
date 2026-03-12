// Apollo Federation Subgraph Server for KB-3 Guideline Evidence Service
// Integrates with Apollo Federation gateway using federation schema

import { ApolloServer } from '@apollo/server';
import { startStandaloneServer } from '@apollo/server/standalone';
import { buildSubgraphSchema } from '@apollo/subgraph';
import { readFileSync } from 'fs';
import { join } from 'path';
import gql from 'graphql-tag';

import { KB3ServiceContainer, loadConfigFromEnv } from '../index';
import { federationResolvers } from './federation-resolvers';
import type { KB3Context } from './federation-types';

export class KB3FederationServer {
  private apollo: ApolloServer<KB3Context>;
  private container: KB3ServiceContainer;
  private config: any;

  constructor(container: KB3ServiceContainer, config: any) {
    this.container = container;
    this.config = config;
    
    // Load federation schema
    const federationTypeDefs = gql(
      readFileSync(
        join(__dirname, '..', 'graphql', 'federation-schema.graphql'),
        'utf-8'
      )
    );

    // Build subgraph schema
    const schema = buildSubgraphSchema([
      {
        typeDefs: federationTypeDefs,
        resolvers: federationResolvers
      }
    ]);

    // Create Apollo Server as subgraph
    this.apollo = new ApolloServer<KB3Context>({
      schema,
      introspection: config.service.environment !== 'production',
      includeStacktraceInErrorResponses: config.service.environment === 'development',
      formatError: (formattedError, error) => {
        // Log errors for debugging
        console.error('GraphQL Error:', {
          message: formattedError.message,
          path: formattedError.path,
          locations: formattedError.locations,
          stack: config.service.environment === 'development' ? error.stack : undefined
        });

        // Return formatted error
        return {
          message: formattedError.message,
          locations: formattedError.locations,
          path: formattedError.path,
          extensions: {
            code: formattedError.extensions?.code,
            timestamp: new Date().toISOString(),
            service: 'kb3-guidelines'
          }
        };
      }
    });
  }

  async start(): Promise<{ url: string; server: any }> {
    console.log('Starting KB-3 Apollo Federation Subgraph...');

    try {
      // Start the subgraph server
      const { url, server } = await startStandaloneServer(this.apollo, {
        context: async ({ req }): Promise<KB3Context> => {
          // Extract authentication and request context
          const authHeader = req.headers.authorization;
          const userId = req.headers['x-user-id'] as string;
          const patientId = req.headers['x-patient-id'] as string;

          return {
            // Service dependencies
            guidelineService: this.container.getGuidelineService(),
            databaseService: this.container.getDatabaseService(),
            neo4jService: this.container.getNeo4jService(),
            cacheService: this.container.getCacheService(),
            conflictResolver: this.container.getConflictResolver(),
            safetyEngine: this.container.getSafetyEngine(),
            
            // Request context
            user: {
              id: userId,
              authorization: authHeader
            },
            
            // Patient context for clinical pathways
            patient: {
              id: patientId
            },
            
            // Audit context
            audit: {
              requestId: req.headers['x-request-id'] as string || `req_${Date.now()}`,
              timestamp: new Date(),
              source: 'federation-gateway'
            },
            
            // Performance tracking
            performance: {
              startTime: Date.now(),
              queries: []
            }
          };
        },
        
        listen: {
          port: this.config.federation?.port || 8084,
          host: this.config.federation?.host || '0.0.0.0'
        }
      });

      console.log(`🚀 KB-3 Subgraph ready at ${url}`);
      console.log(`📊 GraphQL Playground: ${url}graphql`);
      console.log(`🔗 Federation schema available for gateway composition`);

      return { url, server };

    } catch (error) {
      console.error('Failed to start KB-3 Federation subgraph:', error);
      throw error;
    }
  }

  async stop(): Promise<void> {
    console.log('Shutting down KB-3 Federation subgraph...');
    await this.apollo?.stop();
    console.log('KB-3 Federation subgraph stopped');
  }

  // Health check for federation gateway
  async healthCheck(): Promise<any> {
    try {
      const containerHealth = await this.container.healthCheck();
      
      return {
        status: containerHealth.status,
        service: 'kb3-guidelines-federation',
        version: this.config.service.version,
        federation: {
          schema_version: '2.3',
          subgraph_name: 'kb3-guidelines',
          capabilities: [
            'guideline-search',
            'clinical-pathways', 
            'conflict-resolution',
            'safety-overrides',
            'cross-kb-validation'
          ]
        },
        dependencies: containerHealth.services,
        timestamp: new Date().toISOString()
      };
      
    } catch (error) {
      return {
        status: 'unhealthy',
        service: 'kb3-guidelines-federation',
        error: error.message,
        timestamp: new Date().toISOString()
      };
    }
  }

  // Metrics endpoint for federation monitoring
  async getMetrics(): Promise<any> {
    try {
      const containerMetrics = await this.container.getMetrics();
      
      return {
        ...containerMetrics,
        federation: {
          subgraph_name: 'kb3-guidelines',
          schema_version: '2.3',
          entity_types: ['Guideline', 'ClinicalPathway', 'SafetyOverride', 'ConflictResolution'],
          federation_queries: containerMetrics.performance?.federation_queries || 0,
          reference_resolutions: containerMetrics.performance?.reference_resolutions || 0,
          average_response_time: containerMetrics.performance?.avg_response_ms || 0
        }
      };
      
    } catch (error) {
      return {
        error: error.message,
        service: 'kb3-guidelines-federation',
        timestamp: new Date().toISOString()
      };
    }
  }
}

// Factory function for creating federation server
export async function createKB3FederationServer(config?: any): Promise<KB3FederationServer> {
  const serviceConfig = config || loadConfigFromEnv();
  
  // Initialize the KB-3 service container
  const container = new KB3ServiceContainer(serviceConfig);
  await container.initialize();
  
  // Create federation server
  const federationServer = new KB3FederationServer(container, serviceConfig);
  
  return federationServer;
}

// Standalone server runner for development
export async function startKB3Federation(config?: any): Promise<{ 
  server: KB3FederationServer; 
  url: string; 
}> {
  const federationServer = await createKB3FederationServer(config);
  const { url, server } = await federationServer.start();
  
  // Graceful shutdown handling
  process.on('SIGTERM', async () => {
    console.log('Received SIGTERM, shutting down Federation server...');
    await federationServer.stop();
    process.exit(0);
  });

  process.on('SIGINT', async () => {
    console.log('Received SIGINT, shutting down Federation server...');
    await federationServer.stop();
    process.exit(0);
  });

  return { server: federationServer, url };
}

// Export main components
export { KB3FederationServer };
export * from './federation-types';
export * from './federation-resolvers';