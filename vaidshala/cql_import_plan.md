# Vaidshala CQL Repository: Comprehensive Import Plan & Atomiser Architecture

## Document Information
| Field | Value |
|-------|-------|
| **Version** | 1.0 |
| **Date** | January 2026 |
| **Status** | STRATEGIC PLANNING |
| **Scope** | CQL Import Strategy + Atomiser Engine Design |
| **Related** | Guidelines_Ingestion_CQL_Repository_Architecture |

---

## 1. Executive Summary

This document provides a **dual-track strategy** for populating the Vaidshala CQL Repository:

1. **Track A: Direct Import** - Leverage existing authoritative CQL from CMS, CDC, WHO, and AHRQ
2. **Track B: Atomiser Pipeline** - Novel LLM-powered extraction engine to convert narrative CPG text → structured CQL

The Atomiser represents a **first-of-its-kind** approach that treats clinical practice guidelines as "multi-dimensional knowledge objects" requiring parallel extraction of:
- **Clinical Logic** (conditions, eligibility, contraindications)
- **Temporal Constraints** (deadlines, sequences, bundles)
- **Evidence Metadata** (GRADE ratings, citations, COR/LOE)

---

## 2. Track A: Direct CQL Import Strategy

### 2.1 Priority Import Sources

| Priority | Source | Content | Repository URL | Import Complexity |
|----------|--------|---------|----------------|-------------------|
| **P0** | CMS eCQM Library | 100+ quality measures with CQL | github.com/cqframework/ecqm-content-qicore-2024 | Low |
| **P0** | CDC Opioid Prescribing IG | 12 recommendations as CQL | github.com/cqframework/opioid-cds-r4 | Low |
| **P1** | WHO SMART Guidelines | HIV, Immunization, ANC, TB | github.com/WorldHealthOrganization/smart-* | Medium |
| **P1** | AHRQ CDS Connect | Pain management, CVD, Diabetes | github.com/AHRQ-CDS/* | Medium |
| **P2** | CQF Common Libraries | FHIRHelpers, QICoreCommon | github.com/cqframework/cqf-common | Low |
| **P2** | CPG-on-FHIR Examples | Reference implementations | github.com/HL7/cqf-recommendations | Medium |

### 2.2 Specific CQL Files to Import

#### 2.2.1 Foundation Libraries (Import First)

```
vaidshala/clinical-knowledge-core/
├── tier-0-foundation/
│   ├── FHIRHelpers.cql          # From: cqf-common (required by all)
│   └── FHIRCommon.cql           # From: cqf-common
├── tier-1-primitives/
│   ├── QICoreCommon.cql         # From: ecqm-content-qicore-2024
│   └── CQMCommon.cql            # From: ecqm-content-qicore-2024
```

**Import Commands:**
```bash
# Clone required repositories
git clone https://github.com/cqframework/ecqm-content-qicore-2024 /tmp/ecqm-2024
git clone https://github.com/cqframework/cqf-common /tmp/cqf-common
git clone https://github.com/cqframework/opioid-cds-r4 /tmp/opioid-cds

# Copy foundation libraries
cp /tmp/cqf-common/input/cql/FHIRHelpers.cql tier-0-foundation/
cp /tmp/cqf-common/input/cql/FHIRCommon.cql tier-0-foundation/
cp /tmp/ecqm-2024/input/cql/QICoreCommon.cql tier-1-primitives/
cp /tmp/ecqm-2024/input/cql/CQMCommon.cql tier-1-primitives/
```

#### 2.2.2 VTE Prophylaxis (Direct Match to Your Priority)

| CMS Measure | Description | Source File | Target Location |
|-------------|-------------|-------------|-----------------|
| CMS108 | VTE Prophylaxis | VenousThromboembolismProphylaxis.cql | tier-4b-guidelines/VTEGuidelines.cql |
| CMS190 | ICU VTE Prophylaxis | IntensiveCareUnitVenousThromboembolismProphylaxis.cql | tier-4b-guidelines/ICUVTEGuidelines.cql |

**Key CQL Patterns to Extract:**
```cql
// From CMS108 - Example pattern for VTE prophylaxis
define "Encounter With Age Range and Without VTE Diagnosis or Obstetrical Conditions":
  "Encounter With Age Range" QualifyingEncounter
    where not exists (
      QualifyingEncounter.diagnoses EncounterDiagnoses
        where EncounterDiagnoses.code in "Obstetrics"
          or EncounterDiagnoses.code in "Venous Thromboembolism"
          or EncounterDiagnoses.code in "Obstetrics VTE"
    )

define "VTE Prophylaxis by Medication Administered or Device Applied":
  ( ["MedicationAdministration": "Low Dose Unfractionated Heparin for VTE Prophylaxis"]
    union ["MedicationAdministration": "Low Molecular Weight Heparin for VTE Prophylaxis"]
    union ["MedicationAdministration": "Injectable Factor Xa Inhibitor for VTE Prophylaxis"]
    union ["MedicationAdministration": "Warfarin"]
    union ["MedicationAdministration": "Rivaroxaban for VTE Prophylaxis"]
  ) VTEMedication
    where VTEMedication.status = 'completed'
```

#### 2.2.3 Diabetes Measures (Direct Match)

| CMS Measure | Description | Target Use |
|-------------|-------------|------------|
| CMS122 | Diabetes: Hemoglobin A1c Poor Control | A1c monitoring logic |
| CMS134 | Diabetes: Medical Attention for Nephropathy | CKD screening logic |
| CMS131 | Diabetes: Eye Exam | Retinopathy screening |

#### 2.2.4 Cardiovascular Measures

| CMS Measure | Description | Relevance to GDMT |
|-------------|-------------|-------------------|
| CMS144 | Heart Failure: Beta-Blocker Therapy | HF GDMT component |
| CMS145 | Coronary Artery Disease: Beta-Blocker Therapy | CAD management |
| CMS347 | Statin Therapy for CVD Prevention | Primary prevention |

#### 2.2.5 CDC Opioid CQL (Reference Implementation)

The CDC Opioid IG is the **gold standard** for CPG→CQL transformation. Import as templates:

```
opioid-cds-r4/input/cql/
├── OpioidCDSCommon.cql           # Common functions
├── OpioidCDSREC01.cql            # Recommendation 1 logic
├── OpioidCDSREC02.cql            # Recommendation 2 logic
├── ...
├── OpioidCDSREC12.cql            # Recommendation 12 logic
├── OMTKLogic.cql                 # MME calculation
└── MMECalculator.cql             # FHIR R4 MME calculator
```

**Why This Matters:** The CDC IG demonstrates:
- How to structure recommendation-specific libraries
- PlanDefinition + CQL integration patterns
- Evidence linking via documentation elements
- CDS Hooks integration

### 2.3 Import Transformation Requirements

When importing CQL, the following transformations may be needed:

| Transformation | Reason | Tool |
|----------------|--------|------|
| Library namespace adjustment | Match Vaidshala naming | sed/awk script |
| Version normalization | Consistent versioning | CQL formatter |
| ValueSet URL mapping | Point to KB-7 Terminology | Custom mapper |
| Include statement updates | Reflect new library locations | Dependency resolver |

**Example Transformation Script:**
```bash
#!/bin/bash
# transform_cql.sh - Adapt imported CQL for Vaidshala

INPUT_FILE=$1
OUTPUT_FILE=$2

# Update library namespace
sed -i 's/library CMS108/library VTEGuidelines/g' $INPUT_FILE

# Update includes to Vaidshala paths
sed -i 's|include FHIRHelpers|include "tier-0-foundation/FHIRHelpers"|g' $INPUT_FILE

# Map ValueSet URLs to KB-7
sed -i 's|http://cts.nlm.nih.gov/fhir/ValueSet/|http://vaidshala.io/kb7/valueset/|g' $INPUT_FILE
```

---

## 3. Track B: The Atomiser Architecture

### 3.1 Conceptual Overview

The **Atomiser** is a multi-stage LLM pipeline that "atomises" narrative clinical practice guidelines into discrete, structured knowledge units suitable for CQL generation.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           ATOMISER PIPELINE                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐     ┌───────────┐ │
│  │   SOURCE    │     │   ATOMISE   │     │  TRANSFORM  │     │  VALIDATE │ │
│  │   INGEST    │ ──▶ │   EXTRACT   │ ──▶ │   GENERATE  │ ──▶ │   APPROVE │ │
│  │             │     │             │     │             │     │           │ │
│  │ PDF/DOCX    │     │ Logic       │     │ CQL Defns   │     │ SME Review│ │
│  │ Guideline   │     │ Temporal    │     │ KB-3 Schema │     │ Compile   │ │
│  │ Text        │     │ Evidence    │     │ KB-15 Meta  │     │ Test      │ │
│  └─────────────┘     └─────────────┘     └─────────────┘     └───────────┘ │
│                                                                             │
│  ════════════════════════════════════════════════════════════════════════  │
│                                                                             │
│        LOGIC ATOMS        TEMPORAL ATOMS       EVIDENCE ATOMS               │
│       ┌──────────┐        ┌──────────┐        ┌──────────┐                 │
│       │Condition │        │Deadline  │        │GRADE     │                 │
│       │Eligibility        │Sequence  │        │Citation  │                 │
│       │Intervention       │Bundle    │        │COR/LOE   │                 │
│       │Contraind.│        │Frequency │        │Authority │                 │
│       └──────────┘        └──────────┘        └──────────┘                 │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 The Three Atom Types

Clinical practice guidelines contain three fundamentally different types of extractable knowledge:

#### 3.2.1 Logic Atoms (→ CQL Tier 4b)

**Definition:** Discrete clinical decision points that can be evaluated as true/false for a patient.

**Examples from ACC/AHA HF Guidelines:**
```
LOGIC ATOM: GDMT_Eligibility_ARNI
├── Condition: "Patient has HFrEF with LVEF ≤40%"
├── Prerequisite: "Symptomatic HF (NYHA II-IV)"
├── Contraindication: "History of angioedema with ACEi"
├── Contraindication: "Concurrent ACEi use (36-hour washout)"
├── Contraindication: "eGFR < 30 mL/min/1.73m²"
├── Contraindication: "Serum potassium > 5.5 mEq/L"
└── Contraindication: "Systolic BP < 90 mmHg"
```

**CQL Pattern Generated:**
```cql
define "ARNI Eligible":
  "Has HFrEF"
    and "NYHA Class II to IV"
    and not "History of Angioedema with ACEi"
    and not "On ACEi Within 36 Hours"
    and "eGFR >= 30"
    and "Serum Potassium <= 5.5"
    and "Systolic BP >= 90"
```

#### 3.2.2 Temporal Atoms (→ KB-3 Temporal Brain)

**Definition:** Time-bound constraints that govern WHEN and IN WHAT ORDER clinical actions must occur.

**Examples from Surviving Sepsis Campaign 2021:**
```
TEMPORAL ATOM: Sepsis_Hour1_Bundle
├── Trigger Event: "sepsis_recognition" (T0)
├── Step: Lactate measurement
│   ├── Deadline: T0 + 30 minutes
│   └── Prerequisite: None
├── Step: Blood cultures
│   ├── Deadline: T0 + 45 minutes
│   └── Prerequisite: None
├── Step: Broad-spectrum antibiotics
│   ├── Deadline: T0 + 1 hour
│   └── Prerequisite: blood_cultures (BEFORE antibiotics)
└── Bundle Completion: ALL steps within T0 + 1 hour
```

**KB-3 Schema Generated:**
```sql
INSERT INTO protocol_temporal_constraints VALUES
('SEP-HOUR1-001', 'lactate_initial', 'RELATIVE', '30 minutes', 'sepsis_recognition', NULL, NULL, 'HOUR_1_BUNDLE', 'SSC-2021', 'I'),
('SEP-HOUR1-002', 'blood_cultures', 'RELATIVE', '45 minutes', 'sepsis_recognition', NULL, '["antibiotic_admin"]', 'HOUR_1_BUNDLE', 'SSC-2021', 'I'),
('SEP-HOUR1-003', 'antibiotic_admin', 'RELATIVE', '1 hour', 'sepsis_recognition', '["blood_cultures"]', NULL, 'HOUR_1_BUNDLE', 'SSC-2021', 'I');
```

#### 3.2.3 Evidence Atoms (→ KB-15 Evidence Engine)

**Definition:** Provenance metadata linking recommendations to their supporting evidence.

**Examples from ACC/AHA Guidelines:**
```
EVIDENCE ATOM: ARNI_Recommendation
├── Class of Recommendation (COR): I (Strong)
├── Level of Evidence (LOE): A (Multiple RCTs)
├── Key Studies:
│   ├── PARADIGM-HF (NEJM 2014)
│   ├── PIONEER-HF (NEJM 2019)
│   └── PARAGON-HF (NEJM 2019)
├── Guideline Source: "2022 AHA/ACC/HFSA HF Guideline"
├── Section Reference: "7.3.1 Renin-Angiotensin System Inhibition"
└── Last Updated: 2022-04-01
```

**KB-15 Schema Generated:**
```json
{
  "recommendation_id": "HF-GDMT-ARNI-001",
  "evidence_envelope": {
    "class_of_recommendation": "I",
    "level_of_evidence": "A",
    "grade_certainty": "High",
    "citations": [
      {"pmid": "25176015", "trial": "PARADIGM-HF", "journal": "NEJM", "year": 2014},
      {"pmid": "30415601", "trial": "PIONEER-HF", "journal": "NEJM", "year": 2019}
    ],
    "guideline_source": {
      "organization": "ACC/AHA/HFSA",
      "title": "2022 Guideline for the Management of Heart Failure",
      "doi": "10.1016/j.jacc.2021.12.012"
    }
  }
}
```

### 3.3 Atomiser Stage 1: Source Ingestion

```
┌──────────────────────────────────────────────────────────────────┐
│                    STAGE 1: SOURCE INGESTION                      │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Input Formats:                                                  │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐               │
│  │  PDF    │ │  DOCX   │ │  HTML   │ │  XML    │               │
│  │(Journal)│ │(Working)│ │(Website)│ │(Struct.)│               │
│  └────┬────┘ └────┬────┘ └────┬────┘ └────┬────┘               │
│       │           │           │           │                      │
│       └───────────┴───────────┴───────────┘                      │
│                         │                                        │
│                         ▼                                        │
│           ┌─────────────────────────────┐                        │
│           │      TEXT EXTRACTION        │                        │
│           │  ┌───────────────────────┐  │                        │
│           │  │ PyMuPDF / pdfplumber  │  │                        │
│           │  │ python-docx           │  │                        │
│           │  │ BeautifulSoup         │  │                        │
│           │  └───────────────────────┘  │                        │
│           └─────────────┬───────────────┘                        │
│                         │                                        │
│                         ▼                                        │
│           ┌─────────────────────────────┐                        │
│           │    STRUCTURE DETECTION      │                        │
│           │  ┌───────────────────────┐  │                        │
│           │  │ Section Headers       │  │                        │
│           │  │ Tables (COR/LOE)      │  │                        │
│           │  │ Recommendation Boxes  │  │                        │
│           │  │ Figure References     │  │                        │
│           │  └───────────────────────┘  │                        │
│           └─────────────────────────────┘                        │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

**Key Innovation: Structure-Aware Chunking**

Unlike generic RAG systems, the Atomiser uses **guideline-specific structure detection**:

```python
class GuidelineChunker:
    """
    Chunk guidelines by semantic structure, not token count.
    Preserves recommendation boundaries and evidence linkages.
    """
    
    RECOMMENDATION_PATTERNS = [
        r"(Class\s+[I]+[ab]?|Class\s+III).*?(LOE|Level\s+of\s+Evidence)\s*[:=]?\s*([A-C])",
        r"(COR)\s*[:=]?\s*([12][ab]?|3).*?(LOE)\s*[:=]?\s*([A-C])",
        r"(Recommendation)\s+(\d+\.?\d*)[:\.]",
        r"(GRADE)\s*[:=]?\s*(Strong|Weak|Conditional)",
    ]
    
    TEMPORAL_PATTERNS = [
        r"within\s+(\d+)\s*(hours?|minutes?|days?)",
        r"(before|after|prior\s+to)\s+(.+?)(?:\.|,|;)",
        r"(every|q)\s*(\d+)\s*(hours?|days?|weeks?)",
        r"(immediately|as\s+soon\s+as|STAT)",
    ]
    
    def chunk_by_recommendation(self, text: str) -> List[RecommendationChunk]:
        """Split text preserving recommendation + evidence linkage."""
        pass
```

### 3.4 Atomiser Stage 2: Multi-Head Extraction

The core innovation: **parallel extraction heads** that simultaneously process the same text for different knowledge dimensions.

```
┌──────────────────────────────────────────────────────────────────┐
│              STAGE 2: MULTI-HEAD EXTRACTION                       │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│                    ┌─────────────────┐                           │
│                    │  RECOMMENDATION │                           │
│                    │     CHUNK       │                           │
│                    └────────┬────────┘                           │
│                             │                                    │
│         ┌───────────────────┼───────────────────┐                │
│         │                   │                   │                │
│         ▼                   ▼                   ▼                │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐        │
│  │   LOGIC     │     │  TEMPORAL   │     │  EVIDENCE   │        │
│  │ EXTRACTION  │     │ EXTRACTION  │     │ EXTRACTION  │        │
│  │    HEAD     │     │    HEAD     │     │    HEAD     │        │
│  └──────┬──────┘     └──────┬──────┘     └──────┬──────┘        │
│         │                   │                   │                │
│         ▼                   ▼                   ▼                │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐        │
│  │ Conditions  │     │ Deadlines   │     │ COR/LOE     │        │
│  │ Eligibility │     │ Sequences   │     │ Citations   │        │
│  │ Contras     │     │ Bundles     │     │ GRADE       │        │
│  │ Thresholds  │     │ Frequencies │     │ Source      │        │
│  └─────────────┘     └─────────────┘     └─────────────┘        │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

#### 3.4.1 Logic Extraction Head

**System Prompt:**
```
You are a clinical informaticist extracting computable logic from clinical practice guidelines.

For each recommendation text, extract:
1. PATIENT POPULATION: Who does this apply to? (conditions, demographics, settings)
2. CLINICAL CONDITION: What clinical state triggers this recommendation?
3. INTERVENTION: What action is recommended?
4. ELIGIBILITY CRITERIA: What must be true for patient to qualify?
5. CONTRAINDICATIONS: What excludes a patient from this intervention?
6. THRESHOLDS: Specific numeric cutoffs (lab values, vitals, durations)

Output as structured JSON with confidence scores (0-1) for each extraction.
Flag any ambiguity requiring SME review.
```

**Example Input:**
```
"In patients with HFrEF (LVEF ≤40%) with current or previous symptoms, 
the use of an ARNi is recommended to reduce morbidity and mortality. 
ARNi should not be given within 36 hours of the last dose of an ACEi. 
ARNi should not be used in patients with a history of angioedema."
(COR: I, LOE: A)
```

**Example Output:**
```json
{
  "recommendation_id": "extracted_001",
  "logic_atoms": [
    {
      "type": "PATIENT_POPULATION",
      "value": "HFrEF patients",
      "qualifiers": ["LVEF ≤40%", "current or previous symptoms"],
      "confidence": 0.95
    },
    {
      "type": "INTERVENTION",
      "value": "ARNi therapy",
      "action": "RECOMMEND",
      "confidence": 0.98
    },
    {
      "type": "CONTRAINDICATION",
      "value": "ACEi within 36 hours",
      "temporal_qualifier": "36 hours",
      "confidence": 0.97
    },
    {
      "type": "CONTRAINDICATION", 
      "value": "history of angioedema",
      "confidence": 0.99
    },
    {
      "type": "THRESHOLD",
      "parameter": "LVEF",
      "operator": "<=",
      "value": 40,
      "unit": "%",
      "confidence": 0.99
    }
  ],
  "needs_sme_review": false
}
```

#### 3.4.2 Temporal Extraction Head

**System Prompt:**
```
You are a temporal reasoning specialist extracting time constraints from clinical guidelines.

For each recommendation, extract:
1. DEADLINES: When must action occur? (absolute time, relative to trigger)
2. SEQUENCES: What order must actions occur? (prerequisite relationships)
3. BUNDLES: Are actions grouped that must complete together?
4. FREQUENCIES: How often should action recur?
5. TRIGGER EVENTS: What clinical event starts the clock?

Use ISO 8601 durations (P1H = 1 hour, P3D = 3 days, etc.)
Express relative times as offsets from named trigger events.
```

**Example Input (Sepsis):**
```
"For adults with sepsis or septic shock, we recommend that antimicrobial 
therapy be initiated as soon as possible and within 1 hour of recognition. 
Obtain blood cultures before initiating antimicrobial therapy, provided 
that doing so does not substantially delay antimicrobial administration. 
Measure lactate level and remeasure if initial lactate is elevated (>2 mmol/L)."
```

**Example Output:**
```json
{
  "temporal_atoms": [
    {
      "step_id": "antibiotic_admin",
      "deadline_type": "RELATIVE",
      "deadline_value": "PT1H",
      "deadline_from_event": "sepsis_recognition",
      "urgency": "AS_SOON_AS_POSSIBLE",
      "confidence": 0.97
    },
    {
      "step_id": "blood_cultures",
      "deadline_type": "RELATIVE",
      "deadline_value": "PT1H",
      "deadline_from_event": "sepsis_recognition",
      "must_complete_before": ["antibiotic_admin"],
      "exception": "unless_substantially_delays_antibiotics",
      "confidence": 0.94
    },
    {
      "step_id": "lactate_initial",
      "deadline_type": "RELATIVE",
      "deadline_value": "PT1H",
      "deadline_from_event": "sepsis_recognition",
      "confidence": 0.92
    },
    {
      "step_id": "lactate_recheck",
      "deadline_type": "CONDITIONAL",
      "condition": "initial_lactate > 2",
      "deadline_value": "PT2H-PT4H",
      "deadline_from_event": "lactate_initial",
      "confidence": 0.88
    }
  ],
  "bundle": {
    "bundle_id": "HOUR_1_BUNDLE",
    "steps": ["lactate_initial", "blood_cultures", "antibiotic_admin"],
    "completion_deadline": "PT1H"
  }
}
```

#### 3.4.3 Evidence Extraction Head

**System Prompt:**
```
You are a medical evidence analyst extracting provenance metadata from guidelines.

For each recommendation, extract:
1. CLASS OF RECOMMENDATION (COR): I, IIa, IIb, III (No Benefit), III (Harm)
2. LEVEL OF EVIDENCE (LOE): A (multiple RCTs), B-R (single RCT), B-NR (non-randomized), C-LD (limited data), C-EO (expert opinion)
3. GRADE RATING: If present (High, Moderate, Low, Very Low certainty)
4. KEY CITATIONS: PMID, trial names, journal references
5. GUIDELINE SOURCE: Organization, title, publication year, DOI

Preserve exact wording for COR/LOE as stated in source.
```

**Example Output:**
```json
{
  "evidence_atoms": {
    "class_of_recommendation": {
      "value": "I",
      "meaning": "Strong recommendation, benefit >>> risk",
      "confidence": 0.99
    },
    "level_of_evidence": {
      "value": "A",
      "meaning": "High-quality evidence from multiple RCTs",
      "confidence": 0.99
    },
    "key_citations": [
      {
        "trial_name": "PARADIGM-HF",
        "pmid": "25176015",
        "year": 2014,
        "journal": "NEJM",
        "finding": "41% reduction in HF hospitalization",
        "confidence": 0.95
      }
    ],
    "guideline_source": {
      "organization": "ACC/AHA/HFSA",
      "year": 2022,
      "doi": "10.1016/j.jacc.2021.12.012"
    }
  }
}
```

### 3.5 Atomiser Stage 3: CQL Generation

```
┌──────────────────────────────────────────────────────────────────┐
│              STAGE 3: CQL CODE GENERATION                         │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │                 EXTRACTED ATOMS                          │   │
│   │  ┌─────────┐  ┌─────────┐  ┌─────────┐                  │   │
│   │  │  LOGIC  │  │TEMPORAL │  │EVIDENCE │                  │   │
│   │  │  ATOMS  │  │  ATOMS  │  │  ATOMS  │                  │   │
│   │  └────┬────┘  └────┬────┘  └────┬────┘                  │   │
│   └───────┼────────────┼────────────┼────────────────────────┘   │
│           │            │            │                            │
│           │            │            │                            │
│           ▼            ▼            ▼                            │
│   ┌───────────────────────────────────────────────────────┐     │
│   │              CQL TEMPLATE ENGINE                       │     │
│   │                                                        │     │
│   │  Templates:                                            │     │
│   │  • ValueSet declarations                               │     │
│   │  • Condition detection definitions                     │     │
│   │  • Eligibility definitions                             │     │
│   │  • Contraindication definitions                        │     │
│   │  • Care gap definitions                                │     │
│   │  • Protocol status tuples                              │     │
│   └───────────────────────────────────────────────────────┘     │
│                            │                                     │
│                            ▼                                     │
│   ┌───────────────────────────────────────────────────────┐     │
│   │              OUTPUT: CQL + KB-3 + KB-15               │     │
│   │                                                        │     │
│   │  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐   │     │
│   │  │ Guidelines   │ │ Temporal     │ │ Evidence     │   │     │
│   │  │ .cql         │ │ Constraints  │ │ Metadata     │   │     │
│   │  │ (Tier 4b)    │ │ (KB-3 SQL)   │ │ (KB-15 JSON) │   │     │
│   │  └──────────────┘ └──────────────┘ └──────────────┘   │     │
│   └───────────────────────────────────────────────────────┘     │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

**CQL Template Example:**
```cql
// Template: Guideline Library Header
library {{guideline_name}} version '{{version}}'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1'
include QICoreCommon version '1.0.0'
include ClinicalCalculators version '1.0.0'

// ValueSets from KB-7 Terminology
{% for valueset in valuesets %}
valueset "{{valueset.name}}": '{{valueset.oid}}'
{% endfor %}

// Context
context Patient

// ============================================
// CONDITION DETECTION DEFINITIONS
// ============================================
{% for condition in conditions %}
/**
 * {{condition.description}}
 * Source: {{condition.source_text | truncate(100)}}
 * Confidence: {{condition.confidence}}
 */
