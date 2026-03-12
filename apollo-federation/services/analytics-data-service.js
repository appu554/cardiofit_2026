/**
 * Analytics Data Service for Module 6
 *
 * Handles data access for the Analytics Dashboard API:
 * - Redis: Real-time metrics and caching
 * - PostgreSQL: Historical analytics data (requires 'pg' package)
 * - Kafka: Streaming analytics data (requires 'kafkajs' package)
 *
 * Installation required:
 * npm install pg kafkajs
 */

const Redis = require('ioredis');
require('dotenv').config();

// Logger utility
const logger = {
  info: (message, ...args) => console.log(`[Analytics] ${new Date().toISOString()} - ${message}`, ...args),
  error: (message, ...args) => console.error(`[Analytics ERROR] ${new Date().toISOString()} - ${message}`, ...args),
  warn: (message, ...args) => console.warn(`[Analytics WARN] ${new Date().toISOString()} - ${message}`, ...args),
  debug: (message, ...args) => console.debug(`[Analytics DEBUG] ${new Date().toISOString()} - ${message}`, ...args)
};

class AnalyticsDataService {
  constructor() {
    this.redis = null;
    this.pg = null;
    this.kafka = null;
    this.connected = {
      redis: false,
      postgres: false,
      kafka: false
    };
  }

  /**
   * Initialize all data connections
   */
  async initialize() {
    logger.info('Initializing Analytics Data Service...');

    await this.initRedis();
    await this.initPostgres();
    await this.initKafka();

    logger.info('Analytics Data Service initialization complete', {
      redis: this.connected.redis,
      postgres: this.connected.postgres,
      kafka: this.connected.kafka
    });
  }

  /**
   * Initialize Redis connection
   */
  async initRedis() {
    try {
      this.redis = new Redis({
        host: process.env.REDIS_HOST || 'localhost',
        port: parseInt(process.env.REDIS_PORT || '6379'),
        db: parseInt(process.env.REDIS_DB || '0'),
        retryStrategy: (times) => {
          const delay = Math.min(times * 50, 2000);
          return delay;
        },
        maxRetriesPerRequest: 3
      });

      this.redis.on('connect', () => {
        logger.info('Redis connected successfully');
        this.connected.redis = true;
      });

      this.redis.on('error', (err) => {
        logger.error('Redis error:', err.message);
        this.connected.redis = false;
      });

      // Test connection
      await this.redis.ping();
      logger.info('Redis connection verified');

    } catch (error) {
      logger.error('Failed to initialize Redis:', error.message);
      this.connected.redis = false;
    }
  }

  /**
   * Initialize PostgreSQL connection
   * Note: Requires 'pg' package - run: npm install pg
   */
  async initPostgres() {
    try {
      // Check if pg is installed
      let Pool;
      try {
        const pg = require('pg');
        Pool = pg.Pool;
      } catch (requireError) {
        logger.warn('PostgreSQL driver not installed. Run: npm install pg');
        this.connected.postgres = false;
        return;
      }

      this.pg = new Pool({
        host: process.env.POSTGRES_HOST || 'localhost',
        port: parseInt(process.env.POSTGRES_PORT || '5432'),
        database: process.env.POSTGRES_DB || 'analytics_db',
        user: process.env.POSTGRES_USER || 'postgres',
        password: process.env.POSTGRES_PASSWORD || 'postgres',
        max: 20, // Maximum pool size
        idleTimeoutMillis: 30000,
        connectionTimeoutMillis: 2000,
      });

      // Test connection
      const client = await this.pg.connect();
      await client.query('SELECT NOW()');
      client.release();

      this.connected.postgres = true;
      logger.info('PostgreSQL connected successfully');

    } catch (error) {
      logger.error('Failed to initialize PostgreSQL:', error.message);
      this.connected.postgres = false;
    }
  }

  /**
   * Initialize Kafka consumer
   * Note: Requires 'kafkajs' package - run: npm install kafkajs
   */
  async initKafka() {
    try {
      // Check if kafkajs is installed
      let Kafka;
      try {
        const kafkajs = require('kafkajs');
        Kafka = kafkajs.Kafka;
      } catch (requireError) {
        logger.warn('Kafka driver not installed. Run: npm install kafkajs');
        this.connected.kafka = false;
        return;
      }

      this.kafka = new Kafka({
        clientId: 'analytics-dashboard-api',
        brokers: (process.env.KAFKA_BROKERS || 'localhost:9092').split(','),
      });

      // Create admin client to test connection
      const admin = this.kafka.admin();
      await admin.connect();
      await admin.listTopics();
      await admin.disconnect();

      this.connected.kafka = true;
      logger.info('Kafka connected successfully');

    } catch (error) {
      logger.error('Failed to initialize Kafka:', error.message);
      this.connected.kafka = false;
    }
  }

