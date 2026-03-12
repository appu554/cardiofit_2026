// Neo4j Service for KB-3 Guideline Evidence
// Graph database operations for guideline relationships and conflict detection

import neo4j, { Driver, Session, Result, Record } from 'neo4j-driver';

export interface Neo4jConfig {
  uri: string;
  username: string;
  password: string;
  database?: string;
  maxConnectionPoolSize?: number;
  connectionAcquisitionTimeout?: number;
  maxConnectionLifetime?: number;
}

export class Neo4jService {
  private driver: Driver;
  private config: Neo4jConfig;

  constructor(config: Neo4jConfig) {
    this.config = config;
    this.driver = neo4j.driver(
      config.uri,
      neo4j.auth.basic(config.username, config.password),
      {
        maxConnectionPoolSize: config.maxConnectionPoolSize || 50,
        connectionAcquisitionTimeout: config.connectionAcquisitionTimeout || 60000,
        maxConnectionLifetime: config.maxConnectionLifetime || 3600000, // 1 hour
      }
    );
  }

  async initialize(): Promise<void> {
    try {
      // Test connection
      await this.verifyConnectivity();
      
      // Initialize schema
      await this.initializeSchema();
      
      // Create indexes for performance
      await this.createIndexes();
      
      console.log(`Neo4j connected: ${this.config.uri}`);
    } catch (error) {
      console.error('Neo4j initialization failed:', error);
      throw error;
    }
  }

  async run(query: string, parameters?: Record<string, any>): Promise<Result> {
    const session = this.driver.session({
      database: this.config.database || 'neo4j'
    });
    
    try {
      const result = await session.run(query, parameters);
      return result;
    } catch (error) {
      console.error('Neo4j query error:', error);
      console.error('Query:', query);
      console.error('Parameters:', parameters);
      throw error;
    } finally {
      await session.close();
    }
  }

  async runTransaction<T>(
    work: (tx: any) => Promise<T>
  ): Promise<T> {
    const session = this.driver.session({
      database: this.config.database || 'neo4j'
    });
    
    try {
      const result = await session.executeWrite(work);
      return result;
    } finally {
      await session.close();
    }
  }

  private async verifyConnectivity(): Promise<void> {
    await this.run('RETURN 1 as test');
  }

  private async initializeSchema(): Promise<void> {
    // Create constraints for uniqueness
    const constraints = [
      'CREATE CONSTRAINT guideline_unique IF NOT EXISTS FOR (g:Guideline) REQUIRE g.guideline_id IS UNIQUE',
      'CREATE CONSTRAINT recommendation_unique IF NOT EXISTS FOR (r:Recommendation) REQUIRE r.rec_id IS UNIQUE',
      'CREATE CONSTRAINT conflict_unique IF NOT EXISTS FOR (c:Conflict) REQUIRE c.conflict_id IS UNIQUE',
      'CREATE CONSTRAINT safety_override_unique IF NOT EXISTS FOR (s:SafetyOverride) REQUIRE s.override_id IS UNIQUE'
    ];

    for (const constraint of constraints) {
      try {
        await this.run(constraint);
      } catch (error) {
        // Constraint might already exist
        if (!error.message.includes('already exists')) {
          throw error;
        }
      }
    }
  }

  private async createIndexes(): Promise<void> {
    const indexes = [
      'CREATE INDEX guideline_region_idx IF NOT EXISTS FOR (g:Guideline) ON (g.region)',
      'CREATE INDEX guideline_condition_idx IF NOT EXISTS FOR (g:Guideline) ON (g.condition_primary)',
      'CREATE INDEX guideline_organization_idx IF NOT EXISTS FOR (g:Guideline) ON (g.organization)',
      'CREATE INDEX recommendation_domain_idx IF NOT EXISTS FOR (r:Recommendation) ON (r.domain)',
      'CREATE INDEX recommendation_grade_idx IF NOT EXISTS FOR (r:Recommendation) ON (r.evidence_grade)',
      'CREATE INDEX conflict_type_idx IF NOT EXISTS FOR (c:Conflict) ON (c.type)',
      'CREATE INDEX safety_priority_idx IF NOT EXISTS FOR (s:SafetyOverride) ON (s.priority)'
    ];

    for (const index of indexes) {
      try {
        await this.run(index);
      } catch (error) {
        // Index might already exist
        if (!error.message.includes('already exists')) {
          console.warn('Index creation warning:', error.message);
        }
      }
    }
  }

