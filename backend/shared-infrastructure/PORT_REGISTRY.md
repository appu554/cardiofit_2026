# Canonical Port Registry

Single source of truth for all service port assignments. Referenced by docker-compose files, config.go defaults, and CLAUDE.md documentation.

## Knowledge Base Services

| Service | Port | Module Path | Status |
|---------|------|-------------|--------|
| KB-1 Drug Rules | 8081 | `kb-1-drug-rules/` | Active |
| KB-2 Clinical Context | 8086 | `kb-2-clinical-context/` | Active |
| KB-3 Guidelines | 8087 | `kb-3-guidelines/` | Active |
| KB-4 Patient Safety | 8088 | `kb-4-patient-safety/` | Active |
| KB-5 Drug Interactions | 8089 | `kb-5-drug-interactions/` | Active |
| KB-6 Formulary | 8091 | `kb-6-formulary/` | Active |
| KB-7 Terminology | 8092 | `kb-7-terminology/` | Active |
| KB-8 Calculator | 8093 | `kb-8-calculator-service/` | Active |
| KB-9 Care Gaps | 8094 | `kb-9-care-gaps/` | Active |
| KB-10 Rules Engine | 8095 | `kb-10-rules-engine/` | Active |
| KB-11 Population Health | 8096 | `kb-11-population-health/` | Active |
| KB-12 Ordersets/CarePlans | 8097 | `kb-12-ordersets-careplans/` | Active |
| KB-13 Quality Measures | 8098 | `kb-13-quality-measures/` | Active |
| KB-14 Care Navigator | 8099 | `kb-14-care-navigator/` | Active |
| KB-16 Lab Interpretation | 8100 | `kb-16-lab-interpretation/` | Active |
| KB-17 Population Registry | 8101 | `kb-17-population-registry/` | Active |
| KB-18 Governance Engine | 8102 | `kb-18-governance-engine/` | Active |

## Vaidshala Clinical Runtime Services

| Service | Port | Module Path | Status |
|---------|------|-------------|--------|
| KB-19 Protocol Orchestrator | 8103 | `kb-19-protocol-orchestrator/` | Active |
| KB-20 Patient Profile | 8131 | `kb-20-patient-profile/` | Active |
| KB-21 Behavioral Intelligence | 8133 | `kb-21-behavioral-intelligence/` | Active |
| KB-22 HPI Engine | 8132 | `kb-22-hpi-engine/` | Active |
| KB-23 Decision Cards | 8134 | `kb-23-decision-cards/` | Active |
| V-MCU Engine | 8140 | `vaidshala/clinical-runtime-platform/engines/vmcu/` | Planned |

## Platform Services

| Service | Port | Location | Status |
|---------|------|----------|--------|
| Auth Service | 8001 | `backend/services/auth-service/` | Active |
| Patient Service | 8003 | `backend/services/patient-service/` | Active |
| Medication Service | 8004 | `backend/services/medication-service/` | Active |
| Observation Service | 8010 | `backend/services/observation-service/` | Active |
| FHIR Service | 8014 | `backend/services/fhir-service/` | Active |
| Apollo Federation | 4000 | `apollo-federation/` | Active |

## Infrastructure Services

| Service | Port (Docker) | Port (Local) |
|---------|--------------|--------------|
| PostgreSQL | 5433 | 5432 |
| Redis | 6380 | 6379 |
| Adminer (DB UI) | 8082 | - |
| Grafana | 3000 | - |
| Prometheus | 9090 | - |

## Per-Service Database Ports (Docker)

| Service | PostgreSQL Port | Redis Port |
|---------|----------------|------------|
| KB-21 | 5434 | 6381 |
| KB-22 | 5437 | 6386 |
| KB-23 | 5438 | 6387 |

## Rules

1. **No port reuse**: Each service gets exactly one port. Collisions cause silent failures.
2. **Config.go defaults must match**: Every service's `config.go` default port must match this registry.
3. **Docker-compose must match**: Every `docker-compose.yml` port mapping must match this registry.
4. **New services**: Allocate from the next available port in the appropriate range.
