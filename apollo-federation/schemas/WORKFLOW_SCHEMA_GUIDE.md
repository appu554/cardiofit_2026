# Workflow Orchestration GraphQL Schema Guide

## Overview

This guide provides detailed documentation for the `workflow-orchestration-schema.graphql` which exposes the Calculate > Validate > Commit workflow through Apollo Federation.

## Schema Structure

### Primary Mutation

#### `orchestrateMedicationRequest`
The main entry point for executing the complete medication workflow.

```graphql
mutation OrchestrateMedicationRequest($input: MedicationOrchestrationInput!) {
  orchestrateMedicationRequest(input: $input) {
    status
    correlationId
    
    # Success path fields
    medicationOrderId
    calculation {
      proposalSetId
      snapshotId
      executionTimeMs
      proposalCount
    }
    validation {
      validationId
      verdict
      riskScore
      findingsCount
      processingTimeMs
    }
    commitment {
      orderId
      auditTrailId
      persistenceStatus
      eventStatus
    }
    performance {
      totalTimeMs
      meetsTarget
      calculateTimeMs
      validateTimeMs
      commitTimeMs
    }
    
    # Warning/Override path fields
    validationFindings {
      findingId
      severity
      category
      description
      clinicalSignificance
      recommendation
      confidenceScore
      engineSource
    }
    overrideTokens
    proposals {
      proposalId
      proposalType
      status
      medicationData {
        code
        name
        dosage
        frequency
        route
        confidenceScore
      }
    }
    snapshotId
    
    # Error/Blocked path fields  
    blockingFindings {
      # Same structure as validationFindings
    }
    alternativeApproaches {
      medicationCode
      medicationName
      rationale
      dosage
      frequency
      route
      confidenceScore
    }
    
    # Meta fields
    errorCode
    errorMessage
    timestamp
  }
}
```

### Input Types

#### `MedicationOrchestrationInput`
Primary input for workflow orchestration.

```graphql
input MedicationOrchestrationInput {
  patientId: String!
  medicationRequest: MedicationRequestInput!
  clinicalIntent: ClinicalIntentInput!
  providerContext: ProviderContextInput!
  correlationId: String              # Optional - auto-generated if not provided
  urgency: WorkflowUrgency = ROUTINE # ROUTINE, URGENT, EMERGENT, STAT
  preferences: WorkflowPreferences   # Optional workflow preferences
}
```

#### `MedicationRequestInput`
Medication details for the request.

```graphql
input MedicationRequestInput {
  medicationCode: String!    # NDC or other standard medication code
  medicationName: String!    # Human-readable medication name
  dosage: String!           # e.g., "10mg", "5ml"
  frequency: String!        # e.g., "once daily", "twice daily", "PRN"
  route: String = "oral"    # Administration route
  duration: String          # Treatment duration, optional
  indication: String        # Clinical indication, optional
  priority: String = "routine" # Request priority
}
```

#### `ClinicalIntentInput`
Clinical context and treatment goals.

```graphql
input ClinicalIntentInput {
  primaryIndication: String!      # Primary reason for medication
  targetOutcome: String          # Desired clinical outcome
  treatmentGoal: String          # Specific treatment objective
  urgency: String               # Clinical urgency level
  specialConsiderations: [String!] # Special patient considerations
}
```

#### `ProviderContextInput`
Healthcare provider information.

```graphql
input ProviderContextInput {
  providerId: String!        # Unique provider identifier
  specialty: String         # Medical specialty
  experienceLevel: String   # Provider experience level
  organizationId: String    # Healthcare organization ID
  encounterId: String       # Current encounter ID
}
```

### Response Types

#### `MedicationOrchestrationResponse`
Comprehensive response covering all possible workflow outcomes.

```graphql
type MedicationOrchestrationResponse {
  status: OrchestrationStatus!    # SUCCESS, REQUIRES_PROVIDER_DECISION, BLOCKED_UNSAFE, ERROR, IN_PROGRESS, CANCELLED
  correlationId: String!
  
  # Success path - populated when status = SUCCESS
  medicationOrderId: String
  calculation: CalculationResult
  validation: ValidationResult  
  commitment: CommitmentResult
  performance: PerformanceMetrics
  
  # Warning path - populated when status = REQUIRES_PROVIDER_DECISION
  validationFindings: [ValidationFinding!]
  overrideTokens: [String!]
  proposals: [ProposalDetails!]
  snapshotId: String
  
  # Blocked path - populated when status = BLOCKED_UNSAFE
  blockingFindings: [ValidationFinding!]
  alternativeApproaches: [AlternativeProposal!]
  
  # Error handling
  errorCode: String
  errorMessage: String
  timestamp: DateTime!
}
```

