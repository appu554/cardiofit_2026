#!/usr/bin/env node

/**
 * KB-2 Clinical Context Apollo Federation Integration Test
 * 
 * This script tests the complete integration of KB-2 Clinical Context service
 * with Apollo Federation, including all GraphQL operations and Patient extensions.
 */

const axios = require('axios');
const { performance } = require('perf_hooks');

// Configuration
const APOLLO_GATEWAY_URL = process.env.APOLLO_GATEWAY_URL || 'http://localhost:4000/graphql';
const KB2_DIRECT_URL = process.env.KB2_CLINICAL_CONTEXT_URL || 'http://localhost:8082';
const TEST_TIMEOUT = 30000;

// Test configuration
const TEST_CONFIG = {
  enableDirectServiceTests: true,
  enableFederationTests: true,
  enablePerformanceTests: true,
  enablePatientExtensionTests: true,
  testPatientId: 'test-patient-kb2-001'
};

// Logger utility
const logger = {
  info: (message, data) => console.log(`[TEST][INFO] ${new Date().toISOString()} - ${message}`, data ? JSON.stringify(data, null, 2) : ''),
  error: (message, error) => console.error(`[TEST][ERROR] ${new Date().toISOString()} - ${message}`, error?.message || error),
  success: (message, data) => console.log(`[TEST][SUCCESS] ✅ ${message}`, data || ''),
  warn: (message, data) => console.warn(`[TEST][WARN] ⚠️  ${message}`, data || '')
};

// Test data
const TEST_PATIENT_DATA = {
  id: TEST_CONFIG.testPatientId,
  age: 65,
  gender: 'male',
  conditions: ['diabetes', 'hypertension', 'hyperlipidemia'],
  medications: ['metformin', 'lisinopril', 'atorvastatin'],
  labs: [
    {
      name: 'HbA1c',
      value: 8.5,
      unit: '%',
      referenceRange: '4.0-5.6',
      testDate: '2024-09-01T10:00:00Z'
    },
    {
      name: 'Total Cholesterol',
      value: 250,
      unit: 'mg/dL',
      referenceRange: '<200',
      testDate: '2024-09-01T10:00:00Z'
    }
  ],
  vitals: [
    {
      name: 'Blood Pressure Systolic',
      value: 145,
      unit: 'mmHg',
      measurementDate: '2024-09-01T10:00:00Z'
    },
    {
      name: 'Blood Pressure Diastolic',
      value: 90,
      unit: 'mmHg',
      measurementDate: '2024-09-01T10:00:00Z'
    }
  ],
  procedures: ['ecg', 'chest_xray'],
  allergies: ['penicillin'],
  familyHistory: ['diabetes', 'coronary_artery_disease'],
  socialHistory: {
    smokingStatus: 'never',
    alcoholUse: 'occasional',
    exerciseFrequency: 'weekly',
    dietaryPatterns: 'standard'
  }
};

// Test suite
class KB2IntegrationTest {
  constructor() {
    this.testResults = {
      passed: 0,
      failed: 0,
      total: 0,
      errors: []
    };
    this.startTime = performance.now();
  }

  async runTest(testName, testFunction) {
    this.testResults.total++;
    logger.info(`Running test: ${testName}`);
    
    const testStartTime = performance.now();
    
    try {
      await testFunction();
      this.testResults.passed++;
      const duration = Math.round(performance.now() - testStartTime);
      logger.success(`${testName} (${duration}ms)`);
    } catch (error) {
      this.testResults.failed++;
      this.testResults.errors.push({
        test: testName,
        error: error.message,
        stack: error.stack
      });
      logger.error(`${testName} failed:`, error);
    }
  }

  async makeGraphQLRequest(url, query, variables = {}) {
    try {
      const response = await axios.post(url, {
        query,
        variables
      }, {
        headers: {
          'Content-Type': 'application/json',
          'X-Test-ID': `kb2-integration-${Date.now()}`,
          'Authorization': 'Bearer test-token',
          'X-User-ID': 'test-user-001',
          'X-User-Role': 'clinician'
        },
        timeout: TEST_TIMEOUT
      });

      if (response.data.errors) {
        throw new Error(`GraphQL errors: ${JSON.stringify(response.data.errors)}`);
      }

      return response.data.data;
    } catch (error) {
      if (error.response) {
        throw new Error(`HTTP ${error.response.status}: ${JSON.stringify(error.response.data)}`);
      }
      throw error;
    }
  }

  async makeRestRequest(url, method = 'GET', data = null) {
    try {
      const response = await axios({
        method,
        url,
        data,
        headers: {
          'Content-Type': 'application/json',
          'X-Test-ID': `kb2-integration-${Date.now()}`,
          'Authorization': 'Bearer test-token',
          'X-User-ID': 'test-user-001'
        },
        timeout: TEST_TIMEOUT
      });

      return response.data;
    } catch (error) {
      if (error.response) {
        throw new Error(`HTTP ${error.response.status}: ${JSON.stringify(error.response.data)}`);
      }
      throw error;
    }
  }

