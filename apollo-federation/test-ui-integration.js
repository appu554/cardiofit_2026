/**
 * Comprehensive UI Interaction Integration Tests
 * Tests Apollo Federation with Workflow UI interaction capabilities
 */

const { ApolloServer } = require('@apollo/server');
const { createTestClient } = require('apollo-server-testing');
const WebSocket = require('ws');
const fetch = (...args) => import('node-fetch').then(({default: fetch}) => fetch(...args));

// Test configuration
const GATEWAY_URL = 'http://localhost:4000';
const GRAPHQL_ENDPOINT = `${GATEWAY_URL}/graphql`;
const WEBSOCKET_ENDPOINT = `ws://localhost:4000/subscriptions`;

const logger = {
  info: (message) => console.log(`[TEST] ${new Date().toISOString()} - ${message}`),
  error: (message, error) => console.error(`[ERROR] ${new Date().toISOString()} - ${message}`, error?.message || error),
  success: (message) => console.log(`[SUCCESS] ${new Date().toISOString()} - ${message}`),
  warn: (message) => console.warn(`[WARN] ${new Date().toISOString()} - ${message}`)
};

// Test queries and mutations
const TEST_QUERIES = {
  // Health check query
  HEALTH_CHECK: `
    query HealthCheck {
      health {
        status
        service
        timestamp
      }
    }
  `,

  // Basic workflow orchestration
  EXECUTE_MEDICATION_WORKFLOW: `
    mutation ExecuteMedicationWorkflow($input: MedicationOrchestrationInput!) {
      orchestrateMedicationRequest(input: $input) {
        status
        correlationId
        medicationOrderId
        validation {
          verdict
          riskScore
          findingsCount
        }
        validationFindings {
          findingId
          severity
          description
          clinicalSignificance
        }
        overrideTokens
        errorCode
        errorMessage
      }
    }
  `,

  // UI notification update
  UPDATE_UI_NOTIFICATION: `
    mutation UpdateUINotification($workflowId: ID!, $notification: UINotificationInput!) {
      updateUINotification(workflowId: $workflowId, notification: $notification) {
        id
        workflowId
        status
        title
        message
        severity
        actions {
          id
          label
          type
          isEnabled
        }
        createdAt
      }
    }
  `,

  // Clinical override request
  REQUEST_CLINICAL_OVERRIDE: `
    mutation RequestClinicalOverride($workflowId: ID!, $validationId: String!, $request: OverrideRequestInput!) {
      requestClinicalOverride(
        workflowId: $workflowId
        validationId: $validationId
        request: $request
      ) {
        id
        workflowId
        status
        verdict
        requiredLevel
        requestedBy {
          id
          name
          role
        }
        expiresAt
      }
    }
  `,

  // Resolve clinical override
  RESOLVE_CLINICAL_OVERRIDE: `
    mutation ResolveClinicalOverride($sessionId: ID!, $decision: OverrideDecisionInput!) {
      resolveClinicalOverride(sessionId: $sessionId, decision: $decision) {
        medicationOrderId
        persistenceStatus
        eventPublicationStatus
        auditTrailId
        result
        processingTimeMs
      }
    }
  `,

  // Get workflow UI state
  GET_WORKFLOW_UI_STATE: `
    query GetWorkflowUIState($workflowId: ID!) {
      workflowUIState(workflowId: $workflowId) {
        workflowId
        currentPhase
        uiMode
        activeModals {
          id
          type
          title
          priority
        }
        pendingActions {
          id
          type
          description
          priority
        }
        lastUpdated
      }
    }
  `,

  // Get pending overrides
  GET_PENDING_OVERRIDES: `
    query GetPendingOverrides($clinicianId: ID, $urgency: ReviewUrgency) {
      pendingOverrides(clinicianId: $clinicianId, urgency: $urgency) {
        id
        workflowId
        status
        verdict
        requiredLevel
        requestedBy {
          id
          name
          role
        }
        expiresAt
      }
    }
  `
};

