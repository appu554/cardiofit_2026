/**
 * Test Suite for Analytics Dashboard API
 *
 * Tests Module 6 GraphQL queries against the Analytics Service
 *
 * Prerequisites:
 * 1. Redis running on localhost:6379
 * 2. Analytics service running: npm run start:analytics
 * 3. Flink Module 6 producing data to Kafka topics
 *
 * Usage:
 *   node test-analytics-dashboard.js
 */

const axios = require('axios');

const ANALYTICS_SERVICE_URL = process.env.ANALYTICS_SERVICE_URL || 'http://localhost:8050/graphql';

// Test utilities
const logger = {
  info: (msg) => console.log(`ℹ️  ${msg}`),
  success: (msg) => console.log(`✅ ${msg}`),
  error: (msg) => console.error(`❌ ${msg}`),
  warn: (msg) => console.warn(`⚠️  ${msg}`)
};

async function query(gql, variables = {}) {
  try {
    const response = await axios.post(ANALYTICS_SERVICE_URL, {
      query: gql,
      variables
    }, {
      headers: {
        'Content-Type': 'application/json'
      }
    });

    if (response.data.errors) {
      logger.error(`GraphQL Errors: ${JSON.stringify(response.data.errors, null, 2)}`);
      return null;
    }

    return response.data.data;
  } catch (error) {
    logger.error(`Request failed: ${error.message}`);
    if (error.response) {
      logger.error(`Response: ${JSON.stringify(error.response.data, null, 2)}`);
    }
    return null;
  }
}

// ====================  Test Cases ====================

async function testHealthCheck() {
  logger.info('Testing analytics health check...');

  const gql = `
    query {
      analyticsHealth {
        status
        timestamp
        redisConnected
        postgresConnected
        kafkaConnected
      }
    }
  `;

  const result = await query(gql);
  if (result?.analyticsHealth) {
    logger.success('Health check passed');
    console.log(JSON.stringify(result.analyticsHealth, null, 2));
    return result.analyticsHealth.redisConnected;
  }

  logger.error('Health check failed');
  return false;
}

async function testHospitalKPIs() {
  logger.info('Testing hospital-wide KPIs...');

  const gql = `
    query {
      hospitalKPIs {
        timestamp
        totalPatients
        highRiskPatients
        criticalPatients
        avgMortalityRisk
        avgSepsisRisk
        avgReadmissionRisk
        activeAlerts
        criticalAlerts
      }
    }
  `;

  const result = await query(gql);
  if (result?.hospitalKPIs) {
    logger.success('Hospital KPIs retrieved');
    console.log(JSON.stringify(result.hospitalKPIs, null, 2));
    return true;
  }

  logger.error('Failed to retrieve hospital KPIs');
  return false;
}

async function testDepartmentMetrics() {
  logger.info('Testing department metrics...');

  const gql = `
    query {
      allDepartmentMetrics {
        department
        timestamp
        totalPatients
        highRiskPatients
        criticalPatients
        avgMortalityRisk
        avgSepsisRisk
        riskDistribution {
          LOW
          MODERATE
          HIGH
          CRITICAL
        }
        departmentRiskLevel
        highRiskPercentage
        criticalPercentage
        overallRiskScore
        requiresImmediateAttention
      }
    }
  `;

  const result = await query(gql);
  if (result?.allDepartmentMetrics) {
    logger.success(`Retrieved metrics for ${result.allDepartmentMetrics.length} departments`);
    result.allDepartmentMetrics.forEach(dept => {
      console.log(`\nDepartment: ${dept.department}`);
      console.log(`  Patients: ${dept.totalPatients} (High Risk: ${dept.highRiskPatients}, Critical: ${dept.criticalPatients})`);
      console.log(`  Risk Level: ${dept.departmentRiskLevel}`);
      console.log(`  Avg Mortality: ${(dept.avgMortalityRisk * 100).toFixed(2)}%`);
      console.log(`  Avg Sepsis: ${(dept.avgSepsisRisk * 100).toFixed(2)}%`);
    });
    return true;
  }

  logger.error('Failed to retrieve department metrics');
  return false;
}