  // Guideline operations
  async createGuideline(guideline: any): Promise<void> {
    await this.run(`
      MERGE (g:Guideline {guideline_id: $guideline_id})
      SET g += $properties,
          g.updated_at = datetime()
    `, {
      guideline_id: guideline.guideline_id,
      properties: guideline
    });
  }

  async createRecommendation(recommendation: any, guideline_id: string): Promise<void> {
    await this.run(`
      MATCH (g:Guideline {guideline_id: $guideline_id})
      MERGE (r:Recommendation {rec_id: $rec_id})
      SET r += $properties
      MERGE (g)-[:CONTAINS]->(r)
    `, {
      guideline_id,
      rec_id: recommendation.rec_id,
      properties: recommendation
    });
  }

  async createConflictRelationship(
    rec1_id: string,
    rec2_id: string,
    conflict_data: any
  ): Promise<void> {
    await this.run(`
      MATCH (r1:Recommendation {rec_id: $rec1_id})
      MATCH (r2:Recommendation {rec_id: $rec2_id})
      MERGE (r1)-[:CONFLICTS_WITH $conflict_data]->(r2)
    `, {
      rec1_id,
      rec2_id,
      conflict_data
    });
  }

  // Query methods for conflict detection
  async findConflictingRecommendations(guideline_ids: string[]): Promise<Record[]> {
    const result = await this.run(`
      MATCH (g1:Guideline)-[:CONTAINS]->(r1:Recommendation)
      MATCH (g2:Guideline)-[:CONTAINS]->(r2:Recommendation)
      WHERE g1.guideline_id IN $guideline_ids
        AND g2.guideline_id IN $guideline_ids
        AND g1.guideline_id < g2.guideline_id
        AND r1.domain = r2.domain
        AND (
          // Target value conflicts
          (r1.statement =~ '.*target.*' AND r2.statement =~ '.*target.*'
           AND r1.statement <> r2.statement)
          OR
          // Evidence grade mismatches for similar recommendations
          (r1.evidence_grade <> r2.evidence_grade 
           AND r1.statement =~ r2.statement)
          OR
          // First-line treatment conflicts
          (r1.statement =~ '.*first.line.*' AND r2.statement =~ '.*first.line.*'
           AND r1.statement <> r2.statement)
        )
      RETURN g1, r1, g2, r2, r1.domain as domain
    `, { guideline_ids });
    
    return result.records;
  }

  async findGuidelinesByCondition(
    condition: string,
    region?: string
  ): Promise<Record[]> {
    let whereClause = 'WHERE g.condition_primary = $condition';
    const parameters: any = { condition };
    
    if (region) {
      whereClause += ' AND (g.region = $region OR g.region IN ["Global", "WHO"])';
      parameters.region = region;
    }
    
    const result = await this.run(`
      MATCH (g:Guideline)-[:CONTAINS]->(r:Recommendation)
      ${whereClause}
      AND g.status = 'active'
      RETURN g, collect(r) as recommendations
      ORDER BY g.effective_date DESC
    `, parameters);
    
    return result.records;
  }

  async findSupersededGuidelines(): Promise<Record[]> {
    const result = await this.run(`
      MATCH (old:Guideline)-[:SUPERSEDED_BY]->(new:Guideline)
      WHERE old.status = 'superseded'
      RETURN old, new
      ORDER BY old.superseded_date DESC
    `);
    
    return result.records;
  }

  async findCrossKBReferences(guideline_id: string): Promise<Record[]> {
    const result = await this.run(`
      MATCH (g:Guideline {guideline_id: $guideline_id})-[:CONTAINS]->(r:Recommendation)
      MATCH (r)-[:LINKS_TO]->(ext:ExternalReference)
      RETURN r, ext
    `, { guideline_id });
    
    return result.records;
  }

