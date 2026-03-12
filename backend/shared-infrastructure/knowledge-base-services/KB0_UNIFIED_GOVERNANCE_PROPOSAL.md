# KB Unified Governance Infrastructure

## Executive Summary

**Problem**: We're building approval workflows for 19 Knowledge Bases. Without shared infrastructure, we'll repeat:
- Ingestion pipeline code (FDA, TGA, CMS, SNOMED, etc.)
- Approval workflow state machines
- Pharmacist/CMO dashboard UIs
- Audit logging systems
- Version management

**Solution**: A **Unified Clinical Knowledge Governance Platform (KB-0)** that provides:
1. Common ingestion pipeline framework
2. Shared approval workflow engine
3. Unified dashboard for all knowledge types
4. Cross-KB audit and compliance reporting

---

## 1. KB Analysis: What Needs Governance?

### Classification of KBs by Governance Needs

| KB | Service | Ingestion Source | Clinical Review? | Approval Level | Risk |
|----|---------|------------------|------------------|----------------|------|
| **KB-1** | Drug Dosing | FDA, TGA, CDSCO | ✅ Pharmacist | CMO | 🔴 HIGH |
| **KB-4** | Patient Safety | Literature, FDA | ✅ Pharmacist | CMO | 🔴 HIGH |
| **KB-5** | Drug Interactions | FDA, Lexicomp | ✅ Pharmacist | CMO | 🔴 HIGH |
| **KB-12** | Order Sets | Clinical protocols | ✅ Physician | CMO | 🔴 HIGH |
| **KB-19** | Protocol Orchestrator | Guidelines (IDSA, ACC) | ✅ Specialist | CMO | 🔴 HIGH |
| **KB-6** | Formulary | Hospital P&T, NLEM | ✅ Pharmacist | P&T Chair | 🟡 MEDIUM |
| **KB-8** | Calculators | Literature | ✅ Physician | Clinical Lead | 🟡 MEDIUM |
| **KB-9** | Care Gaps | CMS eCQM | ✅ Quality Team | Quality Dir | 🟡 MEDIUM |
| **KB-13** | Quality Measures | CMS, HEDIS | ✅ Quality Team | Quality Dir | 🟡 MEDIUM |
| **KB-16** | Lab Interpretation | Lab reference | ✅ Pathologist | Lab Dir | 🟡 MEDIUM |
| **KB-15** | Evidence Engine | PubMed, Cochrane | ✅ Physician | Clinical Lead | 🟡 MEDIUM |
| **KB-7** | Terminology | NLM, SNOMED, NCTS | ⚠️ Automated + Spot | Terminology Mgr | 🟢 LOW |
| **KB-3** | Temporal Logic | Clinical logic | ⚠️ Engineering | Tech Lead | 🟢 LOW |
| **KB-10** | Rules Engine | Internal logic | ⚠️ Engineering | Tech Lead | 🟢 LOW |
| **KB-2** | Clinical Context | FHIR profiles | ⚠️ Engineering | Tech Lead | 🟢 LOW |
| **KB-11** | Population Health | Analytics config | ⚠️ Analyst | Analytics Lead | 🟢 LOW |
| **KB-14** | Care Navigator | Workflow config | ⚠️ Clinical Ops | Ops Lead | 🟢 LOW |
| **KB-17** | Population Registry | Analytics config | ⚠️ Analyst | Analytics Lead | 🟢 LOW |
| **KB-18** | Governance Engine | Meta-governance | ⚠️ Compliance | Compliance | 🟢 LOW |

### Summary

| Category | KBs | Clinical Review | Risk Level |
|----------|-----|-----------------|------------|
| **Dosing & Safety** | KB-1, KB-4, KB-5, KB-12, KB-19 | Full pharmacist/physician + CMO | 🔴 HIGH |
| **Quality & Evidence** | KB-6, KB-8, KB-9, KB-13, KB-15, KB-16 | Specialist review | 🟡 MEDIUM |
| **Infrastructure** | KB-2, KB-3, KB-7, KB-10, KB-11, KB-14, KB-17, KB-18 | Automated + spot check | 🟢 LOW |

