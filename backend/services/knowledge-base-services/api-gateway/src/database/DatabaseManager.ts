import { Pool, PoolClient, QueryResult } from 'pg';
import { createLogger, Logger } from 'winston';
import dotenv from 'dotenv';

dotenv.config();

export interface DatabaseConfig {
  host: string;
  port: number;
  database: string;
  user: string;
  password: string;
  ssl?: boolean;
  max?: number;
  idleTimeoutMillis?: number;
  connectionTimeoutMillis?: number;
}

export class DatabaseManager {
  private pool: Pool | null = null;
  private logger: Logger;
  private config: DatabaseConfig;
  private connectionAttempts: number = 0;
  private maxConnectionAttempts: number = 5;
  private reconnectDelay: number = 5000;

  constructor(config?: Partial<DatabaseConfig>) {
    this.logger = createLogger({
      defaultMeta: { service: 'database-manager' }
    });

    // Default configuration with environment variable overrides
    this.config = {
      host: process.env.DB_HOST || 'localhost',
      port: parseInt(process.env.DB_PORT || '5433'),
      database: process.env.DB_NAME || 'clinical_governance',
      user: process.env.DB_USER || 'postgres',
      password: process.env.DB_PASSWORD || 'kb_postgres_password',
      ssl: process.env.DB_SSL === 'true',
      max: parseInt(process.env.DB_POOL_SIZE || '20'),
      idleTimeoutMillis: parseInt(process.env.DB_IDLE_TIMEOUT || '30000'),
      connectionTimeoutMillis: parseInt(process.env.DB_CONNECTION_TIMEOUT || '5000'),
      ...config
    };
  }

  async connect(): Promise<void> {
    if (this.pool) {
      this.logger.warn('Database connection already exists');
      return;
    }

    try {
      this.logger.info('Connecting to database...', {
        host: this.config.host,
        port: this.config.port,
        database: this.config.database,
        user: this.config.user
      });

      this.pool = new Pool({
        host: this.config.host,
        port: this.config.port,
        database: this.config.database,
        user: this.config.user,
        password: this.config.password,
        ssl: this.config.ssl ? { rejectUnauthorized: false } : false,
        max: this.config.max,
        idleTimeoutMillis: this.config.idleTimeoutMillis,
        connectionTimeoutMillis: this.config.connectionTimeoutMillis,
        // Additional pool configuration
        min: 2,
        acquireTimeoutMillis: 60000,
        createTimeoutMillis: 30000,
        destroyTimeoutMillis: 5000,
        propagateCreateError: false
      });

      // Set up event handlers
      this.pool.on('connect', (client) => {
        this.logger.debug('New database client connected', {
          totalCount: this.pool?.totalCount,
          idleCount: this.pool?.idleCount,
          waitingCount: this.pool?.waitingCount
        });
      });

      this.pool.on('error', (err) => {
        this.logger.error('Database pool error', { error: err.message });
        this.handleConnectionError(err);
      });

      this.pool.on('remove', (client) => {
        this.logger.debug('Database client removed from pool');
      });

      // Test the connection
      await this.testConnection();

      this.connectionAttempts = 0;
      this.logger.info('Database connected successfully', {
        totalCount: this.pool.totalCount,
        idleCount: this.pool.idleCount
      });

    } catch (error) {
      this.logger.error('Failed to connect to database', {
        error: error.message,
        attempt: this.connectionAttempts + 1,
        maxAttempts: this.maxConnectionAttempts
      });

      this.connectionAttempts++;
      
      if (this.connectionAttempts < this.maxConnectionAttempts) {
        this.logger.info(`Retrying connection in ${this.reconnectDelay}ms...`);
        await this.delay(this.reconnectDelay);
        return this.connect();
      }
      
      throw new Error(`Failed to connect to database after ${this.maxConnectionAttempts} attempts: ${error.message}`);
    }
  }

  async disconnect(): Promise<void> {
    if (!this.pool) {
      this.logger.warn('No database connection to close');
      return;
    }

    try {
      this.logger.info('Closing database connection...');
      await this.pool.end();
      this.pool = null;
      this.logger.info('Database connection closed successfully');
    } catch (error) {
      this.logger.error('Error closing database connection', { error: error.message });
      throw error;
    }
  }

