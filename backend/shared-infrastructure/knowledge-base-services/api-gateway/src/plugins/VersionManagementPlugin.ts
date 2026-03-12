import { ApolloServerPlugin, GraphQLRequestListener } from '@apollo/server';
import { GraphQLRequestContext } from '@apollo/server';
import { KBVersionManager } from '../services/KBVersionManager';
import { createLogger, Logger } from 'winston';

export class VersionManagementPlugin implements ApolloServerPlugin {
  private versionManager: KBVersionManager;
  private logger: Logger;

  constructor(versionManager: KBVersionManager) {
    this.versionManager = versionManager;
    this.logger = createLogger({
      defaultMeta: { service: 'version-management-plugin' }
    });
  }

  async requestDidStart(): Promise<GraphQLRequestListener<any>> {
    return {
      async willSendRequest(requestContext: GraphQLRequestContext<any>) {
        await this.handleVersionRequest(requestContext);
      },

      async didResolveOperation(requestContext: GraphQLRequestContext<any>) {
        await this.validateOperationVersion(requestContext);
      },

      async willSendResponse(requestContext: GraphQLRequestContext<any>) {
        await this.addVersionHeaders(requestContext);
      }
    };
  }

  private async handleVersionRequest(
    requestContext: GraphQLRequestContext<any>
  ): Promise<void> {
    const { context, request } = requestContext;

    try {
      // Check for version-specific requests
      const requestedVersionSet = this.extractVersionSetFromRequest(request);
      
      if (requestedVersionSet) {
        this.logger.debug('Version-specific request detected', {
          requestedVersion: requestedVersionSet,
          transactionId: context.transactionId
        });

        // Validate the requested version set exists and is valid
        const versionSet = await this.versionManager.getVersionSet(requestedVersionSet);
        
        if (!versionSet) {
          throw new Error(`Requested version set not found: ${requestedVersionSet}`);
        }

        if (!versionSet.validated) {
          this.logger.warn('Using unvalidated version set', {
            versionSetId: requestedVersionSet,
            transactionId: context.transactionId
          });
        }

        // Override context version set
        context.versionSet = versionSet;
        context.versionSetOverride = true;

        this.logger.info('Version set override applied', {
          originalVersionSet: context.versionSet?.id,
          overrideVersionSet: requestedVersionSet,
          transactionId: context.transactionId
        });
      }

      // Handle version compatibility checks
      await this.performVersionCompatibilityCheck(context);

    } catch (error) {
      this.logger.error('Version request handling failed', {
        error: error.message,
        transactionId: context.transactionId
      });
      throw error;
    }
  }

  private async validateOperationVersion(
    requestContext: GraphQLRequestContext<any>
  ): Promise<void> {
    const { context, document } = requestContext;

    try {
      // Extract KB services needed for this operation
      const requiredKBs = this.extractRequiredKBsFromOperation(document);
      
      if (requiredKBs.length > 0) {
        // Validate that all required KBs are available in the version set
        const versionSet = context.versionSet;
        const missingKBs = requiredKBs.filter(
          kb => !versionSet.kb_versions[kb]
        );

        if (missingKBs.length > 0) {
          this.logger.error('Required KBs missing from version set', {
            missingKBs,
            versionSetId: versionSet.id,
            transactionId: context.transactionId
          });
          
          throw new Error(
            `Required KB services not available in version set: ${missingKBs.join(', ')}`
          );
        }

        // Check for deprecated versions
        const deprecatedKBs = await this.checkDeprecatedVersions(
          versionSet.kb_versions,
          requiredKBs
        );

        if (deprecatedKBs.length > 0) {
          this.logger.warn('Using deprecated KB versions', {
            deprecatedKBs,
            versionSetId: versionSet.id,
            transactionId: context.transactionId
          });

          // Add warning to context for response headers
          context.warnings = context.warnings || [];
          context.warnings.push(
            `Deprecated KB versions in use: ${deprecatedKBs.map(kb => `${kb.name}@${kb.version}`).join(', ')}`
          );
        }

        this.logger.debug('Operation version validation completed', {
          requiredKBs,
          versionSetId: versionSet.id,
          transactionId: context.transactionId
        });
      }

    } catch (error) {
      this.logger.error('Operation version validation failed', {
        error: error.message,
        transactionId: context.transactionId
      });
      throw error;
    }
  }

