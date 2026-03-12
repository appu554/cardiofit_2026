# KB-7 Terminology Service Implementation Plan
## Enterprise Clinical Decision Support Platform

### 📋 Document Overview
- **Document**: Implementation Plan for KB-7 Terminology Service Enhancement
- **Version**: 1.0
- **Date**: 2025-09-19
- **Status**: Approved for Implementation
- **Scope**: Transform basic terminology mapping to enterprise clinical decision support

---

## 🎯 Executive Summary

### Current State Analysis
The KB-7 Terminology Service represents **30-40% implementation** of the comprehensive specification outlined in the documentation. While operational for basic terminology mappings, it lacks critical components for production clinical environments:

**✅ Currently Functional**:
- Go-based microservice with PostgreSQL storage
- Basic REST API with GraphQL federation support
- SNOMED CT, RxNorm, LOINC, ICD-10 loaders
- Redis caching for performance optimization

**❌ Critical Missing Components**:
- Dual-stream synchronization architecture (0% implemented)
- Semantic reasoning with GraphDB/SPARQL (0% implemented)
- Clinical governance workflows (0% implemented)
- Australian healthcare terminologies (0% implemented)
- Real-time decision support capabilities (0% implemented)

### Strategic Implementation Approach
**5-Phase Progressive Enhancement Strategy** designed to:
1. **Ensure clinical safety** through governance workflows
2. **Enable semantic intelligence** with reasoning capabilities
3. **Support regional deployment** with Australian terminologies
4. **Scale to enterprise grade** with real-time streaming architecture
5. **Achieve operational excellence** with production hardening

---

## 🗺️ Implementation Phases

### Phase 1: Clinical Safety Foundation
**Timeline**: 4-6 weeks | **Priority**: 🚨 Critical | **Risk**: High patient safety impact

#### 1.1 GitOps Clinical Governance Workflow (Weeks 1-2)

**Objective**: Prevent unsafe terminology changes through clinical review process

**Deliverables**:
```yaml
# .github/workflows/terminology-review.yml
clinical_review:
  triggers: [pull_request]
  paths: ['kb-7-terminology/data/**', 'kb-7-terminology/mappings/**']
  required_reviewers:
    - clinical-informatics-lead
    - senior-ontologist
  review_template: |
    ## Clinical Impact Assessment
    - [ ] Patient safety implications reviewed
    - [ ] Drug interaction impacts assessed
    - [ ] Regulatory compliance verified
    - [ ] Rollback strategy documented
```

**Implementation Tasks**:
1. Create GitHub workflow templates for terminology changes
2. Implement PR branch protection rules with clinical reviewer requirements
3. Add automated assignment to clinical review team
4. Create clinical justification templates and validation rules

**Acceptance Criteria**:
- All terminology changes require clinical review before merge
- Automated assignment to clinical informatics team
- PR blocking until clinical approval received
- Clinical impact assessment template completion mandatory

#### 1.2 Provenance & Audit System (Weeks 2-4)

**Objective**: Complete audit trail for all terminology modifications with W3C PROV-O compatibility

**Database Schema Extensions**:
```sql
-- Audit trail tables
CREATE TABLE terminology_changes (
    change_id BIGSERIAL PRIMARY KEY,
    table_name VARCHAR(100) NOT NULL,
    record_id BIGINT NOT NULL,
    operation VARCHAR(20) NOT NULL, -- INSERT, UPDATE, DELETE
    old_values JSONB,
    new_values JSONB,
    change_timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    user_id VARCHAR(100),
    session_id UUID,
    clinical_justification TEXT,
    approval_status VARCHAR(20) DEFAULT 'pending',
    approved_by VARCHAR(100),
    approved_at TIMESTAMP WITH TIME ZONE
);

-- PROV-O style provenance tracking
CREATE TABLE change_provenance (
    provenance_id BIGSERIAL PRIMARY KEY,
    change_id BIGINT REFERENCES terminology_changes(change_id),
    prov_entity JSONB, -- W3C PROV-O Entity
    prov_activity JSONB, -- W3C PROV-O Activity
    prov_agent JSONB, -- W3C PROV-O Agent
    source_checksum VARCHAR(64), -- SHA256 of source data
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Source tracking manifest
CREATE TABLE terminology_sources (
    source_id BIGSERIAL PRIMARY KEY,
    source_name VARCHAR(100) NOT NULL, -- SNOMED CT, RxNorm, etc.
    version VARCHAR(50) NOT NULL,
    download_url TEXT,
    download_date TIMESTAMP WITH TIME ZONE,
    file_size BIGINT,
    sha256_checksum VARCHAR(64),
    validation_status VARCHAR(20) DEFAULT 'pending',
    manifest_data JSONB -- Complete sources.json entry
);
```

**Go Service Extensions**:
```go
// internal/audit/provenance.go
type ProvenanceTracker struct {
    db     *sql.DB
    logger *logrus.Logger
}

type ChangeRecord struct {
    TableName           string                 `json:"table_name"`
    RecordID           int64                  `json:"record_id"`
    Operation          string                 `json:"operation"`
    OldValues          map[string]interface{} `json:"old_values,omitempty"`
    NewValues          map[string]interface{} `json:"new_values,omitempty"`
    ClinicalJustification string              `json:"clinical_justification"`
    UserID             string                 `json:"user_id"`
    SessionID          string                 `json:"session_id"`
}

func (p *ProvenanceTracker) TrackChange(change *ChangeRecord) error {
    // Implement PROV-O compliant change tracking
}
```

**Implementation Tasks**:
1. Extend PostgreSQL schema with audit and provenance tables
2. Implement Go middleware for automatic change tracking
3. Create PROV-O compatible metadata generation
4. Add SHA256 checksum validation for all terminology updates
5. Implement sources.json manifest system

**Acceptance Criteria**:
- All terminology modifications automatically tracked
- W3C PROV-O compatible provenance metadata generated
- SHA256 checksums validated for all terminology sources
- Complete audit trail retrievable for regulatory compliance

#### 1.3 Clinical Policy Flags (Weeks 3-4)

**Objective**: Implement clinical safety constraints through policy flags

**Schema Extensions**:
```sql
-- Add policy flags to mapping tables
ALTER TABLE concept_mappings ADD COLUMN policy_flags JSONB DEFAULT '{}';
ALTER TABLE terminology_concepts ADD COLUMN policy_flags JSONB DEFAULT '{}';

-- Policy flag examples
-- {"doNotAutoMap": true, "requiresClinicalReview": true, "safetyLevel": "high"}
-- {"australianOnly": true, "regulatoryStatus": "approved"}
-- {"deprecationDate": "2025-12-31", "replacementConcept": "SCTID:123456"}
```

**Go Service Policy Engine**:
```go
// internal/policy/engine.go
type PolicyEngine struct {
    rules map[string]PolicyRule
}

type PolicyRule interface {
    Evaluate(ctx context.Context, concept *Concept, operation string) (*PolicyDecision, error)
}

type PolicyDecision struct {
    Allowed    bool     `json:"allowed"`
    Reason     string   `json:"reason,omitempty"`
    Warnings   []string `json:"warnings,omitempty"`
    Requirements []string `json:"requirements,omitempty"`
}

// Built-in policy rules
func (pe *PolicyEngine) RegisterDefaultRules() {
    pe.rules["doNotAutoMap"] = &DoNotAutoMapRule{}
    pe.rules["requiresClinicalReview"] = &ClinicalReviewRule{}
    pe.rules["australianOnly"] = &RegionalRestrictionRule{}
}
```

**Implementation Tasks**:
1. Add JSONB policy_flags columns to relevant tables
2. Implement policy engine with pluggable rules
3. Create built-in safety rules (doNotAutoMap, requiresClinicalReview)
4. Add API middleware for policy validation
5. Implement policy flag management interface

**Acceptance Criteria**:
- Policy flags prevent unsafe automated mapping operations
- Clinical review requirements enforced through policy engine
- Regional restrictions properly validated
- Policy violations logged with detailed reasoning

### Phase 2: Semantic Intelligence Layer
**Timeline**: 8-12 weeks | **Priority**: 🟡 High | **Risk**: Medium complexity

#### 2.1 GraphDB Semantic Store Implementation (Weeks 7-10)

**Objective**: Deploy semantic reasoning infrastructure with RDF/OWL support

**Infrastructure Architecture**:
```yaml
# docker-compose.semantic.yml
version: '3.8'
services:
  graphdb:
    image: ontotext/graphdb:10.1.5
    ports:
      - "7200:7200"
    volumes:
      - graphdb-data:/opt/graphdb/home
    environment:
      - GDB_HEAP_SIZE=4g
      - GDB_JAVA_OPTS=-Xmx4g -Xms2g

  graphdb-workbench:
    image: ontotext/graphdb:10.1.5
    ports:
      - "7201:7200"
    depends_on:
      - graphdb
    environment:
      - GDB_CLUSTER_ENABLED=true
      - GDB_CLUSTER_NODE_ID=1
```

**RDF Data Model Design**:
```turtle
# KB-7 Terminology Ontology (kb7-terminology.ttl)
@prefix kb7: <http://cardiofit.ai/ontology/kb7#> .
@prefix snomed: <http://snomed.info/id/> .
@prefix rxnorm: <http://purl.bioontology.org/ontology/RXNORM/> .
@prefix skos: <http://www.w3.org/2004/02/skos/core#> .
@prefix prov: <http://www.w3.org/ns/prov#> .

# Drug Class Hierarchy
snomed:387517004 a kb7:DrugClass ;
    rdfs:label "ACE Inhibitors"@en ;
    kb7:hasSubClass snomed:386872004 ; # Lisinopril class
    kb7:clinicalSignificance "high" ;
    prov:wasGeneratedBy kb7:SNOMEDImport_2025_01 .

# Drug Interactions
kb7:interaction_ACE_Potassium a kb7:DrugInteraction ;
    kb7:involves snomed:387517004, snomed:387525002 ; # ACE inhibitors, Potassium-sparing diuretics
    kb7:severity "major" ;
    kb7:mechanism "hyperkalemia risk" ;
    kb7:clinicalGuidance "Monitor serum potassium levels closely" .
```

**Go Service GraphDB Integration**:
```go
// internal/semantic/graphdb.go
type GraphDBClient struct {
    baseURL    string
    repository string
    httpClient *http.Client
}

type SPARQLQuery struct {
    Query     string            `json:"query"`
    Variables map[string]string `json:"bindings,omitempty"`
}

func (g *GraphDBClient) ExecuteSPARQL(query *SPARQLQuery) (*SPARQLResults, error) {
    // Execute SPARQL queries against GraphDB
}

func (g *GraphDBClient) LoadTurtleFile(filepath string) error {
    // Load .ttl files into GraphDB repository
}
```

**Implementation Tasks**:
1. Deploy GraphDB cluster (development, staging, production)
2. Design RDF/OWL data model for medical terminologies
3. Implement Go client for GraphDB SPARQL queries
4. Create turtle (.ttl) file format loaders
5. Implement named graphs for versioning and provenance

**Acceptance Criteria**:
- GraphDB cluster operational with SPARQL endpoint
- Medical terminologies loaded as RDF/OWL triples
- SPARQL queries returning semantic relationships
- Named graphs providing version isolation

#### 2.2 ROBOT Tool Pipeline Integration (Weeks 9-12)

**Objective**: Automated ontology validation and transformation pipeline

