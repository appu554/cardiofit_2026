# KB-6 Formulary & Stock Implementation Workflow

**Project**: Clinical Synthesis Hub - CardioFit  
**Service**: KB-6 Formulary & Stock Knowledge Base  
**Generated**: 2025-09-03  
**Framework Version**: 2.0.0  

## Executive Summary

This workflow transforms the KB-6 service from its current "infrastructure-ready" state into a production-grade formulary and stock management system that serves as the single source of truth for medication coverage, cost optimization, and inventory intelligence within the medication service ecosystem.

## Current State Assessment

### ✅ Foundation Components (Completed)
- **Database Schema**: Comprehensive PostgreSQL schema with indexes, functions, and triggers
- **Data Models**: Complete Go structs covering all business entities
- **Infrastructure**: Multi-environment configuration with PostgreSQL, Redis, Elasticsearch
- **Framework Integration**: OpenAPI specs, framework.yaml, compliance checklist

### 🔴 Critical Implementation Gaps (90% of functionality missing)
- **API Handlers**: No business logic endpoints implemented
- **gRPC Service**: Missing ScoringEngine integration contract
- **Caching Strategy**: Redis connected but no cache management
- **Search Engine**: Elasticsearch configured but no indexing/query logic
- **ETL Integration**: No data ingestion pipeline
- **Performance Optimization**: No SLA compliance implementation

## Implementation Workflow Structure

### Phase 1: Core Service Foundation (Weeks 1-3)
**Objective**: Establish functional API layer and basic caching for ScoringEngine integration

#### 1.1 gRPC Service Implementation
**Priority**: CRITICAL - Blocks ScoringEngine integration

**Tasks**:
- [ ] Implement `kb6.proto` protocol buffer definitions
- [ ] Create gRPC server implementation with `GetFormularyStatus` and `GetStock` handlers
- [ ] Add transaction ID tracking for request correlation
- [ ] Implement dataset version response fields for audit trail
- [ ] Create mock data service for development testing

**Deliverables**:
- `proto/kb6.proto` - gRPC service definition
- `internal/grpc/server.go` - gRPC server implementation
- `internal/services/formulary_service.go` - Business logic layer
- `internal/services/inventory_service.go` - Stock management logic

**Validation Criteria**:
- [ ] gRPC health checks pass
- [ ] ScoringEngine can successfully call formulary status endpoint
- [ ] Response includes proper evidence envelope (dataset_version, provenance)
- [ ] Performance baseline: p95 < 150ms (SQL fallback acceptable)

#### 1.2 REST API Core Endpoints
**Priority**: HIGH - Enables UI and external integrations

**Tasks**:
- [ ] Implement Gin router with middleware stack
- [ ] Add `/api/v1/formulary/status` endpoint with coverage checking
- [ ] Add `/api/v1/inventory/stock` endpoint with real-time availability
- [ ] Implement request validation and error handling
- [ ] Add Prometheus metrics collection

**Deliverables**:
- `internal/handlers/formulary_handlers.go` - REST endpoint handlers
- `internal/handlers/inventory_handlers.go` - Stock endpoint handlers
- `internal/middleware/` - Authentication, logging, metrics middleware
- `internal/utils/response.go` - Standardized response formatting

**Validation Criteria**:
- [ ] All endpoints return proper JSON responses
- [ ] Error handling follows RFC7807 standard
- [ ] Metrics are collected and exposed at `/metrics`
- [ ] Request/response logging includes transaction IDs

#### 1.3 Redis Caching Layer
**Priority**: HIGH - Required for SLA compliance

**Tasks**:
- [ ] Implement cache key generation with dataset versioning
- [ ] Add L2 Redis cache for formulary "hot" data (15min TTL)
- [ ] Add L2 Redis cache for stock data (30s TTL for high-frequency items)
- [ ] Implement cache miss fallback to PostgreSQL
- [ ] Add cache hit rate monitoring

**Deliverables**:
- `internal/cache/redis_manager.go` - Redis cache abstraction
- `internal/cache/keys.go` - Deterministic key generation
- `internal/services/cached_formulary_service.go` - Cache-aware service layer
- Cache invalidation strategy documentation

**Validation Criteria**:
- [ ] Cache hit rate > 80% for formulary data
- [ ] Cache hit rate > 70% for stock data
- [ ] Performance target: p95 < 40ms for cache hits
- [ ] Proper cache invalidation on dataset updates

