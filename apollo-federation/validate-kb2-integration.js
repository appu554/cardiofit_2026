#!/usr/bin/env node

/**
 * KB-2 Integration Validation Script
 * 
 * Validates that the KB-2 federation integration files are syntactically correct
 * and can be loaded without errors.
 */

const path = require('path');

// Logger utility
const logger = {
  info: (message, data) => console.log(`[VALIDATE][INFO] ${new Date().toISOString()} - ${message}`, data || ''),
  error: (message, error) => console.error(`[VALIDATE][ERROR] ${new Date().toISOString()} - ${message}`, error?.message || error),
  success: (message) => console.log(`[VALIDATE][SUCCESS] ✅ ${message}`),
  warn: (message, data) => console.warn(`[VALIDATE][WARN] ⚠️  ${message}`, data || '')
};

class KB2IntegrationValidator {
  constructor() {
    this.errors = [];
    this.warnings = [];
  }

  validate(testName, testFunction) {
    try {
      logger.info(`Validating: ${testName}`);
      testFunction();
      logger.success(`${testName} - Valid`);
      return true;
    } catch (error) {
      this.errors.push({ test: testName, error: error.message });
      logger.error(`${testName} - Invalid:`, error);
      return false;
    }
  }

  validateSchemaLoad() {
    const schemaPath = path.join(__dirname, 'schemas', 'kb2-clinical-context-schema.js');
    const schema = require(schemaPath);
    
    if (!schema || typeof schema !== 'string') {
      throw new Error('Schema is not a valid GraphQL schema string');
    }

    if (!schema.includes('extend type Patient @key(fields: "id")')) {
      throw new Error('Schema missing Patient extension with federation key');
    }

    if (!schema.includes('type ClinicalContext')) {
      throw new Error('Schema missing ClinicalContext type definition');
    }

    if (!schema.includes('type ClinicalPhenotype')) {
      throw new Error('Schema missing ClinicalPhenotype type definition');
    }

    if (!schema.includes('type RiskAssessment')) {
      throw new Error('Schema missing RiskAssessment type definition');
    }

    if (!schema.includes('type TreatmentPreference')) {
      throw new Error('Schema missing TreatmentPreference type definition');
    }

    if (!schema.includes('evaluatePatientPhenotypes')) {
      throw new Error('Schema missing phenotype evaluation query');
    }

    if (!schema.includes('assessPatientRisk')) {
      throw new Error('Schema missing risk assessment query');
    }

    if (!schema.includes('getPatientTreatmentPreferences')) {
      throw new Error('Schema missing treatment preferences query');
    }

    if (!schema.includes('assemblePatientContext')) {
      throw new Error('Schema missing context assembly query');
    }
  }

  validateResolversLoad() {
    const resolversPath = path.join(__dirname, 'resolvers', 'kb2-clinical-context-resolvers.js');
    const resolvers = require(resolversPath);
    
    if (!resolvers || typeof resolvers !== 'object') {
      throw new Error('Resolvers is not a valid resolver object');
    }

    if (!resolvers.Query) {
      throw new Error('Resolvers missing Query field');
    }

    if (!resolvers.Patient) {
      throw new Error('Resolvers missing Patient field for federation');
    }

    const requiredQueries = [
      'evaluatePatientPhenotypes',
      'assessPatientRisk', 
      'getPatientTreatmentPreferences',
      'assemblePatientContext',
      'availablePhenotypes',
      'patientContextHistory'
    ];

    for (const query of requiredQueries) {
      if (!resolvers.Query[query]) {
        throw new Error(`Resolvers missing Query.${query} resolver`);
      }
      
      if (typeof resolvers.Query[query] !== 'function') {
        throw new Error(`Query.${query} is not a function`);
      }
    }

    const requiredPatientResolvers = [
      'clinicalContext',
      'phenotypes',
      'riskAssessments', 
      'treatmentPreferences'
    ];

    for (const resolver of requiredPatientResolvers) {
      if (!resolvers.Patient[resolver]) {
        throw new Error(`Resolvers missing Patient.${resolver} resolver`);
      }
      
      if (typeof resolvers.Patient[resolver] !== 'function') {
        throw new Error(`Patient.${resolver} is not a function`);
      }
    }
  }

  validateSubgraphService() {
    const servicePath = path.join(__dirname, 'services', 'kb2-clinical-context-service.js');
    const service = require(servicePath);
    
    if (!service || typeof service !== 'object') {
      throw new Error('Service is not a valid service object');
    }

    if (!service.createSubgraphServer || typeof service.createSubgraphServer !== 'function') {
      throw new Error('Service missing createSubgraphServer function');
    }

    if (!service.startServer || typeof service.startServer !== 'function') {
      throw new Error('Service missing startServer function');
    }
  }

  validateIndexFiles() {
    // Validate schema index
    const schemaIndexPath = path.join(__dirname, 'schemas', 'index.js');
    const schemaIndex = require(schemaIndexPath);
    
    if (!schemaIndex.kb2ClinicalContextSchema) {
      throw new Error('Schema index missing kb2ClinicalContextSchema export');
    }

    // Validate resolver index
    const resolverIndexPath = path.join(__dirname, 'resolvers', 'index.js');
    const resolverIndex = require(resolverIndexPath);
    
    if (!resolverIndex.kb2ClinicalContextResolvers) {
      throw new Error('Resolver index missing kb2ClinicalContextResolvers export');
    }
  }

