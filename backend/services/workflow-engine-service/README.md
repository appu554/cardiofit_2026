# Workflow Engine Service

A comprehensive workflow management service for the Clinical Synthesis Hub, providing BPMN 2.0 workflow execution, task management, and FHIR integration.

## Overview

The Workflow Engine Service manages clinical workflows, human tasks, and process automation within the healthcare system. It integrates with Google Healthcare API for FHIR resource management and provides Apollo Federation support for GraphQL queries.

## Features

### Core Workflow Management
- **Workflow Definition Management**: Create, version, and deploy BPMN 2.0 workflows
- **Workflow Execution**: Execute workflow instances with state management
- **Task Management**: Create, assign, and track human tasks
- **Event Processing**: Handle workflow events and triggers
- **Timer Management**: Schedule time-based activities and escalations

### FHIR Integration
- **PlanDefinition Resources**: Store workflow definitions as FHIR PlanDefinition
- **Task Resources**: Manage human tasks as FHIR Task resources
- **Google Healthcare API**: Full integration with Google Cloud Healthcare API

### Apollo Federation
- **GraphQL Schema**: Comprehensive GraphQL API with Federation support
- **Cross-Service Queries**: Extend Patient and User types with workflow data
- **Real-time Updates**: Support for workflow and task status updates

## Architecture

```
API Gateway (8005) → Auth Service (8001) → Apollo Federation Gateway (4000) → Workflow Engine Service (8015) → Google Healthcare API
                                                                                                              → PostgreSQL Database
```

## Installation

1. **Install Dependencies**
   ```bash
   cd backend/services/workflow-engine-service
   pip install -r requirements.txt
   ```

2. **Set up Supabase Database**
   ```bash
   # Run the database setup script
   python setup_database.py

   # This will create all necessary tables in your Supabase PostgreSQL instance
   ```

3. **Configure Environment**
   ```bash
   cp .env.example .env
   # Edit .env with your configuration
   ```

4. **Set up Google Healthcare API Credentials**
   ```bash
   # Place your Google Cloud service account key in:
   credentials/google-credentials.json
   ```

## Configuration

Key environment variables:

- `SERVICE_PORT`: Service port (default: 8015)
- `SUPABASE_URL`: Supabase project URL
- `SUPABASE_KEY`: Supabase anon key
- `SUPABASE_JWT_SECRET`: Supabase JWT secret
- `DATABASE_URL`: Supabase PostgreSQL connection string
- `GOOGLE_CLOUD_PROJECT`: Google Cloud project ID
- `GOOGLE_APPLICATION_CREDENTIALS`: Path to Google Cloud credentials
- `AUTH_SERVICE_URL`: Authentication service URL
- `CAMUNDA_ENGINE_URL`: Local Camunda BPM engine URL (optional)
- `USE_CAMUNDA_CLOUD`: Enable Camunda Cloud integration (recommended)
- `CAMUNDA_CLOUD_CLIENT_ID`: Camunda Cloud client ID
- `CAMUNDA_CLOUD_CLIENT_SECRET`: Camunda Cloud client secret
- `CAMUNDA_CLOUD_CLUSTER_ID`: Camunda Cloud cluster ID
- `CAMUNDA_CLOUD_REGION`: Camunda Cloud region

## Running the Service

### Option 1: With Camunda Cloud (Recommended)

1. **Set up Camunda Cloud** (see `CAMUNDA_CLOUD_SETUP.md`)
2. **Configure environment**:
   ```bash
   cp .env.example .env
   # Edit .env with your Camunda Cloud credentials
   ```
3. **Run the service**:
   ```bash
   python run_service.py
   ```

### Option 2: Standalone Mode

```bash
# Development mode (without Camunda)
python run_service.py

# Or using uvicorn directly
uvicorn app.main:app --host 0.0.0.0 --port 8015 --reload
```

### Testing the Integration

```bash
# Test Camunda Cloud connection
python test_camunda_cloud.py

# Check service health
curl http://localhost:8015/health
```

## API Endpoints

### REST Endpoints
- `GET /`: Service information
- `GET /health`: Health check

### GraphQL Federation Endpoint
- `POST /api/federation`: Apollo Federation schema endpoint

## GraphQL Schema

### Queries
- `workflowDefinitions`: Get available workflow definitions
- `workflowDefinition(id)`: Get specific workflow definition
- `tasks`: Get tasks with filters
- `task(id)`: Get specific task
- `workflowInstances`: Get workflow instances

### Mutations
- `startWorkflow`: Start a new workflow instance
- `signalWorkflow`: Send signal to workflow
- `completeTask`: Complete a human task
- `claimTask`: Claim a task
- `delegateTask`: Delegate task to another user

### Federation Extensions
- `Patient.tasks`: Get tasks for a patient
- `Patient.workflowInstances`: Get workflow instances for a patient
- `User.assignedTasks`: Get tasks assigned to a user

## Database Schema

The service uses Supabase PostgreSQL for workflow state management:

- `workflow_definitions`: Workflow definition metadata
- `workflow_instances`: Active workflow instances
- `workflow_tasks`: Human tasks and assignments
- `workflow_events`: Audit trail and events
- `workflow_timers`: Scheduled events and timers

## FHIR Resources

### PlanDefinition
Workflow definitions are stored as FHIR PlanDefinition resources with BPMN XML in extensions.

### Task
Human tasks are created as FHIR Task resources with workflow context.

## Development

### Project Structure
```
app/
├── core/           # Configuration and settings
├── db/             # Database models and connection
├── models/         # SQLAlchemy models
├── services/       # Business logic services
├── graphql/        # GraphQL schema and resolvers
└── main.py         # FastAPI application
```

### Adding New Workflow Types
1. Create BPMN 2.0 workflow definition
2. Deploy as PlanDefinition resource
3. Configure task assignments and escalations
4. Test workflow execution

## Testing

```bash
# Run tests
pytest

# Run with coverage
pytest --cov=app
```

## Monitoring

The service provides health checks and logging:
- Health endpoint: `GET /health`
- Structured logging with correlation IDs
- Database connection monitoring
- Google Healthcare API status

## Security

- JWT token authentication via Auth Service
- RBAC integration for task assignments
- Secure Google Cloud credentials handling
- Input validation and sanitization

## Integration

### With Other Services
- **Patient Service**: Patient context for workflows
- **Order Service**: Order-based workflow triggers
- **Scheduling Service**: Appointment workflow integration
- **Encounter Service**: Encounter-based workflows

### External Systems
- **Google Healthcare API**: FHIR resource storage
- **Camunda BPM**: Optional workflow engine
- **PostgreSQL**: Workflow state persistence

## Troubleshooting

### Common Issues
1. **Database Connection**: Check Supabase PostgreSQL connection and credentials
2. **Google Healthcare API**: Verify service account permissions
3. **Federation**: Ensure Apollo Federation Gateway can reach the service
4. **Authentication**: Check Auth Service connectivity
5. **Supabase Setup**: Ensure tables are created with `python setup_database.py`

### Logs
Check service logs for detailed error information:
```bash
tail -f workflow-engine-service.log
```

## Contributing

1. Follow the established code patterns
2. Add tests for new functionality
3. Update documentation
4. Ensure FHIR compliance
5. Test Apollo Federation integration
