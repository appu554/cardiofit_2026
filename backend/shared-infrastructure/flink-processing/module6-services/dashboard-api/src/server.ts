import express, { Request, Response } from 'express';
import { ApolloServer } from '@apollo/server';
import { expressMiddleware } from '@apollo/server/express4';
import { ApolloServerPluginDrainHttpServer } from '@apollo/server/plugin/drainHttpServer';
import { ApolloServerPluginLandingPageLocalDefault } from '@apollo/server/plugin/landingPage/default';
import http from 'http';
import cors from 'cors';
import bodyParser from 'body-parser';
import { readFileSync } from 'fs';
import { join } from 'path';
import config, { logger } from './config';
import { resolvers } from './resolvers';
import kafkaConsumerService from './services/kafka-consumer.service';
import analyticsDataService from './services/analytics-data.service';
import cacheService from './services/cache.service';

// Read GraphQL schema
const typeDefs = readFileSync(join(__dirname, 'schema', 'types.graphql'), 'utf-8');

async function startServer() {
  const app = express();
  const httpServer = http.createServer(app);

  // Apollo Server configuration
  const server = new ApolloServer({
    typeDefs,
    resolvers,
    plugins: [
      ApolloServerPluginDrainHttpServer({ httpServer }),
      ...(config.graphql.playground
        ? [ApolloServerPluginLandingPageLocalDefault({ embed: true })]
        : []),
    ],
    introspection: config.graphql.introspection,
    formatError: (formattedError, error) => {
      logger.error({ error: formattedError }, 'GraphQL error');
      return formattedError;
    },
  });

  await server.start();
  logger.info('Apollo Server started');

  // Middleware
  app.use(
    cors({
      origin: config.security.corsOrigin,
      credentials: true,
    })
  );

  app.use(bodyParser.json());
  app.use(bodyParser.urlencoded({ extended: true }));

  // GraphQL endpoint
  app.use(
    '/graphql',
    expressMiddleware(server, {
      context: async ({ req }) => ({
        req,
        logger,
      }),
    })
  );

  // Health check endpoint
  app.get('/health', async (req: Request, res: Response) => {
    try {
      const [kafkaHealthy, dataServicesHealth, redisHealthy] = await Promise.all([
        kafkaConsumerService.healthCheck(),
        analyticsDataService.healthCheck(),
        cacheService.ping(),
      ]);

      const healthy =
        kafkaHealthy && dataServicesHealth.postgres && dataServicesHealth.influxdb && redisHealthy;

      const status = {
        status: healthy ? 'healthy' : 'unhealthy',
        timestamp: new Date().toISOString(),
        services: {
          kafka: kafkaHealthy ? 'up' : 'down',
          postgres: dataServicesHealth.postgres ? 'up' : 'down',
          influxdb: dataServicesHealth.influxdb ? 'up' : 'down',
          redis: redisHealthy ? 'up' : 'down',
        },
        uptime: process.uptime(),
        memory: process.memoryUsage(),
      };

      res.status(healthy ? 200 : 503).json(status);
    } catch (error) {
      logger.error({ error }, 'Health check failed');
      res.status(503).json({
        status: 'unhealthy',
        error: 'Health check failed',
        timestamp: new Date().toISOString(),
      });
    }
  });

  // Readiness probe
  app.get('/ready', async (req: Request, res: Response) => {
    try {
      const kafkaReady = await kafkaConsumerService.healthCheck();
      if (kafkaReady) {
        res.status(200).json({ ready: true, timestamp: new Date().toISOString() });
      } else {
        res.status(503).json({ ready: false, timestamp: new Date().toISOString() });
      }
    } catch (error) {
      res.status(503).json({ ready: false, timestamp: new Date().toISOString() });
    }
  });

  // Liveness probe
  app.get('/live', (req: Request, res: Response) => {
    res.status(200).json({ alive: true, timestamp: new Date().toISOString() });
  });

  // Metrics endpoint (basic)
  app.get('/metrics', async (req: Request, res: Response) => {
    try {
      const metrics = {
        timestamp: new Date().toISOString(),
        uptime: process.uptime(),
        memory: process.memoryUsage(),
        cpu: process.cpuUsage(),
      };
      res.json(metrics);
    } catch (error) {
      res.status(500).json({ error: 'Failed to get metrics' });
    }
  });

  // Root endpoint
  app.get('/', (req: Request, res: Response) => {
    res.json({
      service: 'Dashboard API',
      version: '1.0.0',
      graphql: '/graphql',
      health: '/health',
      timestamp: new Date().toISOString(),
    });
  });

  // Start Kafka consumers
  try {
    await kafkaConsumerService.start();
    logger.info('Kafka consumers started successfully');
  } catch (error) {
    logger.error({ error }, 'Failed to start Kafka consumers');
    process.exit(1);
  }

  // Start HTTP server
  await new Promise<void>((resolve) => {
    httpServer.listen(config.server.port, config.server.host, () => {
      logger.info(
        {
          port: config.server.port,
          host: config.server.host,
          env: config.server.env,
        },
        `Dashboard API server running at http://${config.server.host}:${config.server.port}`
      );
      logger.info(`GraphQL endpoint: http://${config.server.host}:${config.server.port}/graphql`);
      resolve();
    });
  });

  // Graceful shutdown
  const shutdown = async (signal: string) => {
    logger.info({ signal }, 'Shutdown signal received');

    try {
      // Stop accepting new requests
      await new Promise<void>((resolve, reject) => {
        httpServer.close((err) => {
          if (err) reject(err);
          else resolve();
        });
      });

      // Stop Kafka consumers
      await kafkaConsumerService.stop();

      // Close database connections
      await analyticsDataService.close();

      // Close Redis connection
      await cacheService.close();

      logger.info('Graceful shutdown completed');
      process.exit(0);
    } catch (error) {
      logger.error({ error }, 'Error during shutdown');
      process.exit(1);
    }
  };

  // Handle shutdown signals
  process.on('SIGTERM', () => shutdown('SIGTERM'));
  process.on('SIGINT', () => shutdown('SIGINT'));

  // Handle uncaught errors
  process.on('uncaughtException', (error) => {
    logger.error({ error }, 'Uncaught exception');
    shutdown('uncaughtException');
  });

  process.on('unhandledRejection', (reason, promise) => {
    logger.error({ reason, promise }, 'Unhandled rejection');
    shutdown('unhandledRejection');
  });
}

// Start the server
startServer().catch((error) => {
  logger.error({ error }, 'Failed to start server');
  process.exit(1);
});
