# Apollo Federation UI Interaction Deployment Guide

## Overview

This guide provides step-by-step instructions for deploying the enhanced Apollo Federation gateway with real-time UI interaction capabilities for clinical workflow management.

## Prerequisites

### Software Requirements
- **Node.js**: >= 16.0.0
- **npm**: >= 8.0.0
- **Redis**: >= 6.0 (for session management and pub/sub)
- **Apollo Rover CLI**: Latest version

### Service Dependencies
The following services must be running before starting the enhanced gateway:
- **Workflow Engine Go** (port 8015) - Primary service for UI interactions
- **Patient Service** (port 8003) - FHIR patient management
- **Medication Service** (port 8004) - Medication orchestration
- **Knowledge Base Services** (ports 8081-8089) - Clinical decision support

## Quick Start

### 1. Install Dependencies
```bash
cd apollo-federation
npm install
```

### 2. Install Apollo Rover CLI (if not already installed)
```bash
curl -sSL https://rover.apollo.dev/nix/latest | sh
source ~/.bashrc  # or ~/.zshrc
```

### 3. Start Required Services
```bash
# Start Redis (required for UI state management)
redis-server

# Start Workflow Engine Go (in another terminal)
cd ../backend/services/workflow-engine-service/workflow-engine-go
go run cmd/server/main.go

# Start other required services (follow CLAUDE.md instructions)
```

### 4. Generate Enhanced Supergraph
```bash
npm run generate-supergraph:ui
```

### 5. Run Integration Tests
```bash
npm run test:ui-integration
```

### 6. Start Enhanced Gateway
```bash
npm run start:ui
```

## Detailed Deployment Steps

### Step 1: Service Health Check

Before deployment, verify all services are healthy:

```bash
# Check service health
curl http://localhost:8015/health  # Workflow Engine
curl http://localhost:8003/health  # Patient Service
curl http://localhost:8004/health  # Medication Service

# Use the automated health check script
npm run test:connectivity
```

### Step 2: Schema Composition

The enhanced gateway combines schemas from multiple services:

```bash
# Generate supergraph with UI interaction schema
npm run generate-supergraph:ui

# This will create:
# - supergraph-ui-config.yaml (federation configuration)
# - supergraph-with-ui.graphql (combined schema)
```

Expected output:
```
✅ Service health checks passed: 12/14 services
🎨 UI interaction features detected: 4/4
📁 Supergraph schema generated: supergraph-with-ui.graphql
```

### Step 3: Configuration

#### Environment Variables
Create or update `.env` file:

```bash
# Gateway Configuration
PORT=4000
CORS_ORIGIN=http://localhost:3000

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379

# Service URLs
WORKFLOW_ENGINE_URL=http://localhost:8015
PATIENT_SERVICE_URL=http://localhost:8003
MEDICATION_SERVICE_URL=http://localhost:8004

# Security
JWT_SECRET=your-jwt-secret-here
AUTH_ENABLED=true

# Logging
LOG_LEVEL=info
ENABLE_QUERY_LOGGING=true
```

#### Redis Configuration
Ensure Redis is configured for pub/sub and session storage:

```bash
# Start Redis with appropriate configuration
redis-server --maxmemory 256mb --maxmemory-policy allkeys-lru
```

### Step 4: Testing

#### Automated Integration Tests
```bash
# Run all UI interaction tests
npm run test:ui-integration

# Expected test results:
# ✅ connectivity
# ✅ orchestration
# ✅ uiInteraction
# ✅ clinicalOverride
# ✅ pendingOverrides
```

#### Manual Testing
Test key functionality using GraphQL Playground at `http://localhost:4000/graphql`:

**1. Test Basic Workflow Orchestration:**
```graphql
mutation {
  orchestrateMedicationRequest(input: {
    patientId: "test-patient-123"
    medicationRequest: {
      medicationCode: "197361"
      medicationName: "Lisinopril"
      dosage: "10mg"
      frequency: "once daily"
    }
    clinicalIntent: {
      primaryIndication: "Hypertension"
    }
    providerContext: {
      providerId: "test-provider-123"
      specialty: "Cardiology"
    }
  }) {
    status
    correlationId
    validation {
      verdict
      riskScore
    }
    overrideTokens
  }
}
```

**2. Test UI Notification:**
```graphql
mutation {
  updateUINotification(
    workflowId: "test-workflow-123"
    notification: {
      status: ACTION_REQUIRED
      title: "Clinical Override Required"
      message: "Please review validation findings"
      severity: WARNING
    }
  ) {
    id
    workflowId
    status
    actions {
      id
      label
      type
    }
  }
}
```