async function testSpecificDepartment(department = 'ICU') {
  logger.info(`Testing specific department: ${department}...`);

  const gql = `
    query GetDepartment($dept: String!) {
      departmentMetrics(department: $dept) {
        department
        totalPatients
        highRiskPatients
        criticalPatients
        departmentRiskLevel
        requiresImmediateAttention
      }
    }
  `;

  const result = await query(gql, { dept: department });
  if (result?.departmentMetrics) {
    logger.success(`Retrieved metrics for ${department}`);
    console.log(JSON.stringify(result.departmentMetrics, null, 2));
    return true;
  }

  logger.warn(`No metrics found for department: ${department}`);
  return false;
}

async function testHighRiskPatients() {
  logger.info('Testing high-risk patients query...');

  const gql = `
    query {
      highRiskPatients(limit: 10) {
        patientId
        department
        mortalityRisk
        sepsisRisk
        readmissionRisk
        overallRiskScore
        riskLevel
        isHighRisk
        isCritical
        lastUpdated
      }
    }
  `;

  const result = await query(gql);
  if (result?.highRiskPatients) {
    logger.success(`Retrieved ${result.highRiskPatients.length} high-risk patients`);
    result.highRiskPatients.slice(0, 3).forEach(patient => {
      console.log(`\nPatient: ${patient.patientId}`);
      console.log(`  Department: ${patient.department}`);
      console.log(`  Risk Level: ${patient.riskLevel} (Score: ${(patient.overallRiskScore * 100).toFixed(2)}%)`);
      console.log(`  Mortality: ${(patient.mortalityRisk * 100).toFixed(2)}%`);
      console.log(`  Sepsis: ${(patient.sepsisRisk * 100).toFixed(2)}%`);
    });
    return true;
  }

  logger.error('Failed to retrieve high-risk patients');
  return false;
}

async function testAlertMetrics() {
  logger.info('Testing alert metrics...');

  const now = new Date().toISOString();
  const oneHourAgo = new Date(Date.now() - 3600000).toISOString();

  const gql = `
    query GetAlertMetrics($start: String!, $end: String!) {
      alertMetrics(startTime: $start, endTime: $end) {
        timestamp
        totalAlerts
        criticalAlerts
        warningAlerts
        infoAlerts
        avgResolutionTime
      }
    }
  `;

  const result = await query(gql, { start: oneHourAgo, end: now });
  if (result?.alertMetrics) {
    logger.success('Alert metrics retrieved');
    console.log(JSON.stringify(result.alertMetrics, null, 2));
    return true;
  }

  logger.error('Failed to retrieve alert metrics');
  return false;
}

// ====================  Main Test Runner ====================

async function runTests() {
  console.log('\n═══════════════════════════════════════════════════');
  console.log('  Module 6 Analytics Dashboard API Test Suite');
  console.log('═══════════════════════════════════════════════════\n');

  const results = {
    total: 0,
    passed: 0,
    failed: 0
  };

  const tests = [
    { name: 'Health Check', fn: testHealthCheck },
    { name: 'Hospital KPIs', fn: testHospitalKPIs },
    { name: 'Department Metrics', fn: testDepartmentMetrics },
    { name: 'Specific Department', fn: testSpecificDepartment },
    { name: 'High-Risk Patients', fn: testHighRiskPatients },
    { name: 'Alert Metrics', fn: testAlertMetrics }
  ];

  for (const test of tests) {
    results.total++;
    console.log(`\n${'─'.repeat(50)}`);
    console.log(`Test ${results.total}/${tests.length}: ${test.name}`);
    console.log('─'.repeat(50));

    try {
      const passed = await test.fn();
      if (passed) {
        results.passed++;
      } else {
        results.failed++;
      }
    } catch (error) {
      logger.error(`Test threw exception: ${error.message}`);
      results.failed++;
    }

    await new Promise(resolve => setTimeout(resolve, 500)); // Brief pause between tests
  }

  // Summary
  console.log('\n═══════════════════════════════════════════════════');
  console.log('  Test Results Summary');
  console.log('═══════════════════════════════════════════════════');
  console.log(`Total Tests: ${results.total}`);
  console.log(`✅ Passed: ${results.passed}`);
  console.log(`❌ Failed: ${results.failed}`);
  console.log(`Success Rate: ${((results.passed / results.total) * 100).toFixed(2)}%`);
  console.log('═══════════════════════════════════════════════════\n');

  if (results.failed > 0) {
    logger.warn('Some tests failed. Check logs above for details.');
    process.exit(1);
  } else {
    logger.success('All tests passed! 🎉');
    process.exit(0);
  }
}

// Run tests
runTests().catch(error => {
  logger.error(`Fatal error: ${error.message}`);
  process.exit(1);
});
