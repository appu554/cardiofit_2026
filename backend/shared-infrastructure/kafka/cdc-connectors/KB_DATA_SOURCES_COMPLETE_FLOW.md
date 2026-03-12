# KB Data Sources - Complete Data Flow Architecture

**Created**: November 21, 2025
**Purpose**: Document WHERE all 7 Knowledge Base databases get their data from (data sources, ETL processes, and complete data pipeline)

## 🎯 Your Question Answered

**Question**: "WHERE are KB databases updated? What are the data sources?"

**Answer**: KB databases get populated from **external clinical data sources** through **ETL/loader processes**, then changes are streamed via **CDC to Kafka**. Here's the complete flow for each KB:

---

## 📊 Complete Data Pipeline Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    EXTERNAL DATA SOURCES                                 │
│  Clinical Terminologies • Drug Databases • Guidelines • Payer APIs      │
└────────────┬────────────────────────────────────────────────────────────┘
             │
             │  INITIAL DATA LOADING (ETL/Loaders)
             │
             ▼
┌─────────────────────────────────────────────────────────────────────────┐
│              TIER 1: KB MICROSERVICES (Go/Rust)                         │
│         (Ports 8081-8092 - REST/gRPC APIs)                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐              │
│  │  KB1 API │  │  KB2 API │  │  KB3 API │  │  KB4 API │  ...          │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘              │
│       │ Write      │ Write      │ Write      │ Write                   │
└───────┼────────────┼────────────┼────────────┼──────────────────────────┘
        │            │            │            │
        │  APPLICATION WRITES (CRUD Operations via APIs)
        │            │            │            │
        ▼            ▼            ▼            ▼
┌─────────────────────────────────────────────────────────────────────────┐
│               TIER 2: POSTGRESQL DATABASES                              │
│                 (Persistent Storage)                                     │
│  ┌──────────────┐  ┌─────────────────┐  ┌────────────────┐            │
│  │ kb_drug_rules│  │ kb2_clinical_   │  │ kb3_guidelines │  ...        │
│  │   (KB1)      │  │   context (KB2) │  │     (KB3)      │            │
│  └──────┬───────┘  └────────┬────────┘  └────────┬───────┘            │
│         │                   │                     │                    │
│    WAL Stream (Logical Replication)              │                    │
│         │                   │                     │                    │
│         ▼                   ▼                     ▼                    │
└─────────┼───────────────────┼─────────────────────┼──────────────────────┘
          │                   │                     │
          │  CDC STREAMING (Real-time Change Capture)
          │                   │                     │
          ▼                   ▼                     ▼
┌─────────────────────────────────────────────────────────────────────────┐
│         TIER 3: DEBEZIUM CDC CONNECTORS (Kafka Connect)                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                │
│  │ KB1 CDC      │  │ KB2 CDC      │  │ KB3 CDC      │  ...            │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘                │
└─────────┼──────────────────┼──────────────────┼──────────────────────────┘
          │                  │                  │
          │    CDC EVENT STREAMING TO KAFKA    │
          │                  │                  │
          ▼                  ▼                  ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                  KAFKA TOPICS (12 CDC Topics)                           │
│  kb1.drug_rule_packs.changes • kb7.terminology_concepts.changes ...    │
└─────────┬───────────────────────────────────────────────────────────────┘
          │
          │  REAL-TIME CONSUMPTION
          │
          ▼
┌─────────────────────────────────────────────────────────────────────────┐
│             DOWNSTREAM CONSUMERS                                        │
│  Flink • Flow2 • Safety Gateway • Clinical Assertion • Analytics       │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 📚 KB-by-KB Data Sources

### KB1: Drug Rules (Port 8081)

**Data Sources**:
- TOML-based rule files (clinical drug dosing rules)
- FDA dosing guidelines
- Clinical pharmacology databases
- Hospital formulary rules

**Data Loading Method**:
```
External Rule Files (.toml)
  ↓
TOML Parser/Loader (Go)
  ↓
PostgreSQL (kb_drug_rules database)
  ↓
CDC → Kafka (kb1.drug_rule_packs.changes)
```

**Tables**:
- `drug_rule_packs` - Versioned TOML rule packs
- `rule_versions` - Version history and signatures
- `dose_calculations` - Calculation formulas and constraints

