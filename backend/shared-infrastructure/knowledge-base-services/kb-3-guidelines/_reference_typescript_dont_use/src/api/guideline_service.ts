// KB-3 Guideline Evidence API Service
// Main API service for clinical guideline management with conflict resolution

import { ConflictResolutionEngine, Conflict, Resolution, PatientContext as EnginePatientContext } from '../engines/conflict_resolver';
import { ProductionConflictResolver } from '../engines/production_conflict_resolver';
import { SafetyOverrideEngine } from '../engines/safety_override_engine';
import { MultiLayerCache } from '../services/cache_service';
import { DatabaseService } from '../services/database_service';
import { Neo4jService } from '../services/neo4j_service';
import { AuditLogger } from '../services/audit_logger';

// Type guard for error handling
function isError(error: unknown): error is Error {
  return error instanceof Error;
}

export interface GuidelineQuery {
  condition?: string;
  icd10_codes?: string[];
  region: string;
  patientContext?: PatientContext;
  includeConflicts?: boolean;
  includeSafetyOverrides?: boolean;
}

export interface PatientContext extends EnginePatientContext {
  contraindications: string[];
}

export interface GuidelineResponse {
  guidelines: Guideline[];
  conflicts: Conflict[];
  resolutions: Resolution[];
  safety_overrides: SafetyOverride[];
  metadata: {
    kb_version: string;
    conflict_count: number;
    safety_overrides_applied: boolean;
    query_time_ms: number;
    cache_hit: boolean;
  };
}

export interface ValidationReport {
  total_links: number;
  valid: number;
  broken: Array<{
    source: string;
    target: string;
    error: string;
  }>;
  missing: Array<{
    source: string;
    expected_target: string;
  }>;
}

export class KB3GuidelineService {
  private conflictResolver: ConflictResolutionEngine;
  private safetyEngine: SafetyOverrideEngine;
  private cache: MultiLayerCache;
  private database: DatabaseService;
  private neo4j: Neo4jService;
  private auditLogger: AuditLogger;
  private version: string = '3.0.0';

  constructor(
    database: DatabaseService,
    neo4j: Neo4jService,
    cache: MultiLayerCache,
    auditLogger: AuditLogger
  ) {
    this.database = database;
    this.neo4j = neo4j;
    this.cache = cache;
    this.auditLogger = auditLogger;
    
    this.conflictResolver = new ProductionConflictResolver(database, neo4j, auditLogger);
    this.safetyEngine = new SafetyOverrideEngine(database, auditLogger);
  }

  async initialize(): Promise<void> {
    await this.safetyEngine.initialize();
    await this.cache.preload();
    
    await this.auditLogger.log({
      event_type: 'kb3_service_initialized',
      severity: 'info',
      event_data: { version: this.version }
    });
  }

  async getGuidelines(query: GuidelineQuery): Promise<GuidelineResponse> {
    const startTime = Date.now();
    
    // Check cache first
    const cacheKey = this.generateCacheKey(query);
    const cached = await this.cache.get(cacheKey);
    if (cached) {
      return {
        ...cached,
        metadata: {
          ...cached.metadata,
          cache_hit: true,
          query_time_ms: Date.now() - startTime
        }
      };
    }

    try {
      // Query guidelines from database and graph
      const guidelines = await this.queryGuidelines(query);
      
      let conflicts: Conflict[] = [];
      let resolutions: Resolution[] = [];
      
      // Detect and resolve conflicts if requested
      if (query.includeConflicts !== false) {
        conflicts = await this.conflictResolver.detectConflicts(guidelines);
        
        for (const conflict of conflicts) {
          const resolution = await this.conflictResolver.resolveConflict(
            conflict,
            query.patientContext || {} as PatientContext
          );
          resolutions.push(resolution);
        }
      }

      // Apply safety overrides
      let safety_overrides: SafetyOverride[] = [];
      let safety_applied = false;
      
      if (query.includeSafetyOverrides !== false && query.patientContext) {
        const recommendations = this.extractRecommendations(guidelines);
        const safetyResult = await this.safetyEngine.evaluate(
          recommendations,
          query.patientContext
        );
        
        safety_overrides = safetyResult.overrides || [];
        safety_applied = safetyResult.applied;
      }

      const response: GuidelineResponse = {
        guidelines,
        conflicts,
        resolutions,
        safety_overrides,
        metadata: {
          kb_version: this.version,
          conflict_count: conflicts.length,
          safety_overrides_applied: safety_applied,
          query_time_ms: Date.now() - startTime,
          cache_hit: false
        }
      };

      // Cache the response
      await this.cache.set(cacheKey, response, 1800); // 30 minutes TTL

      // Audit the query
      await this.auditLogger.log({
        event_type: 'guideline_query',
        event_category: 'clinical_operation',
        severity: 'info',
        event_data: {
          query,
          guideline_count: guidelines.length,
          conflict_count: conflicts.length,
          safety_overrides: safety_applied
        }
      });

      return response;
      
    } catch (error) {
      await this.auditLogger.log({
        event_type: 'guideline_query_error',
        event_category: 'system_error',
        severity: 'error',
        event_data: { query, error: error.message }
      });
      throw error;
    }
  }

