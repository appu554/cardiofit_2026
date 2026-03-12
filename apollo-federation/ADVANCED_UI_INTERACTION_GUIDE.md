# Advanced 3-Phase Pattern with UI Interaction via Apollo Federation

## Overview

This guide explains how to implement the advanced 3-phase workflow pattern (Calculate → Validate → Commit) with real-time UI interaction for clinical override management through Apollo Federation.

## Architecture Flow

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│                 │    │                  │    │                     │
│   Frontend UI   │◄──►│ Apollo Federation│◄──►│ Workflow Engine Go  │
│                 │    │                  │    │                     │
│ • React/Vue     │    │ • GraphQL Gateway│    │ • Strategic         │
│ • Real-time     │    │ • Subscriptions  │    │   Orchestrator      │
│ • Override UX   │    │ • UI Coordination│    │ • Commit Logic      │
│                 │    │                  │    │                     │
└─────────────────┘    └──────────────────┘    └─────────────────────┘
                                │
                                │
                       ┌────────▼─────────┐
                       │                  │
                       │ Redis + PubSub   │
                       │                  │
                       │ • Session State  │
                       │ • Real-time Msg  │
                       │ • UI State Mgmt  │
                       │                  │
                       └──────────────────┘
```

## 🚀 Implementation Steps

### Step 1: Integrate New Schema

Add the UI interaction schema to your Apollo Federation gateway:

```bash
cd apollo-federation/schemas
# File already created: workflow-ui-interaction-schema.graphql
```

Update your federation configuration:

```javascript
// apollo-federation/index.js
const { buildSubgraphSchema } = require('@apollo/subgraph');
const { readFileSync } = require('fs');
const path = require('path');
const workflowUIResolvers = require('./resolvers/workflow-ui-interaction-resolver');

// Load schemas
const workflowOrchestrationSchema = readFileSync(
  path.join(__dirname, 'schemas/workflow-orchestration-schema.graphql'),
  'utf8'
);

const workflowUISchema = readFileSync(
  path.join(__dirname, 'schemas/workflow-ui-interaction-schema.graphql'),
  'utf8'
);

// Combine schemas
const typeDefs = `
  ${workflowOrchestrationSchema}
  ${workflowUISchema}
`;

// Create federated schema
const schema = buildSubgraphSchema({
  typeDefs,
  resolvers: workflowUIResolvers
});

module.exports = { schema };
```

### Step 2: Enhanced Workflow Engine Integration

Update your Workflow Engine Go service to support UI interactions:

```go
// internal/orchestration/ui_coordinator.go
package orchestration

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/go-redis/redis/v8"
    "go.uber.org/zap"
)

type UICoordinator struct {
    redis           *redis.Client
    logger          *zap.Logger
    apolloGateway   string
}

type UINotification struct {
    WorkflowID  string `json:"workflow_id"`
    Status      string `json:"status"`
    Title       string `json:"title"`
    Message     string `json:"message"`
    Severity    string `json:"severity"`
    Actions     []UIAction `json:"actions"`
}

type UIAction struct {
    ID      string `json:"id"`
    Label   string `json:"label"`
    Type    string `json:"type"`
    Payload map[string]interface{} `json:"payload"`
}

// RequestOverride sends override request to UI via Apollo Federation
func (u *UICoordinator) RequestOverride(ctx context.Context, request *OverrideRequest) error {
    notification := &UINotification{
        WorkflowID: request.WorkflowID,
        Status:     "ACTION_REQUIRED",
        Title:      "Clinical Override Required",
        Message:    formatOverrideMessage(request),
        Severity:   "WARNING",
        Actions: []UIAction{
            {
                ID:    "override",
                Label: "Override",
                Type:  "OVERRIDE",
                Payload: map[string]interface{}{
                    "verdict":           request.Verdict,
                    "findings":          request.Findings,
                    "override_allowed":  request.OverrideAllowed,
                    "required_level":    request.RequiredLevel,
                },
            },
            {
                ID:    "modify",
                Label: "Modify Proposal",
                Type:  "MODIFY",
                Payload: map[string]interface{}{
                    "proposal_id": request.ProposalID,
                },
            },
            {
                ID:    "cancel",
                Label: "Cancel",
                Type:  "CANCEL",
                Payload: nil,
            },
        },
    }

    // Send to Apollo Federation via GraphQL mutation
    return u.sendToApolloFederation(ctx, "updateUINotification", notification)
}

