import { DatabaseManager } from '../database/DatabaseManager';
import { createLogger, Logger } from 'winston';
import crypto from 'crypto';

export interface KBVersionSet {
  id: string;
  versionSetName: string;
  description?: string;
  kb_versions: Record<string, string>;
  validated: boolean;
  validationResults?: any;
  validationTimestamp?: Date;
  environment: string;
  active: boolean;
  activatedAt?: Date;
  deactivatedAt?: Date;
  createdBy: string;
  approvedBy?: string;
  approvalTimestamp?: Date;
  approvalNotes?: string;
  createdAt: Date;
  updatedAt: Date;
}

export interface VersionDeployment {
  id: string;
  versionSetId: string;
  environment: string;
  deployedBy: string;
  deployedAt: Date;
  rollbackVersionSetId?: string;
  status: 'deploying' | 'deployed' | 'failed' | 'rolled_back';
  deploymentLog: any[];
  validationResults: any[];
}

export interface KBVersionInfo {
  kbName: string;
  currentVersion: string;
  availableVersions: string[];
  lastUpdate: Date;
  healthStatus: 'healthy' | 'warning' | 'error';
  metadata: any;
}

export class KBVersionManager {
  private dbManager: DatabaseManager;
  private logger: Logger;
  private cache: Map<string, KBVersionSet> = new Map();
  private cacheExpiry: Map<string, number> = new Map();
  private readonly CACHE_TTL = 300000; // 5 minutes

  constructor(dbManager: DatabaseManager) {
    this.dbManager = dbManager;
    this.logger = createLogger({
      defaultMeta: { service: 'kb-version-manager' }
    });
  }

  async getActiveVersionSet(environment: string = 'development'): Promise<KBVersionSet> {
    const cacheKey = `active_${environment}`;
    
    // Check cache first
    if (this.isCacheValid(cacheKey)) {
      const cached = this.cache.get(cacheKey);
      if (cached) {
        this.logger.debug('Returning cached active version set', { 
          environment, 
          versionSetId: cached.id 
        });
        return cached;
      }
    }

    try {
      const query = `
        SELECT 
          id, version_set_name, description, kb_versions, validated,
          validation_results, validation_timestamp, environment, active,
          activated_at, deactivated_at, created_by, approved_by,
          approval_timestamp, approval_notes, created_at, updated_at
        FROM kb_version_sets
        WHERE environment = $1 AND active = true
        ORDER BY activated_at DESC
        LIMIT 1
      `;

      const result = await this.dbManager.query(query, [environment]);
      
      if (result.rows.length === 0) {
        throw new Error(`No active version set found for environment: ${environment}`);
      }

      const versionSet = this.mapRowToVersionSet(result.rows[0]);
      
      // Cache the result
      this.cache.set(cacheKey, versionSet);
      this.cacheExpiry.set(cacheKey, Date.now() + this.CACHE_TTL);

      this.logger.info('Retrieved active version set', {
        environment,
        versionSetId: versionSet.id,
        versionSetName: versionSet.versionSetName
      });

      return versionSet;

    } catch (error) {
      this.logger.error('Failed to get active version set', {
        environment,
        error: error.message
      });
      throw error;
    }
  }

  async getVersionSet(versionSetId: string): Promise<KBVersionSet> {
    const cacheKey = `version_set_${versionSetId}`;
    
    // Check cache first
    if (this.isCacheValid(cacheKey)) {
      const cached = this.cache.get(cacheKey);
      if (cached) {
        return cached;
      }
    }

    try {
      const query = `
        SELECT 
          id, version_set_name, description, kb_versions, validated,
          validation_results, validation_timestamp, environment, active,
          activated_at, deactivated_at, created_by, approved_by,
          approval_timestamp, approval_notes, created_at, updated_at
        FROM kb_version_sets
        WHERE id = $1
      `;

      const result = await this.dbManager.query(query, [versionSetId]);
      
      if (result.rows.length === 0) {
        throw new Error(`Version set not found: ${versionSetId}`);
      }

      const versionSet = this.mapRowToVersionSet(result.rows[0]);
      
      // Cache the result
      this.cache.set(cacheKey, versionSet);
      this.cacheExpiry.set(cacheKey, Date.now() + this.CACHE_TTL);

      return versionSet;

    } catch (error) {
      this.logger.error('Failed to get version set', {
        versionSetId,
        error: error.message
      });
      throw error;
    }
  }

