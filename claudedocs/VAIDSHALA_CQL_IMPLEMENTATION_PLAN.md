# Vaidshala CQL Repository: Actionable Implementation Plan

**Version:** 1.0
**Date:** January 2026
**Status:** READY FOR EXECUTION
**Philosophy:** Deterministic First, LLM as Last Resort

---

## Executive Summary

This document provides an **actionable, prioritized implementation plan** to close the gaps between the Vaidshala CQL architecture specifications and the current implementation state.

### Current State vs Target

| Category | Current | Target | Gap |
|----------|---------|--------|-----|
| CMS eCQM Measures | 4 | 10 | 6 missing |
| CDC Opioid IG | 0 | 13 | 13 missing |
| WHO SMART CQL | 0 | ~30 | All missing |
| Terminology Layer | 5% | 80% | Critical blocker |
| Table Extractor | 0% | 100% | Not built |
| Atomiser Pipeline | 0% | 100% | Not built |
| Runtime Services | 1/6 | 6/6 | 5 placeholders |

### Implementation Phases

| Phase | Focus | Duration | LLM Use | Priority |
|-------|-------|----------|---------|----------|
| **1** | Foundation Imports | 1 week | None | P0 |
| **2** | WHO SMART Harvest | 1 week | None | P0 |
| **3** | Terminology Population | 1 week | None | P0 |
| **4** | Table Extraction Pipeline | 1 week | <5% | P1 |
| **5** | Atomiser Implementation | 2 weeks | Targeted | P2 |

---

## Phase 1: Foundation Imports (Zero LLM)

### 1.1 Import CDC Opioid IG (Gold Standard)

**Why First:** The CDC Opioid IG is the most complete CPG→CQL implementation. It serves as the **reference pattern** for all other guideline implementations.

**Repository:** `github.com/cqframework/opioid-cds-r4`

**Tasks:**

```bash
# Task 1.1.1: Clone CDC Opioid Repository
git clone https://github.com/cqframework/opioid-cds-r4.git /tmp/opioid-cds-r4

# Task 1.1.2: Create target directory
mkdir -p vaidshala/clinical-knowledge-core/tier-4-guidelines/cdc-opioid

# Task 1.1.3: Copy CQL files
cp /tmp/opioid-cds-r4/input/cql/*.cql \
   vaidshala/clinical-knowledge-core/tier-4-guidelines/cdc-opioid/

# Task 1.1.4: Copy PlanDefinitions (for KB-3 temporal extraction)
mkdir -p vaidshala/clinical-knowledge-core/tier-4-guidelines/cdc-opioid/plandefinitions
cp /tmp/opioid-cds-r4/input/resources/plandefinition/*.json \
   vaidshala/clinical-knowledge-core/tier-4-guidelines/cdc-opioid/plandefinitions/

# Task 1.1.5: Copy ValueSets (for KB-7)
mkdir -p vaidshala/clinical-knowledge-core/tier-0.5-terminology/valuesets/cdc-opioid
cp /tmp/opioid-cds-r4/input/vocabulary/valueset/*.json \
   vaidshala/clinical-knowledge-core/tier-0.5-terminology/valuesets/cdc-opioid/
```

**Files to Import:**

| File | Purpose | Lines |
|------|---------|-------|
| OpioidCDSCommon.cql | Shared functions | ~300 |
| OpioidCDSREC01.cql | Non-pharmacologic therapy first | ~150 |
| OpioidCDSREC02.cql | Immediate-release opioids | ~150 |
| OpioidCDSREC03.cql | Lowest effective dosage | ~150 |
| OpioidCDSREC04.cql | Extended-release caution | ~150 |
| OpioidCDSREC05.cql | MME thresholds (50/90) | ~200 |
| OpioidCDSREC06.cql | Duration limits | ~150 |
| OpioidCDSREC07.cql | Risk evaluation | ~150 |
| OpioidCDSREC08.cql | Naloxone consideration | ~150 |
| OpioidCDSREC09.cql | PDMP review | ~150 |
| OpioidCDSREC10.cql | Urine drug testing | ~150 |
| OpioidCDSREC11.cql | Benzodiazepine caution | ~150 |
| OpioidCDSREC12.cql | MAT for OUD | ~150 |
| OMTKLogic.cql | MME calculation engine | ~500 |

**Validation:**
```bash
# Verify CQL syntax (requires CQL compiler)
for f in vaidshala/clinical-knowledge-core/tier-4-guidelines/cdc-opioid/*.cql; do
  echo "Validating: $f"
  # cql-compiler --validate "$f" || echo "FAILED: $f"
done
```

---

### 1.2 Import Missing CMS eCQM Measures

**Repository:** `github.com/cqframework/ecqm-content-qicore-2024`

**Current State:** CMS2, CMS122, CMS134, CMS165 exist
**Missing:** CMS108, CMS190, CMS131, CMS144, CMS145, CMS71, CMS347, CMS506

