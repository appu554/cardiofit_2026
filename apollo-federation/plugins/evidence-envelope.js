const fetch = (...args) => import('node-fetch').then(({default: fetch}) => fetch(...args));
const crypto = require('crypto');

// Evidence Envelope Plugin for Apollo Federation
// Tracks all GraphQL operations and provides audit trail functionality
class EvidenceEnvelopePlugin {
  constructor(options = {}) {
    this.evidenceServiceUrl = options.evidenceServiceUrl || 'http://localhost:8088/graphql';
    this.enableLogging = options.enableLogging !== false;
    this.logger = options.logger || console;
  }

  // Generate correlation ID for request tracking
  generateCorrelationId() {
    return `corr_${crypto.randomBytes(16).toString('hex')}_${Date.now()}`;
  }

  // Create transaction record in Evidence Envelope
  async createTransaction(transactionData) {
    try {
      const mutation = `
        mutation CreateTransaction($input: TransactionInput!) {
          createTransaction(input: $input) {
            transactionId
            createdAt
          }
        }
      `;

      const response = await fetch(this.evidenceServiceUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          query: mutation,
          variables: { input: transactionData }
        })
      });

      if (!response.ok) {
        throw new Error(`Evidence Envelope API error: ${response.status}`);
      }

      const result = await response.json();
      
      if (result.errors) {
        throw new Error(`GraphQL errors: ${JSON.stringify(result.errors)}`);
      }

      return result.data.createTransaction;
    } catch (error) {
      this.logger.error('Failed to create transaction in Evidence Envelope:', error);
      return null;
    }
  }

  // Complete transaction with response data
  async completeTransaction(transactionId, responseData, processingTimeMs, httpStatus = 200) {
    try {
      const mutation = `
        mutation CompleteTransaction(
          $transactionId: String!
          $responsePayload: JSON
          $httpStatus: Int!
          $processingTimeMs: Int!
        ) {
          completeTransaction(
            transactionId: $transactionId
            responsePayload: $responsePayload
            httpStatus: $httpStatus
            processingTimeMs: $processingTimeMs
          )
        }
      `;

      const response = await fetch(this.evidenceServiceUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          query: mutation,
          variables: {
            transactionId,
            responsePayload: responseData,
            httpStatus,
            processingTimeMs
          }
        })
      });

      if (!response.ok) {
        this.logger.warn(`Failed to complete transaction ${transactionId}: ${response.status}`);
        return false;
      }

      const result = await response.json();
      return !result.errors;
    } catch (error) {
      this.logger.error('Failed to complete transaction:', error);
      return false;
    }
  }

  // Apollo Server plugin implementation
  requestDidStart() {
    return {
      didResolveOperation: async (requestContext) => {
        const startTime = Date.now();
        const correlationId = this.generateCorrelationId();
        
        // Extract user context from headers
        const userId = requestContext.request.http?.headers?.get('x-user-id');
        const sessionId = requestContext.request.http?.headers?.get('x-session-id');
        
        // Determine target service from operation
        const targetService = this.extractTargetService(requestContext.request.query);
        
        // Create transaction data
        const transactionData = {
          userId: userId || null,
          sessionId: sessionId || null,
          sourceService: 'apollo-federation',
          targetService: targetService,
          operationType: requestContext.request.operationName ? 'query' : 'mutation',
          graphqlOperation: requestContext.request.query,
          requestPayload: {
            query: requestContext.request.query,
            variables: requestContext.request.variables,
            operationName: requestContext.request.operationName
          },
          correlationId: correlationId
        };

        // Create transaction in Evidence Envelope
        const transactionResult = await this.createTransaction(transactionData);
        
        if (transactionResult) {
          // Store transaction info in request context for completion
          requestContext.request.transactionId = transactionResult.transactionId;
          requestContext.request.startTime = startTime;
          requestContext.request.correlationId = correlationId;
          
          if (this.enableLogging) {
            this.logger.info(`[Evidence Envelope] Transaction created: ${transactionResult.transactionId}`, {
              correlationId,
              userId,
              targetService
            });
          }
        }
      },

      didEncounterErrors: async (requestContext) => {
        // Complete transaction with error status
        if (requestContext.request.transactionId) {
          const processingTime = Date.now() - requestContext.request.startTime;
          const errorResponse = {
            errors: requestContext.errors.map(error => ({
              message: error.message,
              extensions: error.extensions
            }))
          };

          await this.completeTransaction(
            requestContext.request.transactionId,
            errorResponse,
            processingTime,
            500
          );
        }
      },

      willSendResponse: async (requestContext) => {
        // Complete successful transaction
        if (requestContext.request.transactionId) {
          const processingTime = Date.now() - requestContext.request.startTime;
          
          await this.completeTransaction(
            requestContext.request.transactionId,
            requestContext.response.http.body,
            processingTime,
            requestContext.response.http.status || 200
          );

          if (this.enableLogging) {
            this.logger.info(`[Evidence Envelope] Transaction completed: ${requestContext.request.transactionId}`, {
              processingTime: `${processingTime}ms`,
              correlationId: requestContext.request.correlationId
            });
          }
        }
      }
    };
  }

  // Extract target service from GraphQL query
  extractTargetService(query) {
    if (!query) return 'unknown';

    // Simple pattern matching to identify target services
    const servicePatterns = [
      { pattern: /\b(patient|Patient)\b/i, service: 'patients' },
      { pattern: /\b(medication|Medication|drug|Drug)\b/i, service: 'medications' },
      { pattern: /\b(interaction|drugInteraction)\b/i, service: 'kb1-drug-interactions' },
      { pattern: /\b(phenotype|clinicalContext)\b/i, service: 'kb2-clinical-context' },
      { pattern: /\b(adverseEvent|sideEffect)\b/i, service: 'kb3-adverse-events' },
      { pattern: /\b(analytics|trend|pattern)\b/i, service: 'kb4-analytics' },
      { pattern: /\b(outcome|effectiveness)\b/i, service: 'kb5-outcomes' },
      { pattern: /\b(formulary|coverage|insurance)\b/i, service: 'kb6-formulary' },
      { pattern: /\b(terminology|code|coding)\b/i, service: 'kb7-terminology' },
      { pattern: /\b(transaction|audit|evidence)\b/i, service: 'evidence-envelope' }
    ];

    for (const { pattern, service } of servicePatterns) {
      if (pattern.test(query)) {
        return service;
      }
    }

    return 'federation-gateway';
  }
}

// Helper function to create Evidence Envelope plugin
function createEvidenceEnvelopePlugin(options = {}) {
  return new EvidenceEnvelopePlugin(options);
}

module.exports = {
  EvidenceEnvelopePlugin,
  createEvidenceEnvelopePlugin
};