---

## 2. Common Ingestion Sources

### Regulatory Sources (Shared Across KBs)

| Source | Format | KBs Using | Frequency |
|--------|--------|-----------|-----------|
| **FDA DailyMed SPL** | XML | KB-1, KB-4, KB-5 | Daily |
| **TGA Product Info** | PDF | KB-1, KB-4, KB-5, KB-6 | Monthly |
| **CDSCO Package Inserts** | PDF | KB-1, KB-4, KB-5, KB-6 | Monthly |
| **EMA SmPC** | PDF/XML | KB-1, KB-4, KB-5 | Monthly |
| **CMS eCQM** | CQL/ELM | KB-9, KB-13 | Annually |
| **HEDIS Measures** | PDF/Spec | KB-9, KB-13 | Annually |
| **NLM RxNorm** | RRF | KB-1, KB-5, KB-6, KB-7 | Weekly |
| **SNOMED CT** | RF2 | KB-7 (all consume) | Biannual |
| **SNOMED CT-AU (NCTS)** | RF2 | KB-7 (AU region) | Monthly |
| **ICD-10-CM/PCS** | Flat files | KB-7, KB-13 | Annually |
| **LOINC** | CSV | KB-7, KB-16 | Biannual |
| **Clinical Guidelines** | PDF/HTML | KB-15, KB-19 | Variable |

### Insight: 70% of ingestion code is reusable

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         INGESTION CODE REUSE                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Shared (70%)                        KB-Specific (30%)                     │
│   ────────────                        ─────────────────                     │
│   • FDA SPL XML parser                • Dosing extraction (KB-1)           │
│   • TGA PDF parser                    • Interaction extraction (KB-5)      │
│   • CMS eCQM CQL loader               • Safety rule extraction (KB-4)      │
│   • SNOMED RF2 loader                 • Order set templates (KB-12)        │
│   • RxNorm RRF loader                 • Quality measure mapping (KB-9)     │
│   • Version management                • Lab range extraction (KB-16)       │
│   • Hash/integrity checking                                                │
│   • Audit logging                                                          │
│   • Change detection                                                       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Common Approval Workflows

### Workflow Patterns Across KBs

| Pattern | KBs | Reviewers | Approver | Duration |
|---------|-----|-----------|----------|----------|
| **Drug Safety** | KB-1, KB-4, KB-5 | 2× Pharmacist | CMO | 24-72h |
| **Clinical Protocol** | KB-12, KB-19 | Specialist + Pharmacist | CMO | 48-96h |
| **Quality Measure** | KB-9, KB-13 | Quality Analyst | Quality Director | 24-48h |
| **Evidence Review** | KB-8, KB-15, KB-16 | Subject Expert | Clinical Lead | 24-48h |
| **Formulary** | KB-6 | Pharmacist | P&T Chair | 48h-2wk |
| **Terminology** | KB-7 | Automated | Terminology Manager | Automated |
| **Infrastructure** | Others | Engineering | Tech Lead | 24h |

### Insight: 3 Workflow Templates Cover 90% of Cases

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      WORKFLOW TEMPLATE REUSE                                 │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   Template A: HIGH-RISK CLINICAL                                            │
│   ─────────────────────────────                                             │
│   DRAFT → REVIEW(1) → REVIEW(2) → CMO → ACTIVE                             │
│   Used by: KB-1, KB-4, KB-5, KB-12, KB-19                                  │
│                                                                             │
│   Template B: MEDIUM-RISK QUALITY                                           │
│   ────────────────────────────                                              │
│   DRAFT → REVIEW(1) → DIRECTOR → ACTIVE                                    │
│   Used by: KB-6, KB-8, KB-9, KB-13, KB-15, KB-16                           │
│                                                                             │
│   Template C: LOW-RISK INFRASTRUCTURE                                       │
│   ──────────────────────────────────                                        │
│   DRAFT → AUTO-VALIDATE → LEAD → ACTIVE                                    │
│   Used by: KB-2, KB-3, KB-7, KB-10, KB-11, KB-14, KB-17, KB-18             │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Proposed Architecture: KB-0 Governance Platform

### Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         KB-0: GOVERNANCE PLATFORM                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                    INGESTION FRAMEWORK                               │  │
│   ├─────────────────────────────────────────────────────────────────────┤  │
│   │                                                                     │  │
│   │   ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐ │  │
│   │   │  FDA    │  │  TGA    │  │  CMS    │  │ SNOMED  │  │  LOINC  │ │  │
│   │   │ Adapter │  │ Adapter │  │ Adapter │  │ Adapter │  │ Adapter │ │  │
│   │   └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘ │  │
│   │        │            │            │            │            │      │  │
│   │        └────────────┴────────────┼────────────┴────────────┘      │  │
│   │                                  ▼                                 │  │
│   │   ┌─────────────────────────────────────────────────────────────┐ │  │
│   │   │              CANONICAL KNOWLEDGE ITEM                        │ │  │
│   │   │  (source, hash, type, jurisdiction, version, content)       │ │  │
│   │   └─────────────────────────────────────────────────────────────┘ │  │
│   │                                                                     │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                    │                                        │
│                                    ▼                                        │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                    WORKFLOW ENGINE                                   │  │
│   ├─────────────────────────────────────────────────────────────────────┤  │
│   │                                                                     │  │
│   │   ┌─────────────┐  ┌─────────────┐  ┌─────────────┐               │  │
│   │   │  Template A │  │  Template B │  │  Template C │               │  │
│   │   │  High-Risk  │  │  Med-Risk   │  │  Low-Risk   │               │  │
│   │   │  Clinical   │  │  Quality    │  │  Infra      │               │  │
│   │   └─────────────┘  └─────────────┘  └─────────────┘               │  │
│   │                                                                     │  │
│   │   State Machine: DRAFT → REVIEW → APPROVED → ACTIVE → RETIRED     │  │
│   │                                                                     │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                    │                                        │
│                                    ▼                                        │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                    UNIFIED DASHBOARD                                 │  │
│   ├─────────────────────────────────────────────────────────────────────┤  │
│   │                                                                     │  │
│   │   ┌───────────────┐  ┌───────────────┐  ┌───────────────┐         │  │
│   │   │  Pharmacist   │  │  Physician    │  │  CMO          │         │  │
│   │   │  Dashboard    │  │  Dashboard    │  │  Dashboard    │         │  │
│   │   └───────────────┘  └───────────────┘  └───────────────┘         │  │
│   │                                                                     │  │
│   │   ┌───────────────┐  ┌───────────────┐  ┌───────────────┐         │  │
│   │   │  Quality Dir  │  │  Compliance   │  │  Engineering  │         │  │
│   │   │  Dashboard    │  │  Dashboard    │  │  Dashboard    │         │  │
│   │   └───────────────┘  └───────────────┘  └───────────────┘         │  │
│   │                                                                     │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                    │                                        │
│                                    ▼                                        │
│   ┌─────────────────────────────────────────────────────────────────────┐  │
│   │                    AUDIT & COMPLIANCE                                │  │
│   ├─────────────────────────────────────────────────────────────────────┤  │
│   │                                                                     │  │
│   │   • Immutable audit log (PostgreSQL + append-only)                 │  │
│   │   • Cross-KB compliance reporting                                  │  │
│   │   • Regulatory export (FDA, TGA, CMS audit formats)               │  │
│   │   • Retention management (10+ years)                               │  │
│   │                                                                     │  │
│   └─────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 5. KB-0 Component Design

### 5.1 Knowledge Item (Universal Schema)