**Tasks:**

```bash
# Task 1.2.1: Clone eCQM Repository
git clone https://github.com/cqframework/ecqm-content-qicore-2024.git /tmp/ecqm-2024

# Task 1.2.2: List available measures
ls /tmp/ecqm-2024/input/cql/ | grep -E "^[A-Z]" | head -20

# Task 1.2.3: Import VTE Measures (P0)
cp /tmp/ecqm-2024/input/cql/VenousThromboembolismProphylaxisFHIR.cql \
   vaidshala/clinical-knowledge-core/tier-4-guidelines/cms-ecqm/CMS108/

cp /tmp/ecqm-2024/input/cql/IntensiveCareUnitVenousThromboembolismProphylaxisFHIR.cql \
   vaidshala/clinical-knowledge-core/tier-4-guidelines/cms-ecqm/CMS190/

# Task 1.2.4: Import Cardiovascular Measures (P1)
mkdir -p vaidshala/clinical-knowledge-core/tier-4-guidelines/cms-ecqm/CMS144
mkdir -p vaidshala/clinical-knowledge-core/tier-4-guidelines/cms-ecqm/CMS145
mkdir -p vaidshala/clinical-knowledge-core/tier-4-guidelines/cms-ecqm/CMS347

cp /tmp/ecqm-2024/input/cql/HeartFailureBetaBlockerTherapyforLVSDFHIR.cql \
   vaidshala/clinical-knowledge-core/tier-4-guidelines/cms-ecqm/CMS144/

cp /tmp/ecqm-2024/input/cql/CoronaryArteryDiseaseBetaBlockerTherapyPriorMIorLVSDFHIR.cql \
   vaidshala/clinical-knowledge-core/tier-4-guidelines/cms-ecqm/CMS145/

cp /tmp/ecqm-2024/input/cql/StatinTherapyforthePreventionandTreatmentofCardiovascularDiseaseFHIR.cql \
   vaidshala/clinical-knowledge-core/tier-4-guidelines/cms-ecqm/CMS347/

# Task 1.2.5: Import AFib Measure
mkdir -p vaidshala/clinical-knowledge-core/tier-4-guidelines/cms-ecqm/CMS71
cp /tmp/ecqm-2024/input/cql/AnticoagulationTherapyforAtrialFibrillationFlutterFHIR.cql \
   vaidshala/clinical-knowledge-core/tier-4-guidelines/cms-ecqm/CMS71/

# Task 1.2.6: Import Diabetes Eye Exam
mkdir -p vaidshala/clinical-knowledge-core/tier-4-guidelines/cms-ecqm/CMS131
cp /tmp/ecqm-2024/input/cql/DiabetesEyeExamFHIR.cql \
   vaidshala/clinical-knowledge-core/tier-4-guidelines/cms-ecqm/CMS131/
```

**Import Manifest:**

| CMS ID | Measure Name | Source File | Target Directory |
|--------|--------------|-------------|------------------|
| CMS108 | VTE Prophylaxis | VenousThromboembolismProphylaxisFHIR.cql | cms-ecqm/CMS108/ |
| CMS190 | ICU VTE | IntensiveCareUnitVenousThromboembolismProphylaxisFHIR.cql | cms-ecqm/CMS190/ |
| CMS131 | Diabetes Eye Exam | DiabetesEyeExamFHIR.cql | cms-ecqm/CMS131/ |
| CMS144 | HF Beta-Blocker | HeartFailureBetaBlockerTherapyforLVSDFHIR.cql | cms-ecqm/CMS144/ |
| CMS145 | CAD Beta-Blocker | CoronaryArteryDiseaseBetaBlockerTherapyPriorMIorLVSDFHIR.cql | cms-ecqm/CMS145/ |
| CMS71 | AFib Anticoag | AnticoagulationTherapyforAtrialFibrillationFlutterFHIR.cql | cms-ecqm/CMS71/ |
| CMS347 | Statin Therapy | StatinTherapyforthePreventionandTreatmentofCardiovascularDiseaseFHIR.cql | cms-ecqm/CMS347/ |

---

### 1.3 Import Foundation Libraries

**Repository:** `github.com/cqframework/cqf-common`

**Current State:** FHIRHelpers exists, but may be outdated
**Task:** Verify and update foundation libraries

```bash
# Task 1.3.1: Clone CQF Common
git clone https://github.com/cqframework/cqf-common.git /tmp/cqf-common

# Task 1.3.2: Compare and update FHIRHelpers
diff vaidshala/clinical-knowledge-core/tier-0-fhir/helpers/FHIRHelpers.cql \
     /tmp/cqf-common/input/cql/FHIRHelpers.cql

# Task 1.3.3: Import FHIRCommon (if missing)
cp /tmp/cqf-common/input/cql/FHIRCommon.cql \
   vaidshala/clinical-knowledge-core/tier-0-fhir/helpers/

# Task 1.3.4: Update vendor libraries
cp /tmp/ecqm-2024/input/cql/MATGlobalCommonFunctionsFHIR.cql \
   vaidshala/clinical-knowledge-core/tier-2-cqm-infra/vendor/
```

