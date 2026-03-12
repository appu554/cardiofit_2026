# KB Update Paths - Where and How Data Gets Updated

**Question**: "When there are updates or new data, WHERE are they updated?"

**Answer**: Updates happen through **4 different paths** depending on the data source and update type.

---

## 📍 The 4 Update Paths

### Path 1: **Scheduled Bulk Updates** (External Source → ETL → Database)

**When**: Monthly/quarterly releases from authoritative sources

**Example 1: SNOMED CT Monthly Release (KB7)**
```
SNOMED International releases new version
  ↓
Admin downloads new RF2 files
  ↓
Places files in: kb-7-terminology/data/snomed/
  ↓
Runs ETL loader: go run internal/etl/enhanced_loaders.go
  ↓
Loader INSERTS/UPDATES PostgreSQL (kb_terminology database)
  ↓
CDC detects changes in PostgreSQL WAL
  ↓
CDC streams to Kafka: kb7.terminology_concepts.changes
  ↓
Downstream consumers (Flink, Flow2) receive updates
```

**Command**:
```bash
# Step 1: Download new SNOMED release
wget https://snomed.org/releases/SNOMED_CT_202401.zip

# Step 2: Extract to data directory
unzip SNOMED_CT_202401.zip -d /kb-7-terminology/data/snomed/

# Step 3: Run loader to UPDATE database
cd kb-7-terminology
go run internal/etl/enhanced_loaders.go --system=SNOMED --data-dir=data/snomed/

# PostgreSQL is now UPDATED with new concepts
# CDC automatically streams changes to Kafka
```

**Other Examples**:
- **RxNorm** (KB7): Monthly drug name updates from NLM
- **LOINC** (KB7): Bi-annual lab code releases
- **ICD-10** (KB7): Annual diagnosis code updates
- **Payer Formularies** (KB6): Monthly formulary changes from insurance companies

---

### Path 2: **Real-Time API Writes** (Application → KB API → Database)

**When**: Applications or users create/update data in real-time

**Example 2: Creating New Drug Rule via KB1 API**
```
Clinical Application (Web UI or Service)
  ↓
HTTP POST /api/v1/drug-rules
  {
    "drug_name": "Vancomycin",
    "rule_type": "renal_dosing",
    "formula": "15 mg/kg adjusted for CrCl"
  }
  ↓
KB1 Service (Go API) receives request
  ↓
Validates and INSERTS into PostgreSQL (kb_drug_rules.drug_rule_packs table)
  ↓
PostgreSQL WAL records the INSERT operation
  ↓
CDC connector detects WAL change
  ↓
CDC streams to Kafka: kb1.drug_rule_packs.changes
  ↓
Flow2 Engine receives event and updates dosing rules cache
```

**Code Example** (How applications write to KB):
```javascript
// Frontend application creating a new drug rule
const response = await fetch('http://localhost:8081/api/v1/drug-rules', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    drug_rxnorm: '11289',
    rule_pack_name: 'Cardiovascular Drugs v2.0',
    dosing_rules: { /* ... */ }
  })
});

// KB1 API handles this and INSERTS into PostgreSQL
// CDC automatically picks up the change
```

**Other Examples**:
- **KB2**: Clinical Decision Support app adds new patient phenotype
- **KB3**: Evidence team publishes updated clinical guideline
- **KB5**: Pharmacist adds new drug interaction warning
- **KB6**: Pharmacy system updates drug inventory levels

---

### Path 3: **Scheduled API Integrations** (External API → KB Service → Database)

**When**: Automated nightly/hourly syncs with external systems

**Example 3: Payer Formulary Updates (KB6)**
```
External Payer API (Blue Cross, UnitedHealth, etc.)
  ↓
Scheduled Job runs (e.g., nightly cron at 2 AM)
  ↓
KB6 Integration Service calls payer APIs
  ↓
GET https://api.bcbs.com/formulary/2025/plans/PPO123
  ↓
KB6 Service transforms response to internal format
  ↓
UPSERTS into PostgreSQL (kb_formulary.formulary_entries table)
  ↓
CDC detects batch of INSERTs/UPDATEs
  ↓
CDC streams to Kafka: kb6.formulary_drugs.changes
  ↓
Flow2 Engine updates formulary cache for real-time lookups
```