  // Test 1: Direct KB-2 Service Health
  async testKB2ServiceHealth() {
    if (!TEST_CONFIG.enableDirectServiceTests) return;
    
    const healthData = await this.makeRestRequest(`${KB2_DIRECT_URL}/health`);
    
    if (healthData.status !== 'healthy' && healthData.status !== 'ok') {
      throw new Error(`KB-2 service unhealthy: ${healthData.status}`);
    }
  }

  // Test 2: Direct KB-2 Phenotype Evaluation
  async testDirectPhenotypeEvaluation() {
    if (!TEST_CONFIG.enableDirectServiceTests) return;
    
    const requestBody = {
      patients: [TEST_PATIENT_DATA],
      include_explanation: false,
      include_implications: true,
      confidence_threshold: 0.7
    };

    const result = await this.makeRestRequest(
      `${KB2_DIRECT_URL}/v1/phenotypes/evaluate`,
      'POST',
      requestBody
    );

    if (!result.results || result.results.length === 0) {
      throw new Error('No phenotype evaluation results returned');
    }

    if (!result.processing_time) {
      throw new Error('Processing time not provided');
    }

    const patientResult = result.results[0];
    if (patientResult.patient_id !== TEST_PATIENT_DATA.id) {
      throw new Error('Patient ID mismatch in results');
    }
  }

  // Test 3: Federation Phenotype Evaluation
  async testFederationPhenotypeEvaluation() {
    if (!TEST_CONFIG.enableFederationTests) return;
    
    const query = `
      mutation EvaluatePhenotypes($input: PhenotypeEvaluationInput!) {
        evaluatePatientPhenotypes(input: $input) {
          results {
            patientId
            phenotypes {
              id
              name
              category
              matched
              confidence
              implications {
                type
                severity
                description
                recommendations
              }
            }
            evaluationSummary {
              totalPhenotypes
              matchedPhenotypes
              averageConfidence
              processingTime
            }
          }
          processingTime
          slaCompliant
          metadata {
            cacheHitRate
            componentsProcessed
          }
        }
      }
    `;

    const variables = {
      input: {
        patients: [TEST_PATIENT_DATA],
        includeExplanation: false,
        includeImplications: true,
        confidenceThreshold: 0.7
      }
    };

    const result = await this.makeGraphQLRequest(APOLLO_GATEWAY_URL, query, variables);
    
    if (!result.evaluatePatientPhenotypes) {
      throw new Error('No phenotype evaluation results from federation');
    }

    const evaluation = result.evaluatePatientPhenotypes;
    if (evaluation.results.length === 0) {
      throw new Error('No patient results from federation');
    }

    if (!evaluation.slaCompliant) {
      logger.warn('SLA not compliant', { processingTime: evaluation.processingTime });
    }
  }

  // Test 4: Federation Risk Assessment
  async testFederationRiskAssessment() {
    if (!TEST_CONFIG.enableFederationTests) return;
    
    const query = `
      mutation AssessRisk($input: RiskAssessmentInput!) {
        assessPatientRisk(input: $input) {
          patientId
          riskAssessments {
            id
            model
            category
            score
            category_result
            recommendations {
              priority
              action
              rationale
            }
            riskFactors {
              name
              value
              contribution
              modifiable
              severity
            }
          }
          overallRiskProfile {
            overallRisk
            primaryConcerns
            recommendedActions
          }
          processingTime
          slaCompliant
        }
      }
    `;

    const variables = {
      input: {
        patientId: TEST_PATIENT_DATA.id,
        patientData: TEST_PATIENT_DATA,
        riskCategories: ['CARDIOVASCULAR', 'DIABETES'],
        includeFactors: true,
        includeRecommendations: true
      }
    };

    const result = await this.makeGraphQLRequest(APOLLO_GATEWAY_URL, query, variables);
    
    if (!result.assessPatientRisk) {
      throw new Error('No risk assessment results from federation');
    }

    const assessment = result.assessPatientRisk;
    if (assessment.patientId !== TEST_PATIENT_DATA.id) {
      throw new Error('Patient ID mismatch in risk assessment');
    }

    if (assessment.riskAssessments.length === 0) {
      throw new Error('No risk assessments returned');
    }
  }