  async createVersionSet(
    versionSetData: Partial<KBVersionSet>,
    createdBy: string
  ): Promise<KBVersionSet> {
    try {
      // Generate unique ID and name if not provided
      const id = versionSetData.id || this.generateVersionSetId();
      const versionSetName = versionSetData.versionSetName || 
        `version_set_${new Date().toISOString().slice(0, 10)}`;

      // Validate KB versions format
      if (!versionSetData.kb_versions || Object.keys(versionSetData.kb_versions).length === 0) {
        throw new Error('KB versions are required');
      }

      const query = `
        INSERT INTO kb_version_sets (
          id, version_set_name, description, kb_versions, validated,
          environment, active, created_by, created_at, updated_at
        ) VALUES (
          $1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()
        )
        RETURNING *
      `;

      const values = [
        id,
        versionSetName,
        versionSetData.description || null,
        JSON.stringify(versionSetData.kb_versions),
        false, // Not validated initially
        versionSetData.environment || 'development',
        false, // Not active initially
        createdBy
      ];

      const result = await this.dbManager.query(query, values);
      const newVersionSet = this.mapRowToVersionSet(result.rows[0]);

      this.logger.info('Created new version set', {
        versionSetId: id,
        versionSetName,
        createdBy,
        environment: versionSetData.environment
      });

      // Invalidate cache for the environment
      this.invalidateEnvironmentCache(versionSetData.environment || 'development');

      return newVersionSet;

    } catch (error) {
      this.logger.error('Failed to create version set', {
        error: error.message,
        versionSetData
      });
      throw error;
    }
  }

  async validateVersionSet(versionSetId: string): Promise<{ valid: boolean; results: any }> {
    try {
      const versionSet = await this.getVersionSet(versionSetId);
      const validationResults: any = {
        timestamp: new Date().toISOString(),
        checks: [],
        overall: true
      };

      // Check 1: Version format validation
      for (const [kbName, version] of Object.entries(versionSet.kb_versions)) {
        const versionCheck = this.validateVersionFormat(version);
        validationResults.checks.push({
          check: 'version_format',
          kbName,
          version,
          passed: versionCheck.valid,
          message: versionCheck.message
        });
        if (!versionCheck.valid) {
          validationResults.overall = false;
        }
      }

      // Check 2: KB service availability
      for (const [kbName, version] of Object.entries(versionSet.kb_versions)) {
        const availabilityCheck = await this.checkKBAvailability(kbName, version);
        validationResults.checks.push({
          check: 'kb_availability',
          kbName,
          version,
          passed: availabilityCheck.available,
          message: availabilityCheck.message,
          responseTime: availabilityCheck.responseTime
        });
        if (!availabilityCheck.available) {
          validationResults.overall = false;
        }
      }

      // Check 3: Compatibility validation
      const compatibilityCheck = await this.validateKBCompatibility(versionSet.kb_versions);
      validationResults.checks.push({
        check: 'kb_compatibility',
        passed: compatibilityCheck.compatible,
        message: compatibilityCheck.message,
        conflicts: compatibilityCheck.conflicts
      });
      if (!compatibilityCheck.compatible) {
        validationResults.overall = false;
      }

      // Update version set with validation results
      await this.updateVersionSetValidation(versionSetId, validationResults);

      this.logger.info('Version set validation completed', {
        versionSetId,
        passed: validationResults.overall,
        checkCount: validationResults.checks.length
      });

      return {
        valid: validationResults.overall,
        results: validationResults
      };

    } catch (error) {
      this.logger.error('Failed to validate version set', {
        versionSetId,
        error: error.message
      });
      throw error;
    }
  }

