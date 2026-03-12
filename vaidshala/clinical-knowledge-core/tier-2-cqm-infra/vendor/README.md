# Vendor Libraries

This directory contains **vendor code** - external CQL libraries imported
from authoritative sources. These are dependencies required by CMS eCQM
measures and other imported guidelines.

## Vendor Policy

```
╔══════════════════════════════════════════════════════════════════╗
║  VENDOR CODE - DO NOT MODIFY                                     ║
║                                                                  ║
║  These libraries are imported unchanged from their sources.      ║
║  Modifications break provenance and regulatory compliance.       ║
╚══════════════════════════════════════════════════════════════════╝
```

## Expected Libraries

The following libraries will be imported here:

| Library | Version | Source | Purpose |
|---------|---------|--------|---------|
| QICoreCommon | 2.0.000 | CQF | QI-Core FHIR patterns |
| SupplementalDataElements | 4.0.000 | CMS | SDE for reporting |
| FHIRCommon | 1.1.000 | CQF | FHIR common patterns |
| MATGlobalCommonFunctions | 8.0.000 | CMS | Shared measure logic |

## Import Source

- **QI-Core Libraries**: https://github.com/cqframework/cqf-content
- **CMS Libraries**: https://ecqi.healthit.gov/ecqms

## Import Procedure

1. Download library from source
2. Verify SHA256 checksum
3. Place in this directory
4. Create manifest entry
5. Do NOT modify content

## Governance

Changes to vendor libraries require:
- CTO + CMO approval
- Source verification
- Checksum validation
- Full regression testing
