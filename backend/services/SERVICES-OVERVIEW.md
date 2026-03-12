# CardioFit Backend Services - Complete Overview

## Service Architecture Summary

This document provides a consolidated view of the three backend services currently running in the CardioFit clinical platform ecosystem.

## 🏥 Services Overview

### 1. Patient Service (Python/FastAPI)
**Location**: `backend/services/patient-service/`
**Technology**: Python 3.11+ with FastAPI framework
**Port**: 8003
**Purpose**: FHIR-compliant patient data management

**Quick Start**:
```bash
cd backend/services/patient-service
python3 -m venv venv && source venv/bin/activate
pip3 install -r requirements.txt
python3 run_service.py
```

**Health Check**: `curl http://localhost:8003/health`

---

### 2. Medication Service V2 (Go/Gin)
**Location**: `backend/services/medication-service-v2/`
**Technology**: Go 1.21+ with Gin web framework
**Port**: 8005
**Purpose**: Medication management and clinical decision support

**Quick Start**:
```bash
cd backend/services/medication-service-v2
go mod tidy
go build -o medication-service-v2 cmd/simple-server/main.go
./medication-service-v2
```

**Health Check**: `curl http://localhost:8005/health`

---

### 3. Context Gateway Service (Go/gRPC+HTTP)
**Location**: `backend/services/context-gateway-go/`
**Technology**: Go 1.23+ with gRPC and HTTP APIs
**Ports**: 8017 (gRPC), 8117 (HTTP)
**Purpose**: Clinical snapshot and recipe management with dual-layer storage

**Dependencies**: MongoDB + Redis (via Docker)

**Quick Start**:
```bash
cd backend/services/context-gateway-go
docker-compose up -d  # Start MongoDB & Redis
go mod tidy
go build -o context-gateway cmd/main.go
./context-gateway
```

**Health Check**: `curl http://localhost:8117/health`

## 🚀 One-Command Startup (All Services)

### Prerequisites Check
```bash
# Check required tools
python3 --version  # 3.11+
go version         # 1.21+
docker --version   # For Context Gateway dependencies
```

### Sequential Startup Script
```bash
#!/bin/bash
set -e

echo "🏥 Starting CardioFit Backend Services..."

# 1. Start Context Gateway dependencies
echo "📦 Starting infrastructure dependencies..."
cd backend/services/context-gateway-go
docker-compose up -d
cd ../../..

# 2. Start Patient Service
echo "👤 Starting Patient Service..."
cd backend/services/patient-service
python3 -m venv venv 2>/dev/null || true
source venv/bin/activate
pip3 install -r requirements.txt -q
python3 run_service.py &
PATIENT_PID=$!
cd ../../..

# 3. Start Medication Service V2
echo "💊 Starting Medication Service V2..."
cd backend/services/medication-service-v2
go mod tidy -q
go build -o medication-service-v2 cmd/simple-server/main.go
./medication-service-v2 &
MEDICATION_PID=$!
cd ../../..

# 4. Start Context Gateway
echo "🌐 Starting Context Gateway..."
cd backend/services/context-gateway-go
go mod tidy -q
go build -o context-gateway cmd/main.go
./context-gateway &
CONTEXT_PID=$!
cd ../../..

# Wait for services to start
sleep 5

# Health checks
echo "🔍 Performing health checks..."
curl -s http://localhost:8003/health >/dev/null && echo "✅ Patient Service (8003) - Healthy"
curl -s http://localhost:8005/health >/dev/null && echo "✅ Medication Service (8005) - Healthy"
curl -s http://localhost:8117/health >/dev/null && echo "✅ Context Gateway (8117) - Healthy"

echo "🎉 All services started successfully!"
echo "   Patient Service:    http://localhost:8003"
echo "   Medication Service: http://localhost:8005"
echo "   Context Gateway:    http://localhost:8117"
echo ""
echo "💡 To stop all services, run: pkill -f 'run_service.py|medication-service-v2|context-gateway'"
```

## 🔗 Service Integration Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Patient       │    │   Medication    │    │   Context       │
│   Service       │    │   Service V2    │    │   Gateway       │
│   (Python)      │    │   (Go)          │    │   (Go)          │
│   Port: 8003    │    │   Port: 8005    │    │   Port: 8017/17 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
         ┌─────────────────────────────────────────────┐
         │          Apollo Federation                  │
         │          (GraphQL Gateway)                  │
         │          Port: 4000                        │
         └─────────────────────────────────────────────┘
                                 │
         ┌─────────────────────────────────────────────┐
         │          Frontend Application               │
         │          (Angular)                         │
         └─────────────────────────────────────────────┘