### Phase 2: Search and Analytics Engine (Weeks 4-5)
**Objective**: Enable complex formulary search and cost optimization capabilities

#### 2.1 Elasticsearch Integration
**Priority**: HIGH - Enables therapeutic alternatives discovery

**Tasks**:
- [ ] Create Elasticsearch index mappings for formulary data
- [ ] Implement document indexing pipeline from PostgreSQL
- [ ] Add complex search queries with filters (tier, coverage, restrictions)
- [ ] Implement therapeutic alternatives search algorithms
- [ ] Add search result ranking by relevance and cost

**Deliverables**:
- `internal/search/elasticsearch_client.go` - ES client wrapper
- `internal/search/formulary_indexer.go` - Document indexing logic
- `internal/search/query_builder.go` - Search query construction
- `internal/handlers/search_handlers.go` - Search API endpoints

**Validation Criteria**:
- [ ] Index contains all active formulary entries
- [ ] Search response time: p95 < 200ms
- [ ] Therapeutic alternatives accurately grouped by class
- [ ] Search relevance ranking produces clinically appropriate results

#### 2.2 Cost Analysis Engine
**Priority**: MEDIUM - Enables cost optimization recommendations

**Tasks**:
- [ ] Implement cost calculation algorithms using pricing data
- [ ] Add alternative drug cost comparison logic
- [ ] Create cost savings recommendation engine
- [ ] Add bulk cost analysis for multiple drugs
- [ ] Implement cost trend analysis for predictive insights

**Deliverables**:
- `internal/services/cost_analysis_service.go` - Cost calculation logic
- `internal/algorithms/savings_calculator.go` - Optimization algorithms
- `internal/handlers/cost_handlers.go` - Cost analysis endpoints
- Cost optimization recommendation documentation

**Validation Criteria**:
- [ ] Cost calculations match expected formulary pricing
- [ ] Alternative recommendations include safety and efficacy ratings
- [ ] Bulk analysis handles up to 100 drugs efficiently
- [ ] Savings calculations are accurate within 1% margin

### Phase 3: Data Pipeline and ETL Integration (Weeks 6-9)
**Objective**: Enable real-world data ingestion and dataset management

#### 3.1 ETL Framework Integration
**Priority**: CRITICAL - Enables production data flows

**Tasks**:
- [ ] Integrate with Unified ETL pipeline for PBM formulary feeds
- [ ] Add KB-7 terminology normalization integration
- [ ] Implement data validation and quarantine workflows
- [ ] Add support for multiple PBM feed formats
- [ ] Create data quality monitoring and alerting

**Deliverables**:
- `internal/etl/formulary_processor.go` - PBM data processing
- `internal/etl/kb7_integration.go` - Terminology normalization
- `internal/etl/validation_engine.go` - Data validation rules
- `internal/etl/quarantine_manager.go` - Failed processing handling

**Validation Criteria**:
- [ ] Successfully processes sample PBM feeds
- [ ] KB-7 normalization maintains >95% code mapping success
- [ ] Data validation catches common feed errors
- [ ] Quarantine workflow prevents bad data from reaching production

#### 3.2 Dataset Versioning and Promotion
**Priority**: HIGH - Enables blue/green deployment strategy

**Tasks**:
- [ ] Implement dataset versioning with semantic versioning scheme
- [ ] Add blue/green dataset promotion workflows
- [ ] Create governance approval integration for dataset changes
- [ ] Implement atomic dataset pointer switching
- [ ] Add rollback capabilities for failed promotions

**Deliverables**:
- `internal/governance/dataset_manager.go` - Dataset lifecycle management
- `internal/governance/approval_workflow.go` - Governance integration
- `internal/governance/promotion_engine.go` - Blue/green promotion logic
- Dataset promotion runbook documentation

**Validation Criteria**:
- [ ] Dataset promotion completes without service downtime
- [ ] Rollback restores previous dataset within 5 minutes
- [ ] All dataset changes include proper audit trail
- [ ] Governance approval gates function correctly

#### 3.3 Real-time Inventory Integration
**Priority**: MEDIUM - Enables live stock data

**Tasks**:
- [ ] Integrate with inventory management system streams
- [ ] Implement real-time stock level updates via event processing
- [ ] Add stock alert generation and escalation workflows
- [ ] Create inventory reconciliation processes
- [ ] Add predictive stock forecasting model integration

