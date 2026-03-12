# CMS eCQM Import Guide

## Overview

This directory contains CMS Electronic Clinical Quality Measures (eCQM) imported
as **vendor code** from the official eCQI Resource Center. These measures are
authoritative clinical quality logic maintained by CMS.

## Import Rules (MANDATORY)

### ⚠️ VENDOR CODE POLICY

```
╔══════════════════════════════════════════════════════════════════╗
║  CMS eCQM content is IMPORTED, NOT AUTHORED                     ║
║                                                                  ║
║  • DO NOT modify CQL logic                                       ║
║  • DO NOT rename files or libraries                              ║
║  • DO NOT change version numbers                                 ║
║  • DO NOT add custom code to these directories                   ║
║                                                                  ║
║  If adaptation is needed, create a WRAPPER in tier-5-regional    ║
╚══════════════════════════════════════════════════════════════════╝
```

## Source

- **Official Source**: https://ecqi.healthit.gov/ecqms/ep-ecqms
- **Content Year**: 2025 (CMS eCQM release year)
- **FHIR Version**: R4 (4.0.1)
- **CQL Version**: 1.5

## Directory Structure

Each measure follows this structure:

```
CMS{XXX}/
├── CMS{XXX}-v{version}.cql          # Main measure logic
├── manifest.yaml                     # Import metadata
├── evidence.md                       # Source documentation link
└── dependencies/                     # Required libraries
    ├── QICoreCommon.cql             # QI-Core patterns
    ├── SupplementalDataElements.cql # SDE requirements
    └── ...                          # Other dependencies
```

## Import Manifest Template

Each measure MUST have a `manifest.yaml`:

```yaml
measure_id: "CMS122"
measure_name: "Diabetes: Hemoglobin A1c (HbA1c) Poor Control (>9%)"
cms_version: "v12"
import_date: "YYYY-MM-DD"
import_source: "https://ecqi.healthit.gov/ecqm/ep/2024/cms122v12"
sha256_checksum: "abc123..."

dependencies:
  - library: "QICoreCommon"
    version: "2.0.000"
  - library: "SupplementalDataElements"
    version: "4.0.000"

clinical_domain: "cardiometabolic"
conditions_addressed:
  - "diabetes_mellitus_type_2"

import_status: "pending"  # pending | imported | validated | production
imported_by: "system"
validated_by: null
validation_date: null
```

## Priority Measures for Import

Per CTO/CMO directive, import these measures first:

| Measure | Name | Clinical Domain | Priority |
|---------|------|-----------------|----------|
| CMS122 | Diabetes: HbA1c Poor Control | Cardiometabolic | P1 |
| CMS165 | Controlling High Blood Pressure | Cardiometabolic | P1 |
| CMS2 | Screening for Depression | Mental Health | P1 |
| CMS134 | Diabetes: Nephropathy Screening | Renal | P1 |
| CMS123 | Diabetes: Foot Exam | Cardiometabolic | P2 |
| CMS131 | Diabetes: Eye Exam | Cardiometabolic | P2 |

## Import Procedure

### Step 1: Download from eCQI Resource Center

```bash
# Example: Download CMS122 v12
curl -O https://ecqi.healthit.gov/sites/default/files/ecqm/measures/CMS122v12.zip
unzip CMS122v12.zip -d CMS122/
```

### Step 2: Create Manifest

Create `manifest.yaml` with import metadata and checksum.

### Step 3: Validate Structure

```bash
# Verify required files present
ls CMS122/*.cql
ls CMS122/dependencies/
```

### Step 4: Attempt Compilation

```bash
# Run CQL-to-ELM compiler
make compile-cql MEASURE=CMS122
```

### Step 5: Document Expected Errors

First compilation will likely fail due to:
- Missing QI-Core dependencies (expected)
- Version mismatches (expected)
- Terminology references (expected)

These are documented in `compilation-notes.md` for each measure.

### Step 6: Resolve Dependencies

Import required dependency libraries (QICoreCommon, etc.) into
`tier-2-cqm-infra/vendor/` as vendor code.

## Governance

| Action | Approval Required |
|--------|-------------------|
| Import new measure | Technical Review |
| Update measure version | Clinical + Technical |
| Remove measure | CMO Approval |
| Modify measure (forbidden) | ❌ Not Allowed |

## Related Files

- `coverage-index.yaml` - Tracks which conditions are covered
- `tier-5-regional-adapters/` - Regional adaptations/wrappers
- `tier-2-cqm-infra/vendor/` - Shared vendor dependencies
