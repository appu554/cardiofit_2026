# Workflow Engine Service API Documentation

## Overview

The Workflow Engine Service provides comprehensive workflow management capabilities through a GraphQL API with Apollo Federation integration. This service manages workflow definitions, workflow instances, and human tasks within the Clinical Synthesis Hub.

## Base URL

- **Service URL**: `http://localhost:8015`
- **Federation Endpoint**: `http://localhost:8015/api/federation`
- **Health Check**: `http://localhost:8015/health`

## Authentication

All API requests require authentication through the API Gateway. Include the following headers:

```http
Authorization: Bearer <token>
X-User-ID: <user-id>
X-User-Role: <user-role>
X-User-Roles: <comma-separated-roles>
X-User-Permissions: <comma-separated-permissions>
```

## GraphQL Schema

### Core Types

#### WorkflowDefinition
```graphql
type WorkflowDefinition @key(fields: "id") {
  id: ID!
  name: String!
  description: String
  version: Int!
  bpmnXml: String!
  category: String
  isActive: Boolean!
  createdAt: DateTime!
  updatedAt: DateTime!
}
```

#### WorkflowInstance
```graphql
type WorkflowInstance @key(fields: "id") {
  id: ID!
  definitionId: ID!
  patientId: ID!
  status: WorkflowStatus!
  variables: JSON
  createdAt: DateTime!
  updatedAt: DateTime!
  startTime: DateTime
  endTime: DateTime
}
```

#### Task
```graphql
type Task @key(fields: "id") {
  id: ID!
  name: String!
  description: String
  status: TaskStatus!
  assigneeId: ID
  workflowInstanceId: ID!
  formData: JSON
  outputVariables: JSON
  dueDate: DateTime
  createdAt: DateTime!
  updatedAt: DateTime!
  claimedAt: DateTime
  completedAt: DateTime
}
```

### Enums

#### WorkflowStatus
```graphql
enum WorkflowStatus {
  ACTIVE
  COMPLETED
  CANCELED
  FAILED
  SUSPENDED
}
```

#### TaskStatus
```graphql
enum TaskStatus {
  READY
  CLAIMED
  COMPLETED
  CANCELED
  FAILED
}
```

### Input Types

#### KeyValuePairInput
```graphql
input KeyValuePairInput {
  key: String!
  value: String!
}
```

## Queries

### workflowDefinitions
Get a list of workflow definitions, optionally filtered by category.

```graphql
query GetWorkflowDefinitions($category: String) {
  workflowDefinitions(category: $category) {
    id
    name
    description
    version
    category
    isActive
    createdAt
    updatedAt
  }
}
```

**Parameters:**
- `category` (optional): Filter by workflow category

**Example:**
```json
{
  "query": "query GetWorkflowDefinitions($category: String) { workflowDefinitions(category: $category) { id name description version category isActive } }",
  "variables": {
    "category": "admission"
  }
}
```

### workflowDefinition
Get a specific workflow definition by ID.

```graphql
query GetWorkflowDefinition($id: ID!) {
  workflowDefinition(id: $id) {
    id
    name
    description
    version
    bpmnXml
    category
    isActive
    createdAt
    updatedAt
  }
}
```

**Parameters:**
- `id` (required): Workflow definition ID

### workflowInstances
Get a list of workflow instances, optionally filtered by status or patient ID.

```graphql
query GetWorkflowInstances($status: WorkflowStatus, $patientId: ID) {
  workflowInstances(status: $status, patientId: $patientId) {
    id
    definitionId
    patientId
    status
    variables
    createdAt
    updatedAt
    startTime
    endTime
  }
}
```

**Parameters:**
- `status` (optional): Filter by workflow status
- `patientId` (optional): Filter by patient ID

### tasks
Get a list of tasks assigned to a specific user, optionally filtered by status.

```graphql
query GetTasks($assignee: ID!, $status: TaskStatus) {
  tasks(assignee: $assignee, status: $status) {
    id
    name
    description
    status
    assigneeId
    workflowInstanceId
    formData
    dueDate
    createdAt
    updatedAt
  }
}
```

**Parameters:**
- `assignee` (required): User ID of the task assignee
- `status` (optional): Filter by task status

### task
Get a specific task by ID.

```graphql
query GetTask($id: ID!) {
  task(id: $id) {
    id
    name
    description
    status
    assigneeId
    workflowInstanceId
    formData
    outputVariables
    dueDate
    createdAt
    updatedAt
    claimedAt
    completedAt
  }
}
```

**Parameters:**
- `id` (required): Task ID