**ROBOT Pipeline Architecture**:
```makefile
# kb-7-terminology/ontology/Makefile
ODK_VERSION = 1.4.1
ROBOT_VERSION = 1.9.5

# Install ROBOT and ODK
install-tools:
	@echo "Installing ROBOT and ODK tools..."
	curl -L https://github.com/ontodev/robot/releases/download/v$(ROBOT_VERSION)/robot.jar -o bin/robot.jar
	pip install ontology-development-kit==$(ODK_VERSION)

# SNOMED CT RF2 to OWL conversion
snomed-convert:
	@echo "Converting SNOMED CT RF2 to OWL..."
	java -jar bin/robot.jar convert \
		--input data/snomed/rf2/sct2_Concept.txt \
		--format rf2 \
		--output build/snomed-ct.owl

# OWL reasoning validation
validate-ontology:
	@echo "Running OWL reasoning validation..."
	java -jar bin/robot.jar reason \
		--input build/snomed-ct.owl \
		--reasoner ELK \
		--equivalent-classes-allowed none \
		--output build/snomed-ct-reasoned.owl

# Quality control checks
quality-control:
	@echo "Running quality control checks..."
	java -jar bin/robot.jar report \
		--input build/snomed-ct-reasoned.owl \
		--output reports/quality-report.tsv
```

**Go Service ROBOT Integration**:
```go
// internal/ontology/robot.go
type ROBOTProcessor struct {
    robotJarPath string
    workingDir   string
    logger       *logrus.Logger
}

type ValidationResult struct {
    Success     bool              `json:"success"`
    Errors      []string          `json:"errors,omitempty"`
    Warnings    []string          `json:"warnings,omitempty"`
    OutputFile  string            `json:"output_file,omitempty"`
    QualityReport *QualityReport  `json:"quality_report,omitempty"`
}

func (r *ROBOTProcessor) ConvertRF2ToOWL(rf2Dir, outputFile string) (*ValidationResult, error) {
    // Execute ROBOT convert command
}

func (r *ROBOTProcessor) ValidateWithReasoner(owlFile string) (*ValidationResult, error) {
    // Execute ROBOT reason command with ELK reasoner
}
```

**SHACL Validation Rules**:
```turtle
# kb7-validation-rules.ttl
@prefix sh: <http://www.w3.org/ns/shacl#> .
@prefix kb7: <http://cardiofit.ai/ontology/kb7#> .

# Every drug concept must have an RxNorm code
kb7:DrugConceptShape a sh:NodeShape ;
    sh:targetClass kb7:DrugConcept ;
    sh:property [
        sh:path kb7:hasRxNormCode ;
        sh:minCount 1 ;
        sh:maxCount 1 ;
        sh:datatype xsd:string ;
        sh:message "Drug concept must have exactly one RxNorm code" ;
    ] .

# Drug interactions must specify severity level
kb7:DrugInteractionShape a sh:NodeShape ;
    sh:targetClass kb7:DrugInteraction ;
    sh:property [
        sh:path kb7:severity ;
        sh:minCount 1 ;
        sh:in ("minor" "moderate" "major" "contraindicated") ;
        sh:message "Drug interaction must specify valid severity level" ;
    ] .
```

**Implementation Tasks**:
1. Install and configure ROBOT tools and ODK
2. Create automated RF2 to OWL conversion pipeline
3. Implement OWL reasoning validation with ELK/HermiT
4. Design SHACL validation rules for clinical data quality
5. Integrate ROBOT pipeline with Go service ETL process

**Acceptance Criteria**:
- RF2 files automatically converted to OWL format
- OWL reasoning validation catching logical inconsistencies
- SHACL rules enforcing clinical data quality constraints
- Quality control reports generated for every terminology update

#### 2.3 Semantic Query Capabilities (Weeks 10-12)

**Objective**: Enable semantic queries through API with GraphQL integration

**GraphQL Schema Extensions**:
```graphql
# Semantic query extensions
extend type Query {
  # Semantic search with relationship traversal
  searchConceptsSemantic(
    query: String!
    includeHierarchy: Boolean = false
    maxDepth: Int = 3
    relationshipTypes: [String!] = []
  ): SemanticSearchResult!

  # Drug class hierarchy traversal
  getDrugHierarchy(
    conceptId: String!
    direction: HierarchyDirection = DESCENDANTS
    maxLevels: Int = 5
  ): DrugHierarchy!

  # Drug interaction detection
  checkDrugInteractions(
    drugIds: [String!]!
    patientContext: PatientContext
  ): DrugInteractionResult!
}

type SemanticSearchResult {
  concepts: [SemanticConcept!]!
  relationships: [ConceptRelationship!]!
  hierarchy: ConceptHierarchy
  totalCount: Int!
}

type DrugHierarchy {
  rootConcept: SemanticConcept!
  children: [DrugHierarchy!]!
  depth: Int!
  relationshipType: String!
}
```

**SPARQL Query Templates**:
```go
// internal/semantic/queries.go
const (
    DrugHierarchyQuery = `
    PREFIX snomed: <http://snomed.info/id/>
    PREFIX kb7: <http://cardiofit.ai/ontology/kb7#>
    PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

    SELECT ?concept ?label ?parent ?relationshipType WHERE {
        ?concept rdfs:subClassOf* snomed:%s .
        ?concept rdfs:label ?label .
        OPTIONAL {
            ?concept rdfs:subClassOf ?parent .
            ?parent kb7:relationshipType ?relationshipType .
        }
        FILTER(lang(?label) = "en")
    }
    ORDER BY ?label
    `

    DrugInteractionQuery = `
    PREFIX kb7: <http://cardiofit.ai/ontology/kb7#>
    PREFIX snomed: <http://snomed.info/id/>

    SELECT ?interaction ?drug1 ?drug2 ?severity ?mechanism ?guidance WHERE {
        ?interaction a kb7:DrugInteraction ;
                    kb7:involves ?drug1, ?drug2 ;
                    kb7:severity ?severity ;
                    kb7:mechanism ?mechanism ;
                    kb7:clinicalGuidance ?guidance .

        VALUES ?drug1 { %s }
        VALUES ?drug2 { %s }

        FILTER(?drug1 != ?drug2)
    }
    ORDER BY DESC(?severity)
    `
)
```

**Implementation Tasks**:
1. Extend GraphQL schema with semantic query capabilities
2. Implement SPARQL query templates for common use cases
3. Create semantic search service with relationship traversal
4. Add drug hierarchy navigation with configurable depth
5. Implement drug interaction detection service

**Acceptance Criteria**:
- GraphQL API supports semantic queries with relationship traversal
- Drug hierarchy queries return complete parent/child relationships
- Drug interaction detection identifies conflicts with severity levels
- Semantic search includes inference from reasoning engine

### Phase 3: Australian Healthcare Compliance
**Timeline**: 12-16 weeks | **Priority**: 🟡 Medium-High | **Risk**: Medium - Institutional access required

#### 3.1 Australian Terminology Integration (Weeks 13-16)

**Objective**: Full integration of Australian healthcare terminologies

**AMT (Australian Medicines Terminology) Integration**:
```go
// internal/loaders/amt_loader.go
type AMTLoader struct {
    db          *sql.DB
    httpClient  *http.Client
    nctsAuth    *NCTSAuthenticator
    logger      *logrus.Logger
}

type AMTConcept struct {
    TPUUID      string            `json:"tp_uuid"`      // AMT Trade Product UUID
    PreferredTerm string          `json:"preferred_term"`
    CTGMCode     string            `json:"ctgm_code"`    // Clinical Trial Generic Medicine
    MPUUCode     string            `json:"mpuu_code"`    // Medicine Pack Unit of Use
    Strength     string            `json:"strength"`
    Form         string            `json:"form"`
    Route        string            `json:"route"`
    Manufacturer string            `json:"manufacturer"`
    PBS          *PBSInformation   `json:"pbs,omitempty"`
    TGA          *TGAInformation   `json:"tga,omitempty"`
    Mappings     *AMTMappings      `json:"mappings"`
}

type AMTMappings struct {
    SNOMEDCTCode string `json:"snomedct_code,omitempty"`
    RxNormCode   string `json:"rxnorm_code,omitempty"`
    ATCCode      string `json:"atc_code,omitempty"`
}

func (loader *AMTLoader) LoadFromNCTSPortal() error {
    // Implement NCTS portal authentication and download
    // Process AMT RF2 files
    // Create crosswalk mappings to international terminologies
}
```

**NCTS Portal Automation**:
```go
// internal/automation/ncts_client.go
type NCTSAuthenticator struct {
    username    string
    password    string
    selenium    selenium.WebDriver
    baseURL     string
}

func (n *NCTSAuthenticator) AuthenticateAndDownload(terminology string) (*DownloadResult, error) {
    // Selenium automation for NCTS portal
    // Handle multi-factor authentication if required
    // Download terminology files with verification
}
```

**ICD-10-AM Integration**:
```go
// internal/loaders/icd10am_loader.go
type ICD10AMLoader struct {
    db              *sql.DB
    ihacpaAccess    *IHACPAClient
    logger          *logrus.Logger
}

type ICD10AMConcept struct {
    Code            string   `json:"code"`
    Description     string   `json:"description"`
    Category        string   `json:"category"`
    ACSCode         string   `json:"acs_code"`        // Australian Coding Standards
    DRGImpact       bool     `json:"drg_impact"`      // Diagnosis Related Group impact
    ClinicalCoding  *ClinicalCodingGuidance `json:"clinical_coding,omitempty"`
    Mappings        *ICD10AMMappings `json:"mappings"`
}
```

**Implementation Tasks**:
1. Implement AMT loader with NCTS portal integration
2. Create SNOMED CT-AU support with Australian extensions
3. Add ICD-10-AM loader with IHACPA institutional access
4. Implement Selenium automation for authenticated downloads
5. Create Australian-specific crosswalks and validation rules

**Acceptance Criteria**:
- AMT concepts loaded and searchable through API
- SNOMED CT-AU integrated with Australian clinical extensions
- ICD-10-AM available for Australian hospital billing
- Automated downloads from NCTS portal operational

#### 3.2 Regional Crosswalk Management (Weeks 15-16)

**Objective**: Australian-specific mappings and compliance validation

**Regional Mapping Tables**:
```sql
-- Australian crosswalk mappings
CREATE TABLE amt_snomed_mappings (
    mapping_id BIGSERIAL PRIMARY KEY,
    amt_tpuuid VARCHAR(36) NOT NULL,
    snomed_concept_id BIGINT NOT NULL,
    mapping_type VARCHAR(20) NOT NULL, -- exact, broader, narrower
    confidence_score DECIMAL(3,2),
    clinical_verified BOOLEAN DEFAULT false,
    pbs_listed BOOLEAN DEFAULT false,
    tga_approved BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ICD-10-AM to SNOMED CT crosswalks
CREATE TABLE icd10am_snomed_mappings (
    mapping_id BIGSERIAL PRIMARY KEY,
    icd10am_code VARCHAR(10) NOT NULL,
    snomed_concept_id BIGINT NOT NULL,
    drg_impact BOOLEAN DEFAULT false,
    clinical_notes TEXT,
    mapping_quality VARCHAR(20) DEFAULT 'draft', -- draft, reviewed, approved
    reviewed_by VARCHAR(100),
    reviewed_at TIMESTAMP WITH TIME ZONE
);
```

