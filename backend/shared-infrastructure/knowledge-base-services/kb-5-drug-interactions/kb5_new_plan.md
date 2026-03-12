# KB-5 Governance Implementation Plan

## Executive Summary

Add governance metadata (provenance tracking) to KB-5 Drug-Drug Interactions to transform "software opinions" into "medico-legal clinical guidance" by embedding the three-layer authority model directly in `DrugInteraction` records.

## Current State Analysis (Verified by Code Review)

### What Already Exists

| Component | Location | Lines | Status |
|-----------|----------|-------|--------|
| KB-0 Governance Platform | `kb-0-governance-platform/` | 770+ | ✅ Fully implemented |
| Authority Registry | `kb-0/.../types.go:59-83` | - | ✅ 17+ authorities |
| ClinicalSource enum | `kb-5/.../governance_models.go:131-148` | 17 | ✅ 13 sources defined |
| AttributionEnvelope | `kb-5/.../governance_models.go:193-227` | 35 | ✅ Provenance structure |
| GovernedInteractionResult | `kb-5/.../governance_models.go:234-252` | 19 | ✅ Wrapper exists |
| DrugInteraction struct | `kb-5/.../interactions.go:14-60` | 46 | ❌ **NO governance fields** |

### The Specific Gap

```go
// CURRENT: interactions.go:14-60 - Missing governance
type DrugInteraction struct {
    Severity       string  // "major" - but WHO SAYS SO?
    EvidenceLevel  string  // "established" - ACCORDING TO WHOM?
    Mechanism      string  // CYP3A4 inhibition - WHERE'S THE DATA?
    // NO source attribution, NO legal defensibility
}
```

### Missing Authorities in KB-0

| Authority | Purpose | Gap |
|-----------|---------|-----|
| CredibleMeds | QT prolongation risk | Not registered |
| UW DDI Database | Transporter interactions (P-gp, OATP) | Not registered |
| Flockhart CYP Table | Gold standard CYP classification | Not registered |
| Stockley | DDI reference text (UK) | Not registered |
| AMH | Australian Medicines Handbook | Not registered |
| BNF | British National Formulary | Not registered |

## Implementation Approach

### Phase 1: Add DDIGovernance Struct to governance_models.go

**File**: `kb-5-drug-interactions/internal/models/governance_models.go`
**Location**: After line 191 (after ProgramFlag constants, before AttributionEnvelope)

Add new struct implementing three-layer authority model:

```go
// DDIGovernance provides per-interaction provenance tracking
// Implements the three-layer authority model for medico-legal defensibility
type DDIGovernance struct {
    // Layer 1: REGULATORY AUTHORITY - "What is LEGALLY prohibited?"
    RegulatoryAuthority   ClinicalSource `gorm:"size:50" json:"regulatory_authority"`    // FDA_LABEL, TGA, etc.
    RegulatoryDocument    string         `gorm:"size:500" json:"regulatory_document"`    // "Warfarin FDA Label Section 7"
    RegulatoryURL         string         `gorm:"type:text" json:"regulatory_url"`        // Link to source
    RegulatoryJurisdiction string        `gorm:"size:20" json:"regulatory_jurisdiction"` // US, AU, IN, EU

    // Layer 2: PHARMACOLOGY AUTHORITY - "WHY does this happen?"
    PharmacologyAuthority ClinicalSource `gorm:"size:50" json:"pharmacology_authority"`  // DRUGBANK, PHARMGKB
    MechanismEvidence     string         `gorm:"size:500" json:"mechanism_evidence"`     // "CYP2C9 Ki = 7.1 μM"
    TransporterData       string         `gorm:"size:200" json:"transporter_data"`       // "P-gp substrate"
    CYPPathway            string         `gorm:"size:100" json:"cyp_pathway"`            // "CYP3A4, CYP2C9"

    // Layer 3: CLINICAL PRACTICE AUTHORITY - "HOW do clinicians manage?"
    ClinicalAuthority     ClinicalSource `gorm:"size:50" json:"clinical_authority"`      // LEXICOMP, MICROMEDEX
    SeveritySource        string         `gorm:"size:200" json:"severity_source"`        // "Lexicomp 2024"
    ManagementSource      string         `gorm:"size:200" json:"management_source"`      // "ACCP Guidelines 2024"

    // Quality & Review Metadata
    EvidenceGrade         string         `gorm:"size:10" json:"evidence_grade"`          // A, B, C, D per ACCP
    LastReviewedDate      *time.Time     `json:"last_reviewed_date,omitempty"`
    NextReviewDue         *time.Time     `json:"next_review_due,omitempty"`
    ReviewedBy            string         `gorm:"size:100" json:"reviewed_by,omitempty"`
}
```

