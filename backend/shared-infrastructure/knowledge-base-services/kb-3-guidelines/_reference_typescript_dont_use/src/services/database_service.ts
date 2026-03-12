// Database Service for KB-3 Guideline Evidence
// PostgreSQL connection and transaction management

import { Pool, PoolClient, QueryResult } from 'pg';

export interface DatabaseConfig {
  host: string;
  port: number;
  database: string;
  username: string;
  password: string;
  ssl?: boolean;
  max?: number;
  idleTimeoutMillis?: number;
  connectionTimeoutMillis?: number;
}

export interface TransactionClient {
  query(sql: string, params?: any[]): Promise<QueryResult>;
  commit(): Promise<void>;
  rollback(): Promise<void>;
}

export class DatabaseService {
  private pool: Pool;
  private config: DatabaseConfig;

  constructor(config: DatabaseConfig) {
    this.config = config;
    this.pool = new Pool({
      host: config.host,
      port: config.port,
      database: config.database,
      user: config.username,
      password: config.password,
      ssl: config.ssl,
      max: config.max || 20,
      idleTimeoutMillis: config.idleTimeoutMillis || 30000,
      connectionTimeoutMillis: config.connectionTimeoutMillis || 2000,
    });

    // Set up error handling
    this.pool.on('error', (err) => {
      console.error('PostgreSQL pool error:', err);
    });
  }

  async initialize(): Promise<void> {
    try {
      // Test connection
      const client = await this.pool.connect();
      await client.query('SELECT NOW()');
      client.release();
      
      // Set up schema if needed
      await this.ensureSchema();
      
      console.log(`Database connected: ${this.config.host}:${this.config.port}/${this.config.database}`);
    } catch (error) {
      console.error('Database initialization failed:', error);
      throw error;
    }
  }

  async query(sql: string, params?: any[]): Promise<QueryResult> {
    try {
      const result = await this.pool.query(sql, params);
      return result;
    } catch (error) {
      console.error('Database query error:', error);
      console.error('SQL:', sql);
      console.error('Params:', params);
      throw error;
    }
  }

  async beginTransaction(): Promise<TransactionClient> {
    const client = await this.pool.connect();
    
    try {
      await client.query('BEGIN');
      
      return {
        query: async (sql: string, params?: any[]) => {
          return await client.query(sql, params);
        },
        
        commit: async () => {
          try {
            await client.query('COMMIT');
          } finally {
            client.release();
          }
        },
        
        rollback: async () => {
          try {
            await client.query('ROLLBACK');
          } finally {
            client.release();
          }
        }
      };
    } catch (error) {
      client.release();
      throw error;
    }
  }

  private async ensureSchema(): Promise<void> {
    // Check if schema exists
    const schemaCheck = await this.query(`
      SELECT schema_name FROM information_schema.schemata 
      WHERE schema_name = 'guideline_evidence'
    `);
    
    if (schemaCheck.rows.length === 0) {
      console.log('Creating guideline_evidence schema...');
      await this.createSchema();
    }
  }

