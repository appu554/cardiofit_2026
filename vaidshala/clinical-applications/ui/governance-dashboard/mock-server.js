/**
 * Mock KB-0 Governance API Server
 *
 * Provides sample data for testing the Governance Dashboard UI
 * without requiring the full Go backend and PostgreSQL setup.
 *
 * Run with: node mock-server.js
 * Listens on: http://localhost:8080
 */

const http = require('http');

// =============================================================================
// MOCK DATA
// =============================================================================

const mockFacts = [
  {
    id: 'f1a2b3c4-d5e6-7890-abcd-ef1234567890',
    factType: 'DRUG_INTERACTION',
    drugRxcui: '197361',
    drugName: 'Warfarin',
    interactingDrugRxcui: '310965',
    interactingDrugName: 'Aspirin',
    severity: 'HIGH',
    evidenceLevel: 'A',
    sourceAuthority: 'ONC',
    sourceType: 'AUTHORITATIVE',
    confidence: 0.97,
    content: {
      description: 'Concurrent use of warfarin and aspirin significantly increases the risk of major bleeding events.',
      mechanism: 'Both agents affect hemostasis through different mechanisms. Warfarin inhibits vitamin K-dependent clotting factors while aspirin irreversibly inhibits platelet cyclooxygenase.',
      recommendation: 'Avoid concurrent use unless benefits clearly outweigh risks. If used together, monitor INR closely and watch for signs of bleeding.',
      alternatives: ['Clopidogrel with dose adjustment', 'Consider direct-acting oral anticoagulants'],
      references: [
        { source: 'ONC Constitutional DDI Rules', citation: 'Rule DDI-001' },
        { source: 'FDA Drug Safety Communication', pubmedId: '28765432' }
      ],
      clinicalNotes: 'Risk is highest in patients over 65 with history of GI bleeding.'
    },
    status: 'PENDING_REVIEW',
    assignedReviewer: 'PharmD John Smith',
    reviewPriority: 'CRITICAL',
    slaDueDate: new Date(Date.now() + 18 * 60 * 60 * 1000).toISOString(),
    createdAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(),
    updatedAt: new Date(Date.now() - 1 * 60 * 60 * 1000).toISOString(),
  },
  {
    id: 'f2a2b3c4-d5e6-7890-abcd-ef1234567891',
    factType: 'CONTRAINDICATION',
    drugRxcui: '6809',
    drugName: 'Metformin',
    severity: 'CRITICAL',
    evidenceLevel: 'A',
    sourceAuthority: 'FDA',
    sourceType: 'AUTHORITATIVE',
    confidence: 0.99,
    content: {
      description: 'Metformin is contraindicated in patients with severe renal impairment (eGFR < 30 mL/min/1.73m²).',
      mechanism: 'Reduced renal clearance of metformin increases plasma levels, raising the risk of lactic acidosis.',
      recommendation: 'Do not initiate metformin in patients with eGFR < 30. Discontinue if eGFR falls below 30.',
      alternatives: ['Sulfonylureas with renal dosing', 'DPP-4 inhibitors', 'Insulin therapy'],
      references: [
        { source: 'FDA Label', url: 'https://www.accessdata.fda.gov/drugsatfda_docs/label/metformin.pdf' }
      ]
    },
    status: 'PENDING_REVIEW',
    reviewPriority: 'HIGH',
    slaDueDate: new Date(Date.now() + 36 * 60 * 60 * 1000).toISOString(),
    createdAt: new Date(Date.now() - 1 * 24 * 60 * 60 * 1000).toISOString(),
    updatedAt: new Date(Date.now() - 30 * 60 * 1000).toISOString(),
  },
  {
    id: 'f3a2b3c4-d5e6-7890-abcd-ef1234567892',
    factType: 'DOSING_RULE',
    drugRxcui: '10689',
    drugName: 'Lisinopril',
    severity: 'MODERATE',
    evidenceLevel: 'B',
    sourceAuthority: 'OHDSI',
    sourceType: 'CURATED',
    confidence: 0.82,
    content: {
      description: 'Starting dose for lisinopril in hypertension should be 10mg once daily.',
      recommendation: 'Titrate dose based on blood pressure response. Maximum dose 40mg daily.',
      references: [
        { source: 'JNC 8 Guidelines' }
      ]
    },
    status: 'PENDING_REVIEW',
    reviewPriority: 'STANDARD',
    slaDueDate: new Date(Date.now() + 5 * 24 * 60 * 60 * 1000).toISOString(),
    createdAt: new Date(Date.now() - 3 * 24 * 60 * 60 * 1000).toISOString(),
    updatedAt: new Date().toISOString(),
  },
  {
    id: 'f4a2b3c4-d5e6-7890-abcd-ef1234567893',
    factType: 'SAFETY_SIGNAL',
    drugRxcui: '283742',
    drugName: 'Pembrolizumab',
    severity: 'HIGH',
    evidenceLevel: 'B',
    sourceAuthority: 'FDA',
    sourceType: 'AUTHORITATIVE',
    confidence: 0.91,
    content: {
      description: 'Immune-mediated pneumonitis reported in 3.4% of patients receiving pembrolizumab.',
      mechanism: 'PD-1 blockade may result in immune-mediated adverse reactions including pneumonitis.',
      recommendation: 'Monitor for signs of pneumonitis. Withhold for Grade 2, discontinue for Grade 3-4.',
    },
    status: 'PENDING_REVIEW',
    reviewPriority: 'CRITICAL',
    slaDueDate: new Date(Date.now() + 12 * 60 * 60 * 1000).toISOString(),
    conflictGroupId: 'cg-001',
    createdAt: new Date(Date.now() - 6 * 60 * 60 * 1000).toISOString(),
    updatedAt: new Date().toISOString(),
  },
  {
    id: 'f5a2b3c4-d5e6-7890-abcd-ef1234567894',
    factType: 'DRUG_INTERACTION',
    drugRxcui: '283742',
    drugName: 'Pembrolizumab',
    interactingDrugRxcui: '8076',
    interactingDrugName: 'Prednisone',
    severity: 'MODERATE',
    evidenceLevel: 'C',
    sourceAuthority: 'OHDSI',
    sourceType: 'LLM_EXTRACTED',
    confidence: 0.72,
    content: {
      description: 'High-dose corticosteroids may reduce efficacy of pembrolizumab.',
    },
    status: 'PENDING_REVIEW',
    reviewPriority: 'LOW',
    slaDueDate: new Date(Date.now() + 10 * 24 * 60 * 60 * 1000).toISOString(),
    conflictGroupId: 'cg-001',
    createdAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(),
    updatedAt: new Date().toISOString(),
  },
];

