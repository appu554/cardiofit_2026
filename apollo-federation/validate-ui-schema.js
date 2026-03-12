/**
 * UI Interaction Schema Validation Script
 * Validates GraphQL schema composition and UI interaction capabilities
 */

const fs = require('fs');
const path = require('path');
const { buildSchema, validate, parse } = require('graphql');

const logger = {
  info: (message) => console.log(`[VALIDATE] ${new Date().toISOString()} - ${message}`),
  error: (message, error) => console.error(`[ERROR] ${new Date().toISOString()} - ${message}`, error?.message || error),
  success: (message) => console.log(`[SUCCESS] ${new Date().toISOString()} - ${message}`),
  warn: (message) => console.warn(`[WARN] ${new Date().toISOString()} - ${message}`)
};

// Load and validate schema files
function loadSchemaFiles() {
  const schemaDir = path.join(__dirname, 'schemas');
  const schemas = {};

  try {
    // Core workflow orchestration schema
    const orchestrationPath = path.join(schemaDir, 'workflow-orchestration-schema.graphql');
    if (fs.existsSync(orchestrationPath)) {
      schemas.orchestration = fs.readFileSync(orchestrationPath, 'utf8');
      logger.success('Loaded workflow orchestration schema');
    } else {
      logger.warn('Workflow orchestration schema not found');
    }

    // UI interaction schema
    const uiPath = path.join(schemaDir, 'workflow-ui-interaction-schema.graphql');
    if (fs.existsSync(uiPath)) {
      schemas.uiInteraction = fs.readFileSync(uiPath, 'utf8');
      logger.success('Loaded UI interaction schema');
    } else {
      logger.error('UI interaction schema not found - this is required');
      return null;
    }

    return schemas;
  } catch (error) {
    logger.error('Failed to load schema files:', error);
    return null;
  }
}

// Validate GraphQL syntax
function validateGraphQLSyntax(schema, name) {
  try {
    parse(schema);
    logger.success(`✅ ${name} syntax is valid`);
    return true;
  } catch (error) {
    logger.error(`❌ ${name} syntax error:`, error.message);
    return false;
  }
}

// Check for required UI interaction types
function validateUIInteractionTypes(schema) {
  const requiredTypes = [
    'OverrideSession',
    'UINotification',
    'PeerReviewSession',
    'WorkflowUIState',
    'OverrideDecisionInput',
    'UINotificationInput'
  ];

  const requiredMutations = [
    'updateUINotification',
    'requestClinicalOverride',
    'resolveClinicalOverride',
    'requestPeerReview'
  ];

  const requiredSubscriptions = [
    'workflowUIUpdates',
    'overrideRequired',
    'peerReviewUpdates'
  ];

  const requiredEnums = [
    'OverrideLevel',
    'UINotificationStatus',
    'ReviewUrgency',
    'OverrideDecision'
  ];

  let score = 0;
  let total = 0;

  // Check types
  requiredTypes.forEach(type => {
    total++;
    if (schema.includes(`type ${type}`)) {
      logger.success(`✅ Type ${type} found`);
      score++;
    } else {
      logger.warn(`⚠️ Type ${type} missing`);
    }
  });

  // Check mutations
  requiredMutations.forEach(mutation => {
    total++;
    if (schema.includes(mutation)) {
      logger.success(`✅ Mutation ${mutation} found`);
      score++;
    } else {
      logger.warn(`⚠️ Mutation ${mutation} missing`);
    }
  });

  // Check subscriptions
  requiredSubscriptions.forEach(subscription => {
    total++;
    if (schema.includes(subscription)) {
      logger.success(`✅ Subscription ${subscription} found`);
      score++;
    } else {
      logger.warn(`⚠️ Subscription ${subscription} missing`);
    }
  });

  // Check enums
  requiredEnums.forEach(enumType => {
    total++;
    if (schema.includes(`enum ${enumType}`)) {
      logger.success(`✅ Enum ${enumType} found`);
      score++;
    } else {
      logger.warn(`⚠️ Enum ${enumType} missing`);
    }
  });

  const percentage = Math.round((score / total) * 100);
  logger.info(`📊 UI Interaction completeness: ${score}/${total} (${percentage}%)`);

  return percentage >= 80; // 80% minimum for deployment
}