  async compareGuidelines(
    guideline_ids: string[],
    domain?: string
  ): Promise<GuidelineComparison> {
    const guidelines = await this.getGuidelinesByIds(guideline_ids);
    const conflicts = await this.conflictResolver.detectConflicts(guidelines);
    
    const comparison: GuidelineComparison = {
      guidelines,
      conflicts: conflicts.filter(c => !domain || c.domain === domain),
      differences: await this.analyzeDifferences(guidelines, domain),
      consensus_points: await this.findConsensus(guidelines, domain)
    };

    await this.auditLogger.log({
      event_type: 'guideline_comparison',
      severity: 'info',
      event_data: { guideline_ids, domain, conflict_count: conflicts.length }
    });

    return comparison;
  }

  async getClinicalPathway(
    conditions: string[],
    contraindications: string[],
    region: string,
    patientFactors?: PatientContext
  ): Promise<ClinicalPathway> {
    // Get applicable guidelines for conditions
    const applicableGuidelines = await this.getGuidelinesForConditions(
      conditions,
      region
    );

    // Apply patient-specific filtering
    const filteredGuidelines = await this.filterByContraindications(
      applicableGuidelines,
      contraindications
    );

    // Resolve conflicts
    const conflicts = await this.conflictResolver.detectConflicts(filteredGuidelines);
    const resolutions = [];
    
    for (const conflict of conflicts) {
      const resolution = await this.conflictResolver.resolveConflict(
        conflict,
        patientFactors || {} as PatientContext
      );
      resolutions.push(resolution);
    }

    // Build prioritized recommendations
    const recommendations = await this.prioritizeRecommendations(
      filteredGuidelines,
      resolutions,
      patientFactors
    );

    // Generate decision points
    const decisionPoints = await this.generateDecisionPoints(
      recommendations,
      patientFactors
    );

    return {
      primary_guideline: filteredGuidelines[0],
      recommendations,
      decision_points: decisionPoints,
      conflicts_resolved: resolutions
    };
  }

  async validateCrossKBLinks(): Promise<ValidationReport> {
    const report: ValidationReport = {
      total_links: 0,
      valid: 0,
      broken: [],
      missing: []
    };

    // Get all pending linkages
    const links = await this.database.query(`
      SELECT * FROM guideline_evidence.kb_linkages 
      WHERE validation_status = 'pending'
    `);

    report.total_links = links.rows.length;

    for (const link of links.rows) {
      try {
        const isValid = await this.validateLink(link);
        
        if (isValid) {
          report.valid++;
          await this.markLinkValid(link.id);
        } else {
          report.broken.push({
            source: `${link.source_guideline}:${link.source_rec_id}`,
            target: `${link.target_kb}:${link.target_id}`,
            error: 'Target not found'
          });
          await this.markLinkBroken(link.id, 'Target not found');
        }
      } catch (error) {
        report.broken.push({
          source: `${link.source_guideline}:${link.source_rec_id}`,
          target: `${link.target_kb}:${link.target_id}`,
          error: error.message
        });
      }
    }

    await this.auditLogger.log({
      event_type: 'cross_kb_validation',
      severity: 'info',
      event_data: {
        total_links: report.total_links,
        valid: report.valid,
        broken_count: report.broken.length
      }
    });

    return report;
  }

  private async queryGuidelines(query: GuidelineQuery): Promise<Guideline[]> {
    let sql = `
      SELECT g.*, 
             array_agg(DISTINCT r.rec_id) as recommendation_ids
      FROM guideline_evidence.guidelines g
      LEFT JOIN guideline_evidence.recommendations r ON g.guideline_id = r.guideline_id
      WHERE g.status = 'active'
    `;
    
    const params: any[] = [];
    let paramIndex = 1;

    if (query.condition) {
      sql += ` AND g.condition_primary = $${paramIndex}`;
      params.push(query.condition);
      paramIndex++;
    }

    if (query.icd10_codes && query.icd10_codes.length > 0) {
      sql += ` AND g.icd10_codes && $${paramIndex}`;
      params.push(query.icd10_codes);
      paramIndex++;
    }

    if (query.region) {
      sql += ` AND (g.region = $${paramIndex} OR g.region IN ('Global', 'WHO'))`;
      params.push(query.region);
      paramIndex++;
    }

    sql += ` GROUP BY g.id ORDER BY g.effective_date DESC`;

    const result = await this.database.query(sql, params);
    return result.rows;
  }