---

### Phase 1 Validation Checklist

```markdown
[ ] CDC Opioid IG cloned
[ ] 14 CQL files copied to tier-4-guidelines/cdc-opioid/
[ ] PlanDefinitions copied for KB-3 extraction
[ ] ValueSets copied for KB-7
[ ] CMS108 (VTE) imported
[ ] CMS190 (ICU VTE) imported
[ ] CMS131 (Diabetes Eye) imported
[ ] CMS144 (HF Beta-Blocker) imported
[ ] CMS145 (CAD Beta-Blocker) imported
[ ] CMS71 (AFib) imported
[ ] CMS347 (Statin) imported
[ ] Foundation libraries updated
[ ] All CQL files compile without errors
```

---

## Phase 2: WHO SMART Guidelines Harvest (Zero LLM)

### 2.1 Import WHO SMART ANC (Antenatal Care)

**Repository:** `github.com/WorldHealthOrganization/smart-anc`
**Status:** Published (most mature)

```bash
# Task 2.1.1: Clone SMART ANC
git clone https://github.com/WorldHealthOrganization/smart-anc.git /tmp/smart-anc

# Task 2.1.2: Create target directory
mkdir -p vaidshala/clinical-knowledge-core/tier-4-guidelines/who/anc

# Task 2.1.3: Copy CQL files
cp /tmp/smart-anc/input/cql/*.cql \
   vaidshala/clinical-knowledge-core/tier-4-guidelines/who/anc/

# Task 2.1.4: Copy PlanDefinitions for KB-3
mkdir -p vaidshala/clinical-knowledge-core/tier-4-guidelines/who/anc/plandefinitions
cp /tmp/smart-anc/input/resources/plandefinition/*.json \
   vaidshala/clinical-knowledge-core/tier-4-guidelines/who/anc/plandefinitions/

# Task 2.1.5: Copy ValueSets for KB-7
mkdir -p vaidshala/clinical-knowledge-core/tier-0.5-terminology/valuesets/who-anc
cp /tmp/smart-anc/input/vocabulary/valueset/*.json \
   vaidshala/clinical-knowledge-core/tier-0.5-terminology/valuesets/who-anc/

# Task 2.1.6: Add jurisdiction header to CQL files
for f in vaidshala/clinical-knowledge-core/tier-4-guidelines/who/anc/*.cql; do
  sed -i '' '1i\
// Jurisdiction: WHO/Global\
// Adaptation Required: YES - verify local formulary and protocols\
' "$f"
done
```

### 2.2 Import WHO SMART Immunization

**Repository:** `github.com/WorldHealthOrganization/smart-immunizations`

```bash
# Task 2.2.1: Clone SMART Immunization
git clone https://github.com/WorldHealthOrganization/smart-immunizations.git /tmp/smart-imm

# Task 2.2.2: Create target and copy
mkdir -p vaidshala/clinical-knowledge-core/tier-4-guidelines/who/immunization
cp /tmp/smart-imm/input/cql/*.cql \
   vaidshala/clinical-knowledge-core/tier-4-guidelines/who/immunization/

# Task 2.2.3: Copy PlanDefinitions
mkdir -p vaidshala/clinical-knowledge-core/tier-4-guidelines/who/immunization/plandefinitions
cp /tmp/smart-imm/input/resources/plandefinition/*.json \
   vaidshala/clinical-knowledge-core/tier-4-guidelines/who/immunization/plandefinitions/

# Task 2.2.4: Copy ValueSets
mkdir -p vaidshala/clinical-knowledge-core/tier-0.5-terminology/valuesets/who-immunization
cp /tmp/smart-imm/input/vocabulary/valueset/*.json \
   vaidshala/clinical-knowledge-core/tier-0.5-terminology/valuesets/who-immunization/
```

### 2.3 Import WHO SMART HIV

**Repository:** `github.com/WorldHealthOrganization/smart-hiv`

```bash
# Task 2.3.1: Clone SMART HIV
git clone https://github.com/WorldHealthOrganization/smart-hiv.git /tmp/smart-hiv

# Task 2.3.2: Create target and copy
mkdir -p vaidshala/clinical-knowledge-core/tier-4-guidelines/who/hiv
cp /tmp/smart-hiv/input/cql/*.cql \
   vaidshala/clinical-knowledge-core/tier-4-guidelines/who/hiv/

# Task 2.3.3: Copy supporting resources
mkdir -p vaidshala/clinical-knowledge-core/tier-4-guidelines/who/hiv/plandefinitions
cp /tmp/smart-hiv/input/resources/plandefinition/*.json \
   vaidshala/clinical-knowledge-core/tier-4-guidelines/who/hiv/plandefinitions/ 2>/dev/null || true

mkdir -p vaidshala/clinical-knowledge-core/tier-0.5-terminology/valuesets/who-hiv
cp /tmp/smart-hiv/input/vocabulary/valueset/*.json \
   vaidshala/clinical-knowledge-core/tier-0.5-terminology/valuesets/who-hiv/ 2>/dev/null || true
```

