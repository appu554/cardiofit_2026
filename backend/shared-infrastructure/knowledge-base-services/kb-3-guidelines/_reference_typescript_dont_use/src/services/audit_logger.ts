// Audit Logger Service for KB-3 Guideline Evidence
// Comprehensive audit logging for clinical decision compliance

import { DatabaseService } from './database_service';

export interface AuditEntry {
  event_type: string;
  event_category?: string;
  severity: 'info' | 'warning' | 'error' | 'critical';
  event_data: any;
  timestamp?: Date;
  user_id?: string;
  session_id?: string;
  requires_signature?: boolean;
}

export class AuditLogger {
  private database: DatabaseService | null;
  private bufferSize: number = 100;
  private flushInterval: number = 5000; // 5 seconds
  private buffer: AuditEntry[] = [];
  private flushTimer: NodeJS.Timeout | null = null;

  constructor(database: DatabaseService | null = null) {
    this.database = database;
    
    if (database) {
      this.startPeriodicFlush();
    }
  }

  async log(entry: AuditEntry): Promise<void> {
    const auditEntry: AuditEntry = {
      ...entry,
      event_category: entry.event_category || 'system_operation',
      timestamp: entry.timestamp || new Date()
    };

    // Generate checksum for integrity if signature required
    if (entry.requires_signature) {
      (auditEntry as any).checksum = this.generateChecksum(auditEntry);
    }

    // Add to buffer
    this.buffer.push(auditEntry);

    // Log critical events immediately to console
    if (entry.severity === 'critical' || entry.severity === 'error') {
      console.error(`[AUDIT ${entry.severity.toUpperCase()}] ${entry.event_type}:`, entry.event_data);
    }

    // Flush if buffer is full
    if (this.buffer.length >= this.bufferSize) {
      await this.flush();
    }

    // For critical events requiring immediate persistence
    if (entry.severity === 'critical' || entry.requires_signature) {
      await this.flush();
    }
  }

  private async flush(): Promise<void> {
    if (!this.database || this.buffer.length === 0) {
      return;
    }

    const entries = [...this.buffer];
    this.buffer = [];

    try {
      // Batch insert for performance
      const values = entries.map((entry, index) => {
        const baseIndex = index * 9;
        return `($${baseIndex + 1}, $${baseIndex + 2}, $${baseIndex + 3}, $${baseIndex + 4}, $${baseIndex + 5}, $${baseIndex + 6}, $${baseIndex + 7}, $${baseIndex + 8}, $${baseIndex + 9})`;
      }).join(', ');

      const params = entries.flatMap(entry => [
        entry.event_type,
        entry.event_category,
        entry.severity,
        JSON.stringify(entry.event_data),
        entry.timestamp,
        entry.user_id || null,
        entry.session_id || null,
        entry.requires_signature || false,
        (entry as any).checksum || null
      ]);

      const sql = `
        INSERT INTO guideline_evidence.audit_log (
          event_type, event_category, severity, event_data, timestamp,
          user_id, session_id, requires_signature, checksum
        ) VALUES ${values}
      `;

      await this.database.query(sql, params);
      
    } catch (error) {
      console.error('Failed to persist audit entries:', error);
      
      // Re-add entries to buffer for retry (up to 3 retries)
      if (!entries[0].retryCount || entries[0].retryCount < 3) {
        entries.forEach(entry => {
          (entry as any).retryCount = ((entry as any).retryCount || 0) + 1;
        });
        this.buffer.unshift(...entries);
      } else {
        console.error('Dropping audit entries after 3 failed retries');
      }
    }
  }

  private generateChecksum(entry: AuditEntry): string {
    const crypto = require('crypto');
    const data = {
      event_type: entry.event_type,
      event_data: entry.event_data,
      timestamp: entry.timestamp,
      user_id: entry.user_id,
      session_id: entry.session_id
    };
    
    const hash = crypto.createHash('sha256');
    hash.update(JSON.stringify(data));
    return hash.digest('hex');
  }

  private startPeriodicFlush(): void {
    this.flushTimer = setInterval(() => {
      this.flush().catch(error => {
        console.error('Periodic flush failed:', error);
      });
    }, this.flushInterval);
  }

  async close(): Promise<void> {
    if (this.flushTimer) {
      clearInterval(this.flushTimer);
      this.flushTimer = null;
    }
    
    // Flush remaining entries
    await this.flush();
  }

  // Query methods for audit analysis
  async getAuditEntries(
    filters: {
      event_type?: string;
      severity?: string;
      start_time?: Date;
      end_time?: Date;
      user_id?: string;
    } = {},
    limit: number = 100
  ): Promise<any[]> {
    if (!this.database) {
      throw new Error('Database connection required for audit queries');
    }

    let sql = 'SELECT * FROM guideline_evidence.audit_log WHERE 1=1';
    const params: any[] = [];
    let paramIndex = 1;

    if (filters.event_type) {
      sql += ` AND event_type = $${paramIndex}`;
      params.push(filters.event_type);
      paramIndex++;
    }

    if (filters.severity) {
      sql += ` AND severity = $${paramIndex}`;
      params.push(filters.severity);
      paramIndex++;
    }

    if (filters.start_time) {
      sql += ` AND timestamp >= $${paramIndex}`;
      params.push(filters.start_time);
      paramIndex++;
    }

    if (filters.end_time) {
      sql += ` AND timestamp <= $${paramIndex}`;
      params.push(filters.end_time);
      paramIndex++;
    }

    if (filters.user_id) {
      sql += ` AND user_id = $${paramIndex}`;
      params.push(filters.user_id);
      paramIndex++;
    }

    sql += ` ORDER BY timestamp DESC LIMIT $${paramIndex}`;
    params.push(limit);

    const result = await this.database.query(sql, params);
    return result.rows;
  }