**Pseudocode**:
```go
// Scheduled job in KB6 service (runs nightly)
func SyncPayerFormularies() {
    payers := []string{"bcbs", "uhc", "cigna", "aetna"}

    for _, payer := range payers {
        // Call external payer API
        formularyData := fetchPayerFormulary(payer)

        // Batch UPDATE database
        for _, entry := range formularyData {
            db.Exec(`
                INSERT INTO formulary_entries (payer_id, drug_rxnorm, tier, copay)
                VALUES ($1, $2, $3, $4)
                ON CONFLICT (payer_id, drug_rxnorm) DO UPDATE
                SET tier = $3, copay = $4, updated_at = NOW()
            `, entry.PayerID, entry.DrugCode, entry.Tier, entry.Copay)
        }
    }
    // CDC streams all changes to Kafka automatically
}
```

**Other Examples**:
- **KB6**: Drug pricing updates from AWP/WAC databases
- **KB5**: FDA drug safety alerts integration
- **KB3**: Clinical guideline updates from professional societies

---

### Path 4: **Database Migrations** (Schema Updates + Seed Data)

**When**: Initial deployment or major version upgrades

**Example 4: KB5 Enhanced Schema Migration**
```
Developer creates migration: 002_enhanced_schema.sql
  ↓
Migration script contains:
  - New table definitions
  - Seed data (initial drug interactions)
  - Indexes and constraints
  ↓
Run migration: ./migrate_database.sh
  ↓
PostgreSQL CREATES tables and INSERTS seed data
  ↓
CDC detects all INSERTs from seed data
  ↓
CDC streams to Kafka: kb5.drug_interactions.changes
```

**Migration Example**:
```sql
-- 002_enhanced_schema.sql for KB5
-- Creates new tables and inserts initial data

-- Create table
CREATE TABLE ddi_pharmacogenomic_rules (
    id BIGSERIAL PRIMARY KEY,
    gene_symbol TEXT NOT NULL,
    variant_allele TEXT NOT NULL,
    drug_code TEXT NOT NULL
);

-- Insert seed data (this triggers CDC)
INSERT INTO ddi_pharmacogenomic_rules (gene_symbol, variant_allele, drug_code)
VALUES
    ('CYP2D6', '*4/*4', 'codeine'),
    ('CYP2C19', '*2/*2', 'clopidogrel'),
    ('SLCO1B1', '*5', 'simvastatin');

-- CDC streams these INSERTs to Kafka automatically
```

---

## 🔍 Detailed Update Flow by KB

### KB1: Drug Rules Updates

**Update Scenarios**:
1. **Clinician creates new dosing rule** → POST to KB1 API → PostgreSQL → CDC → Kafka
2. **Monthly rule pack release** → Upload TOML file → KB1 parses → PostgreSQL → CDC → Kafka
3. **FDA updates dosing guidelines** → Admin imports → PostgreSQL → CDC → Kafka

**WHERE it happens**:
- API Endpoint: `http://localhost:8081/api/v1/drug-rules` (POST/PUT/PATCH)
- Database: `kb_drug_rules.drug_rule_packs` table
- CDC Topic: `kb1.drug_rule_packs.changes`

---

### KB2: Clinical Context Updates

**Update Scenarios**:
1. **CDS system adds patient phenotype** → POST to KB2 API → PostgreSQL → CDC → Kafka
2. **Clinical team updates state transitions** → KB2 Admin UI → PostgreSQL → CDC → Kafka

**WHERE it happens**:
- API Endpoint: `http://localhost:8086/api/v1/phenotypes` (POST/PUT)
- Database: `kb2_clinical_context.clinical_phenotypes` table
- CDC Topic: `kb2.clinical_phenotypes.changes`

---

### KB3: Guidelines Updates

**Update Scenarios**:
1. **New ACC/AHA guideline published** → Import via KB3 API → PostgreSQL → CDC → Kafka
2. **Evidence team updates protocol** → PUT to KB3 API → PostgreSQL → CDC → Kafka