  private async createSchema(): Promise<void> {
    const schemaSQL = `
      CREATE SCHEMA IF NOT EXISTS guideline_evidence;
      SET search_path TO guideline_evidence;

      -- Guidelines metadata table
      CREATE TABLE IF NOT EXISTS guidelines (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        guideline_id VARCHAR(100) UNIQUE NOT NULL,
        organization VARCHAR(100) NOT NULL,
        region VARCHAR(20) NOT NULL,
        condition_primary VARCHAR(200) NOT NULL,
        icd10_codes TEXT[] NOT NULL,
        version VARCHAR(50) NOT NULL,
        effective_date DATE NOT NULL,
        superseded_date DATE,
        supersedes VARCHAR(100),
        evidence_summary JSONB NOT NULL,
        quality_metrics JSONB NOT NULL,
        status VARCHAR(20) DEFAULT 'active',
        approval_status VARCHAR(20) DEFAULT 'pending',
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW(),
        created_by VARCHAR(100),
        approved_by VARCHAR(100),
        digital_signature TEXT
      );

      CREATE INDEX IF NOT EXISTS idx_guideline_region ON guidelines(region);
      CREATE INDEX IF NOT EXISTS idx_guideline_condition ON guidelines(condition_primary);
      CREATE INDEX IF NOT EXISTS idx_guideline_status ON guidelines(status, approval_status);
      CREATE INDEX IF NOT EXISTS idx_guideline_effective ON guidelines(effective_date);

      -- Recommendations table
      CREATE TABLE IF NOT EXISTS recommendations (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        rec_id VARCHAR(100) UNIQUE NOT NULL,
        guideline_id VARCHAR(100) NOT NULL REFERENCES guidelines(guideline_id),
        domain VARCHAR(50) NOT NULL,
        statement TEXT NOT NULL,
        evidence_grade VARCHAR(20) NOT NULL,
        quality_score INTEGER,
        population VARCHAR(100),
        created_at TIMESTAMPTZ DEFAULT NOW()
      );

      CREATE INDEX IF NOT EXISTS idx_rec_guideline ON recommendations(guideline_id);
      CREATE INDEX IF NOT EXISTS idx_rec_domain ON recommendations(domain);
      CREATE INDEX IF NOT EXISTS idx_rec_grade ON recommendations(evidence_grade);

      -- Conflict resolutions table
      CREATE TABLE IF NOT EXISTS conflict_resolutions (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        conflict_id VARCHAR(200) NOT NULL,
        guideline_1_id VARCHAR(100) NOT NULL,
        guideline_2_id VARCHAR(100) NOT NULL,
        resolution_rule VARCHAR(50) NOT NULL,
        winning_guideline VARCHAR(100),
        rationale TEXT NOT NULL,
        safety_override BOOLEAN DEFAULT FALSE,
        patient_factors JSONB,
        clinical_context JSONB,
        resolver_version VARCHAR(20),
        timestamp TIMESTAMPTZ DEFAULT NOW(),
        signed BOOLEAN DEFAULT FALSE,
        checksum VARCHAR(64)
      );

      CREATE INDEX IF NOT EXISTS idx_conflict_timestamp ON conflict_resolutions(timestamp);
      CREATE INDEX IF NOT EXISTS idx_conflict_guidelines ON conflict_resolutions(guideline_1_id, guideline_2_id);
      CREATE INDEX IF NOT EXISTS idx_conflict_rule ON conflict_resolutions(resolution_rule);

      -- Safety overrides table
      CREATE TABLE IF NOT EXISTS safety_overrides (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        override_id VARCHAR(100) UNIQUE NOT NULL,
        priority INTEGER NOT NULL,
        trigger_conditions JSONB NOT NULL,
        override_action JSONB NOT NULL,
        affected_guidelines TEXT[] NOT NULL,
        active BOOLEAN DEFAULT TRUE,
        requires_audit BOOLEAN DEFAULT TRUE,
        created_at TIMESTAMPTZ DEFAULT NOW()
      );

      CREATE INDEX IF NOT EXISTS idx_safety_priority ON safety_overrides(priority);
      CREATE INDEX IF NOT EXISTS idx_safety_active ON safety_overrides(active);

      -- Cross-KB linkages table
      CREATE TABLE IF NOT EXISTS kb_linkages (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        source_guideline VARCHAR(100) NOT NULL,
        source_rec_id VARCHAR(100) NOT NULL,
        target_kb VARCHAR(20) NOT NULL,
        target_id VARCHAR(100) NOT NULL,
        link_type VARCHAR(50) NOT NULL,
        validation_status VARCHAR(20) DEFAULT 'pending',
        last_validated TIMESTAMPTZ,
        validation_errors JSONB,
        created_at TIMESTAMPTZ DEFAULT NOW(),
        UNIQUE(source_guideline, source_rec_id, target_kb, target_id)
      );

      CREATE INDEX IF NOT EXISTS idx_kb_links_status ON kb_linkages(validation_status);
      CREATE INDEX IF NOT EXISTS idx_kb_links_target ON kb_linkages(target_kb, target_id);

      -- Guideline transitions table
      CREATE TABLE IF NOT EXISTS guideline_transitions (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        old_guideline_id VARCHAR(100),
        new_guideline_id VARCHAR(100),
        transition_date DATE NOT NULL,
        major_changes JSONB NOT NULL,
        clinical_impact_score INTEGER,
        requires_notification BOOLEAN DEFAULT FALSE,
        requires_training BOOLEAN DEFAULT FALSE,
        transition_period_days INTEGER DEFAULT 30,
        parallel_run BOOLEAN DEFAULT TRUE,
        created_at TIMESTAMPTZ DEFAULT NOW()
      );

      CREATE INDEX IF NOT EXISTS idx_transition_date ON guideline_transitions(transition_date);

      -- Safety override audit table
      CREATE TABLE IF NOT EXISTS safety_override_audit (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        override_id VARCHAR(100) NOT NULL,
        patient_context JSONB NOT NULL,
        action_taken JSONB NOT NULL,
        rationale TEXT NOT NULL,
        applied_at TIMESTAMPTZ DEFAULT NOW(),
        engine_version VARCHAR(20),
        checksum VARCHAR(64)
      );

      CREATE INDEX IF NOT EXISTS idx_safety_audit_override ON safety_override_audit(override_id);
      CREATE INDEX IF NOT EXISTS idx_safety_audit_time ON safety_override_audit(applied_at);

      -- Audit log table
      CREATE TABLE IF NOT EXISTS audit_log (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        event_type VARCHAR(100) NOT NULL,
        event_category VARCHAR(50) NOT NULL,
        severity VARCHAR(20) NOT NULL,
        event_data JSONB NOT NULL,
        timestamp TIMESTAMPTZ DEFAULT NOW(),
        user_id VARCHAR(100),
        session_id VARCHAR(100),
        requires_signature BOOLEAN DEFAULT FALSE,
        checksum VARCHAR(64)
      );

      CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_log(timestamp);
      CREATE INDEX IF NOT EXISTS idx_audit_type ON audit_log(event_type);
      CREATE INDEX IF NOT EXISTS idx_audit_severity ON audit_log(severity);
    `;

    await this.query(schemaSQL);
    console.log('Database schema created successfully');
  }