// Test subscriptions
const TEST_SUBSCRIPTIONS = {
  WORKFLOW_UI_UPDATES: `
    subscription WorkflowUIUpdates($workflowId: ID!) {
      workflowUIUpdates(workflowId: $workflowId) {
        workflowId
        updateType
        phase
        notification {
          id
          title
          message
          severity
        }
        timestamp
      }
    }
  `,

  OVERRIDE_REQUIRED: `
    subscription OverrideRequired($workflowId: ID!) {
      overrideRequired(workflowId: $workflowId) {
        workflowId
        validationId
        verdict
        criticalFindings {
          findingId
          severity
          description
        }
        overrideOptions {
          level
          available
          requirements
        }
        timeoutAt
      }
    }
  `
};

// Test data
const TEST_DATA = {
  sampleMedicationRequest: {
    patientId: "test-patient-123",
    correlationId: "test-correlation-456",
    medicationRequest: {
      medicationCode: "197361",
      medicationName: "Lisinopril",
      dosage: "10mg",
      frequency: "once daily",
      route: "oral",
      indication: "Hypertension"
    },
    clinicalIntent: {
      primaryIndication: "Hypertension management",
      targetOutcome: "Blood pressure control",
      treatmentGoal: "Systolic BP < 130 mmHg"
    },
    providerContext: {
      providerId: "test-provider-789",
      specialty: "Cardiology",
      organizationId: "test-org-001"
    }
  },

  sampleUINotification: {
    status: "ACTION_REQUIRED",
    title: "Clinical Override Required",
    message: "Validation found potential drug interaction requiring clinical review",
    severity: "WARNING",
    actions: [
      {
        id: "override",
        label: "Override",
        type: "OVERRIDE",
        isPrimary: true,
        payload: { overrideLevel: "CLINICAL_JUDGMENT" }
      },
      {
        id: "cancel",
        label: "Cancel",
        type: "CANCEL",
        isPrimary: false
      }
    ]
  },

  sampleOverrideRequest: {
    verdict: "WARNING",
    findings: [
      {
        findingId: "drug-interaction-001",
        severity: "HIGH",
        description: "Potential interaction between Lisinopril and patient's current medication",
        overridable: true,
        evidence: { interactionType: "pharmacokinetic", severity: "moderate" }
      }
    ],
    urgency: "ROUTINE"
  },

  sampleOverrideDecision: {
    decision: "OVERRIDE",
    overrideLevel: "CLINICAL_JUDGMENT",
    reason: {
      code: "CLINICAL_BENEFIT",
      category: "CLINICAL_BENEFIT",
      freeText: "Clinical benefit outweighs interaction risk for this patient"
    },
    clinicalJustification: "Patient has well-controlled BP with this medication, interaction risk is minimal given current renal function",
    acknowledgements: ["I understand the potential risks", "I will monitor for adverse effects"]
  }
};

// Helper functions
async function makeGraphQLRequest(query, variables = {}, headers = {}) {
  try {
    const response = await fetch(GRAPHQL_ENDPOINT, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-User-ID': 'test-user-123',
        'X-User-Role': 'ATTENDING',
        'X-Clinician-ID': 'test-clinician-456',
        'X-Department': 'CARDIOLOGY',
        'X-Authority-Level': 'ATTENDING',
        ...headers
      },
      body: JSON.stringify({ query, variables })
    });

    const result = await response.json();

    if (!response.ok || result.errors) {
      throw new Error(`GraphQL Error: ${JSON.stringify(result.errors || result)}`);
    }

    return result.data;
  } catch (error) {
    logger.error('GraphQL request failed:', error);
    throw error;
  }
}

async function testWebSocketConnection() {
  return new Promise((resolve, reject) => {
    const ws = new WebSocket(WEBSOCKET_ENDPOINT, 'graphql-ws');

    ws.on('open', () => {
      logger.success('WebSocket connection established');
      ws.close();
      resolve(true);
    });

    ws.on('error', (error) => {
      logger.error('WebSocket connection failed:', error);
      reject(error);
    });

    setTimeout(() => {
      ws.close();
      reject(new Error('WebSocket connection timeout'));
    }, 5000);
  });
}

