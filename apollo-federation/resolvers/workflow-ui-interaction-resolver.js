/**
 * Advanced UI Interaction Resolvers for Clinical Override Management
 * Integrates with Workflow Engine Go service for real-time UI updates
 */

const { PubSub, withFilter } = require('graphql-subscriptions');
const { GraphQLError } = require('graphql');
const Redis = require('ioredis');
const { v4: uuidv4 } = require('uuid');

// Initialize PubSub for real-time subscriptions
const pubsub = new PubSub();
const redis = new Redis({
  host: process.env.REDIS_HOST || 'localhost',
  port: process.env.REDIS_PORT || 6379,
});

// WebSocket connection manager for real-time UI updates
const activeConnections = new Map();

/**
 * Main resolver implementation for UI interactions
 */
const workflowUIInteractionResolvers = {
  Mutation: {
    /**
     * Updates UI notification state and broadcasts to connected clients
     */
    updateUINotification: async (_, { workflowId, notification }, context) => {
      // Verify user authorization
      if (!context.user) {
        throw new GraphQLError('Authentication required', {
          extensions: { code: 'UNAUTHENTICATED' }
        });
      }

      // Create notification object
      const uiNotification = {
        id: uuidv4(),
        workflowId,
        status: notification.status,
        title: notification.title,
        message: notification.message,
        severity: notification.severity,
        payload: notification.payload,
        actions: notification.actions?.map(action => ({
          ...action,
          isEnabled: true
        })),
        createdAt: new Date().toISOString(),
        expiresAt: notification.expiresAt,
        acknowledgedBy: []
      };

      // Store in Redis for persistence
      await redis.setex(
        `notification:${uiNotification.id}`,
        3600, // 1 hour TTL
        JSON.stringify(uiNotification)
      );

      // Broadcast to subscribed clients
      await pubsub.publish('WORKFLOW_UI_UPDATE', {
        workflowUIUpdates: {
          workflowId,
          updateType: 'NOTIFICATION',
          phase: await getCurrentPhase(workflowId),
          notification: uiNotification,
          timestamp: new Date().toISOString()
        }
      });

      return uiNotification;
    },

    /**
     * Initiates clinical override workflow with proper governance
     */
    requestClinicalOverride: async (_, { workflowId, validationId, request }, context) => {
      const clinician = await validateClinician(context.user);

      // Determine required override level based on findings
      const requiredLevel = determineOverrideLevel(request.findings);

      // Check if clinician has authority for this level
      if (!hasOverrideAuthority(clinician, requiredLevel)) {
        throw new GraphQLError('Insufficient authority for override level', {
          extensions: {
            code: 'FORBIDDEN',
            requiredLevel,
            clinicianLevel: clinician.authorityLevel
          }
        });
      }

      // Create override session
      const session = {
        id: uuidv4(),
        workflowId,
        validationId,
        status: 'PENDING',
        verdict: request.verdict,
        findings: request.findings,
        requiredLevel,
        requestedBy: clinician,
        requestedAt: new Date().toISOString(),
        expiresAt: calculateExpiration(request.urgency),
        auditTrail: [{
          id: uuidv4(),
          action: 'OVERRIDE_REQUESTED',
          actor: clinician,
          timestamp: new Date().toISOString(),
          details: { request }
        }]
      };

      // Store session
      await redis.setex(
        `override:${session.id}`,
        7200, // 2 hour TTL
        JSON.stringify(session)
      );

      // Notify relevant parties
      await notifyOverrideRequest(session);

      // Publish to subscription
      await pubsub.publish('OVERRIDE_REQUIRED', {
        overrideRequired: {
          workflowId,
          validationId,
          verdict: request.verdict,
          criticalFindings: request.findings.filter(f => f.severity === 'CRITICAL'),
          overrideOptions: getOverrideOptions(requiredLevel),
          timeoutAt: session.expiresAt,
          escalationPath: getEscalationPath(requiredLevel)
        }
      });

      return session;
    },

    /**
     * Resolves override decision and triggers commit phase
     */
    resolveClinicalOverride: async (_, { sessionId, decision }, context) => {
      const clinician = await validateClinician(context.user);

      // Retrieve session
      const sessionData = await redis.get(`override:${sessionId}`);
      if (!sessionData) {
        throw new GraphQLError('Override session not found or expired', {
          extensions: { code: 'NOT_FOUND' }
        });
      }

      const session = JSON.parse(sessionData);

      // Validate decision authority
      if (decision.overrideLevel === 'PEER_REVIEW' && !decision.coSignature) {
        throw new GraphQLError('Co-signature required for peer review override', {
          extensions: { code: 'VALIDATION_ERROR' }
        });
      }

      // Process decision
      let commitResult;

      switch (decision.decision) {
        case 'OVERRIDE':
          commitResult = await processOverrideCommit(session, decision, clinician);
          break;

        case 'MODIFY':
          commitResult = await processModifiedProposal(session, decision, clinician);
          break;

        case 'CANCEL':
          commitResult = await cancelWorkflow(session.workflowId, decision.reason);
          break;

        case 'DEFER':
          commitResult = await deferDecision(session, decision);
          break;

        case 'ESCALATE':
          return await escalateDecision(session, decision, clinician);

        default:
          throw new GraphQLError('Invalid override decision', {
            extensions: { code: 'VALIDATION_ERROR' }
          });
      }

      // Update session
      session.status = 'COMPLETED';
      session.decision = {
        ...decision,
        decidedBy: clinician,
        decidedAt: new Date().toISOString()
      };

      await redis.setex(
        `override:${sessionId}`,
        86400, // 24 hour TTL for completed sessions
        JSON.stringify(session)
      );

      // Audit trail
      await createAuditEntry({
        action: 'OVERRIDE_APPROVED',
        actor: clinician,
        workflowId: session.workflowId,
        details: { decision, commitResult }
      });

      // Send to learning loop
      await publishToLearningLoop({
        sessionId,
        workflowId: session.workflowId,
        verdict: session.verdict,
        findings: session.findings,
        decision,
        clinician,
        timestamp: new Date().toISOString()
      });

      return commitResult;
    },

    /**
     * Initiates peer review for complex overrides
     */
    requestPeerReview: async (_, { overrideSessionId, peerIds, urgency }, context) => {
      const requestor = await validateClinician(context.user);

      // Create peer review session
      const reviewSession = {
        id: uuidv4(),
        overrideSessionId,
        status: 'REQUESTED',
        requestedBy: requestor,
        reviewers: await Promise.all(peerIds.map(async id => ({
          clinician: await getClinician(id),
          status: 'INVITED',
          joinedAt: null,
          decision: null
        }))),
        urgency,
        consensus: null,
        startedAt: new Date().toISOString(),
        completedAt: null,
        chatMessages: []
      };

      // Store session
      await redis.setex(
        `peer-review:${reviewSession.id}`,
        getReviewTimeout(urgency),
        JSON.stringify(reviewSession)
      );

      // Notify reviewers
      await notifyReviewers(reviewSession);

      return reviewSession;
    },

    /**
     * Acknowledges warning with tracking
     */
    acknowledgeWarning: async (_, { workflowId, warningId, acknowledgement }, context) => {
      const clinician = await validateClinician(context.user);

      const result = {
        success: true,
        warningId,
        acknowledgedBy: clinician,
        timestamp: acknowledgement.timestamp,
        comments: acknowledgement.comments
      };

      // Store acknowledgement
      await redis.sadd(
        `workflow:${workflowId}:acknowledgements`,
        JSON.stringify(result)
      );

      // Update UI state
      await updateWorkflowUIState(workflowId, {
        acknowledgedWarnings: [warningId]
      });

      return result;
    }
  },

  Subscription: {
    /**
     * Real-time workflow UI updates
     */
    workflowUIUpdates: {
      subscribe: withFilter(
        () => pubsub.asyncIterator(['WORKFLOW_UI_UPDATE']),
        (payload, variables) => {
          return payload.workflowUIUpdates.workflowId === variables.workflowId;
        }
      )
    },

    /**
     * Override requirement notifications
     */
    overrideRequired: {
      subscribe: withFilter(
        () => pubsub.asyncIterator(['OVERRIDE_REQUIRED']),
        (payload, variables) => {
          return payload.overrideRequired.workflowId === variables.workflowId;
        }
      )
    },

    /**
     * Peer review updates
     */
    peerReviewUpdates: {
      subscribe: withFilter(
        () => pubsub.asyncIterator(['PEER_REVIEW_UPDATE']),
        (payload, variables) => {
          return payload.peerReviewUpdates.sessionId === variables.sessionId;
        }
      )
    },

    /**
     * System-wide clinical alerts
     */
    clinicalAlerts: {
      subscribe: withFilter(
        () => pubsub.asyncIterator(['CLINICAL_ALERT']),
        (payload, variables) => {
          const alert = payload.clinicalAlert;

          // Filter by severity if specified
          if (variables.severity && !variables.severity.includes(alert.severity)) {
            return false;
          }

          // Filter by department if specified
          if (variables.departments && !variables.departments.includes(alert.department)) {
            return false;
          }

          return true;
        }
      )
    }
  },

  Query: {
    /**
     * Get current workflow UI state
     */
    workflowUIState: async (_, { workflowId }, context) => {
      const stateData = await redis.get(`workflow:${workflowId}:ui-state`);

      if (!stateData) {
        // Initialize default state
        return {
          workflowId,
          currentPhase: 'CALCULATE',
          uiMode: 'STANDARD',
          activeModals: [],
          pendingActions: [],
          notifications: [],
          lastUpdated: new Date().toISOString(),
          userContext: {}
        };
      }

      return JSON.parse(stateData);
    },

    /**
     * Get pending overrides for clinician/department
     */
    pendingOverrides: async (_, { clinicianId, department, urgency }, context) => {
      const pattern = 'override:*';
      const keys = await redis.keys(pattern);

      const overrides = await Promise.all(
        keys.map(async key => {
          const data = await redis.get(key);
          return JSON.parse(data);
        })
      );

      return overrides.filter(override => {
        if (override.status !== 'PENDING') return false;
        if (clinicianId && override.requestedBy.id !== clinicianId) return false;
        if (department && override.requestedBy.department !== department) return false;
        if (urgency && override.urgency !== urgency) return false;
        return true;
      });
    },

    /**
     * Get override history with analytics
     */
    overrideHistory: async (_, { clinicianId, patientId, dateRange }, context) => {
      // This would connect to a persistent database for historical data
      // For now, returning a mock structure
      return {
        edges: [],
        pageInfo: {
          hasNextPage: false,
          hasPreviousPage: false,
          startCursor: null,
          endCursor: null
        },
        totalCount: 0,
        statistics: {
          totalOverrides: 0,
          byLevel: [],
          byReason: [],
          averageReviewTime: 0
        }
      };
    },

    /**
     * Analyze override patterns for learning loop
     */
    overridePatterns: async (_, { timeRange, groupBy }, context) => {
      // This would connect to analytics service
      // Returning mock data structure
      return {
        patterns: [],
        recommendations: [],
        falsePositiveRate: 0,
        clinicianAdherence: 0
      };
    }
  }
};

