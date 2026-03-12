# Clinical Knowledge Graph: The Phased, Hybrid Federation Strategy

**Document Version:** 2.0 (Final)
**Date:** 2025-07-17

## Executive Summary: The Hybrid Federation Strategy

This document outlines the final, unified architecture and implementation plan for our Clinical Knowledge Graph. The chosen strategy is a **Phased, Hybrid Federation Architecture**, built on the core principle: **Build the platform, federate the knowledge.**

We will begin by building a powerful foundational graph using the best available free sources to achieve approximately 80% of the required functionality at minimal cost. We will then strategically layer in premium commercial data sources to fill critical gaps in drug safety and pathway content, reduce legal risk, and ensure the real-time accuracy required for a production clinical system.

This unified graph will serve as the single source of truth for all clinical intelligence engines, starting with the CAE and Protocol Engines, ensuring consistency, efficiency, and a foundation for unparalleled cross-domain reasoning.

## 1. Final Architecture: The Unified Knowledge Federation Model

Instead of building a monolithic, self-contained knowledge base, we will create a **Knowledge Federation Layer**. This layer acts as an intelligent orchestrator that can query multiple, tiered sources and synthesize their findings into a single, coherent answer.

```
┌────────────────────────────────────────────────────────────────┐
│                    KNOWLEDGE FEDERATION LAYER                  │
├────────────────────────────────────────────────────────────────┤
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              Intelligent Source Orchestrator             │  │
│  │  • Source ranking by evidence & reliability              │  │
│  │  • Conflict resolution (e.g., "local policy overrides global") │
│  │  • Freshness scoring & staleness detection               │  │
│  └──────────────────────────────────────────────────────────┘  │
├────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐   │
│  │   Tier 1    │  │   Tier 2    │  │      Tier 3         │   │
│  │ Foundational│  │  Critical   │  │   Supplemental      │   │
│  │ (Free)      │  │ (Commercial)│  │   (RWE & AI)        │   │
│  └─────────────┘  └─────────────┘  └─────────────────────┘   │
└────────────────────────────────────────────────────────────────┘
```

## 2. The Unified Tiered Source Strategy (For BOTH Engines)

This is the heart of our hybrid approach. We will populate the federation layer using a tiered strategy that explicitly serves both the CAE and Protocol engines from a single, unified graph.

| Tier | Data Type                 | Primary Source         | Engine Focus           |
| :--- | :------------------------ | :-------------------------- | :--------------------- |
| 1    | Drug Terminology          | RxNorm                      | **BOTH (Foundation)**  |
| 1    | Computable Pathways       | AHRQ CDS Connect            | Protocol Engine        |
| 1    | Guideline Content         | NICE Pathways               | Protocol Engine        |
| 1    | Drug Interactions         | DrugBank (Academic)         | CAE Engine             |
| 1    | Specialized Safety        | CredibleMeds                | CAE Engine             |
| 1    | Evidence Base             | PubMed Central              | **BOTH (Trust Layer)**   |
| 2    | Premium Pathways          | BMJ Clinical Intelligence   | Protocol Engine        |
| 2    | Premium Safety            | FDB / Medi-Span             | CAE Engine             |
| 3    | Real-World Evidence       | OpenFDA Adverse Events      | CAE Engine             |
| 3    | Complex Case Reasoning    | Glass Health / AI Engines   | Protocol Engine        |

## 3. The Unified Knowledge Compilation Pipeline

This pipeline is source-agnostic. Its job is to ingest data from any source (free or commercial), harmonize it to our common graph model, and build the optimized graph file that our runtime engines will use.

*   **Ingestion Stage:** Will have dedicated ingesters for each source (e.g., `RxNormIngester`, `AHRQIngester`, `BMJIngester`).
*   **Harmonization Stage:** This stage is critical. It ensures that the drug "Warfarin" from the DrugBank interaction data is mapped to the exact same `(Drug)` node as the "Warfarin" mentioned in a NICE guideline. This is why using RxNorm as the master key is non-negotiable.
*   **Graph Construction Stage:** Builds a single graph containing nodes and relationships for both CAE and Protocol domains.

## 4. The Unified Implementation Roadmap