  private async getGuidelinesForConditions(
    conditions: string[],
    region: string
  ): Promise<Guideline[]> {
    const guidelines = [];
    
    for (const condition of conditions) {
      const result = await this.queryGuidelines({
        condition,
        region,
        includeConflicts: false
      });
      guidelines.push(...result.guidelines);
    }

    return guidelines;
  }

  private async validateLink(link: any): Promise<boolean> {
    // Check if target KB and target ID exist
    // This would call the specific KB service
    switch (link.target_kb) {
      case 'KB-1':
        return await this.validateKB1Link(link.target_id);
      case 'KB-2':
        return await this.validateKB2Link(link.target_id);
      case 'KB-4':
        return await this.validateKB4Link(link.target_id);
      default:
        return false;
    }
  }

  private async validateKB1Link(targetId: string): Promise<boolean> {
    // Mock validation - would call KB-1 service
    return targetId.startsWith('drug_');
  }

  private async validateKB2Link(targetId: string): Promise<boolean> {
    // Mock validation - would call KB-2 service
    return targetId.startsWith('dose_');
  }

  private async validateKB4Link(targetId: string): Promise<boolean> {
    // Mock validation - would call KB-4 service
    return targetId.startsWith('interaction_');
  }

  private async markLinkValid(linkId: string): Promise<void> {
    await this.database.query(`
      UPDATE guideline_evidence.kb_linkages 
      SET validation_status = 'valid', 
          last_validated = NOW(),
          validation_errors = NULL
      WHERE id = $1
    `, [linkId]);
  }

  private async markLinkBroken(linkId: string, error: string): Promise<void> {
    await this.database.query(`
      UPDATE guideline_evidence.kb_linkages 
      SET validation_status = 'broken',
          last_validated = NOW(),
          validation_errors = $2
      WHERE id = $1
    `, [linkId, JSON.stringify({ error })]);
  }

  private extractRecommendations(guidelines: Guideline[]): Recommendation[] {
    const recommendations = [];
    
    for (const guideline of guidelines) {
      if (guideline.recommendations) {
        recommendations.push(...guideline.recommendations);
      }
    }
    
    return recommendations;
  }

  private generateCacheKey(query: GuidelineQuery): string {
    const keyParts = [
      'guidelines',
      query.condition || 'any',
      query.region || 'global',
      query.icd10_codes?.join(',') || 'none',
      query.includeConflicts ? 'conflicts' : 'no-conflicts',
      query.includeSafetyOverrides ? 'safety' : 'no-safety'
    ];
    
    return keyParts.join(':');
  }

  private async getGuidelinesByIds(ids: string[]): Promise<Guideline[]> {
    const result = await this.database.query(`
      SELECT * FROM guideline_evidence.guidelines 
      WHERE guideline_id = ANY($1) AND status = 'active'
      ORDER BY effective_date DESC
    `, [ids]);
    
    return result.rows;
  }

  private async analyzeDifferences(
    guidelines: Guideline[],
    domain?: string
  ): Promise<any[]> {
    // Compare recommendations across guidelines
    const differences = [];
    
    for (let i = 0; i < guidelines.length; i++) {
      for (let j = i + 1; j < guidelines.length; j++) {
        const diff = await this.compareGuidelinePair(
          guidelines[i],
          guidelines[j],
          domain
        );
        if (diff.hasDifferences) {
          differences.push(diff);
        }
      }
    }
    
    return differences;
  }

  private async compareGuidelinePair(
    g1: Guideline,
    g2: Guideline,
    domain?: string
  ): Promise<any> {
    // Neo4j query to find differences
    const query = `
      MATCH (g1:Guideline {guideline_id: $g1_id})-[:CONTAINS]->(r1:Recommendation)
      MATCH (g2:Guideline {guideline_id: $g2_id})-[:CONTAINS]->(r2:Recommendation)
      WHERE r1.domain = r2.domain
      ${domain ? 'AND r1.domain = $domain' : ''}
      AND r1.statement <> r2.statement
      RETURN r1, r2, r1.domain as domain
    `;

    const params: any = { g1_id: g1.guideline_id, g2_id: g2.guideline_id };
    if (domain) params.domain = domain;

    const result = await this.neo4j.run(query, params);
    
    return {
      guideline1: g1.guideline_id,
      guideline2: g2.guideline_id,
      hasDifferences: result.records.length > 0,
      differences: result.records.map(record => ({
        domain: record.get('domain'),
        recommendation1: record.get('r1').properties,
        recommendation2: record.get('r2').properties
      }))
    };
  }