// Validate resolver file
function validateResolverFile() {
  const resolverPath = path.join(__dirname, 'resolvers', 'workflow-ui-interaction-resolver.js');

  if (!fs.existsSync(resolverPath)) {
    logger.error('❌ UI interaction resolver file not found');
    return false;
  }

  try {
    const resolverCode = fs.readFileSync(resolverPath, 'utf8');

    // Check for key resolver functions
    const requiredResolvers = [
      'updateUINotification',
      'requestClinicalOverride',
      'resolveClinicalOverride',
      'workflowUIUpdates',
      'overrideRequired'
    ];

    let resolverScore = 0;
    requiredResolvers.forEach(resolver => {
      if (resolverCode.includes(resolver)) {
        logger.success(`✅ Resolver ${resolver} implemented`);
        resolverScore++;
      } else {
        logger.warn(`⚠️ Resolver ${resolver} missing`);
      }
    });

    // Check for PubSub usage
    if (resolverCode.includes('pubsub') || resolverCode.includes('PubSub')) {
      logger.success('✅ Real-time subscription support detected');
    } else {
      logger.warn('⚠️ No real-time subscription support found');
    }

    // Check for Redis usage
    if (resolverCode.includes('redis') || resolverCode.includes('Redis')) {
      logger.success('✅ Redis session management detected');
    } else {
      logger.warn('⚠️ No Redis session management found');
    }

    const resolverPercentage = Math.round((resolverScore / requiredResolvers.length) * 100);
    logger.info(`📊 Resolver completeness: ${resolverScore}/${requiredResolvers.length} (${resolverPercentage}%)`);

    return resolverPercentage >= 80;
  } catch (error) {
    logger.error('❌ Failed to validate resolver file:', error);
    return false;
  }
}

// Check gateway configuration
function validateGatewayConfig() {
  const gatewayPath = path.join(__dirname, 'workflow-ui-gateway.js');

  if (!fs.existsSync(gatewayPath)) {
    logger.error('❌ Enhanced gateway configuration not found');
    return false;
  }

  try {
    const gatewayCode = fs.readFileSync(gatewayPath, 'utf8');

    // Check for key features
    const features = [
      { name: 'WebSocket support', check: code => code.includes('WebSocketServer') },
      { name: 'Subscription handling', check: code => code.includes('subscriptions') },
      { name: 'Real-time PubSub', check: code => code.includes('PubSub') },
      { name: 'Enhanced context', check: code => code.includes('createAuthContext') },
      { name: 'Error handling', check: code => code.includes('formatError') }
    ];

    let featureScore = 0;
    features.forEach(feature => {
      if (feature.check(gatewayCode)) {
        logger.success(`✅ ${feature.name} configured`);
        featureScore++;
      } else {
        logger.warn(`⚠️ ${feature.name} not configured`);
      }
    });

    const featurePercentage = Math.round((featureScore / features.length) * 100);
    logger.info(`📊 Gateway features: ${featureScore}/${features.length} (${featurePercentage}%)`);

    return featurePercentage >= 80;
  } catch (error) {
    logger.error('❌ Failed to validate gateway configuration:', error);
    return false;
  }
}