## Mutations

### startWorkflow
Start a new workflow instance.

```graphql
mutation StartWorkflow($definitionId: ID!, $patientId: ID!, $variables: [KeyValuePairInput]) {
  startWorkflow(definitionId: $definitionId, patientId: $patientId, initialVariables: $variables) {
    id
    definitionId
    patientId
    status
    variables
    createdAt
  }
}
```

**Parameters:**
- `definitionId` (required): ID of the workflow definition to start
- `patientId` (required): ID of the patient for whom the workflow is started
- `variables` (optional): Initial workflow variables

**Example:**
```json
{
  "query": "mutation StartWorkflow($definitionId: ID!, $patientId: ID!, $variables: [KeyValuePairInput]) { startWorkflow(definitionId: $definitionId, patientId: $patientId, initialVariables: $variables) { id definitionId patientId status variables createdAt } }",
  "variables": {
    "definitionId": "patient-admission-workflow",
    "patientId": "patient-123",
    "variables": [
      {"key": "patientName", "value": "John Doe"},
      {"key": "admissionType", "value": "emergency"},
      {"key": "priority", "value": "high"}
    ]
  }
}
```

### signalWorkflow
Send a signal to a running workflow instance.

```graphql
mutation SignalWorkflow($instanceId: ID!, $signalName: String!, $variables: [KeyValuePairInput]) {
  signalWorkflow(instanceId: $instanceId, signalName: $signalName, variables: $variables)
}
```

**Parameters:**
- `instanceId` (required): ID of the workflow instance
- `signalName` (required): Name of the signal to send
- `variables` (optional): Signal variables

### claimTask
Claim a task for the current user.

```graphql
mutation ClaimTask($taskId: ID!) {
  claimTask(taskId: $taskId) {
    id
    status
    assigneeId
    claimedAt
  }
}
```

**Parameters:**
- `taskId` (required): ID of the task to claim

### completeTask
Complete a task with output variables.

```graphql
mutation CompleteTask($taskId: ID!, $outputVariables: [KeyValuePairInput]) {
  completeTask(taskId: $taskId, outputVariables: $outputVariables) {
    id
    status
    outputVariables
    completedAt
  }
}
```

**Parameters:**
- `taskId` (required): ID of the task to complete
- `outputVariables` (optional): Task output variables

**Example:**
```json
{
  "query": "mutation CompleteTask($taskId: ID!, $outputVariables: [KeyValuePairInput]) { completeTask(taskId: $taskId, outputVariables: $outputVariables) { id status outputVariables completedAt } }",
  "variables": {
    "taskId": "task-123",
    "outputVariables": [
      {"key": "reviewResult", "value": "approved"},
      {"key": "notes", "value": "Patient data verified"},
      {"key": "reviewedBy", "value": "Dr. Smith"}
    ]
  }
}
```

### delegateTask
Delegate a task to another user.

```graphql
mutation DelegateTask($taskId: ID!, $userId: ID!) {
  delegateTask(taskId: $taskId, userId: $userId) {
    id
    assigneeId
    delegatedAt
    delegatedBy
  }
}
```

**Parameters:**
- `taskId` (required): ID of the task to delegate
- `userId` (required): ID of the user to delegate the task to

## Federation Integration

The Workflow Engine Service integrates with Apollo Federation to extend Patient and User types:

### Extended Patient Type
```graphql
extend type Patient @key(fields: "id") {
  id: ID! @external
  tasks(status: TaskStatus): [Task]
  workflowInstances(status: WorkflowStatus): [WorkflowInstance]
}
```

### Extended User Type
```graphql
extend type User @key(fields: "id") {
  id: ID! @external
  assignedTasks(status: TaskStatus): [Task]
}
```

## Error Handling

The API returns standard GraphQL errors with additional context:

```json
{
  "errors": [
    {
      "message": "Task not found",
      "locations": [{"line": 2, "column": 3}],
      "path": ["task"],
      "extensions": {
        "code": "NOT_FOUND",
        "taskId": "invalid-task-id"
      }
    }
  ],
  "data": null
}
```

## Rate Limiting

API requests are subject to rate limiting:
- 1000 requests per minute per user
- 100 concurrent requests per user

## Health Check

Check service health:

```http
GET /health
```

Response:
```json
{
  "status": "healthy",
  "service": "workflow-engine-service",
  "version": "1.0.0",
  "timestamp": "2024-01-01T00:00:00Z",
  "dependencies": {
    "database": "healthy",
    "camunda": "healthy",
    "google_fhir": "healthy"
  }
}
```