  // Safety override operations
  async createSafetyOverride(override: any): Promise<void> {
    await this.run(`
      MERGE (s:SafetyOverride {override_id: $override_id})
      SET s += $properties,
          s.created_at = datetime()
    `, {
      override_id: override.override_id,
      properties: override
    });
  }

  async linkSafetyOverrideToRecommendations(
    override_id: string,
    recommendation_ids: string[]
  ): Promise<void> {
    for (const rec_id of recommendation_ids) {
      await this.run(`
        MATCH (s:SafetyOverride {override_id: $override_id})
        MATCH (r:Recommendation {rec_id: $rec_id})
        MERGE (s)-[:OVERRIDES]->(r)
      `, { override_id, rec_id });
    }
  }

  async findApplicableSafetyOverrides(
    guideline_ids: string[],
    conditions: string[]
  ): Promise<Record[]> {
    const result = await this.run(`
      MATCH (s:SafetyOverride)-[:OVERRIDES]->(r:Recommendation)
      MATCH (g:Guideline)-[:CONTAINS]->(r)
      WHERE g.guideline_id IN $guideline_ids
      AND s.active = true
      AND ANY(condition IN $conditions WHERE s.condition =~ condition)
      RETURN s, collect(DISTINCT g.guideline_id) as affected_guidelines
      ORDER BY s.priority
    `, { guideline_ids, conditions });
    
    return result.records;
  }

  // Analytics and reporting
  async getConflictAnalytics(timeframe_days: number = 30): Promise<any> {
    const result = await this.run(`
      MATCH (r1:Recommendation)-[conf:CONFLICTS_WITH]-(r2:Recommendation)
      WHERE conf.detected_date >= date() - duration({days: $timeframe_days})
      RETURN 
        conf.type as conflict_type,
        count(*) as conflict_count,
        collect(DISTINCT r1.domain) as affected_domains
      ORDER BY conflict_count DESC
    `, { timeframe_days });
    
    return result.records.map(record => ({
      conflict_type: record.get('conflict_type'),
      count: record.get('conflict_count').toNumber(),
      affected_domains: record.get('affected_domains')
    }));
  }

  async getGuidelineConnectivity(): Promise<any> {
    const result = await this.run(`
      MATCH (g:Guideline)
      OPTIONAL MATCH (g)-[:CONTAINS]->(r:Recommendation)
      OPTIONAL MATCH (r)-[:CONFLICTS_WITH]-(conflicted)
      RETURN 
        g.guideline_id as guideline_id,
        count(DISTINCT r) as recommendation_count,
        count(DISTINCT conflicted) as conflict_count
      ORDER BY conflict_count DESC
    `);
    
    return result.records.map(record => ({
      guideline_id: record.get('guideline_id'),
      recommendation_count: record.get('recommendation_count').toNumber(),
      conflict_count: record.get('conflict_count').toNumber()
    }));
  }

  async getEvidenceQualityDistribution(): Promise<any> {
    const result = await this.run(`
      MATCH (r:Recommendation)
      RETURN 
        r.evidence_grade as grade,
        count(*) as count,
        avg(r.quality_score) as avg_quality_score
      ORDER BY 
        CASE r.evidence_grade
          WHEN 'A' THEN 1
          WHEN 'B' THEN 2  
          WHEN 'C' THEN 3
          WHEN 'D' THEN 4
          ELSE 5
        END
    `);
    
    return result.records.map(record => ({
      grade: record.get('grade'),
      count: record.get('count').toNumber(),
      avg_quality: record.get('avg_quality_score')?.toNumber() || 0
    }));
  }

  // Bulk operations for performance
  async bulkCreateGuidelines(guidelines: any[]): Promise<void> {
    const session = this.driver.session({ database: this.config.database || 'neo4j' });
    
    try {
      await session.executeWrite(async tx => {
        for (const guideline of guidelines) {
          await tx.run(`
            MERGE (g:Guideline {guideline_id: $guideline_id})
            SET g += $properties
          `, {
            guideline_id: guideline.guideline_id,
            properties: guideline
          });
          
          // Create recommendations
          if (guideline.recommendations) {
            for (const rec of guideline.recommendations) {
              await tx.run(`
                MATCH (g:Guideline {guideline_id: $guideline_id})
                MERGE (r:Recommendation {rec_id: $rec_id})
                SET r += $rec_properties
                MERGE (g)-[:CONTAINS]->(r)
              `, {
                guideline_id: guideline.guideline_id,
                rec_id: rec.rec_id,
                rec_properties: rec
              });
            }
          }
        }
      });
    } finally {
      await session.close();
    }
  }

