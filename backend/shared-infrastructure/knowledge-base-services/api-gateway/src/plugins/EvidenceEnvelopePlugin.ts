import { ApolloServerPlugin, GraphQLRequestListener } from '@apollo/server';
import { GraphQLRequestContext } from '@apollo/server';
import crypto from 'crypto';
import { v4 as uuidv4 } from 'uuid';
import { DatabaseManager } from '../database/DatabaseManager';

interface EvidenceEnvelope {
  id: string;
  transactionId: string;
  versionSetId: string;
  kbVersions: Record<string, string>;
  decisionChain: DecisionNode[];
  kbResponses: KBResponse[];
  safetyAttestations: SafetyAttestation[];
  performanceMetrics: PerformanceMetrics;
  patientId?: string;
  encounterId?: string;
  clinicalDomain?: string;
  requestType?: string;
  orchestratorVersion?: string;
  orchestratorNode?: string;
  startTime: Date;
  endTime?: Date;
  totalDurationMs?: number;
  checksum?: string;
  signed: boolean;
  signature?: string;
  metadata: Record<string, any>;
}

interface DecisionNode {
  phase: string;
  timestamp: Date;
  input: any;
  output: any;
  kbCalls: string[];
  durationMs: number;
  confidence?: number;
}

interface KBResponse {
  kb: string;
  version: string;
  digest?: string;
  latency: number;
  cacheHit: boolean;
  responseSize?: number;
  errorCount?: number;
  timestamp: Date;
}

interface SafetyAttestation {
  type: string;
  result: string;
  confidence: number;
  evidence: any[];
  reviewer?: string;
  timestamp: Date;
}

interface PerformanceMetrics {
  totalRequests: number;
  totalLatencyMs: number;
  avgLatencyMs: number;
  maxLatencyMs: number;
  cacheHitRate: number;
  errorRate: number;
}

export class EvidenceEnvelopePlugin implements ApolloServerPlugin {
  private dbManager: DatabaseManager;

  constructor(dbManager: DatabaseManager) {
    this.dbManager = dbManager;
  }

  async requestDidStart(): Promise<GraphQLRequestListener<any>> {
    return {
      async willSendResponse(requestContext: GraphQLRequestContext<any>) {
        await this.finalizeEvidenceEnvelope(requestContext);
      },

      async didEncounterErrors(requestContext: GraphQLRequestContext<any>) {
        await this.handleEnvelopeErrors(requestContext);
      }
    };
  }

  private async finalizeEvidenceEnvelope(
    requestContext: GraphQLRequestContext<any>
  ): Promise<void> {
    const { context } = requestContext;
    
    if (!context.evidenceEnvelope) {
      // Initialize evidence envelope
      context.evidenceEnvelope = await this.initializeEvidenceEnvelope(context);
    }

    const envelope = context.evidenceEnvelope;
    
    try {
      // Set completion time
      envelope.endTime = new Date();
      envelope.totalDurationMs = envelope.endTime.getTime() - envelope.startTime.getTime();

      // Calculate performance metrics
      envelope.performanceMetrics = this.calculatePerformanceMetrics(envelope);

      // Generate checksum for integrity
      envelope.checksum = this.calculateChecksum(envelope);

      // Sign envelope if required
      if (process.env.REQUIRE_EVIDENCE_SIGNING === 'true') {
        envelope.signature = await this.signEnvelope(envelope);
        envelope.signed = true;
      }

      // Persist complete envelope
      await this.persistEvidenceEnvelope(envelope);

      // Add envelope ID to response headers
      if (requestContext.response && requestContext.response.http) {
        requestContext.response.http.headers.set(
          'x-evidence-envelope-id',
          envelope.id
        );
        requestContext.response.http.headers.set(
          'x-transaction-id',
          envelope.transactionId
        );
        requestContext.response.http.headers.set(
          'x-version-set-id',
          envelope.versionSetId
        );
      }

      // Log completion
      console.log('Evidence envelope finalized', {
        envelopeId: envelope.id,
        transactionId: envelope.transactionId,
        duration: envelope.totalDurationMs,
        kbCallCount: envelope.kbResponses.length,
        hasErrors: requestContext.response?.errors && requestContext.response.errors.length > 0
      });

    } catch (error) {
      console.error('Failed to finalize evidence envelope', {
        envelopeId: envelope.id,
        error: error.message
      });
      
      // Still try to save what we have
      try {
        envelope.metadata.finalizationError = error.message;
        await this.persistEvidenceEnvelope(envelope);
      } catch (persistError) {
        console.error('Failed to persist envelope with error', {
          envelopeId: envelope.id,
          persistError: persistError.message
        });
      }
    }
  }