  async query(text: string, params?: any[]): Promise<QueryResult> {
    if (!this.pool) {
      throw new Error('Database not connected. Call connect() first.');
    }

    const start = Date.now();
    let client: PoolClient | null = null;

    try {
      client = await this.pool.connect();
      
      this.logger.debug('Executing query', {
        query: this.sanitizeQueryForLogging(text),
        paramCount: params ? params.length : 0
      });

      const result = await client.query(text, params);
      
      const duration = Date.now() - start;
      this.logger.debug('Query completed', {
        duration: `${duration}ms`,
        rowCount: result.rowCount
      });

      return result;

    } catch (error) {
      const duration = Date.now() - start;
      this.logger.error('Query failed', {
        error: error.message,
        duration: `${duration}ms`,
        query: this.sanitizeQueryForLogging(text)
      });
      throw error;
    } finally {
      if (client) {
        client.release();
      }
    }
  }

  async transaction<T>(callback: (client: PoolClient) => Promise<T>): Promise<T> {
    if (!this.pool) {
      throw new Error('Database not connected. Call connect() first.');
    }

    const client = await this.pool.connect();
    
    try {
      await client.query('BEGIN');
      this.logger.debug('Transaction started');

      const result = await callback(client);
      
      await client.query('COMMIT');
      this.logger.debug('Transaction committed');
      
      return result;

    } catch (error) {
      await client.query('ROLLBACK');
      this.logger.error('Transaction rolled back', { error: error.message });
      throw error;
    } finally {
      client.release();
    }
  }

  async queryWithTransaction(queries: Array<{ text: string; params?: any[] }>): Promise<QueryResult[]> {
    return this.transaction(async (client) => {
      const results: QueryResult[] = [];
      
      for (const query of queries) {
        const result = await client.query(query.text, query.params);
        results.push(result);
      }
      
      return results;
    });
  }

  async testConnection(): Promise<boolean> {
    try {
      const result = await this.query('SELECT NOW() as current_time, version() as pg_version');
      
      if (result.rows.length > 0) {
        this.logger.info('Database connection test successful', {
          currentTime: result.rows[0].current_time,
          pgVersion: result.rows[0].pg_version.split(' ')[0] // Just PostgreSQL version
        });
        return true;
      }
      
      return false;
    } catch (error) {
      this.logger.error('Database connection test failed', { error: error.message });
      throw error;
    }
  }

  async checkHealth(): Promise<{
    connected: boolean;
    poolStats?: any;
    latency?: number;
    error?: string;
  }> {
    if (!this.pool) {
      return {
        connected: false,
        error: 'Database pool not initialized'
      };
    }

    try {
      const start = Date.now();
      await this.query('SELECT 1');
      const latency = Date.now() - start;

      return {
        connected: true,
        poolStats: {
          totalCount: this.pool.totalCount,
          idleCount: this.pool.idleCount,
          waitingCount: this.pool.waitingCount
        },
        latency
      };

    } catch (error) {
      return {
        connected: false,
        error: error.message
      };
    }
  }

  async initializeSchema(): Promise<void> {
    if (!this.pool) {
      throw new Error('Database not connected');
    }

    try {
      this.logger.info('Initializing database schema...');

      // Check if tables exist
      const tablesExist = await this.checkTablesExist([
        'kb_version_sets',
        'evidence_envelopes',
        'kb_response_log'
      ]);

      if (!tablesExist.allExist) {
        this.logger.info('Creating missing database tables...', {
          missingTables: tablesExist.missing
        });
        
        await this.createTables();
        
        this.logger.info('Database schema initialized successfully');
      } else {
        this.logger.info('Database schema already exists');
      }

    } catch (error) {
      this.logger.error('Failed to initialize database schema', { error: error.message });
      throw error;
    }
  }

  private async checkTablesExist(tableNames: string[]): Promise<{
    allExist: boolean;
    existing: string[];
    missing: string[];
  }> {
    const query = `
      SELECT table_name 
      FROM information_schema.tables 
      WHERE table_schema = 'public' 
      AND table_name = ANY($1)
    `;

    const result = await this.query(query, [tableNames]);
    const existing = result.rows.map(row => row.table_name);
    const missing = tableNames.filter(name => !existing.includes(name));

    return {
      allExist: missing.length === 0,
      existing,
      missing
    };
  }

