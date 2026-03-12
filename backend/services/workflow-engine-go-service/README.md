# Workflow Engine Go Service

## 🚀 Advanced 3-Phase Pattern Implementation

This is the **primary** Workflow Engine service for the Clinical Synthesis Hub CardioFit platform, implementing the Advanced Calculate → Validate → Commit pattern with real-time UI interaction capabilities.

> **Note**: The Python implementation in `workflow-engine-service/` is now considered **LEGACY** and should only be used for backward compatibility.

## Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────────┐
│   Frontend UI   │◄──►│ Apollo Federation│◄──►│  This Go Service    │
│   (React/Vue)   │    │   (GraphQL)      │    │ (Advanced Pattern)  │
└─────────────────┘    └──────────────────┘    └─────────────────────┘
                                │                         │
                                │                         ▼
                       ┌────────▼─────────┐    ┌─────────────────────┐
                       │  Redis/PubSub    │    │   External Services │
                       │  (Session State) │    │ • Flow2 Go Engine   │
                       └──────────────────┘    │ • Safety Gateway    │
                                               │ • Medication Service │
                                               └─────────────────────┘
```

## 🎯 Key Features

### Advanced 3-Phase Pattern
- **Calculate Phase**: Integration with Flow2 Go/Rust engines for medication intelligence
- **Validate Phase**: Comprehensive safety validation via Safety Gateway
- **Commit Phase**: Idempotent persistence with full audit trail

### Real-Time UI Interaction
- **UI Coordinator**: Bidirectional communication with Apollo Federation
- **Override Governance**: Hierarchical clinical override framework
- **WebSocket Support**: Real-time notifications and updates
- **Session Management**: Redis-based state persistence

### Production-Ready Components
- **Idempotency Manager**: Transaction safety and retry protection
- **Circuit Breaker**: Resilient service communication
- **Monitoring**: Prometheus metrics and health checks
- **Distributed Tracing**: OpenTelemetry integration

## 📁 Project Structure

```
workflow-engine-go-service/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── domain/                  # Domain models and business logic
│   │   ├── workflow.go         # Workflow domain entities
│   │   └── override.go         # Override governance models
│   ├── orchestration/           # Core orchestration logic
│   │   ├── strategic_orchestrator.go  # Advanced pattern coordinator
│   │   ├── ui_coordinator.go          # UI interaction handler
│   │   ├── override.go                # Override governance framework
│   │   └── idempotency.go            # Idempotency protection
│   ├── repositories/            # Data persistence layer
│   │   ├── workflow_repo.go    # Workflow state persistence
│   │   └── snapshot_repo.go    # Snapshot management
│   └── services/                # Business services
│       └── orchestration_service.go  # Service layer
├── pkg/
│   └── clients/                 # External service clients
│       ├── flow2_client.go     # Flow2 Go Engine client
│       ├── safety_client.go    # Safety Gateway client
│       └── medication_client.go # Medication Service client
├── graph/                       # GraphQL schema and resolvers
│   ├── schema.graphqls         # GraphQL schema
│   └── resolver.go             # GraphQL resolvers
├── migrations/                  # Database migrations
├── scripts/                     # Utility scripts
├── doc/                        # Documentation
├── go.mod                      # Go module definition
├── go.sum                      # Go module checksums
├── Makefile                    # Build and development commands
└── docker-compose.yml          # Local development environment
```

## 🚦 Service Status

| Component | Status | Description |
|-----------|--------|-------------|
| **Strategic Orchestrator** | ✅ Complete | Advanced 3-phase pattern with UI interaction |
| **UI Coordinator** | ✅ Complete | Apollo Federation bidirectional communication |
| **Override Governance** | ✅ Complete | Hierarchical clinical override framework |
| **Idempotency Manager** | ✅ Complete | Transaction safety and retry logic |
| **WebSocket Server** | ⚠️ Pending | Real-time subscriptions (Apollo side) |
| **Unit Tests** | ❌ Missing | Need comprehensive test coverage |
| **Integration Tests** | ❌ Missing | End-to-end workflow testing |
| **Documentation** | ✅ Complete | This README and inline documentation |

## 🔧 Installation & Setup

### Prerequisites
- Go 1.21 or higher
- PostgreSQL 14+
- Redis 6+
- Docker & Docker Compose (for local development)

### Quick Start

1. **Clone and navigate to the service:**
```bash
cd backend/services/workflow-engine-go-service
```

2. **Set up environment variables:**
```bash
cp .env.example .env
# Edit .env with your configuration
```

3. **Install dependencies:**
```bash
go mod download
go mod tidy
```

4. **Run database migrations:**
```bash
make migrate-up
```

5. **Start the service:**
```bash
# Development mode with hot reload
make dev

# Or production mode
make run
```

## 🔌 API Endpoints

### REST API
- `POST /api/v1/workflow/execute` - Execute medication workflow
- `POST /api/v1/workflow/override` - Process clinical override
- `GET /api/v1/workflow/:id/status` - Get workflow status
- `GET /health` - Health check endpoint
- `GET /metrics` - Prometheus metrics

### GraphQL (via Apollo Federation)
```graphql
mutation ExecuteMedicationWorkflow($input: MedicationWorkflowInput!) {
  executeMedicationWorkflow(input: $input) {
    workflowInstanceId
    status
    validationResult {
      verdict
      findings
    }
    overrideSession {
      sessionId
      requiredLevel
    }
  }
}

subscription WorkflowUIUpdates($workflowId: ID!) {
  workflowUIUpdates(workflowId: $workflowId) {
    phase
    status
    notification {
      title
      message
      severity
    }
  }
}
```

## 🧪 Testing

```bash
# Run all tests
make test

# Run with coverage
make test-coverage

# Run specific test
go test ./internal/orchestration -run TestStrategicOrchestrator

# Run integration tests
make test-integration
```

## 📊 Monitoring

### Prometheus Metrics
- `workflow_requests_total` - Total workflow requests by status
- `workflow_phase_duration_seconds` - Phase execution duration
- `safety_gateway_validations_total` - Validation requests by verdict
- `workflow_active_total` - Currently active workflows

### Health Checks
```bash
curl http://localhost:8020/health
```

Response:
```json
{
  "status": "healthy",
  "services": {
    "flow2_go": "healthy",
    "safety_gateway": "healthy",
    "medication_service": "healthy",
    "redis": "healthy",
    "database": "healthy"
  }
}
```

## 🔄 Migration from Python (Legacy)

If you're currently using the Python workflow engine service:

1. **Gradual Migration**: Both services can run in parallel
2. **Feature Flag**: Use feature flags to route traffic
3. **Data Migration**: Workflow states are compatible
4. **Rollback Plan**: Python service remains as fallback

### Routing Logic Example
```go
func RouteWorkflow(request *WorkflowRequest) string {
    // Route advanced workflows to Go service
    if request.RequiresUIInteraction ||
       request.RequiresOverride ||
       request.Priority == "HIGH" {
        return "go-service"
    }
    // Legacy workflows can still use Python
    return "python-legacy"
}
```

## 🤝 Contributing

1. Create a feature branch
2. Implement changes with tests
3. Update documentation
4. Submit pull request

## 📝 License

Proprietary - Clinical Synthesis Hub

## 🆘 Support

For issues or questions:
- Check [API Documentation](./doc/API.md)
- Review [Architecture Guide](./doc/ARCHITECTURE.md)
- Contact the platform team

---

**Note**: This service is the primary implementation for the Workflow Engine. The Python implementation at `../workflow-engine-service/` is maintained for backward compatibility only.