```yaml
# Every piece of governed knowledge across all KBs follows this schema
KnowledgeItem:
  # Identity
  id: string                    # Unique ID (e.g., "kb1:warfarin:us:2025.1")
  kb: enum                      # KB-1 through KB-19
  type: enum                    # DOSING_RULE, SAFETY_ALERT, INTERACTION, MEASURE, etc.
  
  # Content reference
  contentRef: string            # Pointer to KB-specific content (YAML, CQL, JSON)
  contentHash: string           # SHA256 for integrity
  
  # Source attribution
  source:
    authority: enum             # FDA, TGA, CMS, SNOMED, IDSA, etc.
    document: string
    section: string
    url: string
    jurisdiction: enum          # US, AU, IN, EU, GLOBAL
    effectiveDate: date
  
  # Classification
  riskLevel: enum               # HIGH, MEDIUM, LOW
  workflowTemplate: enum        # CLINICAL_HIGH, QUALITY_MED, INFRA_LOW
  requiresDualReview: boolean
  
  # State
  state: enum                   # DRAFT, REVIEWED, APPROVED, ACTIVE, etc.
  version: string
  
  # Governance trail (populated by workflow)
  governance:
    createdAt: datetime
    createdBy: string
    reviews: list[Review]
    approval: Approval
    activatedAt: datetime
    retiredAt: datetime
```

### 5.2 Ingestion Adapter Interface

```go
// Every source adapter implements this interface
type IngestionAdapter interface {
    // Metadata
    GetName() string
    GetAuthority() Authority
    GetSupportedKBs() []KB
    
    // Discovery
    CheckForUpdates(ctx context.Context) ([]UpdateInfo, error)
    
    // Ingestion
    Fetch(ctx context.Context, itemID string) ([]byte, error)
    Parse(ctx context.Context, data []byte) (*RawContent, error)
    
    // Transformation (KB-specific)
    Transform(ctx context.Context, raw *RawContent, targetKB KB) (*KnowledgeItem, error)
    
    // Validation
    Validate(ctx context.Context, item *KnowledgeItem) ([]ValidationError, error)
}

// Shared adapters
type FDADailyMedAdapter struct { ... }  // Used by KB-1, KB-4, KB-5
type TGAProductInfoAdapter struct { ... } // Used by KB-1, KB-4, KB-5, KB-6
type CMSeCQMAdapter struct { ... }        // Used by KB-9, KB-13
type SNOMEDRFAdapter struct { ... }       // Used by KB-7 (all KBs consume)
type RxNormAdapter struct { ... }         // Used by KB-1, KB-5, KB-6, KB-7
type LOINCAdapter struct { ... }          // Used by KB-7, KB-16
```

### 5.3 Workflow Template Definition

