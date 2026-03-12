# KB-1 Drug Dosing Service: Governance Enhancement Proposal

## Executive Summary

**Current State**: KB-1 is a high-performance dosing engine with hardcoded rules, but lacks regulatory provenance, jurisdiction awareness, and audit trails.

**Target State**: A **Dynamic, Jurisdiction-Aware, Regulator-Governed Drug Registry** that makes every dose computation defensible.

**Risk Level**: 🔴 **CRITICAL** - KB-1 computes doses that get administered. Errors can kill.

---

## 1. Architecture Transformation

### Before (Current KB-1)
```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        KB-1 CURRENT STATE                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   rules.go                                                                  │
│   ├── createMetforminRule()      ← Hardcoded in Go                         │
│   ├── createWarfarinRule()       ← No source attribution                   │
│   ├── createVancomycinRule()     ← No jurisdiction                         │
│   └── ...24 drugs                ← No version control                      │
│                                                                             │
│   Problems:                                                                 │
│   ❌ Where did "Max 2000mg" come from?                                     │
│   ❌ Which regulator approved this?                                        │
│   ❌ Is this valid in Australia or only US?                                │
│   ❌ When was this rule last updated?                                      │
│   ❌ Who approved this for production?                                     │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### After (Governed KB-1)
```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        KB-1 TARGET STATE                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   kb1-knowledge/                                                            │
│   ├── us/                        ← FDA-governed rules                       │
│   │   ├── metformin.yaml                                                   │
│   │   ├── warfarin.yaml                                                    │
│   │   └── vancomycin.yaml                                                  │
│   ├── au/                        ← TGA-governed rules                       │
│   │   ├── metformin.yaml                                                   │
│   │   └── warfarin.yaml                                                    │
│   ├── in/                        ← CDSCO-governed rules                     │
│   │   └── metformin.yaml                                                   │
│   └── _compendia/                ← Licensed data (Lexicomp, AMH, etc.)     │
│                                                                             │
│   Every rule answers:                                                       │
│   ✅ "Max 2000mg" → FDA Metformin PI, Section 2.1                          │
│   ✅ Approved by FDA, effective 2024-01                                    │
│   ✅ Valid for US jurisdiction only                                        │
│   ✅ Version 2025.1, approved by CMO on 2025-01-15                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 2. Governance Schema

### 2.1 Core Drug Dosing Rule Schema

```yaml
# Schema version: 1.0.0
# This schema governs all drug dosing rules in KB-1

drug:
  rxnormCode: string          # RxNorm CUI (primary identifier)
  name: string                # Brand/Generic name
  genericName: string         # INN/USAN name
  drugClass: string           # Therapeutic class
  atcCode: string             # WHO ATC classification (optional)

dosing:
  primaryMethod: enum         # FIXED, WEIGHT_BASED, BSA_BASED, TITRATION
  
  adult:
    standard:
      - indication: string
        route: enum
        dose: number
        unit: enum
        frequency: enum
        notes: string
    
    maxDaily: number
    maxSingle: number
    maxUnit: enum
  
  pediatric:
    ageRanges:
      - ageGroup: enum
        minMonths: number
        maxMonths: number
        dosePerKg: number
        maxDose: number
  
  geriatric:
    startLow: boolean
    doseReduction: number
    beersListStatus: string
  
  renal:
    adjustmentBasis: enum     # CrCl, eGFR
    adjustments:
      - minCrCl: number
        maxCrCl: number
        dosePercent: number
        avoid: boolean
        notes: string
  
  hepatic:
    childPughA: object
    childPughB: object
    childPughC: object

safety:
  highAlertDrug: boolean
  narrowTherapeuticIndex: boolean
  blackBoxWarning: boolean
  blackBoxText: string
  monitoring: list[string]
  
governance:                   # ← THE CRITICAL ADDITION
  authority: enum             # FDA, TGA, CDSCO, EMA, NICE
  document: string            # Official document name
  section: string             # Specific section reference
  url: string                 # Link to source
  jurisdiction: enum          # US, AU, IN, EU, UK, GLOBAL
  evidenceLevel: enum         # A, B, C, D, Expert
  effectiveDate: date         # When this became effective
  expirationDate: date        # When to re-review (optional)
  version: string             # Internal version (e.g., 2025.1)
  approvedBy: string          # CMO / Clinical Pharmacist
  approvedAt: datetime        # Approval timestamp
  lastReviewedAt: datetime    # Last clinical review
  nextReviewDue: date         # When to review again
  changeLog:
    - date: date
      change: string
      reviewer: string
```