```

## 📊 Service Specifications Comparison

| Feature | Patient Service | Medication Service V2 | Context Gateway |
|---------|-----------------|----------------------|-----------------|
| **Language** | Python 3.11+ | Go 1.21+ | Go 1.23+ |
| **Framework** | FastAPI | Gin | gRPC + HTTP |
| **Primary Port** | 8003 | 8005 | 8017 (gRPC) + 8117 (HTTP) |
| **Database** | Shared modules | Stateless | MongoDB + Redis |
| **Dependencies** | Virtual env | Go modules | Docker containers |
| **Startup Time** | ~3 seconds | ~1 second | ~5 seconds |
| **API Style** | RESTful JSON | RESTful JSON | gRPC + GraphQL Federation |
| **FHIR Support** | ✅ Full | ❌ Planned | ✅ Context-aware |
| **Health Endpoint** | `/health` | `/health` | `/health` + `/status` |

## 🛠️ Development Workflow

### Local Development Setup
```bash
# 1. Clone and navigate
git clone <repository>
cd cardiofit/backend/services

# 2. Set up each service (see individual STARTUP-GUIDE.md files)

# 3. Start in order: Context Gateway deps → Patient → Medication → Context Gateway
```

### Testing All Services
```bash
# Quick health check all services
curl -s http://localhost:8003/health | jq
curl -s http://localhost:8005/health | jq
curl -s http://localhost:8117/health | jq

# Detailed Context Gateway status
curl -s http://localhost:8117/status | jq
```

### Integration Testing
```bash
# Test service communication patterns
# Patient → Context Gateway
# Medication → Context Gateway
# All → Apollo Federation (when available)
```

## 🏗️ Production Deployment

### Docker Compose (Complete Stack)
```yaml
version: '3.8'
services:
  patient-service:
    build: ./patient-service
    ports: ["8003:8003"]
    depends_on: [mongodb]

  medication-service:
    build: ./medication-service-v2
    ports: ["8005:8005"]

  context-gateway:
    build: ./context-gateway-go
    ports: ["8017:8017", "8117:8117"]
    depends_on: [mongodb, redis]

  mongodb:
    image: mongo:7.0
    ports: ["27017:27017"]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
```

### Kubernetes Deployment
Each service includes production-ready configurations for:
- Resource limits and requests
- Health check probes
- Service discovery
- Configuration management
- Horizontal scaling

## 📚 Documentation Links

### Individual Service Guides
- **Patient Service**: [`patient-service/STARTUP-GUIDE.md`](./patient-service/STARTUP-GUIDE.md)
- **Medication Service V2**: [`medication-service-v2/STARTUP-GUIDE.md`](./medication-service-v2/STARTUP-GUIDE.md)
- **Context Gateway**: [`context-gateway-go/STARTUP-GUIDE.md`](./context-gateway-go/STARTUP-GUIDE.md)

### API Documentation
- Patient Service: RESTful APIs for FHIR patient resources
- Medication Service V2: RESTful APIs for medication management
- Context Gateway: gRPC APIs + GraphQL Federation + HTTP endpoints

### Architecture Documentation
- Clinical data flow patterns
- FHIR compliance implementation
- Microservices communication patterns
- Security and audit trail implementation

## 🚨 Troubleshooting Quick Reference

### Common Port Conflicts
```bash
# Check what's using service ports
lsof -i :8003 :8005 :8017 :8117 :27017 :6379

# Kill conflicting processes
pkill -f 'run_service.py|medication-service-v2|context-gateway'
```

### Quick Service Restart
```bash
# Individual service restart
cd backend/services/patient-service && python3 run_service.py &
cd backend/services/medication-service-v2 && ./medication-service-v2 &
cd backend/services/context-gateway-go && ./context-gateway &
```

### Dependencies Check
```bash
# Patient Service
python3 --version && pip3 list | grep -E 'fastapi|requests|fhir'

# Medication Service V2
go version && go list -m all | grep gin

# Context Gateway
go version && docker ps | grep -E 'mongo|redis'
```

## 💡 Next Steps

1. **Apollo Federation Integration**: Set up GraphQL gateway
2. **Authentication Layer**: Implement shared authentication
3. **Monitoring Stack**: Add Prometheus + Grafana
4. **API Documentation**: Generate OpenAPI/GraphQL schemas
5. **End-to-End Testing**: Integration test suites

---

This overview provides a complete reference for managing all three CardioFit backend services. Each service has its detailed startup guide with comprehensive configuration options, troubleshooting, and production deployment guidance.