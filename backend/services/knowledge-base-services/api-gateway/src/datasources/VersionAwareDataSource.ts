import { RemoteGraphQLDataSource } from '@apollo/gateway';
import { GraphQLRequestContext, GraphQLResponse } from 'apollo-server-types';
import { createLogger, Logger } from 'winston';
import crypto from 'crypto';

interface DataSourceConfig {
  url: string;
  name: string;
  versionSet: any;
  healthCheck?: {
    interval: number;
    timeout: number;
    retries: number;
  };
  circuitBreaker?: {
    threshold: number;
    duration: number;
    bucketSize: number;
  };
}

interface KBMetadata {
  kb: string;
  version: string;
  digest?: string;
  latency: number;
  cacheHit: boolean;
  responseSize?: number;
  errorCount?: number;
}

interface CircuitBreakerState {
  isOpen: boolean;
  failureCount: number;
  lastFailureTime: number;
  successCount: number;
}

export class VersionAwareDataSource extends RemoteGraphQLDataSource {
  private config: DataSourceConfig;
  private logger: Logger;
  private circuitBreakerState: CircuitBreakerState;
  private healthCheckInterval: NodeJS.Timeout | null = null;
  private isHealthy: boolean = true;
  private requestMetrics: Map<string, number[]> = new Map();

  constructor(config: DataSourceConfig) {
    super({ url: config.url });
    
    this.config = config;
    this.logger = createLogger({
      defaultMeta: { service: `kb-datasource-${config.name}` }
    });
    
    this.circuitBreakerState = {
      isOpen: false,
      failureCount: 0,
      lastFailureTime: 0,
      successCount: 0
    };

    this.initializeHealthCheck();
    this.initializeCircuitBreaker();
  }

  willSendRequest({ request, context }: { request: any; context: any }): void {
    const startTime = Date.now();
    
    try {
      // Check circuit breaker
      if (this.isCircuitBreakerOpen()) {
        this.logger.warn('Circuit breaker is open', { 
          service: this.config.name,
          failureCount: this.circuitBreakerState.failureCount
        });
        throw new Error(`Circuit breaker open for ${this.config.name}`);
      }

      // Inject version and tracking headers
      const envelope = context.evidenceEnvelope || context;
      const kbVersion = this.config.versionSet.kb_versions?.[this.config.name] || 
                        this.config.versionSet[this.config.name];
      
      if (!kbVersion) {
        this.logger.warn('No version found for KB', { 
          kbName: this.config.name,
          availableVersions: Object.keys(this.config.versionSet.kb_versions || this.config.versionSet)
        });
      }

      // Set headers for downstream service
      request.http.headers.set('x-kb-version', kbVersion || 'latest');
      request.http.headers.set('x-transaction-id', context.transactionId);
      request.http.headers.set('x-evidence-envelope-id', context.evidenceEnvelopeId);
      request.http.headers.set('x-request-id', context.requestId);
      request.http.headers.set('x-user-id', context.userContext?.userId);
      request.http.headers.set('x-request-start', startTime.toString());
      request.http.headers.set('x-kb-service', this.config.name);

      // Add authentication if available
      if (context.userContext?.token) {
        request.http.headers.set('authorization', `Bearer ${context.userContext.token}`);
      }

      this.logger.debug('Sending request to KB service', {
        kbName: this.config.name,
        version: kbVersion,
        transactionId: context.transactionId,
        operationName: request.operationName
      });

    } catch (error) {
      this.logger.error('Error preparing request', {
        kbName: this.config.name,
        error: error.message,
        transactionId: context?.transactionId
      });
      throw error;
    }
  }