  private async initializeEvidenceEnvelope(context: any): Promise<EvidenceEnvelope> {
    const envelope: EvidenceEnvelope = {
      id: context.evidenceEnvelopeId || `env_${uuidv4().replace(/-/g, '')}`,
      transactionId: context.transactionId,
      versionSetId: context.versionSet?.id || 'default',
      kbVersions: context.versionSet?.kb_versions || {},
      decisionChain: [],
      kbResponses: [],
      safetyAttestations: [],
      performanceMetrics: {
        totalRequests: 0,
        totalLatencyMs: 0,
        avgLatencyMs: 0,
        maxLatencyMs: 0,
        cacheHitRate: 0,
        errorRate: 0
      },
      
      // Extract clinical context from request if available
      patientId: this.extractFromRequest(context, 'patientId'),
      encounterId: this.extractFromRequest(context, 'encounterId'),
      clinicalDomain: this.extractFromRequest(context, 'clinicalDomain'),
      requestType: this.extractFromRequest(context, 'requestType'),
      
      orchestratorVersion: process.env.SERVICE_VERSION || '1.0.0',
      orchestratorNode: process.env.HOSTNAME || 'api-gateway',
      
      startTime: new Date(),
      signed: false,
      
      metadata: {
        userAgent: context.userAgent,
        ipAddress: context.ipAddress,
        userId: context.userContext?.userId,
        requestId: context.requestId
      }
    };

    // Add methods to envelope for tracking
    envelope.addKBResponse = (response: KBResponse) => {
      envelope.kbResponses.push({
        ...response,
        timestamp: new Date()
      });
    };

    envelope.addDecisionNode = (node: DecisionNode) => {
      envelope.decisionChain.push(node);
    };

    envelope.addSafetyAttestation = (attestation: SafetyAttestation) => {
      envelope.safetyAttestations.push({
        ...attestation,
        timestamp: new Date()
      });
    };

    // Store initial envelope
    await this.persistEvidenceEnvelope(envelope);

    return envelope;
  }

  private extractFromRequest(context: any, field: string): any {
    // Try to extract from GraphQL variables
    const variables = context.request?.variables;
    if (variables) {
      if (variables[field]) return variables[field];
      if (variables.input && variables.input[field]) return variables.input[field];
      if (variables.context && variables.context[field]) return variables.context[field];
    }
    
    // Try to extract from headers
    const headerKey = `x-${field.toLowerCase().replace(/([A-Z])/g, '-$1')}`;
    return context.request?.http?.headers?.get(headerKey);
  }

  private calculatePerformanceMetrics(envelope: EvidenceEnvelope): PerformanceMetrics {
    const responses = envelope.kbResponses;
    
    if (responses.length === 0) {
      return {
        totalRequests: 0,
        totalLatencyMs: 0,
        avgLatencyMs: 0,
        maxLatencyMs: 0,
        cacheHitRate: 0,
        errorRate: 0
      };
    }

    const totalLatency = responses.reduce((sum, r) => sum + r.latency, 0);
    const maxLatency = Math.max(...responses.map(r => r.latency));
    const cacheHits = responses.filter(r => r.cacheHit).length;
    const errorCount = responses.reduce((sum, r) => sum + (r.errorCount || 0), 0);

    return {
      totalRequests: responses.length,
      totalLatencyMs: totalLatency,
      avgLatencyMs: Math.round(totalLatency / responses.length),
      maxLatencyMs: maxLatency,
      cacheHitRate: cacheHits / responses.length,
      errorRate: errorCount / responses.length
    };
  }