// ResolveOverride handles override decision from UI
func (u *UICoordinator) ResolveOverride(ctx context.Context, decision *OverrideDecision) (*CommitResult, error) {
    u.logger.Info("Processing override decision",
        zap.String("workflow_id", decision.WorkflowID),
        zap.String("decision", decision.Decision),
        zap.String("level", decision.OverrideLevel))

    switch decision.Decision {
    case "OVERRIDE":
        return u.processOverrideCommit(ctx, decision)
    case "MODIFY":
        return u.processModification(ctx, decision)
    case "CANCEL":
        return u.cancelWorkflow(ctx, decision.WorkflowID)
    default:
        return nil, fmt.Errorf("invalid override decision: %s", decision.Decision)
    }
}

func (u *UICoordinator) sendToApolloFederation(ctx context.Context, mutation string, data interface{}) error {
    // Implementation would send GraphQL mutation to Apollo Federation
    // For now, store in Redis for the resolver to pick up
    jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }

    return u.redis.Publish(ctx, "workflow-ui-updates", jsonData).Err()
}
```

### Step 3: Update Strategic Orchestrator

Modify the existing orchestrator to use UI coordination:

```go
// internal/orchestration/strategic_orchestrator.go

// Add to StrategicOrchestrator struct
type StrategicOrchestrator struct {
    // ... existing fields
    uiCoordinator       *UICoordinator
}

// Modify shouldCommitBasedOnValidation method
func (o *StrategicOrchestrator) shouldCommitBasedOnValidation(validationResult *clients.SafetyValidationResponse, commitMode string) bool {
    switch validationResult.Verdict {
    case "SAFE":
        return true

    case "WARNING":
        // Instead of automatic commit, request UI interaction
        if len(validationResult.OverrideTokens) > 0 {
            o.requestUIOverride(validationResult)
            return false // Wait for UI decision
        }
        return commitMode != "safe_only"

    case "UNSAFE":
        // Always require UI interaction for unsafe verdicts
        o.requestUIOverride(validationResult)
        return false

    default:
        return false
    }
}

func (o *StrategicOrchestrator) requestUIOverride(validationResult *clients.SafetyValidationResponse) {
    request := &OverrideRequest{
        WorkflowID:      o.currentWorkflowID,
        ValidationID:    validationResult.ValidationID,
        Verdict:         validationResult.Verdict,
        Findings:        validationResult.Findings,
        OverrideAllowed: len(validationResult.OverrideTokens) > 0,
        RequiredLevel:   determineRequiredLevel(validationResult.Findings),
    }

    ctx := context.Background()
    if err := o.uiCoordinator.RequestOverride(ctx, request); err != nil {
        o.logger.Error("Failed to request UI override", zap.Error(err))
    }
}
```

### Step 4: Frontend Integration

Example React component for handling override requests:

```typescript
// frontend/src/components/ClinicalOverrideModal.tsx
import React, { useState } from 'react';
import { useMutation, useSubscription } from '@apollo/client';
import {
  REQUEST_CLINICAL_OVERRIDE,
  RESOLVE_CLINICAL_OVERRIDE,
  OVERRIDE_REQUIRED_SUBSCRIPTION
} from './graphql/overrideQueries';

interface ClinicalOverrideModalProps {
  workflowId: string;
  onClose: () => void;
}