---

### Phase 2 Validation Checklist

```markdown
[ ] WHO SMART ANC cloned and CQL imported
[ ] WHO SMART Immunization cloned and CQL imported
[ ] WHO SMART HIV cloned and CQL imported
[ ] All PlanDefinitions extracted for KB-3 temporal parsing
[ ] All ValueSets copied to tier-0.5-terminology
[ ] Jurisdiction headers added to WHO CQL files
[ ] CQL files compile with updated includes
```

---

## Phase 3: Terminology Layer Population (Zero LLM)

### 3.1 Create Terminology Import Scripts

The terminology layer (tier-0.5) is currently 95% empty. This blocks ValueSet resolution for all imported CQL.

**Script: `scripts/import_terminology.py`**

```python
#!/usr/bin/env python3
"""
Import ValueSets from CQL source repositories into KB-7 compatible format.
Deterministic - no LLM.
"""

import json
import os
from pathlib import Path
from typing import List, Dict

class TerminologyImporter:
    def __init__(self, source_dir: str, target_dir: str):
        self.source_dir = Path(source_dir)
        self.target_dir = Path(target_dir)

    def import_valuesets(self) -> List[Dict]:
        """Import all ValueSet JSON files from source."""
        imported = []

        for vs_file in self.source_dir.glob("**/*.json"):
            try:
                with open(vs_file) as f:
                    vs = json.load(f)

                if vs.get("resourceType") == "ValueSet":
                    # Transform to KB-7 format
                    kb7_vs = self._transform_to_kb7(vs)

                    # Write to target
                    target_file = self.target_dir / f"{vs['id']}.json"
                    target_file.parent.mkdir(parents=True, exist_ok=True)

                    with open(target_file, 'w') as f:
                        json.dump(kb7_vs, f, indent=2)

                    imported.append({
                        'id': vs['id'],
                        'name': vs.get('name'),
                        'url': vs.get('url'),
                        'source': str(vs_file)
                    })

            except Exception as e:
                print(f"Error processing {vs_file}: {e}")

        return imported

    def _transform_to_kb7(self, vs: Dict) -> Dict:
        """Transform FHIR ValueSet to KB-7 format."""
        return {
            "resourceType": "ValueSet",
            "id": vs.get("id"),
            "url": vs.get("url", "").replace(
                "http://cts.nlm.nih.gov/fhir/ValueSet/",
                "http://vaidshala.io/kb7/valueset/"
            ),
            "identifier": vs.get("identifier", []),
            "version": vs.get("version"),
            "name": vs.get("name"),
            "title": vs.get("title"),
            "status": vs.get("status", "active"),
            "compose": vs.get("compose", {}),
            "expansion": vs.get("expansion", {}),
            "_kb7_metadata": {
                "imported_from": vs.get("url"),
                "import_timestamp": "2026-01-28T00:00:00Z",
                "ohdsi_mapped": False
            }
        }


if __name__ == "__main__":
    # Import from all source directories
    sources = [
        ("/tmp/opioid-cds-r4/input/vocabulary/valueset", "cdc-opioid"),
        ("/tmp/ecqm-2024/input/vocabulary/valueset", "cms-ecqm"),
        ("/tmp/smart-anc/input/vocabulary/valueset", "who-anc"),
        ("/tmp/smart-imm/input/vocabulary/valueset", "who-immunization"),
    ]

    base_target = Path("vaidshala/clinical-knowledge-core/tier-0.5-terminology/valuesets")

    for source_dir, target_subdir in sources:
        if Path(source_dir).exists():
            importer = TerminologyImporter(
                source_dir,
                base_target / target_subdir
            )
            results = importer.import_valuesets()
            print(f"Imported {len(results)} ValueSets from {source_dir}")
```

### 3.2 Create CodeSystem Stubs