  private async findConsensus(
    guidelines: Guideline[],
    domain?: string
  ): Promise<any[]> {
    // Find recommendations that are consistent across guidelines
    const consensus = [];
    
    const query = `
      MATCH (g:Guideline)-[:CONTAINS]->(r:Recommendation)
      WHERE g.guideline_id IN $guideline_ids
      ${domain ? 'AND r.domain = $domain' : ''}
      WITH r.domain as domain, r.statement as statement, count(*) as guideline_count
      WHERE guideline_count = $total_guidelines
      RETURN domain, statement, guideline_count
    `;

    const result = await this.neo4j.run(query, {
      guideline_ids: guidelines.map(g => g.guideline_id),
      total_guidelines: guidelines.length,
      ...(domain && { domain })
    });

    for (const record of result.records) {
      consensus.push({
        domain: record.get('domain'),
        statement: record.get('statement'),
        consensus_level: 'unanimous'
      });
    }

    return consensus;
  }

  private async filterByContraindications(
    guidelines: Guideline[],
    contraindications: string[]
  ): Promise<Guideline[]> {
    if (!contraindications.length) return guidelines;

    // Filter out guidelines with contraindicated recommendations
    const filtered = [];
    
    for (const guideline of guidelines) {
      const hasContraindication = guideline.recommendations?.some(rec =>
        contraindications.some(contra => 
          rec.statement.toLowerCase().includes(contra.toLowerCase())
        )
      );
      
      if (!hasContraindication) {
        filtered.push(guideline);
      }
    }

    return filtered;
  }

  private async prioritizeRecommendations(
    guidelines: Guideline[],
    resolutions: Resolution[],
    patientFactors?: PatientContext
  ): Promise<PrioritizedRecommendation[]> {
    const recommendations = this.extractRecommendations(guidelines);
    const prioritized: PrioritizedRecommendation[] = [];

    for (const rec of recommendations) {
      // Calculate priority based on evidence grade, regional preference, etc.
      let priority = this.calculatePriority(rec, patientFactors);
      
      // Adjust based on conflict resolutions
      const relevantResolution = resolutions.find(res =>
        res.affects_recommendation === rec.rec_id
      );
      
      if (relevantResolution) {
        priority = this.adjustPriorityByResolution(priority, relevantResolution);
      }

      prioritized.push({
        recommendation: rec,
        priority,
        rationale: this.generatePriorityRationale(rec, relevantResolution),
        alternatives: await this.findAlternatives(rec)
      });
    }

    return prioritized.sort((a, b) => b.priority - a.priority);
  }

  private calculatePriority(
    rec: Recommendation,
    patientFactors?: PatientContext
  ): number {
    let priority = 50; // Base priority

    // Evidence grade weighting
    switch (rec.evidence_grade) {
      case 'A': priority += 20; break;
      case 'B': priority += 15; break;
      case 'C': priority += 10; break;
      case 'D': priority += 5; break;
      default: priority += 0;
    }

    // Regional preference
    if (patientFactors?.region && rec.region === patientFactors.region) {
      priority += 10;
    }

    // Quality score
    if (rec.quality_score) {
      priority += Math.floor(rec.quality_score / 10);
    }

    return Math.min(100, Math.max(0, priority));
  }

  private adjustPriorityByResolution(
    basePriority: number,
    resolution: Resolution
  ): number {
    if (resolution.safety_override) {
      return resolution.action === 'contraindicate' ? 0 : basePriority;
    }

    // Adjust based on resolution rule
    switch (resolution.rule_used) {
      case 'safety_first': return Math.max(90, basePriority);
      case 'regional_preference': return basePriority + 15;
      case 'evidence_strength': return basePriority + 10;
      case 'publication_recency': return basePriority + 5;
      default: return basePriority;
    }
  }

  private generatePriorityRationale(
    rec: Recommendation,
    resolution?: Resolution
  ): string {
    const factors = [];
    
    factors.push(`Evidence grade ${rec.evidence_grade}`);
    
    if (rec.quality_score) {
      factors.push(`Quality score ${rec.quality_score}`);
    }
    
    if (resolution) {
      factors.push(`Conflict resolved via ${resolution.rule_used}`);
    }

    return `Priority based on: ${factors.join(', ')}`;
  }

