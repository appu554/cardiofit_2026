# Filtered Records Audit Log

**Document Version**: 1.0
**Last Updated**: 2026-01-20
**Purpose**: Audit trail of intentionally excluded records during Phase 1 data ingestion

---

## Overview

This document tracks all records that were intentionally filtered or excluded during the Phase 1 data ingestion process. Each exclusion is documented with rationale for regulatory audit compliance.

---

## 1. KB-5: ONC Drug-Drug Interactions

### Filtering Rules Applied

| Rule | Rationale | Records Affected |
|------|-----------|------------------|
| Severity filter: Only HIGH/CONTRAINDICATED | Focus on clinically significant interactions | 0 excluded |
| Duplicate pair detection | Prevent redundant storage | 0 excluded |
| Invalid RxCUI check | Data quality enforcement | 0 excluded |

### Excluded Records

**None** - All 25 ONC high-priority pairs were included.

---

## 2. KB-6: CMS Formulary Coverage

### Filtering Rules Applied

| Rule | Rationale | Records Affected |
|------|-----------|------------------|
| `NOT_COVERED` status exclusion | Non-covered drugs don't need CDS alerts | 1 excluded |
| Invalid RxCUI check | Data quality enforcement | 0 excluded |
| Duplicate entry detection | Prevent redundant storage | 0 excluded |

### Excluded Records

| Record # | Drug Name | RxCUI | Reason | Details |
|----------|-----------|-------|--------|---------|
| 1 | Prilosec (omeprazole) | 7646 | NOT_COVERED | Contract H1234, Plan 001 does not cover brand-name Prilosec; generic omeprazole is covered separately |

### Configuration Option

To include `NOT_COVERED` drugs (e.g., for "coverage denied" alerts):

```go
// In formulary loader configuration:
config.IncludeNotCovered = true  // Default: false
```

---

## 3. KB-16: LOINC Lab Reference Ranges

### Filtering Rules Applied

| Rule | Rationale | Records Affected |
|------|-----------|------------------|
| Invalid LOINC code check | Data quality enforcement | 0 excluded |
| Missing reference range check | Incomplete data exclusion | 0 excluded |
| Duplicate LOINC/demographic key | Prevent redundant storage | 0 excluded |

### Excluded Records

**None** - All 50 lab reference ranges were included.

---

## Summary Statistics

```
╔═══════════════════════════════════════════════════════════════════╗
║              PHASE 1 FILTERING SUMMARY                            ║
╠═══════════════════════════════════════════════════════════════════╣
║                                                                   ║
║  KB-5 ONC DDI                                                     ║
║  ────────────────────────────────────────────────────────────     ║
║  Total source records:    25 pairs                                ║
║  Records loaded:          50 (bidirectional)                      ║
║  Records filtered:        0                                       ║
║  Filter rate:             0%                                      ║
║                                                                   ║
║  KB-6 CMS Formulary                                               ║
║  ────────────────────────────────────────────────────────────     ║
║  Total source records:    30                                      ║
║  Records loaded:          29                                      ║
║  Records filtered:        1 (NOT_COVERED)                         ║
║  Filter rate:             3.3%                                    ║
║                                                                   ║
║  KB-16 LOINC Labs                                                 ║
║  ────────────────────────────────────────────────────────────     ║
║  Total source records:    50                                      ║
║  Records loaded:          50                                      ║
║  Records filtered:        0                                       ║
║  Filter rate:             0%                                      ║
║                                                                   ║
╠═══════════════════════════════════════════════════════════════════╣
║  TOTAL FILTERED:          1 record (0.9% overall)                 ║
╚═══════════════════════════════════════════════════════════════════╝
```

---

## Audit Trail

| Date | Action | Operator | Notes |
|------|--------|----------|-------|
| 2026-01-20 | Initial dry-run ingestion | System | Phase 1 data loaded |
| 2026-01-20 | Filtered records documented | System | This audit log created |

---

## Regulatory Compliance Notes

### FDA 21 CFR Part 11 Alignment

- **Traceability**: All filtering decisions are documented with rationale
- **Reproducibility**: Filtering rules are deterministic and documented in code
- **Audit Trail**: This document provides complete exclusion history

### HIPAA Considerations

- No PHI is present in filtered records
- All drug/lab identifiers are public reference data

---

## Change Log

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-01-20 | System | Initial document |

---

## Related Documents

- [PHASE1_DEFINITION_OF_DONE.md](../../docs/PHASE1_DEFINITION_OF_DONE.md)
- [LLM_CONSTITUTION.md](../../docs/LLM_CONSTITUTION.md)
- [CACHE_POLICY.md](../../docs/CACHE_POLICY.md)