  async deployVersionSet(
    versionSetId: string,
    environment: string,
    deployedBy: string,
    options: {
      validateFirst?: boolean;
      force?: boolean;
      rollbackOnFailure?: boolean;
    } = {}
  ): Promise<VersionDeployment> {
    try {
      const versionSet = await this.getVersionSet(versionSetId);
      
      // Validate first if requested
      if (options.validateFirst && !versionSet.validated) {
        const validation = await this.validateVersionSet(versionSetId);
        if (!validation.valid && !options.force) {
          throw new Error('Version set validation failed. Use force=true to override.');
        }
      }

      // Get current active version set for potential rollback
      let currentVersionSet: KBVersionSet | null = null;
      try {
        currentVersionSet = await this.getActiveVersionSet(environment);
      } catch (error) {
        // No active version set exists
      }

      // Create deployment record
      const deploymentId = this.generateDeploymentId();
      const deployment: VersionDeployment = {
        id: deploymentId,
        versionSetId,
        environment,
        deployedBy,
        deployedAt: new Date(),
        rollbackVersionSetId: currentVersionSet?.id,
        status: 'deploying',
        deploymentLog: [],
        validationResults: []
      };

      // Log deployment start
      deployment.deploymentLog.push({
        timestamp: new Date().toISOString(),
        level: 'info',
        message: 'Deployment started',
        details: { versionSetId, environment }
      });

      try {
        // Deactivate current version set
        if (currentVersionSet) {
          await this.deactivateVersionSet(currentVersionSet.id);
          deployment.deploymentLog.push({
            timestamp: new Date().toISOString(),
            level: 'info',
            message: 'Previous version set deactivated',
            details: { previousVersionSetId: currentVersionSet.id }
          });
        }

        // Activate new version set
        await this.activateVersionSet(versionSetId, environment);
        deployment.deploymentLog.push({
          timestamp: new Date().toISOString(),
          level: 'info',
          message: 'New version set activated',
          details: { versionSetId }
        });

        // Perform post-deployment validation
        const postValidation = await this.performPostDeploymentValidation(versionSet);
        deployment.validationResults = postValidation.results;

        if (postValidation.success) {
          deployment.status = 'deployed';
          deployment.deploymentLog.push({
            timestamp: new Date().toISOString(),
            level: 'info',
            message: 'Deployment completed successfully'
          });
        } else {
          deployment.status = 'failed';
          deployment.deploymentLog.push({
            timestamp: new Date().toISOString(),
            level: 'error',
            message: 'Post-deployment validation failed',
            details: postValidation.errors
          });

          // Rollback if requested
          if (options.rollbackOnFailure && currentVersionSet) {
            await this.rollbackDeployment(deploymentId, currentVersionSet.id);
          }
        }

      } catch (error) {
        deployment.status = 'failed';
        deployment.deploymentLog.push({
          timestamp: new Date().toISOString(),
          level: 'error',
          message: 'Deployment failed',
          details: { error: error.message }
        });
        throw error;
      }

      // Store deployment record
      await this.storeDeployment(deployment);

      this.logger.info('Version set deployment completed', {
        deploymentId,
        versionSetId,
        environment,
        status: deployment.status
      });

      // Invalidate caches
      this.invalidateEnvironmentCache(environment);

      return deployment;

    } catch (error) {
      this.logger.error('Failed to deploy version set', {
        versionSetId,
        environment,
        error: error.message
      });
      throw error;
    }
  }

  async rollbackVersionSet(
    currentVersionSetId: string,
    targetVersionSetId: string,
    performedBy: string
  ): Promise<void> {
    try {
      const currentVersionSet = await this.getVersionSet(currentVersionSetId);
      const targetVersionSet = await this.getVersionSet(targetVersionSetId);

      if (currentVersionSet.environment !== targetVersionSet.environment) {
        throw new Error('Cannot rollback across different environments');
      }

      // Deactivate current version
      await this.deactivateVersionSet(currentVersionSetId);

      // Activate target version
      await this.activateVersionSet(targetVersionSetId, currentVersionSet.environment);

      // Log the rollback
      this.logger.info('Version set rolled back', {
        from: currentVersionSetId,
        to: targetVersionSetId,
        environment: currentVersionSet.environment,
        performedBy
      });

      // Invalidate caches
      this.invalidateEnvironmentCache(currentVersionSet.environment);

    } catch (error) {
      this.logger.error('Failed to rollback version set', {
        currentVersionSetId,
        targetVersionSetId,
        error: error.message
      });
      throw error;
    }
  }

  async getVersionHistory(
    kbName?: string,
    environment?: string,
    limit: number = 50
  ): Promise<KBVersionSet[]> {
    try {
      let query = `
        SELECT 
          id, version_set_name, description, kb_versions, validated,
          validation_results, validation_timestamp, environment, active,
          activated_at, deactivated_at, created_by, approved_by,
          approval_timestamp, approval_notes, created_at, updated_at
        FROM kb_version_sets
        WHERE 1=1
      `;
      
      const params: any[] = [];
      let paramIndex = 1;

      if (environment) {
        query += ` AND environment = $${paramIndex}`;
        params.push(environment);
        paramIndex++;
      }

      if (kbName) {
        query += ` AND kb_versions ? $${paramIndex}`;
        params.push(kbName);
        paramIndex++;
      }

      query += ` ORDER BY created_at DESC LIMIT $${paramIndex}`;
      params.push(limit);

      const result = await this.dbManager.query(query, params);
      
      return result.rows.map(row => this.mapRowToVersionSet(row));

    } catch (error) {
      this.logger.error('Failed to get version history', {
        kbName,
        environment,
        error: error.message
      });
      throw error;
    }
  }

  // Private helper methods

  private isCacheValid(key: string): boolean {
    const expiry = this.cacheExpiry.get(key);
    return expiry !== undefined && Date.now() < expiry;
  }