define "{{condition.name}}":
  {{condition.cql_expression}}

{% endfor %}

// ============================================
// ELIGIBILITY DEFINITIONS  
// ============================================
{% for eligibility in eligibility_criteria %}
define "{{eligibility.intervention}} Eligible":
  "{{eligibility.population}}"
    {% for req in eligibility.requirements %}
    and {{req}}
    {% endfor %}
    {% for contra in eligibility.contraindications %}
    and not "{{contra}}"
    {% endfor %}

{% endfor %}

// ============================================
// CARE GAP DEFINITIONS
// ============================================
{% for gap in care_gaps %}
define "{{gap.name}}":
  "{{gap.eligible_population}}"
    and not "{{gap.current_treatment}}"
{% endfor %}

// ============================================
// PROTOCOL STATUS TUPLE (for KB-19)
// ============================================
define "Protocol Status":
  {
    {% for protocol in protocols %}
    {{protocol.id}}: {
      applicable: "{{protocol.applicable_definition}}",
      gaps: "{{protocol.gaps_definition}}",
      contraindicated: "{{protocol.contraindicated_definition}}"
    }{% if not loop.last %},{% endif %}
    {% endfor %}
  }
```

### 3.6 Atomiser Stage 4: Validation & Governance

```
┌──────────────────────────────────────────────────────────────────┐
│              STAGE 4: VALIDATION & GOVERNANCE                     │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                  AUTOMATED VALIDATION                        │ │
│  │                                                              │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │ │
│  │  │    CQL      │  │  Semantic   │  │   Test      │         │ │
│  │  │  Compiler   │  │ Consistency │  │   Cases     │         │ │
│  │  │   Check     │  │   Check     │  │ Execution   │         │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘         │ │
│  │                                                              │ │
│  │  Checks:                                                     │ │
│  │  • Syntax validation (CQL compiler)                          │ │
│  │  • ValueSet resolution (all OIDs exist in KB-7)              │ │
│  │  • Circular dependency detection                             │ │
│  │  • Threshold consistency (same parameter, same cutoffs)      │ │
│  │  • Temporal constraint feasibility                           │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                            │                                     │
│                            ▼                                     │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                    SME REVIEW QUEUE                          │ │
│  │                                                              │ │
│  │  Items flagged for human review:                             │ │
│  │  • Confidence score < 0.85                                   │ │
│  │  • Ambiguous temporal expressions                            │ │
│  │  • Missing evidence linkage                                  │ │
│  │  • Novel interventions not in existing ValueSets             │ │
│  │  • Conflicting recommendations between sources               │ │
│  │                                                              │ │
│  │  Review workflow → KB-18 Governance                          │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                            │                                     │
│                            ▼                                     │
│  ┌─────────────────────────────────────────────────────────────┐ │
│  │                  ACTIVATION PIPELINE                         │ │
│  │                                                              │ │
│  │  DRAFT → PENDING_REVIEW → UNDER_REVIEW → APPROVED → ACTIVE  │ │
│  │                                                              │ │
│  └─────────────────────────────────────────────────────────────┘ │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

