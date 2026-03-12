// KB7-AU Load Test Suite
// k6 load testing for terminology service endpoints
//
// Run: k6 run --vus 50 --duration 5m load_test.js
// With env: k6 run --env API_URL=http://localhost:8087 load_test.js

import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// ============================================================================
// Custom Metrics
// ============================================================================
const errorRate = new Rate('kb7_errors');
const subsumptionLatency = new Trend('kb7_subsumption_latency_ms');
const valuesetLatency = new Trend('kb7_valueset_latency_ms');
const containsLatency = new Trend('kb7_contains_latency_ms');
const ruleValidationLatency = new Trend('kb7_rule_validation_latency_ms');
const healthCheckLatency = new Trend('kb7_health_latency_ms');

const subsumptionRequests = new Counter('kb7_subsumption_requests');
const valuesetRequests = new Counter('kb7_valueset_requests');
const containsRequests = new Counter('kb7_contains_requests');

// ============================================================================
// Test Configuration
// ============================================================================
export const options = {
  stages: [
    { duration: '30s', target: 10 },   // Warm up
    { duration: '1m', target: 25 },    // Ramp to 25 VUs
    { duration: '2m', target: 50 },    // Stay at 50 VUs (normal load)
    { duration: '1m', target: 75 },    // Ramp to 75 VUs (peak load)
    { duration: '2m', target: 50 },    // Back to normal
    { duration: '30s', target: 0 },    // Ramp down
  ],
  thresholds: {
    // Response time thresholds
    'http_req_duration': ['p(95)<500', 'p(99)<1000'],
    'kb7_subsumption_latency_ms': ['p(95)<200', 'p(99)<500'],
    'kb7_valueset_latency_ms': ['p(95)<300', 'p(99)<800'],
    'kb7_contains_latency_ms': ['p(95)<150', 'p(99)<400'],
    'kb7_rule_validation_latency_ms': ['p(95)<250', 'p(99)<600'],

    // Error rate thresholds
    'http_req_failed': ['rate<0.01'],  // <1% error rate
    'kb7_errors': ['rate<0.02'],       // <2% custom error rate
  },
};

// ============================================================================
// Test Data - AU Clinical SNOMED Codes
// ============================================================================
const BASE_URL = __ENV.API_URL || 'http://localhost:8087';

// AU Clinical SNOMED concepts for testing
const AU_CLINICAL_CONCEPTS = {
  // Sepsis-related
  sepsis: [
    { code: '91302008', display: 'Sepsis' },
    { code: '10001005', display: 'Bacterial sepsis' },
    { code: '238150007', display: 'Severe sepsis' },
    { code: '76571007', display: 'Septic shock' },
  ],
  // Renal-related
  renal: [
    { code: '14669001', display: 'Acute kidney injury' },
    { code: '90708001', display: 'Kidney disease' },
    { code: '709044004', display: 'CKD' },
    { code: '46177005', display: 'End-stage renal disease' },
  ],
  // Cardiac-related
  cardiac: [
    { code: '84114007', display: 'Heart failure' },
    { code: '49436004', display: 'Atrial fibrillation' },
    { code: '22298006', display: 'Myocardial infarction' },
    { code: '38341003', display: 'Hypertension' },
  ],
  // Metabolic
  metabolic: [
    { code: '73211009', display: 'Diabetes mellitus' },
    { code: '44054006', display: 'Type 2 diabetes' },
    { code: '46635009', display: 'Type 1 diabetes' },
    { code: '190330002', display: 'Hypoglycemia' },
  ],
  // Medications
  medications: [
    { code: '387517004', display: 'Paracetamol' },
    { code: '372735009', display: 'Vancomycin' },
    { code: '373265006', display: 'Analgesic' },
    { code: '373297006', display: 'Anti-infective' },
  ],
};

// Known subsumption relationships for testing
const SUBSUMPTION_TEST_CASES = [
  // Positive cases (should return true)
  { child: '91302008', parent: '404684003', expected: true, desc: 'Sepsis IS-A Clinical finding' },
  { child: '10001005', parent: '91302008', expected: true, desc: 'Bacterial sepsis IS-A Sepsis' },
  { child: '14669001', parent: '90708001', expected: true, desc: 'AKI IS-A Kidney disease' },
  { child: '44054006', parent: '73211009', expected: true, desc: 'Type 2 DM IS-A Diabetes' },
  { child: '387517004', parent: '373265006', expected: true, desc: 'Paracetamol IS-A Analgesic' },

  // Negative cases (should return false)
  { child: '91302008', parent: '71388002', expected: false, desc: 'Sepsis NOT-A Procedure' },
  { child: '73211009', parent: '84114007', expected: false, desc: 'Diabetes NOT-A Heart failure' },
];