// Helper Functions

/**
 * Determines required override level based on validation findings
 */
function determineOverrideLevel(findings) {
  const criticalCount = findings.filter(f => f.severity === 'CRITICAL').length;
  const highCount = findings.filter(f => f.severity === 'HIGH').length;

  if (criticalCount > 0) {
    return 'SUPERVISORY';
  } else if (highCount > 2) {
    return 'PEER_REVIEW';
  }
  return 'CLINICAL_JUDGMENT';
}

/**
 * Checks if clinician has authority for override level
 */
function hasOverrideAuthority(clinician, requiredLevel) {
  const authorityMap = {
    'CLINICAL_JUDGMENT': ['RESIDENT', 'ATTENDING', 'SPECIALIST', 'DEPARTMENT_HEAD', 'CHIEF_MEDICAL_OFFICER'],
    'PEER_REVIEW': ['ATTENDING', 'SPECIALIST', 'DEPARTMENT_HEAD', 'CHIEF_MEDICAL_OFFICER'],
    'SUPERVISORY': ['DEPARTMENT_HEAD', 'CHIEF_MEDICAL_OFFICER'],
    'EMERGENCY': ['RESIDENT', 'ATTENDING', 'SPECIALIST', 'DEPARTMENT_HEAD', 'CHIEF_MEDICAL_OFFICER']
  };

  return authorityMap[requiredLevel]?.includes(clinician.authorityLevel) || false;
}

