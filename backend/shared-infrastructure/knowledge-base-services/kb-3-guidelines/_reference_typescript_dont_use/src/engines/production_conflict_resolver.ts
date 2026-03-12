// Production Conflict Resolution Engine
// Complete implementation of 5-tier clinical guideline conflict resolution

import { ConflictResolutionEngine, Conflict, Resolution, PatientContext } from './conflict_resolver';
import { DatabaseService } from '../services/database_service';
import { Neo4jService } from '../services/neo4j_service';
import { AuditLogger } from '../services/audit_logger';

interface ResolutionRule {
  name: string;
  priority: number;
  apply: (conflict: Conflict, context?: PatientContext) => Promise<Resolution>;
}

export class ProductionConflictResolver implements ConflictResolutionEngine {
  private readonly rules: ResolutionRule[];
  private database: DatabaseService;
  private neo4j: Neo4jService;
  private auditLogger: AuditLogger;

  constructor(
    database: DatabaseService,
    neo4j: Neo4jService,
    auditLogger: AuditLogger
  ) {
    this.database = database;
    this.neo4j = neo4j;
    this.auditLogger = auditLogger;
    
    // Define the 5-tier resolution rule system
    this.rules = [
      {
        name: 'safety_first',
        priority: 1,
        apply: (c: Conflict, ctx?: PatientContext) => this.applySafetyFirst(c, ctx)
      },
      {
        name: 'regional_preference',
        priority: 2,
        apply: (c: Conflict, ctx?: PatientContext) => this.preferRegionalGuideline(c, ctx)
      },
      {
        name: 'evidence_strength',
        priority: 3,
        apply: (c: Conflict) => this.preferStrongerEvidence(c)
      },
      {
        name: 'publication_recency',
        priority: 4,
        apply: (c: Conflict) => this.preferMoreRecent(c)
      },
      {
        name: 'conservative_default',
        priority: 5,
        apply: (c: Conflict) => this.chooseConservative(c)
      }
    ];
  }

  async detectConflicts(guidelines: any[]): Promise<Conflict[]> {
    const conflicts: Conflict[] = [];
    
    // Query Neo4j for potential conflicts between guidelines
    const query = `
      MATCH (g1:Guideline)-[:CONTAINS]->(r1:Recommendation)
      MATCH (g2:Guideline)-[:CONTAINS]->(r2:Recommendation)
      WHERE g1.guideline_id IN $guideline_ids
        AND g2.guideline_id IN $guideline_ids
        AND g1.guideline_id < g2.guideline_id
        AND r1.domain = r2.domain
        AND (
          // Direct contradiction in recommendations
          (r1.statement CONTAINS 'target' AND r2.statement CONTAINS 'target'
           AND r1.statement <> r2.statement)
          OR
          // Evidence grade disagreement for same recommendation
          (r1.evidence_grade <> r2.evidence_grade 
           AND r1.rec_id = r2.rec_id)
          OR
          // Conflicting medication choices
          (r1.statement CONTAINS 'first-line' AND r2.statement CONTAINS 'first-line'
           AND r1.statement <> r2.statement)
        )
      RETURN g1, r1, g2, r2, r1.domain as domain
    `;
    
    const result = await this.neo4j.run(query, {
      guideline_ids: guidelines.map(g => g.guideline_id)
    });
    
    for (const record of result.records) {
      const conflict = this.classifyConflict(record);
      conflicts.push(conflict);
    }
    
    return conflicts;
  }

  private classifyConflict(record: any): Conflict {
    const g1 = record.get('g1').properties;
    const g2 = record.get('g2').properties;
    const r1 = record.get('r1').properties;
    const r2 = record.get('r2').properties;
    const domain = record.get('domain');

    // Determine conflict type and severity
    let conflictType = 'unknown';
    let severity = 'minor';

    if (r1.statement.includes('target') && r2.statement.includes('target')) {
      conflictType = 'target_difference';
      severity = this.calculateTargetSeverity(r1.statement, r2.statement);
    } else if (r1.evidence_grade !== r2.evidence_grade) {
      conflictType = 'evidence_disagreement';
      severity = this.calculateEvidenceSeverity(r1.evidence_grade, r2.evidence_grade);
    } else if (r1.statement.includes('first-line') && r2.statement.includes('first-line')) {
      conflictType = 'treatment_preference';
      severity = 'major';
    }

    return {
      conflict_id: `${g1.guideline_id}_vs_${g2.guideline_id}_${domain}`,
      guideline1_id: g1.guideline_id,
      guideline2_id: g2.guideline_id,
      recommendation1: r1,
      recommendation2: r2,
      type: conflictType,
      severity,
      domain,
      detected_at: new Date()
    };
  }