---

## 4. Implementation Roadmap

### 4.1 Phase 1: Foundation Import (Weeks 1-2)

| Task | Deliverable | Owner |
|------|-------------|-------|
| Clone CQF repositories | Local copies of all source repos | DevOps |
| Import FHIRHelpers + foundation | tier-0, tier-1 libraries active | CQL Engineer |
| Import VTE measures (CMS108/190) | VTEGuidelines.cql in tier-4b | Clinical Informaticist |
| ValueSet mapping to KB-7 | OID resolution verified | Terminologist |
| Integration test with CQL engine | All imports compile & execute | QA |

### 4.2 Phase 2: Atomiser MVP (Weeks 3-5)

| Task | Deliverable | Owner |
|------|-------------|-------|
| Guideline chunker implementation | Structure-aware PDF/DOCX parser | NLP Engineer |
| Logic extraction head | LLM prompt + output schema | ML Engineer |
| Temporal extraction head | Temporal atom extraction | ML Engineer |
| Evidence extraction head | COR/LOE/citation extraction | ML Engineer |
| CQL template engine | Jinja2 templates for CQL generation | CQL Engineer |

### 4.3 Phase 3: Priority Guidelines (Weeks 6-10)

| Week | Guideline | Approach |
|------|-----------|----------|
| 6 | Sepsis (SSC 2021) | Atomiser - heavy temporal constraints |
| 7 | HF GDMT (ACC/AHA 2022) | Atomiser + import CMS144/145 |
| 8 | T2DM (ADA 2024) | Import CMS122/134 + Atomiser for gaps |
| 9 | AFib (2023) | Import CMS71 + Atomiser |
| 10 | CKD (KDIGO 2024) | Atomiser - limited existing CQL |