// Test suites
async function testBasicConnectivity() {
  logger.info('Testing basic connectivity...');

  try {
    // Test HTTP endpoint
    const healthResponse = await fetch(`${GATEWAY_URL}/health`);
    if (!healthResponse.ok) {
      throw new Error(`Health check failed: ${healthResponse.status}`);
    }

    const healthData = await healthResponse.json();
    logger.success(`Gateway health check passed: ${healthData.status}`);

    // Test GraphQL endpoint
    const graphqlResponse = await makeGraphQLRequest(`
      query {
        __typename
      }
    `);

    logger.success('GraphQL endpoint is accessible');

    // Test WebSocket connection
    await testWebSocketConnection();

    return true;
  } catch (error) {
    logger.error('Basic connectivity test failed:', error);
    return false;
  }
}

async function testWorkflowOrchestration() {
  logger.info('Testing workflow orchestration...');

  try {
    const result = await makeGraphQLRequest(
      TEST_QUERIES.EXECUTE_MEDICATION_WORKFLOW,
      { input: TEST_DATA.sampleMedicationRequest }
    );

    if (!result.orchestrateMedicationRequest) {
      throw new Error('No orchestration result returned');
    }

    const orchestrationResult = result.orchestrateMedicationRequest;
    logger.success(`Workflow orchestration completed with status: ${orchestrationResult.status}`);

    // Check for validation results
    if (orchestrationResult.validation) {
      logger.info(`Validation verdict: ${orchestrationResult.validation.verdict}`);
      logger.info(`Risk score: ${orchestrationResult.validation.riskScore}`);
    }

    // Check for override tokens if warnings present
    if (orchestrationResult.overrideTokens && orchestrationResult.overrideTokens.length > 0) {
      logger.info(`Override tokens available: ${orchestrationResult.overrideTokens.length}`);
    }

    return orchestrationResult;
  } catch (error) {
    logger.error('Workflow orchestration test failed:', error);
    return null;
  }
}

async function testUIInteraction(workflowId) {
  logger.info('Testing UI interaction capabilities...');

  try {
    // Test UI notification update
    const notificationResult = await makeGraphQLRequest(
      TEST_QUERIES.UPDATE_UI_NOTIFICATION,
      {
        workflowId,
        notification: TEST_DATA.sampleUINotification
      }
    );

    if (!notificationResult.updateUINotification) {
      throw new Error('UI notification update failed');
    }

    const notification = notificationResult.updateUINotification;
    logger.success(`UI notification created: ${notification.id}`);

    // Test getting workflow UI state
    const stateResult = await makeGraphQLRequest(
      TEST_QUERIES.GET_WORKFLOW_UI_STATE,
      { workflowId }
    );

    if (stateResult.workflowUIState) {
      logger.success(`Workflow UI state retrieved: ${stateResult.workflowUIState.uiMode}`);
    }

    return notification;
  } catch (error) {
    logger.error('UI interaction test failed:', error);
    return null;
  }
}

async function testClinicalOverrideFlow(workflowId) {
  logger.info('Testing clinical override flow...');

  try {
    // Request clinical override
    const overrideResult = await makeGraphQLRequest(
      TEST_QUERIES.REQUEST_CLINICAL_OVERRIDE,
      {
        workflowId,
        validationId: 'test-validation-789',
        request: TEST_DATA.sampleOverrideRequest
      }
    );

    if (!overrideResult.requestClinicalOverride) {
      throw new Error('Clinical override request failed');
    }

    const overrideSession = overrideResult.requestClinicalOverride;
    logger.success(`Override session created: ${overrideSession.id}`);

    // Resolve the override
    const resolveResult = await makeGraphQLRequest(
      TEST_QUERIES.RESOLVE_CLINICAL_OVERRIDE,
      {
        sessionId: overrideSession.id,
        decision: TEST_DATA.sampleOverrideDecision
      }
    );

    if (!resolveResult.resolveClinicalOverride) {
      throw new Error('Override resolution failed');
    }

    const commitResult = resolveResult.resolveClinicalOverride;
    logger.success(`Override resolved with result: ${commitResult.result}`);

    return { overrideSession, commitResult };
  } catch (error) {
    logger.error('Clinical override flow test failed:', error);
    return null;
  }
}