/**
 * Calculates expiration time based on urgency
 */
function calculateExpiration(urgency) {
  const expirationMinutes = {
    'ROUTINE': 240,     // 4 hours
    'URGENT': 60,       // 1 hour
    'STAT': 15,         // 15 minutes
    'EMERGENCY': 5      // 5 minutes
  };

  const minutes = expirationMinutes[urgency] || 240;
  const expiration = new Date();
  expiration.setMinutes(expiration.getMinutes() + minutes);
  return expiration.toISOString();
}

/**
 * Gets available override options for level
 */
function getOverrideOptions(level) {
  return [
    {
      level: 'CLINICAL_JUDGMENT',
      available: true,
      requirements: ['Clinical justification required'],
      authorizedRoles: ['ATTENDING', 'SPECIALIST']
    },
    {
      level: 'PEER_REVIEW',
      available: level !== 'SUPERVISORY',
      requirements: ['Co-signature required', 'Peer review documentation'],
      authorizedRoles: ['ATTENDING', 'SPECIALIST']
    },
    {
      level: 'SUPERVISORY',
      available: level === 'SUPERVISORY',
      requirements: ['Department head approval', 'Risk assessment documentation'],
      authorizedRoles: ['DEPARTMENT_HEAD', 'CHIEF_MEDICAL_OFFICER']
    }
  ];
}