  async didReceiveResponse({ 
    response, 
    request, 
    context 
  }: { 
    response: GraphQLResponse; 
    request: any; 
    context: any; 
  }): Promise<GraphQLResponse> {
    
    const endTime = Date.now();
    const startTime = parseInt(request.http.headers.get('x-request-start') || '0');
    const responseTime = endTime - startTime;

    try {
      // Extract KB response metadata
      const kbMetadata: KBMetadata = {
        kb: this.config.name,
        version: response.http?.headers?.get('x-kb-actual-version') || 
                 request.http.headers.get('x-kb-version'),
        digest: response.http?.headers?.get('x-content-digest'),
        latency: responseTime,
        cacheHit: response.http?.headers?.get('x-cache-hit') === 'true',
        responseSize: this.calculateResponseSize(response),
        errorCount: response.errors ? response.errors.length : 0
      };

      // Record metrics
      this.recordMetrics(kbMetadata);

      // Update circuit breaker state
      if (response.errors && response.errors.length > 0) {
        this.recordFailure();
        this.logger.warn('KB service returned errors', {
          kbName: this.config.name,
          errors: response.errors,
          transactionId: context.transactionId
        });
      } else {
        this.recordSuccess();
      }

      // Add KB call to evidence envelope
      if (context.evidenceEnvelope && context.evidenceEnvelope.addKBResponse) {
        context.evidenceEnvelope.addKBResponse(kbMetadata);
      }

      // Validate response against expected schema version if needed
      await this.validateResponseSchema(response, kbMetadata.version);

      this.logger.debug('Received response from KB service', {
        kbName: this.config.name,
        latency: responseTime,
        cacheHit: kbMetadata.cacheHit,
        hasErrors: !!response.errors,
        transactionId: context.transactionId
      });

      return response;

    } catch (error) {
      this.recordFailure();
      this.logger.error('Error processing KB response', {
        kbName: this.config.name,
        error: error.message,
        transactionId: context?.transactionId
      });
      throw error;
    }
  }

  async didEncounterError(error: Error, request: any, context: any): Promise<void> {
    this.recordFailure();
    
    this.logger.error('KB service request failed', {
      kbName: this.config.name,
      error: error.message,
      stack: error.stack,
      transactionId: context?.transactionId,
      url: this.config.url
    });

    // If circuit breaker threshold reached, open circuit
    if (this.shouldOpenCircuitBreaker()) {
      this.openCircuitBreaker();
    }
  }

  private initializeHealthCheck(): void {
    if (this.config.healthCheck) {
      const { interval } = this.config.healthCheck;
      
      this.healthCheckInterval = setInterval(async () => {
        try {
          await this.performHealthCheck();
        } catch (error) {
          this.logger.error('Health check failed', {
            kbName: this.config.name,
            error: error.message
          });
        }
      }, interval);
    }
  }

  private initializeCircuitBreaker(): void {
    if (this.config.circuitBreaker) {
      // Reset circuit breaker periodically
      setInterval(() => {
        if (this.circuitBreakerState.isOpen) {
          const timeSinceLastFailure = Date.now() - this.circuitBreakerState.lastFailureTime;
          if (timeSinceLastFailure > this.config.circuitBreaker!.duration) {
            this.halfOpenCircuitBreaker();
          }
        }
      }, 5000);
    }
  }

  private async performHealthCheck(): Promise<void> {
    const healthUrl = `${this.config.url.replace('/graphql', '')}/health`;
    
    try {
      const response = await fetch(healthUrl, {
        method: 'GET',
        timeout: this.config.healthCheck!.timeout
      });

      this.isHealthy = response.ok;
      
      if (!this.isHealthy) {
        this.logger.warn('KB service health check failed', {
          kbName: this.config.name,
          status: response.status,
          statusText: response.statusText
        });
      }
    } catch (error) {
      this.isHealthy = false;
      this.logger.error('KB service health check error', {
        kbName: this.config.name,
        error: error.message
      });
    }
  }