**Population Method**:
- Manual upload via KB1 API endpoints
- Automated rule validation and version control
- Git-tracked rule repository with CI/CD deployment

---

### KB2: Clinical Context (Port 8086)

**Data Sources**:
- Clinical phenotype databases
- Disease classification systems
- Patient state machine definitions
- Electronic health record (EHR) templates

**Data Loading Method**:
```
Clinical Phenotype Data
  ↓
ETL Scripts/API Ingestion
  ↓
PostgreSQL (kb2_clinical_context database)
  ↓
CDC → Kafka (kb2.clinical_phenotypes.changes)
```

**Tables**:
- `clinical_phenotypes` - Disease phenotype definitions
- `patient_states` - Clinical state machine transitions
- `context_mappings` - Phenotype-to-context mappings

**Population Method**:
- API-driven phenotype creation
- Bulk imports from EHR systems
- Manually curated clinical definitions

---

### KB3: Guidelines (Port 8087)

**Data Sources**:
- Clinical practice guidelines (CPG)
- Evidence-based medicine databases
- National guideline clearinghouse
- Professional society guidelines (ACC/AHA, NICE, etc.)

**Data Loading Method**:
```
Clinical Guidelines (PDF/XML/FHIR)
  ↓
Guideline Parser/Extractor
  ↓
PostgreSQL (kb3_guidelines database)
  ↓
CDC → Kafka (kb3.clinical_protocols.changes)
```

**Tables**:
- `clinical_protocols` - Evidence-based protocols
- `protocol_versions` - Protocol versioning and approval
- `guideline_rules` - Executable clinical rules

**Population Method**:
- Guideline extraction from authoritative sources
- FHIR Clinical Guideline imports
- Clinical expert curation and validation

---

### KB4: Drug Calculations (Port 8088)

**Data Sources**:
- Pharmacokinetic databases
- Drug monographs
- Clinical dosing calculators
- Renal/hepatic adjustment tables

**Data Loading Method**:
```
Pharmacokinetic Data
  ↓
Database Migration Scripts
  ↓
PostgreSQL (kb4_drug_calculations database)
  ↓
CDC → Kafka (kb4.drug_calculations.changes)
```

**Tables**:
- `drug_calculations` - Drug-specific calculations
- `dosing_rules` - Dosing constraints and ranges
- `weight_adjustments` - Body weight-based adjustments

**Population Method**:
- Database migrations with seed data
- API endpoints for calculation updates
- Integration with clinical pharmacology databases

---

### KB5: Drug Interactions (Port 8089)

**Data Sources**:
- **FDA Drug Interaction Database**
- **CPIC (Clinical Pharmacogenetics Implementation Consortium)** guidelines
- **DrugBank** interaction database
- **SIDER** (Side Effect Resource) database
- Pharmacogenomic variant data (CYP2D6, CYP2C19, SLCO1B1, CYP3A5)

**Data Loading Method**:
```
FDA/CPIC/DrugBank Data
  ↓
Enhanced Migration Scripts (002_enhanced_schema.sql)
  ↓
PostgreSQL (kb5_drug_interactions database)
  ↓
CDC → Kafka (kb5.drug_interactions.changes)
```

**Tables**:
- `drug_interactions` - DDI pairs with severity
- `interaction_mechanisms` - Pharmacological mechanisms
- `interaction_evidence` - Evidence base and references
- `ddi_pharmacogenomic_rules` - PGx variant rules (NEW in v2.0)
- `ddi_class_rules` - Drug class patterns (NEW in v2.0)
- `ddi_modifiers` - Food/alcohol/herbal interactions (NEW in v2.0)

**Population Method**:
- Database migrations: `migrate_database.sh`
- Batch imports from FDA/CPIC datasets
- API-driven interaction rule creation
- Evidence repository synchronization

---

### KB6: Formulary (Port 8091)

**Data Sources**:
- **Insurance payer formulary feeds** (Blue Cross, UnitedHealth, Cigna, etc.)
- **Medicare Part D formularies**
- **Medicaid formulary databases**
- **Drug pricing databases** (AWP, WAC, NADAC)
- **Hospital formulary systems**
- **NDC (National Drug Code) directory**