**Deliverables**:
- `internal/inventory/stream_processor.go` - Real-time event processing
- `internal/inventory/alert_manager.go` - Stock alert system
- `internal/inventory/reconciliation.go` - Data consistency checking
- `internal/inventory/forecasting.go` - Demand prediction integration

**Validation Criteria**:
- [ ] Stock levels update within 30 seconds of inventory changes
- [ ] Alert generation triggers for stockout conditions
- [ ] Reconciliation processes maintain >99% data accuracy
- [ ] Forecasting models provide reasonable demand predictions

### Phase 4: Integration Ecosystem and Advanced Features (Weeks 10-12)
**Objective**: Complete ecosystem integration and advanced intelligence features

#### 4.1 Evidence Envelope and Audit Trail
**Priority**: HIGH - Required for clinical compliance

**Tasks**:
- [ ] Implement evidence envelope generation for all responses
- [ ] Add comprehensive audit trail logging for dataset changes
- [ ] Create decision reproducibility tracking
- [ ] Implement compliance reporting capabilities
- [ ] Add data lineage tracking through the entire pipeline

**Deliverables**:
- `internal/audit/evidence_envelope.go` - Evidence tracking
- `internal/audit/decision_logger.go` - Decision audit trail
- `internal/audit/compliance_reporter.go` - Regulatory reporting
- Audit trail schema and retention policies

**Validation Criteria**:
- [ ] All API responses include proper evidence envelopes
- [ ] Decision reproducibility achieves 100% accuracy
- [ ] Audit trail captures all data modifications
- [ ] Compliance reports meet regulatory requirements

#### 4.2 Safety Gateway Integration
**Priority**: MEDIUM - Enables safety event publishing

**Tasks**:
- [ ] Integrate with Safety Gateway event publishing system
- [ ] Add formulary safety alert generation
- [ ] Implement drug recall notification workflows
- [ ] Create safety profile monitoring for alternatives
- [ ] Add contraindication checking integration

**Deliverables**:
- `internal/safety/gateway_client.go` - Safety Gateway integration
- `internal/safety/alert_publisher.go` - Safety event publishing
- `internal/safety/recall_processor.go` - Drug recall handling
- Safety integration API contracts

**Validation Criteria**:
- [ ] Safety events publish successfully to gateway
- [ ] Drug recalls trigger appropriate formulary updates
- [ ] Alternative recommendations consider safety profiles
- [ ] Contraindication checks integrate with formulary status

#### 4.3 Flow2 Orchestrator Integration
**Priority**: MEDIUM - Enables workflow orchestration

**Tasks**:
- [ ] Implement Flow2 Orchestrator communication protocols
- [ ] Add workflow-based formulary decision making
- [ ] Create complex authorization workflow integration
- [ ] Implement step therapy workflow support
- [ ] Add prior authorization status tracking

**Deliverables**:
- `internal/orchestration/flow2_client.go` - Flow2 integration
- `internal/orchestration/workflow_manager.go` - Workflow coordination
- `internal/orchestration/auth_tracker.go` - Authorization tracking
- Orchestration workflow documentation

**Validation Criteria**:
- [ ] Workflow integration functions correctly with Flow2
- [ ] Step therapy workflows execute proper sequence
- [ ] Prior authorization tracking maintains accurate status
- [ ] Complex decision workflows complete successfully

## Implementation Dependencies and Critical Path

### External Dependencies
1. **KB-7 Terminology Service** - Must be operational for code normalization
2. **Unified ETL Pipeline** - Required for data ingestion capabilities
3. **Safety Gateway** - Needed for safety event integration
4. **Flow2 Orchestrator** - Required for workflow integration

### Internal Dependencies
1. **Phase 1 → Phase 2**: Core APIs must be stable before adding search complexity
2. **Phase 2 → Phase 3**: Search functionality should be tested before data pipeline integration
3. **Phase 3 → Phase 4**: Data management must be solid before ecosystem integration

### Critical Path Analysis
- **Week 1-2**: gRPC service implementation (blocks ScoringEngine)
- **Week 3**: Redis caching (enables SLA compliance)
- **Week 6-7**: ETL integration (enables production data)
- **Week 8**: Dataset promotion (enables blue/green deployment)

## Validation Gates and Quality Assurance