  private recordMetrics(metadata: KBMetadata): void {
    const key = `${metadata.kb}_latency`;
    
    if (!this.requestMetrics.has(key)) {
      this.requestMetrics.set(key, []);
    }
    
    const latencies = this.requestMetrics.get(key)!;
    latencies.push(metadata.latency);
    
    // Keep only last 100 measurements
    if (latencies.length > 100) {
      latencies.shift();
    }
    
    // Log metrics periodically
    if (latencies.length % 10 === 0) {
      const avg = latencies.reduce((a, b) => a + b, 0) / latencies.length;
      const p95 = this.calculatePercentile(latencies, 0.95);
      
      this.logger.info('KB service metrics', {
        kbName: this.config.name,
        avgLatency: Math.round(avg),
        p95Latency: Math.round(p95),
        requestCount: latencies.length
      });
    }
  }

  private calculatePercentile(values: number[], percentile: number): number {
    const sorted = [...values].sort((a, b) => a - b);
    const index = Math.ceil(sorted.length * percentile) - 1;
    return sorted[index] || 0;
  }

  private recordSuccess(): void {
    this.circuitBreakerState.successCount++;
    
    if (this.circuitBreakerState.isOpen && this.circuitBreakerState.successCount >= 3) {
      this.closeCircuitBreaker();
    }
  }

  private recordFailure(): void {
    this.circuitBreakerState.failureCount++;
    this.circuitBreakerState.lastFailureTime = Date.now();
    this.circuitBreakerState.successCount = 0;
  }

  private isCircuitBreakerOpen(): boolean {
    return this.circuitBreakerState.isOpen;
  }

  private shouldOpenCircuitBreaker(): boolean {
    if (!this.config.circuitBreaker) return false;
    
    const { threshold, bucketSize } = this.config.circuitBreaker;
    const recentFailures = this.circuitBreakerState.failureCount;
    
    return recentFailures >= Math.ceil(bucketSize * threshold);
  }

  private openCircuitBreaker(): void {
    this.circuitBreakerState.isOpen = true;
    this.circuitBreakerState.lastFailureTime = Date.now();
    
    this.logger.error('Circuit breaker opened for KB service', {
      kbName: this.config.name,
      failureCount: this.circuitBreakerState.failureCount
    });
  }

  private halfOpenCircuitBreaker(): void {
    this.circuitBreakerState.isOpen = false;
    this.circuitBreakerState.successCount = 0;
    
    this.logger.info('Circuit breaker half-opened for KB service', {
      kbName: this.config.name
    });
  }

  private closeCircuitBreaker(): void {
    this.circuitBreakerState.isOpen = false;
    this.circuitBreakerState.failureCount = 0;
    this.circuitBreakerState.successCount = 0;
    
    this.logger.info('Circuit breaker closed for KB service', {
      kbName: this.config.name
    });
  }

  private calculateResponseSize(response: GraphQLResponse): number {
    try {
      return JSON.stringify(response.data || {}).length;
    } catch {
      return 0;
    }
  }

  private async validateResponseSchema(response: GraphQLResponse, version: string): Promise<void> {
    // TODO: Implement schema validation based on KB version
    // This would validate that the response conforms to the expected schema
    // for the specific KB version being used
    
    if (response.errors && response.errors.length > 0) {
      const criticalErrors = response.errors.filter(error => 
        error.extensions?.code === 'INTERNAL_ERROR' ||
        error.extensions?.code === 'UNAUTHENTICATED'
      );
      
      if (criticalErrors.length > 0) {
        this.logger.error('Critical errors in KB response', {
          kbName: this.config.name,
          version,
          errors: criticalErrors
        });
      }
    }
  }

  // Cleanup method
  destroy(): void {
    if (this.healthCheckInterval) {
      clearInterval(this.healthCheckInterval);
      this.healthCheckInterval = null;
    }
  }

  // Getter methods for monitoring
  getMetrics(): any {
    return {
      kbName: this.config.name,
      isHealthy: this.isHealthy,
      circuitBreakerState: { ...this.circuitBreakerState },
      requestMetrics: Object.fromEntries(this.requestMetrics.entries())
    };
  }
}