**WHERE it happens**:
- API Endpoint: `http://localhost:8087/api/v1/guidelines` (POST/PUT)
- Database: `kb3_guidelines.clinical_protocols` table
- CDC Topic: `kb3.clinical_protocols.changes`

---

### KB4: Drug Calculations Updates

**Update Scenarios**:
1. **Pharmacist updates dosing formula** → PATCH to KB4 API → PostgreSQL → CDC → Kafka
2. **New drug added with calculations** → POST to KB4 API → PostgreSQL → CDC → Kafka

**WHERE it happens**:
- API Endpoint: `http://localhost:8088/api/v1/calculations` (POST/PATCH)
- Database: `kb4_drug_calculations.drug_calculations` table
- CDC Topic: `kb4.drug_calculations.changes`

---

### KB5: Drug Interactions Updates

**Update Scenarios**:
1. **FDA publishes new DDI warning** → Scheduled job fetches → KB5 processes → PostgreSQL → CDC → Kafka
2. **Pharmacist adds interaction** → POST to KB5 API → PostgreSQL → CDC → Kafka
3. **Monthly CPIC update** → Import script runs → PostgreSQL → CDC → Kafka

**WHERE it happens**:
- API Endpoint: `http://localhost:8089/api/v1/interactions` (POST/PUT)
- Database: `kb5_drug_interactions.drug_interactions` table
- Scheduled Job: Nightly FDA sync (inserts/updates in batch)
- CDC Topic: `kb5.drug_interactions.changes`

---

### KB6: Formulary Updates

**Update Scenarios**:
1. **Payer sends formulary update** → Webhook received → KB6 processes → PostgreSQL → CDC → Kafka
2. **Monthly formulary sync** → Scheduled job → Fetch from payer APIs → PostgreSQL → CDC → Kafka
3. **Pharmacy updates inventory** → POST to KB6 API → PostgreSQL → CDC → Kafka
4. **Drug pricing change** → Scheduled job fetches AWP/WAC → PostgreSQL → CDC → Kafka

**WHERE it happens**:
- API Endpoint: `http://localhost:8091/api/v1/formulary` (POST/PUT)
- Database: `kb_formulary.formulary_entries` table
- Scheduled Jobs:
  - Nightly payer sync (2 AM)
  - Hourly pricing updates
  - Daily inventory reconciliation
- Webhook Endpoint: `/api/v1/webhooks/payer-updates`
- CDC Topic: `kb6.formulary_drugs.changes`

---

### KB7: Terminology Updates

**Update Scenarios**:
1. **Monthly SNOMED release** → Admin downloads → Run ETL loader → PostgreSQL → CDC → Kafka
2. **Monthly RxNorm update** → Scheduled job downloads → ETL processes → PostgreSQL → CDC → Kafka
3. **ICD-10 annual update** → Import script → ETL loader → PostgreSQL → CDC → Kafka
4. **Manual concept addition** → POST to KB7 API → PostgreSQL → CDC → Kafka

**WHERE it happens**:
- ETL Loaders: `enhanced_loaders.go` (batch processing)
- API Endpoint: `http://localhost:8092/api/v1/concepts` (POST/PUT)
- Database: `kb_terminology.terminology_concepts` table
- Scheduled Jobs:
  - Monthly RxNorm sync (1st of month)
  - Bi-annual SNOMED updates (January, July)
- CDC Topic: `kb7.terminology_concepts.changes`

---

## 📊 Update Frequency Summary

| KB | Real-time API Updates | Scheduled Batch Updates | Update Frequency |
|----|----------------------|------------------------|------------------|
| **KB1** | ✅ Clinician creates rules | ⏰ Monthly rule pack releases | Real-time + Monthly |
| **KB2** | ✅ CDS adds phenotypes | - | Real-time |
| **KB3** | ✅ Evidence team updates | ⏰ Quarterly guideline imports | Real-time + Quarterly |
| **KB4** | ✅ Pharmacist updates formulas | - | Real-time |
| **KB5** | ✅ Adds interactions | ⏰ Nightly FDA sync | Real-time + Daily |
| **KB6** | ✅ Inventory updates | ⏰ Nightly formulary sync, Hourly pricing | Real-time + Hourly/Daily |
| **KB7** | ✅ Manual concepts | ⏰ Monthly RxNorm, Bi-annual SNOMED | Real-time + Monthly |