**Regional Policy Engine**:
```go
// internal/policy/australian.go
type AustralianComplianceEngine struct {
    pbsRules    *PBSComplianceRules
    tgaRules    *TGAComplianceRules
    codingRules *ClinicalCodingRules
}

func (ace *AustralianComplianceEngine) ValidateAMTUsage(concept *AMTConcept) (*ComplianceResult, error) {
    // Validate PBS listing status
    // Check TGA approval status
    // Verify clinical coding compliance
}

func (ace *AustralianComplianceEngine) ValidateICD10AMUsage(code string, context *ClinicalContext) (*ComplianceResult, error) {
    // Validate DRG impact assessment
    // Check clinical coding guidelines compliance
    // Verify hospital billing appropriateness
}
```

**Implementation Tasks**:
1. Create Australian-specific mapping tables and indexes
2. Implement SNOMED CT-AU to ICD-10-AM crosswalks
3. Add PBS and TGA compliance validation rules
4. Create Australian clinical coding guidelines enforcement
5. Extend GraphQL schema for Australian terminologies

**Acceptance Criteria**:
- Australian crosswalks operational with confidence scoring
- PBS and TGA compliance rules enforced
- Clinical coding guidelines validated automatically
- Regional policy flags preventing inappropriate usage

### Phase 3.5: Hybrid Multi-Store Architecture
**Timeline**: 17-20 weeks | **Priority**: 🚨 Critical Foundation | **Risk**: Medium - Proven patterns

**Objective**: Implement hybrid multi-store architecture optimized for different query patterns before dual-stream complexity

#### 3.5.1 PostgreSQL Terminology Store (Weeks 17-18)

**Objective**: Fast lookup tier for exact code matching and bulk operations

**PostgreSQL Schema Design**:
```sql
-- Optimized terminology storage for fast lookups
CREATE TABLE terminology_concepts (
    id BIGSERIAL PRIMARY KEY,
    system VARCHAR(20) NOT NULL,  -- 'SNOMED', 'RxNorm', 'ICD10', 'LOINC', 'AMT'
    code VARCHAR(50) NOT NULL,
    display_name TEXT NOT NULL,
    active BOOLEAN DEFAULT true,
    version VARCHAR(20),
    effective_date DATE,
    status_reason VARCHAR(100),
    metadata JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(system, code, version)
);

-- Cross-terminology mappings for fast translation
CREATE TABLE terminology_mappings (
    mapping_id BIGSERIAL PRIMARY KEY,
    source_system VARCHAR(20) NOT NULL,
    source_code VARCHAR(50) NOT NULL,
    target_system VARCHAR(20) NOT NULL,
    target_code VARCHAR(50) NOT NULL,
    mapping_type VARCHAR(50) NOT NULL,  -- 'exact', 'narrow', 'broad', 'related'
    confidence DECIMAL(3,2) DEFAULT 0.95,
    mapping_method VARCHAR(20) DEFAULT 'manual', -- 'manual', 'automated', 'ml'
    clinical_verified BOOLEAN DEFAULT false,
    created_by VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Concept relationships for navigation
CREATE TABLE concept_relationships (
    relationship_id BIGSERIAL PRIMARY KEY,
    source_concept_id BIGINT REFERENCES terminology_concepts(id),
    target_concept_id BIGINT REFERENCES terminology_concepts(id),
    relationship_type VARCHAR(50) NOT NULL, -- 'is_a', 'part_of', 'has_ingredient'
    relationship_group INTEGER DEFAULT 0,
    active BOOLEAN DEFAULT true,
    effective_date DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Performance indexes
CREATE INDEX idx_terminology_lookup ON terminology_concepts(system, code);
CREATE INDEX idx_terminology_active ON terminology_concepts(system, active) WHERE active = true;
CREATE INDEX idx_terminology_search ON terminology_concepts USING gin(to_tsvector('english', display_name));
CREATE INDEX idx_mapping_source ON terminology_mappings(source_system, source_code);
CREATE INDEX idx_mapping_target ON terminology_mappings(target_system, target_code);
CREATE INDEX idx_relationship_source ON concept_relationships(source_concept_id);
CREATE INDEX idx_relationship_type ON concept_relationships(relationship_type);
```

**Data Migration Strategy**:
```python
# scripts/migrate_graphdb_to_hybrid.py
class GraphDBToHybridMigrator:
    """Migrate existing 23,337 triples to optimized hybrid architecture"""

    def __init__(self, graphdb_endpoint, postgres_connection):
        self.graphdb = GraphDBClient(graphdb_endpoint)
        self.postgres = PostgresClient(postgres_connection)
        self.migration_stats = {"concepts": 0, "mappings": 0, "relationships": 0}

    async def migrate_all_data(self):
        """Full migration from GraphDB to hybrid stores"""

        print("🔄 Starting GraphDB to Hybrid Migration...")

        # Step 1: Extract all concepts from GraphDB
        concepts = await self.extract_concepts_from_graphdb()
        await self.load_concepts_to_postgresql(concepts)
        self.migration_stats["concepts"] = len(concepts)

        # Step 2: Extract external terminology mappings
        mappings = await self.extract_mappings_from_graphdb()
        await self.load_mappings_to_postgresql(mappings)
        self.migration_stats["mappings"] = len(mappings)

        # Step 3: Extract relationships for PostgreSQL navigation
        relationships = await self.extract_relationships_from_graphdb()
        await self.load_relationships_to_postgresql(relationships)
        self.migration_stats["relationships"] = len(relationships)

        # Step 4: Optimize GraphDB for reasoning only
        await self.optimize_graphdb_for_reasoning()

        print(f"✅ Migration Complete: {self.migration_stats}")

    async def extract_concepts_from_graphdb(self):
        """Extract all medication concepts with metadata"""
        sparql = """
        PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX skos: <http://www.w3.org/2004/02/skos/core#>

        SELECT ?concept ?system ?code ?label ?altLabel ?safetyLevel ?active ?version
        WHERE {
            ?concept a kb7:MedicationConcept .
            OPTIONAL { ?concept rdfs:label ?label }
            OPTIONAL { ?concept skos:altLabel ?altLabel }
            OPTIONAL { ?concept kb7:safetyLevel ?safetyLevel }
            OPTIONAL { ?concept kb7:active ?active }
            OPTIONAL { ?concept kb7:version ?version }

            # Extract system and code from concept URI
            BIND(
                IF(STRSTARTS(STR(?concept), "http://snomed.info/id/"), "SNOMED",
                IF(STRSTARTS(STR(?concept), "http://purl.bioontology.org/ontology/RXNORM/"), "RxNorm",
                IF(STRSTARTS(STR(?concept), "http://cardiofit.ai/kb7/ontology#"), "KB7", "Unknown")))
                AS ?system
            )

            BIND(
                IF(?system = "SNOMED", STRAFTER(STR(?concept), "http://snomed.info/id/"),
                IF(?system = "RxNorm", STRAFTER(STR(?concept), "http://purl.bioontology.org/ontology/RXNORM/"),
                STRAFTER(STR(?concept), "http://cardiofit.ai/kb7/ontology#")))
                AS ?code
            )
        }
        """
        return await self.graphdb.query(sparql)

    async def optimize_graphdb_for_reasoning(self):
        """Keep only core reasoning data in GraphDB"""

        # Clear non-essential data, keep only:
        # - Core ontology classes and properties
        # - Essential is-a hierarchies
        # - Drug interaction rules
        # - Clinical reasoning axioms

        core_reasoning_data = """
        PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
        PREFIX owl: <http://www.w3.org/2002/07/owl#>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

        CONSTRUCT {
            ?class a owl:Class .
            ?property a owl:ObjectProperty .
            ?concept rdfs:subClassOf ?parent .
            ?interaction a kb7:DrugInteraction .
            ?interaction kb7:involves ?drug .
            ?interaction kb7:safetyLevel ?level .
        }
        WHERE {
            {
                ?class a owl:Class .
                FILTER(STRSTARTS(STR(?class), "http://cardiofit.ai/kb7/ontology#"))
            } UNION {
                ?property a owl:ObjectProperty .
                FILTER(STRSTARTS(STR(?property), "http://cardiofit.ai/kb7/ontology#"))
            } UNION {
                ?concept rdfs:subClassOf ?parent .
            } UNION {
                ?interaction a kb7:DrugInteraction .
                ?interaction kb7:involves ?drug .
                ?interaction kb7:safetyLevel ?level .
            }
        }
        """

        await self.graphdb.clear_repository()
        await self.graphdb.load_construct_query(core_reasoning_data)
        print("✅ GraphDB optimized for reasoning only")
```

**Implementation Tasks**:
1. Set up PostgreSQL with optimized terminology schema
2. Create migration scripts from current GraphDB (23,337 triples)
3. Migrate full dataset to PostgreSQL for fast lookups
4. Optimize GraphDB to contain only core reasoning data
5. Implement performance benchmarking and validation

**Acceptance Criteria**:
- Code lookups <10ms via PostgreSQL
- Full dataset migrated with 100% data integrity
- GraphDB reduced to <5,000 core reasoning triples
- Performance benchmarks established for hybrid queries

#### 3.5.2 Query Router Implementation (Week 19)

**Objective**: Intelligent query routing to optimal data stores