```yaml
# Template A: High-Risk Clinical (KB-1, KB-4, KB-5, KB-12, KB-19)
WorkflowTemplate:
  id: CLINICAL_HIGH
  name: "High-Risk Clinical Content"
  
  states:
    - DRAFT
    - PRIMARY_REVIEW
    - SECONDARY_REVIEW
    - CMO_APPROVAL
    - APPROVED
    - ACTIVE
    - HOLD
    - RETIRED
    - REJECTED
  
  transitions:
    - from: DRAFT
      to: PRIMARY_REVIEW
      actor: pharmacist|physician
      action: submit_review
    
    - from: PRIMARY_REVIEW
      to: SECONDARY_REVIEW
      actor: pharmacist|physician
      condition: is_high_risk
      action: submit_review
    
    - from: [PRIMARY_REVIEW, SECONDARY_REVIEW]
      to: CMO_APPROVAL
      actor: system
      condition: all_reviews_complete
    
    - from: CMO_APPROVAL
      to: APPROVED
      actor: cmo
      action: approve
      requires:
        - attestation: medical_responsibility
        - attestation: clinical_standards
    
    - from: APPROVED
      to: ACTIVE
      actor: system
      action: activate
  
  reviewChecklist:
    - id: dose_verification
      label: "Dose verified against regulatory label"
      required: true
    - id: renal_adjustment
      label: "Renal adjustments verified"
      required: true
    - id: hepatic_adjustment
      label: "Hepatic adjustments verified"
      required: true
    - id: interactions_checked
      label: "Drug interactions reviewed"
      required: true
    - id: black_box_confirmed
      label: "Black box warnings confirmed"
      required: when(has_black_box)
  
  sla:
    review_target: 24h
    approval_target: 48h
    escalation_after: 72h

---

# Template B: Medium-Risk Quality (KB-6, KB-8, KB-9, KB-13, KB-15, KB-16)
WorkflowTemplate:
  id: QUALITY_MED
  name: "Medium-Risk Quality Content"
  
  states:
    - DRAFT
    - REVIEW
    - DIRECTOR_APPROVAL
    - APPROVED
    - ACTIVE
    - RETIRED
  
  transitions:
    - from: DRAFT
      to: REVIEW
      actor: quality_analyst|specialist
      action: submit_review
    
    - from: REVIEW
      to: DIRECTOR_APPROVAL
      actor: system
      condition: review_complete
    
    - from: DIRECTOR_APPROVAL
      to: APPROVED
      actor: quality_director|clinical_lead
      action: approve
    
    - from: APPROVED
      to: ACTIVE
      actor: system
      action: activate
  
  reviewChecklist:
    - id: content_accuracy
      label: "Content accuracy verified"
      required: true
    - id: source_validated
      label: "Source document validated"
      required: true
    - id: jurisdiction_appropriate
      label: "Jurisdiction appropriateness confirmed"
      required: true
  
  sla:
    review_target: 24h
    approval_target: 24h

---

# Template C: Low-Risk Infrastructure (KB-2, KB-3, KB-7, KB-10, KB-11, KB-14, KB-17, KB-18)
WorkflowTemplate:
  id: INFRA_LOW
  name: "Low-Risk Infrastructure Content"
  
  states:
    - DRAFT
    - AUTO_VALIDATION
    - LEAD_APPROVAL
    - ACTIVE
    - RETIRED
  
  transitions:
    - from: DRAFT
      to: AUTO_VALIDATION
      actor: system
      action: auto_validate
    
    - from: AUTO_VALIDATION
      to: LEAD_APPROVAL
      actor: system
      condition: validation_passed
    
    - from: LEAD_APPROVAL
      to: ACTIVE
      actor: tech_lead|terminology_manager
      action: approve
  
  autoValidation:
    - schema_validation
    - reference_integrity
    - version_compatibility
    - regression_tests
  
  sla:
    validation_target: 1h
    approval_target: 24h
```

---

## 6. Unified Dashboard Design

