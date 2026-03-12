# Vaidshala Clinical Knowledge System
## Guideline Asset Registry & Strategic Import Plan

**Version:** 2.0 (Registry-First Architecture)  
**Date:** January 2026  
**Status:** FINAL REVIEW  
**Philosophy:** Deterministic First, LLM as Last Resort

---

## Executive Summary

This document implements a **three-step, registry-first approach** to populating the Vaidshala CQL repository:

1. **Step 1: Create Guideline Asset Registry** - Catalog all existing computable clinical knowledge
2. **Step 2: Map Assets to KB Architecture** - Route each asset to CQL Tier 4b, KB-3, KB-12, KB-15
3. **Step 3: Design Atomiser for Gaps ONLY** - LLM extraction constrained to residual coverage

This approach minimizes LLM dependency, maximizes governance safety, and leverages the substantial body of existing authoritative CQL content.

---

## Part 1: Guideline Asset Registry

### 1.1 Registry Structure

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    VAIDSHALA GUIDELINE ASSET REGISTRY                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  TIER 1: PLATINUM (Production-Ready CQL)                                    │
│  ═══════════════════════════════════════                                   │
│  • CMS eCQM Library (100+ measures)                                        │
│  • CDC Opioid Prescribing IG (12 recommendations)                          │
│  • CDC Opioid MME Calculator                                               │
│                                                                             │
│  TIER 2: GOLD (L3 Machine-Readable)                                        │
│  ═════════════════════════════════                                         │
│  • WHO SMART Guidelines (HIV, ANC, Immunization, TB)                       │
│  • CPG-on-FHIR Reference Implementations                                   │
│                                                                             │
│  TIER 3: SILVER (Structured but Requires Adaptation)                       │
│  ════════════════════════════════════════════════════                      │
│  • AHRQ CDS Connect Artifacts (archived, GitHub available)                 │
│  • EBMonFHIR CPG Examples                                                  │
│                                                                             │
│  TIER 4: BRONZE (Narrative with Structured Tables)                         │
│  ════════════════════════════════════════════════════                      │
│  • ACC/AHA Guidelines (COR/LOE tables)                                     │
│  • Surviving Sepsis Campaign 2021                                          │
│  • KDIGO, GOLD, CHEST Guidelines                                           │
│                                                                             │
│  TIER 5: GAPS (Require Atomiser)                                           │
│  ══════════════════════════════                                            │
│  • Complex sequencing logic                                                │
│  • Dose titration protocols                                                │
│  • Rare disease management                                                 │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 Tier 1: PLATINUM Assets (Direct CQL Import)

#### 1.2.1 CMS eCQM Library

| Repository | URL | Content | Last Updated |
|------------|-----|---------|--------------|
| ecqm-content-qicore-2024 | github.com/cqframework/ecqm-content-qicore-2024 | FHIR R4/QICore measures | June 2024 |
| ecqm-content-cms-2025 | github.com/cqframework/ecqm-content-cms-2025 | 2025 Connectathon content | Active |
| CQL-Formatting-and-Usage-Wiki | github.com/esacinc/CQL-Formatting-and-Usage-Wiki | Style guides, examples | Active |

**Specific Measures Relevant to Vaidshala Priority Guidelines:**

| CMS ID | Measure Name | Clinical Domain | Vaidshala Target |
|--------|--------------|-----------------|------------------|
| CMS108 | VTE Prophylaxis | VTE | Tier 4b - VTEGuidelines.cql |
| CMS190 | ICU VTE Prophylaxis | VTE/ICU | Tier 4b - ICUVTEGuidelines.cql |
| CMS122 | Diabetes: HbA1c Poor Control | Diabetes | Tier 4b - T2DMGuidelines.cql |
| CMS134 | Diabetes: Nephropathy | Diabetes/CKD | Tier 4b - DiabetesCKD.cql |
| CMS131 | Diabetes: Eye Exam | Diabetes | Tier 4b - DiabetesScreening.cql |
| CMS144 | HF: Beta-Blocker Therapy | Heart Failure | Tier 4b - HFBetaBlocker.cql |
| CMS145 | CAD: Beta-Blocker Therapy | Cardiovascular | Tier 4b - CADGuidelines.cql |
| CMS71 | AFib: Anticoagulation Therapy | Atrial Fibrillation | Tier 4b - AFibGuidelines.cql |
| CMS347 | Statin Therapy for CVD Prevention | Cardiovascular | Tier 4b - StatinGuidelines.cql |
| CMS506 | Safe Use of Opioids | Pain/Safety | Tier 4b - OpioidSafety.cql |

**Shared Libraries (Foundation):**

| Library | Purpose | Target Location |
|---------|---------|-----------------|
| FHIRHelpers | FHIR data model access | tier-0-foundation/ |
| FHIRCommon | Common FHIR patterns | tier-0-foundation/ |
| QICoreCommon | QICore profile support | tier-1-primitives/ |
| CQMCommon | Quality measure patterns | tier-2-cqm-infrastructure/ |
| MATGlobalCommonFunctions | Shared measure functions | tier-2-cqm-infrastructure/ |

#### 1.2.2 CDC Opioid Prescribing IG

| Repository | URL | Version | Status |
|------------|-----|---------|--------|
| opioid-cds-r4 | github.com/cqframework/opioid-cds-r4 | 2022.1.0 | Production |
| opioid-mme-r4 | fhir.org/guides/cdc/opioid-mme-r4 | R4 | Production |

**Why This is "Platinum":**
- **12 complete PlanDefinitions** mapping recommendations to CQL
- **MME Calculator** with pure CQL implementation (OMTKLogic)
- **CDS Hooks integration** patterns
- **Test cases** included
- **Evidence linking** via PlanDefinition.action.documentation

**Content Structure:**