**Query Router Service**:
```go
// pkg/terminology/hybrid_query_router.go
package terminology

import (
    "context"
    "fmt"
    "time"

    "github.com/cardiofit/medication-service-v2/internal/cache"
    "github.com/cardiofit/medication-service-v2/pkg/graphdb"
    "github.com/cardiofit/medication-service-v2/pkg/postgres"
)

type HybridTerminologyService struct {
    graphdb    *graphdb.Client
    postgres   *postgres.TerminologyClient
    cache      *cache.RedisClient
    metrics    *QueryMetrics
}

type QueryMetrics struct {
    PostgresQueries   int64
    GraphDBQueries    int64
    CacheHits         int64
    AverageLatency    map[string]time.Duration
}

type QueryIntent int

const (
    LookupIntent QueryIntent = iota  // Fast exact code lookup
    ReasoningIntent                  // Semantic reasoning/subsumption
    MappingIntent                    // Cross-terminology mapping
    SearchIntent                     // Fuzzy text search
    RelationshipIntent              // Concept relationship traversal
)

func NewHybridTerminologyService(config *Config) *HybridTerminologyService {
    return &HybridTerminologyService{
        graphdb:  graphdb.NewClient(config.GraphDBEndpoint),
        postgres: postgres.NewTerminologyClient(config.PostgresConnection),
        cache:    cache.NewRedisClient(config.RedisConnection),
        metrics:  &QueryMetrics{AverageLatency: make(map[string]time.Duration)},
    }
}

// Core query routing logic
func (h *HybridTerminologyService) routeQuery(ctx context.Context, intent QueryIntent, query *TerminologyQuery) (*TerminologyResult, error) {
    start := time.Now()
    defer func() {
        h.updateMetrics(intent, time.Since(start))
    }()

    switch intent {
    case LookupIntent:
        return h.handleLookupQuery(ctx, query)
    case ReasoningIntent:
        return h.handleReasoningQuery(ctx, query)
    case MappingIntent:
        return h.handleMappingQuery(ctx, query)
    case RelationshipIntent:
        return h.handleRelationshipQuery(ctx, query)
    default:
        return nil, fmt.Errorf("unsupported query intent: %v", intent)
    }
}

// Fast exact lookups go to PostgreSQL
func (h *HybridTerminologyService) LookupConcept(ctx context.Context, system, code string) (*Concept, error) {
    cacheKey := fmt.Sprintf("concept:%s:%s", system, code)

    // Check cache first
    if cached, err := h.cache.Get(ctx, cacheKey); err == nil {
        h.metrics.CacheHits++
        return cached.(*Concept), nil
    }

    // Fast lookup from PostgreSQL
    concept, err := h.postgres.QueryConcept(ctx, system, code)
    if err != nil {
        return nil, err
    }

    // Cache result
    h.cache.Set(ctx, cacheKey, concept, 1*time.Hour)
    h.metrics.PostgresQueries++

    return concept, nil
}

// Semantic reasoning goes to GraphDB
func (h *HybridTerminologyService) FindSubconcepts(ctx context.Context, parentCode string) ([]Concept, error) {
    sparql := fmt.Sprintf(`
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
        PREFIX sct: <http://snomed.info/id/>

        SELECT ?concept ?label WHERE {
            ?concept rdfs:subClassOf* sct:%s .
            OPTIONAL { ?concept rdfs:label ?label }
        }
        LIMIT 50
    `, parentCode)

    result, err := h.graphdb.SPARQLQuery(ctx, sparql)
    if err != nil {
        return nil, err
    }

    h.metrics.GraphDBQueries++
    return h.convertSPARQLToConcepts(result), nil
}

// Cross-terminology mapping via PostgreSQL
func (h *HybridTerminologyService) TranslateConcept(ctx context.Context, fromSystem, fromCode, toSystem string) (*ConceptMapping, error) {
    return h.postgres.QueryMapping(ctx, fromSystem, fromCode, toSystem)
}

// Drug interaction detection via GraphDB reasoning
func (h *HybridTerminologyService) CheckDrugInteractions(ctx context.Context, medicationCodes []string) ([]DrugInteraction, error) {
    sparql := `
        PREFIX kb7: <http://cardiofit.ai/kb7/ontology#>
        PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>

        SELECT ?interaction ?drug1 ?drug2 ?safetyLevel ?description WHERE {
            ?interaction a kb7:DrugInteraction .
            ?interaction kb7:involves ?drug1 .
            ?interaction kb7:involves ?drug2 .
            ?interaction kb7:safetyLevel ?safetyLevel .
            ?interaction rdfs:comment ?description .
            FILTER(?drug1 != ?drug2)
            FILTER(?drug1 IN (%s))
            FILTER(?drug2 IN (%s))
        }
    `

    drugList := fmt.Sprintf("kb7:%s", strings.Join(medicationCodes, ", kb7:"))
    query := fmt.Sprintf(sparql, drugList, drugList)

    result, err := h.graphdb.SPARQLQuery(ctx, query)
    if err != nil {
        return nil, err
    }

    h.metrics.GraphDBQueries++
    return h.convertSPARQLToInteractions(result), nil
}

// Performance monitoring
func (h *HybridTerminologyService) GetMetrics() *QueryMetrics {
    return h.metrics
}

func (h *HybridTerminologyService) updateMetrics(intent QueryIntent, duration time.Duration) {
    switch intent {
    case LookupIntent:
        h.metrics.AverageLatency["lookup"] = duration
    case ReasoningIntent:
        h.metrics.AverageLatency["reasoning"] = duration
    }
}
```

**Query Decision Matrix**:
```go
type QueryDecision struct {
    Intent      QueryIntent
    TargetStore string
    CacheableMinutes int
    PerformanceTarget time.Duration
}

var QueryRouting = map[string]QueryDecision{
    "exact_code_lookup":     {LookupIntent, "postgresql", 60, 10 * time.Millisecond},
    "subsumption_query":     {ReasoningIntent, "graphdb", 30, 50 * time.Millisecond},
    "cross_terminology":     {MappingIntent, "postgresql", 120, 15 * time.Millisecond},
    "drug_interaction":      {ReasoningIntent, "graphdb", 15, 100 * time.Millisecond},
    "concept_hierarchy":     {RelationshipIntent, "postgresql", 45, 25 * time.Millisecond},
    "fuzzy_text_search":     {SearchIntent, "elasticsearch", 30, 50 * time.Millisecond},
}
```

**Implementation Tasks**:
1. Build Go query router with intelligent routing logic
2. Implement caching layer with Redis for hot paths
3. Create performance monitoring and metrics collection
4. Add query optimization and decision matrix
5. Integrate with existing FHIR terminology endpoints

**Acceptance Criteria**:
- Query router correctly routes 100% of request types
- Performance targets met for each query type
- Cache hit ratio >90% for frequent lookups
- Metrics and monitoring operational

#### 3.5.3 FHIR Integration Update (Week 20)

**Objective**: Update FHIR terminology endpoints to use hybrid architecture

**Updated FHIR Terminology Service**:
```go
// internal/fhir/terminology_service.go
type FHIRTerminologyService struct {
    hybridService *terminology.HybridTerminologyService
    validator     *fhir.ResourceValidator
}

func (f *FHIRTerminologyService) CodeSystemLookup(ctx context.Context, req *fhir.LookupRequest) (*fhir.LookupResponse, error) {
    // Route to PostgreSQL for fast lookup
    concept, err := f.hybridService.LookupConcept(ctx, req.System, req.Code)
    if err != nil {
        return nil, err
    }

    return &fhir.LookupResponse{
        Name:        concept.DisplayName,
        Version:     concept.Version,
        Display:     concept.DisplayName,
        Property:    f.buildFHIRProperties(concept),
    }, nil
}

func (f *FHIRTerminologyService) ValueSetExpansion(ctx context.Context, req *fhir.ExpandRequest) (*fhir.ExpandResponse, error) {
    // Route to GraphDB for semantic expansion
    if req.Filter != "" {
        concepts, err := f.hybridService.FindSubconcepts(ctx, req.Filter)
        if err != nil {
            return nil, err
        }

        return &fhir.ExpandResponse{
            ValueSet: f.buildExpandedValueSet(concepts),
        }, nil
    }

    // Route to PostgreSQL for simple value set lookup
    return f.hybridService.ExpandValueSet(ctx, req.ValueSet)
}

func (f *FHIRTerminologyService) ConceptMapTranslate(ctx context.Context, req *fhir.TranslateRequest) (*fhir.TranslateResponse, error) {
    // Route to PostgreSQL for mapping lookup
    mapping, err := f.hybridService.TranslateConcept(ctx, req.System, req.Code, req.TargetSystem)
    if err != nil {
        return nil, err
    }

    return &fhir.TranslateResponse{
        Result: true,
        Match: []fhir.ConceptMapMatch{
            {
                Equivalence: fhir.ConceptMapEquivalence(mapping.MappingType),
                Concept: fhir.Coding{
                    System:  mapping.TargetSystem,
                    Code:    mapping.TargetCode,
                    Display: mapping.TargetDisplay,
                },
            },
        },
    }, nil
}
```

**Implementation Tasks**:
1. Update existing FHIR endpoints to use hybrid query router
2. Implement proper error handling and fallback mechanisms
3. Add FHIR-specific caching and performance optimization
4. Create comprehensive integration tests
5. Update API documentation with performance characteristics

**Acceptance Criteria**:
- All FHIR terminology operations use hybrid architecture
- FHIR response times <200ms for 95% of requests
- Backward compatibility maintained with existing clients
- Integration tests pass with 100% coverage

**Phase 3.5 Success Criteria**:
- ✅ PostgreSQL lookup queries <10ms (95th percentile)
- ✅ GraphDB reasoning queries <50ms (95th percentile)
- ✅ Query router uptime >99.9%
- ✅ Data migration completed with 100% integrity
- ✅ FHIR endpoints using hybrid architecture
- ✅ Cache hit ratio >90% for frequent operations
- ✅ Performance monitoring and alerting operational

### Phase 4: Dual-Stream Architecture
**Timeline**: 21-26 weeks | **Priority**: 🚨 Critical | **Risk**: High complexity (reduced from Very High with hybrid foundation)

#### 4.1 Change Data Capture (CDC) Implementation (Weeks 21-22)

**Objective**: Real-time event streaming from database changes

**Debezium Configuration**:
```yaml
# debezium-kb7-connector.yml
apiVersion: kafka.strimzi.io/v1beta2
kind: KafkaConnector
metadata:
  name: kb7-terminology-cdc
spec:
  class: io.debezium.connector.postgresql.PostgresConnector
  tasksMax: 2
  config:
    database.hostname: kb7-postgres
    database.port: 5433
    database.user: kb7_cdc_user
    database.password: ${file:/opt/kafka/secrets/postgres-credentials.txt:password}
    database.dbname: terminology_db
    database.server.name: kb7-terminology
    table.include.list: "public.terminology_concepts,public.concept_mappings,public.value_sets"
    plugin.name: pgoutput
    publication.autocreate.mode: filtered
    topic.prefix: kb7-cdc
    transforms: "terminology_filter,envelope_extraction"
    transforms.terminology_filter.type: "com.cardiofit.transforms.TerminologyEventFilter"
    transforms.terminology_filter.clinical.impact.threshold: "medium"
    transforms.envelope_extraction.type: "io.debezium.transforms.ExtractNewRecordState"
    key.converter: "org.apache.kafka.connect.json.JsonConverter"
    value.converter: "org.apache.kafka.connect.json.JsonConverter"
```

**Kafka Topic Schema Design**:
```json
{
  "namespace": "com.cardiofit.kb7",
  "type": "record",
  "name": "TerminologyChangeEvent",
  "fields": [
    {
      "name": "eventId",
      "type": "string",
      "doc": "Unique identifier for this change event"
    },
    {
      "name": "eventType",
      "type": {
        "type": "enum",
        "name": "ChangeType",
        "symbols": ["CREATE", "UPDATE", "DELETE"]
      }
    },
    {
      "name": "tableName",
      "type": "string"
    },
    {
      "name": "conceptId",
      "type": "string"
    },
    {
      "name": "terminologySystem",
      "type": "string"
    },
    {
      "name": "clinicalImpact",
      "type": {
        "type": "enum",
        "name": "ClinicalImpact",
        "symbols": ["LOW", "MEDIUM", "HIGH", "CRITICAL"]
      }
    },
    {
      "name": "beforeState",
      "type": ["null", "string"],
      "default": null
    },
    {
      "name": "afterState",
      "type": ["null", "string"],
      "default": null
    },
    {
      "name": "timestamp",
      "type": "long",
      "logicalType": "timestamp-millis"
    },
    {
      "name": "source",
      "type": {
        "type": "record",
        "name": "EventSource",
        "fields": [
          {"name": "version", "type": "string"},
          {"name": "connector", "type": "string"},
          {"name": "name", "type": "string"},
          {"name": "db", "type": "string"},
          {"name": "table", "type": "string"}
        ]
      }
    }
  ]
}
```

**Implementation Tasks**:
1. Configure Debezium PostgreSQL connector for KB-7 tables
2. Set up Kafka topics with Schema Registry integration
3. Implement clinical impact filtering transformation
4. Create outbox pattern for transactionally consistent events
5. Add monitoring and alerting for CDC pipeline health

**Acceptance Criteria**:
- Database changes automatically captured and streamed to Kafka
- Clinical impact filtering routing events appropriately
- Schema Registry managing event schema evolution
- CDC pipeline monitoring showing <100ms capture latency