**3. Test Real-time Subscription:**
```graphql
subscription {
  workflowUIUpdates(workflowId: "test-workflow-123") {
    workflowId
    updateType
    notification {
      title
      message
      severity
    }
    timestamp
  }
}
```

### Step 5: Production Deployment

#### Docker Deployment

**Create Dockerfile:**
```dockerfile
FROM node:18-alpine

WORKDIR /app

COPY package*.json ./
RUN npm ci --only=production

COPY . .

EXPOSE 4000

CMD ["npm", "run", "start:ui"]
```

**Create docker-compose.yml:**
```yaml
version: '3.8'

services:
  apollo-federation-ui:
    build: .
    ports:
      - "4000:4000"
    environment:
      - REDIS_HOST=redis
      - WORKFLOW_ENGINE_URL=http://workflow-engine:8015
    depends_on:
      - redis
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:4000/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    command: redis-server --maxmemory 256mb --maxmemory-policy allkeys-lru
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 3
```

#### Kubernetes Deployment

**Create k8s deployment:**
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: apollo-federation-ui
spec:
  replicas: 3
  selector:
    matchLabels:
      app: apollo-federation-ui
  template:
    metadata:
      labels:
        app: apollo-federation-ui
    spec:
      containers:
      - name: gateway
        image: your-registry/apollo-federation-ui:latest
        ports:
        - containerPort: 4000
        env:
        - name: REDIS_HOST
          value: "redis-service"
        - name: WORKFLOW_ENGINE_URL
          value: "http://workflow-engine-service:8015"
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 4000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 4000
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: apollo-federation-ui-service
spec:
  selector:
    app: apollo-federation-ui
  ports:
  - port: 4000
    targetPort: 4000
  type: LoadBalancer
```

## Monitoring and Observability

### Health Checks

The enhanced gateway provides comprehensive health endpoints:

```bash
# Basic health check
curl http://localhost:4000/health

# Detailed health with service status
curl http://localhost:4000/graphql -X POST -H "Content-Type: application/json" \
  -d '{"query": "query { orchestrationHealth { overall services { workflows { status responseTimeMs } } } }"}'
```

### Metrics and Logging

**Enable structured logging:**
```javascript
// Set environment variables
LOG_LEVEL=debug
ENABLE_QUERY_LOGGING=true
ENABLE_PERFORMANCE_LOGGING=true
```

**Key metrics to monitor:**
- GraphQL query response times
- WebSocket connection count
- Override request frequency
- Service health status
- Redis connection status

### Performance Tuning

**Redis Optimization:**
```bash
# Configure Redis for optimal performance
redis-cli CONFIG SET maxmemory-policy allkeys-lru
redis-cli CONFIG SET maxmemory 512mb
redis-cli CONFIG SET timeout 300
```

**Node.js Optimization:**
```bash
# Set Node.js options for production
export NODE_OPTIONS="--max-old-space-size=2048"
export UV_THREADPOOL_SIZE=128
```

## Troubleshooting

### Common Issues

**1. Schema Composition Errors**
```bash
# Check service federation endpoints
curl http://localhost:8015/api/federation -X POST \
  -H "Content-Type: application/json" \
  -d '{"query": "query { _service { sdl } }"}'
```

**2. WebSocket Connection Issues**
```bash
# Test WebSocket connection
wscat -c ws://localhost:4000/subscriptions
```

**3. Redis Connection Problems**
```bash
# Test Redis connectivity
redis-cli ping
redis-cli INFO replication
```

**4. Service Discovery Issues**
```bash
# Run service health check script
npm run test:connectivity
```

### Debugging GraphQL

Enable detailed query logging:
```bash
export DEBUG=apollo-gateway:*
npm run dev:ui
```

### Performance Issues

Monitor query performance:
```graphql
query {
  __schema {
    queryType {
      name
    }
  }
}
```

## Security Considerations

### Authentication
- JWT tokens validated on all requests
- Role-based access control for override operations
- Clinician authority level validation

### Data Protection
- All override decisions cryptographically signed
- Audit trails maintained for compliance
- PHI data handling follows HIPAA guidelines

### Network Security
- HTTPS required in production
- CORS configured for approved origins
- Rate limiting on GraphQL endpoints

## Support and Documentation

### API Documentation
- GraphQL Playground: `http://localhost:4000/graphql`
- Schema SDL: Available via introspection
- Real-time API: WebSocket endpoint at `/subscriptions`

### Additional Resources
- [Apollo Federation Documentation](https://www.apollographql.com/docs/federation/)
- [GraphQL WebSocket Protocol](https://github.com/enisdenjo/graphql-ws)
- [Clinical Workflow Integration Guide](./ADVANCED_UI_INTERACTION_GUIDE.md)

For support, refer to the project README or create an issue in the repository.