---

## 3. Data Source Hierarchy

### 3.1 Primary Regulatory Sources (Non-Negotiable)

| Jurisdiction | Authority | Source | Format | Priority |
|-------------|-----------|--------|--------|----------|
| **US** | FDA | DailyMed SPL | XML | 1 |
| **Australia** | TGA | Product Information | PDF | 1 |
| **India** | CDSCO | Package Inserts | PDF | 1 |
| **EU** | EMA | SmPC | PDF/XML | 1 |
| **UK** | MHRA/NICE | SmPC + Guidelines | PDF | 1 |
| **Global** | WHO | Essential Medicines | PDF | 2 |

### 3.2 Clinical Practice Guidelines (Secondary)

| Domain | Authority | When to Use |
|--------|-----------|-------------|
| Antibiotics | IDSA | When FDA label lacks specifics |
| Anticoagulation | ACCP/CHEST | Warfarin dosing protocols |
| Heart Failure | ACC/AHA | Titration schedules |
| Diabetes | ADA | A1c targets, titration |
| CKD Dosing | KDIGO | Renal adjustment specifics |
| Chemotherapy | NCCN/ASCO | BSA-based protocols |
| Pediatrics | Harriet Lane / BNF-C | Age/weight dosing |

### 3.3 Drug Information Compendia (Operational)

| Compendium | Coverage | Integration Priority |
|------------|----------|---------------------|
| **Lexicomp** | Global, comprehensive | Future (licensed) |
| **Micromedex** | Global, evidence-based | Future (licensed) |
| **BNF** | UK + international | Future (licensed) |
| **AMH** | Australia-specific | Future (licensed) |
| **NFI** | India-specific | Future (licensed) |

**Structure must support these from day one, even if not licensed yet.**

---

## 4. Ingestion Pipeline Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    KB-1 DRUG REGISTRY INGESTION PIPELINE                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    LAYER 1: IDENTITY (RxNorm Spine)                  │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │   RxNorm → Ingredient → Brand → Formulation → Strength              │   │
│  │                                                                     │   │
│  │   Example:                                                          │   │
│  │   6809 → Metformin → Glucophage → Tablet → 500mg, 850mg, 1000mg    │   │
│  │                                                                     │   │
│  │   Source: NLM RxNorm (weekly updates via KB-7)                     │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    ↓                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    LAYER 2: REGULATORY DATA                          │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐              │   │
│  │   │  FDA        │   │  TGA        │   │  CDSCO      │              │   │
│  │   │  DailyMed   │   │  PI PDFs    │   │  PI PDFs    │              │   │
│  │   │  SPL XML    │   │  (parsed)   │   │  (parsed)   │              │   │
│  │   └──────┬──────┘   └──────┬──────┘   └──────┬──────┘              │   │
│  │          │                 │                 │                      │   │
│  │          ▼                 ▼                 ▼                      │   │
│  │   ┌─────────────────────────────────────────────────────────────┐  │   │
│  │   │              DOSING DATA EXTRACTION                          │  │   │
│  │   │                                                              │  │   │
│  │   │   Section 2: Dosage and Administration                      │  │   │
│  │   │   Section 4: Contraindications                              │  │   │
│  │   │   Section 5: Warnings (Black Box)                           │  │   │
│  │   │   Section 8: Use in Specific Populations                    │  │   │
│  │   │   Section 12: Clinical Pharmacology (renal/hepatic)         │  │   │
│  │   │                                                              │  │   │
│  │   └─────────────────────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    ↓                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    LAYER 3: NORMALIZATION                            │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │   FDA format     → Normalized YAML (us/metformin.yaml)             │   │
│  │   TGA format     → Normalized YAML (au/metformin.yaml)             │   │
│  │   CDSCO format   → Normalized YAML (in/metformin.yaml)             │   │
│  │                                                                     │   │
│  │   Same schema. Different content per jurisdiction.                 │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    ↓                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    LAYER 4: CLINICAL REVIEW                          │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │   1. Auto-extracted rule flagged for review                        │   │
│  │   2. Clinical Pharmacist validates dosing accuracy                 │   │
│  │   3. CMO approves for production                                   │   │
│  │   4. Governance metadata signed                                    │   │
│  │   5. Rule goes live in KB-1                                        │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                    ↓                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    LAYER 5: KB-1 RUNTIME                             │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │   Request: "Dose Metformin for 65yo Australian with eGFR 35"       │   │
│  │                                                                     │   │
│  │   1. Load au/metformin.yaml                                        │   │
│  │   2. Apply renal adjustment (TGA guidance)                         │   │
│  │   3. Return dose + full provenance                                 │   │
│  │                                                                     │   │
│  │   Response:                                                        │   │
│  │   {                                                                │   │
│  │     "dose": 500,                                                   │   │
│  │     "unit": "mg",                                                  │   │
│  │     "frequency": "BID",                                            │   │
│  │     "maxDaily": 1000,                                              │   │
│  │     "source": {                                                    │   │
│  │       "authority": "TGA",                                          │   │
│  │       "document": "Glucophage Product Information",                │   │
│  │       "section": "4.2 Dose and method of administration",          │   │
│  │       "jurisdiction": "AU"                                         │   │
│  │     }                                                              │   │
│  │   }                                                                │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 5. Implementation Phases

