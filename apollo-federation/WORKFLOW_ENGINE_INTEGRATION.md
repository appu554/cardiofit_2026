# WorkflowEngineService Apollo Federation Integration

## ✅ Integration Status: IN PROGRESS

The WorkflowEngineService is being integrated into the Apollo Federation Gateway configuration with all necessary federation directives and entity extensions.

## 🔧 Configuration Changes Made

### 1. Apollo Federation Gateway Files Updated

#### **supergraph.yaml**
```yaml
workflows:
  routing_url: http://localhost:8015/api/federation
  schema:
    subgraph_url: http://localhost:8015/api/federation
```

#### **rover-gateway.js**
```javascript
// Added workflow service to service list
{ name: 'workflows', url: (process.env.WORKFLOW_ENGINE_SERVICE_URL || 'http://localhost:8015/api/federation') }
```

#### **generate-supergraph.js**
```javascript
// Added workflow service to service list
{
  name: 'workflows',
  url: (process.env.WORKFLOW_ENGINE_SERVICE_URL || 'http://localhost:8015/api/federation')
}
```

#### **index.js**
```javascript
// Added workflow service to service list
{
  name: 'workflows',
  url: 'http://localhost:8015/api/federation'
}
```

#### **.env.example**
```bash
# Added workflow engine service URL
WORKFLOW_ENGINE_SERVICE_URL=http://localhost:8015/api
```

### 2. API Gateway Configuration Updated

#### **backend/services/api-gateway/app/config.py**
```python
# Added workflow engine service URL
WORKFLOW_ENGINE_SERVICE_URL: str = os.getenv("WORKFLOW_ENGINE_SERVICE_URL", "http://localhost:8015")
```

## 🚀 Integration Steps

### Step 1: Start the WorkflowEngineService
```bash
cd backend/services/workflow-engine-service
python run_service.py
```

**Verify**: Service should be running on port 8015 with federation endpoint at `/api/federation`

### Step 2: Regenerate Supergraph Schema
```bash
cd apollo-federation
node regenerate-supergraph-with-workflows.js
```

**This script will**:
- Check health of all services including workflow service
- Validate federation endpoints are available
- Generate new supergraph schema with workflow types
- Validate the schema includes all workflow-related types

### Step 3: Start Apollo Federation Gateway
```bash
cd apollo-federation
npm start
```

**Verify**: Gateway should start successfully and include workflow service in the federated schema

### Step 4: Test Federation Integration
Use the provided Postman collection to test:
```bash
# Import: backend/services/workflow-engine-service/postman/WorkflowEngine_Federation_Flow.json
# Test the complete flow: API Gateway → Auth → Apollo Federation → Workflow Service
```

## 📊 Federation Schema Overview

### Entity Extensions Added
- **Patient.tasks**: Get tasks assigned to a patient
- **Patient.workflowInstances**: Get workflow instances for a patient
- **User.assignedTasks**: Get tasks assigned to a user

### Core Types Available
- **WorkflowDefinition**: Workflow definition management
- **WorkflowInstance_Summary**: Workflow instance information
- **Task**: Human tasks with FHIR Task integration

### Key Queries
- `workflowDefinitions`: Get available workflow definitions
- `workflowDefinition(id)`: Get specific workflow definition
- `tasks`: Get tasks with filters
- `task(id)`: Get specific task
- `workflowInstances`: Get workflow instances

### Key Mutations
- `startWorkflow`: Start a new workflow instance
- `signalWorkflow`: Send signal to workflow
- `completeTask`: Complete a human task
- `claimTask`: Claim a task
- `delegateTask`: Delegate task to another user

## 🧪 Testing Queries

### Example Federation Queries

#### Get Patient with Tasks
```graphql
query GetPatientWithTasks($patientId: ID!) {
  patient(id: $patientId) {
    id
    name {
      family
      given
    }
    tasks(status: READY) {
      id
      description
      priority
      status
    }
    workflowInstances(status: ACTIVE) {
      id
      status
      startTime
    }
  }
}
```

#### Get User with Assigned Tasks
```graphql
query GetUserTasks($userId: ID!) {
  user(id: $userId) {
    id
    assignedTasks(status: READY) {
      id
      description
      priority
      dueDate
      for {
        reference
        display
      }
    }
  }
}
```

#### Start Workflow for Patient
```graphql
mutation StartPatientWorkflow($patientId: ID!) {
  startWorkflow(
    definitionId: "1"
    patientId: $patientId
    initialVariables: [
      { key: "patientData", value: "{\"name\":\"John Doe\"}" }
    ]
  ) {
    id
    status
    startTime
  }
}
```

## 🔍 Validation Checklist

- [ ] WorkflowEngineService running on port 8015
- [ ] Federation endpoint `/api/federation` accessible
- [ ] Supergraph schema generated successfully
- [ ] Apollo Federation Gateway includes workflow service
- [ ] Patient.tasks extension working
- [ ] Patient.workflowInstances extension working
- [ ] User.assignedTasks extension working
- [ ] Cross-service queries functioning
- [ ] Workflow mutations working through federation

## 📝 Next Steps

1. **Complete supergraph generation**
2. **Test federation queries**
3. **Validate entity extensions**
4. **Create comprehensive test suite**
5. **Update documentation**