  // Test 5: Federation Treatment Preferences
  async testFederationTreatmentPreferences() {
    if (!TEST_CONFIG.enableFederationTests) return;
    
    const query = `
      mutation GetTreatmentPreferences($input: TreatmentPreferencesInput!) {
        getPatientTreatmentPreferences(input: $input) {
          patientId
          condition
          preferences {
            id
            condition
            firstLine {
              medication {
                name
                genericName
                drugClass
              }
              preferenceScore
              reasons
            }
            rationale
            guidelineSource
            confidenceLevel
          }
          processingTime
        }
      }
    `;

    const variables = {
      input: {
        patientId: TEST_PATIENT_DATA.id,
        condition: 'diabetes',
        patientData: TEST_PATIENT_DATA,
        preferenceProfile: {
          onceDailyPreferred: true,
          injectableAccepted: false,
          costConscious: true
        },
        includeAlternatives: true
      }
    };

    const result = await this.makeGraphQLRequest(APOLLO_GATEWAY_URL, query, variables);
    
    if (!result.getPatientTreatmentPreferences) {
      throw new Error('No treatment preferences from federation');
    }

    const preferences = result.getPatientTreatmentPreferences;
    if (preferences.patientId !== TEST_PATIENT_DATA.id) {
      throw new Error('Patient ID mismatch in treatment preferences');
    }
  }

  // Test 6: Federation Clinical Context Assembly
  async testFederationContextAssembly() {
    if (!TEST_CONFIG.enableFederationTests) return;
    
    const query = `
      mutation AssembleContext($input: ClinicalContextInput!) {
        assemblePatientContext(input: $input) {
          patientId
          context {
            patientId
            phenotypes {
              id
              name
              matched
              confidence
            }
            riskAssessments {
              model
              category
              score
            }
            treatmentPreferences {
              condition
              firstLine {
                medication {
                  name
                }
              }
            }
            contextMetadata {
              processingTime
              slaCompliant
              dataCompleteness
              confidenceScore
              componentsEvaluated
            }
          }
          warnings {
            severity
            category
            message
          }
          recommendations {
            priority
            category
            recommendation
            rationale
          }
          processingTime
          slaCompliant
        }
      }
    `;

    const variables = {
      input: {
        patientId: TEST_PATIENT_DATA.id,
        patientData: TEST_PATIENT_DATA,
        detailLevel: 'COMPREHENSIVE',
        includePhenotypes: true,
        includeRisks: true,
        includeTreatments: true,
        useCache: true
      }
    };

    const result = await this.makeGraphQLRequest(APOLLO_GATEWAY_URL, query, variables);
    
    if (!result.assemblePatientContext) {
      throw new Error('No clinical context assembly from federation');
    }

    const context = result.assemblePatientContext;
    if (context.patientId !== TEST_PATIENT_DATA.id) {
      throw new Error('Patient ID mismatch in context assembly');
    }

    if (!context.context.contextMetadata) {
      throw new Error('Context metadata missing');
    }
  }

  // Test 7: Patient Extension - Clinical Context
  async testPatientExtensionClinicalContext() {
    if (!TEST_CONFIG.enablePatientExtensionTests) return;
    
    const query = `
      query GetPatientWithContext($patientId: ID!) {
        patient(id: $patientId) {
          id
          name {
            family
            given
          }
          clinicalContext {
            patientId
            phenotypes {
              id
              name
              matched
              confidence
            }
            riskAssessments {
              model
              category
              score
              category_result
            }
            treatmentPreferences {
              condition
              firstLine {
                medication {
                  name
                  genericName
                }
              }
            }
            contextMetadata {
              processingTime
              slaCompliant
              dataCompleteness
              confidenceScore
            }
          }
        }
      }
    `;

    const variables = {
      patientId: TEST_PATIENT_DATA.id
    };

    const result = await this.makeGraphQLRequest(APOLLO_GATEWAY_URL, query, variables);
    
    if (!result.patient) {
      throw new Error('Patient not found in federation query');
    }

    const patient = result.patient;
    if (patient.id !== TEST_PATIENT_DATA.id) {
      throw new Error('Patient ID mismatch in extension query');
    }

    // Clinical context might be null if patient doesn't exist in patient service
    if (patient.clinicalContext) {
      if (patient.clinicalContext.patientId !== TEST_PATIENT_DATA.id) {
        throw new Error('Patient ID mismatch in clinical context extension');
      }
    } else {
      logger.warn('Clinical context is null - patient may not exist in patient service');
    }
  }

  // Test 8: Patient Extension - Phenotypes Only
  async testPatientExtensionPhenotypes() {
    if (!TEST_CONFIG.enablePatientExtensionTests) return;
    
    const query = `
      query GetPatientPhenotypes($patientId: ID!) {
        patient(id: $patientId) {
          id
          phenotypes {
            id
            name
            category
            matched
            confidence
            implications {
              type
              severity
              description
            }
          }
        }
      }
    `;

    const variables = {
      patientId: TEST_PATIENT_DATA.id
    };

    const result = await this.makeGraphQLRequest(APOLLO_GATEWAY_URL, query, variables);
    
    if (!result.patient) {
      throw new Error('Patient not found for phenotypes query');
    }

    const patient = result.patient;
    if (patient.phenotypes && patient.phenotypes.length > 0) {
      logger.info(`Found ${patient.phenotypes.length} phenotypes for patient`);
    } else {
      logger.warn('No phenotypes found for patient extension');
    }
  }