  async bulkCreateConflicts(conflicts: any[]): Promise<void> {
    const session = this.driver.session({ database: this.config.database || 'neo4j' });
    
    try {
      await session.executeWrite(async tx => {
        for (const conflict of conflicts) {
          await tx.run(`
            MATCH (r1:Recommendation {rec_id: $rec1_id})
            MATCH (r2:Recommendation {rec_id: $rec2_id})
            MERGE (r1)-[:CONFLICTS_WITH $conflict_props]->(r2)
          `, {
            rec1_id: conflict.recommendation1_id,
            rec2_id: conflict.recommendation2_id,
            conflict_props: {
              type: conflict.type,
              severity: conflict.severity,
              detected_date: neo4j.types.Date.fromStandardDate(new Date())
            }
          });
        }
      });
    } finally {
      await session.close();
    }
  }

  // Specialized queries for clinical workflows
  async getClinicalPathway(
    conditions: string[],
    region: string,
    patientFactors?: any
  ): Promise<Record[]> {
    const result = await this.run(`
      MATCH (g:Guideline)-[:CONTAINS]->(r:Recommendation)
      WHERE ANY(condition IN $conditions WHERE g.condition_primary =~ condition)
      AND (g.region = $region OR g.region IN ['Global', 'WHO'])
      AND g.status = 'active'
      
      // Check for applicable safety overrides
      OPTIONAL MATCH (s:SafetyOverride)-[:OVERRIDES]->(r)
      WHERE s.active = true
      
      // Get conflicts
      OPTIONAL MATCH (r)-[conf:CONFLICTS_WITH]-(other_r:Recommendation)
      
      RETURN g, r, s, collect(DISTINCT {
        conflict_rec: other_r,
        conflict_type: conf.type
      }) as conflicts
      
      ORDER BY 
        CASE r.evidence_grade
          WHEN 'A' THEN 1
          WHEN 'B' THEN 2
          WHEN 'C' THEN 3
          WHEN 'D' THEN 4
          ELSE 5
        END,
        g.effective_date DESC
    `, { conditions, region });
    
    return result.records;
  }

  async findGuidelineConflicts(guideline_ids: string[]): Promise<Record[]> {
    const result = await this.run(`
      MATCH (g1:Guideline {guideline_id: $g1_id})-[:CONTAINS]->(r1:Recommendation)
      MATCH (g2:Guideline {guideline_id: $g2_id})-[:CONTAINS]->(r2:Recommendation)
      MATCH (r1)-[conf:CONFLICTS_WITH]-(r2)
      RETURN 
        g1.guideline_id as guideline1,
        g2.guideline_id as guideline2,
        r1, r2, conf.type as conflict_type,
        conf.severity as severity
    `, { g1_id: guideline_ids[0], g2_id: guideline_ids[1] });
    
    return result.records;
  }

  async getGuidelineEvolution(guideline_family: string): Promise<Record[]> {
    const result = await this.run(`
      MATCH path = (old:Guideline)-[:SUPERSEDED_BY*]->(current:Guideline)
      WHERE old.guideline_id =~ $pattern
      AND current.status = 'active'
      RETURN nodes(path) as evolution_path,
             length(path) as version_count
      ORDER BY version_count DESC
    `, { pattern: `${guideline_family}.*` });
    
    return result.records;
  }