export const ClinicalOverrideModal: React.FC<ClinicalOverrideModalProps> = ({
  workflowId,
  onClose
}) => {
  const [overrideReason, setOverrideReason] = useState('');
  const [selectedLevel, setSelectedLevel] = useState('CLINICAL_JUDGMENT');

  // Subscribe to override requirements
  const { data: overrideData } = useSubscription(OVERRIDE_REQUIRED_SUBSCRIPTION, {
    variables: { workflowId }
  });

  const [resolveOverride] = useMutation(RESOLVE_CLINICAL_OVERRIDE);

  const handleOverride = async () => {
    try {
      await resolveOverride({
        variables: {
          sessionId: overrideData?.overrideRequired?.sessionId,
          decision: {
            decision: 'OVERRIDE',
            overrideLevel: selectedLevel,
            reason: {
              code: 'CLINICAL_BENEFIT',
              category: 'CLINICAL_BENEFIT',
              freeText: overrideReason
            },
            clinicalJustification: overrideReason,
            acknowledgements: ['I understand the risks']
          }
        }
      });
      onClose();
    } catch (error) {
      console.error('Override failed:', error);
    }
  };

  if (!overrideData?.overrideRequired) {
    return null;
  }

  const { verdict, criticalFindings, overrideOptions } = overrideData.overrideRequired;

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center">
      <div className="bg-white rounded-lg p-6 max-w-2xl w-full">
        <h2 className="text-xl font-bold mb-4 text-red-600">
          Clinical Override Required
        </h2>

        <div className="mb-4">
          <p className="font-semibold">Validation Verdict: {verdict}</p>
          <div className="mt-2">
            <h3 className="font-medium">Critical Findings:</h3>
            <ul className="list-disc pl-5">
              {criticalFindings.map((finding, index) => (
                <li key={index} className="text-red-600">
                  {finding.description}
                </li>
              ))}
            </ul>
          </div>
        </div>

        <div className="mb-4">
          <label className="block text-sm font-medium mb-2">
            Override Level:
          </label>
          <select
            value={selectedLevel}
            onChange={(e) => setSelectedLevel(e.target.value)}
            className="w-full border rounded px-3 py-2"
          >
            {overrideOptions
              .filter(option => option.available)
              .map(option => (
                <option key={option.level} value={option.level}>
                  {option.level} - {option.requirements.join(', ')}
                </option>
              ))}
          </select>
        </div>

        <div className="mb-4">
          <label className="block text-sm font-medium mb-2">
            Clinical Justification (Required):
          </label>
          <textarea
            value={overrideReason}
            onChange={(e) => setOverrideReason(e.target.value)}
            className="w-full border rounded px-3 py-2 h-24"
            placeholder="Provide clinical justification for override..."
            required
          />
        </div>

        <div className="flex justify-end space-x-2">
          <button
            onClick={onClose}
            className="px-4 py-2 border rounded hover:bg-gray-50"
          >
            Cancel
          </button>
          <button
            onClick={handleOverride}
            disabled={!overrideReason.trim()}
            className="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 disabled:opacity-50"
          >
            Override
          </button>
        </div>
      </div>
    </div>
  );
};
```

### Step 5: GraphQL Queries and Mutations

```typescript
// frontend/src/graphql/overrideQueries.ts
import { gql } from '@apollo/client';

export const OVERRIDE_REQUIRED_SUBSCRIPTION = gql`
  subscription OverrideRequired($workflowId: ID!) {
    overrideRequired(workflowId: $workflowId) {
      workflowId
      validationId
      verdict
      criticalFindings {
        findingId
        severity
        description
        clinicalSignificance
        recommendation
      }
      overrideOptions {
        level
        available
        requirements
        authorizedRoles
      }
      timeoutAt
      escalationPath {
        level
        role
        contactMethod
        timeoutMinutes
      }
    }
  }
`;

export const RESOLVE_CLINICAL_OVERRIDE = gql`
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
`;