### 4.4 Phase 4: KB Integration (Weeks 11-12)

| Task | Deliverable |
|------|-------------|
| KB-3 temporal constraint loader | Atomiser temporal atoms → KB-3 |
| KB-15 evidence metadata loader | Atomiser evidence atoms → KB-15 |
| KB-19 orchestrator configuration | Protocol routing rules configured |
| End-to-end test | Patient case → CQL evaluation → KB-19 recommendation |

---

## 5. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| CQL Compilation Success | 100% | Automated CI |
| ValueSet Resolution | 100% | KB-7 query validation |
| Atomiser Extraction Accuracy | >90% (logic), >85% (temporal), >95% (evidence) | SME validation sample |
| Temporal Constraint Accuracy | >90% deadlines correctly extracted | Gold standard comparison |
| Evidence Traceability | 100% recommendations have citation | KB-15 audit |
| Time to CQL (Atomiser) | <4 hours per guideline | Pipeline metrics |
| SME Review Burden | <20% of extractions flagged | Governance queue metrics |

---

## 6. Risk Mitigation

| Risk | Mitigation |
|------|------------|
| LLM hallucination in extraction | Multi-stage validation + SME review for low-confidence |
| ValueSet gaps (terms not in KB-7) | Terminology expansion workflow + SNOMED-CT fallback |
| Temporal ambiguity ("as soon as possible") | Conservative defaults + flag for SME |
| Version conflicts in imported CQL | Namespace isolation + version pinning |
| Guideline updates invalidate CQL | Differential update pipeline + governance versioning |