  private calculateChecksum(envelope: EvidenceEnvelope): string {
    // Create deterministic representation for checksumming
    const checksumData = {
      transactionId: envelope.transactionId,
      kbVersions: envelope.kbVersions,
      decisionChain: envelope.decisionChain.map(node => ({
        phase: node.phase,
        timestamp: node.timestamp.toISOString(),
        kbCalls: node.kbCalls.sort()
      })),
      kbResponses: envelope.kbResponses.map(response => ({
        kb: response.kb,
        version: response.version,
        digest: response.digest,
        latency: response.latency
      })),
      safetyAttestations: envelope.safetyAttestations
    };

    const jsonString = JSON.stringify(checksumData, null, 0);
    return crypto.createHash('sha256').update(jsonString).digest('hex');
  }

  private async signEnvelope(envelope: EvidenceEnvelope): Promise<string> {
    // TODO: Implement digital signature
    // This would use a private key to sign the envelope checksum
    // For now, return a placeholder
    const data = `${envelope.id}:${envelope.checksum}:${envelope.endTime?.toISOString()}`;
    return crypto.createHash('sha256').update(data).digest('hex');
  }

  private async persistEvidenceEnvelope(envelope: EvidenceEnvelope): Promise<void> {
    try {
      const query = `
        INSERT INTO evidence_envelopes (
          id, transaction_id, version_set_id, kb_versions, decision_chain,
          safety_attestations, performance_metrics, patient_id, encounter_id,
          clinical_domain, request_type, orchestrator_version, orchestrator_node,
          started_at, completed_at, total_duration_ms, checksum, signed,
          signature, created_at
        ) VALUES (
          $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16,
          $17, $18, $19, NOW()
        )
        ON CONFLICT (id) 
        DO UPDATE SET
          decision_chain = EXCLUDED.decision_chain,
          safety_attestations = EXCLUDED.safety_attestations,
          performance_metrics = EXCLUDED.performance_metrics,
          completed_at = EXCLUDED.completed_at,
          total_duration_ms = EXCLUDED.total_duration_ms,
          checksum = EXCLUDED.checksum,
          signed = EXCLUDED.signed,
          signature = EXCLUDED.signature
      `;

      const values = [
        envelope.id,
        envelope.transactionId,
        envelope.versionSetId,
        JSON.stringify(envelope.kbVersions),
        JSON.stringify(envelope.decisionChain),
        JSON.stringify(envelope.safetyAttestations),
        JSON.stringify(envelope.performanceMetrics),
        envelope.patientId,
        envelope.encounterId,
        envelope.clinicalDomain,
        envelope.requestType,
        envelope.orchestratorVersion,
        envelope.orchestratorNode,
        envelope.startTime,
        envelope.endTime,
        envelope.totalDurationMs,
        envelope.checksum,
        envelope.signed,
        envelope.signature
      ];

      await this.dbManager.query(query, values);

      // Also store KB responses in separate table for analytics
      for (const response of envelope.kbResponses) {
        await this.storeKBResponse(envelope.id, response);
      }

    } catch (error) {
      console.error('Failed to persist evidence envelope', {
        envelopeId: envelope.id,
        error: error.message
      });
      throw error;
    }
  }

  private async storeKBResponse(envelopeId: string, response: KBResponse): Promise<void> {
    const query = `
      INSERT INTO kb_response_log (
        envelope_id, kb_name, kb_version, latency_ms, cache_hit,
        response_size, error_count, timestamp
      ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `;

    const values = [
      envelopeId,
      response.kb,
      response.version,
      response.latency,
      response.cacheHit,
      response.responseSize,
      response.errorCount || 0,
      response.timestamp
    ];

    await this.dbManager.query(query, values);
  }

  private async handleEnvelopeErrors(
    requestContext: GraphQLRequestContext<any>
  ): Promise<void> {
    const { context } = requestContext;
    
    if (context.evidenceEnvelope) {
      // Record errors in envelope
      const errors = requestContext.errors || [];
      context.evidenceEnvelope.metadata.errors = errors.map(error => ({
        message: error.message,
        locations: error.locations,
        path: error.path,
        timestamp: new Date().toISOString()
      }));

      // Still try to persist the envelope with error information
      try {
        await this.persistEvidenceEnvelope(context.evidenceEnvelope);
      } catch (persistError) {
        console.error('Failed to persist envelope with errors', {
          envelopeId: context.evidenceEnvelope.id,
          persistError: persistError.message
        });
      }
    }
  }
}