This phased roadmap provides a clear path from day one to a mature, federated system serving all clinical intelligence needs.

### **Phase 1: Foundation & Dual MVP (Weeks 1-4)**

**Strategic Goal:** To rapidly establish the core infrastructure and prove the architecture's fundamental value by serving two distinct clinical engines from a single, unified graph. This phase prioritizes speed and validation over comprehensive knowledge.

**Key Activities & Technical Deep Dive:**

*   **Infrastructure Setup (Weeks 1-2):**
    *   **✅ GraphDB Instance Already Running:** Local GraphDB instance at `http://localhost:7200` with repository `cae-clinical-intelligence` is already operational and integrated with CAE service.
    *   **Establish Git Repository:** Create a new repository for the `knowledge-pipeline-service`.
    *   **Enhance Core Graph Schema:** Extend the existing GraphDB schema with additional clinical entities using SPARQL/RDF ontology definitions.
*   **Core Pipeline Construction (Weeks 1-2):**
    *   Implement the basic framework for the ingestion and harmonization service using GraphDB SPARQL endpoints.
    *   Create the first ingester: `RxNormIngester`. This script will parse the RxNorm RRF files to extract drug names, RXCUIs, and their relationships, converting to RDF/Turtle format for GraphDB.
    *   Create a second simple ingester: `CredibleMedsIngester`. This will parse the manually downloaded TSV file from the CredibleMeds website and insert as RDF triples into the existing GraphDB repository.
*   **CAE MVP - QT Prolongation (Weeks 3-4):**
    *   **Knowledge Loading:** The `CredibleMedsIngester` will populate RDF triples like `cae:drug_123 cae:hasQTRisk cae:QTProlongationRisk` with severity properties in the existing GraphDB repository.
    *   **Engine Logic:** The CAE will leverage its existing GraphDB client to query for QT risk using SPARQL queries against the `cae-clinical-intelligence` repository.
    *   **API Exposure:** The existing CAE gRPC service at `localhost:8027` will be enhanced to include QT risk checking, accessible through the Safety Gateway Platform.
*   **Protocol MVP - Sepsis Bundle (Weeks 3-4):**
    *   **Knowledge Loading:** A simple JSON or YAML file representing the Sepsis "Hour-1 Bundle" will be created manually. An ingester will parse this file to create RDF triples like `cae:SepsisHour1Bundle cae:hasStep cae:AdministerAntibiotics` in the GraphDB repository.
    *   **Engine Logic:** The Protocol Engine will be integrated with the existing GraphDB client to query pathway steps using SPARQL queries against the same `cae-clinical-intelligence` repository.
    *   **API Exposure:** The Safety Gateway Platform will expose protocol endpoints that leverage the unified GraphDB knowledge base through the existing CAE service integration.

**Primary Outcome:** A functional, end-to-end platform demonstrating that the existing GraphDB instance at `localhost:7200` with repository `cae-clinical-intelligence`, enhanced with additional clinical knowledge, can successfully answer both drug-property queries (CAE) and pathway-structure queries (Protocol Engine) through a unified RDF knowledge base.

---

### **Phase 2: Scaling with Free Sources (Weeks 5-12)**

**Strategic Goal:** To significantly broaden the graph's knowledge base for both domains using the best-in-class free and open-source datasets, moving from MVP to a genuinely useful tool.

**Key Activities & Technical Deep Dive:**

*   **Protocol Engine Enhancement (Weeks 5-8):**
    *   Develop ingesters for **AHRQ CDS Connect** and **NICE Pathways**. This involves parsing structured formats (XML, JSON) to extract computable pathway logic, decision nodes, and steps, converting them to RDF triples for the GraphDB repository.
    *   Expand the existing GraphDB schema to include `cae:Guideline`, `cae:Pathway`, and `cae:Step` classes, with properties like `cae:publishedBy` and `cae:recommends`.
*   **CAE Engine Enhancement (Weeks 7-10):**
    *   Develop a robust ingester for **DrugBank (Academic version)**. This involves parsing the large XML file to create detailed RDF drug entities with pharmacology properties, and establishing `cae:interactsWith` relationships in the GraphDB repository.
    *   The harmonization logic will be critical here to ensure DrugBank drugs are correctly mapped to the master RxNorm drug entities using `cae:hasRxCUI` properties in the unified GraphDB knowledge base.