const mockAuditEvents = [
  {
    id: 'audit-001',
    factId: 'f1a2b3c4-d5e6-7890-abcd-ef1234567890',
    eventType: 'FACT_CREATED',
    actorId: 'system',
    actorName: 'ONC Extractor',
    actorRole: 'SYSTEM',
    newState: 'DRAFT',
    signature: 'sha256:a1b2c3d4e5f6...',
    createdAt: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000).toISOString(),
  },
  {
    id: 'audit-002',
    factId: 'f1a2b3c4-d5e6-7890-abcd-ef1234567890',
    eventType: 'FACT_SUBMITTED_FOR_REVIEW',
    actorId: 'system',
    actorName: 'Governance Executor',
    actorRole: 'SYSTEM',
    previousState: 'DRAFT',
    newState: 'PENDING_REVIEW',
    reason: 'High confidence fact auto-submitted for review',
    signature: 'sha256:b2c3d4e5f6g7...',
    createdAt: new Date(Date.now() - 1.5 * 24 * 60 * 60 * 1000).toISOString(),
  },
  {
    id: 'audit-003',
    factId: 'f1a2b3c4-d5e6-7890-abcd-ef1234567890',
    eventType: 'REVIEWER_ASSIGNED',
    actorId: 'system',
    actorName: 'Auto-Assignment',
    actorRole: 'SYSTEM',
    reason: 'Assigned to PharmD John Smith based on workload balancing',
    signature: 'sha256:c3d4e5f6g7h8...',
    createdAt: new Date(Date.now() - 1 * 24 * 60 * 60 * 1000).toISOString(),
  },
];