```bash
# Task 3.2.1: Create SNOMED stub
cat > vaidshala/clinical-knowledge-core/tier-0.5-terminology/codesystems/snomed/SNOMED-CT.json << 'EOF'
{
  "resourceType": "CodeSystem",
  "id": "snomed-ct",
  "url": "http://snomed.info/sct",
  "identifier": [{
    "system": "urn:ietf:rfc:3986",
    "value": "urn:oid:2.16.840.1.113883.6.96"
  }],
  "name": "SNOMED_CT",
  "title": "SNOMED Clinical Terms",
  "status": "active",
  "content": "not-present",
  "_kb7_metadata": {
    "resolution_strategy": "external_terminology_server",
    "terminology_server_url": "https://tx.fhir.org/r4"
  }
}
EOF

# Task 3.2.2: Create LOINC stub
cat > vaidshala/clinical-knowledge-core/tier-0.5-terminology/codesystems/loinc/LOINC.json << 'EOF'
{
  "resourceType": "CodeSystem",
  "id": "loinc",
  "url": "http://loinc.org",
  "identifier": [{
    "system": "urn:ietf:rfc:3986",
    "value": "urn:oid:2.16.840.1.113883.6.1"
  }],
  "name": "LOINC",
  "title": "Logical Observation Identifiers Names and Codes",
  "status": "active",
  "content": "not-present",
  "_kb7_metadata": {
    "resolution_strategy": "external_terminology_server",
    "terminology_server_url": "https://tx.fhir.org/r4"
  }
}
EOF

# Task 3.2.3: Create RxNorm stub
cat > vaidshala/clinical-knowledge-core/tier-0.5-terminology/codesystems/rxnorm/RxNorm.json << 'EOF'
{
  "resourceType": "CodeSystem",
  "id": "rxnorm",
  "url": "http://www.nlm.nih.gov/research/umls/rxnorm",
  "identifier": [{
    "system": "urn:ietf:rfc:3986",
    "value": "urn:oid:2.16.840.1.113883.6.88"
  }],
  "name": "RxNorm",
  "title": "RxNorm",
  "status": "active",
  "content": "not-present",
  "_kb7_metadata": {
    "resolution_strategy": "external_terminology_server",
    "terminology_server_url": "https://tx.fhir.org/r4"
  }
}
EOF

# Task 3.2.4: Create ICD-10 stub
cat > vaidshala/clinical-knowledge-core/tier-0.5-terminology/codesystems/icd10/ICD10CM.json << 'EOF'
{
  "resourceType": "CodeSystem",
  "id": "icd10cm",
  "url": "http://hl7.org/fhir/sid/icd-10-cm",
  "identifier": [{
    "system": "urn:ietf:rfc:3986",
    "value": "urn:oid:2.16.840.1.113883.6.90"
  }],
  "name": "ICD10CM",
  "title": "ICD-10 Clinical Modification",
  "status": "active",
  "content": "not-present",
  "_kb7_metadata": {
    "resolution_strategy": "external_terminology_server",
    "terminology_server_url": "https://tx.fhir.org/r4"
  }
}
EOF
```

---

## Phase 4: Table Extraction Pipeline (<5% LLM)

### 4.1 Create Deterministic COR/LOE Extractor

**File: `scripts/table_extractor.py`**