---

## 7. Appendix: CQL Style Guide for Vaidshala

### 7.1 Naming Conventions

```cql
// Library names: ConditionGuidelines
library HFGDMTGuidelines version '1.0.0'

// Definition names: "Verb Phrase" or "Noun Phrase"
define "Has HFrEF":
define "On Beta Blocker":
define "ARNI Eligible - Blood Pressure Safe":
define "Missing GDMT Pillars":

// Parameters: camelCase
parameter MeasurementPeriod Interval<DateTime>

// Code system aliases: SCREAMING_SNAKE
codesystem "SNOMED": 'http://snomed.info/sct'
codesystem "RXNORM": 'http://www.nlm.nih.gov/research/umls/rxnorm'
```

### 7.2 Definition Structure

```cql
/**
 * [Brief description]
 * 
 * Source: [Guideline reference]
 * COR: [Class of Recommendation]
 * LOE: [Level of Evidence]
 */
define "Definition Name":
  [CQL expression]
```

### 7.3 Protocol Status Tuple Pattern

```cql
// Standard output format for KB-19 consumption
define "GDMT Protocol Status":
  {
    protocol_id: 'HF-GDMT-001',
    applicable: "Has HFrEF" and "Symptomatic HF",
    current_pillars: {
      arni_or_acei_arb: "On ARNI or ACEi or ARB",
      beta_blocker: "On Evidence Based Beta Blocker",
      mra: "On MRA",
      sglt2i: "On SGLT2i"
    },
    gaps: "Missing GDMT Pillars",
    next_action: case
      when not "On ARNI or ACEi or ARB" then 'INITIATE_RASI'
      when not "On Evidence Based Beta Blocker" then 'INITIATE_BB'
      when not "On MRA" then 'INITIATE_MRA'
      when not "On SGLT2i" then 'INITIATE_SGLT2I'
      else 'OPTIMIZE_DOSES'
    end
  }
```

---

**Document Status: READY FOR IMPLEMENTATION**

This architecture enables Vaidshala to rapidly populate its CQL repository by:
1. Importing proven CQL from authoritative sources (Track A)
2. Generating new CQL from narrative guidelines via the Atomiser (Track B)

The Atomiser's multi-head extraction approach ensures that clinical logic, temporal constraints, and evidence metadata are extracted in parallel and routed to their appropriate knowledge bases (CQL, KB-3, KB-15).