  validateSupergraphConfig() {
    const fs = require('fs');
    const yamlPath = path.join(__dirname, 'supergraph.yaml');
    
    if (!fs.existsSync(yamlPath)) {
      throw new Error('supergraph.yaml not found');
    }

    const yamlContent = fs.readFileSync(yamlPath, 'utf8');
    
    if (!yamlContent.includes('kb2-clinical-context:')) {
      throw new Error('supergraph.yaml missing kb2-clinical-context subgraph');
    }

    if (!yamlContent.includes('http://localhost:8082/api/federation')) {
      throw new Error('supergraph.yaml missing correct KB-2 federation URL');
    }
  }

  validateTestScript() {
    const testPath = path.join(__dirname, 'test-kb2-integration.js');
    const fs = require('fs');
    
    if (!fs.existsSync(testPath)) {
      throw new Error('Integration test script not found');
    }

    // Basic syntax validation by requiring the file
    const TestClass = require(testPath);
    
    if (typeof TestClass !== 'function') {
      throw new Error('Test script does not export a constructor function');
    }
  }

  validateEnvironmentConfig() {
    // Check that required environment variables are documented
    const requiredVars = [
      'KB2_CLINICAL_CONTEXT_URL',
      'KB2_SUBGRAPH_PORT'
    ];

    const warnings = [];
    for (const envVar of requiredVars) {
      if (!process.env[envVar]) {
        warnings.push(`Environment variable ${envVar} not set (will use default)`);
      }
    }

    if (warnings.length > 0) {
      this.warnings.push(...warnings);
      logger.warn('Environment configuration warnings:', warnings);
    }
  }

  validateDependencies() {
    const packageJsonPath = path.join(__dirname, 'package.json');
    const fs = require('fs');
    
    if (!fs.existsSync(packageJsonPath)) {
      throw new Error('package.json not found');
    }

    const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
    const dependencies = { ...packageJson.dependencies, ...packageJson.devDependencies };

    const requiredDeps = [
      '@apollo/server',
      '@apollo/subgraph',
      '@apollo/gateway',
      'apollo-server-express',
      'axios',
      'express',
      'cors'
    ];

    for (const dep of requiredDeps) {
      if (!dependencies[dep]) {
        throw new Error(`Missing required dependency: ${dep}`);
      }
    }
  }

  runAllValidations() {
    logger.info('Starting KB-2 Federation Integration Validation');

    let validationsPassed = 0;
    let totalValidations = 0;

    const validations = [
      { name: 'Schema Load', fn: () => this.validateSchemaLoad() },
      { name: 'Resolvers Load', fn: () => this.validateResolversLoad() },
      { name: 'Subgraph Service', fn: () => this.validateSubgraphService() },
      { name: 'Index Files', fn: () => this.validateIndexFiles() },
      { name: 'Supergraph Config', fn: () => this.validateSupergraphConfig() },
      { name: 'Test Script', fn: () => this.validateTestScript() },
      { name: 'Environment Config', fn: () => this.validateEnvironmentConfig() },
      { name: 'Dependencies', fn: () => this.validateDependencies() }
    ];

    for (const validation of validations) {
      totalValidations++;
      if (this.validate(validation.name, validation.fn)) {
        validationsPassed++;
      }
    }

    // Report results
    logger.info('\n=== KB-2 Integration Validation Results ===');
    logger.info(`Total Validations: ${totalValidations}`);
    logger.info(`Passed: ${validationsPassed}`);
    logger.info(`Failed: ${this.errors.length}`);
    logger.info(`Warnings: ${this.warnings.length}`);

    if (this.errors.length > 0) {
      logger.error('\n=== Validation Errors ===');
      this.errors.forEach(error => {
        logger.error(`${error.test}: ${error.error}`);
      });
    }

    if (this.warnings.length > 0) {
      logger.warn('\n=== Validation Warnings ===');
      this.warnings.forEach(warning => {
        logger.warn(warning);
      });
    }

    if (this.errors.length === 0) {
      logger.success('\n🎉 All KB-2 integration validations passed!');
      
      if (this.warnings.length > 0) {
        logger.warn(`✅ Integration is valid with ${this.warnings.length} warning(s)`);
      } else {
        logger.success('✅ Integration is completely valid with no warnings');
      }

      logger.info('\n=== Next Steps ===');
      logger.info('1. Start KB-2 service: cd backend/services/knowledge-base-services/kb-2-clinical-context-go && make run');
      logger.info('2. Start federation gateway: npm start');
      logger.info('3. Run integration tests: node test-kb2-integration.js');
      logger.info('4. Access GraphQL playground: http://localhost:4000/graphql');

      process.exit(0);
    } else {
      logger.error(`\n❌ ${this.errors.length} validation(s) failed`);
      logger.error('Please fix the errors above before proceeding');
      process.exit(1);
    }
  }
}

// Run validations if this file is executed directly
if (require.main === module) {
  const validator = new KB2IntegrationValidator();
  validator.runAllValidations();
}

module.exports = KB2IntegrationValidator;