```python
#!/usr/bin/env python3
"""
Deterministic extraction of COR/LOE recommendation tables from PDF guidelines.
LLM used ONLY for genuinely ambiguous cells (<5% of content).
"""

import re
import json
from pathlib import Path
from typing import List, Dict, Optional, Tuple
from dataclasses import dataclass, asdict

# Requires: pip install pdfplumber

@dataclass
class RecommendationRow:
    cor: Optional[str]
    loe: Optional[str]
    recommendation_text: str
    temporal_constraints: List[str]
    needs_llm_review: bool
    confidence: float
    source_page: int
    source_guideline: str


class GuidelineTableExtractor:
    """
    Extract COR/LOE tables from PDF guidelines.
    Deterministic regex patterns - LLM only for ambiguous cases.
    """

    # ACC/AHA Class of Recommendation patterns
    COR_PATTERNS = [
        (r'(?:Class\s*)?I(?![IVab\d])', 'I'),
        (r'(?:Class\s*)?IIa', 'IIa'),
        (r'(?:Class\s*)?IIb', 'IIb'),
        (r'(?:Class\s*)?III.*?(?:Harm|harm)', 'III-Harm'),
        (r'(?:Class\s*)?III.*?(?:No\s*Benefit|no\s*benefit|NB)', 'III-NoBenefit'),
        (r'(?:Class\s*)?III(?![Ia-z])', 'III'),
    ]

    # Level of Evidence patterns
    LOE_PATTERNS = [
        (r'(?:LOE\s*|Level\s*)?A(?![a-z])', 'A'),
        (r'(?:LOE\s*|Level\s*)?B-?R(?:andomized)?', 'B-R'),
        (r'(?:LOE\s*|Level\s*)?B-?NR', 'B-NR'),
        (r'(?:LOE\s*|Level\s*)?C-?LD', 'C-LD'),
        (r'(?:LOE\s*|Level\s*)?C-?EO', 'C-EO'),
        (r'(?:LOE\s*|Level\s*)?B(?![a-zA-Z-])', 'B'),
        (r'(?:LOE\s*|Level\s*)?C(?![a-zA-Z-])', 'C'),
    ]

    # Temporal constraint patterns
    TEMPORAL_PATTERNS = [
        (r'within\s+(\d+)\s*(hours?|minutes?|days?)', 'DEADLINE'),
        (r'every\s+(\d+)\s*(hours?|days?|weeks?|months?)', 'RECURRING'),
        (r'(before|prior\s+to)\s+(\w+)', 'SEQUENCE_BEFORE'),
        (r'(after|following)\s+(\w+)', 'SEQUENCE_AFTER'),
        (r'(immediately|STAT|as\s+soon\s+as\s+possible|ASAP)', 'URGENT'),
        (r'(\d+)\s*(?:to|-)\s*(\d+)\s*(hours?|days?|weeks?)', 'RANGE'),
    ]

    def __init__(self, guideline_source: str):
        self.guideline_source = guideline_source

    def extract_from_pdf(self, pdf_path: str) -> List[RecommendationRow]:
        """Extract recommendation tables from PDF."""
        import pdfplumber

        recommendations = []

        with pdfplumber.open(pdf_path) as pdf:
            for page_num, page in enumerate(pdf.pages, 1):
                tables = page.extract_tables()

                for table in tables:
                    if self._is_recommendation_table(table):
                        rows = self._parse_table(table, page_num)
                        recommendations.extend(rows)

        return recommendations

    def _is_recommendation_table(self, table: List[List[str]]) -> bool:
        """Detect if table is a recommendation table by headers."""
        if not table or not table[0]:
            return False

        header_text = ' '.join(str(cell) for cell in table[0] if cell).lower()

        indicators = ['class', 'cor', 'recommendation', 'loe', 'level', 'evidence']
        return sum(1 for ind in indicators if ind in header_text) >= 2

    def _parse_table(self, table: List[List[str]], page_num: int) -> List[RecommendationRow]:
        """Parse recommendation table rows."""
        rows = []

        for row in table[1:]:  # Skip header
            if not row or all(cell is None or str(cell).strip() == '' for cell in row):
                continue

            parsed = self._parse_row(row, page_num)
            if parsed:
                rows.append(parsed)

        return rows

    def _parse_row(self, row: List[str], page_num: int) -> Optional[RecommendationRow]:
        """Parse a single recommendation row."""
        if len(row) < 2:
            return None

        # Try to extract COR from first column
        cor_text = str(row[0]) if row[0] else ''
        cor, cor_confidence = self._match_pattern(cor_text, self.COR_PATTERNS)

        # Try to extract LOE from second column
        loe_text = str(row[1]) if len(row) > 1 and row[1] else ''
        loe, loe_confidence = self._match_pattern(loe_text, self.LOE_PATTERNS)

        # Recommendation text from remaining columns
        rec_text = ' '.join(str(cell) for cell in row[2:] if cell).strip()
        if not rec_text and len(row) > 1:
            rec_text = str(row[-1]) if row[-1] else ''

        # Extract temporal constraints from recommendation text
        temporal = self._extract_temporal(rec_text)

        # Determine if LLM review needed
        needs_llm = (cor is None or loe is None) and rec_text
        confidence = min(cor_confidence, loe_confidence) if cor and loe else 0.5

        return RecommendationRow(
            cor=cor,
            loe=loe,
            recommendation_text=rec_text,
            temporal_constraints=temporal,
            needs_llm_review=needs_llm,
            confidence=confidence,
            source_page=page_num,
            source_guideline=self.guideline_source
        )

    def _match_pattern(self, text: str, patterns: List[Tuple[str, str]]) -> Tuple[Optional[str], float]:
        """Match text against patterns, return match and confidence."""
        text = text.strip()

        for pattern, value in patterns:
            if re.search(pattern, text, re.IGNORECASE):
                return value, 0.95

        return None, 0.0

    def _extract_temporal(self, text: str) -> List[str]:
        """Extract temporal constraints from recommendation text."""
        constraints = []

        for pattern, constraint_type in self.TEMPORAL_PATTERNS:
            matches = re.findall(pattern, text, re.IGNORECASE)
            for match in matches:
                if isinstance(match, tuple):
                    constraints.append(f"{constraint_type}: {' '.join(match)}")
                else:
                    constraints.append(f"{constraint_type}: {match}")

        return constraints

    def to_kb15_format(self, recommendations: List[RecommendationRow]) -> List[Dict]:
        """Convert to KB-15 Evidence Engine format."""
        kb15_entries = []

        for i, rec in enumerate(recommendations):
            kb15_entries.append({
                "recommendation_id": f"{self.guideline_source}-{i+1:03d}",
                "evidence_metadata": {
                    "class_of_recommendation": rec.cor,
                    "level_of_evidence": rec.loe,
                    "extraction_method": "DETERMINISTIC_TABLE" if not rec.needs_llm_review else "NEEDS_LLM_REVIEW",
                    "confidence": rec.confidence,
                    "llm_involvement": rec.needs_llm_review
                },
                "recommendation_text": rec.recommendation_text,
                "temporal_constraints": rec.temporal_constraints,
                "source": {
                    "guideline": self.guideline_source,
                    "page": rec.source_page
                },
                "status": "DRAFT" if rec.needs_llm_review else "PENDING_REVIEW"
            })

        return kb15_entries


def main():
    """Example usage."""
    import sys

    if len(sys.argv) < 3:
        print("Usage: python table_extractor.py <pdf_path> <guideline_name>")
        sys.exit(1)

    pdf_path = sys.argv[1]
    guideline_name = sys.argv[2]

    extractor = GuidelineTableExtractor(guideline_name)
    recommendations = extractor.extract_from_pdf(pdf_path)

    print(f"Extracted {len(recommendations)} recommendations")

    needs_llm = sum(1 for r in recommendations if r.needs_llm_review)
    print(f"Needs LLM review: {needs_llm} ({100*needs_llm/len(recommendations):.1f}%)")

    # Output to JSON
    kb15_entries = extractor.to_kb15_format(recommendations)
    output_file = Path(pdf_path).stem + "_kb15.json"

    with open(output_file, 'w') as f:
        json.dump(kb15_entries, f, indent=2)

    print(f"Output written to {output_file}")


if __name__ == "__main__":
    main()
```

