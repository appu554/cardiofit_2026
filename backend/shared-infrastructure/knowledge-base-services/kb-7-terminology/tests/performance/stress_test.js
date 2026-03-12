// KB7-AU Stress Test Suite
// Stress testing to find breaking points
//
// Run: k6 run stress_test.js
// With env: k6 run --env API_URL=http://localhost:8087 stress_test.js

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// ============================================================================
// Custom Metrics
// ============================================================================
const errorRate = new Rate('kb7_stress_errors');
const responseTime = new Trend('kb7_stress_response_ms');
const requestsPerSecond = new Counter('kb7_stress_requests');

// ============================================================================
// Stress Test Configuration
// ============================================================================
export const options = {
  stages: [
    // Ramp up aggressively
    { duration: '1m', target: 50 },    // Warm up to 50 VUs
    { duration: '2m', target: 100 },   // Normal load
    { duration: '2m', target: 150 },   // Above normal
    { duration: '2m', target: 200 },   // High load
    { duration: '2m', target: 250 },   // Very high load
    { duration: '2m', target: 300 },   // Stress point
    { duration: '2m', target: 400 },   // Breaking point test
    { duration: '2m', target: 500 },   // Maximum stress
    { duration: '5m', target: 500 },   // Sustain maximum stress
    { duration: '3m', target: 100 },   // Recovery
    { duration: '2m', target: 0 },     // Ramp down
  ],
  thresholds: {
    // Relaxed thresholds for stress test
    'http_req_duration': ['p(95)<2000', 'p(99)<5000'],  // Allow up to 2s/5s
    'http_req_failed': ['rate<0.10'],                   // Allow up to 10% errors
    'kb7_stress_errors': ['rate<0.15'],                 // Custom error threshold
  },
};

// ============================================================================
// Test Data
// ============================================================================
const BASE_URL = __ENV.API_URL || 'http://localhost:8087';

// Minimal test data for maximum throughput
const SNOMED_CODES = [
  '91302008', '10001005', '14669001', '73211009',
  '84114007', '387517004', '372735009', '38341003',
];

const SUBSUMPTION_PAIRS = [
  { child: '91302008', parent: '404684003' },
  { child: '10001005', parent: '91302008' },
  { child: '14669001', parent: '90708001' },
  { child: '44054006', parent: '73211009' },
  { child: '387517004', parent: '373265006' },
];

const VALUE_SETS = [
  'SepsisConditions',
  'RenalConditions',
  'CardiacConditions',
  'DiabetesConditions',
];

// ============================================================================
// Helper Functions
// ============================================================================
function randomItem(arr) {
  return arr[Math.floor(Math.random() * arr.length)];
}

// ============================================================================
// Stress Test Scenario
// ============================================================================
export default function () {
  requestsPerSecond.add(1);

  // Random distribution of request types
  const rand = Math.random();

  if (rand < 0.10) {
    // 10% - Health checks (lightweight)
    stressHealthCheck();
  } else if (rand < 0.45) {
    // 35% - Subsumption checks (moderate load)
    stressSubsumptionCheck();
  } else if (rand < 0.70) {
    // 25% - Value set operations (heavier load)
    stressValueSetOp();
  } else {
    // 30% - Contains checks (rule engine bridge)
    stressContainsCheck();
  }

  // Minimal sleep for maximum stress
  sleep(Math.random() * 0.1); // 0-100ms
}

function stressHealthCheck() {
  const start = Date.now();
  const res = http.get(`${BASE_URL}/health`);
  responseTime.add(Date.now() - start);

  const success = check(res, {
    'health 200': (r) => r.status === 200,
  });
  errorRate.add(!success);
}