```
opioid-cds-r4/input/
├── cql/
│   ├── OpioidCDSCommon.cql           # Shared functions
│   ├── OpioidCDSREC01.cql            # Recommendation 1
│   ├── OpioidCDSREC02.cql            # Recommendation 2
│   ├── OpioidCDSREC03.cql            # Recommendation 3
│   ├── OpioidCDSREC04.cql            # Recommendation 4
│   ├── OpioidCDSREC05.cql            # Recommendation 5 (MME thresholds)
│   ├── OpioidCDSREC06.cql            # Recommendation 6
│   ├── OpioidCDSREC07.cql            # Recommendation 7
│   ├── OpioidCDSREC08.cql            # Recommendation 8
│   ├── OpioidCDSREC09.cql            # Recommendation 9
│   ├── OpioidCDSREC10.cql            # Recommendation 10
│   ├── OpioidCDSREC11.cql            # Recommendation 11
│   ├── OpioidCDSREC12.cql            # Recommendation 12
│   └── OMTKLogic.cql                 # MME calculation engine
├── resources/
│   ├── plandefinition/               # PlanDefinition for each rec
│   └── library/                      # CQL Library resources
└── tests/
    └── <test cases per recommendation>
```

### 1.3 Tier 2: GOLD Assets (WHO SMART Guidelines)

#### 1.3.1 Available SMART Guideline Packages

| Domain | IG URL | GitHub | L3 Status |
|--------|--------|--------|-----------|
| HIV | worldhealthorganization.github.io/smart-hiv | github.com/WorldHealthOrganization/smart-hiv | v0.4.3 (Demo) |
| Antenatal Care | smart.who.int/anc-cds | github.com/WorldHealthOrganization/smart-anc | Published |
| Immunization | build.fhir.org/ig/WorldHealthOrganization/smart-immunizations | github.com/WorldHealthOrganization/smart-immunizations | Active |
| Tuberculosis | (DAK published May 2024) | In development | L2 Complete |
| IPS Pilgrimage | smart.who.int/ips-pilgrimage | github.com/WorldHealthOrganization/smart-ips-pilgrimage | v2.0.3 |
| PH4H | worldhealthorganization.github.io/smart-ph4h | github.com/WorldHealthOrganization/smart-ph4h | v0.1.0 (Demo) |

**SMART Guidelines L3 Structure (Key for Import):**

```
smart-<domain>/
├── input/
│   ├── cql/                          # CQL decision logic ✓
│   │   ├── <Domain>Common.cql
│   │   ├── <Domain>Recommendations.cql
│   │   └── <Domain>Indicators.cql
│   ├── resources/
│   │   ├── plandefinition/           # PlanDefinitions with timing ✓
│   │   ├── activitydefinition/       # Activity specifications ✓
│   │   ├── library/                  # CQL Library FHIR resources
│   │   └── questionnaire/            # Data collection forms
│   ├── vocabulary/
│   │   ├── codesystem/               # Terminology ✓
│   │   └── valueset/                 # Value sets for KB-7 ✓
│   └── pagecontent/
│       └── decision-logic.md         # L2 DAK documentation
└── tests/
    └── <test cases>
```

**Key Extraction Points:**

