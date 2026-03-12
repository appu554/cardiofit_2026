# Clinical Runtime Platform

> **The Deterministic Engine** - Executes clinical logic without modification.

## Purpose

This platform provides the runtime infrastructure for executing clinical knowledge:
- FHIR Server for data storage
- CQL Executor for logic evaluation
- Terminology Cache for fast value set lookups
- Audit Service for evidence envelopes
- Regional deployments (India, Australia)

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    RUNTIME PLATFORM                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐    │
│  │ FHIR Server  │   │ CQL Executor │   │ Term Cache   │    │
│  │ (HAPI FHIR)  │   │ (cqf-ruler)  │   │  (Redis)     │    │
│  └──────────────┘   └──────────────┘   └──────────────┘    │
│                                                              │
│  ┌──────────────┐   ┌──────────────┐                        │
│  │ Audit Svc    │   │ Evidence Env │                        │
│  │              │   │              │                        │
│  └──────────────┘   └──────────────┘                        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Services

### FHIR Server (`services/fhir-server/`)
- HAPI FHIR R4 server
- Patient data storage
- FHIR API endpoints

### CQL Executor (`services/cql-executor/`)
- cqf-ruler based execution
- Pulls ELM from knowledge-core
- Deterministic evaluation

### Terminology Cache (`services/terminology-cache/`)
- Redis-backed value set storage
- Pre-expanded value sets
- Fast membership checks

### Audit Service (`services/audit-service/`)
- Clinical decision logging
- Compliance tracking
- Immutable audit trail

### Evidence Envelope (`services/evidence-envelope/`)
- Links inputs → logic → outputs
- Cryptographic verification
- Regulatory compliance

## Deployment

### Kubernetes
```bash
# Base deployment
kubectl apply -k k8s/base/

# Regional overlay (India)
kubectl apply -k k8s/overlays/IN/

# Regional overlay (Australia)
kubectl apply -k k8s/overlays/AU/
```

### Terraform
```bash
# Control plane (global)
cd terraform/control-plane && terraform apply

# Data plane (per region)
cd terraform/data-plane && terraform apply -var="region=IN"
```

## Scripts

- `scripts/pull-and-verify.sh` - Pull signed artifacts from knowledge-core
- `scripts/load-valuesets.sh` - Load value sets into Redis cache
- `scripts/health-check.sh` - Verify all services are healthy

## Observability

- **Prometheus**: Metrics collection
- **Grafana**: Dashboard visualization
- **OpenTelemetry**: Distributed tracing

## Key Principles

1. **Never modify clinical logic** - Only execute what's published
2. **Verify signatures** - Reject unsigned artifacts
3. **Log everything** - Every decision generates evidence
4. **Regional isolation** - PHI stays in region

## License

Proprietary - CardioFit Clinical Synthesis Hub