/**
 * Gets escalation path for override level
 */
function getEscalationPath(level) {
  const basePath = [
    { level: 1, role: 'ATTENDING', contactMethod: 'PAGER', timeoutMinutes: 15 },
    { level: 2, role: 'SPECIALIST', contactMethod: 'PHONE', timeoutMinutes: 30 },
    { level: 3, role: 'DEPARTMENT_HEAD', contactMethod: 'PHONE', timeoutMinutes: 60 }
  ];

  if (level === 'SUPERVISORY') {
    return basePath.slice(2); // Start at department head
  } else if (level === 'PEER_REVIEW') {
    return basePath.slice(1); // Start at specialist
  }
  return basePath;
}

/**
 * Process override commit with Workflow Engine
 */
async function processOverrideCommit(session, decision, clinician) {
  // Call Workflow Engine Go service to commit with override
  const response = await fetch(`${process.env.WORKFLOW_ENGINE_URL}/commit`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Clinician-ID': clinician.id
    },
    body: JSON.stringify({
      workflowId: session.workflowId,
      validationId: session.validationId,
      overrideDecision: {
        level: decision.overrideLevel,
        reason: decision.reason,
        justification: decision.clinicalJustification,
        coSignature: decision.coSignature
      }
    })
  });

  if (!response.ok) {
    throw new GraphQLError('Failed to commit with override', {
      extensions: { code: 'INTERNAL_ERROR' }
    });
  }

  return await response.json();
}

/**
 * Publishes override event to Kafka for learning loop
 */
async function publishToLearningLoop(event) {
  // This would publish to Kafka topic 'clinical-overrides'
  // For now, just log it
  console.log('Publishing to learning loop:', event);

  // Store in Redis for pattern analysis
  await redis.zadd(
    'override-events',
    Date.now(),
    JSON.stringify(event)
  );
}

/**
 * Creates audit entry for compliance
 */
async function createAuditEntry(entry) {
  const auditEntry = {
    id: uuidv4(),
    ...entry,
    timestamp: new Date().toISOString(),
    signature: generateSignature(entry) // Cryptographic signature
  };

  // Store in audit trail
  await redis.lpush(
    `audit:${entry.workflowId}`,
    JSON.stringify(auditEntry)
  );

  return auditEntry;
}

/**
 * Generates cryptographic signature for audit entry
 */
function generateSignature(data) {
  const crypto = require('crypto');
  const hash = crypto.createHash('sha256');
  hash.update(JSON.stringify(data));
  return hash.digest('hex');
}

/**
 * Validates and retrieves clinician information
 */
async function validateClinician(user) {
  if (!user) {
    throw new GraphQLError('Authentication required', {
      extensions: { code: 'UNAUTHENTICATED' }
    });
  }

  // Fetch clinician details from auth service or database
  return {
    id: user.id,
    name: user.name,
    role: user.role || 'ATTENDING',
    department: user.department || 'GENERAL',
    authorityLevel: user.authorityLevel || 'ATTENDING',
    available: true
  };
}

/**
 * Gets current workflow phase from Workflow Engine
 */
async function getCurrentPhase(workflowId) {
  // Query Workflow Engine for current phase
  // For now, return default
  return 'VALIDATE';
}

/**
 * Updates workflow UI state
 */
async function updateWorkflowUIState(workflowId, updates) {
  const currentState = await redis.get(`workflow:${workflowId}:ui-state`);
  const state = currentState ? JSON.parse(currentState) : {};

  const updatedState = {
    ...state,
    ...updates,
    lastUpdated: new Date().toISOString()
  };

  await redis.setex(
    `workflow:${workflowId}:ui-state`,
    3600,
    JSON.stringify(updatedState)
  );

  return updatedState;
}

module.exports = workflowUIInteractionResolvers;