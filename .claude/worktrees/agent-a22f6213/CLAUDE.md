# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

The Clinical Synthesis Hub "CardioFit" is a comprehensive FHIR-compliant healthcare platform built as a microservices architecture. The system combines multiple technologies including Node.js (Apollo Federation), Python (FastAPI microservices), Go (Safety Gateway), Rust (Knowledge Base services), and Java (Stream Processing) to provide clinical decision support and patient management capabilities.

## Architecture

```
Frontend (Angular) → API Gateway → Apollo Federation → Microservices → FHIR Stores/Databases
Stream Services (Java/Python) → Kafka → Multi-Sink Storage
Knowledge Base Services (Rust/Go) ↔ Safety Gateway (Go) ↔ Clinical Reasoning (Python/Neo4j)
```

Key components:
- **Apollo Federation Server** (`apollo-federation/`): GraphQL gateway that combines multiple service schemas
- **Python Microservices** (`backend/services/`): FHIR-compliant services using FastAPI, MongoDB, and Google Healthcare API
- **Knowledge Base Services** (`backend/services/knowledge-base-services/`): Rust/Go-based clinical knowledge services with TOML rule engine
- **Safety Gateway** (`backend/services/safety-gateway-platform/`): Go-based clinical safety platform with gRPC communication
- **Clinical Reasoning Service**: Python-based AI/ML service with Neo4j graph database integration
- **Stream Processing Services** (`backend/stream-services/`): Java Stage 1 (validation) + Python Stage 2 (FHIR transformation)

## Common Development Commands

### Apollo Federation (Node.js)
```bash
cd apollo-federation
npm install                    # Install dependencies
npm start                      # Start federation server
npm run dev                    # Start with nodemon
npm run simple                 # Start simple gateway
npm run generate-supergraph    # Generate supergraph schema
```

### Consolidated Medication Service Platform
The medication service combines Python APIs, Go orchestration, Rust engines, and knowledge bases:
```bash
cd backend/services/medication-service
make help                    # Show all available commands
make run-all                 # Start all services (Python + Go + Rust + KB)
make health-all              # Check health of all services
make test-all                # Run all tests
make stop-all                # Stop all services
```

Service components:
- **Python Medication Service** (port 8004): FHIR medication resources
- **Flow2 Go Engine** (port 8080): Clinical orchestration and intelligence
- **Rust Clinical Engine** (port 8090): High-performance rule evaluation
- **KB-Drug-Rules** (port 8081): Drug calculation and dosing rules
- **KB-Guideline-Evidence** (port 8084): Clinical guidelines and evidence

### Other Python Microservices
Remaining individual Python services:
```bash
cd backend/services/{service-name}
pip install -r requirements.txt    # Install dependencies
python run_service.py              # Start service (adds shared modules to path)
pytest                             # Run tests
```

Services:
- `patient-service` (port 8003)
- `observation-service` (port 8010) 
- `auth-service` (port 8001)
- `fhir-service` (port 8014)

### Safety Gateway (Go)
```bash
cd backend/services/safety-gateway-platform
go mod download              # Install dependencies
go build                     # Build binary
go test ./...               # Run tests
```

### Stream Processing Services (Java/Python)
```bash
cd backend/stream-services
python setup-kafka-topics.py   # Create Kafka topics
python run-stage1.py           # Start Java validation service (port 8041)
python run-stage2.py           # Start Python FHIR transformation (port 8042)
python run-tests.py            # Run end-to-end tests
```

### Clinical Reasoning Service (Python/Neo4j)
```bash
cd backend/services/clinical-reasoning-service
pip install -r requirements.txt
python start_cae_neo4j.py      # Start with Neo4j integration
python test_cae_comprehensive.py  # Run comprehensive tests
```

## Development Workflow

### Starting the Full System
1. **Consolidated Medication Service**: `make run-all` (in medication-service/)
2. **Stream Processing**: Start Kafka topics, then Stage 1 & Stage 2 services (in stream-services/)
3. **Clinical Reasoning**: `python start_cae_neo4j.py` (in clinical-reasoning-service/)
4. **Safety Gateway**: `go build && ./main.exe` (in safety-gateway-platform/)
5. **Other Python Services**: Use individual `run_service.py` scripts for remaining services
6. **Apollo Federation**: `npm start` (in apollo-federation/)

### Testing Services
- **Consolidated Medication Service**: `make test-all` in medication-service/
- **Other Python Services**: Use `pytest` in individual service directories
- **Stream Services**: `python run-tests.py` in stream-services/
- **Clinical Reasoning**: Use specific test files like `test_cae_comprehensive.py`
- **Apollo Federation**: Test GraphQL endpoints at `http://localhost:4000/graphql`

## Key Technical Details

### Shared Python Module System
Python services use a shared module system located in `backend/shared/`. The `run_service.py` scripts automatically configure the Python path to include shared modules for authentication, FHIR models, and common utilities.

### Authentication Flow
```
Client → API Gateway → Auth Service → Individual Services → Google Healthcare API/Databases
```

Services use JWT tokens and HeaderAuthMiddleware from the shared module.

### Database Integrations
- **MongoDB**: Patient, observation, and clinical data (Atlas cluster)
- **PostgreSQL**: Knowledge base services (isolated per service on ports 5432/5433)
- **Neo4j**: Clinical knowledge graph for reasoning service
- **Redis**: Caching and session management (ports 6379/6380)
- **Supabase**: Alternative database option for some services
- **Elasticsearch**: Clinical data search and analytics
- **Kafka**: Stream processing event backbone (Confluent Cloud)

### FHIR Compliance
All clinical services implement FHIR R4 resources with proper validation and data models located in shared modules. Integration with Google Healthcare API for FHIR store operations.

### Service Communication Patterns
- **gRPC**: Safety Gateway ↔ Clinical Reasoning Service
- **GraphQL Federation**: Apollo Federation gateway with schema composition
- **REST APIs**: Standard FastAPI patterns for individual services
- **Event Streaming**: Kafka topics for device data processing pipeline
- **TOML Rule Engine**: Knowledge Base services use TOML-based clinical rules with validation

### Docker Support
Most services include Dockerfile and docker-compose configurations. The consolidated medication-service provides comprehensive Docker setup with infrastructure services (PostgreSQL, Redis, Kafka, monitoring) for its knowledge base components.

## Service Ports
- Apollo Federation: 4000
- Auth Service: 8001
- Patient Service: 8003
- **Consolidated Medication Service**:
  - Python Medication API: 8004
  - Flow2 Go Engine: 8080
  - Rust Clinical Engine: 8090
  - KB-Drug-Rules: 8081
  - KB-Guideline-Evidence: 8084
- Observation Service: 8010
- FHIR Service: 8014
- Stream Services: 8041 (Stage 1), 8042 (Stage 2)
- PostgreSQL: 5432 (local), 5433 (Docker)
- Redis: 6379 (local), 6380 (Docker)  
- Grafana: 3000
- Prometheus: 9090

## Important Notes

- Always use the provided `run_service.py` scripts for Python services rather than direct uvicorn commands
- Check service health endpoints (`/health`) before running integration tests
- The system is designed for HIPAA compliance with proper authentication and audit logging
- Use the consolidated Makefile in medication-service for the most reliable development experience with medication, knowledge base, Flow2, and Rust engine components
- Stream services require Java 17+ for Stage 1 and Python 3.11+ for Stage 2  
- Medication service knowledge bases use isolated database ports (5433, 6380) to avoid conflicts
- Clinical Reasoning Service requires Neo4j for graph database operations