async function testPendingOverrides() {
  logger.info('Testing pending overrides query...');

  try {
    const result = await makeGraphQLRequest(
      TEST_QUERIES.GET_PENDING_OVERRIDES,
      {
        clinicianId: 'test-clinician-456',
        urgency: 'ROUTINE'
      }
    );

    if (!Array.isArray(result.pendingOverrides)) {
      throw new Error('Pending overrides query failed');
    }

    logger.success(`Retrieved ${result.pendingOverrides.length} pending overrides`);
    return result.pendingOverrides;
  } catch (error) {
    logger.error('Pending overrides test failed:', error);
    return null;
  }
}

// Main test execution
async function runAllTests() {
  const startTime = Date.now();
  const results = {
    connectivity: false,
    orchestration: false,
    uiInteraction: false,
    clinicalOverride: false,
    pendingOverrides: false
  };

  logger.info('🧪 Starting comprehensive UI interaction integration tests');

  try {
    // Test 1: Basic connectivity
    results.connectivity = await testBasicConnectivity();

    if (!results.connectivity) {
      throw new Error('Basic connectivity failed - aborting remaining tests');
    }

    // Test 2: Workflow orchestration
    const orchestrationResult = await testWorkflowOrchestration();
    results.orchestration = !!orchestrationResult;

    let workflowId = 'test-workflow-' + Date.now();
    if (orchestrationResult && orchestrationResult.correlationId) {
      workflowId = orchestrationResult.correlationId;
    }

    // Test 3: UI interaction
    const uiResult = await testUIInteraction(workflowId);
    results.uiInteraction = !!uiResult;

    // Test 4: Clinical override flow
    const overrideResult = await testClinicalOverrideFlow(workflowId);
    results.clinicalOverride = !!overrideResult;

    // Test 5: Pending overrides
    const pendingResult = await testPendingOverrides();
    results.pendingOverrides = !!pendingResult;

  } catch (error) {
    logger.error('Test suite execution failed:', error);
  }

  // Results summary
  const duration = Date.now() - startTime;
  const passedTests = Object.values(results).filter(Boolean).length;
  const totalTests = Object.keys(results).length;

  logger.info(`\n📊 Test Results Summary (${duration}ms):`);
  logger.info(`   ✅ Passed: ${passedTests}/${totalTests} tests`);

  Object.entries(results).forEach(([test, passed]) => {
    const status = passed ? '✅' : '❌';
    logger.info(`   ${status} ${test}`);
  });

  if (passedTests === totalTests) {
    logger.success('\n🎉 All UI interaction tests passed! Apollo Federation is ready for clinical use.');
    logger.info('\n🚀 Ready to deploy:');
    logger.info('   1. Start gateway: npm run start:ui');
    logger.info('   2. Frontend integration: Connect UI components to GraphQL endpoints');
    logger.info('   3. Monitor: Set up monitoring for real-time subscriptions');
  } else {
    logger.warn(`\n⚠️ ${totalTests - passedTests} tests failed. Review logs above for details.`);
    process.exit(1);
  }

  return results;
}

// Export for use as module
module.exports = {
  runAllTests,
  testBasicConnectivity,
  testWorkflowOrchestration,
  testUIInteraction,
  testClinicalOverrideFlow,
  TEST_QUERIES,
  TEST_DATA
};

// Run tests if this is the main module
if (require.main === module) {
  runAllTests().catch(error => {
    logger.error('Test execution failed:', error);
    process.exit(1);
  });
}