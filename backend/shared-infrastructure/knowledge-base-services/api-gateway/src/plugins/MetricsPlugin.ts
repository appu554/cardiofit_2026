import { ApolloServerPlugin, GraphQLRequestListener } from '@apollo/server';
import { GraphQLRequestContext } from '@apollo/server';
import { createLogger, Logger } from 'winston';

interface MetricsCollector {
  // Request metrics
  incrementRequestCount(operationType: string, operationName?: string): void;
  recordRequestDuration(duration: number, operationType: string, operationName?: string): void;
  recordRequestError(error: string, operationType: string, operationName?: string): void;
  
  // KB service metrics
  recordKBServiceCall(kbName: string, duration: number, success: boolean): void;
  recordCacheHitRate(kbName: string, hitRate: number): void;
  
  // Version metrics
  recordVersionSetUsage(versionSetId: string): void;
  recordVersionOverride(fromVersion: string, toVersion: string): void;
  
  // Performance metrics
  recordResponseSize(size: number): void;
  recordEvidenceEnvelopeSize(size: number): void;
  
  // Export metrics for monitoring systems
  exportMetrics(): any;
}

class PrometheusMetricsCollector implements MetricsCollector {
  private metrics: Map<string, any> = new Map();
  private counters: Map<string, number> = new Map();
  private histograms: Map<string, number[]> = new Map();
  private logger: Logger;

  constructor() {
    this.logger = createLogger({
      defaultMeta: { service: 'metrics-collector' }
    });
  }

  incrementRequestCount(operationType: string, operationName?: string): void {
    const key = `graphql_requests_total{operation_type="${operationType}",operation_name="${operationName || 'unnamed'}"}`;
    this.counters.set(key, (this.counters.get(key) || 0) + 1);
  }

  recordRequestDuration(duration: number, operationType: string, operationName?: string): void {
    const key = `graphql_request_duration_seconds{operation_type="${operationType}",operation_name="${operationName || 'unnamed'}"}`;
    
    if (!this.histograms.has(key)) {
      this.histograms.set(key, []);
    }
    
    this.histograms.get(key)!.push(duration / 1000); // Convert ms to seconds
    
    // Keep only last 1000 measurements
    const histogram = this.histograms.get(key)!;
    if (histogram.length > 1000) {
      histogram.shift();
    }
  }

  recordRequestError(error: string, operationType: string, operationName?: string): void {
    const key = `graphql_request_errors_total{operation_type="${operationType}",operation_name="${operationName || 'unnamed'}",error_type="${this.categorizeError(error)}"}`;
    this.counters.set(key, (this.counters.get(key) || 0) + 1);
  }

  recordKBServiceCall(kbName: string, duration: number, success: boolean): void {
    // Call count
    const callCountKey = `kb_service_calls_total{kb_name="${kbName}",success="${success}"}`;
    this.counters.set(callCountKey, (this.counters.get(callCountKey) || 0) + 1);
    
    // Duration
    const durationKey = `kb_service_duration_seconds{kb_name="${kbName}"}`;
    if (!this.histograms.has(durationKey)) {
      this.histograms.set(durationKey, []);
    }
    
    this.histograms.get(durationKey)!.push(duration / 1000);
    
    // Keep only last 1000 measurements
    const histogram = this.histograms.get(durationKey)!;
    if (histogram.length > 1000) {
      histogram.shift();
    }
  }

  recordCacheHitRate(kbName: string, hitRate: number): void {
    const key = `kb_cache_hit_rate{kb_name="${kbName}"}`;
    this.metrics.set(key, hitRate);
  }

  recordVersionSetUsage(versionSetId: string): void {
    const key = `kb_version_set_usage_total{version_set_id="${versionSetId}"}`;
    this.counters.set(key, (this.counters.get(key) || 0) + 1);
  }

  recordVersionOverride(fromVersion: string, toVersion: string): void {
    const key = `kb_version_overrides_total{from_version="${fromVersion}",to_version="${toVersion}"}`;
    this.counters.set(key, (this.counters.get(key) || 0) + 1);
  }