*   **Trust & Provenance Layer (Weeks 11-12):**
    *   Begin building the evidence layer using RDF provenance patterns. For every relationship (e.g., an interaction), create `cae:Source` and `cae:Evidence` entities pointing to PubMed IDs or citations.
    *   This creates a trust framework using RDF reification: `cae:drug1 cae:interactsWith cae:drug2` with associated `cae:hasEvidence` and `cae:hasSource` properties in the GraphDB repository.

**Primary Outcome:** A unified GraphDB knowledge base with foundational coverage for the top 20-30 most common clinical pathways and several thousand critical drug-drug and drug-disease interactions, with every assertion traceable to its source through RDF provenance patterns.

---
### Phase 3: Strategic Commercial Integration (Months 4-6)

**Goal:** Fill critical gaps, reduce risk, and ensure timeliness with premium partners.
*   **Actions:**
    *   **Select & Integrate Primary Partners:** Finalize contracts and build the API ingesters for BMJ Clinical Intelligence (for protocols) and FDB/Medi-Span (for drug safety).
    *   **Enhance Harmonization:** Update the conflict resolution logic to prioritize the higher-quality, indemnified commercial data while retaining unique insights from free sources.
*   **Outcome:** A hybrid knowledge graph with comprehensive, evidence-graded, and professionally maintained content.

### **Phase 3: Strategic Commercial Integration (Months 4-6)**

**Strategic Goal:** To elevate the platform to production-grade by filling critical knowledge gaps, ensuring timeliness, and reducing legal risk with indemnified data from premium commercial partners.

**Key Activities & Technical Deep Dive:**

*   **Partner Selection & API Integration (Month 4):**
    *   Finalize contracts with **BMJ Clinical Intelligence** (for protocols) and **FDB/Medi-Span** (for drug safety).
    *   Develop secure, resilient API clients to ingest data from their services. This is a shift from file-based ingestion to real-time or near-real-time API calls.
*   **Advanced Harmonization & Conflict Resolution (Month 5):**
    *   Enhance the harmonization engine with a **confidence hierarchy**. Commercial data will be flagged as a higher-confidence source.
    *   Implement conflict resolution logic. For example: "If DrugBank says interaction is 'minor' and FDB says it is 'major', the graph will store both but flag the FDB assertion as the primary one for clinical alerting."
*   **Production Hardening (Month 6):**
    *   Deploy the GraphDB cluster in a high-availability configuration with backup repositories.
    *   Implement comprehensive monitoring and alerting for the entire knowledge pipeline and GraphDB performance.
    *   Conduct performance and load testing to ensure the GraphDB SPARQL queries meet production SLAs.

**Primary Outcome:** A hybrid, enterprise-grade GraphDB knowledge base with comprehensive, evidence-graded, and professionally maintained content, ready for use in a live clinical environment through the existing CAE service integration.

---

### **Phase 4: Intelligent Enhancement (Ongoing)**

**Strategic Goal:** To evolve the CKG from a static knowledge repository into a dynamic, learning system that improves over time with real-world use.

**Key Activities & Technical Deep Dive:**

*   **Real-World Evidence Ingestion (Ongoing):**
    *   Develop an ingester for the **OpenFDA Adverse Event Reporting System (FAERS) API**. This will allow the graph to identify potential new safety signals not yet in official compendia.
    *   Relationships will be marked with lower confidence, e.g., `(Drug)-[:HAS_POSSIBLE_ADVERSE_EVENT]->(Condition)`.
*   **Learning & Feedback Loop (Ongoing):**
    *   Integrate with a **"Learning Gateway"** service. When a clinician overrides an alert, this service will capture the reason.
    *   The CKG pipeline will analyze these overrides. If a specific alert is overridden frequently for the same reason, it will be flagged for clinical review, creating a self-improvement loop.