  private async createTables(): Promise<void> {
    const createTablesSQL = `
      -- KB Version Sets table
      CREATE TABLE IF NOT EXISTS kb_version_sets (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        version_set_name VARCHAR(100) UNIQUE NOT NULL,
        description TEXT,
        
        -- Version mapping for all KBs
        kb_versions JSONB NOT NULL DEFAULT '{}',
        
        -- Validation status
        validated BOOLEAN DEFAULT FALSE,
        validation_results JSONB,
        validation_timestamp TIMESTAMPTZ,
        
        -- Deployment tracking
        environment VARCHAR(50) NOT NULL CHECK (environment IN ('dev', 'staging', 'production')),
        active BOOLEAN DEFAULT FALSE,
        activated_at TIMESTAMPTZ,
        deactivated_at TIMESTAMPTZ,
        
        -- Governance
        created_by VARCHAR(100) NOT NULL,
        approved_by VARCHAR(100),
        approval_timestamp TIMESTAMPTZ,
        approval_notes TEXT,
        
        -- Audit
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        
        -- Constraints
        CONSTRAINT unique_active_per_env EXCLUDE (environment WITH =) WHERE (active = true)
      );

      -- Evidence Envelopes table
      CREATE TABLE IF NOT EXISTS evidence_envelopes (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        transaction_id VARCHAR(100) UNIQUE NOT NULL,
        
        -- Version snapshot at time of transaction
        version_set_id UUID REFERENCES kb_version_sets(id),
        kb_versions JSONB NOT NULL,
        
        -- Decision tracking
        decision_chain JSONB NOT NULL DEFAULT '[]',
        safety_attestations JSONB NOT NULL DEFAULT '[]',
        performance_metrics JSONB,
        
        -- Clinical context
        patient_id VARCHAR(100),
        encounter_id VARCHAR(100),
        clinical_domain VARCHAR(50),
        request_type VARCHAR(50),
        
        -- Orchestration metadata
        orchestrator_version VARCHAR(50),
        orchestrator_node VARCHAR(100),
        
        -- Timing
        started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        completed_at TIMESTAMPTZ,
        total_duration_ms INTEGER,
        
        -- Immutability
        checksum VARCHAR(64) NOT NULL,
        signed BOOLEAN DEFAULT FALSE,
        signature TEXT,
        
        -- Partitioning key
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
      );

      -- KB Response Log table
      CREATE TABLE IF NOT EXISTS kb_response_log (
        id BIGSERIAL PRIMARY KEY,
        envelope_id UUID REFERENCES evidence_envelopes(id),
        kb_name VARCHAR(50) NOT NULL,
        kb_version VARCHAR(50) NOT NULL,
        latency_ms INTEGER NOT NULL,
        cache_hit BOOLEAN DEFAULT FALSE,
        response_size INTEGER,
        error_count INTEGER DEFAULT 0,
        timestamp TIMESTAMPTZ DEFAULT NOW()
      );

      -- Indexes for performance
      CREATE INDEX IF NOT EXISTS idx_kb_version_sets_environment ON kb_version_sets(environment);
      CREATE INDEX IF NOT EXISTS idx_kb_version_sets_active ON kb_version_sets(active);
      CREATE INDEX IF NOT EXISTS idx_evidence_envelopes_transaction_id ON evidence_envelopes(transaction_id);
      CREATE INDEX IF NOT EXISTS idx_evidence_envelopes_patient_id ON evidence_envelopes(patient_id);
      CREATE INDEX IF NOT EXISTS idx_evidence_envelopes_created_at ON evidence_envelopes(created_at);
      CREATE INDEX IF NOT EXISTS idx_kb_response_log_envelope ON kb_response_log(envelope_id);
      CREATE INDEX IF NOT EXISTS idx_kb_response_log_kb_name ON kb_response_log(kb_name, timestamp DESC);
    `;

    await this.query(createTablesSQL);
  }

  private handleConnectionError(error: Error): void {
    this.logger.error('Database connection error occurred', { error: error.message });
    
    // Attempt to reconnect after delay
    setTimeout(async () => {
      if (this.pool) {
        this.logger.info('Attempting to reconnect to database...');
        try {
          await this.disconnect();
          await this.connect();
        } catch (reconnectError) {
          this.logger.error('Failed to reconnect to database', { 
            error: reconnectError.message 
          });
        }
      }
    }, this.reconnectDelay);
  }

  private sanitizeQueryForLogging(query: string): string {
    // Remove sensitive data from query for logging
    return query
      .replace(/password\s*=\s*'[^']*'/gi, "password = '***'")
      .replace(/token\s*=\s*'[^']*'/gi, "token = '***'")
      .substring(0, 200) + (query.length > 200 ? '...' : '');
  }

  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  // Getter methods
  get isConnected(): boolean {
    return this.pool !== null && !this.pool.ended;
  }

  get poolStats(): any {
    if (!this.pool) return null;
    
    return {
      totalCount: this.pool.totalCount,
      idleCount: this.pool.idleCount,
      waitingCount: this.pool.waitingCount
    };
  }
}