  private async addVersionHeaders(
    requestContext: GraphQLRequestContext<any>
  ): Promise<void> {
    const { context, response } = requestContext;

    if (response && response.http) {
      try {
        // Add version information to response headers
        const versionSet = context.versionSet;
        
        response.http.headers.set('x-kb-version-set-id', versionSet.id);
        response.http.headers.set('x-kb-version-set-name', versionSet.versionSetName);
        response.http.headers.set('x-kb-version-set-validated', versionSet.validated.toString());
        
        // Add individual KB versions
        for (const [kbName, version] of Object.entries(versionSet.kb_versions)) {
          response.http.headers.set(`x-kb-${kbName}-version`, version);
        }

        // Add warnings if any
        if (context.warnings && context.warnings.length > 0) {
          response.http.headers.set('x-kb-warnings', context.warnings.join('; '));
        }

        // Add version override indicator
        if (context.versionSetOverride) {
          response.http.headers.set('x-kb-version-override', 'true');
        }

        this.logger.debug('Version headers added to response', {
          versionSetId: versionSet.id,
          kbCount: Object.keys(versionSet.kb_versions).length,
          hasWarnings: !!(context.warnings && context.warnings.length > 0),
          transactionId: context.transactionId
        });

      } catch (error) {
        this.logger.error('Failed to add version headers', {
          error: error.message,
          transactionId: context.transactionId
        });
        // Don't throw here as this is not critical
      }
    }
  }

  private extractVersionSetFromRequest(request: any): string | null {
    // Check GraphQL variables
    if (request.variables && request.variables.versionSet) {
      return request.variables.versionSet;
    }

    // Check HTTP headers
    if (request.http && request.http.headers) {
      const versionSetHeader = request.http.headers.get('x-kb-version-set');
      if (versionSetHeader) {
        return versionSetHeader;
      }
    }

    return null;
  }

  private extractRequiredKBsFromOperation(document: any): string[] {
    const requiredKBs: Set<string> = new Set();

    // Parse GraphQL document to identify which KB services are needed
    if (document && document.definitions) {
      for (const definition of document.definitions) {
        if (definition.kind === 'OperationDefinition') {
          this.extractKBsFromSelectionSet(definition.selectionSet, requiredKBs);
        }
      }
    }

    return Array.from(requiredKBs);
  }

  private extractKBsFromSelectionSet(selectionSet: any, requiredKBs: Set<string>): void {
    if (!selectionSet || !selectionSet.selections) {
      return;
    }

    for (const selection of selectionSet.selections) {
      if (selection.kind === 'Field') {
        // Map field names to KB services
        const kbMapping: Record<string, string> = {
          'dosing': 'kb_1_dosing',
          'phenotype': 'kb_2_context',
          'guideline': 'kb_3_guidelines',
          'safetySignals': 'kb_4_safety',
          'checkInteractions': 'kb_5_ddi',
          'formularyLookup': 'kb_6_formulary',
          'terminologyLookup': 'kb_7_terminology'
        };

        const fieldName = selection.name.value;
        if (kbMapping[fieldName]) {
          requiredKBs.add(kbMapping[fieldName]);
        }

        // Recursively check nested selections
        if (selection.selectionSet) {
          this.extractKBsFromSelectionSet(selection.selectionSet, requiredKBs);
        }
      } else if (selection.kind === 'InlineFragment' || selection.kind === 'FragmentSpread') {
        if (selection.selectionSet) {
          this.extractKBsFromSelectionSet(selection.selectionSet, requiredKBs);
        }
      }
    }
  }