*   **AI Integration for the Unknown (Pilot):**
    *   For complex cases with no matching protocol in the graph, pilot an integration with an AI engine like **Glass Health**.
    *   The Orchestrator will have a fallback mechanism: "If no protocol found, call Glass Health API with patient summary." The response will be clearly marked as AI-generated.

**Primary Outcome:** A dynamic, self-improving clinical intelligence platform that not only provides known information but also learns from real-world practice and identifies new patterns.

---

## 5. Implementation Blueprint: Technical & Operational Details

This section provides the specific technical, operational, and governance details required to execute the roadmap.

### **1. Technical Implementation Details**

#### **Graph Schema Definition (GraphDB with SPARQL)**
*Core entities and constraints for Phase 1 using existing GraphDB setup.*

**Note**: This implementation uses the existing GraphDB instance at `http://localhost:7200` with repository `cae-clinical-intelligence` that is already integrated with the CAE system.

```sparql
# Core Clinical Ontology Schema (Turtle/RDF)
PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX owl: <http://www.w3.org/2002/07/owl#>

# Core Entity Classes
cae:Drug a owl:Class ;
    rdfs:label "Drug" ;
    rdfs:comment "Pharmaceutical drug with RxNorm identifier" .

cae:Condition a owl:Class ;
    rdfs:label "Medical Condition" ;
    rdfs:comment "Medical condition with SNOMED CT code" .

cae:Pathway a owl:Class ;
    rdfs:label "Clinical Pathway" ;
    rdfs:comment "Clinical care pathway or protocol" .

cae:Patient a owl:Class ;
    rdfs:label "Patient" ;
    rdfs:comment "Patient entity for clinical context" .

# Core Properties
cae:hasRxCUI a owl:DatatypeProperty ;
    rdfs:domain cae:Drug ;
    rdfs:range xsd:string .

cae:hasSnomedCode a owl:DatatypeProperty ;
    rdfs:domain cae:Condition ;
    rdfs:range xsd:string .

cae:interactsWith a owl:ObjectProperty ;
    rdfs:domain cae:Drug ;
    rdfs:range cae:Drug .

cae:hasSeverity a owl:DatatypeProperty ;
    rdfs:range xsd:string .

cae:hasEvidenceLevel a owl:DatatypeProperty ;
    rdfs:range xsd:string .
```

#### **Data Model Versioning Strategy**
*Ensuring schema and knowledge updates are managed and non-disruptive.*
```yaml
version_control:
  graph_schema_version: "Track schema migrations using a tool like Liquibase for Graphs or custom scripts."
  knowledge_version: "Timestamp and version every data load to enable rollback."
  
  migration_strategy:
    - "Blue-green deployments for major schema changes."
    - "API must maintain backward compatibility for at least 1 prior version."
    - "Automated migration scripts will be part of the CI/CD pipeline."
```

### **2. Development Environment Setup**

#### **Week 1 Checklist**
```yaml
development_setup:
  infrastructure:
    - [x] GraphDB instance running at localhost:7200 (ALREADY CONFIGURED)
    - [x] Repository 'cae-clinical-intelligence' created and running (ALREADY CONFIGURED)
    - [x] Python 3.10+ environment with dependencies (ALREADY CONFIGURED)
    - [x] CAE service integration with GraphDB client (ALREADY IMPLEMENTED)
    - [ ] Knowledge pipeline service repository setup
    - [ ] Configuration files for development, staging, and production environments

  existing_setup:
    graphdb:
      endpoint: "http://localhost:7200"
      repository: "cae-clinical-intelligence"
      status: "RUNNING"
      integration: "CAE service already connected"

  initial_dependencies:
    python:
      - SPARQLWrapper # For GraphDB SPARQL queries
      - rdflib # For RDF/Turtle data manipulation
      - pandas  # For data manipulation
      - pydantic  # For data validation
      - fastapi  # For API development
      - pytest  # For testing
      - aiohttp # For async GraphDB client (ALREADY IMPLEMENTED)

  monitoring:
    - [ ] Prometheus metrics exporter configured
    - [ ] Grafana dashboards for key metrics
    - [ ] Initial alert rules for data quality failures
```

### **3. Data Quality Framework**