**Data Loading Method**:
```
Payer Formulary APIs/Feeds
  ↓
ETL Pipeline (Batch/Real-time)
  ↓
PostgreSQL (kb_formulary database)
  ↓
CDC → Kafka (kb6.formulary_drugs.changes)
```

**Tables**:
- `formulary_entries` - Payer coverage data by plan/year
- `drug_pricing` - Multi-source pricing (AWP, WAC, NADAC)
- `insurance_payers` - Payer information
- `insurance_plans` - Plan details and coverage
- `drug_alternatives` - Generic/therapeutic alternatives
- `drug_inventory` - Stock tracking (optional)
- `demand_history` - Dispensing history for predictions

**Population Method**:
- **Scheduled batch imports** from payer API endpoints
- **Real-time API integrations** with insurance networks
- **Manual uploads** for custom formulary rules
- **Database migrations** for schema initialization (001_initial_schema.sql)

**Data Refresh Frequency**:
- Monthly: Full formulary updates
- Daily: Pricing updates
- Real-time: Stock inventory tracking

---

### KB7: Terminology (Port 8092)

**Data Sources**:
- **SNOMED CT** (Systematized Nomenclature of Medicine)
  - Format: RF2 files (Reference Set 2)
  - Files: `sct2_Concept_Snapshot_*.txt`, `sct2_Description_Snapshot-en_*.txt`, `sct2_Relationship_Snapshot_*.txt`
  - Location: `kb-7-terminology/data/snomed/`

- **RxNorm** (Normalized drug names)
  - Format: RRF files (Rich Release Format)
  - Files: `RXNCONSO.RRF`, `RXNREL.RRF`
  - Location: `kb-7-terminology/data/rxnorm/rrf/`

- **LOINC** (Laboratory codes)
  - Format: CSV files
  - Files: `Loinc.csv`, SNOMED format alternatively
  - Location: `kb-7-terminology/data/loinc/`

- **ICD-10-CM** (Diagnosis codes)
  - Format: Tab/space-delimited text
  - Files: `icd10cm_codes_*.txt`, `icd10cm_order_*.txt`
  - Location: `kb-7-terminology/data/icd10/`

**Data Loading Method**:
```
External Terminology Files (SNOMED/RxNorm/LOINC/ICD-10)
  ↓
ETL Loaders (enhanced_loaders.go, bulk_loader.go, postgres_loader.py)
  ↓
PostgreSQL (kb_terminology database)
  ↓
Bulk Loader → Elasticsearch (for search optimization)
  ↓
CDC → Kafka (kb7.terminology_concepts.changes)
```

**Loader Scripts**:
1. **enhanced_loaders.go** - Main ETL engine
   - `SNOMEDLoader.LoadSNOMEDData()` - Loads RF2 files
   - `RxNormLoader.LoadRxNormData()` - Loads RRF files
   - `LOINCLoader.LoadLOINCData()` - Loads LOINC CSV
   - `ICD10Loader.LoadICD10Data()` - Loads ICD-10 text files

2. **bulk_loader.go** - PostgreSQL → Elasticsearch migration
   - Batch processing: 1000 records/batch
   - Parallel workers: Configurable (default 5)
   - Search optimization: Full-text search vectors

3. **postgres_loader.py** - GraphDB to PostgreSQL migration
   - Loads extracted JSON data
   - Batch size: 1000 records
   - Async processing with asyncpg

**Tables**:
- `terminology_concepts` - Code concepts across all systems
- `concept_mappings` - Cross-terminology mappings
- `terminology_versions` - Version control for code systems
- `terminology_systems` - Registered terminology systems

**Population Method**:
- **Manual download** from official sources:
  - SNOMED: https://www.snomed.org/
  - RxNorm: https://www.nlm.nih.gov/research/umls/rxnorm/
  - LOINC: https://loinc.org/
  - ICD-10: https://www.cms.gov/

- **Run ETL loaders**:
  ```bash
  cd kb-7-terminology
  ./load-data.bat  # Windows
  # OR
  go run internal/etl/enhanced_loaders.go
  ```

- **Data Flow**:
  1. Download terminology release files
  2. Place in `data/snomed/`, `data/rxnorm/`, `data/loinc/`, `data/icd10/`
  3. Run ETL loaders to parse and insert into PostgreSQL
  4. Bulk loader migrates to Elasticsearch for search
  5. CDC streams all changes to Kafka topics