  private async performVersionCompatibilityCheck(context: any): Promise<void> {
    const versionSet = context.versionSet;
    
    try {
      // Check for known compatibility issues
      const compatibilityIssues = await this.checkVersionCompatibility(versionSet.kb_versions);
      
      if (compatibilityIssues.length > 0) {
        this.logger.warn('Version compatibility issues detected', {
          issues: compatibilityIssues,
          versionSetId: versionSet.id,
          transactionId: context.transactionId
        });

        // Add to context warnings
        context.warnings = context.warnings || [];
        context.warnings.push(...compatibilityIssues.map(issue => issue.description));

        // If critical issues, throw error
        const criticalIssues = compatibilityIssues.filter(issue => issue.severity === 'critical');
        if (criticalIssues.length > 0) {
          throw new Error(
            `Critical version compatibility issues: ${criticalIssues.map(i => i.description).join('; ')}`
          );
        }
      }

    } catch (error) {
      this.logger.error('Version compatibility check failed', {
        error: error.message,
        versionSetId: versionSet.id,
        transactionId: context.transactionId
      });
      throw error;
    }
  }

  private async checkVersionCompatibility(
    kbVersions: Record<string, string>
  ): Promise<Array<{ description: string; severity: 'warning' | 'critical' }>> {
    const issues: Array<{ description: string; severity: 'warning' | 'critical' }> = [];

    // Define compatibility rules
    const compatibilityRules = [
      {
        name: 'kb_1_kb_5_compatibility',
        check: (versions: Record<string, string>) => {
          // Example: KB-1 Dosing Rules v3.x requires KB-5 DDI v2.5+
          const kb1Version = versions['kb_1_dosing'];
          const kb5Version = versions['kb_5_ddi'];
          
          if (kb1Version && kb5Version) {
            const kb1Major = parseInt(kb1Version.split('.')[0]);
            const kb5Minor = parseFloat(kb5Version.split('.').slice(0, 2).join('.'));
            
            if (kb1Major >= 3 && kb5Minor < 2.5) {
              return {
                description: `KB-1 v${kb1Version} requires KB-5 v2.5+, but found v${kb5Version}`,
                severity: 'critical' as const
              };
            }
          }
          return null;
        }
      },
      {
        name: 'deprecated_version_check',
        check: (versions: Record<string, string>) => {
          // Example: Check for deprecated versions
          const deprecatedVersions = {
            'kb_2_context': ['1.0.0', '1.1.0'],
            'kb_3_guidelines': ['0.9.0']
          };

          const issues = [];
          for (const [kbName, version] of Object.entries(versions)) {
            if (deprecatedVersions[kbName]?.includes(version)) {
              issues.push({
                description: `${kbName} v${version} is deprecated`,
                severity: 'warning' as const
              });
            }
          }
          return issues.length > 0 ? issues : null;
        }
      }
    ];

    // Run compatibility checks
    for (const rule of compatibilityRules) {
      try {
        const result = rule.check(kbVersions);
        if (result) {
          if (Array.isArray(result)) {
            issues.push(...result);
          } else {
            issues.push(result);
          }
        }
      } catch (error) {
        this.logger.error('Compatibility rule check failed', {
          rule: rule.name,
          error: error.message
        });
      }
    }

    return issues;
  }

  private async checkDeprecatedVersions(
    kbVersions: Record<string, string>,
    requiredKBs: string[]
  ): Promise<Array<{ name: string; version: string; reason?: string }>> {
    const deprecatedKBs: Array<{ name: string; version: string; reason?: string }> = [];

    // This would typically query a deprecation database or API
    // For now, simulate with hardcoded deprecated versions
    const deprecationList: Record<string, string[]> = {
      'kb_1_dosing': ['1.0.0', '1.1.0', '2.0.0-beta'],
      'kb_2_context': ['0.9.0'],
      'kb_3_guidelines': ['1.0.0-alpha', '1.0.0-beta']
    };

    for (const kbName of requiredKBs) {
      const version = kbVersions[kbName];
      if (version && deprecationList[kbName]?.includes(version)) {
        deprecatedKBs.push({
          name: kbName,
          version,
          reason: 'Version marked as deprecated'
        });
      }
    }

    return deprecatedKBs;
  }
}