## Usage Examples

### Basic Medication Request

```graphql
mutation {
  orchestrateMedicationRequest(input: {
    patientId: "patient_12345"
    medicationRequest: {
      medicationCode: "313782"
      medicationName: "Lisinopril 10mg"
      dosage: "10mg"
      frequency: "once daily"
      route: "oral"
      indication: "Hypertension"
    }
    clinicalIntent: {
      primaryIndication: "Essential hypertension"
      targetOutcome: "BP control <140/90"
      treatmentGoal: "Reduce cardiovascular risk"
    }
    providerContext: {
      providerId: "provider_789"
      specialty: "internal_medicine" 
      organizationId: "hospital_456"
    }
  }) {
    status
    correlationId
    medicationOrderId
    performance {
      totalTimeMs
      meetsTarget
    }
  }
}
```

### Handling Different Response Types

#### Success Response
```typescript
if (response.data.orchestrateMedicationRequest.status === "SUCCESS") {
  const result = response.data.orchestrateMedicationRequest;
  console.log(`Order created: ${result.medicationOrderId}`);
  console.log(`Performance: ${result.performance.totalTimeMs}ms`);
  console.log(`Meets targets: ${result.performance.meetsTarget}`);
}
```

#### Provider Decision Required
```typescript  
if (response.data.orchestrateMedicationRequest.status === "REQUIRES_PROVIDER_DECISION") {
  const result = response.data.orchestrateMedicationRequest;
  
  // Display validation findings to provider
  result.validationFindings.forEach(finding => {
    console.log(`${finding.severity}: ${finding.description}`);
    console.log(`Recommendation: ${finding.recommendation}`);
  });
  
  // Present override tokens for provider approval
  const overrideToken = result.overrideTokens[0];
  // Use override token with approveWithOverride mutation
}
```

#### Unsafe/Blocked Response
```typescript
if (response.data.orchestrateMedicationRequest.status === "BLOCKED_UNSAFE") {
  const result = response.data.orchestrateMedicationRequest;
  
  // Show blocking findings
  result.blockingFindings.forEach(finding => {
    console.error(`BLOCKED: ${finding.description}`);
  });
  
  // Present alternatives
  result.alternativeApproaches.forEach(alt => {
    console.log(`Alternative: ${alt.medicationName} - ${alt.rationale}`);
  });
}
```

### Override Scenarios

#### Provider Override with Token
```graphql
mutation {
  approveWithOverride(input: {
    correlationId: "corr_abc123"
    overrideToken: "override_token_xyz789"
    clinicianId: "provider_789"
    overrideReason: "Clinical judgment - benefits outweigh risks"
    selectedProposalId: "prop_456"
    acknowledgement: "I acknowledge the validation concerns and approve this medication"
  }) {
    status
    medicationOrderId
    performance {
      totalTimeMs
    }
  }
}
```

### Individual Phase Operations

For granular control, individual phases can be executed separately:

#### Calculate Only
```graphql
mutation {
  calculateMedicationProposal(input: {
    patientId: "patient_12345"
    medicationRequest: { ... }
    clinicalIntent: { ... }
    providerContext: { ... }
    correlationId: "corr_calculate_only"
    executionMode: "snapshot_optimized"
  }) {
    proposalSetId
    snapshotId
    rankedProposals {
      proposalId
      medicationData {
        name
        dosage
        confidenceScore
      }
    }
    executionTimeMs
    result
  }
}
```

#### Validate Existing Proposals
```graphql
mutation {
  validateMedicationProposal(input: {
    proposalSetId: "props_456def789"
    snapshotId: "snap_123456789"
    selectedProposals: [
      {
        proposalId: "prop_001"
        medicationCode: "313782"
        medicationName: "Lisinopril 10mg"
        dosage: "10mg"
        frequency: "once daily"
        route: "oral"
      }
    ]
    validationRequirements: {
      caeEngine: true
      protocolEngine: true
      comprehensiveValidation: true
      allergyCheck: true
      drugInteractionCheck: true
      doseValidation: true
    }
    correlationId: "corr_validate_only"
  }) {
    validationId
    verdict
    findings {
      severity
      category
      description
      recommendation
    }
    riskScore
    processingTimeMs
    result
  }
}
```