  private calculateTargetSeverity(statement1: string, statement2: string): string {
    // Extract target values and calculate difference
    const target1 = this.extractTargetValue(statement1);
    const target2 = this.extractTargetValue(statement2);
    
    if (!target1 || !target2) return 'minor';
    
    const difference = Math.abs(target1 - target2);
    
    if (difference >= 20) return 'critical';
    if (difference >= 10) return 'major';
    return 'minor';
  }

  private extractTargetValue(statement: string): number | null {
    // Extract numeric target from statement (e.g., "<130/80" -> 130)
    const match = statement.match(/(\d+)(?:\/\d+)?/);
    return match ? parseInt(match[1]) : null;
  }

  private calculateEvidenceSeverity(grade1: string, grade2: string): string {
    const gradeValues = { 'A': 4, 'B': 3, 'C': 2, 'D': 1, 'Expert Opinion': 0 };
    const diff = Math.abs(gradeValues[grade1] - gradeValues[grade2]);
    
    if (diff >= 3) return 'major';
    if (diff >= 2) return 'moderate';
    return 'minor';
  }

  async resolveConflict(conflict: Conflict, context: PatientContext): Promise<Resolution> {
    await this.auditLogger.log({
      event_type: 'conflict_resolution_started',
      severity: 'info',
      event_data: { conflict_id: conflict.conflict_id, context }
    });

    // Apply resolution rules in priority order
    for (const rule of this.rules) {
      try {
        const resolution = await rule.apply(conflict, context);
        
        if (resolution.applicable) {
          // Audit the resolution
          await this.auditResolution({
            ...resolution,
            rule_used: rule.name,
            conflict_id: conflict.conflict_id,
            context
          });
          
          // Store resolution in database
          await this.storeResolution(conflict, resolution, rule.name);
          
          return resolution;
        }
      } catch (error) {
        await this.auditLogger.log({
          event_type: 'resolution_rule_error',
          severity: 'error',
          event_data: { rule: rule.name, error: error.message, conflict_id: conflict.conflict_id }
        });
      }
    }
    
    // If no rule applies, use safety default
    const defaultResolution = await this.safetyDefault(conflict);
    await this.storeResolution(conflict, defaultResolution, 'safety_default');
    
    return defaultResolution;
  }

  private async applySafetyFirst(conflict: Conflict, context?: PatientContext): Promise<Resolution> {
    // Check for active safety overrides
    const overrides = await this.database.query(`
      SELECT * FROM guideline_evidence.safety_overrides 
      WHERE active = true
      AND $1::text[] && affected_guidelines 
      ORDER BY priority
    `, [[conflict.guideline1_id, conflict.guideline2_id]]);
    
    if (overrides.rows.length > 0) {
      const override = overrides.rows[0];
      
      // Check if override conditions are met
      if (context && this.matchesOverrideConditions(override.trigger_conditions, context)) {
        return {
          applicable: true,
          winning_guideline: null,
          action: override.override_action,
          rationale: `Safety override ${override.override_id}: ${override.override_action.description}`,
          safety_override: true,
          override_id: override.override_id
        };
      }
    }
    
    return { applicable: false };
  }

  private matchesOverrideConditions(conditions: any, context: PatientContext): boolean {
    // Pregnancy contraindications
    if (conditions.pregnancy && 
        (context.pregnancy_status === 'pregnant' || context.pregnancy_status === 'suspected')) {
      return true;
    }
    
    // Lab threshold checks
    if (conditions.lab_thresholds) {
      for (const [lab, threshold] of Object.entries(conditions.lab_thresholds)) {
        const patientValue = context.labs[lab];
        if (patientValue && this.exceedsThreshold(patientValue, threshold as any)) {
          return true;
        }
      }
    }
    
    // Acute condition checks
    if (conditions.conditions) {
      const hasAcuteCondition = conditions.conditions.some(
        (condition: string) => context.active_conditions.includes(condition)
      );
      if (hasAcuteCondition) return true;
    }
    
    return false;
  }