#### 4.2 Adapter Transformer Service (Weeks 22-23)

**Objective**: Intelligent transformation and routing between streams

**Service Architecture**:
```go
// cmd/adapter-transformer/main.go
type AdapterTransformerService struct {
    kafkaConsumer     *kafka.Consumer
    kafkaProducer     *kafka.Producer
    transformers      map[string]EventTransformer
    router            *EventRouter
    enrichers         map[string]DataEnricher
    deadLetterQueue   *DeadLetterQueueHandler
    metrics           *prometheus.Registry
}

type EventTransformer interface {
    Transform(ctx context.Context, event *TerminologyChangeEvent) (*TransformedEvent, error)
    SupportedEventTypes() []string
    RequiredEnrichments() []string
}

type ClinicalContextEnricher struct {
    knowledgeGraph *neo4j.Driver
    graphDB        *GraphDBClient
    cache          cache.Cache
}

func (cce *ClinicalContextEnricher) EnrichEvent(ctx context.Context, event *TerminologyChangeEvent) (*EnrichedEvent, error) {
    // Add clinical context from Neo4j knowledge graph
    // Include drug class information from GraphDB
    // Attach interaction warnings and contraindications
    // Add regulatory status and compliance flags
}
```

**Transformation Pipeline**:
```go
// internal/transformer/pipeline.go
type TransformationPipeline struct {
    stages []TransformationStage
}

type TransformationStage interface {
    Process(ctx context.Context, event *Event) (*Event, error)
    Name() string
    Metrics() *StageMetrics
}

// Clinical Impact Assessment Stage
type ClinicalImpactAssessmentStage struct {
    riskEngine *ClinicalRiskEngine
}

func (cias *ClinicalImpactAssessmentStage) Process(ctx context.Context, event *Event) (*Event, error) {
    // Analyze clinical impact of terminology change
    // Assess patient safety implications
    // Determine routing priority and target systems
    // Add clinical metadata and warnings
}

// Semantic Enrichment Stage
type SemanticEnrichmentStage struct {
    graphDB    *GraphDBClient
    reasoner   *OwlReasoner
}

func (ses *SemanticEnrichmentStage) Process(ctx context.Context, event *Event) (*Event, error) {
    // Query GraphDB for semantic relationships
    // Add inferred relationships from OWL reasoning
    // Include hierarchical context information
    // Attach related concept recommendations
}
```

**Implementation Tasks**:
1. Build adapter transformer service with pluggable transformers
2. Implement clinical context enrichment from Neo4j
3. Add semantic enhancement using GraphDB queries
4. Create intelligent routing based on event content and priority
5. Implement dead letter queue handling for transformation failures

**Acceptance Criteria**:
- Events enriched with clinical context and semantic information
- Intelligent routing directing events to appropriate downstream systems
- Transformation failures handled gracefully with DLQ processing
- Sub-second transformation latency for standard events

#### 4.3 Multi-Sink Distribution System (Weeks 23-24)

**Objective**: Parallel loading to polyglot persistence architecture

**Sink Connector Architecture**:
```go
// internal/sinks/manager.go
type SinkManager struct {
    sinks       map[string]EventSink
    coordinator *DistributionCoordinator
    consistency *ConsistencyManager
    monitor     *SinkMonitor
}

type EventSink interface {
    Write(ctx context.Context, events []*EnrichedEvent) error
    HealthCheck(ctx context.Context) error
    Name() string
    Config() *SinkConfig
}

// Neo4j Graph Sink
type Neo4jSink struct {
    driver      neo4j.Driver
    batchSize   int
    retryPolicy *RetryPolicy
}

func (n *Neo4jSink) Write(ctx context.Context, events []*EnrichedEvent) error {
    session := n.driver.NewSession(neo4j.SessionConfig{
        AccessMode: neo4j.AccessModeWrite,
    })
    defer session.Close()

    return session.WriteTransaction(func(tx neo4j.Transaction) error {
        for _, event := range events {
            cypherQuery := n.generateCypherFromEvent(event)
            _, err := tx.Run(cypherQuery.Query, cypherQuery.Parameters)
            if err != nil {
                return fmt.Errorf("failed to write event %s: %w", event.ID, err)
            }
        }
        return nil
    })
}

// Elasticsearch Search Index Sink
type ElasticsearchSink struct {
    client      *elasticsearch.Client
    indexName   string
    batchSize   int
}

func (es *ElasticsearchSink) Write(ctx context.Context, events []*EnrichedEvent) error {
    bulk := es.client.Bulk()

    for _, event := range events {
        doc := es.transformToSearchDocument(event)
        bulk.Index().Index(es.indexName).Id(event.ConceptID).Doc(doc)
    }

    response, err := bulk.Do(ctx)
    return es.handleBulkResponse(response, err)
}
```

**Consistency Management**:
```go
// internal/consistency/manager.go
type ConsistencyManager struct {
    coordinationStore *etcd.Client
    checkpoints       map[string]*Checkpoint
}

type TwoPhaseCommitCoordinator struct {
    participants []Participant
    transaction  *Transaction
}

func (tpcc *TwoPhaseCommitCoordinator) Execute(ctx context.Context, operation *DistributedOperation) error {
    // Phase 1: Prepare
    for _, participant := range tpcc.participants {
        if err := participant.Prepare(ctx, operation); err != nil {
            return tpcc.abort(ctx, operation)
        }
    }

    // Phase 2: Commit
    for _, participant := range tpcc.participants {
        if err := participant.Commit(ctx, operation); err != nil {
            // Compensating actions required
            return tpcc.compensate(ctx, operation, participant)
        }
    }

    return nil
}
```

**Implementation Tasks**:
1. Implement parallel sink connectors for Neo4j, Elasticsearch, FHIR stores
2. Add two-phase commit coordination for consistency guarantees
3. Create failure recovery with exponential backoff retry policies
4. Implement load balancing based on sink capacity and health
5. Add comprehensive monitoring and alerting for sink health

**Acceptance Criteria**:
- Events distributed to all configured sinks in parallel
- Consistency maintained across heterogeneous data stores
- Sink failures handled with automatic retry and circuit breaking
- Load balancing optimizing throughput based on sink capacity

#### 4.4 Apache Flink Stream Processing (Weeks 24-25)

**Objective**: Complex event processing with windowed operations

**Flink Job Architecture**:
```java
// src/main/java/com/cardiofit/flink/TerminologyStreamProcessor.java
public class TerminologyStreamProcessor {
    public static void main(String[] args) throws Exception {
        StreamExecutionEnvironment env = StreamExecutionEnvironment.getExecutionEnvironment();

        // Configure Kafka source
        KafkaSource<TerminologyChangeEvent> source = KafkaSource.<TerminologyChangeEvent>builder()
                .setBootstrapServers("localhost:9092")
                .setTopics("kb7-cdc.public.terminology_concepts")
                .setDeserializer(new TerminologyEventDeserializationSchema())
                .setStartingOffsets(OffsetsInitializer.latest())
                .build();

        DataStream<TerminologyChangeEvent> events = env.fromSource(source,
            WatermarkStrategy.forBoundedOutOfOrderness(Duration.ofSeconds(5)),
            "kafka-source");

        // Complex event processing pipeline
        DataStream<ProcessedEvent> processedEvents = events
            .keyBy(event -> event.getTerminologySystem())
            .window(TumblingEventTimeWindows.of(Duration.ofMinutes(5)))
            .aggregate(new TerminologyChangeAggregator())
            .flatMap(new ClinicalImpactAnalyzer())
            .filter(new SafetyThresholdFilter());

        // Multi-sink output
        processedEvents.addSink(new Neo4jSink());
        processedEvents.addSink(new ElasticsearchSink());
        processedEvents.addSink(new FHIRStoreSink());

        env.execute("KB-7 Terminology Stream Processor");
    }
}
```

**State Management for Exactly-Once Processing**:
```java
// Complex event pattern detection
public class DrugSafetyPatternDetector extends KeyedProcessFunction<String, TerminologyChangeEvent, SafetyAlert> {

    private transient ValueState<DrugSafetyContext> safetyContextState;
    private transient MapState<String, InteractionRule> activeRulesState;

    @Override
    public void open(Configuration parameters) {
        ValueStateDescriptor<DrugSafetyContext> safetyDescriptor =
            new ValueStateDescriptor<>("safety-context", DrugSafetyContext.class);
        safetyContextState = getRuntimeContext().getState(safetyDescriptor);

        MapStateDescriptor<String, InteractionRule> rulesDescriptor =
            new MapStateDescriptor<>("active-rules", String.class, InteractionRule.class);
        activeRulesState = getRuntimeContext().getMapState(rulesDescriptor);
    }

    @Override
    public void processElement(TerminologyChangeEvent event, Context ctx, Collector<SafetyAlert> out) throws Exception {
        DrugSafetyContext context = safetyContextState.value();
        if (context == null) {
            context = new DrugSafetyContext();
        }

        // Analyze event for safety implications
        SafetyImpactAnalysis analysis = analyzeSafetyImpact(event, context);

        if (analysis.requiresAlert()) {
            SafetyAlert alert = createSafetyAlert(event, analysis);
            out.collect(alert);

            // Schedule cleanup timer
            ctx.timerService().registerEventTimeTimer(
                ctx.timestamp() + Duration.ofHours(24).toMillis()
            );
        }

        // Update state
        context.addEvent(event);
        safetyContextState.update(context);
    }
}
```

**Implementation Tasks**:
1. Deploy Apache Flink cluster for stream processing
2. Implement windowed aggregations for event batching
3. Add complex event pattern detection for safety alerts
4. Create state management for exactly-once processing guarantees
5. Implement backpressure handling and dynamic scaling

**Acceptance Criteria**:
- Flink jobs processing terminology events with <100ms latency
- Complex event patterns detected for drug safety alerts
- Exactly-once processing guarantees maintained with checkpointing
- Dynamic scaling based on event volume and processing load

#### 4.5 Performance SLA Infrastructure (Weeks 25-26)

**Objective**: End-to-end monitoring with SLA enforcement

**Monitoring Architecture**:
```yaml
# monitoring/prometheus-config.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "kb7-terminology-alerts.yml"

scrape_configs:
  - job_name: 'kb7-terminology-service'
    static_configs:
      - targets: ['localhost:8087']
    metrics_path: '/metrics'
    scrape_interval: 5s

  - job_name: 'adapter-transformer-service'
    static_configs:
      - targets: ['localhost:8088']
    metrics_path: '/metrics'
    scrape_interval: 5s

  - job_name: 'kafka-cluster'
    static_configs:
      - targets: ['localhost:9308']
    metrics_path: '/metrics'
    scrape_interval: 10s

alertmanager_configs:
  - static_configs:
      - targets: ['localhost:9093']
```

**SLA Monitoring Rules**:
```yaml
# kb7-terminology-alerts.yml
groups:
  - name: kb7-terminology-sla
    rules:
      # Patient data stream SLA: < 800ms end-to-end
      - alert: PatientDataLatencyHigh
        expr: histogram_quantile(0.95, rate(kb7_patient_data_processing_duration_seconds_bucket[5m])) > 0.8
        for: 1m
        labels:
          severity: critical
          sla: patient-safety
        annotations:
          summary: "Patient data processing exceeding 800ms SLA"
          description: "95th percentile latency: {{ $value }}s"

      # Knowledge sync stream SLA: < 5 minutes
      - alert: KnowledgeSyncLatencyHigh
        expr: histogram_quantile(0.95, rate(kb7_knowledge_sync_processing_duration_seconds_bucket[5m])) > 300
        for: 2m
        labels:
          severity: warning
          sla: knowledge-sync
        annotations:
          summary: "Knowledge synchronization exceeding 5 minute SLA"
          description: "95th percentile latency: {{ $value }}s"

      # CDC pipeline health
      - alert: CDCPipelineDown
        expr: up{job="debezium-connector"} == 0
        for: 30s
        labels:
          severity: critical
          component: cdc
        annotations:
          summary: "CDC pipeline is down"
          description: "Debezium connector is not responding"
```