### Phase 2: Embed DDIGovernance in DrugInteraction

**File**: `kb-5-drug-interactions/internal/models/interactions.go`
**Location**: After line 59 (after UpdatedBy field, before closing brace)

Add embedded governance field:

```go
    // Governance provenance (Three-layer authority model)
    Governance          DDIGovernance `json:"governance" gorm:"embedded;embeddedPrefix:gov_"`
```

This uses GORM's embedded struct feature with prefix to create columns like:
- `gov_regulatory_authority`
- `gov_regulatory_document`
- `gov_pharmacology_authority`
- etc.

### Phase 3: Register Missing Authorities in KB-0

**File**: `kb-0-governance-platform/internal/models/types.go`

Add missing authorities (Lines ~83):

```go
// Additional DDI-specific authorities
AuthorityCredibleMeds   Authority = "CREDIBLEMEDS"   // QT prolongation risk
AuthorityUWDDI          Authority = "UW_DDI"         // Transporter interactions
AuthorityPharmGKB       Authority = "PHARMGKB"       // Pharmacogenomics
AuthorityFlockhart      Authority = "FLOCKHART_CYP"  // CYP classification gold standard
AuthorityStockley       Authority = "STOCKLEY"       // DDI reference text (UK)
AuthorityAMH            Authority = "AMH"            // Australian Medicines Handbook
AuthorityBNF            Authority = "BNF"            // British National Formulary
```

### Phase 4: Add Database Migration

**File**: `kb-5-drug-interactions/migrations/006_ddi_governance_metadata.sql`

```sql
-- Add governance columns to drug_interactions table
ALTER TABLE drug_interactions
ADD COLUMN IF NOT EXISTS gov_regulatory_source VARCHAR(100),
ADD COLUMN IF NOT EXISTS gov_regulatory_document VARCHAR(500),
ADD COLUMN IF NOT EXISTS gov_regulatory_url TEXT,
ADD COLUMN IF NOT EXISTS gov_pharmacology_source VARCHAR(100),
ADD COLUMN IF NOT EXISTS gov_mechanism_evidence VARCHAR(500),
ADD COLUMN IF NOT EXISTS gov_transporter_data VARCHAR(500),
ADD COLUMN IF NOT EXISTS gov_clinical_source VARCHAR(100),
ADD COLUMN IF NOT EXISTS gov_severity_source VARCHAR(100),
ADD COLUMN IF NOT EXISTS gov_management_source VARCHAR(100),
ADD COLUMN IF NOT EXISTS gov_last_reviewed_date DATE,
ADD COLUMN IF NOT EXISTS gov_next_review_due DATE,
ADD COLUMN IF NOT EXISTS gov_reviewed_by VARCHAR(200),
ADD COLUMN IF NOT EXISTS gov_evidence_grade VARCHAR(10);

-- Create index for authority queries
CREATE INDEX IF NOT EXISTS idx_ddi_regulatory_source
ON drug_interactions(gov_regulatory_source);
```

### Phase 5: Update Seed Data with Governance

**File**: `kb-5-drug-interactions/migrations/` (update existing seed data)

Example transformation:

```sql
-- Before: No provenance
INSERT INTO drug_interactions (interaction_id, drug_a_code, drug_b_code, severity)
VALUES ('ddi-warfarin-fluconazole', 'warfarin', 'fluconazole', 'major');

-- After: Full provenance
INSERT INTO drug_interactions (
    interaction_id, drug_a_code, drug_b_code, severity,
    gov_regulatory_source, gov_regulatory_document,
    gov_pharmacology_source, gov_mechanism_evidence,
    gov_clinical_source, gov_severity_source,
    gov_evidence_grade
) VALUES (
    'ddi-warfarin-fluconazole', 'warfarin', 'fluconazole', 'major',
    'FDA.Section7', 'Warfarin Sodium Tablets FDA Label 2024',
    'DrugBank', 'CYP2C9 inhibition, Ki = 7.1 μM',
    'Lexicomp', 'Lexicomp Drug Interactions 2024',
    'A'
);
```