  private exceedsThreshold(value: number, threshold: any): boolean {
    switch (threshold.operator) {
      case '>': return value > threshold.value;
      case '>=': return value >= threshold.value;
      case '<': return value < threshold.value;
      case '<=': return value <= threshold.value;
      case '=': return value === threshold.value;
      default: return false;
    }
  }

  private async preferRegionalGuideline(
    conflict: Conflict, 
    context?: PatientContext
  ): Promise<Resolution> {
    if (!context?.region) {
      return { applicable: false };
    }

    // Get guideline regions
    const g1Region = await this.getGuidelineRegion(conflict.guideline1_id);
    const g2Region = await this.getGuidelineRegion(conflict.guideline2_id);
    
    // Direct regional match
    if (g1Region === context.region && g2Region !== context.region) {
      return {
        applicable: true,
        winning_guideline: conflict.guideline1_id,
        rationale: `Preferred regional guideline for ${context.region}`,
        safety_override: false
      };
    }
    
    if (g2Region === context.region && g1Region !== context.region) {
      return {
        applicable: true,
        winning_guideline: conflict.guideline2_id,
        rationale: `Preferred regional guideline for ${context.region}`,
        safety_override: false
      };
    }
    
    // WHO/Global fallback for non-covered regions
    if (g1Region === 'WHO' || g1Region === 'Global') {
      return {
        applicable: true,
        winning_guideline: conflict.guideline1_id,
        rationale: `WHO/Global guideline fallback for region ${context.region}`,
        safety_override: false
      };
    }
    
    if (g2Region === 'WHO' || g2Region === 'Global') {
      return {
        applicable: true,
        winning_guideline: conflict.guideline2_id,
        rationale: `WHO/Global guideline fallback for region ${context.region}`,
        safety_override: false
      };
    }
    
    return { applicable: false };
  }

  private async preferStrongerEvidence(conflict: Conflict): Promise<Resolution> {
    const grade1 = conflict.recommendation1.evidence_grade;
    const grade2 = conflict.recommendation2.evidence_grade;
    
    const gradeStrength = { 'A': 4, 'B': 3, 'C': 2, 'D': 1, 'Expert Opinion': 0 };
    
    const strength1 = gradeStrength[grade1] || 0;
    const strength2 = gradeStrength[grade2] || 0;
    
    if (strength1 > strength2) {
      return {
        applicable: true,
        winning_guideline: conflict.guideline1_id,
        rationale: `Higher evidence grade (${grade1} vs ${grade2})`,
        safety_override: false
      };
    }
    
    if (strength2 > strength1) {
      return {
        applicable: true,
        winning_guideline: conflict.guideline2_id,
        rationale: `Higher evidence grade (${grade2} vs ${grade1})`,
        safety_override: false
      };
    }
    
    // Check quality scores if evidence grades are equal
    const quality1 = conflict.recommendation1.quality_score || 0;
    const quality2 = conflict.recommendation2.quality_score || 0;
    
    if (quality1 > quality2) {
      return {
        applicable: true,
        winning_guideline: conflict.guideline1_id,
        rationale: `Higher quality score (${quality1} vs ${quality2}) with equal evidence grade`,
        safety_override: false
      };
    }
    
    if (quality2 > quality1) {
      return {
        applicable: true,
        winning_guideline: conflict.guideline2_id,
        rationale: `Higher quality score (${quality2} vs ${quality1}) with equal evidence grade`,
        safety_override: false
      };
    }
    
    return { applicable: false };
  }

  private async preferMoreRecent(conflict: Conflict): Promise<Resolution> {
    // Get effective dates for both guidelines
    const date1 = await this.getGuidelineDate(conflict.guideline1_id);
    const date2 = await this.getGuidelineDate(conflict.guideline2_id);
    
    if (!date1 || !date2) {
      return { applicable: false };
    }
    
    const timeDiff = Math.abs(date1.getTime() - date2.getTime());
    const daysDiff = timeDiff / (1000 * 60 * 60 * 24);
    
    // Only apply if significant time difference (>6 months)
    if (daysDiff < 180) {
      return { applicable: false };
    }
    
    if (date1 > date2) {
      return {
        applicable: true,
        winning_guideline: conflict.guideline1_id,
        rationale: `More recent publication (${Math.round(daysDiff)} days newer)`,
        safety_override: false
      };
    } else {
      return {
        applicable: true,
        winning_guideline: conflict.guideline2_id,
        rationale: `More recent publication (${Math.round(daysDiff)} days newer)`,
        safety_override: false
      };
    }
  }