#### **RDF/SPARQL Validation for GraphDB**
*Code-level checks to enforce data integrity before loading into GraphDB.*
```python
# Example validator class for GraphDB RDF data
class GraphDBDataQualityValidator:
    def __init__(self, graphdb_client):
        self.graphdb_client = graphdb_client

    def validate_drug_rdf(self, drug_rdf):
        """Every drug RDF entity must have these properties to be loaded."""
        assert 'cae:hasRxCUI' in drug_rdf, "Missing RxCUI property"
        assert 'rdfs:label' in drug_rdf, "Missing drug name"
        assert 'cae:hasSource' in drug_rdf, "Missing data source"
        assert 'cae:lastUpdated' in drug_rdf, "Missing timestamp"

    def validate_interaction_rdf(self, interaction_rdf):
        """Every interaction RDF must be complete."""
        severity_values = ['major', 'moderate', 'minor']
        evidence_levels = ['A', 'B', 'C', 'D']

        assert any(sev in interaction_rdf for sev in severity_values), "Missing severity"
        assert any(ev in interaction_rdf for ev in evidence_levels), "Missing evidence level"
        assert 'cae:hasClinicalSignificance' in interaction_rdf, "Missing clinical significance"

    async def validate_sparql_insert(self, sparql_query):
        """Validate SPARQL insert query before execution."""
        # Test query syntax and required ontology compliance
        return await self.graphdb_client.validate_query(sparql_query)
```

### **4. Error Handling & Resilience**

#### **GraphDB Source Failure Handling**
*The orchestrator must degrade gracefully with GraphDB connectivity issues.*
```python
# GraphDB-aware Source Orchestrator
class GraphDBSourceOrchestrator:
    def __init__(self, graphdb_client, cae_service_client):
        self.graphdb_client = graphdb_client
        self.cae_service_client = cae_service_client

    async def query_with_fallback(self, sparql_query, fallback_sources=None):
        """Queries GraphDB, falls back to CAE service cache or other sources on failure."""
        results = []
        failed_sources = []

        try:
            # Primary: Query GraphDB directly
            result = await self.graphdb_client.query(sparql_query)
            if result.success:
                return result
            else:
                failed_sources.append("GraphDB")
        except Exception as e:
            failed_sources.append(f"GraphDB: {str(e)}")

        try:
            # Fallback: Query through CAE service (which has its own GraphDB connection)
            result = await self.cae_service_client.query_knowledge_base(sparql_query)
            if result:
                return result
            else:
                failed_sources.append("CAE Service")
        except Exception as e:
            failed_sources.append(f"CAE Service: {str(e)}")

        # Final fallback: Return cached result with warning
        if failed_sources:
            return self.get_cached_result(sparql_query,
                                        warning=f"Sources unavailable: {failed_sources}")
```

### **5. Testing Strategy**

#### **Phase 1 Test Cases for GraphDB Integration**
*Critical unit and integration tests for the dual MVP using existing GraphDB setup.*
```python
# Test cases for GraphDB-based CAE and Protocol engines
async def test_qt_prolongation_known_drug_graphdb():
    """Test CAE MVP with a known QT-prolonging drug using GraphDB."""
    # Use existing test patient from GraphDB
    patient_id = "905a60cb-8241-418f-b29b-5b020e851392"

    sparql_query = f"""
    PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
    SELECT ?drug ?qtRisk WHERE {{
        cae:{patient_id} cae:prescribedMedication ?drug .
        ?drug cae:hasQTRisk ?qtRisk .
    }}
    """

    result = await graphdb_client.query(sparql_query)
    assert result.success == True
    assert len(result.data['results']['bindings']) > 0

async def test_sepsis_bundle_steps_graphdb():
    """Test Protocol MVP returns the correct steps from GraphDB."""
    sparql_query = """
    PREFIX cae: <http://clinical-assertion-engine.org/ontology/>
    SELECT ?step ?sequence WHERE {
        cae:SepsisHour1Bundle cae:hasStep ?step .
        ?step cae:hasSequence ?sequence .
    } ORDER BY ?sequence
    """

    result = await graphdb_client.query(sparql_query)
    assert result.success == True
    steps = result.data['results']['bindings']
    assert len(steps) >= 4
    step_names = [step['step']['value'] for step in steps]
    assert "measure_lactate" in str(step_names)
    assert "obtain_blood_cultures" in str(step_names)

async def test_unified_cae_service_integration():
    """Test the CAE service integration with GraphDB knowledge base."""
    # Test using existing CAE gRPC service at localhost:8027
    patient_id = "905a60cb-8241-418f-b29b-5b020e851392"
    medications = ["ciprofloxacin", "warfarin"]

    result = await cae_service_client.validate_safety_request(
        patient_id=patient_id,
        medications=medications,
        conditions=["pneumonia", "chronic_kidney_disease"]
    )

    assert "interaction_warning" in result.warnings
    assert "dosing_adjustment_recommendation" in result.recommendations
    assert result.overall_status in ["SAFE", "UNSAFE", "WARNING"]
```