## Monitoring and Queries

### Workflow Status Monitoring
```graphql
query {
  workflowStatus(correlationId: "corr_abc123") {
    correlationId
    currentPhase
    status
    progress {
      completedPhases
      progressPercentage
      estimatedRemainingMs
    }
    startTime
    lastUpdate
  }
}
```

### Performance Metrics
```graphql
query {
  workflowMetrics(timeRangeHours: 24) {
    totalRequests
    successfulWorkflows
    warningWorkflows
    failedWorkflows
    averageProcessingTimeMs
    performanceTargetsMet
    phaseMetrics {
      calculatePhase {
        averageTimeMs
        successRate
        meetsTarget
      }
      validatePhase {
        averageTimeMs
        successRate  
        meetsTarget
      }
      commitPhase {
        averageTimeMs
        successRate
        meetsTarget
      }
    }
  }
}
```

### System Health
```graphql
query {
  orchestrationHealth {
    overall
    services {
      flow2GoEngine {
        status
        responseTimeMs
        version
      }
      safetyGateway {
        status
        responseTimeMs
        version
      }
      medicationService {
        status
        responseTimeMs
        version
      }
      contextGateway {
        status
        responseTimeMs
        version
      }
    }
    timestamp
    version
  }
}
```

## Error Handling

### GraphQL Error Structure
```typescript
interface GraphQLError {
  message: string;
  extensions: {
    code: string;
    details?: any;
    correlationId?: string;
  };
}
```

### Common Error Codes
- `CALCULATE_TIMEOUT`: Calculate phase exceeded time limits
- `VALIDATE_FAILED`: Validation service unavailable  
- `COMMIT_FAILED`: Commit operation failed
- `INVALID_INPUT`: Request validation failed
- `AUTHORIZATION_FAILED`: Authentication/authorization error
- `SYSTEM_UNAVAILABLE`: System temporarily unavailable

### Error Handling Example
```typescript
try {
  const response = await client.mutate({
    mutation: ORCHESTRATE_MEDICATION,
    variables: { input: requestData }
  });
  
  // Handle response based on status
  handleWorkflowResponse(response.data.orchestrateMedicationRequest);
  
} catch (error) {
  if (error.graphQLErrors?.length > 0) {
    const gqlError = error.graphQLErrors[0];
    console.error(`Workflow error [${gqlError.extensions?.code}]: ${gqlError.message}`);
    
    if (gqlError.extensions?.correlationId) {
      console.log(`Correlation ID for support: ${gqlError.extensions.correlationId}`);
    }
  }
  
  if (error.networkError) {
    console.error('Network error:', error.networkError.message);
  }
}
```

## Best Practices

### Performance Optimization
1. **Use correlation IDs**: Always provide correlation IDs for tracing
2. **Specify required fields**: Only query fields you need
3. **Batch requests**: Use GraphQL's batching capabilities
4. **Cache static data**: Cache medication codes and provider info

### Error Resilience  
1. **Implement retries**: Retry transient errors with exponential backoff
2. **Handle all statuses**: Account for SUCCESS, WARNING, BLOCKED, ERROR
3. **Store correlation IDs**: Save for troubleshooting and support
4. **Monitor performance**: Track response times and success rates

### Security
1. **Validate input**: Client-side validation for better UX
2. **Use proper auth**: Include valid JWT tokens
3. **Sanitize data**: Clean user inputs before submission
4. **Log security events**: Track authorization failures

### Integration Patterns
1. **UI State Management**: Map workflow status to UI states
2. **Progress Indicators**: Use workflow phases for progress tracking
3. **Caching Strategy**: Cache proposals and validation results appropriately
4. **Offline Support**: Handle network disconnections gracefully

---

This GraphQL schema provides a comprehensive interface for the Calculate > Validate > Commit workflow with full support for success, warning, and error scenarios. The schema is designed for production use with proper error handling, performance monitoring, and operational visibility.