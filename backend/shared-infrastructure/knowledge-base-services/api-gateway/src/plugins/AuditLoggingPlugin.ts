import { ApolloServerPlugin, GraphQLRequestListener } from '@apollo/server';
import { GraphQLRequestContext } from '@apollo/server';
import { DatabaseManager } from '../database/DatabaseManager';
import { Logger } from 'winston';

interface AuditLogEntry {
  id?: string;
  timestamp: Date;
  transactionId: string;
  operationType: string;
  operationName?: string;
  userId?: string;
  ipAddress?: string;
  userAgent?: string;
  query: string;
  variables?: any;
  kbServicesUsed: string[];
  responseTime: number;
  success: boolean;
  errorMessage?: string;
  versionSetId: string;
  evidenceEnvelopeId: string;
  clinicalDomain?: string;
  patientId?: string;
  metadata: any;
}

export class AuditLoggingPlugin implements ApolloServerPlugin {
  private dbManager: DatabaseManager;
  private logger: Logger;

  constructor(dbManager: DatabaseManager, logger: Logger) {
    this.dbManager = dbManager;
    this.logger = logger;
  }

  async requestDidStart(): Promise<GraphQLRequestListener<any>> {
    const startTime = Date.now();

    return {
      async willSendResponse(requestContext: GraphQLRequestContext<any>) {
        await this.logRequest(requestContext, startTime);
      },

      async didEncounterErrors(requestContext: GraphQLRequestContext<any>) {
        await this.logErrorRequest(requestContext, startTime);
      }
    };
  }

  private async logRequest(
    requestContext: GraphQLRequestContext<any>,
    startTime: number
  ): Promise<void> {
    const { context, request, response } = requestContext;
    const responseTime = Date.now() - startTime;

    try {
      const auditEntry: AuditLogEntry = {
        timestamp: new Date(),
        transactionId: context.transactionId,
        operationType: this.extractOperationType(request),
        operationName: request.operationName,
        userId: context.userContext?.userId,
        ipAddress: context.ipAddress,
        userAgent: context.userAgent,
        query: this.sanitizeQuery(request.query),
        variables: this.sanitizeVariables(request.variables),
        kbServicesUsed: this.extractKBServicesUsed(context),
        responseTime,
        success: !response?.errors || response.errors.length === 0,
        errorMessage: response?.errors ? 
          response.errors.map(e => e.message).join('; ') : undefined,
        versionSetId: context.versionSet?.id,
        evidenceEnvelopeId: context.evidenceEnvelopeId,
        clinicalDomain: this.extractClinicalDomain(request.variables),
        patientId: this.extractPatientId(request.variables),
        metadata: {
          requestId: context.requestId,
          kbVersions: context.versionSet?.kb_versions,
          warnings: context.warnings,
          versionOverride: context.versionSetOverride
        }
      };

      // Log to database
      await this.persistAuditLog(auditEntry);

      // Log to application logs
      this.logger.info('GraphQL request audited', {
        transactionId: context.transactionId,
        operationType: auditEntry.operationType,
        operationName: auditEntry.operationName,
        responseTime: auditEntry.responseTime,
        success: auditEntry.success,
        kbServicesUsed: auditEntry.kbServicesUsed,
        userId: auditEntry.userId
      });

    } catch (error) {
      this.logger.error('Failed to log audit entry', {
        error: error.message,
        transactionId: context.transactionId
      });
      // Don't throw here - audit logging failure shouldn't break the request
    }
  }

  private async logErrorRequest(
    requestContext: GraphQLRequestContext<any>,
    startTime: number
  ): Promise<void> {
    const { context, request, errors } = requestContext;
    const responseTime = Date.now() - startTime;

    try {
      const auditEntry: AuditLogEntry = {
        timestamp: new Date(),
        transactionId: context.transactionId,
        operationType: this.extractOperationType(request),
        operationName: request.operationName,
        userId: context.userContext?.userId,
        ipAddress: context.ipAddress,
        userAgent: context.userAgent,
        query: this.sanitizeQuery(request.query),
        variables: this.sanitizeVariables(request.variables),
        kbServicesUsed: this.extractKBServicesUsed(context),
        responseTime,
        success: false,
        errorMessage: errors ? errors.map(e => e.message).join('; ') : 'Unknown error',
        versionSetId: context.versionSet?.id,
        evidenceEnvelopeId: context.evidenceEnvelopeId,
        clinicalDomain: this.extractClinicalDomain(request.variables),
        patientId: this.extractPatientId(request.variables),
        metadata: {
          requestId: context.requestId,
          kbVersions: context.versionSet?.kb_versions,
          errorDetails: errors?.map(error => ({
            message: error.message,
            locations: error.locations,
            path: error.path,
            extensions: error.extensions
          }))
        }
      };

      // Log to database
      await this.persistAuditLog(auditEntry);

      // Log error details
      this.logger.error('GraphQL request failed', {
        transactionId: context.transactionId,
        operationType: auditEntry.operationType,
        operationName: auditEntry.operationName,
        responseTime: auditEntry.responseTime,
        errorMessage: auditEntry.errorMessage,
        userId: auditEntry.userId,
        errorDetails: auditEntry.metadata.errorDetails
      });

    } catch (error) {
      this.logger.error('Failed to log error audit entry', {
        error: error.message,
        transactionId: context.transactionId
      });
    }
  }