function stressSubsumptionCheck() {
  const pair = randomItem(SUBSUMPTION_PAIRS);
  const start = Date.now();

  const payload = JSON.stringify({
    subCode: pair.child,
    superCode: pair.parent,
    system: 'http://snomed.info/sct',
  });

  const res = http.post(`${BASE_URL}/v1/subsumption/check`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  responseTime.add(Date.now() - start);

  const success = check(res, {
    'subsumption 200': (r) => r.status === 200,
  });
  errorRate.add(!success);
}

function stressValueSetOp() {
  const valueSetId = randomItem(VALUE_SETS);
  const start = Date.now();

  // Randomly choose list or expand
  const res = Math.random() < 0.5
    ? http.get(`${BASE_URL}/v1/rules/valuesets?limit=10`)
    : http.get(`${BASE_URL}/v1/rules/valuesets/${valueSetId}/expand`);

  responseTime.add(Date.now() - start);

  const success = check(res, {
    'valueset 2xx': (r) => r.status >= 200 && r.status < 300,
  });
  errorRate.add(!success && res.status >= 500);
}

function stressContainsCheck() {
  const code = randomItem(SNOMED_CODES);
  const valueSetId = randomItem(VALUE_SETS);
  const start = Date.now();

  const payload = JSON.stringify({
    code: code,
    system: 'http://snomed.info/sct',
    valueSetId: valueSetId,
  });

  const res = http.post(`${BASE_URL}/v1/rules/valuesets/${valueSetId}/contains`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  responseTime.add(Date.now() - start);

  // 404 is OK for non-existent value sets
  const success = check(res, {
    'contains valid': (r) => r.status === 200 || r.status === 404,
  });
  errorRate.add(!success && res.status >= 500);
}

// ============================================================================
// Lifecycle Hooks
// ============================================================================
export function setup() {
  console.log('=================================================');
  console.log('  KB7-AU STRESS TEST');
  console.log('  Target: ' + BASE_URL);
  console.log('  WARNING: This test will push the system to limits');
  console.log('=================================================');

  // Verify API is accessible
  const healthRes = http.get(`${BASE_URL}/health`);
  if (healthRes.status !== 200) {
    throw new Error('API not accessible at ' + BASE_URL);
  }

  console.log('Health check passed, starting stress test...');

  return { startTime: new Date().toISOString() };
}

export function teardown(data) {
  console.log('=================================================');
  console.log('  Stress Test Complete');
  console.log('  Duration: ' + data.startTime + ' -> ' + new Date().toISOString());
  console.log('=================================================');
}

// ============================================================================
// Summary Handler
// ============================================================================
export function handleSummary(data) {
  let summary = '\n=== KB7-AU STRESS TEST RESULTS ===\n\n';

  if (data.metrics.http_reqs) {
    summary += `Total Requests: ${data.metrics.http_reqs.values.count}\n`;
    summary += `Peak RPS: ${data.metrics.http_reqs.values.rate.toFixed(2)}/s\n`;
  }

  if (data.metrics.http_req_duration) {
    summary += `\nResponse Times:\n`;
    summary += `  p50: ${data.metrics.http_req_duration.values['p(50)'].toFixed(2)}ms\n`;
    summary += `  p95: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms\n`;
    summary += `  p99: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms\n`;
    summary += `  max: ${data.metrics.http_req_duration.values['max'].toFixed(2)}ms\n`;
  }

  if (data.metrics.kb7_stress_errors) {
    const errorPct = (data.metrics.kb7_stress_errors.values.rate * 100).toFixed(2);
    summary += `\nError Rate: ${errorPct}%\n`;
    summary += errorPct > 5 ? '⚠️ HIGH ERROR RATE DETECTED\n' : '✅ Error rate acceptable\n';
  }

  if (data.metrics.http_req_failed) {
    const failedPct = (data.metrics.http_req_failed.values.rate * 100).toFixed(2);
    summary += `Failed Requests: ${failedPct}%\n`;
  }

  summary += '\n=== STRESS TEST ANALYSIS ===\n';

  // Analyze breaking points
  if (data.metrics.http_req_duration) {
    const p99 = data.metrics.http_req_duration.values['p(99)'];
    if (p99 > 5000) {
      summary += '🔴 CRITICAL: p99 latency > 5s - system under severe stress\n';
    } else if (p99 > 2000) {
      summary += '🟡 WARNING: p99 latency > 2s - approaching limits\n';
    } else if (p99 > 1000) {
      summary += '🟢 GOOD: p99 latency < 2s - handling stress well\n';
    } else {
      summary += '✅ EXCELLENT: p99 latency < 1s - system performing great\n';
    }
  }

  summary += '\n===================================\n';

  return {
    'stdout': summary,
    'tests/performance/results/stress_test_summary.json': JSON.stringify({
      timestamp: new Date().toISOString(),
      metrics: {
        totalRequests: data.metrics.http_reqs ? data.metrics.http_reqs.values.count : 0,
        peakRPS: data.metrics.http_reqs ? data.metrics.http_reqs.values.rate : 0,
        errorRate: data.metrics.kb7_stress_errors ? data.metrics.kb7_stress_errors.values.rate : 0,
        p50: data.metrics.http_req_duration ? data.metrics.http_req_duration.values['p(50)'] : null,
        p95: data.metrics.http_req_duration ? data.metrics.http_req_duration.values['p(95)'] : null,
        p99: data.metrics.http_req_duration ? data.metrics.http_req_duration.values['p(99)'] : null,
        max: data.metrics.http_req_duration ? data.metrics.http_req_duration.values['max'] : null,
      },
    }, null, 2),
  };
}