| SMART Component | Vaidshala Target | Extraction Method |
|-----------------|------------------|-------------------|
| input/cql/*.cql | CQL Tier 4b | Direct copy + namespace adapt |
| PlanDefinition.action.timing | KB-3 Temporal | Parse timing[x] elements |
| PlanDefinition.action.documentation | KB-15 Evidence | Extract relatedArtifact |
| vocabulary/valueset/ | KB-7 Terminology | Import + OHDSI mapping |
| PlanDefinition.action.relatedAction | KB-3 Sequences | Extract relationship + offset |

#### 1.3.2 Jurisdiction Handling

**Critical Architecture Note (from your deep review):**

WHO SMART Guidelines are optimized for **global/low-resource settings**. When importing:

```sql
-- KB-15 Evidence Metadata must track jurisdiction
INSERT INTO evidence_metadata (recommendation_id, jurisdiction, preference_rank) VALUES
('HIV-ART-001', 'WHO/Global', 2),
('HIV-ART-001', 'CDC/US', 1);  -- US implementations prefer CDC when available
```

**KB-19 Arbitration Logic:**
```
IF Patient.Location.Country == "US" THEN
  Prefer(CDC, AHRQ) OVER WHO
ELSE IF Patient.Location.ResourceSetting == "Low" THEN
  Prefer(WHO)
ELSE
  Prefer(LocalAdaptation)
```

### 1.4 Tier 3: SILVER Assets (AHRQ CDS Connect)

**Important Update (January 2026):** AHRQ CDS Connect Repository went offline April 28, 2025. However:
- **GitHub repositories remain available** at github.com/AHRQ-CDS
- **Implementation Guides archived** at digital.ahrq.gov
- **CQL Studio** (successor) in development via HL7

#### 1.4.1 Available GitHub Repositories

| Repository | Content | Status |
|------------|---------|--------|
| AHRQ-CDS-Connect-PAIN-MANAGEMENT-SUMMARY | SMART on FHIR pain dashboard | Available |
| CQL-Testing-Framework | Test case execution library | Available |
| AHRQ-CDS-Connect-CQL-SERVICES | Express.js CQL execution | Available |

#### 1.4.2 Archived Implementation Guides (Priority for Vaidshala)

| Artifact | Clinical Domain | L3 Status |
|----------|-----------------|-----------|
| Statin Therapy for Primary Prevention | Cardiovascular | CQL available |
| USPSTF Aspirin Therapy | Cardiovascular Prevention | CQL available |
| Factors to Consider in Managing Chronic Pain | Pain Management | CQL + SMART app |
| CMS Million Hearts ASCVD Risk | Cardiovascular | CQL available |

### 1.5 Tier 4: BRONZE Assets (Structured Tables in PDFs)

These guidelines have **COR/LOE recommendation tables** that can be parsed deterministically:

| Guideline | Source | Table Structure | Import Method |
|-----------|--------|-----------------|---------------|
| 2022 ACC/AHA/HFSA Heart Failure | JACC/Circulation | COR + LOE + Recommendation | Table extraction |
| Surviving Sepsis Campaign 2021 | Critical Care Medicine | Strong/Weak + QoE | Table extraction |
| KDIGO CKD 2024 | Kidney International | Grade + Level | Table extraction |
| GOLD COPD 2024 | goldcopd.org | Evidence levels | Table extraction |
| CHEST VTE 2022 | CHEST Journal | GRADE | Table extraction |

**Table Extraction Approach (Deterministic):**

```python
# ACC/AHA COR/LOE pattern (deterministic regex)
COR_MAPPING = {
    'I': {'strength': 'STRONG', 'direction': 'FOR'},
    'IIa': {'strength': 'MODERATE', 'direction': 'FOR'},
    'IIb': {'strength': 'WEAK', 'direction': 'FOR'},
    'III-NoBenefit': {'strength': 'STRONG', 'direction': 'AGAINST'},
    'III-Harm': {'strength': 'STRONG', 'direction': 'AGAINST'}
}

LOE_MAPPING = {
    'A': {'quality': 'HIGH', 'source': 'Multiple RCTs'},
    'B-R': {'quality': 'MODERATE', 'source': 'Single RCT'},
    'B-NR': {'quality': 'MODERATE', 'source': 'Non-randomized'},
    'C-LD': {'quality': 'LOW', 'source': 'Limited data'},
    'C-EO': {'quality': 'VERY_LOW', 'source': 'Expert opinion'}
}
```

### 1.6 Tier 5: GAPS (Atomiser Required)

After exhausting Tiers 1-4, only these remain for LLM-assisted extraction:

| Content Type | Example | Why LLM Needed |
|--------------|---------|----------------|
| Dose titration sequences | "Start low, titrate q2wk to target" | Complex conditional logic |
| Conditional exceptions | "Unless frail, then consider..." | Nuanced clinical judgment |
| Multi-step care bundles | Sepsis Hour-1 sequencing | Temporal dependencies |
| Rare disease protocols | Amyloidosis management | No existing CQL |

**Estimated Coverage:**

| Tier | Coverage % | LLM Use |
|------|------------|---------|
| Platinum (CMS/CDC) | 40-50% | None |
| Gold (WHO SMART) | 15-25% | None |
| Silver (AHRQ) | 5-10% | None |
| Bronze (Tables) | 10-15% | <5% for ambiguous cells |
| Gaps (Atomiser) | 10-15% | Targeted |

**Total LLM exposure: ~10-15% of content** (vs. 100% in naive approach)

---

## Part 2: Asset-to-KB Mapping

### 2.1 Mapping Schema

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         ASSET → KB MAPPING MATRIX                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  SOURCE ASSET          →    KB DESTINATION    →    EXTRACTION METHOD        │
│  ─────────────────────────────────────────────────────────────────────────  │
│                                                                             │
│  CQL Library (.cql)    →    CQL Tier 4b       →    Direct copy + adapt     │
│  PlanDefinition        →    CQL + KB-3 + KB-15 →  Parse structure          │
│  ActivityDefinition    →    CQL Tier 4b       →    Convert to CQL          │
│  ValueSet              →    KB-7 Terminology  →    Import + OHDSI map      │
│  CodeSystem            →    KB-7 Terminology  →    Import + map            │
│  COR/LOE Table         →    KB-15 Evidence    →    Regex extraction        │
│  Timing (Duration)     →    KB-3 Temporal     →    Parse timing[x]         │
│  relatedAction         →    KB-3 Sequences    →    Parse relationships     │
│  documentation         →    KB-15 Citations   →    Extract relatedArtifact │
│  Narrative gaps        →    CQL + KB-3 + KB-15→   Atomiser (LLM)          │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 CQL Tier 4b Mapping

#### 2.2.1 From eCQM Measures

**Source:** CMS eCQM CQL files  
**Method:** Direct import with namespace adaptation

```bash
# Import script
SOURCE=/tmp/ecqm-content-qicore-2024/input/cql
TARGET=./vaidshala/cql/tier-4b-guidelines

# VTE Measures
cp $SOURCE/VenousThromboembolismProphylaxisFHIR.cql $TARGET/VTEGuidelines.cql
cp $SOURCE/IntensiveCareUnitVenousThromboembolismProphylaxisFHIR.cql $TARGET/ICUVTEGuidelines.cql

# Diabetes Measures
cp $SOURCE/DiabetesHemoglobinA1cHbA1cPoorControl9FHIR.cql $TARGET/T2DMGuidelines.cql
cp $SOURCE/DiabetesMedicalAttentionforNephropathyFHIR.cql $TARGET/DiabetesCKD.cql

# Cardiovascular Measures
cp $SOURCE/HeartFailureBetaBlockerTherapyforLVSDFHIR.cql $TARGET/HFBetaBlocker.cql
cp $SOURCE/StatinTherapyforthePreventionandTreatmentofCardiovascularDiseaseFHIR.cql $TARGET/StatinGuidelines.cql

# Apply namespace transformation
./scripts/adapt_namespace.sh $TARGET/*.cql
```

**Namespace Adaptation Script:**
```bash
#!/bin/bash
# adapt_namespace.sh

for file in "$@"; do
    # Update library declaration
    sed -i 's/library [A-Za-z0-9]*FHIR/library Vaidshala_${basename%.*}/' "$file"
    
    # Update include paths to Vaidshala structure
    sed -i 's|include FHIRHelpers|include "tier-0-foundation/FHIRHelpers"|' "$file"
    sed -i 's|include QICoreCommon|include "tier-1-primitives/QICoreCommon"|' "$file"
    
    # Map ValueSet URLs to KB-7
    sed -i 's|http://cts.nlm.nih.gov/fhir/ValueSet/|http://vaidshala.io/kb7/valueset/|' "$file"
done
```

#### 2.2.2 From WHO SMART Guidelines

**Source:** SMART L3 CQL libraries  
**Method:** Direct import with terminology mapping

```bash
# Import WHO HIV CQL
SOURCE=/tmp/smart-hiv/input/cql
TARGET=./vaidshala/cql/tier-4b-guidelines

cp $SOURCE/HIVCommon.cql $TARGET/WHOHIVCommon.cql
cp $SOURCE/HIVRecommendations.cql $TARGET/WHOHIVRecommendations.cql
cp $SOURCE/HIVIndicators.cql $TARGET/WHOHIVIndicators.cql

# Add jurisdiction metadata header
for file in $TARGET/WHO*.cql; do
    sed -i '1i// Jurisdiction: WHO/Global' "$file"
    sed -i '2i// Adaptation Required: YES - verify formulary and local protocols' "$file"
done
```

#### 2.2.3 From CDC Opioid IG

**Source:** opioid-cds-r4 CQL libraries  
**Method:** Direct import (gold standard reference)

```bash
# Import CDC Opioid (complete module)
SOURCE=/tmp/opioid-cds-r4/input/cql
TARGET=./vaidshala/cql/tier-4b-guidelines/opioid

mkdir -p $TARGET
cp $SOURCE/OpioidCDS*.cql $TARGET/
cp $SOURCE/OMTKLogic.cql $TARGET/
cp $SOURCE/MMECalculator.cql $TARGET/

# Import as reference implementation - minimal modification
```

### 2.3 KB-3 Temporal Brain Mapping

#### 2.3.1 From PlanDefinition.action.timing

**Source:** PlanDefinition timing[x] elements  
**Method:** Deterministic FHIR parsing

```python
# Extract temporal constraints from PlanDefinition
def extract_temporal_from_plandefinition(plandefinition: dict) -> List[dict]:
    """
    Extract KB-3 temporal constraints from FHIR PlanDefinition.
    Pure deterministic parsing - no LLM.
    """
    constraints = []
    
    for action in plandefinition.get('action', []):
        action_id = action.get('id') or action.get('title', '').replace(' ', '_')
        
        # Extract timing[x]
        timing = None
        if 'timingDuration' in action:
            timing = {
                'deadline_type': 'RELATIVE',
                'deadline_value': f"PT{action['timingDuration']['value']}{action['timingDuration']['unit'][0].upper()}"
            }
        elif 'timingTiming' in action:
            repeat = action['timingTiming'].get('repeat', {})
            if repeat.get('frequency'):
                timing = {
                    'deadline_type': 'RECURRING',
                    'frequency': repeat['frequency'],
                    'period': repeat.get('period'),
                    'period_unit': repeat.get('periodUnit')
                }
        
        # Extract relatedAction (sequences)
        for related in action.get('relatedAction', []):
            constraints.append({
                'step_id': action_id,
                'related_step': related['actionId'],
                'relationship': related['relationship'],  # before-start, after-end, etc.
                'offset': related.get('offsetDuration', {}).get('value')
            })
        
        if timing:
            timing['step_id'] = action_id
            constraints.append(timing)
    
    return constraints
```

**Example: Sepsis Hour-1 Bundle (from SSC PlanDefinition)**

```json
{
  "action": [
    {
      "id": "lactate_measurement",
      "title": "Measure Lactate",
      "timingDuration": {"value": 1, "unit": "hour"},
      "relatedAction": []
    },
    {
      "id": "blood_cultures",
      "title": "Obtain Blood Cultures",
      "timingDuration": {"value": 1, "unit": "hour"},
      "relatedAction": [
        {
          "actionId": "antibiotic_admin",
          "relationship": "before-start"
        }
      ]
    },
    {
      "id": "antibiotic_admin",
      "title": "Administer Antibiotics",
      "timingDuration": {"value": 1, "unit": "hour"},
      "relatedAction": []
    }
  ]
}
```

**Generated KB-3 SQL:**

```sql
INSERT INTO protocol_temporal_constraints 
(protocol_id, step_id, deadline_type, deadline_value, deadline_from_event, prerequisites, must_complete_before, bundle_id, source_guideline, recommendation_strength)
VALUES
('SEP-HOUR1', 'lactate_measurement', 'RELATIVE', 'PT1H', 'sepsis_recognition', NULL, NULL, 'HOUR_1_BUNDLE', 'SSC-2021', 'STRONG'),
('SEP-HOUR1', 'blood_cultures', 'RELATIVE', 'PT1H', 'sepsis_recognition', NULL, '["antibiotic_admin"]', 'HOUR_1_BUNDLE', 'SSC-2021', 'STRONG'),
('SEP-HOUR1', 'antibiotic_admin', 'RELATIVE', 'PT1H', 'sepsis_recognition', '["blood_cultures"]', NULL, 'HOUR_1_BUNDLE', 'SSC-2021', 'STRONG');
```

#### 2.3.2 Measure → Guideline Temporal Injection

**Critical Pattern (from your deep review):**

Measures are **temporally blind** ("Did beta blocker happen? Yes/No"). When refactoring:

```sql
-- Inject default temporal constraint from CMS measurement window
-- Later, Atomiser can tighten this from actual guideline text

INSERT INTO protocol_temporal_constraints 
(protocol_id, step_id, deadline_type, deadline_value, deadline_from_event, source, confidence)
VALUES
-- Default CMS window (24 hours from encounter)
('CMS108-VTE', 'vte_prophylaxis', 'RELATIVE', 'PT24H', 'admission', 'CMS_MEASURE_WINDOW', 0.7),
-- Atomiser upgrade later: CHEST guideline specifies within 12 hours for high-risk
('CMS108-VTE', 'vte_prophylaxis', 'RELATIVE', 'PT12H', 'admission', 'CHEST_GUIDELINE', 0.95);

-- KB-3 rule: Guideline temporal > Measure temporal
```

### 2.4 KB-15 Evidence Engine Mapping

#### 2.4.1 From COR/LOE Tables (Deterministic)

**Source:** ACC/AHA guideline PDFs  
**Method:** PDF table extraction + regex parsing

```python
# Deterministic COR/LOE extraction
import pdfplumber
import re

def extract_recommendations_from_table(pdf_path: str) -> List[dict]:
    """
    Extract COR/LOE recommendations from ACC/AHA style tables.
    Deterministic regex - no LLM needed.
    """
    COR_PATTERNS = [
        (r'(?:Class\s*)?I(?![IVab])', 'I'),
        (r'(?:Class\s*)?IIa', 'IIa'),
        (r'(?:Class\s*)?IIb', 'IIb'),
        (r'(?:Class\s*)?III.*(?:Harm|harm)', 'III-Harm'),
        (r'(?:Class\s*)?III.*(?:No\s*Benefit|no\s*benefit)', 'III-NoBenefit'),
    ]
    
    LOE_PATTERNS = [
        (r'(?:LOE\s*|Level\s*)?A(?![a-z])', 'A'),
        (r'(?:LOE\s*|Level\s*)?B-?R', 'B-R'),
        (r'(?:LOE\s*|Level\s*)?B-?NR', 'B-NR'),
        (r'(?:LOE\s*|Level\s*)?C-?LD', 'C-LD'),
        (r'(?:LOE\s*|Level\s*)?C-?EO', 'C-EO'),
    ]
    
    recommendations = []
    
    with pdfplumber.open(pdf_path) as pdf:
        for page in pdf.pages:
            tables = page.extract_tables()
            for table in tables:
                if is_recommendation_table(table):
                    for row in table[1:]:  # Skip header
                        cor = extract_pattern(row[0], COR_PATTERNS)
                        loe = extract_pattern(row[1], LOE_PATTERNS)
                        text = row[2] if len(row) > 2 else ''
                        
                        if cor and loe:
                            recommendations.append({
                                'cor': cor,
                                'loe': loe,
                                'text': text,
                                'needs_llm': False
                            })
                        elif text:  # Ambiguous - flag for review
                            recommendations.append({
                                'cor': None,
                                'loe': None,
                                'text': text,
                                'needs_llm': True  # <5% of cases
                            })
    
    return recommendations
```

**Generated KB-15 Entry:**

```json
{
  "recommendation_id": "HF-GDMT-ARNI-001",
  "evidence_metadata": {
    "class_of_recommendation": "I",
    "level_of_evidence": "A",
    "grade_certainty": "HIGH",
    "strength": "STRONG",
    "direction": "FOR",
    "extraction_method": "DETERMINISTIC_TABLE",
    "llm_involvement": false
  },
  "source": {
    "guideline": "2022 ACC/AHA/HFSA Heart Failure Guideline",
    "doi": "10.1016/j.jacc.2021.12.012",
    "section": "7.3.1",
    "jurisdiction": "US"
  }
}
```

#### 2.4.2 From PlanDefinition.action.documentation

**Source:** FHIR PlanDefinition relatedArtifact  
**Method:** Deterministic FHIR parsing

```python
def extract_evidence_from_plandefinition(plandefinition: dict) -> dict:
    """
    Extract evidence metadata from PlanDefinition.
    Uses action.documentation relatedArtifact elements.
    """
    evidence = {
        'citations': [],
        'quality_of_evidence': None,
        'strength_of_recommendation': None
    }
    
    # Extract from extensions
    for ext in plandefinition.get('extension', []):
        if ext['url'].endswith('qualityOfEvidence'):
            evidence['quality_of_evidence'] = ext['valueCodeableConcept']['coding'][0]['code']
        elif ext['url'].endswith('strengthOfRecommendation'):
            evidence['strength_of_recommendation'] = ext['valueCodeableConcept']['coding'][0]['code']
    
    # Extract citations from action.documentation
    for action in plandefinition.get('action', []):
        for doc in action.get('documentation', []):
            if doc['type'] == 'citation':
                evidence['citations'].append({
                    'display': doc.get('display'),
                    'url': doc.get('document', {}).get('url'),
                    'citation': doc.get('citation')
                })
    
    return evidence
```

### 2.5 KB-12 Interaction Engine Mapping

**Source:** Drug-drug interaction data in CDC Opioid IG, CMS measures  
**Method:** Extract from CQL contraindication logic

```python
def extract_interactions_from_cql(cql_content: str) -> List[dict]:
    """
    Extract drug interactions from CQL contraindication definitions.
    Pattern: "not exists [MedicationRequest: X] concurrent with [Y]"
    """
    interactions = []
    
    # Pattern for concurrent medication checks
    pattern = r'not exists.*\[MedicationRequest.*"([^"]+)"\].*(?:concurrent|overlaps).*\[MedicationRequest.*"([^"]+)"\]'
    
    for match in re.finditer(pattern, cql_content, re.DOTALL):
        interactions.append({
            'drug_a': match.group(1),
            'drug_b': match.group(2),
            'interaction_type': 'CONTRAINDICATED_CONCURRENT',
            'source': 'CQL_LOGIC',
            'severity': 'HIGH'
        })
    
    return interactions
```

---

## Part 3: Atomiser Design (Gaps Only)

### 3.1 When Atomiser is Invoked

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                      ATOMISER INVOCATION CRITERIA                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  INVOKE ATOMISER ONLY WHEN:                                                 │
│                                                                             │
│  1. ✗ No existing CQL (checked CMS, CDC, WHO, AHRQ)                        │
│  2. ✗ No SMART L3 available for domain                                     │
│  3. ✗ COR/LOE table extraction failed (ambiguous cells)                    │
│  4. ✗ Temporal constraints not in PlanDefinition.timing                    │
│  5. Content requires:                                                       │
│     • Complex conditional sequencing                                        │
│     • Dose titration logic                                                  │
│     • Multi-factor exception handling                                       │
│                                                                             │
│  NEVER INVOKE ATOMISER FOR:                                                 │
│                                                                             │
│  • Content available in Tier 1-3 assets                                    │
│  • Simple COR/LOE extraction (use regex)                                   │
│  • Temporal constraints already in FHIR timing[x]                          │
│  • Evidence metadata in relatedArtifact                                    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Constrained Atomiser Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    CONSTRAINED ATOMISER PIPELINE                             │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  INPUT                                                                      │
│  ─────                                                                      │
│  • Specific text chunk (NOT entire guideline)                              │
│  • Pre-identified as "gap" by registry lookup                              │
│  • Tagged with expected output type                                        │
│                                                                             │
│  EXTRACTION                                                                 │
│  ──────────                                                                │
│  • Strict JSON schema (no free-form output)                                │
│  • Confidence scores required                                              │
│  • Uncertainty flagged explicitly                                          │
│                                                                             │
│  VALIDATION                                                                 │
│  ──────────                                                                │
│  • Schema validation (must conform)                                        │
│  • Terminology validation (concepts must exist in KB-7)                    │
│  • Consistency check (no conflicts with imported CQL)                      │
│                                                                             │
│  GOVERNANCE                                                                 │
│  ──────────                                                                │
│  • ALL LLM output → DRAFT status                                           │
│  • Mandatory SME review                                                    │
│  • Confidence ceiling: 0.85 (cannot self-certify higher)                   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.3 Atomiser Extraction Schemas

#### 3.3.1 Titration Sequence Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "TitrationSequenceExtraction",
  "type": "object",
  "required": ["medication", "starting_dose", "target_dose", "titration_steps", "confidence"],
  "properties": {
    "medication": {
      "type": "string",
      "description": "Medication being titrated"
    },
    "starting_dose": {
      "type": "object",
      "properties": {
        "value": {"type": "number"},
        "unit": {"type": "string"},
        "frequency": {"type": "string"}
      }
    },
    "target_dose": {
      "type": "object",
      "properties": {
        "value": {"type": "number"},
        "unit": {"type": "string"},
        "frequency": {"type": "string"}
      }
    },
    "titration_steps": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "step_number": {"type": "integer"},
          "dose": {"type": "object"},
          "duration_at_dose": {"type": "string"},
          "escalation_criteria": {"type": "string"},
          "hold_criteria": {"type": "array", "items": {"type": "string"}}
        }
      }
    },
    "confidence": {
      "type": "number",
      "minimum": 0,
      "maximum": 0.85
    },
    "uncertainty_flags": {
      "type": "array",
      "items": {"type": "string"}
    }
  }
}
```

#### 3.3.2 Conditional Exception Schema

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "ConditionalExceptionExtraction",
  "type": "object",
  "required": ["base_recommendation", "exceptions", "confidence"],
  "properties": {
    "base_recommendation": {
      "type": "string",
      "description": "The standard recommendation"
    },
    "exceptions": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "condition": {"type": "string"},
          "modified_recommendation": {"type": "string"},
          "rationale": {"type": "string"}
        }
      }
    },
    "confidence": {
      "type": "number",
      "maximum": 0.85
    }
  }
}
```

### 3.4 Atomiser Governance Rules

```python
class AtomiserGovernance:
    """
    Governance constraints for all Atomiser output.
    """
    
    MAX_CONFIDENCE = 0.85  # LLM cannot self-certify higher
    
    def process_extraction(self, extraction: dict) -> dict:
        # Force confidence ceiling
        if extraction.get('confidence', 0) > self.MAX_CONFIDENCE:
            extraction['confidence'] = self.MAX_CONFIDENCE
            extraction['confidence_capped'] = True
        
        # Set status to DRAFT (always)
        extraction['status'] = 'DRAFT'
        extraction['requires_sme_review'] = True
        
        # Track provenance
        extraction['provenance'] = {
            'extraction_method': 'ATOMISER_LLM',
            'model': 'claude-opus-4-5-20250514',
            'timestamp': datetime.utcnow().isoformat(),
            'human_reviewed': False
        }
        
        return extraction
```

---

## Part 4: Dual Pipeline Architecture

### 4.1 Complete Flow Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                    VAIDSHALA GUIDELINE INGESTION ARCHITECTURE               │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    MEASURE → GUIDELINE PIPELINE                      │   │
│  │                    (Baseline / Floor)                                │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  CMS eCQM / CDC IG / WHO SMART                                      │   │
│  │         │                                                           │   │
│  │         ▼                                                           │   │
│  │  ┌─────────────────────┐                                            │   │
│  │  │ Asset Registry      │ ◄── Check: Does CQL exist?                │   │
│  │  │ Lookup              │                                            │   │
│  │  └─────────┬───────────┘                                            │   │
│  │            │ YES                                                    │   │
│  │            ▼                                                        │   │
│  │  ┌─────────────────────┐     ┌─────────────────────┐               │   │
│  │  │ Direct Import       │ ──▶ │ CQL Tier 4b        │               │   │
│  │  │ + Namespace Adapt   │     │ (High confidence)   │               │   │
│  │  └─────────────────────┘     └─────────────────────┘               │   │
│  │            │                          │                             │   │
│  │            ▼                          ▼                             │   │
│  │  ┌─────────────────────┐     ┌─────────────────────┐               │   │
│  │  │ Parse PlanDef       │ ──▶ │ KB-3 Temporal       │               │   │
│  │  │ timing[x]           │     │ (From FHIR timing)  │               │   │
│  │  └─────────────────────┘     └─────────────────────┘               │   │
│  │            │                          │                             │   │
│  │            ▼                          ▼                             │   │
│  │  ┌─────────────────────┐     ┌─────────────────────┐               │   │
│  │  │ Parse relatedArtifact│ ──▶│ KB-15 Evidence      │               │   │
│  │  │ documentation       │     │ (From FHIR docs)    │               │   │
│  │  └─────────────────────┘     └─────────────────────┘               │   │
│  │                                        │                            │   │
│  │                                        ▼                            │   │
│  │                               ┌─────────────────────┐               │   │
│  │                               │ KB-19 Orchestrator  │               │   │
│  │                               │ (Low-Med confidence)│               │   │
│  │                               └─────────────────────┘               │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│                           ┌───────────────┐                                 │
│                           │  REGISTRY     │                                 │
│                           │  LOOKUP: NO   │                                 │
│                           │  CQL EXISTS   │                                 │
│                           └───────┬───────┘                                 │
│                                   │                                         │
│                                   ▼                                         │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                    GUIDELINE → CQL PIPELINE                         │   │
│  │                    (Ceiling / Specialist)                           │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │                                                                     │   │
│  │  ┌─────────────────────┐                                            │   │
│  │  │ PDF/DOCX Guideline  │                                            │   │
│  │  │ (Bronze Tier)       │                                            │   │
│  │  └─────────┬───────────┘                                            │   │
│  │            │                                                        │   │
│  │            ▼                                                        │   │
│  │  ┌─────────────────────┐     ┌─────────────────────┐               │   │
│  │  │ Table Extraction    │ ──▶ │ COR/LOE → KB-15    │               │   │
│  │  │ (Deterministic)     │     │ (95% coverage)      │               │   │
│  │  └─────────────────────┘     └─────────────────────┘               │   │
│  │            │                                                        │   │
│  │            │ Gaps identified                                        │   │
│  │            ▼                                                        │   │
│  │  ┌─────────────────────┐                                            │   │
│  │  │ ATOMISER (LLM)      │ ◄── ONLY for gaps (~10-15%)               │   │
│  │  │ Constrained Schema  │                                            │   │
│  │  └─────────┬───────────┘                                            │   │
│  │            │                                                        │   │
│  │            ▼                                                        │   │
│  │  ┌─────────────────────┐     ┌─────────────────────┐               │   │
│  │  │ Validation +        │ ──▶ │ DRAFT Status        │               │   │
│  │  │ SME Review Queue    │     │ → KB-18 Governance  │               │   │
│  │  └─────────────────────┘     └─────────────────────┘               │   │
│  │                                        │                            │   │
│  │                                        ▼                            │   │
│  │                               ┌─────────────────────┐               │   │
│  │                               │ KB-19 Orchestrator  │               │   │
│  │                               │ (High confidence)   │               │   │
│  │                               └─────────────────────┘               │   │
│  │                                                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
│                    ┌───────────────────────────────────┐                   │
│                    │        KB-19 ARBITRATION           │                   │
│                    ├───────────────────────────────────┤                   │
│                    │                                   │                   │
│                    │  IF both_present:                 │                   │
│                    │    → Guideline_CQL WINS           │                   │
│                    │                                   │                   │
│                    │  IF only_measure:                 │                   │
│                    │    → Allowed with warning         │                   │
│                    │    → "Baseline only - see notes"  │                   │
│                    │                                   │                   │
│                    │  IF jurisdiction_conflict:        │                   │
│                    │    → Apply preference rules       │                   │
│                    │                                   │                   │
│                    └───────────────────────────────────┘                   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 4.2 Arbitration Rules

```python
class KB19Arbitrator:
    """
    Arbitration logic for conflicting recommendations.
    Implements: Guideline > Measure, with jurisdiction awareness.
    """
    
    def arbitrate(self, patient: Patient, recommendations: List[Recommendation]) -> Recommendation:
        # Group by topic
        by_topic = group_by_topic(recommendations)
        
        final = []
        for topic, recs in by_topic.items():
            # Separate by source type
            measure_recs = [r for r in recs if r.source_type == 'MEASURE']
            guideline_recs = [r for r in recs if r.source_type == 'GUIDELINE']
            
            if guideline_recs:
                # Apply jurisdiction preference
                preferred = self.apply_jurisdiction_preference(patient, guideline_recs)
                preferred.override_source = 'GUIDELINE'
                final.append(preferred)
            elif measure_recs:
                # Use measure-derived with warning
                rec = max(measure_recs, key=lambda r: r.confidence)
                rec.warning = "Baseline recommendation only. Guideline-specific therapy selection not available."
                rec.confidence_adjusted = rec.confidence * 0.8  # Penalty for measure-only
                final.append(rec)
        
        return final
    
    def apply_jurisdiction_preference(self, patient: Patient, recs: List) -> Recommendation:
        """
        US patients prefer CDC/ACC/AHA over WHO.
        Global/low-resource prefer WHO.
        """
        if patient.location.country == 'US':
            preference_order = ['CDC', 'ACC', 'AHA', 'AHRQ', 'WHO']
        elif patient.location.resource_setting == 'LOW':
            preference_order = ['WHO', 'CDC', 'ACC']
        else:
            preference_order = ['LOCAL', 'WHO', 'CDC']
        
        for pref in preference_order:
            matching = [r for r in recs if pref in r.source_authority]
            if matching:
                return max(matching, key=lambda r: r.confidence)
        
        return max(recs, key=lambda r: r.confidence)
```

---

## Part 5: Implementation Roadmap

### Phase A: Asset Registry + Foundation Import (Weeks 1-3)

| Week | Task | Deliverable | LLM Use |
|------|------|-------------|---------|
| 1 | Build Asset Registry database | Registry schema + initial population | None |
| 1 | Clone CQF repositories | Local copies of all source repos | None |
| 2 | Import foundation libraries | FHIRHelpers, QICoreCommon in tier-0/1 | None |
| 2 | Import VTE measures | CMS108, CMS190 adapted | None |
| 3 | Import Diabetes measures | CMS122, CMS134, CMS131 adapted | None |
| 3 | Import CDC Opioid IG | Complete opioid module | None |

**Exit Criteria:** ~60 CQL files imported, compiling, tests passing

### Phase B: SMART Guidelines + Temporal Extraction (Weeks 4-5)

| Week | Task | Deliverable | LLM Use |
|------|------|-------------|---------|
| 4 | Clone WHO SMART repos | HIV, Immunization, ANC L3 | None |
| 4 | Build PlanDefinition parser | Temporal + evidence extraction | None |
| 5 | Import SMART CQL | WHO domain libraries | None |
| 5 | Generate KB-3 from timing[x] | Temporal constraints SQL | None |

**Exit Criteria:** +30 CQL files, KB-3 populated with FHIR-derived temporals

### Phase C: Table Extraction + Gap Analysis (Weeks 6-7)

| Week | Task | Deliverable | LLM Use |
|------|------|-------------|---------|
| 6 | Build COR/LOE table extractor | Regex-based PDF parser | None |
| 6 | Process ACC/AHA HF guideline | KB-15 evidence entries | <5% |
| 7 | Process SSC 2021 | Sepsis recommendations | <5% |
| 7 | Gap analysis report | Quantified remaining gaps | None |

**Exit Criteria:** KB-15 populated, gaps identified (<15% remaining)

### Phase D: Constrained Atomiser (Weeks 8+)

| Week | Task | Deliverable | LLM Use |
|------|------|-------------|---------|
| 8+ | Titration sequences | HF GDMT titration CQL | Targeted |
| 8+ | Complex exceptions | Sepsis fluid responsiveness | Targeted |
| 8+ | SME review workflow | Governance integration | None |

**Exit Criteria:** ~90% coverage, all LLM content SME-reviewed

---

## Part 6: Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Registry completeness | 100% of known sources cataloged | Asset count |
| Direct import coverage | >70% of priority guidelines | CQL file count |
| LLM extraction | <15% of total content | Extraction logs |
| CQL compilation | 100% success | CI/CD pipeline |
| KB-3 temporal coverage | >80% of actionable protocols | Constraint count |
| KB-15 evidence linkage | 100% of recommendations | Audit query |
| SME review completion | 100% of LLM-derived content | Governance queue |
| Time to first CQL (new guideline) | <4 hours | Pipeline metrics |

---

## Appendix A: Repository URLs

### Primary Sources

| Source | URL | Type |
|--------|-----|------|
| CMS eCQM 2024 | github.com/cqframework/ecqm-content-qicore-2024 | Platinum |
| CMS eCQM 2025 | github.com/cqframework/ecqm-content-cms-2025 | Platinum |
| CDC Opioid R4 | github.com/cqframework/opioid-cds-r4 | Platinum |
| CDC MME Calculator | fhir.org/guides/cdc/opioid-mme-r4 | Platinum |
| WHO SMART HIV | github.com/WorldHealthOrganization/smart-hiv | Gold |
| WHO SMART Immunization | github.com/WorldHealthOrganization/smart-immunizations | Gold |
| WHO SMART ANC | github.com/WorldHealthOrganization/smart-anc | Gold |
| CPG-on-FHIR | github.com/HL7/cqf-recommendations | Reference |
| CQF Common | github.com/cqframework/cqf-common | Foundation |
| AHRQ CDS (archived) | github.com/AHRQ-CDS | Silver |

### Tooling

| Tool | URL | Purpose |
|------|-----|---------|
| CQL Compiler | github.com/cqframework/clinical_quality_language | Validation |
| CQL Engine (Java) | github.com/cqframework/cql-engine | Execution |
| CQL Engine (JS) | github.com/cqframework/cql-execution | Browser execution |
| CQF Tooling | github.com/cqframework/cqf-tooling | IG packaging |
| VS Code CQL | VS Code extension | Authoring |
| CQF Ruler | HAPI FHIR + CQL | Reference server |

---

## Appendix B: Mental Model Summary

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                                                                             │
│     "Measures define the clinical FLOOR"                                    │
│     "Guidelines define the clinical CEILING"                                │
│     "Atomiser fills the gap between them"                                   │
│                                                                             │
│     ─────────────────────────────────────────────────────────────────────   │
│                                                                             │
│     FLOOR (Measures)          │  What must be true (threshold, binary)     │
│                               │  Coverage: 40-55%                          │
│                               │  LLM: None                                  │
│     ─────────────────────────────────────────────────────────────────────   │
│     MIDDLE (SMART/AHRQ)       │  Standard protocols (machine-readable)     │
│                               │  Coverage: +20-35%                          │
│                               │  LLM: None                                  │
│     ─────────────────────────────────────────────────────────────────────   │
│     CEILING (Guidelines)      │  How to get there (therapy, sequence)      │
│                               │  Coverage: +10-15% (Bronze tables)         │
│                               │  LLM: <5% for ambiguous                    │
│     ─────────────────────────────────────────────────────────────────────   │
│     GAPS (Atomiser)           │  Complex sequencing, titration             │
│                               │  Coverage: 10-15%                          │
│                               │  LLM: Targeted, constrained                │
│                                                                             │
│     ═══════════════════════════════════════════════════════════════════    │
│     TOTAL COVERAGE: ~90-95% with <15% LLM exposure                         │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

**Document Status:** READY FOR IMPLEMENTATION

This architecture ensures Vaidshala:
1. **Exhausts existing CQL** before any LLM involvement
2. **Maps assets deterministically** to the appropriate KB
3. **Constrains Atomiser** to genuine gaps with mandatory governance
4. **Arbitrates conflicts** favoring authoritative guideline sources