// Validate package.json dependencies
function validateDependencies() {
  const packagePath = path.join(__dirname, 'package.json');

  if (!fs.existsSync(packagePath)) {
    logger.error('❌ package.json not found');
    return false;
  }

  try {
    const packageJson = JSON.parse(fs.readFileSync(packagePath, 'utf8'));
    const deps = { ...packageJson.dependencies, ...packageJson.devDependencies };

    const requiredDeps = [
      'graphql-subscriptions',
      'graphql-ws',
      'ws',
      'ioredis',
      '@graphql-tools/schema',
      '@apollo/gateway'
    ];

    let depScore = 0;
    requiredDeps.forEach(dep => {
      if (deps[dep]) {
        logger.success(`✅ Dependency ${dep} found (${deps[dep]})`);
        depScore++;
      } else {
        logger.warn(`⚠️ Dependency ${dep} missing`);
      }
    });

    // Check for UI-specific scripts
    const scripts = packageJson.scripts || {};
    const requiredScripts = [
      'start:ui',
      'generate-supergraph:ui',
      'test:ui-integration'
    ];

    let scriptScore = 0;
    requiredScripts.forEach(script => {
      if (scripts[script]) {
        logger.success(`✅ Script ${script} configured`);
        scriptScore++;
      } else {
        logger.warn(`⚠️ Script ${script} missing`);
      }
    });

    const depPercentage = Math.round((depScore / requiredDeps.length) * 100);
    const scriptPercentage = Math.round((scriptScore / requiredScripts.length) * 100);

    logger.info(`📊 Dependencies: ${depScore}/${requiredDeps.length} (${depPercentage}%)`);
    logger.info(`📊 Scripts: ${scriptScore}/${requiredScripts.length} (${scriptPercentage}%)`);

    return depPercentage >= 80 && scriptPercentage >= 80;
  } catch (error) {
    logger.error('❌ Failed to validate package.json:', error);
    return false;
  }
}

// Main validation function
async function validateUIDeployment() {
  const startTime = Date.now();
  const results = {
    schemaLoaded: false,
    syntaxValid: false,
    uiTypesComplete: false,
    resolversValid: false,
    gatewayValid: false,
    dependenciesValid: false
  };

  logger.info('🔍 Starting UI interaction schema validation');

  try {
    // Step 1: Load schema files
    const schemas = loadSchemaFiles();
    results.schemaLoaded = !!schemas;

    if (!schemas) {
      throw new Error('Failed to load required schema files');
    }

    // Step 2: Validate GraphQL syntax
    const orchestrationSyntax = validateGraphQLSyntax(schemas.orchestration || '', 'Orchestration Schema');
    const uiSyntax = validateGraphQLSyntax(schemas.uiInteraction, 'UI Interaction Schema');
    results.syntaxValid = orchestrationSyntax && uiSyntax;

    // Step 3: Validate UI interaction types
    results.uiTypesComplete = validateUIInteractionTypes(schemas.uiInteraction);

    // Step 4: Validate resolver implementation
    results.resolversValid = validateResolverFile();

    // Step 5: Validate gateway configuration
    results.gatewayValid = validateGatewayConfig();

    // Step 6: Validate dependencies
    results.dependenciesValid = validateDependencies();

  } catch (error) {
    logger.error('Validation process failed:', error);
  }

  // Results summary
  const duration = Date.now() - startTime;
  const passedChecks = Object.values(results).filter(Boolean).length;
  const totalChecks = Object.keys(results).length;

  logger.info(`\n📊 Validation Results (${duration}ms):`);
  logger.info(`   ✅ Passed: ${passedChecks}/${totalChecks} checks`);

  Object.entries(results).forEach(([check, passed]) => {
    const status = passed ? '✅' : '❌';
    const checkName = check.replace(/([A-Z])/g, ' $1').toLowerCase();
    logger.info(`   ${status} ${checkName}`);
  });

  if (passedChecks === totalChecks) {
    logger.success('\n🎉 All validation checks passed! UI interaction schema is ready for deployment.');
    logger.info('\n🚀 Next steps:');
    logger.info('   1. npm install (install new dependencies)');
    logger.info('   2. npm run generate-supergraph:ui (generate enhanced schema)');
    logger.info('   3. npm run test:ui-integration (run integration tests)');
    logger.info('   4. npm run start:ui (start enhanced gateway)');
    process.exit(0);
  } else {
    logger.warn(`\n⚠️ ${totalChecks - passedChecks} validation checks failed.`);
    logger.warn('Review the issues above before deploying.');
    process.exit(1);
  }
}

// Export for use as module
module.exports = {
  validateUIDeployment,
  loadSchemaFiles,
  validateGraphQLSyntax,
  validateUIInteractionTypes,
  validateResolverFile,
  validateGatewayConfig,
  validateDependencies
};

// Run validation if this is the main module
if (require.main === module) {
  validateUIDeployment().catch(error => {
    logger.error('Validation execution failed:', error);
    process.exit(1);
  });
}