**Go Service Metrics**:
```go
// internal/metrics/sla_monitor.go
type SLAMonitor struct {
    patientDataLatency    prometheus.Histogram
    knowledgeSyncLatency  prometheus.Histogram
    cdcHealthGauge        prometheus.Gauge
    sinkHealthGauges      map[string]prometheus.Gauge
    alertManager          *AlertManager
}

func NewSLAMonitor() *SLAMonitor {
    return &SLAMonitor{
        patientDataLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name:    "kb7_patient_data_processing_duration_seconds",
            Help:    "Time taken to process patient data events end-to-end",
            Buckets: []float64{0.1, 0.2, 0.4, 0.8, 1.0, 2.0, 5.0},
        }),
        knowledgeSyncLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
            Name:    "kb7_knowledge_sync_processing_duration_seconds",
            Help:    "Time taken to sync knowledge base changes",
            Buckets: []float64{10, 30, 60, 120, 300, 600, 900},
        }),
        cdcHealthGauge: prometheus.NewGauge(prometheus.GaugeOpts{
            Name: "kb7_cdc_pipeline_health",
            Help: "Health status of CDC pipeline (1=healthy, 0=unhealthy)",
        }),
    }
}

func (sm *SLAMonitor) TrackPatientDataProcessing(startTime time.Time) {
    duration := time.Since(startTime).Seconds()
    sm.patientDataLatency.Observe(duration)

    // Check SLA breach
    if duration > 0.8 { // 800ms SLA
        sm.alertManager.SendAlert(&Alert{
            Level:   Critical,
            Title:   "Patient Data SLA Breach",
            Message: fmt.Sprintf("Processing took %.2fs, exceeding 800ms SLA", duration),
            Tags:    []string{"patient-safety", "sla-breach"},
        })
    }
}
```

**Circuit Breaker Implementation**:
```go
// internal/resilience/circuit_breaker.go
type CircuitBreaker struct {
    name           string
    state          CircuitState
    failureCount   int64
    successCount   int64
    lastFailureTime time.Time
    settings       *CircuitBreakerSettings
    mutex          sync.RWMutex
}

type CircuitBreakerSettings struct {
    MaxFailures        int64
    FailureTimeout     time.Duration
    RecoveryTimeout    time.Duration
    HealthcheckInterval time.Duration
}

func (cb *CircuitBreaker) Execute(operation func() (interface{}, error)) (interface{}, error) {
    cb.mutex.RLock()
    state := cb.state
    cb.mutex.RUnlock()

    switch state {
    case Open:
        if time.Since(cb.lastFailureTime) > cb.settings.RecoveryTimeout {
            cb.setState(HalfOpen)
            return cb.executeWithRecovery(operation)
        }
        return nil, fmt.Errorf("circuit breaker is open for %s", cb.name)

    case HalfOpen:
        return cb.executeWithRecovery(operation)

    case Closed:
        return cb.executeWithMonitoring(operation)

    default:
        return nil, fmt.Errorf("unknown circuit breaker state: %v", state)
    }
}
```

**Implementation Tasks**:
1. Deploy Prometheus and Grafana for comprehensive monitoring
2. Implement SLA tracking with histogram metrics
3. Create alerting rules for performance SLA breaches
4. Add circuit breakers for downstream system failures
5. Build comprehensive dashboards for stream health monitoring

**Acceptance Criteria**:
- Real-time monitoring showing <800ms patient data latency
- Knowledge sync latency tracked with <5 minute SLA
- Automatic alerting when SLAs are breached
- Circuit breakers preventing cascade failures

### Phase 5: Production Operations & Hardening
**Timeline**: 24-28 weeks | **Priority**: 🟢 Medium | **Risk**: Low-Medium

#### 5.1 Advanced ETL Automation (Weeks 27-28)

**Objective**: Fully automated terminology lifecycle management

**Automated Download Manager**:
```go
// internal/automation/download_manager.go
type DownloadManager struct {
    scheduler       *cron.Cron
    authenticators  map[string]Authenticator
    downloaders     map[string]Downloader
    validators      map[string]Validator
    artifactStore   *ArtifactRepository
    notificationSvc *NotificationService
}

type ScheduledDownload struct {
    Source          string                `json:"source"`
    Schedule        string                `json:"schedule"`     // Cron expression
    Authentication  *AuthenticationConfig `json:"authentication"`
    Validation      *ValidationConfig     `json:"validation"`
    PostProcessing  []string              `json:"post_processing"`
    Destinations    []string              `json:"destinations"`
}

// Automated SNOMED CT International downloads
func (dm *DownloadManager) SetupSNOMEDDownloads() {
    dm.scheduler.AddFunc("0 2 1 * *", func() { // Monthly on 1st at 2 AM
        result := dm.downloadAndProcess(&ScheduledDownload{
            Source: "snomed-international",
            Authentication: &AuthenticationConfig{
                Type: "api-key",
                Endpoint: "https://snomedct.org/api/v1/",
            },
            Validation: &ValidationConfig{
                ChecksumValidation: true,
                SchemaValidation:   true,
                QualityChecks:      []string{"concept-count", "relationship-integrity"},
            },
            PostProcessing: []string{"rf2-to-owl", "quality-report", "diff-analysis"},
            Destinations:   []string{"graphdb", "postgresql", "artifact-repository"},
        })

        dm.notificationSvc.NotifyDownloadResult(result)
    })
}
```

**Artifact Repository Integration**:
```go
// internal/artifacts/repository.go
type ArtifactRepository struct {
    client      *nexus.Client
    repository  string
    credentials *RepositoryCredentials
}

type TerminologyArtifact struct {
    Name         string            `json:"name"`
    Version      string            `json:"version"`
    Source       string            `json:"source"`
    DownloadDate time.Time         `json:"download_date"`
    Checksum     string            `json:"checksum"`
    Size         int64             `json:"size"`
    Metadata     map[string]string `json:"metadata"`
    Path         string            `json:"path"`
}

func (ar *ArtifactRepository) StoreArtifact(artifact *TerminologyArtifact, data []byte) error {
    // Upload to Nexus/Artifactory with proper versioning
    // Generate immutable artifact ID
    // Store metadata and provenance information
    // Create download links with authentication
}

func (ar *ArtifactRepository) GetArtifact(name, version string) (*TerminologyArtifact, error) {
    // Retrieve artifact with checksum verification
    // Check access permissions
    // Log access for audit trail
}
```

**Implementation Tasks**:
1. Create automated download manager with cron scheduling
2. Implement API key management for LOINC and authenticated sources
3. Add Nexus/Artifactory integration for artifact storage
4. Create immutable source artifacts with complete version tracking
5. Implement comprehensive download provenance records

**Acceptance Criteria**:
- Terminology sources downloaded automatically according to schedule
- API keys managed securely with rotation capabilities
- All artifacts stored immutably with complete provenance
- Download failures handled with retry policies and notifications

#### 5.2 Production Infrastructure (Weeks 27-28)

**Objective**: Blue-green deployment with zero downtime updates

**Blue-Green Deployment Architecture**:
```yaml
# kubernetes/blue-green-deployment.yml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: kb7-terminology-service
spec:
  replicas: 6
  strategy:
    blueGreen:
      activeService: kb7-terminology-active
      previewService: kb7-terminology-preview
      autoPromotionEnabled: false
      prePromotionAnalysis:
        templates:
        - templateName: success-rate
        args:
        - name: service-name
          value: kb7-terminology-preview
      postPromotionAnalysis:
        templates:
        - templateName: success-rate
        args:
        - name: service-name
          value: kb7-terminology-active
      previewReplicaCount: 2
      scaleDownDelaySeconds: 30
      prePromotionDelay: 30
  selector:
    matchLabels:
      app: kb7-terminology-service
  template:
    metadata:
      labels:
        app: kb7-terminology-service
    spec:
      containers:
      - name: kb7-terminology
        image: cardiofit/kb7-terminology:latest
        ports:
        - containerPort: 8087
        readinessProbe:
          httpGet:
            path: /health
            port: 8087
          initialDelaySeconds: 10
          periodSeconds: 5
        livenessProbe:
          httpGet:
            path: /health
            port: 8087
          initialDelaySeconds: 30
          periodSeconds: 10
```

**Database Migration Strategy**:
```go
// internal/migrations/blue_green.go
type BlueGreenMigrationManager struct {
    blueDB      *sql.DB
    greenDB     *sql.DB
    replicator  *DataReplicator
    validator   *MigrationValidator
}

func (bgmm *BlueGreenMigrationManager) ExecuteZeroDowntimeMigration(migration *Migration) error {
    // Phase 1: Prepare green environment
    if err := bgmm.prepareGreenEnvironment(migration); err != nil {
        return fmt.Errorf("failed to prepare green environment: %w", err)
    }

    // Phase 2: Replicate data to green with dual writes
    if err := bgmm.startDualWrites(); err != nil {
        return fmt.Errorf("failed to start dual writes: %w", err)
    }

    // Phase 3: Validate green environment
    if err := bgmm.validateGreenEnvironment(); err != nil {
        bgmm.rollbackToBlue()
        return fmt.Errorf("green environment validation failed: %w", err)
    }

    // Phase 4: Switch traffic to green
    if err := bgmm.switchTrafficToGreen(); err != nil {
        bgmm.rollbackToBlue()
        return fmt.Errorf("traffic switch failed: %w", err)
    }

    // Phase 5: Cleanup blue environment (after confirmation)
    bgmm.scheduleBlueCleanup(24 * time.Hour)

    return nil
}
```

**Implementation Tasks**:
1. Implement blue-green deployment with ArgoCD/Flux
2. Add zero-downtime database migration capabilities
3. Create comprehensive health checks and readiness probes
4. Implement automated rollback triggers for deployment failures
5. Add deployment pipeline with staging gates and approvals

**Acceptance Criteria**:
- Zero-downtime deployments verified with production traffic
- Automatic rollback on deployment failures or health check failures
- Database migrations execute without service interruption
- Deployment pipeline includes staging validation gates

## 📊 Implementation Timeline & Milestones

### Gantt Chart Overview
```
Weeks 1-6:   Phase 1 - Clinical Safety Foundation
Weeks 7-12:  Phase 2 - Semantic Intelligence Layer
Weeks 13-16: Phase 3 - Australian Healthcare Compliance
Weeks 17-20: Phase 3.5 - Hybrid Multi-Store Architecture
Weeks 21-26: Phase 4 - Dual-Stream Architecture
Weeks 27-28: Phase 5 - Production Operations & Hardening
```

### Key Milestones & Gates

#### Milestone 1: Clinical Governance Operational (Week 6)
**Exit Criteria**:
- ✅ All terminology changes require clinical review
- ✅ Complete audit trail with PROV-O compliance
- ✅ Policy flags preventing unsafe operations
- ✅ SHA256 checksum validation operational