  recordResponseSize(size: number): void {
    const key = 'graphql_response_size_bytes';
    if (!this.histograms.has(key)) {
      this.histograms.set(key, []);
    }
    
    this.histograms.get(key)!.push(size);
    
    // Keep only last 1000 measurements
    const histogram = this.histograms.get(key)!;
    if (histogram.length > 1000) {
      histogram.shift();
    }
  }

  recordEvidenceEnvelopeSize(size: number): void {
    const key = 'evidence_envelope_size_bytes';
    if (!this.histograms.has(key)) {
      this.histograms.set(key, []);
    }
    
    this.histograms.get(key)!.push(size);
    
    // Keep only last 1000 measurements
    const histogram = this.histograms.get(key)!;
    if (histogram.length > 1000) {
      histogram.shift();
    }
  }

  exportMetrics(): any {
    const exported: any = {
      counters: {},
      histograms: {},
      gauges: {},
      timestamp: new Date().toISOString()
    };

    // Export counters
    for (const [key, value] of this.counters.entries()) {
      exported.counters[key] = value;
    }

    // Export histograms with percentiles
    for (const [key, values] of this.histograms.entries()) {
      if (values.length > 0) {
        const sorted = [...values].sort((a, b) => a - b);
        
        exported.histograms[key] = {
          count: values.length,
          sum: values.reduce((a, b) => a + b, 0),
          avg: values.reduce((a, b) => a + b, 0) / values.length,
          min: sorted[0],
          max: sorted[sorted.length - 1],
          p50: this.percentile(sorted, 0.5),
          p95: this.percentile(sorted, 0.95),
          p99: this.percentile(sorted, 0.99)
        };
      }
    }

    // Export current gauge values
    for (const [key, value] of this.metrics.entries()) {
      exported.gauges[key] = value;
    }

    return exported;
  }

  private categorizeError(error: string): string {
    const errorLower = error.toLowerCase();
    
    if (errorLower.includes('validation')) return 'validation';
    if (errorLower.includes('authentication')) return 'authentication';
    if (errorLower.includes('authorization')) return 'authorization';
    if (errorLower.includes('timeout')) return 'timeout';
    if (errorLower.includes('network') || errorLower.includes('connection')) return 'network';
    if (errorLower.includes('version')) return 'version';
    if (errorLower.includes('circuit breaker')) return 'circuit_breaker';
    
    return 'other';
  }

  private percentile(sorted: number[], p: number): number {
    const index = Math.ceil(sorted.length * p) - 1;
    return sorted[index] || 0;
  }
}

export class MetricsPlugin implements ApolloServerPlugin {
  private metricsCollector: MetricsCollector;
  private logger: Logger;

  constructor(metricsCollector?: MetricsCollector) {
    this.metricsCollector = metricsCollector || new PrometheusMetricsCollector();
    this.logger = createLogger({
      defaultMeta: { service: 'metrics-plugin' }
    });
  }

  async requestDidStart(): Promise<GraphQLRequestListener<any>> {
    const startTime = Date.now();

    return {
      async didResolveOperation(requestContext: GraphQLRequestContext<any>) {
        this.recordRequestStart(requestContext);
      },

      async willSendResponse(requestContext: GraphQLRequestContext<any>) {
        await this.recordRequestComplete(requestContext, startTime);
      },

      async didEncounterErrors(requestContext: GraphQLRequestContext<any>) {
        this.recordRequestErrors(requestContext, startTime);
      }
    };
  }

  private recordRequestStart(requestContext: GraphQLRequestContext<any>): void {
    const { request } = requestContext;
    
    try {
      const operationType = this.extractOperationType(request);
      const operationName = request.operationName;

      this.metricsCollector.incrementRequestCount(operationType, operationName);

      this.logger.debug('Request metrics recorded', {
        operationType,
        operationName,
        transactionId: requestContext.contextValue?.transactionId
      });

    } catch (error) {
      this.logger.error('Failed to record request start metrics', {
        error: error.message
      });
    }
  }

