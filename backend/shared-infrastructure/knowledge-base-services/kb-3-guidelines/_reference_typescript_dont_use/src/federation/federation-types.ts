// Apollo Federation Context and Type Definitions for KB-3

import type { KB3GuidelineService } from '../api/guideline_service';
import type { DatabaseService } from '../services/database_service';
import type { Neo4jService } from '../services/neo4j_service';
import type { MultiLayerCache } from '../services/cache_service';
import type { AuditLogger } from '../services/audit_logger';
import type { ProductionConflictResolver } from '../engines/production_conflict_resolver';
import type { SafetyOverrideEngine } from '../engines/safety_override_engine';

// Federation context passed to all resolvers
export interface KB3Context {
  // Service dependencies
  guidelineService: KB3GuidelineService;
  databaseService: DatabaseService;
  neo4jService: Neo4jService;
  cacheService: MultiLayerCache;
  conflictResolver: ProductionConflictResolver;
  safetyEngine: SafetyOverrideEngine;
  
  // Request context
  user: {
    id?: string;
    authorization?: string;
  };
  
  // Patient context for clinical operations
  patient: {
    id?: string;
  };
  
  // Audit tracking
  audit: {
    requestId: string;
    timestamp: Date;
    source: string;
  };
  
  // Performance tracking
  performance: {
    startTime: number;
    queries: Array<{
      operation: string;
      duration: number;
      cached: boolean;
    }>;
  };
}

// Federation entity representation interfaces
export interface FederationPatient {
  __typename: 'Patient';
  id: string;
}

export interface FederationMedication {
  __typename: 'Medication';
  id: string;
}

export interface FederationObservation {
  __typename: 'Observation';
  id: string;
}

// Entity reference resolvers input types
export interface PatientReference {
  __typename: 'Patient';
  id: string;
}

export interface MedicationReference {
  __typename: 'Medication';  
  id: string;
}

export interface ObservationReference {
  __typename: 'Observation';
  id: string;
}

// Federation-specific query inputs
export interface FederationGuidelineQuery {
  conditions: string[];
  region: string;
  patient_factors?: any;
  federation_context?: {
    patient_id?: string;
    requesting_service?: string;
  };
}

export interface FederationClinicalPathwayInput {
  conditions: string[];
  contraindications?: string[];
  region?: string;
  patient_factors?: any;
  federation_context?: {
    patient_id: string;
    medications?: string[];
    observations?: string[];
  };
}

// Performance tracking helpers
export class FederationPerformanceTracker {
  constructor(private context: KB3Context) {}

  trackQuery(operation: string, startTime: number, cached: boolean = false): void {
    const duration = Date.now() - startTime;
    this.context.performance.queries.push({
      operation,
      duration,
      cached
    });
  }

  getMetrics(): {
    total_queries: number;
    total_duration: number;
    average_duration: number;
    cache_hit_rate: number;
    session_duration: number;
  } {
    const queries = this.context.performance.queries;
    const totalDuration = queries.reduce((sum, q) => sum + q.duration, 0);
    const cachedQueries = queries.filter(q => q.cached).length;
    
    return {
      total_queries: queries.length,
      total_duration: totalDuration,
      average_duration: queries.length > 0 ? totalDuration / queries.length : 0,
      cache_hit_rate: queries.length > 0 ? cachedQueries / queries.length : 0,
      session_duration: Date.now() - this.context.performance.startTime
    };
  }
}

// Audit helpers for federation operations
export class FederationAuditLogger {
  constructor(private context: KB3Context) {}

  async logEntityResolution(entityType: string, entityId: string, resolved: boolean): Promise<void> {
    const auditEntry = {
      operation: 'entity_resolution',
      entity_type: entityType,
      entity_id: entityId,
      resolved: resolved,
      request_id: this.context.audit.requestId,
      user_id: this.context.user.id,
      timestamp: new Date(),
      source: 'federation'
    };

    // Log to audit system (implementation would depend on your audit infrastructure)
    console.log('Federation Entity Resolution:', auditEntry);
  }

  async logFederationQuery(operation: string, variables: any, success: boolean): Promise<void> {
    const auditEntry = {
      operation: 'federation_query',
      graphql_operation: operation,
      variables: variables,
      success: success,
      request_id: this.context.audit.requestId,
      user_id: this.context.user.id,
      timestamp: new Date(),
      source: 'federation'
    };

    console.log('Federation Query:', auditEntry);
  }
}

// Error handling for federation operations
export class FederationError extends Error {
  constructor(
    message: string,
    public code: string,
    public service: string = 'kb3-guidelines',
    public operation?: string
  ) {
    super(message);
    this.name = 'FederationError';
  }

  toGraphQLError() {
    return {
      message: this.message,
      extensions: {
        code: this.code,
        service: this.service,
        operation: this.operation,
        timestamp: new Date().toISOString()
      }
    };
  }
}