let executorStatus = {
  running: true,
  lastProcessedAt: new Date().toISOString(),
  factsProcessed: 47,
  errors: 2,
  pollIntervalMs: 30000,
};

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

function getSlaStatus(dueDate) {
  const hoursRemaining = (new Date(dueDate) - new Date()) / (1000 * 60 * 60);
  if (hoursRemaining < 0) return 'BREACHED';
  if (hoursRemaining < 8) return 'AT_RISK';
  return 'ON_TRACK';
}

function buildQueueItem(fact) {
  const hoursRemaining = (new Date(fact.slaDueDate) - new Date()) / (1000 * 60 * 60);
  return {
    fact,
    priority: fact.reviewPriority,
    slaDueDate: fact.slaDueDate,
    slaStatus: getSlaStatus(fact.slaDueDate),
    assignedReviewer: fact.assignedReviewer,
    hasConflicts: !!fact.conflictGroupId,
    conflictCount: fact.conflictGroupId ? mockFacts.filter(f => f.conflictGroupId === fact.conflictGroupId).length : 0,
    queuedAt: fact.updatedAt,
  };
}

function buildMetrics() {
  const pending = mockFacts.filter(f => f.status === 'PENDING_REVIEW');
  const overdue = pending.filter(f => getSlaStatus(f.slaDueDate) === 'BREACHED');

  return {
    totalFacts: 1247,
    factsByStatus: {
      DRAFT: 23,
      PENDING_REVIEW: pending.length,
      APPROVED: 892,
      ACTIVE: 856,
      REJECTED: 43,
      SUPERSEDED: 112,
      RETIRED: 45,
    },
    factsByPriority: {
      CRITICAL: pending.filter(f => f.reviewPriority === 'CRITICAL').length,
      HIGH: pending.filter(f => f.reviewPriority === 'HIGH').length,
      STANDARD: pending.filter(f => f.reviewPriority === 'STANDARD').length,
      LOW: pending.filter(f => f.reviewPriority === 'LOW').length,
    },
    pendingReviews: pending.length,
    overdueReviews: overdue.length,
    todayApproved: 12,
    todayRejected: 2,
    avgReviewTimeHours: 18.5,
    slaCompliancePercent: 94.2,
    conflictsPending: 2,
  };
}

function parseBody(req) {
  return new Promise((resolve, reject) => {
    let body = '';
    req.on('data', chunk => body += chunk);
    req.on('end', () => {
      try {
        resolve(body ? JSON.parse(body) : {});
      } catch (e) {
        reject(e);
      }
    });
  });
}

function sendJSON(res, status, data) {
  res.writeHead(status, {
    'Content-Type': 'application/json',
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type, Authorization, X-Session-ID, X-Reviewer-ID',
  });
  res.end(JSON.stringify(data));
}

// =============================================================================
// REQUEST HANDLER
// =============================================================================

