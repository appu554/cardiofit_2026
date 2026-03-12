# DailyMed SPL Fetcher - Technical Design Verification

## Implementation Status: Complete ✅

**Date**: 2026-01-24
**Phase**: 3a.3 Foundation Infrastructure
**Component**: `shared/datasources/dailymed/`

## Files Implemented

| File | Purpose | Lines | Status |
|------|---------|-------|--------|
| `fetcher.go` | Main SPL Fetcher interface & API client | 720 | ✅ Enhanced |
| `table_classifier.go` | Table type detection (GFR, DDI, PK) | 400 | ✅ New |
| `section_router.go` | LOINC → KB routing (Truth Manifest) | 425 | ✅ New |
| `storage.go` | Postgres + S3 storage layer | 650 | ✅ New |
| `delta_syncer.go` | Incremental sync logic | 540 | ✅ New |

**Total**: ~2,735 lines of Go code

---

## Exit Criteria Verification

### From Technical Design Document:

| # | Exit Criterion | Implementation | Status |
|---|----------------|----------------|--------|
| 1 | Fetch SPL by SetID | `SPLFetcher.FetchBySetID()` | ✅ |
| 2 | Fetch SPL by RxCUI | `SPLFetcher.FetchByRxCUI()` | ✅ |
| 3 | Fetch SPL by NDC | `SPLFetcher.FetchByNDC()` | ✅ |
| 4 | Parse 12 LOINC sections | `SPLDocument.GetSection()` + LOINC constants | ✅ |
| 5 | Extract tables | `SPLSection.GetTables()` | ✅ |
| 6 | Classify table types | `TableClassifier.ClassifyTable()` | ✅ |
| 7 | Route sections to KBs | `SectionRouter.RouteDocument()` | ✅ |
| 8 | Store to source_documents | `StorageManager.SaveDocument()` | ✅ |
| 9 | Store to source_sections | `StorageManager.SaveRoutedSections()` | ✅ |
| 10 | Delta sync | `DeltaSyncer.Sync()` | ✅ |
| 11 | Version history | `SPLFetcher.GetVersionHistory()` | ✅ |
| 12 | Error handling | Retry logic with exponential backoff | ✅ |

---

## Component Architecture (as specified)

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         DAILYMED SPL FETCHER                                     │
│                    shared/datasources/dailymed/                                  │
├─────────────────────────────────────────────────────────────────────────────────┤
│  PUBLIC INTERFACE (fetcher.go)                                                   │
│  ├── FetchBySetID(ctx, setID) → (*SPLDocument, error)       ✅                  │
│  ├── FetchByRxCUI(ctx, rxcui) → (*SPLDocument, error)       ✅                  │
│  ├── FetchByNDC(ctx, ndc) → (*SPLDocument, error)           ✅ NEW              │
│  ├── FetchUpdates(ctx, since) → ([]*SPLDocument, error)     ✅ NEW              │
│  ├── GetVersionHistory(ctx, setID) → ([]VersionInfo, error) ✅ NEW              │
│  └── FetchSpecificVersion(ctx, setID, ver) → (*SPLDocument) ✅ NEW              │
├─────────────────────────────────────────────────────────────────────────────────┤
│  SECTION ROUTER (section_router.go)                          ✅ NEW             │
│  ├── RouteDocument() → []*RoutedSection                                         │
│  ├── RouteSection() → *RoutedSection                                            │
│  └── DefaultRoutingMap (Truth Sourcing Manifest)                                │
├─────────────────────────────────────────────────────────────────────────────────┤
│  TABLE CLASSIFIER (table_classifier.go)                      ✅ NEW             │
│  ├── ClassifyTable() → *ClassificationResult                                    │
│  ├── ExtractAndClassifyTables() → []*ExtractedTable                             │
│  └── Table Types: GFR_DOSING, HEPATIC_DOSING, DDI, PK, ADVERSE_EVENTS          │
├─────────────────────────────────────────────────────────────────────────────────┤
│  STORAGE MANAGER (storage.go)                                ✅ NEW             │
│  ├── SaveDocument() → *SourceDocument                                           │
│  ├── SaveRoutedSections()                                                       │
│  ├── GetDocumentBySetID/RxCUI()                                                 │
│  ├── GetSectionsByLOINC/KB()                                                    │
│  └── CreateTables() (migrations)                                                │
├─────────────────────────────────────────────────────────────────────────────────┤
│  DELTA SYNCER (delta_syncer.go)                              ✅ NEW             │
│  ├── Sync() → *SyncResult                                                       │
│  ├── SyncBySetID/RxCUI/NDC()                                                    │
│  ├── RetryFailed()                                                              │
│  └── ScheduledSync (daily/weekly)                                               │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## LOINC Section Routing (Truth Sourcing Manifest)

