# Clinical Versioning Standard

## Overview

Clinical knowledge artifacts follow **Clinical Semantic Versioning (CSV)**, an extension of semantic versioning tailored for healthcare applications.

## Version Format

```
MAJOR.MINOR.PATCH[-REGION][-PRERELEASE]

Examples:
- 1.0.0          # Initial release
- 1.1.0          # New feature
- 1.1.1          # Bug fix
- 2.0.0-IN       # India-specific major version
- 1.2.0-AU       # Australia-specific minor version
- 1.0.0-alpha.1  # Pre-release
```

## Version Components

### MAJOR Version
Increment when making **breaking changes**:
- Removing value set codes
- Changing CQL function signatures
- Modifying clinical thresholds that affect existing logic
- Renaming or removing libraries

**Impact**: Downstream consumers MUST update their code.

### MINOR Version
Increment when adding **backward-compatible features**:
- New CQL libraries
- New value sets
- Additional codes to existing value sets
- New calculators
- New guidelines

**Impact**: Downstream consumers MAY update at their convenience.

### PATCH Version
Increment for **backward-compatible bug fixes**:
- Fixing calculation errors
- Documentation updates
- Performance improvements
- Test additions

**Impact**: Downstream consumers SHOULD update promptly.

## Regional Suffixes

When a version is region-specific:

| Suffix | Region | Description |
|--------|--------|-------------|
| `-IN` | India | NLEM, ICMR, ICD-10-WHO |
| `-AU` | Australia | PBS, RACGP, ICD-10-AM |
| `-US` | United States | CMS, FDA, ICD-10-CM |
| `-UK` | United Kingdom | NICE, BNF |

Example: `2.1.0-AU` means version 2.1.0 with Australia-specific content.

## Pre-Release Versions

```
1.0.0-alpha.1   # Early development
1.0.0-beta.1    # Feature complete, testing
1.0.0-rc.1      # Release candidate
```

Pre-release versions:
- Are NOT deployed to production
- MAY have breaking changes
- Require explicit opt-in

## Version Lifecycle

```
┌─────────┐     ┌─────────┐     ┌─────────┐     ┌─────────┐
│  Draft  │ ──▶ │  Alpha  │ ──▶ │  Beta   │ ──▶ │   RC    │
└─────────┘     └─────────┘     └─────────┘     └─────────┘
                                                     │
                                                     ▼
┌─────────┐     ┌─────────┐     ┌─────────┐     ┌─────────┐
│ Retired │ ◀── │ Sunset  │ ◀── │ Active  │ ◀── │ Release │
└─────────┘     └─────────┘     └─────────┘     └─────────┘
```

### Lifecycle Stages

| Stage | Description | Duration |
|-------|-------------|----------|
| Draft | In development | Unlimited |
| Alpha | Internal testing | 2-4 weeks |
| Beta | External testing | 2-4 weeks |
| RC | Final validation | 1-2 weeks |
| Release | Production use | Varies |
| Active | Recommended version | Until next release |
| Sunset | Deprecated, still supported | 6 months |
| Retired | No longer supported | N/A |

## Compatibility Matrix

When updating, check compatibility:

```
┌──────────────┬────────┬────────┬────────┐
│ Consumer At  │ MAJOR  │ MINOR  │ PATCH  │
├──────────────┼────────┼────────┼────────┤
│ Same MAJOR   │   ✓    │   ✓    │   ✓    │
│ Lower MAJOR  │   ✗    │   ✗    │   ✗    │
│ Higher MAJOR │   ✓*   │   ✓*   │   ✓*   │
└──────────────┴────────┴────────┴────────┘

✓  = Compatible
✗  = Incompatible
✓* = May work, not guaranteed
```

## Changelog Requirements

Every version must have a CHANGELOG entry:

```markdown
## [1.2.0] - 2024-01-15

### Added
- New eGFR calculator for CKD staging
- ICD-10 codes for diabetic nephropathy

### Changed
- Updated HbA1c thresholds per ADA 2024

### Deprecated
- Old GFR formula (use eGFR instead)

### Removed
- None

### Fixed
- SOFA score calculation for platelets

### Security
- None

### Evidence
- ADA Standards of Care 2024
- KDIGO CKD Guidelines 2024
```

## Dependency Versioning

When specifying dependencies between tiers:

```
# Strict: Exact version
tier-1-primitives = "1.0.0"

# Caret: Compatible with (same MAJOR)
tier-1-primitives = "^1.0.0"  # Accepts 1.x.x

# Tilde: Patch updates only
tier-1-primitives = "~1.0.0"  # Accepts 1.0.x
```

## Version Lock Files

Production deployments use lock files:

```json
{
  "lockVersion": 1,
  "artifacts": {
    "tier-0-fhir": "1.0.0",
    "tier-0.5-terminology": "2.1.0-AU",
    "tier-1-primitives": "1.3.2",
    "tier-2-cqm-infra": "1.0.0",
    "tier-3-domain-commons": "1.5.0",
    "tier-4-guidelines": "2024.1.0",
    "tier-5-regional-adapters": "1.0.0-AU"
  },
  "generatedAt": "2024-01-15T10:30:00Z",
  "signature": "base64-encoded-ed25519-signature"
}
```

## Rollback Policy

If a version causes issues:
1. Immediately revert to previous lock file
2. Investigate root cause
3. Patch and re-release with incremented PATCH
4. Document in incident log