async function handleRequest(req, res) {
  const url = new URL(req.url, `http://${req.headers.host}`);
  const path = url.pathname;
  const method = req.method;

  console.log(`${method} ${path}`);

  // CORS preflight
  if (method === 'OPTIONS') {
    res.writeHead(204, {
      'Access-Control-Allow-Origin': '*',
      'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
      'Access-Control-Allow-Headers': 'Content-Type, Authorization, X-Session-ID, X-Reviewer-ID',
    });
    return res.end();
  }

  // Health check
  if (path === '/health') {
    return sendJSON(res, 200, { status: 'healthy', service: 'mock-kb0-governance', timestamp: new Date().toISOString() });
  }

  // Queue endpoints
  if (path === '/api/v2/governance/queue' && method === 'GET') {
    const items = mockFacts
      .filter(f => f.status === 'PENDING_REVIEW')
      .map(buildQueueItem);
    return sendJSON(res, 200, { items, total: items.length, page: 1, pageSize: 20, hasMore: false });
  }

  if (path.match(/^\/api\/v2\/governance\/queue\/priority\/\w+$/) && method === 'GET') {
    const priority = path.split('/').pop();
    const items = mockFacts
      .filter(f => f.status === 'PENDING_REVIEW' && f.reviewPriority === priority)
      .map(buildQueueItem);
    return sendJSON(res, 200, { success: true, data: items });
  }

  // Fact endpoints
  const factMatch = path.match(/^\/api\/v2\/governance\/facts\/([^\/]+)$/);
  if (factMatch && method === 'GET') {
    const factId = factMatch[1];
    const fact = mockFacts.find(f => f.id === factId);
    if (fact) {
      return sendJSON(res, 200, { success: true, data: fact, timestamp: new Date().toISOString() });
    }
    return sendJSON(res, 404, { success: false, error: 'Fact not found' });
  }

  const conflictsMatch = path.match(/^\/api\/v2\/governance\/facts\/([^\/]+)\/conflicts$/);
  if (conflictsMatch && method === 'GET') {
    const factId = conflictsMatch[1];
    const fact = mockFacts.find(f => f.id === factId);
    if (fact && fact.conflictGroupId) {
      const conflictingFacts = mockFacts.filter(f => f.conflictGroupId === fact.conflictGroupId);
      return sendJSON(res, 200, {
        success: true,
        data: {
          groupId: fact.conflictGroupId,
          drugRxcui: fact.drugRxcui,
          drugName: fact.drugName,
          factType: fact.factType,
          facts: conflictingFacts,
          resolutionStrategy: 'AUTHORITY_PRIORITY',
          suggestedWinner: conflictingFacts.sort((a, b) => b.confidence - a.confidence)[0]?.id,
          resolutionReason: 'FDA source has higher authority priority than OHDSI',
        }
      });
    }
    return sendJSON(res, 200, { success: true, data: null });
  }

  const historyMatch = path.match(/^\/api\/v2\/governance\/facts\/([^\/]+)\/history$/);
  if (historyMatch && method === 'GET') {
    const factId = historyMatch[1];
    const events = mockAuditEvents.filter(e => e.factId === factId);
    return sendJSON(res, 200, { success: true, data: events });
  }

  // Review actions
  const approveMatch = path.match(/^\/api\/v2\/governance\/facts\/([^\/]+)\/approve$/);
  if (approveMatch && method === 'POST') {
    const factId = approveMatch[1];
    const body = await parseBody(req);
    console.log('Approving fact:', factId, body);
    return sendJSON(res, 200, {
      success: true,
      data: {
        factId,
        decision: 'APPROVED',
        reviewerId: body.reviewerId || 'current-user',
        reviewerName: 'PharmD Reviewer',
        reason: body.reason,
        decidedAt: new Date().toISOString(),
        signature: 'sha256:approve_' + Date.now(),
      }
    });
  }

  const rejectMatch = path.match(/^\/api\/v2\/governance\/facts\/([^\/]+)\/reject$/);
  if (rejectMatch && method === 'POST') {
    const factId = rejectMatch[1];
    const body = await parseBody(req);
    console.log('Rejecting fact:', factId, body);
    return sendJSON(res, 200, {
      success: true,
      data: {
        factId,
        decision: 'REJECTED',
        reviewerId: body.reviewerId || 'current-user',
        reviewerName: 'PharmD Reviewer',
        reason: body.reason,
        decidedAt: new Date().toISOString(),
        signature: 'sha256:reject_' + Date.now(),
      }
    });
  }

  const escalateMatch = path.match(/^\/api\/v2\/governance\/facts\/([^\/]+)\/escalate$/);
  if (escalateMatch && method === 'POST') {
    const factId = escalateMatch[1];
    const body = await parseBody(req);
    console.log('Escalating fact:', factId, body);
    return sendJSON(res, 200, {
      success: true,
      data: {
        factId,
        decision: 'ESCALATED',
        reviewerId: body.reviewerId || 'current-user',
        reviewerName: 'PharmD Reviewer',
        reason: body.reason,
        decidedAt: new Date().toISOString(),
        signature: 'sha256:escalate_' + Date.now(),
      }
    });
  }

  // Metrics & Dashboard
  if (path === '/api/v2/governance/metrics' && method === 'GET') {
    return sendJSON(res, 200, { success: true, data: buildMetrics(), timestamp: new Date().toISOString() });
  }

  if (path === '/api/v2/governance/dashboard' && method === 'GET') {
    const metrics = buildMetrics();
    return sendJSON(res, 200, {
      success: true,
      data: {
        metrics,
        recentActivity: mockAuditEvents.slice(0, 10),
        queueSummary: {
          critical: metrics.factsByPriority.CRITICAL,
          high: metrics.factsByPriority.HIGH,
          standard: metrics.factsByPriority.STANDARD,
          low: metrics.factsByPriority.LOW,
          overdue: metrics.overdueReviews,
        },
        reviewerWorkload: [
          { reviewerId: 'r1', reviewerName: 'PharmD John Smith', assigned: 3, completed: 28, avgTimeHours: 16.2 },
          { reviewerId: 'r2', reviewerName: 'PharmD Jane Doe', assigned: 2, completed: 35, avgTimeHours: 14.8 },
        ],
      },
      timestamp: new Date().toISOString(),
    });
  }

  // Executor control
  if (path === '/api/v2/governance/executor/status' && method === 'GET') {
    return sendJSON(res, 200, { success: true, data: executorStatus, timestamp: new Date().toISOString() });
  }

  if (path === '/api/v2/governance/executor/start' && method === 'POST') {
    executorStatus.running = true;
    console.log('Executor started');
    return sendJSON(res, 200, { success: true, message: 'Executor started' });
  }

  if (path === '/api/v2/governance/executor/stop' && method === 'POST') {
    executorStatus.running = false;
    console.log('Executor stopped');
    return sendJSON(res, 200, { success: true, message: 'Executor stopped' });
  }

  // 404
  sendJSON(res, 404, { error: 'Not found', path });
}

// =============================================================================
// START SERVER
// =============================================================================

const PORT = process.env.PORT || 8080;
const server = http.createServer(handleRequest);

server.listen(PORT, () => {
  console.log(`
╔═══════════════════════════════════════════════════════════════════╗
║           Mock KB-0 Governance API Server                          ║
╠═══════════════════════════════════════════════════════════════════╣
║  Status:    RUNNING                                                ║
║  Port:      ${PORT}                                                     ║
║  Health:    http://localhost:${PORT}/health                             ║
║  Queue:     http://localhost:${PORT}/api/v2/governance/queue            ║
║  Metrics:   http://localhost:${PORT}/api/v2/governance/metrics          ║
║  Dashboard: http://localhost:${PORT}/api/v2/governance/dashboard        ║
╚═══════════════════════════════════════════════════════════════════╝

Sample Facts Loaded:
  - Warfarin + Aspirin (DDI, CRITICAL)
  - Metformin Renal Contraindication (HIGH)
  - Lisinopril Dosing (STANDARD)
  - Pembrolizumab Safety Signal (CRITICAL)
  - Pembrolizumab + Prednisone (LOW, LLM-extracted)

Press Ctrl+C to stop.
`);
});