### **6. Performance Benchmarks**

#### **GraphDB Performance Targets**
*Clear, measurable performance goals for the GraphDB-based system.*
```yaml
performance_requirements:
  phase_1_targets:
    graphdb_repository_load_time: "< 2 minutes (existing repository already loaded)"
    single_sparql_drug_query_p99: "< 100ms"
    pathway_sparql_retrieval_p99: "< 150ms"
    cae_service_integration_p99: "< 200ms (including GraphDB query)"

  production_targets:
    concurrent_sparql_queries: "500/second (GraphDB limitation)"
    rdf_triple_scale: ">1M triples, >100K entities"
    sparql_query_p99_latency: "< 200ms"
    cae_service_response_p99: "< 300ms (including GraphDB roundtrip)"

  existing_baseline:
    current_graphdb_status: "RUNNING at localhost:7200"
    current_repository: "cae-clinical-intelligence with test data loaded"
    current_cae_integration: "Functional gRPC service at localhost:8027"
```

### **7. Security Considerations**

#### **GraphDB Security Requirements**
*Security requirements leveraging existing GraphDB setup.*
```yaml
security_requirements:
  api_security:
    - "OAuth2/JWT for all API authentication (already implemented in CAE service)."
    - "Rate limiting and throttling per client for GraphDB queries."
    - "Comprehensive audit logging for all SPARQL queries."

  graphdb_security:
    - "GraphDB repository access control and authentication."
    - "Encryption in transit for GraphDB SPARQL endpoints."
    - "Strictly no PHI stored in the clinical knowledge graph (ontology only)."
    - "Secure GraphDB backup and recovery procedures."

  existing_security:
    - "CAE service already implements authentication at localhost:8027."
    - "GraphDB instance at localhost:7200 currently in development mode."
    - "Production deployment will require GraphDB security hardening."

  dependency_management:
    - "Regular vulnerability scanning of GraphDB and RDF libraries."
    - "Secure storage for all commercial API keys (e.g., AWS Secrets Manager)."
    - "GraphDB version management and security patching."
```

### **8. Deployment Pipeline (CI/CD)**

#### **Automated from the Start**
*A robust pipeline to ensure quality and rapid, safe deployments.*
```yaml
deployment_pipeline:
  on_commit_to_dev:
    - "Trigger unit and integration tests."
    - "Run automated schema and data quality validation checks."
    
  on_merge_to_staging:
    - "Deploy to staging environment."
    - "Run performance regression tests against baseline."
    - "Trigger clinical validation test suite."
    
  on_release_to_production:
    - "Use blue-green or canary deployment strategy."
    - "Automated rollback on critical metric failure."
    - "Tag knowledge version and schema version."
```

### **9. Monitoring & Alerting**

#### **Key Metrics to Track**
*What to monitor to ensure system health, data quality, and clinical value.*
```yaml
monitoring_metrics:
  data_quality:
    - "Count of unmapped entities per source."
    - "Number of data conflicts between sources."
    - "Age of knowledge by domain (e.g., last DrugBank update)."
    
  system_health:
    - "Query latency (p95, p99) by query type."
    - "Source API availability and error rates."
    - "Cache hit/miss ratio."
    
  clinical_impact:
    - "Query volume per pathway or drug."
    - "Top 20 most checked interactions."
    - "Alert acceptance vs. override rates (from Learning Gateway)."
```