  // ==================== Redis Operations ====================

  /**
   * Get hospital-wide KPIs from Redis cache
   */
  async getHospitalKPIs() {
    try {
      if (!this.connected.redis) {
        throw new Error('Redis not connected');
      }

      const key = 'analytics:hospital:kpis';
      const data = await this.redis.get(key);

      if (data) {
        return JSON.parse(data);
      }

      // Return default structure if no data
      return {
        timestamp: new Date().toISOString(),
        totalPatients: 0,
        highRiskPatients: 0,
        criticalPatients: 0,
        avgMortalityRisk: 0.0,
        avgSepsisRisk: 0.0,
        avgReadmissionRisk: 0.0,
        activeAlerts: 0,
        criticalAlerts: 0,
        modelAccuracy: null,
        patientsOnProtocols: 0
      };

    } catch (error) {
      logger.error('Error getting hospital KPIs:', error.message);
      throw error;
    }
  }

  /**
   * Get department metrics from Redis
   */
  async getDepartmentMetrics(department) {
    try {
      if (!this.connected.redis) {
        throw new Error('Redis not connected');
      }

      const key = `analytics:department:${department}`;
      const data = await this.redis.get(key);

      if (data) {
        return JSON.parse(data);
      }

      return null;

    } catch (error) {
      logger.error(`Error getting department metrics for ${department}:`, error.message);
      throw error;
    }
  }

  /**
   * Get all department metrics from Redis
   */
  async getAllDepartmentMetrics() {
    try {
      if (!this.connected.redis) {
        throw new Error('Redis not connected');
      }

      const pattern = 'analytics:department:*';
      const keys = await this.redis.keys(pattern);

      const departments = [];
      for (const key of keys) {
        const data = await this.redis.get(key);
        if (data) {
          departments.push(JSON.parse(data));
        }
      }

      return departments;

    } catch (error) {
      logger.error('Error getting all department metrics:', error.message);
      throw error;
    }
  }

  /**
   * Get patient risk profile from Redis
   */
  async getPatientRiskProfile(patientId) {
    try {
      if (!this.connected.redis) {
        throw new Error('Redis not connected');
      }

      const key = `analytics:patient:${patientId}:risk`;
      const data = await this.redis.get(key);

      if (data) {
        return JSON.parse(data);
      }

      return null;

    } catch (error) {
      logger.error(`Error getting patient risk profile for ${patientId}:`, error.message);
      throw error;
    }
  }

  /**
   * Get high-risk patients from Redis
   */
  async getHighRiskPatients(limit = 50) {
    try {
      if (!this.connected.redis) {
        throw new Error('Redis not connected');
      }

      const pattern = 'analytics:patient:*:risk';
      const keys = await this.redis.keys(pattern);

      const patients = [];
      for (const key of keys) {
        const data = await this.redis.get(key);
        if (data) {
          const profile = JSON.parse(data);
          if (profile.isHighRisk) {
            patients.push(profile);
          }
        }

        if (patients.length >= limit) {
          break;
        }
      }

      // Sort by risk score descending
      patients.sort((a, b) => b.overallRiskScore - a.overallRiskScore);

      return patients.slice(0, limit);

    } catch (error) {
      logger.error('Error getting high-risk patients:', error.message);
      throw error;
    }
  }

  /**
   * Cache population health metrics in Redis
   * Called by Kafka consumer to update real-time data
   */
  async cachePopulationMetrics(metrics) {
    try {
      if (!this.connected.redis) {
        logger.warn('Redis not connected, skipping cache');
        return;
      }

      const key = `analytics:department:${metrics.department}`;
      await this.redis.setex(key, 300, JSON.stringify(metrics)); // 5 min TTL

      logger.debug(`Cached population metrics for department: ${metrics.department}`);

    } catch (error) {
      logger.error('Error caching population metrics:', error.message);
    }
  }

  /**
   * Cache patient risk profile in Redis
   */
  async cachePatientRiskProfile(profile) {
    try {
      if (!this.connected.redis) {
        logger.warn('Redis not connected, skipping cache');
        return;
      }

      const key = `analytics:patient:${profile.patientId}:risk`;
      await this.redis.setex(key, 300, JSON.stringify(profile)); // 5 min TTL

      logger.debug(`Cached patient risk profile: ${profile.patientId}`);

    } catch (error) {
      logger.error('Error caching patient risk profile:', error.message);
    }
  }

  // ==================== PostgreSQL Operations ====================