### Phase 1: Foundation (Weeks 1-2)
- [ ] Define governance schema (YAML)
- [ ] Create kb1-knowledge directory structure
- [ ] Convert top 25 hardcoded drugs to governed YAML
- [ ] Implement YAML loader in Go

### Phase 2: US Coverage (Weeks 3-4)
- [ ] Build FDA DailyMed SPL parser
- [ ] Extract dosing from Section 2 (Dosage and Administration)
- [ ] Extract warnings from Section 5 (Black Box)
- [ ] Expand to 100 governed drugs (US)

### Phase 3: Australia Coverage (Weeks 5-6)
- [ ] Build TGA PI PDF parser
- [ ] Map TGA sections to schema
- [ ] Create au/ jurisdiction rules
- [ ] Integrate with KB-7 SNOMED CT-AU

### Phase 4: India Coverage (Weeks 7-8)
- [ ] Build CDSCO parser
- [ ] Handle brand name mapping (via CDCI)
- [ ] Create in/ jurisdiction rules
- [ ] Integrate with KB-7 India brand mapping

### Phase 5: Compendia Preparation (Weeks 9-10)
- [ ] Design Lexicomp/Micromedex adapter interface
- [ ] Create _compendia/ placeholder structure
- [ ] Document licensing requirements
- [ ] Build conflict resolution logic (regulatory > compendia)

---

## 6. Deliverables

| Deliverable | Description | Status |
|-------------|-------------|--------|
| `governance_schema.yaml` | Full schema definition | Included below |
| `kb1-knowledge/` | Governed drug directory | Included below |
| `us/warfarin.yaml` | Sample FDA-governed drug | Included below |
| `au/warfarin.yaml` | Sample TGA-governed drug | Included below |
| `in/metformin.yaml` | Sample CDSCO-governed drug | Included below |
| `loader.go` | YAML rule loader | Included below |
| `fda_ingestion.go` | FDA DailyMed pipeline | Included below |
| `validation_pack.go` | Dosing validation tests | Included below |

---

## 7. Risk Mitigation

| Risk | Mitigation |
|------|------------|
| Wrong dose computed | Every dose links to regulatory source |
| Jurisdiction mismatch | Explicit jurisdiction field, runtime check |
| Stale data | effectiveDate, nextReviewDue fields |
| Unapproved rule | approvedBy, approvedAt required fields |
| Audit failure | Full governance trail in every response |
| Compendia conflict | Regulatory source always wins |

---

## 8. Success Criteria

- [ ] 100% of dosing rules have governance metadata
- [ ] Every dose response includes source attribution
- [ ] Jurisdiction-aware routing (US patient → US rules)
- [ ] CMO approval workflow implemented
- [ ] FDA/TGA/CDSCO coverage for top 100 drugs
- [ ] Validation pack passes with 100% coverage

---

## 9. Approval

| Role | Name | Approval | Date |
|------|------|----------|------|
| CTO | | ☐ Pending | |
| CMO | | ☐ Pending | |
| Clinical Pharmacist | | ☐ Pending | |
| Engineering Lead | | ☐ Pending | |

---

**Next Step**: Review and approve this proposal, then proceed to implementation.