### 6.1 Role-Based Views

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     UNIFIED GOVERNANCE DASHBOARD                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  ROLE: Clinical Pharmacist                              [Dr. Smith] │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  MY REVIEW QUEUE                                                    │   │
│  │  ───────────────                                                    │   │
│  │  ┌──────────────────────────────────────────────────────────────┐  │   │
│  │  │ 🔴 KB-1  Warfarin Dosing (US/FDA)           2h ago   HIGH   │  │   │
│  │  │ 🔴 KB-5  Warfarin-Aspirin Interaction       3h ago   HIGH   │  │   │
│  │  │ 🔴 KB-4  Heparin Safety Alert               5h ago   HIGH   │  │   │
│  │  │ 🟡 KB-6  Formulary Addition: Ozempic        1d ago   MED    │  │   │
│  │  │ 🟡 KB-1  Metformin Dosing (AU/TGA)          1d ago   MED    │  │   │
│  │  └──────────────────────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  │  PENDING MY SECONDARY REVIEW (Dual Review Required)                │   │
│  │  ──────────────────────────────────────────────────                │   │
│  │  ┌──────────────────────────────────────────────────────────────┐  │   │
│  │  │ 🔴 KB-1  Insulin Glargine Dosing   Primary: Dr. Jones   4h  │  │   │
│  │  └──────────────────────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  │  MY RECENT REVIEWS                                                  │   │
│  │  ─────────────────                                                  │   │
│  │  ┌──────────────────────────────────────────────────────────────┐  │   │
│  │  │ ✓ KB-1  Lisinopril Dosing       APPROVED by CMO   Yesterday │  │   │
│  │  │ ✓ KB-5  Metoprolol-Diltiazem    APPROVED by CMO   2d ago    │  │   │
│  │  │ ↺ KB-4  Opioid Safety Alert     REVISION NEEDED   2d ago    │  │   │
│  │  └──────────────────────────────────────────────────────────────┘  │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  CROSS-KB METRICS                                                   │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  │ KB-1   ████████░░░░ 12 pending │ KB-4   ███░░░░░░░░░  3 pending │   │
│  │  │ KB-5   ██████░░░░░░  8 pending │ KB-6   █████░░░░░░░  6 pending │   │
│  │  │                                                                  │   │
│  │  │ Total: 29 pending pharmacist review                             │   │
│  │  │ Avg review time: 18 hours                                       │   │
│  │  │ SLA compliance: 94%                                             │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  ROLE: Chief Medical Officer                              [Dr. Williams]    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  MY APPROVAL QUEUE                                                          │
│  ─────────────────                                                          │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │ 🔴 KB-1   Warfarin Dosing          Dual ✓   Reviewer: Smith, Jones │  │
│  │ 🔴 KB-12  Sepsis Order Set         Dual ✓   Reviewer: Patel, Lee   │  │
│  │ 🔴 KB-19  VTE Protocol Update      Dual ✓   Reviewer: Chen, Kim    │  │
│  │ 🟡 KB-6   Formulary: Ozempic       Single   Reviewer: Smith        │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  CROSS-KB EXECUTIVE SUMMARY                                                 │
│  ──────────────────────────                                                 │
│  │ Active Rules Total:     1,847                                          │
│  │ High-Risk Active:         312                                          │
│  │ Pending CMO Approval:       4                                          │
│  │ Emergency Overrides:        0                                          │
│  │ This Week's Approvals:     23                                          │
│  │ Avg Time to Approval:    6.2h                                          │
│                                                                             │
│  COMPLIANCE STATUS                                                          │
│  ─────────────────                                                          │
│  │ KB-1 Drug Dosing:      ████████████ 100% governed                      │
│  │ KB-4 Patient Safety:   ████████████ 100% governed                      │
│  │ KB-5 Interactions:     ██████████░░  92% governed                      │
│  │ KB-9 Quality Measures: ████████████ 100% governed                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 6.2 KB Filter View