---

## Phase 5: Constrained Atomiser Implementation (Targeted LLM)

### 5.1 Atomiser Architecture

Only invoke for content that:
1. Has no existing CQL (verified against registry)
2. Failed table extraction
3. Requires complex sequencing/titration logic

**File: `scripts/atomiser/constrained_atomiser.py`**

```python
#!/usr/bin/env python3
"""
Constrained Atomiser - LLM extraction for genuine gaps only.
All output: DRAFT status + mandatory SME review + confidence cap at 0.85
"""

import json
from dataclasses import dataclass
from typing import List, Dict, Optional
from datetime import datetime

@dataclass
class AtomiserConfig:
    max_confidence: float = 0.85  # LLM cannot self-certify higher
    require_sme_review: bool = True
    model: str = "claude-opus-4-5-20250514"


class ConstrainedAtomiser:
    """
    Constrained LLM extraction for guideline gaps.

    ONLY invoke when:
    - No existing CQL in registry
    - Table extraction failed
    - Complex sequencing/titration needed
    """

    EXTRACTION_SCHEMA = {
        "type": "object",
        "required": ["population", "intervention", "evidence", "confidence"],
        "properties": {
            "population": {
                "type": "object",
                "properties": {
                    "condition": {"type": "string"},
                    "qualifiers": {"type": "array", "items": {"type": "string"}},
                    "exclusions": {"type": "array", "items": {"type": "string"}}
                }
            },
            "intervention": {
                "type": "object",
                "properties": {
                    "action": {"type": "string", "enum": ["RECOMMEND", "CONSIDER", "AVOID", "CONTRAINDICATED"]},
                    "medication_or_procedure": {"type": "string"},
                    "dose_or_parameters": {"type": "string"}
                }
            },
            "titration_sequence": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "step": {"type": "integer"},
                        "dose": {"type": "string"},
                        "duration": {"type": "string"},
                        "escalation_criteria": {"type": "string"},
                        "hold_criteria": {"type": "array", "items": {"type": "string"}}
                    }
                }
            },
            "temporal_constraints": {
                "type": "array",
                "items": {
                    "type": "object",
                    "properties": {
                        "step_id": {"type": "string"},
                        "deadline_type": {"type": "string", "enum": ["RELATIVE", "ABSOLUTE", "RECURRING"]},
                        "deadline_value": {"type": "string"},
                        "deadline_from_event": {"type": "string"}
                    }
                }
            },
            "evidence": {
                "type": "object",
                "properties": {
                    "cor": {"type": "string"},
                    "loe": {"type": "string"},
                    "source_text": {"type": "string"}
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

    def __init__(self, config: AtomiserConfig = None):
        self.config = config or AtomiserConfig()

    def extract(self, text_chunk: str, expected_output_type: str) -> Dict:
        """
        Extract structured content from text chunk.

        Args:
            text_chunk: Specific text (NOT entire guideline)
            expected_output_type: 'titration', 'exception', 'bundle', etc.

        Returns:
            Structured extraction with DRAFT status
        """
        # This would call Claude API with strict schema
        # For now, return template

        extraction = {
            "raw_text": text_chunk,
            "expected_type": expected_output_type,
            "extraction": None,  # Would be populated by LLM
            "status": "DRAFT",
            "requires_sme_review": True,
            "provenance": {
                "extraction_method": "ATOMISER_LLM",
                "model": self.config.model,
                "timestamp": datetime.utcnow().isoformat(),
                "human_reviewed": False
            }
        }

        return extraction

    def apply_governance(self, extraction: Dict) -> Dict:
        """Apply governance constraints to extraction."""

        # Cap confidence at 0.85
        if extraction.get("confidence", 0) > self.config.max_confidence:
            extraction["confidence"] = self.config.max_confidence
            extraction["confidence_capped"] = True

        # Force DRAFT status
        extraction["status"] = "DRAFT"
        extraction["requires_sme_review"] = True

        return extraction

    def validate_schema(self, extraction: Dict) -> bool:
        """Validate extraction against schema."""
        # Would use jsonschema validation
        required = self.EXTRACTION_SCHEMA["required"]
        return all(key in extraction for key in required)


class AtomiserRegistry:
    """
    Track what content needs Atomiser vs what's already available.
    """

    def __init__(self, registry_path: str):
        self.registry_path = registry_path
        self.existing_cql = self._load_existing_cql()

    def _load_existing_cql(self) -> Dict[str, str]:
        """Load registry of existing CQL content."""
        # Would load from actual registry file
        return {
            "vte_prophylaxis": "cms-ecqm/CMS108",
            "diabetes_a1c": "cms-ecqm/CMS122",
            "opioid_mme": "cdc-opioid/OMTKLogic",
            # ... etc
        }

    def needs_atomiser(self, topic: str) -> bool:
        """Check if topic needs Atomiser or has existing CQL."""
        return topic.lower() not in self.existing_cql

    def get_existing_cql(self, topic: str) -> Optional[str]:
        """Get path to existing CQL if available."""
        return self.existing_cql.get(topic.lower())
```