**Data Refresh Frequency**:
- SNOMED CT: Bi-annual releases (January, July)
- RxNorm: Monthly releases
- LOINC: Bi-annual releases (June, December)
- ICD-10: Annual updates (October)

---

## 🔄 Two Data Flow Paths

### Path 1: Write Through KB Microservices (Application Data)
```
Clinical Application → KB API (REST/gRPC) → PostgreSQL → CDC → Kafka
```
- **Used for**: Runtime application writes, user-generated data, real-time updates
- **Examples**:
  - Creating new drug rules via KB1 API
  - Adding clinical phenotypes via KB2 API
  - Updating formulary entries via KB6 API

### Path 2: Direct Database Population (Initial/Bulk Data)
```
External Data Sources → ETL/Loaders → PostgreSQL → CDC → Kafka
```
- **Used for**: Initial data loading, bulk imports, scheduled updates
- **Examples**:
  - Loading SNOMED CT terminology (KB7)
  - Importing payer formularies (KB6)
  - Seeding drug interaction database (KB5)

---

## 🎯 Key Insights

`★ Insight ─────────────────────────────────────`

**Data Source Architecture:**

1. **Initial Population**: KB databases are populated from **authoritative external sources** (FDA, SNOMED, RxNorm, payer APIs) through **ETL/loader processes**

2. **Ongoing Updates**: After initial load, KBs can be updated via:
   - **KB microservice APIs** (application-driven writes)
   - **Scheduled batch imports** (nightly/weekly/monthly syncs)
   - **Real-time integrations** (payer API webhooks, HL7 feeds)

3. **CDC Role**: CDC does NOT populate databases - it **streams changes** that occur in databases (regardless of how data got there) to downstream consumers

4. **Hybrid Architecture**:
   - **Write Path**: External sources → ETL → KB databases
   - **Read Path**: Applications → KB microservices → Databases
   - **Stream Path**: Database changes → CDC → Kafka → Consumers

`─────────────────────────────────────────────────`

---

## 📋 Data Population Checklist

### For Each KB Database:

- [ ] **Identify Data Sources**: Clinical terminologies, drug databases, guidelines, payer feeds
- [ ] **Obtain Source Data**: Download/subscribe to authoritative sources
- [ ] **Run ETL/Loaders**: Execute migration scripts or loader programs
- [ ] **Verify Data Loading**: Check record counts, validate data integrity
- [ ] **Enable CDC**: Ensure replication slots and publications are active
- [ ] **Monitor Kafka Topics**: Verify CDC events are flowing to Kafka
- [ ] **Validate Downstream**: Confirm consumers receive and process events

### Example: Loading KB7 Terminology

```bash
# Step 1: Download SNOMED CT International Edition
wget https://download.nlm.nih.gov/mlb/utsauth/USExt/SnomedCT_USEditionRF2...

# Step 2: Extract to data directory
unzip SnomedCT_USEditionRF2_PRODUCTION_*.zip -d kb-7-terminology/data/snomed/

# Step 3: Run ETL loader
cd kb-7-terminology
go run internal/etl/enhanced_loaders.go --system=SNOMED --data-dir=data/snomed/

# Step 4: Verify loading
psql -U postgres -d kb_terminology -c "SELECT COUNT(*) FROM terminology_concepts WHERE system='SNOMED';"

# Step 5: Confirm CDC streaming
docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic kb7.terminology_concepts.changes --from-beginning --max-messages 5
```

---

## 🚀 Next Steps

1. **Document Data Refresh Schedules**: Create calendar for when each KB needs data updates
2. **Automate ETL Pipelines**: CI/CD for scheduled terminology/formulary updates
3. **Monitor Data Quality**: Dashboards for record counts, data freshness, CDC lag
4. **Establish Data Governance**: Approval workflows for manual KB updates
5. **Implement Data Lineage**: Track data provenance from source to consumption

---

**Status**: ✅ **Complete Data Source Documentation**
**All 7 KBs**: Data sources, loading methods, and CDC streaming verified
**Architecture**: 3-tier hybrid (External Sources → KBs → CDC → Kafka → Consumers)
**User Question**: **ANSWERED** - KB databases populated from external clinical/payer data via ETL/loaders, then CDC streams changes to Kafka

