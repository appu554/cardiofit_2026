# Clinical Knowledge Core

> **The Clinical Constitution** - All medical logic lives here.

## Purpose

This repository contains all clinical knowledge artifacts:
- CQL (Clinical Quality Language) libraries
- Value sets and code systems
- Clinical calculators
- Guidelines and measures
- Regional adaptations

## Tier Structure

```
clinical-knowledge-core/
├── tier-0-fhir/           # FHIR Foundation
├── tier-0.5-terminology/  # Terminology Layer (SNOMED, ICD, LOINC)
├── tier-1-primitives/     # Utility functions
├── tier-2-cqm-infra/      # Quality measure infrastructure
├── tier-3-domain-commons/ # Clinical calculators
├── tier-4-guidelines/     # Clinical guidelines
├── tier-5-regional/       # Regional adaptations
├── build/                 # Build artifacts
└── tests/                 # Test suites
```

## Governance

**All changes require:**
1. Clinical reviewer approval (physician or pharmacist)
2. Informatics reviewer approval (engineer)
3. Evidence source documentation
4. Version bump following semantic versioning

See [GOVERNANCE.md](GOVERNANCE.md) for detailed rules.

## Versioning

We follow **Clinical Semantic Versioning**:
- **MAJOR**: Breaking changes to interfaces or clinical logic
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes, documentation

See [VERSIONING.md](VERSIONING.md) for details.

## Quick Start

### For Clinical Authors

1. Create or modify CQL in the appropriate tier
2. Add corresponding value sets in `tier-0.5-terminology/`
3. Write tests in `tests/`
4. Submit PR with evidence source

### For Engineers

1. Build: `make build`
2. Test: `make test`
3. Package: `make package`

## Tier Dependencies

```
tier-5 → tier-4 → tier-3 → tier-2 → tier-1 → tier-0 → tier-0.5
```

Each tier can only import from lower-numbered tiers.

## Regional Support

- **India (IN)**: `tier-5-regional-adapters/IN/`
- **Australia (AU)**: `tier-5-regional-adapters/AU/`

## Build Artifacts

After build:
- `build/cql-to-elm/` - Compiled ELM JSON
- `build/valueset-expansion/` - Expanded value sets
- `build/manifests/` - Signed manifests

## License

Proprietary - CardioFit Clinical Synthesis Hub