// Value sets for testing
const VALUE_SETS = [
  'SepsisConditions',
  'RenalConditions',
  'CardiacConditions',
  'DiabetesConditions',
  'VitalSignCodes',
  'MedicationStatus',
];

// ============================================================================
// Helper Functions
// ============================================================================
function randomItem(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

function randomConcept() {
  const categories = Object.keys(AU_CLINICAL_CONCEPTS);
  const category = randomItem(categories);
  return randomItem(AU_CLINICAL_CONCEPTS[category]);
}

function formatDuration(ms) {
  return `${ms.toFixed(2)}ms`;
}

// ============================================================================
// Test Scenarios
// ============================================================================
export default function () {
  const requestType = Math.random();

  // Health check (5% of requests)
  if (requestType < 0.05) {
    group('Health Check', function () {
      const start = Date.now();
      const res = http.get(`${BASE_URL}/health`);
      healthCheckLatency.add(Date.now() - start);

      const success = check(res, {
        'health check status 200': (r) => r.status === 200,
        'health check has status': (r) => {
          try {
            const body = JSON.parse(r.body);
            return body.status !== undefined;
          } catch {
            return false;
          }
        },
      });

      errorRate.add(!success);
    });
  }
  // Subsumption checks (35% of requests)
  else if (requestType < 0.40) {
    group('Subsumption Check', function () {
      const testCase = randomItem(SUBSUMPTION_TEST_CASES);
      const start = Date.now();

      const payload = JSON.stringify({
        subCode: testCase.child,
        superCode: testCase.parent,
        system: 'http://snomed.info/sct',
      });

      const res = http.post(`${BASE_URL}/v1/subsumption/check`, payload, {
        headers: { 'Content-Type': 'application/json' },
      });

      const latency = Date.now() - start;
      subsumptionLatency.add(latency);
      subsumptionRequests.add(1);

      const success = check(res, {
        'subsumption status 200': (r) => r.status === 200,
        'subsumption has result': (r) => {
          try {
            const body = JSON.parse(r.body);
            return typeof body.subsumes === 'boolean' || typeof body.result === 'boolean';
          } catch {
            return false;
          }
        },
      });

      errorRate.add(!success);
    });
  }
  // Value set listing (15% of requests)
  else if (requestType < 0.55) {
    group('Value Set List', function () {
      const start = Date.now();

      const res = http.get(`${BASE_URL}/v1/rules/valuesets?limit=20`);

      const latency = Date.now() - start;
      valuesetLatency.add(latency);
      valuesetRequests.add(1);

      const success = check(res, {
        'valueset list status 200': (r) => r.status === 200,
        'valueset list has data': (r) => {
          try {
            const body = JSON.parse(r.body);
            return Array.isArray(body.value_sets) || Array.isArray(body);
          } catch {
            return false;
          }
        },
      });

      errorRate.add(!success);
    });
  }
  // Value set expansion (20% of requests)
  else if (requestType < 0.75) {
    group('Value Set Expand', function () {
      const valueSetId = randomItem(VALUE_SETS);
      const start = Date.now();

      const res = http.get(`${BASE_URL}/v1/rules/valuesets/${valueSetId}/expand`);

      const latency = Date.now() - start;
      valuesetLatency.add(latency);
      valuesetRequests.add(1);

      const success = check(res, {
        'valueset expand status 2xx': (r) => r.status >= 200 && r.status < 300,
      });

      errorRate.add(!success);
    });
  }
  // Contains/membership check (25% of requests)
  else {
    group('Contains Check', function () {
      const concept = randomConcept();
      const valueSetId = randomItem(VALUE_SETS);
      const start = Date.now();

      const payload = JSON.stringify({
        code: concept.code,
        system: 'http://snomed.info/sct',
        valueSetId: valueSetId,
      });

      const res = http.post(`${BASE_URL}/v1/rules/valuesets/${valueSetId}/contains`, payload, {
        headers: { 'Content-Type': 'application/json' },
      });

      const latency = Date.now() - start;
      containsLatency.add(latency);
      containsRequests.add(1);

      // Contains endpoint may return 404 for non-existent value sets
      const success = check(res, {
        'contains check status valid': (r) => r.status === 200 || r.status === 404,
      });

      errorRate.add(!success && res.status >= 500);
    });
  }

  // Random sleep between 100ms and 500ms
  sleep(Math.random() * 0.4 + 0.1);
}

// ============================================================================
// Lifecycle Hooks
// ============================================================================
export function setup() {
  console.log('=================================================');
  console.log('  KB7-AU Load Test Suite');
  console.log('  Target: ' + BASE_URL);
  console.log('=================================================');

  // Verify API is accessible
  const healthRes = http.get(`${BASE_URL}/health`);
  if (healthRes.status !== 200) {
    console.error(`Health check failed: ${healthRes.status}`);
    console.error('Make sure KB7 service is running at ' + BASE_URL);
    throw new Error('API not accessible');
  }

  console.log('Health check passed, starting load test...');

  return {
    startTime: new Date().toISOString(),
  };
}

export function teardown(data) {
  console.log('=================================================');
  console.log('  Load Test Complete');
  console.log('  Started: ' + data.startTime);
  console.log('  Ended: ' + new Date().toISOString());
  console.log('=================================================');
}

// ============================================================================
// Custom Summary
// ============================================================================
export function handleSummary(data) {
  const summary = {
    testRun: {
      startTime: data.root_group.checks ? new Date().toISOString() : null,
      totalRequests: data.metrics.http_reqs ? data.metrics.http_reqs.values.count : 0,
      errorRate: data.metrics.kb7_errors ? data.metrics.kb7_errors.values.rate : 0,
    },
    latencies: {
      subsumption: {
        p50: data.metrics.kb7_subsumption_latency_ms ? data.metrics.kb7_subsumption_latency_ms.values['p(50)'] : null,
        p95: data.metrics.kb7_subsumption_latency_ms ? data.metrics.kb7_subsumption_latency_ms.values['p(95)'] : null,
        p99: data.metrics.kb7_subsumption_latency_ms ? data.metrics.kb7_subsumption_latency_ms.values['p(99)'] : null,
      },
      valueset: {
        p50: data.metrics.kb7_valueset_latency_ms ? data.metrics.kb7_valueset_latency_ms.values['p(50)'] : null,
        p95: data.metrics.kb7_valueset_latency_ms ? data.metrics.kb7_valueset_latency_ms.values['p(95)'] : null,
        p99: data.metrics.kb7_valueset_latency_ms ? data.metrics.kb7_valueset_latency_ms.values['p(99)'] : null,
      },
      contains: {
        p50: data.metrics.kb7_contains_latency_ms ? data.metrics.kb7_contains_latency_ms.values['p(50)'] : null,
        p95: data.metrics.kb7_contains_latency_ms ? data.metrics.kb7_contains_latency_ms.values['p(95)'] : null,
        p99: data.metrics.kb7_contains_latency_ms ? data.metrics.kb7_contains_latency_ms.values['p(99)'] : null,
      },
    },
    thresholds: data.thresholds || {},
  };

  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'tests/performance/results/load_test_summary.json': JSON.stringify(summary, null, 2),
  };
}

// Simple text summary helper
function textSummary(data, options) {
  let output = '\n=== KB7-AU Load Test Results ===\n\n';

  if (data.metrics.http_reqs) {
    output += `Total Requests: ${data.metrics.http_reqs.values.count}\n`;
    output += `Request Rate: ${data.metrics.http_reqs.values.rate.toFixed(2)}/s\n`;
  }

  if (data.metrics.http_req_duration) {
    output += `\nResponse Times:\n`;
    output += `  p50: ${data.metrics.http_req_duration.values['p(50)'].toFixed(2)}ms\n`;
    output += `  p95: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms\n`;
    output += `  p99: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms\n`;
  }

  if (data.metrics.kb7_errors) {
    output += `\nError Rate: ${(data.metrics.kb7_errors.values.rate * 100).toFixed(2)}%\n`;
  }

  output += '\n=================================\n';

  return output;
}