#### Milestone 2: Semantic Reasoning Functional (Week 12)
**Exit Criteria**:
- ✅ GraphDB cluster operational with SPARQL endpoint
- ✅ Drug hierarchy traversals working ("What are all ACE inhibitors?")
- ✅ ROBOT tool pipeline integrated with quality checks
- ✅ GraphQL API includes semantic query capabilities

#### Milestone 3: Australian Healthcare Ready (Week 16)
**Exit Criteria**:
- ✅ AMT, SNOMED CT-AU, ICD-10-AM fully integrated
- ✅ Australian crosswalks operational with validation
- ✅ PBS and TGA compliance rules enforced
- ✅ Regional policy flags preventing inappropriate usage

#### Milestone 4: Real-Time Decision Support (Week 26)
**Exit Criteria**:
- ✅ <800ms end-to-end latency for patient safety queries
- ✅ <5 minutes knowledge base propagation via CDC
- ✅ Multi-sink distribution with consistency guarantees
- ✅ Apache Flink processing complex event patterns

#### Milestone 5: Production Excellence (Week 28)
**Exit Criteria**:
- ✅ Zero-downtime deployments with blue-green strategy
- ✅ Comprehensive monitoring with SLA alerting
- ✅ Disaster recovery procedures tested
- ✅ Performance benchmarks meeting enterprise requirements

## 🔧 Technical Architecture Diagrams

### Current vs. Target Architecture

#### Current Architecture (30-40% Implementation)
```
┌─────────────────────────────┐
│     Go REST API Service     │
│         (Port 8087)         │
└─────────────┬───────────────┘
              │
              ▼
┌─────────────────────────────┐
│      PostgreSQL Store       │
│    (Basic Mappings Only)    │
└─────────────┬───────────────┘
              │
              ▼
┌─────────────────────────────┐
│      Redis Cache Layer      │
│    (Performance Only)       │
└─────────────────────────────┘
```

#### Target Architecture (Full Implementation)
```
┌──────────── CLINICAL GOVERNANCE ────────────┐
│  GitOps Workflow │ Clinical Review │ Audit  │
└──────────────────┬───────────────────────────┘
                   │
┌──────────── AUTHORITATIVE LAYER ────────────┐
│                  │                          │
│  ┌─────────────────┐    ┌─────────────────┐  │
│  │    GraphDB      │◄──►│   PostgreSQL    │  │
│  │   (Semantic)    │    │   (Mappings)    │  │
│  └─────────────────┘    └─────────────────┘  │
└─────────────────┬────────────────────────────┘
                  │
                  ▼
         ┌─────────────────┐
         │  CDC + Debezium │
         │   Event Stream  │
         └─────────┬───────┘
                   │
                   ▼
  ┌─────────────────────────────────────────┐
  │        Apache Kafka Topics              │
  │  • kb7-terminology-changes              │
  │  • kb7-knowledge-sync                   │
  │  • kb7-critical-updates                 │
  └─────────────┬───────────────────────────┘
                │
                ▼
┌───────────────────────────────────────────┐
│      Adapter Transformer Service         │
│  • Clinical Context Enrichment           │
│  • Semantic Enhancement                   │
│  • Intelligent Routing                   │
└─────────────┬─────────────────────────────┘
              │
              ▼
┌──────────── RUNTIME LAYER ───────────────┐
│                                          │
│  ┌─────────────┐  ┌─────────────────────┐ │
│  │    Neo4j    │  │   Elasticsearch     │ │
│  │   (Graph)   │  │     (Search)        │ │
│  └─────────────┘  └─────────────────────┘ │
│                                          │
│  ┌─────────────┐  ┌─────────────────────┐ │
│  │    Redis    │  │    FHIR Store       │ │
│  │   (Cache)   │  │   (Clinical)        │ │
│  └─────────────┘  └─────────────────────┘ │
└──────────────────────────────────────────┘
                   │
                   ▼
      ┌─────────────────────────────┐
      │   Apollo Federation         │
      │   GraphQL Gateway           │
      └─────────────────────────────┘
                   │
                   ▼
      ┌─────────────────────────────┐
      │   Clinical Applications     │
      │   • Safety Gateway          │
      │   • Decision Support        │
      │   • Patient Management      │
      └─────────────────────────────┘
```

### Data Flow Architecture

#### Phase 1-3: Basic Enhancement
```
Terminology Updates → GitOps Review → PostgreSQL + GraphDB → GraphQL API
                                            │
                                            ▼
                                      Regional Terminologies
                                    (AMT, SNOMED CT-AU, ICD-10-AM)
```

#### Phase 4-5: Enterprise Scale
```
Clinical Events ──┐
                  │
Terminology Updates ──► CDC Capture ──► Kafka Topics ──► Flink Processing
                                                              │
                                                              ▼
                        ┌─────────────── Multi-Sink Distribution
                        │                      │                │
                        ▼                      ▼                ▼
                   Neo4j Graph          Elasticsearch      FHIR Store
                        │                      │                │
                        └──────────► Apollo Federation ◄────────┘
                                           │
                                           ▼
                                 Real-time Clinical Services
```

## 📈 Resource Requirements & Budget

### Infrastructure Requirements

#### Phase 1-2: Development & Testing
- **Compute**: 3 VMs (4 vCPU, 16GB RAM each)
- **Storage**: 500GB SSD for databases and artifacts
- **Network**: Standard networking with load balancer
- **Services**: PostgreSQL, Redis, GraphDB, basic monitoring
- **Estimated Monthly Cost**: $800-1,200

#### Phase 3-4: Staging & Production Preparation
- **Compute**: 8 VMs (8 vCPU, 32GB RAM each)
- **Storage**: 2TB SSD + 5TB backup storage
- **Network**: Enhanced networking with CDN
- **Services**: Full stack + Kafka cluster + Apache Flink
- **Estimated Monthly Cost**: $3,000-4,500

#### Phase 5: Production Scale
- **Compute**: 15+ VMs with auto-scaling (16 vCPU, 64GB RAM)
- **Storage**: 10TB+ with replication and disaster recovery
- **Network**: Multi-region with private networking
- **Services**: Full enterprise stack with monitoring, backup, DR
- **Estimated Monthly Cost**: $8,000-12,000

### Development Team Requirements

#### Core Team (Weeks 1-12)
- **1x Senior Go Developer** (GraphDB, SPARQL, semantic reasoning)
- **1x Clinical Informaticist** (terminology standards, clinical workflows)
- **1x Ontology Specialist** (RDF/OWL, ROBOT tools, SHACL validation)
- **0.5x DevOps Engineer** (infrastructure, CI/CD, monitoring)

#### Expanded Team (Weeks 13-26)
- **2x Senior Go/Java Developers** (stream processing, CDC, Flink)
- **1x Australian Healthcare Specialist** (AMT, SNOMED CT-AU, compliance)
- **1x Stream Processing Engineer** (Apache Flink, Kafka, complex event processing)
- **1x Full-time DevOps Engineer** (Kubernetes, deployment automation)
- **1x QA Engineer** (testing automation, performance validation)

#### Production Team (Weeks 27-28+)
- **1x Site Reliability Engineer** (production operations, monitoring)
- **1x Security Engineer** (security hardening, compliance validation)
- **0.5x Technical Writer** (documentation, operational procedures)

### Software Licensing & Tools

#### Development & Testing
- **GraphDB Free Edition**: $0
- **Apache Tools**: $0 (open source)
- **Development Tools**: ~$2,000 (IDEs, monitoring tools)

#### Production
- **GraphDB Enterprise**: ~$50,000-100,000/year
- **Apache Flink Enterprise Support**: ~$25,000/year
- **Monitoring & APM Tools**: ~$15,000/year
- **Security & Compliance Tools**: ~$10,000/year

## 🎯 Success Criteria & KPIs

### Technical Performance Metrics

#### Phase 1: Clinical Safety
- **100% clinical review compliance** for terminology changes
- **Complete audit trail** with <1 second provenance record creation
- **Policy flag effectiveness** preventing 100% of unsafe operations
- **Zero security incidents** related to unauthorized terminology changes

#### Phase 2: Semantic Intelligence
- **Sub-second semantic queries** for drug hierarchy traversals
- **99.9% GraphDB uptime** with automatic failover
- **100% ROBOT validation** passing for all terminology updates
- **GraphQL response times** <200ms for complex semantic queries

#### Phase 3: Australian Compliance
- **Complete AMT coverage** with 99%+ accuracy crosswalks
- **Real-time NCTS synchronization** with <24 hour update lag
- **PBS/TGA compliance** validation catching 100% of violations
- **Australian clinical coding** accuracy >95% validated by clinical experts

#### Phase 4: Real-Time Decision Support
- **<800ms end-to-end latency** for patient safety queries (99th percentile)
- **<5 minute knowledge propagation** for non-critical updates
- **>99.9% exactly-once processing** guarantees maintained
- **Multi-sink consistency** verified through automated testing

#### Phase 5: Production Excellence
- **Zero-downtime deployments** with <30 second traffic switchover
- **<15 minute recovery time** for any component failure
- **>99.95% overall system availability** measured monthly
- **Performance degradation <5%** under 10x expected load

### Business Impact Metrics

#### Clinical Decision Support Enhancement
- **Drug interaction detection** accuracy >98% compared to clinical review
- **Semantic search recall** >95% for clinical terminology queries
- **Query response improvement** 10x faster vs. manual clinical reference
- **Clinical workflow efficiency** 25%+ reduction in terminology lookup time

#### Regulatory & Compliance
- **Audit compliance** 100% pass rate for regulatory reviews
- **Clinical governance** 100% of changes reviewed within SLA
- **Regional compliance** 100% pass rate for Australian healthcare standards
- **Data lineage tracking** complete for >99.9% of terminology changes

#### Operational Excellence
- **Incident response time** <15 minutes for critical issues
- **Mean time to recovery** <1 hour for any service disruption
- **Deployment frequency** daily releases with zero production issues
- **Monitoring coverage** >95% of critical system components

## 🚨 Risk Assessment & Mitigation

### High-Risk Areas

#### 1. Dual-Stream Architecture Complexity (Phase 4)
**Risk**: Technical complexity may lead to delays and stability issues
**Impact**: Medium - Could delay enterprise readiness by 1-2 months (reduced with Phase 3.5 foundation)
**Probability**: Medium (40% - reduced from 60% with hybrid foundation)

**Mitigation Strategies**:
- **Phase 3.5 Foundation**: Implement hybrid architecture first to reduce complexity
- **Proof of Concept First**: Build minimal CDC pipeline before full implementation
- **Incremental Rollout**: Start with single terminology system, expand gradually
- **Expert Consultation**: Engage Apache Flink and Debezium experts early
- **Fallback Plan**: Simpler event-driven updates without full CDC if needed

#### 0. Hybrid Architecture Migration Risk (Phase 3.5)
**Risk**: Data migration from GraphDB-only to hybrid stores may cause data loss or inconsistency
**Impact**: Medium - Could delay Phase 4 start by 2-4 weeks
**Probability**: Low (20% - proven patterns, careful migration)

**Mitigation Strategies**:
- **Comprehensive Backup**: Full GraphDB backup before migration starts
- **Parallel Validation**: Run hybrid and GraphDB in parallel during transition
- **Rollback Capability**: Ability to revert to GraphDB-only if migration fails
- **Data Integrity Testing**: Automated validation of migrated data completeness
- **Incremental Migration**: Migrate terminology systems one at a time