  async getEventTypeStatistics(timeRange?: { start: Date; end: Date }): Promise<any[]> {
    if (!this.database) {
      throw new Error('Database connection required for statistics');
    }

    let whereClause = '';
    const params: any[] = [];
    
    if (timeRange) {
      whereClause = 'WHERE timestamp BETWEEN $1 AND $2';
      params.push(timeRange.start, timeRange.end);
    }

    const result = await this.database.query(`
      SELECT 
        event_type,
        event_category,
        severity,
        COUNT(*) as event_count,
        COUNT(*) FILTER (WHERE requires_signature = true) as signed_events,
        MIN(timestamp) as first_occurrence,
        MAX(timestamp) as last_occurrence
      FROM guideline_evidence.audit_log
      ${whereClause}
      GROUP BY event_type, event_category, severity
      ORDER BY event_count DESC
    `, params);

    return result.rows;
  }

  async getCriticalEvents(hours: number = 24): Promise<any[]> {
    if (!this.database) {
      throw new Error('Database connection required for critical event queries');
    }

    const result = await this.database.query(`
      SELECT * FROM guideline_evidence.audit_log
      WHERE severity IN ('critical', 'error')
      AND timestamp >= NOW() - INTERVAL '${hours} hours'
      ORDER BY timestamp DESC
    `);

    return result.rows;
  }

  async getSignedEvents(): Promise<any[]> {
    if (!this.database) {
      throw new Error('Database connection required for signed event queries');
    }

    const result = await this.database.query(`
      SELECT * FROM guideline_evidence.audit_log
      WHERE requires_signature = true
      ORDER BY timestamp DESC
    `);

    return result.rows;
  }

  // Compliance reporting
  async generateComplianceReport(timeRange: { start: Date; end: Date }): Promise<any> {
    if (!this.database) {
      throw new Error('Database connection required for compliance reports');
    }

    const stats = await this.getEventTypeStatistics(timeRange);
    const criticalEvents = await this.database.query(`
      SELECT COUNT(*) as critical_count
      FROM guideline_evidence.audit_log
      WHERE severity = 'critical'
      AND timestamp BETWEEN $1 AND $2
    `, [timeRange.start, timeRange.end]);

    const signatureCompliance = await this.database.query(`
      SELECT 
        COUNT(*) FILTER (WHERE requires_signature = true) as signature_required,
        COUNT(*) FILTER (WHERE requires_signature = true AND checksum IS NOT NULL) as signatures_present
      FROM guideline_evidence.audit_log
      WHERE timestamp BETWEEN $1 AND $2
    `, [timeRange.start, timeRange.end]);

    const compliance = signatureCompliance.rows[0];
    const signatureRate = compliance.signature_required > 0 
      ? (compliance.signatures_present / compliance.signature_required) * 100 
      : 100;

    return {
      time_range: timeRange,
      total_events: stats.reduce((sum, stat) => sum + parseInt(stat.event_count), 0),
      event_breakdown: stats,
      critical_events: parseInt(criticalEvents.rows[0].critical_count),
      signature_compliance_rate: Math.round(signatureRate * 100) / 100,
      generated_at: new Date()
    };
  }

  // Security methods
  async verifyAuditIntegrity(entry_id: string): Promise<boolean> {
    if (!this.database) {
      return false;
    }

    const result = await this.database.query(`
      SELECT * FROM guideline_evidence.audit_log WHERE id = $1
    `, [entry_id]);

    if (result.rows.length === 0) {
      return false;
    }

    const entry = result.rows[0];
    if (!entry.requires_signature || !entry.checksum) {
      return true; // Not required to be signed
    }

    // Regenerate checksum and compare
    const expectedChecksum = this.generateChecksum({
      event_type: entry.event_type,
      event_data: entry.event_data,
      timestamp: entry.timestamp,
      user_id: entry.user_id,
      session_id: entry.session_id
    });

    return expectedChecksum === entry.checksum;
  }

  async detectAuditTampering(): Promise<any[]> {
    if (!this.database) {
      return [];
    }

    const tamperedEntries = [];
    
    const signedEntries = await this.getSignedEvents();
    
    for (const entry of signedEntries) {
      const isValid = await this.verifyAuditIntegrity(entry.id);
      if (!isValid) {
        tamperedEntries.push({
          entry_id: entry.id,
          event_type: entry.event_type,
          timestamp: entry.timestamp,
          issue: 'Checksum mismatch'
        });
      }
    }

    return tamperedEntries;
  }
}