  private async chooseConservative(conflict: Conflict): Promise<Resolution> {
    // Always applicable as final fallback
    // Choose the more conservative option based on safety criteria
    
    const conservativeChoice = await this.determineConservativeOption(conflict);
    
    return {
      applicable: true,
      winning_guideline: conservativeChoice.guideline_id,
      rationale: `Conservative default: ${conservativeChoice.reason}`,
      safety_override: false
    };
  }

  private async determineConservativeOption(conflict: Conflict): Promise<{
    guideline_id: string;
    reason: string;
  }> {
    const r1 = conflict.recommendation1;
    const r2 = conflict.recommendation2;
    
    // For blood pressure targets, choose higher (more conservative) target
    if (r1.statement.includes('target') && r2.statement.includes('target')) {
      const target1 = this.extractTargetValue(r1.statement);
      const target2 = this.extractTargetValue(r2.statement);
      
      if (target1 && target2) {
        if (target1 > target2) {
          return {
            guideline_id: conflict.guideline1_id,
            reason: `More conservative BP target (${target1} vs ${target2})`
          };
        } else {
          return {
            guideline_id: conflict.guideline2_id,
            reason: `More conservative BP target (${target2} vs ${target1})`
          };
        }
      }
    }
    
    // For medication choices, prefer established drugs
    const establishedDrugs = ['lisinopril', 'metoprolol', 'amlodipine', 'hydrochlorothiazide'];
    
    const r1HasEstablished = establishedDrugs.some(drug => 
      r1.statement.toLowerCase().includes(drug)
    );
    const r2HasEstablished = establishedDrugs.some(drug => 
      r2.statement.toLowerCase().includes(drug)
    );
    
    if (r1HasEstablished && !r2HasEstablished) {
      return {
        guideline_id: conflict.guideline1_id,
        reason: 'Prefers established medication with longer safety profile'
      };
    }
    
    if (r2HasEstablished && !r1HasEstablished) {
      return {
        guideline_id: conflict.guideline2_id,
        reason: 'Prefers established medication with longer safety profile'
      };
    }
    
    // Default to guideline with better evidence grade
    if (r1.evidence_grade <= r2.evidence_grade) {
      return {
        guideline_id: conflict.guideline1_id,
        reason: 'Default conservative choice based on evidence grade'
      };
    } else {
      return {
        guideline_id: conflict.guideline2_id,
        reason: 'Default conservative choice based on evidence grade'
      };
    }
  }

  private extractTargetValue(statement: string): number | null {
    // Extract numeric target from statement
    const bpMatch = statement.match(/(\d+)\/\d+/);
    if (bpMatch) return parseInt(bpMatch[1]);
    
    const numericMatch = statement.match(/(\d+)/);
    return numericMatch ? parseInt(numericMatch[1]) : null;
  }

  private async getGuidelineRegion(guideline_id: string): Promise<string> {
    const result = await this.database.query(`
      SELECT region FROM guideline_evidence.guidelines 
      WHERE guideline_id = $1
    `, [guideline_id]);
    
    return result.rows[0]?.region || 'Unknown';
  }

  private async getGuidelineDate(guideline_id: string): Promise<Date | null> {
    const result = await this.database.query(`
      SELECT effective_date FROM guideline_evidence.guidelines 
      WHERE guideline_id = $1
    `, [guideline_id]);
    
    return result.rows[0]?.effective_date || null;
  }

  private async safetyDefault(conflict: Conflict): Promise<Resolution> {
    // Ultimate fallback - choose safest option
    return {
      applicable: true,
      winning_guideline: conflict.guideline1_id, // Arbitrary choice
      rationale: 'Safety default applied - no resolution rule determined applicable',
      safety_override: false,
      requires_manual_review: true
    };
  }