  private async findAlternatives(rec: Recommendation): Promise<Recommendation[]> {
    // Find similar recommendations from other guidelines
    const query = `
      MATCH (r:Recommendation {domain: $domain})
      WHERE r.rec_id <> $rec_id
      AND r.statement CONTAINS $key_terms
      RETURN r
      LIMIT 3
    `;

    const keyTerms = this.extractKeyTerms(rec.statement);
    const result = await this.neo4j.run(query, {
      domain: rec.domain,
      rec_id: rec.rec_id,
      key_terms: keyTerms[0] || ''
    });

    return result.records.map(record => record.get('r').properties);
  }

  private extractKeyTerms(statement: string): string[] {
    // Simple keyword extraction - could be enhanced with NLP
    const words = statement.toLowerCase().split(/\s+/);
    const keywords = words.filter(word => 
      word.length > 4 && 
      !['should', 'could', 'would', 'with', 'when', 'where'].includes(word)
    );
    
    return keywords.slice(0, 3);
  }

  private async generateDecisionPoints(
    recommendations: PrioritizedRecommendation[],
    patientFactors?: PatientContext
  ): Promise<DecisionPoint[]> {
    const decisionPoints: DecisionPoint[] = [];

    // Generate decision points based on recommendations
    for (const prioRec of recommendations) {
      const rec = prioRec.recommendation;
      
      // Check for monitoring requirements
      if (rec.statement.includes('monitor')) {
        decisionPoints.push({
          id: `monitor_${rec.rec_id}`,
          question: `Monitor ${this.extractMonitoringTarget(rec.statement)}?`,
          options: [
            { value: 'yes', label: 'Schedule monitoring', next_step: 'monitoring_plan' },
            { value: 'no', label: 'Skip monitoring', next_step: 'risk_acknowledgment' }
          ],
          default_choice: 'yes'
        });
      }

      // Check for dosing decisions
      if (rec.statement.includes('dose') || rec.statement.includes('dosing')) {
        decisionPoints.push({
          id: `dose_${rec.rec_id}`,
          question: `Adjust dose based on patient factors?`,
          options: [
            { value: 'standard', label: 'Standard dose', next_step: 'implementation' },
            { value: 'reduced', label: 'Reduced dose', next_step: 'kb1_dosing_ref' },
            { value: 'custom', label: 'Custom calculation', next_step: 'dosing_calculator' }
          ],
          default_choice: 'standard'
        });
      }
    }

    return decisionPoints;
  }

  private extractMonitoringTarget(statement: string): string {
    // Extract what needs to be monitored
    const monitoringTerms = ['renal function', 'potassium', 'blood pressure', 'liver function'];
    
    for (const term of monitoringTerms) {
      if (statement.toLowerCase().includes(term)) {
        return term;
      }
    }
    
    return 'clinical parameters';
  }

  async getHealthStatus(): Promise<any> {
    try {
      // Check database connectivity
      await this.database.query('SELECT 1');
      
      // Check Neo4j connectivity
      await this.neo4j.run('RETURN 1');
      
      // Check cache
      const cacheHealth = await this.cache.healthCheck();
      
      return {
        status: 'healthy',
        version: this.version,
        database: 'connected',
        neo4j: 'connected',
        cache: cacheHealth,
        timestamp: new Date().toISOString()
      };
      
    } catch (error) {
      return {
        status: 'unhealthy',
        error: error.message,
        timestamp: new Date().toISOString()
      };
    }
  }
}

// Type definitions for response structures
export interface Guideline {
  guideline_id: string;
  organization: string;
  region: string;
  condition_primary: string;
  version: string;
  effective_date: Date;
  recommendations?: Recommendation[];
}

export interface Recommendation {
  rec_id: string;
  domain: string;
  statement: string;
  evidence_grade: string;
  quality_score?: number;
  region?: string;
}

// Using Conflict and Resolution interfaces from conflict_resolver engine

export interface SafetyOverride {
  override_id: string;
  priority: number;
  condition: string;
  action: any;
  rationale: string;
}

export interface GuidelineComparison {
  guidelines: Guideline[];
  conflicts: Conflict[];
  differences: any[];
  consensus_points: any[];
}

export interface ClinicalPathway {
  primary_guideline: Guideline;
  recommendations: PrioritizedRecommendation[];
  decision_points: DecisionPoint[];
  conflicts_resolved: Resolution[];
}

export interface PrioritizedRecommendation {
  recommendation: Recommendation;
  priority: number;
  rationale: string;
  alternatives: Recommendation[];
}

export interface DecisionPoint {
  id: string;
  question: string;
  options: Array<{
    value: string;
    label: string;
    next_step: string;
  }>;
  default_choice: string;
}