### Phase 1 Validation Gate
- [ ] gRPC service passes ScoringEngine integration tests
- [ ] REST APIs meet OpenAPI specification compliance
- [ ] Cache hit rates achieve minimum thresholds
- [ ] Performance baselines established and documented

### Phase 2 Validation Gate
- [ ] Search functionality returns clinically relevant results
- [ ] Cost analysis calculations verified against manual calculations
- [ ] Elasticsearch index performance meets SLA requirements
- [ ] Alternative recommendations validated by clinical team

### Phase 3 Validation Gate
- [ ] ETL pipeline processes sample data successfully
- [ ] Dataset promotion completes without service disruption
- [ ] Data quality metrics meet minimum standards
- [ ] Rollback procedures tested and validated

### Phase 4 Validation Gate
- [ ] Evidence envelopes include all required audit information
- [ ] Integration tests pass with all dependent services
- [ ] Performance under load meets production requirements
- [ ] Security and compliance requirements fully satisfied

## Risk Management and Mitigation Strategies

### High-Risk Areas
1. **Performance SLA Compliance**: Cache implementation critical for meeting p95 < 25ms target
2. **Data Quality**: ETL pipeline must handle malformed PBM feeds gracefully
3. **Integration Complexity**: Multiple service dependencies increase failure risk
4. **Clinical Accuracy**: Cost and alternative calculations must be clinically appropriate

### Mitigation Strategies
1. **Performance**: Implement comprehensive caching strategy early in Phase 1
2. **Data Quality**: Build robust validation and quarantine systems in Phase 3
3. **Integration**: Use circuit breakers and graceful degradation patterns
4. **Clinical Accuracy**: Implement extensive validation and clinical review processes

## Success Metrics and KPIs

### Technical Metrics
- API Response Time: p95 < 25ms for cached requests
- Cache Hit Rate: >95% for formulary data, >85% for stock data
- Uptime: >99.9% service availability
- Data Quality: <1% ETL processing errors

### Business Metrics
- Cost Savings Identified: Track total savings from alternative recommendations
- Formulary Coverage: Monitor coverage percentage across major payers
- Stock Optimization: Measure stockout prevention and inventory efficiency
- Clinical Adoption: Track usage by ScoringEngine and clinical workflows

### Compliance Metrics
- Audit Trail Completeness: 100% of decisions include evidence envelope
- Data Lineage: Complete traceability from source to decision
- Regulatory Reporting: All required reports generated within SLA
- Security Compliance: Zero security violations or data breaches

## Resource Requirements and Timeline

### Development Resources
- **Lead Developer**: Full-time Go developer with healthcare domain knowledge
- **DevOps Engineer**: Part-time for infrastructure and deployment automation
- **Clinical Consultant**: Part-time for validation and requirements clarification
- **QA Engineer**: Part-time for testing and validation activities

### Infrastructure Requirements
- **PostgreSQL Cluster**: High-availability setup with read replicas
- **Redis Cluster**: Multi-node setup for high availability and performance
- **Elasticsearch Cluster**: Multi-node setup for search and analytics
- **Monitoring Stack**: Prometheus, Grafana, and alerting infrastructure

### Timeline Summary
- **Weeks 1-3**: Core Service Foundation (gRPC, REST, Caching)
- **Weeks 4-5**: Search and Analytics Engine
- **Weeks 6-9**: Data Pipeline and ETL Integration
- **Weeks 10-12**: Integration Ecosystem and Advanced Features
- **Total Duration**: 12 weeks (3 months)

## Implementation Insights and Best Practices

`★ Insight ─────────────────────────────────────`
**Architecture Decision**: The workflow prioritizes gRPC implementation in Phase 1 because the ScoringEngine integration is the critical path for the entire medication service ecosystem. Without this integration, KB-6 provides no value to the clinical decision-making process.

**Performance Strategy**: Multi-layer caching is essential not just for SLA compliance but for cost optimization. Formulary data is relatively static but queried frequently, making it ideal for Redis caching with longer TTLs.

**Data Pipeline Design**: The blue/green dataset promotion strategy allows for safe data updates without service disruption, which is critical in healthcare environments where downtime can impact patient care.
`─────────────────────────────────────────────────`

This comprehensive workflow transforms KB-6 from a database-centric service into a fully operational formulary and stock intelligence platform that serves as the backbone for medication cost optimization and clinical decision support within the CardioFit ecosystem.