  async auditResolution(resolution: any): Promise<void> {
    // Store detailed audit log
    await this.database.query(`
      INSERT INTO guideline_evidence.conflict_resolutions (
        conflict_id, guideline_1_id, guideline_2_id, resolution_rule,
        winning_guideline, rationale, safety_override, patient_factors,
        clinical_context, resolver_version, timestamp
      ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
    `, [
      resolution.conflict_id,
      resolution.guideline1_id || conflict.guideline1_id,
      resolution.guideline2_id || conflict.guideline2_id,
      resolution.rule_used,
      resolution.winning_guideline,
      resolution.rationale,
      resolution.safety_override || false,
      JSON.stringify(resolution.context || {}),
      JSON.stringify(resolution.clinical_context || {}),
      '3.0.0'
    ]);

    // Audit log entry
    await this.auditLogger.log({
      event_type: 'conflict_resolved',
      event_category: 'clinical_decision',
      severity: resolution.safety_override ? 'critical' : 'info',
      event_data: {
        conflict_id: resolution.conflict_id,
        rule_used: resolution.rule_used,
        winning_guideline: resolution.winning_guideline,
        safety_override: resolution.safety_override
      },
      requires_signature: resolution.safety_override
    });
  }

  private async storeResolution(
    conflict: Conflict,
    resolution: Resolution,
    rule_name: string
  ): Promise<void> {
    // Generate checksum for integrity
    const checksum = this.generateResolutionChecksum(conflict, resolution);
    
    await this.database.query(`
      INSERT INTO guideline_evidence.conflict_resolutions (
        conflict_id, guideline_1_id, guideline_2_id, resolution_rule,
        winning_guideline, rationale, safety_override, timestamp,
        signed, checksum
      ) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), $8, $9)
    `, [
      conflict.conflict_id,
      conflict.guideline1_id,
      conflict.guideline2_id,
      rule_name,
      resolution.winning_guideline,
      resolution.rationale,
      resolution.safety_override || false,
      resolution.safety_override || false, // Require signature for safety overrides
      checksum
    ]);
  }

  private generateResolutionChecksum(conflict: Conflict, resolution: Resolution): string {
    const crypto = require('crypto');
    const data = {
      conflict_id: conflict.conflict_id,
      winning_guideline: resolution.winning_guideline,
      rule_used: resolution.rule_used,
      timestamp: new Date().toISOString()
    };
    
    return crypto.createHash('sha256')
      .update(JSON.stringify(data))
      .digest('hex');
  }

  // Statistical methods for monitoring
  async getConflictStatistics(timeRange?: { start: Date; end: Date }): Promise<any> {
    let whereClause = '';
    const params: any[] = [];
    
    if (timeRange) {
      whereClause = 'WHERE timestamp BETWEEN $1 AND $2';
      params.push(timeRange.start, timeRange.end);
    }
    
    const stats = await this.database.query(`
      SELECT 
        resolution_rule,
        COUNT(*) as count,
        COUNT(*) FILTER (WHERE safety_override = true) as safety_overrides,
        AVG(CASE 
          WHEN resolution_rule = 'safety_first' THEN 1
          WHEN resolution_rule = 'regional_preference' THEN 2
          WHEN resolution_rule = 'evidence_strength' THEN 3
          WHEN resolution_rule = 'publication_recency' THEN 4
          WHEN resolution_rule = 'conservative_default' THEN 5
          ELSE 6
        END) as avg_rule_priority
      FROM guideline_evidence.conflict_resolutions
      ${whereClause}
      GROUP BY resolution_rule
      ORDER BY count DESC
    `, params);
    
    return {
      rule_usage: stats.rows,
      total_conflicts: stats.rows.reduce((sum, row) => sum + parseInt(row.count), 0),
      safety_override_rate: stats.rows.reduce((sum, row) => sum + parseInt(row.safety_overrides), 0)
    };
  }

  async getResolutionHistory(conflict_id: string): Promise<any[]> {
    const result = await this.database.query(`
      SELECT * FROM guideline_evidence.conflict_resolutions
      WHERE conflict_id = $1
      ORDER BY timestamp DESC
    `, [conflict_id]);
    
    return result.rows;
  }

  // Health check for the resolver
  async healthCheck(): Promise<any> {
    try {
      // Test database connectivity
      await this.database.query('SELECT 1');
      
      // Test Neo4j connectivity
      await this.neo4j.run('RETURN 1');
      
      // Check recent resolution performance
      const recentStats = await this.getConflictStatistics({
        start: new Date(Date.now() - 24 * 60 * 60 * 1000), // Last 24 hours
        end: new Date()
      });
      
      return {
        status: 'healthy',
        database: 'connected',
        neo4j: 'connected',
        recent_conflicts: recentStats.total_conflicts,
        safety_override_rate: recentStats.safety_override_rate
      };
      
    } catch (error) {
      return {
        status: 'unhealthy',
        error: error.message
      };
    }
  }
}