  private async persistAuditLog(auditEntry: AuditLogEntry): Promise<void> {
    const query = `
      INSERT INTO kb_audit_log (
        timestamp, transaction_id, operation_type, operation_name,
        user_id, ip_address, user_agent, query, variables,
        kb_services_used, response_time_ms, success, error_message,
        version_set_id, evidence_envelope_id, clinical_domain, patient_id,
        metadata
      ) VALUES (
        $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
      )
    `;

    const values = [
      auditEntry.timestamp,
      auditEntry.transactionId,
      auditEntry.operationType,
      auditEntry.operationName,
      auditEntry.userId,
      auditEntry.ipAddress,
      auditEntry.userAgent,
      auditEntry.query,
      JSON.stringify(auditEntry.variables),
      JSON.stringify(auditEntry.kbServicesUsed),
      auditEntry.responseTime,
      auditEntry.success,
      auditEntry.errorMessage,
      auditEntry.versionSetId,
      auditEntry.evidenceEnvelopeId,
      auditEntry.clinicalDomain,
      auditEntry.patientId,
      JSON.stringify(auditEntry.metadata)
    ];

    await this.dbManager.query(query, values);
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

  private sanitizeQuery(query?: string): string {
    if (!query) return '';

    // Remove sensitive data patterns from query
    return query
      .replace(/password\s*:\s*"[^"]*"/gi, 'password: "***"')
      .replace(/token\s*:\s*"[^"]*"/gi, 'token: "***"')
      .replace(/apiKey\s*:\s*"[^"]*"/gi, 'apiKey: "***"')
      // Limit query length for storage
      .substring(0, 5000);
  }

  private sanitizeVariables(variables?: any): any {
    if (!variables) return null;

    // Deep clone and sanitize sensitive fields
    const sanitized = JSON.parse(JSON.stringify(variables));
    
    const sensitiveFields = ['password', 'token', 'apiKey', 'secret', 'ssn', 'creditCard'];
    
    function sanitizeObject(obj: any): void {
      if (typeof obj === 'object' && obj !== null) {
        for (const key in obj) {
          if (sensitiveFields.some(field => 
            key.toLowerCase().includes(field.toLowerCase())
          )) {
            obj[key] = '***';
          } else if (typeof obj[key] === 'object') {
            sanitizeObject(obj[key]);
          }
        }
      }
    }

    sanitizeObject(sanitized);
    return sanitized;
  }

  private extractKBServicesUsed(context: any): string[] {
    const kbServices: string[] = [];

    // Extract from evidence envelope if available
    if (context.evidenceEnvelope?.kbResponses) {
      const usedServices = context.evidenceEnvelope.kbResponses.map(
        (response: any) => response.kb
      );
      kbServices.push(...new Set(usedServices));
    }

    // Extract from version set as fallback
    if (kbServices.length === 0 && context.versionSet?.kb_versions) {
      kbServices.push(...Object.keys(context.versionSet.kb_versions));
    }

    return kbServices;
  }

  private extractClinicalDomain(variables?: any): string | undefined {
    if (!variables) return undefined;

    // Try to extract clinical domain from various variable structures
    if (variables.clinicalDomain) {
      return variables.clinicalDomain;
    }

    if (variables.input?.clinicalDomain) {
      return variables.input.clinicalDomain;
    }

    if (variables.context?.clinicalDomain) {
      return variables.context.clinicalDomain;
    }

    // Infer from other fields
    if (variables.condition || variables.input?.condition) {
      return 'clinical_condition';
    }

    if (variables.drugCode || variables.input?.drugCode) {
      return 'medication_management';
    }

    return undefined;
  }

  private extractPatientId(variables?: any): string | undefined {
    if (!variables) return undefined;

    // Try to extract patient ID from various variable structures
    const possibleFields = [
      'patientId',
      'patient_id',
      'input.patientId',
      'input.patient_id',
      'context.patientId',
      'context.patient_id'
    ];

    for (const field of possibleFields) {
      const value = this.getNestedProperty(variables, field);
      if (value) {
        return value.toString();
      }
    }

    return undefined;
  }

  private getNestedProperty(obj: any, path: string): any {
    return path.split('.').reduce((current, key) => {
      return current && current[key] !== undefined ? current[key] : undefined;
    }, obj);
  }
}