---

## Implementation Tracking

### Master Checklist

```markdown
## Phase 1: Foundation Imports
- [ ] Clone CDC Opioid repository
- [ ] Import 14 Opioid CQL files
- [ ] Import Opioid PlanDefinitions
- [ ] Import Opioid ValueSets
- [ ] Clone eCQM repository
- [ ] Import CMS108 (VTE)
- [ ] Import CMS190 (ICU VTE)
- [ ] Import CMS131 (Diabetes Eye)
- [ ] Import CMS144 (HF Beta-Blocker)
- [ ] Import CMS145 (CAD Beta-Blocker)
- [ ] Import CMS71 (AFib)
- [ ] Import CMS347 (Statin)
- [ ] Update foundation libraries
- [ ] Validate all CQL compiles

## Phase 2: WHO SMART Harvest
- [ ] Clone SMART ANC
- [ ] Import ANC CQL files
- [ ] Clone SMART Immunization
- [ ] Import Immunization CQL files
- [ ] Clone SMART HIV
- [ ] Import HIV CQL files
- [ ] Extract PlanDefinitions for KB-3
- [ ] Add jurisdiction headers

## Phase 3: Terminology Population
- [ ] Create terminology import script
- [ ] Import CDC Opioid ValueSets
- [ ] Import CMS eCQM ValueSets
- [ ] Import WHO ValueSets
- [ ] Create CodeSystem stubs (SNOMED, LOINC, RxNorm, ICD-10)
- [ ] Configure KB-7 resolution strategy

## Phase 4: Table Extraction Pipeline
- [ ] Install pdfplumber dependency
- [ ] Create GuidelineTableExtractor class
- [ ] Implement COR/LOE regex patterns
- [ ] Implement temporal constraint extraction
- [ ] Create KB-15 output format
- [ ] Test on ACC/AHA HF guideline
- [ ] Test on SSC 2021

## Phase 5: Constrained Atomiser
- [ ] Create Atomiser extraction schema
- [ ] Implement confidence cap (0.85)
- [ ] Implement DRAFT status enforcement
- [ ] Create AtomiserRegistry for gap detection
- [ ] Implement SME review queue integration
- [ ] Test on titration sequence extraction
```

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| CQL Files Imported | +70 files | File count |
| CDC Opioid Complete | 14/14 files | Import validation |
| CMS Measures Complete | 10/10 measures | Import validation |
| WHO SMART Domains | 3/3 domains | Import validation |
| Terminology Coverage | >80% | ValueSet resolution |
| Table Extraction Accuracy | >95% | SME validation sample |
| LLM Exposure | <15% | Extraction logs |
| All CQL Compiles | 100% | CI/CD pipeline |

---

## Next Steps

After completing this plan:

1. **Research Phase**: Deep dive into each repository before import
2. **Validation Phase**: Compile and test all imported CQL
3. **Integration Phase**: Wire up KB-3 temporal extraction from PlanDefinitions
4. **Runtime Phase**: Connect to medication-advisor-engine

---

**Document Status:** READY FOR EXECUTION

This plan follows the Vaidshala philosophy: **Deterministic First, LLM as Last Resort**