  // Test 9: Available Phenotypes Query
  async testAvailablePhenotypes() {
    if (!TEST_CONFIG.enableFederationTests) return;
    
    const query = `
      query GetAvailablePhenotypes($category: String) {
        availablePhenotypes(category: $category) {
          id
          name
          category
          description
          celRule
          requiredData
          priority
          active
        }
      }
    `;

    const result = await this.makeGraphQLRequest(APOLLO_GATEWAY_URL, query, {});
    
    if (!result.availablePhenotypes || result.availablePhenotypes.length === 0) {
      throw new Error('No available phenotypes returned from federation');
    }

    logger.info(`Found ${result.availablePhenotypes.length} available phenotypes`);
  }

  // Test 10: Performance Test
  async testPerformance() {
    if (!TEST_CONFIG.enablePerformanceTests) return;
    
    const iterations = 5;
    const results = [];

    for (let i = 0; i < iterations; i++) {
      const startTime = performance.now();
      
      await this.testFederationContextAssembly();
      
      const endTime = performance.now();
      results.push(endTime - startTime);
    }

    const avgTime = results.reduce((a, b) => a + b, 0) / results.length;
    const maxTime = Math.max(...results);
    const minTime = Math.min(...results);

    logger.info(`Performance results (${iterations} iterations):`, {
      average: `${Math.round(avgTime)}ms`,
      min: `${Math.round(minTime)}ms`,
      max: `${Math.round(maxTime)}ms`,
      all: results.map(r => `${Math.round(r)}ms`)
    });

    // Performance thresholds
    if (avgTime > 5000) { // 5 seconds
      throw new Error(`Average response time too high: ${Math.round(avgTime)}ms`);
    }
  }

  // Run all tests
  async runAllTests() {
    logger.info('Starting KB-2 Clinical Context Apollo Federation Integration Tests');
    logger.info('Test Configuration:', TEST_CONFIG);

    try {
      // Direct service tests
      await this.runTest('KB-2 Service Health Check', () => this.testKB2ServiceHealth());
      await this.runTest('Direct Phenotype Evaluation', () => this.testDirectPhenotypeEvaluation());

      // Federation tests
      await this.runTest('Federation Phenotype Evaluation', () => this.testFederationPhenotypeEvaluation());
      await this.runTest('Federation Risk Assessment', () => this.testFederationRiskAssessment());
      await this.runTest('Federation Treatment Preferences', () => this.testFederationTreatmentPreferences());
      await this.runTest('Federation Clinical Context Assembly', () => this.testFederationContextAssembly());
      
      // Patient extension tests
      await this.runTest('Patient Extension - Clinical Context', () => this.testPatientExtensionClinicalContext());
      await this.runTest('Patient Extension - Phenotypes', () => this.testPatientExtensionPhenotypes());
      
      // Additional tests
      await this.runTest('Available Phenotypes Query', () => this.testAvailablePhenotypes());
      await this.runTest('Performance Test', () => this.testPerformance());

    } catch (error) {
      logger.error('Test execution error:', error);
    }

    // Report results
    const totalTime = Math.round(performance.now() - this.startTime);
    const passRate = Math.round((this.testResults.passed / this.testResults.total) * 100);

    logger.info('\n=== KB-2 Integration Test Results ===');
    logger.info(`Total Tests: ${this.testResults.total}`);
    logger.info(`Passed: ${this.testResults.passed}`);
    logger.info(`Failed: ${this.testResults.failed}`);
    logger.info(`Pass Rate: ${passRate}%`);
    logger.info(`Total Time: ${totalTime}ms`);

    if (this.testResults.errors.length > 0) {
      logger.error('\n=== Test Errors ===');
      this.testResults.errors.forEach(error => {
        logger.error(`${error.test}: ${error.error}`);
      });
    }

    if (this.testResults.failed === 0) {
      logger.success('\n🎉 All KB-2 integration tests passed!');
      process.exit(0);
    } else {
      logger.error(`\n❌ ${this.testResults.failed} test(s) failed`);
      process.exit(1);
    }
  }
}

// Run tests if this file is executed directly
if (require.main === module) {
  const testSuite = new KB2IntegrationTest();
  testSuite.runAllTests().catch(error => {
    logger.error('Test suite execution failed:', error);
    process.exit(1);
  });
}

module.exports = KB2IntegrationTest;