### **10. Documentation Requirements**

#### **A Living Resource**
*Documentation structure to be created in Week 1.*
```markdown
docs/
├── architecture/         # High-level design
│   ├── graph_schema.md
│   ├── data_flow.md
│   └── api_design.md
├── runbooks/             # How-to guides for operators
│   ├── add_new_source.md
│   ├── handle_source_failure.md
│   └── update_knowledge.md
└── clinical_governance/  # Clinical rules and processes
    ├── evidence_standards.md
    ├── conflict_resolution_rules.md
    └── validation_process.md
```

### **11. Legal & Compliance**

#### **Commercial Data Checklist**
*To be completed before integrating any premium data source.*
```yaml
legal_checklist:
  - [ ] Data Use Agreements (DUA) signed and stored.
  - [ ] Indemnification clauses reviewed by legal.
  - [ ] Usage limits and costs documented.
  - [ ] Attribution requirements (if any) are clear.
  - [ ] Audit trail implementation meets DUA requirements.
```

### **12. Phase 1 Success Criteria**

#### **GraphDB MVP Success Criteria**
*The definition of 'done' for Phase 1 using existing GraphDB setup.*
```yaml
mvp_success_criteria:
  week_4_gates:
    technical:
      - "Both CAE and Protocol MVP successfully query the same GraphDB repository 'cae-clinical-intelligence'."
      - "The knowledge pipeline successfully ingests and harmonizes at least 2 sources into RDF triples."
      - "The full test suite (unit, integration, clinical) is passing with SPARQL queries."
      - "CAE service at localhost:8027 successfully integrates with enhanced GraphDB knowledge."

    clinical:
      - "The QT risk check provides 100% matching results against the CredibleMeds source list via SPARQL queries."
      - "The Sepsis pathway query returns all core steps from the modeled bundle in RDF format."
      - "Output from both engines is validated as correct by a clinical team member using existing test patient 905a60cb-8241-418f-b29b-5b020e851392."

    performance:
      - "All MVP SPARQL queries have a p99 latency under 200ms."
      - "The GraphDB repository enhancement completes in under 5 minutes."
      - "Zero RDF data conflicts are present in the enhanced GraphDB repository."
      - "CAE service integration maintains sub-300ms response times."

    integration:
      - "Safety Gateway Platform successfully queries CAE service which uses GraphDB knowledge."
      - "GraphDB client connection pooling and error handling is robust."
      - "All existing CAE functionality continues to work with enhanced knowledge base."
```

---

## 6. The Power of Synthesis: A Real-World Example

Imagine a patient with pneumonia and a low eGFR (poor kidney function) who is also on Warfarin. The doctor wants to prescribe Ciprofloxacin. A single query to the Unified Knowledge Graph triggers a beautiful synthesis of knowledge:

1.  **The Protocol Engine** traverses the graph and finds: `(Ciprofloxacin) <-[:RECOMMENDED_IN_STEP]- (Pneumonia Pathway)`. Verdict: This is a clinically appropriate drug choice according to the pathway.
2.  **The CAE Engine** traverses the same graph from the same `(Ciprofloxacin)` node and finds:
    *   `(Ciprofloxacin) -[:INTERACTS_WITH {severity: "Major"}]-> (Warfarin)`
    *   `(Ciprofloxacin) -[:HAS_DOSING_RULE]-> (RenalAdjustmentRule)`
    Verdict: This drug has a major interaction and requires a dose adjustment for this patient's eGFR.

**The Final, Unified Response:** The Safety Gateway aggregates these findings into a single, intelligent recommendation:

> "Ciprofloxacin is a recommended antibiotic for the Community-Acquired Pneumonia pathway. However, a major interaction with the patient's current Warfarin was detected (increased bleeding risk), and a dose reduction is required for the patient's eGFR of 28. Consider an alternative like..."

This level of sophisticated, cross-domain reasoning is only possible with a unified architecture. This plan provides a pragmatic, powerful, and sustainable path forward, allowing you to build a proprietary, high-performance platform while leveraging the best of both the free and commercial clinical knowledge ecosystems.
