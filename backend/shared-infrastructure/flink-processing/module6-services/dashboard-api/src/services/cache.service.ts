import Redis from 'ioredis';
import config, { logger } from '../config';

export class CacheService {
  private client: Redis;
  private readonly ttl: number;

  constructor() {
    this.client = new Redis({
      host: config.redis.host,
      port: config.redis.port,
      password: config.redis.password,
      db: config.redis.db,
      retryStrategy: (times: number) => {
        const delay = Math.min(times * 50, 2000);
        logger.warn(`Redis retry attempt ${times}, waiting ${delay}ms`);
        return delay;
      },
      maxRetriesPerRequest: 3,
    });

    this.ttl = config.redis.ttlSeconds;

    this.client.on('connect', () => {
      logger.info('Redis connected');
    });

    this.client.on('error', (err) => {
      logger.error({ err }, 'Redis error');
    });

    this.client.on('close', () => {
      logger.warn('Redis connection closed');
    });
  }

  // Generic get with automatic JSON parsing
  async get<T>(key: string): Promise<T | null> {
    try {
      const value = await this.client.get(key);
      if (!value) return null;
      return JSON.parse(value) as T;
    } catch (error) {
      logger.error({ error, key }, 'Cache get error');
      return null;
    }
  }

  // Generic set with automatic JSON stringification
  async set<T>(key: string, value: T, ttlSeconds?: number): Promise<boolean> {
    try {
      const ttl = ttlSeconds || this.ttl;
      const serialized = JSON.stringify(value);
      await this.client.setex(key, ttl, serialized);
      return true;
    } catch (error) {
      logger.error({ error, key }, 'Cache set error');
      return false;
    }
  }

  // Get multiple keys
  async mget<T>(keys: string[]): Promise<(T | null)[]> {
    try {
      if (keys.length === 0) return [];
      const values = await this.client.mget(...keys);
      return values.map((v) => (v ? JSON.parse(v) : null));
    } catch (error) {
      logger.error({ error, keys }, 'Cache mget error');
      return keys.map(() => null);
    }
  }

  // Delete key
  async del(key: string): Promise<boolean> {
    try {
      await this.client.del(key);
      return true;
    } catch (error) {
      logger.error({ error, key }, 'Cache delete error');
      return false;
    }
  }

  // Delete multiple keys
  async mdel(keys: string[]): Promise<boolean> {
    try {
      if (keys.length === 0) return true;
      await this.client.del(...keys);
      return true;
    } catch (error) {
      logger.error({ error, keys }, 'Cache mdel error');
      return false;
    }
  }

  // Get keys matching pattern
  async keys(pattern: string): Promise<string[]> {
    try {
      return await this.client.keys(pattern);
    } catch (error) {
      logger.error({ error, pattern }, 'Cache keys error');
      return [];
    }
  }

  // Add to sorted set (for time-series data)
  async zadd(key: string, score: number, member: string): Promise<boolean> {
    try {
      await this.client.zadd(key, score, member);
      return true;
    } catch (error) {
      logger.error({ error, key }, 'Cache zadd error');
      return false;
    }
  }

  // Get range from sorted set
  async zrangebyscore(
    key: string,
    min: number,
    max: number,
    withScores = false
  ): Promise<string[]> {
    try {
      if (withScores) {
        return await this.client.zrangebyscore(key, min, max, 'WITHSCORES');
      }
      return await this.client.zrangebyscore(key, min, max);
    } catch (error) {
      logger.error({ error, key }, 'Cache zrangebyscore error');
      return [];
    }
  }

  // Remove old entries from sorted set
  async zremrangebyscore(key: string, min: number, max: number): Promise<boolean> {
    try {
      await this.client.zremrangebyscore(key, min, max);
      return true;
    } catch (error) {
      logger.error({ error, key }, 'Cache zremrangebyscore error');
      return false;
    }
  }

  // Push to list
  async lpush(key: string, value: string): Promise<boolean> {
    try {
      await this.client.lpush(key, value);
      return true;
    } catch (error) {
      logger.error({ error, key }, 'Cache lpush error');
      return false;
    }
  }

  // Get list range
  async lrange(key: string, start: number, stop: number): Promise<string[]> {
    try {
      return await this.client.lrange(key, start, stop);
    } catch (error) {
      logger.error({ error, key }, 'Cache lrange error');
      return [];
    }
  }

  // Trim list
  async ltrim(key: string, start: number, stop: number): Promise<boolean> {
    try {
      await this.client.ltrim(key, start, stop);
      return true;
    } catch (error) {
      logger.error({ error, key }, 'Cache ltrim error');
      return false;
    }
  }

  // Check existence
  async exists(key: string): Promise<boolean> {
    try {
      const result = await this.client.exists(key);
      return result === 1;
    } catch (error) {
      logger.error({ error, key }, 'Cache exists error');
      return false;
    }
  }

  // Set expiration
  async expire(key: string, seconds: number): Promise<boolean> {
    try {
      await this.client.expire(key, seconds);
      return true;
    } catch (error) {
      logger.error({ error, key }, 'Cache expire error');
      return false;
    }
  }

  // Get TTL
  async ttlRemaining(key: string): Promise<number> {
    try {
      return await this.client.ttl(key);
    } catch (error) {
      logger.error({ error, key }, 'Cache ttl error');
      return -1;
    }
  }

  // Flush database (use carefully)
  async flushdb(): Promise<boolean> {
    try {
      await this.client.flushdb();
      logger.warn('Redis database flushed');
      return true;
    } catch (error) {
      logger.error({ error }, 'Cache flushdb error');
      return false;
    }
  }

  // Health check
  async ping(): Promise<boolean> {
    try {
      const result = await this.client.ping();
      return result === 'PONG';
    } catch (error) {
      logger.error({ error }, 'Cache ping error');
      return false;
    }
  }

  // Close connection
  async close(): Promise<void> {
    await this.client.quit();
    logger.info('Redis connection closed');
  }
}

export default new CacheService();