  // Specific query methods for common operations
  async getActiveGuidelines(region?: string): Promise<any[]> {
    let sql = `
      SELECT * FROM guideline_evidence.guidelines 
      WHERE status = 'active' AND approval_status = 'approved'
    `;
    const params: any[] = [];
    
    if (region) {
      sql += ` AND (region = $1 OR region IN ('Global', 'WHO'))`;
      params.push(region);
    }
    
    sql += ` ORDER BY effective_date DESC`;
    
    const result = await this.query(sql, params);
    return result.rows;
  }

  async getGuidelineById(guideline_id: string): Promise<any | null> {
    const result = await this.query(`
      SELECT g.*, 
             json_agg(r.*) as recommendations
      FROM guideline_evidence.guidelines g
      LEFT JOIN guideline_evidence.recommendations r ON g.guideline_id = r.guideline_id
      WHERE g.guideline_id = $1 AND g.status = 'active'
      GROUP BY g.id
    `, [guideline_id]);
    
    return result.rows[0] || null;
  }

  async getConflictsByGuidelineIds(guideline_ids: string[]): Promise<any[]> {
    const result = await this.query(`
      SELECT * FROM guideline_evidence.conflict_resolutions
      WHERE (guideline_1_id = ANY($1) OR guideline_2_id = ANY($1))
      ORDER BY timestamp DESC
    `, [guideline_ids]);
    
    return result.rows;
  }

  async getSafetyOverrides(active_only: boolean = true): Promise<any[]> {
    let sql = 'SELECT * FROM guideline_evidence.safety_overrides';
    const params: any[] = [];
    
    if (active_only) {
      sql += ' WHERE active = true';
    }
    
    sql += ' ORDER BY priority';
    
    const result = await this.query(sql, params);
    return result.rows;
  }