  /**
   * Query historical analytics data from PostgreSQL
   */
  async queryHistoricalData(query, params = []) {
    try {
      if (!this.connected.postgres) {
        throw new Error('PostgreSQL not connected');
      }

      const result = await this.pg.query(query, params);
      return result.rows;

    } catch (error) {
      logger.error('Error querying historical data:', error.message);
      throw error;
    }
  }

  /**
   * Get alert metrics from PostgreSQL
   */
  async getAlertMetrics(startTime, endTime) {
    try {
      if (!this.connected.postgres) {
        logger.warn('PostgreSQL not available, returning mock data');
        return {
          timestamp: new Date().toISOString(),
          totalAlerts: 0,
          criticalAlerts: 0,
          warningAlerts: 0,
          infoAlerts: 0,
          avgResolutionTime: null,
          alertsByType: [],
          alertsByDepartment: []
        };
      }

      // Query alert metrics from PostgreSQL analytics tables
      const query = `
        SELECT
          COUNT(*) as total_alerts,
          SUM(CASE WHEN severity = 'CRITICAL' THEN 1 ELSE 0 END) as critical_alerts,
          SUM(CASE WHEN severity = 'WARNING' THEN 1 ELSE 0 END) as warning_alerts,
          SUM(CASE WHEN severity = 'INFO' THEN 1 ELSE 0 END) as info_alerts,
          AVG(EXTRACT(EPOCH FROM (resolved_at - timestamp))/60) as avg_resolution_time
        FROM alerts
        WHERE timestamp >= $1 AND timestamp <= $2
      `;

      const result = await this.pg.query(query, [startTime, endTime]);
      const row = result.rows[0];

      return {
        timestamp: new Date().toISOString(),
        totalAlerts: parseInt(row.total_alerts) || 0,
        criticalAlerts: parseInt(row.critical_alerts) || 0,
        warningAlerts: parseInt(row.warning_alerts) || 0,
        infoAlerts: parseInt(row.info_alerts) || 0,
        avgResolutionTime: row.avg_resolution_time ? parseFloat(row.avg_resolution_time) : null,
        alertsByType: [],
        alertsByDepartment: []
      };

    } catch (error) {
      logger.error('Error getting alert metrics:', error.message);
      throw error;
    }
  }

  // ==================== Kafka Operations ====================

  /**
   * Subscribe to Kafka topic for real-time analytics
   */
  async subscribeToTopic(topic, callback) {
    try {
      if (!this.connected.kafka) {
        logger.warn('Kafka not available for subscription');
        return null;
      }

      const consumer = this.kafka.consumer({
        groupId: process.env.KAFKA_GROUP_ID || 'analytics-dashboard-api'
      });

      await consumer.connect();
      await consumer.subscribe({ topic, fromBeginning: false });

      await consumer.run({
        eachMessage: async ({ topic, partition, message }) => {
          try {
            const data = JSON.parse(message.value.toString());
            callback(data);
          } catch (error) {
            logger.error(`Error processing Kafka message from ${topic}:`, error.message);
          }
        },
      });

      logger.info(`Subscribed to Kafka topic: ${topic}`);
      return consumer;

    } catch (error) {
      logger.error(`Error subscribing to Kafka topic ${topic}:`, error.message);
      return null;
    }
  }

  // ==================== Health Check ====================

  /**
   * Check health of all connections
   */
  async healthCheck() {
    const status = {
      status: 'ok',
      timestamp: new Date().toISOString(),
      redisConnected: this.connected.redis,
      postgresConnected: this.connected.postgres,
      kafkaConnected: this.connected.kafka
    };

    // Test Redis connection
    if (this.connected.redis) {
      try {
        await this.redis.ping();
      } catch (error) {
        status.redisConnected = false;
        this.connected.redis = false;
      }
    }

    // Test PostgreSQL connection
    if (this.connected.postgres) {
      try {
        const client = await this.pg.connect();
        await client.query('SELECT 1');
        client.release();
      } catch (error) {
        status.postgresConnected = false;
        this.connected.postgres = false;
      }
    }

    if (!status.redisConnected && !status.postgresConnected && !status.kafkaConnected) {
      status.status = 'degraded';
    }

    return status;
  }

  /**
   * Graceful shutdown
   */
  async shutdown() {
    logger.info('Shutting down Analytics Data Service...');

    if (this.redis) {
      await this.redis.quit();
    }

    if (this.pg) {
      await this.pg.end();
    }

    // Kafka consumers are managed separately

    logger.info('Analytics Data Service shutdown complete');
  }
}

// Export singleton instance
const analyticsDataService = new AnalyticsDataService();

module.exports = analyticsDataService;