## Files to Modify

| File | Action | Description |
|------|--------|-------------|
| `kb-5/.../governance_models.go` | Edit | Add DDIGovernance struct |
| `kb-5/.../interactions.go` | Edit | Embed DDIGovernance in DrugInteraction |
| `kb-0/.../types.go` | Edit | Add 7 missing authorities |
| `kb-5/migrations/006_*.sql` | Create | New migration for governance columns |
| `kb-5/migrations/004_seed_*.sql` | Edit | Update seed data with governance |

## Verification Plan

1. **Build KB-5 Service**
   ```bash
   cd backend/shared-infrastructure/knowledge-base-services/kb-5-drug-interactions
   go build ./...
   ```

2. **Run Database Migration**
   ```bash
   make migrate-up
   ```

3. **Verify Governance in API Response**
   ```bash
   curl http://localhost:8089/api/v1/interactions/check \
     -H "Content-Type: application/json" \
     -d '{"drug_codes": ["warfarin", "fluconazole"]}'
   ```

   Expected: Response includes `governance` object with all three layers populated

4. **Run Tests**
   ```bash
   go test ./... -v
   ```

## Risk Assessment

| Risk | Mitigation |
|------|------------|
| Breaking existing API | Governance field is additive, not breaking |
| Migration on production DB | Run in transaction, test on staging first |
| Missing governance data | Default to "UNKNOWN" with review flag |

## Out of Scope

- Updating all 50+ existing interactions with full governance (separate task)
- Implementing governance workflow UI
- P&T committee override workflows
- Real-time authority source updates

## Phase 1.5: Add Missing ClinicalSource Constants

**File**: `kb-5-drug-interactions/internal/models/governance_models.go`
**Location**: After line 148 (after existing ClinicalSource constants)

```go
// Additional DDI-specific clinical sources (Layer 2 & 3)
ClinicalSourceCredibleMeds  ClinicalSource = "CREDIBLEMEDS"   // QT prolongation risk
ClinicalSourceUWDDI         ClinicalSource = "UW_DDI"         // University of Washington DDI Database
ClinicalSourceFlockhart     ClinicalSource = "FLOCKHART_CYP"  // Indiana University CYP Table
ClinicalSourceStockley      ClinicalSource = "STOCKLEY"       // Stockley's Drug Interactions
ClinicalSourceAMH           ClinicalSource = "AMH"            // Australian Medicines Handbook
ClinicalSourceBNF           ClinicalSource = "BNF"            // British National Formulary
ClinicalSourceTGA           ClinicalSource = "TGA"            // Therapeutic Goods Administration (AU)
ClinicalSourceCDSCO         ClinicalSource = "CDSCO"          // Central Drugs Standard Control (IN)
ClinicalSourceEMA           ClinicalSource = "EMA"            // European Medicines Agency
```

## Dependencies

- KB-0 governance platform must be running for authority validation
- PostgreSQL database for migrations
- Go 1.21+ for build

## Implementation Order

```
1. Add ClinicalSource constants (governance_models.go)
2. Add DDIGovernance struct (governance_models.go)
3. Embed in DrugInteraction (interactions.go)
4. Create migration (006_ddi_governance_metadata.sql)
5. Run migration and verify
6. Register authorities in KB-0 (types.go)
7. Update seed data with governance (optional, separate task)
```

## Expected Outcome

After implementation, every DDI API response will include:

```json
{
  "interaction_id": "ddi-warfarin-fluconazole",
  "severity": "major",
  "governance": {
    "regulatory_authority": "FDA_LABEL",
    "regulatory_document": "Warfarin Sodium Tablets FDA Label Section 7",
    "regulatory_jurisdiction": "US",
    "pharmacology_authority": "DRUGBANK",
    "mechanism_evidence": "CYP2C9 inhibition, Ki = 7.1 μM",
    "cyp_pathway": "CYP2C9",
    "clinical_authority": "LEXICOMP",
    "severity_source": "Lexicomp Drug Interactions 2024",
    "evidence_grade": "A"
  }
}
```

This transforms the interaction from **"software opinion"** to **"medico-legally defensible clinical guidance"**.