  private async recordRequestComplete(
    requestContext: GraphQLRequestContext<any>,
    startTime: number
  ): Promise<void> {
    const { request, response, context } = requestContext;
    const duration = Date.now() - startTime;

    try {
      const operationType = this.extractOperationType(request);
      const operationName = request.operationName;

      // Record request duration
      this.metricsCollector.recordRequestDuration(duration, operationType, operationName);

      // Record response size
      if (response?.body) {
        const responseSize = this.calculateResponseSize(response.body);
        this.metricsCollector.recordResponseSize(responseSize);
      }

      // Record KB service calls from evidence envelope
      if (context?.evidenceEnvelope?.kbResponses) {
        for (const kbResponse of context.evidenceEnvelope.kbResponses) {
          this.metricsCollector.recordKBServiceCall(
            kbResponse.kb,
            kbResponse.latency,
            (kbResponse.errorCount || 0) === 0
          );

          if (kbResponse.cacheHit !== undefined) {
            // Note: This records individual cache hits, not overall hit rate
            // Overall hit rate would need to be calculated separately
            this.metricsCollector.recordCacheHitRate(
              kbResponse.kb,
              kbResponse.cacheHit ? 1.0 : 0.0
            );
          }
        }
      }

      // Record version set usage
      if (context?.versionSet?.id) {
        this.metricsCollector.recordVersionSetUsage(context.versionSet.id);

        // Record version override if applicable
        if (context.versionSetOverride) {
          this.metricsCollector.recordVersionOverride(
            'default',
            context.versionSet.id
          );
        }
      }

      // Record evidence envelope size
      if (context?.evidenceEnvelope) {
        const envelopeSize = this.calculateEvidenceEnvelopeSize(context.evidenceEnvelope);
        this.metricsCollector.recordEvidenceEnvelopeSize(envelopeSize);
      }

      this.logger.debug('Request completion metrics recorded', {
        operationType,
        operationName,
        duration,
        transactionId: context?.transactionId,
        kbCallCount: context?.evidenceEnvelope?.kbResponses?.length || 0
      });

    } catch (error) {
      this.logger.error('Failed to record request completion metrics', {
        error: error.message,
        transactionId: requestContext.contextValue?.transactionId
      });
    }
  }

  private recordRequestErrors(
    requestContext: GraphQLRequestContext<any>,
    startTime: number
  ): void {
    const { request, errors } = requestContext;
    const duration = Date.now() - startTime;

    try {
      const operationType = this.extractOperationType(request);
      const operationName = request.operationName;

      // Record request duration (even for errors)
      this.metricsCollector.recordRequestDuration(duration, operationType, operationName);

      // Record specific errors
      if (errors) {
        for (const error of errors) {
          this.metricsCollector.recordRequestError(
            error.message,
            operationType,
            operationName
          );
        }
      }

      this.logger.debug('Request error metrics recorded', {
        operationType,
        operationName,
        duration,
        errorCount: errors?.length || 0,
        transactionId: requestContext.contextValue?.transactionId
      });

    } catch (error) {
      this.logger.error('Failed to record request error metrics', {
        error: error.message
      });
    }
  }

  private extractOperationType(request: any): string {
    if (request.operationName) {
      return request.operationName;
    }

    // Parse query to determine operation type
    if (request.query) {
      const query = request.query.toLowerCase().trim();
      if (query.startsWith('query')) {
        return 'query';
      } else if (query.startsWith('mutation')) {
        return 'mutation';
      } else if (query.startsWith('subscription')) {
        return 'subscription';
      }
    }

    return 'unknown';
  }

  private calculateResponseSize(responseBody: any): number {
    try {
      if (typeof responseBody === 'string') {
        return Buffer.byteLength(responseBody, 'utf8');
      } else {
        return Buffer.byteLength(JSON.stringify(responseBody), 'utf8');
      }
    } catch {
      return 0;
    }
  }

  private calculateEvidenceEnvelopeSize(evidenceEnvelope: any): number {
    try {
      const envelopeData = {
        decisionChain: evidenceEnvelope.decisionChain || [],
        kbResponses: evidenceEnvelope.kbResponses || [],
        safetyAttestations: evidenceEnvelope.safetyAttestations || [],
        performanceMetrics: evidenceEnvelope.performanceMetrics || {}
      };
      
      return Buffer.byteLength(JSON.stringify(envelopeData), 'utf8');
    } catch {
      return 0;
    }
  }

  // Method to get current metrics (for health check endpoints)
  getCurrentMetrics(): any {
    return this.metricsCollector.exportMetrics();
  }
}