export const WORKFLOW_UI_UPDATES_SUBSCRIPTION = gql`
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
        actions {
          id
          label
          type
          payload
        }
      }
      timestamp
    }
  }
`;
```

## 🔄 Complete Flow Example

### 1. Calculate Phase (Unchanged)
```
Frontend → Apollo Federation → Workflow Engine → Flow2 Go Engine
```

### 2. Validate Phase (Enhanced)
```
Workflow Engine → Safety Gateway → Returns WARNING/UNSAFE
                ↓
Workflow Engine → UI Coordinator → Apollo Federation
                ↓
Frontend receives subscription → Shows override modal
```

### 3. Commit Phase (Interactive)
```
Frontend → Override decision → Apollo Federation → Workflow Engine
                                                ↓
Workflow Engine → Processes override → Medication Service → Commit
                ↓
UI Coordinator → Apollo Federation → Frontend receives confirmation
```

## 🎯 Key Benefits

1. **Real-time Interaction**: WebSocket subscriptions for immediate UI updates
2. **Governance Compliance**: Structured override levels and authority validation
3. **Audit Trail**: Complete cryptographic audit trail for all decisions
4. **Learning Loop**: Override events published to Kafka for pattern analysis
5. **Scalability**: Apollo Federation handles multiple UI clients
6. **Type Safety**: GraphQL schema provides type safety across the stack

## 🧪 Testing

### Integration Test Example

```javascript
// apollo-federation/test-workflow-ui-integration.js
const { createTestClient } = require('apollo-server-testing');
const { server } = require('./server');

describe('Workflow UI Integration', () => {
  test('should handle override request flow', async () => {
    const { mutate, query } = createTestClient(server);

    // 1. Start workflow
    const workflowResult = await mutate({
      mutation: EXECUTE_MEDICATION_WORKFLOW,
      variables: {
        input: {
          patientId: "test-patient-123",
          medicationRequest: { /* ... */ },
          clinicalIntent: { /* ... */ },
          providerContext: { /* ... */ }
        }
      }
    });

    expect(workflowResult.data.executeMedicationWorkflow.status)
      .toBe('REQUIRES_PROVIDER_DECISION');

    // 2. Request override
    const overrideResult = await mutate({
      mutation: REQUEST_CLINICAL_OVERRIDE,
      variables: {
        workflowId: workflowResult.data.executeMedicationWorkflow.workflowInstanceId,
        validationId: workflowResult.data.executeMedicationWorkflow.validationId,
        request: {
          verdict: 'WARNING',
          findings: [/* ... */],
          urgency: 'ROUTINE'
        }
      }
    });

    expect(overrideResult.data.requestClinicalOverride.status)
      .toBe('PENDING');

    // 3. Resolve override
    const resolveResult = await mutate({
      mutation: RESOLVE_CLINICAL_OVERRIDE,
      variables: {
        sessionId: overrideResult.data.requestClinicalOverride.id,
        decision: {
          decision: 'OVERRIDE',
          overrideLevel: 'CLINICAL_JUDGMENT',
          reason: {
            code: 'CLINICAL_BENEFIT',
            category: 'CLINICAL_BENEFIT',
            freeText: 'Patient specific condition requires override'
          },
          clinicalJustification: 'Clinical benefit outweighs risk',
          acknowledgements: ['I understand the risks']
        }
      }
    });

    expect(resolveResult.data.resolveClinicalOverride.result)
      .toBe('SUCCESS');
  });
});
```

## 📚 Next Steps

1. **Deploy Schema Updates**: Update Apollo Federation with new schema
2. **Implement Frontend Components**: Create React/Vue components for override UX
3. **Configure Redis**: Set up Redis for session state and pub/sub
4. **Test Integration**: Run end-to-end tests with real UI interaction
5. **Monitor Performance**: Set up metrics for UI interaction response times
6. **Train Clinical Staff**: Provide training on new override workflow

This advanced pattern transforms the basic 3-phase workflow into a sophisticated, interactive system that maintains clinical safety while providing an excellent user experience for healthcare providers.