  private invalidateEnvironmentCache(environment: string): void {
    const cacheKey = `active_${environment}`;
    this.cache.delete(cacheKey);
    this.cacheExpiry.delete(cacheKey);
  }

  private mapRowToVersionSet(row: any): KBVersionSet {
    return {
      id: row.id,
      versionSetName: row.version_set_name,
      description: row.description,
      kb_versions: row.kb_versions,
      validated: row.validated,
      validationResults: row.validation_results,
      validationTimestamp: row.validation_timestamp,
      environment: row.environment,
      active: row.active,
      activatedAt: row.activated_at,
      deactivatedAt: row.deactivated_at,
      createdBy: row.created_by,
      approvedBy: row.approved_by,
      approvalTimestamp: row.approval_timestamp,
      approvalNotes: row.approval_notes,
      createdAt: row.created_at,
      updatedAt: row.updated_at
    };
  }

  private generateVersionSetId(): string {
    const timestamp = Date.now().toString(36);
    const random = crypto.randomBytes(4).toString('hex');
    return `vs_${timestamp}_${random}`;
  }

  private generateDeploymentId(): string {
    const timestamp = Date.now().toString(36);
    const random = crypto.randomBytes(4).toString('hex');
    return `deploy_${timestamp}_${random}`;
  }

  private validateVersionFormat(version: string): { valid: boolean; message: string } {
    // Semantic versioning with optional SHA
    const semverPattern = /^\d+\.\d+\.\d+(\+sha\.[a-f0-9]+)?$/;
    
    if (semverPattern.test(version)) {
      return { valid: true, message: 'Valid semantic version format' };
    }
    
    return { 
      valid: false, 
      message: `Invalid version format: ${version}. Expected format: X.Y.Z or X.Y.Z+sha.HASH` 
    };
  }

  private async checkKBAvailability(
    kbName: string, 
    version: string
  ): Promise<{ available: boolean; message: string; responseTime?: number }> {
    // TODO: Implement actual KB health check
    // For now, simulate the check
    const startTime = Date.now();
    
    try {
      // This would make an actual HTTP request to the KB service
      // const response = await fetch(`${kbServiceUrl}/health`);
      const responseTime = Date.now() - startTime;
      
      return {
        available: true,
        message: 'Service is available',
        responseTime
      };
    } catch (error) {
      return {
        available: false,
        message: `Service unavailable: ${error.message}`
      };
    }
  }

  private async validateKBCompatibility(
    kbVersions: Record<string, string>
  ): Promise<{ compatible: boolean; message: string; conflicts?: any[] }> {
    // TODO: Implement compatibility matrix checking
    // For now, assume all versions are compatible
    return {
      compatible: true,
      message: 'All KB versions are compatible'
    };
  }

  private async updateVersionSetValidation(versionSetId: string, results: any): Promise<void> {
    const query = `
      UPDATE kb_version_sets
      SET validated = $1, validation_results = $2, validation_timestamp = NOW()
      WHERE id = $3
    `;

    await this.dbManager.query(query, [results.overall, JSON.stringify(results), versionSetId]);
    
    // Invalidate cache
    const cacheKey = `version_set_${versionSetId}`;
    this.cache.delete(cacheKey);
    this.cacheExpiry.delete(cacheKey);
  }

  private async activateVersionSet(versionSetId: string, environment: string): Promise<void> {
    const query = `
      UPDATE kb_version_sets
      SET active = true, activated_at = NOW()
      WHERE id = $1 AND environment = $2
    `;

    await this.dbManager.query(query, [versionSetId, environment]);
  }

  private async deactivateVersionSet(versionSetId: string): Promise<void> {
    const query = `
      UPDATE kb_version_sets
      SET active = false, deactivated_at = NOW()
      WHERE id = $1
    `;

    await this.dbManager.query(query, [versionSetId]);
  }

  private async performPostDeploymentValidation(
    versionSet: KBVersionSet
  ): Promise<{ success: boolean; results: any[]; errors?: any[] }> {
    // TODO: Implement post-deployment health checks
    return {
      success: true,
      results: [{
        check: 'post_deployment_health',
        passed: true,
        timestamp: new Date().toISOString()
      }]
    };
  }

  private async rollbackDeployment(deploymentId: string, rollbackVersionSetId: string): Promise<void> {
    // TODO: Implement deployment rollback logic
    this.logger.info('Rolling back deployment', { deploymentId, rollbackVersionSetId });
  }

  private async storeDeployment(deployment: VersionDeployment): Promise<void> {
    // TODO: Store deployment record in database
    this.logger.info('Storing deployment record', { deploymentId: deployment.id });
  }
}