#### 2. GraphDB Licensing & Performance (Phase 2)
**Risk**: Commercial GraphDB licensing costs and performance limitations
**Impact**: Medium - Budget impact $50K-100K annually
**Probability**: Medium (40%)

**Mitigation Strategies**:
- **Open Source Evaluation**: Test Apache Jena Fuseki as alternative
- **Performance Benchmarking**: Validate GraphDB performance early in development
- **License Negotiation**: Explore academic/healthcare pricing tiers
- **Hybrid Architecture**: Use GraphDB for development, evaluate alternatives for production

#### 3. Australian Institutional Access (Phase 3)
**Risk**: Delays in securing NCTS and IHACPA institutional access
**Impact**: Medium - Could delay Australian deployment by 2-4 months
**Probability**: Medium (45%)

**Mitigation Strategies**:
- **Early Engagement**: Start institutional access applications in Phase 1
- **Alternative Sources**: Identify backup terminology sources for development
- **Phased Regional Rollout**: Implement international terminologies first
- **Clinical Partner Support**: Leverage existing Australian clinical relationships

#### 4. Clinical Reviewer Availability (Phase 1)
**Risk**: Limited availability of clinical informaticists for review process
**Impact**: Medium - Could create bottleneck in terminology updates
**Probability**: Medium (50%)

**Mitigation Strategies**:
- **Clinical Advisory Board**: Establish rotating review panel early
- **Automated Pre-screening**: Reduce reviewer workload with AI-assisted triage
- **Priority Classification**: Fast-track critical safety updates
- **External Clinical Consultants**: Engage clinical informatics consultancy if needed

### Medium-Risk Areas

#### 5. Integration Complexity with Existing Services
**Risk**: Complex integration with Apollo Federation and existing Neo4j services
**Impact**: Medium - Integration delays of 2-4 weeks per phase
**Probability**: Medium (40%)

**Mitigation**: Dedicated integration testing environment, early Apollo Federation prototyping

#### 6. Performance at Enterprise Scale
**Risk**: System performance degradation under high load
**Impact**: Medium - May require architecture changes or infrastructure scaling
**Probability**: Medium (35%)

**Mitigation**: Performance testing at each phase, load testing with 10x expected traffic

#### 7. Team Knowledge Gaps
**Risk**: Learning curve for semantic web technologies and stream processing
**Impact**: Low-Medium - Development delays of 1-2 weeks per phase
**Probability**: Medium (45%)

**Mitigation**: Training budget allocation, expert mentorship, knowledge transfer sessions

### Low-Risk Areas

#### 8. Technology Stack Maturity
**Risk**: Chosen technologies may have stability issues
**Impact**: Low - Well-established open source technologies
**Probability**: Low (15%)

**Mitigation**: Use LTS versions, maintain fallback technology options

#### 9. Budget Overruns
**Risk**: Implementation costs exceeding budget by >25%
**Impact**: Low-Medium - Primarily infrastructure and licensing costs
**Probability**: Low (20%)

**Mitigation**: Detailed cost tracking, monthly budget reviews, cost optimization analysis

### Contingency Planning

#### Technical Contingencies
- **GraphDB Alternative**: Apache Jena Fuseki with custom SPARQL endpoints
- **Simplified CDC**: Event-driven updates without Debezium if CDC proves too complex
- **Staged Regional Rollout**: Defer Australian terminologies if institutional access delayed
- **Performance Fallback**: PostgreSQL-only operation with semantic layer as enhancement

#### Resource Contingencies
- **Extended Timeline**: Add 4-6 weeks buffer for complex phases (4-5)
- **Additional Expertise**: Budget for external consultants for specialized areas
- **Infrastructure Scaling**: Auto-scaling capabilities to handle unexpected load
- **Clinical Support**: External clinical informatics consultancy contracts

## 📋 Success Validation & Testing Strategy

### Testing Pyramid Strategy

#### Unit Testing (Developer Responsibility)
- **Go Service Logic**: >90% code coverage for all business logic
- **Semantic Query Logic**: Comprehensive SPARQL query testing
- **Policy Engine**: All policy rules tested with edge cases
- **Transformation Logic**: All event transformers with input validation

#### Integration Testing (Automated CI/CD)
- **Database Integration**: PostgreSQL and GraphDB interaction testing
- **API Integration**: GraphQL federation with semantic queries
- **Stream Processing**: Kafka, Debezium, and Flink pipeline testing
- **Multi-sink Consistency**: Cross-system data consistency validation

#### End-to-End Testing (Staging Environment)
- **Clinical Workflow Testing**: Complete terminology update lifecycle
- **Performance Testing**: Load testing with 10x expected traffic
- **Disaster Recovery Testing**: Failover and recovery procedures
- **Security Testing**: Penetration testing and vulnerability assessment

#### User Acceptance Testing (Clinical Environment)
- **Clinical Expert Review**: Terminology accuracy and semantic correctness
- **Australian Compliance Testing**: Regional terminology validation
- **Performance Validation**: Real-world query performance measurement
- **Clinical Safety Testing**: Drug interaction and contraindication accuracy

### Validation Checkpoints

#### Phase 1 Validation: Clinical Safety
```yaml
validation_criteria:
  governance_workflow:
    - test: "All terminology changes require clinical review"
      method: "Automated PR validation"
      success_criteria: "100% compliance"

  audit_trail:
    - test: "Complete PROV-O compliant audit trail"
      method: "Audit trail analysis"
      success_criteria: "All changes tracked with provenance"

  policy_enforcement:
    - test: "Policy flags prevent unsafe operations"
      method: "Negative testing with unsafe operations"
      success_criteria: "100% prevention of flagged operations"
```

#### Phase 2 Validation: Semantic Intelligence
```yaml
validation_criteria:
  semantic_queries:
    - test: "Drug hierarchy traversal accuracy"
      method: "Clinical expert validation"
      success_criteria: ">95% accuracy vs manual clinical review"

  performance:
    - test: "SPARQL query performance"
      method: "Load testing with complex queries"
      success_criteria: "<1 second for standard drug hierarchy queries"

  reasoning:
    - test: "OWL reasoning correctness"
      method: "ROBOT validation pipeline"
      success_criteria: "Zero logical inconsistencies detected"
```

#### Phase 4 Validation: Real-Time Decision Support
```yaml
validation_criteria:
  latency:
    - test: "End-to-end patient data processing"
      method: "Synthetic patient event testing"
      success_criteria: "<800ms 99th percentile latency"

  consistency:
    - test: "Multi-sink data consistency"
      method: "Cross-system validation queries"
      success_criteria: "100% consistency across all sinks"

  scalability:
    - test: "Stream processing under load"
      method: "10x expected event volume testing"
      success_criteria: "No degradation in processing guarantees"
```

## 🔄 Maintenance & Evolution Strategy

### Post-Implementation Support

#### Operational Support Model
- **L1 Support**: Basic system monitoring and alert response (24/7)
- **L2 Support**: Technical troubleshooting and incident resolution (business hours)
- **L3 Support**: Advanced system engineering and development (on-call)
- **Clinical Support**: Clinical informaticist for terminology and workflow issues

#### Maintenance Schedule
- **Daily**: Automated health checks, backup verification, performance monitoring
- **Weekly**: System performance review, capacity planning analysis
- **Monthly**: Security updates, dependency updates, clinical review of system changes
- **Quarterly**: Disaster recovery testing, compliance audit, performance optimization review
- **Annually**: Architecture review, technology stack evaluation, business continuity planning

### Evolution Roadmap

#### Year 1: Stabilization & Optimization
- **Performance Optimization**: Query optimization, caching improvements, resource tuning
- **Feature Completeness**: Address any gaps identified during initial deployment
- **Clinical Workflow Enhancement**: Based on user feedback and clinical usage patterns
- **Regional Expansion**: Additional international terminologies as needed

#### Year 2: Advanced Features
- **AI/ML Integration**: Machine learning for terminology mapping quality improvement
- **Advanced Analytics**: Clinical terminology usage analytics and insights
- **Mobile API Support**: Mobile-optimized APIs for clinical applications
- **Real-time Clinical Alerts**: Advanced clinical decision support with real-time alerting

#### Year 3: Platform Integration
- **FHIR R5 Support**: Latest FHIR specification implementation
- **Interoperability Standards**: HL7 FHIR terminology services compliance
- **Cloud-Native Optimization**: Kubernetes-native deployment with service mesh
- **Global Deployment**: Multi-region deployment with data residency compliance

### Technology Evolution Strategy

#### Annual Technology Review Process
1. **Technology Landscape Assessment**: Evaluate emerging technologies in semantic web and stream processing
2. **Performance Benchmarking**: Compare current performance against industry standards
3. **Security Assessment**: Review security posture against current threat landscape
4. **Cost Optimization**: Analyze infrastructure costs and optimization opportunities
5. **Clinical Standards Evolution**: Track changes in clinical terminology standards and regulations

#### Planned Technology Migrations
- **GraphDB Version Updates**: Annual updates with performance and feature improvements
- **Apache Flink Evolution**: Stream processing improvements and new features
- **Kubernetes Platform Evolution**: Container orchestration and service mesh enhancements
- **Clinical Standards Updates**: SNOMED CT, RxNorm, and other terminology updates

---

## 📞 Contact & Governance

### Project Governance Structure

#### Executive Steering Committee
- **Chief Medical Officer**: Clinical oversight and final clinical decisions
- **Chief Technology Officer**: Technical direction and resource allocation
- **Chief Information Security Officer**: Security and compliance oversight
- **Clinical Informatics Lead**: Clinical terminology and workflow expertise

#### Technical Advisory Board
- **Senior Ontology Engineer**: Semantic web and terminology expertise
- **Stream Processing Architect**: Real-time data processing and scalability
- **DevOps Lead**: Infrastructure, deployment, and operational excellence
- **Australian Healthcare Compliance Expert**: Regional regulatory expertise

### Communication Plan

#### Regular Reporting
- **Weekly**: Development team standup and progress reports
- **Bi-weekly**: Executive steering committee updates with KPI dashboard
- **Monthly**: Technical advisory board review with architecture decisions
- **Quarterly**: Business stakeholder review with ROI analysis and roadmap updates

#### Escalation Procedures
- **Technical Issues**: Development Lead → Technical Advisory Board → CTO
- **Clinical Issues**: Clinical Informaticist → Clinical Advisory Board → CMO
- **Security Issues**: Security Engineer → CISO (immediate escalation for critical issues)
- **Budget Issues**: Project Manager → Executive Steering Committee

### Documentation Standards

#### Technical Documentation
- **Architecture Decision Records (ADRs)**: All major technical decisions documented
- **API Documentation**: Complete GraphQL schema and REST endpoint documentation
- **Operational Runbooks**: Step-by-step procedures for common operational tasks
- **Security Procedures**: Security configuration and incident response procedures

#### Clinical Documentation
- **Clinical Workflow Guides**: Step-by-step clinical user guides
- **Terminology Change Procedures**: Clinical review and approval workflows
- **Safety Protocols**: Clinical safety procedures and escalation paths
- **Compliance Procedures**: Regulatory compliance and audit procedures

---

*Implementation Plan Version 1.0*
*Generated: 2025-09-19*
*Next Review: 2025-10-19*
*Status: Approved for Implementation*