---

## 🎯 Key Insights

`★ Insight ─────────────────────────────────────`

**WHERE Updates Happen:**

1. **KB Microservice APIs**: Real-time writes from applications
   - Direct HTTP/gRPC calls to KB services (ports 8081-8092)
   - Authenticated CRUD operations
   - Immediate CDC streaming after database write

2. **ETL/Loader Processes**: Bulk imports from external sources
   - Run manually or via scheduled jobs
   - Process external files (SNOMED, RxNorm, formulary feeds)
   - Batch INSERT/UPDATE operations

3. **Scheduled Integration Jobs**: Automated syncs
   - Cron jobs fetch data from external APIs
   - Nightly/hourly/monthly schedules
   - Transform and load into databases

4. **Database Migrations**: Schema and seed data
   - One-time deployments or version upgrades
   - SQL scripts with CREATE + INSERT statements
   - Handled by migration tools (Flyway, golang-migrate)

**Critical Point**: Regardless of HOW data gets written (API, ETL, job, migration), the **CDC system automatically detects ALL database changes** and streams them to Kafka in real-time.

`─────────────────────────────────────────────────`

---

## 🔄 Complete Update Flow Example

**Scenario**: Monthly SNOMED update + Real-time drug rule creation

```
┌─────────────────────────────────────────────────────────────┐
│ MONTH START: Scheduled SNOMED Update                        │
└─────────────────────────────────────────────────────────────┘
1. Cron job triggers: download_snomed_monthly.sh
2. Downloads SNOMED CT 202401 release → data/snomed/
3. ETL loader runs: enhanced_loaders.go
4. 50,000 concepts INSERTED/UPDATED in kb_terminology database
5. CDC streams 50,000 events to kb7.terminology_concepts.changes
6. Flink job consumes and enriches clinical events with new codes

┌─────────────────────────────────────────────────────────────┐
│ SAME DAY: Real-time Drug Rule Creation                      │
└─────────────────────────────────────────────────────────────┘
1. Clinician uses Web UI to create new dosing rule
2. UI sends POST http://localhost:8081/api/v1/drug-rules
3. KB1 service validates and INSERTS into kb_drug_rules database
4. CDC streams 1 event to kb1.drug_rule_packs.changes
5. Flow2 Engine consumes and updates dosing calculator

RESULT: Both bulk updates and real-time writes flow through the
same CDC pipeline to downstream consumers automatically.
```

---

## 🚀 How to Trigger Updates

### For Developers/Admins:

**1. Real-time API Update**:
```bash
# Create new drug interaction via KB5 API
curl -X POST http://localhost:8089/api/v1/interactions \
  -H "Content-Type: application/json" \
  -d '{
    "drug_a": "warfarin",
    "drug_b": "aspirin",
    "severity": "high",
    "clinical_effect": "Increased bleeding risk"
  }'
# → Immediately INSERTS into PostgreSQL
# → CDC streams to Kafka within seconds
```

**2. Scheduled Bulk Update**:
```bash
# Run monthly RxNorm update for KB7
cd kb-7-terminology
./scripts/monthly_rxnorm_sync.sh

# This script:
# 1. Downloads latest RxNorm RRF files
# 2. Runs ETL loader
# 3. Updates PostgreSQL in batches
# 4. CDC automatically streams all changes
```

**3. Database Migration**:
```bash
# Deploy new schema version for KB5
cd kb-5-drug-interactions
./migrate_database.sh

# Migration inserts seed data
# CDC streams all insertions to Kafka
```

---

**Status**: ✅ **All Update Paths Documented**
**Question Answered**: Updates happen via **4 paths** - all flow through databases and trigger CDC streaming automatically
**Next**: Review update schedules and automation requirements for production deployment
