import dotenv from 'dotenv';
import { pino } from 'pino';

dotenv.config();

export interface Config {
  server: {
    port: number;
    host: string;
    env: string;
  };
  kafka: {
    brokers: string[];
    clientId: string;
    groupId: string;
    topics: {
      hospitalKpis: string;
      departmentMetrics: string;
      patientRiskProfiles: string;
      patientEvents: string;
      sepsisSurveillance: string;
      qualityMetrics: string;
    };
  };
  redis: {
    host: string;
    port: number;
    password?: string;
    db: number;
    ttlSeconds: number;
  };
  postgres: {
    host: string;
    port: number;
    database: string;
    user: string;
    password: string;
    maxConnections: number;
  };
  influxdb: {
    url: string;
    token: string;
    org: string;
    bucket: string;
    timeout: number;
  };
  graphql: {
    playground: boolean;
    introspection: boolean;
    debug: boolean;
  };
  security: {
    corsOrigin: string;
    rateLimitWindowMs: number;
    rateLimitMaxRequests: number;
  };
  logging: {
    level: string;
    prettyPrint: boolean;
  };
  monitoring: {
    enableMetrics: boolean;
    healthCheckTimeout: number;
  };
}

const config: Config = {
  server: {
    port: parseInt(process.env.PORT || '4000', 10),
    host: process.env.HOST || '0.0.0.0',
    env: process.env.NODE_ENV || 'development',
  },
  kafka: {
    brokers: (process.env.KAFKA_BROKERS || 'localhost:9092').split(','),
    clientId: process.env.KAFKA_CLIENT_ID || 'dashboard-api',
    groupId: process.env.KAFKA_GROUP_ID || 'dashboard-api-consumers',
    topics: {
      hospitalKpis: 'analytics-patient-census',
      departmentMetrics: 'analytics-department-workload',
      patientRiskProfiles: 'analytics-patient-census', // Keep for now, will be replaced by patientEvents
      patientEvents: 'patient-events-v1', // Raw patient events for individual patient tracking
      sepsisSurveillance: 'analytics-sepsis-surveillance',
      qualityMetrics: 'analytics-ml-performance',
    },
  },
  redis: {
    host: process.env.REDIS_HOST || 'localhost',
    port: parseInt(process.env.REDIS_PORT || '6379', 10),
    password: process.env.REDIS_PASSWORD,
    db: parseInt(process.env.REDIS_DB || '0', 10),
    ttlSeconds: parseInt(process.env.REDIS_TTL_SECONDS || '300', 10),
  },
  postgres: {
    host: process.env.POSTGRES_HOST || 'localhost',
    port: parseInt(process.env.POSTGRES_PORT || '5432', 10),
    database: process.env.POSTGRES_DB || 'clinical_analytics',
    user: process.env.POSTGRES_USER || 'postgres',
    password: process.env.POSTGRES_PASSWORD || 'postgres',
    maxConnections: parseInt(process.env.POSTGRES_MAX_CONNECTIONS || '20', 10),
  },
  influxdb: {
    url: process.env.INFLUX_URL || 'http://localhost:8086',
    token: process.env.INFLUX_TOKEN || '',
    org: process.env.INFLUX_ORG || 'cardiofit',
    bucket: process.env.INFLUX_BUCKET || 'clinical_metrics',
    timeout: parseInt(process.env.INFLUX_TIMEOUT || '30000', 10),
  },
  graphql: {
    playground: process.env.GRAPHQL_PLAYGROUND === 'true',
    introspection: process.env.GRAPHQL_INTROSPECTION === 'true',
    debug: process.env.GRAPHQL_DEBUG === 'true',
  },
  security: {
    corsOrigin: process.env.CORS_ORIGIN || 'http://localhost:4200',
    rateLimitWindowMs: parseInt(process.env.RATE_LIMIT_WINDOW_MS || '900000', 10),
    rateLimitMaxRequests: parseInt(process.env.RATE_LIMIT_MAX_REQUESTS || '100', 10),
  },
  logging: {
    level: process.env.LOG_LEVEL || 'info',
    prettyPrint: process.env.LOG_PRETTY_PRINT === 'true',
  },
  monitoring: {
    enableMetrics: process.env.ENABLE_METRICS === 'true',
    healthCheckTimeout: parseInt(process.env.HEALTH_CHECK_TIMEOUT || '5000', 10),
  },
};

// Create logger instance
export const logger = pino({
  level: config.logging.level,
  transport: config.logging.prettyPrint
    ? {
        target: 'pino-pretty',
        options: {
          colorize: true,
          translateTime: 'SYS:standard',
          ignore: 'pid,hostname',
        },
      }
    : undefined,
});

export default config;