| LOINC | Section | Target KBs | Priority |
|-------|---------|------------|----------|
| 34066-1 | BOXED WARNING | KB-4 | P0_CRITICAL |
| 34068-7 | DOSAGE AND ADMINISTRATION | KB-1, KB-4 | P0_CRITICAL |
| 34070-3 | CONTRAINDICATIONS | KB-4 | P0_CRITICAL |
| 43685-7 | WARNINGS AND PRECAUTIONS | KB-4, KB-16 | P0_CRITICAL |
| 34073-7 | DRUG INTERACTIONS | KB-5 | P1_HIGH |
| 34084-4 | ADVERSE REACTIONS | KB-4 | P1_HIGH |
| 34077-8 | PREGNANCY | KB-4 | P1_HIGH |
| 34080-2 | NURSING MOTHERS | KB-4 (→LactMed) | P1_HIGH |
| 34081-0 | PEDIATRIC USE | KB-1, KB-4 | P1_HIGH |
| 34082-8 | GERIATRIC USE | KB-4 (→Beers) | P1_HIGH |
| 34090-1 | CLINICAL PHARMACOLOGY | KB-1 (→DrugBank) | P2_MEDIUM |
| 42232-0 | RENAL IMPAIRMENT | KB-1 | P0_CRITICAL |
| 42229-5 | HEPATIC IMPAIRMENT | KB-1 (→LiverTox) | P0_CRITICAL |

---

## Table Classification Types

| Type | Description | Target KB | Detection Keywords |
|------|-------------|-----------|-------------------|
| GFR_DOSING | Renal dose adjustments | KB-1 | creatinine clearance, crcl, egfr, ml/min |
| HEPATIC_DOSING | Child-Pugh dosing | KB-1 | child-pugh, hepatic impairment, cirrhosis |
| DDI | Drug-drug interactions | KB-5 | interacting drug, concomitant, cyp, inhibitor |
| PK_PARAMETERS | Pharmacokinetic data | KB-1 | half-life, auc, cmax, clearance |
| ADVERSE_EVENTS | AE incidence tables | KB-4 | adverse reaction, incidence, placebo |
| CONTRAINDICATIONS | Contraindication lists | KB-4 | contraindicated, do not use, hypersensitivity |

---

## Database Schema

### source_documents Table
```sql
CREATE TABLE source_documents (
    id UUID PRIMARY KEY,
    set_id VARCHAR(100) NOT NULL,
    spl_version INTEGER NOT NULL,
    title VARCHAR(1000),
    drug_name VARCHAR(500),
    rxcui VARCHAR(20),
    ndc_codes TEXT[],
    raw_xml_path TEXT NOT NULL,
    raw_xml_hash VARCHAR(64) NOT NULL,
    status VARCHAR(20) DEFAULT 'FETCHED',
    UNIQUE(set_id, spl_version)
);
```

### source_sections Table
```sql
CREATE TABLE source_sections (
    id UUID PRIMARY KEY,
    source_document_id UUID REFERENCES source_documents(id),
    loinc_code VARCHAR(20) NOT NULL,
    section_title VARCHAR(500),
    has_tables BOOLEAN DEFAULT FALSE,
    tables_json JSONB,
    target_kbs TEXT[] NOT NULL,
    extraction_priority VARCHAR(20)
);
```

---

## Sync Strategies

| Strategy | Behavior | Use Case |
|----------|----------|----------|
| DAILY | Fetch docs published since yesterday | Production daily sync |
| WEEKLY | Fetch docs from last 7 days | Weekly catchup |
| MONTHLY | Full hash comparison | Monthly verification |
| FULL | Re-download everything | Initial load, recovery |

---

## Integration Points

### With Phase 3b Ground Truth Sources
- Sections with `RequiresAuthority` routing go to definitive sources:
  - NURSING MOTHERS → LactMed (for RID%)
  - GERIATRIC USE → Beers/STOPP Criteria
  - CLINICAL PHARMACOLOGY → DrugBank (for PK)
  - HEPATIC IMPAIRMENT → LiverTox

### With Phase 3c Consensus Grid
- Extracted tables feed into LLM extraction pipeline
- Table classification determines which extraction schema to apply
- GFR_DOSING → Renal Dose Schema
- HEPATIC_DOSING → Hepatic Dose Schema
- DDI → Drug Interaction Schema

---

## Ready for Phase 3b: Ground Truth Ingestion ✅

The DailyMed SPL Fetcher is complete and ready to provide source data for:
- KB-1 (Drug Rules): Dosing tables, renal/hepatic adjustments
- KB-4 (Patient Safety): Warnings, contraindications, adverse events
- KB-5 (Drug Interactions): DDI tables
- KB-6 (Formulary): How supplied information
- KB-16 (Lab Interpretation): Warning thresholds

All sections are now parsed, routed, and stored for downstream KB extractors to process.