  async logSafetyOverride(
    override_id: string,
    patient_context: any,
    action_taken: any,
    rationale: string
  ): Promise<void> {
    const checksum = this.generateAuditChecksum({
      override_id,
      patient_context,
      action_taken,
      timestamp: new Date()
    });

    await this.query(`
      INSERT INTO guideline_evidence.safety_override_audit (
        override_id, patient_context, action_taken, rationale,
        engine_version, checksum
      ) VALUES ($1, $2, $3, $4, $5, $6)
    `, [
      override_id,
      JSON.stringify(patient_context),
      JSON.stringify(action_taken),
      rationale,
      '3.0.0',
      checksum
    ]);
  }

  private generateAuditChecksum(data: any): string {
    const crypto = require('crypto');
    return crypto.createHash('sha256')
      .update(JSON.stringify(data))
      .digest('hex');
  }

  async close(): Promise<void> {
    await this.pool.end();
    console.log('Database connection pool closed');
  }

  async healthCheck(): Promise<any> {
    try {
      const result = await this.query('SELECT NOW() as timestamp, version() as version');
      return {
        status: 'healthy',
        timestamp: result.rows[0].timestamp,
        database_version: result.rows[0].version,
        pool_total: this.pool.totalCount,
        pool_idle: this.pool.idleCount,
        pool_waiting: this.pool.waitingCount
      };
    } catch (error) {
      return {
        status: 'unhealthy',
        error: error.message
      };
    }
  }

  // Performance monitoring methods
  async getTableStats(): Promise<any> {
    const result = await this.query(`
      SELECT 
        schemaname,
        tablename,
        n_tup_ins as inserts,
        n_tup_upd as updates,
        n_tup_del as deletes,
        n_live_tup as live_rows,
        n_dead_tup as dead_rows
      FROM pg_stat_user_tables 
      WHERE schemaname = 'guideline_evidence'
      ORDER BY n_live_tup DESC
    `);
    
    return result.rows;
  }

  async getQueryPerformance(): Promise<any> {
    const result = await this.query(`
      SELECT 
        query,
        calls,
        total_time,
        mean_time,
        min_time,
        max_time
      FROM pg_stat_statements
      WHERE query LIKE '%guideline_evidence%'
      ORDER BY total_time DESC
      LIMIT 10
    `);
    
    return result.rows;
  }

  // Backup and maintenance
  async vacuum(analyze: boolean = true): Promise<void> {
    const tables = [
      'guidelines',
      'recommendations', 
      'conflict_resolutions',
      'safety_overrides',
      'kb_linkages',
      'audit_log'
    ];
    
    for (const table of tables) {
      const command = analyze ? `VACUUM ANALYZE guideline_evidence.${table}` : `VACUUM guideline_evidence.${table}`;
      await this.query(command);
    }
  }

  async getStorageUsage(): Promise<any> {
    const result = await this.query(`
      SELECT 
        schemaname,
        tablename,
        pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size,
        pg_total_relation_size(schemaname||'.'||tablename) as size_bytes
      FROM pg_tables 
      WHERE schemaname = 'guideline_evidence'
      ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
    `);
    
    return result.rows;
  }

  // Cleanup and archival
  async archiveOldConflictResolutions(days: number = 365): Promise<number> {
    const result = await this.query(`
      DELETE FROM guideline_evidence.conflict_resolutions
      WHERE timestamp < NOW() - INTERVAL '${days} days'
      AND safety_override = false
    `);
    
    return result.rowCount || 0;
  }

  async cleanupInvalidLinks(): Promise<number> {
    const result = await this.query(`
      DELETE FROM guideline_evidence.kb_linkages
      WHERE validation_status = 'broken'
      AND last_validated < NOW() - INTERVAL '30 days'
    `);
    
    return result.rowCount || 0;
  }
}