  // Performance and maintenance
  async getPerformanceStats(): Promise<any> {
    const result = await this.run(`
      CALL db.stats.retrieve('GRAPH COUNTS') YIELD data
      RETURN data
    `);
    
    const counts = result.records[0]?.get('data') || {};
    
    // Get relationship counts
    const relResult = await this.run(`
      MATCH ()-[r]->()
      RETURN type(r) as relationship_type, count(r) as count
      ORDER BY count DESC
    `);
    
    return {
      node_count: counts.nodes || 0,
      relationship_count: counts.relationships || 0,
      relationship_breakdown: relResult.records.map(r => ({
        type: r.get('relationship_type'),
        count: r.get('count').toNumber()
      }))
    };
  }

  async vacuum(): Promise<void> {
    // Clean up orphaned nodes and relationships
    await this.run(`
      MATCH (r:Recommendation)
      WHERE NOT (r)<-[:CONTAINS]-()
      DELETE r
    `);
    
    await this.run(`
      MATCH (c:Conflict)
      WHERE NOT (c)-[:INVOLVES]-()
      DELETE c
    `);
  }

  async healthCheck(): Promise<any> {
    try {
      const start = Date.now();
      await this.run('RETURN 1 as health_check');
      const latency = Date.now() - start;
      
      const stats = await this.getPerformanceStats();
      
      return {
        status: 'healthy',
        latency_ms: latency,
        node_count: stats.node_count,
        relationship_count: stats.relationship_count
      };
    } catch (error) {
      return {
        status: 'unhealthy',
        error: error.message
      };
    }
  }

  async close(): Promise<void> {
    await this.driver.close();
    console.log('Neo4j connection closed');
  }

  // Backup and export utilities
  async exportGuidelines(format: 'json' | 'cypher' = 'json'): Promise<any> {
    if (format === 'cypher') {
      return await this.exportToCypher();
    }
    
    const result = await this.run(`
      MATCH (g:Guideline)-[:CONTAINS]->(r:Recommendation)
      OPTIONAL MATCH (r)-[conf:CONFLICTS_WITH]-(other_r)
      RETURN g, collect(DISTINCT r) as recommendations,
             collect(DISTINCT {
               rec_id: other_r.rec_id,
               conflict_type: conf.type
             }) as conflicts
    `);
    
    return result.records.map(record => ({
      guideline: record.get('g').properties,
      recommendations: record.get('recommendations').map((r: any) => r.properties),
      conflicts: record.get('conflicts').filter((c: any) => c.rec_id)
    }));
  }

  private async exportToCypher(): Promise<string> {
    const guidelines = await this.run(`
      MATCH (g:Guideline)
      RETURN g
    `);
    
    const recommendations = await this.run(`
      MATCH (g:Guideline)-[:CONTAINS]->(r:Recommendation)
      RETURN g.guideline_id as guideline_id, r
    `);
    
    const conflicts = await this.run(`
      MATCH (r1:Recommendation)-[conf:CONFLICTS_WITH]->(r2:Recommendation)
      RETURN r1.rec_id as rec1_id, r2.rec_id as rec2_id, conf
    `);
    
    let cypher = '// KB-3 Guideline Evidence Export\n\n';
    
    // Export guidelines
    for (const record of guidelines.records) {
      const props = record.get('g').properties;
      cypher += `MERGE (g:Guideline ${JSON.stringify(props)})\n`;
    }
    
    cypher += '\n';
    
    // Export recommendations and relationships
    for (const record of recommendations.records) {
      const guideline_id = record.get('guideline_id');
      const rec_props = record.get('r').properties;
      
      cypher += `MATCH (g:Guideline {guideline_id: "${guideline_id}"})\n`;
      cypher += `MERGE (r:Recommendation ${JSON.stringify(rec_props)})\n`;
      cypher += `MERGE (g)-[:CONTAINS]->(r)\n\n`;
    }
    
    // Export conflicts
    for (const record of conflicts.records) {
      const rec1_id = record.get('rec1_id');
      const rec2_id = record.get('rec2_id');
      const conf_props = record.get('conf').properties;
      
      cypher += `MATCH (r1:Recommendation {rec_id: "${rec1_id}"})\n`;
      cypher += `MATCH (r2:Recommendation {rec_id: "${rec2_id}"})\n`;
      cypher += `MERGE (r1)-[:CONFLICTS_WITH ${JSON.stringify(conf_props)}]->(r2)\n\n`;
    }
    
    return cypher;
  }
}