```
┌─────────────────────────────────────────────────────────────────────────────┐
│  FILTER: KB-1 Drug Dosing Service                                           │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  STATUS BREAKDOWN                                                           │
│  ────────────────                                                           │
│  ACTIVE:     847 rules  ████████████████████████████████████████████       │
│  DRAFT:       23 rules  ███░                                                │
│  REVIEWED:     8 rules  █░                                                  │
│  HOLD:         3 rules  ░                                                   │
│  RETIRED:    156 rules  ████████░                                           │
│                                                                             │
│  BY JURISDICTION                                                            │
│  ───────────────                                                            │
│  US (FDA):    412 rules  ████████████████████░░░░                          │
│  AU (TGA):    287 rules  ████████████░░░░░░░░░░░                           │
│  IN (CDSCO): 148 rules  ██████░░░░░░░░░░░░░░░░░░                           │
│                                                                             │
│  BY DRUG CLASS                                                              │
│  ─────────────                                                              │
│  Cardiovascular:  234  Diabetes:      156  Antibiotics:   189              │
│  Anticoagulants:   87  Pain/Opioids:   92  Oncology:      89              │
│                                                                             │
│  RECENT ACTIVITY                                                            │
│  ───────────────                                                            │
│  Today:     Ingested: 12  Reviewed: 8   Approved: 6   Activated: 6        │
│  This Week: Ingested: 67  Reviewed: 52  Approved: 48  Activated: 48       │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Implementation Plan

### Phase 1: KB-0 Core (Weeks 1-4)

| Week | Deliverable |
|------|-------------|
| 1 | Knowledge Item schema, database design |
| 2 | Workflow engine (3 templates) |
| 3 | Unified audit system |
| 4 | Basic dashboard (role-based) |

### Phase 2: Ingestion Adapters (Weeks 5-8)

| Week | Adapter |
|------|---------|
| 5 | FDA DailyMed SPL (KB-1, KB-4, KB-5) |
| 6 | TGA + CDSCO (KB-1, KB-4, KB-5, KB-6) |
| 7 | CMS eCQM (KB-9, KB-13) |
| 8 | SNOMED + RxNorm (KB-7) |

### Phase 3: KB Onboarding (Weeks 9-12)

| Week | KBs |
|------|-----|
| 9 | KB-1 Drug Dosing (migrate from standalone) |
| 10 | KB-4 Patient Safety, KB-5 Drug Interactions |
| 11 | KB-9 Care Gaps, KB-13 Quality Measures |
| 12 | KB-6 Formulary, KB-12 Order Sets |

### Phase 4: Dashboard Enhancement (Weeks 13-16)

| Week | Feature |
|------|---------|
| 13 | Cross-KB analytics |
| 14 | Compliance reporting |
| 15 | SLA monitoring |
| 16 | Regulatory export |

---

## 8. Cost-Benefit Analysis

### Without KB-0 (Build Per-KB)

| KB | Ingestion | Workflow | Dashboard | Audit | Total |
|----|-----------|----------|-----------|-------|-------|
| KB-1 | 2 weeks | 2 weeks | 1 week | 1 week | 6 weeks |
| KB-4 | 2 weeks | 2 weeks | 1 week | 1 week | 6 weeks |
| KB-5 | 2 weeks | 2 weeks | 1 week | 1 week | 6 weeks |
| KB-6 | 1 week | 1 week | 1 week | 1 week | 4 weeks |
| KB-9 | 2 weeks | 1 week | 1 week | 1 week | 5 weeks |
| KB-12 | 1 week | 2 weeks | 1 week | 1 week | 5 weeks |
| KB-13 | 2 weeks | 1 week | 1 week | 1 week | 5 weeks |
| KB-19 | 2 weeks | 2 weeks | 1 week | 1 week | 6 weeks |
| **Total** | | | | | **43 weeks** |

### With KB-0 (Shared Infrastructure)

| Component | Effort |
|-----------|--------|
| KB-0 Core (once) | 8 weeks |
| Ingestion Adapters (shared) | 4 weeks |
| Per-KB Customization (8 KBs × 0.5 weeks) | 4 weeks |
| **Total** | **16 weeks** |

### Savings

| Metric | Value |
|--------|-------|
| **Time Saved** | 27 weeks (63%) |
| **Code Reduction** | ~70% less duplicate code |
| **Maintenance** | Single codebase for governance |
| **Consistency** | Identical audit/compliance across KBs |

---

## 9. Recommendation

### ✅ RECOMMENDED: Build KB-0 Governance Platform

**Reasons**:
1. **70% code reuse** across ingestion pipelines
2. **90% workflow reuse** with 3 templates
3. **Single dashboard** for all clinical reviewers
4. **Unified compliance** for regulators
5. **63% time savings** vs per-KB approach

**Implementation**:
1. Start KB-0 immediately (Weeks 1-4)
2. Migrate KB-1 work into KB-0 (already built)
3. Onboard high-risk KBs first (KB-4, KB-5, KB-12, KB-19)
4. Backfill medium/low-risk KBs

---

## 10. Approval

| Role | Name | Decision | Date |
|------|------|----------|------|
| CTO | | ☐ Approve KB-0 | |
| CMO | | ☐ Approve KB-0 | |
| VP Engineering | | ☐ Approve KB-0 | |
| Compliance | | ☐ Approve KB-0 | |

---

## Appendix: KB-Specific Customizations

Even with KB-0, each KB needs thin customization layers:

| KB | Customization |
|----|---------------|
| KB-1 | Dosing extraction from SPL Section 2 |
| KB-4 | Safety alert severity classification |
| KB-5 | Interaction severity/mechanism extraction |
| KB-6 | Formulary tier/PA mapping |
| KB-9 | CQL measure binding |
| KB-12 | Order set template structure |
| KB-13 | HEDIS measure mapping |
| KB-16 | Lab reference range parsing |
| KB-19 | Guideline protocol extraction |
